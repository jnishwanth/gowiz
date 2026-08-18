package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"wiz-tui/internal/circadian"
	"wiz-tui/internal/config"
)

func TestCLIRun(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	t.Run("Empty_command_error", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			Mock:       true,
			Command:    "",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err == nil {
			t.Errorf("expected error for empty command, got nil")
		}
	})

	t.Run("Single_device_pilot_command", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			TargetIP:   "192.168.1.100",
			Mock:       true,
			Command:    "warm",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := strings.ToLower(buf.String())
		if !strings.Contains(out, "warm white") && !strings.Contains(out, "executed successfully") {
			t.Errorf("unexpected output: %s", buf.String())
		}
	})

	t.Run("Device_naming_and_room_assignment", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			TargetIP:   "192.168.1.101",
			Mock:       true,
			Command:    "name Desk Light",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Desk Light") {
			t.Errorf("expected output to contain 'Desk Light', got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "room Office"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error setting room: %v", err)
		}
		if !strings.Contains(buf.String(), "Office") {
			t.Errorf("expected output to contain 'Office', got: %s", buf.String())
		}
	})

	t.Run("Batch_room_command", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			Mock:       true,
			Command:    "group Office off",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Successfully sent command to group/room 'Office'") {
			t.Errorf("unexpected output: %s", buf.String())
		}
	})

	t.Run("List_presets_and_categories", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			Mock:       true,
			Command:    "preset list",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "evening") {
			t.Errorf("expected preset list output, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "categories"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Nature") {
			t.Errorf("expected category list output, got: %s", buf.String())
		}
	})

	t.Run("Config_info_and_recent_IPs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			Mock:       true,
			Command:    "config",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Configuration File:") {
			t.Errorf("expected config info output, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "recent"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Recent Target IPs:") {
			t.Errorf("expected recent IPs output, got: %s", buf.String())
		}
	})

	t.Run("Export_and_import_config", func(t *testing.T) {
		exportFile := filepath.Join(tempDir, "exported.json")
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			Mock:       true,
			Command:    "export " + exportFile,
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error exporting config: %v", err)
		}
		if _, statErr := os.Stat(exportFile); os.IsNotExist(statErr) {
			t.Fatalf("expected export file to exist at %s", exportFile)
		}

		buf.Reset()
		opts.Command = "import " + exportFile
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error importing config: %v", err)
		}
		if !strings.Contains(buf.String(), "imported successfully") {
			t.Errorf("unexpected import output: %s", buf.String())
		}
	})

	t.Run("Telemetry_info_query", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			TargetIP:   "192.168.1.105",
			Mock:       true,
			Command:    "info",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Bulb Telemetry (192.168.1.105)") {
			t.Errorf("expected telemetry output, got: %s", buf.String())
		}
	})

	t.Run("Preset_save_and_apply", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			TargetIP:   "192.168.1.106",
			Mock:       true,
			Command:    "preset save my_relax",
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Saved active preset") {
			t.Errorf("expected save preset output, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "preset my_relax"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error applying preset: %v", err)
		}

		buf.Reset()
		opts.Command = "preset delete my_relax"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error deleting preset: %v", err)
		}
	})

	t.Run("JSON_output_formatting", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := Options{
			ConfigPath: configPath,
			TargetIP:   "192.168.1.107",
			Mock:       true,
			Command:    "info",
			JSONOutput: true,
			Writer:     buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		jsonStr := buf.String()
		if !strings.Contains(jsonStr, `"ip": "192.168.1.107"`) || !strings.Contains(jsonStr, `"signal_quality"`) {
			t.Errorf("expected JSON telemetry output, got: %s", jsonStr)
		}

		// Pilot command JSON
		buf.Reset()
		opts.Command = "warm"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"status": "ok"`) || !strings.Contains(buf.String(), `"target_ip": "192.168.1.107"`) {
			t.Errorf("expected JSON pilot command response, got: %s", buf.String())
		}

		// Preset list JSON
		buf.Reset()
		opts.Command = "preset list"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"builtin"`) || !strings.Contains(buf.String(), "evening") {
			t.Errorf("expected JSON preset list response, got: %s", buf.String())
		}

		// Categories list JSON
		buf.Reset()
		opts.Command = "categories"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Nature") {
			t.Errorf("expected JSON category list response, got: %s", buf.String())
		}

		// Config summary JSON
		buf.Reset()
		opts.Command = "config"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"config_file"`) || !strings.Contains(buf.String(), `"last_active_ip"`) {
			t.Errorf("expected JSON config summary response, got: %s", buf.String())
		}

		// Recent IPs JSON
		buf.Reset()
		opts.Command = "recent"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "192.168.1.107") {
			t.Errorf("expected JSON recent IPs response, got: %s", buf.String())
		}

		// Device naming and room assignment JSON
		buf.Reset()
		opts.Command = "name Living Room Lamp"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"name": "Living Room Lamp"`) {
			t.Errorf("expected JSON device rename output, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "room LivingRoom"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"room": "LivingRoom"`) {
			t.Errorf("expected JSON room output, got: %s", buf.String())
		}

		// Batch group command JSON
		buf.Reset()
		opts.Command = "group LivingRoom warm"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"device_count": 1`) {
			t.Errorf("expected JSON batch output, got: %s", buf.String())
		}

		// Connect IP JSON
		buf.Reset()
		opts.Command = "connect 192.168.1.200"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"ip": "192.168.1.200"`) {
			t.Errorf("expected JSON connect IP output, got: %s", buf.String())
		}

		// Export JSON
		exportFile := filepath.Join(tempDir, "json_export.json")
		buf.Reset()
		opts.Command = "export " + exportFile
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"action": "export"`) {
			t.Errorf("expected JSON export output, got: %s", buf.String())
		}

		// Import JSON
		buf.Reset()
		opts.Command = "import " + exportFile
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"action": "import"`) {
			t.Errorf("expected JSON import output, got: %s", buf.String())
		}

		// Completion JSON
		buf.Reset()
		opts.Command = "completion zsh"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), `"shell": "zsh"`) || !strings.Contains(buf.String(), "#compdef gowiz") {
			t.Errorf("expected JSON completion output, got: %s", buf.String())
		}
	})

	t.Run("Circadian command CLI execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			TargetIP:   "192.168.1.100",
			Mock:       true,
			Command:    "circadian 14:00",
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Circadian rhythm set") {
			t.Errorf("expected circadian set output, got: %s", buf.String())
		}
	})

	t.Run("Circadian_with_custom_config_phases", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		mgr := config.NewManager(cfgPath)
		_ = mgr.SetCircadianPhases([]circadian.SchedulePhase{
			{Name: "Shift Morning", StartHour: 0, EndHour: 12, StartTemp: 2200, EndTemp: 6500, StartDimming: 20, EndDimming: 100},
			{Name: "Shift Evening", StartHour: 12, EndHour: 24, StartTemp: 6500, EndTemp: 2200, StartDimming: 100, EndDimming: 20},
		})

		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			TargetIP:   "192.168.1.100",
			Mock:       true,
			Command:    "circadian 14:00",
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Shift Evening") {
			t.Errorf("expected output to contain custom phase 'Shift Evening', got: %s", buf.String())
		}
	})

	t.Run("Room_circadian_with_custom_config_phases", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		mgr := config.NewManager(cfgPath)
		_ = mgr.SetRoom("192.168.1.101", "Bedroom")
		_ = mgr.SetRoomCircadianPhases("Bedroom", []circadian.SchedulePhase{
			{Name: "Bedroom Morning", StartHour: 0, EndHour: 12, StartTemp: 2200, EndTemp: 3000, StartDimming: 20, EndDimming: 50},
			{Name: "Bedroom Evening", StartHour: 12, EndHour: 24, StartTemp: 3000, EndTemp: 2200, StartDimming: 50, EndDimming: 20},
		})

		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			TargetIP:   "192.168.1.101",
			Mock:       true,
			Command:    "circadian room Bedroom 14:00",
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "Successfully sent command to group/room 'Bedroom'") {
			t.Errorf("expected output to contain room batch success message, got: %s", buf.String())
		}
	})

	t.Run("Shell completion script generation plain text", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Command:    "completion fish",
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "complete -c gowiz") {
			t.Errorf("expected fish completion script output, got: %s", buf.String())
		}
	})

	t.Run("Daemon mode execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Daemon:     true,
			DaemonOnce: true,
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected daemon run error: %v", err)
		}
		if !strings.Contains(buf.String(), "Starting circadian schedule daemon") {
			t.Errorf("expected daemon startup log message, got: %s", buf.String())
		}
	})

	t.Run("System service generation and status", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer

		// Test systemd text output
		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Command:    "systemd",
			Writer:     &buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected systemd error: %v", err)
		}
		if !strings.Contains(buf.String(), "[Unit]") || !strings.Contains(buf.String(), "--daemon --interval") {
			t.Errorf("expected systemd config, got: %s", buf.String())
		}

		// Test launchd text output
		buf.Reset()
		opts.Command = "launchd"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected launchd error: %v", err)
		}
		if !strings.Contains(buf.String(), "<string>com.gowiz.daemon</string>") {
			t.Errorf("expected launchd config, got: %s", buf.String())
		}

		// Test service status JSON output
		buf.Reset()
		opts.Command = "service status systemd"
		opts.JSONOutput = true
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected service status error: %v", err)
		}
		if !strings.Contains(buf.String(), `"status": "ok"`) {
			t.Errorf("expected JSON status ok, got: %s", buf.String())
		}
	})

	t.Run("Server_mode_execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Server:     true,
			ServerPort: 19876,
			Writer:     &buf,
		}
		err := Run(ctx, opts)
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("unexpected server mode error: %v", err)
		}
		if !strings.Contains(buf.String(), "Starting gowiz HTTP REST API server on 0.0.0.0:19876") {
			t.Errorf("expected server startup message, got: %s", buf.String())
		}
	})

	t.Run("Effect_CLI_execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer

		opts := Options{
			ConfigPath: cfgPath,
			TargetIP:   "192.168.1.150",
			Mock:       true,
			Command:    "flash red 1",
			Writer:     &buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected flash effect error: %v", err)
		}
		if !strings.Contains(buf.String(), "Flashing lights") {
			t.Errorf("expected flash output, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "pulse 1"
		opts.JSONOutput = true
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected pulse effect JSON error: %v", err)
		}
		if !strings.Contains(buf.String(), `"effect": "pulse"`) {
			t.Errorf("expected pulse JSON output, got: %s", buf.String())
		}
	})

	t.Run("Group_set_delete_and_list", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer

		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Command:    "group set desk 192.168.1.100 192.168.1.101",
			Writer:     &buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected group set error: %v", err)
		}
		if !strings.Contains(buf.String(), "Created group 'desk'") {
			t.Errorf("expected created group message, got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "group list"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected group list error: %v", err)
		}
		if !strings.Contains(buf.String(), "desk:") {
			t.Errorf("expected group list to contain 'desk:', got: %s", buf.String())
		}

		buf.Reset()
		opts.Command = "group delete desk"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected group delete error: %v", err)
		}
		if !strings.Contains(buf.String(), "Deleted group 'desk'") {
			t.Errorf("expected deleted group message, got: %s", buf.String())
		}
	})

	t.Run("Discovery_and_scan_command_execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		var buf bytes.Buffer

		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Command:    "discover",
			Writer:     &buf,
		}
		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected discovery error: %v", err)
		}
		if !strings.Contains(buf.String(), "Discovered") {
			t.Errorf("expected discovery output, got: %s", buf.String())
		}

		buf.Reset()
		opts.JSONOutput = true
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected discovery JSON error: %v", err)
		}
		if !strings.Contains(buf.String(), `"status": "ok"`) || !strings.Contains(buf.String(), `"discovered_count"`) {
			t.Errorf("expected discovery JSON output, got: %s", buf.String())
		}
	})

	t.Run("Broadcast_and_all_device_CLI_execution", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "config.json")
		mgr := config.NewManager(cfgPath)
		_ = mgr.AddRecentIP("192.168.1.100")
		_ = mgr.AddRecentIP("192.168.1.101")

		var buf bytes.Buffer
		opts := Options{
			ConfigPath: cfgPath,
			Mock:       true,
			Command:    "all off",
			Writer:     &buf,
		}

		err := Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error running 'all off': %v", err)
		}

		buf.Reset()
		opts.JSONOutput = true
		opts.Command = "broadcast warm"
		err = Run(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error running 'broadcast warm': %v", err)
		}
		if !strings.Contains(buf.String(), `"target": "all"`) && !strings.Contains(buf.String(), `"status": "ok"`) {
			t.Errorf("expected broadcast JSON output, got: %s", buf.String())
		}
	})
}
