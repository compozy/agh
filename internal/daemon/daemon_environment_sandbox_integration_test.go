//go:build integration && !windows

package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	aghcontract "github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	sessionpkg "github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	e2etest "github.com/pedronauck/agh/internal/testutil/e2e"
)

const (
	daemonEnvironmentHelperEnvKey         = "AGH_TEST_DAEMON_ENV_HELPER"
	daemonEnvironmentHelperScenarioEnvKey = "AGH_TEST_DAEMON_ENV_SCENARIO"
	daemonEnvironmentFixtureAgentName     = "environment-helper"
	daemonEnvironmentProfileName          = "local-sandbox"
	daemonEnvironmentScenarioAllowed      = "allowed"
	daemonEnvironmentScenarioBlocked      = "blocked"
)

func TestDaemonEnvironmentACPHelperProcess(t *testing.T) {
	if os.Getenv(daemonEnvironmentHelperEnvKey) != "1" {
		return
	}

	agent := &daemonEnvironmentACPAgent{
		scenario: strings.TrimSpace(os.Getenv(daemonEnvironmentHelperScenarioEnvKey)),
	}
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.conn = conn
	<-conn.Done()
	os.Exit(0)
}

func TestDaemonE2ELocalEnvironmentAllowsToolExecutionAndPersistsMetadata(t *testing.T) {
	harness := startEnvironmentRuntimeHarness(t, daemonEnvironmentScenarioAllowed, aghconfig.PermissionModeApproveAll)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		sessionID      string
		diagnostics    e2etest.ToolHostDiagnosticsArtifact
		sideEffectPath = filepath.Join(harness.WorkspaceRoot, "toolhost", "allowed.txt")
	)
	registerEnvironmentRuntimeArtifacts(t, harness, &sessionID, &diagnostics)

	sessionInfo, stream := mustRunEnvironmentScenarioSession(
		t,
		ctx,
		harness,
		"allowed-local-runtime",
		"write an allowed runtime file",
	)
	sessionID = sessionInfo.ID
	if len(stream) == 0 {
		t.Fatal("prompt stream = empty, want runtime events")
	}
	if err := harness.StopSession(ctx, sessionID); err != nil {
		t.Fatalf("StopSession(%q) error = %v", sessionID, err)
	}

	waitForRuntimeCondition(t, "allowed environment session stopped", 10*time.Second, func() bool {
		current, err := harness.GetSession(ctx, sessionID)
		return err == nil && current.State == sessionpkg.StateStopped
	})

	sessionInfo, err := harness.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) error = %v", sessionID, err)
	}
	if got, want := sessionInfo.State, sessionpkg.StateStopped; got != want {
		t.Fatalf("sessionInfo.State = %q, want %q", got, want)
	}
	if got, want := sessionInfo.StopReason, store.StopUserCanceled; got != want {
		t.Fatalf("sessionInfo.StopReason = %q, want %q", got, want)
	}
	if sessionInfo.Environment == nil {
		t.Fatal("sessionInfo.Environment = nil, want local environment metadata")
	}
	if got, want := sessionInfo.Environment.Backend, string(environment.BackendLocal); got != want {
		t.Fatalf("sessionInfo.Environment.Backend = %q, want %q", got, want)
	}
	if got, want := sessionInfo.Environment.Profile, daemonEnvironmentProfileName; got != want {
		t.Fatalf("sessionInfo.Environment.Profile = %q, want %q", got, want)
	}
	if got, want := sessionInfo.Environment.State, "stopped"; got != want {
		t.Fatalf("sessionInfo.Environment.State = %q, want %q", got, want)
	}

	content, err := os.ReadFile(sideEffectPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", sideEffectPath, err)
	}
	if got, want := string(content), "allowed-runtime"; got != want {
		t.Fatalf("allowed side effect content = %q, want %q", got, want)
	}

	meta := mustReadSessionMeta(t, harness, sessionID)
	if meta.Environment == nil {
		t.Fatal("meta.Environment = nil, want persisted environment metadata")
	}
	if got, want := meta.Environment.RuntimeRootDir, harness.WorkspaceRoot; got != want {
		t.Fatalf("meta.Environment.RuntimeRootDir = %q, want %q", got, want)
	}
	if got, want := meta.Environment.Profile, daemonEnvironmentProfileName; got != want {
		t.Fatalf("meta.Environment.Profile = %q, want %q", got, want)
	}
	if got, want := meta.Environment.State, "stopped"; got != want {
		t.Fatalf("meta.Environment.State = %q, want %q", got, want)
	}

	environmentArtifact, err := harness.SessionEnvironmentArtifact(ctx, sessionID)
	if err != nil {
		t.Fatalf("SessionEnvironmentArtifact(%q) error = %v", sessionID, err)
	}
	if environmentArtifact.Persisted == nil {
		t.Fatal("environmentArtifact.Persisted = nil, want persisted metadata in artifact helper")
	}
	diagnostics = e2etest.ToolHostDiagnosticsArtifact{
		SessionID: sessionID,
		Operations: []e2etest.ToolHostOperationDiagnostic{{
			Operation:        "write_text_file",
			Path:             "toolhost/allowed.txt",
			Outcome:          e2etest.ToolHostOutcomeAllowed,
			SideEffectPath:   sideEffectPath,
			SideEffectExists: true,
		}},
	}
}

