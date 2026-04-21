package daemon

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

type networkCorrelationExpectation struct {
	MessageID       string
	Kind            string
	InteractionID   string
	ReplyTo         string
	TraceID         string
	AuditDirections []string
}

type networkAuditExpectation struct {
	MessageID string
	Direction string
	Kind      string
	Reason    string
}

func validateNetworkCorrelationSurfaces(
	messages []transcript.UIMessage,
	audit []store.NetworkAuditEntry,
	expectation networkCorrelationExpectation,
) error {
	checks := []struct {
		label  string
		needle string
	}{
		{label: "message id", needle: attributeNeedle("id", expectation.MessageID)},
		{label: "kind", needle: attributeNeedle("kind", expectation.Kind)},
		{label: "interaction", needle: attributeNeedle("interaction", expectation.InteractionID)},
		{label: "reply-to", needle: attributeNeedle("reply-to", expectation.ReplyTo)},
		{label: "trace-id", needle: attributeNeedle("trace-id", expectation.TraceID)},
	}

	matched := false
	for _, message := range messages {
		content := strings.TrimSpace(transcript.UIMessageText(message))
		if content == "" {
			continue
		}

		allPresent := true
		for _, check := range checks {
			if check.needle != "" && !strings.Contains(content, check.needle) {
				allPresent = false
				break
			}
		}
		if allPresent {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("transcript missing correlated attributes for message %q", expectation.MessageID)
	}

	for _, direction := range expectation.AuditDirections {
		if err := validateNetworkAuditEntry(audit, networkAuditExpectation{
			MessageID: expectation.MessageID,
			Direction: direction,
			Kind:      expectation.Kind,
		}); err != nil {
			return err
		}
	}

	return nil
}

func validateNetworkAuditEntry(
	entries []store.NetworkAuditEntry,
	expectation networkAuditExpectation,
) error {
	for _, entry := range entries {
		if strings.TrimSpace(entry.MessageID) != strings.TrimSpace(expectation.MessageID) {
			continue
		}
		if strings.TrimSpace(entry.Direction) != strings.TrimSpace(expectation.Direction) {
			continue
		}
		if strings.TrimSpace(entry.Kind) != strings.TrimSpace(expectation.Kind) {
			continue
		}
		if trimmedReason := strings.TrimSpace(expectation.Reason); trimmedReason != "" &&
			strings.TrimSpace(entry.Reason) != trimmedReason {
			continue
		}
		return nil
	}

	return fmt.Errorf(
		"audit missing message_id=%q direction=%q kind=%q reason=%q",
		expectation.MessageID,
		expectation.Direction,
		expectation.Kind,
		expectation.Reason,
	)
}

func attributeNeedle(name string, value string) string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return ""
	}
	return name + `="` + trimmedValue + `"`
}

func TestValidateNetworkCorrelationSurfacesUsesTargetedAttributes(t *testing.T) {
	t.Parallel()

	messages := []transcript.UIMessage{
		{
			Role: transcript.UIRoleAssistant,
			Parts: []transcript.UIMessagePart{
				{
					Type:  "text",
					Text:  `<network-message id="msg_direct_01" kind="direct" interaction="int_patch_42" reply-to="msg_say_01" trace-id="trace_ops_patch_42"></network-message>`,
					State: "done",
				},
			},
		},
	}
	audit := []store.NetworkAuditEntry{
		{MessageID: "msg_direct_01", Direction: "sent", Kind: "direct"},
		{MessageID: "msg_direct_01", Direction: "delivered", Kind: "direct"},
	}

	if err := validateNetworkCorrelationSurfaces(messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_direct_01",
		Kind:            "direct",
		InteractionID:   "int_patch_42",
		ReplyTo:         "msg_say_01",
		TraceID:         "trace_ops_patch_42",
		AuditDirections: []string{"sent", "delivered"},
	}); err != nil {
		t.Fatalf("validateNetworkCorrelationSurfaces() error = %v", err)
	}
}

func TestValidateNetworkCorrelationSurfacesRejectsSplitTranscriptMatches(t *testing.T) {
	t.Parallel()

	messages := []transcript.UIMessage{
		{
			Role: transcript.UIRoleAssistant,
			Parts: []transcript.UIMessagePart{{
				Type:  "text",
				Text:  `<network-message id="msg_direct_01" kind="direct"></network-message>`,
				State: "done",
			}},
		},
		{
			Role: transcript.UIRoleAssistant,
			Parts: []transcript.UIMessagePart{
				{
					Type:  "text",
					Text:  `<network-message interaction="int_patch_42" reply-to="msg_say_01" trace-id="trace_ops_patch_42"></network-message>`,
					State: "done",
				},
			},
		},
	}
	audit := []store.NetworkAuditEntry{
		{MessageID: "msg_direct_01", Direction: "sent", Kind: "direct"},
		{MessageID: "msg_direct_01", Direction: "delivered", Kind: "direct"},
	}

	if err := validateNetworkCorrelationSurfaces(messages, audit, networkCorrelationExpectation{
		MessageID:       "msg_direct_01",
		Kind:            "direct",
		InteractionID:   "int_patch_42",
		ReplyTo:         "msg_say_01",
		TraceID:         "trace_ops_patch_42",
		AuditDirections: []string{"sent", "delivered"},
	}); err == nil {
		t.Fatal("validateNetworkCorrelationSurfaces() error = nil, want split-message correlation failure")
	}
}

func TestValidateNetworkAuditEntryMatchesDuplicateRejection(t *testing.T) {
	t.Parallel()

	entries := []store.NetworkAuditEntry{
		{
			MessageID: "msg_direct_01",
			Direction: "rejected",
			Kind:      "direct",
			Reason:    "duplicate",
		},
	}

	if err := validateNetworkAuditEntry(entries, networkAuditExpectation{
		MessageID: "msg_direct_01",
		Direction: "rejected",
		Kind:      "direct",
		Reason:    "duplicate",
	}); err != nil {
		t.Fatalf("validateNetworkAuditEntry() error = %v", err)
	}
}
