package globaldb

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

var _ taskpkg.DependencyStore = (*GlobalDB)(nil)
var _ taskpkg.EventStore = (*GlobalDB)(nil)
var _ taskpkg.EventSequenceStore = (*GlobalDB)(nil)
var _ taskpkg.IdempotencyStore = (*GlobalDB)(nil)
var _ taskpkg.TriageStore = (*GlobalDB)(nil)

type taskSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type queuedRunReservationInput struct {
	taskID           string
	runID            string
	idempotencyKey   string
	origin           taskpkg.Origin
	requestedChannel string
	metadata         json.RawMessage
	queuedAt         time.Time
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
			task_id, actor_kind, actor_id, is_read, archived, dismissed, last_seen_activity_at, updated_at
		 FROM task_triage_state
		 WHERE task_id = ? AND actor_kind = ? AND actor_id = ?`,
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
			task_id, actor_kind, actor_id, is_read, archived, dismissed, last_seen_activity_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id, actor_kind, actor_id) DO UPDATE SET
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

// ListTaskTriageStates returns all durable triage states persisted for one actor.
func (g *GlobalDB) ListTaskTriageStates(
	ctx context.Context,
	actor taskpkg.ActorIdentity,
) ([]taskpkg.TriageState, error) {
	if err := g.checkReady(ctx, "list task triage states"); err != nil {
		return nil, err
	}

	normalizedActor, err := normalizeTaskTriageActor(actor)
	if err != nil {
		return nil, err
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			task_id, actor_kind, actor_id, is_read, archived, dismissed, last_seen_activity_at, updated_at
		 FROM task_triage_state
		 WHERE actor_kind = ? AND actor_id = ?
		 ORDER BY updated_at DESC, task_id ASC`,
		string(normalizedActor.Kind),
		normalizedActor.Ref,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"store: query task triage states for actor %q/%q: %w",
			normalizedActor.Kind,
			normalizedActor.Ref,
			err,
		)
	}

	states := make([]taskpkg.TriageState, 0)
	for rows.Next() {
		record, scanErr := scanTaskTriageStateRecord(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task triage state query")
		}
		states = append(states, record)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(rows, fmt.Errorf(
			"store: iterate task triage states for actor %q/%q: %w",
			normalizedActor.Kind,
			normalizedActor.Ref,
			err,
		), "task triage state query")
	}
	if err := joinRowsCloseError(rows, nil, "task triage state query"); err != nil {
		return nil, err
	}

	return states, nil
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

	return g.listDependenciesWithExecutor(ctx, g.db, taskID)
}

