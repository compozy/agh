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
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	networkWakePollInterval = time.Second
	networkWakeQueueSize    = 256
	networkWakeScanLimit    = 1000
)

type networkWakeTaskManager interface {
	ClaimNextRun(
		context.Context,
		taskpkg.ClaimCriteria,
		taskpkg.ActorContext,
	) (*taskpkg.ClaimResult, error)
	HeartbeatRunLease(
		context.Context,
		taskpkg.LeaseHeartbeat,
		taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	ReleaseRunLease(
		context.Context,
		taskpkg.LeaseRelease,
		taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	CompleteRunLease(
		context.Context,
		taskpkg.LeaseCompletion,
		taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	FailRunLease(
		context.Context,
		taskpkg.LeaseFailure,
		taskpkg.ActorContext,
	) (*taskpkg.Run, error)
}

type networkWakePrompter interface {
	PromptNetwork(
		ctx context.Context,
		sessionID string,
		message string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
	CancelPrompt(ctx context.Context, sessionID string) error
}

type networkWakeStore interface {
	store.NetworkWakeQueueStore
	store.NetworkAcceptanceStore
}

type networkWakeRunner struct {
	tasks    networkWakeTaskManager
	prompter networkWakePrompter
	store    networkWakeStore
	logger   *slog.Logger
	now      func() time.Time

	notifications chan store.CommittedNetworkNotification
	mu            sync.Mutex
	enabled       bool
	active        map[string]context.CancelFunc
	runCancel     context.CancelFunc
	runDone       chan error
}

func newNetworkWakeRunner(
	tasks networkWakeTaskManager,
	prompter networkWakePrompter,
	wakeStore networkWakeStore,
	logger *slog.Logger,
) (*networkWakeRunner, error) {
	if tasks == nil {
		return nil, errors.New("daemon: network wake task manager is required")
	}
	if prompter == nil {
		return nil, errors.New("daemon: network wake prompter is required")
	}
	if wakeStore == nil {
		return nil, errors.New("daemon: network wake store is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &networkWakeRunner{
		tasks:         tasks,
		prompter:      prompter,
		store:         wakeStore,
		logger:        logger,
		now:           func() time.Time { return time.Now().UTC() },
		notifications: make(chan store.CommittedNetworkNotification, networkWakeQueueSize),
		enabled:       true,
		active:        make(map[string]context.CancelFunc),
	}, nil
}

// NotifyNetworkWake implements network.WakeNotifier after the acceptance commit.
func (r *networkWakeRunner) NotifyNetworkWake(notification store.CommittedNetworkNotification) {
	if r == nil {
		return
	}
	select {
	case r.notifications <- notification:
	default:
		r.logger.Warn(
			"daemon.network_wake.notification_deferred",
			"task_run_id", notification.TaskRunID,
			"recipient_session_id", notification.RecipientSessionID,
		)
	}
}

// SetNetworkEnabled cancels active turns after the availability write commits.
func (r *networkWakeRunner) SetNetworkEnabled(enabled bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.enabled = enabled
	if enabled {
		r.mu.Unlock()
		return
	}
	cancels := make([]context.CancelFunc, 0, len(r.active))
	for _, cancel := range r.active {
		cancels = append(cancels, cancel)
	}
	r.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

// Run executes committed wake task runs until the daemon lifecycle ends.
func (r *networkWakeRunner) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("daemon: network wake runner context is required")
	}
	actor, err := taskpkg.DeriveDaemonActorContext("network-wake-runner", "daemon.network_wake")
	if err != nil {
		return fmt.Errorf("daemon: derive network wake actor: %w", err)
	}
	if err := r.enqueueDurableWakes(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(networkWakePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.cancelActive()
			return nil
		case notification := <-r.notifications:
			if err := r.processNotification(ctx, actor, notification); err != nil {
				r.logger.Error(
					"daemon.network_wake.execution_failed",
					"task_run_id", notification.TaskRunID,
					"recipient_session_id", notification.RecipientSessionID,
					"error", err,
				)
			}
		case <-ticker.C:
			if err := r.enqueueDurableWakes(ctx); err != nil {
				r.logger.Error("daemon.network_wake.scan_failed", "error", err)
			}
		}
	}
}

func (r *networkWakeRunner) enqueueDurableWakes(ctx context.Context) error {
	if !r.networkEnabled() {
		return nil
	}
	notifications, err := r.store.ListQueuedNetworkWakes(ctx, networkWakeScanLimit)
	if err != nil {
		return fmt.Errorf("daemon: list queued network wakes: %w", err)
	}
	for _, notification := range notifications {
		r.NotifyNetworkWake(notification)
	}
	return nil
}

func (r *networkWakeRunner) processNotification(
	ctx context.Context,
	actor taskpkg.ActorContext,
	notification store.CommittedNetworkNotification,
) error {
	if !r.networkEnabled() {
		return nil
	}
	targetSessionID := strings.TrimSpace(notification.RecipientSessionID)
	claim, err := r.tasks.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		RunID:            strings.TrimSpace(notification.TaskRunID),
		RunKind:          taskpkg.RunKindNetworkWake,
		TargetSessionID:  targetSessionID,
		ClaimerSessionID: targetSessionID,
		LeaseDuration:    taskpkg.DefaultRunLeaseDuration,
		Now:              r.now().UTC(),
	}, actor)
	if errors.Is(err, taskpkg.ErrNoClaimableRun) || errors.Is(err, taskpkg.ErrActiveRunLease) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("daemon: claim network wake: %w", err)
	}
	if claim == nil {
		return errors.New("daemon: network wake claim violated run-kind fence")
	}
	if !claim.Run.IsNetworkWake() || claim.Task != nil {
		claimErr := errors.New("daemon: network wake claim violated run-kind fence")
		if releaseErr := r.releaseClaimedWake(
			ctx,
			actor,
			claim,
			"network wake run-kind fence violation",
		); releaseErr != nil {
			return errors.Join(claimErr, releaseErr)
		}
		return claimErr
	}
	wakeID, claimedTarget, ownerKey := claim.Run.NetworkWakeCorrelation()
	if claimedTarget != targetSessionID {
		return r.failClaimedWake(ctx, actor, claim, store.WakeReservation{
			WakeID:    wakeID,
			TaskRunID: claim.Run.ID,
			OwnerKey:  ownerKey,
		}, "network wake target mismatch")
	}
	reservation, messages, err := r.store.LoadNetworkWake(ctx, wakeID)
	if err != nil {
		return r.failClaimedWake(
			ctx,
			actor,
			claim,
			store.WakeReservation{WakeID: wakeID, TaskRunID: claim.Run.ID, OwnerKey: ownerKey},
			"network wake evidence unavailable",
		)
	}
	return r.executeClaimedWake(ctx, actor, claim, reservation, messages, targetSessionID)
}

func (r *networkWakeRunner) executeClaimedWake(
	ctx context.Context,
	actor taskpkg.ActorContext,
	claim *taskpkg.ClaimResult,
	reservation store.WakeReservation,
	messages []store.NetworkMessageEntry,
	targetSessionID string,
) error {
	wallLimit, err := time.ParseDuration(reservation.ReservedWallTime)
	if err != nil || wallLimit <= 0 {
		return r.failClaimedWake(ctx, actor, claim, reservation, "network wake deadline is invalid")
	}
	prompt, meta, err := network.FormatNetworkWakePrompt(messages, targetSessionID)
	if err != nil {
		return r.failClaimedWake(ctx, actor, claim, reservation, "network wake prompt is invalid")
	}
	turnCtx, cancel := context.WithTimeout(ctx, wallLimit)
	r.registerActive(claim.Run.ID, cancel)
	defer func() {
		cancel()
		r.unregisterActive(claim.Run.ID)
	}()
	startedAt := r.now().UTC()
	heartbeatDone := r.startHeartbeat(turnCtx, cancel, actor, claim)
	events, promptErr := r.prompter.PromptNetwork(turnCtx, targetSessionID, prompt, meta)
	outcome := r.collectWakeOutcome(turnCtx, events, promptErr, startedAt)
	if turnCtx.Err() != nil {
		if cancelErr := r.cancelProviderPrompt(ctx, targetSessionID); cancelErr != nil {
			outcome.State = store.NetworkWakeStateFailed
			outcome.Reason = "network wake provider cancellation failed"
			r.logger.Error(
				"daemon.network_wake.provider_cancel_failed",
				"task_run_id", claim.Run.ID,
				"recipient_session_id", targetSessionID,
				"error", cancelErr,
			)
		}
	}
	cancel()
	heartbeatErr := <-heartbeatDone
	if heartbeatErr != nil {
		outcome.State = store.NetworkWakeStateFailed
		outcome.Reason = "network wake lease heartbeat failed"
	}
	return r.terminalizeWake(ctx, actor, claim, reservation, outcome)
}

func (r *networkWakeRunner) cancelProviderPrompt(ctx context.Context, targetSessionID string) error {
	cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
	defer cancel()
	if err := r.prompter.CancelPrompt(cancelCtx, targetSessionID); err != nil {
		return fmt.Errorf("daemon: cancel network wake provider prompt: %w", err)
	}
	return nil
}

func (r *networkWakeRunner) startHeartbeat(
	ctx context.Context,
	cancel context.CancelFunc,
	actor taskpkg.ActorContext,
	claim *taskpkg.ClaimResult,
) <-chan error {
	done := make(chan error, 1)
	go func() {
		interval := taskpkg.DefaultRunLeaseDuration / 3
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				done <- nil
				return
			case <-ticker.C:
				_, err := r.tasks.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
					RunID:         claim.Run.ID,
					ClaimToken:    claim.ClaimToken,
					LeaseDuration: taskpkg.DefaultRunLeaseDuration,
					Now:           r.now().UTC(),
				}, actor)
				if err != nil {
					cancel()
					done <- fmt.Errorf("daemon: heartbeat network wake lease: %w", err)
					return
				}
			}
		}
	}()
	return done
}

