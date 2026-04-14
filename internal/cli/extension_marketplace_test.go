package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

type extensionRegistryTestEnv struct {
	deps      commandDeps
	homePaths aghconfig.HomePaths
}

type extensionRegistrySourceStub struct {
	name         string
	caps         registrypkg.SourceCaps
	searchFunc   func(context.Context, string, registrypkg.SearchOpts) ([]registrypkg.Listing, error)
	infoFunc     func(context.Context, string) (*registrypkg.Detail, error)
	downloadFunc func(context.Context, string, registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error)
	closeFunc    func() error
}

func (s *extensionRegistrySourceStub) Name() string {
	return s.name
}

func (s *extensionRegistrySourceStub) Capabilities() registrypkg.SourceCaps {
	return s.caps
}

func (s *extensionRegistrySourceStub) Search(ctx context.Context, query string, opts registrypkg.SearchOpts) ([]registrypkg.Listing, error) {
	if s.searchFunc == nil {
		return nil, nil
	}
	return s.searchFunc(ctx, query, opts)
}

func (s *extensionRegistrySourceStub) Info(ctx context.Context, slug string) (*registrypkg.Detail, error) {
	if s.infoFunc == nil {
		return nil, fmt.Errorf("missing info for %q", slug)
	}
	return s.infoFunc(ctx, slug)
}

func (s *extensionRegistrySourceStub) Download(ctx context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
	if s.downloadFunc == nil {
		return nil, fmt.Errorf("missing download for %q", slug)
	}
	return s.downloadFunc(ctx, slug, opts)
}

func (s *extensionRegistrySourceStub) Close() error {
	if s.closeFunc == nil {
		return nil
	}
	return s.closeFunc()
}

func TestExtensionSearchCommandUsesSearchableRegistrySources(t *testing.T) {
	t.Parallel()

	skippedSearchCalls := 0
	env := newExtensionRegistryTestEnv(t,
		&extensionRegistrySourceStub{
			name: "github",
			caps: registrypkg.SourceCaps{Search: false},
			searchFunc: func(context.Context, string, registrypkg.SearchOpts) ([]registrypkg.Listing, error) {
				skippedSearchCalls++
				return nil, errors.New("unexpected search call")
			},
		},
		&extensionRegistrySourceStub{
			name: "registry",
			caps: registrypkg.SourceCaps{Search: true},
			searchFunc: func(_ context.Context, query string, opts registrypkg.SearchOpts) ([]registrypkg.Listing, error) {
				if query != "bridge" {
					t.Fatalf("search query = %q, want %q", query, "bridge")
				}
				if opts.Limit != 7 {
					t.Fatalf("search limit = %d, want 7", opts.Limit)
				}
				if opts.Type != registrypkg.PackageTypeExtension {
					t.Fatalf("search type = %q, want %q", opts.Type, registrypkg.PackageTypeExtension)
				}
				return []registrypkg.Listing{{
					Slug:        "acme/bridge-ext",
					Name:        "bridge-ext",
					Description: "Bridge adapter",
					Author:      "acme",
					Version:     "1.2.0",
					Downloads:   42,
					Type:        registrypkg.PackageTypeExtension,
					Source:      "registry",
				}}, nil
			},
		},
	)

	stdout, _, err := executeRootCommand(t, env.deps, "extension", "search", "bridge", "--limit", "7", "-o", "json")
	if err != nil {
		t.Fatalf("extension search error = %v", err)
	}

	var listings []registrypkg.Listing
	if err := json.Unmarshal([]byte(stdout), &listings); err != nil {
		t.Fatalf("json.Unmarshal(extension search) error = %v; stdout=%s", err, stdout)
	}
	if len(listings) != 1 || listings[0].Slug != "acme/bridge-ext" {
		t.Fatalf("extension search listings = %#v, want bridge-ext result", listings)
	}
	if skippedSearchCalls != 0 {
		t.Fatalf("non-searchable source search calls = %d, want 0", skippedSearchCalls)
	}
}

