package contract

import (
	"time"

	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/memory"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod = extensionprotocol.HostAPIMethod

const (
	HostAPIMethodSessionsList   = extensionprotocol.HostAPIMethodSessionsList
	HostAPIMethodSessionsCreate = extensionprotocol.HostAPIMethodSessionsCreate
	HostAPIMethodSessionsPrompt = extensionprotocol.HostAPIMethodSessionsPrompt
	HostAPIMethodSessionsStop   = extensionprotocol.HostAPIMethodSessionsStop
	HostAPIMethodSessionsStatus = extensionprotocol.HostAPIMethodSessionsStatus
	HostAPIMethodSessionsEvents = extensionprotocol.HostAPIMethodSessionsEvents
	HostAPIMethodMemoryRecall   = extensionprotocol.HostAPIMethodMemoryRecall
	HostAPIMethodMemoryStore    = extensionprotocol.HostAPIMethodMemoryStore
	HostAPIMethodMemoryForget   = extensionprotocol.HostAPIMethodMemoryForget
	HostAPIMethodObserveHealth  = extensionprotocol.HostAPIMethodObserveHealth
	HostAPIMethodObserveEvents  = extensionprotocol.HostAPIMethodObserveEvents
	HostAPIMethodSkillsList     = extensionprotocol.HostAPIMethodSkillsList
)

// NamedType links a generated TypeScript export name to a Go type.
type NamedType struct {
	Name  string
	Value any
}

// HostAPIMethodSpec describes one Host API request/response contract.
type HostAPIMethodSpec struct {
	Method         HostAPIMethod
	Params         NamedType
	Result         NamedType
	OptionalParams bool
}

// EmptyResult is the empty JSON-RPC result for methods without payloads.
type EmptyResult struct{}

// SessionsListParams filters visible sessions.
type SessionsListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// SessionsCreateParams starts a new session.
type SessionsCreateParams struct {
	Agent     string `json:"agent"`
	Prompt    string `json:"prompt,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// SessionsPromptParams submits one prompt to an existing session.
type SessionsPromptParams struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// SessionTargetParams identifies an existing session.
type SessionTargetParams struct {
	SessionID string `json:"session_id"`
}

// SessionEventsParams filters persisted session events.
type SessionEventsParams struct {
	SessionID string    `json:"session_id"`
	Type      string    `json:"type,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	TurnID    string    `json:"turn_id,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int64     `json:"offset,omitempty"`
	Since     time.Time `json:"since,omitempty"`
}

// MemoryStoreParams persists one memory document.
type MemoryStoreParams struct {
	Key       string       `json:"key"`
	Content   string       `json:"content"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
	Tags      []string     `json:"tags,omitempty"`
}

// MemoryRecallParams queries stored memory documents.
type MemoryRecallParams struct {
	Query     string       `json:"query"`
	Limit     int          `json:"limit,omitempty"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
}

// MemoryForgetParams removes one stored memory document.
type MemoryForgetParams struct {
	Key       string       `json:"key"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
}

// ObserveEventsParams filters global observability events.
type ObserveEventsParams struct {
	SessionID string    `json:"session_id,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	Type      string    `json:"type,omitempty"`
	Since     time.Time `json:"since,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// SkillsListParams filters skills by workspace scope.
type SkillsListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// SessionSummary is the lightweight host-visible session listing shape.
type SessionSummary struct {
	ID        string               `json:"id"`
	Name      string               `json:"name,omitempty"`
	Agent     string               `json:"agent"`
	Workspace string               `json:"workspace,omitempty"`
	State     session.SessionState `json:"state"`
	CreatedAt time.Time            `json:"created_at"`
}

// SessionStatus is the detailed host-visible session status shape.
type SessionStatus struct {
	SessionID    string               `json:"session_id"`
	Name         string               `json:"name,omitempty"`
	Agent        string               `json:"agent"`
	WorkspaceID  string               `json:"workspace_id,omitempty"`
	Workspace    string               `json:"workspace,omitempty"`
	State        session.SessionState `json:"state"`
	StopReason   store.StopReason     `json:"stop_reason,omitempty"`
	StopDetail   string               `json:"stop_detail,omitempty"`
	ACPSessionID string               `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// SessionEvent is the host-visible session or observe event record.
type SessionEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// SessionCreateResult returns the created session identifier.
type SessionCreateResult struct {
	SessionID string `json:"session_id"`
}

// SessionPromptResult returns the created turn identifier.
type SessionPromptResult struct {
	TurnID string `json:"turn_id"`
}

// MemoryRecallEntry is one scored memory lookup hit.
type MemoryRecallEntry struct {
	Key     string  `json:"key"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// SkillSummary is the lightweight host-visible skill listing shape.
type SkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

// ObserveHealth is the host-visible daemon health payload.
type ObserveHealth = observepkg.Health

// HostAPIMethodSpecs returns the canonical Host API method registry in wire order.
func HostAPIMethodSpecs() []HostAPIMethodSpec {
	return []HostAPIMethodSpec{
		{
			Method:         HostAPIMethodSessionsList,
			Params:         NamedType{Name: "SessionsListParams", Value: SessionsListParams{}},
			Result:         NamedType{Name: "SessionSummary", Value: []SessionSummary{}},
			OptionalParams: true,
		},
		{
			Method: HostAPIMethodSessionsCreate,
			Params: NamedType{Name: "SessionsCreateParams", Value: SessionsCreateParams{}},
			Result: NamedType{Name: "SessionCreateResult", Value: SessionCreateResult{}},
		},
		{
			Method: HostAPIMethodSessionsPrompt,
			Params: NamedType{Name: "SessionsPromptParams", Value: SessionsPromptParams{}},
			Result: NamedType{Name: "SessionPromptResult", Value: SessionPromptResult{}},
		},
		{
			Method: HostAPIMethodSessionsStop,
			Params: NamedType{Name: "SessionTargetParams", Value: SessionTargetParams{}},
			Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
		},
		{
			Method: HostAPIMethodSessionsStatus,
			Params: NamedType{Name: "SessionTargetParams", Value: SessionTargetParams{}},
			Result: NamedType{Name: "SessionStatus", Value: SessionStatus{}},
		},
		{
			Method: HostAPIMethodSessionsEvents,
			Params: NamedType{Name: "SessionEventsParams", Value: SessionEventsParams{}},
			Result: NamedType{Name: "SessionEvent", Value: []SessionEvent{}},
		},
		{
			Method: HostAPIMethodMemoryRecall,
			Params: NamedType{Name: "MemoryRecallParams", Value: MemoryRecallParams{}},
			Result: NamedType{Name: "MemoryRecallEntry", Value: []MemoryRecallEntry{}},
		},
		{
			Method: HostAPIMethodMemoryStore,
			Params: NamedType{Name: "MemoryStoreParams", Value: MemoryStoreParams{}},
			Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
		},
		{
			Method: HostAPIMethodMemoryForget,
			Params: NamedType{Name: "MemoryForgetParams", Value: MemoryForgetParams{}},
			Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
		},
		{
			Method:         HostAPIMethodObserveHealth,
			Params:         NamedType{Name: "EmptyResult", Value: EmptyResult{}},
			Result:         NamedType{Name: "ObserveHealth", Value: ObserveHealth{}},
			OptionalParams: true,
		},
		{
			Method:         HostAPIMethodObserveEvents,
			Params:         NamedType{Name: "ObserveEventsParams", Value: ObserveEventsParams{}},
			Result:         NamedType{Name: "SessionEvent", Value: []SessionEvent{}},
			OptionalParams: true,
		},
		{
			Method:         HostAPIMethodSkillsList,
			Params:         NamedType{Name: "SkillsListParams", Value: SkillsListParams{}},
			Result:         NamedType{Name: "SkillSummary", Value: []SkillSummary{}},
			OptionalParams: true,
		},
	}
}
