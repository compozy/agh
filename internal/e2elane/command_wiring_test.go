package e2elane

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

type turboJSON struct {
	Tasks map[string]turboTask `json:"tasks"`
}

type turboTask struct {
	DependsOn []string `json:"dependsOn"`
}

func TestMakefileE2ETargetsDelegateToLaneSpecificMageTargets(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	recipes := makeTargetRecipes(t, filepath.Join(repoRoot, "Makefile"))

	tests := []struct {
		name        string
		target      string
		wantSnippet string
	}{
		{
			name:        "Should delegate the runtime make target to the runtime mage lane",
			target:      "test-e2e-runtime",
			wantSnippet: "testE2ERuntime",
		},
		{
			name:        "Should delegate the web make target to the web mage lane",
			target:      "test-e2e-web",
			wantSnippet: "testE2EWeb",
		},
		{
			name:        "Should delegate the combined make target to the combined mage lane",
			target:      "test-e2e",
			wantSnippet: "testE2E",
		},
		{
			name:        "Should delegate the nightly make target to the nightly mage lane",
			target:      "test-e2e-nightly",
			wantSnippet: "testE2ENightly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recipe := recipes[tt.target]
			if !strings.Contains(recipe, tt.wantSnippet) {
				t.Fatalf("Makefile target %s recipe = %q, want snippet %q", tt.target, recipe, tt.wantSnippet)
			}
			if strings.Contains(recipe, "testIntegration") {
				t.Fatalf("Makefile target %s recipe unexpectedly referenced testIntegration: %q", tt.target, recipe)
			}
		})
	}
}

func TestMagefileExportsTheE2ELaneTargets(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	targets := mageTargetNames(t, filepath.Join(repoRoot, "magefile.go"))

	for _, target := range []string{"testE2ERuntime", "testE2EWeb", "testE2E", "testE2ENightly"} {
		t.Run("Should export "+target+" as a mage target", func(t *testing.T) {
			t.Parallel()

			if !targets[target] {
				t.Fatalf("mage targets = %#v, want %q", targets, target)
			}
		})
	}
}

func TestE2ELaneCommandsAreExecutableSmoke(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "Should dry run runtime make target",
			args: []string{"make", "-n", "test-e2e-runtime"},
			want: "testE2ERuntime",
		},
		{name: "Should dry run web make target", args: []string{"make", "-n", "test-e2e-web"}, want: "testE2EWeb"},
		{name: "Should dry run combined make target", args: []string{"make", "-n", "test-e2e"}, want: "testE2E"},
		{
			name: "Should dry run nightly make target",
			args: []string{"make", "-n", "test-e2e-nightly"},
			want: "testE2ENightly",
		},
		{name: "Should list make help through mage", args: []string{"make", "help"}, want: "testE2ERuntime"},
		{
			name: "Should list mage targets directly",
			args: []string{"go", "run", "github.com/magefile/mage@v1.15.0", "-l"},
			want: "testE2ERuntime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := runRepoCommand(t, repoRoot, tt.args...)
			if !strings.Contains(output, tt.want) {
				t.Fatalf("%s output = %q, want snippet %q", strings.Join(tt.args, " "), output, tt.want)
			}
		})
	}
}

func TestRootPackageScriptsExposeTheRepoLevelE2ELaneEntryPoints(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, "package.json"))

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{
			name:    "Should expose the runtime e2e lane entry point",
			script:  "test:e2e:runtime",
			command: "make test-e2e-runtime",
		},
		{
			name:    "Should expose the web e2e lane entry point",
			script:  "test:e2e:web",
			command: "make test-e2e-web",
		},
		{
			name:    "Should expose the combined e2e lane entry point",
			script:  "test:e2e",
			command: "make test-e2e",
		},
		{
			name:    "Should expose the nightly e2e lane entry point",
			script:  "test:e2e:nightly",
			command: "make test-e2e-nightly",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pkg.Scripts[tc.script]; got != tc.command {
				t.Fatalf("package.json script %q = %q, want %q", tc.script, got, tc.command)
			}
		})
	}
}

