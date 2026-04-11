package contract_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func TestSessionPayloadJSONShape(t *testing.T) {
	t.Run("Should preserve session payload JSON shape", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
		payload := core.SessionPayloadFromInfo(&session.SessionInfo{
			ID:           "sess-1",
			Name:         "demo",
			AgentName:    "coder",
			WorkspaceID:  "ws_alpha",
			Workspace:    "/workspace",
			State:        session.StateActive,
			ACPSessionID: "acp-123",
			CreatedAt:    now,
			UpdatedAt:    now,
			ACPCaps: acp.ACPCaps{
				SupportsLoadSession: true,
				SupportedModes:      []string{"chat"},
				SupportedModels:     []string{"gpt-test"},
			},
		})

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["agent_name"] != "coder" || got["workspace_id"] != "ws_alpha" || got["workspace_path"] != "/workspace" {
			t.Fatalf("session JSON = %#v", got)
		}
		if _, exists := got["stop_reason"]; exists {
			t.Fatalf("session JSON should omit empty stop_reason: %#v", got)
		}
		if _, exists := got["stop_detail"]; exists {
			t.Fatalf("session JSON should omit empty stop_detail: %#v", got)
		}
		if _, exists := got["acp_session_id"]; !exists {
			t.Fatalf("session JSON missing acp_session_id: %#v", got)
		}
		acpCaps, ok := got["acp_caps"].(map[string]any)
		if !ok {
			t.Fatalf("acp_caps type = %T, want object", got["acp_caps"])
		}
		if acpCaps["supports_load_session"] != true {
			t.Fatalf("acp_caps JSON = %#v", acpCaps)
		}
	})
}

func TestSessionPayloadJSONIncludesSessionStopFields(t *testing.T) {
	t.Run("Should include session stop fields in JSON", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
		payload := core.SessionPayloadFromInfo(&session.SessionInfo{
			ID:          "sess-stopped",
			Name:        "demo",
			AgentName:   "coder",
			WorkspaceID: "ws_alpha",
			Workspace:   "/workspace",
			State:       session.StateStopped,
			StopReason:  store.StopUserCanceled,
			StopDetail:  "requested by API",
			CreatedAt:   now,
			UpdatedAt:   now,
		})

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["stop_reason"] != string(store.StopUserCanceled) {
			t.Fatalf("stop_reason = %#v, want %q", got["stop_reason"], store.StopUserCanceled)
		}
		if got["stop_detail"] != "requested by API" {
			t.Fatalf("stop_detail = %#v, want %q", got["stop_detail"], "requested by API")
		}
	})
}

func TestWorkspacePayloadPreservesOmitEmptyBehavior(t *testing.T) {
	t.Run("Should preserve workspace omit-empty behavior", func(t *testing.T) {
		t.Parallel()

		payload := contract.WorkspacePayload{
			ID:        "ws_alpha",
			RootDir:   "/workspace",
			AddDirs:   []string{},
			Name:      "alpha",
			CreatedAt: time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 7, 11, 30, 0, 0, time.UTC),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if _, exists := got["default_agent"]; exists {
			t.Fatalf("default_agent should be omitted: %#v", got)
		}
		addDirs, ok := got["add_dirs"].([]any)
		if !ok {
			t.Fatalf("add_dirs type = %T, want array", got["add_dirs"])
		}
		if len(addDirs) != 0 {
			t.Fatalf("add_dirs length = %d, want 0", len(addDirs))
		}
	})
}

func TestAgentEventPayloadRoundTripsThroughJSON(t *testing.T) {
	t.Parallel()

	inputTokens := int64(12)
	event := acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "sess-1",
		TurnID:    "turn-1",
		RequestID: "req-1",
		Timestamp: time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC),
		Action:    "fs/read_text_file",
		Resource:  "/tmp/file.txt",
		Decision:  "pending",
		Error:     "",
		Usage: &acp.TokenUsage{
			TurnID:      "turn-1",
			InputTokens: &inputTokens,
			Timestamp:   time.Date(2026, 4, 7, 10, 30, 1, 0, time.UTC),
		},
		Raw: []byte(`{"ok":true}`),
	}

	payload := core.AgentEventPayloadFromEvent(event)
	var roundTrip contract.AgentEventPayload
	marshalJSON(t, payload, &roundTrip)

	if roundTrip.Type != event.Type || roundTrip.RequestID != event.RequestID || roundTrip.Action != event.Action {
		t.Fatalf("roundTrip payload = %#v", roundTrip)
	}
	if roundTrip.Usage == nil || roundTrip.Usage.InputTokens == nil || *roundTrip.Usage.InputTokens != inputTokens {
		t.Fatalf("usage payload = %#v", roundTrip.Usage)
	}
	if string(roundTrip.Raw) != `{"ok":true}` {
		t.Fatalf("raw payload = %s", string(roundTrip.Raw))
	}
}

func TestAutomationJobPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve automation job JSON shape", func(t *testing.T) {
		t.Parallel()

		nextRun := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
		payload := contract.JobPayload{
			ID:          "job-1",
			Scope:       automationpkg.AutomationScopeWorkspace,
			Name:        "nightly-review",
			AgentName:   "coder",
			WorkspaceID: "ws-alpha",
			Prompt:      "review repo",
			Schedule: &automationpkg.ScheduleSpec{
				Mode:     automationpkg.ScheduleModeEvery,
				Interval: "1h",
			},
			Enabled: true,
			Retry: automationpkg.RetryConfig{
				Strategy:   automationpkg.RetryStrategyBackoff,
				MaxRetries: 2,
				BaseDelay:  "1m",
			},
			FireLimit: automationpkg.FireLimitConfig{
				Max:    5,
				Window: "24h",
			},
			Source:    automationpkg.JobSourceDynamic,
			CreatedAt: time.Date(2026, 4, 11, 11, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 11, 11, 30, 0, 0, time.UTC),
			NextRun:   &nextRun,
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["scope"] != string(automationpkg.AutomationScopeWorkspace) {
			t.Fatalf("scope = %#v, want %q", got["scope"], automationpkg.AutomationScopeWorkspace)
		}
		if got["workspace_id"] != "ws-alpha" {
			t.Fatalf("workspace_id = %#v, want %q", got["workspace_id"], "ws-alpha")
		}
		if got["source"] != string(automationpkg.JobSourceDynamic) {
			t.Fatalf("source = %#v, want %q", got["source"], automationpkg.JobSourceDynamic)
		}
		if _, exists := got["next_run"]; !exists {
			t.Fatalf("job payload missing next_run: %#v", got)
		}
	})
}

func TestAutomationTriggerPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve automation trigger JSON shape", func(t *testing.T) {
		t.Parallel()

		payload := contract.TriggerPayload{
			ID:           "trigger-1",
			Scope:        automationpkg.AutomationScopeWorkspace,
			Name:         "deploy-review",
			AgentName:    "coder",
			WorkspaceID:  "ws-alpha",
			Prompt:       `review {{ index .Data "payload" }}`,
			Event:        "webhook",
			Filter:       map[string]string{"branch": "main"},
			Enabled:      true,
			Retry:        automationpkg.DefaultRetryConfig(),
			FireLimit:    automationpkg.DefaultFireLimitConfig(),
			Source:       automationpkg.JobSourceDynamic,
			WebhookID:    "wbh_123",
			EndpointSlug: "deploy-review",
			CreatedAt:    time.Date(2026, 4, 11, 11, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2026, 4, 11, 11, 30, 0, 0, time.UTC),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["scope"] != string(automationpkg.AutomationScopeWorkspace) {
			t.Fatalf("scope = %#v, want %q", got["scope"], automationpkg.AutomationScopeWorkspace)
		}
		if got["workspace_id"] != "ws-alpha" {
			t.Fatalf("workspace_id = %#v, want %q", got["workspace_id"], "ws-alpha")
		}
		if got["source"] != string(automationpkg.JobSourceDynamic) {
			t.Fatalf("source = %#v, want %q", got["source"], automationpkg.JobSourceDynamic)
		}
		if got["endpoint_slug"] != "deploy-review" {
			t.Fatalf("endpoint_slug = %#v, want %q", got["endpoint_slug"], "deploy-review")
		}
		if got["webhook_id"] != "wbh_123" {
			t.Fatalf("webhook_id = %#v, want %q", got["webhook_id"], "wbh_123")
		}
	})
}

