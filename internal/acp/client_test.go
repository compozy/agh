package acp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kballard/go-shellquote"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

const (
	testHelperEnvKey      = "AGH_TEST_ACP_HELPER"
	testHelperScenarioKey = "AGH_TEST_ACP_SCENARIO"
	testHelperFileKey     = "AGH_TEST_ACP_FILE"
)

func TestACPHelperProcess(t *testing.T) {
	if os.Getenv(testHelperEnvKey) != "1" {
		return
	}

	agent := &helperACPAgent{
		scenario: os.Getenv(testHelperScenarioKey),
		filePath: os.Getenv(testHelperFileKey),
	}
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.conn = conn
	<-conn.Done()
	os.Exit(0)
}

func TestParseCommandString(t *testing.T) {
	t.Parallel()

	command, args, err := parseCommandString(`npx -y "agent client" --flag='hello world'`)
	if err != nil {
		t.Fatalf("parseCommandString() error = %v", err)
	}
	if command != "npx" {
		t.Fatalf("parseCommandString() command = %q, want %q", command, "npx")
	}
	wantArgs := []string{"-y", "agent client", "--flag=hello world"}
	if !slices.Equal(args, wantArgs) {
		t.Fatalf("parseCommandString() args = %#v, want %#v", args, wantArgs)
	}
}

func TestPermissionPolicyModes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	policies := map[string]struct {
		mode       aghconfig.PermissionMode
		readOK     bool
		writeOK    bool
		terminalOK bool
	}{
		"deny-all": {
			mode:       aghconfig.PermissionModeDenyAll,
			readOK:     false,
			writeOK:    false,
			terminalOK: false,
		},
		"approve-reads": {
			mode:       aghconfig.PermissionModeApproveReads,
			readOK:     true,
			writeOK:    false,
			terminalOK: false,
		},
		"approve-all": {
			mode:       aghconfig.PermissionModeApproveAll,
			readOK:     true,
			writeOK:    true,
			terminalOK: true,
		},
	}

	for name, tc := range policies {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			policy, err := newPermissionPolicy(tc.mode, root)
			if err != nil {
				t.Fatalf("newPermissionPolicy() error = %v", err)
			}

			assertPermissionResult(t, policy.authorize(permissionReadTextFile), tc.readOK)
			assertPermissionResult(t, policy.authorize(permissionWriteTextFile), tc.writeOK)
			assertPermissionResult(t, policy.authorize(permissionCreateTerminal), tc.terminalOK)
		})
	}
}

func TestPermissionPolicyResolvePathSandbox(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	policy, err := newPermissionPolicy(aghconfig.PermissionModeApproveAll, root)
	if err != nil {
		t.Fatalf("newPermissionPolicy() error = %v", err)
	}

	insideFile := filepath.Join(root, "nested", "file.txt")
	resolvedInside, err := policy.resolvePath(insideFile)
	if err != nil {
		t.Fatalf("resolvePath(%q) error = %v", insideFile, err)
	}
	if !strings.HasSuffix(resolvedInside, filepath.Join("nested", "file.txt")) {
		t.Fatalf("resolvePath(%q) = %q, want suffix %q", insideFile, resolvedInside, filepath.Join("nested", "file.txt"))
	}

	if _, err := policy.resolvePath(filepath.Join(root, "..", "escape.txt")); !errors.Is(err, ErrPathOutsideWorkspace) {
		t.Fatalf("resolvePath(outside) error = %v, want ErrPathOutsideWorkspace", err)
	}
}

