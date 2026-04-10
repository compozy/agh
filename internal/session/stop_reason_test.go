package session

import (
	"errors"
	"testing"

	"github.com/pedronauck/agh/internal/store"
)

func TestClassifyStopReason(t *testing.T) {
	t.Parallel()

	waitErr := errors.New("boom")
	tests := []struct {
		name       string
		cause      StopCause
		waitErr    error
		detail     string
		wantReason store.StopReason
		wantDetail string
	}{
		{
			name:       "shutdown without error",
			cause:      CauseShutdown,
			wantReason: store.StopShutdown,
			wantDetail: "daemon shutdown",
		},
		{
			name:       "shutdown wins over error",
			cause:      CauseShutdown,
			waitErr:    waitErr,
			wantReason: store.StopShutdown,
			wantDetail: "daemon shutdown",
		},
		{
			name:       "user canceled",
			cause:      CauseUserRequested,
			wantReason: store.StopUserCanceled,
		},
		{
			name:       "max iterations",
			cause:      CauseUserRequested,
			detail:     "max_iterations",
			wantReason: store.StopMaxIterations,
			wantDetail: "max_iterations",
		},
		{
			name:       "loop detected",
			cause:      CauseUserRequested,
			detail:     "loop_detected",
			wantReason: store.StopLoopDetected,
			wantDetail: "loop_detected",
		},
		{
			name:       "budget exceeded",
			cause:      CauseUserRequested,
			detail:     "budget_exceeded",
			wantReason: store.StopBudgetExceeded,
			wantDetail: "budget_exceeded",
		},
		{
			name:       "process exited with error",
			cause:      CauseProcessExited,
			waitErr:    waitErr,
			wantReason: store.StopAgentCrashed,
			wantDetail: "boom",
		},
		{
			name:       "process exited without wait error",
			cause:      CauseProcessExited,
			wantReason: store.StopError,
			wantDetail: "process exited unexpectedly",
		},
		{
			name:       "completed",
			cause:      CauseCompleted,
			wantReason: store.StopCompleted,
		},
		{
			name:       "hook denied",
			cause:      CauseHookDenied,
			detail:     "reason",
			wantReason: store.StopHookStopped,
			wantDetail: "reason",
		},
		{
			name:       "fallback error",
			waitErr:    waitErr,
			wantReason: store.StopError,
			wantDetail: "boom",
		},
		{
			name:       "fallback completed",
			wantReason: store.StopCompleted,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotReason, gotDetail := classifyStopReason(tc.cause, tc.waitErr, tc.detail)
			if gotReason != tc.wantReason {
				t.Fatalf("classifyStopReason() reason = %q, want %q", gotReason, tc.wantReason)
			}
			if gotDetail != tc.wantDetail {
				t.Fatalf("classifyStopReason() detail = %q, want %q", gotDetail, tc.wantDetail)
			}
		})
	}
}
