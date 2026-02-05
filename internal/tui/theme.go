package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Core palette — Enhanced Matrix-inspired with Claude Code accents
	Green       = lipgloss.Color("#00FF41") // Primary neon green
	BrightGreen = lipgloss.Color("#39FF14") // Highlight green
	MedGreen    = lipgloss.Color("#00C832") // Mid-tone green
	DarkGreen   = lipgloss.Color("#008F11") // Darker green
	DimGreen    = lipgloss.Color("#004d00") // Subtle green (brighter)
	Cyan        = lipgloss.Color("#00D4FF") // Brighter cyan for assistant
	Teal        = lipgloss.Color("#00AA88") // Teal accent
	Black       = lipgloss.Color("#0D0208")
	DarkBG      = lipgloss.Color("#0a0a0f")
	DarkGray    = lipgloss.Color("#1a1a2e")
	MidGray     = lipgloss.Color("#5a5a6e") // Brighter for readability
	LightGray   = lipgloss.Color("#aaaaaa")
	White       = lipgloss.Color("#FFFFFF")

	// Accent colors — Claude Code inspired
	Purple       = lipgloss.Color("#A855F7") // Thinking/reasoning
	DimPurple    = lipgloss.Color("#7C3AED") // Thinking text
	Orange       = lipgloss.Color("#FF9500") // Warnings/confirmations
	ClaudeAccent = lipgloss.Color("#00FF41") // Reverted to Green (Matrix style) for user theme
	Amber        = lipgloss.Color("#FFB000") // Warm accent (Claude-like)
	Gold         = lipgloss.Color("#FFD700") // Highlights
	Blue         = lipgloss.Color("#3B82F6") // Links/info
	Red          = lipgloss.Color("#EF4444") // Errors
	Magenta      = lipgloss.Color("#EC4899") // Special actions
	SoftYellow   = lipgloss.Color("#FBBF24") // Soft highlights
	MintGreen    = lipgloss.Color("#34D399") // Success indicators

	// New Styles for TUI Improvements
	ThinkingColor      = lipgloss.Color("#6B7280") // Gray/Dim
	ToolExecColor      = lipgloss.Color("#3B82F6") // Blue
	AgentActivityColor = lipgloss.Color("#10B981") // Emerald

	// Status bar — Gradient effect
	StatusBarStyle = lipgloss.NewStyle().
			Background(DarkGreen).
			Foreground(White).
			Bold(true).
			Padding(0, 1)

	StatusProviderStyle = lipgloss.NewStyle().
				Background(Green).
				Foreground(Black).
				Bold(true).
				Padding(0, 1)

	// Token counter — more visible
	TokenStyle = lipgloss.NewStyle().
			Foreground(Teal).
			Italic(true)

	// User messages — WHITE font with green accent label
	UserLabelStyle = lipgloss.NewStyle().
			Foreground(BrightGreen).
			Bold(true)

	UserMsgStyle = lipgloss.NewStyle().
			Foreground(White)

	// Assistant messages — Cyan branding
	AssistantLabelStyle = lipgloss.NewStyle().
				Foreground(Cyan).
				Bold(true)

	AssistantMsgStyle = lipgloss.NewStyle().
				Foreground(White)

	// Tool calls — Enhanced with icons and colors
	ToolCallStyle = lipgloss.NewStyle().
			Foreground(MedGreen)

	ToolLabelStyle = lipgloss.NewStyle().
			Foreground(MintGreen).
			Bold(true)

	CommandStyle = lipgloss.NewStyle().
			Foreground(Amber).
			Bold(true)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(MidGray)

	// Tool-specific colors
	BashIconStyle = lipgloss.NewStyle().
			Foreground(Amber).
			Bold(true)

	FileIconStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	WebIconStyle = lipgloss.NewStyle().
			Foreground(Magenta).
			Bold(true)

	AgentIconStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true)

	// Confirmation dialog — Amber/Orange warning style
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(Amber).
			Bold(true)

	ConfirmInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Amber).
				Padding(0, 1)

	// Input — Minimal "Claude Code" style (Prompt >)
	InputBorderStyle = lipgloss.NewStyle().
				Foreground(Green)

	InputActiveStyle = lipgloss.NewStyle().
				Foreground(Green)

	// Spinner — Multiple styles for different states
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(BrightGreen)

	SpinnerThinkingStyle = lipgloss.NewStyle().
				Foreground(Purple)

	SpinnerToolStyle = lipgloss.NewStyle().
				Foreground(Cyan)

	// Banner
	BannerStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	// Separator — Subtle gradient
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(DimGreen)

	// Error — Red with emphasis
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// Help text — More visible
	HelpStyle = lipgloss.NewStyle().
			Foreground(DimGreen)

	// Thinking blocks — Purple theme (like Claude's reasoning)
	ThinkingLabelStyle = lipgloss.NewStyle().
				Foreground(Purple).
				Bold(true)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(ThinkingColor).
			Italic(true)

	ToolExecStyle = lipgloss.NewStyle().
			Foreground(ToolExecColor).
			Bold(true)

	AgentActivityStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(AgentActivityColor).
				Padding(0, 1).
				Margin(0, 2)

	// Sub-agent activity — Distinct purple/teal
	SubAgentStyle = lipgloss.NewStyle().
			Foreground(Teal).
			Italic(true)

	// Slash command feedback
	SystemMsgStyle = lipgloss.NewStyle().
			Foreground(MintGreen).
			Italic(true)

	// Code blocks and syntax highlighting accents

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(MintGreen).
			Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Foreground(Orange).
			Bold(true)

	// Info style
	InfoStyle = lipgloss.NewStyle().
			Foreground(Blue)

	// Logo & Welcome Styles
	LogoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ClaudeAccent).
			Padding(0, 1).
			Margin(1, 0)

	WelcomeTextStyle = lipgloss.NewStyle().
				Foreground(Green).
				Bold(true).
				Align(lipgloss.Center)

	// --- Block UI Styles (V2) ---

	// UserBlockStyle — Green/Bright rounded box
	UserBlockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BrightGreen).
			Padding(1, 2).
			Margin(1, 0).
			Width(76) // Fixed width for consistent look, or dynamic in app

	// AssistantBlockStyle — Cyan rounded box
	AssistantBlockStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Cyan).
				Padding(1, 2).
				Margin(0, 0, 1, 0)

	// ToolBlockStyle — Subtle, distinct from chat
	ToolBlockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(MidGray).
			Padding(0, 1).
			Margin(0, 0, 1, 2) // Indented slightly

	// ThinkingBlockStyle — Minimalist, dim
	ThinkingBlockStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()). // Or distinct if preferred
				Padding(0, 2).
				Margin(0, 0, 1, 0).
				Foreground(DimPurple)

	// CodeBlockStyle — Rich background
	CodeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a2e")). // Deep blue/black
			Foreground(lipgloss.Color("#e0e0e0")).
			Padding(1, 2).
			Margin(1, 0)

	// Headers for blocks
	RoleHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)

	// --- New Layout Styles ---

	// InputBoxStyle - Distinct box for user input
	InputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FF41")). // Matrix Green
			Padding(0, 1).
			Margin(0, 0)

	// ViewportStyle - Border for the main content area
	ViewportStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, true). // Vertical borders only? Or full box?
		// Let's try a subtle border to separate it from edges
		BorderForeground(lipgloss.Color("#1a1a2e")).
		Padding(0, 1)
)

