package apicore_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/apitest"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type bufferFlusher struct {
	bytes.Buffer
}

func (bufferFlusher) Flush() {}

func TestObserveAndSSEHelpers(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	event := store.EventSummary{ID: "ev-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: timestamp}

	if !apicore.ObserveEventAfterCursor(event, apicore.ObserveCursor{}) {
		t.Fatal("ObserveEventAfterCursor(empty cursor) = false, want true")
	}
	if apicore.ObserveEventAfterCursor(event, apicore.ObserveCursor{Timestamp: timestamp.Add(time.Second), ID: "older"}) {
		t.Fatal("ObserveEventAfterCursor(newer cursor) = true, want false")
	}
	if apicore.ObserveEventAfterCursor(event, apicore.ObserveCursor{Timestamp: timestamp, ID: "zz"}) {
		t.Fatal("ObserveEventAfterCursor(same timestamp higher id) = true, want false")
	}
	if got := apicore.ObserveEventID(event); got == "" {
		t.Fatal("ObserveEventID() = empty, want non-empty")
	}

	writer := &bufferFlusher{}
	next := apicore.EmitObserveEvents(writer, []store.EventSummary{event}, apicore.ObserveCursor{})
	if next.ID != event.ID || next.Timestamp.IsZero() {
		t.Fatalf("EmitObserveEvents() cursor = %#v", next)
	}
	if writer.Len() == 0 {
		t.Fatal("expected SSE output to be written")
	}

	if err := apicore.WriteSSE(writer, apicore.SSEMessage{ID: "2", Name: "done", Data: map[string]string{"ok": "true"}}); err != nil {
		t.Fatalf("WriteSSE() error = %v", err)
	}
	if err := apicore.WriteSSERaw(writer, "3", `"raw"`, "raw"); err != nil {
		t.Fatalf("WriteSSERaw() error = %v", err)
	}
	if err := apicore.WriteSSE(nil, apicore.SSEMessage{}); err == nil {
		t.Fatal("WriteSSE(nil) error = nil, want non-nil")
	}
	if err := apicore.WriteSSERaw(nil, "", "null"); err == nil {
		t.Fatal("WriteSSERaw(nil) error = nil, want non-nil")
	}
}

func TestConversionAndStatusHelpers(t *testing.T) {
	t.Parallel()

	usageValue := int64(10)
	agentEvent := apicore.AgentEventPayloadFromEvent(acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "sess-1",
		TurnID:    "turn-1",
		Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Action:    "fs/read_text_file",
		Usage: &acp.TokenUsage{
			InputTokens: &usageValue,
			Timestamp:   time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		},
		Raw: []byte(`{"ok":true}`),
	})
	if agentEvent.Type != acp.EventTypePermission || agentEvent.Usage == nil || agentEvent.Usage.InputTokens == nil {
		t.Fatalf("agent event payload = %#v", agentEvent)
	}
	if payload := apicore.PayloadJSON("plain-text"); string(payload) == "plain-text" {
		t.Fatalf("PayloadJSON() = %s, want quoted JSON", string(payload))
	}
	if status := apicore.StatusForWorkspaceError(workspacepkg.ErrWorkspacePathTaken); status != http.StatusConflict {
		t.Fatalf("StatusForWorkspaceError() = %d, want %d", status, http.StatusConflict)
	}
	if status := apicore.StatusForMemoryError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("StatusForMemoryError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
	if status := apicore.StatusForMemoryError(nil); status != http.StatusOK {
		t.Fatalf("StatusForMemoryError(nil) = %d, want %d", status, http.StatusOK)
	}
	if got := apicore.NewMemoryValidationError(nil); got != nil {
		t.Fatalf("NewMemoryValidationError(nil) = %v, want nil", got)
	}

	sessions := apicore.SessionPayloadsForWorkspace([]*session.SessionInfo{
		{ID: "sess-1", WorkspaceID: "ws_alpha"},
		{ID: "sess-2", WorkspaceID: "ws_beta"},
	}, "ws_alpha")
	if len(sessions) != 1 || sessions[0].ID != "sess-1" {
		t.Fatalf("SessionPayloadsForWorkspace() = %#v", sessions)
	}
}

func TestBaseHandlersWorkspaceFilteringAndDefaults(t *testing.T) {
	t.Parallel()

	manager := apitest.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{
				{ID: "sess-1", WorkspaceID: "ws_alpha"},
				{ID: "sess-2", WorkspaceID: "ws_beta"},
			}, nil
		},
	}
	workspaces := apitest.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("Get workspace ref = %q, want alpha", ref)
			}
			return workspacepkg.Workspace{ID: "ws_alpha", RootDir: "/workspace"}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, apitest.StubObserver{}, workspaces, nil, nil)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions?workspace=alpha", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("filtered list status = %d, want %d", resp.Code, http.StatusOK)
	}

	handlers := apicore.NewBaseHandlers(apicore.BaseHandlerConfig{})
	handlers.SetHTTPPort(4321)
	if handlers.HTTPPort != 4321 {
		t.Fatalf("HTTPPort = %d, want 4321", handlers.HTTPPort)
	}
	if handlers.TransportName != "" && handlers.TransportName != "apicore" {
		t.Fatalf("TransportName default = %q", handlers.TransportName)
	}
}

