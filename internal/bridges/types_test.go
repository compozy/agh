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
		SecretRef:        "vault:bridges/brg-1/bot-token",
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
	invalidVault.SecretRef = ""
	if err := invalidVault.Validate(); err == nil {
		t.Fatal("BridgeSecretBinding.Validate(empty secret ref) error = nil, want non-nil")
	}

	envRef := valid
	envRef.SecretRef = "env:TG_TOKEN"
	if err := envRef.Validate(); err == nil {
		t.Fatal("BridgeSecretBinding.Validate(env ref) error = nil, want non-nil")
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
		EventFamily:       InboundEventFamilyMessage,
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
		DeliveryTarget:   target,
		Seq:              1,
		EventType:        DeliveryEventTypeStart,
		Content:          MessageContent{Text: "hello"},
		ProviderMetadata: []byte(`{"remote":true}`),
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("DeliveryEvent.Validate(valid) error = %v", err)
	}

	event.DeliveryTarget.BridgeInstanceID = "brg-2"
	if err := event.Validate(); err == nil {
		t.Fatal("DeliveryEvent.Validate(mismatched target instance) error = nil, want non-nil")
	}

	event.DeliveryTarget.BridgeInstanceID = "brg-1"
	event.ProviderMetadata = []byte(`{`)
	if err := event.Validate(); err == nil {
		t.Fatal("DeliveryEvent.Validate(invalid provider metadata) error = nil, want non-nil")
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
		EventFamily:       InboundEventFamilyMessage,
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

func TestInboundMessageEnvelopeNetworkConversationMapping(t *testing.T) {
	t.Parallel()

	base := InboundMessageEnvelope{
		BridgeInstanceID:  "brg-1",
		Scope:             ScopeWorkspace,
		WorkspaceID:       "ws-1",
		PeerID:            "provider-peer",
		ThreadID:          "provider-thread",
		PlatformMessageID: "msg-1",
		ReceivedAt:        time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		Sender:            MessageSender{ID: "user-1", DisplayName: "Alice"},
		Content:           MessageContent{Text: "hello"},
		EventFamily:       InboundEventFamilyMessage,
		IdempotencyKey:    "idem-1",
	}

	t.Run("ShouldNotInferAGHThreadFromProviderThreadID", func(t *testing.T) {
		t.Parallel()

		ref, ok, err := base.NetworkConversationRef()
		if err != nil {
			t.Fatalf("NetworkConversationRef() error = %v", err)
		}
		if ok {
			t.Fatalf("NetworkConversationRef() ok = true with ref %#v, want false", ref)
		}
	})

	t.Run("ShouldReturnExplicitThreadConversationRef", func(t *testing.T) {
		t.Parallel()

		envelope := base
		envelope.Conversation = &NetworkConversationRef{
			Channel:  " builders ",
			Surface:  NetworkConversationSurfaceThread,
			ThreadID: " thread_alpha01 ",
			WorkID:   " work-alpha ",
		}
		ref, ok, err := envelope.NetworkConversationRef()
		if err != nil {
			t.Fatalf("NetworkConversationRef() error = %v", err)
		}
		if !ok {
			t.Fatal("NetworkConversationRef() ok = false, want true")
		}
		if ref.Channel != "builders" || ref.Surface != NetworkConversationSurfaceThread ||
			ref.ThreadID != "thread_alpha01" || ref.WorkID != "work-alpha" {
			t.Fatalf("NetworkConversationRef() = %#v, want trimmed thread mapping", ref)
		}
		if ref.ThreadID == envelope.ThreadID {
			t.Fatalf("AGH thread_id = provider thread_id %q, want separate explicit mapping", ref.ThreadID)
		}
	})

	t.Run("ShouldValidateDirectConversationRefFromResolverOutput", func(t *testing.T) {
		t.Parallel()

		envelope := base
		envelope.Conversation = &NetworkConversationRef{
			Channel:  "builders",
			Surface:  NetworkConversationSurfaceDirect,
			DirectID: "direct_0123456789abcdef0123456789abcdef",
			ThreadID: "thread_alpha01",
		}
		if err := envelope.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want direct/thread collision rejection")
		}

		envelope.Conversation.ThreadID = ""
		if err := envelope.Validate(); err != nil {
			t.Fatalf("Validate(valid direct conversation) error = %v", err)
		}
		ref, ok, err := envelope.NetworkConversationRef()
		if err != nil {
			t.Fatalf("NetworkConversationRef(valid direct) error = %v", err)
		}
		if !ok || ref.DirectID != "direct_0123456789abcdef0123456789abcdef" {
			t.Fatalf("NetworkConversationRef(valid direct) = %#v ok=%v, want direct id", ref, ok)
		}
	})
}

