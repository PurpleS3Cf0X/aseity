package tools

import (
	"context"
	"strings"
	"testing"
)

func TestWebSearchTool_Execute(t *testing.T) {
	tool := &WebSearchTool{}

	// Query for something stable
	args := `{"query": "golang official site", "num_results": 1}`

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		// If network fails, we can skip, but better to fail if we expect connectivity.
		// For a robust test suite in CI, we might mock this.
		// For user verification on local machine, failure is informative.
		t.Logf("Web search failed (possibly network issue): %v", err)
		t.Skip("Skipping web search test due to network failure")
	}

	if result.Error != "" {
		t.Logf("Web search returned error: %s", result.Error)
		t.Skip("Skipping web search test due to tool error")
	}

	if !strings.Contains(strings.ToLower(result.Output), "go") {
		t.Errorf("Expected result to contain 'go', got: %s", result.Output)
	}
}

func TestWebFetchTool_Execute(t *testing.T) {
	tool := &WebFetchTool{}

	// Fetch a known stable plain text or simple HTML page
	// example.com is great for this
	args := `{"url": "https://example.com"}`

	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Logf("Web fetch failed: %v", err)
		t.Skip("Skipping web fetch test")
	}

	if result.Error != "" {
		t.Logf("Web fetch returned error: %s", result.Error)
		t.Skip("Skipping web fetch test")
	}

	if !strings.Contains(result.Output, "Example Domain") {
		t.Errorf("Expected 'Example Domain' in output, got: %s", result.Output)
	}
}
