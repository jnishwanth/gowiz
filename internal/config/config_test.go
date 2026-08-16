package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"wiz-tui/internal/circadian"
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

func TestConfigManagerExportAndImport(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source_config.json")
	exportPath := filepath.Join(tempDir, "exports", "backup.json")
	targetPath := filepath.Join(tempDir, "target_config.json")

	mgr := config.NewManager(srcPath)
	_ = mgr.SetAlias("192.168.1.100", "Office Desk")
	_ = mgr.SetRoom("192.168.1.100", "Office")
	_ = mgr.AddRecentIP("192.168.1.100")
	dim := 80
	_ = mgr.SetPreset("focus_mode", config.Preset{Dimming: &dim})

	// Export to file
	err := mgr.ExportToFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to export config to file: %v", err)
	}

	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatalf("Expected export file to exist at %s", exportPath)
	}

	// Import into new Manager instance
	mgrTarget := config.NewManager(targetPath)
	err = mgrTarget.ImportFromFile(exportPath)
	if err != nil {
		t.Fatalf("Failed to import config from file: %v", err)
	}

	alias, foundAlias := mgrTarget.GetAlias("192.168.1.100")
	if !foundAlias || alias != "Office Desk" {
		t.Errorf("Expected imported alias 'Office Desk', got '%s' (found: %v)", alias, foundAlias)
	}

	room, foundRoom := mgrTarget.GetRoom("192.168.1.100")
	if !foundRoom || room != "Office" {
		t.Errorf("Expected imported room 'Office', got '%s' (found: %v)", room, foundRoom)
	}

	preset, foundPreset := mgrTarget.GetPreset("focus_mode")
	if !foundPreset || preset.Dimming == nil || *preset.Dimming != 80 {
		t.Errorf("Expected imported preset 'focus_mode', got %+v", preset)
	}

	// Verify empty path validation
	if err := mgr.ExportToFile(""); err == nil {
		t.Errorf("Expected error exporting to empty path")
	}
	if err := mgr.ImportFromFile(""); err == nil {
		t.Errorf("Expected error importing from empty path")
	}
}

func TestConfigManagerCircadianPhases(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "circadian_config.json")

	mgr := config.NewManager(cfgPath)
	if phases := mgr.GetCircadianPhases(); phases != nil {
		t.Errorf("expected initial CircadianPhases to be nil, got %v", phases)
	}

	customPhases := []circadian.SchedulePhase{
		{Name: "Custom Morning", StartHour: 6, EndHour: 12, StartTemp: 2700, EndTemp: 6000, StartDimming: 40, EndDimming: 100},
		{Name: "Custom Night", StartHour: 12, EndHour: 24, StartTemp: 6000, EndTemp: 2200, StartDimming: 100, EndDimming: 20},
	}

	err := mgr.SetCircadianPhases(customPhases)
	if err != nil {
		t.Fatalf("unexpected error setting circadian phases: %v", err)
	}

	retrieved := mgr.GetCircadianPhases()
	if len(retrieved) != 2 {
		t.Fatalf("expected 2 retrieved phases, got %d", len(retrieved))
	}
	if retrieved[0].Name != "Custom Morning" || retrieved[1].Name != "Custom Night" {
		t.Errorf("unexpected phase names in retrieved phases: %+v", retrieved)
	}

	// Verify persistence across reload
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	reloaded := mgr2.GetCircadianPhases()
	if len(reloaded) != 2 {
		t.Fatalf("expected 2 reloaded phases, got %d", len(reloaded))
	}

	// Verify validation on invalid phase set
	invalidPhases := []circadian.SchedulePhase{
		{Name: "Invalid", StartHour: 10, EndHour: 5, StartTemp: 2700, EndTemp: 5000, StartDimming: 50, EndDimming: 100},
	}
	if err := mgr.SetCircadianPhases(invalidPhases); err == nil {
		t.Errorf("expected error when setting invalid circadian phases, got nil")
	}

	// Clear phases
	if err := mgr.SetCircadianPhases(nil); err != nil {
		t.Fatalf("unexpected error clearing phases: %v", err)
	}
	if cleared := mgr.GetCircadianPhases(); cleared != nil {
		t.Errorf("expected nil after clearing circadian phases, got %v", cleared)
	}
}

func TestConfigManagerRoomCircadianPhases(t *testing.T) {
	tempDir := t.TempDir()
	cfgPath := filepath.Join(tempDir, "room_circadian_config.json")

	mgr := config.NewManager(cfgPath)
	if phases := mgr.GetRoomCircadianPhases("Bedroom"); phases != nil {
		t.Errorf("expected initial Bedroom room circadian phases to be nil, got %v", phases)
	}

	bedroomPhases := []circadian.SchedulePhase{
		{Name: "Bedroom Sleepy Morning", StartHour: 0, EndHour: 9, StartTemp: 2200, EndTemp: 3000, StartDimming: 20, EndDimming: 50},
		{Name: "Bedroom Coziness", StartHour: 9, EndHour: 24, StartTemp: 3000, EndTemp: 2200, StartDimming: 50, EndDimming: 20},
	}

	err := mgr.SetRoomCircadianPhases("Bedroom", bedroomPhases)
	if err != nil {
		t.Fatalf("unexpected error setting room circadian phases: %v", err)
	}

	// Verify case-insensitive room retrieval
	retrieved := mgr.GetRoomCircadianPhases("bedroom")
	if len(retrieved) != 2 {
		t.Fatalf("expected 2 retrieved phases for bedroom, got %d", len(retrieved))
	}
	if retrieved[0].Name != "Bedroom Sleepy Morning" {
		t.Errorf("unexpected phase name: %s", retrieved[0].Name)
	}

	// Verify fallback to global phases when room phases not set
	globalPhases := []circadian.SchedulePhase{
		{Name: "Global Day", StartHour: 0, EndHour: 24, StartTemp: 5000, EndTemp: 5000, StartDimming: 100, EndDimming: 100},
	}
	_ = mgr.SetCircadianPhases(globalPhases)

	// Bedroom has specific phases
	bRes := mgr.GetCircadianPhasesForRoom("bedroom")
	if len(bRes) != 2 || bRes[0].Name != "Bedroom Sleepy Morning" {
		t.Errorf("expected bedroom-specific phases, got %+v", bRes)
	}

	// Office has no specific phases, should fall back to global
	oRes := mgr.GetCircadianPhasesForRoom("Office")
	if len(oRes) != 1 || oRes[0].Name != "Global Day" {
		t.Errorf("expected fallback to global phases for Office, got %+v", oRes)
	}

	// Verify persistence across reload
	mgr2 := config.NewManager(cfgPath)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	reloaded := mgr2.GetRoomCircadianPhases("Bedroom")
	if len(reloaded) != 2 {
		t.Fatalf("expected 2 reloaded room phases, got %d", len(reloaded))
	}

	// Test error on empty room name
	if err := mgr.SetRoomCircadianPhases("", bedroomPhases); err == nil {
		t.Errorf("expected error when setting room circadian phase with empty room name")
	}
}
