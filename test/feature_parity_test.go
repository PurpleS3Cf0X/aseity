package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeanpaul/aseity/internal/tools"
)

// TestFeatureParity verifies Phase 3.5 additions
func TestFeatureParity(t *testing.T) {
	// Setup temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "aseity_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	// root/file1.txt
	// root/sub/file2.go
	// root/sub/nested/file3.md
	// root/ignore/file4.log

	os.MkdirAll(filepath.Join(tmpDir, "sub", "nested"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "ignore"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "file2.go"), []byte("package main\nfunc main(){}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "nested", "file3.md"), []byte("# Title"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "ignore", "file4.log"), []byte("log data"), 0644)

	ctx := context.Background()

	t.Run("FileLs_Recursive", func(t *testing.T) {
		tool := &tools.FileLsTool{}
		args := fmt.Sprintf(`{"path": "%s", "recursive": true}`, tmpDir)

		res, err := tool.Execute(ctx, args)
		if err != nil {
			t.Fatalf("FileLs failed: %v", err)
		}

		// Verify output structure
		output := res.Output
		if !strings.Contains(output, "file1.txt") {
			t.Error("Missing file1.txt")
		}
		if !strings.Contains(output, "sub/") {
			t.Error("Missing sub/ directory")
		}
		if !strings.Contains(output, "file2.go") {
			t.Error("Missing file2.go")
		}
		if !strings.Contains(output, "nested/") {
			t.Error("Missing nested/ directory")
		}
	})

	t.Run("FileRead_Glob", func(t *testing.T) {
		tool := &tools.FileReadTool{}

		// Test glob matching (e.g. all files in sub and nested)
		pattern := filepath.Join(tmpDir, "sub", "**", "*")
		args := fmt.Sprintf(`{"path": "%s"}`, pattern)

		res, err := tool.Execute(ctx, args)
		if err != nil {
			t.Fatalf("FileRead with glob failed: %v", err)
		}

		output := res.Output

		// Should find file2.go, file3.md AND directory 'nested'
		if !strings.Contains(output, "Found 3 files") {
			t.Errorf("Expected 'Found 3 files', got output:\n%s", output)
		}
		if !strings.Contains(output, "file2.go") {
			t.Error("Missing file2.go content wrapper")
		}
		if !strings.Contains(output, "file3.md") {
			t.Error("Missing file3.md content wrapper")
		}
		// Content check
		if !strings.Contains(output, "package main") {
			t.Error("Missing file2.go content")
		}
		if !strings.Contains(output, "# Title") {
			t.Error("Missing file3.md content")
		}
	})
}
