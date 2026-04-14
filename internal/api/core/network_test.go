package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestNetworkConversionHelpersPreserveMetadata(t *testing.T) {
	t.Parallel()

	deadline := int64(1775823000)

	t.Run("Should map status metadata", func(t *testing.T) {
		t.Parallel()

		status := &network.NetworkStatus{
			Enabled:              true,
			Status:               network.StatusRunning,
			ListenerHost:         "127.0.0.1",
			ListenerPort:         4222,
			LocalPeers:           1,
			RemotePeers:          2,
			Channels:             1,
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
		if payload == nil || payload.Channels != 1 || payload.MessagesDelivered != 3 || len(payload.KindMetrics) != 1 || payload.KindMetrics[0].Kind != string(network.KindSay) {
			t.Fatalf("NetworkStatusPayloadFromStatus() = %#v", payload)
		}
	})

	t.Run("Should convert NetworkSendRequest preserving metadata", func(t *testing.T) {
		t.Parallel()

		req := contract.NetworkSendRequest{
			SessionID:     " sess-a ",
			Channel:       " builders ",
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
		if converted.SessionID != "sess-a" || converted.Channel != "builders" || converted.Kind != network.KindSay {
			t.Fatalf("converted request = %#v", converted)
		}
		if converted.To == nil || *converted.To != "reviewer.sess-b" {
			t.Fatalf("converted.To = %#v, want reviewer.sess-b", converted.To)
		}
		if converted.ExpiresAt == nil || *converted.ExpiresAt != deadline {
			t.Fatalf("converted.ExpiresAt = %#v, want %d", converted.ExpiresAt, deadline)
		}
		if string(converted.Body) != `{"text":"hello"}` {
			t.Fatalf("converted.Body = %s, want preserved JSON", string(converted.Body))
		}
		if string(converted.Ext["agh.workflow_id"]) != `"wf-1"` || string(converted.Ext["agh.handoff_version"]) != `3` {
			t.Fatalf("converted.Ext = %#v, want preserved ext metadata", converted.Ext)
		}
	})

	t.Run("Should convert Envelope preserving metadata", func(t *testing.T) {
		t.Parallel()

		proof := network.Proof{"sig": json.RawMessage(`"abc123"`)}
		traceID := "trace-1"
		causationID := "cause-1"
		replyTo := "msg-root"
		envelope := network.Envelope{
			Protocol:    network.ProtocolV0,
			ID:          "msg-1",
			Kind:        network.KindDirect,
			Channel:     "builders",
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
		if envelopePayload.ExpiresAt == nil || *envelopePayload.ExpiresAt != deadline {
			t.Fatalf("ExpiresAt = %#v, want %d", envelopePayload.ExpiresAt, deadline)
		}
		if string(envelopePayload.Proof["sig"]) != `"abc123"` {
			t.Fatalf("Proof = %#v, want cloned proof payload", envelopePayload.Proof)
		}
		if string(envelopePayload.Ext["agh.handoff_version"]) != `3` {
			t.Fatalf("Ext = %#v, want cloned ext payload", envelopePayload.Ext)
		}
	})
}

func TestBaseHandlersNetworkEndpoints(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true

	fixedNow := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	fixture.Handlers.Network = testutil.StubNetworkService{
		StatusFn: func(context.Context) (*network.NetworkStatus, error) {
			return &network.NetworkStatus{
				Enabled:              true,
				Status:               network.StatusRunning,
				ListenerHost:         "127.0.0.1",
				ListenerPort:         4222,
				LocalPeers:           1,
				RemotePeers:          1,
				Channels:             1,
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
		ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
			if channel != "builders" && channel != "" {
				t.Fatalf("ListPeers() channel = %q, want builders or empty", channel)
			}
			displayName := "Reviewer"
			sessionID := "sess-a"
			peers := []network.PeerInfo{{
				SessionID: &sessionID,
				PeerID:    "reviewer.sess-a",
				Channel:   "builders",
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
			}}
			if channel == "" {
				remoteDisplayName := "Coder"
				peers = append(peers, network.PeerInfo{
					PeerID:  "coder.sess-remote",
					Channel: "builders",
					Local:   false,
					PeerCard: network.PeerCard{
						PeerID:      "coder.sess-remote",
						DisplayName: &remoteDisplayName,
					},
					LastSeen:  timePtr(fixedNow),
					ExpiresAt: timePtr(fixedNow.Add(time.Minute)),
				})
			}
			return peers, nil
		},
		ListChannelsFn: func(context.Context) ([]network.ChannelInfo, error) {
			return []network.ChannelInfo{{Channel: "builders", PeerCount: 2}}, nil
		},
		SendFn: func(_ context.Context, req network.SendRequest) (string, error) {
			if req.SessionID != "sess-a" || req.Channel != "builders" || req.Kind != network.KindSay {
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
				Channel:  "builders",
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
		if statusPayload.Network.Channels != 1 || statusPayload.Network.QueuedMessages != 2 || len(statusPayload.Network.KindMetrics) != 1 {
			t.Fatalf("status payload = %#v", statusPayload.Network)
		}
		if statusPayload.Network.KindMetrics[0].Sent != 4 || statusPayload.Network.KindMetrics[0].Kind != string(network.KindSay) {
			t.Fatalf("kind metrics = %#v", statusPayload.Network.KindMetrics)
		}
	})

	t.Run("ShouldListNetworkPeers", func(t *testing.T) {
		peersResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/peers?channel=builders", nil)
		if peersResp.Code != http.StatusOK {
			t.Fatalf("peers code = %d, want %d", peersResp.Code, http.StatusOK)
		}

		var peersPayload contract.NetworkPeersResponse
		testutil.DecodeJSONResponse(t, peersResp, &peersPayload)
		if len(peersPayload.Peers) != 1 || peersPayload.Peers[0].PeerCard.DisplayName == nil || *peersPayload.Peers[0].PeerCard.DisplayName != "Reviewer" {
			t.Fatalf("peers payload = %#v", peersPayload.Peers)
		}
	})

	t.Run("ShouldListNetworkChannels", func(t *testing.T) {
		channelsResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels", nil)
		if channelsResp.Code != http.StatusOK {
			t.Fatalf("channels code = %d, want %d", channelsResp.Code, http.StatusOK)
		}

		var channelsPayload contract.NetworkChannelsResponse
		testutil.DecodeJSONResponse(t, channelsResp, &channelsPayload)
		if len(channelsPayload.Channels) != 1 || channelsPayload.Channels[0].Channel != "builders" || channelsPayload.Channels[0].PeerCount != 2 {
			t.Fatalf("channels payload = %#v", channelsPayload.Channels)
		}
	})

	t.Run("ShouldSendNetworkMessages", func(t *testing.T) {
		sendResp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{"session_id":"sess-a","channel":"builders","kind":"say","body":{"text":"hello"},"ext":{"agh.workflow_id":"wf-1","agh.handoff_version":3}}`))
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

	t.Run("ShouldReturnDisabledStatus", func(t *testing.T) {
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
	})

	t.Run("ShouldReturnServiceUnavailableWhenNetworkServiceMissing", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/peers", nil)
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("peers unavailable code = %d, want %d", resp.Code, http.StatusServiceUnavailable)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, "network service is required") {
			t.Fatalf("peers unavailable payload = %#v, want network service error", payload)
		}
	})

	t.Run("ShouldMapNetworkStatusErrorTo500", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			StatusFn: func(context.Context) (*network.NetworkStatus, error) {
				return nil, errors.New("boom")
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/status", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("status error code = %d, want %d", resp.Code, http.StatusInternalServerError)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, "boom") {
			t.Fatalf("status error payload = %#v, want boom", payload)
		}
	})

	t.Run("ShouldMapListChannelsErrorTo400", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			ListPeersFn: func(context.Context, string) ([]network.PeerInfo, error) {
				return nil, network.ErrInvalidField
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("channels error code = %d, want %d", resp.Code, http.StatusBadRequest)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, network.ErrInvalidField.Error()) {
			t.Fatalf("channels error payload = %#v, want invalid field", payload)
		}
	})

	t.Run("ShouldReturnBadRequestOnSendDecode", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{}

		resp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("send decode code = %d, want %d", resp.Code, http.StatusBadRequest)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, "decode network send request") {
			t.Fatalf("send decode payload = %#v, want decode error", payload)
		}
	})

	t.Run("ShouldMapSendTargetNotFoundTo404", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			SendFn: func(context.Context, network.SendRequest) (string, error) {
				return "", network.ErrTargetPeerNotFound
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodPost, "/network/send", []byte(`{"session_id":"sess-a","channel":"builders","kind":"say","body":{"text":"hello"}}`))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("send error code = %d, want %d", resp.Code, http.StatusNotFound)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, network.ErrTargetPeerNotFound.Error()) {
			t.Fatalf("send error payload = %#v, want target not found", payload)
		}
	})

	t.Run("ShouldReturnBadRequestWhenInboxMissing", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/inbox", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("inbox missing code = %d, want %d", resp.Code, http.StatusBadRequest)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, "session_id query is required") {
			t.Fatalf("inbox missing payload = %#v, want session_id validation error", payload)
		}
	})

	t.Run("ShouldMapInboxInvalidFieldTo400", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			InboxFn: func(context.Context, string) ([]network.Envelope, error) {
				return nil, network.ErrInvalidField
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/inbox?session_id=sess-a", nil)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("inbox error code = %d, want %d", resp.Code, http.StatusBadRequest)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, network.ErrInvalidField.Error()) {
			t.Fatalf("inbox error payload = %#v, want invalid field", payload)
		}
	})

	t.Run("ShouldPreserveNetworkErrorStatusMappings", func(t *testing.T) {
		t.Parallel()

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
	})
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

func TestBaseHandlersNetworkChannelEndpointsIgnoreStoppedSessions(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	coderSessionID := "sess-coder"
	reviewerSessionID := "sess-reviewer"

	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{
				{
					ID:          coderSessionID,
					Name:        "Coder",
					AgentName:   "coder",
					WorkspaceID: "ws-1",
					Channel:     "builders",
					Type:        session.SessionTypeUser,
					State:       session.StateActive,
					CreatedAt:   createdAt,
					UpdatedAt:   createdAt,
				},
				{
					ID:          reviewerSessionID,
					Name:        "Reviewer",
					AgentName:   "reviewer",
					WorkspaceID: "ws-1",
					Channel:     "retro",
					Type:        session.SessionTypeUser,
					State:       session.StateStopped,
					CreatedAt:   createdAt.Add(time.Minute),
					UpdatedAt:   createdAt.Add(time.Minute),
				},
			}, nil
		},
	}

	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true
	fixture.Handlers.Network = testutil.StubNetworkService{
		ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
			switch channel {
			case "":
				return []network.PeerInfo{
					{
						SessionID: &coderSessionID,
						PeerID:    "coder.sess-coder",
						Channel:   "builders",
						Local:     true,
						PeerCard:  network.PeerCard{PeerID: "coder.sess-coder"},
						JoinedAt:  timePtr(createdAt),
						LastSeen:  timePtr(createdAt),
					},
				}, nil
			case "builders":
				return []network.PeerInfo{
					{
						SessionID: &coderSessionID,
						PeerID:    "coder.sess-coder",
						Channel:   "builders",
						Local:     true,
						PeerCard:  network.PeerCard{PeerID: "coder.sess-coder"},
						JoinedAt:  timePtr(createdAt),
						LastSeen:  timePtr(createdAt),
					},
				}, nil
			case "retro":
				return nil, nil
			default:
				t.Fatalf("unexpected ListPeers() channel %q", channel)
				return nil, nil
			}
		},
	}
	fixture.Handlers.NetworkStore = testutil.StubNetworkStore{
		ListNetworkAuditFn: func(_ context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error) {
			switch query.Channel {
			case "builders":
				return []store.NetworkAuditEntry{{
					ID:        "naud-builders-01",
					SessionID: coderSessionID,
					Direction: network.AuditDirectionSent,
					Kind:      "say",
					Channel:   "builders",
					PeerFrom:  "coder.sess-coder",
					MessageID: "msg-builders-01",
					Size:      1,
					Timestamp: createdAt.Add(2 * time.Minute),
				}}, nil
			case "retro":
				return []store.NetworkAuditEntry{{
					ID:        "naud-retro-01",
					SessionID: reviewerSessionID,
					Direction: network.AuditDirectionSent,
					Kind:      "say",
					Channel:   "retro",
					PeerFrom:  "reviewer.sess-reviewer",
					MessageID: "msg-retro-01",
					Size:      1,
					Timestamp: createdAt.Add(3 * time.Minute),
				}}, nil
			default:
				return nil, nil
			}
		},
		ListNetworkMessagesFn: func(_ context.Context, query store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error) {
			switch query.Channel {
			case "":
				return []store.NetworkMessageEntry{
					{
						MessageID: "msg-builders-01",
						SessionID: coderSessionID,
						Channel:   "builders",
						PeerFrom:  "coder.sess-coder",
						Kind:      "say",
						Intent:    "announce",
						Text:      "hello builders",
						Timestamp: createdAt.Add(2 * time.Minute),
					},
					{
						MessageID: "msg-retro-01",
						SessionID: reviewerSessionID,
						Channel:   "retro",
						PeerFrom:  "reviewer.sess-reviewer",
						Kind:      "say",
						Text:      "retro note",
						Timestamp: createdAt.Add(3 * time.Minute),
					},
				}, nil
			case "builders":
				return []store.NetworkMessageEntry{{
					MessageID: "msg-builders-01",
					SessionID: coderSessionID,
					Channel:   "builders",
					PeerFrom:  "coder.sess-coder",
					Kind:      "say",
					Intent:    "announce",
					Text:      "hello builders",
					Timestamp: createdAt.Add(2 * time.Minute),
				}}, nil
			case "retro":
				return []store.NetworkMessageEntry{{
					MessageID: "msg-retro-01",
					SessionID: reviewerSessionID,
					Channel:   "retro",
					PeerFrom:  "reviewer.sess-reviewer",
					Kind:      "say",
					Text:      "retro note",
					Timestamp: createdAt.Add(3 * time.Minute),
				}}, nil
			default:
				return nil, nil
			}
		},
	}

	channelsResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels", nil)
	if channelsResp.Code != http.StatusOK {
		t.Fatalf("channels code = %d, want %d", channelsResp.Code, http.StatusOK)
	}

	var channelsPayload contract.NetworkChannelsResponse
	testutil.DecodeJSONResponse(t, channelsResp, &channelsPayload)
	if got, want := len(channelsPayload.Channels), 1; got != want {
		t.Fatalf("len(channels) = %d, want %d", got, want)
	}
	sort.Slice(channelsPayload.Channels, func(i, j int) bool {
		return channelsPayload.Channels[i].Channel < channelsPayload.Channels[j].Channel
	})
	if got, want := channelsPayload.Channels[0].Channel, "builders"; got != want {
		t.Fatalf("channels[0].Channel = %q, want %q", got, want)
	}
	if got, want := channelsPayload.Channels[0].SessionCount, 1; got != want {
		t.Fatalf("channels[0].SessionCount = %d, want %d", got, want)
	}

	channelResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels/builders", nil)
	if channelResp.Code != http.StatusOK {
		t.Fatalf("channel detail code = %d, want %d", channelResp.Code, http.StatusOK)
	}

	var channelPayload contract.NetworkChannelResponse
	testutil.DecodeJSONResponse(t, channelResp, &channelPayload)
	if got, want := channelPayload.Channel.Channel, "builders"; got != want {
		t.Fatalf("channel detail channel = %q, want %q", got, want)
	}
	if got, want := channelPayload.Channel.Peers[0].DisplayName, "Coder"; got != want {
		t.Fatalf("channel detail peer display = %q, want %q", got, want)
	}
	if got, want := channelPayload.Channel.MessageCount, 1; got != want {
		t.Fatalf("channel detail message count = %d, want %d", got, want)
	}

	messagesResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels/builders/messages", nil)
	if messagesResp.Code != http.StatusOK {
		t.Fatalf("channel messages code = %d, want %d", messagesResp.Code, http.StatusOK)
	}

	var messagesPayload contract.NetworkChannelMessagesResponse
	testutil.DecodeJSONResponse(t, messagesResp, &messagesPayload)
	if got, want := len(messagesPayload.Messages), 1; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}
	if got, want := messagesPayload.Messages[0].DisplayName, "Coder"; got != want {
		t.Fatalf("message display_name = %q, want %q", got, want)
	}
	if !messagesPayload.Messages[0].Local {
		t.Fatal("message local = false, want true")
	}

	ghostResp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels/retro", nil)
	if ghostResp.Code != http.StatusNotFound {
		t.Fatalf("ghost channel detail code = %d, want %d", ghostResp.Code, http.StatusNotFound)
	}
}

func TestBaseHandlersNetworkChannelMessagesPreserveRemoteAuthors(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	localSessionID := "sess-coder"
	remotePeerID := "reviewer.sess-remote"

	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{{
				ID:          localSessionID,
				Name:        "Coder",
				AgentName:   "coder",
				WorkspaceID: "ws-1",
				Channel:     "builders",
				Type:        session.SessionTypeUser,
				State:       session.StateActive,
				CreatedAt:   createdAt,
				UpdatedAt:   createdAt,
			}}, nil
		},
	}

	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true
	fixture.Handlers.Network = testutil.StubNetworkService{
		ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
			if channel != "builders" {
				t.Fatalf("ListPeers() channel = %q, want builders", channel)
			}
			displayName := "Reviewer"
			return []network.PeerInfo{
				{
					SessionID: &localSessionID,
					PeerID:    "coder.sess-coder",
					Channel:   "builders",
					Local:     true,
					PeerCard:  network.PeerCard{PeerID: "coder.sess-coder"},
				},
				{
					PeerID:  remotePeerID,
					Channel: "builders",
					Local:   false,
					PeerCard: network.PeerCard{
						PeerID:      remotePeerID,
						DisplayName: &displayName,
					},
				},
			}, nil
		},
	}
	fixture.Handlers.NetworkStore = testutil.StubNetworkStore{
		ListNetworkAuditFn: func(_ context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error) {
			if got, want := query.Channel, "builders"; got != want {
				t.Fatalf("ListNetworkAudit() channel = %q, want %q", got, want)
			}
			return []store.NetworkAuditEntry{
				{
					ID:        "naud-1",
					SessionID: localSessionID,
					Direction: network.AuditDirectionReceived,
					Kind:      "say",
					Channel:   "builders",
					PeerFrom:  remotePeerID,
					MessageID: "msg-remote-01",
					Size:      1,
					Timestamp: createdAt.Add(time.Minute),
				},
				{
					ID:        "naud-2",
					SessionID: localSessionID,
					Direction: network.AuditDirectionDelivered,
					Kind:      "say",
					Channel:   "builders",
					PeerFrom:  remotePeerID,
					MessageID: "msg-remote-01",
					Size:      1,
					Timestamp: createdAt.Add(2 * time.Minute),
				},
				{
					ID:        "naud-3",
					SessionID: localSessionID,
					Direction: network.AuditDirectionSent,
					Kind:      "say",
					Channel:   "builders",
					PeerFrom:  "coder.sess-coder",
					MessageID: "msg-local-01",
					Size:      1,
					Timestamp: createdAt.Add(3 * time.Minute),
				},
			}, nil
		},
		ListNetworkMessagesFn: func(_ context.Context, query store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error) {
			if got, want := query.Channel, "builders"; got != want {
				t.Fatalf("ListNetworkMessages() channel = %q, want %q", got, want)
			}
			return []store.NetworkMessageEntry{
				{
					MessageID: "msg-remote-01",
					SessionID: localSessionID,
					Channel:   "builders",
					PeerFrom:  remotePeerID,
					Kind:      "say",
					Intent:    "review",
					Text:      "Please double-check the rollout.",
					Timestamp: createdAt.Add(time.Minute),
				},
				{
					MessageID: "msg-local-01",
					SessionID: localSessionID,
					Channel:   "builders",
					PeerFrom:  "coder.sess-coder",
					Kind:      "say",
					Intent:    "announce",
					Text:      "Starting rollout now.",
					Timestamp: createdAt.Add(3 * time.Minute),
				},
			}, nil
		},
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/channels/builders/messages", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("channel messages code = %d, want %d", resp.Code, http.StatusOK)
	}

	var payload contract.NetworkChannelMessagesResponse
	testutil.DecodeJSONResponse(t, resp, &payload)
	if got, want := len(payload.Messages), 2; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}

	if got, want := payload.Messages[0].DisplayName, "Reviewer"; got != want {
		t.Fatalf("remote display_name = %q, want %q", got, want)
	}
	if payload.Messages[0].Local {
		t.Fatal("remote message local = true, want false")
	}
	if got := payload.Messages[0].SessionID; got != "" {
		t.Fatalf("remote session_id = %q, want empty", got)
	}

	if got, want := payload.Messages[1].DisplayName, "Coder"; got != want {
		t.Fatalf("local display_name = %q, want %q", got, want)
	}
	if !payload.Messages[1].Local {
		t.Fatal("local message local = false, want true")
	}
	if got, want := payload.Messages[1].SessionID, localSessionID; got != want {
		t.Fatalf("local session_id = %q, want %q", got, want)
	}
}

func TestBaseHandlersCreateNetworkChannelCreatesSessionsPerAgent(t *testing.T) {
	t.Parallel()

	var createCalls []session.CreateOpts
	manager := testutil.StubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			createCalls = append(createCalls, opts)
			return &session.Session{
				ID:          "sess-" + opts.AgentName,
				Name:        strings.ToUpper(opts.AgentName),
				AgentName:   opts.AgentName,
				WorkspaceID: opts.Workspace,
				Channel:     opts.Channel,
				Type:        session.SessionTypeUser,
				State:       session.StateActive,
				CreatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
			}, nil
		},
		ListAllFn: func(_ context.Context) ([]*session.SessionInfo, error) {
			infos := make([]*session.SessionInfo, 0, len(createCalls))
			for _, call := range createCalls {
				infos = append(infos, &session.SessionInfo{
					ID:          "sess-" + call.AgentName,
					Name:        strings.ToUpper(call.AgentName),
					AgentName:   call.AgentName,
					WorkspaceID: call.Workspace,
					Channel:     call.Channel,
					Type:        session.SessionTypeUser,
					State:       session.StateActive,
					CreatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
				})
			}
			return infos, nil
		},
	}
	workspaces := testutil.StubWorkspaceService{
		ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
			if ref != "ws-1" {
				t.Fatalf("Resolve() ref = %q, want ws-1", ref)
			}
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{ID: "ws-1", Name: "Workspace"},
				Agents: []aghconfig.AgentDef{
					{Name: "coder"},
					{Name: "reviewer"},
				},
			}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, workspaces, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true
	fixture.Handlers.Network = testutil.StubNetworkService{
		ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
			if channel != "builders" {
				return nil, nil
			}
			coderSessionID := "sess-coder"
			reviewerSessionID := "sess-reviewer"
			return []network.PeerInfo{
				{
					SessionID: &coderSessionID,
					PeerID:    "coder.sess-coder",
					Channel:   "builders",
					Local:     true,
					PeerCard:  network.PeerCard{PeerID: "coder.sess-coder"},
				},
				{
					SessionID: &reviewerSessionID,
					PeerID:    "reviewer.sess-reviewer",
					Channel:   "builders",
					Local:     true,
					PeerCard:  network.PeerCard{PeerID: "reviewer.sess-reviewer"},
				},
			}, nil
		},
	}
	fixture.Handlers.NetworkStore = testutil.StubNetworkStore{
		ListNetworkMessagesFn: func(_ context.Context, query store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error) {
			if query.Channel != "builders" {
				return nil, nil
			}
			return nil, nil
		},
	}

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/network/channels",
		[]byte(`{"channel":"builders","workspace_id":"ws-1","agent_names":["coder","reviewer"]}`),
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create channel code = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	if got, want := len(createCalls), 2; got != want {
		t.Fatalf("len(createCalls) = %d, want %d", got, want)
	}
	for _, call := range createCalls {
		if got, want := call.Workspace, "ws-1"; got != want {
			t.Fatalf("Create() workspace = %q, want %q", got, want)
		}
		if got, want := call.Channel, "builders"; got != want {
			t.Fatalf("Create() channel = %q, want %q", got, want)
		}
	}

	var payload contract.CreateNetworkChannelResponse
	testutil.DecodeJSONResponse(t, resp, &payload)
	if got, want := payload.Channel.SessionCount, 2; got != want {
		t.Fatalf("payload.Channel.SessionCount = %d, want %d", got, want)
	}
}

func TestBaseHandlersNetworkPeerDetailUsesAuditMetrics(t *testing.T) {
	t.Parallel()

	coderSessionID := "sess-coder"
	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{{
				ID:          coderSessionID,
				Name:        "Coder",
				AgentName:   "coder",
				WorkspaceID: "ws-1",
				Channel:     "builders",
				Type:        session.SessionTypeUser,
				State:       session.StateActive,
				CreatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true
	fixture.Handlers.Network = testutil.StubNetworkService{
		ListPeersFn: func(_ context.Context, channel string) ([]network.PeerInfo, error) {
			if channel != "" {
				t.Fatalf("ListPeers() channel = %q, want empty filter", channel)
			}
			return []network.PeerInfo{{
				SessionID: &coderSessionID,
				PeerID:    "coder.sess-coder",
				Channel:   "builders",
				Local:     true,
				PeerCard:  network.PeerCard{PeerID: "coder.sess-coder"},
			}}, nil
		},
	}
	fixture.Handlers.NetworkStore = testutil.StubNetworkStore{
		ListNetworkAuditFn: func(_ context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error) {
			if query.SessionID != coderSessionID {
				t.Fatalf("ListNetworkAudit() session_id = %q, want %q", query.SessionID, coderSessionID)
			}
			return []store.NetworkAuditEntry{
				{SessionID: coderSessionID, Direction: network.AuditDirectionSent, Kind: "say", Channel: "builders", PeerFrom: "coder.sess-coder", MessageID: "msg-1", Size: 1},
				{SessionID: coderSessionID, Direction: network.AuditDirectionReceived, Kind: "direct", Channel: "builders", PeerFrom: "reviewer.sess-remote", MessageID: "msg-2", Size: 1},
				{SessionID: coderSessionID, Direction: network.AuditDirectionDelivered, Kind: "say", Channel: "builders", PeerFrom: "coder.sess-coder", MessageID: "msg-1", Size: 1},
				{SessionID: coderSessionID, Direction: network.AuditDirectionRejected, Kind: "receipt", Channel: "builders", PeerFrom: "reviewer.sess-remote", MessageID: "msg-3", Reason: "busy", Size: 1},
			}, nil
		},
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/network/peers/coder.sess-coder", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("peer detail code = %d, want %d", resp.Code, http.StatusOK)
	}

	var payload contract.NetworkPeerResponse
	testutil.DecodeJSONResponse(t, resp, &payload)
	if got, want := payload.Peer.DisplayName, "Coder"; got != want {
		t.Fatalf("payload.Peer.DisplayName = %q, want %q", got, want)
	}
	if got, want := payload.Peer.Metrics.Sent, int64(1); got != want {
		t.Fatalf("payload.Peer.Metrics.Sent = %d, want %d", got, want)
	}
	if got, want := payload.Peer.Metrics.Received, int64(1); got != want {
		t.Fatalf("payload.Peer.Metrics.Received = %d, want %d", got, want)
	}
	if got, want := payload.Peer.Metrics.Delivered, int64(1); got != want {
		t.Fatalf("payload.Peer.Metrics.Delivered = %d, want %d", got, want)
	}
	if got, want := payload.Peer.Metrics.Rejected, int64(1); got != want {
		t.Fatalf("payload.Peer.Metrics.Rejected = %d, want %d", got, want)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
