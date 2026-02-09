package config

import (
  "bufio"
  "encoding/json"
  "fmt"
  "net"
  "os"
  "path/filepath"
  "strconv"
  "strings"
)

type Config struct {
  Enabled          bool
  Interface        string
  UDPPorts         []int
  QemuDir          string
  VMStatePath      string // JSON map UUID->bool
  AllowSubnetsRaw  []string
  allowSubnets     []*net.IPNet
  DebounceSeconds  int
}

type VMState map[string]bool

func Defaults() Config {
  return Config{
    Enabled:         false,
    Interface:       "br0",
    UDPPorts:        []int{7, 9},
    QemuDir:         "/etc/libvirt/qemu",
    VMStatePath:     "/boot/config/plugins/libwake/vms.json",
    AllowSubnetsRaw: nil,
    DebounceSeconds: 10,
  }
}

func Load(cfgPath string) (Config, VMState, error) {
  cfg := Defaults()

  kv := map[string]string{}
  loaded, err := loadKV(cfgPath)
  if err != nil {
    // If the file doesn't exist, we still return defaults.
    if !os.IsNotExist(err) {
      return cfg, VMState{}, err
    }
  } else {
    kv = loaded
  }

  if v := kv["ENABLED"]; v != "" {
    cfg.Enabled = parseBool(v)
  }
  if v := kv["INTERFACE"]; v != "" {
    cfg.Interface = v
  }
  if v := kv["UDP_PORTS"]; v != "" {
    ports, err := parsePorts(v)
    if err != nil {
      return cfg, VMState{}, fmt.Errorf("UDP_PORTS: %w", err)
    }
    cfg.UDPPorts = ports
  }
  if v := kv["QEMU_DIR"]; v != "" {
    cfg.QemuDir = v
  }
  if v := kv["VM_STATE_PATH"]; v != "" {
    cfg.VMStatePath = v
  }
  if v := kv["ALLOW_SUBNETS"]; v != "" {
    cfg.AllowSubnetsRaw = splitCSV(v)
  }
  if v := kv["DEBOUNCE_SECONDS"]; v != "" {
    if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
      cfg.DebounceSeconds = n
    }
  }

  cfg.allowSubnets = nil
  for _, raw := range cfg.AllowSubnetsRaw {
    raw = strings.TrimSpace(raw)
    if raw == "" {
      continue
    }
    _, netw, err := net.ParseCIDR(raw)
    if err != nil {
      return cfg, VMState{}, fmt.Errorf("ALLOW_SUBNETS entry %q: %w", raw, err)
    }
    cfg.allowSubnets = append(cfg.allowSubnets, netw)
  }

  vms, err := loadVMState(cfg.VMStatePath)
  if err != nil {
    if !os.IsNotExist(err) {
      return cfg, VMState{}, err
    }
    vms = VMState{}
  }

  return cfg, vms, nil
}

func (c Config) SrcAllowed(ip net.IP) bool {
  if len(c.allowSubnets) == 0 {
    return true
  }
  for _, n := range c.allowSubnets {
    if n.Contains(ip) {
      return true
    }
  }
  return false
}

func loadKV(path string) (map[string]string, error) {
  fh, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer fh.Close()

  m := map[string]string{}
  sc := bufio.NewScanner(fh)
  for sc.Scan() {
    line := strings.TrimSpace(sc.Text())
    if line == "" {
      continue
    }
    if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
      continue
    }
    // Accept KEY=value or KEY="value".
    idx := strings.Index(line, "=")
    if idx <= 0 {
      continue
    }
    key := strings.TrimSpace(line[:idx])
    val := strings.TrimSpace(line[idx+1:])
    val = strings.Trim(val, "\"'")
    m[key] = val
  }
  if err := sc.Err(); err != nil {
    return nil, err
  }
  return m, nil
}

func parseBool(s string) bool {
  s = strings.ToLower(strings.TrimSpace(strings.Trim(s, "\"'")))
  switch s {
  case "1", "true", "yes", "on", "enable", "enabled":
    return true
  default:
    return false
  }
}

func parsePorts(s string) ([]int, error) {
  parts := splitCSV(s)
  if len(parts) == 0 {
    return nil, fmt.Errorf("empty")
  }
  out := make([]int, 0, len(parts))
  for _, p := range parts {
    n, err := strconv.Atoi(strings.TrimSpace(p))
    if err != nil || n <= 0 || n > 65535 {
      return nil, fmt.Errorf("invalid port: %q", p)
    }
    out = append(out, n)
  }
  return out, nil
}

func splitCSV(s string) []string {
  parts := strings.Split(s, ",")
  out := make([]string, 0, len(parts))
  for _, p := range parts {
    p = strings.TrimSpace(strings.Trim(p, "\"'"))
    if p != "" {
      out = append(out, p)
    }
  }
  return out
}

func loadVMState(path string) (VMState, error) {
  b, err := os.ReadFile(path)
  if err != nil {
    return nil, err
  }
  var m VMState
  if err := json.Unmarshal(b, &m); err != nil {
    return nil, err
  }
  if m == nil {
    m = VMState{}
  }
  return m, nil
}

func SaveVMState(path string, state VMState) error {
  dir := filepath.Dir(path)
  if err := os.MkdirAll(dir, 0o755); err != nil {
    return err
  }
  b, err := json.MarshalIndent(state, "", "  ")
  if err != nil {
    return err
  }
  return os.WriteFile(path, b, 0o644)
}
