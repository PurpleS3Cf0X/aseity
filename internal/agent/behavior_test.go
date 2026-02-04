package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
)

// MockProviderBehavior records the response style to verify behavioral modes
type MockProviderBehavior struct {
	CapturedResponse string
}

func (m *MockProviderBehavior) Chat(ctx context.Context, msgs []provider.Message, defs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)
		// Simple heuristic response based on input
		lastMsg := msgs[len(msgs)-1].Content

		var response string

		if strings.Contains(lastMsg, "install") {
			// Simulating Action Mode: Correct tool call
			response = `[TOOL:bash|{"command": "npm install pkg"}]`
		} else if strings.Contains(lastMsg, "plan") {
			// Simulating Planning Mode: Thought block
			response = `<thought>I need to analyze requirements first.</thought> Here is the plan...`
		} else if strings.Contains(lastMsg, "explain") {
			// Simulating Explanation Mode: Plain text
			response = "The reason we use Go is for performance."
		} else {
			// Fallback
			response = "I am ready."
		}

		m.CapturedResponse = response

		// Stream it back
		ch <- provider.StreamChunk{
			Delta: response,
		}
	}()

	return ch, nil
}

// TestBehavioralModes verifies that the prompt *encourages* the correct behavior.
// Since we can't fully deterministically test an LLM's adherence without an actual LLM,
// this test verifies that the *system prompt* contains the expected keywords,
// and acts as a harness for manual verification if connected to a real model.
func TestBehavioralModes(t *testing.T) {
	prompt := BuildSystemPrompt()

	// Verify System Prompt Content
	requiredPhrases := []string{
		"Behavioral Protocol",
		"Planning Mode",
		"Action Mode",
		"Explanation Mode",
		"Action Bias",
		"Recusive Task Decomposition",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(prompt, phrase) {
			// Check for typos in my expectation or the file
			if phrase == "Recusive Task Decomposition" {
				if strings.Contains(prompt, "Recursive Task Decomposition") {
					continue
				}
			}
			t.Errorf("System prompt missing key phrase: %q", phrase)
		}
	}
}
