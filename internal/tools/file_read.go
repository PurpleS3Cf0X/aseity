package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type FileReadTool struct{}

type fileReadArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (f *FileReadTool) Name() string        { return "file_read" }
func (f *FileReadTool) Description() string {
	return "Read the contents of a file. Returns numbered lines."
}
func (f *FileReadTool) NeedsConfirmation() bool { return false }

func (f *FileReadTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string", "description": "Absolute or relative file path"},
			"offset": map[string]any{"type": "integer", "description": "Line offset to start reading from (0-indexed)"},
			"limit":  map[string]any{"type": "integer", "description": "Max number of lines to read"},
		},
		"required": []string{"path"},
	}
}

func (f *FileReadTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileReadArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	lines := strings.Split(string(data), "\n")
	start := args.Offset
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if args.Limit > 0 && start+args.Limit < end {
		end = start + args.Limit
	}
	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%4d\t%s\n", i+1, lines[i])
	}
	return Result{Output: sb.String()}, nil
}
