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
	client        wiz.Client
	Registry      *wiz.DeviceRegistry
	mode          Mode
	activePanel   Panel
	commandBuffer string
	searchQuery   string
	statusMessage string
	statusTimer   time.Time
	width         int
	height        int
	sceneCursor   int // Focus index in scene list
	deviceCursor  int // Focus index in device list
	undoStack     map[string][]wiz.PilotParams
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
	Err error
	IP  string
}
type ClearStatusMsg struct{}

func (m Model) Init() tea.Cmd {
	return m.scanNetworkCmd()
}

func (m Model) scanNetworkCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
		defer cancel()
		ips, _ := wiz.DiscoverSmartBulbs(ctx, 1000*time.Millisecond)
		return ScanFinishedMsg(ips)
	}
}

func (m Model) dispatchPilotCmd(params wiz.PilotParams) tea.Cmd {
	targets := m.Registry.GetSelectedOrActive()
	if len(targets) == 0 {
		return nil
	}

	// Save undo state
	for _, dev := range targets {
		prev := wiz.PilotParams{
			State:   &dev.State,
			Dimming: &dev.Brightness,
			R:       &dev.RGB[0], G: &dev.RGB[1], B: &dev.RGB[2],
			Temp:    &dev.Temp,
			SceneID: &dev.SceneID,
			Speed:   &dev.Speed,
		}
		m.undoStack[dev.IP] = append(m.undoStack[dev.IP], prev)

		// Optimistically update local model
		dev.UpdateFromPilot(params)
	}

	client := m.client
	var ips []string
	for _, dev := range targets {
		ips = append(ips, dev.IP)
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		errs := client.SendBatchCommand(ctx, ips, params)
		for _, err := range errs {
			if err != nil {
				return CommandFinishedMsg{Err: err}
			}
		}
		return CommandFinishedMsg{Err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ScanFinishedMsg:
		foundCount := 0
		for _, ip := range msg {
			m.Registry.AddOrUpdate(wiz.NewDevice(ip))
			foundCount++
		}
		m.statusMessage = fmt.Sprintf("Network scan complete. Found %d device(s).", foundCount)

	case CommandFinishedMsg:
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Command failed: %v", msg.Err)
		} else {
			m.statusMessage = "Command executed successfully."
		}

	case ClearStatusMsg:
		m.statusMessage = ""

	case tea.KeyMsg:
		key := msg.String()

		// Global escape to Normal Mode
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

		// Handle key input by active Mode
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
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.mode = ModeHelp
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

	case "G":
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

	case "o", " ":
		// Enter on scene picker activates highlighted scene
		if m.activePanel == PanelScenes {
			scenes := wiz.FilterScenes(m.searchQuery)
			if m.sceneCursor >= 0 && m.sceneCursor < len(scenes) {
				sc := scenes[m.sceneCursor]
				return m, m.dispatchPilotCmd(wiz.NewSceneParams(sc.ID))
			}
		}
		if key == " " && m.mode == ModeVisual {
			active, ok := m.Registry.GetActive()
			if ok {
				m.Registry.ToggleSelection(active.IP)
			}
			return m, nil
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

	case "r":
		return m, m.dispatchPilotCmd(wiz.NewRGBParams(255, 0, 0))
	case "g":
		return m, m.dispatchPilotCmd(wiz.NewRGBParams(0, 255, 0))
	case "b":
		return m, m.dispatchPilotCmd(wiz.NewRGBParams(0, 0, 255))
	case "w":
		return m, m.dispatchPilotCmd(wiz.NewTempParams(2700))
	case "c":
		return m, m.dispatchPilotCmd(wiz.NewTempParams(2200))
	case "s":
		// Sleep mode preset
		return m, m.dispatchPilotCmd(wiz.NewRGBParams(255, 75, 0))
	case "p":
		return m, m.dispatchPilotCmd(wiz.NewRGBParams(147, 51, 234))

	case "a":
		if m.mode == ModeVisual {
			m.Registry.SelectAll()
		}
		return m, nil

	case "R":
		// Rescan network
		m.statusMessage = "Rescanning network..."
		return m, m.scanNetworkCmd()

	case "u":
		// Undo last state for active device
		active, ok := m.Registry.GetActive()
		if ok {
			stack := m.undoStack[active.IP]
			if len(stack) > 0 {
				lastState := stack[len(stack)-1]
				m.undoStack[active.IP] = stack[:len(stack)-1]
				return m, m.dispatchPilotCmd(lastState)
			}
			m.statusMessage = "Nothing to undo."
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
	}
	return m, nil
}

func (m Model) handleCommandKey(key string) (Model, tea.Cmd) {
	switch key {
	case "enter":
		cmdStr := strings.TrimSpace(m.commandBuffer)
		m.mode = ModeNormal
		m.commandBuffer = ""
		return m.executeVimCommand(cmdStr)

	case "backspace":
		if len(m.commandBuffer) > 0 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
	default:
		if len(key) == 1 {
			m.commandBuffer += key
		}
	}
	return m, nil
}

func (m Model) handleSearchKey(key string) (Model, tea.Cmd) {
	switch key {
	case "enter":
		m.searchQuery = m.commandBuffer
		m.mode = ModeNormal
		m.activePanel = PanelScenes
		m.sceneCursor = 0
		return m, nil

	case "backspace":
		if len(m.commandBuffer) > 0 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		}
		m.searchQuery = m.commandBuffer
	default:
		if len(key) == 1 {
			m.commandBuffer += key
			m.searchQuery = m.commandBuffer
		}
	}
	return m, nil
}

func (m Model) executeVimCommand(cmdStr string) (Model, tea.Cmd) {
	if cmdStr == "" {
		return m, nil
	}

	parts := strings.Fields(cmdStr)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "q", "quit":
		return m, tea.Quit

	case "scan":
		m.statusMessage = "Scanning subnet broadcasts..."
		return m, m.scanNetworkCmd()

	case "connect":
		if len(parts) > 1 {
			ip := parts[1]
			m.Registry.AddOrUpdate(wiz.NewDevice(ip))
			m.Registry.SetActive(ip)
			m.statusMessage = fmt.Sprintf("Connected to %s", ip)
		}

	case "scene":
		if len(parts) > 1 {
			target := strings.Join(parts[1:], " ")
			if id, err := strconv.Atoi(target); err == nil {
				return m, m.dispatchPilotCmd(wiz.NewSceneParams(id))
			}
			if scene, found := wiz.GetSceneByName(target); found {
				return m, m.dispatchPilotCmd(wiz.NewSceneParams(scene.ID))
			}
			m.statusMessage = fmt.Sprintf("Unknown scene: %s", target)
		}

	case "temp", "cct":
		if len(parts) > 1 {
			if kelvin, err := strconv.Atoi(parts[1]); err == nil {
				return m, m.dispatchPilotCmd(wiz.NewTempParams(kelvin))
			}
		}

	case "dim", "brightness":
		if len(parts) > 1 {
			if level, err := strconv.Atoi(parts[1]); err == nil {
				return m, m.dispatchPilotCmd(wiz.NewDimmingParams(level))
			}
		}

	case "rgb":
		if len(parts) >= 4 {
			r, _ := strconv.Atoi(parts[1])
			g, _ := strconv.Atoi(parts[2])
			b, _ := strconv.Atoi(parts[3])
			return m, m.dispatchPilotCmd(wiz.NewRGBParams(r, g, b))
		}

	default:
		m.statusMessage = fmt.Sprintf("Unknown command: :%s", cmdStr)
	}

	return m, nil
}

