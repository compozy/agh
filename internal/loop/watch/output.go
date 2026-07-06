package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	outputKindPending   = "watch_source_pending"
	outputKindConfirmed = "watch_source_confirmed"
)

// OutputRef is the JSON payload persisted in loop_generation_outputs.output_ref
// for coordinator-owned watch-source nodes.
type OutputRef struct {
	Kind        string          `json:"kind"`
	StateDigest string          `json:"state_digest,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	SettledAt   *time.Time      `json:"settled_at,omitempty"`
}

// PendingOutputRef stores the last observed source digest while a loop is watching.
func PendingOutputRef(response PollResponse) (string, error) {
	return marshalOutputRef(OutputRef{
		Kind:        outputKindPending,
		StateDigest: strings.TrimSpace(response.StateDigest),
	})
}

// ConfirmedOutputRef stores the confirmed ready payload as the source node output.
func ConfirmedOutputRef(response PollResponse) (string, error) {
	return marshalOutputRef(OutputRef{
		Kind:        outputKindConfirmed,
		StateDigest: strings.TrimSpace(response.StateDigest),
		Payload:     cloneRawMessage(response.Payload),
		SettledAt:   response.SettledAt,
	})
}

// ExpectedStateDigestFromOutputRef recovers the prior state digest for the next poll.
func ExpectedStateDigestFromOutputRef(ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", nil
	}
	var output OutputRef
	if err := json.Unmarshal([]byte(trimmed), &output); err != nil {
		return "", fmt.Errorf("loop watch output ref decode: %w", err)
	}
	if strings.TrimSpace(output.Kind) != outputKindPending {
		return "", nil
	}
	return strings.TrimSpace(output.StateDigest), nil
}

func marshalOutputRef(output OutputRef) (string, error) {
	if strings.TrimSpace(output.Kind) == "" {
		return "", errors.New("loop watch output kind is required")
	}
	payload, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("loop watch output ref: %w", err)
	}
	return string(payload), nil
}

func cloneRawMessage(src json.RawMessage) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}
