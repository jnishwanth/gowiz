package wiz

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDiscovery(t *testing.T) {
	t.Run("inferBroadcastAddress format", func(t *testing.T) {
		ipStr := inferBroadcastAddress()
		if ipStr == "" {
			t.Fatalf("expected non-empty broadcast address")
		}
		ip := net.ParseIP(ipStr)
		if ip == nil {
			t.Errorf("expected valid IP address, got %q", ipStr)
		}
	})

	t.Run("DiscoverSmartBulbs with immediate context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ips, err := DiscoverSmartBulbs(ctx, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ips) == 0 {
			t.Errorf("expected fallback IP when discovery yields no devices")
		}
	})
}
