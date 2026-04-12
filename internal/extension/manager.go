package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/version"
)

const (
	defaultProtocolVersion         = "1"
	defaultHealthCheckInterval     = 30 * time.Second
	defaultHealthCheckTimeout      = 5 * time.Second
	defaultInitializeTimeout       = 5 * time.Second
	defaultHookTimeout             = 5 * time.Second
	defaultShutdownTimeout         = 10 * time.Second
	defaultRestartBackoffMax       = 60 * time.Second
	defaultRestartFailureThreshold = 5
	defaultHealthPollFloor         = 50 * time.Millisecond
	defaultHealthPollCeiling       = time.Second
	defaultSubprocessSignalGrace   = 10 * time.Second
	extensionHookSource            = hookspkg.HookSourceConfig
)

var (
	// ErrChannelRuntimeDeferred reports that a channel-capable extension is
	// installed and registered, but no enabled channel instance exists yet for
	// the runtime launch handshake.
	ErrChannelRuntimeDeferred = errors.New("extension: channel runtime deferred")
)

var safeSubprocessEnvKeys = []string{
	"PATH",
	"HOME",
	"USER",
	"LOGNAME",
	"TMPDIR",
	"TMP",
	"TEMP",
	"LANG",
	"LC_ALL",
	"SHELL",
	"SystemRoot",
	"ComSpec",
	"PATHEXT",
	"USERPROFILE",
}

// Option customizes an extension manager.
type Option func(*Manager)

type processHandle interface {
	HandleMethod(string, subprocess.HandlerFunc) error
	Call(context.Context, string, any, any) error
	Initialize(context.Context, subprocess.InitializeRequest) (subprocess.InitializeResponse, error)
	Shutdown(context.Context) error
	Done() <-chan struct{}
	Wait() error
	HealthState() subprocess.HealthState
	PID() int
}

type processLauncher func(context.Context, subprocess.LaunchConfig) (processHandle, error)

type skillRegistry interface {
	RegisterExternal(owner string, skills []*skillspkg.Skill) error
	RemoveExternal(owner string)
}

// ChannelRuntimeResolver resolves one instance-scoped channel launch payload
// for a channel-capable extension session.
type ChannelRuntimeResolver interface {
	ResolveChannelRuntime(ctx context.Context, extensionName string) (*subprocess.InitializeChannelRuntime, error)
}

// ChannelTelemetrySink records live channel runtime/auth telemetry for
// per-instance observability surfaces.
type ChannelTelemetrySink interface {
	RecordChannelAuthFailure(channelInstanceID string)
	RecordChannelRuntimeIssue(channelInstanceID string, status channelspkg.ChannelStatus, message string)
	ClearChannelRuntimeIssue(channelInstanceID string)
}

// ExtensionPhase names one lifecycle phase or supervisor state for an extension.
type ExtensionPhase string

const (
	ExtensionPhaseDiscover   ExtensionPhase = "discover"
	ExtensionPhaseParse      ExtensionPhase = "parse"
	ExtensionPhaseValidate   ExtensionPhase = "validate"
	ExtensionPhaseRegister   ExtensionPhase = "register"
	ExtensionPhaseInitialize ExtensionPhase = "initialize"
	ExtensionPhaseActivate   ExtensionPhase = "activate"
	ExtensionPhaseRecover    ExtensionPhase = "recover"
	ExtensionPhaseStop       ExtensionPhase = "stop"
)

// ExtensionStatus captures the runtime state exposed to health/observer code.
type ExtensionStatus struct {
	Name                string
	Version             string
	Source              ExtensionSource
	Enabled             bool
	Registered          bool
	Active              bool
	Phase               ExtensionPhase
	PID                 int
	Healthy             bool
	HealthMessage       string
	HealthLastCheckedAt time.Time
	ConsecutiveFailures int
	RestartBackoff      time.Duration
	LastError           string
	LastStartedAt       time.Time
	LastExitedAt        time.Time
}

// Extension is the manager-visible snapshot for one installed extension.
type Extension struct {
	Info             ExtensionInfo
	Manifest         *Manifest
	RootDir          string
	Hooks            []hookspkg.HookDecl
	Agents           []aghconfig.AgentDef
	MCPServers       []aghconfig.MCPServer
	Skills           []*skillspkg.Skill
	GrantedActions   []string
	GrantedSecurity  []string
	InitializeResult *subprocess.InitializeResponse
	Status           ExtensionStatus
}

type managedExtension struct {
	info            ExtensionInfo
	rootDir         string
	manifest        *Manifest
	hooks           []hookspkg.HookDecl
	agents          []aghconfig.AgentDef
	mcpServers      []aghconfig.MCPServer
	skills          []*skillspkg.Skill
	grantedActions  []string
	grantedSecurity []string
	initialize      *subprocess.InitializeResponse
	process         processHandle
	runtime         subprocess.InitializeRuntime
	healthInterval  time.Duration
	generation      int64

	phase               ExtensionPhase
	registered          bool
	active              bool
	awaitingStability   bool
	consecutiveFailures int
	restartBackoff      time.Duration
	lastError           string
	lastStartedAt       time.Time
	lastExitedAt        time.Time
}

var _ channelspkg.DeliveryTransport = (*Manager)(nil)

// Manager orchestrates extension loading, subprocess lifecycle, and resource registration.
type Manager struct {
	mu sync.RWMutex

	registry               *Registry
	capChecker             *CapabilityChecker
	channelRuntimeResolver ChannelRuntimeResolver
	channelTelemetrySink   ChannelTelemetrySink
	skillsRegistry         skillRegistry
	logger                 *slog.Logger
	now                    func() time.Time
	getenv                 func(string) string
	launch                 processLauncher

	hostMethods map[string]subprocess.HandlerFunc

	protocolVersion           string
	supportedProtocolVersions []string
	initializeTimeout         time.Duration
	healthCheckTimeout        time.Duration
	defaultHookTimeout        time.Duration
	defaultShutdownTimeout    time.Duration
	restartBackoffMax         time.Duration
	restartFailureThreshold   int
	healthPollFloor           time.Duration
	healthPollCeiling         time.Duration
	subprocessSignalGrace     time.Duration

	lifecycleCtx context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	started      bool
	stopping     bool

	extensions map[string]*managedExtension
}

// WithCapabilityChecker injects the grant evaluator used for Host API authorization.
func WithCapabilityChecker(checker *CapabilityChecker) Option {
	return func(manager *Manager) {
		manager.capChecker = checker
	}
}

// WithChannelRuntimeResolver injects the channel launch material resolver used
// for channel-capable extension sessions.
func WithChannelRuntimeResolver(resolver ChannelRuntimeResolver) Option {
	return func(manager *Manager) {
		manager.channelRuntimeResolver = resolver
	}
}

// WithChannelTelemetrySink injects the sink used to publish per-instance
// runtime degradation/error signals into observability surfaces.
func WithChannelTelemetrySink(sink ChannelTelemetrySink) Option {
	return func(manager *Manager) {
		manager.channelTelemetrySink = sink
	}
}

// WithSkillsRegistry injects the skills registry used for extension skill registration.
func WithSkillsRegistry(registry skillRegistry) Option {
	return func(manager *Manager) {
		manager.skillsRegistry = registry
	}
}

// WithLogger injects the logger used for extension diagnostics.
func WithLogger(logger *slog.Logger) Option {
	return func(manager *Manager) {
		manager.logger = logger
	}
}

// WithNow overrides the manager clock, mainly for tests.
func WithNow(now func() time.Time) Option {
	return func(manager *Manager) {
		manager.now = now
	}
}

// WithGetenv overrides environment lookup used for manifest template expansion.
func WithGetenv(getenv func(string) string) Option {
	return func(manager *Manager) {
		manager.getenv = getenv
	}
}

// WithHostMethodHandler registers one Host API method handler for launched extensions.
func WithHostMethodHandler(method string, handler subprocess.HandlerFunc) Option {
	return func(manager *Manager) {
		if manager.hostMethods == nil {
			manager.hostMethods = make(map[string]subprocess.HandlerFunc)
		}
		manager.hostMethods[strings.TrimSpace(method)] = handler
	}
}

