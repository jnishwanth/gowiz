package wiz

import (
	"encoding/json"
	"fmt"
)

const (
	DefaultWiZPort = 38899
	FallbackIP     = "192.168.1.115"
)

// PilotParams represent parameters passed to setPilot / returned by getPilot
type PilotParams struct {
	State   *bool  `json:"state,omitempty"`
	Dimming *int   `json:"dimming,omitempty"`
	R       *int   `json:"r,omitempty"`
	G       *int   `json:"g,omitempty"`
	B       *int   `json:"b,omitempty"`
	Temp    *int   `json:"temp,omitempty"`
	SceneID *int   `json:"sceneId,omitempty"`
	Speed   *int   `json:"speed,omitempty"`
	Rssi    *int   `json:"rssi,omitempty"`
	Mac     string `json:"mac,omitempty"`
}

// WiZPayload represents outgoing JSON-RPC structure for WiZ protocol
type WiZPayload struct {
	Method string      `json:"method"`
	Params PilotParams `json:"params"`
}

// WiZResponse represents incoming UDP response from WiZ bulb
type WiZResponse struct {
	Method string      `json:"method"`
	Env    string      `json:"env,omitempty"`
	Result PilotParams `json:"result,omitempty"`
	Error  *WiZError   `json:"error,omitempty"`
}

type WiZError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SystemConfigResult is returned by getSystemConfig
type SystemConfigResult struct {
	Mac    string `json:"mac"`
	Home   int    `json:"homeId"`
	Room   int    `json:"roomId"`
	Rssi   int    `json:"rssi"`
	Module string `json:"moduleName"`
}

type SystemConfigResponse struct {
	Method string             `json:"method"`
	Result SystemConfigResult `json:"result"`
}

// Helper constructors for PilotParams

func NewPowerParams(state bool) PilotParams {
	s := state
	return PilotParams{State: &s}
}

func NewDimmingParams(level int) PilotParams {
	if level < 10 {
		level = 10
	}
	if level > 100 {
		level = 100
	}
	return PilotParams{Dimming: &level}
}

func NewRGBParams(r, g, b int) PilotParams {
	r = clamp(r, 0, 255)
	g = clamp(g, 0, 255)
	b = clamp(b, 0, 255)
	state := true
	return PilotParams{
		State: &state,
		R:     &r,
		G:     &g,
		B:     &b,
	}
}

func NewTempParams(temp int) PilotParams {
	// WiZ white temperature range: 2200K - 6500K
	temp = clamp(temp, 2200, 6500)
	state := true
	return PilotParams{
		State: &state,
		Temp:  &temp,
	}
}

func NewSceneParams(sceneID int, speed ...int) PilotParams {
	if (sceneID < 1 || sceneID > 33) && sceneID != 1000 {
		sceneID = 1
	}
	state := true
	params := PilotParams{
		State:   &state,
		SceneID: &sceneID,
	}
	if len(speed) > 0 && speed[0] > 0 {
		sp := clamp(speed[0], 20, 200)
		params.Speed = &sp
	}
	return params
}

func BuildSetPilotPayload(params PilotParams) ([]byte, error) {
	payload := WiZPayload{
		Method: "setPilot",
		Params: params,
	}
	return json.Marshal(payload)
}

func BuildGetPilotPayload() ([]byte, error) {
	payload := WiZPayload{
		Method: "getPilot",
		Params: PilotParams{},
	}
	return json.Marshal(payload)
}

func BuildGetSystemConfigPayload() ([]byte, error) {
	payload := WiZPayload{
		Method: "getSystemConfig",
		Params: PilotParams{},
	}
	return json.Marshal(payload)
}

func ParseWiZResponse(data []byte) (*WiZResponse, error) {
	var resp WiZResponse
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode WiZ response: %w", err)
	}
	return &resp, nil
}

func Clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func clamp(val, min, max int) int {
	return Clamp(val, min, max)
}
