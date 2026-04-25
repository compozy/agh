//go:build windows

package procutil

import (
	"errors"
	"fmt"
	"syscall"
	"time"
)

var errWindowsProcessGroupUnsupported = errors.New("procutil: windows process groups are unsupported")

// SignalProcessGroupID returns an explicit unsupported error on Windows until
// process-group parity is implemented for Windows runtimes.
func SignalProcessGroupID(pgid int, sig syscall.Signal) error {
	if pgid <= 0 {
		return fmt.Errorf("procutil: invalid process group id %d", pgid)
	}
	return fmt.Errorf("signal process group (pid %d, sig %v): %w", pgid, sig, errWindowsProcessGroupUnsupported)
}

// WaitForProcessGroupIDExit returns an explicit unsupported error on Windows.
func WaitForProcessGroupIDExit(pgid int, _ time.Duration) error {
	if pgid <= 0 {
		return fmt.Errorf("procutil: invalid process group id %d", pgid)
	}
	return fmt.Errorf("wait for process group exit (pid %d): %w", pgid, errWindowsProcessGroupUnsupported)
}

// KillProcessGroupIDAndWait returns an explicit unsupported error on Windows.
func KillProcessGroupIDAndWait(pgid int, _ time.Duration) error {
	if pgid <= 0 {
		return fmt.Errorf("procutil: invalid process group id %d", pgid)
	}
	return fmt.Errorf("kill process group and wait (pid %d): %w", pgid, errWindowsProcessGroupUnsupported)
}
