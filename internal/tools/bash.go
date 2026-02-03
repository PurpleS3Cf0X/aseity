package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
)

type BashTool struct {
	AllowedCommands    []string
	DisallowedCommands []string
	inputCh            chan string
	reqInputFn         func()
}

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (b *BashTool) Name() string { return "bash" }
func (b *BashTool) Description() string {
	return "Execute a bash command. Supports interactive prompts (like sudo passwords). Output is streamed."
}
func (b *BashTool) NeedsConfirmation() bool { return true }

func (b *BashTool) SetInputChan(ch chan string) {
	b.inputCh = ch
}

func (b *BashTool) SetInputRequestCallback(fn func()) {
	b.reqInputFn = fn
}

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
		args.Command = tryParseCommand(rawArgs)
		if args.Command == "" {
			return Result{Error: "invalid arguments: " + err.Error()}, nil
		}
	}

	if err := b.checkCommand(args.Command); err != nil {
		return Result{Error: err.Error()}, nil
	}

	// NOTE: We do NOT enforce "sudo -n" anymore because we support interactivity via PTY

	timeout := 120
	if args.Timeout > 0 {
		timeout = args.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Use PTY to execute
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return Result{Error: "failed to start pty: " + err.Error()}, nil
	}
	defer func() { _ = ptmx.Close() }()

	var outputBuf strings.Builder
	var buf = make([]byte, 1024)

	// Output loop
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			outputBuf.WriteString(chunk)
			if callback != nil {
				callback(chunk)

				// Heuristic: Check for prompts
				// If output ends with "word:" or "? " or "[y/n]", ask for input
				trimmed := strings.TrimSpace(chunk)
				if isPrompt(trimmed) {
					// Request input if we have a channel
					if b.reqInputFn != nil && b.inputCh != nil {
						b.reqInputFn() // Notify Agent -> TUI

						// Block waiting for user input
						select {
						case input := <-b.inputCh:
							// Send input to PTY
							input = strings.TrimSpace(input) + "\n"
							if _, err := ptmx.Write([]byte(input)); err != nil {
								// Ignore write error
							}
						case <-ctx.Done():
							return Result{Output: outputBuf.String(), Error: "timeout awaiting input"}, nil
						}
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			// Linux/Mac PTY returns EIO on close (success)
			if strings.Contains(err.Error(), "input/output error") {
				break
			}
			return Result{Output: outputBuf.String(), Error: err.Error()}, nil
		}
	}

	return Result{Output: outputBuf.String()}, nil
}

func isPrompt(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, ":") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "]") || strings.HasSuffix(s, "$") || strings.HasSuffix(s, "#") {
		// Ignore common shell prompts if we want, but for sudo/password/confirm, they usually look like prompts.
		// "Password:"
		// "Do you want to continue? [Y/n]"
		// "Enter value:"
		return true
	}
	return false
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
