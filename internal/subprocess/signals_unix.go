//go:build !windows

package subprocess

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureManagedCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func terminateManagedProcess(cmd *exec.Cmd) error {
	return signalManagedProcess(cmd, syscall.SIGTERM)
}

func killManagedProcess(cmd *exec.Cmd) error {
	return signalManagedProcess(cmd, syscall.SIGKILL)
}

func signalManagedProcess(cmd *exec.Cmd, sig syscall.Signal) error {
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