// WithInitializeTimeout overrides the initialize handshake timeout.
func WithInitializeTimeout(timeout time.Duration) Option {
	return func(manager *Manager) {
		manager.initializeTimeout = timeout
	}
}

// WithHealthCheckTimeout overrides the negotiated health probe timeout.
func WithHealthCheckTimeout(timeout time.Duration) Option {
	return func(manager *Manager) {
		manager.healthCheckTimeout = timeout
	}
}

// WithDefaultHookTimeout overrides the negotiated default hook timeout.
func WithDefaultHookTimeout(timeout time.Duration) Option {
	return func(manager *Manager) {
		manager.defaultHookTimeout = timeout
	}
}

// WithSubprocessSignalGrace overrides the SIGTERM -> SIGKILL grace interval.
func WithSubprocessSignalGrace(timeout time.Duration) Option {
	return func(manager *Manager) {
		manager.subprocessSignalGrace = timeout
	}
}

func withProcessLauncher(launcher processLauncher) Option {
	return func(manager *Manager) {
		manager.launch = launcher
	}
}

func withRestartBackoffMax(max time.Duration) Option {
	return func(manager *Manager) {
		manager.restartBackoffMax = max
	}
}

func withRestartFailureThreshold(threshold int) Option {
	return func(manager *Manager) {
		manager.restartFailureThreshold = threshold
	}
}

func withHealthPollBounds(floor, ceiling time.Duration) Option {
	return func(manager *Manager) {
		manager.healthPollFloor = floor
		manager.healthPollCeiling = ceiling
	}
}

// NewManager constructs an extension manager with sensible defaults.
func NewManager(registry *Registry, opts ...Option) *Manager {
	manager := &Manager{
		registry:                  registry,
		capChecker:                &CapabilityChecker{},
		logger:                    slog.Default(),
		now:                       func() time.Time { return time.Now().UTC() },
		getenv:                    os.Getenv,
		hostMethods:               make(map[string]subprocess.HandlerFunc),
		protocolVersion:           defaultProtocolVersion,
		supportedProtocolVersions: []string{defaultProtocolVersion},
		initializeTimeout:         defaultInitializeTimeout,
		healthCheckTimeout:        defaultHealthCheckTimeout,
		defaultHookTimeout:        defaultHookTimeout,
		defaultShutdownTimeout:    defaultShutdownTimeout,
		restartBackoffMax:         defaultRestartBackoffMax,
		restartFailureThreshold:   defaultRestartFailureThreshold,
		healthPollFloor:           defaultHealthPollFloor,
		healthPollCeiling:         defaultHealthPollCeiling,
		subprocessSignalGrace:     defaultSubprocessSignalGrace,
		extensions:                make(map[string]*managedExtension),
	}
	manager.launch = func(ctx context.Context, cfg subprocess.LaunchConfig) (processHandle, error) {
		return subprocess.Launch(ctx, cfg)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(manager)
		}
	}

	if manager.capChecker == nil {
		manager.capChecker = &CapabilityChecker{}
	}
	if manager.logger == nil {
		manager.logger = slog.Default()
	}
	if manager.now == nil {
		manager.now = func() time.Time { return time.Now().UTC() }
	}
	if manager.getenv == nil {
		manager.getenv = os.Getenv
	}
	if manager.launch == nil {
		manager.launch = func(ctx context.Context, cfg subprocess.LaunchConfig) (processHandle, error) {
			return subprocess.Launch(ctx, cfg)
		}
	}
	if len(manager.supportedProtocolVersions) == 0 {
		manager.supportedProtocolVersions = []string{defaultProtocolVersion}
	}
	if manager.protocolVersion == "" {
		manager.protocolVersion = manager.supportedProtocolVersions[0]
	}
	if manager.initializeTimeout <= 0 {
		manager.initializeTimeout = defaultInitializeTimeout
	}
	if manager.healthCheckTimeout <= 0 {
		manager.healthCheckTimeout = defaultHealthCheckTimeout
	}
	if manager.defaultHookTimeout <= 0 {
		manager.defaultHookTimeout = defaultHookTimeout
	}
	if manager.defaultShutdownTimeout <= 0 {
		manager.defaultShutdownTimeout = defaultShutdownTimeout
	}
	if manager.restartBackoffMax <= 0 {
		manager.restartBackoffMax = defaultRestartBackoffMax
	}
	if manager.restartFailureThreshold <= 0 {
		manager.restartFailureThreshold = defaultRestartFailureThreshold
	}
	if manager.healthPollFloor <= 0 {
		manager.healthPollFloor = defaultHealthPollFloor
	}
	if manager.healthPollCeiling <= 0 {
		manager.healthPollCeiling = defaultHealthPollCeiling
	}
	if manager.subprocessSignalGrace <= 0 {
		manager.subprocessSignalGrace = defaultSubprocessSignalGrace
	}

	return manager
}

// Start loads every enabled extension through the six-phase pipeline.
func (m *Manager) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("extension: context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("extension: manager is required")
	}
	if m.registry == nil {
		return errors.New("extension: registry is required")
	}

	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return errors.New("extension: manager already started")
	}
	m.lifecycleCtx, m.cancel = context.WithCancel(context.Background())
	m.started = true
	m.stopping = false
	m.extensions = make(map[string]*managedExtension)
	m.mu.Unlock()

	infos, err := m.registry.List()
	if err != nil {
		return fmt.Errorf("extension: list registry entries: %w", err)
	}

	var errs []error
	for _, info := range infos {
		ext := &managedExtension{
			info:  info,
			phase: ExtensionPhaseDiscover,
		}
		m.mu.Lock()
		m.extensions[info.Name] = ext
		m.mu.Unlock()

		if !info.Enabled {
			ext.lastError = ""
			continue
		}

		if err := m.startOne(ctx, ext); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Stop gracefully drains all active extension subprocesses.
func (m *Manager) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("extension: context is required")
	}
	if m == nil {
		return errors.New("extension: manager is required")
	}

	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	m.stopping = true
	cancel := m.cancel
	names := make([]string, 0, len(m.extensions))
	for name := range m.extensions {
		names = append(names, name)
	}
	slices.Sort(names)
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	errCh := make(chan error, len(names))
	var stopWG sync.WaitGroup
	for _, name := range names {
		ext, ok := m.lookupManaged(name)
		if !ok {
			continue
		}

		stopWG.Add(1)
		go func(item *managedExtension) {
			defer stopWG.Done()

			proc := item.process
			if proc != nil {
				if err := proc.Shutdown(ctx); err != nil {
					if waitErr := proc.Wait(); waitErr != nil {
						errCh <- fmt.Errorf("extension %q stop: %w", item.info.Name, errors.Join(err, waitErr))
					}
				}
			}

			m.unregisterResources(item)

			m.mu.Lock()
			item.process = nil
			item.active = false
			item.awaitingStability = false
			item.phase = ExtensionPhaseStop
			m.mu.Unlock()

			m.logger.Info("extension.lifecycle.shutdown", "extension", item.info.Name)
		}(ext)
	}
	stopWG.Wait()
	close(errCh)

	m.wg.Wait()

	m.mu.Lock()
	m.started = false
	m.stopping = false
	m.cancel = nil
	m.lifecycleCtx = nil
	m.mu.Unlock()

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Reload restarts the manager from the current registry state.
func (m *Manager) Reload(ctx context.Context) error {
	if ctx == nil {
		return errors.New("extension: context is required")
	}
	if m == nil {
		return errors.New("extension: manager is required")
	}

	stopErr := m.Stop(ctx)
	startErr := m.Start(ctx)
	return errors.Join(stopErr, startErr)
}

// Get returns the current snapshot for one installed extension.
func (m *Manager) Get(name string) (*Extension, error) {
	if m == nil {
		return nil, errors.New("extension: manager is required")
	}

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, errors.New("extension: extension name is required")
	}

	m.mu.RLock()
	ext := m.extensions[trimmed]
	m.mu.RUnlock()
	if ext != nil {
		return m.cloneExtension(ext), nil
	}

	if m.registry == nil {
		return nil, &ExtensionNotFoundError{Name: trimmed}
	}
	info, err := m.registry.Get(trimmed)
	if err != nil {
		return nil, err
	}
	return &Extension{
		Info:   *info,
		Status: ExtensionStatus{Name: info.Name, Version: info.Version, Source: info.Source, Enabled: info.Enabled},
	}, nil
}

