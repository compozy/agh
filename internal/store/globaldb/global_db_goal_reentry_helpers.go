package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
)

type goalReentryRow struct {
	state         looppkg.GoalControlState
	outputStatus  string
	outputTaskRun string
	grantID       sql.NullInt64
	grantKind     sql.NullString
	grantCause    sql.NullString
	grantTurn     sql.NullInt64
	grantScope    sql.NullString
	grantConsumed sql.NullInt64
}

func goalReentryStillAwaiting(current goalReentryRow, expected looppkg.GoalControlState) bool {
	return current.state.ControlEpoch == expected.ControlEpoch &&
		current.state.RunStatus == expected.RunStatus &&
		current.outputStatus == looppkg.GenerationOutputStatusAwaitingGoal &&
		strings.TrimSpace(current.outputTaskRun) == strings.TrimSpace(expected.TaskRunID)
}

func loadAwaitingGoalControlWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID looppkg.WorkspaceID,
	runID looppkg.RunID,
) (goalReentryRow, bool, error) {
	row, err := scanGoalReentryRow(exec.QueryRowContext(
		ctx,
		`SELECT c.generation, c.node_id, c.item_index, c.control_epoch, c.goal_status,
		        COALESCE(c.control_cause, ''),
		        c.turns_used, c.turn_limit, COALESCE(c.task_run_id, ''), COALESCE(c.queue_entry_id, ''),
		        COALESCE(c.prompt_id, ''), COALESCE(c.prompt_kind, ''),
		        EXISTS (
		          SELECT 1 FROM loop_goal_turns t
		          WHERE t.loop_run_id = c.loop_run_id AND t.generation = c.generation
		            AND t.node_id = c.node_id AND t.item_index = c.item_index
		            AND t.result_status IS NULL
		        ),
		        l.status, l.active_gate_id, o.status, COALESCE(o.task_run_id, ''),
		        c.control_grant_id, c.control_grant_kind, c.control_grant_cause,
		        c.control_grant_turn, c.control_grant_scope, c.control_grant_consumed
		 FROM loop_goal_checkpoints c
		 JOIN loop_runs l ON l.id = c.loop_run_id
		 JOIN loop_generation_outputs o
		   ON o.loop_run_id = c.loop_run_id
		  AND o.generation = c.generation
		  AND o.node_id = c.node_id
		  AND o.item_index = c.item_index
		 WHERE c.loop_run_id = ?
		   AND l.workspace_id = ?
		   AND c.phase = 'awaiting_control'
		   AND o.status = ?
		 ORDER BY c.generation ASC, c.node_id ASC, c.item_index ASC
		 LIMIT 1`,
		string(runID),
		string(workspaceID),
		looppkg.GenerationOutputStatusAwaitingGoal,
	), workspaceID, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return goalReentryRow{}, false, nil
	}
	if err != nil {
		return goalReentryRow{}, false, fmt.Errorf("store: load awaiting Goal control: %w", err)
	}
	if err := finalizeGoalReentryState(&row); err != nil {
		return goalReentryRow{}, false, err
	}
	return row, true, nil
}

func loadGoalReentryRowWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	state looppkg.GoalControlState,
) (goalReentryRow, error) {
	row, err := scanGoalReentryRow(exec.QueryRowContext(
		ctx,
		`SELECT c.generation, c.node_id, c.item_index, c.control_epoch, c.goal_status,
		        COALESCE(c.control_cause, ''),
		        c.turns_used, c.turn_limit, COALESCE(c.task_run_id, ''), COALESCE(c.queue_entry_id, ''),
		        COALESCE(c.prompt_id, ''), COALESCE(c.prompt_kind, ''),
		        EXISTS (
		          SELECT 1 FROM loop_goal_turns t
		          WHERE t.loop_run_id = c.loop_run_id AND t.generation = c.generation
		            AND t.node_id = c.node_id AND t.item_index = c.item_index
		            AND t.result_status IS NULL
		        ),
		        l.status, l.active_gate_id, o.status, COALESCE(o.task_run_id, ''),
		        c.control_grant_id, c.control_grant_kind, c.control_grant_cause,
		        c.control_grant_turn, c.control_grant_scope, c.control_grant_consumed
		 FROM loop_goal_checkpoints c
		 JOIN loop_runs l ON l.id = c.loop_run_id
		 JOIN loop_generation_outputs o
		   ON o.loop_run_id = c.loop_run_id
		  AND o.generation = c.generation
		  AND o.node_id = c.node_id
		  AND o.item_index = c.item_index
		 WHERE c.loop_run_id = ?
		   AND l.workspace_id = ?
		   AND c.generation = ?
		   AND c.node_id = ?
		   AND c.item_index = ?`,
		string(state.LoopRunID),
		string(state.WorkspaceID),
		state.Generation,
		string(state.NodeID),
		state.ItemIndex,
	), state.WorkspaceID, state.LoopRunID)
	if errors.Is(err, sql.ErrNoRows) {
		return goalReentryRow{}, goalReentryStaleError("Goal checkpoint no longer exists")
	}
	if err != nil {
		return goalReentryRow{}, fmt.Errorf("store: load Goal reentry checkpoint: %w", err)
	}
	return row, nil
}

