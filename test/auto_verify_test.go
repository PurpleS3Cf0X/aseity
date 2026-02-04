package test

import (
	"context"
	"strings"
	"testing"

	"github.com/jeanpaul/aseity/internal/tools"
	"github.com/jeanpaul/aseity/internal/types"
)

// MockSpawnerForLoop simulates the Worker-Judge loop
type MockSpawnerForLoop struct {
	callCount int
	calls     []string // Track prompts to verify feedback injection
}

func (m *MockSpawnerForLoop) Spawn(ctx context.Context, task string, files []string, name string) (int, error) {
	m.callCount++
	m.calls = append(m.calls, task)
	return m.callCount, nil
}

func (m *MockSpawnerForLoop) Get(id int) (types.AgentInfo, bool) {
	// Simulate the sequence:
	// 1. Worker -> Bad Code
	// 2. Judge -> FAIL
	// 3. Worker -> Good Code (after retry)
	// 4. Judge -> PASS

	status := "done"
	output := ""

	switch id {
	case 1: // Worker Attempt 1
		output = "Here is the code: func bad() { panic() }"
	case 2: // Judge on Attempt 1
		output = `{"status": "fail", "feedback": "Code invalid: panic usage."}`
	case 3: // Worker Attempt 2
		output = "Here is the fixed code: func good() { return }"
	case 4: // Judge on Attempt 2
		output = `{"status": "pass", "feedback": "LGTM"}`
	default:
		output = "Unknown"
	}

	return types.AgentInfo{
		ID:     id,
		Status: status,
		Output: output,
	}, true
}

func (m *MockSpawnerForLoop) List() []types.AgentInfo { return nil }
func (m *MockSpawnerForLoop) Cancel(id int) error     { return nil }

func TestAutoVerificationLoop(t *testing.T) {
	mock := &MockSpawnerForLoop{}
	tool := tools.NewSpawnAgentTool(mock)

	// Execute with require_review = true
	args := `{"task": "Write code", "require_review": true}`
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 1. Verify Output is the *FINAL* successful output
	if !strings.Contains(res.Output, "func good()") {
		t.Errorf("Expected final result to contain good code, got: %s", res.Output)
	}
	if !strings.Contains(res.Output, "Verified PASS") {
		t.Errorf("Expected result to indicate verification pass")
	}

	// 2. Verify Call Count (Should be 4: Worker, Judge, Worker, Judge)
	if mock.callCount != 4 {
		t.Errorf("Expected 4 distinct agent spawns, got %d", mock.callCount)
	}

	// 3. Verify Feedback Injection in Attempt 2 (Call 3)
	// mock.calls[2] corresponds to the 3rd spawn call (index 2)
	if len(mock.calls) > 2 {
		attempt2Prompt := mock.calls[2]
		if !strings.Contains(attempt2Prompt, "REJECTED") {
			t.Logf("Attempt 2 Prompt: %s", attempt2Prompt)
			for i, c := range mock.calls {
				t.Logf("Call %d: %s", i, c)
			}
			t.Errorf("Expected retry prompt to contain rejection notice")
		}
		if !strings.Contains(attempt2Prompt, "Code invalid: panic usage") {
			t.Errorf("Expected retry prompt to contain the judge's feedback")
		}
	}
}
