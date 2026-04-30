package extensionpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

type lifecycleSource struct {
	name          string
	latestVersion string
	archives      map[string][]byte
	closeErr      error
	closeCount    int
}

var _ registrypkg.Source = (*lifecycleSource)(nil)

func (s *lifecycleSource) Name() string {
	if strings.TrimSpace(s.name) == "" {
		return "github"
	}
	return s.name
}

func (s *lifecycleSource) Capabilities() registrypkg.SourceCaps {
	return registrypkg.SourceCaps{Search: true}
}

func (s *lifecycleSource) Search(
	context.Context,
	string,
	registrypkg.SearchOpts,
) ([]registrypkg.Listing, error) {
	return []registrypkg.Listing{
		{
			Slug:    "acme/lifecycle-ext",
			Name:    "lifecycle-ext",
			Version: s.latestVersion,
			Source:  s.Name(),
			Type:    registrypkg.PackageTypeExtension,
		},
	}, nil
}

func (s *lifecycleSource) Info(context.Context, string) (*registrypkg.Detail, error) {
	return &registrypkg.Detail{
		Listing: registrypkg.Listing{
			Slug:    "acme/lifecycle-ext",
			Name:    "lifecycle-ext",
			Version: s.latestVersion,
			Source:  s.Name(),
			Type:    registrypkg.PackageTypeExtension,
		},
	}, nil
}

func (s *lifecycleSource) Download(
	_ context.Context,
	_ string,
	opts registrypkg.DownloadOpts,
) (*registrypkg.DownloadResult, error) {
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = s.latestVersion
	}
	archive := s.archives[version]
	if archive == nil {
		return nil, fmt.Errorf("test source missing version %q", version)
	}
	return &registrypkg.DownloadResult{
		Reader:      io.NopCloser(bytes.NewReader(archive)),
		Slug:        "acme/lifecycle-ext",
		Version:     version,
		ContentSize: -1,
		ContentType: "application/gzip",
	}, nil
}

func (s *lifecycleSource) Close() error {
	s.closeCount++
	return s.closeErr
}

func TestMarketplaceLifecycleInstallsUpdatesAndRemovesManagedExtensions(t *testing.T) {
	t.Run("Should install update and remove managed marketplace extensions", func(t *testing.T) {
		t.Parallel()

		homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		env := newRegistryTestEnv(t)
		source := newLifecycleSource(t, "1.0.0", "2.0.0")
		loader := func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{source}, nil
		}

		listings, err := SearchMarketplaceExtensions(t.Context(), loader, "life", "github", 10)
		if err != nil {
			t.Fatalf("SearchMarketplaceExtensions() error = %v", err)
		}
		if len(listings) != 1 || listings[0].Slug != "acme/lifecycle-ext" {
			t.Fatalf("SearchMarketplaceExtensions() = %#v, want lifecycle listing", listings)
		}

		source.latestVersion = "1.0.0"
		installed, err := InstallMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceInstallRequest{Slug: "acme/lifecycle-ext", SourceFilter: "github"},
		)
		if err != nil {
			t.Fatalf("InstallMarketplaceManaged() error = %v", err)
		}
		if installed.Source != SourceMarketplace ||
			dereferenceOptionalString(installed.RegistrySlug) != "acme/lifecycle-ext" ||
			dereferenceOptionalString(installed.RegistryName) != "github" ||
			dereferenceOptionalString(installed.RemoteVersion) != "1.0.0" {
			t.Fatalf("installed metadata = %#v, want marketplace provenance", installed)
		}
		requireFileContains(t, filepath.Join(ManagedInstallPath(homePaths, "lifecycle-ext"), "VERSION.txt"), "1.0.0")

		source.latestVersion = "2.0.0"
		checks, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{Names: []string{"lifecycle-ext"}, CheckOnly: true},
			nil,
		)
		if err != nil {
			t.Fatalf("UpdateMarketplaceManaged(check) error = %v", err)
		}
		if len(checks) != 1 || checks[0].Status != MarketplaceUpdateStatusAvailable {
			t.Fatalf("UpdateMarketplaceManaged(check) = %#v, want available update", checks)
		}
		requireFileContains(t, filepath.Join(ManagedInstallPath(homePaths, "lifecycle-ext"), "VERSION.txt"), "1.0.0")

		reloads := 0
		updates, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{Names: []string{"lifecycle-ext"}},
			func(context.Context) error {
				reloads++
				return nil
			},
		)
		if err != nil {
			t.Fatalf("UpdateMarketplaceManaged(apply) error = %v", err)
		}
		if len(updates) != 1 || updates[0].Status != MarketplaceUpdateStatusUpdated || reloads != 1 {
			t.Fatalf("UpdateMarketplaceManaged(apply) = %#v reloads=%d, want updated with one reload", updates, reloads)
		}
		requireFileContains(t, filepath.Join(ManagedInstallPath(homePaths, "lifecycle-ext"), "VERSION.txt"), "2.0.0")

		removeErr := errors.New("reload failed")
		_, err = RemoveManagedExtension(t.Context(), env.registry, "lifecycle-ext", func(context.Context) error {
			return removeErr
		})
		if !errors.Is(err, removeErr) {
			t.Fatalf("RemoveManagedExtension(reload failure) error = %v, want reload failure", err)
		}
		if _, err := env.registry.Get("lifecycle-ext"); err != nil {
			t.Fatalf("registry.Get(after remove rollback) error = %v, want restored row", err)
		}
		requireFileContains(t, filepath.Join(ManagedInstallPath(homePaths, "lifecycle-ext"), "VERSION.txt"), "2.0.0")

		removed, err := RemoveManagedExtension(t.Context(), env.registry, "lifecycle-ext", nil)
		if err != nil {
			t.Fatalf("RemoveManagedExtension() error = %v", err)
		}
		if removed.Status != "removed" {
			t.Fatalf("RemoveManagedExtension() = %#v, want removed status", removed)
		}
		if _, err := env.registry.Get("lifecycle-ext"); !errors.Is(err, ErrExtensionNotFound) {
			t.Fatalf("registry.Get(after remove) error = %v, want ErrExtensionNotFound", err)
		}
		if _, err := os.Stat(ManagedInstallPath(homePaths, "lifecycle-ext")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("os.Stat(managed dir after remove) error = %v, want not exist", err)
		}
	})
}

