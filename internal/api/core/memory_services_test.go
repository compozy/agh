package core_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
)

func TestMemoryExtractorHandlersUseInjectedService(t *testing.T) {
	t.Parallel()

	extractor := &stubMemoryExtractorService{
		status: contract.MemoryExtractorStatusPayload{
			Status:           contract.MemoryExtractorStateRunning,
			QueuedSessions:   2,
			InFlightSessions: 1,
			DroppedTurns:     3,
			CoalescedTurns:   4,
			FailureCount:     1,
		},
		failures: []contract.MemoryExtractorFailurePayload{{
			ID:        "failure-1",
			SessionID: "sess-1",
			Reason:    "decode",
			Path:      "/tmp/failure.json",
			CreatedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
		}},
		retry: contract.MemoryExtractorRetryResponse{Retried: 1},
		drain: contract.MemoryExtractorDrainResponse{
			DrainedAt: time.Date(2026, 5, 5, 12, 1, 0, 0, time.UTC),
		},
	}
	engine := newMemoryServiceRouter(t, &core.BaseHandlerConfig{MemoryExtractor: extractor})

	statusResp := performRequest(t, engine, http.MethodGet, "/memory/extractor/status", nil)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", statusResp.Code, http.StatusOK)
	}
	var statusPayload contract.MemoryExtractorStatusResponse
	decodeJSON(t, statusResp.Body.Bytes(), &statusPayload)
	if got := statusPayload.Extractor.Status; got != contract.MemoryExtractorStateRunning {
		t.Fatalf("Extractor.Status = %q, want running", got)
	}
	if got := statusPayload.Extractor.FailureCount; got != 1 {
		t.Fatalf("Extractor.FailureCount = %d, want 1", got)
	}

	failuresResp := performRequest(t, engine, http.MethodGet, "/memory/extractor/failures", nil)
	var failuresPayload contract.MemoryExtractorFailuresResponse
	decodeJSON(t, failuresResp.Body.Bytes(), &failuresPayload)
	if len(failuresPayload.Failures) != 1 || failuresPayload.Failures[0].ID != "failure-1" {
		t.Fatalf("Failures = %#v, want failure-1", failuresPayload.Failures)
	}

	retryResp := performRequest(
		t,
		engine,
		http.MethodPost,
		"/memory/extractor/retry",
		[]byte(`{"failure_id":"failure-1"}`),
	)
	var retryPayload contract.MemoryExtractorRetryResponse
	decodeJSON(t, retryResp.Body.Bytes(), &retryPayload)
	if retryPayload.Retried != 1 || extractor.retryReq.FailureID != "failure-1" {
		t.Fatalf("Retry response=%#v request=%#v, want failure-1 retried", retryPayload, extractor.retryReq)
	}

	drainResp := performRequest(t, engine, http.MethodPost, "/memory/extractor/drain", nil)
	var drainPayload contract.MemoryExtractorDrainResponse
	decodeJSON(t, drainResp.Body.Bytes(), &drainPayload)
	if drainPayload.DrainedAt.IsZero() || !extractor.drainCalled {
		t.Fatalf("Drain payload=%#v called=%t, want daemon service drain", drainPayload, extractor.drainCalled)
	}
}

