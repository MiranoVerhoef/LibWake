package wol

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
)

const soBindToDevice = 25 // SO_BINDTODEVICE on Linux

// ListenUDP starts UDP listeners on the given ports. If iface is non-empty, the
// sockets are bound to that interface via SO_BINDTODEVICE (requires root).
func ListenUDP(ctx context.Context, iface string, ports []int, onPacket func(payload []byte, src net.Addr)) error {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var innerErr error
			_ = c.Control(func(fd uintptr) {
				// Best-effort reuseaddr.
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if iface != "" {
					innerErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, soBindToDevice, iface)
				}
			})
			return innerErr
		},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(ports))

	for _, port := range ports {
		p := port
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := fmt.Sprintf(":%d", p)
			pc, err := lc.ListenPacket(ctx, "udp4", addr)
			if err != nil {
				errCh <- err
				return
			}
			defer pc.Close()

			buf := make([]byte, 4096)
			for {
				_ = pc.SetReadDeadline(time.Now().Add(750 * time.Millisecond))
				n, src, err := pc.ReadFrom(buf)
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						if ctx.Err() != nil {
							return
						}
						continue
					}
					if ctx.Err() != nil {
						return
					}
					errCh <- err
					return
				}
				if n <= 0 {
					continue
				}
				pkt := make([]byte, n)
				copy(pkt, buf[:n])
				onPacket(pkt, src)
			}
		}()
	}

	// Wait for cancellation or the first error.
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		wg.Wait()
		return nil
	}
}
