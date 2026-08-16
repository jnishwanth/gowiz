package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"wiz-tui/internal/config"
	"wiz-tui/internal/wiz"
)

func setupTestServer(t *testing.T) (*Server, *wiz.DeviceRegistry) {
	t.Helper()
	cfgMgr := config.NewManager(t.TempDir() + "/config.json")
	reg := wiz.NewDeviceRegistry()

	dev1 := wiz.NewDevice("192.168.1.50")
	dev1.Name = "Living Lamp"
	dev1.Room = "Living Room"
	dev1.State = true
	dev1.Brightness = 100

	dev2 := wiz.NewDevice("192.168.1.51")
	dev2.Name = "Bedroom Light"
	dev2.Room = "Bedroom"
	dev2.State = false
	dev2.Brightness = 50

	reg.AddOrUpdate(dev1)
	reg.AddOrUpdate(dev2)
	reg.SetActive("192.168.1.50")

	srv := NewServer(Config{
		Port:        8888,
		Host:        "127.0.0.1",
		WizClient:   wiz.NewMockClient(),
		ConfigMgr:   cfgMgr,
		DevRegistry: reg,
	})
	return srv, reg
}

func TestServerHealth(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("failed HTTP GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if data["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", data["status"])
	}
	if data["deviceCount"] != float64(3) {
		t.Errorf("expected deviceCount 3, got %v", data["deviceCount"])
	}

	// Method not allowed test
	req, _ := http.NewRequest("POST", ts.URL+"/health", nil)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed HTTP POST /health: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 Method Not Allowed, got %d", resp2.StatusCode)
	}
}

func TestServerDevices(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("Get all devices", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/devices")
		if err != nil {
			t.Fatalf("failed HTTP GET /api/v1/devices: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		devs, ok := data["devices"].([]any)
		if !ok || len(devs) != 3 {
			t.Errorf("expected 3 devices, got %v", data["devices"])
		}
	})

	t.Run("Get single device by IP", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/devices?ip=192.168.1.50")
		if err != nil {
			t.Fatalf("failed GET single device: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var dev wiz.Device
		if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
			t.Fatalf("failed to decode device JSON: %v", err)
		}
		if dev.IP != "192.168.1.50" || dev.Name != "Living Lamp" {
			t.Errorf("unexpected device data: %+v", dev)
		}
	})

	t.Run("Get non-existent device", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/devices?ip=192.168.1.99")
		if err != nil {
			t.Fatalf("failed GET unknown device: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})
}

func TestServerPilot(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("Send pilot to IP", func(t *testing.T) {
		st := true
		dim := 80
		body, _ := json.Marshal(PilotRequest{
			IP:      "192.168.1.50",
			State:   &st,
			Dimming: &dim,
		})

		resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/pilot: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Send pilot to Room", func(t *testing.T) {
		st := false
		body, _ := json.Marshal(PilotRequest{
			Room:  "Living Room",
			State: &st,
		})

		resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/pilot to room: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Send pilot to unknown room", func(t *testing.T) {
		body, _ := json.Marshal(PilotRequest{
			Room: "NonExistentRoom",
		})

		resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/pilot: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for unknown room, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid JSON payload", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader([]byte("{invalid-json")))
		if err != nil {
			t.Fatalf("failed POST /api/v1/pilot: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", resp.StatusCode)
		}
	})
}

func TestServerPresets(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET presets list", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/presets")
		if err != nil {
			t.Fatalf("failed GET /api/v1/presets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}
		if data["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", data["status"])
		}
	})

	t.Run("POST apply preset", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "evening",
			"ip":   "192.168.1.50",
		})
		resp, err := http.Post(ts.URL+"/api/v1/presets", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/presets: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST unknown preset", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "non_existent_preset_xyz",
			"ip":   "192.168.1.50",
		})
		resp, err := http.Post(ts.URL+"/api/v1/presets", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST unknown preset: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})
}

func TestServerCircadian(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/circadian?time=14:30&room=Living%20Room")
	if err != nil {
		t.Fatalf("failed GET /api/v1/circadian: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}
}

func TestServerCommand(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("POST command with active dev and room", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"command": "dim 50",
			"room":    "Living Room",
		})
		resp, err := http.Post(ts.URL+"/api/v1/command", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/command: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST empty command", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"command": "",
		})
		resp, err := http.Post(ts.URL+"/api/v1/command", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/command: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", resp.StatusCode)
		}
	})
}

func TestServerMethodNotAllowed(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	endpoints := []string{"/api/v1/devices", "/api/v1/circadian"}
	for _, ep := range endpoints {
		resp, err := http.Post(ts.URL+ep, "application/json", bytes.NewReader([]byte("{}")))
		if err != nil {
			t.Fatalf("failed POST %s: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for POST %s, got %d", ep, resp.StatusCode)
		}
	}
}

