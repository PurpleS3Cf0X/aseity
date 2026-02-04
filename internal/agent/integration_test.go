package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// Mock tool for cancellation test
type SlowTool struct{}

func (s *SlowTool) Name() string            { return "slow_tool" }
func (s *SlowTool) Description() string     { return "Slow tool" }
func (s *SlowTool) Parameters() any         { return nil }
func (s *SlowTool) NeedsConfirmation() bool { return false }
func (s *SlowTool) Execute(ctx context.Context, args string) (tools.Result, error) {
	// Respect context cancellation
	select {
	case <-time.After(5 * time.Second):
		return tools.Result{Output: "Done"}, nil
	case <-ctx.Done():
		return tools.Result{Error: "Cancelled"}, ctx.Err()
	}
}

// Test 1: Parallel Execution with Cancellation
func TestParallelExecutionCancellation(t *testing.T) {
	// Mock slow tool that respects context
	slowTool := &SlowTool{}

	reg := tools.NewRegistry(nil, true)
	reg.Register(slowTool)

	// Mock provider that returns 3 parallel tool calls
	prov := &MockProviderParallelCancel{}
	agent := New(prov, reg, "")

	events := make(chan Event, 100)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	go agent.Send(ctx, "Run slow tools", events)

	// Drain events
	for evt := range events {
		if evt.Type == EventError && evt.Error == "Parallel tool execution interrupted by cancellation" {
			t.Log("Received expected cancellation event")
		}
		if evt.Done {
			break
		}
	}

	duration := time.Since(start)
	if duration > 2*time.Second {
		t.Errorf("Agent took too long to respond to cancellation: %v", duration)
	}
	t.Logf("Agent responded to cancellation in %v", duration)
}

// Mock provider for cancellation test
type MockProviderParallelCancel struct {
	called int
}

func (m *MockProviderParallelCancel) Chat(ctx context.Context, messages []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		m.called++
		if m.called == 1 {
			// Return 3 slow tool calls
			ch <- provider.StreamChunk{
				ToolCalls: []provider.ToolCall{
					{ID: "1", Name: "slow_tool", Args: "{}"},
					{ID: "2", Name: "slow_tool", Args: "{}"},
					{ID: "3", Name: "slow_tool", Args: "{}"},
				},
				Done: true,
			}
		} else {
			ch <- provider.StreamChunk{Delta: "Done", Done: true}
		}
	}()
	return ch, nil
}

func (m *MockProviderParallelCancel) Name() string { return "mock" }
func (m *MockProviderParallelCancel) Models(ctx context.Context) ([]string, error) {
	return []string{"mock"}, nil
}

// Test 2: Quality Gate Retry Exhaustion
func TestQualityGateRetryLimit(t *testing.T) {
	// Mock judge that always fails
	judge := &MockJudgeTool{ShouldPass: false}
	reg := tools.NewRegistry(nil, true)
	reg.Register(judge)

	// Mock provider
	prov := &MockProviderGate{
		responses: []string{
			"Attempt 1",
			"Attempt 2",
			"Attempt 3",
			"Attempt 4", // Should never reach this
		},
	}

	agent := New(prov, reg, "")
	agent.QualityGateEnabled = true
	agent.OriginalGoal = "Test goal"

	events := make(chan Event, 100)
	go agent.Send(context.Background(), "Start", events)

	retryCount := 0
	maxRetrySeen := false

	for evt := range events {
		t.Logf("Event: Type=%v Error='%s' Done=%v", evt.Type, evt.Error, evt.Done)
		if evt.Type == EventError && evt.Done == false {
			// Rejection event (Quality Gate Rejected: ...)
			if strings.HasPrefix(evt.Error, "Quality Gate Rejected") {
				retryCount++
			}
		}
		if evt.Type == EventError && evt.Done == true {
			// Final error
			maxRetrySeen = true
			t.Logf("Received expected max retry error: %s", evt.Error)
			break
		}
	}

	if !maxRetrySeen {
		t.Errorf("Expected quality gate to stop after %d retries, but didn't see final error", MaxQualityGateRetries)
	}

	// We get MaxQualityGateRetries-1 rejection events, then the final error
	// (Attempt 1 fails -> reject event, Attempt 2 fails -> reject event, Attempt 3 hits limit -> final error)
	expectedRejections := MaxQualityGateRetries - 1
	if retryCount != expectedRejections {
		t.Errorf("Expected exactly %d rejection events before final error, got %d", expectedRejections, retryCount)
	}
}

