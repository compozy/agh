package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pedronauck/agh/internal/diagnostics"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	storepkg "github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

const (
	catalogEFilenamePath    = "  e.filename,"
	catalogENamePath        = "  e.name,"
	catalogEScopePath       = "  e.scope,"
	catalogETypePath        = "  e.type,"
	catalogEWorkspaceIDPath = "  e.workspace_id,"
	catalogSelectValue      = "SELECT"
	catalogScopeKey         = "scope"
	catalogUpdatedAtKey     = "updated_at"
	catalogWorkspaceIDKey   = "workspace_id"
)

const (
	defaultSearchLimit         = 10
	maxSearchLimit             = 50
	defaultHistoryLimit        = 25
	maxHistoryLimit            = 100
	catalogMigrationsTable     = "memory_schema_migrations"
	maxOperationSummaryBytes   = 2048
	catalogStateKeyLastReindex = "last_reindex_at"
	catalogStateKeyScopePrefix = "scope_synced::"
	catalogEventAgentName      = "daemon"
)

const (
	memoryEventWriteCommitted         = "memory.write.committed"
	memoryEventWriteRejected          = "memory.write.rejected"
	memoryEventWriteShadowed          = "memory.write.shadowed"
	memoryEventWriteReindex           = "memory.write.reindex"
	memoryEventWriteReverted          = "memory.write.reverted"
	memoryEventRecallExecuted         = "memory.recall.executed"
	memoryEventRecallSkipped          = "memory.recall.skipped"
	memoryEventRecallSignalDropped    = "memory.recall.signal_dropped"
	memoryEventRecallSignalFailed     = "memory.recall.signal_update_failed"
	memoryEventDecisionsSummarized    = "memory.decisions.audit_summarized"
	memoryEventDecisionsPruned        = "memory.decisions.pruned"
	memoryEventDreamStarted           = "memory.dream.run.started"
	memoryEventDreamPromoted          = "memory.dream.run.promoted"
	memoryEventDreamFailed            = "memory.dream.run.failed"
	memoryEventExtractorStarted       = "memory.extractor.started"
	memoryEventExtractorCompleted     = "memory.extractor.completed"
	memoryEventExtractorFailed        = "memory.extractor.failed"
	memoryEventExtractorCoalesced     = "memory.extractor.coalesced"
	memoryEventExtractorDropped       = "memory.extractor.dropped"
	memoryEventDailyRotated           = "memory.daily.rotated"
	memoryEventDailyArchived          = "memory.daily.archived"
	memoryEventDailyRestored          = "memory.daily.restored"
	memoryEventDailyPurged            = "memory.daily.purged"
	memoryEventDailyArchivePurged     = "memory.daily.archive_purged"
	memoryEventProviderEnabled        = "memory.provider.enabled"
	memoryEventProviderDisabled       = "memory.provider.disabled"
	memoryEventProviderCollision      = "memory.provider.collision"
	memoryEventWorkspaceRelocated     = "memory.workspace.relocated"
	memoryEventWorkspaceRecovered     = "memory.workspace.recovered"
	memoryEventAgentPurged            = "memory.agent.purged"
	memoryEventMigrationApplied       = "memory.migration.applied"
	memoryEventMetadataActionKey      = "action"
	memoryEventMetadataFilenameKey    = "filename"
	memoryEventMetadataLegacyIDKey    = "legacy_id"
	memoryEventMetadataSummaryKey     = "summary"
	memoryEventMetadataQueryKey       = "query"
	memoryEventMetadataResultCountKey = "result_count"
)

var catalogSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_catalog_entries (
		id             TEXT PRIMARY KEY,
		scope          TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		workspace_id   TEXT NOT NULL DEFAULT '',
		workspace_root TEXT NOT NULL DEFAULT '',
		filename       TEXT NOT NULL,
		type           TEXT NOT NULL,
		name           TEXT NOT NULL,
		description    TEXT NOT NULL DEFAULT '',
		content        TEXT NOT NULL,
		content_hash   TEXT NOT NULL,
		updated_at     TEXT NOT NULL,
		UNIQUE (scope, workspace_root, filename)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_scope ON memory_catalog_entries(scope);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_workspace_root ON memory_catalog_entries(workspace_root);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_updated_at ON memory_catalog_entries(updated_at);`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS memory_catalog_fts USING fts5(
		name,
		description,
		content,
		content='memory_catalog_entries',
		content_rowid='rowid',
		tokenize='porter unicode61'
	);`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_ai AFTER INSERT ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(rowid, name, description, content)
		VALUES (new.rowid, new.name, new.description, new.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_ad AFTER DELETE ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(memory_catalog_fts, rowid, name, description, content)
		VALUES ('delete', old.rowid, old.name, old.description, old.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_au AFTER UPDATE ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(memory_catalog_fts, rowid, name, description, content)
		VALUES ('delete', old.rowid, old.name, old.description, old.content);
		INSERT INTO memory_catalog_fts(rowid, name, description, content)
		VALUES (new.rowid, new.name, new.description, new.content);
	END;`,
	`CREATE TABLE IF NOT EXISTS memory_catalog_state (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS memory_operation_log (
		id         TEXT PRIMARY KEY,
		type       TEXT NOT NULL,
		agent_name TEXT NOT NULL DEFAULT 'daemon',
		summary    TEXT NOT NULL DEFAULT '',
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_type ON memory_operation_log(type);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_timestamp ON memory_operation_log(timestamp);`,
}

var memoryEventOps = []string{
	memoryEventWriteCommitted,
	memoryEventWriteRejected,
	memoryEventWriteShadowed,
	memoryEventWriteReindex,
	memoryEventWriteReverted,
	memoryEventRecallExecuted,
	memoryEventRecallSkipped,
	memoryEventRecallSignalDropped,
	memoryEventRecallSignalFailed,
	memoryEventDecisionsSummarized,
	memoryEventDecisionsPruned,
	memoryEventDreamStarted,
	memoryEventDreamPromoted,
	memoryEventDreamFailed,
	memoryEventExtractorStarted,
	memoryEventExtractorCompleted,
	memoryEventExtractorFailed,
	memoryEventExtractorCoalesced,
	memoryEventExtractorDropped,
	memoryEventDailyRotated,
	memoryEventDailyArchived,
	memoryEventDailyRestored,
	memoryEventDailyPurged,
	memoryEventDailyArchivePurged,
	memoryEventProviderEnabled,
	memoryEventProviderDisabled,
	memoryEventProviderCollision,
	memoryEventWorkspaceRelocated,
	memoryEventWorkspaceRecovered,
	memoryEventAgentPurged,
	memoryEventMigrationApplied,
}

var memoryV2CatalogStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_catalog_entries (
		id           TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL DEFAULT '',
		scope        TEXT NOT NULL CHECK (scope IN ('global', 'workspace', 'agent')),
		agent_name   TEXT NOT NULL DEFAULT '',
		agent_tier   TEXT NOT NULL DEFAULT '' CHECK (agent_tier IN ('', 'workspace', 'global')),
		type         TEXT NOT NULL CHECK (type IN ('user', 'feedback', 'project', 'reference')),
		slug         TEXT NOT NULL,
		filename     TEXT NOT NULL,
		name         TEXT NOT NULL DEFAULT '',
		description  TEXT NOT NULL DEFAULT '',
		content      TEXT NOT NULL DEFAULT '',
		content_hash TEXT NOT NULL,
		injection    INTEGER NOT NULL DEFAULT 1,
		mtime_ms     INTEGER NOT NULL,
		indexed_at   INTEGER NOT NULL,
		updated_at   TEXT NOT NULL
	);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_memory_catalog_scope_slug
		ON memory_catalog_entries(workspace_id, scope, agent_name, agent_tier, type, slug);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_scope
		ON memory_catalog_entries(scope, agent_name, agent_tier, type);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_workspace
		ON memory_catalog_entries(workspace_id);`,
	`CREATE INDEX IF NOT EXISTS idx_memory_catalog_updated_at
		ON memory_catalog_entries(updated_at);`,
}

var memoryCatalogFTSStatements = []string{
	`CREATE VIRTUAL TABLE IF NOT EXISTS memory_catalog_fts USING fts5(
		name,
		description,
		content,
		content='memory_catalog_entries',
		content_rowid='rowid',
		tokenize='porter unicode61'
	);`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_ai AFTER INSERT ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(rowid, name, description, content)
		VALUES (new.rowid, new.name, new.description, new.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_ad AFTER DELETE ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(memory_catalog_fts, rowid, name, description, content)
		VALUES ('delete', old.rowid, old.name, old.description, old.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_catalog_entries_au AFTER UPDATE ON memory_catalog_entries BEGIN
		INSERT INTO memory_catalog_fts(memory_catalog_fts, rowid, name, description, content)
		VALUES ('delete', old.rowid, old.name, old.description, old.content);
		INSERT INTO memory_catalog_fts(rowid, name, description, content)
		VALUES (new.rowid, new.name, new.description, new.content);
	END;`,
}

var memoryV2ChunkStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_chunks (
		id           TEXT PRIMARY KEY,
		file_id      TEXT NOT NULL REFERENCES memory_catalog_entries(id) ON DELETE CASCADE,
		content      TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		start_line   INTEGER NOT NULL,
		end_line     INTEGER NOT NULL,
		indexed_at   INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_chunks_file ON memory_chunks(file_id);`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS memory_chunks_fts USING fts5(
		content,
		content='memory_chunks',
		content_rowid='rowid',
		tokenize='unicode61'
	);`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS memory_chunks_fts_trigram USING fts5(
		content,
		content='memory_chunks',
		content_rowid='rowid',
		tokenize='trigram'
	);`,
	`CREATE TRIGGER IF NOT EXISTS memory_chunks_ai AFTER INSERT ON memory_chunks BEGIN
		INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.rowid, new.content);
		INSERT INTO memory_chunks_fts_trigram(rowid, content) VALUES (new.rowid, new.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_chunks_ad AFTER DELETE ON memory_chunks BEGIN
		INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
		VALUES ('delete', old.rowid, old.content);
		INSERT INTO memory_chunks_fts_trigram(memory_chunks_fts_trigram, rowid, content)
		VALUES ('delete', old.rowid, old.content);
	END;`,
	`CREATE TRIGGER IF NOT EXISTS memory_chunks_au AFTER UPDATE ON memory_chunks BEGIN
		INSERT INTO memory_chunks_fts(memory_chunks_fts, rowid, content)
		VALUES ('delete', old.rowid, old.content);
		INSERT INTO memory_chunks_fts_trigram(memory_chunks_fts_trigram, rowid, content)
		VALUES ('delete', old.rowid, old.content);
		INSERT INTO memory_chunks_fts(rowid, content) VALUES (new.rowid, new.content);
		INSERT INTO memory_chunks_fts_trigram(rowid, content) VALUES (new.rowid, new.content);
	END;`,
}

var memoryV2EventStatements = []string{
	fmt.Sprintf(`CREATE TABLE IF NOT EXISTS memory_events (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		op           TEXT NOT NULL CHECK (op IN (%s)),
		scope        TEXT CHECK (scope IN ('global', 'workspace', 'agent')),
		agent_name   TEXT,
		agent_tier   TEXT CHECK (agent_tier IS NULL OR agent_tier IN ('workspace', 'global')),
		workspace_id TEXT,
		session_id   TEXT,
		actor_kind   TEXT NOT NULL,
		decision_id  TEXT,
		target_id    TEXT,
		metadata     TEXT NOT NULL DEFAULT '{}',
		ts_ms        INTEGER NOT NULL
	);`, quotedSQLStrings(memoryEventOps)),
	`CREATE INDEX IF NOT EXISTS idx_events_workspace ON memory_events(workspace_id, ts_ms);`,
	`CREATE INDEX IF NOT EXISTS idx_events_op ON memory_events(op, ts_ms);`,
	`CREATE INDEX IF NOT EXISTS idx_events_session ON memory_events(session_id, ts_ms);`,
}

var memoryV2DecisionStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_decisions (
		id                TEXT PRIMARY KEY,
		candidate_hash    TEXT NOT NULL,
		idempotency_key   TEXT NOT NULL UNIQUE,
		frontmatter_hash  TEXT NOT NULL,
		workspace_id      TEXT,
		scope             TEXT NOT NULL CHECK (scope IN ('global', 'workspace', 'agent')),
		agent_name        TEXT,
		agent_tier        TEXT CHECK (agent_tier IS NULL OR agent_tier IN ('workspace', 'global')),
		op                TEXT NOT NULL CHECK (op IN ('noop', 'add', 'update', 'delete', 'reject')),
		targets           TEXT NOT NULL DEFAULT '[]',
		target_filename   TEXT NOT NULL,
		frontmatter       TEXT NOT NULL DEFAULT '{}',
		post_content      TEXT,
		post_content_hash TEXT,
		prior_content     TEXT,
		confidence        REAL NOT NULL,
		source            TEXT NOT NULL CHECK (source IN ('rule', 'llm')),
		rule_trace        TEXT NOT NULL,
		llm_trace         TEXT,
		reason            TEXT,
		prompt_version    TEXT,
		applied_at        INTEGER,
		decided_at        INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_decisions_workspace ON memory_decisions(workspace_id, decided_at);`,
	`CREATE INDEX IF NOT EXISTS idx_decisions_op ON memory_decisions(op, decided_at);`,
	`CREATE INDEX IF NOT EXISTS idx_decisions_unapplied
		ON memory_decisions(applied_at) WHERE applied_at IS NULL;`,
}

var memoryV2RecallSignalStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_recall_signals (
		chunk_id              TEXT PRIMARY KEY REFERENCES memory_chunks(id) ON DELETE CASCADE,
		workspace_id          TEXT,
		recall_count          INTEGER NOT NULL DEFAULT 0,
		last_recalled_at      INTEGER,
		recall_score          REAL NOT NULL DEFAULT 0,
		freshness_started_at  INTEGER NOT NULL DEFAULT 0,
		promoted_at           INTEGER,
		promotion_run_id      TEXT,
		last_score_update_at  INTEGER NOT NULL DEFAULT 0,
		session_count         INTEGER NOT NULL DEFAULT 0,
		last_session_id       TEXT,
		already_surfaced_json TEXT NOT NULL DEFAULT '[]',
		updated_at            INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_recall_signals_workspace
		ON memory_recall_signals(workspace_id, updated_at);`,
	`CREATE INDEX IF NOT EXISTS idx_recall_signals_last_recalled
		ON memory_recall_signals(last_recalled_at);`,
	`CREATE INDEX IF NOT EXISTS idx_signals_unpromoted
		ON memory_recall_signals(promoted_at, recall_score) WHERE promoted_at IS NULL;`,
	`CREATE INDEX IF NOT EXISTS idx_signals_recent
		ON memory_recall_signals(last_recalled_at);`,
}

var memoryV2ConsolidationStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_consolidations (
		id             TEXT PRIMARY KEY,
		workspace_id   TEXT,
		scope          TEXT NOT NULL CHECK (scope IN ('global', 'workspace', 'agent')),
		agent_name     TEXT,
		agent_tier     TEXT CHECK (agent_tier IS NULL OR agent_tier IN ('workspace', 'global')),
		started_at     INTEGER NOT NULL,
		finished_at    INTEGER,
		status         TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'canceled')),
		input_count    INTEGER NOT NULL DEFAULT 0,
		promoted_count INTEGER NOT NULL DEFAULT 0,
		error          TEXT NOT NULL DEFAULT '',
		metadata       TEXT NOT NULL DEFAULT '{}'
	);`,
	`CREATE INDEX IF NOT EXISTS idx_consolidations_workspace
		ON memory_consolidations(workspace_id, started_at);`,
	`CREATE INDEX IF NOT EXISTS idx_consolidations_status
		ON memory_consolidations(status, started_at);`,
}

var catalogSchemaMigrations = []storepkg.Migration{
	{
		Version:    1,
		Name:       "initial_memory_catalog_schema",
		Statements: catalogSchemaStatements,
	},
	{
		Version:  2,
		Name:     "add_memory_operation_scope",
		Checksum: "catalog-add-memory-operation-scope-v1",
		Up:       migrateCatalogOperationScope,
	},
	{
		Version:  3,
		Name:     "memv2_catalog_workspace_identity",
		Checksum: "2026-05-05-memv2-catalog-workspace-identity",
		Up:       migrateCatalogWorkspaceIdentity,
	},
	{
		Version:    4,
		Name:       "memv2_chunks_and_fts",
		Statements: memoryV2ChunkStatements,
	},
	{
		Version:    5,
		Name:       "memv2_decisions",
		Statements: memoryV2DecisionStatements,
	},
	{
		Version:    6,
		Name:       "memv2_recall_signals",
		Statements: memoryV2RecallSignalStatements,
	},
	{
		Version:    7,
		Name:       "memv2_consolidations",
		Statements: memoryV2ConsolidationStatements,
	},
	{
		Version:  8,
		Name:     "memv2_recall_signals_live_flow",
		Checksum: "2026-05-05-memv2-recall-signals-live-flow",
		Up:       migrateRecallSignalsLiveFlow,
	},
}

type catalog struct {
	path    string
	now     func() time.Time
	writeMu sync.Mutex

	mu sync.Mutex
	db *sql.DB
}

type catalogDocument struct {
	ID            string
	Scope         memcontract.Scope
	WorkspaceID   string
	WorkspaceRoot string
	AgentName     string
	AgentTier     memcontract.AgentTier
	Filename      string
	Type          memcontract.Type
	Name          string
	Description   string
	Content       string
	ContentHash   string
	Injection     bool
	UpdatedAt     time.Time
}

type catalogChunk struct {
	id          string
	content     string
	contentHash string
	startLine   int
	endLine     int
}

type catalogWriteExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func newCatalog(path string, now func() time.Time) *catalog {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil
	}
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &catalog{path: cleanPath, now: now}
}

func (c *catalog) enabled() bool {
	return c != nil && strings.TrimSpace(c.path) != ""
}

func (c *catalog) ensureDB(ctx context.Context) (*sql.DB, error) {
	if !c.enabled() {
		return nil, nil
	}
	if ctx == nil {
		return nil, errors.New("memory: catalog context is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return c.db, nil
	}

	db, err := storepkg.OpenSQLiteDatabase(ctx, c.path, func(ctx context.Context, db *sql.DB) error {
		return storepkg.RunMigrations(
			ctx,
			db,
			catalogSchemaMigrations,
			storepkg.WithMigrationsTable(catalogMigrationsTable),
		)
	})
	if err != nil {
		return nil, fmt.Errorf("memory: open catalog database %q: %w", c.path, err)
	}
	c.db = db
	return c.db, nil
}

func migrateCatalogOperationScope(ctx context.Context, tx *sql.Tx) error {
	exists, err := catalogTableExists(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := catalogOperationLogColumns(ctx, tx)
	if err != nil {
		return err
	}
	specs := []struct {
		name string
		sql  string
	}{
		{name: catalogScopeKey, sql: `ALTER TABLE memory_operation_log ADD COLUMN scope TEXT NOT NULL DEFAULT ''`},
		{
			name: "workspace_root",
			sql:  `ALTER TABLE memory_operation_log ADD COLUMN workspace_root TEXT NOT NULL DEFAULT ''`,
		},
		{name: "filename", sql: `ALTER TABLE memory_operation_log ADD COLUMN filename TEXT NOT NULL DEFAULT ''`},
	}
	for _, spec := range specs {
		if _, ok := columns[spec.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("memory: add memory_operation_log.%s column: %w", spec.name, err)
		}
	}
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_scope ON memory_operation_log(scope);`,
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_workspace_root ON memory_operation_log(workspace_root);`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("memory: create memory operation scope index: %w", err)
		}
	}
	return nil
}

func catalogOperationLogColumns(ctx context.Context, tx *sql.Tx) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `PRAGMA table_info(memory_operation_log)`)
	if err != nil {
		return nil, fmt.Errorf("memory: inspect memory_operation_log schema: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid          int
			name         string
			dataType     string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("memory: scan memory_operation_log column: %w", err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate memory_operation_log columns: %w", err)
	}
	return columns, nil
}

func migrateCatalogWorkspaceIdentity(ctx context.Context, tx *sql.Tx) error {
	if err := rebuildCatalogEntriesWithWorkspaceID(ctx, tx); err != nil {
		return err
	}
	if err := migrateOperationLogToEvents(ctx, tx); err != nil {
		return err
	}
	return nil
}

// Migration 008: live recall signals and chunk backfill.
//
// Why: Task 06 / requires recall to update memory_recall_signals live
// and later dreaming gates need recall_score, promotion, and barrier columns.
// Affects: memory catalog tables memory_recall_signals and memory_chunks.
// Idempotent: yes; columns and indexes are guarded, chunks use INSERT OR IGNORE.
// Reversible: no; derived chunks and signal columns are regenerated state.
func migrateRecallSignalsLiveFlow(ctx context.Context, tx *sql.Tx) error {
	if err := execCatalogStatements(ctx, tx, memoryV2ChunkStatements); err != nil {
		return err
	}
	if err := ensureRecallSignalsLiveSchema(ctx, tx); err != nil {
		return err
	}
	if err := backfillMemoryChunks(ctx, tx); err != nil {
		return err
	}
	return nil
}

func ensureRecallSignalsLiveSchema(ctx context.Context, tx *sql.Tx) error {
	exists, err := catalogTableExists(ctx, tx, "memory_recall_signals")
	if err != nil {
		return err
	}
	if !exists {
		return execCatalogStatements(ctx, tx, memoryV2RecallSignalStatements)
	}
	columns, err := catalogTableColumns(ctx, tx, "memory_recall_signals")
	if err != nil {
		return err
	}
	additions := []struct {
		name string
		sql  string
	}{
		{name: catalogWorkspaceIDKey, sql: `ALTER TABLE memory_recall_signals ADD COLUMN workspace_id TEXT`},
		{
			name: "recall_count",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN recall_count INTEGER NOT NULL DEFAULT 0`,
		},
		{name: "last_recalled_at", sql: `ALTER TABLE memory_recall_signals ADD COLUMN last_recalled_at INTEGER`},
		{
			name: "recall_score",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN recall_score REAL NOT NULL DEFAULT 0`,
		},
		{
			name: "freshness_started_at",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN freshness_started_at INTEGER NOT NULL DEFAULT 0`,
		},
		{name: "promoted_at", sql: `ALTER TABLE memory_recall_signals ADD COLUMN promoted_at INTEGER`},
		{name: "promotion_run_id", sql: `ALTER TABLE memory_recall_signals ADD COLUMN promotion_run_id TEXT`},
		{
			name: "last_score_update_at",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN last_score_update_at INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "session_count",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN session_count INTEGER NOT NULL DEFAULT 0`,
		},
		{name: "last_session_id", sql: `ALTER TABLE memory_recall_signals ADD COLUMN last_session_id TEXT`},
		{
			name: "already_surfaced_json",
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN already_surfaced_json TEXT NOT NULL DEFAULT '[]'`,
		},
		{
			name: catalogUpdatedAtKey,
			sql:  `ALTER TABLE memory_recall_signals ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0`,
		},
	}
	for _, addition := range additions {
		if _, ok := columns[addition.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, addition.sql); err != nil {
			return fmt.Errorf("memory: add memory_recall_signals.%s column: %w", addition.name, err)
		}
	}
	for _, stmt := range memoryV2RecallSignalStatements[1:] {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("memory: create recall signal index: %w", err)
		}
	}
	return nil
}

