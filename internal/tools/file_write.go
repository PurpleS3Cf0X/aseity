package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type FileWriteTool struct{}

type fileWriteArgs struct {
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	OldString string `json:"old_string,omitempty"`
	NewString string `json:"new_string,omitempty"`
}

func (f *FileWriteTool) Name() string        { return "file_write" }
func (f *FileWriteTool) Description() string {
	return "Write or edit a file. Provide 'content' to overwrite the entire file, or 'old_string' and 'new_string' to make a targeted replacement."
}
func (f *FileWriteTool) NeedsConfirmation() bool { return true }

func (f *FileWriteTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":       map[string]any{"type": "string", "description": "File path to write to"},
			"content":    map[string]any{"type": "string", "description": "Full file content (overwrites entire file)"},
			"old_string": map[string]any{"type": "string", "description": "Text to find and replace"},
			"new_string": map[string]any{"type": "string", "description": "Replacement text"},
		},
		"required": []string{"path"},
	}
}

func (f *FileWriteTool) Execute(_ context.Context, rawArgs string) (Result, error) {
	var args fileWriteArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	if args.OldString != "" {
		data, err := os.ReadFile(args.Path)
		if err != nil {
			return Result{Error: err.Error()}, nil
		}
		content := string(data)
		if !strings.Contains(content, args.OldString) {
			return Result{Error: "old_string not found in file"}, nil
		}
		count := strings.Count(content, args.OldString)
		if count > 1 {
			return Result{Error: "old_string matches multiple locations; provide more context to make it unique"}, nil
		}
		content = strings.Replace(content, args.OldString, args.NewString, 1)
		if err := os.WriteFile(args.Path, []byte(content), 0644); err != nil {
			return Result{Error: err.Error()}, nil
		}
		return Result{Output: "File edited successfully"}, nil
	}

	if err := os.MkdirAll(filepath.Dir(args.Path), 0755); err != nil {
		return Result{Error: err.Error()}, nil
	}
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return Result{Error: err.Error()}, nil
	}
	return Result{Output: "File written successfully"}, nil
}
