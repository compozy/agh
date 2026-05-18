//go:build integration

package network

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pedronauck/agh/internal/acp"
	sessionpkg "github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cfg := testManagerConfig()
	cfg.GreetInterval = 60
	fixedNow := time.Date(2026, 4, 19, 4, 0, 0, 0, time.UTC)
	manager, err := NewManager(
		ctx,
		cfg,
		newFakeDeliveryPrompter(),
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	subject, err := BroadcastSubject(testWorkspaceID, "builders")
	if err != nil {
		t.Fatalf("BroadcastSubject() error = %v", err)
	}
	published := make(chan Envelope, 4)
	subscribeErr := make(chan error, 1)
	subscription, err := manager.transport.Subscribe(subject, func(msg *nats.Msg) {
		envelope, parseErr := ParseEnvelope(msg.Data, ValidateOptions{Now: fixedNow})
		if parseErr != nil {
			select {
			case subscribeErr <- parseErr:
			default:
			}
			return
		}
		select {
		case published <- envelope:
		default:
		}
	})
	if err != nil {
		t.Fatalf("Subscribe(%q) error = %v", subject, err)
	}
	t.Cleanup(func() {
		if err := subscription.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			t.Fatalf("subscription.Unsubscribe() error = %v", err)
		}
	})

	capabilities := []sessionpkg.NetworkPeerCapability{
		{ID: "review-pr", Summary: "Review pull requests"},
		{ID: "draft-spec", Summary: "Draft technical specs"},
	}
	if err := manager.JoinChannel(
		ctx,
		sessionpkg.NetworkPeerJoin{
			SessionID:    "sess-capable",
			PeerID:       "reviewer.sess-capable",
			DisplayName:  "Reviewer",
			Channel:      "builders",
			Capabilities: append([]sessionpkg.NetworkPeerCapability(nil), capabilities...),
		},
	); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	assertPublishedBriefGreet := func(label string) {
		t.Helper()

		select {
		case err := <-subscribeErr:
			t.Fatalf("%s subscribe error = %v", label, err)
		case envelope := <-published:
			if got, want := envelope.Kind, KindGreet; got != want {
				t.Fatalf("%s envelope kind = %q, want %q", label, got, want)
			}
			decoded, err := envelope.DecodeBody()
			if err != nil {
				t.Fatalf("%s DecodeBody() error = %v", label, err)
			}
			body := decoded.(GreetBody)
			if body.PeerCard.DisplayName == nil || *body.PeerCard.DisplayName != "Reviewer" {
				t.Fatalf("%s greet display name = %#v, want Reviewer", label, body.PeerCard.DisplayName)
			}
			if got, want := body.Summary, "Reviewer ready for Review pull requests +1 more"; got != want {
				t.Fatalf("%s greet summary = %q, want %q", label, got, want)
			}
			if got, want := body.PeerCard.Capabilities, []string{"review-pr", "draft-spec"}; !slices.Equal(got, want) {
				t.Fatalf("%s greet capabilities = %#v, want %#v", label, got, want)
			}
			if got := decodeCapabilityBriefPayload(
				t,
				body.PeerCard.Ext[capabilityBriefExtKey],
			); !slices.Equal(
				got,
				[]capabilityBrief{
					{ID: "review-pr", Summary: "Review pull requests"},
					{ID: "draft-spec", Summary: "Draft technical specs"},
				},
			) {
				t.Fatalf("%s greet capability brief = %#v, want projected brief entries", label, got)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s greet: %v", label, ctx.Err())
		}
	}

	assertPublishedBriefGreet("initial")

	peers, err := manager.ListPeers(ctx, testWorkspaceID, "builders")
	if err != nil {
		t.Fatalf("ListPeers() error = %v", err)
	}
	if got, want := len(peers), 1; got != want {
		t.Fatalf("len(ListPeers()) = %d, want %d", got, want)
	}
	if got := decodeCapabilityBriefPayload(
		t,
		peers[0].PeerCard.Ext[capabilityBriefExtKey],
	); !slices.Equal(
		got,
		[]capabilityBrief{
			{ID: "review-pr", Summary: "Review pull requests"},
			{ID: "draft-spec", Summary: "Draft technical specs"},
		},
	) {
		t.Fatalf("listed peer capability brief = %#v, want projected brief entries", got)
	}

	manager.handleReconnect()
	assertPublishedBriefGreet("reconnect")
}

