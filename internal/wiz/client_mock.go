package wiz

import (
	"context"
	"sync"
)

// MockClient simulates WiZ device state mutations in memory for testing
type MockClient struct {
	mu           sync.Mutex
	Devices      map[string]*PilotParams
	LastSent     map[string]PilotParams
	CommandCalls int
	ShouldFail   bool
}

func NewMockClient() *MockClient {
	state := true
	dim := 100
	temp := 2700
	return &MockClient{
		Devices: map[string]*PilotParams{
			FallbackIP: {
				State:   &state,
				Dimming: &dim,
				Temp:    &temp,
			},
		},
		LastSent: make(map[string]PilotParams),
	}
}

func (m *MockClient) SendCommand(ctx context.Context, ip string, params PilotParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CommandCalls++
	if m.ShouldFail {
		return context.DeadlineExceeded
	}

	m.LastSent[ip] = params

	target, found := m.Devices[ip]
	if !found {
		target = &PilotParams{}
		m.Devices[ip] = target
	}

	if params.State != nil {
		target.State = params.State
	}
	if params.Dimming != nil {
		target.Dimming = params.Dimming
	}
	if params.R != nil {
		target.R = params.R
	}
	if params.G != nil {
		target.G = params.G
	}
	if params.B != nil {
		target.B = params.B
	}
	if params.Temp != nil {
		target.Temp = params.Temp
	}
	if params.SceneID != nil {
		target.SceneID = params.SceneID
	}
	if params.Speed != nil {
		target.Speed = params.Speed
	}

	return nil
}

func (m *MockClient) SendBatchCommand(ctx context.Context, ips []string, params PilotParams) map[string]error {
	results := make(map[string]error)
	for _, ip := range ips {
		results[ip] = m.SendCommand(ctx, ip, params)
	}
	return results
}

func (m *MockClient) GetPilot(ctx context.Context, ip string) (*PilotParams, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		return nil, context.DeadlineExceeded
	}

	target, found := m.Devices[ip]
	if !found {
		t := true
		d := 100
		return &PilotParams{State: &t, Dimming: &d}, nil
	}
	return target, nil
}
