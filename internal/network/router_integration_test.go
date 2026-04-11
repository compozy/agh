//go:build integration

package network

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRoutersDiscoverEachOtherAndExchangeDirectAndBroadcastMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	transport, err := NewTransport(ctx, testNetworkConfig())
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	t.Cleanup(func() {
		_ = transport.Shutdown(context.Background())
	})

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registryA, err := NewPeerRegistry(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPeerRegistry(B) error = %v", err)
	}

	peerA := mustPeerCard(t, "coder.sess-a")
	peerB := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registryA.RegisterLocal("sess-a", "builders", peerA, now); err != nil {
		t.Fatalf("RegisterLocal(A) error = %v", err)
	}
	if _, err := registryB.RegisterLocal("sess-b", "builders", peerB, now); err != nil {
		t.Fatalf("RegisterLocal(B) error = %v", err)
	}

	routerA, err := NewRouter(registryA, transport, DefaultMaxReplayAge)
	if err != nil {
		t.Fatalf("NewRouter(A) error = %v", err)
	}
	routerB, err := NewRouter(registryB, transport, DefaultMaxReplayAge)
	if err != nil {
		t.Fatalf("NewRouter(B) error = %v", err)
	}

	resultsA := make(chan RouteResult, 16)
	resultsB := make(chan RouteResult, 16)
	errCh := make(chan error, 4)

	subscriptions := subscribeRouter(t, transport, routerA, peerA.PeerID, "builders", resultsA, errCh)
	subscriptions = append(subscriptions, subscribeRouter(t, transport, routerB, peerB.PeerID, "builders", resultsB, errCh)...)
	for _, subscription := range subscriptions {
		subscription := subscription
		t.Cleanup(func() {
			_ = subscription.Unsubscribe()
		})
	}

	if _, err := routerA.PublishGreet(ctx, "sess-a", "ready"); err != nil {
		t.Fatalf("routerA.PublishGreet() error = %v", err)
	}
	if _, err := routerB.PublishGreet(ctx, "sess-b", "ready"); err != nil {
		t.Fatalf("routerB.PublishGreet() error = %v", err)
	}

	waitForRouterCondition(t, ctx, func() bool {
		return registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC()) &&
			registryB.HasPresence("builders", peerA.PeerID, time.Now().UTC())
	}, "peer discovery")

	if _, err := routerA.Send(ctx, SendRequest{
		SessionID:     "sess-a",
		Space:         "builders",
		Kind:          KindDirect,
		To:            stringPtr(peerB.PeerID),
		InteractionID: stringPtr("int_direct_integration"),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	}); err != nil {
		t.Fatalf("routerA.Send(direct) error = %v", err)
	}

	directResult := waitForDelivery(t, ctx, resultsB, "sess-b", KindDirect)
	if got, want := directResult.Deliveries[0].Envelope.From, peerA.PeerID; got != want {
		t.Fatalf("direct From = %q, want %q", got, want)
	}

	if _, err := routerB.Send(ctx, SendRequest{
		SessionID: "sess-b",
		Space:     "builders",
		Kind:      KindSay,
		Body:      mustRawJSON(t, SayBody{Text: "broadcast update"}),
	}); err != nil {
		t.Fatalf("routerB.Send(say) error = %v", err)
	}

	waitForDelivery(t, ctx, resultsA, "sess-a", KindSay)
	waitForDelivery(t, ctx, resultsB, "sess-b", KindSay)

	select {
	case receiveErr := <-errCh:
		t.Fatalf("router subscription error = %v", receiveErr)
	default:
	}
}

