package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pedronauck/agh/internal/filesnap"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const workspaceCacheTTL = 10 * time.Minute

// Option customizes a Registry instance.
type Option func(*Registry)

// Registry manages global skills loaded at boot and lazily cached workspace skills.
type Registry struct {
	mu                sync.RWMutex
	globalSkills      map[string]*Skill
	globalLoaded      bool
	globalSnapshots   map[string]filesnap.Snapshot
	workspaceDisabled map[string][]string
	wsCache           map[string]*wsCache

	globalVersion atomic.Int64

	cfg    RegistryConfig
	logger *slog.Logger
	now    func() time.Time
}

type wsCache struct {
	skills     map[string]*Skill
	snapshots  map[string]filesnap.Snapshot
	lastAccess time.Time
}

type workspaceLoad struct {
	paths     []workspaceSkillPath
	snapshots map[string]filesnap.Snapshot
}

type workspaceSkillPath struct {
	filePath string
	source   SkillSource
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
		globalSkills:      make(map[string]*Skill),
		globalSnapshots:   make(map[string]filesnap.Snapshot),
		workspaceDisabled: make(map[string][]string),
		wsCache:           make(map[string]*wsCache),
		cfg:               cfg,
		logger:            slog.Default(),
		now:               time.Now,
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

// LoadContent loads the full markdown body for one resolved skill.
func (r *Registry) LoadContent(ctx context.Context, skill *Skill) (string, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return "", err
	}
	if skill == nil {
		return "", errors.New("skills: skill is required")
	}

	switch skill.Source {
	case SourceBundled:
		if r.cfg.BundledFS == nil {
			return "", errors.New("skills: bundled skills filesystem is required")
		}
		return readBundledSkillContent(r.cfg.BundledFS, skill.FilePath)
	default:
		return ReadSkillContent(skill.FilePath)
	}
}

// ForWorkspace returns the global skill set overlaid with resolver-provided workspace skills.
func (r *Registry) ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*Skill, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return nil, err
	}

	load, err := r.workspaceLoadFromResolved(ctx, resolved)
	if err != nil {
		return nil, err
	}
	if len(load.paths) == 0 {
		return r.List(), nil
	}

	cacheKey := workspaceCacheKey(resolved, load.paths)
	if cacheKey == "" {
		return nil, errors.New("skills: workspace cache key is required")
	}

	now := r.now()

	r.mu.Lock()
	r.evictExpiredWorkspaceLocked(now)

	if cached := r.wsCache[cacheKey]; cached != nil && filesnap.Equal(cached.snapshots, load.snapshots) {
		cached.lastAccess = now
		globalSkills := r.globalSkills
		workspaceSkills := cached.skills
		r.mu.Unlock()
		return mergedSkillList(globalSkills, workspaceSkills), nil
	}
	r.mu.Unlock()

	workspaceDisabled := r.workspaceDisabledSkillsSnapshot(cacheKey, resolved.Config.Skills.DisabledSkills)
	workspaceSkills, err := r.loadWorkspaceSkills(ctx, load.paths, workspaceDisabled)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.evictExpiredWorkspaceLocked(now)
	r.wsCache[cacheKey] = &wsCache{
		skills:     workspaceSkills,
		snapshots:  load.snapshots,
		lastAccess: now,
	}
	globalSkills := r.globalSkills
	r.mu.Unlock()

	return mergedSkillList(globalSkills, workspaceSkills), nil
}

// SetEnabled updates the runtime enabled state for a named skill and keeps the
// disabled-skills overlay in sync so future reloads preserve the change.
func (r *Registry) SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return errors.New("skills: skill name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if cacheKey, workspaceSkill := r.workspaceSkillTargetLocked(trimmedName, resolved); workspaceSkill != nil {
		workspaceSkill.Enabled = enabled
		r.workspaceDisabled[cacheKey] = setDisabledSkill(r.workspaceDisabled[cacheKey], trimmedName, enabled)
		r.globalVersion.Add(1)
		return nil
	}

	globalSkill := r.globalSkills[trimmedName]
	if globalSkill == nil {
		return fmt.Errorf("skills: skill %q not found", trimmedName)
	}

	globalSkill.Enabled = enabled
	r.cfg.DisabledSkills = setDisabledSkill(r.cfg.DisabledSkills, trimmedName, enabled)
	r.globalVersion.Add(1)

	return nil
}

