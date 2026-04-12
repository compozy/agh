//go:build integration

package channels_test

import (
	"testing"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRegistryGlobalAndWorkspaceRoutesStayIsolated(t *testing.T) {
	t.Parallel()

	registry, db := newRegistryTestHarness(t)
	workspaceID := registerWorkspaceForChannelsTests(t, db, "ws-route-scope", "route-scope")

	globalInstance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-global-route",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Global Route",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	workspaceInstance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-workspace-route",
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   workspaceID,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Workspace Route",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	globalRoute, created, err := registry.ResolveOrCreateRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: globalInstance.ID,
		PeerID:            "peer-1",
		SessionID:         "sess-global",
		AgentName:         "coder",
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateRoute(global) error = %v", err)
	}
	if !created {
		t.Fatal("ResolveOrCreateRoute(global) created = false, want true")
	}

	workspaceRoute, created, err := registry.ResolveOrCreateRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: workspaceInstance.ID,
		PeerID:            "peer-1",
		SessionID:         "sess-workspace",
		AgentName:         "coder",
	})
	if err != nil {
		t.Fatalf("ResolveOrCreateRoute(workspace) error = %v", err)
	}
	if !created {
		t.Fatal("ResolveOrCreateRoute(workspace) created = false, want true")
	}

	if globalRoute.RoutingKeyHash == workspaceRoute.RoutingKeyHash {
		t.Fatalf("RoutingKeyHash() = %q for both routes, want distinct records", globalRoute.RoutingKeyHash)
	}

	globalKey, err := registry.BuildRoutingKey(testutil.Context(t), channelspkg.RoutingKey{
		ChannelInstanceID: globalInstance.ID,
		PeerID:            "peer-1",
	})
	if err != nil {
		t.Fatalf("BuildRoutingKey(global) error = %v", err)
	}
	workspaceKey, err := registry.BuildRoutingKey(testutil.Context(t), channelspkg.RoutingKey{
		ChannelInstanceID: workspaceInstance.ID,
		PeerID:            "peer-1",
	})
	if err != nil {
		t.Fatalf("BuildRoutingKey(workspace) error = %v", err)
	}
	if globalKey.Scope == workspaceKey.Scope {
		t.Fatalf("globalKey.Scope = %q and workspaceKey.Scope = %q, want different scopes", globalKey.Scope, workspaceKey.Scope)
	}
	if workspaceKey.WorkspaceID != workspaceID {
		t.Fatalf("workspaceKey.WorkspaceID = %q, want %q", workspaceKey.WorkspaceID, workspaceID)
	}
}

func TestRegistryUpsertRouteRebindsWithoutDuplicateRows(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-rebind",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Route Rebind",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
	})

	first, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
	})
	if err != nil {
		t.Fatalf("UpsertRoute(first) error = %v", err)
	}

	second, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-2",
		AgentName:         "reviewer",
	})
	if err != nil {
		t.Fatalf("UpsertRoute(second) error = %v", err)
	}
	if first.RoutingKeyHash != second.RoutingKeyHash {
		t.Fatalf("UpsertRoute() hashes = %q and %q, want same canonical route", first.RoutingKeyHash, second.RoutingKeyHash)
	}
	if second.SessionID != "sess-2" {
		t.Fatalf("UpsertRoute(second).SessionID = %q, want sess-2", second.SessionID)
	}

	routes, err := registry.ListRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("ListRoutes() error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if routes[0].SessionID != "sess-2" {
		t.Fatalf("routes[0].SessionID = %q, want sess-2", routes[0].SessionID)
	}
}

func TestRegistryListRoutesReturnsOnlyTheRequestedInstance(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	first := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-list-a",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "List A",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	second := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:            "chan-list-b",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "List B",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	if _, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: first.ID,
		PeerID:            "peer-a",
		SessionID:         "sess-a",
		AgentName:         "coder",
	}); err != nil {
		t.Fatalf("UpsertRoute(first) error = %v", err)
	}
	if _, err := registry.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: second.ID,
		PeerID:            "peer-b",
		SessionID:         "sess-b",
		AgentName:         "coder",
	}); err != nil {
		t.Fatalf("UpsertRoute(second) error = %v", err)
	}

	routes, err := registry.ListRoutes(testutil.Context(t), first.ID)
	if err != nil {
		t.Fatalf("ListRoutes(first) error = %v", err)
	}
	if got, want := len(routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if routes[0].ChannelInstanceID != first.ID {
		t.Fatalf("routes[0].ChannelInstanceID = %q, want %q", routes[0].ChannelInstanceID, first.ID)
	}
}
