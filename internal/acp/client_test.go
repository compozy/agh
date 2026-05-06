package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kballard/go-shellquote"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/toolruntime"
)

const (
	testHelperEnvKey      = "AGH_TEST_ACP_HELPER"
	testHelperScenarioKey = "AGH_TEST_ACP_SCENARIO"
	testHelperFileKey     = "AGH_TEST_ACP_FILE"
	testHelperCaptureKey  = "AGH_TEST_ACP_CAPTURE_FILE"
	testWrapperEnvKey     = "AGH_TEST_ACP_WRAPPER"
)

func TestACPHelperProcess(_ *testing.T) {
	if os.Getenv(testHelperEnvKey) != "1" {
		return
	}

	agent := &helperACPAgent{
		scenario: os.Getenv(testHelperScenarioKey),
		filePath: os.Getenv(testHelperFileKey),
	}
	input := io.Reader(os.Stdin)
	capturePath := strings.TrimSpace(os.Getenv(testHelperCaptureKey))
	if capturePath != "" {
		captureFile, err := os.Create(capturePath)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "create capture file: %v\n", err)
			os.Exit(1)
		}
		input = io.TeeReader(os.Stdin, captureFile)

		conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, input)
		agent.conn = conn
		<-conn.Done()
		if err := captureFile.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close capture file: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, input)
	agent.conn = conn
	<-conn.Done()
	os.Exit(0)
}

func TestACPWrapperProcess(_ *testing.T) {
	if os.Getenv(testWrapperEnvKey) != "1" {
		return
	}

	bin, err := os.Executable()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "resolve test binary: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.CommandContext(context.Background(), bin, "-test.run=TestACPHelperProcess")
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "start wrapped helper: %v\n", err)
		os.Exit(1)
	}

	if pidFile := strings.TrimSpace(os.Getenv(testWrapperPIDFileEnvKey)); pidFile != "" {
		if writeErr := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); writeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "write pid file: %v\n", writeErr)
			_ = cmd.Process.Kill()
			os.Exit(1)
		}
	}

	if err := cmd.Wait(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "wrapped helper exited: %v\n", err)
		os.Exit(1)
	}

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
		t.Fatalf(
			"resolvePath(%q) = %q, want suffix %q",
			insideFile,
			resolvedInside,
			filepath.Join("nested", "file.txt"),
		)
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
		t.Fatalf(
			"tokenUsageFromPromptResponse() cache_write_tokens = %#v, want %d",
			promptUsage.CacheWriteTokens,
			cacheWriteTokens,
		)
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

func TestDriverRejectsUninitializedProcessState(t *testing.T) {
	t.Parallel()

	driver := New()

	t.Run("prompt requires connection", func(t *testing.T) {
		t.Parallel()

		proc := &AgentProcess{SessionID: "session-1"}
		events, err := driver.Prompt(context.Background(), proc, PromptRequest{
			TurnID:  "turn-1",
			Message: "hello",
		})
		if err == nil {
			t.Fatalf("Prompt() error = nil, want %v", errProcessConnectionUninitialized)
		}
		if !errors.Is(err, errProcessConnectionUninitialized) {
			t.Fatalf("Prompt() error = %v, want %v", err, errProcessConnectionUninitialized)
		}
		if events != nil {
			t.Fatalf("Prompt() events = %v, want nil", events)
		}
	})

	t.Run("cancel requires connection and does not panic", func(t *testing.T) {
		t.Parallel()

		proc := &AgentProcess{SessionID: "session-1"}
		var (
			err    error
			panicV any
		)
		func() {
			defer func() {
				panicV = recover()
			}()
			err = driver.Cancel(context.Background(), proc)
		}()

		if panicV != nil {
			t.Fatalf("Cancel() panicked: %v", panicV)
		}
		if !errors.Is(err, errProcessConnectionUninitialized) {
			t.Fatalf("Cancel() error = %v, want %v", err, errProcessConnectionUninitialized)
		}
	})

	t.Run("stop requires lifecycle", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err := driver.Stop(ctx, &AgentProcess{})
		if err == nil {
			t.Fatalf("Stop() error = nil, want %v", errProcessLifecycleUninitialized)
		}
		if !errors.Is(err, errProcessLifecycleUninitialized) {
			t.Fatalf("Stop() error = %v, want %v", err, errProcessLifecycleUninitialized)
		}
	})
}

func TestPromptPrependsSystemPromptOnce(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "echo_prompt", "", StartOpts{
		SystemPrompt: "Memory context first.\nThen agent prompt.",
	})
	defer stopProcess(t, driver, proc)

	firstEventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

	secondEventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

func TestPromptActivityReporterReportsWhilePromptIsInFlight(t *testing.T) {
	t.Run("ShouldReportWhilePromptIsInFlight", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(testutil.Context(t))
		defer cancel()

		reports := make(chan PromptActivityReport, 4)
		stop := startPromptActivityReporter(ctx, PromptRequest{
			TurnID:                    "turn-reporter",
			Message:                   "hello",
			ActivityHeartbeatInterval: 5 * time.Millisecond,
			ActivityReporter: func(report PromptActivityReport) {
				select {
				case reports <- report:
				default:
				}
			},
		})
		defer stop()

		first := readPromptActivityReport(t, reports)
		if got, want := first.Kind, "agent_waiting"; got != want {
			t.Fatalf("first report kind = %q, want %q", got, want)
		}
		if first.Timestamp.IsZero() {
			t.Fatal("first report timestamp is zero")
		}

		second := readPromptActivityReport(t, reports)
		if got, want := second.Kind, "agent_waiting"; got != want {
			t.Fatalf("second report kind = %q, want %q", got, want)
		}
		if second.Timestamp.Before(first.Timestamp) {
			t.Fatalf("second report timestamp %s before first %s", second.Timestamp, first.Timestamp)
		}
	})
}

