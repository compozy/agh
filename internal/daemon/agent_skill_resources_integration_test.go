//go:build integration

package daemon

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/resources"
	skillspkg "github.com/compozy/agh/internal/skills"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestAgentSkillPublicationAndBootRebuild(t *testing.T) {
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

	homePaths := agentSkillIntegrationHome(t)
	workspaceRoot := agentSkillIntegrationWorkspace(t)
	now := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	workspace := workspacepkg.Workspace{
		ID:           "ws_agent_skill",
		RootDir:      workspaceRoot,
		Name:         "agent-skill",
		DefaultAgent: "coder",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.InsertWorkspace(testutil.Context(t), workspace); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	workspaceResolver, err := workspacepkg.NewResolver(
		db,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}

	extensionRegistry := extensionpkg.NewRegistry(db.DB())
	extensionSnapshot := agentSkillIntegrationExtension(t, extensionRegistry)
	runtime := &agentSkillIntegrationRuntime{extension: extensionSnapshot}

	initialAgentCatalog := newResourceCatalog(cloneAgentDef)
	initialSkillRegistry := skillspkg.NewRegistry(
		agentSkillIntegrationSkillConfig(homePaths),
		skillspkg.WithLogger(discardLogger()),
	)
	initialMCPCatalog := newResourceCatalog(cloneDaemonMCPServer)
	driver := newAgentSkillIntegrationDriver(
		t,
		kernel,
		agentCodec,
		skillCodec,
		mcpCodec,
		initialAgentCatalog,
		initialSkillRegistry,
		initialMCPCatalog,
	)

	syncer := newAgentSkillSourceSyncer(
		kernel,
		agentStore,
		agentCodec,
		skillStore,
		skillCodec,
		mcpStore,
		mcpCodec,
		agentSkillSyncActor(),
		discardLogger(),
		func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			return driver.Trigger(ctx, kind, reason)
		},
		daemonAgentSkillDeclarationProvider(
			homePaths,
			db,
			workspaceResolver,
			initialSkillRegistry,
			discardLogger(),
		),
		extensionAgentSkillDeclarationProvider(
			extensionRegistry,
			func() extensionRuntime { return runtime },
			discardLogger(),
		),
	)
	if err := syncer.Sync(testutil.Context(t)); err != nil {
		t.Fatalf("syncer.Sync() error = %v", err)
	}

	source := agentSkillSyncActor().Source
	agents, err := agentStore.List(
		testutil.Context(t),
		agentSkillSyncActor(),
		resources.ResourceFilter{Source: &source},
	)
	if err != nil {
		t.Fatalf("agentStore.List() error = %v", err)
	}
	if got, want := len(agents), 2; got != want {
		t.Fatalf("len(agentStore.List()) = %d, want %d (%#v)", got, want, agents)
	}
	skills, err := skillStore.List(
		testutil.Context(t),
		agentSkillSyncActor(),
		resources.ResourceFilter{Source: &source},
	)
	if err != nil {
		t.Fatalf("skillStore.List() error = %v", err)
	}
	if got, want := len(skills), 2; got != want {
		t.Fatalf("len(skillStore.List()) = %d, want %d (%#v)", got, want, skills)
	}
	servers, err := mcpStore.List(testutil.Context(t), agentSkillSyncActor(), resources.ResourceFilter{Source: &source})
	if err != nil {
		t.Fatalf("mcpStore.List() error = %v", err)
	}
	if got, want := len(servers), 4; got != want {
		t.Fatalf("len(mcpStore.List()) = %d, want %d (%#v)", got, want, servers)
	}
	if err := syncer.Sync(testutil.Context(t)); err != nil {
		t.Fatalf("second syncer.Sync() error = %v", err)
	}

	rebuiltAgentCatalog := newResourceCatalog(cloneAgentDef)
	rebuiltSkillRegistry := skillspkg.NewRegistry(
		agentSkillIntegrationSkillConfig(homePaths),
		skillspkg.WithLogger(discardLogger()),
	)
	rebuiltMCPCatalog := newResourceCatalog(cloneDaemonMCPServer)
	bootDriver := newAgentSkillIntegrationDriver(
		t,
		kernel,
		agentCodec,
		skillCodec,
		mcpCodec,
		rebuiltAgentCatalog,
		rebuiltSkillRegistry,
		rebuiltMCPCatalog,
	)
	if err := bootDriver.RunBoot(testutil.Context(t)); err != nil {
		t.Fatalf("bootDriver.RunBoot() error = %v", err)
	}

	resolved, err := workspaceResolver.Resolve(testutil.Context(t), workspace.ID)
	if err != nil {
		t.Fatalf("workspaceResolver.Resolve() error = %v", err)
	}
	agentCatalog := agentCatalogDependency(rebuiltAgentCatalog)
	coder, err := agentCatalog.ResolveAgent("coder", &resolved)
	if err != nil {
		t.Fatalf("ResolveAgent(coder) error = %v", err)
	}
	if !slices.Contains(coder.Tools, "agh__lookup") {
		t.Fatalf("ResolveAgent(coder).Tools = %#v, want canonical lookup tool reference preserved", coder.Tools)
	}
	if !agentHasMCP(coder, "workspace-agent-mcp") {
		t.Fatalf("ResolveAgent(coder).MCPServers = %#v, want workspace-agent-mcp", coder.MCPServers)
	}
	extAgent, err := agentCatalog.ResolveAgent("ext-agent", &resolved)
	if err != nil {
		t.Fatalf("ResolveAgent(ext-agent) error = %v", err)
	}
	if !agentHasMCP(extAgent, "ext-agent-mcp") {
		t.Fatalf("ResolveAgent(ext-agent).MCPServers = %#v, want ext-agent-mcp", extAgent.MCPServers)
	}

	projectedSkills, err := rebuiltSkillRegistry.ForWorkspace(testutil.Context(t), &resolved)
	if err != nil {
		t.Fatalf("rebuiltSkillRegistry.ForWorkspace() error = %v", err)
	}
	review := findIntegrationSkill(projectedSkills, "workspace-review")
	if review == nil {
		t.Fatalf("ForWorkspace() = %#v, want workspace-review", projectedSkills)
	}
	if !skillHasMCP(review, "workspace-skill-mcp") {
		t.Fatalf("workspace-review MCPServers = %#v, want workspace-skill-mcp", review.MCPServers)
	}
	extSkill := findIntegrationSkill(projectedSkills, "ext-skill")
	if extSkill == nil {
		t.Fatalf("ForWorkspace() = %#v, want ext-skill", projectedSkills)
	}
	if !skillHasMCP(extSkill, "ext-skill-mcp") {
		t.Fatalf("ext-skill MCPServers = %#v, want ext-skill-mcp", extSkill.MCPServers)
	}
	if !mcpCatalogHas(rebuiltMCPCatalog, "workspace-agent-mcp") ||
		!mcpCatalogHas(rebuiltMCPCatalog, "workspace-skill-mcp") ||
		!mcpCatalogHas(rebuiltMCPCatalog, "ext-agent-mcp") ||
		!mcpCatalogHas(rebuiltMCPCatalog, "ext-skill-mcp") {
		t.Fatalf("rebuilt MCP catalog = %#v, want all agent/skill MCP attachments", rebuiltMCPCatalog.Snapshot())
	}
}

