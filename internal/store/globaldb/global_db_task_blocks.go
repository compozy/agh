package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const taskRunReleaseReasonBlocked = "blocked"

var _ taskpkg.BlockStore = (*GlobalDB)(nil)

// CreateTaskBlock inserts one task block, stamping workspace_id from the owning task.
func (g *GlobalDB) CreateTaskBlock(
	ctx context.Context,
	mutation taskpkg.CreateTaskBlockMutation,
) (taskpkg.BlockMutationResult, error) {
	if err := g.checkReady(ctx, "create task block"); err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	normalized, err := normalizeCreateTaskBlockMutation(mutation, g.now())
	if err != nil {
		return taskpkg.BlockMutationResult{}, err
	}

	var result taskpkg.BlockMutationResult
	if err := g.withTaskImmediateTransaction(ctx, "create task block", func(exec taskSQLExecutor) error {
		var createErr error
		result, createErr = g.insertTaskBlockWithBreaker(
			ctx,
			exec,
			normalized.Block,
			normalized.RecurrenceLimit,
			normalized.Actor,
		)
		if createErr != nil {
			return createErr
		}
		if err := appendTaskBlockedWatchEvent(ctx, exec, result.Block, normalized.Actor, "", ""); err != nil {
			return err
		}
		return appendNeedsAttentionWatchEventIfEscalated(ctx, exec, result, normalized.Actor)
	}); err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	return result, nil
}

// GetTaskBlock returns one task block when it belongs to the supplied task.
func (g *GlobalDB) GetTaskBlock(ctx context.Context, taskID string, blockID string) (taskpkg.TaskBlock, error) {
	if err := g.checkReady(ctx, "get task block"); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	return g.getTaskBlockWithExecutor(ctx, g.db, taskID, blockID)
}