func readPromptActivityReport(t *testing.T, reports <-chan PromptActivityReport) PromptActivityReport {
	t.Helper()

	select {
	case report := <-reports:
		return report
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt activity report")
	}
	return PromptActivityReport{}
}

func TestPromptTransmitsStructuredMetadata(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "echo_prompt_meta", "", StartOpts{})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
		TurnID:  "turn-meta",
		Message: "network delivery",
		Meta: PromptMeta{
			TurnSource: PromptTurnSourceNetwork,
			Network: &PromptNetworkMeta{
				MessageID:   "msg-meta-1",
				Kind:        "say",
				Channel:     "builders",
				Surface:     "direct",
				DirectID:    "direct_meta_1",
				From:        "ops.peer",
				To:          "worker.peer",
				WorkID:      "work-meta-1",
				ReplyTo:     "msg-root-1",
				TraceID:     "trace-meta-1",
				CausationID: "msg-root-1",
				Trust:       "untrusted",
			},
		},
	})
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}

	events := collectEvents(t, eventsCh)
	if len(events) == 0 {
		t.Fatal("Prompt() returned no events")
	}

	var payload PromptMeta
	if err := json.Unmarshal([]byte(events[0].Text), &payload); err != nil {
		t.Fatalf("json.Unmarshal(prompt meta echo) error = %v", err)
	}
	if got, want := payload.TurnSource, PromptTurnSourceNetwork; got != want {
		t.Fatalf("payload.TurnSource = %q, want %q", got, want)
	}
	if payload.Network == nil {
		t.Fatal("payload.Network = nil, want populated network metadata")
	}
	if got, want := payload.Network.MessageID, "msg-meta-1"; got != want {
		t.Fatalf("payload.Network.MessageID = %q, want %q", got, want)
	}
	if got, want := payload.Network.Surface, "direct"; got != want {
		t.Fatalf("payload.Network.Surface = %q, want %q", got, want)
	}
	if got, want := payload.Network.DirectID, "direct_meta_1"; got != want {
		t.Fatalf("payload.Network.DirectID = %q, want %q", got, want)
	}
	if got, want := payload.Network.WorkID, "work-meta-1"; got != want {
		t.Fatalf("payload.Network.WorkID = %q, want %q", got, want)
	}
	if got, want := payload.Network.Trust, "untrusted"; got != want {
		t.Fatalf("payload.Network.Trust = %q, want %q", got, want)
	}
}

func TestDaemonMatchedEnvPinsCurrentBinary(t *testing.T) {
	t.Parallel()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(
		executable,
	); resolveErr == nil &&
		strings.TrimSpace(resolved) != "" {
		executable = resolved
	}
	binDir := filepath.Dir(executable)

	env := daemonMatchedEnv([]string{
		"PATH=/should-be-ignored",
		"FOO=bar",
		"AGH_BIN=/should-be-replaced",
		"PATH=/usr/local/bin" + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + "/usr/bin",
		"AGH_BIN=/should-also-be-replaced",
	})

	gotAGHBin, ok := envValue(env, "AGH_BIN")
	if !ok || gotAGHBin != executable {
		t.Fatalf("daemonMatchedEnv() AGH_BIN = %q, %v, want %q", gotAGHBin, ok, executable)
	}

	gotPath, ok := envValue(env, "PATH")
	if !ok {
		t.Fatal("daemonMatchedEnv() PATH missing")
	}
	wantPath := binDir + string(os.PathListSeparator) + "/usr/local/bin" + string(os.PathListSeparator) + "/usr/bin"
	if gotPath != wantPath {
		t.Fatalf("daemonMatchedEnv() PATH = %q, want %q", gotPath, wantPath)
	}

	pathCount := 0
	aghBinCount := 0
	for _, variable := range env {
		switch {
		case strings.HasPrefix(variable, "PATH="):
			pathCount++
		case strings.HasPrefix(variable, "AGH_BIN="):
			aghBinCount++
		}
	}
	if pathCount != 1 || aghBinCount != 1 {
		t.Fatalf("daemonMatchedEnv() duplicate entries remain: PATH=%d AGH_BIN=%d env=%#v", pathCount, aghBinCount, env)
	}
}

func TestPromptStreamsSessionUpdates(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{})
	defer stopProcess(t, driver, proc)

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

