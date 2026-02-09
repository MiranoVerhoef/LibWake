package wol

import (
	"encoding/hex"
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

type Event struct {
	MAC      string
	SourceIP string
}

func normalizeMAC(mac string) string {
	mac = strings.ToLower(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ReplaceAll(mac, ".", "")
	if strings.Contains(mac, ":") {
		parts := strings.Split(mac, ":")
		if len(parts) == 6 {
			for i := range parts {
				if len(parts[i]) == 1 {
					parts[i] = "0" + parts[i]
				}
			}
			return strings.Join(parts, ":")
		}
	}
	// fallback if it's 12 hex chars
	if len(mac) == 12 {
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6] + ":" + mac[6:8] + ":" + mac[8:10] + ":" + mac[10:12]
	}
	return mac
}

func ParseMagicPacket(b []byte) (string, bool) {
	// Magic packet: 6x 0xFF then 16 repetitions of target MAC (6 bytes) = 6 + 16*6 = 102 bytes minimum
	if len(b) < 102 {
		return "", false
	}
	for i := 0; i < 6; i++ {
		if b[i] != 0xFF {
			return "", false
		}
	}
	mac := b[6:12]
	// verify repetition
	for i := 0; i < 16; i++ {
		start := 6 + i*6
		if start+6 > len(b) {
			return "", false
		}
		for j := 0; j < 6; j++ {
			if b[start+j] != mac[j] {
				return "", false
			}
		}
	}
	return normalizeMAC(hex.EncodeToString(mac)[0:2]+":"+hex.EncodeToString(mac)[2:4]+":"+hex.EncodeToString(mac)[4:6]+":"+hex.EncodeToString(mac)[6:8]+":"+hex.EncodeToString(mac)[8:10]+":"+hex.EncodeToString(mac)[10:12]), true
}

func ListenUDP(ports []int, onEvent func(Event), logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}
	conns := []*net.UDPConn{}

	for _, port := range ports {
		addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
		c, err := net.ListenUDP("udp4", addr)
		if err != nil {
			// cleanup
			for _, cc := range conns {
				_ = cc.Close()
			}
			return err
		}
		_ = c.SetReadBuffer(1 << 20)
		conns = append(conns, c)
		logger.Printf("listening UDP :%d", port)
	}

	for _, c := range conns {
		conn := c
		go func() {
			buf := make([]byte, 1500)
			for {
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
				n, src, err := conn.ReadFromUDP(buf)
				if err != nil {
					var ne net.Error
					if errors.As(err, &ne) && ne.Timeout() {
						continue
					}
					logger.Printf("udp read error: %v", err)
					continue
				}
				mac, ok := ParseMagicPacket(buf[:n])
				if !ok {
					continue
				}
				onEvent(Event{MAC: mac, SourceIP: src.IP.String()})
			}
		}()
	}

	select {} // run forever
}

func ParseCIDRs(s string) ([]*net.IPNet, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	out := []*net.IPNet{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, n, err := net.ParseCIDR(part)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func SourceAllowed(nets []*net.IPNet, ipStr string) bool {
	if len(nets) == 0 {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
