package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProvider intercepts Chat calls to verify context
type MockProvider struct {
	LastMessages []provider.Message
}

func (m *MockProvider) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	m.LastMessages = msgs
	ch := make(chan provider.StreamChunk)
	close(ch)
	return ch, nil
}
func (m *MockProvider) Name() string                                 { return "mock" }
func (m *MockProvider) Models(ctx context.Context) ([]string, error) { return []string{"mock"}, nil }

func TestRecursiveAgent_ContextLoading(t *testing.T) {
	// 1. Setup Mock Provider
	mockProv := &MockProvider{}

	// 2. Setup Agent Manager
	// 2. Setup Agent Manager
	toolReg := tools.NewRegistry(nil, false)
	mgr := NewAgentManager(mockProv, toolReg, 1)

	// 3. Define a real file to load as context
	absPath, _ := filepath.Abs("../../test_data/secret_plan.txt")

	// 4. Spawn an agent with this file in context_files
	// This simulates the parent agent calling: spawn_agent(task="review plan", context_files=[".../secret_plan.txt"])
	id, err := mgr.Spawn(context.Background(), "Review the plan", []string{absPath})
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// 5. Wait briefly for the goroutine to start and call Chat()
	// Since everything is local/mocked, 100ms is plenty.
	time.Sleep(100 * time.Millisecond)

	// 6. Inspect what the sub-agent "saw"
	// The sub-agent should have called Chat() immediately.
	// We check mockProv.LastMessages for the file content.
	foundContext := false
	for _, msg := range mockProv.LastMessages {
		if strings.Contains(msg.Content, "secret_plan.txt") &&
			strings.Contains(msg.Content, "exhaust ports") {
			foundContext = true
			break
		}
	}

	if !foundContext {
		t.Errorf("Sub-agent did not receive context file content. Messages:\n%+v", mockProv.LastMessages)
	} else {
		t.Logf("Success! Sub-agent #%d received context file content.", id)
	}

	// Cleanup
	mgr.Cancel(id)
}
