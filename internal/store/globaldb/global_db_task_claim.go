package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	globalDBTaskClaimStatusKey = "status"
)

const (
	globalDBTaskClaimHandoffKey = "handoff"
)

type taskRunLeaseSnapshot struct {
	status         taskpkg.RunStatus
	sessionID      string
	leaseUntil     time.Time
	claimTokenHash string
}

const taskPriorityValueSQL = `CASE t.priority
	WHEN 'urgent' THEN 40
	WHEN 'high' THEN 30
	WHEN 'low' THEN 10
	ELSE 20
END`

// ClaimNextRun atomically selects and claims the next eligible queued task run.
func (g *GlobalDB) ClaimNextRun(ctx context.Context, criteria taskpkg.ClaimCriteria) (taskpkg.ClaimResult, error) {
	if err := g.checkReady(ctx, "claim next task run"); err != nil {
		return taskpkg.ClaimResult{}, err
	}
	normalized, err := criteria.Normalize(g.now())
	if err != nil {
		return taskpkg.ClaimResult{}, err
	}

	var result taskpkg.ClaimResult
	if err := g.withTaskImmediateTransaction(ctx, "claim next task run", func(exec taskSQLExecutor) error {
		if err := g.ensureClaimerHasNoActiveLease(ctx, exec, normalized); err != nil {
			return err
		}
		runID, err := g.selectClaimableRunID(ctx, exec, normalized)
		if err != nil {
			return err
		}
		if runID == "" {
			return taskpkg.ErrNoClaimableRun
		}

		claimToken, err := taskpkg.NewClaimToken()
		if err != nil {
			return err
		}
		claimHash, err := taskpkg.ClaimTokenHash(claimToken)
		if err != nil {
			return err
		}
		leaseUntil := normalized.Now.Add(normalized.LeaseDuration).UTC()
		if err := claimRunWithExecutor(ctx, exec, runID, normalized, claimToken, claimHash, leaseUntil); err != nil {
			return err
		}
		if err := setTaskCurrentRunProjectionForRun(ctx, exec, runID); err != nil {
			return err
		}

		run, err := g.getTaskRunWithExecutor(ctx, exec, runID)
		if err != nil {
			return err
		}
		taskRecord, err := g.getTaskWithExecutor(ctx, exec, run.TaskID)
		if err != nil {
			return err
		}
		channel, err := g.coordinationChannelMetadata(ctx, exec, taskRecord, run)
		if err != nil {
			return err
		}
		result = taskpkg.ClaimResult{
			Task:                taskRecord,
			Run:                 run,
			ClaimToken:          claimToken,
			LeaseUntil:          leaseUntil,
			CoordinationChannel: channel,
		}
		return nil
	}); err != nil {
		return taskpkg.ClaimResult{}, err
	}

	return taskpkg.ClaimResult{
		Task:                result.Task,
		Run:                 result.Run,
		ClaimToken:          result.ClaimToken,
		LeaseUntil:          result.LeaseUntil,
		CoordinationChannel: result.CoordinationChannel,
	}, nil
}

