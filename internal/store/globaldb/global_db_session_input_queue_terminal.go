package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

// MarkSessionInputSent records successful dispatch for one queue entry.
func (g *GlobalDB) MarkSessionInputSent(
	ctx context.Context,
	sessionID string,
	entryID string,
	now time.Time,
) error {
	return g.updateSessionInputTerminal(
		ctx,
		"mark session input sent",
		sessionID,
		entryID,
		store.SessionInputQueueStatusSent,
		"",
		now,
	)
}

// ReleaseSessionInput returns a leased entry to the queued state after a dispatch race.
func (g *GlobalDB) ReleaseSessionInput(ctx context.Context, sessionID string, entryID string, now time.Time) error {
	if err := g.checkReady(ctx, "release session input"); err != nil {
		return err
	}
	target := strings.TrimSpace(sessionID)
	entryID = strings.TrimSpace(entryID)
	if target == "" || entryID == "" {
		return errors.New("store: session id and queue entry id are required")
	}
	if now.IsZero() {
		now = g.now()
	}
	nowRaw := store.FormatTimestamp(now.UTC())
	result, err := g.db.ExecContext(ctx, `
		UPDATE session_input_queue
		SET status = ?, dispatch_started_at = NULL, updated_at = ?
		WHERE id = ? AND session_id = ? AND status = ?`,
		store.SessionInputQueueStatusQueued,
		nowRaw,
		entryID,
		target,
		store.SessionInputQueueStatusDispatching,
	)
	if err != nil {
		return fmt.Errorf("store: release session input: %w", err)
	}
	return requireSessionInputRowsAffected(result, "release session input", entryID)
}

// MarkSessionInputFailed records a dispatch failure for one queue entry.
func (g *GlobalDB) MarkSessionInputFailed(
	ctx context.Context,
	sessionID string,
	entryID string,
	summary string,
	now time.Time,
) error {
	return g.updateSessionInputTerminal(
		ctx,
		"mark session input failed",
		sessionID,
		entryID,
		store.SessionInputQueueStatusFailed,
		summary,
		now,
	)
}

func (g *GlobalDB) updateSessionInputTerminal(
	ctx context.Context,
	action string,
	sessionID string,
	entryID string,
	status string,
	summary string,
	now time.Time,
) error {
	if err := g.checkReady(ctx, action); err != nil {
		return err
	}
	target := strings.TrimSpace(sessionID)
	entryID = strings.TrimSpace(entryID)
	if target == "" || entryID == "" {
		return errors.New("store: session id and queue entry id are required")
	}
	if now.IsZero() {
		now = g.now()
	}
	nowRaw := store.FormatTimestamp(now.UTC())
	column := "sent_at"
	if status == store.SessionInputQueueStatusFailed {
		column = "failed_at"
	}
	query := fmt.Sprintf(`
		UPDATE session_input_queue
		SET status = ?, %s = ?, failure_summary = ?, updated_at = ?
		WHERE id = ? AND session_id = ? AND status = ?`, column)
	result, err := g.db.ExecContext(
		ctx,
		query,
		status,
		nowRaw,
		strings.TrimSpace(summary),
		nowRaw,
		entryID,
		target,
		store.SessionInputQueueStatusDispatching,
	)
	if err != nil {
		return fmt.Errorf("store: %s: %w", action, err)
	}
	return requireSessionInputRowsAffected(result, action, entryID)
}

// CancelSessionInput cancels one pending queue entry.
func (g *GlobalDB) CancelSessionInput(
	ctx context.Context,
	sessionID string,
	entryID string,
	now time.Time,
) (store.SessionInputQueueEntry, error) {
	if err := g.checkReady(ctx, "cancel session input"); err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	target := strings.TrimSpace(sessionID)
	entryID = strings.TrimSpace(entryID)
	if target == "" || entryID == "" {
		return store.SessionInputQueueEntry{}, errors.New("store: session id and queue entry id are required")
	}
	if now.IsZero() {
		now = g.now()
	}
	var entry store.SessionInputQueueEntry
	err := g.withImmediateTransaction(ctx, "cancel session input", func(exec globalSQLExecutor) error {
		existing, getErr := getSessionInputQueueEntry(ctx, exec, target, entryID)
		if getErr != nil {
			return getErr
		}
		if existing.Status == store.SessionInputQueueStatusSent ||
			existing.Status == store.SessionInputQueueStatusFailed ||
			existing.Status == store.SessionInputQueueStatusCanceled {
			entry = existing
			return nil
		}
		nowRaw := store.FormatTimestamp(now.UTC())
		if _, updateErr := exec.ExecContext(ctx, `
			UPDATE session_input_queue
			SET status = ?, canceled_at = ?, updated_at = ?
			WHERE id = ? AND session_id = ?`,
			store.SessionInputQueueStatusCanceled,
			nowRaw,
			nowRaw,
			entryID,
			target,
		); updateErr != nil {
			return fmt.Errorf("store: cancel session input: %w", updateErr)
		}
		updated, getUpdatedErr := getSessionInputQueueEntry(ctx, exec, target, entryID)
		if getUpdatedErr != nil {
			return getUpdatedErr
		}
		entry = updated
		return nil
	})
	if err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	return entry, nil
}

// CancelPendingSessionInputs cancels stale entries older than the supplied generation.
func (g *GlobalDB) CancelPendingSessionInputs(
	ctx context.Context,
	sessionID string,
	generation int64,
	now time.Time,
) (int, error) {
	if err := g.checkReady(ctx, "cancel pending session inputs"); err != nil {
		return 0, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return 0, errors.New("store: session id is required")
	}
	if now.IsZero() {
		now = g.now()
	}
	nowRaw := store.FormatTimestamp(now.UTC())
	result, err := g.db.ExecContext(ctx, `
		UPDATE session_input_queue
		SET status = ?, canceled_at = ?, updated_at = ?
		WHERE session_id = ?
		  AND session_generation < ?
		  AND status IN (?, ?)`,
		store.SessionInputQueueStatusCanceled,
		nowRaw,
		nowRaw,
		target,
		generation,
		store.SessionInputQueueStatusQueued,
		store.SessionInputQueueStatusDispatching,
	)
	if err != nil {
		return 0, fmt.Errorf("store: cancel pending session inputs: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: count canceled session inputs: %w", err)
	}
	return int(rows), nil
}
