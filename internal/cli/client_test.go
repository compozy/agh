package cli

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions/sess-1/resume":
					return newHTTPResponse(http.StatusOK, `{"session":{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`), nil
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

	resumed, err := client.ResumeSession(ctx, "sess-1")
	if err != nil || resumed.ID != "sess-1" {
		t.Fatalf("ResumeSession() = %#v, %v", resumed, err)
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
