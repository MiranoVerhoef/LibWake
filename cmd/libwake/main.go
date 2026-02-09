package main

import (
  "context"
  "flag"
  "fmt"
  "log"
  "net"
  "os"
  "os/signal"
  "strings"
  "sync"
  "syscall"
  "time"

  "github.com/MiranoVerhoef/LibWake/internal/config"
  "github.com/MiranoVerhoef/LibWake/internal/libvirt"
  "github.com/MiranoVerhoef/LibWake/internal/runner"
  "github.com/MiranoVerhoef/LibWake/internal/wol"
)

func main() {
  var cfgPath string
  var verbose bool
  var disableEthertype bool

  flag.StringVar(&cfgPath, "config", "/boot/config/plugins/libwake/libwake.cfg", "Path to config file")
  flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
  flag.BoolVar(&disableEthertype, "disable-ethertype", false, "Disable EtherType 0x0842 listener")
  flag.Parse()

  log.SetFlags(log.LstdFlags)

  cfg, vmState, err := config.Load(cfgPath)
  if err != nil {
    log.Fatalf("config: %v", err)
  }
  if !cfg.Enabled {
    log.Printf("libwake disabled (ENABLED=false); exiting")
    return
  }

  domains, err := libvirt.ScanDomains(cfg.QemuDir)
  if err != nil {
    log.Fatalf("scan domains: %v", err)
  }
  macToDomain := make(map[string]libvirt.Domain)
  enabledCount := 0
  for _, d := range domains {
    if !vmState[d.UUID] {
      continue
    }
    enabledCount++
    for _, mac := range d.MACs {
      mac = strings.ToLower(strings.TrimSpace(mac))
      if mac == "" {
        continue
      }
      if _, exists := macToDomain[mac]; exists && verbose {
        log.Printf("warning: MAC %s mapped by multiple domains; last wins (%s)", mac, d)
      }
      macToDomain[mac] = d
    }
  }

  log.Printf("libwake started (iface=%s udp_ports=%v enabled_vms=%d)", cfg.Interface, cfg.UDPPorts, enabledCount)

  ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  defer cancel()

  var mu sync.Mutex
  lastStart := map[string]time.Time{} // by UUID

  handleMagic := func(srcDesc string, payload []byte, srcIP net.IP) {
    if srcIP != nil && !cfg.SrcAllowed(srcIP) {
      if verbose {
        log.Printf("ignored WOL from %s (src %s not allowed)", srcDesc, srcIP)
      }
      return
    }

    mac, ok := wol.FindMagicMAC(payload)
    if !ok {
      return
    }
    d, ok := macToDomain[mac]
    if !ok {
      if verbose {
        log.Printf("WOL for MAC %s (no matching enabled VM)", mac)
      }
      return
    }

    // Debounce per VM.
    mu.Lock()
    if t, exists := lastStart[d.UUID]; exists {
      if time.Since(t) < time.Duration(cfg.DebounceSeconds)*time.Second {
        mu.Unlock()
        if verbose {
          log.Printf("debounced WOL for %s", d)
        }
        return
      }
    }
    mu.Unlock()

    // Don't start if already running.
    if runner.IsRunning(ctx, d.Name) {
      if verbose {
        log.Printf("WOL for %s (already running)", d)
      }
      return
    }

    log.Printf("WOL matched %s (MAC %s) from %s -> starting", d, mac, srcDesc)
    if err := runner.Start(ctx, d.Name); err != nil {
      log.Printf("start %s failed: %v", d, err)
      return
    }

    mu.Lock()
    lastStart[d.UUID] = time.Now()
    mu.Unlock()
  }

  // UDP listeners.
  udpErrCh := make(chan error, 1)
  go func() {
    err := wol.ListenUDP(ctx, cfg.Interface, cfg.UDPPorts, func(payload []byte, src net.Addr) {
      var ip net.IP
      if ua, ok := src.(*net.UDPAddr); ok {
        ip = ua.IP
      }
      srcDesc := src.String()
      handleMagic("udp "+srcDesc, payload, ip)
    })
    if err != nil {
      udpErrCh <- err
    }
  }()

  // EtherType 0x0842 listener.
  ethErrCh := make(chan error, 1)
  if !disableEthertype {
    go func() {
      err := wol.ListenEthertypeWOL(ctx, cfg.Interface, func(frame []byte, _ net.Addr) {
        // Skip Ethernet header if present.
        payload := frame
        if len(frame) > 14 {
          payload = frame[14:]
        }
        handleMagic("ethertype 0x0842", payload, nil)
      })
      if err != nil {
        ethErrCh <- err
      }
    }()
  }

  select {
  case err := <-udpErrCh:
    log.Fatalf("udp listener: %v", err)
  case err := <-ethErrCh:
    log.Fatalf("ethertype listener: %v", err)
  case <-ctx.Done():
    fmt.Println("stopping")
  }
}