func (g *GlobalDB) listDependenciesWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) ([]taskpkg.Dependency, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task dependency task id")
	if err != nil {
		return nil, err
	}

	rows, err := exec.QueryContext(
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

	dependencies := make([]taskpkg.Dependency, 0)
	for rows.Next() {
		record, scanErr := scanTaskDependencyRecord(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task dependency query")
		}
		dependencies = append(dependencies, record)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate task dependencies for %q: %w", trimmedTaskID, err),
			"task dependency query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "task dependency query"); err != nil {
		return nil, err
	}

	return dependencies, nil
}

// ListDependents returns persisted dependency edges that point at one task.
func (g *GlobalDB) ListDependents(ctx context.Context, dependsOnTaskID string) ([]taskpkg.Dependency, error) {
	if err := g.checkReady(ctx, "list task dependents"); err != nil {
		return nil, err
	}

	return g.listDependentsWithExecutor(ctx, g.db, dependsOnTaskID)
}

func (g *GlobalDB) listDependentsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	dependsOnTaskID string,
) ([]taskpkg.Dependency, error) {
	trimmedDependsOnID, err := requireTaskValue(dependsOnTaskID, "task dependent depends_on_task_id")
	if err != nil {
		return nil, err
	}

	rows, err := exec.QueryContext(
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

	dependents := make([]taskpkg.Dependency, 0)
	for rows.Next() {
		record, scanErr := scanTaskDependencyRecord(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task dependent query")
		}
		dependents = append(dependents, record)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate task dependents for %q: %w", trimmedDependsOnID, err),
			"task dependent query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "task dependent query"); err != nil {
		return nil, err
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

	return g.withTaskImmediateTransaction(ctx, "create task event", func(exec taskSQLExecutor) error {
		if err := g.ensureTaskExistsWithExecutor(ctx, exec, normalized.TaskID); err != nil {
			return err
		}
		if strings.TrimSpace(normalized.RunID) != "" {
			run, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
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

		nextSequence, err := nextTaskEventSequenceWithExecutor(ctx, exec)
		if err != nil {
			return err
		}
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO task_events (
				event_seq, id, task_id, run_id, event_type, actor_kind, actor_id, origin_kind, origin_ref, payload_json, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			nextSequence,
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
	})
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
		id, task_id, run_id, event_type, actor_kind, actor_id, origin_kind, origin_ref, payload_json, timestamp
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

	events := make([]taskpkg.Event, 0)
	for rows.Next() {
		event, scanErr := scanTaskEventRecord(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task event query")
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(rows, fmt.Errorf("store: iterate task events: %w", err), "task event query")
	}
	if err := joinRowsCloseError(rows, nil, "task event query"); err != nil {
		return nil, err
	}

	return events, nil
}

// GetTaskEventRecord returns one persisted task event plus its stable row sequence.
func (g *GlobalDB) GetTaskEventRecord(ctx context.Context, eventID string) (taskpkg.EventRecord, error) {
	if err := g.checkReady(ctx, "get task event record"); err != nil {
		return taskpkg.EventRecord{}, err
	}

	trimmedEventID, err := requireTaskValue(eventID, "task event id")
	if err != nil {
		return taskpkg.EventRecord{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			event_seq, id, task_id, run_id, event_type, actor_kind, actor_id, origin_kind, origin_ref, payload_json, timestamp
		 FROM task_events
		 WHERE id = ?`,
		trimmedEventID,
	)

	record, err := scanTaskEventRecordWithSequence(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.EventRecord{}, taskpkg.ErrTaskEventNotFound
		}
		return taskpkg.EventRecord{}, err
	}
	return record, nil
}

// ListTaskEventRecords returns persisted task events ordered by stable sequence for live replay.
func (g *GlobalDB) ListTaskEventRecords(
	ctx context.Context,
	query taskpkg.EventRecordQuery,
) ([]taskpkg.EventRecord, error) {
	if err := g.checkReady(ctx, "list task event records"); err != nil {
		return nil, err
	}
	if err := query.Validate("task_event_record_query"); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT
		event_seq, id, task_id, run_id, event_type, actor_kind, actor_id, origin_kind, origin_ref, payload_json, timestamp
		FROM task_events
		WHERE task_id = ?`
	args := []any{strings.TrimSpace(query.TaskID)}
	if query.AfterSequence > 0 {
		sqlQuery += " AND event_seq > ?"
		args = append(args, query.AfterSequence)
	}
	if query.Descending {
		sqlQuery += " ORDER BY event_seq DESC"
	} else {
		sqlQuery += " ORDER BY event_seq ASC"
	}
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task event records: %w", err)
	}

	records := make([]taskpkg.EventRecord, 0)
	for rows.Next() {
		record, scanErr := scanTaskEventRecordWithSequence(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "task event record query")
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate task event records: %w", err),
			"task event record query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "task event record query"); err != nil {
		return nil, err
	}

	return records, nil
}

// ReserveQueuedRun atomically allocates one queued run attempt and optional idempotency binding.
func (g *GlobalDB) ReserveQueuedRun(
	ctx context.Context,
	taskID string,
	runID string,
	idempotencyKey string,
	origin taskpkg.Origin,
	requestedChannel string,
	metadata json.RawMessage,
	queuedAt time.Time,
) (taskpkg.Task, taskpkg.Run, bool, error) {
	if err := g.checkReady(ctx, "reserve queued task run"); err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}

	input, err := g.normalizeQueuedRunReservationInput(
		taskID,
		runID,
		idempotencyKey,
		origin,
		requestedChannel,
		metadata,
		queuedAt,
	)
	if err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}

	var reservedTask taskpkg.Task
	var reservedRun taskpkg.Run
	var existing bool
	if err := g.withTaskImmediateTransaction(ctx, "reserve queued task run", func(exec taskSQLExecutor) error {
		taskRecord, runRecord, alreadyExists, err := g.reserveQueuedRunWithExecutor(ctx, exec, input)
		if err != nil {
			return err
		}
		reservedTask = taskRecord
		reservedRun = runRecord
		existing = alreadyExists
		return nil
	}); err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}

	return reservedTask, reservedRun, existing, nil
}

func (g *GlobalDB) normalizeQueuedRunReservationInput(
	taskID string,
	runID string,
	idempotencyKey string,
	origin taskpkg.Origin,
	requestedChannel string,
	metadata json.RawMessage,
	queuedAt time.Time,
) (queuedRunReservationInput, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return queuedRunReservationInput{}, err
	}
	trimmedRunID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return queuedRunReservationInput{}, err
	}
	normalizedOrigin := taskpkg.Origin{
		Kind: origin.Kind.Normalize(),
		Ref:  strings.TrimSpace(origin.Ref),
	}
	if err := normalizedOrigin.Validate("task_run.origin"); err != nil {
		return queuedRunReservationInput{}, err
	}
	trimmedKey := strings.TrimSpace(idempotencyKey)
	normalizedMetadata := normalizeTaskJSON(metadata)
	if err := taskpkg.ValidateMetadataSize(normalizedMetadata, "enqueue_run.metadata"); err != nil {
		return queuedRunReservationInput{}, err
	}
	normalizedQueuedAt := queuedAt.UTC()
	if normalizedQueuedAt.IsZero() {
		normalizedQueuedAt = g.now()
	}
	return queuedRunReservationInput{
		taskID:           trimmedTaskID,
		runID:            trimmedRunID,
		idempotencyKey:   trimmedKey,
		origin:           normalizedOrigin,
		requestedChannel: strings.TrimSpace(requestedChannel),
		metadata:         normalizedMetadata,
		queuedAt:         normalizedQueuedAt,
	}, nil
}

func (g *GlobalDB) reserveQueuedRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	input queuedRunReservationInput,
) (taskpkg.Task, taskpkg.Run, bool, error) {
	taskRecord, err := g.getTaskWithExecutor(ctx, exec, input.taskID)
	if err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}
	if err := validateTaskForQueuedRunReservation(taskRecord); err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}

	runRecord, exists, err := g.lookupQueuedRunReservationByIdempotency(
		ctx,
		exec,
		taskRecord,
		input.idempotencyKey,
		input.origin,
	)
	if err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}
	if exists {
		return taskRecord, runRecord, true, nil
	}

	openRunID, err := g.findOpenRunIDForQueuedRunReservation(ctx, exec, taskRecord.ID)
	if err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}
	if openRunID != "" {
		return taskpkg.Task{}, taskpkg.Run{}, false, fmt.Errorf(
			"%w: task %q has open run %q; finish or cancel it before enqueueing another run",
			taskpkg.ErrInvalidStatusTransition,
			taskRecord.ID,
			openRunID,
		)
	}

	runRecord, err = g.createQueuedRunWithExecutor(ctx, exec, taskRecord, input)
	if err != nil {
		return taskpkg.Task{}, taskpkg.Run{}, false, err
	}
	return taskRecord, runRecord, false, nil
}

