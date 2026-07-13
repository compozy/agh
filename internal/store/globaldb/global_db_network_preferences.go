package globaldb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
)

// PutNetworkSubscription upserts one peer delivery preference.
func (g *NetworkRepo) PutNetworkSubscription(ctx context.Context, entry store.NetworkSubscriptionEntry) error {
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
	if err := g.queries.UpsertNetworkSubscription(ctx, sqlcgen.UpsertNetworkSubscriptionParams{
		WorkspaceID:        normalized.WorkspaceID,
		Channel:            normalized.Channel,
		ThreadID:           normalized.ThreadID,
		PeerID:             normalized.PeerID,
		Mode:               normalized.Mode,
		KeywordFiltersJson: stringSliceJSONString(normalized.KeywordFilters),
		CreatedAt:          store.FormatTimestamp(normalized.CreatedAt),
		UpdatedAt:          store.FormatTimestamp(normalized.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("store: upsert network subscription: %w", err)
	}
	return nil
}

// ListNetworkSubscriptions returns peer delivery preferences in deterministic order.
func (g *NetworkRepo) ListNetworkSubscriptions(
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
	// dynamic-sql: optional identity filters and the caller-provided limit change the statement shape.
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
func (g *NetworkRepo) DeleteNetworkSubscription(ctx context.Context, ref store.NetworkSubscriptionRef) error {
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
	if err := g.queries.DeleteNetworkSubscription(ctx, sqlcgen.DeleteNetworkSubscriptionParams{
		WorkspaceID: normalized.WorkspaceID,
		Channel:     normalized.Channel,
		ThreadID:    normalized.ThreadID,
		PeerID:      normalized.PeerID,
	}); err != nil {
		return fmt.Errorf("store: delete network subscription: %w", err)
	}
	return nil
}

// GetNetworkDeliveryGuidanceState returns durable guidance state for one session.
func (g *NetworkRepo) GetNetworkDeliveryGuidanceState(
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
	row, err := g.queries.GetNetworkDeliveryGuidanceState(ctx, trimmed)
	if err != nil {
		return store.NetworkDeliveryGuidanceState{}, err
	}
	createdAt, err := store.ParseTimestamp(row.CreatedAt)
	if err != nil {
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: parse network delivery guidance state created_at: %w",
			err,
		)
	}
	updatedAt, err := store.ParseTimestamp(row.UpdatedAt)
	if err != nil {
		return store.NetworkDeliveryGuidanceState{}, fmt.Errorf(
			"store: parse network delivery guidance state updated_at: %w",
			err,
		)
	}
	return store.NetworkDeliveryGuidanceState{
		SessionID:                 row.SessionID,
		ReplyGuidanceDelivered:    row.ReplyGuidanceDelivered,
		ProtocolGuidanceDelivered: row.ProtocolGuidanceDelivered,
		CreatedAt:                 createdAt,
		UpdatedAt:                 updatedAt,
	}, nil
}

// PutNetworkDeliveryGuidanceState upserts durable guidance state for one session.
func (g *NetworkRepo) PutNetworkDeliveryGuidanceState(
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
	if err := g.queries.UpsertNetworkDeliveryGuidanceState(
		ctx,
		sqlcgen.UpsertNetworkDeliveryGuidanceStateParams{
			SessionID:                 normalized.SessionID,
			ReplyGuidanceDelivered:    normalized.ReplyGuidanceDelivered,
			ProtocolGuidanceDelivered: normalized.ProtocolGuidanceDelivered,
			CreatedAt:                 store.FormatTimestamp(normalized.CreatedAt),
			UpdatedAt:                 store.FormatTimestamp(normalized.UpdatedAt),
		},
	); err != nil {
		return fmt.Errorf("store: upsert network delivery guidance state: %w", err)
	}
	return nil
}

// PutNetworkTaskThreadOrigin upserts the origin link for one promoted task.
func (g *NetworkRepo) PutNetworkTaskThreadOrigin(ctx context.Context, origin store.NetworkTaskThreadOrigin) error {
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
	if err := g.queries.UpsertNetworkTaskThreadOrigin(ctx, sqlcgen.UpsertNetworkTaskThreadOriginParams{
		TaskID:               normalized.TaskID,
		WorkspaceID:          normalized.WorkspaceID,
		Channel:              normalized.Channel,
		ThreadID:             normalized.ThreadID,
		OriginMessageID:      normalized.OriginMessageID,
		Digest:               normalized.Digest,
		SourceMessageIdsJson: stringSliceJSONString(normalized.SourceMessageIDs),
		CreatedAt:            store.FormatTimestamp(normalized.CreatedAt),
		UpdatedAt:            store.FormatTimestamp(normalized.UpdatedAt),
	}); err != nil {
		return fmt.Errorf("store: upsert network task thread origin: %w", err)
	}
	return nil
}

// ListNetworkTaskThreadOrigins returns promoted task origins by task or thread.
func (g *NetworkRepo) ListNetworkTaskThreadOrigins(
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
	// dynamic-sql: optional task/thread filters and the caller-provided limit change the statement shape.
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
func (g *NetworkRepo) PutTaskDesignationRollup(ctx context.Context, rollup store.TaskDesignationRollup) error {
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
	if err := g.queries.UpsertTaskDesignationRollup(ctx, sqlcgen.UpsertTaskDesignationRollupParams{
		DesignationGroupID: normalized.DesignationGroupID,
		TaskID:             normalized.TaskID,
		SummaryJson:        string(normalized.SummaryJSON),
		CreatedAt:          store.FormatTimestamp(normalized.CreatedAt),
	}); err != nil {
		return fmt.Errorf("store: upsert task designation rollup: %w", err)
	}
	return nil
}

// ListTaskDesignationRollups returns rollups by task or group.
func (g *NetworkRepo) ListTaskDesignationRollups(
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
	// dynamic-sql: optional task/group filters and the caller-provided limit change the statement shape.
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
