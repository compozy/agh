package marketplace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	aghconfig "github.com/pedronauck/agh/internal/config"
	registrypkg "github.com/pedronauck/agh/internal/registry"
	registryclawhub "github.com/pedronauck/agh/internal/registry/clawhub"
	"github.com/pedronauck/agh/internal/skills"
	"golang.org/x/text/unicode/norm"
)

const (
	// DefaultRegistry is the built-in marketplace source used when config omits
	// an explicit registry.
	DefaultRegistry = "clawhub"
	// DefaultSearchLimit matches the CLI marketplace search default.
	DefaultSearchLimit = 20
	// SkillMarkdownFileName is the canonical skill manifest file name.
	SkillMarkdownFileName = "SKILL.md"
)

const (
	// UpdateStatusCurrent reports that an installed skill is already current.
	UpdateStatusCurrent = "already up to date"
	// UpdateStatusAvailable reports that an update exists but was not applied.
	UpdateStatusAvailable = "update available"
	// UpdateStatusUpdated reports that an update was applied.
	UpdateStatusUpdated = "updated"
)

var (
	// ErrValidation classifies malformed marketplace requests.
	ErrValidation = errors.New("skills marketplace: validation error")
	// ErrNotFound classifies missing installed or remote marketplace skills.
	ErrNotFound = errors.New("skills marketplace: not found")
	// ErrNotMarketplace classifies installed skills that lack marketplace provenance.
	ErrNotMarketplace = errors.New("skills marketplace: not marketplace")
	// ErrNotConfigured classifies missing or unsupported marketplace registry setup.
	ErrNotConfigured = errors.New("skills marketplace: not configured")

	validSkillNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	validSkillSlugPattern = regexp.MustCompile(`^@[^/\s]+/[^/\s]+$`)
)

// Error preserves a stable operator-facing message while allowing callers to
// map lifecycle failures with errors.Is.
type Error struct {
	Kind    error
	Message string
	Cause   error
}

func (e *Error) Error() string {
	message := strings.TrimSpace(e.Message)
	if e.Cause == nil {
		return message
	}
	if message == "" {
		return e.Cause.Error()
	}
	return fmt.Sprintf("%s: %v", message, e.Cause)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) Is(target error) bool {
	return target == e.Kind
}

// Registry is the marketplace registry contract required by skill lifecycle operations.
type Registry interface {
	registrypkg.Downloader
	Info(ctx context.Context, slug string) (*registrypkg.Detail, error)
	CheckUpdate(
		ctx context.Context,
		slug string,
		currentVersion string,
	) (*registrypkg.UpdateInfo, error)
}

// NamedRegistry can resolve an installed skill's recorded registry source.
type NamedRegistry interface {
	Registry
	SourceNamed(name string) registrypkg.Source
}

// SourceLoader resolves configured marketplace sources.
type SourceLoader func(aghconfig.MarketplaceConfig) ([]registrypkg.Source, error)

// Service exposes daemon-safe marketplace operations for skills.
type Service struct {
	homePaths    aghconfig.HomePaths
	skillsConfig aghconfig.SkillsConfig
	logger       *slog.Logger
	now          func() time.Time
	sourceLoader SourceLoader
}

// Option customizes a Service.
type Option func(*Service)

// WithLogger sets the logger used by the underlying multi-registry.
func WithLogger(logger *slog.Logger) Option {
	return func(service *Service) {
		service.logger = logger
	}
}

// WithNow sets the clock used for install provenance timestamps.
func WithNow(now func() time.Time) Option {
	return func(service *Service) {
		service.now = now
	}
}

// WithSourceLoader overrides marketplace source resolution, primarily for tests.
func WithSourceLoader(loader SourceLoader) Option {
	return func(service *Service) {
		service.sourceLoader = loader
	}
}

// NewService constructs a skill marketplace lifecycle service.
func NewService(
	homePaths aghconfig.HomePaths,
	skillsConfig aghconfig.SkillsConfig,
	opts ...Option,
) *Service {
	service := &Service{
		homePaths:    homePaths,
		skillsConfig: skillsConfig,
		logger:       slog.Default(),
		now:          time.Now,
		sourceLoader: DefaultSourceLoader,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	if service.logger == nil {
		service.logger = slog.Default()
	}
	if service.now == nil {
		service.now = time.Now
	}
	if service.sourceLoader == nil {
		service.sourceLoader = DefaultSourceLoader
	}
	return service
}

// InstalledSkill describes one locally installed marketplace-backed skill.
type InstalledSkill struct {
	Name       string
	Dir        string
	FilePath   string
	Provenance skills.Provenance
}

// InstallResult describes one completed marketplace install.
type InstallResult struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Version  string `json:"version,omitempty"`
	Registry string `json:"registry"`
	Path     string `json:"path"`
	Hash     string `json:"hash"`
	Status   string `json:"status"`
}

