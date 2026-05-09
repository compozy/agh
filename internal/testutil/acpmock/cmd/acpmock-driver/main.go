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
	_ acpsdk.Agent       = (*mockAgent)(nil)
	_ acpsdk.AgentLoader = (*mockAgent)(nil)
)

type cliArgs struct {
	FixturePath     string
	AgentName       string
	DiagnosticsPath string
}

type sessionState struct {
	PromptCount         int
	ConfigOptions       []acpsdk.SessionConfigOption
	activePromptCancel  context.CancelFunc
	promptStarting      bool
	pendingPromptCancel bool
}

type mockAgent struct {
	conn            *acpsdk.AgentSideConnection
	agent           acpmock.AgentFixture
	configTemplate  []acpsdk.SessionConfigOption
	diagnosticsPath string
	lifecycleCtx    context.Context
	cancelLifecycle context.CancelFunc

	mu          sync.Mutex
	sessions    map[string]*sessionState
	nextSession int
	asyncWG     sync.WaitGroup
}

type sandboxRunResult struct {
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

	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())

	agent := &mockAgent{
		agent:           agentFixture,
		configTemplate:  sessionConfigOptionsFromFixture(agentFixture.ConfigOptions),
		diagnosticsPath: strings.TrimSpace(args.DiagnosticsPath),
		lifecycleCtx:    lifecycleCtx,
		cancelLifecycle: cancelLifecycle,
		sessions:        make(map[string]*sessionState),
	}
	conn := acpsdk.NewAgentSideConnection(agent, os.Stdout, os.Stdin)
	agent.SetAgentConnection(conn)
	<-conn.Done()
	cancelLifecycle()
	agent.waitForAsyncControls()
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

func (a *mockAgent) Cancel(_ context.Context, params acpsdk.CancelNotification) error {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return errors.New("acpmock-driver: session id is required")
	}

	a.mu.Lock()
	session := a.sessions[sessionID]
	var cancel context.CancelFunc
	if session != nil {
		cancel = session.activePromptCancel
		if cancel == nil {
			session.pendingPromptCancel = true
		}
	}
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (a *mockAgent) CloseSession(
	context.Context,
	acpsdk.CloseSessionRequest,
) (acpsdk.CloseSessionResponse, error) {
	return acpsdk.CloseSessionResponse{}, nil
}

func (a *mockAgent) ListSessions(
	context.Context,
	acpsdk.ListSessionsRequest,
) (acpsdk.ListSessionsResponse, error) {
	return acpsdk.ListSessionsResponse{}, nil
}

func (a *mockAgent) ResumeSession(
	_ context.Context,
	params acpsdk.ResumeSessionRequest,
) (acpsdk.ResumeSessionResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acpsdk.ResumeSessionResponse{}, errors.New("acpmock-driver: session id is required")
	}
	return acpsdk.ResumeSessionResponse{
		ConfigOptions: a.sessionConfigOptions(sessionID),
	}, nil
}

func (a *mockAgent) NewSession(_ context.Context, params acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	a.mu.Lock()
	a.nextSession++
	sessionID := fmt.Sprintf("%s-session-%d", a.agent.Name, a.nextSession)
	a.sessions[sessionID] = &sessionState{
		ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
	}
	a.mu.Unlock()
	if err := a.writeSessionDiagnostics("session_new", sessionID, params.McpServers); err != nil {
		return acpsdk.NewSessionResponse{}, err
	}
	return acpsdk.NewSessionResponse{
		SessionId:     acpsdk.SessionId(sessionID),
		ConfigOptions: a.sessionConfigOptions(sessionID),
	}, nil
}

func (a *mockAgent) LoadSession(
	_ context.Context,
	params acpsdk.LoadSessionRequest,
) (acpsdk.LoadSessionResponse, error) {
	a.mu.Lock()
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID != "" && a.sessions[sessionID] == nil {
		a.sessions[sessionID] = &sessionState{
			ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
		}
	}
	configOptions := cloneSessionConfigOptions(a.sessions[sessionID].ConfigOptions)
	a.mu.Unlock()
	if err := a.writeSessionDiagnostics("session_load", sessionID, params.McpServers); err != nil {
		return acpsdk.LoadSessionResponse{}, err
	}
	return acpsdk.LoadSessionResponse{ConfigOptions: configOptions}, nil
}

