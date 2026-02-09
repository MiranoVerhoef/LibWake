package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/MiranoVerhoef/LibWake/internal/libvirt"
	"github.com/MiranoVerhoef/LibWake/internal/wol"
)

func main() {
	iface := flag.String("iface", "br0", "Network interface for WOL listening.")
	udpPorts := flag.String("udp-ports", "7,9", "Comma-separated UDP ports to listen on (e.g. 7,9).")
	allowSubnets := flag.String("allow-subnets", "", "Optional comma-separated CIDR allowlist (e.g. 192.168.1.0/24). Empty = allow all.")
	xmlDir := flag.String("xml-dir", "/etc/libvirt/qemu", "Directory containing libvirt domain XMLs (Unraid: /etc/libvirt/qemu).")
	enabledMACsFile := flag.String("enabled-macs", "/boot/config/plugins/libwake/vms.cfg", "Per-VM WOL enable file (mac=yes).")
	enableEther := flag.Bool("ether", true, "Also listen for raw ethernet WOL frames (ethertype 0x0842).")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	ports := []int{}
	for _, p := range strings.Split(*udpPorts, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var pi int
		_, err := fmt.Sscanf(p, "%d", &pi)
		if err != nil || pi <= 0 || pi > 65535 {
			logger.Fatalf("invalid port: %q", p)
		}
		ports = append(ports, pi)
	}
	if len(ports) == 0 {
		logger.Fatalf("no udp ports specified")
	}

	subnets, err := wol.ParseCIDRs(*allowSubnets)
	if err != nil {
		logger.Fatalf("invalid allow-subnets: %v", err)
	}

	index := libvirt.NewIndex(*xmlDir, logger)

	// enabled mac cache (reload periodically)
	enabled := wol.LoadEnabledMACs(*enabledMACsFile)
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			enabled = wol.LoadEnabledMACs(*enabledMACsFile)
		}
	}()

	handler := func(ev wol.Event) {
		if len(subnets) > 0 && !wol.SourceAllowed(subnets, ev.SourceIP) {
			logger.Printf("drop: source %s not allowed", ev.SourceIP)
			return
		}

		if len(enabled) > 0 && !enabled[ev.MAC] {
			logger.Printf("wol: mac=%s -> ignored (disabled)", ev.MAC)
			return
		}

		vm, ok := index.FindByMAC(ev.MAC)
		if !ok {
			logger.Printf("wol: mac=%s from=%s -> no matching VM", ev.MAC, ev.SourceIP)
			return
		}

		logger.Printf("wol: mac=%s from=%s -> start VM %q", ev.MAC, ev.SourceIP, vm)
		if err := libvirt.StartVM(vm); err != nil {
			logger.Printf("start failed for %q: %v", vm, err)
		}
	}

	logger.Printf("LibWake starting: iface=%s ports=%v xmlDir=%s", *iface, ports, *xmlDir)

	if err := wol.ListenUDP(ports, handler, logger); err != nil {
		logger.Fatalf("udp listen failed: %v", err)
	}

	if *enableEther {
		_ = wol.ListenEtherType(*iface, 0x0842, handler, logger)
	}

	select {}
}
