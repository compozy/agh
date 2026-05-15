package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/coordinator"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

type coordinatorTaskStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []taskpkg.RunStatus) ([]taskpkg.Run, error)
}

type coordinatorSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
}

type coordinatorHookDispatcher interface {
	DispatchCoordinatorPreSpawn(
		context.Context,
		hookspkg.CoordinatorPreSpawnPayload,
	) (hookspkg.CoordinatorPreSpawnPayload, error)
	DispatchCoordinatorSpawned(
		context.Context,
		hookspkg.CoordinatorSpawnedPayload,
	) (hookspkg.CoordinatorSpawnedPayload, error)
	DispatchCoordinatorDecision(
		context.Context,
		hookspkg.CoordinatorDecisionPayload,
	) (hookspkg.CoordinatorDecisionPayload, error)
	DispatchCoordinatorStopped(
		context.Context,
		hookspkg.CoordinatorStoppedPayload,
	) (hookspkg.CoordinatorStoppedPayload, error)
	DispatchCoordinatorFailed(
		context.Context,
		hookspkg.CoordinatorFailedPayload,
	) (hookspkg.CoordinatorFailedPayload, error)
}

type coordinatorRuntime struct {
	mu             sync.Mutex
	store          coordinatorTaskStore
	sessions       coordinatorSessionManager
	config         CoordinatorConfigResolver
	hooks          coordinatorHookDispatcher
	contextOverlay taskSessionContextOverlay
	logger         *slog.Logger
	now            func() time.Time
}

var _ taskRunEnqueuedObserver = (*coordinatorRuntime)(nil)
var _ sessionLifecycleObserver = (*coordinatorRuntime)(nil)

type coordinatorRuntimeOption func(*coordinatorRuntime)

func withCoordinatorTaskContextOverlay(overlay taskSessionContextOverlay) coordinatorRuntimeOption {
	return func(runtime *coordinatorRuntime) {
		if runtime != nil {
			runtime.contextOverlay = overlay
		}
	}
}

func newCoordinatorRuntime(
	store coordinatorTaskStore,
	sessions coordinatorSessionManager,
	config CoordinatorConfigResolver,
	hooks coordinatorHookDispatcher,
	logger *slog.Logger,
	now func() time.Time,
	options ...coordinatorRuntimeOption,
) (*coordinatorRuntime, error) {
	if store == nil {
		return nil, errors.New("daemon: coordinator runtime requires task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: coordinator runtime requires session manager")
	}
	if config == nil {
		return nil, errors.New("daemon: coordinator runtime requires config resolver")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	runtime := &coordinatorRuntime{
		store:    store,
		sessions: sessions,
		config:   config,
		hooks:    hooks,
		logger:   logger,
		now:      now,
	}
	for _, option := range options {
		if option != nil {
			option(runtime)
		}
	}
	return runtime, nil
}

func (d *Daemon) bootCoordinator(ctx context.Context, state *bootState, _ *bootCleanup) error {
	if state == nil || state.tasks == nil || state.tasks.store == nil {
		return nil
	}
	if state.sessions == nil {
		return errors.New("daemon: coordinator runtime requires session manager")
	}
	if state.deps.CoordinatorConfig == nil {
		return errors.New("daemon: coordinator runtime requires coordinator config resolver")
	}

	runtime, err := newCoordinatorRuntime(
		state.tasks.store,
		state.sessions,
		state.deps.CoordinatorConfig,
		state.notifier,
		state.logger,
		d.now,
		withCoordinatorTaskContextOverlay(state.situationContext),
	)
	if err != nil {
		return err
	}
	router, err := newReviewRouter(
		state.tasks.manager,
		state.tasks.store,
		state.sessions,
		state.workspaceResolver,
		agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
			soul:      state.soulCatalog,
			heartbeat: state.heartbeatCatalog,
		}),
		state.logger,
		d.now,
		withReviewRouterTaskContextOverlay(state.situationContext),
	)
	if err != nil {
		return err
	}
	if state.notifier != nil {
		state.notifier.AddTaskRunEnqueuedObserver(runtime)
	}
	if state.lifecycleObservers != nil {
		state.lifecycleObservers.Add(runtime)
		state.lifecycleObservers.Add(router)
	}
	if state.reviewRequests != nil {
		state.reviewRequests.Set(router)
	}
	runtime.Recover(ctx)
	state.coordinator = runtime
	return nil
}

