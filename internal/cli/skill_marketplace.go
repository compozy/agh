package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	registrypkg "github.com/compozy/agh/internal/registry"
	registryclawhub "github.com/compozy/agh/internal/registry/clawhub"
	"github.com/compozy/agh/internal/skills"
	skillmarketplace "github.com/compozy/agh/internal/skills/marketplace"
	skillbundled "github.com/compozy/agh/skills"
)

const (
	skillUpdateStatusCurrent   = "already up to date"
	skillUpdateStatusAvailable = "update available"
	skillUpdateStatusUpdated   = "updated"
)

type skillRegistrySourceLoader func(*runtimeContext) ([]registrypkg.Source, error)

type skillRegistry interface {
	skillmarketplace.Registry
}

type installedMarketplaceSkill = skillmarketplace.InstalledSkill

func defaultSkillRegistrySourceLoader(
	runtime *runtimeContext,
) ([]registrypkg.Source, error) {
	registryCfg := runtime.Config.Skills.Marketplace
	registryName := strings.ToLower(strings.TrimSpace(registryCfg.Registry))
	if registryName == "" {
		registryName = defaultMarketplaceRegistry
	}

	switch registryName {
	case defaultMarketplaceRegistry:
		return []registrypkg.Source{
			registryclawhub.NewClient(registryCfg.BaseURL),
		}, nil
	default:
		return nil, fmt.Errorf("cli: unsupported marketplace registry %q", registryCfg.Registry)
	}
}

func loadSkillRegistry(deps commandDeps) (*runtimeContext, *registrypkg.MultiRegistry, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, nil, err
	}

	loader := deps.loadSkillRegistrySources
	if loader == nil {
		loader = defaultSkillRegistrySourceLoader
	}

	sources, err := loader(runtime)
	if err != nil {
		return nil, nil, err
	}
	if len(sources) == 0 {
		return nil, nil, errors.New("cli: no skill registry sources are configured")
	}

	return runtime, registrypkg.NewMultiRegistry(slog.Default(), sources...), nil
}

func normalizeSkillSlug(slug string) (string, error) {
	return skillmarketplace.NormalizeSkillSlug(slug)
}

func installMarketplaceSkill(
	ctx context.Context,
	runtime *runtimeContext,
	registry skillRegistry,
	slug string,
	version string,
	targetDirOverride string,
	now func() time.Time,
) (item skillInstallItem, err error) {
	result, err := skillmarketplace.InstallWithRegistry(
		ctx,
		runtime.HomePaths.SkillsDir,
		registry,
		slug,
		version,
		targetDirOverride,
		now,
	)
	if err != nil {
		return skillInstallItem{}, err
	}
	if err := verifyInstalledMarketplaceSkill(ctx, runtime, result); err != nil {
		return skillInstallItem{}, err
	}
	return skillInstallItem{
		Name:     result.Name,
		Slug:     result.Slug,
		Version:  result.Version,
		Registry: result.Registry,
		Path:     result.Path,
		Hash:     result.Hash,
		Status:   result.Status,
	}, nil
}

