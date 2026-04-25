package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	exec "os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	"golang.org/x/sys/execabs"

	"github.com/pedronauck/agh/internal/toolruntime"
)

const (
	defaultTerminalOutputLimit = 64 * 1024
	networkCommandName         = "network"
)

type wireSessionNotification struct {
	SessionID acpsdk.SessionId `json:"sessionId"`
	Update    json.RawMessage  `json:"update"`
}

type wireSessionUpdateEnvelope struct {
	SessionUpdate string `json:"sessionUpdate"`
}

type wirePromptResponse struct {
	StopReason acpsdk.StopReason `json:"stopReason"`
	Usage      *wireUsage        `json:"usage,omitempty"`
}

// wireNewSessionRequest keeps the workspace extension on the top-level
// session/new payload. The workspace techspec requires the JSON-RPC field name
// `additional_dirs`, even though the upstream ACP SDK does not model it yet.
type wireNewSessionRequest struct {
	Meta           any                `json:"_meta,omitempty"`
	Cwd            string             `json:"cwd"`
	McpServers     []acpsdk.McpServer `json:"mcpServers"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
}

// wireLoadSessionRequest mirrors session/load with the same top-level
// `additional_dirs` field name required by the workspace techspec.
type wireLoadSessionRequest struct {
	Meta           any                `json:"_meta,omitempty"`
	Cwd            string             `json:"cwd"`
	McpServers     []acpsdk.McpServer `json:"mcpServers"`
	AdditionalDirs []string           `json:"additional_dirs,omitempty"`
	SessionID      acpsdk.SessionId   `json:"sessionId"`
}

type wireUsage struct {
	InputTokens      *int64 `json:"inputTokens,omitempty"`
	OutputTokens     *int64 `json:"outputTokens,omitempty"`
	TotalTokens      *int64 `json:"totalTokens,omitempty"`
	ThoughtTokens    *int64 `json:"thoughtTokens,omitempty"`
	CacheReadTokens  *int64 `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens *int64 `json:"cacheWriteTokens,omitempty"`
}

type wireUsageUpdate struct {
	SessionUpdate string    `json:"sessionUpdate"`
	Used          *int64    `json:"used,omitempty"`
	Size          *int64    `json:"size,omitempty"`
	Cost          *wireCost `json:"cost,omitempty"`
}

type wireCost struct {
	Amount   *float64 `json:"amount,omitempty"`
	Currency *string  `json:"currency,omitempty"`
}

type terminalManager struct {
	ctx      context.Context
	logger   *slog.Logger
	registry *toolruntime.Registry

	nextID atomic.Uint64

	mu        sync.RWMutex
	terminals map[string]*managedTerminal
}

type managedTerminal struct {
	id string

	cmd           *exec.Cmd
	processRecord *toolruntime.Handle

	networkOwned   bool
	ownerSessionID string
	ownerTurnID    string

	mu         sync.RWMutex
	output     []byte
	truncated  bool
	exitStatus *acpsdk.TerminalExitStatus
	done       chan struct{}
}

type terminalOwnership struct {
	networkOwned   bool
	ownerSessionID string
	ownerTurnID    string
}

type terminalOutputWriter struct {
	terminal *managedTerminal
}

func (p *AgentProcess) handleInbound(
	ctx context.Context,
	method string,
	params json.RawMessage,
) (any, *acpsdk.RequestError) {
	if method == acpsdk.ClientMethodSessionUpdate {
		if err := p.handleSessionUpdate(params); err != nil {
			return nil, requestError(err)
		}
		return nil, nil
	}

	handlers := map[string]func(context.Context, json.RawMessage) (any, *acpsdk.RequestError){
		acpsdk.ClientMethodFsReadTextFile: func(
			ctx context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequest(ctx, params, p.handleReadTextFile)
		},
		acpsdk.ClientMethodFsWriteTextFile: func(
			ctx context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequest(ctx, params, p.handleWriteTextFile)
		},
		acpsdk.ClientMethodSessionRequestPermission: func(
			ctx context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequest(ctx, params, p.handleRequestPermission)
		},
		acpsdk.ClientMethodTerminalCreate: func(
			ctx context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequest(ctx, params, p.handleCreateTerminal)
		},
		acpsdk.ClientMethodTerminalKill: func(
			_ context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequestNoContext(params, p.handleKillTerminal)
		},
		acpsdk.ClientMethodTerminalOutput: func(
			_ context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequestNoContext(params, p.handleTerminalOutput)
		},
		acpsdk.ClientMethodTerminalWaitForExit: func(
			ctx context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequest(ctx, params, p.handleWaitForTerminalExit)
		},
		acpsdk.ClientMethodTerminalRelease: func(
			_ context.Context,
			params json.RawMessage,
		) (any, *acpsdk.RequestError) {
			return handleInboundRequestNoContext(params, p.handleReleaseTerminal)
		},
	}

	handler, ok := handlers[method]
	if !ok {
		return nil, acpsdk.NewMethodNotFound(method)
	}
	return handler(ctx, params)
}

