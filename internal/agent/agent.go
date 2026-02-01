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
	Type      EventType
	Text      string
	ToolName  string
	ToolArgs  string
	ToolID    string
	Result    string
	Error     string
	Done      bool
	NeedConfirm bool
}

type EventType int

const (
	EventDelta EventType = iota
	EventThinking
	EventToolCall
	EventToolResult
	EventDone
	EventError
	EventConfirmRequest
)

type ConfirmFunc func(toolName, args string) bool

type Agent struct {
	prov    provider.Provider
	tools   *tools.Registry
	conv    *Conversation
	confirm ConfirmFunc
}

func New(prov provider.Provider, registry *tools.Registry, confirmFn ConfirmFunc) *Agent {
	conv := NewConversation()
	conv.AddSystem(BuildSystemPrompt())
	return &Agent{prov: prov, tools: registry, conv: conv, confirm: confirmFn}
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

		// Execute tool calls
		for _, tc := range toolCalls {
			events <- Event{Type: EventToolCall, ToolName: tc.Name, ToolArgs: tc.Args, ToolID: tc.ID}

			if a.tools.NeedsConfirmation(tc.Name) {
				if a.confirm != nil && !a.confirm(tc.Name, tc.Args) {
					result := "User denied this tool call."
					a.conv.AddToolResult(tc.ID, result)
					events <- Event{Type: EventToolResult, ToolID: tc.ID, Result: result}
					continue
				}
			}

			res, err := a.tools.Execute(ctx, tc.Name, tc.Args)
			if err != nil {
				errMsg := fmt.Sprintf("tool execution error: %s", err.Error())
				a.conv.AddToolResult(tc.ID, errMsg)
				events <- Event{Type: EventToolResult, ToolID: tc.ID, Error: errMsg}
				continue
			}

			output := res.Output
			if res.Error != "" {
				output += "\nError: " + res.Error
			}
			a.conv.AddToolResult(tc.ID, output)

			// pretty-print abbreviated args
			var prettyArgs string
			var parsed map[string]any
			if json.Unmarshal([]byte(tc.Args), &parsed) == nil {
				if cmd, ok := parsed["command"]; ok {
					prettyArgs = fmt.Sprintf("%v", cmd)
				} else if p, ok := parsed["path"]; ok {
					prettyArgs = fmt.Sprintf("%v", p)
				}
			}
			if prettyArgs == "" {
				prettyArgs = tc.Args
				if len(prettyArgs) > 80 {
					prettyArgs = prettyArgs[:80] + "..."
				}
			}
			events <- Event{Type: EventToolResult, ToolID: tc.ID, ToolName: tc.Name, ToolArgs: prettyArgs, Result: output}
		}
		// Loop again so the model can see tool results
	}
}
