package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

type networkMessageCursor struct {
	MessageID string
	Timestamp string
}

type networkMessageNullableFields struct {
	sessionID    sql.NullString
	surface      sql.NullString
	threadID     sql.NullString
	directID     sql.NullString
	peerTo       sql.NullString
	workID       sql.NullString
	replyTo      sql.NullString
	traceID      sql.NullString
	causationID  sql.NullString
	intent       sql.NullString
	text         sql.NullString
	previewText  string
	extRaw       string
	bodyRaw      string
	timestampRaw string
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
				workspace_id,
				channel,
			surface,
			thread_id,
			direct_id,
			direction,
			peer_from,
			peer_to,
			kind,
			work_id,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			ext_json,
			body_json,
			timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(workspace_id, message_id) DO NOTHING`,
		entry.MessageID,
		store.NullableString(entry.SessionID),
		entry.WorkspaceID,
		entry.Channel,
		store.NullableString(entry.Surface),
		store.NullableString(entry.ThreadID),
		store.NullableString(entry.DirectID),
		entry.Direction,
		entry.PeerFrom,
		store.NullableString(entry.PeerTo),
		entry.Kind,
		store.NullableString(entry.WorkID),
		store.NullableString(entry.ReplyTo),
		store.NullableString(entry.TraceID),
		store.NullableString(entry.CausationID),
		store.NullableString(entry.Intent),
		store.NullableString(entry.Text),
		entry.PreviewText,
		networkMessageExtJSONString(entry.ExtJSON),
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
			workspace_id,
			channel,
		surface,
		thread_id,
		direct_id,
		direction,
		peer_from,
		peer_to,
		kind,
		work_id,
		reply_to,
		trace_id,
		causation_id,
		intent,
		text,
		preview_text,
		ext_json,
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
		store.StringClause("workspace_id", query.WorkspaceID),
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
		entry    store.NetworkMessageEntry
		nullable networkMessageNullableFields
	)
	if err := scanner.Scan(
		&entry.MessageID,
		&nullable.sessionID,
		&entry.WorkspaceID,
		&entry.Channel,
		&nullable.surface,
		&nullable.threadID,
		&nullable.directID,
		&entry.Direction,
		&entry.PeerFrom,
		&nullable.peerTo,
		&entry.Kind,
		&nullable.workID,
		&nullable.replyTo,
		&nullable.traceID,
		&nullable.causationID,
		&nullable.intent,
		&nullable.text,
		&nullable.previewText,
		&nullable.extRaw,
		&nullable.bodyRaw,
		&nullable.timestampRaw,
	); err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: scan network message: %w", err)
	}

	applyNetworkMessageNullableFields(&entry, nullable)

	timestamp, err := store.ParseTimestamp(nullable.timestampRaw)
	if err != nil {
		return store.NetworkMessageEntry{}, fmt.Errorf("store: parse network message timestamp: %w", err)
	}
	entry.Timestamp = timestamp
	return entry, nil
}

func applyNetworkMessageNullableFields(entry *store.NetworkMessageEntry, nullable networkMessageNullableFields) {
	if value := store.NullString(nullable.sessionID); value != nil {
		entry.SessionID = *value
	}
	if value := store.NullString(nullable.surface); value != nil {
		entry.Surface = *value
	}
	if value := store.NullString(nullable.threadID); value != nil {
		entry.ThreadID = *value
	}
	if value := store.NullString(nullable.directID); value != nil {
		entry.DirectID = *value
	}
	if value := store.NullString(nullable.peerTo); value != nil {
		entry.PeerTo = *value
	}
	if value := store.NullString(nullable.workID); value != nil {
		entry.WorkID = *value
	}
	if value := store.NullString(nullable.replyTo); value != nil {
		entry.ReplyTo = *value
	}
	if value := store.NullString(nullable.traceID); value != nil {
		entry.TraceID = *value
	}
	if value := store.NullString(nullable.causationID); value != nil {
		entry.CausationID = *value
	}
	if value := store.NullString(nullable.intent); value != nil {
		entry.Intent = *value
	}
	if value := store.NullString(nullable.text); value != nil {
		entry.Text = *value
	}
	entry.PreviewText = strings.TrimSpace(nullable.previewText)
	entry.ExtJSON = []byte(networkMessageExtJSONString([]byte(nullable.extRaw)))
	entry.Body = []byte(nullable.bodyRaw)
}

func networkMessageExtJSONString(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
