//go:build mage

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/codegen/openapits"
	"github.com/compozy/agh/internal/e2elane"
	"github.com/magefile/mage/sh"
)

const (
	golangciLintVersion       = "v2.12.2"
	golangciLintTimeout       = "10m"
	goUnitTestTimeout         = "30m"
	goIntegrationPackageLimit = "2"
	goIntegrationTestTimeout  = "30m"
	goplsModernizeVersion     = "v0.22.0"
	gotestsumVersion          = "v1.13.0"
	binDir                    = "bin"
	cliBinary                 = "agh"
	versionPackage            = "github.com/compozy/agh/internal/version"
	openAPISpecPath           = "openapi/agh.json"
	compozyOpenAPISpecPath    = "openapi/compozy-daemon.json"
	webOpenAPITypePath        = "web/src/generated/agh-openapi.d.ts"
	webCompozyOpenAPITypePath = "web/src/generated/compozy-openapi.d.ts"
	webDistDir                = "web/dist"
	webDistIndex              = "web/dist/index.html"
	webDistDirEnvVar          = "AGH_WEB_DIST_DIR"
	webAssetsModulePath       = "github.com/compozy/agh-web-assets"
	webAssetsRemoteURL        = "https://github.com/compozy/agh-web-assets.git"
	webAssetsModuleDistDir    = "dist"
	webAssetsMetadataFile     = "assets.go"
	webAssetsSourceRepository = "github.com/compozy/agh"
	webAssetsTokenEnvVar      = "AGH_WEB_ASSETS_TOKEN"
	releaseTokenEnvVar        = "RELEASE_TOKEN"
	daemonBinaryEnvVar        = "AGH_TEST_DAEMON_BIN"
	driverBinaryEnvVar        = "AGH_TEST_ACPMOCK_DRIVER_BIN"
	designSyncScriptPath      = "scripts/sync-design-md.mjs"
	daytonaSidecarPackage     = "./internal/sandbox/daytona/cmd/agh-daytona-sidecar"
	daytonaSidecarToolchain   = "1.26.3"
	daytonaSidecarRegenHint   = "go run github.com/magefile/mage@v1.15.0 " +
		"daytonaSidecars"
)

type daytonaSidecarAsset struct {
	arch string
	path string
}

type mageStep struct {
	name string
	run  func() error
}

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

var daytonaSidecarAssets = []daytonaSidecarAsset{
	{
		arch: "amd64",
		path: filepath.Join(
			"internal",
			"sandbox",
			"daytona",
			"sidecar_assets",
			"agh-daytona-sidecar-linux-amd64.gz",
		),
	},
	{
		arch: "arm64",
		path: filepath.Join(
			"internal",
			"sandbox",
			"daytona",
			"sidecar_assets",
			"agh-daytona-sidecar-linux-arm64.gz",
		),
	},
}

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

// Test runs unit tests only (no integration tag).
func Test() error {
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-parallel=4", "-timeout", goUnitTestTimeout, "./...")
}

// TestIntegration runs all tests including integration tests.
func TestIntegration() error {
	return runRaceEnabledGoCommand(context.Background(), nil,
		"run", "gotest.tools/gotestsum@"+gotestsumVersion,
		"--format", "pkgname", "--", "-race", "-p", goIntegrationPackageLimit, "-parallel=4",
		"-timeout", goIntegrationTestTimeout, "-tags", "integration", "./...")
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
	return runMageSteps(buildSteps())
}

func buildSteps() []mageStep {
	return []mageStep{
		{name: "CodegenCheck", run: CodegenCheck},
		{name: "buildGo", run: buildGo},
	}
}

func Codegen() error {
	if err := DaytonaSidecars(); err != nil {
		return err
	}
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
	return SyncDesignMD()
}

func CodegenCheck() error {
	if err := DaytonaSidecarsCheck(); err != nil {
		return err
	}
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
	return SyncDesignMDCheck()
}

// SyncDesignMD refreshes generated DESIGN.md token frontmatter and tables.
func SyncDesignMD() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", designSyncScriptPath, "--write")
}

// SyncDesignMDCheck verifies generated DESIGN.md token frontmatter and tables.
func SyncDesignMDCheck() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", designSyncScriptPath, "--check")
}

// BunLint runs the monorepo-wide lint script (oxfmt + oxlint over every workspace).
func BunLint() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "bun:lint")
}

// BunTypecheck runs the monorepo-wide typecheck pipeline (turbo run typecheck across every workspace).
func BunTypecheck() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "bun:typecheck")
}

// BunTest runs the monorepo-wide vitest projects suite from the repo root.
func BunTest() error {
	return runCommandInDir(context.Background(), ".", "bun", "run", "bun:test")
}

