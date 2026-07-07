package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
)

const (
	loopRunEventNodeRunning       = "node_running"
	loopRunEventNodeSucceeded     = "node_succeeded"
	loopRunEventNodeFailed        = "node_failed"
	loopRunEventGateVerdict       = "gate_verdict"
	loopRunEventGenerationStarted = "generation_started"
	loopRunEventChannelMsg        = "channel_msg"
	loopRunEventTokenTick         = "token_tick"
	loopRunEventNeedsApproval     = "needs_approval"
	loopRunEventStatusChanged     = "status_changed"

	maxLoopRunEventPayloadBytes = 16 * 1024
	loopTokenTickMinDelta       = 2000
	loopTokenTickMinInterval    = 5 * time.Second

	loopRunEventPayloadKeyGeneration = "generation"
	loopRunEventPayloadKeyItemIndex  = "item_index"
	loopRunEventPayloadKeyNodeID     = "node_id"
	loopRunEventPayloadKeyReason     = "reason"
	loopRunEventPayloadKeyRole       = "role"
	loopRunEventPayloadKeySummary    = "summary"
	loopRunEventPayloadKeyStatus     = "status"
	loopRunEventPayloadKeyTaskID     = "task_id"
	loopRunEventPayloadKeyTaskRunID  = "task_run_id"
	loopRunEventPayloadKeyTerminal   = "terminal"
	loopRunEventPayloadKeyText       = "text"
	loopRunEventPayloadKeyTitle      = "title"
	loopRunEventPayloadKeyValue      = "value"
	loopRunEventVerdictRevise        = "revise"
	loopRunApprovalFactLabelKey      = "label"
	loopRunNodeOutputRunning         = "running"
)

func appendLoopRunStatusEvent(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	ws looppkg.WorkspaceID,
	from looppkg.Status,
	to looppkg.Status,
	cause looppkg.TransitionCause,
	at time.Time,
) error {
	if from == to {
		return nil
	}
	return appendLoopRunEventWithExecutor(ctx, exec, runID, ws, loopRunEventStatusChanged, map[string]string{
		"from":                       string(from),
		"to":                         string(to),
		loopRunEventPayloadKeyStatus: string(to),
		"cause":                      string(cause),
	}, at)
}

func appendLoopRunEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	ws looppkg.WorkspaceID,
	kind string,
	payload any,
	at time.Time,
) error {
	runID = looppkg.RunID(strings.TrimSpace(string(runID)))
	ws = looppkg.WorkspaceID(strings.TrimSpace(string(ws)))
	kind = strings.TrimSpace(kind)
	if runID == "" {
		return fmt.Errorf("%w: loop event run_id is required", looppkg.ErrValidation)
	}
	if ws == "" {
		return fmt.Errorf("%w: loop event workspace_id is required", looppkg.ErrValidation)
	}
	if !loopRunEventKindValid(kind) {
		return fmt.Errorf("%w: loop run event kind is invalid: %q", looppkg.ErrValidation, kind)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	payloadJSON, err := normalizeLoopRunEventPayload(kind, payload)
	if err != nil {
		return err
	}
	seq, err := nextLoopRunEventSequence(ctx, exec, runID)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(
		ctx,
		`INSERT INTO loop_run_events (id, loop_run_id, workspace_id, seq, kind, payload_json, at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		store.NewID("loopevt"),
		string(runID),
		string(ws),
		seq,
		kind,
		string(payloadJSON),
		store.FormatTimestamp(at),
	)
	if err != nil {
		return fmt.Errorf("store: insert loop run event %q: %w", kind, err)
	}
	return nil
}

func nextLoopRunEventSequence(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
) (int64, error) {
	var next sql.NullInt64
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM loop_run_events WHERE loop_run_id = ?`,
		string(runID),
	).Scan(&next); err != nil {
		return 0, fmt.Errorf("store: select next loop run event sequence: %w", err)
	}
	if !next.Valid {
		return 1, nil
	}
	return next.Int64, nil
}

func loopRunEventKindValid(kind string) bool {
	switch kind {
	case loopRunEventNodeRunning,
		loopRunEventNodeSucceeded,
		loopRunEventNodeFailed,
		loopRunEventGateVerdict,
		loopRunEventGenerationStarted,
		loopRunEventChannelMsg,
		loopRunEventTokenTick,
		loopRunEventNeedsApproval,
		loopRunEventStatusChanged:
		return true
	default:
		return false
	}
}
