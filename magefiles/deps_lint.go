//go:build mage

package main

import (
	"bytes"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/magefile/mage/sh"
)

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
	if err := runGolangCILint(); err != nil {
		return err
	}
	return Modernize()
}

func runGolangCILint() error {
	args := []string{
		"run",
		"--allow-parallel-runners",
		"--timeout",
		golangciLintTimeout,
		"./...",
	}
	if hasPinnedTool("golangci-lint", golangciLintVersion) {
		return sh.RunV("golangci-lint", args...)
	}
	goRunArgs := append(
		[]string{"run", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@" + golangciLintVersion},
		args...,
	)
	return sh.RunV("go", goRunArgs...)
}

func hasPinnedTool(name string, wantVersion string) bool {
	path, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	output, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return false
	}
	versionToken := "version " + strings.TrimPrefix(wantVersion, "v")
	return bytes.Contains(output, []byte(versionToken))
}

// Modernize runs gopls' modernize analyzer for min/max/slices idiom suggestions.
func Modernize() error {
	return sh.RunWithV(
		map[string]string{"CGO_ENABLED": "0"},
		"go",
		"run",
		"golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@"+goplsModernizeVersion,
		"./...",
	)
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
