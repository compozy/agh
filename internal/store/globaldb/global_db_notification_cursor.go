package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pedronauck/agh/internal/notifications"
	"github.com/pedronauck/agh/internal/store"
)

var _ notifications.CursorStore = (*GlobalDB)(nil)

type notificationCursorScanner interface {
	Scan(dest ...any) error
}

// GetCursor returns one durable notification cursor by key.
func (g *GlobalDB) GetCursor(ctx context.Context, key notifications.CursorKey) (notifications.Cursor, error) {
	if err := g.checkReady(ctx, "get notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := key.Normalize()
	if err != nil {
		return notifications.Cursor{}, err
	}
	cursor, found, err := loadNotificationCursor(ctx, g.db, normalized)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

// ListCursors lists durable notification cursors matching the query.
func (g *GlobalDB) ListCursors(
	ctx context.Context,
	query notifications.CursorQuery,
) (cursors []notifications.Cursor, err error) {
	if err := g.checkReady(ctx, "list notification cursors"); err != nil {
		return nil, err
	}
	normalized := query.Normalize()
	sqlQuery := `SELECT
			consumer_id, stream_name, subject_id, last_sequence, last_delivery_id,
			last_delivered_at, last_error, updated_at
		FROM notification_cursors`
	where, args := store.BuildClauses(
		store.StringClause("consumer_id", normalized.ConsumerID),
		store.StringClause("stream_name", normalized.StreamName),
		store.StringClause("subject_id", normalized.SubjectID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += ` ORDER BY stream_name ASC, subject_id ASC, consumer_id ASC`
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query notification cursors: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close notification cursor rows: %w", closeErr)
		}
	}()

	cursors = make([]notifications.Cursor, 0)
	for rows.Next() {
		cursor, scanErr := scanNotificationCursor(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		cursors = append(cursors, cursor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate notification cursors: %w", err)
	}
	return cursors, nil
}

// AdvanceCursor records a monotonic confirmed delivery position.
func (g *GlobalDB) AdvanceCursor(
	ctx context.Context,
	update notifications.AdvanceCursor,
) (cursor notifications.Cursor, err error) {
	if err := g.checkReady(ctx, "advance notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := update.Normalize(g.now())
	if err != nil {
		return notifications.Cursor{}, err
	}

	conn, err := g.db.Conn(ctx)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: open notification cursor transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close notification cursor transaction connection: %w", closeErr)
		}
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: begin notification cursor advance: %w", err)
	}

	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, "notification cursor advance"))
		}
	}()

	current, found, err := loadNotificationCursor(ctx, conn, normalized.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if found {
		if err := validateNotificationCursorAdvance(current, normalized); err != nil {
			return notifications.Cursor{}, err
		}
		if current.LastSequence == normalized.LastSequence &&
			current.LastDeliveryID == normalized.DeliveryID {
			cursor, err = refreshNotificationCursor(ctx, conn, current, normalized)
		} else {
			cursor, err = updateNotificationCursor(ctx, conn, normalized)
		}
	} else {
		cursor, err = insertNotificationCursor(ctx, conn, normalized)
	}
	if err != nil {
		return notifications.Cursor{}, err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: commit notification cursor advance: %w", err)
	}
	finished = true
	return cursor, nil
}

// ResetCursor rewinds or repairs one cursor after an explicit recovery decision.
func (g *GlobalDB) ResetCursor(
	ctx context.Context,
	reset notifications.ResetCursor,
) (cursor notifications.Cursor, err error) {
	if err := g.checkReady(ctx, "reset notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := reset.Normalize(g.now())
	if err != nil {
		return notifications.Cursor{}, err
	}

	conn, err := g.db.Conn(ctx)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: open notification cursor reset transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close notification cursor reset connection: %w", closeErr)
		}
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: begin notification cursor reset: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, "notification cursor reset"))
		}
	}()

	cursor, err = upsertNotificationCursorReset(ctx, conn, normalized)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: commit notification cursor reset: %w", err)
	}
	finished = true
	return cursor, nil
}

