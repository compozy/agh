package daemon

import (
	"context"
	"errors"
	"os"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestResourceAgentCatalogListsGetsAndResolvesByScope(t *testing.T) {
	t.Parallel()

	catalog := newResourceCatalog(cloneAgentDef)
	catalog.Replace(3, []resources.Record[aghconfig.AgentDef]{
		{
			ID:    "global:alpha",
			Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			Spec:  aghconfig.AgentDef{Name: "alpha", Prompt: "global alpha"},
		},
		{
			ID:    "global:coder",
			Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			Spec:  aghconfig.AgentDef{Name: "coder", Prompt: "global coder"},
		},
		{
			ID:    "workspace:coder",
			Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws-1"},
			Spec:  aghconfig.AgentDef{Name: "coder", Prompt: "workspace coder", Tools: []string{"lookup"}},
		},
	})

	dependency := agentCatalogDependency(catalog)
	listed, err := dependency.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}
	if got, want := len(listed), 2; got != want {
		t.Fatalf("len(ListAgents()) = %d, want %d", got, want)
	}
	if listed[0].Name != "alpha" || listed[1].Name != "coder" {
		t.Fatalf("ListAgents() = %#v, want global agents sorted by name", listed)
	}

	got, err := dependency.GetAgent(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("GetAgent(alpha) error = %v", err)
	}
	if got.Prompt != "global alpha" {
		t.Fatalf("GetAgent(alpha).Prompt = %q, want global alpha", got.Prompt)
	}
	if _, err := dependency.GetAgent(context.Background(), "missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("GetAgent(missing) error = %v, want os.ErrNotExist", err)
	}

	resolved := &workspacepkg.ResolvedWorkspace{Workspace: workspacepkg.Workspace{ID: "ws-1"}}
	coder, err := dependency.ResolveAgent("coder", resolved)
	if err != nil {
		t.Fatalf("ResolveAgent(coder) error = %v", err)
	}
	if coder.Prompt != "workspace coder" || len(coder.Tools) != 1 || coder.Tools[0] != "lookup" {
		t.Fatalf("ResolveAgent(coder) = %#v, want workspace override", coder)
	}
}

func TestResourceAgentCatalogFallsBackToResolvedWorkspaceSnapshot(t *testing.T) {
	t.Parallel()

	resolved := &workspacepkg.ResolvedWorkspace{
		Agents: []aghconfig.AgentDef{{
			Name:   "fallback",
			Prompt: "resolved snapshot",
		}},
	}
	got, err := (&resourceAgentCatalog{}).ResolveAgent("fallback", resolved)
	if err != nil {
		t.Fatalf("ResolveAgent(fallback) error = %v", err)
	}
	if got.Prompt != "resolved snapshot" {
		t.Fatalf("ResolveAgent(fallback).Prompt = %q, want resolved snapshot", got.Prompt)
	}
	if _, err := (&resourceAgentCatalog{}).ResolveAgent(
		"missing",
		resolved,
	); !errors.Is(
		err,
		workspacepkg.ErrAgentNotAvailable,
	) {
		t.Fatalf("ResolveAgent(missing) error = %v, want ErrAgentNotAvailable", err)
	}
}

func TestAgentSkillSmallHelpers(t *testing.T) {
	t.Parallel()

	var nilPublisher agentSkillPublisherFunc
	if err := nilPublisher.Sync(context.Background()); err != nil {
		t.Fatalf("nil publisher Sync() error = %v", err)
	}
	called := false
	publisher := agentSkillPublisherFunc(func(context.Context) error {
		called = true
		return nil
	})
	if err := publisher.Sync(context.Background()); err != nil {
		t.Fatalf("publisher Sync() error = %v", err)
	}
	if !called {
		t.Fatal("publisher Sync() did not call function")
	}
	if agentCatalogDependency(nil) != nil {
		t.Fatal("agentCatalogDependency(nil) != nil")
	}
	if newAgentProjector(nil) != nil {
		t.Fatal("newAgentProjector(nil) != nil")
	}
	if newSkillProjector(nil) != nil {
		t.Fatal("newSkillProjector(nil) != nil")
	}

	var nilPlan *skillResourceProjectionPlan
	if nilPlan.Kind() != "" || nilPlan.Revision() != 0 || nilPlan.OperationCount() != 0 {
		t.Fatalf(
			"nil skillResourceProjectionPlan methods = (%q,%d,%d), want zero values",
			nilPlan.Kind(),
			nilPlan.Revision(),
			nilPlan.OperationCount(),
		)
	}
	plan := &skillResourceProjectionPlan{
		revision: 4,
		records: []resources.Record[skillspkg.SkillResourceSpec]{{
			ID: "skill",
		}},
	}
	if plan.Kind() != skillspkg.SkillResourceKind || plan.Revision() != 4 || plan.OperationCount() != 1 {
		t.Fatalf(
			"skillResourceProjectionPlan methods = (%q,%d,%d), want skill,4,1",
			plan.Kind(),
			plan.Revision(),
			plan.OperationCount(),
		)
	}
}

