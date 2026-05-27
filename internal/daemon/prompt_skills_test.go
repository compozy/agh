package daemon

import (
	"context"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/session"
	skillspkg "github.com/compozy/agh/internal/skills"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestNewSkillsCatalogAugmenterUsesCurrentRegistryStatePerPrompt(t *testing.T) {
	t.Run("Should compact unchanged catalog after the first prompt", func(t *testing.T) {
		t.Parallel()

		registry, augmenter := newPromptSkillsAugmenterForTest(t, []*skillspkg.Skill{
			{
				Meta: skillspkg.SkillMeta{
					Name:        "qa-marker-skill",
					Description: "Shows up while enabled.",
				},
				Enabled: true,
			},
		})
		_ = registry
		sess := newPromptSkillsSession("sess-compact")

		first, err := augmenter(context.Background(), sess, "list current skills")
		if err != nil {
			t.Fatalf("augmenter(first) error = %v", err)
		}
		if !strings.Contains(first, `name="qa-marker-skill"`) {
			t.Fatalf("first prompt = %q, want enabled skill entry", first)
		}

		second, err := augmenter(context.Background(), sess, "list current skills again")
		if err != nil {
			t.Fatalf("augmenter(second) error = %v", err)
		}
		if !strings.Contains(second, `<catalog-state unchanged="true">`) {
			t.Fatalf("second prompt = %q, want unchanged catalog marker", second)
		}
		if strings.Contains(second, `name="qa-marker-skill"`) {
			t.Fatalf("second prompt = %q, want compact marker without repeated skill entries", second)
		}
		if !strings.Contains(second, "use `agh__skill_view` for full skill/resource instructions") {
			t.Fatalf("second prompt = %q, want compact skill_view guidance", second)
		}
		if !strings.Contains(second, "use `agh skill view <name>` as an operator fallback") {
			t.Fatalf("second prompt = %q, want operator fallback guidance", second)
		}
		if !strings.HasSuffix(second, "list current skills again") {
			t.Fatalf("second prompt = %q, want original prompt preserved", second)
		}
	})

	t.Run("Should emit refreshed full catalog when registry state changes", func(t *testing.T) {
		t.Parallel()

		registry, augmenter := newPromptSkillsAugmenterForTest(t, []*skillspkg.Skill{
			{
				Meta: skillspkg.SkillMeta{
					Name:        "qa-marker-skill",
					Description: "Shows up while enabled.",
				},
				Enabled: true,
			},
		})
		sess := newPromptSkillsSession("sess-refresh")

		first, err := augmenter(context.Background(), sess, "list current skills")
		if err != nil {
			t.Fatalf("augmenter(first) error = %v", err)
		}
		if !strings.Contains(first, `name="qa-marker-skill"`) {
			t.Fatalf("first prompt = %q, want enabled skill entry", first)
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
		if strings.Contains(second, `<catalog-state unchanged="true">`) {
			t.Fatalf("second prompt = %q, want full refreshed catalog instead of unchanged marker", second)
		}
		if !strings.HasSuffix(second, "list current skills") {
			t.Fatalf("second prompt = %q, want original prompt preserved", second)
		}
	})
}

func BenchmarkSkillsCatalogAugmenterCatalogReplayModes(b *testing.B) {
	registry, augmenter := newPromptSkillsAugmenterForTest(b, []*skillspkg.Skill{
		{
			Meta: skillspkg.SkillMeta{
				Name:        "qa-marker-skill",
				Description: "Shows up while enabled.",
			},
			Enabled: true,
		},
	})
	_ = registry
	sess := newPromptSkillsSession("sess-bench")

	full, err := augmenter(context.Background(), sess, "network note")
	if err != nil {
		b.Fatalf("augmenter(full) error = %v", err)
	}
	compact, err := augmenter(context.Background(), sess, "network note")
	if err != nil {
		b.Fatalf("augmenter(compact) error = %v", err)
	}
	fullBytes := len(full)
	compactBytes := len(compact)
	promptSuffixBytes := len("\n\nnetwork note")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := augmenter(context.Background(), sess, "network note"); err != nil {
			b.Fatalf("augmenter(repeated) error = %v", err)
		}
	}
	b.ReportMetric(float64(fullBytes-promptSuffixBytes), "full-catalog-bytes/op")
	b.ReportMetric(float64(compactBytes-promptSuffixBytes), "compact-catalog-bytes/op")
	b.ReportMetric(float64(fullBytes-compactBytes), "saved-catalog-bytes/op")
	b.ReportMetric(float64(fullBytes), "full-bytes/op")
	b.ReportMetric(float64(compactBytes), "compact-bytes/op")
	b.ReportMetric(float64(fullBytes-compactBytes), "saved-prompt-bytes/op")
}

func newPromptSkillsAugmenterForTest(
	t testing.TB,
	skills []*skillspkg.Skill,
) (*stubPromptSkillsRegistry, session.PromptInputAugmenter) {
	t.Helper()

	registry := &stubPromptSkillsRegistry{
		skillsByAgent: map[string][]*skillspkg.Skill{
			"general": skills,
		},
	}
	resolver := &stubPromptSkillsWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-1",
				RootDir: "/tmp/ws-1",
			},
		},
	}
	augmenter := newSkillsCatalogAugmenter(registry, func() promptSkillsWorkspaceResolver {
		return resolver
	})
	if augmenter == nil {
		t.Fatal("newSkillsCatalogAugmenter() = nil, want augmenter")
	}
	return registry, augmenter
}

func newPromptSkillsSession(sessionID string) *session.Session {
	return &session.Session{
		ID:          sessionID,
		AgentName:   "general",
		WorkspaceID: "ws-1",
		Workspace:   "/tmp/ws-1",
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
