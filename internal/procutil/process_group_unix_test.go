//go:build !windows

package procutil

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestJoinProcessGroupKillResultSuppressesEPERMWhenWaitSucceeds(t *testing.T) {
	t.Parallel()

	signalErr := fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM)
	if err := joinProcessGroupKillResult(signalErr, nil); err != nil {
		t.Fatalf("joinProcessGroupKillResult(EPERM, nil) error = %v, want nil", err)
	}
}

func TestJoinProcessGroupKillResultPreservesWaitFailure(t *testing.T) {
	t.Parallel()

	waitErr := errors.New("wait for process group exit: deadline exceeded")
	signalErr := fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM)

	err := joinProcessGroupKillResult(signalErr, waitErr)
	if !errors.Is(err, waitErr) {
		t.Fatalf("joinProcessGroupKillResult(EPERM, waitErr) = %v, want wrapped waitErr", err)
	}
}

func TestJoinProcessGroupKillResultPreservesNonEPERMSignalFailure(t *testing.T) {
	t.Parallel()

	signalErr := fmt.Errorf("signal process group members: %w", syscall.ESRCH)
	err := joinProcessGroupKillResult(signalErr, nil)
	if !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("joinProcessGroupKillResult(ESRCH, nil) = %v, want wrapped ESRCH", err)
	}
}
