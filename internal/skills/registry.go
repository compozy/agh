package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const workspaceCacheTTL = 10 * time.Minute

// Option customizes a Registry instance.
type Option func(*Registry)

// Registry manages global skills loaded at boot and lazily cached workspace skills.
type Registry struct {
	mu              sync.RWMutex
	globalSkills    map[string]*Skill
	globalLoaded    bool
	globalSnapshots map[string]fileSnapshot
	wsCache         map[string]*wsCache

	globalVersion atomic.Int64

	cfg    RegistryConfig
	logger *slog.Logger
	now    func() time.Time
}

type wsCache struct {
	skills     map[string]*Skill
	snapshots  map[string]fileSnapshot
	lastAccess time.Time
}

type workspaceScan struct {
	agents    []string
	workspace []string
	snapshots map[string]fileSnapshot
}

// WithLogger injects the logger used for registry diagnostics.
func WithLogger(logger *slog.Logger) Option {
	return func(registry *Registry) {
		registry.logger = logger
	}
}

// WithNow injects the clock used for cache timestamps and eviction.
func WithNow(now func() time.Time) Option {
	return func(registry *Registry) {
		registry.now = now
	}
}

// NewRegistry constructs a Registry with the provided configuration.
func NewRegistry(cfg RegistryConfig, opts ...Option) *Registry {
	registry := &Registry{
		globalSkills:    make(map[string]*Skill),
		globalSnapshots: make(map[string]fileSnapshot),
		wsCache:         make(map[string]*wsCache),
		cfg:             cfg,
		logger:          slog.Default(),
		now:             time.Now,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(registry)
		}
	}

	if registry.logger == nil {
		registry.logger = slog.Default()
	}
	if registry.now == nil {
		registry.now = time.Now
	}

	return registry
}

// LoadAll loads bundled and user-level skills into the global registry snapshot.
func (r *Registry) LoadAll(ctx context.Context) error {
	return r.reloadGlobal(ctx)
}

// RefreshGlobal re-scans the global skill sources and swaps them in atomically.
func (r *Registry) RefreshGlobal(ctx context.Context) error {
	return r.reloadGlobal(ctx)
}

// GlobalVersion returns the current global snapshot version.
func (r *Registry) GlobalVersion() int64 {
	return r.globalVersion.Load()
}

// Get returns a cloned global skill by name when present.
func (r *Registry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	skill, ok := r.globalSkills[name]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}

	return cloneSkill(skill), true
}

// List returns the current global skills sorted by skill name.
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	globalSkills := r.globalSkills
	r.mu.RUnlock()

	return mergedSkillList(globalSkills, nil)
}

// ForWorkspace returns the global skill set overlaid with workspace-local skills.
func (r *Registry) ForWorkspace(ctx context.Context, workspace string) ([]*Skill, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return nil, err
	}

	root := strings.TrimSpace(workspace)
	if root == "" {
		return r.List(), nil
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("skills: resolve workspace %q: %w", workspace, err)
	}

	scan, err := r.scanWorkspace(ctx, absRoot)
	if err != nil {
		return nil, err
	}

	now := r.now()

	r.mu.Lock()
	r.evictExpiredWorkspaceLocked(now)

	if cached := r.wsCache[absRoot]; cached != nil && snapshotsEqual(cached.snapshots, scan.snapshots) {
		cached.lastAccess = now
		globalSkills := r.globalSkills
		workspaceSkills := cached.skills
		r.mu.Unlock()
		return mergedSkillList(globalSkills, workspaceSkills), nil
	}
	r.mu.Unlock()

	workspaceSkills, err := r.loadWorkspaceSkills(ctx, scan)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.evictExpiredWorkspaceLocked(now)
	r.wsCache[absRoot] = &wsCache{
		skills:     workspaceSkills,
		snapshots:  scan.snapshots,
		lastAccess: now,
	}
	globalSkills := r.globalSkills
	r.mu.Unlock()

	return mergedSkillList(globalSkills, workspaceSkills), nil
}