func handleInboundRequest[Req any, Resp any](
	ctx context.Context,
	params json.RawMessage,
	fn func(context.Context, Req) (Resp, error),
) (any, *acpsdk.RequestError) {
	var request Req
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
	}

	response, err := fn(ctx, request)
	if err != nil {
		return nil, requestError(err)
	}
	return response, nil
}

func handleInboundRequestNoContext[Req any, Resp any](
	params json.RawMessage,
	fn func(Req) (Resp, error),
) (any, *acpsdk.RequestError) {
	var request Req
	if err := json.Unmarshal(params, &request); err != nil {
		return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
	}

	response, err := fn(request)
	if err != nil {
		return nil, requestError(err)
	}
	return response, nil
}

func (p *AgentProcess) handleReadTextFile(
	ctx context.Context,
	request acpsdk.ReadTextFileRequest,
) (acpsdk.ReadTextFileResponse, error) {
	content, err := p.toolHostOrDefault().ReadTextFile(ctx, request.Path)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, err
	}
	return acpsdk.ReadTextFileResponse{Content: sliceLines(content, request.Line, request.Limit)}, nil
}

func (p *AgentProcess) handleWriteTextFile(
	ctx context.Context,
	request acpsdk.WriteTextFileRequest,
) (acpsdk.WriteTextFileResponse, error) {
	if p.isNetworkTurn() {
		return acpsdk.WriteTextFileResponse{}, ErrToolBlockedForNetworkTurn
	}
	if err := p.toolHostOrDefault().WriteTextFile(ctx, request.Path, request.Content); err != nil {
		return acpsdk.WriteTextFileResponse{}, err
	}
	return acpsdk.WriteTextFileResponse{}, nil
}

func (p *AgentProcess) handleRequestPermission(
	ctx context.Context,
	request acpsdk.RequestPermissionRequest,
) (acpsdk.RequestPermissionResponse, error) {
	turnID := p.activeTurnID()
	resource := ""
	if request.ToolCall.Title != nil {
		resource = *request.ToolCall.Title
	}
	if len(request.ToolCall.Locations) > 0 {
		resource = request.ToolCall.Locations[0].Path
	}
	title := ""
	if request.ToolCall.Title != nil {
		title = *request.ToolCall.Title
	}

	decision, interactive := p.toolHostOrDefault().PermissionDecision(request)
	sessionID := string(request.SessionId)
	toolCallID := strings.TrimSpace(string(request.ToolCall.ToolCallId))

	if !interactive {
		requestID := p.nextPermissionRequestID(turnID, request)
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decision)
		raw := buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	}

	requestID, pending := p.registerPendingPermission(turnID, request)
	defer p.clearPendingPermission(requestID)
	raw := buildPermissionEventRaw(requestID, decisionPending, request)
	p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, "", raw)

	timer := time.NewTimer(p.permissionTimeoutOrDefault())
	defer timer.Stop()

	select {
	case resolvedDecision := <-pending.response:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, resolvedDecision)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	case <-timer.C:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decisionRejectOnce)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPermissionEvent(sessionID, turnID, requestID, title, toolCallID, resource, appliedDecision, raw)
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	case <-ctx.Done():
		return acpsdk.RequestPermissionResponse{
			Outcome: acpsdk.NewRequestPermissionOutcomeCancelled(),
		}, nil
	}
}

func (p *AgentProcess) handleSessionUpdate(params json.RawMessage) error {
	var raw wireSessionNotification
	if err := json.Unmarshal(params, &raw); err != nil {
		return fmt.Errorf("acp: decode session/update notification: %w", err)
	}
	var envelope wireSessionUpdateEnvelope
	if err := json.Unmarshal(raw.Update, &envelope); err != nil {
		return fmt.Errorf("acp: decode session/update envelope: %w", err)
	}

	if envelope.SessionUpdate == "usage_update" {
		var update wireUsageUpdate
		if err := json.Unmarshal(raw.Update, &update); err != nil {
			return fmt.Errorf("acp: decode usage_update: %w", err)
		}
		usage := tokenUsageFromUsageUpdate(p.activeTurnID(), update)
		if !usage.IsZero() {
			merged := p.mergePromptUsage(usage)
			p.emitPromptEvent(AgentEvent{
				Type:      EventTypeUsage,
				SessionID: string(raw.SessionID),
				TurnID:    merged.TurnID,
				Timestamp: usage.Timestamp,
				Usage:     &merged,
				Raw:       CloneRawMessage(raw.Update),
			})
		}
		return nil
	}

	var notification acpsdk.SessionNotification
	if err := json.Unmarshal(params, &notification); err != nil {
		return fmt.Errorf("acp: decode session notification: %w", err)
	}

	event := translateSessionUpdate(notification, raw.Update, p.activeTurnID())
	p.emitPromptEvent(event)
	return nil
}