// ClearTaskBlock stamps one open block as cleared and rejects repeated clears as conflicts.
func (g *GlobalDB) ClearTaskBlock(
	ctx context.Context,
	mutation taskpkg.ClearTaskBlockMutation,
) (taskpkg.TaskBlock, error) {
	if err := g.checkReady(ctx, "clear task block"); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	normalized, err := normalizeClearTaskBlockMutation(mutation, g.now())
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}

	var cleared taskpkg.TaskBlock
	if err := g.withTaskImmediateTransaction(ctx, "clear task block", func(exec taskSQLExecutor) error {
		taskRecord, err := g.getTaskWithExecutor(ctx, exec, normalized.TaskID)
		if err != nil {
			return err
		}
		workspaceID := taskBlockWorkspaceID(taskRecord)
		workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(workspaceID)
		args := []any{
			store.FormatTimestamp(normalized.ClearedAt),
			string(normalized.ClearedBy.Kind),
			normalized.ClearedBy.Ref,
			store.NullableString(normalized.ClearNote),
			normalized.BlockID,
			normalized.TaskID,
		}
		args = append(args, workspaceArgs...)
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_blocks
			    SET cleared_at = ?, cleared_by_kind = ?, cleared_by_ref = ?, clear_note = ?
			  WHERE id = ? AND task_id = ? AND `+workspaceWhere+` AND cleared_at IS NULL`,
			args...,
		)
		if err != nil {
			return fmt.Errorf("store: clear task block %q: %w", normalized.BlockID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: rows affected for task block %q: %w", normalized.BlockID, err)
		}
		if affected == 0 {
			current, loadErr := g.getTaskBlockWithExecutor(ctx, exec, normalized.TaskID, normalized.BlockID)
			if loadErr != nil {
				return loadErr
			}
			if !current.ClearedAt.IsZero() {
				return fmt.Errorf("%w: task block %q is already cleared", taskpkg.ErrConflict, normalized.BlockID)
			}
			return fmt.Errorf("store: task block %q: %w", normalized.BlockID, taskpkg.ErrTaskBlockNotFound)
		}
		cleared, err = g.getTaskBlockWithExecutor(ctx, exec, normalized.TaskID, normalized.BlockID)
		if err != nil {
			return err
		}
		return appendTaskUnblockedWatchEvent(ctx, exec, cleared, normalized.Actor)
	}); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	return cleared, nil
}

// ExpireTaskBlocks finalizes expired transient blocks as daemon-cleared rows, grouped one transaction per task.
func (g *GlobalDB) ExpireTaskBlocks(
	ctx context.Context,
	mutation taskpkg.ExpireTaskBlocksMutation,
) (taskpkg.ExpireTaskBlocksResult, error) {
	if err := g.checkReady(ctx, "expire task blocks"); err != nil {
		return taskpkg.ExpireTaskBlocksResult{}, err
	}
	normalized, err := normalizeExpireTaskBlocksMutation(mutation, g.now())
	if err != nil {
		return taskpkg.ExpireTaskBlocksResult{}, err
	}
	candidates, err := g.listExpiredTaskBlockCandidates(ctx, normalized.Now)
	if err != nil {
		return taskpkg.ExpireTaskBlocksResult{}, err
	}
	blocks := make([]taskpkg.TaskBlock, 0, len(candidates))
	for _, taskID := range uniqueTaskBlockCandidateTaskIDs(candidates) {
		var taskBlocks []taskpkg.TaskBlock
		if err := g.withTaskImmediateTransaction(ctx, "expire task blocks", func(exec taskSQLExecutor) error {
			var expireErr error
			taskBlocks, expireErr = g.expireTaskBlocksForTaskWithExecutor(ctx, exec, taskID, normalized)
			return expireErr
		}); err != nil {
			return taskpkg.ExpireTaskBlocksResult{}, err
		}
		blocks = append(blocks, taskBlocks...)
	}
	return taskpkg.ExpireTaskBlocksResult{Blocks: blocks}, nil
}

// ListTaskBlocks returns task blocks for one task, open-only by default.
func (g *GlobalDB) ListTaskBlocks(
	ctx context.Context,
	taskID string,
	includeCleared bool,
) ([]taskpkg.TaskBlock, error) {
	if err := g.checkReady(ctx, "list task blocks"); err != nil {
		return nil, err
	}
	return g.listTaskBlocksWithExecutor(ctx, g.db, taskID, includeCleared, g.now())
}

func (g *GlobalDB) listTaskBlocksWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	includeCleared bool,
	now time.Time,
) ([]taskpkg.TaskBlock, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return nil, err
	}
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, trimmedTaskID)
	if err != nil {
		return nil, err
	}
	workspaceID := taskBlockWorkspaceID(taskRecord)
	workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(workspaceID)
	query := `SELECT ` + taskBlockSelectColumnsSQL + `
		FROM task_blocks
		WHERE task_id = ? AND ` + workspaceWhere
	args := []any{trimmedTaskID}
	args = append(args, workspaceArgs...)
	if !includeCleared {
		query += ` AND cleared_at IS NULL AND (expires_at IS NULL OR expires_at > ?)`
		args = append(args, store.FormatTimestamp(now.UTC()))
	}
	query += ` ORDER BY created_at ASC, id ASC`

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task blocks for task %q: %w", trimmedTaskID, err)
	}
	blocks := make([]taskpkg.TaskBlock, 0)
	for rows.Next() {
		block, scanErr := scanTaskBlockRecord(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task block query")
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(rows, fmt.Errorf("store: iterate task blocks: %w", err), "task block query")
	}
	if err := joinRowsCloseError(rows, nil, "task block query"); err != nil {
		return nil, err
	}
	return blocks, nil
}

// HasOpenTaskBlocks returns whether a task currently has any open, non-expired block.
func (g *GlobalDB) HasOpenTaskBlocks(ctx context.Context, taskID string) (bool, error) {
	if err := g.checkReady(ctx, "check open task blocks"); err != nil {
		return false, err
	}
	return g.hasOpenTaskBlocksWithExecutor(ctx, g.db, taskID, g.now())
}

func (g *GlobalDB) hasOpenTaskBlocksWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	now time.Time,
) (bool, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return false, err
	}
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, trimmedTaskID)
	if err != nil {
		return false, err
	}
	workspaceID := taskBlockWorkspaceID(taskRecord)
	workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(workspaceID)
	args := []any{trimmedTaskID}
	args = append(args, workspaceArgs...)
	args = append(args, store.FormatTimestamp(now.UTC()))
	row := exec.QueryRowContext(
		ctx,
		`SELECT 1
		   FROM task_blocks
		  WHERE task_id = ? AND `+workspaceWhere+`
		    AND cleared_at IS NULL
		    AND (expires_at IS NULL OR expires_at > ?)
		  LIMIT 1`,
		args...,
	)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("store: check open task blocks for task %q: %w", trimmedTaskID, err)
	}
	return exists == 1, nil
}

// UpsertTaskBlockRecurrence sets the persisted counter for one task and block kind.
func (g *GlobalDB) UpsertTaskBlockRecurrence(
	ctx context.Context,
	recurrence taskpkg.BlockRecurrence,
) (taskpkg.BlockRecurrence, error) {
	if err := g.checkReady(ctx, "upsert task block recurrence"); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	normalized, err := normalizeBlockRecurrence(recurrence, g.now())
	if err != nil {
		return taskpkg.BlockRecurrence{}, err
	}

	var stored taskpkg.BlockRecurrence
	if err := g.withTaskImmediateTransaction(ctx, "upsert task block recurrence", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalized.TaskID); err != nil {
			return err
		}
		stored, err = upsertBlockRecurrenceWithExecutor(ctx, exec, normalized)
		return err
	}); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	return stored, nil
}

// IncrementTaskBlockRecurrence increments and returns the counter for one task and block kind.
func (g *GlobalDB) IncrementTaskBlockRecurrence(
	ctx context.Context,
	taskID string,
	kind taskpkg.BlockKind,
	updatedAt time.Time,
) (taskpkg.BlockRecurrence, error) {
	if err := g.checkReady(ctx, "increment task block recurrence"); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	normalizedTaskID, normalizedKind, now, err := normalizeBlockRecurrenceKey(taskID, kind, updatedAt, g.now())
	if err != nil {
		return taskpkg.BlockRecurrence{}, err
	}

	var stored taskpkg.BlockRecurrence
	if err := g.withTaskImmediateTransaction(ctx, "increment task block recurrence", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalizedTaskID); err != nil {
			return err
		}
		stored, err = incrementBlockRecurrenceWithExecutor(ctx, exec, normalizedTaskID, normalizedKind, now)
		return err
	}); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	return stored, nil
}

// GetTaskBlockRecurrence returns the counter for one task and block kind, or a zero counter when absent.
func (g *GlobalDB) GetTaskBlockRecurrence(
	ctx context.Context,
	taskID string,
	kind taskpkg.BlockKind,
) (taskpkg.BlockRecurrence, error) {
	if err := g.checkReady(ctx, "get task block recurrence"); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	normalizedTaskID, normalizedKind, _, err := normalizeBlockRecurrenceKey(taskID, kind, time.Time{}, g.now())
	if err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	if err := g.ensureTaskExistsWithExecutor(ctx, g.db, normalizedTaskID); err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	return getBlockRecurrenceWithExecutor(ctx, g.db, normalizedTaskID, normalizedKind)
}

// ResetTaskBlockRecurrences clears all breaker counters for one task.
func (g *GlobalDB) ResetTaskBlockRecurrences(ctx context.Context, taskID string) error {
	if err := g.checkReady(ctx, "reset task block recurrences"); err != nil {
		return err
	}
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return err
	}
	return g.withTaskImmediateTransaction(ctx, "reset task block recurrences", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, trimmedTaskID); err != nil {
			return err
		}
		return resetTaskBlockRecurrencesWithExecutor(ctx, exec, trimmedTaskID)
	})
}

// MarkTaskNeedsAttention writes the task-level escalation metadata columns.
func (g *GlobalDB) MarkTaskNeedsAttention(
	ctx context.Context,
	mutation taskpkg.NeedsAttentionMutation,
) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "mark task needs attention"); err != nil {
		return taskpkg.Task{}, err
	}
	normalized, err := normalizeNeedsAttentionMutation(mutation, g.now())
	if err != nil {
		return taskpkg.Task{}, err
	}
	return g.updateTaskNeedsAttention(ctx, normalized)
}

// ClearTaskNeedsAttention clears the task-level escalation metadata columns.
func (g *GlobalDB) ClearTaskNeedsAttention(
	ctx context.Context,
	mutation taskpkg.NeedsAttentionClearMutation,
) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "clear task needs attention"); err != nil {
		return taskpkg.Task{}, err
	}
	trimmedTaskID, err := requireTaskValue(mutation.TaskID, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}
	clearedAt := mutation.ClearedAt.UTC()
	if clearedAt.IsZero() {
		clearedAt = g.now().UTC()
	}
	clearedBy := mutation.ClearedBy
	clearedBy.Kind = clearedBy.Kind.Normalize()
	clearedBy.Ref = strings.TrimSpace(clearedBy.Ref)
	if err := clearedBy.Validate("task.needs_attention_cleared_by"); err != nil {
		return taskpkg.Task{}, err
	}
	origin := mutation.Origin
	origin.Kind = origin.Kind.Normalize()
	origin.Ref = strings.TrimSpace(origin.Ref)
	if err := origin.Validate("task.needs_attention_clear_origin"); err != nil {
		return taskpkg.Task{}, err
	}
	var updated taskpkg.Task
	if err := g.withTaskImmediateTransaction(ctx, "clear task needs attention", func(exec taskSQLExecutor) error {
		result, err := exec.ExecContext(
			ctx,
			`UPDATE tasks
			    SET needs_attention_reason = NULL,
			        needs_attention_at = NULL,
			        needs_attention_by_kind = NULL,
			        needs_attention_by_ref = NULL,
			        updated_at = ?
			  WHERE id = ? AND needs_attention_at IS NOT NULL`,
			store.FormatTimestamp(clearedAt),
			trimmedTaskID,
		)
		if err != nil {
			return fmt.Errorf("store: clear task needs attention %q: %w", trimmedTaskID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: rows affected for task needs attention clear %q: %w", trimmedTaskID, err)
		}
		if affected == 0 {
			if _, loadErr := g.getTaskWithExecutor(ctx, exec, trimmedTaskID); loadErr != nil {
				return loadErr
			}
			return fmt.Errorf(
				"%w: task %q is not escalated",
				taskpkg.ErrInvalidStatusTransition,
				trimmedTaskID,
			)
		}
		updated, err = g.getTaskWithExecutor(ctx, exec, trimmedTaskID)
		if err != nil {
			return err
		}
		return appendTaskEventPayloadWithExecutor(
			ctx,
			exec,
			updated.ID,
			"",
			string(hookspkg.HookTaskRecovered),
			taskpkg.ActorContext{Actor: clearedBy, Origin: origin, Authority: taskpkg.Authority{Write: true}},
			clearedAt,
			taskRecoveredWatchEventPayload{At: clearedAt},
		)
	}); err != nil {
		return taskpkg.Task{}, err
	}
	return updated, nil
}

// SetTaskWakeCreator writes the per-task creator wake opt-in flag.
func (g *GlobalDB) SetTaskWakeCreator(
	ctx context.Context,
	mutation taskpkg.WakeCreatorMutation,
) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "set task wake creator"); err != nil {
		return taskpkg.Task{}, err
	}
	trimmedTaskID, err := requireTaskValue(mutation.TaskID, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}
	updatedAt := mutation.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = g.now().UTC()
	}

	var updated taskpkg.Task
	if err := g.withTaskImmediateTransaction(ctx, "set task wake creator", func(exec taskSQLExecutor) error {
		result, err := exec.ExecContext(
			ctx,
			`UPDATE tasks SET wake_creator = ?, updated_at = ? WHERE id = ?`,
			taskBoolToInt(mutation.WakeCreator),
			store.FormatTimestamp(updatedAt),
			trimmedTaskID,
		)
		if err != nil {
			return fmt.Errorf("store: set task wake creator %q: %w", trimmedTaskID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskNotFound, trimmedTaskID, "task"); err != nil {
			return err
		}
		updated, err = g.getTaskWithExecutor(ctx, exec, trimmedTaskID)
		return err
	}); err != nil {
		return taskpkg.Task{}, err
	}
	return updated, nil
}

// BlockTaskAndReleaseRun inserts a block, evaluates breaker accounting, and releases the active run atomically.
func (g *GlobalDB) BlockTaskAndReleaseRun(
	ctx context.Context,
	mutation taskpkg.BlockTaskAndReleaseRunMutation,
) (taskpkg.BlockTaskAndReleaseRunResult, error) {
	if err := g.checkReady(ctx, "block task and release run"); err != nil {
		return taskpkg.BlockTaskAndReleaseRunResult{}, err
	}
	normalized, err := normalizeBlockTaskAndReleaseRunMutation(mutation, g.now())
	if err != nil {
		return taskpkg.BlockTaskAndReleaseRunResult{}, err
	}

	var result taskpkg.BlockTaskAndReleaseRunResult
	if err := g.withTaskImmediateTransaction(ctx, "block task and release run", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if strings.TrimSpace(current.TaskID) != strings.TrimSpace(normalized.Block.TaskID) {
			return fmt.Errorf(
				"%w: task run %q belongs to task %q, not %q",
				taskpkg.ErrInvalidStatusTransition,
				current.ID,
				current.TaskID,
				normalized.Block.TaskID,
			)
		}

		blockResult, err := g.insertTaskBlockWithBreaker(
			ctx,
			exec,
			normalized.Block,
			normalized.RecurrenceLimit,
			normalized.Actor,
		)
		if err != nil {
			return err
		}
		if err := requeueLeasedRun(ctx, exec, current.ID); err != nil {
			return err
		}
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
		updatedRun, err := g.getTaskRunWithExecutor(ctx, exec, current.ID)
		if err != nil {
			return err
		}
		result = taskpkg.BlockTaskAndReleaseRunResult{
			Block:          blockResult.Block,
			Run:            updatedRun,
			Recurrence:     blockResult.Recurrence,
			EscalatedTask:  blockResult.EscalatedTask,
			ReleaseReason:  taskRunReleaseReasonBlocked,
			PreviousRun:    current,
			ClaimTokenHash: current.ClaimTokenHash,
		}
		if err := appendTaskBlockedWatchEvent(
			ctx,
			exec,
			result.Block,
			normalized.Actor,
			updatedRun.ID,
			current.ClaimTokenHash,
		); err != nil {
			return err
		}
		return appendNeedsAttentionWatchEventIfEscalated(ctx, exec, blockResult, normalized.Actor)
	}); err != nil {
		return taskpkg.BlockTaskAndReleaseRunResult{}, err
	}
	return result, nil
}

const taskBlockSelectColumnsSQL = `id, workspace_id, task_id, kind, reason, details_json,
	created_by_kind, created_by_ref, created_at, expires_at, cleared_at,
	cleared_by_kind, cleared_by_ref, clear_note`

func (g *GlobalDB) insertTaskBlockWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	block taskpkg.TaskBlock,
) (taskpkg.TaskBlock, error) {
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, block.TaskID)
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	normalized := block
	normalized.WorkspaceID = taskBlockWorkspaceID(taskRecord)
	if err := validateTaskBlockForInsert(normalized); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_blocks (
			id, workspace_id, task_id, kind, reason, details_json, created_by_kind, created_by_ref,
			created_at, expires_at, cleared_at, cleared_by_kind, cleared_by_ref, clear_note
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, NULL)`,
		normalized.ID,
		store.NullableString(normalized.WorkspaceID),
		normalized.TaskID,
		string(normalized.Kind),
		normalized.Reason,
		nullableTaskJSON(normalized.Details),
		string(normalized.CreatedBy.Kind),
		normalized.CreatedBy.Ref,
		store.FormatTimestamp(normalized.CreatedAt),
		nullableTaskTimestamp(normalized.ExpiresAt),
	); err != nil {
		return taskpkg.TaskBlock{}, fmt.Errorf("store: create task block %q: %w", normalized.ID, err)
	}
	return g.getTaskBlockWithExecutor(ctx, exec, normalized.TaskID, normalized.ID)
}

