package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	registrypkg "github.com/pedronauck/agh/internal/registry"
	registryclawhub "github.com/pedronauck/agh/internal/registry/clawhub"
	"github.com/pedronauck/agh/internal/skills"
)

type skillRegistrySourceLoader func(runtimeContext) ([]registrypkg.RegistrySource, error)

type skillRegistry interface {
	registrypkg.Downloader
	Info(ctx context.Context, slug string) (*registrypkg.Detail, error)
	CheckUpdate(ctx context.Context, slug string, currentVersion string) (*registrypkg.UpdateInfo, error)
}

type namedSkillRegistry interface {
	skillRegistry
	SourceNamed(name string) registrypkg.RegistrySource
}

type installedMarketplaceSkill struct {
	Name       string
	Dir        string
	FilePath   string
	Provenance skills.Provenance
}

type sourceBackedSkillRegistry struct {
	source registrypkg.RegistrySource
}

func defaultSkillRegistrySourceLoader(runtime runtimeContext) ([]registrypkg.RegistrySource, error) {
	registryCfg := runtime.Config.Skills.Marketplace
	registryName := strings.ToLower(strings.TrimSpace(registryCfg.Registry))
	if registryName == "" {
		registryName = defaultMarketplaceRegistry
	}

	switch registryName {
	case defaultMarketplaceRegistry:
		return []registrypkg.RegistrySource{
			registryclawhub.NewClient(registryCfg.BaseURL),
		}, nil
	default:
		return nil, fmt.Errorf("cli: unsupported marketplace registry %q", registryCfg.Registry)
	}
}

func loadSkillRegistry(deps commandDeps) (runtimeContext, *registrypkg.MultiRegistry, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return runtimeContext{}, nil, err
	}

	loader := deps.loadSkillRegistrySources
	if loader == nil {
		loader = defaultSkillRegistrySourceLoader
	}

	sources, err := loader(runtime)
	if err != nil {
		return runtimeContext{}, nil, err
	}
	if len(sources) == 0 {
		return runtimeContext{}, nil, errors.New("cli: no skill registry sources are configured")
	}

	return runtime, registrypkg.NewMultiRegistry(slog.Default(), sources...), nil
}

func normalizeSkillSlug(slug string) (string, error) {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return "", errors.New("skill slug is required")
	}
	if !validSkillSlugPattern.MatchString(trimmed) {
		return "", errors.New(`skill slug must match "@author/name"`)
	}
	return trimmed, nil
}

func (r sourceBackedSkillRegistry) Download(ctx context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
	if r.source == nil {
		return nil, errors.New("cli: registry source is required")
	}
	return r.source.Download(ctx, slug, opts)
}

func (r sourceBackedSkillRegistry) Info(ctx context.Context, slug string) (*registrypkg.Detail, error) {
	if r.source == nil {
		return nil, errors.New("cli: registry source is required")
	}

	detail, err := r.source.Info(ctx, slug)
	if err != nil {
		return nil, err
	}
	if detail != nil && strings.TrimSpace(detail.Source) == "" {
		detail.Source = strings.TrimSpace(r.source.Name())
	}
	return detail, nil
}

func (r sourceBackedSkillRegistry) CheckUpdate(ctx context.Context, slug string, currentVersion string) (*registrypkg.UpdateInfo, error) {
	detail, err := r.Info(ctx, slug)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, fmt.Errorf("cli: marketplace info returned no detail for %q", slug)
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

func resolveInstalledSkillRegistry(registry skillRegistry, installed installedMarketplaceSkill) (skillRegistry, error) {
	registryName := strings.TrimSpace(installed.Provenance.Registry)
	if registryName == "" {
		return nil, fmt.Errorf("cli: marketplace skill %q is missing registry metadata", installed.Name)
	}

	namedRegistry, ok := registry.(namedSkillRegistry)
	if !ok {
		return registry, nil
	}

	source := namedRegistry.SourceNamed(registryName)
	if source == nil {
		return nil, fmt.Errorf("cli: marketplace registry source %q is not configured for %q", registryName, installed.Name)
	}

	return sourceBackedSkillRegistry{source: source}, nil
}

func installMarketplaceSkill(
	ctx context.Context,
	runtime runtimeContext,
	registry skillRegistry,
	slug string,
	version string,
	targetDirOverride string,
	now func() time.Time,
) (item skillInstallItem, err error) {
	if err := os.MkdirAll(runtime.HomePaths.SkillsDir, 0o755); err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: create skills directory %q: %w", runtime.HomePaths.SkillsDir, err)
	}

	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}

	detail, err := registry.Info(ctx, slug)
	if err != nil {
		return skillInstallItem{}, err
	}
	if detail == nil {
		return skillInstallItem{}, fmt.Errorf("cli: marketplace info returned no detail for %q", slug)
	}

	tempRoot, err := os.MkdirTemp(runtime.HomePaths.SkillsDir, ".agh-skill-stage-*")
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: create temporary install directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempRoot)
	}()

	installer := registrypkg.NewInstaller(registry)
	result, err := installer.Install(ctx, slug, registrypkg.DownloadOpts{
		Version: strings.TrimSpace(version),
	}, tempRoot)
	if err != nil {
		return skillInstallItem{}, err
	}

	hash, err := skills.ComputeDirectoryHash(result.InstallPath)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: compute extracted skill hash for %q: %w", slug, err)
	}

	installedAt := now()
	if installedAt.IsZero() {
		installedAt = time.Now().UTC()
	} else {
		installedAt = installedAt.UTC()
	}
	resolvedVersion := firstNonEmpty(result.Version, detail.Version)
	registryName := firstNonEmpty(detail.Source, defaultMarketplaceRegistry)
	if err := skills.WriteSidecar(result.InstallPath, skills.Provenance{
		Hash:        hash,
		Registry:    registryName,
		Slug:        slug,
		Version:     resolvedVersion,
		InstalledAt: installedAt,
	}); err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: write provenance for %q: %w", slug, err)
	}

	targetDir, err := resolveMarketplaceInstallTarget(runtime.HomePaths.SkillsDir, result.Name, targetDirOverride)
	if err != nil {
		return skillInstallItem{}, fmt.Errorf("cli: resolve install path for %q: %w", slug, err)
	}
	if err := moveInstalledSkillDir(result.InstallPath, targetDir, true); err != nil {
		return skillInstallItem{}, err
	}

	return skillInstallItem{
		Name:     result.Name,
		Slug:     slug,
		Version:  resolvedVersion,
		Registry: registryName,
		Path:     targetDir,
		Hash:     hash,
		Status:   "installed",
	}, nil
}

