//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/udsapi"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store/globaldb"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestCLIRoundTripIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)

	startOut, _, err := executeRootCommand(t, h.deps, "daemon", "start", "-o", "json")
	if err != nil {
		t.Fatalf("daemon start error = %v", err)
	}
	var started DaemonStatus
	if err := json.Unmarshal([]byte(startOut), &started); err != nil {
		t.Fatalf("json.Unmarshal(start) error = %v", err)
	}
	if started.Status != "running" {
		t.Fatalf("start status = %q, want %q", started.Status, "running")
	}

	newOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(newOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created session id")
	}

	promptOut, _, err := executeRootCommand(t, h.deps, "session", "prompt", created.ID, "hello", "-o", "json")
	if err != nil {
		t.Fatalf("session prompt error = %v", err)
	}
	var promptEvents []AgentEventRecord
	if err := json.Unmarshal([]byte(promptOut), &promptEvents); err != nil {
		t.Fatalf("json.Unmarshal(prompt) error = %v", err)
	}
	if len(promptEvents) < 2 {
		t.Fatalf("prompt events = %d, want at least 2", len(promptEvents))
	}

	eventsOut, _, err := executeRootCommand(t, h.deps, "session", "events", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session events error = %v", err)
	}
	var events []SessionEventRecord
	if err := json.Unmarshal([]byte(eventsOut), &events); err != nil {
		t.Fatalf("json.Unmarshal(events) error = %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("session events = %d, want at least 2", len(events))
	}

	stopOut, _, err := executeRootCommand(t, h.deps, "session", "stop", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session stop error = %v", err)
	}
	var stopped SessionRecord
	if err := json.Unmarshal([]byte(stopOut), &stopped); err != nil {
		t.Fatalf("json.Unmarshal(stop) error = %v", err)
	}
	if stopped.State != session.StateStopped {
		t.Fatalf("stopped.State = %q, want %q", stopped.State, session.StateStopped)
	}

	daemonStopOut, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
	if err != nil {
		t.Fatalf("daemon stop error = %v", err)
	}
	var daemonStopped DaemonStatus
	if err := json.Unmarshal([]byte(daemonStopOut), &daemonStopped); err != nil {
		t.Fatalf("json.Unmarshal(daemon stop) error = %v", err)
	}
	if daemonStopped.Status != "stopped" {
		t.Fatalf("daemon stop status = %q, want %q", daemonStopped.Status, "stopped")
	}

	if err := h.runner.waitForExit(); err != nil {
		t.Fatalf("waitForExit() error = %v", err)
	}
}

func TestSessionListOutputFormatsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}

	humanOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "human")
	if err != nil {
		t.Fatalf("session list human error = %v", err)
	}
	if !strings.Contains(humanOut, "Sessions") || !strings.Contains(humanOut, created.ID) {
		t.Fatalf("human output = %q, want session table", humanOut)
	}

	jsonOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("session list json error = %v", err)
	}
	var listed []SessionRecord
	if err := json.Unmarshal([]byte(jsonOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed = %#v, want one created session", listed)
	}

	toonOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "toon")
	if err != nil {
		t.Fatalf("session list toon error = %v", err)
	}
	if !strings.Contains(toonOut, "sessions[1]{id,name,agent_name,state,workspace,space,updated_at}:") {
		t.Fatalf("toon output = %q, want TOON table", toonOut)
	}
}

func TestCLISessionSpaceRoundTripIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	newOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--space", "builders", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new --space error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(newOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new --space) error = %v", err)
	}
	if created.Space != "builders" {
		t.Fatalf("created.Space = %q, want %q", created.Space, "builders")
	}

	listOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("session list error = %v", err)
	}
	var listed []SessionRecord
	if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(listed) = %d, want %d", got, want)
	}
	if listed[0].Space != "builders" {
		t.Fatalf("listed[0].Space = %q, want %q", listed[0].Space, "builders")
	}

	stopOut, _, err := executeRootCommand(t, h.deps, "session", "stop", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session stop error = %v", err)
	}
	var stopped SessionRecord
	if err := json.Unmarshal([]byte(stopOut), &stopped); err != nil {
		t.Fatalf("json.Unmarshal(session stop) error = %v", err)
	}
	if stopped.Space != "builders" || stopped.State != session.StateStopped {
		t.Fatalf("stopped = %#v, want stopped builders session", stopped)
	}

	resumeOut, _, err := executeRootCommand(t, h.deps, "session", "resume", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session resume error = %v", err)
	}
	var resumed SessionRecord
	if err := json.Unmarshal([]byte(resumeOut), &resumed); err != nil {
		t.Fatalf("json.Unmarshal(session resume) error = %v", err)
	}
	if resumed.Space != "builders" || resumed.State != session.StateActive {
		t.Fatalf("resumed = %#v, want active builders session", resumed)
	}
}

func TestCLINetworkRoundTripIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	newOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "net-demo", "--space", "builders", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new --space error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(newOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new --space) error = %v", err)
	}

	statusOut, _, err := executeRootCommand(t, h.deps, "network", "status", "-o", "json")
	if err != nil {
		t.Fatalf("network status error = %v", err)
	}
	var status NetworkStatusRecord
	if err := json.Unmarshal([]byte(statusOut), &status); err != nil {
		t.Fatalf("json.Unmarshal(network status) error = %v", err)
	}
	if !status.Enabled || status.Status != "running" {
		t.Fatalf("network status = %#v, want enabled running", status)
	}

	peersOut, _, err := executeRootCommand(t, h.deps, "network", "peers", "builders", "-o", "json")
	if err != nil {
		t.Fatalf("network peers error = %v", err)
	}
	var peers []NetworkPeerRecord
	if err := json.Unmarshal([]byte(peersOut), &peers); err != nil {
		t.Fatalf("json.Unmarshal(network peers) error = %v", err)
	}
	if len(peers) != 1 || peers[0].SessionID == nil || *peers[0].SessionID != created.ID {
		t.Fatalf("network peers = %#v, want created session peer", peers)
	}

	spacesOut, _, err := executeRootCommand(t, h.deps, "network", "spaces", "-o", "json")
	if err != nil {
		t.Fatalf("network spaces error = %v", err)
	}
	var spaces []NetworkSpaceRecord
	if err := json.Unmarshal([]byte(spacesOut), &spaces); err != nil {
		t.Fatalf("json.Unmarshal(network spaces) error = %v", err)
	}
	if len(spaces) != 1 || spaces[0].Space != "builders" || spaces[0].PeerCount != 1 {
		t.Fatalf("network spaces = %#v, want builders peer_count=1", spaces)
	}

	events, err := h.runner.blockSession(created.ID)
	if err != nil {
		t.Fatalf("blockSession() error = %v", err)
	}
	if events == nil {
		t.Fatal("blockSession() events = nil, want event stream")
	}
	if !h.runner.waitForBlocked(created.ID, 2*time.Second) {
		t.Fatal("timed out waiting for blocked session prompt")
	}

	sendOut, _, err := executeRootCommand(t, h.deps,
		"network", "send",
		"--session", created.ID,
		"--space", "builders",
		"--kind", "say",
		"--body", `{"text":"queued hello"}`,
		"--ext", `{"agh.workflow_id":"wf-1","agh.handoff_version":3}`,
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("network send error = %v", err)
	}
	var sent NetworkSendRecord
	if err := json.Unmarshal([]byte(sendOut), &sent); err != nil {
		t.Fatalf("json.Unmarshal(network send) error = %v", err)
	}
	if sent.ID == "" || string(sent.Ext["agh.workflow_id"]) != `"wf-1"` {
		t.Fatalf("sent = %#v, want message id and ext metadata", sent)
	}

	var inbox []NetworkEnvelopeRecord
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		inboxOut, _, inboxErr := executeRootCommand(t, h.deps, "network", "inbox", "--session", created.ID, "-o", "json")
		if inboxErr != nil {
			t.Fatalf("network inbox error = %v", inboxErr)
		}
		if err := json.Unmarshal([]byte(inboxOut), &inbox); err != nil {
			t.Fatalf("json.Unmarshal(network inbox) error = %v", err)
		}
		if len(inbox) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(inbox) == 0 {
		t.Fatal("network inbox = empty, want queued message while prompt is blocked")
	}
	if string(inbox[0].Ext["agh.workflow_id"]) != `"wf-1"` || string(inbox[0].Ext["agh.handoff_version"]) != `3` {
		t.Fatalf("network inbox = %#v, want workflow metadata", inbox)
	}

	h.runner.releaseBlocked(created.ID)
}

func TestExtensionCommandRoundTripIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	dir := writeExtensionFixture(t, "integration-ext", extensionFixtureOptions{})

	installOut, _, err := executeRootCommand(t, h.deps, "extension", "install", dir, "-o", "json")
	if err != nil {
		t.Fatalf("extension install error = %v", err)
	}
	var installed ExtensionRecord
	if err := json.Unmarshal([]byte(installOut), &installed); err != nil {
		t.Fatalf("json.Unmarshal(extension install) error = %v", err)
	}
	if installed.Name != "integration-ext" || installed.State != "active" || !installed.DaemonRunning {
		t.Fatalf("installed extension = %#v, want active daemon-backed extension", installed)
	}

	listOut, _, err := executeRootCommand(t, h.deps, "extension", "list", "-o", "json")
	if err != nil {
		t.Fatalf("extension list error = %v", err)
	}
	var listed []ExtensionRecord
	if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(extension list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].Name != "integration-ext" || listed[0].State != "active" {
		t.Fatalf("listed extensions = %#v, want one active extension", listed)
	}

	statusOut, _, err := executeRootCommand(t, h.deps, "extension", "status", "integration-ext", "-o", "json")
	if err != nil {
		t.Fatalf("extension status error = %v", err)
	}
	var status ExtensionRecord
	if err := json.Unmarshal([]byte(statusOut), &status); err != nil {
		t.Fatalf("json.Unmarshal(extension status) error = %v", err)
	}
	if status.Name != "integration-ext" || status.State != "active" {
		t.Fatalf("extension status = %#v, want active extension", status)
	}

	if _, _, err := executeRootCommand(t, h.deps, "extension", "disable", "integration-ext", "-o", "json"); err != nil {
		t.Fatalf("extension disable error = %v", err)
	}

	listOut, _, err = executeRootCommand(t, h.deps, "extension", "list", "-o", "json")
	if err != nil {
		t.Fatalf("extension list after disable error = %v", err)
	}
	if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(extension list after disable) error = %v", err)
	}
	if len(listed) != 1 || listed[0].State != "disabled" || listed[0].Enabled {
		t.Fatalf("listed after disable = %#v, want one disabled extension", listed)
	}
}

func TestSessionEventsFollowIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}

	if _, _, err := executeRootCommand(t, h.deps, "session", "prompt", created.ID, "hello", "-o", "json"); err != nil {
		t.Fatalf("session prompt error = %v", err)
	}

	cmd := newRootCommand(h.deps)
	var stderr bytes.Buffer
	stdout := &lockedBuffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"session", "events", created.ID, "--follow", "-o", "json"})

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		done <- cmd.ExecuteContext(ctx)
	}()

	waitForCondition(t, 3*time.Second, func() bool {
		return strings.Contains(stdout.String(), `"type":"agent_message"`)
	})

	if _, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json"); err != nil {
		t.Fatalf("daemon stop error = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("follow command error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("follow output lines = %d, want at least 2", len(lines))
	}
	var sawAgentMessage bool
	for _, line := range lines {
		var event SessionEventRecord
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("json.Unmarshal(follow line) error = %v; line=%s", err, line)
		}
		if event.Type == "agent_message" {
			sawAgentMessage = true
		}
	}
	if !sawAgentMessage {
		t.Fatalf("follow output = %q, want streamed agent_message event", stdout.String())
	}
}

func TestWorkspaceCommandsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	addOut, _, err := executeRootCommand(t, h.deps, "workspace", "add", h.workspace, "--name", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("workspace add error = %v", err)
	}
	var registered WorkspaceRecord
	if err := json.Unmarshal([]byte(addOut), &registered); err != nil {
		t.Fatalf("json.Unmarshal(workspace add) error = %v", err)
	}
	if registered.ID == "" {
		t.Fatal("expected registered workspace id")
	}

	infoOut, _, err := executeRootCommand(t, h.deps, "workspace", "info", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("workspace info error = %v", err)
	}
	var detail WorkspaceDetailRecord
	if err := json.Unmarshal([]byte(infoOut), &detail); err != nil {
		t.Fatalf("json.Unmarshal(workspace info) error = %v", err)
	}
	if detail.Workspace.ID != registered.ID {
		t.Fatalf("workspace info id = %q, want %q", detail.Workspace.ID, registered.ID)
	}

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--workspace", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("session new with workspace error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if created.WorkspaceID != registered.ID {
		t.Fatalf("created.WorkspaceID = %q, want %q", created.WorkspaceID, registered.ID)
	}

	listOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--workspace", "alpha", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("session list --workspace error = %v", err)
	}
	var listed []SessionRecord
	if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed = %#v, want one workspace-filtered session", listed)
	}
}

func TestMemoryWriteListIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	if _, _, err := executeRootCommand(t, h.deps, "memory", "write", "prefs.md", "--type", "user", "--description", "cli memory", "--content", "remember this", "-o", "json"); err != nil {
		t.Fatalf("memory write error = %v", err)
	}

	listOut, _, err := executeRootCommand(t, h.deps, "memory", "list", "--scope", "global", "-o", "json")
	if err != nil {
		t.Fatalf("memory list error = %v", err)
	}

	var memories []memoryListItem
	if err := json.Unmarshal([]byte(listOut), &memories); err != nil {
		t.Fatalf("json.Unmarshal(memory list) error = %v; out=%s", err, listOut)
	}
	if len(memories) != 1 || memories[0].Filename != "prefs.md" {
		t.Fatalf("memories = %#v, want prefs.md", memories)
	}
}

func TestAutomationJobsCreateOutputFormatsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	humanOut, _, err := executeRootCommand(
		t,
		h.deps,
		"automation", "jobs", "create",
		"--name", "nightly-human",
		"--scope", "global",
		"--schedule", "every:30m",
		"--agent", "coder",
		"--prompt", "review repo",
		"-o", "human",
	)
	if err != nil {
		t.Fatalf("automation jobs create human error = %v", err)
	}
	if !strings.Contains(humanOut, "Automation Job") || !strings.Contains(humanOut, "nightly-human") {
		t.Fatalf("human output = %q, want created job detail", humanOut)
	}

	jsonOut, _, err := executeRootCommand(
		t,
		h.deps,
		"automation", "jobs", "create",
		"--name", "nightly-json",
		"--scope", "global",
		"--schedule", "every:45m",
		"--agent", "coder",
		"--prompt", "review repo later",
		"-o", "json",
	)
	if err != nil {
		t.Fatalf("automation jobs create json error = %v", err)
	}
	var created JobRecord
	if err := json.Unmarshal([]byte(jsonOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(automation jobs create) error = %v", err)
	}
	if created.ID == "" || created.Name != "nightly-json" || created.Scope != automationpkg.AutomationScopeGlobal {
		t.Fatalf("created job = %#v, want global created job", created)
	}
}

func TestAutomationTriggerHistoryAndRunsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	workspaceOut := mustExecuteRoot(t, h.deps, "workspace", "add", h.workspace, "--name", "alpha", "-o", "json")
	var workspace WorkspaceRecord
	if err := json.Unmarshal([]byte(workspaceOut), &workspace); err != nil {
		t.Fatalf("json.Unmarshal(workspace add) error = %v", err)
	}
	if workspace.ID == "" {
		t.Fatal("expected workspace id after registration")
	}

	triggerOut := mustExecuteRoot(
		t,
		h.deps,
		"automation", "triggers", "create",
		"--name", "stop-review",
		"--scope", "workspace",
		"--workspace", "alpha",
		"--event", "session.stopped",
		"--agent", "coder",
		"--prompt", `review {{ index .Data "session_id" }}`,
		"-o", "json",
	)
	var createdTrigger TriggerRecord
	if err := json.Unmarshal([]byte(triggerOut), &createdTrigger); err != nil {
		t.Fatalf("json.Unmarshal(trigger create) error = %v", err)
	}
	if createdTrigger.ID == "" || createdTrigger.WorkspaceID != workspace.ID {
		t.Fatalf("created trigger = %#v, want workspace-bound trigger", createdTrigger)
	}

	sessionOut := mustExecuteRoot(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--workspace", "alpha", "-o", "json")
	var createdSession SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &createdSession); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if createdSession.ID == "" {
		t.Fatal("expected session id for trigger test")
	}

	if _, _, err := executeRootCommand(t, h.deps, "session", "stop", createdSession.ID, "-o", "json"); err != nil {
		t.Fatalf("session stop error = %v", err)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		stdout, _, err := executeRootCommand(t, h.deps, "automation", "triggers", "history", createdTrigger.ID, "-o", "json")
		if err != nil {
			return false
		}
		var runs []RunRecord
		if err := json.Unmarshal([]byte(stdout), &runs); err != nil {
			return false
		}
		return len(runs) > 0
	})

	historyHuman, _, err := executeRootCommand(t, h.deps, "automation", "triggers", "history", createdTrigger.ID, "-o", "human")
	if err != nil {
		t.Fatalf("automation triggers history human error = %v", err)
	}
	if !strings.Contains(historyHuman, "Automation Runs") || !strings.Contains(historyHuman, "trigger:"+createdTrigger.ID) {
		t.Fatalf("history human output = %q, want trigger run table", historyHuman)
	}

	historyJSON, _, err := executeRootCommand(t, h.deps, "automation", "triggers", "history", createdTrigger.ID, "-o", "json")
	if err != nil {
		t.Fatalf("automation triggers history json error = %v", err)
	}
	var triggerRuns []RunRecord
	if err := json.Unmarshal([]byte(historyJSON), &triggerRuns); err != nil {
		t.Fatalf("json.Unmarshal(trigger history) error = %v", err)
	}
	if len(triggerRuns) == 0 || triggerRuns[0].TriggerID != createdTrigger.ID {
		t.Fatalf("trigger runs = %#v, want at least one run for trigger %q", triggerRuns, createdTrigger.ID)
	}

	runsHuman, _, err := executeRootCommand(t, h.deps, "automation", "runs", "-o", "human")
	if err != nil {
		t.Fatalf("automation runs human error = %v", err)
	}
	if !strings.Contains(runsHuman, "Automation Runs") || !strings.Contains(runsHuman, createdTrigger.ID) {
		t.Fatalf("runs human output = %q, want shared run table", runsHuman)
	}

	runsJSON, _, err := executeRootCommand(t, h.deps, "automation", "runs", "-o", "json")
	if err != nil {
		t.Fatalf("automation runs json error = %v", err)
	}
	var allRuns []RunRecord
	if err := json.Unmarshal([]byte(runsJSON), &allRuns); err != nil {
		t.Fatalf("json.Unmarshal(automation runs) error = %v", err)
	}
	if len(allRuns) == 0 {
		t.Fatal("expected at least one automation run in shared history")
	}
	found := false
	for _, run := range allRuns {
		if run.TriggerID == createdTrigger.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("allRuns = %#v, want one run for trigger %q", allRuns, createdTrigger.ID)
	}
}

func TestChannelCreateAndGetIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	createOut := mustExecuteRoot(
		t,
		h.deps,
		"channel", "create",
		"--scope", "global",
		"--platform", "telegram",
		"--extension", "ext-telegram",
		"--display-name", "Support",
		"--include-peer",
		"--status", "ready",
		"-o", "json",
	)

	var created ChannelRecord
	if err := json.Unmarshal([]byte(createOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(channel create) error = %v", err)
	}
	if created.ID == "" || created.Platform != "telegram" || created.Status != channelspkg.ChannelStatusReady {
		t.Fatalf("created channel = %#v", created)
	}

	getOut := mustExecuteRoot(t, h.deps, "channel", "get", created.ID, "-o", "json")

	var fetched ChannelRecord
	if err := json.Unmarshal([]byte(getOut), &fetched); err != nil {
		t.Fatalf("json.Unmarshal(channel get) error = %v", err)
	}
	if fetched.ID != created.ID || fetched.DisplayName != "Support" || fetched.ExtensionName != "ext-telegram" {
		t.Fatalf("fetched channel = %#v, want created record", fetched)
	}
}

func TestChannelLifecycleCommandsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	createOut := mustExecuteRoot(
		t,
		h.deps,
		"channel", "create",
		"--scope", "global",
		"--platform", "telegram",
		"--extension", "ext-telegram",
		"--display-name", "Ops",
		"--enabled=false",
		"--include-peer",
		"-o", "json",
	)

	var created ChannelRecord
	if err := json.Unmarshal([]byte(createOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(channel create) error = %v", err)
	}
	if created.Status != channelspkg.ChannelStatusDisabled || created.Enabled {
		t.Fatalf("created lifecycle = %#v, want disabled false", created)
	}

	enableOut := mustExecuteRoot(t, h.deps, "channel", "enable", created.ID, "-o", "json")
	var enabled ChannelRecord
	if err := json.Unmarshal([]byte(enableOut), &enabled); err != nil {
		t.Fatalf("json.Unmarshal(channel enable) error = %v", err)
	}
	if enabled.Status != channelspkg.ChannelStatusStarting || !enabled.Enabled {
		t.Fatalf("enabled channel = %#v, want starting true", enabled)
	}

	disableOut := mustExecuteRoot(t, h.deps, "channel", "disable", created.ID, "-o", "json")
	var disabled ChannelRecord
	if err := json.Unmarshal([]byte(disableOut), &disabled); err != nil {
		t.Fatalf("json.Unmarshal(channel disable) error = %v", err)
	}
	if disabled.Status != channelspkg.ChannelStatusDisabled || disabled.Enabled {
		t.Fatalf("disabled channel = %#v, want disabled false", disabled)
	}

	restartOut := mustExecuteRoot(t, h.deps, "channel", "restart", created.ID, "-o", "json")
	var restarted ChannelRecord
	if err := json.Unmarshal([]byte(restartOut), &restarted); err != nil {
		t.Fatalf("json.Unmarshal(channel restart) error = %v", err)
	}
	if restarted.Status != channelspkg.ChannelStatusStarting || !restarted.Enabled {
		t.Fatalf("restarted channel = %#v, want starting true", restarted)
	}
}

func TestChannelRoutesIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	createOut := mustExecuteRoot(
		t,
		h.deps,
		"channel", "create",
		"--scope", "global",
		"--platform", "telegram",
		"--extension", "ext-telegram",
		"--display-name", "Support",
		"--include-peer",
		"--include-thread",
		"--status", "ready",
		"-o", "json",
	)

	var created ChannelRecord
	if err := json.Unmarshal([]byte(createOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(channel create) error = %v", err)
	}

	channels := h.runner.channelService()
	if channels == nil {
		t.Fatal("channel service = nil, want running integration channel service")
	}
	if _, err := channels.UpsertRoute(context.Background(), channelspkg.ChannelRoute{
		ChannelInstanceID: created.ID,
		Scope:             created.Scope,
		WorkspaceID:       created.WorkspaceID,
		PeerID:            "peer-1",
		ThreadID:          "thread-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
		LastActivityAt:    fixedTestNow,
	}); err != nil {
		t.Fatalf("UpsertRoute() error = %v", err)
	}

	routesOut := mustExecuteRoot(t, h.deps, "channel", "routes", created.ID, "-o", "json")

	var routes []ChannelRouteRecord
	if err := json.Unmarshal([]byte(routesOut), &routes); err != nil {
		t.Fatalf("json.Unmarshal(channel routes) error = %v", err)
	}
	if len(routes) != 1 || routes[0].PeerID != "peer-1" || routes[0].ThreadID != "thread-1" {
		t.Fatalf("routes = %#v, want one inserted route", routes)
	}

	_, _, err := executeRootCommand(t, h.deps, "channel", "routes", "missing-channel", "-o", "json")
	if err == nil || !strings.Contains(err.Error(), "channel instance not found") {
		t.Fatalf("channel routes missing error = %v, want channel instance not found", err)
	}
}

type integrationHarness struct {
	deps      commandDeps
	homePaths aghconfig.HomePaths
	workspace string
	runner    *integrationDaemon
}

type integrationDreamTrigger struct {
	enabled   bool
	triggered bool
	reason    string
	last      time.Time
}

func (t *integrationDreamTrigger) Trigger(context.Context, string) (bool, string, error) {
	return t.triggered, t.reason, nil
}

func (t *integrationDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return t.last, nil
}

func (t *integrationDreamTrigger) Enabled() bool {
	return t.enabled
}

type integrationDaemon struct {
	t         *testing.T
	homePaths aghconfig.HomePaths
	cfg       aghconfig.Config
	pid       int
	startedAt time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan error

	channels *integrationChannelService
	driver   *integrationDriver
	manager  *session.Manager
}

type integrationDaemonProcess struct {
	pid  int
	done <-chan error
}

type integrationExtensionService struct {
	registry *extensionpkg.Registry
	manager  *extensionpkg.Manager
}

type integrationChannelService struct {
	*channelspkg.Service
}

