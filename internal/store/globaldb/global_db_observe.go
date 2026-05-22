package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	eventspkg "github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/store"
)

var memoryOutcomeCaseSQL = sync.OnceValue(buildMemoryOutcomeSQL)

// WriteEventSummary stores a lightweight cross-session summary entry.
func (g *GlobalDB) WriteEventSummary(ctx context.Context, summary store.EventSummary) error {
	if err := g.checkReady(ctx, "write event summary"); err != nil {
		return err
	}
	if err := g.populateEventSummaryProjections(ctx, &summary); err != nil {
		return err
	}
	if err := summary.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(summary.ID) == "" {
		summary.ID = store.NewID("sum")
	}
	if summary.Timestamp.IsZero() {
		summary.Timestamp = g.now()
	}
	summary.EventCorrelation = summary.Normalize()

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO event_summaries (
			id, session_id, workspace_id, type, agent_name, content_json, task_id, run_id, workflow_id, claim_token_hash,
			lease_until, coordinator_session_id, scheduler_reason, hook_event, hook_name,
			actor_kind, actor_id, release_reason, parent_session_id, root_session_id,
			spawn_depth, provider, outcome, summary, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		summary.ID,
		summary.SessionID,
		summary.WorkspaceID,
		summary.Type,
		summary.AgentName,
		string(summary.Content),
		summary.TaskID,
		summary.RunID,
		summary.WorkflowID,
		summary.ClaimTokenHash,
		formatEventSummaryLeaseUntil(summary.LeaseUntil),
		summary.CoordinatorSessionID,
		summary.SchedulerReason,
		summary.HookEvent,
		summary.HookName,
		summary.ActorKind,
		summary.ActorID,
		summary.ReleaseReason,
		summary.ParentSessionID,
		summary.RootSessionID,
		summary.SpawnDepth,
		summary.Provider,
		summary.Outcome,
		store.NullableString(summary.Summary),
		store.FormatTimestamp(summary.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert event summary: %w", err)
	}
	return nil
}

// ListEventSummaries returns global event summaries filtered by the supplied options.
func (g *GlobalDB) ListEventSummaries(
	ctx context.Context,
	query store.EventSummaryQuery,
) ([]store.EventSummary, error) {
	if err := g.validateEventSummaryQuery(ctx, query); err != nil {
		return nil, err
	}

	eventQuery := `SELECT 0 AS source_rank, rowid AS source_rowid, id, session_id, workspace_id,` +
		` type, agent_name, provider, outcome, content_json, task_id, run_id, workflow_id, claim_token_hash,` +
		` lease_until, coordinator_session_id,
		scheduler_reason, hook_event, hook_name, actor_kind, actor_id, release_reason,
		parent_session_id, root_session_id, spawn_depth, summary, timestamp FROM event_summaries`
	eventWhere, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("session_id", query.SessionID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("type", query.Type),
		store.StringClause("task_id", query.TaskID),
		store.StringClause("run_id", query.RunID),
		store.StringClause("actor_kind", query.ActorKind),
		store.StringClause("actor_id", query.ActorID),
		store.StringClause("provider", query.Provider),
		store.StringClause("outcome", query.Outcome),
		store.Int64Clause("rowid", ">", query.AfterSequence),
		store.TimeClause("timestamp", ">=", query.Since),
	)
	eventWhere, args = appendEventRegistryClauses(eventWhere, args, "type", query)
	eventQuery = store.AppendWhere(eventQuery, eventWhere)

	combinedQuery := eventQuery
	if strings.TrimSpace(query.SessionID) == "" {
		memoryQuery, memoryArgs := memoryEventSummaryQuery(query)
		combinedQuery += ` UNION ALL ` + memoryQuery
		args = append(args, memoryArgs...)
	}

	sqlQuery := eventSummaryListQuery(combinedQuery, query.Limit)
	if query.Limit > 0 {
		args = append(args, query.Limit)
	}

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query event summaries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summaries := make([]store.EventSummary, 0)
	for rows.Next() {
		summary, scanErr := scanEventSummary(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate event summaries: %w", err)
	}

	return summaries, nil
}

func memoryEventSummaryQuery(query store.EventSummaryQuery) (string, []any) {
	memoryQuery := `SELECT 1 AS source_rank,
		rowid AS source_rowid,
		'memevt-' || id AS id,
		'' AS session_id,
		COALESCE(json_extract(metadata, '$.workspace_id'), '') AS workspace_id,
		op AS type,
		COALESCE(agent_name, '') AS agent_name,
		'' AS provider,
		` + memoryOutcomeSQL() + ` AS outcome,
		'' AS content_json,
		'' AS task_id, '' AS run_id, '' AS workflow_id, '' AS claim_token_hash, '' AS lease_until,
		'' AS coordinator_session_id, '' AS scheduler_reason, '' AS hook_event, '' AS hook_name,
		'' AS actor_kind, '' AS actor_id, '' AS release_reason, '' AS parent_session_id,
		'' AS root_session_id, 0 AS spawn_depth,
		COALESCE(json_extract(metadata, '$.summary'), '') AS summary,
		printf('%s.%09dZ', strftime('%Y-%m-%dT%H:%M:%S', ts_ms / 1000, 'unixepoch'),
			(ts_ms % 1000) * 1000000) AS timestamp
		FROM memory_events`
	memoryWhere, memoryArgs := store.BuildClauses(
		store.StringClause("json_extract(metadata, '$.workspace_id')", query.WorkspaceID),
		store.StringClause("agent_name", query.AgentName),
		store.StringClause("op", query.Type),
		store.Int64Clause("rowid", ">", query.AfterSequence),
		store.Int64Clause("ts_ms", ">=", timestampMillis(query.Since)),
	)
	memoryWhere, memoryArgs = appendMemoryRegistryClauses(memoryWhere, memoryArgs, query)
	return store.AppendWhere(memoryQuery, memoryWhere), memoryArgs
}

func appendEventRegistryClauses(
	where []string,
	args []any,
	column string,
	query store.EventSummaryQuery,
) ([]string, []any) {
	if shouldApplyErrorOnlyClause(query) {
		where = append(where, "outcome IN (?, ?)")
		args = append(args, string(eventspkg.OutcomeFailure), string(eventspkg.OutcomeWarning))
	} else if errorOnlyContradictsOutcome(query) {
		where = append(where, "1 = 0")
	}
	if names := eventRegistryNamesForQuery(query); len(names) > 0 {
		where, args = appendNamesClause(where, args, column, names)
	}
	return where, args
}

func appendMemoryRegistryClauses(
	where []string,
	args []any,
	query store.EventSummaryQuery,
) ([]string, []any) {
	if strings.TrimSpace(query.Provider) != "" || strings.TrimSpace(query.TaskID) != "" ||
		strings.TrimSpace(query.RunID) != "" ||
		strings.TrimSpace(query.ActorKind) != "" || strings.TrimSpace(query.ActorID) != "" {
		where = append(where, "1 = 0")
		return where, args
	}
	if strings.TrimSpace(query.Outcome) != "" {
		names := memoryNamesForOutcomes(eventspkg.Outcome(strings.TrimSpace(query.Outcome)))
		where, args = appendNamesClause(where, args, "op", names)
	}
	if shouldApplyErrorOnlyClause(query) {
		names := memoryNamesForOutcomes(eventspkg.OutcomeFailure, eventspkg.OutcomeWarning)
		where, args = appendNamesClause(where, args, "op", names)
	} else if errorOnlyContradictsOutcome(query) {
		where = append(where, "1 = 0")
	}
	if component := strings.TrimSpace(query.Component); component != "" {
		if component != eventspkg.ComponentMemory {
			where = append(where, "1 = 0")
			return where, args
		}
		where, args = appendNamesClause(where, args, "op", eventspkg.NamesForComponent(eventspkg.ComponentMemory))
	}
	return where, args
}

func eventRegistryNamesForQuery(query store.EventSummaryQuery) []string {
	component := strings.TrimSpace(query.Component)
	if component == "" {
		return nil
	}
	return eventspkg.NamesForComponent(component)
}

func shouldApplyErrorOnlyClause(query store.EventSummaryQuery) bool {
	return query.ErrorOnly && strings.TrimSpace(query.Outcome) == ""
}

func errorOnlyContradictsOutcome(query store.EventSummaryQuery) bool {
	if !query.ErrorOnly {
		return false
	}
	switch eventspkg.Outcome(strings.TrimSpace(query.Outcome)) {
	case "", eventspkg.OutcomeFailure, eventspkg.OutcomeWarning:
		return false
	default:
		return true
	}
}

func memoryNamesForOutcomes(outcomes ...eventspkg.Outcome) []string {
	names := eventspkg.NamesForOutcomes(outcomes...)
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		meta, ok := eventspkg.Lookup(name)
		if ok && meta.Component == eventspkg.ComponentMemory {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func appendNamesClause(where []string, args []any, column string, names []string) ([]string, []any) {
	if len(names) == 0 {
		where = append(where, "1 = 0")
		return where, args
	}
	where = append(where, sqlInClause(column, len(names)))
	for _, name := range names {
		args = append(args, name)
	}
	return where, args
}

func sqlInClause(column string, count int) string {
	if count <= 0 {
		return "1 = 0"
	}
	placeholders := make([]string, count)
	for index := range placeholders {
		placeholders[index] = "?"
	}
	return column + " IN (" + strings.Join(placeholders, ", ") + ")"
}

func memoryOutcomeSQL() string {
	return memoryOutcomeCaseSQL()
}

func buildMemoryOutcomeSQL() string {
	var builder strings.Builder
	builder.WriteString("CASE op")
	for _, meta := range eventspkg.All() {
		if meta.Component != eventspkg.ComponentMemory {
			continue
		}
		builder.WriteString(" WHEN '")
		builder.WriteString(meta.Name)
		builder.WriteString("' THEN '")
		builder.WriteString(string(meta.Outcome))
		builder.WriteString("'")
	}
	builder.WriteString(" ELSE '")
	builder.WriteString(string(eventspkg.OutcomeInfo))
	builder.WriteString("' END")
	return builder.String()
}

func (g *GlobalDB) populateEventSummaryProjections(ctx context.Context, summary *store.EventSummary) error {
	summary.WorkspaceID = strings.TrimSpace(summary.WorkspaceID)
	summary.Provider = strings.TrimSpace(summary.Provider)
	summary.Outcome = strings.TrimSpace(summary.Outcome)
	if strings.TrimSpace(summary.Outcome) == "" {
		summary.Outcome = string(eventspkg.OutcomeFor(summary.Type))
	}
	if strings.TrimSpace(summary.SessionID) == "" ||
		(summary.WorkspaceID != "" && summary.Provider != "") {
		return nil
	}

	var workspaceID string
	var provider string
	err := g.db.QueryRowContext(
		ctx,
		`SELECT workspace_id, provider FROM sessions WHERE id = ?`,
		strings.TrimSpace(summary.SessionID),
	).Scan(&workspaceID, &provider)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("store: resolve event summary projections: %w", err)
	}
	if summary.WorkspaceID == "" {
		summary.WorkspaceID = strings.TrimSpace(workspaceID)
	}
	if summary.Provider == "" {
		summary.Provider = strings.TrimSpace(provider)
	}
	return nil
}

func eventSummaryListQuery(combinedQuery string, limit int) string {
	baseSelect := `SELECT source_rowid, id, session_id, workspace_id, type, agent_name, provider, outcome, content_json,` +
		` task_id, run_id, workflow_id, claim_token_hash, lease_until, coordinator_session_id,` +
		` scheduler_reason, hook_event,
		hook_name, actor_kind, actor_id, release_reason, parent_session_id, root_session_id,
		spawn_depth, summary, timestamp`
	if limit <= 0 {
		return baseSelect + ` FROM (` + combinedQuery + `) ORDER BY timestamp ASC, source_rank ASC, source_rowid ASC`
	}
	return baseSelect + ` FROM (` + combinedQuery + ` ORDER BY timestamp DESC, source_rank DESC, source_rowid DESC
		LIMIT ?) AS recent_summaries ORDER BY timestamp ASC, source_rank ASC, source_rowid ASC`
}

func (g *GlobalDB) validateEventSummaryQuery(
	ctx context.Context,
	query store.EventSummaryQuery,
) error {
	if err := g.checkReady(ctx, "list event summaries"); err != nil {
		return err
	}
	return query.Validate()
}

// SweepObservability removes global observability rows older than cutoff.
func (g *GlobalDB) SweepObservability(
	ctx context.Context,
	cutoff time.Time,
) (result store.ObservabilityRetentionSweepResult, err error) {
	if err := g.checkReady(ctx, "sweep observability"); err != nil {
		return store.ObservabilityRetentionSweepResult{}, err
	}
	if cutoff.IsZero() {
		return store.ObservabilityRetentionSweepResult{}, errors.New(
			"store: observability retention cutoff is required",
		)
	}

	result = store.ObservabilityRetentionSweepResult{CutoffAt: cutoff.UTC()}
	cutoffValue := store.FormatTimestamp(result.CutoffAt)

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return store.ObservabilityRetentionSweepResult{}, fmt.Errorf(
			"store: begin observability retention sweep: %w",
			err,
		)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "observability retention sweep"))
	}()

	if result.DeletedEventSummaries, err = deleteRowsBefore(
		ctx,
		tx,
		"event_summaries",
		`DELETE FROM event_summaries WHERE timestamp < ?`,
		cutoffValue,
	); err != nil {
		return store.ObservabilityRetentionSweepResult{}, err
	}
	if result.DeletedTokenStats, err = deleteRowsBefore(
		ctx,
		tx,
		"token_stats",
		`DELETE FROM token_stats WHERE updated_at < ?`,
		cutoffValue,
	); err != nil {
		return store.ObservabilityRetentionSweepResult{}, err
	}
	if result.DeletedPermissionLogs, err = deleteRowsBefore(
		ctx,
		tx,
		"permission_log",
		`DELETE FROM permission_log WHERE timestamp < ?`,
		cutoffValue,
	); err != nil {
		return store.ObservabilityRetentionSweepResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return store.ObservabilityRetentionSweepResult{}, fmt.Errorf(
			"store: commit observability retention sweep: %w",
			err,
		)
	}
	return result, nil
}

