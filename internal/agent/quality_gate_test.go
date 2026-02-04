package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProviderGate for testing
type MockProviderGate struct {
	responses []string
	index     int
}

func (m *MockProviderGate) Chat(ctx context.Context, msgs []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		if m.index < len(m.responses) {
			resp := m.responses[m.index]
			m.index++
			ch <- provider.StreamChunk{Delta: resp, Done: true}
		} else {
			ch <- provider.StreamChunk{Done: true}
		}
		close(ch)
	}()
	return ch, nil
}
func (m *MockProviderGate) Name() string { return "mock" }
func (m *MockProviderGate) Models(ctx context.Context) ([]string, error) {
	return []string{"mock"}, nil
}

// MockJudgeTool simulates pass/fail output
type MockJudgeTool struct {
	mu         sync.Mutex
	ShouldPass bool
}

func (j *MockJudgeTool) Name() string            { return "judge_output" }
func (j *MockJudgeTool) Description() string     { return "Mock judge" }
func (j *MockJudgeTool) Parameters() any         { return nil }
func (j *MockJudgeTool) NeedsConfirmation() bool { return false }
func (j *MockJudgeTool) Execute(ctx context.Context, args string) (tools.Result, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	status := "fail"
	feedback := "Needs improvement"
	if j.ShouldPass {
		status = "pass"
		feedback = "LGTM"
	}
	res := map[string]string{
		"status":   status,
		"feedback": feedback,
	}
	b, _ := json.Marshal(res)
	return tools.Result{Output: string(b)}, nil
}

func TestQualityGate_Rejection(t *testing.T) {
	// Setup
	prov := &MockProviderGate{
		responses: []string{
			"I have done the task.", // Attempt 1: Should trigger judge, fail, and be rejected.
			"Okay I fixed it.",      // Attempt 2: Should trigger judge, pass, and finish.
		},
	}

	judge := &MockJudgeTool{ShouldPass: false} // Start failing
	reg := tools.NewRegistry(nil, true)
	reg.Register(judge)

	// Create Agent with Gate Enabled
	a := New(prov, reg, "")
	a.QualityGateEnabled = true
	a.OriginalGoal = "Build a rocket"

	events := make(chan Event, 100)

	// We need to run the loop in a goroutine but allow us to control the mock judge
	// Actually, the loop runs until completion.
	// Since MockProvider has finite responses, it will eventually stop if logic is correct.

	// BUT: Wait, if rejection happens, the loop CONTINUES. MaxTurns will kill it if we don't pass.
	// We want to verify that we received a rejection event first.

	// Let's modify the MockJudgeTool relative to call count?
	// Or we just consume events and see what happens.

	go func() {
		a.Send(context.Background(), "Start", events)
		close(events)
	}()

	judgeCalled := 0
	rejectionSeen := false
	successSeen := false

	for evt := range events {
		t.Logf("Received Event: Type=%d Text='%s' Error='%s' Done=%v", evt.Type, evt.Text, evt.Error, evt.Done)
		if evt.Type == EventJudgeCall {
			judgeCalled++
			// After first judge call (fail), next one should be pass
			if judgeCalled == 1 {
				judge.mu.Lock()
				judge.ShouldPass = true // Allow pass on second try
				judge.mu.Unlock()
			}
		}
		if evt.Type == EventError && evt.Done == false {
			// This is our rejection signal: "Quality Gate Rejected..."
			rejectionSeen = true
		}
		if evt.Type == EventDone {
			successSeen = true
		}
	}

	if !rejectionSeen {
		t.Error("Expected Quality Gate to reject the first attempt, but no rejection event seen")
	}
	if !successSeen {
		t.Error("Expected Quality Gate to eventually pass and finish, but didn't")
	}
	// We expect 3 judge events:
	// 1. "Evaluating..." (Attempt 1)
	// 2. "Evaluating..." (Attempt 2)
	// 3. "Quality Gate Passed" (Attempt 2 Success)
	if judgeCalled != 3 {
		t.Errorf("Expected 3 Judge events, got %d", judgeCalled)
	}
}
