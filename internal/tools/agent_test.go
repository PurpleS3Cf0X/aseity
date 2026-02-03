package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/types"
)

// MockSpawner for testing
type MockSpawner struct {
	CapturedTask  string
	CapturedFiles []string
	CapturedAgent string
	ReturnID      int
	ReturnErr     error

	Agents map[int]types.AgentInfo
}

func (m *MockSpawner) Spawn(ctx context.Context, task string, files []string, agentName string) (int, error) {
	m.CapturedTask = task
	m.CapturedFiles = files
	m.CapturedAgent = agentName

	// Create a mock running agent
	if m.Agents == nil {
		m.Agents = make(map[int]types.AgentInfo)
	}
	m.Agents[m.ReturnID] = types.AgentInfo{
		ID:     m.ReturnID,
		Status: "running",
		Task:   task,
	}

	return m.ReturnID, m.ReturnErr
}

func (m *MockSpawner) List() []types.AgentInfo {
	var list []types.AgentInfo
	for _, a := range m.Agents {
		list = append(list, a)
	}
	return list
}

func (m *MockSpawner) Get(id int) (types.AgentInfo, bool) {
	a, ok := m.Agents[id]
	return a, ok
}

func (m *MockSpawner) Cancel(id int) error {
	if a, ok := m.Agents[id]; ok {
		a.Status = "cancelled"
		m.Agents[id] = a
	}
	return nil
}

// Helper to simulate agent completion
func (m *MockSpawner) CompleteAgent(id int, output string) {
	if a, ok := m.Agents[id]; ok {
		a.Status = "done"
		a.Output = output
		m.Agents[id] = a
	}
}

func TestSpawnAgentTool_Execute(t *testing.T) {
	mock := &MockSpawner{ReturnID: 123}
	tool := NewSpawnAgentTool(mock)

	// Test regular spawning
	args := `{"task": "Research golang", "context_files": ["/tmp/foo.txt"]}`

	// We need to simulate the agent finishing, otherwise Execute blocks waiting for it.
	// SpawnAgentTool polls every 500ms.
	// We can run Execute in a goroutine or just make the mock finish quickly.
	// But MockSpawner is passive.
	// We can set up a goroutine to update the mock status after 1 second.

	go func() {
		time.Sleep(600 * time.Millisecond) // Wait for at least one poll tick
		mock.CompleteAgent(123, "Research completed: Go is great.")
	}()

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify delegation
	if mock.CapturedTask != "Research golang" {
		t.Errorf("Expected task 'Research golang', got '%s'", mock.CapturedTask)
	}
	if len(mock.CapturedFiles) != 1 || mock.CapturedFiles[0] != "/tmp/foo.txt" {
		t.Errorf("Context files not passed correctly: %v", mock.CapturedFiles)
	}

	// Verify output
	if !strings.Contains(result.Output, "Agent #123 completed") {
		t.Errorf("Expected completion message, got: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Go is great") {
		t.Errorf("Expected agent output, got: %s", result.Output)
	}
}

func TestListAgentsTool_Execute(t *testing.T) {
	mock := &MockSpawner{
		Agents: map[int]types.AgentInfo{
			1: {ID: 1, Status: "running", Task: "Task 1"},
			2: {ID: 2, Status: "done", Task: "Task 2"},
		},
	}
	tool := NewListAgentsTool(mock)

	result, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Agent #1 [running]") {
		t.Errorf("Missing Agent #1 in output: %s", result.Output)
	}
	if !strings.Contains(result.Output, "Agent #2 [done]") {
		t.Errorf("Missing Agent #2 in output: %s", result.Output)
	}
}
