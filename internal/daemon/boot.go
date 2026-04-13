package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	core "github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	aghlogger "github.com/pedronauck/agh/internal/logger"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type bootState struct {
	cfg                aghconfig.Config
	logger             *slog.Logger
	closeLogger        func() error
	lock               *Lock
	memoryStore        *memory.Store
	skillsRegistry     *skills.Registry
	mcpResolver        *skills.MCPResolver
	dreamSvc           consolidation.Service
	dreamRuntime       *consolidation.Runtime
	globalMemoryDir    string
	promptAssembler    session.PromptAssembler
	notifier           *hooksNotifier
	registry           Registry
	workspaceResolver  workspacepkg.WorkspaceResolver
	sessions           SessionManager
	network            networkRuntime
	observer           Observer
	lifecycleObservers *sessionLifecycleFanout
	hookTelemetrySinks *hookTelemetryFanout
	hooks              hookRuntime
	extMu              sync.RWMutex
	extensions         extensionRuntime
	automation         automationRuntime
	bridges            *bridgeRuntime
	httpServer         Server
	udsServer          Server
	skillsCancel       context.CancelFunc
	skillsDone         chan struct{}
	startedAt          time.Time
	info               Info
	deps               RuntimeDeps
}

func (s *bootState) currentExtensionRuntime() extensionRuntime {
	if s == nil {
		return nil
	}
	s.extMu.RLock()
	defer s.extMu.RUnlock()
	return s.extensions
}

func (s *bootState) setExtensionRuntime(runtime extensionRuntime) {
	if s == nil {
		return
	}
	s.extMu.Lock()
	defer s.extMu.Unlock()
	s.extensions = runtime
}

type bootCleanup struct {
	fns []func(context.Context) error
}

func (c *bootCleanup) add(fn func(context.Context) error) {
	if fn == nil {
		return
	}
	c.fns = append(c.fns, fn)
}

func (c *bootCleanup) run(err *error) {
	if err == nil || *err == nil {
		return
	}

	var cleanupErrs []error
	for i := len(c.fns) - 1; i >= 0; i-- {
		if cleanupErr := c.fns[i](context.Background()); cleanupErr != nil {
			cleanupErrs = append(cleanupErrs, cleanupErr)
		}
	}
	*err = errors.Join(*err, errors.Join(cleanupErrs...))
}

