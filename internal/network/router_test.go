package network

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	sessionpkg "github.com/pedronauck/agh/internal/session"
)

func TestRouterSendEnforcesPresencePreflight(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	sender := mustPeerCard(t, "coder.sess-a")
	if _, err := registry.RegisterLocal("sess-a", "builders", sender, now); err != nil {
		t.Fatalf("RegisterLocal(sender) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	req := SendRequest{
		SessionID:     "sess-a",
		Channel:       "builders",
		Kind:          KindDirect,
		To:            stringPtr("reviewer.sess-missing"),
		InteractionID: stringPtr("int_missing"),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	}
	if _, err := router.Send(context.Background(), req); !errors.Is(err, ErrTargetPeerNotFound) {
		t.Fatalf("Send(absent target) error = %v, want ErrTargetPeerNotFound", err)
	}
	if got := transport.Count(); got != 0 {
		t.Fatalf("transport publishes after absent preflight = %d, want 0", got)
	}

	expiringPeer := mustPeerCard(t, "reviewer.sess-expired")
	if _, stored, err := registry.RefreshRemote("builders", expiringPeer, now); err != nil {
		t.Fatalf("RefreshRemote(expiring) error = %v", err)
	} else if !stored {
		t.Fatal("RefreshRemote(expiring) stored = false, want true")
	}

	later := now.Add(21 * time.Second)
	expiredRouter, err := NewRouter(
		registry,
		transport,
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return later }),
	)
	if err != nil {
		t.Fatalf("NewRouter(expired) error = %v", err)
	}
	req.To = stringPtr(expiringPeer.PeerID)
	req.InteractionID = stringPtr("int_expired")
	if _, err := expiredRouter.Send(context.Background(), req); !errors.Is(err, ErrTargetPeerNotFound) {
		t.Fatalf("Send(expired target) error = %v, want ErrTargetPeerNotFound", err)
	}
	if got := transport.Count(); got != 0 {
		t.Fatalf("transport publishes after expired preflight = %d, want 0", got)
	}
}

func TestRouterRoutesBroadcastAndDirectToCorrectSubjectsAndTargets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	sender := mustPeerCard(t, "coder.sess-a")
	target := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-a", "builders", sender, now); err != nil {
		t.Fatalf("RegisterLocal(sender) error = %v", err)
	}
	if _, err := registry.RegisterLocal("sess-b", "builders", target, now); err != nil {
		t.Fatalf("RegisterLocal(target) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	sayResult, err := router.Send(context.Background(), SendRequest{
		SessionID: "sess-a",
		Channel:   "builders",
		Kind:      KindSay,
		Body:      mustRawJSON(t, SayBody{Text: "status update"}),
	})
	if err != nil {
		t.Fatalf("Send(say) error = %v", err)
	}
	if got, want := sayResult.Subject, "agh.network.v0.builders.broadcast"; got != want {
		t.Fatalf("Send(say).Subject = %q, want %q", got, want)
	}

	directResult, err := router.Send(context.Background(), SendRequest{
		SessionID:     "sess-a",
		Channel:       "builders",
		Kind:          KindDirect,
		To:            stringPtr(target.PeerID),
		InteractionID: stringPtr("int_route"),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	})
	if err != nil {
		t.Fatalf("Send(direct) error = %v", err)
	}
	wantDirectSubject, err := DirectSubject("builders", target.PeerID)
	if err != nil {
		t.Fatalf("DirectSubject() error = %v", err)
	}
	if got := directResult.Subject; got != wantDirectSubject {
		t.Fatalf("Send(direct).Subject = %q, want %q", got, wantDirectSubject)
	}

	if got, want := transport.Count(), 2; got != want {
		t.Fatalf("transport publish count = %d, want %d", got, want)
	}

	broadcastResult, err := router.Receive(context.Background(), transport.Message(0).payload)
	if err != nil {
		t.Fatalf("Receive(broadcast) error = %v", err)
	}
	if got, want := len(broadcastResult.Deliveries), 2; got != want {
		t.Fatalf("len(broadcast deliveries) = %d, want %d", got, want)
	}

	directInbound, err := router.Receive(context.Background(), transport.Message(1).payload)
	if err != nil {
		t.Fatalf("Receive(direct) error = %v", err)
	}
	if got, want := len(directInbound.Deliveries), 1; got != want {
		t.Fatalf("len(direct deliveries) = %d, want %d", got, want)
	}
	if got, want := directInbound.Deliveries[0].SessionID, "sess-b"; got != want {
		t.Fatalf("direct delivery session = %q, want %q", got, want)
	}
}

