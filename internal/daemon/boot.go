package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	aghlogger "github.com/pedronauck/agh/internal/logger"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (d *Daemon) boot(ctx context.Context) (err error) {
	if ctx == nil {
		return errors.New("daemon: boot context is required")
	}

	d.mu.Lock()
	if d.booting || d.lock != nil || d.registry != nil || d.sessions != nil || d.observer != nil {
		d.mu.Unlock()
		return errors.New("daemon: already booted")
	}
	d.booting = true
	d.mu.Unlock()
	defer func() {
		if err == nil {
			return
		}
		d.mu.Lock()
		d.booting = false
		d.mu.Unlock()
	}()

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

	var (
		memoryStore      *memory.Store
		skillsRegistry   *skills.Registry
		mcpResolver      *skills.MCPResolver
		dreamSvc         consolidation.Service
		dreamRuntime     *consolidation.Runtime
		globalMemoryDir  string
		skillsCancel     context.CancelFunc
		skillsDone       chan struct{}
		prependProviders []session.PromptProvider
		appendProviders  []session.PromptProvider
	)
	if cfg.Memory.Enabled {
		globalMemoryDir = strings.TrimSpace(cfg.Memory.GlobalDir)
		if globalMemoryDir == "" {
			globalMemoryDir = d.homePaths.MemoryDir
		}
		memoryStore = memory.NewStore(globalMemoryDir)
		if err := memoryStore.EnsureDirs(); err != nil {
			return fmt.Errorf("daemon: ensure memory store directories: %w", err)
		}
		prependProviders = append(prependProviders, memory.NewAssembler(memoryStore))
	}

	cleanupFns := make([]func(context.Context) error, 0, 8)
	defer func() {
		if err == nil {
			return
		}
		var cleanupErrs []error
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			if cleanupErr := cleanupFns[i](context.Background()); cleanupErr != nil {
				cleanupErrs = append(cleanupErrs, cleanupErr)
			}
		}
		err = errors.Join(err, errors.Join(cleanupErrs...))
	}()
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return closeLogger()
	})

	if cfg.Skills.Enabled {
		skillsCfg, err := d.skillsRegistryConfig(cfg)
		if err != nil {
			return err
		}

		skillsRegistry = skills.NewRegistry(skillsCfg, skills.WithLogger(logger))
		if err := skillsRegistry.LoadAll(ctx); err != nil {
			return fmt.Errorf("daemon: load skills registry: %w", err)
		}
		mcpResolver = skills.NewMCPResolver(cfg.Skills, logger)
		appendProviders = append(appendProviders, skills.NewCatalogProvider(skillsRegistry))
	}

	promptAssembler := NewComposedAssembler(
		WithPrependPromptProviders(prependProviders...),
		WithAppendPromptProviders(appendProviders...),
	)

	pid := d.pid()
	lock, err := d.acquireLock(d.homePaths.DaemonLock, pid)
	if err != nil {
		return err
	}
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return lock.Release()
	})

	stalePID := lock.StalePID()
	if stalePID == 0 {
		existingInfo, readErr := ReadInfo(d.homePaths.DaemonInfo)
		switch {
		case readErr == nil && existingInfo.PID > 0 && existingInfo.PID != pid && !d.processAlive(existingInfo.PID):
			stalePID = existingInfo.PID
		case readErr != nil && !errors.Is(readErr, os.ErrNotExist):
			logger.Warn("daemon: read stale daemon info failed", "path", d.homePaths.DaemonInfo, "error", readErr)
		}
	}
	if stalePID > 0 {
		if cleanupErr := d.cleanupOrphans(ctx, stalePID); cleanupErr != nil {
			logger.Warn("daemon: cleanup orphan processes failed", "stale_pid", stalePID, "error", cleanupErr)
		}
	}

	if err := removeStaleSocket(cfg.Daemon.Socket); err != nil {
		return err
	}

	registry, err := d.openRegistry(ctx, d.homePaths.DatabaseFile)
	if err != nil {
		return fmt.Errorf("daemon: open global database %q: %w", d.homePaths.DatabaseFile, err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return registry.Close(ctx)
	})

	workspaceResolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(d.homePaths),
		workspacepkg.WithLogger(logger),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(d.homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		return fmt.Errorf("daemon: create workspace resolver: %w", err)
	}

	if cfg.Memory.Enabled && cfg.Memory.Dream.Enabled {
		dreamSvc = d.newDreamService(
			memory.WithMemoryStore(memoryStore),
			memory.WithSessionsDir(d.homePaths.SessionsDir),
			memory.WithMinHours(cfg.Memory.Dream.MinHours),
			memory.WithMinSessions(cfg.Memory.Dream.MinSessions),
			memory.WithLogger(logger),
			memory.WithWorkspaceResolver(workspaceResolver),
		)
	}

	startedAt := d.now().UTC()
	notifier := newHooksNotifier(logger, d.now)
	var skillRegistryDep session.SkillRegistry
	if skillsRegistry != nil {
		skillRegistryDep = skillsRegistry
	}
	var mcpResolverDep session.MCPResolver
	if mcpResolver != nil {
		mcpResolverDep = mcpResolver
	}
	sessions, err := d.newSessionManager(ctx, SessionManagerDeps{
		HomePaths:         d.homePaths,
		Logger:            logger,
		Notifier:          notifier,
		Hooks:             notifier,
		PromptAssembler:   promptAssembler,
		SkillRegistry:     skillRegistryDep,
		MCPResolver:       mcpResolverDep,
		WorkspaceResolver: workspaceResolver,
	})
	if err != nil {
		return fmt.Errorf("daemon: create session manager: %w", err)
	}

	dreamSpawner := consolidation.NewSessionSpawner(sessions, workspaceResolver, cfg, globalMemoryDir)
	var dreamTrigger DreamTrigger
	if dreamSvc != nil {
		lockPath := memory.ConsolidationLockPath(globalMemoryDir)
		dreamRuntime = consolidation.NewRuntime(
			cfg.Memory.Dream.Enabled,
			dreamSvc,
			dreamSpawner,
			cfg.Memory.Dream.CheckInterval,
			logger,
			func() (time.Time, error) {
				return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
			},
		)
		dreamTrigger = dreamRuntime
	}

	var skillsRegistryAPI core.SkillsRegistry
	if skillsRegistry != nil {
		skillsRegistryAPI = skillsRegistry
	}

	deps := RuntimeDeps{
		Config:            cfg,
		HomePaths:         d.homePaths,
		Logger:            logger,
		Sessions:          sessions,
		Registry:          registry,
		MemoryStore:       memoryStore,
		WorkspaceResolver: workspaceResolver,
		WorkspaceService:  workspaceResolver,
		SkillsRegistry:    skillsRegistryAPI,
		DreamTrigger:      dreamTrigger,
		StartedAt:         startedAt,
	}

	observer, err := d.newObserver(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create observer: %w", err)
	}
	deps.Observer = observer

	nativeDecls, nativeExecutors := daemonNativeHooks(observer, dreamRuntime)
	hookOptions := []hookspkg.Option{
		hookspkg.WithLogger(logger),
		hookspkg.WithNow(d.now),
		hookspkg.WithDebugPatchAudit(strings.EqualFold(cfg.Log.Level, "debug")),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(nativeExecutors)),
		hookspkg.WithNativeDeclarations(nativeDecls),
		hookspkg.WithConfigDeclarationProvider(configDeclarationProvider(registry, workspaceResolver, logger)),
		hookspkg.WithAgentDeclarationProvider(agentDeclarationProvider(registry, workspaceResolver, logger)),
		hookspkg.WithSkillDeclarationProvider(skillDeclarationProvider(skillsRegistry, registry, workspaceResolver, cfg.Skills.AllowedMarketplaceHooks, logger)),
	}
	if sink, ok := observer.(hookspkg.TelemetrySink); ok {
		hookOptions = append(hookOptions, hookspkg.WithTelemetrySink(sink))
	}
	hooks := hookspkg.NewHooks(hookOptions...)
	if err := hooks.Rebuild(ctx); err != nil {
		hooks.Close()
		return fmt.Errorf("daemon: rebuild hooks: %w", err)
	}
	if hookAwareObserver, ok := observer.(interface {
		AttachHooks(observe.HookCatalogSource)
	}); ok {
		hookAwareObserver.AttachHooks(hooks)
	}
	notifier.setRuntime(hooks, observer)
	cleanupFns = append(cleanupFns, func(context.Context) error {
		hooks.Close()
		return nil
	})

	if skillsRegistry != nil {
		skillsCancel, skillsDone = startSkillsWatcher(ctx, skillsRegistry, cfg.Skills.PollInterval, func(refreshCtx context.Context) error {
			return hooks.Rebuild(refreshCtx)
		})
		cleanupFns = append(cleanupFns, func(context.Context) error {
			stopSkillsWatcher(skillsCancel, skillsDone)
			return nil
		})
	}

	httpServer, err := d.httpFactory(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create http server: %w", err)
	}
	if err := httpServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start http server: %w", err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	})

	udsServer, err := d.udsFactory(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create uds server: %w", err)
	}
	if err := udsServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start uds server: %w", err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return udsServer.Shutdown(ctx)
	})

	info := Info{
		PID:       pid,
		Port:      resolveDaemonPort(cfg.HTTP.Port, httpServer),
		StartedAt: startedAt,
	}
	if err := WriteInfo(d.homePaths.DaemonInfo, info); err != nil {
		return err
	}
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return RemoveInfo(d.homePaths.DaemonInfo)
	})

	reconcileResult, err := observer.Reconcile(ctx)
	if err != nil {
		return fmt.Errorf("daemon: reconcile sessions: %w", err)
	}
	logger.Info(
		"daemon: boot reconciliation complete",
		"indexed_sessions", len(reconcileResult.Indexed),
		"orphaned_sessions", len(reconcileResult.Orphaned),
	)

	if d.shouldVerifyBoundaries() {
		if boundaryErr := d.Boundaries(ctx); boundaryErr != nil {
			logger.Warn("daemon: boundary verification warning", "error", boundaryErr)
		}
	}

	d.mu.Lock()
	d.config = cfg
	d.logger = logger
	d.closeLogger = closeLogger
	d.booting = false
	d.lock = lock
	d.registry = registry
	d.memoryStore = memoryStore
	d.sessions = sessions
	d.hooks = hooks
	d.observer = observer
	d.httpServer = httpServer
	d.udsServer = udsServer
	d.dreamRuntime = dreamRuntime
	d.workspaceResolver = workspaceResolver
	d.skillsRegistry = skillsRegistry
	d.skillsCancel = skillsCancel
	d.skillsDone = skillsDone
	d.startedAt = startedAt
	d.info = info
	if !d.readyClosed {
		close(d.readyCh)
		d.readyClosed = true
	}
	d.mu.Unlock()

	return nil
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
