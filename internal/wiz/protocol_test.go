package wiz

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPayloadBuilders(t *testing.T) {
	t.Run("NewPowerParams", func(t *testing.T) {
		p := NewPowerParams(true)
		if p.State == nil || !*p.State {
			t.Errorf("expected state to be true, got %v", p.State)
		}
	})

	t.Run("NewDimmingParams clamping", func(t *testing.T) {
		pLow := NewDimmingParams(0)
		if *pLow.Dimming != 10 {
			t.Errorf("expected clamped dimming 10, got %d", *pLow.Dimming)
		}

		pHigh := NewDimmingParams(150)
		if *pHigh.Dimming != 100 {
			t.Errorf("expected clamped dimming 100, got %d", *pHigh.Dimming)
		}
	})

	t.Run("NewRGBParams clamping", func(t *testing.T) {
		p := NewRGBParams(300, -10, 128)
		if *p.R != 255 || *p.G != 0 || *p.B != 128 {
			t.Errorf("expected RGB (255, 0, 128), got (%d, %d, %d)", *p.R, *p.G, *p.B)
		}
		if p.State == nil || !*p.State {
			t.Errorf("expected state to be true on RGB selection")
		}
	})

	t.Run("NewTempParams clamping", func(t *testing.T) {
		p := NewTempParams(1000)
		if *p.Temp != 2200 {
			t.Errorf("expected clamped temp 2200, got %d", *p.Temp)
		}
		p2 := NewTempParams(8000)
		if *p2.Temp != 6500 {
			t.Errorf("expected clamped temp 6500, got %d", *p2.Temp)
		}
	})

	t.Run("NewSceneParams", func(t *testing.T) {
		p := NewSceneParams(5, 100)
		if *p.SceneID != 5 || *p.Speed != 100 {
			t.Errorf("expected scene 5 and speed 100, got scene %d speed %d", *p.SceneID, *p.Speed)
		}
	})

	t.Run("BuildSetPilotPayload JSON structure", func(t *testing.T) {
		params := NewRGBParams(255, 0, 0)
		data, err := BuildSetPilotPayload(params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var payload WiZPayload
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatalf("invalid json generated: %v", err)
		}

		if payload.Method != "setPilot" {
			t.Errorf("expected method setPilot, got %s", payload.Method)
		}
		if *payload.Params.R != 255 {
			t.Errorf("expected R=255, got %d", *payload.Params.R)
		}
	})

	t.Run("ParseWiZResponse", func(t *testing.T) {
		raw := []byte(`{"method":"getPilot","env":"pro","result":{"state":true,"dimming":80,"temp":2700}}`)
		resp, err := ParseWiZResponse(raw)
		if err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Method != "getPilot" {
			t.Errorf("expected method getPilot, got %s", resp.Method)
		}
		if resp.Result.Dimming == nil || *resp.Result.Dimming != 80 {
			t.Errorf("expected dimming 80, got %v", resp.Result.Dimming)
		}
	})

	t.Run("BuildGetPilotPayload and BuildGetSystemConfigPayload", func(t *testing.T) {
		getPilotBytes, err := BuildGetPilotPayload()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(getPilotBytes), `"method":"getPilot"`) {
			t.Errorf("expected getPilot method in payload, got %s", string(getPilotBytes))
		}

		sysBytes, err := BuildGetSystemConfigPayload()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(sysBytes), `"method":"getSystemConfig"`) {
			t.Errorf("expected getSystemConfig method in payload, got %s", string(sysBytes))
		}
	})

	t.Run("ParseWiZResponse with bulb error", func(t *testing.T) {
		rawErr := []byte(`{"method":"setPilot","env":"pro","error":{"code":-1001,"message":"Param error"}}`)
		resp, err := ParseWiZResponse(rawErr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Error == nil || resp.Error.Code != -1001 {
			t.Errorf("expected error code -1001, got %v", resp.Error)
		}
	})
}
