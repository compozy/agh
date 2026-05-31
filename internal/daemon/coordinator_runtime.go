package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/coordinator"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/session"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	coordinatorRuntimeTaskIDKey      = "task_id"
	coordinatorRuntimeWorkspaceIDKey = "workspace_id"
)

type coordinatorTaskStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []taskpkg.RunStatus) ([]taskpkg.Run, error)
}

type coordinatorSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
	PromptSynthetic(ctx context.Context, id string, opts session.SyntheticPromptOpts) (<-chan acp.AgentEvent, error)
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
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	store          coordinatorTaskStore
	sessions       coordinatorSessionManager
	config         CoordinatorConfigResolver
	hooks          coordinatorHookDispatcher
	contextOverlay taskSessionContextOverlay
	logger         *slog.Logger
	now            func() time.Time
	wakeInFlight   map[string]struct{}
	wg             sync.WaitGroup
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
	ctx context.Context,
	store coordinatorTaskStore,
	sessions coordinatorSessionManager,
	config CoordinatorConfigResolver,
	hooks coordinatorHookDispatcher,
	logger *slog.Logger,
	now func() time.Time,
	options ...coordinatorRuntimeOption,
) (*coordinatorRuntime, error) {
	if ctx == nil {
		return nil, errors.New("daemon: coordinator runtime context is required")
	}
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
	lifecycleCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	runtime := &coordinatorRuntime{
		ctx:          lifecycleCtx,
		cancel:       cancel,
		store:        store,
		sessions:     sessions,
		config:       config,
		hooks:        hooks,
		logger:       logger,
		now:          now,
		wakeInFlight: make(map[string]struct{}),
	}
	for _, option := range options {
		if option != nil {
			option(runtime)
		}
	}
	return runtime, nil
}

func (d *Daemon) bootCoordinator(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
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
		ctx,
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
	if cleanup != nil {
		cleanup.add(func(cleanupCtx context.Context) error {
			return runtime.shutdown(cleanupCtx)
		})
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
	existing, err := r.activeCoordinator(ctx, decision.WorkspaceID)
	if err != nil {
		r.mu.Unlock()
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, false, err
	}
	if existing != nil {
		shouldPrompt := r.beginCoordinatorWakeLocked(existing, decision)
		r.mu.Unlock()
		r.dispatchDecision(ctx, decision, reason, coordinator.DecisionExisting)
		if shouldPrompt {
			if err := r.promptCoordinator(ctx, existing, decision, reason); err != nil {
				r.finishCoordinatorWake(existing, decision)
				r.dispatchFailed(ctx, decision, reason, err)
				return existing, false, err
			}
			r.finishCoordinatorWake(existing, decision)
		}
		return existing, false, nil
	}

	info, createdCfg, created, err := r.createCoordinatorSession(ctx, decision, cfg, reason)
	if err != nil {
		r.mu.Unlock()
		return nil, false, err
	}
	if !created {
		r.mu.Unlock()
		return nil, false, nil
	}
	shouldPrompt := r.beginCoordinatorWakeLocked(info, decision)
	r.mu.Unlock()
	if shouldPrompt {
		if err := r.promptCoordinator(ctx, info, decision, reason); err != nil {
			r.finishCoordinatorWake(info, decision)
			r.dispatchFailed(ctx, decision, reason, err)
			return nil, false, err
		}
		r.finishCoordinatorWake(info, decision)
	}
	r.dispatchSpawned(ctx, decision, info, createdCfg, reason)
	return info, created, nil
}

func (r *coordinatorRuntime) createCoordinatorSession(
	ctx context.Context,
	decision coordinator.Decision,
	cfg aghconfig.CoordinatorConfig,
	reason string,
) (*session.Info, aghconfig.CoordinatorConfig, bool, error) {
	preSpawn := r.preSpawnPayload(decision, cfg, reason)
	preSpawn, err := r.dispatchPreSpawn(ctx, preSpawn)
	if err != nil {
		if preSpawn.Denied {
			r.dispatchDecision(ctx, decision, reason, coordinator.DecisionDenied)
			return nil, cfg, false, nil
		}
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, cfg, false, err
	}
	if preSpawn.Denied {
		r.dispatchDecision(ctx, decision, reason, coordinator.DecisionDenied)
		return nil, cfg, false, nil
	}

	cfg.AgentName = firstNonEmpty(preSpawn.AgentName, cfg.AgentName)
	cfg.Provider = firstNonEmpty(preSpawn.Provider, cfg.Provider)
	cfg.Model = firstNonEmpty(preSpawn.Model, cfg.Model)
	info, err := r.startCoordinatorSession(ctx, decision, cfg)
	if err != nil {
		r.dispatchFailed(ctx, decision, reason, err)
		return nil, cfg, false, err
	}
	return info, cfg, true, nil
}

func (r *coordinatorRuntime) promptCoordinator(
	ctx context.Context,
	info *session.Info,
	decision coordinator.Decision,
	reason string,
) error {
	if ctx == nil {
		return errors.New("daemon: coordinator prompt context is required")
	}
	if info == nil {
		return errors.New("daemon: coordinator prompt requires session info")
	}
	sessionID := strings.TrimSpace(info.ID)
	if sessionID == "" {
		return errors.New("daemon: coordinator prompt requires session id")
	}
	message := coordinatorWakeMessage(decision)
	events, err := r.sessions.PromptSynthetic(ctx, sessionID, session.SyntheticPromptOpts{
		Message: message,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:               strings.TrimSpace(decision.TaskID),
			TaskRunID:            strings.TrimSpace(decision.RunID),
			WorkflowID:           strings.TrimSpace(decision.WorkflowID),
			CoordinatorSessionID: sessionID,
			Reason:               strings.TrimSpace(reason),
			Summary:              coordinatorWakeSummary(decision),
		},
		InterruptIfAgentWaiting: true,
	})
	if err != nil {
		return fmt.Errorf("daemon: prompt coordinator session %q: %w", sessionID, err)
	}
	r.drainPromptEvents(sessionID, strings.TrimSpace(decision.RunID), events)
	return nil
}

