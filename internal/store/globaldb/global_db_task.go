package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

var _ taskpkg.RecordStore = (*GlobalDB)(nil)
var _ taskpkg.RunStore = (*GlobalDB)(nil)

// CreateTask inserts one durable task record.
func (g *GlobalDB) CreateTask(ctx context.Context, record taskpkg.Task) error {
	if err := g.checkReady(ctx, "create task"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskForCreate(record)
	if err != nil {
		return err
	}
	if err := g.ensureTaskCreateReferences(ctx, normalized); err != nil {
		return err
	}

	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		store.NullableString(normalized.Identifier),
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		store.NullableString(normalized.ParentTaskID),
		store.NullableString(normalized.NetworkChannel),
		normalized.Title,
		store.NullableString(normalized.Description),
		string(normalized.Priority),
		normalized.MaxAttempts,
		string(normalized.Status),
		string(normalized.ApprovalPolicy),
		string(normalized.ApprovalState),
		taskOwnerKindValue(normalized.Owner),
		taskOwnerRefValue(normalized.Owner),
		string(normalized.CreatedBy.Kind),
		normalized.CreatedBy.Ref,
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
		nullableTaskTimestamp(normalized.ClosedAt),
		nullableTaskJSON(normalized.Metadata),
	)
	if err != nil {
		return fmt.Errorf("store: create task %q: %w", normalized.ID, err)
	}

	return nil
}