func (r *coordinatorRuntime) OnTaskRunEnqueued(ctx context.Context, payload hookspkg.TaskRunEnqueuedPayload) {
	if ctx == nil {
		ctx = context.Background()
	}
	runID := strings.TrimSpace(payload.RunID)
	if runID == "" {
		r.logCoordinatorError("daemon: coordinator enqueue payload missing run id", nil, payload)
		return
	}
	run, err := r.store.GetTaskRun(ctx, runID)
	if err != nil {
		r.logCoordinatorError("daemon: load task run for coordinator enqueue", err, payload)
		return
	}
	taskRecord, err := r.store.GetTask(ctx, run.TaskID)
	if err != nil {
		r.logCoordinatorError("daemon: load task for coordinator enqueue", err, payload)
		return
	}
	if _, _, err := r.bootstrapRun(ctx, taskRecord, run, coordinator.ReasonRunEnqueued); err != nil {
		r.logCoordinatorError("daemon: bootstrap coordinator from enqueue", err, payload)
	}
}

func (r *coordinatorRuntime) OnSessionCreated(context.Context, *session.Session) {
}

func (r *coordinatorRuntime) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if r == nil || sess == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	info := sess.Info()
	if info == nil || info.Type != session.SessionTypeCoordinator {
		return
	}
	r.dispatchStopped(ctx, info)
	r.recoverWorkspace(ctx, strings.TrimSpace(info.WorkspaceID), coordinator.ReasonCoordinatorStopped)
}

func (r *coordinatorRuntime) Recover(ctx context.Context) {
	r.recoverWorkspace(ctx, "", coordinator.ReasonRecovery)
}

func (r *coordinatorRuntime) recoverWorkspace(ctx context.Context, workspaceID string, reason string) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runs, err := r.store.ListTaskRunsByStatus(ctx, coordinator.ExecutableRunStatuses())
	if err != nil {
		r.logCoordinatorError(
			"daemon: list executable runs for coordinator recovery",
			err,
			hookspkg.TaskRunEnqueuedPayload{},
		)
		return
	}

	workspaceID = strings.TrimSpace(workspaceID)
	for _, run := range runs {
		taskRecord, err := r.store.GetTask(ctx, run.TaskID)
		if err != nil {
			r.logCoordinatorError("daemon: load task for coordinator recovery", err, hookspkg.TaskRunEnqueuedPayload{
				TaskRunContext: hookspkg.TaskRunContext{RunID: run.ID, TaskID: run.TaskID},
			})
			continue
		}
		if workspaceID != "" && strings.TrimSpace(taskRecord.WorkspaceID) != workspaceID {
			continue
		}
		if _, _, err := r.bootstrapRun(ctx, taskRecord, run, reason); err != nil {
			r.logCoordinatorError(
				"daemon: recover coordinator for executable run",
				err,
				hookspkg.TaskRunEnqueuedPayload{
					TaskRunContext: hookspkg.TaskRunContext{
						RunID:                 run.ID,
						TaskID:                run.TaskID,
						WorkspaceID:           taskRecord.WorkspaceID,
						CoordinationChannelID: run.CoordinationChannelID,
					},
				},
			)
		}
	}
}