func InstallerCheck() error {
	installer := filepath.Join("packages", "site", "public", "install.sh")
	if err := sh.RunV("sh", "-n", installer); err != nil {
		return err
	}
	return sh.RunV("sh", installer, "--dry-run", "--skip-bootstrap")
}

func WebLint() error {
	return BunLint()
}

func WebTypecheck() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "typecheck:raw")
}

func WebTest() error {
	return runCommandInDir(context.Background(), "web", "bun", "run", "test:raw")
}

func WebBuild() error {
	if err := runCommandInDir(context.Background(), "web", "bun", "run", "build:raw"); err != nil {
		return err
	}
	return ensureWebDist()
}

func ensureWebDist() error {
	if _, err := os.Stat(webDistIndex); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("web build output missing %s", webDistIndex)
		}
		return fmt.Errorf("stat web build output %s: %w", webDistIndex, err)
	}
	return nil
}

func WebAssetsCheck() error {
	if err := ensureWebDist(); err != nil {
		return err
	}
	moduleDir, err := webAssetsModuleDir(context.Background())
	if err != nil {
		return err
	}
	localDigest, err := directoryDigest(webDistDir)
	if err != nil {
		return fmt.Errorf("digest local web build: %w", err)
	}
	moduleDigest, err := directoryDigest(filepath.Join(moduleDir, webAssetsModuleDistDir))
	if err != nil {
		return fmt.Errorf("digest %s module assets: %w", webAssetsModulePath, err)
	}
	metadata, err := readWebAssetsMetadata(moduleDir)
	if err != nil {
		return err
	}
	if metadata.BuildDigest != "" && metadata.BuildDigest != moduleDigest {
		return fmt.Errorf(
			"%s metadata digest %s differs from module dist digest %s",
			webAssetsModulePath,
			metadata.BuildDigest,
			moduleDigest,
		)
	}
	if metadata.SourceRepository != "" && metadata.SourceRepository != webAssetsSourceRepository {
		return fmt.Errorf(
			"%s metadata source repository %q differs from %q",
			webAssetsModulePath,
			metadata.SourceRepository,
			webAssetsSourceRepository,
		)
	}
	if localDigest != moduleDigest {
		return fmt.Errorf(
			"%s is stale: local %s digest %s differs from module %s digest %s",
			webAssetsModulePath,
			webDistDir,
			localDigest,
			webAssetsModuleDistDir,
			moduleDigest,
		)
	}
	return nil
}

func WebAssetsDeterminismCheck() error {
	return webAssetsDeterminismCheck(
		WebBuild,
		func() error {
			return os.RemoveAll(webDistDir)
		},
		func() (string, error) {
			return directoryDigest(webDistDir)
		},
	)
}

func WebAssetsPublicCheck() error {
	ctx := context.Background()
	version, err := pinnedWebAssetsVersion(ctx)
	if err != nil {
		return err
	}
	return webAssetsPublicCheck(ctx, version)
}

func ReleaseWebAssetsSync() error {
	return releaseWebAssetsSync(context.Background())
}

func ReleaseInstallCheck() error {
	return runMageSteps([]mageStep{
		{name: "WebBuild", run: WebBuild},
		{name: "WebAssetsDeterminismCheck", run: WebAssetsDeterminismCheck},
		{name: "WebAssetsCheck", run: WebAssetsCheck},
		{name: "WebAssetsPublicCheck", run: WebAssetsPublicCheck},
		{name: "SourceInstallCheck", run: SourceInstallCheck},
	})
}

func releaseWebAssetsSync(ctx context.Context) error {
	if err := WebAssetsDeterminismCheck(); err != nil {
		return err
	}
	buildDigest, err := directoryDigest(webDistDir)
	if err != nil {
		return fmt.Errorf("digest local web build: %w", err)
	}
	sourceCommit, err := gitCommandOutput(ctx, ".", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve release source commit: %w", err)
	}
	token := webAssetsPublishToken()
	if token == "" {
		return fmt.Errorf("%s or %s is required to publish %s", webAssetsTokenEnvVar, releaseTokenEnvVar, webAssetsModulePath)
	}
	gitCredentials, err := newWebAssetsGitCredentials(token)
	if err != nil {
		return err
	}
	defer gitCredentials.cleanup()

	assetsRepoDir, err := os.MkdirTemp("", "agh-web-assets-release-sync-")
	if err != nil {
		return fmt.Errorf("create web assets sync repo dir: %w", err)
	}
	defer os.RemoveAll(assetsRepoDir)

	if err := cloneWebAssetsRepository(ctx, assetsRepoDir, gitCredentials.env); err != nil {
		return err
	}
	metadata := webAssetsMetadata{
		BuildDigest:      buildDigest,
		SourceRepository: webAssetsSourceRepository,
		SourceCommit:     sourceCommit,
	}
	tag, err := publishWebAssetsModule(ctx, assetsRepoDir, metadata, gitCredentials.env)
	if err != nil {
		return err
	}
	if err := waitForWebAssetsPublic(ctx, tag); err != nil {
		return err
	}
	if err := pinWebAssetsModule(ctx, tag); err != nil {
		return err
	}
	fmt.Printf("Pinned %s@%s for web assets digest %s\n", webAssetsModulePath, tag, buildDigest)
	return nil
}

