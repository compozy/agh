package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/procutil"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	defaultTaskCancelGrace     = 5 * time.Second
	taskRecoveryReasonBoot     = "orphaned_on_boot"
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
	manager             *taskpkg.Service
	store               taskStore
	detached            *harnessDetachedWorkBridge
	reentry             *harnessReentryBridge
	bridgeNotifications *bridgeTerminalTaskNotificationObserver
	roles               *taskRoleRuntime
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
	contextOverlay      taskSessionContextOverlay
	logger              *slog.Logger
}

type taskSessionContextOverlay interface {
	TaskRunPromptOverlay(
		ctx context.Context,
		taskRecord taskpkg.Task,
		run taskpkg.Run,
		profile *taskpkg.ExecutionProfile,
	) (string, error)
}

type taskSessionBridgeOption func(*taskSessionBridge)

func withTaskSessionContextOverlay(overlay taskSessionContextOverlay) taskSessionBridgeOption {
	return func(bridge *taskSessionBridge) {
		if bridge != nil {
			bridge.contextOverlay = overlay
		}
	}
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
	options ...taskSessionBridgeOption,
) (*taskSessionBridge, error) {
	if sessions == nil {
		return nil, errors.New("daemon: task session bridge requires a session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}
	bridge := &taskSessionBridge{
		sessions:            sessions,
		globalWorkspacePath: strings.TrimSpace(globalWorkspacePath),
		logger:              logger,
	}
	for _, option := range options {
		if option != nil {
			option(bridge)
		}
	}
	return bridge, nil
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
		AgentName: taskSessionAgentName(spec.Task),
		Provider:  "",
		Name:      taskSessionName(spec),
		Channel:   taskRunSessionChannel(spec.Run),
		Type:      session.SessionTypeSystem,
	}
	applyTaskSessionWorkerProfile(&opts, spec.ExecutionProfile)
	applyTaskSessionSandboxProfile(&opts, spec.ExecutionProfile)
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
	if b.contextOverlay != nil {
		overlay, err := b.contextOverlay.TaskRunPromptOverlay(ctx, spec.Task, spec.Run, spec.ExecutionProfile)
		if err != nil {
			return nil, fmt.Errorf("daemon: render task session context overlay: %w", err)
		}
		opts.PromptOverlay = joinPromptOverlays(opts.PromptOverlay, overlay)
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

func joinPromptOverlays(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n\n")
}

func applyTaskSessionWorkerProfile(opts *session.CreateOpts, profile *taskpkg.ExecutionProfile) {
	if opts == nil || profile == nil {
		return
	}
	worker := profile.Worker
	opts.AgentName = strings.TrimSpace(worker.AgentName)
	opts.Provider = strings.TrimSpace(worker.Provider)
	opts.Model = strings.TrimSpace(worker.Model)
}

func applyTaskSessionSandboxProfile(opts *session.CreateOpts, profile *taskpkg.ExecutionProfile) {
	if opts == nil || profile == nil {
		return
	}
	switch profile.Sandbox.Mode.Normalize() {
	case taskpkg.SandboxModeNone:
		opts.DisableSandbox = true
		opts.SandboxRef = ""
	case taskpkg.SandboxModeRef:
		opts.DisableSandbox = false
		opts.SandboxRef = strings.TrimSpace(profile.Sandbox.SandboxRef)
	default:
		return
	}
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

	bridge, err := newTaskSessionBridge(
		state.sessions,
		d.homePaths.HomeDir,
		state.logger,
		withTaskSessionContextOverlay(state.situationContext),
	)
	if err != nil {
		return err
	}
	reentry, err := bootHarnessReentryBridge(ctx, state)
	if err != nil {
		return fmt.Errorf("daemon: create harness reentry bridge: %w", err)
	}
	reviewRequests := newRunReviewRequestedForwarder()
	eventObserver, bridgeNotifications := d.composeTaskEventObserver(state, store, reentry)
	manager, err := taskpkg.NewManager(
		taskManagerOptions(
			store,
			bridge,
			eventObserver,
			state.notifier,
			reviewRequests,
			state.cfg.Task.Recovery,
		)...,
	)
	if err != nil {
		return fmt.Errorf("daemon: create task manager: %w", err)
	}
	detached, err := newHarnessDetachedWorkBridge(manager, store, state.sessions)
	if err != nil {
		return fmt.Errorf("daemon: create detached harness bridge: %w", err)
	}

	state.tasks = &taskRuntime{
		manager:             manager,
		store:               store,
		detached:            detached,
		reentry:             reentry,
		bridgeNotifications: bridgeNotifications,
	}
	state.reviewRequests = reviewRequests
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
	if reentry != nil {
		if err := reentry.recover(ctx); err != nil {
			return fmt.Errorf("daemon: recover detached harness reentry bridge: %w", err)
		}
	}
	return nil
}

func bootHarnessReentryBridge(ctx context.Context, state *bootState) (*harnessReentryBridge, error) {
	reentrySessions, ok := state.sessions.(harnessReentrySessionManager)
	if !ok {
		if state != nil && state.logger != nil {
			state.logger.Warn(
				"daemon: synthetic reentry bridge disabled because session manager support is unavailable",
			)
		}
		return nil, nil
	}
	reentryStore, ok := state.registry.(harnessReentryStore)
	if !ok {
		if state != nil && state.logger != nil {
			state.logger.Warn("daemon: synthetic reentry bridge disabled because registry support is unavailable")
		}
		return nil, nil
	}
	return newHarnessReentryBridge(
		ctx,
		state.harnessResolver,
		state.harnessRecorder,
		reentryStore,
		reentrySessions,
		state.logger,
		withHarnessHeartbeatWake(state.registry, reentrySessions, state.cfg.Agents.Heartbeat),
	)
}

func (d *Daemon) bootSpawnReaper(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
	if state == nil || state.sessions == nil || state.tasks == nil || state.tasks.manager == nil {
		return nil
	}
	logger := state.logger
	if logger == nil {
		logger = slog.Default()
	}
	reaper, err := newSpawnReaper(
		ctx,
		state.sessions,
		state.tasks.manager,
		state.notifier,
		logger.With("component", "spawn_reaper"),
		d.now,
		defaultSpawnReaperInterval,
	)
	if err != nil {
		return err
	}
	report, err := reaper.Sweep(ctx)
	if err != nil {
		return err
	}
	if report.Reaped > 0 {
		logger.Info(
			"daemon: spawn reaper boot sweep complete",
			"reaped", report.Reaped,
			"released_leases", report.ReleasedLeases,
			"ttl_expired", report.TTLExpired,
			"parent_stopped", report.ParentStopped,
			"orphaned", report.Orphaned,
		)
	}
	reaper.start()
	state.spawnReaper = reaper
	if cleanup != nil {
		cleanup.add(func(cleanupCtx context.Context) error {
			return reaper.shutdown(cleanupCtx)
		})
	}
	return nil
}

func (d *Daemon) bootTaskRoles(ctx context.Context, state *bootState) error {
	if state == nil || state.tasks == nil || state.tasks.store == nil || state.sessions == nil {
		return nil
	}
	runtime, err := newTaskRoleRuntime(state.tasks.store, state.sessions, d.homePaths.HomeDir, state.logger)
	if err != nil {
		return err
	}
	if state.notifier != nil {
		state.notifier.AddTaskRunEnqueuedObserver(runtime)
	}
	runtime.Recover(ctx)
	state.tasks.roles = runtime
	return nil
}

func taskManagerOptions(
	store taskStore,
	bridge taskpkg.SessionExecutor,
	events taskpkg.EventObserver,
	hooks *hooksNotifier,
	reviewRequests taskpkg.RunReviewRequestedObserver,
	recovery aghconfig.TaskRecoveryConfig,
) []taskpkg.Option {
	options := []taskpkg.Option{
		taskpkg.WithStore(store),
		taskpkg.WithSessionExecutor(bridge),
		taskpkg.WithEventObserver(events),
		taskpkg.WithRunReviewRequestedObserver(reviewRequests),
		taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		taskpkg.WithCancelGracePeriod(defaultTaskCancelGrace),
		taskpkg.WithForceRecoveryOptions(taskpkg.ForceRecoveryOptions{
			AllowAgentForce: recovery.AllowAgentForce,
		}),
	}
	if hooks != nil {
		options = append(options, taskpkg.WithTaskRunHooks(hooks))
	}
	if reader, ok := store.(taskpkg.InspectStateReader); ok {
		options = append(options, taskpkg.WithInspectStateReader(reader))
	}
	return options
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
	if r == nil {
		return
	}
	if r.bridgeNotifications != nil {
		r.bridgeNotifications.shutdown()
	}
	if r.reentry != nil {
		r.reentry.shutdown()
	}
}

func recoverTaskRunsOnBoot(
	ctx context.Context,
	manager *taskpkg.Service,
	store taskStore,
	sessions taskBridgeSessionManager,
	actor taskpkg.ActorContext,
) (taskRecoveryStats, error) {
	expired, err := manager.RecoverExpiredRunLeases(ctx, taskpkg.ExpiredLeaseRecovery{
		Reason: taskRecoveryReasonBoot,
	}, actor)
	if err != nil {
		return taskRecoveryStats{}, fmt.Errorf("daemon: recover expired task run leases on boot: %w", err)
	}

	runs, err := store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{
		taskpkg.TaskRunStatusClaimed,
		taskpkg.TaskRunStatusStarting,
		taskpkg.TaskRunStatusRunning,
	})
	if err != nil {
		return taskRecoveryStats{}, fmt.Errorf("daemon: list task runs for boot recovery: %w", err)
	}

	stats := taskRecoveryStats{requeued: len(expired)}
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

func taskSessionAgentName(taskRecord taskpkg.Task) string {
	if taskRecord.Owner == nil || taskRecord.Owner.IsZero() {
		return ""
	}
	owner := *taskRecord.Owner
	if owner.Kind.Normalize() != taskpkg.OwnerKindPool {
		return ""
	}
	return strings.TrimSpace(owner.Ref)
}

func taskRunSessionChannel(run taskpkg.Run) string {
	if channel := strings.TrimSpace(run.CoordinationChannelID); channel != "" {
		return channel
	}
	return strings.TrimSpace(run.NetworkChannel)
}

func taskStopCause(reason taskpkg.StopReason) session.StopCause {
	switch reason.Normalize() {
	case taskpkg.StopReasonCompleted:
		return session.CauseCompleted
	case taskpkg.StopReasonFailed:
		return session.CauseFailed
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
	case taskpkg.StopReasonCompleted:
		return "task completed"
	case taskpkg.StopReasonFailed:
		return "task failed"
	case taskpkg.StopReasonShutdown:
		return taskStopDetailShutdown
	case taskpkg.StopReasonOrphanedRun:
		return taskStopDetailOrphaned
	default:
		return taskStopDetailCancellation
	}
}
