package network

import (
	"errors"
	"testing"
	"time"
)

func TestEnumValidationAndBodyKindHelpers(t *testing.T) {
	t.Parallel()

	validKinds := []Kind{KindGreet, KindWhois, KindSay, KindDirect, KindRecipe, KindReceipt, KindTrace}
	for _, kind := range validKinds {
		kind := kind
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
		status := status
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

	if err := StateWorking.Validate(); err != nil {
		t.Fatalf("StateWorking.Validate() error = %v", err)
	}
	if err := InteractionState("drifting").Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("InteractionState(drifting).Validate() error = %v, want ErrInvalidField", err)
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
	if got := (DirectBody{}).Kind(); got != KindDirect {
		t.Fatalf("DirectBody.Kind() = %q, want %q", got, KindDirect)
	}
	if got := (RecipeBody{}).Kind(); got != KindRecipe {
		t.Fatalf("RecipeBody.Kind() = %q, want %q", got, KindRecipe)
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
		Protocol:      ProtocolV0,
		ID:            "msg_direct_01",
		Kind:          KindDirect,
		Space:         "builders",
		From:          "coder.sess-abc",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            now.Unix(),
		Body:          mustRawJSON(t, map[string]any{"text": "please review auth.go"}),
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

	if _, err := DecodeBody(KindSay, mustRawJSON(t, []string{"not", "an", "object"})); !errors.Is(err, ErrInvalidField) {
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
			"artifacts_supported":   []string{"recipe"},
			"trust_modes_supported": []string{"unverified"},
		},
	})); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("DecodeBody(whois request with peer_card) error = %v, want ErrInvalidBody", err)
	}

	if _, err := DecodeBody(KindGreet, mustRawJSON(t, map[string]any{
		"peer_card": map[string]any{
			"peer_id":               "reviewer.sess-xyz",
			"capabilities":          []string{"chat.review"},
			"artifacts_supported":   []string{"recipe"},
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

func TestInteractionValidationAndTraceMatrix(t *testing.T) {
	t.Parallel()

	valid := Interaction{
		ID:        "int_patch_42",
		Space:     "builders",
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     StateWorking,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Interaction.Validate() error = %v", err)
	}

	invalid := valid
	invalid.Target = "BadPeer"
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("Interaction.Validate(invalid target) error = %v, want ErrInvalidField", err)
	}

	openErrEnv := Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_trace_01",
		Kind:          KindTrace,
		Space:         "builders",
		From:          "reviewer.sess-xyz",
		To:            stringPtr("coder.sess-abc"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            time.Now().Unix(),
		Body:          mustRawJSON(t, map[string]any{"state": "working"}),
	}
	if _, err := OpenInteraction(openErrEnv, time.Time{}); !errors.Is(err, ErrInvalidField) {
		t.Fatalf("OpenInteraction(non-direct) error = %v, want ErrInvalidField", err)
	}

	matrix := []struct {
		name    string
		current InteractionState
		next    InteractionState
		want    bool
	}{
		{name: "ShouldAllowSubmittedToWorking", current: StateSubmitted, next: StateWorking, want: true},
		{name: "ShouldRejectSubmittedToSubmitted", current: StateSubmitted, next: StateSubmitted, want: false},
		{name: "ShouldAllowWorkingToCompleted", current: StateWorking, next: StateCompleted, want: true},
		{name: "ShouldRejectWorkingToSubmitted", current: StateWorking, next: StateSubmitted, want: false},
		{name: "ShouldAllowNeedsInputToWorking", current: StateNeedsInput, next: StateWorking, want: true},
		{name: "ShouldAllowNeedsInputToCanceled", current: StateNeedsInput, next: StateCanceled, want: true},
		{name: "ShouldRejectCompletedToWorking", current: StateCompleted, next: StateWorking, want: false},
	}

	for _, tc := range matrix {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := canApplyTrace(tc.current, tc.next); got != tc.want {
				t.Fatalf("canApplyTrace(%q, %q) = %v, want %v", tc.current, tc.next, got, tc.want)
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
		Space:    "builders",
		From:     "coder.sess-abc",
		TS:       now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"peer_card": map[string]any{
				"peer_id":               "reviewer.sess-xyz",
				"profiles_supported":    []string{"agh-network/v0"},
				"capabilities":          []string{"workspace.patch.apply"},
				"artifacts_supported":   []string{"recipe"},
				"trust_modes_supported": []string{"unverified"},
			},
		}),
	}
	if _, err := NormalizeEnvelope(greetMismatch, ValidateOptions{Now: now}); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("NormalizeEnvelope(greet mismatch) error = %v, want ErrInvalidBody", err)
	}

	receiptMissingInteraction := Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_receipt_01",
		Kind:     KindReceipt,
		Space:    "builders",
		From:     "reviewer.sess-xyz",
		TS:       now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "accepted",
		}),
	}
	if _, err := NormalizeEnvelope(receiptMissingInteraction, ValidateOptions{Now: now}); !errors.Is(err, ErrMissingField) {
		t.Fatalf("NormalizeEnvelope(receipt missing interaction_id) error = %v, want ErrMissingField", err)
	}

	blankSay, err := DecodeBody(KindSay, mustRawJSON(t, map[string]any{"text": "   "}))
	if !errors.Is(err, ErrInvalidBody) || blankSay != nil {
		t.Fatalf("DecodeBody(blank say) body = %#v, error = %v, want nil + ErrInvalidBody", blankSay, err)
	}

	blankDirect, err := DecodeBody(KindDirect, mustRawJSON(t, map[string]any{"text": "\n"}))
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
		Protocol:      ProtocolV0,
		ID:            "msg_trace_01",
		Kind:          KindTrace,
		Space:         "builders",
		From:          "reviewer.sess-xyz",
		To:            stringPtr("coder.sess-abc"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            now.Unix(),
		Body:          mustRawJSON(t, map[string]any{"state": "working"}),
	}
	if _, err := ApplyInteractionEnvelope(nil, traceEnv, now); !errors.Is(err, ErrInteractionNotFound) {
		t.Fatalf("ApplyInteractionEnvelope(nil trace) error = %v, want ErrInteractionNotFound", err)
	}

	terminal := &Interaction{
		ID:        "int_patch_42",
		Space:     "builders",
		Initiator: "coder.sess-abc",
		Target:    "reviewer.sess-xyz",
		State:     StateCanceled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	receiptEnv := Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_receipt_02",
		Kind:          KindReceipt,
		Space:         "builders",
		From:          "coder.sess-abc",
		To:            stringPtr("reviewer.sess-xyz"),
		InteractionID: stringPtr("int_patch_42"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, map[string]any{
			"for_id": "msg_direct_01",
			"status": "canceled",
		}),
	}
	got, err := ApplyInteractionEnvelope(terminal, receiptEnv, now.Add(time.Second))
	if err != nil {
		t.Fatalf("ApplyInteractionEnvelope(terminal receipt) error = %v", err)
	}
	if got.Action != LifecycleActionIgnored {
		t.Fatalf("ApplyInteractionEnvelope(terminal receipt).Action = %q, want %q", got.Action, LifecycleActionIgnored)
	}
}

func jsonBytes(value string) []byte {
	return []byte(value)
}
