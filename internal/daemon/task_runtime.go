package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	defaultTaskCancelGrace     = 5 * time.Second
	taskRecoveryReasonBoot     = "orphaned_on_boot"
	taskRecoverySessionMissing = "missing"
	taskStopDetailShutdown     = "task shutdown"
	taskStopDetailOrphaned     = "task run orphaned"
	taskStopDetailCancellation = "task cancellation"

	taskRecoveryClassificationLive     = "live"
	taskRecoveryClassificationMissing  = "missing"
	taskRecoveryClassificationCrashed  = "crashed"
	taskRecoveryClassificationOrphaned = "orphaned"
	taskRecoveryClassificationStalled  = "stalled"
)

type taskStore interface {
	taskpkg.Store
}

type taskRuntime struct {
	manager  *taskpkg.Service
	store    taskStore
	detached *harnessDetachedWorkBridge
	reentry  *harnessReentryBridge
}

type taskBridgeSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	Status(ctx context.Context, id string) (*session.Info, error)
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type taskBridgeSessionRequestStopper interface {
	RequestStopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type taskSessionBridge struct {
	sessions            taskBridgeSessionManager
	globalWorkspacePath string
	logger              *slog.Logger
}

type taskRecoveryStats struct {
	requeued      int
	markedRunning int
	failed        int
}

type taskSessionRecoveryEvidence struct {
	live           bool
	state          string
	classification string
	detail         string
}

var _ taskpkg.SessionExecutor = (*taskSessionBridge)(nil)

func newTaskSessionBridge(
	sessions taskBridgeSessionManager,
	globalWorkspacePath string,
	logger *slog.Logger,
) (*taskSessionBridge, error) {
	if sessions == nil {
		return nil, errors.New("daemon: task session bridge requires a session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &taskSessionBridge{
		sessions:            sessions,
		globalWorkspacePath: strings.TrimSpace(globalWorkspacePath),
		logger:              logger,
	}, nil
}

func (b *taskSessionBridge) StartTaskSession(
	ctx context.Context,
	spec *taskpkg.StartTaskSession,
) (*taskpkg.SessionRef, error) {
	if ctx == nil {
		return nil, errors.New("daemon: start task session context is required")
	}
	if spec == nil {
		return nil, errors.New("daemon: start task session spec is required")
	}

	opts := session.CreateOpts{
		Provider: "",
		Name:     taskSessionName(spec),
		Channel:  strings.TrimSpace(spec.Run.NetworkChannel),
		Type:     session.SessionTypeSystem,
	}
	switch spec.Task.Scope.Normalize() {
	case taskpkg.ScopeWorkspace:
		opts.Workspace = strings.TrimSpace(spec.Task.WorkspaceID)
	case taskpkg.ScopeGlobal:
		if b.globalWorkspacePath == "" {
			return nil, errors.New("daemon: task session bridge global workspace path is required")
		}
		opts.WorkspacePath = b.globalWorkspacePath
	default:
		return nil, fmt.Errorf(
			"%w: unsupported task scope %q for task session start",
			taskpkg.ErrValidation,
			spec.Task.Scope,
		)
	}

	created, err := b.sessions.Create(ctx, opts)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, fmt.Errorf("%w: task session bridge create returned nil session", taskpkg.ErrValidation)
	}
	info := created.Info()
	if info == nil {
		return nil, fmt.Errorf("%w: task session bridge create returned nil session info", taskpkg.ErrValidation)
	}
	return &taskpkg.SessionRef{
		SessionID:   strings.TrimSpace(info.ID),
		WorkspaceID: strings.TrimSpace(info.WorkspaceID),
		StartedAt:   info.CreatedAt,
	}, nil
}

func (b *taskSessionBridge) AttachTaskSession(
	ctx context.Context,
	_ string,
	sessionID string,
) (*taskpkg.SessionRef, error) {
	if ctx == nil {
		return nil, errors.New("daemon: attach task session context is required")
	}

	info, err := b.sessions.Status(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf(
			"%w: session %q is unavailable",
			taskpkg.ErrSessionAttachNotAllowed,
			strings.TrimSpace(sessionID),
		)
	}
	if !isTaskSessionStateLive(info.State) {
		return nil, fmt.Errorf(
			"%w: session %q is %q",
			taskpkg.ErrSessionAttachNotAllowed,
			strings.TrimSpace(sessionID),
			info.State,
		)
	}

	return &taskpkg.SessionRef{
		SessionID:   strings.TrimSpace(info.ID),
		WorkspaceID: strings.TrimSpace(info.WorkspaceID),
		StartedAt:   info.CreatedAt,
	}, nil
}

func (b *taskSessionBridge) RequestTaskStop(ctx context.Context, sessionID string, reason taskpkg.StopReason) error {
	if ctx == nil {
		return errors.New("daemon: request task stop context is required")
	}

	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return fmt.Errorf("%w: task session stop id is required", taskpkg.ErrValidation)
	}

	if requester, ok := b.sessions.(taskBridgeSessionRequestStopper); ok {
		if err := requester.RequestStopWithCause(
			ctx,
			trimmedID,
			taskStopCause(reason),
			taskStopDetail(reason),
		); err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				return nil
			}
			return err
		}
		return nil
	}

	return b.ForceTaskStop(ctx, trimmedID, reason)
}