func TestManagerPersistsRuntimeConversationSurfacesAndHandoff(t *testing.T) {
	t.Parallel()

	t.Run("Should persist public direct handoff and summarize-back conversations", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		db := openNetworkRuntimeDB(t)
		prompter := newFakeDeliveryPrompter()
		fixedNow := time.Date(2026, 5, 5, 13, 0, 0, 0, time.UTC)
		var currentUnix atomic.Int64
		currentUnix.Store(fixedNow.Unix())
		cfg := testManagerConfig()
		cfg.GreetInterval = 60
		manager, err := NewManager(
			ctx,
			cfg,
			prompter,
			filepath.Join(t.TempDir(), "network.audit"),
			db,
			WithManagerLogger(discardManagerLogger()),
			WithManagerClock(func() time.Time { return time.Unix(currentUnix.Load(), 0).UTC() }),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		localPeerID := "reviewer.sess-local"
		remotePeerID := "coder.sess-remote"
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-local", localPeerID, "builders")); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}
		remoteCard, err := DefaultPeerCard(remotePeerID)
		if err != nil {
			t.Fatalf("DefaultPeerCard(remote) error = %v", err)
		}
		if _, stored, err := manager.peers.RefreshRemote(
			testWorkspaceID,
			"builders",
			remoteCard,
			fixedNow,
		); err != nil {
			t.Fatalf("RefreshRemote(remote) error = %v", err)
		} else if !stored {
			t.Fatal("RefreshRemote(remote) stored = false, want true")
		}

		threadEnvelope := withThreadSurface(Envelope{
			Protocol:    ProtocolV0,
			WorkspaceID: testWorkspaceID,
			ID:          "msg-thread-runtime",
			Kind:        KindSay,
			Channel:     "builders",
			From:        remotePeerID,
			TS:          fixedNow.Unix(),
			Body:        mustRawJSON(t, SayBody{Text: "public thread request"}),
		})
		deliverInboundEnvelope(t, manager, threadEnvelope)
		prompter.waitForCalls(t, 1)
		threadCall := prompter.call(0)
		if got, want := threadCall.meta.Surface, string(SurfaceThread); got != want {
			t.Fatalf("thread prompt surface = %q, want %q", got, want)
		}
		if got, want := threadCall.meta.ThreadID, testThreadRef().ThreadID; got != want {
			t.Fatalf("thread prompt thread_id = %q, want %q", got, want)
		}
		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow})

		directID, _, _, err := DirectRoomIdentity(testWorkspaceID, "builders", remotePeerID, localPeerID)
		if err != nil {
			t.Fatalf("DirectRoomIdentity(inbound) error = %v", err)
		}
		directEnvelope := Envelope{
			Protocol:    ProtocolV0,
			WorkspaceID: testWorkspaceID,
			ID:          "msg-direct-runtime",
			Kind:        KindSay,
			Channel:     "builders",
			Surface:     surfacePtr(SurfaceDirect),
			DirectID:    stringPtr(directID),
			From:        remotePeerID,
			To:          stringPtr(localPeerID),
			WorkID:      stringPtr("work_direct_runtime"),
			ReplyTo:     stringPtr("msg-thread-runtime"),
			TraceID:     stringPtr("trace-runtime-thread"),
			CausationID: stringPtr("msg-thread-runtime"),
			TS:          fixedNow.Add(time.Second).Unix(),
			Body:        mustRawJSON(t, SayBody{Text: "restricted direct detail"}),
		}
		currentUnix.Store(fixedNow.Add(time.Second).Unix())
		deliverInboundEnvelope(t, manager, directEnvelope)
		prompter.waitForCalls(t, 2)
		directCall := prompter.call(1)
		if got, want := directCall.meta.Surface, string(SurfaceDirect); got != want {
			t.Fatalf("direct prompt surface = %q, want %q", got, want)
		}
		if got, want := directCall.meta.DirectID, directID; got != want {
			t.Fatalf("direct prompt direct_id = %q, want %q", got, want)
		}
		if got, want := directCall.meta.WorkID, "work_direct_runtime"; got != want {
			t.Fatalf("direct prompt work_id = %q, want %q", got, want)
		}
		prompter.finishCall(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow.Add(time.Second)})

		currentUnix.Store(fixedNow.Add(2 * time.Second).Unix())
		handoffID, err := manager.Send(ctx, SendRequest{
			ID:          stringPtr("msg-direct-handoff"),
			SessionID:   "sess-local",
			Channel:     "builders",
			Surface:     surfacePtr(SurfaceDirect),
			DirectID:    stringPtr(directID),
			Kind:        KindSay,
			To:          stringPtr(remotePeerID),
			WorkID:      stringPtr("work_direct_handoff"),
			ReplyTo:     stringPtr("msg-thread-runtime"),
			TraceID:     stringPtr("trace-runtime-thread"),
			CausationID: stringPtr("msg-thread-runtime"),
			Body:        mustRawJSON(t, SayBody{Text: "taking this into a direct room"}),
		})
		if err != nil {
			t.Fatalf("Send(handoff) error = %v", err)
		}
		if got, want := handoffID, "msg-direct-handoff"; got != want {
			t.Fatalf("handoff message id = %q, want %q", got, want)
		}

		currentUnix.Store(fixedNow.Add(3 * time.Second).Unix())
		if _, err := manager.Send(ctx, SendRequest{
			ID:          stringPtr("msg-summary-back"),
			SessionID:   "sess-local",
			Channel:     "builders",
			Surface:     surfacePtr(SurfaceThread),
			ThreadID:    stringPtr(testThreadRef().ThreadID),
			Kind:        KindSay,
			ReplyTo:     stringPtr("msg-direct-runtime"),
			TraceID:     stringPtr("trace-runtime-thread"),
			CausationID: stringPtr("msg-direct-runtime"),
			Body:        mustRawJSON(t, SayBody{Text: "summary: direct review is underway"}),
		}); err != nil {
			t.Fatalf("Send(summary back) error = %v", err)
		}

		threadMessages, err := db.ListConversationMessages(ctx, store.NetworkConversationRef{
			Channel:  "builders",
			Surface:  store.NetworkSurfaceThread,
			ThreadID: testThreadRef().ThreadID,
		}, store.NetworkConversationMessageQuery{Limit: 10})
		if err != nil {
			t.Fatalf("ListConversationMessages(thread) error = %v", err)
		}
		assertMessageTexts(t, threadMessages, []string{"public thread request", "summary: direct review is underway"})
		if joined := conversationTexts(threadMessages); strings.Contains(joined, "restricted direct detail") {
			t.Fatalf("thread messages leaked direct text: %q", joined)
		}

		directMessages, err := db.ListConversationMessages(ctx, store.NetworkConversationRef{
			Channel:  "builders",
			Surface:  store.NetworkSurfaceDirect,
			DirectID: directID,
		}, store.NetworkConversationMessageQuery{Limit: 10})
		if err != nil {
			t.Fatalf("ListConversationMessages(direct) error = %v", err)
		}
		assertMessageTexts(t, directMessages, []string{"restricted direct detail", "taking this into a direct room"})
		handoff := findConversationMessage(t, directMessages, "msg-direct-handoff")
		if got, want := handoff.WorkID, "work_direct_handoff"; got != want {
			t.Fatalf("handoff WorkID = %q, want %q", got, want)
		}
		if got, want := handoff.ReplyTo, "msg-thread-runtime"; got != want {
			t.Fatalf("handoff ReplyTo = %q, want %q", got, want)
		}
		if got, want := handoff.TraceID, "trace-runtime-thread"; got != want {
			t.Fatalf("handoff TraceID = %q, want %q", got, want)
		}
		if got, want := handoff.CausationID, "msg-thread-runtime"; got != want {
			t.Fatalf("handoff CausationID = %q, want %q", got, want)
		}
		work, err := db.GetWork(ctx, "work_direct_handoff")
		if err != nil {
			t.Fatalf("GetWork(handoff) error = %v", err)
		}
		if got, want := work.DirectID, directID; got != want {
			t.Fatalf("handoff work DirectID = %q, want %q", got, want)
		}
	})
}

func deliverInboundEnvelope(t *testing.T, manager *Manager, envelope Envelope) {
	t.Helper()

	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", envelope.ID, err)
	}
	manager.handleInboundMessage(payload)
}

func openNetworkRuntimeDB(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := globaldb.OpenGlobalDB(ctx, filepath.Join(t.TempDir(), "agh.db"))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(context.Background()); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	})
	return db
}

func assertMessageTexts(t *testing.T, messages []store.NetworkConversationMessage, want []string) {
	t.Helper()

	got := make([]string, 0, len(messages))
	for _, message := range messages {
		got = append(got, message.Text)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("conversation texts = %#v, want %#v", got, want)
	}
}

func conversationTexts(messages []store.NetworkConversationMessage) string {
	values := make([]string, 0, len(messages))
	for _, message := range messages {
		values = append(values, message.Text)
	}
	return strings.Join(values, "\n")
}

func findConversationMessage(
	t *testing.T,
	messages []store.NetworkConversationMessage,
	messageID string,
) store.NetworkConversationMessage {
	t.Helper()

	for _, message := range messages {
		if message.MessageID == messageID {
			return message
		}
	}
	t.Fatalf("conversation message %q not found in %#v", messageID, messages)
	return store.NetworkConversationMessage{}
}
