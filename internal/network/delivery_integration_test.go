//go:build integration

package network

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestDeliveryCoordinatorIntegrationDrainsOneQueuedPromptPerTurn(t *testing.T) {
	manager, driver := newDeliveryIntegrationHarness(t)
	networked := createIntegrationSession(t, manager, "coder")

	ctx, cancel := context.WithCancel(testutil.Context(t))
	defer cancel()

	coordinator, err := newDeliveryCoordinator(ctx, 8, manager)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}
	manager.SetTurnEndNotifier(coordinator.onTurnEnd)

	userEvents, err := manager.Prompt(testutil.Context(t), networked.ID, "user turn")
	if err != nil {
		t.Fatalf("Prompt(user) error = %v", err)
	}
	go drainAgentEvents(userEvents)

	driver.waitForPromptCount(t, 1)

	if err := coordinator.accept(testutil.Context(t), []Delivery{
		{SessionID: networked.ID, Envelope: testDeliveryEnvelope(t, "msg-1", "first queued")},
		{SessionID: networked.ID, Envelope: testDeliveryEnvelope(t, "msg-2", "second queued")},
	}); err != nil {
		t.Fatalf("accept(queued) error = %v", err)
	}
	if got := coordinator.queueDepth(networked.ID); got != 2 {
		t.Fatalf("queueDepth() after busy accept = %d, want 2", got)
	}
	if got := driver.promptCount(); got != 1 {
		t.Fatalf("promptCount() before turn end = %d, want 1", got)
	}

	driver.completePrompt(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	driver.waitForPromptCount(t, 2)

	firstNetwork := driver.prompt(1)
	if !strings.Contains(firstNetwork.req.Message, "first queued") {
		t.Fatalf("first network prompt message = %q, want first queued preview", firstNetwork.req.Message)
	}
	if got := coordinator.queueDepth(networked.ID); got != 1 {
		t.Fatalf("queueDepth() after first drain = %d, want 1", got)
	}
	assertPromptCountEventually(t, driver, 2)

	driver.completePrompt(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	driver.waitForPromptCount(t, 3)

	secondNetwork := driver.prompt(2)
	if !strings.Contains(secondNetwork.req.Message, "second queued") {
		t.Fatalf("second network prompt message = %q, want second queued preview", secondNetwork.req.Message)
	}

	driver.completePrompt(2, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	waitForDeliveryCondition(t, "queue drain", func() bool { return coordinator.queueDepth(networked.ID) == 0 })

	cancel()
	coordinator.wait()
}

func TestDeliveryCoordinatorIntegrationMultipleSessionsDoNotBlockEachOther(t *testing.T) {
	manager, driver := newDeliveryIntegrationHarness(t)
	sessionA := createIntegrationSession(t, manager, "coder")
	sessionB := createIntegrationSession(t, manager, "coder")

	ctx, cancel := context.WithCancel(testutil.Context(t))
	defer cancel()

	coordinator, err := newDeliveryCoordinator(ctx, 8, manager)
	if err != nil {
		t.Fatalf("newDeliveryCoordinator() error = %v", err)
	}
	manager.SetTurnEndNotifier(coordinator.onTurnEnd)

	userEventsA, err := manager.Prompt(testutil.Context(t), sessionA.ID, "user turn A")
	if err != nil {
		t.Fatalf("Prompt(user A) error = %v", err)
	}
	userEventsB, err := manager.Prompt(testutil.Context(t), sessionB.ID, "user turn B")
	if err != nil {
		t.Fatalf("Prompt(user B) error = %v", err)
	}
	go drainAgentEvents(userEventsA)
	go drainAgentEvents(userEventsB)

	driver.waitForPromptCount(t, 2)

	if err := coordinator.accept(testutil.Context(t), []Delivery{
		{SessionID: sessionA.ID, Envelope: testDeliveryEnvelope(t, "msg-a", "for session A")},
		{SessionID: sessionB.ID, Envelope: testDeliveryEnvelope(t, "msg-b", "for session B")},
	}); err != nil {
		t.Fatalf("accept(multi-session) error = %v", err)
	}

	driver.completePrompt(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	driver.waitForPromptCount(t, 3)

	networkA := driver.prompt(2)
	if got, want := networkA.sessionID, sessionA.Info().ACPSessionID; got != want {
		t.Fatalf("network A ACP session id = %q, want %q", got, want)
	}
	if !strings.Contains(networkA.req.Message, "for session A") {
		t.Fatalf("network A prompt message = %q, want session A preview", networkA.req.Message)
	}

	driver.completePrompt(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	driver.waitForPromptCount(t, 4)

	networkB := driver.prompt(3)
	if got, want := networkB.sessionID, sessionB.Info().ACPSessionID; got != want {
		t.Fatalf("network B ACP session id = %q, want %q", got, want)
	}
	if !strings.Contains(networkB.req.Message, "for session B") {
		t.Fatalf("network B prompt message = %q, want session B preview", networkB.req.Message)
	}

	driver.completePrompt(2, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	driver.completePrompt(3, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
	waitForDeliveryCondition(t, "multi-session queue drain", func() bool {
		return coordinator.queueDepth(sessionA.ID) == 0 && coordinator.queueDepth(sessionB.ID) == 0
	})

	cancel()
	coordinator.wait()
}

func newDeliveryIntegrationHarness(t *testing.T) (*session.Manager, *integrationPromptDriver) {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	workspaceRoot := filepath.Join(homePaths.HomeDir, "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Providers["claude"] = aghconfig.ProviderConfig{Command: "fake-agent"}
	resolver := integrationWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-primary",
				RootDir: workspaceRoot,
				Name:    "workspace",
			},
			Config: cfg,
			Agents: []aghconfig.AgentDef{
				{
					Name:     aghconfig.DefaultAgentName,
					Provider: "claude",
					Prompt:   "You are a coding assistant.",
				},
				{
					Name:     "coder",
					Provider: "claude",
					Prompt:   "You are a coding assistant.",
				},
			},
		},
	}

	driver := newIntegrationPromptDriver()
	manager, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithWorkspaceResolver(resolver),
		session.WithDriver(driver),
		session.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}

	return manager, driver
}

func createIntegrationSession(t *testing.T, manager *session.Manager, agentName string) *session.Session {
	t.Helper()

	sess, err := manager.Create(testutil.Context(t), session.CreateOpts{
		AgentName: agentName,
		Workspace: "ws-primary",
	})
	if err != nil {
		t.Fatalf("manager.Create() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t), sess.ID); err != nil {
			t.Fatalf("cleanup Stop(%s) error = %v", sess.ID, err)
		}
	})
	return sess
}

type integrationWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
}

func (r integrationWorkspaceResolver) Resolve(_ context.Context, _ string) (workspacepkg.ResolvedWorkspace, error) {
	return r.resolved, nil
}

func (r integrationWorkspaceResolver) ResolveOrRegister(_ context.Context, _ string) (workspacepkg.ResolvedWorkspace, error) {
	return r.resolved, nil
}

type integrationPromptDriver struct {
	mu        sync.Mutex
	prompts   []*integrationPrompt
	processes map[*session.AgentProcess]chan struct{}
}

type integrationPrompt struct {
	sessionID string
	req       acp.PromptRequest
	events    chan acp.AgentEvent
}

func newIntegrationPromptDriver() *integrationPromptDriver {
	return &integrationPromptDriver{
		processes: make(map[*session.AgentProcess]chan struct{}),
	}
}

func (d *integrationPromptDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	done := make(chan struct{})
	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       len(d.processes) + 1,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: "acp-" + opts.AgentName + "-" + strconv.Itoa(len(d.processes)+1),
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
	})
	d.processes[proc] = done
	return proc, nil
}

