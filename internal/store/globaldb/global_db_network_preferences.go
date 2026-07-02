package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// PutNetworkSubscription upserts one peer delivery preference.
func (g *GlobalDB) PutNetworkSubscription(ctx context.Context, entry store.NetworkSubscriptionEntry) error {
	if err := g.checkReady(ctx, "put network subscription"); err != nil {
		return err
	}
	normalized := normalizeNetworkSubscriptionEntry(entry)
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("store: validate network subscription: %w", err)
	}
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_subscriptions (
			workspace_id, channel, thread_id, peer_id, mode, keyword_filters_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, channel, thread_id, peer_id) DO UPDATE SET
			mode = excluded.mode,
			keyword_filters_json = excluded.keyword_filters_json,
			updated_at = excluded.updated_at`,
		normalized.WorkspaceID,
		normalized.Channel,
		normalized.ThreadID,
		normalized.PeerID,
		normalized.Mode,
		stringSliceJSONString(normalized.KeywordFilters),
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert network subscription: %w", err)
	}
	return nil
}

// ListNetworkSubscriptions returns peer delivery preferences in deterministic order.
func (g *GlobalDB) ListNetworkSubscriptions(
	ctx context.Context,
	query store.NetworkSubscriptionQuery,
) (entries []store.NetworkSubscriptionEntry, err error) {
	if err := g.checkReady(ctx, "list network subscriptions"); err != nil {
		return nil, err
	}
	normalized := store.NetworkSubscriptionQuery{
		WorkspaceID: strings.TrimSpace(query.WorkspaceID),
		Channel:     strings.TrimSpace(query.Channel),
		ThreadID:    strings.TrimSpace(query.ThreadID),
		PeerID:      strings.TrimSpace(query.PeerID),
		Limit:       query.Limit,
	}
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network subscription query: %w", err)
	}
	sqlQuery := `SELECT
		workspace_id, channel, thread_id, peer_id, mode, keyword_filters_json, created_at, updated_at
		FROM network_subscriptions`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", normalized.WorkspaceID),
		store.StringClause("channel", normalized.Channel),
		store.StringClause("thread_id", normalized.ThreadID),
		store.StringClause("peer_id", normalized.PeerID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY thread_id DESC, peer_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network subscriptions: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network subscription rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	entries = make([]store.NetworkSubscriptionEntry, 0)
	for rows.Next() {
		entry, scanErr := scanNetworkSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network subscriptions: %w", err)
	}
	return entries, nil
}

// DeleteNetworkSubscription removes one delivery preference.
func (g *GlobalDB) DeleteNetworkSubscription(ctx context.Context, ref store.NetworkSubscriptionRef) error {
	if err := g.checkReady(ctx, "delete network subscription"); err != nil {
		return err
	}
	normalized := store.NetworkSubscriptionRef{
		WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
		Channel:     strings.TrimSpace(ref.Channel),
		ThreadID:    strings.TrimSpace(ref.ThreadID),
		PeerID:      strings.TrimSpace(ref.PeerID),
	}
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("store: validate network subscription ref: %w", err)
	}
	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM network_subscriptions
		WHERE workspace_id = ? AND channel = ? AND thread_id = ? AND peer_id = ?`,
		normalized.WorkspaceID,
		normalized.Channel,
		normalized.ThreadID,
		normalized.PeerID,
	); err != nil {
		return fmt.Errorf("store: delete network subscription: %w", err)
	}
	return nil
}

