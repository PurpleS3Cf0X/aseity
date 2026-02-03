package test

import (
	"context"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProviderFallback simulates a model that only outputs text, no native tool calls
type MockProviderFallback struct {
	ResponseText string
}

func (m *MockProviderFallback) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		// Simulate streaming the text response
		ch <- provider.StreamChunk{Delta: "Sure, I can help with that. "}
		ch <- provider.StreamChunk{Delta: m.ResponseText}
		ch <- provider.StreamChunk{Done: true}
	}()
	return ch, nil
}
func (m *MockProviderFallback) Name() string { return "mock-fallback" }
func (m *MockProviderFallback) Models(ctx context.Context) ([]string, error) {
	return []string{"mock"}, nil
}

func TestToolFallbackMechanism(t *testing.T) {
	// 1. Setup
	// We want the agent to receive a text response containing the fallback pattern
	// and trigger the tool execution.
	expectedTool := "bash"
	expectedArgs := `{"command": "echo 'fallback works'"}`
	responseText := "\n[TOOL:" + expectedTool + "|" + expectedArgs + "]\n"

	mockProv := &MockProviderFallback{ResponseText: responseText}

	// Create a registry with a mock bash tool
	reg := tools.NewRegistry(nil, true)

	// We need to register a "bash" tool that we can verify was called
	mockBash := &tools.BashTool{
		AllowedCommands: []string{"echo"},
	}
	// Wrap execution to detect call
	// Since we can't easily mock the tool implementation in the registry without changing types,
	// let's rely on the Event output from the agent.
	reg.Register(mockBash)

	agt := agent.New(mockProv, reg, "")

	// 2. Execution
	events := make(chan agent.Event)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		agt.Send(ctx, "Run the fallback command", events)
		close(events)
	}()

	// 3. Verification
	// We look for EventToolCall matching our expected values
	foundCall := false
	for evt := range events {
		if evt.Type == agent.EventToolCall {
			if evt.ToolName == expectedTool {
				// We won't check args strictly as formatting might vary, but basic check
				t.Logf("Tool call detected: %s %s", evt.ToolName, evt.ToolArgs)
				foundCall = true
			}
		}
	}

	if !foundCall {
		t.Errorf("Fallback mechanism failed: Did not detect tool call from text '%s'", responseText)
	} else {
		t.Log("Success: Fallback tool call detected.")
	}
}
