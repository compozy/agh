package task

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/network/participation"
)

func (m *Service) resolveQueuedRunParticipation(
	ctx context.Context,
	taskRecord Task,
	runID string,
	runKind RunKind,
	loopRunID string,
	request *participation.Request,
	requestSource participation.Source,
) (participation.Spec, error) {
	definition, err := m.taskParticipationDefinition(ctx, taskRecord.ID)
	if err != nil {
		return participation.Spec{}, err
	}
	if m.participationResolver == nil {
		if request == nil && definition == nil {
			return participation.Spec{
				Version: participation.SpecVersion,
				Mode:    participation.ModeLocal,
				Source:  participation.SourceBuiltInLocal,
			}, nil
		}
		return participation.Spec{}, fmt.Errorf(
			"task: participation resolver is required for task run %q with network intent",
			runID,
		)
	}
	return m.participationResolver.Resolve(ctx, participation.ResolveInput{
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
		Owner: participation.OwnerRef{
			Kind: participation.OwnerKindTaskRun,
			ID:   strings.TrimSpace(runID),
		},
		Request:       request,
		RequestSource: requestSource,
		Definition:    definition,
		RunID:         strings.TrimSpace(runID),
		LoopRunID:     strings.TrimSpace(loopRunID),
		Coordinated: taskRecord.Scope.Normalize() == ScopeWorkspace &&
			runKind.Normalize() == RunKindWorker && strings.TrimSpace(loopRunID) == "",
	})
}

func (m *Service) taskParticipationDefinition(
	ctx context.Context,
	taskID string,
) (*participation.Request, error) {
	profile, err := m.store.GetExecutionProfile(ctx, strings.TrimSpace(taskID))
	switch {
	case errors.Is(err, ErrExecutionProfileNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	case profile.NetworkParticipation == nil:
		return nil, nil
	default:
		request := *profile.NetworkParticipation
		return &request, nil
	}
}

func (m *Service) existingQueuedRun(
	ctx context.Context,
	taskID string,
	idempotencyKey string,
	origin Origin,
) (*Run, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, false, nil
	}
	run, err := m.store.GetTaskRunByIdempotencyKey(ctx, idempotencyKey, origin)
	switch {
	case errors.Is(err, ErrTaskRunIdempotencyNotFound):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	case strings.TrimSpace(run.TaskID) != strings.TrimSpace(taskID):
		return nil, false, fmt.Errorf(
			"%w: idempotency key %q is already bound to task %q",
			ErrValidation,
			idempotencyKey,
			run.TaskID,
		)
	default:
		return &run, true, nil
	}
}
