package tui

import (
	"os"
	"path/filepath"
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
		for i, d := range m.Registry.List() {
			if d.IP == "192.168.1.200" {
				m.deviceCursor = i
				break
			}
		}

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

	t.Run("Command mode buffer, execution, and verb dispatches", func(t *testing.T) {
		// Enter command mode with ':'
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		if m.mode != ModeCommand {
			t.Fatalf("expected ModeCommand, got %v", m.mode)
		}

		// Test backspace and key appending
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = updated.(Model)
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		m = updated.(Model)
		if m.commandBuffer != "" {
			t.Errorf("expected empty buffer after backspace, got %q", m.commandBuffer)
		}

		// Test :dim 75
		for _, r := range "dim 75" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ := m.Registry.GetActive()
		if dev.Brightness != 75 {
			t.Errorf("expected brightness 75 after :dim 75, got %d", dev.Brightness)
		}

		// Test :temp 4000
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "temp 4000" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Temp != 4000 {
			t.Errorf("expected temp 4000 after :temp 4000, got %d", dev.Temp)
		}

		// Test :rgb 255 128 64
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "rgb 255 128 64" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.RGB != [3]int{255, 128, 64} {
			t.Errorf("expected RGB [255,128,64], got %v", dev.RGB)
		}

		// Test :scene Ocean
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "scene Ocean" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.SceneID != 1 {
			t.Errorf("expected SceneID 1 after :scene Ocean, got %d", dev.SceneID)
		}

		// Test :timer 20
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "timer 20" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if m.sleepTimerSecs != 20*60 {
			t.Errorf("expected sleepTimerSecs 1200, got %d", m.sleepTimerSecs)
		}

		// Test :on and :off
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "off" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.State != false {
			t.Errorf("expected state false after :off, got %v", dev.State)
		}

		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "on" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.State != true {
			t.Errorf("expected state true after :on, got %v", dev.State)
		}

		// Test :help
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "help" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if m.mode != ModeHelp {
			t.Errorf("expected ModeHelp after :help, got %v", m.mode)
		}
		// Exit help overlay
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		m = updated.(Model)
		if m.mode != ModeNormal {
			t.Errorf("expected ModeNormal after exiting help, got %v", m.mode)
		}

		// Test unknown command
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "unknowncmd" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if !strings.Contains(m.statusMessage, "Unknown command") {
			t.Errorf("expected status message for unknown command, got %q", m.statusMessage)
		}
	})

	t.Run("Navigation, shortcuts, speed adjustments, and undo stack", func(t *testing.T) {
		m.searchQuery = ""
		// Test j/k navigation in PanelDevices
		m.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.120"))
		m.activePanel = PanelDevices
		m.deviceCursor = 0
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		if m.deviceCursor != 1 {
			t.Errorf("expected deviceCursor 1 after 'j', got %d", m.deviceCursor)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = updated.(Model)
		if m.deviceCursor != 0 {
			t.Errorf("expected deviceCursor 0 after 'k', got %d", m.deviceCursor)
		}

		// Test j/k navigation in PanelControl (dimming adjustments)
		m.activePanel = PanelControl
		dev, _ := m.Registry.GetActive()
		initDim := dev.Brightness
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Brightness >= initDim {
			t.Errorf("expected brightness decrease on 'j' in PanelControl, got %d", dev.Brightness)
		}

		m.activePanel = PanelScenes
		m.sceneCursor = 0

		// Test j/k navigation in PanelScenes
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		if m.sceneCursor != 1 {
			t.Errorf("expected sceneCursor 1 after 'j', got %d", m.sceneCursor)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = updated.(Model)
		if m.sceneCursor != 0 {
			t.Errorf("expected sceneCursor 0 after 'k', got %d", m.sceneCursor)
		}

		// Test G / home / end navigation in PanelScenes
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		m = updated.(Model)
		scenesCount := len(wiz.FilterScenes(""))
		if m.sceneCursor != scenesCount-1 {
			t.Errorf("expected sceneCursor %d after 'G', got %d", scenesCount-1, m.sceneCursor)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
		m = updated.(Model)
		if m.sceneCursor != 0 {
			t.Errorf("expected sceneCursor 0 after Home, got %d", m.sceneCursor)
		}

		// Test numeric key shortcuts 1-9
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.SceneID != 3 {
			t.Errorf("expected SceneID 3 after pressing '3', got %d", dev.SceneID)
		}

		// Test speed adjustment '[' and ']'
		prevSpeed := dev.Speed
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Speed <= prevSpeed {
			t.Errorf("expected speed increase after ']', got prev %d vs new %d", prevSpeed, dev.Speed)
		}

		// Test undo ('u')
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Speed != prevSpeed {
			t.Errorf("expected speed restored to %d after undo 'u', got %d", prevSpeed, dev.Speed)
		}

		// Test sleep timer shortcut ('t')
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		m = updated.(Model)
		if m.sleepTimerSecs != 15*60 {
			t.Errorf("expected 900s sleep timer after 't', got %d", m.sleepTimerSecs)
		}

		// Test rescan shortcut ('R')
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
		m = updated.(Model)
		if !strings.Contains(m.statusMessage, "Rescanning") {
			t.Errorf("expected status message for rescan, got %q", m.statusMessage)
		}
	})

	t.Run("Extended Command mode speed, undo, ctrl+u, and Home key device sync", func(t *testing.T) {
		m.activePanel = PanelScenes
		// Trigger a scene first to set SceneID > 0
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		// Test :speed 150
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "speed 150" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ := m.Registry.GetActive()
		if dev.Speed != 150 {
			t.Errorf("expected speed 150 after :speed 150, got %d", dev.Speed)
		}

		// Test :undo
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "undo" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Speed == 150 {
			t.Errorf("expected speed restored from undo after :undo")
		}

		// Test ctrl+u in command mode
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "partialcommand" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
		m = updated.(Model)
		if m.commandBuffer != "" {
			t.Errorf("expected empty command buffer after ctrl+u, got %q", m.commandBuffer)
		}
		m.mode = ModeNormal

		// Test ctrl+u in search mode
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m = updated.(Model)
		for _, r := range "ocean" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
		m = updated.(Model)
		if m.searchQuery != "" {
			t.Errorf("expected empty search query after ctrl+u, got %q", m.searchQuery)
		}
		m.mode = ModeNormal

		// Test Home key in PanelDevices updates active bulb target
		m.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.50"))
		m.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.51"))
		devicesList := m.Registry.List()
		expectedFirstIP := devicesList[0].IP
		m.activePanel = PanelDevices
		m.deviceCursor = 1
		m.Registry.SetActive(devicesList[1].IP)

		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
		m = updated.(Model)
		if m.deviceCursor != 0 {
			t.Errorf("expected deviceCursor 0 after Home, got %d", m.deviceCursor)
		}
		activeDev, _ := m.Registry.GetActive()
		if activeDev.IP != expectedFirstIP {
			t.Errorf("expected active target IP %s after Home, got %s", expectedFirstIP, activeDev.IP)
		}

		// Test :hex #ffaa00 command in model
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "hex #ffaa00" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.RGB != [3]int{255, 170, 0} {
			t.Errorf("expected RGB [255,170,0] after :hex #ffaa00, got %v", dev.RGB)
		}

		// Test :warm preset command in model
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "warm" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.Temp != 2700 {
			t.Errorf("expected temp 2700 after :warm, got %d", dev.Temp)
		}

		// Test :name command in model
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "name Studio Desk Lamp" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		dev, _ = m.Registry.GetActive()
		if dev.Name != "Studio Desk Lamp" {
			t.Errorf("expected device name 'Studio Desk Lamp', got %q", dev.Name)
		}

		// Test :ocean scene shortcut command in model
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "ocean" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}
		dev, _ = m.Registry.GetActive()
		if dev.SceneID != 1 {
			t.Errorf("expected scene ID 1 after :ocean, got %d", dev.SceneID)
		}

		// Test clampDeviceCursor
		m.deviceCursor = 99
		m.clampDeviceCursor()
		if m.deviceCursor >= len(m.Registry.List()) {
			t.Errorf("expected clamped deviceCursor < len(list), got %d", m.deviceCursor)
		}
	})

	t.Run("Config integration and persistent device naming", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "gowiz_test_model_config.json")

		mockClient := wiz.NewMockClient()
		mCfg := NewModelWithConfig(mockClient, "192.168.1.150", cfgPath)

		// Run :name command to assign custom name
		updated, _ := mCfg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		mCfg = updated.(Model)
		for _, r := range "name Office Ambient Light" {
			updated, _ = mCfg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			mCfg = updated.(Model)
		}
		updated, _ = mCfg.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mCfg = updated.(Model)

		activeDev, ok := mCfg.Registry.GetActive()
		if !ok || activeDev.Name != "Office Ambient Light" {
			t.Fatalf("expected active device name 'Office Ambient Light', got %q", activeDev.Name)
		}

		// Run :config command
		updated, _ = mCfg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		mCfg = updated.(Model)
		for _, r := range "config" {
			updated, _ = mCfg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			mCfg = updated.(Model)
		}
		updated, _ = mCfg.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mCfg = updated.(Model)

		if !strings.Contains(mCfg.statusMessage, "Config:") {
			t.Errorf("expected status message containing Config: info, got %q", mCfg.statusMessage)
		}

		// Instantiate a NEW Model with the same config file path to verify persistence across app restarts
		mRestored := NewModelWithConfig(mockClient, "192.168.1.150", cfgPath)
		restoredDev, ok := mRestored.Registry.Get("192.168.1.150")
		if !ok || restoredDev.Name != "Office Ambient Light" {
			t.Errorf("expected restored device name 'Office Ambient Light' from persistent config, got %q", restoredDev.Name)
		}

		// Test automatic restoration of LastActiveIP when launching without initial IP flag
		mCfg.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.199"))
		mCfg.activePanel = PanelDevices
		mCfg.deviceCursor = 1
		mCfg.setActiveDevice("192.168.1.199")

		mAutoRestored := NewModelWithConfig(mockClient, "", cfgPath)
		autoActiveDev, ok := mAutoRestored.Registry.GetActive()
		if !ok || autoActiveDev.IP != "192.168.1.199" {
			t.Errorf("expected auto-restored active IP '192.168.1.199', got %v", autoActiveDev)
		}
	})

	t.Run("IP target command and recent targets in Model", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "model_ip_target_test.json")

		mockClient := wiz.NewMockClient()
		m := NewModelWithConfig(mockClient, "192.168.1.100", cfgPath)

		// Execute :ip 192.168.1.222 command
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "ip 192.168.1.222" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		activeDev, ok := m.Registry.GetActive()
		if !ok || activeDev.IP != "192.168.1.222" {
			t.Fatalf("expected active IP '192.168.1.222', got %v", activeDev)
		}

		// Execute :recent command
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "recent" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		if !strings.Contains(m.statusMessage, "Recent targets") || !strings.Contains(m.statusMessage, "192.168.1.222") {
			t.Errorf("expected status message showing recent target 192.168.1.222, got %q", m.statusMessage)
		}
	})

	t.Run("Room assignment and group command batch dispatches in Model", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "model_room_test.json")

		mockClient := wiz.NewMockClient()
		m := NewModelWithConfig(mockClient, "192.168.1.10", cfgPath)

		// Add two lights
		m.Registry.AddOrUpdate(wiz.NewDevice("192.168.1.11"))

		// Assign room 'Living Room' to active device (192.168.1.10)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "room Living Room" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		dev10, _ := m.Registry.Get("192.168.1.10")
		if dev10.Room != "Living Room" {
			t.Errorf("expected dev10 room 'Living Room', got '%s'", dev10.Room)
		}

		// Assign room 'Living Room' to second device (192.168.1.11)
		m.setActiveDevice("192.168.1.11")
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "room Living Room" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		dev11, _ := m.Registry.Get("192.168.1.11")
		if dev11.Room != "Living Room" {
			t.Errorf("expected dev11 room 'Living Room', got '%s'", dev11.Room)
		}

		// Run group command :group Living Room dim 80
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "group Living Room dim 80" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		dev10, _ = m.Registry.Get("192.168.1.10")
		dev11, _ = m.Registry.Get("192.168.1.11")
		if dev10.Brightness != 80 || dev11.Brightness != 80 {
			t.Errorf("expected both Living Room lights updated to dim 80, got dev10: %d, dev11: %d", dev10.Brightness, dev11.Brightness)
		}

		// Reload model to test room persistence
		mRestored := NewModelWithConfig(mockClient, "192.168.1.10", cfgPath)
		rDev10, _ := mRestored.Registry.Get("192.168.1.10")
		if rDev10.Room != "Living Room" {
			t.Errorf("expected restored room 'Living Room', got '%s'", rDev10.Room)
		}
	})

	t.Run("Preset management and activation in Model", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "model_preset_test.json")

		mockClient := wiz.NewMockClient()
		m := NewModelWithConfig(mockClient, "192.168.1.100", cfgPath)

		// Set active dev state to 45% dimming, 3000K temp
		activeDev, _ := m.Registry.GetActive()
		activeDev.Brightness = 45
		activeDev.Temp = 3000

		// Save current state as preset 'evening_relax'
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "preset save evening_relax" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		if !strings.Contains(m.statusMessage, "Saved") {
			t.Errorf("expected status message for saved preset, got %q", m.statusMessage)
		}

		// Execute :preset list command
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "preset list" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		if !strings.Contains(m.statusMessage, "Presets") || !strings.Contains(m.statusMessage, "evening_relax") {
			t.Errorf("expected status message listing presets including evening_relax, got %q", m.statusMessage)
		}

		// Mutate device state
		activeDev.Brightness = 100
		activeDev.Temp = 6500

		// Apply custom preset 'evening_relax'
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "preset evening_relax" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			updated, _ = m.Update(msg)
			m = updated.(Model)
		}

		activeDev, _ = m.Registry.GetActive()
		if activeDev.Brightness != 45 || activeDev.Temp != 3000 {
			t.Errorf("expected restored state 45%% dimming and 3000K temp from custom preset, got dim %d, temp %d", activeDev.Brightness, activeDev.Temp)
		}

		// Delete preset 'evening_relax'
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
		m = updated.(Model)
		for _, r := range "preset delete evening_relax" {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = updated.(Model)
		}
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updated.(Model)

		if !strings.Contains(m.statusMessage, "Deleted") {
			t.Errorf("expected status message for deleted preset, got %q", m.statusMessage)
		}
	})
}

