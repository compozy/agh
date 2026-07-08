package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

type networkWatchEventRow struct {
	seq     int64
	message store.NetworkConversationMessage
	at      time.Time
}

type networkWorkOutcome struct {
	opened       bool
	transitioned bool
	state        string
}

func (g *GlobalDB) readNetworkWatchEventsCursor(
	ctx context.Context,
	query normalizedWatchEventsQuery,
) (int64, error) {
	rows, ok, err := g.readNetworkWatchEventRows(ctx, query, 0)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	defer rows.Close()

	var cursor int64
	for rows.Next() {
		row, scanErr := scanNetworkWatchEventRow(rows)
		if scanErr != nil {
			return 0, joinRowsCloseError(rows, scanErr, "network watch-events cursor query")
		}
		events, eventErr := g.networkWatchEventsForRow(ctx, row, query.kinds)
		if eventErr != nil {
			return 0, joinRowsCloseError(rows, eventErr, "network watch-events cursor query")
		}
		if len(events) > 0 && row.seq > cursor {
			cursor = row.seq
		}
	}
	if err := rows.Err(); err != nil {
		return 0, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate network watch-events cursor: %w", err),
			"network watch-events cursor query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "network watch-events cursor query"); err != nil {
		return 0, err
	}
	return cursor, nil
}

func (g *GlobalDB) readNetworkWatchEvents(
	ctx context.Context,
	query normalizedWatchEventsQuery,
) ([]looppkg.WatchEvent, error) {
	rows, ok, err := g.readNetworkWatchEventRows(ctx, query, query.streams[looppkg.WatchEventsNetworkStream])
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	defer rows.Close()

	events := make([]looppkg.WatchEvent, 0)
	for rows.Next() {
		row, scanErr := scanNetworkWatchEventRow(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "network watch-events query")
		}
		rowEvents, eventErr := g.networkWatchEventsForRow(ctx, row, query.kinds)
		if eventErr != nil {
			return nil, joinRowsCloseError(rows, eventErr, "network watch-events query")
		}
		if len(rowEvents) == 0 {
			continue
		}
		events = append(events, rowEvents...)
		if len(events) >= query.limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate network watch-events: %w", err),
			"network watch-events query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "network watch-events query"); err != nil {
		return nil, err
	}
	return events, nil
}

