package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
)

var (
	// ErrConcurrencyLimitReached reports that the shared automation gate rejected a new run.
	ErrConcurrencyLimitReached = errors.New("automation: global concurrency limit reached")
	// ErrFireLimitReached reports that a definition exceeded its rolling fire-limit window.
	ErrFireLimitReached = errors.New("automation: fire limit reached")
)

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
		return fmt.Errorf("%s.kind must match %s.event: %q != %q", nestedPath(path, "envelope"), nestedPath(path, "trigger"), got, want)
	}
	if got, want := r.Envelope.Scope, r.Trigger.Scope; got != want {
		return fmt.Errorf("%s.scope must match %s.scope: %q != %q", nestedPath(path, "envelope"), nestedPath(path, "trigger"), got, want)
	}
	if got, want := strings.TrimSpace(r.Envelope.WorkspaceID), strings.TrimSpace(r.Trigger.WorkspaceID); got != want {
		return fmt.Errorf("%s.workspace_id must match %s.workspace_id: %q != %q", nestedPath(path, "envelope"), nestedPath(path, "trigger"), got, want)
	}

	return nil
}

// SessionCreator is the subset of session.Manager needed by the dispatcher.
type SessionCreator interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
}

// RunStore persists automation run state and restart-safe fire-limit inputs.
type RunStore interface {
	CreateRun(ctx context.Context, run Run) (Run, error)
	UpdateRun(ctx context.Context, run Run) (Run, error)
	CountRuns(ctx context.Context, query RunQuery) (int64, error)
}

// SleepFunc waits for retry backoff with context cancellation support.
type SleepFunc func(ctx context.Context, delay time.Duration) error

// DispatcherOption customizes shared automation dispatch behavior.
type DispatcherOption func(*Dispatcher)

// Dispatcher routes every automation activation through one execution path.
type Dispatcher struct {
	sessions SessionCreator
	runs     RunStore

	logger              *slog.Logger
	now                 func() time.Time
	sleep               SleepFunc
	globalWorkspacePath string
	maxConcurrent       int

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

	prompt, promptErr := req.prompt()
	if promptErr != nil {
		return d.finishRun(ctx, scheduledRun, RunFailed, promptErr)
	}

	createOpts := d.createOpts(req)
	createdSession, createErr := d.sessions.Create(ctx, createOpts)
	if createErr != nil {
		return d.finishRun(ctx, scheduledRun, classifyDispatchError(createErr), createErr)
	}
	if createdSession == nil || strings.TrimSpace(createdSession.ID) == "" {
		return d.finishRun(ctx, scheduledRun, RunFailed, errors.New("automation: session creator returned empty session"))
	}

	runningRun, err := d.transitionRun(ctx, scheduledRun, func(run *Run, now time.Time) {
		run.Status = RunRunning
		run.SessionID = strings.TrimSpace(createdSession.ID)
	})
	if err != nil {
		return cloneRun(scheduledRun), err
	}

	events, promptErr := d.sessions.Prompt(ctx, createdSession.ID, prompt)
	if promptErr != nil {
		return d.finishRun(ctx, runningRun, classifyDispatchError(promptErr), promptErr)
	}

	runErr := collectPromptError(ctx, events)
	if runErr != nil {
		return d.finishRun(ctx, runningRun, classifyDispatchError(runErr), runErr)
	}

	return d.finishRun(ctx, runningRun, RunCompleted, nil)
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

func (d *Dispatcher) transitionRun(ctx context.Context, current *Run, mutate func(run *Run, now time.Time)) (*Run, error) {
	if current == nil {
		return nil, errors.New("automation: run is required")
	}

	next := *current
	mutate(&next, d.now())

	updated, err := d.runs.UpdateRun(ctx, next)
	if err != nil {
		return cloneRun(current), fmt.Errorf("automation: update run %q: %w", current.ID, err)
	}
	return &updated, nil
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

	if runErr == nil {
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

func (d *Dispatcher) createOpts(req DispatchRequest) session.CreateOpts {
	opts := session.CreateOpts{
		AgentName: req.agentName(),
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

func (r DispatchRequest) scope() AutomationScope {
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

func classifyDispatchError(err error) RunStatus {
	if err == nil {
		return RunCompleted
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return RunCancelled
	}
	return RunFailed
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
	for event := range events {
		if trimmed := strings.TrimSpace(event.Error); trimmed != "" {
			errs = append(errs, errors.New(trimmed))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
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
