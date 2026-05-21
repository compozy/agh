package extensionpkg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

const (
	// MarketplaceUpdateStatusCurrent reports that no remote update is available.
	MarketplaceUpdateStatusCurrent = "current"
	// MarketplaceUpdateStatusAvailable reports that a remote update exists but was not applied.
	MarketplaceUpdateStatusAvailable = "available"
	// MarketplaceUpdateStatusUpdated reports that a remote update was applied.
	MarketplaceUpdateStatusUpdated = "updated"
)

// LifecycleRegistry is the installed-extension persistence surface required by
// managed lifecycle helpers.
type LifecycleRegistry interface {
	Get(name string) (*ExtensionInfo, error)
	List() ([]ExtensionInfo, error)
	Install(manifest *Manifest, path string, checksum string, opts ...InstallOption) error
	Disable(name string) error
	Uninstall(name string) error
}

// MarketplaceSourceLoader resolves configured marketplace sources. The
// optional source filter is an already-normalized operator/tool input.
type MarketplaceSourceLoader func(context.Context) ([]registrypkg.Source, error)

// MutationReload is called after a registry/on-disk mutation and before the
// lifecycle helper commits any staged filesystem backup.
type MutationReload func(context.Context) error

// MarketplaceInstallRequest describes one marketplace-backed extension install.
type MarketplaceInstallRequest struct {
	Slug            string
	SourceFilter    string
	Version         string
	Asset           string
	AllowUnverified bool
	InstalledBy     string
}

// ManagedRemoveResult describes one removed managed extension.
type ManagedRemoveResult struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

// MarketplaceUpdateRequest describes one marketplace update batch.
type MarketplaceUpdateRequest struct {
	Names           []string
	All             bool
	CheckOnly       bool
	Version         string
	AllowUnverified bool
	InstalledBy     string
}

// MarketplaceUpdateResult describes one marketplace update outcome.
type MarketplaceUpdateResult struct {
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

type marketplaceManagedInstall struct {
	slug          string
	detail        *registrypkg.Detail
	manifest      *Manifest
	finalDir      string
	checksum      string
	remoteVersion string
}

// SearchMarketplaceExtensions searches configured extension marketplace
// sources with the extension package type filter applied.
func SearchMarketplaceExtensions(
	ctx context.Context,
	loader MarketplaceSourceLoader,
	query string,
	sourceFilter string,
	limit int,
) (_ []registrypkg.Listing, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("extension: marketplace search limit must be positive: %d", limit)
	}

	sources, err := LoadMarketplaceSources(ctx, loader, sourceFilter)
	if err != nil {
		return nil, err
	}

	registry := registrypkg.NewMultiRegistry(slog.Default(), sources...)
	defer func() {
		err = errors.Join(err, registry.Close())
	}()

	return registry.Search(ctx, query, registrypkg.SearchOpts{
		Limit: limit,
		Type:  registrypkg.PackageTypeExtension,
	})
}

// InstallMarketplaceManaged installs one extension through the configured
// marketplace registry into the managed extension root and records marketplace
// provenance in the installed-extension registry.
func InstallMarketplaceManaged(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	loader MarketplaceSourceLoader,
	req MarketplaceInstallRequest,
) (_ *ExtensionInfo, err error) {
	if registry == nil {
		return nil, errors.New("extension: registry is required")
	}
	prepared, err := prepareMarketplaceManagedInstall(ctx, homePaths, registry, loader, req)
	if err != nil {
		return nil, err
	}
	provenance := marketplaceInstallProvenance(prepared, req)
	if err := registry.Install(
		prepared.manifest,
		prepared.finalDir,
		prepared.checksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata(prepared.slug, strings.TrimSpace(prepared.detail.Source), prepared.remoteVersion),
		WithInstallProvenance(provenance),
	); err != nil {
		return nil, errors.Join(err, removeExtensionDir(prepared.finalDir))
	}

	info, err := registry.Get(prepared.manifest.Name)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func prepareMarketplaceManagedInstall(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	loader MarketplaceSourceLoader,
	req MarketplaceInstallRequest,
) (_ marketplaceManagedInstall, err error) {
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		return marketplaceManagedInstall{}, errors.New("extension: marketplace slug is required")
	}
	multi, err := newExtensionMarketplaceRegistry(ctx, loader, req.SourceFilter)
	if err != nil {
		return marketplaceManagedInstall{}, err
	}
	defer func() {
		err = errors.Join(err, multi.Close())
	}()

	detail, err := multi.Info(ctx, slug)
	if err != nil {
		return marketplaceManagedInstall{}, err
	}
	if !req.AllowUnverified {
		return marketplaceManagedInstall{}, newExtensionChecksumUnverifiedError(slug, detail.Source)
	}
	stagingDir, err := NewManagedInstallStagingDir(homePaths)
	if err != nil {
		return marketplaceManagedInstall{}, err
	}
	defer joinRemoveAll(&err, stagingDir, "extension: remove staged extension directory")

	result, err := installMarketplaceArchive(ctx, multi, slug, req, stagingDir)
	if err != nil {
		return marketplaceManagedInstall{}, err
	}
	manifest, finalDir, err := moveMarketplaceInstallIntoPlace(homePaths, registry, slug, result)
	if err != nil {
		return marketplaceManagedInstall{}, err
	}
	return marketplaceManagedInstall{
		slug:          slug,
		detail:        detail,
		manifest:      manifest,
		finalDir:      finalDir,
		checksum:      result.Checksum,
		remoteVersion: firstNonEmpty(result.Version, detail.Version, manifest.Version),
	}, nil
}

