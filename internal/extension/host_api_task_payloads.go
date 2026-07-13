package extensionpkg

import (
	"strings"

	apicontract "github.com/compozy/agh/internal/api/contract"
	taskpkg "github.com/compozy/agh/internal/task"
)

func taskSummaryPayloadFromSummary(record *taskpkg.Summary) apicontract.TaskSummaryPayload {
	if record == nil {
		return apicontract.TaskSummaryPayload{}
	}

	return apicontract.TaskSummaryPayload{
		ID:                 record.ID,
		Identifier:         record.Identifier,
		Scope:              record.Scope,
		WorkspaceID:        record.WorkspaceID,
		ParentTaskID:       record.ParentTaskID,
		NetworkChannel:     taskRunSummaryChannel(record.ActiveRun),
		Title:              record.Title,
		Priority:           record.Priority,
		MaxAttempts:        record.MaxAttempts,
		AutoEnqueueOnReady: record.AutoEnqueueOnReady,
		Status:             record.Status,
		ApprovalPolicy:     record.ApprovalPolicy,
		ApprovalState:      record.ApprovalState,
		Draft:              record.Draft,
		Owner:              cloneOwnership(record.Owner),
		CreatedBy:          record.CreatedBy,
		Origin:             record.Origin,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
		ClosedAt:           optionalTime(record.ClosedAt),
		ChildCount:         int(record.ChildCount),
		DependencyCount:    int(record.DependencyCount),
		Dependencies:       taskDependencyReferencePayloadsFromReferences(record.Dependencies),
		ActiveRun:          taskRunSummaryPayloadFromSummary(record.ActiveRun),
		LastActivityAt:     optionalTime(record.LastActivityAt),
	}
}

func taskPayloadFromTask(record *taskpkg.Task) apicontract.TaskPayload {
	if record == nil {
		return apicontract.TaskPayload{}
	}

	return apicontract.TaskPayload{
		ID:                 record.ID,
		Identifier:         record.Identifier,
		Scope:              record.Scope,
		WorkspaceID:        record.WorkspaceID,
		ParentTaskID:       record.ParentTaskID,
		Title:              record.Title,
		Description:        record.Description,
		Priority:           record.Priority,
		MaxAttempts:        record.MaxAttempts,
		AutoEnqueueOnReady: record.AutoEnqueueOnReady,
		Status:             record.Status,
		ApprovalPolicy:     record.ApprovalPolicy,
		ApprovalState:      record.ApprovalState,
		Owner:              cloneOwnership(record.Owner),
		CreatedBy:          record.CreatedBy,
		Origin:             record.Origin,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
		ClosedAt:           optionalTime(record.ClosedAt),
		Metadata:           cloneRawMessage(record.Metadata),
	}
}

func taskRunSummaryChannel(summary *taskpkg.RunSummary) string {
	if summary == nil || summary.ResolvedNetworkParticipation == nil {
		return ""
	}
	return strings.TrimSpace(summary.ResolvedNetworkParticipation.ChannelID)
}

func taskRunPayloadFromRun(run *taskpkg.Run) apicontract.TaskRunPayload {
	if run == nil {
		return apicontract.TaskRunPayload{}
	}

	return apicontract.TaskRunPayload{
		ID:             run.ID,
		TaskID:         run.TaskID,
		Status:         run.Status,
		Attempt:        int(run.Attempt),
		ClaimedBy:      cloneActorIdentity(run.ClaimedBy),
		SessionID:      run.SessionID,
		Origin:         run.Origin,
		IdempotencyKey: run.IdempotencyKey,
		NetworkChannel: run.NetworkSpecSnapshot().ChannelID,
		QueuedAt:       run.QueuedAt,
		ClaimedAt:      optionalTime(run.ClaimedAt),
		StartedAt:      optionalTime(run.StartedAt),
		EndedAt:        optionalTime(run.EndedAt),
		Error:          run.Error,
		Metadata:       cloneRawMessage(run.Metadata),
		Result:         cloneRawMessage(run.Result),
	}
}
