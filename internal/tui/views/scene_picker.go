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

	header := RenderSectionHeader(title, isFocused, width-6)

	var sb strings.Builder
	sb.WriteString(header + "\n\n")

	if len(scenes) == 0 {
		sb.WriteString(styles.DimText.Render("No scenes match query.\nPress [Esc] to clear filter."))
	} else {
		maxLines := max(height-4, 3)

		// Calculate scroll offset so active cursor is always visible
		offset := 0
		if sceneCursor >= maxLines {
			offset = sceneCursor - maxLines + 1
		}
		if offset > len(scenes)-maxLines && len(scenes) > maxLines {
			offset = len(scenes) - maxLines
		}
		if offset < 0 {
			offset = 0
		}

		endIdx := min(offset+maxLines, len(scenes))
		visibleCount := endIdx - offset

		scrollbarLines := RenderScrollbar(visibleCount, len(scenes), maxLines, offset)

		for i := offset; i < endIdx; i++ {
			scene := scenes[i]
			isCursor := isFocused && i == sceneCursor
			isActiveOnLight := scene.ID == activeSceneID

			swatchHex := scene.AccentColor
			if isFocused {
				swatchHex = scene.GetAccentColor(animFrame)
			}
			swatchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(swatchHex)).Bold(true)

			shortcut := "   "
			if scene.ID >= 1 && scene.ID <= 9 {
				shortcut = styles.StatusWarning.Render(fmt.Sprintf("[%d]", scene.ID))
			}

			nameStr := lipgloss.NewStyle().MaxWidth(max(width-32, 8)).Render(scene.Name)
			nameBadge := fmt.Sprintf("%s %-12s", shortcut, nameStr)

			swatch := swatchStyle.Render("████")

			activeBadge := ""
			if isActiveOnLight {
				activeBadge = " " + styles.StatusSuccess.Render("[ACTIVE ON LIGHT]")
			}

			line := fmt.Sprintf("%s %s%s", nameBadge, swatch, activeBadge)

			scrollChar := " "
			lineIdx := i - offset
			if scrollbarLines != nil && lineIdx < len(scrollbarLines) {
				scrollChar = scrollbarLines[lineIdx]
			}

			if isCursor {
				applyHint := styles.StatusSuccess.Render(" ↵ Enter")
				sb.WriteString(styles.SelectedItemStyle.Render("▸ "+line+applyHint) + " " + scrollChar + "\n")
			} else {
				sb.WriteString("  " + styles.UnselectedItemStyle.Render(line) + " " + scrollChar + "\n")
			}
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(max(width-2, 20)).
		Height(max(height-2, 5)).
		Render(sb.String())
}
