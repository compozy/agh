package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	loopActionRuntimeActorRef       = "loop-action"
	loopActionRuntimeOriginRef      = "daemon.loop-action"
	loopActionHeartbeatMaxInterval  = 5 * time.Minute
	loopActionRuntimeReasonEnqueued = "task_run_enqueued"
	loopActionRuntimeReasonRecover  = "recovery"
)

type loopActionRuntime struct {
	manager *taskpkg.Service
	store   taskStore
	runner  *looppkg.CoordinatorRunner
	logger  *slog.Logger
	now     func() time.Time

	root              context.Context
	cancel            context.CancelFunc
	sem               chan struct{}
	spawnMu           sync.Mutex
	wg                sync.WaitGroup
	stopping          atomic.Bool
	heartbeatInterval func(time.Duration) time.Duration
}

var _ taskRunEnqueuedObserver = (*loopActionRuntime)(nil)

func newLoopActionRuntime(
	manager *taskpkg.Service,
	store taskStore,
	runner *looppkg.CoordinatorRunner,
	logger *slog.Logger,
	now func() time.Time,
) (*loopActionRuntime, error) {
	if manager == nil {
		return nil, errors.New("daemon: loop action runtime requires task manager")
	}
	if store == nil {
		return nil, errors.New("daemon: loop action runtime requires task store")
	}
	if runner == nil {
		return nil, errors.New("daemon: loop action runtime requires coordinator runner")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	root, cancel := context.WithCancel(context.Background())
	return &loopActionRuntime{
		manager:           manager,
		store:             store,
		runner:            runner,
		logger:            logger,
		now:               now,
		root:              root,
		cancel:            cancel,
		sem:               make(chan struct{}, looppkg.LoopMaxFanoutWidth),
		heartbeatInterval: loopActionHeartbeatInterval,
	}, nil
}

func (r *loopActionRuntime) OnTaskRunEnqueued(
	ctx context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = context.WithoutCancel(ctx)
	}
	runID := strings.TrimSpace(payload.RunID)
	if runID == "" {
		r.logError("daemon: loop action enqueue payload missing run id", nil, payload)
		return
	}
	run, err := r.store.GetTaskRun(ctx, runID)
	if err != nil {
		r.logError("daemon: load loop action run", err, payload)
		return
	}
	taskRecord, err := r.store.GetTask(ctx, run.TaskID)
	if err != nil {
		r.logError("daemon: load loop action task", err, payload)
		return
	}
	r.startQueuedRun(taskRecord, run, loopActionRuntimeReasonEnqueued, payload)
}

func (r *loopActionRuntime) Recover(ctx context.Context) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runs, err := r.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		r.logError("daemon: list queued loop action runs", err, hookspkg.TaskRunEnqueuedPayload{})
		return
	}
	for _, run := range runs {
		if !isQueuedLoopActionRun(run) {
			continue
		}
		taskRecord, err := r.store.GetTask(ctx, run.TaskID)
		if err != nil {
			r.logError("daemon: load loop action recovery task", err, loopActionPayload(run))
			continue
		}
		r.startQueuedRun(taskRecord, run, loopActionRuntimeReasonRecover, loopActionPayload(run))
	}
}

func (r *loopActionRuntime) startQueuedRun(
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil {
		return
	}
	r.spawnMu.Lock()
	defer r.spawnMu.Unlock()
	if r.stopping.Load() {
		return
	}
	r.wg.Go(func() {
		if err := r.executeStartedRun(r.root, taskRecord, run, reason); err != nil &&
			!errors.Is(err, context.Canceled) {
			r.logError("daemon: execute loop action run", err, payload)
		}
	})
}

func (r *loopActionRuntime) executeStartedRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
) error {
	if err := r.acquireSlot(ctx); err != nil {
		return err
	}
	defer r.releaseSlot()
	return r.executeQueuedRun(ctx, taskRecord, run, reason)
}

