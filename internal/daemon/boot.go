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
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/heartbeat"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	aghlogger "github.com/pedronauck/agh/internal/logger"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	memcontract "github.com/pedronauck/agh/internal/memory/contract"
	localprovider "github.com/pedronauck/agh/internal/memory/provider/local"
	"github.com/pedronauck/agh/internal/memory/provider/local/memstore"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/sandbox"
	"github.com/pedronauck/agh/internal/sandbox/daytona"
	"github.com/pedronauck/agh/internal/sandbox/local"
	"github.com/pedronauck/agh/internal/session"
	sessionledger "github.com/pedronauck/agh/internal/sessions/ledger"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/situation"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/toolruntime"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	"github.com/pedronauck/agh/internal/vault"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
	skillbundled "github.com/pedronauck/agh/skills"
)

type bootState struct {
	cfg                    aghconfig.Config
	logger                 *slog.Logger
	closeLogger            func() error
	lock                   *Lock
	harnessResolver        *HarnessContextResolver
	harnessRecorder        *harnessLifecycleRecorder
	memoryStore            *memory.Store
	localMemoryProvider    *localprovider.Provider
	memoryProviderRegistry *extensionpkg.MemoryProviderRegistry
	memoryExtractor        *daemonMemoryExtractor
	ledgerMaterializer     session.LedgerMaterializer
	skillsRegistry         *skills.Registry
	mcpResolver            *skills.MCPResolver
	dreamSvc               consolidation.Service
	dreamRuntime           *consolidation.Runtime
	globalMemoryDir        string
	situationContext       *situation.Service
	promptAssembler        session.PromptAssembler
	startupOverlay         session.StartupPromptOverlay
	promptAugmenter        session.PromptInputAugmenter
	notifier               *hooksNotifier
	registry               Registry
	processRegistry        *toolruntime.Registry
	sandboxRegistry        *sandbox.Registry
	workspaceResolver      *workspacepkg.Resolver
	sessions               SessionManager
	hostedMCP              *mcppkg.HostedService
	providerVault          *vault.Service
	modelCatalog           *modelCatalogRuntime
	tasks                  *taskRuntime
	reviewRequests         *runReviewRequestedForwarder
	spawnReaper            *spawnReaper
	scheduler              *schedulerRuntime
	coordinator            *coordinatorRuntime
	network                networkRuntime
	toolRegistry           toolspkg.Registry
	toolsets               core.ToolsetRegistry
	toolApprovals          toolspkg.ApprovalTokenIssuer
	observer               Observer
	lifecycleObservers     *sessionLifecycleFanout
	hookTelemetrySinks     *hookTelemetryFanout
	hooks                  hookRuntime
	hookDispatcher         *hookspkg.Hooks
	hookBindings           hookBindingPublisher
	resourceKernel         *resources.Kernel
	resourceCodecs         *resources.CodecRegistry
	agentCatalog           *resourceCatalog[aghconfig.AgentDef]
	soulCatalog            *resourceCatalog[soul.ResourceSpec]
	heartbeatCatalog       *resourceCatalog[heartbeat.ResourceSpec]
	toolCatalog            *resourceCatalog[toolspkg.Tool]
	mcpServerCatalog       *resourceCatalog[aghconfig.MCPServer]
	agentSkillResources    agentSkillPublisher
	toolMCPResources       toolMCPPublisher
	bundleResources        bundleResourcePublisher
	extMu                  sync.RWMutex
	extensions             extensionRuntime
	resourceReconcile      resources.ReconcileDriver
	automation             automationRuntime
	bridges                *bridgeRuntime
	bundles                *bundlepkg.Service
	httpServer             Server
	udsServer              Server
	skillsCancel           context.CancelFunc
	skillsDone             chan struct{}
	startedAt              time.Time
	info                   Info
	deps                   RuntimeDeps
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

func (c *bootCleanup) run(ctx context.Context, err *error) {
	if err == nil || *err == nil {
		return
	}
	if ctx == nil {
		ctx = context.WithoutCancel(context.TODO())
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
	defer cancel()

	var cleanupErrs []error
	for i := len(c.fns) - 1; i >= 0; i-- {
		if cleanupErr := c.fns[i](cleanupCtx); cleanupErr != nil {
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
	defer cleanup.run(ctx, &err)

	if err := d.bootComponents(ctx, state, cleanup); err != nil {
		return err
	}
	if err := d.markRestartReadyIfRequested(state.info); err != nil {
		return err
	}

	d.publishBootState(state)
	return nil
}

func (d *Daemon) bootComponents(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	steps := []func() error{
		func() error { return d.bootConfig(state, cleanup) },
		func() error { return d.bootPromptProviders(ctx, state) },
		func() error { return d.bootRuntime(ctx, state, cleanup) },
		func() error { return d.bootSessionRepair(ctx, state) },
		func() error { return d.bootTasks(ctx, state) },
		func() error { return d.bootSpawnReaper(ctx, state, cleanup) },
		func() error { return d.bootScheduler(ctx, state, cleanup) },
		func() error { return d.bootNetwork(ctx, state, cleanup) },
		func() error { return d.bootHooks(ctx, state, cleanup) },
		func() error { return d.bootToolRegistry(ctx, state) },
		func() error { return d.bootCoordinator(ctx, state, cleanup) },
		func() error { return d.bootTaskRoles(ctx, state) },
		func() error { return d.bootAutomation(ctx, state, cleanup) },
		func() error { return d.bootBundles(ctx, state) },
		func() error { return d.bootResourceReconcile(ctx, state, cleanup) },
		func() error { return d.bootExtensions(ctx, state, cleanup) },
		func() error { return d.bootSettings(ctx, state) },
		func() error { return d.bootServers(ctx, state, cleanup) },
		func() error { return d.bootFinalize(ctx, state) },
	}
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("daemon: boot canceled: %w", err)
		}
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) beginBoot() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.booting ||
		d.lock != nil ||
		d.registry != nil ||
		d.sessions != nil ||
		d.modelCatalog != nil ||
		d.network != nil ||
		d.toolRegistry != nil ||
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
			aghlogger.WithMirrorToStderr(aghlogger.MirrorToStderrEnabled(os.Getenv)),
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
		provider, err := d.bootMemoryPromptProvider(ctx, state)
		if err != nil {
			return err
		}
		prependProviders = append(prependProviders, provider)
	}

	if state.cfg.Skills.Enabled {
		skillsCfg := d.skillsRegistryConfig(&state.cfg)
		state.skillsRegistry = skills.NewRegistry(skillsCfg, skills.WithLogger(state.logger))
		state.mcpResolver = skills.NewMCPResolver(state.cfg.Skills, state.logger)
		appendProviders = append(appendProviders, skills.NewCatalogProvider(state.skillsRegistry))
	}

	state.situationContext = d.buildSituationContext(state)
	state.harnessResolver = NewHarnessContextResolver(HarnessRuntimeSignals{
		SituationPromptSectionEnabled: state.situationContext != nil,
		MemoryPromptSectionEnabled:    state.memoryStore != nil,
		SkillsPromptSectionEnabled:    state.skillsRegistry != nil,
		ToolsPromptSectionEnabled:     state.cfg.Tools.Enabled,
		SkillsAugmenter:               state.skillsRegistry != nil,
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
	promptAugmenter, err := newPromptInputCompositeAugmenter(
		state.logger,
		state.harnessResolver,
		state.harnessRecorder,
		defaultPromptInputAugmenterDescriptors(
			memory.NewRecallAugmenter(state.memoryStore),
			newSkillsCatalogAugmenter(state.skillsRegistry, func() promptSkillsWorkspaceResolver {
				return state.workspaceResolver
			}),
			state.situationContext.Augment,
		)...,
	)
	if err != nil {
		return fmt.Errorf("daemon: build prompt input composite: %w", err)
	}
	state.promptAugmenter = promptAugmenter
	return nil
}

func (d *Daemon) bootMemoryPromptProvider(
	ctx context.Context,
	state *bootState,
) (session.PromptProvider, error) {
	state.globalMemoryDir = strings.TrimSpace(state.cfg.Memory.GlobalDir)
	if state.globalMemoryDir == "" {
		state.globalMemoryDir = d.homePaths.MemoryDir
	}
	state.memoryStore = memory.NewStore(
		state.globalMemoryDir,
		memory.WithCatalogDatabasePath(d.homePaths.DatabaseFile),
		memory.WithRecallSignalRecorderConfig(state.cfg.Memory.Recall.Signals),
	)
	if err := state.memoryStore.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("daemon: ensure memory store directories: %w", err)
	}
	state.localMemoryProvider = localprovider.New(
		memstore.New(state.memoryStore),
		localprovider.WithLogger(state.logger),
		localprovider.WithClock(d.now),
	)
	providerCtx, cancel := d.memoryProviderInitContext(ctx, state)
	if cancel != nil {
		defer cancel()
	}
	if err := state.localMemoryProvider.Initialize(providerCtx, memcontract.ProviderInit{
		Logger: state.logger,
		Config: map[string]any{
			"name": localprovider.Name,
		},
	}); err != nil {
		return nil, fmt.Errorf("daemon: initialize local memory provider: %w", err)
	}
	return memory.NewAssembler(
		state.memoryStore,
		memory.WithSnapshotProvider(state.localMemoryProvider),
	), nil
}

func (d *Daemon) memoryProviderInitContext(
	ctx context.Context,
	state *bootState,
) (context.Context, context.CancelFunc) {
	if state.cfg.Memory.Provider.Timeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, state.cfg.Memory.Provider.Timeout)
}

func (d *Daemon) buildSituationContext(state *bootState) *situation.Service {
	return situation.NewService(situation.Deps{
		Now: d.now,
		WorkspaceResolverFunc: func() situation.WorkspaceResolver {
			return state.workspaceResolver
		},
		AgentResolverFunc: func() situation.AgentResolver {
			return agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
				soul:      state.soulCatalog,
				heartbeat: state.heartbeatCatalog,
			})
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
		SoulSnapshotsFunc: func() situation.SoulSnapshotStore {
			return soulSnapshotStoreDependency(state.registry)
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
		workspacepkg.WithChangeHook(func(changeCtx context.Context) error {
			return syncWorkspaceDerivedResources(changeCtx, state)
		}),
	)
	if err != nil {
		return fmt.Errorf("daemon: create workspace resolver: %w", err)
	}
	state.registry = registry
	if state.skillsRegistry != nil {
		state.skillsRegistry.SetEventSummaryStore(registry)
	}
	state.workspaceResolver = workspaceResolver
	if state.harnessRecorder != nil {
		state.harnessRecorder.SetStore(registry)
	}
	memoryProviders, err := newDaemonMemoryProviderRegistry(ctx, state)
	if err != nil {
		return fmt.Errorf("daemon: create memory provider registry: %w", err)
	}
	state.memoryProviderRegistry = memoryProviders
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
	sandboxRegistry, err := d.buildSandboxRegistry(state)
	if err != nil {
		return err
	}
	state.sandboxRegistry = sandboxRegistry
	providerVault, err := d.buildProviderVault(state)
	if err != nil {
		return err
	}
	state.providerVault = providerVault
	if err := d.bootModelCatalog(ctx, state, cleanup); err != nil {
		return err
	}
	state.bridges = d.composeBridgeRuntime(state, cleanup)
	hostedMCP, err := d.buildHostedMCPService(state)
	if err != nil {
		return err
	}
	state.hostedMCP = hostedMCP

	if err := d.bootRuntimeResourceGraph(state); err != nil {
		return err
	}
	return d.bootMemorySessionRuntime(ctx, state)
}

func (d *Daemon) bootRuntimeResourceGraph(state *bootState) error {
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
	state.soulCatalog = newResourceCatalog(cloneSoulResourceSpec)
	state.heartbeatCatalog = newResourceCatalog(cloneHeartbeatResourceSpec)
	return nil
}

func (d *Daemon) bootMemorySessionRuntime(ctx context.Context, state *bootState) error {
	ledgerMaterializer, err := d.newSessionLedgerMaterializer(state)
	if err != nil {
		return err
	}
	state.ledgerMaterializer = ledgerMaterializer

	sessions, err := d.newSessionManager(ctx, d.sessionManagerDeps(state))
	if err != nil {
		return fmt.Errorf("daemon: create session manager: %w", err)
	}
	state.sessions = sessions
	memoryExtractor, err := newDaemonMemoryExtractor(ctx, state, sessions, d.now)
	if err != nil {
		return err
	}
	state.memoryExtractor = memoryExtractor
	state.deps = d.runtimeDeps(ctx, state, sessions)
	resourceService, err := d.buildResourceService(state)
	if err != nil {
		return err
	}
	state.deps.Resources = resourceService
	return nil
}

func (d *Daemon) bootSessionRepair(ctx context.Context, state *bootState) error {
	if state == nil {
		return errors.New("daemon: boot session repair state is required")
	}
	if state.sessions == nil {
		return errors.New("daemon: boot session repair requires session manager")
	}

	infos, err := state.sessions.ListAll(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("daemon: boot session repair canceled: %w", ctxErr)
		}
		state.logger.Warn("daemon: boot session repair skipped session list", "error", err)
		return nil
	}

	for _, info := range infos {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("daemon: boot session repair canceled: %w", err)
		}
		if !bootShouldRepairSession(info) {
			continue
		}

		result, repairErr := state.sessions.RepairSession(ctx, session.RepairOpts{SessionID: info.ID})
		if repairErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("daemon: boot session repair canceled: %w", ctxErr)
			}
			state.logger.Warn(
				"daemon: boot session repair failed",
				"session_id", info.ID,
				"error", repairErr,
			)
			continue
		}
		if result == nil {
			continue
		}
		errorIssues := repairIssueCount(result, session.RepairSeverityError)
		if len(result.Actions) == 0 && errorIssues == 0 {
			continue
		}
		state.logger.Info(
			"daemon: boot session repair complete",
			"session_id", result.SessionID,
			"persisted", result.Persisted,
			"actions", len(result.Actions),
			"issues", len(result.Issues),
			"error_issues", errorIssues,
		)
	}
	if recoverer, ok := state.sessions.(sessionHealthRecoverer); ok {
		result, recoveryErr := recoverer.RecoverSessionHealth(ctx)
		if recoveryErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("daemon: session health recovery canceled: %w", ctxErr)
			}
			state.logger.Warn("daemon: session health recovery failed", "error", recoveryErr)
			return nil
		}
		if result.RefreshedActive > 0 || result.Recomputed > 0 || result.MarkedStale > 0 {
			state.logger.Info(
				"daemon: session health recovery complete",
				"refreshed_active", result.RefreshedActive,
				"recomputed", result.Recomputed,
				"marked_stale", result.MarkedStale,
			)
		}
	}
	return nil
}

