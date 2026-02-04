package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

const MaxTurns = 50 // prevent infinite agent loops

type Event struct {
	Type     EventType
	Text     string
	ToolName string
	ToolArgs string
	ToolID   string
	Result   string
	Error    string
	Done     bool
}

type EventType int

const (
	EventDelta EventType = iota
	EventThinking
	EventToolCall
	EventToolResult
	EventToolOutput     // new event for streaming output
	EventInputRequest   // sent when a tool needs user input
	EventConfirmRequest // sent when a tool needs user approval
	EventDone
	EventError
	EventJudgeCall // new event for quality gate evaluation
)

// Agent drives the think-act-observe loop.
type Agent struct {
	prov      provider.Provider
	tools     *tools.Registry
	conv      *Conversation
	ConfirmCh chan bool   // TUI sends true/false here
	InputCh   chan string // TUI sends user input here
	depth     int         // sub-agent nesting depth

	QualityGateEnabled bool   // Enforce strict judge check before completion
	OriginalGoal       string // Track the initial user request for judging
}

func New(prov provider.Provider, registry *tools.Registry, systemPrompt string) *Agent {
	conv := NewConversation()
	if systemPrompt == "" {
		systemPrompt = BuildSystemPrompt()
	}
	conv.AddSystem(systemPrompt)
	return &Agent{
		prov:      prov,
		tools:     registry,
		conv:      conv,
		ConfirmCh: make(chan bool, 1),
		InputCh:   make(chan string, 1),
	}
}

// NewWithConversation creates an agent using an existing conversation history.
func NewWithConversation(prov provider.Provider, registry *tools.Registry, conv *Conversation) *Agent {
	return &Agent{
		prov:      prov,
		tools:     registry,
		conv:      conv,
		ConfirmCh: make(chan bool, 1),
		InputCh:   make(chan string, 1),
	}
}

// NewWithDepth creates a sub-agent with nesting depth tracking.
func NewWithDepth(prov provider.Provider, registry *tools.Registry, depth int, systemPrompt string) *Agent {
	a := New(prov, registry, systemPrompt)
	a.depth = depth
	return a
}

func (a *Agent) Conversation() *Conversation { return a.conv }
func (a *Agent) Depth() int                  { return a.depth }

func (a *Agent) Send(ctx context.Context, userMsg string, events chan<- Event) {
	if a.OriginalGoal == "" {
		a.OriginalGoal = userMsg
	}
	a.conv.AddUser(userMsg)
	a.runLoop(ctx, events)
}

