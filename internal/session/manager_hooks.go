package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

const (
	hookInputClassUserMessage = acp.EventTypeUserMessage
	hookInputClassStartup     = "startup_prompt"

	hookMessageRoleAssistant = "assistant"

	hookMessageDeltaTypeFull    = "full"
	hookMessageDeltaTypeText    = "text"
	hookMessageDeltaTypeThought = "thought"
)

type promptTurnDispatchState struct {
	session     *Session
	turnID      string
	inputClass  string
	userMessage string
	messageSeq  int
	turnEnded   bool
	openMessage *promptMessageDispatchState
}

type promptMessageDispatchState struct {
	id      string
	role    string
	text    strings.Builder
	lastRaw json.RawMessage
}

func newPromptTurnDispatchState(session *Session, turnID string, inputClass string, userMessage string) *promptTurnDispatchState {
	return &promptTurnDispatchState{
		session:     session,
		turnID:      strings.TrimSpace(turnID),
		inputClass:  strings.TrimSpace(inputClass),
		userMessage: userMessage,
	}
}

func (m *Manager) dispatchSessionPreCreate(ctx context.Context, opts CreateOpts) (CreateOpts, error) {
	if m == nil || m.hooks == nil {
		return opts, nil
	}

	payload, err := m.hooks.DispatchSessionPreCreate(ctx, hookspkg.SessionPreCreatePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPreCreate,
			Timestamp: m.now(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionName: strings.TrimSpace(opts.Name),
			SessionType: string(normalizeSessionType(opts.Type)),
			AgentName:   strings.TrimSpace(opts.AgentName),
			WorkspaceID: strings.TrimSpace(opts.Workspace),
			Workspace:   strings.TrimSpace(opts.WorkspacePath),
			State:       string(StateStarting),
		},
	})
	if err != nil {
		return CreateOpts{}, fmt.Errorf("session: dispatch session.pre_create: %w", err)
	}

	next := opts
	next.Name = strings.TrimSpace(payload.SessionName)
	next.Type = normalizeSessionType(SessionType(strings.TrimSpace(payload.SessionType)))
	next.AgentName = strings.TrimSpace(payload.AgentName)

	workspaceID := strings.TrimSpace(payload.WorkspaceID)
	workspacePath := strings.TrimSpace(payload.Workspace)
	switch {
	case workspaceID != "" && workspacePath != "":
		return CreateOpts{}, errors.New("session: session.pre_create produced both workspace id and workspace path")
	case workspaceID != "":
		next.Workspace = workspaceID
		next.WorkspacePath = ""
	case workspacePath != "":
		next.Workspace = ""
		next.WorkspacePath = workspacePath
	default:
		next.Workspace = ""
		next.WorkspacePath = ""
	}

	return next, nil
}

func (m *Manager) dispatchSessionPreResume(ctx context.Context, meta store.SessionMeta) (store.SessionMeta, error) {
	if m == nil || m.hooks == nil {
		return meta, nil
	}

	payload, err := m.hooks.DispatchSessionPreResume(ctx, hookspkg.SessionPreResumePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPreResume,
			Timestamp: m.now(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:    strings.TrimSpace(meta.ID),
			SessionName:  strings.TrimSpace(meta.Name),
			SessionType:  string(normalizeSessionType(SessionType(meta.SessionType))),
			AgentName:    strings.TrimSpace(meta.AgentName),
			WorkspaceID:  strings.TrimSpace(meta.WorkspaceID),
			ACPSessionID: strings.TrimSpace(derefString(meta.ACPSessionID)),
			State:        strings.TrimSpace(meta.State),
			CreatedAt:    meta.CreatedAt,
			UpdatedAt:    meta.UpdatedAt,
		},
	})
	if err != nil {
		return store.SessionMeta{}, fmt.Errorf("session: dispatch session.pre_resume: %w", err)
	}

	next := meta
	next.Name = strings.TrimSpace(payload.SessionName)
	next.AgentName = strings.TrimSpace(payload.AgentName)
	next.WorkspaceID = strings.TrimSpace(payload.WorkspaceID)
	next.SessionType = string(normalizeSessionType(SessionType(strings.TrimSpace(payload.SessionType))))
	return next, nil
}

func (m *Manager) dispatchSessionPostCreate(ctx context.Context, session *Session) {
	m.dispatchSessionLifecycleObservation(ctx, session, hookspkg.HookSessionPostCreate)
}

