package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	agentDefinitionFile = "AGENT.md"
	skillDefinitionFile = "SKILL.md"
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
	snapshots  map[string]fileSnapshot
	lastAccess time.Time
}

type fileSnapshot struct {
	modTime time.Time
	size    int64
}

type workspaceScan struct {
	snapshots map[string]fileSnapshot
	agents    []agentCandidate
	skills    []skillCandidate
}

type agentCandidate struct {
	path string
}

type skillCandidate struct {
	name   string
	dir    string
	source string
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

// Register persists a workspace registration hint and eagerly resolves it.
func (r *Resolver) Register(ctx context.Context, opts RegisterOptions) (Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return Workspace{}, err
	}

	ws, err := r.createWorkspaceRegistration(ctx, opts)
	if err != nil {
		return Workspace{}, err
	}

	resolved, err := r.Resolve(ctx, ws.ID)
	if err != nil {
		deleteErr := r.store.DeleteWorkspace(ctx, ws.ID)
		if deleteErr != nil && !errors.Is(deleteErr, ErrWorkspaceNotFound) {
			return Workspace{}, errors.Join(err, fmt.Errorf("workspace: rollback workspace registration %q: %w", ws.ID, deleteErr))
		}
		return Workspace{}, err
	}

	r.logger.Info("workspace.register",
		"workspace_id", resolved.ID,
		"root_dir", resolved.RootDir,
		"name", resolved.Name,
	)

	return resolved.Workspace, nil
}

// Unregister removes a persisted workspace registration and its cached snapshot.
func (r *Resolver) Unregister(ctx context.Context, id string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("workspace: workspace id is required")
	}

	if err := r.store.DeleteWorkspace(ctx, trimmedID); err != nil {
		return fmt.Errorf("workspace: unregister %q: %w", trimmedID, err)
	}

	r.Invalidate(trimmedID)
	return nil
}

// Update mutates an existing workspace registration.
func (r *Resolver) Update(ctx context.Context, id string, opts UpdateOptions) error {
	if err := checkContext(ctx); err != nil {
		return err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return errors.New("workspace: workspace id is required")
	}

	ws, err := r.store.GetWorkspace(ctx, trimmedID)
	if err != nil {
		return fmt.Errorf("workspace: load workspace %q: %w", trimmedID, err)
	}

	if opts.Name != nil {
		name := strings.TrimSpace(*opts.Name)
		if name == "" {
			return errors.New("workspace: workspace name is required")
		}
		ws.Name = name
	}

	if opts.AdditionalDirs != nil {
		additionalDirs, normalizeErr := normalizeAdditionalDirs(ws.RootDir, *opts.AdditionalDirs)
		if normalizeErr != nil {
			return normalizeErr
		}
		ws.AdditionalDirs = additionalDirs
	}

	if opts.DefaultAgent != nil {
		ws.DefaultAgent = strings.TrimSpace(*opts.DefaultAgent)
	}

	ws.UpdatedAt = r.now()
	if err := r.store.UpdateWorkspace(ctx, ws); err != nil {
		return fmt.Errorf("workspace: update workspace %q: %w", trimmedID, err)
	}

	r.Invalidate(trimmedID)
	return nil
}

// List returns every registered workspace in stable store order.
func (r *Resolver) List(ctx context.Context) ([]Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	workspaces, err := r.store.ListWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("workspace: list workspaces: %w", err)
	}

	return cloneWorkspaces(workspaces), nil
}

