package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

// ListGenerationOutputs loads loop-owned per-node generation state in deterministic order.
func (g *GlobalDB) ListGenerationOutputs(
	ctx context.Context,
	runID looppkg.RunID,
	generation int,
) ([]looppkg.GenerationOutput, error) {
	if err := g.checkReady(ctx, "list loop generation outputs"); err != nil {
		return nil, err
	}
	if runID == "" {
		return nil, fmt.Errorf("%w: loop run id is required", looppkg.ErrValidation)
	}
	if generation <= 0 {
		return nil, fmt.Errorf("%w: generation must be positive", looppkg.ErrValidation)
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT generation, node_id, item_index, status, output_ref, task_run_id, child_loop_run_id
		 FROM loop_generation_outputs
		 WHERE loop_run_id = ?
		   AND generation = ?
		 ORDER BY node_id ASC, item_index ASC`,
		string(runID),
		generation,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list loop run %q generation %d outputs: %w", runID, generation, err)
	}
	defer rows.Close()

	outputs := make([]looppkg.GenerationOutput, 0)
	for rows.Next() {
		output, scanErr := scanGenerationOutput(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		outputs = append(outputs, output)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate loop run %q generation %d outputs: %w", runID, generation, err)
	}
	return outputs, nil
}

type generationOutputScanner interface {
	Scan(dest ...any) error
}

func scanGenerationOutput(row generationOutputScanner) (looppkg.GenerationOutput, error) {
	var output looppkg.GenerationOutput
	var outputRef sql.NullString
	var taskRunID sql.NullString
	var childLoopRunID sql.NullString
	if err := row.Scan(
		&output.Generation,
		&output.NodeID,
		&output.ItemIndex,
		&output.Status,
		&outputRef,
		&taskRunID,
		&childLoopRunID,
	); err != nil {
		return looppkg.GenerationOutput{}, fmt.Errorf("store: scan loop generation output: %w", err)
	}
	if outputRef.Valid {
		output.OutputRef = outputRef.String
	}
	if taskRunID.Valid {
		output.TaskRunID = taskRunID.String
	}
	if childLoopRunID.Valid {
		output.ChildLoopRunID = childLoopRunID.String
	}
	return output, nil
}

func getLoopOutputByRefWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	outputRef string,
) (json.RawMessage, error) {
	var raw string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT payload_json FROM loop_output_blobs WHERE output_ref = ?`,
		outputRef,
	).Scan(&raw); err != nil {
		if errorsIsNoRows(err) {
			return nil, looppkg.ErrOutputRefNotFound
		}
		return nil, fmt.Errorf("store: get loop output %q: %w", outputRef, err)
	}
	return json.RawMessage(raw), nil
}

func sweepOrphanedLoopOutputBlobsWithExecutor(ctx context.Context, exec taskSQLExecutor) error {
	if _, err := exec.ExecContext(
		ctx,
		`DELETE FROM loop_output_blobs
		 WHERE NOT EXISTS (
		   SELECT 1
		   FROM loop_generation_outputs
		   WHERE loop_generation_outputs.output_ref = loop_output_blobs.output_ref
		 )`,
	); err != nil {
		return fmt.Errorf("store: sweep orphaned loop output blobs: %w", err)
	}
	return nil
}

func upsertLoopOutputBlobWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	outputRef string,
	payload json.RawMessage,
	now time.Time,
) error {
	if !looppkg.OutputRefLooksContentAddressed(outputRef) {
		return fmt.Errorf("%w: output_ref is invalid: %q", looppkg.ErrValidation, outputRef)
	}
	if len(payload) == 0 {
		return fmt.Errorf("%w: loop output payload is required", looppkg.ErrValidation)
	}
	return store.UpsertLoopOutputBlob(ctx, exec, outputRef, payload, now)
}
