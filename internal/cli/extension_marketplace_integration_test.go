//go:build integration

package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
	registrypkg "github.com/compozy/agh/internal/registry"
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
}

var _ registrypkg.Source = (*extensionRegistrySourceStub)(nil)

func newExtensionRegistryTestEnv(t *testing.T, sources ...registrypkg.Source) extensionRegistryTestEnv {
	t.Helper()

	h := newIntegrationHarness(t)
	h.runner.extensionSources = append([]registrypkg.Source(nil), sources...)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	t.Cleanup(func() {
		if _, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json"); err != nil {
			t.Fatalf("daemon stop cleanup error = %v", err)
		}
	})
	return extensionRegistryTestEnv{deps: h.deps, homePaths: h.homePaths}
}

func (s *extensionRegistrySourceStub) Name() string {
	if strings.TrimSpace(s.name) == "" {
		return "github"
	}
	return s.name
}

func (s *extensionRegistrySourceStub) Capabilities() registrypkg.SourceCaps {
	return s.caps
}

func (s *extensionRegistrySourceStub) Search(
	ctx context.Context,
	query string,
	opts registrypkg.SearchOpts,
) ([]registrypkg.Listing, error) {
	if s.searchFunc != nil {
		return s.searchFunc(ctx, query, opts)
	}
	return nil, errors.New("extension registry search is not configured")
}

func (s *extensionRegistrySourceStub) Info(ctx context.Context, slug string) (*registrypkg.Detail, error) {
	if s.infoFunc != nil {
		return s.infoFunc(ctx, slug)
	}
	return nil, errors.New("extension registry info is not configured")
}

func (s *extensionRegistrySourceStub) Download(
	ctx context.Context,
	slug string,
	opts registrypkg.DownloadOpts,
) (*registrypkg.DownloadResult, error) {
	if s.downloadFunc != nil {
		return s.downloadFunc(ctx, slug, opts)
	}
	return nil, errors.New("extension registry download is not configured")
}

func (s *extensionRegistrySourceStub) Close() error {
	return nil
}

func newExtensionDownloadResult(
	t *testing.T,
	slug string,
	version string,
	files map[string]string,
) *registrypkg.DownloadResult {
	t.Helper()

	archive := extensionArchive(t, files)
	return &registrypkg.DownloadResult{
		Reader:      io.NopCloser(bytes.NewReader(archive)),
		Slug:        slug,
		Version:     version,
		ContentSize: int64(len(archive)),
		ContentType: "application/gzip",
	}
}

func remoteExtensionArchiveFiles(name string, version string) map[string]string {
	return map[string]string{
		filepath.Join(name, "extension.toml"): fmt.Sprintf(
			"[extension]\nname = %q\nversion = %q\ndescription = \"CLI marketplace integration fixture\"\nmin_agh_version = \"0.5.0\"\n\n[capabilities]\nprovides = [\"memory.backend\"]\n\n[actions]\nrequires = [\"sessions/list\"]\n",
			name,
			version,
		),
		filepath.Join(name, "README.md"): "# " + name + "\n",
	}
}

func extensionArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

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

func TestExtensionSearchCommandIntegrationReturnsRegistryListings(t *testing.T) {
	t.Parallel()

	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "registry",
		caps: registrypkg.SourceCaps{Search: true},
		searchFunc: func(_ context.Context, _ string, opts registrypkg.SearchOpts) ([]registrypkg.Listing, error) {
			if opts.Type != registrypkg.PackageTypeExtension {
				t.Fatalf("search type = %q, want %q", opts.Type, registrypkg.PackageTypeExtension)
			}
			return []registrypkg.Listing{{
				Slug:        "acme/telemetry-ext",
				Name:        "telemetry-ext",
				Description: "Telemetry bridge",
				Author:      "acme",
				Version:     "0.9.0",
				Downloads:   11,
				Type:        registrypkg.PackageTypeExtension,
				Source:      "registry",
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, env.deps, "extension", "search", "telemetry", "-o", "json")
	if err != nil {
		t.Fatalf("extension search integration error = %v", err)
	}

	var listings []registrypkg.Listing
	if err := json.Unmarshal([]byte(stdout), &listings); err != nil {
		t.Fatalf("json.Unmarshal(extension search integration) error = %v; stdout=%s", err, stdout)
	}
	if len(listings) != 1 || listings[0].Slug != "acme/telemetry-ext" {
		t.Fatalf("extension search integration listings = %#v, want telemetry-ext", listings)
	}
}

func TestExtensionInstallCommandIntegrationCreatesManagedInstallAndRegistryRecord(t *testing.T) {
	t.Parallel()

	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{Listing: registrypkg.Listing{
				Slug:    slug,
				Name:    "integration-ext",
				Version: "1.0.0",
				Source:  "github",
			}}, nil
		},
		downloadFunc: func(_ context.Context, slug string, _ registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			return newExtensionDownloadResult(
				t,
				slug,
				"1.0.0",
				remoteExtensionArchiveFiles("integration-ext", "1.0.0"),
			), nil
		},
	})

	stdout, stderr, err := executeRootCommand(
		t,
		env.deps,
		"extension",
		"install",
		"acme/integration-ext",
		"--allow-unverified",
		"--yes",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("extension install integration error = %v", err)
	}

	var payload ExtensionRecord
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(extension install integration) error = %v; stdout=%s", err, stdout)
	}
	if payload.Source != "marketplace" {
		t.Fatalf("extension install integration payload = %#v, want marketplace source", payload)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("extension install integration stderr = %q, want empty daemon-managed lifecycle guidance", stderr)
	}

	info := getInstalledExtension(t, env.homePaths, "integration-ext")
	if info.Source != extensionpkg.SourceMarketplace {
		t.Fatalf("installed source = %v, want marketplace", info.Source)
	}
}

