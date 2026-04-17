package acpmock

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestDriverStreamsStablePermissionAndToolSequence(t *testing.T) {
	t.Parallel()

	nodePath, err := ResolveNodePath()
	if err != nil {
		t.Skipf("ResolveNodePath() error = %v", err)
	}

	driverPath, err := DefaultDriverPath()
	if err != nil {
		t.Fatalf("DefaultDriverPath() error = %v", err)
	}
	fixturePath, err := filepath.Abs(filepath.Join("testdata", "tool_permission_fixture.json"))
	if err != nil {
		t.Fatalf("filepath.Abs(fixture) error = %v", err)
	}

	command := BuildCommand(
		nodePath,
		driverPath,
		fixturePath,
		"golden",
		filepath.Join(t.TempDir(), "golden-diagnostics.jsonl"),
	)

	driver := acp.New(acp.WithPermissionTimeout(2 * time.Second))
	proc, err := driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName:   "golden",
		Command:     command,
		Cwd:         t.TempDir(),
		Permissions: aghconfig.PermissionModeDenyAll,
	})
	if err != nil {
		t.Fatalf("driver.Start() error = %v", err)
	}
	defer stopDriverProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, acp.PromptRequest{
		TurnID:  "turn-tool-permission",
		Message: "exercise golden",
	})
	if err != nil {
		t.Fatalf("driver.Prompt() error = %v", err)
	}

	events := collectPromptEvents(t, eventsCh, func(event acp.AgentEvent) {
		if event.Type == acp.EventTypePermission && event.Decision == "" && event.RequestID != "" {
			if err := driver.ApprovePermission(testutil.Context(t), proc, acp.ApproveRequest{
				RequestID: event.RequestID,
				Decision:  "allow-always",
			}); err != nil {
				t.Fatalf("ApprovePermission() error = %v", err)
			}
		}
	})

	got := normalizeEvents(events)
	wantPath := filepath.Join("testdata", "tool_permission_golden.json")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", wantPath, err)
	}
	var want []map[string]string
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatalf("json.Unmarshal(golden) error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized events = %#v, want %#v", got, want)
	}
}

func TestDriverSupportsNetworkOriginEnvironmentExpectations(t *testing.T) {
	nodePath, err := ResolveNodePath()
	if err != nil {
		t.Skipf("ResolveNodePath() error = %v", err)
	}

	root := t.TempDir()
	fakeAGH := filepath.Join(root, "agh")
	if err := os.WriteFile(fakeAGH, []byte("#!/bin/sh\nprintf network-ok\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", fakeAGH, err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	driverPath, err := DefaultDriverPath()
	if err != nil {
		t.Fatalf("DefaultDriverPath() error = %v", err)
	}
	fixturePath, err := filepath.Abs(filepath.Join("testdata", "permission_env_fixture.json"))
	if err != nil {
		t.Fatalf("filepath.Abs(fixture) error = %v", err)
	}

	command := BuildCommand(
		nodePath,
		driverPath,
		fixturePath,
		"runner",
		filepath.Join(t.TempDir(), "runner-diagnostics.jsonl"),
	)

	driver := acp.New()
	proc, err := driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName:   "runner",
		Command:     command,
		Cwd:         root,
		Permissions: aghconfig.PermissionModeApproveAll,
	})
	if err != nil {
		t.Fatalf("driver.Start() error = %v", err)
	}
	proc.SetTurnSourceProvider(func() string { return "network" })
	defer stopDriverProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, acp.PromptRequest{
		TurnID:  "turn-network-environment",
		Message: "run environment",
	})
	if err != nil {
		t.Fatalf("driver.Prompt() error = %v", err)
	}

	events := collectPromptEvents(t, eventsCh, nil)
	if !containsNormalizedEvent(normalizeEvents(events), map[string]string{
		"type":         acp.EventTypeToolCall,
		"title":        "Run fake network status",
		"tool_call_id": "cmd-1",
	}) {
		t.Fatalf("events = %#v, want tool_call for network status", events)
	}
	if !containsNormalizedEvent(normalizeEvents(events), map[string]string{
		"type":         acp.EventTypeToolResult,
		"title":        "Run fake network status",
		"tool_call_id": "cmd-1",
	}) {
		t.Fatalf("events = %#v, want tool_result for network status", events)
	}
	if !containsNormalizedEvent(normalizeEvents(events), map[string]string{
		"type": acp.EventTypeAgentMessage,
		"text": "network-ok",
	}) {
		t.Fatalf("events = %#v, want network-ok assistant output", events)
	}
}

