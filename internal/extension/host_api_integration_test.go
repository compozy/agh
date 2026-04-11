//go:build integration

package extension

import (
	"context"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func withHostAPIHooks(hooks *hookspkg.Hooks) hostAPITestEnvOption {
	return func(cfg *hostAPITestEnvConfig) {
		cfg.hooks = hooks
	}
}

func (d *hostAPIFakeDriver) promptCalls() []acp.PromptRequest {
	d.mu.Lock()
	defer d.mu.Unlock()

	return append([]acp.PromptRequest(nil), d.promptLog...)
}

func TestHostAPIIntegrationSessionLifecycleThroughHostAPI(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.grant(
		"ext-integration",
		[]string{"sessions/create", "sessions/prompt", "sessions/status", "sessions/events"},
		[]string{"session.write", "session.read"},
	)

	createResult, err := env.call(t, "ext-integration", "sessions/create", map[string]string{
		"agent":     "coder",
		"workspace": env.workspaceID,
	})
	if err != nil {
		t.Fatalf("Handle(sessions/create) error = %v", err)
	}

	var created hostAPISessionCreateResult
	decodeResult(t, createResult, &created)
	if created.SessionID == "" {
		t.Fatal("sessions/create session_id = empty, want non-empty")
	}

	prompt, err := env.submitPrompt(t, "ext-integration", created.SessionID, "integration prompt")
	if err != nil {
		t.Fatalf("submitPrompt() error = %v", err)
	}
	if prompt.TurnID == "" {
		t.Fatal("sessions/prompt turn_id = empty, want non-empty")
	}

	statusResult, err := env.call(t, "ext-integration", "sessions/status", map[string]string{"session_id": created.SessionID})
	if err != nil {
		t.Fatalf("Handle(sessions/status) error = %v", err)
	}

	var status hostAPISessionStatus
	decodeResult(t, statusResult, &status)
	if status.State == "" {
		t.Fatal("sessions/status state = empty, want non-empty")
	}

	eventsResult, err := env.call(t, "ext-integration", "sessions/events", map[string]any{
		"session_id": created.SessionID,
		"turn_id":    prompt.TurnID,
		"limit":      10,
	})
	if err != nil {
		t.Fatalf("Handle(sessions/events) error = %v", err)
	}

	var events []hostAPISessionEvent
	decodeResult(t, eventsResult, &events)
	if len(events) == 0 {
		t.Fatal("sessions/events len = 0, want prompt events")
	}
}

func TestHostAPIIntegrationStoresAndRecallsMemory(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.grant("ext-integration", []string{"memory/store", "memory/recall"}, []string{"memory.write", "memory.read"})

	if _, err := env.call(t, "ext-integration", "memory/store", map[string]any{
		"key":     "deploy-checklist",
		"content": "Run smoke tests before deploy",
		"tags":    []string{"reference", "deploy"},
	}); err != nil {
		t.Fatalf("Handle(memory/store) error = %v", err)
	}

	result, err := env.call(t, "ext-integration", "memory/recall", map[string]any{
		"query": "what should I do before deploy",
		"limit": 5,
	})
	if err != nil {
		t.Fatalf("Handle(memory/recall) error = %v", err)
	}

	var entries []hostAPIMemoryRecallEntry
	decodeResult(t, result, &entries)
	if len(entries) == 0 {
		t.Fatal("memory/recall len = 0, want stored memory")
	}
}

func TestHostAPIIntegrationUnauthorizedExtensionIsDeniedForEveryMethod(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.grant("ext-denied", nil, nil)

	session := env.createSession(t)
	tests := []struct {
		method string
		params any
	}{
		{method: "sessions/list", params: map[string]any{"workspace": env.workspaceID}},
		{method: "sessions/create", params: map[string]any{"agent": "coder", "workspace": env.workspaceID}},
		{method: "sessions/prompt", params: map[string]any{"session_id": session.ID, "message": "hello"}},
		{method: "sessions/stop", params: map[string]any{"session_id": session.ID}},
		{method: "sessions/status", params: map[string]any{"session_id": session.ID}},
		{method: "sessions/events", params: map[string]any{"session_id": session.ID, "limit": 1}},
		{method: "memory/recall", params: map[string]any{"query": "needle"}},
		{method: "memory/store", params: map[string]any{"key": "note", "content": "body"}},
		{method: "memory/forget", params: map[string]any{"key": "note"}},
		{method: "observe/health", params: nil},
		{method: "observe/events", params: map[string]any{"session_id": session.ID, "limit": 1}},
		{method: "skills/list", params: map[string]any{"workspace": env.workspaceID}},
		{method: "automation/jobs", params: map[string]any{"scope": "workspace", "workspace_id": env.workspaceID}},
		{method: "automation/jobs/create", params: map[string]any{
			"name":         "integration-job",
			"scope":        "workspace",
			"workspace_id": env.workspaceID,
			"agent_name":   "coder",
			"prompt":       "run integration job",
			"schedule": map[string]any{
				"mode":     "every",
				"interval": "5m",
			},
		}},
		{method: "automation/triggers/fire", params: map[string]any{
			"event":        "ext.github.push",
			"scope":        "workspace",
			"workspace_id": env.workspaceID,
		}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.method, func(t *testing.T) {
			_, err := env.call(t, "ext-denied", tt.method, tt.params)
			assertCapabilityDenied(t, err, tt.method)
		})
	}
}

func TestHostAPIIntegrationAutomationJobCreateReturnsCreatedJobPayload(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.grant("ext-automation", []string{"automation/jobs/create"}, []string{"automation.write"})

	result, err := env.call(t, "ext-automation", "automation/jobs/create", map[string]any{
		"name":         "nightly-report",
		"scope":        "workspace",
		"workspace_id": env.workspaceID,
		"agent_name":   "coder",
		"prompt":       "Generate nightly report",
		"schedule": map[string]any{
			"mode":     "every",
			"interval": "5m",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/create) error = %v", err)
	}

	var created automationpkg.Job
	decodeResult(t, result, &created)
	if created.ID == "" {
		t.Fatal("automation/jobs/create id = empty, want non-empty")
	}
	if created.Name != "nightly-report" {
		t.Fatalf("automation/jobs/create name = %q, want nightly-report", created.Name)
	}
	if created.Source != automationpkg.JobSourceDynamic {
		t.Fatalf("automation/jobs/create source = %q, want %q", created.Source, automationpkg.JobSourceDynamic)
	}
}

func TestHostAPIIntegrationAutomationTriggerFireDispatchesThroughTriggerEngine(t *testing.T) {
	env := newHostAPITestEnv(t)
	env.grant(
		"ext-automation",
		[]string{"automation/triggers/create", "automation/triggers/fire"},
		[]string{"automation.write"},
	)

	createResult, err := env.call(t, "ext-automation", "automation/triggers/create", map[string]any{
		"name":         "review-push",
		"scope":        "workspace",
		"workspace_id": env.workspaceID,
		"agent_name":   "coder",
		"event":        "ext.github.push",
		"prompt":       `Review push to {{ index .Data "repo" }} by {{ index .Data "author" }}`,
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/create) error = %v", err)
	}

	var trigger automationpkg.Trigger
	decodeResult(t, createResult, &trigger)
	if trigger.ID == "" {
		t.Fatal("automation/triggers/create id = empty, want non-empty")
	}

	fireResult, err := env.call(t, "ext-automation", "automation/triggers/fire", map[string]any{
		"event":        "ext.github.push",
		"scope":        "workspace",
		"workspace_id": env.workspaceID,
		"payload": map[string]any{
			"repo":   "acme/api",
			"author": "dev@acme.com",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/fire) error = %v", err)
	}

	var result automationpkg.TriggerResult
	decodeResult(t, fireResult, &result)
	if result.Matched != 1 {
		t.Fatalf("automation/triggers/fire matched = %d, want 1", result.Matched)
	}
	if len(result.Runs) != 1 {
		t.Fatalf("automation/triggers/fire runs = %d, want 1", len(result.Runs))
	}

	prompts := env.driver.promptCalls()
	if len(prompts) == 0 {
		t.Fatal("driver prompt calls = 0, want trigger dispatch prompt")
	}
	if got, want := prompts[len(prompts)-1].Message, "Review push to acme/api by dev@acme.com"; got != want {
		t.Fatalf("last prompt message = %q, want %q", got, want)
	}
}

func TestHostAPIIntegrationAutomationPreFireHookMutatesPrompt(t *testing.T) {
	hooks := hookspkg.NewHooks(
		hookspkg.WithNativeDeclarations([]hookspkg.HookDecl{{
			Name:         "mutate-automation-prompt",
			Event:        hookspkg.HookAutomationJobPreFire,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}}),
		hookspkg.WithExecutorResolver(func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
			return hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.AutomationJobPreFirePayload) (hookspkg.AutomationFirePatch, error) {
				prompt := payload.Prompt + " with hook mutation"
				return hookspkg.AutomationFirePatch{Prompt: &prompt}, nil
			}), nil
		}),
	)
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("hooks.Rebuild() error = %v", err)
	}
	t.Cleanup(hooks.Close)

	env := newHostAPITestEnv(t, withHostAPIHooks(hooks))
	env.grant(
		"ext-automation",
		[]string{"automation/jobs/create", "automation/jobs/trigger"},
		[]string{"automation.write"},
	)

	createResult, err := env.call(t, "ext-automation", "automation/jobs/create", map[string]any{
		"name":         "hooked-job",
		"scope":        "workspace",
		"workspace_id": env.workspaceID,
		"agent_name":   "coder",
		"prompt":       "Original prompt",
		"schedule": map[string]any{
			"mode":     "every",
			"interval": "5m",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/create) error = %v", err)
	}

	var created automationpkg.Job
	decodeResult(t, createResult, &created)

	if _, err := env.call(t, "ext-automation", "automation/jobs/trigger", map[string]any{"id": created.ID}); err != nil {
		t.Fatalf("Handle(automation/jobs/trigger) error = %v", err)
	}

	prompts := env.driver.promptCalls()
	if len(prompts) == 0 {
		t.Fatal("driver prompt calls = 0, want job dispatch prompt")
	}
	if got, want := prompts[len(prompts)-1].Message, "Original prompt with hook mutation"; got != want {
		t.Fatalf("last prompt message = %q, want %q", got, want)
	}
}
