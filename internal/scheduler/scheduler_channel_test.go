package scheduler

import (
	"testing"
	"time"

	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
	"github.com/jonboulle/clockwork"
)

func TestRunOnceHonorsCoordinationChannel(t *testing.T) {
	t.Parallel()

	t.Run("Should wake only same-channel sessions for channel-bound work", func(t *testing.T) {
		t.Parallel()

		base := time.Date(2026, 4, 26, 13, 30, 0, 0, time.UTC)
		work := workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", []string{"go"}, base)
		work.Run.CoordinationChannelID = "finance"
		source := &fakeTaskSource{pending: []RunSnapshot{work}}
		sessions := &fakeSessionSource{sessions: []SessionSnapshot{
			{
				ID:           "sess-marketing",
				WorkspaceID:  "ws-1",
				Channel:      "marketing",
				State:        "active",
				Capabilities: []string{"go"},
				CreatedAt:    base,
			},
			{
				ID:           "sess-finance",
				WorkspaceID:  "ws-1",
				Channel:      "finance",
				State:        "active",
				Capabilities: []string{"go"},
				CreatedAt:    base.Add(time.Second),
			},
		}}
		waker := &fakeWaker{}
		scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

		result, err := scheduler.RunOnce(testutil.Context(t))
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.WakeAttempts != 1 || result.WakeSucceeded != 1 {
			t.Fatalf("result = %#v, want one successful wake", result)
		}

		targets := waker.targetsSnapshot()
		if got, want := len(targets), 1; got != want {
			t.Fatalf("wake targets = %d, want %d", got, want)
		}
		if got, want := targets[0].Session.ID, "sess-finance"; got != want {
			t.Fatalf("woken session = %q, want %q", got, want)
		}
	})

	t.Run("Should record no match when only wrong-channel sessions are available", func(t *testing.T) {
		t.Parallel()

		base := time.Date(2026, 4, 26, 13, 45, 0, 0, time.UTC)
		work := workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", []string{"go"}, base)
		work.Run.CoordinationChannelID = "finance"
		source := &fakeTaskSource{pending: []RunSnapshot{work}}
		sessions := &fakeSessionSource{sessions: []SessionSnapshot{
			{
				ID:           "sess-marketing",
				WorkspaceID:  "ws-1",
				Channel:      "marketing",
				State:        "active",
				Capabilities: []string{"go"},
				CreatedAt:    base,
			},
		}}
		waker := &fakeWaker{}
		scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

		result, err := scheduler.RunOnce(testutil.Context(t))
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.WakeAttempts != 0 || result.NoMatchRuns != 1 {
			t.Fatalf("result = %#v, want one no-match and no wake attempts", result)
		}
		if got := len(waker.targetsSnapshot()); got != 0 {
			t.Fatalf("wake targets = %d, want 0", got)
		}
	})

	t.Run(
		"Should record no match when only unscoped sessions are available for channel-bound work",
		func(t *testing.T) {
			t.Parallel()

			base := time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
			work := workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", []string{"go"}, base)
			work.Run.CoordinationChannelID = "finance"
			source := &fakeTaskSource{pending: []RunSnapshot{work}}
			sessions := &fakeSessionSource{sessions: []SessionSnapshot{
				{
					ID:           "sess-unscoped",
					WorkspaceID:  "ws-1",
					Channel:      "",
					State:        "active",
					Capabilities: []string{"go"},
					CreatedAt:    base,
				},
			}}
			waker := &fakeWaker{}
			scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

			result, err := scheduler.RunOnce(testutil.Context(t))
			if err != nil {
				t.Fatalf("RunOnce() error = %v", err)
			}
			if result.WakeAttempts != 0 || result.NoMatchRuns != 1 {
				t.Fatalf("result = %#v, want one no-match and no wake attempts", result)
			}
			if got := len(waker.targetsSnapshot()); got != 0 {
				t.Fatalf("wake targets = %d, want 0", got)
			}
		},
	)
}
