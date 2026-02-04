package agent

import (
	"context"
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProvider for testing agents
type MockProvider struct{}

func (m *MockProvider) Chat(ctx context.Context, messages []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		// Simple response simulating a done agent
		ch <- provider.StreamChunk{
			Delta: "I have completed the sub-task.",
			Done:  true,
		}
		close(ch)
	}()
	return ch, nil
}

func (m *MockProvider) Name() string { return "mock" }
func (m *MockProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestRecursiveSpawn(t *testing.T) {
	// Setup
	prov := &MockProvider{}
	toolReg := tools.NewRegistry(nil, true)
	am := NewAgentManager(prov, toolReg, 3)

	// Register tools manually as in main.go
	spawnTool := tools.NewSpawnAgentTool(am)
	waitTool := tools.NewWaitForAgentTool(am)
	toolReg.Register(spawnTool)
	toolReg.Register(waitTool)

	ctx := context.Background()

	// 1. Test Sync Spawn
	// We'll call the tool execute method directly
	args := `{"task": "echo hello"}`
	res, err := spawnTool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Sync spawn failed: %v", err)
	}
	if res.Error != "" {
		t.Fatalf("Sync spawn returned error: %s", res.Error)
	}

	// 2. Test Async Spawn
	argsAsync := `{"task": "sleep 1", "background": true}`
	resAsync, err := spawnTool.Execute(ctx, argsAsync)
	if err != nil {
		t.Fatalf("Async spawn failed: %v", err)
	}
	if resAsync.Error != "" {
		t.Fatalf("Async spawn returned error: %s", resAsync.Error)
	}
	// Extract ID? The message says "Agent #%d spawned..."
	// We can trust the internal manager state for this test.
	agents := am.List()
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}