func TestAgentSkillSourceSyncerReplacesCanonicalSnapshot(t *testing.T) {
	t.Parallel()

	agentStore, agentCodec, skillStore, skillCodec, mcpStore, mcpCodec := agentSkillSyncStores(t)
	desired := agentSkillDesiredResources{
		agents: []agentPublicationInput{{
			sourceKey: "test/agent/coder",
			scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			spec: aghconfig.AgentDef{
				Name:   "coder",
				Prompt: "Use canonical tools.",
				Tools:  []string{"lookup"},
			},
		}},
		skills: []skillPublicationInput{{
			sourceKey: "test/skill/review",
			scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			spec: skillspkg.SkillResourceSpec{
				Name:        "review",
				Description: "Review skill",
				Source:      skillspkg.SkillSourceName(skillspkg.SourceUser),
				Enabled:     true,
			},
		}},
		mcpServers: []mcpServerPublicationInput{{
			sourceKey: "test/mcp/review",
			scope:     resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
			spec: aghconfig.MCPServer{
				Name:    "review-mcp",
				Command: "review-command",
			},
		}},
	}
	triggered := make(map[resources.ResourceKind]int)
	syncer := newAgentSkillSourceSyncer(
		agentStore,
		agentCodec,
		skillStore,
		skillCodec,
		mcpStore,
		mcpCodec,
		agentSkillSyncActor(),
		discardLogger(),
		func(_ context.Context, kind resources.ResourceKind, _ resources.ReconcileReason) error {
			triggered[kind]++
			return nil
		},
		func(context.Context) (agentSkillDesiredResources, error) {
			return desired, nil
		},
	)

	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	assertAgentSkillStoreCounts(t, agentStore, skillStore, mcpStore, 1, 1, 1)
	if triggered[aghconfig.AgentResourceKind] != 1 ||
		triggered[skillspkg.SkillResourceKind] != 1 ||
		triggered[aghconfig.MCPServerResourceKind] != 1 {
		t.Fatalf("triggered = %#v, want one trigger per migrated kind", triggered)
	}

	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("second Sync() error = %v", err)
	}
	if triggered[aghconfig.AgentResourceKind] != 1 ||
		triggered[skillspkg.SkillResourceKind] != 1 ||
		triggered[aghconfig.MCPServerResourceKind] != 1 {
		t.Fatalf("triggered after no-op = %#v, want no additional triggers", triggered)
	}

	desired.agents = nil
	desired.mcpServers = nil
	desired.skills[0].spec.Description = "Updated review skill"
	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("third Sync() error = %v", err)
	}
	assertAgentSkillStoreCounts(t, agentStore, skillStore, mcpStore, 0, 1, 0)
	if triggered[aghconfig.AgentResourceKind] != 2 ||
		triggered[skillspkg.SkillResourceKind] != 2 ||
		triggered[aghconfig.MCPServerResourceKind] != 2 {
		t.Fatalf("triggered after replacement = %#v, want stale-delete/update triggers", triggered)
	}
}

