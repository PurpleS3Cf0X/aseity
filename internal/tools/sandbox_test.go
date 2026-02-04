package tools

import (
	"context"
	"strings"
	"testing"
)

func TestSandboxDef(t *testing.T) {
	s := NewSandboxRunTool()
	if s.Name() != "sandbox_run" {
		t.Errorf("expected name sandbox_run, got %s", s.Name())
	}
	if !s.NeedsConfirmation() {
		t.Error("expected sandbox to need confirmation")
	}
}

func TestSandboxArgs(t *testing.T) {
	// We can't easily execute docker in unit tests without strict environment,
	// but we can test invalid json at least.
	s := NewSandboxRunTool()
	ctx := context.Background()
	res, _ := s.Execute(ctx, "{invalid_json")
	if !strings.Contains(res.Error, "invalid arguments") {
		t.Errorf("expected arg parsing error, got %v", res)
	}
}
