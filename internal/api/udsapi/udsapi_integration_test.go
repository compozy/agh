//go:build integration

package udsapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store/globaldb"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestUDSFullRoundTripWithRealSessionManager(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	statusResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/daemon/status", nil, nil)
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", statusResp.StatusCode, http.StatusOK)
	}
	_ = statusResp.Body.Close()

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions", []byte(`{"agent_name":"coder","name":"demo","workspace_path":"`+runtime.workspace+`"}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create session status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created struct {
		Session sessionPayload `json:"session"`
	}
	decodeHTTPJSON(t, createResp, &created)
	if created.Session.ID == "" {
		t.Fatal("expected created session id")
	}

	listResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions", nil, nil)
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

	promptResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions/"+created.Session.ID+"/prompt", []byte(`{"message":"hello"}`), nil)
	if promptResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(promptResp.Body)
		_ = promptResp.Body.Close()
		t.Fatalf("prompt status = %d, want %d; body=%s", promptResp.StatusCode, http.StatusOK, string(body))
	}
	promptBody, err := io.ReadAll(promptResp.Body)
	_ = promptResp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(prompt SSE) error = %v", err)
	}
	promptEvents := parseSSE(t, string(promptBody))
	if len(promptEvents) < 2 {
		t.Fatalf("prompt SSE events = %d, want at least 2; body=%s", len(promptEvents), string(promptBody))
	}
	if promptEvents[0].Event != "agent_message" || promptEvents[len(promptEvents)-1].Event != "done" {
		t.Fatalf("prompt SSE event types = first:%s last:%s", promptEvents[0].Event, promptEvents[len(promptEvents)-1].Event)
	}

	eventsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+created.Session.ID+"/events", nil, nil)
	if eventsResp.StatusCode != http.StatusOK {
		t.Fatalf("session events status = %d, want %d", eventsResp.StatusCode, http.StatusOK)
	}
	var events struct {
		Events []sessionEventPayload `json:"events"`
	}
	decodeHTTPJSON(t, eventsResp, &events)
	if len(events.Events) < 2 {
		t.Fatalf("persisted session events = %d, want at least 2", len(events.Events))
	}

	stopResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/sessions/"+created.Session.ID, nil, nil)
	if stopResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(stopResp.Body)
		_ = stopResp.Body.Close()
		t.Fatalf("stop session status = %d, want %d; body=%s", stopResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = stopResp.Body.Close()
}

func TestUDSMemoryRoundTripAndConsolidate(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	writeResp := mustUnixRequest(t, runtime.client, http.MethodPut, "http://unix/api/memory/integration.md", []byte(`{"scope":"global","content":"`+escapeJSON(memoryDocument(t, "Integration", "desc", memory.MemoryTypeUser, "hello integration"))+`"}`), nil)
	if writeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(writeResp.Body)
		_ = writeResp.Body.Close()
		t.Fatalf("write status = %d, want %d; body=%s", writeResp.StatusCode, http.StatusOK, string(body))
	}
	_ = writeResp.Body.Close()

	readResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/memory/integration.md?scope=global", nil, nil)
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

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/memory/integration.md?scope=global", nil, nil)
	if deleteResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusOK, string(body))
	}
	_ = deleteResp.Body.Close()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/memory/consolidate", []byte(`{"workspace":"`+runtime.workspace+`"}`), nil)
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

func TestUDSAutomationJobsRoundTrip(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/jobs", []byte(`{"scope":"global","name":"nightly-review","agent_name":"coder","prompt":"review repo","schedule":{"mode":"every","interval":"1h"}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create job status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.JobResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Job.ID == "" {
		t.Fatal("expected created automation job id")
	}

	updateResp := mustUnixRequest(t, runtime.client, http.MethodPatch, "http://unix/api/automation/jobs/"+created.Job.ID, []byte(`{"prompt":"review repo now"}`), nil)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update job status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.JobResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Job.Prompt != "review repo now" {
		t.Fatalf("updated job prompt = %q, want %q", updated.Job.Prompt, "review repo now")
	}

	triggerResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/jobs/"+created.Job.ID+"/trigger", nil, nil)
	if triggerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(triggerResp.Body)
		_ = triggerResp.Body.Close()
		t.Fatalf("trigger job status = %d, want %d; body=%s", triggerResp.StatusCode, http.StatusOK, string(body))
	}
	var run contract.RunResponse
	decodeHTTPJSON(t, triggerResp, &run)
	if run.Run.JobID != created.Job.ID {
		t.Fatalf("job run = %#v, want job_id %q", run.Run, created.Job.ID)
	}

	runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/jobs/"+created.Job.ID+"/runs", nil, nil)
	if runsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(runsResp.Body)
		_ = runsResp.Body.Close()
		t.Fatalf("job runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
	}
	var runs contract.RunsResponse
	decodeHTTPJSON(t, runsResp, &runs)
	if !containsAutomationRun(runs.Runs, run.Run.ID) {
		t.Fatalf("job runs missing %q: %#v", run.Run.ID, runs.Runs)
	}

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/automation/jobs/"+created.Job.ID, nil, nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete job status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = deleteResp.Body.Close()
}

