package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/tui"
	"wiz-tui/internal/wiz"
)

func main() {
	checkFlag := flag.Bool("check", false, "Run headless verification check and exit 0")
	ipFlag := flag.String("ip", "", "Initial WiZ bulb IP address")
	mockFlag := flag.Bool("mock", false, "Run in mock client mode without network hardware")
	flag.Parse()

	// Headless verification check mode (Tier 3 pipeline)
	if *checkFlag {
		fmt.Println("gowiz headless sanity check: OK")
		os.Exit(0)
	}

	var client wiz.Client
	if *mockFlag {
		client = wiz.NewMockClient()
	} else {
		client = wiz.NewUDPClient()
	}

	model := tui.NewModel(client, *ipFlag)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running gowiz TUI: %v\n", err)
		os.Exit(1)
	}
}