func TestRootPackageScriptsExposeSharedCodegenEntryPoints(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, "package.json"))

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{
			name:    "Should expose codegen as the repo level codegen entry point",
			script:  "codegen",
			command: "make codegen",
		},
		{
			name:    "Should expose codegen-check as the repo level codegen check entry point",
			script:  "codegen-check",
			command: "make codegen-check",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pkg.Scripts[tc.script]; got != tc.command {
				t.Fatalf("package.json script %q = %q, want %q", tc.script, got, tc.command)
			}
		})
	}
}

func TestTurboPipelineRunsSharedCodegenCheckBeforeWorkspaceGates(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	cfg := readTurboJSON(t, filepath.Join(repoRoot, "turbo.json"))
	const codegenCheckTask = "//#codegen-check"

	if _, ok := cfg.Tasks[codegenCheckTask]; !ok {
		t.Fatalf("turbo tasks = %#v, want root task %q", cfg.Tasks, codegenCheckTask)
	}

	tests := []struct {
		name string
		task string
	}{
		{
			name: "Should run shared codegen check before workspace builds",
			task: "build",
		},
		{
			name: "Should run shared codegen check before workspace typechecks",
			task: "typecheck",
		},
		{
			name: "Should run shared codegen check before workspace tests",
			task: "test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			task, ok := cfg.Tasks[tc.task]
			if !ok {
				t.Fatalf("turbo task %q missing from %#v", tc.task, cfg.Tasks)
			}
			if !containsString(task.DependsOn, codegenCheckTask) {
				t.Fatalf(
					"turbo task %q dependsOn = %#v, want %q",
					tc.task,
					task.DependsOn,
					codegenCheckTask,
				)
			}
		})
	}
}

func TestTurboPipelineKeepsVercelSiteBuildDeployableWithoutGoToolchain(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	cfg := readTurboJSON(t, filepath.Join(repoRoot, "turbo.json"))
	const codegenCheckTask = "//#codegen-check"

	for _, taskName := range []string{"@agh/site#build", "@agh/ui#build"} {
		task, ok := cfg.Tasks[taskName]
		if !ok {
			t.Fatalf("turbo task %q missing from %#v", taskName, cfg.Tasks)
		}
		if containsString(task.DependsOn, codegenCheckTask) {
			t.Fatalf(
				"turbo task %q dependsOn = %#v, want no %q because Vercel site deploys do not install Go",
				taskName,
				task.DependsOn,
				codegenCheckTask,
			)
		}
	}

	siteBuild, ok := cfg.Tasks["@agh/site#build"]
	if !ok {
		t.Fatalf("turbo task %q missing from %#v", "@agh/site#build", cfg.Tasks)
	}
	if !containsString(siteBuild.DependsOn, "^build") {
		t.Fatalf(
			"turbo task %q dependsOn = %#v, want dependency build propagation",
			"@agh/site#build",
			siteBuild.DependsOn,
		)
	}
}

func TestWebPackageScriptsPreserveDaemonServedModeAndNightlySplit(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{
			name:    "Should route web e2e through the daemon-served wrapper",
			script:  "test:e2e",
			command: "bun run test:e2e:daemon-served",
		},
		{
			name:    "Should run codegen-check before daemon-served browser tests",
			script:  DaemonServedWebScript,
			command: "bun run codegen-check && bun run test:e2e:daemon-served:raw",
		},
		{
			name:    "Should keep nightly specs out of daemon-served browser tests",
			script:  "test:e2e:daemon-served:raw",
			command: "playwright test --grep-invert @nightly",
		},
		{
			name:    "Should run codegen-check before nightly browser tests",
			script:  NightlyWebScript,
			command: "bun run codegen-check && bun run test:e2e:nightly:raw",
		},
		{
			name:    "Should gate nightly browser coverage before Playwright runs",
			script:  "test:e2e:nightly:check",
			command: "bun run e2e/scripts/check-nightly-coverage.ts",
		},
		{
			name:    "Should fail nightly browser lane when expected tests are absent",
			script:  "test:e2e:nightly:raw",
			command: "bun run test:e2e:nightly:check && playwright test --grep @nightly",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pkg.Scripts[tc.script]; got != tc.command {
				t.Fatalf("web package script %q = %q, want %q", tc.script, got, tc.command)
			}
		})
	}
}