func deleteRowsBefore(ctx context.Context, tx *sql.Tx, label string, query string, cutoff string) (int64, error) {
	result, err := tx.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("store: delete old %s rows: %w", label, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store: count deleted %s rows: %w", label, err)
	}
	return count, nil
}

// UpdateTokenStats merges one or more turns of token usage into the session aggregate.
func (g *GlobalDB) UpdateTokenStats(ctx context.Context, update store.TokenStatsUpdate) error {
	if err := g.checkReady(ctx, "update token stats"); err != nil {
		return err
	}
	if err := update.Validate(); err != nil {
		return err
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = g.now()
	}
	if update.Turns <= 0 {
		update.Turns = 1
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO token_stats (
			id, session_id, agent_name, input_tokens, output_tokens, total_tokens,
			total_cost, cost_currency, turn_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id, agent_name) DO UPDATE SET
			input_tokens = CASE
				WHEN excluded.input_tokens IS NULL THEN token_stats.input_tokens
				WHEN token_stats.input_tokens IS NULL THEN excluded.input_tokens
				ELSE token_stats.input_tokens + excluded.input_tokens
			END,
			output_tokens = CASE
				WHEN excluded.output_tokens IS NULL THEN token_stats.output_tokens
				WHEN token_stats.output_tokens IS NULL THEN excluded.output_tokens
				ELSE token_stats.output_tokens + excluded.output_tokens
			END,
			total_tokens = CASE
				WHEN excluded.total_tokens IS NULL THEN token_stats.total_tokens
				WHEN token_stats.total_tokens IS NULL THEN excluded.total_tokens
				ELSE token_stats.total_tokens + excluded.total_tokens
			END,
			total_cost = CASE
				WHEN excluded.total_cost IS NULL THEN token_stats.total_cost
				WHEN token_stats.total_cost IS NULL THEN excluded.total_cost
				ELSE token_stats.total_cost + excluded.total_cost
			END,
			cost_currency = COALESCE(excluded.cost_currency, token_stats.cost_currency),
			turn_count = token_stats.turn_count + excluded.turn_count,
			updated_at = excluded.updated_at`,
		store.NewID("tok"),
		update.SessionID,
		update.AgentName,
		store.NullableInt64(update.InputTokens),
		store.NullableInt64(update.OutputTokens),
		store.NullableInt64(update.TotalTokens),
		store.NullableFloat64(update.CostAmount),
		store.NullableStringPointer(update.CostCurrency),
		update.Turns,
		store.FormatTimestamp(update.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert token stats for session %q: %w", update.SessionID, err)
	}

	return nil
}

// ListTokenStats returns aggregated token usage rows.
func (g *GlobalDB) ListTokenStats(ctx context.Context, query store.TokenStatsQuery) ([]store.TokenStats, error) {
	if err := g.checkReady(ctx, "list token stats"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at FROM token_stats`
	where, args := store.BuildClauses(
		store.StringClause("session_id", query.SessionID),
		store.StringClause("agent_name", query.AgentName),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, id DESC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query token stats: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	stats := make([]store.TokenStats, 0)
	for rows.Next() {
		stat, scanErr := scanTokenStats(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate token stats: %w", err)
	}

	return stats, nil
}

func scanEventSummary(scanner rowScanner) (store.EventSummary, error) {
	var (
		summary        store.EventSummary
		contentJSONRaw string
		summaryText    sql.NullString
		leaseUntilRaw  string
		timestampRaw   string
	)
	if err := scanner.Scan(
		&summary.Sequence,
		&summary.ID,
		&summary.SessionID,
		&summary.WorkspaceID,
		&summary.Type,
		&summary.AgentName,
		&summary.Provider,
		&summary.Outcome,
		&contentJSONRaw,
		&summary.TaskID,
		&summary.RunID,
		&summary.WorkflowID,
		&summary.ClaimTokenHash,
		&leaseUntilRaw,
		&summary.CoordinatorSessionID,
		&summary.SchedulerReason,
		&summary.HookEvent,
		&summary.HookName,
		&summary.ActorKind,
		&summary.ActorID,
		&summary.ReleaseReason,
		&summary.ParentSessionID,
		&summary.RootSessionID,
		&summary.SpawnDepth,
		&summaryText,
		&timestampRaw,
	); err != nil {
		return store.EventSummary{}, fmt.Errorf("store: scan event summary: %w", err)
	}

	if summaryText.Valid {
		summary.Summary = summaryText.String
	}
	if strings.TrimSpace(contentJSONRaw) != "" {
		summary.Content = append(json.RawMessage(nil), []byte(contentJSONRaw)...)
	}
	if parsedLeaseUntil, err := store.ParseNullableTimestamp(leaseUntilRaw); err != nil {
		return store.EventSummary{}, err
	} else if parsedLeaseUntil != nil {
		summary.LeaseUntil = parsedLeaseUntil
	}
	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return store.EventSummary{}, err
	}
	summary.Timestamp = timestamp
	return summary, nil
}

func formatEventSummaryLeaseUntil(value *time.Time) string {
	if value == nil {
		return ""
	}
	return store.FormatNullableTimestamp(*value)
}

func timestampMillis(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().UnixNano() / int64(time.Millisecond)
}

func scanTokenStats(scanner rowScanner) (store.TokenStats, error) {
	var (
		stats        store.TokenStats
		inputTokens  sql.NullInt64
		outputTokens sql.NullInt64
		totalTokens  sql.NullInt64
		totalCost    sql.NullFloat64
		costCurrency sql.NullString
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&stats.ID,
		&stats.SessionID,
		&stats.AgentName,
		&inputTokens,
		&outputTokens,
		&totalTokens,
		&totalCost,
		&costCurrency,
		&stats.TurnCount,
		&updatedAtRaw,
	); err != nil {
		return store.TokenStats{}, fmt.Errorf("store: scan token stats: %w", err)
	}

	stats.InputTokens = store.NullInt64(inputTokens)
	stats.OutputTokens = store.NullInt64(outputTokens)
	stats.TotalTokens = store.NullInt64(totalTokens)
	stats.TotalCost = store.NullFloat64(totalCost)
	stats.CostCurrency = store.NullString(costCurrency)

	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.TokenStats{}, err
	}
	stats.UpdatedAt = updatedAt

	return stats, nil
}