// List returns every currently known registry row in name order.
func (m *Manager) List() []ExtensionInfo {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ExtensionInfo, 0, len(m.extensions))
	for _, ext := range m.extensions {
		infos = append(infos, ext.info)
	}
	slices.SortFunc(infos, func(left, right ExtensionInfo) int {
		return strings.Compare(left.Name, right.Name)
	})
	return infos
}

// Statuses returns the current runtime health snapshot for every known extension.
func (m *Manager) Statuses() []ExtensionStatus {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]ExtensionStatus, 0, len(m.extensions))
	for _, ext := range m.extensions {
		statuses = append(statuses, m.statusLocked(ext))
	}
	slices.SortFunc(statuses, func(left, right ExtensionStatus) int {
		return strings.Compare(left.Name, right.Name)
	})
	return statuses
}

// DeliverChannel calls the negotiated `channels/deliver` service on the named
// channel-capable extension runtime.
func (m *Manager) DeliverChannel(
	ctx context.Context,
	extensionName string,
	req channelspkg.DeliveryRequest,
) (channelspkg.DeliveryAck, error) {
	if ctx == nil {
		return channelspkg.DeliveryAck{}, errors.New("extension: delivery context is required")
	}
	if err := ctx.Err(); err != nil {
		return channelspkg.DeliveryAck{}, err
	}
	if m == nil {
		return channelspkg.DeliveryAck{}, errors.New("extension: manager is required")
	}
	if err := req.Validate(); err != nil {
		return channelspkg.DeliveryAck{}, err
	}

	name := strings.TrimSpace(extensionName)
	if name == "" {
		return channelspkg.DeliveryAck{}, errors.New("extension: delivery extension name is required")
	}

	m.mu.RLock()
	ext := m.extensions[name]
	if ext == nil || ext.process == nil || !ext.active {
		m.mu.RUnlock()
		return channelspkg.DeliveryAck{}, channelspkg.ErrDeliveryTransportUnavailable
	}
	process := ext.process
	initialize := cloneInitializeResponse(ext.initialize)
	m.mu.RUnlock()

	if initialize == nil || !slices.Contains(initialize.ImplementedMethods, string(extensionprotocol.ExtensionServiceMethodChannelsDeliver)) {
		return channelspkg.DeliveryAck{}, fmt.Errorf(
			"extension: extension %q does not implement %q: %w",
			name,
			extensionprotocol.ExtensionServiceMethodChannelsDeliver,
			channelspkg.ErrDeliveryTransportUnavailable,
		)
	}

	var ack channelspkg.DeliveryAck
	if err := process.Call(ctx, string(extensionprotocol.ExtensionServiceMethodChannelsDeliver), req, &ack); err != nil {
		return channelspkg.DeliveryAck{}, fmt.Errorf("extension: deliver channel via %q: %w", name, err)
	}
	return ack, nil
}

