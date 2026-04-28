package hooks

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type matcherFunc[P any] func(HookMatcher, P) bool

var allowedMatcherFieldsByFamily = map[HookEventFamily]map[string]struct{}{
	HookEventFamilySession: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"session_type":   {},
	},
	HookEventFamilySandbox: {
		"agent_name":      {},
		"workspace_id":    {},
		"workspace_root":  {},
		"sandbox_id":      {},
		"sandbox_backend": {},
		"sandbox_profile": {},
		"sync_direction":  {},
	},
	HookEventFamilyInput: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"input_class":    {},
	},
	HookEventFamilyPrompt: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"input_class":    {},
	},
	HookEventFamilyEvent: {
		"agent_name":     {},
		"acp_event_type": {},
		"turn_id":        {},
	},
	HookEventFamilyAutomation: {
		"agent_name":   {},
		"workspace_id": {},
	},
	HookEventFamilyAgent: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
	},
	HookEventFamilyTurn: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"input_class":    {},
	},
	HookEventFamilyTool: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"tool_name":      {},
		"tool_namespace": {},
		"tool_read_only": {},
	},
	HookEventFamilyPermission: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"tool_name":      {},
		"decision_class": {},
	},
	HookEventFamilyMessage: {
		"message_role":       {},
		"message_delta_type": {},
	},
	HookEventFamilyContext: {
		"compaction_reason":   {},
		"compaction_strategy": {},
	},
	HookEventFamilyCoordinator: {
		"agent_name":              {},
		"workspace_id":            {},
		"workspace_root":          {},
		"task_id":                 {},
		"run_id":                  {},
		"workflow_id":             {},
		"coordination_channel_id": {},
		"coordinator_session_id":  {},
	},
	HookEventFamilyTaskRun: {
		"agent_name":              {},
		"workspace_id":            {},
		"task_id":                 {},
		"run_id":                  {},
		"workflow_id":             {},
		"coordination_channel_id": {},
		"release_reason":          {},
	},
	HookEventFamilySpawn: {
		"agent_name":              {},
		"workspace_id":            {},
		"workspace_root":          {},
		"task_id":                 {},
		"run_id":                  {},
		"workflow_id":             {},
		"coordination_channel_id": {},
		"parent_session_id":       {},
		"root_session_id":         {},
		"child_session_id":        {},
		"spawn_role":              {},
	},
}

// ValidateMatcherForEvent ensures only the matcher fields defined for the event
// family are present.
func ValidateMatcherForEvent(event HookEvent, matcher HookMatcher) error {
	if err := event.Validate(); err != nil {
		return err
	}

	fields := matcherFieldNames(matcher)
	if len(fields) == 0 {
		return nil
	}

	allowed := allowedMatcherFieldsByFamily[event.Family()]
	invalid := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := allowed[field]; ok {
			continue
		}
		invalid = append(invalid, field)
	}
	if len(invalid) == 0 {
		return validateMatcherPatterns(matcher)
	}

	sort.Strings(invalid)
	return fmt.Errorf("hooks: matcher fields [%s] are not valid for event %q", strings.Join(invalid, ", "), event)
}

// MatcherFieldAllowedForEvent reports whether a matcher field is valid for the event family.
func MatcherFieldAllowedForEvent(event HookEvent, field string) bool {
	if err := event.Validate(); err != nil {
		return false
	}
	allowed := allowedMatcherFieldsByFamily[event.Family()]
	_, ok := allowed[strings.TrimSpace(field)]
	return ok
}

// MatchesSession matches session-family hooks.
func (m HookMatcher) MatchesSession(payload SessionContext) bool {
	return m.matchSessionContext(payload, true)
}

// MatchesSandboxPrepare matches sandbox prepare hooks.
func (m HookMatcher) MatchesSandboxPrepare(payload SandboxPreparePayload) bool {
	return m.matchSandbox(
		payload.SessionContext,
		payload.SandboxID,
		payload.Backend,
		payload.Profile.Profile,
		"",
	)
}

