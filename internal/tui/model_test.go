package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/wiz"
)

func TestTUIModelUX(t *testing.T) {
	mockClient := wiz.NewMockClient()
	m := NewModel(mockClient, "192.168.1.115")

	t.Run("Direct number hotkey 1-9 activates favorite scene", func(t *testing.T) {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		m = updated.(Model)

		if cmd != nil {
			msg := cmd()
			m.Update(msg)
		}

		activeDev, _ := m.Registry.GetActive()
		if activeDev.SceneID != 2 {
			t.Errorf("expected sceneID 2 after pressing key 2, got %d", activeDev.SceneID)
		}
	})

	t.Run("Sleep timer set and countdown tick", func(t *testing.T) {
		// Set timer using 't' key
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m = updated.(Model)

		if m.sleepTimerSecs != 15*60 {
			t.Errorf("expected 900 seconds sleep timer, got %d", m.sleepTimerSecs)
		}

		// Send TimerTickMsg
		updated, _ = m.Update(TimerTickMsg(time.Now()))
		m = updated.(Model)

		if m.sleepTimerSecs != 15*60-1 {
			t.Errorf("expected sleep timer decremented by 1, got %d", m.sleepTimerSecs)
		}
	})

	t.Run("Window size resize handling", func(t *testing.T) {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		m = updated.(Model)

		if m.width != 120 || m.height != 40 {
			t.Errorf("expected dimensions 120x40, got %dx%d", m.width, m.height)
		}

		view := m.View()
		if len(view) == 0 {
			t.Errorf("rendered view on resize should not be empty")
		}
	})
}
