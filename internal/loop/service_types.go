package loop

import (
	"context"
	"encoding/json"
	"time"

	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/task"
)

// WorkspaceID is the workspace owner for loop aggregate state.
type WorkspaceID string

// RunID is the durable loop_runs primary key.
type RunID string

// NodeID aliases the DSL graph node identity.
type NodeID = dsl.NodeID

// Inputs carries user inputs plus runtime-only start metadata.
type Inputs struct {
	Values          map[string]any `json:"values,omitempty"`
	ParentLoopRunID RunID          `json:"parent_loop_run_id,omitempty"`
	ConfigOverrides LoopConfig     `json:"config_overrides"`
}

// Status is the closed loop_runs.status vocabulary.
type Status string

const (
	// StatusQueued is a live deferred same-loop start.
	StatusQueued Status = "queued"
	// StatusRunning is a live coordinator-owned run.
	StatusRunning Status = "running"
	// StatusWatching is a dormant watch-source run.
	StatusWatching Status = "watching"
	// StatusNeedsApproval is a dormant human-gate run.
	StatusNeedsApproval Status = "needs-approval"
	// StatusPaused is a dormant operator-paused run.
	StatusPaused Status = "paused"
	// StatusDone is a verified terminal outcome.
	StatusDone Status = "done"
	// StatusNoOp is a truthful terminal no-work outcome.
	StatusNoOp Status = "no_op"
	// StatusBlocked is a terminal external-dependency outcome.
	StatusBlocked Status = "blocked"
	// StatusFailed is a terminal unrecoverable failure outcome.
	StatusFailed Status = "failed"
	// StatusExhausted is a terminal hard-limit outcome.
	StatusExhausted Status = "exhausted"
	// StatusStalled is a terminal no-progress outcome.
	StatusStalled Status = "stalled"
)

// TransitionCause records why a status transition happened.
type TransitionCause string

const (
	// TransitionCauseStart records initial run creation.
	TransitionCauseStart TransitionCause = "start"
	// TransitionCausePromote records queued-run promotion.
	TransitionCausePromote TransitionCause = "promote"
	// TransitionCauseOperatorStop records an operator stop/cancel request.
	TransitionCauseOperatorStop TransitionCause = "operator_stop"
	// TransitionCausePauseBoundary records a generation boundary honoring pause_requested.
	TransitionCausePauseBoundary TransitionCause = "pause_boundary"
	// TransitionCauseOperatorResume records operator resume.
	TransitionCauseOperatorResume TransitionCause = "operator_resume"
	// TransitionCauseApproval records a human approval.
	TransitionCauseApproval TransitionCause = "approval"
	// TransitionCauseGateRejected records a human gate rejection.
	TransitionCauseGateRejected TransitionCause = "gate_rejected"
	// TransitionCauseContract records a coordinator contract verdict.
	TransitionCauseContract TransitionCause = "contract"
	// TransitionCauseBudget records a hard budget outcome.
	TransitionCauseBudget TransitionCause = "budget"
	// TransitionCauseIterationCap records a generation iteration-cap outcome.
	TransitionCauseIterationCap TransitionCause = "iteration_cap"
	// TransitionCauseNoProgress records a no-progress outcome.
	TransitionCauseNoProgress TransitionCause = "no_progress"
	// TransitionCauseWatchPoll records a watch-source poll yielding dormancy.
	TransitionCauseWatchPoll TransitionCause = "watch_poll"
)

// StopReason captures the operator-visible stop reason.
type StopReason string

const (
	// StopReasonOperator is the default explicit operator stop.
	StopReasonOperator StopReason = "operator"
)

// GateDecision is the closed approval decision vocabulary consumed by Approve.
type GateDecision string

const (
	// GateDecisionApprove resumes the run.
	GateDecisionApprove GateDecision = "approve"
	// GateDecisionRequestChanges resumes the run after a requested revision.
	GateDecisionRequestChanges GateDecision = "request_changes"
	// GateDecisionReject terminates the run as blocked.
	GateDecisionReject GateDecision = "reject"
)

// ReattemptStrategy configures generation re-attempt breadth.
type ReattemptStrategy string

const (
	// ReattemptFailedOnly retries failed work only.
	ReattemptFailedOnly ReattemptStrategy = "failed_only"
	// ReattemptFullBody retries the whole generation body.
	ReattemptFullBody ReattemptStrategy = "full_body"
)

// LoopConfig is the raw per-loop or per-run override layer.
//
//nolint:revive // TechSpec "Core Interfaces" names this public type LoopConfig.
type LoopConfig struct {
	HumanGateEnabled  *bool               `json:"human_gate_enabled,omitempty"`
	ReattemptStrategy *ReattemptStrategy  `json:"reattempt_strategy,omitempty"`
	EnabledChecks     json.RawMessage     `json:"enabled_checks_json,omitempty"`
	IterationCap      *int                `json:"iteration_cap,omitempty"`
	BudgetTokens      *int                `json:"budget_tokens,omitempty"`
	BudgetWallSec     *int                `json:"budget_wall_sec,omitempty"`
	BudgetOnExceeded  *dsl.BudgetExceeded `json:"budget_on_exceeded,omitempty"`
	NoProgressWindow  *int                `json:"no_progress_window,omitempty"`
	FanOutWidth       *int                `json:"fan_out_width,omitempty"`
	GateMaxRevisions  *int                `json:"gate_max_revisions,omitempty"`
}

