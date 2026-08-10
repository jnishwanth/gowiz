package wiz

import (
	"context"
	"testing"
)

func TestMockClient(t *testing.T) {
	ctx := context.Background()
	mock := NewMockClient()

	t.Run("SendCommand state update", func(t *testing.T) {
		err := mock.SendCommand(ctx, FallbackIP, NewDimmingParams(50))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pilot, err := mock.GetPilot(ctx, FallbackIP)
		if err != nil {
			t.Fatalf("failed to get pilot: %v", err)
		}
		if pilot.Dimming == nil || *pilot.Dimming != 50 {
			t.Errorf("expected dimming 50, got %v", pilot.Dimming)
		}
	})

	t.Run("SendBatchCommand across multiple IPs", func(t *testing.T) {
		ips := []string{"192.168.1.10", "192.168.1.11"}
		results := mock.SendBatchCommand(ctx, ips, NewPowerParams(false))

		for ip, err := range results {
			if err != nil {
				t.Errorf("unexpected error for %s: %v", ip, err)
			}
			pilot, _ := mock.GetPilot(ctx, ip)
			if pilot.State == nil || *pilot.State {
				t.Errorf("expected state false for %s, got %v", ip, pilot.State)
			}
		}
	})
}