func (m Model) View() string {
	if m.mode == ModeHelp {
		return views.RenderHelpOverlay(m.width, m.height)
	}

	// Layout grid
	sideWidth := 30
	mainWidth := max(m.width-sideWidth-6, 40)
	contentHeight := max(m.height-5, 15)

	leftCol := views.RenderDeviceList(m.Registry, m.activePanel == PanelDevices, sideWidth, contentHeight)

	controlHeight := contentHeight / 2
	sceneHeight := contentHeight - controlHeight - 2

	activeDev, _ := m.Registry.GetActive()
	topRight := views.RenderControlPanel(activeDev, m.activePanel == PanelControl, mainWidth, controlHeight)

	activeSceneID := 0
	if activeDev != nil {
		activeSceneID = activeDev.SceneID
	}
	bottomRight := views.RenderScenePicker(m.searchQuery, activeSceneID, m.activePanel == PanelScenes, mainWidth, sceneHeight)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, topRight, bottomRight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	selectedCount := 0
	for _, dev := range m.Registry.List() {
		if dev.Selected {
			selectedCount++
		}
	}

	statusBar := views.RenderStatusBar(
		m.mode.String(),
		activeDev,
		selectedCount,
		m.commandBuffer,
		m.statusMessage,
		m.width,
	)

	titleBar := styles.AppTitleStyle.Render("⚡ gowiz - WiZ Smart Light Controller")

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body, statusBar)
}
