package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/filesnap"
)

// RegisterOptions describes a workspace registration request.
type RegisterOptions struct {
	RootDir        string
	Name           string
	AdditionalDirs []string
	DefaultAgent   string
}

// UpdateOptions describes mutable workspace registration fields.
type UpdateOptions struct {
	Name           *string
	AdditionalDirs *[]string
	DefaultAgent   *string
}

// Resolver resolves persisted workspaces into runtime workspace snapshots.
type Resolver struct {
	store       WorkspaceStore
	homePaths   aghconfig.HomePaths
	loadConfig  ConfigLoader
	logger      *slog.Logger
	now         func() time.Time
	cacheTTL    time.Duration
	idGenerator func(prefix string) string

	mu    sync.RWMutex
	cache map[string]*cachedEntry
}

var _ WorkspaceResolver = (*Resolver)(nil)

type cachedEntry struct {
	workspace  Workspace
	resolved   ResolvedWorkspace
	snapshots  map[string]filesnap.Snapshot
	lastAccess time.Time
}

// NewResolver constructs a workspace resolver backed by the supplied store.
func NewResolver(store WorkspaceStore, opts ...Option) (*Resolver, error) {
	if store == nil {
		return nil, errors.New("workspace: store is required")
	}

	resolvedOpts, err := resolveOptions(opts)
	if err != nil {
		return nil, err
	}

	return &Resolver{
		store:       store,
		homePaths:   resolvedOpts.homePaths,
		loadConfig:  resolvedOpts.loadConfig,
		logger:      resolvedOpts.logger,
		now:         resolvedOpts.now,
		cacheTTL:    resolvedOpts.cacheTTL,
		idGenerator: resolvedOpts.idGenerator,
		cache:       make(map[string]*cachedEntry),
	}, nil
}

// Resolve loads and caches the effective runtime snapshot for a workspace.
func (r *Resolver) Resolve(ctx context.Context, idOrNameOrPath string) (resolved ResolvedWorkspace, err error) {
	start := r.now()
	cacheHit := false
	workspaceID := ""

	defer func() {
		if err == nil {
			r.logger.Debug("workspace.resolve",
				"workspace_id", workspaceID,
				"cache_hit", cacheHit,
				"agents_count", len(resolved.Agents),
				"skills_count", len(resolved.Skills),
				"duration_ms", durationMillis(r.now().Sub(start)),
			)
			return
		}

		r.logger.Warn("workspace.resolve.error",
			"workspace_id", workspaceID,
			"error_type", errorType(err),
			"duration_ms", durationMillis(r.now().Sub(start)),
			"error", err,
		)
	}()

	if err := checkContext(ctx); err != nil {
		return ResolvedWorkspace{}, err
	}

	ws, err := r.lookupWorkspace(ctx, idOrNameOrPath)
	if err != nil {
		return ResolvedWorkspace{}, err
	}
	workspaceID = ws.ID

	ws, err = r.refreshRootDir(ctx, ws)
	if err != nil {
		return ResolvedWorkspace{}, err
	}
	workspaceID = ws.ID

	scan, err := r.scanWorkspace(ctx, ws)
	if err != nil {
		return ResolvedWorkspace{}, err
	}

	now := r.now()

	r.mu.Lock()
	r.evictExpiredLocked(now)
	if cached := r.cache[ws.ID]; cached != nil && cached.canReuse(ws, scan.snapshots) {
		cached.lastAccess = now
		cacheHit = true
		resolved = cloneResolvedWorkspace(cached.resolved)
		resolved.Workspace = cloneWorkspace(ws)
		r.mu.Unlock()
		return resolved, nil
	}
	r.mu.Unlock()

	resolved, err = r.buildResolvedWorkspace(ctx, ws, scan)
	if err != nil {
		return ResolvedWorkspace{}, err
	}

	r.mu.Lock()
	r.evictExpiredLocked(now)
	r.cache[ws.ID] = &cachedEntry{
		workspace:  cloneWorkspace(ws),
		resolved:   cloneResolvedWorkspace(resolved),
		snapshots:  cloneSnapshots(scan.snapshots),
		lastAccess: now,
	}
	r.mu.Unlock()

	return resolved, nil
}