func TestStartApproveAllSetsPermissiveSessionModeWhenSupported(t *testing.T) {
	t.Parallel()

	driver := New()
	captureFile := filepath.Join(t.TempDir(), "session-set-mode-new.jsonl")
	proc := startHelperProcess(t, driver, "mode_mapping", "", StartOpts{
		Permissions: aghconfig.PermissionModeApproveAll,
		Env:         helperEnvWithCapture("mode_mapping", "", captureFile),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetMode)
	request := decodeCapturedSetSessionModeRequest(t, params)
	if got, want := request.SessionID, "sess-new"; got != want {
		t.Fatalf("set-mode session id = %q, want %q", got, want)
	}
	if got, want := request.ModeID, "bypassPermissions"; got != want {
		t.Fatalf("set-mode mode id = %q, want %q", got, want)
	}
}

func TestStartWithToolGatewayPrefersApprovalMediatedSessionMode(t *testing.T) {
	t.Parallel()

	driver := New()
	captureFile := filepath.Join(t.TempDir(), "session-set-mode-gateway.jsonl")
	proc := startHelperProcess(t, driver, "mode_mapping", "", StartOpts{
		Permissions: aghconfig.PermissionModeApproveAll,
		Env:         helperEnvWithCapture("mode_mapping", "", captureFile),
		ToolGateway: toolExecutionGatewayFunc(
			func(_ context.Context, req ToolExecutionRequest) (ToolExecutionRequest, error) {
				return req, nil
			},
		),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetMode)
	request := decodeCapturedSetSessionModeRequest(t, params)
	if got, want := request.SessionID, "sess-new"; got != want {
		t.Fatalf("set-mode session id = %q, want %q", got, want)
	}
	if got, want := request.ModeID, "default"; got != want {
		t.Fatalf("set-mode mode id = %q, want %q", got, want)
	}
}

func TestStartResumeApproveReadsSetsReadOnlyLikeSessionModeWhenSupported(t *testing.T) {
	t.Parallel()

	driver := New()
	captureFile := filepath.Join(t.TempDir(), "session-set-mode-load.jsonl")
	proc := startHelperProcess(t, driver, "load_mode_mapping", "", StartOpts{
		ResumeSessionID: "sess-existing",
		Permissions:     aghconfig.PermissionModeApproveReads,
		Env:             helperEnvWithCapture("load_mode_mapping", "", captureFile),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetMode)
	request := decodeCapturedSetSessionModeRequest(t, params)
	if got, want := request.SessionID, "sess-existing"; got != want {
		t.Fatalf("set-mode session id = %q, want %q", got, want)
	}
	if got, want := request.ModeID, "plan"; got != want {
		t.Fatalf("set-mode mode id = %q, want %q", got, want)
	}
}

func TestStartResumeWithToolGatewayPrefersApprovalMediatedMode(t *testing.T) {
	t.Parallel()

	driver := New()
	captureFile := filepath.Join(t.TempDir(), "session-set-mode-load-gateway.jsonl")
	proc := startHelperProcess(t, driver, "load_mode_mapping", "", StartOpts{
		ResumeSessionID: "sess-existing",
		Permissions:     aghconfig.PermissionModeApproveReads,
		Env:             helperEnvWithCapture("load_mode_mapping", "", captureFile),
		ToolGateway: toolExecutionGatewayFunc(
			func(_ context.Context, req ToolExecutionRequest) (ToolExecutionRequest, error) {
				return req, nil
			},
		),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetMode)
	request := decodeCapturedSetSessionModeRequest(t, params)
	if got, want := request.SessionID, "sess-existing"; got != want {
		t.Fatalf("set-mode session id = %q, want %q", got, want)
	}
	if got, want := request.ModeID, "default"; got != want {
		t.Fatalf("set-mode mode id = %q, want %q", got, want)
	}
}

func TestStartDenyAllWithToolGatewayPrefersApprovalMediatedSessionMode(t *testing.T) {
	t.Parallel()

	driver := New()
	captureFile := filepath.Join(t.TempDir(), "session-set-mode-deny-gateway.jsonl")
	proc := startHelperProcess(t, driver, "mode_mapping", "", StartOpts{
		Permissions: aghconfig.PermissionModeDenyAll,
		Env:         helperEnvWithCapture("mode_mapping", "", captureFile),
		ToolGateway: toolExecutionGatewayFunc(
			func(_ context.Context, req ToolExecutionRequest) (ToolExecutionRequest, error) {
				return req, nil
			},
		),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetMode)
	request := decodeCapturedSetSessionModeRequest(t, params)
	if got, want := request.SessionID, "sess-new"; got != want {
		t.Fatalf("set-mode session id = %q, want %q", got, want)
	}
	if got, want := request.ModeID, "default"; got != want {
		t.Fatalf("set-mode mode id = %q, want %q", got, want)
	}
}

func TestStartSetsPreferredSessionModelWhenProvided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		scenario      string
		resumeSession string
		preferred     string
		wantSession   string
	}{
		{
			name:        "Should set preferred model for new sessions",
			scenario:    "stream_updates",
			preferred:   "openrouter/openai/gpt-5.4",
			wantSession: "sess-new",
		},
		{
			name:          "Should set preferred model for resumed sessions",
			scenario:      "load_session",
			resumeSession: "sess-existing",
			preferred:     "anthropic/claude-opus-4-7",
			wantSession:   "sess-existing",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			driver := New()
			captureFile := filepath.Join(t.TempDir(), "session-set-model.jsonl")
			proc := startHelperProcess(t, driver, tc.scenario, "", StartOpts{
				ResumeSessionID: tc.resumeSession,
				PreferredModel:  tc.preferred,
				Env:             helperEnvWithCapture(tc.scenario, "", captureFile),
			})
			defer stopProcess(t, driver, proc)

			params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionSetModel)
			request := decodeCapturedSetSessionModelRequest(t, params)
			if got := request.SessionID; got != tc.wantSession {
				t.Fatalf("set-model session id = %q, want %q", got, tc.wantSession)
			}
			if got := request.ModelID; got != tc.preferred {
				t.Fatalf("set-model model id = %q, want %q", got, tc.preferred)
			}
		})
	}
}

func TestStartWithEmptyAdditionalDirsKeepsBaselinePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts StartOpts
	}{
		{
			name: "nil additional dirs",
			opts: StartOpts{},
		},
		{
			name: "explicit empty additional dirs",
			opts: StartOpts{AdditionalDirs: []string{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			driver := New()
			captureFile := filepath.Join(t.TempDir(), strings.ReplaceAll(tt.name, " ", "-")+".jsonl")
			opts := tt.opts
			opts.Env = helperEnvWithCapture("stream_updates", "", captureFile)

			proc := startHelperProcess(t, driver, "stream_updates", "", opts)
			defer stopProcess(t, driver, proc)

			params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionNew)
			if _, exists := params["additional_dirs"]; exists {
				t.Fatalf("session/new params include additional_dirs for %s: %#v", tt.name, params)
			}
		})
	}
}

