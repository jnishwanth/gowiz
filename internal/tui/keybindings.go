package tui

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