// MatchesSandboxReady matches sandbox ready hooks.
func (m HookMatcher) MatchesSandboxReady(payload SandboxReadyPayload) bool {
	return m.matchSandbox(payload.SessionContext, payload.SandboxID, payload.Backend, payload.Profile, "")
}

// MatchesSandboxSyncBefore matches sandbox pre-sync hooks.
func (m HookMatcher) MatchesSandboxSyncBefore(payload SandboxSyncBeforePayload) bool {
	return m.matchSandbox(
		payload.SessionContext,
		payload.SandboxID,
		payload.Backend,
		payload.Profile,
		payload.Direction,
	)
}

// MatchesSandboxSyncAfter matches sandbox post-sync hooks.
func (m HookMatcher) MatchesSandboxSyncAfter(payload SandboxSyncAfterPayload) bool {
	return m.matchSandbox(
		payload.SessionContext,
		payload.SandboxID,
		payload.Backend,
		payload.Profile,
		payload.Direction,
	)
}

// MatchesSandboxStop matches sandbox stop hooks.
func (m HookMatcher) MatchesSandboxStop(payload SandboxStopPayload) bool {
	return m.matchSandbox(payload.SessionContext, payload.SandboxID, payload.Backend, payload.Profile, "")
}

// MatchesInput matches input-family hooks.
func (m HookMatcher) MatchesInput(payload InputPreSubmitPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		matchStringField(m.InputClass, payload.InputClass)
}

// MatchesPrompt matches prompt-family hooks.
func (m HookMatcher) MatchesPrompt(payload PromptPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		matchStringField(m.InputClass, payload.InputClass)
}

// MatchesEvent matches event-record-family hooks.
func (m HookMatcher) MatchesEvent(payload EventRecordPayload) bool {
	return matchStringField(m.AgentName, payload.AgentName) &&
		matchStringField(m.ACPEventType, payload.RecordType) &&
		matchStringField(m.TurnID, payload.TurnID)
}

// MatchesAutomation matches automation lifecycle hooks.
func (m HookMatcher) MatchesAutomation(agentName string, workspaceID string) bool {
	return matchStringField(m.AgentName, agentName) &&
		matchStringField(m.WorkspaceID, workspaceID)
}

// MatchesAgentPreStart matches pre-start agent hooks.
func (m HookMatcher) MatchesAgentPreStart(payload AgentPreStartPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false)
}

// MatchesAgentLifecycle matches spawned, crashed, and stopped agent hooks.
func (m HookMatcher) MatchesAgentLifecycle(payload AgentLifecyclePayload) bool {
	return m.matchSessionContext(payload.SessionContext, false)
}

// MatchesTurn matches turn-family hooks.
func (m HookMatcher) MatchesTurn(payload TurnPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		matchStringField(m.InputClass, payload.InputClass)
}

// MatchesMessage matches message-family hooks.
func (m HookMatcher) MatchesMessage(payload MessagePayload) bool {
	return matchStringField(m.MessageRole, payload.Role) &&
		matchStringField(m.MessageDeltaType, payload.DeltaType)
}

// MatchesToolPreCall matches tool pre-call hooks.
func (m HookMatcher) MatchesToolPreCall(payload ToolPreCallPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		m.matchToolCall(payload.ToolCallRef)
}

// MatchesToolPostCall matches tool post-call hooks.
func (m HookMatcher) MatchesToolPostCall(payload ToolPostCallPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		m.matchToolCall(payload.ToolCallRef)
}

// MatchesToolPostError matches tool post-error hooks.
func (m HookMatcher) MatchesToolPostError(payload ToolPostErrorPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		m.matchToolCall(payload.ToolCallRef)
}

// MatchesPermissionRequest matches permission-request hooks.
func (m HookMatcher) MatchesPermissionRequest(payload PermissionRequestPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		m.matchPermission(payload.ToolCall.Kind, payload.DecisionClass)
}