// GetNetworkDeliveryGuidanceState returns durable guidance state for one session.
func (g *GlobalDB) GetNetworkDeliveryGuidanceState(
	ctx context.Context,
	sessionID string,
) (store.NetworkDeliveryGuidanceState, error) {
	if err := g.checkReady(ctx, "get network delivery guidance state"); err != nil {
		return store.NetworkDeliveryGuidanceState{}, err
	}
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: network delivery guidance session_id is required",
		)
	}
	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			session_id, reply_guidance_delivered, protocol_guidance_delivered, created_at, updated_at
		FROM network_delivery_guidance_state
		WHERE session_id = ?`,
		trimmed,
	)
	return scanNetworkDeliveryGuidanceState(row)
}

// PutNetworkDeliveryGuidanceState upserts durable guidance state for one session.
func (g *GlobalDB) PutNetworkDeliveryGuidanceState(
	ctx context.Context,
	state store.NetworkDeliveryGuidanceState,
) error {
	if err := g.checkReady(ctx, "put network delivery guidance state"); err != nil {
		return err
	}
	normalized := state
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("store: validate network delivery guidance state: %w", err)
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_delivery_guidance_state (
			session_id, reply_guidance_delivered, protocol_guidance_delivered, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			reply_guidance_delivered = excluded.reply_guidance_delivered,
			protocol_guidance_delivered = excluded.protocol_guidance_delivered,
			updated_at = excluded.updated_at`,
		normalized.SessionID,
		normalized.ReplyGuidanceDelivered,
		normalized.ProtocolGuidanceDelivered,
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert network delivery guidance state: %w", err)
	}
	return nil
}

