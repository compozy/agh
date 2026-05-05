package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagerApplyRelease(t *testing.T) {
	t.Run("Should apply a verified archive and preserve rollback metadata", func(t *testing.T) {
		t.Parallel()

		verifier := &stubBundleVerifier{}
		applier := &recordingBinaryApplier{}
		manager, executablePath := newManagerWithExecutable(t, Config{
			RuntimeOS:      runtimeOSLinux,
			RuntimeArch:    runtimeArchAMD64,
			BundleVerifier: verifier,
			BinaryApplier:  applier,
		})

		archiveBody := createTarGzBinary(t, "agh", []byte("#!/bin/sh\necho updated\n"), 0o755)
		release, _, server := newReleaseFixtureServer(t, manager, assetFixture{
			archiveBody: archiveBody,
			bundleBody:  []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle+json;version=0.3"}`),
		})
		defer server.Close()
		manager.httpClient = server.Client()

		applied, err := manager.ApplyRelease(context.Background(), release)
		if err != nil {
			t.Fatalf("ApplyRelease() error = %v", err)
		}
		if verifier.calls != 1 {
			t.Fatalf("VerifyChecksums() calls = %d, want 1", verifier.calls)
		}
		if applier.applyCalls != 1 {
			t.Fatalf("ApplyBinary() calls = %d, want 1", applier.applyCalls)
		}
		if applied.TargetPath != executablePath || applied.Version != "v1.1.0" {
			t.Fatalf("applied = %#v, want executable %q and version v1.1.0", applied, executablePath)
		}
		if !strings.Contains(applied.BackupPath, ".agh.agh-backup-") {
			t.Fatalf("applied.BackupPath = %q, want sibling backup naming", applied.BackupPath)
		}
		if applier.targetPath != executablePath || applier.mode != 0o755 {
			t.Fatalf("binary apply = %#v, want target %q mode 0755", applier, executablePath)
		}
		if got := string(applier.sourceBytes); !strings.Contains(got, "updated") {
			t.Fatalf("applier.sourceBytes = %q, want extracted updated binary", got)
		}
	})

	t.Run("Should reject releases that do not publish the checksum bundle", func(t *testing.T) {
		t.Parallel()

		manager, _ := newManagerWithExecutable(t, Config{
			RuntimeOS:   runtimeOSLinux,
			RuntimeArch: runtimeArchAMD64,
		})

		archiveName, err := archiveAssetName(manager.runtimeOS, manager.runtimeArch)
		if err != nil {
			t.Fatalf("archiveAssetName() error = %v", err)
		}
		release := &Release{
			Version: "v1.1.0",
			Assets: []ReleaseAsset{
				{Name: archiveName, DownloadURL: "https://example.invalid/archive"},
				{Name: checksumsAssetName, DownloadURL: "https://example.invalid/checksums.txt"},
			},
		}

		_, err = manager.ApplyRelease(context.Background(), release)
		if err == nil {
			t.Fatal("ApplyRelease() error = nil, want missing bundle asset error")
		}
		if !strings.Contains(err.Error(), checksumsBundleAssetName) {
			t.Fatalf("ApplyRelease() error = %v, want missing bundle asset", err)
		}
	})

	t.Run("Should stop before swapping binaries when provenance verification fails", func(t *testing.T) {
		t.Parallel()

		verifier := &stubBundleVerifier{err: errors.New("invalid bundle")}
		applier := &recordingBinaryApplier{}
		manager, _ := newManagerWithExecutable(t, Config{
			RuntimeOS:      runtimeOSLinux,
			RuntimeArch:    runtimeArchAMD64,
			BundleVerifier: verifier,
			BinaryApplier:  applier,
		})

		archiveBody := createTarGzBinary(t, "agh", []byte("#!/bin/sh\necho updated\n"), 0o755)
		release, _, server := newReleaseFixtureServer(t, manager, assetFixture{
			archiveBody: archiveBody,
			bundleBody:  []byte(`{"invalid":true}`),
		})
		defer server.Close()
		manager.httpClient = server.Client()

		_, err := manager.ApplyRelease(context.Background(), release)
		if err == nil {
			t.Fatal("ApplyRelease() error = nil, want provenance failure")
		}
		if !strings.Contains(err.Error(), "invalid bundle") {
			t.Fatalf("ApplyRelease() error = %v, want verifier failure", err)
		}
		if applier.applyCalls != 0 {
			t.Fatalf("ApplyBinary() calls = %d, want 0 after verifier failure", applier.applyCalls)
		}
	})

	t.Run("Should reject archives whose checksum does not match the signed catalog", func(t *testing.T) {
		t.Parallel()

		verifier := &stubBundleVerifier{}
		applier := &recordingBinaryApplier{}
		manager, _ := newManagerWithExecutable(t, Config{
			RuntimeOS:      runtimeOSLinux,
			RuntimeArch:    runtimeArchAMD64,
			BundleVerifier: verifier,
			BinaryApplier:  applier,
		})

		archiveBody := createTarGzBinary(t, "agh", []byte("#!/bin/sh\necho updated\n"), 0o755)
		archiveName, err := archiveAssetName(manager.runtimeOS, manager.runtimeArch)
		if err != nil {
			t.Fatalf("archiveAssetName() error = %v", err)
		}
		release, _, server := newReleaseFixtureServer(t, manager, assetFixture{
			archiveBody:   archiveBody,
			checksumsBody: fmt.Appendf(nil, "%s  %s\n", sha256Hex([]byte("different")), archiveName),
			bundleBody:    []byte(`{}`),
		})
		defer server.Close()
		manager.httpClient = server.Client()

		_, err = manager.ApplyRelease(context.Background(), release)
		if err == nil {
			t.Fatal("ApplyRelease() error = nil, want checksum mismatch")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("ApplyRelease() error = %v, want checksum mismatch", err)
		}
		if applier.applyCalls != 0 {
			t.Fatalf("ApplyBinary() calls = %d, want 0 after checksum mismatch", applier.applyCalls)
		}
	})

	t.Run("Should reject corrupt archives after checksum verification succeeds", func(t *testing.T) {
		t.Parallel()

		verifier := &stubBundleVerifier{}
		applier := &recordingBinaryApplier{}
		manager, _ := newManagerWithExecutable(t, Config{
			RuntimeOS:      runtimeOSLinux,
			RuntimeArch:    runtimeArchAMD64,
			BundleVerifier: verifier,
			BinaryApplier:  applier,
		})

		archiveBody := []byte("not-a-gzip-archive")
		release, _, server := newReleaseFixtureServer(t, manager, assetFixture{
			archiveBody: archiveBody,
			bundleBody:  []byte(`{}`),
		})
		defer server.Close()
		manager.httpClient = server.Client()

		_, err := manager.ApplyRelease(context.Background(), release)
		if err == nil {
			t.Fatal("ApplyRelease() error = nil, want corrupt archive failure")
		}
		if !strings.Contains(err.Error(), "open gzip archive") {
			t.Fatalf("ApplyRelease() error = %v, want gzip failure", err)
		}
		if applier.applyCalls != 0 {
			t.Fatalf("ApplyBinary() calls = %d, want 0 after corrupt archive", applier.applyCalls)
		}
	})

	t.Run("Should fail when downloading a release asset returns a server error", func(t *testing.T) {
		t.Parallel()

		verifier := &stubBundleVerifier{}
		applier := &recordingBinaryApplier{}
		manager, _ := newManagerWithExecutable(t, Config{
			RuntimeOS:      runtimeOSLinux,
			RuntimeArch:    runtimeArchAMD64,
			BundleVerifier: verifier,
			BinaryApplier:  applier,
		})

		release, _, server := newReleaseFixtureServer(t, manager, assetFixture{
			archiveBody:   []byte("unused"),
			archiveStatus: http.StatusInternalServerError,
			bundleBody:    []byte(`{}`),
		})
		defer server.Close()
		manager.httpClient = server.Client()

		_, err := manager.ApplyRelease(context.Background(), release)
		if err == nil {
			t.Fatal("ApplyRelease() error = nil, want download failure")
		}
		if !strings.Contains(err.Error(), "500 Internal Server Error") {
			t.Fatalf("ApplyRelease() error = %v, want server failure", err)
		}
		if applier.applyCalls != 0 {
			t.Fatalf("ApplyBinary() calls = %d, want 0 after download failure", applier.applyCalls)
		}
	})
}

