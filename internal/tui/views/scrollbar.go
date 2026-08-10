package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
)

// RenderScrollbar renders a vertical scrollbar string for list views
func RenderScrollbar(height, totalItems, visibleLines, offset int) []string {
	if height <= 0 || totalItems <= visibleLines {
		return nil
	}

	// Calculate thumb size (minimum 1 line)
	thumbSize := max(1, height*visibleLines/totalItems)

	// Calculate max offset
	maxOffset := totalItems - visibleLines
	if maxOffset <= 0 {
		return nil
	}

	// Calculate thumb position
	trackSpace := height - thumbSize
	thumbPos := 0
	if trackSpace > 0 && maxOffset > 0 {
		thumbPos = min(trackSpace, offset*trackSpace/maxOffset)
	}

	trackStyle := lipgloss.NewStyle().Foreground(styles.Surface0)
	thumbStyle := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true)

	lines := make([]string, height)
	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			lines[i] = thumbStyle.Render("█")
		} else {
			lines[i] = trackStyle.Render("│")
		}
	}
	return lines
}

// RenderSectionHeader renders a section title with a horizontal line filling remaining width
func RenderSectionHeader(title string, isFocused bool, width int) string {
	var baseStyle lipgloss.Style
	if isFocused {
		baseStyle = styles.FocusedSectionTitleStyle
	} else {
		baseStyle = styles.SectionTitleStyle
	}

	titleText := title
	if isFocused {
		titleText += " [ACTIVE]"
	}

	renderedTitle := baseStyle.Render(titleText)
	titleWidth := lipgloss.Width(renderedTitle)

	remaining := max(0, width-titleWidth-2)
	if remaining > 0 {
		lineStyle := lipgloss.NewStyle().Foreground(styles.Overlay0)
		if isFocused {
			lineStyle = lipgloss.NewStyle().Foreground(styles.Yellow)
		}
		line := lineStyle.Render(strings.Repeat("─", remaining))
		return renderedTitle + " " + line
	}
	return renderedTitle
}
