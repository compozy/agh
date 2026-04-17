package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
)

var (
	_ acpsdk.Agent             = (*mockAgent)(nil)
	_ acpsdk.AgentLoader       = (*mockAgent)(nil)
	_ acpsdk.AgentExperimental = (*mockAgent)(nil)
)

type cliArgs struct {
	FixturePath     string
	AgentName       string
	DiagnosticsPath string
}

type sessionState struct {
	PromptCount int
}

type mockAgent struct {
	conn            *acpsdk.AgentSideConnection
	agent           acpmock.AgentFixture
	diagnosticsPath string

	mu          sync.Mutex
	sessions    map[string]*sessionState
	nextSession int
}

type environmentRunResult struct {
	Output        string
	ExitCode      *int
	ObservedError string
}

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fixture, err := acpmock.LoadFixture(args.FixturePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	agentFixture, err := fixture.Agent(args.AgentName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent := &mockAgent{
		agent:           agentFixture,
		diagnosticsPath: strings.TrimSpace(args.DiagnosticsPath),
		sessions:        make(map[string]*sessionState),
	}
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.SetAgentConnection(conn)
	<-conn.Done()
}

func parseArgs(argv []string) (cliArgs, error) {
	fs := flag.NewFlagSet("acpmock-driver", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var args cliArgs
	fs.StringVar(&args.FixturePath, "fixture", "", "fixture JSON path")
	fs.StringVar(&args.AgentName, "agent", "", "fixture agent name")
	fs.StringVar(&args.DiagnosticsPath, "diagnostics", "", "diagnostics jsonl path")

	if err := fs.Parse(argv); err != nil {
		return cliArgs{}, err
	}
	if strings.TrimSpace(args.FixturePath) == "" {
		return cliArgs{}, errors.New("acpmock-driver: --fixture is required")
	}
	if strings.TrimSpace(args.AgentName) == "" {
		return cliArgs{}, errors.New("acpmock-driver: --agent is required")
	}
	return args, nil
}

func (a *mockAgent) SetAgentConnection(conn *acpsdk.AgentSideConnection) {
	a.conn = conn
}

func (a *mockAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *mockAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (a *mockAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (a *mockAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.nextSession++
	sessionID := fmt.Sprintf("%s-session-%d", a.agent.Name, a.nextSession)
	a.sessions[sessionID] = &sessionState{}
	return acpsdk.NewSessionResponse{SessionId: acpsdk.SessionId(sessionID)}, nil
}

func (a *mockAgent) LoadSession(
	_ context.Context,
	params acpsdk.LoadSessionRequest,
) (acpsdk.LoadSessionResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID != "" && a.sessions[sessionID] == nil {
		a.sessions[sessionID] = &sessionState{}
	}
	return acpsdk.LoadSessionResponse{}, nil
}

func (a *mockAgent) SetSessionMode(
	context.Context,
	acpsdk.SetSessionModeRequest,
) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func (a *mockAgent) SetSessionModel(
	context.Context,
	acpsdk.SetSessionModelRequest,
) (acpsdk.SetSessionModelResponse, error) {
	return acpsdk.SetSessionModelResponse{}, nil
}

func (a *mockAgent) Prompt(ctx context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acpsdk.PromptResponse{}, errors.New("sessionId is required")
	}

	promptMeta, err := decodePromptMeta(params.Meta)
	if err != nil {
		return acpsdk.PromptResponse{}, err
	}

	prompt, occurrence := a.recordPrompt(sessionID, extractPromptText(params.Prompt))
	turn, err := a.agent.SelectTurn(prompt, occurrence, promptMeta)
	if err != nil {
		return acpsdk.PromptResponse{}, err
	}

	record := acpmock.DiagnosticsRecord{
		AgentName:   a.agent.Name,
		SessionID:   sessionID,
		PromptIndex: occurrence,
		Prompt:      prompt,
		PromptMeta:  promptMeta,
		TurnName:    strings.TrimSpace(turn.Name),
		Match:       turn.Match.Normalize(),
		Steps:       make([]acpmock.DiagnosticsStep, 0, len(turn.Steps)),
	}

	for _, step := range turn.Steps {
		entry, execErr := a.executeStep(ctx, acpsdk.SessionId(sessionID), step)
		if execErr != nil {
			record.Steps = append(record.Steps, acpmock.DiagnosticsStep{
				Kind:  acpmock.StepKind("error"),
				Error: execErr.Error(),
			})
			if diagErr := a.writeDiagnostics(record); diagErr != nil {
				return acpsdk.PromptResponse{}, diagErr
			}
			if errors.Is(execErr, context.Canceled) {
				return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonCancelled}, nil
			}
			return acpsdk.PromptResponse{}, execErr
		}
		record.Steps = append(record.Steps, entry)
	}

	if err := a.writeDiagnostics(record); err != nil {
		return acpsdk.PromptResponse{}, err
	}
	return acpsdk.PromptResponse{StopReason: stopReason(turn.StopReason)}, nil
}

func (a *mockAgent) recordPrompt(sessionID string, prompt string) (string, int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.sessions[sessionID]
	if session == nil {
		session = &sessionState{}
		a.sessions[sessionID] = session
	}
	session.PromptCount++
	return prompt, session.PromptCount
}

func (a *mockAgent) executeStep(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	switch step.Kind {
	case acpmock.StepKindAssistant:
		return a.emitTextChunks(ctx, sessionID, acpsdk.UpdateAgentMessageText, step)
	case acpmock.StepKindThought:
		return a.emitTextChunks(ctx, sessionID, acpsdk.UpdateAgentThoughtText, step)
	case acpmock.StepKindBridgeContent:
		return a.emitTextChunks(ctx, sessionID, acpsdk.UpdateAgentMessageText, step)
	case acpmock.StepKindToolCall:
		return a.emitToolCall(ctx, sessionID, step)
	case acpmock.StepKindPermission:
		return a.requestPermission(ctx, sessionID, step)
	case acpmock.StepKindEnvironment:
		return a.executeEnvironmentCommand(ctx, sessionID, step)
	case acpmock.StepKindDriverControl:
		return a.executeDriverControl(ctx, step)
	default:
		return acpmock.DiagnosticsStep{}, fmt.Errorf("unsupported step kind %s", step.Kind)
	}
}

func (a *mockAgent) emitTextChunks(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	update func(string) acpsdk.SessionUpdate,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	chunks := normalizedChunks(step)
	for _, chunk := range chunks {
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    update(chunk),
		}); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
	}
	return acpmock.DiagnosticsStep{Kind: step.Kind, Text: strings.Join(chunks, "")}, nil
}