// RecordCursorError stores a cursor diagnostic without moving delivery progress.
func (g *GlobalDB) RecordCursorError(
	ctx context.Context,
	report notifications.CursorError,
) (notifications.Cursor, error) {
	if err := g.checkReady(ctx, "record notification cursor error"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := report.Normalize(g.now())
	if err != nil {
		return notifications.Cursor{}, err
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO notification_cursors (
			consumer_id, stream_name, subject_id, last_sequence, last_delivery_id,
			last_delivered_at, last_error, updated_at
		) VALUES (?, ?, ?, 0, '', NULL, ?, ?)
		ON CONFLICT(consumer_id, stream_name, subject_id) DO UPDATE SET
			last_error = excluded.last_error,
			updated_at = excluded.updated_at`,
		normalized.Key.ConsumerID,
		normalized.Key.StreamName,
		normalized.Key.SubjectID,
		normalized.LastError,
		store.FormatTimestamp(normalized.Now),
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: record notification cursor error %q/%q/%q: %w",
			normalized.Key.ConsumerID,
			normalized.Key.StreamName,
			normalized.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, g.db, normalized.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func loadNotificationCursor(
	ctx context.Context,
	exec taskSQLExecutor,
	key notifications.CursorKey,
) (notifications.Cursor, bool, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT
			consumer_id, stream_name, subject_id, last_sequence, last_delivery_id,
			last_delivered_at, last_error, updated_at
		FROM notification_cursors
		WHERE consumer_id = ? AND stream_name = ? AND subject_id = ?`,
		key.ConsumerID,
		key.StreamName,
		key.SubjectID,
	)
	cursor, err := scanNotificationCursor(row)
	if errors.Is(err, sql.ErrNoRows) {
		return notifications.Cursor{}, false, nil
	}
	if err != nil {
		return notifications.Cursor{}, false, err
	}
	return cursor, true, nil
}

func validateNotificationCursorAdvance(
	current notifications.Cursor,
	update notifications.AdvanceCursor,
) error {
	switch {
	case update.LastSequence > current.LastSequence:
		return nil
	case update.LastSequence == current.LastSequence && update.DeliveryID == current.LastDeliveryID:
		return nil
	default:
		return fmt.Errorf(
			"%w: current sequence %d delivery %q, update sequence %d delivery %q",
			notifications.ErrNonMonotonicCursor,
			current.LastSequence,
			current.LastDeliveryID,
			update.LastSequence,
			update.DeliveryID,
		)
	}
}

func insertNotificationCursor(
	ctx context.Context,
	exec taskSQLExecutor,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO notification_cursors (
			consumer_id, stream_name, subject_id, last_sequence, last_delivery_id,
			last_delivered_at, last_error, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, '', ?)`,
		update.Key.ConsumerID,
		update.Key.StreamName,
		update.Key.SubjectID,
		update.LastSequence,
		update.DeliveryID,
		store.FormatTimestamp(update.LastDeliveredAt),
		store.FormatTimestamp(update.Now),
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: insert notification cursor %q/%q/%q: %w",
			update.Key.ConsumerID,
			update.Key.StreamName,
			update.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, exec, update.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func updateNotificationCursor(
	ctx context.Context,
	exec taskSQLExecutor,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE notification_cursors
		SET last_sequence = ?, last_delivery_id = ?, last_delivered_at = ?, last_error = '', updated_at = ?
		WHERE consumer_id = ? AND stream_name = ? AND subject_id = ?`,
		update.LastSequence,
		update.DeliveryID,
		store.FormatTimestamp(update.LastDeliveredAt),
		store.FormatTimestamp(update.Now),
		update.Key.ConsumerID,
		update.Key.StreamName,
		update.Key.SubjectID,
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: update notification cursor %q/%q/%q: %w",
			update.Key.ConsumerID,
			update.Key.StreamName,
			update.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, exec, update.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func refreshNotificationCursor(
	ctx context.Context,
	exec taskSQLExecutor,
	current notifications.Cursor,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	lastDeliveredAt := current.LastDeliveredAt
	if lastDeliveredAt.IsZero() {
		lastDeliveredAt = update.LastDeliveredAt
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE notification_cursors
		SET last_sequence = ?, last_delivery_id = ?, last_delivered_at = ?, last_error = '', updated_at = ?
		WHERE consumer_id = ? AND stream_name = ? AND subject_id = ?`,
		update.LastSequence,
		update.DeliveryID,
		store.FormatTimestamp(lastDeliveredAt),
		store.FormatTimestamp(update.Now),
		update.Key.ConsumerID,
		update.Key.StreamName,
		update.Key.SubjectID,
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: refresh notification cursor %q/%q/%q: %w",
			update.Key.ConsumerID,
			update.Key.StreamName,
			update.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, exec, update.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func upsertNotificationCursorReset(
	ctx context.Context,
	exec taskSQLExecutor,
	reset notifications.ResetCursor,
) (notifications.Cursor, error) {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO notification_cursors (
			consumer_id, stream_name, subject_id, last_sequence, last_delivery_id,
			last_delivered_at, last_error, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, '', ?)
		ON CONFLICT(consumer_id, stream_name, subject_id) DO UPDATE SET
			last_sequence = excluded.last_sequence,
			last_delivery_id = excluded.last_delivery_id,
			last_delivered_at = excluded.last_delivered_at,
			last_error = '',
			updated_at = excluded.updated_at`,
		reset.Key.ConsumerID,
		reset.Key.StreamName,
		reset.Key.SubjectID,
		reset.LastSequence,
		reset.LastDeliveryID,
		notificationCursorTimeArg(reset.LastDeliveredAt),
		store.FormatTimestamp(reset.Now),
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: reset notification cursor %q/%q/%q: %w",
			reset.Key.ConsumerID,
			reset.Key.StreamName,
			reset.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, exec, reset.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func scanNotificationCursor(scanner notificationCursorScanner) (notifications.Cursor, error) {
	var (
		cursor             notifications.Cursor
		lastDeliveredAtRaw sql.NullString
		updatedAtRaw       string
	)
	if err := scanner.Scan(
		&cursor.Key.ConsumerID,
		&cursor.Key.StreamName,
		&cursor.Key.SubjectID,
		&cursor.LastSequence,
		&cursor.LastDeliveryID,
		&lastDeliveredAtRaw,
		&cursor.LastError,
		&updatedAtRaw,
	); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: scan notification cursor: %w", err)
	}
	if lastDeliveredAtRaw.Valid {
		parsed, err := store.ParseTimestamp(lastDeliveredAtRaw.String)
		if err != nil {
			return notifications.Cursor{}, fmt.Errorf("store: parse notification cursor delivery time: %w", err)
		}
		cursor.LastDeliveredAt = parsed
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: parse notification cursor update time: %w", err)
	}
	cursor.UpdatedAt = updatedAt
	return cursor, nil
}

func notificationCursorTimeArg(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value)
}
