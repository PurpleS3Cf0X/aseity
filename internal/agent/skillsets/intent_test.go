package skillsets

import (
	"testing"
)

func TestDetectIntent(t *testing.T) {
	tests := []struct {
		name     string
		userMsg  string
		expected Intent
	}{
		{
			name:     "install package",
			userMsg:  "install redis",
			expected: IntentInstall,
		},
		{
			name:     "npm install",
			userMsg:  "npm install express",
			expected: IntentInstall,
		},
		{
			name:     "code review",
			userMsg:  "review this code for bugs",
			expected: IntentCodeReview,
		},
		{
			name:     "deploy",
			userMsg:  "deploy to production",
			expected: IntentDeploy,
		},
		{
			name:     "debug error",
			userMsg:  "fix this error",
			expected: IntentDebug,
		},
		{
			name:     "search",
			userMsg:  "search for documentation",
			expected: IntentSearch,
		},
		{
			name:     "read file",
			userMsg:  "read the config file",
			expected: IntentFileOps,
		},
		{
			name:     "run tests",
			userMsg:  "run unit tests",
			expected: IntentTest,
		},
		{
			name:     "optimize",
			userMsg:  "optimize this query",
			expected: IntentOptimize,
		},
		{
			name:     "security",
			userMsg:  "check for security vulnerabilities",
			expected: IntentSecurity,
		},
		{
			name:     "general",
			userMsg:  "hello",
			expected: IntentGeneral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIntent(tt.userMsg)
			if got != tt.expected {
				t.Errorf("DetectIntent(%q) = %v, want %v", tt.userMsg, IntentName(got), IntentName(tt.expected))
			}
		})
	}
}

func TestGetSkillsetsForIntent(t *testing.T) {
	tests := []struct {
		name     string
		intent   Intent
		contains string
	}{
		{
			name:     "install intent includes command construction",
			intent:   IntentInstall,
			contains: SkillCommandConstruct,
		},
		{
			name:     "debug intent includes error diagnosis",
			intent:   IntentDebug,
			contains: SkillErrorDiagnosis,
		},
		{
			name:     "deploy intent includes sequential planning",
			intent:   IntentDeploy,
			contains: SkillSequentialPlanning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skills := GetSkillsetsForIntent(tt.intent)
			found := false
			for _, skill := range skills {
				if skill == tt.contains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetSkillsetsForIntent(%v) does not contain %q", IntentName(tt.intent), tt.contains)
			}
		})
	}
}

func TestBuildContextualPrompt(t *testing.T) {
	// Create a test profile with weak error diagnosis
	profile := ModelProfile{
		Name: "test-model",
		Tier: 3,
		Skillsets: map[string]float64{
			SkillErrorDiagnosis: 0.50, // Weak
			SkillToolSelection:  0.90, // Strong
		},
	}

	// Test debug intent (should include error diagnosis training)
	prompt := BuildContextualPrompt(IntentDebug, profile)

	if prompt == "" {
		t.Error("BuildContextualPrompt returned empty string")
	}

	// Should include error diagnosis training since model is weak
	if !containsSubstring(prompt, "Error Diagnosis") {
		t.Error("Expected prompt to include Error Diagnosis training")
	}

	// Should NOT include tool selection training since model is strong
	if containsSubstring(prompt, "Tool Selection") {
		t.Error("Did not expect prompt to include Tool Selection training")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