// Get resolves a persisted workspace registration without computing a full snapshot.
func (r *Resolver) Get(ctx context.Context, idOrNameOrPath string) (Workspace, error) {
	if err := checkContext(ctx); err != nil {
		return Workspace{}, err
	}

	ws, err := r.lookupWorkspace(ctx, idOrNameOrPath)
	if err != nil {
		return Workspace{}, err
	}

	return cloneWorkspace(ws), nil
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

func (r *Resolver) createWorkspaceRegistration(ctx context.Context, opts RegisterOptions) (Workspace, error) {
	rootDir, err := canonicalRoot(opts.RootDir)
	if err != nil {
		return Workspace{}, err
	}

	additionalDirs, err := normalizeAdditionalDirs(rootDir, opts.AdditionalDirs)
	if err != nil {
		return Workspace{}, err
	}

	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name, err = r.nextWorkspaceName(ctx, rootDir)
		if err != nil {
			return Workspace{}, err
		}
	}

	now := r.now()
	ws := Workspace{
		ID:             r.idGenerator("ws"),
		RootDir:        rootDir,
		AdditionalDirs: additionalDirs,
		Name:           name,
		DefaultAgent:   strings.TrimSpace(opts.DefaultAgent),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	for {
		if err := checkContext(ctx); err != nil {
			return Workspace{}, err
		}

		if insertErr := r.store.InsertWorkspace(ctx, ws); insertErr != nil {
			switch {
			case errors.Is(insertErr, ErrWorkspaceNameTaken) && strings.TrimSpace(opts.Name) == "":
				name, err = r.nextWorkspaceName(ctx, rootDir)
				if err != nil {
					return Workspace{}, err
				}
				ws.Name = name
				ws.ID = r.idGenerator("ws")
				ws.CreatedAt = now
				ws.UpdatedAt = now
				continue
			default:
				return Workspace{}, fmt.Errorf("workspace: register workspace %q: %w", rootDir, insertErr)
			}
		}

		return ws, nil
	}
}

func (r *Resolver) nextWorkspaceName(ctx context.Context, rootDir string) (string, error) {
	workspaces, err := r.store.ListWorkspaces(ctx)
	if err != nil {
		return "", fmt.Errorf("workspace: list workspaces for name dedup: %w", err)
	}

	taken := make(map[string]struct{}, len(workspaces))
	for _, ws := range workspaces {
		if name := strings.TrimSpace(ws.Name); name != "" {
			taken[name] = struct{}{}
		}
	}

	return uniqueWorkspaceName(rootDir, taken), nil
}

func (r *Resolver) lookupWorkspace(ctx context.Context, idOrNameOrPath string) (Workspace, error) {
	target := strings.TrimSpace(idOrNameOrPath)
	if target == "" {
		return Workspace{}, errors.New("workspace: workspace identifier is required")
	}

	switch {
	case strings.HasPrefix(target, "ws_"), strings.HasPrefix(target, "ws-"):
		ws, err := r.store.GetWorkspace(ctx, target)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace %q: %w", target, err)
		}
		return ws, nil
	case filepath.IsAbs(target):
		canonicalPath, err := canonicalRoot(target)
		if err != nil {
			return Workspace{}, err
		}
		ws, err := r.store.GetWorkspaceByPath(ctx, canonicalPath)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace by path %q: %w", canonicalPath, err)
		}
		return ws, nil
	default:
		ws, err := r.store.GetWorkspaceByName(ctx, target)
		if err != nil {
			return Workspace{}, fmt.Errorf("workspace: lookup workspace by name %q: %w", target, err)
		}
		return ws, nil
	}
}

func (r *Resolver) refreshRootDir(ctx context.Context, ws Workspace) (Workspace, error) {
	rootDir := strings.TrimSpace(ws.RootDir)
	if rootDir == "" {
		return Workspace{}, errors.New("workspace: workspace root directory is required")
	}

	info, err := os.Stat(rootDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, ErrWorkspaceRootMissing
		}
		return Workspace{}, fmt.Errorf("workspace: stat workspace root %q: %w", rootDir, err)
	}
	if !info.IsDir() {
		return Workspace{}, fmt.Errorf("workspace: workspace root %q is not a directory", rootDir)
	}

	canonicalDir, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Workspace{}, ErrWorkspaceRootMissing
		}
		return Workspace{}, fmt.Errorf("workspace: evaluate workspace root %q: %w", rootDir, err)
	}
	canonicalDir, err = filepath.Abs(canonicalDir)
	if err != nil {
		return Workspace{}, fmt.Errorf("workspace: resolve workspace root %q: %w", canonicalDir, err)
	}

	if canonicalDir == rootDir {
		return ws, nil
	}

	updated := cloneWorkspace(ws)
	updated.RootDir = canonicalDir
	updated.UpdatedAt = r.now()
	if err := r.store.UpdateWorkspace(ctx, updated); err != nil {
		return Workspace{}, fmt.Errorf("workspace: update canonical workspace root %q: %w", canonicalDir, err)
	}

	r.Invalidate(updated.ID)
	return updated, nil
}

