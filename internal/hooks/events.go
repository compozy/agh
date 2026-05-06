package hooks

import "fmt"

// HookEventFamily groups hook events into the documented taxonomy families.
type HookEventFamily string

const (
	HookEventFamilySession     HookEventFamily = "session"
	HookEventFamilySandbox     HookEventFamily = "sandbox"
	HookEventFamilyInput       HookEventFamily = "input"
	HookEventFamilyPrompt      HookEventFamily = "prompt"
	HookEventFamilyEvent       HookEventFamily = "event"
	HookEventFamilyAutomation  HookEventFamily = "automation"
	HookEventFamilyAgent       HookEventFamily = "agent"
	HookEventFamilyTurn        HookEventFamily = "turn"
	HookEventFamilyMessage     HookEventFamily = "message"
	HookEventFamilyTool        HookEventFamily = "tool"
	HookEventFamilyPermission  HookEventFamily = "permission"
	HookEventFamilyContext     HookEventFamily = "context"
	HookEventFamilyCoordinator HookEventFamily = "coordinator"
	HookEventFamilyTaskRun     HookEventFamily = "task.run"
	HookEventFamilySpawn       HookEventFamily = "spawn"
	HookEventFamilyNetwork     HookEventFamily = "network"
)

// Validate ensures the event family is part of the supported taxonomy.
func (f HookEventFamily) Validate() error {
	switch f {
	case HookEventFamilySession,
		HookEventFamilySandbox,
		HookEventFamilyInput,
		HookEventFamilyPrompt,
		HookEventFamilyEvent,
		HookEventFamilyAutomation,
		HookEventFamilyAgent,
		HookEventFamilyTurn,
		HookEventFamilyMessage,
		HookEventFamilyTool,
		HookEventFamilyPermission,
		HookEventFamilyContext,
		HookEventFamilyCoordinator,
		HookEventFamilyTaskRun,
		HookEventFamilySpawn,
		HookEventFamilyNetwork:
		return nil
	default:
		return fmt.Errorf("hooks: invalid hook event family %q", f)
	}
}

// HookEvent identifies when a hook fires.
type HookEvent string

