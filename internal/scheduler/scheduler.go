package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jonboulle/clockwork"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

type wakeKey struct {
	runID     string
	sessionID string
}

type selectionResult struct {
	targets          []WakeTarget
	noMatch          []RunSnapshot
	recentlyNotified int
	unclaimable      int
}

// Scheduler owns one mechanical sweep/notify loop.
type Scheduler struct {
	tasks    TaskSource
	sessions SessionSource
	waker    Waker

	logger       *slog.Logger
	clock        clockwork.Clock
	interval     time.Duration
	wakeCooldown time.Duration
	sweepReason  string
	wakeReason   string
	sweepLimit   int
	actor        taskpkg.ActorContext

	mu            sync.Mutex
	runtimeCancel context.CancelFunc
	runtimeDone   chan struct{}
	started       bool
	stopped       bool
	wakeState     map[wakeKey]time.Time
	stats         Stats
	wg            sync.WaitGroup
}

// New constructs a mechanical scheduler over durable task and session sources.
func New(tasks TaskSource, sessions SessionSource, waker Waker, opts ...Option) (*Scheduler, error) {
	if tasks == nil {
		return nil, errors.New("scheduler: task source is required")
	}
	if sessions == nil {
		return nil, errors.New("scheduler: session source is required")
	}
	if waker == nil {
		return nil, errors.New("scheduler: waker is required")
	}

	actor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
	if err != nil {
		return nil, fmt.Errorf("scheduler: derive daemon actor: %w", err)
	}
	s := &Scheduler{
		tasks:        tasks,
		sessions:     sessions,
		waker:        waker,
		logger:       slog.Default(),
		clock:        clockwork.NewRealClock(),
		interval:     defaultInterval,
		wakeCooldown: defaultWakeCooldown,
		sweepReason:  defaultSweepReason,
		wakeReason:   defaultWakeReason,
		sweepLimit:   defaultSweepLimit,
		actor:        actor,
		wakeState:    make(map[wakeKey]time.Time),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if s.clock == nil {
		s.clock = clockwork.NewRealClock()
	}
	if s.interval <= 0 {
		s.interval = defaultInterval
	}
	if s.wakeCooldown < 0 {
		s.wakeCooldown = defaultWakeCooldown
	}
	if strings.TrimSpace(s.sweepReason) == "" {
		s.sweepReason = defaultSweepReason
	}
	if strings.TrimSpace(s.wakeReason) == "" {
		s.wakeReason = defaultWakeReason
	}
	if s.sweepLimit < 0 {
		s.sweepLimit = defaultSweepLimit
	}
	if err := s.actor.Validate(); err != nil {
		return nil, fmt.Errorf("scheduler: validate daemon actor: %w", err)
	}
	return s, nil
}

// Start begins the context-bound background scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("scheduler: start context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return ErrStopped
	}
	if s.started {
		return nil
	}

	runtimeCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan struct{})
	s.runtimeCancel = cancel
	s.runtimeDone = done
	s.started = true
	s.wg.Go(func() {
		defer close(done)
		s.loop(runtimeCtx)
	})
	s.logger.Info("scheduler.started", "interval_ms", s.interval.Milliseconds())
	return nil
}

// Shutdown cancels the scheduler loop and waits for owned goroutines to exit.
func (s *Scheduler) Shutdown(ctx context.Context) error {
	if ctx == nil {
		return errors.New("scheduler: shutdown context is required")
	}

	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	s.started = false
	cancel := s.runtimeCancel
	done := s.runtimeDone
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return fmt.Errorf("scheduler: shutdown runtime: %w", ctx.Err())
		}
	}
	s.wg.Wait()

	s.mu.Lock()
	s.runtimeCancel = nil
	s.runtimeDone = nil
	s.mu.Unlock()
	s.logger.Info("scheduler.shutdown")
	return nil
}

// Stats returns a consistent snapshot of scheduler counters.
func (s *Scheduler) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

