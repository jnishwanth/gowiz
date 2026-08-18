package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	SystemdServiceName = "gowiz-daemon.service"
	LaunchdPlistName   = "com.gowiz.daemon.plist"
)

// GenerateSystemd returns a systemd user unit configuration string for the gowiz circadian daemon.
func GenerateSystemd(execPath string, interval string) string {
	if execPath == "" {
		execPath = "gowiz"
	}
	if interval == "" {
		interval = "1m"
	}
	return fmt.Sprintf(`[Unit]
Description=gowiz Circadian Rhythm Daemon
After=network.target

[Service]
Type=simple
ExecStart=%s --daemon --interval %s
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`, execPath, interval)
}

// GenerateLaunchd returns a macOS LaunchAgent plist XML configuration string for the gowiz circadian daemon.
func GenerateLaunchd(execPath string, interval string) string {
	if execPath == "" {
		execPath = "gowiz"
	}
	if interval == "" {
		interval = "1m"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gowiz.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>--daemon</string>
        <string>--interval</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`, execPath, interval)
}

// DetectTargetOS returns systemd for linux, launchd for darwin/macOS, or default systemd.
func DetectTargetOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "launchd"
	default:
		return "systemd"
	}
}

// GetServicePath returns the absolute path where the service file should be stored for the given system type.
func GetServicePath(serviceType string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user home directory: %w", err)
	}

	st := strings.ToLower(strings.TrimSpace(serviceType))
	if st == "" || st == "auto" {
		st = DetectTargetOS()
	}

	switch st {
	case "systemd", "linux":
		return filepath.Join(home, ".config", "systemd", "user", SystemdServiceName), nil
	case "launchd", "darwin", "mac", "macos":
		return filepath.Join(home, "Library", "LaunchAgents", LaunchdPlistName), nil
	default:
		return "", fmt.Errorf("unsupported service type %q, expected 'systemd' or 'launchd'", serviceType)
	}
}

// Install writes the background daemon service definition file for systemd or launchd.
func Install(serviceType string, execPath string, interval string) (string, error) {
	st := strings.ToLower(strings.TrimSpace(serviceType))
	if st == "" || st == "auto" {
		st = DetectTargetOS()
	}

	path, err := GetServicePath(st)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("failed to create service directory: %w", err)
	}

	var content string
	if st == "launchd" || st == "darwin" || st == "mac" || st == "macos" {
		content = GenerateLaunchd(execPath, interval)
	} else {
		content = GenerateSystemd(execPath, interval)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write service configuration to %s: %w", path, err)
	}

	return fmt.Sprintf("Service installed successfully to %s", path), nil
}

// Uninstall removes the background daemon service configuration file.
func Uninstall(serviceType string) (string, error) {
	st := strings.ToLower(strings.TrimSpace(serviceType))
	if st == "" || st == "auto" {
		st = DetectTargetOS()
	}

	path, err := GetServicePath(st)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Sprintf("Service file %s is not installed.", path), nil
	}

	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("failed to remove service configuration at %s: %w", path, err)
	}

	return fmt.Sprintf("Service uninstalled successfully from %s", path), nil
}

// Status checks if the background daemon service file exists.
func Status(serviceType string) (string, bool, error) {
	st := strings.ToLower(strings.TrimSpace(serviceType))
	if st == "" || st == "auto" {
		st = DetectTargetOS()
	}

	path, err := GetServicePath(st)
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Sprintf("Installed (%s)", path), true, nil
	}
	return fmt.Sprintf("Not installed (%s)", path), false, nil
}
