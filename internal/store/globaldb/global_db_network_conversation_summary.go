package globaldb

import (
	"context"

	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func upsertNetworkThreadParticipant(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	return upsertNetworkThreadParticipantPeer(ctx, exec, entry, entry.PeerFrom)
}

func upsertNetworkThreadTargetParticipant(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	return upsertNetworkThreadParticipantPeer(ctx, exec, entry, entry.PeerTo)
}

func upsertNetworkThreadParticipantPeer(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
	peerID string,
) error {
	trimmedPeerID := strings.TrimSpace(peerID)
	if trimmedPeerID == "" {
		return nil
	}
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO network_thread_participants (
			workspace_id, channel, thread_id, peer_id, first_message_id, first_seen_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, channel, thread_id, peer_id) DO UPDATE SET
			last_seen_at = excluded.last_seen_at`,
		entry.WorkspaceID,
		entry.Channel,
		entry.ThreadID,
		trimmedPeerID,
		entry.MessageID,
		store.FormatTimestamp(entry.Timestamp),
		store.FormatTimestamp(entry.Timestamp),
	)
	if err != nil {
		return fmt.Errorf("store: upsert network thread participant: %w", err)
	}
	return nil
}

func refreshNetworkConversationSummary(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	switch entry.Surface {
	case store.NetworkSurfaceThread:
		return refreshNetworkThreadSummary(ctx, exec, entry)
	case store.NetworkSurfaceDirect:
		return refreshNetworkDirectRoomSummary(ctx, exec, entry)
	default:
		return nil
	}
}

func refreshNetworkThreadSummary(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	latest, err := latestNetworkConversationMessage(
		ctx,
		exec,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
	)
	if err != nil {
		return err
	}
	var messageCount int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM network_timeline_log
		WHERE workspace_id = ? AND channel = ? AND surface = 'thread' AND thread_id = ?`,
		entry.WorkspaceID,
		entry.Channel,
		entry.ThreadID,
	).Scan(&messageCount); err != nil {
		return fmt.Errorf("store: count network thread messages: %w", err)
	}
	var participantCount int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM network_thread_participants
		WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
		entry.WorkspaceID,
		entry.Channel,
		entry.ThreadID,
	).Scan(&participantCount); err != nil {
		return fmt.Errorf("store: count network thread participants: %w", err)
	}
	openWorkCount, err := countOpenNetworkWork(
		ctx,
		exec,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		entry.ThreadID,
		"",
	)
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_threads
		SET last_activity_at = ?, last_activity_sequence = ?,
			message_count = ?, participant_count = ?, open_work_count = ?,
			last_message_preview = ?
		WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
		latest.timestamp,
		latest.sequence,
		messageCount,
		participantCount,
		openWorkCount,
		latest.preview,
		entry.WorkspaceID,
		entry.Channel,
		entry.ThreadID,
	); err != nil {
		return fmt.Errorf("store: update network thread summary: %w", err)
	}
	return nil
}

func refreshNetworkDirectRoomSummary(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
	latest, err := latestNetworkConversationMessage(
		ctx,
		exec,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		entry.DirectID,
	)
	if err != nil {
		return err
	}
	var messageCount int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		FROM network_timeline_log
		WHERE workspace_id = ? AND channel = ? AND surface = 'direct' AND direct_id = ?`,
		entry.WorkspaceID,
		entry.Channel,
		entry.DirectID,
	).Scan(&messageCount); err != nil {
		return fmt.Errorf("store: count network direct messages: %w", err)
	}
	openWorkCount, err := countOpenNetworkWork(
		ctx,
		exec,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		"",
		entry.DirectID,
	)
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_direct_rooms
		SET last_activity_at = ?, last_activity_sequence = ?,
			message_count = ?, open_work_count = ?, last_message_preview = ?
		WHERE workspace_id = ? AND channel = ? AND direct_id = ?`,
		latest.timestamp,
		latest.sequence,
		messageCount,
		openWorkCount,
		latest.preview,
		entry.WorkspaceID,
		entry.Channel,
		entry.DirectID,
	); err != nil {
		return fmt.Errorf("store: update network direct room summary: %w", err)
	}
	return nil
}

type latestNetworkMessage struct {
	sequence  int64
	timestamp string
	preview   string
}

func latestNetworkConversationMessage(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	channel string,
	surface string,
	containerID string,
) (latestNetworkMessage, error) {
	column := globalDBNetworkConversationsThreadIDKey
	if surface == store.NetworkSurfaceDirect {
		column = globalDBNetworkConversationsDirectIDKey
	}
	var latest latestNetworkMessage
	query := fmt.Sprintf(
		`SELECT sequence, timestamp, preview_text
			FROM network_timeline_log
			WHERE workspace_id = ? AND channel = ? AND surface = ? AND %s = ?
			ORDER BY sequence DESC
			LIMIT 1`,
		column,
	)
	if err := exec.QueryRowContext(ctx, query, workspaceID, channel, surface, containerID).
		Scan(&latest.sequence, &latest.timestamp, &latest.preview); err != nil {
		return latestNetworkMessage{}, fmt.Errorf("store: lookup latest network conversation message: %w", err)
	}
	return latest, nil
}

func countOpenNetworkWork(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	channel string,
	surface string,
	threadID string,
	directID string,
) (int, error) {
	where := []string{
		globalDBNetworkConversationsWorkspaceIDValue,
		globalDBNetworkConversationsChannelValue,
		"surface = ?",
		"state NOT IN (?, ?, ?)",
	}
	args := []any{
		workspaceID,
		channel,
		surface,
		store.NetworkWorkStateCompleted,
		store.NetworkWorkStateFailed,
		store.NetworkWorkStateCanceled,
	}
	if surface == store.NetworkSurfaceThread {
		where = append(where, "thread_id = ?")
		args = append(args, threadID)
	} else {
		where = append(where, "direct_id = ?")
		args = append(args, directID)
	}

	var count int
	if err := exec.QueryRowContext(
		ctx,
		store.AppendWhere(`SELECT COUNT(*) FROM network_work`, where),
		args...,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("store: count open network work: %w", err)
	}
	return count, nil
}

func auditEntryForConversationMessage(entry store.NetworkConversationMessage) store.NetworkAuditEntry {
	return store.NetworkAuditEntry{
		ID:          store.NewID("naud"),
		SessionID:   entry.SessionID,
		WorkspaceID: entry.WorkspaceID,
		Direction:   entry.Direction,
		Kind:        entry.Kind,
		Channel:     entry.Channel,
		Surface:     entry.Surface,
		ThreadID:    entry.ThreadID,
		DirectID:    entry.DirectID,
		WorkID:      entry.WorkID,
		PeerFrom:    entry.PeerFrom,
		PeerTo:      entry.PeerTo,
		MessageID:   entry.MessageID,
		Size:        len(entry.Body),
		Timestamp:   entry.Timestamp,
	}
}