// MatchesPermissionResolution matches resolved and denied permission hooks.
func (m HookMatcher) MatchesPermissionResolution(payload PermissionResolutionPayload) bool {
	return m.matchSessionContext(payload.SessionContext, false) &&
		m.matchPermission(payload.ToolCall.Kind, payload.DecisionClass)
}

// MatchesContextCompact matches context-compaction hooks.
func (m HookMatcher) MatchesContextCompact(payload ContextCompactPayload) bool {
	return matchStringField(m.CompactionReason, payload.Reason) &&
		matchStringField(m.CompactionStrategy, payload.Strategy)
}

// MatchesCoordinator matches coordinator-family hooks.
func (m HookMatcher) MatchesCoordinator(payload CoordinatorContext) bool {
	autonomy := m.autonomy()
	return matchStringField(m.AgentName, payload.AgentName) &&
		matchStringField(m.WorkspaceID, payload.WorkspaceID) &&
		matchStringField(m.WorkspaceRoot, payload.Workspace) &&
		matchStringField(autonomy.TaskID, payload.TaskID) &&
		matchStringField(autonomy.RunID, payload.RunID) &&
		matchStringField(autonomy.WorkflowID, payload.WorkflowID) &&
		matchStringField(autonomy.CoordinationChannelID, payload.CoordinationChannelID) &&
		matchStringField(autonomy.CoordinatorSessionID, payload.CoordinatorSessionID)
}

// MatchesTaskRun matches task-run-family hooks.
func (m HookMatcher) MatchesTaskRun(payload TaskRunContext) bool {
	autonomy := m.autonomy()
	return matchStringField(m.AgentName, payload.AgentName) &&
		matchStringField(m.WorkspaceID, payload.WorkspaceID) &&
		matchStringField(autonomy.TaskID, payload.TaskID) &&
		matchStringField(autonomy.RunID, payload.RunID) &&
		matchStringField(autonomy.WorkflowID, payload.WorkflowID) &&
		matchStringField(autonomy.CoordinationChannelID, payload.CoordinationChannelID) &&
		matchStringField(autonomy.ReleaseReason, payload.ReleaseReason)
}

// MatchesSpawn matches spawn-family hooks.
func (m HookMatcher) MatchesSpawn(payload SpawnContext) bool {
	autonomy := m.autonomy()
	return matchStringField(m.AgentName, payload.AgentName) &&
		matchStringField(m.WorkspaceID, payload.WorkspaceID) &&
		matchStringField(m.WorkspaceRoot, payload.Workspace) &&
		matchStringField(autonomy.TaskID, payload.TaskID) &&
		matchStringField(autonomy.RunID, payload.RunID) &&
		matchStringField(autonomy.WorkflowID, payload.WorkflowID) &&
		matchStringField(autonomy.CoordinationChannelID, payload.CoordinationChannelID) &&
		matchStringField(autonomy.ParentSessionID, payload.ParentSessionID) &&
		matchStringField(autonomy.RootSessionID, payload.RootSessionID) &&
		matchStringField(autonomy.ChildSessionID, payload.ChildSessionID) &&
		matchStringField(autonomy.SpawnRole, payload.SpawnRole)
}

var emptyAutonomyMatcher = &AutonomyMatcher{}

func (m HookMatcher) autonomy() *AutonomyMatcher {
	if m.Autonomy == nil {
		return emptyAutonomyMatcher
	}
	return m.Autonomy
}

func selectMatchingHooks[P any](
	snapshot []*ResolvedHook,
	payload P,
	match matcherFunc[P],
) ([]*ResolvedHook, []*ResolvedHook) {
	syncHooks := make([]*ResolvedHook, 0, len(snapshot))
	asyncHooks := make([]*ResolvedHook, 0, len(snapshot))

	for _, hook := range snapshot {
		if hook == nil {
			continue
		}
		if match != nil && !match(hook.Matcher, payload) {
			continue
		}
		switch hook.Mode {
		case HookModeAsync:
			asyncHooks = append(asyncHooks, hook)
		case HookModeSync:
			syncHooks = append(syncHooks, hook)
		}
	}

	return syncHooks, asyncHooks
}