func TestStartIncludesAdditionalDirsInNewSessionPayload(t *testing.T) {
	t.Parallel()

	driver := New()
	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()
	captureFile := filepath.Join(t.TempDir(), "session-new.jsonl")

	proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{
		Cwd:            root,
		AdditionalDirs: []string{additionalOne, additionalTwo},
		Env:            helperEnvWithCapture("stream_updates", "", captureFile),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionNew)
	request := decodeCapturedNewSessionRequest(t, params)
	if got, want := request.Cwd, mustCanonicalDir(t, root); got != want {
		t.Fatalf("session/new cwd = %q, want %q", got, want)
	}
	if got, want := request.AdditionalDirs, []string{
		mustCanonicalDir(t, additionalOne),
		mustCanonicalDir(t, additionalTwo),
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("session/new additional_dirs = %#v, want %#v", got, want)
	}
}

func TestStartIncludesAdditionalDirsInLoadSessionPayload(t *testing.T) {
	t.Parallel()

	driver := New()
	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()
	captureFile := filepath.Join(t.TempDir(), "session-load.jsonl")

	proc := startHelperProcess(t, driver, "load_session", "", StartOpts{
		Cwd:             root,
		AdditionalDirs:  []string{additionalOne, additionalTwo},
		ResumeSessionID: "sess-existing",
		Env:             helperEnvWithCapture("load_session", "", captureFile),
	})
	defer stopProcess(t, driver, proc)

	params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionLoad)
	request := decodeCapturedLoadSessionRequest(t, params)
	if got, want := request.Cwd, mustCanonicalDir(t, root); got != want {
		t.Fatalf("session/load cwd = %q, want %q", got, want)
	}
	if request.SessionID != "sess-existing" {
		t.Fatalf("session/load sessionId = %q, want %q", request.SessionID, "sess-existing")
	}
	if got, want := request.AdditionalDirs, []string{
		mustCanonicalDir(t, additionalOne),
		mustCanonicalDir(t, additionalTwo),
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("session/load additional_dirs = %#v, want %#v", got, want)
	}
}

func TestStartMCPServersSkipsRemoteTransports(t *testing.T) {
	t.Parallel()

	t.Run("Should skip remote transports when starting MCP servers", func(t *testing.T) {
		t.Parallel()

		driver := New()
		captureFile := filepath.Join(t.TempDir(), "session-new-mcp.jsonl")
		proc := startHelperProcess(t, driver, "stream_updates", "", StartOpts{
			Cwd: t.TempDir(),
			Env: helperEnvWithCapture("stream_updates", "", captureFile),
			MCPServers: []aghconfig.MCPServer{
				{
					Name:      "agh-hosted-tools",
					Transport: aghconfig.MCPServerTransportStdio,
					Command:   "/bin/agh",
					Args:      []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"},
					Env:       map[string]string{"AGH_HOME": "/tmp/agh-home"},
				},
				{
					Name:      "remote-http",
					Transport: aghconfig.MCPServerTransportHTTP,
					URL:       "https://example.test/mcp",
				},
				{
					Name:      "remote-sse",
					Transport: aghconfig.MCPServerTransportSSE,
					URL:       "https://example.test/sse",
				},
			},
		})
		defer stopProcess(t, driver, proc)

		params := captureRequestParams(t, captureFile, acpsdk.AgentMethodSessionNew)
		request := decodeCapturedNewSessionRequest(t, params)
		if got, want := len(request.MCPServers), 1; got != want {
			t.Fatalf("session/new mcpServers = %#v, want only hosted stdio entry", request.MCPServers)
		}
		stdio := request.MCPServers[0].Stdio
		if stdio == nil {
			t.Fatalf("session/new mcpServers[0] = %#v, want stdio variant", request.MCPServers[0])
		}
		if stdio.Name != "agh-hosted-tools" || stdio.Command != "/bin/agh" {
			t.Fatalf("hosted stdio entry = %#v, want hosted command", stdio)
		}
		if !slices.Equal(stdio.Args, []string{"tool", "mcp", "--session", "sess-1", "--bind-nonce", "nonce"}) {
			t.Fatalf("hosted stdio args = %#v, want tool mcp bind args", stdio.Args)
		}
		if got, want := len(stdio.Env), 1; got != want || stdio.Env[0].Name != "AGH_HOME" {
			t.Fatalf("hosted stdio env = %#v, want AGH_HOME only", stdio.Env)
		}
	})
}

func TestStartResumeReturnsSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		envScenario string
		wantErr     error
	}{
		"load session failure": {
			envScenario: "load_session_error",
			wantErr:     ErrLoadSessionFailed,
		},
		"agent missing load session support": {
			envScenario: "stream_updates",
			wantErr:     ErrAgentDoesNotSupportSession,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			driver := New()
			_, err := driver.Start(testutil.Context(t), StartOpts{
				AgentName:       "helper",
				Command:         helperCommand(t),
				Cwd:             t.TempDir(),
				Env:             helperEnv(tc.envScenario, ""),
				Permissions:     aghconfig.PermissionModeApproveAll,
				ResumeSessionID: "sess-existing",
			})
			if err == nil {
				t.Fatalf("Start(%s) error = nil, want non-nil", tc.envScenario)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Start(%s) error = %v, want errors.Is(..., %v)", tc.envScenario, err, tc.wantErr)
			}
		})
	}
}

func TestStartIncludesAgentContextInLaunchErrors(t *testing.T) {
	t.Parallel()

	driver := New()
	_, err := driver.Start(testutil.Context(t), StartOpts{
		AgentName:   "missing-helper",
		Command:     "/definitely/missing-binary",
		Cwd:         t.TempDir(),
		Permissions: aghconfig.PermissionModeApproveAll,
	})
	if err == nil {
		t.Fatal("Start() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), `start agent "missing-helper" subprocess "/definitely/missing-binary"`) {
		t.Fatalf("Start() error = %q, want agent and command context", err)
	}
}

func TestIsLoadSessionResourceMissing(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want bool
	}{
		"ShouldDetectResourceMissingRequestError": {
			err: fmt.Errorf(
				"%w: load session %q for %q: %w",
				ErrLoadSessionFailed,
				"sess-existing",
				"helper",
				&acpsdk.RequestError{
					Code:    requestErrorResourceNotFoundCode,
					Message: "Resource not found: sess-existing",
				},
			),
			want: true,
		},
		"ShouldRejectDifferentRequestError": {
			err: fmt.Errorf(
				"%w: load session %q for %q: %w",
				ErrLoadSessionFailed,
				"sess-existing",
				"helper",
				&acpsdk.RequestError{Code: -32603, Message: "Internal error"},
			),
			want: false,
		},
		"ShouldRejectNonLoadSessionError": {
			err:  errors.New("boom"),
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IsLoadSessionResourceMissing(tc.err); got != tc.want {
				t.Fatalf("IsLoadSessionResourceMissing() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCleanupFailedStartReturnsJoinedErrorWhenStopFails(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := &AgentProcess{
		done:   make(chan struct{}),
		stderr: &lockedBuffer{},
	}
	stopErr := errors.New("stop failed")
	proc.setWaitError(stopErr)
	close(proc.done)

	startErr := fmt.Errorf(
		"%w: load session %q for %q: %w",
		ErrLoadSessionFailed,
		"sess-existing",
		"helper",
		errors.New("load failed"),
	)
	err := driver.cleanupFailedStart(proc, startErr)
	if err == nil {
		t.Fatal("cleanupFailedStart() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrLoadSessionFailed) {
		t.Fatalf("cleanupFailedStart() error = %v, want ErrLoadSessionFailed", err)
	}
	if !errors.Is(err, stopErr) {
		t.Fatalf("cleanupFailedStart() error = %v, want stopErr", err)
	}
	if !strings.Contains(err.Error(), "stop failed while cleaning up failed start") {
		t.Fatalf("cleanupFailedStart() error = %v, want cleanup stop context", err)
	}
}

func TestProcessCrashDetected(t *testing.T) {
	t.Parallel()

	driver := New()
	proc := startHelperProcess(t, driver, "crash_on_prompt", "", StartOpts{})

	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
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

func TestPromptStopDoesNotEmitRuntimeError(t *testing.T) {
	t.Parallel()

	t.Run("Should not emit runtime error after explicit stop", func(t *testing.T) {
		driver := New()
		proc := startHelperProcess(t, driver, "block_prompt_until_cancel", "", StartOpts{})
		t.Cleanup(func() {
			stopProcess(t, driver, proc)
		})

		eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
			TurnID:  "turn-stop",
			Message: "block until stopped",
		})
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}

		select {
		case event := <-eventsCh:
			if got, want := event.Type, EventTypeAgentMessage; got != want {
				t.Fatalf("first prompt event = %q, want %q", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for blocking prompt to start")
		}

		if err := driver.Stop(testutil.Context(t), proc); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		for _, event := range collectEvents(t, eventsCh) {
			if event.Type == EventTypeError {
				t.Fatalf("prompt events contain %q after explicit stop: %#v", EventTypeError, event)
			}
		}
	})
}

func TestShouldSuppressPromptErrorOnStop(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Should suppress context canceled errors",
			err:  context.Canceled,
			want: true,
		},
		{
			name: "Should suppress deadline exceeded errors",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "Should suppress wrapped canceled failures",
			err:  WrapFailure(store.FailureCanceled, "stopped", context.Canceled),
			want: true,
		},
		{
			name: "Should suppress request errors carrying canceled details",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error",
				Data:    map[string]any{"error": "context canceled"},
			},
			want: true,
		},
		{
			name: "Should not suppress generic request failures",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error",
				Data:    map[string]any{"details": "Tool invocation failed"},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldSuppressPromptErrorOnStop(tc.err); got != tc.want {
				t.Fatalf("shouldSuppressPromptErrorOnStop() = %v, want %v", got, tc.want)
			}
		})
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

func startHelperProcess(
	t *testing.T,
	driver *Driver,
	scenario string,
	filePath string,
	overrides StartOpts,
) *AgentProcess {
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
	if overrides.AdditionalDirs != nil {
		opts.AdditionalDirs = append([]string(nil), overrides.AdditionalDirs...)
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
	if overrides.PreferredModel != "" {
		opts.PreferredModel = overrides.PreferredModel
	}
	opts.ResumeSessionID = overrides.ResumeSessionID
	opts.Launcher = overrides.Launcher
	opts.ToolHost = overrides.ToolHost
	opts.ToolGateway = overrides.ToolGateway

	proc, err := driver.Start(testutil.Context(t), opts)
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
	if err := driver.Stop(testutil.Context(t), proc); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestStopManagedProcessRespectsContext(t *testing.T) {
	t.Run("ShouldReturnDeadlineExceededWhenManagedProcessShutdownExceedsStopContext", func(t *testing.T) {
		t.Parallel()

		driver := New(WithStopTimeout(5 * time.Second))
		managed, err := subprocess.Launch(context.Background(), subprocess.LaunchConfig{
			Command:          "sh",
			Args:             []string{"-c", "sleep 30"},
			DisableTransport: true,
			ShutdownTimeout:  time.Second,
		})
		if err != nil {
			t.Fatalf("Launch() error = %v", err)
		}

		proc := &AgentProcess{
			managed: managed,
			done:    make(chan struct{}),
		}
		go proc.waitForExit(context.Background(), defaultProcessRecordTimeout)
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := managed.Shutdown(cleanupCtx); shutdownErr != nil {
				t.Fatalf("managed.Shutdown() error = %v", shutdownErr)
			}
			select {
			case <-proc.Done():
			case <-cleanupCtx.Done():
				t.Fatalf("process did not exit during cleanup: %v", cleanupCtx.Err())
			}
		})

		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		startedAt := time.Now()
		err = driver.Stop(stopCtx, proc)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Stop() error = %v, want context deadline exceeded", err)
		}
		if elapsed := time.Since(startedAt); elapsed > time.Second {
			t.Fatalf("Stop() elapsed = %v, want <= 1s", elapsed)
		}
	})
}

func TestRegisterAgentProcessRetainsRegistryForPIDLessSandboxAgents(t *testing.T) {
	t.Run("Should keep registry available for external sandbox terminal tracking", func(t *testing.T) {
		t.Parallel()

		registry := toolruntime.NewRegistry(nil)
		driver := &Driver{processRegistry: registry}
		process := &AgentProcess{PID: 0}

		if err := driver.registerAgentProcess(context.Background(), process); err != nil {
			t.Fatalf("registerAgentProcess(PID=0) error = %v", err)
		}
		if process.processRegistry != registry {
			t.Fatalf("process.processRegistry = %p, want %p", process.processRegistry, registry)
		}
		if process.processRecord != nil {
			t.Fatalf("process.processRecord = %#v, want nil for PID-less agent", process.processRecord)
		}
	})
}

func TestProcessRecordContext(t *testing.T) {
	t.Run("Should detach cancellation while preserving a bounded deadline", func(t *testing.T) {
		t.Parallel()

		parent, cancelParent := context.WithCancel(context.Background())
		cancelParent()

		ctx, cancel := processRecordContext(parent, 25*time.Millisecond)
		defer cancel()
		if err := ctx.Err(); err != nil {
			t.Fatalf("processRecordContext() err = %v, want detached from parent cancellation", err)
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("processRecordContext() deadline missing")
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
			t.Fatalf("processRecordContext() remaining deadline = %s, want bounded positive deadline", remaining)
		}
	})
}

func TestCheckpointProcessOwnerWrapsCheckpointErrors(t *testing.T) {
	t.Run("Should add ACP context while preserving checkpoint root error", func(t *testing.T) {
		t.Parallel()

		root := errors.New("checkpoint failed")
		registry := toolruntime.NewRegistry(&failingToolRuntimeStore{updateErr: root})
		handle, err := registry.Register(context.Background(), toolruntime.RegisterConfig{
			Source:  toolruntime.ProcessSourceACPAgent,
			Owner:   toolruntime.ProcessOwner{SessionID: "old-session"},
			Command: "agent",
		})
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		process := &AgentProcess{
			SessionID:     "new-session",
			processRecord: handle,
		}

		err = process.checkpointProcessOwner(context.Background())
		if !errors.Is(err, root) || !strings.Contains(err.Error(), "checkpoint process owner") {
			t.Fatalf("checkpointProcessOwner() error = %v, want ACP context wrapping root", err)
		}
	})
}

type failingToolRuntimeStore struct {
	updateErr error
	upserts   int
}

func (s *failingToolRuntimeStore) UpsertProcessRecord(context.Context, toolruntime.ProcessRecord) error {
	s.upserts++
	if s.upserts > 1 {
		return s.updateErr
	}
	return nil
}

func (s *failingToolRuntimeStore) UpdateProcessRecordState(
	context.Context,
	toolruntime.ProcessStateUpdate,
) error {
	return s.updateErr
}

func (s *failingToolRuntimeStore) ListProcessRecords(
	context.Context,
	toolruntime.ProcessQuery,
) ([]toolruntime.ProcessRecord, error) {
	return nil, nil
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

func helperEnvWithCapture(scenario string, filePath string, capturePath string) []string {
	env := helperEnv(scenario, filePath)
	if strings.TrimSpace(capturePath) != "" {
		env = append(env, testHelperCaptureKey+"="+capturePath)
	}
	return env
}

type capturedRequestEnvelope struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type capturedNewSessionRequest struct {
	Cwd            string             `json:"cwd"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
	MCPServers     []acpsdk.McpServer `json:"mcpServers"`
}

type capturedLoadSessionRequest struct {
	Cwd            string             `json:"cwd"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
	MCPServers     []acpsdk.McpServer `json:"mcpServers"`
	SessionID      string             `json:"sessionId"`
}

type capturedSetSessionModeRequest struct {
	SessionID string `json:"sessionId"`
	ModeID    string `json:"modeId"`
}

type capturedSetSessionModelRequest struct {
	SessionID string `json:"sessionId"`
	ModelID   string `json:"modelId"`
}

func captureRequestParams(t *testing.T, path string, method string) map[string]json.RawMessage {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	lines := strings.SplitSeq(strings.TrimSpace(string(data)), "\n")
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var envelope capturedRequestEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			t.Fatalf("json.Unmarshal(captured envelope) error = %v", err)
		}
		if envelope.Method != method {
			continue
		}

		var params map[string]json.RawMessage
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			t.Fatalf("json.Unmarshal(captured params) error = %v", err)
		}
		return params
	}

	t.Fatalf("capture file %q does not contain method %q", path, method)
	return nil
}

