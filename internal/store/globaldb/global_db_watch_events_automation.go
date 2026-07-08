package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/automation"
	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

const automationWatchEventsWorkspaceExpr = `COALESCE(
	NULLIF(aj.workspace_id, ''),
	NULLIF(at.workspace_id, ''),
	NULLIF(lr.workspace_id, ''),
	NULLIF(aj.loop_workspace_id, ''),
	NULLIF(at.loop_workspace_id, ''),
	''
)`

type automationWatchEventRow struct {
	seq         int64
	runID       string
	jobID       string
	triggerID   string
	sessionID   string
	status      string
	attempt     int
	startedAt   sql.NullString
	endedAt     sql.NullString
	errorText   string
	agentName   string
	workspaceID string
	retryRaw    string
}

func (g *GlobalDB) readAutomationWatchEventsCursor(
	ctx context.Context,
	query normalizedWatchEventsQuery,
) (int64, error) {
	statuses := automationWatchStatusesForKinds(query.kinds)
	if len(statuses) == 0 {
		return 0, nil
	}
	placeholders, statusArgs := sqlInPlaceholders(statuses)
	args := append([]any{query.workspaceID}, statusArgs...)
	// #nosec G202 -- IN placeholders are generated from normalized status count; values are parameterized.
	return scanWatchEventCursor(
		g.db.QueryRowContext(
			ctx,
			`SELECT COALESCE(MAX(ar.rowid), 0)
			   FROM automation_runs ar
			   LEFT JOIN automation_jobs aj ON aj.id = ar.job_id
			   LEFT JOIN automation_triggers at ON at.id = ar.trigger_id
			   LEFT JOIN loop_runs lr ON lr.id = ar.loop_run_id
			  WHERE `+automationWatchEventsWorkspaceExpr+` = ?
			    AND ar.status IN (`+placeholders+`)`,
			args...,
		),
		looppkg.WatchEventsAutomationStream,
	)
}

func (g *GlobalDB) readAutomationWatchEvents(
	ctx context.Context,
	query normalizedWatchEventsQuery,
) ([]looppkg.WatchEvent, error) {
	statuses := automationWatchStatusesForKinds(query.kinds)
	if len(statuses) == 0 {
		return nil, nil
	}
	placeholders, statusArgs := sqlInPlaceholders(statuses)
	args := append([]any{
		query.workspaceID,
		query.streams[looppkg.WatchEventsAutomationStream],
	}, statusArgs...)
	args = append(args, query.limit)
	// #nosec G202 -- IN placeholders are generated from normalized status count; values are parameterized.
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT
			ar.rowid,
			ar.id,
			COALESCE(ar.job_id, ''),
			COALESCE(ar.trigger_id, ''),
			COALESCE(ar.session_id, ''),
			ar.status,
			ar.attempt,
			ar.started_at,
			ar.ended_at,
			COALESCE(ar.error, ''),
			COALESCE(aj.agent_name, at.agent_name, ''),
			`+automationWatchEventsWorkspaceExpr+`,
			COALESCE(aj.retry, at.retry, '')
		   FROM automation_runs ar
		   LEFT JOIN automation_jobs aj ON aj.id = ar.job_id
		   LEFT JOIN automation_triggers at ON at.id = ar.trigger_id
		   LEFT JOIN loop_runs lr ON lr.id = ar.loop_run_id
		  WHERE `+automationWatchEventsWorkspaceExpr+` = ?
		    AND ar.rowid > ?
		    AND ar.status IN (`+placeholders+`)
		  ORDER BY ar.rowid ASC
		  LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("store: read automation watch-events: %w", err)
	}
	defer rows.Close()

	events := make([]looppkg.WatchEvent, 0)
	for rows.Next() {
		row, scanErr := scanAutomationWatchEventRow(rows)
		if scanErr != nil {
			return nil, joinRowsCloseError(rows, scanErr, "automation watch-events query")
		}
		event, eventErr := automationWatchEventFromRow(row)
		if eventErr != nil {
			return nil, joinRowsCloseError(rows, eventErr, "automation watch-events query")
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, joinRowsCloseError(
			rows,
			fmt.Errorf("store: iterate automation watch-events: %w", err),
			"automation watch-events query",
		)
	}
	if err := joinRowsCloseError(rows, nil, "automation watch-events query"); err != nil {
		return nil, err
	}
	return events, nil
}

