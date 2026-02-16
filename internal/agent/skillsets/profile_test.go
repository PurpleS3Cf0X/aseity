package skillsets

import (
	"testing"
)

func TestDetectModelProfile(t *testing.T) {
	tests := []struct {
		modelName        string
		expectedTier     int
		expectedStrategy string
		expectedFC       bool
	}{
		{"gpt-4", 1, "minimal", true},
		{"gpt-4o", 1, "minimal", true},
		{"gpt-4-turbo", 1, "minimal", true}, // Fuzzy match
		{"claude-3.5-sonnet", 1, "minimal", true},
		{"qwen2.5:14b", 2, "react", false},
		{"deepseek-r1:14b", 2, "react", false},
		{"qwen2.5-coder:14b", 2, "react", false}, // New Tier 2
		{"qwen2.5-coder:7b", 3, "guided", false}, // New Tier 3 (JSON Fallback)
		{"qwen2.5:7b", 3, "guided", false},
		{"qwen2.5:3b", 4, "template", false},
		{"unknown-model", 3, "guided", false}, // Default to Tier 3
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			profile := DetectModelProfile(tt.modelName)
			if profile.Tier != tt.expectedTier {
				t.Errorf("Expected tier %d, got %d", tt.expectedTier, profile.Tier)
			}
			if profile.PromptStrategy != tt.expectedStrategy {
				t.Errorf("Expected strategy %q, got %q", tt.expectedStrategy, profile.PromptStrategy)
			}
			if profile.SupportsNativeFC != tt.expectedFC {
				t.Errorf("Expected native FC %v, got %v", tt.expectedFC, profile.SupportsNativeFC)
			}
		})
	}
}

func TestGetWeakSkillsets(t *testing.T) {
	profile := DetectModelProfile("qwen2.5:14b")
	weak := profile.GetWeakSkillsets(0.80)

	// qwen2.5:14b should have several skills below 80%
	if len(weak) == 0 {
		t.Error("Expected some weak skillsets for qwen2.5:14b")
	}

	// Check that weak skillsets are actually below threshold
	for _, skill := range weak {
		if profile.Skillsets[skill] >= 0.80 {
			t.Errorf("Skillset %s has proficiency %.2f, should be < 0.80", skill, profile.Skillsets[skill])
		}
	}
}

func TestNeedsTraining(t *testing.T) {
	profile := DetectModelProfile("qwen2.5:3b")

	// Tier 4 model should need training for most skills
	needsTraining := 0
	for _, skill := range AllSkillsets() {
		if profile.NeedsTraining(skill) {
			needsTraining++
		}
	}

	if needsTraining < 8 {
		t.Errorf("Expected qwen2.5:3b to need training for at least 8 skills, got %d", needsTraining)
	}
}

func TestTierValidation(t *testing.T) {
	profiles := DefaultProfiles()

	for name, profile := range profiles {
		// Validate tier is 1-4
		if profile.Tier < 1 || profile.Tier > 4 {
			t.Errorf("Profile %s has invalid tier %d", name, profile.Tier)
		}

		// Validate all skillsets are present
		for _, skill := range AllSkillsets() {
			if _, ok := profile.Skillsets[skill]; !ok {
				t.Errorf("Profile %s missing skillset %s", name, skill)
			}
		}

		// Validate proficiency values are 0-1
		for skill, prof := range profile.Skillsets {
			if prof < 0 || prof > 1 {
				t.Errorf("Profile %s has invalid proficiency %.2f for %s", name, prof, skill)
			}
		}
	}
}
