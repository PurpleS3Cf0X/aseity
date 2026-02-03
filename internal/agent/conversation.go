package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeanpaul/aseity/internal/provider"
)

type Conversation struct {
	messages     []provider.Message
	maxTokens    int // approximate context window limit
	totalTokens  int // running estimate
	sessionID    string
	sessionDir   string
}

func NewConversation() *Conversation {
	home, _ := os.UserHomeDir()
	sessionDir := filepath.Join(home, ".config", "aseity", "sessions")
	return &Conversation{
		maxTokens:  100000, // conservative default
		sessionID:  fmt.Sprintf("%d", time.Now().UnixNano()),
		sessionDir: sessionDir,
	}
}

func (c *Conversation) SetMaxTokens(max int) {
	if max > 0 {
		c.maxTokens = max
	}
}

func (c *Conversation) AddSystem(content string) {
	c.messages = append(c.messages, provider.Message{Role: provider.RoleSystem, Content: content})
	c.totalTokens += estimateTokens(content)
}

func (c *Conversation) AddUser(content string) {
	c.messages = append(c.messages, provider.Message{Role: provider.RoleUser, Content: content})
	c.totalTokens += estimateTokens(content)
	c.compactIfNeeded()
}

func (c *Conversation) AddAssistant(content string, toolCalls []provider.ToolCall) {
	c.messages = append(c.messages, provider.Message{
		Role: provider.RoleAssistant, Content: content, ToolCalls: toolCalls,
	})
	c.totalTokens += estimateTokens(content)
	for _, tc := range toolCalls {
		c.totalTokens += estimateTokens(tc.Args)
	}
}

func (c *Conversation) AddToolResult(toolCallID, content string) {
	// Truncate very large tool results
	if len(content) > 30000 {
		content = content[:30000] + "\n... [truncated]"
	}
	c.messages = append(c.messages, provider.Message{
		Role: provider.RoleTool, Content: content, ToolCallID: toolCallID,
	})
	c.totalTokens += estimateTokens(content)
	c.compactIfNeeded()
}

func (c *Conversation) Messages() []provider.Message {
	return c.messages
}

func (c *Conversation) Len() int {
	return len(c.messages)
}

func (c *Conversation) EstimatedTokens() int {
	return c.totalTokens
}

// compactIfNeeded removes old messages if we're approaching the context limit.
// Keeps: system prompt, last N user/assistant exchanges.
func (c *Conversation) compactIfNeeded() {
	if c.totalTokens < c.maxTokens*80/100 {
		return
	}
	c.Compact()
}

// Compact summarizes and trims the conversation to fit within context limits.
func (c *Conversation) Compact() {
	if len(c.messages) <= 4 {
		return
	}

	// Keep system message and recent messages
	var system *provider.Message
	start := 0
	if len(c.messages) > 0 && c.messages[0].Role == provider.RoleSystem {
		sys := c.messages[0]
		system = &sys
		start = 1
	}

	// Build a summary of older messages
	remaining := c.messages[start:]
	if len(remaining) <= 6 {
		return
	}

	// Keep only the last 6 messages (3 exchanges), summarize the rest
	cutoff := len(remaining) - 6
	var summary strings.Builder
	summary.WriteString("[Conversation summary]\n")
	for _, m := range remaining[:cutoff] {
		switch m.Role {
		case provider.RoleUser:
			summary.WriteString(fmt.Sprintf("User: %s\n", truncateText(m.Content, 100)))
		case provider.RoleAssistant:
			summary.WriteString(fmt.Sprintf("Assistant: %s\n", truncateText(m.Content, 100)))
		case provider.RoleTool:
			summary.WriteString(fmt.Sprintf("Tool result: %s\n", truncateText(m.Content, 50)))
		}
	}

	var newMsgs []provider.Message
	if system != nil {
		newMsgs = append(newMsgs, *system)
	}
	newMsgs = append(newMsgs, provider.Message{
		Role:    provider.RoleUser,
		Content: summary.String(),
	})
	newMsgs = append(newMsgs, provider.Message{
		Role:    provider.RoleAssistant,
		Content: "Understood. I have the conversation context.",
	})
	newMsgs = append(newMsgs, remaining[cutoff:]...)

	c.messages = newMsgs
	c.recalcTokens()
}

// Clear removes all non-system messages.
func (c *Conversation) Clear() {
	var newMsgs []provider.Message
	for _, m := range c.messages {
		if m.Role == provider.RoleSystem {
			newMsgs = append(newMsgs, m)
		}
	}
	c.messages = newMsgs
	c.recalcTokens()
}

func (c *Conversation) recalcTokens() {
	c.totalTokens = 0
	for _, m := range c.messages {
		c.totalTokens += estimateTokens(m.Content)
		for _, tc := range m.ToolCalls {
			c.totalTokens += estimateTokens(tc.Args)
		}
	}
}

// Save persists the conversation to disk.
func (c *Conversation) Save() (string, error) {
	if err := os.MkdirAll(c.sessionDir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(c.sessionDir, c.sessionID+".json")
	data, err := json.MarshalIndent(c.messages, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0644)
}

// Export writes the conversation to a file in human-readable format.
func (c *Conversation) Export(path string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Aseity Session %s\n\n", c.sessionID))
	for _, m := range c.messages {
		switch m.Role {
		case provider.RoleSystem:
			continue
		case provider.RoleUser:
			sb.WriteString("## User\n")
			sb.WriteString(m.Content + "\n\n")
		case provider.RoleAssistant:
			sb.WriteString("## Aseity\n")
			sb.WriteString(m.Content + "\n\n")
		case provider.RoleTool:
			sb.WriteString("## Tool Result\n")
			sb.WriteString(m.Content + "\n\n")
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// Load restores a conversation from a session file.
func LoadConversation(path string) (*Conversation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var msgs []provider.Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, err
	}
	conv := NewConversation()
	conv.messages = msgs
	conv.recalcTokens()
	return conv, nil
}

// estimateTokens gives a rough count (~4 chars per token).
func estimateTokens(s string) int {
	return len(s) / 4
}

func truncateText(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