const (
	HookSessionPreCreate        HookEvent = "session.pre_create"
	HookSessionPostCreate       HookEvent = "session.post_create"
	HookSessionPreResume        HookEvent = "session.pre_resume"
	HookSessionPostResume       HookEvent = "session.post_resume"
	HookSessionPreStop          HookEvent = "session.pre_stop"
	HookSessionPostStop         HookEvent = "session.post_stop"
	HookSessionMessagePersisted HookEvent = "session.message_persisted"

	HookSandboxPrepare    HookEvent = "sandbox.prepare"
	HookSandboxReady      HookEvent = "sandbox.ready"
	HookSandboxSyncBefore HookEvent = "sandbox.sync.before"
	HookSandboxSyncAfter  HookEvent = "sandbox.sync.after"
	HookSandboxStop       HookEvent = "sandbox.stop"

	HookInputPreSubmit HookEvent = "input.pre_submit"

	HookPromptPostAssemble HookEvent = "prompt.post_assemble"

	HookEventPreRecord  HookEvent = "event.pre_record"
	HookEventPostRecord HookEvent = "event.post_record"

	HookAutomationJobPreFire      HookEvent = "automation.job.pre_fire"
	HookAutomationJobPostFire     HookEvent = "automation.job.post_fire"
	HookAutomationTriggerPreFire  HookEvent = "automation.trigger.pre_fire"
	HookAutomationTriggerPostFire HookEvent = "automation.trigger.post_fire"
	HookAutomationRunCompleted    HookEvent = "automation.run.completed"
	HookAutomationRunFailed       HookEvent = "automation.run.failed"

	HookAgentPreStart                HookEvent = "agent.pre_start"
	HookAgentSpawned                 HookEvent = "agent.spawned"
	HookAgentCrashed                 HookEvent = "agent.crashed"
	HookAgentStopped                 HookEvent = "agent.stopped"
	HookAgentSoulSnapshotResolved    HookEvent = "agent.soul.snapshot.resolved"
	HookAgentSoulMutationAfter       HookEvent = "agent.soul.mutation.after"
	HookAgentHeartbeatPolicyResolved HookEvent = "agent.heartbeat.policy.resolved"
	HookAgentHeartbeatWakeBefore     HookEvent = "agent.heartbeat.wake.before"
	HookAgentHeartbeatWakeAfter      HookEvent = "agent.heartbeat.wake.after"
	HookSessionHealthUpdateAfter     HookEvent = "session.health.update.after"

	HookTurnStart HookEvent = "turn.start"
	HookTurnEnd   HookEvent = "turn.end"

	HookMessageStart HookEvent = "message.start"
	HookMessageDelta HookEvent = "message.delta"
	HookMessageEnd   HookEvent = "message.end"

	HookToolPreCall   HookEvent = "tool.pre_call"
	HookToolPostCall  HookEvent = "tool.post_call"
	HookToolPostError HookEvent = "tool.post_error"

	HookPermissionRequest  HookEvent = "permission.request"
	HookPermissionResolved HookEvent = "permission.resolved"
	HookPermissionDenied   HookEvent = "permission.denied"

	HookContextPreCompact  HookEvent = "context.pre_compact"
	HookContextPostCompact HookEvent = "context.post_compact"

	HookCoordinatorPreSpawn HookEvent = "coordinator.pre_spawn"
	HookCoordinatorSpawned  HookEvent = "coordinator.spawned"
	HookCoordinatorDecision HookEvent = "coordinator.decision"
	HookCoordinatorStopped  HookEvent = "coordinator.stopped"
	HookCoordinatorFailed   HookEvent = "coordinator.failed"

	HookTaskRunEnqueued       HookEvent = "task.run.enqueued"
	HookTaskRunPreClaim       HookEvent = "task.run.pre_claim"
	HookTaskRunPostClaim      HookEvent = "task.run.post_claim"
	HookTaskRunLeaseExtended  HookEvent = "task.run.lease_extended"
	HookTaskRunLeaseExpired   HookEvent = "task.run.lease_expired"
	HookTaskRunLeaseRecovered HookEvent = "task.run.lease_recovered"
	HookTaskRunReleased       HookEvent = "task.run.released"
	HookTaskRunCompleted      HookEvent = "task.run.completed"
	HookTaskRunFailed         HookEvent = "task.run.failed"

	HookSpawnPreCreate     HookEvent = "spawn.pre_create"
	HookSpawnCreated       HookEvent = "spawn.created"
	HookSpawnParentStopped HookEvent = "spawn.parent_stopped"
	HookSpawnTTLExpired    HookEvent = "spawn.ttl_expired"
	HookSpawnReaped        HookEvent = "spawn.reaped"

	HookNetworkThreadOpened     HookEvent = "network.thread.opened"
	HookNetworkDirectRoomOpened HookEvent = "network.direct_room.opened"
	HookNetworkMessagePersisted HookEvent = "network.message.persisted"
	HookNetworkWorkOpened       HookEvent = "network.work.opened"
	HookNetworkWorkTransitioned HookEvent = "network.work.transitioned"
	HookNetworkWorkClosed       HookEvent = "network.work.closed"
)

type hookEventSpec struct {
	family       HookEventFamily
	syncEligible bool
}

