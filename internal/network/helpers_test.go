package network

import (
	"errors"
	"testing"
	"time"
)

func TestEnumValidationAndBodyKindHelpers(t *testing.T) {
	t.Parallel()

	validKinds := []Kind{KindGreet, KindWhois, KindSay, KindCapability, KindReceipt, KindTrace}
	for _, kind := range validKinds {
		t.Run("ShouldValidateKnownKind"+string(kind), func(t *testing.T) {
			t.Parallel()

			if err := kind.Validate(); err != nil {
				t.Fatalf("Kind(%q).Validate() error = %v", kind, err)
			}
		})
	}
	if err := Kind("invalid").Validate(); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("Kind(invalid).Validate() error = %v, want ErrInvalidKind", err)
	}

	validReceiptStatuses := []ReceiptStatus{
		ReceiptStatusAccepted,
		ReceiptStatusRejected,
		ReceiptStatusDuplicate,
		ReceiptStatusExpired,
		ReceiptStatusUnsupported,
		ReceiptStatusCanceled,
	}
	for _, status := range validReceiptStatuses {
		t.Run("ShouldValidateKnownReceiptStatus"+string(status), func(t *testing.T) {
			t.Parallel()

			if err := status.Validate(); err != nil {
				t.Fatalf("ReceiptStatus(%q).Validate() error = %v", status, err)
			}
		})
	}
	if err := ReceiptStatus("unknown").Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("ReceiptStatus(unknown).Validate() error = %v, want ErrInvalidField", err)
	}

	if err := WhoisTypeRequest.Validate(); err != nil {
		t.Fatalf("WhoisTypeRequest.Validate() error = %v", err)
	}
	if err := WhoisType("other").Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("WhoisType(other).Validate() error = %v, want ErrInvalidField", err)
	}

	if err := ReasonCodeBusy.Validate(); err != nil {
		t.Fatalf("ReasonCodeBusy.Validate() error = %v", err)
	}
	if err := ReasonCode("mystery").Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("ReasonCode(mystery).Validate() error = %v, want ErrInvalidField", err)
	}

	if err := WorkStateWorking.Validate(); err != nil {
		t.Fatalf("WorkStateWorking.Validate() error = %v", err)
	}
	if err := WorkState("drifting").Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("WorkState(drifting).Validate() error = %v, want ErrInvalidField", err)
	}

	if got := (GreetBody{}).Kind(); got != KindGreet {
		t.Fatalf("GreetBody.Kind() = %q, want %q", got, KindGreet)
	}
	if got := (WhoisBody{}).Kind(); got != KindWhois {
		t.Fatalf("WhoisBody.Kind() = %q, want %q", got, KindWhois)
	}
	if got := (SayBody{}).Kind(); got != KindSay {
		t.Fatalf("SayBody.Kind() = %q, want %q", got, KindSay)
	}
	if got := (CapabilityBody{}).Kind(); got != KindCapability {
		t.Fatalf("CapabilityBody.Kind() = %q, want %q", got, KindCapability)
	}
	if got := (ReceiptBody{}).Kind(); got != KindReceipt {
		t.Fatalf("ReceiptBody.Kind() = %q, want %q", got, KindReceipt)
	}
	if got := (TraceBody{}).Kind(); got != KindTrace {
		t.Fatalf("TraceBody.Kind() = %q, want %q", got, KindTrace)
	}

	directed := Envelope{To: stringPtr("reviewer.sess-xyz")}
	if !directed.IsDirected() || directed.IsBroadcast() {
		t.Fatalf("directed envelope helper mismatch")
	}
	broadcast := Envelope{}
	if broadcast.IsDirected() || !broadcast.IsBroadcast() {
		t.Fatalf("broadcast envelope helper mismatch")
	}
}

func TestValidateEnvelopeAndDecodeBodyErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	valid := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_direct_01",
		Kind:     KindSay,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceDirect),
		DirectID: stringPtr(testDirectRef().DirectID),
		From:     "coder.sess-abc",
		To:       stringPtr("reviewer.sess-xyz"),
		WorkID:   stringPtr("work_patch_42"),
		TS:       now.Unix(),
		Body:     mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
	}

	if err := ValidateEnvelope(valid, ValidateOptions{Now: now}); err != nil {
		t.Fatalf("ValidateEnvelope() error = %v", err)
	}

	if _, err := ParseEnvelope([]byte(`{"broken"`), ValidateOptions{Now: now}); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("ParseEnvelope(invalid json) error = %v, want ErrInvalidEnvelope", err)
	}

	if _, err := DecodeBody(Kind("mystery"), mustRawJSON(t, map[string]any{})); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("DecodeBody(invalid kind) error = %v, want ErrInvalidKind", err)
	}

	if _, err := DecodeBody(
		KindSay,
		mustRawJSON(t, []string{"not", "an", "object"}),
	); !errors.Is(
		err,
		ErrInvalidField,
	) {
		t.Fatalf("DecodeBody(non-object) error = %v, want ErrInvalidField", err)
	}
}

