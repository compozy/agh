package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

type networkMessageCursor struct {
	MessageID string
	Timestamp string
}

// WriteNetworkMessage stores one persisted network timeline envelope, ignoring duplicate message ids.
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
		`INSERT INTO network_timeline_log (
			message_id,
			session_id,
			channel,
			direction,
			peer_from,
			peer_to,
			kind,
			interaction_id,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			body_json,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO NOTHING`,
		entry.MessageID,
		store.NullableString(entry.SessionID),
		entry.Channel,
		entry.Direction,
		entry.PeerFrom,
		store.NullableString(entry.PeerTo),
		entry.Kind,
		store.NullableString(entry.InteractionID),
		store.NullableString(entry.ReplyTo),
		store.NullableString(entry.TraceID),
		store.NullableString(entry.CausationID),
		store.NullableString(entry.Intent),
		store.NullableString(entry.Text),
		entry.PreviewText,
		string(entry.Body),
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

	sqlQuery, args, reverseResults, err := g.buildNetworkMessageListQuery(ctx, query)
	if err != nil {
		return nil, err
	}

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

	entries, err = loadNetworkMessageEntries(rows)
	if err != nil {
		return nil, err
	}
	if reverseResults {
		reverseNetworkMessages(entries)
	}

	return entries, nil
}

func (g *GlobalDB) buildNetworkMessageListQuery(
	ctx context.Context,
	query store.NetworkMessageQuery,
) (string, []any, bool, error) {
	sqlQuery := `SELECT
		message_id,
		session_id,
		channel,
		direction,
		peer_from,
		peer_to,
		kind,
		interaction_id,
		reply_to,
		trace_id,
		causation_id,
		intent,
		text,
		preview_text,
		body_json,
		timestamp
	FROM network_timeline_log`

	where, args := networkMessageFilterClauses(query, true)

	reverseResults := false
	switch {
	case strings.TrimSpace(query.BeforeMessageID) != "":
		cursor, cursorErr := g.lookupNetworkMessageCursor(ctx, query.BeforeMessageID, query)
		if cursorErr != nil {
			return "", nil, false, cursorErr
		}
		where = append(where, "(timestamp < ? OR (timestamp = ? AND message_id < ?))")
		args = append(args, cursor.Timestamp, cursor.Timestamp, cursor.MessageID)
		reverseResults = true
	case strings.TrimSpace(query.AfterMessageID) != "":
		cursor, cursorErr := g.lookupNetworkMessageCursor(ctx, query.AfterMessageID, query)
		if cursorErr != nil {
			return "", nil, false, cursorErr
		}
		where = append(where, "(timestamp > ? OR (timestamp = ? AND message_id > ?))")
		args = append(args, cursor.Timestamp, cursor.Timestamp, cursor.MessageID)
	}

	sqlQuery = store.AppendWhere(sqlQuery, where)
	if reverseResults {
		sqlQuery += " ORDER BY timestamp DESC, message_id DESC"
	} else {
		sqlQuery += " ORDER BY timestamp ASC, message_id ASC"
	}
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)
	return sqlQuery, args, reverseResults, nil
}

func networkMessageFilterClauses(query store.NetworkMessageQuery, includeMessageID bool) ([]string, []any) {
	messageID := ""
	if includeMessageID {
		messageID = query.MessageID
	}
	where, args := store.BuildClauses(
		store.StringClause("session_id", query.SessionID),
		store.StringClause("channel", query.Channel),
		store.StringClause("peer_from", query.PeerFrom),
		store.StringClause("peer_to", query.PeerTo),
		store.StringClause("kind", query.Kind),
		store.StringClause("direction", query.Direction),
		store.StringClause("message_id", messageID),
		store.TimeClause("timestamp", ">=", query.Since),
	)
	if peerID := strings.TrimSpace(query.PeerID); peerID != "" {
		where = append(where, "(peer_from = ? OR peer_to = ?)")
		args = append(args, peerID, peerID)
	}
	if query.DirectedOnly {
		where = append(where, "peer_to IS NOT NULL AND TRIM(peer_to) <> ''")
	}
	return where, args
}

func loadNetworkMessageEntries(rows *sql.Rows) ([]store.NetworkMessageEntry, error) {
	entries := make([]store.NetworkMessageEntry, 0)
	for rows.Next() {
		entry, err := scanNetworkMessage(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network messages: %w", err)
	}
	return entries, nil
}

func reverseNetworkMessages(entries []store.NetworkMessageEntry) {
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
}

func (g *GlobalDB) lookupNetworkMessageCursor(
	ctx context.Context,
	messageID string,
	query store.NetworkMessageQuery,
) (networkMessageCursor, error) {
	trimmed := strings.TrimSpace(messageID)
	if trimmed == "" {
		return networkMessageCursor{}, nil
	}

	where, args := networkMessageFilterClauses(query, false)
	where = append([]string{"message_id = ?"}, where...)
	args = append([]any{trimmed}, args...)

	var cursor networkMessageCursor
	err := g.db.QueryRowContext(
		ctx,
		store.AppendWhere(`SELECT message_id, timestamp FROM network_timeline_log`, where),
		args...,
	).Scan(&cursor.MessageID, &cursor.Timestamp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return networkMessageCursor{}, fmt.Errorf("store: network message cursor not found: %w", err)
		}
		return networkMessageCursor{}, fmt.Errorf("store: lookup network message cursor: %w", err)
	}
	return cursor, nil
}

func scanNetworkMessage(scanner rowScanner) (store.NetworkMessageEntry, error) {
	var (
		entry         store.NetworkMessageEntry
		sessionID     sql.NullString
		peerTo        sql.NullString
		interactionID sql.NullString
		replyTo       sql.NullString
		traceID       sql.NullString
		causationID   sql.NullString
		intent        sql.NullString
		text          sql.NullString
		previewText   string
		bodyRaw       string
		timestampRaw  string
	)
	if err := scanner.Scan(
		&entry.MessageID,
		&sessionID,
		&entry.Channel,
		&entry.Direction,
		&entry.PeerFrom,
		&peerTo,
		&entry.Kind,
		&interactionID,
		&replyTo,
		&traceID,
		&causationID,
		&intent,
		&text,
		&previewText,
		&bodyRaw,
		&timestampRaw,
	); err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: scan network message: %w", err)
	}

	if value := store.NullString(sessionID); value != nil {
		entry.SessionID = *value
	}
	if value := store.NullString(peerTo); value != nil {
		entry.PeerTo = *value
	}
	if value := store.NullString(interactionID); value != nil {
		entry.InteractionID = *value
	}
	if value := store.NullString(replyTo); value != nil {
		entry.ReplyTo = *value
	}
	if value := store.NullString(traceID); value != nil {
		entry.TraceID = *value
	}
	if value := store.NullString(causationID); value != nil {
		entry.CausationID = *value
	}
	if value := store.NullString(intent); value != nil {
		entry.Intent = *value
	}
	if value := store.NullString(text); value != nil {
		entry.Text = *value
	}
	entry.PreviewText = strings.TrimSpace(previewText)
	entry.Body = []byte(bodyRaw)

	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: parse network message timestamp: %w", err)
	}
	entry.Timestamp = timestamp
	return entry, nil
}
