package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

func requireCurrentRunLease(run taskpkg.Run, rawToken string, now time.Time) error {
	if strings.TrimSpace(run.ClaimTokenHash) == "" {
		return fmt.Errorf(
			"%w: task run %q has no current claim token hash",
			taskpkg.ErrInvalidClaimToken,
			run.ID,
		)
	}
	if !taskpkg.VerifyClaimToken(rawToken, run.ClaimTokenHash) {
		return fmt.Errorf("%w: task run %q token mismatch", taskpkg.ErrInvalidClaimToken, run.ID)
	}
	switch run.Status.Normalize() {
	case taskpkg.TaskRunStatusClaimed, taskpkg.TaskRunStatusStarting, taskpkg.TaskRunStatusRunning:
	default:
		return fmt.Errorf(
			"%w: task run %q is not actively leased",
			taskpkg.ErrInvalidStatusTransition,
			run.ID,
		)
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
		taskpkg.TaskRunStatusQueued.String(),
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
		taskpkg.TaskRunStatusClaimed.String(),
		taskpkg.TaskRunStatusStarting.String(),
		taskpkg.TaskRunStatusRunning.String(),
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
		taskpkg.TaskRunStatusQueued.String(),
		run.ID,
		snapshot.status.Normalize().String(),
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
		`SELECT channel, workspace_id, purpose, fanout_policy, coordinator_peer_id, created_by, created_at, updated_at
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
