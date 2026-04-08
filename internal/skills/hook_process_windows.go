//go:build windows

package skills

import (
	"errors"
	"os"
	"os/exec"
)

func configureHookCommand(_ *exec.Cmd) {}

func terminateHookCommand(cmd *exec.Cmd) error {
	return signalHookCommand(cmd, os.Kill)
}

func killHookCommand(cmd *exec.Cmd) error {
	return signalHookCommand(cmd, os.Kill)
}

func signalHookCommand(cmd *exec.Cmd, sig os.Signal) error {
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
