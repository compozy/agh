package session

import (
	"context"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// SessionLifecycleHooks groups create/resume/stop session lifecycle hook dispatch.
type SessionLifecycleHooks interface {
	DispatchSessionPreCreate(context.Context, hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error)
	DispatchSessionPostCreate(context.Context, hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error)
	DispatchSessionPreResume(context.Context, hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error)
	DispatchSessionPostResume(context.Context, hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error)
	DispatchSessionPreStop(context.Context, hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error)
	DispatchSessionPostStop(context.Context, hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error)
}

// PromptHooks groups prompt assembly and user-input hook dispatch.
type PromptHooks interface {
	DispatchInputPreSubmit(context.Context, hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error)
	DispatchPromptPostAssemble(context.Context, hookspkg.PromptPayload) (hookspkg.PromptPayload, error)
}

// EventHooks groups event-record persistence hook dispatch.
type EventHooks interface {
	DispatchEventPreRecord(context.Context, hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error)
	DispatchEventPostRecord(context.Context, hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error)
}

// AgentHooks groups agent start and stop lifecycle hook dispatch.
type AgentHooks interface {
	DispatchAgentPreStart(context.Context, hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error)
	DispatchAgentSpawned(context.Context, hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error)
	DispatchAgentCrashed(context.Context, hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error)
	DispatchAgentStopped(context.Context, hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error)
}

// ConversationHooks groups turn/message hook dispatch used during prompt streaming.
type ConversationHooks interface {
	DispatchTurnStart(context.Context, hookspkg.TurnStartPayload) (hookspkg.TurnStartPayload, error)
	DispatchTurnEnd(context.Context, hookspkg.TurnEndPayload) (hookspkg.TurnEndPayload, error)
	DispatchMessageStart(context.Context, hookspkg.MessageStartPayload) (hookspkg.MessageStartPayload, error)
	DispatchMessageDelta(context.Context, hookspkg.MessageDeltaPayload) (hookspkg.MessageDeltaPayload, error)
	DispatchMessageEnd(context.Context, hookspkg.MessageEndPayload) (hookspkg.MessageEndPayload, error)
}

// CompactionHooks groups context compaction hook dispatch.
type CompactionHooks interface {
	DispatchContextPreCompact(context.Context, hookspkg.ContextPreCompactPayload) (hookspkg.ContextPreCompactPayload, error)
	DispatchContextPostCompact(context.Context, hookspkg.ContextPostCompactPayload) (hookspkg.ContextPostCompactPayload, error)
}

// HookSet collects the grouped session hook domains. Nil groups are treated as
// no-op implementations so callers only provide the domains they exercise.
type HookSet struct {
	Session      SessionLifecycleHooks
	Prompt       PromptHooks
	Events       EventHooks
	Agent        AgentHooks
	Conversation ConversationHooks
	Compaction   CompactionHooks
}

var _ SessionLifecycleHooks = noopSessionLifecycleHooks{}
var _ PromptHooks = noopPromptHooks{}
var _ EventHooks = noopEventHooks{}
var _ AgentHooks = noopAgentHooks{}
var _ ConversationHooks = noopConversationHooks{}
var _ CompactionHooks = noopCompactionHooks{}

func (h HookSet) session() SessionLifecycleHooks {
	if h.Session != nil {
		return h.Session
	}
	return noopSessionLifecycleHooks{}
}

func (h HookSet) prompt() PromptHooks {
	if h.Prompt != nil {
		return h.Prompt
	}
	return noopPromptHooks{}
}

func (h HookSet) events() EventHooks {
	if h.Events != nil {
		return h.Events
	}
	return noopEventHooks{}
}

func (h HookSet) agent() AgentHooks {
	if h.Agent != nil {
		return h.Agent
	}
	return noopAgentHooks{}
}

func (h HookSet) conversation() ConversationHooks {
	if h.Conversation != nil {
		return h.Conversation
	}
	return noopConversationHooks{}
}

func (h HookSet) compaction() CompactionHooks {
	if h.Compaction != nil {
		return h.Compaction
	}
	return noopCompactionHooks{}
}

type noopSessionLifecycleHooks struct{}

func (noopSessionLifecycleHooks) DispatchSessionPreCreate(_ context.Context, payload hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error) {
	return payload, nil
}

func (noopSessionLifecycleHooks) DispatchSessionPostCreate(_ context.Context, payload hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error) {
	return payload, nil
}

func (noopSessionLifecycleHooks) DispatchSessionPreResume(_ context.Context, payload hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error) {
	return payload, nil
}

func (noopSessionLifecycleHooks) DispatchSessionPostResume(_ context.Context, payload hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error) {
	return payload, nil
}

func (noopSessionLifecycleHooks) DispatchSessionPreStop(_ context.Context, payload hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error) {
	return payload, nil
}

func (noopSessionLifecycleHooks) DispatchSessionPostStop(_ context.Context, payload hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error) {
	return payload, nil
}

type noopPromptHooks struct{}

func (noopPromptHooks) DispatchInputPreSubmit(_ context.Context, payload hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error) {
	return payload, nil
}

func (noopPromptHooks) DispatchPromptPostAssemble(_ context.Context, payload hookspkg.PromptPayload) (hookspkg.PromptPayload, error) {
	return payload, nil
}

type noopEventHooks struct{}

func (noopEventHooks) DispatchEventPreRecord(_ context.Context, payload hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error) {
	return payload, nil
}

func (noopEventHooks) DispatchEventPostRecord(_ context.Context, payload hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error) {
	return payload, nil
}

type noopAgentHooks struct{}

func (noopAgentHooks) DispatchAgentPreStart(_ context.Context, payload hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error) {
	return payload, nil
}

func (noopAgentHooks) DispatchAgentSpawned(_ context.Context, payload hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error) {
	return payload, nil
}

func (noopAgentHooks) DispatchAgentCrashed(_ context.Context, payload hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error) {
	return payload, nil
}

func (noopAgentHooks) DispatchAgentStopped(_ context.Context, payload hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error) {
	return payload, nil
}

type noopConversationHooks struct{}

func (noopConversationHooks) DispatchTurnStart(_ context.Context, payload hookspkg.TurnStartPayload) (hookspkg.TurnStartPayload, error) {
	return payload, nil
}

func (noopConversationHooks) DispatchTurnEnd(_ context.Context, payload hookspkg.TurnEndPayload) (hookspkg.TurnEndPayload, error) {
	return payload, nil
}

func (noopConversationHooks) DispatchMessageStart(_ context.Context, payload hookspkg.MessageStartPayload) (hookspkg.MessageStartPayload, error) {
	return payload, nil
}

func (noopConversationHooks) DispatchMessageDelta(_ context.Context, payload hookspkg.MessageDeltaPayload) (hookspkg.MessageDeltaPayload, error) {
	return payload, nil
}

func (noopConversationHooks) DispatchMessageEnd(_ context.Context, payload hookspkg.MessageEndPayload) (hookspkg.MessageEndPayload, error) {
	return payload, nil
}

type noopCompactionHooks struct{}

func (noopCompactionHooks) DispatchContextPreCompact(_ context.Context, payload hookspkg.ContextPreCompactPayload) (hookspkg.ContextPreCompactPayload, error) {
	return payload, nil
}

func (noopCompactionHooks) DispatchContextPostCompact(_ context.Context, payload hookspkg.ContextPostCompactPayload) (hookspkg.ContextPostCompactPayload, error) {
	return payload, nil
}
