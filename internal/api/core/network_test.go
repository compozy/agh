package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
)

func TestNetworkConversionHelpersPreserveMetadata(t *testing.T) {
	t.Parallel()

	deadline := int64(1775823000)
	status := &network.NetworkStatus{
		Enabled:              true,
		Status:               network.StatusRunning,
		ListenerHost:         "127.0.0.1",
		ListenerPort:         4222,
		LocalPeers:           1,
		RemotePeers:          2,
		Spaces:               1,
		QueuedMessages:       3,
		QueuedSessions:       1,
		DeliveryWorkers:      1,
		MessagesSent:         4,
		MessagesReceived:     5,
		MessagesRejected:     1,
		MessagesDelivered:    3,
		WorkflowTaggedEvents: 2,
		HandoffTaggedEvents:  1,
		LastDisconnect:       "transport lost",
		KindMetrics: []network.KindMetric{{
			Kind:      network.KindSay,
			Sent:      4,
			Received:  5,
			Rejected:  1,
			Delivered: 3,
		}},
	}

	payload := core.NetworkStatusPayloadFromStatus(status)
	if payload == nil || payload.MessagesDelivered != 3 || len(payload.KindMetrics) != 1 || payload.KindMetrics[0].Kind != string(network.KindSay) {
		t.Fatalf("NetworkStatusPayloadFromStatus() = %#v", payload)
	}

	req := contract.NetworkSendRequest{
		SessionID:     " sess-a ",
		Space:         " builders ",
		Kind:          "say",
		To:            " reviewer.sess-b ",
		Body:          json.RawMessage(`{"text":"hello"}`),
		InteractionID: " int-1 ",
		ReplyTo:       " msg-root ",
		TraceID:       " trace-1 ",
		CausationID:   " cause-1 ",
		ExpiresAt:     &deadline,
		ID:            " msg-1 ",
		Ext: map[string]json.RawMessage{
			"agh.workflow_id":     json.RawMessage(`"wf-1"`),
			"agh.handoff_version": json.RawMessage(`3`),
		},
	}

	converted, err := core.NetworkSendRequestFromPayload(req)
	if err != nil {
		t.Fatalf("NetworkSendRequestFromPayload() error = %v", err)
	}
	if converted.SessionID != "sess-a" || converted.Space != "builders" || converted.Kind != network.KindSay {
		t.Fatalf("converted request = %#v", converted)
	}
	if converted.To == nil || *converted.To != "reviewer.sess-b" {
		t.Fatalf("converted.To = %#v, want reviewer.sess-b", converted.To)
	}
	if string(converted.Body) != `{"text":"hello"}` {
		t.Fatalf("converted.Body = %s, want preserved JSON", string(converted.Body))
	}
	if string(converted.Ext["agh.workflow_id"]) != `"wf-1"` || string(converted.Ext["agh.handoff_version"]) != `3` {
		t.Fatalf("converted.Ext = %#v, want preserved ext metadata", converted.Ext)
	}

	proof := network.Proof{"sig": json.RawMessage(`"abc123"`)}
	traceID := "trace-1"
	causationID := "cause-1"
	replyTo := "msg-root"
	envelope := network.Envelope{
		Protocol:    network.ProtocolV0,
		ID:          "msg-1",
		Kind:        network.KindDirect,
		Space:       "builders",
		From:        "reviewer.sess-b",
		ReplyTo:     &replyTo,
		TraceID:     &traceID,
		CausationID: &causationID,
		TS:          deadline,
		ExpiresAt:   &deadline,
		Body:        json.RawMessage(`{"text":"hello","intent":"review"}`),
		Proof:       &proof,
		Ext:         network.ExtensionMap{"agh.workflow_id": json.RawMessage(`"wf-1"`), "agh.handoff_version": json.RawMessage(`3`)},
	}

	envelopePayload := core.NetworkEnvelopePayloadFromEnvelope(envelope)
	if envelopePayload.TraceID == nil || *envelopePayload.TraceID != "trace-1" {
		t.Fatalf("TraceID = %#v, want trace-1", envelopePayload.TraceID)
	}
	if string(envelopePayload.Proof["sig"]) != `"abc123"` {
		t.Fatalf("Proof = %#v, want cloned proof payload", envelopePayload.Proof)
	}
	if string(envelopePayload.Ext["agh.handoff_version"]) != `3` {
		t.Fatalf("Ext = %#v, want cloned ext payload", envelopePayload.Ext)
	}
}

