package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

// GetSchedulerPauseState returns the singleton scheduler pause state for inspect diagnostics.
func (g *GlobalDB) GetSchedulerPauseState(ctx context.Context) (taskpkg.InspectSchedulerState, error) {
	state, err := g.GetSchedulerPause(ctx)
	if err != nil {
		return taskpkg.InspectSchedulerState{}, err
	}
	return taskpkg.InspectSchedulerState(state), nil
}

// GetSchedulerPause returns the singleton scheduler pause state.
func (g *GlobalDB) GetSchedulerPause(ctx context.Context) (taskpkg.SchedulerPauseState, error) {
	if err := g.checkReady(ctx, "get scheduler pause state"); err != nil {
		return taskpkg.SchedulerPauseState{}, err
	}
	state, err := g.scanSchedulerPauseState(
		ctx,
		`SELECT paused, paused_by, paused_at, reason, updated_at FROM scheduler_pause WHERE id = 1`,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return taskpkg.SchedulerPauseState{}, nil
	}
	return state, err
}

// SetSchedulerPaused marks the scheduler-wide pause singleton as paused.
func (g *GlobalDB) SetSchedulerPaused(
	ctx context.Context,
	actor string,
	reason string,
) (taskpkg.SchedulerPauseState, error) {
	if err := g.checkReady(ctx, "pause scheduler"); err != nil {
		return taskpkg.SchedulerPauseState{}, err
	}
	now := g.now().UTC()
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO scheduler_pause (id, paused, paused_by, paused_at, reason, updated_at)
		 VALUES (1, 1, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   paused = 1,
		   paused_by = excluded.paused_by,
		   paused_at = excluded.paused_at,
		   reason = excluded.reason,
		   updated_at = excluded.updated_at`,
		strings.TrimSpace(actor),
		store.FormatTimestamp(now),
		strings.TrimSpace(reason),
		store.FormatTimestamp(now),
	); err != nil {
		return taskpkg.SchedulerPauseState{}, fmt.Errorf("store: pause scheduler: %w", err)
	}
	return g.GetSchedulerPause(ctx)
}

// SetSchedulerResumed clears the scheduler-wide pause singleton.
func (g *GlobalDB) SetSchedulerResumed(ctx context.Context) (taskpkg.SchedulerPauseState, error) {
	if err := g.checkReady(ctx, "resume scheduler"); err != nil {
		return taskpkg.SchedulerPauseState{}, err
	}
	now := g.now().UTC()
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO scheduler_pause (id, paused, paused_by, paused_at, reason, updated_at)
		 VALUES (1, 0, '', NULL, '', ?)
		 ON CONFLICT(id) DO UPDATE SET
		   paused = 0,
		   paused_by = '',
		   paused_at = NULL,
		   reason = '',
		   updated_at = excluded.updated_at`,
		store.FormatTimestamp(now),
	); err != nil {
		return taskpkg.SchedulerPauseState{}, fmt.Errorf("store: resume scheduler: %w", err)
	}
	return g.GetSchedulerPause(ctx)
}

func (g *GlobalDB) scanSchedulerPauseState(
	ctx context.Context,
	query string,
	args ...any,
) (taskpkg.SchedulerPauseState, error) {
	var (
		paused       int
		pausedBy     string
		pausedAtRaw  sql.NullString
		reason       string
		updatedAtRaw string
	)
	if err := g.db.QueryRowContext(ctx, query, args...).Scan(
		&paused,
		&pausedBy,
		&pausedAtRaw,
		&reason,
		&updatedAtRaw,
	); err != nil {
		return taskpkg.SchedulerPauseState{}, fmt.Errorf("store: get scheduler pause state: %w", err)
	}
	state := taskpkg.SchedulerPauseState{
		Paused:   paused != 0,
		PausedBy: strings.TrimSpace(pausedBy),
		Reason:   strings.TrimSpace(reason),
	}
	if pausedAtRaw.Valid {
		pausedAt, parseErr := store.ParseNullableTimestamp(pausedAtRaw.String)
		if parseErr != nil {
			return taskpkg.SchedulerPauseState{}, parseErr
		}
		if pausedAt != nil {
			state.PausedAt = *pausedAt
		}
	}
	updatedAt, parseErr := store.ParseTimestamp(updatedAtRaw)
	if parseErr != nil {
		return taskpkg.SchedulerPauseState{}, parseErr
	}
	state.UpdatedAt = updatedAt
	return state, nil
}
