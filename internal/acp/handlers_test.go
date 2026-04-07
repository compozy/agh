package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
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

	rootReal := t.TempDir()
	rootLink := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(rootReal, rootLink); err != nil {
		t.Fatalf("os.Symlink(root) error = %v", err)
	}
	additionalReal := t.TempDir()
	additionalLink := filepath.Join(t.TempDir(), "additional-link")
	if err := os.Symlink(additionalReal, additionalLink); err != nil {
		t.Fatalf("os.Symlink(additional) error = %v", err)
	}

	normalizedWithDirs, err := normalizeStartOpts(StartOpts{
		AgentName:      "helper",
		Command:        "sh -c 'echo ok'",
		Cwd:            rootLink,
		AdditionalDirs: []string{additionalLink, rootReal, additionalLink, "   "},
	})
	if err != nil {
		t.Fatalf("normalizeStartOpts(with additional dirs) error = %v", err)
	}
	if got, want := normalizedWithDirs.Cwd, mustCanonicalDir(t, rootReal); got != want {
		t.Fatalf("normalizeStartOpts() cwd = %q, want %q", got, want)
	}
	if got, want := normalizedWithDirs.AdditionalDirs, []string{mustCanonicalDir(t, additionalReal)}; !slices.Equal(got, want) {
		t.Fatalf("normalizeStartOpts() additional dirs = %#v, want %#v", got, want)
	}
}

func TestNormalizeStartOptsRejectsInvalidAdditionalDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := filepath.Join(root, "missing")

	if err := (StartOpts{
		AgentName:      "helper",
		Command:        "sh -c 'echo ok'",
		Cwd:            root,
		AdditionalDirs: []string{"relative/path"},
	}).Validate(); err == nil {
		t.Fatal("StartOpts.Validate(relative additional dir) error = nil, want non-nil")
	}

	if _, err := normalizeStartOpts(StartOpts{
		AgentName:      "helper",
		Command:        "sh -c 'echo ok'",
		Cwd:            root,
		AdditionalDirs: []string{missing},
	}); err == nil {
		t.Fatal("normalizeStartOpts(missing additional dir) error = nil, want non-nil")
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
	proc.permissionTimeout = time.Second
	active, err := proc.beginPrompt("turn-permission", 8)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}
	defer proc.endPrompt(active)

	title := "permission request"
	path := filepath.Join(proc.Cwd, "secret.txt")
	kind := acpsdk.ToolKindEdit
	request := acpsdk.RequestPermissionRequest{
		SessionId: "sess-direct",
		Options: []acpsdk.PermissionOption{
			{OptionId: "allow-once", Name: "allow once", Kind: acpsdk.PermissionOptionKindAllowOnce},
			{OptionId: "allow-always", Name: "allow always", Kind: acpsdk.PermissionOptionKindAllowAlways},
			{OptionId: "reject-once", Name: "reject once", Kind: acpsdk.PermissionOptionKindRejectOnce},
			{OptionId: "reject-always", Name: "reject always", Kind: acpsdk.PermissionOptionKindRejectAlways},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool-1",
			Title:      &title,
			Kind:       &kind,
			RawInput: map[string]any{
				"command": "rm -rf /tmp/demo",
			},
			Locations: []acpsdk.ToolCallLocation{{Path: path}},
		},
	}
	resultCh := make(chan acpsdk.RequestPermissionResponse, 1)
	errCh := make(chan *acpsdk.RequestError, 1)
	go func() {
		response, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodSessionRequestPermission, mustMarshalJSON(request))
		if reqErr != nil {
			errCh <- reqErr
			return
		}
		permissionResponse, ok := response.(acpsdk.RequestPermissionResponse)
		if !ok {
			errCh <- requestError(errors.New("unexpected permission response type"))
			return
		}
		resultCh <- permissionResponse
	}()

	initialEvents := collectEventsUntilCount(t, active.events, 1)
	if len(initialEvents) != 1 || initialEvents[0].Type != EventTypePermission {
		t.Fatalf("initial permission events = %#v, want one permission event", initialEvents)
	}
	if initialEvents[0].Decision != "" {
		t.Fatalf("initial permission decision = %q, want empty", initialEvents[0].Decision)
	}
	if initialEvents[0].RequestID == "" {
		t.Fatal("initial permission request_id = empty, want non-empty")
	}

	raw := decodePermissionEventRaw(t, initialEvents[0].Raw)
	if raw.RequestID != initialEvents[0].RequestID {
		t.Fatalf("raw.request_id = %q, want %q", raw.RequestID, initialEvents[0].RequestID)
	}
	if len(raw.Options) != 4 {
		t.Fatalf("raw.options = %#v, want 4 permission options", raw.Options)
	}
	if got := raw.ToolInput["command"]; got != "rm -rf /tmp/demo" {
		t.Fatalf("raw.tool_input.command = %#v, want %q", got, "rm -rf /tmp/demo")
	}

	if err := proc.ResolvePermission(ApproveRequest{
		RequestID: initialEvents[0].RequestID,
		Decision:  string(decisionAllowAlways),
	}); err != nil {
		t.Fatalf("ResolvePermission() error = %v", err)
	}

	select {
	case reqErr := <-errCh:
		t.Fatalf("handleInbound(permission) error = %v", reqErr)
	case permissionResponse := <-resultCh:
		if permissionResponse.Outcome.Selected == nil || permissionResponse.Outcome.Selected.OptionId != "allow-always" {
			t.Fatalf("permission outcome = %#v, want allow-always option", permissionResponse.Outcome)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for permission response")
	}

	finalEvents := collectEventsUntilCount(t, active.events, 1)
	if len(finalEvents) != 1 || finalEvents[0].Decision != string(decisionAllowAlways) {
		t.Fatalf("final permission events = %#v, want allow-always decision", finalEvents)
	}
}