// UpdateTask replaces the persisted canonical task record.
func (g *GlobalDB) UpdateTask(ctx context.Context, record taskpkg.Task) error {
	if err := g.checkReady(ctx, "update task"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskForUpdate(record)
	if err != nil {
		return err
	}

	current, err := g.GetTask(ctx, normalized.ID)
	if err != nil {
		return err
	}
	if err := taskpkg.ValidateImmutableTaskFields(current, normalized); err != nil {
		return err
	}

	normalized.CreatedAt = current.CreatedAt
	result, err := g.db.ExecContext(
		ctx,
		`UPDATE tasks
		 SET identifier = ?, scope = ?, workspace_id = ?, parent_task_id = ?,
		     network_channel = ?, title = ?, description = ?, priority = ?,
		     max_attempts = ?, status = ?, approval_policy = ?, approval_state = ?,
		     owner_kind = ?, owner_ref = ?, created_by_kind = ?,
		     created_by_ref = ?, origin_kind = ?, origin_ref = ?,
		     created_at = ?, updated_at = ?, closed_at = ?, metadata_json = ?
		 WHERE id = ?`,
		store.NullableString(normalized.Identifier),
		string(normalized.Scope),
		store.NullableString(normalized.WorkspaceID),
		store.NullableString(normalized.ParentTaskID),
		store.NullableString(normalized.NetworkChannel),
		normalized.Title,
		store.NullableString(normalized.Description),
		string(normalized.Priority),
		normalized.MaxAttempts,
		string(normalized.Status),
		string(normalized.ApprovalPolicy),
		string(normalized.ApprovalState),
		taskOwnerKindValue(normalized.Owner),
		taskOwnerRefValue(normalized.Owner),
		string(normalized.CreatedBy.Kind),
		normalized.CreatedBy.Ref,
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
		nullableTaskTimestamp(normalized.ClosedAt),
		nullableTaskJSON(normalized.Metadata),
		normalized.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update task %q: %w", normalized.ID, err)
	}

	return requireRowsAffected(result, taskpkg.ErrTaskNotFound, normalized.ID, "task")
}

// GetTask returns one persisted task by primary key.
func (g *GlobalDB) GetTask(ctx context.Context, id string) (taskpkg.Task, error) {
	if err := g.checkReady(ctx, "get task"); err != nil {
		return taskpkg.Task{}, err
	}

	trimmedID, err := requireTaskValue(id, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json
			 FROM tasks
			 WHERE id = ?`,
		trimmedID,
	)

	record, err := scanTaskRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.Task{}, taskpkg.ErrTaskNotFound
		}
		return taskpkg.Task{}, err
	}
	return record, nil
}

// ListTasks returns durable task summaries that match the supplied filters.
func (g *GlobalDB) ListTasks(ctx context.Context, query taskpkg.Query) ([]taskpkg.Summary, error) {
	if err := g.checkReady(ctx, "list tasks"); err != nil {
		return nil, err
	}
	if err := query.Validate("task_query"); err != nil {
		return nil, err
	}

	normalized := normalizeTaskQuery(query)
	sqlQuery := `SELECT
		id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
		priority, max_attempts, status, approval_policy, approval_state,
		owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
		created_at, updated_at, closed_at, metadata_json
		FROM tasks`
	where, args := store.BuildClauses(
		store.StringClause("scope", string(normalized.Scope)),
		store.StringClause("workspace_id", normalized.WorkspaceID),
		store.StringClause("status", string(normalized.Status)),
		store.StringClause("priority", string(normalized.Priority)),
		store.StringClause("approval_state", string(normalized.ApprovalState)),
		store.StringClause("owner_kind", string(normalized.OwnerKind)),
		store.StringClause("owner_ref", normalized.OwnerRef),
		store.StringClause("parent_task_id", normalized.ParentTaskID),
		store.StringClause("network_channel", normalized.NetworkChannel),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, created_at DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query tasks: %w", err)
	}
	defer func() {
		// Close errors are not actionable here once Next/Err have reported the read outcome.
		_ = rows.Close()
	}()

	summaries := make([]taskpkg.Summary, 0)
	for rows.Next() {
		record, scanErr := scanTaskRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		summaries = append(summaries, taskSummaryFromRecord(record))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate tasks: %w", err)
	}

	return summaries, nil
}

// CountDirectChildren reports how many persisted tasks reference the supplied parent id.
func (g *GlobalDB) CountDirectChildren(ctx context.Context, parentTaskID string) (int, error) {
	if err := g.checkReady(ctx, "count task children"); err != nil {
		return 0, err
	}

	trimmedID, err := requireTaskValue(parentTaskID, "parent task id")
	if err != nil {
		return 0, err
	}

	var count int
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM tasks WHERE parent_task_id = ?`,
		trimmedID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count direct children for task %q: %w", trimmedID, err)
	}

	return count, nil
}

// CreateTaskRun inserts one durable task-run record.
func (g *GlobalDB) CreateTaskRun(ctx context.Context, run taskpkg.Run) error {
	if err := g.checkReady(ctx, "create task run"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskRunForCreate(run)
	if err != nil {
		return err
	}
	if err := g.ensureTaskExists(ctx, normalized.TaskID); err != nil {
		return err
	}

	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO task_runs (
			id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref,
			idempotency_key, network_channel, queued_at, claimed_at, started_at, ended_at, error, result_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.TaskID,
		string(normalized.Status),
		normalized.Attempt,
		taskActorKindValue(normalized.ClaimedBy),
		taskActorRefValue(normalized.ClaimedBy),
		store.NullableString(normalized.SessionID),
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		store.NullableString(normalized.IdempotencyKey),
		store.NullableString(normalized.NetworkChannel),
		store.FormatTimestamp(normalized.QueuedAt),
		nullableTaskTimestamp(normalized.ClaimedAt),
		nullableTaskTimestamp(normalized.StartedAt),
		nullableTaskTimestamp(normalized.EndedAt),
		store.NullableString(normalized.Error),
		nullableTaskJSON(normalized.Result),
	)
	if err != nil {
		return fmt.Errorf("store: create task run %q: %w", normalized.ID, err)
	}

	return nil
}

// UpdateTaskRun replaces the persisted canonical task-run record.
func (g *GlobalDB) UpdateTaskRun(ctx context.Context, run taskpkg.Run) error {
	if err := g.checkReady(ctx, "update task run"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskRunForUpdate(run)
	if err != nil {
		return err
	}

	current, err := g.GetTaskRun(ctx, normalized.ID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.SessionID) != "" &&
		strings.TrimSpace(normalized.SessionID) != strings.TrimSpace(current.SessionID) {
		return taskpkg.ErrSessionAlreadyBound
	}
	if normalized.QueuedAt.IsZero() {
		normalized.QueuedAt = current.QueuedAt
	}
	if err := g.ensureTaskExists(ctx, normalized.TaskID); err != nil {
		return err
	}

	result, err := g.db.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET task_id = ?, status = ?, attempt = ?, claimed_by_kind = ?,
		     claimed_by_ref = ?, session_id = ?, origin_kind = ?,
		     origin_ref = ?, idempotency_key = ?, network_channel = ?,
		     queued_at = ?, claimed_at = ?, started_at = ?, ended_at = ?,
		     error = ?, result_json = ?
		 WHERE id = ?`,
		normalized.TaskID,
		string(normalized.Status),
		normalized.Attempt,
		taskActorKindValue(normalized.ClaimedBy),
		taskActorRefValue(normalized.ClaimedBy),
		store.NullableString(normalized.SessionID),
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		store.NullableString(normalized.IdempotencyKey),
		store.NullableString(normalized.NetworkChannel),
		store.FormatTimestamp(normalized.QueuedAt),
		nullableTaskTimestamp(normalized.ClaimedAt),
		nullableTaskTimestamp(normalized.StartedAt),
		nullableTaskTimestamp(normalized.EndedAt),
		store.NullableString(normalized.Error),
		nullableTaskJSON(normalized.Result),
		normalized.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update task run %q: %w", normalized.ID, err)
	}

	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, normalized.ID, "task run")
}

// GetTaskRun returns one persisted task run by primary key.
func (g *GlobalDB) GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "get task run"); err != nil {
		return taskpkg.Run{}, err
	}

	trimmedID, err := requireTaskValue(id, "task run id")
	if err != nil {
		return taskpkg.Run{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref,
			idempotency_key, network_channel, queued_at, claimed_at, started_at, ended_at, error, result_json
		 FROM task_runs
		 WHERE id = ?`,
		trimmedID,
	)

	run, err := scanTaskRunRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.Run{}, taskpkg.ErrTaskRunNotFound
		}
		return taskpkg.Run{}, err
	}
	return run, nil
}

