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

	var sb strings.Builder
	title := "🎨 Dynamic Scenes"
	if filterQuery != "" {
		title += fmt.Sprintf(" [Filter: %s]", filterQuery)
	}
	if isFocused {
		title += " [ACTIVE]"
		sb.WriteString(styles.FocusedSectionTitleStyle.Render(title) + "\n")
	} else {
		sb.WriteString(styles.SectionTitleStyle.Render(title) + "\n")
	}

	if len(scenes) == 0 {
		sb.WriteString(styles.DimText.Render("No scenes match query.\nPress [Esc] to clear filter."))
	} else {
		maxLines := max(height-5, 4)

		for i, scene := range scenes {
			if i >= maxLines {
				moreCount := len(scenes) - i
				sb.WriteString(styles.DimText.Render(fmt.Sprintf("... +%d more scenes (use j/k to scroll)", moreCount)) + "\n")
				break
			}

			isCursor := isFocused && i == sceneCursor
			isActiveOnLight := scene.ID == activeSceneID

			currentHex := scene.GetAccentColor(animFrame)
			accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(currentHex)).Bold(true)

			shortcut := "   "
			if scene.ID >= 1 && scene.ID <= 9 {
				shortcut = styles.StatusWarning.Render(fmt.Sprintf("[%d]", scene.ID))
			}

			nameStr := lipgloss.NewStyle().MaxWidth(max(width-30, 8)).Render(scene.Name)
			nameBadge := accentStyle.Render(fmt.Sprintf("%s %-12s", shortcut, nameStr))

			swatch := accentStyle.Render("████")

			activeBadge := ""
			if isActiveOnLight {
				activeBadge = " " + styles.StatusSuccess.Render("[ACTIVE ON LIGHT]")
			}

			line := fmt.Sprintf("%s %s%s", nameBadge, swatch, activeBadge)

			if isCursor {
				applyHint := styles.StatusSuccess.Render(" ↵ Press Enter to Apply")
				sb.WriteString(styles.SelectedItemStyle.Render("▸ "+line+applyHint) + "\n")
			} else {
				sb.WriteString("  " + line + "\n")
			}
		}

		if isFocused {
			sb.WriteString("\n" + styles.DimText.Render("💡 Press [j/k] to navigate • [Enter/Space] to Apply scene to bulb"))
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(max(width-2, 20)).
		Height(max(height-2, 6)).
		Render(sb.String())
}
