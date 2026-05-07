//go:build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/pedronauck/agh/internal/codegen/openapits"
	"github.com/pedronauck/agh/internal/e2elane"
)

const (
	golangciLintVersion       = "v2.11.4"
	goplsModernizeVersion     = "v0.21.1"
	gotestsumVersion          = "v1.13.0"
	binDir                    = "bin"
	cliBinary                 = "agh"
	versionPackage            = "github.com/pedronauck/agh/internal/version"
	openAPISpecPath           = "openapi/agh.json"
	compozyOpenAPISpecPath    = "openapi/compozy-daemon.json"
	webOpenAPITypePath        = "web/src/generated/agh-openapi.d.ts"
	webCompozyOpenAPITypePath = "web/src/generated/compozy-openapi.d.ts"
	webDistIndex              = "web/dist/index.html"
	daemonBinaryEnvVar        = "AGH_TEST_DAEMON_BIN"
	driverBinaryEnvVar        = "AGH_TEST_ACPMOCK_DRIVER_BIN"
)

var (
	Default             = Verify
	webOpenAPIArtifacts = []openapits.Artifact{
		{
			SpecPath:   openAPISpecPath,
			OutputPath: webOpenAPITypePath,
		},
		{
			SpecPath:   compozyOpenAPISpecPath,
			OutputPath: webCompozyOpenAPITypePath,
		},
	}
)

var (
	errLaneBinaryOverrideDirectory     = errors.New("lane binary override points to directory")
	errLaneBinaryOverrideNotExecutable = errors.New("lane binary override is not executable")
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
	if err := ensureWebBundle(); err != nil {
		return err
	}
	if err := sh.RunV(
		"go",
		"run",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"+golangciLintVersion,
		"run",
		"--fix",
		"--allow-parallel-runners",
		"./...",
	); err != nil {
		return err
	}
	return Modernize()
}

// Modernize runs gopls' modernize analyzer to apply min/max/slices idiom suggestions.
func Modernize() error {
	return sh.RunV(
		"go",
		"run",
		"golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@"+goplsModernizeVersion,
		"-fix",
		"./...",
	)
}

// Test runs unit tests only (no integration tag).
func Test() error {
	if err := ensureWebBundle(); err != nil {
		return err
	}
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-parallel=4", "./...")
}

// TestIntegration runs all tests including integration tests.
func TestIntegration() error {
	if err := ensureWebBundle(); err != nil {
		return err
	}
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-parallel=4", "-tags", "integration", "./...")
}

// TestE2ERuntime runs the PR-required daemon/runtime E2E lane without sweeping every integration package.
func TestE2ERuntime() error {
	return runE2ELane(e2elane.LaneRuntime)
}

// TestE2EWeb runs the daemon-served Playwright E2E lane for shipped browser workflows.
func TestE2EWeb() error {
	return runE2ELane(e2elane.LaneWeb)
}

// TestE2E runs the default PR-required runtime and browser E2E lanes.
func TestE2E() error {
	return runE2ELane(e2elane.LaneCombined)
}

// TestE2ENightly runs the combined E2E lane plus credentialed nightly coverage.
func TestE2ENightly() error {
	return runE2ELane(e2elane.LaneNightly)
}

func Build() error {
	if err := CodegenCheck(); err != nil {
		return err
	}
	if err := WebBuild(); err != nil {
		return err
	}
	return buildGo()
}

func Codegen() error {
	if err := runCommandInDir(context.Background(), ".", "go", "run", "./cmd/agh-codegen", "all"); err != nil {
		return err
	}
	artifacts, err := availableWebOpenAPIArtifacts()
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if err := openapits.Generate(context.Background(), artifact); err != nil {
			return err
		}
	}
	return nil
}

func CodegenCheck() error {
	if err := runCommandInDir(context.Background(), ".", "go", "run", "./cmd/agh-codegen", "check"); err != nil {
		return err
	}
	artifacts, err := availableWebOpenAPIArtifacts()
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if err := openapits.Check(context.Background(), artifact); err != nil {
			return err
		}
	}
	return nil
}

