//go:build windows

package hooks

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

func configureSubprocessCommand(_ *exec.Cmd) {}

func terminateSubprocessCommand(cmd *exec.Cmd) error {
	return signalSubprocessCommand(cmd, os.Kill)
}

func killSubprocessCommand(cmd *exec.Cmd) error {
	return signalSubprocessCommand(cmd, os.Kill)
}

func signalSubprocessCommand(cmd *exec.Cmd, sig os.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Signal(sig); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	return nil
}

// Windows keeps an explicit no-op fallback until process-group parity lands for
// hook subprocesses.
func forceSubprocessCommandExit(_ *exec.Cmd, _ time.Duration) error {
	return nil
}
