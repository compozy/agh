package skills

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pedronauck/agh/internal/filesnap"
	"github.com/pedronauck/agh/internal/resources"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	workspaceCacheTTL          = 10 * time.Minute
	skillSourceMarketplaceName = "marketplace"
)

// Option customizes a Registry instance.
type Option func(*Registry)

// Registry manages global skills loaded at boot and lazily cached workspace skills.
type Registry struct {
	mu                 sync.RWMutex
	globalSkills       map[string]*Skill
	resourceAuthority  bool
	resourceRevision   int64
	resourceWorkspaces map[string]map[string]*Skill
	globalLoaded       bool
	globalSnapshots    map[string]filesnap.Snapshot
	workspaceDisabled  map[string][]string
	wsCache            map[string]*wsCache

	globalVersion atomic.Int64

	cfg    RegistryConfig
	logger *slog.Logger
	now    func() time.Time
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
		globalSkills:       make(map[string]*Skill),
		resourceWorkspaces: make(map[string]map[string]*Skill),
		globalSnapshots:    make(map[string]filesnap.Snapshot),
		workspaceDisabled:  make(map[string][]string),
		wsCache:            make(map[string]*wsCache),
		cfg:                cfg,
		logger:             slog.Default(),
		now:                time.Now,
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
	skill, ok := r.lookupSkillLocked(name)
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
func (r *Registry) ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*Skill, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return nil, err
	}

	if skills, ok := r.resourceBackedWorkspaceSkills(resolved); ok {
		return skills, nil
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

	if r.resourceAuthority {
		if skill := r.resourceSkillTargetLocked(trimmedName, resolved); skill != nil {
			skill.Enabled = enabled
			r.globalVersion.Add(1)
			return nil
		}
		return fmt.Errorf("skills: skill %q not found", trimmedName)
	}

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
	if r.usesResourceAuthority() {
		return nil
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

// DiscoverGlobal loads global skill definitions for resource publication without
// making the file-system scan authoritative in the registry.
func (r *Registry) DiscoverGlobal(ctx context.Context) ([]*Skill, map[string]filesnap.Snapshot, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return nil, nil, err
	}
	disabledSkills := r.globalDisabledSkillsSnapshot()
	loaded, snapshots, err := r.loadGlobalSkills(ctx, disabledSkills)
	if err != nil {
		return nil, nil, err
	}
	return mergedSkillList(loaded, nil), filesnap.Clone(snapshots), nil
}

// DiscoverWorkspace loads workspace-visible skill definitions for resource publication.
func (r *Registry) DiscoverWorkspace(
	ctx context.Context,
	resolved *workspacepkg.ResolvedWorkspace,
) ([]*Skill, map[string]filesnap.Snapshot, error) {
	if err := checkRegistryContext(ctx); err != nil {
		return nil, nil, err
	}
	load, err := r.workspaceLoadFromResolved(ctx, resolved)
	if err != nil {
		return nil, nil, err
	}
	if len(load.paths) == 0 {
		return nil, load.snapshots, nil
	}
	workspaceDisabled := r.workspaceDisabledSkillsSnapshot(
		workspaceCacheKey(resolved, load.paths),
		resolved.Config.Skills.DisabledSkills,
	)
	loaded, err := r.loadWorkspaceSkills(ctx, load.paths, workspaceDisabled)
	if err != nil {
		return nil, nil, err
	}
	return mergedSkillList(nil, loaded), load.snapshots, nil
}

// ApplyResourceRecords atomically replaces the runtime skill catalog with the
// canonical resource projection.
func (r *Registry) ApplyResourceRecords(revision int64, records []resources.Record[SkillResourceSpec]) error {
	if r == nil {
		return errors.New("skills: registry is required")
	}
	globalSkills := make(map[string]*Skill)
	workspaceSkills := make(map[string]map[string]*Skill)

	ordered := append([]resources.Record[SkillResourceSpec](nil), records...)
	slices.SortFunc(ordered, func(left, right resources.Record[SkillResourceSpec]) int {
		return strings.Compare(skillRecordSortKey(left), skillRecordSortKey(right))
	})

	for _, record := range ordered {
		skill, err := SkillFromResourceSpec(record.Spec)
		if err != nil {
			return fmt.Errorf("skills: convert resource %q: %w", record.ID, err)
		}
		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}
		switch record.Scope.Kind.Normalize() {
		case resources.ResourceScopeKindGlobal:
			globalSkills[name] = skill
		case resources.ResourceScopeKindWorkspace:
			workspaceID := strings.TrimSpace(record.Scope.ID)
			if workspaceID == "" {
				continue
			}
			if workspaceSkills[workspaceID] == nil {
				workspaceSkills[workspaceID] = make(map[string]*Skill)
			}
			workspaceSkills[workspaceID][name] = skill
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.resourceAuthority = true
	r.resourceRevision = revision
	r.resourceWorkspaces = workspaceSkills
	r.globalSkills = globalSkills
	r.wsCache = make(map[string]*wsCache)
	r.globalLoaded = true
	r.globalVersion.Add(1)
	return nil
}

func (r *Registry) loadGlobalSkills(
	ctx context.Context,
	disabledSkills []string,
) (map[string]*Skill, map[string]filesnap.Snapshot, error) {
	skills := make(map[string]*Skill)
	snapshots := make(map[string]filesnap.Snapshot)

	if err := r.loadBundledSkills(ctx, skills, disabledSkills); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(
		ctx,
		r.cfg.UserSkillsDir,
		SourceUser,
		skills,
		snapshots,
		disabledSkills,
	); err != nil {
		return nil, nil, err
	}
	if err := r.loadDirectorySkills(
		ctx,
		r.cfg.UserAgentsDir,
		SourceUser,
		skills,
		snapshots,
		disabledSkills,
	); err != nil {
		return nil, nil, err
	}

	return skills, snapshots, nil
}

func (r *Registry) loadWorkspaceSkills(
	ctx context.Context,
	paths []workspaceSkillPath,
	disabledSkills []string,
) (map[string]*Skill, error) {
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
		refreshSkillHookDecls(skill)
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

func (r *Registry) loadDirectorySkills(
	ctx context.Context,
	dir string,
	source SkillSource,
	dst map[string]*Skill,
	snapshots map[string]filesnap.Snapshot,
	disabledSkills []string,
) error {
	root := strings.TrimSpace(dir)
	if root == "" {
		return nil
	}

	paths, dirSnapshots, err := scanDirectoryWithSnapshots(root)
	if err != nil {
		return err
	}
	maps.Copy(snapshots, dirSnapshots)
	if err := recordSidecarSnapshots(paths, snapshots); err != nil {
		return err
	}

	return r.loadSkillPaths(ctx, paths, source, dst, disabledSkills)
}

func (r *Registry) loadSkillPaths(
	ctx context.Context,
	paths []string,
	source SkillSource,
	dst map[string]*Skill,
	disabledSkills []string,
) error {
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
		refreshSkillHookDecls(skill)
		return nil
	}

	hasSidecar, err := HasSidecar(skill.Dir)
	if err != nil {
		return err
	}
	if !hasSidecar {
		refreshSkillHookDecls(skill)
		return nil
	}

	provenance, err := ReadSidecar(skill.Dir)
	if err != nil {
		return err
	}

	skill.Source = SourceMarketplace
	skill.Provenance = provenance
	skill.InstalledFrom = strings.TrimSpace(provenance.Slug)
	refreshSkillHookDecls(skill)

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

func (r *Registry) usesResourceAuthority() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resourceAuthority
}

func (r *Registry) resourceBackedWorkspaceSkills(resolved *workspacepkg.ResolvedWorkspace) ([]*Skill, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.resourceAuthority {
		return nil, false
	}
	workspaceSkills := r.resourceWorkspaces[resourceWorkspaceKey(resolved)]
	return mergedSkillList(r.globalSkills, workspaceSkills), true
}

func (r *Registry) resourceSkillTargetLocked(name string, resolved *workspacepkg.ResolvedWorkspace) *Skill {
	if r == nil || !r.resourceAuthority {
		return nil
	}
	if key := resourceWorkspaceKey(resolved); key != "" {
		if workspaceSkills := r.resourceWorkspaces[key]; workspaceSkills != nil {
			if skill := workspaceSkills[name]; skill != nil {
				return skill
			}
		}
	}
	return r.globalSkills[name]
}

func (r *Registry) lookupSkillLocked(name string) (*Skill, bool) {
	if r == nil {
		return nil, false
	}
	skill := r.globalSkills[strings.TrimSpace(name)]
	return skill, skill != nil
}

func resourceWorkspaceKey(resolved *workspacepkg.ResolvedWorkspace) string {
	if resolved == nil {
		return ""
	}
	return strings.TrimSpace(resolved.ID)
}

func skillRecordSortKey(record resources.Record[SkillResourceSpec]) string {
	return string(record.Scope.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Scope.ID) + "\x00" +
		string(record.Source.Kind.Normalize()) + "\x00" +
		strings.TrimSpace(record.Source.ID) + "\x00" +
		strings.TrimSpace(record.ID)
}

func mergeDisabledSkills(base []string, extra []string) []string {
	merged := slices.Clone(base)
	for _, name := range extra {
		merged = addDisabledSkill(merged, strings.TrimSpace(name))
	}
	return merged
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

func checkRegistryContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("skills: context is required")
	}
	return ctx.Err()
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
		return skillSourceMarketplaceName
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
	case skillSourceMarketplaceName:
		return SourceMarketplace, false, nil
	case "global":
		return SourceUser, false, nil
	default:
		return 0, false, fmt.Errorf("skills: unsupported workspace skill source %q", source)
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
