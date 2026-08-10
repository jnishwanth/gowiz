package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderStatusBar(mode string, activeDev *wiz.Device, selectedCount int, commandBuf string, statusMsg string, width int) string {
	var modeBadge string
	switch mode {
	case "NORMAL":
		modeBadge = styles.NormalModeBadge.Render(" NORMAL ")
	case "VISUAL":
		modeBadge = styles.VisualModeBadge.Render(" VISUAL ")
	case "COMMAND":
		modeBadge = styles.CommandModeBadge.Render(" COMMAND ")
	case "SEARCH":
		modeBadge = styles.SearchModeBadge.Render(" SEARCH ")
	case "HELP":
		modeBadge = styles.HelpModeBadge.Render(" HELP ")
	default:
		modeBadge = styles.NormalModeBadge.Render(" " + mode + " ")
	}

	targetStr := "Target: None"
	if selectedCount > 0 {
		targetStr = fmt.Sprintf("Selected: %d", selectedCount)
	} else if activeDev != nil {
		targetStr = activeDev.IP
	}
	targetPill := lipgloss.NewStyle().Background(styles.Surface0).Foreground(styles.Text).Padding(0, 1).Render(targetStr)

	var middleText string
	if mode == "COMMAND" {
		middleText = styles.CommandPromptStyle.Render(":") + commandBuf
	} else if mode == "SEARCH" {
		middleText = styles.SearchPromptStyle.Render("/") + commandBuf
	} else if statusMsg != "" {
		middleText = styles.StatusInfo.Render(statusMsg)
	} else {
		middleText = styles.DimText.Render("[:] cmd • [/] search • [?] help • [Space] power • [1-9] scenes")
	}

	left := modeBadge + " " + targetPill
	right := lipgloss.NewStyle().Foreground(styles.Overlay0).Render("gowiz v1.1")

	availWidth := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if availWidth < 5 {
		return modeBadge + " " + targetPill
	}

	midTruncated := lipgloss.NewStyle().MaxWidth(availWidth).Render(middleText)
	padding := strings.Repeat(" ", max(0, availWidth-lipgloss.Width(midTruncated)))
	bar := left + " " + midTruncated + padding + " " + right

	return lipgloss.NewStyle().
		Background(styles.Mantle).
		Width(width).
		Render(bar)
}
