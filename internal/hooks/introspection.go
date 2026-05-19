package hooks

import "time"

const (
	introspectionAuthoredContextObservationPatchValue = "AuthoredContextObservationPatch"
	introspectionAutomationFirePatchValue             = "AutomationFirePatch"
	introspectionAutomationObservationPatchValue      = "AutomationObservationPatch"
	introspectionCoordinatorObservationPatchValue     = "CoordinatorObservationPatch"
	introspectionNetworkObservationPatchValue         = "NetworkObservationPatch"
	introspectionSpawnObservationPatchValue           = "SpawnObservationPatch"
	introspectionTaskRunObservationPatchValue         = "TaskRunObservationPatch"
)

// CatalogFilter narrows the resolved hook catalog for one workspace/agent view.
type CatalogFilter struct {
	WorkspaceID   string
	WorkspaceRoot string
	AgentName     string
	Event         HookEvent
	Source        *HookSource
	Mode          HookMode
}

// CatalogEntry describes one resolved hook in pipeline order.
type CatalogEntry struct {
	Order        int
	Name         string
	Event        HookEvent
	Source       HookSource
	SkillSource  HookSkillSource
	Mode         HookMode
	Required     bool
	Priority     int32
	Timeout      time.Duration
	ExecutorKind HookExecutorKind
	Matcher      HookMatcher
	Metadata     map[string]string
}

// EventFilter narrows the supported hook taxonomy for introspection APIs.
type EventFilter struct {
	Family   HookEventFamily
	SyncOnly bool
}

// EventDescriptor describes one supported hook event for introspection APIs.
type EventDescriptor struct {
	Event         HookEvent
	Family        HookEventFamily
	SyncEligible  bool
	PayloadSchema string
	PatchSchema   string
}

