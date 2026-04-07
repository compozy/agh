package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestRegisterRoutesCoversTechSpecEndpoints(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	routes := engine.Routes()
	got := make([]string, 0, len(routes))
	for _, route := range routes {
		got = append(got, route.Method+" "+route.Path)
	}
	sort.Strings(got)

	want := []string{
		"DELETE /api/memory/:filename",
		"DELETE /api/sessions/:id",
		"DELETE /api/workspaces/:id",
		"GET /api/agents",
		"GET /api/agents/:name",
		"GET /api/daemon/status",
		"GET /api/memory",
		"GET /api/memory/:filename",
		"GET /api/observe/events",
		"GET /api/observe/events/stream",
		"GET /api/observe/health",
		"GET /api/sessions",
		"GET /api/sessions/:id",
		"GET /api/sessions/:id/events",
		"GET /api/sessions/:id/history",
		"GET /api/sessions/:id/transcript",
		"GET /api/sessions/:id/stream",
		"GET /api/workspaces",
		"GET /api/workspaces/:id",
		"PATCH /api/workspaces/:id",
		"POST /api/memory/consolidate",
		"POST /api/sessions",
		"POST /api/sessions/:id/approve",
		"POST /api/sessions/:id/prompt",
		"POST /api/sessions/:id/resume",
		"POST /api/workspaces",
		"POST /api/workspaces/resolve",
		"PUT /api/memory/:filename",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("len(routes) = %d, want %d\nroutes=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("route[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCreateSessionHandlerReturnsSessionID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "coder" || opts.Name != "demo" || opts.Workspace != "alpha" || opts.WorkspacePath != "" {
				t.Fatalf("Create() opts = %#v", opts)
			}
			return newSession("sess-123"), nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder","name":"demo","workspace":"alpha"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Session sessionPayload `json:"session"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Session.ID != "sess-123" {
		t.Fatalf("session.id = %q, want %q", response.Session.ID, "sess-123")
	}
	if response.Session.WorkspaceID != "ws-workspace" || response.Session.WorkspacePath != "/workspace" {
		t.Fatalf("session workspace = %#v", response.Session)
	}
}

func TestCreateSessionHandlerAllowsMissingAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "" {
				t.Fatalf("Create() AgentName = %q, want empty", opts.AgentName)
			}
			if opts.WorkspacePath == "" || opts.Workspace != "" {
				t.Fatalf("Create() workspace opts = %#v", opts)
			}
			return newSession("sess-default"), nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"name":"demo","workspace_path":"/workspace"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
}

func TestListSessionsHandlerReturnsAllSessions(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-a"), newSessionInfo("sess-b")}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(response.Sessions))
	}
}

func TestListSessionsHandlerFiltersByWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	infoA := newSessionInfo("sess-a")
	infoB := newSessionInfo("sess-b")
	infoB.WorkspaceID = "ws-beta"
	infoB.Workspace = "/other"

	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{infoA, infoB}, nil
		},
	}
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("Get() ref = %q, want alpha", ref)
			}
			return workspacepkg.Workspace{ID: "ws-workspace", Name: "alpha"}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions?workspace=alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 1 || response.Sessions[0].ID != "sess-a" {
		t.Fatalf("sessions = %#v", response.Sessions)
	}
	if response.Sessions[0].WorkspaceID != "ws-workspace" {
		t.Fatalf("workspace_id = %q, want ws-workspace", response.Sessions[0].WorkspaceID)
	}
}

func TestCreateWorkspaceHandlerRegistersWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	workspaces := stubWorkspaceService{
		RegisterFn: func(_ context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
			if opts.RootDir != rootDir || opts.Name != "alpha" || len(opts.AdditionalDirs) != 1 || opts.AdditionalDirs[0] != addDir || opts.DefaultAgent != "coder" {
				t.Fatalf("Register() opts = %#v", opts)
			}
			return workspacepkg.Workspace{
				ID:             "ws_alpha123",
				RootDir:        rootDir,
				AdditionalDirs: []string{addDir},
				Name:           "alpha",
				DefaultAgent:   "coder",
				CreatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	body, err := json.Marshal(map[string]any{
		"root_dir":      rootDir,
		"name":          "alpha",
		"add_dirs":      []string{addDir},
		"default_agent": "coder",
	})
	if err != nil {
		t.Fatalf("json.Marshal(create workspace request) error = %v", err)
	}
	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha123" {
		t.Fatalf("workspace.id = %q, want ws_alpha123", response.Workspace.ID)
	}
}

func TestListWorkspacesHandlerReturnsRegisteredRows(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	workspaces := stubWorkspaceService{
		ListFn: func(context.Context) ([]workspacepkg.Workspace, error) {
			return []workspacepkg.Workspace{{
				ID:        "ws_alpha",
				RootDir:   rootDir,
				Name:      "alpha",
				CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC),
			}}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspaces []workspacePayload `json:"workspaces"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Workspaces) != 1 || response.Workspaces[0].ID != "ws_alpha" {
		t.Fatalf("workspaces = %#v", response.Workspaces)
	}
}

func TestGetWorkspaceHandlerReturnsDetail(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	sharedSkillDir := filepath.Join(rootDir, ".agh", "skills", "review")
	resolved := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:        "ws_alpha",
			RootDir:   rootDir,
			Name:      "alpha",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Agents: []aghconfig.AgentDef{{
			Name:     "coder",
			Provider: "fake",
			Prompt:   "hello",
		}},
		Skills: []workspacepkg.SkillPath{{
			Dir:    sharedSkillDir,
			Source: "workspace",
		}},
	}
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			info := newSessionInfo("sess-a")
			info.WorkspaceID = "ws_alpha"
			return []*session.SessionInfo{info}, nil
		},
	}
	workspaces := stubWorkspaceService{
		ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
			if ref != "ws_alpha" {
				t.Fatalf("Resolve() ref = %q, want ws_alpha", ref)
			}
			return resolved, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload        `json:"workspace"`
		Sessions  []sessionPayload        `json:"sessions"`
		Agents    []agentPayload          `json:"agents"`
		Skills    []workspaceSkillPayload `json:"skills"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" || len(response.Sessions) != 1 || len(response.Agents) != 1 || len(response.Skills) != 1 {
		t.Fatalf("workspace detail = %#v", response)
	}
	if response.Skills[0].Name != "review" {
		t.Fatalf("skill name = %q, want review", response.Skills[0].Name)
	}
}

func TestUpdateWorkspaceHandlerUpdatesWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	var updated bool
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if !updated {
				return workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "alpha"}, nil
			}
			return workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "beta", AdditionalDirs: []string{addDir}, DefaultAgent: "reviewer"}, nil
		},
		UpdateFn: func(_ context.Context, id string, opts workspacepkg.UpdateOptions) error {
			if id != "ws_alpha" || opts.Name == nil || *opts.Name != "beta" || opts.AdditionalDirs == nil || len(*opts.AdditionalDirs) != 1 || (*opts.AdditionalDirs)[0] != addDir || opts.DefaultAgent == nil || *opts.DefaultAgent != "reviewer" {
				t.Fatalf("Update() id=%q opts=%#v", id, opts)
			}
			updated = true
			return nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	body, err := json.Marshal(map[string]any{
		"name":          "beta",
		"add_dirs":      []string{addDir},
		"default_agent": "reviewer",
	})
	if err != nil {
		t.Fatalf("json.Marshal(update workspace request) error = %v", err)
	}
	recorder := performRequest(t, engine, http.MethodPatch, "/api/workspaces/ws_alpha", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.Name != "beta" || len(response.Workspace.AddDirs) != 1 {
		t.Fatalf("workspace = %#v", response.Workspace)
	}
}

func TestDeleteWorkspaceHandlerReturnsNoContent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	workspaces := stubWorkspaceService{
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{ID: "ws_alpha", Name: "alpha"}, nil
		},
		UnregisterFn: func(_ context.Context, id string) error {
			if id != "ws_alpha" {
				t.Fatalf("Unregister() id = %q, want ws_alpha", id)
			}
			return nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodDelete, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
}

func TestResolveWorkspaceHandlerReturnsWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	workspaces := stubWorkspaceService{
		ResolveOrRegisterFn: func(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
			if path != rootDir {
				t.Fatalf("ResolveOrRegister() path = %q, want %q", path, rootDir)
			}
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:        "ws_alpha",
					RootDir:   rootDir,
					Name:      "alpha",
					CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	body, err := json.Marshal(map[string]any{"path": rootDir})
	if err != nil {
		t.Fatalf("json.Marshal(resolve workspace request) error = %v", err)
	}
	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces/resolve", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" {
		t.Fatalf("workspace.id = %q, want ws_alpha", response.Workspace.ID)
	}
}

func TestStopSessionHandlerReturnsStopped(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StopFn: func(_ context.Context, id string) error {
			if id != "sess-123" {
				t.Fatalf("Stop() id = %q, want sess-123", id)
			}
			return nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodDelete, "/api/sessions/sess-123", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestPromptSessionHandlerReturnsAISDKSSEStream(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		PromptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
			ch := make(chan acp.AgentEvent, 4)
			ch <- acp.AgentEvent{
				Type:      "agent_message",
				TurnID:    "turn-1",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				Text:      "hello",
			}
			ch <- acp.AgentEvent{
				Type:       "tool_call",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				Title:      "read_file",
				ToolCallID: "call-1",
			}
			ch <- acp.AgentEvent{
				Type:       "tool_result",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC),
				ToolCallID: "call-1",
			}
			ch <- acp.AgentEvent{
				Type:       "done",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 3, 0, time.UTC),
				StopReason: "end_turn",
			}
			close(ch)
			return ch, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"messages":[{"role":"user","parts":[{"type":"text","text":"hello"}]}]}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := recorder.Header().Get("x-vercel-ai-ui-message-stream"); got != "v1" {
		t.Fatalf("x-vercel-ai-ui-message-stream = %q, want v1", got)
	}

	records := parseSSE(t, recorder.Body.String())
	if len(records) < 5 {
		t.Fatalf("len(records) = %d, want at least 5; body=%s", len(records), recorder.Body.String())
	}
	if string(records[len(records)-1].Data) != "[DONE]" {
		t.Fatalf("last data = %q, want [DONE]", string(records[len(records)-1].Data))
	}

	var foundAgentMessage bool
	var foundToolCall bool
	var foundDone bool
	for _, record := range records {
		if record.Event == "agent_message" {
			foundAgentMessage = true
		}
		if record.Event == "tool_call" {
			foundToolCall = true
		}
		if record.Event == "done" {
			foundDone = true
		}
	}
	if !foundAgentMessage || !foundToolCall || !foundDone {
		t.Fatalf("events missing native markers: agent_message=%v tool_call=%v done=%v body=%s", foundAgentMessage, foundToolCall, foundDone, recorder.Body.String())
	}

	var promptParts []map[string]any
	for _, record := range records[:len(records)-1] {
		if len(record.Data) == 0 || string(record.Data) == "[DONE]" {
			continue
		}
		var part map[string]any
		if err := json.Unmarshal(record.Data, &part); err != nil {
			t.Fatalf("json.Unmarshal(part) error = %v; data=%s", err, string(record.Data))
		}
		promptParts = append(promptParts, part)
	}

	types := make([]string, 0, len(promptParts))
	for _, part := range promptParts {
		if value, ok := part["type"].(string); ok {
			types = append(types, value)
		}
	}
	if !contains(types, "start") || !contains(types, "text-start") || !contains(types, "text-delta") || !contains(types, "tool-input-start") || !contains(types, "tool-output-available") || !contains(types, "finish") {
		t.Fatalf("part types = %#v", types)
	}
}

func TestSessionEventsAndHistoryHandlers(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		EventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
			gotQuery = query
			return []store.SessionEvent{{
				ID:        "ev-1",
				SessionID: "sess-123",
				Sequence:  7,
				TurnID:    "turn-1",
				Type:      "agent_message",
				AgentName: "coder",
				Content:   `{"text":"hello"}`,
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
		HistoryFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
			return []store.TurnHistory{{
				TurnID: "turn-1",
				Events: []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: "sess-123",
					Sequence:  7,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				}},
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	eventsResp := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/events?type=agent_message&agent_name=coder&turn_id=turn-1&after_sequence=5&limit=10&since=2026-04-03T12:00:00Z", nil)
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", eventsResp.Code, http.StatusOK, eventsResp.Body.String())
	}
	if gotQuery.Type != "agent_message" || gotQuery.AgentName != "coder" || gotQuery.TurnID != "turn-1" || gotQuery.AfterSequence != 5 || gotQuery.Limit != 10 {
		t.Fatalf("query = %#v", gotQuery)
	}

	historyResp := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/history", nil)
	if historyResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", historyResp.Code, http.StatusOK, historyResp.Body.String())
	}
}

func TestSessionTranscriptHandlerReturnsMessages(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		TranscriptFn: func(context.Context, string) ([]transcript.Message, error) {
			return []transcript.Message{{
				ID:        "msg-1",
				Role:      transcript.RoleAssistant,
				Content:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/transcript", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Messages []transcript.Message `json:"messages"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(response.Messages))
	}
	if got := response.Messages[0].Content; got != "hello" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "hello")
	}
}

func TestListAgentsAndHealthHandlers(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")

	handlers := newTestHandlers(t, stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
		},
	}, stubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{
				Status:         "ok",
				UptimeSeconds:  5,
				ActiveSessions: 1,
				ActiveAgents:   1,
				Version:        "dev",
			}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	agentsResp := performRequest(t, engine, http.MethodGet, "/api/agents", nil)
	if agentsResp.Code != http.StatusOK {
		t.Fatalf("agents status = %d, want %d; body=%s", agentsResp.Code, http.StatusOK, agentsResp.Body.String())
	}
	var agents struct {
		Agents []agentPayload `json:"agents"`
	}
	decodeJSONResponse(t, agentsResp, &agents)
	if len(agents.Agents) != 1 || agents.Agents[0].Name != "coder" {
		t.Fatalf("agents = %#v", agents.Agents)
	}

	healthResp := performRequest(t, engine, http.MethodGet, "/api/observe/health", nil)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d; body=%s", healthResp.Code, http.StatusOK, healthResp.Body.String())
	}
	var health struct {
		Health observe.Health `json:"health"`
	}
	decodeJSONResponse(t, healthResp, &health)
	if health.Health.Status != "ok" || health.Health.ActiveSessions != 1 {
		t.Fatalf("health = %#v", health.Health)
	}
}

