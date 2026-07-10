package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

type taskStatusChangedEventPayload struct {
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
}

func setTaskStatusWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	from taskpkg.Status,
	to taskpkg.Status,
	actor taskpkg.ActorContext,
	timestamp time.Time,
) error {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return err
	}
	if err := actor.Validate(); err != nil {
		return err
	}

	fromStatus := from.Normalize()
	if err := fromStatus.Validate("task.status.from"); err != nil {
		return err
	}
	toStatus := to.Normalize()
	if err := toStatus.Validate("task.status.to"); err != nil {
		return err
	}
	if fromStatus == toStatus {
		return nil
	}

	result, err := exec.ExecContext(
		ctx,
		`UPDATE tasks SET status = ? WHERE id = ? AND status = ?`,
		string(toStatus),
		trimmedTaskID,
		string(fromStatus),
	)
	if err != nil {
		return fmt.Errorf("store: update task %q status: %w", trimmedTaskID, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: inspect task %q status update: %w", trimmedTaskID, err)
	}
	if changed == 0 {
		if err := explainTaskStatusChokepointMiss(ctx, exec, trimmedTaskID, fromStatus, toStatus); err != nil {
			return err
		}
	}

	payload, err := json.Marshal(taskStatusChangedEventPayload{
		FromStatus: string(fromStatus),
		ToStatus:   string(toStatus),
	})
	if err != nil {
		return fmt.Errorf("store: marshal task %q status_changed event: %w", trimmedTaskID, err)
	}
	if err := appendTaskEventWithExecutor(ctx, exec, EventRecordInsert{
		ID:        store.NewID("evt"),
		TaskID:    trimmedTaskID,
		EventType: string(hookspkg.HookTaskStatusChanged),
		Actor:     actor.Actor,
		Origin:    actor.Origin,
		Payload:   payload,
		Timestamp: timestamp,
	}); err != nil {
		return err
	}

	return nil
}

func setTaskStatusIfChangedWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	current taskpkg.Task,
	updated taskpkg.Task,
	actor taskpkg.ActorContext,
	changed bool,
) error {
	if !changed {
		return nil
	}
	return setTaskStatusWithExecutor(
		ctx,
		exec,
		updated.ID,
		current.Status,
		updated.Status,
		actor,
		updated.UpdatedAt,
	)
}

func taskStatusChangedForUpdate(
	current taskpkg.Status,
	updated taskpkg.Status,
	actor taskpkg.ActorContext,
) (bool, error) {
	changed := current.Normalize() != updated.Normalize()
	if !changed {
		return false, nil
	}
	if err := actor.Validate(); err != nil {
		return false, err
	}
	return true, nil
}

func explainTaskStatusChokepointMiss(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	from taskpkg.Status,
	to taskpkg.Status,
) error {
	var current string
	if err := exec.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ErrTaskNotFound
		}
		return fmt.Errorf("store: lookup task %q status: %w", taskID, err)
	}
	return fmt.Errorf(
		"%w: task %q status changed before chokepoint update from %q to %q; current %q",
		taskpkg.ErrInvalidStatusTransition,
		taskID,
		from,
		to,
		strings.TrimSpace(current),
	)
}
