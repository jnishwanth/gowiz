package daemon

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"wiz-tui/internal/circadian"
	"wiz-tui/internal/config"
	"wiz-tui/internal/wiz"
)

// DaemonOptions configures the circadian background daemon scheduler.
type DaemonOptions struct {
	Interval time.Duration
	Once     bool
	Verbose  bool
	Writer   io.Writer
}

// Daemon manages background synchronization of WiZ smart lights to circadian rhythm schedules.
type Daemon struct {
	cfgMgr  *config.Manager
	client  wiz.Client
	options DaemonOptions
}

// NewDaemon initializes a circadian scheduler daemon instance.
func NewDaemon(cfgMgr *config.Manager, client wiz.Client, options DaemonOptions) *Daemon {
	if options.Interval <= 0 {
		options.Interval = 1 * time.Minute
	}
	if options.Writer == nil {
		options.Writer = os.Stdout
	}
	return &Daemon{
		cfgMgr:  cfgMgr,
		client:  client,
		options: options,
	}
}

// Tick evaluates current circadian lighting settings and dispatches updates across registered WiZ lights.
func (d *Daemon) Tick(ctx context.Context) (int, error) {
	if err := d.cfgMgr.Load(); err != nil {
		// Log or ignore load errors if non-existent file
	}
	cfg := d.cfgMgr.GetConfig()

	// Gather deduplicated valid device IP addresses
	ipSet := make(map[string]struct{})
	for ip := range cfg.DeviceAliases {
		if net.ParseIP(ip) != nil {
			ipSet[ip] = struct{}{}
		}
	}
	for ip := range cfg.DeviceRooms {
		if net.ParseIP(ip) != nil {
			ipSet[ip] = struct{}{}
		}
	}
	for _, recIP := range cfg.RecentIPs {
		if net.ParseIP(recIP) != nil && recIP != wiz.FallbackIP {
			ipSet[recIP] = struct{}{}
		}
	}
	if cfg.LastActiveIP != "" && net.ParseIP(cfg.LastActiveIP) != nil && cfg.LastActiveIP != wiz.FallbackIP {
		ipSet[cfg.LastActiveIP] = struct{}{}
	}

	if len(ipSet) == 0 {
		if d.options.Verbose {
			fmt.Fprintln(d.options.Writer, "[gowiz daemon] No target device IPs registered in configuration.")
		}
		return 0, nil
	}

	now := time.Now()
	// Group devices by PilotParams key (e.g. temp:dimming signature)
	type paramGroup struct {
		params wiz.PilotParams
		phase  string
		ips    []string
	}
	groups := make(map[string]*paramGroup)

	for ip := range ipSet {
		room := cfg.DeviceRooms[ip]
		var phases []circadian.SchedulePhase

		if room != "" && len(cfg.RoomCircadianPhases) > 0 {
			roomKey := strings.ToLower(strings.TrimSpace(room))
			if p, found := cfg.RoomCircadianPhases[roomKey]; found && len(p) > 0 {
				phases = p
			}
		}
		if len(phases) == 0 {
			phases = cfg.CircadianPhases
		}

		info := circadian.CalculateWithPhases(now, phases)
		params := info.Params

		key := fmt.Sprintf("%d-%d", info.Temp, info.Dimming)
		if grp, found := groups[key]; found {
			grp.ips = append(grp.ips, ip)
		} else {
			groups[key] = &paramGroup{
				params: params,
				phase:  info.Phase,
				ips:    []string{ip},
			}
		}
	}

	totalUpdated := 0
	var errMsgs []string

	for _, grp := range groups {
		sort.Strings(grp.ips)
		errs := d.client.SendBatchCommand(ctx, grp.ips, grp.params)
		for ip, err := range errs {
			if err != nil {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", ip, err))
			} else {
				totalUpdated++
			}
		}
		if d.options.Verbose {
			fmt.Fprintf(d.options.Writer, "[gowiz daemon] Synced %d light(s) to phase %s (%v)\n", len(grp.ips), grp.phase, grp.ips)
		}
	}

	if len(errMsgs) > 0 {
		return totalUpdated, fmt.Errorf("circadian sync errors (%d failed): %s", len(errMsgs), strings.Join(errMsgs, "; "))
	}

	return totalUpdated, nil
}

// Run executes the circadian daemon, ticking immediately and on every interval until context is cancelled.
func (d *Daemon) Run(ctx context.Context) error {
	if d.options.Verbose {
		fmt.Fprintf(d.options.Writer, "[gowiz daemon] Starting circadian schedule daemon (interval: %v)\n", d.options.Interval)
	}

	_, err := d.Tick(ctx)
	if err != nil && d.options.Verbose {
		fmt.Fprintf(d.options.Writer, "[gowiz daemon] Tick warning: %v\n", err)
	}

	if d.options.Once {
		return err
	}

	ticker := time.NewTicker(d.options.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if d.options.Verbose {
				fmt.Fprintln(d.options.Writer, "[gowiz daemon] Stopping circadian daemon.")
			}
			return ctx.Err()
		case <-ticker.C:
			_, tickErr := d.Tick(ctx)
			if tickErr != nil && d.options.Verbose {
				fmt.Fprintf(d.options.Writer, "[gowiz daemon] Tick warning: %v\n", tickErr)
			}
		}
	}
}
