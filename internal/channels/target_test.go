package channels_test

import (
	"strings"
	"testing"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestBuildDeliveryTargetDefaultsToDirectSend(t *testing.T) {
	t.Parallel()

	instance := testChannelInstanceForTargets()

	target, err := channelspkg.BuildDeliveryTarget(instance, channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-1",
	})
	if err != nil {
		t.Fatalf("BuildDeliveryTarget() error = %v", err)
	}
	if target.ChannelInstanceID != instance.ID {
		t.Fatalf("BuildDeliveryTarget().ChannelInstanceID = %q, want %q", target.ChannelInstanceID, instance.ID)
	}
	if target.PeerID != "peer-1" {
		t.Fatalf("BuildDeliveryTarget().PeerID = %q, want peer-1", target.PeerID)
	}
	if target.Mode != channelspkg.DeliveryModeDirectSend {
		t.Fatalf("BuildDeliveryTarget().Mode = %q, want %q", target.Mode, channelspkg.DeliveryModeDirectSend)
	}
}

func TestDeliveryTargetValidateRejectsMissingDestinationForMode(t *testing.T) {
	t.Parallel()

	target := channelspkg.DeliveryTarget{
		ChannelInstanceID: "chan-1",
		Mode:              channelspkg.DeliveryModeDirectSend,
	}

	err := target.Validate()
	if err == nil || !strings.Contains(err.Error(), "requires peer id or group id") {
		t.Fatalf("DeliveryTarget.Validate() error = %v, want peer/group requirement", err)
	}
}

func TestDeliveryTargetValidateRejectsThreadOnlyWithoutAnchor(t *testing.T) {
	t.Parallel()

	target := channelspkg.DeliveryTarget{
		ChannelInstanceID: "chan-1",
		ThreadID:          "thread-1",
		Mode:              channelspkg.DeliveryModeReply,
	}

	err := target.Validate()
	if err == nil || !strings.Contains(err.Error(), "thread id requires peer id or group id") {
		t.Fatalf("DeliveryTarget.Validate() error = %v, want thread anchor requirement", err)
	}
}

func TestBuildDeliveryTargetExplicitOverridesWinOverDefaults(t *testing.T) {
	t.Parallel()

	instance := testChannelInstanceForTargets()
	instance.DeliveryDefaults = []byte(`{"peer_id":"peer-default","thread_id":"thread-default","group_id":"group-default","mode":"reply"}`)

	target, err := channelspkg.BuildDeliveryTarget(instance, channelspkg.ResolveDeliveryTargetRequest{
		ChannelInstanceID: instance.ID,
		PeerID:            "peer-explicit",
		ThreadID:          "thread-explicit",
		Mode:              channelspkg.DeliveryModeDirectSend,
	})
	if err != nil {
		t.Fatalf("BuildDeliveryTarget() error = %v", err)
	}
	if target.PeerID != "peer-explicit" {
		t.Fatalf("BuildDeliveryTarget().PeerID = %q, want peer-explicit", target.PeerID)
	}
	if target.ThreadID != "thread-explicit" {
		t.Fatalf("BuildDeliveryTarget().ThreadID = %q, want thread-explicit", target.ThreadID)
	}
	if target.GroupID != "group-default" {
		t.Fatalf("BuildDeliveryTarget().GroupID = %q, want group-default", target.GroupID)
	}
	if target.Mode != channelspkg.DeliveryModeDirectSend {
		t.Fatalf("BuildDeliveryTarget().Mode = %q, want %q", target.Mode, channelspkg.DeliveryModeDirectSend)
	}
}

func TestRegistryResolveDeliveryTargetUsesServiceSeam(t *testing.T) {
	t.Parallel()

	registry, _ := newRegistryTestHarness(t)
	instance := createTestChannelInstance(t, registry, channelspkg.CreateInstanceRequest{
		ID:               "chan-target-service",
		Scope:            channelspkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "Target Service",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: []byte(`{"peer_id":"peer-service","thread_id":"thread-service","mode":"reply"}`),
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
	if target.PeerID != "peer-service" {
		t.Fatalf("ResolveDeliveryTarget().PeerID = %q, want peer-service", target.PeerID)
	}
	if target.ThreadID != "thread-service" {
		t.Fatalf("ResolveDeliveryTarget().ThreadID = %q, want thread-service", target.ThreadID)
	}
	if target.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("ResolveDeliveryTarget().Mode = %q, want %q", target.Mode, channelspkg.DeliveryModeReply)
	}
}

func testChannelInstanceForTargets() channelspkg.ChannelInstance {
	return channelspkg.ChannelInstance{
		ID:            "chan-targets",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Targets",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}
}