// ResolveOrRegister resolves an existing workspace by canonical path or auto-registers it.
func (r *Resolver) ResolveOrRegister(ctx context.Context, path string) (ResolvedWorkspace, error) {
	if err := checkContext(ctx); err != nil {
		return ResolvedWorkspace{}, err
	}

	canonicalRoot, err := canonicalRoot(path)
	if err != nil {
		return ResolvedWorkspace{}, err
	}

	ws, err := r.store.GetWorkspaceByPath(ctx, canonicalRoot)
	if err == nil {
		return r.Resolve(ctx, ws.ID)
	}
	if !errors.Is(err, ErrWorkspaceNotFound) {
		return ResolvedWorkspace{}, fmt.Errorf("workspace: lookup workspace by path %q: %w", canonicalRoot, err)
	}

	ws, err = r.createWorkspaceRegistration(ctx, RegisterOptions{RootDir: canonicalRoot})
	if err != nil {
		if errors.Is(err, ErrWorkspacePathTaken) {
			existing, lookupErr := r.store.GetWorkspaceByPath(ctx, canonicalRoot)
			if lookupErr != nil {
				return ResolvedWorkspace{}, fmt.Errorf("workspace: reload concurrent workspace registration for %q: %w", canonicalRoot, lookupErr)
			}
			return r.Resolve(ctx, existing.ID)
		}
		return ResolvedWorkspace{}, err
	}

	resolved, err := r.Resolve(ctx, ws.ID)
	if err != nil {
		deleteErr := r.store.DeleteWorkspace(ctx, ws.ID)
		if deleteErr != nil && !errors.Is(deleteErr, ErrWorkspaceNotFound) {
			return ResolvedWorkspace{}, errors.Join(err, fmt.Errorf("workspace: rollback auto-registered workspace %q: %w", ws.ID, deleteErr))
		}
		return ResolvedWorkspace{}, err
	}

	r.logger.Info("workspace.register",
		"workspace_id", resolved.ID,
		"root_dir", resolved.RootDir,
		"name", resolved.Name,
	)

	return resolved, nil
}

// Invalidate deletes one workspace snapshot from the in-memory cache.
func (r *Resolver) Invalidate(workspaceID string) {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID == "" {
		return
	}

	r.mu.Lock()
	delete(r.cache, trimmedID)
	r.mu.Unlock()
}

func (r *Resolver) buildResolvedWorkspace(ctx context.Context, ws Workspace, scan workspaceScan) (ResolvedWorkspace, error) {
	if err := checkContext(ctx); err != nil {
		return ResolvedWorkspace{}, err
	}

	cfg, err := r.loadConfig(ws.RootDir)
	if err != nil {
		return ResolvedWorkspace{}, fmt.Errorf("workspace: load config for %q: %w", ws.RootDir, err)
	}
	applyDefaultAgentOverride(&cfg, ws.DefaultAgent)

	agents, err := loadAgents(ctx, scan.agents)
	if err != nil {
		return ResolvedWorkspace{}, err
	}

	skills := mergeSkillPaths(scan.skills)

	return ResolvedWorkspace{
		Workspace:  cloneWorkspace(ws),
		Config:     cloneConfig(cfg),
		Agents:     cloneAgentDefs(agents),
		Skills:     cloneSkillPaths(skills),
		ResolvedAt: r.now(),
	}, nil
}

func (c *cachedEntry) canReuse(ws Workspace, snapshots map[string]filesnap.Snapshot) bool {
	if c == nil {
		return false
	}
	if !filesnap.Equal(c.snapshots, snapshots) {
		return false
	}
	if strings.TrimSpace(c.workspace.DefaultAgent) != strings.TrimSpace(ws.DefaultAgent) {
		return false
	}
	if strings.TrimSpace(c.workspace.RootDir) != strings.TrimSpace(ws.RootDir) {
		return false
	}
	if !slices.Equal(c.workspace.AdditionalDirs, ws.AdditionalDirs) {
		return false
	}

	return true
}

func (r *Resolver) evictExpiredLocked(now time.Time) {
	cutoff := now.Add(-r.cacheTTL)
	for workspaceID, entry := range r.cache {
		if entry.lastAccess.Before(cutoff) {
			r.logger.Debug("workspace.cache.evict",
				"workspace_id", workspaceID,
				"age_minutes", int(now.Sub(entry.lastAccess).Minutes()),
			)
			delete(r.cache, workspaceID)
		}
	}
}