func TestExtensionInstallCommandInstallsMarketplaceExtensionAndPrintsRestartMessage(t *testing.T) {
	t.Parallel()

	downloadCalls := 0
	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		caps: registrypkg.SourceCaps{Search: false},
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{
				Listing: registrypkg.Listing{
					Slug:    slug,
					Name:    "remote-ext",
					Version: "1.0.0",
					Source:  "github",
				},
			}, nil
		},
		downloadFunc: func(_ context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			downloadCalls++
			if slug != "acme/remote-ext" {
				t.Fatalf("download slug = %q, want %q", slug, "acme/remote-ext")
			}
			if opts.Version != "" {
				t.Fatalf("download version = %q, want empty latest", opts.Version)
			}
			return newExtensionDownloadResult(t, slug, "1.0.0", remoteExtensionArchiveFiles("remote-ext", "1.0.0")), nil
		},
	})
	env.deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
		return aghdaemon.Info{PID: 999, StartedAt: fixedTestNow}, nil
	}
	env.deps.processAlive = func(int) bool { return true }
	env.deps.newClient = func(string) (DaemonClient, error) {
		return nil, errors.New("unexpected daemon client call for registry install")
	}

	stdout, stderr, err := executeRootCommand(t, env.deps, "extension", "install", "acme/remote-ext", "-o", "json")
	if err != nil {
		t.Fatalf("extension install registry error = %v", err)
	}

	var payload ExtensionRecord
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(extension install) error = %v; stdout=%s", err, stdout)
	}
	if payload.Name != "remote-ext" || payload.Source != "marketplace" {
		t.Fatalf("extension install payload = %#v, want marketplace remote-ext", payload)
	}
	if !strings.Contains(stderr, "Restart the daemon to activate") {
		t.Fatalf("extension install stderr = %q, want restart guidance", stderr)
	}
	if downloadCalls != 1 {
		t.Fatalf("download calls = %d, want 1", downloadCalls)
	}

	info := getInstalledExtension(t, env.homePaths, "remote-ext")
	if info.Source != extensionpkg.SourceMarketplace {
		t.Fatalf("installed source = %v, want marketplace", info.Source)
	}
	if info.RegistrySlug == nil || *info.RegistrySlug != "acme/remote-ext" {
		t.Fatalf("installed RegistrySlug = %#v, want acme/remote-ext", info.RegistrySlug)
	}
	if info.RegistryName == nil || *info.RegistryName != "github" {
		t.Fatalf("installed RegistryName = %#v, want github", info.RegistryName)
	}
	if info.RemoteVersion == nil || *info.RemoteVersion != "1.0.0" {
		t.Fatalf("installed RemoteVersion = %#v, want 1.0.0", info.RemoteVersion)
	}

	manifestPath := filepath.Join(managedExtensionInstallPath(env.homePaths, "remote-ext"), "extension.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("installed extension manifest stat error = %v", err)
	}
}

func TestExtensionRemoveCommandDeletesDirectoryAndRegistryRecord(t *testing.T) {
	t.Parallel()

	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{Listing: registrypkg.Listing{
				Slug:    slug,
				Name:    "remove-ext",
				Version: "1.0.0",
				Source:  "github",
			}}, nil
		},
		downloadFunc: func(_ context.Context, slug string, _ registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			return newExtensionDownloadResult(t, slug, "1.0.0", remoteExtensionArchiveFiles("remove-ext", "1.0.0")), nil
		},
	})

	if _, _, err := executeRootCommand(t, env.deps, "extension", "install", "acme/remove-ext", "-o", "json"); err != nil {
		t.Fatalf("extension install before remove error = %v", err)
	}

	stdout, _, err := executeRootCommand(t, env.deps, "extension", "remove", "remove-ext", "-o", "json")
	if err != nil {
		t.Fatalf("extension remove error = %v", err)
	}

	var payload extensionRemoveItem
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(extension remove) error = %v; stdout=%s", err, stdout)
	}
	if payload.Status != "removed" {
		t.Fatalf("extension remove payload = %#v, want removed status", payload)
	}

	installDir := managedExtensionInstallPath(env.homePaths, "remove-ext")
	if _, err := os.Stat(installDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("extension install dir stat error = %v, want not exist", err)
	}

	registry, cleanup := openExtensionRegistry(t, env.homePaths)
	defer cleanup()
	_, err = registry.Get("remove-ext")
	if !errors.Is(err, extensionpkg.ErrExtensionNotFound) {
		t.Fatalf("registry.Get(remove-ext) error = %v, want ErrExtensionNotFound", err)
	}
}

func TestExtensionRemoveCommandReturnsClearErrorForMissingExtension(t *testing.T) {
	t.Parallel()

	env := newExtensionRegistryTestEnv(t)

	_, _, err := executeRootCommand(t, env.deps, "extension", "remove", "missing-ext", "-o", "json")
	if err == nil {
		t.Fatal("extension remove missing error = nil, want failure")
	}
	if !errors.Is(err, extensionpkg.ErrExtensionNotFound) {
		t.Fatalf("extension remove missing error = %v, want ErrExtensionNotFound", err)
	}
}