func TestObserveEventsAndApproveHandlers(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{{
				ID:        "sum-1",
				SessionID: "sess-1",
				Type:      "agent_message",
				AgentName: "coder",
				Summary:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	observeResp := performRequest(t, engine, http.MethodGet, "/api/observe/events?session_id=sess-1", nil)
	if observeResp.Code != http.StatusOK {
		t.Fatalf("observe status = %d, want %d; body=%s", observeResp.Code, http.StatusOK, observeResp.Body.String())
	}
	var observed struct {
		Events []observeEventPayload `json:"events"`
	}
	decodeJSONResponse(t, observeResp, &observed)
	if len(observed.Events) != 1 || observed.Events[0].ID != "sum-1" {
		t.Fatalf("events = %#v", observed.Events)
	}

	approveResp := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", nil)
	if approveResp.Code != http.StatusBadRequest {
		t.Fatalf("approve status = %d, want %d", approveResp.Code, http.StatusBadRequest)
	}
}

func TestApproveSessionHandlerValidatesAndRoutes(t *testing.T) {
	homePaths := newTestHomePaths(t)

	t.Run("missing decision", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("invalid decision", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"maybe"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("session not found", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			ApproveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrSessionNotFound
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/missing/approve", []byte(`{"turn_id":"turn-1","decision":"allow-once"}`))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
		}
	})

	t.Run("pending permission missing", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			ApproveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrPendingPermissionNotFound
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"reject-once"}`))
		if recorder.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
		}
	})

	t.Run("session not active", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			ApproveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrSessionNotActive
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"reject-once"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("valid request", func(t *testing.T) {
		var (
			gotID  string
			gotReq acp.ApproveRequest
		)
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			ApproveFn: func(_ context.Context, id string, req acp.ApproveRequest) error {
				gotID = id
				gotReq = req
				return nil
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"request_id":"req-1","turn_id":"turn-1","decision":"allow-always"}`))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if gotID != "sess-1" {
			t.Fatalf("approve id = %q, want sess-1", gotID)
		}
		if gotReq.RequestID != "req-1" || gotReq.TurnID != "turn-1" || gotReq.Decision != "allow-always" {
			t.Fatalf("approve request = %#v", gotReq)
		}
	})
}

func TestErrorResponsesUseConsistentShape(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, context.DeadlineExceeded
		},
	}, stubObserver{}, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var payload contract.ErrorPayload
	decodeJSONResponse(t, recorder, &payload)
	if payload.Error == "" {
		t.Fatal("expected non-empty error payload")
	}
}

func TestStatusForSessionErrorIncludesApprovalCases(t *testing.T) {
	if status := core.StatusForSessionError(session.ErrSessionNotActive); status != http.StatusBadRequest {
		t.Fatalf("statusForSessionError(ErrSessionNotActive) = %d, want %d", status, http.StatusBadRequest)
	}
	if status := core.StatusForSessionError(session.ErrPendingPermissionNotFound); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrPendingPermissionNotFound) = %d, want %d", status, http.StatusConflict)
	}
	if status := core.StatusForSessionError(session.ErrPendingPermissionConflict); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrPendingPermissionConflict) = %d, want %d", status, http.StatusConflict)
	}
	if status := core.StatusForSessionError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("statusForSessionError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestCORSHeadersPresentOnResponses(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{}, nil
		},
	}, stubObserver{}, homePaths))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/sessions", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