func (r *coordinatorRuntime) drainPromptEvents(sessionID string, runID string, events <-chan acp.AgentEvent) {
	if r == nil || events == nil {
		return
	}
	r.wg.Go(func() {
		drainCoordinatorPromptEvents(r.ctx, r.logger, sessionID, runID, events)
	})
}

func drainCoordinatorPromptEvents(
	ctx context.Context,
	logger *slog.Logger,
	sessionID string,
	runID string,
	events <-chan acp.AgentEvent,
) {
	if events == nil {
		return
	}
	if ctx == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Type == acp.EventTypeError && logger != nil {
				logger.Warn(
					"daemon: coordinator prompt returned agent error",
					"session_id", sessionID,
					"run_id", runID,
				)
			}
		}
	}
}

func (r *coordinatorRuntime) beginCoordinatorWakeLocked(info *session.Info, decision coordinator.Decision) bool {
	if r == nil {
		return false
	}
	key := coordinatorWakeInFlightKey(info, decision)
	if key == "" {
		return true
	}
	if r.wakeInFlight == nil {
		r.wakeInFlight = make(map[string]struct{})
	}
	if _, ok := r.wakeInFlight[key]; ok {
		return false
	}
	r.wakeInFlight[key] = struct{}{}
	return true
}

func (r *coordinatorRuntime) finishCoordinatorWake(info *session.Info, decision coordinator.Decision) {
	if r == nil {
		return
	}
	key := coordinatorWakeInFlightKey(info, decision)
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.wakeInFlight, key)
}

func coordinatorWakeInFlightKey(info *session.Info, decision coordinator.Decision) string {
	if info == nil {
		return ""
	}
	sessionID := strings.TrimSpace(info.ID)
	runID := strings.TrimSpace(decision.RunID)
	if sessionID == "" || runID == "" {
		return ""
	}
	return sessionID + "\x00" + runID
}

func (r *coordinatorRuntime) shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: coordinator runtime shutdown context is required")
	}
	if r.cancel != nil {
		r.cancel()
	}
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("daemon: shutdown coordinator runtime: %w", ctx.Err())
	}
}

func coordinatorWakeMessage(decision coordinator.Decision) string {
	taskID := strings.TrimSpace(decision.TaskID)
	runID := strings.TrimSpace(decision.RunID)
	return fmt.Sprintf(
		"A task run is queued for this coordinator.\n\n"+
			"Task: %s\nRun: %s\n\n"+
			"Claim the run through the AGH task claim path by running `agh task next -o json` once without long-polling, then route from durable receipts. "+
			"If the receipts require human input, park the run with the AGH task block path.",
		taskID,
		runID,
	)
}

func coordinatorWakeSummary(decision coordinator.Decision) string {
	return fmt.Sprintf(
		"Coordinator wake for task %s run %s",
		strings.TrimSpace(decision.TaskID),
		strings.TrimSpace(decision.RunID),
	)
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
		coordinatorRuntimeTaskIDKey, strings.TrimSpace(payload.TaskID),
		"run_id", strings.TrimSpace(payload.RunID),
		coordinatorRuntimeWorkspaceIDKey, strings.TrimSpace(payload.WorkspaceID),
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