func webAssetsModuleDir(ctx context.Context) (string, error) {
	if err := runCommandInDir(ctx, ".", "go", "mod", "download", webAssetsModulePath); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Dir}}", webAssetsModulePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("locate %s module: %w\n%s", webAssetsModulePath, err, output)
	}
	moduleDir := strings.TrimSpace(string(output))
	if moduleDir == "" {
		return "", fmt.Errorf("locate %s module: go list returned an empty directory", webAssetsModulePath)
	}
	return moduleDir, nil
}

func webAssetsPublishToken() string {
	if token := strings.TrimSpace(os.Getenv(webAssetsTokenEnvVar)); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv(releaseTokenEnvVar))
}

type webAssetsGitCredentials struct {
	dir string
	env map[string]string
}

func newWebAssetsGitCredentials(token string) (*webAssetsGitCredentials, error) {
	askpassDir, err := os.MkdirTemp("", "agh-web-assets-askpass-")
	if err != nil {
		return nil, fmt.Errorf("create web assets git credentials dir: %w", err)
	}

	askpassPath := filepath.Join(askpassDir, "askpass.sh")
	askpassScript := strings.Join([]string{
		"#!/bin/sh",
		"case \"$1\" in",
		"*Username*) printf '%s\\n' x-access-token ;;",
		"*Password*) printf '%s\\n' \"$AGH_WEB_ASSETS_GIT_TOKEN\" ;;",
		"*) printf '\\n' ;;",
		"esac",
		"",
	}, "\n")
	if err := os.WriteFile(askpassPath, []byte(askpassScript), 0o700); err != nil {
		return nil, fmt.Errorf("write web assets git credentials helper: %w", err)
	}
	return &webAssetsGitCredentials{
		dir: askpassDir,
		env: map[string]string{
			"AGH_WEB_ASSETS_GIT_TOKEN": token,
			"GIT_ASKPASS":              askpassPath,
			"GIT_TERMINAL_PROMPT":      "0",
		},
	}, nil
}

func (c *webAssetsGitCredentials) cleanup() {
	if c == nil || c.dir == "" {
		return
	}
	if err := os.RemoveAll(c.dir); err != nil {
		fmt.Printf("Warning: remove web assets git credentials dir %s: %v\n", c.dir, err)
	}
}

func cloneWebAssetsRepository(ctx context.Context, destDir string, gitEnv map[string]string) error {
	if err := runCommandInDirWithEnv(ctx, ".", gitEnv, "git", "clone", "--no-single-branch", webAssetsRemoteURL, destDir); err != nil {
		return fmt.Errorf("clone %s: %w", webAssetsRemoteURL, err)
	}
	return nil
}

