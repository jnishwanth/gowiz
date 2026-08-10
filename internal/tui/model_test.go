package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/wiz"
)

func TestTUIStateSynchronization(t *testing.T) {
	mockClient := wiz.NewMockClient()
	m := NewModel(mockClient, "192.168.1.115")

	t.Run("Responsive terminal layout rendering line count bound", func(t *testing.T) {
		sizes := []struct {
			w, h int
			name string
		}{
			{140, 50, "Ultrawide 140x50"},
			{80, 24, "Standard VT100 80x24"},
			{60, 18, "Compact Pane 60x18"},
			{45, 14, "Mobile Narrow Pane 45x14"},
		}

		for _, sz := range sizes {
			updated, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
			mod := updated.(Model)
			viewOutput := mod.View()

			if len(viewOutput) == 0 {
				t.Errorf("%s: view output is empty", sz.name)
			}
			if !strings.Contains(viewOutput, "gowiz") {
				t.Errorf("%s: view output missing title header", sz.name)
			}

			lines := strings.Split(viewOutput, "\n")
			lineCount := len(lines)

			if lineCount > sz.h {
				t.Errorf("%s: line count %d EXCEEDS terminal height %d! This causes top cutoff scroll!", sz.name, lineCount, sz.h)
			}
		}
	})

	t.Run("Panel tabbing consistency", func(t *testing.T) {
		if m.activePanel != PanelDevices {
			t.Fatalf("expected initial PanelDevices, got %v", m.activePanel)
		}

		// Tab to PanelControl
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
		if m.activePanel != PanelControl {
			t.Errorf("expected PanelControl after 1st Tab, got %v", m.activePanel)
		}
		if !strings.Contains(m.View(), "Focus: 2. Controls") {
			t.Errorf("expected status bar focus 2. Controls")
		}

		// Tab to PanelScenes
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
		if m.activePanel != PanelScenes {
			t.Errorf("expected PanelScenes after 2nd Tab, got %v", m.activePanel)
		}
		if !strings.Contains(m.View(), "Focus: 3. Scenes") {
			t.Errorf("expected status bar focus 3. Scenes")
		}

		// Tab back to PanelDevices
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
		if m.activePanel != PanelDevices {
			t.Errorf("expected PanelDevices after 3rd Tab, got %v", m.activePanel)
		}

		// Shift+Tab back to PanelScenes
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		m = updated.(Model)
		if m.activePanel != PanelScenes {
			t.Errorf("expected PanelScenes after Shift+Tab, got %v", m.activePanel)
		}
	})

	t.Run("Confirmed command state update", func(t *testing.T) {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		m = updated.(Model)

		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		activeDev, _ := m.Registry.GetActive()
		if !activeDev.Online {
			t.Errorf("expected confirmed bulb to be marked Online")
		}
	})

	t.Run("Unreachable bulb failure state handling", func(t *testing.T) {
		mockClient.ShouldFail = true

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m = updated.(Model)

		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		activeDev, _ := m.Registry.GetActive()
		if activeDev.Online {
			t.Errorf("expected unreachable bulb to be marked Offline")
		}

		mockClient.ShouldFail = false
	})

	t.Run("Telemetry sync message updates device state", func(t *testing.T) {
		state := true
		dim := 42
		temp := 3200

		msg := TelemetryReceivedMsg{
			IP: "192.168.1.115",
			Pilot: &wiz.PilotParams{
				State:   &state,
				Dimming: &dim,
				Temp:    &temp,
			},
			Err: nil,
		}

		updated, _ := m.Update(msg)
		m = updated.(Model)

		dev, _ := m.Registry.Get("192.168.1.115")
		if dev.Brightness != 42 || dev.Temp != 3200 {
			t.Errorf("expected telemetry sync brightness 42 temp 3200, got %d and %d", dev.Brightness, dev.Temp)
		}
	})
}
