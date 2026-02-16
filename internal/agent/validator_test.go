package agent

import (
	"context"
	"testing"

	"github.com/jeanpaul/aseity/internal/provider"
)

// MockProvider for testing validator
type MockProvider struct {
	Response string
}

func (m *MockProvider) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk, 1)
	go func() {
		ch <- provider.StreamChunk{Delta: m.Response, Done: true}
		close(ch)
	}()
	return ch, nil
}
func (m *MockProvider) Name() string      { return "mock" }
func (m *MockProvider) ModelName() string { return "mock-model" }
func (m *MockProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestValidator_Check(t *testing.T) {
	// We can't easily mock the LLM's *decision* without a real LLM,
	// unless we use the real provider in the test.
	// But we CAN test the parsing logic.

	ctx := context.Background()

	tests := []struct {
		name        string
		llmResponse string // What the LLM says
		wantValid   bool
	}{
		{
			name:        "LLM says VALID",
			llmResponse: "VALID",
			wantValid:   true,
		},
		{
			name:        "LLM says INVALID",
			llmResponse: "INVALID: Hallucination",
			wantValid:   false,
		},
		{
			name:        "LLM explanation then STOP",
			llmResponse: "This is a hallucination. STOP.",
			wantValid:   false,
		},
		{
			name:        "LLM explanation valid",
			llmResponse: "This matches the user request. VALID.",
			wantValid:   true, // parser looks for VALID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProv := &MockProvider{Response: tt.llmResponse}
			v := NewValidator(mockProv)

			history := []provider.Message{{Role: "user", Content: "irrelevant"}}
			tc := provider.ToolCall{Name: "bash", Args: "ls"}

			gotValid, _ := v.Check(ctx, history, tc)
			if gotValid != tt.wantValid {
				t.Errorf("Check() valid = %v, want %v (Response: %q)", gotValid, tt.wantValid, tt.llmResponse)
			}
		})
	}
}

// NOTE: This unit test only verifies the PARSING logic.
// To verify the PROMPT efficiency, we still need the integration test with a specific setup.
// But since the integration test passed (by failing the logic check appropriately), we know the prompt works for Case 2.
// We assume it works for Case 1 (hallucination) based on the logic: "Did user ask for it?".
