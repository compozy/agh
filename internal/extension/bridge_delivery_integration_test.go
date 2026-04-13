//go:build integration

package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type scriptedPromptEvent struct {
	Type  string
	Text  string
	Error string
	Delay time.Duration
}

type scriptedPromptDriver struct {
	now       time.Time
	script    []scriptedPromptEvent
	processes map[*session.AgentProcess]*scriptedPromptProcess
	prompts   []acp.PromptRequest
	mu        sync.Mutex
	startSeq  atomic.Int64
}

type scriptedPromptProcess struct {
	done sync.Once
	ch   chan struct{}
}

type deliveryIntegrationEnv struct {
	now           time.Time
	homePaths     aghconfig.HomePaths
	workspace     workspacepkg.ResolvedWorkspace
	workspaces    *hostAPIFakeWorkspaceResolver
	globalDB      *globaldb.GlobalDB
	bridges       *bridgepkg.Service
	sessions      *session.Manager
	manager       *Manager
	broker        *bridgepkg.Broker
	handler       *HostAPIHandler
	checker       *CapabilityChecker
	extensionName string
}

func TestBridgeDeliveryIntegrationPromptProducesOrderedDeliveryStream(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	driver := newScriptedPromptDriver(time.Date(2026, 4, 11, 3, 0, 0, 0, time.UTC), []scriptedPromptEvent{
		{Type: acp.EventTypeAgentMessage, Text: "hello"},
		{Type: acp.EventTypeAgentMessage, Text: " world"},
		{Type: acp.EventTypeDone},
	})
	markerPath := filepath.Join(t.TempDir(), "deliveries.jsonl")
	env := newDeliveryIntegrationEnv(t, driver, "ext-bridge-order", "record_deliveries", markerPath)

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-order",
		ExtensionName: env.extensionName,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-order",
		"received_at":         env.now.Format(time.RFC3339Nano),
		"idempotency_key":     "idem-order",
		"content":             map[string]any{"text": "hello"},
	}

	if _, err := env.callWithContext(t, env.bridgeContext(instance), env.extensionName, "bridges/messages/ingest", params); err != nil {
		t.Fatalf("Handle(bridges/messages/ingest) error = %v", err)
	}

	waitForDeliveryMarkers(t, markerPath, func(markers []managerDeliveryMarker) bool {
		return len(markers) >= 2 && markers[len(markers)-1].Request.Event.EventType == bridgepkg.DeliveryEventTypeFinal
	})

	markers := readDeliveryMarkers(t, markerPath)
	assertMarkerDeliveryProgress(t, markers)
}

func TestBridgeDeliveryIntegrationSlowAdapterCoalescesIntermediateDeltas(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	driver := newScriptedPromptDriver(time.Date(2026, 4, 11, 3, 5, 0, 0, time.UTC), []scriptedPromptEvent{
		{Type: acp.EventTypeAgentMessage, Text: "h"},
		{Type: acp.EventTypeAgentMessage, Text: "el"},
		{Type: acp.EventTypeAgentMessage, Text: "lo"},
		{Type: acp.EventTypeAgentMessage, Text: "!"},
		{Type: acp.EventTypeDone},
	})
	markerPath := filepath.Join(t.TempDir(), "slow-deliveries.jsonl")
	env := newDeliveryIntegrationEnv(
		t,
		driver,
		"ext-bridge-slow",
		"slow_record_deliveries",
		markerPath,
		bridgepkg.WithDeliveryBrokerQueueCapacity(2),
	)

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-slow",
		ExtensionName: env.extensionName,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-slow",
		"received_at":         env.now.Format(time.RFC3339Nano),
		"idempotency_key":     "idem-slow",
		"content":             map[string]any{"text": "hello"},
	}

	if _, err := env.callWithContext(t, env.bridgeContext(instance), env.extensionName, "bridges/messages/ingest", params); err != nil {
		t.Fatalf("Handle(bridges/messages/ingest) error = %v", err)
	}

	waitForDeliveryMarkers(t, markerPath, func(markers []managerDeliveryMarker) bool {
		return len(markers) >= 2 && markers[len(markers)-1].Request.Event.EventType == bridgepkg.DeliveryEventTypeFinal
	})

	markers := readDeliveryMarkers(t, markerPath)
	if len(markers) >= 5 {
		t.Fatalf("len(delivery markers) = %d, want coalesced stream smaller than 5 projected events", len(markers))
	}
	if got := markers[0].Request.Event.EventType; got != bridgepkg.DeliveryEventTypeStart {
		t.Fatalf("first delivery event = %q, want start", got)
	}
	last := markers[len(markers)-1].Request.Event
	if got := last.EventType; got != bridgepkg.DeliveryEventTypeFinal {
		t.Fatalf("last delivery event = %q, want final", got)
	}
	if got, want := last.Seq, int64(5); got != want {
		t.Fatalf("last delivery seq = %d, want %d", got, want)
	}
}

