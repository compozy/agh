package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

var (
	// ErrConcurrencyLimitReached reports that the shared automation gate rejected a new run.
	ErrConcurrencyLimitReached = errors.New("automation: global concurrency limit reached")
	// ErrFireLimitReached reports that a definition exceeded its rolling fire-limit window.
	ErrFireLimitReached = errors.New("automation: fire limit reached")
)

const dispatcherSessionStopTimeout = 2 * time.Second

// DispatchKind identifies which activation path produced a dispatch request.
type DispatchKind string

const (
	// DispatchKindSchedule identifies time-based schedule execution.
	DispatchKindSchedule DispatchKind = "schedule"
	// DispatchKindTrigger identifies event-driven trigger execution.
	DispatchKindTrigger DispatchKind = "trigger"
	// DispatchKindManual identifies explicit user-initiated job execution.
	DispatchKindManual DispatchKind = "manual"
	// DispatchKindExtension identifies extension-fired automation execution.
	DispatchKindExtension DispatchKind = "extension"
)

// Validate ensures the dispatch kind is one of the supported activation paths.
func (k DispatchKind) Validate(path string) error {
	switch k {
	case DispatchKindSchedule, DispatchKindTrigger, DispatchKindManual, DispatchKindExtension:
		return nil
	default:
		return fmt.Errorf(
			"%s must be one of %q, %q, %q, or %q: %q",
			path,
			DispatchKindSchedule,
			DispatchKindTrigger,
			DispatchKindManual,
			DispatchKindExtension,
			k,
		)
	}
}

// DispatchRequest describes one normalized automation execution attempt.
//
// Exactly one of Job or Trigger must be provided. Triggers also require an
// activation envelope so prompt templates can render against the normalized
// trigger payload. Prompt allows later callers to inject a pre-render override
// after pre-fire hooks patch the outbound prompt.
type DispatchRequest struct {
	Kind     DispatchKind        `json:"kind"`
	Job      *Job                `json:"job,omitempty"`
	Trigger  *Trigger            `json:"trigger,omitempty"`
	Envelope *ActivationEnvelope `json:"envelope,omitempty"`
	Prompt   string              `json:"prompt,omitempty"`
}

// Validate ensures the request can be executed by the shared dispatcher.
func (r DispatchRequest) Validate(path string) error {
	if err := r.Kind.Validate(nestedPath(path, "kind")); err != nil {
		return err
	}

	hasJob := r.Job != nil
	hasTrigger := r.Trigger != nil
	switch {
	case hasJob && hasTrigger:
		return errors.New(path + " must not define both job and trigger")
	case !hasJob && !hasTrigger:
		return errors.New(path + " must define either job or trigger")
	}

	if hasJob {
		if err := r.Job.Validate(nestedPath(path, "job")); err != nil {
			return err
		}
		return nil
	}

	if err := r.Trigger.Validate(nestedPath(path, "trigger")); err != nil {
		return err
	}
	if r.Envelope == nil {
		return errors.New(nestedPath(path, "envelope") + " is required for trigger dispatch")
	}
	if err := r.Envelope.Validate(nestedPath(path, "envelope")); err != nil {
		return err
	}
	if got, want := strings.TrimSpace(r.Envelope.Kind), strings.TrimSpace(r.Trigger.Event); got != want {
		return fmt.Errorf(
			"%s.kind must match %s.event: %q != %q",
			nestedPath(path, "envelope"),
			nestedPath(path, "trigger"),
			got,
			want,
		)
	}
	if got, want := r.Envelope.Scope, r.Trigger.Scope; got != want {
		return fmt.Errorf(
			"%s.scope must match %s.scope: %q != %q",
			nestedPath(path, "envelope"),
			nestedPath(path, "trigger"),
			got,
			want,
		)
	}
	if got, want := strings.TrimSpace(r.Envelope.WorkspaceID), strings.TrimSpace(r.Trigger.WorkspaceID); got != want {
		return fmt.Errorf(
			"%s.workspace_id must match %s.workspace_id: %q != %q",
			nestedPath(path, "envelope"),
			nestedPath(path, "trigger"),
			got,
			want,
		)
	}

	return nil
}

// SessionCreator is the subset of session.Manager needed by the dispatcher.
type SessionCreator interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

// RunStore persists automation run state and restart-safe fire-limit inputs.
type RunStore interface {
	CreateRun(ctx context.Context, run Run) (Run, error)
	UpdateRun(ctx context.Context, run Run) (Run, error)
	CountRuns(ctx context.Context, query RunQuery) (int64, error)
}

// TaskService exposes the minimal task-domain surface used by task-backed
// automation jobs.
type TaskService interface {
	CreateTask(ctx context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error)
	EnqueueRun(ctx context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.Run, error)
}

