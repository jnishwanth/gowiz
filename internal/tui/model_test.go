package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/wiz"
)

func TestTUIStateSynchronization(t *testing.T) {
	mockClient := wiz.NewMockClient()
	m := NewModel(mockClient, "192.168.1.115")

	t.Run("Confirmed command state update", func(t *testing.T) {
		// Send dimming command
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		m = updated.(Model)

		if cmd != nil {
			msg := cmd()
			// Process CommandFinishedMsg
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
