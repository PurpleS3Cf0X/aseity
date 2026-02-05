package skillsets

import (
	"fmt"
	"strings"
)

// FailurePattern represents a common failure and its correction
type FailurePattern struct {
	ToolName    string
	ErrorType   string
	Correction  string
	Example     string
	Occurrences int // Track how often this pattern occurs
}

// CommonFailures contains well-known failure patterns
var CommonFailures = []FailurePattern{
	{
		ToolName:   "bash",
		ErrorType:  "command not found",
		Correction: "Install the missing command first",
		Example:    "brew install <command>",
	},
	{
		ToolName:   "bash",
		ErrorType:  "permission denied",
		Correction: "Check file permissions or use sudo",
		Example:    "sudo <command>",
	},
	{
		ToolName:   "bash",
		ErrorType:  "no such file or directory",
		Correction: "Create the directory or check the path",
		Example:    "mkdir -p <directory>",
	},
	{
		ToolName:   "bash",
		ErrorType:  "connection refused",
		Correction: "Start the service first",
		Example:    "brew services start <service>",
	},
	{
		ToolName:   "bash",
		ErrorType:  "port already in use",
		Correction: "Stop the process using the port or use a different port",
		Example:    "lsof -ti:<port> | xargs kill",
	},
	{
		ToolName:   "file_read",
		ErrorType:  "no such file",
		Correction: "Check if the file exists and the path is correct",
		Example:    "ls -la <path>",
	},
	{
		ToolName:   "file_write",
		ErrorType:  "permission denied",
		Correction: "Check write permissions on the directory",
		Example:    "chmod +w <file>",
	},
	{
		ToolName:   "web_search",
		ErrorType:  "network error",
		Correction: "Check internet connection",
		Example:    "ping google.com",
	},
}

// CorrectiveAction represents a suggested action to fix a failure
type CorrectiveAction struct {
	Description string
	ToolCall    string // Suggested tool call to fix the issue
	Reasoning   string
}

// AnalyzeFailure analyzes a failure and suggests corrections
func AnalyzeFailure(toolName, errorMsg string, previousAttempts int) *CorrectiveAction {
	// Find matching pattern
	for i, pattern := range CommonFailures {
		if pattern.ToolName == toolName && strings.Contains(strings.ToLower(errorMsg), pattern.ErrorType) {
			// Increment occurrence counter
			CommonFailures[i].Occurrences++

			// Suggest different approach if this is a retry
			if previousAttempts > 0 {
				return &CorrectiveAction{
					Description: fmt.Sprintf("%s (Attempt %d)", pattern.Correction, previousAttempts+1),
					ToolCall:    pattern.Example,
					Reasoning:   fmt.Sprintf("Previous attempt failed with '%s'. Trying alternative approach.", pattern.ErrorType),
				}
			}

			return &CorrectiveAction{
				Description: pattern.Correction,
				ToolCall:    pattern.Example,
				Reasoning:   fmt.Sprintf("Common pattern detected: %s", pattern.ErrorType),
			}
		}
	}

	// No pattern found - generic suggestion
	return &CorrectiveAction{
		Description: "Try a different approach",
		ToolCall:    "",
		Reasoning:   "No known pattern for this error. Manual intervention may be needed.",
	}
}

// LearnFromFailure adds a new failure pattern or updates existing one
func LearnFromFailure(toolName, errorType, correction, example string) {
	// Check if pattern already exists
	for i, pattern := range CommonFailures {
		if pattern.ToolName == toolName && pattern.ErrorType == errorType {
			// Update existing pattern
			CommonFailures[i].Occurrences++
			return
		}
	}

	// Add new pattern
	CommonFailures = append(CommonFailures, FailurePattern{
		ToolName:    toolName,
		ErrorType:   errorType,
		Correction:  correction,
		Example:     example,
		Occurrences: 1,
	})
}

// GetMostCommonFailures returns the N most common failure patterns
func GetMostCommonFailures(n int) []FailurePattern {
	// Sort by occurrences (simple bubble sort for small N)
	sorted := make([]FailurePattern, len(CommonFailures))
	copy(sorted, CommonFailures)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Occurrences < sorted[j+1].Occurrences {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// ShouldRetry determines if a failure should be retried
func ShouldRetry(errorMsg string, attemptCount int) bool {
	// Don't retry more than 3 times
	if attemptCount >= 3 {
		return false
	}

	// Retry on transient errors
	transientErrors := []string{
		"connection refused",
		"timeout",
		"network error",
		"temporary failure",
		"try again",
	}

	for _, err := range transientErrors {
		if strings.Contains(strings.ToLower(errorMsg), err) {
			return true
		}
	}

	// Don't retry on permanent errors
	permanentErrors := []string{
		"not found",
		"does not exist",
		"invalid",
		"forbidden",
		"unauthorized",
	}

	for _, err := range permanentErrors {
		if strings.Contains(strings.ToLower(errorMsg), err) {
			return false
		}
	}

	// Default: retry once
	return attemptCount < 1
}

// GenerateAlternativeApproach suggests a different way to accomplish the goal
func GenerateAlternativeApproach(toolName, originalCommand, errorMsg string) string {
	switch toolName {
	case "bash":
		// If package manager fails, suggest alternative
		if strings.Contains(originalCommand, "apt-get") {
			return "Try using 'brew' on macOS or 'yum' on RedHat-based systems"
		}
		if strings.Contains(originalCommand, "brew") {
			return "Try using 'apt-get' on Debian-based systems or download manually"
		}
		if strings.Contains(originalCommand, "npm install") {
			return "Try 'yarn add' as an alternative, or check if package name is correct"
		}
		if strings.Contains(originalCommand, "pip install") {
			return "Try 'pip3 install' or 'python -m pip install'"
		}

		// If command not found, suggest installation
		if strings.Contains(errorMsg, "command not found") {
			return "Install the command first using your package manager"
		}

	case "file_read":
		if strings.Contains(errorMsg, "not found") {
			return "Check if the file exists using 'ls' or create it first"
		}

	case "web_search":
		if strings.Contains(errorMsg, "network") {
			return "Check internet connection or try again later"
		}
	}

	return "Consider breaking down the task into smaller steps"
}
