package memory

import (
	"os"
	"strings"
	"testing"
)

func TestAutoMemory_Persistence(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "memory_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override baseDir for testing (using a trick or just constructing it if we exposed it)
	// Since NewAutoMemory hardcodes the path, we might need a way to inject it or just use a local struct.
	// Let's modify NewAutoMemory to accept options? NO, let's keep it simple.
	// We can construct the struct directly.

	m := &AutoMemory{
		baseDir:   tmpDir,
		Learnings: []Learning{},
	}

	// 1. Add Learning
	err = m.AddLearning("fact", "Go is great")
	if err != nil {
		t.Fatalf("AddLearning failed: %v", err)
	}

	// 2. Add Duplicate (should be ignored)
	err = m.AddLearning("fact", "Go is great")
	if err != nil {
		t.Fatalf("AddLearning duplicate failed: %v", err)
	}

	if len(m.Learnings) != 1 {
		t.Errorf("Expected 1 learning, got %d", len(m.Learnings))
	}

	// 3. Save
	if err := m.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 4. Load into new instance
	m2 := &AutoMemory{
		baseDir:   tmpDir,
		Learnings: []Learning{},
	}
	if err := m2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(m2.Learnings) != 1 {
		t.Errorf("Loaded expected 1 learning, got %d", len(m2.Learnings))
	}

	if m2.Learnings[0].Content != "Go is great" {
		t.Errorf("Content mismatch: got %q", m2.Learnings[0].Content)
	}
}

func TestAutoMemory_RetrieveContext(t *testing.T) {
	m := &AutoMemory{
		Learnings: []Learning{},
	}

	// Empty context
	ctx, err := m.RetrieveContext()
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}
	if ctx != "" {
		t.Error("Expected empty context for empty memory")
	}

	// Populated
	m.AddLearning("preference", "I like green")
	ctx, err = m.RetrieveContext()
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}

	if !strings.Contains(ctx, "I like green") {
		t.Error("Context missing learning content")
	}
}