func (r *Resolver) scanWorkspace(ctx context.Context, ws Workspace) (workspaceScan, error) {
	if err := checkContext(ctx); err != nil {
		return workspaceScan{}, err
	}

	scan := workspaceScan{
		snapshots: make(map[string]fileSnapshot),
		agents:    make([]agentCandidate, 0),
		skills:    make([]skillCandidate, 0),
	}

	if err := addSnapshotIfExists(r.homePaths.ConfigFile, scan.snapshots); err != nil {
		return workspaceScan{}, fmt.Errorf("workspace: snapshot global config %q: %w", r.homePaths.ConfigFile, err)
	}
	if err := addSnapshotIfExists(filepath.Join(ws.RootDir, aghconfig.DirName, aghconfig.ConfigName), scan.snapshots); err != nil {
		return workspaceScan{}, fmt.Errorf("workspace: snapshot workspace config %q: %w", ws.RootDir, err)
	}

	for _, root := range aghconfig.WorkspaceDiscoveryRoots(ws.RootDir, ws.AdditionalDirs, r.homePaths) {
		if err := checkContext(ctx); err != nil {
			return workspaceScan{}, err
		}

		if err := scanAgentSource(root, scan.snapshots, &scan.agents); err != nil {
			return workspaceScan{}, err
		}
		if err := scanSkillSource(root, scan.snapshots, &scan.skills); err != nil {
			return workspaceScan{}, err
		}
	}

	return scan, nil
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

func (c *cachedEntry) canReuse(ws Workspace, snapshots map[string]fileSnapshot) bool {
	if c == nil {
		return false
	}
	if !snapshotsEqual(c.snapshots, snapshots) {
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

func scanAgentSource(root aghconfig.WorkspaceDiscoveryRoot, snapshots map[string]fileSnapshot, dst *[]agentCandidate) error {
	agentsDir := root.AgentsDir()
	if err := addSnapshotIfExists(agentsDir, snapshots); err != nil {
		return fmt.Errorf("workspace: snapshot agents directory %q: %w", agentsDir, err)
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("workspace: read agents directory %q: %w", agentsDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		agentPath := filepath.Join(agentsDir, entry.Name(), agentDefinitionFile)
		if err := addSnapshotIfExists(agentPath, snapshots); err != nil {
			return fmt.Errorf("workspace: snapshot agent definition %q: %w", agentPath, err)
		}
		if _, ok := snapshots[agentPath]; !ok {
			continue
		}

		*dst = append(*dst, agentCandidate{
			path: agentPath,
		})
	}

	return nil
}

func scanSkillSource(root aghconfig.WorkspaceDiscoveryRoot, snapshots map[string]fileSnapshot, dst *[]skillCandidate) error {
	skillsDir := root.SkillsDir()
	if err := addSnapshotIfExists(skillsDir, snapshots); err != nil {
		return fmt.Errorf("workspace: snapshot skills directory %q: %w", skillsDir, err)
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("workspace: read skills directory %q: %w", skillsDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsDir, entry.Name())
		skillFile := filepath.Join(skillDir, skillDefinitionFile)
		if err := addSnapshotIfExists(skillDir, snapshots); err != nil {
			return fmt.Errorf("workspace: snapshot skill directory %q: %w", skillDir, err)
		}
		if err := addSnapshotIfExists(skillFile, snapshots); err != nil {
			return fmt.Errorf("workspace: snapshot skill definition %q: %w", skillFile, err)
		}
		if _, ok := snapshots[skillFile]; !ok {
			continue
		}

		*dst = append(*dst, skillCandidate{
			name:   entry.Name(),
			dir:    skillDir,
			source: string(root.Source),
		})
	}

	return nil
}

func loadAgents(ctx context.Context, candidates []agentCandidate) ([]aghconfig.AgentDef, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	agents := make([]aghconfig.AgentDef, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}

		agent, err := aghconfig.LoadAgentDefFile(candidate.path)
		if err != nil {
			return nil, fmt.Errorf("workspace: load agent definition %q: %w", candidate.path, err)
		}

		if _, ok := seen[agent.Name]; ok {
			continue
		}

		seen[agent.Name] = struct{}{}
		agents = append(agents, agent)
	}

	return agents, nil
}

func mergeSkillPaths(candidates []skillCandidate) []SkillPath {
	if len(candidates) == 0 {
		return nil
	}

	skills := make([]SkillPath, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if _, ok := seen[candidate.name]; ok {
			continue
		}

		seen[candidate.name] = struct{}{}
		skills = append(skills, SkillPath{
			Dir:    candidate.dir,
			Source: candidate.source,
		})
	}

	return skills
}

func applyDefaultAgentOverride(cfg *aghconfig.Config, defaultAgent string) {
	if cfg == nil {
		return
	}
	if trimmed := strings.TrimSpace(defaultAgent); trimmed != "" {
		cfg.Defaults.Agent = trimmed
	}
}

func canonicalRoot(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("workspace: workspace root directory is required")
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("workspace: resolve workspace root %q: %w", trimmed, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrWorkspaceRootMissing
		}
		return "", fmt.Errorf("workspace: stat workspace root %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace: workspace root %q is not a directory", absPath)
	}

	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrWorkspaceRootMissing
		}
		return "", fmt.Errorf("workspace: evaluate workspace root %q: %w", absPath, err)
	}

	canonicalPath, err = filepath.Abs(canonicalPath)
	if err != nil {
		return "", fmt.Errorf("workspace: resolve canonical workspace root %q: %w", canonicalPath, err)
	}

	return canonicalPath, nil
}