func scanGoalReentryRow(
	row generationOutputScanner,
	workspaceID looppkg.WorkspaceID,
	runID looppkg.RunID,
) (goalReentryRow, error) {
	var value goalReentryRow
	value.state.WorkspaceID = workspaceID
	value.state.LoopRunID = runID
	if err := row.Scan(
		&value.state.Generation,
		&value.state.NodeID,
		&value.state.ItemIndex,
		&value.state.ControlEpoch,
		&value.state.GoalStatus,
		&value.state.Cause,
		&value.state.TurnsUsed,
		&value.state.TurnLimit,
		&value.state.TaskRunID,
		&value.state.QueueEntryID,
		&value.state.PromptID,
		&value.state.PromptKind,
		&value.state.OpenTurn,
		&value.state.RunStatus,
		&value.state.GateID,
		&value.outputStatus,
		&value.outputTaskRun,
		&value.grantID,
		&value.grantKind,
		&value.grantCause,
		&value.grantTurn,
		&value.grantScope,
		&value.grantConsumed,
	); err != nil {
		return goalReentryRow{}, err
	}
	return value, nil
}

func finalizeGoalReentryState(row *goalReentryRow) error {
	if row == nil {
		return fmt.Errorf("%w: Goal reentry row is required", looppkg.ErrValidation)
	}
	row.state.Cause = goalControlCause(row.state)
	if row.state.Cause == "" || strings.TrimSpace(row.state.TaskRunID) == "" ||
		strings.TrimSpace(row.outputTaskRun) != strings.TrimSpace(row.state.TaskRunID) {
		return goalReentryStaleError("Goal reentry correlation is incomplete")
	}
	return nil
}

func goalControlCause(state looppkg.GoalControlState) looppkg.ReasonCode {
	if strings.TrimSpace(string(state.Cause)) != "" {
		return looppkg.ReasonCode(strings.TrimSpace(string(state.Cause)))
	}
	if state.RunStatus == looppkg.StatusPaused {
		return "operator_resume"
	}
	if state.RunStatus != looppkg.StatusNeedsApproval {
		return ""
	}
	prefix := fmt.Sprintf(
		"goal:%s:%d:%d:",
		strings.TrimSpace(string(state.NodeID)),
		state.Generation,
		state.ItemIndex,
	)
	gateID := strings.TrimSpace(string(state.GateID))
	if !strings.HasPrefix(gateID, prefix) {
		return ""
	}
	return looppkg.ReasonCode(strings.TrimSpace(strings.TrimPrefix(gateID, prefix)))
}

func validateGoalReactivationRequest(req looppkg.GoalReactivationRequest) error {
	if !goalReactivationStateValid(req.State) {
		return fmt.Errorf("%w: Goal reactivation state is incomplete", looppkg.ErrValidation)
	}
	if err := req.Actor.Validate(); err != nil {
		return fmt.Errorf("%w: Goal reactivation actor: %w", looppkg.ErrValidation, err)
	}
	if !goalGrantScopeValid(req) {
		return fmt.Errorf("%w: Goal grant kind/scope is invalid", looppkg.ErrValidation)
	}
	return nil
}

func goalReactivationStateValid(state looppkg.GoalControlState) bool {
	return strings.TrimSpace(string(state.WorkspaceID)) != "" &&
		strings.TrimSpace(string(state.LoopRunID)) != "" &&
		state.Generation >= 1 &&
		strings.TrimSpace(string(state.NodeID)) != "" &&
		state.ItemIndex >= 0 &&
		state.ControlEpoch >= 1 &&
		strings.TrimSpace(state.TaskRunID) != "" &&
		state.Cause != ""
}

func goalGrantScopeValid(req looppkg.GoalReactivationRequest) bool {
	switch req.Kind {
	case looppkg.GoalGrantPlainResume:
		return req.Scope == looppkg.GoalGrantScopeReactivate && req.TurnIncrement == 0
	case looppkg.GoalGrantTurnExtension:
		return req.Scope == looppkg.GoalGrantScopeTurnLimit && req.TurnIncrement > 0
	case looppkg.GoalGrantBudget:
		return (req.Scope == looppkg.GoalGrantScopeSettleCurrent ||
			req.Scope == looppkg.GoalGrantScopeWorkAndSettle) && req.TurnIncrement == 0 &&
			((req.Scope == looppkg.GoalGrantScopeSettleCurrent) == req.State.OpenTurn)
	case looppkg.GoalGrantReseed:
		return req.Scope == looppkg.GoalGrantScopeRotateBinding && req.TurnIncrement == 0
	default:
		return false
	}
}

func goalReentryStaleError(detail string) error {
	return &looppkg.ReasonError{
		Code: looppkg.ReasonCodeGoalControlStale,
		Err:  fmt.Errorf("%w: %s", looppkg.ErrTransitionConflict, detail),
	}
}

func goalSuccessorMetadata(raw json.RawMessage, controlEpoch int64) (json.RawMessage, error) {
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("store: decode Goal worker metadata: %w", err)
	}
	metadata["goal_segment_epoch"] = controlEpoch
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("store: encode Goal worker metadata: %w", err)
	}
	return encoded, nil
}

func goalReentryGrantConsumed(kind looppkg.GoalGrantKind) bool {
	return kind == looppkg.GoalGrantTurnExtension || kind == looppkg.GoalGrantPlainResume
}

func goalReentryGrantTurn(req looppkg.GoalReactivationRequest) int {
	turn := req.State.TurnsUsed
	if req.Kind == looppkg.GoalGrantBudget && req.Scope == looppkg.GoalGrantScopeWorkAndSettle {
		turn++
	}
	return turn
}

func normalizeGoalReentryDecisions(
	records []looppkg.GateDecisionRecord,
	now func() time.Time,
) ([]looppkg.GateDecisionRecord, error) {
	if len(records) == 0 {
		return nil, nil
	}
	return normalizeLoopGateDecisionRecords(records, now())
}