func (b *taskSessionBridge) ForceTaskStop(ctx context.Context, sessionID string, reason taskpkg.StopReason) error {
	if ctx == nil {
		return errors.New("daemon: force task stop context is required")
	}

	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return fmt.Errorf("%w: task session stop id is required", taskpkg.ErrValidation)
	}

	if err := b.sessions.StopWithCause(ctx, trimmedID, taskStopCause(reason), taskStopDetail(reason)); err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func (d *Daemon) bootTasks(ctx context.Context, state *bootState) error {
	if state == nil || state.registry == nil || state.sessions == nil {
		return nil
	}

	store, ok := state.registry.(taskStore)
	if !ok {
		state.logger.Warn("daemon: task runtime skipped because registry does not implement task store")
		return nil
	}

	bridge, err := newTaskSessionBridge(state.sessions, d.homePaths.HomeDir, state.logger)
	if err != nil {
		return err
	}
	reentrySessions, ok := state.sessions.(harnessReentrySessionManager)
	if !ok {
		return errors.New("daemon: session manager does not support synthetic reentry bridge")
	}
	reentryStore, ok := state.registry.(harnessReentryStore)
	if !ok {
		return errors.New("daemon: global registry does not support harness reentry summaries")
	}
	reentry, err := newHarnessReentryBridge(
		ctx,
		state.harnessResolver,
		state.harnessRecorder,
		reentryStore,
		reentrySessions,
		state.logger,
	)
	if err != nil {
		return fmt.Errorf("daemon: create harness reentry bridge: %w", err)
	}
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(store),
		taskpkg.WithSessionExecutor(bridge),
		taskpkg.WithEventObserver(reentry),
		taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		taskpkg.WithCancelGracePeriod(defaultTaskCancelGrace),
	)
	if err != nil {
		return fmt.Errorf("daemon: create task manager: %w", err)
	}
	detached, err := newHarnessDetachedWorkBridge(manager, store, state.sessions)
	if err != nil {
		return fmt.Errorf("daemon: create detached harness bridge: %w", err)
	}

	state.tasks = &taskRuntime{
		manager:  manager,
		store:    store,
		detached: detached,
		reentry:  reentry,
	}
	state.deps.Tasks = manager

	actor, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
	if err != nil {
		return fmt.Errorf("daemon: derive task boot recovery actor: %w", err)
	}

	stats, err := recoverTaskRunsOnBoot(ctx, manager, store, state.sessions, actor)
	if err != nil {
		return err
	}
	if stats.requeued+stats.markedRunning+stats.failed > 0 {
		state.logger.Info(
			"daemon: task boot recovery complete",
			"requeued_runs", stats.requeued,
			"resumed_running_runs", stats.markedRunning,
			"failed_runs", stats.failed,
		)
	}
	if err := reentry.recover(ctx); err != nil {
		return fmt.Errorf("daemon: recover detached harness reentry bridge: %w", err)
	}
	return nil
}

func (r *taskRuntime) submitDetachedHarnessWork(
	ctx context.Context,
	req detachedHarnessSubmitRequest,
) (*detachedHarnessSubmission, error) {
	if r == nil {
		return nil, errors.New("daemon: task runtime is required")
	}
	if r.detached == nil {
		return nil, errors.New("daemon: detached harness bridge is required")
	}
	return r.detached.submit(ctx, req)
}

func (r *taskRuntime) shutdown() {
	if r == nil || r.reentry == nil {
		return
	}
	r.reentry.shutdown()
}

