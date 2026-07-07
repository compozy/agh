package globaldb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/diagnostics"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
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

func normalizeLoopRunEventPayload(kind string, payload any) (json.RawMessage, error) {
	var raw json.RawMessage
	switch typed := payload.(type) {
	case nil:
		raw = json.RawMessage(`{}`)
	case json.RawMessage:
		raw = cloneLoopEventRawJSON(typed)
	case []byte:
		raw = cloneLoopEventRawJSON(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("store: marshal loop run event payload: %w", err)
		}
		raw = data
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("%w: loop run event payload must be valid JSON", looppkg.ErrValidation)
	}
	if kind == loopRunEventTokenTick {
		return normalizeLoopTokenTickEventPayload(raw)
	}
	redacted, err := diagnostics.RedactJSON(raw)
	if err != nil {
		return boundedLoopRunEventTextPayload(string(raw))
	}
	if len(redacted) > maxLoopRunEventPayloadBytes {
		return boundedLoopRunEventTextPayload(string(redacted))
	}
	return cloneLoopEventRawJSON(redacted), nil
}

func normalizeLoopTokenTickEventPayload(raw json.RawMessage) (json.RawMessage, error) {
	var tick map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tick); err != nil {
		return nil, fmt.Errorf("%w: token_tick payload is invalid: %w", looppkg.ErrValidation, err)
	}
	var taskRunID string
	if taskRunRaw := tick[loopRunEventPayloadKeyTaskRunID]; len(bytes.TrimSpace(taskRunRaw)) > 0 {
		if err := json.Unmarshal(taskRunRaw, &taskRunID); err != nil {
			return nil, fmt.Errorf("%w: token_tick payload task_run_id is invalid: %w", looppkg.ErrValidation, err)
		}
	}
	tokensUsedRaw := tick[columnTokensUsed]
	if len(bytes.TrimSpace(tokensUsedRaw)) == 0 {
		return nil, fmt.Errorf("%w: token_tick payload tokens_used is required", looppkg.ErrValidation)
	}
	var tokensUsed int64
	if err := json.Unmarshal(tokensUsedRaw, &tokensUsed); err != nil {
		return nil, fmt.Errorf("%w: token_tick payload tokens_used is invalid: %w", looppkg.ErrValidation, err)
	}
	var terminal bool
	if terminalRaw := tick[loopRunEventPayloadKeyTerminal]; len(bytes.TrimSpace(terminalRaw)) > 0 {
		if err := json.Unmarshal(terminalRaw, &terminal); err != nil {
			return nil, fmt.Errorf("%w: token_tick payload terminal is invalid: %w", looppkg.ErrValidation, err)
		}
	}
	data, err := json.Marshal(map[string]any{
		loopRunEventPayloadKeyTaskRunID: strings.TrimSpace(taskRunID),
		columnTokensUsed:                tokensUsed,
		loopRunEventPayloadKeyTerminal:  terminal,
	})
	if err != nil {
		return nil, fmt.Errorf("store: marshal token_tick loop run event payload: %w", err)
	}
	return data, nil
}

func boundedLoopRunEventTextPayload(value string) (json.RawMessage, error) {
	bounded := diagnostics.RedactAndBound(taskpkg.RedactClaimTokens(value), maxLoopRunEventPayloadBytes/2)
	data, err := json.Marshal(map[string]any{
		loopRunEventPayloadKeyText: bounded,
		"truncated":                true,
	})
	if err != nil {
		return nil, fmt.Errorf("store: marshal bounded loop run event payload: %w", err)
	}
	return data, nil
}

func appendLoopNodeRunningEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	at time.Time,
) error {
	if !taskRunIsLoopWorker(run) {
		return nil
	}
	loopRun, metadata, ok, err := loopRunAndNodeMetadataForTaskRun(ctx, exec, run)
	if err != nil || !ok {
		return err
	}
	payload := map[string]any{
		loopRunEventPayloadKeyNodeID:     metadata.NodeID,
		loopRunEventPayloadKeyGeneration: metadata.Generation,
		loopRunEventPayloadKeyItemIndex:  metadata.ItemIndex,
		loopRunEventPayloadKeyTaskRunID:  run.ID,
		loopRunEventPayloadKeyTaskID:     run.TaskID,
		loopRunEventPayloadKeyStatus:     loopRunNodeOutputRunning,
	}
	return appendLoopRunEventWithExecutor(
		ctx,
		exec,
		loopRun.ID,
		loopRun.WorkspaceID,
		loopRunEventNodeRunning,
		payload,
		at,
	)
}

func appendLoopNodeTerminalEventsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
	outcome string,
	outputRef string,
	resultPayload json.RawMessage,
	tokensUsed int64,
	at time.Time,
) error {
	if !taskRunIsLoopWorker(run) {
		return nil
	}
	loopRun, metadata, ok, err := loopRunAndNodeMetadataForTaskRun(ctx, exec, run)
	if err != nil || !ok {
		return err
	}
	kind := loopRunEventNodeSucceeded
	status := loopNodeOutputSucceeded
	if outcome == loopNodeOutcomeFailure {
		kind = loopRunEventNodeFailed
		status = loopNodeOutputFailed
	}
	if err := appendLoopRunEventWithExecutor(ctx, exec, loopRun.ID, loopRun.WorkspaceID, kind, map[string]any{
		loopRunEventPayloadKeyNodeID:     metadata.NodeID,
		loopRunEventPayloadKeyGeneration: metadata.Generation,
		loopRunEventPayloadKeyItemIndex:  metadata.ItemIndex,
		loopRunEventPayloadKeyTaskRunID:  run.ID,
		loopRunEventPayloadKeyTaskID:     run.TaskID,
		loopRunEventPayloadKeyStatus:     status,
		"output_ref":                     strings.TrimSpace(outputRef),
	}, at); err != nil {
		return err
	}
	if payload, ok := loopChannelMessagePayload(run, metadata, resultPayload); ok {
		if err := appendLoopRunEventWithExecutor(
			ctx,
			exec,
			loopRun.ID,
			loopRun.WorkspaceID,
			loopRunEventChannelMsg,
			payload,
			at,
		); err != nil {
			return err
		}
	}
	return appendLoopTokenTickEventWithExecutor(
		ctx,
		exec,
		loopRun.ID,
		loopRun.WorkspaceID,
		run.ID,
		tokensUsed,
		true,
		at,
	)
}

func taskRunIsLoopWorker(run taskpkg.Run) bool {
	return strings.TrimSpace(run.LoopRunID) != "" && run.RunKind.Normalize() != taskpkg.RunKindCoordinator
}

func loopRunAndNodeMetadataForTaskRun(
	ctx context.Context,
	exec taskSQLExecutor,
	run taskpkg.Run,
) (looppkg.Run, loopNodeRunMetadata, bool, error) {
	metadata, ok, err := loopNodeMetadataFromTaskRun(run.Metadata)
	if err != nil || !ok {
		return looppkg.Run{}, loopNodeRunMetadata{}, false, err
	}
	loopRun, err := getLoopRunByIDWithExecutor(ctx, exec, looppkg.RunID(strings.TrimSpace(run.LoopRunID)))
	if err != nil {
		return looppkg.Run{}, loopNodeRunMetadata{}, false, err
	}
	return loopRun, metadata, true, nil
}

func loopChannelMessagePayload(
	run taskpkg.Run,
	metadata loopNodeRunMetadata,
	resultPayload json.RawMessage,
) (map[string]any, bool) {
	text := channelMessageText(resultPayload)
	if strings.TrimSpace(text) == "" {
		return nil, false
	}
	author := loopCoordinatorActorRef
	if run.ClaimedBy != nil && strings.TrimSpace(run.ClaimedBy.Ref) != "" {
		author = strings.TrimSpace(run.ClaimedBy.Ref)
	}
	return map[string]any{
		"id":                             run.ID + ":result",
		"author":                         author,
		loopRunEventPayloadKeyRole:       string(taskpkg.ActorKindAgentSession),
		loopRunEventPayloadKeyText:       text,
		"is_result":                      true,
		loopRunEventPayloadKeyNodeID:     metadata.NodeID,
		loopRunEventPayloadKeyGeneration: metadata.Generation,
		loopRunEventPayloadKeyItemIndex:  metadata.ItemIndex,
		loopRunEventPayloadKeyTaskRunID:  run.ID,
	}, true
}

func channelMessageText(raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	for _, key := range []string{"message", "text", loopRunEventPayloadKeySummary} {
		if value, ok := envelope[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendLoopTokenTickEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	ws looppkg.WorkspaceID,
	taskRunID string,
	tokensUsed int64,
	terminal bool,
	at time.Time,
) error {
	if !terminal {
		ok, err := shouldPersistLoopTokenTick(ctx, exec, runID, tokensUsed, at)
		if err != nil || !ok {
			return err
		}
	}
	return appendLoopRunEventWithExecutor(ctx, exec, runID, ws, loopRunEventTokenTick, map[string]any{
		loopRunEventPayloadKeyTaskRunID: strings.TrimSpace(taskRunID),
		columnTokensUsed:                tokensUsed,
		loopRunEventPayloadKeyTerminal:  terminal,
	}, at)
}

func shouldPersistLoopTokenTick(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	tokensUsed int64,
	at time.Time,
) (bool, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	var payload string
	var atRaw string
	err := exec.QueryRowContext(
		ctx,
		`SELECT payload_json, at
		   FROM loop_run_events
		  WHERE loop_run_id = ? AND kind = ?
		  ORDER BY seq DESC
		  LIMIT 1`,
		string(runID),
		loopRunEventTokenTick,
	).Scan(&payload, &atRaw)
	if errorsIsNoRows(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: load last loop token tick: %w", err)
	}
	var last map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload), &last); err != nil {
		return false, fmt.Errorf("store: decode last loop token tick: %w", err)
	}
	var lastTokensUsed int64
	if err := json.Unmarshal(last[columnTokensUsed], &lastTokensUsed); err != nil {
		return false, fmt.Errorf("store: decode last loop token tick tokens_used: %w", err)
	}
	lastAt, err := parseLoopRunTimestamp(atRaw)
	if err != nil {
		return false, err
	}
	if at.Sub(lastAt) < loopTokenTickMinInterval {
		return false, nil
	}
	delta := tokensUsed - lastTokensUsed
	if delta < 0 {
		delta = -delta
	}
	return delta >= loopTokenTickMinDelta, nil
}

func appendLoopGenerationStartedEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	generation int,
	at time.Time,
) error {
	if generation <= 0 || generation <= run.Generation {
		return nil
	}
	payload := map[string]any{
		loopRunEventPayloadKeyGeneration: generation,
		"reattempt_strategy":             string(run.ReattemptStrategy),
		columnLoopName:                   run.LoopName,
	}
	return appendLoopRunEventWithExecutor(
		ctx,
		exec,
		run.ID,
		run.WorkspaceID,
		loopRunEventGenerationStarted,
		payload,
		at,
	)
}

func appendLoopGateVerdictEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	generation int,
	terminal taskpkg.CoordinatorTerminal,
	at time.Time,
) error {
	if len(bytes.TrimSpace(terminal.Details)) == 0 {
		return nil
	}
	payload := loopGateVerdictEventPayload(generation, terminal)
	return appendLoopRunEventWithExecutor(ctx, exec, run.ID, run.WorkspaceID, loopRunEventGateVerdict, payload, at)
}

func loopGateVerdictEventPayload(
	generation int,
	terminal taskpkg.CoordinatorTerminal,
) map[string]any {
	gateID := strings.TrimSpace(terminal.GateID)
	var verdict gate.Verdict
	if err := json.Unmarshal(terminal.Details, &verdict); err != nil {
		return map[string]any{
			"node_id":                        gateID,
			loopRunEventPayloadKeyGeneration: generation,
			"verdict":                        loopRunEventVerdictRevise,
			loopRunEventPayloadKeyReason:     strings.TrimSpace(terminal.ReasonCode),
			"route":                          strings.TrimSpace(terminal.Status),
			"details":                        cloneLoopEventRawJSON(terminal.Details),
		}
	}
	verdictLabel := loopRunEventVerdictRevise
	if verdict.Outcome == gate.VerdictOutcomeApproved {
		verdictLabel = "pass"
	}
	criteria := make([]map[string]any, 0, len(verdict.Criteria))
	for _, criterion := range verdict.Criteria {
		status := loopRunEventVerdictRevise
		if criterion.Passed {
			status = "pass"
		}
		criteria = append(criteria, map[string]any{
			"id":                         strings.TrimSpace(criterion.ID),
			"type":                       string(criterion.Type),
			loopRunEventPayloadKeyStatus: status,
			"note":                       string(criterion.Outcome),
		})
	}
	issues := make([]map[string]any, 0, len(verdict.BlockingIssues))
	for _, issue := range verdict.BlockingIssues {
		issues = append(issues, map[string]any{
			"id":   strings.TrimSpace(issue.ID),
			"note": strings.TrimSpace(issue.Note),
		})
	}
	payload := map[string]any{
		"node_id":                        firstNonEmptyString(gateID, "definition_of_done"),
		loopRunEventPayloadKeyGeneration: generation,
		"verdict":                        verdictLabel,
		loopRunEventPayloadKeyReason: firstNonEmptyString(
			verdict.Route.ReasonCode,
			string(verdict.Outcome),
		),
		"route":           string(verdict.Route.Action),
		"criteria":        criteria,
		"blocking_issues": issues,
	}
	if len(verdict.Criteria) > 0 && verdict.Criteria[0].Confidence != nil {
		payload["confidence"] = *verdict.Criteria[0].Confidence
	}
	return payload
}

func appendLoopNeedsApprovalEventWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	run looppkg.Run,
	gateID looppkg.NodeID,
	generation int,
	title string,
	facts []map[string]string,
	at time.Time,
) error {
	return appendLoopRunEventWithExecutor(ctx, exec, run.ID, run.WorkspaceID, loopRunEventNeedsApproval, map[string]any{
		"gate_id":                        strings.TrimSpace(string(gateID)),
		loopRunEventPayloadKeyTitle:      firstNonEmptyString(title, "Approve to resume"),
		loopRunEventPayloadKeyGeneration: generation,
		"facts":                          facts,
	}, at)
}

func loopApprovalFact(label string, value string) map[string]string {
	return map[string]string{
		loopRunApprovalFactLabelKey: label,
		loopRunEventPayloadKeyValue: value,
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneLoopEventRawJSON(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}
