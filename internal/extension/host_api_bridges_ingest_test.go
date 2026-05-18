package extensionpkg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestHostAPIHandlerBridgesMessagesIngestContract(t *testing.T) {
	t.Parallel()

	t.Run("Should suppress webhook retry after request cancellation launches detached prompt", func(t *testing.T) {
		t.Parallel()

		env := newHostAPITestEnv(t)
		env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

		releasePrompt := make(chan struct{})
		releaseOnce := sync.Once{}
		t.Cleanup(func() {
			releaseOnce.Do(func() { close(releasePrompt) })
		})
		driver := newContractBridgePromptDriver(env.currentTime(), releasePrompt)
		broker := &recordingPromptDeliveryBroker{}
		env.useContractBridgePromptDriver(t, driver, broker)

		instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
			ID:            "brg-ingest-cancel-dedup",
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
		})
		bridgeCtx := env.bridgeContext(t, instance)
		params := map[string]any{
			"bridge_instance_id":  instance.ID,
			"scope":               instance.Scope,
			"workspace_id":        instance.WorkspaceID,
			"peer_id":             "peer-1",
			"platform_message_id": "msg-cancel-dedup",
			"received_at":         env.currentTime().Format(time.RFC3339Nano),
			"idempotency_key":     "idem-cancel-dedup",
			"content":             map[string]any{"text": "hello"},
		}
		raw, err := marshalParams(params)
		if err != nil {
			t.Fatalf("marshalParams() error = %v", err)
		}

		firstCtx, firstCancel := context.WithCancel(bridgeCtx)
		defer firstCancel()
		firstDone := make(chan error, 1)
		go func() {
			_, callErr := env.handler.Handle(firstCtx, "telegram-adapter", "bridges/messages/ingest", raw)
			firstDone <- callErr
		}()

		select {
		case <-driver.promptStarted:
		case err := <-firstDone:
			t.Fatalf("first ingest finished before prompt started: %v", err)
		case <-time.After(time.Second):
			t.Fatal("first prompt did not start before timeout")
		}
		firstCancel()

		retryDone := make(chan error, 1)
		go func() {
			_, callErr := env.handler.Handle(bridgeCtx, "telegram-adapter", "bridges/messages/ingest", raw)
			retryDone <- callErr
		}()

		releaseOnce.Do(func() { close(releasePrompt) })

		select {
		case err := <-firstDone:
			if err != nil && firstCtx.Err() == nil {
				t.Fatalf("first ingest error = %v with active context, want nil or canceled-context result", err)
			}
		case <-time.After(time.Second):
			t.Fatal("first ingest did not finish after prompt release")
		}
		select {
		case err := <-retryDone:
			if err != nil {
				t.Fatalf("retry ingest error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("retry ingest did not finish after prompt release")
		}

		if got := driver.promptCount(); got != 1 {
			t.Fatalf("driver.promptCount() = %d, want 1", got)
		}
		regs := broker.snapshotRegistrations()
		if got := len(regs); got != 1 {
			t.Fatalf("len(prompt delivery registrations) = %d, want 1", got)
		}
		if got, want := regs[0].TurnID, "turn-1"; got != want {
			t.Fatalf("registration turn id = %q, want %q from the original prompt", got, want)
		}
		if _, err := env.registry.GetBridgeIngestDedup(
			testutil.Context(t),
			"idem-cancel-dedup",
			env.currentTime(),
		); err != nil {
			t.Fatalf("GetBridgeIngestDedup() error = %v", err)
		}
	})
}

