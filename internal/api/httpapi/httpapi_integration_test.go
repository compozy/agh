//go:build integration

package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestHTTPFullRoundTripWithRealSessionManager(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	indexResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/"), nil, nil)
	if indexResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(indexResp.Body)
		_ = indexResp.Body.Close()
		t.Fatalf("root status = %d, want %d; body=%s", indexResp.StatusCode, http.StatusOK, string(body))
	}
	indexBody, err := io.ReadAll(indexResp.Body)
	_ = indexResp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(root) error = %v", err)
	}
	if !strings.Contains(string(indexBody), `<div id="app"></div>`) {
		t.Fatalf("root body = %q, want SPA shell", string(indexBody))
	}

	deepLinkResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/session/demo"), nil, nil)
	if deepLinkResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(deepLinkResp.Body)
		_ = deepLinkResp.Body.Close()
		t.Fatalf("deep link status = %d, want %d; body=%s", deepLinkResp.StatusCode, http.StatusOK, string(body))
	}
	deepLinkBody, err := io.ReadAll(deepLinkResp.Body)
	_ = deepLinkResp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(deep link) error = %v", err)
	}
	if !strings.Contains(string(deepLinkBody), `<div id="app"></div>`) {
		t.Fatalf("deep link body = %q, want SPA shell", string(deepLinkBody))
	}

	statusResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/daemon/status"), nil, nil)
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", statusResp.StatusCode, http.StatusOK)
	}
	_ = statusResp.Body.Close()

	origin := fmt.Sprintf("http://%s:%d", runtime.host, runtime.port)
	createResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions"), []byte(`{"agent_name":"coder","name":"demo","workspace_path":"`+runtime.workspace+`"}`), map[string]string{"Origin": origin})
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	if got := createResp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, origin)
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, createResp, &created)
	if created.Session.ID == "" {
		t.Fatal("expected created session id")
	}

	listResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions"), nil, nil)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list sessions status = %d, want %d", listResp.StatusCode, http.StatusOK)
	}
	var listed struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeHTTPJSON(t, listResp, &listed)
	if len(listed.Sessions) != 1 || listed.Sessions[0].ID != created.Session.ID {
		t.Fatalf("listed sessions = %#v", listed.Sessions)
	}
	canonicalWorkspace, err := filepath.EvalSymlinks(runtime.workspace)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q) error = %v", runtime.workspace, err)
	}
	if listed.Sessions[0].WorkspaceID == "" || listed.Sessions[0].WorkspacePath != canonicalWorkspace {
		t.Fatalf("listed session workspace = %#v", listed.Sessions[0])
	}

	filteredResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions?workspace="+created.Session.WorkspaceID), nil, nil)
	if filteredResp.StatusCode != http.StatusOK {
		t.Fatalf("filtered sessions status = %d, want %d", filteredResp.StatusCode, http.StatusOK)
	}
	var filtered struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeHTTPJSON(t, filteredResp, &filtered)
	if len(filtered.Sessions) != 1 || filtered.Sessions[0].ID != created.Session.ID {
		t.Fatalf("filtered sessions = %#v", filtered.Sessions)
	}

	promptResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions/"+created.Session.ID+"/prompt"), []byte(`{"message":"hello"}`), nil)
	if promptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(promptResp.Body)
		_ = promptResp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", promptResp.StatusCode, http.StatusOK, string(body))
	}
	if got := promptResp.Header.Get("x-vercel-ai-ui-message-stream"); got != "v1" {
		t.Fatalf("x-vercel-ai-ui-message-stream = %q, want v1", got)
	}
	promptBody, err := io.ReadAll(promptResp.Body)
	_ = promptResp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(prompt SSE) error = %v", err)
	}
	promptEvents := parseSSE(t, string(promptBody))
	if len(promptEvents) < 6 {
		t.Fatalf("prompt SSE events = %d, want at least 6; body=%s", len(promptEvents), string(promptBody))
	}
	if promptEvents[len(promptEvents)-1].Event != "" || string(promptEvents[len(promptEvents)-1].Data) != "[DONE]" {
		t.Fatalf("last prompt record = %#v, want [DONE]", promptEvents[len(promptEvents)-1])
	}

	eventsResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions/"+created.Session.ID+"/events"), nil, nil)
	if eventsResp.StatusCode != http.StatusOK {
		t.Fatalf("session events status = %d, want %d", eventsResp.StatusCode, http.StatusOK)
	}
	var events struct {
		Events []sessionEventPayload `json:"events"`
	}
	decodeHTTPJSON(t, eventsResp, &events)
	if len(events.Events) < 4 {
		t.Fatalf("persisted session events = %d, want at least 4", len(events.Events))
	}
}

