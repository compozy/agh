package skills

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestBuildCatalogReturnsEmptyStringWhenNoSkills(t *testing.T) {
	t.Parallel()

	testCases := map[string][]*Skill{
		"nil":   nil,
		"empty": {},
	}

	for name, skills := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := BuildCatalog(skills); got != "" {
				t.Fatalf("BuildCatalog() = %q, want empty string", got)
			}
		})
	}
}

func TestBuildCatalogFormatsCatalogSortedEscapedAndWithUsageInstructions(t *testing.T) {
	t.Parallel()

	skills := []*Skill{
		{
			Meta: SkillMeta{
				Name:        "zeta",
				Description: "Last skill",
			},
			Enabled: true,
		},
		{
			Meta: SkillMeta{
				Name:        `alpha"<&>`,
				Description: `Use < & > and "quotes" safely`,
			},
			Enabled: true,
		},
	}

	got := BuildCatalog(skills)

	want := strings.Join([]string{
		"<available-skills>",
		`  <skill name="alpha&quot;&lt;&amp;&gt;">Use &lt; &amp; &gt; and "quotes" safely</skill>`,
		`  <skill name="zeta">Last skill</skill>`,
		"</available-skills>",
		"",
		"Use `agh skill view <name>` to load full instructions for any skill.",
		"Use `agh skill view <name> --file <path>` to read a specific skill resource file.",
	}, "\n")

	if got != want {
		t.Fatalf("BuildCatalog() mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestBuildCatalogTruncatesDescriptionsAtTwoHundredCharactersWithEllipsis(t *testing.T) {
	t.Parallel()

	description := strings.Repeat("a", catalogDescriptionLimit+5)
	got := BuildCatalog([]*Skill{
		{
			Meta: SkillMeta{
				Name:        "long",
				Description: description,
			},
			Enabled: true,
		},
	})

	wantDescription := strings.Repeat("a", catalogDescriptionLimit-len(catalogEllipsis)) + catalogEllipsis
	wantLine := `  <skill name="long">` + wantDescription + `</skill>`

	if !strings.Contains(got, wantLine) {
		t.Fatalf("BuildCatalog() missing truncated line %q in %q", wantLine, got)
	}

	if utf8.RuneCountInString(wantDescription) != catalogDescriptionLimit {
		t.Fatalf("truncated description rune count = %d, want %d", utf8.RuneCountInString(wantDescription), catalogDescriptionLimit)
	}
}

func TestBuildCatalogExcludesDisabledSkills(t *testing.T) {
	t.Parallel()

	got := BuildCatalog([]*Skill{
		{
			Meta: SkillMeta{
				Name:        "enabled",
				Description: "Visible skill",
			},
			Enabled: true,
		},
		{
			Meta: SkillMeta{
				Name:        "disabled",
				Description: "Hidden skill",
			},
			Enabled: false,
		},
	})

	if strings.Contains(got, `name="disabled"`) {
		t.Fatalf("BuildCatalog() included disabled skill: %q", got)
	}
	if !strings.Contains(got, `name="enabled"`) {
		t.Fatalf("BuildCatalog() missing enabled skill: %q", got)
	}
}

func TestCatalogProviderPromptSectionReturnsEmptyStringWhenWorkspaceHasNoSkills(t *testing.T) {
	t.Parallel()

	provider := NewCatalogProvider(newTestRegistry(t, RegistryConfig{}))

	got, err := provider.PromptSection(context.Background(), workspacepkg.ResolvedWorkspace{})
	if err != nil {
		t.Fatalf("PromptSection() error = %v", err)
	}
	if got != "" {
		t.Fatalf("PromptSection() = %q, want empty string", got)
	}
}

func TestCatalogProviderPromptSectionUsesWorkspaceScopedSkills(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	userDir := filepath.Join(root, "user")
	workspaceOne := filepath.Join(root, "workspace-one")
	workspaceTwo := filepath.Join(root, "workspace-two")

	writeSkillFile(t, userDir, filepath.Join("global", skillFileName), skillWithDescription("global", "Global skill"))
	writeSkillFile(t, filepath.Join(workspaceOne, ".agh", "skills"), filepath.Join("alpha", skillFileName), skillWithDescription("alpha", "Workspace one skill"))
	writeSkillFile(t, filepath.Join(workspaceTwo, ".agh", "skills"), filepath.Join("beta", skillFileName), skillWithDescription("beta", "Workspace two skill"))

	registry := newTestRegistry(t, RegistryConfig{
		UserSkillsDir: userDir,
	})
	if err := registry.LoadAll(context.Background()); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	provider := NewCatalogProvider(registry)

	got, err := provider.PromptSection(context.Background(), resolvedWorkspaceForTest("ws_catalog_one", workspaceOne,
		resolvedSkillPath(filepath.Join(workspaceOne, ".agh", "skills", "alpha"), "workspace"),
	))
	if err != nil {
		t.Fatalf("PromptSection() error = %v", err)
	}

	want := strings.Join([]string{
		"<available-skills>",
		`  <skill name="alpha">Workspace one skill</skill>`,
		`  <skill name="global">Global skill</skill>`,
		"</available-skills>",
		"",
		"Use `agh skill view <name>` to load full instructions for any skill.",
		"Use `agh skill view <name> --file <path>` to read a specific skill resource file.",
	}, "\n")

	if got != want {
		t.Fatalf("PromptSection() mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}

	if strings.Contains(got, "beta") {
		t.Fatalf("PromptSection() leaked workspace-two skill into workspace-one catalog: %q", got)
	}
}
