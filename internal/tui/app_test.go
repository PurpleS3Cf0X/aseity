package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jeanpaul/aseity/internal/provider"
	"github.com/jeanpaul/aseity/internal/tools"
)

// Mock objects for initialization
type mockProvider struct{}

func (m mockProvider) Chat(ctx context.Context, msgs []provider.Message, toolDefs []provider.ToolDef) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	close(ch)
	return ch, nil
}
func (m mockProvider) Name() string { return "mock-provider" }

func (m mockProvider) ModelName() string          { return "test-model" }
func (m *MockProviderFallback) ModelName() string { return "test-model" }
func (m mockProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestMenuTrigger(t *testing.T) {
	// Setup
	prov := mockProvider{}
	reg := tools.NewRegistry(nil, false) // Fixed args
	model := NewModel(prov, reg, "mock-provider", "mock-model", nil, false)

	// Ensure menu starts inactive
	if model.menu.active {
		t.Error("Menu should be inactive on startup")
	}

	// Send WindowSize first to init dimensions (prevents panic from negative viewport)
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model = updatedModel.(Model)

	// Send '/' key to trigger menu
	propModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m := propModel.(Model)

	if !m.menu.active {
		t.Error("Menu should be active after pressing '/'")
	}

	// Verify View contains menu specific styling (e.g. LogoBox)
	view := m.View()
	// Check for a menu-specific item that is NOT in the footer help text
	if !strings.Contains(view, "/compact") {
		t.Error("View should display menu items like '/compact'")
	}
}

func TestMenuSelection(t *testing.T) {
	prov := mockProvider{}
	reg := tools.NewRegistry(nil, false) // Fixed args
	model := NewModel(prov, reg, "mock-provider", "mock-model", nil, false)

	// Activate menu manually
	model.menu.active = true

	// Select second item (should be /compact or similar)
	// Default order: /help, /compact, /clear...
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // j or down arrow

	// Select it
	propModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := propModel.(Model)

	// Menu should be closed
	if m.menu.active {
		t.Error("Menu should close after selection")
	}

	// Input should be populated with command or executed
	// Our logic: SetValue(cmdStr)
	val := m.textarea.Value()
	if !strings.HasPrefix(val, "/") {
		t.Errorf("Textarea should be populated with command, got '%s'", val)
	}
}

func TestInputPromptDesign(t *testing.T) {
	prov := mockProvider{}
	reg := tools.NewRegistry(nil, false) // Fixed args
	model := NewModel(prov, reg, "mock-provider", "mock-model", nil, false)

	// Check View for minimal prompt "> "
	view := model.View()
	if !strings.Contains(view, "> ") {
		t.Error("View should contain the minimal prompt '> '")
	}
}

func TestHeaderRendering(t *testing.T) {
	prov := mockProvider{}
	reg := tools.NewRegistry(nil, false) // Fixed args
	model := NewModel(prov, reg, "mock-provider", "mock-model", nil, false)
	model.width = 100
	model.height = 50

	view := model.View()

	// Check for new header elements
	if !strings.Contains(view, "Aseity") {
		t.Error("Header should contain 'Aseity' logo/text")
	}
	if !strings.Contains(view, "mock-model") {
		t.Error("Header should display model name")
	}
	// Check for Tips section
	if !strings.Contains(view, "Tips") {
		t.Error("Header should contain 'Tips' section")
	}
}