func decodeCapturedNewSessionRequest(t *testing.T, params map[string]json.RawMessage) capturedNewSessionRequest {
	t.Helper()

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(new-session params) error = %v", err)
	}
	var request capturedNewSessionRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("json.Unmarshal(new-session request) error = %v", err)
	}
	return request
}

func decodeCapturedLoadSessionRequest(t *testing.T, params map[string]json.RawMessage) capturedLoadSessionRequest {
	t.Helper()

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(load-session params) error = %v", err)
	}
	var request capturedLoadSessionRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("json.Unmarshal(load-session request) error = %v", err)
	}
	return request
}

func decodeCapturedSetSessionModeRequest(
	t *testing.T,
	params map[string]json.RawMessage,
) capturedSetSessionModeRequest {
	t.Helper()

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(set-session-mode params) error = %v", err)
	}
	var request capturedSetSessionModeRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("json.Unmarshal(set-session-mode request) error = %v", err)
	}
	return request
}

func decodeCapturedSetSessionModelRequest(
	t *testing.T,
	params map[string]json.RawMessage,
) capturedSetSessionModelRequest {
	t.Helper()

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(set-session-model params) error = %v", err)
	}
	var request capturedSetSessionModelRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("json.Unmarshal(set-session-model request) error = %v", err)
	}
	return request
}

