package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	toolspkg "github.com/pedronauck/agh/internal/tools"
)

func TestToolHandlersExposeOperatorSessionInvokeAndToolsets(t *testing.T) {
	t.Parallel()

	t.Run("Should expose operator session invoke and toolset handlers", func(t *testing.T) {
		t.Parallel()

		registry := newAPITestToolRegistry(t, false)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:      "api-core-test",
			Sessions:           testutil.StubSessionManager{},
			Observer:           testutil.StubObserver{},
			Tasks:              testutil.StubTaskManager{},
			Workspaces:         testutil.StubWorkspaceService{},
			Tools:              registry,
			Toolsets:           registry,
			ToolApprovals:      toolspkg.NewApprovalTokenStore(time.Minute),
			HomePaths:          testutil.NewTestHomePaths(t),
			Config:             testutil.ConfigWithDisabledNetwork(testutil.NewTestHomePaths(t)),
			Logger:             testutil.DiscardLogger(),
			StartedAt:          time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
			Now:                func() time.Time { return time.Date(2026, 4, 29, 12, 0, 1, 0, time.UTC) },
			PollInterval:       time.Millisecond,
			StreamDone:         make(chan struct{}),
			MaskInternalErrors: false,
		})
		engine := newToolCoreEngine(t, handlers)

		listResp := performRequest(t, engine, http.MethodGet, "/tools", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf(
				"operator list status = %d, want %d; body=%s",
				listResp.Code,
				http.StatusOK,
				listResp.Body.String(),
			)
		}
		var list contract.ToolsResponse
		decodeToolJSON(t, listResp.Body.Bytes(), &list)
		if got, want := len(list.Tools), 2; got != want {
			t.Fatalf("operator tool count = %d, want %d", got, want)
		}

		sessionResp := performRequest(t, engine, http.MethodGet, "/sessions/sess-1/tools", nil)
		if sessionResp.Code != http.StatusOK {
			t.Fatalf("session list status = %d, want %d", sessionResp.Code, http.StatusOK)
		}
		var sessionTools contract.ToolsResponse
		decodeToolJSON(t, sessionResp.Body.Bytes(), &sessionTools)
		if got, want := len(sessionTools.Tools), 1; got != want {
			t.Fatalf("session tool count = %d, want %d", got, want)
		}
		if sessionTools.Tools[0].Descriptor.ToolID != toolspkg.ToolIDSkillView {
			t.Fatalf("session tool = %q, want %q", sessionTools.Tools[0].Descriptor.ToolID, toolspkg.ToolIDSkillView)
		}

		searchResp := performRequest(t, engine, http.MethodPost, "/tools/search", []byte(`{"query":"skill","limit":1}`))
		if searchResp.Code != http.StatusOK {
			t.Fatalf("search status = %d, want %d", searchResp.Code, http.StatusOK)
		}
		var search contract.ToolsResponse
		decodeToolJSON(t, searchResp.Body.Bytes(), &search)
		if got, want := len(search.Tools), 1; got != want {
			t.Fatalf("search tool count = %d, want %d", got, want)
		}

		sessionSearchResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/sessions/sess-1/tools/search",
			[]byte(`{"query":"skill","limit":1,"workspace_id":"ws-1","session_id":"sess-other"}`),
		)
		if sessionSearchResp.Code != http.StatusOK {
			t.Fatalf("session search status = %d, want %d", sessionSearchResp.Code, http.StatusOK)
		}
		var sessionSearch contract.ToolsResponse
		decodeToolJSON(t, sessionSearchResp.Body.Bytes(), &sessionSearch)
		if got, want := len(sessionSearch.Tools), 1; got != want {
			t.Fatalf("session search tool count = %d, want %d", got, want)
		}
		searchScope, searchQuery := registry.lastSearch()
		if searchScope.SessionID != "sess-1" || searchScope.WorkspaceID != "ws-1" || searchScope.Operator {
			t.Fatalf("session search scope = %#v, want session workspace scope", searchScope)
		}
		if searchQuery.Query != "skill" || searchQuery.Limit != 1 {
			t.Fatalf("session search query = %#v, want skill limit", searchQuery)
		}

		getResp := performRequest(t, engine, http.MethodGet, "/tools/agh__skill_view", nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get status = %d, want %d", getResp.Code, http.StatusOK)
		}
		var gotTool contract.ToolResponse
		decodeToolJSON(t, getResp.Body.Bytes(), &gotTool)
		if gotTool.Tool.Descriptor.Backend.Kind != toolspkg.BackendNativeGo {
			t.Fatalf("backend kind = %q, want %q", gotTool.Tool.Descriptor.Backend.Kind, toolspkg.BackendNativeGo)
		}

		invokeResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/tools/agh__skill_view/invoke",
			[]byte(`{"session_id":"sess-1","workspace_id":"ws-1","input":{"message":"hello"}}`),
		)
		if invokeResp.Code != http.StatusOK {
			t.Fatalf("invoke status = %d, want %d; body=%s", invokeResp.Code, http.StatusOK, invokeResp.Body.String())
		}
		var invoke contract.ToolInvokeResponse
		decodeToolJSON(t, invokeResp.Body.Bytes(), &invoke)
		if invoke.ToolID != toolspkg.ToolIDSkillView || invoke.Status != "completed" {
			t.Fatalf("invoke response = %#v, want completed skill_view", invoke)
		}
		if registry.callCount(toolspkg.ToolIDSkillView) != 1 {
			t.Fatalf("registry call count = %d, want 1", registry.callCount(toolspkg.ToolIDSkillView))
		}

		toolsetsResp := performRequest(t, engine, http.MethodGet, "/toolsets", nil)
		if toolsetsResp.Code != http.StatusOK {
			t.Fatalf("toolsets status = %d, want %d", toolsetsResp.Code, http.StatusOK)
		}
		var toolsets contract.ToolsetsResponse
		decodeToolJSON(t, toolsetsResp.Body.Bytes(), &toolsets)
		if got, want := len(toolsets.Toolsets), 1; got != want {
			t.Fatalf("toolset count = %d, want %d", got, want)
		}
		if toolsets.Toolsets[0].Status != "expanded" ||
			len(toolsets.Toolsets[0].ExpandedTools) != 1 ||
			toolsets.Toolsets[0].ExpandedTools[0] != toolspkg.ToolIDSkillView {
			t.Fatalf("toolset payload = %#v, want expanded skill_view", toolsets.Toolsets[0])
		}

		toolsetResp := performRequest(t, engine, http.MethodGet, "/toolsets/agh__catalog", nil)
		if toolsetResp.Code != http.StatusOK {
			t.Fatalf("toolset get status = %d, want %d", toolsetResp.Code, http.StatusOK)
		}
		var toolset contract.ToolsetResponse
		decodeToolJSON(t, toolsetResp.Body.Bytes(), &toolset)
		if toolset.Toolset.ID != toolspkg.ToolsetIDCatalog || registry.lastToolsetID() != toolspkg.ToolsetIDCatalog {
			t.Fatalf("toolset get = %#v id=%q, want catalog", toolset.Toolset, registry.lastToolsetID())
		}
	})
}

