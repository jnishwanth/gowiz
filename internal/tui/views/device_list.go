package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderDeviceList(reg *wiz.DeviceRegistry, deviceCursor int, isFocused bool, width, height int) string {
	devices := reg.List()
	activeDev, _ := reg.GetActive()

	innerWidth := max(width-6, 14)
	header := RenderSectionHeader("⚡ Bulbs", isFocused, innerWidth)

	var sb strings.Builder
	sb.WriteString(header + "\n")

	if len(devices) == 0 {
		sb.WriteString(styles.DimText.Render("No WiZ lights.\nPress [R] to scan."))
	} else {
		maxDeviceRows := max(height-6, 1)

		offset := 0
		if deviceCursor >= maxDeviceRows {
			offset = deviceCursor - maxDeviceRows + 1
		}
		if offset > len(devices)-maxDeviceRows && len(devices) > maxDeviceRows {
			offset = len(devices) - maxDeviceRows
		}
		if offset < 0 {
			offset = 0
		}

		endIdx := min(offset+maxDeviceRows, len(devices))
		visibleCount := endIdx - offset

		scrollbarLines := RenderScrollbar(visibleCount, len(devices), maxDeviceRows, offset)

		for i := offset; i < endIdx; i++ {
			dev := devices[i]
			isCursor := isFocused && i == deviceCursor
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

			displayLabel := dev.IP
			defaultName := fmt.Sprintf("WiZ Light (%s)", dev.IP)
			defaultFallbackName := fmt.Sprintf("WiZ Light (%s) [Fallback]", dev.IP)
			if dev.Name != "" && dev.Name != defaultName && dev.Name != defaultFallbackName {
				if innerWidth > 28 {
					displayLabel = fmt.Sprintf("%s (%s)", dev.Name, dev.IP)
				} else {
					displayLabel = dev.Name
				}
			} else if dev.IsFallback && innerWidth > 22 {
				displayLabel += " (FB)"
			}

			labelTruncated := lipgloss.NewStyle().MaxWidth(max(innerWidth-12, 8)).Render(displayLabel)
			line := fmt.Sprintf("%s %s %d.%s", checkbox, statusDot, i+1, labelTruncated)

			if isCursor {
				line = styles.SelectedItemStyle.Render("▸ " + line + " ↵")
			} else if isTarget {
				line = styles.SelectedItemStyle.Render("  " + line)
			} else {
				line = "  " + styles.UnselectedItemStyle.Render(line)
			}

			scrollChar := " "
			lineIdx := i - offset
			if scrollbarLines != nil && lineIdx < len(scrollbarLines) {
				scrollChar = scrollbarLines[lineIdx]
			}

			leftWidth := lipgloss.Width(line)
			padSpaces := max(0, innerWidth-leftWidth-1)
			fullRow := line + strings.Repeat(" ", padSpaces) + scrollChar

			sb.WriteString(fullRow + "\n")
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(max(width-2, 15)).
		Height(max(height-2, 4)).
		Render(sb.String())
}
