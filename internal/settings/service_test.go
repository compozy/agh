package settings

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestGetSectionBuildsSupportedSections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())

	service := testService(t, homePaths, Dependencies{
		GeneralRuntime: fakeGeneralRuntimeProvider{
			status: DaemonRuntimeStatus{
				Available:      true,
				Status:         "running",
				PID:            1234,
				UptimeSeconds:  99,
				ActiveSessions: 4,
				ActiveAgents:   3,
				TotalSessions:  6,
				Version:        "1.2.3",
			},
		},
		MemoryRuntime: fakeMemoryRuntimeProvider{
			status: MemoryHealthStatus{
				Available:          true,
				FileCount:          5,
				DreamEnabled:       true,
				LastConsolidatedAt: pointerTime(time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)),
			},
		},
		SkillsRuntime: newFakeSkillsRuntime(
			testSkill("alpha", false),
			testSkill("beta", false),
			testSkill("gamma", true),
		),
		AutomationRuntime: fakeAutomationRuntimeProvider{
			status: AutomationRuntimeStatus{
				Available:        true,
				Running:          true,
				SchedulerRunning: true,
				JobTotal:         4,
				JobEnabled:       3,
				TriggerTotal:     2,
				TriggerEnabled:   1,
				NextFire:         pointerTime(time.Date(2026, 4, 17, 13, 0, 0, 0, time.UTC)),
			},
		},
		NetworkRuntime: fakeNetworkRuntimeProvider{
			status: NetworkRuntimeStatus{
				Available:    true,
				Enabled:      true,
				Status:       "running",
				ListenerHost: "127.0.0.1",
				ListenerPort: 4222,
			},
		},
		ObservabilityRuntime: fakeObservabilityRuntimeProvider{
			status: ObservabilityRuntimeStatus{
				Available:          true,
				Status:             "ok",
				GlobalDBSizeBytes:  2048,
				SessionDBSizeBytes: 4096,
			},
		},
		Extensions: fakeExtensionStatusProvider{
			items: []InstalledExtension{{
				Name:    "linear",
				Version: "1.0.0",
				Enabled: true,
				State:   "active",
				Health:  "healthy",
			}},
		},
		TransportParity: fakeTransportParityProvider{
			status: TransportParityStatus{
				Known:          true,
				SettingsHTTP:   true,
				SettingsUDS:    true,
				ExtensionsHTTP: true,
				ExtensionsUDS:  true,
			},
		},
		RestartActionAvailable:     true,
		ConsolidateActionAvailable: true,
		LogTailAvailable:           true,
	})

	tests := []struct {
		name   SectionName
		assert func(t *testing.T, envelope SectionEnvelope)
	}{
		{
			name: SectionGeneral,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.General == nil {
					t.Fatal("General section = nil")
				}
				if got, want := envelope.General.Settings.Defaults.Agent, "writer"; got != want {
					t.Fatalf("General defaults agent = %q, want %q", got, want)
				}
				if got, want := envelope.General.Runtime.PID, 1234; got != want {
					t.Fatalf("General runtime PID = %d, want %d", got, want)
				}
				if !envelope.General.Actions.Restart.Available {
					t.Fatal("General restart action unavailable")
				}
			},
		},
		{
			name: SectionMemory,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.Memory == nil {
					t.Fatal("Memory section = nil")
				}
				if got, want := envelope.Memory.Config.Dream.Agent, "writer"; got != want {
					t.Fatalf("Memory dream agent = %q, want %q", got, want)
				}
				if got, want := envelope.Memory.Health.FileCount, 5; got != want {
					t.Fatalf("Memory file count = %d, want %d", got, want)
				}
			},
		},
		{
			name: SectionSkills,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.Skills == nil {
					t.Fatal("Skills section = nil")
				}
				if got, want := envelope.Skills.Config.Marketplace.Registry, "clawhub"; got != want {
					t.Fatalf("Skills marketplace registry = %q, want %q", got, want)
				}
				if got, want := envelope.Skills.DiscoveredCount, 3; got != want {
					t.Fatalf("Skills discovered count = %d, want %d", got, want)
				}
				if got, want := envelope.Skills.DisabledCount, 2; got != want {
					t.Fatalf("Skills disabled count = %d, want %d", got, want)
				}
			},
		},
		{
			name: SectionAutomation,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.Automation == nil {
					t.Fatal("Automation section = nil")
				}
				if got, want := envelope.Automation.Config.Timezone, "UTC"; got != want {
					t.Fatalf("Automation timezone = %q, want %q", got, want)
				}
				if got, want := envelope.Automation.Runtime.JobTotal, 4; got != want {
					t.Fatalf("Automation jobs = %d, want %d", got, want)
				}
			},
		},
		{
			name: SectionNetwork,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.Network == nil {
					t.Fatal("Network section = nil")
				}
				if got, want := envelope.Network.Config.DefaultChannel, "ops"; got != want {
					t.Fatalf("Network default channel = %q, want %q", got, want)
				}
				if got, want := envelope.Network.Runtime.Status, "running"; got != want {
					t.Fatalf("Network runtime status = %q, want %q", got, want)
				}
			},
		},
		{
			name: SectionObservability,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.Observability == nil {
					t.Fatal("Observability section = nil")
				}
				if got, want := envelope.Observability.Config.Transcripts.SegmentBytes, 512; got != want {
					t.Fatalf("Observability transcripts segment bytes = %d, want %d", got, want)
				}
				if got, want := envelope.Observability.Runtime.GlobalDBSizeBytes, int64(2048); got != want {
					t.Fatalf("Observability global db bytes = %d, want %d", got, want)
				}
				if !envelope.Observability.LogTailSupport.Available {
					t.Fatal("Log tail capability unavailable")
				}
			},
		},
		{
			name: SectionHooksExtensions,
			assert: func(t *testing.T, envelope SectionEnvelope) {
				t.Helper()
				if envelope.HooksExtensions == nil {
					t.Fatal("HooksExtensions section = nil")
				}
				if got, want := len(envelope.HooksExtensions.Hooks), 1; got != want {
					t.Fatalf("Hooks count = %d, want %d", got, want)
				}
				if got, want := len(envelope.HooksExtensions.Installed), 1; got != want {
					t.Fatalf("Installed extensions count = %d, want %d", got, want)
				}
				if !envelope.HooksExtensions.TransportParity.ExtensionsHTTP {
					t.Fatal("Extensions HTTP parity = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			envelope, err := service.GetSection(ctx, SectionRequest{Section: tt.name})
			if err != nil {
				t.Fatalf("GetSection(%q) error = %v", tt.name, err)
			}
			tt.assert(t, envelope)
		})
	}
}