func TestDaemonE2ELocalEnvironmentBlockedOperationLeavesFailureDiagnostics(t *testing.T) {
	harness := startEnvironmentRuntimeHarness(t, daemonEnvironmentScenarioBlocked, aghconfig.PermissionModeApproveReads)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		sessionID      string
		diagnostics    e2etest.ToolHostDiagnosticsArtifact
		sideEffectPath = filepath.Join(harness.WorkspaceRoot, "toolhost", "blocked.txt")
	)
	registerEnvironmentRuntimeArtifacts(t, harness, &sessionID, &diagnostics)

	sessionInfo, stream := mustRunEnvironmentScenarioSession(
		t,
		ctx,
		harness,
		"blocked-local-runtime",
		"attempt a blocked runtime write",
	)
	sessionID = sessionInfo.ID
	if len(stream) == 0 {
		t.Fatal("prompt stream = empty, want runtime events")
	}
	if err := harness.StopSession(ctx, sessionID); err != nil {
		t.Fatalf("StopSession(%q) error = %v", sessionID, err)
	}

	waitForRuntimeCondition(t, "blocked environment session stopped", 10*time.Second, func() bool {
		current, err := harness.GetSession(ctx, sessionID)
		return err == nil && current.State == sessionpkg.StateStopped
	})

	sessionInfo, err := harness.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSession(%q) error = %v", sessionID, err)
	}
	if got, want := sessionInfo.State, sessionpkg.StateStopped; got != want {
		t.Fatalf("sessionInfo.State = %q, want %q", got, want)
	}
	if sessionInfo.Environment == nil {
		t.Fatal("sessionInfo.Environment = nil, want stopped environment metadata")
	}
	if got, want := sessionInfo.Environment.Backend, string(environment.BackendLocal); got != want {
		t.Fatalf("sessionInfo.Environment.Backend = %q, want %q", got, want)
	}
	if got, want := sessionInfo.Environment.Profile, daemonEnvironmentProfileName; got != want {
		t.Fatalf("sessionInfo.Environment.Profile = %q, want %q", got, want)
	}

	if _, err := os.Stat(sideEffectPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want not exist", sideEffectPath, err)
	}

	eventsResp := mustSessionEvents(t, ctx, harness, sessionID)
	events := decodeAgentEvents(t, eventsResp.Events)
	blockedMessage := findAgentMessageContaining(events, "blocked by approve-reads")
	if blockedMessage == "" {
		t.Fatalf("events = %#v, want blocked-operation failure signal", events)
	}

	meta := mustReadSessionMeta(t, harness, sessionID)
	if meta.Environment == nil {
		t.Fatal("meta.Environment = nil, want persisted metadata after blocked operation")
	}
	if got, want := meta.Environment.State, "stopped"; got != want {
		t.Fatalf("meta.Environment.State = %q, want %q", got, want)
	}

	environmentArtifact, err := harness.SessionEnvironmentArtifact(ctx, sessionID)
	if err != nil {
		t.Fatalf("SessionEnvironmentArtifact(%q) error = %v", sessionID, err)
	}
	if environmentArtifact.API == nil {
		t.Fatal("environmentArtifact.API = nil, want public session environment projection")
	}

	diagnostics = e2etest.ToolHostDiagnosticsArtifact{
		SessionID: sessionID,
		Operations: []e2etest.ToolHostOperationDiagnostic{{
			Operation:        "write_text_file",
			Path:             "toolhost/blocked.txt",
			Outcome:          e2etest.ToolHostOutcomeBlocked,
			Error:            blockedMessage,
			SideEffectPath:   sideEffectPath,
			SideEffectExists: false,
		}},
	}
}