func (g *GlobalDB) insertTaskBlockWithBreaker(
	ctx context.Context,
	exec taskSQLExecutor,
	block taskpkg.TaskBlock,
	recurrenceLimit int,
	actor taskpkg.ActorContext,
) (taskpkg.BlockMutationResult, error) {
	hadClearedPrior, err := g.hasClearedTaskBlockKindWithExecutor(ctx, exec, block.TaskID, block.Kind)
	if err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	created, err := g.insertTaskBlockWithExecutor(ctx, exec, block)
	if err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	result := taskpkg.BlockMutationResult{
		Block: created,
		Recurrence: taskpkg.BlockRecurrence{
			TaskID: created.TaskID,
			Kind:   created.Kind,
		},
	}
	if !hadClearedPrior {
		return result, nil
	}
	recurrence, err := incrementBlockRecurrenceWithExecutor(ctx, exec, created.TaskID, created.Kind, created.CreatedAt)
	if err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	result.Recurrence = recurrence
	if recurrenceLimit == 0 {
		return result, nil
	}
	if recurrence.Count < recurrenceLimit {
		return result, nil
	}
	escalated, changed, err := g.markTaskNeedsAttentionIfClearWithExecutor(ctx, exec, taskpkg.NeedsAttentionMutation{
		TaskID:   created.TaskID,
		Reason:   blockRecurrenceNeedsAttentionReason(recurrence),
		Actor:    actor.Actor,
		MarkedAt: created.CreatedAt,
		Origin:   actor.Origin,
	})
	if err != nil {
		return taskpkg.BlockMutationResult{}, err
	}
	if changed {
		result.EscalatedTask = &escalated
	}
	return result, nil
}

