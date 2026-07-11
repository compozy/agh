package globaldb

import (
	"context"
	"database/sql"

	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
)

func (g *GlobalDB) lookupNetworkConversationMessageCursor(
	ctx context.Context,
	ref store.NetworkConversationRef,
	messageID string,
	query store.NetworkConversationMessageQuery,
) (networkMessageCursor, error) {
	cursorQuery := query
	cursorQuery.BeforeMessageID = ""
	cursorQuery.AfterMessageID = ""
	where, args := networkConversationMessageFilterClauses(ref, cursorQuery)
	where = append([]string{"message_id = ?"}, where...)
	args = append([]any{strings.TrimSpace(messageID)}, args...)

	var cursor networkMessageCursor
	if err := g.db.QueryRowContext(
		ctx,
		store.AppendWhere(`SELECT sequence FROM network_timeline_log`, where),
		args...,
	).Scan(&cursor.Sequence); err != nil {
		return networkMessageCursor{}, fmt.Errorf("store: network conversation message cursor not found: %w", err)
	}
	return cursor, nil
}

func networkConversationMessageSelect() string {
	return `SELECT
		sequence,
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
		COALESCE((
			SELECT MAX(audit.size)
			FROM network_audit_log AS audit
			WHERE audit.workspace_id = network_timeline_log.workspace_id
				AND audit.message_id = network_timeline_log.message_id
		), 0),
		timestamp
	FROM network_timeline_log`
}

func networkConversationMessageFilterClauses(
	ref store.NetworkConversationRef,
	query store.NetworkConversationMessageQuery,
) ([]string, []any) {
	where := []string{
		globalDBNetworkConversationsWorkspaceIDValue,
		globalDBNetworkConversationsChannelValue,
		"surface = ?",
	}
	args := []any{ref.WorkspaceID, ref.Channel, ref.Surface}
	if ref.Surface == store.NetworkSurfaceThread {
		where = append(where, "thread_id = ?")
		args = append(args, ref.ThreadID)
	} else {
		where = append(where, "direct_id = ?")
		args = append(args, ref.DirectID)
	}
	if strings.TrimSpace(query.Kind) != "" {
		where = append(where, "kind = ?")
		args = append(args, strings.TrimSpace(query.Kind))
	}
	if strings.TrimSpace(query.WorkID) != "" {
		where = append(where, "work_id = ?")
		args = append(args, strings.TrimSpace(query.WorkID))
	}
	return where, args
}

func normalizeNetworkConversationRef(ref store.NetworkConversationRef) store.NetworkConversationRef {
	return store.NetworkConversationRef{
		WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
		Channel:     strings.TrimSpace(ref.Channel),
		Surface:     strings.TrimSpace(ref.Surface),
		ThreadID:    strings.TrimSpace(ref.ThreadID),
		DirectID:    strings.TrimSpace(ref.DirectID),
	}
}

func normalizeNetworkChannelRef(ref store.NetworkChannelRef) store.NetworkChannelRef {
	return store.NetworkChannelRef{
		WorkspaceID: strings.TrimSpace(ref.WorkspaceID),
		Channel:     strings.TrimSpace(ref.Channel),
	}
}

func scanNetworkThreadSummaries(rows *sql.Rows) ([]store.NetworkThreadSummary, error) {
	summaries := make([]store.NetworkThreadSummary, 0)
	for rows.Next() {
		summary, err := scanNetworkThreadSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network thread rows: %w", err)
	}
	return summaries, nil
}

func scanNetworkThreadSummary(scanner rowScanner) (store.NetworkThreadSummary, error) {
	var (
		summary     store.NetworkThreadSummary
		title       sql.NullString
		sessionID   sql.NullString
		lastPreview sql.NullString
		openedRaw   string
		activityRaw string
	)
	if err := scanner.Scan(
		&summary.WorkspaceID,
		&summary.Channel,
		&summary.ThreadID,
		&summary.RootMessageID,
		&title,
		&summary.OpenedByPeerID,
		&sessionID,
		&openedRaw,
		&summary.OpenedSequence,
		&activityRaw,
		&summary.LastActivitySequence,
		&summary.MessageCount,
		&summary.ParticipantCount,
		&summary.OpenWorkCount,
		&summary.DeliveredCount,
		&summary.PromptSizeBytes,
		&summary.EstimatedPromptTokens,
		&lastPreview,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkThreadSummary{}, fmt.Errorf(
				"%w: network thread: %w",
				store.ErrNetworkConversationNotFound,
				err,
			)
		}
		return store.NetworkThreadSummary{}, fmt.Errorf("store: scan network thread summary: %w", err)
	}
	if value := store.NullString(title); value != nil {
		summary.Title = *value
	}
	if value := store.NullString(sessionID); value != nil {
		summary.OpenedSessionID = *value
	}
	if value := store.NullString(lastPreview); value != nil {
		summary.LastMessagePreview = *value
	}
	openedAt, err := store.ParseTimestamp(openedRaw)
	if err != nil {
		return store.NetworkThreadSummary{}, fmt.Errorf("store: parse network thread opened_at: %w", err)
	}
	activityAt, err := store.ParseTimestamp(activityRaw)
	if err != nil {
		return store.NetworkThreadSummary{}, fmt.Errorf("store: parse network thread last_activity_at: %w", err)
	}
	summary.OpenedAt = openedAt
	summary.LastActivityAt = activityAt
	if err := summary.Validate(); err != nil {
		return store.NetworkThreadSummary{}, err
	}
	return summary, nil
}

