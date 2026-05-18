package e2elane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoModuleToolchainContract(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)

	t.Run("Should declare the verified Go minimum version", func(t *testing.T) {
		t.Parallel()

		goMod := readTextFile(t, filepath.Join(repoRoot, "go.mod"))
		if !strings.Contains(goMod, "\ngo 1.25.5\n") {
			t.Fatalf("go.mod does not declare the verified minimum Go version: %q", goMod)
		}
		if strings.Contains(goMod, "\ntoolchain ") {
			t.Fatalf("go.mod uses a preferred toolchain without changing the verified minimum version: %q", goMod)
		}
	})

	t.Run("Should install the preferred Go patch version in CI and release lanes", func(t *testing.T) {
		t.Parallel()

		files := []string{
			".github/actions/setup-go/action.yml",
			".github/workflows/ci.yml",
			".github/workflows/release.yml",
		}
		for _, file := range files {
			t.Run("Should pin "+file+" to Go 1.25.5", func(t *testing.T) {
				t.Parallel()

				contents := readTextFile(t, filepath.Join(repoRoot, file))
				if !strings.Contains(contents, "1.25.5") {
					t.Fatalf("%s does not reference preferred Go version 1.25.5", file)
				}
				if strings.Contains(contents, "1.25.4") {
					t.Fatalf("%s still references stale Go version 1.25.4", file)
				}
			})
		}
	})
}

func TestReleaseWorkflowGeneratesSiteChangelogContract(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	releaseWorkflow := readTextFile(t, filepath.Join(repoRoot, ".github", "workflows", "release.yml"))
	ciWorkflow := readTextFile(t, filepath.Join(repoRoot, ".github", "workflows", "ci.yml"))

	t.Run("Should run the releasepr version with configured artifact hooks", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(releaseWorkflow, "PR_RELEASE_MODULE: github.com/compozy/releasepr@v0.0.22") {
			t.Fatalf("release workflow does not reference releasepr v0.0.22")
		}
	})

	t.Run("Should install Bun before running pr-release", func(t *testing.T) {
		t.Parallel()

		setupBunIndex := strings.Index(releaseWorkflow, "uses: ./.github/actions/setup-bun")
		prReleaseIndex := strings.Index(releaseWorkflow, "Run PR Release Orchestrator")
		if setupBunIndex == -1 {
			t.Fatal("release-pr job does not install Bun for release artifact generation")
		}
		if prReleaseIndex == -1 {
			t.Fatal("release workflow does not run the PR release orchestrator")
		}
		if setupBunIndex > prReleaseIndex {
			t.Fatal("release-pr job installs Bun after running pr-release")
		}
	})

	t.Run("Should validate current and historical release outputs", func(t *testing.T) {
		t.Parallel()

		for _, want := range []string{
			"[ ! -s RELEASE_BODY.md ]",
			"[ ! -s RELEASE_NOTES.md ]",
			"packages/site/content/blog/changelog",
		} {
			if !strings.Contains(releaseWorkflow, want) {
				t.Fatalf("release workflow does not validate %q", want)
			}
		}
	})

	t.Run("Should feed GoReleaser the current release body", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(releaseWorkflow, "--release-notes=RELEASE_BODY.md") {
			t.Fatal("release workflow does not pass RELEASE_BODY.md to GoReleaser")
		}
		if strings.Contains(releaseWorkflow, "--release-notes=RELEASE_NOTES.md") {
			t.Fatal("release workflow still passes historical RELEASE_NOTES.md to GoReleaser")
		}
	})

	t.Run("Should treat release automation config as relevant CI input", func(t *testing.T) {
		t.Parallel()

		for _, want := range []string{".pr-release", "RELEASE_BODY.md", "RELEASE_NOTES.md"} {
			if !strings.Contains(ciWorkflow, want) {
				t.Fatalf("CI change detector does not include %q", want)
			}
		}
	})
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return string(contents)
}
