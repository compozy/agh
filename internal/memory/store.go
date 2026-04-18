package memory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/fileutil"
	"github.com/pedronauck/agh/internal/frontmatter"
)

const (
	indexFilename     = "MEMORY.md"
	maxScanEntries    = 200
	defaultIndexLines = 200
	defaultIndexBytes = 25_000
	dirPerm           = 0o755
	filePerm          = 0o644
	memoryDirName     = "memory"
)

var (
	// ErrValidation marks memory input and metadata validation failures.
	ErrValidation = errors.New("memory: validation error")
)

// Store manages memory files for the global and workspace scopes.
type Store struct {
	globalDir     string
	workspaceDir  string
	maxIndexLines int
	maxIndexBytes int
	logger        *slog.Logger
	catalog       *catalog
}

var _ Backend = (*Store)(nil)

// NewStore constructs a Store for the provided global memory directory.
func NewStore(globalDir string, opts ...StoreOption) *Store {
	store := &Store{
		globalDir:     cleanDirPath(globalDir),
		maxIndexLines: defaultIndexLines,
		maxIndexBytes: defaultIndexBytes,
		logger:        slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store
}

// StoreOption customizes a Store instance.
type StoreOption func(*Store)

// WithCatalogDatabasePath enables the derived SQLite-backed memory catalog in
// the shared global database file.
func WithCatalogDatabasePath(path string) StoreOption {
	return func(store *Store) {
		if store == nil {
			return
		}
		store.catalog = newCatalog(path, func() time.Time {
			return time.Now().UTC()
		})
	}
}

// ForWorkspace returns a clone of the store bound to the supplied workspace root.
func (s *Store) ForWorkspace(workspaceRoot string) *Store {
	clone := *s
	clone.workspaceDir = workspaceMemoryDir(workspaceRoot)
	return &clone
}

// List is the backend-aligned alias for Scan.
func (s *Store) List(scope Scope) ([]Header, error) {
	return s.Scan(scope)
}

// LoadPromptIndex is the backend-aligned alias for LoadIndex.
func (s *Store) LoadPromptIndex(scope Scope) (content string, truncated bool, err error) {
	return s.LoadIndex(scope)
}

// EnsureDirs creates the configured memory directories when missing.
func (s *Store) EnsureDirs() error {
	for _, dir := range []string{s.globalDir, s.workspaceDir} {
		if strings.TrimSpace(dir) == "" {
			continue
		}

		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("memory: ensure directory %q: %w", dir, err)
		}
	}

	if strings.TrimSpace(s.globalDir) == "" {
		return wrapValidationError("ensure directory", "global", errors.New("global directory is required"))
	}

	return nil
}

// Read returns the raw file contents for a memory file in the requested scope.
func (s *Store) Read(scope Scope, filename string) ([]byte, error) {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("memory: read %q: %w", path, err)
	}

	return content, nil
}

// Exists reports whether a memory file exists in the requested scope.
func (s *Store) Exists(scope Scope, filename string) (bool, error) {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("memory: stat %q: %w", path, err)
	}

	return true, nil
}

// Write validates the memory frontmatter and persists the raw file contents atomically.
func (s *Store) Write(scope Scope, filename string, content []byte) error {
	var header Header
	if _, err := parseFrontmatter(content, &header); err != nil {
		return fmt.Errorf("memory: parse frontmatter %q: %w", filename, fmt.Errorf("%w: %v", ErrValidation, err))
	}
	if err := header.Validate(); err != nil {
		return wrapValidationError("validate frontmatter", filename, err)
	}

	path, err := s.pathFor(scope, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("memory: ensure directory %q: %w", filepath.Dir(path), err)
	}
	if err := fileutil.AtomicWriteFile(path, content, filePerm); err != nil {
		return fmt.Errorf("memory: write %q: %w", path, err)
	}
	s.syncScopeAfterMutation(scope.Normalize(), filepath.Base(path), "write")
	s.logMutationEvent("write", scope.Normalize(), filepath.Base(path))

	return nil
}

