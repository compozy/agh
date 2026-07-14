package network

import (
	"encoding/json"
	"strings"
)

func previewForBody(body Body) string {
	switch value := body.(type) {
	case GreetBody:
		return ResolveGreetSummary(value.PeerCard, value.Summary)
	case WhoisBody:
		if value.Type == WhoisTypeRequest {
			return strings.TrimSpace(value.Query)
		}
		return ""
	case SayBody:
		return strings.TrimSpace(value.Text)
	case CapabilityBody:
		if summary := strings.TrimSpace(value.Capability.Summary); summary != "" {
			return summary
		}
		if outcome := strings.TrimSpace(value.Capability.Outcome); outcome != "" {
			return outcome
		}
		return strings.TrimSpace(value.Capability.ID)
	case ReceiptBody:
		if value.Detail != nil {
			return strings.TrimSpace(*value.Detail)
		}
		return ""
	case TraceBody:
		return strings.TrimSpace(value.Message)
	default:
		return ""
	}
}

// PreviewTextForRawBody derives operator-facing preview text from persisted JSON.
func PreviewTextForRawBody(kind Kind, raw json.RawMessage) string {
	body, err := DecodeBody(kind, raw)
	if err != nil {
		return ""
	}
	return previewForBody(body)
}
