package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderScenePicker(filterQuery string, activeSceneID int, isFocused bool, width, height int) string {
	scenes := wiz.FilterScenes(filterQuery)

	var sb strings.Builder
	title := "🎨 WiZ Dynamic Scenes (32)"
	if filterQuery != "" {
		title += fmt.Sprintf(" [Filter: %s]", filterQuery)
	}
	sb.WriteString(styles.SectionTitleStyle.Render(title) + "\n\n")

	if len(scenes) == 0 {
		sb.WriteString(styles.DimText.Render("No scenes match search query.\nPress [Esc] to clear filter."))
	} else {
		// Render scene list
		for _, scene := range scenes {
			isCurrent := scene.ID == activeSceneID
			accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(scene.AccentColor)).Bold(true)

			prefix := "  "
			if isCurrent {
				prefix = styles.StatusSuccess.Render("▸ ")
			}

			badge := accentStyle.Render(fmt.Sprintf("[%2d] %-15s", scene.ID, scene.Name))
			category := styles.DimText.Render(fmt.Sprintf("(%s)", scene.Category))

			sb.WriteString(fmt.Sprintf("%s%s %s\n", prefix, badge, category))
		}
	}

	panelStyle := styles.PanelStyle
	if isFocused {
		panelStyle = styles.ActivePanelStyle
	}

	return panelStyle.
		Width(width).
		Height(height).
		Render(sb.String())
}