type agentSkillIntegrationRuntime struct {
	extension *extensionpkg.Extension
}

func (r *agentSkillIntegrationRuntime) Start(context.Context) error  { return nil }
func (r *agentSkillIntegrationRuntime) Stop(context.Context) error   { return nil }
func (r *agentSkillIntegrationRuntime) Reload(context.Context) error { return nil }

func (r *agentSkillIntegrationRuntime) Get(name string) (*extensionpkg.Extension, error) {
	if r.extension == nil || r.extension.Info.Name != name {
		return nil, &extensionpkg.ExtensionNotFoundError{Name: name}
	}
	return r.extension, nil
}

func (r *agentSkillIntegrationRuntime) HookDeclarations(context.Context) ([]hookspkg.HookDecl, error) {
	return nil, nil
}

func newAgentSkillIntegrationDriver(
	t *testing.T,
	kernel resources.RawStore,
	agentCodec resources.KindCodec[aghconfig.AgentDef],
	skillCodec resources.KindCodec[skillspkg.SkillResourceSpec],
	mcpCodec resources.KindCodec[aghconfig.MCPServer],
	agentCatalog *resourceCatalog[aghconfig.AgentDef],
	skillRegistry *skillspkg.Registry,
	mcpCatalog *resourceCatalog[aghconfig.MCPServer],
) resources.ReconcileDriver {
	t.Helper()

	agentRegistration, err := resources.NewTypedProjectorRegistration(agentCodec, newAgentProjector(agentCatalog))
	if err != nil {
		t.Fatalf("resources.NewTypedProjectorRegistration(agent) error = %v", err)
	}
	skillRegistration, err := resources.NewTypedProjectorRegistration(skillCodec, newSkillProjector(skillRegistry))
	if err != nil {
		t.Fatalf("resources.NewTypedProjectorRegistration(skill) error = %v", err)
	}
	mcpRegistration, err := resources.NewTypedProjectorRegistration(mcpCodec, newMCPServerProjector(mcpCatalog))
	if err != nil {
		t.Fatalf("resources.NewTypedProjectorRegistration(mcp) error = %v", err)
	}
	driver, err := resources.NewReconcileDriver(
		kernel,
		resources.MutationActor{
			Kind: resources.MutationActorKindDaemon,
			ID:   "agent-skill-integration",
			Source: resources.ResourceSource{
				Kind: resources.ResourceSourceKind("daemon"),
				ID:   "agent-skill-integration",
			},
			MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		},
		[]resources.ProjectorRegistration{agentRegistration, skillRegistration, mcpRegistration},
		resources.WithReconcileLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("resources.NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		if err := driver.Close(context.Background()); err != nil {
			t.Fatalf("driver.Close() error = %v", err)
		}
	})
	return driver
}