func TestRouterRejectsDuplicateBeforeReprocessingLifecycleState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	target := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", target, now); err != nil {
		t.Fatalf("RegisterLocal(target) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	directPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_direct_dup",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(target.PeerID),
		InteractionID: stringPtr("int_dup"),
		TS:            now.Unix(),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(direct) error = %v", err)
	}
	first, err := router.Receive(context.Background(), directPayload)
	if err != nil {
		t.Fatalf("Receive(first direct) error = %v", err)
	}
	if got, want := len(first.Deliveries), 1; got != want {
		t.Fatalf("len(first direct deliveries) = %d, want %d", got, want)
	}

	receiptPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_receipt_cancel",
		Kind:          KindReceipt,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(target.PeerID),
		InteractionID: stringPtr("int_dup"),
		ReplyTo:       stringPtr("msg_direct_dup"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, ReceiptBody{
			ForID:  "msg_direct_dup",
			Status: ReceiptStatusCanceled,
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(receipt) error = %v", err)
	}
	if _, err := router.Receive(context.Background(), receiptPayload); err != nil {
		t.Fatalf("Receive(cancel receipt) error = %v", err)
	}

	duplicate, err := router.Receive(context.Background(), directPayload)
	if err != nil {
		t.Fatalf("Receive(duplicate direct) error = %v", err)
	}
	if !duplicate.Duplicate || !duplicate.Rejected {
		t.Fatalf("duplicate result = %#v, want duplicate rejection", duplicate)
	}
	if got, want := len(duplicate.Deliveries), 0; got != want {
		t.Fatalf("len(duplicate deliveries) = %d, want %d", got, want)
	}
	if duplicate.ReasonCode == nil || *duplicate.ReasonCode != ReasonCodeDuplicate {
		t.Fatalf("duplicate reason = %v, want %q", duplicate.ReasonCode, ReasonCodeDuplicate)
	}
	if got, want := len(duplicate.Generated), 1; got != want {
		t.Fatalf("len(duplicate generated) = %d, want %d", got, want)
	}
	receiptBody, decodeErr := duplicate.Generated[0].DecodeBody()
	if decodeErr != nil {
		t.Fatalf("DecodeBody(duplicate receipt) error = %v", decodeErr)
	}
	receipt := receiptBody.(ReceiptBody)
	if got, want := receipt.Status, ReceiptStatusDuplicate; got != want {
		t.Fatalf("duplicate receipt status = %q, want %q", got, want)
	}
	if receipt.ReasonCode == nil || *receipt.ReasonCode != ReasonCodeDuplicate {
		t.Fatalf("duplicate receipt reason = %v, want %q", receipt.ReasonCode, ReasonCodeDuplicate)
	}
}

func TestRouterWhoisRequestGeneratesResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	responder := mustPeerCard(t, "reviewer.sess-b")
	responder.Capabilities = []string{"chat.review"}
	if _, err := registry.RegisterLocal("sess-b", "builders", responder, now); err != nil {
		t.Fatalf("RegisterLocal(responder) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	requestPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_request",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     "coder.sess-a",
		TS:       now.Unix(),
		Body: mustRawJSON(t, WhoisBody{
			Type:  WhoisTypeRequest,
			Query: "chat.review",
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(whois request) error = %v", err)
	}

	result, err := router.Receive(context.Background(), requestPayload)
	if err != nil {
		t.Fatalf("Receive(whois request) error = %v", err)
	}
	if got, want := len(result.Generated), 1; got != want {
		t.Fatalf("len(whois responses) = %d, want %d", got, want)
	}
	if got, want := transport.Count(), 1; got != want {
		t.Fatalf("transport publish count = %d, want %d", got, want)
	}
	response := result.Generated[0]
	if got, want := response.Kind, KindWhois; got != want {
		t.Fatalf("response.Kind = %q, want %q", got, want)
	}
	if response.To == nil || *response.To != "coder.sess-a" {
		t.Fatalf("response.To = %v, want %q", response.To, "coder.sess-a")
	}
	if response.ReplyTo == nil || *response.ReplyTo != "msg_whois_request" {
		t.Fatalf("response.ReplyTo = %v, want %q", response.ReplyTo, "msg_whois_request")
	}
	decoded, decodeErr := response.DecodeBody()
	if decodeErr != nil {
		t.Fatalf("DecodeBody(response) error = %v", decodeErr)
	}
	body := decoded.(WhoisBody)
	if body.PeerCard == nil || body.PeerCard.PeerID != responder.PeerID {
		t.Fatalf("response peer_card = %#v, want peer %q", body.PeerCard, responder.PeerID)
	}
	if len(response.Ext) != 0 {
		t.Fatalf("response.Ext = %#v, want lean whois response with no rich ext", response.Ext)
	}
}

func TestRouterWhoisRichCapabilityDiscoveryReturnsCapabilityCatalog(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 19, 6, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	responder := mustPeerCard(t, "reviewer.sess-rich")
	catalog := []sessionpkg.NetworkPeerCapability{
		{
			ID:                "review-pr",
			Summary:           "Review pull requests",
			Outcome:           "Actionable review findings with risk assessment",
			ContextNeeded:     []string{"pull request link", "acceptance criteria"},
			ArtifactsExpected: []string{"review summary"},
			ExecutionOutline:  []string{"inspect diff", "run focused checks"},
			Constraints:       []string{"no speculative blockers"},
			Examples:          []string{"backend regression review"},
		},
		{
			ID:      "draft-spec",
			Summary: "Draft technical specifications",
			Outcome: "A reviewed implementation plan",
		},
	}
	if _, err := registry.RegisterLocalWithCapabilityCatalog(
		"sess-rich",
		"builders",
		responder,
		catalog,
		now,
	); err != nil {
		t.Fatalf("RegisterLocalWithCapabilityCatalog() error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	requestPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_rich_request",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     "coder.sess-a",
		To:       stringPtr(responder.PeerID),
		TS:       now.Unix(),
		Body: mustRawJSON(t, WhoisBody{
			Type: WhoisTypeRequest,
		}),
		Ext: ExtensionMap{
			whoisIncludeExtKey: mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(whois rich request) error = %v", err)
	}

	result, err := router.Receive(context.Background(), requestPayload)
	if err != nil {
		t.Fatalf("Receive(whois rich request) error = %v", err)
	}
	if got, want := len(result.Generated), 1; got != want {
		t.Fatalf("len(rich whois responses) = %d, want %d", got, want)
	}
	response := result.Generated[0]
	payload := decodeWhoisCapabilityCatalogPayload(t, response.Ext[whoisCapabilityCatalogExtKey])
	wantPayload := whoisCapabilityCatalogPayload{
		Capabilities: []whoisCapabilityCatalogEntry{
			{
				ID:                "review-pr",
				Summary:           "Review pull requests",
				Outcome:           "Actionable review findings with risk assessment",
				ContextNeeded:     []string{"pull request link", "acceptance criteria"},
				ArtifactsExpected: []string{"review summary"},
				ExecutionOutline:  []string{"inspect diff", "run focused checks"},
				Constraints:       []string{"no speculative blockers"},
				Examples:          []string{"backend regression review"},
			},
			{
				ID:      "draft-spec",
				Summary: "Draft technical specifications",
				Outcome: "A reviewed implementation plan",
			},
		},
	}
	if !reflect.DeepEqual(payload, wantPayload) {
		t.Fatalf("rich capability catalog = %#v, want %#v", payload, wantPayload)
	}
}

func TestRouterWhoisRichCapabilityDiscoveryFiltersRequestedIDsInCatalogOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 19, 6, 10, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	responder := mustPeerCard(t, "reviewer.sess-filtered")
	catalog := []sessionpkg.NetworkPeerCapability{
		{ID: "review-pr", Summary: "Review pull requests", Outcome: "Actionable feedback"},
		{ID: "draft-spec", Summary: "Draft technical specifications", Outcome: "A reviewed implementation plan"},
	}
	if _, err := registry.RegisterLocalWithCapabilityCatalog(
		"sess-filtered",
		"builders",
		responder,
		catalog,
		now,
	); err != nil {
		t.Fatalf("RegisterLocalWithCapabilityCatalog() error = %v", err)
	}

	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	requestPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_filtered_request",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     "coder.sess-a",
		To:       stringPtr(responder.PeerID),
		TS:       now.Unix(),
		Body:     mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
		Ext: ExtensionMap{
			whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
			whoisCapabilityIDsExtKey: mustRawJSON(t, []string{"draft-spec"}),
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(filtered whois request) error = %v", err)
	}

	result, err := router.Receive(context.Background(), requestPayload)
	if err != nil {
		t.Fatalf("Receive(filtered whois request) error = %v", err)
	}
	response := result.Generated[0]
	payload := decodeWhoisCapabilityCatalogPayload(t, response.Ext[whoisCapabilityCatalogExtKey])
	wantPayload := whoisCapabilityCatalogPayload{
		Capabilities: []whoisCapabilityCatalogEntry{{
			ID:      "draft-spec",
			Summary: "Draft technical specifications",
			Outcome: "A reviewed implementation plan",
		}},
	}
	if !reflect.DeepEqual(payload, wantPayload) {
		t.Fatalf("filtered capability catalog = %#v, want %#v", payload, wantPayload)
	}
}

func TestRouterWhoisRichCapabilityDiscoveryReturnsEmptyCatalogForUnknownIDsOrMissingCatalog(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 19, 6, 20, 0, 0, time.UTC)
	tests := []struct {
		name          string
		capabilityIDs []string
		registerFunc  func(t *testing.T, registry *PeerRegistry, now time.Time) string
	}{
		{
			name:          "unknown capability id",
			capabilityIDs: []string{"missing-capability"},
			registerFunc: func(t *testing.T, registry *PeerRegistry, now time.Time) string {
				t.Helper()

				responder := mustPeerCard(t, "reviewer.sess-unknown-id")
				if _, err := registry.RegisterLocalWithCapabilityCatalog(
					"sess-unknown-id",
					"builders",
					responder,
					[]sessionpkg.NetworkPeerCapability{{
						ID:      "review-pr",
						Summary: "Review pull requests",
						Outcome: "Actionable feedback",
					}},
					now,
				); err != nil {
					t.Fatalf("RegisterLocalWithCapabilityCatalog() error = %v", err)
				}
				return responder.PeerID
			},
		},
		{
			name: "no capability catalog",
			registerFunc: func(t *testing.T, registry *PeerRegistry, now time.Time) string {
				t.Helper()

				responder := mustPeerCard(t, "reviewer.sess-no-catalog")
				if _, err := registry.RegisterLocal("sess-no-catalog", "builders", responder, now); err != nil {
					t.Fatalf("RegisterLocal() error = %v", err)
				}
				return responder.PeerID
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
			if err != nil {
				t.Fatalf("NewPeerRegistry() error = %v", err)
			}
			peerID := tc.registerFunc(t, registry, now)

			router, err := NewRouter(
				registry,
				&spyRouterTransport{},
				DefaultMaxReplayAge,
				WithRouterClock(func() time.Time { return now }),
			)
			if err != nil {
				t.Fatalf("NewRouter() error = %v", err)
			}

			requestPayload, err := json.Marshal(Envelope{
				Protocol: ProtocolV0,
				ID:       "msg_whois_empty_request",
				Kind:     KindWhois,
				Channel:  "builders",
				From:     "coder.sess-a",
				To:       stringPtr(peerID),
				TS:       now.Unix(),
				Body:     mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
				Ext: ExtensionMap{
					whoisIncludeExtKey: mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
				},
			})
			if err != nil {
				t.Fatalf("json.Marshal(empty rich whois request) error = %v", err)
			}
			if tc.capabilityIDs != nil {
				var env Envelope
				if err := json.Unmarshal(requestPayload, &env); err != nil {
					t.Fatalf("json.Unmarshal(empty rich whois request) error = %v", err)
				}
				env.Ext[whoisCapabilityIDsExtKey] = mustRawJSON(t, tc.capabilityIDs)
				requestPayload, err = json.Marshal(env)
				if err != nil {
					t.Fatalf("json.Marshal(empty rich whois request with ids) error = %v", err)
				}
			}

			result, err := router.Receive(context.Background(), requestPayload)
			if err != nil {
				t.Fatalf("Receive(empty rich whois request) error = %v", err)
			}
			response := result.Generated[0]
			payload := decodeWhoisCapabilityCatalogPayload(t, response.Ext[whoisCapabilityCatalogExtKey])
			wantPayload := whoisCapabilityCatalogPayload{Capabilities: []whoisCapabilityCatalogEntry{}}
			if !reflect.DeepEqual(payload, wantPayload) {
				t.Fatalf("empty capability catalog = %#v, want %#v", payload, wantPayload)
			}
		})
	}
}

func TestRouterWhoisRequestIgnoresUnknownAGHExtKeys(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 19, 6, 30, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	responder := mustPeerCard(t, "reviewer.sess-unknown-ext")
	if _, err := registry.RegisterLocal("sess-unknown-ext", "builders", responder, now); err != nil {
		t.Fatalf("RegisterLocal() error = %v", err)
	}

	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	requestPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_unknown_ext",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     "coder.sess-a",
		To:       stringPtr(responder.PeerID),
		TS:       now.Unix(),
		Body:     mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
		Ext: ExtensionMap{
			"agh.unknown": mustRawJSON(t, map[string]any{"note": "ignored"}),
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(unknown ext request) error = %v", err)
	}

	result, err := router.Receive(context.Background(), requestPayload)
	if err != nil {
		t.Fatalf("Receive(unknown ext request) error = %v", err)
	}
	if got, want := len(result.Generated), 1; got != want {
		t.Fatalf("len(unknown ext responses) = %d, want %d", got, want)
	}
	if len(result.Generated[0].Ext) != 0 {
		t.Fatalf("response.Ext = %#v, want lean response for ignored AGH ext key", result.Generated[0].Ext)
	}
}

func TestRouterWhoisRichCapabilityDiscoveryRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 19, 6, 40, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}

	responder := mustPeerCard(t, "reviewer.sess-oversized")
	if _, err := registry.RegisterLocalWithCapabilityCatalog(
		"sess-oversized",
		"builders",
		responder,
		[]sessionpkg.NetworkPeerCapability{{
			ID:      "review-pr",
			Summary: "Review pull requests",
			Outcome: strings.Repeat("x", maxProtocolEnvelopeBytes),
		}},
		now,
	); err != nil {
		t.Fatalf("RegisterLocalWithCapabilityCatalog() error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	requestPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_oversized_request",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     "coder.sess-a",
		To:       stringPtr(responder.PeerID),
		TS:       now.Unix(),
		Body:     mustRawJSON(t, WhoisBody{Type: WhoisTypeRequest}),
		Ext: ExtensionMap{
			whoisIncludeExtKey: mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(oversized rich whois request) error = %v", err)
	}

	_, err = router.Receive(context.Background(), requestPayload)
	if !errors.Is(err, ErrEnvelopeTooLarge) {
		t.Fatalf("Receive(oversized rich whois request) error = %v, want ErrEnvelopeTooLarge", err)
	}
	if got := transport.Count(); got != 0 {
		t.Fatalf("transport publish count = %d, want 0 after oversized rich whois rejection", got)
	}
}

func TestRouterWhoisResponseRefreshesRemotePresenceAndDeliversToRequester(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 12, 45, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "coder.sess-a")
	if _, err := registry.RegisterLocal("sess-a", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}
	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	remote := mustPeerCard(t, "reviewer.sess-b")
	responsePayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_response",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     remote.PeerID,
		To:       stringPtr(local.PeerID),
		ReplyTo:  stringPtr("msg_whois_request"),
		TS:       now.Unix(),
		Body: mustRawJSON(t, WhoisBody{
			Type:     WhoisTypeResponse,
			PeerCard: &remote,
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(whois response) error = %v", err)
	}

	result, err := router.Receive(context.Background(), responsePayload)
	if err != nil {
		t.Fatalf("Receive(whois response) error = %v", err)
	}
	if result.Rejected || len(result.Generated) != 0 {
		t.Fatalf("whois response result = %#v, want delivery plus cache refresh", result)
	}
	if got, want := len(result.Deliveries), 1; got != want {
		t.Fatalf("len(whois response deliveries) = %d, want %d", got, want)
	}
	if got, want := result.Deliveries[0].SessionID, "sess-a"; got != want {
		t.Fatalf("whois response delivery session = %q, want %q", got, want)
	}
	if _, ok := registry.RemoteByPeer("builders", remote.PeerID, now); !ok {
		t.Fatalf("RemoteByPeer(%q) = missing after whois response", remote.PeerID)
	}
}

func TestRouterHeartbeatPublishAndLeaveHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 0, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(5*time.Millisecond, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "coder.sess-a")
	if err := applyCapabilityBriefProjection(&local, []sessionpkg.NetworkPeerCapability{{
		ID:      "review-pr",
		Summary: "Review pull requests",
	}}); err != nil {
		t.Fatalf("applyCapabilityBriefProjection() error = %v", err)
	}
	if _, err := registry.RegisterLocal("sess-a", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	greet, err := router.PublishGreet(context.Background(), "sess-a", "hello")
	if err != nil {
		t.Fatalf("PublishGreet() error = %v", err)
	}
	if got, want := greet.Subject, "agh.network.v0.builders.broadcast"; got != want {
		t.Fatalf("PublishGreet().Subject = %q, want %q", got, want)
	}
	firstMessage := transport.Message(0)
	var firstEnvelope Envelope
	if err := json.Unmarshal(firstMessage.payload, &firstEnvelope); err != nil {
		t.Fatalf("json.Unmarshal(first greet envelope) error = %v", err)
	}
	decoded, err := firstEnvelope.DecodeBody()
	if err != nil {
		t.Fatalf("DecodeBody(first greet) error = %v", err)
	}
	firstGreet := decoded.(GreetBody)
	if got := decodeCapabilityBriefPayload(
		t,
		firstGreet.PeerCard.Ext[capabilityBriefExtKey],
	); !slices.Equal(
		got,
		[]capabilityBrief{{
			ID:      "review-pr",
			Summary: "Review pull requests",
		}},
	) {
		t.Fatalf("first greet capability brief = %#v, want review-pr brief entry", got)
	}

	ctx := t.Context()
	heartbeat, err := router.StartHeartbeat(ctx, "sess-a", "alive")
	if err != nil {
		t.Fatalf("StartHeartbeat() error = %v", err)
	}
	if heartbeat.Done() == nil {
		t.Fatal("Heartbeat.Done() = nil, want non-nil channel")
	}
	waitForPublishCount(t, transport, 2)
	heartbeat.Stop()

	countAfterStop := transport.Count()
	time.Sleep(20 * time.Millisecond)
	if got := transport.Count(); got != countAfterStop {
		t.Fatalf("transport publishes after heartbeat stop = %d, want stable %d", got, countAfterStop)
	}

	left, ok := router.Leave("sess-a")
	if !ok {
		t.Fatal("Leave(sess-a) ok = false, want true")
	}
	if got, want := left.SessionID, "sess-a"; got != want {
		t.Fatalf("Leave(sess-a).SessionID = %q, want %q", got, want)
	}
	if _, present := registry.LookupPresence("builders", local.PeerID, now); present {
		t.Fatalf("LookupPresence(builders, %q) = present after leave, want removed", local.PeerID)
	}
}

func TestRouterReceiveRejectsNotTargetAndMapsMalformedErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 15, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}
	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	notTargetPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_not_target",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr("reviewer.sess-other"),
		InteractionID: stringPtr("int_not_target"),
		TS:            now.Unix(),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(not_target) error = %v", err)
	}
	notTarget, err := router.Receive(context.Background(), notTargetPayload)
	if err != nil {
		t.Fatalf("Receive(not target) error = %v", err)
	}
	if !notTarget.Rejected || notTarget.ReasonCode == nil || *notTarget.ReasonCode != ReasonCodeNotTarget {
		t.Fatalf("not target result = %#v, want reason %q", notTarget, ReasonCodeNotTarget)
	}

	malformed, err := router.Receive(context.Background(), []byte(`{"protocol":"agh-network/v0","kind":"direct"`))
	if err != nil {
		t.Fatalf("Receive(malformed JSON) error = %v", err)
	}
	if !malformed.Rejected || malformed.ReasonCode == nil || *malformed.ReasonCode != ReasonCodeMalformed {
		t.Fatalf("malformed result = %#v, want reason %q", malformed, ReasonCodeMalformed)
	}

	unsupported, err := router.Receive(context.Background(), []byte(`{
		"protocol":"agh-network/v0",
		"id":"msg_bad_kind",
		"kind":"mystery",
		"channel":"builders",
		"from":"coder.sess-a",
		"ts":1775826900,
		"body":{}
	}`))
	if err != nil {
		t.Fatalf("Receive(unsupported kind) error = %v", err)
	}
	if !unsupported.Rejected || unsupported.ReasonCode == nil || *unsupported.ReasonCode != ReasonCodeUnsupportedKind {
		t.Fatalf("unsupported result = %#v, want reason %q", unsupported, ReasonCodeUnsupportedKind)
	}
}

