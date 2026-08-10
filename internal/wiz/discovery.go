package wiz

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DiscoverSmartBulbs scans local network broadcasts to find all active WiZ smart lights
func DiscoverSmartBulbs(ctx context.Context, timeout time.Duration) ([]string, error) {
	broadcastIP := inferBroadcastAddress()

	listenAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return []string{FallbackIP}, nil
	}
	listenConn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		return []string{FallbackIP}, nil
	}
	defer listenConn.Close()

	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", broadcastIP, DefaultWiZPort))
	if err != nil {
		return []string{FallbackIP}, nil
	}

	payload, _ := BuildGetSystemConfigPayload()
	_, err = listenConn.WriteToUDP(payload, targetAddr)
	if err != nil {
		return []string{FallbackIP}, nil
	}

	_ = listenConn.SetReadDeadline(time.Now().Add(timeout))

	foundIPs := make(map[string]bool)
	buf := make([]byte, 2048)

	for {
		select {
		case <-ctx.Done():
			break
		default:
		}

		_, rAddr, err := listenConn.ReadFromUDP(buf)
		if err != nil {
			// Scan deadline reached
			break
		}
		foundIPs[rAddr.IP.String()] = true
	}

	var ips []string
	for ip := range foundIPs {
		ips = append(ips, ip)
	}

	if len(ips) == 0 {
		return []string{FallbackIP}, nil
	}

	return ips, nil
}

// inferBroadcastAddress identifies active broadcast IP
func inferBroadcastAddress() string {
	conn, err := net.Dial("udp", "9.9.9.9:80")
	if err != nil {
		return "255.255.255.255"
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "255.255.255.255"
	}

	ipParts := strings.Split(localAddr.IP.String(), ".")
	if len(ipParts) != 4 {
		return "255.255.255.255"
	}
	ipParts[3] = "255"
	return strings.Join(ipParts, ".")
}
