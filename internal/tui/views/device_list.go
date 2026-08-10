package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderDeviceList(reg *wiz.DeviceRegistry, isFocused bool, width, height int) string {
	devices := reg.List()
	activeDev, _ := reg.GetActive()

	var sb strings.Builder
	title := "⚡ Bulbs"
	if isFocused {
		title += " [ACTIVE]"
		sb.WriteString(styles.FocusedSectionTitleStyle.Render(title) + "\n\n")
	} else {
		sb.WriteString(styles.SectionTitleStyle.Render(title) + "\n\n")
	}

	if len(devices) == 0 {
		sb.WriteString(styles.DimText.Render("No WiZ lights.\nPress [R] to scan."))
	} else {
		maxLines := max(height-4, 5)

		for i, dev := range devices {
			if i >= maxLines {
				sb.WriteString(styles.DimText.Render(fmt.Sprintf("... +%d more", len(devices)-i)) + "\n")
				break
			}

			isTarget := activeDev != nil && activeDev.IP == dev.IP

			statusDot := styles.StatusSuccess.Render("●")
			if !dev.State {
				statusDot = styles.DimText.Render("○")
			}
			if !dev.Online {
				statusDot = styles.StatusError.Render("✖")
			}

			checkbox := "[ ]"
			if dev.Selected {
				checkbox = styles.StatusWarning.Render("[✓]")
			}

			ipText := dev.IP
			if dev.IsFallback && width > 28 {
				ipText += " (FB)"
			}

			ipTruncated := lipgloss.NewStyle().MaxWidth(max(width-14, 8)).Render(ipText)
			line := fmt.Sprintf("%s %s %d.%s", checkbox, statusDot, i+1, ipTruncated)

			if isTarget {
				sb.WriteString(styles.SelectedItemStyle.Render("▸ "+line) + "\n")
			} else {
				sb.WriteString("  " + styles.UnselectedItemStyle.Render(line) + "\n")
			}
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(max(width-2, 15)).
		Height(max(height-2, 6)).
		Render(sb.String())
}