func TestAdditionalBodyValidationBranches(t *testing.T) {
	t.Parallel()

	if _, err := DecodeBody(KindWhois, mustRawJSON(t, map[string]any{
		"type": "request",
		"peer_card": map[string]any{
			"peer_id":               "reviewer.sess-xyz",
			"profiles_supported":    []string{"agh-network/v0"},
			"capabilities":          []string{"chat.review"},
			"artifacts_supported":   []string{"capability"},
			"trust_modes_supported": []string{"unverified"},
		},
	})); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("DecodeBody(whois request with peer_card) error = %v, want ErrInvalidBody", err)
	}

	if _, err := DecodeBody(KindGreet, mustRawJSON(t, map[string]any{
		"peer_card": map[string]any{
			"peer_id":               "reviewer.sess-xyz",
			"capabilities":          []string{"chat.review"},
			"artifacts_supported":   []string{"capability"},
			"trust_modes_supported": []string{"unverified"},
		},
	})); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("DecodeBody(greet missing profiles_supported) error = %v, want ErrInvalidBody", err)
	}

	if _, err := DecodeBody(KindTrace, mustRawJSON(t, map[string]any{
		"state": "unknown",
	})); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("DecodeBody(trace invalid state) error = %v, want ErrInvalidField", err)
	}

	if _, err := DecodeBody(KindReceipt, mustRawJSON(t, map[string]any{
		"for_id": "msg_direct_01",
		"status": "expired",
	})); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("DecodeBody(receipt missing reason_code) error = %v, want ErrInvalidBody", err)
	}
}

func TestWorkValidationAndTraceMatrix(t *testing.T) {
	t.Parallel()

	valid := Work{
		ID:        "work_patch_42",
		Ref:       testDirectRef(),
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     WorkStateWorking,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Work.Validate() error = %v", err)
	}

	invalid := valid
	invalid.Target = "BadPeer"
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("Work.Validate(invalid target) error = %v, want ErrInvalidField", err)
	}

	openErrEnv := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_trace_01",
		Kind:     KindTrace,
		Channel:  "builders",
		From:     "reviewer.sess-xyz",
		To:       stringPtr("coder.sess-abc"),
		WorkID:   stringPtr("work_patch_42"),
		TS:       time.Now().Unix(),
		Body:     mustRawJSON(t, map[string]any{"state": "working"}),
	}
	if _, err := OpenWork(withDirectSurface(openErrEnv), time.Time{}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("OpenWork(non-opener) error = %v, want ErrInvalidField", err)
	}

	matrix := []struct {
		name    string
		current WorkState
		next    WorkState
		want    bool
	}{
		{name: "ShouldAllowSubmittedToWorking", current: WorkStateSubmitted, next: WorkStateWorking, want: true},
		{name: "ShouldRejectSubmittedToSubmitted", current: WorkStateSubmitted, next: WorkStateSubmitted, want: false},
		{name: "ShouldAllowWorkingToCompleted", current: WorkStateWorking, next: WorkStateCompleted, want: true},
		{name: "ShouldRejectWorkingToSubmitted", current: WorkStateWorking, next: WorkStateSubmitted, want: false},
		{name: "ShouldAllowNeedsInputToWorking", current: WorkStateNeedsInput, next: WorkStateWorking, want: true},
		{name: "ShouldAllowNeedsInputToCanceled", current: WorkStateNeedsInput, next: WorkStateCanceled, want: true},
		{name: "ShouldRejectCompletedToWorking", current: WorkStateCompleted, next: WorkStateWorking, want: false},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := canApplyTrace(tc.current, tc.next); got != tc.want {
				t.Fatalf("canApplyTrace(%q, %q) = %v, want %v", tc.current, tc.next, got, tc.want)
			}
			err := ValidateWorkTransition(tc.current, tc.next)
			if tc.want && err != nil {
				t.Fatalf("ValidateWorkTransition(%q, %q) error = %v", tc.current, tc.next, err)
			}
			if !tc.want && !errors.Is(err, ErrInvalidStateTransition) {
				t.Fatalf(
					"ValidateWorkTransition(%q, %q) error = %v, want ErrInvalidStateTransition",
					tc.current,
					tc.next,
					err,
				)
			}
		})
	}
}