func (r *Registry) reloadGlobal(ctx context.Context) error {
	if err := checkRegistryContext(ctx); err != nil {
		return err
	}

	disabledSkills := r.globalDisabledSkillsSnapshot()
	loaded, snapshots, err := r.loadGlobalSkills(ctx, disabledSkills)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.evictExpiredWorkspaceLocked(r.now())
	if r.globalLoaded && filesnap.Equal(r.globalSnapshots, snapshots) {
		return nil
	}

	r.globalSnapshots = filesnap.Clone(snapshots)
	r.globalLoaded = true
	r.globalSkills = loaded
	r.globalVersion.Add(1)

	return nil
}

func (r *Registry) loadGlobalSkills(ctx context.Context, disabledSkills []string) (map[string]*Skill, map[string]filesnap.Snapshot, error) {
	skills := make(map[string]*Skill)
	snapshots := make(map[string]filesnap.Snapshot)

	if err := r.loadBundledSkills(ctx, skills, disabledSkills); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(ctx, r.cfg.UserSkillsDir, SourceUser, skills, snapshots, disabledSkills); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(ctx, r.cfg.UserAgentsDir, SourceUser, skills, snapshots, disabledSkills); err != nil {
		return nil, nil, err
	}

	return skills, snapshots, nil
}

func (r *Registry) loadWorkspaceSkills(ctx context.Context, paths []workspaceSkillPath, disabledSkills []string) (map[string]*Skill, error) {
	skills := make(map[string]*Skill)

	for _, path := range paths {
		if err := checkRegistryContext(ctx); err != nil {
			return nil, err
		}

		skill, content, err := parseSkillFileDocument(path.filePath)
		if err != nil {
			return nil, err
		}
		skill.Source = path.source
		if !r.processSkill(skills, skill, content, disabledSkills) {
			continue
		}
	}

	return skills, nil
}

func (r *Registry) loadBundledSkills(ctx context.Context, dst map[string]*Skill, disabledSkills []string) error {
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

		skill, content, err := parseBundledSkillDocument(r.cfg.BundledFS, skillPath)
		if err != nil {
			return err
		}
		if !r.processSkill(dst, skill, content, disabledSkills) {
			continue
		}
	}

	return nil
}

func (r *Registry) loadDirectorySkills(ctx context.Context, dir string, source SkillSource, dst map[string]*Skill, snapshots map[string]filesnap.Snapshot, disabledSkills []string) error {
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
	if err := recordSidecarSnapshots(paths, snapshots); err != nil {
		return err
	}

	return r.loadSkillPaths(ctx, paths, source, dst, disabledSkills)
}

func (r *Registry) loadSkillPaths(ctx context.Context, paths []string, source SkillSource, dst map[string]*Skill, disabledSkills []string) error {
	for _, skillPath := range paths {
		if err := checkRegistryContext(ctx); err != nil {
			return err
		}

		skill, content, err := parseSkillFileDocument(skillPath)
		if err != nil {
			return err
		}
		if err := r.assignSourceAndProvenance(skill, source); err != nil {
			return err
		}
		if !r.processSkill(dst, skill, content, disabledSkills) {
			continue
		}
	}

	return nil
}

func (r *Registry) processSkill(dst map[string]*Skill, skill *Skill, content string, disabledSkills []string) bool {
	r.applyDisabled(skill, disabledSkills)

	verifyErr := r.verifyMarketplaceSkill(skill)
	warnings := VerifyContent(content)
	r.logVerificationWarnings(skill, warnings)
	if verifyErr != nil {
		return false
	}
	if hasCriticalWarning(warnings) {
		return false
	}

	r.overlaySkill(dst, skill)
	return true
}

func (r *Registry) assignSourceAndProvenance(skill *Skill, source SkillSource) error {
	if skill == nil {
		return errors.New("skills: skill is required")
	}

	skill.Source = source
	if source != SourceUser {
		return nil
	}

	hasSidecar, err := HasSidecar(skill.Dir)
	if err != nil {
		return err
	}
	if !hasSidecar {
		return nil
	}

	provenance, err := ReadSidecar(skill.Dir)
	if err != nil {
		return err
	}

	skill.Source = SourceMarketplace
	skill.Provenance = provenance
	skill.InstalledFrom = strings.TrimSpace(provenance.Slug)

	return nil
}

