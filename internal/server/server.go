package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"wiz-tui/internal/circadian"
	"wiz-tui/internal/config"
	"wiz-tui/internal/tui"
	"wiz-tui/internal/wiz"
)

// Config defines the initialization options for the HTTP REST API server.
type Config struct {
	Port        int
	Host        string
	WizClient   wiz.Client
	ConfigMgr   *config.Manager
	DevRegistry *wiz.DeviceRegistry
	CommandReg  *tui.CommandRegistry
}

// Server implements a lightweight, zero-dependency HTTP REST API server for gowiz.
type Server struct {
	cfg        Config
	httpServer *http.Server
	startTime  time.Time
	mu         sync.RWMutex
}

// NewServer initializes a new Server with defaults.
func NewServer(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.WizClient == nil {
		cfg.WizClient = wiz.NewMockClient()
	}
	if cfg.DevRegistry == nil {
		cfg.DevRegistry = wiz.NewDeviceRegistry()
	}
	if cfg.CommandReg == nil {
		cfg.CommandReg = tui.NewCommandRegistry()
	}
	return &Server{
		cfg:       cfg,
		startTime: time.Now(),
	}
}

// ListenAddr returns the host:port server address.
func (s *Server) ListenAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

// Router configures and returns the HTTP request handler mux.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/devices", s.handleDevices)
	mux.HandleFunc("/api/v1/pilot", s.handlePilot)
	mux.HandleFunc("/api/v1/presets", s.handlePresets)
	mux.HandleFunc("/api/v1/circadian", s.handleCircadian)
	mux.HandleFunc("/api/v1/command", s.handleCommand)
	return mux
}

// Start launches the HTTP server and blocks until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	s.httpServer = &http.Server{
		Addr:    s.ListenAddr(),
		Handler: s.Router(),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"status": "error",
		"error":  msg,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cfgPath string
	recentCount := 0
	presetsCount := 0
	if s.cfg.ConfigMgr != nil {
		cfgPath = s.cfg.ConfigMgr.FilePath()
		c := s.cfg.ConfigMgr.GetConfig()
		recentCount = len(c.RecentIPs)
		presetsCount = len(c.Presets)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"service":       "gowiz-api",
		"version":       "1.0.0",
		"uptimeSeconds": time.Since(s.startTime).Seconds(),
		"deviceCount":   len(s.cfg.DevRegistry.List()),
		"recentCount":   recentCount,
		"presetsCount":  presetsCount,
		"configPath":    cfgPath,
	})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	targetIP := strings.TrimSpace(r.URL.Query().Get("ip"))
	if targetIP != "" {
		dev, found := s.cfg.DevRegistry.Get(targetIP)
		if !found {
			writeError(w, http.StatusNotFound, fmt.Sprintf("device with IP %s not found", targetIP))
			return
		}
		writeJSON(w, http.StatusOK, dev)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"devices": s.cfg.DevRegistry.List(),
	})
}

// PilotRequest defines the JSON structure for controlling lights via HTTP POST /api/v1/pilot.
type PilotRequest struct {
	IP      string `json:"ip,omitempty"`
	Room    string `json:"room,omitempty"`
	State   *bool  `json:"state,omitempty"`
	Dimming *int   `json:"dimming,omitempty"`
	Temp    *int   `json:"temp,omitempty"`
	R       *int   `json:"r,omitempty"`
	G       *int   `json:"g,omitempty"`
	B       *int   `json:"b,omitempty"`
	SceneID *int   `json:"sceneId,omitempty"`
	Speed   *int   `json:"speed,omitempty"`
}

func (s *Server) handlePilot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req PilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	params := wiz.PilotParams{
		State:   req.State,
		Dimming: req.Dimming,
		Temp:    req.Temp,
		R:       req.R,
		G:       req.G,
		B:       req.B,
		SceneID: req.SceneID,
		Speed:   req.Speed,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := r.Context()
	if req.Room != "" {
		targets := s.cfg.DevRegistry.GetDevicesByRoom(req.Room)
		if len(targets) == 0 {
			writeError(w, http.StatusNotFound, fmt.Sprintf("no devices found in room '%s'", req.Room))
			return
		}
		ips := make([]string, len(targets))
		for i, dev := range targets {
			ips[i] = dev.IP
		}
		errs := s.cfg.WizClient.SendBatchCommand(ctx, ips, params)
		failed := 0
		for _, err := range errs {
			if err != nil {
				failed++
			}
		}
		if failed > 0 {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to send pilot to %d/%d devices in room '%s'", failed, len(ips), req.Room))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"room":         req.Room,
			"devicesCount": len(ips),
		})
		return
	}

	targetIP := req.IP
	if targetIP == "" {
		if activeDev, found := s.cfg.DevRegistry.GetActive(); found {
			targetIP = activeDev.IP
		}
	}
	if targetIP == "" {
		writeError(w, http.StatusBadRequest, "no target IP specified or active device available")
		return
	}

	if err := s.cfg.WizClient.SendCommand(ctx, targetIP, params); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to send pilot to %s: %v", targetIP, err))
		return
	}

	if s.cfg.ConfigMgr != nil {
		_ = s.cfg.ConfigMgr.SetLastActiveIP(targetIP)
		_ = s.cfg.ConfigMgr.AddRecentIP(targetIP)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"targetIP": targetIP,
	})
}

