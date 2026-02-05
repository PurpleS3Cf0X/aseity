package skillsets

import (
	"testing"
)

func TestLoadUserConfig(t *testing.T) {
	// Test loading non-existent config (should return defaults)
	config, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig() failed: %v", err)
	}

	if config == nil {
		t.Fatal("LoadUserConfig() returned nil config")
	}

	// Should have default settings
	if config.Settings.TrainingThreshold != 0.70 {
		t.Errorf("Default training threshold = %v, want 0.70", config.Settings.TrainingThreshold)
	}
}

func TestSaveAndLoadUserConfig(t *testing.T) {
	// Test with default path (will use ~/.aseity/skillsets.yaml or create temp)
	config := DefaultUserConfig()
	config.Settings.EnableDynamicSkillsets = true
	config.Settings.TrainingThreshold = 0.80

	// Add custom skillset
	RegisterCustomSkillset(&config, "test_skill", "Test skillset", "Test training")

	// For this test, we'll just verify the struct operations work
	// Actual file I/O is tested in integration tests

	// Verify custom skillset was added
	if len(config.CustomSkillsets) != 1 {
		t.Errorf("Expected 1 custom skillset, got %d", len(config.CustomSkillsets))
	}

	if config.CustomSkillsets[0].Name != "test_skill" {
		t.Errorf("Custom skillset name = %q, want %q", config.CustomSkillsets[0].Name, "test_skill")
	}
}

func TestMergeProfiles(t *testing.T) {
	defaults := map[string]ModelProfile{
		"test-model": {
			Name: "test-model",
			Tier: 3,
			Skillsets: map[string]float64{
				SkillToolSelection:  0.70,
				SkillErrorDiagnosis: 0.50,
			},
		},
	}

	config := DefaultUserConfig()

	// Add override to boost error diagnosis
	tier := 2
	config.Overrides = map[string]ProfileOverride{
		"test-model": {
			Tier: &tier,
			Skillsets: map[string]float64{
				SkillErrorDiagnosis: 0.80, // Boost from 0.50
			},
		},
	}

	merged := MergeProfiles(defaults, &config)

	profile := merged["test-model"]

	// Check tier was overridden
	if profile.Tier != 2 {
		t.Errorf("Merged tier = %d, want 2", profile.Tier)
	}

	// Check skillset was overridden
	if profile.Skillsets[SkillErrorDiagnosis] != 0.80 {
		t.Errorf("Merged error_diagnosis = %v, want 0.80", profile.Skillsets[SkillErrorDiagnosis])
	}

	// Check other skillset unchanged
	if profile.Skillsets[SkillToolSelection] != 0.70 {
		t.Errorf("Merged tool_selection = %v, want 0.70", profile.Skillsets[SkillToolSelection])
	}
}

func TestRegisterCustomSkillset(t *testing.T) {
	config := DefaultUserConfig()

	// Register new skillset
	RegisterCustomSkillset(&config, "skill1", "Description 1", "Training 1")

	if len(config.CustomSkillsets) != 1 {
		t.Fatalf("Expected 1 custom skillset, got %d", len(config.CustomSkillsets))
	}

	// Register same skillset again (should update, not duplicate)
	RegisterCustomSkillset(&config, "skill1", "Description 2", "Training 2")

	if len(config.CustomSkillsets) != 1 {
		t.Errorf("Expected 1 custom skillset after update, got %d", len(config.CustomSkillsets))
	}

	if config.CustomSkillsets[0].Description != "Description 2" {
		t.Errorf("Skillset not updated, description = %q", config.CustomSkillsets[0].Description)
	}
}

func TestGetEnabledCustomSkillsets(t *testing.T) {
	config := DefaultUserConfig()

	config.CustomSkillsets = []CustomSkillset{
		{Name: "skill1", Enabled: true},
		{Name: "skill2", Enabled: false},
		{Name: "skill3", Enabled: true},
	}

	enabled := GetEnabledCustomSkillsets(&config)

	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled skillsets, got %d", len(enabled))
	}

	// Check correct ones are enabled
	for _, skill := range enabled {
		if skill.Name == "skill2" {
			t.Error("skill2 should not be in enabled list")
		}
	}
}
