package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGoReleaserConfigPreservesTrustArtifactsAndPackageTargets(t *testing.T) {
	t.Parallel()

	root := findRepoRootForReleaseConfigTest(t)
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yml"))
	if err != nil {
		t.Fatalf("os.ReadFile(.goreleaser.yml) error = %v", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal(.goreleaser.yml) error = %v", err)
	}

	t.Run("Should preserve checksum signing configuration", func(t *testing.T) {
		checksum := mapAt(t, cfg, "checksum")
		if got, want := stringAt(t, checksum, "name_template"), "checksums.txt"; got != want {
			t.Fatalf("checksum.name_template = %q, want %q", got, want)
		}
		if got, want := stringAt(t, checksum, "algorithm"), "sha256"; got != want {
			t.Fatalf("checksum.algorithm = %q, want %q", got, want)
		}

		signs := sliceAt(t, cfg, "signs")
		if len(signs) == 0 {
			t.Fatal("signs is empty, want checksum signing preserved")
		}
		firstSign := asMap(t, signs[0], "signs[0]")
		if got, want := stringAt(t, firstSign, "cmd"), "cosign"; got != want {
			t.Fatalf("signs[0].cmd = %q, want %q", got, want)
		}
		if got, want := stringAt(t, firstSign, "artifacts"), "checksum"; got != want {
			t.Fatalf("signs[0].artifacts = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, firstSign, "args"), "sign-blob") {
			t.Fatalf("signs[0].args = %#v, want sign-blob", firstSign["args"])
		}
		if got, want := stringAt(t, firstSign, "signature"), "${artifact}.sigstore.json"; got != want {
			t.Fatalf("signs[0].signature = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, firstSign, "args"), "--bundle=${signature}") {
			t.Fatalf("signs[0].args = %#v, want --bundle=${signature}", firstSign["args"])
		}
	})

	t.Run("Should preserve SBOM artifact coverage", func(t *testing.T) {
		sboms := sliceAt(t, cfg, "sboms")
		assertSBOMArtifact(t, sboms, "archive")
		assertSBOMArtifact(t, sboms, "package")
		assertSBOMArtifact(t, sboms, "source")
	})

	t.Run("Should publish stable archives and curl installer asset", func(t *testing.T) {
		t.Parallel()

		archives := sliceAt(t, cfg, "archives")
		if len(archives) != 1 {
			t.Fatalf("archives len = %d, want 1", len(archives))
		}
		archive := asMap(t, archives[0], "archives[0]")
		if got, want := stringAt(t, archive, "id"), "agh-archive"; got != want {
			t.Fatalf("archives[0].id = %q, want %q", got, want)
		}
		nameTemplate := stringAt(t, archive, "name_template")
		for _, want := range []string{
			"{{ .ProjectName }}_{{ .Os }}_",
			`{{- if eq .Arch "amd64" }}x86_64`,
			`{{- else }}{{ .Arch }}{{ end }}`,
		} {
			if !strings.Contains(nameTemplate, want) {
				t.Fatalf("archives[0].name_template = %q, want to contain %q", nameTemplate, want)
			}
		}
		if strings.Contains(nameTemplate, "{{ .Version }}") {
			t.Fatalf("archives[0].name_template = %q, want stable name without version", nameTemplate)
		}

		release := mapAt(t, cfg, "release")
		github := mapAt(t, release, "github")
		if got, want := stringAt(t, github, "owner"), "compozy"; got != want {
			t.Fatalf("release.github.owner = %q, want %q", got, want)
		}
		if got, want := stringAt(t, github, "name"), "agh"; got != want {
			t.Fatalf("release.github.name = %q, want %q", got, want)
		}

		extraFiles := sliceAt(t, release, "extra_files")
		assertReleaseExtraFile(t, extraFiles, "./packages/site/public/install.sh", "install.sh")
	})

	t.Run("Should configure Homebrew formula and Linux package targets", func(t *testing.T) {
		if _, ok := cfg["homebrew_casks"]; ok {
			t.Fatal("homebrew_casks configured, want Homebrew formula via brews")
		}
		brews := sliceAt(t, cfg, "brews")
		if len(brews) != 1 {
			t.Fatalf("brews len = %d, want 1", len(brews))
		}
		formula := asMap(t, brews[0], "brews[0]")
		if got, want := stringAt(t, formula, "name"), "agh"; got != want {
			t.Fatalf("brews[0].name = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, formula, "ids"), "agh-archive") {
			t.Fatalf("brews[0].ids = %#v, want agh-archive", formula["ids"])
		}
		if got, want := stringAt(t, formula, "directory"), "Formula"; got != want {
			t.Fatalf("brews[0].directory = %q, want %q", got, want)
		}
		repository := mapAt(t, formula, "repository")
		if got, want := stringAt(t, repository, "owner"), "compozy"; got != want {
			t.Fatalf("brews[0].repository.owner = %q, want %q", got, want)
		}
		if got, want := stringAt(t, repository, "name"), "homebrew-compozy"; got != want {
			t.Fatalf("brews[0].repository.name = %q, want %q", got, want)
		}
		if _, ok := formula["binaries"]; ok {
			t.Fatal("brews[0].binaries configured, want formula archive install semantics")
		}

		nfpms := sliceAt(t, cfg, "nfpms")
		if len(nfpms) != 1 {
			t.Fatalf("nfpms len = %d, want 1", len(nfpms))
		}
		nfpm := asMap(t, nfpms[0], "nfpms[0]")
		if got, want := stringAt(t, nfpm, "id"), "agh-linux-packages"; got != want {
			t.Fatalf("nfpms[0].id = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, nfpm, "ids"), "agh") {
			t.Fatalf("nfpms[0].ids = %#v, want agh build id", nfpm["ids"])
		}
		formats := sliceAt(t, nfpm, "formats")
		for _, want := range []string{"deb", "rpm"} {
			if !stringSliceContains(formats, want) {
				t.Fatalf("nfpms[0].formats = %#v, want %s", formats, want)
			}
		}
	})

	t.Run("Should configure public NPM package target", func(t *testing.T) {
		npms := sliceAt(t, cfg, "npms")
		if len(npms) != 1 {
			t.Fatalf("npms len = %d, want 1", len(npms))
		}
		npm := asMap(t, npms[0], "npms[0]")
		assertEqualString(t, "npms[0].name", stringAt(t, npm, "name"), "@compozy/agh")
		if !stringSliceContains(sliceAt(t, npm, "ids"), "agh-archive") {
			t.Fatalf("npms[0].ids = %#v, want agh-archive", npm["ids"])
		}
		assertEqualString(t, "npms[0].access", stringAt(t, npm, "access"), "public")
		assertEqualString(t, "npms[0].format", stringAt(t, npm, "format"), "tar.gz")
		assertEqualString(
			t,
			"npms[0].repository",
			stringAt(t, npm, "repository"),
			"git+https://github.com/compozy/agh.git",
		)
		assertEqualString(t, "npms[0].homepage", stringAt(t, npm, "homepage"), "https://agh.network")
	})
}

func TestPackagingMetadataStaysAlignedWithRuntimeAndInstaller(t *testing.T) {
	t.Parallel()

	root := findRepoRootForReleaseConfigTest(t)
	goreleaser := readYAMLMap(t, root, ".goreleaser.yml")
	ciWorkflow := readYAMLMap(t, root, filepath.Join(".github", "workflows", "ci.yml"))
	releaseWorkflow := readYAMLMap(t, root, filepath.Join(".github", "workflows", "release.yml"))
	setupBun := readYAMLMap(t, root, filepath.Join(".github", "actions", "setup-bun", "action.yml"))
	setupGo := readYAMLMap(t, root, filepath.Join(".github", "actions", "setup-go", "action.yml"))
	rootPackage := readJSONMap(t, root, "package.json")
	prRelease := readYAMLMap(t, root, ".pr-release")
	installScript := readTextFile(t, root, filepath.Join("packages", "site", "public", "install.sh"))

	t.Run("Should keep toolchain versions synchronized across package metadata and workflows", func(t *testing.T) {
		t.Parallel()

		bunVersionFile := strings.TrimSpace(readTextFile(t, root, ".bun-version"))
		packageManager := stringAt(t, rootPackage, "packageManager")
		bunVersion, ok := strings.CutPrefix(packageManager, "bun@")
		if !ok {
			t.Fatalf("packageManager = %q, want bun@<version>", packageManager)
		}
		assertEqualString(t, "packageManager bun version", bunVersion, bunVersionFile)
		assertEqualString(
			t,
			"ci env BUN_VERSION",
			workflowEnvValue(t, ciWorkflow, "BUN_VERSION"),
			bunVersionFile,
		)
		assertEqualString(
			t,
			"release env BUN_VERSION",
			workflowEnvValue(t, releaseWorkflow, "BUN_VERSION"),
			bunVersionFile,
		)
		setupBunInputs := mapAt(t, setupBun, "inputs")
		setupBunVersion := mapAt(t, setupBunInputs, "bun-version")
		assertEqualString(t, "setup-bun default", stringAt(t, setupBunVersion, "default"), bunVersionFile)

		goVersion := goDirectiveVersion(t, readTextFile(t, root, "go.mod"))
		assertEqualString(t, "ci env GO_VERSION", workflowEnvValue(t, ciWorkflow, "GO_VERSION"), goVersion)
		assertEqualString(
			t,
			"release env GO_VERSION",
			workflowEnvValue(t, releaseWorkflow, "GO_VERSION"),
			goVersion,
		)
		setupGoInputs := mapAt(t, setupGo, "inputs")
		setupGoVersion := mapAt(t, setupGoInputs, "go-version")
		assertEqualString(t, "setup-go default", stringAt(t, setupGoVersion, "default"), goVersion)
	})

	t.Run("Should keep Bun workspace release artifacts backed by workspace metadata", func(t *testing.T) {
		t.Parallel()

		workspaces := stringsFromSlice(t, sliceAt(t, rootPackage, "workspaces"), "workspaces")
		for _, workspace := range []string{
			"packages/ui",
			"packages/site",
			"web",
			"sdk/typescript",
			"sdk/create-extension",
			"sdk/examples/prompt-enhancer",
		} {
			if !stringListContains(workspaces, workspace) {
				t.Fatalf("package.json workspaces = %#v, want %q", workspaces, workspace)
			}
		}

		cachePaths := setupBunCachePaths(t, setupBun)
		for _, workspace := range workspaces {
			packagePath := filepath.Join(root, workspace, "package.json")
			if _, err := os.Stat(packagePath); err != nil {
				t.Fatalf("workspace %q package.json stat error = %v", workspace, err)
			}
			cachePath := filepath.ToSlash(filepath.Join(workspace, "node_modules"))
			if !strings.Contains(cachePaths, cachePath) {
				t.Fatalf("setup-bun cache paths = %q, want workspace cache path %q", cachePaths, cachePath)
			}
		}

		scripts := mapAt(t, rootPackage, "scripts")
		artifact := siteChangelogArtifact(t, prRelease)
		args := stringsFromSlice(t, sliceAt(t, artifact, "args"), "release_artifacts[site-changelog].args")
		if !stringListContains(args, "release:site-changelog") {
			t.Fatalf("site-changelog args = %#v, want release:site-changelog", args)
		}
		changelogScript := stringAt(t, scripts, "release:site-changelog")
		if !strings.Contains(changelogScript, "packages/site/scripts/generate-changelog-release.ts") {
			t.Fatalf("scripts.release:site-changelog = %q, want packages/site changelog generator", changelogScript)
		}
	})

	t.Run("Should keep GoReleaser archives aligned with the public installer", func(t *testing.T) {
		t.Parallel()

		projectName := stringAt(t, goreleaser, "project_name")
		assertEqualString(t, "goreleaser project_name", projectName, "agh")
		build := firstMapAt(t, goreleaser, "builds")
		buildID := stringAt(t, build, "id")
		assertEqualString(t, "build id", buildID, "agh")
		assertEqualString(t, "build binary", stringAt(t, build, "binary"), "agh")
		assertEqualString(t, "build main", stringAt(t, build, "main"), "./cmd/agh")
		ldflags := strings.Join(stringsFromSlice(t, sliceAt(t, build, "ldflags"), "builds[0].ldflags"), "\n")
		assertContainsText(
			t,
			"GoReleaser ldflags",
			ldflags,
			"github.com/compozy/agh/internal/version.Version",
		)
		assertNotContainsText(t, "GoReleaser ldflags", ldflags, "github.com/pedronauck/agh")

		archive := firstMapAt(t, goreleaser, "archives")
		if !stringSliceContains(sliceAt(t, archive, "ids"), buildID) {
			t.Fatalf("archives[0].ids = %#v, want build id %q", archive["ids"], buildID)
		}
		nameTemplate := stringAt(t, archive, "name_template")
		for _, fragment := range []string{
			"{{ .ProjectName }}_{{ .Os }}_",
			`{{- if eq .Arch "amd64" }}x86_64`,
			`{{- else if eq .Arch "386" }}i386`,
			`{{- else }}{{ .Arch }}{{ end }}`,
		} {
			if !strings.Contains(nameTemplate, fragment) {
				t.Fatalf("archives[0].name_template = %q, want fragment %q", nameTemplate, fragment)
			}
		}
		if !strings.Contains(installScript, `ARCHIVE_NAME="agh_${OS}_${ARCH}.tar.gz"`) {
			t.Fatalf("install.sh archive naming must stay aligned with GoReleaser template")
		}

		goos := stringsFromSlice(t, sliceAt(t, build, "goos"), "builds[0].goos")
		goarch := stringsFromSlice(t, sliceAt(t, build, "goarch"), "builds[0].goarch")
		for _, platform := range []string{"linux", "darwin"} {
			if !stringListContains(goos, platform) {
				t.Fatalf("builds[0].goos = %#v, want installer platform %q", goos, platform)
			}
		}
		for _, arch := range []string{"amd64", "arm64"} {
			if !stringListContains(goarch, arch) {
				t.Fatalf("builds[0].goarch = %#v, want installer architecture %q", goarch, arch)
			}
		}

		release := mapAt(t, goreleaser, "release")
		github := mapAt(t, release, "github")
		releaseRepo := shellAssignment(t, installScript, "RELEASE_REPO")
		goreleaserRepo := stringAt(t, github, "owner") + "/" + stringAt(t, github, "name")
		assertEqualString(t, "installer RELEASE_REPO", releaseRepo, goreleaserRepo)
		if !strings.Contains(installScript, `TARGET="${INSTALL_DIR}/agh"`) {
			t.Fatalf("install.sh must install the same binary name GoReleaser builds")
		}
	})
}

func TestPRReleaseConfigGeneratesReleaseArtifacts(t *testing.T) {
	t.Parallel()

	t.Run("Should generate and format release artifacts", func(t *testing.T) {
		t.Parallel()

		root := findRepoRootForReleaseConfigTest(t)
		cfg := readYAMLMap(t, root, ".pr-release")
		artifacts := sliceAt(t, cfg, "release_artifacts")
		if len(artifacts) != 2 {
			t.Fatalf("release_artifacts len = %d, want 2", len(artifacts))
		}

		siteArtifact := asMap(t, artifacts[0], "release_artifacts[0]")
		if got, want := stringAt(t, siteArtifact, "name"), "site-changelog"; got != want {
			t.Fatalf("release_artifacts[0].name = %q, want %q", got, want)
		}
		if got, want := stringAt(t, siteArtifact, "command"), "bun"; got != want {
			t.Fatalf("release_artifacts[0].command = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, siteArtifact, "args"), "release:site-changelog") {
			t.Fatalf("release_artifacts[0].args = %#v, want release:site-changelog", siteArtifact["args"])
		}
		if !stringSliceContains(sliceAt(t, siteArtifact, "add"), "packages/site/content/blog/changelog/*.mdx") {
			t.Fatalf("release_artifacts[0].add = %#v, want site changelog glob", siteArtifact["add"])
		}

		formatArtifact := asMap(t, artifacts[1], "release_artifacts[1]")
		if got, want := stringAt(t, formatArtifact, "name"), "format-release-artifacts"; got != want {
			t.Fatalf("release_artifacts[1].name = %q, want %q", got, want)
		}
		if got, want := stringAt(t, formatArtifact, "command"), "bun"; got != want {
			t.Fatalf("release_artifacts[1].command = %q, want %q", got, want)
		}
		formatArgs := sliceAt(t, formatArtifact, "args")
		for _, arg := range []string{
			"x",
			"oxfmt",
			"CHANGELOG.md",
			"RELEASE_BODY.md",
			"RELEASE_NOTES.md",
			"package.json",
			"packages/site/content/blog/changelog/*.mdx",
		} {
			if !stringSliceContains(formatArgs, arg) {
				t.Fatalf("release_artifacts[1].args = %#v, want %q", formatArtifact["args"], arg)
			}
		}
		formatAdds := sliceAt(t, formatArtifact, "add")
		for _, path := range []string{
			"CHANGELOG.md",
			"RELEASE_BODY.md",
			"RELEASE_NOTES.md",
			"package.json",
			"packages/site/content/blog/changelog/*.mdx",
		} {
			if !stringSliceContains(formatAdds, path) {
				t.Fatalf("release_artifacts[1].add = %#v, want %q", formatArtifact["add"], path)
			}
		}
	})
}

func TestReleaseWorkflowPreservesInstallerSourceTextGuards(t *testing.T) {
	t.Parallel()

	root := findRepoRootForReleaseConfigTest(t)
	workflow := readTextFile(t, root, filepath.Join(".github", "workflows", "release.yml"))
	header := readTextFile(t, root, ".goreleaser.release-header.md.tmpl")
	footer := readTextFile(t, root, ".goreleaser.release-footer.md.tmpl")

	t.Run("Should keep release workflow guards for public installer provenance", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"sh -n packages/site/public/install.sh",
			"grep -q 'checksums.txt.sigstore.json' packages/site/public/install.sh",
			"install.sh must verify checksums.txt with checksums.txt.sigstore.json",
			"grep -q 'packages/site/public/install.sh' .goreleaser.yml",
			".goreleaser.yml must upload packages/site/public/install.sh as a release extra file",
			`grep -q 'name: "@compozy/agh"' .goreleaser.yml`,
			".goreleaser.yml must publish the @compozy/agh npm package",
			`grep -q -- '--bundle=\${signature}' .goreleaser.yml`,
			".goreleaser.yml must sign checksums with a Sigstore bundle artifact",
		} {
			assertContainsText(t, "release workflow", workflow, snippet)
		}
	})

	t.Run("Should keep release header aligned with public install methods", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"brew install compozy/compozy/agh",
			"npm install -g @compozy/agh",
			"go install github.com/compozy/agh/cmd/agh@{{ .Tag }}",
			"curl -fsSL https://agh.network/install.sh | sh",
			"Verified Binary Installer",
		} {
			assertContainsText(t, "GoReleaser release header", header, snippet)
		}
		assertNotContainsText(t, "GoReleaser release header", header, "github.com/pedronauck/agh")
	})

	t.Run("Should keep GoReleaser invocation tied to generated release text", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"goreleaser/goreleaser-action@v6",
			"distribution: goreleaser-pro",
			"release --clean",
			"--release-notes=RELEASE_BODY.md",
			"--release-header-tmpl=.goreleaser.release-header.md.tmpl",
			"--release-footer-tmpl=.goreleaser.release-footer.md.tmpl",
		} {
			assertContainsText(t, "release workflow", workflow, snippet)
		}
	})

	t.Run("Should keep release artifacts honest about verification posture", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"### Verification posture",
			"`make verify` covers codegen drift",
			"`pr-release dry-run`, `make test-e2e-nightly`, and `make test-integration`",
			"`goreleaser release --clean` publishes the release",
			"`checksums.txt.sigstore.json`",
			"Syft SBOMs for archives, packages, and source",
			"does not claim a manual post-release install smoke",
			"--bundle checksums.txt.sigstore.json",
			"--certificate-identity-regexp",
			"--certificate-oidc-issuer https://token.actions.githubusercontent.com",
		} {
			assertContainsText(t, "GoReleaser release footer", footer, snippet)
		}
		assertNotContainsText(t, "GoReleaser release footer", footer, "All release artifacts are signed")
	})
}

