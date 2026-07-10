//go:build mage

package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

func Build() error {
	return runMageSteps(buildSteps())
}

func buildSteps() []mageStep {
	return []mageStep{
		{name: "CodegenCheck", run: CodegenCheck},
		{name: "buildGo", run: buildGo},
	}
}

func buildGo() error {
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

// SourceInstallCheck verifies the public root go install path from source-visible files.

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