// ListTaskRuns returns persisted runs that match the supplied filters.
func (g *GlobalDB) ListTaskRuns(ctx context.Context, query taskpkg.RunQuery) ([]taskpkg.Run, error) {
	if err := g.checkReady(ctx, "list task runs"); err != nil {
		return nil, err
	}
	if err := query.Validate("task_run_query"); err != nil {
		return nil, err
	}

	normalized := normalizeTaskRunQuery(query)
	sqlQuery := `SELECT
		id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref,
		idempotency_key, network_channel, queued_at, claimed_at, started_at, ended_at, error, result_json
		FROM task_runs`
	where, args := store.BuildClauses(
		store.StringClause("task_id", normalized.TaskID),
		store.StringClause("status", string(normalized.Status)),
		store.StringClause("session_id", normalized.SessionID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY queued_at DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task runs: %w", err)
	}
	defer func() {
		// Close errors are not actionable here once Next/Err have reported the read outcome.
		_ = rows.Close()
	}()

	runs := make([]taskpkg.Run, 0)
	for rows.Next() {
		run, scanErr := scanTaskRunRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task runs: %w", err)
	}

	return runs, nil
}

// ListTaskRunsByStatus returns persisted runs that match any of the supplied statuses.
func (g *GlobalDB) ListTaskRunsByStatus(
	ctx context.Context,
	statuses []taskpkg.RunStatus,
) ([]taskpkg.Run, error) {
	if err := g.checkReady(ctx, "list task runs by status"); err != nil {
		return nil, err
	}
	if len(statuses) == 0 {
		return []taskpkg.Run{}, nil
	}

	placeholders := make([]string, 0, len(statuses))
	args := make([]any, 0, len(statuses))
	for _, status := range statuses {
		normalized := status.Normalize()
		if err := normalized.Validate("task_run_statuses"); err != nil {
			return nil, err
		}
		placeholders = append(placeholders, "?")
		args = append(args, string(normalized))
	}

	rows, err := g.db.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT
				id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref,
				idempotency_key, network_channel, queued_at, claimed_at, started_at, ended_at, error, result_json
			 FROM task_runs
			 WHERE status IN (%s)
			 ORDER BY queued_at ASC, id ASC`,
			strings.Join(placeholders, ", "),
		),
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query task runs by status: %w", err)
	}
	defer func() {
		// Close errors are not actionable here once Next/Err have reported the read outcome.
		_ = rows.Close()
	}()

	runs := make([]taskpkg.Run, 0)
	for rows.Next() {
		run, scanErr := scanTaskRunRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task runs by status: %w", err)
	}

	return runs, nil
}

// CountActiveSessionBindings reports how many non-terminal runs are bound to one session.
func (g *GlobalDB) CountActiveSessionBindings(ctx context.Context, sessionID string) (int, error) {
	if err := g.checkReady(ctx, "count active task-run session bindings"); err != nil {
		return 0, err
	}

	trimmedSessionID, err := requireTaskValue(sessionID, "task run session id")
	if err != nil {
		return 0, err
	}

	var count int
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM task_runs
		 WHERE session_id = ?
		   AND status IN (?, ?, ?)`,
		trimmedSessionID,
		string(taskpkg.TaskRunStatusClaimed),
		string(taskpkg.TaskRunStatusStarting),
		string(taskpkg.TaskRunStatusRunning),
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count active task-run session bindings for %q: %w", trimmedSessionID, err)
	}

	return count, nil
}

