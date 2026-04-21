//go:build integration

package network

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	sessionpkg "github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRoutersDiscoverEachOtherAndExchangeDirectAndBroadcastMessages(t *testing.T) {
	t.Parallel()

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
	registryA, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(time.Second)
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
		Channel:       "builders",
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
		Channel:   "builders",
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

func TestRoutersExchangeBroadcastCapabilityTransfers(t *testing.T) {
	t.Parallel()

	t.Run("Should exchange broadcast capability transfers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
		defer cancel()

		transport, err := NewTransport(ctx, testNetworkConfig())
		if err != nil {
			t.Fatalf("NewTransport() error = %v", err)
		}
		t.Cleanup(func() {
			_ = transport.Shutdown(context.Background())
		})

		now := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
		registryA, err := NewPeerRegistry(time.Second)
		if err != nil {
			t.Fatalf("NewPeerRegistry(A) error = %v", err)
		}
		registryB, err := NewPeerRegistry(time.Second)
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
			SessionID: "sess-a",
			Channel:   "builders",
			Kind:      KindCapability,
			Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
				ID:               "review-fix",
				Summary:          "Review fix flow",
				Outcome:          "A reusable review fix workflow.",
				Version:          "1.0.0",
				ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
				Requirements:     []string{"workspace-write"},
			}),
		}); err != nil {
			t.Fatalf("routerA.Send(capability broadcast) error = %v", err)
		}

		resultA := waitForDelivery(t, ctx, resultsA, "sess-a", KindCapability)
		resultB := waitForDelivery(t, ctx, resultsB, "sess-b", KindCapability)
		for _, result := range []RouteResult{resultA, resultB} {
			if got, want := len(result.Deliveries), 1; got != want {
				t.Fatalf("len(capability broadcast deliveries) = %d, want %d", got, want)
			}
			decoded, err := result.Deliveries[0].Envelope.DecodeBody()
			if err != nil {
				t.Fatalf("DecodeBody(capability broadcast) error = %v", err)
			}
			body := decoded.(CapabilityBody)
			if got, want := body.Capability.ID, "review-fix"; got != want {
				t.Fatalf("capability broadcast id = %q, want %q", got, want)
			}
		}

		select {
		case receiveErr := <-errCh:
			t.Fatalf("router subscription error = %v", receiveErr)
		default:
		}
	})
}

