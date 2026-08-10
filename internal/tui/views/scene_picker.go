package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderScenePicker(filterQuery string, activeSceneID int, isFocused bool, animFrame int, width, height int) string {
	scenes := wiz.FilterScenes(filterQuery)

	var sb strings.Builder
	title := "🎨 Dynamic Scenes"
	if filterQuery != "" {
		title += fmt.Sprintf(" [Filter: %s]", filterQuery)
	}
	sb.WriteString(styles.SectionTitleStyle.Render(title) + "\n")

	if len(scenes) == 0 {
		sb.WriteString(styles.DimText.Render("No scenes match query.\nPress [Esc] to clear filter."))
	} else {
		maxLines := max(height-4, 5)

		for i, scene := range scenes {
			if i >= maxLines {
				moreCount := len(scenes) - i
				sb.WriteString(styles.DimText.Render(fmt.Sprintf("... +%d more scenes (use j/k to scroll)", moreCount)) + "\n")
				break
			}

			isCurrent := scene.ID == activeSceneID

			// Dynamic color shifting swatch mimicking actual bulb light output
			currentHex := scene.GetAccentColor(animFrame)
			accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(currentHex)).Bold(true)

			prefix := "  "
			if isCurrent {
				prefix = styles.StatusSuccess.Render("▸ ")
			}

			shortcut := "   "
			if scene.ID >= 1 && scene.ID <= 9 {
				shortcut = styles.StatusWarning.Render(fmt.Sprintf("[%d]", scene.ID))
			}

			nameStr := lipgloss.NewStyle().MaxWidth(max(width-28, 8)).Render(scene.Name)
			nameBadge := lipgloss.NewStyle().Foreground(lipgloss.Color(scene.AccentColor)).Bold(true).Render(fmt.Sprintf("%s %-13s", shortcut, nameStr))

			// Live animated bulb output color swatch
			swatch := accentStyle.Render("████")

			desc := ""
			if width > 52 {
				desc = "  " + styles.DimText.Render(scene.Description)
			}

			sb.WriteString(fmt.Sprintf("%s%s %s%s\n", prefix, nameBadge, swatch, desc))
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