func backfillMemoryChunks(ctx context.Context, tx *sql.Tx) error {
	for _, table := range []string{"memory_catalog_entries", "memory_chunks"} {
		exists, err := catalogTableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO memory_chunks (
			id, file_id, content, content_hash, start_line, end_line, indexed_at
		)
		SELECT
			e.id || '::chunk:0001',
			e.id,
			trim(e.name || char(10) || e.description || char(10) || e.content),
			e.content_hash,
			1,
			CASE
				WHEN e.content = '' THEN 1
				ELSE length(e.content) - length(replace(e.content, char(10), '')) + 1
			END,
			e.mtime_ms
		FROM memory_catalog_entries e
		WHERE NOT EXISTS (SELECT 1 FROM memory_chunks c WHERE c.file_id = e.id)`,
	); err != nil {
		return fmt.Errorf("memory: backfill memory chunks: %w", err)
	}
	return nil
}

func rebuildCatalogEntriesWithWorkspaceID(ctx context.Context, tx *sql.Tx) error {
	exists, err := catalogTableExists(ctx, tx, "memory_catalog_entries")
	if err != nil {
		return err
	}
	if !exists {
		return execCatalogStatements(ctx, tx, append(memoryV2CatalogStatements, memoryCatalogFTSStatements...))
	}

	columns, err := catalogTableColumns(ctx, tx, "memory_catalog_entries")
	if err != nil {
		return err
	}
	if _, hasWorkspaceRoot := columns["workspace_root"]; !hasWorkspaceRoot {
		if err := execCatalogStatements(
			ctx,
			tx,
			append(memoryV2CatalogStatements, memoryCatalogFTSStatements...),
		); err != nil {
			return err
		}
		return nil
	}

	if err := dropCatalogFTS(ctx, tx); err != nil {
		return err
	}
	if err := createCatalogEntriesNew(ctx, tx); err != nil {
		return err
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT id, scope, workspace_id, workspace_root, filename, type, name,
			description, content, content_hash, updated_at
		 FROM memory_catalog_entries
		 ORDER BY rowid ASC`,
	)
	if err != nil {
		return fmt.Errorf("memory: read legacy catalog entries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		doc, scanErr := scanLegacyCatalogEntry(rows)
		if scanErr != nil {
			return scanErr
		}
		workspaceID, identityErr := workspaceIDForLegacyRoot(ctx, doc.Scope, doc.WorkspaceID, doc.WorkspaceRoot)
		if identityErr != nil {
			return identityErr
		}
		doc.WorkspaceID = workspaceID
		doc.WorkspaceRoot = ""
		doc.ID = catalogDocID(doc.Scope, doc.WorkspaceID, doc.Filename)
		if err := insertCatalogDocumentIntoTx(ctx, tx, "memory_catalog_entries_new", doc); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("memory: iterate legacy catalog entries: %w", err)
	}

	for _, stmt := range []string{
		`DROP TABLE memory_catalog_entries`,
		`ALTER TABLE memory_catalog_entries_new RENAME TO memory_catalog_entries`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("memory: rebuild catalog entries: %w", err)
		}
	}
	return execCatalogStatements(ctx, tx, append(memoryV2CatalogStatements[1:], memoryCatalogFTSStatements...))
}

