package core

import (
	"context"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/network/participation"
	taskpkg "github.com/compozy/agh/internal/task"
)

// taskUpdateHasRowFields reports whether the patch mutates task-row fields
// (everything except network_participation, which lives on the execution profile).
func taskUpdateHasRowFields(req contract.UpdateTaskRequest) bool {
	return req.Title != nil ||
		req.Description != nil ||
		req.Priority != nil ||
		req.MaxAttempts != nil ||
		req.AutoEnqueueOnReady != nil ||
		req.ApprovalPolicy != nil ||
		req.Metadata != nil ||
		req.Owner != nil ||
		req.ClearOwner
}

// applyTaskNetworkParticipation persists task-definition network intent on the
// execution-profile owning stream.
func applyTaskNetworkParticipation(
	ctx context.Context,
	manager TaskService,
	taskID string,
	request *participation.Request,
	actor taskpkg.ActorContext,
) error {
	if request == nil {
		return nil
	}
	profile, err := manager.GetExecutionProfile(ctx, taskID, actor)
	if err != nil {
		return err
	}
	profile.NetworkParticipation = participation.CloneRequest(request)
	_, err = manager.SetExecutionProfile(ctx, taskID, &profile, actor)
	return err
}
