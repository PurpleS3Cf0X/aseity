package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const maxFileReadSize = 10 * 1024 * 1024 // 10MB limit

type FileReadTool struct{}

type fileReadArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (f *FileReadTool) Name() string { return "file_read" }
func (f *FileReadTool) Description() string {
	return "Read the contents of a file. Returns numbered lines. Files larger than 10MB will be rejected."
}
func (f *FileReadTool) NeedsConfirmation() bool { return false }

func (f *FileReadTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string", "description": "Absolute or relative file path"},
			"offset": map[string]any{"type": "integer", "description": "Line offset to start reading from (0-indexed)"},
			"limit":  map[string]any{"type": "integer", "description": "Max number of lines to read (default 2000)"},
		},
		"required": []string{"path"},
	}
}

func (f *FileReadTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileReadArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// Check file size before reading
	info, err := os.Stat(args.Path)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	if info.IsDir() {
		return Result{Error: fmt.Sprintf("%s is a directory, not a file", args.Path)}, nil
	}
	if info.Size() > maxFileReadSize {
		return Result{Error: fmt.Sprintf("file is too large (%d bytes, max %d). Use offset/limit to read a portion.", info.Size(), maxFileReadSize)}, nil
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

	// Default limit of 2000 lines
	limit := args.Limit
	if limit <= 0 {
		limit = 2000
	}

	end := len(lines)
	if start+limit < end {
		end = start + limit
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		line := lines[i]
		// Truncate very long lines
		if len(line) > 2000 {
			line = line[:2000] + "... [line truncated]"
		}
		fmt.Fprintf(&sb, "%4d\t%s\n", i+1, line)
	}

	if end < len(lines) {
		fmt.Fprintf(&sb, "\n... (%d more lines, use offset=%d to continue)\n", len(lines)-end, end)
	}

	return Result{Output: sb.String()}, nil
}
