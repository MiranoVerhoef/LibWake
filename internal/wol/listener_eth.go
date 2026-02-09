package wol

import (
  "context"
  "errors"
  "net"
  "syscall"
)

const ethertypeWOL = 0x0842

func htons(v uint16) uint16 {
  return (v<<8)&0xff00 | v>>8
}

// ListenEthertypeWOL listens for EtherType 0x0842 frames on the given interface.
// This covers Ethernet-style WOL packets which don't use UDP.
func ListenEthertypeWOL(ctx context.Context, iface string, onFrame func(frame []byte, src net.Addr)) error {
  if iface == "" {
    return nil
  }

  ni, err := net.InterfaceByName(iface)
  if err != nil {
    return err
  }

  proto := htons(ethertypeWOL)
  fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(proto))
  if err != nil {
    return err
  }

  sa := &syscall.SockaddrLinklayer{Protocol: proto, Ifindex: ni.Index}
  if err := syscall.Bind(fd, sa); err != nil {
    _ = syscall.Close(fd)
    return err
  }

  // Close the socket on cancellation.
  go func() {
    <-ctx.Done()
    _ = syscall.Close(fd)
  }()

  buf := make([]byte, 65535)
  for {
    n, from, err := syscall.Recvfrom(fd, buf, 0)
    if err != nil {
      if ctx.Err() != nil {
        return nil
      }
      if errors.Is(err, syscall.EINTR) {
        continue
      }
      // Closing the fd from another goroutine can surface as EBADF.
      if errors.Is(err, syscall.EBADF) {
        return nil
      }
      return err
    }
    if n <= 0 {
      continue
    }
    frame := make([]byte, n)
    copy(frame, buf[:n])
    onFrame(frame, from)
  }
}
