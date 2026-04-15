package bridges

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateScopeWorkspaceID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scope       Scope
		workspaceID string
		wantErr     bool
	}{
		{name: "global without workspace", scope: ScopeGlobal},
		{name: "workspace with workspace", scope: ScopeWorkspace, workspaceID: "ws-1"},
		{name: "workspace missing workspace id", scope: ScopeWorkspace, wantErr: true},
		{name: "global with workspace id", scope: ScopeGlobal, workspaceID: "ws-1", wantErr: true},
		{name: "unsupported scope", scope: "tenant", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateScopeWorkspaceID(tt.scope, tt.workspaceID)
			if tt.wantErr && err == nil {
				t.Fatal("ValidateScopeWorkspaceID() error = nil, want non-nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateScopeWorkspaceID() error = %v, want nil", err)
			}
		})
	}
}

func TestBridgeStatusAndRoutingPolicyValidation(t *testing.T) {
	t.Parallel()

	instance := BridgeInstance{
		ID:            "brg-1",
		Scope:         ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "telegram-adapter",
		DisplayName:   "Telegram",
		Enabled:       true,
		Status:        "bogus",
		RoutingPolicy: RoutingPolicy{IncludePeer: true},
	}
	if err := instance.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(invalid status) error = nil, want non-nil")
	}

	instance.Status = BridgeStatusReady
	instance.RoutingPolicy = RoutingPolicy{IncludeThread: true}
	if err := instance.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(invalid routing policy) error = nil, want non-nil")
	}

	instance.RoutingPolicy = RoutingPolicy{IncludePeer: true, IncludeThread: true}
	if err := instance.Validate(); err != nil {
		t.Fatalf("BridgeInstance.Validate(valid) error = %v", err)
	}
}

func TestRoutingKeyHashStable(t *testing.T) {
	t.Parallel()

	first := RoutingKey{
		Scope:            ScopeWorkspace,
		WorkspaceID:      " ws-1 ",
		BridgeInstanceID: " brg-1 ",
		PeerID:           "peer-1",
		ThreadID:         " thread-1 ",
	}
	second := RoutingKey{
		Scope:            "workspace",
		WorkspaceID:      "ws-1",
		BridgeInstanceID: "brg-1",
		PeerID:           "peer-1",
		ThreadID:         "thread-1",
	}

	firstSerialized, err := first.Serialize()
	if err != nil {
		t.Fatalf("first.Serialize() error = %v", err)
	}
	secondSerialized, err := second.Serialize()
	if err != nil {
		t.Fatalf("second.Serialize() error = %v", err)
	}
	if firstSerialized != secondSerialized {
		t.Fatalf("Serialize() mismatch = %q vs %q", firstSerialized, secondSerialized)
	}

	firstHash, err := first.Hash()
	if err != nil {
		t.Fatalf("first.Hash() error = %v", err)
	}
	secondHash, err := second.Hash()
	if err != nil {
		t.Fatalf("second.Hash() error = %v", err)
	}
	if firstHash != secondHash {
		t.Fatalf("Hash() mismatch = %q vs %q", firstHash, secondHash)
	}
}

func TestBridgeSecretBindingValidation(t *testing.T) {
	t.Parallel()

	valid := BridgeSecretBinding{
		BridgeInstanceID: "brg-1",
		BindingName:      "bot_token",
		VaultRef:         "vault://bot-token",
		Kind:             "token",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("BridgeSecretBinding.Validate(valid) error = %v", err)
	}

	invalidName := valid
	invalidName.BindingName = " "
	if err := invalidName.Validate(); err == nil {
		t.Fatal("BridgeSecretBinding.Validate(empty name) error = nil, want non-nil")
	}

	invalidVault := valid
	invalidVault.VaultRef = ""
	if err := invalidVault.Validate(); err == nil {
		t.Fatal("BridgeSecretBinding.Validate(empty vault ref) error = nil, want non-nil")
	}
}