// HookDeclarations returns the manifest-declared hook resources from loaded extensions.
func (m *Manager) HookDeclarations(ctx context.Context) ([]hookspkg.HookDecl, error) {
	if ctx == nil {
		return nil, errors.New("extension: context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("extension: manager is required")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	decls := make([]hookspkg.HookDecl, 0)
	names := make([]string, 0, len(m.extensions))
	for name := range m.extensions {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		ext := m.extensions[name]
		if !ext.registered {
			continue
		}
		for _, decl := range ext.hooks {
			decls = append(decls, cloneHookDecl(decl))
		}
	}
	return decls, nil
}

// AgentDefinitions returns the currently registered extension agent definitions.
func (m *Manager) AgentDefinitions() []aghconfig.AgentDef {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []aghconfig.AgentDef
	names := make([]string, 0, len(m.extensions))
	for name := range m.extensions {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		ext := m.extensions[name]
		if !ext.registered {
			continue
		}
		for _, agent := range ext.agents {
			agents = append(agents, cloneAgentDef(agent))
		}
	}
	return agents
}

// MCPServers returns the currently registered extension MCP server declarations.
func (m *Manager) MCPServers() []aghconfig.MCPServer {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var servers []aghconfig.MCPServer
	names := make([]string, 0, len(m.extensions))
	for name := range m.extensions {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		ext := m.extensions[name]
		if !ext.registered {
			continue
		}
		for _, server := range ext.mcpServers {
			servers = append(servers, cloneMCPServer(server))
		}
	}
	return servers
}

func (m *Manager) startOne(ctx context.Context, ext *managedExtension) error {
	if err := m.discoverExtension(ext); err != nil {
		return err
	}
	if err := m.parseExtension(ext); err != nil {
		return err
	}
	if err := m.validateExtension(ext); err != nil {
		return err
	}
	if err := m.registerExtension(ctx, ext); err != nil {
		return err
	}
	if err := m.initializeExtension(ctx, ext); err != nil {
		return err
	}
	m.activateExtension(ext)
	return nil
}

func (m *Manager) discoverExtension(ext *managedExtension) error {
	manifestPath := strings.TrimSpace(ext.info.ManifestPath)
	if manifestPath == "" {
		err := errors.New("manifest path is required")
		m.setFailure(ext, ExtensionPhaseDiscover, err)
		return phaseError(ext.info.Name, ExtensionPhaseDiscover, err)
	}

	rootDir := filepath.Dir(manifestPath)
	if rootDir == "." || rootDir == "" {
		err := fmt.Errorf("invalid manifest path %q", manifestPath)
		m.setFailure(ext, ExtensionPhaseDiscover, err)
		return phaseError(ext.info.Name, ExtensionPhaseDiscover, err)
	}

	ext.rootDir = rootDir
	ext.phase = ExtensionPhaseDiscover
	return nil
}

func (m *Manager) parseExtension(ext *managedExtension) error {
	manifest, err := loadManifestAtPath(ext.info.ManifestPath)
	if err != nil {
		m.setFailure(ext, ExtensionPhaseParse, err)
		return phaseError(ext.info.Name, ExtensionPhaseParse, err)
	}

	ext.manifest = manifest
	ext.phase = ExtensionPhaseParse
	return nil
}

func (m *Manager) validateExtension(ext *managedExtension) error {
	if ext.manifest == nil {
		err := errors.New("manifest is required")
		m.setFailure(ext, ExtensionPhaseValidate, err)
		return phaseError(ext.info.Name, ExtensionPhaseValidate, err)
	}
	if ext.info.Name != ext.manifest.Name {
		err := fmt.Errorf("registry name %q does not match manifest name %q", ext.info.Name, ext.manifest.Name)
		m.setFailure(ext, ExtensionPhaseValidate, err)
		return phaseError(ext.info.Name, ExtensionPhaseValidate, err)
	}
	if ext.info.Version != "" && ext.info.Version != ext.manifest.Version {
		err := fmt.Errorf("registry version %q does not match manifest version %q", ext.info.Version, ext.manifest.Version)
		m.setFailure(ext, ExtensionPhaseValidate, err)
		return phaseError(ext.info.Name, ExtensionPhaseValidate, err)
	}
	if requiresSubprocess(ext.manifest) && strings.TrimSpace(ext.manifest.Subprocess.Command) == "" {
		err := errors.New("subprocess command is required when runtime capabilities or actions are declared")
		m.setFailure(ext, ExtensionPhaseValidate, err)
		return phaseError(ext.info.Name, ExtensionPhaseValidate, err)
	}

	ext.grantedActions = effectiveActionGrants(ext.info.Source, ext.manifest.Actions.Requires)
	ext.grantedSecurity = effectiveSecurityGrants(ext.info.Source, ext.manifest.Security.Capabilities)
	m.capChecker.Register(ext.info.Name, ext.info.Source, ext.manifest)
	ext.phase = ExtensionPhaseValidate
	return nil
}

func (m *Manager) registerExtension(ctx context.Context, ext *managedExtension) error {
	if err := ctx.Err(); err != nil {
		m.setFailure(ext, ExtensionPhaseRegister, err)
		return err
	}

	skills, err := m.loadSkillResources(ext)
	if err != nil {
		m.setFailure(ext, ExtensionPhaseRegister, err)
		return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
	}
	agents, err := m.loadAgentResources(ext)
	if err != nil {
		m.setFailure(ext, ExtensionPhaseRegister, err)
		return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
	}
	hooks, err := m.loadHookResources(ext)
	if err != nil {
		m.setFailure(ext, ExtensionPhaseRegister, err)
		return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
	}
	mcpServers, err := m.loadMCPResources(ext)
	if err != nil {
		m.setFailure(ext, ExtensionPhaseRegister, err)
		return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
	}

	if len(skills) > 0 {
		if m.skillsRegistry == nil {
			err := errors.New("skills registry is required for extension skill resources")
			m.setFailure(ext, ExtensionPhaseRegister, err)
			return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
		}
		if err := m.skillsRegistry.RegisterExternal(ext.info.Name, skills); err != nil {
			m.setFailure(ext, ExtensionPhaseRegister, err)
			return phaseError(ext.info.Name, ExtensionPhaseRegister, err)
		}
	}

	m.mu.Lock()
	ext.skills = skills
	ext.agents = agents
	ext.hooks = hooks
	ext.mcpServers = mcpServers
	ext.registered = true
	ext.phase = ExtensionPhaseRegister
	m.mu.Unlock()
	return nil
}

func (m *Manager) initializeExtension(ctx context.Context, ext *managedExtension) error {
	if !requiresSubprocess(ext.manifest) {
		m.mu.Lock()
		ext.phase = ExtensionPhaseInitialize
		ext.active = false
		ext.lastError = ""
		m.mu.Unlock()
		return nil
	}

	process, response, runtime, healthInterval, err := m.launchRuntime(ctx, ext)
	if err != nil {
		if errors.Is(err, ErrChannelRuntimeDeferred) {
			m.mu.Lock()
			ext.process = nil
			ext.initialize = nil
			ext.runtime = subprocess.InitializeRuntime{}
			ext.healthInterval = 0
			ext.awaitingStability = false
			ext.lastStartedAt = time.Time{}
			ext.phase = ExtensionPhaseInitialize
			ext.lastError = ""
			m.mu.Unlock()
			return nil
		}
		m.setFailure(ext, ExtensionPhaseInitialize, err)
		return phaseError(ext.info.Name, ExtensionPhaseInitialize, err)
	}

	m.mu.Lock()
	ext.process = process
	ext.initialize = &response
	ext.runtime = runtime
	ext.healthInterval = healthInterval
	ext.awaitingStability = true
	ext.lastStartedAt = m.now()
	ext.phase = ExtensionPhaseInitialize
	ext.lastError = ""
	ext.generation++
	generation := ext.generation
	m.mu.Unlock()

	m.wg.Add(1)
	go m.superviseExtension(ext.info.Name, generation)
	return nil
}

func (m *Manager) activateExtension(ext *managedExtension) {
	if ext == nil {
		return
	}

	m.mu.Lock()
	ext.phase = ExtensionPhaseActivate
	ext.active = ext.process != nil || !requiresSubprocess(ext.manifest)
	ext.restartBackoff = 0
	ext.lastError = ""
	name := ext.info.Name
	source := ext.info.Source.String()
	active := ext.active
	skillCount := len(ext.skills)
	agentCount := len(ext.agents)
	hookCount := len(ext.hooks)
	mcpServerCount := len(ext.mcpServers)
	m.mu.Unlock()

	m.logger.Info(
		"extension.lifecycle.loaded",
		"extension", name,
		"source", source,
		"active", active,
		"skill_count", skillCount,
		"agent_count", agentCount,
		"hook_count", hookCount,
		"mcp_server_count", mcpServerCount,
	)
}

func (m *Manager) superviseExtension(name string, generation int64) {
	defer m.wg.Done()

	for {
		proc, interval, ok := m.currentProcess(name, generation)
		if !ok {
			return
		}

		reason, shouldRecover := m.monitorProcess(name, generation, proc, interval)
		if !shouldRecover {
			return
		}

		nextGeneration, recovered := m.recoverExtension(name, reason)
		if !recovered {
			return
		}
		generation = nextGeneration
	}
}

func (m *Manager) monitorProcess(name string, generation int64, proc processHandle, healthInterval time.Duration) (error, bool) {
	ticker := time.NewTicker(m.healthPollInterval(healthInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.shouldStopSupervision(name, generation, proc) {
				return nil, false
			}

			health := proc.HealthState()
			if !health.Healthy {
				reason := fmt.Errorf("health check failed: %s", strings.TrimSpace(health.Message))
				if strings.TrimSpace(health.LastError) != "" {
					reason = fmt.Errorf("%w: %s", reason, health.LastError)
				}

				shutdownCtx, cancel := context.WithTimeout(context.Background(), m.shutdownDeadlineForProcess(name, generation))
				err := proc.Shutdown(shutdownCtx)
				cancel()
				if err != nil {
					reason = errors.Join(reason, err)
				}
				return reason, true
			}
			if !health.LastCheckedAt.IsZero() {
				m.markStable(name, generation)
			}
		case <-proc.Done():
			if m.shouldStopSupervision(name, generation, proc) {
				return nil, false
			}

			err := proc.Wait()
			if err == nil {
				err = errors.New("process exited unexpectedly")
			}
			return err, true
		case <-m.lifecycleDone():
			return nil, false
		}
	}
}

func (m *Manager) recoverExtension(name string, reason error) (int64, bool) {
	for {
		backoff, disable, ok := m.recordFailure(name, reason)
		if !ok {
			return 0, false
		}
		if disable {
			m.disableExtension(name, reason)
			return 0, false
		}
		if !m.waitBackoff(backoff) {
			return 0, false
		}

		ext, ok := m.lookupManaged(name)
		if !ok {
			return 0, false
		}
		process, response, runtime, healthInterval, err := m.launchRuntime(m.lifecycleContext(), ext)
		if err != nil {
			reason = err
			continue
		}

		m.mu.Lock()
		if m.stopping || ext.generation == 0 && !ext.info.Enabled {
			m.mu.Unlock()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), m.defaultShutdownTimeout)
			_ = process.Shutdown(shutdownCtx)
			cancel()
			return 0, false
		}

		ext.process = process
		ext.initialize = &response
		ext.runtime = runtime
		ext.healthInterval = healthInterval
		ext.awaitingStability = true
		ext.active = true
		ext.phase = ExtensionPhaseActivate
		ext.lastError = ""
		ext.lastStartedAt = m.now()
		ext.generation++
		nextGeneration := ext.generation
		name := ext.info.Name
		source := ext.info.Source.String()
		m.mu.Unlock()

		m.logger.Info("extension.lifecycle.loaded", "extension", name, "source", source, "recovered", true)

		return nextGeneration, true
	}
}

