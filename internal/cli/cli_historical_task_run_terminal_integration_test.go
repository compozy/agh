//go:build integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/compozy/agh/internal/session"
	taskpkg "github.com/compozy/agh/internal/task"
)

func TestCLIHistoricalChannelTaskRunTerminalAfterDaemonRestartIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		channel        string
		title          string
		terminalArgs   []string
		wantRunStatus  taskpkg.RunStatus
		wantTaskStatus taskpkg.Status
		wantEventType  string
	}{
		{
			name:    "Should fail a historical task run after daemon restart",
			channel: "history-run-fail",
			title:   "CLI historical run fail restart",
			terminalArgs: []string{
				"task",
				"run",
				"fail",
				"--error",
				"operator-detected failure",
				"--metadata",
				`{"source":"integration","mode":"historical-restart"}`,
			},
			wantRunStatus:  taskpkg.TaskRunStatusFailed,
			wantTaskStatus: taskpkg.TaskStatusReady,
			wantEventType:  "task.run_failed",
		},
		{
			name:    "Should cancel a historical task run after daemon restart",
			channel: "history-run-cancel",
			title:   "CLI historical run cancel restart",
			terminalArgs: []string{
				"task",
				"run",
				"cancel",
				"--reason",
				"operator-request",
				"--metadata",
				`{"source":"integration","mode":"historical-restart"}`,
			},
			wantRunStatus:  taskpkg.TaskRunStatusCanceled,
			wantTaskStatus: taskpkg.TaskStatusCanceled,
			wantEventType:  "task.run_canceled",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newIntegrationHarness(t)
			mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
			defer func() {
				if _, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json"); err != nil {
					t.Fatalf("daemon stop during cleanup error = %v", err)
				}
				if err := h.runner.waitForExit(); err != nil {
					t.Fatalf("waitForExit() during cleanup error = %v", err)
				}
			}()

			if _, _, err := executeRootCommand(
				t,
				h.deps,
				"workspace",
				"add",
				h.workspace,
				"--name",
				"alpha",
				"-o",
				"json",
			); err != nil {
				t.Fatalf("workspace add error = %v", err)
			}

			seedOut := mustExecuteRoot(
				t,
				h.deps,
				"session",
				"new",
				"--agent",
				"coder",
				"--name",
				tt.channel+"-seed",
				"--workspace",
				"alpha",
				"--network",
				"live",
				"--network-channel-strategy",
				"named",
				"--network-channel",
				tt.channel,
				"-o",
				"json",
			)
			var seed SessionRecord
			if err := json.Unmarshal([]byte(seedOut), &seed); err != nil {
				t.Fatalf("json.Unmarshal(session new) error = %v", err)
			}
			if seed.ID == "" || seed.State != session.StateActive ||
				resolvedParticipationChannelID(seed.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("seed = %#v, want active seed session on %q", seed, tt.channel)
			}

			stopSeedOut := mustExecuteRoot(t, h.deps, "session", "stop", seed.ID, "-o", "json")
			var stoppedSeed SessionRecord
			if err := json.Unmarshal([]byte(stopSeedOut), &stoppedSeed); err != nil {
				t.Fatalf("json.Unmarshal(session stop) error = %v", err)
			}
			if stoppedSeed.State != session.StateStopped ||
				resolvedParticipationChannelID(stoppedSeed.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("stoppedSeed = %#v, want stopped seed session on %q", stoppedSeed, tt.channel)
			}

			channelBeforeRestart := readCLIHistoricalChannel(t, h.deps, tt.channel)
			if got, want := channelBeforeRestart.PeerCount, 0; got != want {
				t.Fatalf("channelBeforeRestart.PeerCount = %d, want %d", got, want)
			}
			if channelBeforeRestart.PresenceCount < 1 {
				t.Fatalf("channelBeforeRestart.PresenceCount = %d, want at least 1", channelBeforeRestart.PresenceCount)
			}
			if channelBeforeRestart.HistoricalParticipantCount < 1 {
				t.Fatalf(
					"channelBeforeRestart.HistoricalParticipantCount = %d, want at least 1",
					channelBeforeRestart.HistoricalParticipantCount,
				)
			}

			createOut := mustExecuteRoot(
				t,
				h.deps,
				"task",
				"create",
				"--scope",
				"workspace",
				"--workspace",
				"alpha",
				"--network",
				"live",
				"--network-channel-strategy",
				"named",
				"--network-channel",
				tt.channel,
				"--title",
				tt.title,
				"-o",
				"json",
			)
			var created TaskRecord
			if err := json.Unmarshal([]byte(createOut), &created); err != nil {
				t.Fatalf("json.Unmarshal(task create) error = %v", err)
			}
			if created.ID == "" ||
				resolvedParticipationChannelID(created.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("created = %#v, want historical channel task", created)
			}

			enqueueOut := mustExecuteRoot(
				t,
				h.deps,
				"task",
				"run",
				"enqueue",
				created.ID,
				"--idempotency-key",
				"idem-"+tt.channel,
				"--network",
				"live",
				"--network-channel-strategy",
				"named",
				"--network-channel",
				tt.channel,
				"-o",
				"json",
			)
			var enqueued TaskRunRecord
			if err := json.Unmarshal([]byte(enqueueOut), &enqueued); err != nil {
				t.Fatalf("json.Unmarshal(task run enqueue) error = %v", err)
			}
			if enqueued.ID == "" || enqueued.Status != taskpkg.TaskRunStatusQueued {
				t.Fatalf("enqueued = %#v, want queued run", enqueued)
			}
			if resolvedParticipationChannelID(enqueued.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("enqueued = %#v, want preserved historical channel", enqueued)
			}

			mustExecuteRoot(t, h.deps, "daemon", "stop", "-o", "json")
			if err := h.runner.waitForExit(); err != nil {
				t.Fatalf("waitForExit() after stop error = %v", err)
			}
			mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")

			channelAfterRestart := readCLIHistoricalChannel(t, h.deps, tt.channel)
			if got, want := channelAfterRestart.PeerCount, 0; got != want {
				t.Fatalf("channelAfterRestart.PeerCount = %d, want %d", got, want)
			}
			if channelAfterRestart.PresenceCount < 1 {
				t.Fatalf("channelAfterRestart.PresenceCount = %d, want at least 1", channelAfterRestart.PresenceCount)
			}
			if channelAfterRestart.HistoricalParticipantCount < 1 {
				t.Fatalf(
					"channelAfterRestart.HistoricalParticipantCount = %d, want at least 1",
					channelAfterRestart.HistoricalParticipantCount,
				)
			}

			agentDeps, _ := newIntegrationAgentCommandDeps(
				t,
				h,
				tt.channel+"-worker",
				"alpha",
				tt.channel,
			)
			claimOut := mustExecuteRoot(t, agentDeps, "task", "next", "--run-id", enqueued.ID, "-o", "json")
			var next AgentTaskNextRecord
			if err := json.Unmarshal([]byte(claimOut), &next); err != nil {
				t.Fatalf("json.Unmarshal(task next exact claim) error = %v", err)
			}
			if !next.Claimed || next.Claim == nil || next.Claim.Run.ID != enqueued.ID ||
				next.Claim.Run.Status != taskpkg.TaskRunStatusClaimed {
				t.Fatalf("task next = %#v, want exact claimed run %q", next, enqueued.ID)
			}
			if resolvedParticipationChannelID(next.Claim.Run.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("task next = %#v, want preserved historical channel", next)
			}

			startOut := mustExecuteRoot(t, h.deps, "task", "run", "start", enqueued.ID, "-o", "json")
			var started TaskRunRecord
			if err := json.Unmarshal([]byte(startOut), &started); err != nil {
				t.Fatalf("json.Unmarshal(task run start) error = %v", err)
			}
			if started.Status != taskpkg.TaskRunStatusRunning || started.SessionID == "" {
				t.Fatalf("started = %#v, want running run with session", started)
			}
			if resolvedParticipationChannelID(started.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("started = %#v, want preserved historical channel", started)
			}

			args := append(append([]string{}, tt.terminalArgs...), enqueued.ID, "-o", "json")
			terminalOut := mustExecuteRoot(t, h.deps, args...)
			var terminalRun TaskRunRecord
			if err := json.Unmarshal([]byte(terminalOut), &terminalRun); err != nil {
				t.Fatalf("json.Unmarshal(terminal run) error = %v", err)
			}
			if terminalRun.Status != tt.wantRunStatus {
				t.Fatalf("terminalRun = %#v, want status %q", terminalRun, tt.wantRunStatus)
			}
			if resolvedParticipationChannelID(terminalRun.ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("terminalRun = %#v, want preserved historical channel", terminalRun)
			}

			getOut := mustExecuteRoot(t, h.deps, "task", "get", created.ID, "-o", "json")
			var detail TaskDetailRecord
			if err := json.Unmarshal([]byte(getOut), &detail); err != nil {
				t.Fatalf("json.Unmarshal(task get) error = %v", err)
			}
			if detail.Task.Status != tt.wantTaskStatus || detail.Summary.Status != tt.wantTaskStatus {
				t.Fatalf("detail = %#v, want task status %q", detail, tt.wantTaskStatus)
			}
			if len(detail.Runs) != 1 {
				t.Fatalf("detail.Runs = %#v, want exactly one run", detail.Runs)
			}
			if detail.Runs[0].Status != tt.wantRunStatus || detail.Runs[0].SessionID == "" {
				t.Fatalf("detail.Runs[0] = %#v, want terminal run with session", detail.Runs[0])
			}
			if resolvedParticipationChannelID(detail.Runs[0].ResolvedNetworkParticipation) != tt.channel {
				t.Fatalf("detail.Runs[0] = %#v, want preserved historical channel", detail.Runs[0])
			}
			if !containsCLITaskEventType(detail.Events, tt.wantEventType) {
				t.Fatalf("detail.Events = %#v, want event type %q", detail.Events, tt.wantEventType)
			}

			assertNoActiveSessions(t, h.deps)
		})
	}
}

func readCLIHistoricalChannel(
	t *testing.T,
	deps commandDeps,
	channel string,
) NetworkChannelRecord {
	t.Helper()

	channelsOut := mustExecuteRoot(t, deps, "network", "channels", "-o", "json")
	var channels []NetworkChannelRecord
	if err := json.Unmarshal([]byte(channelsOut), &channels); err != nil {
		t.Fatalf("json.Unmarshal(network channels) error = %v", err)
	}
	for _, item := range channels {
		if item.Channel == channel {
			return item
		}
	}
	t.Fatalf("network channels missing %q: %#v", channel, channels)
	return NetworkChannelRecord{}
}

func containsCLITaskEventType(events []TaskEventRecord, want string) bool {
	for _, event := range events {
		if event.EventType == want {
			return true
		}
	}
	return false
}
