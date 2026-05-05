package network

import (
	"errors"
	"testing"
	"time"
)

func TestOpenWork(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		env  Envelope
	}{
		{
			name: "direct opener",
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_direct_01",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
			},
		},
		{
			name: "capability opener",
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_capability_01",
				Kind:     KindCapability,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_capability_42"),
				TS:       at.Unix(),
				Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review fix flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				}),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			work, err := OpenWork(withDirectSurface(tc.env), at)
			if err != nil {
				t.Fatalf("OpenWork() error = %v", err)
			}
			if work.State != WorkStateSubmitted {
				t.Fatalf("OpenWork().State = %q, want %q", work.State, WorkStateSubmitted)
			}
			if work.Initiator != tc.env.From || work.Target != *tc.env.To {
				t.Fatalf(
					"OpenWork() participants = (%q,%q), want (%q,%q)",
					work.Initiator,
					work.Target,
					tc.env.From,
					*tc.env.To,
				)
			}
		})
	}
}

func TestApplyWorkEnvelope(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	work := Work{
		ID:        "int_patch_42",
		Ref:       testDirectRef(),
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     WorkStateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	cases := []struct {
		name       string
		current    *Work
		env        Envelope
		wantAction LifecycleAction
		wantState  WorkState
		wantReason *ReasonCode
		wantErr    error
	}{
		{
			name:    "open from nil work",
			current: nil,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_direct_01",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
			},
			wantAction: LifecycleActionOpened,
			wantState:  WorkStateSubmitted,
		},
		{
			name:    "open capability from nil work",
			current: nil,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_capability_01",
				Kind:     KindCapability,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_capability_42"),
				TS:       at.Unix(),
				Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review fix flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				}),
			},
			wantAction: LifecycleActionOpened,
			wantState:  WorkStateSubmitted,
		},
		{
			name:    "trace working advances state",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_trace_01",
				Kind:     KindTrace,
				Channel:  "builders",
				From:     "reviewer.sess-xyz",
				To:       stringPtr("coder.sess-abc"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  WorkStateWorking,
		},
		{
			name: "direct resumes work from needs_input",
			current: &Work{
				ID:        "int_patch_42",
				Ref:       testDirectRef(),
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     WorkStateNeedsInput,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_direct_02",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"text": "here is the missing detail"}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  WorkStateWorking,
		},
		{
			name:    "direct without target is rejected",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_direct_missing_to",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"text": "missing target"}),
			},
			wantErr: ErrMissingField,
		},
		{
			name:    "capability outside participant pair is rejected",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_capability_bad_target",
				Kind:     KindCapability,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("outsider.sess-123"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
					ID:               "review-fix",
					Summary:          "Review fix flow",
					Outcome:          "A reusable review fix workflow.",
					Version:          "1.0.0",
					ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
					Requirements:     []string{"workspace-write"},
				}),
			},
			wantErr: ErrWorkActorNotAllowed,
		},
		{
			name:    "receipt rejected fails work",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_receipt_01",
				Kind:     KindReceipt,
				Channel:  "builders",
				From:     "reviewer.sess-xyz",
				To:       stringPtr("coder.sess-abc"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body: mustRawJSON(t, map[string]any{
					"for_id":      "msg_direct_01",
					"status":      "rejected",
					"reason_code": "busy",
				}),
			},
			wantAction: LifecycleActionAdvanced,
			wantState:  WorkStateFailed,
		},
		{
			name: "post terminal trace is ignored",
			current: &Work{
				ID:        "int_patch_42",
				Ref:       testDirectRef(),
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     WorkStateCompleted,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_trace_02",
				Kind:     KindTrace,
				Channel:  "builders",
				From:     "reviewer.sess-xyz",
				To:       stringPtr("coder.sess-abc"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantAction: LifecycleActionIgnored,
			wantState:  WorkStateCompleted,
		},
		{
			name: "post terminal direct is rejected",
			current: &Work{
				ID:        "int_patch_42",
				Ref:       testDirectRef(),
				Initiator: "coder.sess-abc",
				Target:    "reviewer.sess-xyz",
				State:     WorkStateCompleted,
				CreatedAt: at,
				UpdatedAt: at,
			},
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_direct_03",
				Kind:     KindSay,
				Channel:  "builders",
				From:     "coder.sess-abc",
				To:       stringPtr("reviewer.sess-xyz"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"text": "try again"}),
			},
			wantAction: LifecycleActionRejectWork,
			wantState:  WorkStateCompleted,
			wantReason: reasonCodePtr(ReasonCodeWorkClosed),
		},
		{
			name:    "third party actor rejected",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_trace_bad",
				Kind:     KindTrace,
				Channel:  "builders",
				From:     "intruder.sess-123",
				To:       stringPtr("coder.sess-abc"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"state": "working"}),
			},
			wantErr: ErrWorkActorNotAllowed,
		},
		{
			name:    "invalid submitted regression rejected",
			current: &work,
			env: Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_trace_bad_state",
				Kind:     KindTrace,
				Channel:  "builders",
				From:     "reviewer.sess-xyz",
				To:       stringPtr("coder.sess-abc"),
				WorkID:   stringPtr("int_patch_42"),
				TS:       at.Unix(),
				Body:     mustRawJSON(t, map[string]any{"state": "submitted"}),
			},
			wantErr: ErrInvalidStateTransition,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ApplyWorkEnvelope(tc.current, withDirectSurface(tc.env), at.Add(time.Second))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("ApplyWorkEnvelope() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ApplyWorkEnvelope() error = %v", err)
			}
			if got.Action != tc.wantAction {
				t.Fatalf("ApplyWorkEnvelope().Action = %q, want %q", got.Action, tc.wantAction)
			}
			if got.Work.State != tc.wantState {
				t.Fatalf("ApplyWorkEnvelope().State = %q, want %q", got.Work.State, tc.wantState)
			}
			if tc.wantReason != nil {
				if got.ReasonCode == nil || *got.ReasonCode != *tc.wantReason {
					t.Fatalf("ApplyWorkEnvelope().ReasonCode = %v, want %v", got.ReasonCode, tc.wantReason)
				}
			}
		})
	}
}