func TestBridgeRouteCanonicalizeAndDedupValidation(t *testing.T) {
	t.Parallel()

	route := BridgeRoute{
		Scope:            ScopeWorkspace,
		WorkspaceID:      "ws-1",
		BridgeInstanceID: "brg-1",
		PeerID:           "peer-1",
		SessionID:        "sess-1",
		AgentName:        "coder",
	}
	canonical, err := route.Canonicalize()
	if err != nil {
		t.Fatalf("BridgeRoute.Canonicalize() error = %v", err)
	}
	if canonical.RoutingKeyHash == "" {
		t.Fatal("BridgeRoute.Canonicalize() routing key hash = empty, want non-empty")
	}

	record := IngestDedupRecord{
		IdempotencyKey:   "idem-1",
		BridgeInstanceID: "brg-1",
		ReceivedAt:       time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		ExpiresAt:        time.Date(2026, 4, 10, 10, 5, 0, 0, time.UTC),
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("IngestDedupRecord.Validate(valid) error = %v", err)
	}

	record.ExpiresAt = record.ReceivedAt
	if err := record.Validate(); err == nil {
		t.Fatal("IngestDedupRecord.Validate(equal expiry) error = nil, want non-nil")
	}
}

func TestBridgeInstanceValidateDeliveryDefaultsJSON(t *testing.T) {
	t.Parallel()

	instance := BridgeInstance{
		ID:               "brg-json",
		Scope:            ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "telegram-adapter",
		DisplayName:      "JSON Telegram",
		Enabled:          true,
		Status:           BridgeStatusReady,
		RoutingPolicy:    RoutingPolicy{IncludePeer: true},
		DeliveryDefaults: []byte(`{"parse_mode":"markdown"}`),
	}
	if err := instance.Validate(); err != nil {
		t.Fatalf("BridgeInstance.Validate(valid json) error = %v", err)
	}

	instance.DeliveryDefaults = []byte(`{`)
	if err := instance.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(invalid json) error = nil, want non-nil")
	}
}

func TestBridgeInstanceValidateProviderConfigDMPolicyAndDegradation(t *testing.T) {
	t.Parallel()

	base := BridgeInstance{
		ID:               "brg-provider",
		Scope:            ScopeGlobal,
		Platform:         "slack",
		ExtensionName:    "slack-adapter",
		DisplayName:      "Slack Provider",
		Enabled:          true,
		Status:           BridgeStatusReady,
		RoutingPolicy:    RoutingPolicy{IncludePeer: true},
		ProviderConfig:   json.RawMessage(`{"mode":"bot","tenant":"acme"}`),
		DeliveryDefaults: json.RawMessage(`{"mode":"reply","thread_id":"thread-1"}`),
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("BridgeInstance.Validate(valid provider config) error = %v", err)
	}

	invalidProviderConfig := base
	invalidProviderConfig.ProviderConfig = json.RawMessage(`{`)
	if err := invalidProviderConfig.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(invalid provider config) error = nil, want non-nil")
	}

	invalidDMPolicy := base
	invalidDMPolicy.DMPolicy = "disabled"
	if err := invalidDMPolicy.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(invalid dm policy) error = nil, want non-nil")
	}

	validDegradation := base
	validDegradation.Status = BridgeStatusDegraded
	validDegradation.Degradation = &BridgeDegradation{
		Reason:  BridgeDegradationReasonRateLimited,
		Message: "provider API is throttling",
	}
	if err := validDegradation.Validate(); err != nil {
		t.Fatalf("BridgeInstance.Validate(valid degradation) error = %v", err)
	}

	invalidDegradation := validDegradation
	invalidDegradation.Degradation = &BridgeDegradation{Message: "missing reason"}
	if err := invalidDegradation.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(missing degradation reason) error = nil, want non-nil")
	}

	readyWithDegradation := base
	readyWithDegradation.Degradation = &BridgeDegradation{Reason: BridgeDegradationReasonAuthFailed}
	if err := readyWithDegradation.Validate(); err == nil {
		t.Fatal("BridgeInstance.Validate(ready with degradation) error = nil, want non-nil")
	}
}

func TestBridgeSecretSlotAndConfigSchemaValidation(t *testing.T) {
	t.Parallel()

	slot := BridgeSecretSlot{Name: "bot_token", Description: "Bot token", Required: true}
	if err := slot.Validate(); err != nil {
		t.Fatalf("BridgeSecretSlot.Validate(valid) error = %v", err)
	}
	if err := (BridgeSecretSlot{}).Validate(); err == nil {
		t.Fatal("BridgeSecretSlot.Validate(empty) error = nil, want non-nil")
	}

	schema := BridgeProviderConfigSchema{Schema: "agh.bridge.slack", Version: "v1"}
	if err := schema.Validate(); err != nil {
		t.Fatalf("BridgeProviderConfigSchema.Validate(valid) error = %v", err)
	}
	if err := (BridgeProviderConfigSchema{}).Validate(); err != nil {
		t.Fatalf("BridgeProviderConfigSchema.Validate(zero) error = %v", err)
	}
}

