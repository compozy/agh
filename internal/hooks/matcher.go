package hooks

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

var allowedMatcherFieldsByFamily = map[HookEventFamily]map[string]struct{}{
	HookEventFamilySession: {
		"agent_name":     {},
		"workspace_id":   {},
		"workspace_root": {},
		"session_type":   {},
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
		"tool_name":      {},
		"tool_namespace": {},
		"tool_read_only": {},
	},
	HookEventFamilyPermission: {
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
		return nil
	}

	sort.Strings(invalid)
	return fmt.Errorf("hooks: matcher fields [%s] are not valid for event %q", strings.Join(invalid, ", "), event)
}

// MatchesSession matches session-family hooks.
func (m HookMatcher) MatchesSession(payload SessionContext) bool {
	return m.matchSessionContext(payload, true)
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
	return m.matchToolCall(payload.ToolCallRef)
}

// MatchesToolPostCall matches tool post-call hooks.
func (m HookMatcher) MatchesToolPostCall(payload ToolPostCallPayload) bool {
	return m.matchToolCall(payload.ToolCallRef)
}

// MatchesToolPostError matches tool post-error hooks.
func (m HookMatcher) MatchesToolPostError(payload ToolPostErrorPayload) bool {
	return m.matchToolCall(payload.ToolCallRef)
}

// MatchesPermissionRequest matches permission-request hooks.
func (m HookMatcher) MatchesPermissionRequest(payload PermissionRequestPayload) bool {
	return m.matchPermission(payload.ToolCall.Kind, payload.DecisionClass)
}

// MatchesPermissionResolution matches resolved and denied permission hooks.
func (m HookMatcher) MatchesPermissionResolution(payload PermissionResolutionPayload) bool {
	return m.matchPermission(payload.ToolCall.Kind, payload.DecisionClass)
}

// MatchesContextCompact matches context-compaction hooks.
func (m HookMatcher) MatchesContextCompact(payload ContextCompactPayload) bool {
	return matchStringField(m.CompactionReason, payload.Reason) &&
		matchStringField(m.CompactionStrategy, payload.Strategy)
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
	if matcher.ToolReadOnly != nil {
		value := *matcher.ToolReadOnly
		normalized.ToolReadOnly = &value
	}
	return normalized
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

	return fields
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
	return err == nil && matched
}