func (p *AgentProcess) emitPermissionEvent(
	sessionID string,
	turnID string,
	requestID string,
	title string,
	toolCallID string,
	resource string,
	decision permissionDecision,
	raw json.RawMessage,
) {
	p.emitPromptEvent(AgentEvent{
		Type:       EventTypePermission,
		SessionID:  sessionID,
		TurnID:     turnID,
		RequestID:  requestID,
		Timestamp:  timeNowUTC(),
		Title:      title,
		ToolCallID: toolCallID,
		Action:     string(permissionRequestToolGrant),
		Resource:   resource,
		Decision:   string(decision),
		Raw:        CloneRawMessage(raw),
	})
}

func (p *AgentProcess) handleCreateTerminal(
	ctx context.Context,
	request acpsdk.CreateTerminalRequest,
) (acpsdk.CreateTerminalResponse, error) {
	ownership := terminalOwnership{
		ownerSessionID: p.SessionID,
		ownerTurnID:    p.activeTurnID(),
	}
	if p.isNetworkTurn() {
		argv, err := terminalArgv(request)
		if err != nil {
			return acpsdk.CreateTerminalResponse{}, fmt.Errorf("%w: %s", ErrToolBlockedForNetworkTurn, err)
		}
		if !isAllowedNetworkTerminalArgv(argv) {
			return acpsdk.CreateTerminalResponse{}, ErrToolBlockedForNetworkTurn
		}
		ownership = terminalOwnership{
			networkOwned:   true,
			ownerSessionID: p.SessionID,
			ownerTurnID:    p.activeTurnID(),
		}
	}

	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		return localHost.createTerminal(ctx, request, ownership)
	}
	response, err := host.CreateTerminal(ctx, request)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	p.recordTerminalOwnership(response.TerminalId, ownership)
	if err := p.registerExternalTerminalProcess(ctx, host, response.TerminalId, request, ownership); err != nil {
		if killErr := host.KillTerminal(response.TerminalId); killErr != nil {
			slog.Default().Warn(
				"acp: cleanup unregistered terminal",
				"terminal_id", response.TerminalId,
				"error", killErr,
			)
		}
		return acpsdk.CreateTerminalResponse{}, err
	}
	return response, nil
}

func (p *AgentProcess) recordTerminalOwnership(id string, ownership terminalOwnership) {
	if strings.TrimSpace(id) == "" || !ownership.networkOwned {
		return
	}

	p.terminalOwnershipMu.Lock()
	defer p.terminalOwnershipMu.Unlock()
	if p.terminalOwnership == nil {
		p.terminalOwnership = make(map[string]terminalOwnership)
	}
	p.terminalOwnership[id] = ownership
}