// Delete removes a memory file and strips any matching entry from the local MEMORY.md index.
func (s *Store) Delete(scope Scope, filename string) error {
	path, err := s.pathFor(scope, filename)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("memory: delete %q: %w", path, err)
	}
	if filepath.Base(path) == indexFilename {
		return nil
	}
	s.syncScopeAfterMutation(scope.Normalize(), filepath.Base(path), "delete")
	s.logMutationEvent("delete", scope.Normalize(), filepath.Base(path))

	return nil
}

// Scan lists memory headers for a scope, sorted newest-first and capped at 200 files.
func (s *Store) Scan(scope Scope) ([]Header, error) {
	return s.scan(scope, maxScanEntries)
}

func (s *Store) scan(scope Scope, limit int) ([]Header, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Header{}, nil
		}
		return nil, fmt.Errorf("memory: scan %q: %w", dir, err)
	}

	capacity := len(entries)
	if limit > 0 {
		capacity = min(capacity, limit)
	}
	headers := make([]Header, 0, capacity)
	candidates := make([]scanCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || shouldSkipFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			s.warn(
				"memory: skip memory file with unreadable metadata",
				"scope",
				scope,
				"filename",
				entry.Name(),
				"error",
				err,
			)
			continue
		}

		candidates = append(candidates, scanCandidate{
			name:    entry.Name(),
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].name < candidates[j].name
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate.path)
		if err != nil {
			s.warn("memory: skip unreadable memory file", "scope", scope, "path", candidate.path, "error", err)
			continue
		}

		var header Header
		if _, err := parseFrontmatter(content, &header); err != nil {
			s.warn("memory: skip malformed memory file", "scope", scope, "path", candidate.path, "error", err)
			continue
		}
		if err := header.Validate(); err != nil {
			s.warn("memory: skip invalid memory metadata", "scope", scope, "path", candidate.path, "error", err)
			continue
		}

		header.Filename = candidate.name
		header.FilePath = candidate.path
		header.ModTime = candidate.modTime
		headers = append(headers, header)

		if limit > 0 && len(headers) == limit {
			break
		}
	}

	return headers, nil
}

// LoadIndex reads MEMORY.md for a scope and truncates it to the prompt-safe limits.
func (s *Store) LoadIndex(scope Scope) (content string, truncated bool, err error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return "", false, err
	}

	headers, err := s.scan(scope.Normalize(), 0)
	if err != nil {
		return "", false, err
	}

	path := filepath.Join(dir, indexFilename)
	indexBytes, err := os.ReadFile(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		if len(headers) == 0 {
			return "", false, nil
		}
		generated, generatedTruncated := truncateIndex(renderIndex(headers), s.maxIndexLines, s.maxIndexBytes)
		return generated, generatedTruncated, nil
	default:
		return "", false, fmt.Errorf("memory: load index %q: %w", path, err)
	}

	indexContent := string(indexBytes)
	if indexMatchesHeaders(indexContent, headers) {
		content, truncated = truncateIndex(indexContent, s.maxIndexLines, s.maxIndexBytes)
		if truncated {
			s.warn(
				"memory: truncated memory index",
				"scope", scope,
				"path", path,
				"max_lines", s.maxIndexLines,
				"max_bytes", s.maxIndexBytes,
			)
		}
		return content, truncated, nil
	}

	s.warn("memory: synthesized prompt index from memory files", "scope", scope, "path", path)
	generated, generatedTruncated := truncateIndex(renderIndex(headers), s.maxIndexLines, s.maxIndexBytes)
	return generated, generatedTruncated, nil
}

