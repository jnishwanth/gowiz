package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/config"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/tui/views"
	"wiz-tui/internal/wiz"
)

type Model struct {
	client              wiz.Client
	Registry            *wiz.DeviceRegistry
	configManager       *config.Manager
	mode                Mode
	activePanel         Panel
	commandBuffer       string
	searchQuery         string
	statusMessage       string
	statusTimer         time.Time
	width               int
	height              int
	sceneCursor         int
	deviceCursor        int
	sleepTimerSecs      int
	fadeActive          bool
	fadeStartDim        int
	fadeTargetDim       int
	fadeStartTemp       int
	fadeTargetTemp      int
	fadeDurationSecs    int
	fadeElapsedSecs     int
	fadeTurnOffOnFinish bool
	fadeLabel           string
	animFrame           int
	undoStack           map[string][]wiz.PilotParams
}

func NewModelWithConfig(client wiz.Client, initialIP string, configPath string) Model {
	cfgMgr := config.NewManager(configPath)
	_ = cfgMgr.Load()
	cfg := cfgMgr.GetConfig()

	reg := wiz.NewDeviceRegistry()
	reg.ApplyAliases(cfg.DeviceAliases)
	reg.ApplyRooms(cfg.DeviceRooms)

	for _, recIP := range cfg.RecentIPs {
		if recIP != "" && recIP != wiz.FallbackIP {
			reg.AddOrUpdate(wiz.NewDevice(recIP))
		}
	}

	targetIP := initialIP
	if targetIP == "" || targetIP == wiz.FallbackIP {
		if cfg.LastActiveIP != "" {
			targetIP = cfg.LastActiveIP
		}
	}

	if targetIP != "" && targetIP != wiz.FallbackIP {
		reg.AddOrUpdate(wiz.NewDevice(targetIP))
		reg.SetActive(targetIP)
	}

	devCursor := 0
	if activeDev, ok := reg.GetActive(); ok {
		devices := reg.List()
		for i, d := range devices {
			if d.IP == activeDev.IP {
				devCursor = i
				break
			}
		}
	}

	return Model{
		client:        client,
		Registry:      reg,
		configManager: cfgMgr,
		mode:          ModeNormal,
		activePanel:   PanelDevices,
		width:         80,
		height:        24,
		sceneCursor:   0,
		deviceCursor:  devCursor,
		undoStack:     make(map[string][]wiz.PilotParams),
	}
}

func NewModel(client wiz.Client, initialIP string) Model {
	return NewModelWithConfig(client, initialIP, "")
}

type ScanFinishedMsg []string
type CommandFinishedMsg struct {
	Err    error
	IPs    []string
	Params wiz.PilotParams
}
type TelemetryReceivedMsg struct {
	IP    string
	Pilot *wiz.PilotParams
	Err   error
}
type TimerTickMsg time.Time
type TelemetryTickMsg time.Time
type AnimTickMsg time.Time

func tickTimerCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg(t)
	})
}

func tickTelemetryCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return TelemetryTickMsg(t)
	})
}

func tickAnimCmd() tea.Cmd {
	return tea.Tick(750*time.Millisecond, func(t time.Time) tea.Msg {
		return AnimTickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.scanNetworkCmd(), tickTimerCmd(), tickTelemetryCmd(), tickAnimCmd(), m.pollTelemetryCmd())
}

func (m Model) scanNetworkCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
		defer cancel()
		ips, _ := wiz.DiscoverSmartBulbs(ctx, 1000*time.Millisecond)
		return ScanFinishedMsg(ips)
	}
}

func (m Model) pollTelemetryCmd() tea.Cmd {
	activeDev, ok := m.Registry.GetActive()
	if !ok || activeDev == nil {
		return nil
	}
	ip := activeDev.IP
	client := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		pilot, err := client.GetPilot(ctx, ip)
		return TelemetryReceivedMsg{IP: ip, Pilot: pilot, Err: err}
	}
}

