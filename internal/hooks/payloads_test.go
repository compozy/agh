package hooks

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestPayloadsAndPatchesJSONRoundTrip(t *testing.T) {
	t.Parallel()

	sampleSession := SessionContext{
		SessionID:    "sess-1",
		SessionName:  "demo",
		SessionType:  "user",
		AgentName:    "codex",
		WorkspaceID:  "ws-1",
		Workspace:    "/tmp/demo",
		ACPSessionID: "acp-1",
		State:        "active",
	}
	sampleTurn := TurnContext{TurnID: "turn-1"}
	samplePayloadBase := func(event HookEvent) PayloadBase {
		return PayloadBase{
			Event:     event,
			Timestamp: time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC),
		}
	}
	sampleContextBlocks := []ContextBlock{
		{
			Kind: "policy",
			Text: "ctx",
			Metadata: map[string]string{
				"source": "test",
			},
		},
	}
	sampleRaw := json.RawMessage(`{"key":"value"}`)
	allowOnce := "allow-once"
	reason := "blocked"
	toolName := "grep"
	toolNamespace := "fs"
	strategy := "summarize"
	text := "patched"
	role := "assistant"
	deltaType := "text"
	sessionName := "patched-session"
	sessionType := "system"
	agentName := "native"
	workspaceID := "ws-2"
	workspace := "/tmp/other"
	title := "result"
	readOnly := true

	assertJSONRoundTrip(t, "SessionPreCreatePayload", SessionPreCreatePayload{
		PayloadBase:    samplePayloadBase(HookSessionPreCreate),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionPostCreatePayload", SessionPostCreatePayload{
		PayloadBase:    samplePayloadBase(HookSessionPostCreate),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionPreResumePayload", SessionPreResumePayload{
		PayloadBase:    samplePayloadBase(HookSessionPreResume),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionPostResumePayload", SessionPostResumePayload{
		PayloadBase:    samplePayloadBase(HookSessionPostResume),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionPreStopPayload", SessionPreStopPayload{
		PayloadBase:    samplePayloadBase(HookSessionPreStop),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionPostStopPayload", SessionPostStopPayload{
		PayloadBase:    samplePayloadBase(HookSessionPostStop),
		SessionContext: sampleSession,
	})
	assertJSONRoundTrip(t, "SessionCreatePatch", SessionCreatePatch{
		ControlPatch: ControlPatch{Deny: true, DenyReason: "policy"},
		SessionName:  &sessionName,
		SessionType:  &sessionType,
		AgentName:    &agentName,
		WorkspaceID:  &workspaceID,
		Workspace:    &workspace,
	})
	assertJSONRoundTrip(t, "SessionPostCreatePatch", SessionPostCreatePatch{
		ControlPatch: ControlPatch{DenyReason: "observe"},
		SessionName:  &sessionName,
	})
	assertJSONRoundTrip(t, "SessionPreResumePatch", SessionPreResumePatch{
		SessionType: &sessionType,
	})
	assertJSONRoundTrip(t, "SessionPostResumePatch", SessionPostResumePatch{
		AgentName: &agentName,
	})
	assertJSONRoundTrip(t, "SessionPreStopPatch", SessionPreStopPatch{
		ControlPatch: ControlPatch{Deny: true, DenyReason: "stop"},
	})
	assertJSONRoundTrip(t, "SessionPostStopPatch", SessionPostStopPatch{
		Workspace: &workspace,
	})

	assertJSONRoundTrip(t, "SandboxPreparePayload", SandboxPreparePayload{
		PayloadBase:    samplePayloadBase(HookSandboxPrepare),
		SessionContext: sampleSession,
		SandboxID:      "env-1",
		Backend:        "daytona",
		Profile: SandboxProfilePayload{
			Profile:        "daytona-dev",
			Backend:        "daytona",
			SyncMode:       "session-bidirectional",
			Persistence:    "transient",
			RuntimeRootDir: "/workspace",
			DestroyOnStop:  true,
			Env:            map[string]string{"BASE": "1"},
		},
		LocalRootDir:        "/local",
		LocalAdditionalDirs: []string{"/local-extra"},
		AgentCommand:        "codex",
		AgentEnv:            []string{"BASE=1"},
		Permissions:         "approve-all",
		ResumeACPState:      "acp-1",
		EnvOverrides:        map[string]string{"SECRET": "token"},
		Denied:              true,
		DenyReason:          "policy",
	})
	assertJSONRoundTrip(t, "SandboxReadyPayload", SandboxReadyPayload{
		PayloadBase:           samplePayloadBase(HookSandboxReady),
		SessionContext:        sampleSession,
		SandboxID:             "env-1",
		Backend:               "daytona",
		Profile:               "daytona-dev",
		InstanceID:            "instance-1",
		RuntimeRootDir:        "/runtime",
		RuntimeAdditionalDirs: []string{"/runtime-extra"},
	})
	assertJSONRoundTrip(t, "SandboxSyncBeforePayload", SandboxSyncBeforePayload{
		PayloadBase:     samplePayloadBase(HookSandboxSyncBefore),
		SessionContext:  sampleSession,
		SandboxID:       "env-1",
		Backend:         "daytona",
		Profile:         "daytona-dev",
		InstanceID:      "instance-1",
		RuntimeRootDir:  "/runtime",
		Direction:       "to_runtime",
		Reason:          "start",
		FileCount:       3,
		ExcludePatterns: []string{"node_modules/**"},
		Denied:          true,
		DenyReason:      "blocked",
	})
	assertJSONRoundTrip(t, "SandboxSyncAfterPayload", SandboxSyncAfterPayload{
		PayloadBase:      samplePayloadBase(HookSandboxSyncAfter),
		SessionContext:   sampleSession,
		SandboxID:        "env-1",
		Backend:          "daytona",
		Profile:          "daytona-dev",
		InstanceID:       "instance-1",
		RuntimeRootDir:   "/runtime",
		Direction:        "from_runtime",
		Reason:           "stop",
		FilesSynced:      5,
		BytesTransferred: 4096,
		DurationMS:       37,
		Errors:           []string{"retryable warning"},
	})
	assertJSONRoundTrip(t, "SandboxStopPayload", SandboxStopPayload{
		PayloadBase:    samplePayloadBase(HookSandboxStop),
		SessionContext: sampleSession,
		SandboxID:      "env-1",
		Backend:        "daytona",
		Profile:        "daytona-dev",
		InstanceID:     "instance-1",
		RuntimeRootDir: "/runtime",
		StopReason:     "user_requested",
		WillDestroy:    true,
		Denied:         true,
		DenyReason:     "retain",
	})
	assertJSONRoundTrip(t, "SandboxPreparePatch", SandboxPreparePatch{
		ControlPatch: ControlPatch{Deny: true, DenyReason: "policy"},
		EnvOverrides: map[string]string{"SECRET": "token"},
	})
	assertJSONRoundTrip(t, "SandboxSyncBeforePatch", SandboxSyncBeforePatch{
		ControlPatch:    ControlPatch{Deny: true, DenyReason: "sync blocked"},
		ExcludePatterns: []string{"tmp/**"},
	})
	assertJSONRoundTrip(t, "SandboxStopPatch", SandboxStopPatch{
		ControlPatch: ControlPatch{Deny: true, DenyReason: "retain"},
	})

	assertJSONRoundTrip(t, "InputPreSubmitPayload", InputPreSubmitPayload{
		PayloadBase:    samplePayloadBase(HookInputPreSubmit),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		InputClass:     "user_message",
		Message:        "hello",
		ContextBlocks:  sampleContextBlocks,
	})
	assertJSONRoundTrip(t, "InputPreSubmitPatch", InputPreSubmitPatch{
		ControlPatch:  ControlPatch{Deny: true, DenyReason: "input"},
		Message:       &text,
		ContextBlocks: sampleContextBlocks,
	})

	assertJSONRoundTrip(t, "PromptPayload", PromptPayload{
		PayloadBase:    samplePayloadBase(HookPromptPostAssemble),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		InputClass:     "user_message",
		Prompt:         "assembled",
		ContextBlocks:  sampleContextBlocks,
	})
	assertJSONRoundTrip(t, "PromptPatch", PromptPatch{
		ControlPatch:  ControlPatch{DenyReason: "prompt"},
		Prompt:        &text,
		ContextBlocks: sampleContextBlocks,
	})

	assertJSONRoundTrip(t, "EventPreRecordPayload", EventPreRecordPayload{
		PayloadBase:    samplePayloadBase(HookEventPreRecord),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		RecordType:     "tool_call",
		Sequence:       1,
		Content:        sampleRaw,
	})
	assertJSONRoundTrip(t, "EventPostRecordPayload", EventPostRecordPayload{
		PayloadBase:    samplePayloadBase(HookEventPostRecord),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		RecordType:     "tool_result",
		Sequence:       2,
		Content:        sampleRaw,
	})
	assertJSONRoundTrip(t, "EventPreRecordPatch", EventPreRecordPatch{
		Labels: map[string]string{"stage": "pre"},
	})
	assertJSONRoundTrip(t, "EventPostRecordPatch", EventPostRecordPatch{
		Labels: map[string]string{"stage": "post"},
	})
	assertJSONRoundTrip(t, "AutomationSchedulePayload", AutomationSchedulePayload{
		Mode:     "every",
		Interval: "5m",
	})
	assertJSONRoundTrip(t, "AutomationJobPreFirePayload", AutomationJobPreFirePayload{
		JobID:       "job-1",
		JobName:     "nightly-report",
		AgentName:   "researcher",
		WorkspaceID: "ws-1",
		Prompt:      "Generate the nightly report",
		Schedule: &AutomationSchedulePayload{
			Mode: "cron",
			Expr: "0 9 * * *",
		},
		Attempt: 1,
	})
	assertJSONRoundTrip(t, "AutomationJobPostFirePayload", AutomationJobPostFirePayload{
		JobID:       "job-1",
		JobName:     "nightly-report",
		AgentName:   "researcher",
		WorkspaceID: "ws-1",
		RunID:       "run-1",
		SessionID:   "sess-1",
	})
	assertJSONRoundTrip(t, "AutomationTriggerPreFirePayload", AutomationTriggerPreFirePayload{
		TriggerID:   "trigger-1",
		TriggerName: "push-review",
		Event:       "ext.github.push",
		AgentName:   "reviewer",
		WorkspaceID: "ws-1",
		Prompt:      "Review push",
		Payload:     map[string]any{"repo": "acme/api"},
		Attempt:     2,
	})
	assertJSONRoundTrip(t, "AutomationTriggerPostFirePayload", AutomationTriggerPostFirePayload{
		TriggerID:   "trigger-1",
		TriggerName: "push-review",
		Event:       "ext.github.push",
		AgentName:   "reviewer",
		WorkspaceID: "ws-1",
		RunID:       "run-2",
		SessionID:   "sess-2",
	})
	assertJSONRoundTrip(t, "AutomationRunCompletedPayload", AutomationRunCompletedPayload{
		RunID:       "run-3",
		JobID:       "job-1",
		AgentName:   "researcher",
		WorkspaceID: "ws-1",
		SessionID:   "sess-3",
		Attempt:     1,
		DurationMS:  4500,
	})
	assertJSONRoundTrip(t, "AutomationRunFailedPayload", AutomationRunFailedPayload{
		RunID:       "run-4",
		TriggerID:   "trigger-1",
		AgentName:   "reviewer",
		WorkspaceID: "ws-1",
		SessionID:   "sess-4",
		Error:       "boom",
		Attempt:     2,
		WillRetry:   true,
	})
	assertJSONRoundTrip(t, "AutomationFirePatch", AutomationFirePatch{
		Prompt: &text,
		Cancel: true,
	})
	assertJSONRoundTrip(t, "AutomationObservationPatch", AutomationObservationPatch{})

	assertJSONRoundTrip(t, "AgentPreStartPayload", AgentPreStartPayload{
		PayloadBase:    samplePayloadBase(HookAgentPreStart),
		SessionContext: sampleSession,
		Command:        "codex",
		Args:           []string{"serve"},
		Cwd:            "/tmp/demo",
		Provider:       "openai",
		Model:          "gpt-5.4",
	})
	assertJSONRoundTrip(t, "AgentSpawnedPayload", AgentSpawnedPayload{
		PayloadBase:    samplePayloadBase(HookAgentSpawned),
		SessionContext: sampleSession,
		Command:        "codex",
		Args:           []string{"serve"},
		Cwd:            "/tmp/demo",
		PID:            123,
		Provider:       "openai",
		Model:          "gpt-5.4",
	})
	assertJSONRoundTrip(t, "AgentCrashedPayload", AgentCrashedPayload{
		PayloadBase:    samplePayloadBase(HookAgentCrashed),
		SessionContext: sampleSession,
		Command:        "codex",
		Args:           []string{"serve"},
		Cwd:            "/tmp/demo",
		PID:            123,
		Provider:       "openai",
		Model:          "gpt-5.4",
		Error:          "boom",
	})
	assertJSONRoundTrip(t, "AgentStoppedPayload", AgentStoppedPayload{
		PayloadBase:    samplePayloadBase(HookAgentStopped),
		SessionContext: sampleSession,
		Command:        "codex",
		Args:           []string{"serve"},
		Cwd:            "/tmp/demo",
		PID:            123,
		Provider:       "openai",
		Model:          "gpt-5.4",
	})
	assertJSONRoundTrip(t, "AgentStartPatch", AgentStartPatch{
		ControlPatch: ControlPatch{DenyReason: "agent"},
		Command:      &toolName,
		Args:         []string{"--safe"},
	})
	assertJSONRoundTrip(t, "AgentSpawnedPatch", AgentSpawnedPatch{
		Labels: map[string]string{"state": "spawned"},
	})
	assertJSONRoundTrip(t, "AgentCrashedPatch", AgentCrashedPatch{
		Labels: map[string]string{"state": "crashed"},
	})
	assertJSONRoundTrip(t, "AgentStoppedPatch", AgentStoppedPatch{
		Labels: map[string]string{"state": "stopped"},
	})

	assertJSONRoundTrip(t, "TurnStartPayload", TurnStartPayload{
		PayloadBase:    samplePayloadBase(HookTurnStart),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		InputClass:     "user_message",
		UserMessage:    "hello",
	})
	assertJSONRoundTrip(t, "TurnEndPayload", TurnEndPayload{
		PayloadBase:    samplePayloadBase(HookTurnEnd),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		InputClass:     "user_message",
		UserMessage:    "bye",
	})
	assertJSONRoundTrip(t, "TurnStartPatch", TurnStartPatch{
		ControlPatch: ControlPatch{DenyReason: "turn"},
		Labels:       map[string]string{"phase": "start"},
	})
	assertJSONRoundTrip(t, "TurnEndPatch", TurnEndPatch{
		Labels: map[string]string{"phase": "end"},
	})

	assertJSONRoundTrip(t, "MessageStartPayload", MessageStartPayload{
		PayloadBase:    samplePayloadBase(HookMessageStart),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		MessageID:      "msg-1",
		Role:           "assistant",
		DeltaType:      "full",
		Text:           "hello",
		Raw:            sampleRaw,
	})
	assertJSONRoundTrip(t, "MessageDeltaPayload", MessageDeltaPayload{
		PayloadBase:    samplePayloadBase(HookMessageDelta),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		MessageID:      "msg-1",
		Role:           "assistant",
		DeltaType:      "text",
		Text:           "hel",
		Raw:            sampleRaw,
	})
	assertJSONRoundTrip(t, "MessageEndPayload", MessageEndPayload{
		PayloadBase:    samplePayloadBase(HookMessageEnd),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		MessageID:      "msg-1",
		Role:           "assistant",
		DeltaType:      "full",
		Text:           "hello",
		Raw:            sampleRaw,
	})
	assertJSONRoundTrip(t, "MessageStartPatch", MessageStartPatch{
		ControlPatch: ControlPatch{DenyReason: "message"},
		Role:         &role,
		DeltaType:    &deltaType,
		Text:         &text,
	})
	assertJSONRoundTrip(t, "MessageDeltaPatch", MessageDeltaPatch{
		DeltaType: &deltaType,
	})
	assertJSONRoundTrip(t, "MessageEndPatch", MessageEndPatch{
		Text: &text,
	})

	assertJSONRoundTrip(t, "ToolPreCallPayload", ToolPreCallPayload{
		PayloadBase:    samplePayloadBase(HookToolPreCall),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		ToolCallRef: ToolCallRef{
			ToolCallID:    "tool-1",
			ToolName:      "grep",
			ToolNamespace: "fs",
			ReadOnly:      true,
		},
		ToolInput: sampleRaw,
	})
	assertJSONRoundTrip(t, "ToolPostCallPayload", ToolPostCallPayload{
		PayloadBase:    samplePayloadBase(HookToolPostCall),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		ToolCallRef: ToolCallRef{
			ToolCallID:    "tool-1",
			ToolName:      "grep",
			ToolNamespace: "fs",
			ReadOnly:      true,
		},
		Title:      "grep result",
		ToolInput:  sampleRaw,
		ToolResult: sampleRaw,
	})
	assertJSONRoundTrip(t, "ToolPostErrorPayload", ToolPostErrorPayload{
		PayloadBase:    samplePayloadBase(HookToolPostError),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		ToolCallRef: ToolCallRef{
			ToolCallID:    "tool-1",
			ToolName:      "grep",
			ToolNamespace: "fs",
			ReadOnly:      true,
		},
		Title:     "grep error",
		ToolInput: sampleRaw,
		Error:     "failed",
	})
	assertJSONRoundTrip(t, "ToolCallPatch", ToolCallPatch{
		ControlPatch:  ControlPatch{DenyReason: "tool"},
		ToolName:      &toolName,
		ToolNamespace: &toolNamespace,
		ReadOnly:      &readOnly,
		ToolInput:     sampleRaw,
	})
	assertJSONRoundTrip(t, "ToolResultPatch", ToolResultPatch{
		ControlPatch: ControlPatch{DenyReason: "result"},
		Title:        &title,
		ToolResult:   sampleRaw,
	})
	assertJSONRoundTrip(t, "ToolPostErrorPatch", ToolPostErrorPatch{
		Error: &reason,
	})

	assertJSONRoundTrip(t, "PermissionRequestPayload", PermissionRequestPayload{
		PayloadBase:    samplePayloadBase(HookPermissionRequest),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		RequestID:      "req-1",
		Action:         "session/request_permission",
		Resource:       "/tmp/demo.txt",
		Decision:       "pending",
		DecisionClass:  "interactive",
		ToolInput:      sampleRaw,
		ToolCall: PermissionToolCall{
			ID:     "tool-1",
			Kind:   "read",
			Title:  "Read file",
			Status: "pending",
			Locations: []ToolLocation{
				{Path: "/tmp/demo.txt", StartLine: 1, EndLine: 1},
			},
		},
		Options: []PermissionOption{
			{Decision: "allow-once", OptionID: "allow-once", Kind: "allow"},
		},
	})
	assertJSONRoundTrip(t, "PermissionResolvedPayload", PermissionResolvedPayload{
		PayloadBase:    samplePayloadBase(HookPermissionResolved),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		RequestID:      "req-1",
		Action:         "session/request_permission",
		Resource:       "/tmp/demo.txt",
		Decision:       "allow-once",
		DecisionClass:  "interactive",
		ToolInput:      sampleRaw,
		ToolCall:       PermissionToolCall{ID: "tool-1", Kind: "read", Title: "Read file", Status: "done"},
	})
	assertJSONRoundTrip(t, "PermissionDeniedPayload", PermissionDeniedPayload{
		PayloadBase:    samplePayloadBase(HookPermissionDenied),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		RequestID:      "req-2",
		Action:         "session/request_permission",
		Resource:       "/tmp/secret.txt",
		Decision:       "reject-once",
		DecisionClass:  "interactive",
		ToolInput:      sampleRaw,
		ToolCall:       PermissionToolCall{ID: "tool-2", Kind: "read", Title: "Read secret", Status: "done"},
	})
	assertJSONRoundTrip(t, "PermissionRequestPatch", PermissionRequestPatch{
		ControlPatch:  ControlPatch{Deny: true, DenyReason: "permission"},
		Decision:      &allowOnce,
		DecisionClass: &role,
		Reason:        &reason,
	})
	assertJSONRoundTrip(t, "PermissionResolvedPatch", PermissionResolvedPatch{})
	assertJSONRoundTrip(t, "PermissionDeniedPatch", PermissionDeniedPatch{})

	assertJSONRoundTrip(t, "ContextPreCompactPayload", ContextPreCompactPayload{
		PayloadBase:    samplePayloadBase(HookContextPreCompact),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		Reason:         "token_limit",
		Strategy:       "summarize",
		Summary:        "before",
		ContextBlocks:  sampleContextBlocks,
	})
	assertJSONRoundTrip(t, "ContextPostCompactPayload", ContextPostCompactPayload{
		PayloadBase:    samplePayloadBase(HookContextPostCompact),
		SessionContext: sampleSession,
		TurnContext:    sampleTurn,
		Reason:         "token_limit",
		Strategy:       "summarize",
		Summary:        "after",
		ContextBlocks:  sampleContextBlocks,
	})
	assertJSONRoundTrip(t, "ContextPreCompactPatch", ContextPreCompactPatch{
		ControlPatch:  ControlPatch{DenyReason: "compact"},
		Reason:        &reason,
		Strategy:      &strategy,
		ContextBlocks: sampleContextBlocks,
	})
	assertJSONRoundTrip(t, "ContextPostCompactPatch", ContextPostCompactPatch{
		Strategy: &strategy,
	})
}

func assertJSONRoundTrip[T any](t *testing.T, name string, sample T) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		t.Parallel()

		data, err := json.Marshal(sample)
		if err != nil {
			t.Fatalf("json.Marshal(%s) error = %v", name, err)
		}

		var decoded T
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(%s) error = %v", name, err)
		}

		if !reflect.DeepEqual(sample, decoded) {
			t.Fatalf("%s round-trip mismatch\ngot:  %#v\nwant: %#v", name, decoded, sample)
		}
	})
}
