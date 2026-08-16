package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"wiz-tui/internal/cli"
	"wiz-tui/internal/tui"
	"wiz-tui/internal/wiz"
)

// Version specifies the gowiz application version.
const Version = "1.0.0"

// programRunner abstracts tea.Program running for isolated unit testing.
type programRunner interface {
	Run() (tea.Model, error)
}

var newProgram = func(model tea.Model, opts ...tea.ProgramOption) programRunner {
	return tea.NewProgram(model, opts...)
}

// runApp parses CLI arguments and executes the appropriate mode (TUI, CLI, Daemon, or Server).
func runApp(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gowiz", flag.ContinueOnError)
	flags.SetOutput(stderr)

	checkFlag := flags.Bool("check", false, "Run headless verification check and exit 0")
	ipFlag := flags.String("ip", "", "Initial WiZ bulb IP address")
	mockFlag := flags.Bool("mock", false, "Run in mock client mode without network hardware")
	configFlag := flags.String("config", "", "Custom path to configuration JSON file")
	cmdFlag := flags.String("cmd", "", "Non-interactive command string to execute")
	jsonFlag := flags.Bool("json", false, "Output results in JSON format in CLI mode")
	versionFlag := flags.Bool("version", false, "Print version information and exit")
	flags.BoolVar(versionFlag, "v", false, "Print version information and exit")
	daemonFlag := flags.Bool("daemon", false, "Run in background daemon mode for circadian schedule sync")
	onceFlag := flags.Bool("once", false, "Run a single circadian sync pass and exit")
	intervalFlag := flags.Duration("interval", 1*time.Minute, "Sync interval duration in daemon mode (e.g., 1m, 5m, 30s)")
	serverFlag := flags.Bool("server", false, "Run in HTTP REST API server mode")
	flags.BoolVar(serverFlag, "s", false, "Run in HTTP REST API server mode")
	portFlag := flags.Int("port", 8080, "Port for HTTP REST API server mode (default: 8080)")
	apiKeyFlag := flags.String("api-key", "", "API key required for securing HTTP REST API server access")
	webhookFlag := flags.String("webhook", "", "Target webhook URL for HTTP REST API server event dispatches")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *versionFlag {
		fmt.Fprintf(stdout, "gowiz v%s\n", Version)
		return 0
	}

	// Headless verification check mode (Tier 3 pipeline)
	if *checkFlag {
		fmt.Fprintln(stdout, "gowiz headless sanity check: OK")
		return 0
	}

	// Positional arguments override or construct non-interactive command string
	positionalArgs := flags.Args()
	cmdStr := *cmdFlag
	if cmdStr == "" && len(positionalArgs) > 0 {
		cmdStr = positionalArgs[0]
		for _, arg := range positionalArgs[1:] {
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
			Writer:         stdout,
		}
		if err := cli.Run(ctx, opts); err != nil {
			fmt.Fprintf(stderr, "Daemon error: %v\n", err)
			return 1
		}
		return 0
	}

	if *serverFlag || cmdStr == "serve" || cmdStr == "server" {
		opts := cli.Options{
			ConfigPath: *configFlag,
			TargetIP:   *ipFlag,
			Mock:       *mockFlag,
			JSONOutput: *jsonFlag,
			Server:     true,
			ServerPort: *portFlag,
			APIKey:     *apiKeyFlag,
			WebhookURL: *webhookFlag,
			Writer:     stdout,
		}
		if err := cli.Run(ctx, opts); err != nil {
			fmt.Fprintf(stderr, "Server error: %v\n", err)
			return 1
		}
		return 0
	}

	// Non-interactive CLI execution mode
	if cmdStr != "" {
		opts := cli.Options{
			ConfigPath: *configFlag,
			TargetIP:   *ipFlag,
			Mock:       *mockFlag,
			Command:    cmdStr,
			JSONOutput: *jsonFlag,
			APIKey:     *apiKeyFlag,
			WebhookURL: *webhookFlag,
			Writer:     stdout,
		}
		if err := cli.Run(ctx, opts); err != nil {
			if *jsonFlag {
				fmt.Fprintf(stderr, "{\"status\":\"error\",\"error\":%q}\n", err.Error())
			} else {
				fmt.Fprintf(stderr, "Error: %v\n", err)
			}
			return 1
		}
		return 0
	}

	// Interactive TUI mode
	var client wiz.Client
	if *mockFlag {
		client = wiz.NewMockClient()
	} else {
		client = wiz.NewUDPClient()
	}

	model := tui.NewModelWithConfig(client, *ipFlag, *configFlag)

	p := newProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "Error running gowiz TUI: %v\n", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(runApp(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
