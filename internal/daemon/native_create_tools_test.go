package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	apitest "github.com/compozy/agh/internal/api/testutil"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestNativeNetworkChannelCreate(t *testing.T) {
	t.Parallel()

	var written store.NetworkChannelEntry
	writeCalls := 0
	netStore := apitest.StubNetworkStore{
		WriteNetworkChannelFn: func(_ context.Context, entry store.NetworkChannelEntry) error {
			writeCalls++
			written = entry
			return nil
		},
	}
	registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
		Network:      &nativeNetworkStub{},
		NetworkStore: netStore,
		Workspaces:   nativeNetworkTestWorkspaceService(t),
		Sessions:     nativeNetworkTestSessionManager(nativeNetworkTestWorkspaceID),
	}, nativeApproveAllPolicyInputs())

	t.Run("Should register a channel with purpose through the network store", func(t *testing.T) {
		result, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDNetworkChannelCreate,
			Input: json.RawMessage(
				`{"workspace_id":"ws-native-network","channel":"design","purpose":"UI reviews"}`,
			),
		})
		if err != nil {
			t.Fatalf("Registry.Call(network_channel_create) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte(`"design"`))
		if writeCalls != 1 {
			t.Fatalf("WriteNetworkChannel calls = %d, want 1", writeCalls)
		}
		if written.Channel != "design" ||
			written.WorkspaceID != nativeNetworkTestWorkspaceID ||
			written.Purpose != "UI reviews" {
			t.Fatalf("written entry = %#v, want design/native-workspace/UI reviews", written)
		}
	})

	t.Run("Should persist the registered workspace id when AGH identity differs", func(t *testing.T) {
		registryWorkspaceID := "ws-native-network"
		identityWorkspaceID := "01KSGVKVZVS4WP4HVMFE08J96Y"
		var stored store.NetworkChannelEntry
		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Network: &nativeNetworkStub{},
			NetworkStore: apitest.StubNetworkStore{
				WriteNetworkChannelFn: func(_ context.Context, entry store.NetworkChannelEntry) error {
					stored = entry
					return nil
				},
			},
			Workspaces: apitest.StubWorkspaceService{
				ResolveFn: func(ctx context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
					if err := ctx.Err(); err != nil {
						return workspacepkg.ResolvedWorkspace{}, err
					}
					if ref != registryWorkspaceID {
						t.Fatalf("Resolve() ref = %q, want %q", ref, registryWorkspaceID)
					}
					return workspacepkg.ResolvedWorkspace{
						Workspace: workspacepkg.Workspace{
							ID:      registryWorkspaceID,
							RootDir: t.TempDir(),
							Name:    "native-network",
						},
						WorkspaceID: identityWorkspaceID,
					}, nil
				},
			},
			Sessions: nativeNetworkTestSessionManager(registryWorkspaceID),
		}, nativeApproveAllPolicyInputs())

		result, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDNetworkChannelCreate,
			Input: json.RawMessage(
				"{\"workspace_id\":\"ws-native-network\",\"channel\":\"general\",\"purpose\":\"Announcements\"}",
			),
		})
		if err != nil {
			t.Fatalf("Registry.Call(network_channel_create) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte("\"general\""))
		if stored.WorkspaceID != registryWorkspaceID {
			t.Fatalf(
				"stored workspace_id = %q, want registry id %q; identity id %q must not be persisted",
				stored.WorkspaceID,
				registryWorkspaceID,
				identityWorkspaceID,
			)
		}
	})

	t.Run("Should create a durable channel when registry and identity ids differ", func(t *testing.T) {
		registryWorkspaceID := "ws-native-network"
		identityWorkspaceID := "01KSGVKVZVS4WP4HVMFE08J96Y"
		root := t.TempDir()
		db, err := globaldb.OpenGlobalDB(t.Context(), filepath.Join(t.TempDir(), store.GlobalDatabaseName))
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if closeErr := db.Close(cleanupCtx); closeErr != nil {
				t.Fatalf("Close() error = %v", closeErr)
			}
		})
		workspace := workspacepkg.Workspace{
			ID:      registryWorkspaceID,
			RootDir: root,
			Name:    "native-network",
		}
		if err := db.InsertWorkspace(t.Context(), workspace); err != nil {
			t.Fatalf("InsertWorkspace() error = %v", err)
		}
		registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
			Network:      &nativeNetworkStub{},
			NetworkStore: db,
			Workspaces: apitest.StubWorkspaceService{
				ResolveFn: func(ctx context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
					if err := ctx.Err(); err != nil {
						return workspacepkg.ResolvedWorkspace{}, err
					}
					if ref != registryWorkspaceID {
						t.Fatalf("Resolve() ref = %q, want %q", ref, registryWorkspaceID)
					}
					return workspacepkg.ResolvedWorkspace{
						Workspace:   workspace,
						WorkspaceID: identityWorkspaceID,
					}, nil
				},
			},
			Sessions: nativeNetworkTestSessionManager(registryWorkspaceID),
		}, nativeApproveAllPolicyInputs())

		result, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDNetworkChannelCreate,
			Input: json.RawMessage(
				"{\"workspace_id\":\"ws-native-network\",\"channel\":\"durable\",\"purpose\":\"Durable coordination\"}",
			),
		})
		if err != nil {
			t.Fatalf("Registry.Call(network_channel_create) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte("\"durable\""))
		entry, err := db.GetNetworkChannel(t.Context(), store.NetworkChannelRef{
			WorkspaceID: registryWorkspaceID,
			Channel:     "durable",
		})
		if err != nil {
			t.Fatalf("GetNetworkChannel() error = %v", err)
		}
		if entry.Purpose != "Durable coordination" {
			t.Fatalf("entry purpose = %q, want Durable coordination", entry.Purpose)
		}
	})

	t.Run("Should reject an invalid channel name", func(t *testing.T) {
		_, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDNetworkChannelCreate,
			Input: json.RawMessage(
				`{"workspace_id":"ws-native-network","channel":"Bad Name","purpose":"x"}`,
			),
		})
		if err == nil {
			t.Fatal("Registry.Call(invalid channel) error = nil, want validation error")
		}
	})

	t.Run("Should require a purpose", func(t *testing.T) {
		_, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDNetworkChannelCreate,
			Input: json.RawMessage(
				`{"workspace_id":"ws-native-network","channel":"general","purpose":"   "}`,
			),
		})
		if err == nil {
			t.Fatal("Registry.Call(blank purpose) error = nil, want validation error")
		}
	})
}

