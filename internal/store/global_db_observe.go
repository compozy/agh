package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// WriteEventSummary stores a lightweight cross-session summary entry.
func (g *GlobalDB) WriteEventSummary(ctx context.Context, summary EventSummary) error {
	if err := g.checkReady(ctx, "write event summary"); err != nil {
		return err
	}
	if err := summary.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(summary.ID) == "" {
		summary.ID = newID("sum")
	}
	if summary.Timestamp.IsZero() {
		summary.Timestamp = g.now()
	}

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO event_summaries (id, session_id, type, agent_name, summary, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		summary.ID,
		summary.SessionID,
		summary.Type,
		summary.AgentName,
		nullableString(summary.Summary),
		formatTimestamp(summary.Timestamp),
	); err != nil {
		return fmt.Errorf("store: insert event summary: %w", err)
	}
	return nil
}

// ListEventSummaries returns global event summaries filtered by the supplied options.
func (g *GlobalDB) ListEventSummaries(ctx context.Context, query EventSummaryQuery) ([]EventSummary, error) {
	if err := g.checkReady(ctx, "list event summaries"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	baseQuery := `SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`
	where, args := buildClauses(
		stringClause("session_id", query.SessionID),
		stringClause("agent_name", query.AgentName),
		stringClause("type", query.Type),
		timeClause("timestamp", ">=", query.Since),
	)
	baseQuery = appendWhere(baseQuery, where)

	sqlQuery := baseQuery
	if query.Limit > 0 {
		sqlQuery = `SELECT id, session_id, type, agent_name, summary, timestamp
			FROM (` + baseQuery + ` ORDER BY timestamp DESC LIMIT ?) AS recent_summaries
			ORDER BY timestamp ASC, id ASC`
		args = append(args, query.Limit)
	} else {
		sqlQuery += " ORDER BY timestamp ASC, id ASC"
	}

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query event summaries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	summaries := make([]EventSummary, 0)
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

// UpdateTokenStats merges one or more turns of token usage into the session aggregate.
func (g *GlobalDB) UpdateTokenStats(ctx context.Context, update TokenStatsUpdate) error {
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
		newID("tok"),
		update.SessionID,
		update.AgentName,
		nullableInt64(update.InputTokens),
		nullableInt64(update.OutputTokens),
		nullableInt64(update.TotalTokens),
		nullableFloat64(update.CostAmount),
		nullableStringPointer(update.CostCurrency),
		update.Turns,
		formatTimestamp(update.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert token stats for session %q: %w", update.SessionID, err)
	}

	return nil
}

// ListTokenStats returns aggregated token usage rows.
func (g *GlobalDB) ListTokenStats(ctx context.Context, query TokenStatsQuery) ([]TokenStats, error) {
	if err := g.checkReady(ctx, "list token stats"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	sqlQuery := `SELECT id, session_id, agent_name, input_tokens, output_tokens, total_tokens, total_cost, cost_currency, turn_count, updated_at FROM token_stats`
	where, args := buildClauses(
		stringClause("session_id", query.SessionID),
		stringClause("agent_name", query.AgentName),
	)
	sqlQuery = appendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY updated_at DESC, id DESC"
	sqlQuery, args = appendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query token stats: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	stats := make([]TokenStats, 0)
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

func scanEventSummary(scanner rowScanner) (EventSummary, error) {
	var (
		summary      EventSummary
		summaryText  sql.NullString
		timestampRaw string
	)
	if err := scanner.Scan(
		&summary.ID,
		&summary.SessionID,
		&summary.Type,
		&summary.AgentName,
		&summaryText,
		&timestampRaw,
	); err != nil {
		return EventSummary{}, fmt.Errorf("store: scan event summary: %w", err)
	}

	if summaryText.Valid {
		summary.Summary = summaryText.String
	}
	timestamp, err := parseTimestamp(timestampRaw)
	if err != nil {
		return EventSummary{}, err
	}
	summary.Timestamp = timestamp
	return summary, nil
}

func scanTokenStats(scanner rowScanner) (TokenStats, error) {
	var (
		stats        TokenStats
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
		return TokenStats{}, fmt.Errorf("store: scan token stats: %w", err)
	}

	stats.InputTokens = nullInt64(inputTokens)
	stats.OutputTokens = nullInt64(outputTokens)
	stats.TotalTokens = nullInt64(totalTokens)
	stats.TotalCost = nullFloat64(totalCost)
	stats.CostCurrency = nullString(costCurrency)

	updatedAt, err := parseTimestamp(updatedAtRaw)
	if err != nil {
		return TokenStats{}, err
	}
	stats.UpdatedAt = updatedAt

	return stats, nil
}