func (a *mockAgent) emitToolCall(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	toolCallID := acpsdk.ToolCallId(strings.TrimSpace(step.ToolCallID))
	title := strings.TrimSpace(step.Title)

	startOpts := []acpsdk.ToolCallStartOpt{
		acpsdk.WithStartKind(toolKind(step.ToolKind, acpsdk.ToolKindOther)),
		acpsdk.WithStartStatus(acpsdk.ToolCallStatusPending),
	}
	if locations := toolLocations(step.Path); len(locations) > 0 {
		startOpts = append(startOpts, acpsdk.WithStartLocations(locations))
	}
	if rawInput, ok := parseRawJSON(step.RawInput); ok {
		startOpts = append(startOpts, acpsdk.WithStartRawInput(rawInput))
	}

	if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
		SessionId: sessionID,
		Update:    acpsdk.StartToolCall(toolCallID, title, startOpts...),
	}); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	if err := pauseForDelivery(ctx); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	finalStatus := strings.TrimSpace(step.Status)
	rawOutput, hasRawOutput := parseRawJSON(step.RawOutput)
	hasFinalUpdate := finalStatus != "" || strings.TrimSpace(step.ContentText) != "" || hasRawOutput
	if hasFinalUpdate {
		updateOpts := make([]acpsdk.ToolCallUpdateOpt, 0, 4)
		if finalStatus != "" {
			updateOpts = append(updateOpts, acpsdk.WithUpdateStatus(acpsdk.ToolCallStatus(finalStatus)))
		}
		if title != "" {
			updateOpts = append(updateOpts, acpsdk.WithUpdateTitle(title))
		}
		if text := strings.TrimSpace(step.ContentText); text != "" {
			updateOpts = append(updateOpts, acpsdk.WithUpdateContent(textToolContent(text)))
		}
		if hasRawOutput {
			updateOpts = append(updateOpts, acpsdk.WithUpdateRawOutput(rawOutput))
		}
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateToolCall(toolCallID, updateOpts...),
		}); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
	}

	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindToolCall,
		ToolCallID: string(toolCallID),
	}, nil
}

