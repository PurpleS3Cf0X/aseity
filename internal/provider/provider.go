package provider

import "context"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"arguments"`
}

type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

// Usage tracks token consumption for a request
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type StreamChunk struct {
	Delta     string
	Thinking  string // Model's internal reasoning/chain-of-thought
	ToolCalls []ToolCall
	Done      bool
	Error     error
	Usage     *Usage // Token usage (populated in final chunk when Done=true)
}

type Provider interface {
	Chat(ctx context.Context, msgs []Message, tools []ToolDef) (<-chan StreamChunk, error)
	Name() string
	ModelName() string // Returns the model name for capability detection
	Models(ctx context.Context) ([]string, error)
}
