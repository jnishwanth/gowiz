package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// SendOSCNotification emits OSC 99 / OSC 777 terminal notifications so background events
// (e.g. sleep timer expiration or scan completion) notify the user's terminal desktop.
func SendOSCNotification(title, message string) tea.Cmd {
	return func() tea.Msg {
		seq777 := ansi.URxvtExt("notify", title, message)
		seq99Title := ansi.DesktopNotification(title, "i=gowiz", "p=title", "a=gowiz")
		seq99Body := ""
		if message != "" {
			seq99Body = ansi.DesktopNotification(message, "i=gowiz", "p=body", "a=gowiz")
		}
		fmt.Print(seq777 + seq99Title + seq99Body)
		return nil
	}
}