func scanAutomationWatchEventRow(row rowScanner) (automationWatchEventRow, error) {
	var event automationWatchEventRow
	if err := row.Scan(
		&event.seq,
		&event.runID,
		&event.jobID,
		&event.triggerID,
		&event.sessionID,
		&event.status,
		&event.attempt,
		&event.startedAt,
		&event.endedAt,
		&event.errorText,
		&event.agentName,
		&event.workspaceID,
		&event.retryRaw,
	); err != nil {
		return automationWatchEventRow{}, fmt.Errorf("store: scan automation watch-event: %w", err)
	}
	return event, nil
}

func automationWatchEventFromRow(row automationWatchEventRow) (looppkg.WatchEvent, error) {
	kind, ok := automationWatchKindForStatus(row.status)
	if !ok {
		return looppkg.WatchEvent{}, fmt.Errorf("store: unsupported automation watch-events status %q", row.status)
	}
	at, err := automationWatchEventAt(row)
	if err != nil {
		return looppkg.WatchEvent{}, err
	}
	payload := map[string]any{
		"job_id":     strings.TrimSpace(row.jobID),
		"trigger_id": strings.TrimSpace(row.triggerID),
		"agent_name": strings.TrimSpace(row.agentName),
		"session_id": strings.TrimSpace(row.sessionID),
		"attempt":    row.attempt,
	}
	if kind == hookspkg.HookAutomationRunCompleted {
		durationMS, durationErr := automationWatchEventDurationMS(row)
		if durationErr != nil {
			return looppkg.WatchEvent{}, durationErr
		}
		payload["duration_ms"] = durationMS
	} else {
		payload["error"] = strings.TrimSpace(row.errorText)
		payload["will_retry"] = automationWatchEventWillRetry(row)
	}
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Seq:         row.seq,
		Stream:      looppkg.WatchEventsAutomationStream,
		At:          formatWatchEventAt(at),
		WorkspaceID: strings.TrimSpace(row.workspaceID),
		RunID:       strings.TrimSpace(row.runID),
		SessionID:   strings.TrimSpace(row.sessionID),
		Payload:     payload,
		LedgerKind:  string(kind),
	}, nil
}

func automationWatchStatusesForKinds(kinds []string) []string {
	statuses := make([]string, 0, 2)
	for _, kind := range kinds {
		switch strings.TrimSpace(kind) {
		case string(hookspkg.HookAutomationRunCompleted):
			statuses = append(statuses, string(automation.RunCompleted))
		case string(hookspkg.HookAutomationRunFailed):
			statuses = append(statuses, string(automation.RunFailed))
		}
	}
	return uniqueTrimmedStrings(statuses)
}

func automationWatchKindForStatus(status string) (hookspkg.HookEvent, bool) {
	switch automation.RunStatus(strings.TrimSpace(status)) {
	case automation.RunCompleted:
		return hookspkg.HookAutomationRunCompleted, true
	case automation.RunFailed:
		return hookspkg.HookAutomationRunFailed, true
	default:
		return "", false
	}
}

func automationWatchEventAt(row automationWatchEventRow) (time.Time, error) {
	for _, raw := range []sql.NullString{row.endedAt, row.startedAt} {
		if !raw.Valid || strings.TrimSpace(raw.String) == "" {
			continue
		}
		parsed, err := store.ParseTimestamp(raw.String)
		if err != nil {
			return time.Time{}, fmt.Errorf("store: parse automation watch-event timestamp: %w", err)
		}
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("store: automation watch-event %q has no timestamp", row.runID)
}

func automationWatchEventDurationMS(row automationWatchEventRow) (int64, error) {
	if !row.startedAt.Valid || !row.endedAt.Valid ||
		strings.TrimSpace(row.startedAt.String) == "" ||
		strings.TrimSpace(row.endedAt.String) == "" {
		return 0, nil
	}
	started, err := store.ParseTimestamp(row.startedAt.String)
	if err != nil {
		return 0, fmt.Errorf("store: parse automation watch-event started_at: %w", err)
	}
	ended, err := store.ParseTimestamp(row.endedAt.String)
	if err != nil {
		return 0, fmt.Errorf("store: parse automation watch-event ended_at: %w", err)
	}
	return ended.UTC().Sub(started.UTC()).Milliseconds(), nil
}

func automationWatchEventWillRetry(row automationWatchEventRow) bool {
	var retry automation.RetryConfig
	if err := decodeAutomationJSON(row.retryRaw, &retry, "automation.watch_events.retry"); err != nil {
		return false
	}
	return retry.Strategy == automation.RetryStrategyBackoff && row.attempt <= retry.MaxRetries
}
