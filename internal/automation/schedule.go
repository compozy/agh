package automation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	gocron "github.com/go-co-op/gocron/v2"
	"github.com/jonboulle/clockwork"
)

var (
	// ErrScheduledJobNotFound reports that a scheduler registration does not exist.
	ErrScheduledJobNotFound = errors.New("automation: scheduled job not found")
	// ErrScheduledJobAlreadyRegistered reports that the job is already registered with the scheduler.
	ErrScheduledJobAlreadyRegistered = errors.New("automation: scheduled job already registered")
	// ErrSchedulerStopped reports that the scheduler has already been stopped and cannot accept new work.
	ErrSchedulerStopped = errors.New("automation: scheduler stopped")
)

const defaultSchedulerStopTimeout = 10 * time.Second

// ScheduleDispatcher is the execution surface used by scheduled jobs.
type ScheduleDispatcher interface {
	Dispatch(ctx context.Context, req DispatchRequest) (*Run, error)
}

// SchedulerStore persists durable scheduler cursor state and run
// reservations before dispatch.
type SchedulerStore interface {
	GetSchedulerState(ctx context.Context, jobID string) (SchedulerState, error)
	SaveSchedulerState(ctx context.Context, state SchedulerState) (SchedulerState, error)
	DeleteSchedulerState(ctx context.Context, jobID string) error
	ClaimScheduledRun(ctx context.Context, claim SchedulerClaim) (SchedulerClaimResult, error)
	RecordRunDeliveryError(ctx context.Context, runID string, runErr error) (Run, error)
}

// SchedulerOption customizes scheduled-job runtime behavior.
type SchedulerOption func(*Scheduler)

// ScheduledJobState exposes runtime schedule metadata for one registered job.
type ScheduledJobState struct {
	JobID               string                 `json:"job_id"`
	Registered          bool                   `json:"registered"`
	NextRun             *time.Time             `json:"next_run,omitempty"`
	LastRun             *time.Time             `json:"last_run,omitempty"`
	LastScheduledAt     *time.Time             `json:"last_scheduled_at,omitempty"`
	LastFireID          string                 `json:"last_fire_id,omitempty"`
	CatchUpPolicy       SchedulerCatchUpPolicy `json:"catch_up_policy,omitempty"`
	MisfireGraceSeconds int                    `json:"misfire_grace_seconds,omitempty"`
	LastMisfireAt       *time.Time             `json:"last_misfire_at,omitempty"`
	MisfireCount        int                    `json:"misfire_count,omitempty"`
	Durable             *SchedulerState        `json:"durable,omitempty"`
}

// Scheduler owns durable cursor-driven scheduled-job dispatch.
type Scheduler struct {
	dispatcher  ScheduleDispatcher
	store       SchedulerStore
	logger      *slog.Logger
	clock       clockwork.Clock
	location    *time.Location
	stopTimeout time.Duration

	mu            sync.RWMutex
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc
	wg            sync.WaitGroup
	started       bool
	stopped       bool
	registrations map[string]scheduledRegistration
}

type scheduledRegistration struct {
	definition   Job
	registeredAt time.Time
	state        SchedulerState
	cancel       context.CancelFunc
}

type schedulePlan struct {
	register bool
	nextRun  time.Time
}

// NewScheduler constructs a scheduled-job runtime over gocron.
func NewScheduler(dispatcher ScheduleDispatcher, opts ...SchedulerOption) (*Scheduler, error) {
	if dispatcher == nil {
		return nil, errors.New("automation: scheduler dispatcher is required")
	}

	scheduler := &Scheduler{
		dispatcher:    dispatcher,
		logger:        slog.Default(),
		clock:         clockwork.NewRealClock(),
		location:      time.UTC,
		stopTimeout:   defaultSchedulerStopTimeout,
		registrations: make(map[string]scheduledRegistration),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(scheduler)
		}
	}

	if scheduler.logger == nil {
		scheduler.logger = slog.Default()
	}
	if scheduler.clock == nil {
		scheduler.clock = clockwork.NewRealClock()
	}
	if scheduler.location == nil {
		scheduler.location = time.UTC
	}
	if scheduler.stopTimeout <= 0 {
		scheduler.stopTimeout = defaultSchedulerStopTimeout
	}

	return scheduler, nil
}