func TestDriverAdvertisesAndSupportsLoadSession(t *testing.T) {
	t.Parallel()

	nodePath, err := ResolveNodePath()
	if err != nil {
		t.Skipf("ResolveNodePath() error = %v", err)
	}

	driverPath, err := DefaultDriverPath()
	if err != nil {
		t.Fatalf("DefaultDriverPath() error = %v", err)
	}
	fixturePath, err := filepath.Abs(filepath.Join("testdata", "browser_session_lifecycle_fixture.json"))
	if err != nil {
		t.Fatalf("filepath.Abs(fixture) error = %v", err)
	}

	command := BuildCommand(
		nodePath,
		driverPath,
		fixturePath,
		"browser-lifecycle-agent",
		filepath.Join(t.TempDir(), "browser-lifecycle-diagnostics.jsonl"),
	)

	driver := acp.New()
	cwd := t.TempDir()

	proc, err := driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName:   "browser-lifecycle-agent",
		Command:     command,
		Cwd:         cwd,
		Permissions: aghconfig.PermissionModeDenyAll,
	})
	if err != nil {
		t.Fatalf("driver.Start() error = %v", err)
	}
	if !proc.Caps.SupportsLoadSession {
		t.Fatal("driver.Start() SupportsLoadSession = false, want true")
	}

	originalSessionID := proc.SessionID
	stopDriverProcess(t, driver, proc)

	resumed, err := driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName:       "browser-lifecycle-agent",
		Command:         command,
		Cwd:             cwd,
		Permissions:     aghconfig.PermissionModeDenyAll,
		ResumeSessionID: originalSessionID,
	})
	if err != nil {
		t.Fatalf("driver.Start(resume) error = %v", err)
	}
	defer stopDriverProcess(t, driver, resumed)

	if !resumed.Caps.SupportsLoadSession {
		t.Fatal("driver.Start(resume) SupportsLoadSession = false, want true")
	}
	if got := resumed.SessionID; got != originalSessionID {
		t.Fatalf("resumed SessionID = %q, want %q", got, originalSessionID)
	}
}

func collectPromptEvents(
	t testing.TB,
	eventsCh <-chan acp.AgentEvent,
	onEvent func(acp.AgentEvent),
) []acp.AgentEvent {
	t.Helper()

	events := make([]acp.AgentEvent, 0, 8)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				return events
			}
			events = append(events, event)
			if onEvent != nil {
				onEvent(event)
			}
		case <-timeout.C:
			t.Fatalf("timed out waiting for prompt events; collected %#v", events)
		}
	}
}

func containsNormalizedEvent(events []map[string]string, want map[string]string) bool {
	for _, event := range events {
		match := true
		for key, value := range want {
			if event[key] != value {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func stopDriverProcess(t testing.TB, driver *acp.Driver, proc *acp.AgentProcess) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.Stop(ctx, proc); err != nil {
		t.Fatalf("driver.Stop() error = %v", err)
	}
}
func normalizeEvents(events []acp.AgentEvent) []map[string]string {
	normalized := make([]map[string]string, 0, len(events))
	for _, event := range events {
		item := map[string]string{
			"type": event.Type,
		}
		if event.Text != "" {
			item["text"] = event.Text
		}
		if event.Title != "" {
			item["title"] = event.Title
		}
		if event.ToolCallID != "" {
			item["tool_call_id"] = event.ToolCallID
		}
		if event.Resource != "" {
			item["resource"] = event.Resource
		}
		if event.Decision != "" {
			item["decision"] = event.Decision
		}
		if event.StopReason != "" {
			item["stop_reason"] = event.StopReason
		}
		normalized = append(normalized, item)
	}
	return normalized
}