func recoverTaskRunsOnBoot(
	ctx context.Context,
	manager *taskpkg.Service,
	store taskStore,
	sessions taskBridgeSessionManager,
	actor taskpkg.ActorContext,
) (taskRecoveryStats, error) {
	runs, err := store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning,
	})
	if err != nil {
		return taskRecoveryStats{}, fmt.Errorf("daemon: list task runs for boot recovery: %w", err)
	}

	stats := taskRecoveryStats{}
	for _, run := range runs {
		recovery, err := planTaskRunRecovery(ctx, sessions, run)
		if err != nil {
			return taskRecoveryStats{}, fmt.Errorf("daemon: plan boot recovery for task run %q: %w", run.ID, err)
		}
		if recovery == nil {
			continue
		}
		if _, err := manager.RecoverRunOnBoot(ctx, run.ID, *recovery, actor); err != nil {
			return taskRecoveryStats{}, fmt.Errorf("daemon: recover task run %q on boot: %w", run.ID, err)
		}
		switch recovery.Action.Normalize() {
		case taskpkg.RunBootRecoveryRequeue:
			stats.requeued++
		case taskpkg.RunBootRecoveryMarkRunning:
			stats.markedRunning++
		case taskpkg.RunBootRecoveryFail:
			stats.failed++
		}
	}

	return stats, nil
}

func planTaskRunRecovery(
	ctx context.Context,
	sessions taskBridgeSessionManager,
	run taskpkg.Run,
) (*taskpkg.RunBootRecovery, error) {
	if sessions == nil {
		return nil, errors.New("daemon: task recovery requires a session manager")
	}

	evidence, err := inspectTaskSessionRecovery(ctx, sessions, strings.TrimSpace(run.SessionID))
	if err != nil {
		return nil, err
	}

	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusClaimed:
		if evidence.live {
			return &taskpkg.RunBootRecovery{
				Action:         taskpkg.RunBootRecoveryMarkRunning,
				Reason:         taskRecoveryReasonBoot,
				SessionState:   evidence.state,
				Classification: evidence.classification,
				Detail:         evidence.detail,
			}, nil
		}
		return &taskpkg.RunBootRecovery{
			Action:         taskpkg.RunBootRecoveryRequeue,
			Reason:         taskRecoveryReasonBoot,
			SessionState:   evidence.state,
			Classification: evidence.classification,
			Detail:         evidence.detail,
		}, nil

	case taskpkg.TaskRunStatusStarting:
		if evidence.live {
			return &taskpkg.RunBootRecovery{
				Action:         taskpkg.RunBootRecoveryMarkRunning,
				Reason:         taskRecoveryReasonBoot,
				SessionState:   evidence.state,
				Classification: evidence.classification,
				Detail:         evidence.detail,
			}, nil
		}
		return &taskpkg.RunBootRecovery{
			Action:         taskpkg.RunBootRecoveryFail,
			Reason:         taskRecoveryReasonBoot,
			SessionState:   evidence.state,
			Classification: evidence.classification,
			Detail:         evidence.detail,
		}, nil

	case taskpkg.TaskRunStatusRunning:
		if evidence.live {
			return nil, nil
		}
		return &taskpkg.RunBootRecovery{
			Action:         taskpkg.RunBootRecoveryFail,
			Reason:         taskRecoveryReasonBoot,
			SessionState:   evidence.state,
			Classification: evidence.classification,
			Detail:         evidence.detail,
		}, nil

	default:
		return nil, nil
	}
}

func taskSessionRuntimeState(
	ctx context.Context,
	sessions taskBridgeSessionManager,
	sessionID string,
) (bool, string, error) {
	evidence, err := inspectTaskSessionRecovery(ctx, sessions, sessionID)
	if err != nil {
		return false, "", err
	}
	return evidence.live, evidence.state, nil
}

func isTaskSessionStateLive(state session.State) bool {
	switch state {
	case session.StateStarting, session.StateActive, session.StateStopping:
		return true
	default:
		return false
	}
}

