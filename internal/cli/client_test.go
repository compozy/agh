package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/memory"
	taskpkg "github.com/pedronauck/agh/internal/task"
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
					return newHTTPResponse(
						http.StatusOK,
						`{"sessions":[{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(session create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"agent_name":"coder"`) {
						t.Fatalf("session create body = %s, want agent_name", body)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"session":{"id":"sess-new","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions/sess-1":
					return newHTTPResponse(
						http.StatusOK,
						`{"session":{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions/sess-1/stop":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/sessions/sess-1/resume":
					return newHTTPResponse(
						http.StatusOK,
						`{"session":{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/tmp","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`,
					), nil
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
					return newHTTPResponse(
						http.StatusOK,
						`{"events":[{"id":"evt-1","session_id":"sess-1","sequence":1,"turn_id":"turn-1","type":"tool_call","agent_name":"coder","timestamp":"2026-04-03T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/sessions/sess-1/history":
					if got := req.URL.Query().Get("limit"); got != "2" {
						t.Fatalf("history limit query = %q, want %q", got, "2")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"history":[{"turn_id":"turn-1","events":[{"id":"evt-1","session_id":"sess-1","sequence":1,"turn_id":"turn-1","type":"agent_message","agent_name":"coder","content":{"text":"hi"},"timestamp":"2026-04-03T12:00:00Z"}]}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/workspaces":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(workspace create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"root_dir":"/workspace/project"`) {
						t.Fatalf("workspace create body = %s, want root_dir", body)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/workspaces":
					return newHTTPResponse(
						http.StatusOK,
						`{"workspaces":[{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/workspaces/alpha":
					return newHTTPResponse(
						http.StatusOK,
						`{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"alpha","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"},"sessions":[{"id":"sess-1","agent_name":"coder","workspace_id":"ws-1","workspace_path":"/workspace/project","state":"active","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:00:00Z"}],"agents":[{"name":"coder","provider":"fake","prompt":"hi"}],"skills":[{"name":"review","dir":"/workspace/project/.agh/skills/review","source":"workspace"}]}`,
					), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/workspaces/ws-1":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(workspace update body) error = %v", err)
					}
					if !strings.Contains(string(body), `"name":"beta"`) {
						t.Fatalf("workspace update body = %s, want name", body)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"workspace":{"id":"ws-1","root_dir":"/workspace/project","name":"beta","created_at":"2026-04-03T12:00:00Z","updated_at":"2026-04-03T12:05:00Z"}}`,
					), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/workspaces/ws-1":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/agents":
					return newHTTPResponse(
						http.StatusOK,
						`{"agents":[{"name":"coder","provider":"fake","prompt":"You are coder."}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/agents/coder":
					return newHTTPResponse(
						http.StatusOK,
						`{"agent":{"name":"coder","provider":"fake","prompt":"You are coder."}}`,
					), nil
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
					return newHTTPResponse(
						http.StatusOK,
						`{"hooks":[{"order":1,"name":"permission-guard","event":"tool.pre_call","source":"config","mode":"sync","priority":10,"executor_kind":"subprocess"}]}`,
					), nil
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
					return newHTTPResponse(
						http.StatusOK,
						`{"runs":[{"hook_name":"permission-guard","event":"permission.request","source":"config","mode":"sync","duration_ms":12,"outcome":"failed","error":"boom","recorded_at":"2026-04-03T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/hooks/events":
					if got := req.URL.Query().Get("family"); got != "tool" {
						t.Fatalf("hook events family query = %q, want %q", got, "tool")
					}
					if got := req.URL.Query().Get("sync_only"); got != "true" {
						t.Fatalf("hook events sync_only query = %q, want %q", got, "true")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"events":[{"event":"tool.pre_call","family":"tool","sync_eligible":true,"payload_schema":"ToolPreCallPayload","patch_schema":"ToolCallPatch"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/observe/events":
					if got := req.URL.Query().Get("session_id"); got != "sess-1" {
						t.Fatalf("observe session_id query = %q, want %q", got, "sess-1")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"events":[{"id":"sum-1","session_id":"sess-1","type":"agent_message","agent_name":"coder","timestamp":"2026-04-03T12:00:00Z"}]}`,
					), nil
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
					return newHTTPResponse(
						http.StatusOK,
						`{"health":{"status":"ok","uptime_seconds":10,"active_sessions":1,"active_agents":1,"global_db_size_bytes":100,"session_db_size_bytes":200,"version":"dev"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/memory":
					if got := req.URL.Query().Get("scope"); got != "global" {
						t.Fatalf("memory scope query = %q, want %q", got, "global")
					}
					return newHTTPResponse(
						http.StatusOK,
						`[{"filename":"memory.md","mod_time":"2026-04-03T12:00:00Z","name":"Memory","description":"desc","type":"user"}]`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/memory/search":
					if got := req.URL.Query().Get("q"); got != "release plan" {
						t.Fatalf("memory search q = %q, want %q", got, "release plan")
					}
					if got := req.URL.Query().Get("scope"); got != "workspace" {
						t.Fatalf("memory search scope = %q, want %q", got, "workspace")
					}
					if got := req.URL.Query().Get("workspace"); got != "/workspace/project" {
						t.Fatalf("memory search workspace = %q, want %q", got, "/workspace/project")
					}
					if got := req.URL.Query().Get("limit"); got != "5" {
						t.Fatalf("memory search limit = %q, want %q", got, "5")
					}
					return newHTTPResponse(
						http.StatusOK,
						`[{"filename":"release.md","scope":"workspace","workspace":"/workspace/project","type":"project","name":"Release Plan","description":"plan","score":3.4,"snippet":"Ship phases incrementally","mod_time":"2026-04-03T12:00:00Z"}]`,
					), nil
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
				case req.Method == http.MethodPost && req.URL.Path == "/api/memory/reindex":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(memory reindex body) error = %v", err)
					}
					if !strings.Contains(string(body), `"scope":"workspace"`) ||
						!strings.Contains(string(body), `"workspace":"/workspace/project"`) {
						t.Fatalf("memory reindex body = %s, want scope/workspace", body)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"indexed_files":2,"scope":"workspace","workspace":"/workspace/project","completed_at":"2026-04-03T12:00:00Z"}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/daemon/status":
					return newHTTPResponse(
						http.StatusOK,
						`{"daemon":{"status":"running","pid":10,"started_at":"2026-04-03T12:00:00Z","socket":"/tmp/agh.sock","http_host":"localhost","http_port":2123,"active_sessions":1,"total_sessions":1,"version":"dev"}}`,
					), nil
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

	searchResults, err := client.SearchMemory(ctx, "release plan", MemorySearchQuery{
		Scope:     memory.ScopeWorkspace,
		Workspace: "/workspace/project",
		Limit:     5,
	})
	if err != nil || len(searchResults) != 1 || searchResults[0].Filename != "release.md" {
		t.Fatalf("SearchMemory() = %#v, %v", searchResults, err)
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

	reindexed, err := client.ReindexMemory(ctx, MemoryReindexRequest{
		Scope:     "workspace",
		Workspace: "/workspace/project",
	})
	if err != nil || reindexed.IndexedFiles != 2 {
		t.Fatalf("ReindexMemory() = %#v, %v", reindexed, err)
	}

	consolidated, err := client.ConsolidateMemory(ctx, "/workspace/project")
	if err != nil || !consolidated.Triggered {
		t.Fatalf("ConsolidateMemory() = %#v, %v", consolidated, err)
	}
}

func TestUnixSocketClientExtensionMethods(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/extensions":
					return newHTTPResponse(
						http.StatusOK,
						`{"extensions":[{"name":"ext-a","version":"0.1.0","type":"resource","source":"user","enabled":true,"state":"active","daemon_running":true}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/extensions":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(extension install body) error = %v", err)
					}
					if !strings.Contains(string(body), `"path":"/tmp/ext-a"`) ||
						!strings.Contains(string(body), `"checksum":"abc123"`) {
						t.Fatalf("extension install body = %s, want path and checksum", body)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"extension":{"name":"ext-a","version":"0.1.0","type":"resource","source":"user","enabled":true,"state":"active","daemon_running":true}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/extensions/ext-a/enable":
					return newHTTPResponse(
						http.StatusOK,
						`{"extension":{"name":"ext-a","version":"0.1.0","type":"resource","source":"user","enabled":true,"state":"active","daemon_running":true}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/extensions/ext-a/disable":
					return newHTTPResponse(
						http.StatusOK,
						`{"extension":{"name":"ext-a","version":"0.1.0","type":"resource","source":"user","enabled":false,"state":"disabled","daemon_running":true}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/extensions/ext-a":
					return newHTTPResponse(
						http.StatusOK,
						`{"extension":{"name":"ext-a","version":"0.1.0","type":"resource","source":"user","enabled":true,"state":"active","daemon_running":true}}`,
					), nil
				default:
					return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
				}
			}),
		},
	}

	ctx := context.Background()

	listed, err := client.ListExtensions(ctx)
	if err != nil || len(listed) != 1 || listed[0].Name != "ext-a" {
		t.Fatalf("ListExtensions() = %#v, %v", listed, err)
	}

	installed, err := client.InstallExtension(ctx, InstallExtensionRequest{
		Path:     "/tmp/ext-a",
		Checksum: "abc123",
	})
	if err != nil || installed.Name != "ext-a" {
		t.Fatalf("InstallExtension() = %#v, %v", installed, err)
	}

	enabled, err := client.EnableExtension(ctx, " ext-a ")
	if err != nil || !enabled.Enabled {
		t.Fatalf("EnableExtension() = %#v, %v", enabled, err)
	}

	disabled, err := client.DisableExtension(ctx, " ext-a ")
	if err != nil || disabled.Enabled {
		t.Fatalf("DisableExtension() = %#v, %v", disabled, err)
	}

	status, err := client.ExtensionStatus(ctx, "ext-a")
	if err != nil || status.State != "active" {
		t.Fatalf("ExtensionStatus() = %#v, %v", status, err)
	}
}

func TestUnixSocketClientAutomationMethods(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2 * time.Minute)

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/jobs":
					if got := req.URL.Query().Get("scope"); got != "workspace" {
						t.Fatalf("job scope query = %q, want %q", got, "workspace")
					}
					if got := req.URL.Query().Get("workspace_id"); got != "ws-alpha" {
						t.Fatalf("job workspace_id query = %q, want %q", got, "ws-alpha")
					}
					if got := req.URL.Query().Get("source"); got != "dynamic" {
						t.Fatalf("job source query = %q, want %q", got, "dynamic")
					}
					if got := req.URL.Query().Get("limit"); got != "3" {
						t.Fatalf("job limit query = %q, want %q", got, "3")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"jobs":[{"id":"job-1","scope":"workspace","workspace_id":"ws-alpha","name":"nightly","agent_name":"coder","prompt":"review repo","schedule":{"mode":"every","interval":"1h"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/automation/jobs":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(job create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"workspace_id":"ws-alpha"`) ||
						!strings.Contains(string(body), `"mode":"every"`) {
						t.Fatalf("job create body = %s, want workspace_id and schedule", body)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"job":{"id":"job-created","scope":"workspace","workspace_id":"ws-alpha","name":"nightly","agent_name":"coder","prompt":"review repo","schedule":{"mode":"every","interval":"1h"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/jobs/job-created":
					return newHTTPResponse(
						http.StatusOK,
						`{"job":{"id":"job-created","scope":"workspace","workspace_id":"ws-alpha","name":"nightly","agent_name":"coder","prompt":"review repo","schedule":{"mode":"every","interval":"1h"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z","next_run":"2026-04-11T13:00:00Z"}}`,
					), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/automation/jobs/job-created":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(job update body) error = %v", err)
					}
					if !strings.Contains(string(body), `"enabled":false`) ||
						!strings.Contains(string(body), `"prompt":"review now"`) {
						t.Fatalf("job update body = %s, want enabled and prompt", body)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"job":{"id":"job-created","scope":"workspace","workspace_id":"ws-alpha","name":"nightly","agent_name":"coder","prompt":"review now","schedule":{"mode":"every","interval":"1h"},"enabled":false,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:05:00Z"}}`,
					), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/automation/jobs/job-created":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/automation/jobs/job-created/trigger":
					return newHTTPResponse(
						http.StatusOK,
						`{"run":{"id":"run-job","job_id":"job-created","session_id":"sess-job","status":"completed","attempt":1,"started_at":"2026-04-11T12:00:00Z","ended_at":"2026-04-11T12:02:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/jobs/job-created/runs":
					if got := req.URL.Query().Get("status"); got != "completed" {
						t.Fatalf("job runs status query = %q, want %q", got, "completed")
					}
					if got := req.URL.Query().Get("since"); got != "2026-04-11T11:00:00Z" {
						t.Fatalf("job runs since query = %q, want %q", got, "2026-04-11T11:00:00Z")
					}
					if got := req.URL.Query().Get("until"); got != "2026-04-11T13:00:00Z" {
						t.Fatalf("job runs until query = %q, want %q", got, "2026-04-11T13:00:00Z")
					}
					if got := req.URL.Query().Get("limit"); got != "2" {
						t.Fatalf("job runs limit query = %q, want %q", got, "2")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"runs":[{"id":"run-job","job_id":"job-created","session_id":"sess-job","status":"completed","attempt":1,"started_at":"2026-04-11T12:00:00Z","ended_at":"2026-04-11T12:02:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/triggers":
					if got := req.URL.Query().Get("scope"); got != "workspace" {
						t.Fatalf("trigger scope query = %q, want %q", got, "workspace")
					}
					if got := req.URL.Query().Get("workspace_id"); got != "ws-alpha" {
						t.Fatalf("trigger workspace_id query = %q, want %q", got, "ws-alpha")
					}
					if got := req.URL.Query().Get("event"); got != "webhook" {
						t.Fatalf("trigger event query = %q, want %q", got, "webhook")
					}
					if got := req.URL.Query().Get("source"); got != "dynamic" {
						t.Fatalf("trigger source query = %q, want %q", got, "dynamic")
					}
					if got := req.URL.Query().Get("limit"); got != "2" {
						t.Fatalf("trigger limit query = %q, want %q", got, "2")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"triggers":[{"id":"trg-1","scope":"workspace","workspace_id":"ws-alpha","name":"deploy-review","agent_name":"coder","prompt":"review {{ index .Data \"payload\" }}","event":"webhook","filter":{"data.branch":"main"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","webhook_id":"wbh_123","endpoint_slug":"deploy-review","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/automation/triggers":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(trigger create body) error = %v", err)
					}
					if !strings.Contains(string(body), `"event":"webhook"`) ||
						!strings.Contains(string(body), `"data.branch":"main"`) {
						t.Fatalf("trigger create body = %s, want event and filter", body)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"trigger":{"id":"trg-created","scope":"workspace","workspace_id":"ws-alpha","name":"deploy-review","agent_name":"coder","prompt":"review {{ index .Data \"payload\" }}","event":"webhook","filter":{"data.branch":"main"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","webhook_id":"wbh_123","endpoint_slug":"deploy-review","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/triggers/trg-created":
					return newHTTPResponse(
						http.StatusOK,
						`{"trigger":{"id":"trg-created","scope":"workspace","workspace_id":"ws-alpha","name":"deploy-review","agent_name":"coder","prompt":"review {{ index .Data \"payload\" }}","event":"webhook","filter":{"data.branch":"main"},"enabled":true,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","webhook_id":"wbh_123","endpoint_slug":"deploy-review","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/automation/triggers/trg-created":
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatalf("io.ReadAll(trigger update body) error = %v", err)
					}
					if !strings.Contains(string(body), `"prompt":"inspect {{ index .Data \"payload\" }}"`) ||
						!strings.Contains(string(body), `"enabled":false`) {
						t.Fatalf("trigger update body = %s, want prompt and enabled", body)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"trigger":{"id":"trg-created","scope":"workspace","workspace_id":"ws-alpha","name":"deploy-review","agent_name":"coder","prompt":"inspect {{ index .Data \"payload\" }}","event":"webhook","filter":{"data.branch":"main"},"enabled":false,"retry":{"strategy":"none"},"fire_limit":{"max":12,"window":"1h"},"source":"dynamic","webhook_id":"wbh_123","endpoint_slug":"deploy-review","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:05:00Z"}}`,
					), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/automation/triggers/trg-created":
					return newHTTPResponse(http.StatusNoContent, ``), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/triggers/trg-created/runs":
					if got := req.URL.Query().Get("status"); got != "completed" {
						t.Fatalf("trigger runs status query = %q, want %q", got, "completed")
					}
					if got := req.URL.Query().Get("limit"); got != "1" {
						t.Fatalf("trigger runs limit query = %q, want %q", got, "1")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"runs":[{"id":"run-trigger","trigger_id":"trg-created","session_id":"sess-trigger","status":"completed","attempt":1,"started_at":"2026-04-11T12:00:00Z","ended_at":"2026-04-11T12:02:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/runs":
					if got := req.URL.Query().Get("job_id"); got != "job-created" {
						t.Fatalf("runs job_id query = %q, want %q", got, "job-created")
					}
					if got := req.URL.Query().Get("trigger_id"); got != "trg-created" {
						t.Fatalf("runs trigger_id query = %q, want %q", got, "trg-created")
					}
					if got := req.URL.Query().Get("status"); got != "completed" {
						t.Fatalf("runs status query = %q, want %q", got, "completed")
					}
					if got := req.URL.Query().Get("since"); got != "2026-04-11T12:00:00Z" {
						t.Fatalf("runs since query = %q, want %q", got, "2026-04-11T12:00:00Z")
					}
					if got := req.URL.Query().Get("until"); got != "2026-04-11T13:00:00Z" {
						t.Fatalf("runs until query = %q, want %q", got, "2026-04-11T13:00:00Z")
					}
					if got := req.URL.Query().Get("limit"); got != "5" {
						t.Fatalf("runs limit query = %q, want %q", got, "5")
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"runs":[{"id":"run-shared","job_id":"job-created","trigger_id":"trg-created","session_id":"sess-shared","status":"completed","attempt":1,"started_at":"2026-04-11T12:00:00Z","ended_at":"2026-04-11T12:02:00Z"}]}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/automation/runs/run-shared":
					return newHTTPResponse(
						http.StatusOK,
						`{"run":{"id":"run-shared","job_id":"job-created","trigger_id":"trg-created","session_id":"sess-shared","status":"completed","attempt":1,"started_at":"2026-04-11T12:00:00Z","ended_at":"2026-04-11T12:02:00Z"}}`,
					), nil
				default:
					return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
				}
			}),
		},
	}

	ctx := context.Background()

	t.Run("Should list automation jobs", func(t *testing.T) {
		jobs, err := client.ListAutomationJobs(ctx, AutomationJobQuery{
			Scope:       automationpkg.AutomationScopeWorkspace,
			WorkspaceID: "ws-alpha",
			Source:      automationpkg.JobSourceDynamic,
			Limit:       3,
		})
		if err != nil || len(jobs) != 1 || jobs[0].ID != "job-1" {
			t.Fatalf("ListAutomationJobs() = %#v, %v", jobs, err)
		}
	})

	t.Run("Should create automation jobs", func(t *testing.T) {
		createdJob, err := client.CreateAutomationJob(ctx, AutomationJobCreateRequest{
			Scope:       automationpkg.AutomationScopeWorkspace,
			WorkspaceID: "ws-alpha",
			Name:        "nightly",
			AgentName:   "coder",
			Prompt:      "review repo",
			Schedule: automationpkg.ScheduleSpec{
				Mode:     automationpkg.ScheduleModeEvery,
				Interval: "1h",
			},
		})
		if err != nil || createdJob.ID != "job-created" {
			t.Fatalf("CreateAutomationJob() = %#v, %v", createdJob, err)
		}
	})

	t.Run("Should get automation jobs", func(t *testing.T) {
		job, err := client.GetAutomationJob(ctx, "job-created")
		if err != nil || job.NextRun == nil {
			t.Fatalf("GetAutomationJob() = %#v, %v", job, err)
		}
	})

	t.Run("Should update automation jobs", func(t *testing.T) {
		updatedJob, err := client.UpdateAutomationJob(ctx, "job-created", AutomationJobUpdateRequest{
			Prompt:  ptr("review now"),
			Enabled: ptr(false),
		})
		if err != nil || updatedJob.Enabled {
			t.Fatalf("UpdateAutomationJob() = %#v, %v", updatedJob, err)
		}
	})

	t.Run("Should trigger automation jobs", func(t *testing.T) {
		triggeredRun, err := client.TriggerAutomationJob(ctx, "job-created")
		if err != nil || triggeredRun.ID != "run-job" {
			t.Fatalf("TriggerAutomationJob() = %#v, %v", triggeredRun, err)
		}
	})

	t.Run("Should list automation job runs", func(t *testing.T) {
		jobRuns, err := client.AutomationJobRuns(ctx, "job-created", AutomationRunQuery{
			Status: automationpkg.RunCompleted,
			Since:  startedAt.Add(-time.Hour),
			Until:  startedAt.Add(time.Hour),
			Limit:  2,
		})
		if err != nil || len(jobRuns) != 1 || jobRuns[0].JobID != "job-created" {
			t.Fatalf("AutomationJobRuns() = %#v, %v", jobRuns, err)
		}
	})

	t.Run("Should delete automation jobs", func(t *testing.T) {
		if err := client.DeleteAutomationJob(ctx, "job-created"); err != nil {
			t.Fatalf("DeleteAutomationJob() error = %v", err)
		}
	})

	t.Run("Should list automation triggers", func(t *testing.T) {
		triggers, err := client.ListAutomationTriggers(ctx, AutomationTriggerQuery{
			Scope:       automationpkg.AutomationScopeWorkspace,
			WorkspaceID: "ws-alpha",
			Event:       "webhook",
			Source:      automationpkg.JobSourceDynamic,
			Limit:       2,
		})
		if err != nil || len(triggers) != 1 || triggers[0].ID != "trg-1" {
			t.Fatalf("ListAutomationTriggers() = %#v, %v", triggers, err)
		}
	})

	t.Run("Should create automation triggers", func(t *testing.T) {
		createdTrigger, err := client.CreateAutomationTrigger(ctx, AutomationTriggerCreateRequest{
			Scope:         automationpkg.AutomationScopeWorkspace,
			WorkspaceID:   "ws-alpha",
			Name:          "deploy-review",
			AgentName:     "coder",
			Prompt:        `review {{ index .Data "payload" }}`,
			Event:         "webhook",
			Filter:        map[string]string{"data.branch": "main"},
			EndpointSlug:  "deploy-review",
			WebhookSecret: "shared-secret",
		})
		if err != nil || createdTrigger.ID != "trg-created" {
			t.Fatalf("CreateAutomationTrigger() = %#v, %v", createdTrigger, err)
		}
	})

	t.Run("Should get automation triggers", func(t *testing.T) {
		trigger, err := client.GetAutomationTrigger(ctx, "trg-created")
		if err != nil || trigger.WebhookID != "wbh_123" {
			t.Fatalf("GetAutomationTrigger() = %#v, %v", trigger, err)
		}
	})

	t.Run("Should update automation triggers", func(t *testing.T) {
		updatedTrigger, err := client.UpdateAutomationTrigger(ctx, "trg-created", AutomationTriggerUpdateRequest{
			Prompt:  ptr(`inspect {{ index .Data "payload" }}`),
			Enabled: ptr(false),
		})
		if err != nil || updatedTrigger.Enabled {
			t.Fatalf("UpdateAutomationTrigger() = %#v, %v", updatedTrigger, err)
		}
	})

	t.Run("Should list automation trigger runs", func(t *testing.T) {
		triggerRuns, err := client.AutomationTriggerRuns(ctx, "trg-created", AutomationRunQuery{
			Status: automationpkg.RunCompleted,
			Limit:  1,
		})
		if err != nil || len(triggerRuns) != 1 || triggerRuns[0].TriggerID != "trg-created" {
			t.Fatalf("AutomationTriggerRuns() = %#v, %v", triggerRuns, err)
		}
	})

	t.Run("Should delete automation triggers", func(t *testing.T) {
		if err := client.DeleteAutomationTrigger(ctx, "trg-created"); err != nil {
			t.Fatalf("DeleteAutomationTrigger() error = %v", err)
		}
	})

	t.Run("Should list automation runs", func(t *testing.T) {
		runs, err := client.ListAutomationRuns(ctx, AutomationRunQuery{
			JobID:     "job-created",
			TriggerID: "trg-created",
			Status:    automationpkg.RunCompleted,
			Since:     startedAt,
			Until:     startedAt.Add(time.Hour),
			Limit:     5,
		})
		if err != nil || len(runs) != 1 || runs[0].ID != "run-shared" {
			t.Fatalf("ListAutomationRuns() = %#v, %v", runs, err)
		}
	})

	t.Run("Should get automation runs", func(t *testing.T) {
		run, err := client.GetAutomationRun(ctx, "run-shared")
		if err != nil || run.EndedAt == nil || !run.EndedAt.Equal(endedAt) {
			t.Fatalf("GetAutomationRun() = %#v, %v", run, err)
		}
	})
}

func TestUnixSocketClientTaskMethods(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/tasks":
					if got := req.URL.Query().Get("scope"); got != "workspace" {
						t.Fatalf("task scope query = %q, want %q", got, "workspace")
					}
					if got := req.URL.Query().Get("workspace"); got != "alpha" {
						t.Fatalf("task workspace query = %q, want %q", got, "alpha")
					}
					if got := req.URL.Query().Get("status"); got != "ready" {
						t.Fatalf("task status query = %q, want %q", got, "ready")
					}
					if got := req.URL.Query().Get("owner_kind"); got != "pool" {
						t.Fatalf("task owner_kind query = %q, want %q", got, "pool")
					}
					if got := req.URL.Query().Get("owner_ref"); got != "triage" {
						t.Fatalf("task owner_ref query = %q, want %q", got, "triage")
					}
					if got := req.URL.Query().Get("parent_task_id"); got != "task-root" {
						t.Fatalf("task parent_task_id query = %q, want %q", got, "task-root")
					}
					if got := req.URL.Query().Get("network_channel"); got != "builders" {
						t.Fatalf("task network_channel query = %q, want %q", got, "builders")
					}
					if got := req.URL.Query().Get("limit"); got != "3" {
						t.Fatalf("task limit query = %q, want %q", got, "3")
					}
					body := mustJSON(
						t,
						contract.TasksResponse{Tasks: []contract.TaskSummaryPayload{sampleTaskSummaryRecord()}},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/tasks":
					var payload contract.CreateTaskRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(create task body) error = %v", err)
					}
					if payload.Scope != taskpkg.ScopeWorkspace ||
						payload.Workspace != "alpha" ||
						payload.NetworkChannel != "builders" ||
						payload.Title != "Investigate flaky task runs" ||
						payload.Owner == nil ||
						payload.Owner.Kind != taskpkg.OwnerKindPool ||
						payload.Owner.Ref != "triage" {
						t.Fatalf("create task payload = %#v", payload)
					}
					body := mustJSON(t, contract.TaskResponse{Task: sampleTaskRecord()})
					return newHTTPResponse(http.StatusCreated, string(body)), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/tasks/task-1":
					body := mustJSON(t, contract.TaskDetailResponse{Task: sampleTaskDetailRecord()})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/tasks/task-1":
					var payload contract.UpdateTaskRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(update task body) error = %v", err)
					}
					if payload.Title == nil || *payload.Title != "Investigate resolved" ||
						payload.NetworkChannel == nil ||
						*payload.NetworkChannel != "ops" {
						t.Fatalf("update task payload = %#v", payload)
					}
					updated := sampleTaskRecord()
					updated.Title = "Investigate resolved"
					updated.NetworkChannel = "ops"
					body := mustJSON(t, contract.TaskResponse{Task: updated})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/tasks/task-1/cancel":
					var payload contract.CancelTaskRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(cancel task body) error = %v", err)
					}
					if payload.Reason != "operator-request" {
						t.Fatalf("cancel task payload = %#v, want reason", payload)
					}
					canceled := sampleTaskRecord()
					canceled.Status = taskpkg.TaskStatusCanceled
					body := mustJSON(t, contract.TaskResponse{Task: canceled})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/tasks/task-1/children":
					var payload contract.CreateTaskChildRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(create child body) error = %v", err)
					}
					if payload.Scope != taskpkg.ScopeWorkspace || payload.Workspace != "alpha" ||
						payload.Title != "Check runtime logs" {
						t.Fatalf("create child payload = %#v", payload)
					}
					child := sampleTaskRecord()
					child.ID = "task-child"
					child.Title = "Check runtime logs"
					body := mustJSON(t, contract.TaskResponse{Task: child})
					return newHTTPResponse(http.StatusCreated, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/tasks/task-1/dependencies":
					var payload contract.AddTaskDependencyRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(add dependency body) error = %v", err)
					}
					if payload.DependsOnTaskID != "task-blocker" || payload.Kind != taskpkg.DependencyKindBlocks {
						t.Fatalf("add dependency payload = %#v", payload)
					}
					body := mustJSON(t, contract.TaskDetailResponse{Task: sampleTaskDetailRecord()})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodDelete && req.URL.Path == "/api/tasks/task-1/dependencies/task-blocker":
					body := mustJSON(t, contract.TaskDetailResponse{Task: sampleTaskDetailRecord()})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/tasks/task-1/runs":
					var payload contract.EnqueueTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(enqueue run body) error = %v", err)
					}
					if payload.IdempotencyKey != "idem-1" || payload.NetworkChannel != "builders" {
						t.Fatalf("enqueue run payload = %#v", payload)
					}
					body := mustJSON(t, contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusQueued)})
					return newHTTPResponse(http.StatusCreated, string(body)), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/tasks/task-1/runs":
					if got := req.URL.Query().Get("status"); got != "running" {
						t.Fatalf("task runs status query = %q, want %q", got, "running")
					}
					if got := req.URL.Query().Get("session_id"); got != "sess-1" {
						t.Fatalf("task runs session_id query = %q, want %q", got, "sess-1")
					}
					if got := req.URL.Query().Get("limit"); got != "2" {
						t.Fatalf("task runs limit query = %q, want %q", got, "2")
					}
					body := mustJSON(
						t,
						contract.TaskRunsResponse{
							Runs: []contract.TaskRunPayload{sampleTaskRunRecord(taskpkg.TaskRunStatusRunning)},
						},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/claim":
					var payload contract.ClaimTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(claim run body) error = %v", err)
					}
					if payload.IdempotencyKey != "idem-claim" {
						t.Fatalf("claim run payload = %#v", payload)
					}
					body := mustJSON(
						t,
						contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusClaimed)},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/start":
					var payload contract.StartTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(start run body) error = %v", err)
					}
					if payload.IdempotencyKey != "idem-start" {
						t.Fatalf("start run payload = %#v", payload)
					}
					body := mustJSON(
						t,
						contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusRunning)},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/attach-session":
					var payload contract.AttachTaskRunSessionRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(attach session body) error = %v", err)
					}
					if payload.SessionID != "sess-attach" {
						t.Fatalf("attach session payload = %#v", payload)
					}
					body := mustJSON(
						t,
						contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusStarting)},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/complete":
					var payload contract.CompleteTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(complete run body) error = %v", err)
					}
					if string(payload.Result) != `{"ok":true}` {
						t.Fatalf("complete run payload = %#v", payload)
					}
					body := mustJSON(
						t,
						contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusCompleted)},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/fail":
					var payload contract.FailTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(fail run body) error = %v", err)
					}
					if payload.Error != "boom" || string(payload.Metadata) != `{"code":"E_TASK"}` {
						t.Fatalf("fail run payload = %#v", payload)
					}
					body := mustJSON(t, contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusFailed)})
					return newHTTPResponse(http.StatusOK, string(body)), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/task-runs/run-1/cancel":
					var payload contract.CancelTaskRunRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(cancel run body) error = %v", err)
					}
					if payload.Reason != "operator-request" || string(payload.Metadata) != `{"source":"cli"}` {
						t.Fatalf("cancel run payload = %#v", payload)
					}
					body := mustJSON(
						t,
						contract.TaskRunResponse{Run: sampleTaskRunRecord(taskpkg.TaskRunStatusCanceled)},
					)
					return newHTTPResponse(http.StatusOK, string(body)), nil
				default:
					return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
				}
			}),
		},
	}

	ctx := context.Background()

	t.Run("Should list tasks", func(t *testing.T) {
		tasks, err := client.ListTasks(ctx, TaskListQuery{
			Scope:          taskpkg.ScopeWorkspace,
			Workspace:      "alpha",
			Status:         taskpkg.TaskStatusReady,
			OwnerKind:      taskpkg.OwnerKindPool,
			OwnerRef:       "triage",
			ParentTaskID:   "task-root",
			NetworkChannel: "builders",
			Limit:          3,
		})
		if err != nil || len(tasks) != 1 || tasks[0].ID != "task-1" {
			t.Fatalf("ListTasks() = %#v, %v", tasks, err)
		}
	})

	t.Run("Should create get update and cancel tasks", func(t *testing.T) {
		created, err := client.CreateTask(ctx, CreateTaskRequest{
			Scope:          taskpkg.ScopeWorkspace,
			Workspace:      "alpha",
			NetworkChannel: "builders",
			Title:          "Investigate flaky task runs",
			Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "triage"},
		})
		if err != nil || created.ID != "task-1" {
			t.Fatalf("CreateTask() = %#v, %v", created, err)
		}

		detail, err := client.GetTask(ctx, "task-1")
		if err != nil || detail.Task.ID != "task-1" || len(detail.Dependencies) != 1 {
			t.Fatalf("GetTask() = %#v, %v", detail, err)
		}

		updated, err := client.UpdateTask(ctx, "task-1", UpdateTaskRequest{
			Title:          ptr("Investigate resolved"),
			NetworkChannel: ptr("ops"),
		})
		if err != nil || updated.Title != "Investigate resolved" || updated.NetworkChannel != "ops" {
			t.Fatalf("UpdateTask() = %#v, %v", updated, err)
		}

		canceled, err := client.CancelTask(ctx, "task-1", CancelTaskRequest{Reason: "operator-request"})
		if err != nil || canceled.Status != taskpkg.TaskStatusCanceled {
			t.Fatalf("CancelTask() = %#v, %v", canceled, err)
		}
	})

	t.Run("Should manage child tasks dependencies and runs", func(t *testing.T) {
		child, err := client.CreateChildTask(ctx, "task-1", CreateTaskChildRequest{
			Scope:     taskpkg.ScopeWorkspace,
			Workspace: "alpha",
			Title:     "Check runtime logs",
		})
		if err != nil || child.ID != "task-child" {
			t.Fatalf("CreateChildTask() = %#v, %v", child, err)
		}

		detail, err := client.AddTaskDependency(ctx, "task-1", AddTaskDependencyRequest{
			DependsOnTaskID: "task-blocker",
			Kind:            taskpkg.DependencyKindBlocks,
		})
		if err != nil || len(detail.Dependencies) != 1 {
			t.Fatalf("AddTaskDependency() = %#v, %v", detail, err)
		}

		detail, err = client.RemoveTaskDependency(ctx, "task-1", "task-blocker")
		if err != nil || len(detail.Runs) != 1 {
			t.Fatalf("RemoveTaskDependency() = %#v, %v", detail, err)
		}

		enqueued, err := client.EnqueueTaskRun(ctx, "task-1", EnqueueTaskRunRequest{
			IdempotencyKey: "idem-1",
			NetworkChannel: "builders",
		})
		if err != nil || enqueued.Status != taskpkg.TaskRunStatusQueued {
			t.Fatalf("EnqueueTaskRun() = %#v, %v", enqueued, err)
		}

		runs, err := client.ListTaskRuns(ctx, "task-1", TaskRunListQuery{
			Status:    taskpkg.TaskRunStatusRunning,
			SessionID: "sess-1",
			Limit:     2,
		})
		if err != nil || len(runs) != 1 || runs[0].Status != taskpkg.TaskRunStatusRunning {
			t.Fatalf("ListTaskRuns() = %#v, %v", runs, err)
		}

		claimed, err := client.ClaimTaskRun(ctx, "run-1", ClaimTaskRunRequest{IdempotencyKey: "idem-claim"})
		if err != nil || claimed.Status != taskpkg.TaskRunStatusClaimed {
			t.Fatalf("ClaimTaskRun() = %#v, %v", claimed, err)
		}

		started, err := client.StartTaskRun(ctx, "run-1", StartTaskRunRequest{IdempotencyKey: "idem-start"})
		if err != nil || started.Status != taskpkg.TaskRunStatusRunning {
			t.Fatalf("StartTaskRun() = %#v, %v", started, err)
		}

		attached, err := client.AttachTaskRunSession(
			ctx,
			"run-1",
			AttachTaskRunSessionRequest{SessionID: "sess-attach"},
		)
		if err != nil || attached.Status != taskpkg.TaskRunStatusStarting {
			t.Fatalf("AttachTaskRunSession() = %#v, %v", attached, err)
		}

		completed, err := client.CompleteTaskRun(
			ctx,
			"run-1",
			CompleteTaskRunRequest{Result: mustJSON(t, map[string]bool{"ok": true})},
		)
		if err != nil || completed.Status != taskpkg.TaskRunStatusCompleted {
			t.Fatalf("CompleteTaskRun() = %#v, %v", completed, err)
		}

		failed, err := client.FailTaskRun(
			ctx,
			"run-1",
			FailTaskRunRequest{Error: "boom", Metadata: mustJSON(t, map[string]string{"code": "E_TASK"})},
		)
		if err != nil || failed.Status != taskpkg.TaskRunStatusFailed {
			t.Fatalf("FailTaskRun() = %#v, %v", failed, err)
		}

		canceled, err := client.CancelTaskRun(
			ctx,
			"run-1",
			CancelTaskRunRequest{Reason: "operator-request", Metadata: mustJSON(t, map[string]string{"source": "cli"})},
		)
		if err != nil || canceled.Status != taskpkg.TaskRunStatusCanceled {
			t.Fatalf("CancelTaskRun() = %#v, %v", canceled, err)
		}
	})
}

func TestUnixSocketClientBridgeMethods(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{
		socketPath: "/tmp/agh.sock",
		httpClient: &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/api/bridges":
					return newHTTPResponse(
						http.StatusOK,
						`{"bridges":[{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"ready","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/bridges":
					var payload contract.CreateBridgeRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(create bridge body) error = %v", err)
					}
					if payload.Scope != bridgepkg.ScopeGlobal ||
						payload.WorkspaceID != "" ||
						payload.Platform != "telegram" ||
						payload.ExtensionName != "ext-telegram" ||
						payload.DisplayName != "Support" ||
						!payload.Enabled ||
						payload.Status != bridgepkg.BridgeStatusStarting ||
						!reflect.DeepEqual(payload.RoutingPolicy, bridgepkg.RoutingPolicy{IncludePeer: true}) ||
						len(payload.DeliveryDefaults) != 0 {
						t.Fatalf("create bridge payload = %#v", payload)
					}
					return newHTTPResponse(
						http.StatusCreated,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/bridges/brg-a":
					return newHTTPResponse(
						http.StatusOK,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"ready","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`,
					), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/api/bridges/brg-a":
					var payload contract.UpdateBridgeRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(update bridge body) error = %v", err)
					}
					if payload.DisplayName == nil ||
						*payload.DisplayName != "Support Ops" ||
						payload.RoutingPolicy == nil ||
						!reflect.DeepEqual(
							*payload.RoutingPolicy,
							bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
						) ||
						payload.DeliveryDefaults != nil {
						t.Fatalf("update bridge payload = %#v, want updated display name", payload)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support Ops","enabled":true,"status":"ready","routing_policy":{"include_peer":true,"include_thread":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:05:00Z"}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/bridges/brg-a/enable":
					return newHTTPResponse(
						http.StatusOK,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:06:00Z"}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/bridges/brg-a/disable":
					return newHTTPResponse(
						http.StatusOK,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":false,"status":"disabled","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:07:00Z"}}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/bridges/brg-a/restart":
					return newHTTPResponse(
						http.StatusOK,
						`{"bridge":{"id":"brg-a","scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:08:00Z"}}`,
					), nil
				case req.Method == http.MethodGet && req.URL.Path == "/api/bridges/brg-a/routes":
					return newHTTPResponse(
						http.StatusOK,
						`{"routes":[{"routing_key_hash":"hash-a","scope":"global","bridge_instance_id":"brg-a","peer_id":"peer-1","thread_id":"thread-1","session_id":"sess-1","agent_name":"coder","last_activity_at":"2026-04-11T12:09:00Z","created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:09:00Z"}]}`,
					), nil
				case req.Method == http.MethodPost && req.URL.Path == "/api/bridges/brg-a/test-delivery":
					var payload contract.BridgeTestDeliveryRequest
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatalf("json.Decode(test delivery body) error = %v", err)
					}
					if payload.Message != "hello" || payload.Target.PeerID != "peer-1" ||
						payload.Target.ThreadID != "thread-1" ||
						payload.Target.Mode != bridgepkg.DeliveryModeReply {
						t.Fatalf("test delivery payload = %#v", payload)
					}
					return newHTTPResponse(
						http.StatusOK,
						`{"status":"resolved","message":"hello","delivery_target":{"bridge_instance_id":"brg-a","peer_id":"peer-1","thread_id":"thread-1","mode":"reply"}}`,
					), nil
				default:
					return newHTTPResponse(http.StatusNotFound, `{"error":"missing"}`), nil
				}
			}),
		},
	}

	ctx := context.Background()

	listed, err := client.ListBridges(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != "brg-a" {
		t.Fatalf("ListBridges() = %#v, %v", listed, err)
	}

	created, err := client.CreateBridge(ctx, CreateBridgeRequest{
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusStarting,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	if err != nil || created.ID != "brg-a" {
		t.Fatalf("CreateBridge() = %#v, %v", created, err)
	}

	status, err := client.GetBridge(ctx, "brg-a")
	if err != nil || status.Status != bridgepkg.BridgeStatusReady {
		t.Fatalf("GetBridge() = %#v, %v", status, err)
	}

	updated, err := client.UpdateBridge(ctx, "brg-a", UpdateBridgeRequest{
		DisplayName: ptr("Support Ops"),
		RoutingPolicy: &bridgepkg.RoutingPolicy{
			IncludePeer:   true,
			IncludeThread: true,
		},
	})
	if err != nil || updated.DisplayName != "Support Ops" || !updated.RoutingPolicy.IncludeThread {
		t.Fatalf("UpdateBridge() = %#v, %v", updated, err)
	}

	enabled, err := client.EnableBridge(ctx, " brg-a ")
	if err != nil || enabled.Status != bridgepkg.BridgeStatusStarting || !enabled.Enabled {
		t.Fatalf("EnableBridge() = %#v, %v", enabled, err)
	}

	disabled, err := client.DisableBridge(ctx, " brg-a ")
	if err != nil || disabled.Status != bridgepkg.BridgeStatusDisabled || disabled.Enabled {
		t.Fatalf("DisableBridge() = %#v, %v", disabled, err)
	}

	restarted, err := client.RestartBridge(ctx, " brg-a ")
	if err != nil || restarted.Status != bridgepkg.BridgeStatusStarting || !restarted.Enabled {
		t.Fatalf("RestartBridge() = %#v, %v", restarted, err)
	}

	routes, err := client.BridgeRoutes(ctx, "brg-a")
	if err != nil || len(routes) != 1 || routes[0].ThreadID != "thread-1" {
		t.Fatalf("BridgeRoutes() = %#v, %v", routes, err)
	}

	delivery, err := client.TestBridgeDelivery(ctx, "brg-a", BridgeTestDeliveryRequest{
		Message: "hello",
		Target: BridgeDeliveryTargetInput{
			PeerID:   "peer-1",
			ThreadID: "thread-1",
			Mode:     bridgepkg.DeliveryModeReply,
		},
	})
	if err != nil || delivery.DeliveryTarget.Mode != bridgepkg.DeliveryModeReply ||
		delivery.DeliveryTarget.ThreadID != "thread-1" {
		t.Fatalf("TestBridgeDelivery() = %#v, %v", delivery, err)
	}
}

func TestReadAPIErrorAndHelpers(t *testing.T) {
	t.Parallel()

	resp := newHTTPResponse(http.StatusBadRequest, `{"error":"boom"}`)
	defer func() {
		_ = resp.Body.Close()
	}()
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

	if got := hookEventsValues(
		HookEventsQuery{Family: "tool", SyncOnly: true},
	); got.Get("family") != "tool" ||
		got.Get("sync_only") != "true" {
		t.Fatalf("hookEventsValues() = %v, want family/sync_only", got)
	}

	if got := memoryValues(
		memory.ScopeWorkspace,
		"/workspace/project",
	); got.Get("scope") != "workspace" ||
		got.Get("workspace") != "/workspace/project" {
		t.Fatalf("memoryValues() = %v, want scope/workspace", got)
	}

	if got := memorySearchValues("release plan", MemorySearchQuery{
		Scope:     memory.ScopeWorkspace,
		Workspace: "/workspace/project",
		Limit:     5,
	}); got.Get("q") != "release plan" ||
		got.Get("scope") != "workspace" ||
		got.Get("workspace") != "/workspace/project" ||
		got.Get("limit") != "5" {
		t.Fatalf("memorySearchValues() = %v, want q/scope/workspace/limit", got)
	}

	if got := automationJobValues(AutomationJobQuery{
		Scope:       automationpkg.AutomationScopeWorkspace,
		WorkspaceID: "ws-alpha",
		Source:      automationpkg.JobSourceDynamic,
		Limit:       3,
	}); got.Get("scope") != "workspace" || got.Get("workspace_id") != "ws-alpha" || got.Get("source") != "dynamic" || got.Get("limit") != "3" {
		t.Fatalf("automationJobValues() = %v, want all job filters", got)
	}

	if got := automationTriggerValues(AutomationTriggerQuery{
		Scope:       automationpkg.AutomationScopeWorkspace,
		WorkspaceID: "ws-alpha",
		Event:       "webhook",
		Source:      automationpkg.JobSourceDynamic,
		Limit:       2,
	}); got.Get("scope") != "workspace" || got.Get("workspace_id") != "ws-alpha" || got.Get("event") != "webhook" || got.Get("source") != "dynamic" || got.Get("limit") != "2" {
		t.Fatalf("automationTriggerValues() = %v, want all trigger filters", got)
	}

	if got := automationRunValues(AutomationRunQuery{
		JobID:     "job-1",
		TriggerID: "trg-1",
		Status:    automationpkg.RunCompleted,
		Since:     fixedTestNow,
		Until:     fixedTestNow.Add(time.Hour),
		Limit:     4,
	}); got.Get("job_id") != "job-1" || got.Get("trigger_id") != "trg-1" || got.Get("status") != "completed" || got.Get("limit") != "4" {
		t.Fatalf("automationRunValues() = %v, want all run filters", got)
	}

	if got := taskValues(TaskListQuery{
		Scope:          taskpkg.ScopeWorkspace,
		Workspace:      "alpha",
		Status:         taskpkg.TaskStatusReady,
		OwnerKind:      taskpkg.OwnerKindPool,
		OwnerRef:       "triage",
		ParentTaskID:   "task-root",
		NetworkChannel: "builders",
		Limit:          3,
	}); got.Get("scope") != "workspace" || got.Get("workspace") != "alpha" || got.Get("status") != "ready" || got.Get("owner_kind") != "pool" || got.Get("owner_ref") != "triage" || got.Get("parent_task_id") != "task-root" || got.Get("network_channel") != "builders" || got.Get("limit") != "3" {
		t.Fatalf("taskValues() = %v, want all task filters", got)
	}

	if got := taskRunValues(TaskRunListQuery{
		Status:    taskpkg.TaskRunStatusRunning,
		SessionID: "sess-1",
		Limit:     2,
	}); got.Get("status") != "running" || got.Get("session_id") != "sess-1" || got.Get("limit") != "2" {
		t.Fatalf("taskRunValues() = %v, want all task run filters", got)
	}

	plain := newHTTPResponse(http.StatusInternalServerError, "plain failure")
	defer func() {
		_ = plain.Body.Close()
	}()
	if err := readAPIError(plain); err == nil || !strings.Contains(err.Error(), "plain failure") {
		t.Fatalf("readAPIError(plain) = %v, want plain failure", err)
	}

	large := newHTTPResponse(http.StatusInternalServerError, strings.Repeat("x", 2<<20))
	defer func() {
		_ = large.Body.Close()
	}()
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

func TestDecodeSSERejectsNilArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		ctx     context.Context
		body    io.Reader
		handler SSEHandler
		wantErr string
	}{
		{
			name:    "nil context",
			ctx:     nil,
			body:    strings.NewReader("event: ping\n\n"),
			handler: func(SSEEvent) error { return nil },
			wantErr: "sse: context is required",
		},
		{
			name:    "nil body",
			ctx:     context.Background(),
			body:    nil,
			handler: func(SSEEvent) error { return nil },
			wantErr: "sse: body is required",
		},
		{
			name:    "nil handler",
			ctx:     context.Background(),
			body:    strings.NewReader("event: ping\n\n"),
			handler: nil,
			wantErr: "sse: handler is required",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := decodeSSE(tt.ctx, tt.body, tt.handler)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("decodeSSE() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeSSEPropagatesHandlerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	body := strings.Join([]string{
		"id: 1",
		"event: done",
		`data: {"ok":true}`,
		"",
	}, "\n")

	err := decodeSSE(context.Background(), strings.NewReader(body), func(SSEEvent) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("decodeSSE() error = %v, want %v", err, wantErr)
	}
}

func TestDecodeSSEPreservesMultiLineData(t *testing.T) {
	t.Parallel()

	body := strings.Join([]string{
		"id: 1",
		"event: message",
		`data: {"first":true}`,
		`data: {"second":true}`,
		"",
	}, "\n")

	var seen SSEEvent
	err := decodeSSE(context.Background(), strings.NewReader(body), func(event SSEEvent) error {
		seen = event
		return nil
	})
	if err != nil {
		t.Fatalf("decodeSSE() error = %v", err)
	}
	if got, want := string(seen.Data), "{\"first\":true}\n{\"second\":true}"; got != want {
		t.Fatalf("decodeSSE() data = %q, want %q", got, want)
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

	err := client.doSSE(
		context.Background(),
		http.MethodGet,
		"/api/observe/events/stream",
		observeEventValues(ObserveEventQuery{Since: time.Now().UTC()}),
		nil,
		"cursor-9",
		func(SSEEvent) error {
			return nil
		},
	)
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

	response, err := client.doRequest(nilContext(), http.MethodGet, "/api/daemon/status", nil, nil, "")
	if response != nil {
		defer func() {
			_ = response.Body.Close()
		}()
	}
	if err == nil {
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
		{
			name:    "Should alias CreateSessionRequest to the shared contract",
			cliType: CreateSessionRequest{},
			want:    contract.CreateSessionRequest{},
		},
		{
			name:    "Should alias SessionRecord to the shared contract",
			cliType: SessionRecord{},
			want:    contract.SessionPayload{},
		},
		{
			name:    "Should alias SessionEventRecord to the shared contract",
			cliType: SessionEventRecord{},
			want:    contract.SessionEventPayload{},
		},
		{
			name:    "Should alias TurnHistoryRecord to the shared contract",
			cliType: TurnHistoryRecord{},
			want:    contract.TurnHistoryPayload{},
		},
		{
			name:    "Should alias AgentRecord to the shared contract",
			cliType: AgentRecord{},
			want:    contract.AgentPayload{},
		},
		{
			name:    "Should alias AgentEventRecord to the shared contract",
			cliType: AgentEventRecord{},
			want:    contract.AgentEventPayload{},
		},
		{
			name:    "Should alias HookCatalogQuery to the shared contract",
			cliType: HookCatalogQuery{},
			want:    contract.HookCatalogQuery{},
		},
		{
			name:    "Should alias HookCatalogRecord to the shared contract",
			cliType: HookCatalogRecord{},
			want:    contract.HookCatalogPayload{},
		},
		{
			name:    "Should alias HookRunsQuery to the shared contract",
			cliType: HookRunsQuery{},
			want:    contract.HookRunsQuery{},
		},
		{
			name:    "Should alias HookRunRecord to the shared contract",
			cliType: HookRunRecord{},
			want:    contract.HookRunPayload{},
		},
		{
			name:    "Should alias HookEventsQuery to the shared contract",
			cliType: HookEventsQuery{},
			want:    contract.HookEventsQuery{},
		},
		{
			name:    "Should alias HookEventRecord to the shared contract",
			cliType: HookEventRecord{},
			want:    contract.HookEventPayload{},
		},
		{
			name:    "Should alias ObserveEventRecord to the shared contract",
			cliType: ObserveEventRecord{},
			want:    contract.ObserveEventPayload{},
		},
		{
			name:    "Should alias WorkspaceCreateRequest to the shared contract",
			cliType: WorkspaceCreateRequest{},
			want:    contract.CreateWorkspaceRequest{},
		},
		{
			name:    "Should alias WorkspaceUpdateRequest to the shared contract",
			cliType: WorkspaceUpdateRequest{},
			want:    contract.UpdateWorkspaceRequest{},
		},
		{
			name:    "Should alias WorkspaceRecord to the shared contract",
			cliType: WorkspaceRecord{},
			want:    contract.WorkspacePayload{},
		},
		{
			name:    "Should alias WorkspaceSkillRecord to the shared contract",
			cliType: WorkspaceSkillRecord{},
			want:    contract.WorkspaceSkillPayload{},
		},
		{
			name:    "Should alias MemoryReadRecord to the shared contract",
			cliType: MemoryReadRecord{},
			want:    contract.MemoryReadResponse{},
		},
		{
			name:    "Should alias MemoryWriteRequest to the shared contract",
			cliType: MemoryWriteRequest{},
			want:    contract.MemoryWriteRequest{},
		},
		{
			name:    "Should alias MemoryMutationRecord to the shared contract",
			cliType: MemoryMutationRecord{},
			want:    contract.MemoryMutationResponse{},
		},
		{
			name:    "Should alias MemoryConsolidateRecord to the shared contract",
			cliType: MemoryConsolidateRecord{},
			want:    contract.MemoryConsolidateResponse{},
		},
		{
			name:    "Should alias AutomationJobCreateRequest to the shared contract",
			cliType: AutomationJobCreateRequest{},
			want:    contract.CreateJobRequest{},
		},
		{
			name:    "Should alias AutomationJobUpdateRequest to the shared contract",
			cliType: AutomationJobUpdateRequest{},
			want:    contract.UpdateJobRequest{},
		},
		{
			name:    "Should alias AutomationTriggerCreateRequest to the shared contract",
			cliType: AutomationTriggerCreateRequest{},
			want:    contract.CreateTriggerRequest{},
		},
		{
			name:    "Should alias AutomationTriggerUpdateRequest to the shared contract",
			cliType: AutomationTriggerUpdateRequest{},
			want:    contract.UpdateTriggerRequest{},
		},
		{name: "Should alias JobRecord to the shared contract", cliType: JobRecord{}, want: contract.JobPayload{}},
		{
			name:    "Should alias TriggerRecord to the shared contract",
			cliType: TriggerRecord{},
			want:    contract.TriggerPayload{},
		},
		{name: "Should alias RunRecord to the shared contract", cliType: RunRecord{}, want: contract.RunPayload{}},
		{
			name:    "Should alias DaemonStatus to the shared contract",
			cliType: DaemonStatus{},
			want:    contract.DaemonStatusPayload{},
		},
		{
			name:    "Should alias CreateBridgeRequest to the shared contract",
			cliType: CreateBridgeRequest{},
			want:    contract.CreateBridgeRequest{},
		},
		{
			name:    "Should alias UpdateBridgeRequest to the shared contract",
			cliType: UpdateBridgeRequest{},
			want:    contract.UpdateBridgeRequest{},
		},
		{
			name:    "Should alias BridgeTestDeliveryRequest to the shared contract",
			cliType: BridgeTestDeliveryRequest{},
			want:    contract.BridgeTestDeliveryRequest{},
		},
		{
			name:    "Should alias BridgeDeliveryTargetInput to the shared contract",
			cliType: BridgeDeliveryTargetInput{},
			want:    contract.BridgeDeliveryTargetInput{},
		},
		{
			name:    "Should alias BridgeRecord to the bridge domain type",
			cliType: BridgeRecord{},
			want:    bridgepkg.BridgeInstance{},
		},
		{
			name:    "Should alias BridgeRouteRecord to the bridge domain type",
			cliType: BridgeRouteRecord{},
			want:    bridgepkg.BridgeRoute{},
		},
		{
			name:    "Should alias DeliveryTargetRecord to the bridge domain type",
			cliType: DeliveryTargetRecord{},
			want:    bridgepkg.DeliveryTarget{},
		},
		{
			name:    "Should alias BridgeTestDeliveryRecord to the shared contract",
			cliType: BridgeTestDeliveryRecord{},
			want:    contract.BridgeTestDeliveryResponse{},
		},
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
	if len(cliSessions.Sessions) != 1 || cliSessions.Sessions[0].ACPCaps == nil ||
		!cliSessions.Sessions[0].ACPCaps.SupportsLoadSession {
		t.Fatalf("cli session decode = %#v, want decoded shared contract payload", cliSessions)
	}

	memoryRequest := MemoryWriteRequest{Content: "payload", Scope: "workspace", Workspace: "/workspace/project"}
	cliMemoryJSON, err := json.Marshal(memoryRequest)
	if err != nil {
		t.Fatalf("json.Marshal(cli memory request) error = %v", err)
	}
	var sharedMemoryRequest = memoryRequest
	sharedMemoryJSON, err := json.Marshal(sharedMemoryRequest)
	if err != nil {
		t.Fatalf("json.Marshal(shared memory request) error = %v", err)
	}
	if !bytes.Equal(cliMemoryJSON, sharedMemoryJSON) {
		t.Fatalf("memory request json = %s, want %s", cliMemoryJSON, sharedMemoryJSON)
	}

	bridgeRequest := CreateBridgeRequest{
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}
	cliBridgeJSON, err := json.Marshal(bridgeRequest)
	if err != nil {
		t.Fatalf("json.Marshal(cli bridge request) error = %v", err)
	}
	var sharedBridgeRequest = bridgeRequest
	sharedBridgeJSON, err := json.Marshal(sharedBridgeRequest)
	if err != nil {
		t.Fatalf("json.Marshal(shared bridge request) error = %v", err)
	}
	if !bytes.Equal(cliBridgeJSON, sharedBridgeJSON) {
		t.Fatalf("bridge request json = %s, want %s", cliBridgeJSON, sharedBridgeJSON)
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

	bridgeResponse := `{"bridge":{"id":"brg-1","scope":"workspace","workspace_id":"ws-alpha","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"ready","routing_policy":{"include_peer":true,"include_thread":true},"delivery_defaults":{"mode":"reply"},"created_at":"2026-04-11T12:00:00Z","updated_at":"2026-04-11T12:00:00Z"}}`
	var cliBridge struct {
		Bridge BridgeRecord `json:"bridge"`
	}
	if err := json.Unmarshal([]byte(bridgeResponse), &cliBridge); err != nil {
		t.Fatalf("json.Unmarshal(cli bridge response) error = %v", err)
	}
	var sharedBridge struct {
		Bridge bridgepkg.BridgeInstance `json:"bridge"`
	}
	if err := json.Unmarshal([]byte(bridgeResponse), &sharedBridge); err != nil {
		t.Fatalf("json.Unmarshal(shared bridge response) error = %v", err)
	}
	if !reflect.DeepEqual(cliBridge, sharedBridge) {
		t.Fatalf("bridge decode = %#v, want %#v", cliBridge, sharedBridge)
	}
}
