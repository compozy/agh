package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
)

var _ goal.ToolStore = (*GlobalDB)(nil)

// FindGoalReportTarget returns the only currently prompting Goal bound to the session.
func (g *GlobalDB) FindGoalReportTarget(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (goal.ToolReportTarget, bool, error) {
	if err := g.checkReady(ctx, "find Goal report target"); err != nil {
		return goal.ToolReportTarget{}, false, err
	}
	return findGoalReportTargetWithExecutor(ctx, g.db, workspaceID, sessionID)
}

func findGoalReportTargetWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (target goal.ToolReportTarget, found bool, err error) {
	workspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(workspaceID)))
	sessionID = strings.TrimSpace(sessionID)
	if workspaceID == "" || sessionID == "" {
		return goal.ToolReportTarget{}, false, fmt.Errorf(
			"%w: Goal report workspace_id and session_id are required",
			looppkg.ErrValidation,
		)
	}
	rows, err := exec.QueryContext(
		ctx,
		`SELECT checkpoints.loop_run_id, checkpoints.generation, checkpoints.node_id,
		        checkpoints.item_index, checkpoints.control_epoch, checkpoints.binding_epoch,
		        checkpoints.prompt_id, runs.origin_session_id, checkpoints.session_id
		 FROM loop_goal_checkpoints AS checkpoints
		 JOIN loop_runs AS runs ON runs.id = checkpoints.loop_run_id
		 JOIN loop_session_bindings AS bindings
		   ON bindings.loop_run_id = checkpoints.loop_run_id
		  AND bindings.handle = checkpoints.binding_handle
		  AND bindings.binding_epoch = checkpoints.binding_epoch
		  AND bindings.session_id = checkpoints.session_id
		 WHERE runs.workspace_id = ?
		   AND runs.origin_kind = 'session'
		   AND runs.goal_cleared_at IS NULL
		   AND runs.status IN ('queued','running','watching','needs-approval','paused')
		   AND checkpoints.phase = 'prompting'
		   AND checkpoints.goal_status = 'active'
		   AND checkpoints.session_id = ?
		   AND checkpoints.prompt_id IS NOT NULL
		   AND bindings.workspace_id = ?
		   AND bindings.state = 'active'
		 ORDER BY checkpoints.updated_at DESC, runs.created_at DESC, runs.rowid DESC
		 LIMIT 2`,
		string(workspaceID),
		sessionID,
		string(workspaceID),
	)
	if err != nil {
		return goal.ToolReportTarget{}, false, fmt.Errorf("store: query Goal report target: %w", err)
	}
	defer func() {
		err = joinRowsCloseError(rows, err, "Goal report target")
	}()
	for rows.Next() {
		var candidate goal.ToolReportTarget
		candidate.Key.WorkspaceID = workspaceID
		if scanErr := rows.Scan(
			&candidate.Key.LoopRunID,
			&candidate.Key.Generation,
			&candidate.Key.NodeID,
			&candidate.Key.ItemIndex,
			&candidate.ExpectedControlEpoch,
			&candidate.ExpectedBindingEpoch,
			&candidate.PromptID,
			&candidate.OriginSessionID,
			&candidate.BoundSessionID,
		); scanErr != nil {
			return goal.ToolReportTarget{}, false, fmt.Errorf("store: scan Goal report target: %w", scanErr)
		}
		if validateErr := candidate.Validate(); validateErr != nil {
			return goal.ToolReportTarget{}, false, validateErr
		}
		if found {
			return goal.ToolReportTarget{}, false, fmt.Errorf(
				"store: multiple active Goal report targets for session %q",
				sessionID,
			)
		}
		target = candidate
		found = true
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return goal.ToolReportTarget{}, false, fmt.Errorf("store: iterate Goal report targets: %w", rowsErr)
	}
	return target, found, nil
}