func TestTokenUsageParsing(t *testing.T) {
	t.Parallel()

	inputTokens := int64(10)
	outputTokens := int64(12)
	totalTokens := int64(22)
	thoughtTokens := int64(3)
	cacheReadTokens := int64(4)
	cacheWriteTokens := int64(5)
	used := int64(80)
	size := int64(100)
	amount := 1.25
	currency := "USD"

	promptUsage := tokenUsageFromPromptResponse("turn-1", &wireUsage{
		InputTokens:      &inputTokens,
		OutputTokens:     &outputTokens,
		TotalTokens:      &totalTokens,
		ThoughtTokens:    &thoughtTokens,
		CacheReadTokens:  &cacheReadTokens,
		CacheWriteTokens: &cacheWriteTokens,
	})
	if promptUsage.InputTokens == nil || *promptUsage.InputTokens != inputTokens {
		t.Fatalf("tokenUsageFromPromptResponse() input_tokens = %#v, want %d", promptUsage.InputTokens, inputTokens)
	}
	if promptUsage.CacheWriteTokens == nil || *promptUsage.CacheWriteTokens != cacheWriteTokens {
		t.Fatalf("tokenUsageFromPromptResponse() cache_write_tokens = %#v, want %d", promptUsage.CacheWriteTokens, cacheWriteTokens)
	}

	merged := promptUsage.Merge(tokenUsageFromUsageUpdate("turn-1", wireUsageUpdate{
		Used: &used,
		Size: &size,
		Cost: &wireCost{
			Amount:   &amount,
			Currency: &currency,
		},
	}))
	if merged.ContextUsed == nil || *merged.ContextUsed != used {
		t.Fatalf("merged.ContextUsed = %#v, want %d", merged.ContextUsed, used)
	}
	if merged.CostCurrency == nil || *merged.CostCurrency != currency {
		t.Fatalf("merged.CostCurrency = %#v, want %q", merged.CostCurrency, currency)
	}

	empty := tokenUsageFromPromptResponse("turn-2", nil)
	if !empty.IsZero() {
		t.Fatalf("tokenUsageFromPromptResponse(nil) should be zero, got %#v", empty)
	}
}

func TestPromptPrependsSystemPromptOnce(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "echo_prompt", "", StartOpts{
		SystemPrompt: "Memory context first.\nThen agent prompt.",
	})
	defer stopProcess(t, driver, proc)

	firstEventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-1",
		Message: "first request",
	})
	if err != nil {
		t.Fatalf("Prompt(first) error = %v", err)
	}
	firstEvents := collectEvents(t, firstEventsCh)
	if len(firstEvents) == 0 {
		t.Fatal("Prompt(first) returned no events")
	}
	if !strings.Contains(firstEvents[0].Text, "Session instructions") {
		t.Fatalf("first prompt text = %q, want injected system prompt prefix", firstEvents[0].Text)
	}
	if !strings.Contains(firstEvents[0].Text, "Memory context first.\nThen agent prompt.") {
		t.Fatalf("first prompt text = %q, want system prompt content", firstEvents[0].Text)
	}
	if !strings.Contains(firstEvents[0].Text, "User request:\n\nfirst request") {
		t.Fatalf("first prompt text = %q, want user request content", firstEvents[0].Text)
	}

	secondEventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-2",
		Message: "second request",
	})
	if err != nil {
		t.Fatalf("Prompt(second) error = %v", err)
	}
	secondEvents := collectEvents(t, secondEventsCh)
	if len(secondEvents) == 0 {
		t.Fatal("Prompt(second) returned no events")
	}
	if secondEvents[0].Text != "second request" {
		t.Fatalf("second prompt text = %q, want plain user request", secondEvents[0].Text)
	}
}

func TestPromptStreamsSessionUpdates(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-stream",
		Message: "hello",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if len(events) == 0 {
		t.Fatal("Prompt() returned no events")
	}

	var eventTypes []string
	for _, event := range events {
		eventTypes = append(eventTypes, event.Type)
	}
	if !slices.Contains(eventTypes, EventTypeAgentMessage) {
		t.Fatalf("Prompt() event types = %#v, want agent message", eventTypes)
	}
	if !slices.Contains(eventTypes, EventTypeThought) {
		t.Fatalf("Prompt() event types = %#v, want thought", eventTypes)
	}
	if !slices.Contains(eventTypes, EventTypeToolCall) {
		t.Fatalf("Prompt() event types = %#v, want tool call", eventTypes)
	}
	if !slices.Contains(eventTypes, EventTypeDone) {
		t.Fatalf("Prompt() event types = %#v, want done", eventTypes)
	}
	if proc.SessionID != "sess-new" {
		t.Fatalf("Start() session id = %q, want %q", proc.SessionID, "sess-new")
	}
	if !slices.Equal(proc.Caps.SupportedModes, []string{"new-mode"}) {
		t.Fatalf("Start() supported modes = %#v, want %#v", proc.Caps.SupportedModes, []string{"new-mode"})
	}
	if !slices.Equal(proc.Caps.SupportedModels, []string{"new-model"}) {
		t.Fatalf("Start() supported models = %#v, want %#v", proc.Caps.SupportedModels, []string{"new-model"})
	}
}