func (a *mockAgent) requestPermission(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	title := strings.TrimSpace(step.Title)
	if title == "" {
		title = "permission request"
	}
	toolKindValue := toolKind(step.ToolKind, acpsdk.ToolKindOther)
	statusValue := toolStatus(step.Status, acpsdk.ToolCallStatusPending)

	response, err := a.conn.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options:   defaultPermissionOptions(),
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: acpsdk.ToolCallId(strings.TrimSpace(step.ToolCallID)),
			Title:      acpsdk.Ptr(title),
			Kind:       acpsdk.Ptr(toolKindValue),
			Status:     acpsdk.Ptr(statusValue),
			Locations:  toolLocations(step.Path),
			RawInput:   mustRawJSON(step.RawInput),
			RawOutput:  mustRawJSON(step.RawOutput),
		},
	})
	if err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	decision := selectedDecision(response)
	if expected := strings.TrimSpace(step.ExpectDecision); expected != "" && decision != expected {
		return acpmock.DiagnosticsStep{}, fmt.Errorf(
			"permission decision %s did not match expected %s",
			decision,
			expected,
		)
	}

	switch {
	case step.EmitDecision:
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(decision),
		}); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
	case strings.TrimSpace(step.EmitText) != "":
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(strings.TrimSpace(step.EmitText)),
		}); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return acpmock.DiagnosticsStep{}, err
		}
	}

	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindPermission,
		ToolCallID: strings.TrimSpace(step.ToolCallID),
		Decision:   decision,
	}, nil
}

func (a *mockAgent) executeEnvironmentCommand(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	toolCallID, title := environmentDescriptor(step)
	if err := a.startEnvironmentToolCall(ctx, sessionID, step, toolCallID, title); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	result := a.runEnvironmentCommand(ctx, sessionID, step)
	if expected := strings.TrimSpace(step.ExpectErrorContains); expected != "" {
		return a.finishEnvironmentFailure(ctx, sessionID, step, toolCallID, title, result, expected)
	}
	if err := validateEnvironmentResult(step, result); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	if err := a.finishEnvironmentSuccess(ctx, sessionID, step, toolCallID, title, result); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindEnvironment,
		ToolCallID: toolCallID,
		Command:    strings.TrimSpace(step.Command),
		Args:       append([]string(nil), step.Args...),
		ExitCode:   result.ExitCode,
		Output:     result.Output,
	}, nil
}

