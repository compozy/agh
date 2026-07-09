package loop

import (
	"slices"
	"strings"

	"github.com/compozy/agh/internal/hooks"
)

const (
	watchEventsTaskStream       = "task_events"
	watchEventsLoopStream       = "loop_run_events"
	watchEventsAutomationStream = "automation_runs"
	watchEventsNetworkStream    = "network_timeline_log"
	watchEventsObserveStream    = "event_summaries"
	watchEventsSessionStream    = "session_events"
	watchEventsSessionSeparator = ":"

	loopRunEventLedgerStatusChanged = "status_changed"
	loopRunEventLedgerNodeSucceeded = "node_succeeded"
	loopRunEventLedgerNodeFailed    = "node_failed"

	watchEventsPayloadParentTaskID = "parent_task_id"
	watchEventsPayloadDetails      = "details"
	watchEventsPayloadError        = "error"
	watchEventsPayloadReason       = "reason"

	watchEventsPayloadAgentName   = "agent_name"
	watchEventsPayloadAttempt     = "attempt"
	watchEventsPayloadCausationID = "causation_id"
	watchEventsPayloadDirectID    = "direct_id"
	watchEventsPayloadDirection   = "direction"
	watchEventsPayloadDurationMS  = "duration_ms"
	watchEventsPayloadJobID       = "job_id"
	watchEventsPayloadMessageID   = "message_id"
	watchEventsPayloadPeerFrom    = "peer_from"
	watchEventsPayloadPeerTo      = "peer_to"
	watchEventsPayloadSurface     = "surface"
	watchEventsPayloadThreadID    = "thread_id"
	watchEventsPayloadTraceID     = "trace_id"
	watchEventsPayloadTriggerID   = "trigger_id"
	watchEventsPayloadWillRetry   = "will_retry"
	watchEventsPayloadWorkState   = "work_state"

	watchEventsPayloadCoordinatorSessionID  = "coordinator_session_id"
	watchEventsPayloadCoordinationChannelID = "coordination_channel_id"
	watchEventsPayloadDecisionKind          = "decision_kind"
	watchEventsPayloadDecision              = "decision"
	watchEventsPayloadModel                 = "model"
	watchEventsPayloadProvider              = "provider"
	watchEventsPayloadRecordType            = "record_type"
	watchEventsPayloadSequence              = "sequence"
	watchEventsPayloadStopReason            = "stop_reason"
	watchEventsPayloadTurnID                = "turn_id"
	watchEventsPayloadWorkflowID            = "workflow_id"
)

const (
	// WatchEventsTaskStream is the task_events replay stream name.
	WatchEventsTaskStream = watchEventsTaskStream
	// WatchEventsLoopStream is the loop_run_events replay stream name.
	WatchEventsLoopStream = watchEventsLoopStream
	// WatchEventsAutomationStream is the automation_runs replay stream name.
	WatchEventsAutomationStream = watchEventsAutomationStream
	// WatchEventsNetworkStream is the network_timeline_log replay stream name.
	WatchEventsNetworkStream = watchEventsNetworkStream
	// WatchEventsObserveStream is the event_summaries replay stream name.
	WatchEventsObserveStream = watchEventsObserveStream
	// WatchEventsSessionStream is the per-session event replay stream base name.
	WatchEventsSessionStream = watchEventsSessionStream
)

// WatchEventsContract describes one supported watch-events family row.
type WatchEventsContract struct {
	Kind          hooks.HookEvent
	Stream        string
	LedgerTypes   []string
	PayloadFields []string
	RequiredVars  []string
}

