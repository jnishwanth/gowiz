package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"wiz-tui/internal/tui/styles"
	"wiz-tui/internal/tui/views"
	"wiz-tui/internal/wiz"
)

type Model struct {
	client         wiz.Client
	Registry       *wiz.DeviceRegistry
	mode           Mode
	activePanel    Panel
	commandBuffer  string
	searchQuery    string
	statusMessage  string
	statusTimer    time.Time
	width          int
	height         int
	sceneCursor    int
	deviceCursor   int
	sleepTimerSecs int
	animFrame      int
	undoStack      map[string][]wiz.PilotParams
}

func NewModel(client wiz.Client, initialIP string) Model {
	reg := wiz.NewDeviceRegistry()
	if initialIP != "" && initialIP != wiz.FallbackIP {
		reg.AddOrUpdate(wiz.NewDevice(initialIP))
		reg.SetActive(initialIP)
	}

	return Model{
		client:       client,
		Registry:     reg,
		mode:         ModeNormal,
		activePanel:  PanelDevices,
		width:        80,
		height:       24,
		sceneCursor:  0,
		deviceCursor: 0,
		undoStack:    make(map[string][]wiz.PilotParams),
	}
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

func (m Model) dispatchPilotCmd(params wiz.PilotParams) tea.Cmd {
	targets := m.Registry.GetSelectedOrActive()
	if len(targets) == 0 {
		return nil
	}

	var ips []string
	for _, dev := range targets {
		ips = append(ips, dev.IP)

		prev := wiz.PilotParams{
			State:   &dev.State,
			Dimming: &dev.Brightness,
			R:       &dev.RGB[0], G: &dev.RGB[1], B: &dev.RGB[2],
			Temp:    &dev.Temp,
			SceneID: &dev.SceneID,
			Speed:   &dev.Speed,
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
		if m.sleepTimerSecs > 0 {
			m.sleepTimerSecs--
			if m.sleepTimerSecs == 0 {
				m.setStatusMessage("Sleep timer expired. Turning off lights.")
				cmds = append(cmds, m.dispatchPilotCmd(wiz.NewPowerParams(false)))
				cmds = append(cmds, SendOSCNotification("gowiz Sleep Timer", "Sleep timer expired. Turning off WiZ lights."))
			}
		}
		if m.statusMessage != "" && time.Since(m.statusTimer) > 4*time.Second {
			m.statusMessage = ""
		}
		cmds = append(cmds, tickTimerCmd())

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
				m.Registry.SetActive(dev.IP)
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
			m.deviceCursor = 0
		} else if m.activePanel == PanelScenes {
			m.sceneCursor = 0
		}
		return m, nil

	case "G", "end":
		devices := m.Registry.List()
		if m.activePanel == PanelDevices && len(devices) > 0 {
			m.deviceCursor = len(devices) - 1
			m.Registry.SetActive(devices[m.deviceCursor].IP)
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
			m.Registry.SetActive(devices[m.deviceCursor].IP)
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
			m.Registry.SetActive(devices[m.deviceCursor].IP)
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

		parts := strings.Fields(cmdStr)
		verb := strings.ToLower(parts[0])

		switch verb {
		case "q", "quit", "exit":
			return m, tea.Quit

		case "scan":
			m.setStatusMessage("Scanning network...")
			return m, m.scanNetworkCmd()

		case "scene":
			if len(parts) > 1 {
				arg := strings.Join(parts[1:], " ")
				if id, err := strconv.Atoi(arg); err == nil {
					scene := wiz.GetSceneByID(id)
					m.setStatusMessage(fmt.Sprintf("Scene set: %s", scene.Name))
					return m, m.dispatchPilotCmd(wiz.NewSceneParams(id))
				} else if scene, found := wiz.GetSceneByName(arg); found {
					m.setStatusMessage(fmt.Sprintf("Scene set: %s", scene.Name))
					return m, m.dispatchPilotCmd(wiz.NewSceneParams(scene.ID))
				} else {
					m.setStatusMessage(fmt.Sprintf("Unknown scene: %s", arg))
				}
			}

		case "dim":
			if len(parts) > 1 {
				if dim, err := strconv.Atoi(parts[1]); err == nil {
					m.setStatusMessage(fmt.Sprintf("Brightness set: %d%%", dim))
					return m, m.dispatchPilotCmd(wiz.NewDimmingParams(dim))
				}
			}

		case "temp":
			if len(parts) > 1 {
				if temp, err := strconv.Atoi(parts[1]); err == nil {
					m.setStatusMessage(fmt.Sprintf("Color temp set: %dK", temp))
					return m, m.dispatchPilotCmd(wiz.NewTempParams(temp))
				}
			}

		case "rgb":
			if len(parts) >= 4 {
				r, _ := strconv.Atoi(parts[1])
				g, _ := strconv.Atoi(parts[2])
				b, _ := strconv.Atoi(parts[3])
				m.setStatusMessage(fmt.Sprintf("RGB color set: R:%d G:%d B:%d", r, g, b))
				return m, m.dispatchPilotCmd(wiz.NewRGBParams(r, g, b))
			}

		case "timer":
			if len(parts) > 1 {
				if mins, err := strconv.Atoi(parts[1]); err == nil {
					m.sleepTimerSecs = mins * 60
					m.setStatusMessage(fmt.Sprintf("Sleep timer set: %d minutes.", mins))
				}
			}

		default:
			m.setStatusMessage(fmt.Sprintf("Unknown command: :%s", cmdStr))
		}

		return m, nil

	case "backspace":
		if len(m.commandBuffer) > 0 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
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

	default:
		if len(key) == 1 {
			m.searchQuery += key
		}
		m.clampSceneCursor()
		return m, nil
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

	// Compact / Narrow Terminal Mode (width < 65 or height < 15)
	if m.width < 65 || m.height < 15 {
		var activeBody string
		switch m.activePanel {
		case PanelDevices:
			activeBody = views.RenderDeviceList(m.Registry, m.deviceCursor, true, m.width, contentHeight)
		case PanelControl:
			activeBody = views.RenderControlPanel(activeDev, true, m.sleepTimerSecs, m.width, contentHeight)
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
	topRight := views.RenderControlPanel(activeDev, m.activePanel == PanelControl, m.sleepTimerSecs, mainWidth, initialControlHeight)
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

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
