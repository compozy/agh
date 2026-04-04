package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GlobalDB owns the global session index and observability database.
type GlobalDB struct {
	db   *sql.DB
	path string
	now  func() time.Time
}

var _ SessionRegistry = (*GlobalDB)(nil)

// OpenGlobalDB opens or creates the global AGH index database.
func OpenGlobalDB(ctx context.Context, path string) (*GlobalDB, error) {
	if ctx == nil {
		return nil, errors.New("store: open global database context is required")
	}

	db, err := openGlobalSQLite(ctx, path)
	if err != nil {
		return nil, err
	}

	return &GlobalDB{
		db:   db,
		path: strings.TrimSpace(path),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

// Path reports the on-disk path for the global database file.
func (g *GlobalDB) Path() string {
	if g == nil {
		return ""
	}
	return g.path
}

// RegisterSession inserts or refreshes a session index row.
func (g *GlobalDB) RegisterSession(ctx context.Context, session SessionInfo) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: register session context is required")
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
func (g *GlobalDB) UpdateSessionState(ctx context.Context, update SessionStateUpdate) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: update session state context is required")
	}
	if err := update.Validate(); err != nil {
		return err
	}

	updatedAt := update.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = g.now()
	}

	var (
		query string
		args  []any
	)
	if update.ACPSessionID != nil {
		query = `UPDATE sessions SET state = ?, acp_session_id = ?, updated_at = ? WHERE id = ?`
		args = []any{
			update.State,
			nullableStringPointer(update.ACPSessionID),
			formatTimestamp(updatedAt),
			update.ID,
		}
	} else {
		query = `UPDATE sessions SET state = ?, updated_at = ? WHERE id = ?`
		args = []any{
			update.State,
			formatTimestamp(updatedAt),
			update.ID,
		}
	}

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
func (g *GlobalDB) ListSessions(ctx context.Context, query SessionListQuery) ([]SessionInfo, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: list sessions context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at FROM sessions`
	where, args := buildClauses(
		stringClause("state", query.State),
		stringClause("agent_name", query.AgentName),
	)
	sqlQuery = appendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, created_at DESC, id DESC"
	sqlQuery, args = appendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query sessions: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]SessionInfo, 0)
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
func (g *GlobalDB) ReconcileSessions(ctx context.Context, sessions []SessionInfo) (ReconcileResult, error) {
	if g == nil {
		return ReconcileResult{}, errors.New("store: global database is required")
	}
	if ctx == nil {
		return ReconcileResult{}, errors.New("store: reconcile sessions context is required")
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("store: begin session reconcile transaction: %w", err)
	}

	existing, err := g.loadSessionIDs(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return ReconcileResult{}, err
	}

	result := ReconcileResult{
		Indexed:  make([]string, 0),
		Orphaned: make([]string, 0),
	}
	seen := make(map[string]struct{}, len(sessions))

	for _, session := range sessions {
		if err := session.Validate(); err != nil {
			_ = tx.Rollback()
			return ReconcileResult{}, err
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
			return ReconcileResult{}, fmt.Errorf("store: reconcile session %q: %w", normalized.ID, err)
		}
	}

	orphanedAt := formatTimestamp(g.now())
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
			return ReconcileResult{}, fmt.Errorf("store: mark orphaned session %q: %w", id, err)
		}
		result.Orphaned = append(result.Orphaned, id)
	}

	if err := tx.Commit(); err != nil {
		return ReconcileResult{}, fmt.Errorf("store: commit session reconcile transaction: %w", err)
	}

	return result, nil
}

// WriteEventSummary stores a lightweight cross-session summary entry.
func (g *GlobalDB) WriteEventSummary(ctx context.Context, summary EventSummary) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: write event summary context is required")
	}
	if err := summary.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(summary.ID) == "" {
		summary.ID = newID("sum")
	}
	if summary.Timestamp.IsZero() {
		summary.Timestamp = g.now()
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO event_summaries (id, session_id, type, agent_name, summary, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		summary.ID,
		summary.SessionID,
		summary.Type,
		summary.AgentName,
		nullableString(summary.Summary),
		formatTimestamp(summary.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert event summary: %w", err)
	}
	return nil
}

// ListEventSummaries returns global event summaries filtered by the supplied options.
func (g *GlobalDB) ListEventSummaries(ctx context.Context, query EventSummaryQuery) ([]EventSummary, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: list event summaries context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	baseQuery := `SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`
	where, args := buildClauses(
		stringClause("session_id", query.SessionID),
		stringClause("agent_name", query.AgentName),
		stringClause("type", query.Type),
		timeClause("timestamp", ">=", query.Since),
	)
	baseQuery = appendWhere(baseQuery, where)

	sqlQuery := baseQuery
	if query.Limit > 0 {
		sqlQuery = `SELECT id, session_id, type, agent_name, summary, timestamp
			FROM (` + baseQuery + ` ORDER BY timestamp DESC LIMIT ?) AS recent_summaries
			ORDER BY timestamp ASC, id ASC`
		args = append(args, query.Limit)
	} else {
		sqlQuery += " ORDER BY timestamp ASC, id ASC"
	}

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query event summaries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summaries := make([]EventSummary, 0)
	for rows.Next() {
		summary, scanErr := scanEventSummary(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate event summaries: %w", err)
	}

	return summaries, nil
}

// UpdateTokenStats merges one or more turns of token usage into the session aggregate.
func (g *GlobalDB) UpdateTokenStats(ctx context.Context, update TokenStatsUpdate) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: update token stats context is required")
	}
	if err := update.Validate(); err != nil {
		return err
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = g.now()
	}
	if update.Turns <= 0 {
		update.Turns = 1
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO token_stats (
			id, session_id, agent_name, input_tokens, output_tokens, total_tokens,
			total_cost, cost_currency, turn_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, agent_name) DO UPDATE SET
			input_tokens = CASE
				WHEN excluded.input_tokens IS NULL THEN token_stats.input_tokens
				WHEN token_stats.input_tokens IS NULL THEN excluded.input_tokens
				ELSE token_stats.input_tokens + excluded.input_tokens
			END,
			output_tokens = CASE
				WHEN excluded.output_tokens IS NULL THEN token_stats.output_tokens
				WHEN token_stats.output_tokens IS NULL THEN excluded.output_tokens
				ELSE token_stats.output_tokens + excluded.output_tokens
			END,
			total_tokens = CASE
				WHEN excluded.total_tokens IS NULL THEN token_stats.total_tokens
				WHEN token_stats.total_tokens IS NULL THEN excluded.total_tokens
				ELSE token_stats.total_tokens + excluded.total_tokens
			END,
			total_cost = CASE
				WHEN excluded.total_cost IS NULL THEN token_stats.total_cost
				WHEN token_stats.total_cost IS NULL THEN excluded.total_cost
				ELSE token_stats.total_cost + excluded.total_cost
			END,
			cost_currency = COALESCE(excluded.cost_currency, token_stats.cost_currency),
			turn_count = token_stats.turn_count + excluded.turn_count,
			updated_at = excluded.updated_at`,
		newID("tok"),
		update.SessionID,
		update.AgentName,
		nullableInt64(update.InputTokens),
		nullableInt64(update.OutputTokens),
		nullableInt64(update.TotalTokens),
		nullableFloat64(update.CostAmount),
		nullableStringPointer(update.CostCurrency),
		update.Turns,
		formatTimestamp(update.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert token stats for session %q: %w", update.SessionID, err)
	}

	return nil
}

