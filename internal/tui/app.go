package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

type agentEventMsg agent.Event

type Model struct {
	width, height int
	viewport      viewport.Model
	textarea      textarea.Model
	spinner       spinner.Model
	messages      []chatMessage
	thinking      bool
	showThinking  bool
	confirming    bool // waiting for user to approve/deny
	confirmEvt    *agent.Event
	providerName  string
	modelName     string

	agent   *agent.Agent
	prov    provider.Provider
	toolReg *tools.Registry
	eventCh chan agent.Event
	ctx     context.Context
	cancel  context.CancelFunc
}

type chatMessage struct {
	role    string
	content string
}

func NewModel(prov provider.Provider, toolReg *tools.Registry, provName, modelName string) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Esc to quit)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(White)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(DimGreen)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(DarkGreen)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	vp := viewport.New(80, 20)
	ctx, cancel := context.WithCancel(context.Background())
	ag := agent.New(prov, toolReg)

	return Model{
		viewport:     vp,
		textarea:     ta,
		spinner:      sp,
		showThinking: true,
		providerName: provName,
		modelName:    modelName,
		prov:         prov,
		toolReg:      toolReg,
		agent:        ag,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 3
		inputH := 6
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerH - inputH
		m.textarea.SetWidth(msg.Width - 4)
		m.rebuildView()

	case tea.KeyMsg:
		// Handle confirmation prompt
		if m.confirming {
			switch msg.String() {
			case "y", "Y", "enter":
				m.confirming = false
				m.agent.ConfirmCh <- true
				m.messages = append(m.messages, chatMessage{role: "confirm", content: "  Approved"})
				m.rebuildView()
				return m, m.waitForEvent()
			case "n", "N":
				m.confirming = false
				m.agent.ConfirmCh <- false
				m.messages = append(m.messages, chatMessage{role: "confirm_deny", content: "  Denied"})
				m.rebuildView()
				return m, m.waitForEvent()
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEsc:
			// Save session on exit
			m.agent.Conversation().Save()
			m.cancel()
			return m, tea.Quit
		case tea.KeyCtrlC:
			if m.thinking {
				m.thinking = false
				m.confirming = false
				m.cancel()
				m.ctx, m.cancel = context.WithCancel(context.Background())
				m.agent = agent.New(m.prov, m.toolReg)
				m.messages = append(m.messages, chatMessage{role: "system", content: "  Cancelled."})
				m.rebuildView()
				return m, nil
			}
			m.agent.Conversation().Save()
			m.cancel()
			return m, tea.Quit
		case tea.KeyCtrlT:
			m.showThinking = !m.showThinking
			m.rebuildView()
			return m, nil
		case tea.KeyEnter:
			if msg.Alt {
				break
			}
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			m.textarea.Reset()

			// Handle slash commands
			if strings.HasPrefix(text, "/") {
				return m.handleSlashCommand(text)
			}

			m.messages = append(m.messages, chatMessage{role: "user", content: text})
			m.thinking = true
			m.rebuildView()

			m.eventCh = make(chan agent.Event, 64)
			go m.agent.Send(m.ctx, text, m.eventCh)
			return m, m.waitForEvent()
		}

	case agentEventMsg:
		evt := agent.Event(msg)
		switch evt.Type {
		case agent.EventThinking:
			if len(m.messages) == 0 || m.messages[len(m.messages)-1].role != "thinking" {
				m.messages = append(m.messages, chatMessage{role: "thinking"})
			}
			m.messages[len(m.messages)-1].content += evt.Text

		case agent.EventDelta:
			if len(m.messages) == 0 || m.messages[len(m.messages)-1].role != "assistant" {
				m.messages = append(m.messages, chatMessage{role: "assistant"})
			}
			m.messages[len(m.messages)-1].content += evt.Text

		case agent.EventToolCall:
			m.messages = append(m.messages, chatMessage{
				role:    "tool",
				content: formatToolCallDisplay(evt.ToolName, evt.ToolArgs),
			})

		case agent.EventConfirmRequest:
			m.confirming = true
			m.confirmEvt = &evt
			m.messages = append(m.messages, chatMessage{
				role:    "confirm_prompt",
				content: fmt.Sprintf("  Allow %s? [y/n]", evt.ToolName),
			})
			m.rebuildView()
			return m, nil // stop consuming events until user responds

		case agent.EventToolResult:
			if evt.Result != "" {
				result := evt.Result
				if len(result) > 500 {
					result = result[:500] + "\n  ... (truncated)"
				}
				m.messages = append(m.messages, chatMessage{role: "tool_result", content: result})
			}
			if evt.Error != "" {
				m.messages = append(m.messages, chatMessage{role: "error", content: evt.Error})
			}

		case agent.EventError:
			m.messages = append(m.messages, chatMessage{role: "error", content: evt.Error})
			m.thinking = false
			m.rebuildView()
			return m, nil

		case agent.EventDone:
			m.thinking = false
			m.rebuildView()
			return m, nil
		}

		m.rebuildView()
		if !evt.Done {
			return m, m.waitForEvent()
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	if !m.thinking && !m.confirming {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleSlashCommand processes /commands entered by the user.
func (m *Model) handleSlashCommand(text string) (Model, tea.Cmd) {
	parts := strings.Fields(text)
	cmd := parts[0]

	switch cmd {
	case "/help":
		m.messages = append(m.messages, chatMessage{
			role: "system",
			content: `  Available commands:
    /help     — show this help
    /clear    — clear conversation history
    /compact  — compress conversation to save context
    /save     — export conversation to markdown file
    /tokens   — show estimated token usage
    /quit     — exit aseity`,
		})

	case "/clear":
		m.agent.Conversation().Clear()
		m.messages = nil
		m.messages = append(m.messages, chatMessage{role: "system", content: "  Conversation cleared."})

	case "/compact":
		before := m.agent.Conversation().EstimatedTokens()
		m.agent.Conversation().Compact()
		after := m.agent.Conversation().EstimatedTokens()
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: fmt.Sprintf("  Compacted: ~%dk -> ~%dk tokens", before/1000, after/1000),
		})

	case "/save":
		path := "aseity-session.md"
		if len(parts) > 1 {
			path = parts[1]
		}
		if err := m.agent.Conversation().Export(path); err != nil {
			m.messages = append(m.messages, chatMessage{role: "error", content: err.Error()})
		} else {
			m.messages = append(m.messages, chatMessage{role: "system", content: "  Saved to " + path})
		}

	case "/tokens":
		tokens := m.agent.Conversation().EstimatedTokens()
		msgCount := m.agent.Conversation().Len()
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: fmt.Sprintf("  ~%dk tokens, %d messages", tokens/1000, msgCount),
		})

	case "/quit":
		m.agent.Conversation().Save()
		m.cancel()
		return *m, tea.Quit

	default:
		m.messages = append(m.messages, chatMessage{
			role:    "error",
			content: fmt.Sprintf("Unknown command: %s (type /help for available commands)", cmd),
		})
	}

	m.rebuildView()
	return *m, nil
}