// WithSchedulerLogger overrides the scheduler logger.
func WithSchedulerLogger(logger *slog.Logger) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.logger = logger
	}
}

// WithSchedulerStore injects durable scheduler cursor persistence.
func WithSchedulerStore(store SchedulerStore) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.store = store
	}
}

// WithSchedulerClock overrides the scheduler clock, mainly for tests.
func WithSchedulerClock(clock clockwork.Clock) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.clock = clock
	}
}

// WithSchedulerLocation overrides the timezone used for schedule evaluation.
func WithSchedulerLocation(location *time.Location) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.location = location
	}
}

// WithSchedulerStopTimeout overrides the graceful shutdown timeout used by gocron.
func WithSchedulerStopTimeout(timeout time.Duration) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.stopTimeout = timeout
	}
}

// Start begins scheduled-job execution.
func (s *Scheduler) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: scheduler start context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return ErrSchedulerStopped
	}
	if s.started {
		return nil
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.WithoutCancel(ctx))
	s.runtimeCtx = runtimeCtx
	s.runtimeCancel = runtimeCancel
	for jobID := range s.registrations {
		s.startJobLoopLocked(jobID)
	}
	s.started = true
	s.logger.Info("automation.scheduler.started", "jobs_loaded", len(s.registrations))
	return nil
}

// Stop shuts the scheduler down and cancels in-flight dispatches.
func (s *Scheduler) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("automation: scheduler stop context is required")
	}

	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	s.started = false
	cancel := s.runtimeCancel
	for jobID, registration := range s.registrations {
		if registration.cancel != nil {
			registration.cancel()
			registration.cancel = nil
			s.registrations[jobID] = registration
		}
	}
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	startedAt := time.Now()
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	var shutdownErr error
	select {
	case <-done:
	case <-ctx.Done():
		shutdownErr = ctx.Err()
	}
	s.mu.Lock()
	s.registrations = make(map[string]scheduledRegistration)
	s.runtimeCtx = nil
	s.runtimeCancel = nil
	s.mu.Unlock()

	s.logger.Info("automation.scheduler.shutdown", "shutdown_duration_ms", time.Since(startedAt).Milliseconds())
	if shutdownErr != nil {
		return fmt.Errorf("automation: shutdown scheduler runtime: %w", shutdownErr)
	}
	return nil
}

// Shutdown is an alias for Stop to match daemon runtime conventions.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	return s.Stop(ctx)
}

