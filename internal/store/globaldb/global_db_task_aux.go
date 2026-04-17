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

var _ taskpkg.DependencyStore = (*GlobalDB)(nil)
var _ taskpkg.EventStore = (*GlobalDB)(nil)
var _ taskpkg.IdempotencyStore = (*GlobalDB)(nil)
var _ taskpkg.TriageStore = (*GlobalDB)(nil)

type taskSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// GetTaskTriageState returns the durable actor-scoped triage state for one task.
func (g *GlobalDB) GetTaskTriageState(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorIdentity,
) (taskpkg.TriageState, error) {
	if err := g.checkReady(ctx, "get task triage state"); err != nil {
		return taskpkg.TriageState{}, err
	}

	trimmedTaskID, normalizedActor, err := normalizeTaskTriageLookup(taskID, actor)
	if err != nil {
		return taskpkg.TriageState{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			task_id, actor_kind, actor_ref, is_read, archived, dismissed, last_seen_activity_at, updated_at
		 FROM task_triage_state
		 WHERE task_id = ? AND actor_kind = ? AND actor_ref = ?`,
		trimmedTaskID,
		string(normalizedActor.Kind),
		normalizedActor.Ref,
	)

	record, err := scanTaskTriageStateRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.TriageState{}, taskpkg.ErrTaskTriageStateNotFound
		}
		return taskpkg.TriageState{}, err
	}
	return record, nil
}

// UpsertTaskTriageState inserts or replaces one durable actor-scoped triage state.
func (g *GlobalDB) UpsertTaskTriageState(ctx context.Context, state taskpkg.TriageState) error {
	if err := g.checkReady(ctx, "upsert task triage state"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskTriageStateForUpsert(state)
	if err != nil {
		return err
	}
	if err := g.ensureTaskExists(ctx, normalized.TaskID); err != nil {
		return err
	}

	_, err = g.db.ExecContext(
		ctx,
		`INSERT INTO task_triage_state (
			task_id, actor_kind, actor_ref, is_read, archived, dismissed, last_seen_activity_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id, actor_kind, actor_ref) DO UPDATE SET
			is_read = excluded.is_read,
			archived = excluded.archived,
			dismissed = excluded.dismissed,
			last_seen_activity_at = excluded.last_seen_activity_at,
			updated_at = excluded.updated_at`,
		normalized.TaskID,
		string(normalized.Actor.Kind),
		normalized.Actor.Ref,
		normalized.Read,
		normalized.Archived,
		normalized.Dismissed,
		nullableTaskTimestamp(normalized.LastSeenActivityAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf(
			"store: upsert task triage state for task %q actor %q/%q: %w",
			normalized.TaskID,
			normalized.Actor.Kind,
			normalized.Actor.Ref,
			err,
		)
	}

	return nil
}

// CreateDependency inserts one durable task-dependency edge under a single SQLite write lock.
func (g *GlobalDB) CreateDependency(ctx context.Context, dependency taskpkg.Dependency) error {
	if err := g.checkReady(ctx, "create task dependency"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskDependencyForCreate(dependency)
	if err != nil {
		return err
	}

	return g.withTaskImmediateTransaction(ctx, "create task dependency", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalized.TaskID); err != nil {
			return err
		}
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalized.DependsOnTaskID); err != nil {
			return err
		}

		exists, err := g.taskDependencyExists(ctx, exec, normalized)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf(
				"%w: dependency edge %q -> %q already exists",
				taskpkg.ErrValidation,
				normalized.TaskID,
				normalized.DependsOnTaskID,
			)
		}

		count, err := g.countDependenciesWithExecutor(ctx, exec, normalized.TaskID)
		if err != nil {
			return err
		}
		if err := taskpkg.ValidateDependencyCount(count + 1); err != nil {
			return err
		}

		hasPath, err := g.hasDependencyPathWithExecutor(ctx, exec, normalized.DependsOnTaskID, normalized.TaskID)
		if err != nil {
			return err
		}
		if hasPath {
			return fmt.Errorf(
				"%w: adding dependency %q -> %q would create a cycle",
				taskpkg.ErrCycleDetected,
				normalized.TaskID,
				normalized.DependsOnTaskID,
			)
		}

		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO task_dependencies (task_id, depends_on_task_id, kind, created_at)
			 VALUES (?, ?, ?, ?)`,
			normalized.TaskID,
			normalized.DependsOnTaskID,
			string(normalized.Kind),
			store.FormatTimestamp(normalized.CreatedAt),
		); err != nil {
			return fmt.Errorf(
				"store: create task dependency %q -> %q: %w",
				normalized.TaskID,
				normalized.DependsOnTaskID,
				err,
			)
		}

		return nil
	})
}