func (p *AgentProcess) registerExternalTerminalProcess(
	ctx context.Context,
	host ToolHost,
	id string,
	request acpsdk.CreateTerminalRequest,
	ownership terminalOwnership,
) error {
	if p.processRegistry == nil || strings.TrimSpace(id) == "" {
		return nil
	}
	argv, err := terminalArgv(request)
	if err != nil {
		return err
	}
	registerCtx := ctx
	if registerCtx == nil {
		registerCtx = context.Background()
	}
	cwd := p.Cwd
	if request.Cwd != nil {
		cwd = *request.Cwd
	}
	var handle *toolruntime.Handle
	handle, err = p.processRegistry.Register(registerCtx, toolruntime.RegisterConfig{
		Source: toolruntime.ProcessSourceEnvironmentTerminal,
		Owner: toolruntime.ProcessOwner{
			SessionID:  ownership.ownerSessionID,
			TurnID:     ownership.ownerTurnID,
			TerminalID: id,
		},
		Command: argv[0],
		Args:    argv[1:],
		Cwd:     cwd,
		Interrupt: func(callbackCtx context.Context, _ toolruntime.ProcessRecord) error {
			if killErr := host.KillTerminal(id); killErr != nil {
				return killErr
			}
			if handle != nil {
				return handle.Complete(
					context.WithoutCancel(callbackCtx),
					toolruntime.ProcessCompletion{Err: errors.New("terminal interrupted")},
				)
			}
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("acp: register terminal process %q: %w", id, err)
	}

	p.terminalProcessMu.Lock()
	defer p.terminalProcessMu.Unlock()
	if p.terminalProcesses == nil {
		p.terminalProcesses = make(map[string]*toolruntime.Handle)
	}
	p.terminalProcesses[id] = handle
	return nil
}

func (p *AgentProcess) completeExternalTerminalProcess(
	ctx context.Context,
	id string,
	completion toolruntime.ProcessCompletion,
) {
	handle := p.takeExternalTerminalProcess(id)
	if handle == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := handle.Complete(ctx, completion); err != nil {
		slog.Default().Warn("acp: complete terminal process record", "terminal_id", id, "error", err)
	}
}

func (p *AgentProcess) takeExternalTerminalProcess(id string) *toolruntime.Handle {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	p.terminalProcessMu.Lock()
	defer p.terminalProcessMu.Unlock()
	if p.terminalProcesses == nil {
		return nil
	}
	handle := p.terminalProcesses[id]
	delete(p.terminalProcesses, id)
	return handle
}

func (p *AgentProcess) handleKillTerminal(
	request acpsdk.KillTerminalCommandRequest,
) (acpsdk.KillTerminalCommandResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, false); err != nil {
		return acpsdk.KillTerminalCommandResponse{}, err
	}
	if err := p.toolHostOrDefault().KillTerminal(request.TerminalId); err != nil {
		return acpsdk.KillTerminalCommandResponse{}, err
	}
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{Err: errors.New("terminal killed")},
	)
	return acpsdk.KillTerminalCommandResponse{}, nil
}

func (p *AgentProcess) handleTerminalOutput(
	request acpsdk.TerminalOutputRequest,
) (acpsdk.TerminalOutputResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, true); err != nil {
		return acpsdk.TerminalOutputResponse{}, err
	}
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		output, truncated, exitStatus, err := localHost.terminalOutputStatus(request.TerminalId)
		if err != nil {
			return acpsdk.TerminalOutputResponse{}, err
		}
		return acpsdk.TerminalOutputResponse{
			Output:     output,
			Truncated:  truncated,
			ExitStatus: exitStatus,
		}, nil
	}
	output, err := host.TerminalOutput(request.TerminalId)
	if err != nil {
		return acpsdk.TerminalOutputResponse{}, err
	}
	return acpsdk.TerminalOutputResponse{
		Output: output,
	}, nil
}

func (p *AgentProcess) handleWaitForTerminalExit(
	ctx context.Context,
	request acpsdk.WaitForTerminalExitRequest,
) (acpsdk.WaitForTerminalExitResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, true); err != nil {
		return acpsdk.WaitForTerminalExitResponse{}, err
	}
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		exitStatus, err := localHost.waitForTerminalExitStatus(ctx, request.TerminalId)
		if err != nil {
			return acpsdk.WaitForTerminalExitResponse{}, err
		}
		if exitStatus == nil {
			return acpsdk.WaitForTerminalExitResponse{}, nil
		}
		return acpsdk.WaitForTerminalExitResponse{
			ExitCode: exitStatus.ExitCode,
			Signal:   exitStatus.Signal,
		}, nil
	}
	exitCode, err := host.WaitForTerminalExit(ctx, request.TerminalId)
	if err != nil {
		return acpsdk.WaitForTerminalExitResponse{}, err
	}
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{ExitCode: acpsdk.Ptr(exitCode)},
	)
	return acpsdk.WaitForTerminalExitResponse{
		ExitCode: acpsdk.Ptr(exitCode),
	}, nil
}

func (p *AgentProcess) handleReleaseTerminal(
	request acpsdk.ReleaseTerminalRequest,
) (acpsdk.ReleaseTerminalResponse, error) {
	if err := p.ensureNetworkTurnTerminalAccess(request.TerminalId, false); err != nil {
		return acpsdk.ReleaseTerminalResponse{}, err
	}
	if err := p.toolHostOrDefault().ReleaseTerminal(request.TerminalId); err != nil {
		return acpsdk.ReleaseTerminalResponse{}, err
	}
	p.completeExternalTerminalProcess(
		context.Background(),
		request.TerminalId,
		toolruntime.ProcessCompletion{Error: "terminal released"},
	)
	return acpsdk.ReleaseTerminalResponse{}, nil
}

func (p *AgentProcess) toolHostOrDefault() ToolHost {
	p.toolHostMu.Lock()
	defer p.toolHostMu.Unlock()

	if p.toolHost != nil {
		return p.toolHost
	}
	procCtx := p.processCtx
	if procCtx == nil {
		procCtx = context.Background()
	}
	host := newLocalToolHostFromPolicy(
		procCtx,
		p.Cwd,
		p.permissions,
		slog.Default(),
		WithLocalProcessRegistry(p.processRegistry),
	)
	if p.terminals != nil {
		host.terminals = p.terminals
	} else {
		p.terminals = host.terminals
	}
	p.toolHost = host
	return host
}

