package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
)

func TestAgentChannelCoreHandlersUseIdentityAndCoordinationMetadata(t *testing.T) {
	t.Parallel()

	var sent []network.SendRequest
	source := agentCoreEnvelope(t, "msg-source", "builders", contract.CoordinationMessageRequest)
	source.From = "reviewer.sess-peer"
	networkService := &agentCoreNetworkService{
		ListChannelsFn: func(context.Context) ([]network.ChannelInfo, error) {
			return []network.ChannelInfo{{Channel: "builders", PeerCount: 2}}, nil
		},
		SendFn: func(_ context.Context, request network.SendRequest) (string, error) {
			sent = append(sent, request)
			if len(sent) == 1 {
				return "msg-send", nil
			}
			return "msg-reply", nil
		},
		WaitInboxFn: func(_ context.Context, sessionID string, channel string) ([]network.Envelope, error) {
			if sessionID != "sess-agent" || channel != "builders" {
				t.Fatalf("WaitInbox() = %q/%q, want caller session builders", sessionID, channel)
			}
			return []network.Envelope{
				agentCoreEnvelope(t, "msg-other", "other", contract.CoordinationMessageStatus),
				agentCoreEnvelope(t, "msg-builders", "builders", contract.CoordinationMessageResult),
			}, nil
		},
		InboxFn: func(_ context.Context, sessionID string) ([]network.Envelope, error) {
			if sessionID != "sess-agent" {
				t.Fatalf("Inbox() sessionID = %q, want sess-agent", sessionID)
			}
			return []network.Envelope{source}, nil
		},
	}
	engine := newAgentCoreTestRouter(t, networkService)

	contextResp := performAgentCoreRequest(t, engine, http.MethodGet, "/agent/context", nil, agentCoreHeaders())
	if contextResp.Code != http.StatusOK {
		t.Fatalf("context status = %d, want %d; body=%s", contextResp.Code, http.StatusOK, contextResp.Body.String())
	}
	var contextPayload contract.AgentContextResponse
	decodeAgentCoreResponse(t, contextResp, &contextPayload)
	if contextPayload.Context.Self.SessionID != "sess-agent" ||
		contextPayload.Context.Workspace.ID != "ws-1" {
		t.Fatalf("context = %#v, want caller identity", contextPayload.Context)
	}

	channelsResp := performAgentCoreRequest(t, engine, http.MethodGet, "/agent/channels", nil, agentCoreHeaders())
	if channelsResp.Code != http.StatusOK {
		t.Fatalf("channels status = %d, want %d; body=%s", channelsResp.Code, http.StatusOK, channelsResp.Body.String())
	}
	var channels contract.AgentChannelsResponse
	decodeAgentCoreResponse(t, channelsResp, &channels)
	if len(channels.Channels) != 1 ||
		channels.Channels[0].ID != "builders" ||
		len(channels.Channels[0].AllowedMessageKinds) != len(contract.CoordinationMessageKinds()) {
		t.Fatalf("channels = %#v, want builders with MVP kinds", channels.Channels)
	}

	sendResp := performAgentCoreRequest(
		t,
		engine,
		http.MethodPost,
		"/agent/channels/builders/send",
		[]byte(
			`{"body":{"text":"status"},"metadata":{"task_id":"task-1","run_id":"run-1","workflow_id":"wf-1","coordination_channel_id":"builders","message_kind":"status","correlation_id":"run-1"}}`,
		),
		agentCoreHeaders(),
	)
	if sendResp.Code != http.StatusAccepted {
		t.Fatalf("send status = %d, want %d; body=%s", sendResp.Code, http.StatusAccepted, sendResp.Body.String())
	}
	if len(sent) != 1 ||
		sent[0].SessionID != "sess-agent" ||
		sent[0].Channel != "builders" ||
		sent[0].Kind != network.KindSay {
		t.Fatalf("sent requests = %#v, want caller-bound say", sent)
	}
	if metadata := decodeAgentCoreCoordinationExt(
		t,
		sent[0].Ext,
	); metadata.MessageKind != contract.CoordinationMessageStatus ||
		metadata.WorkflowID != "wf-1" {
		t.Fatalf("send metadata = %#v, want status wf-1", metadata)
	}

	recvResp := performAgentCoreRequest(
		t,
		engine,
		http.MethodGet,
		"/agent/channels/builders/recv?wait=true&limit=1",
		nil,
		agentCoreHeaders(),
	)
	if recvResp.Code != http.StatusOK {
		t.Fatalf("recv status = %d, want %d; body=%s", recvResp.Code, http.StatusOK, recvResp.Body.String())
	}
	var messages contract.AgentChannelMessagesResponse
	decodeAgentCoreResponse(t, recvResp, &messages)
	if len(messages.Messages) != 1 ||
		messages.Messages[0].MessageID != "msg-builders" ||
		messages.Messages[0].Metadata.MessageKind != contract.CoordinationMessageResult {
		t.Fatalf("messages = %#v, want builders result message", messages.Messages)
	}

	queuedResp := performAgentCoreRequest(
		t,
		engine,
		http.MethodGet,
		"/agent/channels/builders/recv",
		nil,
		agentCoreHeaders(),
	)
	if queuedResp.Code != http.StatusOK {
		t.Fatalf("queued recv status = %d, want %d; body=%s", queuedResp.Code, http.StatusOK, queuedResp.Body.String())
	}
	var queuedMessages contract.AgentChannelMessagesResponse
	decodeAgentCoreResponse(t, queuedResp, &queuedMessages)
	if len(queuedMessages.Messages) != 1 || queuedMessages.Messages[0].MessageID != "msg-source" {
		t.Fatalf("queued messages = %#v, want source inbox message", queuedMessages.Messages)
	}

	replyResp := performAgentCoreRequest(
		t,
		engine,
		http.MethodPost,
		"/agent/channels/reply",
		[]byte(
			`{"reply_to_message_id":"msg-source","body":{"text":"ack"},"metadata":{"task_id":"","run_id":"","coordination_channel_id":"","message_kind":"","correlation_id":""}}`,
		),
		agentCoreHeaders(),
	)
	if replyResp.Code != http.StatusAccepted {
		t.Fatalf("reply status = %d, want %d; body=%s", replyResp.Code, http.StatusAccepted, replyResp.Body.String())
	}
	if len(sent) != 2 ||
		sent[1].SessionID != "sess-agent" ||
		sent[1].Kind != network.KindDirect ||
		sent[1].To == nil ||
		*sent[1].To != "reviewer.sess-peer" ||
		sent[1].ReplyTo == nil ||
		*sent[1].ReplyTo != "msg-source" {
		t.Fatalf("reply request = %#v, want direct reply to source", sent)
	}
	if metadata := decodeAgentCoreCoordinationExt(
		t,
		sent[1].Ext,
	); metadata.MessageKind != contract.CoordinationMessageReply ||
		metadata.TaskID != "task-1" ||
		metadata.RunID != "run-1" {
		t.Fatalf("reply metadata = %#v, want inherited reply metadata", metadata)
	}

	badReplyKind := performAgentCoreRequest(
		t,
		engine,
		http.MethodPost,
		"/agent/channels/reply",
		[]byte(
			`{"reply_to_message_id":"msg-source","body":{"text":"ack"},"metadata":{"task_id":"task-1","run_id":"run-1","coordination_channel_id":"builders","message_kind":"status","correlation_id":"run-1"}}`,
		),
		agentCoreHeaders(),
	)
	if badReplyKind.Code != http.StatusBadRequest {
		t.Fatalf(
			"bad reply kind status = %d, want %d; body=%s",
			badReplyKind.Code,
			http.StatusBadRequest,
			badReplyKind.Body.String(),
		)
	}
	if len(sent) != 2 {
		t.Fatalf("sent request count after bad reply kind = %d, want unchanged 2", len(sent))
	}
}