func (r *Registry) reloadGlobal(ctx context.Context) error {
	if err := checkRegistryContext(ctx); err != nil {
		return err
	}

	loaded, snapshots, err := r.loadGlobalSkills(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.evictExpiredWorkspaceLocked(r.now())
	r.globalSnapshots = cloneFileSnapshots(snapshots)
	r.globalLoaded = true
	if reflect.DeepEqual(r.globalSkills, loaded) {
		return nil
	}

	r.globalSkills = loaded
	r.globalVersion.Add(1)

	return nil
}

func (r *Registry) loadGlobalSkills(ctx context.Context) (map[string]*Skill, map[string]fileSnapshot, error) {
	skills := make(map[string]*Skill)
	snapshots := make(map[string]fileSnapshot)

	if err := r.loadBundledSkills(ctx, skills); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(ctx, r.cfg.UserSkillsDir, SourceUser, skills, snapshots); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(ctx, r.cfg.UserAgentsDir, SourceUser, skills, snapshots); err != nil {
		return nil, nil, err
	}

	return skills, snapshots, nil
}

func (r *Registry) loadWorkspaceSkills(ctx context.Context, scan workspaceScan) (map[string]*Skill, error) {
	skills := make(map[string]*Skill)

	if err := r.loadSkillPaths(ctx, scan.agents, SourceAgents, skills); err != nil {
		return nil, err
	}
	if err := r.loadSkillPaths(ctx, scan.workspace, SourceWorkspace, skills); err != nil {
		return nil, err
	}

	return skills, nil
}

func (r *Registry) loadBundledSkills(ctx context.Context, dst map[string]*Skill) error {
	if r.cfg.BundledFS == nil {
		return nil
	}

	paths, err := scanBundledFS(r.cfg.BundledFS)
	if err != nil {
		return err
	}

	for _, skillPath := range paths {
		if err := checkRegistryContext(ctx); err != nil {
			return err
		}

		skill, err := parseBundledSkill(r.cfg.BundledFS, skillPath)
		if err != nil {
			return err
		}
		r.applyDisabled(skill)

		warnings := VerifyContent(skill.Content)
		r.logVerificationWarnings(skill, warnings)
		if hasCriticalWarning(warnings) {
			continue
		}

		r.overlaySkill(dst, skill)
	}

	return nil
}

func (r *Registry) loadDirectorySkills(ctx context.Context, dir string, source SkillSource, dst map[string]*Skill, snapshots map[string]fileSnapshot) error {
	root := strings.TrimSpace(dir)
	if root == "" {
		return nil
	}

	paths, dirSnapshots, err := scanDirectoryWithSnapshots(root)
	if err != nil {
		return err
	}
	for path, snapshot := range dirSnapshots {
		snapshots[path] = snapshot
	}

	return r.loadSkillPaths(ctx, paths, source, dst)
}

func (r *Registry) loadSkillPaths(ctx context.Context, paths []string, source SkillSource, dst map[string]*Skill) error {
	for _, skillPath := range paths {
		if err := checkRegistryContext(ctx); err != nil {
			return err
		}

		skill, err := ParseSkillFile(skillPath)
		if err != nil {
			return err
		}
		skill.Source = source
		r.applyDisabled(skill)

		warnings := VerifyContent(skill.Content)
		r.logVerificationWarnings(skill, warnings)
		if hasCriticalWarning(warnings) {
			continue
		}

		r.overlaySkill(dst, skill)
	}

	return nil
}

func (r *Registry) applyDisabled(skill *Skill) {
	if skill == nil {
		return
	}

	for _, disabled := range r.cfg.DisabledSkills {
		if strings.TrimSpace(disabled) == skill.Meta.Name {
			skill.Enabled = false
			return
		}
	}
}

func (r *Registry) overlaySkill(dst map[string]*Skill, skill *Skill) {
	if existing, ok := dst[skill.Meta.Name]; ok {
		r.logger.Warn(
			"skills: overriding skill",
			"name", skill.Meta.Name,
			"old_source", skillSourceName(existing.Source),
			"new_source", skillSourceName(skill.Source),
			"old_path", existing.FilePath,
			"new_path", skill.FilePath,
		)
	}

	dst[skill.Meta.Name] = skill
}

func (r *Registry) logVerificationWarnings(skill *Skill, warnings []Warning) {
	for _, warning := range warnings {
		if warning.Severity == SeverityInfo {
			r.logger.Info(
				"skills: verification warning",
				"name", skill.Meta.Name,
				"source", skillSourceName(skill.Source),
				"path", skill.FilePath,
				"severity", warningSeverityName(warning.Severity),
				"pattern", warning.Pattern,
				"message", warning.Message,
			)
			continue
		}

		r.logger.Warn(
			"skills: verification warning",
			"name", skill.Meta.Name,
			"source", skillSourceName(skill.Source),
			"path", skill.FilePath,
			"severity", warningSeverityName(warning.Severity),
			"pattern", warning.Pattern,
			"message", warning.Message,
		)
	}
}

func (r *Registry) scanWorkspace(ctx context.Context, workspace string) (workspaceScan, error) {
	agentsDir := filepath.Join(workspace, ".agents", "skills")
	workspaceDir := filepath.Join(workspace, ".agh", "skills")

	agents, err := scanDirectory(agentsDir)
	if err != nil {
		return workspaceScan{}, err
	}
	if err := checkRegistryContext(ctx); err != nil {
		return workspaceScan{}, err
	}

	project, err := scanDirectory(workspaceDir)
	if err != nil {
		return workspaceScan{}, err
	}
	if err := checkRegistryContext(ctx); err != nil {
		return workspaceScan{}, err
	}

	scan := workspaceScan{
		agents:    make([]string, 0, len(agents)),
		workspace: make([]string, 0, len(project)),
		snapshots: make(map[string]fileSnapshot, len(agents)+len(project)),
	}

	appendSnapshots := func(paths []string, dst *[]string) error {
		for _, skillPath := range paths {
			if err := checkRegistryContext(ctx); err != nil {
				return err
			}

			snapshot, err := snapshotFile(skillPath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					continue
				}
				return fmt.Errorf("skills: snapshot workspace skill %q: %w", skillPath, err)
			}

			scan.snapshots[skillPath] = snapshot
			*dst = append(*dst, skillPath)
		}
		return nil
	}

	if err := appendSnapshots(agents, &scan.agents); err != nil {
		return workspaceScan{}, err
	}
	if err := appendSnapshots(project, &scan.workspace); err != nil {
		return workspaceScan{}, err
	}

	return scan, nil
}

