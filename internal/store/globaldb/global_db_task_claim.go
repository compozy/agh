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
		if err := claimRunWithExecutor(ctx, exec, runID, normalized, claimHash, leaseUntil); err != nil {
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
			 SET lease_until = ?, heartbeat_at = ?, claim_token = NULL
			 WHERE id = ? AND claim_token_hash = ? AND status IN (?, ?, ?)`,
			store.FormatTimestamp(leaseUntil),
			store.FormatTimestamp(normalized.Now),
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
		updated, err = g.getTaskRunWithExecutor(ctx, exec, current.ID)
		return err
	}); err != nil {
		return taskpkg.Run{}, err
	}
	return updated, nil
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

func claimRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID string,
	criteria taskpkg.ClaimCriteria,
	claimHash string,
	leaseUntil time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE task_runs
		 SET status = ?, claimed_by_kind = ?, claimed_by_ref = ?, session_id = ?,
		     claim_token = NULL, claim_token_hash = ?, lease_until = ?, heartbeat_at = ?,
		     claimed_at = ?, started_at = NULL, ended_at = NULL, error = NULL, result_json = NULL
		 WHERE id = ? AND status = ?`,
		string(taskpkg.TaskRunStatusClaimed),
		taskActorKindValue(criteria.ClaimedBy),
		taskActorRefValue(criteria.ClaimedBy),
		criteria.ClaimerSessionID,
		claimHash,
		store.FormatTimestamp(leaseUntil),
		store.FormatTimestamp(criteria.Now),
		store.FormatTimestamp(criteria.Now),
		runID,
		string(taskpkg.TaskRunStatusQueued),
	)
	if err != nil {
		return fmt.Errorf("store: claim task run %q: %w", runID, err)
	}
	return requireRowsAffected(result, taskpkg.ErrNoClaimableRun, runID, "task run claim")
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
		ID:                  channelID,
		Channel:             channelID,
		DisplayName:         channelID,
		WorkspaceID:         taskRecord.WorkspaceID,
		TaskID:              run.TaskID,
		RunID:               run.ID,
		WorkflowID:          taskRunMetadataString(run.Metadata, "workflow_id"),
		AllowedMessageKinds: []string{"status", "request", "reply", "blocker", "handoff", "result", "review_request"},
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
	if len(capabilities) == 0 {
		return ""
	}
	return ` AND req.capability_id NOT IN (` + claimPlaceholders(len(capabilities)) + `)`
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
