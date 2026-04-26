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
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/environment/daytona"
	"github.com/pedronauck/agh/internal/environment/local"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	aghlogger "github.com/pedronauck/agh/internal/logger"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/situation"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/toolruntime"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type bootState struct {
	cfg                 aghconfig.Config
	logger              *slog.Logger
	closeLogger         func() error
	lock                *Lock
	harnessResolver     *HarnessContextResolver
	harnessRecorder     *harnessLifecycleRecorder
	memoryStore         *memory.Store
	skillsRegistry      *skills.Registry
	mcpResolver         *skills.MCPResolver
	dreamSvc            consolidation.Service
	dreamRuntime        *consolidation.Runtime
	globalMemoryDir     string
	situationContext    *situation.Service
	promptAssembler     session.PromptAssembler
	startupOverlay      session.StartupPromptOverlay
	promptAugmenter     session.PromptInputAugmenter
	notifier            *hooksNotifier
	registry            Registry
	processRegistry     *toolruntime.Registry
	environmentRegistry *environment.Registry
	workspaceResolver   *workspacepkg.Resolver
	sessions            SessionManager
	tasks               *taskRuntime
	scheduler           *schedulerRuntime
	network             networkRuntime
	observer            Observer
	lifecycleObservers  *sessionLifecycleFanout
	hookTelemetrySinks  *hookTelemetryFanout
	hooks               hookRuntime
	hookDispatcher      *hookspkg.Hooks
	hookBindings        hookBindingPublisher
	resourceKernel      *resources.Kernel
	resourceCodecs      *resources.CodecRegistry
	agentCatalog        *resourceCatalog[aghconfig.AgentDef]
	toolCatalog         *resourceCatalog[toolspkg.Tool]
	mcpServerCatalog    *resourceCatalog[aghconfig.MCPServer]
	agentSkillResources agentSkillPublisher
	toolMCPResources    toolMCPPublisher
	bundleResources     bundleResourcePublisher
	extMu               sync.RWMutex
	extensions          extensionRuntime
	resourceReconcile   resources.ReconcileDriver
	automation          automationRuntime
	bridges             *bridgeRuntime
	bundles             *bundlepkg.Service
	httpServer          Server
	udsServer           Server
	skillsCancel        context.CancelFunc
	skillsDone          chan struct{}
	startedAt           time.Time
	info                Info
	deps                RuntimeDeps
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
	if err := d.bootTasks(ctx, state); err != nil {
		return err
	}
	if err := d.bootScheduler(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootNetwork(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootHooks(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootAutomation(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootBundles(ctx, state); err != nil {
		return err
	}
	if err := d.bootResourceReconcile(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootExtensions(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootSettings(ctx, state); err != nil {
		return err
	}
	if err := d.bootServers(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootFinalize(ctx, state); err != nil {
		return err
	}
	if err := d.markRestartReadyIfRequested(state.info); err != nil {
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
		d.resourceReconcile != nil ||
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

func (d *Daemon) bootPromptProviders(_ context.Context, state *bootState) error {
	var prependProviders []session.PromptProvider
	var appendProviders []session.PromptProvider
	var err error

	if state.cfg.Memory.Enabled {
		state.globalMemoryDir = strings.TrimSpace(state.cfg.Memory.GlobalDir)
		if state.globalMemoryDir == "" {
			state.globalMemoryDir = d.homePaths.MemoryDir
		}
		state.memoryStore = memory.NewStore(
			state.globalMemoryDir,
			memory.WithCatalogDatabasePath(d.homePaths.DatabaseFile),
		)
		if err := state.memoryStore.EnsureDirs(); err != nil {
			return fmt.Errorf("daemon: ensure memory store directories: %w", err)
		}
		prependProviders = append(prependProviders, memory.NewAssembler(state.memoryStore))
	}

	if state.cfg.Skills.Enabled {
		skillsCfg, err := d.skillsRegistryConfig(&state.cfg)
		if err != nil {
			return err
		}

		state.skillsRegistry = skills.NewRegistry(skillsCfg, skills.WithLogger(state.logger))
		state.mcpResolver = skills.NewMCPResolver(state.cfg.Skills, state.logger)
		appendProviders = append(appendProviders, skills.NewCatalogProvider(state.skillsRegistry))
	}

	state.situationContext = d.buildSituationContext(state)
	state.harnessResolver = NewHarnessContextResolver(HarnessRuntimeSignals{
		SituationPromptSectionEnabled: state.situationContext != nil,
		MemoryPromptSectionEnabled:    state.memoryStore != nil,
		SkillsPromptSectionEnabled:    state.skillsRegistry != nil,
		SituationAugmenter:            state.situationContext != nil,
		DurableMemoryAugmenter:        state.memoryStore != nil,
		SyntheticTurnsEnabled:         true,
		DetachedTaskRuntimeEnabled:    true,
	})
	state.harnessRecorder = newHarnessLifecycleRecorder(state.logger, d.now)
	state.promptAssembler = NewComposedAssembler(
		WithSectionSelector(NewSectionSelector(state.harnessResolver, state.harnessRecorder)),
		WithPromptSectionDescriptors(
			defaultStartupPromptSectionDescriptorsFromProviders(
				prependProviders,
				appendProviders,
				state.situationContext,
			)...,
		),
	)
	state.promptAugmenter, err = newPromptInputCompositeAugmenter(
		state.logger,
		state.harnessResolver,
		state.harnessRecorder,
		defaultPromptInputAugmenterDescriptors(
			memory.NewRecallAugmenter(state.memoryStore),
			state.situationContext.Augment,
		)...,
	)
	if err != nil {
		return fmt.Errorf("daemon: build prompt input composite: %w", err)
	}
	return nil
}

func (d *Daemon) buildSituationContext(state *bootState) *situation.Service {
	return situation.NewService(situation.Deps{
		Now: d.now,
		WorkspaceResolverFunc: func() situation.WorkspaceResolver {
			return state.workspaceResolver
		},
		AgentResolverFunc: func() situation.AgentResolver {
			return agentCatalogDependency(state.agentCatalog)
		},
		SkillRegistryFunc: func() situation.SkillRegistry {
			return skillRegistryDependency(state.skillsRegistry)
		},
		TaskStoreFunc: func() situation.TaskStore {
			if state.tasks == nil {
				return nil
			}
			return state.tasks.store
		},
		NetworkFunc: func() situation.NetworkReader {
			return state.network
		},
		CoordinatorConfigFunc: func() situation.CoordinatorConfigResolver {
			return state.deps.CoordinatorConfig
		},
	})
}

func (d *Daemon) bootRuntime(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if err := d.bootLockAndSocket(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootRegistryState(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.bootRuntimeServices(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.attachRuntimeObserver(ctx, state); err != nil {
		return err
	}
	return nil
}

func (d *Daemon) bootLockAndSocket(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	pid := d.pid()
	lock, err := d.acquireLock(d.homePaths.DaemonLock, pid)
	if err != nil {
		return err
	}
	cleanup.add(func(context.Context) error {
		return lock.Release()
	})
	state.lock = lock

	if stalePID := d.resolveStaleDaemonPID(lock, pid, state.logger); stalePID > 0 {
		if cleanupErr := d.cleanupOrphans(ctx, stalePID); cleanupErr != nil {
			state.logger.Warn("daemon: cleanup orphan processes failed", "stale_pid", stalePID, "error", cleanupErr)
		}
	}

	if err := removeStaleSocket(state.cfg.Daemon.Socket); err != nil {
		return err
	}
	return nil
}

func (d *Daemon) resolveStaleDaemonPID(lock *Lock, pid int, logger *slog.Logger) int {
	if lock == nil {
		return 0
	}
	if stalePID := lock.StalePID(); stalePID > 0 {
		return stalePID
	}

	existingInfo, err := ReadInfo(d.homePaths.DaemonInfo)
	switch {
	case err == nil && existingInfo.PID > 0 && existingInfo.PID != pid && !d.processAlive(existingInfo.PID):
		return existingInfo.PID
	case err != nil && !errors.Is(err, os.ErrNotExist) && logger != nil:
		logger.Warn("daemon: read stale daemon info failed", "path", d.homePaths.DaemonInfo, "error", err)
	}
	return 0
}

func (d *Daemon) bootRegistryState(
	ctx context.Context,
	state *bootState,
	cleanup *bootCleanup,
) error {
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
	if state.harnessRecorder != nil {
		state.harnessRecorder.SetStore(registry)
	}
	return nil
}

func (d *Daemon) bootRuntimeServices(
	ctx context.Context,
	state *bootState,
	cleanup *bootCleanup,
) error {
	if state.cfg.Memory.Enabled && state.cfg.Memory.Dream.Enabled {
		state.dreamSvc = d.newDreamService(
			memory.WithMemoryStore(state.memoryStore),
			memory.WithSessionsDir(d.homePaths.SessionsDir),
			memory.WithMinHours(state.cfg.Memory.Dream.MinHours),
			memory.WithMinSessions(state.cfg.Memory.Dream.MinSessions),
			memory.WithLogger(state.logger),
			memory.WithWorkspaceResolver(state.workspaceResolver),
		)
	}

	state.startedAt = d.now().UTC()
	state.notifier = newHooksNotifier(state.logger, d.now)
	if err := d.bootProcessRegistry(ctx, state); err != nil {
		return err
	}
	environmentRegistry, err := d.buildEnvironmentRegistry(state)
	if err != nil {
		return err
	}
	state.environmentRegistry = environmentRegistry
	state.bridges = d.composeBridgeRuntime(state, cleanup)

	resourceKernel, err := d.buildResourceKernel(state.registry)
	if err != nil {
		return err
	}
	state.resourceKernel = resourceKernel
	state.resourceCodecs, err = d.buildResourceCodecs(state.bridges)
	if err != nil {
		return err
	}
	bridgeResources, err := bridgeInstanceResourceStore(resourceRawStore(resourceKernel), state.resourceCodecs)
	if err != nil {
		return err
	}
	if state.bridges != nil && bridgeResources != nil {
		state.bridges.setResourceDefinitions(
			bridgeResources,
			resourceReconcileActor(),
			func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
				if state.resourceReconcile == nil {
					return nil
				}
				return state.resourceReconcile.Trigger(ctx, kind, reason)
			},
		)
	}
	state.agentCatalog = newResourceCatalog(cloneAgentDef)

	sessions, err := d.newSessionManager(ctx, d.sessionManagerDeps(state))
	if err != nil {
		return fmt.Errorf("daemon: create session manager: %w", err)
	}
	state.sessions = sessions
	state.deps = d.runtimeDeps(state, sessions)
	resourceService, err := d.buildResourceService(state)
	if err != nil {
		return err
	}
	state.deps.Resources = resourceService
	return nil
}

func (d *Daemon) sessionManagerDeps(state *bootState) SessionManagerDeps {
	return SessionManagerDeps{
		HomePaths: d.homePaths,
		Logger:    state.logger,
		Notifier:  d.sessionNotifier(state),
		Hooks: session.HookSet{
			Session:      state.notifier,
			Environment:  state.notifier,
			Prompt:       state.notifier,
			Events:       state.notifier,
			Agent:        state.notifier,
			Conversation: state.notifier,
			Compaction:   state.notifier,
		},
		PromptAssembler:      state.promptAssembler,
		StartupPromptOverlay: state.startupOverlay,
		PromptInputAugmenter: state.promptAugmenter,
		MemoryStore:          state.memoryStore,
		AgentResolver:        agentCatalogDependency(state.agentCatalog),
		SkillRegistry:        skillRegistryDependency(state.skillsRegistry),
		MCPResolver:          mcpResolverDependency(state.mcpResolver),
		WorkspaceResolver:    state.workspaceResolver,
		EnvironmentRegistry:  state.environmentRegistry,
		SessionSupervision:   state.cfg.Session.Supervision,
		ProcessRegistry:      state.processRegistry,
	}
}

func (d *Daemon) bootProcessRegistry(ctx context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: process registry state is required")
	}
	var store toolruntime.Store
	if processStore, ok := state.registry.(toolruntime.Store); ok {
		store = processStore
	}
	state.processRegistry = toolruntime.NewRegistry(
		store,
		toolruntime.WithLogger(state.logger),
		toolruntime.WithDaemonPID(d.pid()),
		toolruntime.WithNow(d.now),
	)
	report, err := state.processRegistry.ReconcileBoot(ctx)
	if err != nil {
		return fmt.Errorf("daemon: reconcile tool process registry: %w", err)
	}
	if report.Checked > 0 && state.logger != nil {
		state.logger.Info(
			"daemon: reconciled tool process registry",
			"checked", report.Checked,
			"recovered", report.Recovered,
			"stale", report.Stale,
		)
	}
	return nil
}

func (d *Daemon) buildEnvironmentRegistry(state *bootState) (*environment.Registry, error) {
	if state == nil {
		return nil, errors.New("daemon: environment registry state is required")
	}
	registry, err := local.NewRegistry(
		local.WithLogger(state.logger),
		local.WithProcessRegistry(state.processRegistry),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create environment registry: %w", err)
	}
	if err := registry.Register(daytona.NewProvider(
		daytona.WithLogger(state.logger),
		daytona.WithProcessRegistry(state.processRegistry),
	)); err != nil {
		return nil, fmt.Errorf("daemon: register daytona environment provider: %w", err)
	}
	return registry, nil
}

func (d *Daemon) sessionNotifier(state *bootState) session.Notifier {
	if state == nil {
		return nil
	}

	notifier := session.Notifier(state.notifier)
	if state.bridges != nil {
		notifier = extensionpkg.NewBridgeDeliveryNotifier(state.bridges.Broker(), state.notifier)
	}
	return notifier
}

func skillRegistryDependency(registry *skills.Registry) session.SkillRegistry {
	if registry == nil {
		return nil
	}
	return registry
}

func mcpResolverDependency(resolver *skills.MCPResolver) session.MCPResolver {
	if resolver == nil {
		return nil
	}
	return resolver
}

func (d *Daemon) runtimeDeps(state *bootState, sessions SessionManager) RuntimeDeps {
	if state != nil && state.dreamSvc != nil {
		lockPath := memory.ConsolidationLockPath(state.globalMemoryDir)
		state.dreamRuntime = consolidation.NewRuntime(
			state.cfg.Memory.Dream.Enabled,
			state.dreamSvc,
			consolidation.NewSessionSpawner(
				sessions,
				state.workspaceResolver,
				&state.cfg,
				state.globalMemoryDir,
			),
			state.cfg.Memory.Dream.CheckInterval,
			state.logger,
			func() (time.Time, error) {
				return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
			},
		)
	}

	return RuntimeDeps{
		Config:            state.cfg,
		HomePaths:         d.homePaths,
		Logger:            state.logger,
		Sessions:          sessions,
		Bridges:           state.bridges,
		Registry:          state.registry,
		MemoryStore:       state.memoryStore,
		WorkspaceResolver: state.workspaceResolver,
		WorkspaceService:  state.workspaceResolver,
		AgentCatalog:      agentCatalogDependency(state.agentCatalog),
		AgentContext:      state.situationContext,
		CoordinatorConfig: newCoordinatorConfigResolver(
			&state.cfg,
			state.workspaceResolver,
			agentCatalogDependency(state.agentCatalog),
		),
		SkillsRegistry: skillsRegistryAPI(state.skillsRegistry),
		DreamTrigger:   dreamTriggerFromRuntime(state.dreamRuntime),
		StartedAt:      state.startedAt,
	}
}

func skillsRegistryAPI(registry *skills.Registry) core.SkillsRegistry {
	if registry == nil {
		return nil
	}
	return registry
}

func dreamTriggerFromRuntime(runtime *consolidation.Runtime) DreamTrigger {
	if runtime == nil {
		return nil
	}
	return runtime
}

func (d *Daemon) buildResourceKernel(registry Registry) (*resources.Kernel, error) {
	if registry == nil {
		return nil, errors.New("daemon: resource service registry is required")
	}

	dbSource, ok := registry.(extensionDBSource)
	if !ok || dbSource.DB() == nil {
		return nil, nil
	}

	kernel, err := resources.NewKernel(dbSource.DB())
	if err != nil {
		return nil, fmt.Errorf("daemon: create resource kernel: %w", err)
	}
	return kernel, nil
}

func (d *Daemon) buildResourceCodecs(bridges *bridgeRuntime) (*resources.CodecRegistry, error) {
	registry := resources.NewCodecRegistry()
	if err := registerDaemonResourceCodecs(registry, bridges); err != nil {
		return nil, err
	}
	return registry, nil
}

func registerDaemonResourceCodecs(registry *resources.CodecRegistry, bridges *bridgeRuntime) error {
	if err := registerDaemonResourceCodec(registry, "hook binding", newHookBindingCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "tool", toolspkg.NewResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "mcp server", aghconfig.NewMCPServerResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "agent", aghconfig.NewAgentResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "skill", skills.NewResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "automation job", automationpkg.NewJobResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(
		registry,
		"automation trigger",
		automationpkg.NewTriggerResourceCodec,
	); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "bridge instance", func() (
		resources.KindCodec[bridgepkg.BridgeInstanceSpec],
		error,
	) {
		return bridgepkg.NewBridgeInstanceResourceCodec(bridgeProviderLookup(bridges))
	}); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "bundle", bundlepkg.NewBundleResourceCodec); err != nil {
		return err
	}
	return registerDaemonResourceCodec(
		registry,
		"bundle activation",
		bundlepkg.NewActivationResourceCodec,
	)
}

func registerDaemonResourceCodec[T any](
	registry *resources.CodecRegistry,
	label string,
	build func() (resources.KindCodec[T], error),
) error {
	codec, err := build()
	if err != nil {
		return fmt.Errorf("daemon: build %s codec: %w", label, err)
	}
	if err := resources.RegisterCodec(registry, codec); err != nil {
		return fmt.Errorf("daemon: register %s codec: %w", label, err)
	}
	return nil
}

func (d *Daemon) buildResourceService(state *bootState) (core.ResourceService, error) {
	if state == nil {
		return nil, nil
	}
	rawStore := resourceRawStore(state.resourceKernel)
	if rawStore == nil {
		return nil, nil
	}

	service, err := core.NewOperatorResourceService(&core.ResourceServiceConfig{
		RawStore:      rawStore,
		CodecRegistry: state.resourceCodecs,
		Trigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state == nil || state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: create resource service: %w", err)
	}
	return service, nil
}

func (d *Daemon) attachRuntimeObserver(ctx context.Context, state *bootState) error {
	observer, err := d.newObserver(ctx, state.deps)
	if err != nil {
		return fmt.Errorf("daemon: create observer: %w", err)
	}
	state.observer = observer
	state.deps.Observer = observer
	return nil
}

func (d *Daemon) bootResourceReconcile(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil {
		return errors.New("daemon: reconcile boot state is required")
	}
	if d.newResourceReconcile == nil {
		return errors.New("daemon: resource reconcile driver factory is required")
	}
	if state.agentCatalog == nil {
		state.agentCatalog = newResourceCatalog(cloneAgentDef)
	}
	if state.toolCatalog == nil {
		state.toolCatalog = newResourceCatalog(cloneToolSpec)
	}
	if state.mcpServerCatalog == nil {
		state.mcpServerCatalog = newResourceCatalog(cloneDaemonMCPServer)
	}

	driver, err := d.newResourceReconcile(ctx, resourceReconcileDriverDeps{
		Config:           state.cfg,
		Logger:           state.logger,
		Registry:         state.registry,
		ResourceStore:    resourceRawStore(state.resourceKernel),
		CodecRegistry:    state.resourceCodecs,
		Hooks:            state.hookDispatcher,
		AgentCatalog:     state.agentCatalog,
		ToolCatalog:      state.toolCatalog,
		MCPServerCatalog: state.mcpServerCatalog,
		SkillsRegistry:   state.skillsRegistry,
		Automation:       automationResourceTarget(state.automation),
		Bridges:          bridgeResourceTarget(state.bridges),
		Bundles:          state.bundles,
	})
	if err != nil {
		return fmt.Errorf("daemon: create resource reconcile driver: %w", err)
	}
	if driver == nil {
		return errors.New("daemon: resource reconcile driver factory returned nil")
	}

	state.resourceReconcile = driver
	cleanup.add(func(ctx context.Context) error {
		return driver.Close(ctx)
	})
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
		network.WithManagerTaskService(state.deps.Tasks),
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
	if state == nil {
		return errors.New("daemon: hook boot state is required")
	}

	nativeDecls, nativeExecutors := d.initializeHookObservers(state)
	providers := d.hookBindingProviders(state, nativeDecls)
	hooks := hookspkg.NewHooks(d.hookRuntimeOptions(state, nativeExecutors)...)
	hookBindings, err := d.newHookBindingPublisher(state, hooks, providers)
	if err != nil {
		hooks.Close()
		return err
	}
	if err := hookBindings.Sync(ctx); err != nil {
		hooks.Close()
		return fmt.Errorf("daemon: sync hook bindings: %w", err)
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
		state.skillsCancel, state.skillsDone = startSkillsWatcher(
			ctx,
			state.skillsRegistry,
			state.cfg.Skills.PollInterval,
			func(refreshCtx context.Context) error {
				if state.agentSkillResources != nil {
					if err := state.agentSkillResources.Sync(refreshCtx); err != nil {
						return err
					}
				}
				return hookBindings.Sync(refreshCtx)
			},
		)
		cleanup.add(func(context.Context) error {
			stopSkillsWatcher(state.skillsCancel, state.skillsDone)
			return nil
		})
	}

	state.hooks = hooks
	state.hookDispatcher = hooks
	state.hookBindings = hookBindings
	return nil
}

func (d *Daemon) initializeHookObservers(state *bootState) ([]hookspkg.HookDecl, map[string]hookspkg.Executor) {
	state.lifecycleObservers = newSessionLifecycleFanout()
	if state.observer != nil {
		state.lifecycleObservers.Add(state.observer)
	}
	if state.harnessRecorder != nil {
		state.lifecycleObservers.Add(state.harnessRecorder)
	}
	state.hookTelemetrySinks = newHookTelemetryFanout()
	if sink, ok := state.observer.(hookspkg.TelemetrySink); ok {
		state.hookTelemetrySinks.Add(sink)
	}
	return daemonNativeHooks(state.lifecycleObservers, state.dreamRuntime)
}

func (d *Daemon) hookBindingProviders(
	state *bootState,
	nativeDecls []hookspkg.HookDecl,
) []hookBindingDeclarationProvider {
	return []hookBindingDeclarationProvider{
		func(context.Context) ([]hookspkg.HookDecl, error) {
			return hookCloneDeclarations(nativeDecls), nil
		},
		configDeclarationProvider(state.registry, state.workspaceResolver, state.logger),
		agentDeclarationProvider(state.registry, state.workspaceResolver, state.logger),
		skillDeclarationProvider(
			state.skillsRegistry,
			state.registry,
			state.workspaceResolver,
			state.cfg.Skills.AllowedMarketplaceHooks,
			state.logger,
		),
		extensionDeclarationProvider(state.currentExtensionRuntime),
	}
}

func (d *Daemon) hookRuntimeOptions(
	state *bootState,
	nativeExecutors map[string]hookspkg.Executor,
) []hookspkg.Option {
	return []hookspkg.Option{
		hookspkg.WithLogger(state.logger),
		hookspkg.WithNow(d.now),
		hookspkg.WithDebugPatchAudit(strings.EqualFold(state.cfg.Log.Level, "debug")),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(nativeExecutors, state.processRegistry)),
		hookspkg.WithTelemetrySink(state.hookTelemetrySinks),
	}
}

func (d *Daemon) newHookBindingPublisher(
	state *bootState,
	hooks *hookspkg.Hooks,
	providers []hookBindingDeclarationProvider,
) (hookBindingPublisher, error) {
	hookBindings := hookBindingPublisher(hookBindingPublisherFunc(func(reloadCtx context.Context) error {
		decls, err := chainDeclarationProviders(providers...)(reloadCtx)
		if err != nil {
			return err
		}
		nextState, err := hooks.BuildBindingState(decls)
		if err != nil {
			return err
		}
		return hooks.ApplyBindingState(nextState, 0)
	}))
	if state.resourceKernel == nil || state.resourceCodecs == nil {
		return hookBindings, nil
	}

	hookCodec, err := resources.ResolveCodec[hookspkg.HookDecl](state.resourceCodecs, hookBindingResourceKind)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve hook binding codec: %w", err)
	}
	hookStore, err := newHookBindingStore(state.resourceKernel, hookCodec)
	if err != nil {
		return nil, fmt.Errorf("daemon: create hook binding store: %w", err)
	}
	return newHookBindingSourceSyncer(
		hookStore,
		hookCodec,
		hookBindingSyncActor(),
		state.logger,
		func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		providers...,
	), nil
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

	var tasks taskpkg.Manager
	if state.tasks != nil {
		tasks = state.tasks.manager
	}

	manager, err := d.newAutomationManager(automationManagerDeps{
		Store:               store,
		Sessions:            state.sessions,
		Tasks:               tasks,
		WorkspaceResolver:   state.workspaceResolver,
		Config:              state.cfg.Automation,
		Hooks:               state.hooks,
		Logger:              state.logger.With("component", "automation"),
		GlobalWorkspacePath: d.homePaths.HomeDir,
		ResourceStore:       resourceRawStore(state.resourceKernel),
		ResourceCodecs:      state.resourceCodecs,
		ResourceTrigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
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

func (d *Daemon) bootBundles(_ context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: boot bundle state is required")
	}

	dbSource, ok := state.registry.(interface {
		extensionDBSource
	})
	if !ok {
		return nil
	}

	extRegistry := extensionpkg.NewRegistry(dbSource.DB())
	resourceStore, err := newBundleResourceStore(state, d.now)
	if err != nil {
		return err
	}
	if resourceStore == nil {
		return nil
	}
	service := bundlepkg.NewService(
		resourceStore,
		extRegistry,
		func(name string) (*extensionpkg.Extension, error) {
			return loadExtensionSnapshot(extRegistry, state.currentExtensionRuntime(), state.logger, name)
		},
		bundlepkg.WithWorkspaceResolver(state.workspaceResolver),
		bundlepkg.WithConfiguredDefaultChannel(state.cfg.Network.DefaultChannel),
		bundlepkg.WithLogger(state.logger),
		bundlepkg.WithNow(d.now),
	)
	if service == nil {
		return nil
	}
	state.bundles = service
	state.deps.Bundles = service
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
	if err := d.configureExtensionResourcePublishers(state, extRegistry); err != nil {
		return err
	}
	manager := d.newExtensionManager(d.extensionManagerDeps(state, extRegistry))
	if manager == nil {
		state.logger.Warn("daemon: extension manager factory returned nil; skipping extensions")
		return syncExtensionResourcePublishers(ctx, state)
	}

	cleanup.add(func(ctx context.Context) error {
		return manager.Stop(ctx)
	})

	if err := manager.Start(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if extensionRuntimeHasRegisteredEntries(ctx, extRegistry, manager) {
			state.logger.Error(
				"daemon: extension manager start failed; continuing with healthy extensions only",
				"error",
				err,
			)
		} else {
			state.logger.Error("daemon: extension manager start failed; continuing without blocking boot", "error", err)
		}
	}
	if state.bridges != nil {
		state.bridges.setExtensionRuntime(manager)
	}
	state.setExtensionRuntime(manager)
	d.attachExtensionRuntime(ctx, state, extRegistry, manager)

	return nil
}

func (d *Daemon) configureExtensionResourcePublishers(
	state *bootState,
	extRegistry *extensionpkg.Registry,
) error {
	agentSkillResources, err := d.newAgentSkillPublisher(state, extRegistry)
	if err != nil {
		return err
	}
	state.agentSkillResources = agentSkillResources
	toolMCPResources, err := d.newToolMCPPublisher(state, extRegistry)
	if err != nil {
		return err
	}
	state.toolMCPResources = toolMCPResources
	bundleResources, err := d.newBundlePublisher(state, extRegistry)
	if err != nil {
		return err
	}
	state.bundleResources = bundleResources
	return nil
}

func syncExtensionResourcePublishers(ctx context.Context, state *bootState) error {
	if state.agentSkillResources != nil {
		if err := state.agentSkillResources.Sync(ctx); err != nil {
			return err
		}
	}
	if state.hookBindings != nil {
		if err := state.hookBindings.Sync(ctx); err != nil {
			return err
		}
	}
	if state.toolMCPResources != nil {
		if err := state.toolMCPResources.Sync(ctx); err != nil {
			return err
		}
	}
	if state.bundleResources != nil {
		if err := state.bundleResources.Sync(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) extensionManagerDeps(
	state *bootState,
	extRegistry *extensionpkg.Registry,
) extensionManagerDeps {
	return extensionManagerDeps{
		Registry:   extRegistry,
		Extensions: state.cfg.Extensions,
		Sessions:   state.sessions,
		Automation: func() extensionpkg.HostAPIAutomationManager {
			return state.automation
		},
		Tasks:             state.deps.Tasks,
		MemoryStore:       state.memoryStore,
		Observer:          state.observer,
		SkillsRegistry:    state.skillsRegistry,
		WorkspaceResolver: state.workspaceResolver,
		Logger:            state.logger,
		BridgeRegistry:    state.bridges,
		BridgeDedupStore:  bridgeRuntimeDedupStore(state.bridges),
		BridgeBroker:      bridgeRuntimeBroker(state.bridges),
		BridgeRuntime:     state.bridges,
		ResourceStore:     resourceRawStore(state.resourceKernel),
		SourceSessions:    resourceSourceSessions(state.resourceKernel),
		ResourceCodecs:    state.resourceCodecs,
		ResourceTrigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		ProcessRegistry: state.processRegistry,
	}
}

func (d *Daemon) attachExtensionRuntime(
	ctx context.Context,
	state *bootState,
	extRegistry *extensionpkg.Registry,
	manager extensionRuntime,
) {
	state.deps.Extensions = newDaemonExtensionService(
		extRegistry,
		manager,
		state.hookBindings,
		state.agentSkillResources,
		state.toolMCPResources,
		state.bundleResources,
		d.homePaths,
		state.logger,
		d.now,
	)
	if state.agentSkillResources != nil {
		if err := state.agentSkillResources.Sync(ctx); err != nil {
			state.logger.Error("daemon: sync agent/skill resources after extension boot failed", "error", err)
		}
	}
	if state.hookBindings != nil {
		if err := state.hookBindings.Sync(ctx); err != nil {
			state.logger.Error("daemon: sync hook bindings after extension boot failed", "error", err)
		}
	}
	if state.toolMCPResources != nil {
		if err := state.toolMCPResources.Sync(ctx); err != nil {
			state.logger.Error("daemon: sync tool/mcp resources after extension boot failed", "error", err)
		}
	}
	if state.bundleResources != nil {
		if err := state.bundleResources.Sync(ctx); err != nil {
			state.logger.Error("daemon: sync bundle resources after extension boot failed", "error", err)
		}
	}
	if state.hookBindings != nil {
		return
	}
	if rebuildable, ok := state.hooks.(interface {
		Rebuild(context.Context) error
	}); ok {
		if err := rebuildable.Rebuild(ctx); err != nil {
			state.logger.Error("daemon: rebuild hooks after extension boot failed", "error", err)
		}
	}
}

func resourceRawStore(kernel *resources.Kernel) resources.RawStore {
	if kernel == nil {
		return nil
	}
	return kernel
}

func resourceSourceSessions(kernel *resources.Kernel) resources.SourceSessionManager {
	if kernel == nil {
		return nil
	}
	return kernel
}

func extensionRuntimeHasRegisteredEntries(
	ctx context.Context,
	registry *extensionpkg.Registry,
	runtime extensionRuntime,
) bool {
	if ctx == nil || registry == nil || runtime == nil {
		return false
	}
	if err := ctx.Err(); err != nil {
		return false
	}

	infos, err := registry.List()
	if err != nil {
		return false
	}

	for _, info := range infos {
		if !info.Enabled {
			continue
		}

		ext, err := runtime.Get(info.Name)
		if err != nil || ext == nil {
			continue
		}
		if ext.Status.Registered {
			return true
		}
	}

	return false
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

func (d *Daemon) bootSettings(_ context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: boot settings state is required")
	}

	surface := newSettingsRuntimeSurface(d, state)
	service, err := settingspkg.NewService(d.homePaths, settingspkg.Dependencies{
		WorkspaceResolver:          state.workspaceResolver,
		GeneralRuntime:             surface,
		MemoryRuntime:              surface,
		SkillsRuntime:              state.skillsRegistry,
		AutomationRuntime:          surface,
		NetworkRuntime:             surface,
		ObservabilityRuntime:       surface,
		Extensions:                 surface,
		TransportParity:            surface,
		MCPAuth:                    surface,
		RestartActionAvailable:     true,
		ConsolidateActionAvailable: state.dreamRuntime != nil && state.dreamRuntime.Enabled(),
		LogTailAvailable:           strings.TrimSpace(d.homePaths.LogFile) != "",
	})
	if err != nil {
		return fmt.Errorf("daemon: create settings service: %w", err)
	}

	state.deps.Settings = service
	state.deps.SettingsRestart = settingsRestartController{daemon: d}
	return nil
}

func daemonNetworkInfo(
	ctx context.Context,
	cfg aghconfig.NetworkConfig,
	service core.NetworkService,
) (*NetworkInfo, error) {
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
	if state.resourceReconcile != nil {
		if err := state.resourceReconcile.RunBoot(ctx); err != nil {
			return fmt.Errorf("daemon: boot resource reconcile: %w", err)
		}
	}

	d.reconcileDaemonEnvironments(ctx, state)

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
	d.harnessResolver = state.harnessResolver
	d.registry = state.registry
	d.memoryStore = state.memoryStore
	d.situationContext = state.situationContext
	d.sessions = state.sessions
	d.tasks = state.tasks
	d.scheduler = state.scheduler
	d.network = state.network
	d.hooks = state.hooks
	d.extensions = state.currentExtensionRuntime()
	d.bridges = state.bridges
	d.observer = state.observer
	d.resourceReconcile = state.resourceReconcile
	d.agentCatalog = state.agentCatalog
	d.toolCatalog = state.toolCatalog
	d.mcpServerCatalog = state.mcpServerCatalog
	d.automation = state.automation
	d.httpServer = state.httpServer
	d.udsServer = state.udsServer
	d.dreamRuntime = state.dreamRuntime
	d.workspaceResolver = state.workspaceResolver
	d.environmentRegistry = state.environmentRegistry
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

func (d *Daemon) skillsRegistryConfig(cfg *aghconfig.Config) (skills.RegistryConfig, error) {
	userAgentsDir, err := aghconfig.ResolveUserAgentsSkillsDir(d.getenv)
	if err != nil {
		return skills.RegistryConfig{}, err
	}
	if cfg == nil {
		return skills.RegistryConfig{
			BundledFS:     bundled.FS(),
			UserSkillsDir: d.homePaths.SkillsDir,
			UserAgentsDir: userAgentsDir,
		}, nil
	}

	return skills.RegistryConfig{
		BundledFS:      bundled.FS(),
		UserSkillsDir:  d.homePaths.SkillsDir,
		UserAgentsDir:  userAgentsDir,
		DisabledSkills: append([]string(nil), cfg.Skills.DisabledSkills...),
	}, nil
}

func startSkillsWatcher(
	ctx context.Context,
	registry *skills.Registry,
	interval time.Duration,
	afterRefresh func(context.Context) error,
) (context.CancelFunc, chan struct{}) {
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
