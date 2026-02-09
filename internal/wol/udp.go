package wol

import (
	"encoding/hex"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Event struct {
	MAC      string
	SourceIP string
}

func normalizeMAC(mac string) string {
	mac = strings.ToLower(strings.TrimSpace(mac))
	mac = strings.ReplaceAll(mac, "-", ":")
	// handle 12 hex chars
	mac = strings.ReplaceAll(mac, ":", "")
	if len(mac) == 12 {
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6] + ":" + mac[6:8] + ":" + mac[8:10] + ":" + mac[10:12]
	}
	// if already has colons, re-add with normalization
	mac = strings.ReplaceAll(mac, ":", "")
	if len(mac) == 12 {
		return mac[0:2] + ":" + mac[2:4] + ":" + mac[4:6] + ":" + mac[6:8] + ":" + mac[8:10] + ":" + mac[10:12]
	}
	return strings.ToLower(strings.TrimSpace(mac))
}

// ParseMagicPacket validates and extracts the MAC address from a WOL magic packet.
// Magic packet: 6x 0xFF then 16 repetitions of target MAC (6 bytes) = 102 bytes minimum.
func ParseMagicPacket(b []byte) (string, bool) {
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
	h := hex.EncodeToString(mac)
	return normalizeMAC(h), true
}

// ListenUDP listens on the given UDP ports and calls onEvent for valid WOL packets.
func ListenUDP(ports []int, onEvent func(Event), logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}
	conns := []*net.UDPConn{}
	for _, port := range ports {
		addr := &net.UDPAddr{IP: net.IPv4zero, Port: port}
		c, err := net.ListenUDP("udp4", addr)
		if err != nil {
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
			buf := make([]byte, 2048)
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
	return nil
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

// LoadEnabledMACs reads a simple INI-like file where each line is: aa:bb:cc:dd:ee:ff=yes
// Returns a set of enabled MACs.
func LoadEnabledMACs(path string) map[string]bool {
	out := map[string]bool{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";") {
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			continue
		}
		mac := normalizeMAC(parts[0])
		val := strings.ToLower(strings.TrimSpace(parts[1]))
		if mac == "" {
			continue
		}
		if val == "yes" || val == "true" || val == "1" || val == "on" {
			out[mac] = true
		}
	}
	return out
}
