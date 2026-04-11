package automation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var (
	// ErrManagerNotRunning reports that a runtime-only manager action was called
	// before Start or after Shutdown.
	ErrManagerNotRunning = errors.New("automation: manager not running")
	// ErrDefinitionReadOnly reports that a config-backed definition cannot be
	// mutated through the runtime CRUD surface.
	ErrDefinitionReadOnly = errors.New("automation: definition is config-backed and read-only")
)

const managerRuntimeCleanupTimeout = 2 * time.Second

type managerRuntimeComponent interface {
	Shutdown(ctx context.Context) error
}

// SessionManager is the runtime session surface required by the automation
// manager. It extends the dispatcher path with lookup support for hook-derived
// trigger ingress.
type SessionManager interface {
	SessionCreator
	Status(ctx context.Context, id string) (*session.SessionInfo, error)
}

// Store is the automation persistence surface consumed by the composed
// automation manager.
type Store interface {
	RunStore
	GetRun(ctx context.Context, id string) (Run, error)
	CreateJob(ctx context.Context, job Job) (Job, error)
	UpdateJob(ctx context.Context, job Job) (Job, error)
	DeleteJob(ctx context.Context, id string) error
	GetJob(ctx context.Context, id string) (Job, error)
	ListJobs(ctx context.Context, query JobListQuery) ([]Job, error)
	CreateTrigger(ctx context.Context, trigger Trigger) (Trigger, error)
	UpdateTrigger(ctx context.Context, trigger Trigger) (Trigger, error)
	DeleteTrigger(ctx context.Context, id string) error
	GetTrigger(ctx context.Context, id string) (Trigger, error)
	ListTriggers(ctx context.Context, query TriggerListQuery) ([]Trigger, error)
	ListRuns(ctx context.Context, query RunQuery) ([]Run, error)
	SetJobEnabledOverlay(ctx context.Context, overlay JobEnabledOverlay) (JobEnabledOverlay, error)
	GetJobEnabledOverlay(ctx context.Context, jobID string) (JobEnabledOverlay, error)
	ListJobEnabledOverlays(ctx context.Context) ([]JobEnabledOverlay, error)
	DeleteJobEnabledOverlay(ctx context.Context, jobID string) error
	SetTriggerEnabledOverlay(ctx context.Context, overlay TriggerEnabledOverlay) (TriggerEnabledOverlay, error)
	GetTriggerEnabledOverlay(ctx context.Context, triggerID string) (TriggerEnabledOverlay, error)
	ListTriggerEnabledOverlays(ctx context.Context) ([]TriggerEnabledOverlay, error)
	DeleteTriggerEnabledOverlay(ctx context.Context, triggerID string) error
	SetTriggerWebhookSecret(ctx context.Context, triggerID string, secret string) error
	GetTriggerWebhookSecret(ctx context.Context, triggerID string) (string, error)
	DeleteTriggerWebhookSecret(ctx context.Context, triggerID string) error
}

// WebhookSecretResolver resolves the write-only webhook secret needed to
// register persisted webhook triggers into the runtime engine.
type WebhookSecretResolver interface {
	SecretForTrigger(ctx context.Context, trigger Trigger) (string, error)
}

type storeWebhookSecretResolver struct {
	store interface {
		GetTriggerWebhookSecret(ctx context.Context, triggerID string) (string, error)
	}
}

func (r storeWebhookSecretResolver) SecretForTrigger(ctx context.Context, trigger Trigger) (string, error) {
	if r.store == nil {
		return "", ErrTriggerWebhookSecretNotFound
	}
	return r.store.GetTriggerWebhookSecret(ctx, strings.TrimSpace(trigger.ID))
}

// ResourceStatus reports total and enabled counts for one automation resource
// family.
type ResourceStatus struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

// SyncStats summarizes one TOML synchronization pass.
type SyncStats struct {
	JobsSynced      int       `json:"jobs_synced"`
	TriggersSynced  int       `json:"triggers_synced"`
	JobsRemoved     int       `json:"jobs_removed"`
	TriggersRemoved int       `json:"triggers_removed"`
	SyncedAt        time.Time `json:"synced_at"`
}

// ManagerStatus exposes automation lifecycle, count, and next-fire metadata
// without transport-specific wrappers.
type ManagerStatus struct {
	Running          bool                `json:"running"`
	SchedulerRunning bool                `json:"scheduler_running"`
	Jobs             ResourceStatus      `json:"jobs"`
	Triggers         ResourceStatus      `json:"triggers"`
	ScheduledJobs    []ScheduledJobState `json:"scheduled_jobs,omitempty"`
	NextFire         *time.Time          `json:"next_fire,omitempty"`
	LastSync         SyncStats           `json:"last_sync"`
}

// Option customizes automation manager construction.
type Option func(*managerOptions)

type managerOptions struct {
	store               Store
	sessions            SessionManager
	workspaceResolver   workspacepkg.WorkspaceResolver
	config              aghconfig.AutomationConfig
	logger              *slog.Logger
	globalWorkspacePath string
	webhookSecrets      WebhookSecretResolver
	dispatcherOptions   []DispatcherOption
	schedulerOptions    []SchedulerOption
	triggerOptions      []TriggerEngineOption
	now                 func() time.Time
}

// Manager composes persistence, dispatch, schedules, triggers, and runtime
// status into one daemon-owned automation subsystem.
type Manager struct {
	store               Store
	sessions            SessionManager
	workspaceResolver   workspacepkg.WorkspaceResolver
	config              aghconfig.AutomationConfig
	logger              *slog.Logger
	globalWorkspacePath string
	webhookSecrets      WebhookSecretResolver
	dispatcher          *Dispatcher
	schedulerOptions    []SchedulerOption
	triggerOptions      []TriggerEngineOption
	now                 func() time.Time

	mu            sync.RWMutex
	running       bool
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc
	scheduler     *Scheduler
	triggers      *TriggerEngine
	lastSync      SyncStats
}

// WithStore injects the automation persistence store.
func WithStore(store Store) Option {
	return func(opts *managerOptions) {
		opts.store = store
	}
}

// WithSessions injects the runtime session manager used by the dispatcher and
// hook-derived trigger ingress.
func WithSessions(sessions SessionManager) Option {
	return func(opts *managerOptions) {
		opts.sessions = sessions
	}
}

// WithWorkspaceResolver injects the canonical workspace resolver used to turn
// TOML workspace references into registered workspace IDs.
func WithWorkspaceResolver(resolver workspacepkg.WorkspaceResolver) Option {
	return func(opts *managerOptions) {
		opts.workspaceResolver = resolver
	}
}

// WithConfig injects the loaded automation config.
func WithConfig(cfg aghconfig.AutomationConfig) Option {
	return func(opts *managerOptions) {
		opts.config = cfg
	}
}

// WithLogger injects the subsystem logger.
func WithLogger(logger *slog.Logger) Option {
	return func(opts *managerOptions) {
		opts.logger = logger
	}
}

// WithGlobalWorkspacePath injects the fallback workspace path used for global
// automation sessions.
func WithGlobalWorkspacePath(path string) Option {
	return func(opts *managerOptions) {
		opts.globalWorkspacePath = strings.TrimSpace(path)
	}
}

// WithWebhookSecretResolver injects the write-only secret source used for
// webhook trigger runtime registration.
func WithWebhookSecretResolver(resolver WebhookSecretResolver) Option {
	return func(opts *managerOptions) {
		opts.webhookSecrets = resolver
	}
}

// WithHooks injects the automation lifecycle hook dispatcher used by the shared dispatcher path.
func WithHooks(hooks AutomationHookDispatcher) Option {
	return func(opts *managerOptions) {
		if hooks == nil {
			return
		}
		opts.dispatcherOptions = append(opts.dispatcherOptions, WithDispatcherHooks(hooks))
	}
}

