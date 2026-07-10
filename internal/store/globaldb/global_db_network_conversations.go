package globaldb

import (
	"context"

	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/store"
)

const (
	globalDBNetworkConversationsWorkspaceIDValue = "workspace_id = ?"
)

const (
	globalDBNetworkConversationsChannelValue = "channel = ?"
	globalDBNetworkConversationsDirectIDKey  = "direct_id"
	globalDBNetworkConversationsRejectedKey  = "rejected"
	globalDBNetworkConversationsThreadIDKey  = "thread_id"
)

type networkThreadCursor struct {
	ThreadID       string
	LastActivityAt time.Time
}

type networkDirectRoomCursor struct {
	DirectID       string
	LastActivityAt time.Time
}

type networkWorkMutation struct {
	opened       bool
	transitioned bool
	state        string
}

// ResolveDirectRoom inserts or returns the deterministic two-party room.
func (g *GlobalDB) ResolveDirectRoom(
	ctx context.Context,
	entry store.NetworkDirectRoomEntry,
) (summary store.NetworkDirectRoomSummary, err error) {
	if err := g.checkReady(ctx, "resolve network direct room"); err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}
	normalized, err := g.normalizeDirectRoomEntry(entry)
	if err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}

	if err := g.withNetworkImmediateTransaction(
		ctx,
		"resolve network direct room",
		func(exec networkSQLExecutor) error {
			resolved, _, resolveErr := resolveDirectRoomWithExecutor(ctx, exec, normalized)
			if resolveErr != nil {
				return resolveErr
			}
			summary = resolved
			return nil
		},
	); err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}
	return summary, nil
}

// WriteConversationMessage persists one message and its derived state atomically.
func (g *GlobalDB) WriteConversationMessage(
	ctx context.Context,
	entry store.NetworkConversationMessage,
) (result store.NetworkConversationWriteResult, err error) {
	if err := g.checkReady(ctx, "write network conversation message"); err != nil {
		return store.NetworkConversationWriteResult{}, err
	}
	normalized, err := g.normalizeConversationMessage(entry)
	if err != nil {
		return store.NetworkConversationWriteResult{}, err
	}
	result.MessageID = normalized.MessageID

	if err := g.withNetworkImmediateTransaction(
		ctx,
		"write network conversation message",
		func(exec networkSQLExecutor) error {
			inserted, insertErr := insertNetworkTimelineMessageWithExecutor(ctx, exec, normalized)
			if insertErr != nil {
				return insertErr
			}
			if !inserted {
				result.Duplicate = true
				result.LastActivityAt = lookupNetworkMessageTimestamp(
					ctx,
					exec,
					normalized.WorkspaceID,
					normalized.MessageID,
				)
				return nil
			}

			opened, ensureErr := ensureNetworkConversationContainer(ctx, exec, normalized)
			if ensureErr != nil {
				return ensureErr
			}
			result.ConversationOpened = opened

			work, workErr := applyNetworkWorkMutation(ctx, exec, normalized)
			if workErr != nil {
				return workErr
			}
			result.WorkOpened = work.opened
			result.WorkTransitioned = work.transitioned
			result.WorkState = work.state
			if projectionErr := persistNetworkTimelineWorkProjection(
				ctx,
				exec,
				normalized,
				work,
			); projectionErr != nil {
				return projectionErr
			}

			if normalized.Surface == store.NetworkSurfaceThread {
				if participantErr := upsertNetworkThreadParticipantsForMessage(
					ctx,
					exec,
					normalized,
				); participantErr != nil {
					return participantErr
				}
			}
			if summaryErr := refreshNetworkConversationSummary(ctx, exec, normalized); summaryErr != nil {
				return summaryErr
			}
			auditEntry := auditEntryForConversationMessage(normalized)
			if auditErr := insertNetworkAuditWithExecutor(ctx, exec, auditEntry); auditErr != nil {
				return auditErr
			}

			result.LastActivityAt = normalized.Timestamp
			return nil
		},
	); err != nil {
		return store.NetworkConversationWriteResult{}, err
	}
	return result, nil
}

func upsertNetworkThreadParticipantsForMessage(
	ctx context.Context,
	exec networkSQLExecutor,
	message store.NetworkConversationMessage,
) error {
	if err := upsertNetworkThreadParticipant(ctx, exec, message); err != nil {
		return err
	}
	if err := upsertNetworkThreadTargetParticipant(ctx, exec, message); err != nil {
		return err
	}
	for _, mention := range message.Mentions {
		if err := upsertNetworkThreadParticipantPeer(ctx, exec, message, mention); err != nil {
			return err
		}
	}
	return nil
}