func (g *GlobalDB) lookupQueuedRunReservationByIdempotency(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
	idempotencyKey string,
	origin taskpkg.Origin,
) (taskpkg.Run, bool, error) {
	if idempotencyKey == "" {
		return taskpkg.Run{}, false, nil
	}

	current, err := getTaskRunIdempotencyRecord(ctx, exec, idempotencyKey, origin)
	switch {
	case errors.Is(err, taskpkg.ErrTaskRunIdempotencyNotFound):
		return taskpkg.Run{}, false, nil
	case err != nil:
		return taskpkg.Run{}, false, err
	}

	run, err := g.getTaskRunWithExecutor(ctx, exec, current.RunID)
	if err != nil {
		return taskpkg.Run{}, false, err
	}
	if strings.TrimSpace(run.TaskID) != taskRecord.ID {
		return taskpkg.Run{}, false, fmt.Errorf(
			"%w: idempotency key %q is already bound to task %q",
			taskpkg.ErrValidation,
			idempotencyKey,
			run.TaskID,
		)
	}
	return run, true, nil
}

func (g *GlobalDB) createQueuedRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
	input queuedRunReservationInput,
) (taskpkg.Run, error) {
	nextAttempt, err := nextTaskRunAttemptWithExecutor(ctx, exec, taskRecord)
	if err != nil {
		return taskpkg.Run{}, err
	}

	networkChannel := resolveStoredRunChannel(input.requestedChannel, taskRecord.NetworkChannel)
	coordinationChannelID := coordinationChannelIDForQueuedRun(taskRecord, networkChannel, input.runID)
	if err := ensureQueuedRunCoordinationChannel(
		ctx,
		exec,
		taskRecord,
		coordinationChannelID,
		input.origin,
		input.queuedAt,
	); err != nil {
		return taskpkg.Run{}, err
	}
	run := taskpkg.Run{
		ID:                    input.runID,
		TaskID:                taskRecord.ID,
		Status:                taskpkg.TaskRunStatusQueued,
		Attempt:               nextAttempt,
		Origin:                input.origin,
		IdempotencyKey:        input.idempotencyKey,
		NetworkChannel:        networkChannel,
		CoordinationChannelID: coordinationChannelID,
		Metadata:              input.metadata,
		QueuedAt:              input.queuedAt,
	}
	normalizedRun, err := g.normalizeTaskRunForCreate(run)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if err := insertQueuedTaskRun(ctx, exec, normalizedRun); err != nil {
		return taskpkg.Run{}, err
	}
	if err := g.saveQueuedRunIdempotencyWithExecutor(ctx, exec, normalizedRun); err != nil {
		return taskpkg.Run{}, err
	}
	return normalizedRun, nil
}