func TestBaseHandlersNetworkEndpoints(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true

	fixedNow := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	deadline := int64(1775823000)
	fixture.Handlers.Network = testutil.StubNetworkService{
		StatusFn: func(context.Context) (*network.NetworkStatus, error) {
			return &network.NetworkStatus{
				Enabled:              true,
				Status:               network.StatusRunning,
				ListenerHost:         "127.0.0.1",
				ListenerPort:         4222,
				LocalPeers:           1,
				RemotePeers:          1,
				Spaces:               1,
				QueuedMessages:       2,
				QueuedSessions:       1,
				DeliveryWorkers:      1,
				MessagesSent:         4,
				MessagesReceived:     5,
				MessagesRejected:     1,
				MessagesDelivered:    3,
				WorkflowTaggedEvents: 2,
				HandoffTaggedEvents:  1,
				LastDisconnect:       "transport lost",
				KindMetrics: []network.KindMetric{{
					Kind:      network.KindSay,
					Sent:      4,
					Received:  5,
					Rejected:  1,
					Delivered: 3,
				}},
			}, nil
		},
		ListPeersFn: func(_ context.Context, space string) ([]network.PeerInfo, error) {
			if space != "builders" {
				t.Fatalf("ListPeers() space = %q, want builders", space)
			}
			displayName := "Reviewer"
			sessionID := "sess-a"
			return []network.PeerInfo{{
				SessionID: &sessionID,
				PeerID:    "reviewer.sess-a",
				Space:     "builders",
				Local:     true,
				PeerCard: network.PeerCard{
					PeerID:              "reviewer.sess-a",
					DisplayName:         &displayName,
					ProfilesSupported:   []string{"v0"},
					Capabilities:        []string{"send"},
					ArtifactsSupported:  []string{"text"},
					TrustModesSupported: []string{"untrusted"},
					Ext:                 network.ExtensionMap{"agh.workflow_id": json.RawMessage(`"wf-1"`)},
				},
				JoinedAt:  timePtr(fixedNow),
				LastSeen:  timePtr(fixedNow),
				ExpiresAt: timePtr(fixedNow.Add(time.Minute)),
			}}, nil
		},
		ListSpacesFn: func(context.Context) ([]network.SpaceInfo, error) {
			return []network.SpaceInfo{{Space: "builders", PeerCount: 2}}, nil
		},
		SendFn: func(_ context.Context, req network.SendRequest) (string, error) {
			if req.SessionID != "sess-a" || req.Space != "builders" || req.Kind != network.KindSay {
				t.Fatalf("Send() req = %#v", req)
			}
			if string(req.Ext["agh.workflow_id"]) != `"wf-1"` || string(req.Ext["agh.handoff_version"]) != `3` {
				t.Fatalf("Send() ext = %#v, want workflow/handoff metadata", req.Ext)
			}
			return "msg-1", nil
		},
		InboxFn: func(_ context.Context, sessionID string) ([]network.Envelope, error) {
			if sessionID != "sess-a" {
				t.Fatalf("Inbox() sessionID = %q, want sess-a", sessionID)
			}
			replyTo := "msg-root"
			traceID := "trace-1"
			proof := network.Proof{"sig": json.RawMessage(`"abc123"`)}
			return []network.Envelope{{
				Protocol: network.ProtocolV0,
				ID:       "msg-inbox",
				Kind:     network.KindDirect,
				Space:    "builders",
				From:     "reviewer.sess-a",
				ReplyTo:  &replyTo,
				TraceID:  &traceID,
				TS:       fixedNow.Unix(),
				Body:     json.RawMessage(`{"text":"follow up","intent":"review"}`),
				Proof:    &proof,
				Ext: network.ExtensionMap{
					"agh.workflow_id":     json.RawMessage(`"wf-1"`),
					"agh.handoff_version": json.RawMessage(`3`),
				},
			}}, nil
		},
	}

	t.Run("ShouldReturnNetworkStatus", func(t *testing.T) {
		statusResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/status", nil)
		if statusResp.Code != http.StatusOK {
			t.Fatalf("status code = %d, want %d", statusResp.Code, http.StatusOK)
		}

		var statusPayload contract.NetworkStatusResponse
		testutil.DecodeJSONResponse(t, statusResp, &statusPayload)
		if statusPayload.Network.QueuedMessages != 2 || len(statusPayload.Network.KindMetrics) != 1 {
			t.Fatalf("status payload = %#v", statusPayload.Network)
		}
		if statusPayload.Network.KindMetrics[0].Sent != 4 || deadline == 0 {
			t.Fatalf("kind metrics = %#v", statusPayload.Network.KindMetrics)
		}
	})

	t.Run("ShouldListNetworkPeers", func(t *testing.T) {
		peersResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/peers?space=builders", nil)
		if peersResp.Code != http.StatusOK {
			t.Fatalf("peers code = %d, want %d", peersResp.Code, http.StatusOK)
		}

		var peersPayload contract.NetworkPeersResponse
		testutil.DecodeJSONResponse(t, peersResp, &peersPayload)
		if len(peersPayload.Peers) != 1 || peersPayload.Peers[0].PeerCard.DisplayName == nil || *peersPayload.Peers[0].PeerCard.DisplayName != "Reviewer" {
			t.Fatalf("peers payload = %#v", peersPayload.Peers)
		}
	})

	t.Run("ShouldListNetworkSpaces", func(t *testing.T) {
		spacesResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/spaces", nil)
		if spacesResp.Code != http.StatusOK {
			t.Fatalf("spaces code = %d, want %d", spacesResp.Code, http.StatusOK)
		}

		var spacesPayload contract.NetworkSpacesResponse
		testutil.DecodeJSONResponse(t, spacesResp, &spacesPayload)
		if len(spacesPayload.Spaces) != 1 || spacesPayload.Spaces[0].PeerCount != 2 {
			t.Fatalf("spaces payload = %#v", spacesPayload.Spaces)
		}
	})

	t.Run("ShouldSendNetworkMessages", func(t *testing.T) {
		sendResp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{"session_id":"sess-a","space":"builders","kind":"say","body":{"text":"hello"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}`))
		if sendResp.Code != http.StatusOK {
			t.Fatalf("send code = %d, want %d; body=%s", sendResp.Code, http.StatusOK, sendResp.Body.String())
		}

		var sendPayload contract.NetworkSendResponse
		testutil.DecodeJSONResponse(t, sendResp, &sendPayload)
		if sendPayload.Message.ID != "msg-1" || string(sendPayload.Message.Ext["agh.workflow_id"]) != `"wf-1"` {
			t.Fatalf("send payload = %#v", sendPayload.Message)
		}
		if got := sendPayload.Message.ExpiresAt; got != nil {
			t.Fatalf("send expires_at = %#v, want nil without flag", got)
		}
	})

	t.Run("ShouldReturnNetworkInboxMessages", func(t *testing.T) {
		inboxResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/inbox?session_id=sess-a", nil)
		if inboxResp.Code != http.StatusOK {
			t.Fatalf("inbox code = %d, want %d", inboxResp.Code, http.StatusOK)
		}

		var inboxPayload contract.NetworkInboxResponse
		testutil.DecodeJSONResponse(t, inboxResp, &inboxPayload)
		if len(inboxPayload.Messages) != 1 {
			t.Fatalf("inbox payload len = %d, want 1", len(inboxPayload.Messages))
		}
		if string(inboxPayload.Messages[0].Proof["sig"]) != `"abc123"` || string(inboxPayload.Messages[0].Ext["agh.handoff_version"]) != `3` {
			t.Fatalf("inbox payload = %#v", inboxPayload.Messages[0])
		}
		if got := inboxPayload.Messages[0].ExpiresAt; got != nil {
			t.Fatalf("ExpiresAt = %#v, want nil for non-expiring inbox message", got)
		}
	})
}

func TestBaseHandlersNetworkErrorsAndDisabledMode(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)

	disabledResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/status", nil)
	if disabledResp.Code != http.StatusOK {
		t.Fatalf("disabled status code = %d, want %d", disabledResp.Code, http.StatusOK)
	}
	var disabledPayload contract.NetworkStatusResponse
	testutil.DecodeJSONResponse(t, disabledResp, &disabledPayload)
	if disabledPayload.Network.Enabled || disabledPayload.Network.Status != "disabled" {
		t.Fatalf("disabled payload = %#v, want disabled status", disabledPayload.Network)
	}

	fixture.Handlers.Config.Network.Enabled = true

	peersUnavailable := performRequest(t, fixture.Engine, http.MethodGet, "/network/peers", nil)
	if peersUnavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("peers unavailable code = %d, want %d", peersUnavailable.Code, http.StatusServiceUnavailable)
	}

	fixture.Handlers.Network = testutil.StubNetworkService{
		StatusFn: func(context.Context) (*network.NetworkStatus, error) {
			return nil, errors.New("boom")
		},
		ListSpacesFn: func(context.Context) ([]network.SpaceInfo, error) {
			return nil, network.ErrInvalidField
		},
		SendFn: func(context.Context, network.SendRequest) (string, error) {
			return "", network.ErrTargetPeerNotFound
		},
		InboxFn: func(context.Context, string) ([]network.Envelope, error) {
			return nil, network.ErrInvalidField
		},
	}

	statusResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/status", nil)
	if statusResp.Code != http.StatusInternalServerError {
		t.Fatalf("status error code = %d, want %d", statusResp.Code, http.StatusInternalServerError)
	}

	spacesResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/spaces", nil)
	if spacesResp.Code != http.StatusBadRequest {
		t.Fatalf("spaces error code = %d, want %d", spacesResp.Code, http.StatusBadRequest)
	}

	sendDecodeResp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{`))
	if sendDecodeResp.Code != http.StatusBadRequest {
		t.Fatalf("send decode code = %d, want %d", sendDecodeResp.Code, http.StatusBadRequest)
	}

	sendResp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{"session_id":"sess-a","space":"builders","kind":"say","body":{"text":"hello"}}`))
	if sendResp.Code != http.StatusNotFound {
		t.Fatalf("send error code = %d, want %d", sendResp.Code, http.StatusNotFound)
	}

	inboxMissingResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/inbox", nil)
	if inboxMissingResp.Code != http.StatusBadRequest {
		t.Fatalf("inbox missing code = %d, want %d", inboxMissingResp.Code, http.StatusBadRequest)
	}

	inboxResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/inbox?session_id=sess-a", nil)
	if inboxResp.Code != http.StatusBadRequest {
		t.Fatalf("inbox error code = %d, want %d", inboxResp.Code, http.StatusBadRequest)
	}

	validationErr := core.NewNetworkValidationError(errors.New("missing session_id"))
	if !errors.Is(validationErr, core.ErrNetworkValidation) {
		t.Fatalf("validationErr = %v, want ErrNetworkValidation", validationErr)
	}
	if got := core.StatusForNetworkError(validationErr); got != http.StatusBadRequest {
		t.Fatalf("StatusForNetworkError(validation) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := core.StatusForNetworkError(network.ErrTargetPeerNotFound); got != http.StatusNotFound {
		t.Fatalf("StatusForNetworkError(not found) = %d, want %d", got, http.StatusNotFound)
	}
	if got := core.StatusForNetworkError(network.ErrInvalidField); got != http.StatusBadRequest {
		t.Fatalf("StatusForNetworkError(invalid field) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestValidationErrorHelpersPreserveInnerErrorChain(t *testing.T) {
	t.Parallel()

	t.Run("ShouldPreserveMemoryValidationCause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("bad memory payload")
		wrapped := core.NewMemoryValidationError(cause)
		if !errors.Is(wrapped, memory.ErrValidation) {
			t.Fatalf("NewMemoryValidationError() = %v, want memory.ErrValidation", wrapped)
		}
		if !errors.Is(wrapped, cause) {
			t.Fatalf("NewMemoryValidationError() = %v, want wrapped cause", wrapped)
		}
	})

	t.Run("ShouldPreserveNetworkValidationCause", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("missing session_id")
		wrapped := core.NewNetworkValidationError(cause)
		if !errors.Is(wrapped, core.ErrNetworkValidation) {
			t.Fatalf("NewNetworkValidationError() = %v, want ErrNetworkValidation", wrapped)
		}
		if !errors.Is(wrapped, cause) {
			t.Fatalf("NewNetworkValidationError() = %v, want wrapped cause", wrapped)
		}
	})
}

func timePtr(value time.Time) *time.Time {
	return &value
}