func (r *loopActionRuntime) acquireSlot(ctx context.Context) error {
	select {
	case r.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *loopActionRuntime) releaseSlot() {
	<-r.sem
}

func (r *loopActionRuntime) executeQueuedRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
) error {
	if !isQueuedLoopActionRun(run) {
		return nil
	}
	actor, err := taskpkg.DeriveDaemonActorContext(
		loopActionRuntimeActorRef,
		loopActionRuntimeOriginRef,
	)
	if err != nil {
		return err
	}
	leaseDuration, err := r.leaseDurationForRun(ctx, run)
	if err != nil {
		return err
	}
	claim, err := r.manager.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		RunID:            strings.TrimSpace(run.ID),
		Scope:            taskRecord.Scope.Normalize(),
		WorkspaceID:      strings.TrimSpace(taskRecord.WorkspaceID),
		RunKind:          taskpkg.RunKindWorker,
		ClaimerSessionID: loopActionClaimerSessionID(run.ID),
		ClaimedBy: &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKindDaemon,
			Ref:  loopActionRuntimeActorRef,
		},
		LeaseDuration: leaseDuration,
		Now:           r.now().UTC(),
	}, actor)
	if err != nil {
		if errors.Is(err, taskpkg.ErrNoClaimableRun) {
			return nil
		}
		return err
	}
	result, err := r.executeClaimedRun(ctx, claim, actor, leaseDuration)
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		return r.failClaimedRun(ctx, claim, actor, reason, result.TokensUsed, err)
	}
	_, err = r.manager.CompleteRunLease(context.WithoutCancel(ctx), taskpkg.LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     result,
		TokensUsed: result.TokensUsed,
		Now:        r.now().UTC(),
	}, actor)
	return err
}

func (r *loopActionRuntime) executeClaimedRun(
	ctx context.Context,
	claim *taskpkg.ClaimResult,
	actor taskpkg.ActorContext,
	leaseDuration time.Duration,
) (taskpkg.RunResult, error) {
	runCtx, cancelRun := context.WithCancel(ctx)
	usage := &loopActionUsageState{}
	runCtx = looppkg.ContextWithActionUsageReporter(runCtx, usage)
	heartbeatErrC := r.startHeartbeat(runCtx, cancelRun, claim, actor, leaseDuration, usage)
	result, runErr := r.runner.ExecuteActionRun(runCtx, claim.Run, actor)
	cancelRun()
	heartbeatErr := <-heartbeatErrC
	if ctx.Err() != nil {
		return taskpkg.RunResult{}, ctx.Err()
	}
	if tokensUsed := usage.TokensUsed(); tokensUsed > result.TokensUsed {
		result.TokensUsed = tokensUsed
	}
	return result, errors.Join(runErr, heartbeatErr)
}

func (r *loopActionRuntime) startHeartbeat(
	ctx context.Context,
	cancelRun context.CancelFunc,
	claim *taskpkg.ClaimResult,
	actor taskpkg.ActorContext,
	leaseDuration time.Duration,
	usage *loopActionUsageState,
) <-chan error {
	errC := make(chan error, 1)
	go func() {
		err := r.heartbeatClaim(ctx, claim, actor, leaseDuration, usage)
		if err != nil {
			cancelRun()
		}
		errC <- err
	}()
	return errC
}

