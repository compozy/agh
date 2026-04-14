package extension

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

const registryTestExtensionsTableSchema = `CREATE TABLE IF NOT EXISTS extensions (
	name          TEXT PRIMARY KEY,
	version       TEXT NOT NULL,
	source        TEXT NOT NULL,
	enabled       BOOLEAN NOT NULL DEFAULT 1,
	manifest_path TEXT NOT NULL,
	installed_at  TEXT NOT NULL,
	capabilities  TEXT NOT NULL DEFAULT '{}',
	actions       TEXT NOT NULL DEFAULT '{}',
	checksum      TEXT NOT NULL,
	registry_slug TEXT,
	registry_name TEXT,
	remote_version TEXT
);`

const legacyRegistryTestExtensionsTableSchema = `CREATE TABLE IF NOT EXISTS extensions (
	name          TEXT PRIMARY KEY,
	version       TEXT NOT NULL,
	source        TEXT NOT NULL,
	enabled       BOOLEAN NOT NULL DEFAULT 1,
	manifest_path TEXT NOT NULL,
	installed_at  TEXT NOT NULL,
	capabilities  TEXT NOT NULL DEFAULT '{}',
	actions       TEXT NOT NULL DEFAULT '{}',
	checksum      TEXT NOT NULL
);`

func TestRegistryInstallPersistsExtension(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "alpha-registry", registryManifestOptions{
		capabilities: []string{"memory.backend", "prompt.provider"},
		actions:      []string{"sessions/list", "observe/health"},
		extraFiles: map[string]string{
			"skills/alpha.md": "# alpha\n",
		},
	})

	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name != manifest.Name {
		t.Fatalf("Name = %q, want %q", got.Name, manifest.Name)
	}
	if got.Version != manifest.Version {
		t.Fatalf("Version = %q, want %q", got.Version, manifest.Version)
	}
	if got.Source != SourceUser {
		t.Fatalf("Source = %v, want %v", got.Source, SourceUser)
	}
	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if got.ManifestPath != filepath.Join(dir, manifestTOMLFileName) {
		t.Fatalf("ManifestPath = %q, want %q", got.ManifestPath, filepath.Join(dir, manifestTOMLFileName))
	}
	if !got.InstalledAt.Equal(env.installedAt) {
		t.Fatalf("InstalledAt = %v, want %v", got.InstalledAt, env.installedAt)
	}
	if !reflect.DeepEqual(got.Capabilities, normalizeCapabilitiesConfig(manifest.Capabilities)) {
		t.Fatalf("Capabilities = %#v, want %#v", got.Capabilities, normalizeCapabilitiesConfig(manifest.Capabilities))
	}
	if !reflect.DeepEqual(got.Actions, normalizeActionsConfig(manifest.Actions)) {
		t.Fatalf("Actions = %#v, want %#v", got.Actions, normalizeActionsConfig(manifest.Actions))
	}
	if got.Checksum != checksum {
		t.Fatalf("Checksum = %q, want %q", got.Checksum, checksum)
	}
	if got.RegistrySlug != nil || got.RegistryName != nil || got.RemoteVersion != nil {
		t.Fatalf(
			"local install registry metadata = (%v, %v, %v), want all nil",
			got.RegistrySlug,
			got.RegistryName,
			got.RemoteVersion,
		)
	}
}

func TestRegistryInstallRejectsDuplicateName(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "duplicate-registry", registryManifestOptions{})

	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install(first) error = %v", err)
	}

	err := env.registry.Install(manifest, dir, checksum)
	if err == nil {
		t.Fatal("Install(duplicate) error = nil, want ErrExtensionExists")
	}
	if !errors.Is(err, ErrExtensionExists) {
		t.Fatalf("Install(duplicate) error = %v, want ErrExtensionExists", err)
	}

	var existsErr *ExtensionExistsError
	if !errors.As(err, &existsErr) {
		t.Fatalf("Install(duplicate) error type = %T, want *ExtensionExistsError", err)
	}
	if existsErr.Name != manifest.Name {
		t.Fatalf("ExtensionExistsError.Name = %q, want %q", existsErr.Name, manifest.Name)
	}
}