func TestUDSAutomationTriggerRunsAndOmitsWebhookRoutes(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	resolveResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/workspaces/resolve", []byte(`{"path":"`+runtime.workspace+`"}`), nil)
	if resolveResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resolveResp.Body)
		_ = resolveResp.Body.Close()
		t.Fatalf("resolve workspace status = %d, want %d; body=%s", resolveResp.StatusCode, http.StatusOK, string(body))
	}
	var resolved contract.WorkspaceResponse
	decodeHTTPJSON(t, resolveResp, &resolved)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/automation/triggers", []byte(`{"scope":"workspace","workspace_id":"`+resolved.Workspace.ID+`","name":"session-stop-review","agent_name":"coder","prompt":"review {{ index .Data \"session_id\" }}","event":"session.stopped","filter":{"data.session_type":"user"}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		_ = createResp.Body.Close()
		t.Fatalf("create trigger status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, string(body))
	}
	var created contract.TriggerResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Trigger.ID == "" {
		t.Fatal("expected created automation trigger id")
	}

	updateResp := mustUnixRequest(t, runtime.client, http.MethodPatch, "http://unix/api/automation/triggers/"+created.Trigger.ID, []byte(`{"prompt":"inspect {{ index .Data \"session_id\" }}"}`), nil)
	if updateResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(updateResp.Body)
		_ = updateResp.Body.Close()
		t.Fatalf("update trigger status = %d, want %d; body=%s", updateResp.StatusCode, http.StatusOK, string(body))
	}
	var updated contract.TriggerResponse
	decodeHTTPJSON(t, updateResp, &updated)
	if updated.Trigger.Prompt != `inspect {{ index .Data "session_id" }}` {
		t.Fatalf("updated trigger prompt = %q", updated.Trigger.Prompt)
	}

	sessionID := createIntegrationSession(t, runtime)
	stopIntegrationSession(t, runtime, sessionID)

	var runs contract.RunsResponse
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs", nil, nil)
		if runsResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(runsResp.Body)
			_ = runsResp.Body.Close()
			t.Fatalf("trigger runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
		}

		runs = contract.RunsResponse{}
		decodeHTTPJSON(t, runsResp, &runs)
		if len(runs.Runs) > 0 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("expected trigger run history, got %#v", runs.Runs)
		case <-ticker.C:
		}
	}
	runID := runs.Runs[0].ID

	runResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/runs/"+runID, nil, nil)
	if runResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(runResp.Body)
		_ = runResp.Body.Close()
		t.Fatalf("get trigger run status = %d, want %d; body=%s", runResp.StatusCode, http.StatusOK, string(body))
	}
	var run contract.RunResponse
	decodeHTTPJSON(t, runResp, &run)
	if run.Run.TriggerID != created.Trigger.ID {
		t.Fatalf("trigger run = %#v, want trigger_id %q", run.Run, created.Trigger.ID)
	}

	webhookResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/webhooks/global/deploy-review--wbh_test", []byte(`{"payload":"deploy"}`), nil)
	if webhookResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(webhookResp.Body)
		_ = webhookResp.Body.Close()
		t.Fatalf("webhook route status = %d, want %d; body=%s", webhookResp.StatusCode, http.StatusNotFound, string(body))
	}
	_ = webhookResp.Body.Close()

	deleteResp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/automation/triggers/"+created.Trigger.ID, nil, nil)
	if deleteResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(deleteResp.Body)
		_ = deleteResp.Body.Close()
		t.Fatalf("delete trigger status = %d, want %d; body=%s", deleteResp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = deleteResp.Body.Close()
}