func TestToolApprovalHandlersMintAndConsumeSingleUseTokens(t *testing.T) {
	t.Parallel()

	t.Run("Should mint and consume single-use tokens", func(t *testing.T) {
		t.Parallel()

		approvals := toolspkg.NewApprovalTokenStore(
			time.Minute,
			toolspkg.WithApprovalTokenClock(func() time.Time {
				return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
			}),
		)
		registry := newAPITestToolRegistry(t, true, approvals)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName: "api-core-test",
			Sessions:      testutil.StubSessionManager{},
			Observer:      testutil.StubObserver{},
			Tasks:         testutil.StubTaskManager{},
			Workspaces:    testutil.StubWorkspaceService{},
			Tools:         registry,
			Toolsets:      registry,
			ToolApprovals: approvals,
			HomePaths:     testutil.NewTestHomePaths(t),
			Config:        testutil.ConfigWithDisabledNetwork(testutil.NewTestHomePaths(t)),
			Logger:        testutil.DiscardLogger(),
			StartedAt:     time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
			Now:           func() time.Time { return time.Date(2026, 4, 29, 12, 0, 1, 0, time.UTC) },
			PollInterval:  time.Millisecond,
			StreamDone:    make(chan struct{}),
		})
		engine := newToolCoreEngine(t, handlers)

		missingTokenResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/tools/ext__ask_tool/invoke",
			[]byte(`{"session_id":"sess-1","workspace_id":"ws-1","input":{"message":"hello"}}`),
		)
		if missingTokenResp.Code != http.StatusAccepted {
			t.Fatalf(
				"missing token status = %d, want %d; body=%s",
				missingTokenResp.Code,
				http.StatusAccepted,
				missingTokenResp.Body.String(),
			)
		}
		var missingToken contract.ToolErrorResponse
		decodeToolJSON(t, missingTokenResp.Body.Bytes(), &missingToken)
		if missingToken.Error.Code != toolspkg.ErrorCodeApprovalRequired ||
			!containsReason(missingToken.Error.ReasonCodes, toolspkg.ReasonApprovalTokenMissing) {
			t.Fatalf("missing token error = %#v, want approval token missing", missingToken.Error)
		}
		if registry.callCount("ext__ask_tool") != 0 {
			t.Fatal("approval-required tool executed without token")
		}

		approvalResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/tools/ext__ask_tool/approvals",
			[]byte(`{"session_id":"sess-1","workspace_id":"ws-1","input":{"message":"hello"}}`),
		)
		if approvalResp.Code != http.StatusCreated {
			t.Fatalf(
				"approval status = %d, want %d; body=%s",
				approvalResp.Code,
				http.StatusCreated,
				approvalResp.Body.String(),
			)
		}
		var approval contract.ToolApprovalResponse
		decodeToolJSON(t, approvalResp.Body.Bytes(), &approval)
		if approval.Approval.ApprovalToken == "" || approval.Approval.InputDigest == "" {
			t.Fatalf("approval response = %#v, want token and digest", approval)
		}

		body := []byte(`{"session_id":"sess-1","workspace_id":"ws-1","approval_token":"` +
			approval.Approval.ApprovalToken + `","input":{"message":"hello"}}`)
		invokeResp := performRequest(t, engine, http.MethodPost, "/tools/ext__ask_tool/invoke", body)
		if invokeResp.Code != http.StatusOK {
			t.Fatalf("invoke status = %d, want %d; body=%s", invokeResp.Code, http.StatusOK, invokeResp.Body.String())
		}
		var invokePayload contract.ToolInvokeResponse
		decodeToolJSON(t, invokeResp.Body.Bytes(), &invokePayload)
		if invokePayload.ToolID != "ext__ask_tool" || invokePayload.Status != "completed" {
			t.Fatalf("invoke payload = %#v, want completed ext__ask_tool", invokePayload)
		}
		if registry.callCount("ext__ask_tool") != 1 {
			t.Fatalf("registry call count = %d, want 1", registry.callCount("ext__ask_tool"))
		}

		replayResp := performRequest(t, engine, http.MethodPost, "/tools/ext__ask_tool/invoke", body)
		if replayResp.Code != http.StatusForbidden {
			t.Fatalf(
				"replay status = %d, want %d; body=%s",
				replayResp.Code,
				http.StatusForbidden,
				replayResp.Body.String(),
			)
		}
		var replay contract.ToolErrorResponse
		decodeToolJSON(t, replayResp.Body.Bytes(), &replay)
		if !containsReason(replay.Error.ReasonCodes, toolspkg.ReasonApprovalTokenReplayed) {
			t.Fatalf("replay error = %#v, want replay reason", replay.Error)
		}
	})
}