func (m Model) dispatchPilotCmdToDevices(targets []*wiz.Device, params wiz.PilotParams) tea.Cmd {
	if len(targets) == 0 {
		return nil
	}

	var ips []string
	for _, dev := range targets {
		ips = append(ips, dev.IP)

		st := dev.State
		dim := dev.Brightness
		r, g, b := dev.RGB[0], dev.RGB[1], dev.RGB[2]
		temp := dev.Temp
		sc := dev.SceneID
		sp := dev.Speed

		prev := wiz.PilotParams{
			State:   &st,
			Dimming: &dim,
			R:       &r, G: &g, B: &b,
			Temp:    &temp,
			SceneID: &sc,
			Speed:   &sp,
		}
		m.undoStack[dev.IP] = append(m.undoStack[dev.IP], prev)
	}

	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		errs := client.SendBatchCommand(ctx, ips, params)

		for _, err := range errs {
			if err != nil {
				return CommandFinishedMsg{Err: err, IPs: ips, Params: params}
			}
		}
		return CommandFinishedMsg{Err: nil, IPs: ips, Params: params}
	}
}

func (m Model) dispatchPilotCmd(params wiz.PilotParams) tea.Cmd {
	return m.dispatchPilotCmdToDevices(m.Registry.GetSelectedOrActive(), params)
}

func (m *Model) processTimerTick() []tea.Cmd {
	var cmds []tea.Cmd
	if m.sleepTimerSecs > 0 {
		m.sleepTimerSecs--
		if m.sleepTimerSecs == 0 {
			m.setStatusMessage("Sleep timer expired. Turning off lights.")
			cmds = append(cmds, m.dispatchPilotCmd(wiz.NewPowerParams(false)))
			cmds = append(cmds, SendOSCNotification("gowiz Sleep Timer", "Sleep timer expired. Turning off WiZ lights."))
		}
	}

	if fadeCmd := m.processFadeStep(); fadeCmd != nil {
		cmds = append(cmds, fadeCmd)
	}

	if m.statusMessage != "" && time.Since(m.statusTimer) > 4*time.Second {
		m.statusMessage = ""
	}

	cmds = append(cmds, tickTimerCmd())
	return cmds
}

