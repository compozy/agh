package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/store"
)

type networkWakeSettlementRow struct {
	wakeID         string
	taskRunID      string
	ownerKey       string
	state          string
	reservedWallMS int64
	reservedInput  int64
	reservedOutput int64
}

// SettleNetworkWake applies one terminal outcome exactly once.
func (g *NetworkRepo) SettleNetworkWake(
	ctx context.Context,
	reservation store.WakeReservation,
	outcome store.NetworkWakeOutcome,
) error {
	if err := g.checkReady(ctx, "settle network wake"); err != nil {
		return err
	}
	reservation.WakeID = strings.TrimSpace(reservation.WakeID)
	reservation.TaskRunID = strings.TrimSpace(reservation.TaskRunID)
	reservation.OwnerKey = strings.TrimSpace(reservation.OwnerKey)
	if reservation.WakeID == "" {
		return fmt.Errorf("store: network wake reservation wake_id is required")
	}
	if err := outcome.Validate(); err != nil {
		return err
	}
	settledAt := g.now().UTC()
	return g.withNetworkImmediateTransaction(ctx, "settle network wake", func(exec networkSQLExecutor) error {
		row, err := getNetworkWakeSettlementRow(ctx, exec, reservation.WakeID)
		if err != nil {
			return err
		}
		if row.state != "open" {
			return nil
		}
		if reservation.TaskRunID != "" && reservation.TaskRunID != row.taskRunID {
			return fmt.Errorf("store: network wake reservation task_run_id mismatch")
		}
		if reservation.OwnerKey != "" && reservation.OwnerKey != row.ownerKey {
			return fmt.Errorf("store: network wake reservation owner_key mismatch")
		}
		run, err := g.tasks.getTaskRunWithExecutor(ctx, exec, row.taskRunID)
		if err != nil {
			return fmt.Errorf("store: load network wake task run for settlement: %w", err)
		}
		bounds := run.NetworkSpecSnapshot().Bounds
		chargedWall, chargedInput, chargedOutput := settledNetworkWakeUsage(row, outcome)
		changed, err := compareAndSetNetworkWakeTerminal(
			ctx,
			exec,
			row.wakeID,
			outcome,
			chargedWall,
			chargedInput,
			chargedOutput,
			settledAt,
		)
		if err != nil || !changed {
			return err
		}
		return settleNetworkWakeBudget(
			ctx,
			exec,
			row,
			bounds,
			chargedWall,
			chargedInput,
			chargedOutput,
			settledAt,
		)
	})
}

func getNetworkWakeSettlementRow(
	ctx context.Context,
	exec networkSQLExecutor,
	wakeID string,
) (networkWakeSettlementRow, error) {
	// dynamic-sql: settlement locks this accounting row through the enclosing immediate transaction.
	const query = `
SELECT wake_id, task_run_id, owner_key, state, reserved_wall_ms,
       input_tokens, output_tokens
FROM network_live_wakes
WHERE wake_id = ?`
	var row networkWakeSettlementRow
	if err := exec.QueryRowContext(ctx, query, wakeID).Scan(
		&row.wakeID,
		&row.taskRunID,
		&row.ownerKey,
		&row.state,
		&row.reservedWallMS,
		&row.reservedInput,
		&row.reservedOutput,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return networkWakeSettlementRow{}, fmt.Errorf("store: network wake %q not found", wakeID)
		}
		return networkWakeSettlementRow{}, fmt.Errorf("store: get network wake settlement row: %w", err)
	}
	return row, nil
}

func settledNetworkWakeUsage(
	row networkWakeSettlementRow,
	outcome store.NetworkWakeOutcome,
) (wallMS int64, inputTokens int64, outputTokens int64) {
	wallMS = durationMillisecondsCeil(outcome.ActualWallTime)
	if wallMS == 0 && outcome.UsageState == store.NetworkWakeUsageUnavailable {
		wallMS = row.reservedWallMS
	}
	if outcome.UsageState == store.NetworkWakeUsageUnavailable {
		return wallMS, row.reservedInput, row.reservedOutput
	}
	return wallMS, outcome.ActualInputTokens, outcome.ActualOutputTokens
}