// RemoveResult describes one removed marketplace skill.
type RemoveResult struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

// UpdateRequest describes a single-skill or batch marketplace update.
type UpdateRequest struct {
	Name      string
	All       bool
	CheckOnly bool
}

// UpdateResult describes one marketplace update outcome.
type UpdateResult struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Path           string `json:"path"`
	Status         string `json:"status"`
}

// SourceBackedRegistry adapts one configured source into the registry contract.
type SourceBackedRegistry struct {
	Source registrypkg.Source
}

// DefaultSourceLoader resolves the configured marketplace registry source.
func DefaultSourceLoader(registryCfg aghconfig.MarketplaceConfig) ([]registrypkg.Source, error) {
	registryName := strings.ToLower(strings.TrimSpace(registryCfg.Registry))
	if registryName == "" {
		registryName = DefaultRegistry
	}

	switch registryName {
	case DefaultRegistry:
		return []registrypkg.Source{
			registryclawhub.NewClient(registryCfg.BaseURL),
		}, nil
	default:
		return nil, classifiedf(
			ErrNotConfigured,
			"unsupported marketplace registry %q",
			registryCfg.Registry,
		)
	}
}

// Search queries configured marketplace sources for skill packages.
func (s *Service) Search(
	ctx context.Context,
	query string,
	limit int,
) (_ []registrypkg.Listing, err error) {
	if limit <= 0 {
		return nil, classifiedf(ErrValidation, "search limit must be positive: %d", limit)
	}
	if strings.TrimSpace(query) == "" {
		return nil, classifiedf(ErrValidation, "skill marketplace search query is required")
	}

	registry, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	return registry.Search(ctx, query, registrypkg.SearchOpts{
		Limit: limit,
		Type:  registrypkg.PackageTypeSkill,
	})
}

// Info resolves remote metadata for one marketplace skill.
func (s *Service) Info(ctx context.Context, slug string) (_ *registrypkg.Detail, err error) {
	normalizedSlug, err := NormalizeSkillSlug(slug)
	if err != nil {
		return nil, err
	}

	registry, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	detail, err := registry.Info(ctx, normalizedSlug)
	if err != nil {
		return nil, classifyRegistryLookup(err)
	}
	if detail == nil {
		return nil, classifiedf(
			ErrNotFound,
			"marketplace info returned no detail for %q",
			normalizedSlug,
		)
	}
	return detail, nil
}

// Install installs one marketplace skill into the configured AGH skills root.
func (s *Service) Install(ctx context.Context, slug string, version string) (_ InstallResult, err error) {
	registry, err := s.loadRegistry()
	if err != nil {
		return InstallResult{}, err
	}
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	return InstallWithRegistry(ctx, s.homePaths.SkillsDir, registry, slug, version, "", s.now)
}

// Update checks or applies marketplace updates for one skill or every installed marketplace skill.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (_ []UpdateResult, err error) {
	registry, err := s.loadRegistry()
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	return UpdateWithRegistry(ctx, s.homePaths.SkillsDir, registry, req, s.now)
}

// Remove removes one installed marketplace skill.
func (s *Service) Remove(_ context.Context, name string) (RemoveResult, error) {
	return RemoveSkill(s.homePaths.SkillsDir, name)
}

// Download implements Registry.
func (r SourceBackedRegistry) Download(
	ctx context.Context,
	slug string,
	opts registrypkg.DownloadOpts,
) (*registrypkg.DownloadResult, error) {
	if r.Source == nil {
		return nil, classifiedf(ErrNotConfigured, "registry source is required")
	}
	return r.Source.Download(ctx, slug, opts)
}