// PutNetworkTaskThreadOrigin upserts the origin link for one promoted task.
func (g *GlobalDB) PutNetworkTaskThreadOrigin(ctx context.Context, origin store.NetworkTaskThreadOrigin) error {
	if err := g.checkReady(ctx, "put network task thread origin"); err != nil {
		return err
	}
	normalized := normalizeNetworkTaskThreadOrigin(origin)
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("store: validate network task thread origin: %w", err)
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_task_thread_origins (
			task_id, workspace_id, channel, thread_id, origin_message_id, digest,
			source_message_ids_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			channel = excluded.channel,
			thread_id = excluded.thread_id,
			origin_message_id = excluded.origin_message_id,
			digest = excluded.digest,
			source_message_ids_json = excluded.source_message_ids_json,
			updated_at = excluded.updated_at`,
		normalized.TaskID,
		normalized.WorkspaceID,
		normalized.Channel,
		normalized.ThreadID,
		normalized.OriginMessageID,
		normalized.Digest,
		stringSliceJSONString(normalized.SourceMessageIDs),
		store.FormatTimestamp(normalized.CreatedAt),
		store.FormatTimestamp(normalized.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert network task thread origin: %w", err)
	}
	return nil
}

// ListNetworkTaskThreadOrigins returns promoted task origins by task or thread.
func (g *GlobalDB) ListNetworkTaskThreadOrigins(
	ctx context.Context,
	query store.NetworkTaskThreadOriginQuery,
) (origins []store.NetworkTaskThreadOrigin, err error) {
	if err := g.checkReady(ctx, "list network task thread origins"); err != nil {
		return nil, err
	}
	normalized := store.NetworkTaskThreadOriginQuery{
		TaskID:      strings.TrimSpace(query.TaskID),
		WorkspaceID: strings.TrimSpace(query.WorkspaceID),
		Channel:     strings.TrimSpace(query.Channel),
		ThreadID:    strings.TrimSpace(query.ThreadID),
		Limit:       query.Limit,
	}
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network task thread origin query: %w", err)
	}
	sqlQuery := `SELECT
		task_id, workspace_id, channel, thread_id, origin_message_id, digest,
		source_message_ids_json, created_at, updated_at
		FROM network_task_thread_origins`
	where, args := store.BuildClauses(
		store.StringClause("task_id", normalized.TaskID),
		store.StringClause("workspace_id", normalized.WorkspaceID),
		store.StringClause("channel", normalized.Channel),
		store.StringClause("thread_id", normalized.ThreadID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY created_at DESC, task_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network task thread origins: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network task thread origin rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	origins = make([]store.NetworkTaskThreadOrigin, 0)
	for rows.Next() {
		origin, scanErr := scanNetworkTaskThreadOrigin(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		origins = append(origins, origin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network task thread origins: %w", err)
	}
	return origins, nil
}

// PutTaskDesignationRollup upserts one terminal rollup artifact.
func (g *GlobalDB) PutTaskDesignationRollup(ctx context.Context, rollup store.TaskDesignationRollup) error {
	if err := g.checkReady(ctx, "put task designation rollup"); err != nil {
		return err
	}
	normalized := rollup
	normalized.DesignationGroupID = strings.TrimSpace(normalized.DesignationGroupID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.SummaryJSON = json.RawMessage(strings.TrimSpace(string(normalized.SummaryJSON)))
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = g.now()
	}
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("store: validate task designation rollup: %w", err)
	}
	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO task_designation_rollups (
			designation_group_id, task_id, summary_json, created_at
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(designation_group_id) DO UPDATE SET
			task_id = excluded.task_id,
			summary_json = excluded.summary_json,
			created_at = excluded.created_at`,
		normalized.DesignationGroupID,
		normalized.TaskID,
		string(normalized.SummaryJSON),
		store.FormatTimestamp(normalized.CreatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert task designation rollup: %w", err)
	}
	return nil
}

// ListTaskDesignationRollups returns rollups by task or group.
func (g *GlobalDB) ListTaskDesignationRollups(
	ctx context.Context,
	query store.TaskDesignationRollupQuery,
) (rollups []store.TaskDesignationRollup, err error) {
	if err := g.checkReady(ctx, "list task designation rollups"); err != nil {
		return nil, err
	}
	normalized := store.TaskDesignationRollupQuery{
		DesignationGroupID: strings.TrimSpace(query.DesignationGroupID),
		TaskID:             strings.TrimSpace(query.TaskID),
		Limit:              query.Limit,
	}
	if err := normalized.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate task designation rollup query: %w", err)
	}
	sqlQuery := `SELECT designation_group_id, task_id, summary_json, created_at FROM task_designation_rollups`
	where, args := store.BuildClauses(
		store.StringClause("designation_group_id", normalized.DesignationGroupID),
		store.StringClause("task_id", normalized.TaskID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY created_at DESC, designation_group_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, normalized.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query task designation rollups: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close task designation rollup rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	rollups = make([]store.TaskDesignationRollup, 0)
	for rows.Next() {
		rollup, scanErr := scanTaskDesignationRollup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		rollups = append(rollups, rollup)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate task designation rollups: %w", err)
	}
	return rollups, nil
}

func normalizeNetworkSubscriptionEntry(entry store.NetworkSubscriptionEntry) store.NetworkSubscriptionEntry {
	normalized := store.NetworkSubscriptionEntry{
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		Channel:     strings.TrimSpace(entry.Channel),
		ThreadID:    strings.TrimSpace(entry.ThreadID),
		PeerID:      strings.TrimSpace(entry.PeerID),
		Mode:        strings.TrimSpace(entry.Mode),
		CreatedAt:   entry.CreatedAt,
		UpdatedAt:   entry.UpdatedAt,
	}
	normalized.KeywordFilters = normalizeStringSlice(entry.KeywordFilters)
	return normalized
}

func normalizeNetworkTaskThreadOrigin(origin store.NetworkTaskThreadOrigin) store.NetworkTaskThreadOrigin {
	return store.NetworkTaskThreadOrigin{
		TaskID:           strings.TrimSpace(origin.TaskID),
		WorkspaceID:      strings.TrimSpace(origin.WorkspaceID),
		Channel:          strings.TrimSpace(origin.Channel),
		ThreadID:         strings.TrimSpace(origin.ThreadID),
		OriginMessageID:  strings.TrimSpace(origin.OriginMessageID),
		Digest:           strings.TrimSpace(origin.Digest),
		SourceMessageIDs: normalizeStringSlice(origin.SourceMessageIDs),
		CreatedAt:        origin.CreatedAt,
		UpdatedAt:        origin.UpdatedAt,
	}
}

func scanNetworkSubscription(scanner rowScanner) (store.NetworkSubscriptionEntry, error) {
	var (
		entry        store.NetworkSubscriptionEntry
		filtersRaw   string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&entry.WorkspaceID,
		&entry.Channel,
		&entry.ThreadID,
		&entry.PeerID,
		&entry.Mode,
		&filtersRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return store.NetworkSubscriptionEntry{}, fmt.Errorf("store: scan network subscription: %w", err)
	}
	entry.KeywordFilters = stringSliceFromJSON(filtersRaw)
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return store.NetworkSubscriptionEntry{}, fmt.Errorf("store: parse network subscription created_at: %w", err)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.NetworkSubscriptionEntry{}, fmt.Errorf("store: parse network subscription updated_at: %w", err)
	}
	entry.CreatedAt = createdAt
	entry.UpdatedAt = updatedAt
	return entry, nil
}

func scanNetworkDeliveryGuidanceState(scanner rowScanner) (store.NetworkDeliveryGuidanceState, error) {
	var (
		state        store.NetworkDeliveryGuidanceState
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&state.SessionID,
		&state.ReplyGuidanceDelivered,
		&state.ProtocolGuidanceDelivered,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkDeliveryGuidanceState{}, err
		}
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: scan network delivery guidance state: %w",
			err,
		)
	}
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: parse network delivery guidance state created_at: %w",
			err,
		)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: parse network delivery guidance state updated_at: %w",
			err,
		)
	}
	state.CreatedAt = createdAt
	state.UpdatedAt = updatedAt
	return state, nil
}

