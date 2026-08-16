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

func TestConfigManagerRecentIPs(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "recent_ips_test.json")

	mgr := config.NewManager(cfgPath)

	// Add initial IPs
	_ = mgr.AddRecentIP("192.168.1.10")
	_ = mgr.AddRecentIP("192.168.1.20")
	_ = mgr.AddRecentIP("192.168.1.10") // Duplicate should move to front

	recents := mgr.GetRecentIPs()
	if len(recents) != 2 {
		t.Fatalf("Expected 2 recent IPs, got %d", len(recents))
	}
	if recents[0] != "192.168.1.10" || recents[1] != "192.168.1.20" {
		t.Errorf("Unexpected recent IPs order: %v", recents)
	}

	// Add more than 10 unique IPs to verify limit capping
	for i := 1; i <= 15; i++ {
		_ = mgr.AddRecentIP(filepath.Join("10.0.0.", string(rune('0'+i))))
	}

	recentsCapped := mgr.GetRecentIPs()
	if len(recentsCapped) != 10 {
		t.Errorf("Expected capped 10 recent IPs, got %d", len(recentsCapped))
	}

	// Verify persistence
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if len(mgr2.GetRecentIPs()) != 10 {
		t.Errorf("Expected reloaded 10 recent IPs, got %d", len(mgr2.GetRecentIPs()))
	}
}

func TestConfigManagerDeviceRooms(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "rooms_test.json")

	mgr := config.NewManager(cfgPath)

	err := mgr.SetRoom("192.168.1.115", "Living Room")
	if err != nil {
		t.Fatalf("Failed to set room: %v", err)
	}

	room1, found1 := mgr.GetRoom("192.168.1.115")
	if !found1 || room1 != "Living Room" {
		t.Errorf("Expected 'Living Room', got '%s' (found: %v)", room1, found1)
	}

	// Verify persistence
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	room2, found2 := mgr2.GetRoom("192.168.1.115")
	if !found2 || room2 != "Living Room" {
		t.Errorf("Expected reloaded room 'Living Room', got '%s'", room2)
	}

// Remove room assignment
	if err := mgr2.SetRoom("192.168.1.115", ""); err != nil {
		t.Fatalf("Failed to clear room: %v", err)
	}
	_, found3 := mgr2.GetRoom("192.168.1.115")
	if found3 {
		t.Errorf("Expected room assignment to be deleted when passed empty string")
	}
}

func TestConfigManagerPresets(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "presets_test.json")

	mgr := config.NewManager(cfgPath)

	dim := 50
	temp := 3000
	preset := config.Preset{
		Dimming: &dim,
		Temp:    &temp,
	}

	err := mgr.SetPreset("CozyWarm", preset)
	if err != nil {
		t.Fatalf("Failed to save preset: %v", err)
	}

	p, found := mgr.GetPreset("cozywarm")
	if !found {
		t.Fatalf("Expected preset 'cozywarm' to be found")
	}
	if p.Dimming == nil || *p.Dimming != 50 || p.Temp == nil || *p.Temp != 3000 {
		t.Errorf("Unexpected preset values: %+v", p)
	}

	all := mgr.GetPresets()
	if len(all) != 1 || all["cozywarm"].Dimming == nil {
		t.Errorf("Expected 1 stored preset in map, got %d", len(all))
	}

	// Verify persistence across new manager instance
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	p2, found2 := mgr2.GetPreset("CozyWarm")
	if !found2 || p2.Dimming == nil || *p2.Dimming != 50 {
		t.Errorf("Expected reloaded preset 'cozywarm' with dimming 50, got: %+v", p2)
	}

	// Delete preset
	if err := mgr2.DeletePreset("cozywarm"); err != nil {
		t.Fatalf("Failed to delete preset: %v", err)
	}

	_, foundDeleted := mgr2.GetPreset("cozywarm")
	if foundDeleted {
		t.Errorf("Expected deleted preset to no longer exist")
	}
}

