package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

func networkWatchEventFromRow(
	row networkWatchEventRow,
	kind hookspkg.HookEvent,
	workState string,
) looppkg.WatchEvent {
	payload := map[string]any{
		watchEventsPayloadSessionIDKey:   strings.TrimSpace(row.message.SessionID),
		watchEventsPayloadChannelKey:     strings.TrimSpace(row.message.Channel),
		watchEventsPayloadSurfaceKey:     strings.TrimSpace(row.message.Surface),
		watchEventsPayloadThreadIDKey:    strings.TrimSpace(row.message.ThreadID),
		watchEventsPayloadDirectIDKey:    strings.TrimSpace(row.message.DirectID),
		watchEventsPayloadMessageIDKey:   strings.TrimSpace(row.message.MessageID),
		taskRunResultKindKey:             strings.TrimSpace(row.message.Kind),
		watchEventsPayloadDirectionKey:   strings.TrimSpace(row.message.Direction),
		watchEventsPayloadWorkIDKey:      strings.TrimSpace(row.message.WorkID),
		watchEventsPayloadWorkStateKey:   strings.TrimSpace(workState),
		watchEventsPayloadPeerFromKey:    strings.TrimSpace(row.message.PeerFrom),
		watchEventsPayloadPeerToKey:      strings.TrimSpace(row.message.PeerTo),
		watchEventsPayloadTraceIDKey:     strings.TrimSpace(row.message.TraceID),
		watchEventsPayloadCausationIDKey: strings.TrimSpace(row.message.CausationID),
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
	column := watchEventsPayloadThreadIDKey
	containerID := strings.TrimSpace(message.ThreadID)
	if surface == store.NetworkSurfaceDirect {
		column = watchEventsPayloadDirectIDKey
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
	rows, err := g.readNetworkWorkTimelineRows(ctx, row, workID)
	if err != nil {
		return networkWorkOutcome{}, err
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

func (g *GlobalDB) readNetworkWorkTimelineRows(
	ctx context.Context,
	row networkWatchEventRow,
	workID string,
) (*sql.Rows, error) {
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
		return nil, fmt.Errorf("store: read network work timeline: %w", err)
	}
	return rows, nil
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
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	return set
}