func TestResolvePermissionUnknownRequest(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	err := proc.ResolvePermission(ApproveRequest{
		RequestID: "missing",
		Decision:  string(decisionAllowOnce),
	})
	if !errors.Is(err, ErrPendingPermissionNotFound) {
		t.Fatalf("ResolvePermission(missing) error = %v, want ErrPendingPermissionNotFound", err)
	}
}

func TestHandleInboundPermissionRequestTimeout(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	proc.permissionTimeout = 25 * time.Millisecond
	active, err := proc.beginPrompt("turn-timeout", 8)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}
	defer proc.endPrompt(active)

	title := "permission request"
	kind := acpsdk.ToolKindEdit
	response, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodSessionRequestPermission, mustMarshalJSON(acpsdk.RequestPermissionRequest{
		SessionId: "sess-direct",
		Options: []acpsdk.PermissionOption{
			{OptionId: "allow-once", Name: "allow once", Kind: acpsdk.PermissionOptionKindAllowOnce},
			{OptionId: "reject-once", Name: "reject once", Kind: acpsdk.PermissionOptionKindRejectOnce},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool-timeout",
			Title:      &title,
			Kind:       &kind,
		},
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(permission timeout) error = %v", reqErr)
	}

	permissionResponse, ok := response.(acpsdk.RequestPermissionResponse)
	if !ok {
		t.Fatalf("handleInbound(permission timeout) type = %T, want RequestPermissionResponse", response)
	}
	if permissionResponse.Outcome.Selected == nil || permissionResponse.Outcome.Selected.OptionId != "reject-once" {
		t.Fatalf("permission timeout outcome = %#v, want reject-once option", permissionResponse.Outcome)
	}

	events := collectEventsUntilCount(t, active.events, 2)
	if events[0].Decision != "" {
		t.Fatalf("initial timeout decision = %q, want empty", events[0].Decision)
	}
	if events[1].Decision != string(decisionRejectOnce) {
		t.Fatalf("final timeout decision = %q, want %q", events[1].Decision, decisionRejectOnce)
	}
}

func TestEmitPermissionEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decision permissionDecision
	}{
		{name: "ShouldHandleInteractivePending", decision: ""},
		{name: "ShouldAllowOnceAutomatically", decision: decisionAllowOnce},
		{name: "ShouldRejectOnceOnTimeout", decision: decisionRejectOnce},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
			active, err := proc.beginPrompt("turn-permission-event", 4)
			if err != nil {
				t.Fatalf("beginPrompt() error = %v", err)
			}
			defer proc.endPrompt(active)

			raw := mustMarshalJSON(map[string]any{"decision": string(tt.decision), "value": "original"})
			wantRaw := append(json.RawMessage(nil), raw...)

			proc.emitPermissionEvent("sess-emit", "turn-permission-event", "req-1", "permission request", "tool-1", "/tmp/demo.txt", tt.decision, raw)
			event := collectEventsUntilCount(t, active.events, 1)[0]

			raw[0] = '['

			if event.Type != EventTypePermission {
				t.Fatalf("event.Type = %q, want %q", event.Type, EventTypePermission)
			}
			if event.SessionID != "sess-emit" || event.TurnID != "turn-permission-event" || event.RequestID != "req-1" {
				t.Fatalf("event ids = %#v, want session/turn/request populated", event)
			}
			if event.Title != "permission request" || event.ToolCallID != "tool-1" {
				t.Fatalf("event title/tool = %#v, want copied fields", event)
			}
			if event.Action != string(permissionRequestToolGrant) || event.Resource != "/tmp/demo.txt" {
				t.Fatalf("event action/resource = %#v, want permission action/resource", event)
			}
			if event.Decision != string(tt.decision) {
				t.Fatalf("event.Decision = %q, want %q", event.Decision, tt.decision)
			}
			if event.Timestamp.IsZero() {
				t.Fatal("event.Timestamp = zero, want populated")
			}
			if string(event.Raw) != string(wantRaw) {
				t.Fatalf("event.Raw = %s, want %s", string(event.Raw), string(wantRaw))
			}
		})
	}
}

func TestResolvePermissionByTurnIDConflictsWhenMultipleRequestsPending(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	turnID := "turn-conflict"
	_, first := proc.registerPendingPermission(turnID, acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{ToolCallId: "tool-1"},
	})
	_, second := proc.registerPendingPermission(turnID, acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{ToolCallId: "tool-2"},
	})
	t.Cleanup(func() {
		proc.clearPendingPermission(first.requestID)
		proc.clearPendingPermission(second.requestID)
	})

	err := proc.ResolvePermission(ApproveRequest{
		TurnID:   turnID,
		Decision: string(decisionRejectOnce),
	})
	if !errors.Is(err, ErrPendingPermissionConflict) {
		t.Fatalf("ResolvePermission(turn conflict) error = %v, want ErrPendingPermissionConflict", err)
	}
}

func TestResolvePermissionConcurrentSafety(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)

	const total = 8
	type registered struct {
		requestID string
		response  chan permissionDecision
	}

	registeredPending := make([]registered, 0, total)
	for i := 0; i < total; i++ {
		requestID, pending := proc.registerPendingPermission(
			fmt.Sprintf("turn-%d", i),
			acpsdk.RequestPermissionRequest{ToolCall: acpsdk.RequestPermissionToolCall{ToolCallId: acpsdk.ToolCallId(fmt.Sprintf("tool-%d", i))}},
		)
		registeredPending = append(registeredPending, registered{
			requestID: requestID,
			response:  pending.response,
		})
	}

	var wg sync.WaitGroup
	for _, pending := range registeredPending {
		pending := pending
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := proc.ResolvePermission(ApproveRequest{
				RequestID: pending.requestID,
				Decision:  string(decisionAllowOnce),
			}); err != nil {
				t.Errorf("ResolvePermission(%q) error = %v", pending.requestID, err)
			}
		}()
	}
	wg.Wait()

	for _, pending := range registeredPending {
		select {
		case decision := <-pending.response:
			if decision != decisionAllowOnce {
				t.Fatalf("pending response = %q, want %q", decision, decisionAllowOnce)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for pending response %q", pending.requestID)
		}
	}
	if len(proc.pendingPermissions) != 0 {
		t.Fatalf("pendingPermissions = %#v, want empty", proc.pendingPermissions)
	}
}

