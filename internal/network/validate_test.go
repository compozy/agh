package network

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestNormalizeEnvelopeValidKinds(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	opts := ValidateOptions{Now: now, MaxReplayAge: DefaultMaxReplayAge}

	target := "reviewer.sess-xyz"
	interactionID := "int_patch_42"
	replyTo := "msg_direct_00"
	traceID := "trace_patch_42"

	cases := []struct {
		name     string
		envelope Envelope
		wantKind Kind
		wantType reflect.Type
	}{
		{
			name: "Should normalize greet envelopes",
			envelope: Envelope{
				Protocol: " agh-network/v0 ",
				ID:       " msg_greet_01 ",
				Kind:     " greet ",
				Channel:  " builders ",
				From:     " coder.sess-abc ",
				To:       nil,
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"peer_card": map[string]any{
						"peer_id":               "coder.sess-abc",
						"profiles_supported":    []string{"agh-network/v0"},
						"capabilities":          []string{"workspace.patch.apply"},
						"artifacts_supported":   []string{"capability"},
						"trust_modes_supported": []string{"unverified"},
					},
					"summary": "hello",
				}),
			},
			wantKind: KindGreet,
			wantType: reflect.TypeFor[GreetBody](),
		},
		{
			name: "Should normalize whois response envelopes",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_whois_01",
				Kind:     KindWhois,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       &target,
				ReplyTo:  &replyTo,
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"type": "response",
					"peer_card": map[string]any{
						"peer_id":               "coder.sess-abc",
						"profiles_supported":    []string{"agh-network/v0"},
						"capabilities":          []string{"chat.translate"},
						"artifacts_supported":   []string{"capability"},
						"trust_modes_supported": []string{"unverified"},
					},
				}),
			},
			wantKind: KindWhois,
			wantType: reflect.TypeFor[WhoisBody](),
		},
		{
			name: "Should normalize say envelopes",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_say_01",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"text":   "working through it",
					"intent": "status_update",
				}),
			},
			wantKind: KindSay,
			wantType: reflect.TypeFor[SayBody](),
		},
		{
			name: "Should normalize direct envelopes",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_direct_01",
				Kind:          KindDirect,
				Channel:       "builders",
				From:          "coder.sess-abc",
				To:            &target,
				InteractionID: &interactionID,
				TraceID:       &traceID,
				TS:            now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"text":   "please review auth.go",
					"intent": "review_request",
				}),
			},
			wantKind: KindDirect,
			wantType: reflect.TypeFor[DirectBody](),
		},
		{
			name: "Should normalize capability envelopes",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_capability_01",
				Kind:     KindCapability,
				Channel:  "builders",
				From:     "coder.sess-abc",
				TS:       now.Unix(),
				Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review fix flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				}),
			},
			wantKind: KindCapability,
			wantType: reflect.TypeFor[CapabilityBody](),
		},
		{
			name: "Should normalize receipt envelopes",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_receipt_01",
				Kind:          KindReceipt,
				Channel:       "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr(target),
				InteractionID: stringPtr(interactionID),
				TS:            now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"for_id": "msg_direct_01",
					"status": "accepted",
					"detail": "Proceed.",
				}),
			},
			wantKind: KindReceipt,
			wantType: reflect.TypeFor[ReceiptBody](),
		},
		{
			name: "Should normalize trace envelopes",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_trace_01",
				Kind:          KindTrace,
				Channel:       "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr(interactionID),
				TS:            now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"state":   "working",
					"message": "started",
				}),
			},
			wantKind: KindTrace,
			wantType: reflect.TypeFor[TraceBody](),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			normalized, err := NormalizeEnvelope(tc.envelope, opts)
			if err != nil {
				t.Fatalf("NormalizeEnvelope() error = %v", err)
			}

			if normalized.Kind != tc.wantKind {
				t.Fatalf("NormalizeEnvelope().Kind = %q, want %q", normalized.Kind, tc.wantKind)
			}
			if normalized.Channel != "builders" {
				t.Fatalf("NormalizeEnvelope().Channel = %q, want builders", normalized.Channel)
			}
			if normalized.From == "" || strings.Contains(normalized.From, " ") {
				t.Fatalf("NormalizeEnvelope().From = %q, want trimmed peer id", normalized.From)
			}

			body, err := normalized.DecodeBody()
			if err != nil {
				t.Fatalf("DecodeBody() error = %v", err)
			}
			if got := reflect.TypeOf(body); got != tc.wantType {
				t.Fatalf("DecodeBody() type = %v, want %v", got, tc.wantType)
			}
		})
	}
}

func TestParseEnvelopeRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	opts := ValidateOptions{Now: now, MaxReplayAge: DefaultMaxReplayAge}

	base := Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_direct_01",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-abc",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"text": "please review auth.go",
		}),
	}

	cases := []struct {
		name      string
		mutate    func(Envelope) Envelope
		wantErr   error
		wantMatch string
	}{
		{
			name: "Should reject legacy recipe kinds",
			mutate: func(env Envelope) Envelope {
				env.Kind = Kind("recipe")
				env.To = nil
				env.InteractionID = nil
				env.Body = mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id": "review-fix",
					},
				})
				return env
			},
			wantErr:   ErrInvalidKind,
			wantMatch: `kind="recipe"`,
		},
		{
			name: "Should reject invalid channels",
			mutate: func(env Envelope) Envelope {
				env.Channel = "bad.channel"
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "channel",
		},
		{
			name: "Should reject invalid from peer IDs",
			mutate: func(env Envelope) Envelope {
				env.From = "BadPeer"
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "peer_id",
		},
		{
			name: "Should reject invalid destination peer IDs",
			mutate: func(env Envelope) Envelope {
				env.To = stringPtr("missing channel")
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "to",
		},
		{
			name: "Should reject missing interaction IDs",
			mutate: func(env Envelope) Envelope {
				env.InteractionID = nil
				return env
			},
			wantErr:   ErrMissingField,
			wantMatch: "interaction_id",
		},
		{
			name: "Should reject whois responses without reply_to",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindWhois
				env.To = nil
				env.InteractionID = nil
				env.Body = mustRawJSON(t, map[string]any{
					"type": "response",
					"peer_card": map[string]any{
						"peer_id":               "coder.sess-abc",
						"profiles_supported":    []string{"agh-network/v0"},
						"capabilities":          []string{"chat.translate"},
						"artifacts_supported":   []string{"capability"},
						"trust_modes_supported": []string{"unverified"},
					},
				})
				return env
			},
			wantErr:   ErrMissingField,
			wantMatch: "reply_to",
		},
		{
			name: "Should reject expired messages",
			mutate: func(env Envelope) Envelope {
				env.ExpiresAt = int64Ptr(now.Add(-time.Second).Unix())
				return env
			},
			wantErr:   ErrExpired,
			wantMatch: "expires_at",
		},
		{
			name: "Should reject replay timestamps that are too old",
			mutate: func(env Envelope) Envelope {
				env.ExpiresAt = nil
				env.TS = now.Add(-10 * time.Minute).Unix()
				return env
			},
			wantErr:   ErrReplayTooOld,
			wantMatch: "max_replay_age",
		},
		{
			name: "Should reject future timestamps outside the replay window",
			mutate: func(env Envelope) Envelope {
				env.ExpiresAt = nil
				env.TS = now.Add(10 * time.Minute).Unix()
				return env
			},
			wantErr:   ErrReplayTooOld,
			wantMatch: "max_replay_age",
		},
		{
			name: "Should reject greet task write capabilities without proof",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindGreet
				env.To = nil
				env.InteractionID = nil
				env.Body = mustRawJSON(t, map[string]any{
					"peer_card": map[string]any{
						"peer_id":               "coder.sess-abc",
						"profiles_supported":    []string{"agh-network/v0"},
						"capabilities":          []string{networkTaskWriteCapability},
						"artifacts_supported":   []string{"capability"},
						"trust_modes_supported": []string{"unverified"},
					},
				})
				return env
			},
			wantErr:   ErrVerificationFailed,
			wantMatch: "requires proof",
		},
		{
			name: "Should reject raw secrets in the body payload",
			mutate: func(env Envelope) Envelope {
				env.Body = mustRawJSON(t, map[string]any{
					"text":         "please review auth.go",
					"access_token": "provider-token",
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "raw secret material",
		},
		{
			name: "Should reject raw secrets in body keys",
			mutate: func(env Envelope) Envelope {
				env.Body = mustRawJSON(t, map[string]any{
					"agh_claim_secret-token": "",
					"text":                   "please review auth.go",
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "raw secret material",
		},
		{
			name: "Should reject raw secrets in extensions",
			mutate: func(env Envelope) Envelope {
				env.Ext = ExtensionMap{
					"agh.handoff": mustRawJSON(t, map[string]any{
						"note": "Bearer provider-token",
					}),
				}
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "raw secret material",
		},
		{
			name: "Should reject raw secrets in proofs",
			mutate: func(env Envelope) Envelope {
				proof := Proof{
					"agh.proof": mustRawJSON(t, map[string]any{
						"access_token": "provider-token",
					}),
				}
				env.Proof = &proof
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "network proof",
		},
		{
			name: "Should reject accepted receipts with reason codes",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindReceipt
				env.From = "reviewer.sess-xyz"
				env.Body = mustRawJSON(t, map[string]any{
					"for_id":      "msg_direct_01",
					"status":      "accepted",
					"reason_code": "busy",
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "accepted receipt",
		},
		{
			name: "Should reject capability digest mismatches",
			mutate: func(env Envelope) Envelope {
				capability := canonicalCapabilityPayload(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review fix flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				})
				capability.Digest = "sha256:not-the-canonical-digest"

				env.Kind = KindCapability
				env.InteractionID = nil
				env.To = nil
				env.Body = mustRawJSON(t, CapabilityBody{Capability: capability})
				return env
			},
			wantErr:   ErrVerificationFailed,
			wantMatch: "canonical digest",
		},
		{
			name: "Should reject capability bodies without nested capability objects",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindCapability
				env.InteractionID = nil
				env.To = nil
				env.Body = mustRawJSON(t, map[string]any{
					"id":      "review-fix",
					"summary": "Review fix flow",
					"outcome": "A reusable review fix workflow.",
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: `{"capability":{...}}`,
		},
		{
			name: "Should reject capabilities without outcomes",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindCapability
				env.InteractionID = nil
				env.To = nil
				env.Body = mustRawJSON(t, map[string]any{
					"capability": map[string]any{
						"id":      "review-fix",
						"summary": "Review fix flow",
						"digest":  "sha256:missing-outcome",
					},
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "capability.outcome is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NormalizeEnvelope(tc.mutate(base), opts)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NormalizeEnvelope() error = %v, want %v", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantMatch) {
				t.Fatalf("NormalizeEnvelope() error = %v, want substring %q", err, tc.wantMatch)
			}
		})
	}
}

func TestNormalizeEnvelopeAllowsWhitespaceOnlyStrings(t *testing.T) {
	t.Run("Should allow whitespace-only optional fields", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
		opts := ValidateOptions{Now: now, MaxReplayAge: DefaultMaxReplayAge}

		envelope := Envelope{
			Protocol: "agh-network/v0",
			ID:       "msg_say_whitespace_01",
			Kind:     KindSay,
			Channel:  "builders",
			From:     "coder.sess-abc",
			TS:       now.Unix(),
			Body: mustRawJSON(t, map[string]any{
				"text":    "progress update",
				"summary": "   ",
			}),
		}

		if _, err := NormalizeEnvelope(envelope, opts); err != nil {
			t.Fatalf("NormalizeEnvelope(whitespace-only optional field) error = %v", err)
		}
	})
}

func TestRouteTokenKnownVectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		peerID string
		want   string
	}{
		{
			name:   "Should return the known route token for reviewer.sess-xyz",
			peerID: "reviewer.sess-xyz",
			want:   "790dd5515558f7784877abcbca51c5ba",
		},
		{
			name:   "Should return the known route token for coder.sess-abc",
			peerID: "coder.sess-abc",
			want:   "07f9c1120ea61cb8f1a14ebec70c8912",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := RouteToken(tc.peerID)
			if err != nil {
				t.Fatalf("RouteToken() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("RouteToken() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtRoundTripPreservesOpaqueKeys(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve opaque ext keys on round trip", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
		raw := []byte(`{
	  "protocol": "agh-network/v0",
	  "id": "msg_direct_ext_01",
	  "kind": "direct",
	  "channel": "builders",
	  "from": "coder.sess-abc",
	  "to": "reviewer.sess-xyz",
	  "interaction_id": "int_patch_42",
	  "ts": 1775822400,
	  "body": {"text": "review this"},
	  "proof": {"profile": "agh-network.trust.ed25519-jcs/v1"},
	  "ext": {
	    "unknown.vendor": {"nested": [1, true, "x"]},
	    "agh.workflow": {"ticket": "NET-42"},
	    "agh.handoff": {"turn": 3},
	    "agh.capability_catalog": {
	      "capabilities": [
	        {"id": "review-pr", "summary": "Review pull requests", "outcome": "Actionable review findings"}
	      ]
	    }
	  }
	}`)

		env, err := ParseEnvelope(raw, ValidateOptions{Now: now})
		if err != nil {
			t.Fatalf("ParseEnvelope() error = %v", err)
		}
		if env.Proof == nil {
			t.Fatalf("ParseEnvelope().Proof = nil, want preserved proof payload")
		}

		encoded, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		roundTripped, err := ParseEnvelope(encoded, ValidateOptions{Now: now})
		if err != nil {
			t.Fatalf("ParseEnvelope(round trip) error = %v", err)
		}

		if !reflect.DeepEqual(extSnapshot(env.Ext), extSnapshot(roundTripped.Ext)) {
			t.Fatalf("Ext round-trip mismatch = %#v, want %#v", extSnapshot(roundTripped.Ext), extSnapshot(env.Ext))
		}
	})
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return json.RawMessage(data)
}

func canonicalCapabilityPayload(t *testing.T, capability CapabilityEnvelopePayload) CapabilityEnvelopePayload {
	t.Helper()

	if strings.TrimSpace(capability.Digest) != "" {
		capability.Digest = strings.TrimSpace(capability.Digest)
		return capability
	}

	digest, err := aghconfig.CanonicalCapabilityDigest(aghconfig.CapabilityDef{
		ID:                capability.ID,
		Summary:           capability.Summary,
		Outcome:           capability.Outcome,
		Version:           capability.Version,
		ContextNeeded:     append([]string(nil), capability.ContextNeeded...),
		ArtifactsExpected: append([]string(nil), capability.ArtifactsExpected...),
		ExecutionOutline:  append([]string(nil), capability.ExecutionOutline...),
		Constraints:       append([]string(nil), capability.Constraints...),
		Examples:          append([]string(nil), capability.Examples...),
		Requirements:      append([]string(nil), capability.Requirements...),
	})
	if err != nil {
		t.Fatalf("CanonicalCapabilityDigest() error = %v", err)
	}

	capability.Digest = digest
	return capability
}

func mustCapabilityBodyJSON(t *testing.T, capability CapabilityEnvelopePayload) json.RawMessage {
	t.Helper()

	return mustRawJSON(t, CapabilityBody{Capability: canonicalCapabilityPayload(t, capability)})
}

func stringPtr(value string) *string {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func extSnapshot(ext ExtensionMap) map[string]any {
	if len(ext) == 0 {
		return map[string]any{}
	}

	snapshot := make(map[string]any, len(ext))
	for key, value := range ext {
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			snapshot[key] = string(value)
			continue
		}
		snapshot[key] = decoded
	}
	return snapshot
}
