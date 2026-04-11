package automation

import (
	"context"
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

// SchedulerOption customizes scheduled-job runtime behavior.
type SchedulerOption func(*Scheduler)

// ScheduledJobState exposes runtime schedule metadata for one registered job.
type ScheduledJobState struct {
	JobID      string     `json:"job_id"`
	Registered bool       `json:"registered"`
	NextRun    *time.Time `json:"next_run,omitempty"`
	LastRun    *time.Time `json:"last_run,omitempty"`
}

// Scheduler wraps gocron behind the automation runtime surface for scheduled jobs.
type Scheduler struct {
	dispatcher  ScheduleDispatcher
	logger      *slog.Logger
	clock       clockwork.Clock
	location    *time.Location
	stopTimeout time.Duration

	mu            sync.RWMutex
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc
	scheduler     gocron.Scheduler
	started       bool
	stopped       bool
	registrations map[string]scheduledRegistration
}

type scheduledRegistration struct {
	definition   Job
	registeredAt time.Time
	job          gocron.Job
}

type schedulePlan struct {
	register bool
	nextRun  time.Time
	jobDef   gocron.JobDefinition
}

// NewScheduler constructs a scheduled-job runtime over gocron.
func NewScheduler(dispatcher ScheduleDispatcher, opts ...SchedulerOption) (*Scheduler, error) {
	if dispatcher == nil {
		return nil, errors.New("automation: scheduler dispatcher is required")
	}

	runtimeCtx, runtimeCancel := context.WithCancel(context.Background())
	scheduler := &Scheduler{
		dispatcher:    dispatcher,
		logger:        slog.Default(),
		clock:         clockwork.NewRealClock(),
		location:      time.UTC,
		stopTimeout:   defaultSchedulerStopTimeout,
		runtimeCtx:    runtimeCtx,
		runtimeCancel: runtimeCancel,
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

	gocronScheduler, err := gocron.NewScheduler(
		gocron.WithClock(scheduler.clock),
		gocron.WithLocation(scheduler.location),
		gocron.WithStopTimeout(scheduler.stopTimeout),
	)
	if err != nil {
		runtimeCancel()
		return nil, fmt.Errorf("automation: create scheduler runtime: %w", err)
	}
	scheduler.scheduler = gocronScheduler

	return scheduler, nil
}

// WithSchedulerLogger overrides the scheduler logger.
func WithSchedulerLogger(logger *slog.Logger) SchedulerOption {
	return func(scheduler *Scheduler) {
		scheduler.logger = logger
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

	s.scheduler.Start()
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
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	startedAt := time.Now()
	shutdownErr := s.scheduler.ShutdownWithContext(ctx)

	s.mu.Lock()
	s.registrations = make(map[string]scheduledRegistration)
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
func (s *Scheduler) Register(job Job) (ScheduledJobState, error) {
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

	return s.registerLocked(normalized)
}

// Update replaces an existing scheduled job registration.
func (s *Scheduler) Update(job Job) (ScheduledJobState, error) {
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
		if err := s.unregisterLocked(normalized.ID, current); err != nil {
			return ScheduledJobState{}, err
		}
		return unregisteredJobState(normalized.ID), nil
	}

	plan, err := s.buildSchedulePlan(normalized)
	if err != nil {
		return ScheduledJobState{}, err
	}
	if !plan.register {
		if err := s.unregisterLocked(normalized.ID, current); err != nil {
			return ScheduledJobState{}, err
		}
		return unregisteredJobState(normalized.ID), nil
	}

	registeredAt := s.now()
	updatedJob, err := s.scheduler.Update(
		current.job.ID(),
		plan.jobDef,
		gocron.NewTask(s.executeScheduledJob, normalized),
		s.jobOptions(normalized)...,
	)
	if err != nil {
		return ScheduledJobState{}, fmt.Errorf("automation: update scheduled job %q: %w", normalized.ID, err)
	}

	s.registrations[normalized.ID] = scheduledRegistration{
		definition:   normalized,
		registeredAt: registeredAt,
		job:          updatedJob,
	}
	s.logger.Info("automation.scheduler.updated", "job_id", normalized.ID, "job_name", normalized.Name, "schedule_mode", normalized.Schedule.Mode)
	return s.snapshotLocked(normalized.ID, s.registrations[normalized.ID]), nil
}

// Unregister removes a scheduled job registration.
func (s *Scheduler) Unregister(jobID string) error {
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

	return s.unregisterLocked(trimmedID, current)
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

func (s *Scheduler) registerLocked(job Job) (ScheduledJobState, error) {
	if !job.Enabled {
		return unregisteredJobState(job.ID), nil
	}

	plan, err := s.buildSchedulePlan(job)
	if err != nil {
		return ScheduledJobState{}, err
	}
	if !plan.register {
		s.logger.Info("automation.scheduler.skipped_past_one_time_job", "job_id", job.ID, "job_name", job.Name)
		return unregisteredJobState(job.ID), nil
	}

	registeredAt := s.now()
	registeredJob, err := s.scheduler.NewJob(
		plan.jobDef,
		gocron.NewTask(s.executeScheduledJob, job),
		s.jobOptions(job)...,
	)
	if err != nil {
		return ScheduledJobState{}, fmt.Errorf("automation: register scheduled job %q: %w", job.ID, err)
	}

	registration := scheduledRegistration{
		definition:   job,
		registeredAt: registeredAt,
		job:          registeredJob,
	}
	s.registrations[job.ID] = registration
	s.logger.Info("automation.scheduler.registered", "job_id", job.ID, "job_name", job.Name, "schedule_mode", job.Schedule.Mode)
	return s.snapshotLocked(job.ID, registration), nil
}

func (s *Scheduler) unregisterLocked(jobID string, registration scheduledRegistration) error {
	if err := s.scheduler.RemoveJob(registration.job.ID()); err != nil {
		return fmt.Errorf("automation: unregister scheduled job %q: %w", jobID, err)
	}
	delete(s.registrations, jobID)
	s.logger.Info("automation.scheduler.unregistered", "job_id", jobID, "job_name", registration.definition.Name)
	return nil
}

func (s *Scheduler) snapshotLocked(jobID string, registration scheduledRegistration) ScheduledJobState {
	state := ScheduledJobState{
		JobID:      jobID,
		Registered: true,
	}

	if nextRun, err := registration.job.NextRun(); err == nil && !nextRun.IsZero() {
		state.NextRun = timePointer(nextRun)
	} else {
		predicted := predictNextRun(registration.definition, registration.registeredAt, s.location)
		if !predicted.IsZero() {
			state.NextRun = timePointer(predicted)
		}
	}

	if lastRun, err := registration.job.LastRun(); err == nil && !lastRun.IsZero() {
		state.LastRun = timePointer(lastRun)
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
			jobDef:   gocron.CronJob(expr, false),
		}, nil
	case ScheduleModeEvery:
		interval, err := time.ParseDuration(strings.TrimSpace(job.Schedule.Interval))
		if err != nil {
			return schedulePlan{}, fmt.Errorf("automation: parse interval schedule for job %q: %w", job.ID, err)
		}
		return schedulePlan{
			register: true,
			nextRun:  now.Add(interval),
			jobDef:   gocron.DurationJob(interval),
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
			jobDef:   gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(atTime)),
		}, nil
	default:
		return schedulePlan{}, fmt.Errorf("automation: unsupported schedule mode %q", job.Schedule.Mode)
	}
}

func (s *Scheduler) jobOptions(job Job) []gocron.JobOption {
	return []gocron.JobOption{
		gocron.WithName(job.Name),
		gocron.WithTags("automation", job.ID),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	}
}

func (s *Scheduler) executeScheduledJob(job Job) error {
	dispatchCtx, cancel := context.WithCancel(s.runtimeCtx)
	defer cancel()

	s.logger.Info(
		"automation.scheduler.job_fired",
		"job_id", job.ID,
		"job_name", job.Name,
		"agent", job.AgentName,
		"schedule_mode", job.Schedule.Mode,
	)

	_, err := s.dispatcher.Dispatch(dispatchCtx, DispatchRequest{
		Kind: DispatchKindSchedule,
		Job:  &job,
	})

	if job.Schedule != nil && job.Schedule.Mode == ScheduleModeAt {
		if unregisterErr := s.unregisterAfterOneTimeFire(job.ID); unregisterErr != nil && !errors.Is(unregisterErr, ErrScheduledJobNotFound) && !errors.Is(unregisterErr, ErrSchedulerStopped) {
			if err == nil {
				err = unregisterErr
			} else {
				err = errors.Join(err, unregisterErr)
			}
		}
	}

	return err
}

func (s *Scheduler) unregisterAfterOneTimeFire(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	registration, exists := s.registrations[jobID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrScheduledJobNotFound, jobID)
	}
	return s.unregisterLocked(jobID, registration)
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

func unregisteredJobState(jobID string) ScheduledJobState {
	return ScheduledJobState{
		JobID:      jobID,
		Registered: false,
	}
}