func TestHTTPSessionTranscriptEndpointWithRealSessionManager(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	sessionID := createIntegrationSession(t, runtime)
	sendPrompt(t, runtime, sessionID, "hello")

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/transcript"), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("transcript status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	var payload struct {
		Messages []transcript.Message `json:"messages"`
	}
	decodeHTTPJSON(t, resp, &payload)
	if len(payload.Messages) != 4 {
		t.Fatalf("len(messages) = %d, want 4", len(payload.Messages))
	}
	if got := payload.Messages[0].Role; got != transcript.RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, transcript.RoleUser)
	}
	if got := payload.Messages[0].Content; got != "hello" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "hello")
	}
	if got := payload.Messages[1].Role; got != transcript.RoleAssistant {
		t.Fatalf("messages[1].Role = %q, want %q", got, transcript.RoleAssistant)
	}
	if got := payload.Messages[2].Role; got != transcript.RoleToolCall {
		t.Fatalf("messages[2].Role = %q, want %q", got, transcript.RoleToolCall)
	}
	if got := payload.Messages[3].Role; got != transcript.RoleToolResult {
		t.Fatalf("messages[3].Role = %q, want %q", got, transcript.RoleToolResult)
	}
}

func TestHTTPSessionStreamReconnectsWithLastEventID(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	sessionID := createIntegrationSession(t, runtime)
	sendPrompt(t, runtime, sessionID, "hello")
	stopIntegrationSession(t, runtime, sessionID)

	streamResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/stream"), nil, nil)
	if streamResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamResp.Body)
		_ = streamResp.Body.Close()
		t.Fatalf("session stream status = %d, want %d; body=%s", streamResp.StatusCode, http.StatusOK, string(body))
	}
	initial := collectLiveSSE(t, streamResp.Body, 6, 2*time.Second)
	_ = streamResp.Body.Close()
	if len(initial) < 6 {
		t.Fatalf("initial stream events = %d, want 6", len(initial))
	}
	if initial[len(initial)-1].Event != session.EventTypeSessionStopped {
		t.Fatalf("last event = %q, want %q", initial[len(initial)-1].Event, session.EventTypeSessionStopped)
	}

	headers := map[string]string{"Last-Event-ID": initial[0].ID}
	replayResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/stream"), nil, headers)
	if replayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(replayResp.Body)
		_ = replayResp.Body.Close()
		t.Fatalf("replay stream status = %d, want %d; body=%s", replayResp.StatusCode, http.StatusOK, string(body))
	}
	replayed := collectLiveSSE(t, replayResp.Body, 5, 2*time.Second)
	_ = replayResp.Body.Close()
	if len(replayed) < 5 {
		t.Fatalf("replayed events = %d, want 5", len(replayed))
	}
	if replayed[0].ID != initial[1].ID {
		t.Fatalf("replayed first id = %q, want %q", replayed[0].ID, initial[1].ID)
	}
}

