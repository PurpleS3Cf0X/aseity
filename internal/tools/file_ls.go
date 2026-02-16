package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileLsTool struct{}

type fileLsArgs struct {
	Path      string `json:"path,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

func (f *FileLsTool) Name() string { return "file_ls" }
func (f *FileLsTool) Description() string {
	return "List files and directories. Supports recursive listing to view project structure."
}
func (f *FileLsTool) NeedsConfirmation() bool { return false }

func (f *FileLsTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":      map[string]any{"type": "string", "description": "Directory to list (default: current dir)"},
			"recursive": map[string]any{"type": "boolean", "description": "List subdirectories recursively"},
			"limit":     map[string]any{"type": "integer", "description": "Max files to list (default 500)"},
		},
	}
}

func (f *FileLsTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileLsArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	root := args.Path
	if root == "" {
		root = "."
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 500
	}

	var sb strings.Builder
	count := 0

	// Handle single directory (non-recursive)
	if !args.Recursive {
		entries, err := os.ReadDir(root)
		if err != nil {
			return Result{Error: err.Error()}, nil
		}
		sb.WriteString(fmt.Sprintf("Listing %s:\n", root))
		for _, e := range entries {
			if count >= limit {
				sb.WriteString("... (limit reached)")
				break
			}
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			sb.WriteString(e.Name() + suffix + "\n")
			count++
		}
		return Result{Output: sb.String()}, nil
	}

	// Recursive walk (simple tree representation)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if count >= limit {
			return filepath.SkipAll
		}

		// Skip .git and hidden directories to reduce noise
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		// Calculate depth for indentation
		depth := strings.Count(rel, string(os.PathSeparator))
		indent := strings.Repeat("  ", depth)

		suffix := ""
		if d.IsDir() {
			suffix = "/"
		}

		sb.WriteString(fmt.Sprintf("%s%s%s\n", indent, d.Name(), suffix))
		count++
		return nil
	})

	if err != nil {
		return Result{Error: err.Error()}, nil
	}

	if count >= limit {
		sb.WriteString("... (limit reached)")
	}

	return Result{Output: sb.String()}, nil
}