func TestToolHandlersPropagateScopeDefaultsAndSanitizeErrors(t *testing.T) {
	t.Parallel()

	t.Run("Should propagate query scope defaults into invoke requests", func(t *testing.T) {
		t.Parallel()

		registry := newAPITestToolRegistry(t, false)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:      "api-core-test",
			Sessions:           testutil.StubSessionManager{},
			Observer:           testutil.StubObserver{},
			Tasks:              testutil.StubTaskManager{},
			Workspaces:         testutil.StubWorkspaceService{},
			Tools:              registry,
			Toolsets:           registry,
			ToolApprovals:      toolspkg.NewApprovalTokenStore(time.Minute),
			HomePaths:          testutil.NewTestHomePaths(t),
			Config:             testutil.ConfigWithDisabledNetwork(testutil.NewTestHomePaths(t)),
			Logger:             testutil.DiscardLogger(),
			StartedAt:          time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
			Now:                func() time.Time { return time.Date(2026, 4, 29, 12, 0, 1, 0, time.UTC) },
			PollInterval:       time.Millisecond,
			StreamDone:         make(chan struct{}),
			MaskInternalErrors: false,
		})
		engine := newToolCoreEngine(t, handlers)

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/tools/agh__skill_view/invoke?session_id=sess-query&workspace_id=ws-query&agent_name=coder",
			[]byte(`{"input":{"message":"hello"}}`),
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("invoke status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
		var invokePayload contract.ToolInvokeResponse
		decodeToolJSON(t, resp.Body.Bytes(), &invokePayload)
		if invokePayload.ToolID != toolspkg.ToolIDSkillView || invokePayload.Status != "completed" {
			t.Fatalf("invoke payload = %#v, want completed skill_view", invokePayload)
		}
		scope, call := registry.lastCall()
		if scope.SessionID != "sess-query" || scope.WorkspaceID != "ws-query" || scope.AgentName != "coder" {
			t.Fatalf("call scope = %#v, want query defaults", scope)
		}
		if call.SessionID != "sess-query" || call.WorkspaceID != "ws-query" || call.AgentName != "coder" {
			t.Fatalf("call request = %#v, want normalized scope values", call)
		}
	})

	t.Run("Should return safe tool error messages", func(t *testing.T) {
		t.Parallel()

		registry := newAPITestToolRegistry(t, true)
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:      "api-core-test",
			Sessions:           testutil.StubSessionManager{},
			Observer:           testutil.StubObserver{},
			Tasks:              testutil.StubTaskManager{},
			Workspaces:         testutil.StubWorkspaceService{},
			Tools:              registry,
			Toolsets:           registry,
			ToolApprovals:      toolspkg.NewApprovalTokenStore(time.Minute),
			HomePaths:          testutil.NewTestHomePaths(t),
			Config:             testutil.ConfigWithDisabledNetwork(testutil.NewTestHomePaths(t)),
			Logger:             testutil.DiscardLogger(),
			StartedAt:          time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC),
			Now:                func() time.Time { return time.Date(2026, 4, 29, 12, 0, 1, 0, time.UTC) },
			PollInterval:       time.Millisecond,
			StreamDone:         make(chan struct{}),
			MaskInternalErrors: false,
		})
		engine := newToolCoreEngine(t, handlers)

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/tools/ext__ask_tool/invoke",
			[]byte(`{"session_id":"sess-1","workspace_id":"ws-1","input":{"token":"agh_claim_secret"}}`),
		)
		if resp.Code != http.StatusAccepted {
			t.Fatalf("invoke status = %d, want %d; body=%s", resp.Code, http.StatusAccepted, resp.Body.String())
		}
		var payload contract.ToolErrorResponse
		decodeToolJSON(t, resp.Body.Bytes(), &payload)
		if payload.Error.Message != "tool approval required" {
			t.Fatalf("tool error message = %q, want safe approval message", payload.Error.Message)
		}
		if strings.Contains(resp.Body.String(), "agh_claim_secret") ||
			strings.Contains(resp.Body.String(), "tool approval token is required") {
			t.Fatalf("tool error leaked raw backend detail: %s", resp.Body.String())
		}
	})
}