func (r *Registry) verifyMarketplaceSkill(skill *Skill) error {
	if skill == nil || skill.Source != SourceMarketplace || skill.Provenance == nil {
		return nil
	}

	err := VerifyHash(skill.Dir, skill.Provenance)
	if err == nil {
		return nil
	}

	var mismatch *HashMismatchError
	if errors.As(err, &mismatch) {
		r.logger.Warn(
			"skills: marketplace skill hash mismatch",
			"skill_name", skill.Meta.Name,
			"expected_hash", mismatch.ExpectedHash,
			"actual_hash", mismatch.ActualHash,
			"path", skill.FilePath,
		)
		return err
	}

	r.logger.Warn(
		"skills: marketplace skill hash verification failed",
		"skill_name", skill.Meta.Name,
		"path", skill.FilePath,
		"error", err,
	)

	return err
}

func recordSidecarSnapshots(paths []string, snapshots map[string]filesnap.Snapshot) error {
	for _, skillPath := range paths {
		sidecarPath := filepath.Join(filepath.Dir(skillPath), sidecarFileName)
		snapshot, err := filesnap.FromPath(sidecarPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return fmt.Errorf("skills: snapshot provenance sidecar %q: %w", sidecarPath, err)
		}

		snapshots[sidecarPath] = snapshot
	}

	return nil
}

func (r *Registry) applyDisabled(skill *Skill, disabledSkills []string) {
	if skill == nil {
		return
	}

	for _, disabled := range disabledSkills {
		if strings.TrimSpace(disabled) == skill.Meta.Name {
			skill.Enabled = false
			return
		}
	}
}

func addDisabledSkill(disabled []string, name string) []string {
	for _, existing := range disabled {
		if strings.TrimSpace(existing) == name {
			return disabled
		}
	}
	return append(disabled, name)
}

func removeDisabledSkill(disabled []string, name string) []string {
	if len(disabled) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(disabled))
	for _, existing := range disabled {
		if strings.TrimSpace(existing) == name {
			continue
		}
		filtered = append(filtered, existing)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func setDisabledSkill(disabled []string, name string, enabled bool) []string {
	if enabled {
		return removeDisabledSkill(disabled, name)
	}
	return addDisabledSkill(disabled, name)
}

func (r *Registry) globalDisabledSkillsSnapshot() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return slices.Clone(r.cfg.DisabledSkills)
}

func (r *Registry) workspaceDisabledSkillsSnapshot(cacheKey string, configured []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	disabledSkills := mergeDisabledSkills(r.cfg.DisabledSkills, configured)
	if cacheKey == "" {
		return disabledSkills
	}
	return mergeDisabledSkills(disabledSkills, r.workspaceDisabled[cacheKey])
}

func mergeDisabledSkills(base []string, extra []string) []string {
	merged := slices.Clone(base)
	for _, name := range extra {
		merged = addDisabledSkill(merged, strings.TrimSpace(name))
	}
	return merged
}