func findRepoRootForReleaseConfigTest(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".goreleaser.yml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root with .goreleaser.yml not found")
		}
		dir = parent
	}
}

func readTextFile(t *testing.T, root string, rel string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", rel, err)
	}
	return string(data)
}

func readYAMLMap(t *testing.T, root string, rel string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", rel, err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal(%s) error = %v", rel, err)
	}
	return cfg
}

func readJSONMap(t *testing.T, root string, rel string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", rel, err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", rel, err)
	}
	return cfg
}

func mapAt(t *testing.T, src map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := src[key]
	if !ok {
		t.Fatalf("%s missing", key)
	}
	return asMap(t, value, key)
}

func asMap(t *testing.T, value any, label string) map[string]any {
	t.Helper()

	item, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s type = %T, want map[string]any", label, value)
	}
	return item
}

func sliceAt(t *testing.T, src map[string]any, key string) []any {
	t.Helper()

	value, ok := src[key]
	if !ok {
		t.Fatalf("%s missing", key)
	}
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("%s type = %T, want []any", key, value)
	}
	return items
}

func stringAt(t *testing.T, src map[string]any, key string) string {
	t.Helper()

	value, ok := src[key]
	if !ok {
		t.Fatalf("%s missing", key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s type = %T, want string", key, value)
	}
	return text
}

func stringSliceContains(values []any, want string) bool {
	for _, value := range values {
		if text, ok := value.(string); ok && text == want {
			return true
		}
	}
	return false
}

func stringsFromSlice(t *testing.T, values []any, label string) []string {
	t.Helper()

	items := make([]string, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s[%d] type = %T, want string", label, index, value)
		}
		items = append(items, text)
	}
	return items
}

