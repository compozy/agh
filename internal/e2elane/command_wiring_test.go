package e2elane

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func TestMakefileE2ETargetsDelegateToLaneSpecificMageTargets(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)

	tests := []struct {
		target      string
		wantSnippet string
	}{
		{target: "test-e2e-runtime", wantSnippet: "testE2ERuntime"},
		{target: "test-e2e-web", wantSnippet: "testE2EWeb"},
		{target: "test-e2e", wantSnippet: "testE2E"},
		{target: "test-e2e-nightly", wantSnippet: "testE2ENightly"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			t.Parallel()

			output := runCommand(t, repoRoot, "make", "-n", tt.target)
			if !strings.Contains(output, tt.wantSnippet) {
				t.Fatalf("make -n %s output = %q, want snippet %q", tt.target, output, tt.wantSnippet)
			}
			if strings.Contains(output, "testIntegration") {
				t.Fatalf("make -n %s output unexpectedly referenced testIntegration: %q", tt.target, output)
			}
		})
	}
}

func TestMakeHelpListsTheE2ELaneTargets(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)

	output := runCommand(t, repoRoot, "make", "help")
	for _, target := range []string{"testE2ERuntime", "testE2EWeb", "testE2E", "testE2ENightly"} {
		if !strings.Contains(output, target) {
			t.Fatalf("make help output = %q, want target %q", output, target)
		}
	}
}

func TestRootPackageScriptsExposeTheRepoLevelE2ELaneEntryPoints(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, "package.json"))

	want := map[string]string{
		"test:e2e:runtime": "make test-e2e-runtime",
		"test:e2e:web":     "make test-e2e-web",
		"test:e2e":         "make test-e2e",
		"test:e2e:nightly": "make test-e2e-nightly",
	}

	for script, command := range want {
		if got := pkg.Scripts[script]; got != command {
			t.Fatalf("package.json script %q = %q, want %q", script, got, command)
		}
	}
}

func TestRootPackageScriptsExposeSharedCodegenEntryPoints(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, "package.json"))

	want := map[string]string{
		"codegen":       "make codegen",
		"codegen-check": "make codegen-check",
	}

	for script, command := range want {
		if got := pkg.Scripts[script]; got != command {
			t.Fatalf("package.json script %q = %q, want %q", script, got, command)
		}
	}
}

func TestWebPackageScriptsPreserveDaemonServedModeAndNightlySplit(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

	if got := pkg.Scripts["test:e2e"]; got != "bun run test:e2e:daemon-served" {
		t.Fatalf("web package test:e2e = %q, want daemon-served wrapper", got)
	}
	if got := pkg.Scripts[DaemonServedWebScript]; got != "bun run codegen-check && bun run test:e2e:daemon-served:raw" {
		t.Fatalf("web package daemon-served script = %q", got)
	}
	if got := pkg.Scripts["test:e2e:daemon-served:raw"]; got != "playwright test --grep-invert @nightly" {
		t.Fatalf("web package daemon-served raw script = %q", got)
	}
	if got := pkg.Scripts[NightlyWebScript]; got != "bun run codegen-check && bun run test:e2e:nightly:raw" {
		t.Fatalf("web package nightly script = %q", got)
	}
	if got := pkg.Scripts["test:e2e:nightly:raw"]; got != "playwright test --grep @nightly --pass-with-no-tests" {
		t.Fatalf("web package nightly raw script = %q", got)
	}
}

func TestWebPackageScriptsRouteSharedCodegenIntoDependentCommands(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

	want := map[string]string{
		"codegen":       "bun run --cwd .. codegen",
		"codegen-check": "bun run --cwd .. codegen-check",
		"dev":           "bun run codegen && bun run dev:raw",
		"build":         "bun run codegen-check && bun run build:raw",
		"test":          "bun run codegen-check && bun run test:raw",
		"typecheck":     "bun run codegen-check && bun run typecheck:raw",
	}

	for script, command := range want {
		if got := pkg.Scripts[script]; got != command {
			t.Fatalf("web package script %q = %q, want %q", script, got, command)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readPackageJSON(t *testing.T, path string) packageJSON {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
	}
	return pkg
}

func runCommand(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output)
}
