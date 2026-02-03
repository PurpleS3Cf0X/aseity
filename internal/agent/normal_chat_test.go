package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

func TestNormalChat_NoTools(t *testing.T) {
	prov := provider.NewOpenAI("ollama", "http://localhost:11434/v1", "", "qwen2.5:3b")

	models, err := prov.Models(context.Background())
	if err != nil {
		t.Skipf("Skipping: Ollama not reachable: %v", err)
	}
	found := false
	for _, m := range models {
		if m == "qwen2.5:3b" {
			found = true
			break
		}
	}
	if !found {
		t.Skipf("Skipping: qwen2.5:3b not found")
	}

	// Empty registry = no tools
	reg := tools.NewRegistry(nil, false)
	agent := New(prov, reg)

	// Turn 1
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events := make(chan Event)
	go func() {
		agent.Send(ctx, "What is 2 + 2? Answer with just the number.", events)
		close(events)
	}()

	var resp1 string
	for evt := range events {
		if evt.Type == EventDelta {
			resp1 += evt.Text
		}
		if evt.Type == EventError {
			t.Fatalf("Error: %s", evt.Error)
		}
	}
	t.Logf("Response 1: %s", resp1)
	if !strings.Contains(resp1, "4") {
		t.Errorf("Expected 4, got %s", resp1)
	}

	// Turn 2 using THE SAME AGENT (memory test)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	events2 := make(chan Event)
	go func() {
		agent.Send(ctx2, "Double that number.", events2)
		close(events2)
	}()

	var resp2 string
	for evt := range events2 {
		if evt.Type == EventDelta {
			resp2 += evt.Text
		}
	}
	t.Logf("Response 2: %s", resp2)
	if !strings.Contains(resp2, "8") {
		t.Errorf("Expected 8 (double 4), got %s", resp2)
	}
}
