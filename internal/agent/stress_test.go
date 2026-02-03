package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
)

// TestStressConversation_12Turns runs a real conversation against local Ollama
// to verify stability over multiple turns.
func TestStressConversation_12Turns(t *testing.T) {
	// 1. Setup Provider (Real Ollama)
	// We use the default local URL
	prov := provider.NewOpenAI("ollama", "http://localhost:11434/v1", "", "qwen2.5:3b")

	// Verify connectivity first
	models, err := prov.Models(context.Background())
	if err != nil {
		t.Skipf("Skipping stress test: Ollama not reachable: %v", err)
	}
	found := false
	for _, m := range models {
		if m == "qwen2.5:3b" {
			found = true
			break
		}
	}
	if !found {
		t.Skipf("Skipping stress test: qwen2.5:3b not found in %v", models)
	}

	// 2. Initialize Conversation History
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "You are a helpful counting assistant. respond briefly."},
	}

	// 3. Run 12 Turns
	for i := 1; i <= 12; i++ {
		t.Logf("--- Turn %d ---", i)

		// Add User Message
		prompt := fmt.Sprintf("Please say the number %d and nothing else.", i)
		msgs = append(msgs, provider.Message{Role: provider.RoleUser, Content: prompt})

		// Get Response
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stream, err := prov.Chat(ctx, msgs, nil)
		if err != nil {
			t.Fatalf("Turn %d failed to start chat: %v", i, err)
		}

		fullResponse := ""
		for chunk := range stream {
			if chunk.Error != nil {
				t.Fatalf("Turn %d stream error: %v", i, chunk.Error)
			}
			fullResponse += chunk.Delta
		}

		if fullResponse == "" {
			t.Errorf("Turn %d returned empty response", i)
		}
		t.Logf("Agent: %s", fullResponse)

		// Append Assistant Message to History
		msgs = append(msgs, provider.Message{Role: provider.RoleAssistant, Content: fullResponse})
	}
}