func (m *Model) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-m.eventCh
		if !ok {
			return agentEventMsg(agent.Event{Type: agent.EventDone, Done: true})
		}
		return agentEventMsg(evt)
	}
}

func formatToolCallDisplay(name, args string) string {
	icon := ToolCallStyle.Render("●")
	switch name {
	case "bash":
		return fmt.Sprintf("  %s %s\n  %s",
			icon,
			ToolLabelStyle.Render("bash"),
			CommandStyle.Render("$ "+args),
		)
	case "file_read":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("read"), args)
	case "file_write":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("write"), args)
	case "file_search":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("search"), args)
	case "web_search":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("web search"), args)
	case "web_fetch":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("fetch"), args)
	case "spawn_agent":
		return fmt.Sprintf("  %s %s %s", icon, ToolLabelStyle.Render("spawn agent"), truncate(args, 60))
	case "list_agents":
		return fmt.Sprintf("  %s %s", icon, ToolLabelStyle.Render("list agents"))
	default:
		return fmt.Sprintf("  %s %s(%s)", icon, name, truncate(args, 60))
	}
}

func (m *Model) rebuildView() {
	var sb strings.Builder
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			sb.WriteString(UserLabelStyle.Render("  You") + "\n")
			sb.WriteString(UserMsgStyle.Render("  "+msg.content) + "\n\n")
		case "thinking":
			if m.showThinking && msg.content != "" {
				sb.WriteString(ThinkingLabelStyle.Render("  ▸ Thinking") + "\n")
				lines := strings.Split(msg.content, "\n")
				maxLines := 20
				if len(lines) > maxLines {
					for _, line := range lines[:maxLines] {
						sb.WriteString(ThinkingStyle.Render("    "+line) + "\n")
					}
					sb.WriteString(ThinkingStyle.Render(fmt.Sprintf("    ... (%d more lines)", len(lines)-maxLines)) + "\n")
				} else {
					for _, line := range lines {
						sb.WriteString(ThinkingStyle.Render("    "+line) + "\n")
					}
				}
				sb.WriteString("\n")
			} else if !m.showThinking && msg.content != "" {
				lines := strings.Count(msg.content, "\n") + 1
				sb.WriteString(ThinkingLabelStyle.Render(fmt.Sprintf("  ▸ Thinking (%d lines, Ctrl+T to expand)", lines)) + "\n\n")
			}
		case "assistant":
			sb.WriteString(AssistantLabelStyle.Render("  Aseity") + "\n")
			for _, line := range strings.Split(msg.content, "\n") {
				sb.WriteString(AssistantMsgStyle.Render("  "+line) + "\n")
			}
			sb.WriteString("\n")
		case "tool":
			sb.WriteString(msg.content + "\n")
		case "tool_result":
			for _, line := range strings.Split(msg.content, "\n") {
				sb.WriteString(ToolResultStyle.Render("    "+line) + "\n")
			}
			sb.WriteString("\n")
		case "confirm_prompt":
			sb.WriteString(ConfirmStyle.Render(msg.content) + "\n")
		case "confirm":
			sb.WriteString(BannerStyle.Render(msg.content+" ✓") + "\n\n")
		case "confirm_deny":
			sb.WriteString(ErrorStyle.Render(msg.content+" ✗") + "\n\n")
		case "error":
			sb.WriteString(ErrorStyle.Render("  Error: "+msg.content) + "\n\n")
		case "system":
			sb.WriteString(SystemMsgStyle.Render(msg.content) + "\n\n")
		case "subagent":
			sb.WriteString(SubAgentStyle.Render("  "+msg.content) + "\n")
		}
	}
	if m.thinking && !m.confirming {
		sb.WriteString(SpinnerStyle.Render("  "+m.spinner.View()+" Thinking...") + "\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	thinkLabel := ""
	if m.showThinking {
		thinkLabel = " [thinking:on]"
	}

	// Token counter
	tokenInfo := ""
	if m.agent != nil {
		tokens := m.agent.Conversation().EstimatedTokens()
		if tokens > 0 {
			tokenInfo = fmt.Sprintf(" ~%dk", tokens/1000)
		}
	}

	header := StatusProviderStyle.Render(" "+m.providerName) +
		StatusBarStyle.Render(" "+m.modelName+" ") +
		StatusBarStyle.Copy().Width(m.width-lipgloss.Width(m.providerName)-lipgloss.Width(m.modelName)-6-len(thinkLabel)-len(tokenInfo)).
			Render("aseity") +
		TokenStyle.Render(tokenInfo) +
		HelpStyle.Render(thinkLabel)
	separator := SeparatorStyle.Render(strings.Repeat("─", m.width))

	inputStyle := InputBorderStyle
	if m.confirming {
		inputStyle = ConfirmInputStyle
	} else if !m.thinking {
		inputStyle = InputActiveStyle
	}
	input := inputStyle.Width(m.width - 4).Render(m.textarea.View())

	help := ""
	if m.confirming {
		help = ConfirmStyle.Render("  y:approve  n:deny  Ctrl+C:cancel")
	} else {
		help = HelpStyle.Render("  Enter:send  Alt+Enter:newline  Ctrl+T:thinking  Ctrl+C:cancel  Esc:quit  /help")
	}

	return header + "\n" + separator + "\n" + m.viewport.View() + "\n" + input + "\n" + help
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
