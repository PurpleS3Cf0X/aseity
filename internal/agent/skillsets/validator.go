package skillsets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeanpaul/aseity/internal/provider"
)

// ValidateToolCall validates a tool call based on the validation level
func ValidateToolCall(call provider.ToolCall, level ValidationLevel, context ValidationContext) error {
	switch level {
	case ValidationNone:
		return nil
	case ValidationLight:
		return validateJSON(call.Args)
	case ValidationMedium:
		if err := validateJSON(call.Args); err != nil {
			return err
		}
		return validateParameters(call)
	case ValidationStrict:
		if err := validateJSON(call.Args); err != nil {
			return err
		}
		if err := validateParameters(call); err != nil {
			return err
		}
		return validateContext(call, context)
	default:
		return nil
	}
}

// ValidationContext provides environment information for validation
type ValidationContext struct {
	OS           string   // "darwin", "linux", "windows"
	CWD          string   // Current working directory
	AvailableEnv []string // Available environment variables
}

// validateJSON checks if the arguments are valid JSON
func validateJSON(args string) error {
	var temp interface{}
	if err := json.Unmarshal([]byte(args), &temp); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	return nil
}

// validateParameters checks if required parameters are present and valid
func validateParameters(call provider.ToolCall) error {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(call.Args), &params); err != nil {
		return err
	}

	switch call.Name {
	case "bash":
		if _, ok := params["command"]; !ok {
			return fmt.Errorf("bash tool requires 'command' parameter")
		}
		if cmd, ok := params["command"].(string); ok {
			if strings.TrimSpace(cmd) == "" {
				return fmt.Errorf("bash command cannot be empty")
			}
		}

	case "file_read":
		if _, ok := params["path"]; !ok {
			return fmt.Errorf("file_read tool requires 'path' parameter")
		}
		if path, ok := params["path"].(string); ok {
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("file path cannot be empty")
			}
		}

	case "file_write":
		if _, ok := params["path"]; !ok {
			return fmt.Errorf("file_write tool requires 'path' parameter")
		}
		if _, ok := params["content"]; !ok {
			return fmt.Errorf("file_write tool requires 'content' parameter")
		}

	case "web_search":
		if _, ok := params["query"]; !ok {
			return fmt.Errorf("web_search tool requires 'query' parameter")
		}
		if query, ok := params["query"].(string); ok {
			if strings.TrimSpace(query) == "" {
				return fmt.Errorf("search query cannot be empty")
			}
		}
	}

	return nil
}

// validateContext checks if the tool call is appropriate for the current context
func validateContext(call provider.ToolCall, ctx ValidationContext) error {
	if call.Name != "bash" {
		return nil // Only validate bash commands for now
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(call.Args), &params); err != nil {
		return err
	}

	cmd, ok := params["command"].(string)
	if !ok {
		return nil
	}

	// Check for OS-specific commands
	if ctx.OS == "darwin" || ctx.OS == "linux" {
		// Unix-like systems
		if strings.Contains(cmd, "apt-get") && ctx.OS == "darwin" {
			return fmt.Errorf("apt-get is not available on macOS, use 'brew' instead")
		}
		if strings.Contains(cmd, "yum") && ctx.OS == "darwin" {
			return fmt.Errorf("yum is not available on macOS, use 'brew' instead")
		}
	} else if ctx.OS == "windows" {
		// Windows-specific validation
		if strings.Contains(cmd, "brew") {
			return fmt.Errorf("brew is not available on Windows, use 'choco' or 'winget' instead")
		}
		if strings.Contains(cmd, "apt-get") {
			return fmt.Errorf("apt-get is not available on Windows")
		}
	}

	// Check for dangerous commands
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=/dev/zero",
		"> /dev/sda",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return fmt.Errorf("dangerous command detected: %s", pattern)
		}
	}

	return nil
}

// SuggestCorrection suggests a correction for a failed tool call
func SuggestCorrection(call provider.ToolCall, errorMsg string, ctx ValidationContext) string {
	// OS-specific corrections
	if strings.Contains(errorMsg, "apt-get") && ctx.OS == "darwin" {
		return "Use 'brew' instead of 'apt-get' on macOS"
	}
	if strings.Contains(errorMsg, "command not found") {
		return "The command may not be installed. Try installing it first."
	}
	if strings.Contains(errorMsg, "permission denied") {
		return "You may need elevated permissions. Consider using sudo (with caution)."
	}
	if strings.Contains(errorMsg, "no such file or directory") {
		return "The file or directory doesn't exist. Check the path and try again."
	}
	if strings.Contains(errorMsg, "invalid JSON") {
		return "Fix the JSON syntax in the tool arguments."
	}

	// Tool-specific corrections
	switch call.Name {
	case "bash":
		if strings.Contains(errorMsg, "empty") {
			return "Provide a non-empty command to execute."
		}
	case "file_read":
		if strings.Contains(errorMsg, "path") {
			return "Provide a valid file path."
		}
	case "web_search":
		if strings.Contains(errorMsg, "query") {
			return "Provide a non-empty search query."
		}
	}

	return "The tool execution failed. Analyze the error message above. You MUST try a different tool (e.g., using 'web_fetch' instead of 'web_crawl') or start a new search. Do not give up."
}
