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

	innerWidth := max(width-6, 18)
	header := RenderSectionHeader("⚡ Bulbs", isFocused, innerWidth)

	var sb strings.Builder
	sb.WriteString(header + "\n\n")

	if len(devices) == 0 {
		sb.WriteString(styles.DimText.Render("No WiZ lights.\nPress [R] to scan."))
	} else {
		maxLines := max(height-4, 3)

		offset := 0
		if deviceCursor >= maxLines {
			offset = deviceCursor - maxLines + 1
		}
		if offset > len(devices)-maxLines && len(devices) > maxLines {
			offset = len(devices) - maxLines
		}
		if offset < 0 {
			offset = 0
		}

		endIdx := min(offset+maxLines, len(devices))
		visibleCount := endIdx - offset

		scrollbarLines := RenderScrollbar(visibleCount, len(devices), maxLines, offset)

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

			ipText := dev.IP
			if dev.IsFallback && innerWidth > 22 {
				ipText += " (FB)"
			}

			ipTruncated := lipgloss.NewStyle().MaxWidth(max(innerWidth-12, 8)).Render(ipText)
			line := fmt.Sprintf("%s %s %d.%s", checkbox, statusDot, i+1, ipTruncated)

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
			padSpaces := max(0, innerWidth-leftWidth)
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
		Height(max(height-2, 5)).
		Render(sb.String())
}