func migrateOperationLogToEvents(ctx context.Context, tx *sql.Tx) error {
	if err := execCatalogStatements(ctx, tx, memoryV2EventStatements); err != nil {
		return err
	}
	exists, err := catalogTableExists(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := catalogTableColumns(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	selectSQL := legacyOperationLogSelectSQL(columns)
	rows, err := tx.QueryContext(ctx, selectSQL)
	if err != nil {
		return fmt.Errorf("memory: read legacy operation log: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		record, scanErr := scanLegacyOperationLog(rows)
		if scanErr != nil {
			return scanErr
		}
		workspaceID, identityErr := workspaceIDForLegacyRoot(
			ctx,
			record.Scope,
			record.Workspace,
			record.Workspace,
		)
		if identityErr != nil {
			return identityErr
		}
		record.Workspace = workspaceID
		if err := insertMemoryEventTx(ctx, tx, record, true); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("memory: iterate legacy operation log: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE memory_operation_log`); err != nil {
		return fmt.Errorf("memory: drop legacy memory_operation_log: %w", err)
	}
	return nil
}

func createCatalogEntriesNew(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, strings.Replace(
		memoryV2CatalogStatements[0],
		"memory_catalog_entries",
		"memory_catalog_entries_new",
		1,
	)); err != nil {
		return fmt.Errorf("memory: create rebuilt catalog entries table: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_memory_catalog_new_scope_slug
		 ON memory_catalog_entries_new(workspace_id, scope, agent_name, agent_tier, type, slug);`,
	); err != nil {
		return fmt.Errorf("memory: create rebuilt catalog unique index: %w", err)
	}
	return nil
}

func dropCatalogFTS(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`DROP TRIGGER IF EXISTS memory_catalog_entries_ai`,
		`DROP TRIGGER IF EXISTS memory_catalog_entries_ad`,
		`DROP TRIGGER IF EXISTS memory_catalog_entries_au`,
		`DROP TABLE IF EXISTS memory_catalog_fts`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("memory: drop legacy catalog fts: %w", err)
		}
	}
	return nil
}

func scanLegacyCatalogEntry(scanner interface{ Scan(dest ...any) error }) (catalogDocument, error) {
	var (
		doc        catalogDocument
		scopeRaw   string
		typeRaw    string
		updatedRaw string
	)
	if err := scanner.Scan(
		&doc.ID,
		&scopeRaw,
		&doc.WorkspaceID,
		&doc.WorkspaceRoot,
		&doc.Filename,
		&typeRaw,
		&doc.Name,
		&doc.Description,
		&doc.Content,
		&doc.ContentHash,
		&updatedRaw,
	); err != nil {
		return catalogDocument{}, fmt.Errorf("memory: scan legacy catalog entry: %w", err)
	}
	updatedAt, err := storepkg.ParseTimestamp(updatedRaw)
	if err != nil {
		return catalogDocument{}, fmt.Errorf("memory: parse legacy catalog updated_at %q: %w", updatedRaw, err)
	}
	doc.Scope = memcontract.Scope(scopeRaw).Normalize()
	doc.Type = memcontract.Type(typeRaw).Normalize()
	doc.Injection = true
	doc.UpdatedAt = updatedAt.UTC()
	return doc, nil
}

func legacyOperationLogSelectSQL(columns map[string]struct{}) string {
	selectColumn := func(name string, fallback string) string {
		if _, ok := columns[name]; ok {
			return name
		}
		return fallback + " AS " + name
	}
	return fmt.Sprintf(
		`SELECT id, type, %s, %s, %s, agent_name, summary, timestamp FROM memory_operation_log ORDER BY timestamp ASC, id ASC`,
		selectColumn(catalogScopeKey, "''"),
		selectColumn("workspace_root", "''"),
		selectColumn("filename", "''"),
	)
}

func scanLegacyOperationLog(scanner interface{ Scan(dest ...any) error }) (memcontract.OperationRecord, error) {
	var (
		record       memcontract.OperationRecord
		operationRaw string
		scopeRaw     string
		timestampRaw string
	)
	if err := scanner.Scan(
		&record.ID,
		&operationRaw,
		&scopeRaw,
		&record.Workspace,
		&record.Filename,
		&record.AgentName,
		&record.Summary,
		&timestampRaw,
	); err != nil {
		return memcontract.OperationRecord{}, fmt.Errorf("memory: scan legacy operation log: %w", err)
	}
	timestamp, err := storepkg.ParseTimestamp(timestampRaw)
	if err != nil {
		return memcontract.OperationRecord{}, fmt.Errorf(
			"memory: parse legacy operation timestamp %q: %w",
			timestampRaw,
			err,
		)
	}
	record.Operation = memcontract.Operation(operationRaw).Normalize()
	record.Scope = memcontract.Scope(scopeRaw).Normalize()
	record.Timestamp = timestamp.UTC()
	return record, nil
}

func workspaceIDForLegacyRoot(
	ctx context.Context,
	scope memcontract.Scope,
	existingWorkspaceID string,
	workspaceRoot string,
) (string, error) {
	normalizedScope := scope.Normalize()
	if normalizedScope != memcontract.ScopeWorkspace {
		return strings.TrimSpace(existingWorkspaceID), nil
	}
	if aghworkspace.IsWorkspaceID(existingWorkspaceID) {
		return strings.TrimSpace(existingWorkspaceID), nil
	}
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", errors.New("memory: legacy workspace row missing workspace root for workspace_id backfill")
	}
	identity, err := aghworkspace.EnsureIdentity(ctx, root)
	if err != nil {
		return "", fmt.Errorf("memory: resolve workspace identity for %q: %w", root, err)
	}
	return identity.WorkspaceID, nil
}

func execCatalogStatements(ctx context.Context, tx *sql.Tx, statements []string) error {
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("memory: apply catalog schema statement: %w", err)
		}
	}
	return nil
}

func catalogTableExists(ctx context.Context, tx *sql.Tx, table string) (bool, error) {
	var name string
	err := tx.QueryRowContext(
		ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		strings.TrimSpace(table),
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("memory: check table %q existence: %w", table, err)
	}
	return true, nil
}

func catalogTableColumns(ctx context.Context, tx *sql.Tx, table string) (map[string]struct{}, error) {
	name, err := storepkg.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, name))
	if err != nil {
		return nil, fmt.Errorf("memory: inspect %s schema: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()
	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid          int
			name         string
			dataType     string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, fmt.Errorf("memory: scan %s column: %w", table, err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate %s columns: %w", table, err)
	}
	return columns, nil
}

func (c *catalog) replaceScope(
	ctx context.Context,
	scope memcontract.Scope,
	workspaceID string,
	agentName string,
	agentTier memcontract.AgentTier,
	docs []catalogDocument,
) (err error) {
	return c.withCatalogWriteTx(ctx, "catalog scope replace", func(tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM memory_catalog_entries
			 WHERE scope = ? AND workspace_id = ? AND agent_name = ? AND agent_tier = ?`,
			string(scope.Normalize()),
			strings.TrimSpace(workspaceID),
			strings.TrimSpace(agentName),
			string(agentTier.Normalize()),
		); err != nil {
			return fmt.Errorf("memory: clear catalog scope %q/%q: %w", scope, workspaceID, err)
		}
		for _, doc := range docs {
			if err := insertCatalogDocumentTx(ctx, tx, doc); err != nil {
				return err
			}
			if err := replaceCatalogChunksTx(ctx, tx, doc); err != nil {
				return err
			}
		}
		return c.upsertCatalogScopeStateTx(ctx, tx, scope, workspaceID)
	})
}

func (c *catalog) upsertDocument(ctx context.Context, doc catalogDocument) (err error) {
	return c.withCatalogWriteTx(ctx, "catalog document upsert", func(tx *storepkg.WriteTx) error {
		if err := upsertCatalogDocumentTx(ctx, tx, doc); err != nil {
			return err
		}
		if err := replaceCatalogChunksTx(ctx, tx, doc); err != nil {
			return err
		}
		if err := c.upsertCatalogScopeStateTx(ctx, tx, doc.Scope, doc.WorkspaceID); err != nil {
			return err
		}
		if err := upsertCatalogStateTx(
			ctx,
			tx,
			catalogStateKeyLastReindex,
			storepkg.FormatTimestamp(c.now().UTC()),
		); err != nil {
			return fmt.Errorf("memory: persist catalog reindex timestamp: %w", err)
		}
		return nil
	})
}

func (c *catalog) withCatalogWriteTx(
	ctx context.Context,
	operation string,
	fn func(*storepkg.WriteTx) error,
) (err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}

	if err := storepkg.ExecuteWrite(ctx, db, func(writeCtx context.Context, tx *storepkg.WriteTx) error {
		if err := writeCtx.Err(); err != nil {
			return err
		}
		if err := fn(tx); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("memory: %s: %w", operation, err)
	}
	return nil
}

func insertCatalogDocumentTx(ctx context.Context, tx catalogWriteExecutor, doc catalogDocument) error {
	return insertCatalogDocumentNewTx(ctx, tx, doc)
}

func insertCatalogDocumentNewTx(ctx context.Context, tx catalogWriteExecutor, doc catalogDocument) error {
	return insertCatalogDocumentIntoTx(ctx, tx, "memory_catalog_entries", doc)
}

func insertCatalogDocumentIntoTx(
	ctx context.Context,
	tx catalogWriteExecutor,
	table string,
	doc catalogDocument,
) error {
	tableName, err := storepkg.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s (
			id, workspace_id, scope, agent_name, agent_tier, type, slug, filename, name,
			description, content, content_hash, injection, mtime_ms, indexed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, tableName),
		doc.ID,
		strings.TrimSpace(doc.WorkspaceID),
		string(doc.Scope.Normalize()),
		strings.TrimSpace(doc.AgentName),
		string(doc.AgentTier.Normalize()),
		string(doc.Type.Normalize()),
		catalogSlug(doc.Filename),
		doc.Filename,
		doc.Name,
		doc.Description,
		doc.Content,
		doc.ContentHash,
		boolToInt(doc.Injection),
		timeToUnixMillis(doc.UpdatedAt),
		timeToUnixMillis(doc.UpdatedAt),
		storepkg.FormatTimestamp(doc.UpdatedAt),
	); err != nil {
		return fmt.Errorf("memory: insert catalog entry %q: %w", doc.Filename, err)
	}
	return nil
}

func upsertCatalogDocumentTx(ctx context.Context, tx catalogWriteExecutor, doc catalogDocument) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_catalog_entries (
			id, workspace_id, scope, agent_name, agent_tier, type, slug, filename, name,
			description, content, content_hash, injection, mtime_ms, indexed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, scope, agent_name, agent_tier, type, slug) DO UPDATE SET
			id = excluded.id,
			filename = excluded.filename,
			type = excluded.type,
			name = excluded.name,
			description = excluded.description,
			content = excluded.content,
			content_hash = excluded.content_hash,
			injection = excluded.injection,
			mtime_ms = excluded.mtime_ms,
			indexed_at = excluded.indexed_at,
			updated_at = excluded.updated_at`,
		doc.ID,
		strings.TrimSpace(doc.WorkspaceID),
		string(doc.Scope.Normalize()),
		strings.TrimSpace(doc.AgentName),
		string(doc.AgentTier.Normalize()),
		string(doc.Type.Normalize()),
		catalogSlug(doc.Filename),
		doc.Filename,
		doc.Name,
		doc.Description,
		doc.Content,
		doc.ContentHash,
		boolToInt(doc.Injection),
		timeToUnixMillis(doc.UpdatedAt),
		timeToUnixMillis(doc.UpdatedAt),
		storepkg.FormatTimestamp(doc.UpdatedAt),
	); err != nil {
		return fmt.Errorf("memory: upsert catalog entry %q: %w", doc.Filename, err)
	}
	return nil
}

func replaceCatalogChunksTx(ctx context.Context, tx catalogWriteExecutor, doc catalogDocument) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM memory_chunks WHERE file_id = ?`, doc.ID); err != nil {
		return fmt.Errorf("memory: delete catalog chunks for %q: %w", doc.Filename, err)
	}
	for _, chunk := range catalogChunksForDocument(doc) {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_chunks (
				id, file_id, content, content_hash, start_line, end_line, indexed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			chunk.id,
			doc.ID,
			chunk.content,
			chunk.contentHash,
			chunk.startLine,
			chunk.endLine,
			timeToUnixMillis(doc.UpdatedAt),
		); err != nil {
			return fmt.Errorf("memory: insert catalog chunk for %q: %w", doc.Filename, err)
		}
	}
	return nil
}

