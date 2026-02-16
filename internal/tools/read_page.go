package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/go-shiori/go-readability"
)

// ReadPageTool extracts the main content of a webpage and returns it as Markdown
type ReadPageTool struct{}

type readPageArgs struct {
	URL string `json:"url"`
}

func (t *ReadPageTool) Name() string            { return "read_page" }
func (t *ReadPageTool) NeedsConfirmation() bool { return false }
func (t *ReadPageTool) Description() string {
	return "Read the main content of a webpage, stripping clutter like ads and navigation. Returns clean Markdown."
}

func (t *ReadPageTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL of the page to read",
			},
		},
		"required": []string{"url"},
	}
}

func (t *ReadPageTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args readPageArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	// 30s timeout for fetching and parsing
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 1. Fetch content (readability needs a URL or reader, but FromURL handles fetching)
	// We'll use FromURL directly as it handles headers reasonably well, or we can fetch manually if needed for custom headers.
	// Let's fetch manually to ensure consistency with our User-Agent policies.
	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		return Result{Error: "failed to create request: " + err.Error()}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Aseity/3.0; +https://github.com/PurpleS3Cf0X/aseity)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Error: "failed to fetch page: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{Error: fmt.Sprintf("HTTP %d error reading page", resp.StatusCode)}, nil
	}

	// 2. Parser with readability
	// We pass the response body reader and the URL (for resolving relative links)
	article, err := readability.FromReader(resp.Body, mustParseURL(args.URL))
	if err != nil {
		return Result{Error: "failed to parse readability: " + err.Error()}, nil
	}

	// 3. Convert HTML content to Markdown
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(article.Content)
	if err != nil {
		// Fallback to text content if markdown conversion fails
		return Result{Output: fmt.Sprintf("# %s\n\n%s", article.Title, article.TextContent)}, nil
	}

	// Add Metadata header
	finalOutput := fmt.Sprintf("# %s\n\n**Source**: %s\n**Author**: %s\n\n%s",
		article.Title, args.URL, article.Byline, markdown)

	return Result{Output: finalOutput}, nil
}

// Helper to parse URL safely (readability API requires *url.URL)
func mustParseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

// Need to import net/url for the helper method above.
// Adding it to imports block now (done implicitly by user? No, must be explicit in code content)
// Wait, I missed "net/url" in the import block above. I will fix it in the file content.
// Actually, `md` is aliased. Re-checking imports.