func (r *networkWakeRunner) collectWakeOutcome(
	ctx context.Context,
	events <-chan acp.AgentEvent,
	promptErr error,
	startedAt time.Time,
) store.NetworkWakeOutcome {
	outcome := store.NetworkWakeOutcome{State: store.NetworkWakeStateFailed}
	if promptErr != nil {
		outcome.Reason = "network wake prompt failed"
		return r.finishWakeUsage(outcome, acp.TokenUsage{}, startedAt)
	}
	var usage acp.TokenUsage
	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				outcome.State = store.NetworkWakeStateDeadlineExceeded
				outcome.Reason = "network wake deadline exceeded"
			} else {
				outcome.State = store.NetworkWakeStateCanceled
				outcome.Reason = "network wake canceled"
			}
			return r.finishWakeUsage(outcome, usage, startedAt)
		case event, ok := <-events:
			if !ok {
				outcome.Reason = "network wake event stream closed without terminal event"
				return r.finishWakeUsage(outcome, usage, startedAt)
			}
			if event.Usage != nil {
				usage = usage.Merge(*event.Usage)
			}
			switch event.Type {
			case acp.EventTypeError:
				outcome.Reason = "network wake prompt failed"
				return r.finishWakeUsage(outcome, usage, startedAt)
			case acp.EventTypeDone:
				if event.PromptStopReason == acp.PromptStopReasonCancelled {
					outcome.State = store.NetworkWakeStateCanceled
					outcome.Reason = "network wake canceled"
				} else {
					outcome.State = store.NetworkWakeStateSucceeded
				}
				return r.finishWakeUsage(outcome, usage, startedAt)
			}
		}
	}
}

func (r *networkWakeRunner) finishWakeUsage(
	outcome store.NetworkWakeOutcome,
	usage acp.TokenUsage,
	startedAt time.Time,
) store.NetworkWakeOutcome {
	outcome.ActualWallTime = max(r.now().UTC().Sub(startedAt), 0)
	if usage.InputTokens == nil || usage.OutputTokens == nil {
		outcome.UsageState = store.NetworkWakeUsageUnavailable
		return outcome
	}
	outcome.UsageState = store.NetworkWakeUsageActual
	outcome.ActualInputTokens = max(*usage.InputTokens, 0)
	outcome.ActualOutputTokens = max(*usage.OutputTokens, 0)
	return outcome
}

func (r *networkWakeRunner) networkEnabled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enabled
}

func (r *networkWakeRunner) registerActive(runID string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[strings.TrimSpace(runID)] = cancel
}

func (r *networkWakeRunner) unregisterActive(runID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, strings.TrimSpace(runID))
}

func (r *networkWakeRunner) cancelActive() {
	r.SetNetworkEnabled(false)
}
