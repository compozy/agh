package hooks

import "time"

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
	Priority     int
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
	HookEnvironmentPrepare: {
		Event:         HookEnvironmentPrepare,
		Family:        HookEventFamilyEnvironment,
		SyncEligible:  true,
		PayloadSchema: "EnvironmentPreparePayload",
		PatchSchema:   "EnvironmentPreparePatch",
	},
	HookEnvironmentReady: {
		Event:         HookEnvironmentReady,
		Family:        HookEventFamilyEnvironment,
		SyncEligible:  false,
		PayloadSchema: "EnvironmentReadyPayload",
		PatchSchema:   "EnvironmentReadyPatch",
	},
	HookEnvironmentSyncBefore: {
		Event:         HookEnvironmentSyncBefore,
		Family:        HookEventFamilyEnvironment,
		SyncEligible:  true,
		PayloadSchema: "EnvironmentSyncBeforePayload",
		PatchSchema:   "EnvironmentSyncBeforePatch",
	},
	HookEnvironmentSyncAfter: {
		Event:         HookEnvironmentSyncAfter,
		Family:        HookEventFamilyEnvironment,
		SyncEligible:  false,
		PayloadSchema: "EnvironmentSyncAfterPayload",
		PatchSchema:   "EnvironmentSyncAfterPatch",
	},
	HookEnvironmentStop: {
		Event:         HookEnvironmentStop,
		Family:        HookEventFamilyEnvironment,
		SyncEligible:  true,
		PayloadSchema: "EnvironmentStopPayload",
		PatchSchema:   "EnvironmentStopPatch",
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
		PatchSchema:   "AutomationFirePatch",
	},
	HookAutomationJobPostFire: {
		Event:         HookAutomationJobPostFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationJobPostFirePayload",
		PatchSchema:   "AutomationObservationPatch",
	},
	HookAutomationTriggerPreFire: {
		Event:         HookAutomationTriggerPreFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  true,
		PayloadSchema: "AutomationTriggerPreFirePayload",
		PatchSchema:   "AutomationFirePatch",
	},
	HookAutomationTriggerPostFire: {
		Event:         HookAutomationTriggerPostFire,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationTriggerPostFirePayload",
		PatchSchema:   "AutomationObservationPatch",
	},
	HookAutomationRunCompleted: {
		Event:         HookAutomationRunCompleted,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationRunCompletedPayload",
		PatchSchema:   "AutomationObservationPatch",
	},
	HookAutomationRunFailed: {
		Event:         HookAutomationRunFailed,
		Family:        HookEventFamilyAutomation,
		SyncEligible:  false,
		PayloadSchema: "AutomationRunFailedPayload",
		PatchSchema:   "AutomationObservationPatch",
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
		PatchSchema:   "CoordinatorObservationPatch",
	},
	HookCoordinatorDecision: {
		Event:         HookCoordinatorDecision,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorDecisionPayload",
		PatchSchema:   "CoordinatorObservationPatch",
	},
	HookCoordinatorStopped: {
		Event:         HookCoordinatorStopped,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorStoppedPayload",
		PatchSchema:   "CoordinatorObservationPatch",
	},
	HookCoordinatorFailed: {
		Event:         HookCoordinatorFailed,
		Family:        HookEventFamilyCoordinator,
		SyncEligible:  true,
		PayloadSchema: "CoordinatorFailedPayload",
		PatchSchema:   "CoordinatorObservationPatch",
	},
	HookTaskRunEnqueued: {
		Event:         HookTaskRunEnqueued,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunEnqueuedPayload",
		PatchSchema:   "TaskRunObservationPatch",
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
		PatchSchema:   "TaskRunObservationPatch",
	},
	HookTaskRunLeaseExtended: {
		Event:         HookTaskRunLeaseExtended,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseExtendedPayload",
		PatchSchema:   "TaskRunObservationPatch",
	},
	HookTaskRunLeaseExpired: {
		Event:         HookTaskRunLeaseExpired,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseExpiredPayload",
		PatchSchema:   "TaskRunObservationPatch",
	},
	HookTaskRunLeaseRecovered: {
		Event:         HookTaskRunLeaseRecovered,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunLeaseRecoveredPayload",
		PatchSchema:   "TaskRunObservationPatch",
	},
	HookTaskRunReleased: {
		Event:         HookTaskRunReleased,
		Family:        HookEventFamilyTaskRun,
		SyncEligible:  true,
		PayloadSchema: "TaskRunReleasedPayload",
		PatchSchema:   "TaskRunObservationPatch",
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
		PatchSchema:   "SpawnObservationPatch",
	},
	HookSpawnParentStopped: {
		Event:         HookSpawnParentStopped,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnParentStoppedPayload",
		PatchSchema:   "SpawnObservationPatch",
	},
	HookSpawnTTLExpired: {
		Event:         HookSpawnTTLExpired,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnTTLExpiredPayload",
		PatchSchema:   "SpawnObservationPatch",
	},
	HookSpawnReaped: {
		Event:         HookSpawnReaped,
		Family:        HookEventFamilySpawn,
		SyncEligible:  true,
		PayloadSchema: "SpawnReapedPayload",
		PatchSchema:   "SpawnObservationPatch",
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
	return filter == value
}

func cloneHookMatcher(src HookMatcher) HookMatcher {
	cloned := src
	if src.ToolReadOnly != nil {
		value := *src.ToolReadOnly
		cloned.ToolReadOnly = &value
	}
	if src.Autonomy != nil {
		value := *src.Autonomy
		cloned.Autonomy = &value
	}
	return cloned
}