func TestAgentMeCoreHandlerEnrichesContextAndChannels(t *testing.T) {
	t.Parallel()

	engine := newAgentCoreTestRouter(t, &agentCoreNetworkService{
		ListChannelsFn: func(context.Context) ([]network.ChannelInfo, error) {
			return []network.ChannelInfo{{Channel: "builders", PeerCount: 1}}, nil
		},
	})

	recorder := performAgentCoreRequest(t, engine, http.MethodGet, "/agent/me", nil, agentCoreHeaders())
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.AgentMeResponse
	decodeAgentCoreResponse(t, recorder, &response)
	if response.Me.Self.SessionID != "sess-agent" ||
		response.Me.Workspace.ID != "ws-1" ||
		len(response.Me.Capabilities) != 1 ||
		len(response.Me.Channels) != 1 ||
		len(response.Me.ActiveTaskLeases) != 1 ||
		response.Me.Limits.MaxChildren != 2 {
		t.Fatalf("agent me = %#v, want context and channel enrichment", response.Me)
	}
}

func TestAgentChannelCoreHandlersRejectInvalidIdentityAndClaimToken(t *testing.T) {
	t.Parallel()

	sendCalled := false
	engine := newAgentCoreTestRouter(t, &agentCoreNetworkService{
		SendFn: func(context.Context, network.SendRequest) (string, error) {
			sendCalled = true
			return "unexpected", nil
		},
	})

	denied := performAgentCoreRequest(
		t,
		engine,
		http.MethodPost,
		"/agent/channels/builders/send",
		[]byte(
			`{"body":{"text":"ok"},"metadata":{"task_id":"task-1","run_id":"run-1","coordination_channel_id":"builders","message_kind":"status","correlation_id":"run-1"}}`,
		),
		map[string]string{},
	)
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("denied status = %d, want %d", denied.Code, http.StatusUnauthorized)
	}

	rawToken := performAgentCoreRequest(
		t,
		engine,
		http.MethodPost,
		"/agent/channels/builders/send",
		[]byte(
			`{"body":{"claim_token":"secret"},"metadata":{"task_id":"task-1","run_id":"run-1","coordination_channel_id":"builders","message_kind":"status","correlation_id":"run-1"}}`,
		),
		agentCoreHeaders(),
	)
	if rawToken.Code != http.StatusBadRequest {
		t.Fatalf(
			"claim_token status = %d, want %d; body=%s",
			rawToken.Code,
			http.StatusBadRequest,
			rawToken.Body.String(),
		)
	}
	if sendCalled {
		t.Fatal("Send should not be called for denied/raw-token requests")
	}
}

