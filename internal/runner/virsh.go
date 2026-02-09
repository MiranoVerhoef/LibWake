package runner

import (
  "bytes"
  "context"
  "os/exec"
  "strings"
)

func IsRunning(ctx context.Context, domainName string) bool {
  cmd := exec.CommandContext(ctx, "virsh", "domstate", domainName)
  var out bytes.Buffer
  cmd.Stdout = &out
  cmd.Stderr = &out
  if err := cmd.Run(); err != nil {
    return false
  }
  s := strings.ToLower(strings.TrimSpace(out.String()))
  // Common outputs: "running", "shut off", "paused", "idle"
  return strings.Contains(s, "running") || strings.Contains(s, "idle")
}

func Start(ctx context.Context, domainName string) error {
  cmd := exec.CommandContext(ctx, "virsh", "start", domainName)
  return cmd.Run()
}
