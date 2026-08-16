package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type mockProgram struct{}

func (m *mockProgram) Run() (tea.Model, error) {
	return nil, nil
}

func TestRunApp_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "gowiz v"+Version) {
		t.Errorf("expected version output, got: %q", stdout.String())
	}
}

func TestRunApp_VersionShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"-v"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "gowiz v"+Version) {
		t.Errorf("expected version output, got: %q", stdout.String())
	}
}

func TestRunApp_Check(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "gowiz headless sanity check: OK") {
		t.Errorf("expected check output, got: %q", stdout.String())
	}
}

func TestRunApp_InvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--invalid-flag-1234"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2 on invalid flag, got %d", code)
	}
}

func TestRunApp_CLICommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--mock", "--ip", "192.168.1.100", "on"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Turning on lights...") {
		t.Errorf("expected success message in stdout, got: %q", stdout.String())
	}
}

func TestRunApp_CLICommandJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--mock", "--ip", "192.168.1.100", "--json", "off"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"status": "ok"`) {
		t.Errorf("expected JSON ok response in stdout, got: %q", stdout.String())
	}
}

func TestRunApp_CLICommandError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--cmd", "preset nonexistentscenename"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 on CLI error, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Errorf("expected error message in stderr, got: %q", stderr.String())
	}
}

func TestRunApp_CLICommandErrorJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--cmd", "preset nonexistentscenename", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1 on CLI error, got %d", code)
	}
	if !strings.Contains(stderr.String(), `"status":"error"`) {
		t.Errorf("expected JSON error message in stderr, got: %q", stderr.String())
	}
}

func TestRunApp_DaemonMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--daemon", "--once", "--mock"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for daemon mode, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunApp_DaemonPositional(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--mock", "--once", "daemon"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for daemon subcommand, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunApp_ServerMode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := runApp(ctx, []string{"--server", "--mock", "--port", "0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for server mode, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Starting gowiz HTTP REST API server") {
		t.Errorf("expected server start message, got: %q", stdout.String())
	}
}

func TestRunApp_TUI(t *testing.T) {
	origNewProgram := newProgram
	defer func() { newProgram = origNewProgram }()

	newProgram = func(model tea.Model, opts ...tea.ProgramOption) programRunner {
		return &mockProgram{}
	}

	var stdout, stderr bytes.Buffer
	code := runApp(context.Background(), []string{"--mock"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for TUI mode with mock runner, got %d (stderr: %s)", code, stderr.String())
	}
}
