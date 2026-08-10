package wiz

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client defines the interface for communicating with WiZ smart devices
type Client interface {
	SendCommand(ctx context.Context, ip string, params PilotParams) error
	SendBatchCommand(ctx context.Context, ips []string, params PilotParams) map[string]error
	GetPilot(ctx context.Context, ip string) (*PilotParams, error)
}

// UDPClient implements real WiZ UDP communication with confirmed acknowledgement
type UDPClient struct {
	Timeout time.Duration
}

func NewUDPClient(timeout ...time.Duration) *UDPClient {
	t := 1500 * time.Millisecond
	if len(timeout) > 0 && timeout[0] > 0 {
		t = timeout[0]
	}
	return &UDPClient{Timeout: t}
}

func (c *UDPClient) SendCommand(ctx context.Context, ip string, params PilotParams) error {
	if ip == "" {
		return fmt.Errorf("ip address cannot be empty")
	}

	payload, err := BuildSetPilotPayload(params)
	if err != nil {
		return fmt.Errorf("failed to build setPilot payload: %w", err)
	}

	listenAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return fmt.Errorf("failed to resolve local UDP addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP: %w", err)
	}
	defer conn.Close()

	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, DefaultWiZPort))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP target %s: %w", ip, err)
	}

	maxAttempts := 3
	attemptTimeout := 500 * time.Millisecond
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 {
			attemptTimeout = remaining / time.Duration(maxAttempts)
		}
	}

	buf := make([]byte, 2048)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(attemptTimeout))
		_, err = conn.WriteToUDP(payload, targetAddr)
		if err != nil {
			continue
		}

		n, _, readErr := conn.ReadFromUDP(buf)
		if readErr != nil {
			// Timeout or socket error, retry attempt
			continue
		}

		resp, parseErr := ParseWiZResponse(buf[:n])
		if parseErr != nil {
			continue
		}

		if resp.Error != nil {
			return fmt.Errorf("bulb returned error code %d: %s", resp.Error.Code, resp.Error.Message)
		}

		// Confirmed acknowledgement from physical bulb
		return nil
	}

	return fmt.Errorf("bulb at %s did not acknowledge command (unreachable or packet drop)", ip)
}

func (c *UDPClient) SendBatchCommand(ctx context.Context, ips []string, params PilotParams) map[string]error {
	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, ip := range ips {
		wg.Add(1)
		go func(targetIP string) {
			defer wg.Done()
			err := c.SendCommand(ctx, targetIP, params)
			mu.Lock()
			results[targetIP] = err
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	return results
}

func (c *UDPClient) GetPilot(ctx context.Context, ip string) (*PilotParams, error) {
	if ip == "" {
		return nil, fmt.Errorf("ip address cannot be empty")
	}

	payload, err := BuildGetPilotPayload()
	if err != nil {
		return nil, err
	}

	listenAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, DefaultWiZPort))
	if err != nil {
		return nil, err
	}

	timeout := c.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))

	_, err = conn.WriteToUDP(payload, targetAddr)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("failed reading UDP response from %s: %w", ip, err)
	}

	resp, err := ParseWiZResponse(buf[:n])
	if err != nil {
		return nil, err
	}

	return &resp.Result, nil
}