func (d *integrationPromptDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	events := make(chan acp.AgentEvent, 8)
	d.prompts = append(d.prompts, &integrationPrompt{
		sessionID: proc.SessionID,
		req:       req,
		events:    events,
	})
	return events, nil
}

func (d *integrationPromptDriver) Cancel(_ context.Context, _ *session.AgentProcess) error {
	return nil
}

func (d *integrationPromptDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	done := d.processes[proc]
	delete(d.processes, proc)
	d.mu.Unlock()

	if done != nil {
		close(done)
	}
	return nil
}

func (d *integrationPromptDriver) promptCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.prompts)
}

func (d *integrationPromptDriver) waitForPromptCount(t *testing.T, want int) {
	t.Helper()
	waitForDeliveryCondition(t, "prompt count", func() bool { return d.promptCount() >= want })
}

func (d *integrationPromptDriver) prompt(index int) integrationPrompt {
	d.mu.Lock()
	defer d.mu.Unlock()
	item := d.prompts[index]
	return integrationPrompt{
		sessionID: item.sessionID,
		req:       item.req,
	}
}

func (d *integrationPromptDriver) completePrompt(index int, events ...acp.AgentEvent) {
	d.mu.Lock()
	prompt := d.prompts[index]
	d.mu.Unlock()

	for _, event := range events {
		prompt.events <- event
	}
	close(prompt.events)
}

func assertPromptCountEventually(t *testing.T, driver *integrationPromptDriver, want int) {
	t.Helper()

	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := driver.promptCount(); got != want {
			t.Fatalf("promptCount() changed early: got %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForDeliveryCondition(t *testing.T, label string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", label)
}

func drainAgentEvents(events <-chan acp.AgentEvent) {
	for range events {
	}
}
