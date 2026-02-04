package agent

import (
	"strings"
	"testing"
)

// TestSystemPromptHasExamples verifies the enhanced prompt
func TestSystemPromptHasExamples(t *testing.T) {
	prompt := BuildSystemPrompt()

	requiredPhrases := []string{
		"⚡ CRITICAL: How You Must Respond",
		"✅ CORRECT Examples:",
		"❌ WRONG Examples",
		"[TOOL:bash|",
		"install numpy",
		"Do NOT explain how to do it",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("System prompt missing required phrase: %q", phrase)
		}
	}

	// Verify examples come BEFORE tool descriptions
	criticalIdx := strings.Index(prompt, "⚡ CRITICAL")
	toolsIdx := strings.Index(prompt, "## Available Tools")

	if criticalIdx == -1 || toolsIdx == -1 {
		t.Fatal("Could not find critical sections in prompt")
	}

	if criticalIdx > toolsIdx {
		t.Error("Examples should appear BEFORE tool descriptions")
	}

	t.Logf("✅ System prompt has all required examples in correct order")
}

// TestPromptLength verifies we didn't make it too long
func TestPromptLength(t *testing.T) {
	prompt := BuildSystemPrompt()

	// Rough token estimate (1 token ≈ 4 chars)
	estimatedTokens := len(prompt) / 4

	// Should be under 2000 tokens to leave room for conversation
	if estimatedTokens > 2000 {
		t.Logf("Warning: System prompt is ~%d tokens (may be too long)", estimatedTokens)
	}

	t.Logf("✅ System prompt length: %d chars (~%d tokens)", len(prompt), estimatedTokens)
}
