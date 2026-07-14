package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (r *networkWakeRunner) terminalizeWake(
	ctx context.Context,
	actor taskpkg.ActorContext,
	claim *taskpkg.ClaimResult,
	reservation store.WakeReservation,
	outcome store.NetworkWakeOutcome,
) error {
	terminalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
	defer cancel()
	tokensUsed := outcome.ActualInputTokens + outcome.ActualOutputTokens
	var runErr error
	if outcome.State == store.NetworkWakeStateSucceeded {
		payload, err := json.Marshal(map[string]string{"network_wake_id": reservation.WakeID})
		if err != nil {
			return fmt.Errorf("daemon: encode network wake result: %w", err)
		}
		_, runErr = r.tasks.CompleteRunLease(terminalCtx, taskpkg.LeaseCompletion{
			RunID:      claim.Run.ID,
			ClaimToken: claim.ClaimToken,
			Result:     taskpkg.RunResult{Value: payload, TokensUsed: tokensUsed},
			TokensUsed: tokensUsed,
			Now:        r.now().UTC(),
			Actor:      actor,
		}, actor)
	} else {
		_, runErr = r.tasks.FailRunLease(terminalCtx, taskpkg.LeaseFailure{
			RunID:      claim.Run.ID,
			ClaimToken: claim.ClaimToken,
			Failure:    taskpkg.RunFailure{Error: outcome.Reason},
			TokensUsed: tokensUsed,
			Now:        r.now().UTC(),
			Actor:      actor,
		}, actor)
	}
	if runErr != nil {
		terminalErr := fmt.Errorf("daemon: terminalize network wake task run: %w", runErr)
		if releaseErr := r.releaseClaimedWake(
			ctx,
			actor,
			claim,
			"network wake terminal transition failed",
		); releaseErr != nil {
			return errors.Join(terminalErr, releaseErr)
		}
		return terminalErr
	}
	if err := r.store.SettleNetworkWake(terminalCtx, reservation, outcome); err != nil {
		return fmt.Errorf("daemon: settle network wake ledger: %w", err)
	}
	return nil
}

func (r *networkWakeRunner) releaseClaimedWake(
	ctx context.Context,
	actor taskpkg.ActorContext,
	claim *taskpkg.ClaimResult,
	reason string,
) error {
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultShutdownTimeout)
	defer cancel()
	if _, err := r.tasks.ReleaseRunLease(releaseCtx, taskpkg.LeaseRelease{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Reason:     reason,
		Now:        r.now().UTC(),
	}, actor); err != nil {
		return fmt.Errorf("daemon: release network wake task run after failure: %w", err)
	}
	return nil
}

func (r *networkWakeRunner) failClaimedWake(
	ctx context.Context,
	actor taskpkg.ActorContext,
	claim *taskpkg.ClaimResult,
	reservation store.WakeReservation,
	reason string,
) error {
	return r.terminalizeWake(ctx, actor, claim, reservation, store.NetworkWakeOutcome{
		State:      store.NetworkWakeStateFailed,
		UsageState: store.NetworkWakeUsageUnavailable,
		Reason:     reason,
	})
}