func inspectTaskSessionRecovery(
	ctx context.Context,
	sessions taskBridgeSessionManager,
	sessionID string,
) (taskSessionRecoveryEvidence, error) {
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return taskSessionRecoveryEvidence{
			state:          taskRecoverySessionMissing,
			classification: taskRecoveryClassificationMissing,
			detail:         "run has no bound session",
		}, nil
	}

	info, err := sessions.Status(ctx, trimmedID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return taskSessionRecoveryEvidence{
				state:          taskRecoverySessionMissing,
				classification: taskRecoveryClassificationMissing,
				detail:         "bound session metadata is missing",
			}, nil
		}
		return taskSessionRecoveryEvidence{}, err
	}
	if info == nil {
		return taskSessionRecoveryEvidence{
			state:          taskRecoverySessionMissing,
			classification: taskRecoveryClassificationMissing,
			detail:         "bound session metadata is missing",
		}, nil
	}

	evidence := taskSessionRecoveryEvidence{
		live:  isTaskSessionStateLive(info.State),
		state: string(info.State),
	}
	if evidence.state == "" {
		evidence.state = taskRecoverySessionMissing
	}
	if evidence.live {
		evidence.classification = taskRecoveryClassificationLive
		return evidence, nil
	}

	evidence.classification, evidence.detail = classifyRecoveredTaskSession(info, time.Now().UTC())
	return evidence, nil
}

func classifyRecoveredTaskSession(info *session.Info, now time.Time) (string, string) {
	if info == nil {
		return taskRecoveryClassificationMissing, "session metadata is unavailable"
	}
	if liveness := info.Liveness; liveness != nil {
		if strings.TrimSpace(liveness.StallState) == store.SessionStallStateDetected {
			return taskRecoveryClassificationStalled, firstTaskRecoveryDetail(
				liveness.StallReason,
				info.StopDetail,
				"session liveness monitor marked the process stalled",
			)
		}
		if lastActivityAt := taskSessionLastActivityAt(liveness); lastActivityAt != nil &&
			!lastActivityAt.IsZero() &&
			now.Sub(lastActivityAt.UTC()) >= session.DefaultLivenessStallAfter &&
			taskSessionMatchesRecordedSubprocess(liveness) {
			return taskRecoveryClassificationStalled, firstTaskRecoveryDetail(
				liveness.StallReason,
				store.SessionStallReasonActivityTimeout,
				info.StopDetail,
			)
		}
		if taskSessionMatchesRecordedSubprocess(liveness) {
			return taskRecoveryClassificationOrphaned, fmt.Sprintf(
				"subprocess pid %d is still alive without a live daemon owner",
				liveness.SubprocessPID,
			)
		}
	}
	return taskRecoveryClassificationCrashed, firstTaskRecoveryDetail(
		info.StopDetail,
		"bound session is not live",
	)
}

func taskSessionLastActivityAt(liveness *store.SessionLivenessMeta) *time.Time {
	if liveness == nil {
		return nil
	}
	if liveness.Activity != nil &&
		liveness.Activity.LastActivityAt != nil &&
		!liveness.Activity.LastActivityAt.IsZero() {
		return liveness.Activity.LastActivityAt
	}
	return liveness.LastUpdateAt
}

func taskSessionMatchesRecordedSubprocess(liveness *store.SessionLivenessMeta) bool {
	if liveness == nil || liveness.SubprocessPID <= 0 {
		return false
	}
	if liveness.SubprocessStartedAt == nil || liveness.SubprocessStartedAt.IsZero() {
		return false
	}
	return procutil.MatchesStartTime(liveness.SubprocessPID, *liveness.SubprocessStartedAt)
}

func firstTaskRecoveryDetail(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func taskSessionName(spec *taskpkg.StartTaskSession) string {
	if spec == nil {
		return "task:#0"
	}

	base := strings.TrimSpace(spec.Task.Title)
	if base == "" {
		base = strings.TrimSpace(spec.Task.Identifier)
	}
	if base == "" {
		base = strings.TrimSpace(spec.Run.ID)
	}
	return fmt.Sprintf("task:%s#%d", base, spec.Run.Attempt)
}

func taskStopCause(reason taskpkg.StopReason) session.StopCause {
	switch reason.Normalize() {
	case taskpkg.StopReasonShutdown:
		return session.CauseShutdown
	case taskpkg.StopReasonOrphanedRun:
		return session.CauseFailed
	default:
		return session.CauseUserRequested
	}
}

func taskStopDetail(reason taskpkg.StopReason) string {
	switch reason.Normalize() {
	case taskpkg.StopReasonShutdown:
		return taskStopDetailShutdown
	case taskpkg.StopReasonOrphanedRun:
		return taskStopDetailOrphaned
	default:
		return taskStopDetailCancellation
	}
}
