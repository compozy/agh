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
	"sync"
	"time"
	"unicode/utf8"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/fileutil"
	"github.com/compozy/agh/internal/frontmatter"
	memcontract "github.com/compozy/agh/internal/memory/contract"
	memoryrecall "github.com/compozy/agh/internal/memory/recall"
	storepkg "github.com/compozy/agh/internal/store"
	aghworkspace "github.com/compozy/agh/internal/workspace"
	"github.com/goccy/go-yaml"
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
	globalDir        string
	workspaceDir     string
	workspaceRoot    string
	agentName        string
	agentTier        memcontract.AgentTier
	agentWorkspaceID string
	maxIndexLines    int
	maxIndexBytes    int
	logger           *slog.Logger
	catalog          *catalog
	mu               *sync.Mutex
	recallSignals    recallSignalRecorderConfig
	recallRecorders  *recallRecorderRegistry
}

var _ memcontract.Backend = (*Store)(nil)

type recallSignalRecorderConfig struct {
	queueCapacity  int
	workerRetryMax int
	metricsEnabled bool
}

type recallRecorderRegistry struct {
	mu        sync.Mutex
	recorders map[string]*memoryrecall.SignalRecorder
}

// NewStore constructs a Store for the provided global memory directory.
func NewStore(globalDir string, opts ...StoreOption) *Store {
	store := &Store{
		globalDir:     cleanDirPath(globalDir),
		maxIndexLines: defaultIndexLines,
		maxIndexBytes: defaultIndexBytes,
		logger:        slog.Default(),
		mu:            &sync.Mutex{},
		recallSignals: recallSignalRecorderConfig{
			queueCapacity:  256,
			workerRetryMax: 3,
			metricsEnabled: true,
		},
		recallRecorders: &recallRecorderRegistry{recorders: make(map[string]*memoryrecall.SignalRecorder)},
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

// WithRecallSignalRecorderConfig configures asynchronous recall-signal writes.
func WithRecallSignalRecorderConfig(config aghconfig.MemoryRecallSignalsConfig) StoreOption {
	return func(store *Store) {
		if store == nil {
			return
		}
		if config.QueueCapacity > 0 {
			store.recallSignals.queueCapacity = config.QueueCapacity
		}
		if config.WorkerRetryMax >= 0 {
			store.recallSignals.workerRetryMax = config.WorkerRetryMax
		}
		store.recallSignals.metricsEnabled = config.MetricsEnabled
	}
}

// RecallSignalRecorderStats returns per-workspace async signal recorder counters.
func (s *Store) RecallSignalRecorderStats(workspaceID string) memoryrecall.SignalRecorderStats {
	if s == nil || s.recallRecorders == nil {
		return memoryrecall.SignalRecorderStats{}
	}
	key := recallSignalRecorderKey(workspaceID)
	s.recallRecorders.mu.Lock()
	recorder := s.recallRecorders.recorders[key]
	s.recallRecorders.mu.Unlock()
	return recorder.Stats()
}

// CloseRecallSignalRecorders drains and stops every async recall-signal worker.
func (s *Store) CloseRecallSignalRecorders(ctx context.Context) error {
	if s == nil || s.recallRecorders == nil {
		return nil
	}
	s.recallRecorders.mu.Lock()
	recorders := make([]*memoryrecall.SignalRecorder, 0, len(s.recallRecorders.recorders))
	for key, recorder := range s.recallRecorders.recorders {
		recorders = append(recorders, recorder)
		delete(s.recallRecorders.recorders, key)
	}
	s.recallRecorders.mu.Unlock()
	var errs []error
	for _, recorder := range recorders {
		if err := recorder.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ListMemoryEventSummaries returns canonical memory events from every visible
// memory authority, keeping workspace DB rows visible to observe adapters.
func (s *Store) ListMemoryEventSummaries(
	ctx context.Context,
	workspaces []string,
	query storepkg.EventSummaryQuery,
) ([]storepkg.EventSummary, error) {
	if ctx == nil {
		return nil, errors.New("memory: event summary context is required")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sources, err := s.observabilitySources(ctx, workspaces)
	if err != nil {
		return nil, err
	}

	summaries := make([]storepkg.EventSummary, 0)
	for _, source := range sources {
		sourceSummaries, err := source.catalog.listEventSummaries(ctx, source.id, source.filters, query)
		if err != nil {
			return nil, fmt.Errorf("memory: list %s memory events: %w", source.id, err)
		}
		summaries = append(summaries, sourceSummaries...)
	}

	sortEventSummaries(summaries)
	return clampEventSummaries(summaries, query.Limit), nil
}

// ForWorkspace returns a clone of the store bound to the supplied workspace root.
func (s *Store) ForWorkspace(workspaceRoot string) *Store {
	clone := *s
	clone.workspaceRoot = canonicalWorkspaceRoot(workspaceRoot)
	clone.workspaceDir = workspaceMemoryDir(clone.workspaceRoot)
	return &clone
}

// ForAgent returns a clone of the store bound to one agent memory tier.
func (s *Store) ForAgent(workspaceID string, agentName string, tier memcontract.AgentTier) *Store {
	clone := *s
	clone.agentWorkspaceID = strings.TrimSpace(workspaceID)
	clone.agentName = strings.TrimSpace(agentName)
	clone.agentTier = tier.Normalize()
	return &clone
}

// List is the backend-aligned alias for Scan.
func (s *Store) List(scope memcontract.Scope) ([]memcontract.Header, error) {
	return s.Scan(scope)
}

// LoadPromptIndex is the backend-aligned alias for LoadIndex.
func (s *Store) LoadPromptIndex(scope memcontract.Scope) (content string, truncated bool, err error) {
	return s.LoadIndex(scope)
}

// EnsureDirs creates the configured memory directories when missing.
func (s *Store) EnsureDirs() error {
	dirs := []string{s.globalDir, s.workspaceDir}
	if s.agentConfigured() {
		agentDir, err := s.agentMemoryDir()
		if err != nil {
			return err
		}
		dirs = append(dirs, agentDir)
	}
	for _, dir := range dirs {
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
func (s *Store) Read(scope memcontract.Scope, filename string) ([]byte, error) {
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
func (s *Store) Exists(scope memcontract.Scope, filename string) (bool, error) {
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
func (s *Store) Write(scope memcontract.Scope, filename string, content []byte) error {
	return s.writeRaw(context.Background(), scope, filename, content, true)
}

// Delete removes a memory file and strips any matching entry from the local MEMORY.md index.
func (s *Store) Delete(scope memcontract.Scope, filename string) error {
	return s.deleteRaw(context.Background(), scope, filename, true)
}

// Scan lists memory headers for a scope, sorted newest-first and capped at 200 files.
func (s *Store) Scan(scope memcontract.Scope) ([]memcontract.Header, error) {
	return s.scan(scope, maxScanEntries)
}

func (s *Store) scan(scope memcontract.Scope, limit int) ([]memcontract.Header, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []memcontract.Header{}, nil
		}
		return nil, fmt.Errorf("memory: scan %q: %w", dir, err)
	}

	capacity := len(entries)
	if limit > 0 {
		capacity = min(capacity, limit)
	}
	headers := make([]memcontract.Header, 0, capacity)
	candidates := s.scanCandidates(scope, dir, entries)

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

		var header memcontract.Header
		if _, err := parseFrontmatter(content, &header); err != nil {
			s.warn("memory: skip malformed memory file", "scope", scope, "path", candidate.path, "error", err)
			continue
		}
		if err := header.Validate(); err != nil {
			s.warn("memory: skip invalid memory metadata", "scope", scope, "path", candidate.path, "error", err)
			continue
		}
		completedHeader, err := s.completeHeaderForScope(scope.Normalize(), header)
		if err != nil {
			s.warn(
				"memory: skip memory file with invalid scope metadata",
				"scope",
				scope,
				"path",
				candidate.path,
				"error",
				err,
			)
			continue
		}

		completedHeader.Filename = candidate.name
		completedHeader.FilePath = candidate.path
		completedHeader.ModTime = candidate.modTime
		headers = append(headers, completedHeader)

		if limit > 0 && len(headers) == limit {
			break
		}
	}

	return headers, nil
}

func (s *Store) scanCandidates(scope memcontract.Scope, dir string, entries []os.DirEntry) []scanCandidate {
	candidates := make([]scanCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || shouldSkipFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			s.warn(
				"memory: skip memory file with unreadable metadata",
				"scope", scope,
				"filename", entry.Name(),
				"error", err,
			)
			continue
		}

		candidates = append(candidates, scanCandidate{
			name:    entry.Name(),
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}
	return candidates
}

// LoadIndex reads MEMORY.md for a scope and truncates it to the prompt-safe limits.
func (s *Store) LoadIndex(scope memcontract.Scope) (content string, truncated bool, err error) {
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
func (s *Store) Search(
	ctx context.Context,
	query string,
	opts memcontract.SearchOptions,
) ([]memcontract.SearchResult, error) {
	if ctx == nil {
		return nil, errors.New("memory: search context is required")
	}

	scope, workspaceRoot, workspaceID, err := s.normalizeScopeAndWorkspace(ctx, opts.Scope, opts.Workspace)
	if err != nil {
		return nil, err
	}
	if _, err := searchQueryTerms(query); err != nil {
		return nil, err
	}
	limit := clampSearchLimit(opts.Limit)

	if err := s.ensureCatalogReady(ctx, scope, workspaceRoot, workspaceID); err != nil {
		return nil, err
	}
	if s.catalog != nil {
		results, err := s.catalog.search(ctx, query, scope, workspaceID, limit)
		if err != nil {
			return nil, err
		}
		if err := s.logCatalogEvent(
			ctx,
			memcontract.OperationRecord{
				Operation: memcontract.OperationSearch,
				Scope:     operationRecordScope(scope, workspaceID),
				Workspace: workspaceID,
				Summary:   fmt.Sprintf("query=%q results=%d", strings.TrimSpace(query), len(results)),
			},
		); err != nil {
			s.warn("memory: record search event failed", "error", err)
		}
		if len(results) > 0 {
			return results, nil
		}
	}

	docs, err := s.collectSearchDocuments(scope, workspaceRoot, workspaceID)
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
func (s *Store) Reindex(ctx context.Context, opts memcontract.ReindexOptions) (memcontract.ReindexResult, error) {
	if ctx == nil {
		return memcontract.ReindexResult{}, errors.New("memory: reindex context is required")
	}

	scope, workspaceRoot, workspaceID, err := s.normalizeScopeAndWorkspace(ctx, opts.Scope, opts.Workspace)
	if err != nil {
		return memcontract.ReindexResult{}, err
	}

	indexed, err := s.reindexScopes(ctx, scope, workspaceRoot, workspaceID)
	if err != nil {
		return memcontract.ReindexResult{}, err
	}
	completedAt := time.Now().UTC()
	if err := s.logCatalogEvent(
		ctx,
		memcontract.OperationRecord{
			Operation: memcontract.OperationReindex,
			Scope:     operationRecordScope(scope, workspaceID),
			Workspace: workspaceID,
			Summary: fmt.Sprintf(
				"scope=%s workspace=%s indexed=%d",
				string(scope.Normalize()),
				workspaceID,
				indexed,
			),
		},
	); err != nil {
		s.warn("memory: record reindex event failed", "error", err)
	}
	return memcontract.ReindexResult{
		IndexedFiles: indexed,
		Scope:        scope.Normalize(),
		Workspace:    workspaceID,
		CompletedAt:  completedAt,
	}, nil
}

// HealthStats returns derived-catalog stats for the visible scopes.
func (s *Store) HealthStats(ctx context.Context, workspaces []string) (memcontract.HealthStats, error) {
	if ctx == nil {
		return memcontract.HealthStats{}, errors.New("memory: health stats context is required")
	}
	if s.catalog == nil {
		return memcontract.HealthStats{}, nil
	}

	sources, err := s.healthSources(ctx, workspaces)
	if err != nil {
		return memcontract.HealthStats{}, err
	}

	accumulator := newHealthAccumulator()
	for _, source := range sources {
		if err := accumulator.addSource(ctx, source); err != nil {
			return memcontract.HealthStats{}, err
		}
	}
	return accumulator.stats(), nil
}

// History returns durable memory operation history ordered newest-first.
func (s *Store) History(
	ctx context.Context,
	query memcontract.OperationHistoryQuery,
) ([]memcontract.OperationRecord, error) {
	if ctx == nil {
		return nil, errors.New("memory: history context is required")
	}
	if s.catalog == nil {
		return []memcontract.OperationRecord{}, nil
	}
	normalized := query
	scope, _, workspaceID, err := s.normalizeScopeAndWorkspace(ctx, query.Scope, query.Workspace)
	if err != nil {
		return nil, err
	}
	normalized.Scope = scope
	normalized.Workspace = workspaceID
	normalized.Operation = query.Operation.Normalize()
	return s.catalog.listOperations(ctx, normalized)
}

func operationRecordScope(scope memcontract.Scope, workspaceID string) memcontract.Scope {
	normalized := scope.Normalize()
	if normalized == "" && strings.TrimSpace(workspaceID) != "" {
		return memcontract.ScopeWorkspace
	}
	return normalized
}

func (s *Store) dirForScope(scope memcontract.Scope) (string, error) {
	normalized := scope.Normalize()
	if err := normalized.Validate(); err != nil {
		return "", wrapValidationError("resolve scope", string(scope), err)
	}

	switch normalized {
	case memcontract.ScopeGlobal:
		if s.globalDir == "" {
			return "", wrapValidationError("resolve scope", string(scope), errors.New("global directory is required"))
		}
		return s.globalDir, nil
	case memcontract.ScopeWorkspace:
		if s.workspaceDir == "" {
			return "", wrapValidationError(
				"resolve scope",
				string(scope),
				errors.New("workspace directory is required"),
			)
		}
		return s.workspaceDir, nil
	case memcontract.ScopeAgent:
		return s.agentMemoryDir()
	default:
		return "", wrapValidationError("resolve scope", string(scope), fmt.Errorf("unsupported scope %q", scope))
	}
}

func (s *Store) agentConfigured() bool {
	return strings.TrimSpace(s.agentName) != "" || strings.TrimSpace(string(s.agentTier)) != ""
}

func (s *Store) agentMemoryDir() (string, error) {
	agentName, err := cleanPathSegment("agent", s.agentName)
	if err != nil {
		return "", err
	}
	tier := s.agentTier.Normalize()
	if err := tier.Validate(); err != nil {
		return "", wrapValidationError("resolve agent tier", string(s.agentTier), err)
	}
	switch tier {
	case memcontract.AgentTierGlobal:
		root, err := globalHomeFromMemoryDir(s.globalDir)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, "agents", agentName, memoryDirName), nil
	case memcontract.AgentTierWorkspace:
		if strings.TrimSpace(s.workspaceRoot) == "" {
			return "", wrapValidationError(
				"resolve agent workspace",
				agentName,
				errors.New("workspace directory is required"),
			)
		}
		return filepath.Join(s.workspaceRoot, ".agh", "agents", agentName, memoryDirName), nil
	default:
		return "", wrapValidationError(
			"resolve agent tier",
			string(s.agentTier),
			fmt.Errorf("unsupported agent tier %q", s.agentTier),
		)
	}
}

func globalHomeFromMemoryDir(globalDir string) (string, error) {
	dir := cleanDirPath(globalDir)
	if dir == "" {
		return "", wrapValidationError(
			"resolve global agent memory",
			"global",
			errors.New("global directory is required"),
		)
	}
	if filepath.Base(dir) == memoryDirName {
		return filepath.Dir(dir), nil
	}
	return filepath.Dir(dir), nil
}

func cleanPathSegment(kind string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", wrapValidationError("resolve "+kind, value, fmt.Errorf("%s is required", kind))
	}
	if strings.ContainsRune(trimmed, 0) || filepath.IsAbs(trimmed) {
		return "", wrapValidationError("resolve "+kind, value, fmt.Errorf("invalid %s path segment", kind))
	}
	if trimmed == "." || trimmed == ".." || strings.ContainsAny(trimmed, `/\`) {
		return "", wrapValidationError("resolve "+kind, value, fmt.Errorf("invalid %s path segment", kind))
	}
	return trimmed, nil
}

func (s *Store) pathFor(scope memcontract.Scope, filename string) (string, error) {
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

func (s *Store) completeHeaderForScope(
	scope memcontract.Scope,
	header memcontract.Header,
) (memcontract.Header, error) {
	normalized := scope.Normalize()
	if normalized == "" {
		return header, nil
	}
	header.Scope = normalized
	if normalized != memcontract.ScopeAgent {
		return header, nil
	}
	agentName, err := cleanPathSegment("agent", s.agentName)
	if err != nil {
		return memcontract.Header{}, err
	}
	tier := s.agentTier.Normalize()
	if err := tier.Validate(); err != nil {
		return memcontract.Header{}, wrapValidationError("resolve agent tier", string(s.agentTier), err)
	}
	if strings.TrimSpace(header.AgentName) == "" {
		header.AgentName = agentName
	} else if strings.TrimSpace(header.AgentName) != agentName {
		return memcontract.Header{}, wrapValidationError(
			"resolve agent",
			header.AgentName,
			fmt.Errorf("frontmatter agent does not match store agent %q", agentName),
		)
	}
	if header.AgentTier.Normalize() == "" {
		header.AgentTier = tier
	} else if header.AgentTier.Normalize() != tier {
		return memcontract.Header{}, wrapValidationError(
			"resolve agent tier",
			string(header.AgentTier),
			fmt.Errorf("frontmatter agent tier does not match store tier %q", tier),
		)
	}
	return header, nil
}

func (s *Store) syncScope(ctx context.Context, scope memcontract.Scope) error {
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

func (s *Store) syncIndex(scope memcontract.Scope, headers []memcontract.Header) error {
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

	if err := fileutil.AtomicWrite(indexPath, []byte(renderIndex(headers))); err != nil {
		return fmt.Errorf("memory: write index %q: %w", indexPath, err)
	}
	return nil
}

func (s *Store) syncCatalogScope(ctx context.Context, scope memcontract.Scope, headers []memcontract.Header) error {
	if s.catalog == nil {
		return nil
	}
	workspaceRoot, workspaceID, err := s.catalogWorkspaceForScope(ctx, scope)
	if err != nil {
		return err
	}
	docs, err := s.documentsForHeaders(scope, workspaceRoot, workspaceID, headers)
	if err != nil {
		return err
	}
	if err := s.catalog.replaceScope(
		ctx,
		scope,
		workspaceID,
		s.catalogAgentName(scope),
		s.catalogAgentTier(scope),
		docs,
	); err != nil {
		return err
	}
	if err := s.catalog.setLastReindex(ctx, time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (s *Store) syncScopeAfterWriteErr(
	ctx context.Context,
	scope memcontract.Scope,
	header memcontract.Header,
	content []byte,
) error {
	if needsFullSync, err := s.needsFullSyncAfterMutation(ctx, scope, strings.TrimSpace(header.Filename)); err != nil {
		return err
	} else if needsFullSync {
		return s.syncScope(ctx, scope)
	}

	if err := s.syncIndexAfterWrite(scope, header); err != nil {
		return err
	}
	if s.catalog == nil {
		return nil
	}

	_, workspaceID, err := s.catalogWorkspaceForScope(ctx, scope)
	if err != nil {
		return err
	}
	doc, err := buildCatalogDocument(scope, workspaceID, header, content)
	if err != nil {
		return err
	}
	return s.catalog.upsertDocument(ctx, doc)
}

func (s *Store) syncScopeAfterDeleteErr(ctx context.Context, scope memcontract.Scope, filename string) error {
	needsFullSync, err := s.needsFullSyncAfterMutation(ctx, scope, strings.TrimSpace(filename))
	if err != nil {
		return err
	}
	if needsFullSync {
		return s.syncScope(ctx, scope)
	}

	if err := s.syncIndexAfterDelete(scope, filename); err != nil {
		return err
	}
	if s.catalog == nil {
		return nil
	}

	_, workspaceID, err := s.catalogWorkspaceForScope(ctx, scope)
	if err != nil {
		return err
	}
	return s.catalog.deleteDocument(
		ctx,
		scope,
		workspaceID,
		s.catalogAgentName(scope),
		s.catalogAgentTier(scope),
		filename,
	)
}

func (s *Store) needsFullSyncAfterMutation(
	ctx context.Context,
	scope memcontract.Scope,
	filename string,
) (bool, error) {
	indexMissing, err := s.indexMissingWithExistingDocuments(scope, filename)
	if err != nil {
		return false, err
	}
	if indexMissing {
		return true, nil
	}
	if s.catalog == nil {
		return false, nil
	}

	_, workspaceID, err := s.catalogWorkspaceForScope(ctx, scope)
	if err != nil {
		return false, err
	}
	ready, err := s.catalog.scopeReady(ctx, scope, workspaceID)
	if err != nil {
		return false, err
	}
	return !ready, nil
}

func (s *Store) indexMissingWithExistingDocuments(scope memcontract.Scope, mutatedFilename string) (bool, error) {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return false, err
	}
	indexPath := filepath.Join(dir, indexFilename)
	if _, err := os.Stat(indexPath); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("memory: stat index %q: %w", indexPath, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("memory: inspect memory directory %q: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || shouldSkipFile(entry.Name()) || entry.Name() == strings.TrimSpace(mutatedFilename) {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (s *Store) syncIndexAfterWrite(scope memcontract.Scope, header memcontract.Header) error {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return err
	}
	indexPath := filepath.Join(dir, indexFilename)
	lines, err := readIndexLines(indexPath)
	if err != nil {
		return err
	}

	filename := strings.TrimSpace(header.Filename)
	updated := []string{renderIndexLine(header)}
	for _, line := range lines {
		target, ok := firstMarkdownLinkTarget(line)
		if ok && target == filename {
			continue
		}
		updated = append(updated, line)
	}
	return writeIndexLines(indexPath, updated)
}

func (s *Store) syncIndexAfterDelete(scope memcontract.Scope, filename string) error {
	dir, err := s.dirForScope(scope)
	if err != nil {
		return err
	}
	indexPath := filepath.Join(dir, indexFilename)
	lines, err := readIndexLines(indexPath)
	if err != nil {
		return err
	}

	targetFilename := strings.TrimSpace(filename)
	updated := make([]string, 0, len(lines))
	for _, line := range lines {
		target, ok := firstMarkdownLinkTarget(line)
		if ok && target == targetFilename {
			continue
		}
		updated = append(updated, line)
	}
	return writeIndexLines(indexPath, updated)
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

func (s *Store) normalizeScopeAndWorkspace(
	ctx context.Context,
	scope memcontract.Scope,
	workspace string,
) (memcontract.Scope, string, string, error) {
	normalizedScope := scope.Normalize()
	if normalizedScope != "" {
		if err := normalizedScope.Validate(); err != nil {
			return "", "", "", wrapValidationError("resolve scope", string(scope), err)
		}
	}

	workspaceRoot := canonicalWorkspaceRoot(workspace)
	if workspaceRoot == "" {
		workspaceRoot = strings.TrimSpace(s.workspaceRoot)
	}
	if normalizedScope == memcontract.ScopeWorkspace && workspaceRoot == "" {
		return "", "", "", wrapValidationError(
			"resolve scope",
			string(scope),
			errors.New("workspace directory is required"),
		)
	}
	if normalizedScope == memcontract.ScopeAgent && s.agentTier.Normalize() == memcontract.AgentTierWorkspace &&
		workspaceRoot == "" {
		return "", "", "", wrapValidationError(
			"resolve scope",
			string(scope),
			errors.New("workspace directory is required"),
		)
	}
	workspaceID := ""
	if workspaceRoot != "" {
		identity, err := aghworkspace.EnsureIdentity(ctx, workspaceRoot)
		if err != nil {
			return "", "", "", fmt.Errorf("memory: resolve workspace identity for %q: %w", workspaceRoot, err)
		}
		workspaceID = identity.WorkspaceID
	}
	return normalizedScope, workspaceRoot, workspaceID, nil
}

func (s *Store) workspaceIDForRoot(ctx context.Context, workspaceRoot string) (string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", wrapValidationError(
			"resolve workspace identity",
			"workspace",
			errors.New("workspace directory is required"),
		)
	}
	identity, err := aghworkspace.EnsureIdentity(ctx, root)
	if err != nil {
		return "", fmt.Errorf("memory: resolve workspace identity for %q: %w", root, err)
	}
	return identity.WorkspaceID, nil
}

func (s *Store) catalogWorkspaceForScope(
	ctx context.Context,
	scope memcontract.Scope,
) (string, string, error) {
	switch scope.Normalize() {
	case memcontract.ScopeWorkspace:
		workspaceRoot := strings.TrimSpace(s.workspaceRoot)
		workspaceID, err := s.workspaceIDForRoot(ctx, workspaceRoot)
		return workspaceRoot, workspaceID, err
	case memcontract.ScopeAgent:
		if s.agentTier.Normalize() != memcontract.AgentTierWorkspace {
			return "", "", nil
		}
		workspaceRoot := strings.TrimSpace(s.workspaceRoot)
		workspaceID := strings.TrimSpace(s.agentWorkspaceID)
		if workspaceID != "" {
			return workspaceRoot, workspaceID, nil
		}
		resolvedID, err := s.workspaceIDForRoot(ctx, workspaceRoot)
		return workspaceRoot, resolvedID, err
	default:
		return "", "", nil
	}
}

func (s *Store) catalogAgentName(scope memcontract.Scope) string {
	if scope.Normalize() != memcontract.ScopeAgent {
		return ""
	}
	return strings.TrimSpace(s.agentName)
}

func (s *Store) catalogAgentTier(scope memcontract.Scope) memcontract.AgentTier {
	if scope.Normalize() != memcontract.ScopeAgent {
		return ""
	}
	return s.agentTier.Normalize()
}

func (s *Store) ensureCatalogReady(
	ctx context.Context,
	scope memcontract.Scope,
	workspaceRoot string,
	workspaceID string,
) error {
	if s.catalog == nil {
		return nil
	}

	filters := []catalogFilter{{scope: memcontract.ScopeGlobal}}
	switch scope.Normalize() {
	case memcontract.ScopeGlobal:
		filters = filters[:1]
	case memcontract.ScopeWorkspace:
		filters = []catalogFilter{{
			scope:         memcontract.ScopeWorkspace,
			workspaceRoot: workspaceRoot,
			workspaceID:   workspaceID,
		}}
	case memcontract.ScopeAgent:
		filters = []catalogFilter{{
			scope:         memcontract.ScopeAgent,
			workspaceRoot: workspaceRoot,
			workspaceID:   workspaceID,
		}}
	default:
		if strings.TrimSpace(workspaceID) != "" {
			filters = append(filters, catalogFilter{
				scope:         memcontract.ScopeWorkspace,
				workspaceRoot: workspaceRoot,
				workspaceID:   workspaceID,
			})
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

	ready, err := s.catalog.scopeReady(ctx, filter.scope, filter.workspaceID)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}

	entryCount, err := s.catalog.scopeEntryCount(ctx, filter.scope, filter.workspaceID)
	if err != nil {
		return err
	}
	if entryCount > 0 {
		return s.catalog.setScopeReady(ctx, filter.scope, filter.workspaceID)
	}

	_, err = s.reindexScopes(ctx, filter.scope, filter.workspaceRoot, filter.workspaceID)
	return err
}

func (s *Store) reindexScopes(
	ctx context.Context,
	scope memcontract.Scope,
	workspaceRoot string,
	workspaceID string,
) (int, error) {
	if s.catalog == nil {
		return 0, nil
	}

	total := 0
	seenWorkspaceRoot := strings.TrimSpace(workspaceRoot)
	seenWorkspaceID := strings.TrimSpace(workspaceID)

	reindexScope := func(scope memcontract.Scope, workspaceRoot string, workspaceID string) error {
		headers, err := s.headersForCatalogScope(scope, workspaceRoot)
		if err != nil {
			return err
		}
		docs, err := s.documentsForHeaders(scope, workspaceRoot, workspaceID, headers)
		if err != nil {
			return err
		}
		if err := s.catalog.replaceScope(
			ctx,
			scope,
			workspaceID,
			s.catalogAgentName(scope),
			s.catalogAgentTier(scope),
			docs,
		); err != nil {
			return err
		}
		total += len(docs)
		return nil
	}

	switch scope.Normalize() {
	case memcontract.ScopeGlobal:
		if err := reindexScope(memcontract.ScopeGlobal, "", ""); err != nil {
			return 0, err
		}
	case memcontract.ScopeWorkspace:
		if err := reindexScope(memcontract.ScopeWorkspace, seenWorkspaceRoot, seenWorkspaceID); err != nil {
			return 0, err
		}
	case memcontract.ScopeAgent:
		if err := reindexScope(memcontract.ScopeAgent, seenWorkspaceRoot, seenWorkspaceID); err != nil {
			return 0, err
		}
	default:
		if err := reindexScope(memcontract.ScopeGlobal, "", ""); err != nil {
			return 0, err
		}
		if seenWorkspaceRoot != "" {
			if err := reindexScope(memcontract.ScopeWorkspace, seenWorkspaceRoot, seenWorkspaceID); err != nil {
				return 0, err
			}
		}
	}

	if err := s.catalog.setLastReindex(ctx, time.Now().UTC()); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) headersForCatalogScope(scope memcontract.Scope, workspaceRoot string) ([]memcontract.Header, error) {
	target := s
	if scope.Normalize() == memcontract.ScopeWorkspace {
		target = s.ForWorkspace(workspaceRoot)
	}
	return target.scan(scope, 0)
}

func (s *Store) documentsForHeaders(
	scope memcontract.Scope,
	workspaceRoot string,
	workspaceID string,
	headers []memcontract.Header,
) ([]catalogDocument, error) {
	target := s
	if scope.Normalize() == memcontract.ScopeWorkspace {
		target = s.ForWorkspace(workspaceRoot)
	}

	docs := make([]catalogDocument, 0, len(headers))
	for _, header := range headers {
		rawContent, err := target.Read(scope, header.Filename)
		if err != nil {
			return nil, err
		}
		doc, err := buildCatalogDocument(scope, workspaceID, header, rawContent)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (s *Store) collectSearchDocuments(
	scope memcontract.Scope,
	workspaceRoot string,
	workspaceID string,
) ([]catalogDocument, error) {
	scopes := []struct {
		scope       memcontract.Scope
		workspace   string
		workspaceID string
	}{{scope: memcontract.ScopeGlobal}}
	switch scope.Normalize() {
	case memcontract.ScopeWorkspace:
		scopes = []struct {
			scope       memcontract.Scope
			workspace   string
			workspaceID string
		}{{scope: memcontract.ScopeWorkspace, workspace: workspaceRoot, workspaceID: workspaceID}}
	case memcontract.ScopeAgent:
		scopes = []struct {
			scope       memcontract.Scope
			workspace   string
			workspaceID string
		}{{scope: memcontract.ScopeAgent, workspace: workspaceRoot, workspaceID: workspaceID}}
	default:
		if strings.TrimSpace(workspaceRoot) != "" {
			scopes = append(scopes, struct {
				scope       memcontract.Scope
				workspace   string
				workspaceID string
			}{scope: memcontract.ScopeWorkspace, workspace: workspaceRoot, workspaceID: workspaceID})
		}
	}

	docs := make([]catalogDocument, 0)
	for _, item := range scopes {
		headers, err := s.headersForCatalogScope(item.scope, item.workspace)
		if err != nil {
			return nil, err
		}
		items, err := s.documentsForHeaders(item.scope, item.workspace, item.workspaceID, headers)
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
			actual[catalogDocIDForHeader(filter.scope, filter.workspaceID, header)] = struct{}{}
		}
	}
	return actual, nil
}

func (s *Store) logCatalogEvent(ctx context.Context, record memcontract.OperationRecord) error {
	if s.catalog == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return s.catalog.logEvent(ctx, record)
}

func (s *Store) logMutationEvent(action string, scope memcontract.Scope, filename string) {
	_, workspaceID, err := s.catalogWorkspaceForScope(context.Background(), scope)
	if err != nil {
		s.warn("memory: resolve workspace identity for mutation event failed", "error", err)
		return
	}

	if err := s.logCatalogEvent(
		context.Background(),
		memcontract.OperationRecord{
			Operation: memcontract.Operation("memory." + strings.TrimSpace(action)),
			Scope:     scope.Normalize(),
			Workspace: workspaceID,
			Filename:  strings.TrimSpace(filename),
			AgentName: s.agentNameForEvent(scope),
			Summary:   fmt.Sprintf("scope=%s filename=%s", scope.Normalize(), strings.TrimSpace(filename)),
		},
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

func (s *Store) agentNameForEvent(scope memcontract.Scope) string {
	if scope.Normalize() != memcontract.ScopeAgent {
		return ""
	}
	return strings.TrimSpace(s.agentName)
}

func (s *Store) lockMutations() func() {
	if s == nil || s.mu == nil {
		return func() {}
	}
	s.mu.Lock()
	return s.mu.Unlock
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

func renderIndex(headers []memcontract.Header) string {
	if len(headers) == 0 {
		return ""
	}
	lines := make([]string, 0, len(headers))
	for _, header := range headers {
		lines = append(lines, renderIndexLine(header))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderIndexLine(header memcontract.Header) string {
	header.Normalize()
	if header.Description == "" {
		return fmt.Sprintf("- [%s](%s)", header.Name, header.Filename)
	}
	return fmt.Sprintf("- [%s](%s) - %s", header.Name, header.Filename, header.Description)
}

func readIndexLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("memory: read index %q: %w", path, err)
	}

	lines := make([]string, 0)
	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines, nil
}

func writeIndexLines(path string, lines []string) error {
	if len(lines) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("memory: remove empty index %q: %w", path, err)
		}
		return nil
	}
	if err := fileutil.AtomicWrite(path, []byte(strings.Join(lines, "\n")+"\n")); err != nil {
		return fmt.Errorf("memory: write index %q: %w", path, err)
	}
	return nil
}

func indexMatchesHeaders(content string, headers []memcontract.Header) bool {
	return strings.TrimSpace(content) == strings.TrimSpace(renderIndex(headers))
}

func shouldSkipFile(name string) bool {
	return name == indexFilename || strings.HasPrefix(name, ".") || strings.Contains(name, ".tmp-")
}

func cleanDirPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	return filepath.Clean(trimmed)
}

func canonicalWorkspaceRoot(path string) string {
	clean := cleanDirPath(path)
	if clean == "" {
		return ""
	}
	return clean
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
