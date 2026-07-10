//go:build mage

package main

import (
	"context"
	"strings"

	"github.com/compozy/agh/internal/e2elane"
)

// Test runs unit tests only (no integration tag).
func Test() error {
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-p", goUnitTestPackageLimit(), "-parallel=4",
		"-timeout", goUnitTestTimeout, "./...")
}

// TestIntegration runs all tests including integration tests.
func TestIntegration() error {
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-p", goIntegrationPackageLimit, "-parallel=4",
		"-timeout", goIntegrationTestTimeout, "-tags", "integration", "./...")
}

func runIntegrationSuite(ctx context.Context, suite e2elane.GoSuite, env map[string]string) error {
	args := []string{
		"run",
		"gotest.tools/gotestsum@" + gotestsumVersion,
		"--format",
		"pkgname",
		"--",
		"-race",
		"-p",
		goIntegrationPackageLimit,
		"-parallel=4",
		"-timeout",
		goIntegrationTestTimeout,
		"-count=1",
		"-tags",
		"integration",
	}
	if strings.TrimSpace(suite.Run) != "" {
		args = append(args, "-run", suite.Run)
	}
	args = append(args, suite.Packages...)
	return runRaceEnabledGoCommand(ctx, env, args...)
}
