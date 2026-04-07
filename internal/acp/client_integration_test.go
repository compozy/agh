//go:build integration

package acp

import (
	"github.com/pedronauck/agh/internal/testutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestACPIntegrationRoundTrip(t *testing.T) {
	driver := New()
	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
		TurnID:  "turn-integration-permission",
		Message: "request permission",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := make([]AgentEvent, 0, 8)
	var pendingRequestID string
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				goto done
			}
			events = append(events, event)
			if event.Type == EventTypePermission && event.Decision == "" && pendingRequestID == "" {
				pendingRequestID = event.RequestID
				if pendingRequestID == "" {
					t.Fatal("permission request_id = empty, want non-empty")
				}
				if err := driver.ApprovePermission(testutil.Context(t), proc, ApproveRequest{
					RequestID: pendingRequestID,
					Decision:  string(decisionAllowAlways),
				}); err != nil {
					t.Fatalf("ApprovePermission() error = %v", err)
				}
			}
		case <-timeout.C:
			t.Fatalf("timed out waiting for prompt events; collected %#v", events)
		}
	}
done:

	permissionEvents := make([]AgentEvent, 0, len(events))
	for _, event := range events {
		if event.Type == EventTypePermission {
			permissionEvents = append(permissionEvents, event)
		}
	}
	if len(permissionEvents) == 0 {
		t.Fatalf("Prompt() events = %#v, want permission event", events)
	}
	if permissionEvents[0].Decision != "" {
		t.Fatalf("initial permission event decision = %q, want empty", permissionEvents[0].Decision)
	}
	if permissionEvents[len(permissionEvents)-1].Decision != string(decisionAllowAlways) {
		t.Fatalf("final permission event decision = %q, want %q", permissionEvents[len(permissionEvents)-1].Decision, decisionAllowAlways)
	}
	if !containsEventText(events, "allow-always") {
		t.Fatalf("Prompt() events = %#v, want approved permission outcome", events)
	}
}

func TestACPIntegrationRequestPermissionTimeout(t *testing.T) {
	driver := New(WithPermissionTimeout(25 * time.Millisecond))

	root := t.TempDir()
	target := filepath.Join(root, "danger.txt")
	proc := startHelperProcess(t, driver, "permission", target, StartOpts{
		Cwd:         root,
		Permissions: aghconfig.PermissionModeDenyAll,
	})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
		TurnID:  "turn-integration-timeout",
		Message: "request permission",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if !containsEventText(events, "reject-once") {
		t.Fatalf("Prompt() events = %#v, want reject-once permission outcome", events)
	}

	permissionEvents := make([]AgentEvent, 0, len(events))
	for _, event := range events {
		if event.Type == EventTypePermission {
			permissionEvents = append(permissionEvents, event)
		}
	}
	if len(permissionEvents) < 2 {
		t.Fatalf("Prompt() permission events = %#v, want initial and final permission events", permissionEvents)
	}
	if permissionEvents[len(permissionEvents)-1].Decision != string(decisionRejectOnce) {
		t.Fatalf("final permission event decision = %q, want %q", permissionEvents[len(permissionEvents)-1].Decision, decisionRejectOnce)
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