func TestDeliveryTargetEnvelopeAndEventValidation(t *testing.T) {
	t.Parallel()

	target := DeliveryTarget{BridgeInstanceID: "brg-1", PeerID: "peer-1", Mode: "direct"}
	if target.IsZero() {
		t.Fatal("DeliveryTarget.IsZero() = true, want false")
	}
	if err := target.Validate(); err != nil {
		t.Fatalf("DeliveryTarget.Validate(valid) error = %v", err)
	}
	if err := (DeliveryTarget{}).Validate(); err == nil {
		t.Fatal("DeliveryTarget.Validate(empty) error = nil, want non-nil")
	}
	if !(DeliveryTarget{}).IsZero() {
		t.Fatal("DeliveryTarget.IsZero(empty) = false, want true")
	}

	envelope := InboundMessageEnvelope{
		BridgeInstanceID:  "brg-1",
		Scope:             ScopeWorkspace,
		WorkspaceID:       "ws-1",
		PeerID:            "peer-1",
		PlatformMessageID: "msg-1",
		ReceivedAt:        time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		Sender:            MessageSender{ID: "user-1", DisplayName: "Alice"},
		Content:           MessageContent{Text: "hello"},
		IdempotencyKey:    "idem-1",
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("InboundMessageEnvelope.Validate(valid) error = %v", err)
	}
	envelope.IdempotencyKey = ""
	if err := envelope.Validate(); err == nil {
		t.Fatal("InboundMessageEnvelope.Validate(empty idempotency key) error = nil, want non-nil")
	}

	event := DeliveryEvent{
		DeliveryID:       "deliv-1",
		BridgeInstanceID: "brg-1",
		RoutingKey: RoutingKey{
			Scope:            ScopeWorkspace,
			WorkspaceID:      "ws-1",
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
		},
		DeliveryTarget: target,
		Seq:            1,
		EventType:      DeliveryEventTypeStart,
		Content:        MessageContent{Text: "hello"},
		Metadata:       []byte(`{"remote":true}`),
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("DeliveryEvent.Validate(valid) error = %v", err)
	}

	event.DeliveryTarget.BridgeInstanceID = "brg-2"
	if err := event.Validate(); err == nil {
		t.Fatal("DeliveryEvent.Validate(mismatched target instance) error = nil, want non-nil")
	}

	event.DeliveryTarget.BridgeInstanceID = "brg-1"
	event.Metadata = []byte(`{`)
	if err := event.Validate(); err == nil {
		t.Fatal("DeliveryEvent.Validate(invalid metadata) error = nil, want non-nil")
	}
}

func TestInboundMessageEnvelopeNormalizeClonesAttachments(t *testing.T) {
	t.Parallel()

	envelope := InboundMessageEnvelope{
		BridgeInstanceID:  " brg-1 ",
		Scope:             ScopeWorkspace,
		WorkspaceID:       " ws-1 ",
		PeerID:            " peer-1 ",
		PlatformMessageID: " msg-1 ",
		ReceivedAt:        time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		Sender:            MessageSender{ID: " user-1 ", DisplayName: " Alice "},
		Content:           MessageContent{Text: " hello "},
		Attachments: []MessageAttachment{{
			ID:       " att-1 ",
			Name:     " image.png ",
			MIMEType: " image/png ",
			URL:      " https://example.test/image.png ",
		}},
		IdempotencyKey: " idem-1 ",
	}

	if err := envelope.Validate(); err != nil {
		t.Fatalf("InboundMessageEnvelope.Validate() error = %v", err)
	}
	if got := envelope.Attachments[0].ID; got != " att-1 " {
		t.Fatalf("Validate() mutated original attachment id to %q", got)
	}

	normalized := envelope.normalize()
	if got := normalized.Attachments[0].ID; got != "att-1" {
		t.Fatalf("normalized attachment id = %q, want trimmed value", got)
	}
	normalized.Attachments[0].ID = "changed"
	if got := envelope.Attachments[0].ID; got != " att-1 " {
		t.Fatalf("normalize() mutated original attachment id to %q", got)
	}
}

func TestBridgeRouteValidateHashMismatch(t *testing.T) {
	t.Parallel()

	route := BridgeRoute{
		RoutingKeyHash:   "wrong",
		Scope:            ScopeGlobal,
		BridgeInstanceID: "brg-1",
		PeerID:           "peer-1",
		SessionID:        "sess-1",
		AgentName:        "coder",
	}
	if err := route.Validate(); err == nil {
		t.Fatal("BridgeRoute.Validate(hash mismatch) error = nil, want non-nil")
	}
}