// Rebuild clears scheduler-owned ephemeral wake state after reading durable
// task/session state. The returned counts are observability only; durable
// recovery remains in RunOnce through the task service.
func (s *Scheduler) Rebuild(ctx context.Context) (RebuildResult, error) {
	if ctx == nil {
		return RebuildResult{}, errors.New("scheduler: rebuild context is required")
	}
	now := s.clock.Now().UTC()

	pending, err := s.tasks.PendingRuns(ctx)
	if err != nil {
		return RebuildResult{}, fmt.Errorf("scheduler: rebuild pending runs: %w", err)
	}
	active, err := s.tasks.ActiveRuns(ctx)
	if err != nil {
		return RebuildResult{}, fmt.Errorf("scheduler: rebuild active runs: %w", err)
	}
	sessions, err := s.sessions.Sessions(ctx)
	if err != nil {
		return RebuildResult{}, fmt.Errorf("scheduler: rebuild sessions: %w", err)
	}

	s.mu.Lock()
	cleared := len(s.wakeState)
	s.wakeState = make(map[wakeKey]time.Time)
	s.stats.Rebuilds++
	s.stats.LastRebuildAt = now
	s.mu.Unlock()

	result := RebuildResult{
		PendingRuns:     len(pending),
		ActiveRuns:      len(active),
		SessionsScanned: len(sessions),
		ClearedWakeKeys: cleared,
		RebuiltAt:       now,
	}
	s.logger.Info(
		"scheduler.rebuild",
		"pending_runs", result.PendingRuns,
		"active_runs", result.ActiveRuns,
		"sessions_scanned", result.SessionsScanned,
		"cleared_wake_keys", result.ClearedWakeKeys,
	)
	return result, nil
}

// RunOnce executes one sweep/notify pass.
func (s *Scheduler) RunOnce(ctx context.Context) (CycleResult, error) {
	if ctx == nil {
		return CycleResult{}, errors.New("scheduler: run context is required")
	}
	now := s.clock.Now().UTC()
	result := CycleResult{}
	errs := s.sweepExpiredLeases(ctx, now, &result)

	pending, active, sessions, err := s.loadCycleSnapshots(ctx, &result)
	if err != nil {
		return result, errors.Join(append(errs, err)...)
	}

	selection := s.selectWakeTargets(now, pending, sessions, active)
	applySelection(&result, selection)
	errs = append(errs, s.dispatchWakeTargets(ctx, now, selection.targets, &result)...)

	s.recordCycle(now, result)
	if result.NoMatchRuns > 0 {
		s.logger.Info("scheduler.wake.no_match", "runs", result.NoMatchRunIDs)
	}
	return result, errors.Join(errs...)
}

func (s *Scheduler) sweepExpiredLeases(ctx context.Context, now time.Time, result *CycleResult) []error {
	recovered, err := s.tasks.RecoverExpiredRunLeases(ctx, taskpkg.ExpiredLeaseRecovery{
		Now:    now,
		Reason: s.sweepReason,
		Limit:  s.sweepLimit,
	}, s.actor)
	if err != nil {
		s.recordRecoveryError(err)
		s.logger.Warn("scheduler.lease_sweep.error", "error", err)
		return []error{fmt.Errorf("scheduler: recover expired leases: %w", err)}
	}

	result.RecoveredLeases = len(recovered)
	result.RecoveredRunIDs = recoveredRunIDs(recovered)
	s.recordRecovered(len(recovered), now)
	s.logger.Info("scheduler.lease_sweep", "recovered_leases", len(recovered))
	return nil
}

func (s *Scheduler) loadCycleSnapshots(
	ctx context.Context,
	result *CycleResult,
) ([]RunSnapshot, []taskpkg.Run, []SessionSnapshot, error) {
	pending, err := s.tasks.PendingRuns(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("scheduler: list pending runs: %w", err)
	}
	active, err := s.tasks.ActiveRuns(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("scheduler: list active runs: %w", err)
	}
	sessions, err := s.sessions.Sessions(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("scheduler: list sessions: %w", err)
	}

	result.PendingRuns = len(pending)
	result.ActiveRuns = len(active)
	result.SessionsScanned = len(sessions)
	return pending, active, sessions, nil
}

func applySelection(result *CycleResult, selection selectionResult) {
	result.WakeAttempts = len(selection.targets)
	result.NoMatchRuns = len(selection.noMatch)
	result.RecentlyNotified = selection.recentlyNotified
	result.UnclaimableRuns = selection.unclaimable
	result.NoMatchRunIDs = runIDs(selection.noMatch)
}

func (s *Scheduler) dispatchWakeTargets(
	ctx context.Context,
	now time.Time,
	targets []WakeTarget,
	result *CycleResult,
) []error {
	var errs []error
	for idx := range targets {
		target := &targets[idx]
		target.Reason = s.wakeReason
		if err := s.waker.Wake(ctx, target); err != nil {
			result.WakeFailed++
			s.recordWakeError(err)
			errs = append(errs, fmt.Errorf(
				"scheduler: wake session %q for run %q: %w",
				target.Session.ID,
				target.Work.Run.ID,
				err,
			))
			s.logger.Warn(
				"scheduler.wake.error",
				"session_id", target.Session.ID,
				"run_id", target.Work.Run.ID,
				"task_id", target.Work.Task.ID,
				"error", err,
			)
			continue
		}
		result.WakeSucceeded++
		result.SelectedRunIDs = append(result.SelectedRunIDs, target.Work.Run.ID)
		s.markWoken(now, target)
		s.logger.Info(
			"scheduler.wake",
			"session_id", target.Session.ID,
			"run_id", target.Work.Run.ID,
			"task_id", target.Work.Task.ID,
		)
	}
	return errs
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := s.clock.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.Chan():
			if _, err := s.RunOnce(ctx); err != nil {
				s.logger.Warn("scheduler.cycle.error", "error", err)
			}
		}
	}
}

