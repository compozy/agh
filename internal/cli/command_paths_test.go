package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"syscall"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/procutil"
)

type stubRunner struct {
	ran bool
}

func (s *stubRunner) Run(context.Context) error {
	s.ran = true
	return nil
}

func TestCommandPathsAndHelpers(t *testing.T) {
	t.Parallel()

	statusSession := SessionRecord{
		ID:            "sess-1",
		AgentName:     "coder",
		WorkspaceID:   "ws-1",
		WorkspacePath: "/workspace/project",
		State:         "active",
		CreatedAt:     fixedTestNow,
		UpdatedAt:     fixedTestNow,
	}

	getCalls := 0
	client := stubClient{
		getAgentFn: func(context.Context, string) (AgentRecord, error) {
			return AgentRecord{Name: "coder", Provider: "fake", Prompt: "hi"}, nil
		},
		networkStatusFn: func(context.Context) (NetworkStatusRecord, error) {
			return NetworkStatusRecord{Enabled: true, Status: "running"}, nil
		},
		networkPeersFn: func(context.Context, NetworkPeersQuery) ([]NetworkPeerRecord, error) {
			return []NetworkPeerRecord{{PeerID: "reviewer.sess-1", Space: "builders"}}, nil
		},
		networkSpacesFn: func(context.Context) ([]NetworkSpaceRecord, error) {
			return []NetworkSpaceRecord{{Space: "builders", PeerCount: 1}}, nil
		},
		networkSendFn: func(context.Context, NetworkSendRequest) (NetworkSendRecord, error) {
			return NetworkSendRecord{ID: "msg-1", SessionID: "sess-1", Space: "builders", Kind: "say"}, nil
		},
		networkInboxFn: func(context.Context, string) ([]NetworkEnvelopeRecord, error) {
			return []NetworkEnvelopeRecord{{ID: "msg-1", Kind: "say", Space: "builders", From: "reviewer.sess-1"}}, nil
		},
		observeEventsFn: func(context.Context, ObserveEventQuery) ([]ObserveEventRecord, error) {
			return []ObserveEventRecord{{ID: "sum-1", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: fixedTestNow}}, nil
		},
		streamObserveEventsFn: func(_ context.Context, _ ObserveEventQuery, _ string, handler SSEHandler) error {
			return handler(SSEEvent{Event: "done", Data: mustJSON(t, ObserveEventRecord{ID: "sum-1", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: fixedTestNow})})
		},
		observeHealthFn: func(context.Context) (HealthStatus, error) {
			return HealthStatus{Status: "ok", UptimeSeconds: 10}, nil
		},
		getSessionFn: func(context.Context, string) (SessionRecord, error) {
			getCalls++
			if getCalls == 1 {
				return statusSession, nil
			}
			stopped := statusSession
			stopped.State = "stopped"
			return stopped, nil
		},
		resumeSessionFn: func(context.Context, string) (SessionRecord, error) {
			return statusSession, nil
		},
		streamSessionFn: func(_ context.Context, _ string, _ SessionEventQuery, _ string, handler SSEHandler) error {
			return handler(SSEEvent{Event: "session_stopped"})
		},
		sessionHistoryFn: func(context.Context, string, SessionEventQuery) ([]TurnHistoryRecord, error) {
			return []TurnHistoryRecord{{TurnID: "turn-1"}}, nil
		},
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{Status: "running", PID: 10, StartedAt: fixedTestNow}, nil
		},
		getChannelFn: func(context.Context, string) (ChannelRecord, error) {
			return ChannelRecord{ID: "chan-1", Scope: "global", Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: "ready"}, nil
		},
		channelRoutesFn: func(context.Context, string) ([]ChannelRouteRecord, error) {
			return []ChannelRouteRecord{{RoutingKeyHash: "hash-1", Scope: "global", ChannelInstanceID: "chan-1", PeerID: "peer-1", SessionID: "sess-1", AgentName: "coder", LastActivityAt: fixedTestNow}}, nil
		},
		testChannelDeliveryFn: func(context.Context, string, ChannelTestDeliveryRequest) (ChannelTestDeliveryRecord, error) {
			return ChannelTestDeliveryRecord{Status: "resolved", DeliveryTarget: DeliveryTargetRecord{ChannelInstanceID: "chan-1", PeerID: "peer-1", Mode: "reply"}}, nil
		},
	}
	deps := newTestDeps(t, client)
	runner := &stubRunner{}
	deps.newDaemon = func() (daemonRunner, error) { return runner, nil }

	tests := [][]string{
		{"agent", "info", "coder", "-o", "json"},
		{"network", "status", "-o", "json"},
		{"network", "peers", "builders", "-o", "json"},
		{"network", "spaces", "-o", "json"},
		{"network", "send", "--session", "sess-1", "--space", "builders", "--kind", "say", "--body", `{"text":"hello"}`, "-o", "json"},
		{"network", "inbox", "--session", "sess-1", "-o", "json"},
		{"observe", "events", "-o", "json"},
		{"observe", "events", "--follow", "-o", "json"},
		{"observe", "health", "-o", "json"},
		{"channel", "get", "chan-1", "-o", "json"},
		{"channel", "routes", "chan-1", "-o", "json"},
		{"channel", "test-delivery", "chan-1", "--peer-id", "peer-1", "--mode", "reply", "-o", "json"},
		{"session", "status", "sess-1", "-o", "json"},
		{"session", "resume", "sess-1", "-o", "json"},
		{"session", "wait", "sess-1", "-o", "json"},
		{"session", "history", "sess-1", "-o", "json"},
		{"daemon", "status", "-o", "json"},
	}

	for _, args := range tests {
		if _, _, err := executeRootCommand(t, deps, args...); err != nil {
			t.Fatalf("executeRootCommand(%v) error = %v", args, err)
		}
	}

	if _, _, err := executeRootCommand(t, deps, "daemon", "start", "--foreground"); err != nil {
		t.Fatalf("daemon start --foreground error = %v", err)
	}
	if !runner.ran {
		t.Fatal("daemon runner did not execute")
	}

	if wd, err := currentWorkingDirectory(deps); err != nil || wd != "/workspace/project" {
		t.Fatalf("currentWorkingDirectory() = %q, %v", wd, err)
	}

	if err := procutil.Signal(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Fatalf("procutil.Signal(os.Getpid(), 0) error = %v", err)
	}
	if !procutil.Alive(os.Getpid()) {
		t.Fatal("procutil.Alive(os.Getpid()) = false, want true")
	}
}

func TestExecuteContextVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := ExecuteContext(context.Background(), []string{"version", "-o", "json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("ExecuteContext(version) code = %d, want 0", code)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(version) error = %v", err)
	}
	if _, ok := payload["Version"]; !ok {
		t.Fatalf("version payload = %#v, want Version field", payload)
	}
}

func TestDaemonStatusFallbackStartingAndStopped(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{}, os.ErrNotExist
		},
	})
	info := aghdaemon.Info{PID: 42, StartedAt: fixedTestNow}
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) { return info, nil }
	deps.processAlive = func(pid int) bool { return pid == 42 }

	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		t.Fatalf("loadRuntimeContext() error = %v", err)
	}

	status, err := daemonStatusFromDeps(context.Background(), deps, runtime)
	if err != nil {
		t.Fatalf("daemonStatusFromDeps(starting) error = %v", err)
	}
	if status.Status != "starting" {
		t.Fatalf("starting status = %q, want %q", status.Status, "starting")
	}

	deps.processAlive = func(int) bool { return false }
	status, err = daemonStatusFromDeps(context.Background(), deps, runtime)
	if err != nil {
		t.Fatalf("daemonStatusFromDeps(stopped) error = %v", err)
	}
	if status.Status != "stopped" {
		t.Fatalf("stopped status = %q, want %q", status.Status, "stopped")
	}
}

func TestWriteCommandOutputErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := executeRootCommand(t, newTestDeps(t, stubClient{}), "version", "-o", "bogus"); err == nil || !strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("invalid output error = %v, want invalid output format", err)
	}
}

func TestDaemonStartRejectsNilDetachedProcess(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{})
	deps.spawnDetached = func(aghconfig.HomePaths) (daemonProcess, error) {
		return nil, nil
	}

	if _, _, err := executeRootCommand(t, deps, "daemon", "start", "-o", "json"); err == nil || !strings.Contains(err.Error(), "detached daemon process is required") {
		t.Fatalf("daemon start nil detached process error = %v, want detached daemon process is required", err)
	}
}