func (r *loopActionRuntime) heartbeatClaim(
	ctx context.Context,
	claim *taskpkg.ClaimResult,
	actor taskpkg.ActorContext,
	leaseDuration time.Duration,
	usage *loopActionUsageState,
) error {
	interval := r.heartbeatInterval(leaseDuration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_, err := r.manager.HeartbeatRunLease(ctx, taskpkg.LeaseHeartbeat{
				RunID:         claim.Run.ID,
				ClaimToken:    claim.ClaimToken,
				LeaseDuration: leaseDuration,
				Now:           r.now().UTC(),
				TokensUsed:    usage.TokensUsed(),
			}, actor)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (r *loopActionRuntime) leaseDurationForRun(
	ctx context.Context,
	run taskpkg.Run,
) (time.Duration, error) {
	leaseDuration := taskpkg.DefaultRunLeaseDuration
	timeout, ok, err := r.runner.ActionRunTimeout(ctx, run)
	if err != nil {
		if errors.Is(err, looppkg.ErrValidation) {
			return leaseDuration, nil
		}
		return 0, err
	}
	if ok && timeout > leaseDuration {
		leaseDuration = timeout
	}
	if leaseDuration > taskpkg.MaxRunLeaseDuration {
		return taskpkg.MaxRunLeaseDuration, nil
	}
	return leaseDuration, nil
}

func loopActionHeartbeatInterval(leaseDuration time.Duration) time.Duration {
	interval := leaseDuration / 3
	if interval <= 0 {
		return time.Second
	}
	if interval > loopActionHeartbeatMaxInterval {
		return loopActionHeartbeatMaxInterval
	}
	return interval
}

func (r *loopActionRuntime) failClaimedRun(
	ctx context.Context,
	claim *taskpkg.ClaimResult,
	actor taskpkg.ActorContext,
	reason string,
	tokensUsed int64,
	cause error,
) error {
	if claim == nil {
		return cause
	}
	metadata, err := json.Marshal(map[string]string{
		"reason_code": "loop_action_failed",
		"reason":      strings.TrimSpace(reason),
	})
	if err != nil {
		return errors.Join(cause, err)
	}
	_, failErr := r.manager.FailRunLease(context.WithoutCancel(ctx), taskpkg.LeaseFailure{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Failure: taskpkg.RunFailure{
			Error:    cause.Error(),
			Metadata: metadata,
		},
		TokensUsed: tokensUsed,
		Now:        r.now().UTC(),
	}, actor)
	return errors.Join(cause, failErr)
}

type loopActionUsageState struct {
	tokensUsed atomic.Int64
}

func (s *loopActionUsageState) ReportActionTokensUsed(tokensUsed int64) {
	if s == nil || tokensUsed <= 0 {
		return
	}
	for {
		current := s.tokensUsed.Load()
		if tokensUsed <= current {
			return
		}
		if s.tokensUsed.CompareAndSwap(current, tokensUsed) {
			return
		}
	}
}

func (s *loopActionUsageState) TokensUsed() int64 {
	if s == nil {
		return 0
	}
	return s.tokensUsed.Load()
}

func (r *loopActionRuntime) shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.spawnMu.Lock()
	if r.stopping.CompareAndSwap(false, true) && r.cancel != nil {
		r.cancel()
	}
	r.spawnMu.Unlock()
	return r.wait(ctx)
}

func (r *loopActionRuntime) wait(ctx context.Context) error {
	if r == nil {
		return nil
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
		return fmt.Errorf("daemon: wait for loop action runtime: %w", ctx.Err())
	}
}

func isQueuedLoopActionRun(run taskpkg.Run) bool {
	return run.Status.Normalize() == taskpkg.TaskRunStatusQueued &&
		run.RunKind.Normalize() == taskpkg.RunKindWorker &&
		strings.TrimSpace(run.LoopRunID) != ""
}

func loopActionClaimerSessionID(runID string) string {
	return loopActionRuntimeActorRef + ":" + strings.TrimSpace(runID)
}

func loopActionPayload(run taskpkg.Run) hookspkg.TaskRunEnqueuedPayload {
	return hookspkg.TaskRunEnqueuedPayload{
		TaskRunContext: hookspkg.TaskRunContext{
			TaskID: run.TaskID,
			RunID:  run.ID,
		},
	}
}

func (r *loopActionRuntime) logError(
	msg string,
	err error,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil || r.logger == nil {
		return
	}
	args := []any{
		"task_id", strings.TrimSpace(payload.TaskID),
		daemonLogRunIDKey, strings.TrimSpace(payload.RunID),
	}
	if err != nil {
		args = append(args, "error", err)
	}
	r.logger.Error(msg, args...)
}

func (r *loopActionRuntime) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s:%p", loopActionRuntimeActorRef, r)
}
