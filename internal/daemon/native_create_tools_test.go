package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	apitest "github.com/compozy/agh/internal/api/testutil"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	toolspkg "github.com/compozy/agh/internal/tools"
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
