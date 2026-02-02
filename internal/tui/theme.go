package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Core palette
	Green       = lipgloss.Color("#00FF41")
	BrightGreen = lipgloss.Color("#39FF14")
	MedGreen    = lipgloss.Color("#00C832")
	DarkGreen   = lipgloss.Color("#008F11")
	DimGreen    = lipgloss.Color("#003B00")
	Cyan        = lipgloss.Color("#00D4AA")
	Black       = lipgloss.Color("#0D0208")
	DarkBG      = lipgloss.Color("#0a0a0f")
	DarkGray    = lipgloss.Color("#1a1a2e")
	MidGray     = lipgloss.Color("#3a3a4e")
	LightGray   = lipgloss.Color("#aaaaaa")
	White       = lipgloss.Color("#e0e0e0")

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(DarkGreen).
			Foreground(Black).
			Bold(true).
			Padding(0, 1)

	StatusProviderStyle = lipgloss.NewStyle().
				Background(Green).
				Foreground(Black).
				Bold(true).
				Padding(0, 1)

	// User messages
	UserLabelStyle = lipgloss.NewStyle().
			Foreground(BrightGreen).
			Bold(true)

	UserMsgStyle = lipgloss.NewStyle().
			Foreground(Green)

	// Assistant messages
	AssistantLabelStyle = lipgloss.NewStyle().
				Foreground(Cyan).
				Bold(true)

	AssistantMsgStyle = lipgloss.NewStyle().
				Foreground(White)

	// Tool calls
	ToolCallStyle = lipgloss.NewStyle().
			Foreground(DarkGreen).
			Italic(true)

	ToolLabelStyle = lipgloss.NewStyle().
			Foreground(MedGreen).
			Bold(true)

	CommandStyle = lipgloss.NewStyle().
			Foreground(BrightGreen).
			Bold(true)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(MidGray)

	// Confirmation dialog
	ConfirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)

	ConfirmInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FFD700")).
				Padding(0, 1)

	// Input
	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(DarkGreen).
				Padding(0, 1)

	InputActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Green).
				Padding(0, 1)

	// Spinner
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(BrightGreen)

	// Banner
	BannerStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	// Separator
	SeparatorStyle = lipgloss.NewStyle().
			Foreground(DimGreen)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4136")).
			Bold(true)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(DimGreen)

	// Thinking blocks
	ThinkingLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#556B2F")).
				Italic(true).
				Bold(true)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4a5a3a")).
			Italic(true)

	// Sub-agent activity
	SubAgentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00AA77")).
			Italic(true)
)

const Banner = `
   ██████╗ ███████╗███████╗██╗████████╗██╗   ██╗
  ██╔══██╗██╔════╝██╔════╝██║╚══██╔══╝╚██╗ ██╔╝
  ███████║███████╗█████╗  ██║   ██║    ╚████╔╝
  ██╔══██║╚════██║██╔══╝  ██║   ██║     ╚██╔╝
  ██║  ██║███████║███████╗██║   ██║      ██║
  ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝   ╚═╝      ╚═╝
`
