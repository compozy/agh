package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

type sqlQueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type legacySessionRow struct {
	ID           string
	Name         sql.NullString
	AgentName    string
	Workspace    string
	SessionType  string
	State        string
	ACPSessionID sql.NullString
	CreatedAt    string
	UpdatedAt    string
}

type legacyWorkspaceSeed struct {
	rootDir   string
	createdAt string
	updatedAt string
}

type legacySessionMetaCompat struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	AgentName    string    `json:"agent_name"`
	Workspace    string    `json:"workspace,omitempty"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	SessionType  string    `json:"session_type,omitempty"`
	State        string    `json:"state"`
	ACPSessionID *string   `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func migrateGlobalSchema(ctx context.Context, db *sql.DB) error {
	hasSessions, err := tableExists(ctx, db, "sessions")
	if err != nil {
		return err
	}
	if !hasSessions {
		return nil
	}

	columns, err := tableColumns(ctx, db, "sessions")
	if err != nil {
		return err
	}

	_, hasWorkspaceID := columns["workspace_id"]
	_, hasLegacyWorkspace := columns["workspace"]
	if !hasWorkspaceID && hasLegacyWorkspace {
		conn, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("store: open global schema migration connection: %w", err)
		}
		foreignKeysDisabled := false
		defer func() {
			if foreignKeysDisabled {
				_, _ = conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
			}
			_ = conn.Close()
		}()

		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
			return fmt.Errorf("store: disable foreign keys for global schema migration: %w", err)
		}
		foreignKeysDisabled = true

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: begin global schema migration transaction: %w", err)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		if _, err := tx.ExecContext(ctx, globalSchemaStatements[0]); err != nil {
			return fmt.Errorf("store: create workspaces table during migration: %w", err)
		}

		sessionRows, workspaceSeeds, err := loadLegacySessions(ctx, tx)
		if err != nil {
			return err
		}

		workspaceIDs, err := ensureMigratedWorkspaces(ctx, tx, workspaceSeeds)
		if err != nil {
			return err
		}

		if err := createMigratedGlobalTables(ctx, tx); err != nil {
			return err
		}
		if err := copyMigratedSessions(ctx, tx, sessionRows, workspaceIDs); err != nil {
			return err
		}
		if err := copyGlobalTableIfExists(ctx, tx, "event_summaries", "event_summaries_new", `INSERT INTO event_summaries_new (id, session_id, type, agent_name, summary, timestamp) SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`); err != nil {
			return err
		}
		if err := copyGlobalTableIfExists(ctx, tx, "token_stats", "token_stats_new", `INSERT INTO token_stats_new (id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at) SELECT id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at FROM token_stats`); err != nil {
			return err
		}
		if err := copyGlobalTableIfExists(ctx, tx, "permission_log", "permission_log_new", `INSERT INTO permission_log_new (id, session_id, agent_name, action, resource, decision, policy_used, timestamp) SELECT id, session_id, agent_name, action, resource, decision, policy_used, timestamp FROM permission_log`); err != nil {
			return err
		}
		if err := swapMigratedGlobalTables(ctx, tx); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: commit global schema migration: %w", err)
		}
	}

	if err := migrateSessionColumns(ctx, db); err != nil {
		return err
	}
	return migrateNetworkAuditTable(ctx, db)
}

func migrateSessionColumns(ctx context.Context, db *sql.DB) error {
	columns, err := tableColumns(ctx, db, "sessions")
	if err != nil {
		return err
	}

	if _, ok := columns["stop_reason"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN stop_reason TEXT`); err != nil {
			return fmt.Errorf("store: add sessions.stop_reason column: %w", err)
		}
	}
	if _, ok := columns["stop_detail"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN stop_detail TEXT`); err != nil {
			return fmt.Errorf("store: add sessions.stop_detail column: %w", err)
		}
	}
	if _, ok := columns["space"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN space TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("store: add sessions.space column: %w", err)
		}
	}

	return nil
}