func (r *coordinatorRuntime) bootstrapRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
) (*session.Info, bool, error) {
	if ctx == nil {
		return nil, false, errors.New("daemon: coordinator bootstrap context is required")
	}

	preflightConfig := defaultEnabledCoordinatorConfig()
	preflight := coordinator.DecideBootstrap(taskRecord, run, preflightConfig)
	if !preflight.ShouldBootstrap {
		r.dispatchDecision(ctx, preflight, reason, "")
		return nil, false, nil
	}

	cfg, err := r.config.ResolveCoordinatorConfig(ctx, preflight.WorkspaceID)
	if err != nil {
		r.dispatchFailed(ctx, preflight, reason, err)
		return nil, false, fmt.Errorf("daemon: resolve coordinator config: %w", err)
	}
	decision := coordinator.DecideBootstrap(taskRecord, run, cfg)
	if !decision.ShouldBootstrap {
		r.dispatchDecision(ctx, decision, reason, "")
		return nil, false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.activeCoordinator(ctx, decision.WorkspaceID)
	if err != nil {
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, false, err
	}
	if existing != nil {
		r.dispatchDecision(ctx, decision, reason, coordinator.DecisionExisting)
		return existing, false, nil
	}

	info, created, err := r.createCoordinatorSession(ctx, decision, cfg, reason)
	if err != nil {
		return nil, false, err
	}
	return info, created, nil
}

func (r *coordinatorRuntime) createCoordinatorSession(
	ctx context.Context,
	decision coordinator.Decision,
	cfg aghconfig.CoordinatorConfig,
	reason string,
) (*session.Info, bool, error) {
	preSpawn := r.preSpawnPayload(decision, cfg, reason)
	preSpawn, err := r.dispatchPreSpawn(ctx, preSpawn)
	if err != nil {
		if preSpawn.Denied {
			r.dispatchDecision(ctx, decision, reason, coordinator.DecisionDenied)
			return nil, false, nil
		}
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, false, err
	}
	if preSpawn.Denied {
		r.dispatchDecision(ctx, decision, reason, coordinator.DecisionDenied)
		return nil, false, nil
	}

	cfg.AgentName = firstNonEmpty(preSpawn.AgentName, cfg.AgentName)
	cfg.Provider = firstNonEmpty(preSpawn.Provider, cfg.Provider)
	cfg.Model = firstNonEmpty(preSpawn.Model, cfg.Model)
	info, err := r.startCoordinatorSession(ctx, decision, cfg)
	if err != nil {
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, false, err
	}
	r.dispatchSpawned(ctx, decision, info, cfg, reason)
	return info, true, nil
}

func (r *coordinatorRuntime) startCoordinatorSession(
	ctx context.Context,
	decision coordinator.Decision,
	cfg aghconfig.CoordinatorConfig,
) (*session.Info, error) {
	policy := coordinator.PermissionPolicy(decision.CoordinationChannelID)
	now := r.now().UTC()
	promptOverlay := coordinator.PromptOverlay(coordinator.PromptInput{
		WorkspaceID:           decision.WorkspaceID,
		TaskID:                decision.TaskID,
		RunID:                 decision.RunID,
		WorkflowID:            decision.WorkflowID,
		CoordinationChannelID: decision.CoordinationChannelID,
	})
	if r.contextOverlay != nil &&
		strings.TrimSpace(decision.TaskID) != "" &&
		strings.TrimSpace(decision.RunID) != "" {
		taskRecord, err := r.store.GetTask(ctx, decision.TaskID)
		if err != nil {
			return nil, fmt.Errorf("daemon: load coordinator task context task %q: %w", decision.TaskID, err)
		}
		run, err := r.store.GetTaskRun(ctx, decision.RunID)
		if err != nil {
			return nil, fmt.Errorf("daemon: load coordinator task context run %q: %w", decision.RunID, err)
		}
		taskOverlay, err := r.contextOverlay.TaskRunPromptOverlay(ctx, taskRecord, run, nil)
		if err != nil {
			return nil, fmt.Errorf("daemon: render coordinator task context overlay: %w", err)
		}
		promptOverlay = joinPromptOverlays(taskOverlay, promptOverlay)
	}
	created, err := r.sessions.Create(ctx, session.CreateOpts{
		AgentName:     cfg.AgentName,
		Provider:      cfg.Provider,
		Model:         cfg.Model,
		Name:          coordinatorSessionName(decision.WorkspaceID),
		Workspace:     decision.WorkspaceID,
		Channel:       decision.CoordinationChannelID,
		PromptOverlay: promptOverlay,
		Type:          session.SessionTypeCoordinator,
		Lineage:       coordinator.Lineage(now, cfg, policy),
	})
	if err != nil {
		return nil, fmt.Errorf("daemon: create coordinator session: %w", err)
	}
	if created == nil {
		return nil, errors.New("daemon: coordinator session create returned nil")
	}
	info := created.Info()
	if info == nil {
		return nil, errors.New("daemon: coordinator session create returned nil info")
	}
	return info, nil
}

func (r *coordinatorRuntime) activeCoordinator(ctx context.Context, workspaceID string) (*session.Info, error) {
	infos, err := r.sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list sessions for coordinator singleton: %w", err)
	}
	now := r.now().UTC()
	for _, info := range infos {
		if coordinator.HealthySession(info, workspaceID, now) {
			return info, nil
		}
	}
	return nil, nil
}

func (r *coordinatorRuntime) dispatchPreSpawn(
	ctx context.Context,
	payload hookspkg.CoordinatorPreSpawnPayload,
) (hookspkg.CoordinatorPreSpawnPayload, error) {
	if r.hooks == nil {
		return payload, nil
	}
	result, err := r.hooks.DispatchCoordinatorPreSpawn(ctx, payload)
	if err != nil {
		return result, fmt.Errorf("daemon: dispatch coordinator pre-spawn hook: %w", err)
	}
	return result, nil
}

func (r *coordinatorRuntime) dispatchSpawned(
	ctx context.Context,
	decision coordinator.Decision,
	info *session.Info,
	cfg aghconfig.CoordinatorConfig,
	reason string,
) {
	if r.hooks == nil || info == nil {
		return
	}
	_, err := r.hooks.DispatchCoordinatorSpawned(ctx, hookspkg.CoordinatorSpawnedPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookCoordinatorSpawned, Timestamp: r.now().UTC()},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:           decision.WorkspaceID,
			Workspace:             info.Workspace,
			AgentName:             info.AgentName,
			CoordinatorSessionID:  info.ID,
			TaskID:                decision.TaskID,
			RunID:                 decision.RunID,
			WorkflowID:            decision.WorkflowID,
			CoordinationChannelID: decision.CoordinationChannelID,
			Provider:              cfg.Provider,
			Model:                 cfg.Model,
		},
		DecisionKind: "lifecycle",
		Decision:     reason,
	})
	if err != nil {
		r.logger.Warn("daemon: dispatch coordinator spawned hook failed", "error", err)
	}
}