func TestInvalidScopeCombinationsReturnDescriptiveError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	service := testService(t, homePaths, Dependencies{})

	t.Run("section workspace unsupported", func(t *testing.T) {
		t.Parallel()
		_, err := service.GetSection(ctx, SectionRequest{
			Section:     SectionGeneral,
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
		})
		if err == nil || !strings.Contains(err.Error(), "does not support workspace scope") {
			t.Fatalf("GetSection(workspace) error = %v, want unsupported workspace scope", err)
		}
	})

	t.Run("providers workspace unsupported", func(t *testing.T) {
		t.Parallel()
		_, err := service.ListCollection(ctx, CollectionRequest{
			Collection:  CollectionProviders,
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
		})
		if err == nil || !strings.Contains(err.Error(), "does not support workspace scope") {
			t.Fatalf("ListCollection(providers workspace) error = %v, want unsupported workspace scope", err)
		}
	})

	t.Run("workspace mcp requires workspace id", func(t *testing.T) {
		t.Parallel()
		_, err := service.ListCollection(ctx, CollectionRequest{
			Collection: CollectionMCPServers,
			Scope:      ScopeWorkspace,
		})
		if err == nil || !strings.Contains(err.Error(), "requires a workspace_id") {
			t.Fatalf("ListCollection(mcp workspace) error = %v, want workspace_id error", err)
		}
	})
}

func TestListMCPServersIncludesPrecedenceMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")

	writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "alpha"
command = "global-config"
`)
	writeFile(t, filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "alpha": { "command": "global-sidecar" },
    "beta": { "command": "beta-sidecar" }
  }
}`)
	writeFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.ConfigName), `
[[mcp_servers]]
name = "alpha"
command = "workspace-config"
`)
	writeFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "alpha": { "command": "workspace-sidecar" }
  }
}`)

	service := testService(t, homePaths, Dependencies{
		WorkspaceResolver: fakeWorkspaceResolver{
			resolved: map[string]workspacepkg.ResolvedWorkspace{
				"ws-1": {
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: workspaceRoot},
				},
			},
		},
	})

	globalEnvelope, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionMCPServers})
	if err != nil {
		t.Fatalf("ListCollection(global mcp) error = %v", err)
	}
	globalAlpha := findMCPItem(t, globalEnvelope.MCPServers, "alpha")
	if got, want := globalAlpha.SourceMetadata.EffectiveSource.Kind, SourceKindGlobalMCPSidecar; got != want {
		t.Fatalf("global alpha effective source = %q, want %q", got, want)
	}
	if got, want := len(globalAlpha.SourceMetadata.ShadowedSources), 1; got != want {
		t.Fatalf("global alpha shadowed sources = %d, want %d", got, want)
	}
	if got, want := globalAlpha.SourceMetadata.ShadowedSources[0].Kind, SourceKindGlobalConfig; got != want {
		t.Fatalf("global alpha shadowed source = %q, want %q", got, want)
	}
	if got, want := globalAlpha.SourceMetadata.AvailableTargets, []WriteTargetKind{
		WriteTargetGlobalConfig,
		WriteTargetGlobalMCPSidecar,
	}; !equalWriteTargets(got, want) {
		t.Fatalf("global alpha available targets = %#v, want %#v", got, want)
	}

	workspaceEnvelope, err := service.ListCollection(ctx, CollectionRequest{
		Collection:  CollectionMCPServers,
		Scope:       ScopeWorkspace,
		WorkspaceID: "ws-1",
	})
	if err != nil {
		t.Fatalf("ListCollection(workspace mcp) error = %v", err)
	}
	workspaceAlpha := findMCPItem(t, workspaceEnvelope.MCPServers, "alpha")
	if got, want := workspaceAlpha.SourceMetadata.EffectiveSource.Kind, SourceKindWorkspaceMCPSidecar; got != want {
		t.Fatalf("workspace alpha effective source = %q, want %q", got, want)
	}
	if got, want := len(workspaceAlpha.SourceMetadata.ShadowedSources), 3; got != want {
		t.Fatalf("workspace alpha shadowed sources = %d, want %d", got, want)
	}
	if got, want := workspaceAlpha.SourceMetadata.AvailableTargets, []WriteTargetKind{
		WriteTargetWorkspaceConfig,
		WriteTargetWorkspaceMCPSidecar,
	}; !equalWriteTargets(got, want) {
		t.Fatalf("workspace alpha available targets = %#v, want %#v", got, want)
	}
}

func TestMCPTargetAutoSelectsExistingSourceAndDefaultsNewEntriesToSidecar(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "alpha"
command = "before"
`)

	service := testService(t, homePaths, Dependencies{})

	result, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
		Name:              "alpha",
		Target:            TargetAuto,
		MCPServer: &aghconfig.MCPServer{
			Command: "after",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(existing alpha) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("existing alpha write target = %q, want %q", got, want)
	}
	configPayload := readFile(t, homePaths.ConfigFile)
	if !strings.Contains(configPayload, `command = "after"`) {
		t.Fatalf("config payload missing updated alpha command:\n%s", configPayload)
	}

	result, err = service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
		Name:              "beta",
		Target:            TargetAuto,
		MCPServer: &aghconfig.MCPServer{
			Command: "beta-command",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(new beta) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetGlobalMCPSidecar; got != want {
		t.Fatalf("new beta write target = %q, want %q", got, want)
	}
	sidecarPayload := readFile(t, filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName))
	if !strings.Contains(sidecarPayload, `"beta"`) || !strings.Contains(sidecarPayload, `"beta-command"`) {
		t.Fatalf("sidecar payload missing beta server:\n%s", sidecarPayload)
	}
}

