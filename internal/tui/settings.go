package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jeanpaul/aseity/internal/agent/skillsets"
)

// SettingsModel represents the settings menu
type SettingsModel struct {
	list           list.Model
	active         bool
	width          int
	height         int
	selectedAction string
	profile        *skillsets.ModelProfile
	userConfig     *skillsets.UserConfig
}

type settingsItem struct {
	title  string
	desc   string
	action string
}

func (i settingsItem) Title() string       { return i.title }
func (i settingsItem) Description() string { return i.desc }
func (i settingsItem) FilterValue() string { return i.title }

// NewSettingsModel creates a new settings menu
func NewSettingsModel(profile *skillsets.ModelProfile, config *skillsets.UserConfig) SettingsModel {
	items := []list.Item{
		settingsItem{title: "📊 View Profile", desc: "Show current model profile and capabilities", action: "view_profile"},
		settingsItem{title: "🎓 Manage Skillsets", desc: "Toggle skillsets on/off", action: "manage_skillsets"},
		settingsItem{title: "✓ Validation Level", desc: "Change validation strictness", action: "validation"},
		settingsItem{title: "🔧 Training Settings", desc: "Configure skillset training", action: "training"},
		settingsItem{title: "💾 Save Config", desc: "Save current settings to file", action: "save"},
		settingsItem{title: "🔄 Reset Defaults", desc: "Reset to default configuration", action: "reset"},
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(Green).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(Green).PaddingLeft(1)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy().Foreground(DimGreen)

	l := list.New(items, d, 50, 14)
	l.Title = "Settings"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(Green).Bold(true).MarginLeft(2)

	return SettingsModel{
		list:       l,
		active:     false,
		profile:    profile,
		userConfig: config,
	}
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.active = false
			return m, nil
		case "enter":
			if selected, ok := m.list.SelectedItem().(settingsItem); ok {
				m.selectedAction = selected.action
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m SettingsModel) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(Green).
		Bold(true).
		Padding(1, 2).
		Render("⚙️  Aseity Settings")

	b.WriteString(header + "\n\n")

	// Current profile info
	if m.profile != nil {
		info := fmt.Sprintf("Model: %s | Tier: %d | Strategy: %s | Validation: %s",
			m.profile.Name,
			m.profile.Tier,
			m.profile.PromptStrategy,
			validationLevelName(m.profile.ValidationLevel))

		infoStyle := lipgloss.NewStyle().
			Foreground(DimGreen).
			Padding(0, 2)

		b.WriteString(infoStyle.Render(info) + "\n\n")
	}

	// Menu list
	b.WriteString(m.list.View())

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(DimGreen).
		Padding(1, 2).
		Render("↑/↓: Navigate | Enter: Select | Esc: Close")

	b.WriteString("\n" + footer)

	// Border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Green).
		Padding(1, 2).
		Render(b.String())
}

func (m *SettingsModel) Activate() {
	m.active = true
}

func (m *SettingsModel) Deactivate() {
	m.active = false
}

func (m *SettingsModel) IsActive() bool {
	return m.active
}

func (m *SettingsModel) GetSelectedAction() string {
	action := m.selectedAction
	m.selectedAction = "" // Clear after reading
	return action
}

func validationLevelName(level skillsets.ValidationLevel) string {
	names := map[skillsets.ValidationLevel]string{
		skillsets.ValidationNone:   "None",
		skillsets.ValidationLight:  "Light",
		skillsets.ValidationMedium: "Medium",
		skillsets.ValidationStrict: "Strict",
	}
	if name, ok := names[level]; ok {
		return name
	}
	return "Unknown"
}
