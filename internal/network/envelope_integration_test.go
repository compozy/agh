//go:build integration

package network

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestProtocolFixturesRoundTripWithoutSemanticDrift(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	opts := ValidateOptions{Now: now, MaxReplayAge: DefaultMaxReplayAge}

	fixtures := []struct {
		name string
		raw  []byte
	}{
		{
			name: "greet",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_greet_01",
			  "kind": "greet",
			  "channel": "builders",
			  "from": "coder.sess-abc",
			  "to": null,
			  "ts": 1775822400,
			  "body": {
			    "peer_card": {
			      "peer_id": "coder.sess-abc",
			      "profiles_supported": ["agh-network/v0"],
			      "capabilities": ["workspace.patch.apply"],
			      "artifacts_supported": ["capability"],
			      "trust_modes_supported": ["unverified"]
			    }
			  }
			}`),
		},
		{
			name: "whois response",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_whois_01",
			  "kind": "whois",
			  "channel": "builders",
			  "from": "reviewer.sess-xyz",
			  "to": "coder.sess-abc",
			  "reply_to": "msg_greet_01",
			  "ts": 1775822400,
			  "body": {
			    "type": "response",
			    "peer_card": {
			      "peer_id": "reviewer.sess-xyz",
			      "profiles_supported": ["agh-network/v0"],
			      "capabilities": ["chat.review"],
			      "artifacts_supported": ["capability"],
			      "trust_modes_supported": ["unverified"]
			    }
			  }
			}`),
		},
		{
			name: "say",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_say_01",
			  "kind": "say",
			  "channel": "builders",
			  "from": "coder.sess-abc",
			  "to": null,
			  "ts": 1775822400,
			  "body": {
			    "text": "I can take the failing migration tests.",
			    "intent": "status_update"
			  }
			}`),
		},
		{
			name: "direct",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_direct_01",
			  "kind": "direct",
			  "channel": "builders",
			  "from": "coder.sess-abc",
			  "to": "reviewer.sess-xyz",
			  "interaction_id": "int_patch_42",
			  "reply_to": "msg_say_01",
			  "trace_id": "trace_ops_patch_42",
			  "causation_id": "msg_say_01",
			  "ts": 1775822400,
			  "expires_at": 1775823000,
			  "body": {
			    "text": "Please inspect auth.go and tell me what is failing.",
			    "intent": "review_request"
			  },
			  "proof": null,
			  "ext": {
			    "agh.workflow": {"ticket": "NET-42"}
			  }
			}`),
		},
		{
			name: "capability",
			raw: mustEnvelopeBytes(t, Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_capability_01",
				Kind:     KindCapability,
				Channel:  "builders",
				From:     "curator.sess-123",
				TS:       1775822400,
				Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review Fix Flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				}),
			}),
		},
		{
			name: "receipt",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_receipt_01",
			  "kind": "receipt",
			  "channel": "builders",
			  "from": "reviewer.sess-xyz",
			  "to": "coder.sess-abc",
			  "interaction_id": "int_patch_42",
			  "reply_to": "msg_direct_01",
			  "ts": 1775822400,
			  "body": {
			    "for_id": "msg_direct_01",
			    "status": "accepted",
			    "detail": "Proceed and report progress with trace messages."
			  }
			}`),
		},
		{
			name: "trace",
			raw: []byte(`{
			  "protocol": "agh-network/v0",
			  "id": "msg_trace_01",
			  "kind": "trace",
			  "channel": "builders",
			  "from": "reviewer.sess-xyz",
			  "to": "coder.sess-abc",
			  "interaction_id": "int_patch_42",
			  "reply_to": "msg_receipt_01",
			  "trace_id": "trace_ops_patch_42",
			  "causation_id": "msg_receipt_01",
			  "ts": 1775822400,
			  "body": {
			    "state": "working",
			    "message": "Started inspecting auth.go"
			  }
			}`),
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			env, err := ParseEnvelope(fixture.raw, opts)
			if err != nil {
				t.Fatalf("ParseEnvelope() error = %v", err)
			}

			firstBody, err := env.DecodeBody()
			if err != nil {
				t.Fatalf("DecodeBody() error = %v", err)
			}

			encoded, err := json.Marshal(env)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			roundTrip, err := ParseEnvelope(encoded, opts)
			if err != nil {
				t.Fatalf("ParseEnvelope(round trip) error = %v", err)
			}

			secondBody, err := roundTrip.DecodeBody()
			if err != nil {
				t.Fatalf("DecodeBody(round trip) error = %v", err)
			}

			if !reflect.DeepEqual(envelopeSnapshot(env), envelopeSnapshot(roundTrip)) {
				t.Fatalf("round-trip envelope mismatch = %#v, want %#v", envelopeSnapshot(roundTrip), envelopeSnapshot(env))
			}
			if !reflect.DeepEqual(firstBody, secondBody) {
				t.Fatalf("round-trip body mismatch = %#v, want %#v", secondBody, firstBody)
			}
		})
	}
}

func mustEnvelopeBytes(t *testing.T, env Envelope) []byte {
	t.Helper()

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal(Envelope) error = %v", err)
	}
	return data
}

func envelopeSnapshot(env Envelope) map[string]any {
	snapshot := map[string]any{
		"protocol": env.Protocol,
		"id":       env.ID,
		"kind":     env.Kind,
		"channel":  env.Channel,
		"from":     env.From,
		"ts":       env.TS,
	}
	if env.To != nil {
		snapshot["to"] = *env.To
	}
	if env.InteractionID != nil {
		snapshot["interaction_id"] = *env.InteractionID
	}
	if env.ReplyTo != nil {
		snapshot["reply_to"] = *env.ReplyTo
	}
	if env.TraceID != nil {
		snapshot["trace_id"] = *env.TraceID
	}
	if env.CausationID != nil {
		snapshot["causation_id"] = *env.CausationID
	}
	if env.ExpiresAt != nil {
		snapshot["expires_at"] = *env.ExpiresAt
	}
	snapshot["proof"] = proofSnapshot(env.Proof)
	snapshot["ext"] = extSnapshot(env.Ext)
	return snapshot
}

func proofSnapshot(proof *Proof) map[string]any {
	if proof == nil || len(*proof) == 0 {
		return map[string]any{}
	}

	snapshot := make(map[string]any, len(*proof))
	for key, value := range *proof {
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			snapshot[key] = string(value)
			continue
		}
		snapshot[key] = decoded
	}
	return snapshot
}