func (c *catalog) upsertCatalogScopeStateTx(
	ctx context.Context,
	tx catalogWriteExecutor,
	scope memcontract.Scope,
	workspaceID string,
) error {
	if err := upsertCatalogStateTx(
		ctx,
		tx,
		catalogScopeStateKey(scope, workspaceID),
		storepkg.FormatTimestamp(c.now().UTC()),
	); err != nil {
		return fmt.Errorf(
			"memory: persist catalog scope state %q/%q: %w",
			scope.Normalize(),
			strings.TrimSpace(workspaceID),
			err,
		)
	}
	return nil
}

func (c *catalog) deleteDocument(
	ctx context.Context,
	scope memcontract.Scope,
	workspaceID string,
	agentName string,
	agentTier memcontract.AgentTier,
	filename string,
) (err error) {
	return c.withCatalogWriteTx(ctx, "catalog document delete", func(tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM memory_chunks
			 WHERE file_id IN (
				SELECT id FROM memory_catalog_entries
				WHERE scope = ? AND workspace_id = ? AND agent_name = ? AND agent_tier = ? AND filename = ?
			 )`,
			string(scope.Normalize()),
			strings.TrimSpace(workspaceID),
			strings.TrimSpace(agentName),
			string(agentTier.Normalize()),
			strings.TrimSpace(filename),
		); err != nil {
			return fmt.Errorf("memory: delete catalog chunks for %q: %w", filename, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM memory_catalog_entries
			 WHERE scope = ? AND workspace_id = ? AND agent_name = ? AND agent_tier = ? AND filename = ?`,
			string(scope.Normalize()),
			strings.TrimSpace(workspaceID),
			strings.TrimSpace(agentName),
			string(agentTier.Normalize()),
			strings.TrimSpace(filename),
		); err != nil {
			return fmt.Errorf("memory: delete catalog entry %q: %w", filename, err)
		}
		if err := upsertCatalogStateTx(
			ctx,
			tx,
			catalogScopeStateKey(scope, workspaceID),
			storepkg.FormatTimestamp(c.now().UTC()),
		); err != nil {
			return fmt.Errorf(
				"memory: persist catalog scope state %q/%q: %w",
				scope.Normalize(),
				strings.TrimSpace(workspaceID),
				err,
			)
		}
		if err := upsertCatalogStateTx(
			ctx,
			tx,
			catalogStateKeyLastReindex,
			storepkg.FormatTimestamp(c.now().UTC()),
		); err != nil {
			return fmt.Errorf("memory: persist catalog reindex timestamp: %w", err)
		}
		return nil
	})
}

func (c *catalog) setLastReindex(ctx context.Context, when time.Time) error {
	if when.IsZero() {
		when = c.now()
	}
	if err := c.upsertState(
		ctx,
		catalogStateKeyLastReindex,
		storepkg.FormatTimestamp(when.UTC()),
	); err != nil {
		return fmt.Errorf("memory: persist catalog reindex timestamp: %w", err)
	}
	return nil
}

