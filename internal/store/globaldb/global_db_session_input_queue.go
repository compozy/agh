package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

const sessionInputGenerationColumn = "input_generation"

const sessionInputQueueColumns = `
	id, session_id, status, mode, text, session_generation, task_run_id, run_generation,
	attempt_count, enqueued_at, dispatch_started_at, sent_at, failed_at, failure_summary,
	canceled_at, updated_at
`

// EnqueueSessionInput appends one queued operator input entry under the configured cap.
func (g *GlobalDB) EnqueueSessionInput(
	ctx context.Context,
	req store.SessionInputQueueInsert,
) (entry store.SessionInputQueueEntry, position int, err error) {
	if err := g.checkReady(ctx, "enqueue session input"); err != nil {
		return store.SessionInputQueueEntry{}, 0, err
	}
	normalized := req.Normalize()
	if normalized.Mode == "" {
		normalized.Mode = store.SessionInputQueueModeQueue
	}
	if err := normalized.Validate(); err != nil {
		return store.SessionInputQueueEntry{}, 0, err
	}

	err = g.withImmediateTransaction(ctx, "enqueue session input", func(exec globalSQLExecutor) error {
		count, countErr := countPendingSessionInputs(ctx, exec, normalized.SessionID)
		if countErr != nil {
			return countErr
		}
		if count >= normalized.QueueCap {
			return fmt.Errorf(
				"%w: session %s cap %d",
				store.ErrSessionInputQueueFull,
				normalized.SessionID,
				normalized.QueueCap,
			)
		}
		inserted, insertErr := insertSessionInputQueueEntry(ctx, exec, normalized)
		if insertErr != nil {
			return insertErr
		}
		entry = inserted
		position = count + 1
		return nil
	})
	if err != nil {
		return store.SessionInputQueueEntry{}, 0, err
	}
	return entry, position, nil
}

// StageSessionSteer replaces any active staged steer entry for the session.
func (g *GlobalDB) StageSessionSteer(
	ctx context.Context,
	req store.SessionInputQueueInsert,
) (entry store.SessionInputQueueEntry, err error) {
	if err := g.checkReady(ctx, "stage session steer"); err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	normalized := req.Normalize()
	normalized.Mode = store.SessionInputQueueModeSteer
	if err := normalized.Validate(); err != nil {
		return store.SessionInputQueueEntry{}, err
	}

	err = g.withImmediateTransaction(ctx, "stage session steer", func(exec globalSQLExecutor) error {
		nowRaw := store.FormatTimestamp(normalized.Now)
		if _, cancelErr := exec.ExecContext(ctx, `
			UPDATE session_input_queue
			SET status = ?, canceled_at = ?, updated_at = ?
			WHERE session_id = ?
			  AND mode = ?
			  AND status IN (?, ?)`,
			store.SessionInputQueueStatusCanceled,
			nowRaw,
			nowRaw,
			normalized.SessionID,
			store.SessionInputQueueModeSteer,
			store.SessionInputQueueStatusQueued,
			store.SessionInputQueueStatusDispatching,
		); cancelErr != nil {
			return fmt.Errorf("store: cancel prior session steer input: %w", cancelErr)
		}
		inserted, insertErr := insertSessionInputQueueEntry(ctx, exec, normalized)
		if insertErr != nil {
			return insertErr
		}
		entry = inserted
		return nil
	})
	if err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	return entry, nil
}