func TestRouterReceiveRejectsCapabilityDigestMismatchBeforeDelivery(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 17, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(
		registry,
		transport,
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	payload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_capability_bad_digest",
		Kind:          KindCapability,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(local.PeerID),
		InteractionID: stringPtr("int_capability_bad_digest"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, CapabilityBody{
			Capability: CapabilityEnvelopePayload{
				ID:               "review-fix",
				Summary:          "Review fix flow",
				Outcome:          "A reusable review fix workflow.",
				Version:          "1.0.0",
				Digest:           "sha256:not-the-canonical-digest",
				ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
				Requirements:     []string{"workspace-write"},
			},
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(capability mismatch) error = %v", err)
	}

	result, err := router.Receive(context.Background(), payload)
	if err != nil {
		t.Fatalf("Receive(capability mismatch) error = %v", err)
	}
	if !result.Rejected || result.ReasonCode == nil || *result.ReasonCode != ReasonCodeVerificationFailed {
		t.Fatalf("capability mismatch result = %#v, want rejected verification_failed", result)
	}
	if got := len(result.Deliveries); got != 0 {
		t.Fatalf("len(capability mismatch deliveries) = %d, want 0", got)
	}
	if got := len(result.Generated); got != 0 {
		t.Fatalf("len(capability mismatch generated) = %d, want 0", got)
	}
	if got := transport.Count(); got != 0 {
		t.Fatalf("transport publish count = %d, want 0", got)
	}
}

func TestRouterReceiveExpiredDirectGeneratesExpiredReceipt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 20, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	expiredAt := now.Add(-time.Second).Unix()
	payload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_expired_direct",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(local.PeerID),
		InteractionID: stringPtr("int_expired"),
		TS:            now.Add(-2 * time.Second).Unix(),
		ExpiresAt:     &expiredAt,
		Body:          mustRawJSON(t, DirectBody{Text: "too late"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(expired direct) error = %v", err)
	}

	result, err := router.Receive(context.Background(), payload)
	if err != nil {
		t.Fatalf("Receive(expired direct) error = %v", err)
	}
	if !result.Rejected || result.ReasonCode == nil || *result.ReasonCode != ReasonCodeExpired {
		t.Fatalf("expired direct result = %#v, want rejected expired", result)
	}
	if result.Envelope == nil || result.Envelope.ID != "msg_expired_direct" {
		t.Fatalf("expired direct envelope = %#v, want partial envelope for auditing", result.Envelope)
	}
	if got, want := len(result.Generated), 1; got != want {
		t.Fatalf("len(expired direct generated) = %d, want %d", got, want)
	}
	if got, want := transport.Count(), 1; got != want {
		t.Fatalf("transport publish count = %d, want %d generated receipt", got, want)
	}

	body, decodeErr := result.Generated[0].DecodeBody()
	if decodeErr != nil {
		t.Fatalf("DecodeBody(expired receipt) error = %v", decodeErr)
	}
	receipt := body.(ReceiptBody)
	if got, want := receipt.Status, ReceiptStatusExpired; got != want {
		t.Fatalf("expired receipt status = %q, want %q", got, want)
	}
	if receipt.ReasonCode == nil || *receipt.ReasonCode != ReasonCodeExpired {
		t.Fatalf("expired receipt reason = %v, want %q", receipt.ReasonCode, ReasonCodeExpired)
	}
}

func TestRouterReceivesGreetAndDirectedWhoisRequest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 25, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}

	transport := &spyRouterTransport{}
	router, err := NewRouter(registry, transport, DefaultMaxReplayAge, WithRouterClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	remote := mustPeerCard(t, "coder.sess-a")
	greetPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_greet_remote",
		Kind:     KindGreet,
		Channel:  "builders",
		From:     remote.PeerID,
		TS:       now.Unix(),
		Body: mustRawJSON(t, GreetBody{
			PeerCard: remote,
			Summary:  "hello",
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(greet) error = %v", err)
	}
	if _, err := router.Receive(context.Background(), greetPayload); err != nil {
		t.Fatalf("Receive(greet) error = %v", err)
	}
	if _, ok := registry.RemoteByPeer("builders", remote.PeerID, now); !ok {
		t.Fatalf("RemoteByPeer(%q) = missing after greet", remote.PeerID)
	}

	whoisPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg_whois_direct",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     remote.PeerID,
		To:       stringPtr(local.PeerID),
		TS:       now.Unix(),
		Body: mustRawJSON(t, WhoisBody{
			Type:  WhoisTypeRequest,
			Query: "not-a-match-but-directed",
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(directed whois) error = %v", err)
	}
	result, err := router.Receive(context.Background(), whoisPayload)
	if err != nil {
		t.Fatalf("Receive(directed whois) error = %v", err)
	}
	if got, want := len(result.Generated), 1; got != want {
		t.Fatalf("len(directed whois responses) = %d, want %d", got, want)
	}
	if got, want := transport.Count(), 1; got != want {
		t.Fatalf("transport publish count = %d, want %d", got, want)
	}
}

func TestRouterConstructionAndHelperErrors(t *testing.T) {
	t.Parallel()

	if _, err := NewRouter(nil, &spyRouterTransport{}, DefaultMaxReplayAge); err == nil {
		t.Fatal("NewRouter(nil peers) error = nil, want non-nil")
	}

	registry, err := NewPeerRegistry(10 * time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "coder.sess-a")
	if _, err := registry.RegisterLocal("sess-a", "builders", local, time.Now().UTC()); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}

	router, err := NewRouter(registry, nil, DefaultMaxReplayAge)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	if _, err := router.PublishGreet(context.Background(), "sess-a", "hello"); err == nil {
		t.Fatal("PublishGreet(nil transport) error = nil, want non-nil")
	}
	if _, err := router.PublishGreet(context.Background(), "missing", "hello"); !errors.Is(err, ErrLocalPeerNotFound) {
		t.Fatalf("PublishGreet(missing local) error = %v, want ErrLocalPeerNotFound", err)
	}
	if _, err := router.StartHeartbeat(
		context.Background(),
		"missing",
		"hello",
	); !errors.Is(
		err,
		ErrLocalPeerNotFound,
	) {
		t.Fatalf("StartHeartbeat(missing local) error = %v, want ErrLocalPeerNotFound", err)
	}

	if _, err := marshalEnvelopeBody(badRouterBody{Ch: make(chan int)}); err == nil {
		t.Fatal("marshalEnvelopeBody(badBody) error = nil, want non-nil")
	}

	future := nowWithUnix(1775822400)
	expiresAt := future.Unix() + 5
	deadline := replayDeadline(Envelope{TS: future.Unix(), ExpiresAt: &expiresAt}, future, time.Minute)
	if got, want := deadline.Unix(), expiresAt; got != want {
		t.Fatalf("replayDeadline(expires_at).Unix() = %d, want %d", got, want)
	}

	farFutureExpiry := future.Add(2 * time.Minute).Unix()
	deadline = replayDeadline(Envelope{TS: future.Unix(), ExpiresAt: &farFutureExpiry}, future, time.Minute)
	if got, want := deadline, future.Add(time.Minute).UTC(); !got.Equal(want) {
		t.Fatalf("replayDeadline(clamped).UTC() = %s, want %s", got, want)
	}
}

func TestInteractionValidationErrors(t *testing.T) {
	t.Parallel()

	if err := (Interaction{}).Validate(); err == nil {
		t.Fatal("Interaction{}.Validate() error = nil, want non-nil")
	}

	if _, err := OpenInteraction(Envelope{Kind: KindSay}, time.Time{}); err == nil {
		t.Fatal("OpenInteraction(non-opener) error = nil, want non-nil")
	}
}

func TestRouterDirectedCapabilityOpensInteractionForReceiptAndTrace(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 37, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	alpha := mustPeerCard(t, "alpha.sess-a")
	delta := mustPeerCard(t, "delta.sess-b")
	if _, err := registry.RegisterLocal("sess-alpha", "builders", alpha, now); err != nil {
		t.Fatalf("RegisterLocal(alpha) error = %v", err)
	}
	if _, err := registry.RegisterLocal("sess-delta", "builders", delta, now); err != nil {
		t.Fatalf("RegisterLocal(delta) error = %v", err)
	}

	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	capabilityPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_capability_open",
		Kind:          KindCapability,
		Channel:       "builders",
		From:          alpha.PeerID,
		To:            stringPtr(delta.PeerID),
		InteractionID: stringPtr("int_capability_open"),
		TS:            now.Unix(),
		Body: mustCapabilityBodyJSON(t, CapabilityEnvelopePayload{
			ID:               "review-fix",
			Summary:          "Review fix flow",
			Outcome:          "A reusable review fix workflow.",
			Version:          "1.0.0",
			ExecutionOutline: []string{"Inspect the issue", "Draft the fix"},
			Requirements:     []string{"workspace-write"},
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(capability) error = %v", err)
	}

	result, err := router.Receive(context.Background(), capabilityPayload)
	if err != nil {
		t.Fatalf("Receive(capability) error = %v", err)
	}
	if got, want := len(result.Deliveries), 1; got != want {
		t.Fatalf("len(capability deliveries) = %d, want %d", got, want)
	}
	if got, want := result.Deliveries[0].SessionID, "sess-delta"; got != want {
		t.Fatalf("capability delivery session = %q, want %q", got, want)
	}

	receiptPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_capability_receipt",
		Kind:          KindReceipt,
		Channel:       "builders",
		From:          delta.PeerID,
		To:            stringPtr(alpha.PeerID),
		InteractionID: stringPtr("int_capability_open"),
		ReplyTo:       stringPtr("msg_capability_open"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, ReceiptBody{
			ForID:  "msg_capability_open",
			Status: ReceiptStatusAccepted,
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(receipt) error = %v", err)
	}

	receiptResult, err := router.Receive(context.Background(), receiptPayload)
	if err != nil {
		t.Fatalf("Receive(capability receipt) error = %v", err)
	}
	if receiptResult.Ignored || receiptResult.Rejected {
		t.Fatalf("capability receipt result = %#v, want delivered receipt", receiptResult)
	}
	if got, want := len(receiptResult.Deliveries), 1; got != want {
		t.Fatalf("len(capability receipt deliveries) = %d, want %d", got, want)
	}
	if got, want := receiptResult.Deliveries[0].SessionID, "sess-alpha"; got != want {
		t.Fatalf("capability receipt delivery session = %q, want %q", got, want)
	}

	tracePayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_capability_trace",
		Kind:          KindTrace,
		Channel:       "builders",
		From:          delta.PeerID,
		To:            stringPtr(alpha.PeerID),
		InteractionID: stringPtr("int_capability_open"),
		ReplyTo:       stringPtr("msg_capability_open"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, TraceBody{
			State: StateWorking,
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(trace) error = %v", err)
	}

	traceResult, err := router.Receive(context.Background(), tracePayload)
	if err != nil {
		t.Fatalf("Receive(capability trace) error = %v", err)
	}
	if traceResult.Ignored || traceResult.Rejected {
		t.Fatalf("capability trace result = %#v, want delivered trace", traceResult)
	}
	if got, want := len(traceResult.Deliveries), 1; got != want {
		t.Fatalf("len(capability trace deliveries) = %d, want %d", got, want)
	}
	if got, want := traceResult.Deliveries[0].SessionID, "sess-alpha"; got != want {
		t.Fatalf("capability trace delivery session = %q, want %q", got, want)
	}
}

func TestRouterReceiveRejectsInvalidLifecycleTransition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 13, 35, 0, 0, time.UTC)
	registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	local := mustPeerCard(t, "reviewer.sess-b")
	if _, err := registry.RegisterLocal("sess-b", "builders", local, now); err != nil {
		t.Fatalf("RegisterLocal(local) error = %v", err)
	}
	router, err := NewRouter(
		registry,
		&spyRouterTransport{},
		DefaultMaxReplayAge,
		WithRouterClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	directPayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_direct_invalid_trace",
		Kind:          KindDirect,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(local.PeerID),
		InteractionID: stringPtr("int_invalid_trace"),
		TS:            now.Unix(),
		Body:          mustRawJSON(t, DirectBody{Text: "please review"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(direct) error = %v", err)
	}
	if _, err := router.Receive(context.Background(), directPayload); err != nil {
		t.Fatalf("Receive(direct) error = %v", err)
	}

	tracePayload, err := json.Marshal(Envelope{
		Protocol:      ProtocolV0,
		ID:            "msg_trace_invalid_state",
		Kind:          KindTrace,
		Channel:       "builders",
		From:          "coder.sess-a",
		To:            stringPtr(local.PeerID),
		InteractionID: stringPtr("int_invalid_trace"),
		TS:            now.Unix(),
		Body: mustRawJSON(t, TraceBody{
			State: StateSubmitted,
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(trace) error = %v", err)
	}

	result, err := router.Receive(context.Background(), tracePayload)
	if err != nil {
		t.Fatalf("Receive(trace invalid state) error = %v", err)
	}
	if !result.Rejected || result.ReasonCode == nil || *result.ReasonCode != ReasonCodeInternal {
		t.Fatalf("invalid lifecycle result = %#v, want reason %q", result, ReasonCodeInternal)
	}
	if got, want := len(result.Deliveries), 0; got != want {
		t.Fatalf("len(invalid lifecycle deliveries) = %d, want %d", got, want)
	}
}

type spyRouterTransport struct {
	mu       sync.Mutex
	messages []publishedMessage
}

type publishedMessage struct {
	subject string
	payload []byte
}

type badRouterBody struct {
	Ch chan int `json:"ch"`
}

func (badRouterBody) Kind() Kind { return KindSay }

func (s *spyRouterTransport) Publish(_ context.Context, subject string, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cloned := append([]byte(nil), payload...)
	s.messages = append(s.messages, publishedMessage{
		subject: subject,
		payload: cloned,
	})
	return nil
}

func (s *spyRouterTransport) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

func (s *spyRouterTransport) Message(index int) publishedMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	message := s.messages[index]
	return publishedMessage{
		subject: message.subject,
		payload: append([]byte(nil), message.payload...),
	}
}

func waitForPublishCount(t *testing.T, transport *spyRouterTransport, want int) {
	t.Helper()

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if transport.Count() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("transport publish count = %d, want at least %d", transport.Count(), want)
}

func nowWithUnix(ts int64) time.Time {
	return time.Unix(ts, 0).UTC()
}
