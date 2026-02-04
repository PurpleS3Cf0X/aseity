package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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
)

// Agent drives the think-act-observe loop.
type Agent struct {
	prov      provider.Provider
	tools     *tools.Registry
	conv      *Conversation
	ConfirmCh chan bool   // TUI sends true/false here
	InputCh   chan string // TUI sends user input here
	depth     int         // sub-agent nesting depth
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

// NewWithDepth creates a sub-agent with nesting depth tracking.
func NewWithDepth(prov provider.Provider, registry *tools.Registry, depth int, systemPrompt string) *Agent {
	a := New(prov, registry, systemPrompt)
	a.depth = depth
	return a
}

func (a *Agent) Conversation() *Conversation { return a.conv }
func (a *Agent) Depth() int                  { return a.depth }

func (a *Agent) Send(ctx context.Context, userMsg string, events chan<- Event) {
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
			events <- Event{Type: EventDone, Done: true}
			return
		}

		for _, tc := range toolCalls {
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