type integrationNotifierFanout struct {
	notifiers []session.Notifier
}

type integrationDriver struct {
	mu       sync.Mutex
	nextPID  int
	nextSess int
	states   map[*session.AgentProcess]chan struct{}
	blocked  map[string]chan struct{}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func newIntegrationChannelService(store channelspkg.RegistryStore) *integrationChannelService {
	return &integrationChannelService{Service: channelspkg.NewRegistry(store)}
}

func (s *integrationChannelService) StartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusStarting,
	})
}

func (s *integrationChannelService) StopInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: false,
		Status:  channelspkg.ChannelStatusDisabled,
	})
}

func (s *integrationChannelService) RestartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	return s.UpdateInstanceState(ctx, channelspkg.UpdateInstanceStateRequest{
		ID:      id,
		Enabled: true,
		Status:  channelspkg.ChannelStatusStarting,
	})
}

func (s *integrationExtensionService) List(ctx context.Context) ([]contract.ExtensionPayload, error) {
	infos, err := s.registry.List()
	if err != nil {
		return nil, err
	}

	items := make([]contract.ExtensionPayload, 0, len(infos))
	for _, info := range infos {
		item, err := s.Status(ctx, info.Name)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *integrationExtensionService) Install(ctx context.Context, req contract.InstallExtensionRequest) (contract.ExtensionPayload, error) {
	manifest, err := extensionpkg.LoadManifest(req.Path)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.registry.Install(manifest, req.Path, req.Checksum); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.manager.Reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.Status(ctx, manifest.Name)
}

func (s *integrationExtensionService) Enable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.registry.Enable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.manager.Reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.Status(ctx, name)
}

func (s *integrationExtensionService) Disable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if err := s.registry.Disable(name); err != nil {
		return contract.ExtensionPayload{}, err
	}
	if err := s.manager.Reload(ctx); err != nil {
		return contract.ExtensionPayload{}, err
	}
	return s.Status(ctx, name)
}

