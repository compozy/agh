package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

// WritePermissionLog stores one permission decision audit row.
func (g *GlobalDB) WritePermissionLog(ctx context.Context, entry store.PermissionLogEntry) error {
	if err := g.checkReady(ctx, "write permission log"); err != nil {
		return err
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = store.NewID("perm")
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = g.now()
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO permission_log (id, session_id, agent_name, action, resource, decision, policy_used, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.SessionID,
		entry.AgentName,
		entry.Action,
		entry.Resource,
		entry.Decision,
		entry.PolicyUsed,
		store.FormatTimestamp(entry.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert permission log entry: %w", err)
	}
	return nil
}

// ListPermissionLog returns permission audit rows filtered by the supplied options.
func (g *GlobalDB) ListPermissionLog(ctx context.Context, query store.PermissionLogQuery) ([]store.PermissionLogEntry, error) {
	if err := g.checkReady(ctx, "list permission log"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, agent_name, action, resource, decision, policy_used, timestamp FROM permission_log`
	where, args := store.BuildClauses(
		store.StringClause("session_id", query.SessionID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("decision", query.Decision),
		store.TimeClause("timestamp", ">=", query.Since),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY timestamp ASC, id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query permission log: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := make([]store.PermissionLogEntry, 0)
	for rows.Next() {
		entry, scanErr := scanPermissionLog(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate permission log: %w", err)
	}

	return entries, nil
}

func scanPermissionLog(scanner rowScanner) (store.PermissionLogEntry, error) {
	var (
		entry        store.PermissionLogEntry
		timestampRaw string
	)
	if err := scanner.Scan(
		&entry.ID,
		&entry.SessionID,
		&entry.AgentName,
		&entry.Action,
		&entry.Resource,
		&entry.Decision,
		&entry.PolicyUsed,
		&timestampRaw,
	); err != nil {
		return store.PermissionLogEntry{}, fmt.Errorf("store: scan permission log: %w", err)
	}

	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return store.PermissionLogEntry{}, err
	}
	entry.Timestamp = timestamp
	return entry, nil
}
