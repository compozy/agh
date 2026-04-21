package store

import (
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func TestSessionLivenessMetaValidate(t *testing.T) {
	t.Parallel()

	t.Run("nil metadata is valid", func(t *testing.T) {
		t.Parallel()

		var meta *SessionLivenessMeta
		if err := meta.Validate(); err != nil {
			t.Fatalf("Validate(nil) error = %v", err)
		}
	})

	t.Run("negative pid is rejected", func(t *testing.T) {
		t.Parallel()

		meta := &SessionLivenessMeta{SubprocessPID: -1}
		if err := meta.Validate(); err == nil {
			t.Fatal("Validate(negative pid) error = nil, want non-nil")
		}
	})

	t.Run("invalid stall state is rejected", func(t *testing.T) {
		t.Parallel()

		meta := &SessionLivenessMeta{StallState: "blocked"}
		if err := meta.Validate(); err == nil {
			t.Fatal("Validate(invalid stall state) error = nil, want non-nil")
		}
	})

	t.Run("stall state requires a reason", func(t *testing.T) {
		t.Parallel()

		meta := &SessionLivenessMeta{StallState: SessionStallStateDetected}
		if err := meta.Validate(); err == nil {
			t.Fatal("Validate(stall state without reason) error = nil, want non-nil")
		}
	})

	t.Run("stall reason requires a state", func(t *testing.T) {
		t.Parallel()

		meta := &SessionLivenessMeta{StallReason: SessionStallReasonActivityTimeout}
		if err := meta.Validate(); err == nil {
			t.Fatal("Validate(stall reason without state) error = nil, want non-nil")
		}
	})

	t.Run("stalled session metadata is valid", func(t *testing.T) {
		t.Parallel()

		meta := &SessionLivenessMeta{
			SubprocessPID: 42,
			StallState:    SessionStallStateDetected,
			StallReason:   SessionStallReasonActivityTimeout,
		}
		if err := meta.Validate(); err != nil {
			t.Fatalf("Validate(valid stalled metadata) error = %v", err)
		}
	})
}

func TestCloneSessionLivenessMeta(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 21, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	lastUpdateAt := startedAt.Add(30 * time.Second)
	meta := &SessionLivenessMeta{
		SubprocessPID:       101,
		SubprocessStartedAt: &startedAt,
		LastUpdateAt:        &lastUpdateAt,
		StallState:          "  " + SessionStallStateDetected + "  ",
		StallReason:         "  " + SessionStallReasonActivityTimeout + "  ",
	}

	cloned := CloneSessionLivenessMeta(meta)
	if cloned == nil {
		t.Fatal("CloneSessionLivenessMeta() = nil, want metadata")
	}
	if cloned == meta {
		t.Fatal("CloneSessionLivenessMeta() returned the original pointer")
	}
	if got := cloned.SubprocessPID; got != meta.SubprocessPID {
		t.Fatalf("cloned.SubprocessPID = %d, want %d", got, meta.SubprocessPID)
	}
	if cloned.SubprocessStartedAt == meta.SubprocessStartedAt {
		t.Fatal("SubprocessStartedAt pointer reused, want deep copy")
	}
	if cloned.LastUpdateAt == meta.LastUpdateAt {
		t.Fatal("LastUpdateAt pointer reused, want deep copy")
	}
	if got := cloned.SubprocessStartedAt.Location(); got != time.UTC {
		t.Fatalf("cloned.SubprocessStartedAt location = %v, want UTC", got)
	}
	if got := cloned.LastUpdateAt.Location(); got != time.UTC {
		t.Fatalf("cloned.LastUpdateAt location = %v, want UTC", got)
	}
	if got := cloned.StallState; got != SessionStallStateDetected {
		t.Fatalf("cloned.StallState = %q, want %q", got, SessionStallStateDetected)
	}
	if got := cloned.StallReason; got != SessionStallReasonActivityTimeout {
		t.Fatalf("cloned.StallReason = %q, want %q", got, SessionStallReasonActivityTimeout)
	}
	if CloneSessionLivenessMeta(nil) != nil {
		t.Fatal("CloneSessionLivenessMeta(nil) != nil, want nil")
	}
}

func TestHookRunQueryValidate(t *testing.T) {
	t.Parallel()

	valid := HookRunQuery{
		SessionID: "sess-1",
		Event:     "session.post_resume",
		Outcome:   hookspkg.HookRunOutcomeApplied,
		Limit:     10,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate(valid query) error = %v", err)
	}

	invalidOutcome := HookRunQuery{
		SessionID: "sess-1",
		Event:     "session.post_resume",
		Outcome:   hookspkg.HookRunOutcome("broken"),
		Limit:     10,
	}
	if err := invalidOutcome.Validate(); err == nil {
		t.Fatal("Validate(invalid outcome) error = nil, want non-nil")
	}

	invalidLimit := HookRunQuery{Limit: -1}
	if err := invalidLimit.Validate(); err == nil {
		t.Fatal("Validate(invalid limit) error = nil, want non-nil")
	}
}