func normalizeAdditionalDirs(rootDir string, dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		return nil, nil
	}

	trimmedRoot := strings.TrimSpace(rootDir)
	normalized := make([]string, 0, len(dirs))
	seen := make(map[string]struct{}, len(dirs))

	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}

		canonicalDir, err := canonicalRoot(trimmed)
		if err != nil {
			return nil, fmt.Errorf("workspace: normalize additional directory %q: %w", trimmed, err)
		}

		if _, ok := seen[canonicalDir]; ok {
			continue
		}
		if trimmedRoot != "" && canonicalDir == trimmedRoot {
			continue
		}

		seen[canonicalDir] = struct{}{}
		normalized = append(normalized, canonicalDir)
	}

	return normalized, nil
}

func addSnapshotIfExists(path string, snapshots map[string]fileSnapshot) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	snapshot, err := snapshotPath(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	snapshots[path] = snapshot
	return nil
}

func snapshotPath(path string) (fileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}, err
	}

	return fileSnapshot{
		modTime: info.ModTime(),
		size:    info.Size(),
	}, nil
}

func snapshotsEqual(left, right map[string]fileSnapshot) bool {
	if len(left) != len(right) {
		return false
	}

	for path, leftSnapshot := range left {
		rightSnapshot, ok := right[path]
		if !ok {
			return false
		}
		if leftSnapshot.size != rightSnapshot.size {
			return false
		}
		if !leftSnapshot.modTime.Equal(rightSnapshot.modTime) {
			return false
		}
	}

	return true
}

func cloneSnapshots(snapshots map[string]fileSnapshot) map[string]fileSnapshot {
	if len(snapshots) == 0 {
		return map[string]fileSnapshot{}
	}

	cloned := make(map[string]fileSnapshot, len(snapshots))
	for path, snapshot := range snapshots {
		cloned[path] = snapshot
	}
	return cloned
}

func cloneResolvedWorkspace(src ResolvedWorkspace) ResolvedWorkspace {
	return ResolvedWorkspace{
		Workspace:  cloneWorkspace(src.Workspace),
		Config:     cloneConfig(src.Config),
		Agents:     cloneAgentDefs(src.Agents),
		Skills:     cloneSkillPaths(src.Skills),
		ResolvedAt: src.ResolvedAt,
	}
}

func cloneWorkspace(src Workspace) Workspace {
	return Workspace{
		ID:             src.ID,
		RootDir:        src.RootDir,
		AdditionalDirs: append([]string(nil), src.AdditionalDirs...),
		Name:           src.Name,
		DefaultAgent:   src.DefaultAgent,
		CreatedAt:      src.CreatedAt,
		UpdatedAt:      src.UpdatedAt,
	}
}

