package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pedronauck/agh/internal/store"
)

// LiveService exposes task-native live and run-detail reads for downstream API handlers.
type LiveService interface {
	Timeline(ctx context.Context, taskID string, query TimelineQuery, actor ActorContext) ([]TimelineItem, error)
	Stream(ctx context.Context, taskID string, query StreamQuery, actor ActorContext) (<-chan StreamEvent, error)
	Tree(ctx context.Context, taskID string, actor ActorContext) (*TreeView, error)
	RunDetail(ctx context.Context, runID string, actor ActorContext) (*RunDetailView, error)
}

// TimelineQuery captures reconnect-friendly task timeline windowing semantics.
type TimelineQuery struct {
	AfterSequence int64 `json:"after_sequence,omitempty"`
	Limit         int   `json:"limit,omitempty"`
}

// StreamQuery captures reconnect-friendly task stream replay semantics.
type StreamQuery struct {
	AfterSequence int64 `json:"after_sequence,omitempty"`
}

// EventRecordQuery captures low-level task event record reads that include a stable sequence.
type EventRecordQuery struct {
	TaskID        string `json:"task_id,omitempty"`
	AfterSequence int64  `json:"after_sequence,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Descending    bool   `json:"descending,omitempty"`
}

// EventRecord is one immutable task event plus its stable stream sequence.
type EventRecord struct {
	Sequence int64 `json:"sequence"`
	Event    Event `json:"event"`
}

// TimelineItem is the normalized task event row consumed by live task surfaces.
type TimelineItem struct {
	Sequence  int64           `json:"sequence"`
	EventID   string          `json:"event_id"`
	Task      Reference       `json:"task"`
	Run       *RunSummary     `json:"run,omitempty"`
	EventType string          `json:"event_type"`
	Actor     ActorIdentity   `json:"actor"`
	Origin    Origin          `json:"origin"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// StreamEvent is one task-scoped replayable live event suitable for SSE transport.
type StreamEvent struct {
	Sequence int64        `json:"sequence"`
	Type     string       `json:"type"`
	Timeline TimelineItem `json:"timeline"`
}

// EventObserver receives immutable task events after durable persistence.
type EventObserver interface {
	OnTaskEvent(ctx context.Context, record EventRecord)
}

// TreeView is the manager-owned live snapshot for one task tree.
type TreeView struct {
	Root        TreeNode   `json:"root"`
	Descendants []TreeNode `json:"descendants,omitempty"`
}

// TreeNode is one node inside a task-tree live snapshot.
type TreeNode struct {
	Task           Reference   `json:"task"`
	ParentTaskID   string      `json:"parent_task_id,omitempty"`
	Depth          int         `json:"depth"`
	ChildCount     int         `json:"child_count,omitempty"`
	ActiveRun      *RunSummary `json:"active_run,omitempty"`
	LastActivityAt time.Time   `json:"last_activity_at"`
}

// RunDetailView is the task-owned run detail payload for task run deep links.
type RunDetailView struct {
	Run     Run                   `json:"run"`
	Task    Reference             `json:"task"`
	Session *RunSessionRef        `json:"session,omitempty"`
	Summary RunOperationalSummary `json:"summary"`
}

// RunSessionRef links one run to its backing session when available.
type RunSessionRef struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	AgentName   string    `json:"agent_name,omitempty"`
	Name        string    `json:"name,omitempty"`
	Channel     string    `json:"channel,omitempty"`
	State       string    `json:"state,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RunOperationalSummary captures run-detail metrics aggregated from runtime data when available.
type RunOperationalSummary struct {
	LastActivityAt time.Time `json:"last_activity_at"`
	LastEventType  string    `json:"last_event_type,omitempty"`
	ToolCallCount  *int64    `json:"tool_call_count,omitempty"`
	TurnCount      *int64    `json:"turn_count,omitempty"`
	InputTokens    *int64    `json:"input_tokens,omitempty"`
	OutputTokens   *int64    `json:"output_tokens,omitempty"`
	TotalTokens    *int64    `json:"total_tokens,omitempty"`
	TotalCost      *float64  `json:"total_cost,omitempty"`
	CostCurrency   *string   `json:"cost_currency,omitempty"`
}

// RuntimeViewReader enriches run-detail reads with session and usage telemetry when available.
type RuntimeViewReader interface {
	GetSession(ctx context.Context, sessionID string) (*RunSessionRef, error)
	ListSessionEvents(ctx context.Context, sessionID string, query store.EventQuery) ([]store.SessionEvent, error)
	ListSessionTokenStats(ctx context.Context, sessionID string) ([]store.TokenStats, error)
}