func (d *Daemon) boot(ctx context.Context) (err error) {
	if ctx == nil {
		return errors.New("daemon: boot context is required")
	}

	if err := d.beginBoot(); err != nil {
		return err
	}
	defer d.finishBoot(&err)

	state := &bootState{}
	cleanup := &bootCleanup{}
	defer cleanup.run(&err)

	if err := d.bootConfig(state, cleanup); err != nil {
		return err
	}
	if err := d.bootPromptProviders(ctx, state); err != nil {
		return err
	}
	if err := d.bootRuntime(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootNetwork(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootHooks(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootExtensions(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootAutomation(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootServers(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootFinalize(ctx, state); err != nil {
		return err
	}

	d.publishBootState(state)
	return nil
}

func (d *Daemon) beginBoot() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.booting ||
		d.lock != nil ||
		d.registry != nil ||
		d.sessions != nil ||
		d.network != nil ||
		d.observer != nil ||
		d.automation != nil ||
		d.bridges != nil {
		return errors.New("daemon: already booted")
	}
	d.booting = true
	return nil
}

func (d *Daemon) finishBoot(err *error) {
	if err == nil || *err == nil {
		return
	}
	d.mu.Lock()
	d.booting = false
	d.mu.Unlock()
}

func (d *Daemon) bootConfig(state *bootState, cleanup *bootCleanup) error {
	cfg, err := d.loadConfig()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("daemon: validate config: %w", err)
	}
	if err := aghconfig.EnsureHomeLayout(d.homePaths); err != nil {
		return fmt.Errorf("daemon: ensure home layout: %w", err)
	}

	logger := d.logger
	closeLogger := d.closeLogger
	if logger == nil {
		logger, closeLogger, err = aghlogger.New(
			aghlogger.WithLevel(cfg.Log.Level),
			aghlogger.WithFile(d.homePaths.LogFile),
		)
		if err != nil {
			return fmt.Errorf("daemon: create logger: %w", err)
		}
	}
	if closeLogger == nil {
		closeLogger = func() error { return nil }
	}

	state.cfg = cfg
	state.logger = logger
	state.closeLogger = closeLogger
	cleanup.add(func(context.Context) error {
		return closeLogger()
	})
	return nil
}

func (d *Daemon) bootPromptProviders(ctx context.Context, state *bootState) error {
	var prependProviders []session.PromptProvider
	var appendProviders []session.PromptProvider

	if state.cfg.Memory.Enabled {
		state.globalMemoryDir = strings.TrimSpace(state.cfg.Memory.GlobalDir)
		if state.globalMemoryDir == "" {
			state.globalMemoryDir = d.homePaths.MemoryDir
		}
		state.memoryStore = memory.NewStore(state.globalMemoryDir)
		if err := state.memoryStore.EnsureDirs(); err != nil {
			return fmt.Errorf("daemon: ensure memory store directories: %w", err)
		}
		prependProviders = append(prependProviders, memory.NewAssembler(state.memoryStore))
	}

	if state.cfg.Skills.Enabled {
		skillsCfg, err := d.skillsRegistryConfig(state.cfg)
		if err != nil {
			return err
		}

		state.skillsRegistry = skills.NewRegistry(skillsCfg, skills.WithLogger(state.logger))
		if err := state.skillsRegistry.LoadAll(ctx); err != nil {
			return fmt.Errorf("daemon: load skills registry: %w", err)
		}
		state.mcpResolver = skills.NewMCPResolver(state.cfg.Skills, state.logger)
		appendProviders = append(appendProviders, skills.NewCatalogProvider(state.skillsRegistry))
	}

	state.promptAssembler = NewComposedAssembler(
		WithPrependPromptProviders(prependProviders...),
		WithAppendPromptProviders(appendProviders...),
	)
	return nil
}

func (d *Daemon) bootRuntime(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	pid := d.pid()
	lock, err := d.acquireLock(d.homePaths.DaemonLock, pid)
	if err != nil {
		return err
	}
	cleanup.add(func(context.Context) error {
		return lock.Release()
	})
	state.lock = lock

	stalePID := lock.StalePID()
	if stalePID == 0 {
		existingInfo, readErr := ReadInfo(d.homePaths.DaemonInfo)
		switch {
		case readErr == nil && existingInfo.PID > 0 && existingInfo.PID != pid && !d.processAlive(existingInfo.PID):
			stalePID = existingInfo.PID
		case readErr != nil && !errors.Is(readErr, os.ErrNotExist):
			state.logger.Warn("daemon: read stale daemon info failed", "path", d.homePaths.DaemonInfo, "error", readErr)
		}
	}
	if stalePID > 0 {
		if cleanupErr := d.cleanupOrphans(ctx, stalePID); cleanupErr != nil {
			state.logger.Warn("daemon: cleanup orphan processes failed", "stale_pid", stalePID, "error", cleanupErr)
		}
	}

	if err := removeStaleSocket(state.cfg.Daemon.Socket); err != nil {
		return err
	}

	registry, err := d.openRegistry(ctx, d.homePaths.DatabaseFile)
	if err != nil {
		return fmt.Errorf("daemon: open global database %q: %w", d.homePaths.DatabaseFile, err)
	}
	cleanup.add(func(ctx context.Context) error {
		return registry.Close(ctx)
	})

	workspaceResolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(d.homePaths),
		workspacepkg.WithLogger(state.logger),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(d.homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		return fmt.Errorf("daemon: create workspace resolver: %w", err)
	}
	state.registry = registry
	state.workspaceResolver = workspaceResolver

	if state.cfg.Memory.Enabled && state.cfg.Memory.Dream.Enabled {
		state.dreamSvc = d.newDreamService(
			memory.WithMemoryStore(state.memoryStore),
			memory.WithSessionsDir(d.homePaths.SessionsDir),
			memory.WithMinHours(state.cfg.Memory.Dream.MinHours),
			memory.WithMinSessions(state.cfg.Memory.Dream.MinSessions),
			memory.WithLogger(state.logger),
			memory.WithWorkspaceResolver(workspaceResolver),
		)
	}

	state.startedAt = d.now().UTC()
	state.notifier = newHooksNotifier(state.logger, d.now)
	state.bridges = d.composeBridgeRuntime(state, cleanup)

	sessionNotifier := session.Notifier(state.notifier)
	if state.bridges != nil {
		sessionNotifier = extensionpkg.NewBridgeDeliveryNotifier(state.bridges.Broker(), state.notifier)
	}

	var skillRegistryDep session.SkillRegistry
	if state.skillsRegistry != nil {
		skillRegistryDep = state.skillsRegistry
	}
	var mcpResolverDep session.MCPResolver
	if state.mcpResolver != nil {
		mcpResolverDep = state.mcpResolver
	}

	sessions, err := d.newSessionManager(ctx, SessionManagerDeps{
		HomePaths: d.homePaths,
		Logger:    state.logger,
		Notifier:  sessionNotifier,
		Hooks: session.HookSet{
			Session:      state.notifier,
			Prompt:       state.notifier,
			Events:       state.notifier,
			Agent:        state.notifier,
			Conversation: state.notifier,
			Compaction:   state.notifier,
		},
		PromptAssembler:   state.promptAssembler,
		SkillRegistry:     skillRegistryDep,
		MCPResolver:       mcpResolverDep,
		WorkspaceResolver: workspaceResolver,
	})
	if err != nil {
		return fmt.Errorf("daemon: create session manager: %w", err)
	}

	dreamSpawner := consolidation.NewSessionSpawner(sessions, workspaceResolver, state.cfg, state.globalMemoryDir)
	var dreamTrigger DreamTrigger
	if state.dreamSvc != nil {
		lockPath := memory.ConsolidationLockPath(state.globalMemoryDir)
		state.dreamRuntime = consolidation.NewRuntime(
			state.cfg.Memory.Dream.Enabled,
			state.dreamSvc,
			dreamSpawner,
			state.cfg.Memory.Dream.CheckInterval,
			state.logger,
			func() (time.Time, error) {
				return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
			},
		)
		dreamTrigger = state.dreamRuntime
	}

	var skillsRegistryAPI core.SkillsRegistry
	if state.skillsRegistry != nil {
		skillsRegistryAPI = state.skillsRegistry
	}

	state.deps = RuntimeDeps{
		Config:            state.cfg,
		HomePaths:         d.homePaths,
		Logger:            state.logger,
		Sessions:          sessions,
		Bridges:           state.bridges,
		Registry:          registry,
		MemoryStore:       state.memoryStore,
		WorkspaceResolver: workspaceResolver,
		WorkspaceService:  workspaceResolver,
		SkillsRegistry:    skillsRegistryAPI,
		DreamTrigger:      dreamTrigger,
		StartedAt:         state.startedAt,
	}

	observer, err := d.newObserver(ctx, state.deps)
	if err != nil {
		return fmt.Errorf("daemon: create observer: %w", err)
	}

	state.sessions = sessions
	state.observer = observer
	state.deps.Observer = observer
	return nil
}

func (d *Daemon) bootNetwork(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil {
		return errors.New("daemon: boot network state is required")
	}
	if !state.cfg.Network.Enabled {
		return nil
	}
	if state.sessions == nil {
		return errors.New("daemon: session manager is required before booting network")
	}

	bindable, ok := state.sessions.(networkBindableSessionManager)
	if !ok {
		return errMissingNetworkBindingSurface
	}

	manager, err := network.NewManager(
		ctx,
		state.cfg.Network,
		bindable,
		d.homePaths.NetworkAuditFile,
		state.registry,
		network.WithManagerLogger(state.logger),
	)
	if err != nil {
		return fmt.Errorf("daemon: create network manager: %w", err)
	}

	bindable.SetNetworkPeerLifecycle(manager)
	bindable.SetTurnEndNotifier(manager.OnTurnEnd)
	cleanup.add(func(ctx context.Context) error {
		return manager.Shutdown(ctx)
	})

	state.network = manager
	state.deps.Network = manager
	return nil
}

func (d *Daemon) bootHooks(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	state.lifecycleObservers = newSessionLifecycleFanout()
	if state.observer != nil {
		state.lifecycleObservers.Add(state.observer)
	}
	state.hookTelemetrySinks = newHookTelemetryFanout()
	if sink, ok := state.observer.(hookspkg.TelemetrySink); ok {
		state.hookTelemetrySinks.Add(sink)
	}

	nativeDecls, nativeExecutors := daemonNativeHooks(state.lifecycleObservers, state.dreamRuntime)
	hookOptions := []hookspkg.Option{
		hookspkg.WithLogger(state.logger),
		hookspkg.WithNow(d.now),
		hookspkg.WithDebugPatchAudit(strings.EqualFold(state.cfg.Log.Level, "debug")),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(nativeExecutors)),
		hookspkg.WithNativeDeclarations(nativeDecls),
		hookspkg.WithConfigDeclarationProvider(chainDeclarationProviders(
			configDeclarationProvider(state.registry, state.workspaceResolver, state.logger),
			extensionDeclarationProvider(state.currentExtensionRuntime),
		)),
		hookspkg.WithAgentDeclarationProvider(agentDeclarationProvider(state.registry, state.workspaceResolver, state.logger)),
		hookspkg.WithSkillDeclarationProvider(skillDeclarationProvider(state.skillsRegistry, state.registry, state.workspaceResolver, state.cfg.Skills.AllowedMarketplaceHooks, state.logger)),
		hookspkg.WithTelemetrySink(state.hookTelemetrySinks),
	}

	hooks := hookspkg.NewHooks(hookOptions...)
	if err := hooks.Rebuild(ctx); err != nil {
		hooks.Close()
		return fmt.Errorf("daemon: rebuild hooks: %w", err)
	}
	if hookAwareObserver, ok := state.observer.(interface {
		AttachHooks(observe.HookCatalogSource)
	}); ok {
		hookAwareObserver.AttachHooks(hooks)
	}
	state.notifier.setRuntime(hooks, state.observer)
	cleanup.add(func(context.Context) error {
		hooks.Close()
		return nil
	})

	if state.skillsRegistry != nil {
		state.skillsCancel, state.skillsDone = startSkillsWatcher(ctx, state.skillsRegistry, state.cfg.Skills.PollInterval, func(refreshCtx context.Context) error {
			return hooks.Rebuild(refreshCtx)
		})
		cleanup.add(func(context.Context) error {
			stopSkillsWatcher(state.skillsCancel, state.skillsDone)
			return nil
		})
	}

	state.hooks = hooks
	return nil
}

func (d *Daemon) bootAutomation(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil {
		return nil
	}
	if !state.cfg.Automation.Enabled {
		state.logger.Info("daemon: automation disabled")
		return nil
	}

	store, ok := state.registry.(automationpkg.Store)
	if !ok {
		return errors.New("daemon: global registry does not implement automation store")
	}
	if d.newAutomationManager == nil {
		return errors.New("daemon: automation manager factory is required")
	}

	manager, err := d.newAutomationManager(automationManagerDeps{
		Store:               store,
		Sessions:            state.sessions,
		WorkspaceResolver:   state.workspaceResolver,
		Config:              state.cfg.Automation,
		Hooks:               state.hooks,
		Logger:              state.logger.With("component", "automation"),
		GlobalWorkspacePath: d.homePaths.HomeDir,
	})
	if err != nil {
		return fmt.Errorf("daemon: create automation manager: %w", err)
	}
	if manager == nil {
		return errors.New("daemon: automation manager factory returned nil")
	}
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start automation manager: %w", err)
	}

	cleanup.add(func(ctx context.Context) error {
		return manager.Shutdown(ctx)
	})
	if state.lifecycleObservers != nil {
		state.lifecycleObservers.Add(manager.SessionObserver())
	}
	if state.hookTelemetrySinks != nil {
		state.hookTelemetrySinks.Add(manager.HookTelemetrySink())
	}

	state.automation = manager
	state.deps.Automation = manager
	return nil
}

func (d *Daemon) bootExtensions(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil || state.registry == nil {
		return nil
	}

	dbSource, ok := state.registry.(extensionDBSource)
	if !ok || dbSource.DB() == nil {
		state.logger.Warn("daemon: skipping extensions because global registry does not expose a SQL database handle")
		return nil
	}

	extRegistry := extensionpkg.NewRegistry(dbSource.DB())
	manager := d.newExtensionManager(extensionManagerDeps{
		Registry: extRegistry,
		Sessions: state.sessions,
		Automation: func() extensionpkg.HostAPIAutomationManager {
			return state.automation
		},
		MemoryStore:       state.memoryStore,
		Observer:          state.observer,
		SkillsRegistry:    state.skillsRegistry,
		WorkspaceResolver: state.workspaceResolver,
		Logger:            state.logger,
		BridgeRegistry:    state.bridges,
		BridgeDedupStore:  bridgeRuntimeDedupStore(state.bridges),
		BridgeBroker:      bridgeRuntimeBroker(state.bridges),
		BridgeRuntime:     state.bridges,
	})
	if manager == nil {
		state.logger.Warn("daemon: extension manager factory returned nil; skipping extensions")
		return nil
	}

	if state.bridges != nil {
		state.bridges.setExtensionRuntime(manager)
	}
	state.setExtensionRuntime(manager)
	state.deps.Extensions = newDaemonExtensionService(extRegistry, manager, state.hooks, state.logger, d.now)
	cleanup.add(func(ctx context.Context) error {
		return manager.Stop(ctx)
	})

	if err := manager.Start(ctx); err != nil {
		state.logger.Error("daemon: extension manager start failed; continuing without blocking boot", "error", err)
	}
	if state.hooks != nil {
		if err := state.hooks.Rebuild(ctx); err != nil {
			state.logger.Error("daemon: rebuild hooks after extension boot failed; continuing without extension hooks", "error", err)
		}
	}

	return nil
}

func (d *Daemon) bootServers(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	httpServer, err := d.httpFactory(ctx, state.deps)
	if err != nil {
		return fmt.Errorf("daemon: create http server: %w", err)
	}
	if err := httpServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start http server: %w", err)
	}
	cleanup.add(func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	})

	udsServer, err := d.udsFactory(ctx, state.deps)
	if err != nil {
		return fmt.Errorf("daemon: create uds server: %w", err)
	}
	if err := udsServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start uds server: %w", err)
	}
	cleanup.add(func(ctx context.Context) error {
		return udsServer.Shutdown(ctx)
	})

	networkInfo, err := daemonNetworkInfo(ctx, state.cfg.Network, state.deps.Network)
	if err != nil {
		return err
	}
	info := Info{
		PID:       d.pid(),
		Port:      resolveDaemonPort(state.cfg.HTTP.Port, httpServer),
		StartedAt: state.startedAt,
		Network:   networkInfo,
	}
	if err := WriteInfo(d.homePaths.DaemonInfo, info); err != nil {
		return err
	}
	cleanup.add(func(context.Context) error {
		return RemoveInfo(d.homePaths.DaemonInfo)
	})

	state.httpServer = httpServer
	state.udsServer = udsServer
	state.info = info
	return nil
}