// BunLint runs the monorepo-wide lint script (oxfmt + oxlint over every workspace).
func BunLint() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "lint")
}

// BunTypecheck runs the monorepo-wide typecheck pipeline (turbo run typecheck across every workspace).
func BunTypecheck() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "typecheck")
}

// BunTest runs the monorepo-wide vitest projects suite from the repo root.
func BunTest() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "tests")
}

func InstallerCheck() error {
	installer := filepath.Join("packages", "site", "public", "install.sh")
	if err := sh.RunV("sh", "-n", installer); err != nil {
		return err
	}
	return sh.RunV("sh", installer, "--dry-run", "--skip-bootstrap")
}

func WebLint() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "lint")
}

func WebTypecheck() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "typecheck:raw")
}

func WebTest() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "test:raw")
}

func WebBuild() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "build:raw")
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

// Boundaries verifies that package import rules are not violated.
// Rules: no package may import daemon/, api/httpapi/, api/udsapi/, or cli/.
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
		{"internal/config", "internal/api/httpapi"},
		{"internal/acp", "internal/api/httpapi"},
		{"internal/session", "internal/api/httpapi"},
		{"internal/store", "internal/api/httpapi"},
		{"internal/observe", "internal/api/httpapi"},
		{"internal/config", "internal/api/udsapi"},
		{"internal/acp", "internal/api/udsapi"},
		{"internal/session", "internal/api/udsapi"},
		{"internal/store", "internal/api/udsapi"},
		{"internal/observe", "internal/api/udsapi"},
		{"internal/config", "internal/cli"},
		{"internal/acp", "internal/cli"},
		{"internal/session", "internal/cli"},
		{"internal/store", "internal/cli"},
		{"internal/observe", "internal/cli"},
		{"internal/api/contract", "internal/daemon"},
		{"internal/api/contract", "internal/api/httpapi"},
		{"internal/api/contract", "internal/api/udsapi"},
		{"internal/api/contract", "internal/cli"},
		{"internal/api/core", "internal/daemon"},
		{"internal/api/core", "internal/api/httpapi"},
		{"internal/api/core", "internal/api/udsapi"},
		{"internal/api/core", "internal/cli"},
		{"internal/api/httpapi", "internal/daemon"},
		{"internal/api/httpapi", "internal/api/udsapi"},
		{"internal/api/httpapi", "internal/cli"},
		{"internal/api/udsapi", "internal/daemon"},
		{"internal/api/udsapi", "internal/api/httpapi"},
		{"internal/api/udsapi", "internal/cli"},
		{"internal/modelcatalog", "internal/daemon"},
		{"internal/modelcatalog", "internal/api/contract"},
		{"internal/modelcatalog", "internal/api/core"},
		{"internal/modelcatalog", "internal/api/httpapi"},
		{"internal/modelcatalog", "internal/api/udsapi"},
		{"internal/modelcatalog", "internal/cli"},
		{"internal/memory/contract", "internal/memory/controller"},
		{"internal/memory/contract", "internal/memory/recall"},
		{"internal/memory/contract", "internal/memory/extractor"},
		{"internal/memory/contract", "internal/memory/provider/local"},
		{"internal/memory/contract", "internal/store/workspacedb"},
		{"internal/memory/controller", "internal/daemon"},
		{"internal/memory/controller", "internal/api/httpapi"},
		{"internal/memory/controller", "internal/api/udsapi"},
		{"internal/memory/controller", "internal/cli"},
		{"internal/memory/recall", "internal/daemon"},
		{"internal/memory/recall", "internal/api/httpapi"},
		{"internal/memory/recall", "internal/api/udsapi"},
		{"internal/memory/recall", "internal/cli"},
		{"internal/memory/extractor", "internal/daemon"},
		{"internal/memory/extractor", "internal/api/httpapi"},
		{"internal/memory/extractor", "internal/api/udsapi"},
		{"internal/memory/extractor", "internal/cli"},
		{"internal/memory/provider/local", "internal/daemon"},
		{"internal/memory/provider/local", "internal/api/httpapi"},
		{"internal/memory/provider/local", "internal/api/udsapi"},
		{"internal/memory/provider/local", "internal/cli"},
		{"internal/sessions/ledger", "internal/daemon"},
		{"internal/sessions/ledger", "internal/api/httpapi"},
		{"internal/sessions/ledger", "internal/api/udsapi"},
		{"internal/sessions/ledger", "internal/cli"},
		{"internal/store/workspacedb", "internal/daemon"},
		{"internal/store/workspacedb", "internal/api/httpapi"},
		{"internal/store/workspacedb", "internal/api/udsapi"},
		{"internal/store/workspacedb", "internal/cli"},
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

