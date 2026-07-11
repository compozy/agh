package globaldb

import (
	"context"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
)

func validateClaimPreparedWorkRequest(req goal.ClaimPreparedWorkPromptRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.BindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || req.UsageBaseTokens < 0 ||
		(!req.UsageBaseReported && req.UsageBaseTokens != 0) ||
		strings.TrimSpace(req.ActorKind) == "" || strings.TrimSpace(req.ActorID) == "" {
		return fmt.Errorf("%w: Goal work claim identity is incomplete", looppkg.ErrValidation)
	}
	return nil
}

func validateClaimPreparedCompactionRequest(req goal.ClaimPreparedCompactionRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.BindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || req.UsageBaseTokens < 0 ||
		(!req.UsageBaseReported && req.UsageBaseTokens != 0) ||
		(req.UsageSequence != nil && *req.UsageSequence < 0) {
		return fmt.Errorf("%w: Goal compaction claim identity is incomplete", looppkg.ErrValidation)
	}
	return nil
}

func validateGoalDecisionGrant(
	checkpoint goal.Checkpoint,
	decision goal.BudgetDecision,
	turn int,
	work bool,
) error {
	if decision.GrantID == 0 {
		return nil
	}
	grant := checkpoint.ControlGrant
	if grant == nil || grant.Consumed || grant.Kind != goal.ControlGrantBudget ||
		grant.ID != decision.GrantID || grant.Turn != turn || grant.Turn != decision.GrantTurn ||
		grant.Scope != decision.GrantScope || grant.Cause != decision.Cause {
		return goalPromptFencedError("Goal budget grant identity changed")
	}
	if work && grant.Scope != goal.ControlGrantScopeWorkAndSettle {
		return goalPromptFencedError("settle-current grant cannot start Goal work")
	}
	if !work && grant.Scope != goal.ControlGrantScopeSettleCurrent &&
		grant.Scope != goal.ControlGrantScopeWorkAndSettle {
		return goalPromptFencedError("Goal grant does not authorize compaction")
	}
	return nil
}

func projectGoalCheckpointCounts(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
	status string,
	turnsUsed int,
	turnLimit int,
) error {
	_, err := exec.ExecContext(
		ctx,
		`UPDATE loop_generation_outputs
		 SET goal_status = ?, goal_turns_used = ?, goal_turn_limit = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?`,
		status,
		turnsUsed,
		turnLimit,
		string(key.LoopRunID),
		key.Generation,
		string(key.NodeID),
		key.ItemIndex,
	)
	if err != nil {
		return fmt.Errorf("store: project Goal checkpoint counts: %w", err)
	}
	return nil
}

func nullableGoalInt64Value(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