var phaseAWatchEvents = []WatchEventsContract{
	{
		Kind:          hooks.HookTaskStatusChanged,
		Stream:        watchEventsTaskStream,
		LedgerTypes:   []string{string(hooks.HookTaskStatusChanged)},
		PayloadFields: []string{"from_status", "to_status", watchEventsPayloadParentTaskID},
	},
	{
		Kind:        hooks.HookTaskBlocked,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskBlocked)},
		PayloadFields: []string{
			"block_id",
			actionKindMetaKey,
			watchEventsPayloadReason,
			watchEventsPayloadDetails,
			watchEventsPayloadParentTaskID,
		},
	},
	{
		Kind:        hooks.HookTaskUnblocked,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskUnblocked)},
		PayloadFields: []string{
			"block_id",
			actionKindMetaKey,
			watchEventsPayloadReason,
			watchEventsPayloadDetails,
			"cleared_at",
			"clear_note",
			watchEventsPayloadParentTaskID,
		},
	},
	{
		Kind:        hooks.HookTaskNeedsAttention,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskNeedsAttention)},
		PayloadFields: []string{
			watchEventsPayloadReason,
			"note",
			"at",
			watchEventsPayloadParentTaskID,
		},
	},
	{
		Kind:        hooks.HookTaskRecovered,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskRecovered)},
		PayloadFields: []string{
			watchEventsPayloadReason,
			"note",
			"at",
			watchEventsPayloadParentTaskID,
		},
	},
	{
		Kind:        hooks.HookTaskRunCompleted,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskRunCompleted)},
		PayloadFields: []string{
			"previous_run_status",
			"previous_session_id",
			"recovery_action",
			"recovery_reason",
		},
	},
	{
		Kind:        hooks.HookTaskRunFailed,
		Stream:      watchEventsTaskStream,
		LedgerTypes: []string{string(hooks.HookTaskRunFailed)},
		PayloadFields: []string{
			"previous_run_status",
			"previous_session_id",
			"recovery_action",
			"recovery_reason",
			"error",
		},
	},
	{
		Kind:        hooks.HookLoopTerminal,
		Stream:      watchEventsLoopStream,
		LedgerTypes: []string{loopRunEventLedgerStatusChanged},
		PayloadFields: []string{
			reasonMetaStatus,
			"from",
			"to",
			"cause",
			"reason_code",
			watchEventsPayloadDetails,
		},
	},
	{
		Kind:        hooks.HookLoopNodeTerminal,
		Stream:      watchEventsLoopStream,
		LedgerTypes: []string{loopRunEventLedgerNodeSucceeded, loopRunEventLedgerNodeFailed},
		PayloadFields: []string{
			"node_id",
			"generation",
			"item_index",
			"task_id",
			"task_run_id",
			reasonMetaStatus,
			"run_status",
			"error",
			watchEventsPayloadDetails,
			"output_ref",
		},
	},
}

var phaseBWatchEvents = []WatchEventsContract{
	// automation_runs is rowid-cursored and workspace-scoped by joining the run's job/trigger
	// plus loop_run fallback; the run row has no workspace_id column.
	{
		Kind:        hooks.HookAutomationRunCompleted,
		Stream:      watchEventsAutomationStream,
		LedgerTypes: []string{string(hooks.HookAutomationRunCompleted)},
		PayloadFields: []string{
			watchEventsPayloadJobID,
			watchEventsPayloadTriggerID,
			watchEventsPayloadAgentName,
			watchEventsFieldSessionID,
			watchEventsPayloadAttempt,
			watchEventsPayloadDurationMS,
		},
	},
	{
		Kind:        hooks.HookAutomationRunFailed,
		Stream:      watchEventsAutomationStream,
		LedgerTypes: []string{string(hooks.HookAutomationRunFailed)},
		PayloadFields: []string{
			watchEventsPayloadJobID,
			watchEventsPayloadTriggerID,
			watchEventsPayloadAgentName,
			watchEventsFieldSessionID,
			watchEventsPayloadAttempt,
			watchEventsPayloadError,
			watchEventsPayloadWillRetry,
		},
	},
	// network_timeline_log is rowid-cursored and workspace-scoped directly by its workspace_id
	// column; work events reconstruct the co-durable network_work state for the timeline row.
	{
		Kind:        hooks.HookNetworkMessagePersisted,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkMessagePersisted)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadThreadID,
			watchEventsPayloadDirectID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsFieldWorkID,
			watchEventsPayloadWorkState,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
			watchEventsPayloadTraceID,
			watchEventsPayloadCausationID,
		},
	},
	{
		Kind:        hooks.HookNetworkThreadOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkThreadOpened)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadThreadID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
		},
	},
	{
		Kind:        hooks.HookNetworkDirectRoomOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkDirectRoomOpened)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadDirectID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
		},
	},
	{
		Kind:        hooks.HookNetworkWorkOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkOpened)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadThreadID,
			watchEventsPayloadDirectID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsFieldWorkID,
			watchEventsPayloadWorkState,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
			watchEventsPayloadTraceID,
			watchEventsPayloadCausationID,
		},
	},
	{
		Kind:        hooks.HookNetworkWorkTransitioned,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkTransitioned)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadThreadID,
			watchEventsPayloadDirectID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsFieldWorkID,
			watchEventsPayloadWorkState,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
			watchEventsPayloadTraceID,
			watchEventsPayloadCausationID,
		},
	},
	{
		Kind:        hooks.HookNetworkWorkClosed,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkClosed)},
		PayloadFields: []string{
			watchEventsFieldSessionID,
			watchEventsFieldChannel,
			watchEventsPayloadSurface,
			watchEventsPayloadThreadID,
			watchEventsPayloadDirectID,
			watchEventsPayloadMessageID,
			actionKindMetaKey,
			watchEventsPayloadDirection,
			watchEventsFieldWorkID,
			watchEventsPayloadWorkState,
			watchEventsPayloadPeerFrom,
			watchEventsPayloadPeerTo,
			watchEventsPayloadTraceID,
			watchEventsPayloadCausationID,
		},
	},
}

