package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jeanpaul/aseity/internal/provider"
)

// Validator is responsible for checking tool calls before execution.
type Validator struct {
	prov provider.Provider
}

// NewValidator creates a new Validator instance.
func NewValidator(prov provider.Provider) *Validator {
	return &Validator{prov: prov}
}

// Check validates a proposed tool call against the conversation history.
// It returns true if the tool call is valid, or false and a reason if it should be rejected.
func (v *Validator) Check(ctx context.Context, history []provider.Message, toolCall provider.ToolCall) (bool, string) {
	// 1. Construct the validation prompt
	// logic: We ask the model to act as a QA Lead and review the tool call.

	// Extract the last user message to understand intent
	userIntent := "Unknown"
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			userIntent = history[i].Content
			break
		}
	}

	prompt := fmt.Sprintf(`
SYSTEM: Security Check.
USER REQUEST: "%s"
TOOL CALL: %s %s

INSTRUCTION:
Did the user explicitly ask for the specific path/filename arguments used above?
- If the user did NOT mention them, and they look like specific placeholders (e.g. /Users/r3d0ne, /tmp/foo), you MUST reject.
- If the user implied them (e.g. "list current dir"), it is VALID.

OUTPUT ONLY: "VALID" or "INVALID: [Reason]"
`, userIntent, toolCall.Name, toolCall.Args)

	// 2. Call the provider (fast check)
	// We use the same provider for now. In the future, we could use a smaller/faster model.
	msgs := []provider.Message{
		{Role: "system", Content: "You are a validation engine. Output only VALID or INVALID:[Reason]."},
		{Role: "user", Content: prompt},
	}

	// We don't want the validator to call tools, just text.
	stream, err := v.prov.Chat(ctx, msgs, nil)
	if err != nil {
		// If validation fails technically, we default to allow (fail open) or block (fail closed).
		// For now, let's log and allow to avoid blocking valid work due to API errors.
		return true, ""
	}

	response := ""
	for chunk := range stream {
		if chunk.Error != nil {
			return true, ""
		}
		response += chunk.Delta
	}

	// 3. Parse response (relaxed)
	upperResp := strings.ToUpper(response)
	if strings.Contains(upperResp, "INVALID") || strings.Contains(upperResp, "STOP") {
		// Try to extract reason
		reason := "Model rejected this action."
		if parts := strings.SplitN(response, ":", 2); len(parts) > 1 {
			reason = strings.TrimSpace(parts[1])
		} else if idx := strings.Index(upperResp, "BECAUSE"); idx != -1 {
			reason = strings.TrimSpace(response[idx+7:])
		} else {
			// Use the whole checking logic as reason if short enough
			if len(response) < 200 {
				reason = response
			}
		}
		return false, reason
	}

	return true, ""
}
