package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/store"
)

var _ modelcatalog.Store = (*GlobalDB)(nil)

type modelCatalogSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type modelCatalogRowKey struct {
	sourceID   string
	providerID string
	modelID    string
}

// ReplaceSourceRows atomically replaces all model rows and status for one provider-scoped source.
func (g *GlobalDB) ReplaceSourceRows(
	ctx context.Context,
	sourceID string,
	providerID string,
	rows []modelcatalog.ModelRow,
	status modelcatalog.SourceStatus,
) error {
	if err := g.checkReady(ctx, "replace model catalog source rows"); err != nil {
		return err
	}
	normalizedRows, normalizedStatus, err := normalizeModelCatalogReplacement(sourceID, providerID, rows, status)
	if err != nil {
		return err
	}

	return g.withModelCatalogImmediateTransaction(
		ctx,
		"model catalog source replacement",
		func(exec modelCatalogSQLExecutor) error {
			if err := upsertModelCatalogSourceStatus(ctx, exec, normalizedStatus); err != nil {
				return err
			}
			if _, err := exec.ExecContext(
				ctx,
				`DELETE FROM model_catalog_reasoning_efforts WHERE source_id = ? AND provider_id = ?`,
				normalizedStatus.SourceID,
				normalizedStatus.ProviderID,
			); err != nil {
				return fmt.Errorf("store: delete model catalog reasoning efforts: %w", err)
			}
			if _, err := exec.ExecContext(
				ctx,
				`DELETE FROM model_catalog_rows WHERE source_id = ? AND provider_id = ?`,
				normalizedStatus.SourceID,
				normalizedStatus.ProviderID,
			); err != nil {
				return fmt.Errorf("store: delete model catalog source rows: %w", err)
			}
			for _, row := range normalizedRows {
				if err := insertModelCatalogRow(ctx, exec, row); err != nil {
					return err
				}
				if err := insertModelCatalogReasoningEfforts(ctx, exec, row); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// ListRows returns deterministic catalog source rows matching the query.
func (g *GlobalDB) ListRows(
	ctx context.Context,
	opts modelcatalog.ListOptions,
) (catalogRows []modelcatalog.ModelRow, err error) {
	if err := g.checkReady(ctx, "list model catalog rows"); err != nil {
		return nil, err
	}
	sqlQuery := `SELECT
			source_id,
			provider_id,
			model_id,
			source_kind,
			priority,
			available,
			stale,
			refreshed_at,
			expires_at,
			display_name,
			context_window,
			max_input_tokens,
			max_output_tokens,
			supports_tools,
			supports_reasoning,
			default_reasoning_effort,
			cost_input_per_million,
			cost_output_per_million,
			last_error
		FROM model_catalog_rows`
	where, args := modelCatalogRowFilterClauses(opts, "")
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += ` ORDER BY provider_id ASC, model_id ASC, priority DESC, refreshed_at DESC, source_id ASC`

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query model catalog rows: %w", err)
	}
	defer func() {
		if closeErr := joinRowsCloseError(rows, nil, "model catalog row query"); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	catalogRows = make([]modelcatalog.ModelRow, 0)
	for rows.Next() {
		row, scanErr := scanModelCatalogRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		catalogRows = append(catalogRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate model catalog rows: %w", err)
	}

	efforts, err := listModelCatalogReasoningEfforts(ctx, g.db, opts)
	if err != nil {
		return nil, err
	}
	for index := range catalogRows {
		key := modelCatalogKey(catalogRows[index].SourceID, catalogRows[index].ProviderID, catalogRows[index].ModelID)
		catalogRows[index].ReasoningEfforts = efforts[key]
	}
	return catalogRows, nil
}

// ListSourceStatus returns provider-scoped source status rows.
func (g *GlobalDB) ListSourceStatus(
	ctx context.Context,
	providerID string,
) (statuses []modelcatalog.SourceStatus, err error) {
	if err := g.checkReady(ctx, "list model catalog source status"); err != nil {
		return nil, err
	}
	sqlQuery := `SELECT
			source_id,
			provider_id,
			source_kind,
			priority,
			refresh_state,
			last_refresh_at,
			next_refresh_at,
			last_success_at,
			last_error,
			row_count,
			stale
		FROM model_catalog_sources`
	where, args := store.BuildClauses(store.StringClause("provider_id", providerID))
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += ` ORDER BY provider_id ASC, source_id ASC`

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query model catalog source status: %w", err)
	}
	defer func() {
		if closeErr := joinRowsCloseError(
			rows,
			nil,
			"model catalog source status query",
		); closeErr != nil &&
			err == nil {
			err = closeErr
		}
	}()

	statuses = make([]modelcatalog.SourceStatus, 0)
	for rows.Next() {
		status, scanErr := scanModelCatalogSourceStatus(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		statuses = append(statuses, status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate model catalog source status: %w", err)
	}
	return statuses, nil
}

func normalizeModelCatalogReplacement(
	sourceID string,
	providerID string,
	rows []modelcatalog.ModelRow,
	status modelcatalog.SourceStatus,
) ([]modelcatalog.ModelRow, modelcatalog.SourceStatus, error) {
	trimmedSourceID, err := requireModelCatalogValue(sourceID, "source id")
	if err != nil {
		return nil, modelcatalog.SourceStatus{}, err
	}
	trimmedProviderID, err := requireModelCatalogValue(providerID, "provider id")
	if err != nil {
		return nil, modelcatalog.SourceStatus{}, err
	}
	normalizedStatus, err := normalizeModelCatalogStatus(trimmedSourceID, trimmedProviderID, status)
	if err != nil {
		return nil, modelcatalog.SourceStatus{}, err
	}

	normalizedRows := make([]modelcatalog.ModelRow, 0, len(rows))
	for index, row := range rows {
		normalizedRow, err := normalizeModelCatalogRow(trimmedSourceID, trimmedProviderID, normalizedStatus, row)
		if err != nil {
			return nil, modelcatalog.SourceStatus{}, fmt.Errorf("store: normalize model catalog row %d: %w", index, err)
		}
		if normalizedStatus.Priority == 0 {
			normalizedStatus.Priority = normalizedRow.Priority
		}
		if normalizedRow.Priority != normalizedStatus.Priority {
			return nil, modelcatalog.SourceStatus{}, fmt.Errorf(
				"store: model catalog row %q priority %d does not match source priority %d",
				normalizedRow.ModelID,
				normalizedRow.Priority,
				normalizedStatus.Priority,
			)
		}
		normalizedRows = append(normalizedRows, normalizedRow)
	}
	normalizedStatus.RowCount = len(normalizedRows)
	return normalizedRows, normalizedStatus, nil
}

func normalizeModelCatalogStatus(
	sourceID string,
	providerID string,
	status modelcatalog.SourceStatus,
) (modelcatalog.SourceStatus, error) {
	normalized := status
	if normalized.SourceID = strings.TrimSpace(normalized.SourceID); normalized.SourceID == "" {
		normalized.SourceID = sourceID
	}
	if normalized.ProviderID = strings.TrimSpace(normalized.ProviderID); normalized.ProviderID == "" {
		normalized.ProviderID = providerID
	}
	if normalized.SourceID != sourceID {
		return modelcatalog.SourceStatus{}, fmt.Errorf(
			"store: model catalog status source id %q does not match %q",
			normalized.SourceID,
			sourceID,
		)
	}
	if normalized.ProviderID != providerID {
		return modelcatalog.SourceStatus{}, fmt.Errorf(
			"store: model catalog status provider id %q does not match %q",
			normalized.ProviderID,
			providerID,
		)
	}
	normalized.SourceKind = modelcatalog.SourceKind(strings.TrimSpace(string(normalized.SourceKind)))
	if normalized.SourceKind == "" {
		return modelcatalog.SourceStatus{}, fmt.Errorf("store: model catalog status source kind is required")
	}
	normalized.RefreshState = strings.TrimSpace(normalized.RefreshState)
	if normalized.RefreshState == "" {
		normalized.RefreshState = string(modelcatalog.RefreshStateIdle)
	}
	normalized.LastError = strings.TrimSpace(normalized.LastError)
	if normalized.RowCount < 0 {
		return modelcatalog.SourceStatus{}, fmt.Errorf(
			"store: model catalog status row count %d is invalid",
			normalized.RowCount,
		)
	}
	return normalized, nil
}

func normalizeModelCatalogRow(
	sourceID string,
	providerID string,
	status modelcatalog.SourceStatus,
	row modelcatalog.ModelRow,
) (modelcatalog.ModelRow, error) {
	normalized := row
	if normalized.SourceID = strings.TrimSpace(normalized.SourceID); normalized.SourceID == "" {
		normalized.SourceID = sourceID
	}
	if normalized.ProviderID = strings.TrimSpace(normalized.ProviderID); normalized.ProviderID == "" {
		normalized.ProviderID = providerID
	}
	if normalized.SourceID != sourceID {
		return modelcatalog.ModelRow{}, fmt.Errorf(
			"source id %q does not match %q",
			normalized.SourceID,
			sourceID,
		)
	}
	if normalized.ProviderID != providerID {
		return modelcatalog.ModelRow{}, fmt.Errorf(
			"provider id %q does not match %q",
			normalized.ProviderID,
			providerID,
		)
	}
	modelID, err := requireModelCatalogValue(normalized.ModelID, "model id")
	if err != nil {
		return modelcatalog.ModelRow{}, err
	}
	normalized.ModelID = modelID
	normalized.SourceKind = modelcatalog.SourceKind(strings.TrimSpace(string(normalized.SourceKind)))
	if normalized.SourceKind == "" {
		normalized.SourceKind = status.SourceKind
	}
	if normalized.SourceKind != status.SourceKind {
		return modelcatalog.ModelRow{}, fmt.Errorf(
			"source kind %q does not match status source kind %q",
			normalized.SourceKind,
			status.SourceKind,
		)
	}
	normalized.DisplayName = strings.TrimSpace(normalized.DisplayName)
	normalized.LastError = strings.TrimSpace(normalized.LastError)
	if normalized.DefaultReasoningEffort != nil {
		effort := modelcatalog.ReasoningEffort(strings.TrimSpace(string(*normalized.DefaultReasoningEffort)))
		if effort == "" {
			normalized.DefaultReasoningEffort = nil
		} else {
			normalized.DefaultReasoningEffort = &effort
		}
	}
	for index, effort := range normalized.ReasoningEfforts {
		trimmed := modelcatalog.ReasoningEffort(strings.TrimSpace(string(effort)))
		if trimmed == "" {
			return modelcatalog.ModelRow{}, fmt.Errorf("reasoning effort %d is required", index)
		}
		normalized.ReasoningEfforts[index] = trimmed
	}
	return normalized, nil
}

func requireModelCatalogValue(value string, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("store: model catalog %s is required", field)
	}
	return trimmed, nil
}

func upsertModelCatalogSourceStatus(
	ctx context.Context,
	exec modelCatalogSQLExecutor,
	status modelcatalog.SourceStatus,
) error {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO model_catalog_sources (
			source_id,
			provider_id,
			source_kind,
			priority,
			refresh_state,
			last_refresh_at,
			next_refresh_at,
			last_success_at,
			last_error,
			row_count,
			stale
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_id, provider_id) DO UPDATE SET
			source_kind = excluded.source_kind,
			priority = excluded.priority,
			refresh_state = excluded.refresh_state,
			last_refresh_at = excluded.last_refresh_at,
			next_refresh_at = excluded.next_refresh_at,
			last_success_at = excluded.last_success_at,
			last_error = excluded.last_error,
			row_count = excluded.row_count,
			stale = excluded.stale`,
		status.SourceID,
		status.ProviderID,
		string(status.SourceKind),
		status.Priority,
		status.RefreshState,
		store.FormatNullableTimestamp(status.LastRefresh),
		store.FormatNullableTimestamp(status.NextRefresh),
		store.FormatNullableTimestamp(status.LastSuccess),
		status.LastError,
		status.RowCount,
		boolToSQLiteInt(status.Stale),
	); err != nil {
		return fmt.Errorf("store: upsert model catalog source status: %w", err)
	}
	return nil
}

func insertModelCatalogRow(ctx context.Context, exec modelCatalogSQLExecutor, row modelcatalog.ModelRow) error {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO model_catalog_rows (
			source_id,
			provider_id,
			model_id,
			source_kind,
			priority,
			available,
			stale,
			refreshed_at,
			expires_at,
			display_name,
			context_window,
			max_input_tokens,
			max_output_tokens,
			supports_tools,
			supports_reasoning,
			default_reasoning_effort,
			cost_input_per_million,
			cost_output_per_million,
			last_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.SourceID,
		row.ProviderID,
		row.ModelID,
		string(row.SourceKind),
		row.Priority,
		nullableBoolToSQLiteInt(row.Available),
		boolToSQLiteInt(row.Stale),
		store.FormatNullableTimestamp(row.RefreshedAt),
		store.FormatNullableTimestamp(row.ExpiresAt),
		row.DisplayName,
		store.NullableInt64(row.ContextWindow),
		store.NullableInt64(row.MaxInputTokens),
		store.NullableInt64(row.MaxOutputTokens),
		nullableBoolToSQLiteInt(row.SupportsTools),
		nullableBoolToSQLiteInt(row.SupportsReasoning),
		nullableReasoningEffort(row.DefaultReasoningEffort),
		store.NullableFloat64(row.CostInputPerMillion),
		store.NullableFloat64(row.CostOutputPerMillion),
		row.LastError,
	); err != nil {
		return fmt.Errorf(
			"store: insert model catalog row %q/%q/%q: %w",
			row.SourceID,
			row.ProviderID,
			row.ModelID,
			err,
		)
	}
	return nil
}

func insertModelCatalogReasoningEfforts(
	ctx context.Context,
	exec modelCatalogSQLExecutor,
	row modelcatalog.ModelRow,
) error {
	for rank, effort := range row.ReasoningEfforts {
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO model_catalog_reasoning_efforts (
				source_id,
				provider_id,
				model_id,
				effort,
				rank
			) VALUES (?, ?, ?, ?, ?)`,
			row.SourceID,
			row.ProviderID,
			row.ModelID,
			string(effort),
			rank,
		); err != nil {
			return fmt.Errorf("store: insert model catalog reasoning effort %q: %w", effort, err)
		}
	}
	return nil
}

func listModelCatalogReasoningEfforts(
	ctx context.Context,
	exec modelCatalogSQLExecutor,
	opts modelcatalog.ListOptions,
) (efforts map[modelCatalogRowKey][]modelcatalog.ReasoningEffort, err error) {
	sqlQuery := `SELECT
			e.source_id,
			e.provider_id,
			e.model_id,
			e.effort
		FROM model_catalog_reasoning_efforts e
		JOIN model_catalog_rows r
			ON r.source_id = e.source_id
			AND r.provider_id = e.provider_id
			AND r.model_id = e.model_id`
	where, args := modelCatalogRowFilterClauses(opts, "r")
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += ` ORDER BY e.source_id ASC, e.provider_id ASC, e.model_id ASC, e.rank ASC, e.effort ASC`

	rows, err := exec.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query model catalog reasoning efforts: %w", err)
	}
	defer func() {
		if closeErr := joinRowsCloseError(
			rows,
			nil,
			"model catalog reasoning effort query",
		); closeErr != nil &&
			err == nil {
			err = closeErr
		}
	}()

	efforts = make(map[modelCatalogRowKey][]modelcatalog.ReasoningEffort)
	for rows.Next() {
		var (
			sourceID   string
			providerID string
			modelID    string
			effort     string
		)
		if err := rows.Scan(&sourceID, &providerID, &modelID, &effort); err != nil {
			return nil, fmt.Errorf("store: scan model catalog reasoning effort: %w", err)
		}
		key := modelCatalogKey(sourceID, providerID, modelID)
		efforts[key] = append(efforts[key], modelcatalog.ReasoningEffort(effort))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate model catalog reasoning efforts: %w", err)
	}
	return efforts, nil
}

func scanModelCatalogRow(scanner interface{ Scan(dest ...any) error }) (modelcatalog.ModelRow, error) {
	var (
		row                       modelcatalog.ModelRow
		sourceKind                string
		availableRaw              sql.NullInt64
		staleRaw                  int
		refreshedAtRaw            string
		expiresAtRaw              string
		contextWindowRaw          sql.NullInt64
		maxInputTokensRaw         sql.NullInt64
		maxOutputTokensRaw        sql.NullInt64
		supportsToolsRaw          sql.NullInt64
		supportsReasoningRaw      sql.NullInt64
		defaultReasoningEffortRaw sql.NullString
		costInputPerMillionRaw    sql.NullFloat64
		costOutputPerMillionRaw   sql.NullFloat64
	)
	if err := scanner.Scan(
		&row.SourceID,
		&row.ProviderID,
		&row.ModelID,
		&sourceKind,
		&row.Priority,
		&availableRaw,
		&staleRaw,
		&refreshedAtRaw,
		&expiresAtRaw,
		&row.DisplayName,
		&contextWindowRaw,
		&maxInputTokensRaw,
		&maxOutputTokensRaw,
		&supportsToolsRaw,
		&supportsReasoningRaw,
		&defaultReasoningEffortRaw,
		&costInputPerMillionRaw,
		&costOutputPerMillionRaw,
		&row.LastError,
	); err != nil {
		return modelcatalog.ModelRow{}, fmt.Errorf("store: scan model catalog row: %w", err)
	}
	row.SourceKind = modelcatalog.SourceKind(sourceKind)
	available, err := nullableSQLiteIntToBool(availableRaw, "available")
	if err != nil {
		return modelcatalog.ModelRow{}, err
	}
	row.Available = available
	row.Stale = staleRaw != 0
	if row.RefreshedAt, err = parseOptionalModelCatalogTimestamp(refreshedAtRaw, "refreshed_at"); err != nil {
		return modelcatalog.ModelRow{}, err
	}
	if row.ExpiresAt, err = parseOptionalModelCatalogTimestamp(expiresAtRaw, "expires_at"); err != nil {
		return modelcatalog.ModelRow{}, err
	}
	row.ContextWindow = store.NullInt64(contextWindowRaw)
	row.MaxInputTokens = store.NullInt64(maxInputTokensRaw)
	row.MaxOutputTokens = store.NullInt64(maxOutputTokensRaw)
	if row.SupportsTools, err = nullableSQLiteIntToBool(supportsToolsRaw, "supports_tools"); err != nil {
		return modelcatalog.ModelRow{}, err
	}
	if row.SupportsReasoning, err = nullableSQLiteIntToBool(supportsReasoningRaw, "supports_reasoning"); err != nil {
		return modelcatalog.ModelRow{}, err
	}
	row.DefaultReasoningEffort = nullReasoningEffort(defaultReasoningEffortRaw)
	row.CostInputPerMillion = store.NullFloat64(costInputPerMillionRaw)
	row.CostOutputPerMillion = store.NullFloat64(costOutputPerMillionRaw)
	return row, nil
}

func scanModelCatalogSourceStatus(scanner interface{ Scan(dest ...any) error }) (modelcatalog.SourceStatus, error) {
	var (
		status         modelcatalog.SourceStatus
		sourceKind     string
		lastRefreshRaw string
		nextRefreshRaw string
		lastSuccessRaw string
		staleRaw       int
	)
	if err := scanner.Scan(
		&status.SourceID,
		&status.ProviderID,
		&sourceKind,
		&status.Priority,
		&status.RefreshState,
		&lastRefreshRaw,
		&nextRefreshRaw,
		&lastSuccessRaw,
		&status.LastError,
		&status.RowCount,
		&staleRaw,
	); err != nil {
		return modelcatalog.SourceStatus{}, fmt.Errorf("store: scan model catalog source status: %w", err)
	}
	var err error
	status.SourceKind = modelcatalog.SourceKind(sourceKind)
	if status.LastRefresh, err = parseOptionalModelCatalogTimestamp(lastRefreshRaw, "last_refresh_at"); err != nil {
		return modelcatalog.SourceStatus{}, err
	}
	if status.NextRefresh, err = parseOptionalModelCatalogTimestamp(nextRefreshRaw, "next_refresh_at"); err != nil {
		return modelcatalog.SourceStatus{}, err
	}
	if status.LastSuccess, err = parseOptionalModelCatalogTimestamp(lastSuccessRaw, "last_success_at"); err != nil {
		return modelcatalog.SourceStatus{}, err
	}
	status.Stale = staleRaw != 0
	return status, nil
}

func modelCatalogRowFilterClauses(opts modelcatalog.ListOptions, alias string) ([]string, []any) {
	column := func(name string) string {
		if strings.TrimSpace(alias) == "" {
			return name
		}
		return strings.TrimSpace(alias) + "." + name
	}

	where := make([]string, 0, 3)
	args := make([]any, 0, 2)
	if providerID := strings.TrimSpace(opts.ProviderID); providerID != "" {
		where = append(where, column("provider_id")+" = ?")
		args = append(args, providerID)
	}
	if sourceID := strings.TrimSpace(opts.SourceID); sourceID != "" {
		where = append(where, column("source_id")+" = ?")
		args = append(args, sourceID)
	}
	if !opts.IncludeStale && !opts.IncludeAll {
		where = append(where, column("stale")+" = 0")
	}
	return where, args
}

func (g *GlobalDB) withModelCatalogImmediateTransaction(
	ctx context.Context,
	action string,
	run func(exec modelCatalogSQLExecutor) error,
) (err error) {
	conn, err := g.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open connection for %s: %w", action, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("store: close %s transaction connection: %w", action, closeErr)
		}
	}()

	rollbackCtx := context.WithoutCancel(ctx)
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("store: begin immediate %s transaction: %w", action, err)
	}

	finished := false
	defer func() {
		if !finished {
			joinCleanupError(&err, rollbackImmediate(rollbackCtx, conn, action))
		}
	}()

	if err := run(conn); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("store: commit %s transaction: %w", action, err)
	}
	finished = true
	return nil
}

