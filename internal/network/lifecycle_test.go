package network

import (
	"errors"
	"testing"
	"time"
)

func TestOpenInteraction(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		env  Envelope
	}{
		{
			name: "direct opener",
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_direct_01",
				Kind:          KindDirect,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
			},
		},
		{
			name: "recipe opener",
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_recipe_01",
				Kind:          KindRecipe,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_recipe_42"),
				TS:            at.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id":    "review-fix",
						"version":      "1.0.0",
						"content_type": "text/markdown",
						"digest":       "sha256:abc123",
						"inline":       "# Review fix flow",
					},
				}),
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			interaction, err := OpenInteraction(tc.env, at)
			if err != nil {
				t.Fatalf("OpenInteraction() error = %v", err)
			}
			if interaction.State != StateSubmitted {
				t.Fatalf("OpenInteraction().State = %q, want %q", interaction.State, StateSubmitted)
			}
			if interaction.Initiator != tc.env.From || interaction.Target != *tc.env.To {
				t.Fatalf("OpenInteraction() participants = (%q,%q), want (%q,%q)", interaction.Initiator, interaction.Target, tc.env.From, *tc.env.To)
			}
		})
	}
}

func TestApplyInteractionEnvelope(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	interaction := Interaction{
		ID:        "int_patch_42",
		Space:     "builders",
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     StateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	cases := []struct {
		name       string
		current    *Interaction
		env        Envelope
		wantAction LifecycleAction
		wantState  InteractionState
		wantReason *ReasonCode
		wantErr    error
	}{
		{
			name:    "open from nil interaction",
			current: nil,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_direct_01",
				Kind:          KindDirect,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
			},
			wantAction: LifecycleActionOpened,
			wantState:  StateSubmitted,
		},
		{
			name:    "open recipe from nil interaction",
			current: nil,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_recipe_01",
				Kind:          KindRecipe,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_recipe_42"),
				TS:            at.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id":    "review-fix",
						"version":      "1.0.0",
						"content_type": "text/markdown",
						"digest":       "sha256:abc123",
						"inline":       "# Review fix flow",
					},
				}),
			},
			wantAction: LifecycleActionOpened,
			wantState:  StateSubmitted,
		},
		{
			name:    "trace working advances state",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_trace_01",
				Kind:          KindTrace,
				Space:         "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  StateWorking,
		},
		{
			name: "direct resumes work from needs_input",
			current: &Interaction{
				ID:        "int_patch_42",
				Space:     "builders",
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     StateNeedsInput,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_direct_02",
				Kind:          KindDirect,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"text": "here is the missing detail"}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  StateWorking,
		},
		{
			name:    "direct without target is rejected",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_direct_missing_to",
				Kind:          KindDirect,
				Space:         "builders",
				From:          "coder.sess-abc",
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"text": "missing target"}),
			},
			wantErr: ErrMissingField,
		},
		{
			name:    "recipe outside participant pair is rejected",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_recipe_bad_target",
				Kind:          KindRecipe,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("outsider.sess-123"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"recipe": map[string]any{
						"recipe_id":    "review-fix",
						"version":      "1.0.0",
						"content_type": "text/markdown",
						"digest":       "sha256:abc123",
						"inline":       "# Review fix flow",
					},
				}),
			},
			wantErr: ErrInteractionActorNotAllowed,
		},
		{
			name:    "receipt rejected fails interaction",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_receipt_01",
				Kind:          KindReceipt,
				Space:         "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"for_id":      "msg_direct_01",
					"status":      "rejected",
					"reason_code": "busy",
				}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  StateFailed,
		},
		{
			name: "post terminal trace is ignored",
			current: &Interaction{
				ID:        "int_patch_42",
				Space:     "builders",
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     StateCompleted,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_trace_02",
				Kind:          KindTrace,
				Space:         "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantAction: LifecycleActionIgnored,
			wantState:  StateCompleted,
		},
		{
			name: "post terminal direct is rejected",
			current: &Interaction{
				ID:        "int_patch_42",
				Space:     "builders",
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     StateCompleted,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_direct_03",
				Kind:          KindDirect,
				Space:         "builders",
				From:          "coder.sess-abc",
				To:            stringPtr("reviewer.sess-xyz"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"text": "try again"}),
			},
			wantAction: LifecycleActionRejectDirect,
			wantState:  StateCompleted,
			wantReason: reasonCodePtr(ReasonCodeInteractionClosed),
		},
		{
			name:    "third party actor rejected",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_trace_bad",
				Kind:          KindTrace,
				Space:         "builders",
				From:          "intruder.sess-123",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantErr: ErrInteractionActorNotAllowed,
		},
		{
			name:    "invalid submitted regression rejected",
			current: &interaction,
			env: Envelope{
				Protocol:      ProtocolV0,
				ID:            "msg_trace_bad_state",
				Kind:          KindTrace,
				Space:         "builders",
				From:          "reviewer.sess-xyz",
				To:            stringPtr("coder.sess-abc"),
				InteractionID: stringPtr("int_patch_42"),
				TS:            at.Unix(),
				Body:          mustRawJSON(t, map[string]any{"state": "submitted"}),
			},
			wantErr: ErrInvalidStateTransition,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ApplyInteractionEnvelope(tc.current, tc.env, at.Add(time.Second))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("ApplyInteractionEnvelope() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ApplyInteractionEnvelope() error = %v", err)
			}
			if got.Action != tc.wantAction {
				t.Fatalf("ApplyInteractionEnvelope().Action = %q, want %q", got.Action, tc.wantAction)
			}
			if got.Interaction.State != tc.wantState {
				t.Fatalf("ApplyInteractionEnvelope().State = %q, want %q", got.Interaction.State, tc.wantState)
			}
			if tc.wantReason != nil {
				if got.ReasonCode == nil || *got.ReasonCode != *tc.wantReason {
					t.Fatalf("ApplyInteractionEnvelope().ReasonCode = %v, want %v", got.ReasonCode, tc.wantReason)
				}
			}
		})
	}
}

func TestCancellationRaceHonorsFirstTerminalMessage(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	current := &Interaction{
		ID:        "int_patch_42",
		Space:     "builders",
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     StateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	receiptCanceled := Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_receipt_cancel",
		Kind:          KindReceipt,
		Space:         "builders",
		From:          "coder.sess-abc",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            at.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "canceled",
		}),
	}

	first, err := ApplyInteractionEnvelope(current, receiptCanceled, at.Add(time.Second))
	if err != nil {
		t.Fatalf("ApplyInteractionEnvelope(first) error = %v", err)
	}
	if first.Interaction.State != StateCanceled {
		t.Fatalf("first state = %q, want %q", first.Interaction.State, StateCanceled)
	}

	traceCanceled := Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_trace_cancel",
		Kind:          KindTrace,
		Space:         "builders",
		From:          "reviewer.sess-xyz",
		To:            stringPtr("coder.sess-abc"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            at.Unix(),
		Body:          mustRawJSON(t, map[string]any{"state": "canceled"}),
	}

	second, err := ApplyInteractionEnvelope(&first.Interaction, traceCanceled, at.Add(2*time.Second))
	if err != nil {
		t.Fatalf("ApplyInteractionEnvelope(second) error = %v", err)
	}
	if second.Action != LifecycleActionIgnored {
		t.Fatalf("second action = %q, want %q", second.Action, LifecycleActionIgnored)
	}
	if second.Interaction.State != StateCanceled {
		t.Fatalf("second state = %q, want %q", second.Interaction.State, StateCanceled)
	}
}

func reasonCodePtr(code ReasonCode) *ReasonCode {
	return &code
}