func (a *mockAgent) writeSessionDiagnostics(
	event string,
	sessionID string,
	servers []acpsdk.McpServer,
) error {
	if len(servers) == 0 {
		return nil
	}
	return a.writeDiagnostics(acpmock.DiagnosticsRecord{
		AgentName:      a.agent.Name,
		SessionID:      sessionID,
		LifecycleEvent: event,
		MCPServers:     append([]acpsdk.McpServer(nil), servers...),
	})
}

func (a *mockAgent) SetSessionMode(
	context.Context,
	acpsdk.SetSessionModeRequest,
) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func (a *mockAgent) SetSessionConfigOption(
	_ context.Context,
	request acpsdk.SetSessionConfigOptionRequest,
) (acpsdk.SetSessionConfigOptionResponse, error) {
	if request.ValueId == nil {
		return acpsdk.SetSessionConfigOptionResponse{}, errors.New(
			"acpmock-driver: only value-id session config options are supported",
		)
	}
	if err := a.setConfigOptionValue(
		string(request.ValueId.SessionId),
		string(request.ValueId.ConfigId),
		string(request.ValueId.Value),
	); err != nil {
		return acpsdk.SetSessionConfigOptionResponse{}, err
	}
	return acpsdk.SetSessionConfigOptionResponse{
		ConfigOptions: a.sessionConfigOptions(string(request.ValueId.SessionId)),
	}, nil
}

func (a *mockAgent) UnstableSetSessionModel(
	context.Context,
	acpsdk.UnstableSetSessionModelRequest,
) (acpsdk.UnstableSetSessionModelResponse, error) {
	return acpsdk.UnstableSetSessionModelResponse{}, nil
}

func sessionConfigOptionsFromFixture(
	options []acpmock.SessionConfigOptionFixture,
) []acpsdk.SessionConfigOption {
	if len(options) == 0 {
		return nil
	}
	result := make([]acpsdk.SessionConfigOption, 0, len(options))
	for _, option := range options {
		values := make(acpsdk.SessionConfigSelectOptionsUngrouped, 0, len(option.Values))
		for _, value := range option.Values {
			label := strings.TrimSpace(value.Label)
			if label == "" {
				label = strings.TrimSpace(value.Value)
			}
			values = append(values, acpsdk.SessionConfigSelectOption{
				Name:  label,
				Value: acpsdk.SessionConfigValueId(strings.TrimSpace(value.Value)),
			})
		}
		result = append(result, acpsdk.SessionConfigOption{
			Select: &acpsdk.SessionConfigOptionSelect{
				Id:           acpsdk.SessionConfigId(strings.TrimSpace(option.ID)),
				Name:         strings.TrimSpace(option.Name),
				CurrentValue: acpsdk.SessionConfigValueId(strings.TrimSpace(option.Current)),
				Options: acpsdk.SessionConfigSelectOptions{
					Ungrouped: &values,
				},
				Type: "select",
			},
		})
	}
	return result
}

func (a *mockAgent) setConfigOptionValue(sessionID string, configID string, value string) error {
	trimmedSessionID := strings.TrimSpace(sessionID)
	trimmedConfigID := strings.TrimSpace(configID)
	trimmedValue := strings.TrimSpace(value)
	if trimmedSessionID == "" {
		return errors.New("acpmock-driver: session id is required")
	}
	if trimmedConfigID == "" {
		return errors.New("acpmock-driver: session config option id is required")
	}
	if trimmedValue == "" {
		return errors.New("acpmock-driver: session config option value is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.sessions[trimmedSessionID]
	if session == nil {
		session = &sessionState{
			ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
		}
		a.sessions[trimmedSessionID] = session
	}
	for idx := range session.ConfigOptions {
		option := session.ConfigOptions[idx].Select
		if option == nil || string(option.Id) != trimmedConfigID {
			continue
		}
		if option.Options.Ungrouped == nil {
			return fmt.Errorf("acpmock-driver: config option %q has no selectable values", trimmedConfigID)
		}
		for _, candidate := range *option.Options.Ungrouped {
			if string(candidate.Value) == trimmedValue {
				option.CurrentValue = acpsdk.SessionConfigValueId(trimmedValue)
				return nil
			}
		}
		return fmt.Errorf(
			"acpmock-driver: config option %q value %q is not available",
			trimmedConfigID,
			trimmedValue,
		)
	}
	return fmt.Errorf("acpmock-driver: config option %q is not available", trimmedConfigID)
}

func (a *mockAgent) ensureSessionState(sessionID string) *sessionState {
	trimmedSessionID := strings.TrimSpace(sessionID)
	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.sessions[trimmedSessionID]
	if session == nil {
		session = &sessionState{
			ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
		}
		a.sessions[trimmedSessionID] = session
	}
	return session
}

func (a *mockAgent) sessionConfigOptions(sessionID string) []acpsdk.SessionConfigOption {
	session := a.ensureSessionState(sessionID)
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneSessionConfigOptions(session.ConfigOptions)
}

func cloneSessionConfigOptions(options []acpsdk.SessionConfigOption) []acpsdk.SessionConfigOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]acpsdk.SessionConfigOption, 0, len(options))
	for _, option := range options {
		if option.Select != nil {
			selectCopy := *option.Select
			if option.Select.Options.Ungrouped != nil {
				values := append(acpsdk.SessionConfigSelectOptionsUngrouped(nil), (*option.Select.Options.Ungrouped)...)
				selectCopy.Options.Ungrouped = &values
			}
			cloned = append(cloned, acpsdk.SessionConfigOption{Select: &selectCopy})
			continue
		}
		if option.Boolean != nil {
			booleanCopy := *option.Boolean
			cloned = append(cloned, acpsdk.SessionConfigOption{Boolean: &booleanCopy})
		}
	}
	return cloned
}

