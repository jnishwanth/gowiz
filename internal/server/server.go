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
	APIKey      string
	WebhookURL  string
	WizClient   wiz.Client
	ConfigMgr   *config.Manager
	DevRegistry *wiz.DeviceRegistry
	CommandReg  *tui.CommandRegistry
}

// Server implements a lightweight, zero-dependency HTTP REST API server for gowiz.
type Server struct {
	cfg         Config
	httpServer  *http.Server
	broadcaster *Broadcaster
	startTime   time.Time
	mu          sync.RWMutex
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
		cfg:         cfg,
		broadcaster: NewBroadcaster(cfg.WebhookURL),
		startTime:   time.Now(),
	}
}

// Broadcaster returns the server's Broadcaster instance.
func (s *Server) Broadcaster() *Broadcaster {
	return s.broadcaster
}

// ListenAddr returns the host:port server address.
func (s *Server) ListenAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func (s *Server) wrap(handler http.Handler, requireAuth bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, X-API-Key, Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if requireAuth && s.cfg.APIKey != "" {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					key = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}

			if key != s.cfg.APIKey {
				writeError(w, http.StatusUnauthorized, "unauthorized: invalid or missing API key")
				return
			}
		}

		handler.ServeHTTP(w, r)
	})
}

func (s *Server) wrapFunc(handler http.HandlerFunc, requireAuth bool) http.Handler {
	return s.wrap(http.HandlerFunc(handler), requireAuth)
}

// Router configures and returns the HTTP request handler mux.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", s.wrapFunc(s.handleHealth, false))
	mux.Handle("/api/v1/health", s.wrapFunc(s.handleHealth, false))
	mux.Handle("/api/v1/devices", s.wrapFunc(s.handleDevices, true))
	mux.Handle("/api/v1/rooms", s.wrapFunc(s.handleRooms, true))
	mux.Handle("/api/v1/openapi.json", s.wrapFunc(s.handleOpenAPI, false))
	mux.Handle("/docs", s.wrapFunc(s.handleDocs, false))
	mux.Handle("/api/v1/docs", s.wrapFunc(s.handleDocs, false))
	mux.Handle("/api/v1/pilot", s.wrapFunc(s.handlePilot, true))
	mux.Handle("/api/v1/presets", s.wrapFunc(s.handlePresets, true))
	mux.Handle("/api/v1/circadian", s.wrapFunc(s.handleCircadian, true))
	mux.Handle("/api/v1/command", s.wrapFunc(s.handleCommand, true))
	mux.Handle("/events", s.wrap(s.broadcaster, true))
	mux.Handle("/api/v1/events", s.wrap(s.broadcaster, true))
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

	subscribers, webhookSent, webhookFailed := s.broadcaster.Stats()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":              "ok",
		"service":             "gowiz-api",
		"version":             "1.0.0",
		"uptimeSeconds":       time.Since(s.startTime).Seconds(),
		"deviceCount":         len(s.cfg.DevRegistry.List()),
		"recentCount":         recentCount,
		"presetsCount":        presetsCount,
		"configPath":          cfgPath,
		"apiKeyProtected":     s.cfg.APIKey != "",
		"sseSubscribers":      subscribers,
		"webhookURL":          s.broadcaster.WebhookURL(),
		"webhookEventsSent":   webhookSent,
		"webhookEventsFailed": webhookFailed,
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

		s.broadcaster.Publish(Event{
			Type:    EventDeviceUpdated,
			Room:    req.Room,
			Status:  "ok",
			Payload: params,
		})

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

	s.broadcaster.Publish(Event{
		Type:    EventDeviceUpdated,
		IP:      targetIP,
		Status:  "ok",
		Payload: params,
	})

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

			s.broadcaster.Publish(Event{
				Type:    EventPresetApplied,
				Room:    req.Room,
				Command: req.Name,
				Status:  "ok",
				Payload: params,
			})

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

		s.broadcaster.Publish(Event{
			Type:    EventPresetApplied,
			IP:      targetIP,
			Command: req.Name,
			Status:  "ok",
			Payload: params,
		})

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

	s.broadcaster.Publish(Event{
		Type:    EventCommandExecuted,
		IP:      req.IP,
		Room:    res.TargetRoom,
		Command: cmdStr,
		Status:  "ok",
		Payload: res,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"command":   cmdStr,
		"statusMsg": res.StatusMsg,
		"result":    res,
	})
}