func matchSessionPreCreate(matcher HookMatcher, payload SessionPreCreatePayload) bool {
	return matcher.MatchesSession(payload.SessionContext)
}

func matchSessionLifecycle(matcher HookMatcher, payload SessionLifecyclePayload) bool {
	return matcher.MatchesSession(payload.SessionContext)
}

func matchSandboxPrepare(matcher HookMatcher, payload SandboxPreparePayload) bool {
	return matcher.MatchesSandboxPrepare(payload)
}

func matchSandboxReady(matcher HookMatcher, payload SandboxReadyPayload) bool {
	return matcher.MatchesSandboxReady(payload)
}

func matchSandboxSyncBefore(matcher HookMatcher, payload SandboxSyncBeforePayload) bool {
	return matcher.MatchesSandboxSyncBefore(payload)
}

func matchSandboxSyncAfter(matcher HookMatcher, payload SandboxSyncAfterPayload) bool {
	return matcher.MatchesSandboxSyncAfter(payload)
}

func matchSandboxStop(matcher HookMatcher, payload SandboxStopPayload) bool {
	return matcher.MatchesSandboxStop(payload)
}

func matchInputPreSubmit(matcher HookMatcher, payload InputPreSubmitPayload) bool {
	return matcher.MatchesInput(payload)
}

func matchPrompt(matcher HookMatcher, payload PromptPayload) bool {
	return matcher.MatchesPrompt(payload)
}

func matchEventRecord(matcher HookMatcher, payload EventRecordPayload) bool {
	return matcher.MatchesEvent(payload)
}

