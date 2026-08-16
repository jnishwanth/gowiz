package tui

import (
	"testing"
	"wiz-tui/internal/circadian"
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

	t.Run("Preset command dispatches", func(t *testing.T) {
		resList := ExecuteCommand("preset list", dev)
		if !resList.ListPresets {
			t.Errorf("expected ListPresets true for :preset list")
		}

		resPresets := ExecuteCommand("presets", dev)
		if !resPresets.ListPresets {
			t.Errorf("expected ListPresets true for :presets")
		}

		resSave := ExecuteCommand("preset save cozy_night", dev)
		if resSave.SavePresetName != "cozy_night" {
			t.Errorf("expected SavePresetName 'cozy_night', got '%s'", resSave.SavePresetName)
		}

		resDelete := ExecuteCommand("preset delete cozy_night", dev)
		if resDelete.DeletePresetName != "cozy_night" {
			t.Errorf("expected DeletePresetName 'cozy_night', got '%s'", resDelete.DeletePresetName)
		}

		resEvening := ExecuteCommand("preset evening", dev)
		if resEvening.PilotParams == nil || *resEvening.PilotParams.Temp != 2700 || *resEvening.PilotParams.Dimming != 60 {
			t.Errorf("expected 2700K 60%% dimming for built-in evening preset")
		}

		resCustom := ExecuteCommand("preset my_custom_mode", dev)
		if resCustom.ApplyPresetName != "my_custom_mode" {
			t.Errorf("expected ApplyPresetName 'my_custom_mode', got '%s'", resCustom.ApplyPresetName)
		}

		resGroupPreset := ExecuteCommand("group Living Room preset evening", dev)
		if resGroupPreset.TargetRoom != "Living Room" || resGroupPreset.PilotParams == nil {
			t.Errorf("expected TargetRoom 'Living Room' with PilotParams for group preset command")
		}
	})

	t.Run("Fade and Sun transition commands", func(t *testing.T) {
		resFade := ExecuteCommand("fade 20 45", dev)
		if !resFade.IsFadeCommand || resFade.SetFadeDimming != 20 || resFade.SetFadeDuration != 45 {
			t.Errorf("expected fade dimming 20, duration 45, got dim: %d, dur: %d", resFade.SetFadeDimming, resFade.SetFadeDuration)
		}

		resFadeOff := ExecuteCommand("fade off 30", dev)
		if !resFadeOff.IsFadeCommand || !resFadeOff.FadeTurnOff || resFadeOff.SetFadeDuration != 30 {
			t.Errorf("expected fade off turnOff true, duration 30")
		}

		resSunrise := ExecuteCommand("sunrise 120", dev)
		if !resSunrise.IsFadeCommand || resSunrise.SetFadeDimming != 100 || resSunrise.SetFadeColorTemp != 4200 || resSunrise.SetFadeDuration != 120 {
			t.Errorf("expected sunrise 100%% 4200K over 120s")
		}

		resSunset := ExecuteCommand("sunset 60", dev)
		if !resSunset.IsFadeCommand || !resSunset.FadeTurnOff || resSunset.SetFadeColorTemp != 2700 || resSunset.SetFadeDuration != 60 {
			t.Errorf("expected sunset fade turnOff true, 2700K over 60s")
		}
	})

	t.Run("Category command dispatches", func(t *testing.T) {
		resList := ExecuteCommand("categories", dev)
		if !resList.ListCategories {
			t.Errorf("expected ListCategories true for :categories")
		}

		resCatList := ExecuteCommand("cat", dev)
		if !resCatList.ListCategories {
			t.Errorf("expected ListCategories true for empty :cat")
		}

		resCategoryNature := ExecuteCommand("category Nature", dev)
		if resCategoryNature.SetSearchQuery != "Nature" || !resCategoryNature.FocusScenes {
			t.Errorf("expected SetSearchQuery 'Nature' and FocusScenes true for :category Nature, got SetSearchQuery: '%s', FocusScenes: %v",
				resCategoryNature.SetSearchQuery, resCategoryNature.FocusScenes)
		}

		resCatCozy := ExecuteCommand("cat Cozy", dev)
		if resCatCozy.SetSearchQuery != "Cozy" || !resCatCozy.FocusScenes {
			t.Errorf("expected SetSearchQuery 'Cozy' and FocusScenes true for :cat Cozy")
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

	t.Run("Info, Diag, Export, and Import command dispatches", func(t *testing.T) {
		resInfo := ExecuteCommand("info", dev)
		if !resInfo.ShowInfo {
			t.Errorf("expected ShowInfo true for :info command")
		}

		resDiag := ExecuteCommand("diag", dev)
		if !resDiag.ShowInfo {
			t.Errorf("expected ShowInfo true for :diag command")
		}

		resStatus := ExecuteCommand("status", dev)
		if !resStatus.ShowInfo {
			t.Errorf("expected ShowInfo true for :status command")
		}

		resExport := ExecuteCommand("export my_backup.json", dev)
		if resExport.ExportPath != "my_backup.json" {
			t.Errorf("expected ExportPath 'my_backup.json', got '%s'", resExport.ExportPath)
		}

		resExportDef := ExecuteCommand("export", dev)
		if resExportDef.ExportPath != "gowiz-config-backup.json" {
			t.Errorf("expected default ExportPath 'gowiz-config-backup.json', got '%s'", resExportDef.ExportPath)
		}

		resImport := ExecuteCommand("import my_backup.json", dev)
		if resImport.ImportPath != "my_backup.json" {
			t.Errorf("expected ImportPath 'my_backup.json', got '%s'", resImport.ImportPath)
		}

		resImportEmpty := ExecuteCommand("import", dev)
		if resImportEmpty.StatusMsg != "Usage: :import <filepath>" {
			t.Errorf("expected usage message for empty :import args, got '%s'", resImportEmpty.StatusMsg)
		}
	})

	t.Run("Shell completion command dispatches", func(t *testing.T) {
		resDefault := ExecuteCommand("completion", dev)
		if resDefault.CompletionShell != "bash" {
			t.Errorf("expected default shell 'bash', got %q", resDefault.CompletionShell)
		}

		resZsh := ExecuteCommand("completion zsh", dev)
		if resZsh.CompletionShell != "zsh" {
			t.Errorf("expected shell 'zsh', got %q", resZsh.CompletionShell)
		}

		resFish := ExecuteCommand("completion fish", dev)
		if resFish.CompletionShell != "fish" {
			t.Errorf("expected shell 'fish', got %q", resFish.CompletionShell)
		}

		resInvalid := ExecuteCommand("completion powershell", dev)
		if resInvalid.CompletionShell != "" {
			t.Errorf("expected empty CompletionShell on invalid shell input")
		}
		if resInvalid.StatusMsg == "" {
			t.Errorf("expected status message error on invalid shell input")
		}
	})

	t.Run("Circadian rhythm command dispatches", func(t *testing.T) {
		resNow := ExecuteCommand("circadian", dev)
		if resNow.PilotParams == nil {
			t.Fatalf("expected PilotParams for default :circadian command")
		}
		if resNow.FadeLabel == "" {
			t.Errorf("expected non-empty FadeLabel badge for circadian command")
		}

		resTime := ExecuteCommand("rhythm 14:30", dev)
		if resTime.PilotParams == nil {
			t.Fatalf("expected PilotParams for :rhythm 14:30 command")
		}

		resInfo := ExecuteCommand("circadian info", dev)
		if resInfo.StatusMsg == "" {
			t.Errorf("expected StatusMsg for :circadian info command")
		}

		resErr := ExecuteCommand("circadian 99:99", dev)
		if resErr.PilotParams != nil {
			t.Errorf("expected nil PilotParams on invalid time arg")
		}

		customPhases := []circadian.SchedulePhase{
			{Name: "Custom Dawn", StartHour: 0, EndHour: 12, StartTemp: 2700, EndTemp: 6000, StartDimming: 50, EndDimming: 100},
			{Name: "Custom Dusk", StartHour: 12, EndHour: 24, StartTemp: 6000, EndTemp: 2200, StartDimming: 100, EndDimming: 30},
		}
		resCustom := ExecuteCommandWithPhases("circadian 14:00", dev, customPhases)
		if resCustom.PilotParams == nil {
			t.Fatalf("expected PilotParams for ExecuteCommandWithPhases")
		}
		if resCustom.FadeLabel != "🌅 Custom Dusk" {
			t.Errorf("expected FadeLabel '🌅 Custom Dusk', got %q", resCustom.FadeLabel)
		}
	})
}