func (g *GlobalDB) readNetworkWatchEventRows(
	ctx context.Context,
	query normalizedWatchEventsQuery,
	after int64,
) (*sql.Rows, bool, error) {
	predicate, ok := networkWatchEventsCandidatePredicate(query.kinds)
	if !ok {
		return nil, false, nil
	}
	// #nosec G202 -- predicate is assembled from fixed SQL fragments, values are parameterized.
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			ntl.rowid,
			ntl.message_id,
			COALESCE(ntl.session_id, ''),
			ntl.workspace_id,
			ntl.channel,
			COALESCE(ntl.surface, ''),
			COALESCE(ntl.thread_id, ''),
			COALESCE(ntl.direct_id, ''),
			ntl.direction,
			ntl.peer_from,
			COALESCE(ntl.peer_to, ''),
			ntl.kind,
			COALESCE(ntl.work_id, ''),
			COALESCE(ntl.reply_to, ''),
			COALESCE(ntl.trace_id, ''),
			COALESCE(ntl.causation_id, ''),
			COALESCE(ntl.intent, ''),
			COALESCE(ntl.text, ''),
			ntl.body_json,
			ntl.timestamp
		   FROM network_timeline_log ntl
		  WHERE ntl.workspace_id = ?
		    AND ntl.rowid > ?
		    AND (`+predicate+`)
		  ORDER BY ntl.rowid ASC`,
		query.workspaceID,
		after,
	)
	if err != nil {
		return nil, false, fmt.Errorf("store: read network watch-events: %w", err)
	}
	return rows, true, nil
}

func scanNetworkWatchEventRow(row rowScanner) (networkWatchEventRow, error) {
	var (
		event networkWatchEventRow
		body  string
		atRaw string
	)
	if err := row.Scan(
		&event.seq,
		&event.message.MessageID,
		&event.message.SessionID,
		&event.message.WorkspaceID,
		&event.message.Channel,
		&event.message.Surface,
		&event.message.ThreadID,
		&event.message.DirectID,
		&event.message.Direction,
		&event.message.PeerFrom,
		&event.message.PeerTo,
		&event.message.Kind,
		&event.message.WorkID,
		&event.message.ReplyTo,
		&event.message.TraceID,
		&event.message.CausationID,
		&event.message.Intent,
		&event.message.Text,
		&body,
		&atRaw,
	); err != nil {
		return networkWatchEventRow{}, fmt.Errorf("store: scan network watch-event: %w", err)
	}
	at, err := store.ParseTimestamp(atRaw)
	if err != nil {
		return networkWatchEventRow{}, fmt.Errorf("store: parse network watch-event timestamp: %w", err)
	}
	event.message.Body = json.RawMessage(body)
	event.message.Timestamp = at
	event.at = at
	return event, nil
}

func networkWatchEventsCandidatePredicate(kinds []string) (string, bool) {
	kindSet := stringSet(kinds)
	if _, ok := kindSet[string(hookspkg.HookNetworkMessagePersisted)]; ok {
		return "1 = 1", true
	}
	predicates := make([]string, 0, 3)
	if _, ok := kindSet[string(hookspkg.HookNetworkThreadOpened)]; ok {
		predicates = append(predicates, "ntl.surface = 'thread'")
	}
	if _, ok := kindSet[string(hookspkg.HookNetworkDirectRoomOpened)]; ok {
		predicates = append(predicates, "ntl.surface = 'direct'")
	}
	if networkWorkKindRequested(kindSet) {
		predicates = append(predicates, "COALESCE(ntl.work_id, '') <> ''")
	}
	if len(predicates) == 0 {
		return "", false
	}
	return strings.Join(predicates, " OR "), true
}

func (g *GlobalDB) networkWatchEventsForRow(
	ctx context.Context,
	row networkWatchEventRow,
	kinds []string,
) ([]looppkg.WatchEvent, error) {
	kindSet := stringSet(kinds)
	events := make([]looppkg.WatchEvent, 0, 5)
	workState := ""
	var workOutcome networkWorkOutcome
	var workOutcomeLoaded bool
	loadWorkOutcome := func() (networkWorkOutcome, error) {
		if workOutcomeLoaded {
			return workOutcome, nil
		}
		var err error
		workOutcome, err = g.networkWorkOutcomeAtTimelineRow(ctx, row)
		workOutcomeLoaded = true
		return workOutcome, err
	}
	if _, ok := kindSet[string(hookspkg.HookNetworkMessagePersisted)]; ok {
		if strings.TrimSpace(row.message.WorkID) != "" {
			outcome, err := loadWorkOutcome()
			if err != nil {
				return nil, err
			}
			workState = outcome.state
		}
		events = append(events, networkWatchEventFromRow(row, hookspkg.HookNetworkMessagePersisted, workState))
	}
	if _, ok := kindSet[string(hookspkg.HookNetworkThreadOpened)]; ok {
		opened, err := g.networkConversationOpenedAtTimelineRow(ctx, row, store.NetworkSurfaceThread)
		if err != nil {
			return nil, err
		}
		if opened {
			events = append(events, networkWatchEventFromRow(row, hookspkg.HookNetworkThreadOpened, ""))
		}
	}
	if _, ok := kindSet[string(hookspkg.HookNetworkDirectRoomOpened)]; ok {
		opened, err := g.networkConversationOpenedAtTimelineRow(ctx, row, store.NetworkSurfaceDirect)
		if err != nil {
			return nil, err
		}
		if opened {
			events = append(events, networkWatchEventFromRow(row, hookspkg.HookNetworkDirectRoomOpened, ""))
		}
	}
	if networkWorkKindRequested(kindSet) {
		outcome, err := loadWorkOutcome()
		if err != nil {
			return nil, err
		}
		if outcome.opened {
			appendRequestedNetworkWorkEvent(&events, row, kindSet, hookspkg.HookNetworkWorkOpened, outcome.state)
		}
		if outcome.transitioned {
			appendRequestedNetworkWorkEvent(&events, row, kindSet, hookspkg.HookNetworkWorkTransitioned, outcome.state)
		}
		if (outcome.opened || outcome.transitioned) && networkWorkStateIsTerminal(outcome.state) {
			appendRequestedNetworkWorkEvent(&events, row, kindSet, hookspkg.HookNetworkWorkClosed, outcome.state)
		}
	}
	return events, nil
}

func networkWatchEventFromRow(
	row networkWatchEventRow,
	kind hookspkg.HookEvent,
	workState string,
) looppkg.WatchEvent {
	payload := map[string]any{
		"session_id":   strings.TrimSpace(row.message.SessionID),
		"channel":      strings.TrimSpace(row.message.Channel),
		"surface":      strings.TrimSpace(row.message.Surface),
		"thread_id":    strings.TrimSpace(row.message.ThreadID),
		"direct_id":    strings.TrimSpace(row.message.DirectID),
		"message_id":   strings.TrimSpace(row.message.MessageID),
		"kind":         strings.TrimSpace(row.message.Kind),
		"direction":    strings.TrimSpace(row.message.Direction),
		"work_id":      strings.TrimSpace(row.message.WorkID),
		"work_state":   strings.TrimSpace(workState),
		"peer_from":    strings.TrimSpace(row.message.PeerFrom),
		"peer_to":      strings.TrimSpace(row.message.PeerTo),
		"trace_id":     strings.TrimSpace(row.message.TraceID),
		"causation_id": strings.TrimSpace(row.message.CausationID),
	}
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Seq:         row.seq,
		Stream:      looppkg.WatchEventsNetworkStream,
		At:          formatWatchEventAt(row.at),
		WorkspaceID: strings.TrimSpace(row.message.WorkspaceID),
		SessionID:   strings.TrimSpace(row.message.SessionID),
		Channel:     strings.TrimSpace(row.message.Channel),
		WorkID:      strings.TrimSpace(row.message.WorkID),
		Payload:     payload,
		LedgerKind:  string(kind),
	}
}

func appendRequestedNetworkWorkEvent(
	events *[]looppkg.WatchEvent,
	row networkWatchEventRow,
	kindSet map[string]struct{},
	kind hookspkg.HookEvent,
	workState string,
) {
	if _, ok := kindSet[string(kind)]; !ok {
		return
	}
	*events = append(*events, networkWatchEventFromRow(row, kind, workState))
}

func (g *GlobalDB) networkConversationOpenedAtTimelineRow(
	ctx context.Context,
	row networkWatchEventRow,
	surface string,
) (bool, error) {
	message := row.message
	if strings.TrimSpace(message.Surface) != surface {
		return false, nil
	}
	column := "thread_id"
	containerID := strings.TrimSpace(message.ThreadID)
	if surface == store.NetworkSurfaceDirect {
		column = "direct_id"
		containerID = strings.TrimSpace(message.DirectID)
	}
	if containerID == "" {
		return false, nil
	}
	var previous int
	// #nosec G202 -- column is selected from a fixed whitelist above; values are parameterized.
	if err := g.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		   FROM network_timeline_log
		  WHERE workspace_id = ?
		    AND channel = ?
		    AND surface = ?
		    AND `+column+` = ?
		    AND rowid < ?`,
		message.WorkspaceID,
		message.Channel,
		surface,
		containerID,
		row.seq,
	).Scan(&previous); err != nil {
		return false, fmt.Errorf("store: read network conversation open anchor: %w", err)
	}
	return previous == 0, nil
}

