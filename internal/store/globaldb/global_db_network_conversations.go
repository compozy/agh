package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
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

			if normalized.Surface == store.NetworkSurfaceThread {
				if participantErr := upsertNetworkThreadParticipant(ctx, exec, normalized); participantErr != nil {
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
		opened_at, last_activity_at, message_count, participant_count, open_work_count, last_message_preview
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
			opened_at, last_activity_at, message_count, participant_count, open_work_count, last_message_preview
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
		ExtJSON:     append(json.RawMessage(nil), entry.ExtJSON...),
		Body:        append(json.RawMessage(nil), entry.Body...),
		Timestamp:   entry.Timestamp,
	}
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
			ext_json,
			body_json,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

func applyNetworkWorkMutation(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if strings.TrimSpace(entry.WorkID) == "" {
		return networkWorkMutation{}, nil
	}

	current, err := getNetworkWorkWithExecutor(ctx, exec, entry.WorkspaceID, entry.WorkID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return openNetworkWorkWithExecutor(ctx, exec, entry)
		}
		return networkWorkMutation{}, err
	}
	if !networkWorkMatchesMessage(current, entry) {
		return networkWorkMutation{}, fmt.Errorf("%w: work_id=%q", store.ErrNetworkWorkContainerMismatch, entry.WorkID)
	}
	if networkWorkStateIsTerminal(current.State) {
		return networkWorkMutation{}, fmt.Errorf("%w: work_id=%q", store.ErrNetworkWorkClosed, entry.WorkID)
	}

	next, transitioned, err := nextNetworkWorkState(current.State, entry)
	if err != nil {
		return networkWorkMutation{}, err
	}
	if !transitioned {
		return networkWorkMutation{state: current.State}, nil
	}

	var terminalAt any
	if networkWorkStateIsTerminal(next) {
		terminalAt = store.FormatTimestamp(entry.Timestamp)
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE network_work
		SET state = ?, last_activity_at = ?, terminal_at = ?
		WHERE workspace_id = ? AND work_id = ?`,
		next,
		store.FormatTimestamp(entry.Timestamp),
		terminalAt,
		entry.WorkspaceID,
		entry.WorkID,
	); err != nil {
		return networkWorkMutation{}, fmt.Errorf("store: update network work: %w", err)
	}
	return networkWorkMutation{transitioned: true, state: next}, nil
}

func openNetworkWorkWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) (networkWorkMutation, error) {
	if entry.Kind != store.NetworkKindSay && entry.Kind != store.NetworkKindCapability {
		return networkWorkMutation{}, fmt.Errorf(
			"store: network work %q does not exist: %w",
			entry.WorkID,
			sql.ErrNoRows,
		)
	}
	if strings.TrimSpace(entry.PeerTo) == "" {
		return networkWorkMutation{}, fmt.Errorf("store: network work target peer is required")
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO network_work (
			work_id, workspace_id, channel, surface, thread_id, direct_id, opened_by_peer_id, opened_session_id,
			target_peer_id, state, opened_at, last_activity_at, terminal_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		entry.WorkID,
		entry.WorkspaceID,
		entry.Channel,
		entry.Surface,
		store.NullableString(entry.ThreadID),
		store.NullableString(entry.DirectID),
		entry.PeerFrom,
		entry.SessionID,
		entry.PeerTo,
		store.NetworkWorkStateSubmitted,
		store.FormatTimestamp(entry.Timestamp),
		store.FormatTimestamp(entry.Timestamp),
	); err != nil {
		return networkWorkMutation{}, fmt.Errorf("store: insert network work: %w", err)
	}
	return networkWorkMutation{opened: true, state: store.NetworkWorkStateSubmitted}, nil
}

func getNetworkWorkWithExecutor(
	ctx context.Context,
	exec networkSQLExecutor,
	workspaceID string,
	workID string,
) (store.NetworkWorkEntry, error) {
	row := exec.QueryRowContext(
		ctx,
		`SELECT
			work_id, workspace_id, channel, surface, thread_id, direct_id, opened_by_peer_id, opened_session_id,
			target_peer_id, state, opened_at, last_activity_at, terminal_at
		FROM network_work
		WHERE workspace_id = ? AND work_id = ?`,
		workspaceID,
		workID,
	)
	return scanNetworkWorkEntry(row)
}

func networkWorkMatchesMessage(work store.NetworkWorkEntry, entry store.NetworkConversationMessage) bool {
	return work.WorkspaceID == entry.WorkspaceID &&
		work.Channel == entry.Channel &&
		work.Surface == entry.Surface &&
		strings.TrimSpace(work.ThreadID) == strings.TrimSpace(entry.ThreadID) &&
		strings.TrimSpace(work.DirectID) == strings.TrimSpace(entry.DirectID)
}

func nextNetworkWorkState(current string, entry store.NetworkConversationMessage) (string, bool, error) {
	switch entry.Kind {
	case store.NetworkKindSay, store.NetworkKindCapability:
		if current == store.NetworkWorkStateNeedsInput {
			return store.NetworkWorkStateWorking, true, nil
		}
		return current, false, nil
	case store.NetworkKindReceipt:
		return nextNetworkWorkStateFromReceipt(current, entry.Body)
	case store.NetworkKindTrace:
		return nextNetworkWorkStateFromTrace(current, entry.Body)
	default:
		return current, false, nil
	}
}

func nextNetworkWorkStateFromReceipt(current string, body json.RawMessage) (string, bool, error) {
	var receipt struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return "", false, fmt.Errorf("store: decode network receipt body: %w", err)
	}
	switch strings.TrimSpace(receipt.Status) {
	case "accepted", "duplicate", "expired", "unsupported":
		return current, false, nil
	case globalDBNetworkConversationsRejectedKey:
		return store.NetworkWorkStateFailed, true, nil
	case "canceled":
		return store.NetworkWorkStateCanceled, true, nil
	default:
		return "", false, fmt.Errorf("store: unsupported network receipt status %q", receipt.Status)
	}
}

func nextNetworkWorkStateFromTrace(current string, body json.RawMessage) (string, bool, error) {
	var trace struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(body, &trace); err != nil {
		return "", false, fmt.Errorf("store: decode network trace body: %w", err)
	}
	next := strings.TrimSpace(trace.State)
	if !canAdvanceNetworkWorkState(current, next) {
		return "", false, fmt.Errorf("store: invalid network work transition %s -> %s", current, next)
	}
	return next, true, nil
}

func canAdvanceNetworkWorkState(current string, next string) bool {
	if networkWorkStateIsTerminal(current) {
		return false
	}
	switch current {
	case store.NetworkWorkStateSubmitted, store.NetworkWorkStateWorking, store.NetworkWorkStateNeedsInput:
	default:
		return false
	}
	switch next {
	case store.NetworkWorkStateWorking,
		store.NetworkWorkStateNeedsInput,
		store.NetworkWorkStateCompleted,
		store.NetworkWorkStateFailed,
		store.NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

func networkWorkStateIsTerminal(state string) bool {
	switch state {
	case store.NetworkWorkStateCompleted, store.NetworkWorkStateFailed, store.NetworkWorkStateCanceled:
		return true
	default:
		return false
	}
}

func upsertNetworkThreadParticipant(
	ctx context.Context,
	exec networkSQLExecutor,
	entry store.NetworkConversationMessage,
) error {
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
		entry.PeerFrom,
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
		`SELECT COUNT(DISTINCT peer_from)
		FROM network_timeline_log
		WHERE workspace_id = ? AND channel = ? AND surface = 'thread' AND thread_id = ?`,
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
		SET last_activity_at = ?, message_count = ?, participant_count = ?, open_work_count = ?,
			last_message_preview = ?
		WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
		latest.timestamp,
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
		SET last_activity_at = ?, message_count = ?, open_work_count = ?, last_message_preview = ?
		WHERE workspace_id = ? AND channel = ? AND direct_id = ?`,
		latest.timestamp,
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
		`SELECT timestamp, preview_text
			FROM network_timeline_log
			WHERE workspace_id = ? AND channel = ? AND surface = ? AND %s = ?
			ORDER BY timestamp DESC, message_id DESC
			LIMIT 1`,
		column,
	)
	if err := exec.QueryRowContext(ctx, query, workspaceID, channel, surface, containerID).
		Scan(&latest.timestamp, &latest.preview); err != nil {
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

func (g *GlobalDB) lookupNetworkThreadCursor(
	ctx context.Context,
	ref store.NetworkChannelRef,
	threadID string,
) (networkThreadCursor, error) {
	row := g.db.QueryRowContext(
		ctx,
		`SELECT thread_id, last_activity_at
			FROM network_threads
			WHERE workspace_id = ? AND channel = ? AND thread_id = ?`,
		ref.WorkspaceID,
		ref.Channel,
		strings.TrimSpace(threadID),
	)
	var (
		cursor networkThreadCursor
		raw    string
	)
	if err := row.Scan(&cursor.ThreadID, &raw); err != nil {
		return networkThreadCursor{}, fmt.Errorf("store: lookup network thread cursor: %w", err)
	}
	timestamp, err := store.ParseTimestamp(raw)
	if err != nil {
		return networkThreadCursor{}, fmt.Errorf("store: parse network thread cursor: %w", err)
	}
	cursor.LastActivityAt = timestamp
	return cursor, nil
}

func (g *GlobalDB) lookupNetworkDirectRoomCursor(
	ctx context.Context,
	ref store.NetworkChannelRef,
	directID string,
	peerID string,
) (networkDirectRoomCursor, error) {
	where := []string{
		globalDBNetworkConversationsWorkspaceIDValue,
		globalDBNetworkConversationsChannelValue,
		"direct_id = ?",
	}
	args := []any{ref.WorkspaceID, ref.Channel, strings.TrimSpace(directID)}
	if trimmedPeer := strings.TrimSpace(peerID); trimmedPeer != "" {
		where = append(where, "(peer_a = ? OR peer_b = ?)")
		args = append(args, trimmedPeer, trimmedPeer)
	}
	row := g.db.QueryRowContext(
		ctx,
		store.AppendWhere(
			`SELECT direct_id, last_activity_at FROM network_direct_rooms`,
			where,
		),
		args...,
	)
	var (
		cursor networkDirectRoomCursor
		raw    string
	)
	if err := row.Scan(&cursor.DirectID, &raw); err != nil {
		return networkDirectRoomCursor{}, fmt.Errorf("store: lookup network direct room cursor: %w", err)
	}
	timestamp, err := store.ParseTimestamp(raw)
	if err != nil {
		return networkDirectRoomCursor{}, fmt.Errorf("store: parse network direct room cursor: %w", err)
	}
	cursor.LastActivityAt = timestamp
	return cursor, nil
}

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
		store.AppendWhere(`SELECT message_id, timestamp FROM network_timeline_log`, where),
		args...,
	).Scan(&cursor.MessageID, &cursor.Timestamp); err != nil {
		return networkMessageCursor{}, fmt.Errorf("store: network conversation message cursor not found: %w", err)
	}
	return cursor, nil
}

func networkConversationMessageSelect() string {
	return `SELECT
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
		ext_json,
		body_json,
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
		&activityRaw,
		&summary.MessageCount,
		&summary.ParticipantCount,
		&summary.OpenWorkCount,
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
		&activityRaw,
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
