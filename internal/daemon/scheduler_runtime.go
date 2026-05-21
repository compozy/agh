package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	schedulerpkg "github.com/pedronauck/agh/internal/scheduler"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/situation"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	schedulerRuntimeTaskKey = "task"
)

const (
	defaultMechanicalSchedulerInterval   = 15 * time.Second
	defaultMechanicalSchedulerSweepLimit = 100
	mechanicalSchedulerSweepReason       = "scheduler_sweep"
	mechanicalSchedulerWakeReason        = "pending_task_run"
)

type schedulerRuntime struct {
	scheduler *schedulerpkg.Scheduler
	waker     *schedulerSessionWaker
}

type schedulerTaskSource struct {
	manager *taskpkg.Service
	store   taskStore
}

type effectiveTaskPauseStore interface {
	IsTaskEffectivelyPaused(ctx context.Context, taskID string) (bool, string, error)
}

type schedulerSessionSource struct {
	sessions  SessionManager
	situation *situation.Service
	logger    *slog.Logger
}

type schedulerPromptState interface {
	IsPrompting(sessionID string) bool
}

type schedulerSyntheticPrompter interface {
	PromptSynthetic(ctx context.Context, id string, opts session.SyntheticPromptOpts) (<-chan acp.AgentEvent, error)
}

type schedulerSessionWaker struct {
	ctx           context.Context
	cancel        context.CancelFunc
	sessions      SessionManager
	heartbeatWake heartbeat.WakeService
	logger        *slog.Logger
	wg            sync.WaitGroup
}

func (d *Daemon) bootScheduler(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil || state.tasks == nil || state.tasks.manager == nil || state.tasks.store == nil {
		return nil
	}
	if state.sessions == nil {
		return errors.New("daemon: scheduler requires session manager")
	}
	logger := state.logger
	if logger == nil {
		logger = slog.Default()
	}
	logSchedulerPauseState(ctx, state.tasks.store, logger)

	waker := newSchedulerSessionWaker(ctx, state.sessions, logger)
	if err := waker.configureHeartbeatWake(state.registry, state.sessions, state.cfg.Agents.Heartbeat); err != nil {
		return err
	}
	runtime, err := newSchedulerRuntime(
		ctx,
		state.tasks.manager,
		state.tasks.store,
		state.sessions,
		state.situationContext,
		waker,
		logger,
	)
	if err != nil {
		if shutdownErr := waker.shutdown(context.Background()); shutdownErr != nil {
			logger.Warn("daemon: cleanup scheduler waker after create failure", "error", shutdownErr)
		}
		return err
	}
	if _, err := runtime.scheduler.Rebuild(ctx); err != nil {
		if shutdownErr := runtime.shutdown(context.Background()); shutdownErr != nil {
			logger.Warn("daemon: cleanup scheduler after rebuild failure", "error", shutdownErr)
		}
		return fmt.Errorf("daemon: rebuild scheduler state: %w", err)
	}
	if err := runtime.scheduler.Start(ctx); err != nil {
		if shutdownErr := runtime.shutdown(context.Background()); shutdownErr != nil {
			logger.Warn("daemon: cleanup scheduler after start failure", "error", shutdownErr)
		}
		return fmt.Errorf("daemon: start scheduler: %w", err)
	}
	state.scheduler = runtime
	if cleanup != nil {
		cleanup.add(func(cleanupCtx context.Context) error {
			return runtime.shutdown(cleanupCtx)
		})
	}
	return nil
}

func logSchedulerPauseState(ctx context.Context, store taskStore, logger *slog.Logger) {
	if store == nil || logger == nil {
		return
	}
	if pauseReader, ok := store.(interface {
		GetSchedulerPause(context.Context) (taskpkg.SchedulerPauseState, error)
	}); ok {
		state, err := pauseReader.GetSchedulerPause(ctx)
		if err != nil {
			logger.Warn("daemon: scheduler pause state unavailable", "error", err)
		} else if state.Paused {
			logger.Warn(
				"daemon: scheduler booting paused",
				"paused_by", state.PausedBy,
				"reason", state.Reason,
			)
		}
	}
	if counter, ok := store.(interface {
		CountPausedTasks(context.Context) (int, error)
	}); ok {
		count, err := counter.CountPausedTasks(ctx)
		if err != nil {
			logger.Warn("daemon: paused task count unavailable", "error", err)
		} else if count > 0 {
			logger.Info("daemon: scheduler found paused tasks", "paused_task_count", count)
		}
	}
}