func TestInboundMessageEnvelopeValidatesTypedInteractionFamilies(t *testing.T) {
	t.Parallel()

	base := InboundMessageEnvelope{
		BridgeInstanceID: "brg-1",
		Scope:            ScopeWorkspace,
		WorkspaceID:      "ws-1",
		PeerID:           "peer-1",
		ThreadID:         "thread-1",
		ReceivedAt:       time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		Sender:           MessageSender{ID: "user-1", DisplayName: "Alice"},
		IdempotencyKey:   "idem-1",
	}

	t.Run("command", func(t *testing.T) {
		event := base
		event.EventFamily = InboundEventFamilyCommand
		event.Command = &InboundCommand{Command: "/help", Text: "bridge"}
		if err := event.Validate(); err != nil {
			t.Fatalf("command Validate() error = %v", err)
		}

		event.Command = &InboundCommand{}
		if err := event.Validate(); err == nil {
			t.Fatal("command Validate() error = nil, want non-nil")
		}
	})

	t.Run("action", func(t *testing.T) {
		event := base
		event.EventFamily = InboundEventFamilyAction
		event.Action = &InboundAction{ActionID: "approve", MessageID: "msg-1", Value: "run-1"}
		if err := event.Validate(); err != nil {
			t.Fatalf("action Validate() error = %v", err)
		}

		event.Action = &InboundAction{}
		if err := event.Validate(); err == nil {
			t.Fatal("action Validate() error = nil, want non-nil")
		}
	})

	t.Run("reaction", func(t *testing.T) {
		event := base
		event.EventFamily = InboundEventFamilyReaction
		event.Reaction = &InboundReaction{MessageID: "msg-1", Emoji: "thumbs_up", Added: true}
		if err := event.Validate(); err != nil {
			t.Fatalf("reaction Validate() error = %v", err)
		}

		event.Reaction = &InboundReaction{Emoji: "thumbs_up", Added: true}
		if err := event.Validate(); err == nil {
			t.Fatal("reaction Validate() error = nil, want non-nil")
		}
	})
}

func TestInboundMessageEnvelopeRejectsUnsupportedFamilyCombinations(t *testing.T) {
	t.Parallel()

	event := InboundMessageEnvelope{
		BridgeInstanceID:  "brg-1",
		Scope:             ScopeWorkspace,
		WorkspaceID:       "ws-1",
		PeerID:            "peer-1",
		PlatformMessageID: "msg-1",
		ReceivedAt:        time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		EventFamily:       InboundEventFamilyCommand,
		Command:           &InboundCommand{Command: "/help"},
		Content:           MessageContent{Text: "should-not-be-here"},
		IdempotencyKey:    "idem-1",
	}

	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want command/message combination rejection")
	}
}

func TestDeliveryEventValidatesEditAndDeleteSemantics(t *testing.T) {
	t.Parallel()

	base := DeliveryEvent{
		DeliveryID:       "del-1",
		BridgeInstanceID: "brg-1",
		RoutingKey: RoutingKey{
			Scope:            ScopeWorkspace,
			WorkspaceID:      "ws-1",
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
		},
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
			Mode:             DeliveryModeReply,
		},
		Seq:       2,
		EventType: DeliveryEventTypeFinal,
		Content:   MessageContent{Text: "updated"},
		Final:     true,
	}

	edit := base
	edit.Operation = DeliveryOperationEdit
	edit.Reference = &DeliveryMessageReference{RemoteMessageID: "remote-1"}
	if err := edit.Validate(); err != nil {
		t.Fatalf("edit Validate() error = %v", err)
	}

	edit.Reference = nil
	if err := edit.Validate(); err == nil {
		t.Fatal("edit Validate() error = nil, want reference validation")
	}

	deleteEvent := base
	deleteEvent.EventType = DeliveryEventTypeDelete
	deleteEvent.Operation = DeliveryOperationDelete
	deleteEvent.Reference = &DeliveryMessageReference{DeliveryID: "del-prev"}
	deleteEvent.Content = MessageContent{}
	if err := deleteEvent.Validate(); err != nil {
		t.Fatalf("delete Validate() error = %v", err)
	}

	deleteEvent.Content = MessageContent{Text: "not allowed"}
	if err := deleteEvent.Validate(); err == nil {
		t.Fatal("delete Validate() error = nil, want content rejection")
	}
}

func TestInboundProviderMetadataRoundTripKeepsFamilySelection(t *testing.T) {
	t.Parallel()

	event := InboundMessageEnvelope{
		BridgeInstanceID: "brg-1",
		Scope:            ScopeWorkspace,
		WorkspaceID:      "ws-1",
		PeerID:           "peer-1",
		ReceivedAt:       time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		EventFamily:      InboundEventFamilyAction,
		Action:           &InboundAction{ActionID: "approve", MessageID: "msg-1"},
		ProviderMetadata: json.RawMessage(`{"provider":"slack","raw_action_id":"A123"}`),
		IdempotencyKey:   "idem-1",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded InboundMessageEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got, want := decoded.EventFamily, InboundEventFamilyAction; got != want {
		t.Fatalf("decoded.EventFamily = %q, want %q", got, want)
	}
	if decoded.Action == nil || decoded.Command != nil || decoded.Reaction != nil {
		t.Fatalf("decoded interaction family = %#v, want action only", decoded)
	}
	if got, want := string(decoded.ProviderMetadata), `{"provider":"slack","raw_action_id":"A123"}`; got != want {
		t.Fatalf("decoded.ProviderMetadata = %s, want %s", got, want)
	}
}

func TestDeliveryEventRejectsInvalidTypedPayloadCombinations(t *testing.T) {
	t.Parallel()

	base := DeliveryEvent{
		DeliveryID:       "del-typed",
		BridgeInstanceID: "brg-1",
		RoutingKey: RoutingKey{
			Scope:            ScopeWorkspace,
			WorkspaceID:      "ws-1",
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
		},
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
			Mode:             DeliveryModeReply,
		},
		Seq:   3,
		Final: true,
	}

	t.Run("error event requires typed error payload", func(t *testing.T) {
		event := base
		event.EventType = DeliveryEventTypeError
		if err := event.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want typed error validation")
		}
	})

	t.Run("resume event requires typed resume payload", func(t *testing.T) {
		event := base
		event.EventType = DeliveryEventTypeResume
		event.Final = false
		if err := event.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want typed resume validation")
		}
	})

	t.Run("delete event requires delete operation", func(t *testing.T) {
		event := base
		event.EventType = DeliveryEventTypeDelete
		event.Operation = DeliveryOperationPost
		if err := event.Validate(); err == nil {
			t.Fatal("Validate() error = nil, want delete operation validation")
		}
	})
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