type sessionHealthRecoverer interface {
	RecoverSessionHealth(ctx context.Context) (session.HealthRecoveryResult, error)
}

func bootShouldRepairSession(info *session.Info) bool {
	if info == nil || strings.TrimSpace(info.ID) == "" {
		return false
	}
	if info.State != session.StateStopped {
		return false
	}
	switch info.StopReason {
	case store.StopAgentCrashed, store.StopError:
		return true
	default:
		return false
	}
}

func repairIssueCount(result *session.RepairResult, severity string) int {
	if result == nil {
		return 0
	}
	count := 0
	for _, issue := range result.Issues {
		if strings.TrimSpace(issue.Severity) == severity {
			count++
		}
	}
	return count
}

func (d *Daemon) sessionManagerDeps(state *bootState) SessionManagerDeps {
	return SessionManagerDeps{
		HomePaths: d.homePaths,
		Logger:    state.logger,
		Notifier:  d.sessionNotifier(state),
		Hooks: session.HookSet{
			Session:         state.notifier,
			Sandbox:         state.notifier,
			Prompt:          state.notifier,
			Events:          state.notifier,
			Agent:           state.notifier,
			Conversation:    state.notifier,
			Tools:           state.notifier,
			Compaction:      state.notifier,
			Spawn:           state.notifier,
			AuthoredContext: state.notifier,
		},
		PromptAssembler:      state.promptAssembler,
		StartupPromptOverlay: state.startupOverlay,
		PromptInputAugmenter: state.promptAugmenter,
		MemoryStore:          state.memoryStore,
		LedgerMaterializer:   state.ledgerMaterializer,
		AgentResolver: agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		SkillRegistry:       skillRegistryDependency(state.skillsRegistry),
		MCPResolver:         mcpResolverDependency(state.mcpResolver),
		WorkspaceResolver:   state.workspaceResolver,
		SandboxRegistry:     state.sandboxRegistry,
		SessionSupervision:  state.cfg.Session.Supervision,
		SessionHealthConfig: state.cfg.Agents.Heartbeat,
		ProcessRegistry:     state.processRegistry,
		HostedMCP:           hostedMCPLauncher(state.hostedMCP),
		ProviderSecrets:     sessionProviderVaultDependency(state.providerVault),
		SoulStore:           soulSnapshotStoreDependency(state.registry),
		SoulRunChecker:      soulRunActivityCheckerDependency(state.registry),
		SessionHealthStore:  sessionHealthStoreDependency(state.registry),
	}
}

