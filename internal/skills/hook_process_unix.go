//go:build !windows

package skills

import (
	"errors"
	"os/exec"
	"syscall"
)

func configureHookCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func terminateHookCommand(cmd *exec.Cmd) error {
	return signalHookCommand(cmd, syscall.SIGTERM)
}

func killHookCommand(cmd *exec.Cmd) error {
	return signalHookCommand(cmd, syscall.SIGKILL)
}

func signalHookCommand(cmd *exec.Cmd, sig syscall.Signal) error {
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