func daemonNetworkInfo(ctx context.Context, cfg aghconfig.NetworkConfig, service core.NetworkService) (*NetworkInfo, error) {
	if !cfg.Enabled {
		return &NetworkInfo{
			Enabled: false,
			Status:  network.StatusDisabled,
		}, nil
	}
	if service == nil {
		return nil, errors.New("daemon: network service is required when network is enabled")
	}

	status, err := service.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: read network status: %w", err)
	}
	if status == nil {
		return nil, errors.New("daemon: network status is required")
	}

	return &NetworkInfo{
		Enabled:      status.Enabled,
		Status:       strings.TrimSpace(status.Status),
		ListenerHost: strings.TrimSpace(status.ListenerHost),
		ListenerPort: status.ListenerPort,
	}, nil
}

func (d *Daemon) bootFinalize(ctx context.Context, state *bootState) error {
	reconcileResult, err := state.observer.Reconcile(ctx)
	if err != nil {
		return fmt.Errorf("daemon: reconcile sessions: %w", err)
	}
	state.logger.Info(
		"daemon: boot reconciliation complete",
		"indexed_sessions", len(reconcileResult.Indexed),
		"orphaned_sessions", len(reconcileResult.Orphaned),
	)

	if d.shouldVerifyBoundaries() {
		if boundaryErr := d.Boundaries(ctx); boundaryErr != nil {
			state.logger.Warn("daemon: boundary verification warning", "error", boundaryErr)
		}
	}
	return nil
}

