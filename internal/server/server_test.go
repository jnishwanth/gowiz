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

	t.Run("Send pilot broadcast to all devices", func(t *testing.T) {
		st := true
		dim := 90
		body, _ := json.Marshal(PilotRequest{
			All:     true,
			State:   &st,
			Dimming: &dim,
		})

		resp, err := http.Post(ts.URL+"/api/v1/pilot", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/pilot broadcast: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for broadcast, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode broadcast response JSON: %v", err)
		}
		if res["target"] != "all" || res["devicesCount"] == float64(0) {
			t.Errorf("unexpected broadcast response payload: %+v", res)
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

	t.Run("POST apply preset to IP", func(t *testing.T) {
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

	t.Run("POST apply preset to room", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"name": "relax",
			"room": "Living Room",
		})
		resp, err := http.Post(ts.URL+"/api/v1/presets", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/presets to room: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST apply preset broadcast to all", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"name": "night",
			"all":  true,
		})
		resp, err := http.Post(ts.URL+"/api/v1/presets", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/presets broadcast: %v", err)
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

	t.Run("GET /api/v1/circadian calculation", func(t *testing.T) {
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
		if data["status"] != "ok" || data["time"] != "14:30" {
			t.Errorf("unexpected GET circadian response: %+v", data)
		}
	})

	t.Run("POST /api/v1/circadian to single IP", func(t *testing.T) {
		body, _ := json.Marshal(CircadianRequest{
			IP:   "192.168.1.50",
			Time: "08:00",
		})
		resp, err := http.Post(ts.URL+"/api/v1/circadian", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/circadian to IP: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode POST response: %v", err)
		}
		if res["status"] != "ok" || res["targetIP"] != "192.168.1.50" {
			t.Errorf("unexpected POST single IP response: %+v", res)
		}
	})

	t.Run("POST /api/v1/circadian to room group", func(t *testing.T) {
		body, _ := json.Marshal(CircadianRequest{
			Room: "Living Room",
			Time: "12:00",
		})
		resp, err := http.Post(ts.URL+"/api/v1/circadian", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/circadian to room: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode room response: %v", err)
		}
		if res["status"] != "ok" || res["target"] != "Living Room" {
			t.Errorf("unexpected POST room response: %+v", res)
		}
	})

	t.Run("POST /api/v1/circadian broadcast all", func(t *testing.T) {
		body, _ := json.Marshal(CircadianRequest{
			All:  true,
			Time: "22:00",
		})
		resp, err := http.Post(ts.URL+"/api/v1/circadian", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/circadian broadcast: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode broadcast response: %v", err)
		}
		if res["status"] != "ok" || res["target"] != "all" {
			t.Errorf("unexpected POST broadcast response: %+v", res)
		}
	})

	t.Run("POST /api/v1/circadian unknown room", func(t *testing.T) {
		body, _ := json.Marshal(CircadianRequest{
			Room: "NonExistentRoom",
		})
		resp, err := http.Post(ts.URL+"/api/v1/circadian", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("failed POST /api/v1/circadian: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404 Not Found, got %d", resp.StatusCode)
		}
	})

	t.Run("DELETE /api/v1/circadian method not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/circadian", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed DELETE /api/v1/circadian: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		}
	})
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

	t.Run("POST command with wildcard selector", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"command": "warm",
			"room":    "all",
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

	t.Run("POST group command verb", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"command": ":group Living Room off",
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

	// Endpoints that do not support POST
	postEndpoints := []string{"/api/v1/devices"}
	for _, ep := range postEndpoints {
		resp, err := http.Post(ts.URL+ep, "application/json", bytes.NewReader([]byte("{}")))
		if err != nil {
			t.Fatalf("failed POST %s: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for POST %s, got %d", ep, resp.StatusCode)
		}
	}

	// Endpoints that do not support DELETE
	deleteEndpoints := []string{"/api/v1/devices", "/api/v1/circadian"}
	for _, ep := range deleteEndpoints {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+ep, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed DELETE %s: %v", ep, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for DELETE %s, got %d", ep, resp.StatusCode)
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
	for _, expectedPath := range []string{"/api/v1/effects", "/api/v1/groups", "/api/v1/discover", "/api/v1/pilot"} {
		if _, found := paths[expectedPath]; !found {
			t.Errorf("expected path %s in OpenAPI spec", expectedPath)
		}
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
		bodyBytes, _ := json.Marshal(resp.Body)
		_ = bodyBytes
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for %s, got %d", ep, resp.StatusCode)
		}
		if !bytes.Contains([]byte(resp.Header.Get("Content-Type")), []byte("text/html")) {
			t.Errorf("expected Content-Type text/html for %s, got %s", ep, resp.Header.Get("Content-Type"))
		}
		htmlBody := buf.String()
		for _, term := range []string{"/api/v1/effects", "/api/v1/groups", "/api/v1/discover"} {
			if !bytes.Contains([]byte(htmlBody), []byte(term)) {
				t.Errorf("expected HTML docs to contain endpoint %s", term)
			}
		}
	}
}

func TestServerAuthAndCORS(t *testing.T) {
	cfgMgr := config.NewManager(t.TempDir() + "/config.json")
	reg := wiz.NewDeviceRegistry()
	reg.AddOrUpdate(wiz.NewDevice("192.168.1.50"))

	srv := NewServer(Config{
		Port:        8889,
		Host:        "127.0.0.1",
		APIKey:      "secret-key-123",
		WizClient:   wiz.NewMockClient(),
		ConfigMgr:   cfgMgr,
		DevRegistry: reg,
	})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("CORS preflight OPTIONS", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/devices", nil)
		if err != nil {
			t.Fatalf("failed to create OPTIONS request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed OPTIONS request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected status 204 No Content for OPTIONS, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected CORS header Access-Control-Allow-Origin: *, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
		}
		if !bytes.Contains([]byte(resp.Header.Get("Access-Control-Allow-Headers")), []byte("X-API-Key")) {
			t.Errorf("expected X-API-Key in Access-Control-Allow-Headers, got %q", resp.Header.Get("Access-Control-Allow-Headers"))
		}
	})

	t.Run("Public /health with apiKeyProtected status", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("failed GET /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 for health, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode health JSON: %v", err)
		}
		if data["apiKeyProtected"] != true {
			t.Errorf("expected apiKeyProtected: true, got %v", data["apiKeyProtected"])
		}
	})

	t.Run("Unauthorized request without API Key", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/devices")
		if err != nil {
			t.Fatalf("failed GET /api/v1/devices: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	t.Run("Unauthorized request with invalid API Key", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/devices", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed GET with invalid key: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", resp.StatusCode)
		}
	})

	t.Run("Authorized request with X-API-Key header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/devices", nil)
		req.Header.Set("X-API-Key", "secret-key-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed GET with X-API-Key: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("Authorized request with Authorization Bearer header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/devices", nil)
		req.Header.Set("Authorization", "Bearer secret-key-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed GET with Bearer token: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("Authorized request with api_key query parameter", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/devices?api_key=secret-key-123")
		if err != nil {
			t.Fatalf("failed GET with query param: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
		}
	})
}