// Info implements Registry.
func (r SourceBackedRegistry) Info(
	ctx context.Context,
	slug string,
) (*registrypkg.Detail, error) {
	if r.Source == nil {
		return nil, classifiedf(ErrNotConfigured, "registry source is required")
	}

	detail, err := r.Source.Info(ctx, slug)
	if err != nil {
		return nil, err
	}
	if detail != nil && strings.TrimSpace(detail.Source) == "" {
		detail.Source = strings.TrimSpace(r.Source.Name())
	}
	return detail, nil
}

// CheckUpdate implements Registry.
func (r SourceBackedRegistry) CheckUpdate(
	ctx context.Context,
	slug string,
	currentVersion string,
) (*registrypkg.UpdateInfo, error) {
	detail, err := r.Info(ctx, slug)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, classifiedf(
			ErrNotFound,
			"marketplace info returned no detail for %q",
			slug,
		)
	}

	latestVersion := strings.TrimSpace(detail.Version)
	return &registrypkg.UpdateInfo{
		Slug:           strings.TrimSpace(slug),
		CurrentVersion: strings.TrimSpace(currentVersion),
		LatestVersion:  latestVersion,
		HasUpdate:      registrypkg.VersionIsNewer(currentVersion, latestVersion),
		Source:         strings.TrimSpace(detail.Source),
	}, nil
}

// NormalizeSkillSlug validates the canonical marketplace skill slug shape.
func NormalizeSkillSlug(slug string) (string, error) {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return "", classifiedf(ErrValidation, "skill slug is required")
	}
	if !validSkillSlugPattern.MatchString(trimmed) {
		return "", classifiedf(ErrValidation, `skill slug must match "@author/name"`)
	}
	return trimmed, nil
}