func TestExtensionUpdateCommandCheckOnlyShowsAvailableUpdatesWithoutDownloading(t *testing.T) {
	t.Parallel()

	latestVersion := "1.0.0"
	downloadCalls := 0
	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{Listing: registrypkg.Listing{
				Slug:    slug,
				Name:    "update-ext",
				Version: latestVersion,
				Source:  "github",
			}}, nil
		},
		downloadFunc: func(_ context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			downloadCalls++
			version := firstNonEmpty(opts.Version, latestVersion)
			return newExtensionDownloadResult(t, slug, version, remoteExtensionArchiveFiles("update-ext", version)), nil
		},
	})

	if _, _, err := executeRootCommand(t, env.deps, "extension", "install", "acme/update-ext", "-o", "json"); err != nil {
		t.Fatalf("extension install before update check error = %v", err)
	}
	latestVersion = "1.2.0"

	stdout, _, err := executeRootCommand(t, env.deps, "extension", "update", "update-ext", "--check", "-o", "json")
	if err != nil {
		t.Fatalf("extension update --check error = %v", err)
	}

	var items []extensionUpdateItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("json.Unmarshal(extension update --check) error = %v; stdout=%s", err, stdout)
	}
	if len(items) != 1 || items[0].Status != "update available" || items[0].LatestVersion != "1.2.0" {
		t.Fatalf("extension update --check items = %#v, want update available to 1.2.0", items)
	}
	if downloadCalls != 1 {
		t.Fatalf("download calls after --check = %d, want only the initial install download", downloadCalls)
	}
}

func TestExtensionUpdateCommandReinstallsNewerVersion(t *testing.T) {
	t.Parallel()

	latestVersion := "1.0.0"
	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{Listing: registrypkg.Listing{
				Slug:    slug,
				Name:    "update-ext",
				Version: latestVersion,
				Source:  "github",
			}}, nil
		},
		downloadFunc: func(_ context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			version := firstNonEmpty(opts.Version, latestVersion)
			return newExtensionDownloadResult(t, slug, version, remoteExtensionArchiveFiles("update-ext", version)), nil
		},
	})

	if _, _, err := executeRootCommand(t, env.deps, "extension", "install", "acme/update-ext", "-o", "json"); err != nil {
		t.Fatalf("extension install before update error = %v", err)
	}
	latestVersion = "1.2.0"

	stdout, stderr, err := executeRootCommand(t, env.deps, "extension", "update", "update-ext", "-o", "json")
	if err != nil {
		t.Fatalf("extension update error = %v", err)
	}

	var items []extensionUpdateItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("json.Unmarshal(extension update) error = %v; stdout=%s", err, stdout)
	}
	if len(items) != 1 || items[0].Status != "updated" || items[0].LatestVersion != "1.2.0" {
		t.Fatalf("extension update items = %#v, want updated to 1.2.0", items)
	}
	if !strings.Contains(stderr, "Restart the daemon to activate") {
		t.Fatalf("extension update stderr = %q, want restart guidance", stderr)
	}

	info := getInstalledExtension(t, env.homePaths, "update-ext")
	if info.Version != "1.2.0" {
		t.Fatalf("installed version after update = %q, want %q", info.Version, "1.2.0")
	}
	if info.RemoteVersion == nil || *info.RemoteVersion != "1.2.0" {
		t.Fatalf("installed RemoteVersion after update = %#v, want 1.2.0", info.RemoteVersion)
	}

	manifest, err := extensionpkg.LoadManifest(managedExtensionInstallPath(env.homePaths, "update-ext"))
	if err != nil {
		t.Fatalf("LoadManifest(updated extension) error = %v", err)
	}
	if manifest.Version != "1.2.0" {
		t.Fatalf("manifest version after update = %q, want %q", manifest.Version, "1.2.0")
	}
}

func newExtensionRegistryTestEnv(t *testing.T, sources ...registrypkg.RegistrySource) extensionRegistryTestEnv {
	t.Helper()

	deps, homePaths := newExtensionLocalDeps(t, stubClient{})
	cfg := aghconfig.DefaultWithHome(homePaths)
	deps.loadConfig = func() (aghconfig.Config, error) {
		return cfg, nil
	}
	deps.loadExtensionRegistrySources = func(runtimeContext) ([]registrypkg.RegistrySource, error) {
		return sources, nil
	}

	return extensionRegistryTestEnv{
		deps:      deps,
		homePaths: homePaths,
	}
}

func newExtensionDownloadResult(t *testing.T, slug string, version string, files map[string]string) *registrypkg.DownloadResult {
	t.Helper()

	return &registrypkg.DownloadResult{
		Reader:      io.NopCloser(bytes.NewReader(mustTarGz(t, files))),
		Slug:        slug,
		Version:     version,
		ContentSize: -1,
		ContentType: "application/gzip",
	}
}

func remoteExtensionArchiveFiles(name string, version string) map[string]string {
	return map[string]string{
		filepath.Join(name, "extension.toml"): strings.Replace(extensionFixtureManifest(name, extensionFixtureOptions{}), `version = "0.1.0"`, fmt.Sprintf(`version = %q`, version), 1),
		filepath.Join(name, "README.md"):      "remote fixture\n",
	}
}
