package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	taskpkg "github.com/pedronauck/agh/internal/task"
)

func updateTaskCurrentRunProjectionForRunUpdate(
	ctx context.Context,
	exec taskSQLExecutor,
	current taskpkg.Run,
	next taskpkg.Run,
) error {
	currentActive := taskRunProjectsCurrent(current.Status)
	nextActive := taskRunProjectsCurrent(next.Status)
	if currentActive && strings.TrimSpace(current.TaskID) != strings.TrimSpace(next.TaskID) {
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
	}
	if nextActive {
		return setTaskCurrentRunProjection(ctx, exec, next.TaskID, next.ID)
	}
	if currentActive {
		return clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID)
	}
	return nil
}

func setTaskCurrentRunProjectionForRun(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
) error {
	trimmedRunID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return err
	}

	var taskID string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT task_id FROM task_runs WHERE id = ?`,
		trimmedRunID,
	).Scan(&taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.ErrTaskRunNotFound
		}
		return fmt.Errorf("store: load task id for current run projection %q: %w", trimmedRunID, err)
	}
	return setTaskCurrentRunProjection(ctx, exec, taskID, trimmedRunID)
}

func setTaskCurrentRunProjection(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	runID string,
) error {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return err
	}
	trimmedRunID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return err
	}

	currentRunID, err := currentRunProjection(ctx, exec, trimmedTaskID)
	if err != nil {
		return err
	}
	if currentRunID != "" && currentRunID != trimmedRunID {
		return fmt.Errorf(
			"%w: task %q already projects active run %q",
			taskpkg.ErrInvalidStatusTransition,
			trimmedTaskID,
			currentRunID,
		)
	}

	result, err := exec.ExecContext(
		ctx,
		`UPDATE tasks SET current_run_id = ? WHERE id = ?`,
		trimmedRunID,
		trimmedTaskID,
	)
	if err != nil {
		return fmt.Errorf(
			"store: set current run projection for task %q to %q: %w",
			trimmedTaskID,
			trimmedRunID,
			err,
		)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskNotFound, trimmedTaskID, "task current run projection")
}

func clearTaskCurrentRunProjection(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
	runID string,
) error {
	trimmedTaskID, err := requireTaskValue(taskID, "task id")
	if err != nil {
		return err
	}
	trimmedRunID, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return err
	}

	currentRunID, err := currentRunProjection(ctx, exec, trimmedTaskID)
	if err != nil {
		return err
	}
	if currentRunID == "" {
		return nil
	}
	if currentRunID != trimmedRunID {
		return fmt.Errorf(
			"%w: task %q projects active run %q, not %q",
			taskpkg.ErrInvalidStatusTransition,
			trimmedTaskID,
			currentRunID,
			trimmedRunID,
		)
	}

	result, err := exec.ExecContext(
		ctx,
		`UPDATE tasks SET current_run_id = NULL WHERE id = ? AND current_run_id = ?`,
		trimmedTaskID,
		trimmedRunID,
	)
	if err != nil {
		return fmt.Errorf(
			"store: clear current run projection for task %q run %q: %w",
			trimmedTaskID,
			trimmedRunID,
			err,
		)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskNotFound, trimmedTaskID, "task current run projection")
}

func currentRunProjection(
	ctx context.Context,
	exec taskSQLExecutor,
	taskID string,
) (string, error) {
	var current sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT current_run_id FROM tasks WHERE id = ?`,
		taskID,
	).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", taskpkg.ErrTaskNotFound
		}
		return "", fmt.Errorf("store: load current run projection for task %q: %w", taskID, err)
	}
	if !current.Valid {
		return "", nil
	}
	return strings.TrimSpace(current.String), nil
}

func taskRunProjectsCurrent(status taskpkg.RunStatus) bool {
	switch status.Normalize() {
	case taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning:
		return true
	default:
		return false
	}
}