func TestMarketplaceLifecycleRollsBackFailedUpdateReload(t *testing.T) {
	t.Run("Should roll back failed update reloads", func(t *testing.T) {
		t.Parallel()

		homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		env := newRegistryTestEnv(t)
		source := newLifecycleSource(t, "1.0.0", "2.0.0")
		loader := func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{source}, nil
		}

		source.latestVersion = "1.0.0"
		if _, err := InstallMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceInstallRequest{Slug: "acme/lifecycle-ext"},
		); err != nil {
			t.Fatalf("InstallMarketplaceManaged() error = %v", err)
		}
		source.latestVersion = "2.0.0"

		reloadErr := errors.New("reload rejected update")
		_, err = UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{Names: []string{"lifecycle-ext"}},
			func(context.Context) error {
				return reloadErr
			},
		)
		if !errors.Is(err, reloadErr) {
			t.Fatalf("UpdateMarketplaceManaged(reload failure) error = %v, want reload rejection", err)
		}

		info, err := env.registry.Get("lifecycle-ext")
		if err != nil {
			t.Fatalf("registry.Get(after update rollback) error = %v", err)
		}
		if info.Version != "1.0.0" || dereferenceOptionalString(info.RemoteVersion) != "1.0.0" {
			t.Fatalf("registry info after rollback = %#v, want original version", info)
		}
		requireFileContains(t, filepath.Join(ManagedInstallPath(homePaths, "lifecycle-ext"), "VERSION.txt"), "1.0.0")
	})
}