func (m *Model) processFadeStep() tea.Cmd {
	if !m.fadeActive {
		return nil
	}

	m.fadeElapsedSecs++
	if m.fadeElapsedSecs >= m.fadeDurationSecs {
		m.fadeActive = false
		lbl := m.fadeLabel
		if lbl == "" {
			lbl = "Fade"
		}
		m.setStatusMessage(fmt.Sprintf("%s transition complete.", lbl))

		if m.fadeTurnOffOnFinish {
			return m.dispatchPilotCmd(wiz.NewPowerParams(false))
		}

		st := true
		params := wiz.PilotParams{State: &st}
		if m.fadeTargetDim > 0 {
			dim := wiz.Clamp(m.fadeTargetDim, 10, 100)
			params.Dimming = &dim
		}
		if m.fadeTargetTemp > 0 {
			temp := wiz.Clamp(m.fadeTargetTemp, 2200, 6500)
			params.Temp = &temp
		}
		return m.dispatchPilotCmd(params)
	}

	progress := float64(m.fadeElapsedSecs) / float64(m.fadeDurationSecs)
	curDim := m.fadeStartDim + int(float64(m.fadeTargetDim-m.fadeStartDim)*progress)
	curDim = wiz.Clamp(curDim, 10, 100)

	st := true
	params := wiz.PilotParams{
		State:   &st,
		Dimming: &curDim,
	}

	if m.fadeTargetTemp > 0 && m.fadeStartTemp > 0 {
		curTemp := m.fadeStartTemp + int(float64(m.fadeTargetTemp-m.fadeStartTemp)*progress)
		curTemp = wiz.Clamp(curTemp, 2200, 6500)
		params.Temp = &curTemp
	}

	return m.dispatchPilotCmd(params)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case AnimTickMsg:
		m.animFrame++
		cmds = append(cmds, tickAnimCmd())

	case TimerTickMsg:
		cmds = append(cmds, m.processTimerTick()...)

	case TelemetryTickMsg:
		cmds = append(cmds, m.pollTelemetryCmd(), tickTelemetryCmd())

	case TelemetryReceivedMsg:
		if dev, found := m.Registry.Get(msg.IP); found {
			if msg.Err == nil && msg.Pilot != nil {
				dev.UpdateFromPilot(*msg.Pilot)
				dev.Online = true
			} else {
				dev.Online = false
			}
		}

	case ScanFinishedMsg:
		foundCount := 0
		for _, ip := range msg {
			m.Registry.AddOrUpdate(wiz.NewDevice(ip))
			foundCount++
		}
		m.clampDeviceCursor()
		m.setStatusMessage(fmt.Sprintf("Network scan complete. Found %d device(s).", foundCount))
		cmds = append(cmds, SendOSCNotification("gowiz Network Scan", fmt.Sprintf("Found %d WiZ device(s) on network.", foundCount)))

	case CommandFinishedMsg:
		if msg.Err != nil {
			m.setStatusMessage(fmt.Sprintf("⚠️ Light unreachable: %v", msg.Err))
			for _, ip := range msg.IPs {
				if dev, found := m.Registry.Get(ip); found {
					dev.Online = false
				}
			}
		} else {
			m.setStatusMessage("✓ Command confirmed by physical light.")
			for _, ip := range msg.IPs {
				if dev, found := m.Registry.Get(ip); found {
					dev.UpdateFromPilot(msg.Params)
					dev.Online = true
				}
			}
		}

	case tea.KeyMsg:
		key := msg.String()

		if key == "esc" {
			if m.mode == ModeHelp || m.mode == ModeCommand || m.mode == ModeSearch {
				m.mode = ModeNormal
				m.commandBuffer = ""
				return m, nil
			}
			if m.mode == ModeVisual {
				m.mode = ModeNormal
				m.Registry.ClearSelections()
				return m, nil
			}
		}

		switch m.mode {
		case ModeCommand:
			return m.handleCommandKey(key)

		case ModeSearch:
			return m.handleSearchKey(key)

		case ModeHelp:
			if key == "q" || key == "?" || key == "esc" {
				m.mode = ModeNormal
			}
			return m, nil

		case ModeNormal, ModeVisual:
			return m.handleNormalOrVisualKey(key)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleNormalOrVisualKey(key string) (tea.Model, tea.Cmd) {
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		sceneID := int(key[0] - '0')
		scene := wiz.GetSceneByID(sceneID)
		m.setStatusMessage(fmt.Sprintf("Sending Scene: %s...", scene.Name))
		return m, m.dispatchPilotCmd(wiz.NewSceneParams(sceneID))
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.mode = ModeHelp
		return m, nil

	case "enter":
		switch m.activePanel {
		case PanelScenes:
			scenes := wiz.FilterScenes(m.searchQuery)
			if m.sceneCursor >= 0 && m.sceneCursor < len(scenes) {
				scene := scenes[m.sceneCursor]
				m.setStatusMessage(fmt.Sprintf("Sending Scene: %s...", scene.Name))
				return m, m.dispatchPilotCmd(wiz.NewSceneParams(scene.ID))
			}
		case PanelDevices:
			devices := m.Registry.List()
			if m.deviceCursor >= 0 && m.deviceCursor < len(devices) {
				dev := devices[m.deviceCursor]
				m.setActiveDevice(dev.IP)
				m.setStatusMessage(fmt.Sprintf("Active bulb set to %s", dev.IP))
			}
		case PanelControl:
			active, ok := m.Registry.GetActive()
			if ok {
				return m, m.dispatchPilotCmd(wiz.NewPowerParams(!active.State))
			}
		}
		return m, nil

	case ":":
		m.mode = ModeCommand
		m.commandBuffer = ""
		return m, nil

	case "/":
		m.mode = ModeSearch
		m.commandBuffer = ""
		return m, nil

	case "v":
		if m.mode == ModeVisual {
			m.mode = ModeNormal
		} else {
			m.mode = ModeVisual
		}
		return m, nil

	case "tab":
		m.activePanel = m.activePanel.Next()
		return m, nil

	case "shift+tab":
		m.activePanel = m.activePanel.Prev()
		return m, nil

	case "j", "down":
		return m.navigateDown()

	case "k", "up":
		return m.navigateUp()

	case "home":
		if m.activePanel == PanelDevices {
			devices := m.Registry.List()
			if len(devices) > 0 {
				m.deviceCursor = 0
				m.setActiveDevice(devices[0].IP)
			}
		} else if m.activePanel == PanelScenes {
			m.sceneCursor = 0
		}
		return m, nil

	case "G", "end":
		devices := m.Registry.List()
		if m.activePanel == PanelDevices && len(devices) > 0 {
			m.deviceCursor = len(devices) - 1
			m.setActiveDevice(devices[m.deviceCursor].IP)
		} else if m.activePanel == PanelScenes {
			scenes := wiz.FilterScenes(m.searchQuery)
			if len(scenes) > 0 {
				m.sceneCursor = len(scenes) - 1
			}
		}
		return m, nil

	case " ", "space":
		if m.mode == ModeVisual {
			devices := m.Registry.List()
			if m.deviceCursor >= 0 && m.deviceCursor < len(devices) {
				m.Registry.ToggleSelection(devices[m.deviceCursor].IP)
			}
			return m, nil
		}
		active, ok := m.Registry.GetActive()
		if ok {
			return m, m.dispatchPilotCmd(wiz.NewPowerParams(!active.State))
		}
		return m, m.dispatchPilotCmd(wiz.NewPowerParams(true))

	case "f", "x":
		return m, m.dispatchPilotCmd(wiz.NewPowerParams(false))

	case "l", "right", "+", "=":
		active, ok := m.Registry.GetActive()
		if ok {
			newDim := active.Brightness + 10
			return m, m.dispatchPilotCmd(wiz.NewDimmingParams(newDim))
		}

	case "h", "left", "-":
		active, ok := m.Registry.GetActive()
		if ok {
			newDim := active.Brightness - 10
			return m, m.dispatchPilotCmd(wiz.NewDimmingParams(newDim))
		}

	case "t":
		m.sleepTimerSecs = 15 * 60
		m.setStatusMessage("Sleep timer set: 15 minutes.")
		return m, nil

	case "[":
		active, ok := m.Registry.GetActive()
		if ok && active.SceneID > 0 {
			newSp := clamp(active.Speed-10, 20, 200)
			return m, m.dispatchPilotCmd(wiz.NewSceneParams(active.SceneID, newSp))
		}

	case "]":
		active, ok := m.Registry.GetActive()
		if ok && active.SceneID > 0 {
			newSp := clamp(active.Speed+10, 20, 200)
			return m, m.dispatchPilotCmd(wiz.NewSceneParams(active.SceneID, newSp))
		}

	case "a":
		if m.mode == ModeVisual {
			m.Registry.SelectAll()
		}
		return m, nil

	case "R":
		m.setStatusMessage("Rescanning network...")
		return m, m.scanNetworkCmd()

	case "u":
		active, ok := m.Registry.GetActive()
		if ok {
			stack := m.undoStack[active.IP]
			if len(stack) > 0 {
				lastState := stack[len(stack)-1]
				m.undoStack[active.IP] = stack[:len(stack)-1]
				return m, m.dispatchPilotCmd(lastState)
			}
			m.setStatusMessage("Nothing to undo.")
		}
	}

	return m, nil
}

func (m Model) navigateDown() (Model, tea.Cmd) {
	if m.activePanel == PanelDevices {
		devices := m.Registry.List()
		if len(devices) > 0 {
			m.deviceCursor = (m.deviceCursor + 1) % len(devices)
			m.setActiveDevice(devices[m.deviceCursor].IP)
		}
	} else if m.activePanel == PanelScenes {
		scenes := wiz.FilterScenes(m.searchQuery)
		if len(scenes) > 0 {
			m.sceneCursor = (m.sceneCursor + 1) % len(scenes)
		}
	} else if m.activePanel == PanelControl {
		active, ok := m.Registry.GetActive()
		if ok {
			newDim := active.Brightness - 10
			return m, m.dispatchPilotCmd(wiz.NewDimmingParams(newDim))
		}
	}
	return m, nil
}

func (m Model) navigateUp() (Model, tea.Cmd) {
	if m.activePanel == PanelDevices {
		devices := m.Registry.List()
		if len(devices) > 0 {
			m.deviceCursor--
			if m.deviceCursor < 0 {
				m.deviceCursor = len(devices) - 1
			}
			m.setActiveDevice(devices[m.deviceCursor].IP)
		}
	} else if m.activePanel == PanelScenes {
		scenes := wiz.FilterScenes(m.searchQuery)
		if len(scenes) > 0 {
			m.sceneCursor--
			if m.sceneCursor < 0 {
				m.sceneCursor = len(scenes) - 1
			}
		}
	} else if m.activePanel == PanelControl {
		active, ok := m.Registry.GetActive()
		if ok {
			newDim := active.Brightness + 10
			return m, m.dispatchPilotCmd(wiz.NewDimmingParams(newDim))
		}
	}
	return m, nil
}

func (m Model) handleCommandKey(key string) (Model, tea.Cmd) {
	switch key {
	case "enter":
		cmdStr := strings.TrimSpace(m.commandBuffer)
		m.mode = ModeNormal
		m.commandBuffer = ""

		if cmdStr == "" {
			return m, nil
		}

		activeDev, _ := m.Registry.GetActive()
		res := ExecuteCommand(cmdStr, activeDev)

		if res.Quit {
			return m, tea.Quit
		}
		if res.Help {
			m.mode = ModeHelp
			return m, nil
		}
		if res.Scan {
			m.setStatusMessage("Scanning network...")
			return m, m.scanNetworkCmd()
		}
		if res.SetSleepTimer > 0 {
			m.sleepTimerSecs = res.SetSleepTimer
		}
		if res.IsFadeCommand {
			startDim := 50
			startTemp := 2700
			if activeDev != nil {
				if activeDev.Brightness > 0 {
					startDim = activeDev.Brightness
				}
				if activeDev.Temp > 0 {
					startTemp = activeDev.Temp
				}
			}
			m.fadeActive = true
			m.fadeStartDim = startDim
			m.fadeTargetDim = res.SetFadeDimming
			m.fadeStartTemp = startTemp
			m.fadeTargetTemp = res.SetFadeColorTemp
			m.fadeDurationSecs = res.SetFadeDuration
			m.fadeElapsedSecs = 0
			m.fadeTurnOffOnFinish = res.FadeTurnOff
			m.fadeLabel = res.FadeLabel
			if m.fadeLabel == "" {
				m.fadeLabel = "🌆 Fade"
			}
		}
		if res.Undo {
			if activeDev != nil {
				stack := m.undoStack[activeDev.IP]
				if len(stack) > 0 {
					lastState := stack[len(stack)-1]
					m.undoStack[activeDev.IP] = stack[:len(stack)-1]
					m.setStatusMessage("Undid previous state change.")
					return m, m.dispatchPilotCmd(lastState)
				}
				m.setStatusMessage("Nothing to undo.")
			}
			return m, nil
		}

		if res.ConfigInfo {
			path := "~/.config/gowiz/config.json"
			aliasCount := 0
			if m.configManager != nil {
				path = m.configManager.FilePath()
				aliasCount = len(m.configManager.GetConfig().DeviceAliases)
			}
			m.setStatusMessage(fmt.Sprintf("Config: %s (%d saved alias(es))", path, aliasCount))
			return m, nil
		}

		if res.ShowRecent {
			if m.configManager != nil {
				recents := m.configManager.GetRecentIPs()
				if len(recents) == 0 {
					m.setStatusMessage("No recent IP targets saved.")
				} else {
					m.setStatusMessage(fmt.Sprintf("Recent targets (%d): %s", len(recents), strings.Join(recents, ", ")))
				}
			} else {
				m.setStatusMessage("No configuration manager attached.")
			}
			return m, nil
		}

		if res.ListPresets {
			var list []string
			for k := range BuiltinPresets {
				list = append(list, k+" [builtin]")
			}
			if m.configManager != nil {
				customs := m.configManager.GetPresets()
				for k := range customs {
					list = append(list, k+" [custom]")
				}
			}
			if len(list) == 0 {
				m.setStatusMessage("No presets defined.")
			} else {
				m.setStatusMessage(fmt.Sprintf("Presets (%d): %s", len(list), strings.Join(list, ", ")))
			}
			return m, nil
		}

		if res.SavePresetName != "" && activeDev != nil {
			preset := PresetFromDevice(activeDev)
			if m.configManager != nil {
				_ = m.configManager.SetPreset(res.SavePresetName, preset)
				m.setStatusMessage(fmt.Sprintf("Saved current state as preset '%s'", res.SavePresetName))
			} else {
				m.setStatusMessage("Config manager unavailable; preset not saved.")
			}
			return m, nil
		}

		if res.DeletePresetName != "" {
			if m.configManager != nil {
				_ = m.configManager.DeletePreset(res.DeletePresetName)
				m.setStatusMessage(fmt.Sprintf("Deleted preset '%s'", res.DeletePresetName))
			} else {
				m.setStatusMessage("Config manager unavailable; preset not deleted.")
			}
			return m, nil
		}

		if res.ApplyPresetName != "" {
			var params wiz.PilotParams
			found := false

			if m.configManager != nil {
				if cp, ok := m.configManager.GetPreset(res.ApplyPresetName); ok {
					params = PresetToPilotParams(cp)
					found = true
				}
			}
			if !found {
				if bp, ok := BuiltinPresets[res.ApplyPresetName]; ok {
					params = PresetToPilotParams(bp)
					found = true
				}
			}

			if !found {
				m.setStatusMessage(fmt.Sprintf("Preset not found: '%s'", res.ApplyPresetName))
				return m, nil
			}

			targets := m.Registry.GetSelectedOrActive()
			if res.TargetRoom != "" {
				targets = m.Registry.GetDevicesByRoom(res.TargetRoom)
				if len(targets) == 0 {
					m.setStatusMessage(fmt.Sprintf("No devices found in room '%s'", res.TargetRoom))
					return m, nil
				}
			}

			if res.StatusMsg != "" {
				m.setStatusMessage(res.StatusMsg)
			} else {
				m.setStatusMessage(fmt.Sprintf("Activated preset '%s'", res.ApplyPresetName))
			}
			return m, m.dispatchPilotCmdToDevices(targets, params)
		}

		if res.TargetIP != "" {
			m.Registry.AddOrUpdate(wiz.NewDevice(res.TargetIP))
			m.setActiveDevice(res.TargetIP)
			devices := m.Registry.List()
			for i, dev := range devices {
				if dev.IP == res.TargetIP {
					m.deviceCursor = i
					break
				}
			}
			m.setStatusMessage(fmt.Sprintf("Target bulb set to %s", res.TargetIP))
			return m, m.pollTelemetryCmd()
		}

		if res.NewDeviceName != "" && activeDev != nil {
			m.Registry.SetName(activeDev.IP, res.NewDeviceName)
			if m.configManager != nil {
				_ = m.configManager.SetAlias(activeDev.IP, res.NewDeviceName)
			}
		}

		if res.SetDeviceRoom != "" && activeDev != nil {
			if res.SetDeviceRoom == "CLEAR" {
				m.Registry.SetRoom(activeDev.IP, "")
				if m.configManager != nil {
					_ = m.configManager.SetRoom(activeDev.IP, "")
				}
			} else {
				m.Registry.SetRoom(activeDev.IP, res.SetDeviceRoom)
				if m.configManager != nil {
					_ = m.configManager.SetRoom(activeDev.IP, res.SetDeviceRoom)
				}
			}
		}

		if res.TargetRoom != "" {
			roomDevs := m.Registry.GetDevicesByRoom(res.TargetRoom)
			if len(roomDevs) == 0 {
				m.setStatusMessage(fmt.Sprintf("No devices found in room '%s'", res.TargetRoom))
				return m, nil
			}
			if res.PilotParams != nil {
				if res.StatusMsg != "" {
					m.setStatusMessage(res.StatusMsg)
				}
				return m, m.dispatchPilotCmdToDevices(roomDevs, *res.PilotParams)
			}
			m.setStatusMessage(fmt.Sprintf("Room '%s' has %d device(s)", res.TargetRoom, len(roomDevs)))
			return m, nil
		}

		if res.StatusMsg != "" {
			m.setStatusMessage(res.StatusMsg)
		}
		if res.PilotParams != nil {
			return m, m.dispatchPilotCmd(*res.PilotParams)
		}
		return m, nil

	case "backspace":
		if len(m.commandBuffer) > 0 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
		return m, nil

	case "ctrl+u":
		m.commandBuffer = ""
		return m, nil

	default:
		if len(key) == 1 {
			m.commandBuffer += key
		}
		return m, nil
	}
}

func (m Model) handleSearchKey(key string) (Model, tea.Cmd) {
	switch key {
	case "enter":
		m.mode = ModeNormal
		m.activePanel = PanelScenes
		m.clampSceneCursor()
		return m, nil

	case "esc":
		m.mode = ModeNormal
		m.clampSceneCursor()
		return m, nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		m.clampSceneCursor()
		return m, nil

	case "ctrl+u":
		m.searchQuery = ""
		m.clampSceneCursor()
		return m, nil

	default:
		if len(key) == 1 {
			m.searchQuery += key
		}
		m.clampSceneCursor()
		return m, nil
	}
}

func (m *Model) clampDeviceCursor() {
	devices := m.Registry.List()
	if len(devices) == 0 {
		m.deviceCursor = 0
	} else if m.deviceCursor >= len(devices) {
		m.deviceCursor = len(devices) - 1
	}
}

func (m *Model) clampSceneCursor() {
	scenes := wiz.FilterScenes(m.searchQuery)
	if len(scenes) == 0 {
		m.sceneCursor = 0
	} else if m.sceneCursor >= len(scenes) {
		m.sceneCursor = len(scenes) - 1
	}
}

func (m Model) View() string {
	if m.mode == ModeHelp {
		return views.RenderHelpOverlay(m.width, m.height)
	}

	activeDev, _ := m.Registry.GetActive()
	activeSceneID := 0
	if activeDev != nil {
		activeSceneID = activeDev.SceneID
	}

	selectedCount := 0
	for _, dev := range m.Registry.List() {
		if dev.Selected {
			selectedCount++
		}
	}

	statusBar := views.RenderStatusBar(
		m.mode.String(),
		m.activePanel.String(),
		activeDev,
		selectedCount,
		m.commandBuffer,
		m.statusMessage,
		m.width,
	)

	titleBar := styles.AppTitleStyle.Render("⚡ gowiz - WiZ Smart Light Controller")

	// Total available content height for body (m.height - titleBar line - statusBar line)
	contentHeight := max(m.height-2, 6)

	fadeRemainingSecs := 0
	if m.fadeActive {
		fadeRemainingSecs = max(m.fadeDurationSecs-m.fadeElapsedSecs, 0)
	}

	// Compact / Narrow Terminal Mode (width < 65 or height < 15)
	if m.width < 65 || m.height < 15 {
		var activeBody string
		switch m.activePanel {
		case PanelDevices:
			activeBody = views.RenderDeviceList(m.Registry, m.deviceCursor, true, m.width, contentHeight)
		case PanelControl:
			activeBody = views.RenderControlPanel(activeDev, true, m.sleepTimerSecs, fadeRemainingSecs, m.fadeLabel, m.width, contentHeight)
		case PanelScenes:
			activeBody = views.RenderScenePicker(m.searchQuery, activeSceneID, m.sceneCursor, true, m.animFrame, m.width, contentHeight)
		}
		return lipgloss.JoinVertical(lipgloss.Left, titleBar, activeBody, statusBar)
	}

	// Standard 2-Column Responsive Grid Layout
	sideWidth := clamp(int(float64(m.width)*0.28), 20, 30)
	mainWidth := max(m.width-sideWidth, 30)

	leftCol := views.RenderDeviceList(m.Registry, m.deviceCursor, m.activePanel == PanelDevices, sideWidth, contentHeight)

	// Render Control Panel first and measure its actual rendered line height
	initialControlHeight := clamp(int(float64(contentHeight)*0.40), 8, 12)
	topRight := views.RenderControlPanel(activeDev, m.activePanel == PanelControl, m.sleepTimerSecs, fadeRemainingSecs, m.fadeLabel, mainWidth, initialControlHeight)
	actualControlLines := lipgloss.Height(topRight)

	// Scene picker gets the exact remaining height so total rightCol height == contentHeight
	sceneHeight := max(contentHeight-actualControlLines, 4)
	bottomRight := views.RenderScenePicker(m.searchQuery, activeSceneID, m.sceneCursor, m.activePanel == PanelScenes, m.animFrame, mainWidth, sceneHeight)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRight, bottomRight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body, statusBar)
}

func (m *Model) setStatusMessage(msg string) {
	m.statusMessage = msg
	m.statusTimer = time.Now()
}

func (m *Model) setActiveDevice(ip string) {
	if m.Registry.SetActive(ip) {
		if m.configManager != nil {
			_ = m.configManager.AddRecentIP(ip)
		}
	}
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