func (r *Registry) evictExpiredWorkspaceLocked(now time.Time) {
	cutoff := now.Add(-workspaceCacheTTL)
	for workspace, entry := range r.wsCache {
		if entry.lastAccess.Before(cutoff) {
			delete(r.wsCache, workspace)
		}
	}
}

func checkRegistryContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("skills: context is required")
	}
	return ctx.Err()
}

func hasCriticalWarning(warnings []Warning) bool {
	for _, warning := range warnings {
		if warning.Severity == SeverityCritical {
			return true
		}
	}
	return false
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
		if !leftSnapshot.modTime.Equal(rightSnapshot.modTime) {
			return false
		}
		if leftSnapshot.size != rightSnapshot.size {
			return false
		}
	}

	return true
}

func mergedSkillList(globalSkills, workspaceSkills map[string]*Skill) []*Skill {
	if len(globalSkills) == 0 && len(workspaceSkills) == 0 {
		return nil
	}

	merged := make(map[string]*Skill, len(globalSkills)+len(workspaceSkills))
	for name, skill := range globalSkills {
		merged[name] = skill
	}
	for name, skill := range workspaceSkills {
		merged[name] = skill
	}

	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	slices.Sort(names)

	skills := make([]*Skill, 0, len(names))
	for _, name := range names {
		skills = append(skills, cloneSkill(merged[name]))
	}

	return skills
}

