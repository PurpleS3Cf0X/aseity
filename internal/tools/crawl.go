package tools

import (
	"context"
	"encoding/json"
	"fmt"
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
			return Result{Error: "Web crawler requires a Chromium-based browser (Chrome, Brave, Edge). Please install configured browser to use this feature."}, nil
		}
		return Result{Error: fmt.Sprintf("crawl failed: %v", err)}, nil
	}

	output := fmt.Sprintf("Crawled: %s\n\nContent:\n%s", args.URL, truncateText(textContent, 5000))

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

func truncateText(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n... (truncated)"
	}
	return s
}
