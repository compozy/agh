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
	taskpkg "github.com/pedronauck/agh/internal/task"
)

func TestSessionPayloadJSONShape(t *testing.T) {
	t.Run("Should preserve session payload JSON shape", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
		payload := core.SessionPayloadFromInfo(&session.Info{
			ID:           "sess-1",
			Name:         "demo",
			AgentName:    "coder",
			Provider:     "fake",
			WorkspaceID:  "ws_alpha",
			Workspace:    "/workspace",
			State:        session.StateActive,
			ACPSessionID: "acp-123",
			Environment: &store.SessionEnvironmentMeta{
				EnvironmentID: "env-json",
				Backend:       "local",
				Profile:       "local",
				State:         "prepared",
				InstanceID:    "instance-json",
			},
			CreatedAt: now,
			UpdatedAt: now,
			ACPCaps: acp.Caps{
				SupportsLoadSession: true,
				SupportedModes:      []string{"chat"},
				SupportedModels:     []string{"gpt-test"},
			},
		})

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["agent_name"] != "coder" ||
			got["provider"] != "fake" ||
			got["workspace_id"] != "ws_alpha" ||
			got["workspace_path"] != "/workspace" {
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
		environmentPayload, ok := got["environment"].(map[string]any)
		if !ok {
			t.Fatalf("environment type = %T, want object", got["environment"])
		}
		if environmentPayload["environment_id"] != "env-json" ||
			environmentPayload["backend"] != "local" ||
			environmentPayload["instance_id"] != "instance-json" {
			t.Fatalf("environment JSON = %#v", environmentPayload)
		}
	})
}

func TestCreateSessionRequestJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should decode optional provider when present", func(t *testing.T) {
		t.Parallel()

		var req contract.CreateSessionRequest
		if err := json.Unmarshal(
			[]byte(`{"agent_name":"coder","provider":"fake","workspace":"alpha"}`),
			&req,
		); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if req.AgentName != "coder" || req.Provider != "fake" || req.Workspace != "alpha" {
			t.Fatalf("request = %#v", req)
		}
	})

	t.Run("Should omit provider cleanly when absent", func(t *testing.T) {
		t.Parallel()

		var req contract.CreateSessionRequest
		if err := json.Unmarshal(
			[]byte(`{"agent_name":"coder","workspace_path":"/workspace"}`),
			&req,
		); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if req.Provider != "" {
			t.Fatalf("request.Provider = %q, want empty", req.Provider)
		}
		if req.WorkspacePath != "/workspace" {
			t.Fatalf("request = %#v", req)
		}
	})
}

