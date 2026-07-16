package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

var _ store.NetworkUsageStore = (*NetworkRepo)(nil)

// GetNetworkUsage returns workspace-fenced ledger details and their exact aggregate.
func (g *NetworkRepo) GetNetworkUsage(
	ctx context.Context,
	query store.NetworkUsageQuery,
) (report store.NetworkUsageReport, err error) {
	if err := g.checkReady(ctx, "get network usage"); err != nil {
		return store.NetworkUsageReport{}, err
	}
	normalized := store.NetworkUsageQuery{
		WorkspaceID: strings.TrimSpace(query.WorkspaceID),
		Channel:     strings.TrimSpace(query.Channel),
		RunID:       strings.TrimSpace(query.RunID),
		OwnerKey:    strings.TrimSpace(query.OwnerKey),
	}
	if err := normalized.Validate(); err != nil {
		return store.NetworkUsageReport{}, fmt.Errorf("store: validate network usage query: %w", err)
	}

	const selectUsage = `SELECT wake_id, task_run_id, owner_key, workspace_id, channel,
	root_id, depth, state, reserved_wall_ms, actual_wall_ms, reserved_at,
	settled_at, input_tokens, output_tokens, usage_state, reason
FROM network_live_wakes
WHERE workspace_id = ?
	AND (? = '' OR channel = ?)
	AND (? = '' OR owner_key = ?)
	AND (? = '' OR owner_key IN (?, ?) OR task_run_id = ?)
ORDER BY reserved_at ASC, wake_id ASC`
	rows, err := g.db.QueryContext(
		ctx,
		selectUsage,
		normalized.WorkspaceID,
		normalized.Channel,
		normalized.Channel,
		normalized.OwnerKey,
		normalized.OwnerKey,
		normalized.RunID,
		"task_run:"+normalized.RunID,
		"loop_run:"+normalized.RunID,
		normalized.RunID,
	)
	if err != nil {
		return store.NetworkUsageReport{}, fmt.Errorf("store: query network usage: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network usage rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	report.Details = make([]store.NetworkWakeUsageDetail, 0)
	for rows.Next() {
		detail, scanErr := scanNetworkWakeUsage(rows)
		if scanErr != nil {
			return store.NetworkUsageReport{}, scanErr
		}
		report.Details = append(report.Details, detail)
		addNetworkWakeUsage(&report.Total, detail)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return store.NetworkUsageReport{}, fmt.Errorf("store: iterate network usage: %w", rowsErr)
	}
	report.Budget, err = g.loadNetworkUsageBudget(ctx, normalized)
	if err != nil {
		return store.NetworkUsageReport{}, err
	}
	return report, nil
}

func (g *NetworkRepo) loadNetworkUsageBudget(
	ctx context.Context,
	query store.NetworkUsageQuery,
) (*store.NetworkBudgetUsage, error) {
	if query.OwnerKey == "" && query.RunID == "" {
		return nil, nil
	}
	ownerKeys := []string{query.OwnerKey}
	if query.OwnerKey == "" {
		ownerKeys = []string{"task_run:" + query.RunID, "loop_run:" + query.RunID}
	}
	const selectBudget = `SELECT owner_key, wakes_used, wall_ms_used, input_tokens_used,
	output_tokens_used, exhausted_reason, updated_at
FROM network_participation_budgets
WHERE owner_key IN (?, ?)
ORDER BY CASE WHEN owner_key = ? THEN 0 ELSE 1 END
LIMIT 1`
	primary := ownerKeys[0]
	secondary := primary
	if len(ownerKeys) > 1 {
		secondary = ownerKeys[1]
	}
	var budget store.NetworkBudgetUsage
	var wallMS int64
	var updatedAtRaw string
	err := g.db.QueryRowContext(ctx, selectBudget, primary, secondary, primary).Scan(
		&budget.OwnerKey,
		&budget.WakesUsed,
		&wallMS,
		&budget.InputTokensUsed,
		&budget.OutputTokensUsed,
		&budget.ExhaustedReason,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: query network usage budget: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return nil, fmt.Errorf("store: parse network usage budget updated_at: %w", err)
	}
	budget.WallTimeUsed = time.Duration(wallMS) * time.Millisecond
	budget.UpdatedAt = updatedAt
	return &budget, nil
}

func scanNetworkWakeUsage(rows *sql.Rows) (store.NetworkWakeUsageDetail, error) {
	var detail store.NetworkWakeUsageDetail
	var reservedWallMS int64
	var actualWallMS sql.NullInt64
	var reservedAtRaw string
	var settledAtRaw sql.NullString
	var inputTokens sql.NullInt64
	var outputTokens sql.NullInt64
	if err := rows.Scan(
		&detail.WakeID,
		&detail.TaskRunID,
		&detail.OwnerKey,
		&detail.WorkspaceID,
		&detail.Channel,
		&detail.RootID,
		&detail.Depth,
		&detail.State,
		&reservedWallMS,
		&actualWallMS,
		&reservedAtRaw,
		&settledAtRaw,
		&inputTokens,
		&outputTokens,
		&detail.UsageState,
		&detail.Reason,
	); err != nil {
		return store.NetworkWakeUsageDetail{}, fmt.Errorf("store: scan network usage: %w", err)
	}
	reservedAt, err := store.ParseTimestamp(reservedAtRaw)
	if err != nil {
		return store.NetworkWakeUsageDetail{}, fmt.Errorf("store: parse network usage reserved_at: %w", err)
	}
	detail.ReservedAt = reservedAt
	if settledAtRaw.Valid {
		settledAt, parseErr := store.ParseTimestamp(settledAtRaw.String)
		if parseErr != nil {
			return store.NetworkWakeUsageDetail{}, fmt.Errorf("store: parse network usage settled_at: %w", parseErr)
		}
		detail.SettledAt = &settledAt
	}
	chargedWallMS := reservedWallMS
	if actualWallMS.Valid {
		chargedWallMS = actualWallMS.Int64
	}
	detail.ChargedWallTime = time.Duration(chargedWallMS) * time.Millisecond
	detail.InputTokens = inputTokens.Int64
	detail.OutputTokens = outputTokens.Int64
	if strings.TrimSpace(detail.UsageState) == "" {
		detail.UsageState = store.NetworkWakeUsageReserved
	}
	return detail, nil
}

func addNetworkWakeUsage(total *store.NetworkUsageSummary, detail store.NetworkWakeUsageDetail) {
	total.WakeCount++
	total.ChargedWallTime += detail.ChargedWallTime
	total.InputTokens += detail.InputTokens
	total.OutputTokens += detail.OutputTokens
	switch detail.UsageState {
	case store.NetworkWakeUsageActual:
		total.ActualWakeCount++
	case store.NetworkWakeUsageUnavailable:
		total.UnavailableWakeCount++
	default:
		total.ReservedWakeCount++
	}
}