type assetFixture struct {
	archiveBody   []byte
	checksumsBody []byte
	bundleBody    []byte
	archiveStatus int
}

func newReleaseFixtureServer(
	t *testing.T,
	manager *Manager,
	fixture assetFixture,
) (*Release, string, *httptest.Server) {
	t.Helper()

	archiveName, err := archiveAssetName(manager.runtimeOS, manager.runtimeArch)
	if err != nil {
		t.Fatalf("archiveAssetName() error = %v", err)
	}

	checksumsBody := fixture.checksumsBody
	if len(checksumsBody) == 0 {
		checksumsBody = fmt.Appendf(nil, "%s  %s\n", sha256Hex(fixture.archiveBody), archiveName)
	}

	server := newReleaseAssetServer(t, map[string]assetResponse{
		"/archive": {
			status: fixture.archiveStatus,
			body:   fixture.archiveBody,
		},
		"/checksums.txt": {
			body: checksumsBody,
		},
		"/checksums.txt.sigstore.json": {
			body: fixture.bundleBody,
		},
	})

	release := &Release{
		Version:    "v1.1.0",
		ReleaseURL: "https://github.com/compozy/agh/releases/tag/v1.1.0",
		Assets: []ReleaseAsset{
			{Name: archiveName, DownloadURL: server.URL + "/archive"},
			{Name: checksumsAssetName, DownloadURL: server.URL + "/checksums.txt"},
			{Name: checksumsBundleAssetName, DownloadURL: server.URL + "/checksums.txt.sigstore.json"},
		},
	}
	return release, archiveName, server
}