// Register adds a new scheduled job registration.
func (s *Scheduler) Register(ctx context.Context, job Job) (ScheduledJobState, error) {
	if ctx == nil {
		return ScheduledJobState{}, errors.New("automation: scheduler register context is required")
	}
	normalized, err := normalizeScheduledJob(job)
	if err != nil {
		return ScheduledJobState{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureMutable(); err != nil {
		return ScheduledJobState{}, err
	}
	if _, exists := s.registrations[normalized.ID]; exists {
		return ScheduledJobState{}, fmt.Errorf("%w: %s", ErrScheduledJobAlreadyRegistered, normalized.ID)
	}

	return s.registerLocked(ctx, normalized)
}

// Update replaces an existing scheduled job registration.
func (s *Scheduler) Update(ctx context.Context, job Job) (ScheduledJobState, error) {
	if ctx == nil {
		return ScheduledJobState{}, errors.New("automation: scheduler update context is required")
	}
	normalized, err := normalizeScheduledJob(job)
	if err != nil {
		return ScheduledJobState{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureMutable(); err != nil {
		return ScheduledJobState{}, err
	}
	current, exists := s.registrations[normalized.ID]
	if !exists {
		return ScheduledJobState{}, fmt.Errorf("%w: %s", ErrScheduledJobNotFound, normalized.ID)
	}

	if !normalized.Enabled {
		s.unregisterLocked(normalized.ID, current)
		if err := s.deleteSchedulerState(ctx, normalized.ID); err != nil {
			return ScheduledJobState{}, err
		}
		return unregisteredJobState(normalized.ID), nil
	}

	plan, err := s.buildSchedulePlan(normalized)
	if err != nil {
		return ScheduledJobState{}, err
	}
	if !plan.register {
		s.unregisterLocked(normalized.ID, current)
		return unregisteredJobState(normalized.ID), nil
	}

	state, err := s.reconcileSchedulerState(ctx, normalized, plan)
	if err != nil {
		return ScheduledJobState{}, err
	}

	registeredAt := s.now()
	if current.cancel != nil {
		current.cancel()
	}
	s.registrations[normalized.ID] = scheduledRegistration{
		definition:   normalized,
		registeredAt: registeredAt,
		state:        state,
	}
	if s.started {
		s.startJobLoopLocked(normalized.ID)
	}
	s.logger.Info(
		"automation.scheduler.updated",
		"job_id",
		normalized.ID,
		"job_name",
		normalized.Name,
		"schedule_mode",
		normalized.Schedule.Mode,
	)
	return s.snapshotLocked(normalized.ID, s.registrations[normalized.ID]), nil
}

// Unregister removes a scheduled job registration.
func (s *Scheduler) Unregister(ctx context.Context, jobID string) error {
	if ctx == nil {
		return errors.New("automation: scheduler unregister context is required")
	}
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return errors.New("automation: scheduled job id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.registrations[trimmedID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrScheduledJobNotFound, trimmedID)
	}

	s.unregisterLocked(trimmedID, current)
	return s.deleteSchedulerState(ctx, trimmedID)
}

// State returns runtime metadata for one scheduled job.
func (s *Scheduler) State(jobID string) (ScheduledJobState, error) {
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return ScheduledJobState{}, errors.New("automation: scheduled job id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	current, exists := s.registrations[trimmedID]
	if !exists {
		return ScheduledJobState{}, fmt.Errorf("%w: %s", ErrScheduledJobNotFound, trimmedID)
	}

	return s.snapshotLocked(trimmedID, current), nil
}

// States returns runtime metadata for every registered scheduled job.
func (s *Scheduler) States() []ScheduledJobState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states := make([]ScheduledJobState, 0, len(s.registrations))
	for jobID, registration := range s.registrations {
		states = append(states, s.snapshotLocked(jobID, registration))
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].JobID < states[j].JobID
	})
	return states
}

func (s *Scheduler) ensureMutable() error {
	if s.stopped {
		return ErrSchedulerStopped
	}
	return nil
}

func (s *Scheduler) registerLocked(ctx context.Context, job Job) (ScheduledJobState, error) {
	if !job.Enabled {
		if err := s.deleteSchedulerState(ctx, job.ID); err != nil {
			return ScheduledJobState{}, err
		}
		return unregisteredJobState(job.ID), nil
	}

	plan, err := s.buildSchedulePlan(job)
	if err != nil {
		return ScheduledJobState{}, err
	}
	if !plan.register {
		s.logger.Info("automation.scheduler.skipped_past_one_time_job", "job_id", job.ID, "job_name", job.Name)
		state, err := s.reconcileSchedulerState(ctx, job, plan)
		if err != nil {
			return ScheduledJobState{}, err
		}
		if s.store != nil {
			return stateFromDurableState(state, false), nil
		}
		return unregisteredJobState(job.ID), nil
	}

	state, err := s.reconcileSchedulerState(ctx, job, plan)
	if err != nil {
		return ScheduledJobState{}, err
	}
	registeredAt := s.now()
	registration := scheduledRegistration{
		definition:   job,
		registeredAt: registeredAt,
		state:        state,
	}
	s.registrations[job.ID] = registration
	if s.started {
		s.startJobLoopLocked(job.ID)
	}
	s.logger.Info(
		"automation.scheduler.registered",
		"job_id",
		job.ID,
		"job_name",
		job.Name,
		"schedule_mode",
		job.Schedule.Mode,
	)
	return s.snapshotLocked(job.ID, registration), nil
}

func (s *Scheduler) unregisterLocked(jobID string, registration scheduledRegistration) {
	if registration.cancel != nil {
		registration.cancel()
	}
	delete(s.registrations, jobID)
	s.logger.Info("automation.scheduler.unregistered", "job_id", jobID, "job_name", registration.definition.Name)
}

func (s *Scheduler) snapshotLocked(jobID string, registration scheduledRegistration) ScheduledJobState {
	state := stateFromDurableState(registration.state, true)
	if state.JobID == "" {
		state.JobID = jobID
	}
	if state.NextRun == nil {
		predicted := predictNextRun(registration.definition, registration.registeredAt, s.location)
		if !predicted.IsZero() {
			state.NextRun = timePointer(predicted)
		}
	}
	return state
}

func stateFromDurableState(durable SchedulerState, registered bool) ScheduledJobState {
	state := ScheduledJobState{
		JobID:               durable.JobID,
		Registered:          registered,
		NextRun:             durable.NextRunAt,
		LastRun:             durable.LastRunAt,
		LastScheduledAt:     durable.LastScheduledAt,
		LastFireID:          durable.LastFireID,
		CatchUpPolicy:       durable.CatchUpPolicy,
		MisfireGraceSeconds: durable.MisfireGraceSeconds,
		LastMisfireAt:       durable.LastMisfireAt,
		MisfireCount:        durable.MisfireCount,
	}
	if strings.TrimSpace(durable.JobID) != "" {
		state.JobID = durable.JobID
		durableCopy := durable
		state.Durable = &durableCopy
	}
	return state
}

func (s *Scheduler) buildSchedulePlan(job Job) (schedulePlan, error) {
	now := s.now()
	if job.Schedule == nil {
		return schedulePlan{}, errors.New("automation: job schedule is required")
	}

	switch job.Schedule.Mode {
	case ScheduleModeCron:
		cronImpl := gocron.NewDefaultCron(false)
		expr := strings.TrimSpace(job.Schedule.Expr)
		if err := cronImpl.IsValid(expr, s.location, now); err != nil {
			return schedulePlan{}, fmt.Errorf("automation: validate cron schedule for job %q: %w", job.ID, err)
		}
		return schedulePlan{
			register: true,
			nextRun:  cronImpl.Next(now),
		}, nil
	case ScheduleModeEvery:
		interval, err := time.ParseDuration(strings.TrimSpace(job.Schedule.Interval))
		if err != nil {
			return schedulePlan{}, fmt.Errorf("automation: parse interval schedule for job %q: %w", job.ID, err)
		}
		return schedulePlan{
			register: true,
			nextRun:  now.Add(interval),
		}, nil
	case ScheduleModeAt:
		atTime, err := time.Parse(time.RFC3339, strings.TrimSpace(job.Schedule.Time))
		if err != nil {
			return schedulePlan{}, fmt.Errorf("automation: parse one-time schedule for job %q: %w", job.ID, err)
		}
		if !atTime.After(now) {
			return schedulePlan{register: false}, nil
		}
		return schedulePlan{
			register: true,
			nextRun:  atTime,
		}, nil
	default:
		return schedulePlan{}, fmt.Errorf("automation: unsupported schedule mode %q", job.Schedule.Mode)
	}
}

func (s *Scheduler) startJobLoopLocked(jobID string) {
	registration, exists := s.registrations[jobID]
	if !exists || s.runtimeCtx == nil {
		return
	}
	if registration.cancel != nil {
		registration.cancel()
	}

	jobCtx, cancel := context.WithCancel(s.runtimeCtx)
	registration.cancel = cancel
	s.registrations[jobID] = registration
	s.wg.Add(1)
	go s.runJobLoop(jobCtx, jobID)
}

func (s *Scheduler) runJobLoop(ctx context.Context, jobID string) {
	defer s.wg.Done()

	for {
		registration, ok := s.registrationSnapshot(jobID)
		if !ok {
			return
		}
		if registration.state.NextRunAt == nil || registration.state.NextRunAt.IsZero() {
			return
		}

		delay := max(registration.state.NextRunAt.Sub(s.now()), 0)
		timer := s.clock.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.Chan():
		}

		if err := s.executeScheduledJob(ctx, jobID); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Warn("automation.scheduler.dispatch_failed", "job_id", jobID, "error", err)
		}
	}
}