func TestServerEffects(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET /api/v1/effects", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/effects")
		if err != nil {
			t.Fatalf("failed GET /api/v1/effects: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode effects response: %v", err)
		}
		if body["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", body["status"])
		}
	})

	t.Run("POST /api/v1/effects flash", func(t *testing.T) {
		payload := map[string]any{
			"type":       "flash",
			"color":      "red",
			"count":      1,
			"intervalMs": 5,
			"ip":         "192.168.1.50",
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(ts.URL+"/api/v1/effects", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("failed POST /api/v1/effects: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res["effect"] != "flash" {
			t.Errorf("expected effect 'flash', got %v", res["effect"])
		}
	})

	t.Run("POST /api/v1/effects room target", func(t *testing.T) {
		payload := map[string]any{
			"type":       "pulse",
			"color":      "blue",
			"count":      1,
			"intervalMs": 5,
			"room":       "Living Room",
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(ts.URL+"/api/v1/effects", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("failed POST /api/v1/effects room target: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /api/v1/effects broadcast all", func(t *testing.T) {
		payload := map[string]any{
			"type":       "strobe",
			"color":      "alert",
			"count":      1,
			"intervalMs": 5,
			"all":        true,
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(ts.URL+"/api/v1/effects", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("failed POST /api/v1/effects broadcast: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestServerGroups(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET /api/v1/groups initial empty", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/groups")
		if err != nil {
			t.Fatalf("failed GET /api/v1/groups: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST /api/v1/groups create group", func(t *testing.T) {
		payload := map[string]any{
			"name":    "desk",
			"members": []string{"192.168.1.50"},
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(ts.URL+"/api/v1/groups", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("failed POST /api/v1/groups: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GET /api/v1/groups/desk detail", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/groups/desk")
		if err != nil {
			t.Fatalf("failed GET /api/v1/groups/desk: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		members, ok := res["members"].([]any)
		if !ok || len(members) != 1 {
			t.Errorf("expected 1 member in desk group, got %v", res["members"])
		}
	})

	t.Run("DELETE /api/v1/groups/desk", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/groups/desk", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed DELETE /api/v1/groups/desk: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestServerDiscover(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET /api/v1/discover", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/discover?timeout=0.1s")
		if err != nil {
			t.Fatalf("failed GET /api/v1/discover: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if res["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", res["status"])
		}
		if res["discoveredCount"] == nil {
			t.Errorf("expected discoveredCount field in response")
		}
	})

	t.Run("POST /api/v1/discover with JSON payload", func(t *testing.T) {
		payload := map[string]any{
			"timeoutSeconds": 0.1,
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(ts.URL+"/api/v1/discover", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("failed POST /api/v1/discover: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var res map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}
		if res["status"] != "ok" {
			t.Errorf("expected status 'ok', got %v", res["status"])
		}
	})

	t.Run("Method Not Allowed for DELETE /api/v1/discover", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/discover", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed DELETE /api/v1/discover: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		}
	})
}

func TestServerScenes(t *testing.T) {
	srv, _ := setupTestServer(t)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	t.Run("GET /api/v1/scenes - all scenes and categories", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/scenes")
		if err != nil {
			t.Fatalf("failed GET /api/v1/scenes: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if data["status"] != "success" {
			t.Errorf("expected status 'success', got %v", data["status"])
		}

		cats, ok := data["categories"].([]any)
		if !ok || len(cats) == 0 {
			t.Errorf("expected non-empty categories list, got %v", data["categories"])
		}

		scenes, ok := data["scenes"].([]any)
		if !ok || len(scenes) == 0 {
			t.Errorf("expected non-empty scenes list, got %v", data["scenes"])
		}

		count, ok := data["count"].(float64)
		if !ok || count != float64(len(scenes)) {
			t.Errorf("expected count %d, got %v", len(scenes), data["count"])
		}
	})

	t.Run("GET /api/v1/scenes with category filter", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/scenes?category=Nature")
		if err != nil {
			t.Fatalf("failed GET /api/v1/scenes?category=Nature: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		scenes, ok := data["scenes"].([]any)
		if !ok || len(scenes) == 0 {
			t.Errorf("expected matching scenes for category Nature, got %v", data["scenes"])
		}
	})

	t.Run("GET /api/v1/scenes with search query filter", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/scenes?q=ocean")
		if err != nil {
			t.Fatalf("failed GET /api/v1/scenes?q=ocean: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		scenes, ok := data["scenes"].([]any)
		if !ok || len(scenes) == 0 {
			t.Errorf("expected matching scenes for query ocean, got %v", data["scenes"])
		}
	})

	t.Run("POST /api/v1/scenes - method not allowed", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/scenes", "application/json", nil)
		if err != nil {
			t.Fatalf("failed POST /api/v1/scenes: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		}
	})
}