// ListThreads returns public-thread summaries for one channel.
func (g *GlobalDB) ListThreads(
	ctx context.Context,
	ref store.NetworkChannelRef,
	query store.NetworkThreadQuery,
) (summaries []store.NetworkThreadSummary, err error) {
	if err := g.checkReady(ctx, "list network threads"); err != nil {
		return nil, err
	}
	normalizedRef := normalizeNetworkChannelRef(ref)
	if err := normalizedRef.Validate(); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network thread query: %w", err)
	}

	sqlQuery := `SELECT
		workspace_id, channel, thread_id, root_message_id, title, opened_by_peer_id, opened_session_id,
		opened_at, last_activity_at, message_count, participant_count, open_work_count,
		COALESCE((
			SELECT SUM(stats.delivered_count)
			FROM network_thread_peer_token_stats AS stats
			WHERE stats.workspace_id = network_threads.workspace_id
				AND stats.channel = network_threads.channel
				AND stats.thread_id = network_threads.thread_id
		), 0),
		COALESCE((
			SELECT SUM(stats.prompt_size_bytes)
			FROM network_thread_peer_token_stats AS stats
			WHERE stats.workspace_id = network_threads.workspace_id
				AND stats.channel = network_threads.channel
				AND stats.thread_id = network_threads.thread_id
		), 0),
		COALESCE((
			SELECT SUM(stats.estimated_prompt_tokens)
			FROM network_thread_peer_token_stats AS stats
			WHERE stats.workspace_id = network_threads.workspace_id
				AND stats.channel = network_threads.channel
				AND stats.thread_id = network_threads.thread_id
		), 0),
		last_message_preview
	FROM network_threads`
	where := []string{globalDBNetworkConversationsWorkspaceIDValue, globalDBNetworkConversationsChannelValue}
	args := []any{normalizedRef.WorkspaceID, normalizedRef.Channel}
	if after := strings.TrimSpace(query.After); after != "" {
		cursor, cursorErr := g.lookupNetworkThreadCursor(ctx, normalizedRef, after)
		if cursorErr != nil {
			return nil, cursorErr
		}
		cursorAt := store.FormatTimestamp(cursor.LastActivityAt)
		where = append(where, "(last_activity_at < ? OR (last_activity_at = ? AND thread_id > ?))")
		args = append(args, cursorAt, cursorAt, cursor.ThreadID)
	}
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY last_activity_at DESC, thread_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network threads: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network thread rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	return scanNetworkThreadSummaries(rows)
}

// GetThread returns one public-thread summary.
func (g *GlobalDB) GetThread(
	ctx context.Context,
	channelRef store.NetworkChannelRef,
	threadID string,
) (store.NetworkThreadSummary, error) {
	if err := g.checkReady(ctx, "get network thread"); err != nil {
		return store.NetworkThreadSummary{}, err
	}
	ref := store.NetworkConversationRef{
		WorkspaceID: strings.TrimSpace(channelRef.WorkspaceID),
		Channel:     strings.TrimSpace(channelRef.Channel),
		Surface:     store.NetworkSurfaceThread,
		ThreadID:    strings.TrimSpace(threadID),
	}
	if err := ref.Validate(); err != nil {
		return store.NetworkThreadSummary{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			workspace_id, channel, thread_id, root_message_id, title, opened_by_peer_id, opened_session_id,
			opened_at, last_activity_at, message_count, participant_count, open_work_count,
			COALESCE((
				SELECT SUM(stats.delivered_count)
				FROM network_thread_peer_token_stats AS stats
				WHERE stats.workspace_id = network_threads.workspace_id
					AND stats.channel = network_threads.channel
					AND stats.thread_id = network_threads.thread_id
			), 0),
			COALESCE((
				SELECT SUM(stats.prompt_size_bytes)
				FROM network_thread_peer_token_stats AS stats
				WHERE stats.workspace_id = network_threads.workspace_id
					AND stats.channel = network_threads.channel
					AND stats.thread_id = network_threads.thread_id
			), 0),
			COALESCE((
				SELECT SUM(stats.estimated_prompt_tokens)
				FROM network_thread_peer_token_stats AS stats
				WHERE stats.workspace_id = network_threads.workspace_id
					AND stats.channel = network_threads.channel
					AND stats.thread_id = network_threads.thread_id
			), 0),
			last_message_preview
		FROM network_threads
		WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
		ref.WorkspaceID,
		ref.Channel,
		ref.ThreadID,
	)
	return scanNetworkThreadSummary(row)
}

// ListDirectRooms returns direct-room summaries for one channel.
func (g *GlobalDB) ListDirectRooms(
	ctx context.Context,
	ref store.NetworkChannelRef,
	query store.NetworkDirectRoomQuery,
) (summaries []store.NetworkDirectRoomSummary, err error) {
	if err := g.checkReady(ctx, "list network direct rooms"); err != nil {
		return nil, err
	}
	normalizedRef := normalizeNetworkChannelRef(ref)
	if err := normalizedRef.Validate(); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network direct room query: %w", err)
	}

	sqlQuery := `SELECT
		workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at,
		message_count, open_work_count, last_message_preview
	FROM network_direct_rooms`
	where := []string{globalDBNetworkConversationsWorkspaceIDValue, globalDBNetworkConversationsChannelValue}
	args := []any{normalizedRef.WorkspaceID, normalizedRef.Channel}
	if peerID := strings.TrimSpace(query.PeerID); peerID != "" {
		where = append(where, "(peer_a = ? OR peer_b = ?)")
		args = append(args, peerID, peerID)
	}
	if after := strings.TrimSpace(query.After); after != "" {
		cursor, cursorErr := g.lookupNetworkDirectRoomCursor(ctx, normalizedRef, after, query.PeerID)
		if cursorErr != nil {
			return nil, cursorErr
		}
		cursorAt := store.FormatTimestamp(cursor.LastActivityAt)
		where = append(where, "(last_activity_at < ? OR (last_activity_at = ? AND direct_id > ?))")
		args = append(args, cursorAt, cursorAt, cursor.DirectID)
	}
	sqlQuery = store.AppendWhere(sqlQuery, where)
	sqlQuery += " ORDER BY last_activity_at DESC, direct_id ASC"
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network direct rooms: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network direct room rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	return scanNetworkDirectRoomSummaries(rows)
}

