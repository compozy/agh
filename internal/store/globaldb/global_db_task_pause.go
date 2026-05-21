package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

// PauseTask marks one task as paused for future claim eligibility.
func (g *GlobalDB) PauseTask(ctx context.Context, mutation taskpkg.PauseMutation) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "pause task"); err != nil {
		return taskpkg.Task{}, err
	}
	taskID, err := requireTaskValue(mutation.TaskID, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}
	actor := strings.TrimSpace(mutation.Actor)
	reason := strings.TrimSpace(mutation.Reason)
	if actor == "" {
		return taskpkg.Task{}, fmt.Errorf("%w: pause actor is required", taskpkg.ErrValidation)
	}
	if reason == "" {
		return taskpkg.Task{}, fmt.Errorf("%w: pause reason is required", taskpkg.ErrValidation)
	}
	pausedAt := mutation.PausedAt.UTC()
	if pausedAt.IsZero() {
		pausedAt = g.now().UTC()
	}

	var updated taskpkg.Task
	if err := g.withTaskImmediateTransaction(ctx, "pause task", func(exec taskSQLExecutor) error {
		if _, err := g.getTaskWithExecutor(ctx, exec, taskID); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE tasks
			    SET paused = 1, paused_by = ?, paused_at = ?, paused_reason = ?, updated_at = ?
			  WHERE id = ?`,
			actor,
			store.FormatTimestamp(pausedAt),
			reason,
			store.FormatTimestamp(pausedAt),
			taskID,
		)
		if err != nil {
			return fmt.Errorf("store: pause task %q: %w", taskID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskNotFound, taskID, "task"); err != nil {
			return err
		}
		record, err := g.getTaskWithExecutor(ctx, exec, taskID)
		if err != nil {
			return err
		}
		updated = record
		return nil
	}); err != nil {
		return taskpkg.Task{}, err
	}
	return updated, nil
}

// ResumeTask clears one task pause for future claim eligibility.
func (g *GlobalDB) ResumeTask(ctx context.Context, mutation taskpkg.ResumeMutation) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "resume task"); err != nil {
		return taskpkg.Task{}, err
	}
	taskID, err := requireTaskValue(mutation.TaskID, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}
	resumedAt := mutation.ResumedAt.UTC()
	if resumedAt.IsZero() {
		resumedAt = g.now().UTC()
	}

	var updated taskpkg.Task
	if err := g.withTaskImmediateTransaction(ctx, "resume task", func(exec taskSQLExecutor) error {
		if _, err := g.getTaskWithExecutor(ctx, exec, taskID); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE tasks
			    SET paused = 0, paused_by = '', paused_at = NULL, paused_reason = '', updated_at = ?
			  WHERE id = ?`,
			store.FormatTimestamp(resumedAt),
			taskID,
		)
		if err != nil {
			return fmt.Errorf("store: resume task %q: %w", taskID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskNotFound, taskID, "task"); err != nil {
			return err
		}
		record, err := g.getTaskWithExecutor(ctx, exec, taskID)
		if err != nil {
			return err
		}
		updated = record
		return nil
	}); err != nil {
		return taskpkg.Task{}, err
	}
	return updated, nil
}

// CountPausedTasks returns the number of directly paused tasks.
func (g *GlobalDB) CountPausedTasks(ctx context.Context) (int, error) {
	if err := g.checkReady(ctx, "count paused tasks"); err != nil {
		return 0, err
	}
	var count int
	if err := g.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE paused = 1`).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count paused tasks: %w", err)
	}
	return count, nil
}

// CountActiveTaskRunClaims returns currently leased task runs.
func (g *GlobalDB) CountActiveTaskRunClaims(ctx context.Context) (int, error) {
	if err := g.checkReady(ctx, "count active task-run claims"); err != nil {
		return 0, err
	}
	var count int
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		   FROM task_runs
		  WHERE status IN (?, ?, ?)`,
		string(taskpkg.TaskRunStatusClaimed),
		string(taskpkg.TaskRunStatusStarting),
		string(taskpkg.TaskRunStatusRunning),
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count active task-run claims: %w", err)
	}
	return count, nil
}

// CountQueuedTaskRuns returns queued run pressure, optionally excluding paused tasks.
func (g *GlobalDB) CountQueuedTaskRuns(ctx context.Context, includePaused bool) (int, error) {
	if err := g.checkReady(ctx, "count queued task runs"); err != nil {
		return 0, err
	}
	query := `SELECT COUNT(1)
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE tr.status = ?`
	args := []any{string(taskpkg.TaskRunStatusQueued)}
	if !includePaused {
		query += " AND " + effectiveTaskPauseExclusionSQL()
	}
	var count int
	if err := g.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count queued task runs: %w", err)
	}
	return count, nil
}

