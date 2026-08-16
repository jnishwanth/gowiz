package wiz

import (
	"testing"
)

func TestDeviceRegistry(t *testing.T) {
	reg := NewDeviceRegistry()

	t.Run("Default state has FallbackIP active", func(t *testing.T) {
		active, ok := reg.GetActive()
		if !ok || active.IP != FallbackIP {
			t.Errorf("expected active target FallbackIP, got %v", active)
		}
	})

	t.Run("AddOrUpdate new device", func(t *testing.T) {
		dev2 := NewDevice("192.168.1.120")
		reg.AddOrUpdate(dev2)

		devices := reg.List()
		if len(devices) != 2 {
			t.Errorf("expected 2 devices in list, got %d", len(devices))
		}
	})

	t.Run("SetActive target", func(t *testing.T) {
		success := reg.SetActive("192.168.1.120")
		if !success {
			t.Errorf("failed to set active target to 192.168.1.120")
		}

		active, _ := reg.GetActive()
		if active.IP != "192.168.1.120" {
			t.Errorf("expected active IP 192.168.1.120, got %s", active.IP)
		}
	})

	t.Run("SetName custom name", func(t *testing.T) {
		ok := reg.SetName("192.168.1.120", "Desk Lamp")
		if !ok {
			t.Errorf("expected SetName to succeed for existing IP")
		}
		dev, _ := reg.Get("192.168.1.120")
		if dev.Name != "Desk Lamp" {
			t.Errorf("expected device name 'Desk Lamp', got '%s'", dev.Name)
		}

		fail := reg.SetName("10.0.0.99", "Non Existent")
		if fail {
			t.Errorf("expected SetName to return false for non-existent IP")
		}
	})

	t.Run("Multi-selection logic", func(t *testing.T) {
		reg.ToggleSelection("192.168.1.120")
		selected := reg.GetSelectedOrActive()
		if len(selected) != 1 || selected[0].IP != "192.168.1.120" {
			t.Errorf("expected selected device 192.168.1.120, got %v", selected)
		}

		reg.SelectAll()
		allSelected := reg.GetSelectedOrActive()
		if len(allSelected) != 2 {
			t.Errorf("expected all 2 devices selected, got %d", len(allSelected))
		}

		reg.ClearSelections()
		afterClear := reg.GetSelectedOrActive()
		// Should fall back to active device
		if len(afterClear) != 1 || afterClear[0].IP != "192.168.1.120" {
			t.Errorf("expected fallback to active device after clear selections")
		}
	})

	t.Run("UpdateFromPilot", func(t *testing.T) {
		dev := NewDevice("192.168.1.100")
		state := true
		dim := 75
		r, g, b := 255, 100, 50
		temp := 3500
		scene := 2
		rssi := -55

		dev.UpdateFromPilot(PilotParams{
			State:   &state,
			Dimming: &dim,
			R:       &r, G: &g, B: &b,
			Temp:    &temp,
			SceneID: &scene,
			Rssi:    &rssi,
		})

		if dev.Brightness != 75 || dev.RGB != [3]int{255, 100, 50} || dev.Temp != 3500 || dev.SceneID != 2 || dev.Rssi != -55 {
			t.Errorf("device pilot update failed: %+v", dev)
		}
	})

	t.Run("ApplyAliases mapping", func(t *testing.T) {
		regAliases := NewDeviceRegistry()
		regAliases.ApplyAliases(map[string]string{
			"192.168.1.200": "Kitchen Ceiling",
		})

		devNew := NewDevice("192.168.1.200")
		regAliases.AddOrUpdate(devNew)

		fetched, ok := regAliases.Get("192.168.1.200")
		if !ok || fetched.Name != "Kitchen Ceiling" {
			t.Errorf("expected alias 'Kitchen Ceiling', got '%s' (ok: %v)", fetched.Name, ok)
		}
	})
}