func (m *Manager) launchRuntime(ctx context.Context, ext *managedExtension) (processHandle, subprocess.InitializeResponse, subprocess.InitializeRuntime, time.Duration, error) {
	launchCfg, runtime, healthInterval, err := m.launchConfigFor(ctx, ext)
	if err != nil {
		return nil, subprocess.InitializeResponse{}, subprocess.InitializeRuntime{}, 0, err
	}

	process, err := m.launch(ctx, launchCfg)
	if err != nil {
		return nil, subprocess.InitializeResponse{}, subprocess.InitializeRuntime{}, 0, fmt.Errorf("launch subprocess: %w", err)
	}

	for method, handler := range m.hostMethods {
		if err := process.HandleMethod(method, m.wrapHostHandler(ext.info.Name, method, runtime.Channel, handler)); err != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), m.defaultShutdownTimeout)
			_ = process.Shutdown(shutdownCtx)
			cancel()
			return nil, subprocess.InitializeResponse{}, subprocess.InitializeRuntime{}, 0, fmt.Errorf("register host method %q: %w", method, err)
		}
	}

	request := subprocess.InitializeRequest{
		ProtocolVersion:          m.protocolVersion,
		SupportedProtocolVersion: slices.Clone(m.supportedProtocolVersions),
		AGHVersion:               version.Current().Version,
		Extension: subprocess.InitializeExtension{
			Name:       ext.manifest.Name,
			Version:    ext.manifest.Version,
			SourceTier: ext.info.Source.String(),
		},
		Capabilities: subprocess.InitializeCapabilities{
			Provides:        normalizeUniqueStrings(ext.manifest.Capabilities.Provides),
			GrantedActions:  hostAPIMethodsFromStrings(ext.grantedActions),
			GrantedSecurity: normalizeUniqueStrings(ext.grantedSecurity),
		},
		Methods: subprocess.InitializeMethods{
			DaemonRequests:    daemonRequestMethods(),
			ExtensionServices: capabilityMethods(ext.manifest.Capabilities.Provides),
		},
		Runtime: runtime,
	}

	initCtx, cancel := context.WithTimeout(ctx, m.initializeTimeout)
	defer cancel()

	response, err := process.Initialize(initCtx, request)
	if err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.defaultShutdownTimeout)
		_ = process.Shutdown(shutdownCtx)
		shutdownCancel()
		return nil, subprocess.InitializeResponse{}, subprocess.InitializeRuntime{}, 0, fmt.Errorf("initialize subprocess: %w", err)
	}
	if err := validateSupportedHookEvents(response.SupportedHookEvents); err != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.defaultShutdownTimeout)
		_ = process.Shutdown(shutdownCtx)
		shutdownCancel()
		return nil, subprocess.InitializeResponse{}, subprocess.InitializeRuntime{}, 0, err
	}

	return process, response, runtime, healthInterval, nil
}

func (m *Manager) launchConfigFor(ctx context.Context, ext *managedExtension) (subprocess.LaunchConfig, subprocess.InitializeRuntime, time.Duration, error) {
	if ext.manifest == nil {
		return subprocess.LaunchConfig{}, subprocess.InitializeRuntime{}, 0, errors.New("manifest is required")
	}

	command, err := m.resolveCommand(ext.rootDir, ext.manifest.Subprocess.Command)
	if err != nil {
		return subprocess.LaunchConfig{}, subprocess.InitializeRuntime{}, 0, err
	}
	args, err := m.resolveStringSlice(ext.rootDir, ext.manifest.Subprocess.Args)
	if err != nil {
		return subprocess.LaunchConfig{}, subprocess.InitializeRuntime{}, 0, err
	}
	env, err := m.resolveEnvMap(ext.rootDir, ext.manifest.Subprocess.Env)
	if err != nil {
		return subprocess.LaunchConfig{}, subprocess.InitializeRuntime{}, 0, err
	}

	healthInterval := durationOr(ext.manifest.Subprocess.HealthCheckInterval, defaultHealthCheckInterval)
	shutdownTimeout := durationOr(ext.manifest.Subprocess.ShutdownTimeout, m.defaultShutdownTimeout)
	channelRuntime, err := m.resolveChannelRuntime(ctx, ext)
	if err != nil {
		return subprocess.LaunchConfig{}, subprocess.InitializeRuntime{}, 0, err
	}
	runtime := subprocess.InitializeRuntime{
		HealthCheckIntervalMS: healthInterval.Milliseconds(),
		HealthCheckTimeoutMS:  m.healthCheckTimeout.Milliseconds(),
		ShutdownTimeoutMS:     shutdownTimeout.Milliseconds(),
		DefaultHookTimeoutMS:  m.defaultHookTimeout.Milliseconds(),
		Channel:               channelRuntime,
	}

	launchCfg := subprocess.LaunchConfig{
		Command:         command,
		Args:            args,
		Dir:             ext.rootDir,
		Env:             env,
		Logger:          m.logger,
		ShutdownTimeout: shutdownTimeout,
		PostSignalGrace: m.subprocessSignalGrace,
	}
	return launchCfg, runtime, healthInterval, nil
}

func (m *Manager) wrapHostHandler(
	extName string,
	method string,
	channelRuntime *subprocess.InitializeChannelRuntime,
	handler subprocess.HandlerFunc,
) subprocess.HandlerFunc {
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		if err := m.capChecker.CheckHostAPI(extName, method); err != nil {
			return nil, rpcCapabilityDenied(err)
		}

		hostCtx := withHostAPIExtensionName(ctx, extName)
		if channelRuntime != nil {
			hostCtx = withHostAPIChannelRuntime(hostCtx, channelRuntime)
		}
		return handler(hostCtx, params)
	}
}

func (m *Manager) loadSkillResources(ext *managedExtension) ([]*skillspkg.Skill, error) {
	if ext.manifest == nil || len(ext.manifest.Resources.Skills) == 0 {
		return nil, nil
	}

	source := skillSourceForExtension(ext.info.Source)
	loaded := make(map[string]*skillspkg.Skill)
	for _, resourcePath := range ext.manifest.Resources.Skills {
		resourceRoot, err := resolveResourcePath(ext.rootDir, resourcePath)
		if err != nil {
			return nil, err
		}
		files, err := collectMarkdownFiles(resourceRoot)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			skill, err := skillspkg.ParseSkillFileWithSource(file, source)
			if err != nil {
				return nil, err
			}
			loaded[skill.Meta.Name] = skill
		}
	}

	skills := make([]*skillspkg.Skill, 0, len(loaded))
	for _, name := range sortedKeys(loaded) {
		skills = append(skills, loaded[name])
	}
	return skills, nil
}

func (m *Manager) loadAgentResources(ext *managedExtension) ([]aghconfig.AgentDef, error) {
	if ext.manifest == nil || len(ext.manifest.Resources.Agents) == 0 {
		return nil, nil
	}

	loaded := make(map[string]aghconfig.AgentDef)
	for _, resourcePath := range ext.manifest.Resources.Agents {
		resourceRoot, err := resolveResourcePath(ext.rootDir, resourcePath)
		if err != nil {
			return nil, err
		}
		files, err := collectMarkdownFiles(resourceRoot)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			agent, err := aghconfig.LoadAgentDefFile(file)
			if err != nil {
				return nil, err
			}
			loaded[agent.Name] = agent
		}
	}

	agents := make([]aghconfig.AgentDef, 0, len(loaded))
	for _, name := range sortedKeys(loaded) {
		agents = append(agents, cloneAgentDef(loaded[name]))
	}
	return agents, nil
}

func (m *Manager) loadHookResources(ext *managedExtension) ([]hookspkg.HookDecl, error) {
	if ext.manifest == nil || len(ext.manifest.Resources.Hooks) == 0 {
		return nil, nil
	}

	decls := make([]hookspkg.HookDecl, 0, len(ext.manifest.Resources.Hooks))
	for idx, cfg := range ext.manifest.Resources.Hooks {
		decl, err := m.hookConfigToDecl(ext, cfg)
		if err != nil {
			return nil, fmt.Errorf("extension hook %d (%q): %w", idx, strings.TrimSpace(cfg.Name), err)
		}
		decls = append(decls, decl)
	}
	return decls, nil
}

