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

	t.Run("ApplyRooms, SetRoom, and GetDevicesByRoom", func(t *testing.T) {
		regRooms := NewDeviceRegistry()
		regRooms.ApplyRooms(map[string]string{
			"192.168.1.210": "Living Room",
		})

		dev1 := NewDevice("192.168.1.210")
		dev2 := NewDevice("192.168.1.211")
		regRooms.AddOrUpdate(dev1)
		regRooms.AddOrUpdate(dev2)

		regRooms.SetRoom("192.168.1.211", "living room") // case insensitive matching test

		livingDevices := regRooms.GetDevicesByRoom("Living Room")
		if len(livingDevices) != 2 {
			t.Errorf("expected 2 devices in Living Room, got %d", len(livingDevices))
		}

		dev, _ := regRooms.Get("192.168.1.210")
		if dev.Room != "Living Room" {
			t.Errorf("expected device room 'Living Room', got '%s'", dev.Room)
		}
	})

	t.Run("SignalPercentage, SignalQuality, and SignalBar", func(t *testing.T) {
		dev := NewDevice("192.168.1.30")
		if dev.SignalPercentage() != 0 || dev.SignalQuality() != "Unknown" || dev.SignalBar() != "📶 Online" {
			t.Errorf("expected zero RSSI defaults, got %d, %s, %s", dev.SignalPercentage(), dev.SignalQuality(), dev.SignalBar())
		}

		dev.Rssi = -40
		if dev.SignalPercentage() != 83 || dev.SignalQuality() != "Excellent" {
			t.Errorf("expected -40 dBm to be 83%% Excellent, got %d%% %s", dev.SignalPercentage(), dev.SignalQuality())
		}

		dev.Rssi = -50
		if dev.SignalQuality() != "Good" {
			t.Errorf("expected -50 dBm to be Good quality, got %s", dev.SignalQuality())
		}

		dev.Rssi = -65
		if dev.SignalQuality() != "Fair" {
			t.Errorf("expected -65 dBm to be Fair quality, got %s", dev.SignalQuality())
		}

		dev.Rssi = -100
		if dev.SignalPercentage() != 0 || dev.SignalQuality() != "Weak" {
			t.Errorf("expected -100 dBm clamped to 0%% Weak, got %d%% %s", dev.SignalPercentage(), dev.SignalQuality())
		}

		dev.Rssi = -20
		if dev.SignalPercentage() != 100 || dev.SignalQuality() != "Excellent" {
			t.Errorf("expected -20 dBm clamped to 100%% Excellent, got %d%% %s", dev.SignalPercentage(), dev.SignalQuality())
		}
	})

	t.Run("GetDevicesByGroup and GetDevicesByGroupOrRoom", func(t *testing.T) {
		regGroup := NewDeviceRegistry()
		d1 := NewDevice("192.168.1.50")
		d1.Name = "Desk Light 1"
		d2 := NewDevice("192.168.1.51")
		d2.Room = "Office"

		regGroup.AddOrUpdate(d1)
		regGroup.AddOrUpdate(d2)

		groups := map[string][]string{
			"desk": {"192.168.1.50"},
		}

		gDevs := regGroup.GetDevicesByGroup("Desk", groups)
		if len(gDevs) != 1 || gDevs[0].IP != "192.168.1.50" {
			t.Errorf("expected 1 group device, got %v", gDevs)
		}

		// Test fallback to room via GetDevicesByGroupOrRoom
		rDevs := regGroup.GetDevicesByGroupOrRoom("Office", groups)
		if len(rDevs) != 1 || rDevs[0].IP != "192.168.1.51" {
			t.Errorf("expected fallback room device 192.168.1.51, got %v", rDevs)
		}
	})

	t.Run("GetDevicesBySelector with all/wildcard targets", func(t *testing.T) {
		regAll := NewDeviceRegistry()
		d1 := NewDevice("192.168.1.10")
		d2 := NewDevice("192.168.1.11")
		regAll.AddOrUpdate(d1)
		regAll.AddOrUpdate(d2)

		groups := map[string][]string{"office": {"192.168.1.10"}}

		for _, wildcard := range []string{"all", "ALL", "*", "everyone", "broadcast"} {
			devs := regAll.GetDevicesBySelector(wildcard, groups)
			if len(devs) != 3 {
				t.Errorf("expected 3 devices for wildcard %q, got %d", wildcard, len(devs))
			}
		}

		groupDevs := regAll.GetDevicesBySelector("office", groups)
		if len(groupDevs) != 1 || groupDevs[0].IP != "192.168.1.10" {
			t.Errorf("expected 1 device for group 'office', got %v", groupDevs)
		}

		// Test direct IP matching
		ipDevs := regAll.GetDevicesBySelector("192.168.1.11", groups)
		if len(ipDevs) != 1 || ipDevs[0].IP != "192.168.1.11" {
			t.Errorf("expected 1 device for direct IP '192.168.1.11', got %v", ipDevs)
		}

		// Test direct MAC and Name matching
		d1.MAC = "a8bb50123456"
		d1.Name = "Reading Lamp"
		macDevs := regAll.GetDevicesBySelector("A8BB50123456", groups)
		if len(macDevs) != 1 || macDevs[0].IP != "192.168.1.10" {
			t.Errorf("expected 1 device for MAC 'A8BB50123456', got %v", macDevs)
		}

		nameDevs := regAll.GetDevicesBySelector("reading lamp", groups)
		if len(nameDevs) != 1 || nameDevs[0].IP != "192.168.1.10" {
			t.Errorf("expected 1 device for Name 'reading lamp', got %v", nameDevs)
		}

		// Test empty and unmatched targets
		if devs := regAll.GetDevicesBySelector("", groups); devs != nil {
			t.Errorf("expected nil for empty target, got %v", devs)
		}
		if devs := regAll.GetDevicesBySelector("nonexistent", groups); devs != nil {
			t.Errorf("expected nil for non-existent target, got %v", devs)
		}
	})
}
