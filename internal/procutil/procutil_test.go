package procutil

import (
	"os"
	"syscall"
	"testing"
)

func TestAliveCurrentProcess(t *testing.T) {
	t.Parallel()

	if !Alive(os.Getpid()) {
		t.Fatal("Alive(current pid) = false, want true")
	}
}

func TestAliveRejectsNonPositivePIDs(t *testing.T) {
	t.Parallel()

	testCases := []int{0, -1}
	for _, pid := range testCases {
		pid := pid
		t.Run("pid", func(t *testing.T) {
			t.Parallel()
			if Alive(pid) {
				t.Fatalf("Alive(%d) = true, want false", pid)
			}
		})
	}
}

func TestSignalCurrentProcessWithSignalZero(t *testing.T) {
	t.Parallel()

	if err := Signal(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Fatalf("Signal(current pid, 0) error = %v, want nil", err)
	}
}

func TestSignalRejectsNonPositivePID(t *testing.T) {
	t.Parallel()

	if err := Signal(0, syscall.SIGTERM); err == nil {
		t.Fatal("Signal(0, SIGTERM) error = nil, want non-nil")
	}
}

func TestSignalReturnsErrorForMissingProcess(t *testing.T) {
	t.Parallel()

	if err := Signal(999999, syscall.Signal(0)); err == nil {
		t.Fatal("Signal(missing pid, 0) error = nil, want non-nil")
	}
}
