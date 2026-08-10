package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
)

func RenderHelpOverlay(width, height int) string {
	var sb strings.Builder
	sb.WriteString(styles.AppTitleStyle.Render("⌨  gowiz Vim Keyboard Workflow Reference") + "\n\n")

	sections := []struct {
		Title string
		Keys  [][2]string
	}{
		{
			Title: "Navigation & Panel Focus",
			Keys: [][2]string{
				{"Tab / Shift+Tab", "Cycle active panel (Devices / Controls / Scenes)"},
				{"j / k  or  ↓ / ↑", "Navigate items in active panel"},
				{"h / l  or  ← / →", "Adjust brightness (-10% / +10%)"},
				{"gg / G", "Jump to top / bottom of list"},
			},
		},
		{
			Title: "Light Control Shortcuts (NORMAL Mode)",
			Keys: [][2]string{
				{"o / Space", "Turn ON light"},
				{"x / f", "Turn OFF light"},
				{"r / g / b", "Quick RGB Red / Green / Blue"},
				{"w / c", "Warm White (2700K) / Cozy (2200K)"},
				{"s", "Sleep Mode preset"},
				{"p", "Purple preset"},
				{"u", "Undo last state change"},
			},
		},
		{
			Title: "Visual Mode & Multi-Bulb Control",
			Keys: [][2]string{
				{"v", "Toggle Visual Multi-Select Mode"},
				{"Space", "Toggle selection checkbox on focused bulb"},
				{"a", "Select All bulbs"},
				{"Esc", "Clear selection / Return to NORMAL mode"},
			},
		},
		{
			Title: "Command Mode (:) & Search (/)",
			Keys: [][2]string{
				{":scene <name|id>", "Activate WiZ scene by name or ID (e.g. :scene sunset)"},
				{":temp <2200-6500>", "Set Color Temperature in Kelvin"},
				{":rgb <r> <g> <b>", "Set RGB color values (0-255)"},
				{":dim <10-100>", "Set exact dimming brightness percentage"},
				{":scan", "Rescan local network for WiZ devices"},
				{":connect <ip>", "Directly target WiZ bulb IP address"},
				{"/ <query>", "Instant search / filter 32 dynamic WiZ scenes"},
				{":q / q", "Quit gowiz"},
			},
		},
	}

	for _, sec := range sections {
		sb.WriteString(styles.SectionTitleStyle.Render(sec.Title) + "\n")
		for _, k := range sec.Keys {
			keyPill := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render(fmtKey(k[0], 18))
			desc := styles.UnselectedItemStyle.Render(k[1])
			sb.WriteString(keyPill + "  " + desc + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(styles.DimText.Render("Press [Esc] or [?] to close help overlay"))

	return styles.ActivePanelStyle.
		Width(max(width-10, 50)).
		Height(max(height-6, 20)).
		Render(sb.String())
}

func fmtKey(key string, width int) string {
	if len(key) >= width {
		return key
	}
	return key + strings.Repeat(" ", width-len(key))
}