func TestUDSSessionStreamReconnectsWithLastEventID(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	sessionID := createIntegrationSession(t, runtime)
	sendPrompt(t, runtime, sessionID, "hello")
	stopIntegrationSession(t, runtime, sessionID)

	streamResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+sessionID+"/stream", nil, nil)
	if streamResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(streamResp.Body)
		_ = streamResp.Body.Close()
		t.Fatalf("session stream status = %d, want %d; body=%s", streamResp.StatusCode, http.StatusOK, string(body))
	}
	initial := collectLiveSSEUntilEvent(t, streamResp.Body, session.EventTypeSessionStopped, 2*time.Second)
	_ = streamResp.Body.Close()
	if len(initial) < 3 {
		t.Fatalf("initial stream events = %d, want 3", len(initial))
	}
	if initial[len(initial)-1].Event != session.EventTypeSessionStopped {
		t.Fatalf("last event = %q, want %q", initial[len(initial)-1].Event, session.EventTypeSessionStopped)
	}

	headers := map[string]string{"Last-Event-ID": initial[0].ID}
	replayResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/sessions/"+sessionID+"/stream", nil, headers)
	if replayResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(replayResp.Body)
		_ = replayResp.Body.Close()
		t.Fatalf("replay stream status = %d, want %d; body=%s", replayResp.StatusCode, http.StatusOK, string(body))
	}
	replayed := collectLiveSSE(t, replayResp.Body, 2, 2*time.Second)
	_ = replayResp.Body.Close()
	if len(replayed) < 2 {
		t.Fatalf("replayed events = %d, want 2", len(replayed))
	}
	if replayed[0].ID != initial[1].ID {
		t.Fatalf("replayed first id = %q, want %q", replayed[0].ID, initial[1].ID)
	}
	if replayed[0].ID == initial[0].ID {
		t.Fatalf("replayed first id = %q, should skip Last-Event-ID", replayed[0].ID)
	}
}

func TestUDSShutdownWaitsForInflightRequests(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithSocketPath(socketPath),
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

	client := newUnixClient(t, socketPath)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := client.Get("http://unix/api/sessions")
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
	registry  *globaldb.GlobalDB
	channels  *integrationChannelService
	memory    *memory.Store
	dream     *integrationDreamTrigger
	socket    string
	workspace string
}

type integrationDreamTrigger struct {
	enabled   bool
	triggered bool
	reason    string
	last      time.Time
	calls     int
}

type integrationChannelService struct {
	*channelspkg.Service
}

var _ core.ChannelService = (*integrationChannelService)(nil)

func newIntegrationChannelService(store channelspkg.RegistryStore) *integrationChannelService {
	return &integrationChannelService{Service: channelspkg.NewRegistry(store)}
}

func (s *integrationChannelService) StartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if _, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusStarting,
	}); err != nil {
		return nil, fmt.Errorf("start channel instance %q: %w", id, err)
	}
	instance, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusReady,
	})
	if err != nil {
		return nil, fmt.Errorf("mark channel instance %q ready: %w", id, err)
	}
	return instance, nil
}

func (s *integrationChannelService) StopInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	instance, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: false,
		Status:  channelspkg.ChannelStatusDisabled,
	})
	if err != nil {
		return nil, fmt.Errorf("stop channel instance %q: %w", id, err)
	}
	return instance, nil
}