func TestTUIFadeTransitions(t *testing.T) {
	mock := wiz.NewMockClient()
	dev := wiz.NewDevice("192.168.1.50")
	dev.State = true
	dev.Brightness = 100
	dev.Temp = 6500

	m := NewModelWithConfig(mock, "192.168.1.50", "")
	m.Registry.AddOrUpdate(dev)
	m.setActiveDevice("192.168.1.50")

	// Trigger :fade 20 10 (fade to 20% over 10 seconds)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(Model)
	for _, r := range "fade 20 10" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !m.fadeActive {
		t.Fatalf("expected fadeActive to be true after :fade command")
	}
	if m.fadeTargetDim != 20 || m.fadeDurationSecs != 10 {
		t.Errorf("expected target dim 20, duration 10, got dim %d, dur %d", m.fadeTargetDim, m.fadeDurationSecs)
	}

	// Simulate TimerTicks and verify progression
	for i := 0; i < 5; i++ {
		updated, cmd := m.Update(TimerTickMsg{})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				updated, _ = m.Update(msg)
				m = updated.(Model)
			}
		}
	}

	if m.fadeElapsedSecs != 5 {
		t.Errorf("expected 5 elapsed fade seconds after 5 ticks, got %d", m.fadeElapsedSecs)
	}

	// Finish remaining ticks to complete fade
	for i := 0; i < 6; i++ {
		updated, cmd := m.Update(TimerTickMsg{})
		m = updated.(Model)
		if cmd != nil {
			msg := cmd()
			if msg != nil {
				updated, _ = m.Update(msg)
				m = updated.(Model)
			}
		}
	}

	if m.fadeActive {
		t.Errorf("expected fadeActive to be false after duration elapsed")
	}
	if !strings.Contains(m.statusMessage, "complete") {
		t.Errorf("expected fade completion status message, got %q", m.statusMessage)
	}
}

