package network

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
)

const networkWakeTrustUntrusted = "untrusted"

// FormatNetworkWakePrompt renders only committed message content plus a compact wake header.
func FormatNetworkWakePrompt(
	messages []store.NetworkMessageEntry,
	targetSessionID string,
) (string, acp.PromptNetworkMeta, error) {
	if len(messages) == 0 {
		return "", acp.PromptNetworkMeta{}, errors.New("network: wake requires at least one message")
	}
	target := strings.TrimSpace(targetSessionID)
	if target == "" {
		return "", acp.PromptNetworkMeta{}, errors.New("network: wake target session id is required")
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Network wake: %d committed message(s).\n", len(messages))
	for _, message := range messages {
		body, err := DecodeBody(Kind(message.Kind), message.Body)
		if err != nil {
			return "", acp.PromptNetworkMeta{}, fmt.Errorf(
				"network: decode wake message %q: %w",
				message.MessageID,
				err,
			)
		}
		text := previewForBody(body)
		fmt.Fprintf(
			&builder,
			"- %s from %s: %s\n",
			strings.TrimSpace(message.MessageID),
			strings.TrimSpace(message.PeerFrom),
			strings.TrimSpace(text),
		)
	}
	first := messages[0]
	meta := acp.PromptNetworkMeta{
		MessageID:    first.MessageID,
		Kind:         first.Kind,
		Channel:      first.Channel,
		Surface:      first.Surface,
		ThreadID:     first.ThreadID,
		DirectID:     first.DirectID,
		From:         first.PeerFrom,
		To:           first.PeerTo,
		Mentions:     append([]string(nil), first.Mentions...),
		WorkID:       first.WorkID,
		ReplyTo:      first.ReplyTo,
		TraceID:      first.TraceID,
		CausationID:  first.CausationID,
		Trust:        networkWakeTrustUntrusted,
		DeliveryMode: wakeDeliveryMode(first, target),
	}
	return strings.TrimSpace(builder.String()), meta.Normalize(), nil
}

func wakeDeliveryMode(message store.NetworkMessageEntry, targetSessionID string) string {
	if strings.TrimSpace(message.PeerTo) == strings.TrimSpace(targetSessionID) {
		return store.NetworkWakeTriggerDirect
	}
	return store.NetworkWakeTriggerMention
}