// EffectiveConfig is the fully resolved non-null runtime config.
type EffectiveConfig struct {
	HumanGateEnabled  bool               `json:"human_gate_enabled"`
	ReattemptStrategy ReattemptStrategy  `json:"reattempt_strategy"`
	EnabledChecks     json.RawMessage    `json:"enabled_checks_json"`
	IterationCap      int                `json:"iteration_cap"`
	BudgetTokens      int                `json:"budget_tokens"`
	BudgetWallSec     int                `json:"budget_wall_sec"`
	BudgetOnExceeded  dsl.BudgetExceeded `json:"budget_on_exceeded"`
	NoProgressWindow  int                `json:"no_progress_window"`
	FanOutWidth       int                `json:"fan_out_width"`
	GateMaxRevisions  int                `json:"gate_max_revisions"`
}

// LoopDefaults carries the `[loops.defaults.*]` layer consumed by the resolver.
//
//nolint:revive // Kept parallel to LoopConfig for the loop defaults layer.
type LoopDefaults struct {
	Delivery LoopConfig
	Watch    LoopConfig
}

// Run is the durable loop_run aggregate returned by the service.
type Run struct {
	ID                  RunID
	WorkspaceID         WorkspaceID
	LoopName            string
	Status              Status
	Generation          int
	ReattemptStrategy   ReattemptStrategy
	CreatedAt           time.Time
	LastProgressAt      time.Time
	StartedBy           task.ActorIdentity
	StartedOrigin       task.Origin
	ConsecutiveFailures int
	IterationCap        int
	BudgetTokens        int
	BudgetWallSec       int
	BudgetOnExceeded    dsl.BudgetExceeded
	TokensUsed          int64
	ParentLoopRunID     RunID
	PauseRequested      bool
	Inputs              map[string]any
}

// PlanNodePreview is one gen-1 node materialized by DryRun.
type PlanNodePreview struct {
	ID        dsl.NodeID    `json:"id"`
	Class     dsl.NodeClass `json:"class"`
	Kind      string        `json:"kind"`
	DependsOn []dsl.NodeID  `json:"depends_on,omitempty"`
}

// PlanPreview is the no-state preview returned by DryRun.
type PlanPreview struct {
	LoopName        string            `json:"loop_name"`
	ResolvedInputs  map[string]any    `json:"resolved_inputs"`
	Generation      int               `json:"generation"`
	Nodes           []PlanNodePreview `json:"nodes"`
	Contract        dsl.Contract      `json:"contract"`
	EffectiveConfig EffectiveConfig   `json:"effective_config"`
}

// DisplayCost is derived UI-only cost information.
type DisplayCost struct {
	Tokens           int64   `json:"tokens"`
	PricePerTokenUSD float64 `json:"price_per_token_usd"`
	USD              float64 `json:"usd"`
}

// DefinitionResolver resolves a loop name to a compiled definition.
type DefinitionResolver interface {
	ResolveLoop(ctx context.Context, ws WorkspaceID, name string) (*ResolvedDefinition, error)
}

// DefinitionResolverFunc adapts a function to DefinitionResolver.
type DefinitionResolverFunc func(context.Context, WorkspaceID, string) (*ResolvedDefinition, error)

// ResolveLoop implements DefinitionResolver.
func (f DefinitionResolverFunc) ResolveLoop(
	ctx context.Context,
	ws WorkspaceID,
	name string,
) (*ResolvedDefinition, error) {
	return f(ctx, ws, name)
}

// Store is the loop aggregate persistence contract.
type Store interface {
	CreateLoopRunForStart(ctx context.Context, run Run, policy dsl.ConcurrencyPolicy) (Run, error)
	GetLoopRun(ctx context.Context, ws WorkspaceID, runID RunID) (Run, error)
	GetLoopRunByID(ctx context.Context, runID RunID) (Run, error)
	FindActiveLoopRun(ctx context.Context, ws WorkspaceID, loopName string) (*Run, error)
	CompareAndSwapLoopRunStatus(
		ctx context.Context,
		runID RunID,
		from Status,
		to Status,
		cause TransitionCause,
		at time.Time,
	) error
	SetLoopRunPauseRequested(ctx context.Context, ws WorkspaceID, runID RunID, requested bool) error
	UpsertLoopConfig(ctx context.Context, ws WorkspaceID, loopName string, cfg LoopConfig) error
	GetLoopConfig(ctx context.Context, ws WorkspaceID, loopName string) (*LoopConfig, error)
}

// Service is the task_04 loop aggregate API surface.
type Service interface {
	Start(ctx context.Context, ws WorkspaceID, name string, inputs Inputs, actor task.ActorContext) (*Run, error)
	DryRun(ctx context.Context, ws WorkspaceID, name string, inputs Inputs) (*PlanPreview, error)
	Stop(ctx context.Context, ws WorkspaceID, runID RunID, reason StopReason) error
	Pause(ctx context.Context, ws WorkspaceID, runID RunID) error
	// Resume clears pause_requested on running runs or transitions paused runs back to running.
	Resume(ctx context.Context, ws WorkspaceID, runID RunID) error
	Approve(ctx context.Context, ws WorkspaceID, runID RunID, gateID NodeID, decision GateDecision) error
	Configure(ctx context.Context, ws WorkspaceID, name string, cfg LoopConfig) error
	GetConfig(ctx context.Context, ws WorkspaceID, name string) (*LoopConfig, error)
	Get(ctx context.Context, ws WorkspaceID, runID RunID) (*Run, error)
	Transition(ctx context.Context, runID RunID, to Status, cause TransitionCause) error
}
