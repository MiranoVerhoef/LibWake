package wol

import (
  "context"
  "errors"
  "fmt"
  "net"
  "syscall"
)

const ethertypeWOL = 0x0842

func htons(v uint16) uint16 {
  return (v<<8)&0xff00 | v>>8
}

// linkLayerAddr is a minimal net.Addr implementation for AF_PACKET sources.
type linkLayerAddr struct {
  Ifindex int
  HW      net.HardwareAddr
}

func (a linkLayerAddr) Network() string { return "linklayer" }

func (a linkLayerAddr) String() string {
  if len(a.HW) > 0 {
    return a.HW.String()
  }
  return fmt.Sprintf("ifindex=%d", a.Ifindex)
}

type unknownAddr string

func (a unknownAddr) Network() string { return "unknown" }
func (a unknownAddr) String() string  { return string(a) }

func sockaddrToAddr(sa syscall.Sockaddr) net.Addr {
  switch v := sa.(type) {
  case *syscall.SockaddrLinklayer:
    var hw net.HardwareAddr
    if v.Halen > 0 {
      hw = make(net.HardwareAddr, v.Halen)
      copy(hw, v.Addr[:v.Halen])
    }
    return linkLayerAddr{Ifindex: v.Ifindex, HW: hw}
  case *syscall.SockaddrInet4:
    ip := net.IPv4(v.Addr[0], v.Addr[1], v.Addr[2], v.Addr[3])
    return &net.IPAddr{IP: ip}
  case *syscall.SockaddrInet6:
    ip := make(net.IP, net.IPv6len)
    copy(ip, v.Addr[:])
    return &net.IPAddr{IP: ip}
  default:
    return unknownAddr(fmt.Sprintf("%T", sa))
  }
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
    onFrame(frame, sockaddrToAddr(from))
  }
}