type daemonEnvironmentACPAgent struct {
	conn     *acpsdk.AgentSideConnection
	scenario string
}

func (a *daemonEnvironmentACPAgent) Authenticate(
	context.Context,
	acpsdk.AuthenticateRequest,
) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *daemonEnvironmentACPAgent) Initialize(
	context.Context,
	acpsdk.InitializeRequest,
) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (a *daemonEnvironmentACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *daemonEnvironmentACPAgent) NewSession(
	context.Context,
	acpsdk.NewSessionRequest,
) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{SessionId: "daemon-environment-helper"}, nil
}

func (a *daemonEnvironmentACPAgent) LoadSession(
	context.Context,
	acpsdk.LoadSessionRequest,
) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (a *daemonEnvironmentACPAgent) Prompt(
	ctx context.Context,
	params acpsdk.PromptRequest,
) (acpsdk.PromptResponse, error) {
	switch a.scenario {
	case daemonEnvironmentScenarioAllowed:
		return a.promptAllowed(ctx, params)
	case daemonEnvironmentScenarioBlocked:
		return a.promptBlocked(ctx, params)
	default:
		return acpsdk.PromptResponse{}, fmt.Errorf("unknown environment helper scenario %q", a.scenario)
	}
}

func (a *daemonEnvironmentACPAgent) SetSessionMode(
	context.Context,
	acpsdk.SetSessionModeRequest,
) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func (a *daemonEnvironmentACPAgent) promptAllowed(
	ctx context.Context,
	params acpsdk.PromptRequest,
) (acpsdk.PromptResponse, error) {
	const relativePath = "toolhost/allowed.txt"

	if _, err := a.conn.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
		SessionId: params.SessionId,
		Path:      relativePath,
		Content:   "allowed-runtime",
	}); err != nil {
		return acpsdk.PromptResponse{}, err
	}

	readResp, err := a.conn.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
		SessionId: params.SessionId,
		Path:      relativePath,
	})
	if err != nil {
		return acpsdk.PromptResponse{}, err
	}

	return a.sendMessageAndEndTurn(ctx, params.SessionId, "allowed:"+readResp.Content)
}

func (a *daemonEnvironmentACPAgent) promptBlocked(
	ctx context.Context,
	params acpsdk.PromptRequest,
) (acpsdk.PromptResponse, error) {
	const relativePath = "toolhost/blocked.txt"

	message := "blocked:write unexpectedly succeeded"
	if _, err := a.conn.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
		SessionId: params.SessionId,
		Path:      relativePath,
		Content:   "should-not-write",
	}); err != nil {
		message = "blocked:" + err.Error()
	}

	return a.sendMessageAndEndTurn(ctx, params.SessionId, message)
}