type agentCoreNetworkService struct {
	SendFn         func(context.Context, network.SendRequest) (string, error)
	ListChannelsFn func(context.Context) ([]network.ChannelInfo, error)
	InboxFn        func(context.Context, string) ([]network.Envelope, error)
	WaitInboxFn    func(context.Context, string, string) ([]network.Envelope, error)
}

func (s *agentCoreNetworkService) Send(ctx context.Context, req network.SendRequest) (string, error) {
	if s.SendFn != nil {
		return s.SendFn(ctx, req)
	}
	return "", nil
}

func (s *agentCoreNetworkService) ListPeers(context.Context, string) ([]network.PeerInfo, error) {
	return nil, nil
}

func (s *agentCoreNetworkService) ListChannels(ctx context.Context) ([]network.ChannelInfo, error) {
	if s.ListChannelsFn != nil {
		return s.ListChannelsFn(ctx)
	}
	return nil, nil
}

func (s *agentCoreNetworkService) Status(context.Context) (*network.Status, error) {
	return &network.Status{Enabled: true, Status: network.StatusRunning}, nil
}

func (s *agentCoreNetworkService) Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error) {
	if s.InboxFn != nil {
		return s.InboxFn(ctx, sessionID)
	}
	return nil, nil
}

func (s *agentCoreNetworkService) WaitInbox(
	ctx context.Context,
	sessionID string,
	channel string,
) ([]network.Envelope, error) {
	if s.WaitInboxFn != nil {
		return s.WaitInboxFn(ctx, sessionID, channel)
	}
	return s.Inbox(ctx, sessionID)
}

type agentCoreContextService func(context.Context, *session.Info) (contract.AgentContextPayload, error)

func (f agentCoreContextService) ContextForSession(
	ctx context.Context,
	info *session.Info,
) (contract.AgentContextPayload, error) {
	return f(ctx, info)
}

func newAgentCoreTestRouter(t *testing.T, networkService *agentCoreNetworkService) *gin.Engine {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Network.Enabled = true
	handlers := NewBaseHandlers(&BaseHandlerConfig{
		TransportName:       "udsapi",
		Sessions:            agentCoreSessionManager(t),
		Network:             networkService,
		AgentContextService: agentCoreContextService(agentCoreContextPayload),
		Config:              cfg,
		Logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		StreamDone:          make(chan struct{}),
		StartedAt:           time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 26, 10, 0, 1, 0, time.UTC)
		},
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/agent/me", handlers.AgentMe)
	engine.GET("/agent/context", handlers.AgentContext)
	engine.GET("/agent/channels", handlers.AgentChannels)
	engine.GET("/agent/channels/:channel/recv", handlers.AgentChannelRecv)
	engine.POST("/agent/channels/:channel/send", handlers.AgentChannelSend)
	engine.POST("/agent/channels/reply", handlers.AgentChannelReply)
	return engine
}