func (s *integrationChannelService) RestartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if _, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusStarting,
	}); err != nil {
		return nil, fmt.Errorf("restart channel instance %q: %w", id, err)
	}
	instance, err := s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusReady,
	})
	if err != nil {
		return nil, fmt.Errorf("mark restarted channel instance %q ready: %w", id, err)
	}
	return instance, nil
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

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event any) {
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

	proc := session.NewAgentProcess(session.AgentProcessOptions{
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
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	ch := make(chan acp.AgentEvent, 2)
	ch <- acp.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
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
	return nil
}

func newIntegrationRuntime(t *testing.T) integrationRuntime {
	t.Helper()

	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	writeAgentDef(t, homePaths, "coder")

	workspace := t.TempDir()
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath
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

	memoryStore := memory.NewStore(homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		t.Fatalf("memoryStore.EnsureDirs() error = %v", err)
	}
	channelService := newIntegrationChannelService(registry)
	dreamTrigger := &integrationDreamTrigger{
		enabled:   true,
		triggered: true,
		last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}

	automationManager, err := automationpkg.New(
		automationpkg.WithStore(registry),
		automationpkg.WithSessions(manager),
		automationpkg.WithWorkspaceResolver(resolver),
		automationpkg.WithConfig(cfg.Automation),
		automationpkg.WithLogger(discardLogger()),
		automationpkg.WithGlobalWorkspacePath(homePaths.HomeDir),
	)
	if err != nil {
		t.Fatalf("automation.New() error = %v", err)
	}
	if err := automationManager.Start(context.Background()); err != nil {
		t.Fatalf("automationManager.Start() error = %v", err)
	}
	fanout.notifiers = append(fanout.notifiers, automationManager.SessionObserver())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := automationManager.Shutdown(ctx); err != nil {
			t.Fatalf("automationManager.Shutdown() error = %v", err)
		}
	})

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(manager),
		WithObserver(observer),
		WithAutomation(automationManager),
		WithChannelService(channelService),
		WithWorkspaceResolver(resolver),
		WithMemoryStore(memoryStore),
		WithDreamTrigger(dreamTrigger),
		WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("udsapi.New() error = %v", err)
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
		client:    newUnixClient(t, socketPath),
		server:    server,
		manager:   manager,
		observer:  observer,
		registry:  registry,
		channels:  channelService,
		memory:    memoryStore,
		dream:     dreamTrigger,
		socket:    socketPath,
		workspace: workspace,
	}
}

func createIntegrationSession(t *testing.T, runtime integrationRuntime) string {
	t.Helper()

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions", []byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`), nil)
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

	resp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/sessions/"+sessionID+"/prompt", []byte(`{"message":"`+message+`"}`), nil)
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

	resp := mustUnixRequest(t, runtime.client, http.MethodDelete, "http://unix/api/sessions/"+sessionID, nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("stop status = %d, want %d; body=%s", resp.StatusCode, http.StatusNoContent, string(body))
	}
	_ = resp.Body.Close()
}

func mustUnixRequest(t *testing.T, client *http.Client, method, url string, body []byte, headers map[string]string) *http.Response {
	t.Helper()

	var reader io.Reader
	if len(body) > 0 {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	return resp
}

func decodeHTTPJSON(t *testing.T, resp *http.Response, dest any) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll(response) error = %v", err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, string(body))
	}
}

func containsAutomationRun(runs []contract.RunPayload, id string) bool {
	for _, run := range runs {
		if run.ID == id {
			return true
		}
	}
	return false
}

func collectLiveSSE(t *testing.T, body io.ReadCloser, want int, timeout time.Duration) []sseRecord {
	t.Helper()

	records := make([]sseRecord, 0, want)
	recordCh := make(chan sseRecord, want+1)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(body)
		current := sseRecord{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				recordCh <- current
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
			recordCh <- current
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
			errCh <- err
			return
		}
		close(recordCh)
	}()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for len(records) < want {
		select {
		case record, ok := <-recordCh:
			if !ok {
				return records
			}
			records = append(records, record)
		case err := <-errCh:
			t.Fatalf("SSE scanner error = %v", err)
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d SSE events; got %d", want, len(records))
		}
	}

	return records
}

func collectLiveSSEUntilEvent(t *testing.T, body io.ReadCloser, wantEvent string, timeout time.Duration) []sseRecord {
	t.Helper()

	records := make([]sseRecord, 0, 4)
	recordCh := make(chan sseRecord, 8)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(body)
		current := sseRecord{}
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				recordCh <- current
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
			recordCh <- current
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, os.ErrClosed) {
			errCh <- err
			return
		}
		close(recordCh)
	}()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case record, ok := <-recordCh:
			if !ok {
				t.Fatalf("stream ended before event %q; got %d records", wantEvent, len(records))
			}
			records = append(records, record)
			if record.Event == wantEvent {
				return records
			}
		case err := <-errCh:
			t.Fatalf("SSE scanner error = %v", err)
		case <-deadline.C:
			t.Fatalf("timed out waiting for SSE event %q; got %d records", wantEvent, len(records))
		}
	}
}
