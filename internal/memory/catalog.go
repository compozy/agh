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

	"github.com/compozy/agh/internal/diagnostics"
	eventspkg "github.com/compozy/agh/internal/events"
	memcontract "github.com/compozy/agh/internal/memory/contract"
	storepkg "github.com/compozy/agh/internal/store"
)

const (
	catalogEFilenamePath    = "  e.filename,"
	catalogENamePath        = "  e.name,"
	catalogEScopePath       = "  e.scope,"
	catalogETypePath        = "  e.type,"
	catalogEWorkspaceIDPath = "  e.workspace_id,"
	catalogSelectValue      = "SELECT"
)

const (
	defaultSearchLimit         = 10
	maxSearchLimit             = 50
	defaultHistoryLimit        = 25
	maxHistoryLimit            = 100
	maxOperationSummaryBytes   = 2048
	catalogStateKeyLastReindex = "last_reindex_at"
	catalogStateKeyScopePrefix = "scope_synced::"
	catalogEventAgentName      = "daemon"
)

const (
	memoryEventWriteCommitted         = eventspkg.MemoryWriteCommitted
	memoryEventWriteRejected          = eventspkg.MemoryWriteRejected
	memoryEventWriteShadowed          = eventspkg.MemoryWriteShadowed
	memoryEventWriteReindex           = eventspkg.MemoryWriteReindex
	memoryEventWriteReverted          = eventspkg.MemoryWriteReverted
	memoryEventRecallExecuted         = eventspkg.MemoryRecallExecuted
	memoryEventRecallSkipped          = eventspkg.MemoryRecallSkipped
	memoryEventRecallSignalDropped    = eventspkg.MemoryRecallDropped
	memoryEventRecallSignalFailed     = eventspkg.MemoryRecallFailed
	memoryEventDecisionsSummarized    = eventspkg.MemoryDecisionsSummary
	memoryEventDecisionsPruned        = eventspkg.MemoryDecisionsPruned
	memoryEventDreamStarted           = eventspkg.MemoryDreamStarted
	memoryEventDreamPromoted          = eventspkg.MemoryDreamPromoted
	memoryEventDreamFailed            = eventspkg.MemoryDreamFailed
	memoryEventExtractorStarted       = eventspkg.MemoryExtractorStarted
	memoryEventExtractorCompleted     = eventspkg.MemoryExtractorComplete
	memoryEventExtractorFailed        = eventspkg.MemoryExtractorFailed
	memoryEventExtractorCoalesced     = eventspkg.MemoryExtractorCoalesced
	memoryEventExtractorDropped       = eventspkg.MemoryExtractorDropped
	memoryEventDailyRotated           = eventspkg.MemoryDailyRotated
	memoryEventDailyArchived          = eventspkg.MemoryDailyArchived
	memoryEventDailyRestored          = eventspkg.MemoryDailyRestored
	memoryEventDailyPurged            = eventspkg.MemoryDailyPurged
	memoryEventDailyArchivePurged     = eventspkg.MemoryDailyArchivePurged
	memoryEventProviderEnabled        = eventspkg.MemoryProviderEnabled
	memoryEventProviderDisabled       = eventspkg.MemoryProviderDisabled
	memoryEventProviderCollision      = eventspkg.MemoryProviderCollision
	memoryEventWorkspaceRelocated     = eventspkg.MemoryWorkspaceRelocated
	memoryEventWorkspaceRecovered     = eventspkg.MemoryWorkspaceRecovered
	memoryEventAgentPurged            = eventspkg.MemoryAgentPurged
	memoryEventMigrationApplied       = eventspkg.MemoryMigrationApplied
	memoryEventMetadataActionKey      = "action"
	memoryEventMetadataFilenameKey    = "filename"
	memoryEventMetadataLegacyIDKey    = "legacy_id"
	memoryEventMetadataSummaryKey     = "summary"
	memoryEventMetadataQueryKey       = "query"
	memoryEventMetadataResultCountKey = "result_count"
)

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

func (c *catalog) open(ctx context.Context) error {
	if !c.enabled() {
		return nil
	}
	if ctx == nil {
		return errors.New("memory: catalog open context is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		return nil
	}

	db, err := storepkg.OpenSQLiteDatabase(ctx, c.path, func(ctx context.Context, db *sql.DB) error {
		return storepkg.Apply(ctx, db, MigrationStream())
	})
	if err != nil {
		return fmt.Errorf("memory: open catalog database %q: %w", c.path, err)
	}
	c.db = db
	return nil
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
	if c.db == nil {
		return nil, fmt.Errorf("memory: catalog database %q is not open", c.path)
	}
	return c.db, nil
}

func (c *catalog) close(ctx context.Context) error {
	if !c.enabled() {
		return nil
	}
	if ctx == nil {
		return errors.New("memory: catalog close context is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db == nil {
		return nil
	}
	db := c.db
	c.db = nil
	return errors.Join(storepkg.Checkpoint(ctx, db), db.Close())
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
		return c.upsertCatalogIdentityStateTx(
			ctx,
			tx,
			newCatalogIdentity(scope, workspaceID, agentName, agentTier),
		)
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
		if err := c.upsertCatalogIdentityStateTx(
			ctx,
			tx,
			newCatalogIdentity(doc.Scope, doc.WorkspaceID, doc.AgentName, doc.AgentTier),
		); err != nil {
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
		if err := c.upsertCatalogIdentityStateTx(
			ctx,
			tx,
			newCatalogIdentity(scope, workspaceID, agentName, agentTier),
		); err != nil {
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
