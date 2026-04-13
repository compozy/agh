package extensiontest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	"github.com/pedronauck/agh/internal/subprocess"
	aghtestutil "github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	EnvHandshakePath = "AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH"
	EnvInstancePath  = "AGH_BRIDGE_ADAPTER_INSTANCE_PATH"
	EnvStatePath     = "AGH_BRIDGE_ADAPTER_STATE_PATH"
	EnvDeliveryPath  = "AGH_BRIDGE_ADAPTER_DELIVERY_PATH"
	EnvIngestPath    = "AGH_BRIDGE_ADAPTER_INGEST_PATH"
	EnvUpdatesPath   = "AGH_BRIDGE_ADAPTER_UPDATES_PATH"
	EnvStartsPath    = "AGH_BRIDGE_ADAPTER_STARTS_PATH"
	EnvShutdownPath  = "AGH_BRIDGE_ADAPTER_SHUTDOWN_PATH"
	EnvCrashOncePath = "AGH_BRIDGE_ADAPTER_CRASH_ONCE_PATH"
)

// MarkerPaths contains the standardized marker and control files used by the
// reference adapter and the conformance harness.
type MarkerPaths struct {
	Handshake string
	Instance  string
	State     string
	Delivery  string
	Ingest    string
	Updates   string
	Starts    string
	Shutdown  string
	CrashOnce string
}

// Env returns the environment expected by the reference adapter marker contract.
func (m MarkerPaths) Env() map[string]string {
	return map[string]string{
		EnvHandshakePath: m.Handshake,
		EnvInstancePath:  m.Instance,
		EnvStatePath:     m.State,
		EnvDeliveryPath:  m.Delivery,
		EnvIngestPath:    m.Ingest,
		EnvUpdatesPath:   m.Updates,
		EnvStartsPath:    m.Starts,
		EnvShutdownPath:  m.Shutdown,
		EnvCrashOncePath: m.CrashOnce,
	}
}

// NewMarkerPaths returns a temp-root-relative marker layout used by the
// subprocess-backed adapter tests.
func NewMarkerPaths(root string) MarkerPaths {
	return MarkerPaths{
		Handshake: filepath.Join(root, "adapter-handshake.json"),
		Instance:  filepath.Join(root, "adapter-instance.json"),
		State:     filepath.Join(root, "adapter-states.jsonl"),
		Delivery:  filepath.Join(root, "adapter-deliveries.jsonl"),
		Ingest:    filepath.Join(root, "adapter-ingest.jsonl"),
		Updates:   filepath.Join(root, "adapter-updates.jsonl"),
		Starts:    filepath.Join(root, "adapter-starts.log"),
		Shutdown:  filepath.Join(root, "adapter-shutdown.log"),
		CrashOnce: filepath.Join(root, "adapter-crash-once.json"),
	}
}

// HandshakeRecord captures the adapter initialize marker.
type HandshakeRecord struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

// DeliveryRecord captures one `bridges/deliver` request plus the adapter ack
// when the subprocess remained alive long enough to return it.
type DeliveryRecord struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
	Ack     *bridgepkg.DeliveryAck    `json:"ack,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

// StateRecord captures one adapter-driven `bridges/instances/report_state`
// result marker.
type StateRecord struct {
	Status   bridgepkg.BridgeStatus   `json:"status"`
	Instance bridgepkg.BridgeInstance `json:"instance,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

// IngestRecord captures one fake inbound update mapped into a normalized ingest.
type IngestRecord struct {
	Envelope bridgepkg.InboundMessageEnvelope              `json:"envelope"`
	Result   extensioncontract.BridgesMessagesIngestResult `json:"result,omitempty"`
	Error    string                                        `json:"error,omitempty"`
}

// ConformanceReport is the collected adapter evidence used by the validator.
type ConformanceReport struct {
	Handshake  *HandshakeRecord
	Instance   *bridgepkg.BridgeInstance
	States     []StateRecord
	Deliveries []DeliveryRecord
	Ingests    []IngestRecord
}

// ConformanceExpectation configures the reusable adapter validator.
type ConformanceExpectation struct {
	InstanceID          string
	ExtensionName       string
	BoundSecretNames    []string
	RequireStateReport  bool
	RequireDelivery     bool
	RequireResume       bool
	ExpectedFinalStatus bridgepkg.BridgeStatus
}

