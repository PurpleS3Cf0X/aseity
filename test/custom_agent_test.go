package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProvider intercepts Chat calls to verify system prompts
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

func TestCustomAgentLifecycle(t *testing.T) {
	// 1. Setup Environment
	// Use a temp dir for agents config to avoid messing up user's home
	tmpDir, err := os.MkdirTemp("", "aseity_test_agents")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Mock the GetAgentsDir function or Config loading path?
	// Since GetAgentsDir uses os.UserHomeDir, we can trick it by setting HOME env var for this test process?
	// Or we can modify config to support an override.
	// Setting HOME is safer for integration test without code changes.
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	// Ensure config dir exists
	configDir := filepath.Join(tmpDir, ".config", "aseity", "agents")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// 2. Initialize Components
	mockProv := &MockProvider{}
	toolReg := tools.NewRegistry(nil, true) // Allow all tools
	toolReg.Register(tools.NewCreateAgentTool())

	mgr := agent.NewAgentManager(mockProv, toolReg, 1)
	toolReg.Register(tools.NewSpawnAgentTool(mgr))

	// 3. Test: Create Custom Agent via Tool
	createTool := tools.NewCreateAgentTool()
	createArgs := `{"name": "TestBot", "description": "Just testing", "system_prompt": "You are a test bot."}`
	res, err := createTool.Execute(context.Background(), createArgs)
	if err != nil {
		t.Fatalf("CreateAgentTool failed: %v", err)
	}
	t.Logf("Create Output: %s", res.Output)

	// Verify persistence file exists
	expectedFile := filepath.Join(configDir, "TestBot.yaml")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Agent config file was not created at %s", expectedFile)
	}

	// 4. Test: Spawn Custom Agent
	// Spawning involves running a goroutine. We need to wait for it to start.
	id, err := mgr.Spawn(context.Background(), "Say hello", nil, "TestBot")
	if err != nil {
		t.Fatalf("Spawn failed: %v", err)
	}

	// Wait for agent to initialize and call Chat
	time.Sleep(200 * time.Millisecond)

	// 5. Verify System Prompt
	// Check if the mock provider received the custom system prompt
	foundPrompt := false
	for _, msg := range mockProv.LastMessages {
		if msg.Role == "system" && msg.Content == "You are a test bot." {
			foundPrompt = true
			break
		}
	}

	if !foundPrompt {
		t.Errorf("Custom system prompt not found in sub-agent messages. Messages: %+v", mockProv.LastMessages)
	} else {
		t.Log("Success: Custom system prompt injection verified.")
	}

	// Cleanup
	mgr.Cancel(id)
}
