package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderStatusBar(mode string, activeDev *wiz.Device, selectedCount int, commandBuf string, statusMsg string, width int) string {
	// Mode Badge
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

	// Selection / Target info
	targetStr := "Target: None"
	if selectedCount > 0 {
		targetStr = fmt.Sprintf("Selected: %d bulbs", selectedCount)
	} else if activeDev != nil {
		targetStr = fmt.Sprintf("Target: %s", activeDev.IP)
	}
	targetPill := lipgloss.NewStyle().Background(styles.Surface0).Foreground(styles.Text).Padding(0, 1).Render(targetStr)

	// Command input line vs Status message
	var middleText string
	if mode == "COMMAND" {
		middleText = styles.CommandPromptStyle.Render(":") + commandBuf
	} else if mode == "SEARCH" {
		middleText = styles.SearchPromptStyle.Render("/") + commandBuf
	} else if statusMsg != "" {
		middleText = styles.StatusInfo.Render(statusMsg)
	} else {
		middleText = styles.DimText.Render("Press [:] command • [/] search • [?] help • [q] quit")
	}

	left := modeBadge + " " + targetPill
	right := lipgloss.NewStyle().Foreground(styles.Overlay0).Render("gowiz v1.0")

	// Calculate spacing
	availWidth := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if availWidth < 10 {
		availWidth = 10
	}

	midTruncated := lipgloss.NewStyle().MaxWidth(availWidth).Render(middleText)

	padding := strings.Repeat(" ", max(0, availWidth-lipgloss.Width(midTruncated)))
	bar := left + " " + midTruncated + padding + " " + right

	return lipgloss.NewStyle().
		Background(styles.Mantle).
		Width(width).
		Render(bar)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