func publishWebAssetsModule(
	ctx context.Context,
	assetsRepoDir string,
	metadata webAssetsMetadata,
	gitEnv map[string]string,
) (string, error) {
	if err := runCommandInDirWithEnv(ctx, assetsRepoDir, gitEnv, "git", "fetch", "--tags", "origin"); err != nil {
		return "", fmt.Errorf("fetch web assets tags: %w", err)
	}
	if err := runCommandInDir(ctx, assetsRepoDir, "git", "config", "user.name", "github-actions[bot]"); err != nil {
		return "", fmt.Errorf("configure web assets git user name: %w", err)
	}
	if err := runCommandInDir(ctx, assetsRepoDir, "git", "config", "user.email", "github-actions[bot]@users.noreply.github.com"); err != nil {
		return "", fmt.Errorf("configure web assets git user email: %w", err)
	}

	tag, err := matchingWebAssetsTag(ctx, assetsRepoDir, metadata)
	if err != nil {
		return "", err
	}
	if tag != "" {
		fmt.Printf("Reusing existing assets tag %s for %s\n", tag, metadata.SourceCommit)
		return tag, nil
	}

	if err := prepareWebAssetsRepo(webDistDir, assetsRepoDir, metadata); err != nil {
		return "", err
	}
	hasDiff, err := gitHasDiff(ctx, assetsRepoDir, webAssetsModuleDistDir, webAssetsMetadataFile)
	if err != nil {
		return "", err
	}
	if hasDiff {
		if err := runCommandInDir(ctx, assetsRepoDir, "git", "add", webAssetsModuleDistDir, webAssetsMetadataFile); err != nil {
			return "", fmt.Errorf("stage web assets module update: %w", err)
		}
		message := fmt.Sprintf("build: sync AGH web assets %s", shortCommit(metadata.SourceCommit))
		if err := runCommandInDir(ctx, assetsRepoDir, "git", "commit", "-m", message); err != nil {
			return "", fmt.Errorf("commit web assets module update: %w", err)
		}
	}

	tags, err := gitTags(assetsRepoDir)
	if err != nil {
		return "", err
	}
	nextTag, err := nextWebAssetsTag(tags)
	if err != nil {
		return "", err
	}
	if err := runCommandInDir(ctx, assetsRepoDir, "git", "tag", "-a", nextTag, "-m", "AGH web assets "+metadata.SourceCommit); err != nil {
		return "", fmt.Errorf("tag web assets module %s: %w", nextTag, err)
	}
	if hasDiff {
		if err := runCommandInDirWithEnv(ctx, assetsRepoDir, gitEnv, "git", "push", "origin", "HEAD:main", nextTag); err != nil {
			return "", fmt.Errorf("push web assets module update %s: %w", nextTag, err)
		}
		return nextTag, nil
	}
	if err := runCommandInDirWithEnv(ctx, assetsRepoDir, gitEnv, "git", "push", "origin", nextTag); err != nil {
		return "", fmt.Errorf("push web assets module tag %s: %w", nextTag, err)
	}
	return nextTag, nil
}

func matchingWebAssetsTag(ctx context.Context, assetsRepoDir string, metadata webAssetsMetadata) (string, error) {
	output, err := gitCommandOutput(ctx, assetsRepoDir, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil {
		return "", fmt.Errorf("list web assets tags: %w", err)
	}
	for _, line := range strings.Split(output, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		source, ok, err := gitShowFile(ctx, assetsRepoDir, tag, webAssetsMetadataFile)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		tagMetadata := parseWebAssetsMetadataSource(source)
		if tagMetadata.BuildDigest == metadata.BuildDigest && tagMetadata.SourceCommit == metadata.SourceCommit {
			return tag, nil
		}
	}
	return "", nil
}

func gitShowFile(ctx context.Context, repoDir string, ref string, path string) (string, bool, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "show", ref+":"+path)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return string(output), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 128 {
		return "", false, nil
	}
	return "", false, fmt.Errorf("read %s from %s: %w\n%s", path, ref, err, output)
}

func gitHasDiff(ctx context.Context, repoDir string, paths ...string) (bool, error) {
	args := []string{"-C", repoDir, "diff", "--quiet", "--"}
	args = append(args, paths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("check web assets diff: %w", err)
}

func gitCommandOutput(ctx context.Context, dir string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func parseWebAssetsMetadataSource(source string) webAssetsMetadata {
	return webAssetsMetadata{
		BuildDigest:      goStringConst(source, "BuildDigest"),
		SourceRepository: goStringConst(source, "SourceRepository"),
		SourceCommit:     goStringConst(source, "SourceCommit"),
	}
}

func shortCommit(commit string) string {
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}

func waitForWebAssetsPublic(ctx context.Context, version string) error {
	const attempts = 30
	const delay = 20 * time.Second

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := webAssetsPublicCheck(ctx, version); err != nil {
			lastErr = err
		} else {
			return nil
		}
		if attempt == attempts {
			break
		}
		fmt.Printf(
			"Waiting for %s@%s to resolve publicly (attempt %d/%d): %v\n",
			webAssetsModulePath,
			version,
			attempt,
			attempts,
			lastErr,
		)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("wait for public %s@%s: %w", webAssetsModulePath, version, ctx.Err())
		case <-timer.C:
		}
	}
	return fmt.Errorf("%s@%s did not resolve publicly after %d attempts: %w", webAssetsModulePath, version, attempts, lastErr)
}

func pinWebAssetsModule(ctx context.Context, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return errors.New("web assets module version is required")
	}
	moduleVersion := webAssetsModulePath + "@" + version
	env := webAssetsPublicModuleEnv("")
	if err := runCommandInDirWithEnv(ctx, ".", env, "go", "get", moduleVersion); err != nil {
		return fmt.Errorf("pin %s: %w", moduleVersion, err)
	}
	if err := runCommandInDirWithEnv(ctx, ".", env, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("tidy after pinning %s: %w", moduleVersion, err)
	}
	return nil
}