func newTerminalManager(
	ctx context.Context,
	logger *slog.Logger,
	registries ...*toolruntime.Registry,
) *terminalManager {
	var registry *toolruntime.Registry
	if len(registries) > 0 {
		registry = registries[0]
	}
	return &terminalManager{
		ctx:       ctx,
		logger:    logger,
		registry:  registry,
		terminals: make(map[string]*managedTerminal),
	}
}

func (m *terminalManager) create(
	ctx context.Context,
	cwd string,
	request acpsdk.CreateTerminalRequest,
	ownership terminalOwnership,
) (acpsdk.CreateTerminalResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	argv, err := terminalArgv(request)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}

	cmd, err := newManagedTerminalCommand(argv)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}
	configureManagedCommand(cmd)
	cmd.Dir = cwd
	cmd.Env = mergeCommandEnv(os.Environ(), request.Env)

	term := &managedTerminal{
		id:             fmt.Sprintf("term-%d", m.nextID.Add(1)),
		cmd:            cmd,
		networkOwned:   ownership.networkOwned,
		ownerSessionID: strings.TrimSpace(ownership.ownerSessionID),
		ownerTurnID:    strings.TrimSpace(ownership.ownerTurnID),
		done:           make(chan struct{}),
	}
	writer := &terminalOutputWriter{terminal: term}
	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Start(); err != nil {
		return acpsdk.CreateTerminalResponse{}, fmt.Errorf("acp: start terminal command %q: %w", argv[0], err)
	}
	if err := m.registerTerminalProcess(ctx, term, cwd, argv); err != nil {
		if killErr := killManagedProcess(cmd); killErr != nil {
			m.logTerminalKillError(term.id, "registration cleanup", killErr)
		}
		if waitErr := cmd.Wait(); waitErr != nil {
			m.logTerminalKillError(term.id, "registration wait cleanup", waitErr)
		}
		return acpsdk.CreateTerminalResponse{}, err
	}
	if m.ctx != nil {
		watchTerminalShutdown(m.ctx, term.done, func() {
			term.checkpointInterrupting(context.Background(), "manager shutdown")
			if err := killManagedProcess(cmd); err != nil {
				m.logTerminalKillError(term.id, "manager shutdown", err)
			}
		})
	}

	m.mu.Lock()
	m.terminals[term.id] = term
	m.mu.Unlock()

	waitCtx := context.WithoutCancel(ctx)
	go term.wait(waitCtx)

	return acpsdk.CreateTerminalResponse{TerminalId: term.id}, nil
}

func (m *terminalManager) registerTerminalProcess(
	ctx context.Context,
	term *managedTerminal,
	cwd string,
	argv []string,
) error {
	if m.registry == nil || term == nil || term.cmd == nil || term.cmd.Process == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	handle, err := m.registry.Register(ctx, toolruntime.RegisterConfig{
		Source: toolruntime.ProcessSourceACPTerminal,
		Owner: toolruntime.ProcessOwner{
			SessionID:  term.ownerSessionID,
			TurnID:     term.ownerTurnID,
			TerminalID: term.id,
		},
		PID:            term.cmd.Process.Pid,
		ProcessGroupID: term.cmd.Process.Pid,
		Command:        argv[0],
		Args:           argv[1:],
		Cwd:            cwd,
		Interrupt: func(_ context.Context, _ toolruntime.ProcessRecord) error {
			return terminateManagedProcess(term.cmd)
		},
	})
	if err != nil {
		return fmt.Errorf("acp: register terminal process %q: %w", term.id, err)
	}
	term.processRecord = handle
	return nil
}

func newManagedTerminalCommand(argv []string) (*exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, errors.New("acp: terminal command is required")
	}

	resolvedPath, err := execabs.LookPath(argv[0])
	if err != nil {
		return nil, fmt.Errorf("acp: resolve terminal executable %q: %w", argv[0], err)
	}

	commandArgs := make([]string, 0, len(argv))
	commandArgs = append(commandArgs, resolvedPath)
	commandArgs = append(commandArgs, argv[1:]...)
	return &exec.Cmd{
		Path: resolvedPath,
		Args: commandArgs,
	}, nil
}

func (m *terminalManager) kill(id string) error {
	term, err := m.lookup(id)
	if err != nil {
		return err
	}
	term.checkpointInterrupting(context.Background(), "terminal killed")
	if err := killManagedProcess(term.cmd); err != nil {
		return fmt.Errorf("acp: kill terminal %q: %w", id, err)
	}
	return nil
}