func (a *Agent) runLoop(ctx context.Context, events chan<- Event) {
	for turn := 0; turn < MaxTurns; turn++ {
		// Construct the context with a dynamic reminder
		msgs := a.conv.Messages()

		// Inject a reminder at the end of context to keep the model focused
		reminder := fmt.Sprintf("Turn %d/%d. Review the history. If you just ran a command, did it work? If it failed, try a DIFFERENT approach. Do not repeat mistakes.", turn+1, MaxTurns)
		msgs = append(msgs, provider.Message{Role: provider.RoleSystem, Content: reminder})

		stream, err := a.prov.Chat(ctx, msgs, a.tools.ToolDefs())
		if err != nil {
			events <- Event{Type: EventError, Error: err.Error(), Done: true}
			return
		}

		var textBuf strings.Builder
		var toolCalls []provider.ToolCall

		for chunk := range stream {

			if chunk.Error != nil {
				events <- Event{Type: EventError, Error: chunk.Error.Error(), Done: true}
				return
			}
			if chunk.Thinking != "" {
				events <- Event{Type: EventThinking, Text: chunk.Thinking}
			}
			if chunk.Delta != "" {
				textBuf.WriteString(chunk.Delta)
				events <- Event{Type: EventDelta, Text: chunk.Delta}
			}
			if chunk.Done {
				toolCalls = chunk.ToolCalls
			}
		}

		assistantText := textBuf.String()
		a.conv.AddAssistant(assistantText, toolCalls)

		// FALLBACK: If no native tool calls, check for text-based pattern [TOOL:name|json_args]
		if len(toolCalls) == 0 {
			// Regex to capture: [TOOL:name|args]
			// We handle nested braces loosely or just take until the last ] on the line if simple
			// For robustness, let's assume one tool per line or block.
			// Format: `[TOOL:<name>|<args>]`
			re := regexp.MustCompile(`\[TOOL:(\w+)\|(.+?)\]`)
			matches := re.FindAllStringSubmatch(assistantText, -1)

			for _, match := range matches {
				if len(match) == 3 {
					toolName := match[1]
					toolArgs := match[2]
					// Basic JSON validation/cleanup if needed?
					// Ideally the model produces valid JSON.
					toolCalls = append(toolCalls, provider.ToolCall{
						ID:   fmt.Sprintf("fallback-%d", time.Now().UnixNano()),
						Name: toolName,
						Args: toolArgs,
					})
				}
			}
		}

		if len(toolCalls) == 0 {
			// Check Quality Gate before finishing
			if a.QualityGateEnabled && a.OriginalGoal != "" {
				// Don't finish yet — run the judge
				// We need the last assistant message as the "content" to judge.
				// Since we just added it:
				contentToJudge := assistantText

				events <- Event{Type: EventJudgeCall, Text: "Evaluating response against goal..."}

				// Construct Judge Arguments JSON
				judgeArgs := map[string]string{
					"original_goal": a.OriginalGoal,
					"content":       contentToJudge,
				}
				jsonArgs, _ := json.Marshal(judgeArgs)

				// Run Judge Tool manually
				// We assume "judge_output" is registered, or we instantiate it directly if possible.
				// But tools are in registry.
				res, err := a.tools.Execute(ctx, "judge_output", string(jsonArgs), nil)

				passed := false
				feedback := ""

				if err != nil {
					// If judge tool missing or failed execution, warn but maybe allow finish?
					// Or fail safe. Let's fail safe and report error.
					feedback = fmt.Sprintf("Quality Gate execution failed: %v", err)
				} else {
					// Parse Judge Result
					// Expected standard JSON from JudgeTool: { "status": "pass"|"fail", "feedback": "..." }
					// But tools.Execute returns a Result struct where Output is the string JSON.
					var verdict struct {
						Status   string `json:"status"`
						Feedback string `json:"feedback"`
					}
					if json.Unmarshal([]byte(res.Output), &verdict) == nil {
						if strings.ToLower(verdict.Status) == "pass" {
							passed = true
						}
						feedback = verdict.Feedback
					} else {
						// Malformed judge output
						feedback = "Judge returned malformed output: " + res.Output
					}
				}

				if !passed {
					// REJECT - Insert system message and Continue Loop
					rejectMsg := fmt.Sprintf("⛔ QUALITY GATE FAILED.\nYour response was rejected by the Critic.\nFeedback: %s\n\nYou MUST fix these issues and try again. Do not repeat the same mistake.", feedback)
					a.conv.AddSystem(rejectMsg)
					events <- Event{Type: EventError, Error: "Quality Gate Rejected: " + feedback, Done: false}
					// Actually EventError usually signals stopping?
					// No, we want to CONTINUE the loop.
					// So send delta or something? Or just nothing and loop.
					// Sending EventDelta might confuse TUI into thinking it's assistant text.
					// Let's rely on the internal loop continuing.
					continue
				} else {
					// PASS - Allow finish
					// Maybe inform user?
					events <- Event{Type: EventJudgeCall, Text: "Quality Gate Passed. ✅"}
				}
			}

			events <- Event{Type: EventDone, Done: true}
			return
		}

		// --- PARALLEL EXECUTION LOGIC ---
		var parallelGroup []provider.ToolCall
		var sequentialGroup []provider.ToolCall

		for _, tc := range toolCalls {
			if IsSafeToParallelize(tc.Name) {
				parallelGroup = append(parallelGroup, tc)
			} else {
				sequentialGroup = append(sequentialGroup, tc)
			}
		}

		// 1. Execute Parallel Group
		if len(parallelGroup) > 0 {
			var wg sync.WaitGroup
			var mu sync.Mutex // Protect conversation writes
			wg.Add(len(parallelGroup))

			for _, tc := range parallelGroup {
				go func(tc provider.ToolCall) {
					defer wg.Done()

					prettyArgs := formatToolArgs(tc.Name, tc.Args)
					events <- Event{
						Type: EventToolCall, ToolName: tc.Name,
						ToolArgs: prettyArgs, ToolID: tc.ID,
					}

					res, err := a.tools.Execute(ctx, tc.Name, tc.Args, nil)

					// Lock before writing to conversation
					mu.Lock()
					defer mu.Unlock()

					if err != nil {
						errMsg := fmt.Sprintf("Error executing %s: %v", tc.Name, err)
						a.conv.AddToolResult(tc.ID, errMsg)
						// Event channel is safe
						events <- Event{Type: EventError, Error: errMsg, ToolID: tc.ID}
						events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Result: errMsg}
						return
					}

					if res.Error != "" {
						a.conv.AddToolResult(tc.ID, res.Error)
						events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Result: res.Error}
					} else {
						a.conv.AddToolResult(tc.ID, res.Output)
						events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Result: res.Output}
					}
				}(tc)
			}

			wg.Wait()
		}

		// 2. Execute Sequential Group
		for _, tc := range sequentialGroup {
			prettyArgs := formatToolArgs(tc.Name, tc.Args)

			events <- Event{
				Type: EventToolCall, ToolName: tc.Name,
				ToolArgs: prettyArgs, ToolID: tc.ID,
			}

			// If confirmation needed, ask the TUI and block
			if a.tools.NeedsConfirmation(tc.Name) {
				events <- Event{
					Type: EventConfirmRequest, ToolName: tc.Name,
					ToolArgs: prettyArgs, ToolID: tc.ID,
				}

				select {
				case approved := <-a.ConfirmCh:
					if !approved {
						result := "User denied this operation."
						a.conv.AddToolResult(tc.ID, result)
						events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Result: result}
						continue
					}
				case <-ctx.Done():
					return
				}
			}

			// Execute
			streamCallback := func(chunk string) {
				events <- Event{
					Type:     EventToolOutput,
					ToolID:   tc.ID,
					ToolName: tc.Name,
					Text:     chunk,
				}
			}

			// If tool supports interactivity, inject the input channel and request callback
			if t, ok := a.tools.Get(tc.Name); ok {
				if interactive, ok := t.(interface {
					SetInputChan(chan string)
					SetInputRequestCallback(func())
				}); ok {
					interactive.SetInputChan(a.InputCh)
					interactive.SetInputRequestCallback(func() {
						events <- Event{
							Type:     EventInputRequest,
							ToolName: tc.Name,
							ToolID:   tc.ID,
						}
					})
				}
				// Inject StreamCallback
				if streamable, ok := t.(interface {
					SetStreamCallback(func(string))
				}); ok {
					streamable.SetStreamCallback(streamCallback)
				}
			}

			res, err := a.tools.Execute(ctx, tc.Name, tc.Args, streamCallback)
			if err != nil {
				errMsg := err.Error()
				fmt.Printf("DEBUG: System error: '%s'\n", errMsg) // DEBUG
				// Nudge: Check for common model mistakes
				if strings.Contains(errMsg, "unknown tool") {
					suggestions := make([]string, 0)
					// Simple heuristic for common hallucinations
					lowerName := strings.ToLower(tc.Name)
					if strings.Contains(lowerName, "fetch") {
						suggestions = append(suggestions, "web_fetch")
					} else if strings.Contains(lowerName, "crawl") {
						suggestions = append(suggestions, "web_crawl")
					} else if strings.Contains(lowerName, "write") {
						suggestions = append(suggestions, "file_write")
					} else if strings.Contains(lowerName, "read") {
						suggestions = append(suggestions, "file_read")
					}

					if len(suggestions) > 0 {
						errMsg += fmt.Sprintf(". Did you mean '%s'? Please retry with the correct name.", suggestions[0])
					} else {
						errMsg += ". Please check the 'Available Tools' list and retry with a valid tool name."
					}
				} else if strings.Contains(errMsg, "invalid json") {
					errMsg += ". Your JSON structure was malformed. Please ensure all quotes are escaped properly and retry."
				}

				formattedErr := fmt.Sprintf("tool execution error: %s", errMsg)
				a.conv.AddToolResult(tc.ID, formattedErr)
				events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Error: formattedErr}
				continue
			}

			output := res.Output
			// Nudge logic for Result errors (like unknown tool returned by Registry)
			if res.Error != "" {
				errMsg := res.Error
				fmt.Printf("DEBUG: Nudging result error: '%s'\n", errMsg) // DEBUG
				if strings.Contains(errMsg, "unknown tool") {
					suggestions := make([]string, 0)
					// Simple heuristic for common hallucinations
					lowerName := strings.ToLower(tc.Name)
					if strings.Contains(lowerName, "fetch") {
						suggestions = append(suggestions, "web_fetch")
					} else if strings.Contains(lowerName, "crawl") {
						suggestions = append(suggestions, "web_crawl")
					} else if strings.Contains(lowerName, "write") {
						suggestions = append(suggestions, "file_write")
					} else if strings.Contains(lowerName, "read") {
						suggestions = append(suggestions, "file_read")
					}

					if len(suggestions) > 0 {
						errMsg += fmt.Sprintf(". Did you mean '%s'? Please retry with the correct name.", suggestions[0])
					} else {
						errMsg += ". Please check the 'Available Tools' list and retry with a valid tool name."
					}
				}
				output += "\nError: " + errMsg
			}
			a.conv.AddToolResult(tc.ID, output)
			events <- Event{
				Type: EventToolResult, ToolID: tc.ID,
				ToolName: tc.Name, ToolArgs: prettyArgs, Result: output,
			}
		}
	}

	events <- Event{Type: EventError, Error: fmt.Sprintf("reached maximum of %d turns — stopping to prevent infinite loop", MaxTurns), Done: true}
}

func formatToolArgs(toolName, rawArgs string) string {
	var parsed map[string]any
	if json.Unmarshal([]byte(rawArgs), &parsed) != nil {
		if len(rawArgs) > 80 {
			return rawArgs[:80] + "..."
		}
		return rawArgs
	}

	switch toolName {
	case "bash":
		if cmd, ok := parsed["command"]; ok {
			return fmt.Sprintf("%v", cmd)
		}
	case "file_read", "file_write":
		if p, ok := parsed["path"]; ok {
			return fmt.Sprintf("%v", p)
		}
	case "file_search":
		if p, ok := parsed["pattern"]; ok {
			return fmt.Sprintf("pattern=%v", p)
		}
		if g, ok := parsed["grep"]; ok {
			return fmt.Sprintf("grep=%v", g)
		}
	case "web_search":
		if q, ok := parsed["query"]; ok {
			return fmt.Sprintf("%v", q)
		}
	case "web_fetch":
		if u, ok := parsed["url"]; ok {
			return fmt.Sprintf("%v", u)
		}
	}

	s := rawArgs
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return s
}
