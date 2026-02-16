package config

import (
	"testing"
)

// TestDefaultModel verifies qwen2.5:14b is the default
func TestDefaultModel(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != "glm4" {
		t.Errorf("Default model = %q, want %q", cfg.DefaultModel, "glm4")
	}

	t.Logf("✅ Default model is %s", cfg.DefaultModel)
}

// TestDefaultProvider verifies ollama is the default
func TestDefaultProvider(t *testing.T) {
	cfg := DefaultConfig()
	expected := "ollama"

	if cfg.DefaultProvider != expected {
		t.Errorf("Default provider = %q, want %q", cfg.DefaultProvider, expected)
	}

	t.Logf("✅ Default provider is %s", cfg.DefaultProvider)
}
