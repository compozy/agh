package store

import (
	"strings"
	"testing"
	"time"
)

func TestValidStopReason(t *testing.T) {
	t.Parallel()

	validReasons := []StopReason{
		StopCompleted,
		StopUserCanceled,
		StopMaxIterations,
		StopLoopDetected,
		StopTimeout,
		StopBudgetExceeded,
		StopError,
		StopAgentCrashed,
		StopHookStopped,
		StopShutdown,
	}

	for _, reason := range validReasons {
		reason := reason
		t.Run("Should validate "+string(reason), func(t *testing.T) {
			t.Parallel()

			if !ValidStopReason(reason) {
				t.Fatalf("ValidStopReason(%q) = false, want true", reason)
			}
		})
	}

	invalidReasons := []StopReason{"", "unknown", " completed "}
	for _, reason := range invalidReasons {
		reason := reason
		name := strings.TrimSpace(string(reason))
		if name == "" {
			name = "empty"
		}
		name = strings.ReplaceAll(name, " ", "_")
		t.Run("Should reject invalid "+name, func(t *testing.T) {
			t.Parallel()

			if ValidStopReason(reason) {
				t.Fatalf("ValidStopReason(%q) = true, want false", reason)
			}
		})
	}
}

func TestSessionMetaValidateStopReason(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC)
	validReason := StopAgentCrashed
	invalidReason := StopReason("invalid")

	tests := []struct {
		name      string
		reason    *StopReason
		wantError bool
	}{
		{name: "Should validate nil stop reason", reason: nil},
		{name: "Should validate supported stop reason", reason: &validReason},
		{name: "Should reject invalid stop reason", reason: &invalidReason, wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta := SessionMeta{
				ID:          "sess-meta",
				AgentName:   "coder",
				WorkspaceID: "ws-meta",
				State:       "stopped",
				StopReason:  tt.reason,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			err := meta.Validate()
			if tt.wantError && err == nil {
				t.Fatal("Validate() error = nil, want non-nil")
			}
			if tt.wantError && !strings.Contains(err.Error(), "invalid session stop reason") {
				t.Fatalf("Validate() error = %v, want to contain %q", err, "invalid session stop reason")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