func TestRegistryInstallPersistsMarketplaceMetadata(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "marketplace-registry", registryManifestOptions{})

	if err := env.registry.Install(
		manifest,
		dir,
		checksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata("acme/marketplace-registry", "github", "v1.4.0"),
	); err != nil {
		t.Fatalf("Install(marketplace) error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Source != SourceMarketplace {
		t.Fatalf("Source = %v, want %v", got.Source, SourceMarketplace)
	}
	if got.RegistrySlug == nil || *got.RegistrySlug != "acme/marketplace-registry" {
		t.Fatalf("RegistrySlug = %#v, want acme/marketplace-registry", got.RegistrySlug)
	}
	if got.RegistryName == nil || *got.RegistryName != "github" {
		t.Fatalf("RegistryName = %#v, want github", got.RegistryName)
	}
	if got.RemoteVersion == nil || *got.RemoteVersion != "v1.4.0" {
		t.Fatalf("RemoteVersion = %#v, want v1.4.0", got.RemoteVersion)
	}
}

func TestRegistryInstallRejectsChecksumMismatch(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, _ := createRegistryTestExtension(t, "checksum-registry", registryManifestOptions{
		extraFiles: map[string]string{
			"bin/extension": "#!/bin/sh\n",
		},
	})

	err := env.registry.Install(manifest, dir, strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("Install(checksum mismatch) error = nil, want ErrExtensionChecksumMismatch")
	}
	if !errors.Is(err, ErrExtensionChecksumMismatch) {
		t.Fatalf("Install(checksum mismatch) error = %v, want ErrExtensionChecksumMismatch", err)
	}

	var mismatchErr *ExtensionChecksumMismatchError
	if !errors.As(err, &mismatchErr) {
		t.Fatalf("Install(checksum mismatch) error type = %T, want *ExtensionChecksumMismatchError", err)
	}
	if mismatchErr.ExpectedChecksum != strings.Repeat("0", 64) {
		t.Fatalf("ExpectedChecksum = %q, want %q", mismatchErr.ExpectedChecksum, strings.Repeat("0", 64))
	}

	got, err := env.registry.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(List()) = %d, want 0", len(got))
	}
}

func TestRegistryGetReturnsNotFound(t *testing.T) {
	env := newRegistryTestEnv(t)

	_, err := env.registry.Get("missing")
	if err == nil {
		t.Fatal("Get(missing) error = nil, want ErrExtensionNotFound")
	}
	if !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrExtensionNotFound", err)
	}

	var notFoundErr *ExtensionNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Get(missing) error type = %T, want *ExtensionNotFoundError", err)
	}
	if notFoundErr.Name != "missing" {
		t.Fatalf("ExtensionNotFoundError.Name = %q, want %q", notFoundErr.Name, "missing")
	}
}

func TestRegistryListReturnsAllInstalledExtensions(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	alphaDir, alphaManifest, alphaChecksum := createRegistryTestExtension(t, "alpha", registryManifestOptions{
		capabilities: []string{"memory.backend"},
		actions:      []string{"sessions/list"},
	})
	betaDir, betaManifest, betaChecksum := createRegistryTestExtension(t, "beta", registryManifestOptions{
		capabilities: []string{"prompt.provider"},
		actions:      []string{"observe/health"},
	})

	if err := env.registry.Install(alphaManifest, alphaDir, alphaChecksum); err != nil {
		t.Fatalf("Install(alpha) error = %v", err)
	}
	if err := env.registry.installWithSource(betaManifest, betaDir, betaChecksum, SourceMarketplace); err != nil {
		t.Fatalf("installWithSource(beta) error = %v", err)
	}

	got, err := env.registry.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(got))
	}

	names := []string{got[0].Name, got[1].Name}
	if !slices.Equal(names, []string{"alpha", "beta"}) {
		t.Fatalf("names = %v, want %v", names, []string{"alpha", "beta"})
	}
	if got[0].Source != SourceUser {
		t.Fatalf("alpha source = %v, want %v", got[0].Source, SourceUser)
	}
	if got[1].Source != SourceMarketplace {
		t.Fatalf("beta source = %v, want %v", got[1].Source, SourceMarketplace)
	}
}

