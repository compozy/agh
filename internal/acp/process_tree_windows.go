//go:build windows

package acp

import (
	"errors"
	"os"
	"os/exec"
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
