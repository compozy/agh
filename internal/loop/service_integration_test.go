//go:build integration

package loop_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	"github.com/compozy/agh/internal/testutil"
)

func TestServiceIntegrationShouldPersistConfigureAndReflectEffectiveConfig(t *testing.T) {
	t.Parallel()

	t.Run("Should persist loop config and reflect it in later run config", func(t *testing.T) {
		t.Parallel()

		globalDB := openLoopServiceGlobalDB(t)
		svc := newIntegrationService(t, globalDB, validDefinition())
		ctx := testutil.Context(t)
		onExceeded := dsl.BudgetExceededEscalate

		err := svc.Configure(ctx, "ws-1", "valid-loop", loop.LoopConfig{
			BudgetTokens:     new(2222),
			BudgetWallSec:    new(333),
			BudgetOnExceeded: &onExceeded,
			FanOutWidth:      new(9),
			NoProgressWindow: new(4),
		})
		if err != nil {
			t.Fatalf("Configure() error = %v", err)
		}

		run, err := svc.Start(ctx, "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		}, humanActor(t))
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if run.BudgetTokens != 2222 || run.BudgetWallSec != 333 ||
			run.BudgetOnExceeded != dsl.BudgetExceededEscalate {
			t.Fatalf("run budget = tokens:%d wall:%d on_exceeded:%q, want configured values",
				run.BudgetTokens,
				run.BudgetWallSec,
				run.BudgetOnExceeded,
			)
		}

		preview, err := svc.DryRun(ctx, "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		})
		if err != nil {
			t.Fatalf("DryRun() error = %v", err)
		}
		if preview.EffectiveConfig.FanOutWidth != 9 {
			t.Fatalf("DryRun EffectiveConfig.FanOutWidth = %d, want 9", preview.EffectiveConfig.FanOutWidth)
		}
	})
}

func TestServiceIntegrationDryRunShouldCreateNoState(t *testing.T) {
	t.Parallel()

	t.Run("Should create no loop run or task run for a real resolved definition", func(t *testing.T) {
		t.Parallel()

		globalDB := openLoopServiceGlobalDB(t)
		svc := newIntegrationService(t, globalDB, validDefinition())
		ctx := testutil.Context(t)

		loopRunsBefore := countRows(ctx, t, globalDB, "loop_runs")
		taskRunsBefore := countRows(ctx, t, globalDB, "task_runs")
		preview, err := svc.DryRun(ctx, "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		})
		if err != nil {
			t.Fatalf("DryRun() error = %v", err)
		}
		if preview.Generation != 1 || len(preview.Nodes) == 0 {
			t.Fatalf("DryRun preview = %#v, want gen-1 nodes", preview)
		}
		if got := countRows(ctx, t, globalDB, "loop_runs"); got != loopRunsBefore {
			t.Fatalf("loop_runs count = %d, want unchanged %d", got, loopRunsBefore)
		}
		if got := countRows(ctx, t, globalDB, "task_runs"); got != taskRunsBefore {
			t.Fatalf("task_runs count = %d, want unchanged %d", got, taskRunsBefore)
		}
	})
}

func TestServiceIntegrationConfigureShouldClampAtWriteTime(t *testing.T) {
	t.Parallel()

	t.Run("Should clamp loop config overrides before persisting", func(t *testing.T) {
		t.Parallel()

		globalDB := openLoopServiceGlobalDB(t)
		svc := newIntegrationService(t, globalDB, validDefinition())
		ctx := testutil.Context(t)

		err := svc.Configure(ctx, "ws-1", "valid-loop", loop.LoopConfig{
			FanOutWidth:      new(loop.LoopMaxFanoutWidth + 100),
			NoProgressWindow: new(loop.LoopMaxNoProgressWindow + 100),
			GateMaxRevisions: new(loop.LoopMaxGateRevisions + 100),
		})
		if err != nil {
			t.Fatalf("Configure() error = %v", err)
		}

		cfg, err := globalDB.GetLoopConfig(ctx, "ws-1", "valid-loop")
		if err != nil {
			t.Fatalf("GetLoopConfig() error = %v", err)
		}
		if cfg.FanOutWidth == nil || *cfg.FanOutWidth != loop.LoopMaxFanoutWidth {
			t.Fatalf("stored FanOutWidth = %#v, want %d", cfg.FanOutWidth, loop.LoopMaxFanoutWidth)
		}
		if cfg.NoProgressWindow == nil || *cfg.NoProgressWindow != loop.LoopMaxNoProgressWindow {
			t.Fatalf(
				"stored NoProgressWindow = %#v, want %d",
				cfg.NoProgressWindow,
				loop.LoopMaxNoProgressWindow,
			)
		}
		if cfg.GateMaxRevisions == nil || *cfg.GateMaxRevisions != loop.LoopMaxGateRevisions {
			t.Fatalf(
				"stored GateMaxRevisions = %#v, want %d",
				cfg.GateMaxRevisions,
				loop.LoopMaxGateRevisions,
			)
		}
	})
}

func openLoopServiceGlobalDB(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	globalDB, err := globaldb.OpenGlobalDB(
		testutil.Context(t),
		filepath.Join(t.TempDir(), store.GlobalDatabaseName),
	)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	return globalDB
}

func newIntegrationService(
	t *testing.T,
	loopStore loop.Store,
	def dsl.Definition,
) loop.Service {
	t.Helper()

	resolved := compileDefinition(t, def)
	svc, err := loop.NewService(
		loopStore,
		loop.DefinitionResolverFunc(func(
			context.Context,
			loop.WorkspaceID,
			string,
		) (*loop.ResolvedDefinition, error) {
			return resolved, nil
		}),
		loop.GoalRunPolicyResolverFunc(func(
			context.Context,
			loop.WorkspaceID,
		) (*loop.GoalRunPolicy, error) {
			return &loop.GoalRunPolicy{ContextNudgeRatio: 0.8}, nil
		}),
		loop.WithClock(func() time.Time {
			return time.Date(2026, 7, 4, 15, 0, 0, 0, time.UTC)
		}),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func countRows(ctx context.Context, t *testing.T, globalDB *globaldb.GlobalDB, table string) int {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM " + table
	if err := globalDB.DB().QueryRowContext(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count %s rows error = %v", table, err)
	}
	return count
}
