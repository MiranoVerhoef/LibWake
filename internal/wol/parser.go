package wol

import (
	"bytes"
	"net"
)

const (
	macLen       = 6
	magicFFCount = 6
	magicRepeat  = 16
	magicMinSize = magicFFCount + (magicRepeat * macLen) // 6 + 96 = 102
)

// FindMagicMAC scans payload for a WOL magic packet and returns the target MAC.
//
// Compatibility:
//   - Allows leading padding.
//   - Ignores trailing bytes (e.g. SecureOn password or padding).
func FindMagicMAC(payload []byte) (string, bool) {
	if len(payload) < magicMinSize {
		return "", false
	}

	ff := bytes.Repeat([]byte{0xff}, magicFFCount)

	for i := 0; i <= len(payload)-magicMinSize; i++ {
		if !bytes.Equal(payload[i:i+magicFFCount], ff) {
			continue
		}

		mac := payload[i+magicFFCount : i+magicFFCount+macLen]
		if isZero(mac) {
			continue
		}

		ok := true
		for j := 0; j < magicRepeat; j++ {
			start := i + magicFFCount + (j * macLen)
			end := start + macLen
			if end > len(payload) || !bytes.Equal(payload[start:end], mac) {
				ok = false
				break
			}
		}

		if ok {
			return net.HardwareAddr(mac).String(), true
		}
	}

	return "", false
}

func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