// WithDispatcherOptions appends dispatcher options used when constructing the
// shared dispatcher.
func WithDispatcherOptions(options ...DispatcherOption) Option {
	return func(opts *managerOptions) {
		opts.dispatcherOptions = append(opts.dispatcherOptions, options...)
	}
}

// WithSchedulerOptions appends scheduler options used when constructing the
// runtime scheduler.
func WithSchedulerOptions(options ...SchedulerOption) Option {
	return func(opts *managerOptions) {
		opts.schedulerOptions = append(opts.schedulerOptions, options...)
	}
}

// WithTriggerEngineOptions appends trigger-engine options used when
// constructing the runtime engine.
func WithTriggerEngineOptions(options ...TriggerEngineOption) Option {
	return func(opts *managerOptions) {
		opts.triggerOptions = append(opts.triggerOptions, options...)
	}
}

// WithManagerNow overrides the manager clock used for sync bookkeeping.
func WithManagerNow(now func() time.Time) Option {
	return func(opts *managerOptions) {
		opts.now = now
	}
}

// New constructs the composed automation manager.
func New(opts ...Option) (*Manager, error) {
	options := managerOptions{
		logger: slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		config: aghconfig.AutomationConfig{
			Timezone:          DefaultTimezone,
			MaxConcurrentJobs: DefaultMaxConcurrentJobs,
			DefaultFireLimit:  DefaultFireLimitConfig(),
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	if options.store == nil {
		return nil, errors.New("automation: store is required")
	}
	if options.sessions == nil {
		return nil, errors.New("automation: session manager is required")
	}
	if options.workspaceResolver == nil {
		return nil, errors.New("automation: workspace resolver is required")
	}
	if options.logger == nil {
		options.logger = slog.Default()
	}
	if options.now == nil {
		options.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if strings.TrimSpace(options.config.Timezone) == "" {
		options.config.Timezone = DefaultTimezone
	}
	if options.config.MaxConcurrentJobs <= 0 {
		options.config.MaxConcurrentJobs = DefaultMaxConcurrentJobs
	}
	if options.config.DefaultFireLimit.Max == 0 || strings.TrimSpace(options.config.DefaultFireLimit.Window) == "" {
		options.config.DefaultFireLimit = DefaultFireLimitConfig()
	}
	if options.webhookSecrets == nil {
		options.webhookSecrets = storeWebhookSecretResolver{store: options.store}
	}
	if strings.TrimSpace(options.globalWorkspacePath) == "" {
		return nil, errors.New("automation: global workspace path is required")
	}

	dispatcherOpts := []DispatcherOption{
		WithDispatcherLogger(options.logger),
		WithDispatcherGlobalWorkspacePath(options.globalWorkspacePath),
		WithDispatcherMaxConcurrent(options.config.MaxConcurrentJobs),
	}
	dispatcherOpts = append(dispatcherOpts, options.dispatcherOptions...)
	dispatcher, err := NewDispatcher(options.sessions, options.store, dispatcherOpts...)
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		store:               options.store,
		sessions:            options.sessions,
		workspaceResolver:   options.workspaceResolver,
		config:              options.config,
		logger:              options.logger,
		globalWorkspacePath: options.globalWorkspacePath,
		webhookSecrets:      options.webhookSecrets,
		dispatcher:          dispatcher,
		schedulerOptions:    append([]SchedulerOption(nil), options.schedulerOptions...),
		triggerOptions:      append([]TriggerEngineOption(nil), options.triggerOptions...),
		now:                 options.now,
	}

	return manager, nil
}

// Start synchronizes TOML definitions into persistence, loads effective
// automation state, and starts the runtime scheduler and trigger engine.
func (m *Manager) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: manager start context is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	syncStats, err := m.syncConfigDefinitions(ctx)
	if err != nil {
		return err
	}

	jobs, err := m.loadEffectiveJobs(ctx, JobListQuery{})
	if err != nil {
		return err
	}
	triggers, err := m.loadEffectiveTriggers(ctx, TriggerListQuery{})
	if err != nil {
		return err
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.WithoutCancel(ctx))
	scheduler, triggerEngine, err := m.buildRuntimes(ctx)
	if err != nil {
		runtimeCancel()
		return err
	}

	if err := m.loadSchedulerRegistrations(jobs, scheduler); err != nil {
		return errors.Join(err, m.shutdownStartupRuntime(ctx, runtimeCancel, scheduler, triggerEngine))
	}
	if err := m.loadTriggerRegistrations(ctx, triggers, triggerEngine); err != nil {
		return errors.Join(err, m.shutdownStartupRuntime(ctx, runtimeCancel, scheduler, triggerEngine))
	}
	if err := triggerEngine.Start(ctx); err != nil {
		return errors.Join(err, m.shutdownStartupRuntime(ctx, runtimeCancel, scheduler, triggerEngine))
	}
	if err := scheduler.Start(ctx); err != nil {
		return errors.Join(err, m.shutdownStartupRuntime(ctx, runtimeCancel, scheduler, triggerEngine))
	}

	m.running = true
	m.runtimeCtx = runtimeCtx
	m.runtimeCancel = runtimeCancel
	m.scheduler = scheduler
	m.triggers = triggerEngine
	m.lastSync = syncStats

	m.logger.Info(
		"automation.manager.started",
		"jobs_synced", syncStats.JobsSynced,
		"triggers_synced", syncStats.TriggersSynced,
		"jobs_removed", syncStats.JobsRemoved,
		"triggers_removed", syncStats.TriggersRemoved,
		"jobs_loaded", len(jobs),
		"triggers_loaded", len(triggers),
	)
	return nil
}

// Shutdown stops trigger ingestion, cancels in-flight work, and shuts down the
// runtime scheduler.
func (m *Manager) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: manager shutdown context is required")
	}

	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}

	cancel := m.runtimeCancel
	scheduler := m.scheduler
	triggerEngine := m.triggers
	m.running = false
	m.runtimeCtx = nil
	m.runtimeCancel = nil
	m.scheduler = nil
	m.triggers = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	var errs []error
	if triggerEngine != nil {
		if err := triggerEngine.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if scheduler != nil {
		if err := scheduler.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	m.logger.Info("automation.manager.shutdown")
	return errors.Join(errs...)
}

// Jobs returns overlay-aware job definitions from persistence.
func (m *Manager) Jobs(ctx context.Context) ([]Job, error) {
	return m.ListJobs(ctx, JobListQuery{})
}

// ListJobs returns overlay-aware job definitions using the supplied filters.
func (m *Manager) ListJobs(ctx context.Context, query JobListQuery) ([]Job, error) {
	if ctx == nil {
		return nil, errors.New("automation: list jobs context is required")
	}
	return m.loadEffectiveJobs(ctx, query)
}

// GetJob returns one overlay-aware job definition by id.
func (m *Manager) GetJob(ctx context.Context, id string) (Job, error) {
	if ctx == nil {
		return Job{}, errors.New("automation: get job context is required")
	}
	return m.effectiveJob(ctx, strings.TrimSpace(id))
}

// CreateJob stores a new dynamic automation job and registers it into the
// runtime when the scheduler is active.
func (m *Manager) CreateJob(ctx context.Context, job Job) (Job, error) {
	if ctx == nil {
		return Job{}, errors.New("automation: create job context is required")
	}

	next := cloneJob(job)
	if next.Source == "" {
		next.Source = JobSourceDynamic
	}
	if next.Source != JobSourceDynamic {
		return Job{}, ErrDefinitionReadOnly
	}

	created, err := m.store.CreateJob(ctx, next)
	if err != nil {
		return Job{}, err
	}

	current, err := m.effectiveJobFromStored(ctx, created)
	if err != nil {
		return Job{}, errors.Join(err, m.cleanupCreatedJob(ctx, created.ID))
	}
	if err := m.applyJobToRuntime(current); err != nil {
		return Job{}, errors.Join(err, m.cleanupCreatedJob(ctx, created.ID))
	}

	return current, nil
}

// UpdateJob replaces one existing dynamic automation job definition.
func (m *Manager) UpdateJob(ctx context.Context, job Job) (Job, error) {
	if ctx == nil {
		return Job{}, errors.New("automation: update job context is required")
	}

	currentStored, err := m.store.GetJob(ctx, strings.TrimSpace(job.ID))
	if err != nil {
		return Job{}, err
	}
	if currentStored.Source != JobSourceDynamic {
		return Job{}, ErrDefinitionReadOnly
	}

	previousEffective, err := m.effectiveJobFromStored(ctx, currentStored)
	if err != nil {
		return Job{}, err
	}

	next := cloneJob(job)
	next.ID = currentStored.ID
	next.Source = currentStored.Source
	next.CreatedAt = currentStored.CreatedAt

	updatedStored, err := m.store.UpdateJob(ctx, next)
	if err != nil {
		return Job{}, err
	}

	currentEffective, err := m.effectiveJobFromStored(ctx, updatedStored)
	if err != nil {
		if _, rollbackErr := m.store.UpdateJob(ctx, currentStored); rollbackErr != nil {
			return Job{}, errors.Join(err, fmt.Errorf("automation: restore job %q after load failure: %w", currentStored.ID, rollbackErr))
		}
		return Job{}, err
	}
	if err := m.applyJobToRuntime(currentEffective); err != nil {
		if _, rollbackErr := m.store.UpdateJob(ctx, currentStored); rollbackErr != nil {
			return Job{}, errors.Join(err, rollbackErr)
		}
		if restoreErr := m.applyJobToRuntime(previousEffective); restoreErr != nil {
			return Job{}, errors.Join(err, restoreErr)
		}
		return Job{}, err
	}

	return currentEffective, nil
}

// DeleteJob removes one dynamic automation job definition and unregisters it
// from the runtime scheduler when needed.
func (m *Manager) DeleteJob(ctx context.Context, id string) error {
	if ctx == nil {
		return errors.New("automation: delete job context is required")
	}

	currentStored, err := m.store.GetJob(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if currentStored.Source != JobSourceDynamic {
		return ErrDefinitionReadOnly
	}

	previousEffective, err := m.effectiveJobFromStored(ctx, currentStored)
	if err != nil {
		return err
	}

	scheduler := m.schedulerSnapshot()
	if scheduler != nil {
		if err := scheduler.Unregister(currentStored.ID); err != nil && !errors.Is(err, ErrScheduledJobNotFound) {
			return err
		}
	}

	if err := m.store.DeleteJob(ctx, currentStored.ID); err != nil {
		if restoreErr := m.applyJobToRuntime(previousEffective); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}

	return nil
}

// TriggerJob forces one immediate manual execution through the shared
// dispatcher path.
func (m *Manager) TriggerJob(ctx context.Context, id string) (Run, error) {
	if ctx == nil {
		return Run{}, errors.New("automation: trigger job context is required")
	}

	job, err := m.effectiveJob(ctx, strings.TrimSpace(id))
	if err != nil {
		return Run{}, err
	}

	run, err := m.dispatcher.Dispatch(ctx, DispatchRequest{
		Kind: DispatchKindManual,
		Job:  &job,
	})
	if err != nil {
		if run != nil {
			return *run, err
		}
		return Run{}, err
	}
	if run == nil {
		return Run{}, errors.New("automation: manual job dispatch returned no run")
	}
	return *run, nil
}

// Triggers returns overlay-aware trigger definitions from persistence.
func (m *Manager) Triggers(ctx context.Context) ([]Trigger, error) {
	return m.ListTriggers(ctx, TriggerListQuery{})
}

// ListTriggers returns overlay-aware trigger definitions using the supplied
// filters.
func (m *Manager) ListTriggers(ctx context.Context, query TriggerListQuery) ([]Trigger, error) {
	if ctx == nil {
		return nil, errors.New("automation: list triggers context is required")
	}
	return m.loadEffectiveTriggers(ctx, query)
}

// GetTrigger returns one overlay-aware trigger definition by id.
func (m *Manager) GetTrigger(ctx context.Context, id string) (Trigger, error) {
	if ctx == nil {
		return Trigger{}, errors.New("automation: get trigger context is required")
	}
	return m.effectiveTrigger(ctx, strings.TrimSpace(id))
}

// CreateTrigger stores a new dynamic trigger definition plus its write-only
// webhook secret when applicable, then registers it into the runtime engine.
func (m *Manager) CreateTrigger(ctx context.Context, trigger Trigger, webhookSecret string) (Trigger, error) {
	if ctx == nil {
		return Trigger{}, errors.New("automation: create trigger context is required")
	}

	next := cloneTrigger(trigger)
	if next.Source == "" {
		next.Source = JobSourceDynamic
	}
	if next.Source != JobSourceDynamic {
		return Trigger{}, ErrDefinitionReadOnly
	}

	created, err := m.store.CreateTrigger(ctx, next)
	if err != nil {
		return Trigger{}, err
	}
	created, err = m.ensureTriggerWebhookID(ctx, created)
	if err != nil {
		return Trigger{}, errors.Join(err, m.cleanupCreatedTrigger(ctx, created.ID))
	}
	if err := m.syncTriggerWebhookSecret(ctx, Trigger{}, created, stringPointer(webhookSecret)); err != nil {
		return Trigger{}, errors.Join(err, m.cleanupCreatedTrigger(ctx, created.ID))
	}

	current, err := m.effectiveTriggerFromStored(ctx, created)
	if err != nil {
		return Trigger{}, errors.Join(err, m.cleanupCreatedTrigger(ctx, created.ID))
	}
	if err := m.applyTriggerToRuntime(ctx, current); err != nil {
		return Trigger{}, errors.Join(err, m.cleanupCreatedTrigger(ctx, created.ID))
	}

	return current, nil
}

// UpdateTrigger replaces one existing dynamic trigger definition.
func (m *Manager) UpdateTrigger(ctx context.Context, trigger Trigger, webhookSecret *string) (Trigger, error) {
	if ctx == nil {
		return Trigger{}, errors.New("automation: update trigger context is required")
	}

	currentStored, err := m.store.GetTrigger(ctx, strings.TrimSpace(trigger.ID))
	if err != nil {
		return Trigger{}, err
	}
	if currentStored.Source != JobSourceDynamic {
		return Trigger{}, ErrDefinitionReadOnly
	}

	previousEffective, err := m.effectiveTriggerFromStored(ctx, currentStored)
	if err != nil {
		return Trigger{}, err
	}
	previousSecret, err := m.currentWebhookSecret(ctx, currentStored)
	if err != nil {
		return Trigger{}, err
	}

	next := cloneTrigger(trigger)
	next.ID = currentStored.ID
	next.Source = currentStored.Source
	next.CreatedAt = currentStored.CreatedAt

	updatedStored, err := m.store.UpdateTrigger(ctx, next)
	if err != nil {
		return Trigger{}, err
	}
	updatedStored, err = m.ensureTriggerWebhookID(ctx, updatedStored)
	if err != nil {
		if _, rollbackErr := m.store.UpdateTrigger(ctx, currentStored); rollbackErr != nil {
			return Trigger{}, errors.Join(err, rollbackErr)
		}
		return Trigger{}, err
	}
	if err := m.syncTriggerWebhookSecret(ctx, currentStored, updatedStored, webhookSecret); err != nil {
		if _, rollbackErr := m.store.UpdateTrigger(ctx, currentStored); rollbackErr != nil {
			return Trigger{}, errors.Join(err, rollbackErr)
		}
		if restoreErr := m.restoreWebhookSecret(ctx, currentStored, previousSecret); restoreErr != nil {
			return Trigger{}, errors.Join(err, restoreErr)
		}
		return Trigger{}, err
	}

	currentEffective, err := m.effectiveTriggerFromStored(ctx, updatedStored)
	if err != nil {
		if _, rollbackErr := m.store.UpdateTrigger(ctx, currentStored); rollbackErr != nil {
			return Trigger{}, errors.Join(err, rollbackErr)
		}
		if restoreErr := m.restoreWebhookSecret(ctx, currentStored, previousSecret); restoreErr != nil {
			return Trigger{}, errors.Join(err, restoreErr)
		}
		return Trigger{}, err
	}
	if err := m.applyTriggerToRuntime(ctx, currentEffective); err != nil {
		if _, rollbackErr := m.store.UpdateTrigger(ctx, currentStored); rollbackErr != nil {
			return Trigger{}, errors.Join(err, rollbackErr)
		}
		if restoreErr := m.restoreWebhookSecret(ctx, currentStored, previousSecret); restoreErr != nil {
			return Trigger{}, errors.Join(err, restoreErr)
		}
		if runtimeErr := m.applyTriggerToRuntime(ctx, previousEffective); runtimeErr != nil {
			return Trigger{}, errors.Join(err, runtimeErr)
		}
		return Trigger{}, err
	}

	return currentEffective, nil
}

// DeleteTrigger removes one dynamic trigger definition and clears any
// persisted webhook secret.
func (m *Manager) DeleteTrigger(ctx context.Context, id string) error {
	if ctx == nil {
		return errors.New("automation: delete trigger context is required")
	}

	currentStored, err := m.store.GetTrigger(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if currentStored.Source != JobSourceDynamic {
		return ErrDefinitionReadOnly
	}

	previousEffective, err := m.effectiveTriggerFromStored(ctx, currentStored)
	if err != nil {
		return err
	}
	previousSecret, err := m.currentWebhookSecret(ctx, currentStored)
	if err != nil {
		return err
	}

	engine := m.triggerEngineSnapshot()
	if engine != nil {
		if err := engine.Unregister(currentStored.ID); err != nil && !errors.Is(err, ErrTriggerNotFound) {
			return err
		}
	}

	if err := m.store.DeleteTriggerWebhookSecret(ctx, currentStored.ID); err != nil {
		return err
	}
	if err := m.store.DeleteTrigger(ctx, currentStored.ID); err != nil {
		if restoreErr := m.restoreWebhookSecret(ctx, currentStored, previousSecret); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		if runtimeErr := m.applyTriggerToRuntime(ctx, previousEffective); runtimeErr != nil {
			return errors.Join(err, runtimeErr)
		}
		return err
	}

	return nil
}

// Runs returns persisted automation run history.
func (m *Manager) Runs(ctx context.Context, query RunQuery) ([]Run, error) {
	return m.ListRuns(ctx, query)
}

// ListRuns returns persisted automation run history using the supplied
// filters.
func (m *Manager) ListRuns(ctx context.Context, query RunQuery) ([]Run, error) {
	if ctx == nil {
		return nil, errors.New("automation: list runs context is required")
	}
	return m.store.ListRuns(ctx, query)
}

// GetRun returns one persisted automation run by id.
func (m *Manager) GetRun(ctx context.Context, id string) (Run, error) {
	if ctx == nil {
		return Run{}, errors.New("automation: get run context is required")
	}
	return m.store.GetRun(ctx, strings.TrimSpace(id))
}

// Status returns aggregate automation lifecycle and next-fire metadata.
func (m *Manager) Status(ctx context.Context) (ManagerStatus, error) {
	if ctx == nil {
		return ManagerStatus{}, errors.New("automation: status context is required")
	}

	jobs, err := m.loadEffectiveJobs(ctx, JobListQuery{})
	if err != nil {
		return ManagerStatus{}, err
	}
	triggers, err := m.loadEffectiveTriggers(ctx, TriggerListQuery{})
	if err != nil {
		return ManagerStatus{}, err
	}

	status := ManagerStatus{
		Running:  m.isRunning(),
		LastSync: m.lastSyncSnapshot(),
		Jobs: ResourceStatus{
			Total:   len(jobs),
			Enabled: countEnabledJobs(jobs),
		},
		Triggers: ResourceStatus{
			Total:   len(triggers),
			Enabled: countEnabledTriggers(triggers),
		},
	}

	scheduler := m.schedulerSnapshot()
	if scheduler != nil {
		status.SchedulerRunning = status.Running
		status.ScheduledJobs = scheduler.States()
		if nextFire, ok := earliestNextFire(status.ScheduledJobs); ok {
			status.NextFire = &nextFire
		}
	}

	return status, nil
}

// SetJobEnabled updates the effective enabled state for one job. Config-backed
// jobs use overlay rows while dynamic jobs mutate their persisted definition.
func (m *Manager) SetJobEnabled(ctx context.Context, id string, enabled bool) (Job, error) {
	if ctx == nil {
		return Job{}, errors.New("automation: set job enabled context is required")
	}

	stored, err := m.store.GetJob(ctx, strings.TrimSpace(id))
	if err != nil {
		return Job{}, err
	}
	previous, err := m.effectiveJobFromStored(ctx, stored)
	if err != nil {
		return Job{}, err
	}

	switch stored.Source {
	case JobSourceConfig:
		if err := m.persistJobOverlay(ctx, stored, enabled); err != nil {
			return Job{}, err
		}
	default:
		stored.Enabled = enabled
		if _, err := m.store.UpdateJob(ctx, stored); err != nil {
			return Job{}, err
		}
	}

	current, err := m.effectiveJob(ctx, stored.ID)
	if err != nil {
		return Job{}, err
	}
	if err := m.applyJobToRuntime(current); err != nil {
		if rollbackErr := m.rollbackJobEnabled(ctx, stored, previous.Enabled); rollbackErr != nil {
			return Job{}, errors.Join(err, rollbackErr)
		}
		return Job{}, err
	}
	return current, nil
}

// SetTriggerEnabled updates the effective enabled state for one trigger.
// Config-backed triggers use overlay rows while dynamic triggers mutate their
// persisted definition.
func (m *Manager) SetTriggerEnabled(ctx context.Context, id string, enabled bool) (Trigger, error) {
	if ctx == nil {
		return Trigger{}, errors.New("automation: set trigger enabled context is required")
	}

	stored, err := m.store.GetTrigger(ctx, strings.TrimSpace(id))
	if err != nil {
		return Trigger{}, err
	}
	previous, err := m.effectiveTriggerFromStored(ctx, stored)
	if err != nil {
		return Trigger{}, err
	}

	switch stored.Source {
	case JobSourceConfig:
		if err := m.persistTriggerOverlay(ctx, stored, enabled); err != nil {
			return Trigger{}, err
		}
	default:
		stored.Enabled = enabled
		if _, err := m.store.UpdateTrigger(ctx, stored); err != nil {
			return Trigger{}, err
		}
	}

	current, err := m.effectiveTrigger(ctx, stored.ID)
	if err != nil {
		return Trigger{}, err
	}
	if err := m.applyTriggerToRuntime(ctx, current); err != nil {
		if rollbackErr := m.rollbackTriggerEnabled(ctx, stored, previous.Enabled); rollbackErr != nil {
			return Trigger{}, errors.Join(err, rollbackErr)
		}
		return Trigger{}, err
	}
	return current, nil
}

// HandleWebhook validates, normalizes, and dispatches a webhook delivery
// through the running trigger engine.
func (m *Manager) HandleWebhook(ctx context.Context, request WebhookRequest) (TriggerResult, error) {
	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return TriggerResult{}, ErrManagerNotRunning
	}
	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()
	return engine.HandleWebhook(mergedCtx, request)
}

// FireExtensionTrigger routes one extension-originated ext.* event through the shared trigger engine.
func (m *Manager) FireExtensionTrigger(ctx context.Context, request ExtensionTriggerRequest) (TriggerResult, error) {
	if err := request.Validate("extension_trigger"); err != nil {
		return TriggerResult{}, err
	}

	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return TriggerResult{}, ErrManagerNotRunning
	}

	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()

	envelope := ActivationEnvelope{
		Kind:        strings.TrimSpace(request.Event),
		Scope:       request.Scope,
		WorkspaceID: strings.TrimSpace(request.WorkspaceID),
		Source:      ActivationSourceExtension,
		Data:        cloneJSONMap(request.Payload),
	}
	if envelope.Data == nil {
		envelope.Data = map[string]any{}
	}
	return engine.Fire(mergedCtx, envelope)
}

// SessionObserver exposes the existing session notifier seam for automation
// trigger ingress.
func (m *Manager) SessionObserver() session.Notifier {
	return managerSessionObserver{manager: m}
}

// HookTelemetrySink exposes the existing hook telemetry sink seam for
// hook-completion trigger ingress.
func (m *Manager) HookTelemetrySink() hookspkg.TelemetrySink {
	return managerHookTelemetrySink{manager: m}
}

// MemoryObserver exposes the automation memory-consolidation observer seam for
// callers that can publish completion events.
func (m *Manager) MemoryObserver() MemoryConsolidationObserver {
	return managerMemoryObserver{manager: m}
}

func (m *Manager) loadEffectiveJobs(ctx context.Context, query JobListQuery) ([]Job, error) {
	jobs, err := m.store.ListJobs(ctx, query)
	if err != nil {
		return nil, err
	}
	overlays, err := m.store.ListJobEnabledOverlays(ctx)
	if err != nil {
		return nil, err
	}

	overlayByID := make(map[string]bool, len(overlays))
	for _, overlay := range overlays {
		overlayByID[overlay.JobID] = overlay.EnabledOverride
	}

	effective := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		next := cloneJob(job)
		if next.Source == JobSourceConfig {
			if enabled, ok := overlayByID[next.ID]; ok {
				next.Enabled = enabled
			}
		}
		effective = append(effective, next)
	}
	sortJobs(effective)
	return effective, nil
}