func (a *mockAgent) Prompt(ctx context.Context, params acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acpsdk.PromptResponse{}, errors.New("sessionId is required")
	}
	a.beginPromptRegistration(sessionID)
	defer a.clearPromptCancel(sessionID)

	promptMeta, err := decodePromptMeta(params.Meta)
	if err != nil {
		return acpsdk.PromptResponse{}, err
	}

	prompt := extractPromptText(params.Prompt)
	turn, occurrence, err := a.selectTurn(sessionID, prompt, promptMeta)
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

	promptCtx, cancelPrompt := context.WithCancel(ctx)
	defer cancelPrompt()
	a.registerPromptCancel(sessionID, cancelPrompt)

	for _, step := range turn.Steps {
		entry, execErr := a.executeStep(promptCtx, acpsdk.SessionId(sessionID), step)
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

func (a *mockAgent) beginPromptRegistration(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.sessions[sessionID]
	if session == nil {
		session = &sessionState{
			ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
		}
		a.sessions[sessionID] = session
	}
	session.promptStarting = true
}

func (a *mockAgent) registerPromptCancel(sessionID string, cancel context.CancelFunc) {
	shouldCancel := false
	a.mu.Lock()
	session := a.sessions[sessionID]
	if session == nil {
		session = &sessionState{
			ConfigOptions: cloneSessionConfigOptions(a.configTemplate),
		}
		a.sessions[sessionID] = session
	}
	session.activePromptCancel = cancel
	session.promptStarting = false
	if session.pendingPromptCancel {
		session.pendingPromptCancel = false
		shouldCancel = true
	}
	a.mu.Unlock()
	if shouldCancel {
		cancel()
	}
}

func (a *mockAgent) clearPromptCancel(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if session := a.sessions[sessionID]; session != nil {
		session.activePromptCancel = nil
		session.promptStarting = false
		session.pendingPromptCancel = false
	}
}

func (a *mockAgent) selectTurn(
	sessionID string,
	prompt string,
	promptMeta acp.PromptMeta,
) (acpmock.TurnFixture, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	session := a.sessions[sessionID]
	if session == nil {
		session = &sessionState{}
		a.sessions[sessionID] = session
	}
	occurrence := session.PromptCount + 1
	turn, err := a.agent.SelectTurn(prompt, occurrence, promptMeta)
	if err != nil {
		return acpmock.TurnFixture{}, occurrence, err
	}
	session.PromptCount = occurrence
	return turn, occurrence, nil
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
	case acpmock.StepKindSandbox:
		return a.executeSandboxCommand(ctx, sessionID, step)
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
		ToolCall: acpsdk.ToolCallUpdate{
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

func (a *mockAgent) executeSandboxCommand(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) (acpmock.DiagnosticsStep, error) {
	toolCallID, title := sandboxDescriptor(step)
	if err := a.startSandboxToolCall(ctx, sessionID, step, toolCallID, title); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	result := a.runSandboxCommand(ctx, sessionID, step)
	if expected := strings.TrimSpace(step.ExpectErrorContains); expected != "" {
		return a.finishSandboxFailure(ctx, sessionID, step, toolCallID, title, result, expected)
	}
	if err := validateSandboxResult(step, result); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	if err := a.finishSandboxSuccess(ctx, sessionID, step, toolCallID, title, result); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}

	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindSandbox,
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
		lifecycleCtx := a.lifecycleContext()
		a.asyncWG.Add(1)
		go func(promptCtx context.Context, lifetimeCtx context.Context, control acpmock.DriverControlStep) {
			defer a.asyncWG.Done()

			if err := waitDriverControlDelay(promptCtx, lifetimeCtx, control.DelayMS); err != nil {
				return
			}
			if err := a.performDriverControl(promptCtx, control); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "acpmock async driver_control %s error: %v\n", control.Action, err)
			}
		}(ctx, lifecycleCtx, control)
		return diagnostics, nil
	}
	if err := waitDriverControlDelay(ctx, a.lifecycleContext(), control.DelayMS); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	return diagnostics, a.performDriverControl(ctx, control)
}