// SchedulerBacklog returns queued task runs ordered by scheduler priority.
func (g *GlobalDB) SchedulerBacklog(
	ctx context.Context,
	query taskpkg.SchedulerBacklogQuery,
) (backlog taskpkg.SchedulerBacklog, err error) {
	if err := g.checkReady(ctx, "list scheduler backlog"); err != nil {
		return taskpkg.SchedulerBacklog{}, err
	}
	normalized := normalizeSchedulerBacklogQuery(query)
	where := []string{"tr.status = ?"}
	args := []any{string(taskpkg.TaskRunStatusQueued)}
	if normalized.WorkspaceID != "" {
		where = append(where, "t.workspace_id = ?")
		args = append(args, normalized.WorkspaceID)
	}
	if !normalized.IncludePaused {
		where = append(where, effectiveTaskPauseExclusionSQL())
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	var countBuilder strings.Builder
	countBuilder.WriteString(`SELECT COUNT(1)
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE `)
	countBuilder.WriteString(whereSQL)
	countQuery := countBuilder.String()
	if err := g.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return taskpkg.SchedulerBacklog{}, fmt.Errorf("store: count scheduler backlog: %w", err)
	}

	var rowBuilder strings.Builder
	rowBuilder.WriteString(`SELECT tr.id
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE `)
	rowBuilder.WriteString(whereSQL)
	rowBuilder.WriteString(`
		ORDER BY `)
	rowBuilder.WriteString(taskPriorityValueSQL)
	rowBuilder.WriteString(` DESC, tr.queued_at ASC, tr.id ASC`)
	rowQuery := rowBuilder.String()
	rowQuery, args = store.AppendLimit(rowQuery, args, normalized.Limit)
	rows, err := g.db.QueryContext(ctx, rowQuery, args...)
	if err != nil {
		return taskpkg.SchedulerBacklog{}, fmt.Errorf("store: query scheduler backlog: %w", err)
	}
	defer func() {
		err = joinRowsCloseError(rows, err, "scheduler backlog")
	}()

	items := make([]taskpkg.SchedulerBacklogRun, 0)
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			return taskpkg.SchedulerBacklog{}, fmt.Errorf("store: scan scheduler backlog run id: %w", err)
		}
		run, err := g.GetTaskRun(ctx, runID)
		if err != nil {
			return taskpkg.SchedulerBacklog{}, err
		}
		taskRecord, err := g.GetTask(ctx, run.TaskID)
		if err != nil {
			return taskpkg.SchedulerBacklog{}, err
		}
		effectivePaused, pausedByTaskID, err := g.IsTaskEffectivelyPaused(ctx, taskRecord.ID)
		if err != nil {
			return taskpkg.SchedulerBacklog{}, err
		}
		items = append(items, taskpkg.SchedulerBacklogRun{
			Task:            taskRecord,
			Run:             run,
			EffectivePaused: effectivePaused,
			PausedByTaskID:  pausedByTaskID,
		})
	}
	if err := rows.Err(); err != nil {
		return taskpkg.SchedulerBacklog{}, fmt.Errorf("store: iterate scheduler backlog: %w", err)
	}
	return taskpkg.SchedulerBacklog{Runs: items, Total: total}, nil
}

func effectiveTaskPauseExclusionSQL() string {
	return `COALESCE(t.paused, 0) = 0
		AND NOT EXISTS (
			WITH RECURSIVE ancestors(id, parent_task_id, paused) AS (
				SELECT parent.id, parent.parent_task_id, parent.paused
				  FROM tasks parent
				 WHERE parent.id = t.parent_task_id
				UNION ALL
				SELECT parent.id, parent.parent_task_id, parent.paused
				  FROM tasks parent
				  JOIN ancestors a ON parent.id = a.parent_task_id
			)
			SELECT 1 FROM ancestors WHERE COALESCE(paused, 0) = 1
		)`
}

func normalizeSchedulerBacklogQuery(query taskpkg.SchedulerBacklogQuery) taskpkg.SchedulerBacklogQuery {
	normalized := query
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	if normalized.Limit <= 0 {
		normalized.Limit = 50
	}
	if normalized.Limit > 500 {
		normalized.Limit = 500
	}
	return normalized
}

// IsTaskEffectivelyPaused reports whether a task or one of its ancestors is paused.
func (g *GlobalDB) IsTaskEffectivelyPaused(ctx context.Context, taskID string) (bool, string, error) {
	if err := g.checkReady(ctx, "get effective task pause"); err != nil {
		return false, "", err
	}
	trimmedID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return false, "", err
	}
	var pausedByTaskID string
	err = g.db.QueryRowContext(
		ctx,
		`WITH RECURSIVE chain(id, parent_task_id, paused, depth) AS (
			SELECT id, parent_task_id, paused, 0
			  FROM tasks
			 WHERE id = ?
			UNION ALL
			SELECT parent.id, parent.parent_task_id, parent.paused, chain.depth + 1
			  FROM tasks parent
			  JOIN chain ON parent.id = chain.parent_task_id
		)
		SELECT id
		  FROM chain
		 WHERE COALESCE(paused, 0) = 1
		 ORDER BY depth ASC
		 LIMIT 1`,
		trimmedID,
	).Scan(&pausedByTaskID)
	if errors.Is(err, sql.ErrNoRows) {
		if existsErr := g.ensureTaskExists(ctx, trimmedID); existsErr != nil {
			return false, "", existsErr
		}
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("store: get effective task pause for %q: %w", trimmedID, err)
	}
	return true, strings.TrimSpace(pausedByTaskID), nil
}
