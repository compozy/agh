//go:build integration

package channels_test

import (
	"testing"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRegistryResolveDeliveryTargetUsesInstanceDefaults(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:               "chan-target-defaults",
		Scope:            channelspkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "Target Defaults",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: []byte(`{"peer_id":"peer-default","thread_id":"thread-default","mode":"reply","parse_mode":"markdown"}`),
	})

	target, err := registry.ResolveDeliveryTarget(testutil.Context(t), channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: instance.ID,
	})
	if err != nil {
		t.Fatalf("ResolveDeliveryTarget() error = %v", err)
	}
	if target.ChannelInstanceID != instance.ID {
		t.Fatalf("ResolveDeliveryTarget().ChannelInstanceID = %q, want %q", target.ChannelInstanceID, instance.ID)
	}
	if target.PeerID != "peer-default" {
		t.Fatalf("ResolveDeliveryTarget().PeerID = %q, want peer-default", target.PeerID)
	}
	if target.ThreadID != "thread-default" {
		t.Fatalf("ResolveDeliveryTarget().ThreadID = %q, want thread-default", target.ThreadID)
	}
	if target.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("ResolveDeliveryTarget().Mode = %q, want %q", target.Mode, channelspkg.DeliveryModeReply)
	}
}

func TestRegistryResolveDeliveryTargetKeepsWorkspaceScopeIsolated(t *testing.T) {
	t.Parallel()

	registry, db := newRegistryTestHarness(t)
	workspaceID := registerWorkspaceForChannelsTests(t, db, "ws-target-scope", "target-scope")

	globalInstance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:               "chan-target-global",
		Scope:            channelspkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "Global Targets",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludeGroup: true},
		DeliveryDefaults: []byte(`{"group_id":"global-group","mode":"direct-send"}`),
	})
	workspaceInstance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:               "chan-target-workspace",
		Scope:            channelspkg.ScopeWorkspace,
		WorkspaceID:      workspaceID,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "Workspace Targets",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: []byte(`{"peer_id":"workspace-peer","thread_id":"workspace-thread","mode":"reply"}`),
	})

	globalTarget, err := registry.ResolveDeliveryTarget(testutil.Context(t), channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: globalInstance.ID,
	})
	if err != nil {
		t.Fatalf("ResolveDeliveryTarget(global) error = %v", err)
	}

	workspaceTarget, err := registry.ResolveDeliveryTarget(testutil.Context(t), channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: workspaceInstance.ID,
	})
	if err != nil {
		t.Fatalf("ResolveDeliveryTarget(workspace) error = %v", err)
	}

	if globalTarget.ChannelInstanceID != globalInstance.ID {
		t.Fatalf("globalTarget.ChannelInstanceID = %q, want %q", globalTarget.ChannelInstanceID, globalInstance.ID)
	}
	if globalTarget.GroupID != "global-group" {
		t.Fatalf("globalTarget.GroupID = %q, want global-group", globalTarget.GroupID)
	}
	if workspaceTarget.ChannelInstanceID != workspaceInstance.ID {
		t.Fatalf("workspaceTarget.ChannelInstanceID = %q, want %q", workspaceTarget.ChannelInstanceID, workspaceInstance.ID)
	}
	if workspaceTarget.PeerID != "workspace-peer" {
		t.Fatalf("workspaceTarget.PeerID = %q, want workspace-peer", workspaceTarget.PeerID)
	}
	if workspaceTarget.ThreadID != "workspace-thread" {
		t.Fatalf("workspaceTarget.ThreadID = %q, want workspace-thread", workspaceTarget.ThreadID)
	}
	if workspaceTarget.GroupID != "" {
		t.Fatalf("workspaceTarget.GroupID = %q, want empty", workspaceTarget.GroupID)
	}
	if workspaceTarget.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("workspaceTarget.Mode = %q, want %q", workspaceTarget.Mode, channelspkg.DeliveryModeReply)
	}
}
