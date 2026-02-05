package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuType defines which menu context we are in
type MenuType int

const (
	MenuNone MenuType = iota
	MenuSlashCommands
	MenuAgentSelection
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type MenuModel struct {
	list     list.Model
	active   bool
	menuType MenuType
	width    int
	height   int
}

func NewMenuModel() MenuModel {
	// Default slash commands
	items := []list.Item{
		item{title: "/help", desc: "Show help commands"},
		item{title: "/compact", desc: "Summarize history to save tokens"},
		item{title: "/clear", desc: "Clear conversation history"},
		item{title: "/agents", desc: "Manage or switch agents"},
		item{title: "/settings", desc: "Open settings menu"},
		item{title: "/skillsets", desc: "View and manage skillsets"},
		item{title: "/profile", desc: "Show current model profile"},
		item{title: "/quit", desc: "Exit the application"},
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(Green).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(Green).PaddingLeft(1)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy().Foreground(DimGreen)

	l := list.New(items, d, 30, 14) // Fixed size for menu popup
	l.Title = "Commands"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(Green).Bold(true).MarginLeft(2)

	return MenuModel{
		list:   l,
		active: false,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.active = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MenuModel) View() string {
	if !m.active {
		return ""
	}
	// Render list in a box
	return LogoBoxStyle.Render(m.list.View())
}
