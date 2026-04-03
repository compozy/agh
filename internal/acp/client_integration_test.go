//go:build integration

package acp

import (
	"os"
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestACPIntegrationRoundTrip(t *testing.T) {
	driver := New()
	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-integration-roundtrip",
		Message: "run roundtrip",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if len(events) == 0 {
		t.Fatal("Prompt() returned no events")
	}
	if events[len(events)-1].Type != EventTypeDone {
		t.Fatalf("Prompt() last event = %#v, want done", events[len(events)-1])
	}
}

func TestACPIntegrationReadTextFileRequest(t *testing.T) {
	driver := New()

	root := t.TempDir()
	target := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(target, []byte("from-disk"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	proc := startHelperProcess(t, driver, "fs_read", target, StartOpts{
		Cwd:         root,
		Permissions: aghconfig.PermissionModeApproveReads,
	})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-integration-fs",
		Message: "read file",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if !containsEventText(events, "from-disk") {
		t.Fatalf("Prompt() events = %#v, want file content", events)
	}
}

func TestACPIntegrationRequestPermissionPolicy(t *testing.T) {
	driver := New()

	root := t.TempDir()
	target := filepath.Join(root, "danger.txt")
	proc := startHelperProcess(t, driver, "permission", target, StartOpts{
		Cwd:         root,
		Permissions: aghconfig.PermissionModeDenyAll,
	})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-integration-permission",
		Message: "request permission",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if !containsEventText(events, "reject") {
		t.Fatalf("Prompt() events = %#v, want rejected permission outcome", events)
	}

	permissionEvents := make([]AgentEvent, 0, len(events))
	for _, event := range events {
		if event.Type == EventTypePermission {
			permissionEvents = append(permissionEvents, event)
		}
	}
	if len(permissionEvents) == 0 {
		t.Fatalf("Prompt() events = %#v, want permission event", events)
	}
	if permissionEvents[0].Decision != "deny" {
		t.Fatalf("permission event decision = %q, want deny", permissionEvents[0].Decision)
	}
}

func containsEventText(events []AgentEvent, want string) bool {
	for _, event := range events {
		if event.Text == want {
			return true
		}
	}
	return false
}
