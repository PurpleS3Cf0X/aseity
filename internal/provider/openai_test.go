package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockOllamaHandler validates requests and sends back responses
func MockOllamaHandler(t *testing.T, validation func(*oaiRequest)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		var req oaiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Run custom validation logic
		if validation != nil {
			validation(&req)
		}

		// Send a dummy stream response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Simulate a simple response
		response := oaiStreamChunk{
			Choices: []struct {
				Delta struct {
					Content   string        `json:"content"`
					ToolCalls []oaiToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			}{
				{
					Delta: struct {
						Content   string        `json:"content"`
						ToolCalls []oaiToolCall `json:"tool_calls"`
					}{Content: "Hello"},
				},
			},
		}

		data, _ := json.Marshal(response)
		w.Write([]byte("data: " + string(data) + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}
}

func TestOpenAI_EmptyContentMarshaling(t *testing.T) {
	// This test specifically verifies that empty content is NOT omitted from the JSON
	// failing this means the regression (invalid message content type) would return.

	// Create a more robust check using map[string]interface{}
	// because `json.NewDecoder` will just zero-value missing fields.
	rawCheckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}

		msgs, ok := raw["messages"].([]interface{})
		if !ok {
			t.Error("messages field missing or invalid")
			return
		}

		for _, m := range msgs {
			msgMap, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			if role, ok := msgMap["role"]; ok && role == "user" {
				if _, hasContent := msgMap["content"]; !hasContent {
					t.Error("JSON missing 'content' field for user message")
				}
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer rawCheckServer.Close()

	p := NewOpenAI("test", rawCheckServer.URL, "key", "model")

	msgs := []Message{
		{Role: RoleUser, Content: ""}, // Empty content should still send "content": ""
	}

	// We expect no error, even though the stream is empty/done immediately
	_, err := p.Chat(context.Background(), msgs, nil)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestOpenAI_LongConversation(t *testing.T) {
	// Stress test with large history
	msgCount := 100

	server := httptest.NewServer(MockOllamaHandler(t, func(req *oaiRequest) {
		if len(req.Messages) != msgCount {
			t.Errorf("Expected %d messages, got %d", msgCount, len(req.Messages))
		}
	}))
	defer server.Close()

	p := NewOpenAI("test", server.URL, "key", "model")

	var msgs []Message
	for i := 0; i < msgCount; i++ {
		msgs = append(msgs, Message{
			Role:    RoleUser,
			Content: strings.Repeat("a", 10), // valid content
		})
	}

	_, err := p.Chat(context.Background(), msgs, nil)
	if err != nil {
		t.Fatalf("Chat with long history failed: %v", err)
	}
}

func TestOpenAI_ToolCallFlow(t *testing.T) {
	// Verify tool calls don't break message structure
	server := httptest.NewServer(MockOllamaHandler(t, func(req *oaiRequest) {
		// Just ensure it parses
	}))
	defer server.Close()

	p := NewOpenAI("test", server.URL, "key", "model")

	msgs := []Message{
		{Role: RoleUser, Content: "run this"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "1", Name: "bash", Args: "{}"}}},
		{Role: RoleTool, ToolCallID: "1", Content: "output"},
	}

	_, err := p.Chat(context.Background(), msgs, nil)
	if err != nil {
		t.Fatalf("Chat with tool history failed: %v", err)
	}
}