func (s *Scheduler) registrationSnapshot(jobID string) (scheduledRegistration, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	registration, ok := s.registrations[jobID]
	return registration, ok
}

func (s *Scheduler) executeScheduledJob(ctx context.Context, jobID string) error {
	registration, ok := s.registrationSnapshot(jobID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrScheduledJobNotFound, jobID)
	}
	if registration.state.NextRunAt == nil || registration.state.NextRunAt.IsZero() {
		return nil
	}

	job := registration.definition
	scheduledAt := *registration.state.NextRunAt
	claimedAt := s.now()
	nextRun := nextRunAfter(job, scheduledAt, s.location)
	fireID := scheduledFireID(job.ID, scheduledAt)
	claim := SchedulerClaim{
		JobID:        job.ID,
		RunID:        scheduledRunID(job.ID, scheduledAt),
		FireID:       fireID,
		ScheduledAt:  scheduledAt,
		NextRunAt:    cloneTimePointer(nextRun),
		ClaimedAt:    claimedAt,
		ScheduleHash: scheduleHash(job.Schedule),
	}

	var reservedRun *Run
	var postClaimState SchedulerState
	if s.store != nil {
		result, err := s.store.ClaimScheduledRun(persistenceContext(ctx), claim)
		if err != nil {
			if errors.Is(err, ErrScheduledFireAlreadyClaimed) {
				s.updateRegistrationState(job.ID, registration.state)
				return nil
			}
			return fmt.Errorf("automation: claim scheduled job %q: %w", job.ID, err)
		}
		postClaimState = result.State
		s.updateRegistrationState(job.ID, postClaimState)
		reservedRun = &result.Run
	} else {
		state := registration.state
		state.NextRunAt = cloneTimePointer(nextRun)
		state.LastRunAt = timePointer(claimedAt)
		state.LastScheduledAt = timePointer(scheduledAt)
		state.LastFireID = fireID
		state.ScheduleHash = claim.ScheduleHash
		state.CatchUpPolicy = SchedulerCatchUpPolicySkipMissed
		state.UpdatedAt = claimedAt
		postClaimState = state
		s.updateRegistrationState(job.ID, postClaimState)
	}

	s.logScheduledFire(job, fireID, scheduledAt)

	run, err := s.dispatcher.Dispatch(ctx, DispatchRequest{
		Kind:        DispatchKindSchedule,
		Job:         &job,
		ReservedRun: reservedRun,
	})
	if fireLimitErr, ok := errors.AsType[*FireLimitError](err); ok {
		if adjustErr := s.deferAfterFireLimit(ctx, job.ID, postClaimState, fireLimitErr); adjustErr != nil {
			return errors.Join(err, adjustErr)
		}
		return nil
	}
	if err != nil && s.store != nil {
		runID := strings.TrimSpace(claim.RunID)
		if run != nil && strings.TrimSpace(run.ID) != "" {
			runID = run.ID
		}
		if _, recordErr := s.store.RecordRunDeliveryError(persistenceContext(ctx), runID, err); recordErr != nil {
			err = errors.Join(err, recordErr)
		}
	}
	return err
}

