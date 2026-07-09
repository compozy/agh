package globaldb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

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
		TaskID: taskID, Kind: kind, Count: recurrence.Count, UpdatedAt: updatedAt,
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