var phaseCWatchEvents = []WatchEventsContract{
	// event_summaries is rowid-cursored and workspace-scoped directly by workspace_id.
	// Phase C uses dedicated coordinator lifecycle rows because hook.dispatch rows only
	// exist when a matching public hook is configured.
	{
		Kind:        hooks.HookCoordinatorSpawned,
		Stream:      watchEventsObserveStream,
		LedgerTypes: []string{string(hooks.HookCoordinatorSpawned)},
		PayloadFields: []string{
			watchEventsPayloadAgentName,
			watchEventsPayloadCoordinatorSessionID,
			watchEventsPayloadCoordinationChannelID,
			watchEventsPayloadProvider,
			watchEventsPayloadModel,
			watchEventsPayloadWorkflowID,
			watchEventsPayloadDecisionKind,
			watchEventsPayloadDecision,
		},
	},
	{
		Kind:        hooks.HookCoordinatorDecision,
		Stream:      watchEventsObserveStream,
		LedgerTypes: []string{string(hooks.HookCoordinatorDecision)},
		PayloadFields: []string{
			watchEventsPayloadAgentName,
			watchEventsPayloadCoordinatorSessionID,
			watchEventsPayloadCoordinationChannelID,
			watchEventsPayloadProvider,
			watchEventsPayloadModel,
			watchEventsPayloadWorkflowID,
			watchEventsPayloadDecisionKind,
			watchEventsPayloadDecision,
		},
	},
	{
		Kind:        hooks.HookCoordinatorStopped,
		Stream:      watchEventsObserveStream,
		LedgerTypes: []string{string(hooks.HookCoordinatorStopped)},
		PayloadFields: []string{
			watchEventsPayloadAgentName,
			watchEventsPayloadCoordinatorSessionID,
			watchEventsPayloadProvider,
			watchEventsPayloadDecisionKind,
			watchEventsPayloadDecision,
			watchEventsPayloadStopReason,
		},
	},
	{
		Kind:        hooks.HookCoordinatorFailed,
		Stream:      watchEventsObserveStream,
		LedgerTypes: []string{string(hooks.HookCoordinatorFailed)},
		PayloadFields: []string{
			watchEventsPayloadAgentName,
			watchEventsPayloadCoordinatorSessionID,
			watchEventsPayloadCoordinationChannelID,
			watchEventsPayloadWorkflowID,
			watchEventsPayloadDecisionKind,
			watchEventsPayloadDecision,
			watchEventsPayloadError,
		},
	},
	// session_events streams are sequence-cursored per session. The linter requires
	// a session_id equality constraint so evaluation never scans all session DBs.
	{
		Kind:        hooks.HookEventPostRecord,
		Stream:      watchEventsSessionStream,
		LedgerTypes: []string{string(hooks.HookEventPostRecord)},
		PayloadFields: []string{
			watchEventsPayloadRecordType,
			watchEventsPayloadSequence,
			watchEventsPayloadTurnID,
			watchEventsPayloadAgentName,
			watchEventsFieldSessionID,
		},
		RequiredVars: []string{watchEventsFieldSessionID},
	},
}

// SupportedWatchEvents returns the closed watch-events registry for shipped phases.
func SupportedWatchEvents() map[hooks.HookEvent]WatchEventsContract {
	rows := make(
		[]WatchEventsContract,
		0,
		len(phaseAWatchEvents)+len(phaseBWatchEvents)+len(phaseCWatchEvents),
	)
	rows = append(rows, phaseAWatchEvents...)
	rows = append(rows, phaseBWatchEvents...)
	rows = append(rows, phaseCWatchEvents...)
	contracts := make(map[hooks.HookEvent]WatchEventsContract, len(rows))
	for _, contract := range rows {
		contracts[contract.Kind] = cloneWatchEventsContract(contract)
	}
	return contracts
}

func cloneWatchEventsContract(contract WatchEventsContract) WatchEventsContract {
	contract.LedgerTypes = slices.Clone(contract.LedgerTypes)
	contract.PayloadFields = slices.Clone(contract.PayloadFields)
	contract.RequiredVars = slices.Clone(contract.RequiredVars)
	return contract
}

func sortedSupportedWatchEventKinds() []string {
	contracts := SupportedWatchEvents()
	kinds := make([]string, 0, len(contracts))
	for kind := range contracts {
		kinds = append(kinds, string(kind))
	}
	slices.Sort(kinds)
	return kinds
}

// WatchEventsSessionStreamForSession returns the cursor stream key for one session.
func WatchEventsSessionStreamForSession(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	return watchEventsSessionStream + watchEventsSessionSeparator + trimmed
}

// WatchEventsSessionIDFromStream extracts the session id from a dynamic session event stream key.
func WatchEventsSessionIDFromStream(stream string) (string, bool) {
	trimmed := strings.TrimSpace(stream)
	prefix := watchEventsSessionStream + watchEventsSessionSeparator
	if !strings.HasPrefix(trimmed, prefix) {
		return "", false
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	return sessionID, sessionID != ""
}

// WatchEventsBaseStream returns the stable registry stream for a cursor stream key.
func WatchEventsBaseStream(stream string) string {
	if _, ok := WatchEventsSessionIDFromStream(stream); ok {
		return watchEventsSessionStream
	}
	return strings.TrimSpace(stream)
}