// Search performs bounded lexical memory search across the visible scopes.
func (s *Store) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		return nil, errors.New("memory: search context is required")
	}

	scope, workspaceRoot, err := s.normalizeScopeAndWorkspace(opts.Scope, opts.Workspace)
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	if err := s.ensureCatalogReady(ctx, scope, workspaceRoot); err != nil {
		return nil, err
	}
	if s.catalog != nil {
		results, err := s.catalog.search(ctx, query, scope, workspaceRoot, limit)
		if err != nil {
			return nil, err
		}
		if err := s.logCatalogEvent(
			ctx,
			"memory.search",
			fmt.Sprintf("query=%q results=%d", strings.TrimSpace(query), len(results)),
		); err != nil {
			s.warn("memory: record search event failed", "error", err)
		}
		if len(results) > 0 {
			return results, nil
		}
	}

	docs, err := s.collectSearchDocuments(scope, workspaceRoot)
	if err != nil {
		return nil, err
	}
	results, err := fallbackSearchDocuments(query, docs, limit)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Reindex rebuilds the derived catalog from the Markdown source of truth.
func (s *Store) Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error) {
	if ctx == nil {
		return ReindexResult{}, errors.New("memory: reindex context is required")
	}

	scope, workspaceRoot, err := s.normalizeScopeAndWorkspace(opts.Scope, opts.Workspace)
	if err != nil {
		return ReindexResult{}, err
	}

	indexed, err := s.reindexScopes(ctx, scope, workspaceRoot)
	if err != nil {
		return ReindexResult{}, err
	}
	completedAt := time.Now().UTC()
	if err := s.logCatalogEvent(
		ctx,
		"memory.reindex",
		fmt.Sprintf("scope=%s workspace=%s indexed=%d", string(scope.Normalize()), workspaceRoot, indexed),
	); err != nil {
		s.warn("memory: record reindex event failed", "error", err)
	}
	return ReindexResult{
		IndexedFiles: indexed,
		Scope:        scope.Normalize(),
		Workspace:    workspaceRoot,
		CompletedAt:  completedAt,
	}, nil
}

// HealthStats returns derived-catalog stats for the visible scopes.
func (s *Store) HealthStats(ctx context.Context, workspaces []string) (HealthStats, error) {
	if ctx == nil {
		return HealthStats{}, errors.New("memory: health stats context is required")
	}
	if s.catalog == nil {
		return HealthStats{}, nil
	}

	filters := make([]catalogFilter, 0, len(workspaces)+1)
	filters = append(filters, catalogFilter{scope: ScopeGlobal})
	seen := map[string]struct{}{}
	for _, workspace := range workspaces {
		trimmed := strings.TrimSpace(workspace)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		filters = append(filters, catalogFilter{scope: ScopeWorkspace, workspaceRoot: trimmed})
	}

	for _, filter := range filters {
		if err := s.ensureCatalogFilterReady(ctx, filter); err != nil {
			return HealthStats{}, err
		}
	}

	entries, err := s.catalog.listEntries(ctx, filters)
	if err != nil {
		return HealthStats{}, err
	}
	actual, err := s.collectActualCatalogIDs(filters)
	if err != nil {
		return HealthStats{}, err
	}

	orphaned := 0
	for _, entry := range entries {
		if _, exists := actual[entry.ID]; !exists {
			orphaned++
		}
	}

	lastReindex, err := s.catalog.lastReindex(ctx)
	if err != nil {
		return HealthStats{}, err
	}
	return HealthStats{
		IndexedFiles:  len(entries),
		OrphanedFiles: orphaned,
		LastReindex:   lastReindex,
	}, nil
}

func (s *Store) dirForScope(scope Scope) (string, error) {
	normalized := scope.Normalize()
	if err := normalized.Validate(); err != nil {
		return "", wrapValidationError("resolve scope", string(scope), err)
	}

	switch normalized {
	case ScopeGlobal:
		if s.globalDir == "" {
			return "", wrapValidationError("resolve scope", string(scope), errors.New("global directory is required"))
		}
		return s.globalDir, nil
	case ScopeWorkspace:
		if s.workspaceDir == "" {
			return "", wrapValidationError(
				"resolve scope",
				string(scope),
				errors.New("workspace directory is required"),
			)
		}
		return s.workspaceDir, nil
	default:
		return "", wrapValidationError("resolve scope", string(scope), fmt.Errorf("unsupported scope %q", scope))
	}
}

