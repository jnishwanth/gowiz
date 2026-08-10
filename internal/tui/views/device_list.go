package views

import (
	"fmt"
	"strings"

	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderDeviceList(reg *wiz.DeviceRegistry, isFocused bool, width, height int) string {
	devices := reg.List()
	activeDev, _ := reg.GetActive()

	var sb strings.Builder
	sb.WriteString(styles.SectionTitleStyle.Render("⚡ Discovered Bulbs") + "\n\n")

	if len(devices) == 0 {
		sb.WriteString(styles.DimText.Render("No WiZ devices found.\nPress [R] to scan."))
	} else {
		for i, dev := range devices {
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
			if dev.IsFallback {
				ipText += " (Fallback)"
			}

			line := fmt.Sprintf("%s %s %d. %s", checkbox, statusDot, i+1, ipText)

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
		Width(width).
		Height(height).
		Render(sb.String())
}