func TestRoutersPreserveCapabilityLifecycleAcrossPeers(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve capability lifecycle across peers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
		defer cancel()

		transport, err := NewTransport(ctx, testNetworkConfig())
		if err != nil {
			t.Fatalf("NewTransport() error = %v", err)
		}
		t.Cleanup(func() {
			_ = transport.Shutdown(context.Background())
		})

		now := time.Date(2026, 4, 20, 12, 30, 0, 0, time.UTC)
		registryA, err := NewPeerRegistry(time.Second)
		if err != nil {
			t.Fatalf("NewPeerRegistry(A) error = %v", err)
		}
		registryB, err := NewPeerRegistry(time.Second)
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

		const interactionID = "int_capability_lifecycle"

		if _, err := routerA.Send(ctx, SendRequest{
			SessionID:     "sess-a",
			Channel:       "builders",
			Kind:          KindCapability,
			To:            stringPtr(peerB.PeerID),
			InteractionID: stringPtr(interactionID),
			Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
				ID:               "review-fix",
				Summary:          "Review fix flow",
				Outcome:          "A reusable review fix workflow.",
				Version:          "1.0.0",
				ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
				Requirements:     []string{"workspace-write"},
			}),
		}); err != nil {
			t.Fatalf("routerA.Send(capability directed) error = %v", err)
		}

		capabilityResult := waitForDelivery(t, ctx, resultsB, "sess-b", KindCapability)
		if got, want := len(capabilityResult.Deliveries), 1; got != want {
			t.Fatalf("len(capability directed deliveries) = %d, want %d", got, want)
		}
		capabilityEnvelope := capabilityResult.Deliveries[0].Envelope

		if _, err := routerB.Send(ctx, SendRequest{
			SessionID:     "sess-b",
			Channel:       "builders",
			Kind:          KindTrace,
			To:            stringPtr(peerA.PeerID),
			InteractionID: stringPtr(interactionID),
			ReplyTo:       stringPtr(capabilityEnvelope.ID),
			Body: mustRawJSON(t, TraceBody{
				State:   StateNeedsInput,
				Message: "need more detail",
			}),
		}); err != nil {
			t.Fatalf("routerB.Send(trace needs_input) error = %v", err)
		}

		traceNeedsInput := waitForDelivery(t, ctx, resultsA, "sess-a", KindTrace)
		traceNeedsInputBody, err := traceNeedsInput.Deliveries[0].Envelope.DecodeBody()
		if err != nil {
			t.Fatalf("DecodeBody(trace needs_input) error = %v", err)
		}
		if got, want := traceNeedsInputBody.(TraceBody).State, StateNeedsInput; got != want {
			t.Fatalf("trace needs_input state = %q, want %q", got, want)
		}

		if _, err := routerA.Send(ctx, SendRequest{
			SessionID:     "sess-a",
			Channel:       "builders",
			Kind:          KindDirect,
			To:            stringPtr(peerB.PeerID),
			InteractionID: stringPtr(interactionID),
			ReplyTo:       stringPtr(traceNeedsInput.Deliveries[0].Envelope.ID),
			Body:          mustRawJSON(t, DirectBody{Text: "here is the missing detail", Intent: "reply"}),
		}); err != nil {
			t.Fatalf("routerA.Send(direct follow-up) error = %v", err)
		}

		directResult := waitForDelivery(t, ctx, resultsB, "sess-b", KindDirect)
		directBody, err := directResult.Deliveries[0].Envelope.DecodeBody()
		if err != nil {
			t.Fatalf("DecodeBody(direct follow-up) error = %v", err)
		}
		if got, want := directBody.(DirectBody).Text, "here is the missing detail"; got != want {
			t.Fatalf("direct follow-up text = %q, want %q", got, want)
		}

		if _, err := routerB.Send(ctx, SendRequest{
			SessionID:     "sess-b",
			Channel:       "builders",
			Kind:          KindTrace,
			To:            stringPtr(peerA.PeerID),
			InteractionID: stringPtr(interactionID),
			ReplyTo:       stringPtr(directResult.Deliveries[0].Envelope.ID),
			Body: mustRawJSON(t, TraceBody{
				State:   StateCompleted,
				Message: "completed",
			}),
		}); err != nil {
			t.Fatalf("routerB.Send(trace completed) error = %v", err)
		}

		traceCompleted := waitForDelivery(t, ctx, resultsA, "sess-a", KindTrace)
		traceCompletedBody, err := traceCompleted.Deliveries[0].Envelope.DecodeBody()
		if err != nil {
			t.Fatalf("DecodeBody(trace completed) error = %v", err)
		}
		if got, want := traceCompletedBody.(TraceBody).State, StateCompleted; got != want {
			t.Fatalf("trace completed state = %q, want %q", got, want)
		}

		if _, err := routerA.Send(ctx, SendRequest{
			SessionID:     "sess-a",
			Channel:       "builders",
			Kind:          KindCapability,
			To:            stringPtr(peerB.PeerID),
			InteractionID: stringPtr(interactionID),
			ReplyTo:       stringPtr(directResult.Deliveries[0].Envelope.ID),
			Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
				ID:               "review-fix-follow-up",
				Summary:          "Review follow-up flow",
				Outcome:          "A post-completion follow-up workflow.",
				Version:          "1.0.0",
				ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
				Requirements:     []string{"workspace-write"},
			}),
		}); !errors.Is(err, ErrInteractionClosed) {
			t.Fatalf("routerA.Send(post-terminal capability) error = %v, want ErrInteractionClosed", err)
		}

		select {
		case receiveErr := <-errCh:
			t.Fatalf("router subscription error = %v", receiveErr)
		default:
		}
	})
}