func (m *Manager) loadEffectiveTriggers(ctx context.Context, query TriggerListQuery) ([]Trigger, error) {
	triggers, err := m.store.ListTriggers(ctx, query)
	if err != nil {
		return nil, err
	}
	overlays, err := m.store.ListTriggerEnabledOverlays(ctx)
	if err != nil {
		return nil, err
	}

	overlayByID := make(map[string]bool, len(overlays))
	for _, overlay := range overlays {
		overlayByID[overlay.TriggerID] = overlay.EnabledOverride
	}

	effective := make([]Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		next := cloneTrigger(trigger)
		if next.Source == JobSourceConfig {
			if enabled, ok := overlayByID[next.ID]; ok {
				next.Enabled = enabled
			}
		}
		effective = append(effective, next)
	}
	sortTriggers(effective)
	return effective, nil
}

func (m *Manager) effectiveJob(ctx context.Context, id string) (Job, error) {
	job, err := m.store.GetJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	return m.effectiveJobFromStored(ctx, job)
}

func (m *Manager) effectiveJobFromStored(ctx context.Context, job Job) (Job, error) {
	effective := cloneJob(job)
	if effective.Source != JobSourceConfig {
		return effective, nil
	}

	overlay, err := m.store.GetJobEnabledOverlay(ctx, effective.ID)
	switch {
	case err == nil:
		effective.Enabled = overlay.EnabledOverride
	case errors.Is(err, ErrJobOverlayNotFound):
	default:
		return Job{}, err
	}

	return effective, nil
}

