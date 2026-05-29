package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

// LoadRunStarvation reads a run's durable escalation budget. The bool reports row presence;
// absence is not an error — the convergence backstop treats it as a fresh budget.
func (g *GlobalDB) LoadRunStarvation(
	ctx context.Context,
	runID string,
) (taskpkg.RunStarvation, bool, error) {
	if err := g.checkReady(ctx, "load run starvation"); err != nil {
		return taskpkg.RunStarvation{}, false, err
	}
	id, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return taskpkg.RunStarvation{}, false, err
	}
	record, err := scanRunStarvation(g.db.QueryRowContext(
		ctx,
		`SELECT run_id, wake_count, first_starved_at, last_wake_at, escalation_tier,
			spawn_requested_at, starved_event_at, updated_at
			FROM task_run_starvation WHERE run_id = ?`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return taskpkg.RunStarvation{}, false, nil
	}
	if err != nil {
		return taskpkg.RunStarvation{}, false, err
	}
	return record, true, nil
}

func scanRunStarvation(scanner interface{ Scan(...any) error }) (taskpkg.RunStarvation, error) {
	var (
		record         taskpkg.RunStarvation
		firstStarvedAt string
		updatedAt      string
		lastWakeAt     sql.NullString
		spawnRequested sql.NullString
		starvedEvent   sql.NullString
	)
	if err := scanner.Scan(
		&record.RunID,
		&record.WakeCount,
		&firstStarvedAt,
		&lastWakeAt,
		&record.EscalationTier,
		&spawnRequested,
		&starvedEvent,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return taskpkg.RunStarvation{}, err
		}
		return taskpkg.RunStarvation{}, fmt.Errorf("store: scan run starvation: %w", err)
	}
	parsed, err := store.ParseTimestamp(firstStarvedAt)
	if err != nil {
		return taskpkg.RunStarvation{}, fmt.Errorf("store: parse run starvation first_starved_at: %w", err)
	}
	record.FirstStarvedAt = parsed
	if record.UpdatedAt, err = store.ParseTimestamp(updatedAt); err != nil {
		return taskpkg.RunStarvation{}, fmt.Errorf("store: parse run starvation updated_at: %w", err)
	}
	last, err := parseNullableStarvationTime(lastWakeAt)
	if err != nil {
		return taskpkg.RunStarvation{}, err
	}
	if last != nil {
		record.LastWakeAt = *last
	}
	if record.SpawnRequestedAt, err = parseNullableStarvationTime(spawnRequested); err != nil {
		return taskpkg.RunStarvation{}, err
	}
	if record.StarvedEventAt, err = parseNullableStarvationTime(starvedEvent); err != nil {
		return taskpkg.RunStarvation{}, err
	}
	return record, nil
}

// ListRunStarvation returns every escalation budget row so the convergence backstop can reconcile
// rows whose run has left the queued set.
func (g *GlobalDB) ListRunStarvation(ctx context.Context) (rows []taskpkg.RunStarvation, err error) {
	if err := g.checkReady(ctx, "list run starvation"); err != nil {
		return nil, err
	}
	cursor, err := g.db.QueryContext(
		ctx,
		`SELECT run_id, wake_count, first_starved_at, last_wake_at, escalation_tier,
			spawn_requested_at, starved_event_at, updated_at
			FROM task_run_starvation`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list run starvation: %w", err)
	}
	defer func() {
		if closeErr := cursor.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close run starvation rows: %w", closeErr)
		}
	}()
	for cursor.Next() {
		row, scanErr := scanRunStarvation(cursor)
		if scanErr != nil {
			return nil, scanErr
		}
		rows = append(rows, row)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate run starvation: %w", err)
	}
	return rows, nil
}

// UpsertRunStarvation writes the run's escalation budget, advancing it in place on conflict.
func (g *GlobalDB) UpsertRunStarvation(
	ctx context.Context,
	mutation taskpkg.RunStarvationMutation,
) (taskpkg.RunStarvation, error) {
	if err := g.checkReady(ctx, "upsert run starvation"); err != nil {
		return taskpkg.RunStarvation{}, err
	}
	id, err := requireTaskValue(mutation.RunID, "task run id")
	if err != nil {
		return taskpkg.RunStarvation{}, err
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO task_run_starvation (
			run_id, wake_count, first_starved_at, last_wake_at, escalation_tier,
			spawn_requested_at, starved_event_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			wake_count = excluded.wake_count,
			first_starved_at = excluded.first_starved_at,
			last_wake_at = excluded.last_wake_at,
			escalation_tier = excluded.escalation_tier,
			spawn_requested_at = excluded.spawn_requested_at,
			starved_event_at = excluded.starved_event_at,
			updated_at = excluded.updated_at`,
		id,
		mutation.WakeCount,
		store.FormatTimestamp(mutation.FirstStarvedAt),
		nullableStarvationTime(mutation.LastWakeAt),
		mutation.EscalationTier,
		nullableStarvationTimePtr(mutation.SpawnRequestedAt),
		nullableStarvationTimePtr(mutation.StarvedEventAt),
		store.FormatTimestamp(mutation.UpdatedAt),
	); err != nil {
		return taskpkg.RunStarvation{}, fmt.Errorf("store: upsert run starvation: %w", err)
	}
	return taskpkg.RunStarvation{
		RunID:            id,
		WakeCount:        mutation.WakeCount,
		FirstStarvedAt:   mutation.FirstStarvedAt,
		LastWakeAt:       mutation.LastWakeAt,
		EscalationTier:   mutation.EscalationTier,
		SpawnRequestedAt: cloneTimePointer(mutation.SpawnRequestedAt),
		StarvedEventAt:   cloneTimePointer(mutation.StarvedEventAt),
		UpdatedAt:        mutation.UpdatedAt,
	}, nil
}

// ClearRunStarvation removes a run's escalation budget once it leaves the starved set.
func (g *GlobalDB) ClearRunStarvation(ctx context.Context, runID string) error {
	if err := g.checkReady(ctx, "clear run starvation"); err != nil {
		return err
	}
	id, err := requireTaskValue(runID, "task run id")
	if err != nil {
		return err
	}
	if _, err := g.db.ExecContext(ctx, `DELETE FROM task_run_starvation WHERE run_id = ?`, id); err != nil {
		return fmt.Errorf("store: clear run starvation: %w", err)
	}
	return nil
}

func parseNullableStarvationTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	parsed, err := store.ParseTimestamp(value.String)
	if err != nil {
		return nil, fmt.Errorf("store: parse run starvation timestamp: %w", err)
	}
	return &parsed, nil
}

func nullableStarvationTime(value time.Time) sql.NullString {
	if value.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: store.FormatTimestamp(value), Valid: true}
}

func nullableStarvationTimePtr(value *time.Time) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return nullableStarvationTime(*value)
}
