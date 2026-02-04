package test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/jeanpaul/aseity/internal/tools"
)

func TestSandboxToolDockerMissing(t *testing.T) {
	// 1. Initialize Tool
	tool := tools.NewSandboxRunTool()

	// 2. Execute with dummy args
	args := `{"command": "echo hello"}`
	result, _ := tool.Execute(context.Background(), args)

	// 3. Verify Logic
	// Since we know Docker is missing on this specific test runner, we expect the "docker command not found" error.
	// If Docker IS present (e.g. user installed it), this test might actually run execution or fail differently.
	// So we check for either "docker command not found" OR "sandbox execution failed" (implied existence but failure)
	// OR success (if docker works).

	// Check if docker exists first to know what to expect
	_, err := exec.LookPath("docker")
	dockerExists := err == nil

	if !dockerExists {
		expected := "docker command not found"
		if result.Error != expected && !strings.Contains(result.Error, expected) {
			t.Errorf("Expected error '%s', got '%s'", expected, result.Error)
		} else {
			t.Logf("Correctly identified missing docker: %s", result.Error)
		}
	} else {
		// If docker exists, we expect it to try running.
		// Since we didn't mock exec, it will try actual `docker run`.
		// It might fail if daemon is not running.
		t.Logf("Docker detected. Result: Output='%s', Error='%s'", result.Output, result.Error)
	}
}
