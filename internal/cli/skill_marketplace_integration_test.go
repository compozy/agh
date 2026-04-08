//go:build integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestSkillInstallCommandIntegrationCreatesSkillDirectoryAndSidecar(t *testing.T) {
	t.Parallel()

	server := newMarketplaceTestServer(t, marketplaceServerFixture{
		downloads: map[string]marketplaceDownloadFixture{
			"@agh/review": {
				version: "1.2.0",
				files: map[string]string{
					"review/SKILL.md": skillDocument("review", "Review helper", "body"),
				},
			},
		},
	})
	defer server.Close()

	env := newSkillTestEnv(t, func(cfg *aghconfig.Config) {
		cfg.Skills.Marketplace = aghconfig.MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  server.URL(),
		}
	})

	stdout, _, err := executeRootCommand(t, env.deps, "skill", "install", "@agh/review", "-o", "json")
	if err != nil {
		t.Fatalf("skill install error = %v", err)
	}

	var payload skillInstallItem
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(skill install) error = %v; stdout=%s", err, stdout)
	}
	if payload.Status != "installed" {
		t.Fatalf("skill install payload = %#v, want installed status", payload)
	}

	skillDir := filepath.Join(env.homePaths.SkillsDir, "review")
	if _, err := os.Stat(filepath.Join(skillDir, skillMarkdownFileName)); err != nil {
		t.Fatalf("installed SKILL.md stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(skillDir, ".agh-meta.json")); err != nil {
		t.Fatalf("installed sidecar stat error = %v", err)
	}
}

func TestSkillInstallAndRemoveIntegrationRefreshesRegistry(t *testing.T) {
	t.Parallel()

	server := newMarketplaceTestServer(t, marketplaceServerFixture{
		downloads: map[string]marketplaceDownloadFixture{
			"@agh/cleanup": {
				version: "1.0.0",
				files: map[string]string{
					"cleanup/SKILL.md": skillDocument("cleanup", "Cleanup helper", "body"),
				},
			},
		},
	})
	defer server.Close()

	env := newSkillTestEnv(t, func(cfg *aghconfig.Config) {
		cfg.Skills.Marketplace = aghconfig.MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  server.URL(),
		}
	})

	if _, _, err := executeRootCommand(t, env.deps, "skill", "install", "@agh/cleanup", "-o", "json"); err != nil {
		t.Fatalf("skill install error = %v", err)
	}

	registry := skills.NewRegistry(skills.RegistryConfig{
		BundledFS:     skillbundled.FS(),
		UserSkillsDir: env.homePaths.SkillsDir,
		UserAgentsDir: filepath.Join(env.userHome, ".agents", "skills"),
	})
	if err := registry.LoadAll(testutil.Context(t)); err != nil {
		t.Fatalf("registry.LoadAll() error = %v", err)
	}

	installed, ok := registry.Get("cleanup")
	if !ok {
		t.Fatal("registry.Get(cleanup) ok = false, want installed marketplace skill")
	}
	if installed.Source != skills.SourceMarketplace {
		t.Fatalf("installed.Source = %v, want marketplace", installed.Source)
	}

	if _, _, err := executeRootCommand(t, env.deps, "skill", "remove", "cleanup", "-o", "json"); err != nil {
		t.Fatalf("skill remove error = %v", err)
	}
	if err := registry.RefreshGlobal(testutil.Context(t)); err != nil {
		t.Fatalf("registry.RefreshGlobal() error = %v", err)
	}
	if _, ok := registry.Get("cleanup"); ok {
		t.Fatal("registry.Get(cleanup) ok = true after remove, want missing skill")
	}
}

func TestSkillInstallCommandIntegrationWritesMatchingHash(t *testing.T) {
	t.Parallel()

	server := newMarketplaceTestServer(t, marketplaceServerFixture{
		downloads: map[string]marketplaceDownloadFixture{
			"@agh/hashcheck": {
				version: "2.0.0",
				files: map[string]string{
					"hashcheck/SKILL.md": skillDocument("hashcheck", "Hash helper", "body"),
				},
			},
		},
	})
	defer server.Close()

	env := newSkillTestEnv(t, func(cfg *aghconfig.Config) {
		cfg.Skills.Marketplace = aghconfig.MarketplaceConfig{
			Registry: "clawhub",
			BaseURL:  server.URL(),
		}
	})

	if _, _, err := executeRootCommand(t, env.deps, "skill", "install", "@agh/hashcheck", "-o", "json"); err != nil {
		t.Fatalf("skill install error = %v", err)
	}

	skillDir := filepath.Join(env.homePaths.SkillsDir, "hashcheck")
	skillBytes, err := os.ReadFile(filepath.Join(skillDir, skillMarkdownFileName))
	if err != nil {
		t.Fatalf("ReadFile(SKILL.md) error = %v", err)
	}
	provenance, err := skills.ReadSidecar(skillDir)
	if err != nil {
		t.Fatalf("ReadSidecar() error = %v", err)
	}
	if provenance == nil {
		t.Fatal("ReadSidecar() = nil, want provenance")
	}

	if got, want := provenance.Hash, skills.ComputeHash(skillBytes); got != want {
		t.Fatalf("provenance.Hash = %q, want %q", got, want)
	}
}
