package core

import (
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	taskpkg "github.com/compozy/agh/internal/task"
)

// TaskRunSummaryPayloadFromSummary converts one operator-facing run summary into the shared payload.
func TaskRunSummaryPayloadFromSummary(summary *taskpkg.RunSummary) *contract.TaskRunSummaryPayload {
	if summary == nil {
		return nil
	}

	return &contract.TaskRunSummaryPayload{
		ID:                    summary.ID,
		TaskID:                summary.TaskID,
		Status:                summary.Status,
		Attempt:               summary.Attempt,
		PreviousRunID:         summary.PreviousRunID,
		FailureKind:           summary.FailureKind,
		MaxAttempts:           summary.MaxAttempts,
		SessionID:             summary.SessionID,
		ClaimedBy:             cloneActorIdentity(summary.ClaimedBy),
		ClaimTokenHash:        summary.ClaimTokenHash,
		LeaseUntil:            optionalTime(summary.LeaseUntil),
		HeartbeatAt:           optionalTime(summary.HeartbeatAt),
		CoordinationChannelID: taskRunSummaryChannel(summary),
		DesignationGroupID:    summary.DesignationGroupID,
		Designation:           cloneRunDesignationSummary(summary.Designation),
		QueuedAt:              summary.QueuedAt,
		ClaimedAt:             optionalTime(summary.ClaimedAt),
		StartedAt:             optionalTime(summary.StartedAt),
		EndedAt:               optionalTime(summary.EndedAt),
		Error:                 summary.Error,
	}
}

func taskRunSummaryChannel(summary *taskpkg.RunSummary) string {
	if summary == nil || summary.ResolvedNetworkParticipation == nil {
		return ""
	}
	return strings.TrimSpace(summary.ResolvedNetworkParticipation.ChannelID)
}
