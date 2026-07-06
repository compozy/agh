package globaldb

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/testutil"
)

func TestGlobalDBLoopConfigShouldPersistOverrides(t *testing.T) {
	t.Parallel()

	t.Run("Should round trip loop config by workspace and loop name", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		humanGate := true
		reattempt := looppkg.ReattemptFullBody
		onExceeded := dsl.BudgetExceededEscalate

		err := globalDB.UpsertLoopConfig(ctx, "ws-1", "delivery", looppkg.LoopConfig{
			HumanGateEnabled:  &humanGate,
			ReattemptStrategy: &reattempt,
			EnabledChecks:     []byte(`{"command":true}`),
			IterationCap:      new(11),
			BudgetTokens:      new(2000),
			BudgetWallSec:     new(300),
			BudgetOnExceeded:  &onExceeded,
			NoProgressWindow:  new(4),
			FanOutWidth:       new(5),
			GateMaxRevisions:  new(6),
		})
		if err != nil {
			t.Fatalf("UpsertLoopConfig() error = %v", err)
		}

		got, err := globalDB.GetLoopConfig(ctx, "ws-1", "delivery")
		if err != nil {
			t.Fatalf("GetLoopConfig() error = %v", err)
		}
		if got.HumanGateEnabled == nil || !*got.HumanGateEnabled {
			t.Fatalf("HumanGateEnabled = %#v, want true", got.HumanGateEnabled)
		}
		if got.ReattemptStrategy == nil || *got.ReattemptStrategy != looppkg.ReattemptFullBody {
			t.Fatalf("ReattemptStrategy = %#v, want full_body", got.ReattemptStrategy)
		}
		if string(got.EnabledChecks) != `{"command":true}` {
			t.Fatalf("EnabledChecks = %s, want command check", got.EnabledChecks)
		}
		if got.FanOutWidth == nil || *got.FanOutWidth != 5 {
			t.Fatalf("FanOutWidth = %#v, want 5", got.FanOutWidth)
		}
		_, err = globalDB.GetLoopConfig(ctx, "ws-2", "delivery")
		if !errors.Is(err, looppkg.ErrConfigNotFound) {
			t.Fatalf("GetLoopConfig(other workspace) error = %v, want ErrConfigNotFound", err)
		}
	})
}

func TestGlobalDBLoopRunStatusShouldUseCompareAndSwap(t *testing.T) {
	t.Parallel()

	t.Run("Should allow only one concurrent transition from the same status", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 4, 14, 0, 0, 0, time.UTC)
		run := testLoopRun("looprun-cas", now, looppkg.StatusRunning)
		created, err := globalDB.CreateLoopRunForStart(ctx, run, dsl.ConcurrencyAllow)
		if err != nil {
			t.Fatalf("CreateLoopRunForStart() error = %v", err)
		}
		if created.Status != looppkg.StatusRunning {
			t.Fatalf("CreateLoopRunForStart() status = %s, want running", created.Status)
		}

		attempts := make([]error, 8)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(len(attempts))
		for idx := range attempts {
			go func(idx int) {
				defer wg.Done()
				<-start
				attempts[idx] = globalDB.CompareAndSwapLoopRunStatus(
					context.Background(),
					run.ID,
					looppkg.StatusRunning,
					looppkg.StatusPaused,
					looppkg.TransitionCausePauseBoundary,
					now.Add(time.Duration(idx)*time.Millisecond),
				)
			}(idx)
		}
		close(start)
		wg.Wait()

		wins := 0
		conflicts := 0
		for idx, err := range attempts {
			if err == nil {
				wins++
				continue
			}
			if errors.Is(err, looppkg.ErrTransitionConflict) {
				conflicts++
				continue
			}
			t.Fatalf("attempt %d error = %v, want nil or ErrTransitionConflict", idx, err)
		}
		if wins != 1 {
			t.Fatalf("wins = %d, want 1", wins)
		}
		if conflicts != len(attempts)-1 {
			t.Fatalf("conflicts = %d, want %d", conflicts, len(attempts)-1)
		}
		stored, err := globalDB.GetLoopRun(ctx, "ws-1", run.ID)
		if err != nil {
			t.Fatalf("GetLoopRun() error = %v", err)
		}
		if stored.Status != looppkg.StatusPaused {
			t.Fatalf("stored status = %q, want paused", stored.Status)
		}
		if got, want := stored.IterationCap, run.IterationCap; got != want {
			t.Fatalf("stored iteration cap = %d, want %d", got, want)
		}
		eventCount := countLoopRunEvents(ctx, t, globalDB, run.ID)
		if eventCount != 2 {
			t.Fatalf("status event count = %d, want create + transition events", eventCount)
		}
	})
}

