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
	schedulerpkg "github.com/pedronauck/agh/internal/scheduler"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/situation"
	taskpkg "github.com/pedronauck/agh/internal/task"
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
	ctx      context.Context
	cancel   context.CancelFunc
	sessions SessionManager
	logger   *slog.Logger
	wg       sync.WaitGroup
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

	waker := newSchedulerSessionWaker(ctx, state.sessions, logger)
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

	scheduler, err := schedulerpkg.New(
		schedulerTaskSource{manager: manager, store: store},
		schedulerSessionSource{sessions: sessions, situation: situation, logger: logger},
		waker,
		schedulerpkg.WithLogger(logger),
		schedulerpkg.WithInterval(defaultMechanicalSchedulerInterval),
		schedulerpkg.WithSweepReason(mechanicalSchedulerSweepReason),
		schedulerpkg.WithWakeReason(mechanicalSchedulerWakeReason),
		schedulerpkg.WithSweepLimit(defaultMechanicalSchedulerSweepLimit),
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
	return s.joinRunsWithTasks(ctx, runs)
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
		work = append(work, schedulerpkg.RunSnapshot{Task: taskRecord, Run: run})
	}
	return work, nil
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
	//nolint:gosec // cancel is owned by the wake dispatcher and used during daemon shutdown.
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
	if target == nil {
		return errors.New("daemon: scheduler wake target is required")
	}
	sessionID := strings.TrimSpace(target.Session.ID)
	if sessionID == "" {
		return errors.New("daemon: scheduler wake session id is required")
	}
	message := schedulerWakeMessage(target)
	if synthetic, ok := w.sessions.(schedulerSyntheticPrompter); ok {
		events, err := synthetic.PromptSynthetic(ctx, sessionID, session.SyntheticPromptOpts{
			Message: message,
			Metadata: acp.PromptSyntheticMeta{
				TaskID:    strings.TrimSpace(target.Work.Task.ID),
				TaskRunID: strings.TrimSpace(target.Work.Run.ID),
				Reason:    strings.TrimSpace(target.Reason),
				Summary:   schedulerWakeSummary(target),
			},
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
	return "task"
}