// ListTokenStats returns aggregated token usage rows.
func (g *GlobalDB) ListTokenStats(ctx context.Context, query TokenStatsQuery) ([]TokenStats, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: list token stats context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at FROM token_stats`
	where, args := buildClauses(
		stringClause("session_id", query.SessionID),
		stringClause("agent_name", query.AgentName),
	)
	sqlQuery = appendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, id DESC"
	sqlQuery, args = appendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query token stats: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	stats := make([]TokenStats, 0)
	for rows.Next() {
		stat, scanErr := scanTokenStats(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate token stats: %w", err)
	}

	return stats, nil
}

// WritePermissionLog stores one permission decision audit row.
func (g *GlobalDB) WritePermissionLog(ctx context.Context, entry PermissionLogEntry) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if ctx == nil {
		return errors.New("store: write permission log context is required")
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = newID("perm")
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
		formatTimestamp(entry.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert permission log entry: %w", err)
	}
	return nil
}

// ListPermissionLog returns permission audit rows filtered by the supplied options.
func (g *GlobalDB) ListPermissionLog(ctx context.Context, query PermissionLogQuery) ([]PermissionLogEntry, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if ctx == nil {
		return nil, errors.New("store: list permission log context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, agent_name, action, resource, decision, policy_used, timestamp FROM permission_log`
	where, args := buildClauses(
		stringClause("session_id", query.SessionID),
		stringClause("agent_name", query.AgentName),
		stringClause("decision", query.Decision),
		timeClause("timestamp", ">=", query.Since),
	)
	sqlQuery = appendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY timestamp ASC, id ASC"
	sqlQuery, args = appendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query permission log: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	entries := make([]PermissionLogEntry, 0)
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

// Close checkpoints the WAL and closes the database.
func (g *GlobalDB) Close(ctx context.Context) error {
	if g == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close global database context is required")
	}

	checkpointErr := checkpoint(ctx, g.db)
	closeErr := g.db.Close()
	return errors.Join(checkpointErr, closeErr)
}

