package loop

import (
	"context"
	"time"

	"github.com/compozy/agh/internal/task"
)

// GoalPromptLease is the neutral identity required to revoke one managed Goal prompt lease.
type GoalPromptLease struct {
	QueueEntryID  string
	SessionID     string
	OwnerKind     string
	LoopRunID     string
	TaskRunID     string
	RunGeneration int
	PromptAttempt int
	ControlEpoch  int64
	BindingEpoch  int64
	PromptID      string
	PromptKind    string
}

// GoalRunStopRequest atomically revokes Goal state and transitions its owning Run.
type GoalRunStopRequest struct {
	WorkspaceID    WorkspaceID
	RunID          RunID
	ExpectedStatus Status
	Actor          task.ActorContext
	StoppedAt      time.Time
}

// GoalRunStopResult carries exact in-memory leases that may be canceled after commit.
type GoalRunStopResult struct {
	RevokedPromptLeases []GoalPromptLease
}

// GoalRunStopStore is the optional atomic Goal-aware Run stop extension.
type GoalRunStopStore interface {
	StopGoalRun(context.Context, GoalRunStopRequest) (GoalRunStopResult, error)
}

// GoalPromptLeaseRevoker cancels one exact in-memory prompt lease after durable revocation commits.
type GoalPromptLeaseRevoker interface {
	RevokeGoalPromptLease(GoalPromptLease, string)
}

// GoalPromptLeaseRevokerFunc adapts a function to GoalPromptLeaseRevoker.
type GoalPromptLeaseRevokerFunc func(GoalPromptLease, string)

// RevokeGoalPromptLease implements GoalPromptLeaseRevoker.
func (f GoalPromptLeaseRevokerFunc) RevokeGoalPromptLease(lease GoalPromptLease, reason string) {
	if f != nil {
		f(lease, reason)
	}
}
