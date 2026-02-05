package skillsets

import (
	"testing"
)

func TestAnalyzeFailure(t *testing.T) {
	tests := []struct {
		name             string
		toolName         string
		errorMsg         string
		previousAttempts int
		wantCorrection   string
	}{
		{
			name:             "command not found - first attempt",
			toolName:         "bash",
			errorMsg:         "redis-server: command not found",
			previousAttempts: 0,
			wantCorrection:   "Install the missing command first",
		},
		{
			name:             "command not found - retry",
			toolName:         "bash",
			errorMsg:         "redis-server: command not found",
			previousAttempts: 1,
			wantCorrection:   "Install the missing command first (Attempt 2)",
		},
		{
			name:             "permission denied",
			toolName:         "bash",
			errorMsg:         "permission denied",
			previousAttempts: 0,
			wantCorrection:   "Check file permissions or use sudo",
		},
		{
			name:             "connection refused",
			toolName:         "bash",
			errorMsg:         "connection refused",
			previousAttempts: 0,
			wantCorrection:   "Start the service first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := AnalyzeFailure(tt.toolName, tt.errorMsg, tt.previousAttempts)
			if action.Description != tt.wantCorrection {
				t.Errorf("AnalyzeFailure() correction = %v, want %v", action.Description, tt.wantCorrection)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name         string
		errorMsg     string
		attemptCount int
		want         bool
	}{
		{"transient error - first attempt", "connection refused", 0, true},
		{"transient error - second attempt", "connection refused", 1, true},
		{"transient error - too many attempts", "connection refused", 3, false},
		{"permanent error - not found", "file not found", 0, false},
		{"permanent error - invalid", "invalid syntax", 0, false},
		{"unknown error - first attempt", "some random error", 0, true},
		{"unknown error - second attempt", "some random error", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRetry(tt.errorMsg, tt.attemptCount)
			if got != tt.want {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAlternativeApproach(t *testing.T) {
	tests := []struct {
		name            string
		toolName        string
		originalCommand string
		errorMsg        string
		wantContains    string
	}{
		{
			name:            "apt-get alternative",
			toolName:        "bash",
			originalCommand: "apt-get install redis",
			errorMsg:        "command not found",
			wantContains:    "brew",
		},
		{
			name:            "npm alternative",
			toolName:        "bash",
			originalCommand: "npm install express",
			errorMsg:        "command not found",
			wantContains:    "yarn",
		},
		{
			name:            "pip alternative",
			toolName:        "bash",
			originalCommand: "pip install numpy",
			errorMsg:        "command not found",
			wantContains:    "pip3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateAlternativeApproach(tt.toolName, tt.originalCommand, tt.errorMsg)
			if !contains(got, tt.wantContains) {
				t.Errorf("GenerateAlternativeApproach() = %v, want to contain %v", got, tt.wantContains)
			}
		})
	}
}

func TestLearnFromFailure(t *testing.T) {
	// Reset CommonFailures
	originalLen := len(CommonFailures)

	// Learn a new pattern
	LearnFromFailure("custom_tool", "custom error", "custom correction", "custom example")

	if len(CommonFailures) != originalLen+1 {
		t.Errorf("LearnFromFailure() should add new pattern, got %d patterns, want %d", len(CommonFailures), originalLen+1)
	}

	// Learn same pattern again (should increment occurrence)
	LearnFromFailure("custom_tool", "custom error", "custom correction", "custom example")

	if len(CommonFailures) != originalLen+1 {
		t.Errorf("LearnFromFailure() should not add duplicate pattern, got %d patterns, want %d", len(CommonFailures), originalLen+1)
	}

	// Check occurrence count
	for _, pattern := range CommonFailures {
		if pattern.ToolName == "custom_tool" && pattern.ErrorType == "custom error" {
			if pattern.Occurrences != 2 {
				t.Errorf("LearnFromFailure() occurrence count = %d, want 2", pattern.Occurrences)
			}
		}
	}
}

func TestGetMostCommonFailures(t *testing.T) {
	// Reset and add test patterns
	CommonFailures = []FailurePattern{
		{ToolName: "tool1", ErrorType: "error1", Occurrences: 5},
		{ToolName: "tool2", ErrorType: "error2", Occurrences: 10},
		{ToolName: "tool3", ErrorType: "error3", Occurrences: 3},
	}

	top2 := GetMostCommonFailures(2)

	if len(top2) != 2 {
		t.Errorf("GetMostCommonFailures(2) returned %d patterns, want 2", len(top2))
	}

	if top2[0].Occurrences != 10 {
		t.Errorf("Most common failure should have 10 occurrences, got %d", top2[0].Occurrences)
	}

	if top2[1].Occurrences != 5 {
		t.Errorf("Second most common failure should have 5 occurrences, got %d", top2[1].Occurrences)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
