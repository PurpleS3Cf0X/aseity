package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	return b.ExecuteStream(ctx, rawArgs, nil)
}

func (b *BashTool) ExecuteStream(ctx context.Context, rawArgs string, callback func(string)) (Result, error) {
	var args bashArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		// Try to handle malformed arguments from some models
		args.Command = tryParseCommand(rawArgs)
		if args.Command == "" {
			return Result{Error: "invalid arguments: " + err.Error() + " (raw: " + truncateStr(rawArgs, 50) + ")"}, nil
		}
	}

	// Enforce command allowlist/blocklist
	if err := b.checkCommand(args.Command); err != nil {
		return Result{Error: err.Error()}, nil
	}

	// Enforce non-interactive sudo to prevent hanging
	args.Command = enforceNonInteractive(args.Command)

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

	var stdoutBuf, stderrBuf strings.Builder
	var mu sync.Mutex

	// Writers that capture output and optionally stream it
	outWriter := io.Writer(&stdoutBuf)
	errWriter := io.Writer(&stderrBuf)

	if callback != nil {
		// Create a writer that invokes callback for each write
		streamer := &callbackWriter{
			callback: callback,
			mu:       &mu,
		}
		outWriter = io.MultiWriter(outWriter, streamer)
		errWriter = io.MultiWriter(errWriter, streamer)
	}

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	err := cmd.Run()

	output := stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderrBuf.String()
	}
	output = strings.TrimSpace(output)

	// Smart Truncation
	// If output is too large for context window, save to file and truncate
	const MaxOutputChars = 2000
	if len(output) > MaxOutputChars {
		// Create temp file
		tmpFile, err := os.CreateTemp("", "aseity_output_*.txt")
		if err == nil {
			defer tmpFile.Close()
			if _, err := tmpFile.WriteString(output); err == nil {
				truncated := output[:MaxOutputChars]
				output = fmt.Sprintf("%s\n\n... [Output too large (%d chars). Full output saved to %s. Use 'file_read' to view it.]",
					truncated, len(output), tmpFile.Name())
			}
		}
	}

	if err != nil {
		errMsg := err.Error()
		// Check if it failed due to sudo password requirement
		if strings.Contains(output, "password is required") || strings.Contains(output, "interactive mode") {
			errMsg += " (Command failed because it required intreactive input. Try running 'sudo' manually first or run aseity as root.)"
		}
		return Result{Output: output, Error: errMsg}, nil
	}
	return Result{Output: output}, nil
}

type callbackWriter struct {
	callback func(string)
	mu       *sync.Mutex
}

func (w *callbackWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callback(string(p))
	return len(p), nil
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

// tryParseCommand attempts to extract a command from malformed arguments
// Some models send Python-style lists like ['echo', 'hello'] instead of {"command": "..."}
func tryParseCommand(rawArgs string) string {
	rawArgs = strings.TrimSpace(rawArgs)

	// Try to parse as a plain string (some models just send the command directly)
	if !strings.HasPrefix(rawArgs, "{") && !strings.HasPrefix(rawArgs, "[") {
		return rawArgs
	}

	// Try to parse Python-style list: ['echo', 'hello', 'world']
	if strings.HasPrefix(rawArgs, "[") && strings.HasSuffix(rawArgs, "]") {
		// Convert Python-style quotes to JSON-style
		jsonLike := strings.ReplaceAll(rawArgs, "'", "\"")
		var parts []string
		if json.Unmarshal([]byte(jsonLike), &parts) == nil && len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	// Try to extract "command" field even with trailing garbage
	if idx := strings.Index(rawArgs, `"command"`); idx >= 0 {
		// Find the value
		rest := rawArgs[idx+9:] // skip `"command"`
		rest = strings.TrimLeft(rest, `: "`)
		if endIdx := strings.Index(rest, `"`); endIdx > 0 {
			return rest[:endIdx]
		}
	}

	return ""
}

func enforceNonInteractive(cmd string) string {
	// If the command uses sudo, ensure it runs non-interactively (-n)
	// This prevents the process from hanging indefinitely waiting for a password.

	// Case 1: Command starts with sudo
	if strings.HasPrefix(cmd, "sudo ") && !strings.Contains(cmd, "sudo -n") {
		cmd = strings.Replace(cmd, "sudo ", "sudo -n ", 1)
	}

	// Case 2: Command contains sudo inside (e.g. "apt update && sudo apt install")
	// We recklessly replace " sudo " with " sudo -n " because sudo handles multiple -n flags fine,
	// and it's better to be safe than stuck.
	if strings.Contains(cmd, " sudo ") && !strings.Contains(cmd, " sudo -n") {
		cmd = strings.ReplaceAll(cmd, " sudo ", " sudo -n ")
	}

	return cmd
}