func (g *GlobalDB) registerSession(ctx context.Context, exec sqlExecutor, session SessionInfo) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO sessions (
			id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			agent_name = excluded.agent_name,
			workspace = excluded.workspace,
			session_type = excluded.session_type,
			state = excluded.state,
			acp_session_id = excluded.acp_session_id,
			updated_at = excluded.updated_at`,
		session.ID,
		nullableString(session.Name),
		session.AgentName,
		session.Workspace,
		normalizeSessionType(session.SessionType),
		session.State,
		nullableStringPointer(session.ACPSessionID),
		formatTimestamp(session.CreatedAt),
		formatTimestamp(session.UpdatedAt),
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

func scanSessionInfo(scanner rowScanner) (SessionInfo, error) {
	var (
		session      SessionInfo
		name         sql.NullString
		sessionType  string
		acpSessionID sql.NullString
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&session.ID,
		&name,
		&session.AgentName,
		&session.Workspace,
		&sessionType,
		&session.State,
		&acpSessionID,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return SessionInfo{}, fmt.Errorf("store: scan session info: %w", err)
	}

	if name.Valid {
		session.Name = name.String
	}
	session.SessionType = normalizeSessionType(sessionType)
	session.ACPSessionID = nullString(acpSessionID)

	createdAt, err := parseTimestamp(createdAtRaw)
	if err != nil {
		return SessionInfo{}, err
	}
	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return SessionInfo{}, err
	}
	session.CreatedAt = createdAt
	session.UpdatedAt = updatedAt

	return session, nil
}

func scanEventSummary(scanner rowScanner) (EventSummary, error) {
	var (
		summary      EventSummary
		summaryText  sql.NullString
		timestampRaw string
	)
	if err := scanner.Scan(
		&summary.ID,
		&summary.SessionID,
		&summary.Type,
		&summary.AgentName,
		&summaryText,
		&timestampRaw,
	); err != nil {
		return EventSummary{}, fmt.Errorf("store: scan event summary: %w", err)
	}

	if summaryText.Valid {
		summary.Summary = summaryText.String
	}
	timestamp, err := parseTimestamp(timestampRaw)
	if err != nil {
		return EventSummary{}, err
	}
	summary.Timestamp = timestamp
	return summary, nil
}

func scanTokenStats(scanner rowScanner) (TokenStats, error) {
	var (
		stats        TokenStats
		inputTokens  sql.NullInt64
		outputTokens sql.NullInt64
		totalTokens  sql.NullInt64
		totalCost    sql.NullFloat64
		costCurrency sql.NullString
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&stats.ID,
		&stats.SessionID,
		&stats.AgentName,
		&inputTokens,
		&outputTokens,
		&totalTokens,
		&totalCost,
		&costCurrency,
		&stats.TurnCount,
		&updatedAtRaw,
	); err != nil {
		return TokenStats{}, fmt.Errorf("store: scan token stats: %w", err)
	}

	stats.InputTokens = nullInt64(inputTokens)
	stats.OutputTokens = nullInt64(outputTokens)
	stats.TotalTokens = nullInt64(totalTokens)
	stats.TotalCost = nullFloat64(totalCost)
	stats.CostCurrency = nullString(costCurrency)

	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return TokenStats{}, err
	}
	stats.UpdatedAt = updatedAt

	return stats, nil
}

func scanPermissionLog(scanner rowScanner) (PermissionLogEntry, error) {
	var (
		entry        PermissionLogEntry
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
		return PermissionLogEntry{}, fmt.Errorf("store: scan permission log: %w", err)
	}

	timestamp, err := parseTimestamp(timestampRaw)
	if err != nil {
		return PermissionLogEntry{}, err
	}
	entry.Timestamp = timestamp
	return entry, nil
}
