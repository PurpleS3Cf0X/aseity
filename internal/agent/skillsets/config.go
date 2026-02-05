package skillsets

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents user's custom skillset configuration
type UserConfig struct {
	Models          map[string]ModelProfile    `yaml:"models"`
	Overrides       map[string]ProfileOverride `yaml:"overrides"`
	Settings        GlobalSettings             `yaml:"settings"`
	CustomSkillsets []CustomSkillset           `yaml:"custom_skillsets"`
}

// ProfileOverride allows overriding specific fields of a model profile
type ProfileOverride struct {
	Tier            *int               `yaml:"tier,omitempty"`
	PromptStrategy  *string            `yaml:"prompt_strategy,omitempty"`
	ValidationLevel *ValidationLevel   `yaml:"validation_level,omitempty"`
	Skillsets       map[string]float64 `yaml:"skillsets,omitempty"`
}

// GlobalSettings contains global configuration
type GlobalSettings struct {
	DefaultValidationLevel ValidationLevel `yaml:"default_validation_level"`
	EnableAutoCorrection   bool            `yaml:"enable_auto_correction"`
	TrainingThreshold      float64         `yaml:"skillset_training_threshold"`
	EnableDynamicSkillsets bool            `yaml:"enable_dynamic_skillsets"`
}

// CustomSkillset represents a user-defined skillset
type CustomSkillset struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Training    string `yaml:"training"`
	Enabled     bool   `yaml:"enabled"`
}

// DefaultUserConfig returns default configuration
func DefaultUserConfig() UserConfig {
	return UserConfig{
		Models:    make(map[string]ModelProfile),
		Overrides: make(map[string]ProfileOverride),
		Settings: GlobalSettings{
			DefaultValidationLevel: ValidationMedium,
			EnableAutoCorrection:   true,
			TrainingThreshold:      0.70,
			EnableDynamicSkillsets: true,
		},
		CustomSkillsets: []CustomSkillset{},
	}
}

// LoadUserConfig loads configuration from ~/.aseity/skillsets.yaml
func LoadUserConfig() (*UserConfig, error) {
	configPath := GetConfigPath()

	// If config doesn't exist, return default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultUserConfig()
		return &config, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse YAML
	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveUserConfig saves configuration to ~/.aseity/skillsets.yaml
func SaveUserConfig(config *UserConfig) error {
	configPath := GetConfigPath()

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".aseity/skillsets.yaml"
	}
	return filepath.Join(home, ".aseity", "skillsets.yaml")
}

// MergeProfiles merges default profiles with user customizations
func MergeProfiles(defaults map[string]ModelProfile, config *UserConfig) map[string]ModelProfile {
	merged := make(map[string]ModelProfile)

	// Start with defaults
	for name, profile := range defaults {
		merged[name] = profile
	}

	// Add custom models
	for name, profile := range config.Models {
		merged[name] = profile
	}

	// Apply overrides
	for name, override := range config.Overrides {
		if profile, ok := merged[name]; ok {
			profile = applyOverride(profile, override)
			merged[name] = profile
		}
	}

	return merged
}

// applyOverride applies an override to a profile
func applyOverride(profile ModelProfile, override ProfileOverride) ModelProfile {
	if override.Tier != nil {
		profile.Tier = *override.Tier
	}
	if override.PromptStrategy != nil {
		profile.PromptStrategy = *override.PromptStrategy
	}
	if override.ValidationLevel != nil {
		profile.ValidationLevel = *override.ValidationLevel
	}
	if override.Skillsets != nil {
		for skill, proficiency := range override.Skillsets {
			profile.Skillsets[skill] = proficiency
		}
	}
	return profile
}

// RegisterCustomSkillset adds a custom skillset to the configuration
func RegisterCustomSkillset(config *UserConfig, name, description, training string) {
	skillset := CustomSkillset{
		Name:        name,
		Description: description,
		Training:    training,
		Enabled:     true,
	}

	// Check if already exists
	for i, existing := range config.CustomSkillsets {
		if existing.Name == name {
			config.CustomSkillsets[i] = skillset
			return
		}
	}

	// Add new
	config.CustomSkillsets = append(config.CustomSkillsets, skillset)
}

// GetEnabledCustomSkillsets returns all enabled custom skillsets
func GetEnabledCustomSkillsets(config *UserConfig) []CustomSkillset {
	enabled := []CustomSkillset{}
	for _, skillset := range config.CustomSkillsets {
		if skillset.Enabled {
			enabled = append(enabled, skillset)
		}
	}
	return enabled
}
