package contract_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestCreateBridgeRequestValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  contract.CreateBridgeRequest
	}{
		{
			name: "workspace scope requires workspace id",
			req: contract.CreateBridgeRequest{
				Scope:         bridgepkg.ScopeWorkspace,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			},
		},
		{
			name: "global scope rejects workspace id",
			req: contract.CreateBridgeRequest{
				Scope:         bridgepkg.ScopeGlobal,
				WorkspaceID:   "ws-alpha",
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			},
		},
		{
			name: "routing policy rejects thread without peer or group",
			req: contract.CreateBridgeRequest{
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludeThread: true},
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

func TestCreateBridgeRequestPreservesNormalizedFieldsAndDefaults(t *testing.T) {
	t.Parallel()

	req := contract.CreateBridgeRequest{
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      " ws-alpha ",
		Platform:         " telegram ",
		ExtensionName:    " ext-telegram ",
		DisplayName:      " Support ",
		Enabled:          true,
		Status:           bridgepkg.BridgeStatusDegraded,
		DMPolicy:         bridgepkg.BridgeDMPolicyPairing,
		RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
		ProviderConfig:   contract.BridgeProviderConfigPayload(`{"mode":"bot","tenant":"acme"}`),
		DeliveryDefaults: contract.BridgeDeliveryDefaultsPayload(`{"mode":"reply","peer_id":"peer-1"}`),
		Degradation: &bridgepkg.BridgeDegradation{
			Reason:  bridgepkg.BridgeDegradationReasonRateLimited,
			Message: "provider throttled",
		},
	}

	mapped, err := req.ToCreateInstanceRequest()
	if err != nil {
		t.Fatalf("ToCreateInstanceRequest() error = %v", err)
	}
	if mapped.WorkspaceID != "ws-alpha" || mapped.Platform != "telegram" || mapped.ExtensionName != "ext-telegram" || mapped.DisplayName != "Support" {
		t.Fatalf("mapped request = %#v", mapped)
	}
	if mapped.DMPolicy != bridgepkg.BridgeDMPolicyPairing {
		t.Fatalf("mapped.DMPolicy = %q, want %q", mapped.DMPolicy, bridgepkg.BridgeDMPolicyPairing)
	}
	if string(mapped.ProviderConfig) != `{"mode":"bot","tenant":"acme"}` {
		t.Fatalf("mapped.ProviderConfig = %s", string(mapped.ProviderConfig))
	}
	if string(mapped.DeliveryDefaults) != `{"mode":"reply","peer_id":"peer-1"}` {
		t.Fatalf("mapped.DeliveryDefaults = %s", string(mapped.DeliveryDefaults))
	}
	if mapped.Degradation == nil || mapped.Degradation.Reason != bridgepkg.BridgeDegradationReasonRateLimited {
		t.Fatalf("mapped.Degradation = %#v", mapped.Degradation)
	}

	req.DeliveryDefaults[0] = '['
	req.ProviderConfig[0] = '['
	req.Degradation.Message = "changed"
	if string(mapped.DeliveryDefaults) != `{"mode":"reply","peer_id":"peer-1"}` {
		t.Fatalf("mapped.DeliveryDefaults mutated with source slice = %s", string(mapped.DeliveryDefaults))
	}
	if string(mapped.ProviderConfig) != `{"mode":"bot","tenant":"acme"}` {
		t.Fatalf("mapped.ProviderConfig mutated with source slice = %s", string(mapped.ProviderConfig))
	}
	if mapped.Degradation.Message != "provider throttled" {
		t.Fatalf("mapped.Degradation.Message = %q, want %q", mapped.Degradation.Message, "provider throttled")
	}
}

func TestBridgeRoutesResponseJSONShape(t *testing.T) {
	t.Parallel()

	payload := contract.BridgeRoutesResponse{
		Routes: []bridgepkg.BridgeRoute{
			{
				RoutingKeyHash:   "hash-1",
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-alpha",
				BridgeInstanceID: "brg-1",
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
				GroupID:          "group-1",
				SessionID:        "sess-1",
				AgentName:        "coder",
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

func TestBridgeTestDeliveryRequestPreservesTypedTargetShape(t *testing.T) {
	t.Parallel()

	req := contract.BridgeTestDeliveryRequest{
		Message: "hello",
		Target: contract.BridgeDeliveryTargetInput{
			PeerID:   "peer-1",
			ThreadID: "thread-1",
			GroupID:  "group-1",
			Mode:     bridgepkg.DeliveryModeReply,
		},
	}

	mapped, err := req.ToResolveDeliveryTargetRequest("brg-1")
	if err != nil {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v", err)
	}
	if mapped.BridgeInstanceID != "brg-1" || mapped.PeerID != "peer-1" || mapped.ThreadID != "thread-1" || mapped.GroupID != "group-1" || mapped.Mode != bridgepkg.DeliveryModeReply {
		t.Fatalf("mapped target = %#v", mapped)
	}

	data, err := json.Marshal(contract.BridgeTestDeliveryResponse{
		Status:         "resolved",
		Message:        "hello",
		DeliveryTarget: bridgepkg.DeliveryTarget{BridgeInstanceID: "brg-1", PeerID: "peer-1", ThreadID: "thread-1", GroupID: "group-1", Mode: bridgepkg.DeliveryModeReply},
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
	if target["peer_id"] != "peer-1" || target["thread_id"] != "thread-1" || target["group_id"] != "group-1" || target["mode"] != string(bridgepkg.DeliveryModeReply) {
		t.Fatalf("delivery_target JSON = %#v", target)
	}
}

func TestBridgeTestDeliveryRequestRejectsMismatchedInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.BridgeTestDeliveryRequest{
		Target: contract.BridgeDeliveryTargetInput{
			BridgeInstanceID: "brg-2",
			PeerID:           "peer-1",
		},
	}

	if _, err := req.ToResolveDeliveryTargetRequest("brg-1"); !errors.Is(err, contract.ErrBridgeInstanceMismatch) {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v, want %v", err, contract.ErrBridgeInstanceMismatch)
	}
}

func TestBridgeTestDeliveryRequestAcceptsExplicitMatchingInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.BridgeTestDeliveryRequest{
		Target: contract.BridgeDeliveryTargetInput{
			BridgeInstanceID: " brg-1 ",
			PeerID:           " peer-1 ",
			Mode:             "direct",
		},
	}

	mapped, err := req.ToResolveDeliveryTargetRequest(" brg-1 ")
	if err != nil {
		t.Fatalf("ToResolveDeliveryTargetRequest() error = %v", err)
	}
	if mapped.BridgeInstanceID != "brg-1" || mapped.PeerID != "peer-1" || mapped.Mode != bridgepkg.DeliveryModeDirectSend {
		t.Fatalf("mapped target = %#v", mapped)
	}
}

func TestBridgeTestDeliveryRequestRejectsBlankInstanceID(t *testing.T) {
	t.Parallel()

	req := contract.BridgeTestDeliveryRequest{
		Target: contract.BridgeDeliveryTargetInput{PeerID: "peer-1"},
	}

	if _, err := req.ToResolveDeliveryTargetRequest("   "); err == nil {
		t.Fatal("ToResolveDeliveryTargetRequest() error = nil, want non-nil")
	}
}

func TestUpdateBridgeRequestPreservesOptionalFields(t *testing.T) {
	t.Parallel()

	displayName := "Support Escalations"
	dmPolicy := bridgepkg.BridgeDMPolicyAllowlist
	rawProviderConfig := contract.BridgeProviderConfigPayload(`{"mode":"bot","tenant":"ws-alpha"}`)
	rawDefaults := contract.BridgeDeliveryDefaultsPayload(`{"mode":"reply","peer_id":"peer-1"}`)
	req := contract.UpdateBridgeRequest{
		DisplayName:      &displayName,
		DMPolicy:         &dmPolicy,
		RoutingPolicy:    &bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		ProviderConfig:   &rawProviderConfig,
		DeliveryDefaults: &rawDefaults,
		Degradation: &bridgepkg.BridgeDegradation{
			Reason: bridgepkg.BridgeDegradationReasonAuthFailed,
		},
	}

	mapped, err := req.ToUpdateInstanceRequest("brg-1")
	if err != nil {
		t.Fatalf("ToUpdateInstanceRequest() error = %v", err)
	}
	if mapped.ID != "brg-1" {
		t.Fatalf("mapped.ID = %q, want brg-1", mapped.ID)
	}
	if mapped.DisplayName == nil || *mapped.DisplayName != displayName {
		t.Fatalf("mapped.DisplayName = %#v", mapped.DisplayName)
	}
	if mapped.DMPolicy == nil || *mapped.DMPolicy != bridgepkg.BridgeDMPolicyAllowlist {
		t.Fatalf("mapped.DMPolicy = %#v", mapped.DMPolicy)
	}
	if mapped.RoutingPolicy == nil || !mapped.RoutingPolicy.IncludePeer || !mapped.RoutingPolicy.IncludeThread {
		t.Fatalf("mapped.RoutingPolicy = %#v", mapped.RoutingPolicy)
	}
	if mapped.ProviderConfig == nil || string(*mapped.ProviderConfig) != string(rawProviderConfig) {
		t.Fatalf("mapped.ProviderConfig = %s, want %s", stringValue(mapped.ProviderConfig), string(rawProviderConfig))
	}
	if mapped.DeliveryDefaults == nil || string(*mapped.DeliveryDefaults) != string(rawDefaults) {
		t.Fatalf("mapped.DeliveryDefaults = %s, want %s", stringValue(mapped.DeliveryDefaults), string(rawDefaults))
	}
	if mapped.Degradation == nil || mapped.Degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("mapped.Degradation = %#v", mapped.Degradation)
	}

	rawProviderConfig[0] = '['
	rawDefaults[0] = '['
	if string(*mapped.ProviderConfig) != `{"mode":"bot","tenant":"ws-alpha"}` {
		t.Fatalf("mapped.ProviderConfig mutated with source slice = %s", string(*mapped.ProviderConfig))
	}
	if string(*mapped.DeliveryDefaults) != `{"mode":"reply","peer_id":"peer-1"}` {
		t.Fatalf("mapped.DeliveryDefaults mutated with source slice = %s", string(*mapped.DeliveryDefaults))
	}
}

func TestBridgeRequestsKeepProviderConfigDistinctFromDeliveryDefaults(t *testing.T) {
	t.Parallel()

	createReq := contract.CreateBridgeRequest{
		Scope:            bridgepkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "ext-telegram",
		DisplayName:      "Support",
		Enabled:          true,
		Status:           bridgepkg.BridgeStatusReady,
		RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
		ProviderConfig:   contract.BridgeProviderConfigPayload(`{"mode":"bot","tenant":"acme"}`),
		DeliveryDefaults: contract.BridgeDeliveryDefaultsPayload(`{"peer_id":"peer-default","mode":"reply"}`),
	}

	createMapped, err := createReq.ToCreateInstanceRequest()
	if err != nil {
		t.Fatalf("ToCreateInstanceRequest() error = %v", err)
	}
	if got, want := string(createMapped.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
		t.Fatalf("createMapped.ProviderConfig = %s, want %s", got, want)
	}
	if got, want := string(createMapped.DeliveryDefaults), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
		t.Fatalf("createMapped.DeliveryDefaults = %s, want %s", got, want)
	}

	updateProviderConfig := contract.BridgeProviderConfigPayload(`{"mode":"comments"}`)
	updateDeliveryDefaults := contract.BridgeDeliveryDefaultsPayload(`{"group_id":"ops","mode":"direct-send"}`)
	updateReq := contract.UpdateBridgeRequest{
		ProviderConfig:   &updateProviderConfig,
		DeliveryDefaults: &updateDeliveryDefaults,
	}

	updateMapped, err := updateReq.ToUpdateInstanceRequest("brg-1")
	if err != nil {
		t.Fatalf("ToUpdateInstanceRequest() error = %v", err)
	}
	if updateMapped.ProviderConfig == nil || string(*updateMapped.ProviderConfig) != `{"mode":"comments"}` {
		t.Fatalf("updateMapped.ProviderConfig = %s", stringValue(updateMapped.ProviderConfig))
	}
	if updateMapped.DeliveryDefaults == nil || string(*updateMapped.DeliveryDefaults) != `{"group_id":"ops","mode":"direct-send"}` {
		t.Fatalf("updateMapped.DeliveryDefaults = %s", stringValue(updateMapped.DeliveryDefaults))
	}
}

func TestBridgeRequestsRejectUnsupportedProviderConfigAndDeliveryDefaultsShapes(t *testing.T) {
	t.Parallel()

	badProviderConfig := contract.CreateBridgeRequest{
		Scope:          bridgepkg.ScopeGlobal,
		Platform:       "telegram",
		ExtensionName:  "ext-telegram",
		DisplayName:    "Support",
		Enabled:        true,
		Status:         bridgepkg.BridgeStatusReady,
		RoutingPolicy:  bridgepkg.RoutingPolicy{IncludePeer: true},
		ProviderConfig: contract.BridgeProviderConfigPayload(`"bot"`),
	}
	if _, err := badProviderConfig.ToCreateInstanceRequest(); err == nil {
		t.Fatal("ToCreateInstanceRequest(provider_config string) error = nil, want non-nil")
	}

	badDefaults := contract.UpdateBridgeRequest{
		DeliveryDefaults: ptr(contract.BridgeDeliveryDefaultsPayload(`{"mode":"reply","parse_mode":"markdown"}`)),
	}
	if _, err := badDefaults.ToUpdateInstanceRequest("brg-1"); err == nil {
		t.Fatal("ToUpdateInstanceRequest(delivery defaults extra field) error = nil, want non-nil")
	}
}

func TestUpdateBridgeRequestRejectsBlankDisplayName(t *testing.T) {
	t.Parallel()

	displayName := "   "
	req := contract.UpdateBridgeRequest{DisplayName: &displayName}

	if _, err := req.ToUpdateInstanceRequest("brg-1"); err == nil {
		t.Fatal("ToUpdateInstanceRequest() error = nil, want non-nil")
	}
}

func TestBridgeInstanceMismatchErrorSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	err := contract.ErrBridgeInstanceMismatch
	if err.Error() != "bridge instance id must match request path" {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, contract.ErrBridgeInstanceMismatch) {
		t.Fatal("expected errors.Is to match ErrBridgeInstanceMismatch")
	}
}

func TestBridgeJSONPayloadsMarshalAndUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		validate func(*testing.T, []byte)
	}{
		{
			name: "provider config object round-trips compactly",
			raw:  "{\n  \"tenant\": \"acme\",\n  \"mode\": \"bot\"\n}",
			validate: func(t *testing.T, encoded []byte) {
				t.Helper()
				if got, want := string(encoded), `{"tenant":"acme","mode":"bot"}`; got != want {
					t.Fatalf("provider config encoded = %s, want %s", got, want)
				}
			},
		},
		{
			name: "provider config blank marshals to null",
			raw:  "",
			validate: func(t *testing.T, encoded []byte) {
				t.Helper()
				if got, want := string(encoded), "null"; got != want {
					t.Fatalf("provider config encoded = %s, want %s", got, want)
				}
			},
		},
		{
			name: "delivery defaults object round-trips compactly",
			raw:  "{\n  \"peer_id\": \"peer-1\",\n  \"mode\": \"reply\"\n}",
			validate: func(t *testing.T, encoded []byte) {
				t.Helper()
				if got, want := string(encoded), `{"peer_id":"peer-1","mode":"reply"}`; got != want {
					t.Fatalf("delivery defaults encoded = %s, want %s", got, want)
				}
			},
		},
		{
			name: "delivery defaults null stays null",
			raw:  "null",
			validate: func(t *testing.T, encoded []byte) {
				t.Helper()
				if got, want := string(encoded), "null"; got != want {
					t.Fatalf("delivery defaults encoded = %s, want %s", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var providerPayload contract.BridgeProviderConfigPayload
			if err := providerPayload.UnmarshalJSON([]byte(tt.raw)); err == nil {
				encoded, marshalErr := providerPayload.MarshalJSON()
				if marshalErr != nil {
					t.Fatalf("provider MarshalJSON() error = %v", marshalErr)
				}
				if tt.name == "provider config object round-trips compactly" || tt.name == "provider config blank marshals to null" {
					tt.validate(t, encoded)
				}
			}

			var deliveryPayload contract.BridgeDeliveryDefaultsPayload
			if err := deliveryPayload.UnmarshalJSON([]byte(tt.raw)); err == nil {
				encoded, marshalErr := deliveryPayload.MarshalJSON()
				if marshalErr != nil {
					t.Fatalf("delivery MarshalJSON() error = %v", marshalErr)
				}
				if tt.name == "delivery defaults object round-trips compactly" || tt.name == "delivery defaults null stays null" {
					tt.validate(t, encoded)
				}
			}
		})
	}
}

func TestBridgeJSONPayloadsRejectInvalidShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  string
		raw     string
		wantErr string
	}{
		{
			name:    "provider config rejects scalar",
			target:  "provider",
			raw:     `"bot"`,
			wantErr: "bridge provider config must be a JSON object or null",
		},
		{
			name:    "provider config rejects invalid json",
			target:  "provider",
			raw:     "{not-json",
			wantErr: "bridge provider config must be valid JSON",
		},
		{
			name:    "delivery defaults rejects unsupported field",
			target:  "delivery",
			raw:     `{"mode":"reply","parse_mode":"markdown"}`,
			wantErr: `bridge delivery defaults field "parse_mode" is not supported`,
		},
		{
			name:    "delivery defaults rejects thread without peer or group",
			target:  "delivery",
			raw:     `{"thread_id":"thr-1"}`,
			wantErr: "bridge delivery defaults thread_id requires peer_id or group_id",
		},
		{
			name:    "delivery defaults rejects non-string field",
			target:  "delivery",
			raw:     `{"peer_id":7}`,
			wantErr: `bridge delivery defaults field "peer_id" must be a string`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch tt.target {
			case "provider":
				var payload contract.BridgeProviderConfigPayload
				if err := payload.UnmarshalJSON([]byte(tt.raw)); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("provider UnmarshalJSON(%s) error = %v, want substring %q", tt.raw, err, tt.wantErr)
				}
			case "delivery":
				var payload contract.BridgeDeliveryDefaultsPayload
				if err := payload.UnmarshalJSON([]byte(tt.raw)); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("delivery UnmarshalJSON(%s) error = %v, want substring %q", tt.raw, err, tt.wantErr)
				}
			default:
				t.Fatalf("unknown target %q", tt.target)
			}
		})
	}
}

func TestPutBridgeSecretBindingRequestValidation(t *testing.T) {
	t.Parallel()

	req := contract.PutBridgeSecretBindingRequest{
		VaultRef: " vault://telegram/bot ",
		Kind:     " env ",
	}

	binding, err := req.ToBridgeSecretBinding(" brg-1 ", " bot_token ")
	if err != nil {
		t.Fatalf("ToBridgeSecretBinding() error = %v", err)
	}
	if binding.BridgeInstanceID != "brg-1" || binding.BindingName != "bot_token" || binding.VaultRef != "vault://telegram/bot" || binding.Kind != "env" {
		t.Fatalf("binding = %#v", binding)
	}

	req.Kind = "   "
	if _, err := req.ToBridgeSecretBinding("brg-1", "bot_token"); err == nil {
		t.Fatal("ToBridgeSecretBinding(blank kind) error = nil, want non-nil")
	}
}

func stringValue(payload *json.RawMessage) string {
	if payload == nil {
		return "<nil>"
	}
	return string(*payload)
}

func ptr[T any](value T) *T {
	return &value
}
