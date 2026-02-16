package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// MockProvider simulates an LLM response sequences
type MockProvider struct {
	responses []string // Queue of responses
	calls     int
}

func (m *MockProvider) Name() string      { return "mock" }
func (m *MockProvider) ModelName() string { return "mock-model" }
func (m *MockProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func (m *MockProvider) Chat(ctx context.Context, msgs []provider.Message, tools []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)

	go func() {
		defer close(ch)
		time.Sleep(100 * time.Millisecond) // Simulate latency

		response := "Default response"
		if m.calls < len(m.responses) {
			response = m.responses[m.calls]
		}
		m.calls++

		// Check if response is a tool call (starts with [TOOL:)
		if strings.HasPrefix(strings.TrimSpace(response), "[TOOL:") {
			// Parse simple tool call
			// Format: [TOOL:name|args]
			parts := strings.SplitN(response[6:len(response)-1], "|", 2)
			if len(parts) == 2 {
				toolCall := provider.ToolCall{
					ID:   fmt.Sprintf("call-%d", m.calls),
					Name: parts[0],
					Args: parts[1],
				}
				ch <- provider.StreamChunk{
					Done:      true,
					ToolCalls: []provider.ToolCall{toolCall},
				}
				return
			}
		}

		// Otherwise just text
		ch <- provider.StreamChunk{
			Delta: response,
			Done:  true,
		}
	}()

	return ch, nil
}

func main() {
	fmt.Println("🤖 Verifying Agent Tool Capabilities...")

	// 1. Setup Mock Provider
	// Scenario: User asks for weather -> Agent calls tool -> Tool returns -> Agent answers
	mock := &MockProvider{
		responses: []string{
			`[TOOL:web_search|{"query": "golang 1.25 release date"}]`,
			"Based on the search results, Go 1.25 is expected in August 2025.",
		},
	}

	// 2. Setup Tools
	reg := tools.NewRegistry([]string{"web_search"}, true) // Auto approve
	// Mock the actual tool execution to avoid network calls?
	// For this test, let's use the real registry but check the calls.
	// Actually, system prompt requires real tools.
	tools.RegisterDefaults(reg, nil, nil)

	// 3. Create Agent
	agt := agent.New(mock, reg, "")

	// 4. Run Agent
	ctx := context.Background()
	events := make(chan agent.Event, 100)

	fmt.Println("\n🧪 Test Case 1: Simple Web Search")
	go agt.Send(ctx, "When is Go 1.25 coming out?", events)

	for evt := range events {
		switch evt.Type {
		case agent.EventThinking:
			fmt.Printf(" [Thinking] %s\n", evt.Text)
		case agent.EventToolCall:
			fmt.Printf(" [Tool Call] %s(%s)\n", evt.ToolName, evt.ToolArgs)
			if evt.ToolName == "web_search" && strings.Contains(evt.ToolArgs, "golang 1.25") {
				fmt.Println(" ✅ Verified: Agent correctly called web_search")
			}
		case agent.EventDelta:
			fmt.Print(evt.Text)
		case agent.EventDone:
			fmt.Println("\n [Done]")
			return
		case agent.EventError:
			fmt.Printf(" [Error] %s\n", evt.Error)
			os.Exit(1)
		}
	}
}