func (m *terminalManager) output(id string) (string, bool, *acpsdk.TerminalExitStatus, error) {
	term, err := m.lookup(id)
	if err != nil {
		return "", false, nil, err
	}
	output, truncated, exitStatus := term.snapshot()
	return output, truncated, exitStatus, nil
}

func (m *terminalManager) wait(ctx context.Context, id string) (*acpsdk.TerminalExitStatus, error) {
	term, err := m.lookup(id)
	if err != nil {
		return nil, err
	}
	select {
	case <-term.done:
		_, _, exitStatus := term.snapshot()
		return exitStatus, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *terminalManager) release(id string) error {
	term, err := m.lookup(id)
	if err != nil {
		return err
	}
	term.checkpointInterrupting(context.Background(), "terminal released")
	if killErr := killManagedProcess(term.cmd); killErr != nil {
		m.logTerminalKillError(id, "release", killErr)
	}
	m.mu.Lock()
	delete(m.terminals, id)
	m.mu.Unlock()
	return nil
}

func (m *terminalManager) closeAll() {
	m.mu.RLock()
	terminals := make([]*managedTerminal, 0, len(m.terminals))
	for _, terminal := range m.terminals {
		terminals = append(terminals, terminal)
	}
	m.mu.RUnlock()

	for _, terminal := range terminals {
		terminal.checkpointInterrupting(context.Background(), "terminal manager closing")
		if err := killManagedProcess(terminal.cmd); err != nil {
			m.logTerminalKillError(terminal.id, "close_all", err)
		}
	}
}

func (m *terminalManager) logTerminalKillError(id string, reason string, err error) {
	if err == nil || m.logger == nil {
		return
	}
	m.logger.Warn("acp: kill terminal", "terminal_id", id, "reason", reason, "error", err)
}

func (m *terminalManager) lookup(id string) (*managedTerminal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	term, ok := m.terminals[id]
	if !ok {
		return nil, fmt.Errorf("acp: terminal %q not found", id)
	}
	return term, nil
}

func (w *terminalOutputWriter) Write(p []byte) (int, error) {
	w.terminal.appendOutput(p)
	return len(p), nil
}

func (t *managedTerminal) appendOutput(p []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var truncated bool
	t.output, truncated = appendTerminalOutputWindow(t.output, p, defaultTerminalOutputLimit)
	if truncated {
		t.truncated = true
	}
}

func (t *managedTerminal) wait(ctx context.Context) {
	err := t.cmd.Wait()
	groupWaitErr := forceManagedProcessGroupExit(t.cmd, 250*time.Millisecond)
	exitStatus := &acpsdk.TerminalExitStatus{}
	if t.cmd.ProcessState != nil {
		exitCode := t.cmd.ProcessState.ExitCode()
		if exitCode >= 0 {
			exitStatus.ExitCode = acpsdk.Ptr(exitCode)
		}
	}
	if err != nil && exitStatus.ExitCode == nil {
		signalText := err.Error()
		exitStatus.Signal = &signalText
	}
	if groupWaitErr != nil && exitStatus.Signal == nil {
		signalText := groupWaitErr.Error()
		exitStatus.Signal = &signalText
	}

	t.mu.Lock()
	t.exitStatus = exitStatus
	t.mu.Unlock()
	t.completeProcess(ctx, exitStatus, err, groupWaitErr)
	close(t.done)
}

func (t *managedTerminal) checkpointInterrupting(ctx context.Context, reason string) {
	if t == nil || t.processRecord == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := t.processRecord.Checkpoint(ctx, toolruntime.ProcessCheckpoint{
		State: toolruntime.ProcessStateInterrupting,
		Error: reason,
	}); err != nil {
		slog.Default().Warn("acp: checkpoint terminal process record", "terminal_id", t.id, "error", err)
	}
}

func (t *managedTerminal) completeProcess(
	ctx context.Context,
	exitStatus *acpsdk.TerminalExitStatus,
	waitErr error,
	groupWaitErr error,
) {
	if t == nil || t.processRecord == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	completion := toolruntime.ProcessCompletion{}
	if exitStatus != nil && exitStatus.ExitCode != nil {
		completion.ExitCode = exitStatus.ExitCode
	}
	if waitErr != nil {
		completion.Err = waitErr
	} else if groupWaitErr != nil {
		completion.Err = groupWaitErr
	}
	if err := t.processRecord.Complete(ctx, completion); err != nil {
		slog.Default().Warn("acp: complete terminal process record", "terminal_id", t.id, "error", err)
	}
}

func (t *managedTerminal) snapshot() (string, bool, *acpsdk.TerminalExitStatus) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	output := string(append([]byte(nil), t.output...))
	var exitStatus *acpsdk.TerminalExitStatus
	if t.exitStatus != nil {
		copyStatus := *t.exitStatus
		exitStatus = &copyStatus
	}
	return output, t.truncated, exitStatus
}