func (m *Manager) dispatchSessionPostResume(ctx context.Context, session *Session) {
	m.dispatchSessionLifecycleObservation(ctx, session, hookspkg.HookSessionPostResume)
}

func (m *Manager) dispatchSessionPreStop(ctx context.Context, session *Session) error {
	if m == nil || m.hooks == nil || session == nil {
		return nil
	}
	ctx = hookDispatchContext(ctx, session)

	payload, err := m.hooks.DispatchSessionPreStop(ctx, hookSessionLifecyclePayload(session, hookspkg.HookSessionPreStop, m.now()))
	if err != nil {
		return fmt.Errorf("session: dispatch session.pre_stop: %w", err)
	}

	session.applyHookSessionContext(payload.SessionContext, m.now())
	return nil
}

func (m *Manager) dispatchSessionPostStop(ctx context.Context, session *Session) {
	m.dispatchSessionLifecycleObservation(ctx, session, hookspkg.HookSessionPostStop)
}

func (m *Manager) dispatchSessionLifecycleObservation(ctx context.Context, session *Session, event hookspkg.HookEvent) {
	if m == nil || m.hooks == nil || session == nil {
		return
	}
	ctx = hookDispatchContext(ctx, session)

	payload := hookSessionLifecyclePayload(session, event, m.now())
	var err error
	switch event {
	case hookspkg.HookSessionPostCreate:
		_, err = m.hooks.DispatchSessionPostCreate(ctx, hookspkg.SessionPostCreatePayload(payload))
	case hookspkg.HookSessionPostResume:
		_, err = m.hooks.DispatchSessionPostResume(ctx, hookspkg.SessionPostResumePayload(payload))
	case hookspkg.HookSessionPostStop:
		_, err = m.hooks.DispatchSessionPostStop(ctx, hookspkg.SessionPostStopPayload(payload))
	default:
		return
	}
	if err != nil {
		m.warnHookDispatch(ctx, session, event, err)
	}
}

func (m *Manager) dispatchInputPreSubmit(ctx context.Context, session *Session, turnID string, message string) (string, error) {
	if m == nil || m.hooks == nil {
		return message, nil
	}
	ctx = hookDispatchContext(ctx, session)

	payload, err := m.hooks.DispatchInputPreSubmit(ctx, hookspkg.InputPreSubmitPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookInputPreSubmit,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		TurnContext:    hookspkg.TurnContext{TurnID: strings.TrimSpace(turnID)},
		InputClass:     hookInputClassUserMessage,
		Message:        message,
	})
	if err != nil {
		return "", fmt.Errorf("session: dispatch input.pre_submit: %w", err)
	}

	return payload.Message, nil
}

func (m *Manager) dispatchPromptPostAssemble(ctx context.Context, sessionCtx hookspkg.SessionContext, prompt string) (string, error) {
	if m == nil || m.hooks == nil {
		return prompt, nil
	}

	payload, err := m.hooks.DispatchPromptPostAssemble(ctx, hookspkg.PromptPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookPromptPostAssemble,
			Timestamp: m.now(),
		},
		SessionContext: sessionCtx,
		InputClass:     hookInputClassStartup,
		Prompt:         prompt,
	})
	if err != nil {
		return "", fmt.Errorf("session: dispatch prompt.post_assemble: %w", err)
	}

	return strings.TrimSpace(payload.Prompt), nil
}

func (m *Manager) dispatchTurnStart(ctx context.Context, state *promptTurnDispatchState) error {
	if m == nil || m.hooks == nil || state == nil {
		return nil
	}
	ctx = hookDispatchContext(ctx, state.session)

	_, err := m.hooks.DispatchTurnStart(ctx, hookspkg.TurnStartPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTurnStart,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(state.session),
		TurnContext:    hookspkg.TurnContext{TurnID: state.turnID},
		InputClass:     state.inputClass,
		UserMessage:    state.userMessage,
	})
	if err != nil {
		return fmt.Errorf("session: dispatch turn.start: %w", err)
	}

	return nil
}