func TestHTTPApprovePermissionFullFlow(t *testing.T) {
	runtime := newIntegrationRuntimeWithPermissionWait(t, 250*time.Millisecond)
	sessionID := createIntegrationSession(t, runtime)

	promptResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/prompt"), []byte(`{"message":"request permission"}`), nil)
	if promptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(promptResp.Body)
		_ = promptResp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", promptResp.StatusCode, http.StatusOK, string(body))
	}

	requestIDCh := make(chan string, 1)
	resultCh := make(chan []sseRecord, 1)
	go streamPermissionPrompt(t, promptResp.Body, requestIDCh, resultCh)

	var requestID string
	select {
	case requestID = <-requestIDCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for permission request id from SSE stream")
	}
	if requestID == "" {
		t.Fatal("permission request_id = empty, want non-empty")
	}

	approveResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/approve"), []byte(fmt.Sprintf(`{"request_id":"%s","decision":"allow-always"}`, requestID)), nil)
	if approveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(approveResp.Body)
		_ = approveResp.Body.Close()
		t.Fatalf("approve status = %d, want %d; body=%s", approveResp.StatusCode, http.StatusOK, string(body))
	}
	_ = approveResp.Body.Close()

	var records []sseRecord
	select {
	case records = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for completed SSE stream")
	}

	permissionPayloads := extractPermissionPayloads(t, records)
	if len(permissionPayloads) < 2 {
		t.Fatalf("permission payloads = %#v, want initial and final permission events", permissionPayloads)
	}
	if permissionPayloads[0].RequestID != requestID || permissionPayloads[0].Decision != "" {
		t.Fatalf("initial permission payload = %#v", permissionPayloads[0])
	}
	if permissionPayloads[len(permissionPayloads)-1].Decision != "allow-always" {
		t.Fatalf("final permission payload = %#v, want allow-always", permissionPayloads[len(permissionPayloads)-1])
	}
	if !recordsContainTextDelta(records, "allow-always") {
		t.Fatalf("records = %#v, want allow-always text delta", records)
	}
}

func TestHTTPApprovePermissionTimeout(t *testing.T) {
	runtime := newIntegrationRuntimeWithPermissionWait(t, 25*time.Millisecond)
	sessionID := createIntegrationSession(t, runtime)

	promptResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/prompt"), []byte(`{"message":"request permission"}`), nil)
	if promptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(promptResp.Body)
		_ = promptResp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", promptResp.StatusCode, http.StatusOK, string(body))
	}

	requestIDCh := make(chan string, 1)
	resultCh := make(chan []sseRecord, 1)
	go streamPermissionPrompt(t, promptResp.Body, requestIDCh, resultCh)

	select {
	case <-requestIDCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for permission request id from SSE stream")
	}

	var records []sseRecord
	select {
	case records = <-resultCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for completed SSE stream")
	}

	permissionPayloads := extractPermissionPayloads(t, records)
	if len(permissionPayloads) < 2 {
		t.Fatalf("permission payloads = %#v, want initial and final permission events", permissionPayloads)
	}
	if permissionPayloads[len(permissionPayloads)-1].Decision != "reject-once" {
		t.Fatalf("final permission payload = %#v, want reject-once", permissionPayloads[len(permissionPayloads)-1])
	}
	if !recordsContainTextDelta(records, "reject-once") {
		t.Fatalf("records = %#v, want reject-once text delta", records)
	}
}

func TestHTTPMemoryRoundTripAndDelete(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	writeResp := mustHTTPRequest(t, runtime.client, http.MethodPut, mustURL(runtime.host, runtime.port, "/api/memory/integration.md"), []byte(`{"scope":"global","content":"`+escapeJSON(memoryDocument(t, "Integration", "desc", memory.MemoryTypeUser, "hello integration"))+`"}`), nil)
	if writeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(writeResp.Body)
		_ = writeResp.Body.Close()
		t.Fatalf("write status = %d, want %d; body=%s", writeResp.StatusCode, http.StatusOK, string(body))
	}
	_ = writeResp.Body.Close()

	readResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/memory/integration.md?scope=global"), nil, nil)
	if readResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(readResp.Body)
		_ = readResp.Body.Close()
		t.Fatalf("read status = %d, want %d; body=%s", readResp.StatusCode, http.StatusOK, string(body))
	}
	var readPayload memoryReadResponse
	decodeHTTPJSON(t, readResp, &readPayload)
	if !strings.Contains(readPayload.Content, "hello integration") {
		t.Fatalf("content = %q, want written body", readPayload.Content)
	}

	listResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/memory?scope=global"), nil, nil)
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		_ = listResp.Body.Close()
		t.Fatalf("list status = %d, want %d; body=%s", listResp.StatusCode, http.StatusOK, string(body))
	}
	var headers []memory.MemoryHeader
	decodeHTTPJSON(t, listResp, &headers)
	if len(headers) != 1 || headers[0].Filename != "integration.md" {
		t.Fatalf("headers = %#v, want integration.md", headers)
	}

	deleteResp := mustHTTPRequest(t, runtime.client, http.MethodDelete, mustURL(runtime.host, runtime.port, "/api/memory/integration.md?scope=global"), nil, nil)
	if deleteResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusOK, string(body))
	}
	_ = deleteResp.Body.Close()

	emptyList := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/memory?scope=global"), nil, nil)
	if emptyList.StatusCode != http.StatusOK {
		t.Fatalf("post-delete list status = %d, want %d", emptyList.StatusCode, http.StatusOK)
	}
	decodeHTTPJSON(t, emptyList, &headers)
	if len(headers) != 0 {
		t.Fatalf("headers = %#v, want empty list after delete", headers)
	}
}