func Verify() error {
	steps := []func() error{
		CodegenCheck,
		InstallerCheck,
		BunLint,
		BunTypecheck,
		BunTest,
		WebBuild,
		Fmt,
		Lint,
		Test,
		buildGo,
		Boundaries,
	}

	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}

	return nil
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

func ensureWebBundle() error {
	if _, err := os.Stat(webDistIndex); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := CodegenCheck(); err != nil {
		return err
	}
	return WebBuild()
}

func runE2ELane(lane e2elane.Lane) (runErr error) {
	ctx := context.Background()

	plan, err := e2elane.PlanForLane(lane)
	if err != nil {
		return err
	}

	if shouldEnsureWebBundle(plan) {
		if err := ensureWebBundle(); err != nil {
			return err
		}
	}

	laneEnv, err := prepareE2ELaneEnv()
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := laneEnv.Cleanup(); cleanupErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("cleanup e2e lane environment: %w", cleanupErr))
		}
	}()

	for _, suite := range plan.GoSuites {
		if err := runIntegrationSuite(ctx, suite, laneEnv.Values); err != nil {
			return err
		}
	}

	for _, suite := range plan.ScriptSuites {
		if err := runCommandInDirWithEnv(ctx, suite.Dir, laneEnv.Values, "bun", "run", suite.Script); err != nil {
			return err
		}
	}

	return nil
}

func shouldEnsureWebBundle(plan e2elane.Plan) bool {
	return len(plan.GoSuites) > 0 || plan.RequiresDaemonServedBrowser
}

type e2eLaneEnv struct {
	Values  map[string]string
	cleanup func() error
}

func (env e2eLaneEnv) Cleanup() error {
	if env.cleanup == nil {
		return nil
	}
	return env.cleanup()
}

func prepareE2ELaneEnv() (e2eLaneEnv, error) {
	var cleanups []func() error
	daemonPath, cleanup, err := resolveOrBuildLaneBinary(daemonBinaryEnvVar, func(outputPath string) error {
		return runCommandInDir(
			context.Background(),
			".",
			"go",
			"build",
			"-ldflags",
			buildLDFlags(),
			"-o",
			outputPath,
			"./cmd/agh",
		)
	}, cliBinary)
	if err != nil {
		return e2eLaneEnv{}, err
	}
	cleanups = append(cleanups, cleanup)

	driverPath, cleanup, err := resolveOrBuildLaneBinary(driverBinaryEnvVar, func(outputPath string) error {
		return runCommandInDir(
			context.Background(),
			".",
			"go",
			"build",
			"-o",
			outputPath,
			"./internal/testutil/acpmock/cmd/acpmock-driver",
		)
	}, "acpmock-driver")
	if err != nil {
		return e2eLaneEnv{}, errors.Join(err, runCleanups(cleanups))
	}
	cleanups = append(cleanups, cleanup)

	return e2eLaneEnv{
		Values: map[string]string{
			daemonBinaryEnvVar: daemonPath,
			driverBinaryEnvVar: driverPath,
		},
		cleanup: func() error {
			return runCleanups(cleanups)
		},
	}, nil
}