func matchAutomationJobPreFire(matcher HookMatcher, payload AutomationJobPreFirePayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAutomationJobPostFire(matcher HookMatcher, payload AutomationJobPostFirePayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAutomationTriggerPreFire(matcher HookMatcher, payload AutomationTriggerPreFirePayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAutomationTriggerPostFire(matcher HookMatcher, payload AutomationTriggerPostFirePayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAutomationRunCompleted(matcher HookMatcher, payload AutomationRunCompletedPayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAutomationRunFailed(matcher HookMatcher, payload AutomationRunFailedPayload) bool {
	return matcher.MatchesAutomation(payload.AgentName, payload.WorkspaceID)
}

func matchAgentPreStart(matcher HookMatcher, payload AgentPreStartPayload) bool {
	return matcher.MatchesAgentPreStart(payload)
}

func matchAgentLifecycle(matcher HookMatcher, payload AgentLifecyclePayload) bool {
	return matcher.MatchesAgentLifecycle(payload)
}

func matchTurn(matcher HookMatcher, payload TurnPayload) bool {
	return matcher.MatchesTurn(payload)
}

func matchMessage(matcher HookMatcher, payload MessagePayload) bool {
	return matcher.MatchesMessage(payload)
}

func matchToolPreCall(matcher HookMatcher, payload ToolPreCallPayload) bool {
	return matcher.MatchesToolPreCall(payload)
}

func matchToolPostCall(matcher HookMatcher, payload ToolPostCallPayload) bool {
	return matcher.MatchesToolPostCall(payload)
}

func matchToolPostError(matcher HookMatcher, payload ToolPostErrorPayload) bool {
	return matcher.MatchesToolPostError(payload)
}

func matchPermissionRequest(matcher HookMatcher, payload PermissionRequestPayload) bool {
	return matcher.MatchesPermissionRequest(payload)
}

func matchPermissionResolution(matcher HookMatcher, payload PermissionResolutionPayload) bool {
	return matcher.MatchesPermissionResolution(payload)
}

func matchContextCompact(matcher HookMatcher, payload ContextCompactPayload) bool {
	return matcher.MatchesContextCompact(payload)
}

func matchCoordinatorPreSpawn(matcher HookMatcher, payload CoordinatorPreSpawnPayload) bool {
	return matcher.MatchesCoordinator(payload.CoordinatorContext)
}

func matchCoordinatorLifecycle(matcher HookMatcher, payload CoordinatorLifecyclePayload) bool {
	return matcher.MatchesCoordinator(payload.CoordinatorContext)
}

func matchTaskRunEnqueued(matcher HookMatcher, payload TaskRunEnqueuedPayload) bool {
	return matcher.MatchesTaskRun(payload.TaskRunContext)
}

func matchTaskRunPreClaim(matcher HookMatcher, payload TaskRunPreClaimPayload) bool {
	return matcher.MatchesTaskRun(payload.TaskRunContext)
}

func matchTaskRunPostClaim(matcher HookMatcher, payload TaskRunPostClaimPayload) bool {
	return matcher.MatchesTaskRun(payload.TaskRunContext)
}

func matchTaskRunLease(matcher HookMatcher, payload TaskRunLeasePayload) bool {
	return matcher.MatchesTaskRun(payload.TaskRunContext)
}

func matchSpawnPreCreate(matcher HookMatcher, payload SpawnPreCreatePayload) bool {
	return matcher.MatchesSpawn(payload.SpawnContext)
}

func matchSpawnLifecycle(matcher HookMatcher, payload SpawnLifecyclePayload) bool {
	return matcher.MatchesSpawn(payload.SpawnContext)
}

func (m HookMatcher) matchSessionContext(payload SessionContext, includeSessionType bool) bool {
	if !matchStringField(m.AgentName, payload.AgentName) {
		return false
	}
	if !matchStringField(m.WorkspaceID, payload.WorkspaceID) {
		return false
	}
	if !matchStringField(m.WorkspaceRoot, payload.Workspace) {
		return false
	}
	if includeSessionType && !matchStringField(m.SessionType, payload.SessionType) {
		return false
	}
	return true
}

func (m HookMatcher) matchSandbox(
	session SessionContext,
	sandboxID string,
	backend string,
	profile string,
	direction string,
) bool {
	return m.matchSessionContext(session, false) &&
		matchStringField(m.SandboxID, sandboxID) &&
		matchStringField(m.SandboxBackend, backend) &&
		matchStringField(m.SandboxProfile, profile) &&
		matchStringField(m.SyncDirection, direction)
}

func (m HookMatcher) matchToolCall(payload ToolCallRef) bool {
	if !matchStringField(m.ToolName, payload.ToolName) {
		return false
	}
	if !matchStringField(m.ToolNamespace, payload.ToolNamespace) {
		return false
	}
	if m.ToolReadOnly != nil && payload.ReadOnly != *m.ToolReadOnly {
		return false
	}
	return true
}

func (m HookMatcher) matchPermission(toolName string, decisionClass string) bool {
	return matchStringField(m.ToolName, toolName) &&
		matchStringField(m.DecisionClass, decisionClass)
}

func normalizeHookMatcher(matcher HookMatcher) HookMatcher {
	normalized := HookMatcher{
		AgentName:          strings.TrimSpace(matcher.AgentName),
		AgentType:          strings.TrimSpace(matcher.AgentType),
		WorkspaceID:        strings.TrimSpace(matcher.WorkspaceID),
		WorkspaceRoot:      strings.TrimSpace(matcher.WorkspaceRoot),
		SessionType:        strings.TrimSpace(matcher.SessionType),
		SandboxID:          strings.TrimSpace(matcher.SandboxID),
		SandboxBackend:     strings.TrimSpace(matcher.SandboxBackend),
		SandboxProfile:     strings.TrimSpace(matcher.SandboxProfile),
		SyncDirection:      strings.TrimSpace(matcher.SyncDirection),
		InputClass:         strings.TrimSpace(matcher.InputClass),
		ACPEventType:       strings.TrimSpace(matcher.ACPEventType),
		TurnID:             strings.TrimSpace(matcher.TurnID),
		ToolName:           strings.TrimSpace(matcher.ToolName),
		ToolNamespace:      strings.TrimSpace(matcher.ToolNamespace),
		DecisionClass:      strings.TrimSpace(matcher.DecisionClass),
		MessageRole:        strings.TrimSpace(matcher.MessageRole),
		MessageDeltaType:   strings.TrimSpace(matcher.MessageDeltaType),
		CompactionReason:   strings.TrimSpace(matcher.CompactionReason),
		CompactionStrategy: strings.TrimSpace(matcher.CompactionStrategy),
	}
	normalized.Autonomy = normalizeAutonomyMatcher(matcher.Autonomy)
	if matcher.ToolReadOnly != nil {
		value := *matcher.ToolReadOnly
		normalized.ToolReadOnly = &value
	}
	return normalized
}

func normalizeAutonomyMatcher(matcher *AutonomyMatcher) *AutonomyMatcher {
	if matcher == nil {
		return nil
	}
	normalized := AutonomyMatcher{
		TaskID:                strings.TrimSpace(matcher.TaskID),
		RunID:                 strings.TrimSpace(matcher.RunID),
		WorkflowID:            strings.TrimSpace(matcher.WorkflowID),
		CoordinationChannelID: strings.TrimSpace(matcher.CoordinationChannelID),
		CoordinatorSessionID:  strings.TrimSpace(matcher.CoordinatorSessionID),
		ParentSessionID:       strings.TrimSpace(matcher.ParentSessionID),
		RootSessionID:         strings.TrimSpace(matcher.RootSessionID),
		ChildSessionID:        strings.TrimSpace(matcher.ChildSessionID),
		SpawnRole:             strings.TrimSpace(matcher.SpawnRole),
		ReleaseReason:         strings.TrimSpace(matcher.ReleaseReason),
	}
	if (&normalized).empty() {
		return nil
	}
	return &normalized
}

func (m *AutonomyMatcher) empty() bool {
	return m.TaskID == "" &&
		m.RunID == "" &&
		m.WorkflowID == "" &&
		m.CoordinationChannelID == "" &&
		m.CoordinatorSessionID == "" &&
		m.ParentSessionID == "" &&
		m.RootSessionID == "" &&
		m.ChildSessionID == "" &&
		m.SpawnRole == "" &&
		m.ReleaseReason == ""
}

func matcherFieldNames(matcher HookMatcher) []string {
	fields := make([]string, 0, 16)

	appendIf := func(name string, present bool) {
		if present {
			fields = append(fields, name)
		}
	}

	appendIf("agent_name", matcher.AgentName != "")
	appendIf("agent_type", matcher.AgentType != "")
	appendIf("workspace_id", matcher.WorkspaceID != "")
	appendIf("workspace_root", matcher.WorkspaceRoot != "")
	appendIf("session_type", matcher.SessionType != "")
	appendIf("sandbox_id", matcher.SandboxID != "")
	appendIf("sandbox_backend", matcher.SandboxBackend != "")
	appendIf("sandbox_profile", matcher.SandboxProfile != "")
	appendIf("sync_direction", matcher.SyncDirection != "")
	appendIf("input_class", matcher.InputClass != "")
	appendIf("acp_event_type", matcher.ACPEventType != "")
	appendIf("turn_id", matcher.TurnID != "")
	appendIf("tool_name", matcher.ToolName != "")
	appendIf("tool_namespace", matcher.ToolNamespace != "")
	appendIf("tool_read_only", matcher.ToolReadOnly != nil)
	appendIf("decision_class", matcher.DecisionClass != "")
	appendIf("message_role", matcher.MessageRole != "")
	appendIf("message_delta_type", matcher.MessageDeltaType != "")
	appendIf("compaction_reason", matcher.CompactionReason != "")
	appendIf("compaction_strategy", matcher.CompactionStrategy != "")
	if matcher.Autonomy != nil {
		appendAutonomyMatcherFieldNames(&fields, matcher.Autonomy)
	}

	return fields
}

func appendAutonomyMatcherFieldNames(fields *[]string, matcher *AutonomyMatcher) {
	appendIf := func(name string, present bool) {
		if present {
			*fields = append(*fields, name)
		}
	}

	appendIf("task_id", matcher.TaskID != "")
	appendIf("run_id", matcher.RunID != "")
	appendIf("workflow_id", matcher.WorkflowID != "")
	appendIf("coordination_channel_id", matcher.CoordinationChannelID != "")
	appendIf("coordinator_session_id", matcher.CoordinatorSessionID != "")
	appendIf("parent_session_id", matcher.ParentSessionID != "")
	appendIf("root_session_id", matcher.RootSessionID != "")
	appendIf("child_session_id", matcher.ChildSessionID != "")
	appendIf("spawn_role", matcher.SpawnRole != "")
	appendIf("release_reason", matcher.ReleaseReason != "")
}

func validateMatcherPatterns(matcher HookMatcher) error {
	patterns := []struct {
		field   string
		pattern string
	}{
		{field: "agent_name", pattern: matcher.AgentName},
		{field: "agent_type", pattern: matcher.AgentType},
		{field: "workspace_id", pattern: matcher.WorkspaceID},
		{field: "workspace_root", pattern: matcher.WorkspaceRoot},
		{field: "session_type", pattern: matcher.SessionType},
		{field: "sandbox_id", pattern: matcher.SandboxID},
		{field: "sandbox_backend", pattern: matcher.SandboxBackend},
		{field: "sandbox_profile", pattern: matcher.SandboxProfile},
		{field: "sync_direction", pattern: matcher.SyncDirection},
		{field: "input_class", pattern: matcher.InputClass},
		{field: "acp_event_type", pattern: matcher.ACPEventType},
		{field: "turn_id", pattern: matcher.TurnID},
		{field: "tool_name", pattern: matcher.ToolName},
		{field: "tool_namespace", pattern: matcher.ToolNamespace},
		{field: "decision_class", pattern: matcher.DecisionClass},
		{field: "message_role", pattern: matcher.MessageRole},
		{field: "message_delta_type", pattern: matcher.MessageDeltaType},
		{field: "compaction_reason", pattern: matcher.CompactionReason},
		{field: "compaction_strategy", pattern: matcher.CompactionStrategy},
	}
	for _, item := range patterns {
		if err := validateMatcherPattern(item.field, item.pattern); err != nil {
			return err
		}
	}
	return validateAutonomyMatcherPatterns(matcher.Autonomy)
}

func validateAutonomyMatcherPatterns(matcher *AutonomyMatcher) error {
	if matcher == nil {
		return nil
	}
	patterns := []struct {
		field   string
		pattern string
	}{
		{field: "task_id", pattern: matcher.TaskID},
		{field: "run_id", pattern: matcher.RunID},
		{field: "workflow_id", pattern: matcher.WorkflowID},
		{field: "coordination_channel_id", pattern: matcher.CoordinationChannelID},
		{field: "coordinator_session_id", pattern: matcher.CoordinatorSessionID},
		{field: "parent_session_id", pattern: matcher.ParentSessionID},
		{field: "root_session_id", pattern: matcher.RootSessionID},
		{field: "child_session_id", pattern: matcher.ChildSessionID},
		{field: "spawn_role", pattern: matcher.SpawnRole},
		{field: "release_reason", pattern: matcher.ReleaseReason},
	}
	for _, item := range patterns {
		if err := validateMatcherPattern(item.field, item.pattern); err != nil {
			return err
		}
	}
	return nil
}

func validateMatcherPattern(field string, pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || !strings.ContainsAny(pattern, "*?[]") {
		return nil
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return fmt.Errorf("hooks: matcher.%s pattern %q is invalid: %w", field, pattern, err)
	}
	return nil
}

func matchStringField(pattern string, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}

	value = strings.TrimSpace(value)
	if !strings.ContainsAny(pattern, "*?[]") {
		return pattern == value
	}

	matched, err := path.Match(pattern, value)
	// Invalid patterns are treated as non-matching at runtime; validation should
	// reject them earlier during normalization.
	return err == nil && matched
}
