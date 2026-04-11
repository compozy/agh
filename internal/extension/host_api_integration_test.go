//go:build integration

package extension

import "testing"

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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.method, func(t *testing.T) {
			_, err := env.call(t, "ext-denied", tt.method, tt.params)
			assertCapabilityDenied(t, err, tt.method)
		})
	}
}