func TestBridgeDeliveryIntegrationRestartResumesActiveDelivery(t *testing.T) {
	withDaemonVersion(t, "0.5.0")

	driver := newScriptedPromptDriver(time.Date(2026, 4, 11, 3, 10, 0, 0, time.UTC), []scriptedPromptEvent{
		{Type: acp.EventTypeAgentMessage, Text: "hello"},
		{Type: acp.EventTypeDone},
	})
	markerPath := filepath.Join(t.TempDir(), "resume-deliveries.jsonl")
	env := newDeliveryIntegrationEnv(
		t,
		driver,
		"ext-bridge-resume",
		"exit_once_record_deliveries",
		markerPath,
		bridgepkg.WithDeliveryBrokerRetryDelay(20*time.Millisecond),
	)

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-resume",
		ExtensionName: env.extensionName,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-resume",
		"received_at":         env.now.Format(time.RFC3339Nano),
		"idempotency_key":     "idem-resume",
		"content":             map[string]any{"text": "hello"},
	}

	if _, err := env.callWithContext(t, env.bridgeContext(instance), env.extensionName, "bridges/messages/ingest", params); err != nil {
		t.Fatalf("Handle(bridges/messages/ingest) error = %v", err)
	}

	waitForDeliveryMarkers(t, markerPath, func(markers []managerDeliveryMarker) bool {
		for _, marker := range markers {
			if marker.Request.Event.EventType == bridgepkg.DeliveryEventTypeResume {
				return true
			}
		}
		return false
	})

	markers := readDeliveryMarkers(t, markerPath)
	if len(markers) < 2 {
		t.Fatalf("len(delivery markers) = %d, want at least start + resume", len(markers))
	}
	if got := markers[0].Request.Event.EventType; got != bridgepkg.DeliveryEventTypeStart {
		t.Fatalf("first delivery event = %q, want start", got)
	}

	resumeIndex := -1
	for idx, marker := range markers {
		if marker.Request.Event.EventType == bridgepkg.DeliveryEventTypeResume {
			resumeIndex = idx
			break
		}
	}
	if resumeIndex < 0 {
		t.Fatalf("delivery markers = %#v, want resume event", markers)
	}
	if markers[resumeIndex].PID == markers[0].PID {
		t.Fatalf("resume marker pid = %d, want restart to use a different process than %d", markers[resumeIndex].PID, markers[0].PID)
	}
	if markers[resumeIndex].Request.Snapshot == nil {
		t.Fatal("resume request snapshot = nil, want resumable state")
	}
	if got, want := markers[resumeIndex].Request.Snapshot.DeliveryID, markers[0].Request.Event.DeliveryID; got != want {
		t.Fatalf("resume snapshot delivery id = %q, want %q", got, want)
	}
	if got, want := markers[resumeIndex].Request.Snapshot.LatestEventType, bridgepkg.DeliveryEventTypeFinal; got != want {
		t.Fatalf("resume snapshot latest event type = %q, want %q", got, want)
	}
}