func (m *Manager) loadMCPResources(ext *managedExtension) ([]aghconfig.MCPServer, error) {
	if ext.manifest == nil || len(ext.manifest.Resources.MCPServers) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(ext.manifest.Resources.MCPServers))
	for name := range ext.manifest.Resources.MCPServers {
		names = append(names, name)
	}
	slices.Sort(names)

	servers := make([]aghconfig.MCPServer, 0, len(names))
	for _, name := range names {
		decl := ext.manifest.Resources.MCPServers[name]
		command, err := m.resolveCommand(ext.rootDir, decl.Command)
		if err != nil {
			return nil, err
		}
		args, err := m.resolveStringSlice(ext.rootDir, decl.Args)
		if err != nil {
			return nil, err
		}
		env, err := m.resolveStringMap(ext.rootDir, decl.Env)
		if err != nil {
			return nil, err
		}
		server := aghconfig.MCPServer{
			Name:    strings.TrimSpace(name),
			Command: command,
			Args:    args,
			Env:     env,
		}
		if err := server.Validate("extension.resources.mcp_servers[" + name + "]"); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func (m *Manager) hookConfigToDecl(ext *managedExtension, cfg HookConfig) (hookspkg.HookDecl, error) {
	command := strings.TrimSpace(cfg.Command)
	args := slices.Clone(cfg.Args)
	env := cloneStringMap(cfg.Env)
	kind := hookspkg.HookExecutorKind(strings.TrimSpace(cfg.Executor.Kind))

	rootSpecified := command != "" || len(args) > 0 || len(env) > 0
	nestedSpecified := strings.TrimSpace(cfg.Executor.Command) != "" || len(cfg.Executor.Args) > 0 || len(cfg.Executor.Env) > 0
	if rootSpecified && nestedSpecified {
		return hookspkg.HookDecl{}, errors.New("hook executor fields must be declared either at the top level or under executor, not both")
	}
	if nestedSpecified {
		command = strings.TrimSpace(cfg.Executor.Command)
		args = slices.Clone(cfg.Executor.Args)
		env = cloneStringMap(cfg.Executor.Env)
	}

	resolvedCommand, err := m.resolveCommand(ext.rootDir, command)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}
	resolvedArgs, err := m.resolveStringSlice(ext.rootDir, args)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}
	resolvedEnv, err := m.resolveStringMap(ext.rootDir, env)
	if err != nil {
		return hookspkg.HookDecl{}, err
	}

	matcher := hookspkg.HookMatcher{
		AgentName:          strings.TrimSpace(cfg.Matcher.AgentName),
		AgentType:          strings.TrimSpace(cfg.Matcher.AgentType),
		WorkspaceID:        strings.TrimSpace(cfg.Matcher.WorkspaceID),
		WorkspaceRoot:      strings.TrimSpace(cfg.Matcher.WorkspaceRoot),
		SessionType:        strings.TrimSpace(cfg.Matcher.SessionType),
		InputClass:         strings.TrimSpace(cfg.Matcher.InputClass),
		ACPEventType:       strings.TrimSpace(cfg.Matcher.ACPEventType),
		TurnID:             strings.TrimSpace(cfg.Matcher.TurnID),
		ToolName:           strings.TrimSpace(cfg.Matcher.ToolName),
		ToolNamespace:      strings.TrimSpace(cfg.Matcher.ToolNamespace),
		DecisionClass:      strings.TrimSpace(cfg.Matcher.DecisionClass),
		MessageRole:        strings.TrimSpace(cfg.Matcher.MessageRole),
		MessageDeltaType:   strings.TrimSpace(cfg.Matcher.MessageDeltaType),
		CompactionReason:   strings.TrimSpace(cfg.Matcher.CompactionReason),
		CompactionStrategy: strings.TrimSpace(cfg.Matcher.CompactionStrategy),
	}
	if cfg.Matcher.ToolReadOnly != nil {
		value := *cfg.Matcher.ToolReadOnly
		matcher.ToolReadOnly = &value
	}

	decl := hookspkg.HookDecl{
		Name:         strings.TrimSpace(cfg.Name),
		Event:        hookspkg.HookEvent(strings.TrimSpace(cfg.Event)),
		Source:       extensionHookSource,
		Mode:         hookspkg.HookMode(strings.TrimSpace(cfg.Mode)),
		Required:     cfg.Required,
		Timeout:      time.Duration(cfg.Timeout),
		Matcher:      matcher,
		ExecutorKind: kind,
		Command:      resolvedCommand,
		Args:         resolvedArgs,
		WorkingDir:   ext.rootDir,
		Env:          resolvedEnv,
		Metadata: map[string]string{
			"extension": ext.info.Name,
		},
	}
	if cfg.Priority != nil {
		decl.Priority = *cfg.Priority
		decl.PrioritySet = true
	}

	if err := hookspkg.ValidateHookDecl(decl); err != nil {
		return hookspkg.HookDecl{}, err
	}
	return decl, nil
}

func (m *Manager) resolveCommand(rootDir string, value string) (string, error) {
	resolved, err := m.resolveString(rootDir, value)
	if err != nil {
		return "", err
	}
	if resolved == "" {
		return "", nil
	}
	if filepath.IsAbs(resolved) {
		return filepath.Clean(resolved), nil
	}
	if strings.ContainsRune(resolved, filepath.Separator) || strings.HasPrefix(resolved, ".") {
		return resolvePathWithinRoot(rootDir, resolved)
	}
	return resolved, nil
}

func (m *Manager) resolveStringSlice(rootDir string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	resolved := make([]string, 0, len(values))
	for _, value := range values {
		item, err := m.resolveString(rootDir, value)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, item)
	}
	return resolved, nil
}

func (m *Manager) resolveStringMap(rootDir string, env map[string]string) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}

	resolved := make(map[string]string, len(env))
	for key, value := range env {
		item, err := m.resolveString(rootDir, value)
		if err != nil {
			return nil, err
		}
		resolved[key] = item
	}
	return resolved, nil
}

func (m *Manager) resolveEnvMap(rootDir string, env map[string]string) ([]string, error) {
	resolvedMap, err := m.resolveStringMap(rootDir, env)
	if err != nil {
		return nil, err
	}

	valuesMap := make(map[string]string, len(safeSubprocessEnvKeys)+len(resolvedMap))
	order := make([]string, 0, len(safeSubprocessEnvKeys)+len(resolvedMap))
	for _, key := range safeSubprocessEnvKeys {
		if _, exists := valuesMap[key]; exists {
			continue
		}
		valuesMap[key] = m.getenv(key)
		order = append(order, key)
	}

	keys := make([]string, 0, len(resolvedMap))
	for key := range resolvedMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		if _, exists := valuesMap[key]; !exists {
			order = append(order, key)
		}
		valuesMap[key] = resolvedMap[key]
	}

	values := make([]string, 0, len(order))
	for _, key := range order {
		values = append(values, key+"="+valuesMap[key])
	}
	return values, nil
}

func (m *Manager) resolveString(rootDir string, value string) (string, error) {
	resolved := strings.TrimSpace(value)
	if resolved == "" {
		return "", nil
	}

	resolved = strings.ReplaceAll(resolved, "{{config_dir}}", rootDir)
	for {
		start := strings.Index(resolved, "{{env:")
		if start < 0 {
			break
		}
		end := strings.Index(resolved[start:], "}}")
		if end < 0 {
			return "", fmt.Errorf("invalid env template %q", value)
		}
		end += start
		key := strings.TrimSpace(strings.TrimPrefix(resolved[start:end], "{{env:"))
		resolved = resolved[:start] + m.getenv(key) + resolved[end+2:]
	}
	return resolved, nil
}

func (m *Manager) setFailure(ext *managedExtension, phase ExtensionPhase, err error) {
	if ext == nil || err == nil {
		return
	}

	m.mu.Lock()
	ext.phase = phase
	ext.lastError = err.Error()
	ext.active = false
	name := ext.info.Name
	m.mu.Unlock()

	m.logger.Error("extension.lifecycle.failed", "extension", name, "phase", phase, "error", err)
}

func (m *Manager) lookupManaged(name string) (*managedExtension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ext := m.extensions[name]
	return ext, ext != nil
}

func (m *Manager) currentProcess(name string, generation int64) (processHandle, time.Duration, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext := m.extensions[name]
	if ext == nil || ext.process == nil || ext.generation != generation {
		return nil, 0, false
	}
	return ext.process, ext.healthInterval, true
}

func (m *Manager) shouldStopSupervision(name string, generation int64, proc processHandle) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.stopping {
		return true
	}
	ext := m.extensions[name]
	return ext == nil || ext.process == nil || ext.process != proc || ext.generation != generation
}

