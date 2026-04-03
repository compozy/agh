//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func TestHTTPFullRoundTripWithRealSessionManager(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	statusResp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/daemon/status"), nil, nil)
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", statusResp.StatusCode, http.StatusOK)
	}
	_ = statusResp.Body.Close()

	createResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions"), []byte(`{"agent_name":"coder","name":"demo","workspace":"`+runtime.workspace+`"}`), map[string]string{"Origin": "https://example.com"})
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	if got := createResp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
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
	initial := collectLiveSSE(t, streamResp.Body, 5, 2*time.Second)
	_ = streamResp.Body.Close()
	if len(initial) < 5 {
		t.Fatalf("initial stream events = %d, want 5", len(initial))
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
	replayed := collectLiveSSE(t, replayResp.Body, 4, 2*time.Second)
	_ = replayResp.Body.Close()
	if len(replayed) < 4 {
		t.Fatalf("replayed events = %d, want 4", len(replayed))
	}
	if replayed[0].ID != initial[1].ID {
		t.Fatalf("replayed first id = %q, want %q", replayed[0].ID, initial[1].ID)
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
			listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
				entered <- struct{}{}
				<-release
				return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
			},
		}),
		WithObserver(stubObserver{
			healthFn: func(context.Context) (observe.Health, error) { return observe.Health{Status: "ok"}, nil },
		}),
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
	host      string
	port      int
	workspace string
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

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event session.AgentEvent) {
	for _, notifier := range f.notifiers {
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

type integrationDriver struct {
	mu       sync.Mutex
	nextPID  int
	nextSess int
	states   map[*session.AgentProcess]chan struct{}
}

func newIntegrationDriver() *integrationDriver {
	return &integrationDriver{
		nextPID:  2000,
		nextSess: 1,
		states:   make(map[*session.AgentProcess]chan struct{}),
	}
}

func (d *integrationDriver) Start(_ context.Context, opts session.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextPID++
	d.nextSess++
	done := make(chan struct{})
	sessionID := strings.TrimSpace(opts.ResumeSessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("acp-session-%d", d.nextSess)
	}

	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       d.nextPID,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: sessionID,
		Caps: session.ACPCaps{
			SupportsLoadSession: true,
			SupportedModels:     []string{"fake-model"},
		},
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req session.PromptRequest) (<-chan session.AgentEvent, error) {
	ch := make(chan session.AgentEvent, 4)
	ch <- session.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
	}
	ch <- session.AgentEvent{
		Type:       "tool_call",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		Title:      "read_file",
		ToolCallID: "call-1",
	}
	ch <- session.AgentEvent{
		Type:       "tool_result",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		ToolCallID: "call-1",
	}
	ch <- session.AgentEvent{
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
	return nil
}

func newIntegrationRuntime(t *testing.T) integrationRuntime {
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

	registry, err := store.OpenGlobalDB(context.Background(), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := registry.Close(context.Background()); err != nil {
			t.Fatalf("registry.Close() error = %v", err)
		}
	})

	fanout := &integrationNotifierFanout{}
	manager, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithConfig(cfg),
		session.WithLogger(discardLogger()),
		session.WithDriver(newIntegrationDriver()),
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

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(manager),
		WithObserver(observer),
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
		host:      cfg.HTTP.Host,
		port:      server.Port(),
		workspace: workspace,
	}
}

func createIntegrationSession(t *testing.T, runtime integrationRuntime) string {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions"), []byte(`{"agent_name":"coder","workspace":"`+runtime.workspace+`"}`), nil)
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
