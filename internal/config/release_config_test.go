package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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
		t.Parallel()

		sboms := sliceAt(t, cfg, "sboms")
		assertUniqueSBOMIDs(t, sboms)
		assertSBOMArtifact(t, sboms, "archive", "archive")
		assertSBOMArtifact(t, sboms, "package", "package")
		assertSBOMArtifact(t, sboms, "source", "source")
	})

	t.Run("Should build embedded web bundle before release binaries", func(t *testing.T) {
		t.Parallel()

		before := mapAt(t, cfg, "before")
		hooks := sliceAt(t, before, "hooks")
		if !stringSliceContains(hooks, "go run github.com/magefile/mage@v1.17.0 webBuild") {
			t.Fatalf("before.hooks = %#v, want webBuild before GoReleaser builds embedded web assets", hooks)
		}
		if !stringSliceContains(hooks, "go run github.com/magefile/mage@v1.17.0 webAssetsCheck") {
			t.Fatalf("before.hooks = %#v, want webAssetsCheck before GoReleaser builds binaries", hooks)
		}
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
	syncWebAssetsWorkflow := readYAMLMap(t, root, filepath.Join(".github", "workflows", "sync-web-assets.yml"))
	setupBun := readYAMLMap(t, root, filepath.Join(".github", "actions", "setup-bun", "action.yml"))
	setupGo := readYAMLMap(t, root, filepath.Join(".github", "actions", "setup-go", "action.yml"))
	setupNode := readYAMLMap(t, root, filepath.Join(".github", "actions", "setup-node", "action.yml"))
	rootPackage := readJSONMap(t, root, "package.json")
	prRelease := readYAMLMap(t, root, ".pr-release")
	installScript := readTextFile(t, root, filepath.Join("packages", "site", "public", "install.sh"))
	ciWorkflowText := readTextFile(t, root, filepath.Join(".github", "workflows", "ci.yml"))
	releaseWorkflowText := readTextFile(t, root, filepath.Join(".github", "workflows", "release.yml"))
	syncWebAssetsWorkflowText := readTextFile(t, root, filepath.Join(".github", "workflows", "sync-web-assets.yml"))
	cliffConfig := readTextFile(t, root, "cliff.toml")
	gitignore := readTextFile(t, root, ".gitignore")
	goMod := readTextFile(t, root, "go.mod")
	magefile := readTextFile(t, root, "magefile.go")
	staticSource := readTextFile(t, root, filepath.Join("internal", "api", "httpapi", "static.go"))
	goreleaserText := readTextFile(t, root, ".goreleaser.yml")

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

		nodeVersion := workflowEnvValue(t, releaseWorkflow, "NODE_VERSION")
		setupNodeInputs := mapAt(t, setupNode, "inputs")
		setupNodeVersion := mapAt(t, setupNodeInputs, "node-version")
		assertEqualString(t, "setup-node default", stringAt(t, setupNodeVersion, "default"), nodeVersion)
	})

	t.Run("Should isolate release verification gates across runners", func(t *testing.T) {
		t.Parallel()

		jobs := mapAt(t, releaseWorkflow, "jobs")
		assertWorkflowJobCommand(
			t,
			jobs,
			"dry-run",
			60,
			"go run \"${{ env.PR_RELEASE_MODULE }}\" dry-run --ci-output",
		)
		assertWorkflowJobCommand(t, jobs, "e2e-nightly", 90, "make test-e2e-nightly")
		assertWorkflowJobCommand(t, jobs, "integration", 120, "make test-integration")

		dryRun := mapAt(t, jobs, "dry-run")
		dryRunCommands := workflowRunCommands(t, sliceAt(t, dryRun, "steps"))
		for _, unwanted := range []string{"make test-e2e-nightly", "make test-integration"} {
			if stringListContains(dryRunCommands, unwanted) {
				t.Fatalf("dry-run step commands = %#v, want %q isolated in its own job", dryRunCommands, unwanted)
			}
		}
	})

	t.Run("Should keep the tagless repair release on v0.0.2", func(t *testing.T) {
		t.Parallel()

		assertEqualString(
			t,
			"release env INITIAL_VERSION",
			workflowEnvValue(t, releaseWorkflow, "INITIAL_VERSION"),
			"v0.0.2",
		)
		assertContainsText(t, "cliff initial tag", cliffConfig, `initial_tag = "v0.0.2"`)
		assertNotContainsText(t, "cliff initial tag", cliffConfig, `initial_tag = "v0.0.1"`)
	})

	t.Run("Should keep Go source installs backed by the external web assets module", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"webAssetsModulePath       = \"github.com/compozy/agh-web-assets\"",
			"func WebAssetsCheck() error",
			"func ReleaseWebAssetsSync() error",
			"func ReleaseInstallCheck() error",
			"newWebAssetsGitCredentials(token)",
			"cloneWebAssetsRepository(ctx, assetsRepoDir, gitCredentials.env)",
			"publishWebAssetsModule(ctx, assetsRepoDir, metadata, gitCredentials.env)",
			"runCommandInDirWithEnv(ctx, assetsRepoDir, gitEnv, \"git\", \"push\", \"origin\", \"HEAD:main\", nextTag)",
			"runCommandInDirWithEnv(ctx, assetsRepoDir, gitEnv, \"git\", \"push\", \"origin\", nextTag)",
			"WebAssetsPublicCheck",
			"SourceInstallCheck",
		} {
			assertContainsText(t, "magefile", magefile, snippet)
		}
		assertContainsText(t, "go.mod", goMod, "github.com/compozy/agh-web-assets")
		assertContainsText(t, "httpapi static fs", staticSource, "github.com/compozy/agh-web-assets")
		assertContainsText(t, "httpapi static fs", staticSource, `fs.Sub(webassets.DistFS, webassets.DistDir)`)
		assertContainsText(t, "httpapi static fs", staticSource, "embedded web bundle missing index.html")
		assertContainsText(t, "httpapi static fs", staticSource, "AGH_WEB_DIST_DIR")
		assertContainsText(t, "gitignore", gitignore, "web/embed/")
		assertNotContainsText(t, "gitignore", gitignore, "!web/embed/")
		assertNotContainsText(t, "gitignore", gitignore, "!web/embed/**")
		assertNotContainsText(t, "gitignore", gitignore, "!web/dist/")
		assertContainsText(
			t,
			"release workflow",
			releaseWorkflowText,
			"go run github.com/magefile/mage@v1.17.0 releaseInstallCheck",
		)
		assertNotContainsText(
			t,
			"release workflow",
			releaseWorkflowText,
			"go run github.com/magefile/mage@v1.17.0 webAssetsCheck",
		)
		assertNotContainsText(t, "release workflow", releaseWorkflowText, "web/embed/index.html")
	})

	t.Run("Should run asset install validation only in release-owned lanes", func(t *testing.T) {
		t.Parallel()

		assertContainsText(
			t,
			"release workflow",
			releaseWorkflowText,
			"go run github.com/magefile/mage@v1.17.0 releaseInstallCheck",
		)
		for _, forbidden := range []string{"webAssetsPrepare", "webAssetsNextTag"} {
			assertNotContainsText(t, "release workflow", releaseWorkflowText, forbidden)
			assertNotContainsText(t, "GoReleaser config", goreleaserText, forbidden)
		}

		jobs := mapAt(t, releaseWorkflow, "jobs")
		release := mapAt(t, jobs, "release")
		releaseSteps := sliceAt(t, release, "steps")
		assertWorkflowRunBeforeUses(
			t,
			releaseSteps,
			"go run github.com/magefile/mage@v1.17.0 releaseInstallCheck",
			"goreleaser/goreleaser-action@v6",
		)
		assertWorkflowJobCommand(
			t,
			jobs,
			"dry-run",
			90,
			"go run github.com/magefile/mage@v1.17.0 releaseInstallCheck",
		)
	})

	t.Run("Should keep privileged web assets sync out of pull request events", func(t *testing.T) {
		t.Parallel()

		assertNotContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "pull_request:")
		assertNotContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "pull_request_target:")
		assertContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "cancel-in-progress: false")
		assertContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "origin/main")
		assertContainsText(
			t,
			"sync web assets workflow",
			syncWebAssetsWorkflowText,
			"GOPROXY=https://proxy.golang.org,direct",
		)
		assertContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "GOPRIVATE=")
		assertContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "GONOPROXY: \"\"")
		assertNotContainsText(t, "sync web assets workflow", syncWebAssetsWorkflowText, "GOPROXY: direct")
		waitIndex := strings.Index(syncWebAssetsWorkflowText, "- name: Wait for public module availability")
		updateIndex := strings.Index(syncWebAssetsWorkflowText, "- name: Update pinned assets module")
		if waitIndex == -1 || updateIndex == -1 {
			t.Fatalf("sync web assets workflow missing wait/update steps")
		}
		if waitIndex > updateIndex {
			t.Fatalf("sync web assets workflow updates go.mod before public proxy availability is proven")
		}

		permissions := mapAt(t, syncWebAssetsWorkflow, "permissions")
		assertEqualString(t, "sync contents permission", stringAt(t, permissions, "contents"), "write")
		assertEqualString(t, "sync pull request permission", stringAt(t, permissions, "pull-requests"), "write")

		jobs := mapAt(t, syncWebAssetsWorkflow, "jobs")
		syncJob := mapAt(t, jobs, "sync")
		if got := stringAt(t, syncJob, "if"); !strings.Contains(got, "github.repository == 'compozy/agh'") {
			t.Fatalf("sync job if = %q, want repository guard", got)
		}
		commands := strings.Join(workflowRunCommands(t, sliceAt(t, syncJob, "steps")), "\n")
		for _, snippet := range []string{
			"webAssetsDeterminismCheck",
			"webAssetsPrepare",
			"webAssetsNextTag",
			"go get",
			"releaseInstallCheck",
		} {
			assertContainsText(t, "sync web assets commands", commands, snippet)
		}
	})

	t.Run("Should keep CI path filtering resilient to rollback pushes", func(t *testing.T) {
		t.Parallel()

		for _, snippet := range []string{
			"git cat-file -e \"${PUSH_BEFORE_SHA}^{commit}\"",
			"Base push SHA $PUSH_BEFORE_SHA is unavailable; treating current tree as changed.",
			"git ls-tree -r --name-only \"$CURRENT_SHA\"",
		} {
			assertContainsText(t, "CI workflow", ciWorkflowText, snippet)
		}
	})

	t.Run("Should bootstrap Bun dependencies before production GoReleaser hooks", func(t *testing.T) {
		t.Parallel()

		jobs := mapAt(t, releaseWorkflow, "jobs")
		release := mapAt(t, jobs, "release")
		steps := sliceAt(t, release, "steps")
		setupBun := workflowStepByUses(t, steps, "./.github/actions/setup-bun")
		with := mapAt(t, setupBun, "with")
		assertEqualString(
			t,
			"release setup-bun bun-version",
			stringAt(t, with, "bun-version"),
			"${{ env.BUN_VERSION }}",
		)
		assertEqualString(t, "release setup-bun install-playwright", stringAt(t, with, "install-playwright"), "false")
		switch installDependencies := with["install-dependencies"].(type) {
		case string:
			if installDependencies == "false" {
				t.Fatal("release setup-bun disables dependency installation before GoReleaser")
			}
		case bool:
			if !installDependencies {
				t.Fatal("release setup-bun disables dependency installation before GoReleaser")
			}
		}
		assertWorkflowUsesBefore(t, steps, "./.github/actions/setup-bun", "./.github/actions/setup-release")
		assertWorkflowUsesBefore(t, steps, "./.github/actions/setup-bun", "goreleaser/goreleaser-action@v6")
	})

	t.Run("Should bootstrap npm authentication before release publishing", func(t *testing.T) {
		t.Parallel()

		assertEqualString(
			t,
			"release env NODE_AUTH_TOKEN",
			workflowEnvValue(t, releaseWorkflow, "NODE_AUTH_TOKEN"),
			"${{ secrets.NPM_TOKEN }}",
		)
		assertEqualString(
			t,
			"release env NPM_TOKEN",
			workflowEnvValue(t, releaseWorkflow, "NPM_TOKEN"),
			"${{ secrets.NPM_TOKEN }}",
		)

		runs := mapAt(t, setupNode, "runs")
		setupNodeSteps := sliceAt(t, runs, "steps")
		setupNodeAction := workflowStepByUses(t, setupNodeSteps, "actions/setup-node@v6")
		with := mapAt(t, setupNodeAction, "with")
		assertEqualString(
			t,
			"setup-node registry-url",
			stringAt(t, with, "registry-url"),
			"https://registry.npmjs.org",
		)
		assertEqualString(t, "setup-node scope", stringAt(t, with, "scope"), "@compozy")
		if stringListContains(workflowRunCommands(t, setupNodeSteps), "npm whoami") {
			t.Fatalf(
				"setup-node step commands = %#v, want npm auth checks in release-owned jobs",
				workflowRunCommands(t, setupNodeSteps),
			)
		}

		jobs := mapAt(t, releaseWorkflow, "jobs")
		for _, jobName := range []string{"release-pr", "dry-run", "release"} {
			job := mapAt(t, jobs, jobName)
			steps := sliceAt(t, job, "steps")
			workflowStepByUses(t, steps, "./.github/actions/setup-node")
			assertWorkflowUsesBefore(t, steps, "./.github/actions/setup-node", "./.github/actions/setup-bun")
		}
		release := mapAt(t, jobs, "release")
		releaseSteps := sliceAt(t, release, "steps")
		assertWorkflowUsesBefore(t, releaseSteps, "./.github/actions/setup-node", "goreleaser/goreleaser-action@v6")
		dryRun := mapAt(t, jobs, "dry-run")
		dryRunSteps := sliceAt(t, dryRun, "steps")
		assertWorkflowRunBeforeUses(
			t,
			dryRunSteps,
			"npm view \"@compozy/agh@${RELEASE_VERSION}\" version --registry=https://registry.npmjs.org",
			"./.github/actions/setup-release",
		)
		dryRunCommands := strings.Join(workflowRunCommands(t, dryRunSteps), "\n")
		for _, unwanted := range []string{
			"NPM_TOKEN is required to publish @compozy/agh",
			"npm whoami --registry=https://registry.npmjs.org",
		} {
			if strings.Contains(dryRunCommands, unwanted) {
				t.Fatalf(
					"dry-run commands contain %q, want npm publish authentication only in production release",
					unwanted,
				)
			}
		}
		assertWorkflowRunBeforeUses(
			t,
			releaseSteps,
			"npm whoami --registry=https://registry.npmjs.org",
			"goreleaser/goreleaser-action@v6",
		)
		assertWorkflowRunBeforeUses(
			t,
			releaseSteps,
			"npm view \"@compozy/agh@${RELEASE_VERSION}\" version --registry=https://registry.npmjs.org",
			"goreleaser/goreleaser-action@v6",
		)
		for _, snippet := range []string{
			"AGH_WEB_ASSETS_TOKEN: ${{ secrets.AGH_WEB_ASSETS_TOKEN || secrets.RELEASE_TOKEN }}",
			"name: Resolve dry-run release version",
			"echo \"RELEASE_VERSION=$VERSION\" >> \"$GITHUB_ENV\"",
			"echo \"RELEASE_TAG=$TAG\" >> \"$GITHUB_ENV\"",
			"name: Verify npm package version availability",
			"name: Verify npm publish authentication",
			"NPM_TOKEN is required to publish @compozy/agh",
			"npm whoami --registry=https://registry.npmjs.org",
			"npm_view_output=\"$(mktemp)\"",
			"npm view \"@compozy/agh@${RELEASE_VERSION}\" version --registry=https://registry.npmjs.org >\"$npm_view_output\" 2>&1",
			"npm versions are immutable",
			"grep -Eq 'E404|404 Not Found' \"$npm_view_output\"",
			"Could not confirm @compozy/agh@${RELEASE_VERSION} is unpublished",
		} {
			assertContainsText(t, "release workflow", releaseWorkflowText, snippet)
		}
	})

	t.Run("Should give CI verify enough budget for the full monorepo gate", func(t *testing.T) {
		t.Parallel()

		jobs := mapAt(t, ciWorkflow, "jobs")
		assertWorkflowJobCommand(t, jobs, "verify", 90, "make verify")
		magefile := readTextFile(t, root, "magefile.go")
		if !regexp.MustCompile(`goUnitTestTimeout\s*=\s*"30m"`).MatchString(magefile) {
			t.Fatal("magefile unit test timeout missing goUnitTestTimeout = 30m")
		}
		assertContainsText(
			t,
			"magefile unit test command",
			magefile,
			"\"-timeout\", goUnitTestTimeout, \"./...\"",
		)
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
		if len(artifacts) != 3 {
			t.Fatalf("release_artifacts len = %d, want 3", len(artifacts))
		}

		syncArtifact := asMap(t, artifacts[0], "release_artifacts[0]")
		if got, want := stringAt(t, syncArtifact, "name"), "sync-web-assets-module"; got != want {
			t.Fatalf("release_artifacts[0].name = %q, want %q", got, want)
		}
		if got, want := stringAt(t, syncArtifact, "command"), "go"; got != want {
			t.Fatalf("release_artifacts[0].command = %q, want %q", got, want)
		}
		syncArgs := sliceAt(t, syncArtifact, "args")
		for _, arg := range []string{
			"run",
			"github.com/magefile/mage@v1.17.0",
			"releaseWebAssetsSync",
		} {
			if !stringSliceContains(syncArgs, arg) {
				t.Fatalf("release_artifacts[0].args = %#v, want %q", syncArtifact["args"], arg)
			}
		}
		syncAdds := sliceAt(t, syncArtifact, "add")
		for _, path := range []string{"go.mod", "go.sum"} {
			if !stringSliceContains(syncAdds, path) {
				t.Fatalf("release_artifacts[0].add = %#v, want %q", syncArtifact["add"], path)
			}
		}
		if got, want := intAt(t, syncArtifact, "timeout_seconds"), 900; got != want {
			t.Fatalf("release_artifacts[0].timeout_seconds = %d, want %d", got, want)
		}

		siteArtifact := asMap(t, artifacts[1], "release_artifacts[1]")
		if got, want := stringAt(t, siteArtifact, "name"), "site-changelog"; got != want {
			t.Fatalf("release_artifacts[1].name = %q, want %q", got, want)
		}
		if got, want := stringAt(t, siteArtifact, "command"), "bun"; got != want {
			t.Fatalf("release_artifacts[1].command = %q, want %q", got, want)
		}
		if !stringSliceContains(sliceAt(t, siteArtifact, "args"), "release:site-changelog") {
			t.Fatalf("release_artifacts[1].args = %#v, want release:site-changelog", siteArtifact["args"])
		}
		if !stringSliceContains(sliceAt(t, siteArtifact, "add"), "packages/site/content/blog/changelog/*.mdx") {
			t.Fatalf("release_artifacts[1].add = %#v, want site changelog glob", siteArtifact["add"])
		}

		formatArtifact := asMap(t, artifacts[2], "release_artifacts[2]")
		if got, want := stringAt(t, formatArtifact, "name"), "format-release-artifacts"; got != want {
			t.Fatalf("release_artifacts[2].name = %q, want %q", got, want)
		}
		if got, want := stringAt(t, formatArtifact, "command"), "bun"; got != want {
			t.Fatalf("release_artifacts[2].command = %q, want %q", got, want)
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
				t.Fatalf("release_artifacts[2].args = %#v, want %q", formatArtifact["args"], arg)
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
				t.Fatalf("release_artifacts[2].add = %#v, want %q", formatArtifact["add"], path)
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
			`grep -q -- "--bundle=\${signature}" .goreleaser.yml`,
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
			"go install github.com/compozy/agh@{{ .Tag }}",
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
			"refs/heads/main",
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

func intAt(t *testing.T, src map[string]any, key string) int {
	t.Helper()

	value, ok := src[key]
	if !ok {
		t.Fatalf("%s missing", key)
	}
	number, ok := value.(int)
	if !ok {
		t.Fatalf("%s type = %T, want int", key, value)
	}
	return number
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

func workflowRunCommands(t *testing.T, steps []any) []string {
	t.Helper()

	commands := make([]string, 0, len(steps))
	for _, entry := range steps {
		step := asMap(t, entry, "workflow steps[]")
		command, ok := step["run"].(string)
		if ok {
			commands = append(commands, command)
		}
	}
	return commands
}

func workflowStepByUses(t *testing.T, steps []any, uses string) map[string]any {
	t.Helper()

	for _, entry := range steps {
		step := asMap(t, entry, "workflow steps[]")
		if got, ok := step["uses"].(string); ok && got == uses {
			return step
		}
	}
	t.Fatalf("workflow steps missing uses %q", uses)
	return nil
}

func workflowStepIndexByUses(t *testing.T, steps []any, uses string) int {
	t.Helper()

	for index, entry := range steps {
		step := asMap(t, entry, "workflow steps[]")
		if got, ok := step["uses"].(string); ok && got == uses {
			return index
		}
	}
	t.Fatalf("workflow steps missing uses %q", uses)
	return -1
}

func assertWorkflowUsesBefore(t *testing.T, steps []any, earlier string, later string) {
	t.Helper()

	earlierIndex := workflowStepIndexByUses(t, steps, earlier)
	laterIndex := workflowStepIndexByUses(t, steps, later)
	if earlierIndex >= laterIndex {
		t.Fatalf("workflow step %q index = %d, want before %q index %d", earlier, earlierIndex, later, laterIndex)
	}
}

func assertWorkflowRunBeforeUses(t *testing.T, steps []any, runSnippet string, laterUses string) {
	t.Helper()

	runIndex := -1
	for index, entry := range steps {
		step := asMap(t, entry, "workflow steps[]")
		if command, ok := step["run"].(string); ok && strings.Contains(command, runSnippet) {
			runIndex = index
			break
		}
	}
	if runIndex == -1 {
		t.Fatalf("workflow steps missing run containing %q", runSnippet)
	}
	laterIndex := workflowStepIndexByUses(t, steps, laterUses)
	if runIndex >= laterIndex {
		t.Fatalf("workflow run %q index = %d, want before %q index %d", runSnippet, runIndex, laterUses, laterIndex)
	}
}

func assertWorkflowJobCommand(t *testing.T, jobs map[string]any, jobName string, minimumTimeout int, command string) {
	t.Helper()

	job := mapAt(t, jobs, jobName)
	if got := intAt(t, job, "timeout-minutes"); got < minimumTimeout {
		t.Fatalf("%s timeout-minutes = %d, want at least %d", jobName, got, minimumTimeout)
	}
	commands := workflowRunCommands(t, sliceAt(t, job, "steps"))
	if !stringListContains(commands, command) {
		t.Fatalf("%s step commands = %#v, want %q", jobName, commands, command)
	}
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

func assertUniqueSBOMIDs(t *testing.T, sboms []any) {
	t.Helper()

	seen := make(map[string]struct{}, len(sboms))
	for index, entry := range sboms {
		sbom := asMap(t, entry, "sboms[]")
		id := stringAt(t, sbom, "id")
		if _, ok := seen[id]; ok {
			t.Fatalf("sboms[%d].id = %q, want unique SBOM IDs", index, id)
		}
		seen[id] = struct{}{}
	}
}

func assertSBOMArtifact(t *testing.T, sboms []any, id string, artifact string) {
	t.Helper()

	for _, entry := range sboms {
		sbom := asMap(t, entry, "sboms[]")
		if stringAt(t, sbom, "id") == id && stringAt(t, sbom, "artifacts") == artifact {
			return
		}
	}
	t.Fatalf("sboms = %#v, want id %q with artifacts %q", sboms, id, artifact)
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
