package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

// ListThreadParticipants returns peers recorded in one public thread.
func (g *GlobalDB) ListThreadParticipants(
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

	rows, err := g.db.QueryContext(
		ctx,
		`SELECT workspace_id, channel, thread_id, peer_id, first_message_id, first_seen_at, last_seen_at
		FROM network_thread_participants
		WHERE workspace_id = ? AND channel = ? AND thread_id = ?
		ORDER BY last_seen_at DESC, peer_id ASC`,
		conversationRef.WorkspaceID,
		conversationRef.Channel,
		conversationRef.ThreadID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query network thread participants: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network thread participant rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	participants = make([]store.NetworkThreadParticipant, 0)
	for rows.Next() {
		participant, scanErr := scanNetworkThreadParticipant(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		participants = append(participants, participant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network thread participants: %w", err)
	}
	return participants, nil
}

// UpdateNetworkThreadPeerTokenStats merges one delivered prompt into the thread/peer aggregate.
func (g *GlobalDB) UpdateNetworkThreadPeerTokenStats(
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

	if _, err := g.db.ExecContext(
		ctx,
		`INSERT INTO network_thread_peer_token_stats (
			workspace_id, channel, thread_id, peer_id, delivered_count, prompt_size_bytes,
			estimated_prompt_tokens, first_delivered_at, last_delivered_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, channel, thread_id, peer_id) DO UPDATE SET
			delivered_count = network_thread_peer_token_stats.delivered_count + excluded.delivered_count,
			prompt_size_bytes = network_thread_peer_token_stats.prompt_size_bytes + excluded.prompt_size_bytes,
			estimated_prompt_tokens =
				network_thread_peer_token_stats.estimated_prompt_tokens + excluded.estimated_prompt_tokens,
			first_delivered_at = CASE
				WHEN network_thread_peer_token_stats.first_delivered_at <= excluded.first_delivered_at
					THEN network_thread_peer_token_stats.first_delivered_at
				ELSE excluded.first_delivered_at
			END,
			last_delivered_at = CASE
				WHEN network_thread_peer_token_stats.last_delivered_at >= excluded.last_delivered_at
					THEN network_thread_peer_token_stats.last_delivered_at
				ELSE excluded.last_delivered_at
			END,
			updated_at = excluded.updated_at`,
		strings.TrimSpace(update.WorkspaceID),
		strings.TrimSpace(update.Channel),
		strings.TrimSpace(update.ThreadID),
		strings.TrimSpace(update.PeerID),
		update.DeliveredCount,
		update.PromptSizeBytes,
		update.EstimatedPromptTokens,
		store.FormatTimestamp(update.DeliveredAt),
		store.FormatTimestamp(update.DeliveredAt),
		store.FormatTimestamp(update.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store: upsert network thread peer token stats: %w", err)
	}
	return nil
}

// ListNetworkThreadPeerTokenStats returns prompt-cost aggregates for one public thread.
func (g *GlobalDB) ListNetworkThreadPeerTokenStats(
	ctx context.Context,
	query store.NetworkThreadPeerTokenStatsQuery,
) (stats []store.NetworkThreadPeerTokenStats, err error) {
	if err := g.checkReady(ctx, "list network thread peer token stats"); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

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

func scanNetworkThreadParticipant(scanner rowScanner) (store.NetworkThreadParticipant, error) {
	var (
		participant store.NetworkThreadParticipant
		firstRaw    string
		lastRaw     string
	)
	if err := scanner.Scan(
		&participant.WorkspaceID,
		&participant.Channel,
		&participant.ThreadID,
		&participant.PeerID,
		&participant.FirstMessageID,
		&firstRaw,
		&lastRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkThreadParticipant{}, fmt.Errorf(
				"%w: network thread participant: %w",
				store.ErrNetworkConversationNotFound,
				err,
			)
		}
		return store.NetworkThreadParticipant{}, fmt.Errorf("store: scan network thread participant: %w", err)
	}
	firstSeenAt, err := store.ParseTimestamp(firstRaw)
	if err != nil {
		return store.NetworkThreadParticipant{}, fmt.Errorf(
			"store: parse network thread participant first_seen_at: %w",
			err,
		)
	}
	lastSeenAt, err := store.ParseTimestamp(lastRaw)
	if err != nil {
		return store.NetworkThreadParticipant{}, fmt.Errorf(
			"store: parse network thread participant last_seen_at: %w",
			err,
		)
	}
	participant.FirstSeenAt = firstSeenAt
	participant.LastSeenAt = lastSeenAt
	return participant, nil
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