func updateMarketplaceSkills(
	ctx context.Context,
	runtime runtimeContext,
	registry skillRegistry,
	args []string,
	updateAll bool,
	checkOnly bool,
	now func() time.Time,
) ([]skillUpdateItem, error) {
	if updateAll {
		installedSkills, err := listInstalledMarketplaceSkills(runtime.HomePaths.SkillsDir)
		if err != nil {
			return nil, err
		}

		items := make([]skillUpdateItem, 0, len(installedSkills))
		for _, installed := range installedSkills {
			item, err := updateMarketplaceSkill(ctx, runtime, registry, installed, checkOnly, now)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	}

	name, err := normalizeSkillName(args[0])
	if err != nil {
		return nil, err
	}

	installed, err := findInstalledMarketplaceSkill(runtime.HomePaths.SkillsDir, name)
	if err != nil {
		return nil, err
	}

	item, err := updateMarketplaceSkill(ctx, runtime, registry, installed, checkOnly, now)
	if err != nil {
		return nil, err
	}
	return []skillUpdateItem{item}, nil
}

func updateMarketplaceSkill(
	ctx context.Context,
	runtime runtimeContext,
	registry skillRegistry,
	installed installedMarketplaceSkill,
	checkOnly bool,
	now func() time.Time,
) (skillUpdateItem, error) {
	slug := strings.TrimSpace(installed.Provenance.Slug)
	if slug == "" {
		return skillUpdateItem{}, fmt.Errorf("cli: marketplace skill %q is missing registry slug metadata", installed.Name)
	}
	resolvedRegistry, err := resolveInstalledSkillRegistry(registry, installed)
	if err != nil {
		return skillUpdateItem{}, err
	}

	currentVersion := strings.TrimSpace(installed.Provenance.Version)
	updateInfo, err := resolvedRegistry.CheckUpdate(ctx, slug, currentVersion)
	if err != nil {
		return skillUpdateItem{}, err
	}
	if updateInfo == nil {
		return skillUpdateItem{}, fmt.Errorf("cli: marketplace update check returned no result for %q", slug)
	}

	latestVersion := strings.TrimSpace(updateInfo.LatestVersion)
	item := skillUpdateItem{
		Name:           installed.Name,
		Slug:           slug,
		CurrentVersion: currentVersion,
		LatestVersion:  firstNonEmpty(latestVersion, currentVersion),
		Path:           installed.Dir,
	}

	if !updateInfo.HasUpdate {
		item.Status = "already up to date"
		return item, nil
	}
	if checkOnly {
		item.Status = "update available"
		return item, nil
	}

	installedItem, err := installMarketplaceSkill(ctx, runtime, resolvedRegistry, slug, latestVersion, installed.Dir, now)
	if err != nil {
		return skillUpdateItem{}, err
	}

	item.Name = installedItem.Name
	item.LatestVersion = firstNonEmpty(installedItem.Version, latestVersion)
	item.Path = installedItem.Path
	item.Status = "updated"
	return item, nil
}

func searchMarketplaceSkills(ctx context.Context, deps commandDeps, query string, limit int) (_ []registrypkg.Listing, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("cli: search limit must be positive: %d", limit)
	}

	_, registry, err := loadSkillRegistry(deps)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	listings, err := registry.Search(ctx, query, registrypkg.SearchOpts{
		Limit: limit,
		Type:  registrypkg.PackageTypeSkill,
	})
	if err != nil {
		return nil, err
	}

	return listings, nil
}

