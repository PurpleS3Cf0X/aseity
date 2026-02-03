package tui

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/jeanpaul/aseity/internal/agent"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// Custom spinner frames for different states
var (
	// Thinking spinner — Braille dots animation (smooth)
	ThinkingSpinner = spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		FPS:    time.Second / 12,
	}

	// Tool execution spinner — Bouncing dots
	ToolSpinner = spinner.Spinner{
		Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		FPS:    time.Second / 10,
	}

	// Processing spinner — Pulse effect
	ProcessingSpinner = spinner.Spinner{
		Frames: []string{"●", "◐", "◑", "◒", "◓", "◔", "◕", "◖", "◗", "◘"},
		FPS:    time.Second / 8,
	}

	// Waiting spinner — Subtle blink
	WaitingSpinner = spinner.Spinner{
		Frames: []string{"◇", "◈", "◆", "◈"},
		FPS:    time.Second / 4,
	}

	// Network spinner — Globe animation
	NetworkSpinner = spinner.Spinner{
		Frames: []string{"🌍", "🌎", "🌏"},
		FPS:    time.Second / 3,
	}

	// Fun/Cool Retro Spinners (Strictly ASCII/Block + Colors)
	FunSpinners = []struct {
		Spinner spinner.Spinner
		Color   lipgloss.Style
	}{
		// 1. KITT (Knight Rider) - Amber (Matches new theme)
		{
			Spinner: spinner.Spinner{Frames: []string{"[=    ]", "[==   ]", "[===  ]", "[ ====]", "[  ===]", "[   ==]", "[    =]", "[   ==]", "[  ===]", "[ ====]", "[===  ]", "[==   ]"}, FPS: time.Second / 12},
			Color:   lipgloss.NewStyle().Foreground(Amber),
		},
		// 2. Retro Prompt - Green
		{
			Spinner: spinner.Spinner{Frames: []string{">_", "> "}, FPS: time.Second / 2},
			Color:   lipgloss.NewStyle().Foreground(Green),
		},
		// 3. Radar scan - Cyan
		{
			Spinner: spinner.Spinner{Frames: []string{"(     )", "( =   )", "( ==  )", "( === )", "(  ===)", "(   ==)", "(    =)", "(     )"}, FPS: time.Second / 12},
			Color:   lipgloss.NewStyle().Foreground(Cyan),
		},
		// 4. Loading Bar - Magenta
		{
			Spinner: spinner.Spinner{Frames: []string{"[    ]", "[=   ]", "[==  ]", "[=== ]", "[====]", "[ ===]", "[  ==]", "[   =]"}, FPS: time.Second / 10},
			Color:   lipgloss.NewStyle().Foreground(Magenta),
		},
		// 5. Blinking Block - White
		{
			Spinner: spinner.Spinner{Frames: []string{"█", " "}, FPS: time.Second / 2},
			Color:   lipgloss.NewStyle().Foreground(White),
		},
		// 6. Classic Pipe - Yellow
		{
			Spinner: spinner.Spinner{Frames: []string{"|", "/", "-", "\\"}, FPS: time.Second / 10},
			Color:   lipgloss.NewStyle().Foreground(SoftYellow),
		},
		// 7. Binary - Bright Green
		{
			Spinner: spinner.Spinner{Frames: []string{"101010", "010101"}, FPS: time.Second / 4},
			Color:   lipgloss.NewStyle().Foreground(BrightGreen),
		},
		// 8. Ping Pong - Purple
		{
			Spinner: spinner.Spinner{Frames: []string{"|  .  |", "| .   |", "|.    |", "| .   |", "|  .  |", "|   . |", "|    .|", "|   . |"}, FPS: time.Second / 8},
			Color:   lipgloss.NewStyle().Foreground(Purple),
		},
		// 9. Hash Noise - Blue
		{
			Spinner: spinner.Spinner{Frames: []string{"#", "##", "###", "####", "   #", "  ##", " ###", "####"}, FPS: time.Second / 8},
			Color:   lipgloss.NewStyle().Foreground(Blue),
		},
		// 10. Starfield - LightGray
		{
			Spinner: spinner.Spinner{Frames: []string{"+", "x", "*", "x"}, FPS: time.Second / 6},
			Color:   lipgloss.NewStyle().Foreground(LightGray),
		},
	}
)