func newToolCoreEngine(t *testing.T, handlers *core.BaseHandlers) *gin.Engine {
	t.Helper()
	engine := gin.New()
	engine.GET("/tools", handlers.ListTools)
	engine.POST("/tools/search", handlers.SearchTools)
	engine.GET("/tools/:id", handlers.GetTool)
	engine.POST("/tools/:id/approvals", handlers.CreateToolApproval)
	engine.POST("/tools/:id/invoke", handlers.InvokeTool)
	engine.GET("/sessions/:id/tools", handlers.ListSessionTools)
	engine.POST("/sessions/:id/tools/search", handlers.SearchSessionTools)
	engine.GET("/toolsets", handlers.ListToolsets)
	engine.GET("/toolsets/:id", handlers.GetToolset)
	return engine
}

type apiTestToolRegistry struct {
	registry        *toolspkg.RuntimeRegistry
	mu              sync.Mutex
	calls           map[toolspkg.ToolID]int
	lastCallScope   toolspkg.Scope
	lastCallRequest toolspkg.CallRequest
	lastSearchScope toolspkg.Scope
	lastSearchQuery toolspkg.SearchQuery
	lastToolset     toolspkg.ToolsetID
}

func newAPITestToolRegistry(
	t *testing.T,
	approvalRequired bool,
	approvalConsumers ...toolspkg.ApprovalTokenConsumer,
) *apiTestToolRegistry {
	t.Helper()
	ids := []toolspkg.ToolID{toolspkg.ToolIDSkillView}
	source := toolspkg.SourceRef{Kind: toolspkg.SourceBuiltin, Owner: "agh"}
	descriptors := []toolspkg.Descriptor{
		testToolDescriptor(toolspkg.ToolIDSkillView, source, toolspkg.VisibilityModel),
		testToolDescriptor("agh__operator_diag", source, toolspkg.VisibilityOperator),
	}
	inputs := toolspkg.DefaultPolicyInputs()
	if approvalRequired {
		source = toolspkg.SourceRef{Kind: toolspkg.SourceExtension, Owner: "ext"}
		descriptors = []toolspkg.Descriptor{
			testToolDescriptor("ext__ask_tool", source, toolspkg.VisibilityModel),
		}
		ids = []toolspkg.ToolID{"ext__ask_tool"}
		inputs.ExternalDefault = toolspkg.ExternalDefaultAsk
		inputs.ApprovalAvailable = true
	}
	catalog, err := toolspkg.NewToolsetCatalog(toolspkg.Toolset{
		ID:    "agh__catalog",
		Tools: []string{string(ids[0])},
	})
	if err != nil {
		t.Fatalf("NewToolsetCatalog() error = %v", err)
	}
	wrapper := &apiTestToolRegistry{calls: make(map[toolspkg.ToolID]int)}
	provider := &apiTestToolProvider{
		source:  source,
		handles: make(map[toolspkg.ToolID]*apiTestToolHandle),
	}
	for _, descriptor := range descriptors {
		provider.handles[descriptor.ID] = &apiTestToolHandle{
			descriptor: descriptor,
			call: func(_ context.Context, req toolspkg.CallRequest) (toolspkg.ToolResult, error) {
				wrapper.mu.Lock()
				wrapper.calls[req.ToolID]++
				wrapper.mu.Unlock()
				return toolspkg.ToolResult{
					Content:    []toolspkg.ToolContent{{Type: "text", Text: "ok"}},
					Structured: json.RawMessage(`{"ok":true}`),
					DurationMS: 12,
				}, nil
			},
		}
	}
	options := []toolspkg.RegistryOption{
		toolspkg.WithProviders(provider),
		toolspkg.WithPolicyInputs(inputs, catalog),
	}
	if approvalRequired {
		var consumer toolspkg.ApprovalTokenConsumer
		if len(approvalConsumers) > 0 {
			consumer = approvalConsumers[0]
		}
		options = append(options, toolspkg.WithApprovalBridge(apiTestApprovalBridge{approvals: consumer}))
	}
	registry, err := toolspkg.NewRegistry(options...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	wrapper.registry = registry
	return wrapper
}

func (r *apiTestToolRegistry) List(ctx context.Context, scope toolspkg.Scope) ([]toolspkg.ToolView, error) {
	return r.registry.List(ctx, scope)
}

func (r *apiTestToolRegistry) Search(
	ctx context.Context,
	scope toolspkg.Scope,
	q toolspkg.SearchQuery,
) ([]toolspkg.ToolView, error) {
	r.mu.Lock()
	r.lastSearchScope = scope
	r.lastSearchQuery = q
	r.mu.Unlock()
	return r.registry.Search(ctx, scope, q)
}

func (r *apiTestToolRegistry) Get(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.ToolView, error) {
	return r.registry.Get(ctx, scope, id)
}

func (r *apiTestToolRegistry) Call(
	ctx context.Context,
	scope toolspkg.Scope,
	req toolspkg.CallRequest,
) (toolspkg.ToolResult, error) {
	r.mu.Lock()
	r.lastCallScope = scope
	r.lastCallRequest = req
	r.mu.Unlock()
	return r.registry.Call(ctx, scope, req)
}

func (r *apiTestToolRegistry) ListToolsets(
	ctx context.Context,
	scope toolspkg.Scope,
) ([]toolspkg.ToolsetView, error) {
	return r.registry.ListToolsets(ctx, scope)
}

func (r *apiTestToolRegistry) GetToolset(
	ctx context.Context,
	scope toolspkg.Scope,
	id toolspkg.ToolsetID,
) (toolspkg.ToolsetView, error) {
	r.mu.Lock()
	r.lastToolset = id
	r.mu.Unlock()
	return r.registry.GetToolset(ctx, scope, id)
}

func (r *apiTestToolRegistry) callCount(id toolspkg.ToolID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[id]
}

func (r *apiTestToolRegistry) lastCall() (toolspkg.Scope, toolspkg.CallRequest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastCallScope, r.lastCallRequest
}

func (r *apiTestToolRegistry) lastSearch() (toolspkg.Scope, toolspkg.SearchQuery) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastSearchScope, r.lastSearchQuery
}