func TestHTTPMemoryConsolidateIntegration(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/memory/consolidate"), []byte(`{"workspace":"`+runtime.workspace+`"}`), nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("consolidate status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	var payload memoryConsolidateResponse
	decodeHTTPJSON(t, resp, &payload)
	if !payload.Triggered || runtime.dream.calls != 1 {
		t.Fatalf("payload = %#v dream.calls=%d, want triggered once", payload, runtime.dream.calls)
	}
}

func TestHTTPShutdownWaitsForInflightRequests(t *testing.T) {
	homePaths := newTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{
			ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
				entered <- struct{}{}
				<-release
				return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
			},
		}),
		WithObserver(stubObserver{
			HealthFn: func(context.Context) (observe.Health, error) { return observe.Health{Status: "ok"}, nil },
		}),
		WithWorkspaceResolver(stubWorkspaceService{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	released := false
	defer func() {
		if !released {
			close(release)
		}
		_ = server.Shutdown(context.Background())
	}()

	client := &http.Client{}
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := client.Get(mustURL(cfg.HTTP.Host, server.Port(), "/api/sessions"))
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for in-flight request")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown() returned early: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	released = true

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shutdown")
	}

	select {
	case err := <-errCh:
		t.Fatalf("request error = %v", err)
	case resp := <-respCh:
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("request status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
		}
		_ = resp.Body.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for request response")
	}
}

type integrationRuntime struct {
	client    *http.Client
	server    *Server
	manager   *session.Manager
	observer  *observe.Observer
	memory    *memory.Store
	dream     *integrationDreamTrigger
	host      string
	port      int
	workspace string
}

type integrationDreamTrigger struct {
	enabled   bool
	triggered bool
	reason    string
	last      time.Time
	calls     int
}

func (t *integrationDreamTrigger) Trigger(context.Context, string) (bool, string, error) {
	t.calls++
	return t.triggered, t.reason, nil
}

func (t *integrationDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return t.last, nil
}

func (t *integrationDreamTrigger) Enabled() bool {
	return t.enabled
}

type integrationNotifierFanout struct {
	notifiers []session.Notifier
}

