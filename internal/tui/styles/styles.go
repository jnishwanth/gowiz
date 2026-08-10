package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Macchiato Theme Colors
var (
	Rosewater = lipgloss.Color("#f4dbd6")
	Flamingo  = lipgloss.Color("#f0c6c6")
	Pink      = lipgloss.Color("#f5bde6")
	Mauve     = lipgloss.Color("#c6a0f6")
	Red       = lipgloss.Color("#ed8796")
	Maroon    = lipgloss.Color("#ee99a0")
	Peach     = lipgloss.Color("#f5a97f")
	Yellow    = lipgloss.Color("#eed49f")
	Green     = lipgloss.Color("#a6da95")
	Teal      = lipgloss.Color("#8bd5ca")
	Sky       = lipgloss.Color("#91d7e3")
	Sapphire  = lipgloss.Color("#7dc4e4")
	Blue      = lipgloss.Color("#8aadf4")
	Lavender  = lipgloss.Color("#b7bdf8")
	Text      = lipgloss.Color("#cad3f5")
	Subtext1  = lipgloss.Color("#b8c0e0")
	Subtext0  = lipgloss.Color("#a5adcb")
	Overlay2  = lipgloss.Color("#939ab7")
	Overlay0  = lipgloss.Color("#6e738d")
	Surface0  = lipgloss.Color("#363a4f")
	Base      = lipgloss.Color("#24273a")
	Mantle    = lipgloss.Color("#1e2030")
	Crust     = lipgloss.Color("#181926")
)

var (
	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Mauve).
			Background(Base).
			Padding(1, 2)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(Lavender).
				Background(Base).
				Padding(1, 2)

	// Titles & Headers
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Mauve).
			Background(Surface0).
			Padding(0, 2)

	SectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Blue).
				MarginBottom(1)

	// Mode Pills (Vim Status)
	NormalModeBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(Crust).
			Background(Mauve).
			Padding(0, 1)

	VisualModeBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(Crust).
			Background(Yellow).
			Padding(0, 1)

	CommandModeBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(Crust).
				Background(Green).
				Padding(0, 1)

	SearchModeBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(Crust).
			Background(Peach).
			Padding(0, 1)

	HelpModeBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(Crust).
			Background(Teal).
			Padding(0, 1)

	// Item selections
	SelectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Mauve).
				Background(Surface0)

	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(Text)

	DimText = lipgloss.NewStyle().Foreground(Overlay0)

	StatusSuccess = lipgloss.NewStyle().Foreground(Green).Bold(true)
	StatusError   = lipgloss.NewStyle().Foreground(Red).Bold(true)
	StatusWarning = lipgloss.NewStyle().Foreground(Yellow).Bold(true)
	StatusInfo    = lipgloss.NewStyle().Foreground(Sapphire)

	// Command Bar Input
	CommandPromptStyle = lipgloss.NewStyle().Foreground(Green).Bold(true)
	SearchPromptStyle  = lipgloss.NewStyle().Foreground(Peach).Bold(true)
)
