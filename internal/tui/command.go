package tui

import (
	"fmt"
	"strconv"
	"strings"
	"wiz-tui/internal/wiz"
)

// CommandActionResult encapsulates the outcome of executing a TUI command line verb.
type CommandActionResult struct {
	Quit          bool
	Help          bool
	Scan          bool
	Undo          bool
	StatusMsg     string
	PilotParams   *wiz.PilotParams
	SetSleepTimer int    // sleep timer in seconds, if > 0
	NewDeviceName string // custom device name to set on active device, if non-empty
}

// CommandHandler defines a function signature for processing command line arguments.
type CommandHandler func(args []string, activeDev *wiz.Device) CommandActionResult

// CommandRegistry manages registered TUI command verbs and aliases.
type CommandRegistry struct {
	handlers map[string]CommandHandler
}

// NewCommandRegistry initializes a default TUI command registry.
func NewCommandRegistry() *CommandRegistry {
	reg := &CommandRegistry{
		handlers: make(map[string]CommandHandler),
	}
	reg.registerDefaults()
	return reg
}

// Register adds or replaces a handler for a given verb or alias.
func (r *CommandRegistry) Register(verb string, handler CommandHandler) {
	r.handlers[strings.ToLower(verb)] = handler
}

// Execute processes a raw command string and returns the resulting CommandActionResult.
func (r *CommandRegistry) Execute(cmdStr string, activeDev *wiz.Device) CommandActionResult {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return CommandActionResult{}
	}

	parts := strings.Fields(cmdStr)
	verb := strings.ToLower(parts[0])
	args := parts[1:]

	handler, found := r.handlers[verb]
	if !found {
		return CommandActionResult{
			StatusMsg: fmt.Sprintf("Unknown command: :%s", cmdStr),
		}
	}

	return handler(args, activeDev)
}

var defaultRegistry = NewCommandRegistry()

// ExecuteCommand runs a command string against the default global command registry.
func ExecuteCommand(cmdStr string, activeDev *wiz.Device) CommandActionResult {
	return defaultRegistry.Execute(cmdStr, activeDev)
}

// ParseHexColor converts a hex color string (e.g., "#FF5500", "FF5500", "#F50", "F50") into (r, g, b) integers.
func ParseHexColor(hexStr string) (int, int, int, error) {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) == 3 {
		hexStr = string([]byte{hexStr[0], hexStr[0], hexStr[1], hexStr[1], hexStr[2], hexStr[2]})
	}
	if len(hexStr) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex length")
	}

	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex digits: %w", err)
	}

	r := int((val >> 16) & 0xFF)
	g := int((val >> 8) & 0xFF)
	b := int(val & 0xFF)

	return r, g, b, nil
}

