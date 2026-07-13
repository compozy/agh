package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/compozy/agh/internal/notifications"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
)

// GetCursor returns one durable notification cursor by key.
func (n *NotificationRepo) GetCursor(ctx context.Context, key notifications.CursorKey) (notifications.Cursor, error) {
	if err := n.checkReady(ctx, "get notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := key.Normalize()
	if err != nil {
		return notifications.Cursor{}, err
	}
	cursor, found, err := loadNotificationCursor(ctx, n.queries, normalized)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

// ListCursors lists durable notification cursors matching the query.
func (n *NotificationRepo) ListCursors(
	ctx context.Context,
	query notifications.CursorQuery,
) (cursors []notifications.Cursor, err error) {
	if err := n.checkReady(ctx, "list notification cursors"); err != nil {
		return nil, err
	}
	normalized := query.Normalize()
	// dynamic-sql: optional cursor filters and the limit change the query structure.
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

	rows, err := n.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query notification cursors: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			joinCleanupError(&err, fmt.Errorf("store: close notification cursor rows: %w", closeErr))
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
func (n *NotificationRepo) AdvanceCursor(
	ctx context.Context,
	update notifications.AdvanceCursor,
) (cursor notifications.Cursor, err error) {
	if err := n.checkReady(ctx, "advance notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := update.Normalize(n.now())
	if err != nil {
		return notifications.Cursor{}, err
	}

	conn, err := n.db.Conn(ctx)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: open notification cursor transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			joinCleanupError(&err, fmt.Errorf("store: close notification cursor transaction connection: %w", closeErr))
		}
	}()

	// dynamic-sql: BEGIN IMMEDIATE is explicit SQLite transaction control on the pinned notification connection.
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: begin notification cursor advance: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			rollbackNotificationImmediate(ctx, &err, conn, "notification cursor advance")
		}
	}()

	queries := sqlcgen.New(conn)
	current, found, err := loadNotificationCursor(ctx, queries, normalized.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if found {
		if err := validateNotificationCursorAdvance(current, normalized); err != nil {
			return notifications.Cursor{}, err
		}
		if current.LastSequence == normalized.LastSequence && current.LastDeliveryID == normalized.DeliveryID {
			cursor, err = refreshNotificationCursor(ctx, queries, current, normalized)
		} else {
			cursor, err = updateNotificationCursor(ctx, queries, normalized)
		}
	} else {
		cursor, err = insertNotificationCursor(ctx, queries, normalized)
	}
	if err != nil {
		return notifications.Cursor{}, err
	}
	// dynamic-sql: COMMIT closes the explicit SQLite transaction and is outside sqlc's query model.
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: commit notification cursor advance: %w", err)
	}
	finished = true
	return cursor, nil
}

// ResetCursor rewinds or repairs one cursor after an explicit recovery decision.
func (n *NotificationRepo) ResetCursor(
	ctx context.Context,
	reset notifications.ResetCursor,
) (cursor notifications.Cursor, err error) {
	if err := n.checkReady(ctx, "reset notification cursor"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := reset.Normalize(n.now())
	if err != nil {
		return notifications.Cursor{}, err
	}
	conn, err := n.db.Conn(ctx)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: open notification cursor reset transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			joinCleanupError(&err, fmt.Errorf("store: close notification cursor reset connection: %w", closeErr))
		}
	}()
	// dynamic-sql: BEGIN IMMEDIATE is explicit SQLite transaction control on the pinned notification connection.
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: begin notification cursor reset: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			rollbackNotificationImmediate(ctx, &err, conn, "notification cursor reset")
		}
	}()
	queries := sqlcgen.New(conn)
	cursor, err = upsertNotificationCursorReset(ctx, queries, normalized)
	if err != nil {
		return notifications.Cursor{}, err
	}
	// dynamic-sql: COMMIT closes the explicit SQLite transaction and is outside sqlc's query model.
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: commit notification cursor reset: %w", err)
	}
	finished = true
	return cursor, nil
}

// RecordCursorError stores a cursor diagnostic without moving delivery progress.
func (n *NotificationRepo) RecordCursorError(
	ctx context.Context,
	report notifications.CursorError,
) (cursor notifications.Cursor, err error) {
	if err := n.checkReady(ctx, "record notification cursor error"); err != nil {
		return notifications.Cursor{}, err
	}
	normalized, err := report.Normalize(n.now())
	if err != nil {
		return notifications.Cursor{}, err
	}
	conn, err := n.db.Conn(ctx)
	if err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: open notification cursor error transaction: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			joinCleanupError(
				&err,
				fmt.Errorf("store: close notification cursor error transaction connection: %w", closeErr),
			)
		}
	}()
	// dynamic-sql: BEGIN IMMEDIATE is explicit SQLite transaction control on the pinned notification connection.
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: begin notification cursor error record: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			rollbackNotificationImmediate(ctx, &err, conn, "notification cursor error record")
		}
	}()
	queries := sqlcgen.New(conn)
	if err := queries.RecordNotificationCursorError(ctx, sqlcgen.RecordNotificationCursorErrorParams{
		ConsumerID: normalized.Key.ConsumerID,
		StreamName: normalized.Key.StreamName,
		SubjectID:  normalized.Key.SubjectID,
		LastError:  normalized.LastError,
		UpdatedAt:  store.FormatTimestamp(normalized.Now),
	}); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: record notification cursor error %q/%q/%q: %w",
			normalized.Key.ConsumerID,
			normalized.Key.StreamName,
			normalized.Key.SubjectID,
			err,
		)
	}
	cursor, found, err := loadNotificationCursor(ctx, queries, normalized.Key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	// dynamic-sql: COMMIT closes the explicit SQLite transaction and is outside sqlc's query model.
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return notifications.Cursor{}, fmt.Errorf("store: commit notification cursor error record: %w", err)
	}
	finished = true
	return cursor, nil
}

