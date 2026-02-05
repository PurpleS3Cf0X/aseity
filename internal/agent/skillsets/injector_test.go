package skillsets

import (
	"strings"
	"testing"
)

func TestInjectSkillsets(t *testing.T) {
	basePrompt := "You are an AI assistant."

	tests := []struct {
		modelName        string
		expectedContains []string
	}{
		{
			modelName: "gpt-4",
			expectedContains: []string{
				"Execution Mode",
				"highly capable",
			},
		},
		{
			modelName: "qwen2.5:14b",
			expectedContains: []string{
				"ReAct Framework",
				"<thought>",
				"Reasoning + Acting",
			},
		},
		{
			modelName: "qwen2.5:7b",
			expectedContains: []string{
				"Step-by-Step",
				"Common Patterns",
			},
		},
		{
			modelName: "qwen2.5:3b",
			expectedContains: []string{
				"Execution Templates",
				"Template 1",
				"EXACT templates",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			profile := DetectModelProfile(tt.modelName)
			enhanced := InjectSkillsets(basePrompt, profile)

			// Check base prompt is included
			if !strings.Contains(enhanced, basePrompt) {
				t.Error("Enhanced prompt should contain base prompt")
			}

			// Check expected content
			for _, expected := range tt.expectedContains {
				if !strings.Contains(enhanced, expected) {
					t.Errorf("Enhanced prompt should contain '%s'", expected)
				}
			}
		})
	}
}

func TestGetSkillTraining(t *testing.T) {
	skills := []string{
		SkillToolSelection,
		SkillParameterConstruct,
		SkillCommandConstruct,
		SkillErrorDiagnosis,
		SkillSequentialPlanning,
		SkillSelfCorrection,
	}

	for _, skill := range skills {
		t.Run(skill, func(t *testing.T) {
			training := GetSkillTraining(skill)

			// Should not be empty
			if training == "" {
				t.Error("Training content should not be empty")
			}

			// Should contain skill name
			if !strings.Contains(training, "###") {
				t.Error("Training should have header")
			}

			// Should have problem/solution structure
			if !strings.Contains(training, "Problem") && !strings.Contains(training, "Example") {
				t.Error("Training should have structured content")
			}
		})
	}
}

func TestWeakSkillsetsGetTraining(t *testing.T) {
	// Tier 4 model should get lots of training
	profile := DetectModelProfile("qwen2.5:3b")
	basePrompt := "Base prompt"
	enhanced := InjectSkillsets(basePrompt, profile)

	// Should contain training section
	if !strings.Contains(enhanced, "Skillset Training") {
		t.Error("Tier 4 model should get skillset training")
	}

	// Should contain multiple skill trainings
	trainingCount := strings.Count(enhanced, "###")
	if trainingCount < 5 {
		t.Errorf("Expected at least 5 skill trainings, got %d", trainingCount)
	}
}

func TestTier1NoExtraTraining(t *testing.T) {
	// Tier 1 model should NOT get extra training
	profile := DetectModelProfile("gpt-4")
	basePrompt := "Base prompt"
	enhanced := InjectSkillsets(basePrompt, profile)

	// Should NOT contain training section
	if strings.Contains(enhanced, "Skillset Training") {
		t.Error("Tier 1 model should not get extra skillset training")
	}
}
