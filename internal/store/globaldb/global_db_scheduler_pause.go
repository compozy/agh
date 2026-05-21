package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

var _ interface {
	GetSchedulerPauseState(context.Context) (taskpkg.InspectSchedulerState, error)
} = (*GlobalDB)(nil)

// GetSchedulerPauseState returns the singleton scheduler pause state.
func (g *GlobalDB) GetSchedulerPauseState(ctx context.Context) (taskpkg.InspectSchedulerState, error) {
	if err := g.checkReady(ctx, "get scheduler pause state"); err != nil {
		return taskpkg.InspectSchedulerState{}, err
	}

	var (
		paused       int
		pausedBy     string
		pausedAtRaw  sql.NullString
		reason       string
		updatedAtRaw string
	)
	err := g.db.QueryRowContext(
		ctx,
		`SELECT paused, paused_by, paused_at, reason, updated_at FROM scheduler_pause WHERE id = 1`,
	).Scan(&paused, &pausedBy, &pausedAtRaw, &reason, &updatedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return taskpkg.InspectSchedulerState{}, nil
	}
	if err != nil {
		return taskpkg.InspectSchedulerState{}, fmt.Errorf("store: get scheduler pause state: %w", err)
	}

	state := taskpkg.InspectSchedulerState{
		Paused:   paused != 0,
		PausedBy: pausedBy,
		Reason:   reason,
	}
	if pausedAtRaw.Valid {
		pausedAt, parseErr := store.ParseNullableTimestamp(pausedAtRaw.String)
		if parseErr != nil {
			return taskpkg.InspectSchedulerState{}, parseErr
		}
		if pausedAt != nil {
			state.PausedAt = *pausedAt
		}
	}
	updatedAt, parseErr := store.ParseTimestamp(updatedAtRaw)
	if parseErr != nil {
		return taskpkg.InspectSchedulerState{}, parseErr
	}
	state.UpdatedAt = updatedAt
	return state, nil
}