func TestUpdateSectionGeneralReturnsRestartRequired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	service := testService(t, homePaths, Dependencies{})

	result, err := service.UpdateSection(ctx, SectionUpdateRequest{
		SectionRequest: SectionRequest{Section: SectionGeneral},
		General: &GeneralSettings{
			Defaults: aghconfig.DefaultsConfig{
				Agent:    "editor",
				Provider: "codex",
				Sandbox:  "dev",
			},
			Limits: aghconfig.LimitsConfig{
				MaxSessions:         7,
				MaxConcurrentAgents: 11,
			},
			Permissions:    aghconfig.PermissionsConfig{Mode: aghconfig.PermissionModeApproveReads},
			SessionTimeout: 45 * time.Minute,
			HTTP:           aghconfig.HTTPConfig{Host: "127.0.0.1", Port: 9001},
			Daemon:         aghconfig.DaemonConfig{Socket: "/tmp/agh.sock"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSection(general) error = %v", err)
	}
	if got, want := result.Behavior, MutationBehaviorRestartRequired; got != want {
		t.Fatalf("general behavior = %q, want %q", got, want)
	}
	if !result.RestartRequired {
		t.Fatal("general restart_required = false, want true")
	}
	if got, want := result.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("general write target = %q, want %q", got, want)
	}
}

func TestUpdateSectionSkillsAppliesDisabledSkillsNow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	skillsRuntime := newFakeSkillsRuntime(
		testSkill("alpha", false),
		testSkill("beta", false),
		testSkill("gamma", true),
	)
	service := testService(t, homePaths, Dependencies{SkillsRuntime: skillsRuntime})

	result, err := service.UpdateSection(ctx, SectionUpdateRequest{
		SectionRequest: SectionRequest{Section: SectionSkills},
		Skills: &aghconfig.SkillsConfig{
			Enabled:                 true,
			DisabledSkills:          []string{"beta"},
			PollInterval:            30 * time.Minute,
			AllowedMarketplaceMCP:   []string{"ctx"},
			AllowedMarketplaceHooks: []string{"market"},
			Marketplace: aghconfig.MarketplaceConfig{
				Registry: "clawhub",
				BaseURL:  "https://skills.example",
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSection(skills disabled) error = %v", err)
	}
	if got, want := result.Behavior, MutationBehaviorAppliedNow; got != want {
		t.Fatalf("skills behavior = %q, want %q", got, want)
	}
	if !result.Applied {
		t.Fatal("skills applied = false, want true")
	}
	if result.RestartRequired {
		t.Fatal("skills restart_required = true, want false")
	}
	if got, want := skillsRuntime.enabled["alpha"], true; got != want {
		t.Fatalf("alpha enabled = %v, want %v", got, want)
	}
	if got, want := skillsRuntime.enabled["beta"], false; got != want {
		t.Fatalf("beta enabled = %v, want %v", got, want)
	}
}

func TestUpdateSectionRejectsMixedRuntimeBehaviors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	service := testService(t, homePaths, Dependencies{SkillsRuntime: newFakeSkillsRuntime(testSkill("alpha", false))})

	_, err := service.UpdateSection(ctx, SectionUpdateRequest{
		SectionRequest: SectionRequest{Section: SectionSkills},
		Skills: &aghconfig.SkillsConfig{
			Enabled:                 true,
			DisabledSkills:          []string{"beta"},
			PollInterval:            time.Hour,
			AllowedMarketplaceMCP:   []string{"ctx"},
			AllowedMarketplaceHooks: []string{"market"},
			Marketplace: aghconfig.MarketplaceConfig{
				Registry: "different",
				BaseURL:  "https://skills.example",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "mixes") {
		t.Fatalf("UpdateSection(mixed skills behavior) error = %v, want mixed-behavior failure", err)
	}
}

func TestClassifyMutationReturnsMatrixBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor MutationDescriptor
		want       MutationBehavior
	}{
		{
			name: "applied now",
			descriptor: MutationDescriptor{
				Section:       SectionSkills,
				ChangedFields: []string{"skills.disabled_skills"},
			},
			want: MutationBehaviorAppliedNow,
		},
		{
			name: "restart required",
			descriptor: MutationDescriptor{
				Section:       SectionGeneral,
				ChangedFields: []string{"defaults.agent"},
			},
			want: MutationBehaviorRestartRequired,
		},
		{
			name: "action trigger",
			descriptor: MutationDescriptor{
				Section: SectionMemory,
				Action:  "consolidate",
			},
			want: MutationBehaviorActionTrigger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			classification, err := ClassifyMutation(tt.descriptor)
			if err != nil {
				t.Fatalf("ClassifyMutation() error = %v", err)
			}
			if got, want := classification.Behavior, tt.want; got != want {
				t.Fatalf("behavior = %q, want %q", got, want)
			}
		})
	}
}

func TestClassifyMutationSupportsActionTriggers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		descriptor MutationDescriptor
	}{
		{
			name: "general restart",
			descriptor: MutationDescriptor{
				Section: SectionGeneral,
				Action:  "restart",
			},
		},
		{
			name: "hooks extensions install",
			descriptor: MutationDescriptor{
				Section: SectionHooksExtensions,
				Action:  "extension-install",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classification, err := ClassifyMutation(tt.descriptor)
			if err != nil {
				t.Fatalf("ClassifyMutation() error = %v", err)
			}
			if got, want := classification.Behavior, MutationBehaviorActionTrigger; got != want {
				t.Fatalf("behavior = %q, want %q", got, want)
			}
			if !classification.Applied {
				t.Fatal("classification.Applied = false, want true")
			}
		})
	}
}