func TestHandleInboundPermissionRequestAutoApprovesReadRequests(t *testing.T) {
	t.Parallel()

	proc := newDirectProcess(t, aghconfig.PermissionModeApproveReads)
	active, err := proc.beginPrompt("turn-read", 8)
	if err != nil {
		t.Fatalf("beginPrompt() error = %v", err)
	}
	defer proc.endPrompt(active)

	title := "read file"
	kind := acpsdk.ToolKindRead
	response, reqErr := proc.handleInbound(context.Background(), acpsdk.ClientMethodSessionRequestPermission, mustMarshalJSON(acpsdk.RequestPermissionRequest{
		SessionId: "sess-direct",
		Options: []acpsdk.PermissionOption{
			{OptionId: "allow-once", Name: "allow once", Kind: acpsdk.PermissionOptionKindAllowOnce},
			{OptionId: "reject-once", Name: "reject once", Kind: acpsdk.PermissionOptionKindRejectOnce},
		},
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: "tool-read",
			Title:      &title,
			Kind:       &kind,
			Locations:  []acpsdk.ToolCallLocation{{Path: filepath.Join(proc.Cwd, "notes.txt")}},
		},
	}))
	if reqErr != nil {
		t.Fatalf("handleInbound(read permission) error = %v", reqErr)
	}

	permissionResponse, ok := response.(acpsdk.RequestPermissionResponse)
	if !ok {
		t.Fatalf("handleInbound(read permission) type = %T, want RequestPermissionResponse", response)
	}
	if permissionResponse.Outcome.Selected == nil || permissionResponse.Outcome.Selected.OptionId != "allow-once" {
		t.Fatalf("permission outcome = %#v, want reject option", permissionResponse.Outcome)
	}

	events := collectEventsUntilCount(t, active.events, 1)
	if len(events) != 1 || events[0].Decision != string(decisionAllowOnce) {
		t.Fatalf("permission events = %#v, want allow-once permission event", events)
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
	if string(CloneRawMessage(raw)) != string(raw) {
		t.Fatalf("CloneRawMessage() = %q, want %q", string(CloneRawMessage(raw)), string(raw))
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

	allowOutcome, allowDecision := selectPermissionOutcome([]acpsdk.PermissionOption{
		{OptionId: "allow-once", Name: "allow once", Kind: acpsdk.PermissionOptionKindAllowOnce},
		{OptionId: "allow-always", Name: "allow", Kind: acpsdk.PermissionOptionKindAllowAlways},
	}, decisionAllowOnce)
	if allowOutcome.Selected == nil || allowOutcome.Selected.OptionId != "allow-once" || allowDecision != decisionAllowOnce {
		t.Fatalf("selectPermissionOutcome(allow-once) = %#v, %q", allowOutcome, allowDecision)
	}

	rejectOutcome, rejectDecision := selectPermissionOutcome([]acpsdk.PermissionOption{
		{OptionId: "reject-once", Name: "reject once", Kind: acpsdk.PermissionOptionKindRejectOnce},
		{OptionId: "reject-always", Name: "reject always", Kind: acpsdk.PermissionOptionKindRejectAlways},
	}, decisionRejectAlways)
	if rejectOutcome.Selected == nil || rejectOutcome.Selected.OptionId != "reject-always" || rejectDecision != decisionRejectAlways {
		t.Fatalf("selectPermissionOutcome(reject-always) = %#v, %q", rejectOutcome, rejectDecision)
	}

	cancelOutcome, cancelDecision := selectPermissionOutcome(nil, decisionRejectOnce)
	if cancelOutcome.Cancelled == nil {
		t.Fatalf("selectPermissionOutcome(cancel) = %#v, want cancelled", cancelOutcome)
	}
	if cancelDecision != "" {
		t.Fatalf("selectPermissionOutcome(cancel) decision = %q, want empty", cancelDecision)
	}

	if _, err := parsePermissionDecision("maybe"); err == nil {
		t.Fatal("parsePermissionDecision(invalid) error = nil, want non-nil")
	}
	if err := (ApproveRequest{Decision: string(decisionAllowOnce)}).Validate(); err == nil {
		t.Fatal("ApproveRequest.Validate(missing request id and turn id) error = nil, want non-nil")
	}

	readKind := acpsdk.ToolKindRead
	readDecision, interactive := policy.permissionDecision(acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{
			Kind:      &readKind,
			Locations: []acpsdk.ToolCallLocation{{Path: filepath.Join(root, "inside.txt")}},
		},
	})
	if readDecision != decisionAllowOnce || interactive {
		t.Fatalf("permissionDecision(read) = %q, %v, want %q, false", readDecision, interactive, decisionAllowOnce)
	}

	approveReadsPolicy, err := newPermissionPolicy(aghconfig.PermissionModeApproveReads, root)
	if err != nil {
		t.Fatalf("newPermissionPolicy(approve-reads) error = %v", err)
	}
	editKind := acpsdk.ToolKindEdit
	editDecision, interactive := approveReadsPolicy.permissionDecision(acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{Kind: &editKind},
	})
	if editDecision != decisionPending || !interactive {
		t.Fatalf("permissionDecision(edit) = %q, %v, want %q, true", editDecision, interactive, decisionPending)
	}

	if got := permissionRequestIDFromMeta(map[string]any{"request_id": "req-meta"}); got != "req-meta" {
		t.Fatalf("permissionRequestIDFromMeta() = %q, want %q", got, "req-meta")
	}
	title := "Write file"
	if got := permissionRequestName("turn-1", acpsdk.RequestPermissionRequest{
		ToolCall: acpsdk.RequestPermissionToolCall{
			Title: &title,
			Kind:  &editKind,
		},
	}); got != "turn-1:Write file" {
		t.Fatalf("permissionRequestName() = %q, want %q", got, "turn-1:Write file")
	}

	proc := newDirectProcess(t, aghconfig.PermissionModeDenyAll)
	if got := proc.nextPermissionRequestID("turn-1", acpsdk.RequestPermissionRequest{
		Meta: map[string]any{"request_id": "req-from-meta"},
	}); got != "req-from-meta" {
		t.Fatalf("nextPermissionRequestID(meta) = %q, want %q", got, "req-from-meta")
	}
	if got := (&AgentProcess{}).permissionTimeoutOrDefault(); got != 5*time.Minute {
		t.Fatalf("permissionTimeoutOrDefault() = %v, want %v", got, 5*time.Minute)
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
		AgentName:         "direct",
		Cwd:               root,
		SessionID:         "sess-direct",
		StartedAt:         timeNowUTC(),
		permissions:       policy,
		terminals:         newTerminalManager(ctx, slog.Default()),
		done:              make(chan struct{}),
		cancelProcess:     cancel,
		stderr:            &lockedBuffer{},
		permissionTimeout: time.Second,
	}
	t.Cleanup(proc.terminals.closeAll)
	return proc
}

func decodePermissionEventRaw(t *testing.T, raw json.RawMessage) struct {
	RequestID string                  `json:"request_id"`
	ToolInput map[string]any          `json:"tool_input"`
	Options   []permissionEventOption `json:"options"`
} {
	t.Helper()

	var payload struct {
		RequestID string                  `json:"request_id"`
		ToolInput map[string]any          `json:"tool_input"`
		Options   []permissionEventOption `json:"options"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal(permission raw) error = %v; raw=%s", err, string(raw))
	}
	return payload
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
