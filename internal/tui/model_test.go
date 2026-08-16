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

	t.Run("Space key toggles power in Normal mode", func(t *testing.T) {
		activeDev, ok := m.Registry.GetActive()
		if !ok {
			t.Fatalf("expected active device")
		}
		initialState := activeDev.State

		// Send space key rune (' ')
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		activeDev, _ = m.Registry.GetActive()
		if activeDev.State == initialState {
			t.Errorf("expected power state to toggle from %v to %v", initialState, !initialState)
		}

		// Send space key again via tea.KeySpace
		newState := activeDev.State
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		activeDev, _ = m.Registry.GetActive()
		if activeDev.State == newState {
			t.Errorf("expected power state to toggle back from %v to %v", newState, !newState)
		}
	})

	t.Run("Space key toggles selection in Visual mode", func(t *testing.T) {
		// Enter Visual mode
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
		m = updated.(Model)
		if m.mode != ModeVisual {
			t.Fatalf("expected mode ModeVisual, got %v", m.mode)
		}

		activeDev, _ := m.Registry.GetActive()
		initialSelected := activeDev.Selected

		// Press space key to toggle selection
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
		m = updated.(Model)

		activeDev, _ = m.Registry.GetActive()
		if activeDev.Selected == initialSelected {
			t.Errorf("expected selection to toggle from %v in visual mode", initialSelected)
		}

		// Exit visual mode with esc
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
		if m.mode != ModeNormal {
			t.Errorf("expected mode ModeNormal after Esc, got %v", m.mode)
		}
	})

	t.Run("Enter key action dispatch across panels", func(t *testing.T) {
		// Panel 3 (Scenes): Pressing Enter dispatches scene pilot command
		m.activePanel = PanelScenes
		m.sceneCursor = 0 // First scene (Ocean, ID 1)

		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd == nil {
			t.Fatalf("expected dispatch command on Enter in PanelScenes")
		}
		msg := cmd()
		updated, _ = m.Update(msg)
		m = updated.(Model)

		activeDev, _ := m.Registry.GetActive()
		if activeDev.SceneID != 1 {
			t.Errorf("expected active scene ID 1, got %d", activeDev.SceneID)
		}

		// Panel 1 (Devices): Pressing Enter sets active device
		m.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.200"))
		m.activePanel = PanelDevices
		m.deviceCursor = 1 // Second device

		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		activeDev, _ = m.Registry.GetActive()
		if activeDev.IP != "192.168.1.200" {
			t.Errorf("expected active IP 192.168.1.200 after Enter, got %s", activeDev.IP)
		}

		// Panel 2 (Control): Pressing Enter toggles power state
		m.activePanel = PanelControl
		prevState := activeDev.State
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		activeDev, _ = m.Registry.GetActive()
		if activeDev.State == prevState {
			t.Errorf("expected power state toggle on Enter in PanelControl")
		}
	})

	t.Run("Search mode filtering and cursor bounds clamping", func(t *testing.T) {
		// Enter search mode with '/'
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m = updated.(Model)
		if m.mode != ModeSearch {
			t.Fatalf("expected ModeSearch, got %v", m.mode)
		}

		// Type query "Sunset"
		for _, r := range "Sunset" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		if m.searchQuery != "Sunset" {
			t.Errorf("expected searchQuery 'Sunset', got %q", m.searchQuery)
		}

		// Press Enter to confirm search: transitions to ModeNormal and focuses PanelScenes
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if m.mode != ModeNormal || m.activePanel != PanelScenes {
			t.Errorf("expected ModeNormal and PanelScenes focus after Enter in search, got mode=%v panel=%v", m.mode, m.activePanel)
		}
	})
}
