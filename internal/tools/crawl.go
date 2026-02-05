package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type WebCrawlTool struct{}

type webCrawlArgs struct {
	URL        string `json:"url"`
	WaitFor    string `json:"wait_for,omitempty"`   // CSS selector to wait for
	Screenshot bool   `json:"screenshot,omitempty"` // Take a screenshot?
}

func (w *WebCrawlTool) Name() string            { return "web_crawl" }
func (w *WebCrawlTool) NeedsConfirmation() bool { return false }
func (w *WebCrawlTool) Description() string {
	return "Crawl a website using a headless browser (Chromedp). Capable of rendering JavaScript and SPAs. Returns text content and optionally saves a screenshot."
}

func (w *WebCrawlTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":        map[string]any{"type": "string", "description": "The URL to crawl"},
			"wait_for":   map[string]any{"type": "string", "description": "Optional CSS selector to wait for before extracting text (e.g. '#content' or '.main-article')"},
			"screenshot": map[string]any{"type": "boolean", "description": "Set to true to capture a screenshot of the page"},
		},
		"required": []string{"url"},
	}
}

func (w *WebCrawlTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args webCrawlArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// 1. Try Crawl4AI (if available)
	if w.isCrawl4AIAvailable(ctx) {
		result, err := w.crawlWithService(ctx, args)
		if err == nil {
			return result, nil
		}
		// If failed, fall back to Chromedp
	}

	// 2. Fallback to Chromedp (Headless Browser)
	// Create headless context
	// We use Allocator to manage the browser instance
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Aseity/1.0"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create new context with log output
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout for the crawl
	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var textContent string
	var buf []byte

	actions := []chromedp.Action{
		chromedp.Navigate(args.URL),
	}

	// Wait logic
	if args.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(args.WaitFor))
	} else {
		// Default: Wait for body to be loaded
		actions = append(actions, chromedp.WaitVisible("body"))
	}

	// Extract text from body
	// We could get innerText of body, or just simple text
	// Let's get innerText of body
	actions = append(actions, chromedp.Text("body", &textContent))

	// Screenshot logic
	if args.Screenshot {
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	// Execute actions
	if err := chromedp.Run(ctx, actions...); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			// Fallback to basic HTTP fetch
			return w.fallbackFetch(args.URL)
		}
		// If generic error, try fallback anyway
		return w.fallbackFetch(args.URL)
	}

	output := fmt.Sprintf("Crawled (via Chromedp): %s\n\nContent:\n%s", args.URL, truncateText(textContent, 5000))

	if args.Screenshot && len(buf) > 0 {
		// Save screenshot to temp file
		cwd, _ := os.Getwd()
		filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
		path := filepath.Join(cwd, filename)
		if err := os.WriteFile(path, buf, 0644); err != nil {
			output += fmt.Sprintf("\n\n[Warning: Failed to save screenshot: %v]", err)
		} else {
			output += fmt.Sprintf("\n\n[Screenshot saved to: %s]", path)
		}
	}

	return Result{Output: output}, nil
}

// isCrawl4AIAvailable checks if the service is healthy
func (w *WebCrawlTool) isCrawl4AIAvailable(ctx context.Context) bool {
	url := os.Getenv("CRAWL4AI_URL")
	if url == "" {
		url = "http://localhost:11235"
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// crawlWithService performs the crawl using the Docker microservice
func (w *WebCrawlTool) crawlWithService(ctx context.Context, args webCrawlArgs) (Result, error) {
	url := os.Getenv("CRAWL4AI_URL")
	if url == "" {
		url = "http://localhost:11235"
	}

	reqBody := map[string]interface{}{
		"urls":     args.URL, // Service expects 'urls' (comma separated or single)
		"priority": 10,
	}
	if args.Screenshot {
		reqBody["screenshot"] = true
	}

	jsonBody, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url+"/crawl", strings.NewReader(string(jsonBody)))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Result{}, fmt.Errorf("service returned status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	// Parse response
	// The service returns: { "results": [ { "url": "...", "markdown": "...", "html": "..." } ] }
	var response struct {
		Results []struct {
			URL      string `json:"url"`
			Markdown string `json:"markdown"`
			HTML     string `json:"html"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return Result{}, err
	}

	if len(response.Results) == 0 {
		return Result{}, fmt.Errorf("no results from service")
	}

	// Get markdown content - Crawl4AI provides excellent markdown conversion
	content := response.Results[0].Markdown
	if content == "" {
		// Fallback to HTML if markdown empty
		content = htmlToText(response.Results[0].HTML)
	}

	output := fmt.Sprintf("Crawled (via Crawl4AI Service): %s\n\nContent:\n%s", args.URL, truncateText(content, 5000))
	return Result{Output: output}, nil
}

func (w *WebCrawlTool) fallbackFetch(url string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Result{Error: "fallback failed: " + err.Error()}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Aseity/1.0; +http://aseity.app)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Error: "fallback request failed: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{Error: "failed to read fallback body: " + err.Error()}, nil
	}

	content := string(body)
	// Reuse htmlToText from web.go (same package)
	text := htmlToText(content)

	output := fmt.Sprintf("[WARNING: Native browser not found. Using basic HTTP fallback. Install Chrome for better results.]\n\nCrawled (Fallback): %s\n\nContent:\n%s", url, truncateText(text, 5000))
	return Result{Output: output}, nil
}

func truncateText(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n... (truncated)"
	}
	return s
}
