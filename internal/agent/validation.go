package agent

import (
	"os"
	"runtime"

	"github.com/jeanpaul/aseity/internal/agent/skillsets"
	"github.com/jeanpaul/aseity/internal/provider"
)

// validateToolCall validates a tool call before execution
func (a *Agent) validateToolCall(call provider.ToolCall) error {
	// Create validation context
	ctx := skillsets.ValidationContext{
		OS:  runtime.GOOS,
		CWD: getCurrentWorkingDir(),
	}

	// Validate based on model's validation level
	return skillsets.ValidateToolCall(call, a.profile.ValidationLevel, ctx)
}

// suggestCorrection suggests a correction for a failed tool execution
func (a *Agent) suggestCorrection(call provider.ToolCall, errorMsg string) string {
	ctx := skillsets.ValidationContext{
		OS:  runtime.GOOS,
		CWD: getCurrentWorkingDir(),
	}

	return skillsets.SuggestCorrection(call, errorMsg, ctx)
}

// getCurrentWorkingDir safely gets the current working directory
func getCurrentWorkingDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return cwd
}
