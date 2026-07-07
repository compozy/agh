package globaldb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

// ReconcileLoopCoordinatorsOnBoot re-enqueues missing coordinators for running loops only.
func (g *GlobalDB) ReconcileLoopCoordinatorsOnBoot(
	ctx context.Context,
	origin taskpkg.Origin,
	now time.Time,
) ([]taskpkg.Run, error) {
	if err := g.checkReady(ctx, "reconcile loop coordinators on boot"); err != nil {
		return nil, err
	}
	normalizedOrigin, err := normalizeLoopCoordinatorReconcileOrigin(origin)
	if err != nil {
		return nil, err
	}
	now = g.normalizeLoopCoordinatorReconcileTime(now)

	enqueued := make([]taskpkg.Run, 0)
	if err := g.withTaskImmediateTransaction(
		ctx,
		"reconcile loop coordinators on boot",
		func(exec taskSQLExecutor) error {
			recovered, err := g.reconcileMissingRunningLoopCoordinators(
				ctx,
				exec,
				normalizedOrigin,
				now,
			)
			if err != nil {
				return err
			}
			enqueued = append(enqueued, recovered...)
			promoted, err := g.promoteQueuedLoopCoordinators(ctx, exec, normalizedOrigin, now)
			if err != nil {
				return err
			}
			enqueued = append(enqueued, promoted...)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return enqueued, nil
}

// EnqueueLoopCoordinatorWake reserves a coordinator run for one loop_run.
func (g *GlobalDB) EnqueueLoopCoordinatorWake(
	ctx context.Context,
	loopRunID string,
	idempotencyKey string,
	origin taskpkg.Origin,
	now time.Time,
) (taskpkg.Run, bool, error) {
	if err := g.checkReady(ctx, "enqueue loop coordinator wake"); err != nil {
		return taskpkg.Run{}, false, err
	}
	trimmedLoopRunID := strings.TrimSpace(loopRunID)
	if trimmedLoopRunID == "" {
		return taskpkg.Run{}, false, fmt.Errorf("%w: loop_run_id is required", taskpkg.ErrValidation)
	}
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return taskpkg.Run{}, false, fmt.Errorf("%w: idempotency_key is required", taskpkg.ErrValidation)
	}
	normalizedOrigin, err := normalizeLoopCoordinatorReconcileOrigin(origin)
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	now = g.normalizeLoopCoordinatorReconcileTime(now)

	var run taskpkg.Run
	var added bool
	if err := g.withTaskImmediateTransaction(
		ctx,
		"enqueue loop coordinator wake",
		func(exec taskSQLExecutor) error {
			taskID, err := lastCoordinatorTaskIDForLoopRun(ctx, exec, trimmedLoopRunID)
			switch {
			case err == nil:
			case errorsIsNoRows(err):
				return nil
			default:
				return err
			}
			run, added, err = g.reserveCoordinatorRun(ctx, exec, taskID, trimmedLoopRunID, key, normalizedOrigin, now)
			return err
		},
	); err != nil {
		return taskpkg.Run{}, false, err
	}
	return run, added, nil
}

// PromoteOldestQueuedLoopRun promotes the FIFO queued run for one workspace loop when no run is live.
func (g *GlobalDB) PromoteOldestQueuedLoopRun(
	ctx context.Context,
	workspaceID string,
	loopName string,
	origin taskpkg.Origin,
	now time.Time,
) (taskpkg.Run, bool, error) {
	if err := g.checkReady(ctx, "promote oldest queued loop run"); err != nil {
		return taskpkg.Run{}, false, err
	}
	normalizedOrigin, err := normalizeLoopCoordinatorReconcileOrigin(origin)
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	now = g.normalizeLoopCoordinatorReconcileTime(now)

	var run taskpkg.Run
	var added bool
	if err := g.withTaskImmediateTransaction(
		ctx,
		"promote oldest queued loop run",
		func(exec taskSQLExecutor) error {
			candidate, err := oldestQueuedLoopRunReadyForPromotion(
				ctx,
				exec,
				strings.TrimSpace(workspaceID),
				strings.TrimSpace(loopName),
			)
			if err != nil {
				return err
			}
			if candidate == nil {
				return nil
			}
			run, added, err = g.promoteQueuedLoopCoordinator(ctx, exec, *candidate, normalizedOrigin, now)
			return err
		},
	); err != nil {
		return taskpkg.Run{}, false, err
	}
	return run, added, nil
}

func normalizeLoopCoordinatorReconcileOrigin(origin taskpkg.Origin) (taskpkg.Origin, error) {
	normalized := taskpkg.Origin{
		Kind: origin.Kind.Normalize(),
		Ref:  strings.TrimSpace(origin.Ref),
	}
	if err := normalized.Validate("loop_coordinator_reconcile.origin"); err != nil {
		return taskpkg.Origin{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) AdvanceLoopRunProgress(ctx context.Context, loopRunID string, at time.Time) error {
	if err := g.checkReady(ctx, "advance loop run progress"); err != nil {
		return err
	}
	trimmedLoopRunID := strings.TrimSpace(loopRunID)
	if trimmedLoopRunID == "" {
		return nil
	}
	if at.IsZero() {
		at = g.now()
	}
	at = at.UTC()
	if _, err := g.db.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET last_progress_at = CASE
		     WHEN last_progress_at IS NULL OR last_progress_at < ? THEN ?
		     ELSE last_progress_at
		   END
		 WHERE id = ?`,
		store.FormatTimestamp(at),
		store.FormatTimestamp(at),
		trimmedLoopRunID,
	); err != nil {
		return fmt.Errorf("store: advance loop run %q progress: %w", trimmedLoopRunID, err)
	}
	return nil
}

func (g *GlobalDB) normalizeLoopCoordinatorReconcileTime(now time.Time) time.Time {
	if now.IsZero() {
		return g.now()
	}
	return now.UTC()
}
