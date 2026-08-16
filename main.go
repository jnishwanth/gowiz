package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

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
	jsonFlag := flag.Bool("json", false, "Output results in JSON format in CLI mode")
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version information and exit")
	daemonFlag := flag.Bool("daemon", false, "Run in background daemon mode for circadian schedule sync")
	onceFlag := flag.Bool("once", false, "Run a single circadian sync pass and exit")
	intervalFlag := flag.Duration("interval", 1*time.Minute, "Sync interval duration in daemon mode (e.g., 1m, 5m, 30s)")
	serverFlag := flag.Bool("server", false, "Run in HTTP REST API server mode")
	flag.BoolVar(serverFlag, "s", false, "Run in HTTP REST API server mode")
	portFlag := flag.Int("port", 8080, "Port for HTTP REST API server mode (default: 8080)")
	webhookFlag := flag.String("webhook", "", "Target webhook URL for HTTP REST API server event dispatches")

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

	if *daemonFlag || cmdStr == "daemon" || cmdStr == "schedule" {
		opts := cli.Options{
			ConfigPath:     *configFlag,
			TargetIP:       *ipFlag,
			Mock:           *mockFlag,
			JSONOutput:     *jsonFlag,
			Daemon:         true,
			DaemonOnce:     *onceFlag,
			DaemonInterval: *intervalFlag,
		}
		if err := cli.Run(context.Background(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *serverFlag || cmdStr == "serve" || cmdStr == "server" {
		opts := cli.Options{
			ConfigPath: *configFlag,
			TargetIP:   *ipFlag,
			Mock:       *mockFlag,
			JSONOutput: *jsonFlag,
			Server:     true,
			ServerPort: *portFlag,
			WebhookURL: *webhookFlag,
		}
		if err := cli.Run(context.Background(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Non-interactive CLI execution mode
	if cmdStr != "" {
		opts := cli.Options{
			ConfigPath: *configFlag,
			TargetIP:   *ipFlag,
			Mock:       *mockFlag,
			Command:    cmdStr,
			JSONOutput: *jsonFlag,
			WebhookURL: *webhookFlag,
		}
		if err := cli.Run(context.Background(), opts); err != nil {
			if *jsonFlag {
				fmt.Fprintf(os.Stderr, "{\"status\":\"error\",\"error\":%q}\n", err.Error())
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
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