func (a *mockAgent) executeDriverControl(
	ctx context.Context,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	if step.DriverControl == nil {
		return acpmock.DiagnosticsStep{}, errors.New("driver_control payload is required")
	}

	diagnostics := acpmock.DiagnosticsStep{
		Kind:         acpmock.StepKindDriverControl,
		DriverAction: step.DriverControl.Action,
		Text:         strings.TrimSpace(step.DriverControl.RawJSONRPC),
	}
	control := *step.DriverControl
	if control.Async {
		asyncCtx := context.WithoutCancel(ctx)
		go func(asyncCtx context.Context, control acpmock.DriverControlStep) {
			if err := waitDriverControlDelay(asyncCtx, control.DelayMS); err != nil {
				return
			}
			if err := a.performDriverControl(asyncCtx, control); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "acpmock async driver_control %s error: %v\n", control.Action, err)
			}
		}(asyncCtx, control)
		return diagnostics, nil
	}
	if err := waitDriverControlDelay(ctx, control.DelayMS); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	return diagnostics, a.performDriverControl(ctx, control)
}

func waitDriverControlDelay(ctx context.Context, delayMS int) error {
	if delayMS <= 0 {
		return nil
	}
	timer := time.NewTimer(time.Duration(delayMS) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *mockAgent) performDriverControl(ctx context.Context, control acpmock.DriverControlStep) error {
	switch control.Action {
	case acpmock.DriverControlDisconnect:
		os.Exit(23)
		return nil
	case acpmock.DriverControlWriteRawJSONRPC:
		frame := control.RawJSONRPC
		if !strings.HasSuffix(frame, "\n") {
			frame += "\n"
		}
		_, err := os.Stdout.WriteString(frame)
		return err
	case acpmock.DriverControlBlockUntilCancel:
		<-ctx.Done()
		return ctx.Err()
	default:
		return fmt.Errorf("unsupported driver_control action %s", control.Action)
	}
}

func (a *mockAgent) writeDiagnostics(record acpmock.DiagnosticsRecord) error {
	if strings.TrimSpace(a.diagnosticsPath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.diagnosticsPath), 0o755); err != nil {
		return fmt.Errorf("create diagnostics directory %q: %w", filepath.Dir(a.diagnosticsPath), err)
	}
	file, err := os.OpenFile(a.diagnosticsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open diagnostics %q: %w", a.diagnosticsPath, err)
	}
	defer func() { _ = file.Close() }()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode diagnostics: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write diagnostics %q: %w", a.diagnosticsPath, err)
	}
	return nil
}

func environmentDescriptor(step acpmock.Step) (string, string) {
	toolCallID := strings.TrimSpace(step.ToolCallID)
	title := strings.TrimSpace(step.Title)
	if title == "" {
		title = strings.TrimSpace(step.Command)
	}
	if title == "" {
		title = "environment command"
	}
	return toolCallID, title
}

func (a *mockAgent) startEnvironmentToolCall(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
) error {
	if toolCallID == "" {
		return nil
	}

	if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
		SessionId: sessionID,
		Update: acpsdk.StartToolCall(
			acpsdk.ToolCallId(toolCallID),
			title,
			acpsdk.WithStartKind(toolKind(step.ToolKind, acpsdk.ToolKindExecute)),
			acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress),
			acpsdk.WithStartRawInput(map[string]any{
				"command": strings.TrimSpace(step.Command),
				"args":    append([]string(nil), step.Args...),
			}),
		),
	}); err != nil {
		return err
	}
	return pauseForDelivery(ctx)
}