func (m *Manager) effectiveTrigger(ctx context.Context, id string) (Trigger, error) {
	trigger, err := m.store.GetTrigger(ctx, id)
	if err != nil {
		return Trigger{}, err
	}
	return m.effectiveTriggerFromStored(ctx, trigger)
}

func (m *Manager) effectiveTriggerFromStored(ctx context.Context, trigger Trigger) (Trigger, error) {
	effective := cloneTrigger(trigger)
	if effective.Source != JobSourceConfig {
		return effective, nil
	}

	overlay, err := m.store.GetTriggerEnabledOverlay(ctx, effective.ID)
	switch {
	case err == nil:
		effective.Enabled = overlay.EnabledOverride
	case errors.Is(err, ErrTriggerOverlayNotFound):
	default:
		return Trigger{}, err
	}

	return effective, nil
}

func (m *Manager) buildRuntimes(ctx context.Context) (*Scheduler, *TriggerEngine, error) {
	location, err := time.LoadLocation(strings.TrimSpace(m.config.Timezone))
	if err != nil {
		return nil, nil, fmt.Errorf("automation: load manager timezone %q: %w", m.config.Timezone, err)
	}

	schedulerOpts := []SchedulerOption{
		WithSchedulerLogger(m.logger),
		WithSchedulerLocation(location),
	}
	schedulerOpts = append(schedulerOpts, m.schedulerOptions...)
	scheduler, err := NewScheduler(m.dispatcher, schedulerOpts...)
	if err != nil {
		return nil, nil, err
	}

	triggerOpts := []TriggerEngineOption{
		WithTriggerEngineLogger(m.logger),
		WithTriggerEngineHookSessionResolver(m.sessions),
	}
	triggerOpts = append(triggerOpts, m.triggerOptions...)
	triggerEngine, err := NewTriggerEngine(m.dispatcher, triggerOpts...)
	if err != nil {
		return nil, nil, errors.Join(err, m.shutdownRuntimeComponent(ctx, "scheduler", scheduler))
	}

	return scheduler, triggerEngine, nil
}

