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
