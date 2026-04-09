package hooks

import "testing"

func TestHookMatcherMatchesSession(t *testing.T) {
	t.Parallel()

	matcher := HookMatcher{
		WorkspaceID: "ws-1",
		AgentName:   "claude",
	}
	payload := SessionContext{
		WorkspaceID: "ws-1",
		AgentName:   "claude",
	}
	if !matcher.MatchesSession(payload) {
		t.Fatal("MatchesSession() = false, want true")
	}

	payload.AgentName = "codex"
	if matcher.MatchesSession(payload) {
		t.Fatal("MatchesSession() = true, want false for non-matching agent")
	}
}

func TestHookMatcherMatchesToolWithWildcard(t *testing.T) {
	t.Parallel()

	readOnly := true
	matcher := HookMatcher{
		ToolName:      "read_*",
		ToolNamespace: "fs",
		ToolReadOnly:  &readOnly,
	}

	payload := ToolPreCallPayload{
		ToolCallRef: ToolCallRef{
			ToolName:      "read_text_file",
			ToolNamespace: "fs",
			ReadOnly:      true,
		},
	}
	if !matcher.MatchesToolPreCall(payload) {
		t.Fatal("MatchesToolPreCall() = false, want true")
	}

	payload.ToolNamespace = "terminal"
	if matcher.MatchesToolPreCall(payload) {
		t.Fatal("MatchesToolPreCall() = true, want false for namespace mismatch")
	}
}

func TestHookMatcherMatchesPermission(t *testing.T) {
	t.Parallel()

	matcher := HookMatcher{
		ToolName:      "fs/*",
		DecisionClass: "filesystem",
	}
	payload := PermissionRequestPayload{
		DecisionClass: "filesystem",
		ToolCall: PermissionToolCall{
			Kind: "fs/read_text_file",
		},
	}
	if !matcher.MatchesPermissionRequest(payload) {
		t.Fatal("MatchesPermissionRequest() = false, want true")
	}

	payload.DecisionClass = "terminal"
	if matcher.MatchesPermissionRequest(payload) {
		t.Fatal("MatchesPermissionRequest() = true, want false for decision class mismatch")
	}
}

func TestHookMatcherMatchesMessageAndContext(t *testing.T) {
	t.Parallel()

	messageMatcher := HookMatcher{
		MessageRole:      "assistant",
		MessageDeltaType: "text",
	}
	if !messageMatcher.MatchesMessage(MessagePayload{Role: "assistant", DeltaType: "text"}) {
		t.Fatal("MatchesMessage() = false, want true")
	}
	if messageMatcher.MatchesMessage(MessagePayload{Role: "user", DeltaType: "text"}) {
		t.Fatal("MatchesMessage() = true, want false for role mismatch")
	}

	contextMatcher := HookMatcher{
		CompactionReason:   "token_limit",
		CompactionStrategy: "summary",
	}
	if !contextMatcher.MatchesContextCompact(ContextCompactPayload{
		Reason:   "token_limit",
		Strategy: "summary",
	}) {
		t.Fatal("MatchesContextCompact() = false, want true")
	}
	if contextMatcher.MatchesContextCompact(ContextCompactPayload{
		Reason:   "manual",
		Strategy: "summary",
	}) {
		t.Fatal("MatchesContextCompact() = true, want false for reason mismatch")
	}
}

func TestHookMatcherMatchesAdditionalFamilies(t *testing.T) {
	t.Parallel()

	scopeMatcher := HookMatcher{
		AgentName:     "claude",
		WorkspaceID:   "ws-1",
		WorkspaceRoot: "/workspace/demo",
		InputClass:    "chat",
	}

	if !scopeMatcher.MatchesInput(InputPreSubmitPayload{
		SessionContext: SessionContext{
			AgentName:   "claude",
			WorkspaceID: "ws-1",
			Workspace:   "/workspace/demo",
		},
		InputClass: "chat",
	}) {
		t.Fatal("MatchesInput() = false, want true")
	}

	if !scopeMatcher.MatchesPrompt(PromptPayload{
		SessionContext: SessionContext{
			AgentName:   "claude",
			WorkspaceID: "ws-1",
			Workspace:   "/workspace/demo",
		},
		InputClass: "chat",
	}) {
		t.Fatal("MatchesPrompt() = false, want true")
	}

	if !scopeMatcher.MatchesAgentPreStart(AgentPreStartPayload{
		SessionContext: SessionContext{
			AgentName:   "claude",
			WorkspaceID: "ws-1",
			Workspace:   "/workspace/demo",
		},
	}) {
		t.Fatal("MatchesAgentPreStart() = false, want true")
	}

	if !scopeMatcher.MatchesAgentLifecycle(AgentLifecyclePayload{
		SessionContext: SessionContext{
			AgentName:   "claude",
			WorkspaceID: "ws-1",
			Workspace:   "/workspace/demo",
		},
	}) {
		t.Fatal("MatchesAgentLifecycle() = false, want true")
	}

	if !scopeMatcher.MatchesTurn(TurnPayload{
		SessionContext: SessionContext{
			AgentName:   "claude",
			WorkspaceID: "ws-1",
			Workspace:   "/workspace/demo",
		},
		InputClass: "chat",
	}) {
		t.Fatal("MatchesTurn() = false, want true")
	}

	eventMatcher := HookMatcher{
		AgentName:    "claude",
		ACPEventType: "permission",
		TurnID:       "turn-1",
	}
	if !eventMatcher.MatchesEvent(EventRecordPayload{
		SessionContext: SessionContext{AgentName: "claude"},
		TurnContext:    TurnContext{TurnID: "turn-1"},
		RecordType:     "permission",
	}) {
		t.Fatal("MatchesEvent() = false, want true")
	}

	toolMatcher := HookMatcher{
		ToolName:      "run",
		ToolNamespace: "terminal",
	}
	if !toolMatcher.MatchesToolPostCall(ToolPostCallPayload{
		ToolCallRef: ToolCallRef{ToolName: "run", ToolNamespace: "terminal"},
	}) {
		t.Fatal("MatchesToolPostCall() = false, want true")
	}
	if !toolMatcher.MatchesToolPostError(ToolPostErrorPayload{
		ToolCallRef: ToolCallRef{ToolName: "run", ToolNamespace: "terminal"},
	}) {
		t.Fatal("MatchesToolPostError() = false, want true")
	}

	permissionMatcher := HookMatcher{
		ToolName:      "terminal/run",
		DecisionClass: "command",
	}
	if !permissionMatcher.MatchesPermissionResolution(PermissionResolutionPayload{
		DecisionClass: "command",
		ToolCall: PermissionToolCall{
			Kind: "terminal/run",
		},
	}) {
		t.Fatal("MatchesPermissionResolution() = false, want true")
	}
}
