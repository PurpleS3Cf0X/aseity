package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebCrawlTool_BasicFetch(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
			<head><title>Test Page</title></head>
			<body>
				<h1>Hello World</h1>
				<p>This is a test page.</p>
			</body>
			</html>
		`))
	}))
	defer ts.Close()

	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": ts.URL,
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Error != "" {
		t.Errorf("Expected no error, got: %s", result.Error)
	}

	if !strings.Contains(result.Output, "Hello World") {
		t.Errorf("Expected output to contain 'Hello World', got: %s", result.Output)
	}

	if !strings.Contains(result.Output, "test page") {
		t.Errorf("Expected output to contain 'test page', got: %s", result.Output)
	}
}

func TestWebCrawlTool_InvalidURL(t *testing.T) {
	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": "https://this-domain-definitely-does-not-exist-12345.com",
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))

	// Should either return error or result with error field
	if err == nil && result.Error == "" {
		t.Error("Expected error for invalid URL")
	}
}

func TestWebCrawlTool_MissingURL(t *testing.T) {
	tool := &WebCrawlTool{}
	args := map[string]interface{}{}
	argsJSON, _ := json.Marshal(args)

	ctx := context.Background()
	result, err := tool.Execute(ctx, string(argsJSON))

	if err == nil && result.Error == "" {
		t.Error("Expected error when URL is missing")
	}
}

func TestWebCrawlTool_InvalidJSON(t *testing.T) {
	tool := &WebCrawlTool{}

	ctx := context.Background()
	result, err := tool.Execute(ctx, "invalid json")

	if err == nil && result.Error == "" {
		t.Error("Expected error for invalid JSON")
	}
}

func TestWebCrawlTool_Timeout(t *testing.T) {
	// Create slow server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Second) // Longer than timeout
		w.Write([]byte("Too slow"))
	}))
	defer ts.Close()

	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": ts.URL,
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))

	// Should timeout
	if err == nil && result.Error == "" {
		t.Error("Expected timeout error")
	}
}

func TestWebCrawlTool_ToolInterface(t *testing.T) {
	tool := &WebCrawlTool{}

	// Test Name
	if tool.Name() != "web_crawl" {
		t.Errorf("Expected name 'web_crawl', got '%s'", tool.Name())
	}

	// Test Description
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}
	if !strings.Contains(desc, "web") {
		t.Error("Description should mention web scraping")
	}

	// Test NeedsConfirmation
	if tool.NeedsConfirmation() {
		t.Error("web_crawl should not need confirmation")
	}
}

func TestWebCrawlTool_FallbackBehavior(t *testing.T) {
	// This test verifies that the tool falls back gracefully
	// when chromedp is not available

	tool := &WebCrawlTool{}

	// Create simple test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>Fallback test</body></html>"))
	}))
	defer ts.Close()

	args := map[string]interface{}{
		"url": ts.URL,
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))

	// Should succeed with fallback
	if err != nil {
		t.Fatalf("Fallback should work: %v", err)
	}

	// Check if fallback warning is present
	if strings.Contains(result.Output, "WARNING") || strings.Contains(result.Output, "Fallback") {
		t.Log("Fallback mechanism activated (expected if chromedp not available)")
	}
}