// Test 3: Thread Safety
func TestConversationThreadSafety(t *testing.T) {
	conv := NewConversation()

	var wg sync.WaitGroup
	iterations := 100

	// Spawn 10 goroutines that concurrently add messages
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				conv.AddUser("User message")
				conv.AddAssistant("Assistant message", nil)
				conv.AddSystem("System message")
				conv.AddToolResult("tool-id", "Tool result")
			}
		}(i)
	}

	wg.Wait()

	// Verify final count
	expectedCount := 10 * iterations * 4 // 4 messages per iteration
	actualCount := conv.Len()

	if actualCount != expectedCount {
		t.Errorf("Expected %d messages, got %d (possible race condition)", expectedCount, actualCount)
	}

	t.Logf("Thread safety test passed: %d messages added concurrently", actualCount)
}

// Test 4: Channel Buffering Enforcement
func TestChannelBufferingEnforcement(t *testing.T) {
	prov := &MockProviderGate{responses: []string{"Test"}}
	reg := tools.NewRegistry(nil, true)
	agent := New(prov, reg, "")

	// Test with unbuffered channel
	unbuffered := make(chan Event)
	go agent.Send(context.Background(), "Test", unbuffered)

	evt := <-unbuffered
	if evt.Type != EventError {
		t.Errorf("Expected error event for unbuffered channel, got %v", evt.Type)
	}
	if evt.Error == "" || evt.Done != true {
		t.Errorf("Expected error message about buffering, got: %s", evt.Error)
	}
	t.Logf("Unbuffered channel correctly rejected: %s", evt.Error)

	// Test with adequately buffered channel
	buffered := make(chan Event, 100)
	go agent.Send(context.Background(), "Test", buffered)

	foundNonError := false
	for evt := range buffered {
		if evt.Type != EventError {
			foundNonError = true
		}
		if evt.Done {
			break
		}
	}

	if !foundNonError {
		t.Error("Expected non-error events with buffered channel")
	}
	t.Log("Buffered channel correctly accepted")
}

// Test 5: Mixed Parallel/Sequential Execution
func TestMixedParallelSequential(t *testing.T) {
	// Register both parallel-safe and sequential tools
	reg := tools.NewRegistry(nil, true)
	reg.Register(&MockFastTool{name: "web_search"})  // Parallel-safe
	reg.Register(&MockFastTool{name: "web_crawl"})   // Parallel-safe
	reg.Register(&MockFastTool{name: "run_command"}) // Sequential

	prov := &MockProviderMixed{}
	agent := New(prov, reg, "")

	events := make(chan Event, 100)
	start := time.Now()

	go agent.Send(context.Background(), "Run mixed tools", events)

	parallelCount := 0
	sequentialCount := 0

	for evt := range events {
		if evt.Type == EventToolCall {
			if evt.ToolName == "web_search" || evt.ToolName == "web_crawl" {
				parallelCount++
			} else if evt.ToolName == "run_command" {
				sequentialCount++
			}
		}
		if evt.Done {
			break
		}
	}

	duration := time.Since(start)

	if parallelCount != 2 {
		t.Errorf("Expected 2 parallel tools, got %d", parallelCount)
	}
	if sequentialCount != 1 {
		t.Errorf("Expected 1 sequential tool, got %d", sequentialCount)
	}

	// Parallel tools should execute concurrently, so total time should be < 2 * 100ms
	if duration > 250*time.Millisecond {
		t.Errorf("Execution took too long (%v), parallel optimization may not be working", duration)
	}

	t.Logf("Mixed execution completed in %v", duration)
}

// Mock fast tool for mixed test
type MockFastTool struct {
	name string
}

func (m *MockFastTool) Name() string            { return m.name }
func (m *MockFastTool) Description() string     { return "Fast tool" }
func (m *MockFastTool) Parameters() any         { return nil }
func (m *MockFastTool) NeedsConfirmation() bool { return false }
func (m *MockFastTool) Execute(ctx context.Context, args string) (tools.Result, error) {
	time.Sleep(100 * time.Millisecond)
	return tools.Result{Output: "Done"}, nil
}

// Mock provider for mixed test
type MockProviderMixed struct {
	called int
}

func (m *MockProviderMixed) Chat(ctx context.Context, messages []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		m.called++
		if m.called == 1 {
			// Return 2 parallel + 1 sequential
			ch <- provider.StreamChunk{
				ToolCalls: []provider.ToolCall{
					{ID: "1", Name: "web_search", Args: "{}"},
					{ID: "2", Name: "web_crawl", Args: "{}"},
					{ID: "3", Name: "run_command", Args: "{}"},
				},
				Done: true,
			}
		} else {
			ch <- provider.StreamChunk{Delta: "Done", Done: true}
		}
	}()
	return ch, nil
}

func (m *MockProviderMixed) Name() string { return "mock" }
func (m *MockProviderMixed) Models(ctx context.Context) ([]string, error) {
	return []string{"mock"}, nil
}