func newSchedulerRuntime(
	ctx context.Context,
	manager *taskpkg.Service,
	store taskStore,
	sessions SessionManager,
	situation *situation.Service,
	waker *schedulerSessionWaker,
	logger *slog.Logger,
) (*schedulerRuntime, error) {
	if ctx == nil {
		return nil, errors.New("daemon: scheduler context is required")
	}
	if manager == nil {
		return nil, errors.New("daemon: scheduler task manager is required")
	}
	if store == nil {
		return nil, errors.New("daemon: scheduler task store is required")
	}
	if sessions == nil {
		return nil, errors.New("daemon: scheduler session manager is required")
	}
	if waker == nil {
		return nil, errors.New("daemon: scheduler waker is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	var pauseStore schedulerpkg.PauseStore
	if candidate, ok := store.(schedulerpkg.PauseStore); ok {
		pauseStore = candidate
	}

	scheduler, err := schedulerpkg.New(
		schedulerTaskSource{manager: manager, store: store},
		schedulerSessionSource{sessions: sessions, situation: situation, logger: logger},
		waker,
		schedulerpkg.WithLogger(logger),
		schedulerpkg.WithInterval(defaultMechanicalSchedulerInterval),
		schedulerpkg.WithSweepReason(mechanicalSchedulerSweepReason),
		schedulerpkg.WithWakeReason(mechanicalSchedulerWakeReason),
		schedulerpkg.WithSweepLimit(defaultMechanicalSchedulerSweepLimit),
		schedulerpkg.WithPauseStore(pauseStore),
	)
	if err != nil {
		return nil, fmt.Errorf("daemon: create scheduler: %w", err)
	}
	return &schedulerRuntime{scheduler: scheduler, waker: waker}, nil
}

func (r *schedulerRuntime) shutdown(ctx context.Context) error {
	return errors.Join(r.stopLoop(ctx), r.shutdownWaker(ctx))
}

func (r *schedulerRuntime) stopLoop(ctx context.Context) error {
	if r == nil || r.scheduler == nil {
		return nil
	}
	return r.scheduler.Shutdown(ctx)
}

func (r *schedulerRuntime) shutdownWaker(ctx context.Context) error {
	if r == nil || r.waker == nil {
		return nil
	}
	return r.waker.shutdown(ctx)
}

func (s schedulerTaskSource) PendingRuns(ctx context.Context) ([]schedulerpkg.RunSnapshot, error) {
	runs, err := s.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		return nil, err
	}
	snapshots, err := s.joinRunsWithTasks(ctx, runs)
	if err != nil {
		return nil, err
	}
	return s.filterPausedRuns(ctx, snapshots)
}

func (s schedulerTaskSource) ActiveRuns(ctx context.Context) ([]taskpkg.Run, error) {
	return s.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning,
	})
}

func (s schedulerTaskSource) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
	actor taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	return s.manager.RecoverExpiredRunLeases(ctx, recovery, actor)
}

func (s schedulerTaskSource) joinRunsWithTasks(
	ctx context.Context,
	runs []taskpkg.Run,
) ([]schedulerpkg.RunSnapshot, error) {
	work := make([]schedulerpkg.RunSnapshot, 0, len(runs))
	for _, run := range runs {
		taskRecord, err := s.store.GetTask(ctx, run.TaskID)
		if err != nil {
			return nil, fmt.Errorf("daemon: scheduler load task %q for run %q: %w", run.TaskID, run.ID, err)
		}
		if taskRecord.Paused {
			continue
		}
		if pauseReader, ok := s.store.(interface {
			IsTaskEffectivelyPaused(context.Context, string) (bool, string, error)
		}); ok {
			paused, _, err := pauseReader.IsTaskEffectivelyPaused(ctx, taskRecord.ID)
			if err != nil {
				return nil, err
			}
			if paused {
				continue
			}
		}
		work = append(work, schedulerpkg.RunSnapshot{Task: taskRecord, Run: run})
	}
	return work, nil
}

func (s schedulerTaskSource) filterPausedRuns(
	ctx context.Context,
	snapshots []schedulerpkg.RunSnapshot,
) ([]schedulerpkg.RunSnapshot, error) {
	pauseStore, ok := s.store.(effectiveTaskPauseStore)
	if !ok {
		return snapshots, nil
	}
	filtered := make([]schedulerpkg.RunSnapshot, 0, len(snapshots))
	for idx := range snapshots {
		snapshot := &snapshots[idx]
		paused, _, err := pauseStore.IsTaskEffectivelyPaused(ctx, snapshot.Task.ID)
		if err != nil {
			return nil, fmt.Errorf("daemon: scheduler check task %q pause: %w", snapshot.Task.ID, err)
		}
		if paused {
			continue
		}
		filtered = append(filtered, *snapshot)
	}
	return filtered, nil
}

