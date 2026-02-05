package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWebCrawlTool_Crawl4AIIntegration(t *testing.T) {
	// 1. Mock Crawl4AI Service
	mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle Health Check
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle Crawl Request
		if r.URL.Path == "/crawl" {
			// Verify request body
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			if reqBody["urls"] == "" {
				http.Error(w, "missing urls", http.StatusBadRequest)
				return
			}

			// Mock Response
			response := map[string]interface{}{
				"results": []map[string]interface{}{
					{
						"url":      reqBody["urls"],
						"markdown": "# Crawl4AI Success\n\nThis content came from the mock service.",
						"html":     "<h1>Crawl4AI Success</h1><p>This content came from the mock service.</p>",
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		http.NotFound(w, r)
	}))
	defer mockService.Close()

	// 2. Set Env Var to point to mock service
	os.Setenv("CRAWL4AI_URL", mockService.URL)
	defer os.Unsetenv("CRAWL4AI_URL")

	// 3. Execute Tool
	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": "https://example.com",
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// 4. Verify Results
	expectedContent := "This content came from the mock service"
	if !strings.Contains(result.Output, expectedContent) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedContent, result.Output)
	}

	if !strings.Contains(result.Output, "Crawl4AI Service") {
		t.Errorf("Expected output to indicate Crawl4AI usage, got: %s", result.Output)
	}
}

func TestWebCrawlTool_Crawl4AIFallback(t *testing.T) {
	// Test that it falls back to Chromedp (or HTTP in this case) if service is down

	// Point to non-existent URL
	os.Setenv("CRAWL4AI_URL", "http://localhost:9999")
	defer os.Unsetenv("CRAWL4AI_URL")

	// Create a simple HTTP server to act as the target website for fallback
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Fallback Content</body></html>"))
	}))
	defer targetServer.Close()

	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": targetServer.URL,
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))
	if err != nil {
		t.Fatalf("Fallback Execute failed: %v", err)
	}

	// Should NOT say "Crawl4AI Service"
	if strings.Contains(result.Output, "Crawl4AI Service") {
		t.Error("Should not use Crawl4AI service when down")
	}

	// Should contain fallback content
	// Note: It might use Chromedp or HTTP depending on environment.
	// Both should find "Fallback Content"
	if !strings.Contains(result.Output, "Fallback Content") {
		t.Errorf("Expected fallback content, got: %s", result.Output)
	}
}