func TestNativeAgentCreate(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
		HomePaths:  homePaths,
		Workspaces: nativeNetworkTestWorkspaceService(t),
	}, nativeApproveAllPolicyInputs())

	t.Run("Should author one global AGENT.md", func(t *testing.T) {
		result, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDAgentCreate,
			Input: json.RawMessage(
				`{"scope":"global","name":"scout","provider":"claude","model":"claude-opus-4-7","prompt":"You scout the codebase."}`,
			),
		})
		if err != nil {
			t.Fatalf("Registry.Call(agent_create) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte(`"scout"`))
		path := filepath.Join(homePaths.AgentsDir, "scout", "AGENT.md")
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("agent definition not written at %q: %v", path, statErr)
		}
	})

	t.Run("Should conflict when the agent already exists", func(t *testing.T) {
		_, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDAgentCreate,
			Input: json.RawMessage(
				`{"scope":"global","name":"scout","provider":"claude","prompt":"Duplicate."}`,
			),
		})
		requireToolReason(t, err, toolspkg.ErrToolConflict, toolspkg.ReasonConflictedID)
	})

	t.Run("Should reject a request missing the provider", func(t *testing.T) {
		_, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDAgentCreate,
			Input:  json.RawMessage(`{"scope":"global","name":"nope","prompt":"x"}`),
		})
		if err == nil {
			t.Fatal("Registry.Call(missing provider) error = nil, want validation error")
		}
	})

	t.Run("Should reject reserved internal agent names", func(t *testing.T) {
		_, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDAgentCreate,
			Input: json.RawMessage(
				"{\"scope\":\"global\",\"name\":\"onboarding\",\"provider\":\"claude\",\"prompt\":\"Reserved.\"}",
			),
		})
		requireToolReason(t, err, toolspkg.ErrToolInvalidInput, toolspkg.ReasonSchemaInvalid)
	})

	t.Run("Should deny global scope when the onboarding agent is the caller", func(t *testing.T) {
		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{AgentName: aghconfig.OnboardingAgentName, Operator: true},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDAgentCreate,
				Input: json.RawMessage(
					`{"scope":"global","name":"escalated","provider":"claude","prompt":"injected"}`,
				),
			},
		)
		requireToolReason(t, err, toolspkg.ErrToolDenied, toolspkg.ReasonScopeMismatch)
	})
}

func TestNativeWorkspaceDescribeOmitsInternalManagedAgents(t *testing.T) {
	t.Parallel()

	const workspaceID = "ws-native-network"
	registry := newDaemonNativeRegistry(t, &daemonNativeToolsDeps{
		Workspaces: apitest.StubWorkspaceService{
			ResolveFn: func(ctx context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				if err := ctx.Err(); err != nil {
					return workspacepkg.ResolvedWorkspace{}, err
				}
				if ref != workspaceID {
					t.Fatalf("Resolve() ref = %q, want %q", ref, workspaceID)
				}
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspacepkg.Workspace{
						ID:      workspaceID,
						RootDir: t.TempDir(),
						Name:    "native-network",
					},
					WorkspaceID: workspaceID,
					Agents: []aghconfig.AgentDef{
						{Name: aghconfig.DefaultAgentName, Provider: "codex", Prompt: "General."},
						{Name: aghconfig.OnboardingAgentName, Provider: "codex", Prompt: "Onboarding."},
					},
				}, nil
			},
		},
		Sessions: nativeNetworkTestSessionManager(workspaceID),
		AgentCatalog: nativeAgentCatalogStub{agents: []aghconfig.AgentDef{
			{Name: "catalog-visible", Provider: "codex", Prompt: "Catalog visible."},
			{Name: aghconfig.OnboardingAgentName, Provider: "codex", Prompt: "Catalog onboarding."},
		}},
	}, nativeApproveAllPolicyInputs())

	result, err := registry.Call(t.Context(), toolspkg.Scope{}, toolspkg.CallRequest{
		ToolID: toolspkg.ToolIDWorkspaceDescribe,
		Input:  json.RawMessage("{\"workspace\":\"ws-native-network\"}"),
	})
	if err != nil {
		t.Fatalf("Registry.Call(workspace_describe) error = %v", err)
	}
	requireNativeStructuredContains(t, result, []byte("\"general\""))
	requireNativeStructuredContains(t, result, []byte("\"catalog-visible\""))
	requireNativeStructuredExcludes(t, result, []byte("\"onboarding\""))
}

type nativeAgentCatalogStub struct {
	agents []aghconfig.AgentDef
}

func (s nativeAgentCatalogStub) ListAgents(context.Context) ([]aghconfig.AgentDef, error) {
	return append([]aghconfig.AgentDef(nil), s.agents...), nil
}

func (s nativeAgentCatalogStub) GetAgent(_ context.Context, name string) (aghconfig.AgentDef, error) {
	for _, agent := range s.agents {
		if agent.Name == name {
			return agent, nil
		}
	}
	return aghconfig.AgentDef{}, os.ErrNotExist
}