func newExtensionMarketplaceRegistry(
	ctx context.Context,
	loader MarketplaceSourceLoader,
	sourceFilter string,
) (*registrypkg.MultiRegistry, error) {
	sources, err := LoadMarketplaceSources(ctx, loader, sourceFilter)
	if err != nil {
		return nil, err
	}
	return registrypkg.NewMultiRegistry(slog.Default(), sources...), nil
}

func installMarketplaceArchive(
	ctx context.Context,
	multi *registrypkg.MultiRegistry,
	slug string,
	req MarketplaceInstallRequest,
	stagingDir string,
) (*registrypkg.InstallResult, error) {
	return registrypkg.NewInstaller(multi).Install(ctx, slug, registrypkg.DownloadOpts{
		Version: strings.TrimSpace(req.Version),
		Asset:   strings.TrimSpace(req.Asset),
	}, stagingDir)
}

func moveMarketplaceInstallIntoPlace(
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	slug string,
	result *registrypkg.InstallResult,
) (*Manifest, string, error) {
	manifest, err := LoadManifest(result.InstallPath)
	if err != nil {
		return nil, "", fmt.Errorf("extension: load installed extension manifest for %q: %w", slug, err)
	}
	if err := ensureExtensionNotInstalled(registry, manifest.Name); err != nil {
		return nil, "", err
	}
	finalDir, err := ManagedInstallPathChecked(homePaths, manifest.Name)
	if err != nil {
		return nil, "", err
	}
	if err := registrypkg.MoveInstalledDir(result.InstallPath, finalDir, false); err != nil {
		return nil, "", fmt.Errorf("extension: move %q into managed install path: %w", manifest.Name, err)
	}
	return manifest, finalDir, nil
}

func marketplaceInstallProvenance(
	prepared marketplaceManagedInstall,
	req MarketplaceInstallRequest,
) ExtensionProvenance {
	return ExtensionProvenance{
		Slug:             prepared.slug,
		InstalledFrom:    ExtensionInstalledFromMarketplace,
		SourceURL:        strings.TrimSpace(prepared.detail.Repository),
		ChecksumSHA256:   prepared.checksum,
		ChecksumVerified: false,
		RegistryTier:     registryTierForSource(SourceMarketplace, strings.TrimSpace(prepared.detail.Source)),
		Permissions:      extensionPermissions(prepared.manifest.Capabilities, prepared.manifest.Actions),
		InstalledBy:      firstNonEmpty(req.InstalledBy, extensionTrustInstalledByOperator),
		AllowUnverified:  req.AllowUnverified,
		Warnings: []diagnosticcontract.DiagnosticItem{
			extensionChecksumUnverifiedDiagnostic(prepared.slug, prepared.detail.Source, true),
		},
	}
}

