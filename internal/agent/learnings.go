package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeanpaul/aseity/internal/provider"
)

// ExtractLearnings analyzes the conversation history to find user preferences
// and project insights, saving them to AutoMemory.
func (a *Agent) ExtractLearnings(ctx context.Context) error {
	// Only run if we have a conversation history
	if len(a.conv.Messages()) < 2 {
		return nil
	}

	// 1. Construct the extraction prompt
	prompt := `
You are an insight extractor. Your job is to analyze the conversation and identify 
persistent information that would be useful for future sessions.

Look for:
1. **User Preferences**: "I prefer TypeScript", "Use tabs not spaces", "Always verify with tests"
2. **Project Facts**: "The API is at port 3000", "Use the 'dev' branch"
3. **Corrections**: "Don't use library X", "The database schema has changed"

CRITICAL:
- Only extract *explicit* or *strongly implied* permanent preferences.
- Ignore transient context (e.g., "fix this bug").
- Ignore standard conversational filler.

Return the result as a JSON object:
{
  "learnings": [
    { "category": "preference", "content": "User prefers 'slog' for logging" },
    { "category": "fact", "content": "Project uses wrapping for errors" }
  ]
}

If nothing worth saving is found, return { "learnings": [] }.
`

	// 2. Call the provider (using a separate context/turn to avoiding polluting main history)
	// We send the FULL history + the prompt.
	msgs := append(a.conv.Messages(), provider.Message{
		Role:    provider.RoleSystem,
		Content: prompt,
	})

	// Use a basic tool-less call
	stream, err := a.prov.Chat(ctx, msgs, nil)
	if err != nil {
		return fmt.Errorf("extraction chat failed: %w", err)
	}

	var textBuf strings.Builder
	for chunk := range stream {
		if chunk.Error != nil {
			return chunk.Error
		}
		textBuf.WriteString(chunk.Delta)
	}
	response := textBuf.String()

	// 3. Parse and Save
	response = cleanJSON(response) // Reuse existing helper or implement simple strip

	var result struct {
		Learnings []struct {
			Category string `json:"category"`
			Content  string `json:"content"`
		} `json:"learnings"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// If JSON fails, just log it and move on (not critical)
		return fmt.Errorf("failed to parse learnings: %v", err)
	}

	if len(result.Learnings) > 0 {
		fmt.Printf("\n🧠 Auto-Memory: Saving %d new insights...\n", len(result.Learnings))
		for _, l := range result.Learnings {
			if err := a.autoMemory.AddLearning(l.Category, l.Content); err != nil {
				fmt.Printf("Warning: Failed to add learning: %v\n", err)
			}
		}
		return a.autoMemory.Save()
	}

	return nil
}

// cleanJSON helper to strip markdown code blocks
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
