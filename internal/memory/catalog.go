package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	aghconfig "github.com/pedronauck/agh/internal/config"
	storepkg "github.com/pedronauck/agh/internal/store"
)

const (
	defaultSearchLimit         = 10
	catalogStateKeyLastReindex = "last_reindex_at"
	catalogEventAgentName      = "daemon"
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

type catalog struct {
	path string
	now  func() time.Time

	mu sync.Mutex
	db *sql.DB
}

type catalogDocument struct {
	ID            string
	Scope         Scope
	WorkspaceID   string
	WorkspaceRoot string
	Filename      string
	Type          Type
	Name          string
	Description   string
	Content       string
	ContentHash   string
	UpdatedAt     time.Time
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
		return storepkg.EnsureSchema(ctx, db, catalogSchemaStatements)
	})
	if err != nil {
		return nil, fmt.Errorf("memory: open catalog database %q: %w", c.path, err)
	}
	c.db = db
	return c.db, nil
}

func (c *catalog) replaceScope(
	ctx context.Context,
	scope Scope,
	workspaceRoot string,
	docs []catalogDocument,
) (err error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("memory: begin catalog scope replace: %w", err)
	}
	defer func() {
		if tx == nil {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil &&
			!errors.Is(rollbackErr, sql.ErrTxDone) &&
			err == nil {
			err = fmt.Errorf("memory: rollback catalog scope replace: %w", rollbackErr)
		}
	}()

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM memory_catalog_entries WHERE scope = ? AND workspace_root = ?`,
		string(scope.Normalize()),
		strings.TrimSpace(workspaceRoot),
	); err != nil {
		return fmt.Errorf("memory: clear catalog scope %q/%q: %w", scope, workspaceRoot, err)
	}

	for _, doc := range docs {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO memory_catalog_entries (
				id, scope, workspace_id, workspace_root, filename, type, name,
				description, content, content_hash, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.ID,
			string(doc.Scope.Normalize()),
			strings.TrimSpace(doc.WorkspaceID),
			strings.TrimSpace(doc.WorkspaceRoot),
			doc.Filename,
			string(doc.Type.Normalize()),
			doc.Name,
			doc.Description,
			doc.Content,
			doc.ContentHash,
			storepkg.FormatTimestamp(doc.UpdatedAt),
		); err != nil {
			return fmt.Errorf("memory: insert catalog entry %q: %w", doc.Filename, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("memory: commit catalog scope replace: %w", err)
	}
	tx = nil
	return nil
}

func (c *catalog) setLastReindex(ctx context.Context, when time.Time) error {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	if when.IsZero() {
		when = c.now()
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO memory_catalog_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		catalogStateKeyLastReindex,
		storepkg.FormatTimestamp(when.UTC()),
	); err != nil {
		return fmt.Errorf("memory: persist catalog reindex timestamp: %w", err)
	}
	return nil
}

func (c *catalog) lastReindex(ctx context.Context) (*time.Time, error) {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, nil
	}

	var raw string
	if err := db.QueryRowContext(
		ctx,
		`SELECT value FROM memory_catalog_state WHERE key = ?`,
		catalogStateKeyLastReindex,
	).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("memory: load catalog reindex timestamp: %w", err)
	}

	parsed, err := storepkg.ParseTimestamp(raw)
	if err != nil {
		return nil, fmt.Errorf("memory: parse catalog reindex timestamp %q: %w", raw, err)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func (c *catalog) scopeEntryCount(ctx context.Context, scope Scope, workspaceRoot string) (int, error) {
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
	case ScopeGlobal:
		query += ` WHERE scope = 'global'`
	case ScopeWorkspace:
		query += ` WHERE scope = 'workspace' AND workspace_root = ?`
		args = append(args, strings.TrimSpace(workspaceRoot))
	default:
		return 0, fmt.Errorf("memory: count catalog entries for unsupported scope %q", scope)
	}

	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf(
			"memory: count catalog entries for scope %q workspace %q: %w",
			scope.Normalize(),
			strings.TrimSpace(workspaceRoot),
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
		`SELECT id, scope, workspace_id, workspace_root, filename, type, name, description, content, content_hash, updated_at
		 FROM memory_catalog_entries
		 ORDER BY updated_at DESC, filename ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("memory: list catalog entries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	allowedGlobal := false
	allowedWorkspaces := make(map[string]struct{})
	for _, filter := range filters {
		switch filter.scope.Normalize() {
		case ScopeGlobal:
			allowedGlobal = true
		case ScopeWorkspace:
			allowedWorkspaces[strings.TrimSpace(filter.workspaceRoot)] = struct{}{}
		}
	}

	entries := make([]catalogDocument, 0)
	for rows.Next() {
		entry, scanErr := scanCatalogEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if len(filters) > 0 {
			switch entry.Scope.Normalize() {
			case ScopeGlobal:
				if !allowedGlobal {
					continue
				}
			case ScopeWorkspace:
				if _, ok := allowedWorkspaces[strings.TrimSpace(entry.WorkspaceRoot)]; !ok {
					continue
				}
			default:
				continue
			}
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
	scope Scope,
	workspaceRoot string,
	limit int,
) ([]SearchResult, error) {
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
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	base := strings.Join([]string{
		`SELECT`,
		`  e.scope,`,
		`  e.workspace_root,`,
		`  e.filename,`,
		`  e.type,`,
		`  e.name,`,
		`  e.description,`,
		`  e.updated_at,`,
		`  -bm25(memory_catalog_fts) AS score,`,
		`  snippet(memory_catalog_fts, 2, '[', ']', '...', 18) AS snippet`,
		`FROM memory_catalog_fts`,
		`JOIN memory_catalog_entries e ON e.rowid = memory_catalog_fts.rowid`,
		`WHERE memory_catalog_fts MATCH ?`,
	}, "\n")

	args := []any{match}
	base, args = appendCatalogScopeFilter(base, args, scope, workspaceRoot)
	base += "\nORDER BY bm25(memory_catalog_fts) ASC, e.updated_at DESC, e.filename ASC\nLIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("memory: search catalog: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	results := make([]SearchResult, 0, limit)
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

func (c *catalog) logEvent(ctx context.Context, eventType string, summary string) error {
	db, err := c.ensureDB(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	if strings.TrimSpace(eventType) == "" {
		return errors.New("memory: event type is required")
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO memory_operation_log (id, type, agent_name, summary, timestamp) VALUES (?, ?, ?, ?, ?)`,
		storepkg.NewID("memevt"),
		strings.TrimSpace(eventType),
		catalogEventAgentName,
		strings.TrimSpace(summary),
		storepkg.FormatTimestamp(c.now().UTC()),
	); err != nil {
		return fmt.Errorf("memory: write memory operation log: %w", err)
	}
	return nil
}

type catalogFilter struct {
	scope         Scope
	workspaceRoot string
}

func appendCatalogScopeFilter(base string, args []any, scope Scope, workspaceRoot string) (string, []any) {
	switch scope.Normalize() {
	case ScopeGlobal:
		return base + "\nAND e.scope = 'global'", args
	case ScopeWorkspace:
		return base + "\nAND e.scope = 'workspace' AND e.workspace_root = ?", append(
			args,
			strings.TrimSpace(workspaceRoot),
		)
	default:
		trimmedWorkspace := strings.TrimSpace(workspaceRoot)
		if trimmedWorkspace == "" {
			return base + "\nAND e.scope = 'global'", args
		}
		return base + "\nAND (e.scope = 'global' OR (e.scope = 'workspace' AND e.workspace_root = ?))",
			append(args, trimmedWorkspace)
	}
}

func scanCatalogEntry(scanner interface{ Scan(dest ...any) error }) (catalogDocument, error) {
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
		return catalogDocument{}, fmt.Errorf("memory: scan catalog entry: %w", err)
	}

	updatedAt, err := storepkg.ParseTimestamp(updatedRaw)
	if err != nil {
		return catalogDocument{}, fmt.Errorf("memory: parse catalog updated_at %q: %w", updatedRaw, err)
	}
	doc.Scope = Scope(scopeRaw).Normalize()
	doc.Type = Type(typeRaw).Normalize()
	doc.UpdatedAt = updatedAt.UTC()
	return doc, nil
}

func scanSearchResult(scanner interface{ Scan(dest ...any) error }) (SearchResult, error) {
	var (
		result     SearchResult
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
		return SearchResult{}, fmt.Errorf("memory: scan search result: %w", err)
	}

	updatedAt, err := storepkg.ParseTimestamp(updatedRaw)
	if err != nil {
		return SearchResult{}, fmt.Errorf("memory: parse search result updated_at %q: %w", updatedRaw, err)
	}
	result.Scope = Scope(scopeRaw).Normalize()
	result.Type = Type(typeRaw).Normalize()
	result.ModTime = updatedAt.UTC()
	if snippet.Valid {
		result.Snippet = cleanSnippet(snippet.String)
	}
	if result.Snippet == "" {
		result.Snippet = result.Description
	}
	return result, nil
}

func buildCatalogMatchQuery(query string) (string, error) {
	terms := tokenizeSearchQuery(query)
	if len(terms) == 0 {
		return "", wrapValidationError("search query", query, errors.New("query is required"))
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

func catalogDocID(scope Scope, workspaceRoot string, filename string) string {
	return strings.Join(
		[]string{string(scope.Normalize()), strings.TrimSpace(workspaceRoot), strings.TrimSpace(filename)},
		"::",
	)
}

func hashMemoryContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func buildCatalogDocument(
	scope Scope,
	workspaceRoot string,
	header Header,
	rawContent []byte,
) (catalogDocument, error) {
	body, err := parseFrontmatter(rawContent, &Header{})
	if err != nil {
		return catalogDocument{}, fmt.Errorf("memory: parse memory body for %q: %w", header.Filename, err)
	}
	return catalogDocument{
		ID:            catalogDocID(scope, workspaceRoot, header.Filename),
		Scope:         scope.Normalize(),
		WorkspaceRoot: strings.TrimSpace(workspaceRoot),
		Filename:      header.Filename,
		Type:          header.Type.Normalize(),
		Name:          header.Name,
		Description:   header.Description,
		Content:       strings.TrimSpace(body),
		ContentHash:   hashMemoryContent(rawContent),
		UpdatedAt:     header.ModTime.UTC(),
	}, nil
}

func fallbackSearchDocuments(query string, docs []catalogDocument, limit int) ([]SearchResult, error) {
	terms := tokenizeSearchQuery(query)
	if len(terms) == 0 {
		return nil, wrapValidationError("search query", query, errors.New("query is required"))
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	results := make([]SearchResult, 0, min(limit, len(docs)))
	for _, doc := range docs {
		score := fallbackDocumentScore(doc, terms)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			Filename:    doc.Filename,
			Scope:       doc.Scope,
			Workspace:   doc.WorkspaceRoot,
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

func deriveWorkspaceRoot(memoryDir string) string {
	clean := strings.TrimSpace(memoryDir)
	if clean == "" {
		return ""
	}
	suffix := string(filepath.Separator) + aghconfig.DirName + string(filepath.Separator) + memoryDirName
	if strings.HasSuffix(clean, suffix) {
		return filepath.Dir(filepath.Dir(clean))
	}
	return ""
}