func (s *integrationExtensionService) Status(_ context.Context, name string) (contract.ExtensionPayload, error) {
	ext, err := s.manager.Get(name)
	if err != nil {
		return contract.ExtensionPayload{}, err
	}
	if ext.Manifest == nil && strings.TrimSpace(ext.Info.ManifestPath) != "" {
		manifest, loadErr := extensionpkg.LoadManifest(filepath.Dir(ext.Info.ManifestPath))
		if loadErr == nil {
			ext.Manifest = manifest
		}
	}
	return extensionpkg.DescribeExtension(ext, true, time.Now().UTC()), nil
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func newIntegrationHarness(t *testing.T) integrationHarness {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	socketPath := shortSocketPath(t)
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	writeAgentDef(t, homePaths, "coder")

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath
	cfg.Network.Enabled = true
	cfg.Network.Port = -1
	cfg.Network.GreetInterval = 1
	cfg.Providers = map[string]aghconfig.ProviderConfig{
		"fake": {Command: "fake-agent"},
	}

	runner := &integrationDaemon{
		t:         t,
		homePaths: homePaths,
		cfg:       cfg,
		pid:       4242,
		startedAt: time.Now().UTC(),
	}

	deps := commandDeps{
		loadConfig: func() (aghconfig.Config, error) {
			return cfg, nil
		},
		resolveHome: func() (aghconfig.HomePaths, error) {
			return homePaths, nil
		},
		ensureHome: aghconfig.EnsureHomeLayout,
		newClient:  NewClient,
		newDaemon: func() (daemonRunner, error) {
			return runner, nil
		},
		readDaemonInfo: aghdaemon.ReadInfo,
		signalProcess:  runner.signalProcess,
		processAlive:   runner.processAlive,
		getwd: func() (string, error) {
			return t.TempDir(), nil
		},
		getenv: func(string) string { return "" },
		now: func() time.Time {
			return time.Now().UTC()
		},
		pollInterval: 10 * time.Millisecond,
		startTimeout: 5 * time.Second,
		stopTimeout:  5 * time.Second,
		spawnDetached: func(aghconfig.HomePaths) (daemonProcess, error) {
			return runner.spawnDetached()
		},
	}

	return integrationHarness{
		deps:      deps,
		homePaths: homePaths,
		workspace: t.TempDir(),
		runner:    runner,
	}
}

func (p *integrationDaemonProcess) PID() int {
	return p.pid
}

func (p *integrationDaemonProcess) Wait() error {
	return <-p.done
}

func (d *integrationDaemon) spawnDetached() (daemonProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		return nil, errors.New("integration daemon already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	d.running = true
	d.cancel = cancel
	d.done = done

	go func() {
		err := d.Run(ctx)
		done <- err
		close(done)
		d.mu.Lock()
		d.running = false
		d.cancel = nil
		d.done = nil
		d.mu.Unlock()
	}()

	return &integrationDaemonProcess{pid: d.pid, done: done}, nil
}

func (d *integrationDaemon) Run(ctx context.Context) error {
	registry, err := globaldb.OpenGlobalDB(context.Background(), d.homePaths.DatabaseFile)
	if err != nil {
		return fmt.Errorf("open global db: %w", err)
	}
	defer func() {
		_ = registry.Close(context.Background())
	}()

	fanout := &integrationNotifierFanout{}
	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(d.homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) { return d.cfg, nil }),
	)
	if err != nil {
		return fmt.Errorf("new workspace resolver: %w", err)
	}
	manager, err := session.NewManager(
		session.WithHomePaths(d.homePaths),
		session.WithWorkspaceResolver(resolver),
		session.WithLogger(discardLogger()),
		session.WithDriver(func() *integrationDriver {
			driver := newIntegrationDriver()
			d.mu.Lock()
			d.driver = driver
			d.mu.Unlock()
			return driver
		}()),
		session.WithNotifier(fanout),
	)
	if err != nil {
		return fmt.Errorf("new session manager: %w", err)
	}
	d.mu.Lock()
	d.manager = manager
	d.mu.Unlock()

	observer, err := observe.New(
		context.Background(),
		observe.WithHomePaths(d.homePaths),
		observe.WithRegistry(registry),
		observe.WithSessionSource(manager),
		observe.WithLogger(discardLogger()),
		observe.WithStartTime(d.startedAt),
	)
	if err != nil {
		return fmt.Errorf("new observer: %w", err)
	}
	defer func() {
		_ = observer.Close(context.Background())
	}()
	fanout.notifiers = append(fanout.notifiers, observer)

	memoryStore := memory.NewStore(d.homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure memory dirs: %w", err)
	}
	channelService := newIntegrationChannelService(registry)
	dreamTrigger := &integrationDreamTrigger{
		enabled:   true,
		triggered: true,
		last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}
	extRegistry := extensionpkg.NewRegistry(registry.DB())
	extManager := extensionpkg.NewManager(
		extRegistry,
		extensionpkg.WithLogger(discardLogger()),
	)
	if err := extManager.Start(context.Background()); err != nil {
		return fmt.Errorf("start extension manager: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = extManager.Stop(shutdownCtx)
	}()
	extService := &integrationExtensionService{
		registry: extRegistry,
		manager:  extManager,
	}

	automationManager, err := automationpkg.New(
		automationpkg.WithStore(registry),
		automationpkg.WithSessions(manager),
		automationpkg.WithWorkspaceResolver(resolver),
		automationpkg.WithConfig(d.cfg.Automation),
		automationpkg.WithLogger(discardLogger()),
		automationpkg.WithGlobalWorkspacePath(d.homePaths.HomeDir),
	)
	if err != nil {
		return fmt.Errorf("new automation manager: %w", err)
	}
	if err := automationManager.Start(ctx); err != nil {
		return fmt.Errorf("start automation manager: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = automationManager.Shutdown(shutdownCtx)
	}()
	fanout.notifiers = append(fanout.notifiers, automationManager.SessionObserver())
	networkManager, err := network.NewManager(
		ctx,
		d.cfg.Network,
		manager,
		d.homePaths.NetworkAuditFile,
		registry,
		network.WithManagerLogger(discardLogger()),
	)
	if err != nil {
		return fmt.Errorf("new network manager: %w", err)
	}
	manager.SetNetworkPeerLifecycle(networkManager)
	manager.SetTurnEndNotifier(networkManager.OnTurnEnd)
	server, err := udsapi.New(
		udsapi.WithHomePaths(d.homePaths),
		udsapi.WithConfig(d.cfg),
		udsapi.WithSocketPath(d.cfg.Daemon.Socket),
		udsapi.WithLogger(discardLogger()),
		udsapi.WithStartedAt(d.startedAt),
		udsapi.WithPollInterval(10*time.Millisecond),
		udsapi.WithSessionManager(manager),
		udsapi.WithNetworkService(networkManager),
		udsapi.WithObserver(observer),
		udsapi.WithAutomation(automationManager),
		udsapi.WithChannelService(channelService),
		udsapi.WithWorkspaceResolver(resolver),
		udsapi.WithMemoryStore(memoryStore),
		udsapi.WithDreamTrigger(dreamTrigger),
		udsapi.WithExtensionService(extService),
	)
	if err != nil {
		return fmt.Errorf("new uds server: %w", err)
	}

	if err := server.Start(context.Background()); err != nil {
		return fmt.Errorf("start uds server: %w", err)
	}
	d.mu.Lock()
	d.channels = channelService
	d.mu.Unlock()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, info := range manager.List() {
			if info == nil || info.State == session.StateStopped {
				continue
			}
			_ = manager.Stop(shutdownCtx, info.ID)
		}
		_ = networkManager.Shutdown(shutdownCtx)
		_ = server.Shutdown(shutdownCtx)
		_ = aghdaemon.RemoveInfo(d.homePaths.DaemonInfo)
		d.mu.Lock()
		d.channels = nil
		d.manager = nil
		d.driver = nil
		d.mu.Unlock()
	}()

	if err := aghdaemon.WriteInfo(d.homePaths.DaemonInfo, aghdaemon.Info{
		PID:       d.pid,
		Port:      d.cfg.HTTP.Port,
		StartedAt: d.startedAt,
	}); err != nil {
		return fmt.Errorf("write daemon info: %w", err)
	}

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

func (d *integrationDaemon) signalProcess(pid int, sig syscall.Signal) error {
	d.mu.Lock()
	cancel := d.cancel
	running := d.running
	d.mu.Unlock()

	if !running || pid != d.pid {
		return fmt.Errorf("integration daemon pid %d is not running", pid)
	}
	if sig != syscall.SIGTERM {
		return fmt.Errorf("unsupported signal %v", sig)
	}
	cancel()
	return nil
}

func (d *integrationDaemon) processAlive(pid int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running && pid == d.pid
}

func (d *integrationDaemon) waitForExit() error {
	d.mu.Lock()
	done := d.done
	d.mu.Unlock()
	if done == nil {
		return nil
	}
	return <-done
}

func (d *integrationDaemon) channelService() *integrationChannelService {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.channels
}

func (f *integrationNotifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	for _, notifier := range f.notifiers {
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

func newIntegrationDriver() *integrationDriver {
	return &integrationDriver{
		nextPID:  2000,
		nextSess: 1,
		states:   make(map[*session.AgentProcess]chan struct{}),
		blocked:  make(map[string]chan struct{}),
	}
}

func (d *integrationDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextPID++
	d.nextSess++
	done := make(chan struct{})
	sessionID := strings.TrimSpace(opts.ResumeSessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("acp-session-%d", d.nextSess)
	}

	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       d.nextPID,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: sessionID,
		Caps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModels:     []string{"fake-model"},
		},
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	ch := make(chan acp.AgentEvent, 2)
	ch <- acp.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
	}
	if strings.Contains(req.Message, "__block__") {
		release := make(chan struct{})
		d.mu.Lock()
		d.blocked[proc.SessionID] = release
		d.mu.Unlock()

		go func() {
			<-release
			ch <- acp.AgentEvent{
				Type:       "done",
				SessionID:  proc.SessionID,
				TurnID:     req.TurnID,
				Timestamp:  time.Now().UTC(),
				StopReason: "end_turn",
			}
			close(ch)
			d.mu.Lock()
			delete(d.blocked, proc.SessionID)
			d.mu.Unlock()
		}()
		return ch, nil
	}
	ch <- acp.AgentEvent{
		Type:       "done",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		StopReason: "end_turn",
	}
	close(ch)
	return ch, nil
}

func (d *integrationDriver) releaseBlocked(sessionID string) {
	d.mu.Lock()
	release := d.blocked[sessionID]
	d.mu.Unlock()
	if release == nil {
		return
	}
	select {
	case <-release:
	default:
		close(release)
	}
}

func (d *integrationDriver) waitForBlocked(sessionID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		d.mu.Lock()
		_, ok := d.blocked[sessionID]
		d.mu.Unlock()
		if ok {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func (d *integrationDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *integrationDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	done, ok := d.states[proc]
	if !ok {
		return nil
	}
	select {
	case <-done:
	default:
		close(done)
	}
	delete(d.states, proc)
	if release, ok := d.blocked[proc.SessionID]; ok {
		select {
		case <-release:
		default:
			close(release)
		}
		delete(d.blocked, proc.SessionID)
	}
	return nil
}

func (d *integrationDaemon) releaseBlocked(sessionID string) {
	d.mu.Lock()
	driver := d.driver
	manager := d.manager
	d.mu.Unlock()
	if driver == nil {
		return
	}
	target := sessionID
	if manager != nil {
		if info, err := manager.Status(context.Background(), sessionID); err == nil && strings.TrimSpace(info.ACPSessionID) != "" {
			target = info.ACPSessionID
		}
	}
	driver.releaseBlocked(target)
}

func (d *integrationDaemon) waitForBlocked(sessionID string, timeout time.Duration) bool {
	d.mu.Lock()
	driver := d.driver
	manager := d.manager
	d.mu.Unlock()
	if driver == nil {
		return false
	}
	target := sessionID
	if manager != nil {
		if info, err := manager.Status(context.Background(), sessionID); err == nil && strings.TrimSpace(info.ACPSessionID) != "" {
			target = info.ACPSessionID
		}
	}
	return driver.waitForBlocked(target, timeout)
}

func (d *integrationDaemon) blockSession(sessionID string) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	manager := d.manager
	d.mu.Unlock()
	if manager == nil {
		return nil, errors.New("integration daemon session manager is not ready")
	}
	return manager.Prompt(context.Background(), sessionID, "__block__")
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	root, err := os.MkdirTemp(os.TempDir(), "aghc-")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	return filepath.Join(root, "daemon.sock")
}

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()

	agentDir := filepath.Join(homePaths.AgentsDir, name)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", agentDir, err)
	}
	content := strings.Join([]string{
		"---",
		"name: " + name,
		"provider: fake",
		"model: fake-model",
		"---",
		"",
		"You are the integration test agent.",
	}, "\n")
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(AGENT.md) error = %v", err)
	}
}

func mustExecuteRoot(t *testing.T, deps commandDeps, args ...string) string {
	t.Helper()

	stdout, stderr, err := executeRootCommand(t, deps, args...)
	if err != nil {
		t.Fatalf("executeRootCommand(%v) error = %v; stderr=%s", args, err, stderr)
	}
	return stdout
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}