func (m *Manager) loadSchedulerRegistrations(jobs []Job, scheduler *Scheduler) error {
	for _, job := range jobs {
		if _, err := scheduler.Register(job); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) loadTriggerRegistrations(ctx context.Context, triggers []Trigger, engine *TriggerEngine) error {
	for _, trigger := range triggers {
		registration, shouldRegister, err := m.runtimeTriggerRegistration(ctx, trigger)
		if err != nil {
			return err
		}
		if !shouldRegister {
			if strings.EqualFold(trigger.Event, "webhook") && trigger.Enabled {
				m.logger.Warn(
					"automation.trigger.skipped_webhook_registration",
					"trigger_id", trigger.ID,
					"trigger_name", trigger.Name,
				)
			}
			continue
		}
		if err := engine.Register(registration); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) syncConfigDefinitions(ctx context.Context) (SyncStats, error) {
	desiredJobs := make([]Job, 0, len(m.config.Jobs))
	for idx, raw := range m.config.Jobs {
		job, err := m.resolveConfigJob(ctx, raw)
		if err != nil {
			return SyncStats{}, fmt.Errorf("automation: resolve config job %d: %w", idx, err)
		}
		desiredJobs = append(desiredJobs, job)
	}
	sortJobs(desiredJobs)

	desiredTriggers := make([]Trigger, 0, len(m.config.Triggers))
	for idx, raw := range m.config.Triggers {
		trigger, err := m.resolveConfigTrigger(ctx, raw)
		if err != nil {
			return SyncStats{}, fmt.Errorf("automation: resolve config trigger %d: %w", idx, err)
		}
		desiredTriggers = append(desiredTriggers, trigger)
	}
	sortTriggers(desiredTriggers)

	jobsSynced, jobsRemoved, err := m.syncJobs(ctx, desiredJobs)
	if err != nil {
		return SyncStats{}, err
	}
	triggersSynced, triggersRemoved, err := m.syncTriggers(ctx, desiredTriggers)
	if err != nil {
		return SyncStats{}, err
	}

	stats := SyncStats{
		JobsSynced:      jobsSynced,
		TriggersSynced:  triggersSynced,
		JobsRemoved:     jobsRemoved,
		TriggersRemoved: triggersRemoved,
		SyncedAt:        m.now().UTC(),
	}

	m.logger.Info(
		"automation.config.sync",
		"jobs_synced", stats.JobsSynced,
		"triggers_synced", stats.TriggersSynced,
		"jobs_removed", stats.JobsRemoved,
		"triggers_removed", stats.TriggersRemoved,
	)
	return stats, nil
}

func (m *Manager) syncJobs(ctx context.Context, desired []Job) (int, int, error) {
	existing, err := m.store.ListJobs(ctx, JobListQuery{Source: JobSourceConfig})
	if err != nil {
		return 0, 0, err
	}

	existingByID := make(map[string]Job, len(existing))
	for _, job := range existing {
		existingByID[job.ID] = job
	}

	desiredByID := make(map[string]Job, len(desired))
	synced := 0
	for _, job := range desired {
		desiredByID[job.ID] = job
		current, exists := existingByID[job.ID]
		switch {
		case !exists:
			if _, err := m.store.CreateJob(ctx, job); err != nil {
				return 0, 0, err
			}
		case current.Source != JobSourceConfig:
			return 0, 0, fmt.Errorf("automation: config job %q conflicts with non-config source", job.ID)
		case !sameJobDefinition(current, job):
			if _, err := m.store.UpdateJob(ctx, job); err != nil {
				return 0, 0, err
			}
		}
		synced++
	}

	removed := 0
	for id := range existingByID {
		if _, ok := desiredByID[id]; ok {
			continue
		}
		if err := m.store.DeleteJob(ctx, id); err != nil {
			return 0, 0, err
		}
		removed++
	}

	return synced, removed, nil
}

func (m *Manager) syncTriggers(ctx context.Context, desired []Trigger) (int, int, error) {
	existing, err := m.store.ListTriggers(ctx, TriggerListQuery{Source: JobSourceConfig})
	if err != nil {
		return 0, 0, err
	}

	existingByID := make(map[string]Trigger, len(existing))
	for _, trigger := range existing {
		existingByID[trigger.ID] = trigger
	}

	desiredByID := make(map[string]Trigger, len(desired))
	synced := 0
	for _, trigger := range desired {
		desiredByID[trigger.ID] = trigger
		current, exists := existingByID[trigger.ID]
		switch {
		case !exists:
			if _, err := m.store.CreateTrigger(ctx, trigger); err != nil {
				return 0, 0, err
			}
		case current.Source != JobSourceConfig:
			return 0, 0, fmt.Errorf("automation: config trigger %q conflicts with non-config source", trigger.ID)
		case !sameTriggerDefinition(current, trigger):
			if _, err := m.store.UpdateTrigger(ctx, trigger); err != nil {
				return 0, 0, err
			}
		}
		synced++
	}

	removed := 0
	for id := range existingByID {
		if _, ok := desiredByID[id]; ok {
			continue
		}
		if err := m.store.DeleteTrigger(ctx, id); err != nil {
			return 0, 0, err
		}
		removed++
	}

	return synced, removed, nil
}

func (m *Manager) resolveConfigJob(ctx context.Context, raw aghconfig.AutomationJob) (Job, error) {
	workspaceID, err := m.resolveConfigWorkspace(ctx, raw.Scope, raw.Workspace)
	if err != nil {
		return Job{}, err
	}

	fireLimit := raw.FireLimit
	if fireLimit.Max == 0 || strings.TrimSpace(fireLimit.Window) == "" {
		fireLimit = m.config.DefaultFireLimit
	}
	if fireLimit.Max == 0 || strings.TrimSpace(fireLimit.Window) == "" {
		fireLimit = DefaultFireLimitConfig()
	}

	retry := raw.Retry
	if retry.Strategy == "" {
		retry = DefaultRetryConfig()
	}

	schedule := raw.Schedule
	job := Job{
		ID:          configJobID(raw.Scope, workspaceID, raw.Name),
		Scope:       raw.Scope,
		Name:        strings.TrimSpace(raw.Name),
		AgentName:   strings.TrimSpace(raw.AgentName),
		WorkspaceID: workspaceID,
		Prompt:      strings.TrimSpace(raw.Prompt),
		Schedule:    &schedule,
		Enabled:     raw.Enabled,
		Retry:       retry,
		FireLimit:   fireLimit,
		Source:      JobSourceConfig,
	}
	if err := job.Validate("job"); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (m *Manager) resolveConfigTrigger(ctx context.Context, raw aghconfig.AutomationTrigger) (Trigger, error) {
	workspaceID, err := m.resolveConfigWorkspace(ctx, raw.Scope, raw.Workspace)
	if err != nil {
		return Trigger{}, err
	}

	fireLimit := raw.FireLimit
	if fireLimit.Max == 0 || strings.TrimSpace(fireLimit.Window) == "" {
		fireLimit = m.config.DefaultFireLimit
	}
	if fireLimit.Max == 0 || strings.TrimSpace(fireLimit.Window) == "" {
		fireLimit = DefaultFireLimitConfig()
	}

	retry := raw.Retry
	if retry.Strategy == "" {
		retry = DefaultRetryConfig()
	}

	trigger := Trigger{
		ID:           configTriggerID(raw.Scope, workspaceID, raw.Name),
		Scope:        raw.Scope,
		Name:         strings.TrimSpace(raw.Name),
		AgentName:    strings.TrimSpace(raw.AgentName),
		WorkspaceID:  workspaceID,
		Prompt:       strings.TrimSpace(raw.Prompt),
		Event:        strings.TrimSpace(raw.Event),
		Filter:       cloneFilter(raw.Filter),
		Enabled:      raw.Enabled,
		Retry:        retry,
		FireLimit:    fireLimit,
		Source:       JobSourceConfig,
		EndpointSlug: strings.TrimSpace(raw.EndpointSlug),
	}
	if strings.EqualFold(trigger.Event, "webhook") {
		trigger.WebhookID = configWebhookID(raw.Scope, workspaceID, raw.Name)
	}
	if err := trigger.Validate("trigger"); err != nil {
		return Trigger{}, err
	}
	return trigger, nil
}

func (m *Manager) resolveConfigWorkspace(ctx context.Context, scope AutomationScope, workspaceRef string) (string, error) {
	if scope == AutomationScopeGlobal {
		return "", nil
	}

	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return "", errors.New("automation: workspace reference is required")
	}

	var (
		resolved workspacepkg.ResolvedWorkspace
		err      error
	)
	if isPathLikeWorkspaceRef(trimmedRef) {
		normalizedPath, normalizeErr := aghconfig.ResolvePath(trimmedRef)
		if normalizeErr != nil {
			return "", fmt.Errorf("automation: resolve config workspace %q: %w", trimmedRef, normalizeErr)
		}
		resolved, err = m.workspaceResolver.ResolveOrRegister(ctx, normalizedPath)
	} else {
		resolved, err = m.workspaceResolver.Resolve(ctx, trimmedRef)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resolved.ID) == "" {
		return "", errors.New("automation: resolved workspace id is required")
	}
	return strings.TrimSpace(resolved.ID), nil
}

func (m *Manager) persistJobOverlay(ctx context.Context, definition Job, enabled bool) error {
	if definition.Source != JobSourceConfig {
		return ErrOverlayRequiresConfigSource
	}
	if enabled == definition.Enabled {
		return m.store.DeleteJobEnabledOverlay(ctx, definition.ID)
	}
	_, err := m.store.SetJobEnabledOverlay(ctx, JobEnabledOverlay{
		JobID:           definition.ID,
		EnabledOverride: enabled,
	})
	return err
}

func (m *Manager) persistTriggerOverlay(ctx context.Context, definition Trigger, enabled bool) error {
	if definition.Source != JobSourceConfig {
		return ErrOverlayRequiresConfigSource
	}
	if enabled == definition.Enabled {
		return m.store.DeleteTriggerEnabledOverlay(ctx, definition.ID)
	}
	_, err := m.store.SetTriggerEnabledOverlay(ctx, TriggerEnabledOverlay{
		TriggerID:       definition.ID,
		EnabledOverride: enabled,
	})
	return err
}

func (m *Manager) rollbackJobEnabled(ctx context.Context, definition Job, enabled bool) error {
	switch definition.Source {
	case JobSourceConfig:
		return m.persistJobOverlay(ctx, definition, enabled)
	default:
		definition.Enabled = enabled
		_, err := m.store.UpdateJob(ctx, definition)
		return err
	}
}

func (m *Manager) rollbackTriggerEnabled(ctx context.Context, definition Trigger, enabled bool) error {
	switch definition.Source {
	case JobSourceConfig:
		return m.persistTriggerOverlay(ctx, definition, enabled)
	default:
		definition.Enabled = enabled
		_, err := m.store.UpdateTrigger(ctx, definition)
		return err
	}
}

func (m *Manager) syncTriggerWebhookSecret(ctx context.Context, previous Trigger, current Trigger, webhookSecret *string) error {
	if !strings.EqualFold(strings.TrimSpace(current.Event), "webhook") {
		return m.store.DeleteTriggerWebhookSecret(ctx, current.ID)
	}

	secret, err := m.desiredWebhookSecret(ctx, previous, current, webhookSecret)
	if err != nil {
		return err
	}
	if strings.TrimSpace(secret) == "" {
		return ErrWebhookSecretRequired
	}
	return m.store.SetTriggerWebhookSecret(ctx, current.ID, secret)
}

func (m *Manager) ensureTriggerWebhookID(ctx context.Context, trigger Trigger) (Trigger, error) {
	if !strings.EqualFold(strings.TrimSpace(trigger.Event), "webhook") || strings.TrimSpace(trigger.WebhookID) != "" {
		return trigger, nil
	}

	next := cloneTrigger(trigger)
	next.WebhookID = stableConfigID("wbh", next.ID)
	return m.store.UpdateTrigger(ctx, next)
}

func (m *Manager) desiredWebhookSecret(ctx context.Context, previous Trigger, current Trigger, webhookSecret *string) (string, error) {
	if webhookSecret != nil {
		return strings.TrimSpace(*webhookSecret), nil
	}

	if strings.EqualFold(strings.TrimSpace(previous.Event), "webhook") && strings.TrimSpace(previous.ID) != "" {
		secret, err := m.store.GetTriggerWebhookSecret(ctx, previous.ID)
		switch {
		case err == nil:
			return strings.TrimSpace(secret), nil
		case errors.Is(err, ErrTriggerWebhookSecretNotFound):
			return "", nil
		default:
			return "", err
		}
	}

	if strings.TrimSpace(current.ID) != "" {
		secret, err := m.store.GetTriggerWebhookSecret(ctx, current.ID)
		switch {
		case err == nil:
			return strings.TrimSpace(secret), nil
		case errors.Is(err, ErrTriggerWebhookSecretNotFound):
			return "", nil
		default:
			return "", err
		}
	}

	return "", nil
}

func (m *Manager) currentWebhookSecret(ctx context.Context, trigger Trigger) (string, error) {
	if !strings.EqualFold(strings.TrimSpace(trigger.Event), "webhook") || strings.TrimSpace(trigger.ID) == "" {
		return "", nil
	}

	secret, err := m.store.GetTriggerWebhookSecret(ctx, trigger.ID)
	switch {
	case err == nil:
		return strings.TrimSpace(secret), nil
	case errors.Is(err, ErrTriggerWebhookSecretNotFound):
		return "", nil
	default:
		return "", err
	}
}

func (m *Manager) restoreWebhookSecret(ctx context.Context, trigger Trigger, secret string) error {
	if !strings.EqualFold(strings.TrimSpace(trigger.Event), "webhook") {
		return m.store.DeleteTriggerWebhookSecret(ctx, trigger.ID)
	}
	if strings.TrimSpace(secret) == "" {
		return m.store.DeleteTriggerWebhookSecret(ctx, trigger.ID)
	}
	return m.store.SetTriggerWebhookSecret(ctx, trigger.ID, secret)
}

func (m *Manager) applyJobToRuntime(job Job) error {
	scheduler := m.schedulerSnapshot()
	if scheduler == nil {
		return nil
	}

	_, err := scheduler.State(job.ID)
	switch {
	case err == nil:
		_, err = scheduler.Update(job)
		return err
	case errors.Is(err, ErrScheduledJobNotFound):
		if !job.Enabled {
			return nil
		}
		_, err = scheduler.Register(job)
		return err
	default:
		return err
	}
}

func (m *Manager) applyTriggerToRuntime(ctx context.Context, trigger Trigger) error {
	engine := m.triggerEngineSnapshot()
	if engine == nil {
		return nil
	}

	registration, shouldRegister, err := m.runtimeTriggerRegistration(ctx, trigger)
	if err != nil {
		return err
	}
	if !shouldRegister {
		if err := engine.Unregister(trigger.ID); err != nil && !errors.Is(err, ErrTriggerNotFound) {
			return err
		}
		return nil
	}

	if err := engine.Update(registration); err != nil {
		if errors.Is(err, ErrTriggerNotFound) {
			return engine.Register(registration)
		}
		return err
	}
	return nil
}

func (m *Manager) runtimeTriggerRegistration(ctx context.Context, trigger Trigger) (TriggerRegistration, bool, error) {
	if !trigger.Enabled {
		return TriggerRegistration{}, false, nil
	}

	registration := TriggerRegistration{Trigger: cloneTrigger(trigger)}
	if !strings.EqualFold(trigger.Event, "webhook") {
		return registration, true, nil
	}
	if m.webhookSecrets == nil {
		return TriggerRegistration{}, false, nil
	}

	secret, err := m.webhookSecrets.SecretForTrigger(ctx, trigger)
	if err != nil {
		if errors.Is(err, ErrTriggerWebhookSecretNotFound) {
			return TriggerRegistration{}, false, nil
		}
		return TriggerRegistration{}, false, err
	}
	if strings.TrimSpace(secret) == "" {
		return TriggerRegistration{}, false, nil
	}

	registration.WebhookSecret = strings.TrimSpace(secret)
	return registration, true, nil
}

func (m *Manager) isRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) schedulerSnapshot() *Scheduler {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scheduler
}

func (m *Manager) triggerEngineSnapshot() *TriggerEngine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.triggers
}

