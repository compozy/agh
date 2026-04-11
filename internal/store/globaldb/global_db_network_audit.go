package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

// WriteNetworkAudit stores one network audit row.
func (g *GlobalDB) WriteNetworkAudit(ctx context.Context, entry store.NetworkAuditEntry) error {
	if err := g.checkReady(ctx, "write network audit"); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = store.NewID("naud")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = g.now()
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_audit_log (
			id, session_id, direction, kind, space, peer_from, peer_to, message_id, reason, size, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.SessionID,
		entry.Direction,
		entry.Kind,
		entry.Space,
		entry.PeerFrom,
		store.NullableString(entry.PeerTo),
		entry.MessageID,
		store.NullableString(entry.Reason),
		entry.Size,
		store.FormatTimestamp(entry.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert network audit entry: %w", err)
	}

	return nil
}

// ListNetworkAudit returns network audit rows filtered by the supplied options.
func (g *GlobalDB) ListNetworkAudit(ctx context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error) {
	if err := g.checkReady(ctx, "list network audit"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, direction, kind, space, peer_from, peer_to, message_id, reason, size, timestamp FROM network_audit_log`
	where, args := store.BuildClauses(
		store.StringClause("session_id", query.SessionID),
		store.StringClause("direction", query.Direction),
		store.StringClause("kind", query.Kind),
		store.StringClause("space", query.Space),
		store.StringClause("message_id", query.MessageID),
		store.TimeClause("timestamp", ">=", query.Since),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY timestamp ASC, id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network audit: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := make([]store.NetworkAuditEntry, 0)
	for rows.Next() {
		entry, scanErr := scanNetworkAudit(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network audit: %w", err)
	}

	return entries, nil
}

func scanNetworkAudit(scanner rowScanner) (store.NetworkAuditEntry, error) {
	var (
		entry        store.NetworkAuditEntry
		peerTo       sql.NullString
		reason       sql.NullString
		timestampRaw string
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.SessionID,
		&entry.Direction,
		&entry.Kind,
		&entry.Space,
		&entry.PeerFrom,
		&peerTo,
		&entry.MessageID,
		&reason,
		&entry.Size,
		&timestampRaw,
	); err != nil {
		return store.NetworkAuditEntry{}, fmt.Errorf("store: scan network audit: %w", err)
	}

	if value := store.NullString(peerTo); value != nil {
		entry.PeerTo = *value
	}
	if value := store.NullString(reason); value != nil {
		entry.Reason = *value
	}

	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return store.NetworkAuditEntry{}, err
	}
	entry.Timestamp = timestamp
	return entry, nil
}