func TestSessionPayloadJSONIncludesSessionStopFields(t *testing.T) {
	t.Run("Should include session stop fields in JSON", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
		payload := core.SessionPayloadFromInfo(&session.Info{
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

func TestWorkspaceEnvironmentRefJSONFields(t *testing.T) {
	t.Parallel()

	t.Run("Should serialize create workspace environment_ref", func(t *testing.T) {
		t.Parallel()

		payload := contract.CreateWorkspaceRequest{
			RootDir:        "/workspace",
			EnvironmentRef: "daytona-dev",
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["environment_ref"] != "daytona-dev" {
			t.Fatalf("environment_ref = %#v, want daytona-dev", got["environment_ref"])
		}
	})

	t.Run("Should include workspace payload environment_ref", func(t *testing.T) {
		t.Parallel()

		payload := contract.WorkspacePayload{
			ID:             "ws_alpha",
			RootDir:        "/workspace",
			AddDirs:        []string{},
			Name:           "alpha",
			EnvironmentRef: "daytona-dev",
			CreatedAt:      time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 4, 7, 11, 30, 0, 0, time.UTC),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["environment_ref"] != "daytona-dev" {
			t.Fatalf("environment_ref = %#v, want daytona-dev", got["environment_ref"])
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
			Task: &automationpkg.JobTaskConfig{
				Title:          "Review findings",
				NetworkChannel: "ops-automation",
				Owner: &taskpkg.Ownership{
					Kind: taskpkg.OwnerKindAutomation,
					Ref:  "rule:nightly-review",
				},
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
		taskValue, ok := got["task"].(map[string]any)
		if !ok || taskValue["title"] != "Review findings" || taskValue["network_channel"] != "ops-automation" {
			t.Fatalf("task = %#v, want populated task config", got["task"])
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
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if got := tc.req.HasChanges(); got != tc.want {
					t.Fatalf("UpdateTriggerRequest.HasChanges() = %v, want %v", got, tc.want)
				}
			})
		}
	})
}

func TestTaskPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should marshal task payload JSON shape", func(t *testing.T) {
		t.Parallel()

		payload := contract.TaskPayload{
			ID:             "task-1",
			Identifier:     "TASK-1",
			Scope:          taskpkg.ScopeWorkspace,
			WorkspaceID:    "ws-alpha",
			ParentTaskID:   "task-root",
			NetworkChannel: "builders",
			Title:          "Review task",
			Description:    "Check the API layer",
			Status:         taskpkg.TaskStatusInProgress,
			Owner:          &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"},
			CreatedBy:      taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			CreatedAt:      time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 4, 14, 10, 5, 0, 0, time.UTC),
			Metadata:       json.RawMessage(`{"priority":"high"}`),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["workspace_id"] != "ws-alpha" || got["network_channel"] != "builders" {
			t.Fatalf("task JSON = %#v", got)
		}
		createdBy, ok := got["created_by"].(map[string]any)
		if !ok || createdBy["kind"] != string(taskpkg.ActorKindHuman) || createdBy["ref"] != "local-user" {
			t.Fatalf("created_by JSON = %#v", got["created_by"])
		}
		origin, ok := got["origin"].(map[string]any)
		if !ok || origin["kind"] != string(taskpkg.OriginKindHTTP) || origin["ref"] != "tasks.create" {
			t.Fatalf("origin JSON = %#v", got["origin"])
		}
		owner, ok := got["owner"].(map[string]any)
		if !ok || owner["kind"] != string(taskpkg.OwnerKindPool) || owner["ref"] != "reviewers" {
			t.Fatalf("owner JSON = %#v", got["owner"])
		}
		if _, exists := got["metadata"]; !exists {
			t.Fatalf("task JSON missing metadata: %#v", got)
		}
	})

	t.Run("Should omit zero-valued optional task timestamps", func(t *testing.T) {
		t.Parallel()

		payload := contract.TaskPayload{
			ID:        "task-1",
			Scope:     taskpkg.ScopeGlobal,
			Title:     "Review task",
			Status:    taskpkg.TaskStatusReady,
			CreatedBy: taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			Origin:    taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.create"},
			CreatedAt: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 14, 10, 5, 0, 0, time.UTC),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if _, exists := got["closed_at"]; exists {
			t.Fatalf("task JSON unexpectedly included closed_at: %#v", got)
		}
	})
}

func TestTaskRunPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should marshal task run payload JSON shape", func(t *testing.T) {
		t.Parallel()

		startedAt := time.Date(2026, 4, 14, 10, 1, 0, 0, time.UTC)
		payload := contract.TaskRunPayload{
			ID:             "run-1",
			TaskID:         "task-1",
			Status:         taskpkg.TaskRunStatusRunning,
			Attempt:        2,
			ClaimedBy:      &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "local-user"},
			SessionID:      "sess-1",
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.start_run"},
			IdempotencyKey: "key-1",
			NetworkChannel: "builders",
			QueuedAt:       time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
			StartedAt:      &startedAt,
			Result:         json.RawMessage(`{"ok":true}`),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		if got["session_id"] != "sess-1" || got["idempotency_key"] != "key-1" {
			t.Fatalf("task run JSON = %#v", got)
		}
		if got["network_channel"] != "builders" || got["status"] != string(taskpkg.TaskRunStatusRunning) {
			t.Fatalf("task run JSON = %#v", got)
		}
	})

	t.Run("Should omit zero-valued optional run timestamps", func(t *testing.T) {
		t.Parallel()

		payload := contract.TaskRunPayload{
			ID:       "run-1",
			TaskID:   "task-1",
			Status:   taskpkg.TaskRunStatusQueued,
			Attempt:  1,
			Origin:   taskpkg.Origin{Kind: taskpkg.OriginKindHTTP, Ref: "tasks.enqueue_run"},
			QueuedAt: time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC),
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		for _, field := range []string{"claimed_at", "started_at", "ended_at"} {
			if _, exists := got[field]; exists {
				t.Fatalf("task run JSON unexpectedly included %s: %#v", field, got)
			}
		}
	})
}

