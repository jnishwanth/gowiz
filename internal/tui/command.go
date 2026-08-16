package tui

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"wiz-tui/internal/config"
	"wiz-tui/internal/wiz"
)

// CommandActionResult encapsulates the outcome of executing a TUI command line verb.
type CommandActionResult struct {
	Quit             bool
	Help             bool
	Scan             bool
	Undo             bool
	ConfigInfo       bool
	ShowRecent       bool
	ListPresets      bool
	ListCategories   bool
	StatusMsg        string
	PilotParams      *wiz.PilotParams
	SetSleepTimer    int    // sleep timer in seconds, if > 0
	NewDeviceName    string // custom device name to set on active device, if non-empty
	SetDeviceRoom    string // custom room name to set on active device ("CLEAR" to remove), if non-empty
	SavePresetName   string // custom preset name to save active state under, if non-empty
	DeletePresetName string // custom preset name to delete, if non-empty
	ApplyPresetName  string // preset name to apply, if non-empty
	TargetRoom       string // target room group name for batch room commands, if non-empty
	TargetIP         string // target bulb IP to add/connect to
	SetSearchQuery   string // scene filter query to set, if non-empty
	FocusScenes      bool   // whether to focus scene picker panel
	SetFadeDimming   int    // target dimming level (1-100, or 0 for off fade)
	SetFadeDuration  int    // duration of smooth transition in seconds
	SetFadeColorTemp int    // target color temperature in Kelvin (0 if none)
	FadeTurnOff      bool   // whether to turn off light upon fade completion
	FadeLabel        string // user-facing label for active fade badge
	IsFadeCommand    bool   // flag indicating a fade transition request
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

func intPtr(v int) *int {
	return &v
}

// BuiltinPresets provides standard out-of-the-box lighting ambiance presets.
var BuiltinPresets = map[string]config.Preset{
	"evening": {
		Temp:    intPtr(2700),
		Dimming: intPtr(60),
	},
	"movie": {
		SceneID: intPtr(6), // Cozy
		Dimming: intPtr(20),
	},
	"night": {
		SceneID: intPtr(14), // Night light
		Dimming: intPtr(10),
	},
	"focus": {
		Temp:    intPtr(6500),
		Dimming: intPtr(100),
	},
	"work": {
		Temp:    intPtr(6500),
		Dimming: intPtr(100),
	},
	"relax": {
		SceneID: intPtr(9), // Relax
		Dimming: intPtr(50),
	},
	"party": {
		SceneID: intPtr(4), // Party
		Dimming: intPtr(100),
	},
}

// PresetToPilotParams converts a config.Preset into a wiz.PilotParams payload.
func PresetToPilotParams(p config.Preset) wiz.PilotParams {
	st := true
	params := wiz.PilotParams{
		State:   &st,
		Dimming: p.Dimming,
		Temp:    p.Temp,
		R:       p.R,
		G:       p.G,
		B:       p.B,
		SceneID: p.SceneID,
		Speed:   p.Speed,
	}
	return params
}

// PresetFromDevice constructs a config.Preset snapshot from an active wiz.Device state.
func PresetFromDevice(dev *wiz.Device) config.Preset {
	p := config.Preset{}
	if dev == nil {
		return p
	}
	dim := dev.Brightness
	p.Dimming = &dim

	if dev.SceneID > 0 {
		sc := dev.SceneID
		p.SceneID = &sc
		if dev.Speed > 0 {
			sp := dev.Speed
			p.Speed = &sp
		}
	} else if dev.Temp > 0 {
		t := dev.Temp
		p.Temp = &t
	} else {
		r, g, b := dev.RGB[0], dev.RGB[1], dev.RGB[2]
		p.R, p.G, p.B = &r, &g, &b
	}
	return p
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

	// Smooth Dimming & Transition Fade
	r.Register("fade", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :fade <level|off> [durationSec] (e.g. :fade 20 30)"}
		}

		targetStr := strings.ToLower(args[0])
		duration := 30
		if len(args) > 1 {
			if d, err := strconv.Atoi(args[1]); err == nil && d > 0 {
				duration = d
			}
		}

		if targetStr == "off" || targetStr == "0" {
			return CommandActionResult{
				IsFadeCommand:   true,
				SetFadeDimming:  10,
				SetFadeDuration: duration,
				FadeTurnOff:     true,
				FadeLabel:       "🌆 Fade Off",
				StatusMsg:       fmt.Sprintf("Fading lights off over %d seconds...", duration),
			}
		}

		if level, err := strconv.Atoi(targetStr); err == nil {
			clamped := wiz.Clamp(level, 10, 100)
			return CommandActionResult{
				IsFadeCommand:   true,
				SetFadeDimming:  clamped,
				SetFadeDuration: duration,
				FadeLabel:       fmt.Sprintf("🌆 Fade (%d%%)", clamped),
				StatusMsg:       fmt.Sprintf("Fading brightness to %d%% over %d seconds...", clamped, duration),
			}
		}

		return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid fade level: %s", args[0])}
	})

	// Sunrise Simulation
	r.Register("sunrise", func(args []string, activeDev *wiz.Device) CommandActionResult {
		duration := 60
		if len(args) > 0 {
			if d, err := strconv.Atoi(args[0]); err == nil && d > 0 {
				duration = d
			}
		}
		return CommandActionResult{
			IsFadeCommand:    true,
			SetFadeDimming:   100,
			SetFadeColorTemp: 4200,
			SetFadeDuration:  duration,
			FadeLabel:        "🌅 Sunrise",
			StatusMsg:        fmt.Sprintf("Starting sunrise simulation (%d seconds)...", duration),
		}
	})

	// Sunset Simulation & Dynamic Scene Shortcut
	r.Register("sunset", func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) > 0 {
			duration := 60
			if d, err := strconv.Atoi(args[0]); err == nil && d > 0 {
				duration = d
			}
			return CommandActionResult{
				IsFadeCommand:    true,
				SetFadeDimming:   10,
				SetFadeColorTemp: 2700,
				SetFadeDuration:  duration,
				FadeTurnOff:      true,
				FadeLabel:        "🌆 Sunset",
				StatusMsg:        fmt.Sprintf("Starting sunset simulation (%d seconds)...", duration),
			}
		}
		if sc, found := wiz.GetSceneByName("Sunset"); found {
			params := wiz.NewSceneParams(sc.ID)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Scene set: %s", sc.Name),
				PilotParams: &params,
			}
		}
		return CommandActionResult{StatusMsg: "Scene not found: Sunset"}
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

	// Config Status Verb
	r.Register("config", func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{
			ConfigInfo: true,
			StatusMsg:  "Querying configuration location and saved aliases...",
		}
	})

	// IP Target / Connect Verbs
	ipHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :ip <ip address>"}
		}
		ipStr := strings.TrimSpace(args[0])
		if net.ParseIP(ipStr) == nil {
			return CommandActionResult{StatusMsg: fmt.Sprintf("Invalid IP address: %s", ipStr)}
		}
		return CommandActionResult{
			TargetIP:  ipStr,
			StatusMsg: fmt.Sprintf("Connecting to WiZ bulb at %s...", ipStr),
		}
	}
	r.Register("ip", ipHandler)
	r.Register("connect", ipHandler)
	r.Register("add", ipHandler)

	// Recent Targets Verb
	r.Register("recent", func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{
			ShowRecent: true,
		}
	})

	// Room Assignment & Group Batch Commands
	roomHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			if activeDev != nil && activeDev.Room != "" {
				return CommandActionResult{StatusMsg: fmt.Sprintf("Light [%s] assigned to room '%s'", activeDev.IP, activeDev.Room)}
			}
			return CommandActionResult{StatusMsg: "Usage: :room <room name> (or :room clear)"}
		}
		if len(args) == 1 && strings.EqualFold(args[0], "clear") {
			if activeDev != nil {
				return CommandActionResult{
					SetDeviceRoom: "CLEAR",
					StatusMsg:     fmt.Sprintf("Cleared room assignment for light [%s]", activeDev.IP),
				}
			}
			return CommandActionResult{StatusMsg: "No active device to clear room."}
		}
		roomName := strings.Join(args, " ")
		if activeDev != nil {
			return CommandActionResult{
				SetDeviceRoom: roomName,
				StatusMsg:     fmt.Sprintf("Assigned light [%s] to room '%s'", activeDev.IP, roomName),
			}
		}
		return CommandActionResult{StatusMsg: "No active device to assign room."}
	}
	r.Register("room", roomHandler)

	groupHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{StatusMsg: "Usage: :group <room name> <action> (e.g. :group Living Room on)"}
		}

		subIdx := -1
		for i := len(args) - 1; i >= 1; i-- {
			verb := strings.ToLower(args[i])
			if _, found := r.handlers[verb]; found {
				subIdx = i
				break
			}
		}

		if subIdx != -1 {
			roomName := strings.Join(args[:subIdx], " ")
			subCmdStr := strings.Join(args[subIdx:], " ")
			subRes := r.Execute(subCmdStr, activeDev)
			subRes.TargetRoom = roomName
			if subRes.PilotParams != nil || subRes.ApplyPresetName != "" {
				subRes.StatusMsg = fmt.Sprintf("Group command '%s' sent to room '%s'", subCmdStr, roomName)
			}
			return subRes
		}

		roomName := strings.Join(args, " ")
		return CommandActionResult{
			TargetRoom: roomName,
			StatusMsg:  fmt.Sprintf("Targeting group/room '%s'", roomName),
		}
	}
	r.Register("group", groupHandler)

	// Preset Management & Activation Verbs
	presetHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			return CommandActionResult{ListPresets: true}
		}

		sub := strings.ToLower(args[0])
		if sub == "list" {
			return CommandActionResult{ListPresets: true}
		}

		if sub == "save" {
			if len(args) < 2 {
				return CommandActionResult{StatusMsg: "Usage: :preset save <name>"}
			}
			presetName := strings.ToLower(strings.Join(args[1:], "_"))
			return CommandActionResult{
				SavePresetName: presetName,
				StatusMsg:      fmt.Sprintf("Saving current state as preset '%s'...", presetName),
			}
		}

		if sub == "delete" || sub == "rm" || sub == "remove" {
			if len(args) < 2 {
				return CommandActionResult{StatusMsg: "Usage: :preset delete <name>"}
			}
			presetName := strings.ToLower(strings.Join(args[1:], "_"))
			return CommandActionResult{
				DeletePresetName: presetName,
				StatusMsg:        fmt.Sprintf("Deleting preset '%s'...", presetName),
			}
		}

		presetName := strings.ToLower(strings.Join(args, "_"))
		if bp, found := BuiltinPresets[presetName]; found {
			params := PresetToPilotParams(bp)
			return CommandActionResult{
				StatusMsg:   fmt.Sprintf("Applied preset '%s'", presetName),
				PilotParams: &params,
			}
		}

		return CommandActionResult{
			ApplyPresetName: presetName,
			StatusMsg:       fmt.Sprintf("Applying preset '%s'...", presetName),
		}
	}
	r.Register("preset", presetHandler)
	r.Register("presets", func(args []string, activeDev *wiz.Device) CommandActionResult {
		return CommandActionResult{ListPresets: true}
	})

	// Scene Category Verbs
	categoryHandler := func(args []string, activeDev *wiz.Device) CommandActionResult {
		if len(args) == 0 {
			cats := wiz.GetSceneCategories()
			return CommandActionResult{
				ListCategories: true,
				StatusMsg:      fmt.Sprintf("Categories (%d): %s", len(cats), strings.Join(cats, ", ")),
			}
		}
		catQuery := strings.Join(args, " ")
		matched := wiz.FilterScenesByCategory(catQuery)
		if len(matched) == 0 {
			matched = wiz.FilterScenes(catQuery)
		}
		if len(matched) == 0 {
			return CommandActionResult{StatusMsg: fmt.Sprintf("No scenes found matching category '%s'", catQuery)}
		}
		return CommandActionResult{
			SetSearchQuery: catQuery,
			FocusScenes:    true,
			StatusMsg:      fmt.Sprintf("Filtered scenes by category '%s' (%d match(es))", catQuery, len(matched)),
		}
	}
	r.Register("category", categoryHandler)
	r.Register("cat", categoryHandler)
	r.Register("categories", func(args []string, activeDev *wiz.Device) CommandActionResult {
		cats := wiz.GetSceneCategories()
		return CommandActionResult{
			ListCategories: true,
			StatusMsg:      fmt.Sprintf("Categories (%d): %s", len(cats), strings.Join(cats, ", ")),
		}
	})

	// Direct Scene Shortcut Verbs
	sceneShortcuts := map[string]string{
		"ocean":      "Ocean",
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