func (s *Store) pathFor(scope Scope, filename string) (string, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return "", err
	}

	base, err := cleanFilename(filename)
	if err != nil {
		return "", wrapValidationError("resolve filename", filename, err)
	}

	return filepath.Join(dir, base), nil
}

func (s *Store) syncScope(ctx context.Context, scope Scope) error {
	scope = scope.Normalize()
	headers, err := s.scan(scope, 0)
	if err != nil {
		return err
	}
	if err := s.syncIndex(scope, headers); err != nil {
		return err
	}
	if err := s.syncCatalogScope(ctx, scope, headers); err != nil {
		return err
	}
	return nil
}

func (s *Store) syncIndex(scope Scope, headers []Header) error {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return err
	}

	indexPath := filepath.Join(dir, indexFilename)
	if len(headers) == 0 {
		if err := os.Remove(indexPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("memory: remove empty index %q: %w", indexPath, err)
		}
		return nil
	}

	if err := fileutil.AtomicWriteFile(indexPath, []byte(renderIndex(headers)), filePerm); err != nil {
		return fmt.Errorf("memory: write index %q: %w", indexPath, err)
	}
	return nil
}

func (s *Store) syncCatalogScope(ctx context.Context, scope Scope, headers []Header) error {
	if s.catalog == nil {
		return nil
	}
	workspaceRoot := ""
	if scope.Normalize() == ScopeWorkspace {
		workspaceRoot = deriveWorkspaceRoot(s.workspaceDir)
	}
	docs, err := s.documentsForHeaders(scope, workspaceRoot, headers)
	if err != nil {
		return err
	}
	if err := s.catalog.replaceScope(ctx, scope, workspaceRoot, docs); err != nil {
		return err
	}
	if err := s.catalog.setLastReindex(ctx, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (s *Store) warn(msg string, args ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}

type scanCandidate struct {
	name    string
	path    string
	modTime time.Time
}

func (s *Store) normalizeScopeAndWorkspace(scope Scope, workspace string) (Scope, string, error) {
	normalizedScope := scope.Normalize()
	if normalizedScope != "" {
		if err := normalizedScope.Validate(); err != nil {
			return "", "", wrapValidationError("resolve scope", string(scope), err)
		}
	}

	workspaceRoot := cleanDirPath(workspace)
	if workspaceRoot == "" {
		workspaceRoot = deriveWorkspaceRoot(s.workspaceDir)
	}
	if normalizedScope == ScopeWorkspace && workspaceRoot == "" {
		return "", "", wrapValidationError(
			"resolve scope",
			string(scope),
			errors.New("workspace directory is required"),
		)
	}
	return normalizedScope, workspaceRoot, nil
}

func (s *Store) ensureCatalogReady(ctx context.Context, scope Scope, workspaceRoot string) error {
	if s.catalog == nil {
		return nil
	}

	filters := []catalogFilter{{scope: ScopeGlobal}}
	switch scope.Normalize() {
	case ScopeGlobal:
		filters = filters[:1]
	case ScopeWorkspace:
		filters = []catalogFilter{{scope: ScopeWorkspace, workspaceRoot: workspaceRoot}}
	default:
		if strings.TrimSpace(workspaceRoot) != "" {
			filters = append(filters, catalogFilter{scope: ScopeWorkspace, workspaceRoot: workspaceRoot})
		}
	}

	for _, filter := range filters {
		if err := s.ensureCatalogFilterReady(ctx, filter); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureCatalogFilterReady(ctx context.Context, filter catalogFilter) error {
	if s.catalog == nil {
		return nil
	}

	entryCount, err := s.catalog.scopeEntryCount(ctx, filter.scope, filter.workspaceRoot)
	if err != nil {
		return err
	}
	if entryCount > 0 {
		return nil
	}

	_, err = s.reindexScopes(ctx, filter.scope, filter.workspaceRoot)
	return err
}

func (s *Store) reindexScopes(ctx context.Context, scope Scope, workspaceRoot string) (int, error) {
	if s.catalog == nil {
		return 0, nil
	}

	total := 0
	seenWorkspace := strings.TrimSpace(workspaceRoot)

	reindexScope := func(scope Scope, workspaceRoot string) error {
		headers, err := s.headersForCatalogScope(scope, workspaceRoot)
		if err != nil {
			return err
		}
		docs, err := s.documentsForHeaders(scope, workspaceRoot, headers)
		if err != nil {
			return err
		}
		if err := s.catalog.replaceScope(ctx, scope, workspaceRoot, docs); err != nil {
			return err
		}
		total += len(docs)
		return nil
	}

	switch scope.Normalize() {
	case ScopeGlobal:
		if err := reindexScope(ScopeGlobal, ""); err != nil {
			return 0, err
		}
	case ScopeWorkspace:
		if err := reindexScope(ScopeWorkspace, seenWorkspace); err != nil {
			return 0, err
		}
	default:
		if err := reindexScope(ScopeGlobal, ""); err != nil {
			return 0, err
		}
		if seenWorkspace != "" {
			if err := reindexScope(ScopeWorkspace, seenWorkspace); err != nil {
				return 0, err
			}
		}
	}

	if err := s.catalog.setLastReindex(ctx, time.Now().UTC()); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) headersForCatalogScope(scope Scope, workspaceRoot string) ([]Header, error) {
	target := s
	if scope.Normalize() == ScopeWorkspace {
		target = s.ForWorkspace(workspaceRoot)
	}
	return target.scan(scope, 0)
}

func (s *Store) documentsForHeaders(scope Scope, workspaceRoot string, headers []Header) ([]catalogDocument, error) {
	target := s
	if scope.Normalize() == ScopeWorkspace {
		target = s.ForWorkspace(workspaceRoot)
	}

	docs := make([]catalogDocument, 0, len(headers))
	for _, header := range headers {
		rawContent, err := target.Read(scope, header.Filename)
		if err != nil {
			return nil, err
		}
		doc, err := buildCatalogDocument(scope, workspaceRoot, header, rawContent)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (s *Store) collectSearchDocuments(scope Scope, workspaceRoot string) ([]catalogDocument, error) {
	scopes := []struct {
		scope     Scope
		workspace string
	}{{scope: ScopeGlobal}}
	if scope.Normalize() == ScopeWorkspace {
		scopes = []struct {
			scope     Scope
			workspace string
		}{{scope: ScopeWorkspace, workspace: workspaceRoot}}
	} else if strings.TrimSpace(workspaceRoot) != "" {
		scopes = append(scopes, struct {
			scope     Scope
			workspace string
		}{scope: ScopeWorkspace, workspace: workspaceRoot})
	}

	docs := make([]catalogDocument, 0)
	for _, item := range scopes {
		headers, err := s.headersForCatalogScope(item.scope, item.workspace)
		if err != nil {
			return nil, err
		}
		items, err := s.documentsForHeaders(item.scope, item.workspace, headers)
		if err != nil {
			return nil, err
		}
		docs = append(docs, items...)
	}
	return docs, nil
}

func (s *Store) collectActualCatalogIDs(filters []catalogFilter) (map[string]struct{}, error) {
	actual := make(map[string]struct{})
	for _, filter := range filters {
		headers, err := s.headersForCatalogScope(filter.scope, filter.workspaceRoot)
		if err != nil {
			return nil, err
		}
		for _, header := range headers {
			actual[catalogDocID(filter.scope, filter.workspaceRoot, header.Filename)] = struct{}{}
		}
	}
	return actual, nil
}

func (s *Store) logCatalogEvent(ctx context.Context, eventType string, summary string) error {
	if s.catalog == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.catalog.logEvent(ctx, eventType, summary)
}

func (s *Store) syncScopeAfterMutation(scope Scope, filename string, action string) {
	if err := s.syncScope(context.Background(), scope.Normalize()); err != nil {
		s.warn(
			"memory: sync derived state failed after mutation",
			"action", strings.TrimSpace(action),
			"scope", scope.Normalize(),
			"filename", strings.TrimSpace(filename),
			"error", err,
		)
	}
}

func (s *Store) logMutationEvent(action string, scope Scope, filename string) {
	if err := s.logCatalogEvent(
		context.Background(),
		"memory."+strings.TrimSpace(action),
		fmt.Sprintf("scope=%s filename=%s", scope.Normalize(), strings.TrimSpace(filename)),
	); err != nil {
		s.warn(
			"memory: record mutation event failed",
			"action", strings.TrimSpace(action),
			"scope", scope.Normalize(),
			"filename", strings.TrimSpace(filename),
			"error", err,
		)
	}
}

func truncateIndex(content string, maxLines int, maxBytes int) (string, bool) {
	if content == "" {
		return "", false
	}
	if maxLines <= 0 || maxBytes <= 0 {
		return "", true
	}

	lines := strings.SplitAfter(content, "\n")
	var builder strings.Builder

	for idx, line := range lines {
		if line == "" && idx == len(lines)-1 {
			continue
		}
		if idx >= maxLines {
			return builder.String(), true
		}
		if builder.Len()+len(line) > maxBytes {
			if builder.Len() == 0 {
				return truncateToUTF8Boundary(line, maxBytes), true
			}
			return builder.String(), true
		}
		builder.WriteString(line)
	}

	return builder.String(), false
}

func truncateToUTF8Boundary(value string, maxBytes int) string {
	if maxBytes <= 0 || value == "" {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}

	truncated := value[:maxBytes]
	for truncated != "" && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}

	return truncated
}

func firstMarkdownLinkTarget(line string) (string, bool) {
	start := strings.Index(line, "](")
	if start < 0 {
		return "", false
	}
	start += 2

	depth := 0
	for idx := start; idx < len(line); idx++ {
		switch line[idx] {
		case '\\':
			if idx+1 < len(line) {
				idx++
			}
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return strings.TrimSpace(line[start:idx]), true
			}
			depth--
		}
	}

	return "", false
}

func renderIndex(headers []Header) string {
	if len(headers) == 0 {
		return ""
	}
	lines := make([]string, 0, len(headers))
	for _, header := range headers {
		lines = append(lines, renderIndexLine(header))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderIndexLine(header Header) string {
	header.Normalize()
	if header.Description == "" {
		return fmt.Sprintf("- [%s](%s)", header.Name, header.Filename)
	}
	return fmt.Sprintf("- [%s](%s) - %s", header.Name, header.Filename, header.Description)
}

func indexMatchesHeaders(content string, headers []Header) bool {
	return strings.TrimSpace(content) == strings.TrimSpace(renderIndex(headers))
}

func shouldSkipFile(name string) bool {
	return name == indexFilename || strings.HasPrefix(name, ".")
}

func cleanDirPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	return filepath.Clean(trimmed)
}

func cleanFilename(filename string) (string, error) {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return "", fmt.Errorf("filename is required")
	}
	if trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("filename %q is invalid", filename)
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("filename %q must not include path separators", filename)
	}

	return trimmed, nil
}

func wrapValidationError(operation string, target string, err error) error {
	return fmt.Errorf("memory: %s %q: %w", operation, target, fmt.Errorf("%w: %v", ErrValidation, err))
}

func workspaceMemoryDir(workspaceRoot string) string {
	trimmed := strings.TrimSpace(workspaceRoot)
	if trimmed == "" {
		return ""
	}

	return filepath.Join(filepath.Clean(trimmed), aghconfig.DirName, memoryDirName)
}

func parseFrontmatter(content []byte, dest any) (string, error) {
	return frontmatter.Decode(content, func(data []byte) error {
		if err := yaml.UnmarshalWithOptions(data, dest, yaml.Strict()); err != nil {
			return fmt.Errorf("decode YAML: %w", err)
		}
		return nil
	})
}