func (f *integrationNotifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	for _, notifier := range f.notifiers {
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

type integrationDriver struct {
	mu             sync.Mutex
	nextPID        int
	nextSess       int
	permissionWait time.Duration
	states         map[*session.AgentProcess]chan struct{}
	approvals      map[*session.AgentProcess]chan acp.ApproveRequest
}

func newIntegrationDriver(permissionWait time.Duration) *integrationDriver {
	if permissionWait <= 0 {
		permissionWait = 100 * time.Millisecond
	}
	return &integrationDriver{
		nextPID:        2000,
		nextSess:       1,
		permissionWait: permissionWait,
		states:         make(map[*session.AgentProcess]chan struct{}),
		approvals:      make(map[*session.AgentProcess]chan acp.ApproveRequest),
	}
}

func (d *integrationDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextPID++
	d.nextSess++
	done := make(chan struct{})
	sessionID := strings.TrimSpace(opts.ResumeSessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("acp-session-%d", d.nextSess)
	}

	var proc *session.AgentProcess
	proc = session.NewAgentProcess(session.AgentProcessOptions{
		PID:       d.nextPID,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: sessionID,
		Caps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModels:     []string{"fake-model"},
		},
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
		ApprovePermission: func(ctx context.Context, req acp.ApproveRequest) error {
			d.mu.Lock()
			approvalCh := d.approvals[proc]
			d.mu.Unlock()
			if approvalCh == nil {
				return session.ErrPendingPermissionNotFound
			}

			select {
			case approvalCh <- req:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	if strings.Contains(req.Message, "request permission") {
		events := make(chan acp.AgentEvent, 6)
		requestID := req.TurnID + ":tool-1"
		approvalCh := make(chan acp.ApproveRequest, 1)

		d.mu.Lock()
		d.approvals[proc] = approvalCh
		d.mu.Unlock()

		go func() {
			defer close(events)
			defer func() {
				d.mu.Lock()
				delete(d.approvals, proc)
				d.mu.Unlock()
			}()

			raw := mustIntegrationJSON(tPermissionRaw(requestID))
			ts := time.Now().UTC()
			events <- acp.AgentEvent{
				Type:       "permission",
				SessionID:  proc.SessionID,
				TurnID:     req.TurnID,
				RequestID:  requestID,
				Timestamp:  ts,
				Title:      "permission request",
				ToolCallID: "tool-1",
				Action:     "session/request_permission",
				Resource:   "/tmp/demo.txt",
				Raw:        raw,
			}

			finalDecision := "reject-once"
			select {
			case approval := <-approvalCh:
				finalDecision = approval.Decision
			case <-time.After(d.permissionWait):
			}

			ts = time.Now().UTC()
			events <- acp.AgentEvent{
				Type:       "permission",
				SessionID:  proc.SessionID,
				TurnID:     req.TurnID,
				RequestID:  requestID,
				Timestamp:  ts,
				Title:      "permission request",
				ToolCallID: "tool-1",
				Action:     "session/request_permission",
				Resource:   "/tmp/demo.txt",
				Decision:   finalDecision,
				Raw:        mustIntegrationJSON(tPermissionRawWithDecision(requestID, finalDecision)),
			}
			events <- acp.AgentEvent{
				Type:      "agent_message",
				SessionID: proc.SessionID,
				TurnID:    req.TurnID,
				Timestamp: ts,
				Text:      finalDecision,
			}
			events <- acp.AgentEvent{
				Type:       "done",
				SessionID:  proc.SessionID,
				TurnID:     req.TurnID,
				Timestamp:  ts,
				StopReason: "end_turn",
			}
		}()
		return events, nil
	}

	ch := make(chan acp.AgentEvent, 4)
	ch <- acp.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
	}
	ch <- acp.AgentEvent{
		Type:       "tool_call",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		Title:      "read_file",
		ToolCallID: "call-1",
	}
	ch <- acp.AgentEvent{
		Type:       "tool_result",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		ToolCallID: "call-1",
	}
	ch <- acp.AgentEvent{
		Type:       "done",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		StopReason: "end_turn",
	}
	close(ch)
	return ch, nil
}

func (d *integrationDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *integrationDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	done, ok := d.states[proc]
	if !ok {
		return nil
	}
	select {
	case <-done:
	default:
		close(done)
	}
	delete(d.states, proc)
	delete(d.approvals, proc)
	return nil
}

func newIntegrationRuntime(t *testing.T) integrationRuntime {
	return newIntegrationRuntimeWithPermissionWait(t, 100*time.Millisecond)
}

func newIntegrationRuntimeWithPermissionWait(t *testing.T, permissionWait time.Duration) integrationRuntime {
	t.Helper()

	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")

	workspace := t.TempDir()
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)
	cfg.Providers = map[string]aghconfig.ProviderConfig{
		"fake": {Command: "fake-agent"},
	}

	registry, err := globaldb.OpenGlobalDB(context.Background(), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := registry.Close(context.Background()); err != nil {
			t.Fatalf("registry.Close() error = %v", err)
		}
	})

	fanout := &integrationNotifierFanout{}
	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) { return cfg, nil }),
	)
	if err != nil {
		t.Fatalf("workspace.NewResolver() error = %v", err)
	}
	manager, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithWorkspaceResolver(resolver),
		session.WithLogger(discardLogger()),
		session.WithDriver(newIntegrationDriver(permissionWait)),
		session.WithNotifier(fanout),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}

	observer, err := observe.New(
		context.Background(),
		observe.WithHomePaths(homePaths),
		observe.WithRegistry(registry),
		observe.WithSessionSource(manager),
		observe.WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	fanout.notifiers = append(fanout.notifiers, observer)

	memoryStore := memory.NewStore(homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		t.Fatalf("memoryStore.EnsureDirs() error = %v", err)
	}
	dreamTrigger := &integrationDreamTrigger{
		enabled:   true,
		triggered: true,
		last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(manager),
		WithObserver(observer),
		WithWorkspaceResolver(resolver),
		WithMemoryStore(memoryStore),
		WithDreamTrigger(dreamTrigger),
		WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("httpapi.New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("server.Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Fatalf("server.Shutdown() error = %v", err)
		}
	})

	return integrationRuntime{
		client:    &http.Client{},
		server:    server,
		manager:   manager,
		observer:  observer,
		memory:    memoryStore,
		dream:     dreamTrigger,
		host:      cfg.HTTP.Host,
		port:      server.Port(),
		workspace: workspace,
	}
}