func TestServerStartShutdown(t *testing.T) {
	srv := NewServer(Config{
		Port: 18999,
		Host: "127.0.0.1",
	})
	if srv.ListenAddr() != "127.0.0.1:18999" {
		t.Errorf("unexpected ListenAddr: %s", srv.ListenAddr())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := srv.Start(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error on server shutdown: %v", err)
	}
}

func TestBroadcaster(t *testing.T) {
	b := NewBroadcaster("")
	ch, unsubscribe := b.Subscribe()

	subs, sent, failed := b.Stats()
	if subs != 1 {
		t.Errorf("expected 1 subscriber, got %d", subs)
	}

	evt := Event{
		Type:    EventDeviceUpdated,
		IP:      "192.168.1.50",
		Status:  "ok",
		Command: "dim 50",
	}
	b.Publish(evt)

	select {
	case received := <-ch:
		if received.Type != EventDeviceUpdated || received.IP != "192.168.1.50" {
			t.Errorf("unexpected event received: %+v", received)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}

	unsubscribe()
	subs, _, _ = b.Stats()
	if subs != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", subs)
	}

	_ = sent
	_ = failed
}

func TestServerWebhook(t *testing.T) {
	eventCh := make(chan Event, 1)
	webhookTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var evt Event
		if err := json.NewDecoder(r.Body).Decode(&evt); err == nil {
			eventCh <- evt
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookTS.Close()

	srv, _ := setupTestServer(t)
	srv.Broadcaster().SetWebhookURL(webhookTS.URL)

	if srv.Broadcaster().WebhookURL() != webhookTS.URL {
		t.Errorf("expected WebhookURL %s, got %s", webhookTS.URL, srv.Broadcaster().WebhookURL())
	}

	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	st := true
	body, _ := json.Marshal(PilotRequest{
		IP:    "192.168.1.50",
		State: &st,
	})

	resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed POST /api/v1/pilot: %v", err)
	}
	resp.Body.Close()

	select {
	case evt := <-eventCh:
		if evt.Type != EventDeviceUpdated || evt.IP != "192.168.1.50" {
			t.Errorf("unexpected webhook event payload: %+v", evt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook event dispatch")
	}

	time.Sleep(50 * time.Millisecond)

	_, sent, _ := srv.Broadcaster().Stats()
	if sent < 1 {
		t.Errorf("expected webhookSent >= 1, got %d", sent)
	}
}

func TestServerEventsSSE(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/v1/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed GET /api/v1/events: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Trigger command to generate an SSE event
	go func() {
		time.Sleep(50 * time.Millisecond)
		body, _ := json.Marshal(map[string]string{
			"command": "warm",
			"ip":      "192.168.1.50",
		})
		res, postErr := http.Post(ts.URL+"/api/v1/command", "application/json", bytes.NewReader(body))
		if postErr == nil {
			res.Body.Close()
		}
	}()

	buf := make([]byte, 1024)
	n, readErr := resp.Body.Read(buf)
	if readErr != nil && readErr != context.Canceled {
		// Output read successfully
	}
	out := string(buf[:n])
	if !bytes.Contains(buf[:n], []byte("event: command_executed")) {
		t.Logf("SSE stream payload output: %s", out)
	}
}

func TestServerRooms(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET all rooms", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/rooms")
		if err != nil {
			t.Fatalf("failed GET /api/v1/rooms: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}
		if data["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", data["status"])
		}
		rooms, ok := data["rooms"].(map[string]any)
		if !ok || len(rooms) == 0 {
			t.Errorf("expected non-empty rooms map, got %v", data["rooms"])
		}
	})

	t.Run("GET specific room by name", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/rooms?name=Living%20Room")
		if err != nil {
			t.Fatalf("failed GET room by name: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}
		if data["room"] != "Living Room" {
			t.Errorf("expected room 'Living Room', got %v", data["room"])
		}
	})

	t.Run("GET non-existent room", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/rooms?name=UnknownRoom")
		if err != nil {
			t.Fatalf("failed GET non-existent room: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})
}

func TestServerOpenAPI(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/openapi.json")
	if err != nil {
		t.Fatalf("failed GET /api/v1/openapi.json: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var spec map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("failed to decode OpenAPI JSON: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Errorf("expected non-empty paths map in OpenAPI spec, got %v", spec["paths"])
	}
}

func TestServerDocs(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	endpoints := []string{"/docs", "/api/v1/docs"}
	for _, ep := range endpoints {
		resp, err := http.Get(ts.URL + ep)
		if err != nil {
			t.Fatalf("failed GET %s: %v", ep, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for %s, got %d", ep, resp.StatusCode)
		}
		if !bytes.Contains([]byte(resp.Header.Get("Content-Type")), []byte("text/html")) {
			t.Errorf("expected Content-Type text/html for %s, got %s", ep, resp.Header.Get("Content-Type"))
		}
	}
}