func scanNetworkDirectRoomSummaries(rows *sql.Rows) ([]store.NetworkDirectRoomSummary, error) {
	summaries := make([]store.NetworkDirectRoomSummary, 0)
	for rows.Next() {
		summary, err := scanNetworkDirectRoomSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate network direct room rows: %w", err)
	}
	return summaries, nil
}

func scanNetworkDirectRoomSummary(scanner rowScanner) (store.NetworkDirectRoomSummary, error) {
	var (
		summary     store.NetworkDirectRoomSummary
		lastPreview sql.NullString
		openedRaw   string
		activityRaw string
	)
	if err := scanner.Scan(
		&summary.WorkspaceID,
		&summary.Channel,
		&summary.DirectID,
		&summary.PeerA,
		&summary.PeerB,
		&openedRaw,
		&summary.OpenedSequence,
		&activityRaw,
		&summary.LastActivitySequence,
		&summary.MessageCount,
		&summary.OpenWorkCount,
		&lastPreview,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkDirectRoomSummary{}, fmt.Errorf(
				"%w: network direct room: %w",
				store.ErrNetworkConversationNotFound,
				err,
			)
		}
		return store.NetworkDirectRoomSummary{}, fmt.Errorf("store: scan network direct room summary: %w", err)
	}
	if value := store.NullString(lastPreview); value != nil {
		summary.LastMessagePreview = *value
	}
	openedAt, err := store.ParseTimestamp(openedRaw)
	if err != nil {
		return store.NetworkDirectRoomSummary{}, fmt.Errorf("store: parse network direct room opened_at: %w", err)
	}
	activityAt, err := store.ParseTimestamp(activityRaw)
	if err != nil {
		return store.NetworkDirectRoomSummary{}, fmt.Errorf(
			"store: parse network direct room last_activity_at: %w",
			err,
		)
	}
	summary.OpenedAt = openedAt
	summary.LastActivityAt = activityAt
	if err := summary.Validate(); err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}
	return summary, nil
}

func scanNetworkWorkEntry(scanner rowScanner) (store.NetworkWorkEntry, error) {
	var (
		entry       store.NetworkWorkEntry
		threadID    sql.NullString
		directID    sql.NullString
		sessionID   sql.NullString
		targetPeer  sql.NullString
		openedRaw   string
		activityRaw string
		terminalRaw sql.NullString
	)
	if err := scanner.Scan(
		&entry.WorkID,
		&entry.WorkspaceID,
		&entry.Channel,
		&entry.Surface,
		&threadID,
		&directID,
		&entry.OpenedByPeerID,
		&sessionID,
		&targetPeer,
		&entry.State,
		&openedRaw,
		&activityRaw,
		&terminalRaw,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.NetworkWorkEntry{}, fmt.Errorf(
				"%w: network work: %w",
				store.ErrNetworkConversationNotFound,
				err,
			)
		}
		return store.NetworkWorkEntry{}, fmt.Errorf("store: scan network work: %w", err)
	}
	if value := store.NullString(threadID); value != nil {
		entry.ThreadID = *value
	}
	if value := store.NullString(directID); value != nil {
		entry.DirectID = *value
	}
	if value := store.NullString(sessionID); value != nil {
		entry.OpenedSessionID = *value
	}
	if value := store.NullString(targetPeer); value != nil {
		entry.TargetPeerID = *value
	}
	openedAt, err := store.ParseTimestamp(openedRaw)
	if err != nil {
		return store.NetworkWorkEntry{}, fmt.Errorf("store: parse network work opened_at: %w", err)
	}
	activityAt, err := store.ParseTimestamp(activityRaw)
	if err != nil {
		return store.NetworkWorkEntry{}, fmt.Errorf("store: parse network work last_activity_at: %w", err)
	}
	entry.OpenedAt = openedAt
	entry.LastActivityAt = activityAt
	if value := store.NullString(terminalRaw); value != nil {
		terminalAt, parseErr := store.ParseTimestamp(*value)
		if parseErr != nil {
			return store.NetworkWorkEntry{}, fmt.Errorf("store: parse network work terminal_at: %w", parseErr)
		}
		entry.TerminalAt = &terminalAt
	}
	if err := entry.Validate(); err != nil {
		return store.NetworkWorkEntry{}, err
	}
	return entry, nil
}

func normalizeRequiredNetworkField(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("store: %s is required", label)
	}
	return trimmed, nil
}

func validateNetworkWorkID(workID string) error {
	if strings.TrimSpace(workID) == "" {
		return fmt.Errorf("store: network work_id is required")
	}
	if len(workID) > 128 || strings.ContainsAny(workID, `/\`) || containsControlCharacterForGlobalDB(workID) {
		return fmt.Errorf("store: invalid network work_id %q", workID)
	}
	return nil
}

func containsControlCharacterForGlobalDB(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
