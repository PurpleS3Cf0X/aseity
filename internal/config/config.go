package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DefaultProvider string                    `yaml:"default_provider" mapstructure:"default_provider"`
	DefaultModel    string                    `yaml:"default_model" mapstructure:"default_model"`
	Providers       map[string]ProviderConfig `yaml:"providers" mapstructure:"providers"`
	Tools           ToolsConfig               `yaml:"tools" mapstructure:"tools"`
	Orchestrator    OrchestratorConfig        `yaml:"orchestrator" mapstructure:"orchestrator"`
	Theme           string                    `yaml:"theme" mapstructure:"theme"`
	MaxTurns        int                       `yaml:"max_turns" mapstructure:"max_turns"`
	MaxTokens       int                       `yaml:"max_tokens" mapstructure:"max_tokens"`
}

type OrchestratorConfig struct {
	Enabled      bool `yaml:"enabled" mapstructure:"enabled"`
	AutoDetect   bool `yaml:"auto_detect" mapstructure:"auto_detect"`
	Parallel     bool `yaml:"parallel" mapstructure:"parallel"`
	MaxRetries   int  `yaml:"max_retries" mapstructure:"max_retries"`
	MaxSteps     int  `yaml:"max_steps" mapstructure:"max_steps"`
	ShowProgress bool `yaml:"show_progress" mapstructure:"show_progress"`
}

type ProviderConfig struct {
	Type    string `yaml:"type" mapstructure:"type"`
	BaseURL string `yaml:"base_url" mapstructure:"base_url"`
	APIKey  string `yaml:"api_key" mapstructure:"api_key"`
	Model   string `yaml:"model" mapstructure:"model"`
}

type ToolsConfig struct {
	AutoApprove        []string `yaml:"auto_approve" mapstructure:"auto_approve"`
	AllowedCommands    []string `yaml:"allowed_commands" mapstructure:"allowed_commands"`
	DisallowedCommands []string `yaml:"disallowed_commands" mapstructure:"disallowed_commands"`
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
		DefaultModel:    "qwen2.5-coder:7b",
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

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Search paths
	viper.AddConfigPath(".")
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		viper.AddConfigPath(filepath.Join(xdg, "aseity"))
	}
	home, _ := os.UserHomeDir()
	viper.AddConfigPath(filepath.Join(home, ".config", "aseity"))

	// Environment variables
	viper.SetEnvPrefix("ASEITY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error produced
			return nil, err
		}
		// Config file not found; ignore and use defaults
	}

	// Unmarshal
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	// Manual expansion for keys in config file
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
