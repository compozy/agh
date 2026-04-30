//go:build integration && !windows

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const e2eACPHelperEnvKey = "AGH_TEST_E2E_ACP_HELPER"

type e2eACPAgent struct {
	conn *acpsdk.AgentSideConnection
}

func TestE2EACPHelperProcess(t *testing.T) {
	// This test body is only executed by the subprocess-backed ACP helper command.
	if os.Getenv(e2eACPHelperEnvKey) != "1" {
		return
	}

	agent := &e2eACPAgent{}
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.conn = conn
	<-conn.Done()
	os.Exit(0)
}

func TestStartRuntimeHarnessBootsRealDaemonAndExposesClients(t *testing.T) {
	t.Parallel()

	harness := StartRuntimeHarness(t, RuntimeHarnessOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var httpStatus aghcontract.DaemonStatusResponse
	if err := harness.HTTPJSON(ctx, "GET", "/api/daemon/status", nil, &httpStatus); err != nil {
		t.Fatalf("HTTP daemon status error = %v", err)
	}
	if got, want := httpStatus.Daemon.Status, "running"; got != want {
		t.Fatalf("httpStatus.Daemon.Status = %q, want %q", got, want)
	}

	var udsStatus aghcontract.DaemonStatusResponse
	if err := harness.UDSJSON(ctx, "GET", "/api/daemon/status", nil, &udsStatus); err != nil {
		t.Fatalf("UDS daemon status error = %v", err)
	}
	if got, want := udsStatus.Daemon.HTTPPort, harness.Config.HTTP.Port; got != want {
		t.Fatalf("udsStatus.Daemon.HTTPPort = %d, want %d", got, want)
	}

	var cliStatus aghcontract.DaemonStatusPayload
	if err := harness.CLI.RunJSON(ctx, &cliStatus, "daemon", "status", "-o", "json"); err != nil {
		t.Fatalf("CLI daemon status error = %v", err)
	}
	if got, want := cliStatus.Socket, harness.Config.Daemon.Socket; got != want {
		t.Fatalf("cliStatus.Socket = %q, want %q", got, want)
	}
}

func TestStartRuntimeHarnessRetriesHTTPPortConflicts(t *testing.T) {
	t.Parallel()

	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer func() {
		_ = blocker.Close()
	}()

	conflictPort := blocker.Addr().(*net.TCPAddr).Port
	harness := StartRuntimeHarness(t, RuntimeHarnessOptions{
		ConfigSeed: ConfigSeedOptions{
			HTTPPort: conflictPort,
		},
	})
	if got := harness.Config.HTTP.Port; got == conflictPort {
		t.Fatalf("harness.Config.HTTP.Port = %d, want retry onto a new port", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var status aghcontract.DaemonStatusResponse
	if err := harness.HTTPJSON(ctx, "GET", "/api/daemon/status", nil, &status); err != nil {
		t.Fatalf("HTTP daemon status after retry error = %v", err)
	}
	if got, want := status.Daemon.HTTPPort, harness.Config.HTTP.Port; got != want {
		t.Fatalf("status.Daemon.HTTPPort = %d, want %d", got, want)
	}
}

func TestStartRuntimeHarnessResolvesSeededWorkspaceThroughPublicSurface(t *testing.T) {
	t.Parallel()

	harness := StartRuntimeHarness(t, RuntimeHarnessOptions{
		Workspace: WorkspaceSeedOptions{
			Files: map[string]string{
				"README.md": "shared harness workspace",
			},
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if harness.WorkspaceID == "" {
		t.Fatal("harness.WorkspaceID = empty, want resolved workspace id")
	}

	workspace, err := harness.GetWorkspace(ctx, harness.WorkspaceID)
	if err != nil {
		t.Fatalf("GetWorkspace(%q) error = %v", harness.WorkspaceID, err)
	}
	if got, want := workspace.RootDir, harness.WorkspaceRoot; got != want {
		t.Fatalf("workspace.RootDir = %q, want %q", got, want)
	}
	if got, want := workspace.ID, harness.WorkspaceID; got != want {
		t.Fatalf("workspace.ID = %q, want %q", got, want)
	}
}

func TestStartRuntimeHarnessCapturesTranscriptAndEventsArtifacts(t *testing.T) {
	t.Parallel()

	harness := StartRuntimeHarness(t, RuntimeHarnessOptions{
		Env: map[string]string{
			e2eACPHelperEnvKey: "1",
		},
		ConfigSeed: ConfigSeedOptions{
			AgentDefs: []AgentSeed{{
				Name:        "coder",
				Provider:    "claude",
				Command:     e2eACPHelperCommand(t),
				Permissions: string(aghconfig.PermissionModeApproveReads),
				Prompt:      "You are a deterministic E2E helper.",
			}},
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	created, err := harness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     "coder",
		Name:          "artifact-demo",
		WorkspacePath: harness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	stream, err := harness.PromptSession(ctx, created.ID, "hello harness")
	if err != nil {
		t.Fatalf("PromptSession() error = %v", err)
	}
	if len(stream) == 0 {
		t.Fatal("prompt stream = empty, want agent_message/done events")
	}
	if stream[0].Event == "" {
		t.Fatalf("stream[0].Event = empty, want populated SSE event")
	}

	if err := harness.CaptureSessionTranscript(ctx, created.ID); err != nil {
		t.Fatalf("CaptureSessionTranscript() error = %v", err)
	}
	if err := harness.CaptureSessionEvents(ctx, created.ID); err != nil {
		t.Fatalf("CaptureSessionEvents() error = %v", err)
	}

	manifest := harness.Artifacts.Manifest()
	if got, want := len(manifest.Artifacts), 2; got != want {
		t.Fatalf("len(manifest.Artifacts) = %d, want %d", got, want)
	}

	transcriptPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindTranscript)
	if !ok {
		t.Fatal("ArtifactPath(transcript) = missing, want present")
	}
	transcriptBytes, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", transcriptPath, err)
	}
	transcript := string(transcriptBytes)
	if !strings.Contains(transcript, "echo:") || !strings.Contains(transcript, "hello harness") {
		t.Fatalf("transcript artifact = %s, want echoed assistant content", string(transcriptBytes))
	}

	eventsPath, ok := harness.Artifacts.ArtifactPath(ArtifactKindEvents)
	if !ok {
		t.Fatal("ArtifactPath(events) = missing, want present")
	}
	eventsBytes, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", eventsPath, err)
	}
	if !strings.Contains(string(eventsBytes), "agent_message") {
		t.Fatalf("events artifact = %s, want agent_message event", string(eventsBytes))
	}
}

func TestStartRuntimeHarnessRepeatedCyclesLeaveNoStaleDaemonArtifacts(t *testing.T) {
	for cycle := 0; cycle < 3; cycle++ {
		harness := StartRuntimeHarness(t, RuntimeHarnessOptions{})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		var httpStatus aghcontract.DaemonStatusResponse
		if err := harness.HTTPJSON(ctx, "GET", "/api/daemon/status", nil, &httpStatus); err != nil {
			cancel()
			t.Fatalf("cycle %d HTTP daemon status error = %v", cycle, err)
		}

		if err := harness.Stop(ctx); err != nil {
			cancel()
			t.Fatalf("cycle %d Stop() error = %v", cycle, err)
		}
		cancel()

		if _, err := os.Stat(harness.HomePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cycle %d daemon info stat error = %v, want os.ErrNotExist", cycle, err)
		}
		if _, err := os.Stat(harness.Config.Daemon.Socket); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("cycle %d socket stat error = %v, want os.ErrNotExist", cycle, err)
		}
	}
}

func TestStartRuntimeHarnessCLIStatusCanBeCapturedInRuntimeManifest(t *testing.T) {
	harness := StartRuntimeHarness(t, RuntimeHarnessOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stdout, stderr, err := harness.CLI.Run(ctx, "daemon", "status", "-o", "json")
	if err != nil {
		t.Fatalf("CLI.Run(daemon status) error = %v; stderr=%s", err, strings.TrimSpace(stderr))
	}

	var cliStatus aghcontract.DaemonStatusPayload
	if err := json.Unmarshal([]byte(stdout), &cliStatus); err != nil {
		t.Fatalf("json.Unmarshal(cli status) error = %v; stdout=%s", err, strings.TrimSpace(stdout))
	}
	if got, want := cliStatus.Socket, harness.Config.Daemon.Socket; got != want {
		t.Fatalf("cliStatus.Socket = %q, want %q", got, want)
	}
	if got, want := cliStatus.HTTPPort, harness.Config.HTTP.Port; got != want {
		t.Fatalf("cliStatus.HTTPPort = %d, want %d", got, want)
	}

	outputPath, err := harness.CaptureCLIOutput(
		"daemon status",
		[]string{"daemon", "status", "-o", "json"},
		stdout,
		stderr,
		nil,
	)
	if err != nil {
		t.Fatalf("CaptureCLIOutput() error = %v", err)
	}
	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", outputPath, err)
	}
	if !strings.Contains(string(outputBytes), `"transport": "cli"`) {
		t.Fatalf("CLI output artifact = %s, want CLI transport record", string(outputBytes))
	}

	manifestBytes, err := os.ReadFile(harness.RuntimeManifestPath())
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", harness.RuntimeManifestPath(), err)
	}
	var manifest RuntimeArtifactManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("json.Unmarshal(runtime manifest) error = %v", err)
	}
	if got, want := manifest.Transport.SocketPath, harness.Config.Daemon.Socket; got != want {
		t.Fatalf("manifest.Transport.SocketPath = %q, want %q", got, want)
	}
	if !runtimeManifestHasArtifact(manifest.CapturedArtifacts, ArtifactKindTransportOutputs, "transport_outputs") {
		t.Fatalf("manifest.CapturedArtifacts = %#v, want transport_outputs entry", manifest.CapturedArtifacts.Artifacts)
	}
}

func (a *e2eACPAgent) Authenticate(
	context.Context,
	acpsdk.AuthenticateRequest,
) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *e2eACPAgent) Initialize(
	context.Context,
	acpsdk.InitializeRequest,
) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AuthMethods:     []acpsdk.AuthMethod{},
	}, nil
}

