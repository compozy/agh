package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	registrypkg "github.com/pedronauck/agh/internal/registry"
	registrygithub "github.com/pedronauck/agh/internal/registry/github"
)

const (
	defaultExtensionRegistrySearchLimit = 20
	managedExtensionsDirName            = "extensions"
	extensionInstallRestartMessage      = "Extension installed. Restart the daemon to activate, or it will be discovered on next boot."
	extensionUpdateRestartMessage       = "Extension updated. Restart the daemon to activate the new version."
)

type extensionRegistrySourceLoader func(runtimeContext) ([]registrypkg.RegistrySource, error)

type extensionRemoveItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type extensionUpdateItem struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Registry       string `json:"registry"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Path           string `json:"path"`
	Status         string `json:"status"`
}

type stagedExtensionDirChange struct {
	targetDir string
	backupDir string
}

func defaultExtensionRegistrySourceLoader(runtime runtimeContext) ([]registrypkg.RegistrySource, error) {
	cfg := runtime.Config.Extensions.Marketplace
	registryName := strings.ToLower(strings.TrimSpace(cfg.Registry))
	if registryName == "" && strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("cli: extensions marketplace is not configured")
	}

	switch registryName {
	case "github":
		return []registrypkg.RegistrySource{
			registrygithub.NewClient(cfg.BaseURL),
		}, nil
	default:
		return nil, fmt.Errorf("cli: unsupported extension registry %q", cfg.Registry)
	}
}

func searchExtensions(ctx context.Context, deps commandDeps, query string, sourceFilter string, limit int) (_ []registrypkg.Listing, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("cli: search limit must be positive: %d", limit)
	}

	runtime, sources, err := loadExtensionRegistrySources(ctx, deps, sourceFilter)
	if err != nil {
		return nil, err
	}

	registry := registrypkg.NewMultiRegistry(slog.Default(), sources...)
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	listings, err := registry.Search(ctx, query, registrypkg.SearchOpts{
		Limit: limit,
		Type:  registrypkg.PackageTypeExtension,
	})
	if err != nil {
		return nil, err
	}

	_ = runtime
	return listings, nil
}

func installMarketplaceExtension(
	ctx context.Context,
	deps commandDeps,
	slug string,
	sourceFilter string,
	version string,
	asset string,
) (_ ExtensionRecord, _ string, err error) {
	item, err := withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) (_ ExtensionRecord, err error) {
		sources, err := configuredExtensionRegistrySources(runtime, deps, sourceFilter)
		if err != nil {
			return ExtensionRecord{}, err
		}

		multi := registrypkg.NewMultiRegistry(slog.Default(), sources...)
		defer func() {
			err = errors.Join(err, multi.Close())
		}()

		detail, err := multi.Info(ctx, slug)
		if err != nil {
			return ExtensionRecord{}, err
		}

		stagingDir, err := newManagedExtensionStagingDir(runtime.HomePaths)
		if err != nil {
			return ExtensionRecord{}, err
		}
		defer func() {
			removeErr := os.RemoveAll(stagingDir)
			if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("cli: remove staged extension directory %q: %w", stagingDir, removeErr))
			}
		}()

		installer := registrypkg.NewInstaller(multi)
		result, err := installer.Install(ctx, slug, registrypkg.DownloadOpts{
			Version: strings.TrimSpace(version),
			Asset:   strings.TrimSpace(asset),
		}, stagingDir)
		if err != nil {
			return ExtensionRecord{}, err
		}

		manifest, err := extensionpkg.LoadManifest(result.InstallPath)
		if err != nil {
			return ExtensionRecord{}, fmt.Errorf("cli: load installed extension manifest for %q: %w", slug, err)
		}

		if _, getErr := registry.Get(manifest.Name); getErr == nil {
			return ExtensionRecord{}, &extensionpkg.ExtensionExistsError{Name: manifest.Name}
		} else if !errors.Is(getErr, extensionpkg.ErrExtensionNotFound) {
			return ExtensionRecord{}, getErr
		}

		finalDir := managedExtensionInstallPath(runtime.HomePaths, manifest.Name)
		if err := registrypkg.MoveInstalledDir(result.InstallPath, finalDir, false); err != nil {
			return ExtensionRecord{}, fmt.Errorf("cli: move extension %q into managed install path: %w", manifest.Name, err)
		}

		remoteVersion := firstNonEmpty(result.Version, detail.Version, manifest.Version)
		registryName := firstNonEmpty(detail.Source, strings.TrimSpace(sourceFilter))
		if err := registry.Install(
			manifest,
			finalDir,
			result.Checksum,
			extensionpkg.WithInstallSource(extensionpkg.SourceMarketplace),
			extensionpkg.WithInstallRegistryMetadata(slug, registryName, remoteVersion),
		); err != nil {
			cleanupErr := os.RemoveAll(finalDir)
			if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("cli: remove failed extension install %q: %w", finalDir, cleanupErr))
			}
			return ExtensionRecord{}, err
		}

		info, err := registry.Get(manifest.Name)
		if err != nil {
			return ExtensionRecord{}, err
		}

		return localExtensionRecord(*info, deps.now), nil
	})
	if err != nil {
		return ExtensionRecord{}, "", err
	}
	return item, extensionInstallRestartMessage, nil
}

func removeInstalledExtension(ctx context.Context, deps commandDeps, name string) (_ extensionRemoveItem, err error) {
	return withLocalExtensionRegistry(ctx, deps, func(_ runtimeContext, registry localExtensionRegistry) (_ extensionRemoveItem, err error) {
		info, err := registry.Get(name)
		if err != nil {
			return extensionRemoveItem{}, err
		}

		installDir := filepath.Dir(info.ManifestPath)
		change, err := stageExtensionDirRemoval(installDir)
		if err != nil {
			return extensionRemoveItem{}, err
		}

		if err := registry.Uninstall(info.Name); err != nil {
			rollbackErr := change.Rollback()
			return extensionRemoveItem{}, errors.Join(err, rollbackErr)
		}
		if err := change.Commit(); err != nil {
			return extensionRemoveItem{}, err
		}

		return extensionRemoveItem{
			Name:   info.Name,
			Path:   installDir,
			Status: "removed",
		}, nil
	})
}

func updateMarketplaceExtensions(
	ctx context.Context,
	deps commandDeps,
	args []string,
	updateAll bool,
	checkOnly bool,
) (_ []extensionUpdateItem, err error) {
	return withLocalExtensionRegistry(ctx, deps, func(runtime runtimeContext, registry localExtensionRegistry) (_ []extensionUpdateItem, err error) {
		targets, err := selectMarketplaceExtensionsForUpdate(registry, args, updateAll)
		if err != nil {
			return nil, err
		}

		items := make([]extensionUpdateItem, 0, len(targets))
		for _, info := range targets {
			item, err := updateMarketplaceExtension(ctx, runtime, deps, registry, info, checkOnly)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}

		return items, nil
	})
}

func updateMarketplaceExtension(
	ctx context.Context,
	runtime runtimeContext,
	deps commandDeps,
	registry localExtensionRegistry,
	info extensionpkg.ExtensionInfo,
	checkOnly bool,
) (_ extensionUpdateItem, err error) {
	slug := dereferenceOptionalString(info.RegistrySlug)
	if slug == "" {
		return extensionUpdateItem{}, fmt.Errorf("cli: extension %q is missing registry slug metadata", info.Name)
	}
	registryName := dereferenceOptionalString(info.RegistryName)
	if registryName == "" {
		return extensionUpdateItem{}, fmt.Errorf("cli: extension %q is missing registry source metadata", info.Name)
	}

	sources, err := configuredExtensionRegistrySources(runtime, deps, registryName)
	if err != nil {
		return extensionUpdateItem{}, err
	}

	multi := registrypkg.NewMultiRegistry(slog.Default(), sources...)
	defer func() {
		err = errors.Join(err, multi.Close())
	}()

	currentVersion := firstNonEmpty(dereferenceOptionalString(info.RemoteVersion), info.Version)
	updateInfo, err := multi.CheckUpdate(ctx, slug, currentVersion)
	if err != nil {
		return extensionUpdateItem{}, err
	}

	item := extensionUpdateItem{
		Name:           info.Name,
		Slug:           slug,
		Registry:       registryName,
		CurrentVersion: currentVersion,
		LatestVersion:  firstNonEmpty(updateInfo.LatestVersion, currentVersion),
		Path:           filepath.Dir(info.ManifestPath),
	}

	if !updateInfo.HasUpdate {
		item.Status = "already up to date"
		return item, nil
	}
	if checkOnly {
		item.Status = "update available"
		return item, nil
	}

	stagingDir, err := newManagedExtensionStagingDir(runtime.HomePaths)
	if err != nil {
		return extensionUpdateItem{}, err
	}
	defer func() {
		removeErr := os.RemoveAll(stagingDir)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("cli: remove staged extension directory %q: %w", stagingDir, removeErr))
		}
	}()

	installer := registrypkg.NewInstaller(multi)
	result, err := installer.Install(ctx, slug, registrypkg.DownloadOpts{
		Version: strings.TrimSpace(updateInfo.LatestVersion),
	}, stagingDir)
	if err != nil {
		return extensionUpdateItem{}, err
	}

	manifest, err := extensionpkg.LoadManifest(result.InstallPath)
	if err != nil {
		return extensionUpdateItem{}, fmt.Errorf("cli: load updated extension manifest for %q: %w", info.Name, err)
	}
	if manifest.Name != info.Name {
		return extensionUpdateItem{}, fmt.Errorf(
			"cli: extension update identity mismatch: installed %q, registry returned %q",
			info.Name,
			manifest.Name,
		)
	}

	change, err := stageExtensionDirReplacement(result.InstallPath, item.Path)
	if err != nil {
		return extensionUpdateItem{}, err
	}

	remoteVersion := firstNonEmpty(result.Version, updateInfo.LatestVersion, manifest.Version)
	if err := registry.Install(
		manifest,
		item.Path,
		result.Checksum,
		extensionpkg.WithInstallSource(extensionpkg.SourceMarketplace),
		extensionpkg.WithInstallRegistryMetadata(slug, registryName, remoteVersion),
		extensionpkg.WithInstallReplaceExisting(),
	); err != nil {
		rollbackErr := change.Rollback()
		return extensionUpdateItem{}, errors.Join(err, rollbackErr)
	}
	if err := change.Commit(); err != nil {
		return extensionUpdateItem{}, err
	}

	item.LatestVersion = remoteVersion
	item.Status = "updated"
	return item, nil
}

func loadExtensionRegistrySources(
	_ context.Context,
	deps commandDeps,
	sourceFilter string,
) (runtimeContext, []registrypkg.RegistrySource, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return runtimeContext{}, nil, err
	}

	sources, err := configuredExtensionRegistrySources(runtime, deps, sourceFilter)
	if err != nil {
		return runtimeContext{}, nil, err
	}
	return runtime, sources, nil
}

func configuredExtensionRegistrySources(
	runtime runtimeContext,
	deps commandDeps,
	sourceFilter string,
) ([]registrypkg.RegistrySource, error) {
	loader := deps.loadExtensionRegistrySources
	if loader == nil {
		loader = defaultExtensionRegistrySourceLoader
	}

	sources, err := loader(runtime)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errors.New("cli: no extension registry sources are configured")
	}

	filtered := filterExtensionRegistrySources(sources, sourceFilter)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("cli: extension registry source %q is not configured", sourceFilter)
	}
	return filtered, nil
}

func filterExtensionRegistrySources(sources []registrypkg.RegistrySource, sourceFilter string) []registrypkg.RegistrySource {
	filter := strings.ToLower(strings.TrimSpace(sourceFilter))
	if filter == "" {
		return sources
	}

	filtered := make([]registrypkg.RegistrySource, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(source.Name()), filter) {
			filtered = append(filtered, source)
		}
	}
	return filtered
}

func selectMarketplaceExtensionsForUpdate(
	registry localExtensionRegistry,
	args []string,
	updateAll bool,
) ([]extensionpkg.ExtensionInfo, error) {
	if updateAll {
		infos, err := registry.List()
		if err != nil {
			return nil, err
		}

		items := make([]extensionpkg.ExtensionInfo, 0, len(infos))
		for _, info := range infos {
			if marketplaceExtensionInstalled(info) {
				items = append(items, info)
			}
		}
		return items, nil
	}

	info, err := registry.Get(args[0])
	if err != nil {
		return nil, err
	}
	if !marketplaceExtensionInstalled(*info) {
		return nil, fmt.Errorf("cli: extension %q is not a marketplace-installed extension", info.Name)
	}

	return []extensionpkg.ExtensionInfo{*info}, nil
}

func marketplaceExtensionInstalled(info extensionpkg.ExtensionInfo) bool {
	return info.Source == extensionpkg.SourceMarketplace && dereferenceOptionalString(info.RegistrySlug) != ""
}

func newManagedExtensionStagingDir(homePaths aghconfig.HomePaths) (string, error) {
	root := managedExtensionsRoot(homePaths)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("cli: create managed extensions root %q: %w", root, err)
	}
	return os.MkdirTemp(root, ".agh-extension-stage-*")
}

func managedExtensionsRoot(homePaths aghconfig.HomePaths) string {
	return filepath.Join(homePaths.HomeDir, managedExtensionsDirName)
}

func managedExtensionInstallPath(homePaths aghconfig.HomePaths, name string) string {
	return filepath.Join(managedExtensionsRoot(homePaths), strings.TrimSpace(name))
}

func stageExtensionDirRemoval(targetDir string) (*stagedExtensionDirChange, error) {
	change := &stagedExtensionDirChange{
		targetDir: targetDir,
	}

	info, err := os.Stat(targetDir)
	if errors.Is(err, os.ErrNotExist) {
		return change, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cli: stat extension directory %q: %w", targetDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("cli: extension path %q is not a directory", targetDir)
	}

	backupDir := uniqueExtensionBackupPath(targetDir)
	if err := os.Rename(targetDir, backupDir); err != nil {
		return nil, fmt.Errorf("cli: stage extension directory removal %q: %w", targetDir, err)
	}
	change.backupDir = backupDir
	return change, nil
}

func stageExtensionDirReplacement(stagingDir string, targetDir string) (*stagedExtensionDirChange, error) {
	change := &stagedExtensionDirChange{
		targetDir: targetDir,
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return nil, fmt.Errorf("cli: create extension install parent %q: %w", filepath.Dir(targetDir), err)
	}

	if _, err := os.Stat(targetDir); err == nil {
		backupDir := uniqueExtensionBackupPath(targetDir)
		if err := os.Rename(targetDir, backupDir); err != nil {
			return nil, fmt.Errorf("cli: backup extension directory %q: %w", targetDir, err)
		}
		change.backupDir = backupDir
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("cli: stat extension directory %q: %w", targetDir, err)
	}

	if err := os.Rename(stagingDir, targetDir); err != nil {
		rollbackErr := change.Rollback()
		return nil, errors.Join(
			fmt.Errorf("cli: move staged extension into %q: %w", targetDir, err),
			rollbackErr,
		)
	}

	return change, nil
}

func (c *stagedExtensionDirChange) Commit() error {
	if c == nil || strings.TrimSpace(c.backupDir) == "" {
		return nil
	}
	if err := os.RemoveAll(c.backupDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cli: remove extension backup %q: %w", c.backupDir, err)
	}
	return nil
}

func (c *stagedExtensionDirChange) Rollback() error {
	if c == nil {
		return nil
	}

	if strings.TrimSpace(c.targetDir) != "" {
		if err := os.RemoveAll(c.targetDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cli: remove failed extension target %q: %w", c.targetDir, err)
		}
	}
	if strings.TrimSpace(c.backupDir) == "" {
		return nil
	}
	if err := os.Rename(c.backupDir, c.targetDir); err != nil {
		return fmt.Errorf("cli: restore extension backup %q: %w", c.targetDir, err)
	}
	return nil
}

func uniqueExtensionBackupPath(targetDir string) string {
	return fmt.Sprintf("%s.agh-backup-%d", targetDir, time.Now().UTC().UnixNano())
}

func dereferenceOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func extensionSearchBundle(items []registrypkg.Listing) outputBundle {
	return listBundle(
		items,
		items,
		"Extension Registry Results",
		[]string{"Slug", "Name", "Description", "Author", "Version", "Downloads", "Source"},
		"extensions",
		[]string{"slug", "name", "description", "author", "version", "downloads", "source"},
		func(item registrypkg.Listing) []string {
			return []string{
				stringOrDash(item.Slug),
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Author),
				stringOrDash(item.Version),
				strconv.Itoa(item.Downloads),
				stringOrDash(item.Source),
			}
		},
		func(item registrypkg.Listing) []string {
			return []string{
				item.Slug,
				item.Name,
				item.Description,
				item.Author,
				item.Version,
				strconv.Itoa(item.Downloads),
				item.Source,
			}
		},
	)
}

func extensionRemoveBundle(item extensionRemoveItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Extension Remove", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("extension_remove", []string{"name", "path", "status"}, []string{
				item.Name,
				item.Path,
				item.Status,
			}), nil
		},
	}
}

func extensionUpdateBundle(items []extensionUpdateItem) outputBundle {
	return listBundle(
		items,
		items,
		"Extension Updates",
		[]string{"Name", "Slug", "Registry", "Current", "Latest", "Path", "Status"},
		"extension_updates",
		[]string{"name", "slug", "registry", "current_version", "latest_version", "path", "status"},
		func(item extensionUpdateItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Slug),
				stringOrDash(item.Registry),
				stringOrDash(item.CurrentVersion),
				stringOrDash(item.LatestVersion),
				stringOrDash(item.Path),
				stringOrDash(item.Status),
			}
		},
		func(item extensionUpdateItem) []string {
			return []string{
				item.Name,
				item.Slug,
				item.Registry,
				item.CurrentVersion,
				item.LatestVersion,
				item.Path,
				item.Status,
			}
		},
	)
}
