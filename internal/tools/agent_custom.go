package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jeanpaul/aseity/internal/config"
)

type CreateAgentTool struct{}

func NewCreateAgentTool() *CreateAgentTool {
	return &CreateAgentTool{}
}

func (c *CreateAgentTool) Name() string            { return "create_agent" }
func (c *CreateAgentTool) NeedsConfirmation() bool { return false } // Creating a file is relatively safe, but could prompt given it's a persistent change? Let's say false for ease.
func (c *CreateAgentTool) Description() string {
	return "Create a new custom agent persona. Saves the configuration so it can be spawned later using 'spawn_agent' with 'agent_name'."
}

func (c *CreateAgentTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Unique name for the agent (e.g. 'researcher', 'writer'). No spaces.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short description of what this agent does.",
			},
			"system_prompt": map[string]any{
				"type":        "string",
				"description": "The system instructions that define the agent's behavior, persona, and constraints.",
			},
		},
		"required": []string{"name", "system_prompt"},
	}
}

type createAgentArgs struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
}

func (c *CreateAgentTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args createAgentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if args.Name == "" || args.SystemPrompt == "" {
		return Result{Error: "name and system_prompt are required"}, nil
	}

	cfg := config.AgentConfig{
		Name:        args.Name,
		Description: args.Description,
		Prompt:      args.SystemPrompt,
	}

	if err := config.SaveAgentConfig(cfg); err != nil {
		return Result{Error: fmt.Sprintf("failed to save agent config: %v", err)}, nil
	}

	return Result{Output: fmt.Sprintf("Successfully created agent '%s'. You can now spawn it by asking to 'spawn agent %s'.", args.Name, args.Name)}, nil
}

type DeleteAgentTool struct{}

func NewDeleteAgentTool() *DeleteAgentTool {
	return &DeleteAgentTool{}
}

func (d *DeleteAgentTool) Name() string            { return "delete_agent" }
func (d *DeleteAgentTool) NeedsConfirmation() bool { return true }
func (d *DeleteAgentTool) Description() string {
	return "Delete a custom agent persona configuration file."
}

func (d *DeleteAgentTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name of the agent to delete.",
			},
		},
		"required": []string{"name"},
	}
}

type deleteAgentArgs struct {
	Name string `json:"name"`
}

func (d *DeleteAgentTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args deleteAgentArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if args.Name == "" {
		return Result{Error: "name is required"}, nil
	}

	if err := config.DeleteAgentConfig(args.Name); err != nil {
		return Result{Error: fmt.Sprintf("failed to delete agent: %v", err)}, nil
	}

	return Result{Output: fmt.Sprintf("Successfully deleted agent '%s'.", args.Name)}, nil
}
