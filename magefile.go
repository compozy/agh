//go:build mage

package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	golangciLintVersion = "v2.11.4"
	binDir              = "bin"
	cliBinary           = "agh"
	versionPackage      = "github.com/pedronauck/agh/internal/version"
)

var Default = Verify

func Deps() error {
	return sh.RunV("go", "mod", "tidy")
}

func Fmt() error {
	files, err := goFiles(".")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"-w"}, files...)
	return sh.RunV("gofmt", args...)
}

func Lint() error {
	return sh.RunV(
		"go",
		"run",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"+golangciLintVersion,
		"run",
		"./...",
	)
}

// Test runs unit tests only (no integration tag).
func Test() error {
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest",
		"--format", "pkgname", "--", "-race", "-parallel=4", "./...")
}

// TestIntegration runs all tests including integration tests.
func TestIntegration() error {
	return sh.RunV("go", "run", "gotest.tools/gotestsum@latest",
		"--format", "pkgname", "--", "-race", "-parallel=4", "-tags", "integration", "./...")
}

func Build() error {
	ldflags := buildLDFlags()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	if err := sh.RunV("go", "build", "-ldflags", ldflags, "./..."); err != nil {
		return err
	}
	out := filepath.Join(binDir, cliBinary)
	return sh.RunV("go", "build", "-ldflags", ldflags, "-o", out, "./cmd/"+cliBinary)
}

// Boundaries verifies that package import rules are not violated.
// Rules: no package may import daemon/, httpapi/, udsapi/, or cli/.
func Boundaries() error {
	forbidden := []struct {
		importer string
		imported string
	}{
		{"internal/config", "internal/daemon"},
		{"internal/acp", "internal/daemon"},
		{"internal/session", "internal/daemon"},
		{"internal/store", "internal/daemon"},
		{"internal/observe", "internal/daemon"},
		{"internal/config", "internal/httpapi"},
		{"internal/acp", "internal/httpapi"},
		{"internal/session", "internal/httpapi"},
		{"internal/store", "internal/httpapi"},
		{"internal/observe", "internal/httpapi"},
		{"internal/config", "internal/udsapi"},
		{"internal/acp", "internal/udsapi"},
		{"internal/session", "internal/udsapi"},
		{"internal/store", "internal/udsapi"},
		{"internal/observe", "internal/udsapi"},
		{"internal/config", "internal/cli"},
		{"internal/acp", "internal/cli"},
		{"internal/session", "internal/cli"},
		{"internal/store", "internal/cli"},
		{"internal/observe", "internal/cli"},
	}

	violations := 0
	for _, rule := range forbidden {
		importerDir := rule.importer
		if _, err := os.Stat(importerDir); os.IsNotExist(err) {
			continue
		}
		importPath := "github.com/pedronauck/agh/" + rule.imported
		cmd := exec.Command("grep", "-r", "--include=*.go", "-l", importPath, importerDir)
		out, err := cmd.Output()
		if err != nil {
			continue // grep returns exit 1 when no match — that's good
		}
		if len(strings.TrimSpace(string(out))) > 0 {
			fmt.Printf("VIOLATION: %s imports %s\n", rule.importer, rule.imported)
			for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				fmt.Printf("  %s\n", f)
			}
			violations++
		}
	}

	if violations > 0 {
		return fmt.Errorf("found %d boundary violations", violations)
	}
	fmt.Println("OK: all package boundaries respected")
	return nil
}

func Verify() {
	mg.SerialDeps(Fmt, Lint, Test, Build, Boundaries)
}

func goFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (name == "vendor" || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func buildLDFlags() string {
	version := gitOutput("describe", "--tags", "--always", "--dirty")
	if version == "" {
		version = "dev"
	}

	commit := gitOutput("rev-parse", "--short", "HEAD")
	if commit == "" {
		commit = "unknown"
	}

	buildDate := time.Now().UTC().Format(time.RFC3339)

	return strings.Join([]string{
		"-X " + versionPackage + ".Version=" + version,
		"-X " + versionPackage + ".Commit=" + commit,
		"-X " + versionPackage + ".BuildDate=" + buildDate,
	}, " ")
}

func gitOutput(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