// RemoveManagedExtension removes one installed extension and rolls back the
// registry and on-disk state if the caller's reload hook fails.
func RemoveManagedExtension(
	ctx context.Context,
	registry LifecycleRegistry,
	name string,
	reload MutationReload,
) (_ ManagedRemoveResult, err error) {
	if registry == nil {
		return ManagedRemoveResult{}, errors.New("extension: registry is required")
	}
	info, err := registry.Get(name)
	if err != nil {
		return ManagedRemoveResult{}, err
	}

	installDir, err := InstalledExtensionDir(*info)
	if err != nil {
		return ManagedRemoveResult{}, err
	}
	change, err := stageExtensionDirRemoval(installDir)
	if err != nil {
		return ManagedRemoveResult{}, err
	}

	if err := registry.Uninstall(info.Name); err != nil {
		rollbackErr := change.Rollback()
		return ManagedRemoveResult{}, errors.Join(err, rollbackErr)
	}
	if reload != nil {
		if err := reload(ctx); err != nil {
			restoreErr := restoreRemovedExtensionRecord(registry, *info, installDir, change)
			if restoreErr == nil {
				restoreErr = reload(ctx)
			}
			return ManagedRemoveResult{}, errors.Join(
				fmt.Errorf("extension: reload after remove %q: %w", info.Name, err),
				restoreErr,
			)
		}
	}
	if err := change.Commit(); err != nil {
		restoreErr := restoreRemovedExtensionRecord(registry, *info, installDir, change)
		return ManagedRemoveResult{}, errors.Join(
			fmt.Errorf("extension: finalize extension removal %q: %w", info.Name, err),
			restoreErr,
		)
	}

	return ManagedRemoveResult{
		Name:   info.Name,
		Path:   installDir,
		Status: "removed",
	}, nil
}

