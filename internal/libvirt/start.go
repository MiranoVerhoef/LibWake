package libvirt

import (
	"bytes"
	"os/exec"
)

func StartVM(name string) error {
	// Use virsh (available on Unraid) to start the domain.
	cmd := exec.Command("virsh", "start", name)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return &ExecError{Err: err, Output: out.String()}
	}
	return nil
}

type ExecError struct {
	Err    error
	Output string
}

func (e *ExecError) Error() string {
	return e.Err.Error() + ": " + e.Output
}