func translateSessionUpdate(
	notification acpsdk.SessionNotification,
	rawUpdate json.RawMessage,
	turnID string,
) AgentEvent {
	event := AgentEvent{
		SessionID: string(notification.SessionId),
		TurnID:    turnID,
		Timestamp: timeNowUTC(),
		Raw:       CloneRawMessage(rawUpdate),
	}

	switch {
	case notification.Update.UserMessageChunk != nil:
		event.Type = EventTypeUserMessage
		event.Text = extractContentText(notification.Update.UserMessageChunk.Content)
	case notification.Update.AgentMessageChunk != nil:
		event.Type = EventTypeAgentMessage
		event.Text = extractContentText(notification.Update.AgentMessageChunk.Content)
	case notification.Update.AgentThoughtChunk != nil:
		event.Type = EventTypeThought
		event.Text = extractContentText(notification.Update.AgentThoughtChunk.Content)
	case notification.Update.ToolCall != nil:
		toolCall := notification.Update.ToolCall
		event.Type = EventTypeToolCall
		event.Title = toolCall.Title
		event.ToolCallID = string(toolCall.ToolCallId)
	case notification.Update.ToolCallUpdate != nil:
		toolUpdate := notification.Update.ToolCallUpdate
		event.ToolCallID = string(toolUpdate.ToolCallId)
		if toolUpdate.Title != nil {
			event.Title = *toolUpdate.Title
		}
		if toolUpdate.Status != nil &&
			(*toolUpdate.Status == acpsdk.ToolCallStatusCompleted || *toolUpdate.Status == acpsdk.ToolCallStatusFailed) {
			event.Type = EventTypeToolResult
		} else {
			event.Type = EventTypeToolCall
		}
	case notification.Update.Plan != nil:
		event.Type = EventTypePlan
	case notification.Update.AvailableCommandsUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = "available_commands_update"
	case notification.Update.CurrentModeUpdate != nil:
		event.Type = EventTypeSystem
		event.Title = "current_mode_update"
	default:
		event.Type = EventTypeSystem
	}

	return event
}

func tokenUsageFromPromptResponse(turnID string, usage *wireUsage) TokenUsage {
	if usage == nil {
		return TokenUsage{TurnID: turnID}
	}
	return TokenUsage{
		TurnID:           turnID,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
		ThoughtTokens:    usage.ThoughtTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		Timestamp:        timeNowUTC(),
	}
}

func tokenUsageFromUsageUpdate(turnID string, update wireUsageUpdate) TokenUsage {
	var amount *float64
	var currency *string
	if update.Cost != nil {
		amount = update.Cost.Amount
		currency = update.Cost.Currency
	}
	return TokenUsage{
		TurnID:       turnID,
		ContextUsed:  update.Used,
		ContextSize:  update.Size,
		CostAmount:   amount,
		CostCurrency: currency,
		Timestamp:    timeNowUTC(),
	}
}

func requestError(err error) *acpsdk.RequestError {
	if err == nil {
		return nil
	}
	var requestErr *acpsdk.RequestError
	if errors.As(err, &requestErr) {
		return requestErr
	}
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrPathOutsideWorkspace) ||
		errors.Is(err, ErrToolBlockedForNetworkTurn) {
		return acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
	}
	return acpsdk.NewInternalError(map[string]any{"error": err.Error()})
}

func terminalArgv(request acpsdk.CreateTerminalRequest) ([]string, error) {
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return nil, errors.New("acp: terminal command is required")
	}

	argv, err := shellquote.Split(command)
	if err != nil {
		return nil, fmt.Errorf("acp: parse terminal command %q: %w", request.Command, err)
	}
	if len(argv) == 0 {
		return nil, errors.New("acp: terminal command is required")
	}
	return append(argv, request.Args...), nil
}

func isAllowedNetworkTerminalArgv(argv []string) bool {
	if len(argv) < 3 || argv[0] != "agh" || argv[1] != networkCommandName {
		return false
	}

	switch argv[2] {
	case "send", "peers", "channels", "status", "inbox":
		return true
	default:
		return false
	}
}

