package loop

import (
	"slices"

	"github.com/compozy/agh/internal/hooks"
)

const (
	watchEventsTaskStream       = "task_events"
	watchEventsLoopStream       = "loop_run_events"
	watchEventsAutomationStream = "automation_runs"
	watchEventsNetworkStream    = "network_timeline_log"

	loopRunEventLedgerStatusChanged = "status_changed"
	loopRunEventLedgerNodeSucceeded = "node_succeeded"
	loopRunEventLedgerNodeFailed    = "node_failed"

	watchEventsPayloadParentTaskID = "parent_task_id"
	watchEventsPayloadDetails      = "details"
	watchEventsPayloadError        = "error"
	watchEventsPayloadReason       = "reason"
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
			"job_id",
			"trigger_id",
			"agent_name",
			"session_id",
			"attempt",
			"duration_ms",
		},
	},
	{
		Kind:        hooks.HookAutomationRunFailed,
		Stream:      watchEventsAutomationStream,
		LedgerTypes: []string{string(hooks.HookAutomationRunFailed)},
		PayloadFields: []string{
			"job_id",
			"trigger_id",
			"agent_name",
			"session_id",
			"attempt",
			watchEventsPayloadError,
			"will_retry",
		},
	},
	// network_timeline_log is rowid-cursored and workspace-scoped directly by its workspace_id
	// column; work events reconstruct the co-durable network_work state for the timeline row.
	{
		Kind:        hooks.HookNetworkMessagePersisted,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkMessagePersisted)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"thread_id",
			"direct_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"work_id",
			"work_state",
			"peer_from",
			"peer_to",
			"trace_id",
			"causation_id",
		},
	},
	{
		Kind:        hooks.HookNetworkThreadOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkThreadOpened)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"thread_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"peer_from",
			"peer_to",
		},
	},
	{
		Kind:        hooks.HookNetworkDirectRoomOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkDirectRoomOpened)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"direct_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"peer_from",
			"peer_to",
		},
	},
	{
		Kind:        hooks.HookNetworkWorkOpened,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkOpened)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"thread_id",
			"direct_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"work_id",
			"work_state",
			"peer_from",
			"peer_to",
			"trace_id",
			"causation_id",
		},
	},
	{
		Kind:        hooks.HookNetworkWorkTransitioned,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkTransitioned)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"thread_id",
			"direct_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"work_id",
			"work_state",
			"peer_from",
			"peer_to",
			"trace_id",
			"causation_id",
		},
	},
	{
		Kind:        hooks.HookNetworkWorkClosed,
		Stream:      watchEventsNetworkStream,
		LedgerTypes: []string{string(hooks.HookNetworkWorkClosed)},
		PayloadFields: []string{
			"session_id",
			"channel",
			"surface",
			"thread_id",
			"direct_id",
			"message_id",
			actionKindMetaKey,
			"direction",
			"work_id",
			"work_state",
			"peer_from",
			"peer_to",
			"trace_id",
			"causation_id",
		},
	},
}

// SupportedWatchEvents returns the closed watch-events registry for shipped phases.
func SupportedWatchEvents() map[hooks.HookEvent]WatchEventsContract {
	rows := make([]WatchEventsContract, 0, len(phaseAWatchEvents)+len(phaseBWatchEvents))
	rows = append(rows, phaseAWatchEvents...)
	rows = append(rows, phaseBWatchEvents...)
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