func TestAdditionalEnvelopeAndLifecycleBranches(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)

	greetMismatch := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_greet_01",
		Kind:     KindGreet,
		Channel:  "builders",
		From:     "coder.sess-abc",
		TS:       now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"peer_card": map[string]any{
				"peer_id":               "reviewer.sess-xyz",
				"profiles_supported":    []string{"agh-network/v0"},
				"capabilities":          []string{"workspace.patch.apply"},
				"artifacts_supported":   []string{"capability"},
				"trust_modes_supported": []string{"unverified"},
			},
		}),
	}
	if _, err := NormalizeEnvelope(greetMismatch, ValidateOptions{Now: now}); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("NormalizeEnvelope(greet mismatch) error = %v, want ErrInvalidBody", err)
	}

	receiptMissingWork := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_receipt_01",
		Kind:     KindReceipt,
		Channel:  "builders",
		Surface:  surfacePtr(SurfaceDirect),
		DirectID: stringPtr(testDirectRef().DirectID),
		From:     "reviewer.sess-xyz",
		TS:       now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "accepted",
		}),
	}
	if _, err := NormalizeEnvelope(
		receiptMissingWork,
		ValidateOptions{Now: now},
	); !errors.Is(
		err,
		ErrMissingField,
	) {
		t.Fatalf("NormalizeEnvelope(receipt missing work_id) error = %v, want ErrMissingField", err)
	}

	blankSay, err := DecodeBody(KindSay, mustRawJSON(t, map[string]any{"text": "   "}))
	if !errors.Is(err, ErrInvalidBody) || blankSay != nil {
		t.Fatalf("DecodeBody(blank say) body = %#v, error = %v, want nil + ErrInvalidBody", blankSay, err)
	}

	blankDirect, err := DecodeBody(KindSay, mustRawJSON(t, map[string]any{"text": "\n"}))
	if !errors.Is(err, ErrInvalidBody) || blankDirect != nil {
		t.Fatalf("DecodeBody(blank direct) body = %#v, error = %v, want nil + ErrInvalidBody", blankDirect, err)
	}

	if cloned := cloneRawMessage(nil); cloned != nil {
		t.Fatalf("cloneRawMessage(nil) = %#v, want nil", cloned)
	}
	if cloned := cloneRawMessage(jsonBytes(`{"ok":true}`)); string(cloned) != `{"ok":true}` {
		t.Fatalf("cloneRawMessage(non-nil) = %s, want original bytes", string(cloned))
	}

	traceEnv := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_trace_01",
		Kind:     KindTrace,
		Channel:  "builders",
		From:     "reviewer.sess-xyz",
		To:       stringPtr("coder.sess-abc"),
		WorkID:   stringPtr("work_patch_42"),
		TS:       now.Unix(),
		Body:     mustRawJSON(t, map[string]any{"state": "working"}),
	}
	if _, err := ApplyWorkEnvelope(nil, withDirectSurface(traceEnv), now); !errors.Is(err, ErrWorkNotFound) {
		t.Fatalf("ApplyWorkEnvelope(nil trace) error = %v, want ErrWorkNotFound", err)
	}

	terminal := &Work{
		ID:        "work_patch_42",
		Ref:       testDirectRef(),
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     WorkStateCanceled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	receiptEnv := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_receipt_02",
		Kind:     KindReceipt,
		Channel:  "builders",
		From:     "coder.sess-abc",
		To:       stringPtr("reviewer.sess-xyz"),
		WorkID:   stringPtr("work_patch_42"),
		TS:       now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "canceled",
		}),
	}
	got, err := ApplyWorkEnvelope(terminal, withDirectSurface(receiptEnv), now.Add(time.Second))
	if err != nil {
		t.Fatalf("ApplyWorkEnvelope(terminal receipt) error = %v", err)
	}
	if got.Action != LifecycleActionRejectWork {
		t.Fatalf("ApplyWorkEnvelope(terminal receipt).Action = %q, want %q", got.Action, LifecycleActionRejectWork)
	}
	if got.ReasonCode == nil || *got.ReasonCode != ReasonCodeWorkClosed {
		t.Fatalf("ApplyWorkEnvelope(terminal receipt).ReasonCode = %v, want %q", got.ReasonCode, ReasonCodeWorkClosed)
	}
}

func jsonBytes(value string) []byte {
	return []byte(value)
}
