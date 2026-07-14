package network

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func (m *Manager) recordAcceptedMessage(
	senderSessionID string,
	envelope Envelope,
	dispositions []store.NetworkMessageDisposition,
) {
	if m == nil {
		return
	}
	if m.stats != nil {
		m.stats.recordSent(envelope)
	}
	m.logger.Info(
		"network.message.accepted",
		networkLogFields(envelope, "session_id", senderSessionID)...,
	)
	for _, disposition := range dispositions {
		if disposition.Decision != store.NetworkDispositionDeliver {
			continue
		}
		if m.stats != nil {
			m.stats.recordReceived(envelope)
			m.stats.recordDelivered(envelope)
		}
		m.logger.Info(
			"network.message.dispatched",
			networkLogFields(
				envelope,
				"session_id", disposition.RecipientSessionID,
				"dispatch", "durable_notification",
			)...,
		)
	}
}

func (m *Manager) recordAuditRejected(
	_ context.Context,
	sessionID string,
	envelope Envelope,
	reason string,
) {
	if m == nil {
		return
	}
	if m.stats != nil {
		m.stats.recordRejected(envelope)
	}
	fields := networkLogFields(envelope, "session_id", sessionID)
	fields = append(fields, "reason", strings.TrimSpace(reason))
	m.logger.Info("network.message.rejected", fields...)
}

func networkLogFields(envelope Envelope, extra ...any) []any {
	fields := []any{
		"message_id", strings.TrimSpace(envelope.ID),
		"kind", string(envelope.Kind),
		managerChannelKey, strings.TrimSpace(envelope.Channel),
		"from", strings.TrimSpace(envelope.From),
	}
	if envelope.To != nil {
		fields = append(fields, "to", strings.TrimSpace(*envelope.To))
	}
	if len(envelope.Mentions) > 0 {
		fields = append(fields, "mentions", strings.Join(normalizeEnvelopeMentions(envelope.Mentions), ","))
	}
	if envelope.ReplyTo != nil {
		fields = append(fields, "reply_to", strings.TrimSpace(*envelope.ReplyTo))
	}
	if envelope.Surface != nil {
		fields = append(fields, "surface", strings.TrimSpace(string(*envelope.Surface)))
	}
	if envelope.ThreadID != nil {
		fields = append(fields, "thread_id", strings.TrimSpace(*envelope.ThreadID))
	}
	if envelope.DirectID != nil {
		fields = append(fields, "direct_id", strings.TrimSpace(*envelope.DirectID))
	}
	if envelope.WorkID != nil {
		fields = append(fields, "work_id", strings.TrimSpace(*envelope.WorkID))
	}
	if envelope.TraceID != nil {
		fields = append(fields, "trace_id", strings.TrimSpace(*envelope.TraceID))
	}
	if envelope.CausationID != nil {
		fields = append(fields, "causation_id", strings.TrimSpace(*envelope.CausationID))
	}
	for _, key := range []string{
		"agh.workflow_id",
		"agh.handoff_version",
		"agh.handoff_digest",
		"agh.handoff_source",
	} {
		if value, ok := extensionLogValue(envelope.Ext, key); ok {
			fields = append(fields, key, value)
		}
	}
	return append(fields, extra...)
}

func extensionLogValue(ext ExtensionMap, key string) (string, bool) {
	if len(ext) == 0 {
		return "", false
	}
	raw, ok := ext[key]
	if !ok || len(raw) == 0 {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		text = strings.TrimSpace(text)
		if text != "" {
			return text, true
		}
	}
	compacted := compactJSON(raw)
	return compacted, compacted != ""
}

func hasWorkflowID(ext ExtensionMap) bool {
	_, ok := extensionLogValue(ext, "agh.workflow_id")
	return ok
}

func hasHandoffVersion(ext ExtensionMap) bool {
	_, ok := extensionLogValue(ext, "agh.handoff_version")
	return ok
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return strings.TrimSpace(compacted.String())
}