// AutomationSessionTaskActorRecorder stores trusted task-domain provenance for
// automation-launched sessions that may later create tasks explicitly.
type SessionTaskActorRecorder interface {
	RecordAutomationSessionTaskActor(sessionID string, actor taskpkg.ActorContext) error
	DeleteAutomationSessionTaskActor(sessionID string)
}

// HookDispatcher emits automation lifecycle hooks around shared dispatch.
type HookDispatcher interface {
	DispatchAutomationJobPreFire(
		ctx context.Context,
		payload hookspkg.AutomationJobPreFirePayload,
	) (hookspkg.AutomationJobPreFirePayload, error)
	DispatchAutomationJobPostFire(
		ctx context.Context,
		payload hookspkg.AutomationJobPostFirePayload,
	) (hookspkg.AutomationJobPostFirePayload, error)
	DispatchAutomationTriggerPreFire(
		ctx context.Context,
		payload hookspkg.AutomationTriggerPreFirePayload,
	) (hookspkg.AutomationTriggerPreFirePayload, error)
	DispatchAutomationTriggerPostFire(
		ctx context.Context,
		payload hookspkg.AutomationTriggerPostFirePayload,
	) (hookspkg.AutomationTriggerPostFirePayload, error)
	DispatchAutomationRunCompleted(
		ctx context.Context,
		payload hookspkg.AutomationRunCompletedPayload,
	) (hookspkg.AutomationRunCompletedPayload, error)
	DispatchAutomationRunFailed(
		ctx context.Context,
		payload hookspkg.AutomationRunFailedPayload,
	) (hookspkg.AutomationRunFailedPayload, error)
}

// SleepFunc waits for retry backoff with context cancellation support.
type SleepFunc func(ctx context.Context, delay time.Duration) error

// DispatcherOption customizes shared automation dispatch behavior.
type DispatcherOption func(*Dispatcher)

// Dispatcher routes every automation activation through one execution path.
type Dispatcher struct {
	sessions SessionCreator
	runs     RunStore
	tasks    TaskService

	logger              *slog.Logger
	now                 func() time.Time
	sleep               SleepFunc
	globalWorkspacePath string
	maxConcurrent       int
	hooks               HookDispatcher
	taskActors          SessionTaskActorRecorder

	fireLimitMu sync.Mutex
	gate        chan struct{}
}

// NewDispatcher constructs a shared automation dispatcher.
func NewDispatcher(sessions SessionCreator, runs RunStore, opts ...DispatcherOption) (*Dispatcher, error) {
	if sessions == nil {
		return nil, errors.New("automation: session creator is required")
	}
	if runs == nil {
		return nil, errors.New("automation: run store is required")
	}

	dispatcher := &Dispatcher{
		sessions:      sessions,
		runs:          runs,
		logger:        slog.Default(),
		now:           func() time.Time { return time.Now().UTC() },
		sleep:         sleepWithContext,
		maxConcurrent: DefaultMaxConcurrentJobs,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(dispatcher)
		}
	}

	if dispatcher.logger == nil {
		dispatcher.logger = slog.Default()
	}
	if dispatcher.now == nil {
		dispatcher.now = func() time.Time { return time.Now().UTC() }
	}
	if dispatcher.sleep == nil {
		dispatcher.sleep = sleepWithContext
	}
	if strings.TrimSpace(dispatcher.globalWorkspacePath) == "" {
		return nil, errors.New("automation: global workspace path is required")
	}
	if dispatcher.maxConcurrent <= 0 {
		dispatcher.maxConcurrent = DefaultMaxConcurrentJobs
	}
	dispatcher.gate = make(chan struct{}, dispatcher.maxConcurrent)

	return dispatcher, nil
}

// WithDispatcherLogger overrides the dispatcher logger.
func WithDispatcherLogger(logger *slog.Logger) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.logger = logger
	}
}

// WithDispatcherNow overrides the dispatcher clock.
func WithDispatcherNow(now func() time.Time) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.now = now
	}
}

// WithDispatcherSleep overrides retry waiting, mainly for tests.
func WithDispatcherSleep(sleep SleepFunc) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.sleep = sleep
	}
}

// WithDispatcherGlobalWorkspacePath overrides the fallback path used for global automations.
func WithDispatcherGlobalWorkspacePath(path string) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.globalWorkspacePath = strings.TrimSpace(path)
	}
}

// WithDispatcherMaxConcurrent overrides the shared automation concurrency gate.
func WithDispatcherMaxConcurrent(limit int) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.maxConcurrent = limit
	}
}

// WithDispatcherHooks injects the automation lifecycle hook dispatcher.
func WithDispatcherHooks(hooks HookDispatcher) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.hooks = hooks
	}
}

// WithDispatcherTasks injects the task-domain service used for direct
// task-backed automation jobs.
func WithDispatcherTasks(tasks TaskService) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.tasks = tasks
	}
}

// WithDispatcherTaskActorRecorder injects the session provenance recorder used
// to support automation-linked agent task creation.
func WithDispatcherTaskActorRecorder(recorder SessionTaskActorRecorder) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.taskActors = recorder
	}
}