func (m *Manager) dispatchTurnEnd(ctx context.Context, state *promptTurnDispatchState, eventTime time.Time) {
	if state == nil || state.turnEnded {
		return
	}
	state.turnEnded = true
	if m == nil || m.hooks == nil {
		return
	}
	ctx = hookDispatchContext(ctx, state.session)

	_, err := m.hooks.DispatchTurnEnd(ctx, hookspkg.TurnEndPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTurnEnd,
			Timestamp: hookTimestamp(m.now(), eventTime),
		},
		SessionContext: hookSessionContext(state.session),
		TurnContext:    hookspkg.TurnContext{TurnID: state.turnID},
		InputClass:     state.inputClass,
		UserMessage:    state.userMessage,
	})
	if err != nil {
		m.warnHookDispatch(ctx, state.session, hookspkg.HookTurnEnd, err)
	}
}

func (m *Manager) preparePromptEvent(ctx context.Context, state *promptTurnDispatchState, event acp.AgentEvent) acp.AgentEvent {
	if state == nil {
		return event
	}

	role, deltaType, isMessage := hookMessageDetails(event.Type)
	if !isMessage {
		m.finishPromptMessage(ctx, state, event.Timestamp)
		return event
	}

	if state.openMessage == nil {
		event = m.dispatchMessageStart(ctx, state, event, role)
	}

	if state.openMessage == nil {
		return event
	}

	state.openMessage.text.WriteString(event.Text)
	state.openMessage.lastRaw = cloneSessionRawMessage(event.Raw)
	m.dispatchMessageDelta(ctx, state, event, deltaType)
	return event
}

func (m *Manager) dispatchMessageStart(ctx context.Context, state *promptTurnDispatchState, event acp.AgentEvent, role string) acp.AgentEvent {
	if state == nil {
		return event
	}

	state.messageSeq++
	message := &promptMessageDispatchState{
		id:   nextPromptMessageID(state.turnID, state.messageSeq),
		role: strings.TrimSpace(role),
	}
	state.openMessage = message
	if m == nil || m.hooks == nil {
		return event
	}
	ctx = hookDispatchContext(ctx, state.session)

	payload, err := m.hooks.DispatchMessageStart(ctx, hookspkg.MessageStartPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookMessageStart,
			Timestamp: hookTimestamp(m.now(), event.Timestamp),
		},
		SessionContext: hookSessionContext(state.session),
		TurnContext:    hookspkg.TurnContext{TurnID: state.turnID},
		MessageID:      message.id,
		Role:           message.role,
		DeltaType:      hookMessageDeltaTypeFull,
		Text:           event.Text,
		Raw:            cloneSessionRawMessage(event.Raw),
	})
	if err != nil {
		m.warnHookDispatch(ctx, state.session, hookspkg.HookMessageStart, err)
		return event
	}

	message.role = strings.TrimSpace(payload.Role)
	event.Text = payload.Text
	return event
}

func (m *Manager) dispatchMessageDelta(ctx context.Context, state *promptTurnDispatchState, event acp.AgentEvent, deltaType string) {
	if m == nil || m.hooks == nil || state == nil || state.openMessage == nil {
		return
	}
	ctx = hookDispatchContext(ctx, state.session)

	_, err := m.hooks.DispatchMessageDelta(ctx, hookspkg.MessageDeltaPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookMessageDelta,
			Timestamp: hookTimestamp(m.now(), event.Timestamp),
		},
		SessionContext: hookSessionContext(state.session),
		TurnContext:    hookspkg.TurnContext{TurnID: state.turnID},
		MessageID:      state.openMessage.id,
		Role:           state.openMessage.role,
		DeltaType:      strings.TrimSpace(deltaType),
		Text:           event.Text,
		Raw:            cloneSessionRawMessage(event.Raw),
	})
	if err != nil {
		m.warnHookDispatch(ctx, state.session, hookspkg.HookMessageDelta, err)
	}
}

func (m *Manager) finishPromptMessage(ctx context.Context, state *promptTurnDispatchState, eventTime time.Time) {
	if state == nil || state.openMessage == nil {
		return
	}

	message := state.openMessage
	state.openMessage = nil
	if m == nil || m.hooks == nil {
		return
	}
	ctx = hookDispatchContext(ctx, state.session)

	_, err := m.hooks.DispatchMessageEnd(ctx, hookspkg.MessageEndPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookMessageEnd,
			Timestamp: hookTimestamp(m.now(), eventTime),
		},
		SessionContext: hookSessionContext(state.session),
		TurnContext:    hookspkg.TurnContext{TurnID: state.turnID},
		MessageID:      message.id,
		Role:           message.role,
		DeltaType:      hookMessageDeltaTypeFull,
		Text:           message.text.String(),
		Raw:            cloneSessionRawMessage(message.lastRaw),
	})
	if err != nil {
		m.warnHookDispatch(ctx, state.session, hookspkg.HookMessageEnd, err)
	}
}

