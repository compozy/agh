package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

const loopCatalogAggregateRunsSQL = `WITH requested_names(loop_name) AS (
		SELECT DISTINCT CAST(value AS TEXT) FROM json_each(?)
	)
	SELECT lr.loop_name,
		COUNT(*),
		SUM(CASE WHEN lr.status = 'done' THEN 1 ELSE 0 END),
		SUM(CASE WHEN lr.status IN ('failed', 'blocked', 'exhausted', 'stalled') THEN 1 ELSE 0 END)
	FROM requested_names AS requested
	JOIN loop_runs AS lr INDEXED BY idx_loop_runs_catalog
		ON lr.workspace_id = ? AND lr.loop_name = requested.loop_name
	WHERE lr.created_at >= ?
	GROUP BY lr.loop_name`

const loopCatalogLatestRunHeadColumnsSQL = `lr.id, lr.loop_name, lr.status, lr.created_at`

const loopCatalogLatestRunsSQL = `WITH requested_names(loop_name) AS (
		SELECT DISTINCT CAST(value AS TEXT) FROM json_each(?)
	), latest_ids(id) AS (
		SELECT (
			SELECT candidate.id
			FROM loop_runs AS candidate INDEXED BY idx_loop_runs_catalog
			WHERE candidate.workspace_id = ?
				AND candidate.loop_name = requested.loop_name
			ORDER BY candidate.created_at DESC, candidate.id DESC
			LIMIT 1
		)
		FROM requested_names AS requested
	)
	SELECT ` + loopCatalogLatestRunHeadColumnsSQL + `
	FROM loop_runs AS lr
	JOIN latest_ids ON latest_ids.id = lr.id`

type loopCatalogQueryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (g *GlobalDB) ListLoopCatalogRunSummaries(
	ctx context.Context,
	query looppkg.CatalogRunQuery,
) (summaries map[string]looppkg.CatalogRunSummary, err error) {
	if err := g.checkReady(ctx, "list loop catalog run summaries"); err != nil {
		return nil, err
	}
	normalized, err := normalizeLoopCatalogRunQuery(query)
	if err != nil {
		return nil, err
	}
	summaries = make(map[string]looppkg.CatalogRunSummary, len(normalized.LoopNames))
	for _, name := range normalized.LoopNames {
		summaries[name] = looppkg.CatalogRunSummary{LoopName: name}
	}
	if len(normalized.LoopNames) == 0 {
		return summaries, nil
	}

	namesJSON, err := json.Marshal(normalized.LoopNames)
	if err != nil {
		return nil, fmt.Errorf("store: encode loop catalog names: %w", err)
	}
	tx, err := g.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("store: begin loop catalog read: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("store: rollback loop catalog read: %w", rollbackErr))
		}
	}()

	if err := readLoopCatalogRunSummaries(ctx, tx, string(namesJSON), normalized, summaries); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("store: commit loop catalog read: %w", err)
	}
	return summaries, nil
}

func readLoopCatalogRunSummaries(
	ctx context.Context,
	executor loopCatalogQueryExecutor,
	namesJSON string,
	query looppkg.CatalogRunQuery,
	summaries map[string]looppkg.CatalogRunSummary,
) error {
	if err := readLoopCatalogAggregates(ctx, executor, namesJSON, query, summaries); err != nil {
		return err
	}
	return readLoopCatalogLatestRuns(ctx, executor, namesJSON, query.WorkspaceID, summaries)
}

func readLoopCatalogAggregates(
	ctx context.Context,
	executor loopCatalogQueryExecutor,
	namesJSON string,
	query looppkg.CatalogRunQuery,
	summaries map[string]looppkg.CatalogRunSummary,
) (err error) {
	rows, err := executor.QueryContext(
		ctx,
		loopCatalogAggregateRunsSQL,
		namesJSON,
		string(query.WorkspaceID),
		store.FormatTimestamp(query.AggregateAfter),
	)
	if err != nil {
		return fmt.Errorf("store: query loop catalog aggregates: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close loop catalog aggregate rows: %w", closeErr))
		}
	}()
	for rows.Next() {
		var name string
		var aggregate looppkg.CatalogRunAggregate
		if scanErr := rows.Scan(&name, &aggregate.Runs, &aggregate.Succeeded, &aggregate.Failed); scanErr != nil {
			return fmt.Errorf("store: scan loop catalog aggregate: %w", scanErr)
		}
		summary := summaries[name]
		summary.LoopName = name
		summary.Aggregate30d = aggregate
		summaries[name] = summary
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate loop catalog aggregates: %w", err)
	}
	return nil
}

func readLoopCatalogLatestRuns(
	ctx context.Context,
	executor loopCatalogQueryExecutor,
	namesJSON string,
	workspaceID looppkg.WorkspaceID,
	summaries map[string]looppkg.CatalogRunSummary,
) (err error) {
	rows, err := executor.QueryContext(ctx, loopCatalogLatestRunsSQL, namesJSON, string(workspaceID))
	if err != nil {
		return fmt.Errorf("store: query loop catalog latest runs: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close loop catalog latest rows: %w", closeErr))
		}
	}()
	for rows.Next() {
		var id string
		var loopName string
		var status string
		var createdAtRaw string
		if scanErr := rows.Scan(&id, &loopName, &status, &createdAtRaw); scanErr != nil {
			return fmt.Errorf("store: scan loop catalog latest run: %w", scanErr)
		}
		runStatus := looppkg.Status(status)
		if !runStatus.Valid() {
			return fmt.Errorf(
				"%w: loop catalog latest run %q status is invalid: %q",
				looppkg.ErrValidation,
				id,
				status,
			)
		}
		createdAt, parseErr := parseLoopRunTimestamp(createdAtRaw)
		if parseErr != nil {
			return fmt.Errorf("store: parse loop catalog latest run created_at: %w", parseErr)
		}
		run := looppkg.CatalogRunHead{
			ID:        looppkg.RunID(id),
			LoopName:  loopName,
			Status:    runStatus,
			CreatedAt: createdAt,
		}
		summary := summaries[loopName]
		summary.LoopName = loopName
		summary.LastRun = &run
		summaries[loopName] = summary
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate loop catalog latest runs: %w", err)
	}
	return nil
}

func normalizeLoopCatalogRunQuery(query looppkg.CatalogRunQuery) (looppkg.CatalogRunQuery, error) {
	normalized := query
	normalized.WorkspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(query.WorkspaceID)))
	normalized.AggregateAfter = query.AggregateAfter.UTC()
	if normalized.WorkspaceID == "" {
		return looppkg.CatalogRunQuery{}, fmt.Errorf("%w: workspace_id is required", looppkg.ErrValidation)
	}
	if normalized.AggregateAfter.IsZero() {
		return looppkg.CatalogRunQuery{}, fmt.Errorf("%w: aggregate_after is required", looppkg.ErrValidation)
	}
	names := make([]string, 0, len(query.LoopNames))
	seen := make(map[string]struct{}, len(query.LoopNames))
	for _, raw := range query.LoopNames {
		name, err := looppkg.ValidateName(raw)
		if err != nil {
			return looppkg.CatalogRunQuery{}, fmt.Errorf("%w: %v", looppkg.ErrValidation, err)
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	slices.Sort(names)
	normalized.LoopNames = names
	return normalized, nil
}

var _ looppkg.CatalogRunReader = (*GlobalDB)(nil)