func compareAndSetNetworkWakeTerminal(
	ctx context.Context,
	exec networkSQLExecutor,
	wakeID string,
	outcome store.NetworkWakeOutcome,
	wallMS int64,
	inputTokens int64,
	outputTokens int64,
	settledAt time.Time,
) (bool, error) {
	// dynamic-sql: state='open' is the settlement idempotency fence.
	const query = `
UPDATE network_live_wakes
SET state = ?, actual_wall_ms = ?, settled_at = ?, input_tokens = ?,
    output_tokens = ?, usage_state = ?, reason = ?
WHERE wake_id = ? AND state = 'open'`
	result, err := exec.ExecContext(
		ctx,
		query,
		normalizeNetworkWakeState(outcome.State),
		wallMS,
		store.FormatTimestamp(settledAt),
		inputTokens,
		outputTokens,
		strings.TrimSpace(outcome.UsageState),
		strings.TrimSpace(outcome.Reason),
		wakeID,
	)
	if err != nil {
		return false, fmt.Errorf("store: settle network wake ledger: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: inspect network wake settlement: %w", err)
	}
	return rowsAffected == 1, nil
}

func settleNetworkWakeBudget(
	ctx context.Context,
	exec networkSQLExecutor,
	row networkWakeSettlementRow,
	bounds participation.Bounds,
	wallMS int64,
	inputTokens int64,
	outputTokens int64,
	settledAt time.Time,
) error {
	budget, err := getNetworkWakeBudget(ctx, exec, row.ownerKey)
	if err != nil {
		return err
	}
	next := networkWakeBudget{
		wakesUsed:        budget.wakesUsed,
		wallMSUsed:       budget.wallMSUsed - row.reservedWallMS + wallMS,
		inputTokensUsed:  budget.inputTokensUsed - row.reservedInput + inputTokens,
		outputTokensUsed: budget.outputTokensUsed - row.reservedOutput + outputTokens,
	}
	reason, err := settledNetworkWakeBudgetReason(next, bounds)
	if err != nil {
		return err
	}
	// dynamic-sql: this replacement is paired with the successful ledger CAS above.
	const query = `
UPDATE network_participation_budgets
SET wall_ms_used = ?, input_tokens_used = ?, output_tokens_used = ?,
    exhausted_reason = ?, updated_at = ?
WHERE owner_key = ?`
	result, err := exec.ExecContext(
		ctx,
		query,
		next.wallMSUsed,
		next.inputTokensUsed,
		next.outputTokensUsed,
		reason,
		store.FormatTimestamp(settledAt),
		row.ownerKey,
	)
	if err != nil {
		return fmt.Errorf("store: settle network participation budget: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: inspect network participation budget settlement: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("store: network participation budget %q not found", row.ownerKey)
	}
	return nil
}

func settledNetworkWakeBudgetReason(
	budget networkWakeBudget,
	bounds participation.Bounds,
) (string, error) {
	totalWall, err := time.ParseDuration(bounds.MaxTotalWallTime)
	if err != nil {
		return "", fmt.Errorf("store: parse settled network total wall time: %w", err)
	}
	switch {
	case budget.wakesUsed >= bounds.MaxWakes:
		return networkWakeExhaustionMaxWakes, nil
	case budget.wallMSUsed >= durationMillisecondsCeil(totalWall):
		return "max_total_wall_time", nil
	case budget.inputTokensUsed >= bounds.MaxInputTokens:
		return "max_input_tokens", nil
	case budget.outputTokensUsed >= bounds.MaxOutputTokens:
		return "max_output_tokens", nil
	default:
		return "", nil
	}
}
