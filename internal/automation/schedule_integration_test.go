//go:build integration

package automation

import (
	"context"
	"testing"
	"time"

	"github.com/compozy/agh/internal/testutil"
)

func TestSchedulerIntegrationFastScheduleDispatchesThroughDispatcher(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openAutomationIntegrationDB(t, ctx)
	creator := newRecordingSessionCreator()
	dispatcher := newTestDispatcher(t, creator, db)
	scheduler := newTestScheduler(t, dispatcher)

	job, err := db.CreateJob(ctx, testJob(AutomationScopeGlobal, "integration-schedule", ""))
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	job.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "1s",
	}

	if _, err := scheduler.Register(ctx, job); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	waitUntil(t, 4*time.Second, 25*time.Millisecond, func() bool {
		runs, err := db.ListRuns(ctx, RunQuery{JobID: job.ID})
		if err != nil {
			t.Fatalf("ListRuns() error = %v", err)
		}
		return len(runs) > 0 && runs[0].Status == RunCompleted
	})

	runs, err := db.ListRuns(ctx, RunQuery{JobID: job.ID})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("ListRuns() = 0 runs, want at least one")
	}
	if got, want := runs[0].Status, RunCompleted; got != want {
		t.Fatalf("runs[0].Status = %q, want %q", got, want)
	}
	if got := len(creator.createCalls()); got == 0 {
		t.Fatal("Create() call count = 0, want at least one")
	}
}

func TestSchedulerIntegrationShutdownCancelsInflightDispatch(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db := openAutomationIntegrationDB(t, ctx)

	promptStarted := make(chan struct{}, 1)
	promptRelease := make(chan struct{})
	creator := newRecordingSessionCreator(sessionAttemptPlan{
		promptStarted: promptStarted,
		promptRelease: promptRelease,
	})
	dispatcher := newTestDispatcher(t, creator, db)
	scheduler := newTestScheduler(t, dispatcher)

	job, err := db.CreateJob(ctx, testJob(AutomationScopeGlobal, "integration-shutdown", ""))
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	job.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "1s",
	}

	if _, err := scheduler.Register(ctx, job); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	select {
	case <-promptStarted:
	case <-time.After(4 * time.Second):
		t.Fatal("scheduled dispatch did not reach Prompt() in time")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := scheduler.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	waitUntil(t, 4*time.Second, 25*time.Millisecond, func() bool {
		runs, err := db.ListRuns(ctx, RunQuery{JobID: job.ID})
		if err != nil {
			t.Fatalf("ListRuns() error = %v", err)
		}
		return len(runs) > 0 && runs[0].Status == RunCancelled
	})

	runs, err := db.ListRuns(ctx, RunQuery{JobID: job.ID})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("ListRuns() = 0 runs, want at least one")
	}
	if got, want := runs[0].Status, RunCancelled; got != want {
		t.Fatalf("runs[0].Status = %q, want %q", got, want)
	}
	if runs[0].StartedAt == nil {
		t.Fatal("runs[0].StartedAt = nil, want populated")
	}
	if runs[0].EndedAt == nil {
		t.Fatal("runs[0].EndedAt = nil, want populated")
	}
}

func waitUntil(t *testing.T, timeout time.Duration, interval time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if fn() {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("condition not met within %s", timeout)
		case <-ticker.C:
		}
	}
}