func TestMemoryProviderHandlersUseInjectedService(t *testing.T) {
	t.Parallel()

	providers := &stubMemoryProviderService{
		provider: contract.MemoryProviderPayload{
			Name:    "local",
			Status:  contract.MemoryProviderStateActive,
			Active:  true,
			Builtin: true,
		},
	}
	engine := newMemoryServiceRouter(t, &core.BaseHandlerConfig{MemoryProviders: providers})

	listResp := performRequest(t, engine, http.MethodGet, "/memory/providers?workspace_id=ws-1", nil)
	var listPayload contract.MemoryProviderListResponse
	decodeJSON(t, listResp.Body.Bytes(), &listPayload)
	if len(listPayload.Providers) != 1 || listPayload.Providers[0].Name != "local" {
		t.Fatalf("Providers = %#v, want local", listPayload.Providers)
	}
	if providers.workspaceID != "ws-1" {
		t.Fatalf("workspaceID = %q, want ws-1", providers.workspaceID)
	}

	selectResp := performRequest(
		t,
		engine,
		http.MethodPost,
		"/memory/providers/select?workspace_id=ws-2",
		[]byte(`{"name":"local"}`),
	)
	var selectPayload contract.MemoryProviderResponse
	decodeJSON(t, selectResp.Body.Bytes(), &selectPayload)
	if selectPayload.Provider.Name != "local" || providers.selectedName != "local" || providers.workspaceID != "ws-2" {
		t.Fatalf(
			"select payload=%#v selected=%q workspace=%q, want local/ws-2",
			selectPayload,
			providers.selectedName,
			providers.workspaceID,
		)
	}
}

func TestMemorySessionLedgerHandlersUseInjectedService(t *testing.T) {
	t.Parallel()

	ledger := &stubMemorySessionLedgerService{
		response: contract.MemorySessionLedgerResponse{
			Meta: contract.MemorySessionLedgerMetaPayload{
				Version:   1,
				SessionID: "sess-1",
				Path:      "/tmp/sess-1/ledger.jsonl",
				Checksum:  "abc123",
				CreatedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
			},
			Events: []contract.MemorySessionLedgerEntryPayload{{
				Sequence:  1,
				EventType: "agent_message",
				EmittedAt: time.Date(2026, 5, 5, 12, 0, 1, 0, time.UTC),
			}},
		},
		replay: contract.MemorySessionReplayResponse{
			SessionID: "sess-1",
			Events: []contract.MemorySessionLedgerEntryPayload{{
				Sequence:  1,
				EventType: "agent_message",
			}},
		},
	}
	engine := newMemoryServiceRouter(t, &core.BaseHandlerConfig{MemorySessionLedger: ledger})

	getResp := performRequest(t, engine, http.MethodGet, "/memory/sessions/sess-1/ledger", nil)
	var getPayload contract.MemorySessionLedgerResponse
	decodeJSON(t, getResp.Body.Bytes(), &getPayload)
	if getPayload.Meta.SessionID != "sess-1" || ledger.sessionID != "sess-1" {
		t.Fatalf("ledger payload=%#v session=%q, want sess-1", getPayload.Meta, ledger.sessionID)
	}

	replayResp := performRequest(
		t,
		engine,
		http.MethodPost,
		"/memory/sessions/sess-1/replay",
		[]byte(`{"include_tool_events":true}`),
	)
	var replayPayload contract.MemorySessionReplayResponse
	decodeJSON(t, replayResp.Body.Bytes(), &replayPayload)
	if replayPayload.SessionID != "sess-1" || !ledger.replayReq.IncludeToolEvents {
		t.Fatalf("replay payload=%#v request=%#v, want include tool events", replayPayload, ledger.replayReq)
	}
}

func newMemoryServiceRouter(t *testing.T, cfg *core.BaseHandlerConfig) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	runtimeConfig := testConfigWithDisabledNetwork(homePaths)
	cfg.HomePaths = homePaths
	cfg.Config = runtimeConfig
	cfg.Logger = testutil.DiscardLogger()
	cfg.Now = func() time.Time {
		return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	}
	handlers := core.NewBaseHandlers(cfg)
	engine := gin.New()
	engine.GET("/memory/extractor/status", handlers.GetMemoryExtractorStatus)
	engine.GET("/memory/extractor/failures", handlers.ListMemoryExtractorFailures)
	engine.POST("/memory/extractor/retry", handlers.RetryMemoryExtractor)
	engine.POST("/memory/extractor/drain", handlers.DrainMemoryExtractor)
	engine.GET("/memory/providers", handlers.ListMemoryProviders)
	engine.POST("/memory/providers/select", handlers.SelectMemoryProvider)
	engine.GET("/memory/sessions/:session_id/ledger", handlers.GetMemorySessionLedger)
	engine.POST("/memory/sessions/:session_id/replay", handlers.ReplayMemorySession)
	return engine
}

