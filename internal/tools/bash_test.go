package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBashTool_SmartTruncation(t *testing.T) {
	// Create tool
	tool := &BashTool{}

	// Create a command that produces large output
	// 3000 chars of "a"
	cmd := "printf 'a%.0s' {1..3000}"
	args := `{"command": "` + cmd + `"}`

	// Execute
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Verify truncation
	const MaxOutputChars = 2000
	if len(result.Output) > MaxOutputChars+500 { // Allow some buffer for the footer msg
		t.Errorf("Output length %d exceeds expected range ~2000", len(result.Output))
	}

	// Verify footer message
	if !strings.Contains(result.Output, "Full output saved to") {
		t.Errorf("Output does not contain truncation message. Got: %s", result.Output)
	}

	// Verify file was created
	// Extract filename from output
	// Output format: ... saved to /tmp/aseity_output_X.txt. Use ...
	parts := strings.Split(result.Output, "saved to ")
	if len(parts) < 2 {
		t.Fatalf("Could not parse filename from output: %s", result.Output)
	}
	subparts := strings.Split(parts[1], ". Use")
	filename := strings.TrimSpace(subparts[0])

	// Read file
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Could not read saved output file %s: %v", filename, err)
	}

	if len(content) != 3000 {
		t.Errorf("Saved file content length %d, expected 3000", len(content))
	}

	// Cleanup
	os.Remove(filename)
}

func TestBashTool_Timeout(t *testing.T) {
	tool := &BashTool{}

	// Command that sleeps for 2 seconds
	// Timeout set to 1 second
	args := `{"command": "sleep 2", "timeout": 1}`

	start := time.Now()
	result, err := tool.Execute(context.Background(), args)
	duration := time.Since(start)

	// Expect NO error in return (err is nil), but Error field in Result
	if err != nil {
		t.Errorf("Unexpected tool error: %v", err)
	}

	// Check if Result.Error is populated (timeout usually results in "signal: killed" or similar)
	if result.Error == "" {
		t.Error("Expected execution error (timeout), got empty Result.Error")
	} else {
		t.Logf("Got expected error: %s", result.Error)
	}

	if duration >= 2*time.Second {
		t.Errorf("Command took %v, expected ~1s (timeout)", duration)
	}
}
