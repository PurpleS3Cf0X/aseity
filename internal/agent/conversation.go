package agent

import "github.com/jeanpaul/aseity/internal/provider"

type Conversation struct {
	messages []provider.Message
}

func NewConversation() *Conversation {
	return &Conversation{}
}

func (c *Conversation) AddSystem(content string) {
	c.messages = append(c.messages, provider.Message{Role: provider.RoleSystem, Content: content})
}

func (c *Conversation) AddUser(content string) {
	c.messages = append(c.messages, provider.Message{Role: provider.RoleUser, Content: content})
}

func (c *Conversation) AddAssistant(content string, toolCalls []provider.ToolCall) {
	c.messages = append(c.messages, provider.Message{
		Role: provider.RoleAssistant, Content: content, ToolCalls: toolCalls,
	})
}

func (c *Conversation) AddToolResult(toolCallID, content string) {
	c.messages = append(c.messages, provider.Message{
		Role: provider.RoleTool, Content: content, ToolCallID: toolCallID,
	})
}

func (c *Conversation) Messages() []provider.Message {
	return c.messages
}

func (c *Conversation) Len() int {
	return len(c.messages)
}
