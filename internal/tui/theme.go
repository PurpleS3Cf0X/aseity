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
	Purple     = lipgloss.Color("#A855F7") // Thinking/reasoning
	DimPurple  = lipgloss.Color("#7C3AED") // Thinking text
	Orange     = lipgloss.Color("#FF9500") // Warnings/confirmations
	Gold       = lipgloss.Color("#FFD700") // Highlights
	Blue       = lipgloss.Color("#3B82F6") // Links/info
	Red        = lipgloss.Color("#EF4444") // Errors
	Magenta    = lipgloss.Color("#EC4899") // Special actions
	SoftYellow = lipgloss.Color("#FBBF24") // Soft highlights
	MintGreen  = lipgloss.Color("#34D399") // Success indicators

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
			Foreground(SoftYellow).
			Bold(true)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(MidGray)

	// Tool-specific colors
	BashIconStyle = lipgloss.NewStyle().
			Foreground(Orange).
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

	// Confirmation dialog — Orange warning style
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(Orange).
			Bold(true)

	ConfirmInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Orange).
				Padding(0, 1)

	// Input — Animated border feel
	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DarkGreen).
				Padding(0, 1)

	InputActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(BrightGreen).
				Padding(0, 1)

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
			Foreground(DimPurple).
			Italic(true)

	// Sub-agent activity — Distinct purple/teal
	SubAgentStyle = lipgloss.NewStyle().
			Foreground(Teal).
			Italic(true)

	// Slash command feedback
	SystemMsgStyle = lipgloss.NewStyle().
			Foreground(MintGreen).
			Italic(true)

	// Code blocks and syntax highlighting accents
	CodeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a2e")).
			Foreground(White).
			Padding(0, 1)

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
)

const Banner = `
   ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗
  ██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝
  ███████║███████╗█████╗  ██║   ██║    ╚████╔╝
  ██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝
  ██║  ██║███████║███████╗██║   ██║      ██║
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝
`

// GradientBanner returns a gradient-colored banner (green to cyan)
func GradientBanner() string {
	lines := []string{
		"   ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗",
		"  ██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝",
		"  ███████║███████╗█████╗  ██║   ██║    ╚████╔╝ ",
		"  ██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝  ",
		"  ██║  ██║███████║███████╗██║   ██║      ██║   ",
		"  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝   ",
	}

	// Gradient colors from bright green to cyan
	colors := []lipgloss.Color{
		"#39FF14", // Bright green
		"#00FF41", // Matrix green
		"#00E636", // Mid green
		"#00D4AA", // Teal
		"#00C8D4", // Cyan-teal
		"#00D4FF", // Cyan
	}

	result := "\n"
	for i, line := range lines {
		style := lipgloss.NewStyle().Foreground(colors[i]).Bold(true)
		result += style.Render(line) + "\n"
	}
	return result
}
