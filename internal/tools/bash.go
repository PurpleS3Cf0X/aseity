package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

type BashTool struct{}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (b *BashTool) Name() string        { return "bash" }
func (b *BashTool) Description() string {
	return "Execute a bash command and return its output. Use for git, build tools, running programs, and other terminal operations."
}
func (b *BashTool) NeedsConfirmation() bool { return true }

func (b *BashTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in seconds (default 120)",
			},
		},
		"required": []string{"command"},
	}
}

func (b *BashTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args bashArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}
	timeout := 120
	if args.Timeout > 0 {
		timeout = args.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return Result{Output: output, Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}
