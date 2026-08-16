package tui

import (
	"testing"
	"wiz-tui/internal/wiz"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input       string
		expectR     int
		expectG     int
		expectB     int
		shouldError bool
	}{
		{"#FF5500", 255, 85, 0, false},
		{"ff5500", 255, 85, 0, false},
		{"#00E5FF", 0, 229, 255, false},
		{"#F50", 255, 85, 0, false},
		{"f50", 255, 85, 0, false},
		{"invalid", 0, 0, 0, true},
		{"#GGGGGG", 0, 0, 0, true},
		{"#12", 0, 0, 0, true},
	}

	for _, tt := range tests {
		r, g, b, err := ParseHexColor(tt.input)
		if tt.shouldError {
			if err == nil {
				t.Errorf("ParseHexColor(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseHexColor(%q) unexpected error: %v", tt.input, err)
			}
			if r != tt.expectR || g != tt.expectG || b != tt.expectB {
				t.Errorf("ParseHexColor(%q) = (%d, %d, %d), expected (%d, %d, %d)",
					tt.input, r, g, b, tt.expectR, tt.expectG, tt.expectB)
			}
		}
	}
}

func TestCommandRegistryVerbs(t *testing.T) {
	dev := wiz.NewDevice("192.168.1.100")

	t.Run("Hex color command dispatch", func(t *testing.T) {
		res := ExecuteCommand("hex #FF5500", dev)
		if res.PilotParams == nil {
			t.Fatalf("expected PilotParams for :hex command")
		}
		if *res.PilotParams.R != 255 || *res.PilotParams.G != 85 || *res.PilotParams.B != 0 {
			t.Errorf("expected RGB (255, 85, 0), got (%d, %d, %d)",
				*res.PilotParams.R, *res.PilotParams.G, *res.PilotParams.B)
		}

		resErr := ExecuteCommand("hex badhex", dev)
		if resErr.PilotParams != nil {
			t.Errorf("expected nil PilotParams on bad hex input")
		}
	})

	t.Run("Kelvin color temperature presets", func(t *testing.T) {
		warm := ExecuteCommand("warm", dev)
		if warm.PilotParams == nil || *warm.PilotParams.Temp != 2700 {
			t.Errorf("expected 2700K for :warm command")
		}

		cool := ExecuteCommand("cool", dev)
		if cool.PilotParams == nil || *cool.PilotParams.Temp != 4200 {
			t.Errorf("expected 4200K for :cool command")
		}

		daylight := ExecuteCommand("daylight", dev)
		if daylight.PilotParams == nil || *daylight.PilotParams.Temp != 6500 {
			t.Errorf("expected 6500K for :daylight command")
		}
	})

	t.Run("Power toggle command", func(t *testing.T) {
		dev.State = true
		res := ExecuteCommand("toggle", dev)
		if res.PilotParams == nil || *res.PilotParams.State != false {
			t.Errorf("expected state false when toggling ON light")
		}

		dev.State = false
		resOff := ExecuteCommand("toggle", dev)
		if resOff.PilotParams == nil || *resOff.PilotParams.State != true {
			t.Errorf("expected state true when toggling OFF light")
		}
	})

	t.Run("Device naming command", func(t *testing.T) {
		res := ExecuteCommand("name Desk Light", dev)
		if res.NewDeviceName != "Desk Light" {
			t.Errorf("expected NewDeviceName 'Desk Light', got %q", res.NewDeviceName)
		}

		resUsage := ExecuteCommand("name", dev)
		if resUsage.StatusMsg != "Usage: :name <custom name>" {
			t.Errorf("expected usage message on empty args")
		}

		resRename := ExecuteCommand("rename Reading Lamp", dev)
		if resRename.NewDeviceName != "Reading Lamp" {
			t.Errorf("expected NewDeviceName 'Reading Lamp' via :rename alias")
		}
	})

	t.Run("Direct scene shortcut commands", func(t *testing.T) {
		ocean := ExecuteCommand("ocean", dev)
		if ocean.PilotParams == nil || *ocean.PilotParams.SceneID != 1 {
			t.Errorf("expected SceneID 1 for :ocean command, got %v", ocean.PilotParams)
		}

		fireplace := ExecuteCommand("fireplace", dev)
		if fireplace.PilotParams == nil || *fireplace.PilotParams.SceneID != 5 {
			t.Errorf("expected SceneID 5 for :fireplace command, got %v", fireplace.PilotParams)
		}
	})

	t.Run("Config command dispatch", func(t *testing.T) {
		res := ExecuteCommand("config", dev)
		if !res.ConfigInfo {
			t.Errorf("expected ConfigInfo true for :config command")
		}
	})

	t.Run("IP target command dispatches", func(t *testing.T) {
		resValid := ExecuteCommand("ip 192.168.1.200", dev)
		if resValid.TargetIP != "192.168.1.200" {
			t.Errorf("expected TargetIP '192.168.1.200', got '%s'", resValid.TargetIP)
		}

		resConnect := ExecuteCommand("connect 10.0.0.15", dev)
		if resConnect.TargetIP != "10.0.0.15" {
			t.Errorf("expected TargetIP '10.0.0.15' via :connect, got '%s'", resConnect.TargetIP)
		}

		resAdd := ExecuteCommand("add 172.16.0.4", dev)
		if resAdd.TargetIP != "172.16.0.4" {
			t.Errorf("expected TargetIP '172.16.0.4' via :add, got '%s'", resAdd.TargetIP)
		}

		resInvalid := ExecuteCommand("ip not.an.ip.addr", dev)
		if resInvalid.TargetIP != "" {
			t.Errorf("expected empty TargetIP for invalid IP address, got '%s'", resInvalid.TargetIP)
		}

		resEmpty := ExecuteCommand("ip", dev)
		if resEmpty.StatusMsg != "Usage: :ip <ip address>" {
			t.Errorf("expected usage message for empty :ip args, got '%s'", resEmpty.StatusMsg)
		}
	})

	t.Run("Recent target command dispatch", func(t *testing.T) {
		res := ExecuteCommand("recent", dev)
		if !res.ShowRecent {
			t.Errorf("expected ShowRecent true for :recent command")
		}
	})

	t.Run("Room and Group command dispatches", func(t *testing.T) {
		resRoom := ExecuteCommand("room Living Room", dev)
		if resRoom.SetDeviceRoom != "Living Room" {
			t.Errorf("expected SetDeviceRoom 'Living Room', got '%s'", resRoom.SetDeviceRoom)
		}

		resClear := ExecuteCommand("room clear", dev)
		if resClear.SetDeviceRoom != "CLEAR" {
			t.Errorf("expected SetDeviceRoom 'CLEAR' for :room clear, got '%s'", resClear.SetDeviceRoom)
		}

		resGroupOn := ExecuteCommand("group Living Room on", dev)
		if resGroupOn.TargetRoom != "Living Room" || resGroupOn.PilotParams == nil || *resGroupOn.PilotParams.State != true {
			t.Errorf("expected group ON command target 'Living Room', got TargetRoom: '%s'", resGroupOn.TargetRoom)
		}

		resGroupDim := ExecuteCommand("group Master Bedroom dim 60", dev)
		if resGroupDim.TargetRoom != "Master Bedroom" || resGroupDim.PilotParams == nil || *resGroupDim.PilotParams.Dimming != 60 {
			t.Errorf("expected group dim 60 command target 'Master Bedroom', got TargetRoom: '%s'", resGroupDim.TargetRoom)
		}
	})

	t.Run("Custom verb registration", func(t *testing.T) {
		reg := NewCommandRegistry()
		reg.Register("custom", func(args []string, activeDev *wiz.Device) CommandActionResult {
			return CommandActionResult{StatusMsg: "Custom handler executed"}
		})

		res := reg.Execute("custom", dev)
		if res.StatusMsg != "Custom handler executed" {
			t.Errorf("expected custom handler response, got %q", res.StatusMsg)
		}
	})
}
