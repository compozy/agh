package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

// EventRecordInsert captures the normalized task-event fields written by an
// owning transaction.
type EventRecordInsert struct {
	ID        string
	TaskID    string
	RunID     string
	EventType string
	Actor     taskpkg.ActorIdentity
	Origin    taskpkg.Origin
	Payload   json.RawMessage
	Timestamp time.Time
}

func appendTaskEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	insert EventRecordInsert,
) error {
	event := normalizeTaskEventRecord(taskpkg.Event{
		ID:        insert.ID,
		TaskID:    insert.TaskID,
		RunID:     insert.RunID,
		EventType: insert.EventType,
		Actor:     insert.Actor,
		Origin:    insert.Origin,
		Payload:   insert.Payload,
		Timestamp: insert.Timestamp,
	})
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if err := event.Validate(); err != nil {
		return err
	}

	if err := ensureTaskEventTaskExists(ctx, exec, event.TaskID); err != nil {
		return err
	}
	if strings.TrimSpace(event.RunID) != "" {
		runTaskID, err := taskEventRunTaskID(ctx, exec, event.RunID)
		if err != nil {
			return err
		}
		if runTaskID != event.TaskID {
			return fmt.Errorf(
				"%w: task_event.run_id %q does not belong to task %q",
				taskpkg.ErrValidation,
				event.RunID,
				event.TaskID,
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
		event.ID,
		event.TaskID,
		store.NullableString(event.RunID),
		event.EventType,
		string(event.Actor.Kind),
		event.Actor.Ref,
		string(event.Origin.Kind),
		event.Origin.Ref,
		nullableTaskJSON(event.Payload),
		store.FormatTimestamp(event.Timestamp),
	); err != nil {
		return fmt.Errorf("store: create task event %q: %w", event.ID, err)
	}

	return nil
}

func appendTaskEventPayloadWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	runID string,
	eventType string,
	actor taskpkg.ActorContext,
	timestamp time.Time,
	payload any,
) error {
	if err := actor.Validate(); err != nil {
		return err
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("store: marshal task event %q payload: %w", eventType, err)
	}
	return appendTaskEventWithExecutor(ctx, exec, EventRecordInsert{
		ID:        store.NewID("evt"),
		TaskID:    taskID,
		RunID:     runID,
		EventType: eventType,
		Actor:     actor.Actor,
		Origin:    actor.Origin,
		Payload:   rawPayload,
		Timestamp: timestamp,
	})
}

func ensureTaskEventTaskExists(ctx context.Context, exec taskSQLExecutor, taskID string) error {
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

func taskEventRunTaskID(ctx context.Context, exec taskSQLExecutor, runID string) (string, error) {
	trimmedID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return "", err
	}

	var taskID string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT task_id FROM task_runs WHERE id = ?`,
		trimmedID,
	).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", taskpkg.ErrTaskRunNotFound
		}
		return "", fmt.Errorf("store: lookup task run %q: %w", trimmedID, err)
	}
	return strings.TrimSpace(taskID), nil
}
