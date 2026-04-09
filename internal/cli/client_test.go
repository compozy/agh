package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/memory"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUnixSocketClientMethods(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions":
					if got := req.URL.Query().Get("workspace"); got != "ws-1" {
						t.Fatalf("session workspace query = %q, want %q", got, "ws-1")
					}
					return newHTTPResponse(http.StatusOK, `{"sessions":[{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}]}`), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(session create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"agent_name":"coder"`) {
						t.Fatalf("session create body = %s, want agent_name", body)
					}
					return newHTTPResponse(http.StatusCreated, `{"session":{"id":"sess-new","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions/sess-1":
					return newHTTPResponse(http.StatusOK, `{"session":{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/sessions/sess-1":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions/sess-1/resume":
					return newHTTPResponse(http.StatusOK, `{"session":{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions/sess-1/prompt":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(prompt body) error = %v", err)
					}
					if !strings.Contains(string(body), `"message":"hello"`) {
						t.Fatalf("prompt body = %s, want message", body)
					}
					return newHTTPResponse(http.StatusOK, strings.Join([]string{
						"id: 1",
						"event: agent_message",
						`data: {"session_id":"sess-1","turn_id":"turn-1","type":"agent_message","timestamp":"2026-04-03T12:00:00Z","text":"hello back"}`,
						"",
					}, "\n")), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions/sess-1/events":
					if got := req.URL.Query().Get("type"); got != "tool_call" {
						t.Fatalf("session events type query = %q, want %q", got, "tool_call")
					}
					return newHTTPResponse(http.StatusOK, `{"events":[{"id":"evt-1","session_id":"sess-1","sequence":1,"turn_id":"turn-1","type":"tool_call","agent_name":"coder","timestamp":"2026-04-03T12:00:00Z"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions/sess-1/history":
					if got := req.URL.Query().Get("limit"); got != "2" {
						t.Fatalf("history limit query = %q, want %q", got, "2")
					}
					return newHTTPResponse(http.StatusOK, `{"history":[{"turn_id":"turn-1","events":[{"id":"evt-1","session_id":"sess-1","sequence":1,"turn_id":"turn-1","type":"agent_message","agent_name":"coder","content":{"text":"hi"},"timestamp":"2026-04-03T12:00:00Z"}]}]}`), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/workspaces":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(workspace create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"root_dir":"/workspace/project"`) {
						t.Fatalf("workspace create body = %s, want root_dir", body)
					}
					return newHTTPResponse(http.StatusCreated, `{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/workspaces":
					return newHTTPResponse(http.StatusOK, `{"workspaces":[{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/workspaces/alpha":
					return newHTTPResponse(http.StatusOK, `{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"},"sessions":[{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/workspace/project","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}],"agents":[{"name":"coder","provider":"fake","prompt":"hi"}],"skills":[{"name":"review","dir":"/workspace/project/.agh/skills/review","source":"workspace"}]}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/workspaces/ws-1":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(workspace update body) error = %v", err)
					}
					if !strings.Contains(string(body), `"name":"beta"`) {
						t.Fatalf("workspace update body = %s, want name", body)
					}
					return newHTTPResponse(http.StatusOK, `{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"beta","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:05:00Z"}}`), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/workspaces/ws-1":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/agents":
					return newHTTPResponse(http.StatusOK, `{"agents":[{"name":"coder","provider":"fake","prompt":"You are coder."}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/agents/coder":
					return newHTTPResponse(http.StatusOK, `{"agent":{"name":"coder","provider":"fake","prompt":"You are coder."}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/hooks/catalog":
					if got := req.URL.Query().Get("workspace"); got != "alpha" {
						t.Fatalf("hook catalog workspace query = %q, want %q", got, "alpha")
					}
					if got := req.URL.Query().Get("agent"); got != "coder" {
						t.Fatalf("hook catalog agent query = %q, want %q", got, "coder")
					}
					if got := req.URL.Query().Get("event"); got != "tool.pre_call" {
						t.Fatalf("hook catalog event query = %q, want %q", got, "tool.pre_call")
					}
					if got := req.URL.Query().Get("source"); got != "config" {
						t.Fatalf("hook catalog source query = %q, want %q", got, "config")
					}
					if got := req.URL.Query().Get("mode"); got != "sync" {
						t.Fatalf("hook catalog mode query = %q, want %q", got, "sync")
					}
					return newHTTPResponse(http.StatusOK, `{"hooks":[{"order":1,"name":"permission-guard","event":"tool.pre_call","source":"config","mode":"sync","priority":10,"executor_kind":"subprocess"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/hooks/runs":
					if got := req.URL.Query().Get("session"); got != "sess-1" {
						t.Fatalf("hook runs session query = %q, want %q", got, "sess-1")
					}
					if got := req.URL.Query().Get("event"); got != "permission.request" {
						t.Fatalf("hook runs event query = %q, want %q", got, "permission.request")
					}
					if got := req.URL.Query().Get("outcome"); got != "failed" {
						t.Fatalf("hook runs outcome query = %q, want %q", got, "failed")
					}
					if got := req.URL.Query().Get("since"); got != "2026-04-03T11:00:00Z" {
						t.Fatalf("hook runs since query = %q, want %q", got, "2026-04-03T11:00:00Z")
					}
					if got := req.URL.Query().Get("last"); got != "2" {
						t.Fatalf("hook runs last query = %q, want %q", got, "2")
					}
					return newHTTPResponse(http.StatusOK, `{"runs":[{"hook_name":"permission-guard","event":"permission.request","source":"config","mode":"sync","duration_ms":12,"outcome":"failed","error":"boom","recorded_at":"2026-04-03T12:00:00Z"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/hooks/events":
					if got := req.URL.Query().Get("family"); got != "tool" {
						t.Fatalf("hook events family query = %q, want %q", got, "tool")
					}
					if got := req.URL.Query().Get("sync_only"); got != "true" {
						t.Fatalf("hook events sync_only query = %q, want %q", got, "true")
					}
					return newHTTPResponse(http.StatusOK, `{"events":[{"event":"tool.pre_call","family":"tool","sync_eligible":true,"payload_schema":"ToolPreCallPayload","patch_schema":"ToolCallPatch"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/observe/events":
					if got := req.URL.Query().Get("session_id"); got != "sess-1" {
						t.Fatalf("observe session_id query = %q, want %q", got, "sess-1")
					}
					return newHTTPResponse(http.StatusOK, `{"events":[{"id":"sum-1","session_id":"sess-1","type":"agent_message","agent_name":"coder","timestamp":"2026-04-03T12:00:00Z"}]}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/observe/events/stream":
					if got := req.Header.Get("Last-Event-ID"); got != "cursor-1" {
						t.Fatalf("Last-Event-ID = %q, want %q", got, "cursor-1")
					}
					return newHTTPResponse(http.StatusOK, strings.Join([]string{
						"id: 2026-04-03T12:00:00Z|sum-1",
						"event: agent_message",
						`data: {"id":"sum-1","session_id":"sess-1","type":"agent_message","agent_name":"coder","timestamp":"2026-04-03T12:00:00Z"}`,
						"",
					}, "\n")), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/observe/health":
					return newHTTPResponse(http.StatusOK, `{"health":{"status":"ok","uptime_seconds":10,"active_sessions":1,"active_agents":1,"global_db_size_bytes":100,"session_db_size_bytes":200,"version":"dev"}}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/memory":
					if got := req.URL.Query().Get("scope"); got != "global" {
						t.Fatalf("memory scope query = %q, want %q", got, "global")
					}
					return newHTTPResponse(http.StatusOK, `[{"filename":"memory.md","mod_time":"2026-04-03T12:00:00Z","name":"Memory","description":"desc","type":"user"}]`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/memory/memory.md":
					return newHTTPResponse(http.StatusOK, `{"content":"---\nname: Memory\n---\n\nhello"}`), nil
				case req.Method == http.MethodPut && req.URL.Path == "/api/memory/memory.md":
					return newHTTPResponse(http.StatusOK, `{"ok":true}`), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/memory/memory.md":
					if got := req.URL.Query().Get("scope"); got != "workspace" {
						t.Fatalf("delete memory scope query = %q, want %q", got, "workspace")
					}
					return newHTTPResponse(http.StatusOK, `{"ok":true}`), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/memory/consolidate":
					return newHTTPResponse(http.StatusOK, `{"triggered":true}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/daemon/status":
					return newHTTPResponse(http.StatusOK, `{"daemon":{"status":"running","pid":10,"started_at":"2026-04-03T12:00:00Z","socket":"/tmp/agh.sock","http_host":"localhost","http_port":2123,"active_sessions":1,"total_sessions":1,"version":"dev"}}`), nil
				default:
					return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
				}
			}),
		},
	}

	ctx := context.Background()

	status, err := client.DaemonStatus(ctx)
	if err != nil || status.Status != "running" {
		t.Fatalf("DaemonStatus() = %#v, %v", status, err)
	}

	sessions, err := client.ListSessions(ctx, SessionListQuery{Workspace: "ws-1"})
	if err != nil || len(sessions) != 1 {
		t.Fatalf("ListSessions() = %#v, %v", sessions, err)
	}

	createdSession, err := client.CreateSession(ctx, CreateSessionRequest{
		AgentName: "coder",
		Workspace: "ws-1",
	})
	if err != nil || createdSession.ID != "sess-new" {
		t.Fatalf("CreateSession() = %#v, %v", createdSession, err)
	}

	sessionInfo, err := client.GetSession(ctx, "sess-1")
	if err != nil || sessionInfo.ID != "sess-1" {
		t.Fatalf("GetSession() = %#v, %v", sessionInfo, err)
	}

	if err := client.StopSession(ctx, "sess-1"); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}

	resumed, err := client.ResumeSession(ctx, "sess-1")
	if err != nil || resumed.ID != "sess-1" {
		t.Fatalf("ResumeSession() = %#v, %v", resumed, err)
	}

	promptEvents, err := client.PromptSession(ctx, "sess-1", "hello")
	if err != nil || len(promptEvents) != 1 || promptEvents[0].Text != "hello back" {
		t.Fatalf("PromptSession() = %#v, %v", promptEvents, err)
	}

	sessionEvents, err := client.SessionEvents(ctx, "sess-1", SessionEventQuery{Type: "tool_call"})
	if err != nil || len(sessionEvents) != 1 || sessionEvents[0].Type != "tool_call" {
		t.Fatalf("SessionEvents() = %#v, %v", sessionEvents, err)
	}

	history, err := client.SessionHistory(ctx, "sess-1", SessionEventQuery{Last: 2})
	if err != nil || len(history) != 1 {
		t.Fatalf("SessionHistory() = %#v, %v", history, err)
	}

	agents, err := client.ListAgents(ctx)
	if err != nil || len(agents) != 1 {
		t.Fatalf("ListAgents() = %#v, %v", agents, err)
	}

	agent, err := client.GetAgent(ctx, "coder")
	if err != nil || agent.Name != "coder" {
		t.Fatalf("GetAgent() = %#v, %v", agent, err)
	}

	hookCatalog, err := client.HookCatalog(ctx, HookCatalogQuery{
		Workspace: "alpha",
		Agent:     "coder",
		Event:     "tool.pre_call",
		Source:    "config",
		Mode:      "sync",
	})
	if err != nil || len(hookCatalog) != 1 || hookCatalog[0].ExecutorKind != "subprocess" {
		t.Fatalf("HookCatalog() = %#v, %v", hookCatalog, err)
	}

	hookRuns, err := client.HookRuns(ctx, HookRunsQuery{
		Session: "sess-1",
		Event:   "permission.request",
		Outcome: "failed",
		Since:   "2026-04-03T11:00:00Z",
		Last:    2,
	})
	if err != nil || len(hookRuns) != 1 || hookRuns[0].Outcome != "failed" {
		t.Fatalf("HookRuns() = %#v, %v", hookRuns, err)
	}

	hookEvents, err := client.HookEvents(ctx, HookEventsQuery{
		Family:   "tool",
		SyncOnly: true,
	})
	if err != nil || len(hookEvents) != 1 || hookEvents[0].Family != "tool" {
		t.Fatalf("HookEvents() = %#v, %v", hookEvents, err)
	}

	createdWorkspace, err := client.CreateWorkspace(ctx, WorkspaceCreateRequest{RootDir: "/workspace/project"})
	if err != nil || createdWorkspace.ID != "ws-1" {
		t.Fatalf("CreateWorkspace() = %#v, %v", createdWorkspace, err)
	}

	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil || len(workspaces) != 1 {
		t.Fatalf("ListWorkspaces() = %#v, %v", workspaces, err)
	}

	workspaceDetail, err := client.GetWorkspace(ctx, "alpha")
	if err != nil || workspaceDetail.Workspace.ID != "ws-1" || len(workspaceDetail.Skills) != 1 {
		t.Fatalf("GetWorkspace() = %#v, %v", workspaceDetail, err)
	}

	updatedWorkspace, err := client.UpdateWorkspace(ctx, "ws-1", WorkspaceUpdateRequest{Name: ptr("beta")})
	if err != nil || updatedWorkspace.Name != "beta" {
		t.Fatalf("UpdateWorkspace() = %#v, %v", updatedWorkspace, err)
	}

	if err := client.DeleteWorkspace(ctx, "ws-1"); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}

	events, err := client.ObserveEvents(ctx, ObserveEventQuery{SessionID: "sess-1"})
	if err != nil || len(events) != 1 {
		t.Fatalf("ObserveEvents() = %#v, %v", events, err)
	}

	var streamed []SSEEvent
	if err := client.StreamObserveEvents(ctx, ObserveEventQuery{}, "cursor-1", func(event SSEEvent) error {
		streamed = append(streamed, event)
		return nil
	}); err != nil {
		t.Fatalf("StreamObserveEvents() error = %v", err)
	}
	if len(streamed) != 1 || streamed[0].Event != "agent_message" {
		t.Fatalf("streamed = %#v, want one event", streamed)
	}

	health, err := client.ObserveHealth(ctx)
	if err != nil || health.Status != "ok" {
		t.Fatalf("ObserveHealth() = %#v, %v", health, err)
	}

	memories, err := client.ListMemory(ctx, memory.ScopeGlobal, "")
	if err != nil || len(memories) != 1 {
		t.Fatalf("ListMemory() = %#v, %v", memories, err)
	}

	memoryRecord, err := client.ReadMemory(ctx, "memory.md", memory.ScopeGlobal, "")
	if err != nil || !strings.Contains(memoryRecord.Content, "hello") {
		t.Fatalf("ReadMemory() = %#v, %v", memoryRecord, err)
	}

	written, err := client.WriteMemory(ctx, "memory.md", MemoryWriteRequest{Scope: "global", Content: "payload"})
	if err != nil || !written.OK {
		t.Fatalf("WriteMemory() = %#v, %v", written, err)
	}

	deleted, err := client.DeleteMemory(ctx, "memory.md", memory.ScopeWorkspace, "/workspace/project")
	if err != nil || !deleted.OK {
		t.Fatalf("DeleteMemory() = %#v, %v", deleted, err)
	}

	consolidated, err := client.ConsolidateMemory(ctx, "/workspace/project")
	if err != nil || !consolidated.Triggered {
		t.Fatalf("ConsolidateMemory() = %#v, %v", consolidated, err)
	}
}

func TestReadAPIErrorAndHelpers(t *testing.T) {
	t.Parallel()

	resp := newHTTPResponse(http.StatusBadRequest, `{"error":"boom"}`)
	err := readAPIError(resp)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("readAPIError() = %v, want boom", err)
	}

	if got := sessionEventValues(SessionEventQuery{
		Type:          "agent_message",
		AgentName:     "coder",
		TurnID:        "turn-1",
		Since:         fixedTestNow,
		Last:          3,
		AfterSequence: 9,
	}); got.Get("after_sequence") != "9" || got.Get("limit") != "3" {
		t.Fatalf("sessionEventValues() = %v, want limit/after_sequence", got)
	}

	if got := sessionListValues(SessionListQuery{Workspace: "ws-1"}); got.Get("workspace") != "ws-1" {
		t.Fatalf("sessionListValues() = %v, want workspace filter", got)
	}

	if got := observeEventValues(ObserveEventQuery{
		SessionID: "sess-1",
		AgentName: "coder",
		Type:      "done",
		Since:     fixedTestNow,
		Last:      2,
	}); got.Get("session_id") != "sess-1" || got.Get("limit") != "2" {
		t.Fatalf("observeEventValues() = %v, want session_id/limit", got)
	}

	if got := hookCatalogValues(HookCatalogQuery{
		Workspace: "alpha",
		Agent:     "coder",
		Event:     "tool.pre_call",
		Source:    "config",
		Mode:      "sync",
	}); got.Get("workspace") != "alpha" || got.Get("source") != "config" || got.Get("mode") != "sync" {
		t.Fatalf("hookCatalogValues() = %v, want all hook catalog filters", got)
	}

	if got := hookRunsValues(HookRunsQuery{
		Session: "sess-1",
		Event:   "permission.request",
		Outcome: "failed",
		Since:   "2026-04-03T11:00:00Z",
		Last:    2,
	}); got.Get("outcome") != "failed" || got.Get("last") != "2" || got.Get("since") != "2026-04-03T11:00:00Z" {
		t.Fatalf("hookRunsValues() = %v, want all hook runs filters", got)
	}

	if got := hookEventsValues(HookEventsQuery{Family: "tool", SyncOnly: true}); got.Get("family") != "tool" || got.Get("sync_only") != "true" {
		t.Fatalf("hookEventsValues() = %v, want family/sync_only", got)
	}

	if got := memoryValues(memory.ScopeWorkspace, "/workspace/project"); got.Get("scope") != "workspace" || got.Get("workspace") != "/workspace/project" {
		t.Fatalf("memoryValues() = %v, want scope/workspace", got)
	}

	plain := newHTTPResponse(http.StatusInternalServerError, "plain failure")
	if err := readAPIError(plain); err == nil || !strings.Contains(err.Error(), "plain failure") {
		t.Fatalf("readAPIError(plain) = %v, want plain failure", err)
	}

	large := newHTTPResponse(http.StatusInternalServerError, strings.Repeat("x", 2<<20))
	if err := readAPIError(large); err == nil {
		t.Fatal("readAPIError(large) error = nil, want non-nil")
	} else if len(err.Error()) > (1<<20)+128 {
		t.Fatalf("readAPIError(large) len = %d, want capped body size", len(err.Error()))
	}
}

func TestDecodeSSEStopsEarly(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		"id: 1",
		"event: done",
		`data: {"ok":true}`,
		"",
		"id: 2",
		"event: later",
		`data: {"ok":false}`,
		"",
	}, "\n")

	count := 0
	err := decodeSSE(context.Background(), strings.NewReader(body), func(event SSEEvent) error {
		count++
		if event.Event == "done" {
			return errStopSSE
		}
		return nil
	})
	if err != nil {
		t.Fatalf("decodeSSE() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("decodeSSE() count = %d, want 1", count)
	}
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    &http.Request{Method: http.MethodGet},
	}
}

func ptr[T any](value T) *T {
	return &value
}

func nilContext() context.Context {
	return nil
}

func TestNewClientRequiresSocket(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(""); err == nil {
		t.Fatal("NewClient(\"\") error = nil, want non-nil")
	}
}

func TestDoRequestSetsHeaders(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if got := req.Header.Get("User-Agent"); got != defaultUserAgentName {
					t.Fatalf("User-Agent = %q, want %q", got, defaultUserAgentName)
				}
				if got := req.Header.Get("Last-Event-ID"); got != "cursor-9" {
					t.Fatalf("Last-Event-ID = %q, want %q", got, "cursor-9")
				}
				if got := req.URL.Query().Get("since"); got == "" {
					t.Fatal("expected encoded query string")
				}
				return newHTTPResponse(http.StatusOK, `{"events":[]}`), nil
			}),
		},
	}

	err := client.doSSE(context.Background(), http.MethodGet, "/api/observe/events/stream", observeEventValues(ObserveEventQuery{Since: time.Now().UTC()}), nil, "cursor-9", func(SSEEvent) error {
		return nil
	})
	if err != nil {
		t.Fatalf("doSSE() error = %v", err)
	}
}

func TestDoRequestRejectsNilContext(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{},
	}

	if _, err := client.doRequest(nilContext(), http.MethodGet, "/api/daemon/status", nil, nil, ""); err == nil {
		t.Fatal("doRequest(nil) error = nil, want non-nil")
	}
}

func TestCLIUsesSharedContractAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cliType any
		want    any
	}{
		{name: "Should alias CreateSessionRequest to the shared contract", cliType: CreateSessionRequest{}, want: contract.CreateSessionRequest{}},
		{name: "Should alias SessionRecord to the shared contract", cliType: SessionRecord{}, want: contract.SessionPayload{}},
		{name: "Should alias SessionEventRecord to the shared contract", cliType: SessionEventRecord{}, want: contract.SessionEventPayload{}},
		{name: "Should alias TurnHistoryRecord to the shared contract", cliType: TurnHistoryRecord{}, want: contract.TurnHistoryPayload{}},
		{name: "Should alias AgentRecord to the shared contract", cliType: AgentRecord{}, want: contract.AgentPayload{}},
		{name: "Should alias AgentEventRecord to the shared contract", cliType: AgentEventRecord{}, want: contract.AgentEventPayload{}},
		{name: "Should alias HookCatalogQuery to the shared contract", cliType: HookCatalogQuery{}, want: contract.HookCatalogQuery{}},
		{name: "Should alias HookCatalogRecord to the shared contract", cliType: HookCatalogRecord{}, want: contract.HookCatalogPayload{}},
		{name: "Should alias HookRunsQuery to the shared contract", cliType: HookRunsQuery{}, want: contract.HookRunsQuery{}},
		{name: "Should alias HookRunRecord to the shared contract", cliType: HookRunRecord{}, want: contract.HookRunPayload{}},
		{name: "Should alias HookEventsQuery to the shared contract", cliType: HookEventsQuery{}, want: contract.HookEventsQuery{}},
		{name: "Should alias HookEventRecord to the shared contract", cliType: HookEventRecord{}, want: contract.HookEventPayload{}},
		{name: "Should alias ObserveEventRecord to the shared contract", cliType: ObserveEventRecord{}, want: contract.ObserveEventPayload{}},
		{name: "Should alias WorkspaceCreateRequest to the shared contract", cliType: WorkspaceCreateRequest{}, want: contract.CreateWorkspaceRequest{}},
		{name: "Should alias WorkspaceUpdateRequest to the shared contract", cliType: WorkspaceUpdateRequest{}, want: contract.UpdateWorkspaceRequest{}},
		{name: "Should alias WorkspaceRecord to the shared contract", cliType: WorkspaceRecord{}, want: contract.WorkspacePayload{}},
		{name: "Should alias WorkspaceSkillRecord to the shared contract", cliType: WorkspaceSkillRecord{}, want: contract.WorkspaceSkillPayload{}},
		{name: "Should alias MemoryReadRecord to the shared contract", cliType: MemoryReadRecord{}, want: contract.MemoryReadResponse{}},
		{name: "Should alias MemoryWriteRequest to the shared contract", cliType: MemoryWriteRequest{}, want: contract.MemoryWriteRequest{}},
		{name: "Should alias MemoryMutationRecord to the shared contract", cliType: MemoryMutationRecord{}, want: contract.MemoryMutationResponse{}},
		{name: "Should alias MemoryConsolidateRecord to the shared contract", cliType: MemoryConsolidateRecord{}, want: contract.MemoryConsolidateResponse{}},
		{name: "Should alias DaemonStatus to the shared contract", cliType: DaemonStatus{}, want: contract.DaemonStatusPayload{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotType := reflect.TypeOf(tt.cliType)
			wantType := reflect.TypeOf(tt.want)
			if gotType != wantType {
				t.Fatalf("reflect.TypeOf(%s) = %v, want %v", tt.name, gotType, wantType)
			}
		})
	}
}

func TestSharedContractJSONParity(t *testing.T) {
	t.Parallel()

	sessionResponse := `{"sessions":[{"id":"sess-1","name":"demo","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/workspace/project","state":"active","acp_caps":{"supports_load_session":true,"supported_modes":["chat"]},"created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}]}`
	var cliSessions struct {
		Sessions []SessionRecord `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(sessionResponse), &cliSessions); err != nil {
		t.Fatalf("json.Unmarshal(cli session response) error = %v", err)
	}
	if len(cliSessions.Sessions) != 1 || cliSessions.Sessions[0].ACPCaps == nil || !cliSessions.Sessions[0].ACPCaps.SupportsLoadSession {
		t.Fatalf("cli session decode = %#v, want decoded shared contract payload", cliSessions)
	}

	memoryRequest := MemoryWriteRequest{Content: "payload", Scope: "workspace", Workspace: "/workspace/project"}
	cliMemoryJSON, err := json.Marshal(memoryRequest)
	if err != nil {
		t.Fatalf("json.Marshal(cli memory request) error = %v", err)
	}
	sharedMemoryJSON, err := json.Marshal(contract.MemoryWriteRequest(memoryRequest))
	if err != nil {
		t.Fatalf("json.Marshal(shared memory request) error = %v", err)
	}
	if string(cliMemoryJSON) != string(sharedMemoryJSON) {
		t.Fatalf("memory request json = %s, want %s", cliMemoryJSON, sharedMemoryJSON)
	}

	readResponse := `{"content":"stored memory body"}`
	var cliRead MemoryReadRecord
	if err := json.Unmarshal([]byte(readResponse), &cliRead); err != nil {
		t.Fatalf("json.Unmarshal(cli memory read) error = %v", err)
	}
	var sharedRead contract.MemoryReadResponse
	if err := json.Unmarshal([]byte(readResponse), &sharedRead); err != nil {
		t.Fatalf("json.Unmarshal(shared memory read) error = %v", err)
	}
	if !reflect.DeepEqual(cliRead, sharedRead) {
		t.Fatalf("memory read decode = %#v, want %#v", cliRead, sharedRead)
	}

	observeResponse := `{"events":[{"id":"sum-1","session_id":"sess-1","type":"done","agent_name":"coder","summary":"complete","timestamp":"2026-04-03T12:00:00Z"}]}`
	var cliObserve struct {
		Events []ObserveEventRecord `json:"events"`
	}
	if err := json.Unmarshal([]byte(observeResponse), &cliObserve); err != nil {
		t.Fatalf("json.Unmarshal(cli observe response) error = %v", err)
	}
	var sharedObserve struct {
		Events []contract.ObserveEventPayload `json:"events"`
	}
	if err := json.Unmarshal([]byte(observeResponse), &sharedObserve); err != nil {
		t.Fatalf("json.Unmarshal(shared observe response) error = %v", err)
	}
	if !reflect.DeepEqual(cliObserve, sharedObserve) {
		t.Fatalf("observe decode = %#v, want %#v", cliObserve, sharedObserve)
	}

	hookCatalogResponse := `{"hooks":[{"order":1,"name":"permission-guard","event":"tool.pre_call","source":"config","mode":"sync","priority":10,"executor_kind":"subprocess","matcher":{"tool_name":"shell"},"metadata":{"origin":"config"}}]}`
	var cliHookCatalog struct {
		Hooks []HookCatalogRecord `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(hookCatalogResponse), &cliHookCatalog); err != nil {
		t.Fatalf("json.Unmarshal(cli hook catalog response) error = %v", err)
	}
	var sharedHookCatalog struct {
		Hooks []contract.HookCatalogPayload `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(hookCatalogResponse), &sharedHookCatalog); err != nil {
		t.Fatalf("json.Unmarshal(shared hook catalog response) error = %v", err)
	}
	if !reflect.DeepEqual(cliHookCatalog, sharedHookCatalog) {
		t.Fatalf("hook catalog decode = %#v, want %#v", cliHookCatalog, sharedHookCatalog)
	}

	hookRunsResponse := `{"runs":[{"hook_name":"permission-guard","event":"permission.request","source":"config","mode":"sync","duration_ms":12,"outcome":"failed","error":"boom","recorded_at":"2026-04-03T12:00:00Z"}]}`
	var cliHookRuns struct {
		Runs []HookRunRecord `json:"runs"`
	}
	if err := json.Unmarshal([]byte(hookRunsResponse), &cliHookRuns); err != nil {
		t.Fatalf("json.Unmarshal(cli hook runs response) error = %v", err)
	}
	var sharedHookRuns struct {
		Runs []contract.HookRunPayload `json:"runs"`
	}
	if err := json.Unmarshal([]byte(hookRunsResponse), &sharedHookRuns); err != nil {
		t.Fatalf("json.Unmarshal(shared hook runs response) error = %v", err)
	}
	if !reflect.DeepEqual(cliHookRuns, sharedHookRuns) {
		t.Fatalf("hook runs decode = %#v, want %#v", cliHookRuns, sharedHookRuns)
	}

	hookEventsResponse := `{"events":[{"event":"tool.pre_call","family":"tool","sync_eligible":true,"payload_schema":"ToolPreCallPayload","patch_schema":"ToolCallPatch"}]}`
	var cliHookEvents struct {
		Events []HookEventRecord `json:"events"`
	}
	if err := json.Unmarshal([]byte(hookEventsResponse), &cliHookEvents); err != nil {
		t.Fatalf("json.Unmarshal(cli hook events response) error = %v", err)
	}
	var sharedHookEvents struct {
		Events []contract.HookEventPayload `json:"events"`
	}
	if err := json.Unmarshal([]byte(hookEventsResponse), &sharedHookEvents); err != nil {
		t.Fatalf("json.Unmarshal(shared hook events response) error = %v", err)
	}
	if !reflect.DeepEqual(cliHookEvents, sharedHookEvents) {
		t.Fatalf("hook events decode = %#v, want %#v", cliHookEvents, sharedHookEvents)
	}

	daemonResponse := `{"daemon":{"status":"running","pid":10,"started_at":"2026-04-03T12:00:00Z","socket":"/tmp/agh.sock","http_host":"localhost","http_port":2123,"active_sessions":1,"total_sessions":2,"version":"dev"}}`
	var cliDaemon struct {
		Daemon DaemonStatus `json:"daemon"`
	}
	if err := json.Unmarshal([]byte(daemonResponse), &cliDaemon); err != nil {
		t.Fatalf("json.Unmarshal(cli daemon response) error = %v", err)
	}
	var sharedDaemon struct {
		Daemon contract.DaemonStatusPayload `json:"daemon"`
	}
	if err := json.Unmarshal([]byte(daemonResponse), &sharedDaemon); err != nil {
		t.Fatalf("json.Unmarshal(shared daemon response) error = %v", err)
	}
	if !reflect.DeepEqual(cliDaemon, sharedDaemon) {
		t.Fatalf("daemon decode = %#v, want %#v", cliDaemon, sharedDaemon)
	}
}