func agentSkillIntegrationHome(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("aghconfig.ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("aghconfig.EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func agentSkillIntegrationSkillConfig(homePaths aghconfig.HomePaths) skillspkg.RegistryConfig {
	return skillspkg.RegistryConfig{
		UserSkillsDir: homePaths.SkillsDir,
		UserAgentsDir: homePaths.AgentsDir,
	}
}

func agentSkillIntegrationWorkspace(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "workspace")
	agentDir := filepath.Join(root, aghconfig.DirName, aghconfig.AgentsDirName, "coder")
	writeAgentSkillIntegrationFile(t, filepath.Join(agentDir, "AGENT.md"), `---
name: coder
provider: claude
tools: ["agh__lookup"]
---

Use the workspace tool catalog.
`)
	writeAgentSkillIntegrationFile(t, filepath.Join(agentDir, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "workspace-agent-mcp": {
      "command": "workspace-agent-command"
    }
  }
}`)

	skillDir := filepath.Join(root, aghconfig.DirName, aghconfig.SkillsDirName, "workspace-review")
	writeAgentSkillIntegrationFile(t, filepath.Join(skillDir, "SKILL.md"), `---
name: workspace-review
description: Workspace review skill
---

Review workspace changes.
`)
	writeAgentSkillIntegrationFile(t, filepath.Join(skillDir, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "workspace-skill-mcp": {
      "command": "workspace-skill-command"
    }
  }
}`)

	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q) error = %v", root, err)
	}
	return canonical
}

func agentSkillIntegrationExtension(
	t *testing.T,
	registry *extensionpkg.Registry,
) *extensionpkg.Extension {
	t.Helper()

	dir := t.TempDir()
	writeAgentSkillIntegrationFile(t, filepath.Join(dir, "extension.toml"), `[extension]
name = "agent-skill-ext"
version = "0.1.0"
min_agh_version = "0.5.0"

[resources]
skills = ["skills/"]
agents = ["agents/"]
`)
	agentPath := filepath.Join(dir, "agents", "ext-agent.md")
	writeAgentSkillIntegrationFile(t, agentPath, `---
name: ext-agent
provider: claude
mcp_servers:
  - name: ext-agent-mcp
    command: ext-agent-command
---

Use extension-provided context.
`)
	skillPath := filepath.Join(dir, "skills", "ext-skill.md")
	writeAgentSkillIntegrationFile(t, skillPath, `---
name: ext-skill
description: Extension skill
---

Use extension skill context.
`)
	writeAgentSkillIntegrationFile(t, filepath.Join(dir, "skills", aghconfig.MCPJSONName), `{
  "mcpServers": {
    "ext-skill-mcp": {
      "command": "ext-skill-command"
    }
  }
}`)

	manifest, err := extensionpkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("extensionpkg.LoadManifest() error = %v", err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("extensionpkg.ComputeDirectoryChecksum() error = %v", err)
	}
	if err := registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("registry.Install() error = %v", err)
	}
	info, err := registry.Get(manifest.Name)
	if err != nil {
		t.Fatalf("registry.Get(%q) error = %v", manifest.Name, err)
	}
	agent, err := aghconfig.LoadAgentDefFile(agentPath)
	if err != nil {
		t.Fatalf("aghconfig.LoadAgentDefFile(%q) error = %v", agentPath, err)
	}
	skill, err := skillspkg.ParseSkillFileWithSource(skillPath, skillspkg.SourceUser)
	if err != nil {
		t.Fatalf("skillspkg.ParseSkillFileWithSource(%q) error = %v", skillPath, err)
	}
	return &extensionpkg.Extension{
		Info:     *info,
		Manifest: manifest,
		RootDir:  dir,
		Agents:   []aghconfig.AgentDef{agent},
		Skills:   []*skillspkg.Skill{skill},
		Status: extensionpkg.ExtensionStatus{
			Name:       info.Name,
			Version:    info.Version,
			Source:     info.Source,
			Enabled:    info.Enabled,
			Registered: true,
		},
	}
}

func writeAgentSkillIntegrationFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func agentHasMCP(agent aghconfig.AgentDef, name string) bool {
	for _, server := range agent.MCPServers {
		if server.Name == name {
			return true
		}
	}
	return false
}

func skillHasMCP(skill *skillspkg.Skill, name string) bool {
	if skill == nil {
		return false
	}
	for _, server := range skill.MCPServers {
		if server.Name == name {
			return true
		}
	}
	return false
}

func findIntegrationSkill(skills []*skillspkg.Skill, name string) *skillspkg.Skill {
	for _, skill := range skills {
		if skill != nil && skill.Meta.Name == name {
			return skill
		}
	}
	return nil
}

func mcpCatalogHas(catalog *resourceCatalog[aghconfig.MCPServer], name string) bool {
	if catalog == nil {
		return false
	}
	for _, record := range catalog.Snapshot() {
		if record.Spec.Name == name {
			return true
		}
	}
	return false
}
