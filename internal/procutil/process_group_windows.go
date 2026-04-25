//go:build windows

package procutil

import (
	"syscall"
	"time"
)

// SignalProcessGroupID falls back to signaling the process on Windows until
// process-group parity is implemented for Windows runtimes.
func SignalProcessGroupID(pgid int, sig syscall.Signal) error {
	return Signal(pgid, sig)
}

// WaitForProcessGroupIDExit is a compile-safe no-op on Windows.
func WaitForProcessGroupIDExit(_ int, _ time.Duration) error {
	return nil
}

// KillProcessGroupIDAndWait falls back to process termination on Windows.
func KillProcessGroupIDAndWait(pgid int, _ time.Duration) error {
	return Signal(pgid, syscall.SIGKILL)
}