// Tool icons for visual distinction
var toolIcons = map[string]string{
	"bash":        "CMD",
	"file_read":   "READ",
	"file_write":  "EDIT",
	"file_search": "FIND",
	"web_search":  "WEB",
	"web_fetch":   "GET",
	"spawn_agent": "BOT",
	"list_agents": "LIST",
}

type agentEventMsg agent.Event

type SpinnerState int

const (
	SpinnerIdle SpinnerState = iota
	SpinnerThinking
	SpinnerTool
	SpinnerNetwork
)

type Model struct {
	width, height int
	viewport      viewport.Model
	textarea      textarea.Model
	spinner       spinner.Model
	spinnerState  SpinnerState
	messages      []chatMessage
	thinking      bool
	showThinking  bool
	confirming    bool // waiting for user to approve/deny
	confirmEvt    *agent.Event
	providerName  string
	modelName     string
	currentTool   string // track which tool is running for animation

	agent                  *agent.Agent
	prov                   provider.Provider
	toolReg                *tools.Registry
	eventCh                chan agent.Event
	ctx                    context.Context
	cancel                 context.CancelFunc
	renderer               *glamour.TermRenderer
	frame                  int // animation frame counter
	inputRequest           bool
	currentThinkingSpinner spinner.Spinner
	currentThinkingStyle   lipgloss.Style
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
	sp.Spinner = ThinkingSpinner
	sp.Style = SpinnerThinkingStyle

	vp := viewport.New(80, 20)
	ctx, cancel := context.WithCancel(context.Background())
	ag := agent.New(prov, toolReg)

	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	m := Model{
		viewport:               vp,
		textarea:               ta,
		spinner:                sp,
		showThinking:           true,
		providerName:           provName,
		modelName:              modelName,
		prov:                   prov,
		toolReg:                toolReg,
		agent:                  ag,
		ctx:                    ctx,
		cancel:                 cancel,
		renderer:               r,
		currentThinkingSpinner: ThinkingSpinner,
		currentThinkingStyle:   SpinnerThinkingStyle,
	}

	// Add welcome message
	m.messages = append(m.messages, chatMessage{
		role:    "welcome",
		content: fmt.Sprintf("Welcome to Aseity! You're connected to %s.\n\nI can help you with coding tasks, run commands, search the web, and manage files.\n\nTry asking me to:\n  • Explain some code\n  • Run a git command\n  • Search for documentation\n  • Create or edit a file", modelName),
	})

	return m
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

		// Scroll viewport with Ctrl+Up/Down or PgUp/PgDown (when not typing)
		switch msg.Type {
		case tea.KeyPgUp:
			m.viewport.HalfViewUp()
			return m, nil
		case tea.KeyPgDown:
			m.viewport.HalfViewDown()
			return m, nil
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

			// Handle Tool Input Request
			if m.inputRequest {
				m.inputRequest = false
				m.textarea.Reset()
				m.messages = append(m.messages, chatMessage{role: "system", content: "  > [Input Sent]"})
				m.rebuildView()

				// Send to agent
				m.agent.InputCh <- text

				// Resume event loop
				return m, m.waitForEvent()
			}

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

			// Pick a fun random animation for this turn!
			choice := FunSpinners[rand.Intn(len(FunSpinners))]
			m.currentThinkingSpinner = choice.Spinner
			m.currentThinkingStyle = choice.Color
			m.spinner.Spinner = choice.Spinner
			m.spinner.Style = choice.Color

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
			m.currentTool = evt.ToolName
			// Switch spinner based on tool type
			m.setSpinnerForTool(evt.ToolName)
			m.messages = append(m.messages, chatMessage{
				role:    "tool",
				content: formatToolCallDisplay(evt.ToolName, evt.ToolArgs),
			})

		case agent.EventToolOutput:
			// Append output to the last message if it's a tool execution display
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].role == "tool" {
				m.messages[len(m.messages)-1].content += evt.Text
			} else {
				// Should not happen, but safe fallback
				m.messages = append(m.messages, chatMessage{role: "tool", content: evt.Text})
			}

		case agent.EventConfirmRequest:
			m.confirming = true
			m.confirmEvt = &evt
			m.messages = append(m.messages, chatMessage{
				role:    "confirm_prompt",
				content: fmt.Sprintf("  Allow %s? [y/n]", evt.ToolName),
			})
			m.rebuildView()
			return m, nil // stop consuming events until user responds

		case agent.EventInputRequest:
			m.inputRequest = true
			m.spinnerState = SpinnerTool // Keep spinning
			m.messages = append(m.messages, chatMessage{
				role:    "system",
				content: "  Input required by tool (e.g. password). Type above and press Enter.",
			})
			m.rebuildView()
			// We continue to consume events? No, tool is blocked.
			return m, nil

		case agent.EventToolResult:
			// Reset spinner back to thinking state
			m.resetSpinner()
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
		m.frame++
		m.spinner, cmd = m.spinner.Update(msg)
		// Force view rebuild for animation if on welcome screen (first message)
		if len(m.messages) > 0 && m.messages[0].role == "welcome" {
			m.rebuildView()
		}
		cmds = append(cmds, cmd)

	case tea.MouseMsg:
		// Handle mouse scroll
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.LineUp(3)
		case tea.MouseButtonWheelDown:
			m.viewport.LineDown(3)
		}
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
    /help        — show this help
    /clear       — clear conversation history
    /compact     — compress conversation to save context
    /save [path] — export conversation to markdown file
    /tokens      — show estimated token usage
    /model       — show current model
    /quit        — exit aseity

  Keyboard shortcuts:
    Enter        — send message
    Alt+Enter    — new line
    Ctrl+T       — toggle thinking visibility
    Ctrl+C       — cancel/quit
    PgUp/PgDown  — scroll conversation
    Esc          — quit`,
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

	case "/model":
		m.messages = append(m.messages, chatMessage{
			role:    "system",
			content: fmt.Sprintf("  Provider: %s\n  Model: %s", m.providerName, m.modelName),
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

// setSpinnerForTool changes the spinner animation based on tool type
func (m *Model) setSpinnerForTool(toolName string) {
	switch toolName {
	case "web_search", "web_fetch":
		m.spinner.Spinner = NetworkSpinner
		m.spinner.Style = WebIconStyle
		m.spinnerState = SpinnerNetwork
	case "bash":
		m.spinner.Spinner = ToolSpinner
		m.spinner.Style = BashIconStyle
		m.spinnerState = SpinnerTool
	case "file_read", "file_write", "file_search":
		m.spinner.Spinner = ProcessingSpinner
		m.spinner.Style = FileIconStyle
		m.spinnerState = SpinnerTool
	case "spawn_agent", "list_agents":
		m.spinner.Spinner = ThinkingSpinner
		m.spinner.Style = AgentIconStyle
		m.spinnerState = SpinnerTool
	default:
		m.spinner.Spinner = ToolSpinner
		m.spinner.Style = SpinnerToolStyle
		m.spinnerState = SpinnerTool
	}
}

// resetSpinner returns to thinking state
func (m *Model) resetSpinner() {
	m.spinner.Spinner = m.currentThinkingSpinner
	// We need to store the current thinking color?
	// For simplicity, we just won't override it here if it's already set by the random picker.
	// But setSpinnerForTool overrides it.
	// Let's just default to Purple if we don't remember, OR we should store currentThinkingStyle.
	// Actually, let's just pick a random one again if we lost it, or better:
	// We'll trust that m.spinnder.Style was set when we picked the spinner.
	// But wait, setSpinnerForTool changes m.spinner.Style.
	// So we DO need to restore it.

	// Quick hack: Just reset to the default ThinkingStyle if we don't save the custom one.
	// User wants varied colors.
	// Let's add currentThinkingStyle to Model.
	m.spinner.Style = m.currentThinkingStyle
	m.spinnerState = SpinnerThinking
	m.currentTool = ""
}

// getAnimatedStatus returns a contextual status message
func (m *Model) getAnimatedStatus() string {
	if m.currentTool != "" {
		switch m.currentTool {
		case "bash":
			return "Executing command..."
		case "file_read":
			return "Reading file..."
		case "file_write":
			return "Writing file..."
		case "file_search":
			return "Searching files..."
		case "web_search":
			return "Searching the web..."
		case "web_fetch":
			return "Fetching page..."
		case "spawn_agent":
			return "Spawning sub-agent..."
		case "list_agents":
			return "Listing agents..."
		default:
			return "Running " + m.currentTool + "..."
		}
	}
	return "Thinking..."
}

func formatToolCallDisplay(name, args string) string {
	icon := toolIcons[name]
	if icon == "" {
		icon = "●"
	}

	switch name {
	case "bash":
		return fmt.Sprintf("  %s %s\n  %s",
			BashIconStyle.Render(icon),
			ToolLabelStyle.Render("bash"),
			CommandStyle.Render("$ "+args),
		)
	case "file_read":
		return fmt.Sprintf("  %s %s %s",
			FileIconStyle.Render(icon),
			ToolLabelStyle.Render("read"),
			InfoStyle.Render(args),
		)
	case "file_write":
		return fmt.Sprintf("  %s %s %s",
			FileIconStyle.Render(icon),
			ToolLabelStyle.Render("write"),
			InfoStyle.Render(args),
		)
	case "file_search":
		return fmt.Sprintf("  %s %s %s",
			FileIconStyle.Render(icon),
			ToolLabelStyle.Render("search"),
			InfoStyle.Render(args),
		)
	case "web_search":
		return fmt.Sprintf("  %s %s %s",
			WebIconStyle.Render(icon),
			ToolLabelStyle.Render("web search"),
			InfoStyle.Render(args),
		)
	case "web_fetch":
		return fmt.Sprintf("  %s %s %s",
			WebIconStyle.Render(icon),
			ToolLabelStyle.Render("fetch"),
			InfoStyle.Render(truncate(args, 60)),
		)
	case "spawn_agent":
		return fmt.Sprintf("  %s %s %s",
			AgentIconStyle.Render(icon),
			ToolLabelStyle.Render("spawn agent"),
			InfoStyle.Render(truncate(args, 60)),
		)
	case "list_agents":
		return fmt.Sprintf("  %s %s",
			AgentIconStyle.Render(icon),
			ToolLabelStyle.Render("list agents"),
		)
	default:
		return fmt.Sprintf("  %s %s %s",
			ToolCallStyle.Render("●"),
			ToolLabelStyle.Render(name),
			InfoStyle.Render(truncate(args, 60)),
		)
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
				sb.WriteString(ThinkingLabelStyle.Render("  💭 Reasoning") + "\n")
				lines := strings.Split(msg.content, "\n")
				maxLines := 20
				if len(lines) > maxLines {
					for _, line := range lines[:maxLines] {
						sb.WriteString(ThinkingStyle.Render("    │ "+line) + "\n")
					}
					sb.WriteString(ThinkingStyle.Render(fmt.Sprintf("    └─ ... (%d more lines)", len(lines)-maxLines)) + "\n")
				} else {
					for i, line := range lines {
						prefix := "    │ "
						if i == len(lines)-1 {
							prefix = "    └─"
						}
						sb.WriteString(ThinkingStyle.Render(prefix+line) + "\n")
					}
				}
				sb.WriteString("\n")
			} else if !m.showThinking && msg.content != "" {
				lines := strings.Count(msg.content, "\n") + 1
				sb.WriteString(ThinkingLabelStyle.Render(fmt.Sprintf("  💭 Reasoning (%d lines) ", lines)) + HelpStyle.Render("[Ctrl+T to expand]") + "\n\n")
			}
		case "assistant":
			sb.WriteString(AssistantLabelStyle.Render("  Aseity") + "\n")
			// Render markdown with glamour
			rendered, err := m.renderer.Render(msg.content)
			if err != nil {
				// Fallback to plain text
				for _, line := range strings.Split(msg.content, "\n") {
					sb.WriteString(AssistantMsgStyle.Render("  "+line) + "\n")
				}
			} else {
				// Indent the rendered markdown slightly
				lines := strings.Split(rendered, "\n")
				for _, line := range lines {
					if line != "" {
						sb.WriteString("  " + line + "\n")
					}
				}
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
			sb.WriteString(WarningStyle.Render("  ⚠ ") + ConfirmStyle.Render(msg.content) + "\n")
		case "confirm":
			sb.WriteString(SuccessStyle.Render("  ✓"+msg.content) + "\n\n")
		case "confirm_deny":
			sb.WriteString(ErrorStyle.Render("  ✗"+msg.content) + "\n\n")
		case "error":
			sb.WriteString(ErrorStyle.Render("  ✗ Error: "+msg.content) + "\n\n")
		case "system":
			sb.WriteString(SystemMsgStyle.Render("  ℹ"+msg.content) + "\n\n")
		case "subagent":
			sb.WriteString(SubAgentStyle.Render("  "+msg.content) + "\n")
		case "welcome":
			// Animated Banner
			banner := AnimatedBanner(m.frame)

			// Glowing Border Effect
			borderColor := Green
			if m.frame%20 > 10 {
				borderColor = BrightGreen
			}

			logo := LogoBoxStyle.Copy().
				BorderForeground(borderColor).
				Render(banner)

			text := WelcomeTextStyle.Render(fmt.Sprintf("\nWelcome to Aseity! Connected to %s", m.modelName))

			// Center everything roughly based on width (simplified centering)
			// For a true center we'd need to measure widths, but left-align with padding looks good too.

			sb.WriteString(logo + "\n")
			sb.WriteString(text + "\n\n")

			// Intro text content
			lines := strings.Split(msg.content, "\n")
			// Skip first line of original content as we rendered it custom
			if len(lines) > 0 {
				for _, line := range lines[1:] {
					sb.WriteString(AssistantMsgStyle.Render("  "+line) + "\n")
				}
			}
			sb.WriteString("\n")
		}
	}
	if m.thinking && !m.confirming {
		statusText := m.getAnimatedStatus()
		spinnerView := m.spinner.View()
		switch m.spinnerState {
		case SpinnerThinking:
			sb.WriteString(SpinnerThinkingStyle.Render("  "+spinnerView+" ") + ThinkingLabelStyle.Render(statusText) + "\n")
		case SpinnerTool:
			sb.WriteString(SpinnerToolStyle.Render("  "+spinnerView+" ") + ToolLabelStyle.Render(statusText) + "\n")
		case SpinnerNetwork:
			sb.WriteString(WebIconStyle.Render("  "+spinnerView+" ") + InfoStyle.Render(statusText) + "\n")
		default:
			sb.WriteString(SpinnerStyle.Render("  "+spinnerView+" ") + statusText + "\n")
		}
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

	keyStyle := lipgloss.NewStyle().Foreground(MintGreen)
	sepStyle := lipgloss.NewStyle().Foreground(DimGreen)

	help := ""
	if m.confirming {
		help = "  " +
			SuccessStyle.Render("y") + sepStyle.Render(":approve  ") +
			ErrorStyle.Render("n") + sepStyle.Render(":deny  ") +
			WarningStyle.Render("Ctrl+C") + sepStyle.Render(":cancel")
	} else {
		help = "  " +
			keyStyle.Render("Enter") + sepStyle.Render(":send  ") +
			keyStyle.Render("Alt+Enter") + sepStyle.Render(":newline  ") +
			keyStyle.Render("Ctrl+T") + sepStyle.Render(":thinking  ") +
			keyStyle.Render("Ctrl+C") + sepStyle.Render(":cancel  ") +
			keyStyle.Render("Esc") + sepStyle.Render(":quit  ") +
			keyStyle.Render("/help")
	}

	return header + "\n" + separator + "\n" + m.viewport.View() + "\n" + input + "\n" + help
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