func TestAppendAgentAndSkillResourcesPublishesMCPAttachments(t *testing.T) {
	t.Parallel()

	scope := resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws-append"}
	decl := hookspkg.HookDecl{
		Name:    "tool-hook",
		Event:   hookspkg.HookToolPreCall,
		Source:  hookspkg.HookSourceAgentDefinition,
		Mode:    hookspkg.HookModeSync,
		Command: "echo",
		Args:    []string{"ok"},
	}
	desired := agentSkillDesiredResources{}
	appendAgentResources(&desired, scope, "append", []aghconfig.AgentDef{{
		Name:   "coder",
		Prompt: "Prompt",
		MCPServers: []aghconfig.MCPServer{{
			Name:    "agent-mcp",
			Command: "agent-command",
		}},
		Hooks: []hookspkg.HookDecl{decl},
	}})
	appendSkillResources(&desired, scope, "append", []*skillspkg.Skill{{
		Meta: skillspkg.SkillMeta{
			Name:        "review",
			Description: "Review skill",
		},
		Source:  skillspkg.SourceWorkspace,
		Enabled: true,
		MCPServers: []skillspkg.MCPServerDecl{{
			Name:    "skill-mcp",
			Command: "skill-command",
		}},
		Hooks: []hookspkg.HookDecl{{
			Name:    "skill-hook",
			Event:   hookspkg.HookToolPreCall,
			Source:  hookspkg.HookSourceSkill,
			Mode:    hookspkg.HookModeSync,
			Command: "echo",
		}},
	}})

	if got, want := len(desired.agents), 1; got != want {
		t.Fatalf("len(agents) = %d, want %d", got, want)
	}
	if got, want := len(desired.skills), 1; got != want {
		t.Fatalf("len(skills) = %d, want %d", got, want)
	}
	if got, want := len(desired.mcpServers), 2; got != want {
		t.Fatalf("len(mcpServers) = %d, want %d", got, want)
	}
	if desired.mcpServers[0].spec.Name != "agent-mcp" || desired.mcpServers[1].spec.Name != "skill-mcp" {
		t.Fatalf("mcpServers = %#v, want agent and skill MCP attachments", desired.mcpServers)
	}
}

func agentSkillSyncStores(
	t *testing.T,
) (
	resources.Store[aghconfig.AgentDef],
	resources.KindCodec[aghconfig.AgentDef],
	resources.Store[skillspkg.SkillResourceSpec],
	resources.KindCodec[skillspkg.SkillResourceSpec],
	resources.Store[aghconfig.MCPServer],
	resources.KindCodec[aghconfig.MCPServer],
) {
	t.Helper()

	db := openDaemonTestGlobalDB(t)
	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}
	agentCodec, err := aghconfig.NewAgentResourceCodec()
	if err != nil {
		t.Fatalf("aghconfig.NewAgentResourceCodec() error = %v", err)
	}
	agentStore, err := resources.NewStore(kernel, agentCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(agent) error = %v", err)
	}
	skillCodec, err := skillspkg.NewResourceCodec()
	if err != nil {
		t.Fatalf("skillspkg.NewResourceCodec() error = %v", err)
	}
	skillStore, err := resources.NewStore(kernel, skillCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(skill) error = %v", err)
	}
	mcpCodec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("aghconfig.NewMCPServerResourceCodec() error = %v", err)
	}
	mcpStore, err := resources.NewStore(kernel, mcpCodec)
	if err != nil {
		t.Fatalf("resources.NewStore(mcp) error = %v", err)
	}
	return agentStore, agentCodec, skillStore, skillCodec, mcpStore, mcpCodec
}

func assertAgentSkillStoreCounts(
	t *testing.T,
	agentStore resources.Store[aghconfig.AgentDef],
	skillStore resources.Store[skillspkg.SkillResourceSpec],
	mcpStore resources.Store[aghconfig.MCPServer],
	wantAgents int,
	wantSkills int,
	wantMCP int,
) {
	t.Helper()

	source := agentSkillSyncActor().Source
	agents, err := agentStore.List(
		context.Background(),
		agentSkillSyncActor(),
		resources.ResourceFilter{Source: &source},
	)
	if err != nil {
		t.Fatalf("agentStore.List() error = %v", err)
	}
	skills, err := skillStore.List(
		context.Background(),
		agentSkillSyncActor(),
		resources.ResourceFilter{Source: &source},
	)
	if err != nil {
		t.Fatalf("skillStore.List() error = %v", err)
	}
	servers, err := mcpStore.List(
		context.Background(),
		agentSkillSyncActor(),
		resources.ResourceFilter{Source: &source},
	)
	if err != nil {
		t.Fatalf("mcpStore.List() error = %v", err)
	}
	if len(agents) != wantAgents || len(skills) != wantSkills || len(servers) != wantMCP {
		t.Fatalf(
			"store counts = agents:%d skills:%d mcp:%d, want agents:%d skills:%d mcp:%d",
			len(agents),
			len(skills),
			len(servers),
			wantAgents,
			wantSkills,
			wantMCP,
		)
	}
}