// Dispatch executes one automation request through the shared governance path.
func (d *Dispatcher) Dispatch(ctx context.Context, req DispatchRequest) (*Run, error) {
	if ctx == nil {
		return nil, errors.New("automation: dispatch context is required")
	}
	if err := req.Validate("dispatch"); err != nil {
		return nil, err
	}

	attempt := 1
	var lastRun *Run
	for {
		run, err := d.dispatchAttempt(ctx, req, attempt)
		if run != nil {
			lastRun = cloneRun(run)
		}
		if run != nil && d.hooks != nil {
			willRetry := err != nil && shouldRetry(req.retryConfig(), run, attempt, err)
			d.emitRunLifecycleHooks(ctx, req, *run, err, willRetry)
		}
		if err == nil {
			return lastRun, nil
		}
		if !shouldRetry(req.retryConfig(), run, attempt, err) {
			return lastRun, err
		}

		delay, delayErr := retryDelay(req.retryConfig(), attempt)
		if delayErr != nil {
			return lastRun, errors.Join(err, delayErr)
		}
		nextAttempt := attempt + 1
		d.logger.Info(
			"automation.dispatch.retry_scheduled",
			"run_id", run.ID,
			"job_id", strings.TrimSpace(run.JobID),
			"trigger_id", strings.TrimSpace(run.TriggerID),
			"attempt", nextAttempt,
			"delay", delay.String(),
		)
		if sleepErr := d.sleep(ctx, delay); sleepErr != nil {
			return lastRun, errors.Join(err, sleepErr)
		}
		attempt = nextAttempt
	}
}

func (d *Dispatcher) dispatchAttempt(ctx context.Context, req DispatchRequest, attempt int) (*Run, error) {
	if !d.tryAcquire() {
		return nil, fmt.Errorf(
			"%w: active=%d limit=%d",
			ErrConcurrencyLimitReached,
			len(d.gate),
			cap(d.gate),
		)
	}
	defer d.release()

	scheduledRun, err := d.reserveRun(ctx, req, attempt)
	if err != nil {
		return nil, err
	}
	if req.Job != nil && req.Job.Task != nil {
		return d.dispatchTaskBackedAttempt(ctx, req, scheduledRun, attempt)
	}

	prompt, promptErr := req.prompt()
	if promptErr != nil {
		return d.finishRun(ctx, scheduledRun, RunFailed, promptErr)
	}
	prompt, canceled, hookErr := d.dispatchPreFireHook(ctx, req, prompt, attempt)
	if hookErr != nil {
		return d.finishRun(ctx, scheduledRun, RunFailed, hookErr)
	}
	if canceled {
		return d.finishRun(ctx, scheduledRun, RunCancelled, nil)
	}

	createOpts := d.createOpts(req)
	createdSession, createErr := d.sessions.Create(ctx, createOpts)
	if createErr != nil {
		return d.finishRun(ctx, scheduledRun, classifyDispatchError(createErr), createErr)
	}
	if createdSession == nil || strings.TrimSpace(createdSession.ID) == "" {
		return d.finishRun(
			ctx,
			scheduledRun,
			RunFailed,
			errors.New("automation: session creator returned empty session"),
		)
	}

	runningRun, err := d.transitionRun(ctx, scheduledRun, func(run *Run, _ time.Time) {
		run.Status = RunRunning
		run.SessionID = strings.TrimSpace(createdSession.ID)
	})
	if err != nil {
		return cloneRun(scheduledRun), err
	}
	if err := d.recordAutomationSessionTaskActor(createdSession.ID, runningRun); err != nil {
		return d.finishRunAfterSessionStop(ctx, runningRun, createdSession.ID, RunFailed, err)
	}

	events, promptErr := d.sessions.Prompt(ctx, createdSession.ID, prompt)
	if promptErr != nil {
		return d.finishRunAfterSessionStop(
			ctx,
			runningRun,
			createdSession.ID,
			classifyDispatchError(promptErr),
			promptErr,
		)
	}
	d.dispatchPostFireHook(ctx, req, *runningRun)

	runErr := collectPromptError(ctx, events)
	if runErr != nil {
		return d.finishRunAfterSessionStop(ctx, runningRun, createdSession.ID, classifyDispatchError(runErr), runErr)
	}

	return d.finishRunAfterSessionStop(ctx, runningRun, createdSession.ID, RunCompleted, nil)
}