// DeleteDependency removes one persisted dependency edge.
func (g *GlobalDB) DeleteDependency(ctx context.Context, taskID string, dependsOnID string) error {
	if err := g.checkReady(ctx, "delete task dependency"); err != nil {
		return err
	}

	trimmedTaskID, err := requireTaskValue(taskID, "task dependency task id")
	if err != nil {
		return err
	}
	trimmedDependsOnID, err := requireTaskValue(dependsOnID, "task dependency depends_on_task_id")
	if err != nil {
		return err
	}

	result, err := g.db.ExecContext(
		ctx,
		`DELETE FROM task_dependencies WHERE task_id = ? AND depends_on_task_id = ?`,
		trimmedTaskID,
		trimmedDependsOnID,
	)
	if err != nil {
		return fmt.Errorf(
			"store: delete task dependency %q -> %q: %w",
			trimmedTaskID,
			trimmedDependsOnID,
			err,
		)
	}

	return requireRowsAffected(
		result,
		taskpkg.ErrTaskDependencyNotFound,
		trimmedTaskID+"->"+trimmedDependsOnID,
		"task dependency",
	)
}

// ListDependencies returns the persisted dependency edges for one task.
func (g *GlobalDB) ListDependencies(ctx context.Context, taskID string) ([]taskpkg.Dependency, error) {
	if err := g.checkReady(ctx, "list task dependencies"); err != nil {
		return nil, err
	}

	trimmedTaskID, err := requireTaskValue(taskID, "task dependency task id")
	if err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT task_id, depends_on_task_id, kind, created_at
		 FROM task_dependencies
		 WHERE task_id = ?
		 ORDER BY created_at ASC, depends_on_task_id ASC`,
		trimmedTaskID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query task dependencies for %q: %w", trimmedTaskID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	dependencies := make([]taskpkg.Dependency, 0)
	for rows.Next() {
		record, scanErr := scanTaskDependencyRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		dependencies = append(dependencies, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task dependencies for %q: %w", trimmedTaskID, err)
	}

	return dependencies, nil
}

// ListDependents returns persisted dependency edges that point at one task.
func (g *GlobalDB) ListDependents(ctx context.Context, dependsOnTaskID string) ([]taskpkg.Dependency, error) {
	if err := g.checkReady(ctx, "list task dependents"); err != nil {
		return nil, err
	}

	trimmedDependsOnID, err := requireTaskValue(dependsOnTaskID, "task dependent depends_on_task_id")
	if err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT task_id, depends_on_task_id, kind, created_at
		 FROM task_dependencies
		 WHERE depends_on_task_id = ?
		 ORDER BY created_at ASC, task_id ASC`,
		trimmedDependsOnID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query task dependents for %q: %w", trimmedDependsOnID, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	dependents := make([]taskpkg.Dependency, 0)
	for rows.Next() {
		record, scanErr := scanTaskDependencyRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		dependents = append(dependents, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task dependents for %q: %w", trimmedDependsOnID, err)
	}

	return dependents, nil
}

// CountDependencies reports how many dependency edges are stored for one task.
func (g *GlobalDB) CountDependencies(ctx context.Context, taskID string) (int, error) {
	if err := g.checkReady(ctx, "count task dependencies"); err != nil {
		return 0, err
	}

	trimmedTaskID, err := requireTaskValue(taskID, "task dependency task id")
	if err != nil {
		return 0, err
	}

	return g.countDependenciesWithExecutor(ctx, g.db, trimmedTaskID)
}

// HasDependencyPath reports whether the dependency graph already contains a path from one task to another.
func (g *GlobalDB) HasDependencyPath(ctx context.Context, fromTaskID string, toTaskID string) (bool, error) {
	if err := g.checkReady(ctx, "check task dependency path"); err != nil {
		return false, err
	}

	trimmedFromTaskID, err := requireTaskValue(fromTaskID, "task dependency path from_task_id")
	if err != nil {
		return false, err
	}
	trimmedToTaskID, err := requireTaskValue(toTaskID, "task dependency path to_task_id")
	if err != nil {
		return false, err
	}

	return g.hasDependencyPathWithExecutor(ctx, g.db, trimmedFromTaskID, trimmedToTaskID)
}

// CreateTaskEvent inserts one immutable task audit event.
func (g *GlobalDB) CreateTaskEvent(ctx context.Context, event taskpkg.Event) error {
	if err := g.checkReady(ctx, "create task event"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskEventForCreate(event)
	if err != nil {
		return err
	}
	if err := g.ensureTaskExists(ctx, normalized.TaskID); err != nil {
		return err
	}
	if strings.TrimSpace(normalized.RunID) != "" {
		run, err := g.getTaskRunWithExecutor(ctx, g.db, normalized.RunID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(run.TaskID) != normalized.TaskID {
			return fmt.Errorf(
				"%w: task_event.run_id %q does not belong to task %q",
				taskpkg.ErrValidation,
				normalized.RunID,
				normalized.TaskID,
			)
		}
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO task_events (
			id, task_id, run_id, event_type, actor_kind, actor_ref, origin_kind, origin_ref, payload_json, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.ID,
		normalized.TaskID,
		store.NullableString(normalized.RunID),
		normalized.EventType,
		string(normalized.Actor.Kind),
		normalized.Actor.Ref,
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		nullableTaskJSON(normalized.Payload),
		store.FormatTimestamp(normalized.Timestamp),
	); err != nil {
		return fmt.Errorf("store: create task event %q: %w", normalized.ID, err)
	}

	return nil
}

// ListTaskEvents returns persisted audit events that match the supplied filters.
func (g *GlobalDB) ListTaskEvents(ctx context.Context, query taskpkg.EventQuery) ([]taskpkg.Event, error) {
	if err := g.checkReady(ctx, "list task events"); err != nil {
		return nil, err
	}
	if err := query.Validate("task_event_query"); err != nil {
		return nil, err
	}

	normalized := normalizeTaskEventQuery(query)
	sqlQuery := `SELECT
		id, task_id, run_id, event_type, actor_kind, actor_ref, origin_kind, origin_ref, payload_json, timestamp
		FROM task_events`
	where, args := store.BuildClauses(
		store.StringClause("task_id", normalized.TaskID),
		store.StringClause("run_id", normalized.RunID),
		store.StringClause("event_type", normalized.EventType),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY timestamp DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	events := make([]taskpkg.Event, 0)
	for rows.Next() {
		event, scanErr := scanTaskEventRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task events: %w", err)
	}

	return events, nil
}

// GetTaskRunByIdempotencyKey returns the original persisted run bound to one origin-scoped idempotency key.
func (g *GlobalDB) GetTaskRunByIdempotencyKey(
	ctx context.Context,
	key string,
	origin taskpkg.Origin,
) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "get task run by idempotency key"); err != nil {
		return taskpkg.Run{}, err
	}

	trimmedKey, normalizedOrigin, err := normalizeTaskRunIdempotencyLookup(key, origin)
	if err != nil {
		return taskpkg.Run{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			tr.id, tr.task_id, tr.status, tr.attempt, tr.claimed_by_kind, tr.claimed_by_ref, tr.session_id,
			tr.origin_kind, tr.origin_ref, tr.idempotency_key, tr.network_channel, tr.queued_at, tr.claimed_at,
			tr.started_at, tr.ended_at, tr.error, tr.result_json
		 FROM task_run_idempotency tri
		 JOIN task_runs tr ON tr.id = tri.run_id
		 WHERE tri.idempotency_key = ? AND tri.origin_kind = ? AND tri.origin_ref = ?`,
		trimmedKey,
		string(normalizedOrigin.Kind),
		normalizedOrigin.Ref,
	)

	run, err := scanTaskRunRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.Run{}, taskpkg.ErrTaskRunIdempotencyNotFound
		}
		return taskpkg.Run{}, err
	}
	return run, nil
}

// SaveTaskRunIdempotency inserts one origin-scoped idempotency binding for a persisted run.
func (g *GlobalDB) SaveTaskRunIdempotency(ctx context.Context, record taskpkg.RunIdempotency) error {
	if err := g.checkReady(ctx, "save task run idempotency"); err != nil {
		return err
	}

	normalized, err := g.normalizeTaskRunIdempotencyForCreate(record)
	if err != nil {
		return err
	}

	run, err := g.getTaskRunWithExecutor(ctx, g.db, normalized.RunID)
	if err != nil {
		return err
	}
	if !taskOriginsEqual(run.Origin, normalized.Origin) {
		return fmt.Errorf(
			"%w: task_run_idempotency origin %q/%q does not match run origin %q/%q",
			taskpkg.ErrValidation,
			normalized.Origin.Kind,
			normalized.Origin.Ref,
			run.Origin.Kind,
			run.Origin.Ref,
		)
	}

	result, err := g.db.ExecContext(
		ctx,
		`INSERT INTO task_run_idempotency (
			idempotency_key, origin_kind, origin_ref, run_id, created_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(idempotency_key, origin_kind, origin_ref) DO NOTHING`,
		normalized.IdempotencyKey,
		string(normalized.Origin.Kind),
		normalized.Origin.Ref,
		normalized.RunID,
		store.FormatTimestamp(normalized.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: save task run idempotency %q: %w", normalized.IdempotencyKey, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for task run idempotency %q: %w", normalized.IdempotencyKey, err)
	}
	if rowsAffected > 0 {
		return nil
	}

	current, err := getTaskRunIdempotencyRecord(ctx, g.db, normalized.IdempotencyKey, normalized.Origin)
	if err != nil {
		return err
	}
	if current.RunID != normalized.RunID {
		return fmt.Errorf(
			"%w: idempotency key %q is already bound to run %q",
			taskpkg.ErrValidation,
			normalized.IdempotencyKey,
			current.RunID,
		)
	}

	return nil
}

func (g *GlobalDB) normalizeTaskDependencyForCreate(record taskpkg.Dependency) (taskpkg.Dependency, error) {
	normalized := normalizeTaskDependencyRecord(record)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.Dependency{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskTriageStateForUpsert(
	record taskpkg.TriageState,
) (taskpkg.TriageState, error) {
	normalized := normalizeTaskTriageStateRecord(record)
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.TriageState{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskEventForCreate(record taskpkg.Event) (taskpkg.Event, error) {
	normalized := normalizeTaskEventRecord(record)
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.Event{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeTaskRunIdempotencyForCreate(
	record taskpkg.RunIdempotency,
) (taskpkg.RunIdempotency, error) {
	normalized := normalizeTaskRunIdempotencyRecord(record)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return taskpkg.RunIdempotency{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) withTaskImmediateTransaction(
	ctx context.Context,
	action string,
	run func(exec taskSQLExecutor) error,
) (err error) {
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open connection for %s: %w", action, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin immediate %s transaction: %w", action, err)
	}

	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, action))
		}
	}()

	if err := run(conn); err != nil {
		return err
	}

	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit %s transaction: %w", action, err)
	}

	finished = true
	return nil
}

func (g *GlobalDB) ensureTaskExistsWithExecutor(ctx context.Context, exec taskSQLExecutor, taskID string) error {
	trimmedID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return err
	}

	var exists int
	if err := exec.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ?`, trimmedID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ErrTaskNotFound
		}
		return fmt.Errorf("store: lookup task %q: %w", trimmedID, err)
	}

	return nil
}

func (g *GlobalDB) getTaskRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
) (taskpkg.Run, error) {
	trimmedRunID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return taskpkg.Run{}, err
	}

	row := exec.QueryRowContext(
		ctx,
		`SELECT
			id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id, origin_kind, origin_ref,
			idempotency_key, network_channel, queued_at, claimed_at, started_at, ended_at, error, result_json
		 FROM task_runs
		 WHERE id = ?`,
		trimmedRunID,
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

func (g *GlobalDB) countDependenciesWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (int, error) {
	var count int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM task_dependencies WHERE task_id = ?`,
		taskID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count task dependencies for %q: %w", taskID, err)
	}
	return count, nil
}

func (g *GlobalDB) taskDependencyExists(
	ctx context.Context,
	exec taskSQLExecutor,
	dependency taskpkg.Dependency,
) (bool, error) {
	var exists int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM task_dependencies
		 WHERE task_id = ? AND depends_on_task_id = ? AND kind = ?`,
		dependency.TaskID,
		dependency.DependsOnTaskID,
		string(dependency.Kind),
	).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf(
			"store: lookup task dependency %q -> %q: %w",
			dependency.TaskID,
			dependency.DependsOnTaskID,
			err,
		)
	}
	return true, nil
}

func (g *GlobalDB) hasDependencyPathWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	fromTaskID string,
	toTaskID string,
) (bool, error) {
	var exists int
	if err := exec.QueryRowContext(
		ctx,
		`WITH RECURSIVE dependency_path(node_id) AS (
			SELECT depends_on_task_id
			  FROM task_dependencies
			 WHERE task_id = ?
			UNION
			SELECT td.depends_on_task_id
			  FROM task_dependencies td
			  JOIN dependency_path path ON td.task_id = path.node_id
		)
		SELECT EXISTS(SELECT 1 FROM dependency_path WHERE node_id = ?)`,
		fromTaskID,
		toTaskID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("store: query task dependency path %q -> %q: %w", fromTaskID, toTaskID, err)
	}
	return exists == 1, nil
}

func getTaskRunIdempotencyRecord(
	ctx context.Context,
	exec taskSQLExecutor,
	key string,
	origin taskpkg.Origin,
) (taskpkg.RunIdempotency, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT idempotency_key, run_id, origin_kind, origin_ref, created_at
		 FROM task_run_idempotency
		 WHERE idempotency_key = ? AND origin_kind = ? AND origin_ref = ?`,
		key,
		string(origin.Kind),
		origin.Ref,
	)

	record, err := scanTaskRunIdempotencyRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.RunIdempotency{}, taskpkg.ErrTaskRunIdempotencyNotFound
		}
		return taskpkg.RunIdempotency{}, err
	}
	return record, nil
}

func normalizeTaskTriageLookup(
	taskID string,
	actor taskpkg.ActorIdentity,
) (string, taskpkg.ActorIdentity, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task triage task id")
	if err != nil {
		return "", taskpkg.ActorIdentity{}, err
	}

	normalizedActor := taskpkg.ActorIdentity{
		Kind: actor.Kind.Normalize(),
		Ref:  strings.TrimSpace(actor.Ref),
	}
	if err := normalizedActor.Validate("task_triage_state.actor"); err != nil {
		return "", taskpkg.ActorIdentity{}, err
	}

	return trimmedTaskID, normalizedActor, nil
}

func normalizeTaskDependencyRecord(record taskpkg.Dependency) taskpkg.Dependency {
	normalized := record
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.DependsOnTaskID = strings.TrimSpace(normalized.DependsOnTaskID)
	normalized.Kind = normalized.Kind.Normalize()
	if !normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = normalized.CreatedAt.UTC()
	}
	return normalized
}

func normalizeTaskTriageStateRecord(record taskpkg.TriageState) taskpkg.TriageState {
	normalized := record
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.Actor.Kind = normalized.Actor.Kind.Normalize()
	normalized.Actor.Ref = strings.TrimSpace(normalized.Actor.Ref)
	if !normalized.LastSeenActivityAt.IsZero() {
		normalized.LastSeenActivityAt = normalized.LastSeenActivityAt.UTC()
	}
	if !normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.UpdatedAt.UTC()
	}
	return normalized
}

func normalizeTaskEventRecord(record taskpkg.Event) taskpkg.Event {
	normalized := record
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.EventType = strings.TrimSpace(normalized.EventType)
	normalized.Actor.Kind = normalized.Actor.Kind.Normalize()
	normalized.Actor.Ref = strings.TrimSpace(normalized.Actor.Ref)
	normalized.Origin.Kind = normalized.Origin.Kind.Normalize()
	normalized.Origin.Ref = strings.TrimSpace(normalized.Origin.Ref)
	normalized.Payload = normalizeTaskJSON(normalized.Payload)
	if !normalized.Timestamp.IsZero() {
		normalized.Timestamp = normalized.Timestamp.UTC()
	}
	return normalized
}

func normalizeTaskEventQuery(query taskpkg.EventQuery) taskpkg.EventQuery {
	normalized := query
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.EventType = strings.TrimSpace(normalized.EventType)
	return normalized
}

func normalizeTaskRunIdempotencyLookup(key string, origin taskpkg.Origin) (string, taskpkg.Origin, error) {
	trimmedKey, err := requireTaskValue(key, "task run idempotency key")
	if err != nil {
		return "", taskpkg.Origin{}, err
	}

	normalizedOrigin := taskpkg.Origin{
		Kind: origin.Kind.Normalize(),
		Ref:  strings.TrimSpace(origin.Ref),
	}
	if err := normalizedOrigin.Validate("task_run_idempotency.origin"); err != nil {
		return "", taskpkg.Origin{}, err
	}

	return trimmedKey, normalizedOrigin, nil
}

func normalizeTaskRunIdempotencyRecord(record taskpkg.RunIdempotency) taskpkg.RunIdempotency {
	normalized := record
	normalized.IdempotencyKey = strings.TrimSpace(normalized.IdempotencyKey)
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.Origin.Kind = normalized.Origin.Kind.Normalize()
	normalized.Origin.Ref = strings.TrimSpace(normalized.Origin.Ref)
	if !normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = normalized.CreatedAt.UTC()
	}
	return normalized
}

func scanTaskDependencyRecord(scanner rowScanner) (taskpkg.Dependency, error) {
	var (
		record       taskpkg.Dependency
		kind         string
		createdAtRaw string
	)
	if err := scanner.Scan(
		&record.TaskID,
		&record.DependsOnTaskID,
		&kind,
		&createdAtRaw,
	); err != nil {
		return taskpkg.Dependency{}, fmt.Errorf("store: scan task dependency: %w", err)
	}

	record.Kind = taskpkg.DependencyKind(strings.TrimSpace(kind))
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return taskpkg.Dependency{}, err
	}
	record.CreatedAt = createdAt
	record = normalizeTaskDependencyRecord(record)
	if err := record.Validate(); err != nil {
		return taskpkg.Dependency{}, err
	}

	return record, nil
}

func scanTaskTriageStateRecord(scanner rowScanner) (taskpkg.TriageState, error) {
	var (
		record                taskpkg.TriageState
		actorKind             string
		lastSeenActivityAtRaw sql.NullString
		updatedAtRaw          string
	)
	if err := scanner.Scan(
		&record.TaskID,
		&actorKind,
		&record.Actor.Ref,
		&record.Read,
		&record.Archived,
		&record.Dismissed,
		&lastSeenActivityAtRaw,
		&updatedAtRaw,
	); err != nil {
		return taskpkg.TriageState{}, fmt.Errorf("store: scan task triage state: %w", err)
	}

	record.Actor.Kind = taskpkg.ActorKind(strings.TrimSpace(actorKind))
	if err := assignNullableTaskTimestamp(&record.LastSeenActivityAt, lastSeenActivityAtRaw); err != nil {
		return taskpkg.TriageState{}, fmt.Errorf("store: parse task triage last_seen_activity_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return taskpkg.TriageState{}, fmt.Errorf("store: parse task triage updated_at: %w", err)
	}
	record.UpdatedAt = updatedAt
	record = normalizeTaskTriageStateRecord(record)
	if err := record.Validate(); err != nil {
		return taskpkg.TriageState{}, err
	}

	return record, nil
}

func scanTaskEventRecord(scanner rowScanner) (taskpkg.Event, error) {
	var (
		record      taskpkg.Event
		runID       sql.NullString
		actorKind   string
		originKind  string
		payloadJSON sql.NullString
		timestamp   string
	)
	if err := scanner.Scan(
		&record.ID,
		&record.TaskID,
		&runID,
		&record.EventType,
		&actorKind,
		&record.Actor.Ref,
		&originKind,
		&record.Origin.Ref,
		&payloadJSON,
		&timestamp,
	); err != nil {
		return taskpkg.Event{}, fmt.Errorf("store: scan task event: %w", err)
	}

	record.RunID = taskNullStringValue(runID)
	record.Actor.Kind = taskpkg.ActorKind(strings.TrimSpace(actorKind))
	record.Origin.Kind = taskpkg.OriginKind(strings.TrimSpace(originKind))
	payload, err := decodeTaskJSON(payloadJSON, "task_event.payload_json")
	if err != nil {
		return taskpkg.Event{}, err
	}
	record.Payload = payload

	parsedTimestamp, err := store.ParseTimestamp(timestamp)
	if err != nil {
		return taskpkg.Event{}, err
	}
	record.Timestamp = parsedTimestamp
	record = normalizeTaskEventRecord(record)
	if err := record.Validate(); err != nil {
		return taskpkg.Event{}, err
	}

	return record, nil
}

func scanTaskRunIdempotencyRecord(scanner rowScanner) (taskpkg.RunIdempotency, error) {
	var (
		record       taskpkg.RunIdempotency
		originKind   string
		createdAtRaw string
	)
	if err := scanner.Scan(
		&record.IdempotencyKey,
		&record.RunID,
		&originKind,
		&record.Origin.Ref,
		&createdAtRaw,
	); err != nil {
		return taskpkg.RunIdempotency{}, fmt.Errorf("store: scan task run idempotency: %w", err)
	}

	record.Origin.Kind = taskpkg.OriginKind(strings.TrimSpace(originKind))
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return taskpkg.RunIdempotency{}, err
	}
	record.CreatedAt = createdAt
	record = normalizeTaskRunIdempotencyRecord(record)
	if err := record.Validate(); err != nil {
		return taskpkg.RunIdempotency{}, err
	}

	return record, nil
}

func taskOriginsEqual(left taskpkg.Origin, right taskpkg.Origin) bool {
	return left.Kind.Normalize() == right.Kind.Normalize() &&
		strings.TrimSpace(left.Ref) == strings.TrimSpace(right.Ref)
}