func (s *Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if r.Method == http.MethodGet {
		builtin := make([]string, 0, len(tui.BuiltinPresets))
		for name := range tui.BuiltinPresets {
			builtin = append(builtin, name)
		}
		var custom []string
		if s.cfg.ConfigMgr != nil {
			presets := s.cfg.ConfigMgr.GetPresets()
			for name := range presets {
				if _, isBuiltin := tui.BuiltinPresets[name]; !isBuiltin {
					custom = append(custom, name)
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"builtin": builtin,
			"custom":  custom,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
			IP   string `json:"ip,omitempty"`
			Room string `json:"room,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			writeError(w, http.StatusBadRequest, "preset name required in JSON payload")
			return
		}

		var params wiz.PilotParams
		presetName := strings.ToLower(req.Name)
		if bp, found := tui.BuiltinPresets[presetName]; found {
			params = tui.PresetToPilotParams(bp)
		} else if s.cfg.ConfigMgr != nil {
			if cp, found := s.cfg.ConfigMgr.GetPreset(presetName); found {
				params = tui.PresetToPilotParams(cp)
			} else {
				writeError(w, http.StatusNotFound, fmt.Sprintf("preset '%s' not found", req.Name))
				return
			}
		} else {
			writeError(w, http.StatusNotFound, fmt.Sprintf("preset '%s' not found", req.Name))
			return
		}

		ctx := r.Context()
		if req.Room != "" {
			targets := s.cfg.DevRegistry.GetDevicesByRoom(req.Room)
			if len(targets) == 0 {
				writeError(w, http.StatusNotFound, fmt.Sprintf("no devices found in room '%s'", req.Room))
				return
			}
			ips := make([]string, len(targets))
			for i, dev := range targets {
				ips[i] = dev.IP
			}
			_ = s.cfg.WizClient.SendBatchCommand(ctx, ips, params)
			writeJSON(w, http.StatusOK, map[string]any{
				"status":       "ok",
				"preset":       req.Name,
				"room":         req.Room,
				"devicesCount": len(ips),
			})
			return
		}

		targetIP := req.IP
		if targetIP == "" {
			if activeDev, found := s.cfg.DevRegistry.GetActive(); found {
				targetIP = activeDev.IP
			}
		}
		if targetIP == "" {
			writeError(w, http.StatusBadRequest, "target IP or active device required")
			return
		}

		if err := s.cfg.WizClient.SendCommand(ctx, targetIP, params); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to apply preset to %s: %v", targetIP, err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"preset":   req.Name,
			"targetIP": targetIP,
		})
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) handleCircadian(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	targetTime := time.Now()
	timeArg := r.URL.Query().Get("time")
	if timeArg != "" {
		t, err := circadian.ParseTimeArg(timeArg)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid time format: %v", err))
			return
		}
		targetTime = t
	}

	room := r.URL.Query().Get("room")
	var globalPhases []circadian.SchedulePhase
	var roomPhases map[string][]circadian.SchedulePhase
	if s.cfg.ConfigMgr != nil {
		c := s.cfg.ConfigMgr.GetConfig()
		globalPhases = c.CircadianPhases
		roomPhases = c.RoomCircadianPhases
	}

	phases := globalPhases
	if room != "" && roomPhases != nil {
		if rp, found := roomPhases[strings.ToLower(strings.TrimSpace(room))]; found && len(rp) > 0 {
			phases = rp
		}
	}

	info := circadian.CalculateWithPhases(targetTime, phases)

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"room":   room,
		"time":   targetTime.Format("15:04"),
		"phase":  info.Phase,
		"info":   info.Status,
		"params": info.Params,
	})
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Command string `json:"command"`
		IP      string `json:"ip,omitempty"`
		Room    string `json:"room,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Command == "" {
		writeError(w, http.StatusBadRequest, "command string required in JSON payload")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var activeDev *wiz.Device
	if req.IP != "" {
		activeDev = wiz.NewDevice(req.IP)
	} else if dev, found := s.cfg.DevRegistry.GetActive(); found {
		activeDev = dev
	}

	var globalPhases []circadian.SchedulePhase
	var roomPhases map[string][]circadian.SchedulePhase
	if s.cfg.ConfigMgr != nil {
		c := s.cfg.ConfigMgr.GetConfig()
		globalPhases = c.CircadianPhases
		roomPhases = c.RoomCircadianPhases
	}

	cmdStr := strings.TrimPrefix(strings.TrimSpace(req.Command), ":")
	res := tui.ExecuteCommandWithRoomPhases(cmdStr, activeDev, globalPhases, roomPhases)

	if req.Room != "" {
		res.TargetRoom = req.Room
	}

	ctx := r.Context()
	if res.TargetRoom != "" && res.PilotParams != nil {
		targets := s.cfg.DevRegistry.GetDevicesByRoom(res.TargetRoom)
		if len(targets) > 0 {
			ips := make([]string, len(targets))
			for i, d := range targets {
				ips[i] = d.IP
			}
			_ = s.cfg.WizClient.SendBatchCommand(ctx, ips, *res.PilotParams)
		}
	} else if res.PilotParams != nil && activeDev != nil {
		_ = s.cfg.WizClient.SendCommand(ctx, activeDev.IP, *res.PilotParams)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"command":   cmdStr,
		"statusMsg": res.StatusMsg,
		"result":    res,
	})
}