func TestCategoryCommandInModel(t *testing.T) {
	mock := wiz.NewMockClient()
	m := NewModelWithConfig(mock, "192.168.1.50", "")

	// Dispatch :cat Nature command
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(Model)
	for _, r := range "cat Nature" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.searchQuery != "Nature" {
		t.Errorf("expected searchQuery 'Nature', got %q", m.searchQuery)
	}
	if m.activePanel != PanelScenes {
		t.Errorf("expected activePanel PanelScenes after :cat command, got %v", m.activePanel)
	}
	if !strings.Contains(m.statusMessage, "Filtered scenes by category") {
		t.Errorf("expected category status message, got %q", m.statusMessage)
	}
}

func TestInfoAndExportImportInModel(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "config.json")
	exportPath := filepath.Join(tempDir, "backup.json")

	mock := wiz.NewMockClient()
	m := NewModelWithConfig(mock, "192.168.1.50", cfgPath)

	dev := wiz.NewDevice("192.168.1.50")
	dev.MAC = "a8:bb:cc:dd:ee:ff"
	dev.Rssi = -55
	m.Registry.AddOrUpdate(dev)
	m.setActiveDevice("192.168.1.50")

	// Dispatch :info command
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(Model)
	for _, r := range "info" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !strings.Contains(m.statusMessage, "192.168.1.50") || !strings.Contains(m.statusMessage, "-55 dBm") {
		t.Errorf("expected diagnostic info in status message, got %q", m.statusMessage)
	}

	// Dispatch :export command
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(Model)
	for _, r := range "export " + exportPath {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !strings.Contains(m.statusMessage, "exported") {
		t.Errorf("expected export confirmation status message, got %q", m.statusMessage)
	}
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatalf("expected exported file to exist at %s", exportPath)
	}

	// Dispatch :import command
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(Model)
	for _, r := range "import " + exportPath {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if !strings.Contains(m.statusMessage, "imported") {
		t.Errorf("expected import confirmation status message, got %q", m.statusMessage)
	}
}

