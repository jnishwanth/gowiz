package wiz

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
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

func TestUDPClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	udpClient := NewUDPClient(500 * time.Millisecond)
	if udpClient.Timeout != 500*time.Millisecond {
		t.Errorf("expected timeout 500ms, got %v", udpClient.Timeout)
	}

	t.Run("Empty IP error validation", func(t *testing.T) {
		err := udpClient.SendCommand(ctx, "", NewPowerParams(true))
		if err == nil {
			t.Errorf("expected error for empty IP in SendCommand")
		}
		_, err = udpClient.GetPilot(ctx, "")
		if err == nil {
			t.Errorf("expected error for empty IP in GetPilot")
		}
	})

	t.Run("Unreachable IP timeout", func(t *testing.T) {
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer shortCancel()
		err := udpClient.SendCommand(shortCtx, "192.0.2.1", NewPowerParams(true))
		if err == nil {
			t.Errorf("expected timeout error for unreachable IP")
		}
	})

	t.Run("Local UDP mock server communication", func(t *testing.T) {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", DefaultWiZPort))
		if err != nil {
			t.Skipf("cannot resolve UDP addr: %v", err)
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			t.Skipf("port %d in use, skipping local UDP socket server test: %v", DefaultWiZPort, err)
		}
		defer conn.Close()

		go func() {
			buf := make([]byte, 2048)
			for {
				n, rAddr, readErr := conn.ReadFromUDP(buf)
				if readErr != nil {
					return
				}
				req := string(buf[:n])
				if strings.Contains(req, "setPilot") {
					reply := []byte(`{"method":"setPilot","env":"pro","result":{"success":true}}`)
					_, _ = conn.WriteToUDP(reply, rAddr)
				} else if strings.Contains(req, "getPilot") {
					reply := []byte(`{"method":"getPilot","env":"pro","result":{"state":true,"dimming":90,"temp":2700}}`)
					_, _ = conn.WriteToUDP(reply, rAddr)
				}
			}
		}()

		sendErr := udpClient.SendCommand(ctx, "127.0.0.1", NewDimmingParams(90))
		if sendErr != nil {
			t.Errorf("unexpected SendCommand error: %v", sendErr)
		}

		pilot, getErr := udpClient.GetPilot(ctx, "127.0.0.1")
		if getErr != nil {
			t.Errorf("unexpected GetPilot error: %v", getErr)
		} else if pilot.Dimming == nil || *pilot.Dimming != 90 {
			t.Errorf("expected dimming 90 from mock UDP server, got %v", pilot.Dimming)
		}

		batchRes := udpClient.SendBatchCommand(ctx, []string{"127.0.0.1"}, NewPowerParams(true))
		if batchRes["127.0.0.1"] != nil {
			t.Errorf("unexpected batch command error: %v", batchRes["127.0.0.1"])
		}
	})
}