func migrateNetworkAuditTable(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "network_audit_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	hasSessionFK, err := tableHasForeignKey(ctx, db, "network_audit_log", "sessions")
	if err != nil {
		return err
	}
	if !hasSessionFK {
		return nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open network audit migration connection: %w", err)
	}
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			_, _ = conn.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
		}
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for network audit migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin network audit migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	statements := []string{
		`CREATE TABLE network_audit_log_new (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			direction  TEXT NOT NULL,
			kind       TEXT NOT NULL,
			space      TEXT NOT NULL,
			peer_from  TEXT NOT NULL,
			peer_to    TEXT,
			message_id TEXT NOT NULL,
			reason     TEXT,
			size       INTEGER NOT NULL,
			timestamp  TEXT NOT NULL
		);`,
		`INSERT INTO network_audit_log_new (
			id, session_id, direction, kind, space, peer_from, peer_to, message_id, reason, size, timestamp
		) SELECT
			id, session_id, direction, kind, space, peer_from, peer_to, message_id, reason, size, timestamp
		FROM network_audit_log`,
		`DROP TABLE network_audit_log`,
		`ALTER TABLE network_audit_log_new RENAME TO network_audit_log`,
		`CREATE INDEX idx_net_audit_ts ON network_audit_log(timestamp)`,
		`CREATE INDEX idx_net_audit_session ON network_audit_log(session_id)`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate network_audit_log table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit network audit migration: %w", err)
	}
	return nil
}

func loadLegacySessions(ctx context.Context, exec sqlQueryExecutor) ([]legacySessionRow, map[string]legacyWorkspaceSeed, error) {
	rows, err := exec.QueryContext(ctx, `SELECT id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at FROM sessions ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, nil, fmt.Errorf("store: query legacy sessions for migration: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]legacySessionRow, 0)
	seeds := make(map[string]legacyWorkspaceSeed)
	for rows.Next() {
		var row legacySessionRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.AgentName,
			&row.Workspace,
			&row.SessionType,
			&row.State,
			&row.ACPSessionID,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("store: scan legacy session for migration: %w", err)
		}

		rootDir := strings.TrimSpace(row.Workspace)
		if rootDir == "" {
			return nil, nil, fmt.Errorf("store: migrate legacy session %q: workspace path is required", row.ID)
		}

		seed, ok := seeds[rootDir]
		if !ok {
			seeds[rootDir] = legacyWorkspaceSeed{rootDir: rootDir, createdAt: row.CreatedAt, updatedAt: row.UpdatedAt}
		} else {
			if strings.TrimSpace(row.CreatedAt) != "" && (seed.createdAt == "" || row.CreatedAt < seed.createdAt) {
				seed.createdAt = row.CreatedAt
			}
			if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt > seed.updatedAt {
				seed.updatedAt = row.UpdatedAt
			}
			seeds[rootDir] = seed
		}

		sessions = append(sessions, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterate legacy sessions for migration: %w", err)
	}

	return sessions, seeds, nil
}

func ensureMigratedWorkspaces(ctx context.Context, tx *sql.Tx, seeds map[string]legacyWorkspaceSeed) (map[string]string, error) {
	rootToID, err := loadWorkspaceIDsByRootDir(ctx, tx)
	if err != nil {
		return nil, err
	}
	takenNames, err := loadWorkspaceNames(ctx, tx)
	if err != nil {
		return nil, err
	}

	if len(seeds) == 0 {
		return rootToID, nil
	}

	orderedRoots := make([]string, 0, len(seeds))
	for rootDir := range seeds {
		orderedRoots = append(orderedRoots, rootDir)
	}
	sort.Strings(orderedRoots)

	for _, rootDir := range orderedRoots {
		if _, ok := rootToID[rootDir]; ok {
			continue
		}

		seed := seeds[rootDir]
		name := aghworkspace.UniqueWorkspaceName(rootDir, takenNames)
		workspaceID := store.NewID("ws")
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at)
			 VALUES (?, ?, '[]', ?, '', ?, ?)`,
			workspaceID,
			rootDir,
			name,
			coalesceTimestamp(seed.createdAt),
			coalesceTimestamp(seed.updatedAt),
		); err != nil {
			return nil, fmt.Errorf("store: insert migrated workspace for %q: %w", rootDir, err)
		}

		rootToID[rootDir] = workspaceID
		takenNames[name] = struct{}{}
	}

	return rootToID, nil
}

func createMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE sessions_new (
			id             TEXT PRIMARY KEY,
			name           TEXT,
			agent_name     TEXT NOT NULL,
			workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
			session_type   TEXT NOT NULL DEFAULT 'user',
			space          TEXT NOT NULL DEFAULT '',
			state          TEXT NOT NULL,
			acp_session_id TEXT,
			stop_reason    TEXT,
			stop_detail    TEXT,
			created_at     TEXT NOT NULL,
			updated_at     TEXT NOT NULL
		);`,
		`CREATE TABLE event_summaries_new (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions_new(id),
			type       TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			summary    TEXT,
			timestamp  TEXT NOT NULL
		);`,
		`CREATE TABLE token_stats_new (
			id            TEXT PRIMARY KEY,
			session_id    TEXT NOT NULL REFERENCES sessions_new(id),
			agent_name    TEXT NOT NULL,
			input_tokens  INTEGER,
			output_tokens INTEGER,
			total_tokens  INTEGER,
			total_cost    REAL,
			cost_currency TEXT,
			turn_count    INTEGER NOT NULL DEFAULT 0,
			updated_at    TEXT NOT NULL
		);`,
		`CREATE TABLE permission_log_new (
			id          TEXT PRIMARY KEY,
			session_id  TEXT NOT NULL REFERENCES sessions_new(id),
			agent_name  TEXT NOT NULL,
			action      TEXT NOT NULL,
			resource    TEXT NOT NULL,
			decision    TEXT NOT NULL,
			policy_used TEXT NOT NULL,
			timestamp   TEXT NOT NULL
		);`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create migrated global table: %w", err)
		}
	}

	return nil
}

func copyMigratedSessions(ctx context.Context, tx *sql.Tx, sessions []legacySessionRow, workspaceIDs map[string]string) error {
	for _, row := range sessions {
		workspaceID, ok := workspaceIDs[strings.TrimSpace(row.Workspace)]
		if !ok {
			return fmt.Errorf("store: missing migrated workspace id for legacy root %q", row.Workspace)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO sessions_new (
				id, name, agent_name, workspace_id, session_type, space, state, acp_session_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.ID,
			nullStringValue(row.Name),
			row.AgentName,
			workspaceID,
			store.NormalizeSessionType(row.SessionType),
			"",
			row.State,
			nullStringValue(row.ACPSessionID),
			row.CreatedAt,
			row.UpdatedAt,
		); err != nil {
			return fmt.Errorf("store: copy migrated session %q: %w", row.ID, err)
		}
	}

	return nil
}

func copyGlobalTableIfExists(ctx context.Context, tx *sql.Tx, source string, target string, insertSQL string) error {
	exists, err := tableExists(ctx, tx, source)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertSQL); err != nil {
		return fmt.Errorf("store: copy %s into %s: %w", source, target, err)
	}
	return nil
}

func swapMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`DROP TABLE IF EXISTS event_summaries`,
		`DROP TABLE IF EXISTS token_stats`,
		`DROP TABLE IF EXISTS permission_log`,
		`DROP TABLE IF EXISTS sessions`,
		`ALTER TABLE sessions_new RENAME TO sessions`,
		`ALTER TABLE event_summaries_new RENAME TO event_summaries`,
		`ALTER TABLE token_stats_new RENAME TO token_stats`,
		`ALTER TABLE permission_log_new RENAME TO permission_log`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: swap migrated global tables: %w", err)
		}
	}

	return nil
}

func loadWorkspaceIDsByRootDir(ctx context.Context, exec sqlQueryExecutor) (map[string]string, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]string{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT id, root_dir FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace ids by root_dir: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rootToID := make(map[string]string)
	for rows.Next() {
		var id string
		var rootDir string
		if err := rows.Scan(&id, &rootDir); err != nil {
			return nil, fmt.Errorf("store: scan workspace id by root_dir: %w", err)
		}
		rootToID[strings.TrimSpace(rootDir)] = strings.TrimSpace(id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace ids by root_dir: %w", err)
	}

	return rootToID, nil
}

func loadWorkspaceNames(ctx context.Context, exec sqlQueryExecutor) (map[string]struct{}, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]struct{}{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT name FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace names: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	names := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: scan workspace name: %w", err)
		}
		names[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace names: %w", err)
	}

	return names, nil
}

func tableExists(ctx context.Context, exec sqlQueryExecutor, table string) (bool, error) {
	var name string
	err := exec.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, strings.TrimSpace(table)).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: check table %q existence: %w", table, err)
	}
	return true, nil
}

func tableColumns(ctx context.Context, exec sqlQueryExecutor, table string) (map[string]struct{}, error) {
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return nil, err
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", name))
	if err != nil {
		return nil, fmt.Errorf("store: query table info for %q: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, fmt.Errorf("store: scan table info for %q: %w", table, err)
		}
		columns[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate table info for %q: %w", table, err)
	}

	return columns, nil
}

func tableHasForeignKey(ctx context.Context, exec sqlQueryExecutor, table string, referencedTable string) (bool, error) {
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return false, err
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", name))
	if err != nil {
		return false, fmt.Errorf("store: query foreign key info for %q: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	target := strings.TrimSpace(referencedTable)
	for rows.Next() {
		var (
			id       int
			seq      int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return false, fmt.Errorf("store: scan foreign key info for %q: %w", table, err)
		}
		if strings.EqualFold(strings.TrimSpace(refTable), target) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("store: iterate foreign key info for %q: %w", table, err)
	}

	return false, nil
}

func coalesceTimestamp(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return store.FormatTimestamp(time.Now().UTC())
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func sessionsDirForDatabasePath(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cleanPath), "sessions")
}

func reconcileLegacySessionMetaWorkspaceIDs(ctx context.Context, exec sqlQueryExecutor, sessionsDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("store: reconcile session metadata workspace ids canceled: %w", err)
	}

	rootToID, err := loadWorkspaceIDsByRootDir(ctx, exec)
	if err != nil {
		return err
	}
	if len(rootToID) == 0 {
		return nil
	}

	cleanDir := strings.TrimSpace(sessionsDir)
	if cleanDir == "" {
		return nil
	}

	entries, err := os.ReadDir(cleanDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("store: read sessions directory %q for workspace id reconciliation: %w", cleanDir, err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("store: reconcile session metadata workspace ids canceled: %w", err)
		}
		if !entry.IsDir() {
			continue
		}

		metaPath := store.SessionMetaFile(filepath.Join(cleanDir, entry.Name()))
		needsRewrite, meta, err := loadReconciledLegacySessionMeta(metaPath, rootToID)
		if err != nil {
			return err
		}
		if !needsRewrite {
			continue
		}
		if err := store.WriteSessionMeta(metaPath, meta); err != nil {
			return fmt.Errorf("store: rewrite legacy session meta %q: %w", metaPath, err)
		}
	}

	return nil
}

func loadReconciledLegacySessionMeta(path string, rootToID map[string]string) (bool, store.SessionMeta, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return false, store.SessionMeta{}, nil
	default:
		return false, store.SessionMeta{}, fmt.Errorf("store: read session meta %q for workspace id reconciliation: %w", path, err)
	}

	var raw legacySessionMetaCompat
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, store.SessionMeta{}, nil
	}

	if strings.TrimSpace(raw.WorkspaceID) != "" {
		return false, store.SessionMeta{}, nil
	}

	workspaceRoot := strings.TrimSpace(raw.Workspace)
	if workspaceRoot == "" {
		return false, store.SessionMeta{}, nil
	}

	workspaceID, ok := rootToID[workspaceRoot]
	if !ok {
		return false, store.SessionMeta{}, nil
	}

	meta := store.SessionMeta{
		ID:           raw.ID,
		Name:         raw.Name,
		AgentName:    raw.AgentName,
		WorkspaceID:  workspaceID,
		SessionType:  raw.SessionType,
		State:        raw.State,
		ACPSessionID: raw.ACPSessionID,
		CreatedAt:    raw.CreatedAt,
		UpdatedAt:    raw.UpdatedAt,
	}
	if err := meta.Validate(); err != nil {
		return false, store.SessionMeta{}, nil
	}

	return true, meta, nil
}
