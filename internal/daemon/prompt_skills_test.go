package daemon

import (
	"context"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestNewSkillsCatalogAugmenterUsesCurrentRegistryStatePerPrompt(t *testing.T) {
	t.Parallel()

	registry := &stubPromptSkillsRegistry{
		skillsByAgent: map[string][]*skillspkg.Skill{
			"general": {
				{
					Meta: skillspkg.SkillMeta{
						Name:        "qa-marker-skill",
						Description: "Shows up while enabled.",
					},
					Enabled: true,
				},
			},
		},
	}
	var resolver promptSkillsWorkspaceResolver
	augmenter := newSkillsCatalogAugmenter(registry, func() promptSkillsWorkspaceResolver {
		return resolver
	})
	if augmenter == nil {
		t.Fatal("newSkillsCatalogAugmenter() = nil, want augmenter")
	}

	resolver = &stubPromptSkillsWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: "/tmp/ws-1",
			},
		},
	}

	sess := &session.Session{
		ID:          "sess-1",
		AgentName:   "general",
		WorkspaceID: "ws-1",
		Workspace:   "/tmp/ws-1",
	}

	first, err := augmenter(context.Background(), sess, "list current skills")
	if err != nil {
		t.Fatalf("augmenter(first) error = %v", err)
	}
	if !strings.Contains(first, "<current-available-skills>") {
		t.Fatalf("first prompt = %q, want current catalog block", first)
	}
	if !strings.Contains(first, `name="qa-marker-skill"`) {
		t.Fatalf("first prompt = %q, want enabled skill entry", first)
	}
	if !strings.HasSuffix(first, "list current skills") {
		t.Fatalf("first prompt = %q, want original prompt preserved", first)
	}

	registry.skillsByAgent["general"] = []*skillspkg.Skill{
		{
			Meta: skillspkg.SkillMeta{
				Name:        "qa-marker-skill",
				Description: "Now disabled.",
			},
			Enabled: false,
		},
		{
			Meta: skillspkg.SkillMeta{
				Name:        "replacement-skill",
				Description: "Still enabled after refresh.",
			},
			Enabled: true,
		},
	}
	resolver = &stubPromptSkillsWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: "/tmp/ws-1",
			},
		},
	}

	second, err := augmenter(context.Background(), sess, "list current skills")
	if err != nil {
		t.Fatalf("augmenter(second) error = %v, want live refresh", err)
	}
	if strings.Contains(second, `name="qa-marker-skill"`) {
		t.Fatalf("second prompt = %q, want disabled skill removed from current catalog", second)
	}
	if !strings.Contains(second, `name="replacement-skill"`) {
		t.Fatalf("second prompt = %q, want refreshed enabled skill in current catalog", second)
	}
	if !strings.Contains(
		second,
		"The <current-available-skills> block above is the authoritative current skill state for this turn.",
	) {
		t.Fatalf("second prompt = %q, want authoritative current-catalog guidance", second)
	}
	if !strings.HasSuffix(second, "list current skills") {
		t.Fatalf("second prompt = %q, want original prompt preserved", second)
	}
}

type stubPromptSkillsRegistry struct {
	skillsByAgent map[string][]*skillspkg.Skill
}

func (s *stubPromptSkillsRegistry) ForWorkspace(
	_ context.Context,
	_ *workspacepkg.ResolvedWorkspace,
) ([]*skillspkg.Skill, error) {
	return nil, nil
}

func (s *stubPromptSkillsRegistry) ForAgent(
	_ context.Context,
	_ *workspacepkg.ResolvedWorkspace,
	agentName string,
) ([]*skillspkg.Skill, error) {
	if s == nil {
		return nil, nil
	}
	return append([]*skillspkg.Skill(nil), s.skillsByAgent[agentName]...), nil
}

type stubPromptSkillsWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
}

func (s *stubPromptSkillsWorkspaceResolver) Resolve(
	_ context.Context,
	_ string,
) (workspacepkg.ResolvedWorkspace, error) {
	if s == nil {
		return workspacepkg.ResolvedWorkspace{}, nil
	}
	return s.resolved, nil
}
