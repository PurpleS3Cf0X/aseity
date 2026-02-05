package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	return "Read the contents of a file. Supports Text, PDF, and Excel (.xlsx). Returns numbered lines for text/PDF, and Markdown tables for Excel."
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

	// Check file extension for specialized parsing
	ext := strings.ToLower(filepath.Ext(args.Path))
	var content string
	var parseErr error

	switch ext {
	case ".pdf":
		content, parseErr = parsePDF(args.Path)
	case ".xlsx", ".xlsm":
		content, parseErr = parseExcel(args.Path)
	default:
		// Default text reading
		data, err := os.ReadFile(args.Path)
		if err != nil {
			return Result{Error: err.Error()}, nil
		}
		content = string(data)
	}

	if parseErr != nil {
		return Result{Error: fmt.Sprintf("failed to parse %s: %v", ext, parseErr)}, nil
	}

	lines := strings.Split(content, "\n")
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
	// Add header for context
	if ext == ".pdf" || ext == ".xlsx" || ext == ".xlsm" {
		fmt.Fprintf(&sb, "[Reading %s as %s]\n", filepath.Base(args.Path), ext)
	}

	for i := start; i < end; i++ {
		line := lines[i]
		// Truncate very long lines (unless it's a table row which might be long but structure matters)
		// For tables, 2000 chars is plenty, but let's be safe.
		if len(line) > 4000 {
			line = line[:4000] + "... [line truncated]"
		}
		// Special formatting for different types?
		// Text file: Numbered lines are good for editing.
		// PDF/Excel: Numbered lines might distract from the "clean view" user asked for.
		// Let's keep line numbers for consistency and referencing, but maybe make them subtle?
		// User asked for "proper to the view".
		// For Markdown tables (Excel), line numbers break the copy-paste ability of the table.
		// If it's Excel/PDF, maybe we skip line numbers or output them differently.

		if ext == ".xlsx" || ext == ".xlsm" {
			// For Excel tables, just output the line raw to preserve markdown structure
			fmt.Fprintf(&sb, "%s\n", line)
		} else {
			// For text/code/PDF, numbered lines are useful for "read lines 10-20"
			fmt.Fprintf(&sb, "%4d\t%s\n", i+1, line)
		}
	}

	if end < len(lines) {
		fmt.Fprintf(&sb, "\n... (%d more lines, use offset=%d to continue)\n", len(lines)-end, end)
	}

	return Result{Output: sb.String()}, nil
}