func verifyInstalledMarketplaceSkill(
	ctx context.Context,
	runtime *runtimeContext,
	result skillmarketplace.InstallResult,
) error {
	if runtime == nil {
		return errors.New("cli: skill runtime context is required")
	}

	registry := skills.NewRegistry(skills.RegistryConfig{
		BundledFS:      skillbundled.FS(),
		UserAgentsDir:  runtime.HomePaths.AgentsDir,
		UserSkillsDir:  runtime.HomePaths.SkillsDir,
		DisabledSkills: append([]string(nil), runtime.Config.Skills.DisabledSkills...),
	})
	if err := registry.LoadAll(ctx); err != nil {
		return fmt.Errorf("cli: reload skill registry after marketplace install %q: %w", result.Name, err)
	}

	skill, err := findSkillByName(registry.List(), result.Name)
	if err != nil {
		return fmt.Errorf(
			"cli: installed marketplace skill %q is not visible after registry reload; inspect %s and retry the install: %w",
			result.Name,
			result.Path,
			err,
		)
	}
	if skill.Source != skills.SourceMarketplace {
		return fmt.Errorf(
			"cli: installed marketplace skill %q resolved as %s after registry reload; "+
				"remove the shadowing skill and retry the install",
			result.Name,
			skillSourceLabel(skill.Source),
		)
	}
	if skill.Provenance == nil {
		return fmt.Errorf(
			"cli: installed marketplace skill %q is missing provenance after registry reload; inspect %s and retry the install",
			result.Name,
			result.Path,
		)
	}
	if strings.TrimSpace(skill.Provenance.Slug) != strings.TrimSpace(result.Slug) {
		return fmt.Errorf(
			"cli: installed marketplace skill %q resolved slug %q after registry reload, want %q; "+
				"remove the conflicting skill and retry the install",
			result.Name,
			skill.Provenance.Slug,
			result.Slug,
		)
	}
	if !skill.Enabled {
		return fmt.Errorf(
			"cli: installed marketplace skill %q is visible but disabled after registry reload; "+
				"enable the skill and retry discovery",
			result.Name,
		)
	}
	return nil
}

func updateMarketplaceSkills(
	ctx context.Context,
	runtime *runtimeContext,
	registry skillRegistry,
	args []string,
	updateAll bool,
	checkOnly bool,
	now func() time.Time,
) ([]skillUpdateItem, error) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	results, err := skillmarketplace.UpdateWithRegistry(
		ctx,
		runtime.HomePaths.SkillsDir,
		registry,
		skillmarketplace.UpdateRequest{
			Name:      name,
			All:       updateAll,
			CheckOnly: checkOnly,
		},
		now,
	)
	if err != nil {
		return nil, err
	}
	items := make([]skillUpdateItem, 0, len(results))
	for _, result := range results {
		items = append(items, skillUpdateItem{
			Name:           result.Name,
			Slug:           result.Slug,
			CurrentVersion: result.CurrentVersion,
			LatestVersion:  result.LatestVersion,
			Path:           result.Path,
			Status:         result.Status,
		})
	}
	return items, nil
}

func updateMarketplaceSkill(
	ctx context.Context,
	runtime *runtimeContext,
	registry skillRegistry,
	installed installedMarketplaceSkill,
	checkOnly bool,
	now func() time.Time,
) (skillUpdateItem, error) {
	result, err := skillmarketplace.UpdateSkill(
		ctx,
		runtime.HomePaths.SkillsDir,
		registry,
		installed,
		checkOnly,
		now,
	)
	if err != nil {
		return skillUpdateItem{}, err
	}
	return skillUpdateItem{
		Name:           result.Name,
		Slug:           result.Slug,
		CurrentVersion: result.CurrentVersion,
		LatestVersion:  result.LatestVersion,
		Path:           result.Path,
		Status:         result.Status,
	}, nil
}

func searchMarketplaceSkills(
	ctx context.Context,
	deps commandDeps,
	query string,
	limit int,
) (_ []registrypkg.Listing, err error) {
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
	result, err := skillmarketplace.RemoveSkill(skillsDir, name)
	if err != nil {
		return skillRemoveItem{}, err
	}
	return skillRemoveItem{
		Name:   result.Name,
		Slug:   result.Slug,
		Path:   result.Path,
		Status: result.Status,
	}, nil
}

func findInstalledMarketplaceSkill(
	skillsDir string,
	name string,
) (installedMarketplaceSkill, error) {
	return skillmarketplace.FindInstalledSkill(skillsDir, name)
}

func listInstalledMarketplaceSkills(skillsDir string) ([]installedMarketplaceSkill, error) {
	return skillmarketplace.ListInstalledSkills(skillsDir)
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