func TestMemoryWrapperExports(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	req := contract.MemoryWriteRequest{
		Scope:     "workspace",
		Workspace: workspace,
		Content:   "---\nname: Project\ndescription: desc\ntype: project\n---\n\nbody",
	}
	scope, resolvedWorkspace, err := apicore.ResolveMemoryWriteScope(req)
	if err != nil {
		t.Fatalf("ResolveMemoryWriteScope() error = %v", err)
	}
	if scope != memory.ScopeWorkspace || resolvedWorkspace == "" {
		t.Fatalf("scope=%q workspace=%q", scope, resolvedWorkspace)
	}
	if _, err := apicore.ParseOptionalMemoryScope("bogus"); err == nil {
		t.Fatal("ParseOptionalMemoryScope(bogus) error = nil, want non-nil")
	}
	if _, err := apicore.ResolveMemoryWorkspace(""); err == nil {
		t.Fatal("ResolveMemoryWorkspace(\"\") error = nil, want non-nil")
	}
	if scope, resolved, err := apicore.ResolveMemoryWriteScope(contract.MemoryWriteRequest{
		Content: "---\nname: Global\ndescription: desc\ntype: user\n---\n\nbody",
	}); err != nil || scope != memory.ScopeGlobal || resolved != "" {
		t.Fatalf("ResolveMemoryWriteScope(user default) = %q %q %v", scope, resolved, err)
	}

	store := memory.NewStore(t.TempDir())
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	if err := store.ForWorkspace(workspace).Write(memory.ScopeWorkspace, "note.md", []byte("---\nname: note\ndescription: desc\ntype: project\n---\n\nbody")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	manager := apitest.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			info := apitest.NewSessionInfo("sess-a")
			info.Workspace = workspace
			return []*session.SessionInfo{info}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, apitest.StubObserver{}, apitest.StubWorkspaceService{}, store, nil)
	if _, err := fixture.Handlers.ResolveMemoryLocation("note.md", "workspace", workspace); err != nil {
		t.Fatalf("ResolveMemoryLocation() error = %v", err)
	}
	workspacesOut, err := fixture.Handlers.MemoryHealthWorkspaces(context.Background(), "")
	if err != nil || len(workspacesOut) != 1 {
		t.Fatalf("MemoryHealthWorkspaces() = %#v, %v", workspacesOut, err)
	}
}

func TestObserveStreamAndParseObserveQuery(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	callCount := 0
	observer := apitest.StubObserver{
		QueryEventsFn: func(_ context.Context, _ store.EventSummaryQuery) ([]store.EventSummary, error) {
			callCount++
			ts := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch callCount {
			case 1:
				return []store.EventSummary{{ID: "sum-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: ts}}, nil
			case 2:
				close(done)
				return []store.EventSummary{{ID: "sum-2", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: ts.Add(time.Second)}}, nil
			default:
				return nil, nil
			}
		},
	}
	fixture := newHandlerFixture(t, apitest.StubSessionManager{}, observer, apitest.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.SetStreamDone(done)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/observe/events/stream?agent_name=coder", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("observe stream status = %d, want %d", resp.Code, http.StatusOK)
	}
	if records := apitest.ParseSSE(t, resp.Body.String()); len(records) < 2 {
		t.Fatalf("observe stream records = %d, want at least 2", len(records))
	}
}

func TestBaseHandlersGetAgentNotFound(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t, apitest.StubSessionManager{}, apitest.StubObserver{}, apitest.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.AgentLoader = func(string, aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{}, os.ErrNotExist
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("get missing agent status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}
