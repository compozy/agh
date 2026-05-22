package extensionpkg

import (
	"strings"
	"testing"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestRenderInboundMessageFamilyLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		family   bridgepkg.InboundEventFamily
		envelope bridgepkg.InboundMessageEnvelope
		want     []string
	}{
		{
			name:   "Command family renders command details",
			family: bridgepkg.InboundEventFamilyCommand,
			envelope: bridgepkg.InboundMessageEnvelope{
				Command: &bridgepkg.InboundCommand{
					Command:   "deploy",
					Text:      "--force",
					TriggerID: "trg-1",
				},
			},
			want: []string{"Inbound bridge command", "Command: deploy", "Arguments: --force", "Trigger ID: trg-1"},
		},
		{
			name:   "Action family renders action details",
			family: bridgepkg.InboundEventFamilyAction,
			envelope: bridgepkg.InboundMessageEnvelope{
				Action: &bridgepkg.InboundAction{
					ActionID:  "approve",
					MessageID: "msg-1",
					Value:     "yes",
					TriggerID: "trg-2",
				},
			},
			want: []string{
				"Inbound bridge action",
				"Action ID: approve",
				"Message ID: msg-1",
				"Value: yes",
				"Trigger ID: trg-2",
			},
		},
		{
			name:   "Reaction family renders reaction details",
			family: bridgepkg.InboundEventFamilyReaction,
			envelope: bridgepkg.InboundMessageEnvelope{
				Reaction: &bridgepkg.InboundReaction{
					MessageID: "msg-2",
					Emoji:     ":eyes:",
					RawEmoji:  "U+1F440",
					Added:     false,
				},
			},
			want: []string{
				"Inbound bridge reaction",
				"Message ID: msg-2",
				"Emoji: :eyes:",
				"Raw emoji: U+1F440",
				"Change: removed",
			},
		},
		{
			name:   "Unknown family falls back to generic rendering",
			family: bridgepkg.InboundEventFamily("custom"),
			envelope: bridgepkg.InboundMessageEnvelope{
				PlatformMessageID: "msg-3",
			},
			want: []string{"Inbound bridge message", "Platform message ID: msg-3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lines := renderInboundMessageFamilyLines(tc.family, tc.envelope)
			rendered := strings.Join(lines, "\n")
			for _, want := range tc.want {
				if !strings.Contains(rendered, want) {
					t.Fatalf("rendered lines = %q, want substring %q", rendered, want)
				}
			}
		})
	}
}
