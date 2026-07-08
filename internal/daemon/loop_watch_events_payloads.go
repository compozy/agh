package daemon

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	taskpkg "github.com/compozy/agh/internal/task"
)

const (
	watchEventsPayloadDetailsKey      = "details"
	watchEventsPayloadErrorKey        = "error"
	watchEventsPayloadKindKey         = "kind"
	watchEventsPayloadNodeIDKey       = "node_id"
	watchEventsPayloadParentTaskIDKey = "parent_task_id"
	watchEventsPayloadReasonKey       = "reason"
)

func watchEventsTaskStatusChangedEvent(
	payload hookspkg.TaskStatusChangedPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(hookspkg.HookTaskStatusChanged),
		Stream:      looppkg.WatchEventsTaskStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		Payload: map[string]any{
			"from_status":                     strings.TrimSpace(payload.FromStatus),
			"to_status":                       strings.TrimSpace(payload.ToStatus),
			watchEventsPayloadParentTaskIDKey: strings.TrimSpace(payload.ParentTaskID),
		},
		LedgerKind: string(hookspkg.HookTaskStatusChanged),
	}
}

func watchEventsTaskBlockEvent(
	kind hookspkg.HookEvent,
	payload hookspkg.TaskBlockPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Stream:      looppkg.WatchEventsTaskStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		Payload: map[string]any{
			"block_id":                        strings.TrimSpace(payload.BlockID),
			watchEventsPayloadKindKey:         strings.TrimSpace(payload.Kind),
			watchEventsPayloadReasonKey:       strings.TrimSpace(payload.Reason),
			watchEventsPayloadDetailsKey:      watchEventsRawJSONValue(payload.Details),
			"cleared_at":                      watchEventsOptionalTime(payload.ClearedAt),
			"clear_note":                      strings.TrimSpace(payload.ClearNote),
			watchEventsPayloadParentTaskIDKey: strings.TrimSpace(payload.ParentTaskID),
		},
		LedgerKind: string(kind),
	}
}

func watchEventsTaskAttentionEvent(
	kind hookspkg.HookEvent,
	payload hookspkg.TaskAttentionPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Stream:      looppkg.WatchEventsTaskStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		Payload: map[string]any{
			watchEventsPayloadReasonKey:       strings.TrimSpace(payload.Reason),
			"note":                            strings.TrimSpace(payload.Note),
			"at":                              watchEventsOptionalTime(payload.At),
			watchEventsPayloadParentTaskIDKey: strings.TrimSpace(payload.ParentTaskID),
		},
		LedgerKind: string(kind),
	}
}

func watchEventsTaskRunTerminalEvent(
	payload hookspkg.TaskRunLeasePayload,
	now func() time.Time,
) looppkg.WatchEvent {
	kind := payload.Event
	if strings.TrimSpace(string(kind)) == "" {
		kind = hookspkg.HookTaskRunCompleted
		if taskpkg.ParseRunStatus(payload.RunStatus).Normalize() == taskpkg.TaskRunStatusFailed {
			kind = hookspkg.HookTaskRunFailed
		}
	}
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Stream:      looppkg.WatchEventsTaskStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		LoopRunID:   strings.TrimSpace(payload.LoopRunID),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Channel:     strings.TrimSpace(payload.NetworkChannel),
		Payload: map[string]any{
			"previous_run_status":      strings.TrimSpace(payload.PreviousRunStatus),
			"previous_session_id":      strings.TrimSpace(payload.PreviousSessionID),
			"recovery_action":          strings.TrimSpace(payload.RecoveryAction),
			"recovery_reason":          strings.TrimSpace(payload.RecoveryReason),
			watchEventsPayloadErrorKey: strings.TrimSpace(payload.Error),
		},
		LedgerKind: string(kind),
	}
}

func watchEventsAutomationRunCompletedEvent(
	payload hookspkg.AutomationRunCompletedPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(hookspkg.HookAutomationRunCompleted),
		Stream:      looppkg.WatchEventsAutomationStream,
		At:          watchEventsHookAt(time.Time{}, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		RunID:       strings.TrimSpace(payload.RunID),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Payload: map[string]any{
			"job_id":      strings.TrimSpace(payload.JobID),
			"trigger_id":  strings.TrimSpace(payload.TriggerID),
			"agent_name":  strings.TrimSpace(payload.AgentName),
			"session_id":  strings.TrimSpace(payload.SessionID),
			"attempt":     payload.Attempt,
			"duration_ms": payload.DurationMS,
		},
		LedgerKind: string(hookspkg.HookAutomationRunCompleted),
	}
}