func newDeliveryIntegrationEnv(
	t *testing.T,
	driver session.AgentDriver,
	extensionName string,
	scenario string,
	markerPath string,
	brokerOpts ...bridgepkg.DeliveryBrokerOption,
) *deliveryIntegrationEnv {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	baseNow := time.Date(2026, 4, 11, 3, 0, 0, 0, time.UTC)
	resolvedWorkspace := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      "ws-bridge-delivery",
			RootDir: workspaceRoot,
			Name:    "bridge-delivery-workspace",
		},
		Config: aghconfig.Config{
			Defaults: aghconfig.DefaultsConfig{Agent: "coder"},
			Providers: map[string]aghconfig.ProviderConfig{
				"fake": {Command: "fake-agent"},
			},
			Permissions: aghconfig.PermissionsConfig{Mode: aghconfig.PermissionModeApproveAll},
		},
		Agents: []aghconfig.AgentDef{{
			Name:        "coder",
			Provider:    "fake",
			Permissions: string(aghconfig.PermissionModeApproveAll),
			Prompt:      "You are a reliable coder.",
		}},
		ResolvedAt: baseNow,
	}

	workspaces := newHostAPIFakeWorkspaceResolver(resolvedWorkspace)
	globalDB, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("globaldb.OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("globaldb.Close() error = %v", err)
		}
	})
	if err := globalDB.InsertWorkspace(testutil.Context(t), resolvedWorkspace.Workspace); err != nil {
		t.Fatalf("globalDB.InsertWorkspace() error = %v", err)
	}

	bridgeRegistry := bridgepkg.NewRegistry(globalDB, bridgepkg.WithNow(func() time.Time { return baseNow }))
	registryEnv := newRegistryTestEnv(t)
	fixture := createManagerTestExtension(t, managerTestManifest(extensionName, managerManifestOptions{
		command:      helperCommand(t),
		args:         helperArgs(),
		withEnv:      helperEnv(scenario, markerPath),
		capabilities: []string{"bridge.adapter"},
		actions: []string{
			"bridges/messages/ingest",
			"bridges/instances/get",
			"bridges/instances/report_state",
		},
		security: []string{"bridge.read", "bridge.write"},
	}), nil)
	installManagerFixture(t, registryEnv.registry, fixture, SourceUser, true)

	manager := NewManager(
		registryEnv.registry,
		WithBridgeRuntimeResolver(&stubBridgeRuntimeResolver{
			runtimes: map[string]*subprocess.InitializeBridgeRuntime{
				extensionName: {
					Instance: testBridgeRuntimeInstance(extensionName, "runtime-"+extensionName),
				},
			},
		}),
		WithHealthCheckTimeout(20*time.Millisecond),
		WithSubprocessSignalGrace(15*time.Millisecond),
		withRestartBackoffMax(10*time.Millisecond),
		withHealthPollBounds(time.Millisecond, 2*time.Millisecond),
	)
	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}

	broker := bridgepkg.NewBroker(manager, brokerOpts...)
	skillsRegistry := skillspkg.NewRegistry(skillspkg.RegistryConfig{}, skillspkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	checker := &CapabilityChecker{}
	notifier := NewBridgeDeliveryNotifier(broker, nil)
	sessions, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithDriver(driver),
		session.WithNotifier(notifier),
		session.WithWorkspaceResolver(workspaces),
		session.WithStore(func(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
			return storeSessionDB(ctx, sessionID, path)
		}),
		session.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		session.WithNow(func() time.Time { return baseNow }),
		session.WithSessionIDGenerator(sequentialSessionIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialSessionIDGenerator("turn")),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}

	handler := NewHostAPIHandler(
		sessions,
		memory.NewStore(homePaths.MemoryDir),
		nil,
		skillsRegistry,
		WithHostAPICapabilityChecker(checker),
		WithHostAPIWorkspaceResolver(workspaces),
		WithHostAPIBridgeRegistry(bridgeRegistry),
		WithHostAPIBridgeDedupStore(globalDB),
		WithHostAPIDeliveryBroker(broker),
		WithHostAPINow(func() time.Time { return baseNow }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)
	checker.Register(extensionName, SourceUser, &Manifest{
		Actions:  ActionsConfig{Requires: []string{"bridges/messages/ingest"}},
		Security: SecurityConfig{Capabilities: []string{"bridge.write"}},
	})

	env := &deliveryIntegrationEnv{
		now:           baseNow,
		homePaths:     homePaths,
		workspace:     resolvedWorkspace,
		workspaces:    workspaces,
		globalDB:      globalDB,
		bridges:       bridgeRegistry,
		sessions:      sessions,
		manager:       manager,
		broker:        broker,
		handler:       handler,
		checker:       checker,
		extensionName: extensionName,
	}
	t.Cleanup(func() {
		env.stopSessions(t)
		env.broker.Close()
		if err := env.manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Stop() cleanup error = %v", err)
		}
	})

	return env
}

