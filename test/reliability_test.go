package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/types"
)

// MockProviderForNudge simulates a tool call to a non-existent tool
type MockProviderForNudge struct {
	toolName string
}

func (m *MockProviderForNudge) Chat(ctx context.Context, history []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		// Only trigger once to avoid loops in test
		if len(history) <= 3 { // System + User + Reminder
			ch <- provider.StreamChunk{
				Done: true,
				ToolCalls: []provider.ToolCall{
					{ID: "call_1", Name: m.toolName, Args: "{}"},
				},
			}
		} else {
			ch <- provider.StreamChunk{Done: true, Delta: "Stopping now."}
		}
	}()
	return ch, nil
}

func (m *MockProviderForNudge) Name() string { return "MockNudge" }

func (m *MockProviderForNudge) ModelName() string { return "test-model" }
func (m *MockProviderForNudge) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestToolNudgingLogic(t *testing.T) {
	// 1. Setup Registry with NO tools (so any call fails)
	reg := tools.NewRegistry([]string{}, true)

	// 2. Setup Agent with Mock Provider that calls "fetch" (invalid)
	prov := &MockProviderForNudge{toolName: "fetch"}
	a := agent.New(prov, reg, "")

	// 3. Run Agent and capture events
	events := make(chan agent.Event, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go a.Send(ctx, "Please fetch this", events)

	// 4. Analyze Events
	foundNudge := false
	for evt := range events {
		if evt.Type == agent.EventToolResult && evt.ToolName == "fetch" {
			// Check both Error and Result fields
			combined := evt.Error + evt.Result
			if strings.Contains(combined, "Did you mean 'web_fetch'?") {
				foundNudge = true
			}
			t.Logf("Got Error: %s, Result: %s", evt.Error, evt.Result)
		}
		if evt.Done {
			break
		}
	}

	if !foundNudge {
		t.Errorf("Expected tool error to contain nudge 'Did you mean web_fetch?', but it didn't")
	}
}

// MockSpawner captures the task passed to it
type MockSpawner struct {
	capturedTask string
}

func (m *MockSpawner) Spawn(ctx context.Context, task string, files []string, name string) (int, error) {
	m.capturedTask = task
	return 1, nil
}
func (m *MockSpawner) Get(id int) (types.AgentInfo, bool) {
	return types.AgentInfo{Status: "done", Output: "done"}, true
}
func (m *MockSpawner) List() []types.AgentInfo { return nil }
func (m *MockSpawner) Cancel(id int) error     { return nil }

func TestStructuredSpawning(t *testing.T) {
	mockSpawner := &MockSpawner{}
	spawnTool := tools.NewSpawnAgentTool(mockSpawner)

	// Execute with an agent name to trigger the wrapper logic
	args := `{"task": "Analyze logs", "agent_name": "LogBot"}`
	_, err := spawnTool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Verify the prompts were wrapped
	expectedPrefix := "You are acting as the 'LogBot' agent"
	if !strings.Contains(mockSpawner.capturedTask, expectedPrefix) {
		t.Errorf("Expected task to be wrapped with '%s', got: %s", expectedPrefix, mockSpawner.capturedTask)
	}

	expectedInstruction := "<thought>"
	if !strings.Contains(mockSpawner.capturedTask, expectedInstruction) {
		t.Errorf("Expected task to contain CoT instruction '<thought>', got: %s", mockSpawner.capturedTask)
	}
}