func mustCanonicalDir(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q) error = %v", path, err)
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", resolved, err)
	}
	return filepath.Clean(absolute)
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

type helperACPAgent struct {
	conn     *acpsdk.AgentSideConnection
	scenario string
	filePath string
}

func (a *helperACPAgent) Authenticate(
	context.Context,
	acpsdk.AuthenticateRequest,
) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *helperACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: a.scenario == "load_session" || a.scenario == "load_session_error" ||
				a.scenario == "load_mode_mapping",
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (a *helperACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *helperACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	if a.scenario == "mode_mapping" {
		return acpsdk.NewSessionResponse{
			SessionId: "sess-new",
			Modes:     helperModeStateWithCurrent("default", "default", "plan", "bypassPermissions"),
			Models:    helperModelState("new-model"),
		}, nil
	}
	return acpsdk.NewSessionResponse{
		SessionId: "sess-new",
		Modes:     helperModeState("new-mode"),
		Models:    helperModelState("new-model"),
	}, nil
}

func (a *helperACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	if a.scenario == "load_session_error" {
		return acpsdk.LoadSessionResponse{}, errors.New("load failed")
	}
	if a.scenario == "load_mode_mapping" {
		return acpsdk.LoadSessionResponse{
			Modes:  helperModeStateWithCurrent("default", "default", "plan", "bypassPermissions"),
			Models: helperModelState("loaded-model"),
		}, nil
	}
	return acpsdk.LoadSessionResponse{
		Modes:  helperModeState("loaded-mode"),
		Models: helperModelState("loaded-model"),
	}, nil
}

