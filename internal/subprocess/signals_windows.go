//go:build windows

package subprocess

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

func configureManagedCommand(_ *exec.Cmd) {}

func terminateManagedProcess(cmd *exec.Cmd) error {
	return signalManagedProcess(cmd, os.Kill)
}

func killManagedProcess(cmd *exec.Cmd) error {
	return signalManagedProcess(cmd, os.Kill)
}

func signalManagedProcess(cmd *exec.Cmd, sig os.Signal) error {
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

// Windows does not yet provide process-group parity for managed subprocesses in
// this phase. Keep the fallback explicit and compile-safe instead of implying
// Unix-equivalent behavior.
func forceManagedProcessGroupExit(_ *exec.Cmd, _ time.Duration) error {
	return nil
}
