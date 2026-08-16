package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		if !strings.Contains(buf.String(), "Successfully sent command to room 'Office'") {
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
}