func (r *coordinatorRuntime) dispatchStopped(ctx context.Context, info *session.Info) {
	if r.hooks == nil || info == nil {
		return
	}
	_, err := r.hooks.DispatchCoordinatorStopped(ctx, hookspkg.CoordinatorStoppedPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookCoordinatorStopped, Timestamp: r.now().UTC()},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:          info.WorkspaceID,
			Workspace:            info.Workspace,
			AgentName:            info.AgentName,
			CoordinatorSessionID: info.ID,
			Provider:             info.Provider,
		},
		DecisionKind: "lifecycle",
		Decision:     coordinator.ReasonCoordinatorStopped,
		StopReason:   string(info.StopReason),
	})
	if err != nil {
		r.logger.Warn("daemon: dispatch coordinator stopped hook failed", "error", err)
	}
}

func (r *coordinatorRuntime) dispatchFailed(
	ctx context.Context,
	decision coordinator.Decision,
	reason string,
	failed error,
) {
	if r.hooks == nil || failed == nil {
		return
	}
	_, err := r.hooks.DispatchCoordinatorFailed(ctx, hookspkg.CoordinatorFailedPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookCoordinatorFailed, Timestamp: r.now().UTC()},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:           decision.WorkspaceID,
			TaskID:                decision.TaskID,
			RunID:                 decision.RunID,
			WorkflowID:            decision.WorkflowID,
			CoordinationChannelID: decision.CoordinationChannelID,
		},
		DecisionKind: "bootstrap",
		Decision:     reason,
		Error:        failed.Error(),
	})
	if err != nil {
		r.logger.Warn("daemon: dispatch coordinator failed hook failed", "error", err)
	}
}

func (r *coordinatorRuntime) dispatchDecision(
	ctx context.Context,
	decision coordinator.Decision,
	reason string,
	override string,
) {
	if r.hooks == nil {
		return
	}
	value := decision.Reason
	if strings.TrimSpace(override) != "" {
		value = strings.TrimSpace(override)
	}
	_, err := r.hooks.DispatchCoordinatorDecision(ctx, hookspkg.CoordinatorDecisionPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookCoordinatorDecision, Timestamp: r.now().UTC()},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:           decision.WorkspaceID,
			TaskID:                decision.TaskID,
			RunID:                 decision.RunID,
			WorkflowID:            decision.WorkflowID,
			CoordinationChannelID: decision.CoordinationChannelID,
		},
		DecisionKind: "bootstrap",
		Decision:     firstNonEmpty(value, reason),
	})
	if err != nil {
		r.logger.Warn("daemon: dispatch coordinator decision hook failed", "error", err)
	}
}

func (r *coordinatorRuntime) preSpawnPayload(
	decision coordinator.Decision,
	cfg aghconfig.CoordinatorConfig,
	reason string,
) hookspkg.CoordinatorPreSpawnPayload {
	return hookspkg.CoordinatorPreSpawnPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookCoordinatorPreSpawn, Timestamp: r.now().UTC()},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:           decision.WorkspaceID,
			AgentName:             cfg.AgentName,
			TaskID:                decision.TaskID,
			RunID:                 decision.RunID,
			WorkflowID:            decision.WorkflowID,
			CoordinationChannelID: decision.CoordinationChannelID,
			Provider:              cfg.Provider,
			Model:                 cfg.Model,
		},
		Reason: reason,
	}
}

func (r *coordinatorRuntime) logCoordinatorError(
	message string,
	err error,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil {
		return
	}
	args := []any{
		"task_id", strings.TrimSpace(payload.TaskID),
		"run_id", strings.TrimSpace(payload.RunID),
		"workspace_id", strings.TrimSpace(payload.WorkspaceID),
		"coordination_channel_id", strings.TrimSpace(payload.CoordinationChannelID),
	}
	if err != nil {
		args = append(args, "error", err)
	}
	r.logger.Warn(message, args...)
}

func defaultEnabledCoordinatorConfig() aghconfig.CoordinatorConfig {
	cfg := aghconfig.DefaultCoordinatorConfig()
	cfg.Enabled = true
	return cfg
}

func coordinatorSessionName(workspaceID string) string {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return "AGH Coordinator"
	}
	return "AGH Coordinator " + trimmed
}