func (d *Dispatcher) reserveRun(ctx context.Context, req DispatchRequest, attempt int) (*Run, error) {
	fireLimit := req.fireLimitConfig()
	window, err := time.ParseDuration(fireLimit.Window)
	if err != nil {
		return nil, fmt.Errorf("automation: parse fire-limit window: %w", err)
	}

	now := d.now()
	query := RunQuery{
		Since: now.Add(-window),
		Until: now,
	}
	if req.Job != nil {
		query.JobID = req.Job.ID
	} else {
		query.TriggerID = req.Trigger.ID
	}

	d.fireLimitMu.Lock()
	defer d.fireLimitMu.Unlock()

	count, err := d.runs.CountRuns(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("automation: evaluate fire limit: %w", err)
	}
	if count >= int64(fireLimit.Max) {
		return nil, fmt.Errorf(
			"%w: fires=%d limit=%d window=%s",
			ErrFireLimitReached,
			count,
			fireLimit.Max,
			window.String(),
		)
	}

	run := Run{
		Status:    RunScheduled,
		Attempt:   attempt,
		StartedAt: timePointer(now),
	}
	if req.Job != nil {
		run.JobID = req.Job.ID
	} else {
		run.TriggerID = req.Trigger.ID
	}

	created, err := d.runs.CreateRun(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("automation: create scheduled run: %w", err)
	}
	return &created, nil
}

func (d *Dispatcher) dispatchTaskBackedAttempt(
	ctx context.Context,
	req DispatchRequest,
	scheduledRun *Run,
	attempt int,
) (*Run, error) {
	if d.tasks == nil {
		return d.finishRun(
			ctx,
			scheduledRun,
			RunFailed,
			errors.New("automation: task-backed job requires task service"),
		)
	}

	preFirePrompt := strings.TrimSpace(req.Prompt)
	if preFirePrompt == "" && req.Job != nil {
		preFirePrompt = strings.TrimSpace(req.Job.Prompt)
	}
	preFirePrompt, canceled, hookErr := d.dispatchPreFireHook(ctx, req, preFirePrompt, attempt)
	if hookErr != nil {
		return d.finishRun(ctx, scheduledRun, RunFailed, hookErr)
	}
	if canceled {
		return d.finishRun(ctx, scheduledRun, RunCancelled, nil)
	}

	actor, err := directTaskActorContext(req.Job, scheduledRun.ID)
	if err != nil {
		return d.finishRun(ctx, scheduledRun, RunFailed, err)
	}

	taskRecord, err := d.tasks.CreateTask(ctx, directTaskSpec(req.Job, preFirePrompt), actor)
	if err != nil {
		return d.finishRun(ctx, scheduledRun, classifyDispatchError(err), err)
	}
	if taskRecord == nil || strings.TrimSpace(taskRecord.ID) == "" {
		return d.finishRun(ctx, scheduledRun, RunFailed, errors.New("automation: task service returned empty task"))
	}

	taskRun, err := d.tasks.EnqueueRun(ctx, taskpkg.EnqueueRun{
		TaskID:         taskRecord.ID,
		IdempotencyKey: automationTaskRunIdempotencyKey(scheduledRun.ID),
		NetworkChannel: strings.TrimSpace(taskRecord.NetworkChannel),
	}, actor)
	if err != nil {
		return d.finishRun(ctx, scheduledRun, classifyDispatchError(err), err)
	}
	if taskRun == nil || strings.TrimSpace(taskRun.ID) == "" {
		return d.finishRun(ctx, scheduledRun, RunFailed, errors.New("automation: task service returned empty task run"))
	}

	delegatedRun, err := d.delegateRun(ctx, scheduledRun, taskRecord.ID, taskRun.ID)
	if err != nil {
		return delegatedRun, err
	}
	d.dispatchPostFireHook(ctx, req, *delegatedRun)
	return delegatedRun, nil
}

func (d *Dispatcher) transitionRun(
	ctx context.Context,
	current *Run,
	mutate func(run *Run, now time.Time),
) (*Run, error) {
	if current == nil {
		return nil, errors.New("automation: run is required")
	}

	next := *current
	mutate(&next, d.now())

	updated, err := d.runs.UpdateRun(persistenceContext(ctx), next)
	if err != nil {
		return cloneRun(current), fmt.Errorf("automation: update run %q: %w", current.ID, err)
	}
	return &updated, nil
}

func (d *Dispatcher) delegateRun(ctx context.Context, current *Run, taskID string, taskRunID string) (*Run, error) {
	updatedRun, updateErr := d.transitionRun(ctx, current, func(run *Run, now time.Time) {
		run.TaskID = strings.TrimSpace(taskID)
		run.TaskRunID = strings.TrimSpace(taskRunID)
		run.Status = RunDelegated
		run.EndedAt = timePointer(now)
		run.Error = ""
	})
	if updateErr != nil {
		return updatedRun, updateErr
	}

	d.logger.Info(
		"automation.dispatch.delegated",
		"run_id", updatedRun.ID,
		"job_id", strings.TrimSpace(updatedRun.JobID),
		"trigger_id", strings.TrimSpace(updatedRun.TriggerID),
		"task_id", strings.TrimSpace(updatedRun.TaskID),
		"task_run_id", strings.TrimSpace(updatedRun.TaskRunID),
		"attempt", updatedRun.Attempt,
	)
	return updatedRun, nil
}

