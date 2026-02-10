package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProvider for recursion test
type MockRecProvider struct {
	responses []string
	callCount int
}

func (m *MockRecProvider) Chat(ctx context.Context, history []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)

	if m.callCount >= len(m.responses) {
		go func() {
			ch <- provider.StreamChunk{Delta: "No more mock responses", Done: true}
			close(ch)
		}()
		return ch, nil
	}

	resp := m.responses[m.callCount]
	m.callCount++

	go func() {
		defer close(ch)
		// Check if response is a tool call simulation
		if strings.HasPrefix(resp, "TOOL:") {
			parts := strings.SplitN(resp, "|", 3)
			if len(parts) >= 2 {
				toolName := strings.TrimPrefix(parts[0], "TOOL:")
				toolArgs := parts[1]
				ch <- provider.StreamChunk{
					ToolCalls: []provider.ToolCall{
						{
							ID:   "call_1",
							Name: toolName,
							Args: toolArgs,
						},
					},
					Done: true,
				}
				return
			}
		}

		// Regular text
		ch <- provider.StreamChunk{Delta: resp, Done: true}
	}()

	return ch, nil
}

func (m *MockRecProvider) Name() string      { return "mock" }
func (m *MockRecProvider) ModelName() string { return "mock-model" }
func (m *MockRecProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestRecursiveAgentSpawning(t *testing.T) {
	// 1. Setup
	// Root agent spawns 2 sub-agents
	// Sub-agents return success
	// Root agent waits for both

	mockProv := &MockRecProvider{
		responses: []string{
			// Root Agent: Spawns Agent A
			`TOOL:spawn_agent|{"task": "Task A", "background": true}`,
			// Root Agent: Spawns Agent B
			`TOOL:spawn_agent|{"task": "Task B", "background": true}`,
			// Root Agent: Waits for both
			`TOOL:wait_all_agents|{"agent_ids": [1, 2]}`,
			// Root Agent: Finished
			`Mission accomplished.`,

			// Sub-Agent A (ID 1): Runs and finishes
			`Task A is done.`,

			// Sub-Agent B (ID 2): Runs and finishes
			`Task B is done.`,
		},
	}

	toolReg := tools.NewRegistry([]string{"spawn_agent", "wait_all_agents"}, true)
	// We need to register the tools MANUALLY because main.go does it normally
	// But SpawnAgentTool needs the manager, which we are about to create.

	am := NewAgentManager(mockProv, toolReg, 5, false)

	// Register tools to registry linked to manager
	toolReg.Register(tools.NewSpawnAgentTool(am))
	toolReg.Register(tools.NewWaitAllAgentsTool(am))

	// 2. Run Root Agent
	rootAgent := New(mockProv, toolReg, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events := make(chan Event, 100)
	go rootAgent.Send(ctx, "Coordinate Task A and Task B", events)

	// 3. Monitor
	done := false
	for !done {
		select {
		case evt := <-events:
			if evt.Type == EventError {
				t.Fatalf("Root agent error: %s", evt.Error)
			}
			if evt.Type == EventDone {
				done = true
			}
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}

	// 4. Verification
	agents := am.List()
	if len(agents) != 2 {
		t.Errorf("Expected 2 sub-agents, got %d", len(agents))
	}

	for _, a := range agents {
		if a.Status != "done" {
			t.Errorf("Agent %d status is %s, expected done", a.ID, a.Status)
		}
	}
}
