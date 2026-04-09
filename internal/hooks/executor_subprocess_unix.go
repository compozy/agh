//go:build !windows

package hooks

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureSubprocessCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func terminateSubprocessCommand(cmd *exec.Cmd) error {
	return signalSubprocessCommand(cmd, syscall.SIGTERM)
}

func killSubprocessCommand(cmd *exec.Cmd) error {
	return signalSubprocessCommand(cmd, syscall.SIGKILL)
}

func signalSubprocessCommand(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	return nil
}
