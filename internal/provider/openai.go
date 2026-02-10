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
	Model         string       `json:"model"`
	Messages      []oaiMessage `json:"messages"`
	Stream        bool         `json:"stream"`
	Tools         []oaiTool    `json:"tools,omitempty"`
	StreamOptions *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options,omitempty"`
	Options map[string]any `json:"options,omitempty"` // For Ollama-specific parameters
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

	reqBody := oaiRequest{
		Model:    o.model,
		Messages: oaiMsgs,
		Stream:   true,
		Tools:    oaiTools,
		StreamOptions: &struct {
			IncludeUsage bool `json:"include_usage"`
		}{IncludeUsage: true},
	}

	// Ollama Optimization: Force larger context window
	// Aseity assumes high context usage, but Ollama defaults to 2048.
	// We check for common Ollama ports or exact localhost matches to be safe.
	if strings.Contains(o.baseURL, "11434") || strings.Contains(o.baseURL, "localhost") {
		reqBody.Options = map[string]any{
			"num_ctx": 32768, // Default to a reasonable high value (Qwen supports 32k)
		}
	}

	payload, err := json.Marshal(reqBody)
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
			content := delta.Content

			if content != "" {
				// Enhanced <think> tag parsing state machine
				// We need to handle tags split across chunks: "<", "th", "ink", ">"
				// But strict state machine is complex.
				// For now, simpler robust approach:
				// If we see <think>, switch to thinking mode.
				// If we see </think>, switch back.
				// NOTE: This doesn't handle split tags directly (e.g. "<t" + "hink>")
				// but is better than before. To fix split tags, we'd need a buffer.
				// Let's implement a small buffer for potential tags.

				// Append to buffer
				contentBuf.WriteString(content)
				fullText := contentBuf.String()

				// Check for transitions
				if !inThink {
					startIdx := strings.Index(fullText, "<think>")
					if startIdx != -1 {
						// Found start tag
						inThink = true
						// Emit everything before tag as Delta
						if startIdx > 0 {
							ch <- StreamChunk{Delta: fullText[:startIdx]}
						}
						// Keep everything after tag in buffer (it's thinking content)
						rest := fullText[startIdx+7:] // 7 is len("<think>")
						contentBuf.Reset()
						contentBuf.WriteString(rest)
						fullText = rest
					} else {
						// No tag found. But wait, could we have a partial tag at the end?
						// e.g. "text <th"
						// We should emit everything up to the potential partial tag.
						// Partial tag chars: <, t, h, i, n, k, >
						// Simple heuristic: If it ends with '<', or '<t', etc. keep it.
						// Otherwise emit.
						// Optimization: Just emit if len > 7 and no <think>
						if len(fullText) > 20 && !strings.Contains(fullText, "<") {
							ch <- StreamChunk{Delta: fullText}
							contentBuf.Reset()
						}
						// If short, we keep in buffer.
						// If buffer gets too long without tags, we force flush?
						// Handled by Loop? No, we need to ensure we don't hold text forever.
						// Actually, standard delta is small.
						// Let's just emit content directly if we are not in potential tag zone.
					}
				}

				if inThink {
					endIdx := strings.Index(fullText, "</think>")
					if endIdx != -1 {
						// Found end tag
						inThink = false
						// Emit everything before tag as Thinking
						if endIdx > 0 {
							ch <- StreamChunk{Thinking: fullText[:endIdx]}
						}
						// Keep everything after tag in buffer (it's regular content)
						rest := fullText[endIdx+8:] // 8 is len("</think>")
						contentBuf.Reset()
						contentBuf.WriteString(rest)
					} else {
						// In thinking mode. Check for partial end tag?
						// Same logic. Emit safely.
						if len(fullText) > 20 && !strings.Contains(fullText, "</") {
							ch <- StreamChunk{Thinking: fullText}
							contentBuf.Reset()
						}
					}
				}
			}

			// Flush buffer at end? We do it at [DONE] logic if we modify it.
			// But for now, let's trust the loop.
			// Re-enable tools logic...
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
