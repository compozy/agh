package acp

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestDriverOptionsAndNormalization(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	driver := New(
		WithLogger(logger),
		WithStopTimeout(2*time.Second),
		WithPromptBufferSize(9),
		WithPromptDrainWait(15*time.Millisecond),
	)
	if driver.logger != logger {
		t.Fatal("WithLogger() did not apply")
	}
	if driver.stopTimeout != 2*time.Second {
		t.Fatalf("WithStopTimeout() = %v, want %v", driver.stopTimeout, 2*time.Second)
	}
	if driver.promptBufferCap != 9 {
		t.Fatalf("WithPromptBufferSize() = %d, want 9", driver.promptBufferCap)
	}
	if driver.promptDrainWait != 15*time.Millisecond {
		t.Fatalf("WithPromptDrainWait() = %v, want %v", driver.promptDrainWait, 15*time.Millisecond)
	}

	root := t.TempDir()
	normalized, err := normalizeStartOpts(StartOpts{
		AgentName:   "helper",
		Command:     "sh -c 'echo ok'",
		Cwd:         root,
		Permissions: "",
	})
	if err != nil {
		t.Fatalf("normalizeStartOpts() error = %v", err)
	}
	if normalized.Permissions != aghconfig.PermissionModeApproveReads {
		t.Fatalf("normalizeStartOpts() permissions = %q, want %q", normalized.Permissions, aghconfig.PermissionModeApproveReads)
	}
}

func TestHandleInboundReadWriteFile(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeApproveAll)
	target := filepath.Join(proc.Cwd, "notes.txt")

	if _, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodFsWriteTextFile, mustMarshalJSON(acpsdk.WriteTextFileRequest{
		SessionId: "sess-direct",
		Path:      target,
		Content:   "line1\nline2\nline3",
	})); reqErr != nil {
		t.Fatalf("handleInbound(write) error = %v", reqErr)
	}

	response, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodFsReadTextFile, mustMarshalJSON(acpsdk.ReadTextFileRequest{
		SessionId: "sess-direct",
		Path:      target,
		Line:      acpsdk.Ptr(2),
		Limit:     acpsdk.Ptr(1),
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(read) error = %v", reqErr)
	}
	readResponse, ok := response.(acpsdk.ReadTextFileResponse)
	if !ok {
		t.Fatalf("handleInbound(read) type = %T, want ReadTextFileResponse", response)
	}
	if readResponse.Content != "line2" {
		t.Fatalf("handleInbound(read) content = %q, want %q", readResponse.Content, "line2")
	}
}

func TestHandleInboundWriteDenied(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeApproveReads)
	target := filepath.Join(proc.Cwd, "notes.txt")

	if _, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodFsWriteTextFile, mustMarshalJSON(acpsdk.WriteTextFileRequest{
		SessionId: "sess-direct",
		Path:      target,
		Content:   "nope",
	})); reqErr == nil {
		t.Fatal("handleInbound(write denied) error = nil, want non-nil")
	}
}

func TestHandleInboundPermissionRequest(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	active, err := proc.beginPrompt("turn-permission", 8)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}
	defer proc.endPrompt(active)

	title := "permission request"
	path := filepath.Join(proc.Cwd, "secret.txt")
	response, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodSessionRequestPermission, mustMarshalJSON(acpsdk.RequestPermissionRequest{
		SessionId: "sess-direct",
		Options: []acpsdk.PermissionOption{
			{OptionId: "allow", Name: "allow", Kind: acpsdk.PermissionOptionKindAllowOnce},
			{OptionId: "reject", Name: "reject", Kind: acpsdk.PermissionOptionKindRejectOnce},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool-1",
			Title:      &title,
			Locations:  []acpsdk.ToolCallLocation{{Path: path}},
		},
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(permission) error = %v", reqErr)
	}

	permissionResponse, ok := response.(acpsdk.RequestPermissionResponse)
	if !ok {
		t.Fatalf("handleInbound(permission) type = %T, want RequestPermissionResponse", response)
	}
	if permissionResponse.Outcome.Selected == nil || permissionResponse.Outcome.Selected.OptionId != "reject" {
		t.Fatalf("permission outcome = %#v, want reject option", permissionResponse.Outcome)
	}

	events := collectEventsUntilCount(t, active.events, 1)
	if len(events) != 1 || events[0].Type != EventTypePermission || events[0].Decision != "deny" {
		t.Fatalf("permission events = %#v, want denied permission event", events)
	}
}