func (s *Scheduler) logScheduledFire(job Job, fireID string, scheduledAt time.Time) {
	s.logger.Info(
		"automation.scheduler.job_fired",
		"job_id", job.ID,
		"job_name", job.Name,
		"agent", job.AgentName,
		"schedule_mode", job.Schedule.Mode,
		"fire_id", fireID,
		"scheduled_at", scheduledAt.Format(time.RFC3339Nano),
	)
}

func (s *Scheduler) deferAfterFireLimit(
	ctx context.Context,
	jobID string,
	state SchedulerState,
	fireLimitErr *FireLimitError,
) error {
	if fireLimitErr == nil || fireLimitErr.RetryAt.IsZero() {
		return nil
	}

	target := fireLimitErr.RetryAt.In(s.location)
	if state.NextRunAt != nil && state.NextRunAt.After(target) {
		target = *state.NextRunAt
	}
	state.NextRunAt = timePointer(target)
	state.UpdatedAt = s.now()

	if s.store != nil {
		saved, err := s.store.SaveSchedulerState(persistenceContext(ctx), state)
		if err != nil {
			return fmt.Errorf("automation: defer scheduler after fire limit for job %q: %w", jobID, err)
		}
		state = saved
	}
	s.updateRegistrationState(jobID, state)
	s.logger.Info(
		"automation.scheduler.fire_limit_deferred",
		"job_id", jobID,
		"retry_at", target.Format(time.RFC3339Nano),
		"fires", fireLimitErr.Count,
		"limit", fireLimitErr.Limit,
		"window", fireLimitErr.Window.String(),
	)
	return nil
}