// NormalizeSkillName validates one local skill name.
func NormalizeSkillName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return "", classifiedf(ErrValidation, "skill name is required")
	case trimmed == ".", trimmed == "..":
		return "", classifiedf(ErrValidation, "skill name must not be a relative path segment")
	case filepath.IsAbs(trimmed):
		return "", classifiedf(ErrValidation, "skill name must be relative")
	case strings.Contains(trimmed, "/"), strings.Contains(trimmed, `\`):
		return "", classifiedf(ErrValidation, "skill name must not include path separators")
	case !validSkillNamePattern.MatchString(trimmed):
		return "", classifiedf(
			ErrValidation,
			"skill name must contain only letters, numbers, dots, underscores, and hyphens",
		)
	default:
		return trimmed, nil
	}
}

// InstallWithRegistry installs one marketplace skill using the supplied registry.
func InstallWithRegistry(
	ctx context.Context,
	skillsDir string,
	registry Registry,
	slug string,
	version string,
	targetDirOverride string,
	now func() time.Time,
) (item InstallResult, err error) {
	normalizedSlug, err := NormalizeSkillSlug(slug)
	if err != nil {
		return InstallResult{}, err
	}
	if err := ensureMarketplaceSkillsDir(skillsDir); err != nil {
		return InstallResult{}, err
	}
	detail, err := loadMarketplaceSkillDetail(ctx, registry, normalizedSlug)
	if err != nil {
		return InstallResult{}, err
	}

	tempRoot, err := os.MkdirTemp(skillsDir, ".agh-skill-stage-*")
	if err != nil {
		return InstallResult{}, fmt.Errorf("create temporary install directory: %w", err)
	}
	defer joinRemoveAll(&err, tempRoot, "remove temporary install directory")

	installer := registrypkg.NewInstaller(registry)
	result, err := installer.Install(ctx, normalizedSlug, registrypkg.DownloadOpts{
		Version: strings.TrimSpace(version),
	}, tempRoot)
	if err != nil {
		return InstallResult{}, err
	}

	hash, err := skills.ComputeDirectoryHash(result.InstallPath)
	if err != nil {
		return InstallResult{}, fmt.Errorf(
			"compute extracted skill hash for %q: %w",
			normalizedSlug,
			err,
		)
	}

	installedAt := normalizeMarketplaceInstallTime(now)
	resolvedVersion := firstNonEmpty(result.Version, detail.Version)
	registryName := firstNonEmpty(detail.Source, DefaultRegistry)
	if err := WriteInstalledSkillProvenance(
		result.InstallPath,
		hash,
		registryName,
		normalizedSlug,
		resolvedVersion,
		installedAt,
	); err != nil {
		return InstallResult{}, err
	}

	targetDir, err := ResolveMarketplaceInstallTarget(
		skillsDir,
		result.Name,
		targetDirOverride,
	)
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve install path for %q: %w", normalizedSlug, err)
	}
	if err := registrypkg.MoveInstalledDir(result.InstallPath, targetDir, true); err != nil {
		return InstallResult{}, err
	}

	return InstallResult{
		Name:     result.Name,
		Slug:     normalizedSlug,
		Version:  resolvedVersion,
		Registry: registryName,
		Path:     targetDir,
		Hash:     hash,
		Status:   "installed",
	}, nil
}

// WriteInstalledSkillProvenance writes marketplace provenance for one installed skill.
func WriteInstalledSkillProvenance(
	installPath string,
	hash string,
	registryName string,
	slug string,
	resolvedVersion string,
	installedAt time.Time,
) error {
	if err := skills.WriteSidecar(installPath, skills.Provenance{
		Hash:        hash,
		Registry:    registryName,
		Slug:        slug,
		Version:     resolvedVersion,
		InstalledAt: installedAt,
	}); err != nil {
		return fmt.Errorf("write provenance for %q: %w", slug, err)
	}
	return nil
}

// UpdateWithRegistry checks or applies marketplace updates using the supplied registry.
func UpdateWithRegistry(
	ctx context.Context,
	skillsDir string,
	registry Registry,
	req UpdateRequest,
	now func() time.Time,
) ([]UpdateResult, error) {
	if strings.TrimSpace(req.Name) != "" && req.All {
		return nil, classifiedf(ErrValidation, "cannot combine skill name with --all")
	}
	if !req.All && strings.TrimSpace(req.Name) == "" {
		return nil, classifiedf(ErrValidation, "skill name is required unless --all is set")
	}

	if req.All {
		installedSkills, err := ListInstalledSkills(skillsDir)
		if err != nil {
			return nil, err
		}

		items := make([]UpdateResult, 0, len(installedSkills))
		for _, installed := range installedSkills {
			item, err := UpdateSkill(ctx, skillsDir, registry, installed, req.CheckOnly, now)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	}

	name, err := NormalizeSkillName(req.Name)
	if err != nil {
		return nil, err
	}

	installed, err := FindInstalledSkill(skillsDir, name)
	if err != nil {
		return nil, err
	}

	item, err := UpdateSkill(ctx, skillsDir, registry, installed, req.CheckOnly, now)
	if err != nil {
		return nil, err
	}
	return []UpdateResult{item}, nil
}

// UpdateSkill checks or applies an update for one installed marketplace skill.
func UpdateSkill(
	ctx context.Context,
	skillsDir string,
	registry Registry,
	installed InstalledSkill,
	checkOnly bool,
	now func() time.Time,
) (UpdateResult, error) {
	slug := strings.TrimSpace(installed.Provenance.Slug)
	if slug == "" {
		return UpdateResult{}, classifiedf(
			ErrValidation,
			"marketplace skill %q is missing registry slug metadata",
			installed.Name,
		)
	}
	resolvedRegistry, err := ResolveInstalledSkillRegistry(registry, installed)
	if err != nil {
		return UpdateResult{}, err
	}

	currentVersion := strings.TrimSpace(installed.Provenance.Version)
	updateInfo, err := resolvedRegistry.CheckUpdate(ctx, slug, currentVersion)
	if err != nil {
		return UpdateResult{}, err
	}
	if updateInfo == nil {
		return UpdateResult{}, classifiedf(
			ErrNotFound,
			"marketplace update check returned no result for %q",
			slug,
		)
	}

	latestVersion := strings.TrimSpace(updateInfo.LatestVersion)
	item := UpdateResult{
		Name:           installed.Name,
		Slug:           slug,
		CurrentVersion: currentVersion,
		LatestVersion:  firstNonEmpty(latestVersion, currentVersion),
		Path:           installed.Dir,
	}

	if !updateInfo.HasUpdate {
		item.Status = UpdateStatusCurrent
		return item, nil
	}
	if checkOnly {
		item.Status = UpdateStatusAvailable
		return item, nil
	}

	installedItem, err := InstallWithRegistry(
		ctx,
		skillsDir,
		resolvedRegistry,
		slug,
		latestVersion,
		installed.Dir,
		now,
	)
	if err != nil {
		return UpdateResult{}, err
	}

	item.Name = installedItem.Name
	item.LatestVersion = firstNonEmpty(installedItem.Version, latestVersion)
	item.Path = installedItem.Path
	item.Status = UpdateStatusUpdated
	return item, nil
}

// RemoveSkill removes one installed marketplace skill.
func RemoveSkill(skillsDir string, name string) (RemoveResult, error) {
	normalizedName, err := NormalizeSkillName(name)
	if err != nil {
		return RemoveResult{}, err
	}

	installed, err := FindInstalledSkill(skillsDir, normalizedName)
	if err != nil {
		return RemoveResult{}, err
	}

	if err := os.RemoveAll(installed.Dir); err != nil {
		return RemoveResult{}, fmt.Errorf("remove marketplace skill %q: %w", normalizedName, err)
	}

	return RemoveResult{
		Name:   installed.Name,
		Slug:   installed.Provenance.Slug,
		Path:   installed.Dir,
		Status: "removed",
	}, nil
}

// ResolveInstalledSkillRegistry resolves the installed skill's source registry.
func ResolveInstalledSkillRegistry(
	registry Registry,
	installed InstalledSkill,
) (Registry, error) {
	registryName := strings.TrimSpace(installed.Provenance.Registry)
	if registryName == "" {
		return nil, classifiedf(
			ErrValidation,
			"marketplace skill %q is missing registry metadata",
			installed.Name,
		)
	}

	namedRegistry, ok := registry.(NamedRegistry)
	if !ok {
		return registry, nil
	}

	source := namedRegistry.SourceNamed(registryName)
	if source == nil {
		return nil, classifiedf(
			ErrNotConfigured,
			"marketplace registry source %q is not configured for %q",
			registryName,
			installed.Name,
		)
	}

	return SourceBackedRegistry{Source: source}, nil
}

// FindInstalledSkill reads one local marketplace-installed skill.
func FindInstalledSkill(skillsDir string, name string) (InstalledSkill, error) {
	skillDir, err := registrypkg.PathWithinRoot(skillsDir, name)
	if err != nil {
		return InstalledSkill{}, fmt.Errorf("resolve skill path for %q: %w", name, err)
	}

	info, err := os.Stat(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return InstalledSkill{}, classifiedf(ErrNotFound, "skill %q not found", name)
		}
		return InstalledSkill{}, fmt.Errorf("inspect skill directory %q: %w", skillDir, err)
	}
	if !info.IsDir() {
		return InstalledSkill{}, classifiedf(ErrValidation, "skill %q is not a directory", name)
	}

	hasSidecar, err := skills.HasSidecar(skillDir)
	if err != nil {
		return InstalledSkill{}, err
	}
	if !hasSidecar {
		return InstalledSkill{}, classifiedf(
			ErrNotMarketplace,
			"skill %q is not a marketplace-installed skill",
			name,
		)
	}

	return ReadInstalledSkill(skillDir)
}

// ListInstalledSkills returns every local marketplace-installed skill.
func ListInstalledSkills(skillsDir string) ([]InstalledSkill, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []InstalledSkill{}, nil
		}
		return nil, fmt.Errorf("read installed skills directory %q: %w", skillsDir, err)
	}

	items := make([]InstalledSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir, err := registrypkg.PathWithinRoot(skillsDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf(
				"resolve installed skill path for %q: %w",
				entry.Name(),
				err,
			)
		}

		hasSidecar, err := skills.HasSidecar(skillDir)
		if err != nil {
			return nil, err
		}
		if !hasSidecar {
			continue
		}

		item, err := ReadInstalledSkill(skillDir)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

// ReadInstalledSkill parses one marketplace-installed skill directory.
func ReadInstalledSkill(skillDir string) (InstalledSkill, error) {
	provenance, err := skills.ReadSidecar(skillDir)
	if err != nil {
		return InstalledSkill{}, err
	}
	if provenance == nil {
		return InstalledSkill{}, classifiedf(ErrNotMarketplace, "missing provenance for %q", skillDir)
	}

	skillFile, err := registrypkg.PathWithinRoot(skillDir, SkillMarkdownFileName)
	if err != nil {
		return InstalledSkill{}, fmt.Errorf("resolve skill file in %q: %w", skillDir, err)
	}

	parsedSkill, err := skills.ParseSkillFile(skillFile)
	if err != nil {
		return InstalledSkill{}, err
	}

	return InstalledSkill{
		Name:       parsedSkill.Meta.Name,
		Dir:        parsedSkill.Dir,
		FilePath:   parsedSkill.FilePath,
		Provenance: *provenance,
	}, nil
}

// PathInsideRoot resolves a target path and validates it remains under root.
func PathInsideRoot(root string, target string) (string, error) {
	sanitizedRoot, err := sanitizePathKey(root)
	if err != nil {
		return "", fmt.Errorf("sanitize root %q: %w", root, err)
	}
	if sanitizedRoot == "" {
		return "", registrypkg.ErrPathRootRequired
	}

	absRoot, err := filepath.Abs(sanitizedRoot)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	resolvedRoot, err := realpathDeepestExisting(absRoot)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", absRoot, err)
	}

	sanitizedTarget, err := sanitizePathKey(target)
	if err != nil {
		return "", fmt.Errorf("sanitize target %q: %w", target, err)
	}

	absTarget, err := filepath.Abs(sanitizedTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", target, err)
	}
	resolvedTarget, err := realpathDeepestExisting(absTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", absTarget, err)
	}

	relative, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q within %q: %w", resolvedTarget, resolvedRoot, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", registrypkg.ErrPathOutsideRoot
	}
	return absTarget, nil
}

func sanitizePathKey(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return "", errors.New("path contains null byte")
	}

	decoded := trimmed
	for {
		unescaped, err := url.PathUnescape(decoded)
		if err != nil {
			return "", fmt.Errorf("decode path escapes: %w", err)
		}
		if unescaped == decoded {
			break
		}
		decoded = unescaped
	}
	if !utf8.ValidString(decoded) {
		return "", errors.New("path contains invalid UTF-8")
	}

	return norm.NFC.String(decoded), nil
}

func realpathDeepestExisting(target string) (string, error) {
	current := filepath.Clean(target)
	suffix := make([]string, 0, 4)

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

// ResolveMarketplaceInstallTarget validates the final install destination.
func ResolveMarketplaceInstallTarget(
	skillsDir string,
	parsedName string,
	targetDirOverride string,
) (string, error) {
	if trimmedOverride := strings.TrimSpace(targetDirOverride); trimmedOverride != "" {
		return PathInsideRoot(skillsDir, trimmedOverride)
	}
	return registrypkg.PathWithinRoot(skillsDir, parsedName)
}

func (s *Service) loadRegistry() (*registrypkg.MultiRegistry, error) {
	sources, err := s.sourceLoader(s.skillsConfig.Marketplace)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, classifiedf(ErrNotConfigured, "no skill registry sources are configured")
	}
	return registrypkg.NewMultiRegistry(s.logger, sources...), nil
}

func ensureMarketplaceSkillsDir(skillsDir string) error {
	if strings.TrimSpace(skillsDir) == "" {
		return classifiedf(ErrNotConfigured, "skills directory is not configured")
	}
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("create skills directory %q: %w", skillsDir, err)
	}
	return nil
}

func loadMarketplaceSkillDetail(
	ctx context.Context,
	registry Registry,
	slug string,
) (*registrypkg.Detail, error) {
	if registry == nil {
		return nil, classifiedf(ErrNotConfigured, "skill registry is required")
	}
	detail, err := registry.Info(ctx, slug)
	if err != nil {
		return nil, classifyRegistryLookup(err)
	}
	if detail == nil {
		return nil, classifiedf(ErrNotFound, "marketplace info returned no detail for %q", slug)
	}
	return detail, nil
}

func normalizeMarketplaceInstallTime(now func() time.Time) time.Time {
	if now == nil {
		return time.Now().UTC()
	}
	installedAt := now()
	if installedAt.IsZero() {
		return time.Now().UTC()
	}
	return installedAt.UTC()
}

func joinRemoveAll(errp *error, path string, label string) {
	removeErr := os.RemoveAll(path)
	if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		return
	}
	*errp = errors.Join(*errp, fmt.Errorf("%s %q: %w", label, path, removeErr))
}

func classifyRegistryLookup(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrValidation) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrNotMarketplace) ||
		errors.Is(err, ErrNotConfigured) {
		return err
	}
	message := err.Error()
	if strings.Contains(message, " not found") || strings.Contains(message, "package ") &&
		strings.Contains(message, "not found") {
		return &Error{Kind: ErrNotFound, Message: message, Cause: err}
	}
	return err
}

func classifiedf(kind error, format string, args ...any) error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
