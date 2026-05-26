//go:build mage

package main

import (
	"context"
	"errors"
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

func TestBuildSteps(t *testing.T) {
	t.Parallel()

	t.Run("Should compile Go without frontend asset parity checks", func(t *testing.T) {
		t.Parallel()

		names := mageStepNames(buildSteps())
		assertStringListEqual(t, names, []string{"CodegenCheck", "buildGo"})
		assertStringListExcludes(t, names, "WebBuild")
		assertStringListExcludes(t, names, "WebAssetsCheck")
		assertStringListExcludes(t, names, "SourceInstallCheck")
	})
}

func TestVerifySteps(t *testing.T) {
	t.Parallel()

	t.Run("Should keep frontend quality gates without web asset publication checks", func(t *testing.T) {
		t.Parallel()

		names := mageStepNames(verifySteps())
		assertStringListIncludes(t, names, "BunLint")
		assertStringListIncludes(t, names, "BunTypecheck")
		assertStringListIncludes(t, names, "BunTest")
		assertStringListIncludes(t, names, "WebBuild")
		assertStringListExcludes(t, names, "WebAssetsCheck")
		assertStringListExcludes(t, names, "SourceInstallCheck")
	})
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

func TestWithRaceEnabledEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overrides map[string]string
		want      map[string]string
		wantInput map[string]string
	}{
		{
			name: "Should set cgo for race commands without mutating the input",
			overrides: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "0",
			},
			want: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "1",
			},
			wantInput: map[string]string{
				"CI":          "true",
				"CGO_ENABLED": "0",
			},
		},
		{
			name: "Should work with nil input",
			want: map[string]string{
				"CGO_ENABLED": "1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := withRaceEnabledEnv(tt.overrides)
			for key, want := range tt.want {
				if got[key] != want {
					t.Fatalf("withRaceEnabledEnv() %s = %q, want %q", key, got[key], want)
				}
			}
			if tt.wantInput != nil {
				for key, want := range tt.wantInput {
					if tt.overrides[key] != want {
						t.Fatalf("withRaceEnabledEnv() mutated input %s to %q, want %q", key, tt.overrides[key], want)
					}
				}

				got["EXTRA"] = "value"
				if _, ok := tt.overrides["EXTRA"]; ok {
					t.Fatal("withRaceEnabledEnv() reused the input map")
				}
			}
		})
	}
}

func TestRunRaceEnabledGoCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should wrap subprocess failures with command context", func(t *testing.T) {
		t.Parallel()

		err := runRaceEnabledGoCommand(context.Background(), nil, "definitely-not-a-go-subcommand")
		if err == nil {
			t.Fatal("runRaceEnabledGoCommand() error = nil, want non-nil")
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want exec.ExitError in chain", err)
		}
		if !strings.Contains(err.Error(), "definitely-not-a-go-subcommand") {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want command arguments in message", err)
		}
	})

	t.Run("Should respect canceled contexts", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := runRaceEnabledGoCommand(ctx, nil, "version")
		if err == nil {
			t.Fatal("runRaceEnabledGoCommand() error = nil, want context cancellation")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runRaceEnabledGoCommand() error = %v, want context.Canceled in chain", err)
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

func mageStepNames(steps []mageStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.name)
	}
	return names
}

func assertStringListEqual(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("list = %#v, want %#v", got, want)
	}
	for idx := range got {
		if got[idx] != want[idx] {
			t.Fatalf("list = %#v, want %#v", got, want)
		}
	}
}

func assertStringListIncludes(t *testing.T, values []string, want string) {
	t.Helper()

	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("list = %#v, want to include %q", values, want)
}

func assertStringListExcludes(t *testing.T, values []string, unwanted string) {
	t.Helper()

	for _, value := range values {
		if value == unwanted {
			t.Fatalf("list = %#v, want to exclude %q", values, unwanted)
		}
	}
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
