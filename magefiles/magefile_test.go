//go:build mage

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/e2elane"
)

func TestShouldEnsureWebBundle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan e2elane.Plan
		want bool
	}{
		{
			name: "Should require the bundle for runtime Go suites",
			plan: e2elane.Plan{
				GoSuites: []e2elane.GoSuite{{Packages: []string{"./internal/daemon"}}},
			},
			want: true,
		},
		{
			name: "Should require the bundle for daemon-served browser suites",
			plan: e2elane.Plan{
				ScriptSuites:                []e2elane.ScriptSuite{{Dir: "web", Script: "test:e2e:daemon-served"}},
				RequiresDaemonServedBrowser: true,
			},
			want: true,
		},
		{
			name: "Should skip the bundle for non-browser script suites alone",
			plan: e2elane.Plan{
				ScriptSuites: []e2elane.ScriptSuite{{Dir: "scripts", Script: "echo"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldEnsureWebBundle(tt.plan); got != tt.want {
				t.Fatalf("shouldEnsureWebBundle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryDigest(t *testing.T) {
	t.Parallel()

	t.Run("Should be stable for the same relative paths and bytes", func(t *testing.T) {
		t.Parallel()

		first := t.TempDir()
		writeTestFile(t, first, "b.txt", "two")
		writeTestFile(t, first, "nested/a.txt", "one")
		firstDigest, err := directoryDigest(first)
		if err != nil {
			t.Fatalf("directoryDigest(first) error = %v", err)
		}

		second := t.TempDir()
		writeTestFile(t, second, "nested/a.txt", "one")
		writeTestFile(t, second, "b.txt", "two")
		secondDigest, err := directoryDigest(second)
		if err != nil {
			t.Fatalf("directoryDigest(second) error = %v", err)
		}

		if firstDigest != secondDigest {
			t.Fatalf("directoryDigest() = %s and %s, want stable digest", firstDigest, secondDigest)
		}
	})

	t.Run("Should change when file contents change", func(t *testing.T) {
		t.Parallel()

		first := t.TempDir()
		writeTestFile(t, first, "index.html", "one")
		firstDigest, err := directoryDigest(first)
		if err != nil {
			t.Fatalf("directoryDigest(first) error = %v", err)
		}

		second := t.TempDir()
		writeTestFile(t, second, "index.html", "two")
		secondDigest, err := directoryDigest(second)
		if err != nil {
			t.Fatalf("directoryDigest(second) error = %v", err)
		}
		if firstDigest == secondDigest {
			t.Fatalf("directoryDigest() = %s for different contents", firstDigest)
		}
	})
}

func TestWebAssetsMetadata(t *testing.T) {
	t.Parallel()

	t.Run("Should write deterministic metadata without timestamps", func(t *testing.T) {
		t.Parallel()

		repoDir := t.TempDir()
		metadata := webAssetsMetadata{
			BuildDigest:      "digest-123",
			SourceRepository: webAssetsSourceRepository,
			SourceCommit:     "abcdef123456",
		}
		if err := writeWebAssetsMetadata(repoDir, metadata); err != nil {
			t.Fatalf("writeWebAssetsMetadata(first) error = %v", err)
		}
		first := readTestFile(t, repoDir, webAssetsMetadataFile)
		if err := writeWebAssetsMetadata(repoDir, metadata); err != nil {
			t.Fatalf("writeWebAssetsMetadata(second) error = %v", err)
		}
		second := readTestFile(t, repoDir, webAssetsMetadataFile)
		if first != second {
			t.Fatalf("metadata changed between identical writes")
		}
		for _, forbidden := range []string{"GeneratedAt", "generated_at", "timestamp", "time.Now"} {
			if strings.Contains(first, forbidden) {
				t.Fatalf("metadata contains %q: %s", forbidden, first)
			}
		}

		parsed, err := readWebAssetsMetadata(repoDir)
		if err != nil {
			t.Fatalf("readWebAssetsMetadata() error = %v", err)
		}
		if parsed != metadata {
			t.Fatalf("readWebAssetsMetadata() = %#v, want %#v", parsed, metadata)
		}
	})
}

func TestWebAssetsNextTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want string
	}{
		{
			name: "Should start at v0.0.1 when no semver tags exist",
			tags: []string{"latest", "assets"},
			want: "v0.0.1",
		},
		{
			name: "Should increment the highest semver patch",
			tags: []string{"v0.0.2", "v0.0.10", "v1.2.3", "not-a-tag"},
			want: "v1.2.4",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := nextWebAssetsTag(tt.tags)
			if err != nil {
				t.Fatalf("nextWebAssetsTag() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("nextWebAssetsTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWebAssetsReleaseSyncHelpers(t *testing.T) {
	t.Run("Should prefer the dedicated web assets token over the release token", func(t *testing.T) {
		t.Setenv(webAssetsTokenEnvVar, "assets-token")
		t.Setenv(releaseTokenEnvVar, "release-token")

		if got := webAssetsPublishToken(); got != "assets-token" {
			t.Fatalf("webAssetsPublishToken() = %q, want dedicated token", got)
		}
	})

	t.Run("Should fall back to the release token", func(t *testing.T) {
		t.Setenv(webAssetsTokenEnvVar, "")
		t.Setenv(releaseTokenEnvVar, "release-token")

		if got := webAssetsPublishToken(); got != "release-token" {
			t.Fatalf("webAssetsPublishToken() = %q, want release token", got)
		}
	})

	t.Run("Should keep Git credentials executable without writing the token to disk", func(t *testing.T) {
		t.Parallel()

		credentials, err := newWebAssetsGitCredentials("secret-token")
		if err != nil {
			t.Fatalf("newWebAssetsGitCredentials() error = %v", err)
		}
		t.Cleanup(credentials.cleanup)

		if got := credentials.env["AGH_WEB_ASSETS_GIT_TOKEN"]; got != "secret-token" {
			t.Fatalf("AGH_WEB_ASSETS_GIT_TOKEN = %q, want secret token", got)
		}
		if got := credentials.env["GIT_TERMINAL_PROMPT"]; got != "0" {
			t.Fatalf("GIT_TERMINAL_PROMPT = %q, want 0", got)
		}
		askpassPath := credentials.env["GIT_ASKPASS"]
		if askpassPath == "" {
			t.Fatal("GIT_ASKPASS is empty")
		}
		info, err := os.Stat(askpassPath)
		if err != nil {
			t.Fatalf("stat askpass helper: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("askpass helper mode = %v, want 0700", got)
		}
		askpassSource := readTestFile(t, filepath.Dir(askpassPath), filepath.Base(askpassPath))
		if strings.Contains(askpassSource, "secret-token") {
			t.Fatal("askpass helper wrote the token to disk")
		}

		assertAskpassOutput(t, credentials.env, askpassPath, "Username for https://github.com", "x-access-token")
		assertAskpassOutput(t, credentials.env, askpassPath, "Password for https://github.com", "secret-token")
	})

	t.Run("Should force public module resolution through the Go proxy", func(t *testing.T) {
		t.Parallel()

		env := webAssetsPublicModuleEnv("/tmp/agh-web-assets-test")
		want := map[string]string{
			"GO111MODULE": "on",
			"GOFLAGS":     "-mod=mod",
			"GONOPROXY":   "",
			"GONOSUMDB":   "",
			"GOPRIVATE":   "",
			"GOPROXY":     "https://proxy.golang.org,direct",
			"GOSUMDB":     "sum.golang.org",
			"GOMODCACHE":  filepath.Join("/tmp/agh-web-assets-test", "mod"),
			"GOPATH":      filepath.Join("/tmp/agh-web-assets-test", "gopath"),
		}
		for key, value := range want {
			if env[key] != value {
				t.Fatalf("webAssetsPublicModuleEnv() %s = %q, want %q", key, env[key], value)
			}
		}
	})

	t.Run("Should parse generated assets metadata from a tag", func(t *testing.T) {
		t.Parallel()

		source := strings.Join([]string{
			"package webassets",
			"const (",
			"\tBuildDigest = \"digest-123\"",
			"\tSourceRepository = \"github.com/compozy/agh\"",
			"\tSourceCommit = \"abcdef123456\"",
			")",
		}, "\n")
		got := parseWebAssetsMetadataSource(source)
		want := webAssetsMetadata{
			BuildDigest:      "digest-123",
			SourceRepository: webAssetsSourceRepository,
			SourceCommit:     "abcdef123456",
		}
		if got != want {
			t.Fatalf("parseWebAssetsMetadataSource() = %#v, want %#v", got, want)
		}
	})
}

func TestWebAssetsPrepare(t *testing.T) {
	t.Parallel()

	t.Run("Should be a no-op for the same BuildDigest and SourceCommit", func(t *testing.T) {
		t.Parallel()

		srcDist := t.TempDir()
		writeTestFile(t, srcDist, "index.html", "<!doctype html><div id=\"app\"></div>")
		writeTestFile(t, srcDist, "assets/app.js", "console.log('app');")
		buildDigest, err := directoryDigest(srcDist)
		if err != nil {
			t.Fatalf("directoryDigest(srcDist) error = %v", err)
		}
		metadata := webAssetsMetadata{
			BuildDigest:      buildDigest,
			SourceRepository: webAssetsSourceRepository,
			SourceCommit:     "source-commit-1",
		}

		repoDir := t.TempDir()
		if err := prepareWebAssetsRepo(srcDist, repoDir, metadata); err != nil {
			t.Fatalf("prepareWebAssetsRepo(first) error = %v", err)
		}
		firstDigest, err := directoryDigest(repoDir)
		if err != nil {
			t.Fatalf("directoryDigest(repo first) error = %v", err)
		}

		if err := prepareWebAssetsRepo(srcDist, repoDir, metadata); err != nil {
			t.Fatalf("prepareWebAssetsRepo(second) error = %v", err)
		}
		secondDigest, err := directoryDigest(repoDir)
		if err != nil {
			t.Fatalf("directoryDigest(repo second) error = %v", err)
		}
		if firstDigest != secondDigest {
			t.Fatalf("repo digest changed for identical metadata: %s != %s", firstDigest, secondDigest)
		}
	})
}

func TestWebAssetsDeterminismCheck(t *testing.T) {
	t.Parallel()

	t.Run("Should require two clean builds with matching digests", func(t *testing.T) {
		t.Parallel()

		cleanCount := 0
		buildCount := 0
		err := webAssetsDeterminismCheck(
			func() error {
				buildCount++
				return nil
			},
			func() error {
				cleanCount++
				return nil
			},
			func() (string, error) {
				return "same-digest", nil
			},
		)
		if err != nil {
			t.Fatalf("webAssetsDeterminismCheck() error = %v", err)
		}
		if cleanCount != 2 || buildCount != 2 {
			t.Fatalf("clean/build counts = %d/%d, want 2/2", cleanCount, buildCount)
		}
	})

	t.Run("Should fail when clean builds produce different digests", func(t *testing.T) {
		t.Parallel()

		digests := []string{"first", "second"}
		err := webAssetsDeterminismCheck(
			func() error { return nil },
			func() error { return nil },
			func() (string, error) {
				next := digests[0]
				digests = digests[1:]
				return next, nil
			},
		)
		if err == nil {
			t.Fatal("webAssetsDeterminismCheck() error = nil, want mismatch error")
		}
		if !strings.Contains(err.Error(), "not deterministic") {
			t.Fatalf("webAssetsDeterminismCheck() error = %v, want determinism message", err)
		}
	})
}

func TestInstallerCheck(t *testing.T) {
	t.Parallel()

	t.Run("Should validate the installer script in dry-run mode", func(t *testing.T) {
		t.Parallel()

		if err := InstallerCheck(); err != nil {
			t.Fatalf("InstallerCheck() error = %v", err)
		}
	})
}

func writeTestFile(t *testing.T, root string, rel string, body string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s) error = %v", rel, err)
	}
}

func readTestFile(t *testing.T, root string, rel string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", rel, err)
	}
	return string(data)
}

func assertAskpassOutput(t *testing.T, env map[string]string, askpassPath string, prompt string, want string) {
	t.Helper()

	cmd := exec.Command(askpassPath, prompt)
	cmd.Env = mergeCommandEnv(env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("askpass %q error = %v\n%s", prompt, err, output)
	}
	if got := strings.TrimSpace(string(output)); got != want {
		t.Fatalf("askpass %q output = %q, want %q", prompt, got, want)
	}
}
