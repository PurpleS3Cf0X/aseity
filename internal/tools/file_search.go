package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSearchTool struct{}

type fileSearchArgs struct {
	Pattern string `json:"pattern,omitempty"`
	Path    string `json:"path,omitempty"`
	Grep    string `json:"grep,omitempty"`
}

func (f *FileSearchTool) Name() string        { return "file_search" }
func (f *FileSearchTool) Description() string {
	return "Search for files by glob pattern or search file contents with a text query. Provide 'pattern' for glob matching or 'grep' for content search."
}
func (f *FileSearchTool) NeedsConfirmation() bool { return false }

func (f *FileSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Glob pattern (e.g. '**/*.go')"},
			"path":    map[string]any{"type": "string", "description": "Directory to search in (default: current dir)"},
			"grep":    map[string]any{"type": "string", "description": "Text to search for in file contents"},
		},
	}
}

func (f *FileSearchTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileSearchArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	root := args.Path
	if root == "" {
		root = "."
	}

	if args.Pattern != "" {
		var matches []string
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			matched, _ := filepath.Match(args.Pattern, filepath.Base(path))
			if matched {
				matches = append(matches, path)
			}
			if len(matches) >= 100 {
				return filepath.SkipAll
			}
			return nil
		})
		return Result{Output: strings.Join(matches, "\n")}, nil
	}

	if args.Grep != "" {
		var results []string
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || info.Size() > 1<<20 {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			lines := strings.Split(string(data), "\n")
			for i, line := range lines {
				if strings.Contains(line, args.Grep) {
					results = append(results, fmt.Sprintf("%s:%d: %s", path, i+1, line))
					if len(results) >= 100 {
						return filepath.SkipAll
					}
				}
			}
			return nil
		})
		return Result{Output: strings.Join(results, "\n")}, nil
	}

	return Result{Error: "provide either 'pattern' or 'grep'"}, nil
}