func loadNotificationCursor(
	ctx context.Context,
	queries *sqlcgen.Queries,
	key notifications.CursorKey,
) (notifications.Cursor, bool, error) {
	row, err := queries.GetNotificationCursor(ctx, sqlcgen.GetNotificationCursorParams{
		ConsumerID: key.ConsumerID,
		StreamName: key.StreamName,
		SubjectID:  key.SubjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notifications.Cursor{}, false, nil
		}
		return notifications.Cursor{}, false, fmt.Errorf("store: get notification cursor: %w", err)
	}
	cursor, err := notificationCursorFromGenerated(row)
	if err != nil {
		return notifications.Cursor{}, false, err
	}
	return cursor, true, nil
}

func validateNotificationCursorAdvance(current notifications.Cursor, update notifications.AdvanceCursor) error {
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
	queries *sqlcgen.Queries,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	if err := queries.InsertNotificationCursor(ctx, sqlcgen.InsertNotificationCursorParams{
		ConsumerID:      update.Key.ConsumerID,
		StreamName:      update.Key.StreamName,
		SubjectID:       update.Key.SubjectID,
		LastSequence:    update.LastSequence,
		LastDeliveryID:  update.DeliveryID,
		LastDeliveredAt: notificationCursorTimeArg(update.LastDeliveredAt),
		UpdatedAt:       store.FormatTimestamp(update.Now),
	}); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: insert notification cursor %q/%q/%q: %w",
			update.Key.ConsumerID,
			update.Key.StreamName,
			update.Key.SubjectID,
			err,
		)
	}
	return requireNotificationCursor(ctx, queries, update.Key)
}

func updateNotificationCursor(
	ctx context.Context,
	queries *sqlcgen.Queries,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	return writeNotificationCursor(ctx, queries, update, update.LastDeliveredAt, "update")
}

func refreshNotificationCursor(
	ctx context.Context,
	queries *sqlcgen.Queries,
	current notifications.Cursor,
	update notifications.AdvanceCursor,
) (notifications.Cursor, error) {
	lastDeliveredAt := current.LastDeliveredAt
	if lastDeliveredAt.IsZero() {
		lastDeliveredAt = update.LastDeliveredAt
	}
	return writeNotificationCursor(ctx, queries, update, lastDeliveredAt, "refresh")
}

func writeNotificationCursor(
	ctx context.Context,
	queries *sqlcgen.Queries,
	update notifications.AdvanceCursor,
	lastDeliveredAt time.Time,
	action string,
) (notifications.Cursor, error) {
	if err := queries.UpdateNotificationCursor(ctx, sqlcgen.UpdateNotificationCursorParams{
		LastSequence:    update.LastSequence,
		LastDeliveryID:  update.DeliveryID,
		LastDeliveredAt: notificationCursorTimeArg(lastDeliveredAt),
		UpdatedAt:       store.FormatTimestamp(update.Now),
		ConsumerID:      update.Key.ConsumerID,
		StreamName:      update.Key.StreamName,
		SubjectID:       update.Key.SubjectID,
	}); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: %s notification cursor %q/%q/%q: %w",
			action,
			update.Key.ConsumerID,
			update.Key.StreamName,
			update.Key.SubjectID,
			err,
		)
	}
	return requireNotificationCursor(ctx, queries, update.Key)
}

func upsertNotificationCursorReset(
	ctx context.Context,
	queries *sqlcgen.Queries,
	reset notifications.ResetCursor,
) (notifications.Cursor, error) {
	if err := queries.ResetNotificationCursor(ctx, sqlcgen.ResetNotificationCursorParams{
		ConsumerID:      reset.Key.ConsumerID,
		StreamName:      reset.Key.StreamName,
		SubjectID:       reset.Key.SubjectID,
		LastSequence:    reset.LastSequence,
		LastDeliveryID:  reset.LastDeliveryID,
		LastDeliveredAt: notificationCursorTimeArg(reset.LastDeliveredAt),
		UpdatedAt:       store.FormatTimestamp(reset.Now),
	}); err != nil {
		return notifications.Cursor{}, fmt.Errorf(
			"store: reset notification cursor %q/%q/%q: %w",
			reset.Key.ConsumerID,
			reset.Key.StreamName,
			reset.Key.SubjectID,
			err,
		)
	}
	return requireNotificationCursor(ctx, queries, reset.Key)
}

func requireNotificationCursor(
	ctx context.Context,
	queries *sqlcgen.Queries,
	key notifications.CursorKey,
) (notifications.Cursor, error) {
	cursor, found, err := loadNotificationCursor(ctx, queries, key)
	if err != nil {
		return notifications.Cursor{}, err
	}
	if !found {
		return notifications.Cursor{}, notifications.ErrCursorNotFound
	}
	return cursor, nil
}

func notificationCursorTimeArg(value time.Time) sql.NullString {
	if value.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: store.FormatTimestamp(value), Valid: true}
}