func scanNetworkTaskThreadOrigin(scanner rowScanner) (store.NetworkTaskThreadOrigin, error) {
	var (
		origin       store.NetworkTaskThreadOrigin
		sourceRaw    string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := scanner.Scan(
		&origin.TaskID,
		&origin.WorkspaceID,
		&origin.Channel,
		&origin.ThreadID,
		&origin.OriginMessageID,
		&origin.Digest,
		&sourceRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return store.NetworkTaskThreadOrigin{}, fmt.Errorf("store: scan network task thread origin: %w", err)
	}
	origin.SourceMessageIDs = stringSliceFromJSON(sourceRaw)
	createdAt, err := store.ParseTimestamp(createdAtRaw)
	if err != nil {
		return store.NetworkTaskThreadOrigin{}, fmt.Errorf(
			"store: parse network task thread origin created_at: %w",
			err,
		)
	}
	updatedAt, err := store.ParseTimestamp(updatedAtRaw)
	if err != nil {
		return store.NetworkTaskThreadOrigin{}, fmt.Errorf(
			"store: parse network task thread origin updated_at: %w",
			err,
		)
	}
	origin.CreatedAt = createdAt
	origin.UpdatedAt = updatedAt
	return origin, nil
}

func scanTaskDesignationRollup(scanner rowScanner) (store.TaskDesignationRollup, error) {
	var (
		rollup     store.TaskDesignationRollup
		summaryRaw string
		createdRaw string
	)
	if err := scanner.Scan(
		&rollup.DesignationGroupID,
		&rollup.TaskID,
		&summaryRaw,
		&createdRaw,
	); err != nil {
		return store.TaskDesignationRollup{}, fmt.Errorf("store: scan task designation rollup: %w", err)
	}
	rollup.SummaryJSON = json.RawMessage(summaryRaw)
	createdAt, err := store.ParseTimestamp(createdRaw)
	if err != nil {
		return store.TaskDesignationRollup{}, fmt.Errorf("store: parse task designation rollup created_at: %w", err)
	}
	rollup.CreatedAt = createdAt
	return rollup, nil
}

func normalizeStringSlice(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func stringSliceJSONString(values []string) string {
	raw, err := json.Marshal(normalizeStringSlice(values))
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func stringSliceFromJSON(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return nil
	}
	return normalizeStringSlice(values)
}
