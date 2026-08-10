package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderScenePicker(filterQuery string, activeSceneID int, sceneCursor int, isFocused bool, animFrame int, width, height int) string {
	scenes := wiz.FilterScenes(filterQuery)

	title := "🎨 Dynamic Scenes"
	if filterQuery != "" {
		title += fmt.Sprintf(" [Filter: %s]", filterQuery)
	}

	innerWidth := max(width-6, 14)
	header := RenderSectionHeader(title, isFocused, innerWidth)

	var sb strings.Builder
	sb.WriteString(header + "\n")

	if len(scenes) == 0 {
		sb.WriteString(styles.DimText.Render("No scenes match query.\nPress [Esc] to clear filter."))
	} else {
		// PanelStyle has 2px vertical padding (1 top, 1 bottom) and 2px borders (1 top, 1 bottom) = 4 lines reduction
		// Header takes 1 line + 1 line spacing = 2 lines
		// Maximum inner scene rows allowed = height - 6
		maxSceneRows := max(height-6, 1)

		// Calculate scroll offset so active cursor is always visible
		offset := 0
		if sceneCursor >= maxSceneRows {
			offset = sceneCursor - maxSceneRows + 1
		}
		if offset > len(scenes)-maxSceneRows && len(scenes) > maxSceneRows {
			offset = len(scenes) - maxSceneRows
		}
		if offset < 0 {
			offset = 0
		}

		endIdx := min(offset+maxSceneRows, len(scenes))
		visibleCount := endIdx - offset

		scrollbarLines := RenderScrollbar(visibleCount, len(scenes), maxSceneRows, offset)

		for i := offset; i < endIdx; i++ {
			scene := scenes[i]
			isCursor := isFocused && i == sceneCursor
			isActiveOnLight := scene.ID == activeSceneID

			swatchHex := scene.AccentColor
			if isFocused {
				swatchHex = scene.GetAccentColor(animFrame)
			}
			swatchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(swatchHex)).Bold(true)

			shortcutStr := "   "
			if scene.ID >= 1 && scene.ID <= 9 {
				shortcutStr = styles.StatusWarning.Render(fmt.Sprintf("[%d]", scene.ID))
			}

			nameWidth := min(max(innerWidth-18, 6), 14)
			nameFormatted := lipgloss.NewStyle().Width(nameWidth).MaxWidth(nameWidth).Render(scene.Name)

			swatch := swatchStyle.Render("████")

			activeBadge := ""
			if isActiveOnLight {
				activeBadge = " " + styles.StatusSuccess.Render("[ACTIVE]")
			}

			leftContent := fmt.Sprintf("%s %s %s%s", shortcutStr, nameFormatted, swatch, activeBadge)

			if isCursor {
				applyHint := styles.StatusSuccess.Render(" ↵ Enter")
				leftContent = styles.SelectedItemStyle.Render("▸ " + leftContent + applyHint)
			} else {
				leftContent = "  " + styles.UnselectedItemStyle.Render(leftContent)
			}

			scrollChar := " "
			lineIdx := i - offset
			if scrollbarLines != nil && lineIdx < len(scrollbarLines) {
				scrollChar = scrollbarLines[lineIdx]
			}

			leftWidth := lipgloss.Width(leftContent)
			padSpaces := max(0, innerWidth-leftWidth-1)
			fullRow := leftContent + strings.Repeat(" ", padSpaces) + scrollChar

			sb.WriteString(fullRow + "\n")
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(max(width-2, 20)).
		Height(max(height-2, 4)).
		Render(sb.String())
}