func (g *GlobalDB) normalizeTaskForCreate(record taskpkg.Task) (taskpkg.Task, error) {
	normalized := normalizeTaskRecord(record)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.Task{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskForUpdate(record taskpkg.Task) (taskpkg.Task, error) {
	normalized := normalizeTaskRecord(record)
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.Task{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskRunForCreate(run taskpkg.Run) (taskpkg.Run, error) {
	normalized := normalizeTaskRunRecord(run)
	if normalized.Attempt == 0 {
		normalized.Attempt = 1
	}
	if normalized.QueuedAt.IsZero() {
		normalized.QueuedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.Run{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskRunForUpdate(run taskpkg.Run) (taskpkg.Run, error) {
	normalized := normalizeTaskRunRecord(run)
	if err := normalized.Validate(); err != nil {
		return taskpkg.Run{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) ensureTaskCreateReferences(ctx context.Context, record taskpkg.Task) error {
	if err := taskpkg.ValidateScopeBinding(record.Scope, record.WorkspaceID, "task", "workspace_id"); err != nil {
		return err
	}
	if record.Scope == taskpkg.ScopeWorkspace {
		if err := g.ensureWorkspaceExists(ctx, record.WorkspaceID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(record.ParentTaskID) != "" {
		if err := g.ensureTaskExists(ctx, record.ParentTaskID); err != nil {
			return err
		}
	}
	return nil
}

func (g *GlobalDB) ensureWorkspaceExists(ctx context.Context, workspaceID string) error {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID == "" {
		return nil
	}

	var exists int
	if err := g.db.QueryRowContext(ctx, `SELECT 1 FROM workspaces WHERE id = ?`, trimmedID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return aghworkspace.ErrWorkspaceNotFound
		}
		return fmt.Errorf("store: lookup workspace %q: %w", trimmedID, err)
	}
	return nil
}

func (g *GlobalDB) ensureTaskExists(ctx context.Context, taskID string) error {
	trimmedID := strings.TrimSpace(taskID)
	if trimmedID == "" {
		return taskpkg.ErrTaskNotFound
	}

	var exists int
	if err := g.db.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ?`, trimmedID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ErrTaskNotFound
		}
		return fmt.Errorf("store: lookup task %q: %w", trimmedID, err)
	}
	return nil
}

func scanTaskRecord(scanner rowScanner) (taskpkg.Task, error) {
	var (
		record         taskpkg.Task
		identifier     sql.NullString
		scope          string
		workspaceID    sql.NullString
		parentTaskID   sql.NullString
		networkChannel sql.NullString
		description    sql.NullString
		priority       string
		maxAttempts    int
		status         string
		approvalPolicy string
		approvalState  string
		ownerKind      sql.NullString
		ownerRef       sql.NullString
		createdByKind  string
		originKind     string
		createdAtRaw   string
		updatedAtRaw   string
		closedAtRaw    sql.NullString
		metadataJSON   sql.NullString
	)
	if err := scanner.Scan(
		&record.ID,
		&identifier,
		&scope,
		&workspaceID,
		&parentTaskID,
		&networkChannel,
		&record.Title,
		&description,
		&priority,
		&maxAttempts,
		&status,
		&approvalPolicy,
		&approvalState,
		&ownerKind,
		&ownerRef,
		&createdByKind,
		&record.CreatedBy.Ref,
		&originKind,
		&record.Origin.Ref,
		&createdAtRaw,
		&updatedAtRaw,
		&closedAtRaw,
		&metadataJSON,
	); err != nil {
		return taskpkg.Task{}, fmt.Errorf("store: scan task: %w", err)
	}

	assignScannedTaskRecord(
		&record,
		identifier,
		scope,
		workspaceID,
		parentTaskID,
		networkChannel,
		description,
		priority,
		maxAttempts,
		status,
		approvalPolicy,
		approvalState,
		ownerKind,
		ownerRef,
		createdByKind,
		originKind,
	)
	if err := assignTaskRecordTimestamps(&record, createdAtRaw, updatedAtRaw, closedAtRaw); err != nil {
		return taskpkg.Task{}, err
	}
	if err := assignTaskMetadata(&record.Metadata, metadataJSON, "task.metadata_json"); err != nil {
		return taskpkg.Task{}, err
	}
	record = normalizeTaskRecord(record)
	if err := record.Validate(); err != nil {
		return taskpkg.Task{}, err
	}

	return record, nil
}

func scanTaskRunRecord(scanner rowScanner) (taskpkg.Run, error) {
	var (
		run            taskpkg.Run
		status         string
		claimedByKind  sql.NullString
		claimedByRef   sql.NullString
		sessionID      sql.NullString
		originKind     string
		idempotencyKey sql.NullString
		networkChannel sql.NullString
		queuedAtRaw    string
		claimedAtRaw   sql.NullString
		startedAtRaw   sql.NullString
		endedAtRaw     sql.NullString
		runErr         sql.NullString
		resultJSON     sql.NullString
	)
	if err := scanner.Scan(
		&run.ID,
		&run.TaskID,
		&status,
		&run.Attempt,
		&claimedByKind,
		&claimedByRef,
		&sessionID,
		&originKind,
		&run.Origin.Ref,
		&idempotencyKey,
		&networkChannel,
		&queuedAtRaw,
		&claimedAtRaw,
		&startedAtRaw,
		&endedAtRaw,
		&runErr,
		&resultJSON,
	); err != nil {
		return taskpkg.Run{}, fmt.Errorf("store: scan task run: %w", err)
	}

	assignScannedTaskRunRecord(
		&run,
		status,
		claimedByKind,
		claimedByRef,
		sessionID,
		originKind,
		idempotencyKey,
		networkChannel,
		runErr,
	)
	if err := assignTaskRunTimestamps(&run, queuedAtRaw, claimedAtRaw, startedAtRaw, endedAtRaw); err != nil {
		return taskpkg.Run{}, err
	}
	if err := assignTaskMetadata(&run.Result, resultJSON, "task_run.result_json"); err != nil {
		return taskpkg.Run{}, err
	}
	run = normalizeTaskRunRecord(run)
	if err := run.Validate(); err != nil {
		return taskpkg.Run{}, err
	}

	return run, nil
}

func assignScannedTaskRecord(
	record *taskpkg.Task,
	identifier sql.NullString,
	scope string,
	workspaceID sql.NullString,
	parentTaskID sql.NullString,
	networkChannel sql.NullString,
	description sql.NullString,
	priority string,
	maxAttempts int,
	status string,
	approvalPolicy string,
	approvalState string,
	ownerKind sql.NullString,
	ownerRef sql.NullString,
	createdByKind string,
	originKind string,
) {
	record.Identifier = taskNullStringValue(identifier)
	record.Scope = taskpkg.Scope(strings.TrimSpace(scope))
	record.WorkspaceID = taskNullStringValue(workspaceID)
	record.ParentTaskID = taskNullStringValue(parentTaskID)
	record.NetworkChannel = taskNullStringValue(networkChannel)
	record.Description = taskNullStringValue(description)
	record.Priority = taskpkg.Priority(strings.TrimSpace(priority))
	record.MaxAttempts = maxAttempts
	record.Status = taskpkg.Status(strings.TrimSpace(status))
	record.ApprovalPolicy = taskpkg.ApprovalPolicy(strings.TrimSpace(approvalPolicy))
	record.ApprovalState = taskpkg.ApprovalState(strings.TrimSpace(approvalState))
	record.CreatedBy.Kind = taskpkg.ActorKind(strings.TrimSpace(createdByKind))
	record.Origin.Kind = taskpkg.OriginKind(strings.TrimSpace(originKind))
	if ownerKind.Valid || ownerRef.Valid {
		record.Owner = &taskpkg.Ownership{
			Kind: taskpkg.OwnerKind(strings.TrimSpace(ownerKind.String)),
			Ref:  strings.TrimSpace(ownerRef.String),
		}
	}
}

func assignTaskRecordTimestamps(
	record *taskpkg.Task,
	createdAtRaw string,
	updatedAtRaw string,
	closedAtRaw sql.NullString,
) error {
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return err
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	if !closedAtRaw.Valid {
		return nil
	}
	closedAt, err := store.ParseTimestamp(closedAtRaw.String)
	if err != nil {
		return err
	}
	record.ClosedAt = closedAt
	return nil
}

func assignScannedTaskRunRecord(
	run *taskpkg.Run,
	status string,
	claimedByKind sql.NullString,
	claimedByRef sql.NullString,
	sessionID sql.NullString,
	originKind string,
	idempotencyKey sql.NullString,
	networkChannel sql.NullString,
	runErr sql.NullString,
) {
	run.Status = taskpkg.RunStatus(strings.TrimSpace(status))
	if claimedByKind.Valid || claimedByRef.Valid {
		run.ClaimedBy = &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKind(strings.TrimSpace(claimedByKind.String)),
			Ref:  strings.TrimSpace(claimedByRef.String),
		}
	}
	run.SessionID = taskNullStringValue(sessionID)
	run.Origin.Kind = taskpkg.OriginKind(strings.TrimSpace(originKind))
	run.IdempotencyKey = taskNullStringValue(idempotencyKey)
	run.NetworkChannel = taskNullStringValue(networkChannel)
	run.Error = taskNullStringValue(runErr)
}

func assignTaskRunTimestamps(
	run *taskpkg.Run,
	queuedAtRaw string,
	claimedAtRaw sql.NullString,
	startedAtRaw sql.NullString,
	endedAtRaw sql.NullString,
) error {
	queuedAt, err := store.ParseTimestamp(queuedAtRaw)
	if err != nil {
		return err
	}
	run.QueuedAt = queuedAt
	if err := assignNullableTaskTimestamp(&run.ClaimedAt, claimedAtRaw); err != nil {
		return err
	}
	if err := assignNullableTaskTimestamp(&run.StartedAt, startedAtRaw); err != nil {
		return err
	}
	return assignNullableTaskTimestamp(&run.EndedAt, endedAtRaw)
}

func assignNullableTaskTimestamp(target *time.Time, raw sql.NullString) error {
	if !raw.Valid {
		return nil
	}
	parsed, err := store.ParseTimestamp(raw.String)
	if err != nil {
		return err
	}
	*target = parsed
	return nil
}

func assignTaskMetadata(target *json.RawMessage, raw sql.NullString, field string) error {
	metadata, err := decodeTaskJSON(raw, field)
	if err != nil {
		return err
	}
	*target = metadata
	return nil
}

func normalizeTaskRecord(record taskpkg.Task) taskpkg.Task {
	normalized := record
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Identifier = strings.TrimSpace(normalized.Identifier)
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.ParentTaskID = strings.TrimSpace(normalized.ParentTaskID)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.Status = normalized.Status.Normalize()
	normalized.CreatedBy.Kind = normalized.CreatedBy.Kind.Normalize()
	normalized.CreatedBy.Ref = strings.TrimSpace(normalized.CreatedBy.Ref)
	normalized.Origin.Kind = normalized.Origin.Kind.Normalize()
	normalized.Origin.Ref = strings.TrimSpace(normalized.Origin.Ref)
	if normalized.Owner != nil {
		owner := *normalized.Owner
		owner.Kind = owner.Kind.Normalize()
		owner.Ref = strings.TrimSpace(owner.Ref)
		if owner.IsZero() {
			normalized.Owner = nil
		} else {
			normalized.Owner = &owner
		}
	}
	normalized.Metadata = normalizeTaskJSON(normalized.Metadata)
	normalized.Priority = taskpkg.DefaultPriority
	if record.Priority.Normalize() != "" {
		normalized.Priority = record.Priority.Normalize()
	}
	normalized.MaxAttempts = taskpkg.DefaultTaskMaxAttempts
	if record.MaxAttempts != 0 {
		normalized.MaxAttempts = record.MaxAttempts
	}
	normalized.ApprovalPolicy = taskpkg.DefaultApprovalPolicy
	if record.ApprovalPolicy.Normalize() != "" {
		normalized.ApprovalPolicy = record.ApprovalPolicy.Normalize()
	}
	normalized.ApprovalState = taskpkg.ApprovalStateNotRequired
	if record.ApprovalState.Normalize() != "" {
		normalized.ApprovalState = record.ApprovalState.Normalize()
	} else if normalized.ApprovalPolicy == taskpkg.ApprovalPolicyManual {
		normalized.ApprovalState = taskpkg.ApprovalStatePending
	}
	if !normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = normalized.CreatedAt.UTC()
	}
	if !normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.UpdatedAt.UTC()
	}
	if !normalized.ClosedAt.IsZero() {
		normalized.ClosedAt = normalized.ClosedAt.UTC()
	}
	return normalized
}

func normalizeTaskRunRecord(run taskpkg.Run) taskpkg.Run {
	normalized := run
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Status = normalized.Status.Normalize()
	if normalized.ClaimedBy != nil {
		claimedBy := *normalized.ClaimedBy
		claimedBy.Kind = claimedBy.Kind.Normalize()
		claimedBy.Ref = strings.TrimSpace(claimedBy.Ref)
		normalized.ClaimedBy = &claimedBy
	}
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.Origin.Kind = normalized.Origin.Kind.Normalize()
	normalized.Origin.Ref = strings.TrimSpace(normalized.Origin.Ref)
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	normalized.Error = strings.TrimSpace(normalized.Error)
	normalized.Result = normalizeTaskJSON(normalized.Result)
	if !normalized.QueuedAt.IsZero() {
		normalized.QueuedAt = normalized.QueuedAt.UTC()
	}
	if !normalized.ClaimedAt.IsZero() {
		normalized.ClaimedAt = normalized.ClaimedAt.UTC()
	}
	if !normalized.StartedAt.IsZero() {
		normalized.StartedAt = normalized.StartedAt.UTC()
	}
	if !normalized.EndedAt.IsZero() {
		normalized.EndedAt = normalized.EndedAt.UTC()
	}
	return normalized
}

func normalizeTaskQuery(query taskpkg.Query) taskpkg.Query {
	normalized := query
	normalized.Scope = normalized.Scope.Normalize()
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.Status = normalized.Status.Normalize()
	normalized.Priority = normalized.Priority.Normalize()
	normalized.ApprovalState = normalized.ApprovalState.Normalize()
	normalized.OwnerKind = normalized.OwnerKind.Normalize()
	normalized.OwnerRef = strings.TrimSpace(normalized.OwnerRef)
	normalized.ParentTaskID = strings.TrimSpace(normalized.ParentTaskID)
	normalized.NetworkChannel = strings.TrimSpace(normalized.NetworkChannel)
	return normalized
}

func normalizeTaskRunQuery(query taskpkg.RunQuery) taskpkg.RunQuery {
	normalized := query
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Status = normalized.Status.Normalize()
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	return normalized
}

func taskSummaryFromRecord(record taskpkg.Task) taskpkg.Summary {
	return taskpkg.Summary{
		ID:             record.ID,
		Identifier:     record.Identifier,
		Scope:          record.Scope,
		WorkspaceID:    record.WorkspaceID,
		ParentTaskID:   record.ParentTaskID,
		NetworkChannel: record.NetworkChannel,
		Title:          record.Title,
		Priority:       record.Priority,
		MaxAttempts:    record.MaxAttempts,
		Status:         record.Status,
		ApprovalPolicy: record.ApprovalPolicy,
		ApprovalState:  record.ApprovalState,
		Draft:          record.Status.Normalize() == taskpkg.TaskStatusDraft,
		Owner:          record.Owner,
		CreatedBy:      record.CreatedBy,
		Origin:         record.Origin,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		ClosedAt:       record.ClosedAt,
	}
}

func requireTaskValue(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("store: %s is required", label)
	}
	return trimmed, nil
}

func taskOwnerKindValue(owner *taskpkg.Ownership) any {
	if owner == nil {
		return nil
	}
	return string(owner.Kind)
}

func taskOwnerRefValue(owner *taskpkg.Ownership) any {
	if owner == nil {
		return nil
	}
	return owner.Ref
}

func taskActorKindValue(actor *taskpkg.ActorIdentity) any {
	if actor == nil {
		return nil
	}
	return string(actor.Kind)
}

func taskActorRefValue(actor *taskpkg.ActorIdentity) any {
	if actor == nil {
		return nil
	}
	return actor.Ref
}

func nullableTaskTimestamp(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value)
}

func normalizeTaskJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil
	}
	return json.RawMessage(trimmed)
}

func nullableTaskJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func decodeTaskJSON(raw sql.NullString, label string) (json.RawMessage, error) {
	if !raw.Valid {
		return nil, nil
	}
	trimmed := strings.TrimSpace(raw.String)
	if trimmed == "" {
		return nil, nil
	}
	value := json.RawMessage(trimmed)
	if !json.Valid(value) {
		return nil, fmt.Errorf("store: decode %s: invalid JSON", label)
	}
	return value, nil
}

func taskNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}