func versionIsNewer(current string, latest string) bool {
	return registrypkg.VersionIsNewer(current, latest)
}

func removeMarketplaceSkill(skillsDir string, name string) (skillRemoveItem, error) {
	installed, err := findInstalledMarketplaceSkill(skillsDir, name)
	if err != nil {
		return skillRemoveItem{}, err
	}

	if err := os.RemoveAll(installed.Dir); err != nil {
		return skillRemoveItem{}, fmt.Errorf("cli: remove marketplace skill %q: %w", name, err)
	}

	return skillRemoveItem{
		Name:   installed.Name,
		Slug:   installed.Provenance.Slug,
		Path:   installed.Dir,
		Status: "removed",
	}, nil
}

func findInstalledMarketplaceSkill(skillsDir string, name string) (installedMarketplaceSkill, error) {
	skillDir, err := pathWithinRoot(skillsDir, name)
	if err != nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: resolve skill path for %q: %w", name, err)
	}

	info, err := os.Stat(skillDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return installedMarketplaceSkill{}, fmt.Errorf("skill %q not found", name)
		}
		return installedMarketplaceSkill{}, fmt.Errorf("cli: inspect skill directory %q: %w", skillDir, err)
	}
	if !info.IsDir() {
		return installedMarketplaceSkill{}, fmt.Errorf("skill %q is not a directory", name)
	}

	hasSidecar, err := skills.HasSidecar(skillDir)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}
	if !hasSidecar {
		return installedMarketplaceSkill{}, fmt.Errorf("skill %q is not a marketplace-installed skill", name)
	}

	return readInstalledMarketplaceSkill(skillDir)
}

func listInstalledMarketplaceSkills(skillsDir string) ([]installedMarketplaceSkill, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []installedMarketplaceSkill{}, nil
		}
		return nil, fmt.Errorf("cli: read installed skills directory %q: %w", skillsDir, err)
	}

	items := make([]installedMarketplaceSkill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir, err := pathWithinRoot(skillsDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("cli: resolve installed skill path for %q: %w", entry.Name(), err)
		}

		hasSidecar, err := skills.HasSidecar(skillDir)
		if err != nil {
			return nil, err
		}
		if !hasSidecar {
			continue
		}

		item, err := readInstalledMarketplaceSkill(skillDir)
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

func readInstalledMarketplaceSkill(skillDir string) (installedMarketplaceSkill, error) {
	provenance, err := skills.ReadSidecar(skillDir)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}
	if provenance == nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: missing provenance for %q", skillDir)
	}

	skillFile, err := pathWithinRoot(skillDir, skillMarkdownFileName)
	if err != nil {
		return installedMarketplaceSkill{}, fmt.Errorf("cli: resolve skill file in %q: %w", skillDir, err)
	}

	parsedSkill, err := skills.ParseSkillFile(skillFile)
	if err != nil {
		return installedMarketplaceSkill{}, err
	}

	return installedMarketplaceSkill{
		Name:       parsedSkill.Meta.Name,
		Dir:        parsedSkill.Dir,
		FilePath:   parsedSkill.FilePath,
		Provenance: *provenance,
	}, nil
}

func extractMarketplaceArchive(reader io.Reader, destRoot string) error {
	return registrypkg.ExtractArchive(reader, destRoot)
}

func locateExtractedSkillFile(root string) (string, error) {
	var matches []string

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != skillMarkdownFileName {
			return nil
		}
		matches = append(matches, current)
		if len(matches) > 1 {
			return errors.New("multiple SKILL.md files found in archive")
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", errors.New("archive did not contain SKILL.md")
	}
	return matches[0], nil
}

func moveInstalledSkillDir(extractedDir string, targetDir string, replaceExisting bool) error {
	return registrypkg.MoveInstalledDir(extractedDir, targetDir, replaceExisting)
}

func cleanArchiveEntryPath(entry string) (string, error) {
	return registrypkg.CleanArchiveEntryPath(entry)
}

func pathWithinRoot(root string, child string) (string, error) {
	return registrypkg.PathWithinRoot(root, child)
}

func pathInsideRoot(root string, target string) (string, error) {
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}

	absTarget, err := filepath.Abs(strings.TrimSpace(target))
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", target, err)
	}

	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q within %q: %w", absTarget, absRoot, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay within the root directory")
	}
	return absTarget, nil
}

func resolveMarketplaceInstallTarget(skillsDir string, parsedName string, targetDirOverride string) (string, error) {
	if trimmedOverride := strings.TrimSpace(targetDirOverride); trimmedOverride != "" {
		return pathInsideRoot(skillsDir, trimmedOverride)
	}
	return pathWithinRoot(skillsDir, parsedName)
}

func criticalWarnings(warnings []skills.Warning) []string {
	items := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning.Severity != skills.SeverityCritical {
			continue
		}
		items = append(items, firstNonEmpty(warning.Message, warning.Pattern))
	}
	return items
}