func (r *apiTestToolRegistry) lastToolsetID() toolspkg.ToolsetID {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastToolset
}

type apiTestToolProvider struct {
	source  toolspkg.SourceRef
	handles map[toolspkg.ToolID]*apiTestToolHandle
}

func (p *apiTestToolProvider) ID() toolspkg.SourceRef {
	return p.source
}

func (p *apiTestToolProvider) List(context.Context, toolspkg.Scope) ([]toolspkg.Descriptor, error) {
	descriptors := make([]toolspkg.Descriptor, 0, len(p.handles))
	for _, handle := range p.handles {
		descriptors = append(descriptors, handle.Descriptor())
	}
	return descriptors, nil
}

func (p *apiTestToolProvider) Resolve(
	_ context.Context,
	_ toolspkg.Scope,
	id toolspkg.ToolID,
) (toolspkg.Handle, bool, error) {
	handle, ok := p.handles[id]
	return handle, ok, nil
}

type apiTestToolHandle struct {
	descriptor toolspkg.Descriptor
	call       func(context.Context, toolspkg.CallRequest) (toolspkg.ToolResult, error)
}

func (h *apiTestToolHandle) Descriptor() toolspkg.Descriptor {
	return h.descriptor
}

func (h *apiTestToolHandle) Availability(context.Context, toolspkg.Scope) toolspkg.Availability {
	return toolspkg.Availability{
		Registered: true,
		Enabled:    true,
		Available:  true,
		Authorized: true,
		Executable: true,
	}
}

