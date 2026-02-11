package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GoogleProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGoogle(apiKey, model string) *GoogleProvider {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GoogleProvider{apiKey: apiKey, model: model, client: &http.Client{}}
}

func (g *GoogleProvider) Name() string { return "google" }

func (g *GoogleProvider) ModelName() string { return g.model }

func (g *GoogleProvider) Models(_ context.Context) ([]string, error) {
	return []string{
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"gemini-2.0-flash-exp",
	}, nil
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Tools             []geminiTool    `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string        `json:"text,omitempty"`
	FunctionCall     *geminiFnCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFnResp `json:"functionResponse,omitempty"`
}

type geminiFnCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFnResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFnDecl `json:"functionDeclarations"`
}

type geminiFnDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

func (g *GoogleProvider) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamChunk, error) {
	var contents []geminiContent
	var sysInstruction *geminiContent

	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			sysInstruction = &geminiContent{Parts: []geminiPart{{Text: m.Content}}}
		case RoleUser:
			contents = append(contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: m.Content}}})
		case RoleAssistant:
			parts := []geminiPart{}
			if m.Content != "" {
				parts = append(parts, geminiPart{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var args map[string]any
				json.Unmarshal([]byte(tc.Args), &args)
				parts = append(parts, geminiPart{FunctionCall: &geminiFnCall{Name: tc.Name, Args: args}})
			}
			contents = append(contents, geminiContent{Role: "model", Parts: parts})
		case RoleTool:
			contents = append(contents, geminiContent{
				Role: "user",
				Parts: []geminiPart{{FunctionResponse: &geminiFnResp{
					Name:     "tool",
					Response: map[string]any{"result": m.Content},
				}}},
			})
		}
	}

	var gemTools []geminiTool
	if len(tools) > 0 {
		var decls []geminiFnDecl
		for _, t := range tools {
			decls = append(decls, geminiFnDecl{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
		}
		gemTools = []geminiTool{{FunctionDeclarations: decls}}
	}

	body := geminiRequest{Contents: contents, SystemInstruction: sysInstruction, Tools: gemTools}
	payload, _ := json.Marshal(body)

	// Use header for API key instead of URL parameter
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse", g.model)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google API error: %s", friendlyProviderError(err))
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("google returned %d: %s", resp.StatusCode, parseProviderError("google", resp.StatusCode, b))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			var resp struct {
				Candidates []struct {
					Content      geminiContent `json:"content"`
					FinishReason string        `json:"finishReason"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal([]byte(data), &resp); err != nil {
				continue
			}
			if len(resp.Candidates) == 0 {
				continue
			}
			cand := resp.Candidates[0]
			var toolCalls []ToolCall
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					ch <- StreamChunk{Delta: part.Text}
				}
				if part.FunctionCall != nil {
					args, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, ToolCall{
						ID: part.FunctionCall.Name, Name: part.FunctionCall.Name, Args: string(args),
					})
				}
			}
			if cand.FinishReason != "" {
				ch <- StreamChunk{Done: true, ToolCalls: toolCalls}
				return
			}
		}
	}()
	return ch, nil
}
