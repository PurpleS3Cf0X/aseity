package tui

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
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
	menu                   MenuModel
}

type chatMessage struct {
	role    string
	content string
}

func NewModel(prov provider.Provider, toolReg *tools.Registry, provName, modelName string, conversation *agent.Conversation, qualityGate bool) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(1) // Minimal height
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(White)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(DimGreen)
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(DarkGreen)
	ta.ShowLineNumbers = false

	sp := spinner.New()
	sp.Spinner = ThinkingSpinner
	sp.Style = SpinnerThinkingStyle

	vp := viewport.New(80, 20)
	ctx, cancel := context.WithCancel(context.Background())
	var ag *agent.Agent

	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	m := Model{
		viewport:     vp,
		textarea:     ta,
		spinner:      sp,
		showThinking: true,
		providerName: provName,
		modelName:    modelName,
		prov:         prov,
		toolReg:      toolReg,

		ctx:                    ctx,
		cancel:                 cancel,
		renderer:               r,
		currentThinkingSpinner: ThinkingSpinner,
		currentThinkingStyle:   SpinnerThinkingStyle,
		menu:                   NewMenuModel(), // Init menu directly
	}

	// Ensure viewport handles mouse events
	m.viewport.MouseWheelEnabled = true

	sysPrompt := agent.BuildSystemPrompt()

	if conversation != nil {
		ag = agent.NewWithConversation(prov, toolReg, conversation)
		// Rehydrate messages from conversation
		for _, msg := range conversation.Messages() {
			switch msg.Role {
			case provider.RoleUser:
				m.messages = append(m.messages, chatMessage{role: "user", content: msg.Content})
			case provider.RoleAssistant:
				m.messages = append(m.messages, chatMessage{role: "assistant", content: msg.Content})
				// TODO: Handle tool calls display if needed from history, strictly strictly rehydrating display is hard if we don't store ToolUse details in chatMessage properly.
				// For now, simpler rehydration is acceptable.
			case provider.RoleTool:
				m.messages = append(m.messages, chatMessage{role: "tool_result", content: msg.Content})
			}
		}
		m.messages = append(m.messages, chatMessage{role: "system", content: "  Restored session."})
	} else {
		ag = agent.New(prov, toolReg, sysPrompt)
		// Add welcome message
		m.messages = append(m.messages, chatMessage{
			role:    "welcome",
			content: fmt.Sprintf("Welcome to Aseity! You're connected to %s.\n\nI can help you with coding tasks, run commands, search the web, and manage files.\n\nTry asking me to:\n  • Explain some code\n  • Run a git command\n  • Search for documentation\n  • Create or edit a file", modelName),
		})
	}
	ag.QualityGateEnabled = qualityGate
	m.agent = ag

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		tea.EnableMouseCellMotion, // Enable Mouse Support
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 8 // Estimated new header height
		inputH := 3  // Minimal input
		menuH := 0
		if m.menu.active {
			menuH = 16 // 14 for list + 2 for borders/padding
		}
		m.viewport.Width = msg.Width - 4 // Account for border (2) and padding (2)
		m.viewport.Height = msg.Height - headerH - inputH - menuH
		m.textarea.SetWidth(msg.Width - 6) // Account for input box border/padding (4) + prompt (2)
		m.rebuildView()

	case tea.KeyMsg:
		// Menu Handling
		if m.menu.active {
			var cmd tea.Cmd
			m.menu, cmd = m.menu.Update(msg)

			// Handle selection
			if msg.String() == "enter" && m.menu.active {
				selectedItem := m.menu.list.SelectedItem()
				if selectedItem != nil {
					cmdStr := selectedItem.(item).Title()
					m.menu.active = false
					if cmdStr == "/quit" {
						return m, tea.Quit
					}
					m.textarea.SetValue(cmdStr)
					m.textarea.Focus()
				} else {
					// No selection (custom slash command?), use what was typed
					typed := m.menu.list.FilterValue()
					m.menu.active = false
					m.textarea.SetValue("/" + typed)
					m.textarea.Focus()
				}
			}
			// Handle Esc to exit menu but keep typed text
			if msg.String() == "esc" && m.menu.active {
				typed := m.menu.list.FilterValue()
				m.menu.active = false
				m.textarea.SetValue("/" + typed)
				m.textarea.Focus()

				// Restore viewport height on Escape
				headerH := 8
				inputH := 3
				// menuH is 0
				m.viewport.Height = m.height - headerH - inputH
				m.rebuildView()

				return m, nil
			}
			return m, cmd
		}

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

		// Trigger menu on '/' if input is empty
		if msg.String() == "/" && m.textarea.Value() == "" && !m.thinking && !m.confirming {
			m.menu.active = true
			// Reset state to ensure selection is at top and filter is clear
			m.menu.list.ResetSelected()
			m.menu.list.ResetFilter()

			// Adjust viewport height for menu
			headerH := 8
			inputH := 3
			menuH := 16
			m.viewport.Height = m.height - headerH - inputH - menuH
			m.rebuildView()

			// Forward the '/' to the menu list to activate filtering mode immediately
			var cmd tea.Cmd
			m.menu, cmd = m.menu.Update(msg)
			return m, cmd
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
			// If agent is busy, cancel the operation
			if m.thinking || m.confirming || m.currentTool != "" {
				m.thinking = false
				m.confirming = false
				m.currentTool = ""
				m.cancel()
				m.ctx, m.cancel = context.WithCancel(context.Background())
				// Preserve conversation history!
				oldConv := m.agent.Conversation()
				m.agent = agent.NewWithConversation(m.prov, m.toolReg, oldConv)
				m.messages = append(m.messages, chatMessage{role: "system", content: "  ⚠ Operation cancelled by user"})
				m.rebuildView()
				return m, nil
			}
			// Otherwise, quit normally
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

			// If empty and menu not active, maybe do nothing?
			// "Enter" on empty usually does nothing.

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

			// Block sending if agent is busy
			if m.thinking || m.confirming {
				// Don't spam messages - only show if user actually tried to send
				if text != "" {
					m.messages = append(m.messages, chatMessage{role: "system", content: "  ⏸ Agent is busy... (Ctrl+C to cancel)"})
					m.rebuildView()
				}
				return m, nil
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

	case tea.MouseMsg:
		// Explicitly handle scroll wheel to ensure it works
		if msg.Action == tea.MouseActionPress || msg.Action == tea.MouseActionMotion {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.viewport.LineUp(3)
			case tea.MouseButtonWheelDown:
				m.viewport.LineDown(3)
			default:
				// For clicks or motion, let viewport handle it
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

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

	//	case tea.MouseMsg:
	// Removed duplicate manual handling in favor of viewport.Update
	default:
		// No-op
	}

	var cmd tea.Cmd
	// Input Handling - Allow typing even when thinking, just block sending
	// Only block if menu is active
	if !m.menu.active {
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
    /status      — run git status
    /diff [full] — run git diff --stat (or full)
    /commit <m>  — run git commit -m <m>
    /quit        — exit aseity

  Keyboard shortcuts:
    Enter        — send message (blocks if agent is busy)
    Alt+Enter    — new line
    Ctrl+T       — toggle thinking visibility
    Ctrl+C       — cancel current operation / quit
    PgUp/PgDown  — scroll conversation
    Esc          — quit

  Tips:
    • You can type while the agent is working
    • Press Ctrl+C to interrupt long operations
    • Messages are queued if agent is busy`,
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

	case "/status":
		out, err := exec.Command("git", "status").CombinedOutput()
		if err != nil {
			m.messages = append(m.messages, chatMessage{role: "error", content: fmt.Sprintf("git status failed: %v", err)})
		} else {
			m.messages = append(m.messages, chatMessage{role: "system", content: "  Git Status:\n" + string(out)})
		}

	case "/diff":
		args := []string{"diff", "--stat"}
		if len(parts) > 1 && parts[1] == "full" {
			args = []string{"diff"}
		}
		out, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			m.messages = append(m.messages, chatMessage{role: "error", content: fmt.Sprintf("git diff failed: %v", err)})
		} else {
			if len(out) == 0 {
				m.messages = append(m.messages, chatMessage{role: "system", content: "  No changes."})
			} else {
				m.messages = append(m.messages, chatMessage{role: "system", content: "  Git Diff:\n" + string(out)})
			}
		}

	case "/commit":
		if len(parts) < 2 {
			m.messages = append(m.messages, chatMessage{role: "error", content: "Usage: /commit \"message\""})
		} else {
			// Primitive argument parsing to handle quotes
			// strings.Fields splits by space, so "p 1" becomes ["p", "1"]
			// We need to rejoin everything after /commit
			msg := strings.Join(parts[1:], " ")
			msg = strings.Trim(msg, "\"")

			// We should probably run 'git add .' first?
			// The user expectation of "commit" might imply "add and commit" or just commit.
			// Let's assume standard behavior: commit what is staged, unless user asks otherwise.
			// OR, for convenience, "commit -am" if prompt implies.
			// Let's stick to simple "commit -m".

			out, err := exec.Command("git", "commit", "-m", msg).CombinedOutput()
			if err != nil {
				m.messages = append(m.messages, chatMessage{role: "error", content: fmt.Sprintf("git commit failed: %v\n%s", err, out)})
			} else {
				m.messages = append(m.messages, chatMessage{role: "system", content: "  Git Commit:\n" + string(out)})
			}
		}

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

	// Iterate with index to allow lookahead/grouping
	for i := 0; i < len(m.messages); i++ {
		msg := m.messages[i]

		switch msg.role {
		case "welcome":
			// Welcome message
			sb.WriteString(m.renderAssistantBlock("Aseity", msg.content, true))

		case "user":
			sb.WriteString(m.renderUserBlock(msg.content))

		case "thinking":
			sb.WriteString(m.renderThinkingBlock(msg.content))

		case "assistant":
			sb.WriteString(m.renderAssistantBlock("Aseity", msg.content, false))

		case "tool":
			// Group tool call with its result if the next message is a result
			var result string
			if i+1 < len(m.messages) && m.messages[i+1].role == "tool_result" {
				result = m.messages[i+1].content
				i++ // Skip next message
			}
			// If result is empty, it might be running or just no output?
			// msg.content contains the formatted header from formatToolCallDisplay
			sb.WriteString(m.renderToolBlock(msg.content, result))

		case "tool_result":
			// Orphaned result (shouldn't happen often if grouped above)
			sb.WriteString(m.renderToolBlock("Previous Tool", msg.content))

		case "confirm_prompt":
			sb.WriteString(WarningStyle.Render("  ⚠ ") + ConfirmStyle.Render(msg.content) + "\n\n")

		case "confirm":
			sb.WriteString(SuccessStyle.Render("  ✓ "+msg.content) + "\n\n")

		case "confirm_deny":
			sb.WriteString(ErrorStyle.Render("  ✗ "+msg.content) + "\n\n")

		case "system":
			sb.WriteString(SystemMsgStyle.Render("  ℹ "+msg.content) + "\n\n")

		case "error":
			sb.WriteString(ErrorStyle.Render("  ✗ Error: "+msg.content) + "\n\n")

		case "subagent":
			sb.WriteString(AgentActivityStyle.Render("🤖 Agent Activity:\n"+msg.content) + "\n")
		}
	}

	// Spinner handling
	if m.thinking || m.confirming || m.inputRequest || m.currentTool != "" {
		spinnerFrame := m.spinner.View()
		status := m.getAnimatedStatus()

		var spinBlock string
		if m.currentTool != "" {
			// Tool is running
			spinBlock = fmt.Sprintf(" %s %s", spinnerFrame, status)
			spinBlock = ToolExecStyle.Render(spinBlock)
		} else {
			// Just thinking
			spinBlock = fmt.Sprintf(" %s %s", spinnerFrame, status)
			// Apply the current spinner style
			spinBlock = m.spinner.Style.Render(spinBlock)
		}
		// Wrap in a subtle box or just margin?
		// Let's keep it simple but aligned
		sb.WriteString(spinBlock + "\n")
	}

	// Sticky Bottom Logic
	// Only scroll to bottom if we were already there OR if we are initiating (empty content)
	wasAtBottom := m.viewport.AtBottom()

	m.viewport.SetContent(sb.String())

	if wasAtBottom || len(m.messages) <= 1 {
		m.viewport.GotoBottom()
	}
}

// --- Block Rendering Helpers ---

func (m *Model) renderUserBlock(content string) string {
	return UserBlockStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			RoleHeaderStyle.Foreground(BrightGreen).Render("USER"),
			UserMsgStyle.Render(content),
		),
	) + "\n"
}

func (m *Model) renderAssistantBlock(title, content string, isWelcome bool) string {
	var body string
	if isWelcome {
		// Custom banner handling within the block
		// We expect content to be the text part.
		// Reconstruct banner?
		// The original code rendered banner separately.
		// Let's just render content.
		body = AssistantMsgStyle.Render(content)
	} else {
		// Render markdown
		rendered, err := m.renderer.Render(content)
		if err != nil {
			body = content
		} else {
			body = rendered
		}
	}
	body = strings.TrimRight(body, "\n")

	// Special check: Is this a Plan?
	// If content starts with "# Plan" or similar, maybe distinct style?
	// For now, standard assistant block.

	return AssistantBlockStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			RoleHeaderStyle.Foreground(Cyan).Render(title),
			body,
		),
	) + "\n"
}

func (m *Model) renderThinkingBlock(content string) string {
	if !m.showThinking && content != "" {
		lines := strings.Count(content, "\n") + 1
		return ThinkingBlockStyle.Render(fmt.Sprintf("💭 Reasoning (%d lines) [Ctrl+T to expand]", lines)) + "\n"
	}

	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var formatted strings.Builder
	formatted.WriteString(ThinkingLabelStyle.Render("💭 Reasoning") + "\n")

	maxLines := 15
	if len(lines) > maxLines {
		for _, line := range lines[:maxLines] {
			formatted.WriteString("│ " + line + "\n")
		}
		formatted.WriteString(fmt.Sprintf("└─ ... (%d more lines)\n", len(lines)-maxLines))
	} else {
		for i, line := range lines {
			prefix := "│ "
			if i == len(lines)-1 {
				prefix = "└─ "
			}
			formatted.WriteString(prefix + line + "\n")
		}
	}

	return ThinkingBlockStyle.Render(formatted.String()) + "\n"
}

func (m *Model) renderToolBlock(header, result string) string {
	// header comes from formatToolCallDisplay, so it's already styled/colored.
	// But lipgloss styles might strip if we nest? No, usually fine.

	// Ensure header is clean
	header = strings.TrimSpace(header)

	var content string
	if result != "" {
		// Clean result
		result = strings.TrimSpace(result)
		if len(result) > 500 {
			result = result[:500] + "\n... (truncated)"
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			header,
			lipgloss.NewStyle().Foreground(MidGray).Render("─────"), // Separator
			ToolResultStyle.Render(result),
		)
	} else {
		content = header
	}

	return ToolBlockStyle.Render(content) + "\n"
}

func (m Model) View() string {
	// --- Header Construction (Restored Wave Style) ---
	// Left Column: Animated Banner
	// We use m.frame to animate the gradient colors
	logo := AnimatedBanner(m.frame)

	leftContent := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		fmt.Sprintf("%s / %s", m.providerName, m.modelName),
		// Add some breathing room
	)

	// Right Column: Context/Tips (Vertical stack)
	contextState := "Active"
	if m.thinking {
		contextState = "Thinking..."
	} else if m.currentTool != "" {
		contextState = "Running Tool..."
	}

	rightContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(ClaudeAccent).Bold(true).Render("STATUS"),
		lipgloss.NewStyle().Foreground(White).Render(contextState),
		"",
		lipgloss.NewStyle().Foreground(ClaudeAccent).Bold(true).Render("TIPS"),
		lipgloss.NewStyle().Foreground(DimGreen).Render("/help"),
		lipgloss.NewStyle().Foreground(DimGreen).Render("Ctrl+C to quit"),
	)

	// Layout: Logo on Left, Status on Right, separated by a pipe??
	// Or just side by side with spacing.
	// The animated banner is wide (~40 chars).

	headerInner := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().PaddingLeft(2).Render(leftContent),
		lipgloss.NewStyle().Width(4).Render(""),                                // Reduced Spacer
		lipgloss.NewStyle().PaddingRight(2).PaddingTop(2).Render(rightContent), // Status
	)

	// Remove the border box (LogoBoxStyle) to fix "boundary too long"
	// Just render the inner content centered or left-aligned?
	// User complaint "boundary is too long" -> Border.

	// We'll use a subtle bottom border for the whole header separation if needed, or just space.
	// Actually, let's keep it minimal.
	// Just center the whole header block.
	header := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(DimGreen).
		Width(m.width).
		Align(lipgloss.Center).
		Render(headerInner)

	// --- Input Area (Enhanced Box) ---
	prompt := lipgloss.NewStyle().Foreground(Green).Bold(true).Render("> ")
	if m.thinking {
		prompt = lipgloss.NewStyle().Foreground(Purple).Bold(true).Render("● ")
	} else if m.confirming {
		prompt = lipgloss.NewStyle().Foreground(Amber).Bold(true).Render("? ")
	}

	// Render textarea view inside the box
	inputContent := lipgloss.JoinHorizontal(lipgloss.Top,
		prompt,
		m.textarea.View(),
	)

	// Wrap in bordered box
	// Calculate width to match header/viewport
	inputBox := InputBoxStyle.
		Width(m.width - 4). // Account for margins
		Render(inputContent)

	// --- Footer ---
	keyStyle := lipgloss.NewStyle().Foreground(DimGreen)
	help := keyStyle.Render("Enter: send  •  Alt+Enter: newline  •  /help  •  Esc: quit")

	// --- Layout Assembly ---
	// We use JoinVertical to stack everything

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		header,
		ViewportStyle.Render(m.viewport.View()), // Apply viewport border style
		inputBox,
		lipgloss.NewStyle().PaddingLeft(2).Render(help),
	)

	if m.menu.active {
		// Overlay logic would go here, currently just appending
		return lipgloss.JoinVertical(lipgloss.Left, mainView, m.menu.View())
	}

	return mainView
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