func TestClassifyMutationSupportsCollectionFields(t *testing.T) {
	t.Parallel()

	tests := []MutationDescriptor{
		{
			Section:       SectionName(CollectionProviders),
			ChangedFields: []string{"providers.custom.command"},
		},
		{
			Section:       SectionName(CollectionMCPServers),
			ChangedFields: []string{"mcp-servers.alpha.command"},
		},
		{
			Section:       SectionName(CollectionSandboxes),
			ChangedFields: []string{"sandboxes.dev.backend"},
		},
		{
			Section:       SectionName(CollectionHooks),
			ChangedFields: []string{"hooks.audit.command"},
		},
	}

	for _, descriptor := range tests {
		t.Run(string(descriptor.Section), func(t *testing.T) {
			t.Parallel()

			classification, err := ClassifyMutation(descriptor)
			if err != nil {
				t.Fatalf("ClassifyMutation() error = %v", err)
			}
			if got, want := classification.Behavior, MutationBehaviorRestartRequired; got != want {
				t.Fatalf("behavior = %q, want %q", got, want)
			}
		})
	}
}

func TestListCollectionBuildsProvidersSandboxesAndHooks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig()+`

[providers.codex]
default_model = "gpt-5"

[providers.custom]
command = "custom-acp --stdio"
default_model = "custom-model"
api_key_env = "CUSTOM_API_KEY"

[sandboxes.staging]
backend = "local"

[[hooks.declarations]]
name = "ship"
event = "session.post_create"
mode = "async"
command = "/bin/ship"
`)

	service := testService(t, homePaths, Dependencies{
		CommandLookPath: func(command string) (string, error) {
			if strings.HasPrefix(command, "custom-acp") {
				return "", os.ErrNotExist
			}
			return "/bin/" + command, nil
		},
		LookupEnv: func(key string) (string, bool) {
			if key == "OPENAI_API_KEY" {
				return "token", true
			}
			return "", false
		},
		WorkspaceResolver: fakeWorkspaceResolver{
			listed: []workspacepkg.Workspace{
				{ID: "ws-dev", SandboxRef: "dev"},
				{ID: "ws-stage-a", SandboxRef: "staging"},
				{ID: "ws-stage-b", SandboxRef: "staging"},
			},
		},
	})

	providers, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionProviders})
	if err != nil {
		t.Fatalf("ListCollection(providers) error = %v", err)
	}
	codex := mustFindProviderItem(t, providers.Providers, "codex")
	if got, want := codex.Settings.DefaultModel, "gpt-5"; got != want {
		t.Fatalf("codex default model = %q, want %q", got, want)
	}
	if !codex.Default {
		t.Fatal("codex default = false, want true")
	}
	if got, want := codex.SourceMetadata.EffectiveSource.Kind, SourceKindGlobalConfig; got != want {
		t.Fatalf("codex effective source = %q, want %q", got, want)
	}
	if codex.Fallback == nil || codex.Fallback.Source.Kind != SourceKindBuiltinProvider {
		t.Fatalf("codex fallback = %#v, want builtin fallback", codex.Fallback)
	}
	custom := mustFindProviderItem(t, providers.Providers, "custom")
	if got, want := custom.SourceMetadata.EffectiveSource.Kind, SourceKindGlobalConfig; got != want {
		t.Fatalf("custom effective source = %q, want %q", got, want)
	}
	if custom.CommandAvailable {
		t.Fatal("custom command available = true, want false")
	}
	if custom.APIKeyEnvPresent {
		t.Fatal("custom API key env present = true, want false")
	}
	claude := mustFindProviderItem(t, providers.Providers, "claude")
	if got, want := claude.SourceMetadata.EffectiveSource.Kind, SourceKindBuiltinProvider; got != want {
		t.Fatalf("claude effective source = %q, want %q", got, want)
	}

	sandboxes, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionSandboxes})
	if err != nil {
		t.Fatalf("ListCollection(sandboxes) error = %v", err)
	}
	dev := findSandboxItem(t, sandboxes.Sandboxes, "dev")
	if got, want := dev.WorkspaceUsageCount, 1; got != want {
		t.Fatalf("dev workspace usage = %d, want %d", got, want)
	}
	staging := findSandboxItem(t, sandboxes.Sandboxes, "staging")
	if got, want := staging.WorkspaceUsageCount, 2; got != want {
		t.Fatalf("staging workspace usage = %d, want %d", got, want)
	}

	hooks, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionHooks})
	if err != nil {
		t.Fatalf("ListCollection(hooks) error = %v", err)
	}
	if got, want := hooks.Hooks[0].Name, "audit"; got != want {
		t.Fatalf("hooks[0].Name = %q, want %q", got, want)
	}
	if got, want := hooks.Hooks[1].Name, "ship"; got != want {
		t.Fatalf("hooks[1].Name = %q, want %q", got, want)
	}
	if got, want := hooks.Hooks[1].SourceMetadata.EffectiveSource.Kind, SourceKindGlobalConfig; got != want {
		t.Fatalf("hook effective source = %q, want %q", got, want)
	}
}

