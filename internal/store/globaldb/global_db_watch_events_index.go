package globaldb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	watchpkg "github.com/compozy/agh/internal/loop/watch"
)

const watchEventsPendingOutputKind = "watch_events_pending"

// ListParkedWatchEventSubscriptions rebuilds the in-memory daemon index from
// durable watch-events pending output refs.
func (g *GlobalDB) ListParkedWatchEventSubscriptions(
	ctx context.Context,
) ([]looppkg.ParkedWatchEventSubscription, error) {
	if err := g.checkReady(ctx, "list parked watch-events subscriptions"); err != nil {
		return nil, err
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			lr.workspace_id,
			lr.id,
			lr.loop_name,
			lr.generation,
			lr.inputs_json,
			lgo.node_id,
			COALESCE(lgo.output_ref, '')
		   FROM loop_runs lr
		   JOIN loop_generation_outputs lgo
		     ON lgo.loop_run_id = lr.id
		    AND lgo.generation = lr.generation
		  WHERE lr.status = ?
		    AND json_extract(COALESCE(lgo.output_ref, '{}'), '$.kind') = ?
		  ORDER BY lr.workspace_id ASC, lr.id ASC, lgo.node_id ASC`,
		string(looppkg.StatusWatching),
		watchEventsPendingOutputKind,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list parked watch-events subscriptions: %w", err)
	}
	return readParkedWatchEventSubscriptions(rows)
}

// ListParkedWatchEventSubscriptionsForLoopRun returns the current parked
// subscriptions for one loop run without rebuilding the global index.
func (g *GlobalDB) ListParkedWatchEventSubscriptionsForLoopRun(
	ctx context.Context,
	loopRunID string,
) ([]looppkg.ParkedWatchEventSubscription, error) {
	if err := g.checkReady(ctx, "list parked watch-events subscriptions for loop run"); err != nil {
		return nil, err
	}
	loopRunID = strings.TrimSpace(loopRunID)
	if loopRunID == "" {
		return []looppkg.ParkedWatchEventSubscription{}, nil
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			lr.workspace_id,
			lr.id,
			lr.loop_name,
			lr.generation,
			lr.inputs_json,
			lgo.node_id,
			COALESCE(lgo.output_ref, '')
		   FROM loop_runs lr
		   JOIN loop_generation_outputs lgo
		     ON lgo.loop_run_id = lr.id
		    AND lgo.generation = lr.generation
		  WHERE lr.status = ?
		    AND json_extract(COALESCE(lgo.output_ref, '{}'), '$.kind') = ?
		    AND lr.id = ?
		  ORDER BY lr.workspace_id ASC, lr.id ASC, lgo.node_id ASC`,
		string(looppkg.StatusWatching),
		watchEventsPendingOutputKind,
		loopRunID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list parked watch-events subscriptions for loop run %q: %w", loopRunID, err)
	}
	return readParkedWatchEventSubscriptions(rows)
}

// ListParkedWatchEventSubscriptionsPage returns one bounded recovery page
// strictly after cursor in the same order as the full parked index.
func (g *GlobalDB) ListParkedWatchEventSubscriptionsPage(
	ctx context.Context,
	cursor looppkg.ParkedWatchEventScanCursor,
	limit int,
) ([]looppkg.ParkedWatchEventSubscription, error) {
	if err := g.checkReady(ctx, "list parked watch-events subscriptions page"); err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, fmt.Errorf("%w: parked watch-events page limit must be positive", looppkg.ErrValidation)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			lr.workspace_id,
			lr.id,
			lr.loop_name,
			lr.generation,
			lr.inputs_json,
			lgo.node_id,
			COALESCE(lgo.output_ref, '')
		   FROM loop_runs lr
		   JOIN loop_generation_outputs lgo
		     ON lgo.loop_run_id = lr.id
		    AND lgo.generation = lr.generation
		  WHERE lr.status = ?
		    AND json_extract(COALESCE(lgo.output_ref, '{}'), '$.kind') = ?
		    AND (
			lr.workspace_id > ?
			OR (lr.workspace_id = ? AND lr.id > ?)
			OR (lr.workspace_id = ? AND lr.id = ? AND lgo.node_id > ?)
		    )
		  ORDER BY lr.workspace_id ASC, lr.id ASC, lgo.node_id ASC
		  LIMIT ?`,
		string(looppkg.StatusWatching),
		watchEventsPendingOutputKind,
		strings.TrimSpace(cursor.WorkspaceID),
		strings.TrimSpace(cursor.WorkspaceID),
		strings.TrimSpace(cursor.LoopRunID),
		strings.TrimSpace(cursor.WorkspaceID),
		strings.TrimSpace(cursor.LoopRunID),
		strings.TrimSpace(cursor.NodeID),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list parked watch-events subscriptions page: %w", err)
	}
	return readParkedWatchEventSubscriptions(rows)
}

func readParkedWatchEventSubscriptions(
	rows *sql.Rows,
) ([]looppkg.ParkedWatchEventSubscription, error) {
	defer rows.Close()

	parked := make([]looppkg.ParkedWatchEventSubscription, 0)
	for rows.Next() {
		subscription, scanErr := scanParkedWatchEventSubscription(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "parked watch-events subscriptions query")
		}
		if len(subscription.Subscriptions) == 0 {
			continue
		}
		parked = append(parked, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate parked watch-events subscriptions: %w", err),
			"parked watch-events subscriptions query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "parked watch-events subscriptions query"); err != nil {
		return nil, err
	}
	return parked, nil
}

func scanParkedWatchEventSubscription(row rowScanner) (looppkg.ParkedWatchEventSubscription, error) {
	var (
		subscription looppkg.ParkedWatchEventSubscription
		inputsRaw    string
		outputRef    string
	)
	if err := row.Scan(
		&subscription.WorkspaceID,
		&subscription.LoopRunID,
		&subscription.LoopName,
		&subscription.Generation,
		&inputsRaw,
		&subscription.NodeID,
		&outputRef,
	); err != nil {
		return looppkg.ParkedWatchEventSubscription{}, fmt.Errorf(
			"store: scan parked watch-events subscription: %w",
			err,
		)
	}
	inputs, err := decodeParkedWatchEventsInputs(inputsRaw)
	if err != nil {
		return looppkg.ParkedWatchEventSubscription{}, err
	}
	state, ok, err := watchpkg.EventsPendingFromOutputRef(outputRef)
	if err != nil {
		return looppkg.ParkedWatchEventSubscription{}, err
	}
	if !ok {
		return looppkg.ParkedWatchEventSubscription{}, nil
	}
	subscription.WorkspaceID = strings.TrimSpace(subscription.WorkspaceID)
	subscription.LoopRunID = strings.TrimSpace(subscription.LoopRunID)
	subscription.LoopName = strings.TrimSpace(subscription.LoopName)
	subscription.NodeID = strings.TrimSpace(subscription.NodeID)
	subscription.Inputs = inputs
	subscription.Subscriptions = state.Subscriptions
	subscription.Cursors = state.Cursors
	return subscription, nil
}

func decodeParkedWatchEventsInputs(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}, nil
	}
	decoder := json.NewDecoder(bytes.NewBufferString(trimmed))
	decoder.UseNumber()
	var inputs map[string]any
	if err := decoder.Decode(&inputs); err != nil {
		return nil, fmt.Errorf("store: decode parked watch-events inputs: %w", err)
	}
	if inputs == nil {
		return map[string]any{}, nil
	}
	return inputs, nil
}
