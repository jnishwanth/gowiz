package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeVisual
	ModeCommand
	ModeSearch
	ModeHelp
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeVisual:
		return "VISUAL"
	case ModeCommand:
		return "COMMAND"
	case ModeSearch:
		return "SEARCH"
	case ModeHelp:
		return "HELP"
	default:
		return "UNKNOWN"
	}
}

type Panel int

const (
	PanelDevices Panel = iota
	PanelControl
	PanelScenes
)

func (p Panel) Next() Panel {
	return (p + 1) % 3
}

func (p Panel) Prev() Panel {
	if p == 0 {
		return PanelScenes
	}
	return p - 1
}

func (p Panel) String() string {
	switch p {
	case PanelDevices:
		return "1. Bulbs"
	case PanelControl:
		return "2. Controls"
	case PanelScenes:
		return "3. Scenes"
	default:
		return "Unknown"
	}
}

// KeyMap defines declarative keyboard bindings using bubbles/key
type KeyMap struct {
	Tab      key.Binding
	ShiftTab key.Binding
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Power    key.Binding
	Select   key.Binding
	Visual   key.Binding
	Command  key.Binding
	Search   key.Binding
	Help     key.Binding
	Quit     key.Binding
	Scan     key.Binding
	Timer    key.Binding
}

var DefaultKeyMap = KeyMap{
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("j/k", "navigate")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/k", "navigate")),
	Left:     key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/l", "dim/speed")),
	Right:    key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("h/l", "dim/speed")),
	Power:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "power toggle")),
	Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
	Visual:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "visual mode")),
	Command:  key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command bar")),
	Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search scenes")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Scan:     key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rescan network")),
	Timer:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "15m sleep timer")),
}