func (s schedulerSessionSource) Sessions(ctx context.Context) ([]schedulerpkg.SessionSnapshot, error) {
	if s.sessions == nil {
		return nil, errors.New("daemon: scheduler session source requires session manager")
	}
	infos := s.sessions.List()
	snapshots := make([]schedulerpkg.SessionSnapshot, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		capabilities, err := s.capabilities(ctx, info)
		if err != nil {
			if ctx.Err() != nil {
				return nil, err
			}
			if s.logger != nil {
				s.logger.Warn(
					"scheduler.session_context.error",
					"session_id", info.ID,
					"error", err,
				)
			}
			continue
		}
		snapshots = append(snapshots, schedulerpkg.SessionSnapshot{
			ID:           strings.TrimSpace(info.ID),
			AgentName:    strings.TrimSpace(info.AgentName),
			WorkspaceID:  strings.TrimSpace(info.WorkspaceID),
			Channel:      strings.TrimSpace(info.Channel),
			Type:         strings.TrimSpace(string(info.Type)),
			State:        strings.TrimSpace(string(info.State)),
			Prompting:    isSchedulerSessionPrompting(s.sessions, info.ID),
			Capabilities: capabilities,
			CreatedAt:    info.CreatedAt,
		})
	}
	return snapshots, nil
}

func (s schedulerSessionSource) capabilities(ctx context.Context, info *session.Info) ([]string, error) {
	if s.situation == nil {
		return nil, nil
	}
	payload, err := s.situation.ContextForSession(ctx, info)
	if err != nil {
		return nil, err
	}
	capabilities := make([]string, 0, len(payload.Capabilities.Capabilities))
	for _, capability := range payload.Capabilities.Capabilities {
		if id := strings.TrimSpace(capability.ID); id != "" {
			capabilities = append(capabilities, id)
		}
	}
	return capabilities, nil
}

func isSchedulerSessionPrompting(sessions SessionManager, sessionID string) bool {
	promptState, ok := sessions.(schedulerPromptState)
	return ok && promptState.IsPrompting(strings.TrimSpace(sessionID))
}

func newSchedulerSessionWaker(
	ctx context.Context,
	sessions SessionManager,
	logger *slog.Logger,
) *schedulerSessionWaker {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	wakeCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	return &schedulerSessionWaker{
		ctx:      wakeCtx,
		cancel:   cancel,
		sessions: sessions,
		logger:   logger,
	}
}

func (w *schedulerSessionWaker) Wake(ctx context.Context, target *schedulerpkg.WakeTarget) error {
	if w == nil || w.sessions == nil {
		return errors.New("daemon: scheduler waker requires session manager")
	}
	return w.wakeOne(ctx, target)
}

func (w *schedulerSessionWaker) WakeMany(
	ctx context.Context,
	targets []schedulerpkg.WakeTarget,
) []error {
	errs := make([]error, len(targets))
	if w == nil || w.sessions == nil {
		err := errors.New("daemon: scheduler waker requires session manager")
		for idx := range errs {
			errs[idx] = err
		}
		return errs
	}
	if w.heartbeatWake == nil {
		for idx := range targets {
			errs[idx] = w.wakeOne(ctx, &targets[idx])
		}
		return errs
	}

	requests, indexes := w.prepareHeartbeatWakeBatch(ctx, targets, errs)
	if len(requests) == 0 {
		return errs
	}
	decisions, err := w.heartbeatWake.WakeMany(ctx, requests)
	for decisionIdx, decision := range decisions {
		if decisionIdx >= len(indexes) {
			break
		}
		idx := indexes[decisionIdx]
		sessionID := strings.TrimSpace(targets[idx].Session.ID)
		errs[idx] = w.handleHeartbeatDecision(ctx, &targets[idx], sessionID, decision)
	}
	missingStart := min(len(decisions), len(indexes))
	if err == nil && missingStart < len(indexes) {
		err = errors.New("daemon: heartbeat wake batch returned fewer decisions than requests")
	}
	for _, idx := range indexes[missingStart:] {
		errs[idx] = err
	}
	return errs
}