// UpdateMarketplaceManaged updates one or more marketplace-installed
// extensions and rolls back each changed extension if the reload hook rejects
// the new state.
func UpdateMarketplaceManaged(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	loader MarketplaceSourceLoader,
	req MarketplaceUpdateRequest,
	reload MutationReload,
) ([]MarketplaceUpdateResult, error) {
	targets, err := selectMarketplaceExtensionsForUpdate(registry, req.Names, req.All)
	if err != nil {
		return nil, err
	}

	items := make([]MarketplaceUpdateResult, 0, len(targets))
	for _, info := range targets {
		item, err := updateMarketplaceExtension(
			ctx,
			homePaths,
			registry,
			loader,
			info,
			req,
			reload,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// InstalledExtensionDir returns the root directory for a persisted extension
// registry row after validating the manifest path shape.
func InstalledExtensionDir(info ExtensionInfo) (string, error) {
	manifestPath := filepath.Clean(strings.TrimSpace(info.ManifestPath))
	if manifestPath == "" || manifestPath == "." {
		return "", fmt.Errorf("extension: extension %q has an invalid manifest path %q", info.Name, info.ManifestPath)
	}
	if !filepath.IsAbs(manifestPath) {
		return "", fmt.Errorf(
			"extension: extension %q has a non-absolute manifest path %q",
			info.Name,
			info.ManifestPath,
		)
	}

	switch filepath.Base(manifestPath) {
	case "extension.toml", "extension.json":
	default:
		return "", fmt.Errorf("extension: extension %q has an invalid manifest path %q", info.Name, info.ManifestPath)
	}

	installDir := filepath.Dir(manifestPath)
	if installDir == "." || installDir == string(filepath.Separator) {
		return "", fmt.Errorf("extension: extension %q has an invalid install directory %q", info.Name, installDir)
	}
	return installDir, nil
}

// LoadMarketplaceSources resolves and filters marketplace sources, closing
// rejected sources on every error path.
func LoadMarketplaceSources(
	ctx context.Context,
	loader MarketplaceSourceLoader,
	sourceFilter string,
) ([]registrypkg.Source, error) {
	if ctx == nil {
		return nil, errors.New("extension: context is required")
	}
	if loader == nil {
		return nil, errors.New("extension: marketplace source loader is required")
	}

	sources, err := loader(ctx)
	if err != nil {
		return nil, joinMarketplaceSourceError(err, closeRegistrySources(sources))
	}
	if len(sources) == 0 {
		err := errors.New("extension: no marketplace registry sources are configured")
		return nil, joinMarketplaceSourceError(err, closeRegistrySources(sources))
	}

	filtered := filterExtensionRegistrySources(sources, sourceFilter)
	if len(filtered) == 0 {
		err := fmt.Errorf("extension: marketplace registry source %q is not configured", sourceFilter)
		return nil, joinMarketplaceSourceError(err, closeRegistrySources(sources))
	}
	if err := closeUnselectedExtensionRegistrySources(sources, sourceFilter); err != nil {
		return nil, joinMarketplaceSourceError(err, closeRegistrySources(filtered))
	}
	return filtered, nil
}

func updateMarketplaceExtension(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	loader MarketplaceSourceLoader,
	info ExtensionInfo,
	req MarketplaceUpdateRequest,
	reload MutationReload,
) (_ MarketplaceUpdateResult, err error) {
	slug, registryName, err := marketplaceUpdateMetadata(info)
	if err != nil {
		return MarketplaceUpdateResult{}, err
	}
	multi, err := newExtensionMarketplaceRegistry(ctx, loader, registryName)
	if err != nil {
		return MarketplaceUpdateResult{}, err
	}
	defer func() {
		err = errors.Join(err, multi.Close())
	}()

	currentVersion := firstNonEmpty(dereferenceOptionalString(info.RemoteVersion), info.Version)
	requestedVersion := strings.TrimSpace(req.Version)
	updateInfo, err := multi.CheckUpdate(ctx, slug, currentVersion)
	if err != nil {
		return MarketplaceUpdateResult{}, err
	}
	if requestedVersion != "" {
		updateInfo.HasUpdate = requestedVersion != currentVersion
		updateInfo.LatestVersion = requestedVersion
	}

	installDir, err := InstalledExtensionDir(info)
	if err != nil {
		return MarketplaceUpdateResult{}, err
	}

	item := newMarketplaceUpdateResult(info, slug, registryName, currentVersion, updateInfo.LatestVersion, installDir)
	if !updateInfo.HasUpdate {
		item.Status = MarketplaceUpdateStatusCurrent
		return item, nil
	}
	if req.CheckOnly {
		item.Status = MarketplaceUpdateStatusAvailable
		return item, nil
	}
	if !req.AllowUnverified {
		return MarketplaceUpdateResult{}, newExtensionChecksumUnverifiedError(slug, registryName)
	}

	remoteVersion, err := applyMarketplaceExtensionUpdate(
		ctx,
		homePaths,
		registry,
		multi,
		info,
		updateInfo.LatestVersion,
		req.AllowUnverified,
		req.InstalledBy,
		reload,
	)
	if err != nil {
		return MarketplaceUpdateResult{}, err
	}
	item.LatestVersion = remoteVersion
	item.Status = MarketplaceUpdateStatusUpdated
	return item, nil
}

func marketplaceUpdateMetadata(info ExtensionInfo) (string, string, error) {
	slug := dereferenceOptionalString(info.RegistrySlug)
	if slug == "" {
		return "", "", fmt.Errorf("extension: extension %q is missing registry slug metadata", info.Name)
	}
	registryName := dereferenceOptionalString(info.RegistryName)
	if registryName == "" {
		return "", "", fmt.Errorf("extension: extension %q is missing registry source metadata", info.Name)
	}
	return slug, registryName, nil
}

func newMarketplaceUpdateResult(
	info ExtensionInfo,
	slug string,
	registryName string,
	currentVersion string,
	latestVersion string,
	installDir string,
) MarketplaceUpdateResult {
	return MarketplaceUpdateResult{
		Name:           info.Name,
		Slug:           slug,
		Registry:       registryName,
		CurrentVersion: currentVersion,
		LatestVersion:  firstNonEmpty(latestVersion, currentVersion),
		Path:           installDir,
	}
}

func applyMarketplaceExtensionUpdate(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
	registry LifecycleRegistry,
	multi *registrypkg.MultiRegistry,
	info ExtensionInfo,
	latestVersion string,
	allowUnverified bool,
	installedBy string,
	reload MutationReload,
) (_ string, err error) {
	slug := dereferenceOptionalString(info.RegistrySlug)
	registryName := dereferenceOptionalString(info.RegistryName)
	installDir, err := InstalledExtensionDir(info)
	if err != nil {
		return "", err
	}

	stagingDir, err := NewManagedInstallStagingDir(homePaths)
	if err != nil {
		return "", err
	}
	defer joinRemoveAll(&err, stagingDir, "extension: remove staged extension directory")

	result, err := registrypkg.NewInstaller(multi).Install(ctx, slug, registrypkg.DownloadOpts{
		Version: strings.TrimSpace(latestVersion),
	}, stagingDir)
	if err != nil {
		return "", err
	}

	manifest, err := loadMarketplaceUpdatedExtensionManifest(result.InstallPath, info.Name)
	if err != nil {
		return "", err
	}

	change, err := stageExtensionDirReplacement(result.InstallPath, installDir)
	if err != nil {
		return "", err
	}

	remoteVersion := firstNonEmpty(result.Version, latestVersion, manifest.Version)
	provenance := info.Provenance
	provenance.Slug = slug
	provenance.InstalledFrom = ExtensionInstalledFromMarketplace
	provenance.ChecksumSHA256 = result.Checksum
	provenance.ChecksumVerified = false
	provenance.RegistryTier = registryTierForSource(SourceMarketplace, registryName)
	provenance.Permissions = extensionPermissions(manifest.Capabilities, manifest.Actions)
	provenance.AllowUnverified = allowUnverified
	provenance.InstalledBy = firstNonEmpty(installedBy, provenance.InstalledBy, extensionTrustInstalledByOperator)
	provenance.Warnings = []diagnosticcontract.DiagnosticItem{
		extensionChecksumUnverifiedDiagnostic(slug, registryName, allowUnverified),
	}
	if err := registry.Install(
		manifest,
		installDir,
		result.Checksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata(slug, registryName, remoteVersion),
		WithInstallProvenance(provenance),
		WithInstallReplaceExisting(),
	); err != nil {
		return "", errors.Join(err, change.Rollback())
	}
	if reload != nil {
		if err := reload(ctx); err != nil {
			restoreErr := restoreUpdatedExtensionRecord(registry, info, installDir, change)
			if restoreErr == nil {
				restoreErr = reload(ctx)
			}
			return "", errors.Join(
				fmt.Errorf("extension: reload after update %q: %w", info.Name, err),
				restoreErr,
			)
		}
	}
	if err := change.Commit(); err != nil {
		return "", err
	}
	return remoteVersion, nil
}

func loadMarketplaceUpdatedExtensionManifest(installPath string, installedName string) (*Manifest, error) {
	manifest, err := LoadManifest(installPath)
	if err != nil {
		return nil, fmt.Errorf("extension: load updated extension manifest for %q: %w", installedName, err)
	}
	if manifest.Name != installedName {
		return nil, fmt.Errorf(
			"extension: extension update identity mismatch: installed %q, registry returned %q",
			installedName,
			manifest.Name,
		)
	}
	return manifest, nil
}

func ensureExtensionNotInstalled(registry LifecycleRegistry, name string) error {
	if _, err := registry.Get(name); err == nil {
		return &ExtensionExistsError{Name: name}
	} else if !errors.Is(err, ErrExtensionNotFound) {
		return err
	}
	return nil
}

func selectMarketplaceExtensionsForUpdate(
	registry LifecycleRegistry,
	names []string,
	updateAll bool,
) ([]ExtensionInfo, error) {
	if registry == nil {
		return nil, errors.New("extension: registry is required")
	}
	if updateAll {
		infos, err := registry.List()
		if err != nil {
			return nil, err
		}
		items := make([]ExtensionInfo, 0, len(infos))
		for _, info := range infos {
			if marketplaceExtensionInstalled(info) {
				items = append(items, info)
			}
		}
		return items, nil
	}

	name := ""
	if len(names) > 0 {
		name = strings.TrimSpace(names[0])
	}
	if name == "" {
		return nil, errors.New("extension: extension name is required unless all is set")
	}

	info, err := registry.Get(name)
	if err != nil {
		return nil, err
	}
	if !marketplaceExtensionInstalled(*info) {
		return nil, fmt.Errorf("extension: extension %q is not a marketplace-installed extension", info.Name)
	}
	return []ExtensionInfo{*info}, nil
}

func marketplaceExtensionInstalled(info ExtensionInfo) bool {
	return info.Source == SourceMarketplace && dereferenceOptionalString(info.RegistrySlug) != ""
}

func restoreRemovedExtensionRecord(
	registry LifecycleRegistry,
	info ExtensionInfo,
	installDir string,
	change *stagedExtensionDirChange,
) error {
	rollbackErr := change.Rollback()
	if rollbackErr != nil {
		return rollbackErr
	}
	return reinstallExtensionInfo(registry, info, installDir)
}

func restoreUpdatedExtensionRecord(
	registry LifecycleRegistry,
	info ExtensionInfo,
	installDir string,
	change *stagedExtensionDirChange,
) error {
	rollbackErr := change.Rollback()
	if rollbackErr != nil {
		return rollbackErr
	}
	return reinstallExtensionInfo(registry, info, installDir, WithInstallReplaceExisting())
}

func reinstallExtensionInfo(
	registry LifecycleRegistry,
	info ExtensionInfo,
	installDir string,
	opts ...InstallOption,
) error {
	manifest, err := LoadManifest(installDir)
	if err != nil {
		return fmt.Errorf("extension: reload manifest from %q after rollback: %w", installDir, err)
	}

	installOpts := []InstallOption{
		WithInstallSource(info.Source),
		WithInstallProvenance(info.Provenance),
	}
	if info.Source == SourceMarketplace {
		installOpts = append(installOpts, WithInstallRegistryMetadata(
			dereferenceOptionalString(info.RegistrySlug),
			dereferenceOptionalString(info.RegistryName),
			dereferenceOptionalString(info.RemoteVersion),
		))
	}
	installOpts = append(installOpts, opts...)

	if err := registry.Install(manifest, installDir, info.Checksum, installOpts...); err != nil {
		return fmt.Errorf("extension: restore registry record for %q: %w", info.Name, err)
	}
	if !info.Enabled {
		if err := registry.Disable(info.Name); err != nil {
			return fmt.Errorf("extension: restore disabled state for %q: %w", info.Name, err)
		}
	}
	return nil
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
		return nil, fmt.Errorf("extension: stat extension directory %q: %w", targetDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("extension: extension path %q is not a directory", targetDir)
	}

	backupDir := uniqueExtensionBackupPath(targetDir)
	if err := os.Rename(targetDir, backupDir); err != nil {
		return nil, fmt.Errorf("extension: stage extension directory removal %q: %w", targetDir, err)
	}
	change.backupDir = backupDir
	return change, nil
}

func stageExtensionDirReplacement(stagingDir string, targetDir string) (*stagedExtensionDirChange, error) {
	change := &stagedExtensionDirChange{
		targetDir: targetDir,
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return nil, fmt.Errorf("extension: create install parent %q: %w", filepath.Dir(targetDir), err)
	}

	if _, err := os.Stat(targetDir); err == nil {
		backupDir := uniqueExtensionBackupPath(targetDir)
		if err := os.Rename(targetDir, backupDir); err != nil {
			return nil, fmt.Errorf("extension: backup extension directory %q: %w", targetDir, err)
		}
		change.backupDir = backupDir
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("extension: stat extension directory %q: %w", targetDir, err)
	}

	if err := os.Rename(stagingDir, targetDir); err != nil {
		rollbackErr := change.Rollback()
		return nil, errors.Join(
			fmt.Errorf("extension: move staged extension into %q: %w", targetDir, err),
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
		return fmt.Errorf("extension: remove extension backup %q: %w", c.backupDir, err)
	}
	return nil
}

func (c *stagedExtensionDirChange) Rollback() error {
	if c == nil {
		return nil
	}

	if strings.TrimSpace(c.targetDir) != "" {
		if err := os.RemoveAll(c.targetDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("extension: remove failed extension target %q: %w", c.targetDir, err)
		}
	}
	if strings.TrimSpace(c.backupDir) == "" {
		return nil
	}
	if err := os.Rename(c.backupDir, c.targetDir); err != nil {
		return fmt.Errorf("extension: restore extension backup %q: %w", c.targetDir, err)
	}
	return nil
}

func filterExtensionRegistrySources(sources []registrypkg.Source, sourceFilter string) []registrypkg.Source {
	filter := strings.ToLower(strings.TrimSpace(sourceFilter))
	if filter == "" {
		return sources
	}

	filtered := make([]registrypkg.Source, 0, len(sources))
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

func closeUnselectedExtensionRegistrySources(sources []registrypkg.Source, sourceFilter string) error {
	filter := strings.TrimSpace(sourceFilter)
	if filter == "" {
		return nil
	}

	dropped := make([]registrypkg.Source, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(source.Name()), filter) {
			continue
		}
		dropped = append(dropped, source)
	}
	return closeRegistrySources(dropped)
}

func closeRegistrySources(sources []registrypkg.Source) error {
	errs := make([]error, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		if err := source.Close(); err != nil {
			errs = append(errs, fmt.Errorf("extension: close registry source %q: %w", source.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func joinMarketplaceSourceError(base error, extra error) error {
	if extra == nil {
		return base
	}
	if base == nil {
		return extra
	}
	return errors.Join(base, extra)
}

func joinRemoveAll(errp *error, path string, label string) {
	removeErr := os.RemoveAll(path)
	if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		return
	}
	*errp = errors.Join(*errp, fmt.Errorf("%s %q: %w", label, path, removeErr))
}

func removeExtensionDir(path string) error {
	if err := os.RemoveAll(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("extension: remove failed extension install %q: %w", path, err)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
