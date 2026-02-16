package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunScriptTool writes a script to a temporary file and executes it.
type RunScriptTool struct{}

func NewRunScriptTool() *RunScriptTool {
	return &RunScriptTool{}
}

func (t *RunScriptTool) Name() string            { return "run_script" }
func (t *RunScriptTool) NeedsConfirmation() bool { return true }
func (t *RunScriptTool) Description() string {
	return "Write and execute a script (bash, python, node) in one step. Useful for complex logic or loops."
}

func (t *RunScriptTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"language": map[string]any{
				"type":        "string",
				"description": "Language to run: 'bash', 'python', 'node', 'go'",
				"enum":        []string{"bash", "python", "node", "go"},
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The script content/code to execute.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "Optional filename (e.g., 'test_script.py'). If not provided, a temp file is used.",
			},
		},
		"required": []string{"language", "content"},
	}
}

func (t *RunScriptTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args struct {
		Language string `json:"language"`
		Content  string `json:"content"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// Determine file extension and runner
	var ext, runner string
	var runnerArgs []string

	switch args.Language {
	case "bash":
		ext = ".sh"
		runner = "bash"
	case "python":
		ext = ".py"
		runner = "python3"
	case "node":
		ext = ".js"
		runner = "node"
	case "go":
		ext = ".go"
		runner = "go"
		runnerArgs = []string{"run"}
	default:
		return Result{Error: "unsupported language: " + args.Language}, nil
	}

	// Create file
	cwd, _ := os.Getwd()
	filename := args.Name
	if filename == "" {
		filename = fmt.Sprintf("temp_script_%d%s", 0, ext) // Simple temp name, maybe timestamp would be better but this is fine
		// Let's use os.CreateTemp principle but in CWD for visibility
		f, err := os.CreateTemp(cwd, "script_*"+ext)
		if err != nil {
			return Result{Error: "failed to create temp file: " + err.Error()}, nil
		}
		filename = f.Name()
		defer os.Remove(filename) /// Cleanup temp files? Or keep for debugging? Claude Code keeps them usually.
		// Let's keep them if user provided logic, but if implicit, maybe delete.
		// Actually, keeping them clutters the workspace. The prompt says "Write and execute".
		// Let's auto-remove only if name wasn't provided.
		f.Close()
	} else {
		// If user provided name, we assume they want to keep it or it matters.
		filename = filepath.Join(cwd, filename)
	}

	// Write content
	if err := os.WriteFile(filename, []byte(args.Content), 0755); err != nil {
		return Result{Error: "failed to write script: " + err.Error()}, nil
	}

	// Prepare command
	cmdArgs := append(runnerArgs, filename)
	cmd := exec.CommandContext(ctx, runner, cmdArgs...)
	cmd.Dir = cwd

	output, err := cmd.CombinedOutput()
	outStr := string(output)

	// Clean up if temp
	if args.Name == "" {
		_ = os.Remove(filename)
	}

	if err != nil {
		return Result{
			Output: outStr,
			Error:  fmt.Sprintf("Script failed with %v. Output:\n%s", err, outStr),
		}, nil
	}

	return Result{Output: outStr}, nil
}