func (m *Manager) dispatchEventPreRecord(ctx context.Context, session *Session, event acp.AgentEvent, content string) {
	if m == nil || m.hooks == nil {
		return
	}
	ctx = hookDispatchContext(ctx, session)

	_, err := m.hooks.DispatchEventPreRecord(ctx, hookspkg.EventPreRecordPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEventPreRecord,
			Timestamp: hookTimestamp(m.now(), event.Timestamp),
		},
		SessionContext: hookSessionContext(session),
		TurnContext:    hookspkg.TurnContext{TurnID: strings.TrimSpace(event.TurnID)},
		RecordType:     strings.TrimSpace(event.Type),
		Content:        json.RawMessage(content),
	})
	if err != nil {
		m.warnHookDispatch(ctx, session, hookspkg.HookEventPreRecord, err)
	}
}

func (m *Manager) runContextCompaction(
	ctx context.Context,
	session *Session,
	turnID string,
	reason string,
	strategy string,
	summary string,
	contextBlocks []hookspkg.ContextBlock,
	compact func(context.Context, hookspkg.ContextPreCompactPayload) (hookspkg.ContextPostCompactPayload, error),
) (hookspkg.ContextPostCompactPayload, error) {
	if compact == nil {
		return hookspkg.ContextPostCompactPayload{}, errors.New("session: context compactor is required")
	}

	now := time.Now().UTC
	if m != nil {
		now = m.now
	}
	ctx = hookDispatchContext(ctx, session)

	prePayload := hookspkg.ContextPreCompactPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookContextPreCompact,
			Timestamp: now(),
		},
		SessionContext: hookSessionContext(session),
		TurnContext:    hookspkg.TurnContext{TurnID: strings.TrimSpace(turnID)},
		Reason:         strings.TrimSpace(reason),
		Strategy:       strings.TrimSpace(strategy),
		Summary:        strings.TrimSpace(summary),
		ContextBlocks:  cloneSessionContextBlocks(contextBlocks),
	}

	var err error
	if m != nil && m.hooks != nil {
		prePayload, err = m.hooks.DispatchContextPreCompact(ctx, prePayload)
		if err != nil {
			return hookspkg.ContextPostCompactPayload{}, fmt.Errorf("session: dispatch context.pre_compact: %w", err)
		}
	}

	postPayload, err := compact(ctx, prePayload)
	if err != nil {
		return hookspkg.ContextPostCompactPayload{}, err
	}

	postPayload.Event = hookspkg.HookContextPostCompact
	if postPayload.Timestamp.IsZero() {
		postPayload.Timestamp = now()
	}
	if strings.TrimSpace(postPayload.SessionID) == "" {
		postPayload.SessionContext = prePayload.SessionContext
	}
	if strings.TrimSpace(postPayload.TurnID) == "" {
		postPayload.TurnContext = prePayload.TurnContext
	}
	if strings.TrimSpace(postPayload.Reason) == "" {
		postPayload.Reason = prePayload.Reason
	}
	if strings.TrimSpace(postPayload.Strategy) == "" {
		postPayload.Strategy = prePayload.Strategy
	}
	if postPayload.ContextBlocks == nil {
		postPayload.ContextBlocks = cloneSessionContextBlocks(prePayload.ContextBlocks)
	}

	if m != nil && m.hooks != nil {
		if _, err := m.hooks.DispatchContextPostCompact(ctx, postPayload); err != nil {
			m.warnHookDispatch(ctx, session, hookspkg.HookContextPostCompact, err)
		}
	}

	return postPayload, nil
}