func TestHeartbeatExpiryAndFreshGreetRecovery(t *testing.T) {
	t.Parallel()

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

	registryA, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(time.Second)
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

func TestDirectedWhoisRichDiscoveryDeliversPeerCardAndCapabilityCatalog(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	transport, err := NewTransport(ctx, testNetworkConfig())
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	t.Cleanup(func() {
		_ = transport.Shutdown(context.Background())
	})

	now := time.Date(2026, 4, 19, 6, 50, 0, 0, time.UTC)
	registryA, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(B) error = %v", err)
	}

	peerA := mustPeerCard(t, "coder.sess-a")
	peerB := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registryA.RegisterLocal("sess-a", "builders", peerA, now); err != nil {
		t.Fatalf("RegisterLocal(A) error = %v", err)
	}
	catalog := []sessionpkg.NetworkPeerCapability{
		{
			ID:                "review-pr",
			Summary:           "Review pull requests",
			Outcome:           "Actionable review findings",
			Version:           "1.0.0",
			Digest:            "sha256:review-pr-v1",
			ContextNeeded:     []string{"pull request diff"},
			ArtifactsExpected: []string{"review summary"},
			Requirements:      []string{"workspace-read"},
		},
		{
			ID:           "draft-spec",
			Summary:      "Draft technical specifications",
			Outcome:      "A reviewed implementation plan",
			Version:      "2.1.0",
			Digest:       "sha256:draft-spec-v2",
			Requirements: []string{"repo-map"},
		},
	}
	if _, err := registryB.RegisterLocalWithCapabilityCatalog("sess-b", "builders", peerB, catalog, now); err != nil {
		t.Fatalf("RegisterLocalWithCapabilityCatalog(B) error = %v", err)
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
	errCh := make(chan error, 4)
	subscriptions := subscribeRouter(t, transport, routerA, peerA.PeerID, "builders", resultsA, errCh)
	subscriptions = append(subscriptions, subscribeRouter(t, transport, routerB, peerB.PeerID, "builders", nil, errCh)...)
	for _, subscription := range subscriptions {
		subscription := subscription
		t.Cleanup(func() {
			_ = subscription.Unsubscribe()
		})
	}

	if _, err := routerB.PublishGreet(ctx, "sess-b", "ready"); err != nil {
		t.Fatalf("routerB.PublishGreet() error = %v", err)
	}
	waitForRouterCondition(t, ctx, func() bool {
		return registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC())
	}, "peerB discoverability before directed rich whois")

	if _, err := routerA.Send(ctx, SendRequest{
		SessionID: "sess-a",
		Channel:   "builders",
		Kind:      KindWhois,
		To:        stringPtr(peerB.PeerID),
		Body:      mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
		Ext: ExtensionMap{
			whoisIncludeExtKey: mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
		},
	}); err != nil {
		t.Fatalf("routerA.Send(whois rich) error = %v", err)
	}

	result := waitForDelivery(t, ctx, resultsA, "sess-a", KindWhois)
	delivery := result.Deliveries[0]
	decoded, err := delivery.Envelope.DecodeBody()
	if err != nil {
		t.Fatalf("DecodeBody(rich whois response) error = %v", err)
	}
	body := decoded.(WhoisBody)
	if body.PeerCard == nil || body.PeerCard.PeerID != peerB.PeerID {
		t.Fatalf("response peer_card = %#v, want peer %q", body.PeerCard, peerB.PeerID)
	}

	payload := decodeWhoisCapabilityCatalogPayload(t, delivery.Envelope.Ext[whoisCapabilityCatalogExtKey])
	wantPayload := whoisCapabilityCatalogPayload{
		Capabilities: []whoisCapabilityCatalogEntry{
			{
				ID:                "review-pr",
				Summary:           "Review pull requests",
				Outcome:           "Actionable review findings",
				Version:           "1.0.0",
				Digest:            "sha256:review-pr-v1",
				ContextNeeded:     []string{"pull request diff"},
				ArtifactsExpected: []string{"review summary"},
				Requirements:      []string{"workspace-read"},
			},
			{
				ID:           "draft-spec",
				Summary:      "Draft technical specifications",
				Outcome:      "A reviewed implementation plan",
				Version:      "2.1.0",
				Digest:       "sha256:draft-spec-v2",
				Requirements: []string{"repo-map"},
			},
		},
	}
	if !reflect.DeepEqual(payload, wantPayload) {
		t.Fatalf("rich capability catalog = %#v, want %#v", payload, wantPayload)
	}
	if remote, ok := registryA.RemoteByPeer("builders", peerB.PeerID, time.Now().UTC()); !ok {
		t.Fatalf("RemoteByPeer(%q) missing after rich whois response", peerB.PeerID)
	} else if !remote.CapabilityCatalogKnown || !reflect.DeepEqual(remote.CapabilityCatalog, catalog) {
		t.Fatalf("remote capability catalog = %#v known=%v, want %#v known=true", remote.CapabilityCatalog, remote.CapabilityCatalogKnown, catalog)
	}

	select {
	case receiveErr := <-errCh:
		t.Fatalf("router subscription error = %v", receiveErr)
	default:
	}
}

