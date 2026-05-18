package config

import (
	"os"
	"path/filepath"
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
}

func TestPRReleaseConfigGeneratesSiteChangelogArtifact(t *testing.T) {
	t.Parallel()

	root := findRepoRootForReleaseConfigTest(t)
	data, err := os.ReadFile(filepath.Join(root, ".pr-release"))
	if err != nil {
		t.Fatalf("os.ReadFile(.pr-release) error = %v", err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal(.pr-release) error = %v", err)
	}

	artifacts := sliceAt(t, cfg, "release_artifacts")
	if len(artifacts) != 1 {
		t.Fatalf("release_artifacts len = %d, want 1", len(artifacts))
	}
	artifact := asMap(t, artifacts[0], "release_artifacts[0]")
	if got, want := stringAt(t, artifact, "name"), "site-changelog"; got != want {
		t.Fatalf("release_artifacts[0].name = %q, want %q", got, want)
	}
	if got, want := stringAt(t, artifact, "command"), "bun"; got != want {
		t.Fatalf("release_artifacts[0].command = %q, want %q", got, want)
	}
	if !stringSliceContains(sliceAt(t, artifact, "args"), "release:site-changelog") {
		t.Fatalf("release_artifacts[0].args = %#v, want release:site-changelog", artifact["args"])
	}
	if !stringSliceContains(sliceAt(t, artifact, "add"), "packages/site/content/blog/changelog/*.mdx") {
		t.Fatalf("release_artifacts[0].add = %#v, want site changelog glob", artifact["add"])
	}
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
