package globaldb

import (
	"database/sql"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
	taskpkg "github.com/compozy/agh/internal/task"
)

func forceTaskRunSnapshotParams(previous taskpkg.Run, next taskpkg.Run) sqlcgen.ForceUpdateTaskRunSnapshotParams {
	lineage := taskRunReviewLineage(next)
	return sqlcgen.ForceUpdateTaskRunSnapshotParams{
		TaskID:                 next.TaskID,
		Status:                 next.Status.String(),
		Attempt:                int64(next.Attempt),
		PreviousRunID:          nullableTaskString(next.PreviousRunID),
		FailureKind:            strings.TrimSpace(next.FailureKind),
		ClaimedByKind:          nullableTaskActorKind(next.ClaimedBy),
		ClaimedByRef:           nullableTaskActorRef(next.ClaimedBy),
		SessionID:              nullableTaskString(next.SessionID),
		OriginKind:             string(next.Origin.Kind),
		OriginRef:              next.Origin.Ref,
		IdempotencyKey:         nullableTaskString(next.IdempotencyKey),
		NetworkChannel:         nullableTaskString(next.NetworkChannel),
		ClaimTokenHash:         nullableTaskString(next.ClaimTokenHash),
		LeaseUntil:             nullableTaskTime(next.LeaseUntil),
		HeartbeatAt:            nullableTaskTime(next.HeartbeatAt),
		CoordinationChannelID:  nullableTaskString(next.CoordinationChannelID),
		QueuedAt:               store.FormatTimestamp(next.QueuedAt),
		ClaimedAt:              nullableTaskTime(next.ClaimedAt),
		StartedAt:              nullableTaskTime(next.StartedAt),
		EndedAt:                nullableTaskTime(next.EndedAt),
		Error:                  nullableTaskString(next.Error),
		MetadataJson:           nullableTaskRawJSON(next.Metadata),
		ResultJson:             nullableTaskRawJSON(next.Result),
		ReviewRequired:         lineage.Required,
		ReviewRequestRound:     int64(lineage.RequestRound),
		ReviewPolicySnapshot:   string(lineage.PolicySnapshot),
		ReviewRequestID:        nullableTaskString(lineage.RequestID),
		ParentRunID:            nullableTaskString(lineage.ParentRunID),
		ReviewID:               nullableTaskString(lineage.ReviewID),
		ReviewRound:            int64(lineage.ReviewRound),
		ContinuationReason:     lineage.ContinuationReason,
		MissingWorkJson:        string(lineage.MissingWork),
		NextRoundGuidance:      lineage.NextRoundGuidance,
		ID:                     next.ID,
		PreviousStatus:         previous.Status.Normalize().String(),
		PreviousSessionID:      sql.NullString{String: strings.TrimSpace(previous.SessionID), Valid: true},
		PreviousClaimTokenHash: sql.NullString{String: strings.TrimSpace(previous.ClaimTokenHash), Valid: true},
		PreviousLeaseUntil:     sql.NullString{String: forceRunCASTimestamp(previous.LeaseUntil), Valid: true},
	}
}