func TestExtensionUpdateAndRemoveIntegration(t *testing.T) {
	t.Parallel()

	latestVersion := "1.0.0"
	env := newExtensionRegistryTestEnv(t, &extensionRegistrySourceStub{
		name: "github",
		infoFunc: func(_ context.Context, slug string) (*registrypkg.Detail, error) {
			return &registrypkg.Detail{Listing: registrypkg.Listing{
				Slug:    slug,
				Name:    "integration-update-ext",
				Version: latestVersion,
				Source:  "github",
			}}, nil
		},
		downloadFunc: func(_ context.Context, slug string, opts registrypkg.DownloadOpts) (*registrypkg.DownloadResult, error) {
			version := firstNonEmpty(opts.Version, latestVersion)
			return newExtensionDownloadResult(
				t,
				slug,
				version,
				remoteExtensionArchiveFiles("integration-update-ext", version),
			), nil
		},
	})

	if _, _, err := executeRootCommand(
		t,
		env.deps,
		"extension",
		"install",
		"acme/integration-update-ext",
		"--allow-unverified",
		"--yes",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("extension install before update integration error = %v", err)
	}

	latestVersion = "1.3.0"
	checkOut, _, err := executeRootCommand(
		t,
		env.deps,
		"extension",
		"update",
		"integration-update-ext",
		"--check",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("extension update --check integration error = %v", err)
	}

	var checkItems []extensionUpdateItem
	if err := json.Unmarshal([]byte(checkOut), &checkItems); err != nil {
		t.Fatalf("json.Unmarshal(extension update --check integration) error = %v; stdout=%s", err, checkOut)
	}
	if len(checkItems) != 1 || checkItems[0].Status != "available" {
		t.Fatalf("extension update --check integration items = %#v, want update available", checkItems)
	}

	updateOut, _, err := executeRootCommand(
		t,
		env.deps,
		"extension",
		"update",
		"integration-update-ext",
		"--allow-unverified",
		"--yes",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("extension update integration error = %v", err)
	}
	var updateItems []extensionUpdateItem
	if err := json.Unmarshal([]byte(updateOut), &updateItems); err != nil {
		t.Fatalf("json.Unmarshal(extension update integration) error = %v; stdout=%s", err, updateOut)
	}
	if len(updateItems) != 1 || updateItems[0].Status != "updated" {
		t.Fatalf("extension update integration items = %#v, want updated", updateItems)
	}

	info := getInstalledExtension(t, env.homePaths, "integration-update-ext")
	if info.Version != "1.3.0" {
		t.Fatalf("installed version after update = %q, want %q", info.Version, "1.3.0")
	}

	if _, _, err := executeRootCommand(
		t,
		env.deps,
		"extension",
		"remove",
		"integration-update-ext",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("extension remove integration error = %v", err)
	}

	installDir := extensionpkg.ManagedInstallPath(env.homePaths, "integration-update-ext")
	if _, err := os.Stat(installDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed install dir stat error = %v, want not exist", err)
	}
}

func TestExtensionRemoveMissingIntegrationReturnsClearError(t *testing.T) {
	t.Parallel()

	env := newExtensionRegistryTestEnv(t)

	_, _, err := executeRootCommand(t, env.deps, "extension", "remove", "missing-ext", "-o", "json")
	if err == nil {
		t.Fatal("extension remove missing integration error = nil, want failure")
	}
	if !errors.Is(err, extensionpkg.ErrExtensionNotFound) {
		t.Fatalf("extension remove missing integration error = %v, want ErrExtensionNotFound", err)
	}
}