func (r *Registry) workspaceSkillTargetLocked(name string, resolved *workspacepkg.ResolvedWorkspace) (string, *Skill) {
	if resolved == nil {
		return "", nil
	}

	cacheKey := workspaceCacheKey(*resolved, nil)
	if cacheKey == "" {
		return "", nil
	}

	cached := r.wsCache[cacheKey]
	if cached == nil {
		return cacheKey, nil
	}

	return cacheKey, cached.skills[name]
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

func (r *Registry) workspaceLoadFromResolved(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) (workspaceLoad, error) {
	load := workspaceLoad{
		paths:     make([]workspaceSkillPath, 0, len(resolved.Skills)),
		snapshots: make(map[string]filesnap.Snapshot, len(resolved.Skills)),
	}

	for _, skillPath := range resolved.Skills {
		if err := checkRegistryContext(ctx); err != nil {
			return workspaceLoad{}, err
		}

		source, include, err := skillSourceFromWorkspacePath(skillPath.Source)
		if err != nil {
			return workspaceLoad{}, err
		}
		if !include {
			continue
		}

		skillDir := strings.TrimSpace(skillPath.Dir)
		if skillDir == "" {
			continue
		}

		skillFile := filepath.Join(skillDir, skillFileName)
		snapshot, err := filesnap.FromPath(skillFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return workspaceLoad{}, fmt.Errorf("skills: snapshot workspace skill %q: %w", skillFile, err)
		}

		load.snapshots[skillFile] = snapshot
		load.paths = append(load.paths, workspaceSkillPath{
			filePath: skillFile,
			source:   source,
		})
	}

	return load, nil
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
	clone.MCPServers = cloneMCPServerDecls(skill.MCPServers)
	clone.Hooks = cloneHookDecls(skill.Hooks)
	clone.Provenance = cloneProvenance(skill.Provenance)

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

func cloneMCPServerDecls(decls []MCPServerDecl) []MCPServerDecl {
	if decls == nil {
		return nil
	}

	clone := make([]MCPServerDecl, len(decls))
	for i, decl := range decls {
		clone[i] = MCPServerDecl{
			Name:    decl.Name,
			Command: decl.Command,
			Args:    append([]string(nil), decl.Args...),
			Env:     cloneStringMap(decl.Env),
		}
	}

	return clone
}

func cloneHookDecls(decls []HookDecl) []HookDecl {
	if decls == nil {
		return nil
	}

	clone := make([]HookDecl, len(decls))
	for i, decl := range decls {
		clone[i] = HookDecl{
			Event:   decl.Event,
			Command: decl.Command,
			Args:    append([]string(nil), decl.Args...),
			Timeout: decl.Timeout,
			Env:     cloneStringMap(decl.Env),
		}
	}

	return clone
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	clone := make(map[string]string, len(input))
	for key, value := range input {
		clone[key] = value
	}

	return clone
}

func cloneProvenance(provenance *Provenance) *Provenance {
	if provenance == nil {
		return nil
	}

	clone := *provenance
	return &clone
}

func (r *Registry) globalSnapshotState() (map[string]filesnap.Snapshot, bool) {
	if r == nil {
		return make(map[string]filesnap.Snapshot), false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return filesnap.Clone(r.globalSnapshots), r.globalLoaded
}

func parseBundledSkill(fsys fs.FS, skillPath string) (*Skill, error) {
	skill, _, err := parseBundledSkillDocument(fsys, skillPath)
	return skill, err
}

func readBundledSkillContent(fsys fs.FS, skillPath string) (string, error) {
	_, body, err := parseBundledSkillDocument(fsys, skillPath)
	if err != nil {
		return "", err
	}
	return body, nil
}

func parseBundledSkillDocument(fsys fs.FS, skillPath string) (*Skill, string, error) {
	content, err := fs.ReadFile(fsys, skillPath)
	if err != nil {
		return nil, "", fmt.Errorf("skills: read bundled skill %q: %w", skillPath, err)
	}

	dir := path.Dir(skillPath)
	if dir == "." {
		dir = ""
	}

	return parseSkillDocument(skillPath, dir, content, SourceBundled)
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

// SkillSourceName returns the canonical string label for a skill source.
func SkillSourceName(source SkillSource) string {
	return skillSourceName(source)
}

func skillSourceName(source SkillSource) string {
	switch source {
	case SourceBundled:
		return "bundled"
	case SourceMarketplace:
		return "marketplace"
	case SourceUser:
		return "user"
	case SourceAdditional:
		return "additional"
	case SourceWorkspace:
		return "workspace"
	default:
		return "unknown"
	}
}

func skillSourceFromWorkspacePath(source string) (SkillSource, bool, error) {
	switch strings.TrimSpace(source) {
	case "", "workspace":
		return SourceWorkspace, true, nil
	case "additional":
		return SourceAdditional, true, nil
	case "marketplace":
		return SourceMarketplace, false, nil
	case "global":
		return SourceUser, false, nil
	default:
		return 0, false, fmt.Errorf("skills: unsupported workspace skill source %q", source)
	}
}

func workspaceCacheKey(resolved workspacepkg.ResolvedWorkspace, paths []workspaceSkillPath) string {
	if id := strings.TrimSpace(resolved.ID); id != "" {
		return "id:" + id
	}
	if root := strings.TrimSpace(resolved.RootDir); root != "" {
		return "root:" + root
	}
	if len(paths) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, path := range paths {
		if builder.Len() > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(skillSourceName(path.source))
		builder.WriteByte(':')
		builder.WriteString(path.filePath)
	}

	return builder.String()
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
