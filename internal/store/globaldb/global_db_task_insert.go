package globaldb

import (
	"context"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func insertTaskWithExecutor(ctx context.Context, exec taskSQLExecutor, record taskpkg.Task) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, auto_enqueue_on_ready, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, paused, paused_by, paused_at, paused_reason, wake_creator, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		store.NullableString(record.Identifier),
		string(record.Scope),
		store.NullableString(record.WorkspaceID),
		store.NullableString(record.ParentTaskID),
		store.NullableString(record.NetworkChannel),
		record.Title,
		store.NullableString(record.Description),
		string(record.Priority),
		record.MaxAttempts,
		taskBoolToInt(record.AutoEnqueueOnReady),
		string(record.Status),
		string(record.ApprovalPolicy),
		string(record.ApprovalState),
		taskOwnerKindValue(record.Owner),
		taskOwnerRefValue(record.Owner),
		string(record.CreatedBy.Kind),
		record.CreatedBy.Ref,
		string(record.Origin.Kind),
		record.Origin.Ref,
		store.FormatTimestamp(record.CreatedAt),
		store.FormatTimestamp(record.UpdatedAt),
		nullableTaskTimestamp(record.ClosedAt),
		taskBoolToInt(record.Paused),
		record.PausedBy,
		nullableTaskTimestamp(record.PausedAt),
		record.PausedReason,
		taskBoolToInt(record.WakeCreator),
		nullableTaskJSON(record.Metadata),
	)
	return err
}
