package globaldb

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/testutil"
)

func TestLoopRunEventKindValidShouldMatchPublicContract(t *testing.T) {
	t.Parallel()

	localKinds := map[string]struct{}{
		loopRunEventNodeRunning:       {},
		loopRunEventNodeSucceeded:     {},
		loopRunEventNodeFailed:        {},
		loopRunEventGateVerdict:       {},
		loopRunEventGenerationStarted: {},
		loopRunEventChannelMsg:        {},
		loopRunEventTokenTick:         {},
		loopRunEventNeedsApproval:     {},
		loopRunEventStatusChanged:     {},
	}
	for _, kind := range contract.LoopRunEventKindValues() {
		t.Run("Should accept public kind "+kind, func(t *testing.T) {
			t.Parallel()
			if !loopRunEventKindValid(kind) {
				t.Fatalf("loopRunEventKindValid(%q) = false, want true", kind)
			}
			if _, ok := localKinds[kind]; !ok {
				t.Fatalf("contract kind %q is missing from local loop event constants", kind)
			}
		})
	}
	for kind := range localKinds {
		t.Run("Should publish local kind "+kind, func(t *testing.T) {
			t.Parallel()
			if !slices.Contains(contract.LoopRunEventKindValues(), kind) {
				t.Fatalf("local loop event kind %q is missing from public contract", kind)
			}
		})
	}
}