func TestGlobalDBLoopRunCreateShouldApplyConcurrencyPolicyAtomically(t *testing.T) {
	t.Parallel()

	t.Run("Should allow only one concurrent forbid start", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 4, 14, 15, 0, 0, time.UTC)
		attempts := make([]error, 8)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(len(attempts))
		for idx := range attempts {
			go func(idx int) {
				defer wg.Done()
				<-start
				run := testLoopRun(
					"looprun-forbid-"+time.Duration(idx).String(),
					now.Add(time.Duration(idx)*time.Millisecond),
					looppkg.StatusRunning,
				)
				_, attempts[idx] = globalDB.CreateLoopRunForStart(
					context.Background(),
					run,
					dsl.ConcurrencyForbid,
				)
			}(idx)
		}
		close(start)
		wg.Wait()

		wins := 0
		conflicts := 0
		for idx, err := range attempts {
			if err == nil {
				wins++
				continue
			}
			if errors.Is(err, looppkg.ErrConcurrencyConflict) {
				conflicts++
				continue
			}
			t.Fatalf("attempt %d error = %v, want nil or ErrConcurrencyConflict", idx, err)
		}
		if wins != 1 {
			t.Fatalf("wins = %d, want 1", wins)
		}
		if conflicts != len(attempts)-1 {
			t.Fatalf("conflicts = %d, want %d", conflicts, len(attempts)-1)
		}
		running := countLoopRunsByStatus(ctx, t, globalDB, "ws-1", "delivery", looppkg.StatusRunning)
		if running != 1 {
			t.Fatalf("running loop_runs = %d, want 1", running)
		}
	})

	t.Run("Should queue concurrent queue starts after the first running run", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 4, 14, 20, 0, 0, time.UTC)
		attempts := make([]error, 8)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(len(attempts))
		for idx := range attempts {
			go func(idx int) {
				defer wg.Done()
				<-start
				run := testLoopRun(
					"looprun-queue-"+time.Duration(idx).String(),
					now.Add(time.Duration(idx)*time.Millisecond),
					looppkg.StatusRunning,
				)
				_, attempts[idx] = globalDB.CreateLoopRunForStart(
					context.Background(),
					run,
					dsl.ConcurrencyQueue,
				)
			}(idx)
		}
		close(start)
		wg.Wait()

		for idx, err := range attempts {
			if err != nil {
				t.Fatalf("attempt %d error = %v, want nil", idx, err)
			}
		}
		running := countLoopRunsByStatus(ctx, t, globalDB, "ws-1", "delivery", looppkg.StatusRunning)
		if running != 1 {
			t.Fatalf("running loop_runs = %d, want 1", running)
		}
		queued := countLoopRunsByStatus(ctx, t, globalDB, "ws-1", "delivery", looppkg.StatusQueued)
		if queued != len(attempts)-1 {
			t.Fatalf("queued loop_runs = %d, want %d", queued, len(attempts)-1)
		}
	})
}

func testLoopRun(id string, at time.Time, status looppkg.Status) looppkg.Run {
	return looppkg.Run{
		ID:                looppkg.RunID(id),
		WorkspaceID:       "ws-1",
		LoopName:          "delivery",
		Status:            status,
		ReattemptStrategy: looppkg.ReattemptFailedOnly,
		CreatedAt:         at,
		LastProgressAt:    at,
		IterationCap:      7,
		BudgetOnExceeded:  dsl.BudgetExceededHalt,
		Inputs:            map[string]any{"tasks": "task-ref"},
	}
}

func countLoopRunsByStatus(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	workspaceID looppkg.WorkspaceID,
	loopName string,
	status looppkg.Status,
) int {
	t.Helper()
	var count int
	if err := globalDB.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM loop_runs WHERE workspace_id = ? AND loop_name = ? AND status = ?`,
		string(workspaceID),
		loopName,
		string(status),
	).Scan(&count); err != nil {
		t.Fatalf("count loop_runs by status error = %v", err)
	}
	return count
}

func countLoopRunEvents(
	ctx context.Context,
	t *testing.T,
	globalDB *GlobalDB,
	runID looppkg.RunID,
) int {
	t.Helper()
	var count int
	if err := globalDB.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM loop_run_events WHERE loop_run_id = ?`,
		string(runID),
	).Scan(&count); err != nil {
		t.Fatalf("count loop_run_events error = %v", err)
	}
	return count
}
