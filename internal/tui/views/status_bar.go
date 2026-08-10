package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderStatusBar(mode string, activePanelStr string, activeDev *wiz.Device, selectedCount int, commandBuf string, statusMsg string, width int) string {
	focusPill := lipgloss.NewStyle().Background(styles.Surface0).Foreground(styles.Yellow).Bold(true).Padding(0, 1).Render(fmt.Sprintf("Focus: %s", activePanelStr))

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
	} else if mode == "VISUAL" {
		middleText = styles.StatusWarning.Render("[Space] Toggle Check • [a] Select All • [o/x] Power • [1-9] Scenes")
	} else if statusMsg != "" {
		middleText = styles.StatusInfo.Render(statusMsg)
	} else {
		middleText = styles.DimText.Render("[Tab] Switch Panel • [Space] Power • [1-9] Scenes • [:] Command • [?] Help")
	}

	left := focusPill + " " + targetPill
	right := lipgloss.NewStyle().Foreground(styles.Overlay0).Render("gowiz v1.3")

	availWidth := width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if availWidth < 5 {
		return focusPill + " " + targetPill
	}

	midTruncated := lipgloss.NewStyle().MaxWidth(availWidth).Render(middleText)
	padding := strings.Repeat(" ", max(0, availWidth-lipgloss.Width(midTruncated)))
	bar := left + " " + midTruncated + padding + " " + right

	return lipgloss.NewStyle().
		Background(styles.Mantle).
		Width(width).
		Render(bar)
}
