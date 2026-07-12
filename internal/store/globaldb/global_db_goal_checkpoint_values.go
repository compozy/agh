package globaldb

import (
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

func goalGrantID(grant *goal.ControlGrant) int64 {
	if grant == nil {
		return 0
	}
	return grant.ID
}

func goalGrantKind(grant *goal.ControlGrant) any {
	if grant == nil {
		return nil
	}
	return string(grant.Kind)
}

func goalGrantCause(grant *goal.ControlGrant) any {
	if grant == nil {
		return nil
	}
	return string(grant.Cause)
}

func goalGrantTurn(grant *goal.ControlGrant) any {
	if grant == nil {
		return nil
	}
	return grant.Turn
}

func goalGrantScope(grant *goal.ControlGrant) any {
	if grant == nil {
		return nil
	}
	return string(grant.Scope)
}

func goalGrantConsumed(grant *goal.ControlGrant) int {
	if grant == nil || grant.Consumed {
		return 1
	}
	return 0
}

func goalCancelPromptID(intent *goal.CompactionCancelIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.PromptID)
}

func goalCancelCause(intent *goal.CompactionCancelIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.Cause)
}

func goalCancelRequestedAt(intent *goal.CompactionCancelIntent) any {
	if intent == nil || intent.RequestedAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(intent.RequestedAt.UTC())
}

func goalReportPromptID(intent *goal.ReportIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.PromptID)
}

func goalReportStatus(intent *goal.ReportIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.Status)
}

func goalReportEvidence(intent *goal.ReportIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.EvidenceRef)
}

func goalReportBindingEpoch(intent *goal.ReportIntent) any {
	if intent == nil || intent.BindingEpoch < 1 {
		return nil
	}
	return intent.BindingEpoch
}

func goalReportActorKind(intent *goal.ReportIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.ActorKind)
}

func goalReportActorID(intent *goal.ReportIntent) any {
	if intent == nil {
		return nil
	}
	return nullableGoalString(intent.ActorID)
}

func goalReportRecordedAt(intent *goal.ReportIntent) any {
	if intent == nil || intent.RecordedAt.IsZero() {
		return nil
	}
	return store.FormatTimestamp(intent.RecordedAt.UTC())
}