func (m *Manager) lastSyncSnapshot() SyncStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSync
}

func (m *Manager) triggerRuntime() (*TriggerEngine, context.Context, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.running || m.triggers == nil || m.runtimeCtx == nil {
		return nil, nil, false
	}
	return m.triggers, m.runtimeCtx, true
}

func (m *Manager) fireSessionCreated(ctx context.Context, sess *session.Session) {
	if sess == nil {
		return
	}
	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return
	}
	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()
	if _, err := engine.FireSessionCreated(mergedCtx, sess); err != nil {
		m.logger.Warn("automation.manager.session_created_trigger_failed", "session_id", strings.TrimSpace(sess.ID), "error", err)
	}
}

func (m *Manager) fireSessionStopped(ctx context.Context, sess *session.Session) {
	if sess == nil {
		return
	}
	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return
	}
	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()
	if _, err := engine.FireSessionStopped(mergedCtx, sess); err != nil {
		m.logger.Warn("automation.manager.session_stopped_trigger_failed", "session_id", strings.TrimSpace(sess.ID), "error", err)
	}
}

func (m *Manager) cleanupCreatedJob(ctx context.Context, jobID string) error {
	var errs []error
	if err := m.store.DeleteJob(ctx, jobID); err != nil {
		errs = append(errs, fmt.Errorf("automation: delete created job %q: %w", jobID, err))
	}
	if scheduler := m.schedulerSnapshot(); scheduler != nil {
		if err := scheduler.Unregister(jobID); err != nil && !errors.Is(err, ErrScheduledJobNotFound) {
			errs = append(errs, fmt.Errorf("automation: unregister created job %q: %w", jobID, err))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) cleanupCreatedTrigger(ctx context.Context, triggerID string) error {
	var errs []error
	if err := m.store.DeleteTriggerWebhookSecret(ctx, triggerID); err != nil {
		errs = append(errs, fmt.Errorf("automation: delete created trigger webhook secret %q: %w", triggerID, err))
	}
	if err := m.store.DeleteTrigger(ctx, triggerID); err != nil {
		errs = append(errs, fmt.Errorf("automation: delete created trigger %q: %w", triggerID, err))
	}
	if engine := m.triggerEngineSnapshot(); engine != nil {
		if err := engine.Unregister(triggerID); err != nil && !errors.Is(err, ErrTriggerNotFound) {
			errs = append(errs, fmt.Errorf("automation: unregister created trigger %q: %w", triggerID, err))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) shutdownStartupRuntime(ctx context.Context, runtimeCancel context.CancelFunc, scheduler *Scheduler, triggerEngine *TriggerEngine) error {
	if runtimeCancel != nil {
		runtimeCancel()
	}

	var errs []error
	if err := m.shutdownRuntimeComponent(ctx, "trigger engine", triggerEngine); err != nil {
		errs = append(errs, err)
	}
	if err := m.shutdownRuntimeComponent(ctx, "scheduler", scheduler); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (m *Manager) shutdownRuntimeComponent(ctx context.Context, name string, component managerRuntimeComponent) error {
	if component == nil {
		return nil
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), managerRuntimeCleanupTimeout)
	defer cancel()

	if err := component.Shutdown(cleanupCtx); err != nil {
		return fmt.Errorf("automation: shutdown %s: %w", name, err)
	}
	return nil
}

func (m *Manager) fireHookRecord(ctx context.Context, sessionID string, record hookspkg.HookRunRecord) error {
	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return nil
	}
	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()
	_, err := engine.FireHookCompletion(mergedCtx, sessionID, record)
	return err
}

func (m *Manager) fireMemoryConsolidated(ctx context.Context, event MemoryConsolidatedEvent) error {
	engine, runtimeCtx, ok := m.triggerRuntime()
	if !ok {
		return nil
	}
	mergedCtx, cancel := mergedRuntimeContext(ctx, runtimeCtx)
	defer cancel()
	_, err := engine.FireMemoryConsolidated(mergedCtx, event)
	return err
}

func mergedRuntimeContext(parent context.Context, runtimeCtx context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if runtimeCtx == nil {
		return parent, func() {}
	}

	mergedCtx, cancel := context.WithCancel(context.WithoutCancel(parent))
	stopParent := context.AfterFunc(parent, cancel)
	stopRuntime := context.AfterFunc(runtimeCtx, cancel)

	if err := parent.Err(); err != nil {
		cancel()
	}
	if err := runtimeCtx.Err(); err != nil {
		cancel()
	}

	return mergedCtx, func() {
		stopParent()
		stopRuntime()
		cancel()
	}
}

func earliestNextFire(states []ScheduledJobState) (time.Time, bool) {
	var (
		next time.Time
		set  bool
	)
	for _, state := range states {
		if state.NextRun == nil || state.NextRun.IsZero() {
			continue
		}
		if !set || state.NextRun.Before(next) {
			next = *state.NextRun
			set = true
		}
	}
	return next, set
}

func countEnabledJobs(jobs []Job) int {
	count := 0
	for _, job := range jobs {
		if job.Enabled {
			count++
		}
	}
	return count
}

func countEnabledTriggers(triggers []Trigger) int {
	count := 0
	for _, trigger := range triggers {
		if trigger.Enabled {
			count++
		}
	}
	return count
}

func configJobID(scope AutomationScope, workspaceID string, name string) string {
	return stableConfigID("jobcfg", string(scope), workspaceID, name)
}

func configTriggerID(scope AutomationScope, workspaceID string, name string) string {
	return stableConfigID("trgcfg", string(scope), workspaceID, name)
}

func configWebhookID(scope AutomationScope, workspaceID string, name string) string {
	return stableConfigID("wbh", string(scope), workspaceID, name)
}

func stableConfigID(prefix string, parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.TrimSpace(part))
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\n")))
	if prefix == "wbh" {
		return "wbh_" + hex.EncodeToString(sum[:8])
	}
	return prefix + "_" + hex.EncodeToString(sum[:8])
}

func isPathLikeWorkspaceRef(ref string) bool {
	trimmedRef := strings.TrimSpace(ref)
	return filepath.IsAbs(trimmedRef) ||
		strings.HasPrefix(trimmedRef, ".") ||
		strings.HasPrefix(trimmedRef, "~") ||
		strings.ContainsAny(trimmedRef, `/\`)
}

func sameJobDefinition(left Job, right Job) bool {
	return left.ID == right.ID &&
		left.Scope == right.Scope &&
		left.Name == right.Name &&
		left.AgentName == right.AgentName &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Prompt == right.Prompt &&
		sameSchedule(left.Schedule, right.Schedule) &&
		left.Enabled == right.Enabled &&
		left.Retry == right.Retry &&
		left.FireLimit == right.FireLimit &&
		left.Source == right.Source
}

func sameTriggerDefinition(left Trigger, right Trigger) bool {
	return left.ID == right.ID &&
		left.Scope == right.Scope &&
		left.Name == right.Name &&
		left.AgentName == right.AgentName &&
		left.WorkspaceID == right.WorkspaceID &&
		left.Prompt == right.Prompt &&
		left.Event == right.Event &&
		sameFilter(left.Filter, right.Filter) &&
		left.Enabled == right.Enabled &&
		left.Retry == right.Retry &&
		left.FireLimit == right.FireLimit &&
		left.Source == right.Source &&
		left.WebhookID == right.WebhookID &&
		left.EndpointSlug == right.EndpointSlug
}

func sameSchedule(left *ScheduleSpec, right *ScheduleSpec) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func sameFilter(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func sortJobs(jobs []Job) {
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Name == jobs[j].Name {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].Name < jobs[j].Name
	})
}

func sortTriggers(triggers []Trigger) {
	sort.Slice(triggers, func(i, j int) bool {
		if triggers[i].Name == triggers[j].Name {
			return triggers[i].ID < triggers[j].ID
		}
		return triggers[i].Name < triggers[j].Name
	})
}

func cloneJob(job Job) Job {
	cloned := job
	if job.Schedule != nil {
		schedule := *job.Schedule
		cloned.Schedule = &schedule
	}
	return cloned
}

func cloneTrigger(trigger Trigger) Trigger {
	cloned := trigger
	cloned.Filter = cloneFilter(trigger.Filter)
	return cloned
}

func cloneFilter(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func stringPointer(value string) *string {
	return &value
}

type managerSessionObserver struct {
	manager *Manager
}

func (o managerSessionObserver) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if o.manager != nil {
		o.manager.fireSessionCreated(ctx, sess)
	}
}

func (o managerSessionObserver) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if o.manager != nil {
		o.manager.fireSessionStopped(ctx, sess)
	}
}

func (managerSessionObserver) OnAgentEvent(context.Context, string, any) {
}

type managerHookTelemetrySink struct {
	manager *Manager
}

func (s managerHookTelemetrySink) WriteHookRecord(ctx context.Context, sessionID string, record hookspkg.HookRunRecord) error {
	if s.manager == nil {
		return nil
	}
	return s.manager.fireHookRecord(ctx, sessionID, record)
}

type managerMemoryObserver struct {
	manager *Manager
}

func (o managerMemoryObserver) OnMemoryConsolidated(ctx context.Context, event MemoryConsolidatedEvent) error {
	if o.manager == nil {
		return nil
	}
	return o.manager.fireMemoryConsolidated(ctx, event)
}
