package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/wiz"
)

func RenderControlPanel(dev *wiz.Device, isFocused bool, sleepTimerSecs int, width, height int) string {
	innerWidth := max(width-6, 18)
	header := RenderSectionHeader("🎛 Control Center", isFocused, innerWidth)

	var sb strings.Builder
	sb.WriteString(header + "\n")

	if dev == nil {
		sb.WriteString(styles.DimText.Render("No active device selected."))
	} else {
		powerStatus := styles.StatusSuccess.Render("● ON")
		if !dev.State {
			powerStatus = styles.StatusError.Render("○ OFF")
		}

		displayName := dev.IP
		defaultName := fmt.Sprintf("WiZ Light (%s)", dev.IP)
		defaultFallbackName := fmt.Sprintf("WiZ Light (%s) [Fallback]", dev.IP)
		if dev.Name != "" && dev.Name != defaultName && dev.Name != defaultFallbackName {
			displayName = fmt.Sprintf("%s (%s)", dev.Name, dev.IP)
		}

		targetIP := lipgloss.NewStyle().Foreground(styles.Mauve).Bold(true).Render(displayName)
		rssiStr := "📶 Online"
		if dev.Rssi != 0 {
			rssiStr = fmt.Sprintf("📶 %d dBm", dev.Rssi)
		}
		rssiPill := lipgloss.NewStyle().Foreground(styles.Subtext0).Render(rssiStr)

		sb.WriteString(fmt.Sprintf("%s │ %s │ %s\n", targetIP, powerStatus, rssiPill))

		// Sleep Countdown Badge if running
		if sleepTimerSecs > 0 {
			mins := sleepTimerSecs / 60
			secs := sleepTimerSecs % 60
			timerBadge := lipgloss.NewStyle().Foreground(styles.Crust).Background(styles.Peach).Bold(true).Padding(0, 1).Render(fmt.Sprintf("⏳ Sleep: %02d:%02d", mins, secs))
			sb.WriteString(timerBadge + "\n")
		}

		// Brightness Bar
		barWidth := max(innerWidth-18, 10)
		sb.WriteString(fmt.Sprintf("Brightness [%d%%]\n", dev.Brightness))
		p := progress.New(progress.WithoutPercentage())
		p.Width = barWidth
		p.FullColor = string(styles.Green)
		p.EmptyColor = string(styles.Surface0)
		sb.WriteString(p.ViewAs(float64(dev.Brightness)/100.0) + "\n")

		// Mode Specific Display (Scene / White Temp Gradient / Rich RGB Swatch)
		if dev.SceneID > 0 {
			scene := wiz.GetSceneByID(dev.SceneID)
			accent := lipgloss.NewStyle().Foreground(lipgloss.Color(scene.AccentColor)).Bold(true).Render(scene.Name)
			sb.WriteString(fmt.Sprintf("Active Scene: %s (ID: %d)\n", accent, dev.SceneID))

			// Speed Bar
			sb.WriteString(fmt.Sprintf("Speed [%d%%] ", dev.Speed))
			spBar := progress.New(progress.WithoutPercentage())
			spBar.Width = max(barWidth-10, 8)
			spBar.FullColor = string(styles.Teal)
			spBar.EmptyColor = string(styles.Surface0)
			sb.WriteString(spBar.ViewAs(float64(dev.Speed)/200.0) + "\n")
		} else if dev.Temp > 0 {
			sb.WriteString(fmt.Sprintf("Color Temp: %s (%dK)\n", styles.StatusWarning.Render("Tunable White"), dev.Temp))

			// Kelvin Warm-to-Cool Gradient Bar
			kelvinRatio := float64(dev.Temp-2200) / float64(6500-2200)
			cctBarWidth := max(innerWidth-20, 8)
			cctBar := progress.New(progress.WithoutPercentage())
			cctBar.Width = cctBarWidth
			cctBar.FullColor = string(styles.Yellow)
			cctBar.EmptyColor = string(styles.Sky)
			sb.WriteString("2200K 🟨" + cctBar.ViewAs(kelvinRatio) + "🟦 6500K\n")
		} else {
			hexColor := fmt.Sprintf("#%02x%02x%02x", dev.RGB[0], dev.RGB[1], dev.RGB[2])
			rgbSwatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hexColor)).Bold(true).Render("████████ " + hexColor)
			sb.WriteString(fmt.Sprintf("RGB Color: %s (R:%d G:%d B:%d)\n", rgbSwatch, dev.RGB[0], dev.RGB[1], dev.RGB[2]))
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