func TestGlobalDBLoopConfigShouldPersistOverrides(t *testing.T) {
	t.Parallel()

	t.Run("Should round trip loop config by workspace and loop name", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		humanGate := true
		reattempt := looppkg.ReattemptFullBody
		onExceeded := dsl.BudgetExceededEscalate
		workerModel := "stored-worker"
		judgeModel := "stored-judge"

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
			ModelDefaults: &looppkg.ModelDefaults{
				Worker: &workerModel,
				Judge:  &judgeModel,
			},
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
		if got.ModelDefaults == nil {
			t.Fatal("ModelDefaults = nil, want stored defaults")
		}
		if got.ModelDefaults.Worker == nil || *got.ModelDefaults.Worker != "stored-worker" {
			t.Fatalf("ModelDefaults.Worker = %#v, want stored-worker", got.ModelDefaults.Worker)
		}
		if got.ModelDefaults.Judge == nil || *got.ModelDefaults.Judge != "stored-judge" {
			t.Fatalf("ModelDefaults.Judge = %#v, want stored-judge", got.ModelDefaults.Judge)
		}
		_, err = globalDB.GetLoopConfig(ctx, "ws-2", "delivery")
		if !errors.Is(err, looppkg.ErrConfigNotFound) {
			t.Fatalf("GetLoopConfig(other workspace) error = %v, want ErrConfigNotFound", err)
		}
	})

	t.Run("Should reject empty loop config keys", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		cases := []struct {
			name     string
			ws       looppkg.WorkspaceID
			loopName string
			want     string
		}{
			{name: "Should reject empty workspace", ws: " ", loopName: "delivery", want: "workspace_id is required"},
			{name: "Should reject empty loop name", ws: "ws-1", loopName: " ", want: "loop_name is required"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				err := globalDB.UpsertLoopConfig(ctx, tc.ws, tc.loopName, looppkg.LoopConfig{})
				if !errors.Is(err, looppkg.ErrValidation) {
					t.Fatalf("UpsertLoopConfig() error = %v, want ErrValidation", err)
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("UpsertLoopConfig() error = %v, want %q", err, tc.want)
				}
			})
		}
	})

	t.Run("Should preserve omitted overrides on partial update", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		humanGate := true
		reattempt := looppkg.ReattemptFullBody
		workerModel := "stored-worker"
		if err := globalDB.UpsertLoopConfig(ctx, "ws-1", "delivery", looppkg.LoopConfig{
			HumanGateEnabled:  &humanGate,
			ReattemptStrategy: &reattempt,
			EnabledChecks:     []byte(`{"command":true}`),
			BudgetTokens:      new(2000),
			FanOutWidth:       new(5),
			ModelDefaults:     &looppkg.ModelDefaults{Worker: &workerModel},
		}); err != nil {
			t.Fatalf("UpsertLoopConfig(initial) error = %v", err)
		}
		if err := globalDB.UpsertLoopConfig(ctx, "ws-1", "delivery", looppkg.LoopConfig{
			BudgetTokens: new(5000),
		}); err != nil {
			t.Fatalf("UpsertLoopConfig(partial) error = %v", err)
		}

		got, err := globalDB.GetLoopConfig(ctx, "ws-1", "delivery")
		if err != nil {
			t.Fatalf("GetLoopConfig() error = %v", err)
		}
		if got.HumanGateEnabled == nil || !*got.HumanGateEnabled {
			t.Fatalf("HumanGateEnabled = %#v, want preserved true", got.HumanGateEnabled)
		}
		if got.ReattemptStrategy == nil || *got.ReattemptStrategy != looppkg.ReattemptFullBody {
			t.Fatalf("ReattemptStrategy = %#v, want preserved full_body", got.ReattemptStrategy)
		}
		if string(got.EnabledChecks) != `{"command":true}` {
			t.Fatalf("EnabledChecks = %s, want preserved command check", got.EnabledChecks)
		}
		if got.FanOutWidth == nil || *got.FanOutWidth != 5 {
			t.Fatalf("FanOutWidth = %#v, want preserved 5", got.FanOutWidth)
		}
		if got.BudgetTokens == nil || *got.BudgetTokens != 5000 {
			t.Fatalf("BudgetTokens = %#v, want updated 5000", got.BudgetTokens)
		}
		if got.ModelDefaults == nil || got.ModelDefaults.Worker == nil || *got.ModelDefaults.Worker != "stored-worker" {
			t.Fatalf("ModelDefaults.Worker = %#v, want preserved stored-worker", got.ModelDefaults)
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
		snapshot, err := globalDB.GetLoopDefinitionSnapshot(ctx, created.WorkspaceID, created.DefinitionDigest)
		if err != nil {
			t.Fatalf("GetLoopDefinitionSnapshot() error = %v", err)
		}
		if snapshot.Digest != created.DefinitionDigest || snapshot.ByteSize != len(created.DefinitionSnapshot) {
			t.Fatalf("snapshot = %#v, want digest %q and byte size %d",
				snapshot,
				created.DefinitionDigest,
				len(created.DefinitionSnapshot),
			)
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

	t.Run("Should ignore same-status compare-and-swap without appending an event", func(t *testing.T) {
		t.Parallel()

		globalDB := openFreshTestGlobalDB(t)
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 4, 14, 5, 0, 0, time.UTC)
		run := testLoopRun("looprun-cas-noop", now, looppkg.StatusRunning)
		if _, err := globalDB.CreateLoopRunForStart(ctx, run, dsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart() error = %v", err)
		}
		if err := globalDB.CompareAndSwapLoopRunStatus(
			ctx,
			run.ID,
			looppkg.StatusRunning,
			looppkg.StatusRunning,
			looppkg.TransitionCauseApproval,
			now.Add(time.Second),
		); err != nil {
			t.Fatalf("CompareAndSwapLoopRunStatus(no-op) error = %v", err)
		}
		if eventCount := countLoopRunEvents(ctx, t, globalDB, run.ID); eventCount != 1 {
			t.Fatalf("status event count = %d, want only create event", eventCount)
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
		StartedAt:         at,
		LastProgressAt:    at,
		DefinitionVersion: 1,
		DefinitionDigest:  "sha256:test-definition",
		DefinitionSnapshot: []byte(
			`{"apiVersion":"agh.loop/v1","kind":"Loop","meta":{"name":"delivery","version":1},"contract":{"goal":"test","definition_of_done":"done","iteration_cap":1,"no_progress":{"window":1},"budget":{"tokens":1,"wall_clock_sec":1}},"graph":{"nodes":[],"edges":[]}}`,
		),
		ActiveHumanCriteria: []byte(`[]`),
		StartMetadata:       map[string]any{},
		IterationCap:        7,
		BudgetOnExceeded:    dsl.BudgetExceededHalt,
		Inputs:              map[string]any{"tasks": "task-ref"},
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