func (s *Scheduler) selectWakeTargets(
	now time.Time,
	pending []RunSnapshot,
	sessions []SessionSnapshot,
	active []taskpkg.Run,
) selectionResult {
	orderedPending := append([]RunSnapshot(nil), pending...)
	sortRunsForWake(orderedPending)
	orderedSessions := append([]SessionSnapshot(nil), sessions...)
	sortSessionsForWake(orderedSessions)

	busy := activeSessionIDs(active)
	targetedSessions := make(map[string]struct{})
	state := s.wakeStateSnapshot(now)
	result := selectionResult{}

	for idx := range orderedPending {
		work := &orderedPending[idx]
		if !isPotentiallyClaimable(work) {
			result.unclaimable++
			continue
		}

		var eligible []SessionSnapshot
		for _, candidate := range orderedSessions {
			if _, targeted := targetedSessions[strings.TrimSpace(candidate.ID)]; targeted {
				continue
			}
			if isEligibleSession(work, candidate, busy) {
				eligible = append(eligible, candidate)
			}
		}
		if len(eligible) == 0 {
			result.noMatch = append(result.noMatch, *work)
			continue
		}

		chosen, ok := firstNotRecentlyWoken(now, s.wakeCooldown, state, work, eligible)
		if !ok {
			result.recentlyNotified++
			continue
		}
		targetedSessions[strings.TrimSpace(chosen.ID)] = struct{}{}
		result.targets = append(result.targets, WakeTarget{Work: *work, Session: chosen, Reason: s.wakeReason})
	}
	return result
}

func (s *Scheduler) wakeStateSnapshot(now time.Time) map[wakeKey]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := make(map[wakeKey]time.Time, len(s.wakeState))
	for key, last := range s.wakeState {
		if s.wakeCooldown > 0 && now.Sub(last) >= s.wakeCooldown {
			delete(s.wakeState, key)
			continue
		}
		snapshot[key] = last
	}
	return snapshot
}

func (s *Scheduler) markWoken(now time.Time, target *WakeTarget) {
	if target == nil {
		return
	}
	key := wakeKey{
		runID:     strings.TrimSpace(target.Work.Run.ID),
		sessionID: strings.TrimSpace(target.Session.ID),
	}
	if key.runID == "" || key.sessionID == "" {
		return
	}
	s.mu.Lock()
	s.wakeState[key] = now
	s.mu.Unlock()
}

func (s *Scheduler) recordRecovered(count int, now time.Time) {
	s.mu.Lock()
	s.stats.RecoveredLeases += count
	s.stats.LastCycleAt = now
	s.mu.Unlock()
}

func (s *Scheduler) recordRecoveryError(err error) {
	s.mu.Lock()
	s.stats.RecoveryErrors++
	s.stats.LastRecoveryError = err.Error()
	s.mu.Unlock()
}

func (s *Scheduler) recordWakeError(err error) {
	s.mu.Lock()
	s.stats.WakeFailed++
	s.stats.LastWakeError = err.Error()
	s.mu.Unlock()
}

func (s *Scheduler) recordCycle(now time.Time, result CycleResult) {
	s.mu.Lock()
	s.stats.Cycles++
	s.stats.WakeAttempts += result.WakeAttempts
	s.stats.WakeSucceeded += result.WakeSucceeded
	s.stats.NoMatchRuns += result.NoMatchRuns
	s.stats.RecentlyNotified += result.RecentlyNotified
	s.stats.UnclaimableRuns += result.UnclaimableRuns
	s.stats.LastCycleAt = now
	s.mu.Unlock()
}

func isPotentiallyClaimable(work *RunSnapshot) bool {
	if work == nil {
		return false
	}
	if strings.TrimSpace(work.Run.ID) == "" || strings.TrimSpace(work.Task.ID) == "" {
		return false
	}
	if work.Run.Status.Normalize() != taskpkg.TaskRunStatusQueued {
		return false
	}
	switch work.Task.Status.Normalize() {
	case taskpkg.TaskStatusDraft, taskpkg.TaskStatusBlocked, taskpkg.TaskStatusCanceled:
		return false
	default:
		return true
	}
}