func cloneWorkspaces(src []Workspace) []Workspace {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]Workspace, 0, len(src))
	for _, ws := range src {
		cloned = append(cloned, cloneWorkspace(ws))
	}
	return cloned
}

func cloneConfig(src aghconfig.Config) aghconfig.Config {
	return aghconfig.Config{
		Daemon:        src.Daemon,
		HTTP:          src.HTTP,
		Defaults:      src.Defaults,
		Limits:        src.Limits,
		Permissions:   src.Permissions,
		Providers:     cloneProviders(src.Providers),
		Observability: src.Observability,
		Log:           src.Log,
		Memory:        src.Memory,
		Skills: aghconfig.SkillsConfig{
			Enabled:        src.Skills.Enabled,
			DisabledSkills: append([]string(nil), src.Skills.DisabledSkills...),
			PollInterval:   src.Skills.PollInterval,
		},
	}
}

func cloneProviders(src map[string]aghconfig.ProviderConfig) map[string]aghconfig.ProviderConfig {
	if len(src) == 0 {
		return map[string]aghconfig.ProviderConfig{}
	}

	cloned := make(map[string]aghconfig.ProviderConfig, len(src))
	for name, provider := range src {
		cloned[name] = cloneProvider(provider)
	}
	return cloned
}

func cloneProvider(src aghconfig.ProviderConfig) aghconfig.ProviderConfig {
	return aghconfig.ProviderConfig{
		Command:      src.Command,
		DefaultModel: src.DefaultModel,
		APIKeyEnv:    src.APIKeyEnv,
		MCPServers:   cloneMCPServers(src.MCPServers),
	}
}

func cloneAgentDefs(src []aghconfig.AgentDef) []aghconfig.AgentDef {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]aghconfig.AgentDef, 0, len(src))
	for _, agent := range src {
		cloned = append(cloned, aghconfig.AgentDef{
			Name:        agent.Name,
			Provider:    agent.Provider,
			Command:     agent.Command,
			Model:       agent.Model,
			Tools:       append([]string(nil), agent.Tools...),
			Permissions: agent.Permissions,
			MCPServers:  cloneMCPServers(agent.MCPServers),
			Prompt:      agent.Prompt,
		})
	}

	return cloned
}

func cloneSkillPaths(src []SkillPath) []SkillPath {
	if len(src) == 0 {
		return nil
	}

	return append([]SkillPath(nil), src...)
}

func cloneMCPServers(src []aghconfig.MCPServer) []aghconfig.MCPServer {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]aghconfig.MCPServer, 0, len(src))
	for _, server := range src {
		cloned = append(cloned, aghconfig.MCPServer{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     cloneStringMap(server.Env),
		})
	}

	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func uniqueWorkspaceName(rootDir string, taken map[string]struct{}) string {
	baseName := filepath.Base(filepath.Clean(strings.TrimSpace(rootDir)))
	switch baseName {
	case "", ".", string(filepath.Separator):
		baseName = "workspace"
	}

	candidate := baseName
	for suffix := 2; ; suffix++ {
		if _, ok := taken[candidate]; !ok {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", baseName, suffix)
	}
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("workspace: context is required")
	}
	return ctx.Err()
}

func durationMillis(duration time.Duration) int64 {
	return duration.Milliseconds()
}

func errorType(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrWorkspaceNotFound):
		return "workspace_not_found"
	case errors.Is(err, ErrWorkspaceRootMissing):
		return "workspace_root_missing"
	case errors.Is(err, ErrWorkspaceNameTaken):
		return "workspace_name_taken"
	case errors.Is(err, ErrWorkspacePathTaken):
		return "workspace_path_taken"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline_exceeded"
	default:
		return "error"
	}
}

func generateID(prefix string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		if strings.TrimSpace(prefix) == "" {
			return fmt.Sprintf("%d", now)
		}
		return fmt.Sprintf("%s_%d", prefix, now)
	}

	if strings.TrimSpace(prefix) == "" {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(random[:]))
}
