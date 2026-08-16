package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"wiz-tui/internal/config"
)

func TestConfigManagerDefaults(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "test_config.json")

	mgr := config.NewManager(cfgPath)
	if mgr.FilePath() != cfgPath {
		t.Fatalf("Expected FilePath %s, got %s", cfgPath, mgr.FilePath())
	}

	err := mgr.Load()
	if err != nil {
		t.Fatalf("Expected no error on non-existent config load, got: %v", err)
	}

	cfg := mgr.GetConfig()
	if !cfg.AutoScan {
		t.Errorf("Expected AutoScan default to be true")
	}
	if len(cfg.DeviceAliases) != 0 {
		t.Errorf("Expected empty initial DeviceAliases, got %d", len(cfg.DeviceAliases))
	}
}

func TestConfigManagerPersistenceAndRoundtrip(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "subDir", "gowiz_config.json")

	mgr := config.NewManager(cfgPath)

	err := mgr.SetAlias("192.168.1.50", "Desk Lamp")
	if err != nil {
		t.Fatalf("Failed to set alias: %v", err)
	}

	err = mgr.SetAlias("aa:bb:cc:dd:ee:ff", "Living Room Main")
	if err != nil {
		t.Fatalf("Failed to set MAC alias: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("Expected config file to exist at %s", cfgPath)
	}

	// Reload in a new manager
	mgr2 := config.NewManager(cfgPath)
	err = mgr2.Load()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	alias1, found1 := mgr2.GetAlias("192.168.1.50")
	if !found1 || alias1 != "Desk Lamp" {
		t.Errorf("Expected 'Desk Lamp', got '%s' (found: %v)", alias1, found1)
	}

	alias2, found2 := mgr2.GetAlias("aa:bb:cc:dd:ee:ff")
	if !found2 || alias2 != "Living Room Main" {
		t.Errorf("Expected 'Living Room Main', got '%s' (found: %v)", alias2, found2)
	}

	// Remove an alias
	err = mgr2.SetAlias("192.168.1.50", "")
	if err != nil {
		t.Fatalf("Failed to remove alias: %v", err)
	}

	_, foundRemoved := mgr2.GetAlias("192.168.1.50")
	if foundRemoved {
		t.Errorf("Expected alias to be deleted when passed empty string")
	}
}

func TestConfigManagerInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(cfgPath, []byte("{invalid-json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	mgr := config.NewManager(cfgPath)
	err = mgr.Load()
	if err == nil {
		t.Errorf("Expected error loading invalid JSON config, got nil")
	}
}

func TestDefaultPathFallback(t *testing.T) {
	path := config.DefaultPath()
	if path == "" {
		t.Errorf("DefaultPath returned empty string")
	}
}

func TestConfigManagerLastActiveIP(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "last_active_test.json")

	mgr := config.NewManager(cfgPath)
	err := mgr.SetLastActiveIP("192.168.1.88")
	if err != nil {
		t.Fatalf("Failed to set last active IP: %v", err)
	}

	if mgr.GetLastActiveIP() != "192.168.1.88" {
		t.Errorf("Expected GetLastActiveIP '192.168.1.88', got '%s'", mgr.GetLastActiveIP())
	}

	// Reload in a new manager
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if mgr2.GetLastActiveIP() != "192.168.1.88" {
		t.Errorf("Expected reloaded LastActiveIP '192.168.1.88', got '%s'", mgr2.GetLastActiveIP())
	}
}
