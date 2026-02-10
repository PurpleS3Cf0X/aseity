package agent

import (
	"fmt"
	"strings"
)

// ResultProcessor ensures tool results are actually used in responses
type ResultProcessor struct {
	strictMode bool
}

// NewResultProcessor creates a new result processor
func NewResultProcessor(strict bool) *ResultProcessor {
	return &ResultProcessor{strictMode: strict}
}

// BuildResultPrompt creates a prompt to force the model to acknowledge and use tool results
func (rp *ResultProcessor) BuildResultPrompt(toolName, result string) string {
	// Truncate very long results to avoid context overflow
	truncated := truncateResult(result, 1500)

	return fmt.Sprintf(`
🔍 TOOL RESULT from %s:

%s

CRITICAL - You MUST:
1. READ this result carefully
2. EXTRACT the key information
3. USE this ACTUAL data in your response
4. Do NOT make up information
5. If insufficient, call another tool

What did you learn from this result?`, toolName, truncated)
}

// ValidateResultUsage checks if the response uses data from the tool result
func (rp *ResultProcessor) ValidateResultUsage(response, toolResult string) (bool, float64) {
	if !rp.strictMode {
		return true, 1.0
	}

	keywords := extractSignificantWords(toolResult, 15)
	if len(keywords) == 0 {
		return true, 1.0
	}

	matchCount := 0
	responseLower := strings.ToLower(response)
	for _, kw := range keywords {
		if strings.Contains(responseLower, strings.ToLower(kw)) {
			matchCount++
		}
	}

	overlap := float64(matchCount) / float64(len(keywords))
	passed := overlap >= 0.3 // Require 30% keyword overlap

	return passed, overlap
}

// extractSignificantWords extracts meaningful words from text
func extractSignificantWords(text string, maxWords int) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
	}

	words := strings.Fields(text)
	var significant []string
	seen := make(map[string]bool)

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:()[]{}\"'"))

		if len(word) < 4 || stopWords[word] || seen[word] {
			continue
		}

		significant = append(significant, word)
		seen[word] = true

		if len(significant) >= maxWords {
			break
		}
	}

	return significant
}

// truncateResult truncates long text to avoid context overflow
func truncateResult(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	truncated := text[:maxLen]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxLen/2 {
		truncated = text[:lastNewline]
	}

	return truncated + "\n\n[... truncated ...]"
}