var hookEventSpecs = map[HookEvent]hookEventSpec{
	HookSessionPreCreate:  {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostCreate: {family: HookEventFamilySession, syncEligible: true},
	HookSessionPreResume:  {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostResume: {family: HookEventFamilySession, syncEligible: true},
	HookSessionPreStop:    {family: HookEventFamilySession, syncEligible: true},
	HookSessionPostStop:   {family: HookEventFamilySession, syncEligible: true},
	HookSessionMessagePersisted: {
		family:       HookEventFamilySession,
		syncEligible: false,
	},
	HookSandboxPrepare: {
		family:       HookEventFamilySandbox,
		syncEligible: true,
	},
	HookSandboxReady: {
		family:       HookEventFamilySandbox,
		syncEligible: false,
	},
	HookSandboxSyncBefore: {
		family:       HookEventFamilySandbox,
		syncEligible: true,
	},
	HookSandboxSyncAfter: {
		family:       HookEventFamilySandbox,
		syncEligible: false,
	},
	HookSandboxStop: {
		family:       HookEventFamilySandbox,
		syncEligible: true,
	},
	HookInputPreSubmit: {family: HookEventFamilyInput, syncEligible: true},
	HookPromptPostAssemble: {
		family:       HookEventFamilyPrompt,
		syncEligible: true,
	},
	HookEventPreRecord:  {family: HookEventFamilyEvent, syncEligible: false},
	HookEventPostRecord: {family: HookEventFamilyEvent, syncEligible: false},
	HookAutomationJobPreFire: {
		family:       HookEventFamilyAutomation,
		syncEligible: true,
	},
	HookAutomationJobPostFire: {
		family:       HookEventFamilyAutomation,
		syncEligible: false,
	},
	HookAutomationTriggerPreFire: {
		family:       HookEventFamilyAutomation,
		syncEligible: true,
	},
	HookAutomationTriggerPostFire: {
		family:       HookEventFamilyAutomation,
		syncEligible: false,
	},
	HookAutomationRunCompleted: {
		family:       HookEventFamilyAutomation,
		syncEligible: false,
	},
	HookAutomationRunFailed: {
		family:       HookEventFamilyAutomation,
		syncEligible: false,
	},
	HookAgentPreStart: {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentSpawned:  {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentCrashed:  {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentStopped:  {family: HookEventFamilyAgent, syncEligible: true},
	HookAgentSoulSnapshotResolved: {
		family:       HookEventFamilyAgent,
		syncEligible: false,
	},
	HookAgentSoulMutationAfter: {
		family:       HookEventFamilyAgent,
		syncEligible: false,
	},
	HookAgentHeartbeatPolicyResolved: {
		family:       HookEventFamilyAgent,
		syncEligible: false,
	},
	HookAgentHeartbeatWakeBefore: {
		family:       HookEventFamilyAgent,
		syncEligible: true,
	},
	HookAgentHeartbeatWakeAfter: {
		family:       HookEventFamilyAgent,
		syncEligible: false,
	},
	HookSessionHealthUpdateAfter: {
		family:       HookEventFamilySession,
		syncEligible: false,
	},
	HookTurnStart:     {family: HookEventFamilyTurn, syncEligible: true},
	HookTurnEnd:       {family: HookEventFamilyTurn, syncEligible: true},
	HookMessageStart:  {family: HookEventFamilyMessage, syncEligible: true},
	HookMessageDelta:  {family: HookEventFamilyMessage, syncEligible: false},
	HookMessageEnd:    {family: HookEventFamilyMessage, syncEligible: true},
	HookToolPreCall:   {family: HookEventFamilyTool, syncEligible: true},
	HookToolPostCall:  {family: HookEventFamilyTool, syncEligible: true},
	HookToolPostError: {family: HookEventFamilyTool, syncEligible: true},
	HookPermissionRequest: {
		family:       HookEventFamilyPermission,
		syncEligible: true,
	},
	HookPermissionResolved: {
		family:       HookEventFamilyPermission,
		syncEligible: false,
	},
	HookPermissionDenied: {
		family:       HookEventFamilyPermission,
		syncEligible: false,
	},
	HookContextPreCompact:  {family: HookEventFamilyContext, syncEligible: true},
	HookContextPostCompact: {family: HookEventFamilyContext, syncEligible: true},
	HookCoordinatorPreSpawn: {
		family:       HookEventFamilyCoordinator,
		syncEligible: true,
	},
	HookCoordinatorSpawned: {
		family:       HookEventFamilyCoordinator,
		syncEligible: true,
	},
	HookCoordinatorDecision: {
		family:       HookEventFamilyCoordinator,
		syncEligible: true,
	},
	HookCoordinatorStopped: {
		family:       HookEventFamilyCoordinator,
		syncEligible: true,
	},
	HookCoordinatorFailed: {
		family:       HookEventFamilyCoordinator,
		syncEligible: true,
	},
	HookTaskRunEnqueued: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunPreClaim: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunPostClaim: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunLeaseExtended: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunLeaseExpired: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunLeaseRecovered: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunReleased: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunCompleted: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookTaskRunFailed: {
		family:       HookEventFamilyTaskRun,
		syncEligible: true,
	},
	HookSpawnPreCreate: {
		family:       HookEventFamilySpawn,
		syncEligible: true,
	},
	HookSpawnCreated: {
		family:       HookEventFamilySpawn,
		syncEligible: true,
	},
	HookSpawnParentStopped: {
		family:       HookEventFamilySpawn,
		syncEligible: true,
	},
	HookSpawnTTLExpired: {
		family:       HookEventFamilySpawn,
		syncEligible: true,
	},
	HookSpawnReaped: {
		family:       HookEventFamilySpawn,
		syncEligible: true,
	},
	HookNetworkThreadOpened: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
	HookNetworkDirectRoomOpened: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
	HookNetworkMessagePersisted: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
	HookNetworkWorkOpened: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
	HookNetworkWorkTransitioned: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
	HookNetworkWorkClosed: {
		family:       HookEventFamilyNetwork,
		syncEligible: false,
	},
}

var allHookEvents = []HookEvent{
	HookSessionPreCreate,
	HookSessionPostCreate,
	HookSessionPreResume,
	HookSessionPostResume,
	HookSessionPreStop,
	HookSessionPostStop,
	HookSessionMessagePersisted,
	HookSandboxPrepare,
	HookSandboxReady,
	HookSandboxSyncBefore,
	HookSandboxSyncAfter,
	HookSandboxStop,
	HookInputPreSubmit,
	HookPromptPostAssemble,
	HookEventPreRecord,
	HookEventPostRecord,
	HookAutomationJobPreFire,
	HookAutomationJobPostFire,
	HookAutomationTriggerPreFire,
	HookAutomationTriggerPostFire,
	HookAutomationRunCompleted,
	HookAutomationRunFailed,
	HookAgentPreStart,
	HookAgentSpawned,
	HookAgentCrashed,
	HookAgentStopped,
	HookAgentSoulSnapshotResolved,
	HookAgentSoulMutationAfter,
	HookAgentHeartbeatPolicyResolved,
	HookAgentHeartbeatWakeBefore,
	HookAgentHeartbeatWakeAfter,
	HookSessionHealthUpdateAfter,
	HookTurnStart,
	HookTurnEnd,
	HookMessageStart,
	HookMessageDelta,
	HookMessageEnd,
	HookToolPreCall,
	HookToolPostCall,
	HookToolPostError,
	HookPermissionRequest,
	HookPermissionResolved,
	HookPermissionDenied,
	HookContextPreCompact,
	HookContextPostCompact,
	HookCoordinatorPreSpawn,
	HookCoordinatorSpawned,
	HookCoordinatorDecision,
	HookCoordinatorStopped,
	HookCoordinatorFailed,
	HookTaskRunEnqueued,
	HookTaskRunPreClaim,
	HookTaskRunPostClaim,
	HookTaskRunLeaseExtended,
	HookTaskRunLeaseExpired,
	HookTaskRunLeaseRecovered,
	HookTaskRunReleased,
	HookTaskRunCompleted,
	HookTaskRunFailed,
	HookSpawnPreCreate,
	HookSpawnCreated,
	HookSpawnParentStopped,
	HookSpawnTTLExpired,
	HookSpawnReaped,
	HookNetworkThreadOpened,
	HookNetworkDirectRoomOpened,
	HookNetworkMessagePersisted,
	HookNetworkWorkOpened,
	HookNetworkWorkTransitioned,
	HookNetworkWorkClosed,
}

var _ = func() bool {
	if err := validateHookEventSpecsConsistency(); err != nil {
		panic(err)
	}
	return true
}()

// AllHookEvents returns the full taxonomy in deterministic order.
func AllHookEvents() []HookEvent {
	events := make([]HookEvent, len(allHookEvents))
	copy(events, allHookEvents)
	return events
}

// String returns the literal hook event value.
func (e HookEvent) String() string {
	return string(e)
}

// Family reports the taxonomy family for the event.
func (e HookEvent) Family() HookEventFamily {
	spec, ok := hookEventSpecs[e]
	if !ok {
		return ""
	}
	return spec.family
}

// SyncEligible reports whether the event accepts sync hooks.
func (e HookEvent) SyncEligible() bool {
	spec, ok := hookEventSpecs[e]
	return ok && spec.syncEligible
}

// Validate ensures the event is part of the supported taxonomy.
func (e HookEvent) Validate() error {
	if _, ok := hookEventSpecs[e]; !ok {
		return fmt.Errorf("hooks: invalid hook event %q", e)
	}
	return nil
}

func validateHookEventSpecsConsistency() error {
	eventsFromList := make(map[HookEvent]struct{}, len(allHookEvents))
	for _, event := range allHookEvents {
		eventsFromList[event] = struct{}{}
		if _, ok := hookEventSpecs[event]; !ok {
			return fmt.Errorf("hooks: event %q exists in allHookEvents but is missing from hookEventSpecs", event)
		}
	}
	for event := range hookEventSpecs {
		if _, ok := eventsFromList[event]; !ok {
			return fmt.Errorf("hooks: event %q exists in hookEventSpecs but is missing from allHookEvents", event)
		}
	}
	return nil
}