func TestWebPackageScriptsRouteSharedCodegenThroughTurboOwnedGates(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, WebDir, "package.json"))

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{
			name:    "Should route codegen through the shared repo entry point",
			script:  "codegen",
			command: "bun run --cwd .. codegen",
		},
		{
			name:    "Should route codegen-check through the shared repo entry point",
			script:  "codegen-check",
			command: "bun run --cwd .. codegen-check",
		},
		{
			name:    "Should run codegen before dev",
			script:  "dev",
			command: "bun run codegen && bun run dev:raw",
		},
		{
			name:    "Should leave build raw for the Turbo-owned shared codegen check",
			script:  "build",
			command: "bun run build:raw",
		},
		{
			name:    "Should leave test raw for the Turbo-owned shared codegen check",
			script:  "test",
			command: "bun run test:raw",
		},
		{
			name:    "Should leave typecheck raw for the Turbo-owned shared codegen check",
			script:  "typecheck",
			command: "bun run typecheck:raw",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pkg.Scripts[tc.script]; got != tc.command {
				t.Fatalf("web package script %q = %q, want %q", tc.script, got, tc.command)
			}
		})
	}
}

func TestExtensionSDKPackageScriptsRouteSharedCodegenThroughTurboOwnedGates(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	pkg := readPackageJSON(t, filepath.Join(repoRoot, "sdk", "typescript", "package.json"))
	const extensionSDKBuildScript = "rm -rf dist && " +
		"tsc -p tsconfig.types.json && " +
		"tsc -p tsconfig.esm.json && " +
		"tsc -p tsconfig.cjs.json && " +
		"node ./scripts/postbuild.mjs"

	tests := []struct {
		name    string
		script  string
		command string
	}{
		{
			name:    "Should route codegen-check through the shared repo entry point",
			script:  "codegen-check",
			command: "bun run --cwd ../.. codegen-check",
		},
		{
			name:    "Should leave build raw for the Turbo-owned shared codegen check",
			script:  "build",
			command: extensionSDKBuildScript,
		},
		{
			name:    "Should leave typecheck raw for the Turbo-owned shared codegen check",
			script:  "typecheck",
			command: "tsc -p tsconfig.json --noEmit",
		},
		{
			name:    "Should leave test raw for the Turbo-owned shared codegen check",
			script:  "test",
			command: "vitest run --config vitest.config.ts",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := pkg.Scripts[tc.script]; got != tc.command {
				t.Fatalf("extension SDK package script %q = %q, want %q", tc.script, got, tc.command)
			}
		})
	}
}

func mageTargetNames(t *testing.T, path string) map[string]bool {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v", path, err)
	}

	targets := make(map[string]bool)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !isMageTargetFunction(fn) {
			continue
		}
		targets[mageTargetName(fn.Name.Name)] = true
	}
	return targets
}

func runRepoCommand(t *testing.T, repoRoot string, args ...string) string {
	t.Helper()
	if len(args) == 0 {
		t.Fatal("runRepoCommand requires a command")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %v\n%s", strings.Join(args, " "), ctx.Err(), string(output))
	}
	if err != nil {
		t.Fatalf("%s error = %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func isMageTargetFunction(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || !fn.Name.IsExported() || fn.Type.Params.NumFields() != 0 {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	result, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && result.Name == "error"
}

func mageTargetName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func makeTargetRecipes(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	recipes := make(map[string]string)
	var currentTarget string
	var currentRecipe strings.Builder
	flush := func() {
		if currentTarget == "" {
			return
		}
		recipes[currentTarget] = strings.TrimSpace(currentRecipe.String())
		currentTarget = ""
		currentRecipe.Reset()
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, "\t") {
			if currentTarget != "" {
				currentRecipe.WriteString(strings.TrimSpace(line))
				currentRecipe.WriteByte('\n')
			}
			continue
		}

		flush()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon <= 0 {
			continue
		}
		target := trimmed[:colon]
		if strings.ContainsAny(target, " \t=") {
			continue
		}
		currentTarget = target
	}
	flush()
	return recipes
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

func readTurboJSON(t *testing.T, path string) turboJSON {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	var cfg turboJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
	}
	return cfg
}

func containsString(values []string, want string) bool {
	return slices.Contains(values, want)
}
