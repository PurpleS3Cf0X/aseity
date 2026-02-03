package agent

import (
	"strings"
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
)

func TestConversation_Compact(t *testing.T) {
	conv := NewConversation()
	conv.SetMaxTokens(100) // Low limit to force compaction effectively

	// Add system prompt
	conv.AddSystem("System Prompt")

	// Add many messages
	// Since MaxTokens is low, this loop will trigger multiple compactions.
	// We want to verify that it doesn't crash and keeps recent history.
	for i := 0; i < 20; i++ {
		conv.AddUser(strings.Repeat("User message ", 5))
		conv.AddAssistant(strings.Repeat("Assistant response ", 5), nil)
	}

	finalLen := conv.Len()
	// Based on Compact() logic:
	// 1 System
	// 1 User Summary
	// 1 Assistant Confirmation
	// 6 Recent Messages (3 pairs)
	// Output should be around 9 or 10 depending on when the last compaction happened relative to the last add.
	// In our loop, AddUser triggers compaction. AddAssistant adds one more.
	// So if compaction happened on the last AddUser, we'd have 9 + 1 = 10 messages.
	if finalLen > 15 {
		t.Errorf("Auto-compaction failed: got %d messages, expected reduced count (~10)", finalLen)
	}

	msgs := conv.Messages()
	if msgs[0].Role != provider.RoleSystem {
		t.Error("System prompt was lost or moved during compaction")
	}

	// Check for summary message
	hasSummary := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "[Conversation summary]") {
			hasSummary = true
			break
		}
	}
	if !hasSummary {
		t.Error("Summary message not found after compaction")
	}

	t.Logf("Final message count: %d", finalLen)
}

func TestConversation_TokenEstimation(t *testing.T) {
	conv := NewConversation()
	conv.AddUser("Hello world") // 11 chars -> ~2 tokens

	est := conv.EstimatedTokens()
	if est == 0 {
		t.Error("Estimated tokens should be > 0")
	}
}