func TestRegistryListReturnsEmptySlice(t *testing.T) {
	env := newRegistryTestEnv(t)

	got, err := env.registry.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got == nil {
		t.Fatal("List() = nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len(List()) = %d, want 0", len(got))
	}
}

func TestRegistryEnableAndDisable(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "toggle-registry", registryManifestOptions{})

	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := env.registry.Disable(manifest.Name); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	disabled, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get(disabled) error = %v", err)
	}
	if disabled.Enabled {
		t.Fatal("Enabled after Disable() = true, want false")
	}

	if err := env.registry.Enable(manifest.Name); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	enabled, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get(enabled) error = %v", err)
	}
	if !enabled.Enabled {
		t.Fatal("Enabled after Enable() = false, want true")
	}
}

func TestRegistryUninstallRemovesExtension(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "remove-registry", registryManifestOptions{})

	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := env.registry.Uninstall(manifest.Name); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	_, err := env.registry.Get(manifest.Name)
	if !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("Get(after uninstall) error = %v, want ErrExtensionNotFound", err)
	}
}

func TestRegistryUninstallMissingReturnsNotFound(t *testing.T) {
	env := newRegistryTestEnv(t)

	err := env.registry.Uninstall("missing")
	if err == nil {
		t.Fatal("Uninstall(missing) error = nil, want ErrExtensionNotFound")
	}
	if !errors.Is(err, ErrExtensionNotFound) {
		t.Fatalf("Uninstall(missing) error = %v, want ErrExtensionNotFound", err)
	}
}

func TestRegistryCapabilitiesAndActionsJSONRoundTrip(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "round-trip-registry", registryManifestOptions{
		capabilities: []string{"agent.driver", "memory.backend", "prompt.provider"},
		actions:      []string{"memory/recall", "observe/health", "sessions/list"},
	})

	if err := env.registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !reflect.DeepEqual(got.Capabilities, normalizeCapabilitiesConfig(manifest.Capabilities)) {
		t.Fatalf("Capabilities = %#v, want %#v", got.Capabilities, normalizeCapabilitiesConfig(manifest.Capabilities))
	}
	if !reflect.DeepEqual(got.Actions, normalizeActionsConfig(manifest.Actions)) {
		t.Fatalf("Actions = %#v, want %#v", got.Actions, normalizeActionsConfig(manifest.Actions))
	}
}

func TestRegistryInstallConcurrentDuplicateReturnsSingleExistsError(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "concurrent-registry", registryManifestOptions{})

	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			errs <- env.registry.Install(manifest, dir, checksum)
		}()
	}
	close(start)

	var (
		successes   int
		existsCount int
	)
	for range 2 {
		err := <-errs
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrExtensionExists):
			existsCount++
		default:
			t.Fatalf("concurrent Install() error = %v, want nil or ErrExtensionExists", err)
		}
	}

	if successes != 1 || existsCount != 1 {
		t.Fatalf("concurrent Install() outcomes = successes:%d exists:%d, want 1 and 1", successes, existsCount)
	}
}

func TestRegistryInstallReplaceExistingUpdatesMarketplaceRecord(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "replace-registry", registryManifestOptions{})

	if err := env.registry.Install(
		manifest,
		dir,
		checksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata("acme/replace-registry", "github", "1.0.0"),
	); err != nil {
		t.Fatalf("Install(first) error = %v", err)
	}

	writeFile(t, filepath.Join(dir, manifestTOMLFileName), strings.Replace(registryManifestTOML("replace-registry", registryManifestOptions{}), `version = "0.2.1"`, `version = "0.3.0"`, 1))
	updatedManifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(updated) error = %v", err)
	}
	updatedChecksum, err := ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(updated) error = %v", err)
	}

	if err := env.registry.Install(
		updatedManifest,
		dir,
		updatedChecksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata("acme/replace-registry", "github", "1.1.0"),
		WithInstallReplaceExisting(),
	); err != nil {
		t.Fatalf("Install(replace) error = %v", err)
	}

	got, err := env.registry.Get("replace-registry")
	if err != nil {
		t.Fatalf("Get(updated) error = %v", err)
	}
	if got.Version != "0.3.0" {
		t.Fatalf("Version = %q, want %q", got.Version, "0.3.0")
	}
	if got.RemoteVersion == nil || *got.RemoteVersion != "1.1.0" {
		t.Fatalf("RemoteVersion = %#v, want 1.1.0", got.RemoteVersion)
	}
}

