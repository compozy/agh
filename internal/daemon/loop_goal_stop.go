package daemon

import (
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/session"
)

type loopManagedInputLeaseRevoker interface {
	RevokeManagedInput(session.ManagedInputOwner, string)
}

type loopGoalPromptLeaseRevoker struct {
	sessions loopManagedInputLeaseRevoker
}

var _ looppkg.GoalPromptLeaseRevoker = loopGoalPromptLeaseRevoker{}

func (r loopGoalPromptLeaseRevoker) RevokeGoalPromptLease(lease looppkg.GoalPromptLease, reason string) {
	if r.sessions == nil {
		return
	}
	r.sessions.RevokeManagedInput(session.ManagedInputOwner{
		QueueEntryID:  lease.QueueEntryID,
		SessionID:     lease.SessionID,
		OwnerKind:     lease.OwnerKind,
		LoopRunID:     lease.LoopRunID,
		TaskRunID:     lease.TaskRunID,
		RunGeneration: lease.RunGeneration,
		PromptAttempt: lease.PromptAttempt,
		ControlEpoch:  lease.ControlEpoch,
		BindingEpoch:  lease.BindingEpoch,
		PromptID:      lease.PromptID,
		PromptKind:    lease.PromptKind,
	}, reason)
}
