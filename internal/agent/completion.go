package agent

import (
	"fmt"
	"strings"
)

// CompletionChecker verifies that tasks are fully completed before finishing
type CompletionChecker struct {
	originalGoal string
	toolsCalled  []string
}

// NewCompletionChecker creates a new completion checker
func NewCompletionChecker(goal string) *CompletionChecker {
	return &CompletionChecker{
		originalGoal: goal,
		toolsCalled:  make([]string, 0),
	}
}

// RecordToolCall tracks which tools have been called
func (c *CompletionChecker) RecordToolCall(toolName string) {
	c.toolsCalled = append(c.toolsCalled, toolName)
}

// BuildCompletionPrompt creates a prompt to verify task completion
func (c *CompletionChecker) BuildCompletionPrompt() string {
	toolsList := "None"
	if len(c.toolsCalled) > 0 {
		toolsList = strings.Join(c.toolsCalled, ", ")
	}

	return fmt.Sprintf(`
⚠️ BEFORE YOU FINISH - COMPLETION VERIFICATION:

Original user request: "%s"

Tools you called: %s

CRITICAL CHECKLIST - Answer each question:
☐ Did you call ALL necessary tools to complete this request?
☐ Did you READ and PROCESS the results from each tool?
☐ Did you USE the actual data from tool results in your response?
☐ Did you provide the SPECIFIC information the user asked for?
☐ Did you complete ALL steps mentioned in the request?
☐ Is your response based on REAL data (not generic/hallucinated)?

If ANY checkbox is unchecked, you MUST continue working. Do NOT finish prematurely.

If ALL checkboxes are checked, you may finish your response now.`, c.originalGoal, toolsList)
}

// ShouldPromptCompletion determines if we should inject a completion check
func (c *CompletionChecker) ShouldPromptCompletion(responseLength int) bool {
	// Prompt completion check if:
	// 1. At least one tool was called (so we have results to verify)
	// 2. Response is not too short (< 100 chars suggests incomplete work)
	return len(c.toolsCalled) > 0 && responseLength > 100
}