func TestHeartbeatExpiryAndFreshGreetRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	cfg := testNetworkConfig()
	cfg.GreetInterval = 1

	transport, err := NewTransport(ctx, cfg)
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	t.Cleanup(func() {
		_ = transport.Shutdown(context.Background())
	})

	registryA, err := NewPeerRegistry(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPeerRegistry(B) error = %v", err)
	}

	peerA := mustPeerCard(t, "coder.sess-a")
	peerB := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registryA.RegisterLocal("sess-a", "builders", peerA, time.Now().UTC()); err != nil {
		t.Fatalf("RegisterLocal(A) error = %v", err)
	}
	if _, err := registryB.RegisterLocal("sess-b", "builders", peerB, time.Now().UTC()); err != nil {
		t.Fatalf("RegisterLocal(B) error = %v", err)
	}

	routerA, err := NewRouter(registryA, transport, DefaultMaxReplayAge)
	if err != nil {
		t.Fatalf("NewRouter(A) error = %v", err)
	}
	routerB, err := NewRouter(registryB, transport, DefaultMaxReplayAge)
	if err != nil {
		t.Fatalf("NewRouter(B) error = %v", err)
	}

	errCh := make(chan error, 4)
	subs := subscribeRouter(t, transport, routerA, peerA.PeerID, "builders", nil, errCh)
	subs = append(subs, subscribeRouter(t, transport, routerB, peerB.PeerID, "builders", nil, errCh)...)
	for _, subscription := range subs {
		subscription := subscription
		t.Cleanup(func() {
			_ = subscription.Unsubscribe()
		})
	}

	heartbeat, err := routerB.StartHeartbeat(ctx, "sess-b", "alive")
	if err != nil {
		t.Fatalf("routerB.StartHeartbeat() error = %v", err)
	}
	waitForRouterCondition(t, ctx, func() bool {
		return registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC())
	}, "initial heartbeat discoverability")

	heartbeat.Stop()

	waitForRouterCondition(t, ctx, func() bool {
		return !registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC())
	}, "heartbeat expiry")

	if _, err := routerB.PublishGreet(ctx, "sess-b", "back"); err != nil {
		t.Fatalf("routerB.PublishGreet(recover) error = %v", err)
	}
	waitForRouterCondition(t, ctx, func() bool {
		return registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC())
	}, "fresh greet recovery")

	select {
	case receiveErr := <-errCh:
		t.Fatalf("router subscription error = %v", receiveErr)
	default:
	}
}

func subscribeRouter(
	t *testing.T,
	transport *Transport,
	router *Router,
	peerID string,
	space string,
	results chan<- RouteResult,
	errCh chan<- error,
) []*nats.Subscription {
	t.Helper()

	broadcastSubject, err := BroadcastSubject(space)
	if err != nil {
		t.Fatalf("BroadcastSubject(%q) error = %v", space, err)
	}
	directSubject, err := DirectSubject(space, peerID)
	if err != nil {
		t.Fatalf("DirectSubject(%q, %q) error = %v", space, peerID, err)
	}

	subjects := []string{broadcastSubject, directSubject}
	subscriptions := make([]*nats.Subscription, 0, len(subjects))
	for _, subject := range subjects {
		subject := subject
		subscription, subErr := transport.Subscribe(subject, func(msg *nats.Msg) {
			result, receiveErr := router.Receive(context.Background(), append([]byte(nil), msg.Data...))
			if receiveErr != nil {
				errCh <- receiveErr
				return
			}
			if results != nil {
				select {
				case results <- result:
				default:
				}
			}
		})
		if subErr != nil {
			t.Fatalf("Subscribe(%q) error = %v", subject, subErr)
		}
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions
}

func waitForRouterCondition(t *testing.T, ctx context.Context, condition func() bool, description string) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", description, ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForDelivery(t *testing.T, ctx context.Context, results <-chan RouteResult, sessionID string, kind Kind) RouteResult {
	t.Helper()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s delivery to %s: %v", kind, sessionID, ctx.Err())
		case result := <-results:
			for _, delivery := range result.Deliveries {
				if delivery.SessionID == sessionID && delivery.Envelope.Kind == kind {
					return result
				}
			}
		}
	}
}
