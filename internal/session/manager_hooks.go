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
)

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

func (m *Manager) dispatchEventPreRecord(ctx context.Context, session *Session, event acp.AgentEvent, content string) {
	if m == nil || m.hooks == nil {
		return
	}

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

func (m *Manager) dispatchEventPostRecord(ctx context.Context, session *Session, event acp.AgentEvent, content string) {
	if m == nil || m.hooks == nil {
		return
	}

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