func (w *schedulerSessionWaker) wakeOne(ctx context.Context, target *schedulerpkg.WakeTarget) error {
	sessionID, err := schedulerWakeSessionID(target)
	if err != nil {
		return err
	}
	if req, ok := schedulerHeartbeatWakeRequest(target, sessionID); ok && w.heartbeatWake != nil {
		decision, wakeErr := w.heartbeatWake.Wake(ctx, req)
		if wakeErr != nil {
			return wakeErr
		}
		return w.handleHeartbeatDecision(ctx, target, sessionID, decision)
	}
	return w.wakePendingTaskRun(ctx, target, sessionID)
}

func (w *schedulerSessionWaker) prepareHeartbeatWakeBatch(
	ctx context.Context,
	targets []schedulerpkg.WakeTarget,
	errs []error,
) ([]heartbeat.WakeRequest, []int) {
	requests := make([]heartbeat.WakeRequest, 0, len(targets))
	indexes := make([]int, 0, len(targets))
	for idx := range targets {
		target := &targets[idx]
		sessionID, err := schedulerWakeSessionID(target)
		if err != nil {
			errs[idx] = err
			continue
		}
		req, ok := schedulerHeartbeatWakeRequest(target, sessionID)
		if !ok {
			errs[idx] = w.wakePendingTaskRun(ctx, target, sessionID)
			continue
		}
		requests = append(requests, req)
		indexes = append(indexes, idx)
	}
	return requests, indexes
}

func (w *schedulerSessionWaker) handleHeartbeatDecision(
	ctx context.Context,
	target *schedulerpkg.WakeTarget,
	sessionID string,
	decision heartbeat.WakeDecision,
) error {
	switch decision.Result {
	case heartbeat.WakeResultSent:
		return nil
	case heartbeat.WakeResultSkipped:
		if decision.Reason == heartbeat.WakeReasonHeartbeatNoPolicy {
			return w.wakePendingTaskRun(ctx, target, sessionID)
		}
		return nil
	case heartbeat.WakeResultCoalesced, heartbeat.WakeResultRateLimited:
		return nil
	case heartbeat.WakeResultFailed:
		return fmt.Errorf("daemon: heartbeat wake failed: %s", decision.Reason)
	default:
		return fmt.Errorf("daemon: unexpected heartbeat wake result %q", decision.Result)
	}
}

func (w *schedulerSessionWaker) wakePendingTaskRun(
	ctx context.Context,
	target *schedulerpkg.WakeTarget,
	sessionID string,
) error {
	message := schedulerWakeMessage(target)
	if synthetic, ok := w.sessions.(schedulerSyntheticPrompter); ok {
		metadata := acp.PromptSyntheticMeta{
			TaskID:         strings.TrimSpace(target.Work.Task.ID),
			TaskRunID:      strings.TrimSpace(target.Work.Run.ID),
			ClaimTokenHash: strings.TrimSpace(target.Work.Run.ClaimTokenHash),
			Reason:         strings.TrimSpace(target.Reason),
			Summary:        schedulerWakeSummary(target),
		}
		if strings.TrimSpace(target.Session.Type) == string(session.SessionTypeCoordinator) {
			metadata.CoordinatorSessionID = strings.TrimSpace(sessionID)
		}
		events, err := synthetic.PromptSynthetic(ctx, sessionID, session.SyntheticPromptOpts{
			Message:  message,
			Metadata: metadata,
		})
		if err != nil {
			return err
		}
		w.drainEvents(sessionID, target.Work.Run.ID, events)
		return nil
	}

	events, err := w.sessions.Prompt(ctx, sessionID, message)
	if err != nil {
		return err
	}
	w.drainEvents(sessionID, target.Work.Run.ID, events)
	return nil
}

func schedulerWakeSessionID(target *schedulerpkg.WakeTarget) (string, error) {
	if target == nil {
		return "", errors.New("daemon: scheduler wake target is required")
	}
	sessionID := strings.TrimSpace(target.Session.ID)
	if sessionID == "" {
		return "", errors.New("daemon: scheduler wake session id is required")
	}
	return sessionID, nil
}

func schedulerHeartbeatWakeRequest(
	target *schedulerpkg.WakeTarget,
	sessionID string,
) (heartbeat.WakeRequest, bool) {
	if target == nil {
		return heartbeat.WakeRequest{}, false
	}
	workspaceID := strings.TrimSpace(target.Session.WorkspaceID)
	agentName := strings.TrimSpace(target.Session.AgentName)
	if workspaceID == "" || agentName == "" {
		return heartbeat.WakeRequest{}, false
	}
	return heartbeat.WakeRequest{
		WorkspaceID: workspaceID,
		AgentName:   agentName,
		SessionID:   strings.TrimSpace(sessionID),
		Source:      heartbeat.WakeSourceScheduler,
	}, true
}