func TestRegistryInstallReplaceExistingPreservesEnabledState(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "disabled-replace-registry", registryManifestOptions{})

	if err := env.registry.Install(
		manifest,
		dir,
		checksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata("acme/disabled-replace-registry", "github", "1.0.0"),
	); err != nil {
		t.Fatalf("Install(first) error = %v", err)
	}
	if err := env.registry.Disable(manifest.Name); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	writeFile(t, filepath.Join(dir, manifestTOMLFileName), strings.Replace(registryManifestTOML("disabled-replace-registry", registryManifestOptions{}), `version = "0.2.1"`, `version = "0.3.0"`, 1))
	updatedManifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(updated) error = %v", err)
	}
	updatedChecksum, err := ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(updated) error = %v", err)
	}

	if err := env.registry.Install(
		updatedManifest,
		dir,
		updatedChecksum,
		WithInstallSource(SourceMarketplace),
		WithInstallRegistryMetadata("acme/disabled-replace-registry", "github", "1.1.0"),
		WithInstallReplaceExisting(),
	); err != nil {
		t.Fatalf("Install(replace) error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get(updated) error = %v", err)
	}
	if got.Enabled {
		t.Fatal("Enabled = true after replace, want disabled state preserved")
	}
	if got.Version != "0.3.0" {
		t.Fatalf("Version = %q, want %q", got.Version, "0.3.0")
	}
}

func TestRegistryInstallReplaceExistingWrapsPersistErrors(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	dbPath := filepath.Join(t.TempDir(), "agh-registry-legacy.db")
	db, err := store.OpenSQLiteDatabase(testutil.Context(t), dbPath, func(ctx context.Context, db *sql.DB) error {
		return store.EnsureSchema(ctx, db, []string{legacyRegistryTestExtensionsTableSchema})
	})
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("db.Close() error = %v", closeErr)
		}
	})

	registry := NewRegistry(db)
	dir, manifest, checksum := createRegistryTestExtension(t, "legacy-persist-registry", registryManifestOptions{})

	err = registry.Install(manifest, dir, checksum, WithInstallReplaceExisting())
	if err == nil {
		t.Fatal("Install(replace existing on legacy schema) error = nil, want failure")
	}
	if !strings.Contains(err.Error(), `extension: persist "legacy-persist-registry"`) {
		t.Fatalf("Install(replace existing on legacy schema) error = %v, want wrapped persist context", err)
	}
}

func TestRegistryInstallClearsRemoteMetadataForNonMarketplaceSources(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "non-marketplace-registry", registryManifestOptions{})

	if err := env.registry.Install(
		manifest,
		dir,
		checksum,
		WithInstallRegistryMetadata("acme/non-marketplace", "github", "1.0.0"),
	); err != nil {
		t.Fatalf("Install(non-marketplace metadata) error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.RegistrySlug != nil || got.RegistryName != nil || got.RemoteVersion != nil {
		t.Fatalf(
			"non-marketplace registry metadata = (%v, %v, %v), want all nil",
			got.RegistrySlug,
			got.RegistryName,
			got.RemoteVersion,
		)
	}
}

func TestRegistryInstallAcceptsManifestFilePathAndExplicitSource(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "file-path-install", registryManifestOptions{})
	manifestPath := filepath.Join(dir, manifestTOMLFileName)

	if err := env.registry.installWithSource(manifest, manifestPath, checksum, SourceBundled); err != nil {
		t.Fatalf("installWithSource(file manifest) error = %v", err)
	}

	got, err := env.registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Source != SourceBundled {
		t.Fatalf("Source = %v, want %v", got.Source, SourceBundled)
	}
	if got.ManifestPath != manifestPath {
		t.Fatalf("ManifestPath = %q, want %q", got.ManifestPath, manifestPath)
	}
}