func (d *Daemon) publishBootState(state *bootState) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config = state.cfg
	d.logger = state.logger
	d.closeLogger = state.closeLogger
	d.booting = false
	d.lock = state.lock
	d.registry = state.registry
	d.memoryStore = state.memoryStore
	d.sessions = state.sessions
	d.network = state.network
	d.hooks = state.hooks
	d.extensions = state.currentExtensionRuntime()
	d.bridges = state.bridges
	d.observer = state.observer
	d.automation = state.automation
	d.httpServer = state.httpServer
	d.udsServer = state.udsServer
	d.dreamRuntime = state.dreamRuntime
	d.workspaceResolver = state.workspaceResolver
	d.skillsRegistry = state.skillsRegistry
	d.skillsCancel = state.skillsCancel
	d.skillsDone = state.skillsDone
	d.startedAt = state.startedAt
	d.info = state.info
	if !d.readyClosed {
		close(d.readyCh)
		d.readyClosed = true
	}
}

func (d *Daemon) skillsRegistryConfig(cfg aghconfig.Config) (skills.RegistryConfig, error) {
	userAgentsDir, err := aghconfig.ResolveUserAgentsSkillsDir(d.getenv)
	if err != nil {
		return skills.RegistryConfig{}, err
	}

	return skills.RegistryConfig{
		BundledFS:      bundled.FS(),
		UserSkillsDir:  d.homePaths.SkillsDir,
		UserAgentsDir:  userAgentsDir,
		DisabledSkills: append([]string(nil), cfg.Skills.DisabledSkills...),
	}, nil
}

