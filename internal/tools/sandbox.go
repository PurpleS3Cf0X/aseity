package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type SandboxRunTool struct{}

func NewSandboxRunTool() *SandboxRunTool {
	return &SandboxRunTool{}
}

func (s *SandboxRunTool) Name() string            { return "sandbox_run" }
func (s *SandboxRunTool) NeedsConfirmation() bool { return true }
func (s *SandboxRunTool) Description() string {
	return "Execute a command securely inside a disposable Docker container. Use this for untrusted scripts or to avoid polluting the host OS."
}

func (s *SandboxRunTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"image": map[string]any{
				"type":        "string",
				"description": "The Docker image to use (default: python:3.9-slim).",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to run inside the container.",
			},
			"network": map[string]any{
				"type":        "boolean",
				"description": "Allow network access (default: true). Set false for air-gapped execution.",
			},
		},
		"required": []string{"command"},
	}
}

type sandboxArgs struct {
	Image   string `json:"image"`
	Command string `json:"command"`
	Network bool   `json:"network,omitempty"`
}

func (s *SandboxRunTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	return s.runDocker(ctx, rawArgs, nil)
}

func (s *SandboxRunTool) ExecuteStream(ctx context.Context, rawArgs string, callback func(string)) (Result, error) {
	return s.runDocker(ctx, rawArgs, callback)
}

func (s *SandboxRunTool) runDocker(ctx context.Context, rawArgs string, callback func(string)) (Result, error) {
	var args sandboxArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if args.Image == "" {
		args.Image = "python:3.9-slim"
	}

	// Check for Docker
	dockerCmd := "docker"
	if _, err := exec.LookPath("docker"); err != nil {
		fallback := "/Applications/Docker.app/Contents/Resources/bin/docker"
		if _, err := os.Stat(fallback); err == nil {
			dockerCmd = fallback
		} else {
			return Result{Error: "docker command not found. Please install Docker to use sandbox_run."}, nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return Result{Error: "failed to get current working directory: " + err.Error()}, nil
	}

	dockerArgs := []string{"run", "--rm"}
	dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/workspace", cwd))
	dockerArgs = append(dockerArgs, "-w", "/workspace")

	if !args.Network {
		dockerArgs = append(dockerArgs, "--network", "none")
	}

	dockerArgs = append(dockerArgs, args.Image)
	dockerArgs = append(dockerArgs, "sh", "-c", args.Command)

	cmd := exec.CommandContext(ctx, dockerCmd, dockerArgs...)

	if callback != nil {
		// Streaming mode
		// We'll combine stdout and stderr
		cmd.Stderr = cmd.Stdout
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return Result{Error: fmt.Sprintf("failed to create stdout pipe: %v", err)}, nil
		}

		if err := cmd.Start(); err != nil {
			return Result{Error: fmt.Sprintf("failed to start command: %v", err)}, nil
		}

		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				callback(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}

		if err := cmd.Wait(); err != nil {
			return Result{Error: fmt.Sprintf("command execution failed: %v", err)}, nil
		}
		return Result{Output: "Sandbox execution completed successfully."}, nil

	} else {
		// Non-streaming mode
		out, err := cmd.CombinedOutput()
		output := string(out)

		if err != nil {
			return Result{
				Output: output,
				Error:  fmt.Sprintf("sandbox execution failed: %v", err),
			}, nil
		}
		return Result{Output: output}, nil
	}
}