func directoryDigest(root string) (string, error) {
	paths := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q is a symlink; embedded web assets must be regular files", path)
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("%s contains no files", root)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", fmt.Errorf("resolve %q relative to %q: %w", path, root, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %q: %w", path, err)
		}
		hash.Write([]byte(filepath.ToSlash(rel)))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

type webAssetsMetadata struct {
	BuildDigest      string
	SourceRepository string
	SourceCommit     string
}

func prepareWebAssetsRepo(srcDistDir string, assetsRepoDir string, metadata webAssetsMetadata) error {
	if metadata.BuildDigest == "" {
		return errors.New("web assets build digest is required")
	}
	if metadata.SourceRepository == "" {
		return errors.New("web assets source repository is required")
	}
	if metadata.SourceCommit == "" {
		return errors.New("web assets source commit is required")
	}
	info, err := os.Stat(assetsRepoDir)
	if err != nil {
		return fmt.Errorf("stat assets repo dir %q: %w", assetsRepoDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("assets repo path %q is not a directory", assetsRepoDir)
	}
	destDistDir := filepath.Join(assetsRepoDir, webAssetsModuleDistDir)
	if err := os.RemoveAll(destDistDir); err != nil {
		return fmt.Errorf("remove existing assets dist %q: %w", destDistDir, err)
	}
	if err := copyWebAssetsDist(srcDistDir, destDistDir); err != nil {
		return err
	}
	return writeWebAssetsMetadata(assetsRepoDir, metadata)
}

func copyWebAssetsDist(srcDir string, destDir string) error {
	return filepath.WalkDir(srcDir, func(srcPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", srcPath, walkErr)
		}
		rel, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("resolve %q relative to %q: %w", srcPath, srcDir, err)
		}
		destPath := filepath.Join(destDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", srcPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%q is a symlink; web assets module dist must contain regular files", srcPath)
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read %q: %w", srcPath, err)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("create parent for %q: %w", destPath, err)
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(destPath, data, mode); err != nil {
			return fmt.Errorf("write %q: %w", destPath, err)
		}
		return nil
	})
}

func writeWebAssetsMetadata(assetsRepoDir string, metadata webAssetsMetadata) error {
	content := strings.Join([]string{
		"// Package webassets embeds the production AGH web UI bundle.",
		"package webassets",
		"",
		"import \"embed\"",
		"",
		"// DistDir is the root directory embedded in DistFS.",
		"const DistDir = \"dist\"",
		"",
		"const (",
		"\tBuildDigest = " + strconv.Quote(metadata.BuildDigest),
		"\tSourceRepository = " + strconv.Quote(metadata.SourceRepository),
		"\tSourceCommit = " + strconv.Quote(metadata.SourceCommit),
		")",
		"",
		"// DistFS embeds the generated production AGH web UI bundle.",
		"//",
		"//go:embed all:dist",
		"var DistFS embed.FS",
		"",
	}, "\n")
	path := filepath.Join(assetsRepoDir, webAssetsMetadataFile)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write web assets metadata %q: %w", path, err)
	}
	return nil
}

func readWebAssetsMetadata(moduleDir string) (webAssetsMetadata, error) {
	data, err := os.ReadFile(filepath.Join(moduleDir, webAssetsMetadataFile))
	if err != nil {
		return webAssetsMetadata{}, fmt.Errorf("read %s metadata: %w", webAssetsModulePath, err)
	}
	return parseWebAssetsMetadataSource(string(data)), nil
}

func goStringConst(source string, name string) string {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		left, right, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(left) != name {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(right))
		if len(fields) == 0 {
			return ""
		}
		value, err := strconv.Unquote(fields[0])
		if err != nil {
			return ""
		}
		return value
	}
	return ""
}

func gitTags(dir string) ([]string, error) {
	cmd := exec.Command("git", "-C", dir, "tag", "--list", "v*")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list assets tags in %q: %w", dir, err)
	}
	lines := strings.Split(string(output), "\n")
	tags := make([]string, 0, len(lines))
	for _, line := range lines {
		tag := strings.TrimSpace(line)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

type webAssetsSemver struct {
	major int
	minor int
	patch int
}

func nextWebAssetsTag(tags []string) (string, error) {
	var highest webAssetsSemver
	found := false
	for _, tag := range tags {
		version, ok := parseWebAssetsSemver(tag)
		if !ok {
			continue
		}
		if !found || compareWebAssetsSemver(version, highest) > 0 {
			highest = version
			found = true
		}
	}
	if !found {
		return "v0.0.1", nil
	}
	if highest.patch == int(^uint(0)>>1) {
		return "", fmt.Errorf("cannot increment patch version for %v", highest)
	}
	highest.patch++
	return fmt.Sprintf("v%d.%d.%d", highest.major, highest.minor, highest.patch), nil
}

func parseWebAssetsSemver(tag string) (webAssetsSemver, bool) {
	raw := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return webAssetsSemver{}, false
	}
	values := [3]int{}
	for idx, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return webAssetsSemver{}, false
		}
		values[idx] = value
	}
	return webAssetsSemver{major: values[0], minor: values[1], patch: values[2]}, true
}