func (r *CommandRegistry) registerDefaults() {
	// Quit verbs
	quitHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{Quit: true}
	}
	r.Register("q", quitHandler)
	r.Register("quit", quitHandler)
	r.Register("exit", quitHandler)

	// Help verbs
	helpHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{Help: true}
	}
	r.Register("h", helpHandler)
	r.Register("help", helpHandler)

	// Power ON
	r.Register("on", func(args []string, activeDev *wiz.Device) CommandActionResult {
		params := wiz.NewPowerParams(true)
		return CommandActionResult{
			StatusMsg:   "Turning on lights...",
			PilotParams: &params,
		}
	})

	// Power OFF
	r.Register("off", func(args []string, activeDev *wiz.Device) CommandActionResult {
		params := wiz.NewPowerParams(false)
		return CommandActionResult{
			StatusMsg:   "Turning off lights...",
			PilotParams: &params,
		}
	})

	// Power toggle / explicit
	r.Register("power", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) > 0 {
			arg := strings.ToLower(args[0])
			if arg == "on" || arg == "true" || arg == "1" {
				params := wiz.NewPowerParams(true)
				return CommandActionResult{StatusMsg: "Turning on lights...", PilotParams: &params}
			} else if arg == "off" || arg == "false" || arg == "0" {
				params := wiz.NewPowerParams(false)
				return CommandActionResult{StatusMsg: "Turning off lights...", PilotParams: &params}
			}
		}
		if activeDev != nil {
			params := wiz.NewPowerParams(!activeDev.State)
			return CommandActionResult{PilotParams: &params}
		}
		params := wiz.NewPowerParams(true)
		return CommandActionResult{PilotParams: &params}
	})

	r.Register("toggle", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if activeDev != nil {
			params := wiz.NewPowerParams(!activeDev.State)
			return CommandActionResult{PilotParams: &params}
		}
		params := wiz.NewPowerParams(true)
		return CommandActionResult{PilotParams: &params}
	})

	// Scan
	r.Register("scan", func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{
			Scan:      true,
			StatusMsg: "Scanning network...",
		}
	})

	// Scene
	r.Register("scene", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :scene <name|id>"}
		}
		arg := strings.Join(args, " ")
		if id, err := strconv.Atoi(arg); err == nil {
			scene := wiz.GetSceneByID(id)
			params := wiz.NewSceneParams(id)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Scene set: %s", scene.Name),
				PilotParams: &params,
			}
		} else if scene, found := wiz.GetSceneByName(arg); found {
			params := wiz.NewSceneParams(scene.ID)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Scene set: %s", scene.Name),
				PilotParams: &params,
			}
		}
		return CommandActionResult{StatusMsg: fmt.Sprintf("Unknown scene: %s", arg)}
	})

	// Dimming
	r.Register("dim", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :dim <10-100>"}
		}
		if dim, err := strconv.Atoi(args[0]); err == nil {
			params := wiz.NewDimmingParams(dim)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Brightness set: %d%%", dim),
				PilotParams: &params,
			}
		}
		return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid brightness: %s", args[0])}
	})

	// Color Temperature (Kelvin)
	r.Register("temp", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :temp <2200-6500>"}
		}
		if temp, err := strconv.Atoi(args[0]); err == nil {
			params := wiz.NewTempParams(temp)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Color temp set: %dK", temp),
				PilotParams: &params,
			}
		}
		return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid temperature: %s", args[0])}
	})

	// Kelvin Presets
	r.Register("warm", func(args []string, activeDev *wiz.Device) CommandActionResult {
		params := wiz.NewTempParams(2700)
		return CommandActionResult{
			StatusMsg:   "Color temp set: Warm White (2700K)",
			PilotParams: &params,
		}
	})

	r.Register("cool", func(args []string, activeDev *wiz.Device) CommandActionResult {
		params := wiz.NewTempParams(4200)
		return CommandActionResult{
			StatusMsg:   "Color temp set: Cool White (4200K)",
			PilotParams: &params,
		}
	})

	r.Register("daylight", func(args []string, activeDev *wiz.Device) CommandActionResult {
		params := wiz.NewTempParams(6500)
		return CommandActionResult{
			StatusMsg:   "Color temp set: Daylight (6500K)",
			PilotParams: &params,
		}
	})

	// RGB Color (0-255)
	r.Register("rgb", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) < 3 {
			return CommandActionResult{StatusMsg: "Usage: :rgb <r> <g> <b>"}
		}
		rVal, err1 := strconv.Atoi(args[0])
		gVal, err2 := strconv.Atoi(args[1])
		bVal, err3 := strconv.Atoi(args[2])
		if err1 != nil || err2 != nil || err3 != nil {
			return CommandActionResult{StatusMsg: "Invalid RGB values"}
		}
		params := wiz.NewRGBParams(rVal, gVal, bVal)
		return CommandActionResult{
			StatusMsg:   fmt.Sprintf("RGB color set: R:%d G:%d B:%d", rVal, gVal, bVal),
			PilotParams: &params,
		}
	})

	// Hex Color (#RRGGBB / #RGB / RRGGBB)
	hexHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :hex <#hexcode>"}
		}
		rVal, gVal, bVal, err := ParseHexColor(args[0])
		if err != nil {
			return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid hex color: %s", args[0])}
		}
		params := wiz.NewRGBParams(rVal, gVal, bVal)
		cleanHex := strings.ToUpper(strings.TrimPrefix(args[0], "#"))
		if len(cleanHex) == 3 {
			cleanHex = string([]byte{cleanHex[0], cleanHex[0], cleanHex[1], cleanHex[1], cleanHex[2], cleanHex[2]})
		}
		return CommandActionResult{
			StatusMsg:   fmt.Sprintf("HEX color set: #%s (R:%d G:%d B:%d)", cleanHex, rVal, gVal, bVal),
			PilotParams: &params,
		}
	}
	r.Register("hex", hexHandler)
	r.Register("color", hexHandler)

	// Sleep Timer
	r.Register("timer", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :timer <minutes>"}
		}
		if mins, err := strconv.Atoi(args[0]); err == nil && mins > 0 {
			return CommandActionResult{
				SetSleepTimer: mins * 60,
				StatusMsg:     fmt.Sprintf("Sleep timer set: %d minutes.", mins),
			}
		}
		return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid timer duration: %s", args[0])}
	})

	// Scene Speed
	r.Register("speed", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :speed <20-200>"}
		}
		if sp, err := strconv.Atoi(args[0]); err == nil {
			if activeDev != nil && activeDev.SceneID > 0 {
				spClamped := wiz.Clamp(sp, 20, 200)
				params := wiz.NewSceneParams(activeDev.SceneID, spClamped)
				return CommandActionResult{
					StatusMsg:   fmt.Sprintf("Scene speed set: %d%%", spClamped),
					PilotParams: &params,
				}
			}
			return CommandActionResult{StatusMsg: "Speed requires an active dynamic scene."}
		}
		return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid speed percentage: %s", args[0])}
	})

	// Undo verbs
	undoHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{Undo: true}
	}
	r.Register("u", undoHandler)
	r.Register("undo", undoHandler)

	// Device Custom Naming / Aliasing
	nameHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :name <custom name>"}
		}
		newName := strings.Join(args, " ")
		if activeDev != nil {
			return CommandActionResult{
				NewDeviceName: newName,
				StatusMsg:     fmt.Sprintf("Renamed light [%s] to '%s'", activeDev.IP, newName),
			}
		}
		return CommandActionResult{StatusMsg: "No active device to rename."}
	}
	r.Register("name", nameHandler)
	r.Register("rename", nameHandler)

	// Direct Scene Shortcut Verbs
	sceneShortcuts := map[string]string{
		"ocean":      "Ocean",
		"sunset":     "Sunset",
		"party":      "Party",
		"cozy":       "Cozy",
		"forest":     "Forest",
		"fireplace":  "Fireplace",
		"romance":    "Romance",
		"relax":      "Relax",
		"focus":      "Focus",
		"nightlight": "Night light",
	}
	for verb, sceneName := range sceneShortcuts {
		scName := sceneName
		r.Register(verb, func(args []string, activeDev *wiz.Device) CommandActionResult {
			if sc, found := wiz.GetSceneByName(scName); found {
				params := wiz.NewSceneParams(sc.ID)
				return CommandActionResult{
					StatusMsg:   fmt.Sprintf("Scene set: %s", sc.Name),
					PilotParams: &params,
				}
			}
			return CommandActionResult{StatusMsg: fmt.Sprintf("Scene not found: %s", scName)}
		})
	}
}