func resolveOrBuildLaneBinary(
	envVar string,
	build func(string) error,
	name string,
) (string, func() error, error) {
	if override := strings.TrimSpace(os.Getenv(envVar)); override != "" {
		overridePath, err := resolveLaneBinaryOverride(envVar, override)
		if err != nil {
			return "", nil, err
		}
		return overridePath, noopCleanup, nil
	}

	buildDir, err := os.MkdirTemp("", "agh-e2e-lane-")
	if err != nil {
		return "", nil, err
	}
	outputPath := filepath.Join(buildDir, laneBinaryName(name))
	if err := build(outputPath); err != nil {
		return "", nil, errors.Join(
			fmt.Errorf("build %s e2e lane binary: %w", name, err),
			cleanupLaneBuildDir(buildDir),
		)
	}
	return outputPath, func() error {
		return cleanupLaneBuildDir(buildDir)
	}, nil
}

func resolveLaneBinaryOverride(envVar, override string) (string, error) {
	overridePath := override
	if !filepath.IsAbs(overridePath) {
		absPath, err := filepath.Abs(overridePath)
		if err != nil {
			return "", fmt.Errorf("resolve %s override %q: %w", envVar, override, err)
		}
		overridePath = absPath
	}

	info, err := os.Stat(overridePath)
	if err != nil {
		return "", fmt.Errorf("%s points to %q: %w", envVar, overridePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s points to directory %q: %w", envVar, overridePath, errLaneBinaryOverrideDirectory)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf(
			"%s points to non-executable file %q: %w",
			envVar,
			overridePath,
			errLaneBinaryOverrideNotExecutable,
		)
	}
	return overridePath, nil
}

func noopCleanup() error {
	return nil
}

func runCleanups(cleanups []func() error) error {
	var joined error
	for idx := len(cleanups) - 1; idx >= 0; idx-- {
		if err := cleanups[idx](); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func cleanupLaneBuildDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove e2e lane build dir %q: %w", path, err)
	}
	return nil
}

func laneBinaryName(name string) string {
	if strings.EqualFold(filepath.Ext(name), ".exe") {
		return name
	}
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func runIntegrationSuite(ctx context.Context, suite e2elane.GoSuite, env map[string]string) error {
	args := []string{
		"run",
		"gotest.tools/gotestsum@" + gotestsumVersion,
		"--format",
		"pkgname",
		"--",
		"-race",
		"-parallel=4",
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

func runCommandInDir(ctx context.Context, dir string, name string, args ...string) error {
	return runCommandInDirWithEnv(ctx, dir, nil, name, args...)
}

func runCommandInDirWithEnv(ctx context.Context, dir string, env map[string]string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergeCommandEnv(env)
	return cmd.Run()
}

func mergeCommandEnv(overrides map[string]string) []string {
	env := append([]string(nil), os.Environ()...)
	if len(overrides) == 0 {
		return env
	}
	for key, value := range overrides {
		prefix := key + "="
		replaced := false
		for idx, current := range env {
			if strings.HasPrefix(current, prefix) {
				env[idx] = prefix + value
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, prefix+value)
		}
	}
	return env
}

func runRaceEnabledGoCommand(ctx context.Context, env map[string]string, args ...string) error {
	if err := runCommandInDirWithEnv(ctx, ".", withRaceEnabledEnv(env), "go", args...); err != nil {
		return fmt.Errorf("race-enabled go command %v: %w", args, err)
	}
	return nil
}

func withRaceEnabledEnv(overrides map[string]string) map[string]string {
	env := make(map[string]string, len(overrides)+1)
	for key, value := range overrides {
		env[key] = value
	}
	env["CGO_ENABLED"] = "1"
	return env
}

func availableWebOpenAPIArtifacts() ([]openapits.Artifact, error) {
	artifacts := make([]openapits.Artifact, 0, len(webOpenAPIArtifacts))
	for _, artifact := range webOpenAPIArtifacts {
		if artifact.SpecPath == "" {
			continue
		}
		if _, err := os.Stat(artifact.SpecPath); err != nil {
			if os.IsNotExist(err) {
				if artifact.OutputPath != "" {
					if _, outputErr := os.Stat(artifact.OutputPath); outputErr == nil {
						return nil, fmt.Errorf(
							"%s exists but %s is missing; remove the generated file or restore the spec",
							artifact.OutputPath,
							artifact.SpecPath,
						)
					}
				}
				continue
			}
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}