func TestTerminalLifecycleHandlers(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeApproveAll)

	createResult, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodTerminalCreate, mustMarshalJSON(acpsdk.CreateTerminalRequest{
		SessionId: "sess-direct",
		Command:   "sh",
		Args:      []string{"-c", "printf hi"},
		Cwd:       acpsdk.Ptr(proc.Cwd),
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(create terminal) error = %v", reqErr)
	}
	createResponse, ok := createResult.(acpsdk.CreateTerminalResponse)
	if !ok {
		t.Fatalf("handleInbound(create terminal) type = %T, want CreateTerminalResponse", createResult)
	}

	waitResult, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodTerminalWaitForExit, mustMarshalJSON(acpsdk.WaitForTerminalExitRequest{
		SessionId:  "sess-direct",
		TerminalId: createResponse.TerminalId,
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(wait terminal) error = %v", reqErr)
	}
	waitResponse, ok := waitResult.(acpsdk.WaitForTerminalExitResponse)
	if !ok {
		t.Fatalf("handleInbound(wait terminal) type = %T, want WaitForTerminalExitResponse", waitResult)
	}
	if waitResponse.ExitCode == nil || *waitResponse.ExitCode != 0 {
		t.Fatalf("handleInbound(wait terminal) exit code = %#v, want 0", waitResponse.ExitCode)
	}

	outputResult, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodTerminalOutput, mustMarshalJSON(acpsdk.TerminalOutputRequest{
		SessionId:  "sess-direct",
		TerminalId: createResponse.TerminalId,
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(output terminal) error = %v", reqErr)
	}
	outputResponse, ok := outputResult.(acpsdk.TerminalOutputResponse)
	if !ok {
		t.Fatalf("handleInbound(output terminal) type = %T, want TerminalOutputResponse", outputResult)
	}
	if outputResponse.Output != "hi" {
		t.Fatalf("handleInbound(output terminal) output = %q, want %q", outputResponse.Output, "hi")
	}

	if _, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodTerminalKill, mustMarshalJSON(acpsdk.KillTerminalCommandRequest{
		SessionId:  "sess-direct",
		TerminalId: createResponse.TerminalId,
	})); reqErr != nil {
		t.Fatalf("handleInbound(kill terminal) error = %v", reqErr)
	}

	if _, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodTerminalRelease, mustMarshalJSON(acpsdk.ReleaseTerminalRequest{
		SessionId:  "sess-direct",
		TerminalId: createResponse.TerminalId,
	})); reqErr != nil {
		t.Fatalf("handleInbound(release terminal) error = %v", reqErr)
	}

	if _, _, _, err := proc.terminals.output(createResponse.TerminalId); err == nil {
		t.Fatal("output(released terminal) error = nil, want terminal not found")
	}
}