func (a *helperACPAgent) Prompt(ctx context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	switch a.scenario {
	case "crash_on_prompt":
		os.Exit(23)
	case "block_prompt_until_cancel":
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText("blocking"),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
		<-ctx.Done()
		return acpsdk.PromptResponse{}, ctx.Err()
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
	case "echo_prompt_meta":
		data, err := json.Marshal(params.Meta)
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(string(data)),
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
	case "fs_write_terminal":
		if _, err := a.conn.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			SessionId: params.SessionId,
			Path:      a.filePath,
			Content:   "from-write",
		}); err != nil {
			return acpsdk.PromptResponse{}, err
		}
		readResponse, err := a.conn.ReadTextFile(ctx, acpsdk.ReadTextFileRequest{
			SessionId: params.SessionId,
			Path:      a.filePath,
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(readResponse.Content),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}

		cwd, err := os.Getwd()
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		createResp, err := a.conn.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			SessionId: params.SessionId,
			Command:   "sh",
			Args:      []string{"-c", "printf terminal-ok"},
			Cwd:       acpsdk.Ptr(cwd),
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if _, err := a.conn.WaitForTerminalExit(ctx, acpsdk.WaitForTerminalExitRequest{
			SessionId:  params.SessionId,
			TerminalId: createResp.TerminalId,
		}); err != nil {
			return acpsdk.PromptResponse{}, err
		}
		outputResp, err := a.conn.TerminalOutput(ctx, acpsdk.TerminalOutputRequest{
			SessionId:  params.SessionId,
			TerminalId: createResp.TerminalId,
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(outputResp.Output),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	case "permission":
		title := "permission request"
		locationPath := a.filePath
		if locationPath == "" {
			locationPath = filepath.Join(string(filepath.Separator), "workspace", "demo.txt")
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
		selected := "canceled"
		if outcome.Outcome.Selected != nil {
			selected = string(outcome.Outcome.Selected.OptionId)
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(selected),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	case "network_guardrails":
		targetPath := a.filePath
		if targetPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return acpsdk.PromptResponse{}, err
			}
			targetPath = filepath.Join(cwd, "network-blocked.txt")
		}

		writeResult := "write_unexpected"
		if _, err := a.conn.WriteTextFile(ctx, acpsdk.WriteTextFileRequest{
			SessionId: params.SessionId,
			Path:      targetPath,
			Content:   "blocked",
		}); err != nil {
			writeResult = "write_blocked"
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(writeResult),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}

		shellResult := "shell_unexpected"
		if _, err := a.conn.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			SessionId: params.SessionId,
			Command:   "sh",
			Args:      []string{"-c", "printf nope"},
		}); err != nil {
			shellResult = "shell_blocked"
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(shellResult),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}

		cwd, err := os.Getwd()
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		createResp, err := a.conn.CreateTerminal(ctx, acpsdk.CreateTerminalRequest{
			SessionId: params.SessionId,
			Command:   "agh",
			Args:      []string{"network", "status"},
			Cwd:       acpsdk.Ptr(cwd),
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if _, err := a.conn.WaitForTerminalExit(ctx, acpsdk.WaitForTerminalExitRequest{
			SessionId:  params.SessionId,
			TerminalId: createResp.TerminalId,
		}); err != nil {
			return acpsdk.PromptResponse{}, err
		}
		outputResp, err := a.conn.TerminalOutput(ctx, acpsdk.TerminalOutputRequest{
			SessionId:  params.SessionId,
			TerminalId: createResp.TerminalId,
		})
		if err != nil {
			return acpsdk.PromptResponse{}, err
		}
		if sendErr := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: params.SessionId,
			Update:    acpsdk.UpdateAgentMessageText(outputResp.Output),
		}); sendErr != nil {
			return acpsdk.PromptResponse{}, sendErr
		}
	default:
		updates := []acpsdk.SessionUpdate{
			acpsdk.UpdateAgentMessageText("hello"),
			acpsdk.UpdateAgentThoughtText("thinking"),
			acpsdk.StartToolCall(
				"tool-1",
				"Read file",
				acpsdk.WithStartKind(acpsdk.ToolKindRead),
				acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress),
			),
			acpsdk.UpdateToolCall(
				"tool-1",
				acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted),
				acpsdk.WithUpdateTitle("Read file"),
			),
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

func (a *helperACPAgent) SetSessionMode(
	context.Context,
	acpsdk.SetSessionModeRequest,
) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func (a *helperACPAgent) SetSessionModel(
	context.Context,
	acpsdk.SetSessionModelRequest,
) (acpsdk.SetSessionModelResponse, error) {
	return acpsdk.SetSessionModelResponse{}, nil
}

func helperModeState(id string) *acpsdk.SessionModeState {
	return helperModeStateWithCurrent(id, id)
}

func helperModeStateWithCurrent(current string, available ...string) *acpsdk.SessionModeState {
	modes := make([]acpsdk.SessionMode, 0, len(available))
	for _, id := range available {
		modes = append(modes, acpsdk.SessionMode{
			Id:   acpsdk.SessionModeId(id),
			Name: id,
		})
	}
	return &acpsdk.SessionModeState{
		CurrentModeId:  acpsdk.SessionModeId(current),
		AvailableModes: modes,
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