func (a *mockAgent) runEnvironmentCommand(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) environmentRunResult {
	req := acpsdk.CreateTerminalRequest{
		SessionId: sessionID,
		Command:   strings.TrimSpace(step.Command),
		Args:      append([]string(nil), step.Args...),
	}
	if cwd := strings.TrimSpace(step.Cwd); cwd != "" {
		req.Cwd = acpsdk.Ptr(cwd)
	}

	createResp, err := a.conn.CreateTerminal(ctx, req)
	if err != nil {
		return environmentRunResult{ObservedError: err.Error()}
	}

	result := environmentRunResult{}
	waitResp, waitErr := a.conn.WaitForTerminalExit(ctx, acpsdk.WaitForTerminalExitRequest{
		SessionId:  sessionID,
		TerminalId: createResp.TerminalId,
	})
	if waitErr != nil {
		result.ObservedError = waitErr.Error()
	} else {
		result.ExitCode = waitResp.ExitCode
		outputResp, outputErr := a.conn.TerminalOutput(ctx, acpsdk.TerminalOutputRequest{
			SessionId:  sessionID,
			TerminalId: createResp.TerminalId,
		})
		if outputErr != nil {
			result.ObservedError = outputErr.Error()
		} else {
			result.Output = outputResp.Output
		}
	}

	_, releaseErr := a.conn.ReleaseTerminal(ctx, acpsdk.ReleaseTerminalRequest{
		SessionId:  sessionID,
		TerminalId: createResp.TerminalId,
	})
	if result.ObservedError == "" && releaseErr != nil {
		result.ObservedError = releaseErr.Error()
	}
	return result
}

func (a *mockAgent) finishEnvironmentFailure(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
	result environmentRunResult,
	expected string,
) (acpmock.DiagnosticsStep, error) {
	if result.ObservedError == "" || !strings.Contains(result.ObservedError, expected) {
		return acpmock.DiagnosticsStep{}, fmt.Errorf(
			"environment command error %s did not include %s",
			result.ObservedError,
			expected,
		)
	}
	if err := a.emitEnvironmentFailure(ctx, sessionID, step, toolCallID, title, result.ObservedError); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindEnvironment,
		ToolCallID: toolCallID,
		Command:    strings.TrimSpace(step.Command),
		Args:       append([]string(nil), step.Args...),
		Error:      result.ObservedError,
	}, nil
}

func validateEnvironmentResult(step acpmock.Step, result environmentRunResult) error {
	if result.ObservedError != "" {
		return errors.New(result.ObservedError)
	}
	if step.ExpectExitCode != nil {
		if result.ExitCode == nil || *result.ExitCode != *step.ExpectExitCode {
			got := "<nil>"
			if result.ExitCode != nil {
				got = fmt.Sprintf("%d", *result.ExitCode)
			}
			return fmt.Errorf(
				"environment exit code %s did not match expected %d",
				got,
				*step.ExpectExitCode,
			)
		}
	}
	if expected := strings.TrimSpace(step.ExpectOutputContains); expected != "" &&
		!strings.Contains(result.Output, expected) {
		return fmt.Errorf("environment output %q did not include %s", result.Output, expected)
	}
	return nil
}

func (a *mockAgent) finishEnvironmentSuccess(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
	result environmentRunResult,
) error {
	if toolCallID != "" {
		updateOpts := []acpsdk.ToolCallUpdateOpt{
			acpsdk.WithUpdateStatus(toolStatus(step.Status, acpsdk.ToolCallStatusCompleted)),
			acpsdk.WithUpdateTitle(title),
		}
		if result.Output != "" {
			updateOpts = append(updateOpts, acpsdk.WithUpdateContent(textToolContent(result.Output)))
		}
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateToolCall(acpsdk.ToolCallId(toolCallID), updateOpts...),
		}); err != nil {
			return err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return err
		}
	}

	switch {
	case step.EmitOutput:
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(result.Output),
		}); err != nil {
			return err
		}
		return pauseForDelivery(ctx)
	case strings.TrimSpace(step.EmitText) != "":
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(strings.TrimSpace(step.EmitText)),
		}); err != nil {
			return err
		}
		return pauseForDelivery(ctx)
	default:
		return nil
	}
}

func (a *mockAgent) emitEnvironmentFailure(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
	observedError string,
) error {
	if toolCallID != "" {
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update: acpsdk.UpdateToolCall(
				acpsdk.ToolCallId(toolCallID),
				acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusFailed),
				acpsdk.WithUpdateTitle(title),
				acpsdk.WithUpdateContent(textToolContent(observedError)),
			),
		}); err != nil {
			return err
		}
		if err := pauseForDelivery(ctx); err != nil {
			return err
		}
	}
	if text := strings.TrimSpace(step.EmitText); text != "" {
		if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
			SessionId: sessionID,
			Update:    acpsdk.UpdateAgentMessageText(text),
		}); err != nil {
			return err
		}
		return pauseForDelivery(ctx)
	}
	return nil
}