func TestCollectionMutationsProviderSandboxAndHook(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	service := testService(t, homePaths, Dependencies{})

	providerResult, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		Name:              "custom",
		Provider: &ProviderSettings{
			Command:      "custom-acp --stdio",
			DefaultModel: "custom-model",
			APIKeyEnv:    "CUSTOM_API_KEY",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(provider) error = %v", err)
	}
	if got, want := providerResult.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("provider write target = %q, want %q", got, want)
	}
	if got, want := providerResult.Behavior, MutationBehaviorRestartRequired; got != want {
		t.Fatalf("provider behavior = %q, want %q", got, want)
	}
	configPayload := readFile(t, homePaths.ConfigFile)
	if !strings.Contains(configPayload, "[providers.custom]") ||
		!strings.Contains(configPayload, `default_model = "custom-model"`) {
		t.Fatalf("config payload missing provider overlay:\n%s", configPayload)
	}
	if _, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		Name:              "custom",
	}); err != nil {
		t.Fatalf("DeleteCollectionItem(provider) error = %v", err)
	}
	configPayload = readFile(t, homePaths.ConfigFile)
	if strings.Contains(configPayload, "[providers.custom]") {
		t.Fatalf("provider overlay still present after delete:\n%s", configPayload)
	}

	sandboxResult, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionSandboxes},
		Name:              "staging",
		Sandbox: &aghconfig.SandboxProfile{
			Backend:     "local",
			SyncMode:    "session-bidirectional",
			Persistence: "transient",
			RuntimeRoot: "/tmp/staging",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(sandbox) error = %v", err)
	}
	if got, want := sandboxResult.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("sandbox write target = %q, want %q", got, want)
	}
	configPayload = readFile(t, homePaths.ConfigFile)
	if !strings.Contains(configPayload, "[sandboxes.staging]") ||
		!strings.Contains(configPayload, `runtime_root = "/tmp/staging"`) {
		t.Fatalf("config payload missing sandbox overlay:\n%s", configPayload)
	}
	if _, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionSandboxes},
		Name:              "staging",
	}); err != nil {
		t.Fatalf("DeleteCollectionItem(sandbox) error = %v", err)
	}
	configPayload = readFile(t, homePaths.ConfigFile)
	if strings.Contains(configPayload, "[sandboxes.staging]") {
		t.Fatalf("sandbox overlay still present after delete:\n%s", configPayload)
	}

	hookResult, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionHooks},
		Name:              "ship",
		Hook: &hookspkg.HookDecl{
			Event:   hookspkg.HookToolPreCall,
			Mode:    hookspkg.HookModeAsync,
			Command: "/bin/ship",
			Args:    []string{"--fast"},
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(hook) error = %v", err)
	}
	if got, want := hookResult.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("hook write target = %q, want %q", got, want)
	}
	configPayload = readFile(t, homePaths.ConfigFile)
	if !strings.Contains(configPayload, `name = "ship"`) || !strings.Contains(configPayload, `args = ["--fast"]`) {
		t.Fatalf("config payload missing hook declaration:\n%s", configPayload)
	}
	if _, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionHooks},
		Name:              "ship",
	}); err != nil {
		t.Fatalf("DeleteCollectionItem(hook) error = %v", err)
	}
	configPayload = readFile(t, homePaths.ConfigFile)
	if strings.Contains(configPayload, `name = "ship"`) {
		t.Fatalf("hook declaration still present after delete:\n%s", configPayload)
	}
}

func TestDeleteMCPServerAutoUsesHighestPrecedenceSourceInScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")

	writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "alpha"
command = "global-config"
`)
	writeFile(t, filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "alpha": { "command": "global-sidecar" },
    "beta": { "command": "beta-sidecar" }
  }
}`)
	writeFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.ConfigName), `
[[mcp_servers]]
name = "alpha"
command = "workspace-config"
`)
	writeFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.MCPJSONName), `{
  "mcpServers": {
    "alpha": { "command": "workspace-sidecar" }
  }
}`)

	service := testService(t, homePaths, Dependencies{
		WorkspaceResolver: fakeWorkspaceResolver{
			resolved: map[string]workspacepkg.ResolvedWorkspace{
				"ws-1": {
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: workspaceRoot},
				},
			},
		},
	})

	result, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
		Name:              "alpha",
		Target:            TargetAuto,
	})
	if err != nil {
		t.Fatalf("DeleteCollectionItem(global alpha sidecar) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetGlobalMCPSidecar; got != want {
		t.Fatalf("global alpha first delete target = %q, want %q", got, want)
	}
	sidecarPayload := readFile(t, filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName))
	if strings.Contains(sidecarPayload, `"alpha"`) {
		t.Fatalf("global sidecar alpha still present after delete:\n%s", sidecarPayload)
	}

	result, err = service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
		Name:              "alpha",
		Target:            TargetAuto,
	})
	if err != nil {
		t.Fatalf("DeleteCollectionItem(global alpha config) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("global alpha second delete target = %q, want %q", got, want)
	}
	configPayload := readFile(t, homePaths.ConfigFile)
	if strings.Contains(configPayload, `name = "alpha"`) {
		t.Fatalf("global config alpha still present after delete:\n%s", configPayload)
	}

	result, err = service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{
			Collection:  CollectionMCPServers,
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
		},
		Name:   "alpha",
		Target: TargetAuto,
	})
	if err != nil {
		t.Fatalf("DeleteCollectionItem(workspace alpha sidecar) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetWorkspaceMCPSidecar; got != want {
		t.Fatalf("workspace alpha first delete target = %q, want %q", got, want)
	}
	workspaceSidecarPayload := readFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.MCPJSONName))
	if strings.Contains(workspaceSidecarPayload, `"alpha"`) {
		t.Fatalf("workspace sidecar alpha still present after delete:\n%s", workspaceSidecarPayload)
	}

	result, err = service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{
			Collection:  CollectionMCPServers,
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
		},
		Name:   "alpha",
		Target: TargetAuto,
	})
	if err != nil {
		t.Fatalf("DeleteCollectionItem(workspace alpha config) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetWorkspaceConfig; got != want {
		t.Fatalf("workspace alpha second delete target = %q, want %q", got, want)
	}
	workspaceConfigPayload := readFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.ConfigName))
	if strings.Contains(workspaceConfigPayload, `name = "alpha"`) {
		t.Fatalf("workspace config alpha still present after delete:\n%s", workspaceConfigPayload)
	}
}