func (w *schedulerSessionWaker) configureHeartbeatWake(
	store any,
	sessions SessionManager,
	config aghconfig.HeartbeatConfig,
) error {
	if w == nil || !config.Enabled {
		return nil
	}
	wakeStore, ok := store.(heartbeat.WakeStore)
	if !ok {
		return nil
	}
	healthReader, ok := sessions.(heartbeat.SessionHealthReader)
	if !ok {
		return nil
	}
	service, err := heartbeat.NewManagedWakeService(wakeStore, healthReader, w, config)
	if err != nil {
		return fmt.Errorf("daemon: create scheduler heartbeat wake service: %w", err)
	}
	w.heartbeatWake = service
	return nil
}

func (w *schedulerSessionWaker) PromptHeartbeatWake(
	ctx context.Context,
	req heartbeat.SyntheticWakePromptRequest,
) (heartbeat.SyntheticWakePromptResult, error) {
	if w == nil || w.sessions == nil {
		return heartbeat.SyntheticWakePromptResult{}, errors.New(
			"daemon: scheduler heartbeat prompter requires sessions",
		)
	}
	synthetic, ok := w.sessions.(schedulerSyntheticPrompter)
	if !ok {
		return heartbeat.SyntheticWakePromptResult{}, errors.New(
			"daemon: scheduler heartbeat prompter requires synthetic prompt support",
		)
	}
	events, err := synthetic.PromptSynthetic(ctx, req.SessionID, session.SyntheticPromptOpts{
		Message: req.Message,
		TurnID:  req.TurnID,
		Metadata: acp.PromptSyntheticMeta{
			TaskID:               req.SyntheticCorrelation.TaskID,
			TaskRunID:            req.SyntheticCorrelation.TaskRunID,
			WorkflowID:           req.SyntheticCorrelation.WorkflowID,
			ClaimTokenHash:       req.SyntheticCorrelation.ClaimTokenHash,
			CoordinatorSessionID: req.SyntheticCorrelation.CoordinatorSessionID,
			Reason:               heartbeat.SyntheticReasonHeartbeatWake,
			Summary:              req.Summary,
			WakeEventID:          req.WakeEventID,
			PolicySnapshotID:     req.PolicySnapshotID,
			PolicyDigest:         req.PolicyDigest,
			ConfigDigest:         req.ConfigDigest,
		},
		SkipIfBusy: true,
	})
	if err != nil {
		if errors.Is(err, session.ErrPromptInProgress) {
			return heartbeat.SyntheticWakePromptResult{}, heartbeat.ErrSyntheticPromptBusy
		}
		return heartbeat.SyntheticWakePromptResult{}, err
	}
	w.drainEvents(req.SessionID, req.WakeEventID, events)
	return heartbeat.SyntheticWakePromptResult{SyntheticPromptID: req.TurnID}, nil
}

func (w *schedulerSessionWaker) drainEvents(sessionID string, runID string, events <-chan acp.AgentEvent) {
	if events == nil {
		return
	}
	w.wg.Go(func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				if event.Type == acp.EventTypeError && w.logger != nil {
					w.logger.Warn(
						"scheduler.wake.agent_error",
						"session_id", sessionID,
						"run_id", runID,
					)
				}
			}
		}
	})
}

func (w *schedulerSessionWaker) shutdown(ctx context.Context) error {
	if w == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("daemon: scheduler waker shutdown context is required")
	}
	if w.cancel != nil {
		w.cancel()
	}
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("daemon: shutdown scheduler waker: %w", ctx.Err())
	}
}

func schedulerWakeMessage(target *schedulerpkg.WakeTarget) string {
	if target == nil {
		return ""
	}
	taskTitle := firstSchedulerWakeValue(
		target.Work.Task.Title,
		target.Work.Task.Identifier,
		target.Work.Task.ID,
	)
	return fmt.Sprintf(
		"A task run is queued and may be claimable by this session.\n\n"+
			"Task: %s\nRun: %s\n\n"+
			"Use the existing task claim path before doing any work. "+
			"This notification does not assign ownership.",
		taskTitle,
		strings.TrimSpace(target.Work.Run.ID),
	)
}

func schedulerWakeSummary(target *schedulerpkg.WakeTarget) string {
	if target == nil {
		return ""
	}
	return fmt.Sprintf(
		"Pending task run %s for task %s",
		strings.TrimSpace(target.Work.Run.ID),
		firstSchedulerWakeValue(target.Work.Task.Title, target.Work.Task.ID),
	)
}

func firstSchedulerWakeValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return schedulerRuntimeTaskKey
}
