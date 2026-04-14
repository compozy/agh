//go:build integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	extensionpkg "github.com/pedronauck/agh/internal/extension"
	registrypkg "github.com/pedronauck/agh/internal/registry"
)

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
			return newExtensionDownloadResult(t, slug, "1.0.0", remoteExtensionArchiveFiles("integration-ext", "1.0.0")), nil
		},
	})

	stdout, stderr, err := executeRootCommand(t, env.deps, "extension", "install", "acme/integration-ext", "-o", "json")
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
	if !strings.Contains(stderr, "Restart the daemon to activate") {
		t.Fatalf("extension install integration stderr = %q, want restart guidance", stderr)
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
			return newExtensionDownloadResult(t, slug, version, remoteExtensionArchiveFiles("integration-update-ext", version)), nil
		},
	})

	if _, _, err := executeRootCommand(t, env.deps, "extension", "install", "acme/integration-update-ext", "-o", "json"); err != nil {
		t.Fatalf("extension install before update integration error = %v", err)
	}

	latestVersion = "1.3.0"
	checkOut, _, err := executeRootCommand(t, env.deps, "extension", "update", "integration-update-ext", "--check", "-o", "json")
	if err != nil {
		t.Fatalf("extension update --check integration error = %v", err)
	}

	var checkItems []extensionUpdateItem
	if err := json.Unmarshal([]byte(checkOut), &checkItems); err != nil {
		t.Fatalf("json.Unmarshal(extension update --check integration) error = %v; stdout=%s", err, checkOut)
	}
	if len(checkItems) != 1 || checkItems[0].Status != "update available" {
		t.Fatalf("extension update --check integration items = %#v, want update available", checkItems)
	}

	updateOut, _, err := executeRootCommand(t, env.deps, "extension", "update", "integration-update-ext", "-o", "json")
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

	if _, _, err := executeRootCommand(t, env.deps, "extension", "remove", "integration-update-ext", "-o", "json"); err != nil {
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
