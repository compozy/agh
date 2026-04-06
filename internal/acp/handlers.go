package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	acpsdk "github.com/coder/acp-go-sdk"
)

const defaultTerminalOutputLimit = 64 * 1024

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
	ctx    context.Context
	logger *slog.Logger

	nextID atomic.Uint64

	mu        sync.RWMutex
	terminals map[string]*managedTerminal
}

type managedTerminal struct {
	id string

	cmd *exec.Cmd

	mu         sync.RWMutex
	output     []byte
	truncated  bool
	exitStatus *acpsdk.TerminalExitStatus
	done       chan struct{}
}

type terminalOutputWriter struct {
	terminal *managedTerminal
}

func (p *AgentProcess) handleInbound(ctx context.Context, method string, params json.RawMessage) (any, *acpsdk.RequestError) {
	switch method {
	case acpsdk.ClientMethodFsReadTextFile:
		var request acpsdk.ReadTextFileRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleReadTextFile(ctx, request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodFsWriteTextFile:
		var request acpsdk.WriteTextFileRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleWriteTextFile(ctx, request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodSessionRequestPermission:
		var request acpsdk.RequestPermissionRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleRequestPermission(ctx, request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodSessionUpdate:
		if err := p.handleSessionUpdate(params); err != nil {
			return nil, requestError(err)
		}
		return nil, nil
	case acpsdk.ClientMethodTerminalCreate:
		var request acpsdk.CreateTerminalRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleCreateTerminal(request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodTerminalKill:
		var request acpsdk.KillTerminalCommandRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleKillTerminal(request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodTerminalOutput:
		var request acpsdk.TerminalOutputRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleTerminalOutput(request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodTerminalWaitForExit:
		var request acpsdk.WaitForTerminalExitRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleWaitForTerminalExit(ctx, request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	case acpsdk.ClientMethodTerminalRelease:
		var request acpsdk.ReleaseTerminalRequest
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
		}
		response, err := p.handleReleaseTerminal(request)
		if err != nil {
			return nil, requestError(err)
		}
		return response, nil
	default:
		return nil, acpsdk.NewMethodNotFound(method)
	}
}

func (p *AgentProcess) handleReadTextFile(_ context.Context, request acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	if err := p.permissions.authorize(permissionReadTextFile); err != nil {
		return acpsdk.ReadTextFileResponse{}, err
	}
	resolvedPath, err := p.permissions.resolvePath(request.Path)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, err
	}
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return acpsdk.ReadTextFileResponse{}, fmt.Errorf("acp: read %q: %w", resolvedPath, err)
	}
	return acpsdk.ReadTextFileResponse{Content: sliceLines(string(content), request.Line, request.Limit)}, nil
}

func (p *AgentProcess) handleWriteTextFile(_ context.Context, request acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	if err := p.permissions.authorize(permissionWriteTextFile); err != nil {
		return acpsdk.WriteTextFileResponse{}, err
	}
	resolvedPath, err := p.permissions.resolvePath(request.Path)
	if err != nil {
		return acpsdk.WriteTextFileResponse{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return acpsdk.WriteTextFileResponse{}, fmt.Errorf("acp: create parent directories for %q: %w", resolvedPath, err)
	}
	if err := os.WriteFile(resolvedPath, []byte(request.Content), 0o644); err != nil {
		return acpsdk.WriteTextFileResponse{}, fmt.Errorf("acp: write %q: %w", resolvedPath, err)
	}
	return acpsdk.WriteTextFileResponse{}, nil
}

func (p *AgentProcess) handleRequestPermission(ctx context.Context, request acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
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

	decision, interactive := p.permissions.permissionDecision(request)

	if !interactive {
		requestID := p.nextPermissionRequestID(turnID, request)
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decision)
		raw := buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPromptEvent(AgentEvent{
			Type:       EventTypePermission,
			SessionID:  string(request.SessionId),
			TurnID:     turnID,
			RequestID:  requestID,
			Timestamp:  timeNowUTC(),
			Title:      title,
			ToolCallID: strings.TrimSpace(string(request.ToolCall.ToolCallId)),
			Action:     string(permissionRequestToolGrant),
			Resource:   resource,
			Decision:   string(appliedDecision),
			Raw:        cloneRawJSON(raw),
		})
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	}

	requestID, pending := p.registerPendingPermission(turnID, request)
	defer p.clearPendingPermission(requestID)
	raw := buildPermissionEventRaw(requestID, decisionPending, request)

	p.emitPromptEvent(AgentEvent{
		Type:       EventTypePermission,
		SessionID:  string(request.SessionId),
		TurnID:     turnID,
		RequestID:  requestID,
		Timestamp:  timeNowUTC(),
		Title:      title,
		ToolCallID: strings.TrimSpace(string(request.ToolCall.ToolCallId)),
		Action:     string(permissionRequestToolGrant),
		Resource:   resource,
		Raw:        cloneRawJSON(raw),
	})

	timer := time.NewTimer(p.permissionTimeoutOrDefault())
	defer timer.Stop()

	select {
	case resolvedDecision := <-pending.response:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, resolvedDecision)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPromptEvent(AgentEvent{
			Type:       EventTypePermission,
			SessionID:  string(request.SessionId),
			TurnID:     turnID,
			RequestID:  requestID,
			Timestamp:  timeNowUTC(),
			Title:      title,
			ToolCallID: strings.TrimSpace(string(request.ToolCall.ToolCallId)),
			Action:     string(permissionRequestToolGrant),
			Resource:   resource,
			Decision:   string(appliedDecision),
			Raw:        cloneRawJSON(raw),
		})
		return acpsdk.RequestPermissionResponse{Outcome: outcome}, nil
	case <-timer.C:
		outcome, appliedDecision := selectPermissionOutcome(request.Options, decisionRejectOnce)
		raw = buildPermissionEventRaw(requestID, appliedDecision, request)
		p.emitPromptEvent(AgentEvent{
			Type:       EventTypePermission,
			SessionID:  string(request.SessionId),
			TurnID:     turnID,
			RequestID:  requestID,
			Timestamp:  timeNowUTC(),
			Title:      title,
			ToolCallID: strings.TrimSpace(string(request.ToolCall.ToolCallId)),
			Action:     string(permissionRequestToolGrant),
			Resource:   resource,
			Decision:   string(appliedDecision),
			Raw:        cloneRawJSON(raw),
		})
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
				Raw:       cloneRawJSON(raw.Update),
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

func (p *AgentProcess) handleCreateTerminal(request acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
	if err := p.permissions.authorize(permissionCreateTerminal); err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}

	cwd := p.Cwd
	if request.Cwd != nil {
		cwd = *request.Cwd
	}
	resolvedCwd, err := p.permissions.resolvePath(cwd)
	if err != nil {
		return acpsdk.CreateTerminalResponse{}, err
	}

	return p.terminals.create(resolvedCwd, request)
}

func (p *AgentProcess) handleKillTerminal(request acpsdk.KillTerminalCommandRequest) (acpsdk.KillTerminalCommandResponse, error) {
	if err := p.terminals.kill(request.TerminalId); err != nil {
		return acpsdk.KillTerminalCommandResponse{}, err
	}
	return acpsdk.KillTerminalCommandResponse{}, nil
}

func (p *AgentProcess) handleTerminalOutput(request acpsdk.TerminalOutputRequest) (acpsdk.TerminalOutputResponse, error) {
	output, truncated, exitStatus, err := p.terminals.output(request.TerminalId)
	if err != nil {
		return acpsdk.TerminalOutputResponse{}, err
	}
	return acpsdk.TerminalOutputResponse{
		Output:     output,
		Truncated:  truncated,
		ExitStatus: exitStatus,
	}, nil
}

func (p *AgentProcess) handleWaitForTerminalExit(ctx context.Context, request acpsdk.WaitForTerminalExitRequest) (acpsdk.WaitForTerminalExitResponse, error) {
	exitStatus, err := p.terminals.wait(ctx, request.TerminalId)
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

func (p *AgentProcess) handleReleaseTerminal(request acpsdk.ReleaseTerminalRequest) (acpsdk.ReleaseTerminalResponse, error) {
	if err := p.terminals.release(request.TerminalId); err != nil {
		return acpsdk.ReleaseTerminalResponse{}, err
	}
	return acpsdk.ReleaseTerminalResponse{}, nil
}

func newTerminalManager(ctx context.Context, logger *slog.Logger) *terminalManager {
	return &terminalManager{
		ctx:       ctx,
		logger:    logger,
		terminals: make(map[string]*managedTerminal),
	}
}

func (m *terminalManager) create(cwd string, request acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
	cmd := exec.CommandContext(m.ctx, request.Command, request.Args...)
	configureManagedCommand(cmd)
	cmd.Dir = cwd
	cmd.Env = mergeCommandEnv(os.Environ(), request.Env)

	term := &managedTerminal{
		id:   fmt.Sprintf("term-%d", m.nextID.Add(1)),
		cmd:  cmd,
		done: make(chan struct{}),
	}
	writer := &terminalOutputWriter{terminal: term}
	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Start(); err != nil {
		return acpsdk.CreateTerminalResponse{}, fmt.Errorf("acp: start terminal command %q: %w", request.Command, err)
	}

	m.mu.Lock()
	m.terminals[term.id] = term
	m.mu.Unlock()

	go term.wait()

	return acpsdk.CreateTerminalResponse{TerminalId: term.id}, nil
}

func (m *terminalManager) kill(id string) error {
	term, err := m.lookup(id)
	if err != nil {
		return err
	}
	if err := killManagedProcess(term.cmd); err != nil && !strings.Contains(err.Error(), "process already finished") {
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
	_ = killManagedProcess(term.cmd)
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
		_ = killManagedProcess(terminal.cmd)
	}
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
	t.output = append(t.output, p...)
	if len(t.output) > defaultTerminalOutputLimit {
		t.output = trimUTF8LeadingBytes(t.output[len(t.output)-defaultTerminalOutputLimit:])
		t.truncated = true
	}
}

func (t *managedTerminal) wait() {
	err := t.cmd.Wait()
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

	t.mu.Lock()
	t.exitStatus = exitStatus
	t.mu.Unlock()
	close(t.done)
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

func translateSessionUpdate(notification acpsdk.SessionNotification, rawUpdate json.RawMessage, turnID string) AgentEvent {
	event := AgentEvent{
		SessionID: string(notification.SessionId),
		TurnID:    turnID,
		Timestamp: timeNowUTC(),
		Raw:       cloneRawJSON(rawUpdate),
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
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrPathOutsideWorkspace) {
		return acpsdk.NewInvalidParams(map[string]any{"error": err.Error()})
	}
	return acpsdk.NewInternalError(map[string]any{"error": err.Error()})
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
		start = *line - 1
		if start > len(lines) {
			start = len(lines)
		}
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

func trimUTF8LeadingBytes(data []byte) []byte {
	trimmed := append([]byte(nil), data...)
	for len(trimmed) > 0 && !utf8.Valid(trimmed) {
		_, size := utf8.DecodeRune(trimmed)
		if size <= 0 {
			size = 1
		}
		trimmed = trimmed[size:]
	}
	return trimmed
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

func cloneRawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
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
