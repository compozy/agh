package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

// WriteNetworkMessage stores one persisted network timeline message, ignoring duplicate message ids.
func (g *GlobalDB) WriteNetworkMessage(ctx context.Context, entry store.NetworkMessageEntry) error {
	if err := g.checkReady(ctx, "write network message"); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return fmt.Errorf("store: validate network message entry: %w", err)
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = g.now()
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_message_log (
			message_id, session_id, channel, peer_from, kind, intent, text, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO NOTHING`,
		entry.MessageID,
		store.NullableString(entry.SessionID),
		entry.Channel,
		entry.PeerFrom,
		entry.Kind,
		store.NullableString(entry.Intent),
		entry.Text,
		store.FormatTimestamp(entry.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert network message entry: %w", err)
	}

	return nil
}

// ListNetworkMessages returns persisted network timeline rows filtered by the supplied options.
func (g *GlobalDB) ListNetworkMessages(
	ctx context.Context,
	query store.NetworkMessageQuery,
) (entries []store.NetworkMessageEntry, err error) {
	if err := g.checkReady(ctx, "list network messages"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network message query: %w", err)
	}

	sqlQuery := `SELECT message_id, session_id, channel, peer_from, kind, intent, text, timestamp FROM network_message_log`
	where, args := store.BuildClauses(
		store.StringClause("session_id", query.SessionID),
		store.StringClause("channel", query.Channel),
		store.StringClause("peer_from", query.PeerFrom),
		store.StringClause("message_id", query.MessageID),
		store.TimeClause("timestamp", ">=", query.Since),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY timestamp ASC, message_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network messages: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network messages rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	entries = make([]store.NetworkMessageEntry, 0)
	for rows.Next() {
		entry, scanErr := scanNetworkMessage(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network messages: %w", err)
	}

	return entries, nil
}

func scanNetworkMessage(scanner rowScanner) (store.NetworkMessageEntry, error) {
	var (
		entry        store.NetworkMessageEntry
		sessionID    sql.NullString
		intent       sql.NullString
		timestampRaw string
	)
	if err := scanner.Scan(
		&entry.MessageID,
		&sessionID,
		&entry.Channel,
		&entry.PeerFrom,
		&entry.Kind,
		&intent,
		&entry.Text,
		&timestampRaw,
	); err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: scan network message: %w", err)
	}

	if value := store.NullString(sessionID); value != nil {
		entry.SessionID = *value
	}
	if value := store.NullString(intent); value != nil {
		entry.Intent = strings.TrimSpace(*value)
	}

	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: parse network message timestamp: %w", err)
	}
	entry.Timestamp = timestamp
	return entry, nil
}