// HeartbeatRunLease extends one active task-run lease after token verification.
func (g *GlobalDB) HeartbeatRunLease(ctx context.Context, heartbeat taskpkg.LeaseHeartbeat) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "heartbeat task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := heartbeat.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "heartbeat task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		leaseUntil := normalized.Now.Add(normalized.LeaseDuration).UTC()
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_runs
				 SET lease_until = ?, heartbeat_at = ?, claim_token = ?
				 WHERE id = ? AND claim_token_hash = ? AND status IN (?, ?, ?)`,
			store.FormatTimestamp(leaseUntil),
			store.FormatTimestamp(normalized.Now),
			normalized.ClaimToken,
			normalized.RunID,
			current.ClaimTokenHash,
			string(taskpkg.TaskRunStatusClaimed),
			string(taskpkg.TaskRunStatusStarting),
			string(taskpkg.TaskRunStatusRunning),
		)
		if err != nil {
			return fmt.Errorf("store: heartbeat task run lease %q: %w", normalized.RunID, err)
		}
		if err := requireRowsAffected(
			result,
			taskpkg.ErrTaskRunNotFound,
			normalized.RunID,
			"task run lease",
		); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

// ReleaseRunLease clears an active task-run lease after token verification and requeues the run.
func (g *GlobalDB) ReleaseRunLease(ctx context.Context, release taskpkg.LeaseRelease) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "release task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := release.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "release task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if err := requeueLeasedRun(ctx, exec, current.ID); err != nil {
			return err
		}
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, current.ID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

// CompleteRunLease marks one claimed run complete after token verification.
func (g *GlobalDB) CompleteRunLease(ctx context.Context, completion taskpkg.LeaseCompletion) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "complete task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := completion.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "complete task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if err := requireLeaseTerminalTransition(current, taskpkg.TaskRunStatusCompleted); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_runs
			 SET status = ?, lease_until = NULL, heartbeat_at = NULL, claim_token = NULL,
			     ended_at = ?, error = NULL, result_json = ?
			 WHERE id = ? AND claim_token_hash = ?`,
			string(taskpkg.TaskRunStatusCompleted),
			store.FormatTimestamp(normalized.Now),
			nullableTaskJSON(normalized.Result.Value),
			current.ID,
			current.ClaimTokenHash,
		)
		if err != nil {
			return fmt.Errorf("store: complete task run lease %q: %w", current.ID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, current.ID, "task run lease"); err != nil {
			return err
		}
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, current.ID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

// FailRunLease marks one claimed run failed after token verification.
func (g *GlobalDB) FailRunLease(ctx context.Context, failure taskpkg.LeaseFailure) (taskpkg.Run, error) {
	if err := g.checkReady(ctx, "fail task run lease"); err != nil {
		return taskpkg.Run{}, err
	}
	normalized, err := failure.Normalize(g.now())
	if err != nil {
		return taskpkg.Run{}, err
	}

	var updated taskpkg.Run
	if err := g.withTaskImmediateTransaction(ctx, "fail task run lease", func(exec taskSQLExecutor) error {
		current, err := g.getTaskRunWithExecutor(ctx, exec, normalized.RunID)
		if err != nil {
			return err
		}
		if err := requireCurrentRunLease(current, normalized.ClaimToken, normalized.Now); err != nil {
			return err
		}
		if err := requireLeaseTerminalTransition(current, taskpkg.TaskRunStatusFailed); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE task_runs
			 SET status = ?, lease_until = NULL, heartbeat_at = NULL, claim_token = NULL,
			     ended_at = ?, error = ?, result_json = NULL
			 WHERE id = ? AND claim_token_hash = ?`,
			string(taskpkg.TaskRunStatusFailed),
			store.FormatTimestamp(normalized.Now),
			normalized.Failure.Error,
			current.ID,
			current.ClaimTokenHash,
		)
		if err != nil {
			return fmt.Errorf("store: fail task run lease %q: %w", current.ID, err)
		}
		if err := requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, current.ID, "task run lease"); err != nil {
			return err
		}
		if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
			return err
		}
		updated, err = g.getTaskRunWithExecutor(ctx, exec, current.ID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
}

// ListAutonomyLeaseHandles returns internal-only lease handles for one session.
// Public task-run read projections keep claim_token masked.
func (g *GlobalDB) ListAutonomyLeaseHandles(
	ctx context.Context,
	sessionID string,
) (handles []taskpkg.AutonomyLeaseHandle, err error) {
	if err := g.checkReady(ctx, "list autonomy lease handles"); err != nil {
		return nil, err
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, fmt.Errorf("%w: session_id is required", taskpkg.ErrValidation)
	}

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT tr.id, tr.task_id, COALESCE(t.workspace_id, ''), tr.status,
		        COALESCE(tr.session_id, ''), tr.claimed_by_kind, tr.claimed_by_ref,
		        COALESCE(tr.claim_token, ''), COALESCE(tr.claim_token_hash, ''),
		        tr.lease_until, tr.heartbeat_at
		   FROM task_runs tr
		   JOIN tasks t ON t.id = tr.task_id
		  WHERE tr.session_id = ?
		    AND COALESCE(tr.claim_token_hash, '') <> ''
		  ORDER BY COALESCE(tr.lease_until, '') DESC, tr.id ASC`,
		trimmedSessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list autonomy lease handles for session %q: %w", trimmedSessionID, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close autonomy lease handle rows: %w", closeErr)
		}
	}()

	handles = make([]taskpkg.AutonomyLeaseHandle, 0)
	for rows.Next() {
		handle, scanErr := scanAutonomyLeaseHandle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		handles = append(handles, handle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate autonomy lease handles: %w", err)
	}
	return handles, nil
}

func scanAutonomyLeaseHandle(rows *sql.Rows) (taskpkg.AutonomyLeaseHandle, error) {
	var handle taskpkg.AutonomyLeaseHandle
	var status string
	var claimedByKind sql.NullString
	var claimedByRef sql.NullString
	var leaseUntilRaw sql.NullString
	var heartbeatAtRaw sql.NullString
	if err := rows.Scan(
		&handle.RunID,
		&handle.TaskID,
		&handle.WorkspaceID,
		&status,
		&handle.SessionID,
		&claimedByKind,
		&claimedByRef,
		&handle.ClaimToken,
		&handle.ClaimTokenHash,
		&leaseUntilRaw,
		&heartbeatAtRaw,
	); err != nil {
		return taskpkg.AutonomyLeaseHandle{}, fmt.Errorf("store: scan autonomy lease handle: %w", err)
	}
	handle.Status = taskpkg.RunStatus(strings.TrimSpace(status)).Normalize()
	if claimedByKind.Valid || claimedByRef.Valid {
		handle.ClaimedBy = &taskpkg.ActorIdentity{
			Kind: taskpkg.ActorKind(strings.TrimSpace(claimedByKind.String)),
			Ref:  strings.TrimSpace(claimedByRef.String),
		}
	}
	if err := setAutonomyLeaseHandleTimestamps(&handle, leaseUntilRaw, heartbeatAtRaw); err != nil {
		return taskpkg.AutonomyLeaseHandle{}, err
	}
	return handle, nil
}

func setAutonomyLeaseHandleTimestamps(
	handle *taskpkg.AutonomyLeaseHandle,
	leaseUntilRaw sql.NullString,
	heartbeatAtRaw sql.NullString,
) error {
	if leaseUntilRaw.Valid && strings.TrimSpace(leaseUntilRaw.String) != "" {
		leaseUntil, err := store.ParseTimestamp(leaseUntilRaw.String)
		if err != nil {
			return fmt.Errorf("store: parse autonomy lease_until for run %q: %w", handle.RunID, err)
		}
		handle.LeaseUntil = leaseUntil
	}
	if heartbeatAtRaw.Valid && strings.TrimSpace(heartbeatAtRaw.String) != "" {
		heartbeatAt, err := store.ParseTimestamp(heartbeatAtRaw.String)
		if err != nil {
			return fmt.Errorf("store: parse autonomy heartbeat_at for run %q: %w", handle.RunID, err)
		}
		handle.HeartbeatAt = heartbeatAt
	}
	return nil
}

// RecoverExpiredRunLeases requeues stale active leases without issuing new ownership.
func (g *GlobalDB) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	if err := g.checkReady(ctx, "recover expired task run leases"); err != nil {
		return nil, err
	}
	normalized, err := recovery.Normalize(g.now())
	if err != nil {
		return nil, err
	}

	recovered := make([]taskpkg.ExpiredLeaseRecoveryResult, 0)
	if err := g.withTaskImmediateTransaction(ctx, "recover expired task run leases", func(exec taskSQLExecutor) error {
		runIDs, err := expiredLeaseRunIDs(ctx, exec, normalized)
		if err != nil {
			return err
		}
		for _, runID := range runIDs {
			current, err := g.getTaskRunWithExecutor(ctx, exec, runID)
			if err != nil {
				return err
			}
			if current.LeaseUntil.IsZero() || current.LeaseUntil.After(normalized.Now) {
				continue
			}
			snapshot := taskRunLeaseSnapshot{
				status:         current.Status,
				sessionID:      current.SessionID,
				leaseUntil:     current.LeaseUntil,
				claimTokenHash: current.ClaimTokenHash,
			}
			if err := requeueExpiredLease(ctx, exec, current, snapshot); err != nil {
				return err
			}
			if err := clearTaskCurrentRunProjection(ctx, exec, current.TaskID, current.ID); err != nil {
				return err
			}
			updated, err := g.getTaskRunWithExecutor(ctx, exec, current.ID)
			if err != nil {
				return err
			}
			recovered = append(recovered, taskpkg.ExpiredLeaseRecoveryResult{
				Run:                    updated,
				PreviousRunStatus:      snapshot.status,
				PreviousSessionID:      snapshot.sessionID,
				PreviousLeaseUntil:     snapshot.leaseUntil,
				PreviousClaimTokenHash: snapshot.claimTokenHash,
				Reason:                 normalized.Reason,
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return recovered, nil
}

func (g *GlobalDB) ensureClaimerHasNoActiveLease(
	ctx context.Context,
	exec taskSQLExecutor,
	criteria taskpkg.ClaimCriteria,
) error {
	var count int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM task_runs
		 WHERE session_id = ?
		   AND status IN (?, ?, ?)
		   AND (lease_until IS NULL OR lease_until > ?)`,
		criteria.ClaimerSessionID,
		string(taskpkg.TaskRunStatusClaimed),
		string(taskpkg.TaskRunStatusStarting),
		string(taskpkg.TaskRunStatusRunning),
		store.FormatTimestamp(criteria.Now),
	).Scan(&count); err != nil {
		return fmt.Errorf("store: count active task-run leases for %q: %w", criteria.ClaimerSessionID, err)
	}
	if count > 0 {
		return fmt.Errorf(
			"%w: session %q already owns an active task-run lease",
			taskpkg.ErrActiveRunLease,
			criteria.ClaimerSessionID,
		)
	}
	return nil
}

func (g *GlobalDB) selectClaimableRunID(
	ctx context.Context,
	exec taskSQLExecutor,
	criteria taskpkg.ClaimCriteria,
) (string, error) {
	where := []string{
		"tr.status = ?",
		"COALESCE(t.paused, 0) = 0",
		`NOT EXISTS (SELECT 1 FROM scheduler_pause sp WHERE sp.id = 1 AND sp.paused = 1)`,
		`NOT EXISTS (
			WITH RECURSIVE ancestors(id, parent_task_id, paused) AS (
				SELECT parent.id, parent.parent_task_id, parent.paused
				  FROM tasks parent
				 WHERE parent.id = t.parent_task_id
				UNION ALL
				SELECT parent.id, parent.parent_task_id, parent.paused
				  FROM tasks parent
				  JOIN ancestors a ON parent.id = a.parent_task_id
			)
			SELECT 1 FROM ancestors WHERE COALESCE(paused, 0) = 1
		)`,
		"t.status NOT IN (?, ?, ?)",
		taskPriorityValueSQL + " >= ?",
		`NOT EXISTS (
			SELECT 1
			  FROM task_run_required_capabilities req
			 WHERE req.run_id = tr.id` + missingCapabilityPredicate(criteria.RequiredCapabilities) + `
		)`,
	}
	args := []any{
		string(taskpkg.TaskRunStatusQueued),
		string(taskpkg.TaskStatusDraft),
		string(taskpkg.TaskStatusBlocked),
		string(taskpkg.TaskStatusCanceled),
		criteria.PriorityMin,
	}
	args = append(args, missingCapabilityArgs(criteria.RequiredCapabilities)...)
	if criteria.Scope == taskpkg.ScopeWorkspace {
		where = append(where, "t.scope = ?", "t.workspace_id = ?")
		args = append(args, string(taskpkg.ScopeWorkspace), criteria.WorkspaceID)
	} else {
		where = append(where, "t.scope = ?")
		args = append(args, string(taskpkg.ScopeGlobal))
	}
	if strings.TrimSpace(criteria.CoordinationChannelID) != "" {
		where = append(where, "tr.coordination_channel_id = ?")
		args = append(args, criteria.CoordinationChannelID)
	}
	where, args = appendProfileClaimFilters(where, args, criteria)
	where, args = appendClaimOwnerPredicate(where, args, criteria)
	args = append(args, preferredCapabilityArgs(criteria.RequiredCapabilities)...)

	query := `SELECT tr.id
		FROM task_runs tr
		JOIN tasks t ON t.id = tr.task_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY ` + taskPriorityValueSQL + ` DESC,
		         ` + preferredCapabilityOrder(criteria.RequiredCapabilities) + `
		         tr.queued_at ASC,
		         tr.id ASC
		LIMIT 1`

	var runID string
	if err := exec.QueryRowContext(ctx, query, args...).Scan(&runID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("store: select claimable task run: %w", err)
	}
	return runID, nil
}

func appendProfileClaimFilters(
	where []string,
	args []any,
	criteria taskpkg.ClaimCriteria,
) ([]string, []any) {
	where, args = appendProfileRequiredCapabilityFilter(
		where,
		args,
		profileRoleWorker,
		criteria.RequiredCapabilities,
	)
	where, args = appendProfileRequiredCapabilityFilter(
		where,
		args,
		profileRoleParticipant,
		criteria.RequiredCapabilities,
	)
	agentName := strings.TrimSpace(criteria.AgentName)
	where = append(where, `NOT EXISTS (
			SELECT 1
			  FROM task_execution_profiles tep
			 WHERE tep.task_id = t.id
			   AND COALESCE(tep.worker_agent_name, '') <> ''
			   AND tep.worker_agent_name <> ?
		)`)
	args = append(args, agentName)
	where, args = appendProfileAllowedAgentFilter(where, args, profileRoleWorker, agentName)
	return appendProfileAllowedAgentFilter(where, args, profileRoleParticipant, agentName)
}

func appendProfileRequiredCapabilityFilter(
	where []string,
	args []any,
	role string,
	capabilities []string,
) ([]string, []any) {
	where = append(where, `NOT EXISTS (
			SELECT 1
			  FROM task_profile_capabilities pc
			 WHERE pc.task_id = t.id
			   AND pc.role = ?
			   AND pc.preference = ?`+missingCapabilityPredicateFor("pc.capability_id", capabilities)+`
		)`)
	args = append(args, role, profilePreferenceRequired)
	args = append(args, missingCapabilityArgs(capabilities)...)
	return where, args
}

func appendProfileAllowedAgentFilter(
	where []string,
	args []any,
	role string,
	agentName string,
) ([]string, []any) {
	where = append(where, `(NOT EXISTS (
			SELECT 1
			  FROM task_profile_agents pa_all
			 WHERE pa_all.task_id = t.id
			   AND pa_all.role = ?
			   AND pa_all.preference = ?
		) OR EXISTS (
			SELECT 1
			  FROM task_profile_agents pa_match
			 WHERE pa_match.task_id = t.id
			   AND pa_match.role = ?
			   AND pa_match.preference = ?
			   AND pa_match.agent_name = ?
		))`)
	args = append(args, role, profilePreferenceAllowed, role, profilePreferenceAllowed, agentName)
	return where, args
}

func appendClaimOwnerPredicate(
	where []string,
	args []any,
	criteria taskpkg.ClaimCriteria,
) ([]string, []any) {
	clauses := []string{"COALESCE(t.owner_kind, '') = ''"}
	if agentName := strings.TrimSpace(criteria.AgentName); agentName != "" {
		clauses = append(clauses, "(t.owner_kind = ? AND t.owner_ref = ?)")
		args = append(args, string(taskpkg.OwnerKindPool), agentName)
	}
	if sessionID := strings.TrimSpace(criteria.ClaimerSessionID); sessionID != "" {
		clauses = append(clauses, "(t.owner_kind = ? AND t.owner_ref = ?)")
		args = append(args, string(taskpkg.OwnerKindAgentSession), sessionID)
	}
	where = append(where, "("+strings.Join(clauses, " OR ")+")")
	return where, args
}

func claimRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
	criteria taskpkg.ClaimCriteria,
	claimToken string,
	claimHash string,
	leaseUntil time.Time,
) error {
	metadata, err := claimRunMetadata(ctx, exec, runID, criteria)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
			 SET status = ?, claimed_by_kind = ?, claimed_by_ref = ?, session_id = ?,
			     claim_token = ?, claim_token_hash = ?, lease_until = ?, heartbeat_at = ?,
			     claimed_at = ?, started_at = NULL, ended_at = NULL, error = NULL,
			     metadata_json = ?, result_json = NULL
			 WHERE id = ? AND status = ?`,
		string(taskpkg.TaskRunStatusClaimed),
		taskActorKindValue(criteria.ClaimedBy),
		taskActorRefValue(criteria.ClaimedBy),
		criteria.ClaimerSessionID,
		claimToken,
		claimHash,
		store.FormatTimestamp(leaseUntil),
		store.FormatTimestamp(criteria.Now),
		store.FormatTimestamp(criteria.Now),
		nullableTaskJSON(metadata),
		runID,
		string(taskpkg.TaskRunStatusQueued),
	)
	if err != nil {
		return fmt.Errorf("store: claim task run %q: %w", runID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrNoClaimableRun, runID, "task run claim")
}

func claimRunMetadata(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
	criteria taskpkg.ClaimCriteria,
) (json.RawMessage, error) {
	var raw sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT metadata_json FROM task_runs WHERE id = ?`,
		strings.TrimSpace(runID),
	).Scan(&raw); err != nil {
		return nil, fmt.Errorf("store: load task run metadata for claim %q: %w", runID, err)
	}
	metadata, err := decodeTaskJSON(raw, "task_run.metadata_json")
	if err != nil {
		return nil, err
	}
	if criteria.Soul == nil || strings.TrimSpace(criteria.Soul.Digest) == "" {
		return normalizeTaskJSON(metadata), nil
	}
	merged, err := mergeClaimSoulMetadata(metadata, *criteria.Soul)
	if err != nil {
		return nil, fmt.Errorf("store: merge soul claim metadata for run %q: %w", runID, err)
	}
	return merged, nil
}

func mergeClaimSoulMetadata(
	current json.RawMessage,
	provenance taskpkg.SoulClaimProvenance,
) (json.RawMessage, error) {
	normalized := normalizeTaskJSON(current)
	fields := make(map[string]json.RawMessage)
	if len(normalized) > 0 {
		if err := json.Unmarshal(normalized, &fields); err != nil {
			return nil, fmt.Errorf(
				"%w: task_run.metadata_json must be a JSON object for soul provenance",
				taskpkg.ErrValidation,
			)
		}
		if fields == nil {
			fields = make(map[string]json.RawMessage)
		}
	}
	payload := struct {
		SnapshotID string    `json:"snapshot_id,omitempty"`
		Digest     string    `json:"digest,omitempty"`
		AgentName  string    `json:"agent_name,omitempty"`
		CapturedAt time.Time `json:"captured_at"`
	}{
		SnapshotID: strings.TrimSpace(provenance.SnapshotID),
		Digest:     strings.TrimSpace(provenance.Digest),
		AgentName:  strings.TrimSpace(provenance.AgentName),
		CapturedAt: provenance.CapturedAt.UTC(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("store: marshal soul claim metadata: %w", err)
	}
	fields["soul"] = encoded
	merged, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("store: marshal task run claim metadata: %w", err)
	}
	result := normalizeTaskJSON(merged)
	if err := taskpkg.ValidateMetadataSize(result, "task_run.metadata_json"); err != nil {
		return nil, err
	}
	return result, nil
}

func requireCurrentRunLease(run taskpkg.Run, rawToken string, now time.Time) error {
	if strings.TrimSpace(run.ClaimTokenHash) == "" {
		return fmt.Errorf("%w: task run %q has no current claim token hash", taskpkg.ErrInvalidClaimToken, run.ID)
	}
	if !taskpkg.VerifyClaimToken(rawToken, run.ClaimTokenHash) {
		return fmt.Errorf("%w: task run %q token mismatch", taskpkg.ErrInvalidClaimToken, run.ID)
	}
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusClaimed, taskpkg.TaskRunStatusStarting, taskpkg.TaskRunStatusRunning:
	default:
		return fmt.Errorf("%w: task run %q is not actively leased", taskpkg.ErrInvalidStatusTransition, run.ID)
	}
	if run.LeaseUntil.IsZero() || !run.LeaseUntil.After(now.UTC()) {
		return fmt.Errorf("%w: task run %q lease expired", taskpkg.ErrLeaseExpired, run.ID)
	}
	return nil
}

func requireLeaseTerminalTransition(run taskpkg.Run, target taskpkg.RunStatus) error {
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusClaimed, taskpkg.TaskRunStatusStarting, taskpkg.TaskRunStatusRunning:
		return nil
	default:
		return fmt.Errorf(
			"%w: task run %q cannot transition from %q to %q",
			taskpkg.ErrInvalidStatusTransition,
			run.ID,
			run.Status.Normalize(),
			target.Normalize(),
		)
	}
}

func requeueLeasedRun(ctx context.Context, exec taskSQLExecutor, runID string) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, claimed_by_kind = NULL, claimed_by_ref = NULL, session_id = NULL,
		     claim_token = NULL, claim_token_hash = NULL, lease_until = NULL, heartbeat_at = NULL,
		     claimed_at = NULL, started_at = NULL, ended_at = NULL, error = NULL, result_json = NULL
		 WHERE id = ?`,
		string(taskpkg.TaskRunStatusQueued),
		runID,
	)
	if err != nil {
		return fmt.Errorf("store: requeue task run lease %q: %w", runID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, runID, "task run lease")
}

func expiredLeaseRunIDs(
	ctx context.Context,
	exec taskSQLExecutor,
	recovery taskpkg.ExpiredLeaseRecovery,
) ([]string, error) {
	query := `SELECT id
		FROM task_runs
		WHERE status IN (?, ?, ?)
		  AND lease_until IS NOT NULL
		  AND lease_until <= ?
		ORDER BY lease_until ASC, id ASC`
	args := []any{
		string(taskpkg.TaskRunStatusClaimed),
		string(taskpkg.TaskRunStatusStarting),
		string(taskpkg.TaskRunStatusRunning),
		store.FormatTimestamp(recovery.Now),
	}
	query, args = store.AppendLimit(query, args, recovery.Limit)

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query expired task run leases: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	runIDs := make([]string, 0)
	for rows.Next() {
		var runID string
		if err := rows.Scan(&runID); err != nil {
			return nil, fmt.Errorf("store: scan expired task run lease id: %w", err)
		}
		runIDs = append(runIDs, runID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate expired task run leases: %w", err)
	}
	return runIDs, nil
}

func requeueExpiredLease(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	snapshot taskRunLeaseSnapshot,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, claimed_by_kind = NULL, claimed_by_ref = NULL, session_id = NULL,
		     claim_token = NULL, claim_token_hash = NULL, lease_until = NULL, heartbeat_at = NULL,
		     claimed_at = NULL, started_at = NULL, ended_at = NULL, error = NULL, result_json = NULL
		 WHERE id = ?
		   AND status = ?
		   AND COALESCE(session_id, '') = ?
		   AND claim_token_hash = ?
		   AND lease_until = ?`,
		string(taskpkg.TaskRunStatusQueued),
		run.ID,
		string(snapshot.status.Normalize()),
		strings.TrimSpace(snapshot.sessionID),
		strings.TrimSpace(snapshot.claimTokenHash),
		store.FormatTimestamp(snapshot.leaseUntil),
	)
	if err != nil {
		return fmt.Errorf("store: recover expired task run lease %q: %w", run.ID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrTaskRunNotFound, run.ID, "expired task run lease")
}

func (g *GlobalDB) coordinationChannelMetadata(
	ctx context.Context,
	exec taskSQLExecutor,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
) (*taskpkg.CoordinationChannelMetadata, error) {
	channelID := strings.TrimSpace(run.CoordinationChannelID)
	if channelID == "" {
		return nil, nil
	}
	metadata := &taskpkg.CoordinationChannelMetadata{
		ID:          channelID,
		Channel:     channelID,
		DisplayName: channelID,
		WorkspaceID: taskRecord.WorkspaceID,
		TaskID:      run.TaskID,
		RunID:       run.ID,
		WorkflowID:  taskRunMetadataString(run.Metadata, "workflow_id"),
		AllowedMessageKinds: []string{
			globalDBTaskClaimStatusKey,
			"request",
			"reply",
			"blocker",
			globalDBTaskClaimHandoffKey,
			"result",
			"review_request",
		},
	}

	entry, err := networkChannelEntry(ctx, exec, channelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return metadata, nil
		}
		return nil, err
	}
	metadata.Channel = entry.Channel
	metadata.DisplayName = entry.Channel
	metadata.Purpose = entry.Purpose
	metadata.WorkspaceID = entry.WorkspaceID
	metadata.LastActivityAt = entry.UpdatedAt
	return metadata, nil
}

func networkChannelEntry(
	ctx context.Context,
	exec taskSQLExecutor,
	channelID string,
) (store.NetworkChannelEntry, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT channel, workspace_id, purpose, created_by, created_at, updated_at
		 FROM network_channels
		 WHERE channel = ?`,
		channelID,
	)
	return scanNetworkChannel(row)
}

func missingCapabilityPredicate(capabilities []string) string {
	return missingCapabilityPredicateFor("req.capability_id", capabilities)
}

func missingCapabilityPredicateFor(column string, capabilities []string) string {
	if len(capabilities) == 0 {
		return ""
	}
	return " AND " + column + " NOT IN (" + claimPlaceholders(len(capabilities)) + ")"
}

func missingCapabilityArgs(capabilities []string) []any {
	if len(capabilities) == 0 {
		return nil
	}
	args := make([]any, 0, len(capabilities))
	for _, capability := range capabilities {
		args = append(args, capability)
	}
	return args
}

func preferredCapabilityOrder(capabilities []string) string {
	if len(capabilities) == 0 {
		return "(SELECT 0) DESC,"
	}
	return `(SELECT COUNT(1)
		          FROM task_run_preferred_capabilities pref
		         WHERE pref.run_id = tr.id
		           AND pref.capability_id IN (` + claimPlaceholders(len(capabilities)) + `)) DESC,`
}

func preferredCapabilityArgs(capabilities []string) []any {
	return missingCapabilityArgs(capabilities)
}

func claimPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	values := make([]string, 0, count)
	for range count {
		values = append(values, "?")
	}
	return strings.Join(values, ", ")
}

func taskRunMetadataString(raw []byte, key string) string {
	if len(raw) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ""
	}
	value, ok := decoded[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
