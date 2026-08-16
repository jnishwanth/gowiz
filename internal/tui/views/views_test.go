package views

import (
	"strings"
	"testing"

	"wiz-tui/internal/wiz"
)

func TestViewsRendering(t *testing.T) {
	t.Run("RenderScrollbar", func(t *testing.T) {
		lines := RenderScrollbar(10, 20, 5, 0)
		if len(lines) != 10 {
			t.Errorf("expected 10 scrollbar lines, got %d", len(lines))
		}

		noScroll := RenderScrollbar(10, 5, 10, 0)
		if noScroll != nil {
			t.Errorf("expected nil scrollbar when totalItems <= visibleLines")
		}
	})

	t.Run("RenderSectionHeader", func(t *testing.T) {
		hdrFocused := RenderSectionHeader("Bulbs", true, 30)
		if !strings.Contains(hdrFocused, "Bulbs") || !strings.Contains(hdrFocused, "[ACTIVE]") {
			t.Errorf("expected title and ACTIVE tag in focused header, got %q", hdrFocused)
		}

		hdrUnfocused := RenderSectionHeader("Bulbs", false, 30)
		if !strings.Contains(hdrUnfocused, "Bulbs") || strings.Contains(hdrUnfocused, "[ACTIVE]") {
			t.Errorf("expected title without ACTIVE tag in unfocused header, got %q", hdrUnfocused)
		}
	})

	t.Run("RenderDeviceList", func(t *testing.T) {
		regEmpty := &wiz.DeviceRegistry{}
		emptyOutput := RenderDeviceList(regEmpty, 0, true, 40, 20)
		if !strings.Contains(emptyOutput, "No WiZ lights") {
			t.Errorf("expected empty prompt when no devices registered")
		}

		reg := wiz.NewDeviceRegistry()
		dev1 := wiz.NewDevice("192.168.1.100")
		dev1.State = true
		dev1.Selected = true
		reg.AddOrUpdate(dev1)

		dev2 := wiz.NewDevice("192.168.1.101")
		dev2.Online = false
		reg.AddOrUpdate(dev2)

		dev3 := wiz.NewDevice("192.168.1.102")
		dev3.Name = "Studio Lamp"
		reg.AddOrUpdate(dev3)

		populatedOutput := RenderDeviceList(reg, 0, true, 40, 20)
		if !strings.Contains(populatedOutput, "192.168.1.100") {
			t.Errorf("expected IP in rendered device list output")
		}
		if !strings.Contains(populatedOutput, "Studio Lamp") {
			t.Errorf("expected custom device name 'Studio Lamp' in rendered device list output")
		}
	})

	t.Run("RenderControlPanel", func(t *testing.T) {
		nilDevOutput := RenderControlPanel(nil, false, 0, 40, 20)
		if !strings.Contains(nilDevOutput, "No active device selected") {
			t.Errorf("expected no device message when dev is nil")
		}

		dev := wiz.NewDevice("192.168.1.105")
		dev.Name = "Reading Nook"
		dev.State = true
		dev.Brightness = 75
		dev.SceneID = 1 // Ocean scene

		sceneOutput := RenderControlPanel(dev, true, 300, 40, 20)
		if !strings.Contains(sceneOutput, "Reading Nook") || !strings.Contains(sceneOutput, "Ocean") || !strings.Contains(sceneOutput, "75%") {
			t.Errorf("expected custom device alias 'Reading Nook', scene name, and brightness in control panel")
		}

		dev.SceneID = 0
		dev.Temp = 3000 // White temp mode
		tempOutput := RenderControlPanel(dev, true, 0, 40, 20)
		if !strings.Contains(tempOutput, "3000K") {
			t.Errorf("expected temp K value in control panel output")
		}

		dev.Temp = 0
		dev.RGB = [3]int{255, 0, 128} // RGB mode
		rgbOutput := RenderControlPanel(dev, true, 0, 40, 20)
		if !strings.Contains(rgbOutput, "255") {
			t.Errorf("expected RGB values in control panel output")
		}
	})

	t.Run("RenderScenePicker", func(t *testing.T) {
		pickerOutput := RenderScenePicker("", 1, 0, true, 2, 40, 20)
		if !strings.Contains(pickerOutput, "Ocean") || !strings.Contains(pickerOutput, "[ACTIVE]") {
			t.Errorf("expected Ocean scene and ACTIVE badge in scene picker")
		}

		filteredOutput := RenderScenePicker("NonExistentSceneQuery", 0, 0, false, 0, 40, 20)
		if !strings.Contains(filteredOutput, "No scenes match query") {
			t.Errorf("expected no scenes message for invalid filter query")
		}
	})

	t.Run("RenderStatusBar", func(t *testing.T) {
		dev := wiz.NewDevice("192.168.1.110")
		bar := RenderStatusBar("NORMAL", "1. Bulbs", dev, 0, "", "Scanning completed", 80)
		if !strings.Contains(bar, "192.168.1.110") || !strings.Contains(bar, "Scanning completed") {
			t.Errorf("expected active IP and status message in status bar")
		}

		cmdBar := RenderStatusBar("COMMAND", "1. Bulbs", dev, 0, "dim 50", "", 80)
		if !strings.Contains(cmdBar, ":dim 50") {
			t.Errorf("expected command string in status bar")
		}

		searchBar := RenderStatusBar("SEARCH", "3. Scenes", dev, 0, "ocean", "", 80)
		if !strings.Contains(searchBar, "/ocean") {
			t.Errorf("expected search prompt string in status bar")
		}
	})

	t.Run("RenderHelpOverlay", func(t *testing.T) {
		helpOutput := RenderHelpOverlay(80, 24)
		if !strings.Contains(helpOutput, "gowiz Keyboard Reference") || !strings.Contains(helpOutput, "Navigation & Focus") {
			t.Errorf("expected title and section headers in help overlay output")
		}
	})
}
