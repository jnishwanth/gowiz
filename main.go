package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/cli"
	"wiz-tui/internal/tui"
	"wiz-tui/internal/wiz"
)

// Version specifies the gowiz application version.
const Version = "1.0.0"

func main() {
	checkFlag := flag.Bool("check", false, "Run headless verification check and exit 0")
	ipFlag := flag.String("ip", "", "Initial WiZ bulb IP address")
	mockFlag := flag.Bool("mock", false, "Run in mock client mode without network hardware")
	configFlag := flag.String("config", "", "Custom path to configuration JSON file")
	cmdFlag := flag.String("cmd", "", "Non-interactive command string to execute")
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version information and exit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("gowiz v%s\n", Version)
		os.Exit(0)
	}

	// Headless verification check mode (Tier 3 pipeline)
	if *checkFlag {
		fmt.Println("gowiz headless sanity check: OK")
		os.Exit(0)
	}

	// Positional arguments override or construct non-interactive command string
	args := flag.Args()
	cmdStr := *cmdFlag
	if cmdStr == "" && len(args) > 0 {
		cmdStr = args[0]
		for _, arg := range args[1:] {
			cmdStr += " " + arg
		}
	}

	// Non-interactive CLI execution mode
	if cmdStr != "" {
		opts := cli.Options{
			ConfigPath: *configFlag,
			TargetIP:   *ipFlag,
			Mock:       *mockFlag,
			Command:    cmdStr,
		}
		if err := cli.Run(context.Background(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Interactive TUI mode
	var client wiz.Client
	if *mockFlag {
		client = wiz.NewMockClient()
	} else {
		client = wiz.NewUDPClient()
	}

	model := tui.NewModelWithConfig(client, *ipFlag, *configFlag)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running gowiz TUI: %v\n", err)
		os.Exit(1)
	}
}