func (p *AgentProcess) ensureNetworkTurnTerminalAccess(id string, requireSameTurn bool) error {
	if !p.isNetworkTurn() {
		return nil
	}

	ownership, err := p.lookupTerminalOwnership(id)
	if err != nil {
		return err
	}
	if !ownership.networkOwned {
		return ErrToolBlockedForNetworkTurn
	}
	if requireSameTurn && strings.TrimSpace(ownership.ownerTurnID) != p.activeTurnID() {
		return ErrToolBlockedForNetworkTurn
	}
	return nil
}

func (p *AgentProcess) lookupTerminalOwnership(id string) (terminalOwnership, error) {
	host := p.toolHostOrDefault()
	if localHost, ok := host.(*localToolHost); ok {
		return localHost.terminalOwnership(id)
	}

	p.terminalOwnershipMu.RLock()
	ownership, ok := p.terminalOwnership[id]
	p.terminalOwnershipMu.RUnlock()
	if !ok {
		return terminalOwnership{}, ErrToolBlockedForNetworkTurn
	}
	return ownership, nil
}

func mergeCommandEnv(base []string, variables []acpsdk.EnvVariable) []string {
	merged := make(map[string]string, len(base)+len(variables))
	order := make([]string, 0, len(base)+len(variables))

	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if _, exists := merged[name]; !exists {
			order = append(order, name)
		}
		merged[name] = parts[1]
	}

	for _, variable := range variables {
		if _, exists := merged[variable.Name]; !exists {
			order = append(order, variable.Name)
		}
		merged[variable.Name] = variable.Value
	}

	result := make([]string, 0, len(order))
	for _, name := range order {
		result = append(result, fmt.Sprintf("%s=%s", name, merged[name]))
	}
	return result
}

func sliceLines(content string, line, limit *int) string {
	if line == nil && limit == nil {
		return content
	}

	lines := strings.Split(content, "\n")
	start := 0
	if line != nil && *line > 1 {
		start = min(*line-1, len(lines))
	}
	end := len(lines)
	if limit != nil && *limit >= 0 && start+*limit < end {
		end = start + *limit
	}
	return strings.Join(lines[start:end], "\n")
}

func extractContentText(block acpsdk.ContentBlock) string {
	switch {
	case block.Text != nil:
		return block.Text.Text
	case block.ResourceLink != nil:
		return block.ResourceLink.Uri
	default:
		return ""
	}
}

func appendTerminalOutputWindow(dst []byte, src []byte, limit int) ([]byte, bool) {
	if limit <= 0 {
		return nil, len(dst) > 0 || len(src) > 0
	}
	if len(src) == 0 {
		if len(dst) <= limit {
			return dst, false
		}
		return trimUTF8LeadingBytes(dst[len(dst)-limit:]), true
	}
	if len(dst)+len(src) <= limit {
		return append(dst, src...), false
	}

	var out []byte
	if cap(dst) == limit {
		out = dst[:0]
	} else {
		out = make([]byte, 0, limit)
	}

	if len(src) >= limit {
		out = append(out, src[len(src)-limit:]...)
		return trimUTF8LeadingBytes(out), true
	}

	keepFromDst := limit - len(src)
	if len(dst) > keepFromDst {
		dst = dst[len(dst)-keepFromDst:]
	}
	out = append(out, dst...)
	out = append(out, src...)
	return trimUTF8LeadingBytes(out), true
}

func trimUTF8LeadingBytes(data []byte) []byte {
	trim := 0
	for trim < len(data) && !isValidUTF8LeadingByte(data[trim]) {
		trim++
	}
	if trim == 0 {
		return data
	}
	copy(data, data[trim:])
	return data[:len(data)-trim]
}

func isValidUTF8LeadingByte(b byte) bool {
	return b < utf8.RuneSelf || (b >= 0xC2 && b <= 0xF4)
}

func fallbackPermissionEventRaw(requestID string, decision permissionDecision) json.RawMessage {
	var builder strings.Builder
	builder.WriteString(`{"request_id":`)
	builder.WriteString(strconv.Quote(requestID))
	if decision != "" && decision != decisionPending {
		builder.WriteString(`,"decision":`)
		builder.WriteString(strconv.Quote(string(decision)))
	}
	builder.WriteByte('}')
	return json.RawMessage(builder.String())
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return encoded
}

func watchTerminalShutdown(ctx context.Context, terminalDone <-chan struct{}, onShutdown func()) <-chan struct{} {
	watcherDone := make(chan struct{})
	if ctx == nil {
		close(watcherDone)
		return watcherDone
	}

	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			if onShutdown != nil {
				onShutdown()
			}
		case <-terminalDone:
		}
	}()

	return watcherDone
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}

func (p *AgentProcess) activeTurnID() string {
	p.promptMu.RLock()
	defer p.promptMu.RUnlock()
	active := p.activePrompt
	if active == nil {
		return ""
	}
	return active.turnID
}