func (d *Dispatcher) finishRun(ctx context.Context, current *Run, status RunStatus, runErr error) (*Run, error) {
	updatedRun, updateErr := d.transitionRun(ctx, current, func(run *Run, now time.Time) {
		run.Status = status
		run.EndedAt = timePointer(now)
		if runErr != nil {
			run.Error = runErr.Error()
			return
		}
		run.Error = ""
	})
	if updateErr != nil {
		if runErr == nil {
			return updatedRun, updateErr
		}
		return updatedRun, errors.Join(runErr, updateErr)
	}

	if runErr == nil && status == RunCompleted {
		d.logger.Info(
			"automation.dispatch.completed",
			"run_id", updatedRun.ID,
			"job_id", strings.TrimSpace(updatedRun.JobID),
			"trigger_id", strings.TrimSpace(updatedRun.TriggerID),
			"session_id", strings.TrimSpace(updatedRun.SessionID),
			"attempt", updatedRun.Attempt,
		)
		return updatedRun, nil
	}
	if runErr == nil {
		d.logger.Info(
			"automation.dispatch.finished",
			"run_id", updatedRun.ID,
			"job_id", strings.TrimSpace(updatedRun.JobID),
			"trigger_id", strings.TrimSpace(updatedRun.TriggerID),
			"session_id", strings.TrimSpace(updatedRun.SessionID),
			"attempt", updatedRun.Attempt,
			"status", updatedRun.Status,
		)
		return updatedRun, nil
	}

	level := d.logger.Warn
	if status == RunCancelled {
		level = d.logger.Info
	}
	level(
		"automation.dispatch.failed",
		"run_id", updatedRun.ID,
		"job_id", strings.TrimSpace(updatedRun.JobID),
		"trigger_id", strings.TrimSpace(updatedRun.TriggerID),
		"session_id", strings.TrimSpace(updatedRun.SessionID),
		"attempt", updatedRun.Attempt,
		"status", updatedRun.Status,
		"error", runErr,
	)
	return updatedRun, runErr
}

func (d *Dispatcher) finishRunAfterSessionStop(
	ctx context.Context,
	current *Run,
	sessionID string,
	status RunStatus,
	runErr error,
) (*Run, error) {
	stopErr := d.stopAutomationSession(ctx, sessionID, status, runErr)
	if stopErr == nil {
		d.deleteAutomationSessionTaskActor(sessionID)
	}
	if stopErr != nil {
		wrappedStopErr := fmt.Errorf("automation: stop session %q: %w", strings.TrimSpace(sessionID), stopErr)
		if runErr == nil {
			status = RunFailed
			runErr = wrappedStopErr
		} else {
			runErr = errors.Join(runErr, wrappedStopErr)
		}
	}

	return d.finishRun(ctx, current, status, runErr)
}

func (d *Dispatcher) stopAutomationSession(
	ctx context.Context,
	sessionID string,
	status RunStatus,
	runErr error,
) error {
	trimmedSessionID := strings.TrimSpace(sessionID)
	if d == nil || trimmedSessionID == "" {
		return nil
	}

	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), dispatcherSessionStopTimeout)
	defer cancel()

	cause, detail := dispatchStopCause(status, runErr)
	return d.sessions.StopWithCause(stopCtx, trimmedSessionID, cause, detail)
}

func (d *Dispatcher) dispatchPreFireHook(
	ctx context.Context,
	req DispatchRequest,
	prompt string,
	attempt int,
) (string, bool, error) {
	if d == nil || d.hooks == nil {
		return prompt, false, nil
	}

	switch {
	case req.Job != nil:
		payload := hookspkg.AutomationJobPreFirePayload{
			JobID:       strings.TrimSpace(req.Job.ID),
			JobName:     strings.TrimSpace(req.Job.Name),
			AgentName:   strings.TrimSpace(req.Job.AgentName),
			WorkspaceID: strings.TrimSpace(req.Job.WorkspaceID),
			Prompt:      prompt,
			Schedule:    hookSchedulePayload(req.Job.Schedule),
			Attempt:     attempt,
		}
		next, err := d.hooks.DispatchAutomationJobPreFire(ctx, payload)
		if err != nil {
			if errors.Is(err, hookspkg.ErrAutomationFireCancelled) {
				return prompt, true, nil
			}
			return prompt, false, err
		}
		return strings.TrimSpace(next.Prompt), false, nil
	case req.Trigger != nil:
		payload := hookspkg.AutomationTriggerPreFirePayload{
			TriggerID:   strings.TrimSpace(req.Trigger.ID),
			TriggerName: strings.TrimSpace(req.Trigger.Name),
			Event:       strings.TrimSpace(req.Trigger.Event),
			AgentName:   strings.TrimSpace(req.Trigger.AgentName),
			WorkspaceID: strings.TrimSpace(req.Trigger.WorkspaceID),
			Prompt:      prompt,
			Payload:     cloneJSONMap(req.envelopeData()),
			Attempt:     attempt,
		}
		next, err := d.hooks.DispatchAutomationTriggerPreFire(ctx, payload)
		if err != nil {
			if errors.Is(err, hookspkg.ErrAutomationFireCancelled) {
				return prompt, true, nil
			}
			return prompt, false, err
		}
		return strings.TrimSpace(next.Prompt), false, nil
	default:
		return prompt, false, nil
	}
}

