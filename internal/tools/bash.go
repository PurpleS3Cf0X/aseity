package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BashTool struct {
	AllowedCommands    []string
	DisallowedCommands []string
}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (b *BashTool) Name() string { return "bash" }
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

	// Enforce command allowlist/blocklist
	if err := b.checkCommand(args.Command); err != nil {
		return Result{Error: err.Error()}, nil
	}

	timeout := 120
	if args.Timeout > 0 {
		timeout = args.Timeout
	}
	if timeout > 600 {
		timeout = 600 // hard cap at 10 minutes
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	// Truncate very large outputs
	if len(output) > 50000 {
		output = output[:50000] + "\n... [output truncated at 50000 chars]"
	}

	if err != nil {
		return Result{Output: output, Error: err.Error()}, nil
	}
	return Result{Output: output}, nil
}

func (b *BashTool) checkCommand(cmd string) error {
	// Extract the base command (first word, ignoring env vars and pipes)
	baseCmd := extractBaseCommand(cmd)

	// Check blocklist first
	for _, blocked := range b.DisallowedCommands {
		if baseCmd == blocked || strings.Contains(cmd, blocked) {
			return fmt.Errorf("command %q is blocked by configuration", blocked)
		}
	}

	// Dangerous commands that are always blocked
	dangerous := []string{"rm -rf /", "mkfs", "dd if=", ":(){:|:&};:"}
	for _, d := range dangerous {
		if strings.Contains(cmd, d) {
			return fmt.Errorf("potentially destructive command blocked: contains %q", d)
		}
	}

	// If allowlist is set, only allow those commands
	if len(b.AllowedCommands) > 0 {
		for _, allowed := range b.AllowedCommands {
			if baseCmd == allowed {
				return nil
			}
		}
		return fmt.Errorf("command %q is not in the allowed commands list", baseCmd)
	}

	return nil
}

func extractBaseCommand(cmd string) string {
	// Skip env var assignments (FOO=bar cmd)
	parts := strings.Fields(cmd)
	for _, p := range parts {
		if !strings.Contains(p, "=") {
			// Remove path prefix
			if idx := strings.LastIndex(p, "/"); idx >= 0 {
				p = p[idx+1:]
			}
			return p
		}
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return cmd
}
