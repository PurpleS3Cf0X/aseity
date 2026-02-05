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

type OpenAIProvider struct {
	name    string
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func NewOpenAI(name, baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{},
	}
}

func (o *OpenAIProvider) Name() string { return o.name }

func (o *OpenAIProvider) ModelName() string { return o.model }

func (o *OpenAIProvider) Models(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}
	return models, nil
}

type oaiRequest struct {
	Model    string       `json:"model"`
	Messages []oaiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
	Tools    []oaiTool    `json:"tools,omitempty"`
}

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiTool struct {
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}

type oaiFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type oaiToolCall struct {
	Index    *int            `json:"index,omitempty"`
	ID       string          `json:"id,omitempty"`
	Type     string          `json:"type"`
	Function oaiToolCallFunc `json:"function"`
}

type oaiToolCallFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type oaiStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string        `json:"content"`
			ToolCalls []oaiToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (o *OpenAIProvider) Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamChunk, error) {
	oaiMsgs := make([]oaiMessage, len(msgs))
	for i, m := range msgs {
		om := oaiMessage{Role: string(m.Role), Content: m.Content, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, oaiToolCall{
				ID:       tc.ID,
				Type:     "function",
				Function: oaiToolCallFunc{Name: tc.Name, Arguments: tc.Args},
			})
		}
		oaiMsgs[i] = om
	}

	var oaiTools []oaiTool
	for _, t := range tools {
		oaiTools = append(oaiTools, oaiTool{
			Type:     "function",
			Function: oaiFunction{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
		})
	}

	body := oaiRequest{Model: o.model, Messages: oaiMsgs, Stream: true, Tools: oaiTools}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("provider %s: %s", o.name, parseProviderError(o.name, resp.StatusCode, body))
	}

	ch := make(chan StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		toolCalls := map[int]*ToolCall{}
		// Track <think> blocks for reasoning models (DeepSeek-R1, QwQ, etc.)
		inThink := false
		var contentBuf strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				var tcs []ToolCall
				for _, tc := range toolCalls {
					tcs = append(tcs, *tc)
				}
				// Note: [DONE] marker doesn't include usage, it was in previous chunk
				ch <- StreamChunk{Done: true, ToolCalls: tcs}
				return
			}
			var chunk oaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				// Parse <think>...</think> blocks from reasoning models
				contentBuf.WriteString(delta.Content)
				text := contentBuf.String()

				if !inThink && strings.Contains(text, "<think>") {
					// Split: content before <think> is regular, rest is thinking
					parts := strings.SplitN(text, "<think>", 2)
					if parts[0] != "" {
						ch <- StreamChunk{Delta: parts[0]}
					}
					inThink = true
					contentBuf.Reset()
					contentBuf.WriteString(parts[1])
				} else if inThink && strings.Contains(text, "</think>") {
					parts := strings.SplitN(text, "</think>", 2)
					if parts[0] != "" {
						ch <- StreamChunk{Thinking: parts[0]}
					}
					inThink = false
					contentBuf.Reset()
					if parts[1] != "" {
						contentBuf.WriteString(parts[1])
						ch <- StreamChunk{Delta: parts[1]}
					}
				} else if inThink {
					// Emit thinking content as it streams
					ch <- StreamChunk{Thinking: delta.Content}
					contentBuf.Reset()
				} else {
					ch <- StreamChunk{Delta: delta.Content}
					contentBuf.Reset()
				}
			}
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}
				if _, ok := toolCalls[idx]; !ok {
					toolCalls[idx] = &ToolCall{ID: tc.ID, Name: tc.Function.Name}
				}
				toolCalls[idx].Args += tc.Function.Arguments
				if tc.ID != "" {
					toolCalls[idx].ID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCalls[idx].Name = tc.Function.Name
				}
			}
			if chunk.Choices[0].FinishReason != nil {
				var tcs []ToolCall
				for _, tc := range toolCalls {
					tcs = append(tcs, *tc)
				}
				// Populate usage if available
				var usage *Usage
				if chunk.Usage != nil {
					usage = &Usage{
						InputTokens:  chunk.Usage.PromptTokens,
						OutputTokens: chunk.Usage.CompletionTokens,
						TotalTokens:  chunk.Usage.TotalTokens,
					}
				}
				ch <- StreamChunk{Done: true, ToolCalls: tcs, Usage: usage}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Error: err, Done: true}
		}
	}()
	return ch, nil
}
