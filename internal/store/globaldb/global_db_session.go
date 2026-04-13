package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/store"
)

// RegisterSession inserts or refreshes a session index row.
func (g *GlobalDB) RegisterSession(ctx context.Context, session store.SessionInfo) error {
	if err := g.checkReady(ctx, "register session"); err != nil {
		return err
	}
	if err := session.Validate(); err != nil {
		return err
	}

	normalized := session
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}

	if err := g.registerSession(ctx, g.db, normalized); err != nil {
		return fmt.Errorf("store: register session %q: %w", normalized.ID, err)
	}
	return nil
}

// UpdateSessionState updates the mutable session state fields.
func (g *GlobalDB) UpdateSessionState(ctx context.Context, update store.SessionStateUpdate) error {
	if err := g.checkReady(ctx, "update session state"); err != nil {
		return err
	}
	if err := update.Validate(); err != nil {
		return err
	}

	updatedAt := update.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = g.now()
	}

	assignments := []string{"state = ?"}
	args := []any{update.State}
	if update.ACPSessionID != nil {
		assignments = append(assignments, "acp_session_id = ?")
		args = append(args, store.NullableStringPointer(update.ACPSessionID))
	}
	if update.StopReasonSet {
		assignments = append(assignments, "stop_reason = ?", "stop_detail = ?")
		args = append(args, store.NullableStringPointer(update.StopReason), store.NullableString(update.StopDetail))
	}
	assignments = append(assignments, "updated_at = ?")
	args = append(args, store.FormatTimestamp(updatedAt), update.ID)

	query := fmt.Sprintf("UPDATE sessions SET %s WHERE id = ?", strings.Join(assignments, ", "))

	result, err := g.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("store: update session state %q: %w", update.ID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: rows affected for session state %q: %w", update.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("store: session %q not found", update.ID)
	}
	return nil
}

// ListSessions returns indexed sessions ordered by most recent update.
func (g *GlobalDB) ListSessions(ctx context.Context, query store.SessionListQuery) ([]store.SessionInfo, error) {
	if err := g.checkReady(ctx, "list sessions"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, name, agent_name, workspace_id, channel, session_type, state, acp_session_id, stop_reason, stop_detail, created_at, updated_at FROM sessions`
	where, args := store.BuildClauses(
		store.StringClause("state", query.State),
		store.StringClause("agent_name", query.AgentName),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, created_at DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query sessions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]store.SessionInfo, 0)
	for rows.Next() {
		session, scanErr := scanSessionInfo(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate sessions: %w", err)
	}

	return sessions, nil
}

// ReconcileSessions upserts on-disk sessions and marks missing ones as orphaned.
func (g *GlobalDB) ReconcileSessions(ctx context.Context, sessions []store.SessionInfo) (store.ReconcileResult, error) {
	if err := g.checkReady(ctx, "reconcile sessions"); err != nil {
		return store.ReconcileResult{}, err
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ReconcileResult{}, fmt.Errorf("store: begin session reconcile transaction: %w", err)
	}

	existing, err := g.loadSessionIDs(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return store.ReconcileResult{}, err
	}

	result := store.ReconcileResult{
		Indexed:  make([]string, 0),
		Orphaned: make([]string, 0),
	}
	seen := make(map[string]struct{}, len(sessions))

	for _, session := range sessions {
		if err := session.Validate(); err != nil {
			_ = tx.Rollback()
			return store.ReconcileResult{}, err
		}
		normalized := session
		if normalized.CreatedAt.IsZero() {
			normalized.CreatedAt = g.now()
		}
		if normalized.UpdatedAt.IsZero() {
			normalized.UpdatedAt = normalized.CreatedAt
		}
		if _, ok := seen[normalized.ID]; ok {
			continue
		}
		seen[normalized.ID] = struct{}{}
		if _, ok := existing[normalized.ID]; !ok {
			result.Indexed = append(result.Indexed, normalized.ID)
		}
		if err := g.registerSession(ctx, tx, normalized); err != nil {
			_ = tx.Rollback()
			return store.ReconcileResult{}, fmt.Errorf("store: reconcile session %q: %w", normalized.ID, err)
		}
	}

	orphanedAt := store.FormatTimestamp(g.now())
	for id := range existing {
		if _, ok := seen[id]; ok {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE sessions SET state = ?, updated_at = ? WHERE id = ?`,
			"orphaned",
			orphanedAt,
			id,
		); err != nil {
			_ = tx.Rollback()
			return store.ReconcileResult{}, fmt.Errorf("store: mark orphaned session %q: %w", id, err)
		}
		result.Orphaned = append(result.Orphaned, id)
	}

	if err := tx.Commit(); err != nil {
		return store.ReconcileResult{}, fmt.Errorf("store: commit session reconcile transaction: %w", err)
	}

	return result, nil
}

func (g *GlobalDB) registerSession(ctx context.Context, exec sqlExecutor, session store.SessionInfo) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, name, agent_name, workspace_id, session_type, channel, state, acp_session_id, stop_reason, stop_detail, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			agent_name = excluded.agent_name,
			workspace_id = excluded.workspace_id,
			session_type = excluded.session_type,
			channel = excluded.channel,
			state = excluded.state,
			acp_session_id = excluded.acp_session_id,
			stop_reason = excluded.stop_reason,
			stop_detail = excluded.stop_detail,
			updated_at = excluded.updated_at`,
		session.ID,
		store.NullableString(session.Name),
		session.AgentName,
		session.WorkspaceID,
		store.NormalizeSessionType(session.SessionType),
		strings.TrimSpace(session.Channel),
		session.State,
		store.NullableStringPointer(session.ACPSessionID),
		store.NullableString(string(session.StopReason)),
		store.NullableString(session.StopDetail),
		store.FormatTimestamp(session.CreatedAt),
		store.FormatTimestamp(session.UpdatedAt),
	)
	return err
}

func (g *GlobalDB) loadSessionIDs(ctx context.Context, tx *sql.Tx) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("store: query existing session ids: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("store: scan existing session id: %w", err)
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate existing session ids: %w", err)
	}

	return ids, nil
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func scanSessionInfo(scanner rowScanner) (store.SessionInfo, error) {
	var (
		session      store.SessionInfo
		name         sql.NullString
		channel      string
		sessionType  string
		acpSessionID sql.NullString
		stopReason   sql.NullString
		stopDetail   sql.NullString
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&session.ID,
		&name,
		&session.AgentName,
		&session.WorkspaceID,
		&channel,
		&sessionType,
		&session.State,
		&acpSessionID,
		&stopReason,
		&stopDetail,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return store.SessionInfo{}, fmt.Errorf("store: scan session info: %w", err)
	}

	if name.Valid {
		session.Name = name.String
	}
	session.Channel = strings.TrimSpace(channel)
	session.SessionType = store.NormalizeSessionType(sessionType)
	session.ACPSessionID = store.NullString(acpSessionID)
	if reason := store.NullString(stopReason); reason != nil {
		session.StopReason = store.StopReason(*reason)
	}
	if detail := store.NullString(stopDetail); detail != nil {
		session.StopDetail = *detail
	}

	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return store.SessionInfo{}, err
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.SessionInfo{}, err
	}
	session.CreatedAt = createdAt
	session.UpdatedAt = updatedAt

	return session, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}