func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	roomName := strings.TrimSpace(r.URL.Query().Get("name"))
	if roomName != "" {
		devices := s.cfg.DevRegistry.GetDevicesByRoom(roomName)
		if len(devices) == 0 {
			writeError(w, http.StatusNotFound, fmt.Sprintf("room '%s' not found or contains no devices", roomName))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"room":    roomName,
			"devices": devices,
		})
		return
	}

	roomsMap := make(map[string][]*wiz.Device)
	for _, dev := range s.cfg.DevRegistry.List() {
		rName := dev.Room
		if rName == "" {
			rName = "unassigned"
		}
		roomsMap[rName] = append(roomsMap[rName], dev)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"rooms":  roomsMap,
	})
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "gowiz REST API",
			"description": "Zero-dependency HTTP REST API server for gowiz WiZ smart light control",
			"version":     "1.0.0",
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": map[string]any{
					"summary":   "Service health and metrics",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/devices": map[string]any{
				"get": map[string]any{
					"summary": "Get all devices or query device by IP",
					"parameters": []map[string]any{
						{"name": "ip", "in": "query", "schema": map[string]any{"type": "string"}, "description": "Optional bulb IP address"},
					},
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/rooms": map[string]any{
				"get": map[string]any{
					"summary": "Get all rooms and assigned devices",
					"parameters": []map[string]any{
						{"name": "name", "in": "query", "schema": map[string]any{"type": "string"}, "description": "Optional room name filter"},
					},
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/pilot": map[string]any{
				"post": map[string]any{
					"summary": "Send Pilot control parameters to device or room",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"ip":      map[string]any{"type": "string"},
										"room":    map[string]any{"type": "string"},
										"state":   map[string]any{"type": "boolean"},
										"dimming": map[string]any{"type": "integer"},
										"temp":    map[string]any{"type": "integer"},
										"r":       map[string]any{"type": "integer"},
										"g":       map[string]any{"type": "integer"},
										"b":       map[string]any{"type": "integer"},
										"sceneId": map[string]any{"type": "integer"},
										"speed":   map[string]any{"type": "integer"},
									},
								},
							},
						},
					},
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/presets": map[string]any{
				"get": map[string]any{
					"summary":   "List available lighting presets",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
				"post": map[string]any{
					"summary":   "Apply lighting preset to IP or room",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/circadian": map[string]any{
				"get": map[string]any{
					"summary":   "Calculate circadian rhythm settings for a time and optional room",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/command": map[string]any{
				"post": map[string]any{
					"summary":   "Execute arbitrary gowiz command string",
					"responses": map[string]any{"200": map[string]any{"description": "OK"}},
				},
			},
			"/api/v1/events": map[string]any{
				"get": map[string]any{
					"summary":   "Server-Sent Events (SSE) live streaming endpoint",
					"responses": map[string]any{"200": map[string]any{"description": "Event Stream"}},
				},
			},
		},
	}
	writeJSON(w, http.StatusOK, spec)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>gowiz API Documentation</title>
<style>
body { font-family: system-ui, -apple-system, sans-serif; background: #0f172a; color: #f8fafc; margin: 0; padding: 2rem; }
h1 { color: #38bdf8; margin-bottom: 0.5rem; }
p { color: #94a3b8; }
.card { background: #1e293b; border-radius: 8px; padding: 1.5rem; margin-bottom: 1.5rem; border: 1px solid #334155; }
.method { font-weight: bold; padding: 0.25rem 0.5rem; border-radius: 4px; display: inline-block; margin-right: 0.5rem; font-size: 0.85rem; }
.get { background: #0284c7; color: white; }
.post { background: #16a34a; color: white; }
.endpoint { font-family: monospace; font-size: 1.1rem; color: #e2e8f0; }
pre { background: #0f172a; padding: 1rem; border-radius: 6px; overflow-x: auto; color: #38bdf8; font-size: 0.9rem; }
a { color: #38bdf8; text-decoration: none; }
a:hover { text-decoration: underline; }
</style>
</head>
<body>
<h1>gowiz HTTP REST API</h1>
<p>Interactive endpoint documentation and live integration reference. <a href="/api/v1/openapi.json" target="_blank">OpenAPI 3.0 JSON Spec</a></p>

<div class="card">
  <span class="method get">GET</span><span class="endpoint">/health</span>
  <p>Returns server health status, device counts, uptime, and webhook metrics.</p>
</div>

<div class="card">
  <span class="method get">GET</span><span class="endpoint">/api/v1/devices</span>
  <p>List all registered WiZ light devices or query specific device via <code>?ip=&lt;address&gt;</code>.</p>
</div>

<div class="card">
  <span class="method get">GET</span><span class="endpoint">/api/v1/rooms</span>
  <p>List all room groupings and devices, or filter by room via <code>?name=&lt;room&gt;</code>.</p>
</div>

<div class="card">
  <span class="method post">POST</span><span class="endpoint">/api/v1/pilot</span>
  <p>Send pilot state controls to a bulb IP or room target.</p>
  <pre>{"ip": "192.168.1.50", "state": true, "dimming": 80, "temp": 3000}</pre>
</div>

<div class="card">
  <span class="method get">GET</span><span class="method post">POST</span><span class="endpoint">/api/v1/presets</span>
  <p>GET lists available lighting presets. POST applies preset to target IP or room.</p>
  <pre>{"name": "evening", "room": "Living Room"}</pre>
</div>

<div class="card">
  <span class="method get">GET</span><span class="endpoint">/api/v1/circadian</span>
  <p>Calculate circadian rhythm phase for a target time (<code>?time=14:30</code>) and optional room.</p>
</div>

<div class="card">
  <span class="method post">POST</span><span class="endpoint">/api/v1/command</span>
  <p>Execute arbitrary gowiz command string (e.g., "sunset", "warm", "group Bedroom off").</p>
  <pre>{"command": "ocean", "room": "Living Room"}</pre>
</div>

<div class="card">
  <span class="method get">GET</span><span class="endpoint">/api/v1/events</span>
  <p>Server-Sent Events (SSE) live event streaming channel for real-time bulb updates.</p>
</div>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}