func compareWebAssetsSemver(left webAssetsSemver, right webAssetsSemver) int {
	if left.major != right.major {
		return left.major - right.major
	}
	if left.minor != right.minor {
		return left.minor - right.minor
	}
	return left.patch - right.patch
}

func webAssetsDeterminismCheck(
	build func() error,
	clean func() error,
	digest func() (string, error),
) error {
	if err := clean(); err != nil {
		return fmt.Errorf("clean first web build: %w", err)
	}
	if err := build(); err != nil {
		return fmt.Errorf("first web build: %w", err)
	}
	firstDigest, err := digest()
	if err != nil {
		return fmt.Errorf("digest first web build: %w", err)
	}
	if err := clean(); err != nil {
		return fmt.Errorf("clean second web build: %w", err)
	}
	if err := build(); err != nil {
		return fmt.Errorf("second web build: %w", err)
	}
	secondDigest, err := digest()
	if err != nil {
		return fmt.Errorf("digest second web build: %w", err)
	}
	if firstDigest != secondDigest {
		return fmt.Errorf("web build is not deterministic: first digest %s, second digest %s", firstDigest, secondDigest)
	}
	return nil
}

func pinnedWebAssetsVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", webAssetsModulePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve pinned %s version: %w\n%s", webAssetsModulePath, err, output)
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("resolve pinned %s version: empty version", webAssetsModulePath)
	}
	return version, nil
}