func TestMarketplaceLifecycleValidatesSourcesAndInputs(t *testing.T) {
	t.Run("Should validate marketplace sources and lifecycle inputs", func(t *testing.T) {
		t.Parallel()

		homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
		if err != nil {
			t.Fatalf("ResolveHomePathsFrom() error = %v", err)
		}
		env := newRegistryTestEnv(t)
		source := newLifecycleSource(t, "1.0.0")

		if _, err := SearchMarketplaceExtensions(t.Context(), nil, "life", "", 10); err == nil {
			t.Fatal("SearchMarketplaceExtensions(nil loader) error = nil, want failure")
		}
		if _, err := SearchMarketplaceExtensions(t.Context(), func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{source}, nil
		}, "life", "", 0); err == nil {
			t.Fatal("SearchMarketplaceExtensions(limit 0) error = nil, want failure")
		}

		primary := newLifecycleSource(t, "1.0.0")
		secondary := newLifecycleSource(t, "1.0.0")
		secondary.name = "secondary"
		filtered, err := LoadMarketplaceSources(t.Context(), func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{primary, secondary}, nil
		}, "github")
		if err != nil {
			t.Fatalf("LoadMarketplaceSources(filter) error = %v", err)
		}
		if len(filtered) != 1 || filtered[0].Name() != "github" {
			t.Fatalf("LoadMarketplaceSources(filter) = %#v, want github source", filtered)
		}
		if secondary.closeCount != 1 {
			t.Fatalf("secondary source close count = %d, want 1", secondary.closeCount)
		}

		if _, err := LoadMarketplaceSources(t.Context(), func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{primary}, nil
		}, "missing"); err == nil {
			t.Fatal("LoadMarketplaceSources(missing filter) error = nil, want failure")
		}
		closeErr := errors.New("close failed")
		closing := newLifecycleSource(t, "1.0.0")
		closing.closeErr = closeErr
		if _, err := LoadMarketplaceSources(t.Context(), func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{closing}, errors.New("loader failed")
		}, ""); !errors.Is(err, closeErr) {
			t.Fatalf("LoadMarketplaceSources(loader close failure) error = %v, want close failure joined", err)
		}

		if _, err := InstalledExtensionDir(
			ExtensionInfo{Name: "bad", ManifestPath: "relative/extension.toml"},
		); err == nil {
			t.Fatal("InstalledExtensionDir(relative) error = nil, want failure")
		}
		if _, err := InstalledExtensionDir(
			ExtensionInfo{Name: "bad", ManifestPath: filepath.Join(t.TempDir(), "README.md")},
		); err == nil {
			t.Fatal("InstalledExtensionDir(non-manifest) error = nil, want failure")
		}

		loader := func(context.Context) ([]registrypkg.Source, error) {
			return []registrypkg.Source{source}, nil
		}
		source.latestVersion = "1.0.0"
		if _, err := InstallMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceInstallRequest{Slug: "acme/lifecycle-ext"},
		); err != nil {
			t.Fatalf("InstallMarketplaceManaged() error = %v", err)
		}
		if _, err := InstallMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceInstallRequest{Slug: "acme/lifecycle-ext"},
		); !errors.Is(err, ErrExtensionExists) {
			t.Fatalf("InstallMarketplaceManaged(duplicate) error = %v, want ErrExtensionExists", err)
		}

		current, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{Names: []string{"lifecycle-ext"}},
			func(context.Context) error {
				return errors.New("reload should not run for current extension")
			},
		)
		if err != nil {
			t.Fatalf("UpdateMarketplaceManaged(current) error = %v", err)
		}
		if len(current) != 1 || current[0].Status != MarketplaceUpdateStatusCurrent {
			t.Fatalf("UpdateMarketplaceManaged(current) = %#v, want current status", current)
		}
		allCurrent, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{All: true},
			nil,
		)
		if err != nil {
			t.Fatalf("UpdateMarketplaceManaged(all current) error = %v", err)
		}
		if len(allCurrent) != 1 || allCurrent[0].Status != MarketplaceUpdateStatusCurrent {
			t.Fatalf("UpdateMarketplaceManaged(all current) = %#v, want one current marketplace extension", allCurrent)
		}

		source.latestVersion = "2.0.0"
		source.archives["2.0.0"] = lifecycleTarGzNamed(t, "renamed-ext", "2.0.0")
		if _, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{Names: []string{"lifecycle-ext"}},
			nil,
		); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("UpdateMarketplaceManaged(identity mismatch) error = %v, want identity mismatch", err)
		}

		if _, err := UpdateMarketplaceManaged(
			t.Context(),
			homePaths,
			env.registry,
			loader,
			MarketplaceUpdateRequest{},
			nil,
		); err == nil {
			t.Fatal("UpdateMarketplaceManaged(missing name) error = nil, want failure")
		}
	})
}

func newLifecycleSource(t *testing.T, versions ...string) *lifecycleSource {
	t.Helper()

	archives := make(map[string][]byte, len(versions))
	for _, version := range versions {
		archives[version] = lifecycleTarGz(t, version)
	}
	return &lifecycleSource{
		name:          "github",
		latestVersion: versions[len(versions)-1],
		archives:      archives,
	}
}

func lifecycleTarGz(t *testing.T, version string) []byte {
	t.Helper()

	return lifecycleTarGzNamed(t, "lifecycle-ext", version)
}

func lifecycleTarGzNamed(t *testing.T, name string, version string) []byte {
	t.Helper()

	files := map[string]string{
		filepath.Join(name, "extension.toml"): strings.Replace(
			registryManifestTOML(name, registryManifestOptions{}),
			`version = "0.2.1"`,
			fmt.Sprintf(`version = %q`, version),
			1,
		),
		filepath.Join(name, "VERSION.txt"): version + "\n",
	}

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("tarWriter.WriteHeader(%q) error = %v", name, err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("tarWriter.Write(%q) error = %v", name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() error = %v", err)
	}
	return buffer.Bytes()
}

func requireFileContains(t *testing.T, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("file %q = %q, want contains %q", path, string(content), want)
	}
}
