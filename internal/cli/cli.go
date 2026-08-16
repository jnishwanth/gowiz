package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"wiz-tui/internal/config"
	"wiz-tui/internal/tui"
	"wiz-tui/internal/wiz"
)

// Options specifies runtime settings for non-interactive CLI execution.
type Options struct {
	ConfigPath string
	TargetIP   string
	Mock       bool
	Command    string
	Writer     io.Writer
}

// Run executes a non-interactive command string against WiZ smart lights and configuration.
func Run(ctx context.Context, opts Options) error {
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}

	cfgMgr := config.NewManager(opts.ConfigPath)
	_ = cfgMgr.Load()
	cfg := cfgMgr.GetConfig()

	var client wiz.Client
	if opts.Mock {
		client = wiz.NewMockClient()
	} else {
		client = wiz.NewUDPClient()
	}

	reg := wiz.NewDeviceRegistry()

	for ip := range cfg.DeviceAliases {
		if net.ParseIP(ip) != nil {
			reg.AddOrUpdate(wiz.NewDevice(ip))
		}
	}
	for ip := range cfg.DeviceRooms {
		if net.ParseIP(ip) != nil {
			reg.AddOrUpdate(wiz.NewDevice(ip))
		}
	}
	for _, recIP := range cfg.RecentIPs {
		if recIP != "" && recIP != wiz.FallbackIP {
			reg.AddOrUpdate(wiz.NewDevice(recIP))
		}
	}

	reg.ApplyAliases(cfg.DeviceAliases)
	reg.ApplyRooms(cfg.DeviceRooms)

	targetIP := opts.TargetIP
	if targetIP == "" || targetIP == wiz.FallbackIP {
		if cfg.LastActiveIP != "" {
			targetIP = cfg.LastActiveIP
		}
	}

	if targetIP != "" && targetIP != wiz.FallbackIP {
		reg.AddOrUpdate(wiz.NewDevice(targetIP))
		reg.SetActive(targetIP)
	}

	activeDev, _ := reg.GetActive()

	cmdStr := strings.TrimPrefix(strings.TrimSpace(opts.Command), ":")
	if cmdStr == "" {
		return fmt.Errorf("no command specified")
	}

	result := tui.ExecuteCommand(cmdStr, activeDev)

	// Resolve preset params if custom or builtin preset requested
	if result.ApplyPresetName != "" && result.PilotParams == nil {
		presetName := strings.ToLower(result.ApplyPresetName)
		if bp, found := tui.BuiltinPresets[presetName]; found {
			params := tui.PresetToPilotParams(bp)
			result.PilotParams = &params
		} else if p, found := cfgMgr.GetPreset(presetName); found {
			params := tui.PresetToPilotParams(p)
			result.PilotParams = &params
		} else {
			return fmt.Errorf("preset '%s' not found", presetName)
		}
	}

	// Handle telemetry info request
	if result.ShowInfo {
		if activeDev == nil {
			return fmt.Errorf("no active bulb target configured")
		}
		pilot, err := client.GetPilot(ctx, activeDev.IP)
		if err != nil {
			return fmt.Errorf("failed to query telemetry from %s: %w", activeDev.IP, err)
		}
		activeDev.UpdateFromPilot(*pilot)
		fmt.Fprintf(w, "Bulb Telemetry (%s):\n", activeDev.IP)
		fmt.Fprintf(w, "  IP: %s\n", activeDev.IP)
		if activeDev.Name != "" {
			fmt.Fprintf(w, "  Name: %s\n", activeDev.Name)
		}
		if activeDev.Room != "" {
			fmt.Fprintf(w, "  Room: %s\n", activeDev.Room)
		}
		fmt.Fprintf(w, "  State: %v\n", activeDev.State)
		fmt.Fprintf(w, "  Brightness: %d%%\n", activeDev.Brightness)
		fmt.Fprintf(w, "  Signal: %s (%d%%)\n", activeDev.SignalQuality(), activeDev.SignalPercentage())
		return nil
	}

	// Handle export
	if result.ExportPath != "" {
		if err := cfgMgr.ExportToFile(result.ExportPath); err != nil {
			return fmt.Errorf("failed to export configuration: %w", err)
		}
		fmt.Fprintf(w, "Configuration exported successfully to %s\n", result.ExportPath)
		return nil
	}

	// Handle import
	if result.ImportPath != "" {
		if err := cfgMgr.ImportFromFile(result.ImportPath); err != nil {
			return fmt.Errorf("failed to import configuration: %w", err)
		}
		fmt.Fprintf(w, "Configuration imported successfully from %s\n", result.ImportPath)
		return nil
	}

	// Handle preset list
	if result.ListPresets {
		presets := cfgMgr.GetPresets()
		fmt.Fprintln(w, "Available Presets:")
		for name := range tui.BuiltinPresets {
			fmt.Fprintf(w, "  - %s (builtin)\n", name)
		}
		for name := range presets {
			if _, builtin := tui.BuiltinPresets[name]; !builtin {
				fmt.Fprintf(w, "  - %s (custom)\n", name)
			}
		}
		return nil
	}

	// Handle category list
	if result.ListCategories {
		fmt.Fprintln(w, "Scene Categories:")
		for _, cat := range wiz.GetSceneCategories() {
			fmt.Fprintf(w, "  - %s\n", cat)
		}
		return nil
	}

	// Handle recent target IPs display
	if result.ShowRecent {
		recent := cfg.RecentIPs
		if len(recent) == 0 {
			fmt.Fprintln(w, "No recent target IPs recorded.")
		} else {
			fmt.Fprintln(w, "Recent Target IPs:")
			for _, ip := range recent {
				fmt.Fprintf(w, "  - %s\n", ip)
			}
		}
		return nil
	}

	// Handle config info summary
	if result.ConfigInfo {
		fmt.Fprintf(w, "Configuration File: %s\n", cfgMgr.FilePath())
		fmt.Fprintf(w, "Last Active IP: %s\n", cfg.LastActiveIP)
		fmt.Fprintf(w, "Recent Target IPs: %d\n", len(cfg.RecentIPs))
		fmt.Fprintf(w, "Device Aliases: %d\n", len(cfg.DeviceAliases))
		fmt.Fprintf(w, "Device Rooms: %d\n", len(cfg.DeviceRooms))
		fmt.Fprintf(w, "Custom Presets: %d\n", len(cfg.Presets))
		return nil
	}

	// Handle device custom naming
	if result.NewDeviceName != "" {
		if activeDev == nil {
			return fmt.Errorf("no active bulb target to rename")
		}
		reg.SetName(activeDev.IP, result.NewDeviceName)
		_ = cfgMgr.SetAlias(activeDev.IP, result.NewDeviceName)
		if activeDev.MAC != "" {
			_ = cfgMgr.SetAlias(activeDev.MAC, result.NewDeviceName)
		}
		fmt.Fprintf(w, "Updated device name for %s to '%s'\n", activeDev.IP, result.NewDeviceName)
		return nil
	}

	// Handle room assignment
	if result.SetDeviceRoom != "" {
		if activeDev == nil {
			return fmt.Errorf("no active bulb target to assign room")
		}
		room := result.SetDeviceRoom
		if room == "CLEAR" {
			room = ""
		}
		reg.SetRoom(activeDev.IP, room)
		_ = cfgMgr.SetRoom(activeDev.IP, room)
		if activeDev.MAC != "" {
			_ = cfgMgr.SetRoom(activeDev.MAC, room)
		}
		if room == "" {
			fmt.Fprintf(w, "Cleared room assignment for %s\n", activeDev.IP)
		} else {
			fmt.Fprintf(w, "Assigned %s to room '%s'\n", activeDev.IP, room)
		}
		return nil
	}

	// Handle preset saving
	if result.SavePresetName != "" {
		if activeDev == nil {
			return fmt.Errorf("no active device snapshot to save preset from")
		}
		snap := tui.PresetFromDevice(activeDev)
		_ = cfgMgr.SetPreset(result.SavePresetName, snap)
		fmt.Fprintf(w, "Saved active preset as '%s'\n", result.SavePresetName)
		return nil
	}

	// Handle preset deletion
	if result.DeletePresetName != "" {
		_ = cfgMgr.DeletePreset(result.DeletePresetName)
		fmt.Fprintf(w, "Deleted preset '%s'\n", result.DeletePresetName)
		return nil
	}

	// Handle explicit IP targeting/connection command
	if result.TargetIP != "" && result.PilotParams == nil {
		_ = cfgMgr.SetLastActiveIP(result.TargetIP)
		_ = cfgMgr.AddRecentIP(result.TargetIP)
		fmt.Fprintf(w, "Connected and saved target IP: %s\n", result.TargetIP)
		return nil
	}

	// Handle batch pilot command to room
	if result.TargetRoom != "" && result.PilotParams != nil {
		targets := reg.GetDevicesByRoom(result.TargetRoom)
		if len(targets) == 0 {
			return fmt.Errorf("no devices found in room '%s'", result.TargetRoom)
		}
		ips := make([]string, len(targets))
		for i, dev := range targets {
			ips[i] = dev.IP
		}
		errs := client.SendBatchCommand(ctx, ips, *result.PilotParams)
		failedCount := 0
		for ip, err := range errs {
			if err != nil {
				failedCount++
				fmt.Fprintf(w, "  - %s: %v\n", ip, err)
			}
		}
		if failedCount > 0 {
			return fmt.Errorf("batch room command failed for %d device(s)", failedCount)
		}
		fmt.Fprintf(w, "Successfully sent command to room '%s' (%d device(s))\n", result.TargetRoom, len(ips))
		return nil
	}

	// Handle single bulb pilot command
	if result.PilotParams != nil {
		targetIPToUse := targetIP
		if result.TargetIP != "" {
			targetIPToUse = result.TargetIP
		}
		if targetIPToUse == "" || targetIPToUse == wiz.FallbackIP {
			return fmt.Errorf("no target IP address specified (use --ip <address> or set active bulb)")
		}

		err := client.SendCommand(ctx, targetIPToUse, *result.PilotParams)
		if err != nil {
			return fmt.Errorf("failed to send command to %s: %w", targetIPToUse, err)
		}

		_ = cfgMgr.SetLastActiveIP(targetIPToUse)
		_ = cfgMgr.AddRecentIP(targetIPToUse)

		if result.StatusMsg != "" {
			fmt.Fprintln(w, result.StatusMsg)
		} else {
			fmt.Fprintf(w, "Command executed successfully on %s\n", targetIPToUse)
		}
		return nil
	}

	if result.StatusMsg != "" {
		fmt.Fprintln(w, result.StatusMsg)
	} else {
		fmt.Fprintln(w, "OK")
	}

	return nil
}