func (a *e2eACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *e2eACPAgent) NewSession(
	context.Context,
	acpsdk.NewSessionRequest,
) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{SessionId: "e2e-helper-session"}, nil
}

func (a *e2eACPAgent) LoadSession(
	context.Context,
	acpsdk.LoadSessionRequest,
) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (a *e2eACPAgent) Prompt(
	ctx context.Context,
	params acpsdk.PromptRequest,
) (acpsdk.PromptResponse, error) {
	if a.conn != nil {
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText("echo: " + promptText(params.Prompt)),
		}); err != nil {
			return acpsdk.PromptResponse{}, err
		}
	}
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (a *e2eACPAgent) SetSessionMode(
	context.Context,
	acpsdk.SetSessionModeRequest,
) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func e2eACPHelperCommand(t testing.TB) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return shellquote.Join(bin, "-test.run=TestE2EACPHelperProcess")
}

func promptText(blocks []acpsdk.ContentBlock) string {
	lastText := ""
	for _, block := range blocks {
		switch {
		case block.Text != nil:
			lastText = block.Text.Text
		}
	}
	const userRequestMarker = "User request:"
	if idx := strings.LastIndex(lastText, userRequestMarker); idx >= 0 {
		return strings.TrimSpace(lastText[idx+len(userRequestMarker):])
	}
	return lastText
}

func runtimeManifestHasArtifact(manifest ArtifactManifest, kind ArtifactKind, path string) bool {
	for _, artifact := range manifest.Artifacts {
		if artifact.Kind == kind && artifact.Path == path {
			return true
		}
	}
	return false
}