func TestRegistryInstallRejectsInvalidSourceAndBlankChecksum(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "invalid-source", registryManifestOptions{})

	if err := env.registry.installWithSource(manifest, dir, checksum, ExtensionSource(99)); err == nil {
		t.Fatal("installWithSource(invalid source) error = nil, want non-nil")
	}
	if err := env.registry.Install(manifest, dir, ""); err == nil {
		t.Fatal("Install(blank checksum) error = nil, want non-nil")
	}
}

func TestRegistryInstallRejectsOnDiskManifestIdentityMismatch(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	env := newRegistryTestEnv(t)
	dir, manifest, checksum := createRegistryTestExtension(t, "identity-registry", registryManifestOptions{})

	spoofed := *manifest
	spoofed.Name = "spoofed-registry"

	err := env.registry.installWithSource(&spoofed, dir, checksum, SourceUser)
	if err == nil {
		t.Fatal("installWithSource(mismatched manifest) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "does not match provided identity") {
		t.Fatalf("installWithSource(mismatched manifest) error = %v, want identity mismatch detail", err)
	}
}

func TestRegistryUtilityHelpers(t *testing.T) {
	withDaemonVersion(t, "0.6.0")

	t.Run("capability denied error formatting", func(t *testing.T) {
		var nilDenied *ErrCapabilityDenied
		if got := nilDenied.Error(); got != "" {
			t.Fatalf("nil ErrCapabilityDenied.Error() = %q, want empty", got)
		}

		denied := &ErrCapabilityDenied{}
		if got := denied.Error(); got != "capability_denied" {
			t.Fatalf("ErrCapabilityDenied{}.Error() = %q, want %q", got, "capability_denied")
		}

		denied = &ErrCapabilityDenied{Data: CapabilityDeniedData{Method: "sessions/list"}}
		if got := denied.Error(); got != "capability_denied: sessions/list" {
			t.Fatalf("ErrCapabilityDenied(method).Error() = %q", got)
		}

		denied = &ErrCapabilityDenied{
			Data: CapabilityDeniedData{
				Method:   "sessions/list",
				Required: []string{"session.read"},
				Granted:  []string{"observe.read"},
			},
		}
		if got := denied.Error(); !strings.Contains(got, "requires") {
			t.Fatalf("ErrCapabilityDenied(required).Error() = %q, want requires text", got)
		}
	})

	t.Run("typed registry errors expose sentinels", func(t *testing.T) {
		notFound := &ExtensionNotFoundError{Name: "missing"}
		if got := notFound.Error(); got != "extension: extension not found: missing" {
			t.Fatalf("ExtensionNotFoundError.Error() = %q", got)
		}
		if !errors.Is(notFound, ErrExtensionNotFound) {
			t.Fatalf("errors.Is(notFound, ErrExtensionNotFound) = false")
		}

		existsErr := &ExtensionExistsError{Name: "duplicate"}
		if got := existsErr.Error(); got != "extension: extension already exists: duplicate" {
			t.Fatalf("ExtensionExistsError.Error() = %q", got)
		}
		if !errors.Is(existsErr, ErrExtensionExists) {
			t.Fatalf("errors.Is(existsErr, ErrExtensionExists) = false")
		}

		var nilChecksum *ExtensionChecksumMismatchError
		if got := nilChecksum.Error(); got != "extension: checksum mismatch" {
			t.Fatalf("nil ExtensionChecksumMismatchError.Error() = %q", got)
		}

		mismatch := &ExtensionChecksumMismatchError{ExpectedChecksum: "abc", ActualChecksum: "def"}
		if got := mismatch.Error(); !strings.Contains(got, "expected abc, got def") {
			t.Fatalf("ExtensionChecksumMismatchError.Error() = %q", got)
		}
		if !errors.Is(mismatch, ErrExtensionChecksumMismatch) {
			t.Fatalf("errors.Is(mismatch, ErrExtensionChecksumMismatch) = false")
		}
	})

	t.Run("parse extension source", func(t *testing.T) {
		for input, want := range map[string]ExtensionSource{
			"bundled":     SourceBundled,
			"user":        SourceUser,
			"workspace":   SourceWorkspace,
			"marketplace": SourceMarketplace,
		} {
			got, err := parseExtensionSource(input)
			if err != nil {
				t.Fatalf("parseExtensionSource(%q) error = %v", input, err)
			}
			if got != want {
				t.Fatalf("parseExtensionSource(%q) = %v, want %v", input, got, want)
			}
		}
		if _, err := parseExtensionSource("unknown"); err == nil {
			t.Fatal("parseExtensionSource(unknown) error = nil, want non-nil")
		}
	})

	t.Run("check ready validates receiver", func(t *testing.T) {
		var nilRegistry *Registry
		if err := nilRegistry.checkReady("list"); err == nil {
			t.Fatal("nil registry checkReady() error = nil, want non-nil")
		}

		registry := &Registry{}
		if err := registry.checkReady("list"); err == nil {
			t.Fatal("nil db checkReady() error = nil, want non-nil")
		}
	})

	t.Run("resolve install artifact supports directory and manifest file", func(t *testing.T) {
		dir := t.TempDir()
		jsonPath := filepath.Join(dir, manifestJSONFileName)
		writeFile(t, jsonPath, `{
  "extension": {
    "name": "json-only",
    "version": "0.2.1",
    "min_agh_version": "0.5.0"
  }
}`)

		artifactRoot, manifestPath, err := resolveInstallArtifact(dir)
		if err != nil {
			t.Fatalf("resolveInstallArtifact(dir) error = %v", err)
		}
		if artifactRoot != dir {
			t.Fatalf("artifactRoot = %q, want %q", artifactRoot, dir)
		}
		if manifestPath != jsonPath {
			t.Fatalf("manifestPath = %q, want %q", manifestPath, jsonPath)
		}

		fileRoot, fileManifest, err := resolveInstallArtifact(jsonPath)
		if err != nil {
			t.Fatalf("resolveInstallArtifact(file) error = %v", err)
		}
		if fileRoot != dir || fileManifest != jsonPath {
			t.Fatalf("resolveInstallArtifact(file) = (%q, %q)", fileRoot, fileManifest)
		}

		otherFile := filepath.Join(dir, "notes.txt")
		writeFile(t, otherFile, "notes")
		if _, _, err := resolveInstallArtifact(otherFile); err == nil {
			t.Fatal("resolveInstallArtifact(notes.txt) error = nil, want non-nil")
		}
		if _, _, err := resolveInstallArtifact(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Fatal("resolveInstallArtifact(missing) error = nil, want non-nil")
		}
		if _, _, err := resolveInstallArtifact(""); err == nil {
			t.Fatal("resolveInstallArtifact(blank) error = nil, want non-nil")
		}
	})

	t.Run("checksum helper handles errors and hashes symlinks deterministically", func(t *testing.T) {
		if _, err := ComputeDirectoryChecksum(""); err == nil {
			t.Fatal("ComputeDirectoryChecksum(blank) error = nil, want non-nil")
		}

		filePath := filepath.Join(t.TempDir(), "not-a-dir.txt")
		writeFile(t, filePath, "payload")
		if _, err := ComputeDirectoryChecksum(filePath); err == nil {
			t.Fatal("ComputeDirectoryChecksum(file) error = nil, want non-nil")
		}

		dir := t.TempDir()
		targetPath := filepath.Join(dir, "payload.txt")
		writeFile(t, targetPath, "payload")
		linkPath := filepath.Join(dir, "payload-link.txt")
		if err := os.Symlink("payload.txt", linkPath); err != nil {
			t.Skipf("os.Symlink() unavailable: %v", err)
		}
		packageDir := filepath.Join(dir, "packages", "typescript", "bin")
		if err := os.MkdirAll(packageDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", packageDir, err)
		}
		writeFile(t, filepath.Join(packageDir, "tsc"), "#!/usr/bin/env node\n")
		nodeModulesDir := filepath.Join(dir, "node_modules", ".bin")
		if err := os.MkdirAll(nodeModulesDir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", nodeModulesDir, err)
		}
		if err := os.Symlink(filepath.Join("..", "..", "packages", "typescript"), filepath.Join(dir, "node_modules", "typescript")); err != nil {
			t.Skipf("os.Symlink(directory) unavailable: %v", err)
		}
		if err := os.Symlink(filepath.Join("..", "typescript", "bin", "tsc"), filepath.Join(nodeModulesDir, "tsc")); err != nil {
			t.Skipf("os.Symlink(file) unavailable: %v", err)
		}

		first, err := ComputeDirectoryChecksum(dir)
		if err != nil {
			t.Fatalf("ComputeDirectoryChecksum(dir with symlink) error = %v", err)
		}
		second, err := ComputeDirectoryChecksum(dir)
		if err != nil {
			t.Fatalf("ComputeDirectoryChecksum(dir with repeated symlink hash) error = %v", err)
		}
		if first != second {
			t.Fatalf("ComputeDirectoryChecksum(dir with symlink) = %q on first pass, %q on second pass, want stable checksum", first, second)
		}

		if err := os.Remove(filepath.Join(nodeModulesDir, "tsc")); err != nil {
			t.Fatalf("os.Remove(tsc symlink) error = %v", err)
		}
		if err := os.Symlink(filepath.Join("..", "..", "payload.txt"), filepath.Join(nodeModulesDir, "tsc")); err != nil {
			t.Fatalf("os.Symlink(updated file target) error = %v", err)
		}

		updated, err := ComputeDirectoryChecksum(dir)
		if err != nil {
			t.Fatalf("ComputeDirectoryChecksum(dir after symlink target change) error = %v", err)
		}
		if updated == first {
			t.Fatalf("ComputeDirectoryChecksum(dir after symlink target change) = %q, want checksum change from %q", updated, first)
		}
	})

	t.Run("constraint mapper preserves passthrough errors", func(t *testing.T) {
		original := errors.New("boom")
		if err := mapRegistryConstraintError(original, "duplicate"); !errors.Is(err, original) {
			t.Fatalf("mapRegistryConstraintError(boom) = %v, want wrapped boom", err)
		}
	})
}

type registryTestEnv struct {
	db          *sql.DB
	registry    *Registry
	installedAt time.Time
}

type registryManifestOptions struct {
	capabilities []string
	actions      []string
	extraFiles   map[string]string
}

func newRegistryTestEnv(t *testing.T) registryTestEnv {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "agh-registry.db")
	db, err := store.OpenSQLiteDatabase(testutil.Context(t), dbPath, func(ctx context.Context, db *sql.DB) error {
		return store.EnsureSchema(ctx, db, []string{registryTestExtensionsTableSchema})
	})
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})

	installedAt := time.Date(2026, 4, 10, 15, 30, 0, 0, time.UTC)
	registry := NewRegistry(db)
	registry.now = func() time.Time { return installedAt }

	return registryTestEnv{
		db:          db,
		registry:    registry,
		installedAt: installedAt,
	}
}

func createRegistryTestExtension(t *testing.T, name string, opts registryManifestOptions) (string, *Manifest, string) {
	t.Helper()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, manifestTOMLFileName), registryManifestTOML(name, opts))

	for relPath, content := range opts.extraFiles {
		absPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(absPath), err)
		}
		writeFile(t, absPath, content)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error = %v", dir, err)
	}
	checksum, err := ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%q) error = %v", dir, err)
	}

	return dir, manifest, checksum
}

func registryManifestTOML(name string, opts registryManifestOptions) string {
	capabilities := append([]string(nil), opts.capabilities...)
	if len(capabilities) == 0 {
		capabilities = []string{"memory.backend"}
	}

	actions := append([]string(nil), opts.actions...)
	if len(actions) == 0 {
		actions = []string{"sessions/list"}
	}

	return fmt.Sprintf(
		`[extension]
name = %q
version = "0.2.1"
description = "Registry test extension"
min_agh_version = "0.5.0"

[capabilities]
provides = %s

[actions]
requires = %s
`,
		name,
		tomlStringArray(capabilities),
		tomlStringArray(actions),
	)
}

func tomlStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
