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
type confirmResponseMsg bool

type Model struct {
	width, height int
	viewport      viewport.Model
	textarea      textarea.Model
	spinner       spinner.Model
	messages      []chatMessage
	thinking      bool
	providerName  string
	modelName     string

	agent      *agent.Agent
	prov       provider.Provider
	toolReg    *tools.Registry
	eventCh    chan agent.Event
	ctx        context.Context
	cancelCtx  context.CancelFunc

	pendingConfirm *agent.Event
	chatContent    strings.Builder
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
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(Green)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(DimGreen)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(DarkGreen)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	vp := viewport.New(80, 20)

	ctx, cancel := context.WithCancel(context.Background())

	confirmFn := func(toolName, args string) bool {
		return true // overridden by TUI event loop
	}

	ag := agent.New(prov, toolReg, confirmFn)

	return Model{
		viewport:     vp,
		textarea:     ta,
		spinner:      sp,
		providerName: provName,
		modelName:    modelName,
		prov:         prov,
		toolReg:      toolReg,
		agent:        ag,
		ctx:          ctx,
		cancelCtx:    cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 3
		inputH := 5
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerH - inputH
		m.textarea.SetWidth(msg.Width - 4)
		m.rebuildView()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.cancelCtx()
			return m, tea.Quit
		case tea.KeyCtrlC:
			if m.thinking {
				m.thinking = false
				m.cancelCtx()
				m.ctx, m.cancelCtx = context.WithCancel(context.Background())
				m.agent = agent.New(m.prov, m.toolReg, func(string, string) bool { return true })
				return m, nil
			}
			m.cancelCtx()
			return m, tea.Quit
		case tea.KeyEnter:
			if msg.Alt {
				break // alt+enter = newline in textarea
			}
			text := strings.TrimSpace(m.textarea.Value())
			if text == "" {
				return m, nil
			}
			m.textarea.Reset()
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
		case agent.EventDelta:
			if len(m.messages) == 0 || m.messages[len(m.messages)-1].role != "assistant" {
				m.messages = append(m.messages, chatMessage{role: "assistant"})
			}
			m.messages[len(m.messages)-1].content += evt.Text

		case agent.EventToolCall:
			m.messages = append(m.messages, chatMessage{
				role:    "tool",
				content: fmt.Sprintf("  %s %s(%s)", ToolCallStyle.Render("●"), evt.ToolName, truncate(evt.ToolArgs, 60)),
			})

		case agent.EventToolResult:
			if evt.Result != "" {
				result := evt.Result
				if len(result) > 200 {
					result = result[:200] + "..."
				}
				m.messages = append(m.messages, chatMessage{
					role:    "tool_result",
					content: result,
				})
			}

		case agent.EventError:
			m.messages = append(m.messages, chatMessage{
				role:    "error",
				content: evt.Error,
			})
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
	if !m.thinking {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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

func (m *Model) rebuildView() {
	var sb strings.Builder
	for _, msg := range m.messages {
		switch msg.role {
		case "user":
			sb.WriteString(UserLabelStyle.Render("  You") + "\n")
			sb.WriteString(UserMsgStyle.Render("  "+msg.content) + "\n\n")
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
		case "error":
			sb.WriteString(ErrorStyle.Render("  Error: "+msg.content) + "\n\n")
		}
	}
	if m.thinking {
		sb.WriteString(SpinnerStyle.Render("  "+m.spinner.View()+" Thinking...") + "\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	// Header
	header := StatusProviderStyle.Render(" "+m.providerName) +
		StatusBarStyle.Render(" "+m.modelName+" ") +
		StatusBarStyle.Copy().Width(m.width-lipgloss.Width(m.providerName)-lipgloss.Width(m.modelName)-6).
			Render("aseity")
	separator := SeparatorStyle.Render(strings.Repeat("─", m.width))

	// Input
	inputStyle := InputBorderStyle
	if !m.thinking {
		inputStyle = InputActiveStyle
	}
	input := inputStyle.Width(m.width - 4).Render(m.textarea.View())

	return header + "\n" + separator + "\n" + m.viewport.View() + "\n" + input
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