func (d *Dispatcher) dispatchPostFireHook(ctx context.Context, req DispatchRequest, run Run) {
	if d == nil || d.hooks == nil {
		return
	}

	switch {
	case req.Job != nil:
		if _, err := d.hooks.DispatchAutomationJobPostFire(ctx, hookspkg.AutomationJobPostFirePayload{
			JobID:       strings.TrimSpace(req.Job.ID),
			JobName:     strings.TrimSpace(req.Job.Name),
			AgentName:   strings.TrimSpace(req.Job.AgentName),
			WorkspaceID: strings.TrimSpace(req.Job.WorkspaceID),
			RunID:       strings.TrimSpace(run.ID),
			SessionID:   strings.TrimSpace(run.SessionID),
		}); err != nil {
			d.logHookDispatchError(
				"automation.dispatch.job_post_fire_hook_failed",
				err,
				"job_id",
				strings.TrimSpace(req.Job.ID),
				"run_id",
				strings.TrimSpace(run.ID),
			)
		}
	case req.Trigger != nil:
		if _, err := d.hooks.DispatchAutomationTriggerPostFire(ctx, hookspkg.AutomationTriggerPostFirePayload{
			TriggerID:   strings.TrimSpace(req.Trigger.ID),
			TriggerName: strings.TrimSpace(req.Trigger.Name),
			Event:       strings.TrimSpace(req.Trigger.Event),
			AgentName:   strings.TrimSpace(req.Trigger.AgentName),
			WorkspaceID: strings.TrimSpace(req.Trigger.WorkspaceID),
			RunID:       strings.TrimSpace(run.ID),
			SessionID:   strings.TrimSpace(run.SessionID),
		}); err != nil {
			d.logHookDispatchError(
				"automation.dispatch.trigger_post_fire_hook_failed",
				err,
				"trigger_id",
				strings.TrimSpace(req.Trigger.ID),
				"run_id",
				strings.TrimSpace(run.ID),
			)
		}
	}
}

func (d *Dispatcher) emitRunLifecycleHooks(
	ctx context.Context,
	req DispatchRequest,
	run Run,
	dispatchErr error,
	willRetry bool,
) {
	if d == nil || d.hooks == nil {
		return
	}
	if run.Status == RunCompleted {
		if _, err := d.hooks.DispatchAutomationRunCompleted(ctx, hookspkg.AutomationRunCompletedPayload{
			RunID:       strings.TrimSpace(run.ID),
			JobID:       strings.TrimSpace(run.JobID),
			TriggerID:   strings.TrimSpace(run.TriggerID),
			AgentName:   req.agentName(),
			WorkspaceID: req.workspaceID(),
			SessionID:   strings.TrimSpace(run.SessionID),
			Attempt:     run.Attempt,
			DurationMS:  runDurationMilliseconds(run),
		}); err != nil {
			d.logHookDispatchError(
				"automation.dispatch.run_completed_hook_failed",
				err,
				"run_id",
				strings.TrimSpace(run.ID),
			)
		}
		return
	}
	if run.Status != RunFailed {
		return
	}

	errText := strings.TrimSpace(run.Error)
	if errText == "" && dispatchErr != nil {
		errText = dispatchErr.Error()
	}
	if _, err := d.hooks.DispatchAutomationRunFailed(ctx, hookspkg.AutomationRunFailedPayload{
		RunID:       strings.TrimSpace(run.ID),
		JobID:       strings.TrimSpace(run.JobID),
		TriggerID:   strings.TrimSpace(run.TriggerID),
		AgentName:   req.agentName(),
		WorkspaceID: req.workspaceID(),
		SessionID:   strings.TrimSpace(run.SessionID),
		Error:       errText,
		Attempt:     run.Attempt,
		WillRetry:   willRetry,
	}); err != nil {
		d.logHookDispatchError("automation.dispatch.run_failed_hook_failed", err, "run_id", strings.TrimSpace(run.ID))
	}
}

func (d *Dispatcher) createOpts(req DispatchRequest) session.CreateOpts {
	opts := session.CreateOpts{
		AgentName: req.agentName(),
		Provider:  "",
		Name:      req.name(),
		Type:      session.SessionTypeSystem,
	}

	switch req.scope() {
	case AutomationScopeWorkspace:
		opts.Workspace = req.workspaceID()
	default:
		opts.WorkspacePath = d.globalWorkspacePath
	}

	return opts
}