func agentCoreSessionManager(t *testing.T) sessionManagerStub {
	t.Helper()

	return sessionManagerStub{
		status: func(_ context.Context, id string) (*session.Info, error) {
			if id != "sess-agent" {
				return nil, session.ErrSessionNotFound
			}
			now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			return &session.Info{
				ID:          "sess-agent",
				Name:        "worker",
				AgentName:   "coder",
				Provider:    "test-provider",
				WorkspaceID: "ws-1",
				Workspace:   "/workspace/project",
				Channel:     "builders",
				Type:        session.SessionTypeUser,
				State:       session.StateActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}
}

func agentCoreContextPayload(_ context.Context, info *session.Info) (contract.AgentContextPayload, error) {
	channel := contract.CoordinationChannelPayload{
		ID:          "builders",
		Channel:     "builders",
		DisplayName: "builders",
		WorkspaceID: info.WorkspaceID,
	}
	return contract.AgentContextPayload{
		Self: contract.AgentIdentityPayload{
			SessionID: info.ID,
			AgentName: info.AgentName,
			Provider:  info.Provider,
		},
		Workspace: contract.AgentWorkspacePayload{ID: info.WorkspaceID, RootDir: info.Workspace},
		Session: contract.AgentSessionPayload{
			ID:        info.ID,
			State:     info.State,
			Channel:   info.Channel,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
		},
		Task: contract.AgentTaskContextPayload{
			Available: true,
			Lease: &contract.TaskRunLeaseSummaryPayload{
				TaskID:                "task-1",
				RunID:                 "run-1",
				SessionID:             info.ID,
				CoordinationChannelID: "builders",
				CoordinationChannel:   &channel,
			},
		},
		CoordinationChannel: contract.AgentCoordinationChannelContextPayload{
			Available: true,
			Channel:   &channel,
		},
		InboxSummary: contract.AgentInboxSummaryPayload{},
		PeerRoster:   contract.AgentPeerRosterPayload{},
		Capabilities: contract.AgentCapabilitySectionPayload{
			Capabilities: []contract.AgentCapabilityPayload{{
				ID:     "shell",
				Source: "test",
			}},
		},
		Limits: contract.AgentLimitsPayload{ContextSectionLimit: 20, MaxChildren: 2},
		Provenance: contract.AgentContextProvenancePayload{
			GeneratedAt: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
			Source:      "test",
		},
	}, nil
}

func performAgentCoreRequest(
	t *testing.T,
	engine http.Handler,
	method string,
	path string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func agentCoreHeaders() map[string]string {
	return map[string]string{
		agentidentity.HeaderSessionID:   "sess-agent",
		agentidentity.HeaderAgent:       "coder",
		agentidentity.HeaderWorkspaceID: "ws-1",
	}
}

func decodeAgentCoreResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, recorder.Body.String())
	}
}

func agentCoreEnvelope(
	t *testing.T,
	messageID string,
	channel string,
	kind contract.CoordinationMessageKind,
) network.Envelope {
	t.Helper()

	return network.Envelope{
		Protocol: network.ProtocolV0,
		ID:       messageID,
		Kind:     network.KindSay,
		Channel:  channel,
		From:     "coder.sess-peer",
		TS:       time.Date(2026, 4, 26, 10, 1, 0, 0, time.UTC).Unix(),
		Body:     json.RawMessage(`{"text":"coordination"}`),
		Ext: network.ExtensionMap{
			"coordination": agentCoreCoordinationMetadata(t, kind),
		},
	}
}

func agentCoreCoordinationMetadata(t *testing.T, kind contract.CoordinationMessageKind) json.RawMessage {
	t.Helper()

	content, err := json.Marshal(contract.CoordinationMessageMetadataPayload{
		TaskID:                "task-1",
		RunID:                 "run-1",
		WorkflowID:            "wf-1",
		CoordinationChannelID: "builders",
		MessageKind:           kind,
		CorrelationID:         "run-1",
	})
	if err != nil {
		t.Fatalf("json.Marshal(coordination metadata) error = %v", err)
	}
	return content
}

func decodeAgentCoreCoordinationExt(
	t *testing.T,
	ext network.ExtensionMap,
) contract.CoordinationMessageMetadataPayload {
	t.Helper()

	var metadata contract.CoordinationMessageMetadataPayload
	if err := json.Unmarshal(ext["coordination"], &metadata); err != nil {
		t.Fatalf("json.Unmarshal(coordination ext) error = %v", err)
	}
	return metadata
}
