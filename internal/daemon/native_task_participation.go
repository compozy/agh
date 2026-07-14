package daemon

import (
	"context"

	"github.com/compozy/agh/internal/network/participation"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (n *daemonNativeTools) applyNativeTaskNetworkParticipation(
	ctx context.Context,
	taskID string,
	request *participation.Request,
	actor taskpkg.ActorContext,
) error {
	if request == nil {
		return nil
	}
	profile, err := n.deps.Tasks.GetExecutionProfile(ctx, taskID, actor)
	if err != nil {
		return err
	}
	profile.NetworkParticipation = participation.CloneRequest(request)
	_, err = n.deps.Tasks.SetExecutionProfile(ctx, taskID, &profile, actor)
	return err
}

func (n *daemonNativeTools) updateNativeTask(
	ctx context.Context,
	input taskUpdateInput,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	var taskRecord *taskpkg.Task
	if input.hasRowFields() {
		updated, err := n.deps.Tasks.UpdateTask(ctx, input.TaskID, input.patch(), actor)
		if err != nil {
			return nil, err
		}
		taskRecord = updated
	} else {
		view, err := n.deps.Tasks.GetTask(ctx, input.TaskID, actor)
		if err != nil {
			return nil, err
		}
		cloned := view.Task
		taskRecord = &cloned
	}
	if err := n.applyNativeTaskNetworkParticipation(
		ctx,
		input.TaskID,
		input.NetworkParticipation,
		actor,
	); err != nil {
		return nil, err
	}
	return taskRecord, nil
}

func (i taskUpdateInput) hasRowFields() bool {
	return i.Title != nil ||
		i.Description != nil ||
		i.Priority != nil ||
		i.MaxAttempts != nil ||
		i.ApprovalPolicy != nil ||
		i.Metadata != nil ||
		i.Owner != nil ||
		i.ClearOwner
}