func watchEventsAutomationRunFailedEvent(
	payload hookspkg.AutomationRunFailedPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(hookspkg.HookAutomationRunFailed),
		Stream:      looppkg.WatchEventsAutomationStream,
		At:          watchEventsHookAt(time.Time{}, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		RunID:       strings.TrimSpace(payload.RunID),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Payload: map[string]any{
			"job_id":                   strings.TrimSpace(payload.JobID),
			"trigger_id":               strings.TrimSpace(payload.TriggerID),
			"agent_name":               strings.TrimSpace(payload.AgentName),
			"session_id":               strings.TrimSpace(payload.SessionID),
			"attempt":                  payload.Attempt,
			watchEventsPayloadErrorKey: strings.TrimSpace(payload.Error),
			"will_retry":               payload.WillRetry,
		},
		LedgerKind: string(hookspkg.HookAutomationRunFailed),
	}
}

func watchEventsNetworkEvent(
	kind hookspkg.HookEvent,
	payload hookspkg.NetworkPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(kind),
		Stream:      looppkg.WatchEventsNetworkStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Channel:     strings.TrimSpace(payload.Channel),
		WorkID:      strings.TrimSpace(payload.WorkID),
		Payload: map[string]any{
			"session_id":              strings.TrimSpace(payload.SessionID),
			"channel":                 strings.TrimSpace(payload.Channel),
			"surface":                 strings.TrimSpace(payload.Surface),
			"thread_id":               strings.TrimSpace(payload.ThreadID),
			"direct_id":               strings.TrimSpace(payload.DirectID),
			"message_id":              strings.TrimSpace(payload.MessageID),
			watchEventsPayloadKindKey: strings.TrimSpace(payload.Kind),
			"direction":               strings.TrimSpace(payload.Direction),
			"work_id":                 strings.TrimSpace(payload.WorkID),
			"work_state":              strings.TrimSpace(payload.WorkState),
			"peer_from":               strings.TrimSpace(payload.PeerFrom),
			"peer_to":                 strings.TrimSpace(payload.PeerTo),
			"trace_id":                strings.TrimSpace(payload.TraceID),
			"causation_id":            strings.TrimSpace(payload.CausationID),
		},
		LedgerKind: string(kind),
	}
}

func watchEventsLoopTerminalEvent(
	payload hookspkg.LoopTerminalPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(hookspkg.HookLoopTerminal),
		Stream:      looppkg.WatchEventsLoopStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		LoopRunID:   strings.TrimSpace(payload.LoopRunID),
		LoopName:    strings.TrimSpace(payload.LoopName),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Channel:     strings.TrimSpace(payload.NetworkChannel),
		Payload: map[string]any{
			"status":                     strings.TrimSpace(payload.Status),
			"to":                         strings.TrimSpace(payload.Status),
			"cause":                      strings.TrimSpace(payload.Cause),
			"reason_code":                strings.TrimSpace(payload.ReasonCode),
			watchEventsPayloadDetailsKey: watchEventsRawJSONValue(payload.Details),
		},
		LedgerKind: "status_changed",
	}
}

func watchEventsLoopNodeTerminalEvent(
	payload hookspkg.LoopNodeTerminalPayload,
	now func() time.Time,
) looppkg.WatchEvent {
	return looppkg.WatchEvent{
		Kind:        string(hookspkg.HookLoopNodeTerminal),
		Stream:      looppkg.WatchEventsLoopStream,
		At:          watchEventsHookAt(payload.Timestamp, now),
		WorkspaceID: strings.TrimSpace(payload.WorkspaceID),
		TaskID:      strings.TrimSpace(payload.TaskID),
		RunID:       strings.TrimSpace(payload.RunID),
		LoopRunID:   strings.TrimSpace(payload.LoopRunID),
		LoopName:    strings.TrimSpace(payload.LoopName),
		SessionID:   strings.TrimSpace(payload.SessionID),
		Channel:     strings.TrimSpace(payload.NetworkChannel),
		Payload: map[string]any{
			watchEventsPayloadNodeIDKey:  strings.TrimSpace(payload.NodeID),
			"generation":                 payload.Generation,
			coordinatorRuntimeTaskIDKey:  strings.TrimSpace(payload.TaskID),
			"task_run_id":                strings.TrimSpace(payload.RunID),
			"status":                     strings.TrimSpace(payload.TaskStatus),
			"run_status":                 strings.TrimSpace(payload.RunStatus),
			watchEventsPayloadErrorKey:   strings.TrimSpace(payload.Error),
			watchEventsPayloadDetailsKey: watchEventsRawJSONValue(payload.Details),
		},
		LedgerKind: watchEventsLoopNodeLedgerKind(payload),
	}
}

func watchEventsLoopNodeLedgerKind(payload hookspkg.LoopNodeTerminalPayload) string {
	if strings.TrimSpace(payload.Error) != "" ||
		taskpkg.ParseRunStatus(payload.RunStatus).Normalize() == taskpkg.TaskRunStatusFailed {
		return "node_failed"
	}
	return "node_succeeded"
}

func watchEventsHookAt(timestamp time.Time, now func() time.Time) string {
	if timestamp.IsZero() {
		if now == nil {
			timestamp = time.Now().UTC()
		} else {
			timestamp = now().UTC()
		}
	}
	return timestamp.UTC().Format(time.RFC3339Nano)
}

func watchEventsOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func watchEventsRawJSONValue(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return string(raw)
	}
	return value
}