func isEligibleSession(work *RunSnapshot, candidate SessionSnapshot, busy map[string]struct{}) bool {
	if work == nil {
		return false
	}
	sessionID := strings.TrimSpace(candidate.ID)
	if sessionID == "" {
		return false
	}
	if strings.TrimSpace(candidate.State) != "active" {
		return false
	}
	if candidate.Prompting {
		return false
	}
	if _, isBusy := busy[sessionID]; isBusy {
		return false
	}
	if !scopeMatches(work.Task, candidate) {
		return false
	}
	if !coordinationChannelMatches(work, candidate) {
		return false
	}
	return capabilitiesCover(candidate.Capabilities, work.Run.RequiredCapabilities)
}

func coordinationChannelMatches(work *RunSnapshot, candidate SessionSnapshot) bool {
	if work == nil {
		return false
	}
	runChannel := strings.TrimSpace(work.Run.CoordinationChannelID)
	sessionChannel := strings.TrimSpace(candidate.Channel)
	if runChannel == "" {
		return true
	}
	return sessionChannel == runChannel
}

func scopeMatches(task taskpkg.Task, candidate SessionSnapshot) bool {
	scope := task.Scope.Normalize()
	workspaceID := strings.TrimSpace(task.WorkspaceID)
	sessionWorkspaceID := strings.TrimSpace(candidate.WorkspaceID)
	switch scope {
	case taskpkg.ScopeWorkspace:
		return workspaceID != "" && workspaceID == sessionWorkspaceID
	case taskpkg.ScopeGlobal:
		return sessionWorkspaceID == ""
	default:
		return false
	}
}

func capabilitiesCover(available []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	caps := make(map[string]struct{}, len(available))
	for _, value := range available {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			caps[trimmed] = struct{}{}
		}
	}
	for _, value := range required {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := caps[trimmed]; !ok {
			return false
		}
	}
	return true
}

func activeSessionIDs(active []taskpkg.Run) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, run := range active {
		switch run.Status.Normalize() {
		case taskpkg.TaskRunStatusClaimed, taskpkg.TaskRunStatusStarting, taskpkg.TaskRunStatusRunning:
		default:
			continue
		}
		if sessionID := strings.TrimSpace(run.SessionID); sessionID != "" {
			ids[sessionID] = struct{}{}
		}
	}
	return ids
}

func firstNotRecentlyWoken(
	now time.Time,
	cooldown time.Duration,
	state map[wakeKey]time.Time,
	work *RunSnapshot,
	candidates []SessionSnapshot,
) (SessionSnapshot, bool) {
	if work == nil {
		return SessionSnapshot{}, false
	}
	for _, candidate := range candidates {
		key := wakeKey{
			runID:     strings.TrimSpace(work.Run.ID),
			sessionID: strings.TrimSpace(candidate.ID),
		}
		last, exists := state[key]
		if !exists || cooldown <= 0 || now.Sub(last) >= cooldown {
			return candidate, true
		}
	}
	return SessionSnapshot{}, false
}

func sortRunsForWake(runs []RunSnapshot) {
	slices.SortStableFunc(runs, func(left, right RunSnapshot) int {
		if lv, rv := priorityValue(left.Task.Priority), priorityValue(right.Task.Priority); lv != rv {
			return rv - lv
		}
		if !left.Run.QueuedAt.Equal(right.Run.QueuedAt) {
			if left.Run.QueuedAt.Before(right.Run.QueuedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.Run.ID, right.Run.ID)
	})
}

func sortSessionsForWake(sessions []SessionSnapshot) {
	slices.SortStableFunc(sessions, func(left, right SessionSnapshot) int {
		if !left.CreatedAt.Equal(right.CreatedAt) {
			if left.CreatedAt.Before(right.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.ID, right.ID)
	})
}

func priorityValue(priority taskpkg.Priority) int {
	switch priority.Normalize() {
	case taskpkg.PriorityLow:
		return 10
	case taskpkg.PriorityHigh:
		return 30
	case taskpkg.PriorityUrgent:
		return 40
	default:
		return 20
	}
}

func runIDs(runs []RunSnapshot) []string {
	ids := make([]string, 0, len(runs))
	for idx := range runs {
		if id := strings.TrimSpace(runs[idx].Run.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func recoveredRunIDs(results []taskpkg.ExpiredLeaseRecoveryResult) []string {
	ids := make([]string, 0, len(results))
	for idx := range results {
		if id := strings.TrimSpace(results[idx].Run.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