func (a *daemonEnvironmentACPAgent) sendMessageAndEndTurn(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	message string,
) (acpsdk.PromptResponse, error) {
	if a.conn != nil {
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(message),
		}); err != nil {
			return acpsdk.PromptResponse{}, err
		}
	}
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func startEnvironmentRuntimeHarness(
	t testing.TB,
	scenario string,
	permissions aghconfig.PermissionMode,
) *e2etest.RuntimeHarness {
	t.Helper()

	return e2etest.StartRuntimeHarness(t, e2etest.RuntimeHarnessOptions{
		Env: map[string]string{
			daemonEnvironmentHelperEnvKey:         "1",
			daemonEnvironmentHelperScenarioEnvKey: scenario,
		},
		ConfigSeed: e2etest.ConfigSeedOptions{
			DefaultAgent:       daemonEnvironmentFixtureAgentName,
			DefaultEnvironment: daemonEnvironmentProfileName,
			Environments: map[string]aghconfig.EnvironmentProfile{
				daemonEnvironmentProfileName: {
					Backend:     string(environment.BackendLocal),
					Persistence: string(environment.PersistenceReuse),
				},
			},
			AgentDefs: []e2etest.AgentSeed{{
				Name:        daemonEnvironmentFixtureAgentName,
				Provider:    "claude",
				Command:     daemonEnvironmentHelperCommand(t),
				Permissions: string(permissions),
				Prompt:      "You are a deterministic environment runtime helper.",
			}},
		},
		Workspace: e2etest.WorkspaceSeedOptions{
			Files: map[string]string{
				"README.md": "environment runtime workspace",
			},
		},
	})
}

func mustRunEnvironmentScenarioSession(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	name string,
	message string,
) (aghcontract.SessionPayload, []e2etest.SSEEvent) {
	t.Helper()

	created, err := harness.CreateSession(ctx, aghcontract.CreateSessionRequest{
		AgentName:     daemonEnvironmentFixtureAgentName,
		Name:          name,
		WorkspacePath: harness.WorkspaceRoot,
	})
	if err != nil {
		t.Fatalf("CreateSession(%q) error = %v", name, err)
	}

	stream, err := harness.PromptSession(ctx, created.ID, message)
	if err != nil {
		t.Fatalf("PromptSession(%q) error = %v", created.ID, err)
	}
	return created, stream
}

func registerEnvironmentRuntimeArtifacts(
	t testing.TB,
	harness *e2etest.RuntimeHarness,
	sessionID *string,
	diagnostics *e2etest.ToolHostDiagnosticsArtifact,
) {
	t.Helper()

	t.Cleanup(func() {
		trimmedSessionID := strings.TrimSpace(derefStringValue(sessionID))
		if trimmedSessionID == "" {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := harness.CaptureSessionTranscript(ctx, trimmedSessionID); err != nil {
			t.Logf("CaptureSessionTranscript(%q) error = %v", trimmedSessionID, err)
		}
		if err := harness.CaptureSessionEvents(ctx, trimmedSessionID); err != nil {
			t.Logf("CaptureSessionEvents(%q) error = %v", trimmedSessionID, err)
		}
		if err := harness.CaptureSessionEnvironment(ctx, trimmedSessionID); err != nil {
			t.Logf("CaptureSessionEnvironment(%q) error = %v", trimmedSessionID, err)
		}
		if diagnostics != nil && len(diagnostics.Operations) > 0 {
			if err := harness.CaptureToolHostDiagnosticsJSON(*diagnostics); err != nil {
				t.Logf("CaptureToolHostDiagnosticsJSON(%q) error = %v", trimmedSessionID, err)
			}
		}
	})
}

func daemonEnvironmentHelperCommand(t testing.TB) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return shellquote.Join(bin, "-test.run=TestDaemonEnvironmentACPHelperProcess")
}

func mustSessionEvents(
	t testing.TB,
	ctx context.Context,
	harness *e2etest.RuntimeHarness,
	sessionID string,
) aghcontract.SessionEventsResponse {
	t.Helper()

	events, err := harness.SessionEvents(ctx, sessionID)
	if err != nil {
		t.Fatalf("SessionEvents(%q) error = %v", sessionID, err)
	}
	return events
}

func findAgentMessageContaining(events []aghcontract.AgentEventPayload, fragment string) string {
	want := strings.TrimSpace(fragment)
	for _, event := range events {
		if event.Type != "agent_message" {
			continue
		}
		if strings.Contains(event.Text, want) {
			return event.Text
		}
	}
	return ""
}