func (m *Manager) recordFailure(name string, reason error) (time.Duration, bool, bool) {
	m.mu.Lock()
	ext := m.extensions[name]
	if ext == nil || m.stopping {
		m.mu.Unlock()
		return 0, false, false
	}

	ext.process = nil
	ext.active = false
	ext.awaitingStability = false
	ext.phase = ExtensionPhaseRecover
	ext.lastExitedAt = m.now()
	ext.lastError = reason.Error()
	ext.consecutiveFailures++
	instanceID := managedChannelInstanceID(ext)
	failures := ext.consecutiveFailures
	if ext.consecutiveFailures >= m.restartFailureThreshold {
		m.mu.Unlock()
		m.reportChannelRuntimeIssue(instanceID, channelspkg.ChannelStatusError, reason)
		m.logger.Error("extension.lifecycle.failed", "extension", name, "phase", ExtensionPhaseRecover, "error", reason, "consecutive_failures", failures)
		return 0, true, true
	}

	ext.restartBackoff = restartBackoff(ext.consecutiveFailures, m.restartBackoffMax)
	backoff := ext.restartBackoff
	m.mu.Unlock()
	m.reportChannelRuntimeIssue(instanceID, channelspkg.ChannelStatusDegraded, reason)

	m.logger.Warn(
		"extension.lifecycle.failed",
		"extension", name,
		"phase", ExtensionPhaseRecover,
		"error", reason,
		"consecutive_failures", failures,
		"restart_backoff_ms", backoff.Milliseconds(),
	)
	return backoff, false, true
}

func (m *Manager) disableExtension(name string, reason error) {
	ext, ok := m.lookupManaged(name)
	if !ok {
		return
	}
	instanceID := managedChannelInstanceID(ext)

	if err := m.registry.Disable(name); err != nil {
		reason = errors.Join(reason, err)
	}
	m.unregisterResources(ext)

	m.mu.Lock()
	defer m.mu.Unlock()
	ext.info.Enabled = false
	ext.phase = ExtensionPhaseRecover
	ext.lastError = reason.Error()
	ext.active = false
	ext.process = nil
	ext.awaitingStability = false
	m.reportChannelRuntimeIssue(instanceID, channelspkg.ChannelStatusError, reason)
}

func (m *Manager) unregisterResources(ext *managedExtension) {
	if ext == nil {
		return
	}
	if len(ext.skills) > 0 && m.skillsRegistry != nil {
		m.skillsRegistry.RemoveExternal(ext.info.Name)
	}
	m.capChecker.Unregister(ext.info.Name)

	m.mu.Lock()
	defer m.mu.Unlock()
	ext.registered = false
}

func (m *Manager) markStable(name string, generation int64) {
	m.mu.Lock()
	ext := m.extensions[name]
	if ext == nil || ext.generation != generation || !ext.awaitingStability {
		m.mu.Unlock()
		return
	}
	instanceID := managedChannelInstanceID(ext)
	ext.awaitingStability = false
	ext.consecutiveFailures = 0
	ext.restartBackoff = 0
	m.mu.Unlock()
	m.clearChannelRuntimeIssue(instanceID)
}

func (m *Manager) statusLocked(ext *managedExtension) ExtensionStatus {
	status := ExtensionStatus{
		Name:                ext.info.Name,
		Version:             ext.info.Version,
		Source:              ext.info.Source,
		Enabled:             ext.info.Enabled,
		Registered:          ext.registered,
		Active:              ext.active,
		Phase:               ext.phase,
		ConsecutiveFailures: ext.consecutiveFailures,
		RestartBackoff:      ext.restartBackoff,
		LastError:           ext.lastError,
		LastStartedAt:       ext.lastStartedAt,
		LastExitedAt:        ext.lastExitedAt,
	}
	if ext.process != nil {
		status.PID = ext.process.PID()
		health := ext.process.HealthState()
		status.Healthy = health.Healthy
		status.HealthMessage = health.Message
		status.HealthLastCheckedAt = health.LastCheckedAt
	} else {
		status.Healthy = ext.active
	}
	return status
}

func (m *Manager) cloneExtension(ext *managedExtension) *Extension {
	if ext == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := &Extension{
		Info:            cloneExtensionInfo(ext.info),
		RootDir:         ext.rootDir,
		GrantedActions:  slices.Clone(ext.grantedActions),
		GrantedSecurity: slices.Clone(ext.grantedSecurity),
		Status:          m.statusLocked(ext),
	}
	if ext.manifest != nil {
		clone.Manifest = cloneManifest(ext.manifest)
	}
	for _, decl := range ext.hooks {
		clone.Hooks = append(clone.Hooks, cloneHookDecl(decl))
	}
	for _, agent := range ext.agents {
		clone.Agents = append(clone.Agents, cloneAgentDef(agent))
	}
	for _, server := range ext.mcpServers {
		clone.MCPServers = append(clone.MCPServers, cloneMCPServer(server))
	}
	if len(ext.skills) > 0 {
		clone.Skills = make([]*skillspkg.Skill, 0, len(ext.skills))
		for _, skill := range ext.skills {
			clone.Skills = append(clone.Skills, cloneSkillSnapshot(skill))
		}
	}
	if ext.initialize != nil {
		clone.InitializeResult = cloneInitializeResponse(ext.initialize)
	}
	return clone
}

func (m *Manager) reportChannelRuntimeIssue(channelInstanceID string, status channelspkg.ChannelStatus, reason error) {
	if m == nil || m.channelTelemetrySink == nil {
		return
	}
	trimmedID := strings.TrimSpace(channelInstanceID)
	if trimmedID == "" || reason == nil {
		return
	}
	m.channelTelemetrySink.RecordChannelRuntimeIssue(trimmedID, status, reason.Error())
}

func (m *Manager) clearChannelRuntimeIssue(channelInstanceID string) {
	if m == nil || m.channelTelemetrySink == nil {
		return
	}
	trimmedID := strings.TrimSpace(channelInstanceID)
	if trimmedID == "" {
		return
	}
	m.channelTelemetrySink.ClearChannelRuntimeIssue(trimmedID)
}

func managedChannelInstanceID(ext *managedExtension) string {
	if ext == nil || ext.runtime.Channel == nil {
		return ""
	}
	return strings.TrimSpace(ext.runtime.Channel.Instance.ID)
}

func (m *Manager) waitBackoff(delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-m.lifecycleDone():
		return false
	}
}

func (m *Manager) lifecycleDone() <-chan struct{} {
	m.mu.RLock()
	ctx := m.lifecycleCtx
	m.mu.RUnlock()

	if ctx == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return ctx.Done()
}

func (m *Manager) lifecycleContext() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.lifecycleCtx == nil {
		return context.Background()
	}
	return m.lifecycleCtx
}

func (m *Manager) healthPollInterval(healthInterval time.Duration) time.Duration {
	if healthInterval <= 0 {
		return m.healthPollCeiling
	}
	interval := healthInterval / 4
	if interval < m.healthPollFloor {
		interval = m.healthPollFloor
	}
	if interval > m.healthPollCeiling {
		interval = m.healthPollCeiling
	}
	return interval
}

func (m *Manager) shutdownDeadlineForProcess(name string, generation int64) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext := m.extensions[name]
	if ext == nil || ext.generation != generation || ext.runtime.ShutdownTimeoutMS <= 0 {
		return m.defaultShutdownTimeout
	}
	return time.Duration(ext.runtime.ShutdownTimeoutMS) * time.Millisecond
}

func restartBackoff(failures int, max time.Duration) time.Duration {
	if failures <= 0 {
		return 0
	}
	delay := time.Second << (failures - 1)
	if delay > max {
		return max
	}
	return delay
}

func loadManifestAtPath(path string) (*Manifest, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".toml":
		return loadManifestTOML(path)
	case ".json":
		return loadManifestJSON(path)
	default:
		return nil, fmt.Errorf("extension: unsupported manifest path %q", path)
	}
}

func phaseError(name string, phase ExtensionPhase, err error) error {
	return fmt.Errorf("extension %q %s: %w", name, phase, err)
}

func requiresSubprocess(manifest *Manifest) bool {
	if manifest == nil {
		return false
	}
	if strings.TrimSpace(manifest.Subprocess.Command) != "" {
		return true
	}
	return len(manifest.Capabilities.Provides) > 0 || len(manifest.Actions.Requires) > 0
}

func durationOr(value Duration, fallback time.Duration) time.Duration {
	if value.IsZero() {
		return fallback
	}
	return time.Duration(value)
}

