package tools

import (
	"context"
	"encoding/json"
	"fmt"
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

			rawURLs := reqBody["urls"]
			var urls []string

			// Handle string or list
			switch v := rawURLs.(type) {
			case string:
				urls = []string{v}
			case []interface{}:
				for _, u := range v {
					urls = append(urls, u.(string))
				}
			}

			if len(urls) == 0 {
				http.Error(w, "missing urls", http.StatusBadRequest)
				return
			}

			// Mock Response
			response := map[string]interface{}{}
			var results []map[string]interface{}

			for _, u := range urls {
				results = append(results, map[string]interface{}{
					"url":      u,
					"markdown": fmt.Sprintf("# Success %s\nContent", u),
					"html":     "<h1>Success</h1>",
				})
			}
			response["results"] = results
			json.NewEncoder(w).Encode(response)
			return
		}

		http.NotFound(w, r)
	}))
	defer mockService.Close()

	// 2. Set Env Var
	os.Setenv("CRAWL4AI_URL", mockService.URL)
	defer os.Unsetenv("CRAWL4AI_URL")

	// 3. Execute Tool (Single)
	tool := &WebCrawlTool{}
	args := map[string]interface{}{
		"url": "https://example.com/1",
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result.Output, "Success https://example.com/1") {
		t.Errorf("Expected single output, got: %s", result.Output)
	}

	// 4. Execute Tool (Batch)
	argsBatch := map[string]interface{}{
		"urls": []string{"https://example.com/A", "https://example.com/B"},
	}
	argsBatchJSON, _ := json.Marshal(argsBatch)

	resultBatch, err := tool.Execute(ctx, string(argsBatchJSON))
	if err != nil {
		t.Fatalf("Batch Execution failed: %v", err)
	}

	if !strings.Contains(resultBatch.Output, "Success https://example.com/A") ||
		!strings.Contains(resultBatch.Output, "Success https://example.com/B") {
		t.Errorf("Expected batch output to contain both URLs, got: %s", resultBatch.Output)
	}
}

func TestWebCrawlTool_Crawl4AIFallback(t *testing.T) {
	// Test that it falls back to Chromedp (or HTTP in this case) if service is down

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
		"urls": []string{targetServer.URL + "/1", targetServer.URL + "/2"},
	}
	argsJSON, _ := json.Marshal(args)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, string(argsJSON))
	if err != nil {
		t.Fatalf("Fallback Execute failed: %v", err)
	}

	// Should contain fallback content for both
	// We count occurrences of "Fallback Content"
	count := strings.Count(result.Output, "Fallback Content")
	if count < 2 {
		t.Errorf("Expected fallback content for both URLs (count >= 2), got count %d\nOutput: %s", count, result.Output)
	}
}