func TestAutomationUpdateRequestsHasChanges(t *testing.T) {
	t.Parallel()

	name := "updated"
	agentName := "reviewer"
	workspaceID := "ws-alpha"
	prompt := "updated prompt"
	schedule := automationpkg.ScheduleSpec{
		Mode:     automationpkg.ScheduleModeEvery,
		Interval: "1h",
	}
	retry := automationpkg.RetryConfig{
		Strategy:   automationpkg.RetryStrategyBackoff,
		MaxRetries: 2,
		BaseDelay:  "1m",
	}
	fireLimit := automationpkg.FireLimitConfig{
		Max:    3,
		Window: "15m",
	}
	event := "session.created"
	filter := map[string]string{"kind": "session"}
	webhookID := "wbh_123"
	endpointSlug := "deploy-review"
	secret := "secret"
	disabled := false

	t.Run("Should report changes for automation job update requests", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			req  contract.UpdateJobRequest
			want bool
		}{
			{
				name: "Should return false for an empty job update",
				req:  contract.UpdateJobRequest{},
				want: false,
			},
			{
				name: "Should return true when the job name is set",
				req:  contract.UpdateJobRequest{Name: &name},
				want: true,
			},
			{
				name: "Should return true when the job agent name is set",
				req:  contract.UpdateJobRequest{AgentName: &agentName},
				want: true,
			},
			{
				name: "Should return true when the job workspace id is set",
				req:  contract.UpdateJobRequest{WorkspaceID: &workspaceID},
				want: true,
			},
			{
				name: "Should return true when the job prompt is set",
				req:  contract.UpdateJobRequest{Prompt: &prompt},
				want: true,
			},
			{
				name: "Should return true when the job schedule is set",
				req:  contract.UpdateJobRequest{Schedule: &schedule},
				want: true,
			},
			{
				name: "Should return true when the job enabled flag is set",
				req:  contract.UpdateJobRequest{Enabled: &disabled},
				want: true,
			},
			{
				name: "Should return true when the job retry policy is set",
				req:  contract.UpdateJobRequest{Retry: &retry},
				want: true,
			},
			{
				name: "Should return true when the job fire limit is set",
				req:  contract.UpdateJobRequest{FireLimit: &fireLimit},
				want: true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if got := tc.req.HasChanges(); got != tc.want {
					t.Fatalf("UpdateJobRequest.HasChanges() = %v, want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("Should report changes for automation trigger update requests", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name string
			req  contract.UpdateTriggerRequest
			want bool
		}{
			{
				name: "Should return false for an empty trigger update",
				req:  contract.UpdateTriggerRequest{},
				want: false,
			},
			{
				name: "Should return true when the trigger name is set",
				req:  contract.UpdateTriggerRequest{Name: &name},
				want: true,
			},
			{
				name: "Should return true when the trigger agent name is set",
				req:  contract.UpdateTriggerRequest{AgentName: &agentName},
				want: true,
			},
			{
				name: "Should return true when the trigger workspace id is set",
				req:  contract.UpdateTriggerRequest{WorkspaceID: &workspaceID},
				want: true,
			},
			{
				name: "Should return true when the trigger prompt is set",
				req:  contract.UpdateTriggerRequest{Prompt: &prompt},
				want: true,
			},
			{
				name: "Should return true when the trigger event is set",
				req:  contract.UpdateTriggerRequest{Event: &event},
				want: true,
			},
			{
				name: "Should return true when the trigger filter is set",
				req:  contract.UpdateTriggerRequest{Filter: filter},
				want: true,
			},
			{
				name: "Should return true when the webhook secret is set",
				req:  contract.UpdateTriggerRequest{WebhookSecret: &secret},
				want: true,
			},
			{
				name: "Should return true when the trigger enabled flag is set",
				req:  contract.UpdateTriggerRequest{Enabled: &disabled},
				want: true,
			},
			{
				name: "Should return true when the trigger retry policy is set",
				req:  contract.UpdateTriggerRequest{Retry: &retry},
				want: true,
			},
			{
				name: "Should return true when the trigger fire limit is set",
				req:  contract.UpdateTriggerRequest{FireLimit: &fireLimit},
				want: true,
			},
			{
				name: "Should return true when the trigger webhook id is set",
				req:  contract.UpdateTriggerRequest{WebhookID: &webhookID},
				want: true,
			},
			{
				name: "Should return true when the trigger endpoint slug is set",
				req:  contract.UpdateTriggerRequest{EndpointSlug: &endpointSlug},
				want: true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if got := tc.req.HasChanges(); got != tc.want {
					t.Fatalf("UpdateTriggerRequest.HasChanges() = %v, want %v", got, tc.want)
				}
			})
		}
	})
}

func marshalJSON[T any](t *testing.T, value any, target *T) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}