func coordinationChannelIDForQueuedRun(taskRecord taskpkg.Task, networkChannel string, runID string) string {
	if taskRecord.Scope.Normalize() != taskpkg.ScopeWorkspace {
		return ""
	}
	if trimmed := strings.TrimSpace(networkChannel); trimmed != "" {
		return trimmed
	}
	return derivedRunCoordinationChannelID(runID)
}

func ensureQueuedRunCoordinationChannel(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
	channelID string,
	origin taskpkg.Origin,
	queuedAt time.Time,
) error {
	trimmedChannelID := strings.TrimSpace(channelID)
	if trimmedChannelID == "" {
		return nil
	}
	trimmedWorkspaceID := strings.TrimSpace(taskRecord.WorkspaceID)
	if trimmedWorkspaceID == "" {
		return fmt.Errorf(
			"%w: workspace task %q requires workspace_id for coordination channel",
			taskpkg.ErrValidation,
			taskRecord.ID,
		)
	}

	entry, err := networkChannelEntry(ctx, exec, trimmedChannelID)
	switch {
	case err == nil:
		if strings.TrimSpace(entry.WorkspaceID) != trimmedWorkspaceID {
			return fmt.Errorf(
				"%w: coordination channel %q belongs to workspace %q, not %q",
				taskpkg.ErrValidation,
				trimmedChannelID,
				entry.WorkspaceID,
				trimmedWorkspaceID,
			)
		}
		return nil
	case errors.Is(err, sql.ErrNoRows):
	default:
		return err
	}

	timestamp := queuedAt.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	_, err = exec.ExecContext(
		ctx,
		`INSERT INTO network_channels (
			channel,
			workspace_id,
			purpose,
			created_by,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		trimmedChannelID,
		trimmedWorkspaceID,
		"task_run_coordination",
		coordinationChannelCreatedBy(origin),
		store.FormatTimestamp(timestamp),
		store.FormatTimestamp(timestamp),
	)
	if err != nil {
		return fmt.Errorf("store: create task-run coordination channel %q: %w", trimmedChannelID, err)
	}
	return nil
}

func derivedRunCoordinationChannelID(runID string) string {
	seed := strings.ToLower(strings.TrimSpace(runID))
	cleaned := make([]rune, 0, len(seed))
	lastDash := false
	for _, r := range seed {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			cleaned = append(cleaned, r)
			lastDash = false
		case r == '-':
			if len(cleaned) > 0 && !lastDash {
				cleaned = append(cleaned, r)
				lastDash = true
			}
		default:
			if len(cleaned) > 0 && !lastDash {
				cleaned = append(cleaned, '-')
				lastDash = true
			}
		}
		if len(cleaned) >= 58 {
			break
		}
	}
	value := strings.Trim(string(cleaned), "-_")
	if value == "" || !validCoordinationChannelStart(value[0]) {
		sum := sha256.Sum256([]byte(seed))
		value = fmt.Sprintf("run-%x", sum[:6])
	}
	return "coord-" + value
}

func validCoordinationChannelStart(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9')
}

func coordinationChannelCreatedBy(origin taskpkg.Origin) string {
	kind := strings.TrimSpace(string(origin.Kind.Normalize()))
	ref := strings.TrimSpace(origin.Ref)
	if kind == "" {
		return ref
	}
	if ref == "" {
		return kind
	}
	return kind + ":" + ref
}

func insertQueuedTaskRun(ctx context.Context, exec taskSQLExecutor, run taskpkg.Run) error {
	if err := insertTaskRunWithExecutor(ctx, exec, run); err != nil {
		return err
	}
	return replaceTaskRunCapabilitiesWithExecutor(ctx, exec, run)
}

func (g *GlobalDB) saveQueuedRunIdempotencyWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
) error {
	if strings.TrimSpace(run.IdempotencyKey) == "" {
		return nil
	}

	idempotency := taskpkg.RunIdempotency{
		IdempotencyKey: run.IdempotencyKey,
		RunID:          run.ID,
		Origin:         run.Origin,
		CreatedAt:      run.QueuedAt,
	}
	normalizedID, err := g.normalizeTaskRunIdempotencyForCreate(idempotency)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`INSERT INTO task_run_idempotency (
			idempotency_key, origin_kind, origin_ref, run_id, created_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(idempotency_key, origin_kind, origin_ref) DO NOTHING`,
		normalizedID.IdempotencyKey,
		string(normalizedID.Origin.Kind),
		normalizedID.Origin.Ref,
		normalizedID.RunID,
		store.FormatTimestamp(normalizedID.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("store: save task run idempotency %q: %w", normalizedID.IdempotencyKey, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"store: rows affected for task run idempotency %q: %w",
			normalizedID.IdempotencyKey,
			err,
		)
	}
	if rowsAffected > 0 {
		return nil
	}

	current, err := getTaskRunIdempotencyRecord(ctx, exec, normalizedID.IdempotencyKey, normalizedID.Origin)
	if err != nil {
		return err
	}
	if current.RunID != normalizedID.RunID {
		return fmt.Errorf(
			"%w: idempotency key %q is already bound to run %q",
			taskpkg.ErrValidation,
			normalizedID.IdempotencyKey,
			current.RunID,
		)
	}
	return nil
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
			tr.id, tr.task_id, tr.status, tr.attempt, tr.previous_run_id, tr.failure_kind,
			tr.claimed_by_kind, tr.claimed_by_ref,
			tr.session_id, tr.origin_kind, tr.origin_ref, tr.idempotency_key, tr.network_channel,
			'' AS claim_token, tr.claim_token_hash, tr.lease_until, tr.heartbeat_at,
			tr.coordination_channel_id, tr.queued_at, tr.claimed_at, tr.started_at, tr.ended_at,
			tr.error, tr.metadata_json, tr.result_json, tr.review_required,
			tr.review_request_round, tr.review_policy_snapshot, tr.review_request_id,
			tr.parent_run_id, tr.review_id, tr.review_round, tr.continuation_reason,
			tr.missing_work_json, tr.next_round_guidance
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
	return g.loadTaskRunCapabilities(ctx, g.db, run)
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
		`SELECT `+taskRunSelectColumnsSQL+`
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
	return g.loadTaskRunCapabilities(ctx, exec, run)
}

func (g *GlobalDB) getTaskWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (taskpkg.Task, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return taskpkg.Task{}, err
	}

	row := exec.QueryRowContext(
		ctx,
		`SELECT
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, current_run_id, `+taskLatestEventSeqSelectSQL+`,
			paused, paused_by, paused_at, paused_reason, metadata_json
		 FROM tasks
		 WHERE id = ?`,
		trimmedTaskID,
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

func nextTaskEventSequenceWithExecutor(ctx context.Context, exec taskSQLExecutor) (int64, error) {
	var current int64
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(event_seq), 0) FROM task_events`,
	).Scan(&current); err != nil {
		return 0, fmt.Errorf("store: query next task event sequence: %w", err)
	}
	return current + 1, nil
}

func nextTaskRunAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
) (int, error) {
	var current int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(attempt), 0) FROM task_runs WHERE task_id = ?`,
		taskRecord.ID,
	).Scan(&current); err != nil {
		return 0, fmt.Errorf("store: query next task run attempt for %q: %w", taskRecord.ID, err)
	}
	nextAttempt := current + 1
	maxAttempts := normalizeStoredTaskMaxAttempts(taskRecord.MaxAttempts)
	if nextAttempt > maxAttempts {
		return 0, fmt.Errorf(
			"%w: task %q exhausted max_attempts=%d",
			taskpkg.ErrInvalidStatusTransition,
			taskRecord.ID,
			maxAttempts,
		)
	}
	return nextAttempt, nil
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

func joinRowsCloseError(rows *sql.Rows, base error, context string) error {
	closeErr := rows.Close()
	switch {
	case base != nil && closeErr != nil:
		return errors.Join(base, fmt.Errorf("store: close %s rows: %w", context, closeErr))
	case base != nil:
		return base
	case closeErr != nil:
		return fmt.Errorf("store: close %s rows: %w", context, closeErr)
	default:
		return nil
	}
}

func validateTaskForQueuedRunReservation(taskRecord taskpkg.Task) error {
	switch taskRecord.Status.Normalize() {
	case taskpkg.TaskStatusDraft:
		return fmt.Errorf("%w: task %q is draft", taskpkg.ErrInvalidStatusTransition, taskRecord.ID)
	case taskpkg.TaskStatusCanceled:
		return fmt.Errorf("%w: task %q is canceled", taskpkg.ErrInvalidStatusTransition, taskRecord.ID)
	default:
		return nil
	}
}

func (g *GlobalDB) findOpenRunIDForQueuedRunReservation(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (string, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT id
		   FROM task_runs
		  WHERE task_id = ?
		    AND status NOT IN (?, ?, ?)
		  ORDER BY queued_at DESC, id DESC
		  LIMIT 1`,
		taskID,
		string(taskpkg.TaskRunStatusCompleted),
		string(taskpkg.TaskRunStatusFailed),
		string(taskpkg.TaskRunStatusCanceled),
	)

	var runID string
	if err := row.Scan(&runID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("store: lookup open queued run reservation for task %q: %w", taskID, err)
	}
	return runID, nil
}

func normalizeStoredTaskMaxAttempts(maxAttempts int) int {
	if maxAttempts <= 0 {
		return taskpkg.DefaultTaskMaxAttempts
	}
	return maxAttempts
}

func resolveStoredRunChannel(requested string, taskChannel string) string {
	if strings.TrimSpace(requested) != "" {
		return strings.TrimSpace(requested)
	}
	return strings.TrimSpace(taskChannel)
}

func normalizeTaskTriageLookup(
	taskID string,
	actor taskpkg.ActorIdentity,
) (string, taskpkg.ActorIdentity, error) {
	trimmedTaskID, err := requireTaskValue(taskID, "task triage task id")
	if err != nil {
		return "", taskpkg.ActorIdentity{}, err
	}

	normalizedActor, err := normalizeTaskTriageActor(actor)
	if err != nil {
		return "", taskpkg.ActorIdentity{}, err
	}

	return trimmedTaskID, normalizedActor, nil
}

func normalizeTaskTriageActor(actor taskpkg.ActorIdentity) (taskpkg.ActorIdentity, error) {
	normalizedActor := taskpkg.ActorIdentity{
		Kind: actor.Kind.Normalize(),
		Ref:  strings.TrimSpace(actor.Ref),
	}
	if err := normalizedActor.Validate("task_triage_state.actor"); err != nil {
		return taskpkg.ActorIdentity{}, err
	}
	return normalizedActor, nil
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

func scanTaskEventRecordWithSequence(scanner rowScanner) (taskpkg.EventRecord, error) {
	var (
		record      taskpkg.EventRecord
		runID       sql.NullString
		actorKind   string
		originKind  string
		payloadJSON sql.NullString
		timestamp   string
	)
	if err := scanner.Scan(
		&record.Sequence,
		&record.Event.ID,
		&record.Event.TaskID,
		&runID,
		&record.Event.EventType,
		&actorKind,
		&record.Event.Actor.Ref,
		&originKind,
		&record.Event.Origin.Ref,
		&payloadJSON,
		&timestamp,
	); err != nil {
		return taskpkg.EventRecord{}, fmt.Errorf("store: scan task event record: %w", err)
	}

	record.Event.RunID = taskNullStringValue(runID)
	record.Event.Actor.Kind = taskpkg.ActorKind(strings.TrimSpace(actorKind))
	record.Event.Origin.Kind = taskpkg.OriginKind(strings.TrimSpace(originKind))
	payload, err := decodeTaskJSON(payloadJSON, "task_event.payload_json")
	if err != nil {
		return taskpkg.EventRecord{}, err
	}
	record.Event.Payload = payload

	parsedTimestamp, err := store.ParseTimestamp(timestamp)
	if err != nil {
		return taskpkg.EventRecord{}, err
	}
	record.Event.Timestamp = parsedTimestamp
	record.Event = normalizeTaskEventRecord(record.Event)
	if err := record.Event.Validate(); err != nil {
		return taskpkg.EventRecord{}, err
	}
	if record.Sequence <= 0 {
		return taskpkg.EventRecord{}, fmt.Errorf(
			"%w: task_event_record.sequence must be positive",
			taskpkg.ErrValidation,
		)
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
