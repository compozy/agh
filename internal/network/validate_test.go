package network

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
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
			name: "greet",
			envelope: Envelope{
				Protocol: " agh-network/v0 ",
				ID:       " msg_greet_01 ",
				Kind:     " greet ",
				Space:    " builders ",
				From:     " coder.sess-abc ",
				To:       nil,
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"peer_card": map[string]any{
						"peer_id":               "coder.sess-abc",
						"profiles_supported":    []string{"agh-network/v0"},
						"capabilities":          []string{"workspace.patch.apply"},
						"artifacts_supported":   []string{"recipe"},
						"trust_modes_supported": []string{"unverified"},
					},
					"summary": "hello",
				}),
			},
			wantKind: KindGreet,
			wantType: reflect.TypeOf(GreetBody{}),
		},
		{
			name: "whois response",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_whois_01",
				Kind:     KindWhois,
				Space:    "builders",
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
						"artifacts_supported":   []string{"recipe"},
						"trust_modes_supported": []string{"unverified"},
					},
				}),
			},
			wantKind: KindWhois,
			wantType: reflect.TypeOf(WhoisBody{}),
		},
		{
			name: "say",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_say_01",
				Kind:     KindSay,
				Space:    "builders",
				From:     "coder.sess-abc",
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"text":   "working through it",
					"intent": "status_update",
				}),
			},
			wantKind: KindSay,
			wantType: reflect.TypeOf(SayBody{}),
		},
		{
			name: "direct",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_direct_01",
				Kind:          KindDirect,
				Space:         "builders",
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
			wantType: reflect.TypeOf(DirectBody{}),
		},
		{
			name: "recipe",
			envelope: Envelope{
				Protocol: "agh-network/v0",
				ID:       "msg_recipe_01",
				Kind:     KindRecipe,
				Space:    "builders",
				From:     "coder.sess-abc",
				TS:       now.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id":    "review-fix",
						"version":      "1.0.0",
						"title":        "Review fix flow",
						"content_type": "text/markdown",
						"digest":       "sha256:abc123",
						"inline":       "# Review fix flow",
					},
				}),
			},
			wantKind: KindRecipe,
			wantType: reflect.TypeOf(RecipeBody{}),
		},
		{
			name: "receipt",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_receipt_01",
				Kind:          KindReceipt,
				Space:         "builders",
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
			wantType: reflect.TypeOf(ReceiptBody{}),
		},
		{
			name: "trace",
			envelope: Envelope{
				Protocol:      "agh-network/v0",
				ID:            "msg_trace_01",
				Kind:          KindTrace,
				Space:         "builders",
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
			wantType: reflect.TypeOf(TraceBody{}),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			normalized, err := NormalizeEnvelope(tc.envelope, opts)
			if err != nil {
				t.Fatalf("NormalizeEnvelope() error = %v", err)
			}

			if normalized.Kind != tc.wantKind {
				t.Fatalf("NormalizeEnvelope().Kind = %q, want %q", normalized.Kind, tc.wantKind)
			}
			if normalized.Space != "builders" {
				t.Fatalf("NormalizeEnvelope().Space = %q, want builders", normalized.Space)
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
		Space:         "builders",
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
			name: "invalid space",
			mutate: func(env Envelope) Envelope {
				env.Space = "bad.space"
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "space",
		},
		{
			name: "invalid from",
			mutate: func(env Envelope) Envelope {
				env.From = "BadPeer"
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "peer_id",
		},
		{
			name: "invalid to",
			mutate: func(env Envelope) Envelope {
				env.To = stringPtr("missing space")
				return env
			},
			wantErr:   ErrInvalidField,
			wantMatch: "to",
		},
		{
			name: "missing interaction id",
			mutate: func(env Envelope) Envelope {
				env.InteractionID = nil
				return env
			},
			wantErr:   ErrMissingField,
			wantMatch: "interaction_id",
		},
		{
			name: "whois response missing reply_to",
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
						"artifacts_supported":   []string{"recipe"},
						"trust_modes_supported": []string{"unverified"},
					},
				})
				return env
			},
			wantErr:   ErrMissingField,
			wantMatch: "reply_to",
		},
		{
			name: "expired message",
			mutate: func(env Envelope) Envelope {
				env.ExpiresAt = int64Ptr(now.Add(-time.Second).Unix())
				return env
			},
			wantErr:   ErrExpired,
			wantMatch: "expires_at",
		},
		{
			name: "replay too old",
			mutate: func(env Envelope) Envelope {
				env.ExpiresAt = nil
				env.TS = now.Add(-10 * time.Minute).Unix()
				return env
			},
			wantErr:   ErrReplayTooOld,
			wantMatch: "max_replay_age",
		},
		{
			name: "accepted receipt with reason code",
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
			name: "recipe missing source",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindRecipe
				env.InteractionID = nil
				env.To = nil
				env.Body = mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id":    "review-fix",
						"version":      "1.0.0",
						"content_type": "text/markdown",
						"digest":       "sha256:abc123",
					},
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: "uri or inline",
		},
		{
			name: "recipe missing nested recipe object",
			mutate: func(env Envelope) Envelope {
				env.Kind = KindRecipe
				env.InteractionID = nil
				env.To = nil
				env.Body = mustRawJSON(t, map[string]any{
					"recipe_id":    "review-fix",
					"version":      "1.0.0",
					"content_type": "text/markdown",
					"digest":       "sha256:abc123",
					"inline":       "# Review fix flow",
				})
				return env
			},
			wantErr:   ErrInvalidBody,
			wantMatch: `{"recipe":{...}}`,
		},
	}

	for _, tc := range cases {
		tc := tc
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

func TestRouteTokenKnownVectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		peerID string
		want   string
	}{
		{peerID: "reviewer.sess-xyz", want: "790dd5515558f7784877abcbca51c5ba"},
		{peerID: "coder.sess-abc", want: "07f9c1120ea61cb8f1a14ebec70c8912"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.peerID, func(t *testing.T) {
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

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	raw := []byte(`{
	  "protocol": "agh-network/v0",
	  "id": "msg_direct_ext_01",
	  "kind": "direct",
	  "space": "builders",
	  "from": "coder.sess-abc",
	  "to": "reviewer.sess-xyz",
	  "interaction_id": "int_patch_42",
	  "ts": 1775822400,
	  "body": {"text": "review this"},
	  "proof": {"profile": "agh-network.trust.ed25519-jcs/v1"},
	  "ext": {
	    "unknown.vendor": {"nested": [1, true, "x"]},
	    "agh.workflow": {"ticket": "NET-42"},
	    "agh.handoff": {"turn": 3}
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
}

func mustRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v", value, err)
	}
	return json.RawMessage(data)
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