func TestDirectedWhoisRichDiscoveryFilteringRefreshesRemotePresence(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(testutil.Context(t), 10*time.Second)
	defer cancel()

	transport, err := NewTransport(ctx, testNetworkConfig())
	if err != nil {
		t.Fatalf("NewTransport() error = %v", err)
	}
	t.Cleanup(func() {
		_ = transport.Shutdown(context.Background())
	})

	registryA, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(A) error = %v", err)
	}
	registryB, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry(B) error = %v", err)
	}

	peerA := mustPeerCard(t, "coder.sess-a")
	peerB := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registryA.RegisterLocal("sess-a", "builders", peerA, time.Now().UTC()); err != nil {
		t.Fatalf("RegisterLocal(A) error = %v", err)
	}
	if _, err := registryB.RegisterLocalWithCapabilityCatalog(
		"sess-b",
		"builders",
		peerB,
		[]sessionpkg.NetworkPeerCapability{
			{
				ID:           "review-pr",
				Summary:      "Review pull requests",
				Outcome:      "Actionable review findings",
				Version:      "1.0.0",
				Digest:       "sha256:review-pr-v1",
				Requirements: []string{"workspace-read"},
			},
			{
				ID:           "draft-spec",
				Summary:      "Draft technical specifications",
				Outcome:      "A reviewed implementation plan",
				Version:      "2.1.0",
				Digest:       "sha256:draft-spec-v2",
				Requirements: []string{"repo-map"},
			},
		},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("RegisterLocalWithCapabilityCatalog(B) error = %v", err)
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
	errCh := make(chan error, 4)
	subscriptions := subscribeRouter(t, transport, routerA, peerA.PeerID, "builders", resultsA, errCh)
	subscriptions = append(subscriptions, subscribeRouter(t, transport, routerB, peerB.PeerID, "builders", nil, errCh)...)
	for _, subscription := range subscriptions {
		subscription := subscription
		t.Cleanup(func() {
			_ = subscription.Unsubscribe()
		})
	}

	if _, err := routerB.PublishGreet(ctx, "sess-b", "ready"); err != nil {
		t.Fatalf("routerB.PublishGreet() error = %v", err)
	}
	waitForRouterCondition(t, ctx, func() bool {
		return registryA.HasPresence("builders", peerB.PeerID, time.Now().UTC())
	}, "peerB discoverability before filtered rich whois")

	before, ok := registryA.RemoteByPeer("builders", peerB.PeerID, time.Now().UTC())
	if !ok {
		t.Fatalf("RemoteByPeer(%q) missing after greet", peerB.PeerID)
	}

	if _, err := routerA.Send(ctx, SendRequest{
		SessionID: "sess-a",
		Channel:   "builders",
		Kind:      KindWhois,
		To:        stringPtr(peerB.PeerID),
		Body:      mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
		Ext: ExtensionMap{
			whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
			whoisCapabilityIDsExtKey: mustRawJSON(t, []string{"draft-spec"}),
		},
	}); err != nil {
		t.Fatalf("routerA.Send(filtered whois rich) error = %v", err)
	}

	result := waitForDelivery(t, ctx, resultsA, "sess-a", KindWhois)
	delivery := result.Deliveries[0]
	payload := decodeWhoisCapabilityCatalogPayload(t, delivery.Envelope.Ext[whoisCapabilityCatalogExtKey])
	wantPayload := whoisCapabilityCatalogPayload{
		Capabilities: []whoisCapabilityCatalogEntry{{
			ID:           "draft-spec",
			Summary:      "Draft technical specifications",
			Outcome:      "A reviewed implementation plan",
			Version:      "2.1.0",
			Digest:       "sha256:draft-spec-v2",
			Requirements: []string{"repo-map"},
		}},
	}
	if !reflect.DeepEqual(payload, wantPayload) {
		t.Fatalf("filtered capability catalog = %#v, want %#v", payload, wantPayload)
	}

	waitForRouterCondition(t, ctx, func() bool {
		entry, ok := registryA.RemoteByPeer("builders", peerB.PeerID, time.Now().UTC())
		return ok && entry.LastSeen.After(before.LastSeen)
	}, "remote presence refresh from rich whois response")
	if remote, ok := registryA.RemoteByPeer("builders", peerB.PeerID, time.Now().UTC()); !ok {
		t.Fatalf("RemoteByPeer(%q) missing after filtered rich whois response", peerB.PeerID)
	} else if !remote.CapabilityCatalogKnown {
		t.Fatalf("remote.CapabilityCatalogKnown = false, want true")
	} else if got, want := remote.CapabilityCatalog, []sessionpkg.NetworkPeerCapability{{
		ID:           "draft-spec",
		Summary:      "Draft technical specifications",
		Outcome:      "A reviewed implementation plan",
		Version:      "2.1.0",
		Digest:       "sha256:draft-spec-v2",
		Requirements: []string{"repo-map"},
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("remote capability catalog = %#v, want %#v", got, want)
	}

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
	channel string,
	results chan<- RouteResult,
	errCh chan<- error,
) []*nats.Subscription {
	t.Helper()

	broadcastSubject, err := BroadcastSubject(channel)
	if err != nil {
		t.Fatalf("BroadcastSubject(%q) error = %v", channel, err)
	}
	directSubject, err := DirectSubject(channel, peerID)
	if err != nil {
		t.Fatalf("DirectSubject(%q, %q) error = %v", channel, peerID, err)
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