func TestUpdateTaskRequestHasChanges(t *testing.T) {
	t.Parallel()

	title := "updated"
	channel := "builders"
	owner := &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "reviewers"}
	metadata := json.RawMessage(`{"priority":"high"}`)

	testCases := []struct {
		name string
		req  contract.UpdateTaskRequest
		want bool
	}{
		{name: "Should return false when no task changes are set", req: contract.UpdateTaskRequest{}, want: false},
		{name: "Should return true when title is set", req: contract.UpdateTaskRequest{Title: &title}, want: true},
		{
			name: "Should return true when network channel is set",
			req:  contract.UpdateTaskRequest{NetworkChannel: &channel},
			want: true,
		},
		{name: "Should return true when owner is set", req: contract.UpdateTaskRequest{Owner: owner}, want: true},
		{
			name: "Should return true when metadata is set",
			req:  contract.UpdateTaskRequest{Metadata: &metadata},
			want: true,
		},
		{
			name: "Should return true when clear owner is set",
			req:  contract.UpdateTaskRequest{ClearOwner: true},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.req.HasChanges(); got != tc.want {
				t.Fatalf("UpdateTaskRequest.HasChanges() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNetworkPeerPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should serialize peer-card capabilities as typed brief objects", func(t *testing.T) {
		t.Parallel()

		payload := contract.NetworkPeerPayload{
			PeerID:      "reviewer.sess-a",
			DisplayName: "Reviewer",
			Channel:     "builders",
			Local:       true,
			PeerCard: contract.NetworkPeerCardPayload{
				PeerID: "reviewer.sess-a",
				Capabilities: []contract.NetworkCapabilityBriefPayload{{
					ID:      "review-pr",
					Summary: "Review pull requests",
				}},
				ProfilesSupported:   []string{"agh-network/v0"},
				ArtifactsSupported:  []string{"capability"},
				TrustModesSupported: []string{"untrusted"},
				Ext: map[string]json.RawMessage{
					"agh.workflow_id": json.RawMessage(`"wf-1"`),
				},
			},
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		peerCard, ok := got["peer_card"].(map[string]any)
		if !ok {
			t.Fatalf("peer_card type = %T, want object", got["peer_card"])
		}
		capabilities, ok := peerCard["capabilities"].([]any)
		if !ok || len(capabilities) != 1 {
			t.Fatalf("peer_card.capabilities = %#v, want one object entry", peerCard["capabilities"])
		}
		firstCapability, ok := capabilities[0].(map[string]any)
		if !ok {
			t.Fatalf("capability entry type = %T, want object", capabilities[0])
		}
		if firstCapability["id"] != "review-pr" || firstCapability["summary"] != "Review pull requests" {
			t.Fatalf("capability brief JSON = %#v", firstCapability)
		}
		if _, isString := capabilities[0].(string); isString {
			t.Fatalf("capability brief JSON should be object, got string: %#v", capabilities[0])
		}
	})
}

func TestNetworkPeerDetailPayloadJSONShape(t *testing.T) {
	t.Parallel()

	t.Run("Should serialize rich capability catalogs as structured payloads", func(t *testing.T) {
		t.Parallel()

		payload := contract.NetworkPeerDetailPayload{
			PeerID:      "reviewer.sess-a",
			DisplayName: "Reviewer",
			Channel:     "builders",
			Local:       true,
			PeerCard: contract.NetworkPeerCardPayload{
				PeerID: "reviewer.sess-a",
				Capabilities: []contract.NetworkCapabilityBriefPayload{{
					ID:      "review-pr",
					Summary: "Review pull requests",
				}},
				ProfilesSupported:   []string{"agh-network/v0"},
				ArtifactsSupported:  []string{"capability"},
				TrustModesSupported: []string{"untrusted"},
			},
			CapabilityCatalog: &contract.NetworkCapabilityCatalogPayload{
				Capabilities: []contract.NetworkCapabilityPayload{{
					ID:                "review-pr",
					Summary:           "Review pull requests",
					Outcome:           "Actionable review findings",
					Version:           "1.0.0",
					Digest:            "sha256:review-pr-v1",
					ContextNeeded:     []string{"pull request link"},
					ArtifactsExpected: []string{"review summary"},
					Requirements:      []string{"workspace-read"},
				}},
			},
		}

		var got map[string]any
		marshalJSON(t, payload, &got)

		catalog, ok := got["capability_catalog"].(map[string]any)
		if !ok {
			t.Fatalf("capability_catalog type = %T, want object", got["capability_catalog"])
		}
		capabilities, ok := catalog["capabilities"].([]any)
		if !ok || len(capabilities) != 1 {
			t.Fatalf("capability_catalog.capabilities = %#v, want one object entry", catalog["capabilities"])
		}
		firstCapability, ok := capabilities[0].(map[string]any)
		if !ok {
			t.Fatalf("rich capability entry type = %T, want object", capabilities[0])
		}
		if firstCapability["digest"] != "sha256:review-pr-v1" ||
			firstCapability["outcome"] != "Actionable review findings" {
			t.Fatalf("rich capability JSON = %#v", firstCapability)
		}
		requirements, ok := firstCapability["requirements"].([]any)
		if !ok || len(requirements) != 1 || requirements[0] != "workspace-read" {
			t.Fatalf("requirements JSON = %#v, want workspace-read", firstCapability["requirements"])
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
