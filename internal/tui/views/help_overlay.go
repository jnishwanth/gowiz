package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
)

func RenderHelpOverlay(width, height int) string {
	var sb strings.Builder
	sb.WriteString(styles.AppTitleStyle.Render("⌨  gowiz Ergonomic Keyboard Reference") + "\n\n")

	sections := []struct {
		Title string
		Keys  [][2]string
	}{
		{
			Title: "Navigation & Panel Focus",
			Keys: [][2]string{
				{"Tab / Shift+Tab", "Cycle focus (Bulbs / Control Center / Scenes)"},
				{"j / k  or  ↓ / ↑", "Move selection up / down"},
				{"h / l  or  ← / →", "Adjust brightness (-10% / +10%)"},
				{"g / G  or  Home/End", "Jump to top / bottom of list"},
			},
		},
		{
			Title: "Direct Hotkeys & Controls",
			Keys: [][2]string{
				{"Space", "Instant Power Toggle (ON / OFF)"},
				{"Enter", "Activate highlighted scene or target light"},
				{"1 - 9", "Instant trigger favorite WiZ dynamic scenes"},
				{"t", "Start 15-minute countdown Sleep Timer"},
				{"[ / ]", "Decrease / Increase dynamic scene speed"},
				{"r / g / b", "Quick RGB Red / Green / Blue"},
				{"w / c", "Warm White (2700K) / Cozy Amber (2200K)"},
				{"u", "Undo last state change"},
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
			Title: "Command Mode (:) & Instant Search (/)",
			Keys: [][2]string{
				{":scene <name|id>", "Activate dynamic scene (e.g. :scene sunset)"},
				{":temp <2200-6500>", "Set Color Temperature in Kelvin"},
				{":rgb <r> <g> <b>", "Set RGB color values (0-255)"},
				{":dim <10-100>", "Set exact dimming percentage"},
				{":timer <mins>", "Set custom sleep countdown timer"},
				{":scan", "Rescan subnet for WiZ devices"},
				{"/ <query>", "Instant search / filter 32 dynamic WiZ scenes"},
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
