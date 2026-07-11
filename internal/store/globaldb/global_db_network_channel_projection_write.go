package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func updateNetworkChannelProjection(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	if _, err := exec.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO network_channel_stats (
			workspace_id, channel, message_count, presence_count, historical_participant_count,
			last_activity_at, last_activity_sequence, last_presence_at, last_presence_sequence,
			last_message_id, last_message_sequence, last_message_preview
		) VALUES (?, ?, 0, 0, 0, NULL, 0, NULL, 0, '', 0, '')`,
		entry.WorkspaceID,
		entry.Channel,
	); err != nil {
		return fmt.Errorf("store: ensure network channel projection: %w", err)
	}

	switch {
	case strings.TrimSpace(entry.Kind) == store.NetworkKindGreet:
		if err := updateNetworkChannelPresenceProjection(ctx, exec, entry); err != nil {
			return err
		}
	case networkProjectionMessageIsPublic(entry):
		if err := updateNetworkChannelMessageProjection(ctx, exec, entry); err != nil {
			return err
		}
	}
	return updateNetworkChannelParticipants(ctx, exec, entry)
}

func updateNetworkChannelPresenceProjection(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	timestamp := store.FormatTimestamp(entry.Timestamp)
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_channel_stats
		 SET presence_count = presence_count + 1,
			last_presence_at = ?,
			last_presence_sequence = ?
		 WHERE workspace_id = ? AND channel = ?`,
		timestamp,
		entry.Sequence,
		entry.WorkspaceID,
		entry.Channel,
	); err != nil {
		return fmt.Errorf("store: update network channel presence projection: %w", err)
	}
	return nil
}

func updateNetworkChannelMessageProjection(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	timestamp := store.FormatTimestamp(entry.Timestamp)
	messageID := strings.TrimSpace(entry.MessageID)
	preview := strings.TrimSpace(entry.PreviewText)
	if preview == "" {
		preview = strings.TrimSpace(entry.Text)
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_channel_stats
		 SET message_count = message_count + 1,
			last_message_preview = ?,
			last_message_id = ?,
			last_message_sequence = ?,
			last_activity_at = ?,
			last_activity_sequence = ?
		 WHERE workspace_id = ? AND channel = ?`,
		preview,
		messageID,
		entry.Sequence,
		timestamp,
		entry.Sequence,
		entry.WorkspaceID,
		entry.Channel,
	); err != nil {
		return fmt.Errorf("store: update network channel message projection: %w", err)
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO network_channel_kind_counts (workspace_id, channel, kind, message_count)
		 VALUES (?, ?, ?, 1)
		 ON CONFLICT(workspace_id, channel, kind) DO UPDATE SET
			message_count = message_count + 1`,
		entry.WorkspaceID,
		entry.Channel,
		strings.TrimSpace(entry.Kind),
	); err != nil {
		return fmt.Errorf("store: update network channel kind projection: %w", err)
	}
	return nil
}

func updateNetworkChannelParticipants(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	participants := make(map[string]struct{}, len(entry.Mentions)+2)
	for _, candidate := range append(
		[]string{entry.PeerFrom, entry.PeerTo},
		entry.Mentions...,
	) {
		peerID := strings.TrimSpace(candidate)
		if peerID != "" {
			participants[peerID] = struct{}{}
		}
	}
	for peerID := range participants {
		result, err := exec.ExecContext(
			ctx,
			`INSERT OR IGNORE INTO network_channel_participants (workspace_id, channel, peer_id)
			 VALUES (?, ?, ?)`,
			entry.WorkspaceID,
			entry.Channel,
			peerID,
		)
		if err != nil {
			return fmt.Errorf("store: insert network channel participant: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: inspect network channel participant insert: %w", err)
		}
		if rowsAffected == 0 {
			continue
		}
		if _, err := exec.ExecContext(
			ctx,
			`UPDATE network_channel_stats
			 SET historical_participant_count = historical_participant_count + 1
			 WHERE workspace_id = ? AND channel = ?`,
			entry.WorkspaceID,
			entry.Channel,
		); err != nil {
			return fmt.Errorf("store: increment network channel participant projection: %w", err)
		}
	}
	return nil
}

func networkProjectionMessageIsPublic(entry store.NetworkConversationMessage) bool {
	return strings.TrimSpace(entry.Kind) != store.NetworkKindGreet &&
		strings.TrimSpace(entry.PeerTo) == "" &&
		strings.TrimSpace(entry.DirectID) == "" &&
		strings.TrimSpace(entry.Surface) != store.NetworkSurfaceDirect
}