func (d *Dispatcher) recordAutomationSessionTaskActor(sessionID string, run *Run) error {
	if d == nil || d.taskActors == nil {
		return nil
	}
	actor, err := automationSessionTaskActorContext(sessionID, run)
	if err != nil {
		return err
	}
	return d.taskActors.RecordAutomationSessionTaskActor(strings.TrimSpace(sessionID), actor)
}

func (d *Dispatcher) deleteAutomationSessionTaskActor(sessionID string) {
	if d == nil || d.taskActors == nil {
		return
	}
	d.taskActors.DeleteAutomationSessionTaskActor(strings.TrimSpace(sessionID))
}

func (d *Dispatcher) tryAcquire() bool {
	select {
	case d.gate <- struct{}{}:
		return true
	default:
		return false
	}
}

func (d *Dispatcher) release() {
	select {
	case <-d.gate:
	default:
	}
}

func (r DispatchRequest) agentName() string {
	if r.Job != nil {
		return strings.TrimSpace(r.Job.AgentName)
	}
	return strings.TrimSpace(r.Trigger.AgentName)
}

func (r DispatchRequest) name() string {
	if r.Job != nil {
		return strings.TrimSpace(r.Job.Name)
	}
	return strings.TrimSpace(r.Trigger.Name)
}

func (r DispatchRequest) scope() Scope {
	if r.Job != nil {
		return r.Job.Scope
	}
	return r.Trigger.Scope
}

func (r DispatchRequest) workspaceID() string {
	if r.Job != nil {
		return strings.TrimSpace(r.Job.WorkspaceID)
	}
	return strings.TrimSpace(r.Trigger.WorkspaceID)
}

func (r DispatchRequest) envelopeData() map[string]any {
	if r.Envelope == nil {
		return nil
	}
	return r.Envelope.Data
}

func (r DispatchRequest) retryConfig() RetryConfig {
	if r.Job != nil {
		return r.Job.Retry
	}
	return r.Trigger.Retry
}

func (r DispatchRequest) fireLimitConfig() FireLimitConfig {
	if r.Job != nil {
		return r.Job.FireLimit
	}
	return r.Trigger.FireLimit
}

func (r DispatchRequest) prompt() (string, error) {
	source := strings.TrimSpace(r.Prompt)
	if source == "" {
		if r.Job != nil {
			source = strings.TrimSpace(r.Job.Prompt)
		} else {
			source = strings.TrimSpace(r.Trigger.Prompt)
		}
	}
	if source == "" {
		return "", errors.New("automation: dispatch prompt is required")
	}

	if r.Job != nil {
		return source, nil
	}

	return renderTriggerPrompt(source, r.Envelope)
}

func renderTriggerPrompt(raw string, envelope *ActivationEnvelope) (string, error) {
	if envelope == nil {
		return "", errors.New("automation: activation envelope is required")
	}
	if !strings.Contains(raw, "{{") && !strings.Contains(raw, "}}") {
		return strings.TrimSpace(raw), nil
	}

	tmpl, err := ParseTriggerPromptTemplate(raw)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if execErr := executePromptTemplate(&builder, tmpl, envelope); execErr != nil {
		return "", execErr
	}
	return strings.TrimSpace(builder.String()), nil
}

func executePromptTemplate(builder *strings.Builder, tmpl *template.Template, envelope *ActivationEnvelope) error {
	if builder == nil {
		return errors.New("automation: prompt builder is required")
	}
	if tmpl == nil {
		return errors.New("automation: prompt template is required")
	}
	if envelope == nil {
		return errors.New("automation: activation envelope is required")
	}

	if err := tmpl.Execute(builder, envelope); err != nil {
		return fmt.Errorf("automation: execute trigger prompt template: %w", err)
	}
	return nil
}

func directTaskActorContext(job *Job, runID string) (taskpkg.ActorContext, error) {
	if job == nil {
		return taskpkg.ActorContext{}, errors.New("automation: task-backed dispatch job is required")
	}
	return taskpkg.DeriveAutomationActorContext(strings.TrimSpace(job.ID), automationTaskOriginRef(runID))
}

func automationSessionTaskActorContext(sessionID string, run *Run) (taskpkg.ActorContext, error) {
	if run == nil {
		return taskpkg.ActorContext{}, errors.New("automation: run is required for session task actor context")
	}
	return taskpkg.DeriveAutomationLinkedAgentSessionActorContext(
		strings.TrimSpace(sessionID),
		automationTaskOriginRef(run.ID),
	)
}

func directTaskSpec(job *Job, prompt string) taskpkg.CreateTask {
	if job == nil || job.Task == nil {
		return taskpkg.CreateTask{}
	}

	title := strings.TrimSpace(job.Task.Title)
	if title == "" {
		title = strings.TrimSpace(job.Name)
	}
	description := strings.TrimSpace(job.Task.Description)
	if description == "" {
		description = strings.TrimSpace(prompt)
	}
	if description == "" {
		description = strings.TrimSpace(job.Prompt)
	}

	return taskpkg.CreateTask{
		Scope:          taskScopeForAutomationScope(job.Scope),
		WorkspaceID:    strings.TrimSpace(job.WorkspaceID),
		NetworkChannel: strings.TrimSpace(job.Task.NetworkChannel),
		Title:          title,
		Description:    description,
		Owner:          cloneTaskOwnership(job.Task.Owner),
	}
}

