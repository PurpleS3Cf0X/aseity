package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/types"
)

// MockSpawnerForJudge tests if the JudgeTool correctly spawns a critic
type MockSpawnerForJudge struct {
	capturedTask string
}

func (m *MockSpawnerForJudge) Spawn(ctx context.Context, task string, files []string, name string) (int, error) {
	m.capturedTask = task
	return 101, nil
}

func (m *MockSpawnerForJudge) Get(id int) (types.AgentInfo, bool) {
	// Return a fake valid verdict
	return types.AgentInfo{
		ID:     id,
		Status: "done",
		Output: `{"status": "fail", "feedback": "Logic error in loop condition."}`,
	}, true
}

func (m *MockSpawnerForJudge) List() []types.AgentInfo { return nil }
func (m *MockSpawnerForJudge) Cancel(id int) error     { return nil }

func TestJudgeToolExecution(t *testing.T) {
	mock := &MockSpawnerForJudge{}
	tool := tools.NewJudgeTool(mock)

	// 1. Execute the tool
	args := `{"original_goal": "Sum numbers", "content": "func sum(n int) int { return n*n }"}`
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 2. Verify Spawner was called with correct Prompt
	if !strings.Contains(mock.capturedTask, "STRICT CRITIC") {
		t.Errorf("Expected prompt to contain 'STRICT CRITIC', got: %s", mock.capturedTask)
	}
	if !strings.Contains(mock.capturedTask, "ORIGINAL GOAL") {
		t.Errorf("Expected prompt to include original goal")
	}

	// 3. Verify Output Parsing
	// The mock returns a FAIL verdict
	if !strings.Contains(res.Output, `"status":"fail"`) {
		t.Errorf("Expected parsed JSON output to contain status:fail, got: %s", res.Output)
	}
}

// MockSpawnerForMaliciousContent tests safety/pass-through
type MockSpawnerForPass struct {
	*MockSpawnerForJudge
}

func (m *MockSpawnerForPass) Get(id int) (types.AgentInfo, bool) {
	return types.AgentInfo{
		ID:     id,
		Status: "done",
		Output: "```json\n{\"status\": \"pass\", \"feedback\": \"LGTM\"}\n```", // Markdown wrapped
	}, true
}

func TestJudgeToolMarkdownHandling(t *testing.T) {
	mock := &MockSpawnerForPass{&MockSpawnerForJudge{}}
	tool := tools.NewJudgeTool(mock)

	args := `{"original_goal": "X", "content": "Y"}`
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should extract inner JSON
	if !strings.Contains(res.Output, `"status":"pass"`) {
		t.Errorf("Expected clean JSON output, got: %s", res.Output)
	}
	if strings.Contains(res.Output, "```") {
		t.Errorf("Output should be stripped of markdown blocks: %s", res.Output)
	}
}
