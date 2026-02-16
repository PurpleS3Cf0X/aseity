package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// --- web_search ---

type WebSearchTool struct{}

type webSearchArgs struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results,omitempty"`
}

func (w *WebSearchTool) Name() string            { return "web_search" }
func (w *WebSearchTool) NeedsConfirmation() bool { return false }
func (w *WebSearchTool) Description() string {
	return "Search the web using DuckDuckGo. Returns titles, URLs, and snippets for each result."
}

func (w *WebSearchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":       map[string]any{"type": "string", "description": "Search query"},
			"num_results": map[string]any{"type": "integer", "description": "Max results to return (default 8)"},
		},
		"required": []string{"query"},
	}
}

func (w *WebSearchTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args webSearchArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}
	if args.NumResults == 0 {
		args.NumResults = 8
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Use DuckDuckGo HTML search
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(args.Query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Aseity/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Error: "search failed: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{Error: "failed to read response: " + err.Error()}, nil
	}
	html := string(body)

	// Parse results from DuckDuckGo HTML
	results := parseDDGResults(html, args.NumResults)
	if len(results) == 0 {
		return Result{Output: "No results found for: " + args.Query}, nil
	}

	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "Result %d:\nTitle: %s\nURL: %s\nSnippet: %s\n\n", i+1, r.title, r.url, r.snippet)
	}
	return Result{Output: sb.String()}, nil
}

type searchResult struct {
	title   string
	url     string
	snippet string
}

var (
	ddgResultRe  = regexp.MustCompile(`<a rel="nofollow" class="result__a" href="([^"]*)"[^>]*>(.*?)</a>`)
	ddgSnippetRe = regexp.MustCompile(`<a class="result__snippet"[^>]*>(.*?)</a>`)
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
	htmlEntityRe = regexp.MustCompile(`&[a-z]+;|&#[0-9]+;`)
)

func parseDDGResults(html string, max int) []searchResult {
	links := ddgResultRe.FindAllStringSubmatch(html, max)
	snippets := ddgSnippetRe.FindAllStringSubmatch(html, max)

	var results []searchResult
	for i, link := range links {
		if len(link) < 3 {
			continue
		}
		href := link[1]
		title := stripHTML(link[2])

		// DuckDuckGo wraps URLs in a redirect; extract the actual URL
		if u, err := url.Parse(href); err == nil {
			if actual := u.Query().Get("uddg"); actual != "" {
				href = actual
			}
		}

		snippet := ""
		if i < len(snippets) && len(snippets[i]) >= 2 {
			snippet = stripHTML(snippets[i][1])
		}

		results = append(results, searchResult{
			title:   title,
			url:     href,
			snippet: snippet,
		})
	}
	return results
}

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = htmlEntityRe.ReplaceAllStringFunc(s, func(entity string) string {
		switch entity {
		case "&amp;":
			return "&"
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&quot;":
			return "\""
		case "&#39;", "&apos;":
			return "'"
		case "&nbsp;":
			return " "
		}
		return entity
	})
	return strings.TrimSpace(s)
}

// --- web_fetch ---

type WebFetchTool struct{}

type webFetchArgs struct {
	URL string `json:"url"`
}

func (w *WebFetchTool) Name() string            { return "web_fetch" }
func (w *WebFetchTool) NeedsConfirmation() bool { return false }
func (w *WebFetchTool) Description() string {
	return "Fetch a URL and return its content as readable text. HTML is converted to plain text."
}

func (w *WebFetchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{"type": "string", "description": "The URL to fetch"},
		},
		"required": []string{"url"},
	}
}

func (w *WebFetchTool) Execute(ctx context.Context, rawArgs string) (Result, error) {
	var args webFetchArgs
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return Result{Error: "invalid arguments: " + err.Error()}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", args.URL, nil)
	if err != nil {
		return Result{Error: err.Error()}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Aseity/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{Error: "fetch failed: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Result{Error: fmt.Sprintf("HTTP %d from %s", resp.StatusCode, args.URL)}, nil
	}

	// Read up to 512KB
	limited := io.LimitReader(resp.Body, 512*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return Result{Error: "failed to read body: " + err.Error()}, nil
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "xhtml") {
		content = htmlToText(content)
	}

	// Truncate to ~8000 chars for context window
	if len(content) > 8000 {
		content = content[:8000] + "\n\n[... truncated]"
	}

	return Result{Output: content}, nil
}

// htmlToText does a lightweight conversion of HTML to readable text.
func htmlToText(html string) string {
	// Remove script and style blocks
	// Go regexp doesn't support backreferences like \1, so we handle them separately
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")
	noscriptRe := regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	html = noscriptRe.ReplaceAllString(html, "")

	// Convert common block elements to newlines
	blockRe := regexp.MustCompile(`(?i)</(p|div|h[1-6]|li|tr|br|hr)[^>]*>`)
	html = blockRe.ReplaceAllString(html, "\n")

	brRe := regexp.MustCompile(`(?i)<br\s*/?>`)
	html = brRe.ReplaceAllString(html, "\n")

	// Convert list items
	liRe := regexp.MustCompile(`(?i)<li[^>]*>`)
	html = liRe.ReplaceAllString(html, "\n• ")

	// Convert headings to bold-ish text
	hRe := regexp.MustCompile(`(?i)<h([1-6])[^>]*>`)
	html = hRe.ReplaceAllString(html, "\n## ")

	// Strip remaining tags
	html = htmlTagRe.ReplaceAllString(html, "")

	// Decode entities
	html = htmlEntityRe.ReplaceAllStringFunc(html, func(entity string) string {
		switch entity {
		case "&amp;":
			return "&"
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&quot;":
			return "\""
		case "&#39;", "&apos;":
			return "'"
		case "&nbsp;":
			return " "
		}
		return entity
	})

	// Collapse multiple blank lines
	multiNewline := regexp.MustCompile(`\n{3,}`)
	html = multiNewline.ReplaceAllString(html, "\n\n")

	// Collapse multiple spaces
	multiSpace := regexp.MustCompile(`[ \t]+`)
	html = multiSpace.ReplaceAllString(html, " ")

	// Trim each line
	lines := strings.Split(html, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
