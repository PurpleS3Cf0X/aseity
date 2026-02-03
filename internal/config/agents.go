package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Prompt      string   `yaml:"system_prompt"`
	Tools       []string `yaml:"allowed_tools,omitempty"` // For future use
}

func GetAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "aseity", "agents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func SaveAgentConfig(cfg AgentConfig) error {
	dir, err := GetAgentsDir()
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, cfg.Name+".yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func LoadAgentConfig(name string) (*AgentConfig, error) {
	dir, err := GetAgentsDir()
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(dir, name+".yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("agent '%s' not found", name)
		}
		return nil, err
	}

	var cfg AgentConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ListAgents() ([]string, error) {
	dir, err := GetAgentsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

func DeleteAgentConfig(name string) error {
	dir, err := GetAgentsDir()
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, name+".yaml")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("agent '%s' not found", name)
	}

	return os.Remove(filename)
}
