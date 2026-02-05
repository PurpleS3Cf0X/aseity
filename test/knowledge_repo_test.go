package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProviderKnowledge checks if the system prompt contains knowledge paths
type MockProviderKnowledge struct {
	LastSystemPrompt string
}

func (m *MockProviderKnowledge) Chat(ctx context.Context, msgs []provider.Message, defs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	if len(msgs) > 0 {
		for _, msg := range msgs {
			if msg.Role == "system" {
				// Capture the main system prompt (usually the first one or the one with specific content)
				// We want to avoid overwriting it with the "Turn x/y" reminder if that comes later as a system msg.
				if strings.Contains(msg.Content, "You are") || strings.Contains(msg.Content, "Knowledge Base") {
					m.LastSystemPrompt = msg.Content
					break // Found the main prompt
				}
			}
		}
	}

	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{Delta: "Acknowledged.", Done: true}
	}()
	return ch, nil
}
func (m *MockProviderKnowledge) Name() string { return "mock-knowledge" }

func (m *MockProviderKnowledge) ModelName() string { return "test-model" }
func (m *MockProviderFallback) ModelName() string { return "test-model" }
func (m *MockProviderKnowledge) Models(ctx context.Context) ([]string, error) {
	return []string{"mock"}, nil
}

func TestKnowledgeRepoInjection(t *testing.T) {
	// 1. Setup Environment
	tmpDir, err := os.MkdirTemp("", "aseity_knowledge_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Create a dummy knowledge repo
	knowledgeDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// 2. Init Components
	mockProv := &MockProviderKnowledge{}
	toolReg := tools.NewRegistry(nil, true)
	toolReg.Register(tools.NewCreateAgentTool())

	mgr := agent.NewAgentManager(mockProv, toolReg, 3, false)
	toolReg.Register(tools.NewSpawnAgentTool(mgr))

	// 3. Create Agent with Knowledge Path
	createArgs := `{"name": "DocBot", "description": "Knowledge Agent", "system_prompt": "You are a helper.", "knowledge_paths": ["` + knowledgeDir + `"]}`
	createTool := tools.NewCreateAgentTool()
	if _, err := createTool.Execute(context.Background(), createArgs); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	// 4. Spawn Agent
	spawnTool := tools.NewSpawnAgentTool(mgr)
	spawnArgs := `{"agent_name": "DocBot", "task": "Check docs"}`

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := spawnTool.Execute(ctx, spawnArgs); err != nil {
		// Ignore timeout if it happens due to mock loop end
	}

	// 5. Verify Injection
	if !strings.Contains(mockProv.LastSystemPrompt, "## Knowledge Base") {
		t.Errorf("System prompt does not contain 'Knowledge Base' section.\nPrompt: %s", mockProv.LastSystemPrompt)
	}
	if !strings.Contains(mockProv.LastSystemPrompt, knowledgeDir) {
		t.Errorf("System prompt does not contain correct path '%s'.\nPrompt: %s", knowledgeDir, mockProv.LastSystemPrompt)
	}

	t.Log("Success: Knowledge paths injected into system prompt.")
}