func (h *apiTestToolHandle) Call(ctx context.Context, req toolspkg.CallRequest) (toolspkg.ToolResult, error) {
	if h.call == nil {
		return toolspkg.ToolResult{}, errors.New("test handle call not configured")
	}
	return h.call(ctx, req)
}

type apiTestApprovalBridge struct {
	approvals toolspkg.ApprovalTokenConsumer
}

func (b apiTestApprovalBridge) RequestToolApproval(
	ctx context.Context,
	scope toolspkg.Scope,
	call toolspkg.CallRequest,
	_ *toolspkg.ToolView,
) error {
	if b.approvals == nil {
		return toolspkg.NewToolError(
			toolspkg.ErrorCodeApprovalRequired,
			call.ToolID,
			"tool approval token is required",
			toolspkg.ErrToolApprovalRequired,
			toolspkg.ReasonApprovalRequired,
			toolspkg.ReasonApprovalTokenMissing,
		)
	}
	return b.approvals.ConsumeToolApproval(ctx, scope, call)
}

func testToolDescriptor(
	id toolspkg.ToolID,
	source toolspkg.SourceRef,
	visibility toolspkg.Visibility,
) toolspkg.Descriptor {
	return toolspkg.Descriptor{
		ID:           id,
		Backend:      toolspkg.BackendRef{Kind: toolspkg.BackendNativeGo, NativeName: id.String()},
		DisplayTitle: id.String(),
		Description:  "Test tool " + id.String(),
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		Source:       source,
		Visibility:   visibility,
		Risk:         toolspkg.RiskRead,
		ReadOnly:     true,
		Toolsets:     []toolspkg.ToolsetID{"agh__catalog"},
		Tags:         []string{"skill", "test"},
	}
}

func decodeToolJSON(t *testing.T, data []byte, dest any) {
	t.Helper()
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", data, err)
	}
}

func containsReason(reasons []toolspkg.ReasonCode, want toolspkg.ReasonCode) bool {
	return slices.Contains(reasons, want)
}
