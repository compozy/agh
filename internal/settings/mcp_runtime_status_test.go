package settings

import (
	"context"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestListMCPServersIncludesRuntimeStatus(t *testing.T) {
	t.Run("Should attach daemon-backed runtime status to configured MCP servers", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, strings.Join([]string{
			"[[mcp_servers]]",
			"name = \"ready-docs\"",
			"command = \"docs-mcp\"",
			"",
			"[[mcp_servers]]",
			"name = \"linear\"",
			"transport = \"http\"",
			"url = \"https://mcp.linear.example/mcp\"",
			"",
			"[mcp_servers.auth]",
			"type = \"oauth2_pkce\"",
			"authorization_url = \"https://auth.linear.example/authorize\"",
			"token_url = \"https://auth.linear.example/token\"",
			"client_id = \"agh-desktop\"",
		}, "\n"))
		runtime := &fakeMCPRuntimeProvider{
			statuses: map[string]MCPServerRuntimeStatus{
				"ready-docs": {
					Configured:  true,
					Initialized: true,
					State:       MCPServerRuntimeStateReady,
					Probe:       MCPServerProbeSucceeded,
					ToolCount:   2,
				},
				"linear": {
					Configured: true,
					State:      MCPServerRuntimeStateAuthRequired,
					Probe:      MCPServerProbeSkipped,
					Reason:     "mcp_auth_required",
				},
			},
		}
		service := testService(t, homePaths, Dependencies{MCPRuntime: runtime})

		envelope, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionMCPServers})
		if err != nil {
			t.Fatalf("ListCollection(mcp) error = %v", err)
		}

		ready := findMCPItem(t, envelope.MCPServers, "ready-docs")
		if ready.RuntimeStatus == nil {
			t.Fatal("ready-docs RuntimeStatus = nil, want probe status")
		}
		if got, want := ready.RuntimeStatus.State, MCPServerRuntimeStateReady; got != want {
			t.Fatalf("ready-docs RuntimeStatus.State = %q, want %q", got, want)
		}
		if !ready.RuntimeStatus.Initialized || ready.RuntimeStatus.ToolCount != 2 {
			t.Fatalf("ready-docs RuntimeStatus = %#v, want initialized with 2 tools", ready.RuntimeStatus)
		}

		linear := findMCPItem(t, envelope.MCPServers, "linear")
		if linear.RuntimeStatus == nil {
			t.Fatal("linear RuntimeStatus = nil, want auth-blocked status")
		}
		if got, want := linear.RuntimeStatus.State, MCPServerRuntimeStateAuthRequired; got != want {
			t.Fatalf("linear RuntimeStatus.State = %q, want %q", got, want)
		}
		if got, want := linear.RuntimeStatus.Probe, MCPServerProbeSkipped; got != want {
			t.Fatalf("linear RuntimeStatus.Probe = %q, want %q", got, want)
		}
		if got, want := linear.RuntimeStatus.Reason, "mcp_auth_required"; got != want {
			t.Fatalf("linear RuntimeStatus.Reason = %q, want %q", got, want)
		}
	})
}

type fakeMCPRuntimeProvider struct {
	statuses map[string]MCPServerRuntimeStatus
}

func (f fakeMCPRuntimeProvider) MCPServerRuntimeStatus(
	_ context.Context,
	server aghconfig.MCPServer,
) (MCPServerRuntimeStatus, error) {
	if status, ok := f.statuses[strings.TrimSpace(server.Name)]; ok {
		return status, nil
	}
	return MCPServerRuntimeStatus{
		Configured: true,
		State:      MCPServerRuntimeStateRuntimeUnavailable,
		Probe:      MCPServerProbeFailed,
		Reason:     "test_missing_runtime_status",
	}, nil
}