type stubMemoryExtractorService struct {
	status      contract.MemoryExtractorStatusPayload
	failures    []contract.MemoryExtractorFailurePayload
	retry       contract.MemoryExtractorRetryResponse
	drain       contract.MemoryExtractorDrainResponse
	retryReq    contract.MemoryExtractorRetryRequest
	drainCalled bool
}

func (s *stubMemoryExtractorService) Status(context.Context) (contract.MemoryExtractorStatusPayload, error) {
	return s.status, nil
}

func (s *stubMemoryExtractorService) ListFailures(
	context.Context,
) ([]contract.MemoryExtractorFailurePayload, error) {
	return s.failures, nil
}

func (s *stubMemoryExtractorService) Retry(
	_ context.Context,
	req contract.MemoryExtractorRetryRequest,
) (contract.MemoryExtractorRetryResponse, error) {
	s.retryReq = req
	return s.retry, nil
}

func (s *stubMemoryExtractorService) Drain(context.Context) (contract.MemoryExtractorDrainResponse, error) {
	s.drainCalled = true
	return s.drain, nil
}

type stubMemoryProviderService struct {
	provider     contract.MemoryProviderPayload
	workspaceID  string
	selectedName string
}

func (s *stubMemoryProviderService) List(
	_ context.Context,
	workspaceID string,
) ([]contract.MemoryProviderPayload, error) {
	s.workspaceID = workspaceID
	return []contract.MemoryProviderPayload{s.provider}, nil
}

func (s *stubMemoryProviderService) Get(
	_ context.Context,
	workspaceID string,
	_ string,
) (contract.MemoryProviderPayload, error) {
	s.workspaceID = workspaceID
	return s.provider, nil
}

func (s *stubMemoryProviderService) Select(
	_ context.Context,
	workspaceID string,
	name string,
) (contract.MemoryProviderPayload, error) {
	s.workspaceID = workspaceID
	s.selectedName = name
	return s.provider, nil
}

func (s *stubMemoryProviderService) Enable(
	_ context.Context,
	workspaceID string,
	name string,
	_ string,
) (contract.MemoryProviderLifecycleResponse, error) {
	s.workspaceID = workspaceID
	s.selectedName = name
	return contract.MemoryProviderLifecycleResponse{Provider: s.provider, Changed: true}, nil
}

func (s *stubMemoryProviderService) Disable(
	_ context.Context,
	workspaceID string,
	name string,
	_ string,
) (contract.MemoryProviderLifecycleResponse, error) {
	s.workspaceID = workspaceID
	s.selectedName = name
	return contract.MemoryProviderLifecycleResponse{Provider: s.provider, Changed: true}, nil
}

type stubMemorySessionLedgerService struct {
	response  contract.MemorySessionLedgerResponse
	replay    contract.MemorySessionReplayResponse
	sessionID string
	replayReq contract.MemorySessionReplayRequest
}

func (s *stubMemorySessionLedgerService) Get(
	_ context.Context,
	sessionID string,
) (contract.MemorySessionLedgerResponse, error) {
	s.sessionID = sessionID
	return s.response, nil
}

func (s *stubMemorySessionLedgerService) Replay(
	_ context.Context,
	sessionID string,
	req contract.MemorySessionReplayRequest,
) (contract.MemorySessionReplayResponse, error) {
	s.sessionID = sessionID
	s.replayReq = req
	return s.replay, nil
}

func (s *stubMemorySessionLedgerService) Prune(
	context.Context,
	contract.MemorySessionsPruneRequest,
) (contract.MemorySessionsPruneResponse, error) {
	return contract.MemorySessionsPruneResponse{}, nil
}

func (s *stubMemorySessionLedgerService) Repair(context.Context) (contract.MemorySessionsRepairResponse, error) {
	return contract.MemorySessionsRepairResponse{}, nil
}