func TestUpdateSectionRestartRequiredSections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name    string
		request SectionUpdateRequest
		want    string
	}{
		{
			name: "memory",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionMemory},
				Memory: &aghconfig.MemoryConfig{
					Enabled:   true,
					GlobalDir: "/tmp/updated-memory",
					Dream: aghconfig.DreamConfig{
						Enabled:       true,
						Agent:         "writer",
						MinHours:      12,
						MinSessions:   3,
						CheckInterval: 15 * time.Minute,
					},
				},
			},
			want: `global_dir = "/tmp/updated-memory"`,
		},
		{
			name: "skills restart required",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionSkills},
				Skills: &aghconfig.SkillsConfig{
					Enabled:                 true,
					DisabledSkills:          []string{"alpha", "beta"},
					PollInterval:            45 * time.Minute,
					AllowedMarketplaceMCP:   []string{"ctx"},
					AllowedMarketplaceHooks: []string{"market"},
					Marketplace: aghconfig.MarketplaceConfig{
						Registry: "clawhub",
						BaseURL:  "https://skills-updated.example",
					},
				},
			},
			want: `base_url = "https://skills-updated.example"`,
		},
		{
			name: "automation",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionAutomation},
				Automation: &AutomationSettings{
					Enabled:           true,
					Timezone:          "America/Sao_Paulo",
					MaxConcurrentJobs: 5,
					DefaultFireLimit: automationmodel.FireLimitConfig{
						Max:    9,
						Window: "1h",
					},
				},
			},
			want: `max_concurrent_jobs = 5`,
		},
		{
			name: "network",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionNetwork},
				Network: &aghconfig.NetworkConfig{
					Enabled:        true,
					DefaultChannel: "alerts",
					Port:           4222,
					MaxPayload:     4096,
					GreetInterval:  15,
					MaxReplayAge:   60,
					MaxQueueDepth:  10,
				},
			},
			want: `default_channel = "alerts"`,
		},
		{
			name: "observability",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionObservability},
				Observability: &aghconfig.ObservabilityConfig{
					Enabled:        true,
					RetentionDays:  21,
					MaxGlobalBytes: 4096,
					Transcripts: aghconfig.ObservabilityTranscriptConfig{
						Enabled:            true,
						SegmentBytes:       1024,
						MaxBytesPerSession: 2048,
					},
				},
			},
			want: `segment_bytes = 1024`,
		},
		{
			name: "hooks extensions",
			request: SectionUpdateRequest{
				SectionRequest: SectionRequest{Section: SectionHooksExtensions},
				HooksExtensions: &aghconfig.ExtensionsConfig{
					Marketplace: aghconfig.ExtensionsMarketplaceConfig{
						Registry: "github",
						BaseURL:  "https://extensions-updated.example",
					},
					Resources: aghconfig.ExtensionsResourcesConfig{
						AllowedKinds: []resources.ResourceKind{
							resources.ResourceKind("tool"),
							resources.ResourceKind("mcp_server"),
						},
						MaxScope: resources.ResourceScopeKindWorkspace,
						SnapshotRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
							Requests: 7,
							Window:   time.Minute,
							Queue:    3,
						},
						OperatorWriteRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
							Requests: 9,
							Window:   2 * time.Minute,
							Queue:    4,
						},
					},
				},
			},
			want: `base_url = "https://extensions-updated.example"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			homePaths := testHomePaths(t)
			writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
			service := testService(t, homePaths, Dependencies{
				SkillsRuntime: newFakeSkillsRuntime(testSkill("alpha", false), testSkill("beta", false)),
			})

			result, err := service.UpdateSection(ctx, tt.request)
			if err != nil {
				t.Fatalf("UpdateSection(%s) error = %v", tt.name, err)
			}
			if got, want := result.Behavior, MutationBehaviorRestartRequired; got != want {
				t.Fatalf("behavior = %q, want %q", got, want)
			}
			if !result.RestartRequired {
				t.Fatal("restart_required = false, want true")
			}
			if got, want := result.WriteTarget, WriteTargetGlobalConfig; got != want {
				t.Fatalf("write target = %q, want %q", got, want)
			}
			payload := readFile(t, homePaths.ConfigFile)
			if !strings.Contains(payload, tt.want) {
				t.Fatalf("config payload missing %q:\n%s", tt.want, payload)
			}
		})
	}
}

func TestCollectionHelperMapsIncludeNestedFields(t *testing.T) {
	t.Parallel()

	profileValues := sandboxProfileMap(aghconfig.SandboxProfile{
		Backend:  "daytona",
		SyncMode: "mirror",
		Env:      map[string]string{"TOKEN": "value"},
		Network: aghconfig.NetworkProfile{
			AllowPublicIngress: true,
			AllowOutbound:      true,
			AllowList:          []string{"api.example"},
			DenyList:           []string{"blocked.example"},
			Required:           true,
		},
		Daytona: aghconfig.DaytonaProfile{
			APIURL:      "https://daytona.example",
			Target:      "prod",
			Image:       "agh:latest",
			Snapshot:    "snap-1",
			Class:       "large",
			AutoStop:    "15m",
			AutoArchive: "24h",
		},
	})
	if _, ok := profileValues["env"]; !ok {
		t.Fatalf("sandboxProfileMap() missing env: %#v", profileValues)
	}
	if _, ok := profileValues["network"]; !ok {
		t.Fatalf("sandboxProfileMap() missing network: %#v", profileValues)
	}
	if _, ok := profileValues["daytona"]; !ok {
		t.Fatalf("sandboxProfileMap() missing daytona: %#v", profileValues)
	}

	readOnly := true
	decl := hookspkg.HookDecl{
		Name:         "capture",
		Event:        hookspkg.HookToolPreCall,
		Mode:         hookspkg.HookModeAsync,
		ExecutorKind: hookspkg.HookExecutorSubprocess,
		Command:      "/bin/capture",
		Args:         []string{"--json"},
		Env:          map[string]string{"TOKEN": "value"},
		Matcher: hookspkg.HookMatcher{
			ToolID:           "agh__read",
			ToolReadOnly:     &readOnly,
			MessageRole:      "assistant",
			MessageDeltaType: "text",
		},
	}
	matcher := hookMatcherMap(decl)
	if got, want := matcher["tool_id"], "agh__read"; got != want {
		t.Fatalf("hookMatcherMap()[tool_id] = %#v, want %q", got, want)
	}
	executor := hookExecutorMap(decl)
	if got, want := executor["kind"], string(hookspkg.HookExecutorSubprocess); got != want {
		t.Fatalf("hookExecutorMap()[kind] = %#v, want %q", got, want)
	}
	values := hookDeclarationMap(decl)
	if _, ok := values["executor"]; !ok {
		t.Fatalf("hookDeclarationMap() missing executor: %#v", values)
	}
}

func TestHookDeclarationMapStoresCommandFieldsInExecutorBlock(t *testing.T) {
	t.Parallel()

	values := hookDeclarationMap(hookspkg.HookDecl{
		Name:    "ship",
		Event:   hookspkg.HookToolPreCall,
		Mode:    hookspkg.HookModeAsync,
		Command: "/bin/ship",
		Args:    []string{"--fast"},
		Env:     map[string]string{"TOKEN": "value"},
	})
	executor, ok := values["executor"].(map[string]any)
	if !ok {
		t.Fatalf("hookDeclarationMap() executor = %#v, want map", values["executor"])
	}
	for _, key := range []string{"command", "args", "env"} {
		if _, ok := executor[key]; !ok {
			t.Fatalf("hookDeclarationMap() executor missing %q: %#v", key, executor)
		}
	}
}

func TestUpdateSectionNoChangesReturnsWarning(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	service := testService(t, homePaths, Dependencies{})

	result, err := service.UpdateSection(ctx, SectionUpdateRequest{
		SectionRequest: SectionRequest{Section: SectionGeneral},
		General: &GeneralSettings{
			Defaults: aghconfig.DefaultsConfig{
				Agent:    "writer",
				Provider: "codex",
				Sandbox:  "dev",
			},
			Limits: aghconfig.LimitsConfig{
				MaxSessions:         7,
				MaxConcurrentAgents: 11,
			},
			Permissions:    aghconfig.PermissionsConfig{Mode: aghconfig.PermissionModeApproveReads},
			SessionTimeout: 45 * time.Minute,
			HTTP:           aghconfig.HTTPConfig{Host: "127.0.0.1", Port: 9001},
			Daemon:         aghconfig.DaemonConfig{Socket: "/tmp/agh.sock"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSection(no changes) error = %v", err)
	}
	if got, want := result.Behavior, MutationBehaviorAppliedNow; got != want {
		t.Fatalf("behavior = %q, want %q", got, want)
	}
	if len(result.Warnings) == 0 || result.Warnings[0] != "no changes" {
		t.Fatalf("warnings = %#v, want no changes", result.Warnings)
	}
}

func TestSectionAndCollectionValidationErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, baseSettingsConfig())
	service := testService(t, homePaths, Dependencies{})

	t.Run("unknown section", func(t *testing.T) {
		t.Parallel()
		_, err := service.GetSection(ctx, SectionRequest{Section: SectionName("unknown")})
		if err == nil || !strings.Contains(err.Error(), `unknown section "unknown"`) {
			t.Fatalf("GetSection(unknown) error = %v", err)
		}
	})

	t.Run("missing section payload", func(t *testing.T) {
		t.Parallel()
		_, err := service.UpdateSection(ctx, SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionMemory},
		})
		if err == nil || !strings.Contains(err.Error(), "memory section payload is required") {
			t.Fatalf("UpdateSection(memory nil) error = %v", err)
		}
	})

	t.Run("empty collection name", func(t *testing.T) {
		t.Parallel()
		_, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		})
		if err == nil || !strings.Contains(err.Error(), "collection item name is required") {
			t.Fatalf("PutCollectionItem(empty name) error = %v", err)
		}
	})

	t.Run("missing provider payload", func(t *testing.T) {
		t.Parallel()
		_, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionProviders},
			Name:              "custom",
		})
		if err == nil || !strings.Contains(err.Error(), "provider payload is required") {
			t.Fatalf("PutCollectionItem(provider nil) error = %v", err)
		}
	})

	t.Run("missing mcp payload", func(t *testing.T) {
		t.Parallel()
		_, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
			Name:              "alpha",
		})
		if err == nil || !strings.Contains(err.Error(), "MCP server payload is required") {
			t.Fatalf("PutCollectionItem(mcp nil) error = %v", err)
		}
	})

	t.Run("unknown collection", func(t *testing.T) {
		t.Parallel()
		_, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionName("unknown")},
			Name:              "alpha",
		})
		if err == nil || !strings.Contains(err.Error(), `unknown collection "unknown"`) {
			t.Fatalf("DeleteCollectionItem(unknown) error = %v", err)
		}
	})

	t.Run("delete empty name", func(t *testing.T) {
		t.Parallel()
		_, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		})
		if err == nil || !strings.Contains(err.Error(), "collection item name is required") {
			t.Fatalf("DeleteCollectionItem(empty name) error = %v", err)
		}
	})

	t.Run("update unknown section", func(t *testing.T) {
		t.Parallel()
		_, err := service.UpdateSection(ctx, SectionUpdateRequest{
			SectionRequest: SectionRequest{Section: SectionName("mystery")},
		})
		if err == nil || !strings.Contains(err.Error(), `unknown section "mystery"`) {
			t.Fatalf("UpdateSection(unknown) error = %v", err)
		}
	})
}

type fakeGeneralRuntimeProvider struct {
	status DaemonRuntimeStatus
}

func (f fakeGeneralRuntimeProvider) GeneralRuntimeStatus(context.Context) (DaemonRuntimeStatus, error) {
	return f.status, nil
}

type fakeMemoryRuntimeProvider struct {
	status MemoryHealthStatus
}

func (f fakeMemoryRuntimeProvider) MemoryHealthStatus(context.Context) (MemoryHealthStatus, error) {
	return f.status, nil
}

type fakeAutomationRuntimeProvider struct {
	status AutomationRuntimeStatus
}

func (f fakeAutomationRuntimeProvider) AutomationRuntimeStatus(context.Context) (AutomationRuntimeStatus, error) {
	return f.status, nil
}

type fakeNetworkRuntimeProvider struct {
	status NetworkRuntimeStatus
}

func (f fakeNetworkRuntimeProvider) NetworkRuntimeStatus(context.Context) (NetworkRuntimeStatus, error) {
	return f.status, nil
}

type fakeObservabilityRuntimeProvider struct {
	status ObservabilityRuntimeStatus
}

func (f fakeObservabilityRuntimeProvider) ObservabilityRuntimeStatus(
	context.Context,
) (ObservabilityRuntimeStatus, error) {
	return f.status, nil
}

type fakeExtensionStatusProvider struct {
	items []InstalledExtension
}

func (f fakeExtensionStatusProvider) InstalledExtensions(context.Context) ([]InstalledExtension, error) {
	return append([]InstalledExtension(nil), f.items...), nil
}

type fakeTransportParityProvider struct {
	status TransportParityStatus
}

func (f fakeTransportParityProvider) TransportParityStatus(context.Context) (TransportParityStatus, error) {
	return f.status, nil
}

type fakeWorkspaceResolver struct {
	resolved map[string]workspacepkg.ResolvedWorkspace
	listed   []workspacepkg.Workspace
}

func (f fakeWorkspaceResolver) Resolve(
	_ context.Context,
	idOrNameOrPath string,
) (workspacepkg.ResolvedWorkspace, error) {
	if resolved, ok := f.resolved[idOrNameOrPath]; ok {
		return resolved, nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (f fakeWorkspaceResolver) List(context.Context) ([]workspacepkg.Workspace, error) {
	return append([]workspacepkg.Workspace(nil), f.listed...), nil
}

type fakeSkillsRuntime struct {
	skills  []*skillspkg.Skill
	enabled map[string]bool
}

func newFakeSkillsRuntime(skills ...*skillspkg.Skill) *fakeSkillsRuntime {
	enabled := make(map[string]bool, len(skills))
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		enabled[skill.Meta.Name] = skill.Enabled
	}
	return &fakeSkillsRuntime{
		skills:  append([]*skillspkg.Skill(nil), skills...),
		enabled: enabled,
	}
}

func (f *fakeSkillsRuntime) List() []*skillspkg.Skill {
	out := make([]*skillspkg.Skill, 0, len(f.skills))
	for _, skill := range f.skills {
		if skill == nil {
			continue
		}
		cloned := *skill
		cloned.Enabled = f.enabled[skill.Meta.Name]
		out = append(out, &cloned)
	}
	return out
}

func (f *fakeSkillsRuntime) SetEnabled(name string, _ *workspacepkg.ResolvedWorkspace, enabled bool) error {
	f.enabled[name] = enabled
	return nil
}

func testService(t *testing.T, homePaths aghconfig.HomePaths, deps Dependencies) Service {
	t.Helper()

	if deps.CommandLookPath == nil {
		deps.CommandLookPath = func(string) (string, error) { return "/bin/tool", nil }
	}
	if deps.LookupEnv == nil {
		deps.LookupEnv = func(key string) (string, bool) {
			if key == "OPENAI_API_KEY" {
				return "token", true
			}
			return "", false
		}
	}

	service, err := NewService(homePaths, deps)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func testHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return homePaths
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(payload)
}

func baseSettingsConfig() string {
	return `
