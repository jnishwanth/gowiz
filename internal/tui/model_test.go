package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/wiz"
)

func TestTUIModel(t *testing.T) {
	mockClient := wiz.NewMockClient()
	m := NewModel(mockClient, "192.168.1.115")

	t.Run("Initial model state", func(t *testing.T) {
		if m.mode != ModeNormal {
			t.Errorf("expected initial mode Normal, got %v", m.mode)
		}
		if m.activePanel != PanelDevices {
			t.Errorf("expected initial panel Devices, got %v", m.activePanel)
		}
	})

	t.Run("Navigation tab panel switching", func(t *testing.T) {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: false})
		m = updated.(Model)
		// Send tab
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)

		if m.activePanel != PanelControl {
			t.Errorf("expected panel Control after Tab, got %v", m.activePanel)
		}
	})

	t.Run("Command mode execution", func(t *testing.T) {
		// Enter command mode
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)

		if m.mode != ModeCommand {
			t.Fatalf("expected ModeCommand, got %v", m.mode)
		}

		// Type 'dim 50'
		for _, r := range "dim 50" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}

		// Hit enter
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		if cmd != nil {
			msg := cmd()
			m.Update(msg)
		}

		activeDev, _ := m.Registry.GetActive()
		if activeDev.Brightness != 50 {
			t.Errorf("expected brightness 50 after :dim 50, got %d", activeDev.Brightness)
		}
	})

	t.Run("Visual multi-select mode", func(t *testing.T) {
		// Toggle visual mode with 'v'
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		m = updated.(Model)

		if m.mode != ModeVisual {
			t.Errorf("expected ModeVisual, got %v", m.mode)
		}

		// Toggle select all with 'a'
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m = updated.(Model)

		selected := m.Registry.GetSelectedOrActive()
		if len(selected) != 1 {
			t.Errorf("expected 1 selected bulb, got %d", len(selected))
		}
	})

	t.Run("View rendering sanity", func(t *testing.T) {
		view := m.View()
		if len(view) == 0 {
			t.Errorf("expected non-empty rendered view")
		}
	})
}
