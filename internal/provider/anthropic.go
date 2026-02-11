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

type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropic(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}
	return &AnthropicProvider{apiKey: apiKey, model: model, client: &http.Client{}}
}

func (a *AnthropicProvider) Name() string { return "anthropic" }

func (a *AnthropicProvider) ModelName() string { return a.model }

func (a *AnthropicProvider) Models(_ context.Context) ([]string, error) {
	return []string{
		"claude-3-5-sonnet-20240620",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []anthropicMsg  `json:"messages"`
	Stream    bool            `json:"stream"`
	Tools     []anthropicTool `json:"tools,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

type anthropicEvent struct {
	Type         string          `json:"type"`
	Delta        json.RawMessage `json:"delta,omitempty"`
	Index        int             `json:"index,omitempty"`
	ContentBlock *struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"content_block,omitempty"`
}

func (a *AnthropicProvider) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamChunk, error) {
	var systemPrompt string
	var apiMsgs []anthropicMsg
	for _, m := range msgs {
		if m.Role == RoleSystem {
			if systemPrompt != "" {
				systemPrompt += "\n\n"
			}
			systemPrompt += m.Content
			continue
		}
		if m.Role == RoleTool {
			apiMsgs = append(apiMsgs, anthropicMsg{
				Role: "user",
				Content: []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				}},
			})
			continue
		}
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			var blocks []map[string]any
			if m.Content != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Args), &input)
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": input,
				})
			}
			apiMsgs = append(apiMsgs, anthropicMsg{Role: "assistant", Content: blocks})
			continue
		}
		apiMsgs = append(apiMsgs, anthropicMsg{Role: string(m.Role), Content: m.Content})
	}

	var apiTools []anthropicTool
	for _, t := range tools {
		apiTools = append(apiTools, anthropicTool{Name: t.Name, Description: t.Description, InputSchema: t.Parameters})
	}

	body := anthropicRequest{
		Model: a.model, MaxTokens: 8192, System: systemPrompt,
		Messages: apiMsgs, Stream: true, Tools: apiTools,
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic: %s", parseProviderError("anthropic", resp.StatusCode, b))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)

		var currentToolID, currentToolName string
		var toolArgsBuilder strings.Builder
		var toolCalls []ToolCall
		inThinkingBlock := false

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			var evt anthropicEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "content_block_start":
				if evt.ContentBlock != nil {
					switch evt.ContentBlock.Type {
					case "tool_use":
						currentToolID = evt.ContentBlock.ID
						currentToolName = evt.ContentBlock.Name
						toolArgsBuilder.Reset()
					case "thinking":
						inThinkingBlock = true
					}
				}
			case "content_block_delta":
				var delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					Thinking    string `json:"thinking"`
					PartialJSON string `json:"partial_json"`
				}
				json.Unmarshal(evt.Delta, &delta)
				if delta.Type == "thinking_delta" {
					ch <- StreamChunk{Thinking: delta.Thinking}
				} else if delta.Type == "text_delta" {
					ch <- StreamChunk{Delta: delta.Text}
				} else if delta.Type == "input_json_delta" {
					toolArgsBuilder.WriteString(delta.PartialJSON)
				}
			case "content_block_stop":
				if currentToolID != "" {
					toolCalls = append(toolCalls, ToolCall{
						ID: currentToolID, Name: currentToolName, Args: toolArgsBuilder.String(),
					})
					currentToolID = ""
				}
				if inThinkingBlock {
					inThinkingBlock = false
				}
			case "message_stop":
				ch <- StreamChunk{Done: true, ToolCalls: toolCalls}
				return
			}
		}
	}()
	return ch, nil
}