[defaults]
agent = "writer"
provider = "codex"
sandbox = "dev"

[limits]
max_sessions = 7
max_concurrent_agents = 11

[session.limits]
timeout = "45m"

[permissions]
mode = "approve-reads"

[http]
host = "127.0.0.1"
port = 9001

[daemon]
socket = "/tmp/agh.sock"

[memory]
enabled = true
global_dir = "/tmp/memory"

[memory.dream]
enabled = true
agent = "writer"
min_hours = 12
min_sessions = 2
check_interval = "15m"

[skills]
enabled = true
disabled_skills = ["alpha", "beta"]
poll_interval = "30m"
allowed_marketplace_mcp = ["ctx"]
allowed_marketplace_hooks = ["market"]

[skills.marketplace]
registry = "clawhub"
base_url = "https://skills.example"

[automation]
enabled = true
timezone = "UTC"
max_concurrent_jobs = 3

[automation.default_fire_limit]
max = 9
window = "1h"

[sandboxes.dev]
backend = "local"

[network]
enabled = true
default_channel = "ops"
port = 4222
max_payload = 4096
greet_interval = 15
max_replay_age = 60
max_queue_depth = 10

[observability]
enabled = true
retention_days = 14
max_global_bytes = 2048

[observability.transcripts]
enabled = true
segment_bytes = 512
max_bytes_per_session = 1024