func (g *GlobalDB) hasClearedTaskBlockKindWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	kind taskpkg.BlockKind,
) (bool, error) {
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, taskID)
	if err != nil {
		return false, err
	}
	workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(taskBlockWorkspaceID(taskRecord))
	args := []any{strings.TrimSpace(taskID), string(kind.Normalize())}
	args = append(args, workspaceArgs...)
	row := exec.QueryRowContext(
		ctx,
		`SELECT 1
		   FROM task_blocks
		  WHERE task_id = ? AND kind = ? AND `+workspaceWhere+`
		    AND cleared_at IS NOT NULL
		  LIMIT 1`,
		args...,
	)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf(
			"store: check cleared task block recurrence prior for task %q kind %q: %w",
			taskID,
			kind,
			err,
		)
	}
	return exists == 1, nil
}

func blockRecurrenceNeedsAttentionReason(recurrence taskpkg.BlockRecurrence) string {
	return fmt.Sprintf(
		"task re-blocked %d times with %q blocks",
		recurrence.Count,
		recurrence.Kind.Normalize(),
	)
}

type taskBlockExpiryCandidate struct {
	TaskID  string
	BlockID string
}

func (g *GlobalDB) listExpiredTaskBlockCandidates(
	ctx context.Context,
	now time.Time,
) ([]taskBlockExpiryCandidate, error) {
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT task_id, id
		   FROM task_blocks
		  WHERE cleared_at IS NULL
		    AND expires_at IS NOT NULL
		    AND expires_at <= ?
		  ORDER BY task_id ASC, created_at ASC, id ASC`,
		store.FormatTimestamp(now.UTC()),
	)
	if err != nil {
		return nil, fmt.Errorf("store: query expired task blocks: %w", err)
	}
	candidates := make([]taskBlockExpiryCandidate, 0)
	for rows.Next() {
		var candidate taskBlockExpiryCandidate
		if err := rows.Scan(&candidate.TaskID, &candidate.BlockID); err != nil {
			return nil, joinRowsCloseError(
				rows,
				fmt.Errorf("store: scan expired task block: %w", err),
				"expired task block query",
			)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate expired task blocks: %w", err),
			"expired task block query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "expired task block query"); err != nil {
		return nil, err
	}
	return candidates, nil
}

func uniqueTaskBlockCandidateTaskIDs(candidates []taskBlockExpiryCandidate) []string {
	seen := make(map[string]struct{}, len(candidates))
	taskIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		taskID := strings.TrimSpace(candidate.TaskID)
		if taskID == "" {
			continue
		}
		if _, ok := seen[taskID]; ok {
			continue
		}
		seen[taskID] = struct{}{}
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

func (g *GlobalDB) expireTaskBlocksForTaskWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	mutation taskpkg.ExpireTaskBlocksMutation,
) ([]taskpkg.TaskBlock, error) {
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, taskID)
	if err != nil {
		return nil, err
	}
	workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(taskBlockWorkspaceID(taskRecord))
	blockIDs, err := expiredTaskBlockIDsForTaskWithExecutor(
		ctx,
		exec,
		taskID,
		workspaceWhere,
		workspaceArgs,
		mutation.Now,
	)
	if err != nil {
		return nil, err
	}

	expired := make([]taskpkg.TaskBlock, 0, len(blockIDs))
	for _, blockID := range blockIDs {
		updateArgs := []any{
			store.FormatTimestamp(mutation.Now),
			string(mutation.ClearedBy.Kind),
			mutation.ClearedBy.Ref,
			blockID,
			strings.TrimSpace(taskID),
			store.FormatTimestamp(mutation.Now),
		}
		updateArgs = append(updateArgs, workspaceArgs...)
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_blocks
			    SET cleared_at = ?, cleared_by_kind = ?, cleared_by_ref = ?, clear_note = NULL
			  WHERE id = ? AND task_id = ?
			    AND cleared_at IS NULL
			    AND expires_at IS NOT NULL
			    AND expires_at <= ?
			    AND `+workspaceWhere,
			updateArgs...,
		)
		if err != nil {
			return nil, fmt.Errorf("store: expire task block %q: %w", blockID, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("store: rows affected for expired task block %q: %w", blockID, err)
		}
		if affected == 0 {
			continue
		}
		block, err := g.getTaskBlockWithExecutor(ctx, exec, taskID, blockID)
		if err != nil {
			return nil, err
		}
		expired = append(expired, block)
	}
	return expired, nil
}

func expiredTaskBlockIDsForTaskWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	workspaceWhere string,
	workspaceArgs []any,
	now time.Time,
) ([]string, error) {
	args := []any{strings.TrimSpace(taskID), store.FormatTimestamp(now)}
	args = append(args, workspaceArgs...)
	rows, err := exec.QueryContext(
		ctx,
		`SELECT id
		   FROM task_blocks
		  WHERE task_id = ?
		    AND cleared_at IS NULL
		    AND expires_at IS NOT NULL
		    AND expires_at <= ?
		    AND `+workspaceWhere+`
		  ORDER BY created_at ASC, id ASC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query expired task blocks for task %q: %w", taskID, err)
	}
	blockIDs := make([]string, 0)
	for rows.Next() {
		var blockID string
		if err := rows.Scan(&blockID); err != nil {
			return nil, joinRowsCloseError(
				rows,
				fmt.Errorf("store: scan expired block id: %w", err),
				"expired block id query",
			)
		}
		blockIDs = append(blockIDs, blockID)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate expired block ids: %w", err),
			"expired block id query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "expired block id query"); err != nil {
		return nil, err
	}
	return blockIDs, nil
}

func normalizeCreateTaskBlockMutation(
	mutation taskpkg.CreateTaskBlockMutation,
	defaultNow time.Time,
) (taskpkg.CreateTaskBlockMutation, error) {
	normalized := mutation
	if normalized.RecurrenceLimit < 0 {
		return taskpkg.CreateTaskBlockMutation{}, fmt.Errorf(
			"%w: task_block.recurrence_limit cannot be negative: %d",
			taskpkg.ErrValidation,
			normalized.RecurrenceLimit,
		)
	}
	block, err := normalizeTaskBlockForCreate(normalized.Block, defaultNow)
	if err != nil {
		return taskpkg.CreateTaskBlockMutation{}, err
	}
	if err := normalized.Actor.Validate(); err != nil {
		return taskpkg.CreateTaskBlockMutation{}, err
	}
	normalized.Block = block
	return normalized, nil
}

func normalizeExpireTaskBlocksMutation(
	mutation taskpkg.ExpireTaskBlocksMutation,
	defaultNow time.Time,
) (taskpkg.ExpireTaskBlocksMutation, error) {
	normalized := mutation
	if normalized.Now.IsZero() {
		normalized.Now = defaultNow.UTC()
	} else {
		normalized.Now = normalized.Now.UTC()
	}
	normalized.ClearedBy.Kind = normalized.ClearedBy.Kind.Normalize()
	normalized.ClearedBy.Ref = strings.TrimSpace(normalized.ClearedBy.Ref)
	if err := normalized.ClearedBy.Validate("task_block.expired_by"); err != nil {
		return taskpkg.ExpireTaskBlocksMutation{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) getTaskBlockWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	blockID string,
) (taskpkg.TaskBlock, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	trimmedBlockID, err := requireTaskValue(blockID, "task block id")
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, trimmedTaskID)
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	workspaceWhere, workspaceArgs := taskBlockWorkspaceWhere(taskBlockWorkspaceID(taskRecord))
	args := []any{trimmedBlockID, trimmedTaskID}
	args = append(args, workspaceArgs...)

	row := exec.QueryRowContext(
		ctx,
		`SELECT `+taskBlockSelectColumnsSQL+`
		   FROM task_blocks
		  WHERE id = ? AND task_id = ? AND `+workspaceWhere,
		args...,
	)
	block, err := scanTaskBlockRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.TaskBlock{}, taskpkg.ErrTaskBlockNotFound
		}
		return taskpkg.TaskBlock{}, err
	}
	return block, nil
}

func upsertBlockRecurrenceWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	recurrence taskpkg.BlockRecurrence,
) (taskpkg.BlockRecurrence, error) {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_block_recurrences (task_id, kind, count, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(task_id, kind) DO UPDATE SET
		   count = excluded.count,
		   updated_at = excluded.updated_at`,
		recurrence.TaskID,
		string(recurrence.Kind),
		recurrence.Count,
		store.FormatTimestamp(recurrence.UpdatedAt),
	); err != nil {
		return taskpkg.BlockRecurrence{}, fmt.Errorf(
			"store: upsert task block recurrence for task %q kind %q: %w",
			recurrence.TaskID,
			recurrence.Kind,
			err,
		)
	}
	return getBlockRecurrenceWithExecutor(ctx, exec, recurrence.TaskID, recurrence.Kind)
}

func incrementBlockRecurrenceWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	kind taskpkg.BlockKind,
	updatedAt time.Time,
) (taskpkg.BlockRecurrence, error) {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_block_recurrences (task_id, kind, count, updated_at)
		 VALUES (?, ?, 1, ?)
		 ON CONFLICT(task_id, kind) DO UPDATE SET
		   count = count + 1,
		   updated_at = excluded.updated_at`,
		taskID,
		string(kind),
		store.FormatTimestamp(updatedAt),
	); err != nil {
		return taskpkg.BlockRecurrence{}, fmt.Errorf(
			"store: increment task block recurrence for task %q kind %q: %w",
			taskID,
			kind,
			err,
		)
	}
	return getBlockRecurrenceWithExecutor(ctx, exec, taskID, kind)
}

func resetTaskBlockRecurrencesWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM task_block_recurrences WHERE task_id = ?`, taskID)
	if err != nil {
		return fmt.Errorf("store: reset task block recurrences for task %q: %w", taskID, err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("store: rows affected for task block recurrences reset %q: %w", taskID, err)
	}
	return nil
}

func getBlockRecurrenceWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	kind taskpkg.BlockKind,
) (taskpkg.BlockRecurrence, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT task_id, kind, count, updated_at
		   FROM task_block_recurrences
		  WHERE task_id = ? AND kind = ?`,
		taskID,
		string(kind),
	)
	var recurrence taskpkg.BlockRecurrence
	var kindRaw string
	var updatedAtRaw string
	if err := row.Scan(&recurrence.TaskID, &kindRaw, &recurrence.Count, &updatedAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.BlockRecurrence{
				TaskID: taskID,
				Kind:   kind,
			}, nil
		}
		return taskpkg.BlockRecurrence{}, fmt.Errorf("store: scan task block recurrence: %w", err)
	}
	recurrence.Kind = taskpkg.BlockKind(strings.TrimSpace(kindRaw))
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	recurrence.UpdatedAt = updatedAt
	return recurrence, nil
}

func (g *GlobalDB) updateTaskNeedsAttention(
	ctx context.Context,
	mutation taskpkg.NeedsAttentionMutation,
) (taskpkg.Task, error) {
	var updated taskpkg.Task
	if err := g.withTaskImmediateTransaction(ctx, "mark task needs attention", func(exec taskSQLExecutor) error {
		var changed bool
		var err error
		updated, changed, err = g.markTaskNeedsAttentionIfClearWithExecutor(ctx, exec, mutation)
		if err != nil {
			return err
		}
		if !changed {
			var loadErr error
			updated, loadErr = g.getTaskWithExecutor(ctx, exec, mutation.TaskID)
			return loadErr
		}
		return nil
	}); err != nil {
		return taskpkg.Task{}, err
	}
	return updated, nil
}

func (g *GlobalDB) markTaskNeedsAttentionIfClearWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	mutation taskpkg.NeedsAttentionMutation,
) (taskpkg.Task, bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE tasks
		    SET needs_attention_reason = ?,
		        needs_attention_at = ?,
		        needs_attention_by_kind = ?,
		        needs_attention_by_ref = ?,
		        updated_at = ?
		  WHERE id = ? AND needs_attention_at IS NULL`,
		mutation.Reason,
		store.FormatTimestamp(mutation.MarkedAt),
		string(mutation.Actor.Kind),
		mutation.Actor.Ref,
		store.FormatTimestamp(mutation.MarkedAt),
		mutation.TaskID,
	)
	if err != nil {
		return taskpkg.Task{}, false, fmt.Errorf("store: mark task needs attention %q: %w", mutation.TaskID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return taskpkg.Task{}, false, fmt.Errorf(
			"store: rows affected for task needs attention %q: %w",
			mutation.TaskID,
			err,
		)
	}
	if affected == 0 {
		if _, loadErr := g.getTaskWithExecutor(ctx, exec, mutation.TaskID); loadErr != nil {
			return taskpkg.Task{}, false, loadErr
		}
		return taskpkg.Task{}, false, nil
	}
	updated, err := g.getTaskWithExecutor(ctx, exec, mutation.TaskID)
	if err != nil {
		return taskpkg.Task{}, false, err
	}
	return updated, true, nil
}