const Banner = `
   ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗
  ██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝
  ███████║███████╗█████╗  ██║   ██║    ╚████╔╝
  ██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝
  ██║  ██║███████║███████╗██║   ██║      ██║
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝
`

// AnimatedBanner returns a frame-based gradient banner
func AnimatedBanner(frame int) string {
	lines := []string{
		" ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗",
		"██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝",
		"███████║███████╗█████╗  ██║   ██║    ╚████╔╝",
		"██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝",
		"██║  ██║███████║███████╗██║   ██║      ██║",
		"╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝",
	}

	// Gradient palette
	palette := []lipgloss.Color{
		"#39FF14", // Bright green
		"#00FF41", // Matrix green
		"#00E636", // Mid green
		"#00D4AA", // Teal
		"#00C8D4", // Cyan-teal
		"#00D4FF", // Cyan
		"#0099FF", // Blue-Cyan
		"#00D4FF", // Cyan
		"#00C8D4", // Cyan-teal
		"#00D4AA", // Teal
		"#00E636", // Mid green
		"#00FF41", // Matrix green
	}

	result := ""
	for i, line := range lines {
		// Calculate color index based on frame + line offset
		idx := (frame/2 + i) % len(palette)
		style := lipgloss.NewStyle().Foreground(palette[idx]).Bold(true)
		result += style.Render(line) + "\n"
	}
	return result
}

// GradientBanner keeps compatibility if needed, using static frame 0
func GradientBanner() string {
	return AnimatedBanner(0)
}
