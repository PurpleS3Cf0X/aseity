package config

import (
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
		DefaultModel:    "llama3.2",
		Theme:           "green",
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
	return cfg, nil
}

func (c *Config) ProviderFor(name string) (ProviderConfig, bool) {
	p, ok := c.Providers[name]
	return p, ok
}