func (s *Scheduler) updateRegistrationState(jobID string, state SchedulerState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registration, exists := s.registrations[jobID]
	if !exists {
		return
	}
	registration.state = state
	s.registrations[jobID] = registration
	if state.NextRunAt == nil || state.NextRunAt.IsZero() {
		if registration.cancel != nil {
			registration.cancel()
		}
		delete(s.registrations, jobID)
	}
}

func (s *Scheduler) reconcileSchedulerState(
	ctx context.Context,
	job Job,
	plan schedulePlan,
) (SchedulerState, error) {
	now := s.now()
	state := SchedulerState{
		JobID:         job.ID,
		NextRunAt:     timePointer(plan.nextRun),
		ScheduleHash:  scheduleHash(job.Schedule),
		CatchUpPolicy: SchedulerCatchUpPolicySkipMissed,
		UpdatedAt:     now,
	}
	if s.store == nil {
		if !plan.register {
			state.NextRunAt = nil
			state.LastMisfireAt = timePointer(now)
			state.MisfireCount = 1
		}
		return state, nil
	}

	existing, err := s.store.GetSchedulerState(ctx, job.ID)
	if err != nil && !errors.Is(err, ErrSchedulerStateNotFound) {
		return SchedulerState{}, fmt.Errorf("automation: load scheduler state for job %q: %w", job.ID, err)
	}
	if err == nil {
		state = existing
		state.CatchUpPolicy = schedulerCatchUpPolicyOrDefault(state.CatchUpPolicy)
		state.UpdatedAt = now
		if strings.TrimSpace(state.ScheduleHash) != scheduleHash(job.Schedule) {
			state.NextRunAt = timePointer(plan.nextRun)
			state.ScheduleHash = scheduleHash(job.Schedule)
			state.ConsecutiveResumeFailures = 0
		}
	} else {
		state.ScheduleHash = scheduleHash(job.Schedule)
	}

	if !plan.register {
		state.NextRunAt = nil
		state.LastMisfireAt = timePointer(now)
		state.MisfireCount++
		state.UpdatedAt = now
		return s.store.SaveSchedulerState(ctx, state)
	}

	if state.NextRunAt == nil || state.NextRunAt.IsZero() {
		state.NextRunAt = timePointer(plan.nextRun)
		state.UpdatedAt = now
		return s.store.SaveSchedulerState(ctx, state)
	}

	if !state.NextRunAt.After(now) {
		missedAt := *state.NextRunAt
		state.NextRunAt = nextRunAfterMissed(job, missedAt, now, s.location)
		state.LastScheduledAt = timePointer(missedAt)
		state.LastMisfireAt = timePointer(now)
		state.MisfireCount++
		state.ConsecutiveResumeFailures = 0
		state.UpdatedAt = now
		return s.store.SaveSchedulerState(ctx, state)
	}

	return state, nil
}

func (s *Scheduler) deleteSchedulerState(ctx context.Context, jobID string) error {
	if s.store == nil {
		return nil
	}
	return s.store.DeleteSchedulerState(ctx, jobID)
}

func (s *Scheduler) now() time.Time {
	return s.clock.Now().In(s.location)
}

func normalizeScheduledJob(job Job) (Job, error) {
	job.ID = strings.TrimSpace(job.ID)
	if job.ID == "" {
		return Job{}, errors.New("automation: job.id is required for scheduler registration")
	}
	if err := job.Validate("job"); err != nil {
		return Job{}, err
	}
	return job, nil
}