func (e *deliveryIntegrationEnv) callWithContext(
	t testing.TB,
	ctx context.Context,
	extName string,
	method string,
	params any,
) (any, error) {
	t.Helper()

	raw, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	return e.handler.Handle(ctx, extName, method, raw)
}

func (e *deliveryIntegrationEnv) bridgeContext(instance *bridgepkg.BridgeInstance) context.Context {
	return withHostAPIBridgeRuntime(context.Background(), &subprocess.InitializeBridgeRuntime{
		Instance: *instance,
	})
}

func (e *deliveryIntegrationEnv) createBridgeInstance(
	t *testing.T,
	req bridgepkg.CreateInstanceRequest,
) *bridgepkg.BridgeInstance {
	t.Helper()

	if req.Scope == "" {
		req.Scope = bridgepkg.ScopeWorkspace
	}
	if req.WorkspaceID == "" && req.Scope == bridgepkg.ScopeWorkspace {
		req.WorkspaceID = e.workspace.ID
	}
	if req.Platform == "" {
		req.Platform = "telegram"
	}
	if req.ExtensionName == "" {
		req.ExtensionName = e.extensionName
	}
	if req.DisplayName == "" {
		req.DisplayName = "Bridge Delivery Test"
	}
	if req.Status == "" {
		req.Status = bridgepkg.BridgeStatusReady
		req.Enabled = true
	}

	instance, err := e.bridges.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("bridges.CreateInstance() error = %v", err)
	}
	return instance
}

func (e *deliveryIntegrationEnv) stopSessions(t testing.TB) {
	t.Helper()

	for _, info := range e.sessions.List() {
		if info == nil {
			continue
		}
		_ = e.sessions.Stop(testutil.Context(t), info.ID)
	}
}

func newScriptedPromptDriver(now time.Time, script []scriptedPromptEvent) *scriptedPromptDriver {
	return &scriptedPromptDriver{
		now:       now,
		script:    append([]scriptedPromptEvent(nil), script...),
		processes: make(map[*session.AgentProcess]*scriptedPromptProcess),
	}
}