func waitDriverControlDelay(promptCtx context.Context, lifetimeCtx context.Context, delayMS int) error {
	if err := driverControlContextErr(promptCtx, lifetimeCtx); err != nil {
		return err
	}
	if delayMS <= 0 {
		return nil
	}
	timer := time.NewTimer(time.Duration(delayMS) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C:
		return driverControlContextErr(promptCtx, lifetimeCtx)
	case <-contextDone(promptCtx):
		return promptCtx.Err()
	case <-contextDone(lifetimeCtx):
		return lifetimeCtx.Err()
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

func sandboxDescriptor(step acpmock.Step) (string, string) {
	toolCallID := strings.TrimSpace(step.ToolCallID)
	title := strings.TrimSpace(step.Title)
	if title == "" {
		title = strings.TrimSpace(step.Command)
	}
	if title == "" {
		title = "sandbox command"
	}
	return toolCallID, title
}

func (a *mockAgent) startSandboxToolCall(
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

func (a *mockAgent) runSandboxCommand(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
) sandboxRunResult {
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
		return sandboxRunResult{ObservedError: err.Error()}
	}

	result := sandboxRunResult{}
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

func (a *mockAgent) finishSandboxFailure(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
	result sandboxRunResult,
	expected string,
) (acpmock.DiagnosticsStep, error) {
	if result.ObservedError == "" || !strings.Contains(result.ObservedError, expected) {
		return acpmock.DiagnosticsStep{}, fmt.Errorf(
			"sandbox command error %s did not include %s",
			result.ObservedError,
			expected,
		)
	}
	if err := a.emitSandboxFailure(ctx, sessionID, step, toolCallID, title, result.ObservedError); err != nil {
		return acpmock.DiagnosticsStep{}, err
	}
	return acpmock.DiagnosticsStep{
		Kind:       acpmock.StepKindSandbox,
		ToolCallID: toolCallID,
		Command:    strings.TrimSpace(step.Command),
		Args:       append([]string(nil), step.Args...),
		Error:      result.ObservedError,
	}, nil
}

func validateSandboxResult(step acpmock.Step, result sandboxRunResult) error {
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
				"sandbox exit code %s did not match expected %d",
				got,
				*step.ExpectExitCode,
			)
		}
	}
	if expected := strings.TrimSpace(step.ExpectOutputContains); expected != "" &&
		!strings.Contains(result.Output, expected) {
		return fmt.Errorf("sandbox output %q did not include %s", result.Output, expected)
	}
	return nil
}

func (a *mockAgent) finishSandboxSuccess(
	ctx context.Context,
	sessionID acpsdk.SessionId,
	step acpmock.Step,
	toolCallID string,
	title string,
	result sandboxRunResult,
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

func (a *mockAgent) emitSandboxFailure(
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
	case <-contextDone(ctx):
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func contextDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}

func driverControlContextErr(promptCtx context.Context, lifetimeCtx context.Context) error {
	if promptCtx != nil {
		if err := promptCtx.Err(); err != nil {
			return err
		}
	}
	if lifetimeCtx != nil {
		if err := lifetimeCtx.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (a *mockAgent) lifecycleContext() context.Context {
	if a == nil || a.lifecycleCtx == nil {
		return context.Background()
	}
	return a.lifecycleCtx
}

func (a *mockAgent) waitForAsyncControls() {
	if a == nil {
		return
	}
	a.asyncWG.Wait()
}