// ConsumeSessionSteer atomically marks the staged steer entry as sent and returns it once.
func (g *GlobalDB) ConsumeSessionSteer(
	ctx context.Context,
	sessionID string,
	now time.Time,
) (entry store.SessionInputQueueEntry, ok bool, err error) {
	if err := g.checkReady(ctx, "consume session steer"); err != nil {
		return store.SessionInputQueueEntry{}, false, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return store.SessionInputQueueEntry{}, false, errors.New("store: session id is required")
	}
	if now.IsZero() {
		now = g.now()
	}
	now = now.UTC()

	err = g.withImmediateTransaction(ctx, "consume session steer", func(exec globalSQLExecutor) error {
		row := exec.QueryRowContext(ctx, `
			SELECT `+sessionInputQueueColumns+`
			FROM session_input_queue
			WHERE session_id = ?
			  AND mode = ?
			  AND status = ?
			  AND session_generation = (
				SELECT input_generation FROM sessions WHERE id = ?
			  )
			ORDER BY enqueued_at DESC, id DESC
			LIMIT 1`,
			target,
			store.SessionInputQueueModeSteer,
			store.SessionInputQueueStatusQueued,
			target,
		)
		staged, scanErr := scanSessionInputQueueEntry(row)
		if errors.Is(scanErr, sql.ErrNoRows) {
			return nil
		}
		if scanErr != nil {
			return scanErr
		}
		nowRaw := store.FormatTimestamp(now)
		if _, updateErr := exec.ExecContext(ctx, `
			UPDATE session_input_queue
			SET status = ?, dispatch_started_at = ?, sent_at = ?, attempt_count = attempt_count + 1, updated_at = ?
			WHERE id = ? AND session_id = ? AND mode = ? AND status = ?`,
			store.SessionInputQueueStatusSent,
			nowRaw,
			nowRaw,
			nowRaw,
			staged.ID,
			target,
			store.SessionInputQueueModeSteer,
			store.SessionInputQueueStatusQueued,
		); updateErr != nil {
			return fmt.Errorf("store: consume session steer: %w", updateErr)
		}
		refreshed, getErr := getSessionInputQueueEntry(ctx, exec, target, staged.ID)
		if getErr != nil {
			return getErr
		}
		entry = refreshed
		ok = true
		return nil
	})
	if err != nil {
		return store.SessionInputQueueEntry{}, false, err
	}
	return entry, ok, nil
}

// ClaimNextSessionInput atomically leases the next eligible input for dispatch.
func (g *GlobalDB) ClaimNextSessionInput(
	ctx context.Context,
	sessionID string,
	now time.Time,
) (entry store.SessionInputQueueEntry, ok bool, err error) {
	if err := g.checkReady(ctx, "claim session input"); err != nil {
		return store.SessionInputQueueEntry{}, false, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return store.SessionInputQueueEntry{}, false, errors.New("store: session id is required")
	}
	if now.IsZero() {
		now = g.now()
	}
	now = now.UTC()

	err = g.withImmediateTransaction(ctx, "claim session input", func(exec globalSQLExecutor) error {
		row := exec.QueryRowContext(ctx, `
			SELECT `+sessionInputQueueColumns+`
			FROM session_input_queue
			WHERE session_id = ?
			  AND status = ?
			  AND session_generation = (
				SELECT input_generation FROM sessions WHERE id = ?
			  )
			ORDER BY enqueued_at ASC, id ASC
			LIMIT 1`,
			target,
			store.SessionInputQueueStatusQueued,
			target,
		)
		claimed, scanErr := scanSessionInputQueueEntry(row)
		if errors.Is(scanErr, sql.ErrNoRows) {
			return nil
		}
		if scanErr != nil {
			return scanErr
		}
		nowRaw := store.FormatTimestamp(now)
		if _, updateErr := exec.ExecContext(ctx, `
			UPDATE session_input_queue
			SET status = ?, dispatch_started_at = ?, attempt_count = attempt_count + 1, updated_at = ?
			WHERE id = ? AND session_id = ? AND status = ?`,
			store.SessionInputQueueStatusDispatching,
			nowRaw,
			nowRaw,
			claimed.ID,
			target,
			store.SessionInputQueueStatusQueued,
		); updateErr != nil {
			return fmt.Errorf("store: mark session input dispatching: %w", updateErr)
		}
		refreshed, getErr := getSessionInputQueueEntry(ctx, exec, target, claimed.ID)
		if getErr != nil {
			return getErr
		}
		entry = refreshed
		ok = true
		return nil
	})
	if err != nil {
		return store.SessionInputQueueEntry{}, false, err
	}
	return entry, ok, nil
}

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

