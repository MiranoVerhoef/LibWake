package wol

import (
	"encoding/binary"
	"log"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

func htons(i uint16) uint16 { return (i<<8)&0xff00 | i>>8 }

// ListenEtherType listens for raw ethernet frames with the specified ethertype (e.g. 0x0842 for WOL)
// and calls onEvent when a valid WOL magic packet is found in the payload.
// No libpcap required (uses AF_PACKET).
func ListenEtherType(iface string, ethertype uint16, onEvent func(Event), logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return err
	}

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(ethertype)))
	if err != nil {
		return err
	}

	sll := &unix.SockaddrLinklayer{
		Protocol: htons(ethertype),
		Ifindex:  ifi.Index,
	}
	if err := unix.Bind(fd, sll); err != nil {
		_ = unix.Close(fd)
		return err
	}

	logger.Printf("listening ETH %s ethertype=0x%04x", iface, ethertype)

	go func() {
		defer unix.Close(fd)
		buf := make([]byte, 2048)
		for {
			// timeout so we can stay responsive
			_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &unix.Timeval{Sec: 300, Usec: 0})
			n, from, err := unix.Recvfrom(fd, buf, 0)
			if err != nil {
				// ignore timeouts
				if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
					continue
				}
				logger.Printf("eth recv error: %v", err)
				continue
			}
			if n < 14+102 {
				continue
			}
			// ethernet header is 14 bytes: dst(6) src(6) type(2)
			etype := binary.BigEndian.Uint16(buf[12:14])
			if etype != ethertype {
				continue
			}
			payload := buf[14:n]
			mac, ok := ParseMagicPacket(payload)
			if !ok {
				continue
			}
			srcIP := "ether"
			if sa, ok2 := from.(*unix.SockaddrLinklayer); ok2 {
				_ = sa
			}
			onEvent(Event{MAC: mac, SourceIP: srcIP})
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return nil
}