var hookEventDescriptors = map[HookEvent]EventDescriptor{
	HookSessionPreCreate: {
		Event:         HookSessionPreCreate,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPreCreatePayload",
		PatchSchema:   "SessionCreatePatch",
	},
	HookSessionPostCreate: {
		Event:         HookSessionPostCreate,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPostCreatePayload",
		PatchSchema:   "SessionPostCreatePatch",
	},
	HookSessionPreResume: {
		Event:         HookSessionPreResume,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPreResumePayload",
		PatchSchema:   "SessionPreResumePatch",
	},
	HookSessionPostResume: {
		Event:         HookSessionPostResume,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPostResumePayload",
		PatchSchema:   "SessionPostResumePatch",
	},
	HookSessionPreStop: {
		Event:         HookSessionPreStop,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPreStopPayload",
		PatchSchema:   "SessionPreStopPatch",
	},
	HookSessionPostStop: {
		Event:         HookSessionPostStop,
		Family:        HookEventFamilySession,
		SyncEligible:  true,
		PayloadSchema: "SessionPostStopPayload",
		PatchSchema:   "SessionPostStopPatch",
	},
	HookSessionMessagePersisted: {
		Event:         HookSessionMessagePersisted,
		Family:        HookEventFamilySession,
		SyncEligible:  false,
		PayloadSchema: "SessionMessagePersistedPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookSandboxPrepare: {
		Event:         HookSandboxPrepare,
		Family:        HookEventFamilySandbox,
		SyncEligible:  true,
		PayloadSchema: "SandboxPreparePayload",
		PatchSchema:   "SandboxPreparePatch",
	},
	HookSandboxReady: {
		Event:         HookSandboxReady,
		Family:        HookEventFamilySandbox,
		SyncEligible:  false,
		PayloadSchema: "SandboxReadyPayload",
		PatchSchema:   "SandboxReadyPatch",
	},
	HookSandboxSyncBefore: {
		Event:         HookSandboxSyncBefore,
		Family:        HookEventFamilySandbox,
		SyncEligible:  true,
		PayloadSchema: "SandboxSyncBeforePayload",
		PatchSchema:   "SandboxSyncBeforePatch",
	},
	HookSandboxSyncAfter: {
		Event:         HookSandboxSyncAfter,
		Family:        HookEventFamilySandbox,
		SyncEligible:  false,
		PayloadSchema: "SandboxSyncAfterPayload",
		PatchSchema:   "SandboxSyncAfterPatch",
	},
	HookSandboxStop: {
		Event:         HookSandboxStop,
		Family:        HookEventFamilySandbox,
		SyncEligible:  true,
		PayloadSchema: "SandboxStopPayload",
		PatchSchema:   "SandboxStopPatch",
	},
	HookInputPreSubmit: {
		Event:         HookInputPreSubmit,
		Family:        HookEventFamilyInput,
		SyncEligible:  true,
		PayloadSchema: "InputPreSubmitPayload",
		PatchSchema:   "InputPreSubmitPatch",
	},
	HookPromptPostAssemble: {
		Event:         HookPromptPostAssemble,
		Family:        HookEventFamilyPrompt,
		SyncEligible:  true,
		PayloadSchema: "PromptPayload",
		PatchSchema:   "PromptPatch",
	},
	HookEventPreRecord: {
		Event:         HookEventPreRecord,
		Family:        HookEventFamilyEvent,
		SyncEligible:  false,
		PayloadSchema: "EventPreRecordPayload",
		PatchSchema:   "EventPreRecordPatch",
	},
	HookEventPostRecord: {
		Event:         HookEventPostRecord,
		Family:        HookEventFamilyEvent,
		SyncEligible:  false,
		PayloadSchema: "EventPostRecordPayload",
		PatchSchema:   "EventPostRecordPatch",
	},
	HookAutomationJobPreFire: {
		Event:         HookAutomationJobPreFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  true,
		PayloadSchema: "AutomationJobPreFirePayload",
		PatchSchema:   introspectionAutomationFirePatchValue,
	},
	HookAutomationJobPostFire: {
		Event:         HookAutomationJobPostFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationJobPostFirePayload",
		PatchSchema:   introspectionAutomationObservationPatchValue,
	},
	HookAutomationTriggerPreFire: {
		Event:         HookAutomationTriggerPreFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  true,
		PayloadSchema: "AutomationTriggerPreFirePayload",
		PatchSchema:   introspectionAutomationFirePatchValue,
	},
	HookAutomationTriggerPostFire: {
		Event:         HookAutomationTriggerPostFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationTriggerPostFirePayload",
		PatchSchema:   introspectionAutomationObservationPatchValue,
	},
	HookAutomationRunCompleted: {
		Event:         HookAutomationRunCompleted,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationRunCompletedPayload",
		PatchSchema:   introspectionAutomationObservationPatchValue,
	},
	HookAutomationRunFailed: {
		Event:         HookAutomationRunFailed,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationRunFailedPayload",
		PatchSchema:   introspectionAutomationObservationPatchValue,
	},
	HookAgentPreStart: {
		Event:         HookAgentPreStart,
		Family:        HookEventFamilyAgent,
		SyncEligible:  true,
		PayloadSchema: "AgentPreStartPayload",
		PatchSchema:   "AgentStartPatch",
	},
	HookAgentSpawned: {
		Event:         HookAgentSpawned,
		Family:        HookEventFamilyAgent,
		SyncEligible:  true,
		PayloadSchema: "AgentSpawnedPayload",
		PatchSchema:   "AgentSpawnedPatch",
	},
	HookAgentCrashed: {
		Event:         HookAgentCrashed,
		Family:        HookEventFamilyAgent,
		SyncEligible:  true,
		PayloadSchema: "AgentCrashedPayload",
		PatchSchema:   "AgentCrashedPatch",
	},
	HookAgentStopped: {
		Event:         HookAgentStopped,
		Family:        HookEventFamilyAgent,
		SyncEligible:  true,
		PayloadSchema: "AgentStoppedPayload",
		PatchSchema:   "AgentStoppedPatch",
	},
	HookAgentSoulSnapshotResolved: {
		Event:         HookAgentSoulSnapshotResolved,
		Family:        HookEventFamilyAgent,
		SyncEligible:  false,
		PayloadSchema: "AgentSoulSnapshotResolvedPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookAgentSoulMutationAfter: {
		Event:         HookAgentSoulMutationAfter,
		Family:        HookEventFamilyAgent,
		SyncEligible:  false,
		PayloadSchema: "AgentSoulMutationAfterPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookAgentHeartbeatPolicyResolved: {
		Event:         HookAgentHeartbeatPolicyResolved,
		Family:        HookEventFamilyAgent,
		SyncEligible:  false,
		PayloadSchema: "AgentHeartbeatPolicyResolvedPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookAgentHeartbeatWakeBefore: {
		Event:         HookAgentHeartbeatWakeBefore,
		Family:        HookEventFamilyAgent,
		SyncEligible:  true,
		PayloadSchema: "AgentHeartbeatWakeBeforePayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookAgentHeartbeatWakeAfter: {
		Event:         HookAgentHeartbeatWakeAfter,
		Family:        HookEventFamilyAgent,
		SyncEligible:  false,
		PayloadSchema: "AgentHeartbeatWakeAfterPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookSessionHealthUpdateAfter: {
		Event:         HookSessionHealthUpdateAfter,
		Family:        HookEventFamilySession,
		SyncEligible:  false,
		PayloadSchema: "SessionHealthUpdateAfterPayload",
		PatchSchema:   introspectionAuthoredContextObservationPatchValue,
	},
	HookTurnStart: {
		Event:         HookTurnStart,
		Family:        HookEventFamilyTurn,
		SyncEligible:  true,
		PayloadSchema: "TurnStartPayload",
		PatchSchema:   "TurnStartPatch",
	},
	HookTurnEnd: {
		Event:         HookTurnEnd,
		Family:        HookEventFamilyTurn,
		SyncEligible:  true,
		PayloadSchema: "TurnEndPayload",
		PatchSchema:   "TurnEndPatch",
	},
	HookMessageStart: {
		Event:         HookMessageStart,
		Family:        HookEventFamilyMessage,
		SyncEligible:  true,
		PayloadSchema: "MessageStartPayload",
		PatchSchema:   "MessageStartPatch",
	},
	HookMessageDelta: {
		Event:         HookMessageDelta,
		Family:        HookEventFamilyMessage,
		SyncEligible:  false,
		PayloadSchema: "MessageDeltaPayload",
		PatchSchema:   "MessageDeltaPatch",
	},
	HookMessageEnd: {
		Event:         HookMessageEnd,
		Family:        HookEventFamilyMessage,
		SyncEligible:  true,
		PayloadSchema: "MessageEndPayload",
		PatchSchema:   "MessageEndPatch",
	},
	HookToolPreCall: {
		Event:         HookToolPreCall,
		Family:        HookEventFamilyTool,
		SyncEligible:  true,
		PayloadSchema: "ToolPreCallPayload",
		PatchSchema:   "ToolCallPatch",
	},
	HookToolPostCall: {
		Event:         HookToolPostCall,
		Family:        HookEventFamilyTool,
		SyncEligible:  true,
		PayloadSchema: "ToolPostCallPayload",
		PatchSchema:   "ToolResultPatch",
	},
	HookToolPostError: {
		Event:         HookToolPostError,
		Family:        HookEventFamilyTool,
		SyncEligible:  true,
		PayloadSchema: "ToolPostErrorPayload",
		PatchSchema:   "ToolPostErrorPatch",
	},
	HookPermissionRequest: {
		Event:         HookPermissionRequest,
		Family:        HookEventFamilyPermission,
		SyncEligible:  true,
		PayloadSchema: "PermissionRequestPayload",
		PatchSchema:   "PermissionRequestPatch",
	},
	HookPermissionResolved: {
		Event:         HookPermissionResolved,
		Family:        HookEventFamilyPermission,
		SyncEligible:  false,
		PayloadSchema: "PermissionResolvedPayload",
		PatchSchema:   "PermissionResolvedPatch",
	},
	HookPermissionDenied: {
		Event:         HookPermissionDenied,
		Family:        HookEventFamilyPermission,
		SyncEligible:  false,
		PayloadSchema: "PermissionDeniedPayload",
		PatchSchema:   "PermissionDeniedPatch",
	},
	HookContextPreCompact: {
		Event:         HookContextPreCompact,
		Family:        HookEventFamilyContext,
		SyncEligible:  true,
		PayloadSchema: "ContextPreCompactPayload",
		PatchSchema:   "ContextPreCompactPatch",
	},
	HookContextPostCompact: {
		Event:         HookContextPostCompact,
		Family:        HookEventFamilyContext,
		SyncEligible:  true,
		PayloadSchema: "ContextPostCompactPayload",
		PatchSchema:   "ContextPostCompactPatch",
	},
	HookCoordinatorPreSpawn: {
		Event:         HookCoordinatorPreSpawn,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorPreSpawnPayload",
		PatchSchema:   "CoordinatorSpawnPatch",
	},
	HookCoordinatorSpawned: {
		Event:         HookCoordinatorSpawned,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorSpawnedPayload",
		PatchSchema:   introspectionCoordinatorObservationPatchValue,
	},
	HookCoordinatorDecision: {
		Event:         HookCoordinatorDecision,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorDecisionPayload",
		PatchSchema:   introspectionCoordinatorObservationPatchValue,
	},
	HookCoordinatorStopped: {
		Event:         HookCoordinatorStopped,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorStoppedPayload",
		PatchSchema:   introspectionCoordinatorObservationPatchValue,
	},
	HookCoordinatorFailed: {
		Event:         HookCoordinatorFailed,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorFailedPayload",
		PatchSchema:   introspectionCoordinatorObservationPatchValue,
	},
	HookTaskRunEnqueued: {
		Event:         HookTaskRunEnqueued,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunEnqueuedPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunPreClaim: {
		Event:         HookTaskRunPreClaim,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunPreClaimPayload",
		PatchSchema:   "TaskRunPreClaimPatch",
	},
	HookTaskRunPostClaim: {
		Event:         HookTaskRunPostClaim,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunPostClaimPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunLeaseExtended: {
		Event:         HookTaskRunLeaseExtended,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseExtendedPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunLeaseExpired: {
		Event:         HookTaskRunLeaseExpired,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseExpiredPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunLeaseRecovered: {
		Event:         HookTaskRunLeaseRecovered,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseRecoveredPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunReleased: {
		Event:         HookTaskRunReleased,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunReleasedPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunCompleted: {
		Event:         HookTaskRunCompleted,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunCompletedPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookTaskRunFailed: {
		Event:         HookTaskRunFailed,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunFailedPayload",
		PatchSchema:   introspectionTaskRunObservationPatchValue,
	},
	HookSpawnPreCreate: {
		Event:         HookSpawnPreCreate,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnPreCreatePayload",
		PatchSchema:   "SpawnCreatePatch",
	},
	HookSpawnCreated: {
		Event:         HookSpawnCreated,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnCreatedPayload",
		PatchSchema:   introspectionSpawnObservationPatchValue,
	},
	HookSpawnParentStopped: {
		Event:         HookSpawnParentStopped,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnParentStoppedPayload",
		PatchSchema:   introspectionSpawnObservationPatchValue,
	},
	HookSpawnTTLExpired: {
		Event:         HookSpawnTTLExpired,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnTTLExpiredPayload",
		PatchSchema:   introspectionSpawnObservationPatchValue,
	},
	HookSpawnReaped: {
		Event:         HookSpawnReaped,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnReapedPayload",
		PatchSchema:   introspectionSpawnObservationPatchValue,
	},
	HookNetworkThreadOpened: {
		Event:         HookNetworkThreadOpened,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkThreadOpenedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
	HookNetworkDirectRoomOpened: {
		Event:         HookNetworkDirectRoomOpened,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkDirectRoomOpenedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
	HookNetworkMessagePersisted: {
		Event:         HookNetworkMessagePersisted,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkMessagePersistedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
	HookNetworkWorkOpened: {
		Event:         HookNetworkWorkOpened,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkWorkOpenedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
	HookNetworkWorkTransitioned: {
		Event:         HookNetworkWorkTransitioned,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkWorkTransitionedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
	HookNetworkWorkClosed: {
		Event:         HookNetworkWorkClosed,
		Family:        HookEventFamilyNetwork,
		SyncEligible:  false,
		PayloadSchema: "NetworkWorkClosedPayload",
		PatchSchema:   introspectionNetworkObservationPatchValue,
	},
}

// Catalog returns the currently resolved hook catalog in deterministic pipeline order.
func (h *Hooks) Catalog(filter CatalogFilter) ([]CatalogEntry, error) {
	if h == nil {
		return nil, nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]CatalogEntry, 0)
	for _, event := range AllHookEvents() {
		order := 0
		for _, hook := range h.snapshot[event] {
			if !catalogHookMatchesFilter(hook, filter) {
				continue
			}
			executorKind := HookExecutorKind("")
			if hook.Executor != nil {
				executorKind = hook.Executor.Kind()
			}
			order++
			entries = append(entries, CatalogEntry{
				Order:        order,
				Name:         hook.Name,
				Event:        hook.Event,
				Source:       hook.Source,
				SkillSource:  hook.Decl.SkillSource,
				Mode:         hook.Mode,
				Required:     hook.Required,
				Priority:     hook.Priority,
				Timeout:      hook.Timeout,
				ExecutorKind: executorKind,
				Matcher:      cloneHookMatcher(hook.Matcher),
				Metadata:     cloneStringMap(hook.Metadata),
			})
		}
	}

	return entries, nil
}

// AllEventDescriptors returns the hook taxonomy metadata in deterministic order.
func AllEventDescriptors() []EventDescriptor {
	return FilterEventDescriptors(EventFilter{})
}

// FilterEventDescriptors returns the hook taxonomy metadata in deterministic order.
func FilterEventDescriptors(filter EventFilter) []EventDescriptor {
	descriptors := make([]EventDescriptor, 0, len(allHookEvents))
	for _, event := range AllHookEvents() {
		if descriptor, ok := hookEventDescriptors[event]; ok {
			if filter.Family != "" && descriptor.Family != filter.Family {
				continue
			}
			if filter.SyncOnly && !descriptor.SyncEligible {
				continue
			}
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func catalogHookMatchesFilter(hook *ResolvedHook, filter CatalogFilter) bool {
	if hook == nil {
		return false
	}
	if filter.Event != "" && hook.Event != filter.Event {
		return false
	}
	if filter.Source != nil && hook.Source != *filter.Source {
		return false
	}
	if filter.Mode != "" && hook.Mode != filter.Mode {
		return false
	}
	if !catalogStringMatches(filter.AgentName, hook.Matcher.AgentName) {
		return false
	}
	if !catalogStringMatches(filter.WorkspaceID, hook.Matcher.WorkspaceID) {
		return false
	}
	if !catalogStringMatches(filter.WorkspaceRoot, hook.Matcher.WorkspaceRoot) {
		return false
	}
	return true
}

func catalogStringMatches(filter string, value string) bool {
	if filter == "" || value == "" {
		return true
	}
	return matchStringField(value, filter)
}

func cloneHookMatcher(src HookMatcher) HookMatcher {
	cloned := src
	if src.ToolReadOnly != nil {
		value := *src.ToolReadOnly
		cloned.ToolReadOnly = &value
	}
	if src.NetworkMatcher != nil {
		value := *src.NetworkMatcher
		cloned.NetworkMatcher = &value
	}
	if src.CompactionMatcher != nil {
		value := *src.CompactionMatcher
		cloned.CompactionMatcher = &value
	}
	if src.Autonomy != nil {
		value := *src.Autonomy
		cloned.Autonomy = &value
	}
	return cloned
}