// GetDirectRoom returns one direct-room summary.
func (g *GlobalDB) GetDirectRoom(
	ctx context.Context,
	channelRef store.NetworkChannelRef,
	directID string,
) (store.NetworkDirectRoomSummary, error) {
	if err := g.checkReady(ctx, "get network direct room"); err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}
	ref := store.NetworkConversationRef{
		WorkspaceID: strings.TrimSpace(channelRef.WorkspaceID),
		Channel:     strings.TrimSpace(channelRef.Channel),
		Surface:     store.NetworkSurfaceDirect,
		DirectID:    strings.TrimSpace(directID),
	}
	if err := ref.Validate(); err != nil {
		return store.NetworkDirectRoomSummary{}, err
	}

	row := g.db.QueryRowContext(
		ctx,
		`SELECT
			workspace_id, channel, direct_id, peer_a, peer_b, opened_at, last_activity_at,
			message_count, open_work_count, last_message_preview
		FROM network_direct_rooms
		WHERE workspace_id = ? AND channel = ? AND direct_id = ?`,
		ref.WorkspaceID,
		ref.Channel,
		ref.DirectID,
	)
	return scanNetworkDirectRoomSummary(row)
}

// ListConversationMessages returns messages isolated to one conversation container.
func (g *GlobalDB) ListConversationMessages(
	ctx context.Context,
	ref store.NetworkConversationRef,
	query store.NetworkConversationMessageQuery,
) (entries []store.NetworkConversationMessage, err error) {
	if err := g.checkReady(ctx, "list network conversation messages"); err != nil {
		return nil, err
	}
	normalizedRef := normalizeNetworkConversationRef(ref)
	if err := normalizedRef.Validate(); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("store: validate network conversation message query: %w", err)
	}

	sqlQuery := networkConversationMessageSelect()
	where, args := networkConversationMessageFilterClauses(normalizedRef, query)
	reverseResults := false
	switch {
	case strings.TrimSpace(query.BeforeMessageID) != "":
		cursor, cursorErr := g.lookupNetworkConversationMessageCursor(ctx, normalizedRef, query.BeforeMessageID, query)
		if cursorErr != nil {
			return nil, cursorErr
		}
		where = append(where, "(timestamp < ? OR (timestamp = ? AND message_id < ?))")
		args = append(args, cursor.Timestamp, cursor.Timestamp, cursor.MessageID)
		reverseResults = true
	case strings.TrimSpace(query.AfterMessageID) != "":
		cursor, cursorErr := g.lookupNetworkConversationMessageCursor(ctx, normalizedRef, query.AfterMessageID, query)
		if cursorErr != nil {
			return nil, cursorErr
		}
		where = append(where, "(timestamp > ? OR (timestamp = ? AND message_id > ?))")
		args = append(args, cursor.Timestamp, cursor.Timestamp, cursor.MessageID)
	}
	sqlQuery = store.AppendWhere(sqlQuery, where)
	if reverseResults {
		sqlQuery += " ORDER BY timestamp DESC, message_id DESC"
	} else {
		sqlQuery += " ORDER BY timestamp ASC, message_id ASC"
	}
	sqlQuery, args = store.AppendLimit(sqlQuery, args, query.Limit)

	rows, err := g.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query network conversation messages: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("store: close network conversation message rows: %w", closeErr)
			if err != nil {
				err = errors.Join(err, closeErr)
				return
			}
			err = closeErr
		}
	}()

	entries, err = loadNetworkMessageEntries(rows)
	if err != nil {
		return nil, err
	}
	if reverseResults {
		reverseNetworkMessages(entries)
	}
	return entries, nil
}

// GetWork returns one network work row by workspace_id and work_id.
func (g *GlobalDB) GetWork(ctx context.Context, workspaceID string, workID string) (store.NetworkWorkEntry, error) {
	if err := g.checkReady(ctx, "get network work"); err != nil {
		return store.NetworkWorkEntry{}, err
	}
	trimmedWorkspaceID, err := normalizeRequiredNetworkField(workspaceID, "network work workspace_id")
	if err != nil {
		return store.NetworkWorkEntry{}, err
	}
	trimmed := strings.TrimSpace(workID)
	if err := validateNetworkWorkID(trimmed); err != nil {
		return store.NetworkWorkEntry{}, err
	}
	return getNetworkWorkWithExecutor(ctx, g.db, trimmedWorkspaceID, trimmed)
}