func taskScopeForAutomationScope(scope Scope) taskpkg.Scope {
	switch scope {
	case AutomationScopeWorkspace:
		return taskpkg.ScopeWorkspace
	default:
		return taskpkg.ScopeGlobal
	}
}

func cloneTaskOwnership(owner *taskpkg.Ownership) *taskpkg.Ownership {
	if owner == nil {
		return nil
	}
	cloned := *owner
	return &cloned
}

func automationTaskOriginRef(runID string) string {
	return "run:" + strings.TrimSpace(runID)
}

func automationTaskRunIdempotencyKey(runID string) string {
	return "automation-run:" + strings.TrimSpace(runID)
}

func classifyDispatchError(err error) RunStatus {
	if err == nil {
		return RunCompleted
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return RunCancelled
	}
	return RunFailed
}

func dispatchStopCause(status RunStatus, runErr error) (session.StopCause, string) {
	switch status {
	case RunCompleted:
		return session.CauseCompleted, ""
	case RunCancelled:
		return session.CauseUserRequested, strings.TrimSpace(errorText(runErr))
	default:
		return session.CauseFailed, strings.TrimSpace(errorText(runErr))
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func shouldRetry(cfg RetryConfig, run *Run, attempt int, dispatchErr error) bool {
	if dispatchErr == nil || run == nil || run.Status != RunFailed {
		return false
	}
	if cfg.Strategy != RetryStrategyBackoff {
		return false
	}
	return attempt <= cfg.MaxRetries
}

func retryDelay(cfg RetryConfig, attempt int) (time.Duration, error) {
	if cfg.Strategy != RetryStrategyBackoff {
		return 0, nil
	}

	baseDelay, err := time.ParseDuration(strings.TrimSpace(cfg.BaseDelay))
	if err != nil {
		return 0, fmt.Errorf("automation: parse retry base delay: %w", err)
	}
	if attempt <= 1 {
		return baseDelay, nil
	}
	return baseDelay * time.Duration(1<<(attempt-1)), nil
}

func collectPromptError(ctx context.Context, events <-chan acp.AgentEvent) error {
	if events == nil {
		return errors.New("automation: prompt event stream is required")
	}

	var errs []error
	for {
		select {
		case <-ctx.Done():
			if len(errs) > 0 {
				errs = append(errs, ctx.Err())
				return errors.Join(errs...)
			}
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				if len(errs) > 0 {
					return errors.Join(errs...)
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				return nil
			}
			if trimmed := strings.TrimSpace(event.Error); trimmed != "" {
				errs = append(errs, errors.New(trimmed))
			}
		}
	}
}

func (d *Dispatcher) logHookDispatchError(message string, err error, attrs ...any) {
	if d == nil || err == nil || d.logger == nil {
		return
	}
	fields := append([]any{}, attrs...)
	fields = append(fields, "error", err)
	d.logger.Warn(message, fields...)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func persistenceContext(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	if ctx.Err() == nil {
		return ctx
	}
	return context.WithoutCancel(ctx)
}

func timePointer(value time.Time) *time.Time {
	timestamp := value
	return &timestamp
}

func cloneRun(run *Run) *Run {
	if run == nil {
		return nil
	}

	cloned := *run
	if run.StartedAt != nil {
		startedAt := *run.StartedAt
		cloned.StartedAt = &startedAt
	}
	if run.EndedAt != nil {
		endedAt := *run.EndedAt
		cloned.EndedAt = &endedAt
	}
	return &cloned
}

func hookSchedulePayload(schedule *ScheduleSpec) *hookspkg.AutomationSchedulePayload {
	if schedule == nil {
		return nil
	}
	return &hookspkg.AutomationSchedulePayload{
		Mode:     string(schedule.Mode),
		Expr:     strings.TrimSpace(schedule.Expr),
		Interval: strings.TrimSpace(schedule.Interval),
		Time:     strings.TrimSpace(schedule.Time),
	}
}

func runDurationMilliseconds(run Run) int64 {
	if run.StartedAt == nil || run.EndedAt == nil {
		return 0
	}
	return run.EndedAt.UTC().Sub(run.StartedAt.UTC()).Milliseconds()
}

func cloneJSONMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	maps.Copy(cloned, source)
	return cloned
}

func nestedPath(path string, field string) string {
	trimmedPath := strings.TrimSpace(path)
	trimmedField := strings.TrimSpace(field)
	switch {
	case trimmedPath == "":
		return trimmedField
	case trimmedField == "":
		return trimmedPath
	default:
		return trimmedPath + "." + trimmedField
	}
}