func (d *Daemon) newSessionLedgerMaterializer(state *bootState) (session.LedgerMaterializer, error) {
	if state == nil || !state.cfg.Memory.Enabled {
		return nil, nil
	}
	root := strings.TrimSpace(state.cfg.Memory.Session.LedgerRoot)
	if root == "" {
		root = d.homePaths.SessionsDir
	}
	materializer, err := sessionledger.NewMaterializer(sessionledger.Config{
		RootDir:          root,
		UnboundPartition: state.cfg.Memory.Session.UnboundPartition,
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: create session ledger materializer: %w", err)
	}
	return materializer, nil
}

func (d *Daemon) buildProviderVault(state *bootState) (*vault.Service, error) {
	if state == nil || state.registry == nil {
		return nil, errors.New("daemon: provider vault registry is required")
	}
	vaultStore, ok := state.registry.(vault.Store)
	if !ok {
		if state.logger != nil {
			state.logger.Warn(
				"daemon.provider_vault.disabled",
				"reason",
				"registry_missing_vault_store",
				"registry_type",
				fmt.Sprintf("%T", state.registry),
			)
		}
		return nil, nil
	}
	lookupEnv := func(key string) (string, bool) {
		value := d.getenv(key)
		return value, strings.TrimSpace(value) != ""
	}
	service, err := vault.NewService(
		vaultStore,
		vault.NewFileKeyProvider(d.homePaths.HomeDir, lookupEnv),
		vault.WithLookupEnv(lookupEnv),
		vault.WithNow(d.now),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create provider vault: %w", err)
	}
	return service, nil
}

func sessionProviderVaultDependency(service *vault.Service) session.ProviderSecretResolver {
	if service == nil {
		return nil
	}
	return service
}

func settingsProviderVaultDependency(service *vault.Service) settingspkg.ProviderSecretStore {
	if service == nil {
		return nil
	}
	return service
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

func (d *Daemon) buildSandboxRegistry(state *bootState) (*sandbox.Registry, error) {
	if state == nil {
		return nil, errors.New("daemon: sandbox registry state is required")
	}
	registry, err := local.NewRegistry(
		local.WithLogger(state.logger),
		local.WithProcessRegistry(state.processRegistry),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create sandbox registry: %w", err)
	}
	if err := registry.Register(daytona.NewProvider(
		daytona.WithLogger(state.logger),
		daytona.WithProcessRegistry(state.processRegistry),
	)); err != nil {
		return nil, fmt.Errorf("daemon: register daytona sandbox provider: %w", err)
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

func (d *Daemon) runtimeDeps(ctx context.Context, state *bootState, sessions SessionManager) RuntimeDeps {
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
	authoredContext := authoredContextRuntimeDeps(ctx, state, sessions)
	var memoryProviders core.MemoryProviderService
	if state.memoryProviderRegistry != nil {
		memoryProviders = daemonMemoryProviderService{registry: state.memoryProviderRegistry}
	}

	return RuntimeDeps{
		Config:              state.cfg,
		HomePaths:           d.homePaths,
		Logger:              state.logger,
		Sessions:            sessions,
		Bridges:             state.bridges,
		Registry:            state.registry,
		MemoryStore:         state.memoryStore,
		MemoryExtractor:     state.memoryExtractor,
		MemoryProviders:     memoryProviders,
		MemorySessionLedger: newDaemonMemorySessionLedgerService(state, d.now),
		WorkspaceResolver:   state.workspaceResolver,
		WorkspaceService:    state.workspaceResolver,
		ModelCatalog:        state.modelCatalog,
		AgentCatalog: agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		AgentContext:    state.situationContext,
		SoulAuthoring:   authoredContext.SoulAuthoring,
		SoulRefresher:   authoredContext.SoulRefresher,
		HeartbeatAuthor: authoredContext.HeartbeatAuthoring,
		HeartbeatStatus: authoredContext.HeartbeatStatus,
		HeartbeatWake:   authoredContext.HeartbeatWake,
		SessionHealth:   authoredContext.SessionHealth,
		WakeEvents:      authoredContext.WakeEvents,
		CoordinatorConfig: newCoordinatorConfigResolver(
			&state.cfg,
			state.workspaceResolver,
			agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
				soul:      state.soulCatalog,
				heartbeat: state.heartbeatCatalog,
			}),
		),
		SkillsRegistry: skillsRegistryAPI(state.skillsRegistry),
		ToolRegistry:   state.toolRegistry,
		Toolsets:       state.toolsets,
		ToolApprovals:  state.toolApprovals,
		HostedMCP:      state.hostedMCP,
		DreamTrigger:   dreamTriggerFromRuntime(state.dreamRuntime),
		Vault:          state.providerVault,
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
	if err := registerDaemonResourceCodec(registry, "agent soul", soul.NewResourceCodec); err != nil {
		return err
	}
	if err := registerDaemonResourceCodec(registry, "agent heartbeat", heartbeat.NewResourceCodec); err != nil {
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
	if state.soulCatalog == nil {
		state.soulCatalog = newResourceCatalog(cloneSoulResourceSpec)
	}
	if state.heartbeatCatalog == nil {
		state.heartbeatCatalog = newResourceCatalog(cloneHeartbeatResourceSpec)
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
		SoulCatalog:      state.soulCatalog,
		HeartbeatCatalog: state.heartbeatCatalog,
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
		network.WithManagerHookDispatcher(state.notifier),
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
	state.notifier.setRuntime(hooks, state.observer, state.registry)
	cleanup.add(func(context.Context) error {
		hooks.Close()
		return nil
	})

	if state.skillsRegistry != nil {
		state.skillsCancel, state.skillsDone = startSkillsWatcher(
			ctx,
			state.skillsRegistry,
			state.cfg.Skills.PollInterval,
			workspaceSkillWatcherRoots(d.homePaths, state.registry),
			func(refreshCtx context.Context) error {
				if state.agentSkillResources != nil {
					if err := state.agentSkillResources.Sync(refreshCtx); err != nil {
						return err
					}
				}
				return hookBindings.Sync(refreshCtx)
			},
		)
		cleanup.add(func(cleanupCtx context.Context) error {
			return stopSkillsWatcher(cleanupCtx, state.skillsCancel, state.skillsDone)
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
	return daemonNativeHooks(state.lifecycleObservers, state.dreamRuntime, state.memoryExtractor)
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
		hookspkg.WithExecutorResolver(daemonExecutorResolverWithSecrets(
			nativeExecutors,
			state.providerVault,
			state.processRegistry,
		)),
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
		WebhookSecrets:      state.providerVault,
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
		func(ctx context.Context, name string) (*extensionpkg.Extension, error) {
			return loadExtensionSnapshot(ctx, extRegistry, state.currentExtensionRuntime(), state.logger, name)
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

func syncWorkspaceDerivedResources(ctx context.Context, state *bootState) error {
	return syncExtensionResourcePublishers(ctx, state)
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
		Tasks:                  state.deps.Tasks,
		Network:                state.deps.Network,
		NetworkStore:           state.registry,
		ModelCatalog:           state.modelCatalog,
		MemoryStore:            state.memoryStore,
		MemoryProviderRegistry: state.memoryProviderRegistry,
		Observer:               state.observer,
		SkillsRegistry:         state.skillsRegistry,
		WorkspaceResolver:      state.workspaceResolver,
		Logger:                 state.logger,
		BridgeRegistry:         state.bridges,
		BridgeDedupStore:       bridgeRuntimeDedupStore(state.bridges),
		BridgeBroker:           bridgeRuntimeBroker(state.bridges),
		BridgeRuntime:          state.bridges,
		ResourceStore:          resourceRawStore(state.resourceKernel),
		SourceSessions:         resourceSourceSessions(state.resourceKernel),
		ResourceCodecs:         state.resourceCodecs,
		ResourceTrigger: func(ctx context.Context, kind resources.ResourceKind, reason resources.ReconcileReason) error {
			if state.resourceReconcile == nil {
				return nil
			}
			return state.resourceReconcile.Trigger(ctx, kind, reason)
		},
		SoulAuthoring:   state.deps.SoulAuthoring,
		SoulRefresher:   state.deps.SoulRefresher,
		HeartbeatAuthor: state.deps.HeartbeatAuthor,
		HeartbeatStatus: state.deps.HeartbeatStatus,
		HeartbeatWake:   state.deps.HeartbeatWake,
		SessionHealth:   state.deps.SessionHealth,
		WakeEvents:      state.deps.WakeEvents,
		ProcessRegistry: state.processRegistry,
		SecretResolver:  state.providerVault,
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

func (d *Daemon) bootSettings(ctx context.Context, state *bootState) error {
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
		ProviderSecrets:            settingsProviderVaultDependency(state.providerVault),
		EventSummaries:             state.registry,
		RestartActionAvailable:     true,
		ConsolidateActionAvailable: state.dreamRuntime != nil && state.dreamRuntime.Enabled(),
		LogTailAvailable:           strings.TrimSpace(d.homePaths.LogFile) != "",
	})
	if err != nil {
		return fmt.Errorf("daemon: create settings service: %w", err)
	}

	updateManager, err := newSettingsUpdateManager(d)
	if err != nil {
		return fmt.Errorf("daemon: create settings update manager: %w", err)
	}
	updateManager.PrimeInstallDetection(ctx)

	state.deps.Settings = service
	state.deps.SettingsRestart = settingsRestartController{daemon: d}
	state.deps.SettingsUpdate = settingsUpdateController{manager: updateManager}
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

	d.reconcileDaemonSandboxes(ctx, state)

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
	d.memoryProviderRegistry = state.memoryProviderRegistry
	d.memoryExtractor = state.memoryExtractor
	d.localMemoryProvider = nil
	if state.localMemoryProvider != nil {
		d.localMemoryProvider = state.localMemoryProvider
	}
	d.modelCatalog = state.modelCatalog
	d.situationContext = state.situationContext
	d.sessions = state.sessions
	d.tasks = state.tasks
	d.spawnReaper = state.spawnReaper
	d.scheduler = state.scheduler
	d.network = state.network
	d.toolRegistry = state.toolRegistry
	d.hooks = state.hooks
	d.extensions = state.currentExtensionRuntime()
	d.bridges = state.bridges
	d.observer = state.observer
	d.resourceReconcile = state.resourceReconcile
	d.agentCatalog = state.agentCatalog
	d.soulCatalog = state.soulCatalog
	d.heartbeatCatalog = state.heartbeatCatalog
	d.toolCatalog = state.toolCatalog
	d.mcpServerCatalog = state.mcpServerCatalog
	d.automation = state.automation
	d.httpServer = state.httpServer
	d.udsServer = state.udsServer
	d.dreamRuntime = state.dreamRuntime
	d.workspaceResolver = state.workspaceResolver
	d.sandboxRegistry = state.sandboxRegistry
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

func (d *Daemon) skillsRegistryConfig(cfg *aghconfig.Config) skills.RegistryConfig {
	if cfg == nil {
		return skills.RegistryConfig{
			BundledFS:     skillbundled.FS(),
			UserSkillsDir: d.homePaths.SkillsDir,
		}
	}

	return skills.RegistryConfig{
		BundledFS:      skillbundled.FS(),
		UserSkillsDir:  d.homePaths.SkillsDir,
		UserAgentsDir:  d.homePaths.AgentsDir,
		DisabledSkills: append([]string(nil), cfg.Skills.DisabledSkills...),
	}
}

func startSkillsWatcher(
	ctx context.Context,
	registry *skills.Registry,
	interval time.Duration,
	rootsProvider func(context.Context) ([]string, error),
	afterRefresh func(context.Context) error,
) (context.CancelFunc, chan struct{}) {
	if registry == nil {
		return nil, nil
	}

	watcherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	watcher := skills.NewWatcher(registry, interval)
	watcher.SetRootsProvider(rootsProvider)
	watcher.SetAfterRefresh(afterRefresh)
	go func() {
		defer close(done)
		watcher.Start(watcherCtx)
	}()
	return cancel, done
}

func workspaceSkillWatcherRoots(
	homePaths aghconfig.HomePaths,
	registry Registry,
) func(context.Context) ([]string, error) {
	if registry == nil {
		return nil
	}

	return func(ctx context.Context) ([]string, error) {
		workspaces, err := registry.ListWorkspaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("daemon: list workspaces for skill watcher: %w", err)
		}

		roots := make([]string, 0, len(workspaces)*2)
		for _, workspace := range workspaces {
			for _, root := range aghconfig.WorkspaceDiscoveryRoots(
				workspace.RootDir,
				workspace.AdditionalDirs,
				homePaths,
			) {
				if root.Source == aghconfig.WorkspaceDiscoverySourceGlobal {
					continue
				}
				roots = append(roots, root.SkillsDir(), root.AgentsDir())
			}
		}

		return roots, nil
	}
}

func stopSkillsWatcher(ctx context.Context, cancel context.CancelFunc, done <-chan struct{}) error {
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
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

	resolver := d.bridgeSecretResolver
	if !d.bridgeSecretResolverExplicit && state.providerVault != nil {
		resolver = vaultBridgeSecretResolver{service: state.providerVault}
	}
	runtime := newBridgeRuntime(store, state.logger, d.now, resolver)
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