func (m *Manager) dispatchEventPostRecord(ctx context.Context, session *Session, event acp.AgentEvent, content string) {
	if m == nil || m.hooks == nil {
		return
	}
	ctx = hookDispatchContext(ctx, session)

	_, err := m.hooks.DispatchEventPostRecord(ctx, hookspkg.EventPostRecordPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookEventPostRecord,
			Timestamp: hookTimestamp(m.now(), event.Timestamp),
		},
		SessionContext: hookSessionContext(session),
		TurnContext:    hookspkg.TurnContext{TurnID: strings.TrimSpace(event.TurnID)},
		RecordType:     strings.TrimSpace(event.Type),
		Content:        json.RawMessage(content),
	})
	if err != nil {
		m.warnHookDispatch(ctx, session, hookspkg.HookEventPostRecord, err)
	}
}

func (m *Manager) dispatchAgentPreStart(ctx context.Context, session *Session, resolved aghconfig.ResolvedAgent, opts acp.StartOpts) (acp.StartOpts, error) {
	if m == nil || m.hooks == nil {
		return opts, nil
	}
	ctx = hookDispatchContext(ctx, session)

	command, args := splitCommand(opts.Command)
	payload, err := m.hooks.DispatchAgentPreStart(ctx, hookspkg.AgentPreStartPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookAgentPreStart,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		Command:        command,
		Args:           args,
		Cwd:            strings.TrimSpace(opts.Cwd),
		Provider:       strings.TrimSpace(resolved.Provider),
		Model:          strings.TrimSpace(resolved.Model),
	})
	if err != nil {
		return acp.StartOpts{}, fmt.Errorf("session: dispatch agent.pre_start: %w", err)
	}

	next := opts
	next.Command = joinCommand(payload.Command, payload.Args)
	next.Cwd = strings.TrimSpace(payload.Cwd)
	return next, nil
}

func (m *Manager) dispatchAgentSpawned(ctx context.Context, session *Session, proc *AgentProcess, resolved aghconfig.ResolvedAgent) {
	m.dispatchAgentObservation(ctx, session, proc, resolved, nil, hookspkg.HookAgentSpawned)
}

func (m *Manager) dispatchAgentCrashed(ctx context.Context, session *Session, proc *AgentProcess, waitErr error) {
	m.dispatchAgentObservation(ctx, session, proc, aghconfig.ResolvedAgent{}, waitErr, hookspkg.HookAgentCrashed)
}

func (m *Manager) dispatchAgentStopped(ctx context.Context, session *Session, proc *AgentProcess, waitErr error) {
	m.dispatchAgentObservation(ctx, session, proc, aghconfig.ResolvedAgent{}, waitErr, hookspkg.HookAgentStopped)
}

func (m *Manager) dispatchAgentObservation(ctx context.Context, session *Session, proc *AgentProcess, resolved aghconfig.ResolvedAgent, waitErr error, event hookspkg.HookEvent) {
	if m == nil || m.hooks == nil {
		return
	}
	ctx = hookDispatchContext(ctx, session)

	command, args := agentCommandAndArgs(proc)
	payload := hookspkg.AgentLifecyclePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     event,
			Timestamp: m.now(),
		},
		SessionContext: hookSessionContext(session),
		Command:        command,
		Args:           args,
		Cwd:            agentCwd(proc),
		PID:            agentPID(proc),
		Provider:       strings.TrimSpace(resolved.Provider),
		Model:          strings.TrimSpace(resolved.Model),
	}
	if waitErr != nil {
		payload.Error = waitErr.Error()
	}

	var err error
	switch event {
	case hookspkg.HookAgentSpawned:
		_, err = m.hooks.DispatchAgentSpawned(ctx, hookspkg.AgentSpawnedPayload(payload))
	case hookspkg.HookAgentCrashed:
		_, err = m.hooks.DispatchAgentCrashed(ctx, hookspkg.AgentCrashedPayload(payload))
	case hookspkg.HookAgentStopped:
		_, err = m.hooks.DispatchAgentStopped(ctx, hookspkg.AgentStoppedPayload(payload))
	default:
		return
	}
	if err != nil {
		m.warnHookDispatch(ctx, session, event, err)
	}
}

func hookSessionLifecyclePayload(session *Session, event hookspkg.HookEvent, timestamp time.Time) hookspkg.SessionLifecyclePayload {
	return hookspkg.SessionLifecyclePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     event,
			Timestamp: timestamp,
		},
		SessionContext: hookSessionContext(session),
	}
}

