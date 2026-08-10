package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderControlPanel(dev *wiz.Device, isFocused bool, width, height int) string {
	var sb strings.Builder
	sb.WriteString(styles.SectionTitleStyle.Render("🎛 Control Center") + "\n\n")

	if dev == nil {
		sb.WriteString(styles.DimText.Render("No active device selected."))
	} else {
		powerStatus := styles.StatusSuccess.Render("ON  [o]")
		if !dev.State {
			powerStatus = styles.StatusError.Render("OFF [f/x]")
		}

		sb.WriteString(fmt.Sprintf("%s  Power: %s\n\n", styles.DimText.Render("Target:"), dev.IP))
		sb.WriteString(fmt.Sprintf("State: %s\n\n", powerStatus))

		// Brightness Bar
		sb.WriteString(styles.SectionTitleStyle.Render("Brightness") + fmt.Sprintf(" [%d%%]\n", dev.Brightness))
		p := progress.New(progress.WithoutPercentage())
		p.Width = 32
		p.FullColor = string(styles.Green)
		p.EmptyColor = string(styles.Surface0)
		sb.WriteString(p.ViewAs(float64(dev.Brightness)/100.0) + "\n\n")

		// RGB / Color Temp / Scene Telemetry
		sb.WriteString(styles.SectionTitleStyle.Render("Color & Spectrum Mode") + "\n")
		if dev.SceneID > 0 {
			scene := wiz.GetSceneByID(dev.SceneID)
			accent := lipgloss.NewStyle().Foreground(lipgloss.Color(scene.AccentColor)).Bold(true).Render(scene.Name)
			sb.WriteString(fmt.Sprintf("Active Scene: %s (ID: %d, Speed: %d)\n", accent, dev.SceneID, dev.Speed))
		} else if dev.Temp > 0 {
			sb.WriteString(fmt.Sprintf("Color Temp: %s (%dK)\n", styles.StatusWarning.Render("Tunable White"), dev.Temp))
		} else {
			rgbPill := lipgloss.NewStyle().Foreground(lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", dev.RGB[0], dev.RGB[1], dev.RGB[2]))).Bold(true).Render("████ RGB")
			sb.WriteString(fmt.Sprintf("%s R:%d G:%d B:%d\n", rgbPill, dev.RGB[0], dev.RGB[1], dev.RGB[2]))
		}

		sb.WriteString("\n" + styles.DimText.Render("Vim Shortcuts: [o/x] power • [h/l] dimming • [r/g/b/w] presets"))
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
