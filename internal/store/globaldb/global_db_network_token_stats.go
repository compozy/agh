package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb/sqlcgen"
)

// ListThreadParticipants returns peers recorded in one public thread.
func (g *NetworkRepo) ListThreadParticipants(
	ctx context.Context,
	ref store.NetworkChannelRef,
	threadID string,
) (participants []store.NetworkThreadParticipant, err error) {
	if err := g.checkReady(ctx, "list network thread participants"); err != nil {
		return nil, err
	}
	conversationRef := store.NetworkConversationRef{
		WorkspaceID: ref.WorkspaceID,
		Channel:     ref.Channel,
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    strings.TrimSpace(threadID),
	}
	if err := conversationRef.Validate(); err != nil {
		return nil, err
	}

	rows, err := g.queries.ListNetworkThreadParticipants(ctx, sqlcgen.ListNetworkThreadParticipantsParams{
		WorkspaceID: conversationRef.WorkspaceID,
		Channel:     conversationRef.Channel,
		ThreadID:    conversationRef.ThreadID,
	})
	if err != nil {
		return nil, fmt.Errorf("store: query network thread participants: %w", err)
	}
	participants = make([]store.NetworkThreadParticipant, 0, len(rows))
	for _, row := range rows {
		firstSeenAt, parseErr := store.ParseTimestamp(row.FirstSeenAt)
		if parseErr != nil {
			return nil, fmt.Errorf("store: parse network thread participant first_seen_at: %w", parseErr)
		}
		lastSeenAt, parseErr := store.ParseTimestamp(row.LastSeenAt)
		if parseErr != nil {
			return nil, fmt.Errorf("store: parse network thread participant last_seen_at: %w", parseErr)
		}
		participants = append(participants, store.NetworkThreadParticipant{
			WorkspaceID: row.WorkspaceID, Channel: row.Channel, ThreadID: row.ThreadID,
			PeerID: row.PeerID, FirstMessageID: row.FirstMessageID,
			FirstSeenAt: firstSeenAt, LastSeenAt: lastSeenAt,
		})
	}
	return participants, nil
}

// UpdateNetworkThreadPeerTokenStats merges one delivered prompt into the thread/peer aggregate.
func (g *NetworkRepo) UpdateNetworkThreadPeerTokenStats(
	ctx context.Context,
	update store.NetworkThreadPeerTokenStatsUpdate,
) error {
	if err := g.checkReady(ctx, "update network thread peer token stats"); err != nil {
		return err
	}
	if err := update.Validate(); err != nil {
		return err
	}
	if update.DeliveredCount <= 0 {
		update.DeliveredCount = 1
	}
	if update.DeliveredAt.IsZero() {
		update.DeliveredAt = g.now()
	}
	if update.UpdatedAt.IsZero() {
		update.UpdatedAt = g.now()
	}

	if err := g.queries.UpsertNetworkThreadPeerTokenStats(
		ctx,
		sqlcgen.UpsertNetworkThreadPeerTokenStatsParams{
			WorkspaceID: strings.TrimSpace(update.WorkspaceID), Channel: strings.TrimSpace(update.Channel),
			ThreadID: strings.TrimSpace(update.ThreadID), PeerID: strings.TrimSpace(update.PeerID),
			DeliveredCount: update.DeliveredCount, PromptSizeBytes: update.PromptSizeBytes,
			EstimatedPromptTokens: update.EstimatedPromptTokens,
			FirstDeliveredAt:      store.FormatTimestamp(update.DeliveredAt),
			LastDeliveredAt:       store.FormatTimestamp(update.DeliveredAt),
			UpdatedAt:             store.FormatTimestamp(update.UpdatedAt),
		},
	); err != nil {
		return fmt.Errorf("store: upsert network thread peer token stats: %w", err)
	}
	return nil
}

// ListNetworkThreadPeerTokenStats returns prompt-cost aggregates for one public thread.
func (g *NetworkRepo) ListNetworkThreadPeerTokenStats(
	ctx context.Context,
	query store.NetworkThreadPeerTokenStatsQuery,
) (stats []store.NetworkThreadPeerTokenStats, err error) {
	if err := g.checkReady(ctx, "list network thread peer token stats"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	// dynamic-sql: optional thread/peer filters and the caller-provided limit change the statement shape.
	sqlQuery := `SELECT workspace_id, channel, thread_id, peer_id, delivered_count, prompt_size_bytes,
		estimated_prompt_tokens, first_delivered_at, last_delivered_at, updated_at
		FROM network_thread_peer_token_stats`
	where, args := store.BuildClauses(
		store.StringClause("workspace_id", query.WorkspaceID),
		store.StringClause("channel", query.Channel),
		store.StringClause("thread_id", query.ThreadID),
		store.StringClause("peer_id", query.PeerID),
	)
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY last_delivered_at DESC, peer_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network thread peer token stats: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network thread peer token stats rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	stats = make([]store.NetworkThreadPeerTokenStats, 0)
	for rows.Next() {
		stat, scanErr := scanNetworkThreadPeerTokenStats(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network thread peer token stats: %w", err)
	}
	return stats, nil
}

func scanNetworkThreadPeerTokenStats(scanner rowScanner) (store.NetworkThreadPeerTokenStats, error) {
	var (
		stats     store.NetworkThreadPeerTokenStats
		firstRaw  string
		lastRaw   string
		updateRaw string
	)
	if err := scanner.Scan(
		&stats.WorkspaceID,
		&stats.Channel,
		&stats.ThreadID,
		&stats.PeerID,
		&stats.DeliveredCount,
		&stats.PromptSizeBytes,
		&stats.EstimatedPromptTokens,
		&firstRaw,
		&lastRaw,
		&updateRaw,
	); err != nil {
		return store.NetworkThreadPeerTokenStats{}, fmt.Errorf("store: scan network thread peer token stats: %w", err)
	}
	firstDeliveredAt, err := store.ParseTimestamp(firstRaw)
	if err != nil {
		return store.NetworkThreadPeerTokenStats{}, fmt.Errorf(
			"store: parse network thread peer token stats first_delivered_at: %w",
			err,
		)
	}
	lastDeliveredAt, err := store.ParseTimestamp(lastRaw)
	if err != nil {
		return store.NetworkThreadPeerTokenStats{}, fmt.Errorf(
			"store: parse network thread peer token stats last_delivered_at: %w",
			err,
		)
	}
	updatedAt, err := store.ParseTimestamp(updateRaw)
	if err != nil {
		return store.NetworkThreadPeerTokenStats{}, fmt.Errorf(
			"store: parse network thread peer token stats updated_at: %w",
			err,
		)
	}
	stats.FirstDeliveredAt = firstDeliveredAt
	stats.LastDeliveredAt = lastDeliveredAt
	stats.UpdatedAt = updatedAt
	return stats, nil
}