func hookSessionContext(session *Session) hookspkg.SessionContext {
	if session == nil {
		return hookspkg.SessionContext{}
	}

	info := session.Info()
	if info == nil {
		return hookspkg.SessionContext{}
	}

	return hookspkg.SessionContext{
		SessionID:    strings.TrimSpace(info.ID),
		SessionName:  strings.TrimSpace(info.Name),
		SessionType:  string(info.Type),
		AgentName:    strings.TrimSpace(info.AgentName),
		WorkspaceID:  strings.TrimSpace(info.WorkspaceID),
		Workspace:    strings.TrimSpace(info.Workspace),
		ACPSessionID: strings.TrimSpace(info.ACPSessionID),
		State:        string(info.State),
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

func (s *Session) applyHookSessionContext(payload hookspkg.SessionContext, now time.Time) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Name = strings.TrimSpace(payload.SessionName)
	s.AgentName = strings.TrimSpace(payload.AgentName)
	s.WorkspaceID = strings.TrimSpace(payload.WorkspaceID)
	s.Workspace = strings.TrimSpace(payload.Workspace)
	s.Type = normalizeSessionType(SessionType(strings.TrimSpace(payload.SessionType)))
	if !now.IsZero() {
		s.UpdatedAt = now
	}
}

func hookTimestamp(now time.Time, eventTime time.Time) time.Time {
	if !eventTime.IsZero() {
		return eventTime
	}
	return now
}

func splitCommand(command string) (string, []string) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", nil
	}

	parts, err := shellquote.Split(trimmed)
	if err != nil || len(parts) == 0 {
		return trimmed, nil
	}

	return parts[0], append([]string(nil), parts[1:]...)
}

func joinCommand(command string, args []string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}
	if len(args) == 0 {
		return trimmed
	}

	parts := make([]string, 0, len(args)+1)
	parts = append(parts, trimmed)
	for _, arg := range args {
		if item := strings.TrimSpace(arg); item != "" {
			parts = append(parts, item)
		}
	}
	return shellquote.Join(parts...)
}

func agentCommandAndArgs(proc *AgentProcess) (string, []string) {
	if proc == nil {
		return "", nil
	}
	if len(proc.Args) > 0 {
		return strings.TrimSpace(proc.Command), append([]string(nil), proc.Args...)
	}
	return splitCommand(proc.Command)
}

func agentCwd(proc *AgentProcess) string {
	if proc == nil {
		return ""
	}
	return strings.TrimSpace(proc.Cwd)
}

func agentPID(proc *AgentProcess) int {
	if proc == nil {
		return 0
	}
	return proc.PID
}

func (m *Manager) warnHookDispatch(ctx context.Context, session *Session, event hookspkg.HookEvent, err error) {
	if err == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.sessionLogger(session).WarnContext(
		ctx,
		"session: hook dispatch failed",
		"hook_event", event.String(),
		"error", err,
	)
}

func hookMessageDetails(eventType string) (string, string, bool) {
	switch strings.TrimSpace(eventType) {
	case acp.EventTypeAgentMessage:
		return hookMessageRoleAssistant, hookMessageDeltaTypeText, true
	case acp.EventTypeThought:
		return hookMessageRoleAssistant, hookMessageDeltaTypeThought, true
	default:
		return "", "", false
	}
}

func nextPromptMessageID(turnID string, sequence int) string {
	base := strings.TrimSpace(turnID)
	if base == "" {
		base = "msg"
	}
	if sequence <= 0 {
		return base + "-message"
	}
	return fmt.Sprintf("%s-message-%d", base, sequence)
}

func cloneSessionRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneSessionContextBlocks(blocks []hookspkg.ContextBlock) []hookspkg.ContextBlock {
	if len(blocks) == 0 {
		return nil
	}

	cloned := make([]hookspkg.ContextBlock, 0, len(blocks))
	for _, block := range blocks {
		cloned = append(cloned, hookspkg.ContextBlock{
			Kind:     strings.TrimSpace(block.Kind),
			Text:     block.Text,
			Metadata: cloneStringMap(block.Metadata),
		})
	}
	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(src))
	for key, value := range src {
		cloned[key] = value
	}
	return cloned
}

func hookDispatchContext(ctx context.Context, session *Session) context.Context {
	if ctx == nil || session == nil {
		return ctx
	}

	writer, ok := session.recorderHandle().(hookspkg.HookRunWriter)
	if !ok || writer == nil {
		return ctx
	}

	return hookspkg.WithHookRunWriter(ctx, writer)
}
