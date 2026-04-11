package contract_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
)

func TestCreateChannelRequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  contract.CreateChannelRequest
	}{
		{
			name: "workspace scope requires workspace id",
			req: contract.CreateChannelRequest{
				Scope:         channelspkg.ScopeWorkspace,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
		},
		{
			name: "global scope rejects workspace id",
			req: contract.CreateChannelRequest{
				Scope:         channelspkg.ScopeGlobal,
				WorkspaceID:   "ws-alpha",
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			},
		},
		{
			name: "routing policy rejects thread without peer or group",
			req: contract.CreateChannelRequest{
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludeThread: true},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.req.ToCreateInstanceRequest(); err == nil {
				t.Fatal("ToCreateInstanceRequest() error = nil, want non-nil")
			}
		})
	}
}

func TestCreateChannelRequestPreservesNormalizedFieldsAndDefaults(t *testing.T) {
	t.Parallel()

	req := contract.CreateChannelRequest{
		Scope:            channelspkg.ScopeWorkspace,
		WorkspaceID:      " ws-alpha ",
		Platform:         " telegram ",
		ExtensionName:    " ext-telegram ",
		DisplayName:      " Support ",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true},
		DeliveryDefaults: json.RawMessage(`{"mode":"reply"}`),
	}

	mapped, err := req.ToCreateInstanceRequest()
	if err != nil {
		t.Fatalf("ToCreateInstanceRequest() error = %v", err)
	}
	if mapped.WorkspaceID != "ws-alpha" || mapped.Platform != "telegram" || mapped.ExtensionName != "ext-telegram" || mapped.DisplayName != "Support" {
		t.Fatalf("mapped request = %#v", mapped)
	}
	if string(mapped.DeliveryDefaults) != `{"mode":"reply"}` {
		t.Fatalf("mapped.DeliveryDefaults = %s", string(mapped.DeliveryDefaults))
	}

	req.DeliveryDefaults[0] = '['
	if string(mapped.DeliveryDefaults) != `{"mode":"reply"}` {
		t.Fatalf("mapped.DeliveryDefaults mutated with source slice = %s", string(mapped.DeliveryDefaults))
	}
}

func TestChannelRoutesResponseJSONShape(t *testing.T) {
	t.Parallel()

	payload := contract.ChannelRoutesResponse{
		Routes: []channelspkg.ChannelRoute{
			{
				RoutingKeyHash:    "hash-1",
				Scope:             channelspkg.ScopeWorkspace,
				WorkspaceID:       "ws-alpha",
				ChannelInstanceID: "chan-1",
				PeerID:            "peer-1",
				ThreadID:          "thread-1",
				GroupID:           "group-1",
				SessionID:         "sess-1",
				AgentName:         "coder",
			},
		},
	}

	var got map[string]any
	marshalJSON(t, payload, &got)

	routes, ok := got["routes"].([]any)
	if !ok || len(routes) != 1 {
		t.Fatalf("routes JSON = %#v", got["routes"])
	}
	route, ok := routes[0].(map[string]any)
	if !ok {
		t.Fatalf("route JSON type = %T, want object", routes[0])
	}
	if route["peer_id"] != "peer-1" || route["thread_id"] != "thread-1" || route["group_id"] != "group-1" {
		t.Fatalf("route JSON = %#v", route)
	}
}

func TestChannelTestDeliveryRequestPreservesTypedTargetShape(t *testing.T) {
	t.Parallel()

	req := contract.ChannelTestDeliveryRequest{
		Message: "hello",
		Target: contract.ChannelDeliveryTargetInput{
			PeerID:   "peer-1",
			ThreadID: "thread-1",
			GroupID:  "group-1",
			Mode:     channelspkg.DeliveryModeReply,
		},
	}

	mapped, err := req.ToResolveDeliveryTargetRequest("chan-1")
	if err != nil {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v", err)
	}
	if mapped.ChannelInstanceID != "chan-1" || mapped.PeerID != "peer-1" || mapped.ThreadID != "thread-1" || mapped.GroupID != "group-1" || mapped.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("mapped target = %#v", mapped)
	}

	data, err := json.Marshal(contract.ChannelTestDeliveryResponse{
		Status:         "resolved",
		Message:        "hello",
		DeliveryTarget: channelspkg.DeliveryTarget{ChannelInstanceID: "chan-1", PeerID: "peer-1", ThreadID: "thread-1", GroupID: "group-1", Mode: channelspkg.DeliveryModeReply},
	})
	if err != nil {
		t.Fatalf("json.Marshal(response) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	target, ok := got["delivery_target"].(map[string]any)
	if !ok {
		t.Fatalf("delivery_target type = %T, want object", got["delivery_target"])
	}
	if target["peer_id"] != "peer-1" || target["thread_id"] != "thread-1" || target["group_id"] != "group-1" || target["mode"] != string(channelspkg.DeliveryModeReply) {
		t.Fatalf("delivery_target JSON = %#v", target)
	}
}

func TestChannelTestDeliveryRequestRejectsMismatchedInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.ChannelTestDeliveryRequest{
		Target: contract.ChannelDeliveryTargetInput{
			ChannelInstanceID: "chan-2",
			PeerID:            "peer-1",
		},
	}

	if _, err := req.ToResolveDeliveryTargetRequest("chan-1"); !errors.Is(err, contract.ErrChannelInstanceMismatch) {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v, want %v", err, contract.ErrChannelInstanceMismatch)
	}
}

