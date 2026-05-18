package main

import (
	"context"
	"net/url"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestSlackContractIngressRouting(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 16, 13, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-slack")
	managed.Instance.RoutingPolicy = bridgepkg.RoutingPolicy{
		IncludePeer:   true,
		IncludeThread: true,
	}

	t.Run("Should map root direct messages with a thread dimension", func(t *testing.T) {
		t.Parallel()

		mapped, ignored, err := mapSlackMessageEvent(slackMessageEvent{
			Channel:     "D123",
			ChannelType: "im",
			Text:        "hello",
			TS:          "1775866805.100000",
			Type:        "message",
			User:        "U123",
		}, managed, now, "EvMessage", "T123", now.Unix())
		if err != nil {
			t.Fatalf("mapSlackMessageEvent() error = %v", err)
		}
		if ignored {
			t.Fatal("mapSlackMessageEvent() ignored = true, want false")
		}
		assertSlackRoutableDirectEnvelope(t, managed.Instance, mapped.Envelope, "1775866805.100000")
	})

	t.Run("Should map direct slash commands with a stable root thread dimension", func(t *testing.T) {
		t.Parallel()

		mapped, err := mapSlackSlashCommand(url.Values{
			"channel_id":   {"D123"},
			"channel_name": {"directmessage"},
			"command":      {"/agh"},
			"text":         {"status"},
			"user_id":      {"U123"},
		}, managed, now)
		if err != nil {
			t.Fatalf("mapSlackSlashCommand() error = %v", err)
		}
		assertSlackRoutableDirectEnvelope(t, managed.Instance, mapped.Envelope, "D123")
	})

	t.Run("Should map direct reactions with the reacted message as thread dimension", func(t *testing.T) {
		t.Parallel()

		mapped, err := mapSlackReactionEvent(slackReactionEvent{
			EventTS:  "1775866806.100000",
			Item:     slackReactionItem{Channel: "D123", TS: "1775866805.100000", Type: "message"},
			Reaction: "thumbsup",
			Type:     "reaction_added",
			User:     "U123",
		}, managed, now, "EvReaction", "T123")
		if err != nil {
			t.Fatalf("mapSlackReactionEvent() error = %v", err)
		}
		assertSlackRoutableDirectEnvelope(t, managed.Instance, mapped.Envelope, "1775866805.100000")
	})
}

func TestSlackContractDeliveryRecovery(t *testing.T) {
	t.Parallel()

	t.Run("Should reconcile a lost create ACK without posting a duplicate message", func(t *testing.T) {
		t.Parallel()

		api := &recordingSlackAPI{}
		startReq := testDeliveryRequest(
			"brg-slack",
			"delivery-lost-ack",
			1,
			bridgepkg.DeliveryEventTypeStart,
			false,
		)
		startAck, _, err := executeDelivery(context.Background(), api, startReq, deliveryState{})
		if err != nil {
			t.Fatalf("executeDelivery(start) error = %v", err)
		}

		resumeReq := testDeliveryRequest(
			"brg-slack",
			"delivery-lost-ack",
			1,
			bridgepkg.DeliveryEventTypeResume,
			false,
		)
		resumeReq.Event.Resume = &bridgepkg.DeliveryResumeState{LatestEventType: bridgepkg.DeliveryEventTypeStart}
		resumeReq.Snapshot = &bridgepkg.DeliverySnapshot{
			DeliveryID:       resumeReq.Event.DeliveryID,
			SessionID:        "sess-slack",
			TurnID:           "turn-slack",
			BridgeInstanceID: resumeReq.Event.BridgeInstanceID,
			RoutingKey:       resumeReq.Event.RoutingKey,
			DeliveryTarget:   resumeReq.Event.DeliveryTarget,
			LatestSeq:        resumeReq.Event.Seq,
			LatestEventType:  bridgepkg.DeliveryEventTypeStart,
			CurrentContent:   resumeReq.Event.Content,
			LastSentSeq:      resumeReq.Event.Seq,
			LastAckedSeq:     0,
			UpdatedAt:        time.Date(2026, 5, 16, 13, 1, 0, 0, time.UTC),
		}
		resumeAck, _, err := executeDelivery(context.Background(), api, resumeReq, deliveryState{})
		if err != nil {
			t.Fatalf("executeDelivery(resume) error = %v", err)
		}
		if got, want := resumeAck.RemoteMessageID, startAck.RemoteMessageID; got != want {
			t.Fatalf("resumeAck.RemoteMessageID = %q, want %q", got, want)
		}
		if got, want := len(api.posts), 1; got != want {
			t.Fatalf("PostMessage calls = %d, want %d", got, want)
		}
	})
}

func assertSlackRoutableDirectEnvelope(
	t *testing.T,
	instance bridgepkg.BridgeInstance,
	envelope bridgepkg.InboundMessageEnvelope,
	wantThreadID string,
) {
	t.Helper()

	if got := envelope.ThreadID; got != wantThreadID {
		t.Fatalf("Envelope.ThreadID = %q, want %q", got, wantThreadID)
	}
	if _, err := bridgepkg.BuildRoutingKey(instance, bridgepkg.RoutingDimensions{
		PeerID:   envelope.PeerID,
		ThreadID: envelope.ThreadID,
		GroupID:  envelope.GroupID,
	}); err != nil {
		t.Fatalf("BuildRoutingKey() error = %v", err)
	}
}

type recordingSlackAPI struct {
	posts    []slackPostMessageRequest
	messages []slackConversationMessage
}

func (a *recordingSlackAPI) AuthTest(context.Context) (*slackAuthIdentity, error) {
	return &slackAuthIdentity{UserID: "U_BOT", BotID: "B_BOT"}, nil
}

func (a *recordingSlackAPI) PostMessage(
	_ context.Context,
	req slackPostMessageRequest,
) (*slackPostedMessage, error) {
	ts := "1775866805.100000"
	if len(a.posts) > 0 {
		ts = "1775866805.200000"
	}
	a.posts = append(a.posts, req)
	a.messages = append(a.messages, slackConversationMessage{
		TS:       ts,
		Metadata: req.Metadata,
	})
	return &slackPostedMessage{TS: ts}, nil
}

func (a *recordingSlackAPI) FindDeliveryMessage(
	_ context.Context,
	req slackFindDeliveryMessageRequest,
) (*slackPostedMessage, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	for idx := range a.messages {
		message := a.messages[idx]
		if slackMetadataMatchesDelivery(message.Metadata, req) {
			return &slackPostedMessage{TS: message.TS}, nil
		}
	}
	return nil, nil
}

func (a *recordingSlackAPI) UpdateMessage(context.Context, slackUpdateMessageRequest) error {
	return nil
}

func (a *recordingSlackAPI) DeleteMessage(context.Context, slackDeleteMessageRequest) error {
	return nil
}