func TestHelperUtilities(t *testing.T) {
	t.Parallel()

	if got := attachStderr(errors.New("boom"), " stderr "); got == nil || got.Error() == "boom" {
		t.Fatalf("attachStderr() = %v, want wrapped stderr", got)
	}
	if got := attachStderr(errors.New("boom"), ""); got == nil || got.Error() != "boom" {
		t.Fatalf("attachStderr(empty) = %v, want original error", got)
	}

	env := mergeCommandEnv([]string{"A=1", "B=2"}, []acpsdk.EnvVariable{{Name: "B", Value: "3"}, {Name: "C", Value: "4"}})
	if len(env) != 3 || env[1] != "B=3" || env[2] != "C=4" {
		t.Fatalf("mergeCommandEnv() = %#v, want overridden env", env)
	}

	servers := toSDKMCPServers([]aghconfig.MCPServer{{
		Name:    "github",
		Command: "npx",
		Args:    []string{"-y", "mcp-server"},
		Env: map[string]string{
			"Z_VAR": "z",
			"A_VAR": "a",
		},
	}})
	if len(servers) != 1 || servers[0].Stdio == nil || len(servers[0].Stdio.Env) != 2 {
		t.Fatalf("toSDKMCPServers() = %#v, want one stdio server with env", servers)
	}
	if servers[0].Stdio.Env[0].Name != "A_VAR" {
		t.Fatalf("toSDKMCPServers() env order = %#v, want sorted env keys", servers[0].Stdio.Env)
	}

	if got := extractContentText(acpsdk.ResourceLinkBlock("doc", "file:///tmp/demo.txt")); got != "file:///tmp/demo.txt" {
		t.Fatalf("extractContentText(resource_link) = %q", got)
	}

	trimmed := trimUTF8LeadingBytes([]byte{0xff, 'h', 'i'})
	if string(trimmed) != "hi" {
		t.Fatalf("trimUTF8LeadingBytes() = %q, want %q", string(trimmed), "hi")
	}

	if sliceLines("a\nb\nc", acpsdk.Ptr(2), acpsdk.Ptr(2)) != "b\nc" {
		t.Fatalf("sliceLines() returned unexpected content")
	}

	raw := mustMarshalJSON(map[string]string{"hello": "world"})
	if string(cloneRawJSON(raw)) != string(raw) {
		t.Fatalf("cloneRawJSON() = %q, want %q", string(cloneRawJSON(raw)), string(raw))
	}

	if requestError(ErrPermissionDenied) == nil {
		t.Fatal("requestError(permission) = nil, want request error")
	}
	if requestError(errors.New("boom")) == nil {
		t.Fatal("requestError(internal) = nil, want request error")
	}

	buffer := &lockedBuffer{}
	if _, err := buffer.Write([]byte("stderr")); err != nil {
		t.Fatalf("lockedBuffer.Write() error = %v", err)
	}
	if buffer.String() != "stderr" {
		t.Fatalf("lockedBuffer.String() = %q, want %q", buffer.String(), "stderr")
	}

	if firstNonBlank("", "  ", "value") != "value" {
		t.Fatal("firstNonBlank() did not pick first non-blank value")
	}
	if chooseFloat64(nil, acpsdk.Ptr(1.2)) == nil {
		t.Fatal("chooseFloat64(nil, fallback) = nil, want fallback")
	}
	if chooseString(nil, acpsdk.Ptr("usd")) == nil {
		t.Fatal("chooseString(nil, fallback) = nil, want fallback")
	}
}

func TestHandleSessionUpdateVariants(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeApproveAll)
	active, err := proc.beginPrompt("turn-update", 16)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}
	defer proc.endPrompt(active)

	title := "permission"
	agentMessage := mustMarshalJSON(wireSessionNotification{
		SessionID: "sess-direct",
		Update: mustMarshalJSON(map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "hello"},
		}),
	})
	if err := proc.handleSessionUpdate(agentMessage); err != nil {
		t.Fatalf("handleSessionUpdate(agent_message_chunk) error = %v", err)
	}

	usageUpdate := mustMarshalJSON(wireSessionNotification{
		SessionID: "sess-direct",
		Update: mustMarshalJSON(map[string]any{
			"sessionUpdate": "usage_update",
			"used":          10,
			"size":          20,
			"cost": map[string]any{
				"amount":   1.5,
				"currency": "USD",
			},
		}),
	})
	if err := proc.handleSessionUpdate(usageUpdate); err != nil {
		t.Fatalf("handleSessionUpdate(usage_update) error = %v", err)
	}

	toolCall := mustMarshalJSON(wireSessionNotification{
		SessionID: "sess-direct",
		Update: mustMarshalJSON(map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "tool-1",
			"title":         title,
			"status":        "in_progress",
		}),
	})
	if err := proc.handleSessionUpdate(toolCall); err != nil {
		t.Fatalf("handleSessionUpdate(tool_call) error = %v", err)
	}

	modeUpdate := mustMarshalJSON(wireSessionNotification{
		SessionID: "sess-direct",
		Update: mustMarshalJSON(map[string]any{
			"sessionUpdate": "current_mode_update",
			"currentModeId": "code",
		}),
	})
	if err := proc.handleSessionUpdate(modeUpdate); err != nil {
		t.Fatalf("handleSessionUpdate(current_mode_update) error = %v", err)
	}

	events := collectEventsUntilCount(t, active.events, 4)
	if events[0].Type != EventTypeAgentMessage {
		t.Fatalf("agent message event = %#v, want agent message", events[0])
	}
	if events[1].Type != EventTypeUsage || events[1].Usage == nil || events[1].Usage.ContextUsed == nil || *events[1].Usage.ContextUsed != 10 {
		t.Fatalf("usage event = %#v, want usage metadata", events[1])
	}
	if events[2].Type != EventTypeToolCall {
		t.Fatalf("tool call event = %#v, want tool call", events[2])
	}
	if events[3].Type != EventTypeSystem {
		t.Fatalf("system event = %#v, want system", events[3])
	}
}