func validateSupportedHookEvents(values []string) error {
	for _, value := range values {
		if err := hookspkg.HookEvent(strings.TrimSpace(value)).Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) resolveChannelRuntime(ctx context.Context, ext *managedExtension) (*subprocess.InitializeChannelRuntime, error) {
	if ext == nil || ext.manifest == nil {
		return nil, nil
	}
	if !slices.Contains(ext.manifest.Capabilities.Provides, extensionprotocol.CapabilityProvideChannelAdapter) {
		return nil, nil
	}
	if m.channelRuntimeResolver == nil {
		return nil, fmt.Errorf("extension: channel runtime resolver is required for %q", ext.info.Name)
	}

	channelRuntime, err := m.channelRuntimeResolver.ResolveChannelRuntime(ctx, ext.info.Name)
	if err != nil {
		return nil, fmt.Errorf("extension: resolve channel runtime for %q: %w", ext.info.Name, err)
	}
	if channelRuntime == nil {
		return nil, fmt.Errorf("extension: channel runtime is required for %q", ext.info.Name)
	}
	if err := channelRuntime.Validate(); err != nil {
		return nil, fmt.Errorf("extension: resolve channel runtime for %q: %w", ext.info.Name, err)
	}

	resolved := subprocess.CloneInitializeChannelRuntime(channelRuntime)
	if resolved == nil {
		return nil, fmt.Errorf("extension: channel runtime is required for %q", ext.info.Name)
	}
	if strings.TrimSpace(resolved.Instance.ExtensionName) != ext.info.Name {
		return nil, fmt.Errorf(
			"extension: channel runtime instance %q belongs to extension %q, want %q",
			resolved.Instance.ID,
			resolved.Instance.ExtensionName,
			ext.info.Name,
		)
	}

	return resolved, nil
}

func daemonRequestMethods() []string {
	return []string{"execute_hook", "health_check", "shutdown"}
}

func capabilityMethods(provides []string) []string {
	return extensionprotocol.CapabilityServiceMethods(provides)
}

func hostAPIMethodsFromStrings(values []string) []extensionprotocol.HostAPIMethod {
	normalized := normalizeUniqueStrings(values)
	methods := make([]extensionprotocol.HostAPIMethod, 0, len(normalized))
	for _, value := range normalized {
		methods = append(methods, extensionprotocol.HostAPIMethod(value))
	}
	return methods
}

func skillSourceForExtension(source ExtensionSource) skillspkg.SkillSource {
	switch source {
	case SourceBundled:
		return skillspkg.SourceBundled
	case SourceWorkspace:
		return skillspkg.SourceWorkspace
	case SourceMarketplace:
		return skillspkg.SourceMarketplace
	default:
		return skillspkg.SourceUser
	}
}

func resolveResourcePath(rootDir string, value string) (string, error) {
	return resolvePathWithinRoot(rootDir, value)
}

func resolvePathWithinRoot(rootDir string, value string) (string, error) {
	trimmedRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if trimmedRoot == "" {
		return "", errors.New("extension: root directory is required")
	}

	resolved := strings.TrimSpace(value)
	if resolved == "" {
		return "", nil
	}

	var candidate string
	if filepath.IsAbs(resolved) {
		candidate = filepath.Clean(resolved)
	} else {
		candidate = filepath.Clean(filepath.Join(trimmedRoot, resolved))
	}

	rel, err := filepath.Rel(trimmedRoot, candidate)
	if err != nil {
		return "", fmt.Errorf("extension: resolve path %q: %w", resolved, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("extension: path %q escapes extension root %q", resolved, trimmedRoot)
	}

	return candidate, nil
}

func collectMarkdownFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.EqualFold(filepath.Ext(root), ".md") {
			return []string{root}, nil
		}
		return nil, fmt.Errorf("resource path %q is not a markdown file", root)
	}

	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}

func sortedKeys[T any](items map[string]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func cloneMCPServer(server aghconfig.MCPServer) aghconfig.MCPServer {
	cloned := server
	cloned.Args = slices.Clone(server.Args)
	cloned.Env = cloneStringMap(server.Env)
	return cloned
}

func cloneAgentDef(agent aghconfig.AgentDef) aghconfig.AgentDef {
	cloned := agent
	cloned.Tools = slices.Clone(agent.Tools)
	cloned.MCPServers = make([]aghconfig.MCPServer, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		cloned.MCPServers = append(cloned.MCPServers, cloneMCPServer(server))
	}
	cloned.Hooks = make([]hookspkg.HookDecl, 0, len(agent.Hooks))
	for _, decl := range agent.Hooks {
		cloned.Hooks = append(cloned.Hooks, cloneHookDecl(decl))
	}
	return cloned
}

func cloneHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := src
	cloned.Args = slices.Clone(src.Args)
	cloned.Env = cloneStringMap(src.Env)
	cloned.Metadata = cloneStringMap(src.Metadata)
	if src.Matcher.ToolReadOnly != nil {
		value := *src.Matcher.ToolReadOnly
		cloned.Matcher.ToolReadOnly = &value
	}
	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneExtensionInfo(info ExtensionInfo) ExtensionInfo {
	cloned := info
	cloned.Capabilities = normalizeCapabilitiesConfig(info.Capabilities)
	cloned.Actions = normalizeActionsConfig(info.Actions)
	return cloned
}

func cloneManifest(src *Manifest) *Manifest {
	if src == nil {
		return nil
	}

	cloned := *src
	cloned.Resources = normalizeResourcesConfig(src.Resources)
	cloned.Capabilities = normalizeCapabilitiesConfig(src.Capabilities)
	cloned.Actions = normalizeActionsConfig(src.Actions)
	cloned.Subprocess = normalizeSubprocessConfig(src.Subprocess)
	cloned.Security = normalizeSecurityConfig(src.Security)
	return &cloned
}

func cloneSkillSnapshot(skill *skillspkg.Skill) *skillspkg.Skill {
	if skill == nil {
		return nil
	}

	clone := *skill
	clone.Meta = cloneSkillMeta(skill.Meta)
	clone.MCPServers = cloneSkillMCPServers(skill.MCPServers)
	if len(skill.Hooks) > 0 {
		clone.Hooks = make([]hookspkg.HookDecl, 0, len(skill.Hooks))
		for _, decl := range skill.Hooks {
			clone.Hooks = append(clone.Hooks, cloneHookDecl(decl))
		}
	}
	clone.Provenance = cloneSkillProvenance(skill.Provenance)
	return &clone
}

func cloneSkillMeta(meta skillspkg.SkillMeta) skillspkg.SkillMeta {
	cloned := meta
	cloned.Metadata = cloneSkillMetadataMap(meta.Metadata)
	return cloned
}

func cloneSkillMetadataMap(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = cloneSkillMetadataValue(value)
	}
	return cloned
}

func cloneSkillMetadataValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneSkillMetadataMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for index := range typed {
			cloned[index] = cloneSkillMetadataValue(typed[index])
		}
		return cloned
	default:
		return typed
	}
}

func cloneSkillMCPServers(src []skillspkg.MCPServerDecl) []skillspkg.MCPServerDecl {
	if src == nil {
		return nil
	}

	cloned := make([]skillspkg.MCPServerDecl, len(src))
	for index, decl := range src {
		cloned[index] = skillspkg.MCPServerDecl{
			Name:    decl.Name,
			Command: decl.Command,
			Args:    slices.Clone(decl.Args),
			Env:     cloneStringMap(decl.Env),
		}
	}
	return cloned
}

func cloneSkillProvenance(src *skillspkg.Provenance) *skillspkg.Provenance {
	if src == nil {
		return nil
	}
	cloned := *src
	return &cloned
}

func cloneInitializeResponse(src *subprocess.InitializeResponse) *subprocess.InitializeResponse {
	if src == nil {
		return nil
	}

	cloned := *src
	cloned.ImplementedMethods = slices.Clone(src.ImplementedMethods)
	cloned.SupportedHookEvents = slices.Clone(src.SupportedHookEvents)
	cloned.AcceptedCapabilities.Provides = slices.Clone(src.AcceptedCapabilities.Provides)
	cloned.AcceptedCapabilities.Actions = slices.Clone(src.AcceptedCapabilities.Actions)
	cloned.AcceptedCapabilities.Security = slices.Clone(src.AcceptedCapabilities.Security)
	return &cloned
}
