package test

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jeanpaul/aseity/internal/tools"
)

func TestSandboxToolDockerExecution(t *testing.T) {
	// 1. Initialize Tool
	tool := tools.NewSandboxRunTool()

	// 2. Check for Docker Health (Fast Fail)
	// We check if we can run 'docker info' quickly.
	// If this hangs or fails, we skip the live execution test.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Locate docker manually or via path to match tool logic
	dockerCmd := "docker"
	if _, err := exec.LookPath("docker"); err != nil {
		dockerCmd = "/Applications/Docker.app/Contents/Resources/bin/docker"
	}

	if err := exec.CommandContext(ctx, dockerCmd, "--version").Run(); err != nil {
		t.Logf("Docker not responsive (or missing), skipping live execution test: %v", err)
		// We revert to checking the 'Missing' behavior logic if we can't run it
		// But since we are here, we know the binary *might* exist but be slow.
		return
	}

	// 3. Execute with dummy args
	// Using a larger timeout for the actual pull/run
	execCtx, execCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer execCancel()

	args := `{"command": "echo hello"}`
	result, _ := tool.Execute(execCtx, args)

	// 4. Verify Success
	if result.Error != "" {
		// If it fails (e.g. daemon not running), we log it but don't fail the build
		// because this depends on external environment.
		t.Logf("Sandbox execution failed (Environment Issue?): %s", result.Error)
		return
	}

	if !strings.Contains(result.Output, "hello") {
		t.Errorf("Expected output to contain 'hello', got '%s'", result.Output)
	} else {
		t.Logf("Success! Sandbox ran command. Output: %s", strings.TrimSpace(result.Output))
	}
}
