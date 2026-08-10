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

	innerWidth := max(width-6, 26)
	header := RenderSectionHeader(title, isFocused, innerWidth)

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

			// Swatch pill pulsates only when Dynamic Scenes panel is focused; frozen when unfocused
			swatchHex := scene.AccentColor
			if isFocused {
				swatchHex = scene.GetAccentColor(animFrame)
			}
			swatchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(swatchHex)).Bold(true)

			// 1. Shortcut pill (3 visual chars)
			shortcutStr := "   "
			if scene.ID >= 1 && scene.ID <= 9 {
				shortcutStr = styles.StatusWarning.Render(fmt.Sprintf("[%d]", scene.ID))
			}

			// 2. Fixed-width Scene Name (14 visual chars) for 100% pixel-perfect swatch pill alignment
			nameFormatted := lipgloss.NewStyle().Width(14).Render(scene.Name)

			// 3. Swatch pill (4 visual chars)
			swatch := swatchStyle.Render("████")

			// 4. Status badges
			activeBadge := ""
			if isActiveOnLight {
				activeBadge = " " + styles.StatusSuccess.Render("[ACTIVE]")
			}

			// Build left content line with strictly aligned swatch column
			leftContent := fmt.Sprintf("%s %s %s%s", shortcutStr, nameFormatted, swatch, activeBadge)

			if isCursor {
				applyHint := styles.StatusSuccess.Render(" ↵ Enter")
				leftContent = styles.SelectedItemStyle.Render("▸ " + leftContent + applyHint)
			} else {
				leftContent = "  " + styles.UnselectedItemStyle.Render(leftContent)
			}

			// 5. Very-right scrollbar positioning
			scrollChar := " "
			lineIdx := i - offset
			if scrollbarLines != nil && lineIdx < len(scrollbarLines) {
				scrollChar = scrollbarLines[lineIdx]
			}

			leftWidth := lipgloss.Width(leftContent)
			padSpaces := max(0, innerWidth-leftWidth)
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
		Height(max(height-2, 5)).
		Render(sb.String())
}
