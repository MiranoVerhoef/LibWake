package libvirt

import (
	"encoding/xml"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type domain struct {
	Name    string `xml:"name"`
	Devices struct {
		Interfaces []struct {
			MAC struct {
				Address string `xml:"address,attr"`
			} `xml:"mac"`
		} `xml:"interface"`
	} `xml:"devices"`
}

type Index struct {
	dir    string
	logger *log.Logger

	mu    sync.RWMutex
	byMAC map[string]string
}

func NewIndex(xmlDir string, logger *log.Logger) *Index {
	if logger == nil {
		logger = log.Default()
	}
	x := &Index{
		dir:    xmlDir,
		logger: logger,
		byMAC:  map[string]string{},
	}
	x.refresh()
	go x.loop()
	return x
}

func normalizeMAC(mac string) string {
	mac = strings.ToLower(strings.TrimSpace(mac))
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}

func (x *Index) loop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		x.refresh()
	}
}

func (x *Index) refresh() {
	tmp := map[string]string{}
	_ = filepath.WalkDir(x.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".xml") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var dom domain
		if err := xml.Unmarshal(b, &dom); err != nil {
			return nil
		}
		if dom.Name == "" {
			return nil
		}
		for _, itf := range dom.Devices.Interfaces {
			m := normalizeMAC(itf.MAC.Address)
			if m != "" {
				tmp[m] = dom.Name
			}
		}
		return nil
	})

	x.mu.Lock()
	x.byMAC = tmp
	x.mu.Unlock()
}

func (x *Index) FindByMAC(mac string) (string, bool) {
	m := normalizeMAC(mac)
	x.mu.RLock()
	defer x.mu.RUnlock()
	v, ok := x.byMAC[m]
	return v, ok
}