func (d *scriptedPromptDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	seq := d.startSeq.Add(1)
	procState := &scriptedPromptProcess{ch: make(chan struct{})}
	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       int(seq),
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: fmt.Sprintf("acp-%d", seq),
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

func (d *scriptedPromptDriver) Prompt(_ context.Context, _ *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.prompts = append(d.prompts, req)
	script := append([]scriptedPromptEvent(nil), d.script...)
	startedAt := d.now
	d.mu.Unlock()

	events := make(chan acp.AgentEvent, len(script))
	go func() {
		defer close(events)
		for idx, item := range script {
			if item.Delay > 0 {
				time.Sleep(item.Delay)
			}
			events <- acp.AgentEvent{
				Type:      item.Type,
				TurnID:    req.TurnID,
				Timestamp: startedAt.Add(time.Duration(idx+1) * time.Millisecond),
				Text:      item.Text,
				Error:     item.Error,
			}
		}
	}()
	return events, nil
}

func (d *scriptedPromptDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *scriptedPromptDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	state := d.processes[proc]
	d.mu.Unlock()
	if state == nil {
		return nil
	}
	state.done.Do(func() { close(state.ch) })
	return nil
}

func readDeliveryMarkers(t *testing.T, path string) []managerDeliveryMarker {
	t.Helper()

	lines, err := readFileLines(path)
	if err != nil {
		t.Fatalf("readFileLines(%q) error = %v", path, err)
	}

	markers := make([]managerDeliveryMarker, 0, len(lines))
	for _, line := range lines {
		var marker managerDeliveryMarker
		if err := json.Unmarshal([]byte(line), &marker); err != nil {
			t.Fatalf("json.Unmarshal(delivery marker) error = %v; line=%q", err, line)
		}
		markers = append(markers, marker)
	}
	return markers
}

func waitForDeliveryMarkers(t *testing.T, path string, condition func([]managerDeliveryMarker) bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		markers := readDeliveryMarkersOrEmpty(path)
		if condition(markers) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("delivery markers at %q did not satisfy condition before timeout", path)
}

func readDeliveryMarkersOrEmpty(path string) []managerDeliveryMarker {
	lines, err := readFileLines(path)
	if err != nil {
		return nil
	}
	markers := make([]managerDeliveryMarker, 0, len(lines))
	for _, line := range lines {
		var marker managerDeliveryMarker
		if json.Unmarshal([]byte(line), &marker) != nil {
			continue
		}
		markers = append(markers, marker)
	}
	return markers
}

func assertMarkerEvents(t *testing.T, markers []managerDeliveryMarker, wantTypes []string, wantSeqs []int64) {
	t.Helper()

	if len(markers) < len(wantTypes) {
		t.Fatalf("len(markers) = %d, want at least %d", len(markers), len(wantTypes))
	}
	gotTypes := make([]string, 0, len(markers))
	gotSeqs := make([]int64, 0, len(markers))
	for _, marker := range markers {
		gotTypes = append(gotTypes, marker.Request.Event.EventType)
		gotSeqs = append(gotSeqs, marker.Request.Event.Seq)
	}
	if !slices.Equal(gotTypes[:len(wantTypes)], wantTypes) {
		t.Fatalf("marker event types = %#v, want prefix %#v", gotTypes, wantTypes)
	}
	if !slices.Equal(gotSeqs[:len(wantSeqs)], wantSeqs) {
		t.Fatalf("marker seqs = %#v, want prefix %#v", gotSeqs, wantSeqs)
	}
}

func assertMarkerDeliveryProgress(t *testing.T, markers []managerDeliveryMarker) {
	t.Helper()

	if len(markers) < 2 {
		t.Fatalf("len(markers) = %d, want at least start and final", len(markers))
	}
	if got := markers[0].Request.Event.EventType; got != bridgepkg.DeliveryEventTypeStart {
		t.Fatalf("first marker event = %q, want start", got)
	}
	if got := markers[len(markers)-1].Request.Event.EventType; got != bridgepkg.DeliveryEventTypeFinal {
		t.Fatalf("last marker event = %q, want final", got)
	}

	deliveryID := markers[0].Request.Event.DeliveryID
	lastSeq := int64(0)
	for idx, marker := range markers {
		event := marker.Request.Event
		if event.DeliveryID != deliveryID {
			t.Fatalf("marker delivery_id[%d] = %q, want %q", idx, event.DeliveryID, deliveryID)
		}
		if event.Seq <= lastSeq {
			t.Fatalf("marker seq[%d] = %d, want increasing order after %d", idx, event.Seq, lastSeq)
		}
		lastSeq = event.Seq
	}
}