type permissionStreamPayload struct {
	RequestID string `json:"request_id"`
	Decision  string `json:"decision,omitempty"`
}

func streamPermissionPrompt(t *testing.T, body io.ReadCloser, requestIDCh chan<- string, resultCh chan<- []sseRecord) {
	t.Helper()
	defer func() {
		_ = body.Close()
	}()

	scanner := bufio.NewScanner(body)
	records := make([]sseRecord, 0, 8)
	current := sseRecord{}
	requestIDSent := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
				records = append(records, current)
				if !requestIDSent {
					if payload, ok := extractPermissionPayloadFromRecord(current); ok && payload.Decision == "" && payload.RequestID != "" {
						requestIDCh <- payload.RequestID
						requestIDSent = true
					}
				}
			}
			current = sseRecord{}
			continue
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			current.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
		records = append(records, current)
	}
	if !requestIDSent {
		requestIDCh <- ""
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan prompt SSE error = %v", err)
	}
	resultCh <- records
}

func extractPermissionPayloads(t *testing.T, records []sseRecord) []permissionStreamPayload {
	t.Helper()

	payloads := make([]permissionStreamPayload, 0, len(records))
	for _, record := range records {
		if payload, ok := extractPermissionPayloadFromRecord(record); ok {
			payloads = append(payloads, payload)
		}
	}
	return payloads
}

func extractPermissionPayloadFromRecord(record sseRecord) (permissionStreamPayload, bool) {
	if record.Event != "permission" || len(record.Data) == 0 {
		return permissionStreamPayload{}, false
	}

	var envelope struct {
		Type string                  `json:"type"`
		Data permissionStreamPayload `json:"data"`
	}
	if err := json.Unmarshal(record.Data, &envelope); err != nil || envelope.Type != "data-agh-permission" {
		return permissionStreamPayload{}, false
	}
	return envelope.Data, true
}

func recordsContainTextDelta(records []sseRecord, want string) bool {
	for _, record := range records {
		if record.Event != "agent_message" || len(record.Data) == 0 {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(record.Data, &payload); err != nil {
			continue
		}
		if payload["type"] == "text-delta" && payload["delta"] == want {
			return true
		}
	}
	return false
}

func tPermissionRaw(requestID string) map[string]any {
	return map[string]any{
		"request_id": requestID,
		"tool_input": map[string]any{"command": "demo"},
		"options": []map[string]any{
			{"decision": "allow-once", "kind": "allow_once", "option_id": "allow-once", "label": "allow once"},
			{"decision": "allow-always", "kind": "allow_always", "option_id": "allow-always", "label": "allow always"},
			{"decision": "reject-once", "kind": "reject_once", "option_id": "reject-once", "label": "reject once"},
			{"decision": "reject-always", "kind": "reject_always", "option_id": "reject-always", "label": "reject always"},
		},
	}
}

func tPermissionRawWithDecision(requestID string, decision string) map[string]any {
	payload := tPermissionRaw(requestID)
	payload["decision"] = decision
	return payload
}

func mustIntegrationJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func createIntegrationSession(t *testing.T, runtime integrationRuntime) string {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions"), []byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`), nil)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, string(body))
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, resp, &created)
	return created.Session.ID
}

func sendPrompt(t *testing.T, runtime integrationRuntime, sessionID string, message string) {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/prompt"), []byte(`{"message":"`+message+`"}`), nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
}

func stopIntegrationSession(t *testing.T, runtime integrationRuntime, sessionID string) {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodDelete, mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("stop status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}
	_ = resp.Body.Close()
}