func TestCancellationRaceHonorsFirstTerminalMessage(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	current := &Work{
		ID:        "int_patch_42",
		Ref:       testDirectRef(),
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     WorkStateSubmitted,
		CreatedAt: at,
		UpdatedAt: at,
	}

	receiptCanceled := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_receipt_cancel",
		Kind:     KindReceipt,
		Channel:  "builders",
		From:     "coder.sess-abc",
		To:       stringPtr("reviewer.sess-xyz"),
		WorkID:   stringPtr("int_patch_42"),
		TS:       at.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "canceled",
		}),
	}

	first, err := ApplyWorkEnvelope(current, withDirectSurface(receiptCanceled), at.Add(time.Second))
	if err != nil {
		t.Fatalf("ApplyWorkEnvelope(first) error = %v", err)
	}
	if first.Work.State != WorkStateCanceled {
		t.Fatalf("first state = %q, want %q", first.Work.State, WorkStateCanceled)
	}

	traceCanceled := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_trace_cancel",
		Kind:     KindTrace,
		Channel:  "builders",
		From:     "reviewer.sess-xyz",
		To:       stringPtr("coder.sess-abc"),
		WorkID:   stringPtr("int_patch_42"),
		TS:       at.Unix(),
		Body:     mustRawJSON(t, map[string]any{"state": "canceled"}),
	}

	second, err := ApplyWorkEnvelope(&first.Work, withDirectSurface(traceCanceled), at.Add(2*time.Second))
	if err != nil {
		t.Fatalf("ApplyWorkEnvelope(second) error = %v", err)
	}
	if second.Action != LifecycleActionIgnored {
		t.Fatalf("second action = %q, want %q", second.Action, LifecycleActionIgnored)
	}
	if second.Work.State != WorkStateCanceled {
		t.Fatalf("second state = %q, want %q", second.Work.State, WorkStateCanceled)
	}
}

func reasonCodePtr(code ReasonCode) *ReasonCode {
	return &code
}