func normalizeTaskBlockForCreate(block taskpkg.TaskBlock, defaultNow time.Time) (taskpkg.TaskBlock, error) {
	normalized := block
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Kind = normalized.Kind.Normalize()
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Details = normalizeTaskJSON(normalized.Details)
	normalized.CreatedBy.Kind = normalized.CreatedBy.Kind.Normalize()
	normalized.CreatedBy.Ref = strings.TrimSpace(normalized.CreatedBy.Ref)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = defaultNow.UTC()
	} else {
		normalized.CreatedAt = normalized.CreatedAt.UTC()
	}
	if !normalized.ExpiresAt.IsZero() {
		normalized.ExpiresAt = normalized.ExpiresAt.UTC()
	}
	normalized.ClearedAt = time.Time{}
	normalized.ClearedBy = taskpkg.ActorIdentity{}
	normalized.ClearNote = ""
	if err := validateTaskBlockFields(normalized); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	return normalized, nil
}

func validateTaskBlockFields(block taskpkg.TaskBlock) error {
	if strings.TrimSpace(block.ID) == "" {
		return fmt.Errorf("%w: task_block.id is required", taskpkg.ErrValidation)
	}
	if strings.TrimSpace(block.TaskID) == "" {
		return fmt.Errorf("%w: task_block.task_id is required", taskpkg.ErrValidation)
	}
	if err := block.Kind.Validate("task_block.kind"); err != nil {
		return err
	}
	if strings.TrimSpace(block.Reason) == "" {
		return fmt.Errorf("%w: task_block.reason is required", taskpkg.ErrValidation)
	}
	if err := taskpkg.ValidatePayloadSize(block.Details, "task_block.details"); err != nil {
		return err
	}
	if err := block.CreatedBy.Validate("task_block.created_by"); err != nil {
		return err
	}
	if block.CreatedAt.IsZero() {
		return fmt.Errorf("%w: task_block.created_at is required", taskpkg.ErrValidation)
	}
	return nil
}

func validateTaskBlockForInsert(block taskpkg.TaskBlock) error {
	return validateTaskBlockFields(block)
}

func normalizeClearTaskBlockMutation(
	mutation taskpkg.ClearTaskBlockMutation,
	defaultNow time.Time,
) (taskpkg.ClearTaskBlockMutation, error) {
	normalized := mutation
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.BlockID = strings.TrimSpace(normalized.BlockID)
	normalized.ClearedBy.Kind = normalized.ClearedBy.Kind.Normalize()
	normalized.ClearedBy.Ref = strings.TrimSpace(normalized.ClearedBy.Ref)
	normalized.ClearNote = strings.TrimSpace(normalized.ClearNote)
	if normalized.ClearedAt.IsZero() {
		normalized.ClearedAt = defaultNow.UTC()
	} else {
		normalized.ClearedAt = normalized.ClearedAt.UTC()
	}
	if _, err := requireTaskValue(normalized.TaskID, "task id"); err != nil {
		return taskpkg.ClearTaskBlockMutation{}, err
	}
	if _, err := requireTaskValue(normalized.BlockID, "task block id"); err != nil {
		return taskpkg.ClearTaskBlockMutation{}, err
	}
	if err := normalized.ClearedBy.Validate("task_block.cleared_by"); err != nil {
		return taskpkg.ClearTaskBlockMutation{}, err
	}
	if err := normalized.Actor.Validate(); err != nil {
		return taskpkg.ClearTaskBlockMutation{}, err
	}
	return normalized, nil
}

func normalizeBlockRecurrence(
	recurrence taskpkg.BlockRecurrence,
	defaultNow time.Time,
) (taskpkg.BlockRecurrence, error) {
	taskID, kind, updatedAt, err := normalizeBlockRecurrenceKey(
		recurrence.TaskID,
		recurrence.Kind,
		recurrence.UpdatedAt,
		defaultNow,
	)
	if err != nil {
		return taskpkg.BlockRecurrence{}, err
	}
	if recurrence.Count < 0 {
		return taskpkg.BlockRecurrence{}, fmt.Errorf(
			"%w: task_block_recurrence.count cannot be negative: %d",
			taskpkg.ErrValidation,
			recurrence.Count,
		)
	}
	return taskpkg.BlockRecurrence{
		TaskID:    taskID,
		Kind:      kind,
		Count:     recurrence.Count,
		UpdatedAt: updatedAt,
	}, nil
}

func normalizeBlockRecurrenceKey(
	taskID string,
	kind taskpkg.BlockKind,
	updatedAt time.Time,
	defaultNow time.Time,
) (string, taskpkg.BlockKind, time.Time, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return "", "", time.Time{}, err
	}
	normalizedKind := kind.Normalize()
	if err := normalizedKind.Validate("task_block_recurrence.kind"); err != nil {
		return "", "", time.Time{}, err
	}
	now := updatedAt.UTC()
	if now.IsZero() {
		now = defaultNow.UTC()
	}
	return trimmedTaskID, normalizedKind, now, nil
}

