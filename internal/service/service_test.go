package service

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateSystemd(t *testing.T) {
	unit := GenerateSystemd("/usr/local/bin/gowiz", "5m")
	if !strings.Contains(unit, "ExecStart=/usr/local/bin/gowiz --daemon --interval 5m") {
		t.Errorf("Expected ExecStart in systemd unit, got:\n%s", unit)
	}
	if !strings.Contains(unit, "[Unit]") || !strings.Contains(unit, "[Service]") {
		t.Errorf("Invalid systemd structure")
	}

	// Defaults check
	defUnit := GenerateSystemd("", "")
	if !strings.Contains(defUnit, "ExecStart=gowiz --daemon --interval 1m") {
		t.Errorf("Expected default ExecStart, got:\n%s", defUnit)
	}
}

func TestGenerateLaunchd(t *testing.T) {
	plist := GenerateLaunchd("/usr/local/bin/gowiz", "2m")
	if !strings.Contains(plist, "<string>/usr/local/bin/gowiz</string>") {
		t.Errorf("Expected path in launchd plist, got:\n%s", plist)
	}
	if !strings.Contains(plist, "<string>com.gowiz.daemon</string>") {
		t.Errorf("Expected label in launchd plist")
	}

	defPlist := GenerateLaunchd("", "")
	if !strings.Contains(defPlist, "<string>1m</string>") {
		t.Errorf("Expected default interval in launchd plist")
	}
}

func TestDetectTargetOS(t *testing.T) {
	osType := DetectTargetOS()
	if osType != "launchd" && osType != "systemd" {
		t.Errorf("Unexpected target OS detection result: %s", osType)
	}
}

func TestGetServicePath(t *testing.T) {
	sysPath, err := GetServicePath("systemd")
	if err != nil {
		t.Fatalf("GetServicePath(systemd) failed: %v", err)
	}
	if !strings.HasSuffix(sysPath, SystemdServiceName) {
		t.Errorf("Expected systemd service name suffix, got %s", sysPath)
	}

	macPath, err := GetServicePath("launchd")
	if err != nil {
		t.Fatalf("GetServicePath(launchd) failed: %v", err)
	}
	if !strings.HasSuffix(macPath, LaunchdPlistName) {
		t.Errorf("Expected launchd plist name suffix, got %s", macPath)
	}

	autoPath, err := GetServicePath("auto")
	if err != nil || autoPath == "" {
		t.Fatalf("GetServicePath(auto) failed: %v", err)
	}

	_, err = GetServicePath("invalid_type")
	if err == nil {
		t.Errorf("Expected error for invalid service type")
	}
}

func TestInstallUninstallStatus(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Status before install
	msg, installed, err := Status("systemd")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if installed {
		t.Errorf("Expected not installed, got %s", msg)
	}

	// Install systemd
	installMsg, err := Install("systemd", "/bin/gowiz", "1m")
	if err != nil {
		t.Fatalf("Install(systemd) failed: %v", err)
	}
	if !strings.Contains(installMsg, "installed successfully") {
		t.Errorf("Unexpected install message: %s", installMsg)
	}

	// Check status after install
	msg, installed, err = Status("systemd")
	if err != nil || !installed {
		t.Fatalf("Status after install failed: %v (msg: %s)", err, msg)
	}

	// Uninstall systemd
	unMsg, err := Uninstall("systemd")
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !strings.Contains(unMsg, "uninstalled successfully") {
		t.Errorf("Unexpected uninstall message: %s", unMsg)
	}

	// Status after uninstall
	_, installed, _ = Status("systemd")
	if installed {
		t.Errorf("Expected not installed after uninstall")
	}

	// Install launchd
	installMacMsg, err := Install("launchd", "/bin/gowiz", "5m")
	if err != nil {
		t.Fatalf("Install(launchd) failed: %v", err)
	}
	if !strings.Contains(installMacMsg, "installed successfully") {
		t.Errorf("Unexpected launchd install message: %s", installMacMsg)
	}

	macFile, _ := GetServicePath("launchd")
	if _, err := os.Stat(macFile); os.IsNotExist(err) {
		t.Errorf("Launchd plist file was not created at %s", macFile)
	}

	// Uninstall non-existent
	unNon, err := Uninstall("systemd")
	if err != nil || !strings.Contains(unNon, "is not installed") {
		t.Errorf("Expected non-installed message, got %s", unNon)
	}
}