// AdvanceSessionInputGeneration increments the session generation used to fence stale queue entries.
func (g *GlobalDB) AdvanceSessionInputGeneration(ctx context.Context, sessionID string, now time.Time) (int64, error) {
	if err := g.checkReady(ctx, "advance session input generation"); err != nil {
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
	var generation int64
	err := g.withImmediateTransaction(ctx, "advance session input generation", func(exec globalSQLExecutor) error {
		result, updateErr := exec.ExecContext(ctx, `
			UPDATE sessions
			SET input_generation = input_generation + 1, updated_at = ?
			WHERE id = ?`,
			nowRaw,
			target,
		)
		if updateErr != nil {
			return fmt.Errorf("store: advance session input generation: %w", updateErr)
		}
		if err := requireSessionInputRowsAffected(result, "advance session input generation", target); err != nil {
			return err
		}
		if scanErr := exec.QueryRowContext(
			ctx,
			`SELECT input_generation FROM sessions WHERE id = ?`,
			target,
		).Scan(&generation); scanErr != nil {
			return fmt.Errorf("store: read session input generation: %w", scanErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return generation, nil
}

// CurrentSessionInputGeneration returns the persisted busy-input generation for a session.
func (g *GlobalDB) CurrentSessionInputGeneration(ctx context.Context, sessionID string) (int64, error) {
	if err := g.checkReady(ctx, "read session input generation"); err != nil {
		return 0, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return 0, errors.New("store: session id is required")
	}
	var generation int64
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT input_generation FROM sessions WHERE id = ?`,
		target,
	).Scan(&generation); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%w: %s", store.ErrSessionNotFound, target)
		}
		return 0, fmt.Errorf("store: read session input generation: %w", err)
	}
	return generation, nil
}

// SessionInputQueueSummary returns the current generation and active pending counts for one session.
func (g *GlobalDB) SessionInputQueueSummary(
	ctx context.Context,
	sessionID string,
) (summary store.SessionInputQueueSummary, err error) {
	if err := g.checkReady(ctx, "read session input queue summary"); err != nil {
		return store.SessionInputQueueSummary{}, err
	}
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return store.SessionInputQueueSummary{}, errors.New("store: session id is required")
	}
	summary.SessionID = target
	err = g.db.QueryRowContext(ctx, `
		SELECT input_generation FROM sessions WHERE id = ?`,
		target,
	).Scan(&summary.Generation)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.SessionInputQueueSummary{}, fmt.Errorf("%w: %s", store.ErrSessionNotFound, target)
		}
		return store.SessionInputQueueSummary{}, fmt.Errorf("store: read session input generation: %w", err)
	}
	rows, err := g.db.QueryContext(ctx, `
		SELECT mode, status, COUNT(*)
		FROM session_input_queue
		WHERE session_id = ?
		  AND session_generation = ?
		  AND status IN (?, ?)
		GROUP BY mode, status`,
		target,
		summary.Generation,
		store.SessionInputQueueStatusQueued,
		store.SessionInputQueueStatusDispatching,
	)
	if err != nil {
		return store.SessionInputQueueSummary{}, fmt.Errorf("store: query session input queue summary: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close session input queue summary rows: %w", closeErr)
		}
	}()
	for rows.Next() {
		var mode, status string
		var count int
		if scanErr := rows.Scan(&mode, &status, &count); scanErr != nil {
			return store.SessionInputQueueSummary{}, fmt.Errorf("store: scan session input queue summary: %w", scanErr)
		}
		summary.PendingActive += count
		switch strings.TrimSpace(mode) {
		case store.SessionInputQueueModeQueue:
			summary.PendingQueued += count
		case store.SessionInputQueueModeSteer:
			summary.PendingSteer += count
		}
		if strings.TrimSpace(status) == store.SessionInputQueueStatusDispatching {
			summary.PendingLeased += count
		}
	}
	if iterErr := rows.Err(); iterErr != nil {
		return store.SessionInputQueueSummary{}, fmt.Errorf("store: iterate session input queue summary: %w", iterErr)
	}
	return summary, nil
}

func insertSessionInputQueueEntry(
	ctx context.Context,
	exec globalSQLExecutor,
	req store.SessionInputQueueInsert,
) (store.SessionInputQueueEntry, error) {
	normalized := req.Normalize()
	nowRaw := store.FormatTimestamp(normalized.Now)
	var runGeneration any
	if normalized.RunGeneration != nil {
		runGeneration = *normalized.RunGeneration
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO session_input_queue (
			id, session_id, status, mode, text, session_generation, task_run_id, run_generation,
			attempt_count, enqueued_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		normalized.ID,
		normalized.SessionID,
		store.SessionInputQueueStatusQueued,
		normalized.Mode,
		normalized.Text,
		normalized.SessionGeneration,
		normalized.TaskRunID,
		runGeneration,
		nowRaw,
		nowRaw,
	); err != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: insert session input queue entry: %w", err)
	}
	return getSessionInputQueueEntry(ctx, exec, normalized.SessionID, normalized.ID)
}

func countPendingSessionInputs(ctx context.Context, exec globalSQLExecutor, sessionID string) (int, error) {
	var count int
	if err := exec.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM session_input_queue
		WHERE session_id = ?
		  AND status IN (?, ?)`,
		sessionID,
		store.SessionInputQueueStatusQueued,
		store.SessionInputQueueStatusDispatching,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count pending session inputs: %w", err)
	}
	return count, nil
}

func getSessionInputQueueEntry(
	ctx context.Context,
	exec globalSQLExecutor,
	sessionID string,
	entryID string,
) (store.SessionInputQueueEntry, error) {
	entry, err := scanSessionInputQueueEntry(exec.QueryRowContext(ctx, `
		SELECT `+sessionInputQueueColumns+`
		FROM session_input_queue
		WHERE session_id = ? AND id = ?`,
		sessionID,
		entryID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return store.SessionInputQueueEntry{}, fmt.Errorf("%w: %s", store.ErrSessionInputQueueEntryNotFound, entryID)
	}
	return entry, err
}

func scanSessionInputQueueEntry(row interface {
	Scan(dest ...any) error
}) (store.SessionInputQueueEntry, error) {
	var entry store.SessionInputQueueEntry
	var runGeneration sql.NullInt64
	var dispatchStartedAt sql.NullString
	var sentAt sql.NullString
	var failedAt sql.NullString
	var canceledAt sql.NullString
	var enqueuedAtRaw string
	var updatedAtRaw string
	if err := row.Scan(
		&entry.ID,
		&entry.SessionID,
		&entry.Status,
		&entry.Mode,
		&entry.Text,
		&entry.SessionGeneration,
		&entry.TaskRunID,
		&runGeneration,
		&entry.AttemptCount,
		&enqueuedAtRaw,
		&dispatchStartedAt,
		&sentAt,
		&failedAt,
		&entry.FailureSummary,
		&canceledAt,
		&updatedAtRaw,
	); err != nil {
		return store.SessionInputQueueEntry{}, err
	}
	if runGeneration.Valid {
		value := runGeneration.Int64
		entry.RunGeneration = &value
	}
	parsedEnqueuedAt, err := store.ParseTimestamp(enqueuedAtRaw)
	if err != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input enqueued_at: %w", err)
	}
	entry.EnqueuedAt = parsedEnqueuedAt
	parsedUpdatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input updated_at: %w", err)
	}
	entry.UpdatedAt = parsedUpdatedAt
	var parseErr error
	entry.DispatchStartedAt, parseErr = parseOptionalSessionInputTimestamp(dispatchStartedAt)
	if parseErr != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf(
			"store: parse session input dispatch_started_at: %w",
			parseErr,
		)
	}
	entry.SentAt, parseErr = parseOptionalSessionInputTimestamp(sentAt)
	if parseErr != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input sent_at: %w", parseErr)
	}
	entry.FailedAt, parseErr = parseOptionalSessionInputTimestamp(failedAt)
	if parseErr != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input failed_at: %w", parseErr)
	}
	entry.CanceledAt, parseErr = parseOptionalSessionInputTimestamp(canceledAt)
	if parseErr != nil {
		return store.SessionInputQueueEntry{}, fmt.Errorf("store: parse session input canceled_at: %w", parseErr)
	}
	return entry, nil
}

func parseOptionalSessionInputTimestamp(value sql.NullString) (*time.Time, error) {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil, nil
	}
	parsed, err := store.ParseTimestamp(value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func requireSessionInputRowsAffected(result sql.Result, action string, id string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for %s %q: %w", action, id, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", store.ErrSessionInputQueueEntryNotFound, id)
	}
	return nil
}
