package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// TestCoTParsing validates that <thought> tags are parsed into EventThinking
func TestCoTParsing(t *testing.T) {
	// Mock provider that returns CoT formatted response
	prov := &MockProviderCoT{}

	// Registry with a mock tool
	reg := tools.NewRegistry(nil, true)
	reg.Register(&MockTool{name: "bash"})

	agent := New(prov, reg, "")
	events := make(chan Event, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go agent.Send(ctx, "test query", events)

	var foundThinking bool
	var foundThinkingText string
	var foundToolCall bool

	for evt := range events {
		if evt.Type == EventThinking {
			foundThinking = true
			foundThinkingText = evt.Text
		}
		if evt.Type == EventToolCall {
			foundToolCall = true
		}
		if evt.Done {
			break
		}
	}

	if !foundThinking {
		t.Error("Did not receive EventThinking")
	}

	expectedThought := "I should list files to see what is here."
	if foundThinkingText != expectedThought {
		t.Errorf("Expected thought text %q, got %q", expectedThought, foundThinkingText)
	}

	if !foundToolCall {
		t.Error("Did not receive EventToolCall")
	}
}

// TestJSONFallbackParsing validates that raw JSON tool calls are parsed
func TestJSONFallbackParsing(t *testing.T) {
	// Mock provider that returns JSON formatted response
	prov := &MockProviderJSON{}

	// Registry with a mock tool
	reg := tools.NewRegistry(nil, true)
	reg.Register(&MockTool{name: "sandbox_run"})

	agent := New(prov, reg, "")
	events := make(chan Event, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go agent.Send(ctx, "run command", events)

	var foundToolCall bool
	var toolName string
	var toolArgs string

	for evt := range events {
		if evt.Type == EventToolCall {
			foundToolCall = true
			toolName = evt.ToolName
			toolArgs = evt.ToolArgs
		}
		if evt.Done {
			break
		}
	}

	if !foundToolCall {
		t.Error("Did not receive EventToolCall from JSON input")
	}

	if toolName != "sandbox_run" {
		t.Errorf("Expected tool name 'sandbox_run', got %q", toolName)
	}

	// The args might be normalized, but should contain the command
	if !strings.Contains(toolArgs, "echo 'hello'") {
		t.Errorf("Expected args to contain command, got %q", toolArgs)
	}
}

// MockProviderCoT simulates a model returning <thought> blocks
type MockProviderCoT struct{}

func (m *MockProviderCoT) Chat(ctx context.Context, messages []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)

		response := `<thought>
I should list files to see what is here.
</thought>
[TOOL:bash|{"command": "ls -la"}]`

		// Simulate streaming
		ch <- provider.StreamChunk{
			Delta: response,
			Done:  true,
		}
	}()
	return ch, nil
}

func (m *MockProviderCoT) Name() string      { return "mock-cot" }
func (m *MockProviderCoT) ModelName() string { return "mock-cot-model" }
func (m *MockProviderCoT) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-cot-model"}, nil
}

// MockProviderJSON simulates a model returning raw JSON
type MockProviderJSON struct{}

func (m *MockProviderJSON) Chat(ctx context.Context, messages []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)

		response := `{"name": "sandbox_run", "arguments": {"command": "echo 'hello'"}}`

		ch <- provider.StreamChunk{
			Delta: response,
			Done:  true,
		}
	}()
	return ch, nil
}

func (m *MockProviderJSON) Name() string      { return "mock-json" }
func (m *MockProviderJSON) ModelName() string { return "mock-json-model" }
func (m *MockProviderJSON) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-json-model"}, nil
}

// MockTool for the test
type MockTool struct {
	name string
}

func (m *MockTool) Name() string            { return m.name }
func (m *MockTool) Description() string     { return "mock" }
func (m *MockTool) Parameters() any         { return nil }
func (m *MockTool) NeedsConfirmation() bool { return false }
func (m *MockTool) Execute(ctx context.Context, args string) (tools.Result, error) {
	return tools.Result{Output: "success"}, nil
}
