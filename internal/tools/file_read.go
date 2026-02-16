package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
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
	return "Read the contents of a file or multiple files using glob patterns (e.g. 'internal/**/*.go'). Supports Text, PDF, and Excel. Returns content with line numbers (text) or markdown tables (Excel)."
}
func (f *FileReadTool) NeedsConfirmation() bool { return false }

func (f *FileReadTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":   map[string]any{"type": "string", "description": "File path or glob pattern (e.g. '**/*.go')"},
			"offset": map[string]any{"type": "integer", "description": "Line offset (only for single file)"},
			"limit":  map[string]any{"type": "integer", "description": "Max lines (default 2000)"},
		},
		"required": []string{"path"},
	}
}

func (f *FileReadTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileReadArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// Check for glob
	if isGlob(args.Path) {
		matches, err := doublestar.FilepathGlob(args.Path)
		if err != nil {
			return Result{Error: "glob error: " + err.Error()}, nil
		}
		if len(matches) == 0 {
			return Result{Error: fmt.Sprintf("no files found matching pattern: %s", args.Path)}, nil
		}

		// Safety limit
		if len(matches) > 20 {
			return Result{Error: fmt.Sprintf("matches %d files (limit 20). Please refine your pattern.", len(matches))}, nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d files:\n\n", len(matches)))

		for _, match := range matches {
			content, err := f.readFile(match, 0, 1000) // Default limit 1000 lines per file in batch mode
			if err != nil {
				fmt.Fprintf(&sb, "## Error reading %s: %v\n\n", match, err)
			} else {
				fmt.Fprintf(&sb, "## File: %s\n%s\n\n", match, content)
			}
		}
		return Result{Output: sb.String()}, nil
	}

	// Single file read
	content, err := f.readFile(args.Path, args.Offset, args.Limit)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: content}, nil
}

func isGlob(path string) bool {
	return strings.ContainsAny(path, "*?[{")
}

func (f *FileReadTool) readFile(path string, offset, limit int) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if info.Size() > maxFileReadSize {
		return "", fmt.Errorf("file too large (%d bytes)", info.Size())
	}

	ext := strings.ToLower(filepath.Ext(path))
	var content string
	var parseErr error

	switch ext {
	case ".pdf":
		content, parseErr = parsePDF(path)
	case ".xlsx", ".xlsm":
		content, parseErr = parseExcel(path)
	default:
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		content = string(data)
	}

	if parseErr != nil {
		return "", fmt.Errorf("failed to parse %s: %v", ext, parseErr)
	}

	lines := strings.Split(content, "\n")
	start := offset
	if start > len(lines) {
		start = len(lines)
	}

	if limit <= 0 {
		limit = 2000
	}

	end := len(lines)
	if start+limit < end {
		end = start + limit
	}

	var sb strings.Builder
	// In glob mode, we don't need header context here as Execute adds it.
	// But in single mode we might want it?
	// The original code added "[Reading ...]" only for PDF/Excel.
	// Let's keep it simple and just output content.
	// The caller (Execute) adds headers for globs.

	// Re-add context header for single file reads (Execute wrapper doesn't act for single file)
	// Actually, let's just NOT add it here to obtain clean content.
	// The User usually just wants the file content.

	for i := start; i < end; i++ {
		line := lines[i]
		if len(line) > 4000 {
			line = line[:4000] + "... [truncated]"
		}

		if ext == ".xlsx" || ext == ".xlsm" {
			fmt.Fprintf(&sb, "%s\n", line)
		} else {
			fmt.Fprintf(&sb, "%4d\t%s\n", i+1, line)
		}
	}

	if end < len(lines) {
		fmt.Fprintf(&sb, "\n... (%d more lines)\n", len(lines)-end)
	}

	return sb.String(), nil
}
