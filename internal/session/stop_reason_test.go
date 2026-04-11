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
			name:       "Should classify shutdown without error as shutdown",
			cause:      CauseShutdown,
			wantReason: store.StopShutdown,
			wantDetail: "daemon shutdown",
		},
		{
			name:       "Should prefer shutdown over process error",
			cause:      CauseShutdown,
			waitErr:    waitErr,
			wantReason: store.StopShutdown,
			wantDetail: "daemon shutdown",
		},
		{
			name:       "Should classify user requested stop as user canceled",
			cause:      CauseUserRequested,
			wantReason: store.StopUserCanceled,
		},
		{
			name:       "Should classify max iterations subreason",
			cause:      CauseUserRequested,
			detail:     "max_iterations",
			wantReason: store.StopMaxIterations,
			wantDetail: "max_iterations",
		},
		{
			name:       "Should classify loop detected subreason",
			cause:      CauseUserRequested,
			detail:     "loop_detected",
			wantReason: store.StopLoopDetected,
			wantDetail: "loop_detected",
		},
		{
			name:       "Should classify budget exceeded subreason",
			cause:      CauseUserRequested,
			detail:     "budget_exceeded",
			wantReason: store.StopBudgetExceeded,
			wantDetail: "budget_exceeded",
		},
		{
			name:       "Should classify process exit with wait error as agent crashed",
			cause:      CauseProcessExited,
			waitErr:    waitErr,
			wantReason: store.StopAgentCrashed,
			wantDetail: "boom",
		},
		{
			name:       "Should classify process exit without wait error as generic error",
			cause:      CauseProcessExited,
			wantReason: store.StopError,
			wantDetail: "process exited unexpectedly",
		},
		{
			name:       "Should classify completion",
			cause:      CauseCompleted,
			wantReason: store.StopCompleted,
		},
		{
			name:       "Should classify failure",
			cause:      CauseFailed,
			detail:     "automation prompt failed",
			wantReason: store.StopError,
			wantDetail: "automation prompt failed",
		},
		{
			name:       "Should classify hook denial as hook stopped",
			cause:      CauseHookDenied,
			detail:     "reason",
			wantReason: store.StopHookStopped,
			wantDetail: "reason",
		},
		{
			name:       "Should classify unknown cause with error as generic error",
			waitErr:    waitErr,
			wantReason: store.StopError,
			wantDetail: "boom",
		},
		{
			name:       "Should classify unknown cause without error as completed",
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
