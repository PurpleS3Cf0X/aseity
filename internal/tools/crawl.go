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
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type WebCrawlTool struct{}

type webCrawlArgs struct {
	URL        string   `json:"url,omitempty"`
	URLs       []string `json:"urls,omitempty"` // Support for batch crawling
	WaitFor    string   `json:"wait_for,omitempty"`
	Screenshot bool     `json:"screenshot,omitempty"`
}

func (w *WebCrawlTool) Name() string            { return "web_crawl" }
func (w *WebCrawlTool) NeedsConfirmation() bool { return false }
func (w *WebCrawlTool) Description() string {
	return "Crawl one or more websites using a headless browser. Supports parallel crawling. Capable of rendering JavaScript and SPAs."
}

func (w *WebCrawlTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url":        map[string]any{"type": "string", "description": "Single URL to crawl (legacy)"},
			"urls":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "List of URLs to crawl in parallel"},
			"wait_for":   map[string]any{"type": "string", "description": "Optional CSS selector to wait for"},
			"screenshot": map[string]any{"type": "boolean", "description": "Take screenshots?"},
		},
		"oneOf": []map[string]any{
			{"required": []string{"url"}},
			{"required": []string{"urls"}},
		},
	}
}

func (w *WebCrawlTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args webCrawlArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// Normalize URLs
	var targets []string
	if args.URL != "" {
		targets = append(targets, args.URL)
	}
	if len(args.URLs) > 0 {
		targets = append(targets, args.URLs...)
	}
	if len(targets) == 0 {
		return Result{Error: "no URLs provided"}, nil
	}

	// Deduplicate
	unique := make(map[string]bool)
	var cleanTargets []string
	for _, u := range targets {
		if !unique[u] {
			unique[u] = true
			cleanTargets = append(cleanTargets, u)
		}
	}
	args.URLs = cleanTargets

	// 1. Try Crawl4AI (Service Batch)
	if w.isCrawl4AIAvailable(ctx) {
		result, err := w.crawlBatchWithService(ctx, args)
		if err == nil {
			return result, nil
		}
		// Fallback if service fails
	}

	// 2. Fallback to Concurrent Chromedp/HTTP
	return w.crawlBatchFallback(ctx, args)
}

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

func (w *WebCrawlTool) crawlBatchWithService(ctx context.Context, args webCrawlArgs) (Result, error) {
	url := os.Getenv("CRAWL4AI_URL")
	if url == "" {
		url = "http://localhost:11235"
	}

	reqBody := map[string]interface{}{
		"urls":     args.URLs, // Pass array
		"priority": 10,
	}
	if args.Screenshot {
		reqBody["screenshot"] = true
	}

	jsonBody, _ := json.Marshal(reqBody)

	// Longer timeout for batch
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
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
		return Result{}, fmt.Errorf("service status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
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

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Batch Crawl (via Service) - %d URLs:\n\n", len(response.Results)))

	for _, res := range response.Results {
		content := res.Markdown
		if content == "" {
			content = htmlToText(res.HTML)
		}
		sb.WriteString(fmt.Sprintf("--- SOURCE: %s ---\n%s\n\n", res.URL, truncateText(content, 2000)))
	}

	return Result{Output: sb.String()}, nil
}

func (w *WebCrawlTool) crawlBatchFallback(ctx context.Context, args webCrawlArgs) (Result, error) {
	var wg sync.WaitGroup
	results := make([]string, len(args.URLs))
	errs := make([]error, len(args.URLs))

	// Limit concurrency to 3 to avoid resource exhaustion
	sem := make(chan struct{}, 3)

	for i, u := range args.URLs {
		wg.Add(1)
		go func(idx int, targetUrl string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Use single crawl logic
			res, err := w.crawlSingleFallback(ctx, targetUrl, args.WaitFor, args.Screenshot)
			if err != nil {
				errs[idx] = err
				results[idx] = fmt.Sprintf("Error crawling %s: %v", targetUrl, err)
			} else {
				results[idx] = res
			}
		}(i, u)
	}

	wg.Wait()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Batch Crawl (Fallback) - %d URLs:\n\n", len(args.URLs)))

	for _, r := range results {
		sb.WriteString(r + "\n\n")
	}

	return Result{Output: sb.String()}, nil
}

func (w *WebCrawlTool) crawlSingleFallback(ctx context.Context, urlStr, waitFor string, screenshot bool) (string, error) {
	// Chromedp logic recycled from original Execute
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("user-agent", "Mozilla/5.0 (compatible; Aseity/1.0)"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var textContent string
	var buf []byte

	actions := []chromedp.Action{chromedp.Navigate(urlStr)}
	if waitFor != "" {
		actions = append(actions, chromedp.WaitVisible(waitFor))
	} else {
		actions = append(actions, chromedp.WaitVisible("body"))
	}
	actions = append(actions, chromedp.Text("body", &textContent))
	if screenshot {
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(ctx, actions...); err != nil {
		// Fallback to HTTP
		return w.basicHTTPFetch(urlStr)
	}

	output := fmt.Sprintf("--- SOURCE: %s (Chromedp) ---\n%s", urlStr, truncateText(textContent, 2000))
	if screenshot && len(buf) > 0 {
		cwd, _ := os.Getwd()
		filename := fmt.Sprintf("screenshot_%d_%s.png", time.Now().Unix(), sanitizeFilename(urlStr))
		path := filepath.Join(cwd, filename)
		os.WriteFile(path, buf, 0644)
		output += fmt.Sprintf("\n[Screenshot: %s]", path)
	}
	return output, nil
}

func (w *WebCrawlTool) basicHTTPFetch(urlStr string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	req.Header.Set("User-Agent", "Aseity/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := htmlToText(string(body))
	return fmt.Sprintf("--- SOURCE: %s (HTTP) ---\n%s", urlStr, truncateText(text, 2000)), nil
}

func truncateText(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n... (truncated)"
	}
	return s
}

func sanitizeFilename(url string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, url)
}