func (g *GlobalDB) networkWorkOutcomeAtTimelineRow(
	ctx context.Context,
	row networkWatchEventRow,
) (networkWorkOutcome, error) {
	workID := strings.TrimSpace(row.message.WorkID)
	if workID == "" {
		return networkWorkOutcome{}, nil
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			ntl.rowid,
			ntl.message_id,
			COALESCE(ntl.session_id, ''),
			ntl.workspace_id,
			ntl.channel,
			COALESCE(ntl.surface, ''),
			COALESCE(ntl.thread_id, ''),
			COALESCE(ntl.direct_id, ''),
			ntl.direction,
			ntl.peer_from,
			COALESCE(ntl.peer_to, ''),
			ntl.kind,
			COALESCE(ntl.work_id, ''),
			COALESCE(ntl.reply_to, ''),
			COALESCE(ntl.trace_id, ''),
			COALESCE(ntl.causation_id, ''),
			COALESCE(ntl.intent, ''),
			COALESCE(ntl.text, ''),
			ntl.body_json,
			ntl.timestamp
		   FROM network_timeline_log ntl
		  WHERE ntl.workspace_id = ?
		    AND ntl.work_id = ?
		    AND ntl.rowid <= ?
		  ORDER BY ntl.rowid ASC`,
		row.message.WorkspaceID,
		workID,
		row.seq,
	)
	if err != nil {
		return networkWorkOutcome{}, fmt.Errorf("store: read network work timeline: %w", err)
	}
	defer rows.Close()

	state := ""
	outcome := networkWorkOutcome{}
	for rows.Next() {
		candidate, scanErr := scanNetworkWatchEventRow(rows)
		if scanErr != nil {
			return networkWorkOutcome{}, joinRowsCloseError(rows, scanErr, "network work timeline query")
		}
		isTarget := candidate.seq == row.seq
		if state == "" {
			if networkMessageOpensWork(candidate.message) {
				state = store.NetworkWorkStateSubmitted
				if isTarget {
					outcome.opened = true
				}
			}
			if isTarget {
				outcome.state = state
			}
			continue
		}
		next, transitioned, stateErr := nextNetworkWorkState(state, candidate.message)
		if stateErr != nil {
			return networkWorkOutcome{}, joinRowsCloseError(rows, stateErr, "network work timeline query")
		}
		if transitioned {
			state = next
		}
		if isTarget {
			outcome.transitioned = transitioned
			outcome.state = state
		}
	}
	if err := rows.Err(); err != nil {
		return networkWorkOutcome{}, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate network work timeline: %w", err),
			"network work timeline query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "network work timeline query"); err != nil {
		return networkWorkOutcome{}, err
	}
	return outcome, nil
}

func networkMessageOpensWork(message store.NetworkConversationMessage) bool {
	if strings.TrimSpace(message.WorkID) == "" || strings.TrimSpace(message.PeerTo) == "" {
		return false
	}
	switch strings.TrimSpace(message.Kind) {
	case store.NetworkKindSay, store.NetworkKindCapability:
		return true
	default:
		return false
	}
}

func networkWorkKindRequested(kindSet map[string]struct{}) bool {
	for _, kind := range []hookspkg.HookEvent{
		hookspkg.HookNetworkWorkOpened,
		hookspkg.HookNetworkWorkTransitioned,
		hookspkg.HookNetworkWorkClosed,
	} {
		if _, ok := kindSet[string(kind)]; ok {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	return set
}