func TestChannelTestDeliveryRequestAcceptsExplicitMatchingInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.ChannelTestDeliveryRequest{
		Target: contract.ChannelDeliveryTargetInput{
			ChannelInstanceID: " chan-1 ",
			PeerID:            " peer-1 ",
			Mode:              "direct",
		},
	}

	mapped, err := req.ToResolveDeliveryTargetRequest(" chan-1 ")
	if err != nil {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v", err)
	}
	if mapped.ChannelInstanceID != "chan-1" || mapped.PeerID != "peer-1" || mapped.Mode != channelspkg.DeliveryModeDirectSend {
		t.Fatalf("mapped target = %#v", mapped)
	}
}

func TestChannelTestDeliveryRequestRejectsBlankInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.ChannelTestDeliveryRequest{
		Target: contract.ChannelDeliveryTargetInput{PeerID: "peer-1"},
	}

	if _, err := req.ToResolveDeliveryTargetRequest("   "); err == nil {
		t.Fatal("ToResolveDeliveryTargetRequest() error = nil, want non-nil")
	}
}

func TestUpdateChannelRequestPreservesOptionalFields(t *testing.T) {
	t.Parallel()

	displayName := "Support Escalations"
	rawDefaults := json.RawMessage(`{"mode":"reply"}`)
	req := contract.UpdateChannelRequest{
		DisplayName:      &displayName,
		RoutingPolicy:    &channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: &rawDefaults,
	}

	mapped, err := req.ToUpdateInstanceRequest("chan-1")
	if err != nil {
		t.Fatalf("ToUpdateInstanceRequest() error = %v", err)
	}
	if mapped.ID != "chan-1" {
		t.Fatalf("mapped.ID = %q, want chan-1", mapped.ID)
	}
	if mapped.DisplayName == nil || *mapped.DisplayName != displayName {
		t.Fatalf("mapped.DisplayName = %#v", mapped.DisplayName)
	}
	if mapped.RoutingPolicy == nil || !mapped.RoutingPolicy.IncludePeer || !mapped.RoutingPolicy.IncludeThread {
		t.Fatalf("mapped.RoutingPolicy = %#v", mapped.RoutingPolicy)
	}
	if mapped.DeliveryDefaults == nil || string(*mapped.DeliveryDefaults) != string(rawDefaults) {
		t.Fatalf("mapped.DeliveryDefaults = %s, want %s", stringValue(mapped.DeliveryDefaults), string(rawDefaults))
	}

	rawDefaults[0] = '['
	if string(*mapped.DeliveryDefaults) != `{"mode":"reply"}` {
		t.Fatalf("mapped.DeliveryDefaults mutated with source slice = %s", string(*mapped.DeliveryDefaults))
	}
}

func TestUpdateChannelRequestRejectsBlankDisplayName(t *testing.T) {
	t.Parallel()

	displayName := "   "
	req := contract.UpdateChannelRequest{DisplayName: &displayName}

	if _, err := req.ToUpdateInstanceRequest("chan-1"); err == nil {
		t.Fatal("ToUpdateInstanceRequest() error = nil, want non-nil")
	}
}

func TestChannelInstanceMismatchErrorSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	err := contract.ErrChannelInstanceMismatch
	if err.Error() != "channel instance id must match request path" {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, contract.ErrChannelInstanceMismatch) {
		t.Fatal("expected errors.Is to match ErrChannelInstanceMismatch")
	}
}

func stringValue(payload *json.RawMessage) string {
	if payload == nil {
		return "<nil>"
	}
	return string(*payload)
}
