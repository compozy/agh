package loop

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/task"
)

// ClearInlineGoal hides the newest session-origin Goal and stops it first when still live.
func (s *service) ClearInlineGoal(
	ctx context.Context,
	ws WorkspaceID,
	originSessionID string,
	actor task.ActorContext,
) error {
	if strings.TrimSpace(string(ws)) == "" || strings.TrimSpace(originSessionID) == "" {
		return fmt.Errorf("%w: inline Goal clear identity is incomplete", ErrValidation)
	}
	if err := actor.Validate(); err != nil {
		return fmt.Errorf("%w: inline Goal clear actor: %w", ErrValidation, err)
	}
	clearStore, ok := s.store.(InlineGoalClearStore)
	if !ok {
		return fmt.Errorf("%w: inline Goal clear store is unavailable", ErrActionDependencyMissing)
	}
	clearedAt := s.now().UTC()
	result, err := clearStore.ClearInlineGoal(ctx, InlineGoalClearStoreRequest{
		WorkspaceID:     ws,
		OriginSessionID: strings.TrimSpace(originSessionID),
		Actor:           actor,
		ClearedAt:       clearedAt,
	})
	if err != nil {
		return err
	}
	if !result.Terminalized {
		return nil
	}
	var revokeErr error
	if s.goalLeaseRevoker != nil {
		for _, lease := range result.RevokedPromptLeases {
			if err := s.goalLeaseRevoker.RevokeGoalPromptLease(
				ctx,
				lease,
				string(TransitionCauseGoalClear),
			); err != nil {
				revokeErr = errors.Join(revokeErr, fmt.Errorf("loop: revoke cleared Goal runtime: %w", err))
			}
		}
	}
	s.dispatchCoordinatorTerminal(ctx, result.Run, TransitionCauseGoalClear, clearedAt)
	return revokeErr
}