func defaultPermissionOptions() []acpsdk.PermissionOption {
	return []acpsdk.PermissionOption{
		{
			Kind:     acpsdk.PermissionOptionKindAllowOnce,
			Name:     "allow once",
			OptionId: acpsdk.PermissionOptionId("allow-once"),
		},
		{
			Kind:     acpsdk.PermissionOptionKindAllowAlways,
			Name:     "allow always",
			OptionId: acpsdk.PermissionOptionId("allow-always"),
		},
		{
			Kind:     acpsdk.PermissionOptionKindRejectOnce,
			Name:     "reject once",
			OptionId: acpsdk.PermissionOptionId("reject-once"),
		},
		{
			Kind:     acpsdk.PermissionOptionKindRejectAlways,
			Name:     "reject always",
			OptionId: acpsdk.PermissionOptionId("reject-always"),
		},
	}
}

func selectedDecision(response acpsdk.RequestPermissionResponse) string {
	if response.Outcome.Selected != nil {
		return string(response.Outcome.Selected.OptionId)
	}
	return "canceled"
}

func normalizedChunks(step acpmock.Step) []string {
	if len(step.Chunks) > 0 {
		chunks := make([]string, 0, len(step.Chunks))
		chunks = append(chunks, step.Chunks...)
		return chunks
	}
	if strings.TrimSpace(step.Text) != "" {
		return []string{step.Text}
	}
	return []string{""}
}

func decodePromptMeta(raw any) (acp.PromptMeta, error) {
	if raw == nil {
		return acp.PromptMeta{}, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return acp.PromptMeta{}, fmt.Errorf("encode prompt metadata: %w", err)
	}

	var meta acp.PromptMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return acp.PromptMeta{}, fmt.Errorf("decode prompt metadata: %w", err)
	}
	if err := meta.Validate(); err != nil {
		return acp.PromptMeta{}, err
	}
	return meta.Normalize(), nil
}

func extractPromptText(blocks []acpsdk.ContentBlock) string {
	lastText := ""
	for _, block := range blocks {
		if block.Text != nil {
			lastText = block.Text.Text
		}
	}
	const marker = "User request:"
	if index := strings.LastIndex(lastText, marker); index >= 0 {
		return strings.TrimSpace(lastText[index+len(marker):])
	}
	return strings.TrimSpace(lastText)
}

func toolLocations(rawPath string) []acpsdk.ToolCallLocation {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return nil
	}
	return []acpsdk.ToolCallLocation{{Path: trimmed}}
}

func textToolContent(text string) []acpsdk.ToolCallContent {
	return []acpsdk.ToolCallContent{acpsdk.ToolContent(acpsdk.TextBlock(text))}
}

func toolKind(raw string, fallback acpsdk.ToolKind) acpsdk.ToolKind {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	return acpsdk.ToolKind(trimmed)
}

func toolStatus(raw string, fallback acpsdk.ToolCallStatus) acpsdk.ToolCallStatus {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	return acpsdk.ToolCallStatus(trimmed)
}

func parseRawJSON(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	return value, true
}

func mustRawJSON(raw json.RawMessage) any {
	value, _ := parseRawJSON(raw)
	return value
}

func stopReason(raw string) acpsdk.StopReason {
	switch strings.TrimSpace(raw) {
	case string(acpsdk.StopReasonCancelled):
		return acpsdk.StopReasonCancelled
	default:
		return acpsdk.StopReasonEndTurn
	}
}

func pauseForDelivery(ctx context.Context) error {
	timer := time.NewTimer(5 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
