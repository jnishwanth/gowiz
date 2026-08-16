package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
)

func RenderHelpOverlay(width, height int) string {
	var sb strings.Builder
	sb.WriteString(styles.AppTitleStyle.Render("⌨  gowiz Keyboard Reference") + "\n\n")

	sections := []struct {
		Title string
		Keys  [][2]string
	}{
		{
			Title: "Navigation & Focus",
			Keys: [][2]string{
				{"Tab / Shift+Tab", "Cycle active panel (Bulbs ➔ Controls ➔ Scenes)"},
				{"j / k  or  ↓ / ↑", "Move selection up / down (Adjust brightness when in Controls)"},
				{"h / l  or  ← / →", "Adjust brightness (-10% / +10%) / Dynamic speed"},
				{"Home / End", "Jump to top / bottom of list"},
			},
		},
		{
			Title: "Light Controls & Shortcuts",
			Keys: [][2]string{
				{"Space", "Instant Power Toggle (ON / OFF)"},
				{"Enter", "Apply highlighted scene or target light"},
				{"1 - 9", "Instant trigger favorite WiZ dynamic scenes"},
				{"t", "Start 15-minute countdown Sleep Timer"},
				{"[ / ]", "Decrease / Increase dynamic scene animation speed"},
				{"u", "Undo last state change"},
				{"R", "Rescan local network for WiZ devices"},
			},
		},
		{
			Title: "Visual Mode & Multi-Bulb Control",
			Keys: [][2]string{
				{"v", "Toggle Visual Multi-Select Mode"},
				{"Space", "Toggle selection checkbox on highlighted bulb"},
				{"a", "Select All bulbs"},
				{"Esc", "Clear selections / Return to Normal Mode"},
			},
		},
		{
			Title: "Command Mode (:) & Search (/)",
			Keys: [][2]string{
				{":name <alias>", "Rename active light & persist to config file"},
				{":room <room>", "Assign room to active light (or :room clear)"},
				{":group <name> <cmd>", "Batch control group/room (e.g. :group Desk on)"},
				{":group set <n> <ip...>", "Create custom multi-device group (e.g. :group set desk 192.168.1.50...)"},
				{":group delete <n>", "Delete custom device group"},
				{":group list", "List all configured custom device groups"},
				{":preset <name>", "Apply preset (evening, movie, night, focus, relax...)"},
				{":preset save <name>", "Save active light state as custom preset"},
				{":presets", "List all built-in and saved custom presets"},
				{":ip <address>", "Connect / add target bulb IP (e.g. :ip 192.168.1.50)"},
				{":recent", "List recently targeted bulb IP addresses"},
				{":config", "Display configuration status & saved alias count"},
				{":info / :diag", "Display detailed bulb hardware & signal diagnostics"},
				{":export [file]", "Export configuration backup to JSON file"},
				{":import <file>", "Import and merge configuration from JSON file"},
				{":scene <name|id>", "Activate dynamic scene (e.g. :scene sunset)"},
				{":cat <category>", "Filter scenes by category (Nature, Cozy, White...)"},
				{":categories", "List all available scene categories"},
				{":ocean / :sunset", "Quick scene shortcuts (:party, :cozy, :fireplace...)"},
				{":speed <20-200>", "Set dynamic scene speed percentage"},
				{":temp <2200-6500>", "Set Color Temperature in Kelvin"},
				{":warm / :cool", "Quick color temperature presets (2700K / 4200K)"},
				{":rgb <r> <g> <b>", "Set RGB color values (0-255)"},
				{":hex <#code>", "Set color by HEX code (e.g. :hex #ff5500)"},
				{":dim <10-100>", "Set exact dimming percentage"},
				{":fade <lvl|off> [s]", "Smooth brightness fade over seconds (e.g. :fade 20 30)"},
				{":flash <color> [x]", "Flash lights for notification alert (e.g. :flash red 3)"},
				{":pulse [cycles]", "Pulse light brightness oscillating min/max"},
				{":strobe <color>", "High-frequency strobe alert effect"},
				{":rainbow [steps]", "Sweep RGB spectrum hue sweep"},
				{":effect <type>", "Trigger dynamic light effect (flash, pulse, strobe, rainbow)"},
				{":sunrise / :sunset", "Simulate sunrise / sunset light transitions"},
				{":circadian [room] [t]", "24-hr circadian rhythm (:circadian 14:30 / :circadian room Bedroom)"},
				{":daemon / :schedule", "Run background circadian schedule sync daemon"},
				{":service <action>", "Manage systemd / launchd daemon service (:service install/uninstall)"},
				{":serve [port]", "Start gowiz HTTP REST API server (default port: 8080)"},
				{":toggle", "Toggle power state on active/selected lights"},
				{":timer <mins>", "Set custom sleep countdown timer"},
				{":u / :undo", "Undo previous state change"},
				{":scan / :discover", "Rescan subnet for WiZ devices"},
				{":completion <sh>", "Generate shell autocompletion (bash, zsh, fish)"},
				{"/ <query>", "Instant search / filter dynamic scenes"},
				{":q / q", "Quit gowiz"},
			},
		},
	}

	for _, sec := range sections {
		sb.WriteString(styles.SectionTitleStyle.Render(sec.Title) + "\n")
		for _, k := range sec.Keys {
			keyPill := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render(fmtKey(k[0], 20))
			desc := styles.UnselectedItemStyle.Render(k[1])
			sb.WriteString(keyPill + "  " + desc + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(styles.DimText.Render("Press [Esc] or [?] to close help overlay"))

	return styles.ActivePanelStyle.
		Width(max(width-8, 45)).
		Height(max(height-4, 18)).
		Render(sb.String())
}

func fmtKey(key string, width int) string {
	if len(key) >= width {
		return key
	}
	return key + strings.Repeat(" ", width-len(key))
}