func predictNextRun(job Job, registeredAt time.Time, location *time.Location) time.Time {
	if job.Schedule == nil {
		return time.Time{}
	}

	switch job.Schedule.Mode {
	case ScheduleModeCron:
		cronImpl := gocron.NewDefaultCron(false)
		expr := strings.TrimSpace(job.Schedule.Expr)
		if err := cronImpl.IsValid(expr, location, registeredAt); err != nil {
			return time.Time{}
		}
		return cronImpl.Next(registeredAt)
	case ScheduleModeEvery:
		interval, err := time.ParseDuration(strings.TrimSpace(job.Schedule.Interval))
		if err != nil {
			return time.Time{}
		}
		return registeredAt.Add(interval)
	case ScheduleModeAt:
		atTime, err := time.Parse(time.RFC3339, strings.TrimSpace(job.Schedule.Time))
		if err != nil {
			return time.Time{}
		}
		return atTime
	default:
		return time.Time{}
	}
}

func nextRunAfter(job Job, scheduledAt time.Time, location *time.Location) *time.Time {
	if job.Schedule == nil {
		return nil
	}

	var next time.Time
	switch job.Schedule.Mode {
	case ScheduleModeCron:
		cronImpl := gocron.NewDefaultCron(false)
		expr := strings.TrimSpace(job.Schedule.Expr)
		if err := cronImpl.IsValid(expr, location, scheduledAt); err != nil {
			return nil
		}
		next = cronImpl.Next(scheduledAt)
	case ScheduleModeEvery:
		interval, err := time.ParseDuration(strings.TrimSpace(job.Schedule.Interval))
		if err != nil || interval <= 0 {
			return nil
		}
		next = scheduledAt.Add(interval)
	case ScheduleModeAt:
		return nil
	default:
		return nil
	}
	if next.IsZero() {
		return nil
	}
	return timePointer(next)
}

func nextRunAfterMissed(job Job, missedAt time.Time, now time.Time, location *time.Location) *time.Time {
	if job.Schedule == nil {
		return nil
	}

	switch job.Schedule.Mode {
	case ScheduleModeCron:
		cronImpl := gocron.NewDefaultCron(false)
		expr := strings.TrimSpace(job.Schedule.Expr)
		if err := cronImpl.IsValid(expr, location, now); err != nil {
			return nil
		}
		next := cronImpl.Next(now)
		if next.IsZero() {
			return nil
		}
		return timePointer(next)
	case ScheduleModeEvery:
		interval, err := time.ParseDuration(strings.TrimSpace(job.Schedule.Interval))
		if err != nil || interval <= 0 {
			return nil
		}
		elapsed := now.Sub(missedAt)
		if elapsed < 0 {
			return timePointer(missedAt)
		}
		skippedIntervals := int64(elapsed/interval) + 1
		next := missedAt.Add(time.Duration(skippedIntervals) * interval)
		return timePointer(next)
	case ScheduleModeAt:
		return nil
	default:
		return nil
	}
}

func scheduledFireID(jobID string, scheduledAt time.Time) string {
	return stableSchedulerID("fire", jobID, scheduledAt)
}

func scheduledRunID(jobID string, scheduledAt time.Time) string {
	return stableSchedulerID("run", jobID, scheduledAt)
}

func stableSchedulerID(prefix string, jobID string, scheduledAt time.Time) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(jobID) + "|" + scheduledAt.UTC().Format(time.RFC3339Nano)))
	return prefix + "_" + hex.EncodeToString(hash[:12])
}

func scheduleHash(schedule *ScheduleSpec) string {
	if schedule == nil {
		return ""
	}
	hash := sha256.Sum256([]byte(strings.Join([]string{
		string(schedule.Mode),
		strings.TrimSpace(schedule.Expr),
		strings.TrimSpace(schedule.Interval),
		strings.TrimSpace(schedule.Time),
	}, "|")))
	return hex.EncodeToString(hash[:])
}

func schedulerCatchUpPolicyOrDefault(policy SchedulerCatchUpPolicy) SchedulerCatchUpPolicy {
	if policy == "" {
		return SchedulerCatchUpPolicySkipMissed
	}
	return policy
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	clone := *value
	return &clone
}

func unregisteredJobState(jobID string) ScheduledJobState {
	return ScheduledJobState{
		JobID:      jobID,
		Registered: false,
	}
}
