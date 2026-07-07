package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func oldestQueuedLoopRunReadyForPromotion(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID string,
	loopName string,
) (*loopCoordinatorCandidate, error) {
	if workspaceID == "" || loopName == "" {
		return nil, nil
	}
	var candidate loopCoordinatorCandidate
	if err := exec.QueryRowContext(
		ctx,
		`SELECT q.id, q.workspace_id, q.loop_name, q.generation
		 FROM loop_runs q
		 WHERE q.workspace_id = ?
		   AND q.loop_name = ?
		   AND q.status = 'queued'
		   AND NOT EXISTS (
		     SELECT 1
		     FROM loop_runs active
		     WHERE active.workspace_id = q.workspace_id
		       AND active.loop_name = q.loop_name
		       AND active.status IN ('running', 'watching', 'needs-approval', 'paused')
		   )
		   AND NOT EXISTS (
		     SELECT 1
		     FROM loop_runs older
		     WHERE older.workspace_id = q.workspace_id
		       AND older.loop_name = q.loop_name
		       AND older.status = 'queued'
		       AND (
		         older.created_at < q.created_at OR
		         (older.created_at = q.created_at AND older.id < q.id)
		       )
		   )
		 ORDER BY q.created_at ASC, q.id ASC
		 LIMIT 1`,
		workspaceID,
		loopName,
	).Scan(
		&candidate.loopRunID,
		&candidate.workspaceID,
		&candidate.loopName,
		&candidate.generation,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"store: find queued loop promotion candidate for %q/%q: %w",
			workspaceID,
			loopName,
			err,
		)
	}
	return &candidate, nil
}

func (g *GlobalDB) reconcileMissingRunningLoopCoordinators(
	ctx context.Context,
	exec taskSQLExecutor,
	origin taskpkg.Origin,
	now time.Time,
) ([]taskpkg.Run, error) {
	missing, err := loopRunsMissingActiveCoordinator(ctx, exec)
	if err != nil {
		return nil, err
	}
	enqueued := make([]taskpkg.Run, 0, len(missing))
	for _, candidate := range missing {
		current, err := getLoopRunByIDWithExecutor(ctx, exec, loop.RunID(strings.TrimSpace(candidate.loopRunID)))
		if err != nil {
			return nil, err
		}
		generation := candidate.generation + 1
		run, added, err := g.reserveLoopCoordinatorRunWithExecutor(
			ctx,
			exec,
			current,
			origin,
			now,
			loopCoordinatorRunID(current.ID, generation),
			loopCoordinatorIdempotencyKey(current.ID, generation),
		)
		if err != nil {
			return nil, err
		}
		if added {
			enqueued = append(enqueued, run)
		}
	}
	return enqueued, nil
}

func (g *GlobalDB) promoteQueuedLoopCoordinators(
	ctx context.Context,
	exec taskSQLExecutor,
	origin taskpkg.Origin,
	now time.Time,
) ([]taskpkg.Run, error) {
	promotable, err := queuedLoopRunsReadyForPromotion(ctx, exec)
	if err != nil {
		return nil, err
	}
	enqueued := make([]taskpkg.Run, 0, len(promotable))
	for _, candidate := range promotable {
		run, added, err := g.promoteQueuedLoopCoordinator(ctx, exec, candidate, origin, now)
		if err != nil {
			return nil, err
		}
		if added {
			enqueued = append(enqueued, run)
		}
	}
	return enqueued, nil
}

func (g *GlobalDB) promoteQueuedLoopCoordinator(
	ctx context.Context,
	exec taskSQLExecutor,
	candidate loopCoordinatorCandidate,
	origin taskpkg.Origin,
	now time.Time,
) (taskpkg.Run, bool, error) {
	current, err := getLoopRunByIDWithExecutor(ctx, exec, loop.RunID(strings.TrimSpace(candidate.loopRunID)))
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		current,
		loop.StatusRunning,
		loop.TransitionCausePromote,
		now,
		candidate.generation,
	); err != nil {
		return taskpkg.Run{}, false, err
	}
	generation := candidate.generation + 1
	return g.reserveLoopCoordinatorRunWithExecutor(
		ctx,
		exec,
		current,
		origin,
		now,
		loopCoordinatorRunID(current.ID, generation),
		loopCoordinatorIdempotencyKey(current.ID, generation),
	)
}

