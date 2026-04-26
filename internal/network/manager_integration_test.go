//go:build integration

package network

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	sessionpkg "github.com/pedronauck/agh/internal/session"
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

	subject, err := BroadcastSubject("builders")
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
		_ = subscription.Unsubscribe()
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
			if got := decodeCapabilityBriefPayload(t, body.PeerCard.Ext[capabilityBriefExtKey]); !slices.Equal(got, []capabilityBrief{
				{ID: "review-pr", Summary: "Review pull requests"},
				{ID: "draft-spec", Summary: "Draft technical specs"},
			}) {
				t.Fatalf("%s greet capability brief = %#v, want projected brief entries", label, got)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s greet: %v", label, ctx.Err())
		}
	}

	assertPublishedBriefGreet("initial")

	peers, err := manager.ListPeers(ctx, "builders")
	if err != nil {
		t.Fatalf("ListPeers() error = %v", err)
	}
	if got, want := len(peers), 1; got != want {
		t.Fatalf("len(ListPeers()) = %d, want %d", got, want)
	}
	if got := decodeCapabilityBriefPayload(t, peers[0].PeerCard.Ext[capabilityBriefExtKey]); !slices.Equal(got, []capabilityBrief{
		{ID: "review-pr", Summary: "Review pull requests"},
		{ID: "draft-spec", Summary: "Draft technical specs"},
	}) {
		t.Fatalf("listed peer capability brief = %#v, want projected brief entries", got)
	}

	manager.handleReconnect()
	assertPublishedBriefGreet("reconnect")
}