func TestAccessorsAndValidationHelpers(t *testing.T) {
	t.Parallel()

	if err := (StartOpts{}).Validate(); err == nil {
		t.Fatal("StartOpts.Validate() error = nil, want validation error")
	}
	if err := (PromptRequest{}).Validate(); err == nil {
		t.Fatal("PromptRequest.Validate() error = nil, want validation error")
	}

	nilProc := (*AgentProcess)(nil)
	if done := nilProc.Done(); done == nil {
		t.Fatal("(*AgentProcess)(nil).Done() = nil, want closed channel")
	}
	if err := nilProc.Wait(); err == nil {
		t.Fatal("(*AgentProcess)(nil).Wait() error = nil, want error")
	}
	if stderr := nilProc.Stderr(); stderr != "" {
		t.Fatalf("(*AgentProcess)(nil).Stderr() = %q, want empty", stderr)
	}

	proc := &AgentProcess{stderr: &lockedBuffer{}}
	if _, err := proc.stderr.Write([]byte("boom")); err != nil {
		t.Fatalf("stderr.Write() error = %v", err)
	}
	if proc.Stderr() != "boom" {
		t.Fatalf("Stderr() = %q, want %q", proc.Stderr(), "boom")
	}
}

func TestPermissionHelperBranches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	policy, err := newPermissionPolicy(aghconfig.PermissionModeApproveAll, root)
	if err != nil {
		t.Fatalf("newPermissionPolicy() error = %v", err)
	}

	resolved, err := policy.resolvePathList([]acpsdk.ToolCallLocation{{Path: filepath.Join(root, "inside.txt")}})
	if err != nil {
		t.Fatalf("resolvePathList() error = %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolvePathList() = %#v, want one path", resolved)
	}

	allowOutcome := selectPermissionOutcome([]acpsdk.PermissionOption{
		{OptionId: "allow-always", Name: "allow", Kind: acpsdk.PermissionOptionKindAllowAlways},
	}, decisionAllow)
	if allowOutcome.Selected == nil || allowOutcome.Selected.OptionId != "allow-always" {
		t.Fatalf("selectPermissionOutcome(allow) = %#v, want allow-always", allowOutcome)
	}

	cancelOutcome := selectPermissionOutcome(nil, decisionDeny)
	if cancelOutcome.Cancelled == nil {
		t.Fatalf("selectPermissionOutcome(cancel) = %#v, want cancelled", cancelOutcome)
	}
}

func newDirectProcess(t *testing.T, mode aghconfig.PermissionMode) *AgentProcess {
	t.Helper()

	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	policy, err := newPermissionPolicy(mode, root)
	if err != nil {
		t.Fatalf("newPermissionPolicy() error = %v", err)
	}

	proc := &AgentProcess{
		AgentName:     "direct",
		Cwd:           root,
		SessionID:     "sess-direct",
		StartedAt:     timeNowUTC(),
		permissions:   policy,
		terminals:     newTerminalManager(ctx, slog.Default()),
		done:          make(chan struct{}),
		cancelProcess: cancel,
		stderr:        &lockedBuffer{},
	}
	t.Cleanup(proc.terminals.closeAll)
	return proc
}

func collectEventsUntilCount(t *testing.T, eventsCh <-chan AgentEvent, want int) []AgentEvent {
	t.Helper()

	events := make([]AgentEvent, 0, want)
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for len(events) < want {
		select {
		case event := <-eventsCh:
			events = append(events, event)
		case <-timeout.C:
			t.Fatalf("timeout collecting %d events; got %#v", want, events)
		}
	}
	return events
}