func (g *GlobalDB) reserveCoordinatorRun(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	loopRunID string,
	runID string,
	idempotencyKey string,
	origin taskpkg.Origin,
	now time.Time,
) (taskpkg.Run, bool, error) {
	reservedRunID := strings.TrimSpace(runID)
	if reservedRunID == "" {
		reservedRunID = store.NewID("run")
	}
	reservation := queuedRunReservationInput{
		taskID:         taskID,
		runID:          reservedRunID,
		runKind:        taskpkg.RunKindCoordinator,
		loopRunID:      loopRunID,
		idempotencyKey: idempotencyKey,
		origin:         origin,
		queuedAt:       now,
	}
	_, run, existing, err := g.reserveQueuedRunWithExecutor(ctx, exec, reservation)
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	return run, !existing, nil
}

type loopCoordinatorCandidate struct {
	loopRunID   string
	workspaceID string
	loopName    string
	generation  int
}

func loopRunsMissingActiveCoordinator(
	ctx context.Context,
	exec taskSQLExecutor,
) ([]loopCoordinatorCandidate, error) {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT lr.id, lr.generation
		 FROM loop_runs lr
		 WHERE lr.status = 'running'
		   AND NOT EXISTS (
		     SELECT 1
		     FROM task_runs tr
		     WHERE tr.loop_run_id = lr.id
		       AND tr.run_kind = 'coordinator'
		       AND tr.status IN ('queued', 'claimed', 'starting', 'running')
		   )
		 ORDER BY lr.created_at ASC, lr.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list running loops missing coordinator: %w", err)
	}
	defer rows.Close()

	candidates := make([]loopCoordinatorCandidate, 0)
	for rows.Next() {
		var candidate loopCoordinatorCandidate
		if err := rows.Scan(&candidate.loopRunID, &candidate.generation); err != nil {
			return nil, fmt.Errorf("store: scan loop coordinator candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop coordinator candidates: %w", err)
	}
	return candidates, nil
}

func queuedLoopRunsReadyForPromotion(
	ctx context.Context,
	exec taskSQLExecutor,
) ([]loopCoordinatorCandidate, error) {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT q.id, q.workspace_id, q.loop_name, q.generation
		 FROM loop_runs q
		 WHERE q.status = 'queued'
		   AND NOT EXISTS (
		     SELECT 1
		     FROM loop_runs active
		     WHERE active.workspace_id = q.workspace_id
		       AND active.loop_name = q.loop_name
		       AND active.status IN ('running', 'watching', 'needs-approval', 'paused')
		   )
		   AND NOT EXISTS (
		     SELECT 1
		     FROM loop_runs older
		     WHERE older.workspace_id = q.workspace_id
		       AND older.loop_name = q.loop_name
		       AND older.status = 'queued'
		       AND (
		         older.created_at < q.created_at OR
		         (older.created_at = q.created_at AND older.id < q.id)
		       )
		   )
		 ORDER BY q.created_at ASC, q.id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list queued loops ready for promotion: %w", err)
	}
	defer rows.Close()

	candidates := make([]loopCoordinatorCandidate, 0)
	for rows.Next() {
		var candidate loopCoordinatorCandidate
		if err := rows.Scan(
			&candidate.loopRunID,
			&candidate.workspaceID,
			&candidate.loopName,
			&candidate.generation,
		); err != nil {
			return nil, fmt.Errorf("store: scan queued loop promotion candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate queued loop promotion candidates: %w", err)
	}
	return candidates, nil
}

func lastCoordinatorTaskIDForLoopRun(
	ctx context.Context,
	exec taskSQLExecutor,
	loopRunID string,
) (string, error) {
	var taskID string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT task_id
		 FROM task_runs
		 WHERE loop_run_id = ?
		   AND run_kind = 'coordinator'
		 ORDER BY queued_at DESC, id DESC
		 LIMIT 1`,
		loopRunID,
	).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return "", fmt.Errorf("store: find coordinator task for loop run %q: %w", loopRunID, err)
	}
	return strings.TrimSpace(taskID), nil
}

func errorsIsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
