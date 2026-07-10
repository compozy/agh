package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

func (g *GlobalDB) normalizeDirectRoomEntry(
	entry store.NetworkDirectRoomEntry,
) (store.NetworkDirectRoomEntry, error) {
	now := g.now()
	directID, peerA, peerB, err := store.NetworkDirectRoomIdentity(
		entry.WorkspaceID,
		entry.Channel,
		entry.PeerA,
		entry.PeerB,
	)
	if err != nil {
		return store.NetworkDirectRoomEntry{}, err
	}
	if existing := strings.TrimSpace(entry.DirectID); existing != "" && existing != directID {
		return store.NetworkDirectRoomEntry{}, fmt.Errorf(
			"%w: direct_id=%q expected=%q",
			store.ErrNetworkDirectRoomCollision,
			existing,
			directID,
		)
	}
	normalized := store.NetworkDirectRoomEntry{
		WorkspaceID:    strings.TrimSpace(entry.WorkspaceID),
		Channel:        strings.TrimSpace(entry.Channel),
		DirectID:       directID,
		PeerA:          peerA,
		PeerB:          peerB,
		OpenedAt:       entry.OpenedAt,
		LastActivityAt: entry.LastActivityAt,
	}
	if normalized.OpenedAt.IsZero() {
		normalized.OpenedAt = now
	}
	if normalized.LastActivityAt.IsZero() {
		normalized.LastActivityAt = normalized.OpenedAt
	}
	if err := normalized.Validate(); err != nil {
		return store.NetworkDirectRoomEntry{}, err
	}
	return normalized, nil
}

func (g *GlobalDB) normalizeConversationMessage(
	entry store.NetworkConversationMessage,
) (store.NetworkConversationMessage, error) {
	normalized := store.NetworkConversationMessage{
		MessageID:   strings.TrimSpace(entry.MessageID),
		SessionID:   strings.TrimSpace(entry.SessionID),
		WorkspaceID: strings.TrimSpace(entry.WorkspaceID),
		Channel:     strings.TrimSpace(entry.Channel),
		Surface:     strings.TrimSpace(entry.Surface),
		ThreadID:    strings.TrimSpace(entry.ThreadID),
		DirectID:    strings.TrimSpace(entry.DirectID),
		Direction:   entry.Direction,
		PeerFrom:    strings.TrimSpace(entry.PeerFrom),
		PeerTo:      strings.TrimSpace(entry.PeerTo),
		Kind:        strings.TrimSpace(entry.Kind),
		WorkID:      strings.TrimSpace(entry.WorkID),
		ReplyTo:     strings.TrimSpace(entry.ReplyTo),
		TraceID:     strings.TrimSpace(entry.TraceID),
		CausationID: strings.TrimSpace(entry.CausationID),
		Intent:      strings.TrimSpace(entry.Intent),
		Text:        strings.TrimSpace(entry.Text),
		PreviewText: strings.TrimSpace(entry.PreviewText),
		Mentions:    append([]string(nil), entry.Mentions...),
		ExtJSON:     append(json.RawMessage(nil), entry.ExtJSON...),
		Body:        append(json.RawMessage(nil), entry.Body...),
		Timestamp:   entry.Timestamp,
	}
	mentions, err := store.NormalizeNetworkPeerIDs(normalized.Mentions, "network message mentions")
	if err != nil {
		return store.NetworkConversationMessage{}, err
	}
	normalized.Mentions = mentions
	if normalized.PreviewText == "" {
		normalized.PreviewText = normalized.Text
	}
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = g.now()
	}
	normalized.Timestamp = normalized.Timestamp.UTC()
	if err := normalized.Validate(); err != nil {
		return store.NetworkConversationMessage{}, fmt.Errorf("store: validate network conversation message: %w", err)
	}
	if strings.TrimSpace(normalized.SessionID) == "" {
		return store.NetworkConversationMessage{}, fmt.Errorf(
			"store: network conversation message session_id is required",
		)
	}
	return normalized, nil
}

func resolveDirectRoomWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkDirectRoomEntry,
) (store.NetworkDirectRoomSummary, bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO network_direct_rooms (
			workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at, message_count, open_work_count,
			last_message_preview
		) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, '')`,
		entry.WorkspaceID,
		entry.Channel,
		entry.DirectID,
		entry.PeerA,
		entry.PeerB,
		store.FormatTimestamp(entry.OpenedAt),
		store.FormatTimestamp(entry.LastActivityAt),
	)
	if err != nil {
		return store.NetworkDirectRoomSummary{}, false, fmt.Errorf("store: insert network direct room: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return store.NetworkDirectRoomSummary{}, false, fmt.Errorf("store: inspect network direct room insert: %w", err)
	}

	summary, err := getDirectRoomByPeerPairWithExecutor(
		ctx,
		exec,
		entry.WorkspaceID,
		entry.Channel,
		entry.PeerA,
		entry.PeerB,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkDirectRoomSummary{}, false, fmt.Errorf(
				"%w: direct_id=%q peer_a=%q peer_b=%q",
				store.ErrNetworkDirectRoomCollision,
				entry.DirectID,
				entry.PeerA,
				entry.PeerB,
			)
		}
		return store.NetworkDirectRoomSummary{}, false, err
	}
	if summary.DirectID != entry.DirectID {
		return store.NetworkDirectRoomSummary{}, false, fmt.Errorf(
			"%w: direct_id=%q expected=%q",
			store.ErrNetworkDirectRoomCollision,
			summary.DirectID,
			entry.DirectID,
		)
	}
	return summary, rowsAffected > 0, nil
}

func getDirectRoomByPeerPairWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	channel string,
	peerA string,
	peerB string,
) (store.NetworkDirectRoomSummary, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT
			workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at,
			message_count, open_work_count, last_message_preview
		FROM network_direct_rooms
		WHERE workspace_id = ? AND channel = ? AND peer_a = ? AND peer_b = ?`,
		workspaceID,
		channel,
		peerA,
		peerB,
	)
	return scanNetworkDirectRoomSummary(row)
}

func insertNetworkTimelineMessageWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`INSERT INTO network_timeline_log (
			message_id,
			session_id,
			workspace_id,
			channel,
			surface,
			thread_id,
			direct_id,
			direction,
			peer_from,
			peer_to,
			kind,
			work_id,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			mentions_json,
			ext_json,
			body_json,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, message_id) DO NOTHING`,
		entry.MessageID,
		store.NullableString(entry.SessionID),
		entry.WorkspaceID,
		entry.Channel,
		store.NullableString(entry.Surface),
		store.NullableString(entry.ThreadID),
		store.NullableString(entry.DirectID),
		entry.Direction,
		entry.PeerFrom,
		store.NullableString(entry.PeerTo),
		entry.Kind,
		store.NullableString(entry.WorkID),
		store.NullableString(entry.ReplyTo),
		store.NullableString(entry.TraceID),
		store.NullableString(entry.CausationID),
		store.NullableString(entry.Intent),
		store.NullableString(entry.Text),
		entry.PreviewText,
		networkMentionsJSONString(entry.Mentions),
		networkMessageExtJSONString(entry.ExtJSON),
		string(entry.Body),
		store.FormatTimestamp(entry.Timestamp),
	)
	if err != nil {
		return false, fmt.Errorf("store: insert network conversation message: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: inspect network conversation message insert: %w", err)
	}
	return rowsAffected > 0, nil
}

func lookupNetworkMessageTimestamp(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	messageID string,
) time.Time {
	var timestampRaw string
	if err := exec.QueryRowContext(
		ctx,
		`SELECT timestamp FROM network_timeline_log WHERE workspace_id = ? AND message_id = ?`,
		workspaceID,
		messageID,
	).Scan(&timestampRaw); err != nil {
		return time.Time{}
	}
	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return time.Time{}
	}
	return timestamp
}

func ensureNetworkConversationContainer(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (bool, error) {
	switch entry.Surface {
	case store.NetworkSurfaceThread:
		return ensureNetworkThreadWithExecutor(ctx, exec, entry)
	case store.NetworkSurfaceDirect:
		return ensureNetworkDirectRoomWithExecutor(ctx, exec, entry)
	default:
		return false, nil
	}
}

func ensureNetworkThreadWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (bool, error) {
	result, err := exec.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO network_threads (
			workspace_id, channel, thread_id, root_message_id, title, opened_by_peer_id, opened_session_id,
			opened_at, last_activity_at, message_count, participant_count, open_work_count, last_message_preview
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0, '')`,
		entry.WorkspaceID,
		entry.Channel,
		entry.ThreadID,
		entry.MessageID,
		entry.PreviewText,
		entry.PeerFrom,
		entry.SessionID,
		store.FormatTimestamp(entry.Timestamp),
		store.FormatTimestamp(entry.Timestamp),
	)
	if err != nil {
		return false, fmt.Errorf("store: insert network thread: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: inspect network thread insert: %w", err)
	}
	return rowsAffected > 0, nil
}

func ensureNetworkDirectRoomWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (bool, error) {
	if strings.TrimSpace(entry.PeerTo) == "" {
		return false, fmt.Errorf("store: network direct message peer_to is required")
	}
	directID, peerA, peerB, err := store.NetworkDirectRoomIdentity(
		entry.WorkspaceID,
		entry.Channel,
		entry.PeerFrom,
		entry.PeerTo,
	)
	if err != nil {
		return false, err
	}
	if entry.DirectID != directID {
		return false, fmt.Errorf(
			"%w: direct_id=%q expected=%q",
			store.ErrNetworkDirectRoomCollision,
			entry.DirectID,
			directID,
		)
	}
	_, opened, err := resolveDirectRoomWithExecutor(ctx, exec, store.NetworkDirectRoomEntry{
		WorkspaceID:    entry.WorkspaceID,
		Channel:        entry.Channel,
		DirectID:       directID,
		PeerA:          peerA,
		PeerB:          peerB,
		OpenedAt:       entry.Timestamp,
		LastActivityAt: entry.Timestamp,
	})
	return opened, err
}