func stringListContains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func firstMapAt(t *testing.T, src map[string]any, key string) map[string]any {
	t.Helper()

	items := sliceAt(t, src, key)
	if len(items) == 0 {
		t.Fatalf("%s is empty", key)
	}
	return asMap(t, items[0], key+"[0]")
}

func workflowEnvValue(t *testing.T, workflow map[string]any, key string) string {
	t.Helper()

	env := mapAt(t, workflow, "env")
	return stringAt(t, env, key)
}

func setupBunCachePaths(t *testing.T, action map[string]any) string {
	t.Helper()

	runs := mapAt(t, action, "runs")
	steps := sliceAt(t, runs, "steps")
	for _, step := range steps {
		item := asMap(t, step, "runs.steps[]")
		if id, ok := item["id"].(string); ok && id == "bun-cache" {
			with := mapAt(t, item, "with")
			return stringAt(t, with, "path")
		}
	}
	t.Fatal("setup-bun action missing bun-cache step")
	return ""
}

func siteChangelogArtifact(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()

	for _, entry := range sliceAt(t, cfg, "release_artifacts") {
		artifact := asMap(t, entry, "release_artifacts[]")
		if stringAt(t, artifact, "name") == "site-changelog" {
			return artifact
		}
	}
	t.Fatal("release_artifacts missing site-changelog")
	return nil
}

