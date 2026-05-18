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

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return string(contents)
}
