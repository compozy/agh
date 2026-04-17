package hooks

import (
	"encoding/json"
	"testing"
)

func TestCloneAsyncPayloadCopiesReferenceFields(t *testing.T) {
	t.Parallel()

	t.Run("environment prepare", func(t *testing.T) {
		t.Parallel()

		original := EnvironmentPreparePayload{
			Profile: EnvironmentProfilePayload{
				Env: map[string]string{"PATH": "/usr/bin"},
			},
			LocalAdditionalDirs: []string{"/tmp/a"},
			AgentEnv:            []string{"KEY=before"},
			EnvOverrides:        map[string]string{"KEY": "before"},
		}
		cloned := cloneAsyncPayload(original)

		original.Profile.Env["PATH"] = "/bin"
		original.LocalAdditionalDirs[0] = "/tmp/b"
		original.AgentEnv[0] = "KEY=after"
		original.EnvOverrides["KEY"] = "after"

		if cloned.Profile.Env["PATH"] != "/usr/bin" {
			t.Fatalf("cloned profile env = %q, want %q", cloned.Profile.Env["PATH"], "/usr/bin")
		}
		if cloned.LocalAdditionalDirs[0] != "/tmp/a" {
			t.Fatalf("cloned local dir = %q, want %q", cloned.LocalAdditionalDirs[0], "/tmp/a")
		}
		if cloned.AgentEnv[0] != "KEY=before" {
			t.Fatalf("cloned agent env = %q, want %q", cloned.AgentEnv[0], "KEY=before")
		}
		if cloned.EnvOverrides["KEY"] != "before" {
			t.Fatalf("cloned env override = %q, want %q", cloned.EnvOverrides["KEY"], "before")
		}
	})

	t.Run("environment lifecycle slices", func(t *testing.T) {
		t.Parallel()

		ready := cloneAsyncPayload(EnvironmentReadyPayload{RuntimeAdditionalDirs: []string{"/runtime"}})
		syncBefore := cloneAsyncPayload(EnvironmentSyncBeforePayload{ExcludePatterns: []string{"*.tmp"}})
		syncAfter := cloneAsyncPayload(EnvironmentSyncAfterPayload{Errors: []string{"before"}})

		if ready.RuntimeAdditionalDirs[0] != "/runtime" {
			t.Fatalf("cloned runtime dir = %q, want %q", ready.RuntimeAdditionalDirs[0], "/runtime")
		}
		if syncBefore.ExcludePatterns[0] != "*.tmp" {
			t.Fatalf("cloned exclude pattern = %q, want %q", syncBefore.ExcludePatterns[0], "*.tmp")
		}
		if syncAfter.Errors[0] != "before" {
			t.Fatalf("cloned error entry = %q, want %q", syncAfter.Errors[0], "before")
		}
	})

	t.Run("prompt and context payloads", func(t *testing.T) {
		t.Parallel()

		original := ContextCompactPayload{
			ContextBlocks: []ContextBlock{{
				Kind: "note",
				Text: "before",
				Metadata: map[string]string{
					"scope": "before",
				},
			}},
		}
		cloned := cloneAsyncPayload(original)

		original.ContextBlocks[0].Text = "after"
		original.ContextBlocks[0].Metadata["scope"] = "after"

		if cloned.ContextBlocks[0].Text != "before" {
			t.Fatalf("cloned context text = %q, want %q", cloned.ContextBlocks[0].Text, "before")
		}
		if cloned.ContextBlocks[0].Metadata["scope"] != "before" {
			t.Fatalf("cloned context metadata = %q, want %q", cloned.ContextBlocks[0].Metadata["scope"], "before")
		}

		prompt := PromptPayload{
			ContextBlocks: []ContextBlock{{
				Kind: "prompt",
				Text: "prompt-before",
			}},
		}
		promptClone := cloneAsyncPayload(prompt)
		prompt.ContextBlocks[0].Text = "prompt-after"
		if promptClone.ContextBlocks[0].Text != "prompt-before" {
			t.Fatalf("cloned prompt text = %q, want %q", promptClone.ContextBlocks[0].Text, "prompt-before")
		}
	})

	t.Run("event and message raw json", func(t *testing.T) {
		t.Parallel()

		event := EventRecordPayload{Content: json.RawMessage(`{"value":"before"}`)}
		eventClone := cloneAsyncPayload(event)
		event.Content[10] = 'X'
		if string(eventClone.Content) != `{"value":"before"}` {
			t.Fatalf("cloned event content = %s, want original payload", string(eventClone.Content))
		}

		message := MessagePayload{Raw: json.RawMessage(`{"message":"before"}`)}
		messageClone := cloneAsyncPayload(message)
		message.Raw[12] = 'X'
		if string(messageClone.Raw) != `{"message":"before"}` {
			t.Fatalf("cloned message raw = %s, want original payload", string(messageClone.Raw))
		}
	})

	t.Run("automation payloads", func(t *testing.T) {
		t.Parallel()

		job := AutomationJobPreFirePayload{
			Schedule: &AutomationSchedulePayload{Mode: "cron", Expr: "0 * * * *"},
		}
		jobClone := cloneAsyncPayload(job)
		job.Schedule.Mode = "interval"
		if jobClone.Schedule.Mode != "cron" {
			t.Fatalf("cloned schedule mode = %q, want %q", jobClone.Schedule.Mode, "cron")
		}

		trigger := AutomationTriggerPreFirePayload{
			Payload: map[string]any{
				"outer":    map[string]any{"inner": "before"},
				"list":     []any{"before"},
				"typedMap": map[string][]string{"items": {"before"}},
				"typedList": []map[string]any{
					{"label": "before"},
				},
			},
		}
		triggerClone := cloneAsyncPayload(trigger)
		trigger.Payload["outer"].(map[string]any)["inner"] = "after"
		trigger.Payload["list"].([]any)[0] = "after"
		trigger.Payload["typedMap"].(map[string][]string)["items"][0] = "after"
		trigger.Payload["typedList"].([]map[string]any)[0]["label"] = "after"
		if triggerClone.Payload["outer"].(map[string]any)["inner"] != "before" {
			t.Fatalf("cloned nested payload = %#v, want preserved value", triggerClone.Payload["outer"])
		}
		if triggerClone.Payload["list"].([]any)[0] != "before" {
			t.Fatalf("cloned list payload = %#v, want preserved value", triggerClone.Payload["list"])
		}
		if triggerClone.Payload["typedMap"].(map[string][]string)["items"][0] != "before" {
			t.Fatalf("cloned typed map payload = %#v, want preserved value", triggerClone.Payload["typedMap"])
		}
		if triggerClone.Payload["typedList"].([]map[string]any)[0]["label"] != "before" {
			t.Fatalf("cloned typed list payload = %#v, want preserved value", triggerClone.Payload["typedList"])
		}
	})

	t.Run("agent payloads", func(t *testing.T) {
		t.Parallel()

		preStart := AgentPreStartPayload{Args: []string{"before"}}
		preStartClone := cloneAsyncPayload(preStart)
		preStart.Args[0] = "after"
		if preStartClone.Args[0] != "before" {
			t.Fatalf("cloned pre-start args = %#v, want preserved value", preStartClone.Args)
		}

		lifecycle := AgentLifecyclePayload{Args: []string{"before"}}
		lifecycleClone := cloneAsyncPayload(lifecycle)
		lifecycle.Args[0] = "after"
		if lifecycleClone.Args[0] != "before" {
			t.Fatalf("cloned lifecycle args = %#v, want preserved value", lifecycleClone.Args)
		}
	})

	t.Run("tool payloads", func(t *testing.T) {
		t.Parallel()

		preCall := ToolPreCallPayload{ToolInput: json.RawMessage(`{"tool":"before"}`)}
		preCallClone := cloneAsyncPayload(preCall)
		preCall.ToolInput[9] = 'X'
		if string(preCallClone.ToolInput) != `{"tool":"before"}` {
			t.Fatalf("cloned tool input = %s, want original payload", string(preCallClone.ToolInput))
		}

		postCall := ToolPostCallPayload{
			ToolInput:  json.RawMessage(`{"input":"before"}`),
			ToolResult: json.RawMessage(`{"result":"before"}`),
		}
		postCallClone := cloneAsyncPayload(postCall)
		postCall.ToolInput[10] = 'X'
		postCall.ToolResult[11] = 'X'
		if string(postCallClone.ToolInput) != `{"input":"before"}` {
			t.Fatalf("cloned post-call input = %s, want original payload", string(postCallClone.ToolInput))
		}
		if string(postCallClone.ToolResult) != `{"result":"before"}` {
			t.Fatalf("cloned post-call result = %s, want original payload", string(postCallClone.ToolResult))
		}

		postError := ToolPostErrorPayload{ToolInput: json.RawMessage(`{"error":"before"}`)}
		postErrorClone := cloneAsyncPayload(postError)
		postError.ToolInput[10] = 'X'
		if string(postErrorClone.ToolInput) != `{"error":"before"}` {
			t.Fatalf("cloned post-error input = %s, want original payload", string(postErrorClone.ToolInput))
		}
	})

	t.Run("permission payloads", func(t *testing.T) {
		t.Parallel()

		request := PermissionRequestPayload{
			ToolInput: json.RawMessage(`{"decision":"before"}`),
			ToolCall: PermissionToolCall{
				Locations: []ToolLocation{{Path: "/tmp/before"}},
			},
			Options: []PermissionOption{{Decision: "before"}},
		}
		requestClone := cloneAsyncPayload(request)
		request.ToolInput[13] = 'X'
		request.ToolCall.Locations[0].Path = "/tmp/after"
		request.Options[0].Decision = "after"
		if string(requestClone.ToolInput) != `{"decision":"before"}` {
			t.Fatalf("cloned permission input = %s, want original payload", string(requestClone.ToolInput))
		}
		if requestClone.ToolCall.Locations[0].Path != "/tmp/before" {
			t.Fatalf("cloned permission location = %q, want %q", requestClone.ToolCall.Locations[0].Path, "/tmp/before")
		}
		if requestClone.Options[0].Decision != "before" {
			t.Fatalf("cloned permission option = %q, want %q", requestClone.Options[0].Decision, "before")
		}

		resolution := PermissionResolutionPayload{
			ToolInput: json.RawMessage(`{"resolution":"before"}`),
			ToolCall: PermissionToolCall{
				Locations: []ToolLocation{{Path: "/tmp/resolution-before"}},
			},
		}
		resolutionClone := cloneAsyncPayload(resolution)
		resolution.ToolInput[15] = 'X'
		resolution.ToolCall.Locations[0].Path = "/tmp/resolution-after"
		if string(resolutionClone.ToolInput) != `{"resolution":"before"}` {
			t.Fatalf("cloned resolution input = %s, want original payload", string(resolutionClone.ToolInput))
		}
		if resolutionClone.ToolCall.Locations[0].Path != "/tmp/resolution-before" {
			t.Fatalf(
				"cloned resolution location = %q, want %q",
				resolutionClone.ToolCall.Locations[0].Path,
				"/tmp/resolution-before",
			)
		}
	})

	t.Run("default branch returns value", func(t *testing.T) {
		t.Parallel()

		original := SessionPreCreatePayload{SessionContext: SessionContext{SessionID: "session-default"}}
		cloned := cloneAsyncPayload(original)
		if cloned.SessionID != original.SessionID {
			t.Fatalf("cloned session id = %q, want %q", cloned.SessionID, original.SessionID)
		}
	})
}