func goDirectiveVersion(t *testing.T, goMod string) string {
	t.Helper()

	for line := range strings.SplitSeq(goMod, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "go" {
			return fields[1]
		}
	}
	t.Fatal("go.mod missing go directive")
	return ""
}

func shellAssignment(t *testing.T, script string, key string) string {
	t.Helper()

	prefix := key + "=\""
	for line := range strings.SplitSeq(script, "\n") {
		if value, ok := strings.CutPrefix(line, prefix); ok {
			trimmed, ok := strings.CutSuffix(value, "\"")
			if !ok {
				t.Fatalf("%s assignment = %q, want quoted shell string", key, line)
			}
			return trimmed
		}
	}
	t.Fatalf("install.sh missing %s assignment", key)
	return ""
}

func assertEqualString(t *testing.T, label string, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
}

func assertContainsText(t *testing.T, label string, text string, want string) {
	t.Helper()

	if !strings.Contains(text, want) {
		t.Fatalf("%s missing %q", label, want)
	}
}

func assertNotContainsText(t *testing.T, label string, text string, unwanted string) {
	t.Helper()

	if strings.Contains(text, unwanted) {
		t.Fatalf("%s contains %q", label, unwanted)
	}
}

func assertSBOMArtifact(t *testing.T, sboms []any, artifact string) {
	t.Helper()

	for _, entry := range sboms {
		sbom := asMap(t, entry, "sboms[]")
		if value, ok := sbom["artifacts"].(string); ok && value == artifact {
			return
		}
	}
	t.Fatalf("sboms = %#v, want artifacts %q", sboms, artifact)
}

func assertReleaseExtraFile(t *testing.T, extraFiles []any, glob string, nameTemplate string) {
	t.Helper()

	for _, entry := range extraFiles {
		extraFile := asMap(t, entry, "release.extra_files[]")
		if stringAt(t, extraFile, "glob") == glob &&
			stringAt(t, extraFile, "name_template") == nameTemplate {
			return
		}
	}
	t.Fatalf("release.extra_files = %#v, want glob %q with name_template %q", extraFiles, glob, nameTemplate)
}