func modelCatalogKey(sourceID string, providerID string, modelID string) modelCatalogRowKey {
	return modelCatalogRowKey{
		sourceID:   sourceID,
		providerID: providerID,
		modelID:    modelID,
	}
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableBoolToSQLiteInt(value *bool) any {
	if value == nil {
		return nil
	}
	return boolToSQLiteInt(*value)
}

func nullableSQLiteIntToBool(value sql.NullInt64, field string) (*bool, error) {
	if !value.Valid {
		return nil, nil
	}
	switch value.Int64 {
	case 0:
		converted := false
		return &converted, nil
	case 1:
		converted := true
		return &converted, nil
	default:
		return nil, fmt.Errorf("store: model catalog %s boolean value %d is invalid", field, value.Int64)
	}
}

func nullableReasoningEffort(value *modelcatalog.ReasoningEffort) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(*value))
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullReasoningEffort(value sql.NullString) *modelcatalog.ReasoningEffort {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	effort := modelcatalog.ReasoningEffort(trimmed)
	return &effort
}

func parseOptionalModelCatalogTimestamp(value string, field string) (time.Time, error) {
	parsed, err := store.ParseNullableTimestamp(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("store: parse model catalog %s: %w", field, err)
	}
	if parsed == nil {
		return time.Time{}, nil
	}
	return *parsed, nil
}