func cloneSkill(skill *Skill) *Skill {
	if skill == nil {
		return nil
	}

	clone := *skill
	clone.Meta = cloneSkillMeta(skill.Meta)

	return &clone
}

func cloneSkillMeta(meta SkillMeta) SkillMeta {
	clone := meta
	clone.Metadata = cloneMetadataMap(meta.Metadata)
	return clone
}

func cloneMetadataMap(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	clone := make(map[string]any, len(metadata))
	for key, value := range metadata {
		clone[key] = cloneMetadataValue(value)
	}

	return clone
}

func cloneMetadataValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMetadataMap(typed)
	case []any:
		clone := make([]any, len(typed))
		for i := range typed {
			clone[i] = cloneMetadataValue(typed[i])
		}
		return clone
	default:
		return typed
	}
}

func cloneFileSnapshots(snapshots map[string]fileSnapshot) map[string]fileSnapshot {
	if len(snapshots) == 0 {
		return make(map[string]fileSnapshot)
	}

	clone := make(map[string]fileSnapshot, len(snapshots))
	for path, snapshot := range snapshots {
		clone[path] = snapshot
	}

	return clone
}

func (r *Registry) globalSnapshotState() (map[string]fileSnapshot, bool) {
	if r == nil {
		return make(map[string]fileSnapshot), false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return cloneFileSnapshots(r.globalSnapshots), r.globalLoaded
}

func parseBundledSkill(fsys fs.FS, skillPath string) (*Skill, error) {
	content, err := fs.ReadFile(fsys, skillPath)
	if err != nil {
		return nil, fmt.Errorf("skills: read bundled skill %q: %w", skillPath, err)
	}

	meta, body, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("skills: parse bundled skill %q: %w", skillPath, err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("skills: parse bundled skill %q: %w", skillPath, errSkillNameRequired)
	}

	dir := path.Dir(skillPath)
	if dir == "." {
		dir = ""
	}

	return &Skill{
		Meta:     meta,
		Content:  body,
		Source:   SourceBundled,
		Dir:      dir,
		FilePath: skillPath,
		Enabled:  true,
	}, nil
}

func scanBundledFS(fsys fs.FS) ([]string, error) {
	paths := make([]string, 0, maxScanCandidates)

	walkErr := fs.WalkDir(fsys, ".", func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == "." {
			return nil
		}

		depth := fsPathDepth(current, entry.IsDir())
		if entry.IsDir() {
			if shouldSkipDir(path.Base(current)) {
				return fs.SkipDir
			}
			if depth > maxScanDepth {
				return fs.SkipDir
			}
			return nil
		}

		if path.Base(current) != skillFileName || depth > maxScanDepth {
			return nil
		}

		if _, err := fs.Stat(fsys, current); err != nil {
			return err
		}

		paths = append(paths, current)
		if len(paths) >= maxScanCandidates {
			return errScanLimitReached
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errScanLimitReached) {
		return nil, fmt.Errorf("skills: scan bundled skills: %w", walkErr)
	}

	slices.Sort(paths)
	return paths, nil
}

func fsPathDepth(current string, isDir bool) int {
	trimmed := strings.Trim(current, "/")
	if trimmed == "" {
		return 0
	}

	parts := strings.Split(trimmed, "/")
	if isDir {
		return len(parts)
	}

	return max(len(parts)-1, 0)
}

func skillSourceName(source SkillSource) string {
	switch source {
	case SourceBundled:
		return "bundled"
	case SourceUser:
		return "user"
	case SourceAgents:
		return "agents"
	case SourceWorkspace:
		return "workspace"
	default:
		return "unknown"
	}
}

func warningSeverityName(severity WarningSeverity) string {
	switch severity {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}