func webAssetsPublicCheck(ctx context.Context, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return errors.New("web assets module version is required")
	}
	tmpDir, err := os.MkdirTemp("", "agh-web-assets-public-check-")
	if err != nil {
		return fmt.Errorf("create web assets public check dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	env := webAssetsPublicModuleEnv(tmpDir)
	moduleVersion := webAssetsModulePath + "@" + version
	if err := runCommandInDirWithEnv(ctx, tmpDir, env, "go", "list", "-m", moduleVersion); err != nil {
		return fmt.Errorf("public resolve %s: %w", moduleVersion, err)
	}
	return nil
}

func webAssetsPublicModuleEnv(tmpDir string) map[string]string {
	env := map[string]string{
		"GO111MODULE": "on",
		"GOFLAGS":     "-mod=mod",
		"GONOPROXY":   "",
		"GONOSUMDB":   "",
		"GOPRIVATE":   "",
		"GOPROXY":     "https://proxy.golang.org,direct",
		"GOSUMDB":     "sum.golang.org",
	}
	if tmpDir != "" {
		env["GOMODCACHE"] = filepath.Join(tmpDir, "mod")
		env["GOPATH"] = filepath.Join(tmpDir, "gopath")
	}
	return env
}

// DaytonaSidecars regenerates embedded Linux launcher sidecar assets.
func DaytonaSidecars() error {
	for _, asset := range daytonaSidecarAssets {
		if err := buildDaytonaSidecarAsset(context.Background(), asset, asset.path); err != nil {
			return err
		}
	}
	return nil
}

// DaytonaSidecarsCheck verifies embedded launcher sidecar assets are current.
func DaytonaSidecarsCheck() error {
	tmpDir, err := os.MkdirTemp("", "agh-daytona-sidecar-check-")
	if err != nil {
		return fmt.Errorf("create Daytona sidecar check dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, asset := range daytonaSidecarAssets {
		tmpAssetPath := filepath.Join(tmpDir, filepath.Base(asset.path))
		if err := buildDaytonaSidecarAsset(context.Background(), asset, tmpAssetPath); err != nil {
			return err
		}
		generated, err := os.ReadFile(tmpAssetPath)
		if err != nil {
			return fmt.Errorf("read generated Daytona sidecar asset %q: %w", tmpAssetPath, err)
		}
		current, err := os.ReadFile(asset.path)
		if err != nil {
			return fmt.Errorf(
				"read Daytona sidecar asset %q: %w; run %s",
				asset.path,
				err,
				daytonaSidecarRegenHint,
			)
		}
		if !bytes.Equal(generated, current) {
			return fmt.Errorf(
				"Daytona sidecar asset %q is stale; run %s",
				asset.path,
				daytonaSidecarRegenHint,
			)
		}
	}
	return nil
}

func buildDaytonaSidecarAsset(ctx context.Context, asset daytonaSidecarAsset, outputPath string) error {
	tmpDir, err := os.MkdirTemp("", "agh-daytona-sidecar-build-")
	if err != nil {
		return fmt.Errorf("create Daytona sidecar build dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath := filepath.Join(tmpDir, "agh-daytona-sidecar")
	if err := runCommandInDirWithEnv(
		ctx,
		".",
		map[string]string{
			"CGO_ENABLED":  "0",
			"GODEBUG":      "",
			"GOENV":        "off",
			"GOEXPERIMENT": "",
			"GOFLAGS":      "",
			"GOAMD64":      "v1",
			"GOARM64":      "v8.0",
			"GOOS":         "linux",
			"GOARCH":       asset.arch,
			"GOTOOLCHAIN":  "go" + daytonaSidecarToolchain,
		},
		"go",
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-ldflags",
		"-s -w -buildid=",
		"-o",
		binaryPath,
		daytonaSidecarPackage,
	); err != nil {
		return fmt.Errorf("build Daytona launcher sidecar for linux/%s: %w", asset.arch, err)
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read Daytona launcher sidecar for linux/%s: %w", asset.arch, err)
	}
	if len(binary) == 0 {
		return fmt.Errorf("Daytona launcher sidecar for linux/%s is empty", asset.arch)
	}
	if err := writeGzipAsset(outputPath, binary); err != nil {
		return fmt.Errorf("write Daytona sidecar asset %q: %w", outputPath, err)
	}
	return nil
}

func writeGzipAsset(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create gzip asset dir %q: %w", filepath.Dir(path), err)
	}
	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		closeErr := writer.Close()
		return errors.Join(fmt.Errorf("write gzip payload: %w", err), closeErr)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}
	return os.WriteFile(path, compressed.Bytes(), 0o644)
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
func SourceInstallCheck() error {
	tmpRoot, err := os.MkdirTemp("", "agh-source-install-check-")
	if err != nil {
		return fmt.Errorf("create source install check dir: %w", err)
	}
	defer os.RemoveAll(tmpRoot)

	sourceDir := filepath.Join(tmpRoot, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return fmt.Errorf("create source install check source dir: %w", err)
	}
	if err := copySourceInstallFiles(sourceDir); err != nil {
		return err
	}

	binDir := filepath.Join(tmpRoot, "bin")
	env := map[string]string{
		"CGO_ENABLED":    "0",
		"GOBIN":          binDir,
		"GOMODCACHE":     filepath.Join(tmpRoot, "mod"),
		"GOPATH":         filepath.Join(tmpRoot, "gopath"),
		webDistDirEnvVar: "",
	}
	if err := runCommandInDirWithEnv(context.Background(), sourceDir, env, "go", "install", "."); err != nil {
		return fmt.Errorf("source install go install .: %w", err)
	}
	binary := filepath.Join(binDir, cliBinary)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := runCommandInDirWithEnv(context.Background(), sourceDir, env, binary, "version"); err != nil {
		return fmt.Errorf("source install agh version: %w", err)
	}
	if err := runCommandInDirWithEnv(
		context.Background(),
		sourceDir,
		env,
		"go",
		"test",
		"./internal/api/httpapi",
		"-run",
		"TestStaticRoutesServeEmbedded(IndexForRootAndDeepLinks|Assets)$",
		"-count=1",
	); err != nil {
		return fmt.Errorf("source install embedded web asset test: %w", err)
	}
	return nil
}

func copySourceInstallFiles(destRoot string) error {
	files, err := gitSourceInstallFiles()
	if err != nil {
		return err
	}
	for _, rel := range files {
		info, err := os.Lstat(rel)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("stat source install file %q: %w", rel, err)
		}
		if info.IsDir() {
			continue
		}
		dest := filepath.Join(destRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("create source install parent for %q: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(rel)
			if err != nil {
				return fmt.Errorf("read source install symlink %q: %w", rel, err)
			}
			if err := os.Symlink(target, dest); err != nil {
				return fmt.Errorf("write source install symlink %q: %w", rel, err)
			}
			continue
		}
		data, err := os.ReadFile(rel)
		if err != nil {
			return fmt.Errorf("read source install file %q: %w", rel, err)
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(dest, data, mode); err != nil {
			return fmt.Errorf("write source install file %q: %w", rel, err)
		}
	}
	return nil
}

func gitSourceInstallFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "-c", "-o", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list source install files: %w", err)
	}
	parts := bytes.Split(out, []byte{0})
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		files = append(files, filepath.ToSlash(string(part)))
	}
	return files, nil
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
		{"internal/events", "internal/daemon"},
		{"internal/doctor", "internal/daemon"},
		{"internal/providers", "internal/daemon"},
		{"internal/diagnosticcontract", "internal/daemon"},
		{"internal/config", "internal/api/httpapi"},
		{"internal/acp", "internal/api/httpapi"},
		{"internal/session", "internal/api/httpapi"},
		{"internal/store", "internal/api/httpapi"},
		{"internal/observe", "internal/api/httpapi"},
		{"internal/events", "internal/api/httpapi"},
		{"internal/doctor", "internal/api/httpapi"},
		{"internal/providers", "internal/api/httpapi"},
		{"internal/diagnosticcontract", "internal/api/httpapi"},
		{"internal/config", "internal/api/udsapi"},
		{"internal/acp", "internal/api/udsapi"},
		{"internal/session", "internal/api/udsapi"},
		{"internal/store", "internal/api/udsapi"},
		{"internal/observe", "internal/api/udsapi"},
		{"internal/events", "internal/api/udsapi"},
		{"internal/doctor", "internal/api/udsapi"},
		{"internal/providers", "internal/api/udsapi"},
		{"internal/diagnosticcontract", "internal/api/udsapi"},
		{"internal/config", "internal/cli"},
		{"internal/acp", "internal/cli"},
		{"internal/session", "internal/cli"},
		{"internal/store", "internal/cli"},
		{"internal/observe", "internal/cli"},
		{"internal/events", "internal/cli"},
		{"internal/doctor", "internal/cli"},
		{"internal/providers", "internal/cli"},
		{"internal/diagnosticcontract", "internal/cli"},
		{"internal/providers", "internal/session"},
		{"internal/providers", "internal/acp"},
		{"internal/providers", "internal/api/core"},
		{"internal/api/contract", "internal/daemon"},
		{"internal/api/contract", "internal/api/httpapi"},
		{"internal/api/contract", "internal/api/udsapi"},
		{"internal/api/contract", "internal/cli"},
		{"internal/diagnosticcontract", "internal/api/contract"},
		{"internal/diagnosticcontract", "internal/api/core"},
		{"internal/events", "internal/api/contract"},
		{"internal/events", "internal/api/core"},
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
		importPath := "github.com/compozy/agh/" + rule.imported
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
	return runMageSteps(verifySteps())
}

func verifySteps() []mageStep {
	return []mageStep{
		{name: "CodegenCheck", run: CodegenCheck},
		{name: "InstallerCheck", run: InstallerCheck},
		{name: "BunLint", run: BunLint},
		{name: "BunTypecheck", run: BunTypecheck},
		{name: "BunTest", run: BunTest},
		{name: "WebBuild", run: WebBuild},
		{name: "Fmt", run: Fmt},
		{name: "Lint", run: Lint},
		{name: "Test", run: Test},
		{name: "buildGo", run: buildGo},
		{name: "Boundaries", run: Boundaries},
	}
}

func runMageSteps(steps []mageStep) error {
	for _, step := range steps {
		if err := step.run(); err != nil {
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

func ensureWebAssets() error {
	if _, err := os.Stat(webDistIndex); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := CodegenCheck(); err != nil {
			return err
		}
		if err := WebBuild(); err != nil {
			return err
		}
	}
	return ensureWebDist()
}

func runE2ELane(lane e2elane.Lane) (runErr error) {
	ctx := context.Background()

	plan, err := e2elane.PlanForLane(lane)
	if err != nil {
		return err
	}

	if shouldEnsureWebBundle(plan) {
		if err := ensureWebAssets(); err != nil {
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

	values := map[string]string{
		daemonBinaryEnvVar: daemonPath,
		driverBinaryEnvVar: driverPath,
	}
	if _, err := os.Stat(webDistIndex); err == nil {
		absWebDistDir, absErr := filepath.Abs(webDistDir)
		if absErr != nil {
			return e2eLaneEnv{}, errors.Join(
				fmt.Errorf("resolve %s for e2e lane: %w", webDistDir, absErr),
				runCleanups(cleanups),
			)
		}
		values[webDistDirEnvVar] = absWebDistDir
	} else if !errors.Is(err, os.ErrNotExist) {
		return e2eLaneEnv{}, errors.Join(err, runCleanups(cleanups))
	}

	return e2eLaneEnv{
		Values: values,
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