[extensions.marketplace]
registry = "github"
base_url = "https://ext.example"

[extensions.resources]
allowed_kinds = ["tool", "mcp_server"]
max_scope = "workspace"

[extensions.resources.snapshot_rate_limit]
requests = 5
window = "1m"
queue = 2

[extensions.resources.operator_write_rate_limit]
requests = 7
window = "2m"
queue = 3

[[hooks.declarations]]
name = "audit"
event = "tool.pre_call"
mode = "sync"
command = "/bin/echo"
`
}

func mustFindProviderItem(t *testing.T, items []ProviderItem, name string) ProviderItem {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("Provider item %q not found in %#v", name, items)
	return ProviderItem{}
}

func findSandboxItem(t *testing.T, items []SandboxItem, name string) SandboxItem {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("Sandbox item %q not found in %#v", name, items)
	return SandboxItem{}
}

func findMCPItem(t *testing.T, items []MCPServerItem, name string) MCPServerItem {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("MCP item %q not found in %#v", name, items)
	return MCPServerItem{}
}

func equalWriteTargets(left []WriteTargetKind, right []WriteTargetKind) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func testSkill(name string, enabled bool) *skillspkg.Skill {
	return &skillspkg.Skill{
		Meta:    skillspkg.SkillMeta{Name: name},
		Enabled: enabled,
	}
}

func pointerTime(value time.Time) *time.Time {
	return &value
}