func (c *catalog) lastReindex(ctx context.Context) (*time.Time, error) {
	raw, ok, err := c.stateValue(ctx, catalogStateKeyLastReindex)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	parsed, err := storepkg.ParseTimestamp(raw)
	if err != nil {
		return nil, fmt.Errorf("memory: parse catalog reindex timestamp %q: %w", raw, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func (c *catalog) setScopeReady(ctx context.Context, scope memcontract.Scope, workspaceID string) error {
	if err := c.upsertState(
		ctx,
		catalogScopeStateKey(scope, workspaceID),
		storepkg.FormatTimestamp(c.now().UTC()),
	); err != nil {
		return fmt.Errorf(
			"memory: persist catalog scope state %q/%q: %w",
			scope.Normalize(),
			strings.TrimSpace(workspaceID),
			err,
		)
	}
	return nil
}

func (c *catalog) scopeReady(ctx context.Context, scope memcontract.Scope, workspaceID string) (bool, error) {
	raw, ok, err := c.stateValue(ctx, catalogScopeStateKey(scope, workspaceID))
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if _, err := storepkg.ParseTimestamp(raw); err != nil {
		return false, fmt.Errorf(
			"memory: parse catalog scope state %q/%q timestamp %q: %w",
			scope.Normalize(),
			strings.TrimSpace(workspaceID),
			raw,
			err,
		)
	}
	return true, nil
}

func (c *catalog) scopeEntryCount(ctx context.Context, scope memcontract.Scope, workspaceID string) (int, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return 0, err
	}
	if db == nil {
		return 0, nil
	}

	query := `SELECT COUNT(*) FROM memory_catalog_entries`
	args := make([]any, 0, 1)
	switch scope.Normalize() {
	case memcontract.ScopeGlobal:
		query += ` WHERE scope = 'global'`
	case memcontract.ScopeWorkspace:
		query += ` WHERE scope = 'workspace' AND workspace_id = ?`
		args = append(args, strings.TrimSpace(workspaceID))
	case memcontract.ScopeAgent:
		query += ` WHERE scope = 'agent' AND workspace_id = ?`
		args = append(args, strings.TrimSpace(workspaceID))
	default:
		return 0, fmt.Errorf("memory: count catalog entries for unsupported scope %q", scope)
	}

	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf(
			"memory: count catalog entries for scope %q workspace %q: %w",
			scope.Normalize(),
			strings.TrimSpace(workspaceID),
			err,
		)
	}
	return count, nil
}

func (c *catalog) listEntries(ctx context.Context, filters []catalogFilter) ([]catalogDocument, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	rows, err := db.QueryContext(
		ctx,
		`SELECT id, scope, workspace_id, agent_name, agent_tier, filename, type, name,
			description, content, content_hash, updated_at
		 FROM memory_catalog_entries
		 ORDER BY updated_at DESC, filename ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("memory: list catalog entries: %w", err)
	}
	defer func() {
		// rows.Err() or scanErr above reports any actionable read failure after we drain this SELECT result set.
		_ = rows.Close()
	}()

	entries := make([]catalogDocument, 0)
	for rows.Next() {
		entry, scanErr := scanCatalogEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if !catalogFiltersAllow(filters, entry.Scope, entry.WorkspaceID) {
			continue
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate catalog entries: %w", err)
	}
	return entries, nil
}

func (c *catalog) search(
	ctx context.Context,
	query string,
	scope memcontract.Scope,
	workspaceID string,
	limit int,
) ([]memcontract.SearchResult, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	match, err := buildCatalogMatchQuery(query)
	if err != nil {
		return nil, err
	}
	limit = clampSearchLimit(limit)

	base := strings.Join([]string{
		catalogSelectValue,
		catalogEScopePath,
		catalogEWorkspaceIDPath,
		catalogEFilenamePath,
		catalogETypePath,
		catalogENamePath,
		`  e.description,`,
		`  e.updated_at,`,
		`  -bm25(memory_catalog_fts) AS score,`,
		`  snippet(memory_catalog_fts, 2, '[', ']', '...', 18) AS snippet`,
		`FROM memory_catalog_fts`,
		`JOIN memory_catalog_entries e ON e.rowid = memory_catalog_fts.rowid`,
		`WHERE memory_catalog_fts MATCH ?`,
	}, "\n")

	args := []any{match}
	base, args = appendCatalogScopeFilter(base, args, scope, workspaceID)
	base += "\nORDER BY bm25(memory_catalog_fts) ASC, e.updated_at DESC, e.filename ASC\nLIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: search catalog: %w", err)
	}
	defer func() {
		// rows.Err() or scanErr above reports any actionable read failure after we drain this SELECT result set.
		_ = rows.Close()
	}()

	results := make([]memcontract.SearchResult, 0, limit)
	for rows.Next() {
		result, scanErr := scanSearchResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate search results: %w", err)
	}
	return results, nil
}

func (c *catalog) logEvent(ctx context.Context, record memcontract.OperationRecord) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	operation := record.Operation.Normalize()
	if strings.TrimSpace(string(operation)) == "" {
		return errors.New("memory: operation type is required")
	}
	scope := record.Scope.Normalize()
	switch scope {
	case "", memcontract.ScopeGlobal, memcontract.ScopeWorkspace, memcontract.ScopeAgent:
	default:
		return fmt.Errorf("memory: unsupported operation scope %q", record.Scope)
	}
	timestamp := record.Timestamp
	if timestamp.IsZero() {
		timestamp = c.now().UTC()
	}
	agentName := strings.TrimSpace(record.AgentName)
	if agentName == "" {
		agentName = catalogEventAgentName
	}
	record.Operation = operation
	record.Scope = scope
	record.AgentName = agentName
	record.Timestamp = timestamp.UTC()
	return insertMemoryEventDB(ctx, db, record)
}

func insertMemoryEventDB(ctx context.Context, db *sql.DB, record memcontract.OperationRecord) error {
	return storepkg.ExecuteWrite(ctx, db, func(ctx context.Context, tx *storepkg.WriteTx) error {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_events (
				op, scope, agent_name, agent_tier, workspace_id, session_id, actor_kind,
				decision_id, target_id, metadata, ts_ms
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			canonicalEventOp(record),
			nullStringForEmpty(record.Scope.Normalize()),
			nullStringForEmpty(record.AgentName),
			nil,
			nullStringForEmpty(record.Workspace),
			nil,
			"system",
			nil,
			nullStringForEmpty(record.Filename),
			mustEventMetadata(record),
			timeToUnixMillis(record.Timestamp),
		); err != nil {
			return fmt.Errorf("memory: write memory event: %w", err)
		}
		return nil
	})
}

func insertMemoryEventTx(ctx context.Context, tx *sql.Tx, record memcontract.OperationRecord, legacy bool) error {
	metadata := eventMetadata(record)
	if legacy {
		metadata[memoryEventMetadataLegacyIDKey] = strings.TrimSpace(record.ID)
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("memory: encode memory event metadata: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_events (
			op, scope, agent_name, agent_tier, workspace_id, session_id, actor_kind,
			decision_id, target_id, metadata, ts_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		canonicalEventOp(record),
		nullStringForEmpty(record.Scope.Normalize()),
		nullStringForEmpty(record.AgentName),
		nil,
		nullStringForEmpty(record.Workspace),
		nil,
		"system",
		nil,
		nullStringForEmpty(record.Filename),
		string(payload),
		timeToUnixMillis(record.Timestamp),
	); err != nil {
		return fmt.Errorf("memory: migrate memory event: %w", err)
	}
	return nil
}

func (c *catalog) listOperations(
	ctx context.Context,
	query memcontract.OperationHistoryQuery,
) ([]memcontract.OperationRecord, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	operation := canonicalEventOp(memcontract.OperationRecord{Operation: query.Operation.Normalize()})
	workspace := strings.TrimSpace(query.Workspace)
	switch scope := query.Scope.Normalize(); scope {
	case "", memcontract.ScopeGlobal, memcontract.ScopeWorkspace:
	default:
		return nil, fmt.Errorf("memory: unsupported history scope %q", query.Scope)
	}
	scope := string(query.Scope.Normalize())
	var since int64
	if !query.Since.IsZero() {
		since = timeToUnixMillis(query.Since.UTC())
	}
	limit := clampHistoryLimit(query.Limit)

	rows, err := db.QueryContext(
		ctx,
		`SELECT id, op, scope, workspace_id, agent_name, target_id, metadata, ts_ms
		 FROM memory_events
		 WHERE (? = '' OR op = ?)
		 AND (
			(? = '' AND (? = '' OR scope IS NULL OR scope = 'global' OR (scope = 'workspace' AND workspace_id = ?)))
			OR (? = 'global' AND scope = 'global')
			OR (? = 'workspace' AND scope = 'workspace' AND (? = '' OR workspace_id = ?))
		 )
		 AND (? = 0 OR ts_ms >= ?)
		 ORDER BY ts_ms DESC, id DESC
		 LIMIT ?`,
		operation,
		operation,
		scope,
		workspace,
		workspace,
		scope,
		scope,
		workspace,
		workspace,
		since,
		since,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("memory: list operation history: %w", err)
	}
	defer func() {
		// rows.Err() reports actionable read failures after iteration.
		_ = rows.Close()
	}()

	records := make([]memcontract.OperationRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanOperationRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("memory: iterate operation history: %w", err)
	}
	return records, nil
}

func (c *catalog) operationStats(ctx context.Context, filters []catalogFilter) (int, *time.Time, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return 0, nil, err
	}
	if db == nil {
		return 0, nil, nil
	}

	rows, err := db.QueryContext(
		ctx,
		`SELECT scope, workspace_id, ts_ms FROM memory_events`,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("memory: read operation stats: %w", err)
	}
	defer func() {
		// rows.Err() reports actionable read failures after iteration.
		_ = rows.Close()
	}()

	var (
		count    int
		lastTime *time.Time
	)
	for rows.Next() {
		var (
			scope       sql.NullString
			workspaceID sql.NullString
			tsMillis    int64
		)
		if err := rows.Scan(&scope, &workspaceID, &tsMillis); err != nil {
			return 0, nil, fmt.Errorf("memory: scan operation stats: %w", err)
		}
		if !catalogFiltersAllow(filters, memcontract.Scope(scope.String), workspaceID.String) {
			continue
		}
		count++
		parsed := timeFromUnixMillis(tsMillis)
		if lastTime == nil || parsed.After(*lastTime) {
			lastTime = &parsed
		}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("memory: iterate operation stats: %w", err)
	}
	return count, lastTime, nil
}

type catalogFilter struct {
	scope         memcontract.Scope
	workspaceRoot string
	workspaceID   string
}

func catalogFiltersAllow(filters []catalogFilter, scope memcontract.Scope, workspaceID string) bool {
	if len(filters) == 0 {
		return true
	}
	normalizedScope := scope.Normalize()
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	for _, filter := range filters {
		switch filter.scope.Normalize() {
		case memcontract.ScopeGlobal:
			if normalizedScope == "" || normalizedScope == memcontract.ScopeGlobal {
				return true
			}
		case memcontract.ScopeWorkspace:
			if normalizedScope == memcontract.ScopeWorkspace &&
				normalizedWorkspaceID == strings.TrimSpace(filter.workspaceID) {
				return true
			}
		case memcontract.ScopeAgent:
			if normalizedScope == memcontract.ScopeAgent &&
				normalizedWorkspaceID == strings.TrimSpace(filter.workspaceID) {
				return true
			}
		}
	}
	return false
}

func appendCatalogScopeFilter(base string, args []any, scope memcontract.Scope, workspaceID string) (string, []any) {
	switch scope.Normalize() {
	case memcontract.ScopeGlobal:
		return base + "\nAND e.scope = 'global'", args
	case memcontract.ScopeWorkspace:
		return base + "\nAND e.scope = 'workspace' AND e.workspace_id = ?", append(
			args,
			strings.TrimSpace(workspaceID),
		)
	case memcontract.ScopeAgent:
		return base + "\nAND e.scope = 'agent' AND e.workspace_id = ?", append(
			args,
			strings.TrimSpace(workspaceID),
		)
	default:
		trimmedWorkspace := strings.TrimSpace(workspaceID)
		if trimmedWorkspace == "" {
			return base + "\nAND e.scope = 'global'", args
		}
		return base + "\nAND (e.scope = 'global' OR (e.scope = 'workspace' AND e.workspace_id = ?))",
			append(args, trimmedWorkspace)
	}
}

func scanCatalogEntry(scanner interface{ Scan(dest ...any) error }) (catalogDocument, error) {
	var (
		doc          catalogDocument
		scopeRaw     string
		agentTierRaw string
		typeRaw      string
		updatedRaw   string
	)
	if err := scanner.Scan(
		&doc.ID,
		&scopeRaw,
		&doc.WorkspaceID,
		&doc.AgentName,
		&agentTierRaw,
		&doc.Filename,
		&typeRaw,
		&doc.Name,
		&doc.Description,
		&doc.Content,
		&doc.ContentHash,
		&updatedRaw,
	); err != nil {
		return catalogDocument{}, fmt.Errorf("memory: scan catalog entry: %w", err)
	}

	updatedAt, err := storepkg.ParseTimestamp(updatedRaw)
	if err != nil {
		return catalogDocument{}, fmt.Errorf("memory: parse catalog updated_at %q: %w", updatedRaw, err)
	}
	doc.Scope = memcontract.Scope(scopeRaw).Normalize()
	doc.AgentTier = memcontract.AgentTier(agentTierRaw).Normalize()
	doc.Type = memcontract.Type(typeRaw).Normalize()
	doc.UpdatedAt = updatedAt.UTC()
	return doc, nil
}

func scanSearchResult(scanner interface{ Scan(dest ...any) error }) (memcontract.SearchResult, error) {
	var (
		result     memcontract.SearchResult
		scopeRaw   string
		typeRaw    string
		updatedRaw string
		snippet    sql.NullString
	)
	if err := scanner.Scan(
		&scopeRaw,
		&result.Workspace,
		&result.Filename,
		&typeRaw,
		&result.Name,
		&result.Description,
		&updatedRaw,
		&result.Score,
		&snippet,
	); err != nil {
		return memcontract.SearchResult{}, fmt.Errorf("memory: scan search result: %w", err)
	}

	updatedAt, err := storepkg.ParseTimestamp(updatedRaw)
	if err != nil {
		return memcontract.SearchResult{}, fmt.Errorf("memory: parse search result updated_at %q: %w", updatedRaw, err)
	}
	result.Scope = memcontract.Scope(scopeRaw).Normalize()
	result.Type = memcontract.Type(typeRaw).Normalize()
	result.ModTime = updatedAt.UTC()
	if snippet.Valid {
		result.Snippet = cleanSnippet(snippet.String)
	}
	if result.Snippet == "" {
		result.Snippet = result.Description
	}
	return result, nil
}

func scanOperationRecord(scanner interface{ Scan(dest ...any) error }) (memcontract.OperationRecord, error) {
	var (
		record       memcontract.OperationRecord
		id           int64
		operationRaw string
		scopeRaw     sql.NullString
		workspaceID  sql.NullString
		agentName    sql.NullString
		targetID     sql.NullString
		metadataRaw  string
		tsMillis     int64
	)
	if err := scanner.Scan(
		&id,
		&operationRaw,
		&scopeRaw,
		&workspaceID,
		&agentName,
		&targetID,
		&metadataRaw,
		&tsMillis,
	); err != nil {
		return memcontract.OperationRecord{}, fmt.Errorf("memory: scan operation history row: %w", err)
	}
	metadata, err := parseEventMetadata(metadataRaw)
	if err != nil {
		return memcontract.OperationRecord{}, err
	}
	record.ID = fmt.Sprintf("%d", id)
	record.Operation = operationFromEventOp(operationRaw, metadata)
	record.Scope = memcontract.Scope(scopeRaw.String).Normalize()
	record.Workspace = strings.TrimSpace(workspaceID.String)
	record.Filename = firstNonEmpty(metadata[memoryEventMetadataFilenameKey], targetID.String)
	record.AgentName = strings.TrimSpace(agentName.String)
	record.Summary = strings.TrimSpace(metadata[memoryEventMetadataSummaryKey])
	record.Summary = diagnostics.RedactAndBound(record.Summary, maxOperationSummaryBytes)
	record.Timestamp = timeFromUnixMillis(tsMillis)
	return record, nil
}

func canonicalEventOp(record memcontract.OperationRecord) string {
	switch record.Operation.Normalize() {
	case "":
		return ""
	case memcontract.OperationSearch:
		return memoryEventRecallExecuted
	case memcontract.OperationReindex:
		return memoryEventWriteReindex
	case memcontract.OperationDelete:
		return memoryEventWriteCommitted
	default:
		return memoryEventWriteCommitted
	}
}

func operationFromEventOp(op string, metadata map[string]string) memcontract.Operation {
	switch strings.TrimSpace(op) {
	case memoryEventRecallExecuted, memoryEventRecallSkipped:
		return memcontract.OperationSearch
	case memoryEventWriteReindex:
		return memcontract.OperationReindex
	case memoryEventWriteCommitted:
		if metadata[memoryEventMetadataActionKey] == string(memcontract.OperationDelete) {
			return memcontract.OperationDelete
		}
		return memcontract.OperationWrite
	case memoryEventWriteReverted:
		return memcontract.OperationDelete
	default:
		return memcontract.Operation(op).Normalize()
	}
}

func eventMetadata(record memcontract.OperationRecord) map[string]string {
	metadata := map[string]string{}
	if summary := diagnostics.RedactAndBound(
		record.Summary,
		maxOperationSummaryBytes,
	); strings.TrimSpace(
		summary,
	) != "" {
		metadata[memoryEventMetadataSummaryKey] = summary
	}
	if filename := strings.TrimSpace(record.Filename); filename != "" {
		metadata[memoryEventMetadataFilenameKey] = filename
	}
	if action := strings.TrimSpace(string(record.Operation.Normalize())); action != "" {
		metadata[memoryEventMetadataActionKey] = action
	}
	return metadata
}

func mustEventMetadata(record memcontract.OperationRecord) string {
	payload, err := json.Marshal(eventMetadata(record))
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func parseEventMetadata(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]string{}, nil
	}
	var metadata map[string]string
	if err := json.Unmarshal([]byte(trimmed), &metadata); err != nil {
		return nil, fmt.Errorf("memory: parse memory event metadata: %w", err)
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	return metadata, nil
}

func nullStringForEmpty(value any) any {
	switch typed := value.(type) {
	case memcontract.Scope:
		trimmed := strings.TrimSpace(string(typed))
		if trimmed == "" {
			return nil
		}
		return trimmed
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return trimmed
	default:
		return value
	}
}

func timeToUnixMillis(value time.Time) int64 {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().UnixNano() / int64(time.Millisecond)
}

func timeFromUnixMillis(value int64) time.Time {
	return time.Unix(0, value*int64(time.Millisecond)).UTC()
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func catalogSlug(filename string) string {
	base := strings.TrimSpace(filename)
	base = strings.TrimSuffix(base, ".md")
	base = strings.TrimSpace(base)
	if base == "" {
		return strings.TrimSpace(filename)
	}
	return base
}

func catalogChunksForDocument(doc catalogDocument) []catalogChunk {
	searchText := strings.TrimSpace(strings.Join([]string{doc.Name, doc.Description, doc.Content}, "\n"))
	if searchText == "" {
		searchText = strings.TrimSpace(doc.Filename)
	}
	return []catalogChunk{{
		id:          doc.ID + "::chunk:0001",
		content:     searchText,
		contentHash: hashMemoryContent([]byte(searchText)),
		startLine:   1,
		endLine:     max(1, strings.Count(doc.Content, "\n")+1),
	}}
}

func quotedSQLStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "'"+strings.ReplaceAll(strings.TrimSpace(value), "'", "''")+"'")
	}
	return strings.Join(quoted, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildCatalogMatchQuery(query string) (string, error) {
	terms, err := searchQueryTerms(query)
	if err != nil {
		return "", err
	}
	quoted := make([]string, 0, len(terms))
	for _, term := range terms {
		quoted = append(quoted, quoteCatalogMatchTerm(term))
	}
	return strings.Join(quoted, " AND "), nil
}

func quoteCatalogMatchTerm(term string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(term), `"`, `""`) + `"`
}

func tokenizeSearchQuery(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(query)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func cleanSnippet(value string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")
	return strings.Join(strings.Fields(replacer.Replace(strings.TrimSpace(value))), " ")
}

func catalogDocID(scope memcontract.Scope, workspaceID string, filename string) string {
	return strings.Join(
		[]string{string(scope.Normalize()), strings.TrimSpace(workspaceID), strings.TrimSpace(filename)},
		"::",
	)
}

func catalogDocIDForHeader(scope memcontract.Scope, workspaceID string, header memcontract.Header) string {
	if scope.Normalize() != memcontract.ScopeAgent {
		return catalogDocID(scope, workspaceID, header.Filename)
	}
	return strings.Join(
		[]string{
			string(scope.Normalize()),
			strings.TrimSpace(workspaceID),
			strings.TrimSpace(header.AgentName),
			string(header.AgentTier.Normalize()),
			strings.TrimSpace(header.Filename),
		},
		"::",
	)
}

func hashMemoryContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func buildCatalogDocument(
	scope memcontract.Scope,
	workspaceID string,
	header memcontract.Header,
	rawContent []byte,
) (catalogDocument, error) {
	body, err := parseFrontmatter(rawContent, &memcontract.Header{})
	if err != nil {
		return catalogDocument{}, fmt.Errorf("memory: parse memory body for %q: %w", header.Filename, err)
	}
	return catalogDocument{
		ID:          catalogDocIDForHeader(scope, workspaceID, header),
		Scope:       scope.Normalize(),
		WorkspaceID: strings.TrimSpace(workspaceID),
		AgentName:   strings.TrimSpace(header.AgentName),
		AgentTier:   header.AgentTier.Normalize(),
		Filename:    header.Filename,
		Type:        header.Type.Normalize(),
		Name:        header.Name,
		Description: header.Description,
		Content:     strings.TrimSpace(body),
		ContentHash: hashMemoryContent(rawContent),
		Injection:   !strings.HasPrefix(strings.TrimSpace(header.Filename), "_system"),
		UpdatedAt:   header.ModTime.UTC(),
	}, nil
}

func fallbackSearchDocuments(query string, docs []catalogDocument, limit int) ([]memcontract.SearchResult, error) {
	terms, err := searchQueryTerms(query)
	if err != nil {
		return nil, err
	}
	limit = clampSearchLimit(limit)

	results := make([]memcontract.SearchResult, 0, min(limit, len(docs)))
	for _, doc := range docs {
		score := fallbackDocumentScore(doc, terms)
		if score <= 0 {
			continue
		}
		results = append(results, memcontract.SearchResult{
			Filename:    doc.Filename,
			Scope:       doc.Scope,
			Workspace:   doc.WorkspaceID,
			Type:        doc.Type,
			Name:        doc.Name,
			Description: doc.Description,
			Score:       score,
			Snippet:     fallbackSnippet(doc, terms),
			ModTime:     doc.UpdatedAt.UTC(),
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].ModTime.Equal(results[j].ModTime) {
				return results[i].Filename < results[j].Filename
			}
			return results[i].ModTime.After(results[j].ModTime)
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func searchQueryTerms(query string) ([]string, error) {
	terms := tokenizeSearchQuery(query)
	if len(terms) == 0 {
		return nil, wrapValidationError(
			"search query",
			query,
			errors.New("query must include at least one letter or number"),
		)
	}
	return terms, nil
}

func clampSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	return min(limit, maxSearchLimit)
}

func clampHistoryLimit(limit int) int {
	if limit <= 0 {
		return defaultHistoryLimit
	}
	return min(limit, maxHistoryLimit)
}

func (c *catalog) upsertState(ctx context.Context, key string, value string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	return upsertCatalogState(ctx, db, key, value)
}

func (c *catalog) stateValue(ctx context.Context, key string) (string, bool, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return "", false, err
	}
	if db == nil {
		return "", false, nil
	}

	var raw string
	if err := db.QueryRowContext(
		ctx,
		`SELECT value FROM memory_catalog_state WHERE key = ?`,
		key,
	).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("memory: load catalog state %q: %w", key, err)
	}
	return raw, true, nil
}

func catalogScopeStateKey(scope memcontract.Scope, workspaceID string) string {
	return fmt.Sprintf(
		"%s%s::%s",
		catalogStateKeyScopePrefix,
		scope.Normalize(),
		strings.TrimSpace(workspaceID),
	)
}

func upsertCatalogStateTx(ctx context.Context, tx catalogWriteExecutor, key string, value string) error {
	if tx == nil {
		return errors.New("catalog transaction is required")
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_catalog_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key,
		value,
	); err != nil {
		return fmt.Errorf("persist catalog state %q: %w", key, err)
	}
	return nil
}

func upsertCatalogState(ctx context.Context, db *sql.DB, key string, value string) error {
	if db == nil {
		return nil
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO memory_catalog_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key,
		value,
	); err != nil {
		return fmt.Errorf("persist catalog state %q: %w", key, err)
	}
	return nil
}

func fallbackDocumentScore(doc catalogDocument, terms []string) float64 {
	searchable := strings.ToLower(strings.Join([]string{doc.Name, doc.Description, doc.Content}, "\n"))
	score := 0.0
	for _, term := range terms {
		count := strings.Count(searchable, term)
		if count == 0 {
			continue
		}
		score += float64(count)
		if strings.Contains(strings.ToLower(doc.Name), term) {
			score += 5
		}
		if strings.Contains(strings.ToLower(doc.Description), term) {
			score += 2
		}
	}
	return score
}

func fallbackSnippet(doc catalogDocument, terms []string) string {
	candidates := []string{doc.Description, doc.Content}
	for _, candidate := range candidates {
		cleaned := cleanSnippet(candidate)
		lower := strings.ToLower(cleaned)
		for _, term := range terms {
			if strings.Contains(lower, term) {
				return clipSnippet(cleaned, term, 180)
			}
		}
	}
	return cleanSnippet(doc.Description)
}

func clipSnippet(text string, term string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	index := strings.Index(strings.ToLower(text), strings.ToLower(term))
	if index < 0 {
		return text[:maxLen]
	}
	start := max(0, index-(maxLen/3))
	end := min(len(text), start+maxLen)
	return strings.TrimSpace(text[start:end])
}
