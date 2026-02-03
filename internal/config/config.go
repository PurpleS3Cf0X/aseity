package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProvider string                    `yaml:"default_provider"`
	DefaultModel    string                    `yaml:"default_model"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
	Tools           ToolsConfig               `yaml:"tools"`
	Theme           string                    `yaml:"theme"`
	MaxTurns        int                       `yaml:"max_turns"`
	MaxTokens       int                       `yaml:"max_tokens"`
}

type ProviderConfig struct {
	Type    string `yaml:"type"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

type ToolsConfig struct {
	AutoApprove        []string `yaml:"auto_approve"`
	AllowedCommands    []string `yaml:"allowed_commands"`
	DisallowedCommands []string `yaml:"disallowed_commands"`
}

var envVarRe = regexp.MustCompile(`\$([A-Z_][A-Z0-9_]*)`)

func expandEnv(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		name := strings.TrimPrefix(match, "$")
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match
	})
}

func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "ollama",
		DefaultModel:    "qwen2.5:3b",
		Theme:           "green",
		MaxTurns:        50,
		MaxTokens:       100000,
		Providers: map[string]ProviderConfig{
			"ollama": {Type: "openai", BaseURL: "http://localhost:11434/v1"},
			"vllm":   {Type: "openai", BaseURL: "http://localhost:8000/v1"},
		},
		Tools: ToolsConfig{
			AutoApprove: []string{"file_read", "file_search"},
		},
	}
}

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "aseity", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "aseity", "config.yaml")
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	for name, p := range cfg.Providers {
		p.APIKey = expandEnv(p.APIKey)
		p.BaseURL = expandEnv(p.BaseURL)
		cfg.Providers[name] = p
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) ProviderFor(name string) (ProviderConfig, bool) {
	p, ok := c.Providers[name]
	return p, ok
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.DefaultProvider == "" {
		return fmt.Errorf("config: default_provider is required")
	}
	if _, ok := c.Providers[c.DefaultProvider]; !ok {
		return fmt.Errorf("config: default_provider %q not found in providers", c.DefaultProvider)
	}
	for name, p := range c.Providers {
		validTypes := map[string]bool{"openai": true, "anthropic": true, "google": true}
		if !validTypes[p.Type] {
			return fmt.Errorf("config: provider %q has invalid type %q (must be openai, anthropic, or google)", name, p.Type)
		}
		if p.Type == "openai" && p.BaseURL == "" {
			return fmt.Errorf("config: provider %q (type openai) requires base_url", name)
		}
		if p.Type == "anthropic" && p.APIKey == "" {
			return fmt.Errorf("config: provider %q (type anthropic) requires api_key", name)
		}
		if p.Type == "google" && p.APIKey == "" {
			return fmt.Errorf("config: provider %q (type google) requires api_key", name)
		}
	}
	if c.MaxTurns < 1 {
		c.MaxTurns = 50
	}
	if c.MaxTokens < 1 {
		c.MaxTokens = 100000
	}
	return nil
}