func (e *hostAPITestEnv) useContractBridgePromptDriver(
	t *testing.T,
	driver session.AgentDriver,
	broker *recordingPromptDeliveryBroker,
) {
	t.Helper()

	sessions, err := session.NewManager(
		session.WithHomePaths(e.homePaths),
		session.WithDriver(driver),
		session.WithWorkspaceResolver(e.workspaces),
		session.WithStore(storeSessionDB),
		session.WithSandboxRegistry(mustLocalSandboxRegistry(t)),
		session.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		session.WithNow(func() time.Time { return e.currentTime() }),
		session.WithSessionIDGenerator(sequentialSessionIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialSessionIDGenerator("turn")),
	)
	if err != nil {
		t.Fatalf("session.NewManager(contract bridge prompt driver) error = %v", err)
	}

	taskManager, err := taskpkg.NewManager(
		taskpkg.WithStore(e.registry),
		taskpkg.WithSessionExecutor(&hostAPITestTaskSessionExecutor{
			sessions:            sessions,
			globalWorkspacePath: e.homePaths.HomeDir,
		}),
		taskpkg.WithManagerNow(func() time.Time { return e.currentTime() }),
	)
	if err != nil {
		t.Fatalf("task.NewManager(contract bridge prompt driver) error = %v", err)
	}

	e.sessions = sessions
	e.tasks = taskManager
	e.handler = NewHostAPIHandler(
		e.sessions,
		e.memory,
		nil,
		e.skills,
		WithHostAPITaskManager(e.tasks),
		WithHostAPICapabilityChecker(e.checker),
		WithHostAPIWorkspaceResolver(e.workspaces),
		WithHostAPIBridgeRegistry(e.bridges),
		WithHostAPIBridgeDedupStore(e.registry),
		WithHostAPIDeliveryBroker(broker),
		WithHostAPIResourceStore(e.resources),
		WithHostAPINow(func() time.Time { return e.currentTime() }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)
}

type contractBridgePromptDriver struct {
	mu            sync.Mutex
	now           time.Time
	releasePrompt <-chan struct{}
	processes     map[*session.AgentProcess]*contractBridgePromptProcess
	prompts       []acp.PromptRequest
	promptStarted chan struct{}
	startSeq      atomic.Int64
}

type contractBridgePromptProcess struct {
	done sync.Once
	ch   chan struct{}
}

func newContractBridgePromptDriver(now time.Time, releasePrompt <-chan struct{}) *contractBridgePromptDriver {
	return &contractBridgePromptDriver{
		now:           now,
		releasePrompt: releasePrompt,
		processes:     make(map[*session.AgentProcess]*contractBridgePromptProcess),
		promptStarted: make(chan struct{}, 1),
	}
}

func (d *contractBridgePromptDriver) Start(
	_ context.Context,
	opts acp.StartOpts,
) (*session.AgentProcess, error) {
	seq := d.startSeq.Add(1)
	procState := &contractBridgePromptProcess{ch: make(chan struct{})}
	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       int(seq),
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		ToolHost:  opts.ToolHost,
		SessionID: fmt.Sprintf("acp-contract-%d", seq),
		StartedAt: d.now.Add(time.Duration(seq) * time.Millisecond),
		Done:      procState.ch,
		Wait: func() error {
			<-procState.ch
			return nil
		},
	})

	d.mu.Lock()
	d.processes[proc] = procState
	d.mu.Unlock()
	return proc, nil
}

func (d *contractBridgePromptDriver) Prompt(
	_ context.Context,
	_ *session.AgentProcess,
	req acp.PromptRequest,
) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.prompts = append(d.prompts, req)
	releasePrompt := d.releasePrompt
	d.mu.Unlock()

	select {
	case d.promptStarted <- struct{}{}:
	default:
	}

	events := make(chan acp.AgentEvent, 2)
	go func() {
		defer close(events)
		if releasePrompt != nil {
			<-releasePrompt
		}
		events <- acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			TurnID:    req.TurnID,
			Timestamp: d.now.Add(time.Second),
			Text:      "ack: " + req.Message,
		}
		events <- acp.AgentEvent{
			Type:      acp.EventTypeDone,
			TurnID:    req.TurnID,
			Timestamp: d.now.Add(2 * time.Second),
		}
	}()
	return events, nil
}

func (d *contractBridgePromptDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *contractBridgePromptDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	state := d.processes[proc]
	d.mu.Unlock()
	if state == nil {
		return nil
	}
	state.done.Do(func() { close(state.ch) })
	return nil
}

func (d *contractBridgePromptDriver) promptCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.prompts)
}
