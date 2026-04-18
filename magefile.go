//go:build mage

package main

import (
	"bytes"
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
	"github.com/pedronauck/agh/internal/e2elane"
)

const (
	golangciLintVersion = "v2.11.4"
	binDir              = "bin"
	cliBinary           = "agh"
	versionPackage      = "github.com/pedronauck/agh/internal/version"
	openAPISpecPath     = "openapi/agh.json"
	webOpenAPITypePath  = "web/src/generated/agh-openapi.d.ts"
	webDistIndex        = "web/dist/index.html"
	daemonBinaryEnvVar  = "AGH_TEST_DAEMON_BIN"
	driverBinaryEnvVar  = "AGH_TEST_ACPMOCK_DRIVER_BIN"
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
		"golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest",
		"-fix",
		"./...",
	)
}

// Test runs unit tests only (no integration tag).
func Test() error {
	if err := ensureWebBundle(); err != nil {
		return err
	}
	return runRaceEnabledGoCommand(nil,
		"run", "gotest.tools/gotestsum@latest",
		"--format", "pkgname", "--", "-race", "-parallel=4", "./...")
}

// TestIntegration runs all tests including integration tests.
func TestIntegration() error {
	if err := ensureWebBundle(); err != nil {
		return err
	}
	return runRaceEnabledGoCommand(nil,
		"run", "gotest.tools/gotestsum@latest",
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
	if err := runCommandInDir(".", "go", "run", "./cmd/agh-codegen", "all"); err != nil {
		return err
	}
	return generateWebOpenAPITypes(webOpenAPITypePath)
}

func CodegenCheck() error {
	if err := runCommandInDir(".", "go", "run", "./cmd/agh-codegen", "check"); err != nil {
		return err
	}
	return checkWebOpenAPITypes(webOpenAPITypePath)
}

func WebLint() error {
	return runCommandInDir("web", "bun", "run", "lint")
}

func WebTypecheck() error {
	return runCommandInDir("web", "bun", "run", "typecheck:raw")
}

func WebTest() error {
	return runCommandInDir("web", "bun", "run", "test:raw")
}

func WebBuild() error {
	return runCommandInDir("web", "bun", "run", "build:raw")
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
		WebLint,
		WebTypecheck,
		WebTest,
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

func generateWebOpenAPITypes(outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	if err := runCommandInDir(".", "bunx", "openapi-typescript", openAPISpecPath, "-o", outputPath); err != nil {
		return err
	}
	return runCommandInDir(".", "bunx", "oxfmt", outputPath)
}

func checkWebOpenAPITypes(path string) error {
	file, err := os.CreateTemp("", "agh-openapi-types-*.d.ts")
	if err != nil {
		return err
	}
	_ = file.Close()
	defer os.Remove(file.Name())

	if err := generateWebOpenAPITypes(file.Name()); err != nil {
		return err
	}

	want, err := os.ReadFile(file.Name())
	if err != nil {
		return err
	}

	return checkGeneratedFile(path, want)
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

func runE2ELane(lane e2elane.Lane) error {
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

	for _, suite := range plan.GoSuites {
		if err := runIntegrationSuite(suite, laneEnv); err != nil {
			return err
		}
	}

	for _, suite := range plan.ScriptSuites {
		if err := runCommandInDirWithEnv(suite.Dir, laneEnv, "bun", "run", suite.Script); err != nil {
			return err
		}
	}

	return nil
}

func shouldEnsureWebBundle(plan e2elane.Plan) bool {
	return len(plan.GoSuites) > 0 || plan.RequiresDaemonServedBrowser
}

func prepareE2ELaneEnv() (map[string]string, error) {
	daemonPath, err := resolveOrBuildLaneBinary(daemonBinaryEnvVar, func(outputPath string) error {
		return runCommandInDir(
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
		return nil, err
	}

	driverPath, err := resolveOrBuildLaneBinary(driverBinaryEnvVar, func(outputPath string) error {
		return runCommandInDir(
			".",
			"go",
			"build",
			"-o",
			outputPath,
			"./internal/testutil/acpmock/cmd/acpmock-driver",
		)
	}, "acpmock-driver")
	if err != nil {
		return nil, err
	}

	return map[string]string{
		daemonBinaryEnvVar: daemonPath,
		driverBinaryEnvVar: driverPath,
	}, nil
}

func resolveOrBuildLaneBinary(
	envVar string,
	build func(string) error,
	name string,
) (string, error) {
	if override := strings.TrimSpace(os.Getenv(envVar)); override != "" {
		if filepath.IsAbs(override) {
			return override, nil
		}
		absPath, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	buildDir, err := os.MkdirTemp("", "agh-e2e-lane-")
	if err != nil {
		return "", err
	}
	outputPath := filepath.Join(buildDir, laneBinaryName(name))
	if err := build(outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
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

func runIntegrationSuite(suite e2elane.GoSuite, env map[string]string) error {
	args := []string{
		"run",
		"gotest.tools/gotestsum@latest",
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
	return runRaceEnabledGoCommand(env, args...)
}

func runCommandInDir(dir string, name string, args ...string) error {
	return runCommandInDirWithEnv(dir, nil, name, args...)
}

func runCommandInDirWithEnv(dir string, env map[string]string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
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

func runRaceEnabledGoCommand(env map[string]string, args ...string) error {
	return runCommandInDirWithEnv(".", withRaceEnabledEnv(env), "go", args...)
}

func withRaceEnabledEnv(overrides map[string]string) map[string]string {
	env := make(map[string]string, len(overrides)+1)
	for key, value := range overrides {
		env[key] = value
	}
	env["CGO_ENABLED"] = "1"
	return env
}

func checkGeneratedFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing; run codegen", path)
		}
		return err
	}

	if !bytes.Equal(got, want) {
		return fmt.Errorf("%s is stale; run codegen", path)
	}

	return nil
}
