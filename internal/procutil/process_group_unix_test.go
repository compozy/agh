//go:build !windows

package procutil

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
)

func TestJoinProcessGroupKillResult(t *testing.T) {
	t.Parallel()

	waitErr := errors.New("wait for process group exit: deadline exceeded")
	testCases := []struct {
		name      string
		signalErr error
		waitErr   error
		wantNil   bool
		wantIs    error
	}{
		{
			name:      "Should suppress EPERM when wait succeeds",
			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
			wantNil:   true,
		},
		{
			name:      "Should preserve wait failure when signal returns EPERM",
			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
			waitErr:   waitErr,
			wantIs:    waitErr,
		},
		{
			name:      "Should preserve non-EPERM signal failure",
			signalErr: fmt.Errorf("signal process group members: %w", syscall.ESRCH),
			wantIs:    syscall.ESRCH,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := joinProcessGroupKillResult(tc.signalErr, tc.waitErr)
			if tc.wantNil {
				if err != nil {
					t.Fatalf("joinProcessGroupKillResult() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("joinProcessGroupKillResult() error = %v, want wrapped %v", err, tc.wantIs)
			}
		})
	}
}