func normalizeNeedsAttentionMutation(
	mutation taskpkg.NeedsAttentionMutation,
	defaultNow time.Time,
) (taskpkg.NeedsAttentionMutation, error) {
	normalized := mutation
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Reason = strings.TrimSpace(normalized.Reason)
	normalized.Actor.Kind = normalized.Actor.Kind.Normalize()
	normalized.Actor.Ref = strings.TrimSpace(normalized.Actor.Ref)
	normalized.Origin.Kind = normalized.Origin.Kind.Normalize()
	normalized.Origin.Ref = strings.TrimSpace(normalized.Origin.Ref)
	if normalized.MarkedAt.IsZero() {
		normalized.MarkedAt = defaultNow.UTC()
	} else {
		normalized.MarkedAt = normalized.MarkedAt.UTC()
	}
	if _, err := requireTaskValue(normalized.TaskID, "task id"); err != nil {
		return taskpkg.NeedsAttentionMutation{}, err
	}
	if normalized.Reason == "" {
		return taskpkg.NeedsAttentionMutation{}, fmt.Errorf(
			"%w: task.needs_attention_reason is required",
			taskpkg.ErrValidation,
		)
	}
	if err := normalized.Actor.Validate("task.needs_attention_by"); err != nil {
		return taskpkg.NeedsAttentionMutation{}, err
	}
	if err := normalized.Origin.Validate("task.needs_attention_origin"); err != nil {
		return taskpkg.NeedsAttentionMutation{}, err
	}
	return normalized, nil
}

func normalizeBlockTaskAndReleaseRunMutation(
	mutation taskpkg.BlockTaskAndReleaseRunMutation,
	defaultNow time.Time,
) (taskpkg.BlockTaskAndReleaseRunMutation, error) {
	normalized := mutation
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ClaimToken = strings.TrimSpace(normalized.ClaimToken)
	if normalized.RecurrenceLimit < 0 {
		return taskpkg.BlockTaskAndReleaseRunMutation{}, fmt.Errorf(
			"%w: block_task_and_release_run.recurrence_limit cannot be negative: %d",
			taskpkg.ErrValidation,
			normalized.RecurrenceLimit,
		)
	}
	if normalized.Now.IsZero() {
		normalized.Now = defaultNow.UTC()
	} else {
		normalized.Now = normalized.Now.UTC()
	}
	block, err := normalizeTaskBlockForCreate(normalized.Block, normalized.Now)
	if err != nil {
		return taskpkg.BlockTaskAndReleaseRunMutation{}, err
	}
	if err := normalized.Actor.Validate(); err != nil {
		return taskpkg.BlockTaskAndReleaseRunMutation{}, err
	}
	normalized.Block = block
	if _, err := requireTaskValue(normalized.RunID, "task run id"); err != nil {
		return taskpkg.BlockTaskAndReleaseRunMutation{}, err
	}
	if strings.TrimSpace(normalized.ClaimToken) == "" {
		return taskpkg.BlockTaskAndReleaseRunMutation{}, fmt.Errorf(
			"%w: block_task_and_release_run.claim_token is required",
			taskpkg.ErrValidation,
		)
	}
	return normalized, nil
}

func scanTaskBlockRecord(scanner rowScanner) (taskpkg.TaskBlock, error) {
	var block taskpkg.TaskBlock
	var fields taskBlockScanFields
	if err := scanner.Scan(
		&block.ID,
		&fields.workspaceID,
		&block.TaskID,
		&fields.kind,
		&block.Reason,
		&fields.details,
		&fields.createdByKind,
		&block.CreatedBy.Ref,
		&fields.createdAtRaw,
		&fields.expiresAtRaw,
		&fields.clearedAtRaw,
		&fields.clearedByKind,
		&fields.clearedByRef,
		&fields.clearNote,
	); err != nil {
		return taskpkg.TaskBlock{}, fmt.Errorf("store: scan task block: %w", err)
	}
	block.WorkspaceID = strings.TrimSpace(fields.workspaceID.String)
	block.Kind = taskpkg.BlockKind(strings.TrimSpace(fields.kind))
	block.CreatedBy.Kind = taskpkg.ActorKind(strings.TrimSpace(fields.createdByKind))
	block.ClearNote = taskNullStringValue(fields.clearNote)
	if fields.clearedByKind.Valid || fields.clearedByRef.Valid {
		block.ClearedBy = taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKind(strings.TrimSpace(fields.clearedByKind.String)),
			Ref:  strings.TrimSpace(fields.clearedByRef.String),
		}
	}
	details, err := decodeTaskJSON(fields.details, "task_block.details_json")
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	block.Details = details
	createdAt, err := store.ParseTimestamp(fields.createdAtRaw)
	if err != nil {
		return taskpkg.TaskBlock{}, err
	}
	block.CreatedAt = createdAt
	if err := assignNullableTaskTimestamp(&block.ExpiresAt, fields.expiresAtRaw); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	if err := assignNullableTaskTimestamp(&block.ClearedAt, fields.clearedAtRaw); err != nil {
		return taskpkg.TaskBlock{}, err
	}
	return block, nil
}

type taskBlockScanFields struct {
	workspaceID   sql.NullString
	kind          string
	details       sql.NullString
	createdByKind string
	createdAtRaw  string
	expiresAtRaw  sql.NullString
	clearedAtRaw  sql.NullString
	clearedByKind sql.NullString
	clearedByRef  sql.NullString
	clearNote     sql.NullString
}

func taskBlockWorkspaceID(taskRecord taskpkg.Task) string {
	return strings.TrimSpace(taskRecord.WorkspaceID)
}

func taskBlockWorkspaceWhere(workspaceID string) (string, []any) {
	if strings.TrimSpace(workspaceID) == "" {
		return "workspace_id IS NULL", nil
	}
	return "workspace_id = ?", []any{strings.TrimSpace(workspaceID)}
}
