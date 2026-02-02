package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

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
	EventConfirmRequest // sent when a tool needs user approval
	EventDone
	EventError
)

// ConfirmChan is used by the agent to ask the TUI for approval.
// The agent sends a confirm request event, then blocks reading from this channel.
type Agent struct {
	prov       provider.Provider
	tools      *tools.Registry
	conv       *Conversation
	ConfirmCh  chan bool // TUI sends true/false here
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

func (a *Agent) Send(ctx context.Context, userMsg string, events chan<- Event) {
	a.conv.AddUser(userMsg)
	a.runLoop(ctx, events)
}

func (a *Agent) runLoop(ctx context.Context, events chan<- Event) {
	for {
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

			// Send the tool call event (shows what's about to run)
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

				// Block until TUI responds
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
			res, err := a.tools.Execute(ctx, tc.Name, tc.Args)
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
