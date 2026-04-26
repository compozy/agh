package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jonboulle/clockwork"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	defaultInterval     = 15 * time.Second
	defaultWakeCooldown = time.Minute
	defaultSweepLimit   = 100
	defaultSweepReason  = "scheduler_sweep"
	defaultWakeReason   = "pending_task_run"
)

var (
	// ErrStopped reports that a stopped scheduler cannot be restarted.
	ErrStopped = errors.New("scheduler: stopped")
)

// TaskSource provides durable task-run snapshots and lease recovery.
type TaskSource interface {
	PendingRuns(ctx context.Context) ([]RunSnapshot, error)
	ActiveRuns(ctx context.Context) ([]taskpkg.Run, error)
	RecoverExpiredRunLeases(
		ctx context.Context,
		recovery taskpkg.ExpiredLeaseRecovery,
		actor taskpkg.ActorContext,
	) ([]taskpkg.ExpiredLeaseRecoveryResult, error)
}

// SessionSource provides live runtime sessions that may be notified.
type SessionSource interface {
	Sessions(ctx context.Context) ([]SessionSnapshot, error)
}

// Waker sends an advisory notification to one selected idle session.
type Waker interface {
	Wake(ctx context.Context, target *WakeTarget) error
}

// RunSnapshot joins a durable run with its owning task.
type RunSnapshot struct {
	Task taskpkg.Task
	Run  taskpkg.Run
}

// SessionSnapshot is the scheduler's rebuildable view of one live session.
type SessionSnapshot struct {
	ID           string
	AgentName    string
	WorkspaceID  string
	Channel      string
	State        string
	Prompting    bool
	Capabilities []string
	CreatedAt    time.Time
}

// WakeTarget records the exact run/session pair selected for notification.
type WakeTarget struct {
	Work    RunSnapshot
	Session SessionSnapshot
	Reason  string
}

// CycleResult reports one mechanical scheduler pass.
type CycleResult struct {
	PendingRuns      int
	ActiveRuns       int
	SessionsScanned  int
	RecoveredLeases  int
	WakeAttempts     int
	WakeSucceeded    int
	WakeFailed       int
	NoMatchRuns      int
	RecentlyNotified int
	UnclaimableRuns  int
	SelectedRunIDs   []string
	NoMatchRunIDs    []string
	RecoveredRunIDs  []string
}

// RebuildResult reports the durable state discovered while rebuilding
// scheduler-owned ephemeral state.
type RebuildResult struct {
	PendingRuns     int
	ActiveRuns      int
	SessionsScanned int
	ClearedWakeKeys int
	RebuiltAt       time.Time
}

// Stats is a lock-protected snapshot of scheduler observability counters.
type Stats struct {
	Cycles            int
	Rebuilds          int
	RecoveredLeases   int
	RecoveryErrors    int
	WakeAttempts      int
	WakeSucceeded     int
	WakeFailed        int
	NoMatchRuns       int
	RecentlyNotified  int
	UnclaimableRuns   int
	LastCycleAt       time.Time
	LastRebuildAt     time.Time
	LastRecoveryError string
	LastWakeError     string
}

// Option customizes scheduler runtime behavior.
type Option func(*Scheduler)

// WithLogger overrides the scheduler logger.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Scheduler) {
		s.logger = logger
	}
}

// WithClock overrides the scheduler clock, mainly for deterministic tests.
func WithClock(clock clockwork.Clock) Option {
	return func(s *Scheduler) {
		s.clock = clock
	}
}

// WithInterval overrides the background sweep/notify interval.
func WithInterval(interval time.Duration) Option {
	return func(s *Scheduler) {
		s.interval = interval
	}
}

// WithWakeCooldown overrides how long a run/session wake key is suppressed
// before the scheduler may notify that same session again.
func WithWakeCooldown(cooldown time.Duration) Option {
	return func(s *Scheduler) {
		s.wakeCooldown = cooldown
	}
}

// WithSweepReason overrides the task-service recovery reason.
func WithSweepReason(reason string) Option {
	return func(s *Scheduler) {
		s.sweepReason = reason
	}
}

// WithWakeReason overrides the synthetic wake metadata reason.
func WithWakeReason(reason string) Option {
	return func(s *Scheduler) {
		s.wakeReason = reason
	}
}

// WithSweepLimit overrides the maximum expired leases recovered per pass.
func WithSweepLimit(limit int) Option {
	return func(s *Scheduler) {
		s.sweepLimit = limit
	}
}

// WithActor overrides the daemon actor used for recovery writes.
func WithActor(actor taskpkg.ActorContext) Option {
	return func(s *Scheduler) {
		s.actor = actor
	}
}
