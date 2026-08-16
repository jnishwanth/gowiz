package daemon

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"wiz-tui/internal/circadian"
	"wiz-tui/internal/config"
	"wiz-tui/internal/wiz"
)

func TestDaemonTickNoDevices(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgMgr := config.NewManager(cfgPath)
	client := wiz.NewMockClient()

	var buf bytes.Buffer
	d := NewDaemon(cfgMgr, client, DaemonOptions{
		Interval: 10 * time.Millisecond,
		Once:     true,
		Verbose:  true,
		Writer:   &buf,
	})

	count, err := d.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on empty devices: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 updated devices, got %d", count)
	}
	if !bytes.Contains(buf.Bytes(), []byte("No target device IPs registered")) {
		t.Errorf("expected verbose message about no devices, got: %s", buf.String())
	}
}

func TestDaemonTickWithDevices(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgMgr := config.NewManager(cfgPath)
	_ = cfgMgr.SetAlias("192.168.1.101", "Living Room Light")
	_ = cfgMgr.SetRoom("192.168.1.101", "Living Room")
	_ = cfgMgr.SetAlias("192.168.1.102", "Bedroom Lamp")
	_ = cfgMgr.SetRoom("192.168.1.102", "Bedroom")

	// Set custom room circadian phase
	bedroomPhases := []circadian.SchedulePhase{
		{Name: "Cozy Night", StartHour: 0, EndHour: 24, StartTemp: 2200, EndTemp: 2200, StartDimming: 20, EndDimming: 20},
	}
	_ = cfgMgr.SetRoomCircadianPhases("Bedroom", bedroomPhases)

	client := wiz.NewMockClient()
	var buf bytes.Buffer
	d := NewDaemon(cfgMgr, client, DaemonOptions{
		Interval: 100 * time.Millisecond,
		Verbose:  true,
		Writer:   &buf,
	})

	count, err := d.Tick(context.Background())
	if err != nil {
		t.Fatalf("unexpected tick error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 updated devices, got %d", count)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Synced")) {
		t.Errorf("expected output to contain Synced log, got: %s", buf.String())
	}
}

func TestDaemonRunOnce(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgMgr := config.NewManager(cfgPath)
	_ = cfgMgr.AddRecentIP("192.168.1.50")

	client := wiz.NewMockClient()
	var buf bytes.Buffer
	d := NewDaemon(cfgMgr, client, DaemonOptions{
		Interval: 50 * time.Millisecond,
		Once:     true,
		Verbose:  true,
		Writer:   &buf,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := d.Run(ctx)
	if err != nil {
		t.Fatalf("expected nil error on Run once, got %v", err)
	}
}

func TestDaemonRunCancellation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgMgr := config.NewManager(cfgPath)
	_ = cfgMgr.AddRecentIP("192.168.1.50")

	client := wiz.NewMockClient()
	var buf bytes.Buffer
	d := NewDaemon(cfgMgr, client, DaemonOptions{
		Interval: 10 * time.Millisecond,
		Once:     false,
		Verbose:  true,
		Writer:   &buf,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := d.Run(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Stopping circadian daemon")) {
		t.Errorf("expected stopping daemon log message, got %s", buf.String())
	}
}