// ConformanceIssue reports one adapter contract failure.
type ConformanceIssue struct {
	Code    string
	Message string
}

// ConformanceError aggregates reusable harness failures.
type ConformanceError struct {
	Issues []ConformanceIssue
}

func (e *ConformanceError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
	}
	return strings.Join(parts, "; ")
}

// ValidateConformance checks the adapter evidence against the reusable bridge
// adapter contract enforced by the harness.
func ValidateConformance(report ConformanceReport, expect ConformanceExpectation) error {
	issues := make([]ConformanceIssue, 0)

	if report.Handshake == nil {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_handshake",
			Message: "adapter did not write an initialize marker",
		})
	} else {
		request := report.Handshake.Request
		if !slices.Contains(request.Methods.ExtensionServices, "bridges/deliver") {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_delivery_negotiation",
				Message: "initialize did not negotiate bridges/deliver",
			})
		}
		if request.Runtime.Bridge == nil {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_bridge_runtime",
				Message: "initialize runtime.bridge was nil",
			})
		} else {
			if expect.InstanceID != "" && strings.TrimSpace(request.Runtime.Bridge.Instance.ID) != strings.TrimSpace(expect.InstanceID) {
				issues = append(issues, ConformanceIssue{
					Code: "wrong_instance",
					Message: fmt.Sprintf(
						"initialize runtime bridge instance id = %q, want %q",
						request.Runtime.Bridge.Instance.ID,
						expect.InstanceID,
					),
				})
			}
			if expect.ExtensionName != "" && strings.TrimSpace(request.Runtime.Bridge.Instance.ExtensionName) != strings.TrimSpace(expect.ExtensionName) {
				issues = append(issues, ConformanceIssue{
					Code: "wrong_extension",
					Message: fmt.Sprintf(
						"initialize runtime bridge extension = %q, want %q",
						request.Runtime.Bridge.Instance.ExtensionName,
						expect.ExtensionName,
					),
				})
			}

			bound := make(map[string]struct{}, len(request.Runtime.Bridge.BoundSecrets))
			for _, secret := range request.Runtime.Bridge.BoundSecrets {
				bound[strings.TrimSpace(secret.BindingName)] = struct{}{}
			}
			for _, bindingName := range expect.BoundSecretNames {
				if _, ok := bound[strings.TrimSpace(bindingName)]; !ok {
					issues = append(issues, ConformanceIssue{
						Code: "missing_bound_secret",
						Message: fmt.Sprintf(
							"initialize runtime did not include bound secret %q",
							bindingName,
						),
					})
				}
			}
		}

		for _, method := range request.Capabilities.GrantedActions {
			text := strings.ToLower(strings.TrimSpace(string(method)))
			if strings.Contains(text, "vault/") || strings.Contains(text, "secret/") {
				issues = append(issues, ConformanceIssue{
					Code:    "leaked_secret_surface",
					Message: fmt.Sprintf("initialize granted unexpected secret surface %q", method),
				})
			}
		}
	}

	if expect.RequireStateReport && len(report.States) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_state_report",
			Message: "adapter did not report bridge state",
		})
	}
	if expect.ExpectedFinalStatus != "" && len(report.States) > 0 {
		last := report.States[len(report.States)-1]
		if last.Status.Normalize() != expect.ExpectedFinalStatus.Normalize() {
			issues = append(issues, ConformanceIssue{
				Code: "wrong_final_status",
				Message: fmt.Sprintf(
					"last reported status = %q, want %q",
					last.Status,
					expect.ExpectedFinalStatus,
				),
			})
		}
	}

	if expect.RequireDelivery && len(report.Deliveries) == 0 {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_delivery",
			Message: "adapter did not receive any delivery requests",
		})
	}

	lastSeq := make(map[string]int64)
	pendingAckResume := make(map[string]bool)
	sawResume := false
	for _, record := range report.Deliveries {
		event := record.Request.Event
		deliveryID := strings.TrimSpace(event.DeliveryID)
		eventType := normalizeEventType(event.EventType)
		if eventType == bridgepkg.DeliveryEventTypeResume {
			sawResume = true
			if record.Request.Snapshot == nil {
				issues = append(issues, ConformanceIssue{
					Code:    "missing_resume_snapshot",
					Message: fmt.Sprintf("resume delivery %q omitted its snapshot", deliveryID),
				})
			}
			delete(pendingAckResume, deliveryID)
		}

		if eventType != bridgepkg.DeliveryEventTypeResume {
			if previous, ok := lastSeq[deliveryID]; ok && event.Seq <= previous {
				issues = append(issues, ConformanceIssue{
					Code:    "out_of_order_delivery",
					Message: fmt.Sprintf("delivery %q sequence %d arrived after %d", deliveryID, event.Seq, previous),
				})
			}
			lastSeq[deliveryID] = event.Seq
		}

		if record.Ack == nil {
			pendingAckResume[deliveryID] = true
			continue
		}
		if err := record.Ack.ValidateFor(event); err != nil {
			issues = append(issues, ConformanceIssue{
				Code:    "invalid_ack",
				Message: err.Error(),
			})
		}
		delete(pendingAckResume, deliveryID)

		if event.Seq > 0 && strings.TrimSpace(record.Ack.RemoteMessageID) == "" {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_remote_message_id",
				Message: fmt.Sprintf("delivery %q sequence %d did not return remote_message_id", deliveryID, event.Seq),
			})
		}
		if event.Seq > 1 && eventType != bridgepkg.DeliveryEventTypeResume && strings.TrimSpace(record.Ack.ReplaceRemoteMessageID) == "" {
			issues = append(issues, ConformanceIssue{
				Code:    "missing_replace_remote_message_id",
				Message: fmt.Sprintf("delivery %q sequence %d did not return replace_remote_message_id", deliveryID, event.Seq),
			})
		}
	}

	for deliveryID := range pendingAckResume {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_ack",
			Message: fmt.Sprintf("delivery %q did not return an ack or later resume", deliveryID),
		})
	}
	if expect.RequireResume && !sawResume {
		issues = append(issues, ConformanceIssue{
			Code:    "missing_resume",
			Message: "adapter did not observe a resume delivery after restart",
		})
	}

	if len(issues) > 0 {
		return &ConformanceError{Issues: issues}
	}
	return nil
}