func TestStartResumeUsesLoadSession(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "load_session", "", StartOpts{
		ResumeSessionID: "sess-existing",
	})
	defer stopProcess(t, driver, proc)

	if proc.SessionID != "sess-existing" {
		t.Fatalf("Start() session id = %q, want %q", proc.SessionID, "sess-existing")
	}
	if !proc.Caps.SupportsLoadSession {
		t.Fatal("Start() SupportsLoadSession = false, want true")
	}
	if !slices.Equal(proc.Caps.SupportedModes, []string{"loaded-mode"}) {
		t.Fatalf("Start() supported modes = %#v, want %#v", proc.Caps.SupportedModes, []string{"loaded-mode"})
	}
	if !slices.Equal(proc.Caps.SupportedModels, []string{"loaded-model"}) {
		t.Fatalf("Start() supported models = %#v, want %#v", proc.Caps.SupportedModels, []string{"loaded-model"})
	}
}

func TestProcessCrashDetected(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "crash_on_prompt", "", StartOpts{})

	eventsCh, err := driver.Prompt(testContext(t), proc, PromptRequest{
		TurnID:  "turn-crash",
		Message: "trigger crash",
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if len(events) == 0 || events[len(events)-1].Type != EventTypeError {
		t.Fatalf("Prompt() last event = %#v, want error", events)
	}

	waitErr := waitForProcess(t, proc)
	if waitErr == nil {
		t.Fatal("Wait() error = nil, want process crash")
	}
}

func TestDriverApprovePermissionValidationAndForwarding(t *testing.T) {
	t.Parallel()

	driver := New(WithPermissionTimeout(123 * time.Millisecond))
	if driver.permissionWait != 123*time.Millisecond {
		t.Fatalf("permissionWait = %v, want %v", driver.permissionWait, 123*time.Millisecond)
	}

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	requestID, pending := proc.registerPendingPermission("turn-1", acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{ToolCallId: "tool-1"},
	})

	if err := driver.ApprovePermission(context.Background(), proc, ApproveRequest{
		RequestID: requestID,
		Decision:  string(decisionAllowOnce),
	}); err != nil {
		t.Fatalf("ApprovePermission() error = %v", err)
	}
	select {
	case decision := <-pending.response:
		if decision != decisionAllowOnce {
			t.Fatalf("pending response = %q, want %q", decision, decisionAllowOnce)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pending permission response")
	}

	if err := driver.ApprovePermission(context.Background(), nil, ApproveRequest{
		RequestID: "req-1",
		Decision:  string(decisionAllowOnce),
	}); err == nil {
		t.Fatal("ApprovePermission(nil proc) error = nil, want non-nil")
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := driver.ApprovePermission(canceledCtx, proc, ApproveRequest{
		RequestID: "req-1",
		Decision:  string(decisionAllowOnce),
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ApprovePermission(canceled ctx) error = %v, want context.Canceled", err)
	}
}

func startHelperProcess(t *testing.T, driver *Driver, scenario string, filePath string, overrides StartOpts) *AgentProcess {
	t.Helper()

	command := helperCommand(t)
	opts := StartOpts{
		AgentName:   "helper",
		Command:     command,
		Cwd:         t.TempDir(),
		Env:         helperEnv(scenario, filePath),
		Permissions: aghconfig.PermissionModeApproveAll,
	}
	if overrides.AgentName != "" {
		opts.AgentName = overrides.AgentName
	}
	if overrides.Command != "" {
		opts.Command = overrides.Command
	}
	if overrides.Cwd != "" {
		opts.Cwd = overrides.Cwd
	}
	if overrides.Env != nil {
		opts.Env = overrides.Env
	}
	if overrides.Permissions != "" {
		opts.Permissions = overrides.Permissions
	}
	if overrides.MCPServers != nil {
		opts.MCPServers = overrides.MCPServers
	}
	if overrides.SystemPrompt != "" {
		opts.SystemPrompt = overrides.SystemPrompt
	}
	opts.ResumeSessionID = overrides.ResumeSessionID

	proc, err := driver.Start(testContext(t), opts)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return proc
}

func stopProcess(t *testing.T, driver *Driver, proc *AgentProcess) {
	t.Helper()
	if proc == nil {
		return
	}
	if err := driver.Stop(testContext(t), proc); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func waitForProcess(t *testing.T, proc *AgentProcess) error {
	t.Helper()
	select {
	case <-proc.Done():
		return proc.Wait()
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for process exit")
		return nil
	}
}

func collectEvents(t *testing.T, eventsCh <-chan AgentEvent) []AgentEvent {
	t.Helper()

	events := make([]AgentEvent, 0, 8)
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				return events
			}
			events = append(events, event)
		case <-timeout.C:
			t.Fatalf("timeout waiting for prompt events; collected %#v", events)
		}
	}
}

func helperCommand(t *testing.T) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return shellquote.Join(bin, "-test.run=TestACPHelperProcess")
}

