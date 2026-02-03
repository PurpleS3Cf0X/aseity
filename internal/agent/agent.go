package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	EventConfirmRequest // sent when a tool needs user approval
	EventDone
	EventError
)

// Agent drives the think-act-observe loop.
type Agent struct {
	prov      provider.Provider
	tools     *tools.Registry
	conv      *Conversation
	ConfirmCh chan bool // TUI sends true/false here
	depth     int       // sub-agent nesting depth
}

func New(prov provider.Provider, registry *tools.Registry) *Agent {
	conv := NewConversation()
	conv.AddSystem(BuildSystemPrompt())
	return &Agent{
		prov:      prov,
		tools:     registry,
		conv:      conv,
		ConfirmCh: make(chan bool, 1),
	}
}

// NewWithDepth creates a sub-agent with nesting depth tracking.
func NewWithDepth(prov provider.Provider, registry *tools.Registry, depth int) *Agent {
	a := New(prov, registry)
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
		stream, err := a.prov.Chat(ctx, a.conv.Messages(), a.tools.ToolDefs())
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

			res, err := a.tools.Execute(ctx, tc.Name, tc.Args, streamCallback)
			if err != nil {
				errMsg := fmt.Sprintf("tool execution error: %s", err.Error())
				a.conv.AddToolResult(tc.ID, errMsg)
				events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, Error: errMsg}
				continue
			}

			output := res.Output
			if res.Error != "" {
				output += "\nError: " + res.Error
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