// ScriptedPromptEvent is one deterministic agent event emitted by the harness
// driver when the host creates a session prompt.
type ScriptedPromptEvent struct {
	Type  string
	Text  string
	Error string
	Delay time.Duration
}

// ScriptedPromptDriver is a deterministic in-process session driver used by
// the adapter harness instead of a real ACP subprocess.
type ScriptedPromptDriver struct {
	now       time.Time
	script    []ScriptedPromptEvent
	processes map[*session.AgentProcess]*scriptedPromptProcess
	prompts   []acp.PromptRequest
	mu        sync.Mutex
	startSeq  atomic.Int64
}

type scriptedPromptProcess struct {
	done sync.Once
	ch   chan struct{}
}

// NewScriptedPromptDriver constructs a session driver that replays the provided
// agent events for every prompt.
func NewScriptedPromptDriver(now time.Time, script []ScriptedPromptEvent) *ScriptedPromptDriver {
	return &ScriptedPromptDriver{
		now:       now,
		script:    append([]ScriptedPromptEvent(nil), script...),
		processes: make(map[*session.AgentProcess]*scriptedPromptProcess),
	}
}

// Start implements session.AgentDriver.
func (d *ScriptedPromptDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
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

// Prompt implements session.AgentDriver.
func (d *ScriptedPromptDriver) Prompt(ctx context.Context, _ *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.prompts = append(d.prompts, req)
	script := append([]ScriptedPromptEvent(nil), d.script...)
	startedAt := d.now
	d.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}

	events := make(chan acp.AgentEvent, len(script))
	go func() {
		defer close(events)
		for idx, item := range script {
			if item.Delay > 0 {
				timer := time.NewTimer(item.Delay)
				select {
				case <-timer.C:
				case <-ctx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return
				}
			}
			event := acp.AgentEvent{
				Type:      item.Type,
				TurnID:    req.TurnID,
				Timestamp: startedAt.Add(time.Duration(idx+1) * time.Millisecond),
				Text:      item.Text,
				Error:     item.Error,
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

// Cancel implements session.AgentDriver.
func (d *ScriptedPromptDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

// Stop implements session.AgentDriver.
func (d *ScriptedPromptDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	state := d.processes[proc]
	d.mu.Unlock()
	if state == nil {
		return nil
	}
	state.done.Do(func() { close(state.ch) })
	return nil
}

// HarnessConfig configures one subprocess-backed bridge adapter test harness.
type HarnessConfig struct {
	ExtensionDir             string
	ExtensionName            string
	BridgeInstanceID         string
	DisplayName              string
	Platform                 string
	RoutingPolicy            bridgepkg.RoutingPolicy
	BoundSecrets             []subprocess.InitializeBridgeBoundSecret
	Driver                   session.AgentDriver
	StartTime                time.Time
	CrashOnceOnFirstDelivery bool
	BrokerOptions            []bridgepkg.DeliveryBrokerOption
	ExtraEnv                 map[string]string
}

// Harness wires the manager, host API, session manager, observer, and marker
// contract used to validate bridge adapters end to end.
type Harness struct {
	HomePaths aghconfig.HomePaths
	Markers   MarkerPaths
	Observer  *observepkg.Observer
	Bridges   *bridgepkg.Service
	Broker    *bridgepkg.Broker
	Handler   *extensionpkg.HostAPIHandler
	Manager   *extensionpkg.Manager
	Sessions  *session.Manager
	Instance  *bridgepkg.BridgeInstance
}

// NewHarness starts the reusable adapter conformance harness.
func NewHarness(t testing.TB, cfg HarnessConfig) *Harness {
	t.Helper()

	if strings.TrimSpace(cfg.ExtensionDir) == "" {
		t.Fatal("extensiontest: HarnessConfig.ExtensionDir is required")
	}

	now := cfg.StartTime.UTC()
	if now.IsZero() {
		now = time.Date(2026, 4, 11, 4, 0, 0, 0, time.UTC)
	}

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	markers := NewMarkerPaths(filepath.Join(t.TempDir(), "markers"))
	for key, value := range markers.Env() {
		if key == EnvCrashOncePath {
			continue
		}
		t.Setenv(key, value)
	}
	if cfg.CrashOnceOnFirstDelivery {
		t.Setenv(EnvCrashOncePath, markers.CrashOnce)
	}
	for key, value := range cfg.ExtraEnv {
		t.Setenv(key, value)
	}

	workspace := defaultResolvedWorkspace(filepath.Join(t.TempDir(), "workspace"), now)
	workspaces := staticWorkspaceResolver{resolved: workspace}

	globalDB, err := globaldb.OpenGlobalDB(aghtestutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("globaldb.OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(aghtestutil.Context(t)); err != nil {
			t.Fatalf("globaldb.Close() error = %v", err)
		}
	})
	if err := globalDB.InsertWorkspace(aghtestutil.Context(t), workspace.Workspace); err != nil {
		t.Fatalf("globalDB.InsertWorkspace() error = %v", err)
	}

	manifest, err := extensionpkg.LoadManifest(cfg.ExtensionDir)
	if err != nil {
		t.Fatalf("extension.LoadManifest() error = %v", err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(cfg.ExtensionDir)
	if err != nil {
		t.Fatalf("extension.ComputeDirectoryChecksum() error = %v", err)
	}
	extensionRegistry := extensionpkg.NewRegistry(globalDB.DB())
	if err := extensionRegistry.Install(manifest, cfg.ExtensionDir, checksum); err != nil {
		t.Fatalf("extensionRegistry.Install() error = %v", err)
	}

	extensionName := strings.TrimSpace(cfg.ExtensionName)
	if extensionName == "" {
		extensionName = manifest.Name
	}

	bridgeRegistry := bridgepkg.NewRegistry(globalDB, bridgepkg.WithNow(func() time.Time { return now }))
	createReq := bridgepkg.CreateInstanceRequest{
		ID:            firstNonEmpty(cfg.BridgeInstanceID, "brg-telegram-reference"),
		Scope:         bridgepkg.ScopeWorkspace,
		WorkspaceID:   workspace.ID,
		Platform:      firstNonEmpty(cfg.Platform, "telegram"),
		ExtensionName: extensionName,
		DisplayName:   firstNonEmpty(cfg.DisplayName, "Telegram Reference"),
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusStarting,
		RoutingPolicy: cfg.RoutingPolicy,
	}
	if createReq.RoutingPolicy == (bridgepkg.RoutingPolicy{}) {
		createReq.RoutingPolicy = bridgepkg.RoutingPolicy{IncludePeer: true}
	}
	instance, err := bridgeRegistry.CreateInstance(aghtestutil.Context(t), createReq)
	if err != nil {
		t.Fatalf("bridgeRegistry.CreateInstance() error = %v", err)
	}

	checker := &extensionpkg.CapabilityChecker{}
	checker.Register(extensionName, extensionpkg.SourceUser, manifest)

	var hostHandler *extensionpkg.HostAPIHandler
	telemetrySink := &deferredBridgeTelemetrySink{}
	hostForwarder := func(method string) subprocess.HandlerFunc {
		return func(ctx context.Context, params json.RawMessage) (any, error) {
			if hostHandler == nil {
				return nil, errors.New("extensiontest: host api handler is not initialized")
			}
			return hostHandler.HandleMethod(method)(ctx, params)
		}
	}

	manager := extensionpkg.NewManager(
		extensionRegistry,
		extensionpkg.WithCapabilityChecker(checker),
		extensionpkg.WithBridgeRuntimeResolver(&stubBridgeRuntimeResolver{
			runtimes: map[string]*subprocess.InitializeBridgeRuntime{
				extensionName: {
					Instance:     *instance,
					BoundSecrets: cloneBoundSecrets(cfg.BoundSecrets),
				},
			},
		}),
		extensionpkg.WithBridgeTelemetrySink(telemetrySink),
		extensionpkg.WithHostMethodHandler("bridges/messages/ingest", hostForwarder("bridges/messages/ingest")),
		extensionpkg.WithHostMethodHandler("bridges/instances/get", hostForwarder("bridges/instances/get")),
		extensionpkg.WithHostMethodHandler("bridges/instances/report_state", hostForwarder("bridges/instances/report_state")),
		extensionpkg.WithHealthCheckTimeout(20*time.Millisecond),
		extensionpkg.WithSubprocessSignalGrace(15*time.Millisecond),
	)

	broker := bridgepkg.NewBroker(manager, cfg.BrokerOptions...)
	observer, err := observepkg.New(
		aghtestutil.Context(t),
		observepkg.WithRegistry(globalDB),
		observepkg.WithHomePaths(homePaths),
		observepkg.WithWorkspaceResolver(workspaces),
		observepkg.WithBridgeSource(harnessBridgeSource{service: bridgeRegistry, broker: broker}),
		observepkg.WithNow(func() time.Time { return now }),
		observepkg.WithStartTime(now),
	)
	if err != nil {
		t.Fatalf("observe.New() error = %v", err)
	}
	telemetrySink.observer = observer

	driver := cfg.Driver
	if driver == nil {
		driver = NewScriptedPromptDriver(now, []ScriptedPromptEvent{
			{Type: acp.EventTypeAgentMessage, Text: "hello"},
			{Type: acp.EventTypeDone},
		})
	}
	notifier := extensionpkg.NewBridgeDeliveryNotifier(broker, observer)
	sessions, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithDriver(driver),
		session.WithNotifier(notifier),
		session.WithWorkspaceResolver(workspaces),
		session.WithStore(func(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
			return sessiondb.OpenSessionDB(ctx, sessionID, path)
		}),
		session.WithNow(func() time.Time { return now }),
		session.WithSessionIDGenerator(sequentialIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialIDGenerator("turn")),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}

	hostHandler = extensionpkg.NewHostAPIHandler(
		sessions,
		nil,
		observer,
		skillspkg.NewRegistry(skillspkg.RegistryConfig{}),
		extensionpkg.WithHostAPICapabilityChecker(checker),
		extensionpkg.WithHostAPIWorkspaceResolver(workspaces),
		extensionpkg.WithHostAPIBridgeRegistry(bridgeRegistry),
		extensionpkg.WithHostAPIBridgeDedupStore(globalDB),
		extensionpkg.WithHostAPIDeliveryBroker(broker),
		extensionpkg.WithHostAPINow(func() time.Time { return now }),
		extensionpkg.WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		extensionpkg.WithHostAPIRateLimit(1000, 1000),
	)

	if err := manager.Start(aghtestutil.Context(t)); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}

	harness := &Harness{
		HomePaths: homePaths,
		Markers:   markers,
		Observer:  observer,
		Bridges:   bridgeRegistry,
		Broker:    broker,
		Handler:   hostHandler,
		Manager:   manager,
		Sessions:  sessions,
		Instance:  instance,
	}

	t.Cleanup(func() {
		harness.stopSessions(t)
		harness.Broker.Close()
		if err := harness.Manager.Stop(aghtestutil.Context(t)); err != nil {
			t.Fatalf("manager.Stop() error = %v", err)
		}
	})

	return harness
}

// AppendInboundUpdate appends one fake platform update line for the adapter to ingest.
func (h *Harness) AppendInboundUpdate(t testing.TB, update any) {
	t.Helper()
	appendJSONLine(t, h.Markers.Updates, update)
}

// WaitForHandshake waits until the adapter writes its initialize marker.
func (h *Harness) WaitForHandshake(t testing.TB, timeout time.Duration) HandshakeRecord {
	t.Helper()
	var record HandshakeRecord
	waitForCondition(t, timeout, "adapter handshake", func() bool {
		loaded, err := readJSONFile[HandshakeRecord](h.Markers.Handshake)
		if err != nil {
			return false
		}
		record = loaded
		return true
	})
	return record
}

// WaitForStates waits until the state marker file satisfies the predicate.
func (h *Harness) WaitForStates(t testing.TB, timeout time.Duration, predicate func([]StateRecord) bool) []StateRecord {
	t.Helper()
	var records []StateRecord
	waitForCondition(t, timeout, "adapter state markers", func() bool {
		loaded, err := readJSONLinesFile[StateRecord](h.Markers.State)
		if err != nil {
			return false
		}
		records = loaded
		return predicate(records)
	})
	return records
}

// WaitForDeliveries waits until the delivery marker file satisfies the predicate.
func (h *Harness) WaitForDeliveries(t testing.TB, timeout time.Duration, predicate func([]DeliveryRecord) bool) []DeliveryRecord {
	t.Helper()
	var records []DeliveryRecord
	waitForCondition(t, timeout, "adapter delivery markers", func() bool {
		loaded, err := readJSONLinesFile[DeliveryRecord](h.Markers.Delivery)
		if err != nil {
			return false
		}
		records = loaded
		return predicate(records)
	})
	return records
}

// WaitForIngests waits until the ingest marker file satisfies the predicate.
func (h *Harness) WaitForIngests(t testing.TB, timeout time.Duration, predicate func([]IngestRecord) bool) []IngestRecord {
	t.Helper()
	var records []IngestRecord
	waitForCondition(t, timeout, "adapter ingest markers", func() bool {
		loaded, err := readJSONLinesFile[IngestRecord](h.Markers.Ingest)
		if err != nil {
			return false
		}
		records = loaded
		return predicate(records)
	})
	return records
}

// ObserveHealth returns the current observer health surface.
func (h *Harness) ObserveHealth(t testing.TB) observepkg.Health {
	t.Helper()
	health, err := h.Observer.Health(aghtestutil.Context(t))
	if err != nil {
		t.Fatalf("observer.Health() error = %v", err)
	}
	return health
}

// QueryBridgeHealth returns the current per-instance bridge health rows.
func (h *Harness) QueryBridgeHealth(t testing.TB) []observepkg.BridgeInstanceHealth {
	t.Helper()
	rows, err := h.Observer.QueryBridgeHealth(aghtestutil.Context(t))
	if err != nil {
		t.Fatalf("observer.QueryBridgeHealth() error = %v", err)
	}
	return rows
}

// Report reads the current collected adapter evidence into one reusable report.
func (h *Harness) Report(t testing.TB) ConformanceReport {
	t.Helper()

	report := ConformanceReport{}
	if handshake, err := readJSONFile[HandshakeRecord](h.Markers.Handshake); err == nil {
		report.Handshake = &handshake
	}
	if instance, err := readJSONFile[bridgepkg.BridgeInstance](h.Markers.Instance); err == nil {
		report.Instance = &instance
	}
	if states, err := readJSONLinesFile[StateRecord](h.Markers.State); err == nil {
		report.States = states
	}
	if deliveries, err := readJSONLinesFile[DeliveryRecord](h.Markers.Delivery); err == nil {
		report.Deliveries = deliveries
	}
	if ingests, err := readJSONLinesFile[IngestRecord](h.Markers.Ingest); err == nil {
		report.Ingests = ingests
	}
	return report
}

func (h *Harness) stopSessions(t testing.TB) {
	t.Helper()
	for _, info := range h.Sessions.List() {
		if info == nil {
			continue
		}
		if err := h.Sessions.Stop(aghtestutil.Context(t), info.ID); err != nil {
			t.Fatalf("Sessions.Stop(%q) error = %v", info.ID, err)
		}
	}
}

type harnessBridgeSource struct {
	service *bridgepkg.Service
	broker  *bridgepkg.Broker
}

type deferredBridgeTelemetrySink struct {
	observer *observepkg.Observer
}

func (s *deferredBridgeTelemetrySink) RecordBridgeAuthFailure(bridgeInstanceID string) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.RecordBridgeAuthFailure(bridgeInstanceID)
}

func (s *deferredBridgeTelemetrySink) RecordBridgeRuntimeIssue(bridgeInstanceID string, status bridgepkg.BridgeStatus, message string) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.RecordBridgeRuntimeIssue(bridgeInstanceID, status, message)
}

func (s *deferredBridgeTelemetrySink) ClearBridgeRuntimeIssue(bridgeInstanceID string) {
	if s == nil || s.observer == nil {
		return
	}
	s.observer.ClearBridgeRuntimeIssue(bridgeInstanceID)
}

func (s harnessBridgeSource) ListInstances(ctx context.Context) ([]bridgepkg.BridgeInstance, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListInstances(ctx)
}

func (s harnessBridgeSource) ListRoutes(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListRoutes(ctx, bridgeInstanceID)
}

func (s harnessBridgeSource) DeliveryMetrics() map[string]bridgepkg.BridgeDeliveryMetrics {
	if s.broker == nil {
		return nil
	}
	return s.broker.DeliveryMetrics()
}

type staticWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
}

func (r staticWorkspaceResolver) Resolve(_ context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	trimmed := strings.TrimSpace(idOrPath)
	if trimmed == "" {
		return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
	}
	if trimmed == r.resolved.ID || trimmed == r.resolved.RootDir {
		return r.resolved, nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r staticWorkspaceResolver) ResolveOrRegister(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == r.resolved.RootDir {
		return r.resolved, nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

type stubBridgeRuntimeResolver struct {
	runtimes map[string]*subprocess.InitializeBridgeRuntime
}

func (r *stubBridgeRuntimeResolver) ResolveBridgeRuntime(_ context.Context, extensionName string) (*subprocess.InitializeBridgeRuntime, error) {
	if r == nil || r.runtimes == nil {
		return nil, nil
	}
	return subprocess.CloneInitializeBridgeRuntime(r.runtimes[strings.TrimSpace(extensionName)]), nil
}

func defaultResolvedWorkspace(root string, now time.Time) workspacepkg.ResolvedWorkspace {
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:           "ws-bridge-adapter",
			RootDir:      root,
			Name:         "bridge-adapter-workspace",
			DefaultAgent: "coder",
			CreatedAt:    now,
			UpdatedAt:    now,
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
		ResolvedAt: now,
	}
}

func cloneBoundSecrets(src []subprocess.InitializeBridgeBoundSecret) []subprocess.InitializeBridgeBoundSecret {
	if len(src) == 0 {
		return nil
	}
	return append([]subprocess.InitializeBridgeBoundSecret(nil), src...)
}

func sequentialIDGenerator(prefix string) session.IDGenerator {
	var counter atomic.Int64
	return func() string {
		return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
	}
}

func readJSONFile[T any](path string) (T, error) {
	var item T
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return item, err
	}
	err = json.Unmarshal(payload, &item)
	return item, err
}

func readJSONLinesFile[T any](path string) ([]T, error) {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	lines := nonEmptyLines(string(payload))
	items := make([]T, 0, len(lines))
	for _, line := range lines {
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func appendJSONLine(t testing.TB, path string, value any) {
	t.Helper()
	target := strings.TrimSpace(path)
	if target == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(target), err)
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("os.OpenFile(%q) error = %v", target, err)
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		t.Fatalf("encoder.Encode(%q) error = %v", target, err)
	}
}

func waitForCondition(t testing.TB, timeout time.Duration, label string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s did not satisfy condition before timeout", label)
}

func nonEmptyLines(input string) []string {
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}

func normalizeEventType(eventType string) string {
	return strings.ToLower(strings.TrimSpace(eventType))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