func helperEnv(scenario string, filePath string) []string {
	env := append([]string(nil), os.Environ()...)
	env = append(env,
		testHelperEnvKey+"=1",
		testHelperScenarioKey+"="+scenario,
	)
	if filePath != "" {
		env = append(env, testHelperFileKey+"="+filePath)
	}
	return env
}

func assertPermissionResult(t *testing.T, err error, wantOK bool) {
	t.Helper()
	if wantOK && err != nil {
		t.Fatalf("authorize() error = %v, want nil", err)
	}
	if !wantOK && !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("authorize() error = %v, want ErrPermissionDenied", err)
	}
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

type helperACPAgent struct {
	conn     *acpsdk.AgentSideConnection
	scenario string
	filePath string
}

func (a *helperACPAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *helperACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: a.scenario == "load_session",
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (a *helperACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *helperACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{
		SessionId: "sess-new",
		Modes:     helperModeState("new-mode"),
		Models:    helperModelState("new-model"),
	}, nil
}

func (a *helperACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{
		Modes:  helperModeState("loaded-mode"),
		Models: helperModelState("loaded-model"),
	}, nil
}

func (a *helperACPAgent) Prompt(ctx context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	switch a.scenario {
	case "crash_on_prompt":
		os.Exit(23)
	case "echo_prompt":
		text := ""
		if len(params.Prompt) > 0 && params.Prompt[0].Text != nil {
			text = params.Prompt[0].Text.Text
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(text),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	case "fs_read":
		response, err := a.conn.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			SessionId: params.SessionId,
			Path:      a.filePath,
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(response.Content),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	case "permission":
		title := "permission request"
		locationPath := a.filePath
		if locationPath == "" {
			locationPath = filepath.Join("/", "workspace", "demo.txt")
		}
		outcome, err := a.conn.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
			SessionId: params.SessionId,
			Options: []acpsdk.PermissionOption{
				{OptionId: "allow-once", Name: "allow once", Kind: acpsdk.PermissionOptionKindAllowOnce},
				{OptionId: "allow-always", Name: "allow always", Kind: acpsdk.PermissionOptionKindAllowAlways},
				{OptionId: "reject-once", Name: "reject once", Kind: acpsdk.PermissionOptionKindRejectOnce},
				{OptionId: "reject-always", Name: "reject always", Kind: acpsdk.PermissionOptionKindRejectAlways},
			},
			ToolCall: acpsdk.RequestPermissionToolCall{
				ToolCallId: "tool-1",
				Title:      &title,
				Locations: []acpsdk.ToolCallLocation{
					{Path: locationPath},
				},
			},
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		selected := "cancelled"
		if outcome.Outcome.Selected != nil {
			selected = string(outcome.Outcome.Selected.OptionId)
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(selected),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	default:
		updates := []acpsdk.SessionUpdate{
			acpsdk.UpdateAgentMessageText("hello"),
			acpsdk.UpdateAgentThoughtText("thinking"),
			acpsdk.StartToolCall("tool-1", "Read file", acpsdk.WithStartKind(acpsdk.ToolKindRead), acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress)),
			acpsdk.UpdateToolCall("tool-1", acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted), acpsdk.WithUpdateTitle("Read file")),
		}
		for _, update := range updates {
			if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
				SessionId: params.SessionId,
				Update:    update,
			}); err != nil {
				return acpsdk.PromptResponse{}, err
			}
		}
	}

	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (a *helperACPAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func helperModeState(id string) *acpsdk.SessionModeState {
	return &acpsdk.SessionModeState{
		CurrentModeId: acpsdk.SessionModeId(id),
		AvailableModes: []acpsdk.SessionMode{
			{Id: acpsdk.SessionModeId(id), Name: id},
		},
	}
}

func helperModelState(id string) *acpsdk.SessionModelState {
	return &acpsdk.SessionModelState{
		CurrentModelId: acpsdk.ModelId(id),
		AvailableModels: []acpsdk.ModelInfo{
			{ModelId: acpsdk.ModelId(id), Name: id},
		},
	}
}