// ResolveActiveGoalOriginAlias resolves an active moved binding back to its origin session.
func (g *GlobalDB) ResolveActiveGoalOriginAlias(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (originSessionID string, found bool, err error) {
	if err := g.checkReady(ctx, "resolve active Goal origin alias"); err != nil {
		return "", false, err
	}
	workspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(workspaceID)))
	sessionID = strings.TrimSpace(sessionID)
	if workspaceID == "" || sessionID == "" {
		return "", false, fmt.Errorf(
			"%w: Goal alias workspace_id and session_id are required",
			looppkg.ErrValidation,
		)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT runs.origin_session_id
		 FROM loop_session_bindings AS bindings
		 JOIN loop_runs AS runs ON runs.id = bindings.loop_run_id
		 WHERE bindings.workspace_id = ?
		   AND bindings.session_id = ?
		   AND bindings.state = 'active'
		   AND runs.workspace_id = ?
		   AND runs.origin_kind = 'session'
		   AND runs.origin_session_id <> bindings.session_id
		   AND runs.goal_cleared_at IS NULL
		   AND runs.status IN ('queued','running','watching','needs-approval','paused')
		 ORDER BY bindings.activated_at DESC, runs.created_at DESC, runs.rowid DESC
		 LIMIT 2`,
		string(workspaceID),
		sessionID,
		string(workspaceID),
	)
	if err != nil {
		return "", false, fmt.Errorf("store: query active Goal origin alias: %w", err)
	}
	defer func() {
		err = joinRowsCloseError(rows, err, "active Goal origin alias")
	}()
	for rows.Next() {
		var candidate string
		if scanErr := rows.Scan(&candidate); scanErr != nil {
			return "", false, fmt.Errorf("store: scan active Goal origin alias: %w", scanErr)
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return "", false, fmt.Errorf("%w: Goal origin session is empty", looppkg.ErrValidation)
		}
		if found {
			return "", false, fmt.Errorf(
				"store: multiple active Goal origin aliases for session %q",
				sessionID,
			)
		}
		originSessionID = candidate
		found = true
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return "", false, fmt.Errorf("store: iterate active Goal origin aliases: %w", rowsErr)
	}
	return originSessionID, found, nil
}

// RecordGoalReport content-addresses evidence and records its prompt-bound intent atomically.
func (g *GlobalDB) RecordGoalReport(
	ctx context.Context,
	req goal.RecordToolReportRequest,
) (goal.ReportIntent, error) {
	if err := g.checkReady(ctx, "record Goal tool report"); err != nil {
		return goal.ReportIntent{}, err
	}
	payload, evidenceRef, err := normalizeGoalToolReportRequest(&req)
	if err != nil {
		return goal.ReportIntent{}, err
	}
	now := g.now()
	var intent goal.ReportIntent
	err = g.withTaskImmediateTransaction(ctx, "record Goal tool report", func(exec taskSQLExecutor) error {
		current, found, err := findGoalReportTargetWithExecutor(
			ctx,
			exec,
			req.Target.Key.WorkspaceID,
			req.Target.BoundSessionID,
		)
		if err != nil {
			return err
		}
		if !found || !goalToolReportTargetEqual(current, req.Target) {
			return goalNotActiveError("report target is no longer current")
		}
		if len(payload) > 0 {
			if err := upsertLoopOutputBlobWithExecutor(ctx, exec, evidenceRef, payload, now); err != nil {
				return err
			}
		}
		intent, err = recordReportIntentWithExecutor(ctx, exec, goal.RecordReportIntentRequest{
			Key: req.Target.Key, ExpectedControlEpoch: req.Target.ExpectedControlEpoch,
			ExpectedBindingEpoch: req.Target.ExpectedBindingEpoch, PromptID: req.Target.PromptID,
			Status: req.Status, EvidenceRef: evidenceRef, ActorKind: req.ActorKind, ActorID: req.ActorID,
		}, now)
		return err
	})
	if err != nil {
		return goal.ReportIntent{}, normalizeGoalToolReportError(err)
	}
	return intent, nil
}

func normalizeGoalToolReportRequest(req *goal.RecordToolReportRequest) (json.RawMessage, string, error) {
	if req == nil {
		return nil, "", fmt.Errorf("%w: Goal tool report request is required", looppkg.ErrValidation)
	}
	req.Target.Key.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(req.Target.Key.WorkspaceID)))
	req.Target.Key.LoopRunID = looppkg.RunID(strings.TrimSpace(string(req.Target.Key.LoopRunID)))
	req.Target.Key.NodeID = looppkg.NodeID(strings.TrimSpace(string(req.Target.Key.NodeID)))
	req.Target.PromptID = strings.TrimSpace(req.Target.PromptID)
	req.Target.OriginSessionID = strings.TrimSpace(req.Target.OriginSessionID)
	req.Target.BoundSessionID = strings.TrimSpace(req.Target.BoundSessionID)
	req.Status = strings.TrimSpace(req.Status)
	req.ActorKind = strings.TrimSpace(req.ActorKind)
	req.ActorID = strings.TrimSpace(req.ActorID)
	if err := req.Target.Validate(); err != nil {
		return nil, "", err
	}
	if len(req.Evidence) > goal.MaxReportEvidenceBytes {
		return nil, "", &looppkg.ReasonError{
			Code: looppkg.ReasonCodeGoalEvidenceTooLarge,
			Err: fmt.Errorf(
				"%w: Goal report evidence exceeds %d bytes",
				looppkg.ErrValidation,
				goal.MaxReportEvidenceBytes,
			),
		}
	}
	req.Evidence = strings.TrimSpace(req.Evidence)
	if (req.Status != goalReportStatusComplete && req.Status != goalReportStatusBlocked) ||
		req.ActorKind == "" || req.ActorID == "" ||
		(req.Status == goalReportStatusBlocked && req.Evidence == "") {
		return nil, "", fmt.Errorf("%w: Goal tool report is invalid", looppkg.ErrValidation)
	}
	if req.Evidence == "" {
		return nil, "", nil
	}
	payload, err := json.Marshal(req.Evidence)
	if err != nil {
		return nil, "", fmt.Errorf("store: encode Goal report evidence: %w", err)
	}
	raw := json.RawMessage(payload)
	return raw, looppkg.OutputRefForPayload(raw), nil
}

func goalToolReportTargetEqual(left goal.ToolReportTarget, right goal.ToolReportTarget) bool {
	return left.Key == right.Key &&
		left.ExpectedControlEpoch == right.ExpectedControlEpoch &&
		left.ExpectedBindingEpoch == right.ExpectedBindingEpoch &&
		left.PromptID == right.PromptID &&
		left.OriginSessionID == right.OriginSessionID &&
		left.BoundSessionID == right.BoundSessionID
}

func normalizeGoalToolReportError(err error) error {
	var reasonErr *looppkg.ReasonError
	if !errors.As(err, &reasonErr) {
		return err
	}
	switch reasonErr.Code {
	case looppkg.ReasonCodeContinuousBindingMismatch, looppkg.ReasonCodeGoalControlStale:
		return goalNotActiveError("report target was revoked before persistence")
	default:
		return err
	}
}