func startSkillsWatcher(ctx context.Context, registry *skills.Registry, interval time.Duration, afterRefresh func(context.Context) error) (context.CancelFunc, chan struct{}) {
	if registry == nil {
		return nil, nil
	}

	watcherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	watcher := skills.NewWatcher(registry, interval)
	watcher.SetAfterRefresh(afterRefresh)
	go func() {
		defer close(done)
		watcher.Start(watcherCtx)
	}()
	return cancel, done
}

func stopSkillsWatcher(cancel context.CancelFunc, done <-chan struct{}) {
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func resolveDaemonPort(defaultPort int, server Server) int {
	type portReporter interface {
		Port() int
	}

	if reporter, ok := server.(portReporter); ok && reporter.Port() >= 0 {
		return reporter.Port()
	}
	return defaultPort
}

func (d *Daemon) composeBridgeRuntime(state *bootState, cleanup *bootCleanup) *bridgeRuntime {
	if state == nil || state.registry == nil {
		return nil
	}

	store, ok := state.registry.(bridgeRuntimeStore)
	if !ok {
		if state.logger != nil {
			state.logger.Debug("daemon: skipping bridge runtime because registry does not expose bridge persistence")
		}
		return nil
	}

	runtime := newBridgeRuntime(store, state.logger, d.now, d.bridgeSecretResolver)
	if runtime == nil {
		return nil
	}
	if cleanup != nil {
		cleanup.add(func(context.Context) error {
			runtime.Close()
			return nil
		})
	}
	return runtime
}

func bridgeRuntimeDedupStore(runtime *bridgeRuntime) bridgeDedupStore {
	if runtime == nil {
		return nil
	}
	return runtime.store
}

func bridgeRuntimeBroker(runtime *bridgeRuntime) *bridgepkg.Broker {
	if runtime == nil {
		return nil
	}
	return runtime.Broker()
}

func loadConfigFromHome(homePaths aghconfig.HomePaths) (aghconfig.Config, error) {
	cfg := aghconfig.DefaultWithHome(homePaths)
	if err := aghconfig.ApplyConfigOverlayFile(homePaths.ConfigFile, &cfg); err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: load global config: %w", err)
	}

	socketPath, err := aghconfig.ResolvePath(cfg.Daemon.Socket)
	if err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: normalize daemon socket path: %w", err)
	}
	if strings.TrimSpace(socketPath) != "" {
		cfg.Daemon.Socket = socketPath
	}

	if err := cfg.Validate(); err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: validate config: %w", err)
	}

	return cfg, nil
}
