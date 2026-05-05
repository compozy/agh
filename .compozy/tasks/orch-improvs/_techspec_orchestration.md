# Child TechSpec: Orchestration Hardening

## Executive Summary

This child TechSpec hardens AGH orchestration by extending the existing autonomy substrate instead of introducing a new orchestration subsystem. It is part of the aggregate program described in [`_techspec.md`](_techspec.md), and it is authoritative for orchestration hardening details: task-run state projections, typed task execution profiles, context bundles, cursor-seeded SSE, bundled orchestration skills, max-runtime enforcement, scheduler health, durable notification cursors, and bridge-delivered terminal task notifications.

There is no `_prd.md` for this task; the primary input is `.compozy/tasks/orch-improvs/analysis/`, the archived autonomy/supervisor prior art under `.compozy/tasks/_archived/`, codebase exploration, and the accepted ADRs in `.compozy/tasks/orch-improvs/adrs/`.

MVP boundary: this child TechSpec includes Core 1-7 from the orchestration-improvements synthesis plus task execution profiles, task context bundle enrichment, cursor-seeded task SSE, bundled orchestration skills, and durable notifier cursors. Bulk task endpoints, frontend extension/plugin SDKs, generic event-bus architecture, new orchestration queues, dedicated per-task coordinators, scheduler-owned claims, channel-owned terminal state, and review-gate state machines are deferred or explicitly out of scope unless a later TechSpec pulls them in.

The implementation preserves `task.Service`, `task_runs`, session-bound lease lookup, mechanical scheduler boundaries, coordinator runtime, typed hooks, and `/agent/context` as the core architecture. The primary trade-off is accepting additional typed state, migrations, and contract/codegen work in exchange for queryable orchestration behavior, resumable web streams, deterministic coordinator guidance, and durable notification delivery cursors without creating a second queue or prompt-owned authority.

## Non-Goals

- Do not introduce a second orchestration queue, dispatcher-owned assignment path, or scheduler-owned claim path.
- Do not make coordination channels, bridge threads, skills, prompt overlays, or notification cursors authoritative for ownership or terminal state.
- Do not implement dedicated per-task coordinator sessions in MVP. Task-specific coordinator behavior is `guided` policy for the existing daemon-managed workspace coordinator.
- Do not store task-specific coordinator, worker, participant, review, or sandbox selection only in `metadata_json`.
- Do not let participant policy grant channel permissions, task ownership, claim eligibility by itself, review verdict authority, or terminal-state permission.
- Do not let task-level sandbox selection bypass tool policy, approval policy, provider authorization, or session authorization.
- Do not implement a review gate, reviewer verdict state, review rounds, `pending_review` run status, or reviewer channel policy in this child spec. Those belong to [`_techspec_review_gate.md`](_techspec_review_gate.md).
- Do not use `internal/notifications` as a review workflow bus. Notification cursors remain delivery progress only.

## System Architecture

### Component Overview

| Component | Responsibility | Boundary |
|-----------|----------------|----------|
| `internal/task` service | Own task-run transitions, summaries, runtime projection fields, task execution profile validation, spawn-failure counters, max-runtime enforcement, and task events. | Only `task.Service`/store transition paths may mutate authoritative task/run state. |
| `internal/store/globaldb` | Persist task/run schema changes, task execution profile tables, bridge task subscriptions, notification cursors, and projection fields through numbered migrations. | No boot-time schema reconciliation or compatibility fallback. |
| `internal/api/core` + transports | Expose shared HTTP/UDS handlers for task/run/context/dashboard changes. | Public surfaces must never expose raw claim tokens. |
| `internal/daemon/native_tools.go` + built-in tools | Keep native tool parity for task claim/heartbeat/complete/fail/release and summary inputs. | Tools use session-bound lease lookup, not raw tokens. |
| `internal/situation` | Enrich `/agent/context` with bounded task-run context bundles. | Extends existing situation context; does not create a parallel context system. |
| `internal/session` | Start worker/reviewer sessions with effective task agent/provider/model/sandbox selection. | Session start applies profile input; it does not own task authority or bypass permission checks. |
| `internal/scheduler` | Add read-only health telemetry and max-runtime recovery triggers while preserving wake/recover-only behavior. | Scheduler must not claim, assign, or complete work directly. |
| `internal/coordinator` + daemon coordinator runtime | Load deterministic orchestrator instructions, consume `CoordinatorProfile.mode = "guided"`, and keep coordinator bootstrap behavior. | Coordinator guides work; `task_runs` remains authority. |
| `internal/skills/bundled` | Add `agh-task-worker` and `agh-orchestrator`. | Skills are instructional only, never security or ownership boundaries. |
| `internal/notifications` | New shared durable notification cursor primitive for confirmed delivery progress. | Cursor state is delivery progress only, not task ownership, event fan-out policy, hook dispatch, or queue semantics. |
| `internal/bridges` | First concrete MVP consumer of `internal/notifications`: bridge-delivered terminal task notifications. | Owns bridge subscription/delivery target state; reads durable task events and advances notification cursors only after confirmed bridge delivery. |
| `web/src/systems/tasks` | Consume current-run projection, summaries, scheduler health, and cursor-seeded SSE. | Use generated contract types; do not invent a parallel task event model. |
| `packages/site` + generated docs | Document config, CLI/API changes, and runtime behavior. | Docs must reflect runtime truth, not aspirational orchestration behavior. |

### Data Flow

1. A task is created or updated with an optional `TaskExecutionProfile`.
2. `task.Service` validates the profile against config bounds, task scope, channel/peer availability, sandbox policy, and known agent/provider/model references.
3. A task run is enqueued through existing task APIs.
4. `task.Service` persists the run, emits task events, and updates task-level read projections.
5. Coordinator runtime observes eligible enqueued runs and creates/reuses a daemon-managed coordinator session when policy allows.
6. Coordinator sessions receive runtime bootstrap facts, deterministic `agh-orchestrator` skill content, and task-specific guided profile data when `CoordinatorProfile.mode = "guided"`.
7. Worker session start resolves `WorkerProfile`, `ParticipantPolicy`, and `SandboxPolicy` into effective agent/provider/model/sandbox options, falling back to workspace defaults only when the profile mode is `inherit`.
8. Worker sessions use `/agent/context`, task claim APIs/tools, heartbeats, coordination channels, and `agh-task-worker` guidance.
9. Complete/fail/release paths resolve the caller's active lease through session-bound lookup, never through public raw claim tokens.
10. `task.Service` records bounded `summary`, terminal result/error state, task events, `current_run_id`, spawn-failure counters, and max-runtime outcomes.
11. `/agent/context`, task detail/list/dashboard payloads, and web task surfaces read the queryable projections.
12. Task streams continue to replay from `after_sequence`; task read/list payloads expose `latest_event_seq` so web clients can start SSE without a race.
13. `internal/bridges` delivers one-shot terminal task notifications to bridge targets by replaying durable `task_events` and using `internal/notifications` cursors for confirmed delivery progress only.

## Architectural Boundaries

Implementation must preserve AGH's package boundaries and daemon composition-root rule:

1. `internal/task` owns task/run state transitions, task events, summaries, runtime projections, spawn-failure counters, and max-runtime outcomes. It must not import `internal/scheduler`, `internal/coordinator`, `internal/daemon`, or `web`.
2. `internal/store/globaldb` persists task/run and notification cursor state through numbered migrations. It must not contain orchestration policy or session-control decisions.
3. `internal/scheduler` depends only on narrow task/session/waker interfaces. It must not receive `ClaimNextRun`, terminal mutation, raw claim-token access, or coordinator policy authority.
4. `internal/coordinator` expresses coordinator decisions, prompt overlays, and policy helpers. It must not become task ownership authority, queue authority, or terminal-state authority.
5. `internal/daemon` remains the composition root that wires scheduler, coordinator runtime, situation context, hooks, skills, notifications, and transports.
6. `internal/situation` enriches the existing bounded `/agent/context` payload. It must not introduce a parallel context or memory system.
7. `internal/notifications` owns delivery cursor state only. It must not become a generic event bus, hook dispatcher, task queue, ownership state store, or event fan-out policy surface.
8. `internal/bridges` owns bridge subscription and delivery target state for the first MVP notification consumer. It must not use channel/thread state as replay authority; replay is always based on durable `task_events.event_seq`.
9. `internal/api/core` remains the shared transport implementation for HTTP and UDS parity. Transport packages mount shared handlers instead of duplicating semantics.
10. `internal/skills/bundled` stores instructional content. Skills must not define permissions, ownership, queue semantics, or terminal state.
11. `web/src/systems/tasks` consumes generated contracts and task SSE. It must not invent a separate task event model or infer authority from UI state.
12. `TaskExecutionProfile` belongs to `internal/task` and is stored as typed columns/side tables. It must not be reconstructed from `metadata_json`.
13. `internal/session` may receive effective task profile options from the daemon composition root, but it must not query or mutate task authority directly.
14. `CoordinatorProfile.mode = "guided"` supplies task-specific guidance to the existing coordinator runtime. Dedicated coordinator sessions require a later TechSpec.

### Safety Invariants

1. `task_runs` is the only durable execution queue and ownership source.
2. `ClaimNextRun` remains the only authoritative next-work claim primitive.
3. Raw `claim_token` never crosses HTTP, UDS responses, SSE, channels, logs, web state, skills, memory, or docs examples.
4. Agent heartbeat, complete, fail, and release resolve the caller's active lease through session-bound lookup.
5. Human/operator terminal mutations for token-fenced runs continue to fail unless they go through the existing protected task-service path.
6. `tasks.current_run_id` is a denormalized read projection only; it is never scheduler assignment authority, coordinator ownership authority, claim authority, or terminal-state authority.
7. Scheduler code may recover expired leases and wake sessions, but it must never claim, complete, fail, release, or assign task runs.
8. Coordinator code may guide, spawn, and coordinate sessions, but terminal state is written only by task-service-owned transitions.
9. Coordination channels carry conversation, handoff, blocker, and result content; they never define task ownership or terminal state.
10. Bounded `summary` inputs are validated at API/tool/CLI/service boundaries before persistence.
11. Notification cursors advance only after confirmed delivery and never substitute for task event sequence, task hooks, or task-run ownership.
12. Every schema change ships as a numbered migration with fresh-DB and migrated-DB tests.
13. Task execution profile updates are task-service-owned transitions and emit bounded task events.
14. `WorkerProfile` may constrain worker eligibility, but a worker still must claim through `ClaimNextRun` and mutate through session-bound lease lookup.
15. `ParticipantPolicy` never grants channel permission; channel membership and peer authorization remain enforced by network/bridge owners.
16. `SandboxPolicy` selects the session sandbox only at session start and never bypasses approval policy, tool allowlists, provider authorization, or session authorization.
17. `CoordinatorProfile.mode = "guided"` never creates a second coordinator queue, claim path, or per-task coordinator authority.

Cursor advancement invariants:

1. `internal/notifications` stores one cursor per `(consumer_id, stream_name, subject_id)`; `subject_id = ""` represents an unscoped stream.
2. Cursor advancement is monotonic. `Advance` refuses any `last_sequence` lower than or equal to the stored sequence with a typed `ErrNonMonotonicCursor` error unless the payload is an idempotent replay of the same confirmed delivery metadata.
3. `Reset` is the only operation allowed to lower a cursor, and it requires a non-empty operator or daemon recovery reason.
4. Callers advance only after the delivery path returns success. When both durable delivery recording and cursor advancement are SQLite writes, they must execute in the same global DB transaction.
5. External bridge delivery remains at-least-once: if a process crashes after external delivery but before `Advance`, replay may duplicate delivery, but it must not skip undelivered events.
6. Idempotent replay is accepted only when `last_sequence` and `delivery_id` match the cursor row's last confirmed `last_sequence` and `last_delivery_id`.

`tasks.current_run_id` transition invariants:

1. `ClaimNextRun` sets `tasks.current_run_id = run.id` in the same transaction that claims the run.
2. `StartRun` and `AttachRunSession` preserve the pointer and must fail if the pointer references a different active run.
3. `CompleteRunLease` clears the pointer in the same transaction as the completed terminal run transition.
4. `FailRunLease` clears the pointer in the same transaction as the failed terminal run transition.
5. `ReleaseRunLease` and `ReleaseSessionRunLeases` clear the pointer when the run is returned to the queue or terminally released.
6. `RecoverExpiredRunLeases` clears the pointer for every recovered stale lease before making the run claimable again.
7. Synthetic terminal run creation sets the pointer to the synthetic run and clears it in the same transaction that writes the terminal task/run state.
8. Task cancel, archive, delete, and terminal close paths clear the pointer.
9. `task.Service` is the only projection writer. Store helpers may expose set/clear primitives, but only task-service-owned transition methods may call them.

## Implementation Design

### Core Interfaces

The following signatures are the target contracts for task generation. `OrchestrationManager` is a logical decomposition of the existing `task.Manager` / `task.Service` implementation; it must not introduce a second service struct, second queue authority, or second daemon composition-root dependency. Existing exported interfaces should be extended only where the surrounding package already exposes that capability; otherwise implementation may keep the helper unexported, but the request/response structs and invariants must remain equivalent.

Task orchestration state stays under `internal/task`. The service owns summaries, synthetic terminal runs, spawn-failure breaker state, max-runtime terminal writes, and the `current_run_id` projection:

```go
type OrchestrationManager interface {
	CompleteTask(ctx context.Context, taskID string, completion TaskCompletion, actor ActorContext) (*TaskTerminalResult, error)
	FailTask(ctx context.Context, taskID string, failure TaskFailure, actor ActorContext) (*TaskTerminalResult, error)
	RecordRunSummary(ctx context.Context, runID string, summary RunSummaryInput, actor ActorContext) (*Run, error)
	IncrementSpawnFailure(ctx context.Context, failure SpawnFailure, actor ActorContext) (*Task, error)
	ResetSpawnFailure(ctx context.Context, reset SpawnFailureReset, actor ActorContext) (*Task, error)
	EnforceMaxRuntime(ctx context.Context, exceeded MaxRuntimeExceeded, actor ActorContext) (*Run, error)
}

type TaskCompletion struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Summary string          `json:"summary,omitempty"`
	Now     time.Time       `json:"now"`
}

type TaskFailure struct {
	Error    string          `json:"error"`
	Summary  string          `json:"summary,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Now      time.Time       `json:"now"`
}

type TaskTerminalResult struct {
	Task         Task `json:"task"`
	Run          Run  `json:"run"`
	SyntheticRun bool `json:"synthetic_run,omitempty"`
}

type RunSummaryInput struct {
	Summary string    `json:"summary"`
	Now     time.Time `json:"now"`
}

type SpawnFailureReason string

const (
	SpawnFailureReasonSpawnFailed        SpawnFailureReason = "spawn_failed"
	SpawnFailureReasonSessionUnreachable SpawnFailureReason = "session_unreachable"
	SpawnFailureReasonProviderAuth       SpawnFailureReason = "provider_auth"
)

type SpawnFailure struct {
	TaskID string             `json:"task_id"`
	RunID  string             `json:"run_id,omitempty"`
	Reason SpawnFailureReason `json:"reason"`
	Error  string             `json:"error"`
	Now    time.Time          `json:"now"`
}

type SpawnFailureReset struct {
	TaskID string    `json:"task_id"`
	RunID  string    `json:"run_id,omitempty"`
	Reason string    `json:"reason"`
	Now    time.Time `json:"now"`
}

type MaxRuntimeExceeded struct {
	TaskID            string        `json:"task_id"`
	RunID             string        `json:"run_id"`
	SessionID         string        `json:"session_id"`
	StartedAt         time.Time     `json:"started_at"`
	MaxRuntimeSeconds int64         `json:"max_runtime_seconds"`
	GracePeriod       time.Duration `json:"grace_period"`
	Now               time.Time     `json:"now"`
}

const StopReasonTimedOut StopReason = "timed_out"

type CurrentRunProjectionMutation struct {
	TaskID        string    `json:"task_id"`
	RunID         string    `json:"run_id"`
	Set           bool      `json:"set"`
	Transition    string    `json:"transition"`
	ExpectedRunID string    `json:"expected_run_id,omitempty"`
	Now           time.Time `json:"now"`
}
```

`CurrentRunProjectionMutation` is shown only to pin the internal helper shape. Projection set/clear helpers must remain unexported inside `internal/task` and callable only from task-service transition methods. Boundary tests must fail if scheduler, coordinator, API, bridge, notification, extension, or web-facing code imports or calls the projection helper directly.

`RunResult`, `RunFailure`, `Run`, `RunSummary`, task read DTOs, and agent task request DTOs gain bounded `summary`/orchestration fields:

```go
type RunResult struct {
	Value   json.RawMessage `json:"value,omitempty"`
	Summary string          `json:"summary,omitempty"`
}

type RunFailure struct {
	Error    string          `json:"error"`
	Summary  string          `json:"summary,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type TaskRuntimeLimits struct {
	MaxRuntimeSeconds int64 `json:"max_runtime_seconds"`
	SummaryMaxBytes   int   `json:"summary_max_bytes"`
	ContextMaxBytes   int   `json:"context_body_max_bytes"`
}

type TaskOrchestrationFields struct {
	CurrentRunID      string            `json:"current_run_id,omitempty"`
	LatestEventSeq    int64             `json:"latest_event_seq"`
	MaxRuntimeSeconds int64             `json:"max_runtime_seconds"`
	SpawnFailureCount int               `json:"spawn_failure_count"`
	LastSpawnError    string            `json:"last_spawn_error,omitempty"`
	CurrentRun        *RunSummary       `json:"current_run,omitempty"`
	RuntimeLimits     TaskRuntimeLimits `json:"runtime_limits"`
}
```

Task execution profiles stay task-owned and typed:

```go
type TaskExecutionProfile struct {
	Coordinator  CoordinatorProfile  `json:"coordinator,omitempty"`
	Worker       WorkerProfile       `json:"worker,omitempty"`
	Review       ReviewProfile       `json:"review,omitempty"`
	Participants ParticipantPolicy   `json:"participants,omitempty"`
	Sandbox      SandboxPolicy       `json:"sandbox,omitempty"`
}

type CoordinatorProfile struct {
	Mode          CoordinatorMode `json:"mode"`
	AgentName     string          `json:"agent_name,omitempty"`
	Provider      string          `json:"provider,omitempty"`
	Model         string          `json:"model,omitempty"`
	Guidance      string          `json:"guidance,omitempty"`
}

type WorkerProfile struct {
	Mode                WorkerMode `json:"mode"`
	AgentName           string     `json:"agent_name,omitempty"`
	Provider            string     `json:"provider,omitempty"`
	Model               string     `json:"model,omitempty"`
	AllowedAgentNames   []string   `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames []string   `json:"preferred_agent_names,omitempty"`
	RequiredCapabilities []string  `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string `json:"preferred_capabilities,omitempty"`
}

type ParticipantPolicy struct {
	AllowedChannelIDs     []string `json:"allowed_channel_ids,omitempty"`
	PreferredChannelIDs   []string `json:"preferred_channel_ids,omitempty"`
	AllowedPeerIDs        []string `json:"allowed_peer_ids,omitempty"`
	PreferredPeerIDs      []string `json:"preferred_peer_ids,omitempty"`
	AllowedAgentNames     []string `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames   []string `json:"preferred_agent_names,omitempty"`
	RequiredCapabilities  []string `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string `json:"preferred_capabilities,omitempty"`
}

type SandboxPolicy struct {
	Mode       SandboxMode `json:"mode"`
	SandboxRef string      `json:"sandbox_ref,omitempty"`
}
```

Allowed mode values:

- `CoordinatorMode`: `inherit`, `guided`.
- `WorkerMode`: `inherit`, `select`.
- `SandboxMode`: `inherit`, `none`, `ref`.

`ReviewProfile` is defined by [`_techspec_review_gate.md`](_techspec_review_gate.md) and uses the same task-owned profile row plus review-specific selector fields. `CoordinatorModeDedicated` is not part of MVP.

The shared API contract must expose these fields through generated DTOs:

```go
type CompleteTaskRunRequest struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Summary string          `json:"summary,omitempty"`
}

type FailTaskRunRequest struct {
	Error    string          `json:"error"`
	Summary  string          `json:"summary,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type CompleteTaskRequest struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Summary string          `json:"summary,omitempty"`
}

type FailTaskRequest struct {
	Error    string          `json:"error"`
	Summary  string          `json:"summary,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type CreateBridgeTaskNotificationRequest struct {
	BridgeInstanceID string       `json:"bridge_instance_id"`
	PeerID           string       `json:"peer_id,omitempty"`
	ThreadID         string       `json:"thread_id,omitempty"`
	GroupID          string       `json:"group_id,omitempty"`
	DeliveryMode     DeliveryMode `json:"delivery_mode"`
}

type BridgeTaskNotificationSubscriptionPayload struct {
	Subscription BridgeTaskSubscription `json:"subscription"`
	Cursor       Cursor                 `json:"cursor"`
}
```

`Scope`, `DeliveryMode`, and `ActorIdentity` should reuse existing runtime/bridge contract types where they already exist. If an implementation package lacks one of these types, introduce it in the owning runtime or bridge package rather than defining API-only duplicates.

The primary new shared persistence contract is the notification cursor primitive:

```go
type CursorStore interface {
	Get(ctx context.Context, key CursorKey) (Cursor, error)
	Advance(ctx context.Context, update AdvanceCursor) (Cursor, error)
	Reset(ctx context.Context, key CursorKey, reason string) (Cursor, error)
	List(ctx context.Context, query CursorQuery) ([]Cursor, error)
}

type CursorKey struct {
	ConsumerID string `json:"consumer_id"`
	StreamName string `json:"stream_name"`
	SubjectID  string `json:"subject_id"`
}

type Cursor struct {
	Key             CursorKey `json:"key"`
	LastSequence    int64     `json:"last_sequence"`
	LastDeliveryID  string    `json:"last_delivery_id,omitempty"`
	LastDeliveredAt time.Time `json:"last_delivered_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AdvanceCursor struct {
	Key             CursorKey `json:"key"`
	LastSequence    int64     `json:"last_sequence"`
	LastDeliveredAt time.Time `json:"last_delivered_at"`
	DeliveryID      string    `json:"delivery_id,omitempty"`
	Now             time.Time `json:"now"`
}

type CursorQuery struct {
	ConsumerID string `json:"consumer_id,omitempty"`
	StreamName string `json:"stream_name,omitempty"`
	SubjectID  string `json:"subject_id,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}
```

The first MVP consumer of `CursorStore` is bridge-delivered terminal task notifications owned by `internal/bridges`:

```go
type BridgeTaskSubscription struct {
	SubscriptionID   string        `json:"subscription_id"`
	TaskID           string        `json:"task_id"`
	BridgeInstanceID string        `json:"bridge_instance_id"`
	Scope            Scope         `json:"scope"`
	WorkspaceID      string        `json:"workspace_id,omitempty"`
	PeerID           string        `json:"peer_id,omitempty"`
	ThreadID         string        `json:"thread_id,omitempty"`
	GroupID          string        `json:"group_id,omitempty"`
	DeliveryMode     DeliveryMode  `json:"delivery_mode"`
	CreatedBy        ActorIdentity `json:"created_by"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type TerminalTaskNotification struct {
	DeliveryID     string          `json:"delivery_id"`
	EventType      string          `json:"event_type"`
	Final          bool            `json:"final"`
	Seq            int64           `json:"seq"`
	TaskID         string          `json:"task_id"`
	RunID          string          `json:"run_id,omitempty"`
	Status         Status          `json:"status"`
	Summary        string          `json:"summary,omitempty"`
	Error          string          `json:"error,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	SubscriptionID string          `json:"subscription_id"`
}
```

Cursor identity for this consumer is fixed:

- `consumer_id = "bridge_task_subscription:<subscription_id>"`
- `stream_name = "task_events"`
- `subject_id = <task_id>`

Task context enrichment should extend the existing situation service with bounded task-specific state:

```go
type TaskContextBundleAssembler interface {
	BundleForActiveLease(ctx context.Context, req TaskContextRequest) (TaskContextBundle, error)
	BundleForOperatorTask(ctx context.Context, req OperatorTaskContextRequest) (TaskContextBundle, error)
}

type TaskContextRequest struct {
	SessionID string    `json:"session_id"`
	RunID     string    `json:"run_id,omitempty"`
	Now       time.Time `json:"now"`
}

type OperatorTaskContextRequest struct {
	TaskID string    `json:"task_id"`
	Now    time.Time `json:"now"`
}

type TaskContextBundle struct {
	Task           Reference
	CurrentRun     *RunSummary
	PriorAttempts  []RunSummary
	RecentEvents   []TimelineItem
	HandoffSummary string
	Limits         TaskRuntimeLimits
}
```

Scheduler max-runtime enforcement must use a narrow request interface instead of direct terminal writes:

```go
type RuntimeRecoverySink interface {
	RequestMaxRuntimeRecovery(ctx context.Context, exceeded MaxRuntimeExceeded) error
	RecordSchedulerHealth(ctx context.Context, sample SchedulerHealthSample) error
}

type SchedulerHealthSample struct {
	ObservedAt       time.Time     `json:"observed_at"`
	BadTickCount     int           `json:"bad_tick_count"`
	CooldownUntil     time.Time     `json:"cooldown_until,omitempty"`
	StuckRunCount     int           `json:"stuck_run_count"`
	NotificationLag   time.Duration `json:"notification_lag,omitempty"`
	LastSchedulerError string        `json:"last_scheduler_error,omitempty"`
}
```

Coordinator bootstrap must load `agh-orchestrator` through a deterministic bundled-skill loader:

```go
type BundledSkillLoader interface {
	LoadBundledSkill(ctx context.Context, name string) (BundledSkillContent, error)
}

type BundledSkillContent struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Body    string `json:"body"`
}

type CoordinatorPromptAssembler interface {
	AssembleCoordinatorPrompt(ctx context.Context, input CoordinatorPromptInput) (string, error)
}

type CoordinatorPromptInput struct {
	RuntimeFacts      coordinator.PromptInput
	OrchestratorSkill BundledSkillContent
}
```

Assembly contract: the skill body is prepended before the runtime prompt overlay. `coordinator.PromptOverlay` keeps only runtime facts and public API hints; stable orchestration guidance lives in `agh-orchestrator` and must not be duplicated in the overlay.

### Data Models

#### `tasks`

Add explicit typed fields through a numbered migration:

| Column | Type | Semantics |
|--------|------|-----------|
| `current_run_id` | `TEXT REFERENCES task_runs(id) ON DELETE SET NULL` | Denormalized read projection only. Never claim/assignment authority. |
| `max_runtime_seconds` | `INTEGER NOT NULL DEFAULT 0` | Per-task runtime watchdog override. Zero disables task-specific limit. |
| `spawn_failure_count` | `INTEGER NOT NULL DEFAULT 0` | Task-service-owned spawn circuit breaker counter. |
| `last_spawn_error` | `TEXT NOT NULL DEFAULT ''` | Latest bounded spawn failure summary. |

`current_run_id` is maintained only by `task.Service`/store transition paths. `task_runs` remains the sole authoritative execution queue and ownership source.

`max_runtime_seconds` stores seconds because the DB field is queried as an integer. The config layer parses `[task.orchestration].default_max_runtime` from a duration string into seconds before constructing task-service defaults. Validation rejects negative durations, fractional seconds, and values greater than the implementation's documented maximum watchdog budget.

Spawn failure policy:

1. Increment `spawn_failure_count` only for typed reasons `spawn_failed`, `session_unreachable`, and `provider_auth`.
2. `internal/session.Manager` is the source of truth for spawn-failure classification. Coordinator runtime and scheduler code may observe the failed start, but they pass the error through session classification instead of inventing their own reason mapping.
3. The session start/attach path calls `task.Service.IncrementSpawnFailure(ctx, SpawnFailure{...})` synchronously after classifying a failed task-session spawn.
4. Reset `spawn_failure_count` and `last_spawn_error` from `AttachRunSession` after the task session is successfully attached. Claim alone does not reset because a claimed run can still fail to spawn.
5. When the counter reaches `[task.orchestration].spawn_failure_limit`, task-service-owned transition logic moves the task to `blocked` with `last_spawn_error` preserved and emits `task.spawn_failure_circuit_opened`.
6. `ClaimNextRun` must not return runs for tasks whose spawn circuit is open.
7. Scheduler/session code observes spawn outcomes and calls task-service methods; it must never mutate the counter directly.

`ClaimNextRun` spawn-circuit filter design:

1. The claim query joins `tasks` by `task_runs.task_id`.
2. It excludes rows where `tasks.status = 'blocked'` because a circuit-open task is no longer claimable.
3. It excludes rows where `tasks.spawn_failure_count >= [task.orchestration].spawn_failure_limit` as a belt-and-suspenders guard during transition races.
4. The migration adds a covering index for the joined claim path, for example `(status, spawn_failure_count, id)` on `tasks` plus the existing task-run pending-claim index, and the implementation must verify the SQLite query plan in a store test.
5. `IncrementSpawnFailure` increments the counter and opens the circuit in one `BEGIN IMMEDIATE` transaction; there is no separate async breaker opener.
6. `AttachRunSession` resets the breaker state in the same transaction that attaches the session to the run.
7. Tests must prove a circuit-open task is not returned by `ClaimNextRun` and a successfully attached retry clears the breaker durably.

Review-driven continuation runs are defined by the review-gate child spec, but their execution rows still live in `task_runs`. Migration ownership is intentionally pinned to the review-gate child spec: the review-gate migration creates `task_run_reviews` and the `task_runs` review trigger/continuation columns that reference it in the same numbered migration. This orchestration child must not create `task_runs.review_id`, `task_runs.review_request_id`, or any FK to `task_run_reviews`; doing so would create a migration ordering hazard.

The orchestration child still owns the non-review run fields and invariants that all continuation runs inherit: `task_runs` remains the single execution queue, claim/session fields remain token-fenced, profile selection still follows `TaskExecutionProfile`, and `metadata_json` must not hold queryable review trigger or continuation-run state.

Synthetic terminal runs are in MVP scope. When an operator task-level `CompleteTask` or `FailTask` request targets a task with no active non-terminal run, `task.Service` creates one zero-duration `task_runs` row before writing the terminal task state:

1. `started_at == ended_at == now`.
2. `status` is `completed` or `failed` according to the request.
3. `attempt` is the next free attempt number for the task.
4. `claimed_by` is the server-derived actor from the request.
5. `origin` is the request origin.
6. `summary`, `result`, `error`, and `metadata` are copied from the task-level terminal request after normal validation/redaction.
7. `current_run_id` is set to the synthetic run and cleared in the same transaction.
8. Synthetic runs are never claimable and never expose raw claim-token fields.

Synthetic terminal concurrency contract:

1. Synthetic run creation executes inside `BEGIN IMMEDIATE`.
2. The write uses a `WHERE NOT EXISTS (SELECT 1 FROM task_runs WHERE task_id = ? AND status NOT IN ('completed', 'failed', 'canceled'))` guard and a task-status compare-and-set on the non-terminal source state.
3. The synthetic attempt number is computed while holding that write transaction.
4. If another request already terminalized the task, the second request returns the existing terminal task and latest terminal run without inserting a run or emitting a second terminal event.
5. Concurrent `CompleteTask` and `FailTask` requests are resolved by the first successful task-status CAS; later requests are idempotent reads, not conflicting terminal writes.

#### Task Execution Profiles

Create a task-owned profile table:

| Field | Semantics |
|-------|-----------|
| `task_id` | Primary key and foreign key to `tasks.id`. |
| `coordinator_mode` | `inherit` or `guided`. MVP excludes `dedicated`. |
| `coordinator_agent_name` | Optional task-specific coordinator agent hint for guided mode. |
| `coordinator_provider` | Optional provider hint for guided mode. |
| `coordinator_model` | Optional model hint for guided mode. |
| `coordinator_guidance` | Bounded task-specific coordinator guidance. |
| `worker_mode` | `inherit` or `select`. |
| `worker_agent_name` | Optional exact worker agent for session start. |
| `worker_provider` | Optional worker provider override. |
| `worker_model` | Optional worker model override. |
| `review_agent_name` | Optional reviewer agent hint consumed by the review-gate child spec. |
| `review_provider` | Optional reviewer provider hint. |
| `review_model` | Optional reviewer model hint. |
| `sandbox_mode` | `inherit`, `none`, or `ref`. |
| `sandbox_ref` | Required when `sandbox_mode = "ref"`; empty otherwise. |
| `created_at` | Row creation timestamp. |
| `updated_at` | Row update timestamp. |

Table name: `task_execution_profiles`.

Create selector side tables for matchable profile state:

| Table | Fields | Semantics |
|-------|--------|-----------|
| `task_profile_agents` | `task_id`, `role`, `preference`, `agent_name` | Worker/review/participant allowed or preferred agents. |
| `task_profile_channels` | `task_id`, `role`, `preference`, `channel_id` | Review/participant allowed or preferred coordination channels. |
| `task_profile_peers` | `task_id`, `role`, `preference`, `peer_id` | Review/participant allowed or preferred peers. |
| `task_profile_capabilities` | `task_id`, `role`, `preference`, `capability_id` | Worker/review/participant required or preferred capabilities. |

Allowed `role` values are `worker`, `review`, and `participant`. Allowed `preference` values are `required`, `allowed`, and `preferred`; invalid combinations such as `required` peer rows must be rejected at validation time.

Required indexes:

- `task_execution_profiles_task_id_idx` on `(task_id)`.
- `task_profile_agents_lookup_idx` on `(role, preference, agent_name, task_id)`.
- `task_profile_channels_lookup_idx` on `(role, preference, channel_id, task_id)`.
- `task_profile_peers_lookup_idx` on `(role, preference, peer_id, task_id)`.
- `task_profile_capabilities_lookup_idx` on `(role, preference, capability_id, task_id)`.

Side-table-vs-JSON decision: profile selectors use side tables because claim eligibility, reviewer routing, dashboard filtering, and profile validation need exact-match predicates. `metadata_json` remains an opaque extension payload and must not store runtime selection. `coordinator_guidance` is a bounded text field because it is prompt guidance, not a query dimension.

Profile precedence:

1. Task execution profile field, when set and permitted by config.
2. Task/run native fields such as `network_channel`, `coordination_channel_id`, `required_capabilities`, and `preferred_capabilities`.
3. Workspace defaults such as `DefaultAgent` and `SandboxRef`.
4. Global config defaults.

Continuation-run precedence:

1. A review-created continuation run uses the task's current `TaskExecutionProfile` at enqueue time for worker, participant, coordinator-guidance, review, and sandbox selection.
2. The reviewed run's native coordination/capability fields are copied forward only when the task profile does not set equivalent worker or participant selectors.
3. Continuation columns (`parent_run_id`, `review_id`, `review_round`, `continuation_reason`, `missing_work_json`, `next_round_guidance`) provide context and lineage only. They do not override worker/profile selection and do not grant permissions.

Task profile validation:

1. `coordinator_mode` must be `inherit` or `guided`; `dedicated` is rejected.
2. `coordinator_agent_name`, `coordinator_provider`, and `coordinator_model` are guidance for the existing coordinator only.
3. `worker_agent_name` must be compatible with `allowed_agent_names` when both are set.
4. `sandbox_mode = "none"` is accepted only when config allows task-level sandbox disabling.
5. `sandbox_mode = "ref"` requires a non-empty `sandbox_ref` that resolves through existing sandbox/workspace validation.
6. Provider/model overrides are accepted only when config allows task-level provider override.
7. Channel and peer ids must be validated through the existing network/bridge validators where available.
8. Profile updates for active token-fenced runs are rejected unless they affect future runs only and are recorded as future-effective profile changes. MVP should prefer rejecting active-run profile mutation to avoid hidden runtime drift.

Claim/session behavior:

1. `ClaimNextRun` must filter out workers whose `agent_name` is not eligible for the task profile when the claiming session supplies an agent identity.
2. A run with a required worker agent should remain unclaimable by other agent names even if they match capability filters.
3. Preferred agents/capabilities affect ordering only after authority filters pass.
4. Task worker session start must pass effective `AgentName`, `Provider`, `Model`, and sandbox selection into session create/start options.
5. Workspace defaults are used only after task profile and task/run fields resolve to inherit/empty.

ParticipantPolicy enforcement:

1. Profile validation resolves configured channel, peer, agent, and capability selectors through existing network/bridge/agent validators where available. Unknown explicit selectors fail with deterministic profile validation errors.
2. Coordinator routing must intersect requested coordination/review channels and peers with `ParticipantPolicy.Allowed*` when those lists are non-empty. A route outside the allowed set returns `ErrParticipantPolicyViolation`; preferred lists affect ordering only after allowed checks pass.
3. Worker claim filters use participant and worker agent/capability selectors as narrowing predicates. They cannot widen claim eligibility beyond `ClaimNextRun`, task scope, run status, or lease rules.
4. Safe-spawn and session-start grants for network channels, peers, agents, skills, and capabilities must be a subset of both the parent session/tool policy and the task participant policy. Unknown child atoms count as widening and reject.
5. Review routing must satisfy both `TaskExecutionProfile.Review` and `ParticipantPolicy` when participant allowed lists are set. Review verdict authority still requires a persisted review binding and `RecordRunReview`.
6. Network/bridge packages continue to enforce actual channel membership and peer authorization. `ParticipantPolicy` is an upper-bound selection policy, not a permission grant.
7. Tests must prove a task profile cannot use participant policy to gain channel access, widen a bridge target, claim work as an ineligible agent, or bypass review binding.

#### `task_runs`

Add:

| Column | Type | Semantics |
|--------|------|-----------|
| `summary` | `TEXT NOT NULL DEFAULT ''` | Bounded worker/coordinator handoff or terminal run summary. |
| `claimed_agent_name` | `TEXT NOT NULL DEFAULT ''` | Agent name captured when the run is claimed or session-bound; used for audit and review self-review exclusion. |
| `claimed_peer_id` | `TEXT NOT NULL DEFAULT ''` | Peer id captured when the run is claimed by a network peer; empty for local-only runs. |
| `terminalized_by_session_id` | `TEXT NOT NULL DEFAULT ''` | Session id that wrote the terminal run transition, when any. |
| `terminalized_by_agent_name` | `TEXT NOT NULL DEFAULT ''` | Agent name that wrote the terminal run transition, when any. |
| `terminalized_by_peer_id` | `TEXT NOT NULL DEFAULT ''` | Peer id that wrote the terminal run transition, when any. |
| `terminalized_by_actor_kind` | `TEXT NOT NULL DEFAULT ''` | Server-derived actor kind for the terminal transition. |
| `terminalized_by_actor_ref` | `TEXT NOT NULL DEFAULT ''` | Server-derived actor reference for the terminal transition. |

`metadata_json` and `result_json` remain opaque payloads only. They must not store operational state needed for query predicates, indexes, or contract validation.

#### Notification Cursors

Create a global DB table owned by `internal/notifications`:

| Field | Semantics |
|-------|-----------|
| `consumer_id` | Stable delivery consumer identity. |
| `stream_name` | Cursor stream, for example `task_events` or bridge/thread delivery stream. |
| `subject_id` | Task/channel/thread/bridge scope; use empty string for an unscoped stream. |
| `last_sequence` | Latest confirmed delivered sequence. |
| `last_delivery_id` | Latest confirmed delivery id for idempotent replay checks. |
| `last_delivered_at` | Latest confirmed delivery timestamp. |
| `last_error` | Bounded latest delivery error summary. |
| `updated_at` | Store update timestamp. |

Required key shape:

- Table name: `notification_cursors`.
- Primary key: `(consumer_id, stream_name, subject_id)`.
- `subject_id` is `TEXT NOT NULL DEFAULT ''` to avoid nullable composite-key ambiguity in SQLite.
- Index: `notification_cursors_stream_sequence_idx` on `(stream_name, last_sequence DESC) WHERE last_sequence > 0`.
- Rationale: independent consumers must advance independently; one bridge/thread subscriber must not block another subscriber or a web/SSE replay cursor.

Cursor state is stored in a side table rather than `metadata_json` because it needs monotonic compare-and-set behavior, indexes, and migration-backed validation.

#### Bridge Task Subscriptions

Create a bridge-owned subscription table for the first concrete notification consumer:

| Field | Semantics |
|-------|-----------|
| `subscription_id` | Stable subscription identity. |
| `task_id` | Task whose terminal events should be delivered. |
| `bridge_instance_id` | Bridge instance that owns the delivery target. |
| `scope` | Global/workspace task scope. |
| `workspace_id` | Workspace scope when present. |
| `peer_id` | Optional bridge peer target. |
| `thread_id` | Optional bridge thread target. |
| `group_id` | Optional bridge group target. |
| `delivery_mode` | Delivery mode for terminal notifications. |
| `created_by` | Server-derived actor that created the subscription. |
| `created_at` | Creation timestamp. |
| `updated_at` | Update timestamp. |

`bridge_task_subscriptions` defines notification destinations. `notification_cursors` defines confirmed delivery progress for each subscription. These concerns must remain separate: bridge subscription state is owned by `internal/bridges`; cursor state is owned by `internal/notifications`.

#### Config

Add:

```toml
[task.orchestration]
summary_max_bytes = 4096
context_body_max_bytes = 8192
context_prior_attempts = 5
context_recent_events = 50
spawn_failure_limit = 5
scheduler_bad_tick_threshold = 6
scheduler_bad_tick_cooldown = "5m"
default_max_runtime = "0s"

[task.orchestration.profile]
default_coordinator_mode = "inherit"
default_worker_mode = "inherit"
default_sandbox_mode = "inherit"
allow_task_provider_override = true
allow_task_sandbox_none = true
```

`default_max_runtime = "0s"` disables a default watchdog. Tasks may override with `max_runtime_seconds`. The config parser is the only duration-string boundary; task service and store layers receive integer seconds. Validation rejects runtime watchdog durations greater than `24h`.

Profile config semantics:

- `default_coordinator_mode = "inherit"` keeps current workspace coordinator behavior unless a task explicitly requests guided mode.
- `default_worker_mode = "inherit"` keeps workspace default agent/provider/model behavior unless a task explicitly selects workers.
- `default_sandbox_mode = "inherit"` keeps workspace sandbox behavior unless a task requests `none` or `ref`.
- `allow_task_provider_override` gates task-level worker/reviewer provider and model overrides.
- `allow_task_sandbox_none` gates `SandboxPolicy.mode = "none"`.
- Config docs and CLI/API config inspection must show these defaults and validation errors.

### API Endpoints

Use shared `internal/api/core` handlers and keep HTTP/UDS parity.

| Method | Path | Change |
|--------|------|--------|
| `POST` | `/api/tasks` | Accept optional `execution_profile` with task-service validation. |
| `PATCH` | `/api/tasks/:id` | Accept future-effective `execution_profile` updates when no active run is protected by token-fenced execution. |
| `GET` | `/api/tasks` | Include current-run projection, `latest_event_seq`, max-runtime/spawn-failure/profile read fields where appropriate. |
| `GET` | `/api/tasks/:id` | Include `current_run_id`, current run summary, `latest_event_seq`, `execution_profile`, and orchestration fields. |
| `GET` | `/api/tasks/:id/execution-profile` | Return the task execution profile plus effective defaults and redacted validation metadata. |
| `PATCH` | `/api/tasks/:id/execution-profile` | Update task execution profile through task-service validation. Reject active-run mutations that would alter the current worker session. |
| `GET` | `/api/tasks/:id/timeline` | Preserve `after_sequence`; include summaries in timeline/run payloads when present. |
| `GET` | `/api/tasks/:id/stream?after_sequence=N` | Preserve SSE replay and `Last-Event-ID`; web clients seed `N` from `latest_event_seq`. |
| `GET` | `/api/tasks/dashboard` | Extend with read-only scheduler/orchestration health instead of adding a parallel dashboard endpoint. |
| `GET` | `/api/task-runs/:id` | Include `summary` and runtime limit/projection context. |
| `POST` | `/api/tasks/:id/complete` | Operator-only task-level terminal complete. If no active run exists, synthesize a zero-duration terminal run. If a token-fenced active run exists, reject with a protected-active-run conflict. |
| `POST` | `/api/tasks/:id/fail` | Operator-only task-level terminal fail. If no active run exists, synthesize a zero-duration terminal run. If a token-fenced active run exists, reject with a protected-active-run conflict. |
| `POST` | `/api/tasks/:id/notifications/bridges` | Operator/API task-notification subscription creation for bridge-delivered terminal task notifications. |
| `GET` | `/api/tasks/:id/notifications/bridges` | List bridge terminal-notification subscriptions for the task. |
| `DELETE` | `/api/tasks/:id/notifications/bridges/:subscription_id` | Delete one bridge terminal-notification subscription. |
| `POST` | `/api/task-runs/:id/complete` | Accept bounded `summary` with `result`; maintain token-fence behavior for protected runs. |
| `POST` | `/api/task-runs/:id/fail` | Accept bounded `summary` with `error` and `metadata`. |
| `POST` | `/api/agent/tasks/:run_id/complete` | Accept bounded `summary`; resolve active lease from caller session. |
| `POST` | `/api/agent/tasks/:run_id/fail` | Accept bounded `summary`; resolve active lease from caller session. |
| `GET` | `/api/agent/context` | Extend existing situation payload with bounded task context bundle. |

HTTP/UDS parity matrix:

| Operation | HTTP path | UDS/core handler requirement |
|-----------|-----------|------------------------------|
| Create task with profile | `POST /api/tasks` | Shared task create handler validates `execution_profile` and returns generated DTOs. |
| Patch task profile | `PATCH /api/tasks/:id/execution-profile` | Shared task profile handler; rejects active-run mutations that would affect an already started worker. |
| Read task profile | `GET /api/tasks/:id/execution-profile` | Shared task read authorization/redaction. |
| Task-level complete | `POST /api/tasks/:id/complete` | Shared operator-only handler; rejects session actors and active token-fenced runs. |
| Task-level fail | `POST /api/tasks/:id/fail` | Shared operator-only handler; rejects session actors and active token-fenced runs. |
| Create bridge terminal subscription | `POST /api/tasks/:id/notifications/bridges` | Shared task-notification handler; uses server-derived actor identity. |
| List bridge terminal subscriptions | `GET /api/tasks/:id/notifications/bridges` | Shared task read authorization/redaction. |
| Delete bridge terminal subscription | `DELETE /api/tasks/:id/notifications/bridges/:subscription_id` | Shared task notification handler; deletes only matching task/subscription rows. |
| Run-level complete | `POST /api/task-runs/:id/complete` | Shared protected run-level terminal handler. |
| Run-level fail | `POST /api/task-runs/:id/fail` | Shared protected run-level terminal handler. |
| Agent complete | `POST /api/agent/tasks/:run_id/complete` | Shared session-bound lease lookup; no raw claim token. |
| Agent fail | `POST /api/agent/tasks/:run_id/fail` | Shared session-bound lease lookup; no raw claim token. |
| Agent context | `GET /api/agent/context` | Shared situation handler; active-lease task bundle only. |
| Dashboard health | `GET /api/tasks/dashboard` | Shared read-only orchestration health payload. |

`latest_event_seq` contract:

- Type: signed JSON number backed by Go `int64`.
- Source: the maximum durable `task_events.event_seq` for the task, or `0` when the task has no events.
- Required on every event-bearing task list/detail/dashboard payload that can open a task stream.
- Web opens `EventSource` with `?after_sequence=<latest_event_seq>` after rendering the read payload. Contract tests must prove the seed prevents the read-then-stream race.
- On SSE reconnect, `Last-Event-ID` takes precedence over `?after_sequence`; first-open uses `?after_sequence` seeded from `latest_event_seq`.

Task-level terminal mutation against an active run:

- `POST /api/tasks/:id/complete` and `POST /api/tasks/:id/fail` are operator-only HTTP/operator-UDS paths. They are not exposed under `/agent/*` and are not part of the autonomy toolset.
- Agent/session actors must use `/api/agent/tasks/:run_id/complete|fail`, which resolves the caller's active lease through session-bound lookup.
- If a task has any non-terminal `task_runs` row, task-level terminal mutation rejects with a protected-active-run conflict and does not request session stop, write terminal state, insert a synthetic run, or change `current_run_id`.
- Operators who need to stop an active worker must use the existing protected session/task-run stop or fail path that converges on task-service-owned run transitions.
- This rejection rule avoids racing a concurrent worker `CompleteRunLease` and keeps claim-token fenced runs terminalized only through run-level task-service paths.
- Contract tests must prove session actors cannot call these endpoints to close work they do not own.

`/api/agent/context` task bundle binding:

- Agent callers resolve task context through `LookupActiveRunForSession(session_id, optional_run_id)`.
- If the caller has no active lease, the task bundle is present with `available=false` and empty task/run details; it must not fall back to the last terminal run.
- A caller-supplied `run_id` is accepted only to disambiguate the caller's own active lease and must fail with the existing foreign-run autonomy reason when it does not match the session.
- Operator UDS/admin paths may request `OperatorTaskContextRequest{TaskID}` only through shared core handlers that apply normal task read authorization and redaction.
- Redaction tests must cover no-claim, cross-session attempt, terminal-last-run non-fallback, and raw claim-token absence.

Generated artifacts must be co-shipped with contract edits:

- `openapi/agh.json`
- `web/src/generated/agh-openapi.d.ts`

The implementation PR must run `make codegen-check` after regeneration.

### CLI, Tools, and Agent Surfaces

Agent-manageability is required for every runtime capability:

- CLI task-run complete/fail surfaces must accept `--summary` where the matching API accepts summary.
- CLI task-level terminal surfaces must be explicit, for example `agh task complete-task <task-id>` and `agh task fail-task <task-id>`, so run-level `agh task complete <run-id>` semantics remain unambiguous.
- CLI task profile surfaces must be JSON-capable, for example `agh task profile get <task-id> --json` and `agh task profile set <task-id> --worker-agent <agent> --sandbox-mode ref --sandbox-ref <ref> --json`.
- Native autonomy tools must include `summary` in schemas for complete/fail.
- Tool descriptors must explain summary bounds and raw claim-token restrictions.
- `agh me context` must display the enriched task context bundle without leaking raw tokens.
- Existing session-bound task actions remain the agent-safe mutation path.
- Task profile read/update surfaces must expose deterministic validation errors for unknown agent, provider, model, channel, peer, capability, or sandbox references.
- Config read/update surfaces must include `[task.orchestration]` where AGH already exposes runtime config management.
- Bridge terminal-notification subscriptions are operator/API managed through task-scoped notification endpoints; agents may inspect resulting task context but do not own bridge delivery authority.
- Task execution profiles are agent-manageable through HTTP/UDS/CLI, but they do not grant task ownership or review verdict authority.
- `agh-task-worker` loads for worker sessions that have an active task claim or are entering the session-bound task tool loop. It should not load globally for unrelated operator sessions.
- `agh-orchestrator` loads only for daemon-managed coordinator sessions through coordinator runtime injection.

Bundled skill frontmatter:

```yaml
---
name: agh-task-worker
description: Guidance for AGH worker sessions executing task runs through session-bound task APIs and tools.
version: "1.0.0"
metadata:
  agh:
    bundled: true
    instructional_only: true
    always_load:
      session_types: ["worker"]
      requires_active_task_claim: true
    related_skills: ["agh-session-guide", "agh-tools-guide"]
---
```

```yaml
---
name: agh-orchestrator
description: Guidance for daemon-managed AGH coordinator sessions that plan, spawn, hand off, and supervise task execution without owning task state.
version: "1.0.0"
metadata:
  agh:
    bundled: true
    instructional_only: true
    always_load:
      session_types: ["coordinator"]
      injected_by: "internal/daemon/coordinator_runtime"
    related_skills: ["agh-task-worker", "agh-session-guide", "agh-tools-guide"]
---
```

Skill frontmatter is part of the bundled skill contract. The `metadata.agh.always_load` fields define runtime load triggers. The runtime may index these skills for catalog/search, but `agh-orchestrator` bootstrap must use the deterministic loader path rather than catalog discovery.

### Web and Docs Impact

Web changes must stay inside the existing task system:

- `web/src/systems/tasks/adapters/tasks-api.ts`
- `web/src/systems/tasks/hooks/use-task-live.ts`
- `web/src/systems/tasks/hooks/use-task-actions.ts`
- `web/src/systems/tasks/lib/query-options.ts`
- `web/src/systems/tasks/types.ts`
- `web/src/systems/tasks/mocks/fixtures.ts`
- task dashboard, task detail, task run detail, and tests under `web/src/systems/tasks/**`
- task create/edit/read-model surfaces that expose execution profile fields, only where the backend actually supports profile mutation.

Docs impact:

- `packages/site/content/runtime/core/configuration/config-toml.mdx` for `[task.orchestration]`
- `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx` for summaries, synthetic terminal runs, current-run projection, and max-runtime behavior
- `packages/site/content/runtime/core/autonomy/task-execution-profiles.mdx` for coordinator guided mode, worker selection, participant policy, and sandbox policy
- `packages/site/content/runtime/core/autonomy/coordinator.mdx` for deterministic `agh-orchestrator` injection
- `packages/site/content/runtime/core/skills/bundled.mdx` for `agh-task-worker` and `agh-orchestrator`
- `packages/site/content/runtime/api-reference/tasks.mdx` after OpenAPI regeneration
- `packages/site/content/runtime/api-reference/bridges.mdx` for bridge task notification subscription behavior, if generated API docs split bridge/task references
- `packages/site/content/runtime/api-reference/agent.mdx` for `/api/agent/context` task bundle changes
- `packages/site/content/runtime/cli-reference/task/*.mdx` and `packages/site/content/runtime/cli-reference/me/context.mdx` if task command flags or context output change

## Integration Points

No third-party external service integration is added. Internal integration points are:

| Integration | Purpose | Error Handling |
|-------------|---------|----------------|
| ACP agent sessions | Worker/coordinator sessions consume task execution profiles, context, and task tools. | Session-bound lease lookup prevents cross-session mutation; profile validation errors are deterministic. |
| Session start | Apply effective task agent/provider/model/sandbox profile. | Reject unknown or disallowed providers/models/sandbox refs before starting the worker session. |
| Task hooks | Notify coordinator, notifier, observability, and extension surfaces. | Hooks are best-effort extension points, not safety invariants. |
| Bridge terminal task notifier | `internal/bridges` delivers one-shot terminal task notifications using durable task-event replay plus `internal/notifications` cursor progress. | Advance cursor only after confirmed bridge delivery. |
| Web SSE | Reconnect to task streams from a seeded `after_sequence`. | Fall back to timeline query on connection loss. |
| Config lifecycle | Expose `[task.orchestration]` defaults and validation. | Invalid bounds/durations reject at config validation. |

Hook dispatch mapping:

| Transition | Hook / observe event | Dispatch owner |
|------------|----------------------|----------------|
| Task execution profile updated | `task.execution_profile_updated` | `task.Service` after durable profile update. |
| Run summary recorded | `task.run_summary_recorded` | `task.Service` immediately after durable run update. |
| Current-run projection set/cleared | `task.current_run_projection_updated` | `task.Service` inside the same transition path that updates the projection. |
| Spawn failure incremented | `task.spawn_failure_count_incremented` | `task.Service` after durable counter update. |
| Spawn failure circuit opened | `task.spawn_failure_circuit_opened` | `task.Service` after task moves to `blocked`. |
| Max runtime exceeded | `task.max_runtime_exceeded` | `task.Service` after managed-stop sequence records terminal failure. |
| Scheduler bad tick detected | `scheduler.bad_tick_detected` | `internal/scheduler` through read-only health telemetry. |
| Scheduler cooldown started | `scheduler.bad_tick_cooldown_started` | `internal/scheduler` through read-only health telemetry. |
| Notification cursor advanced | `notification.cursor_advanced` | `internal/notifications` after durable monotonic advance. |
| Notification cursor advance failed | `notification.cursor_advance_failed` | `internal/notifications` on typed advance failure. |
| Terminal notification state mismatch | `notification.terminal_state_mismatch` | `internal/bridges` terminal notifier after a replayed accepted-final terminal event disagrees with current task/review state. |
| Coordinator orchestrator skill injected | `coordinator.orchestrator_skill_injected` | `internal/daemon/coordinator_runtime` after prompt assembly. |

Hooks and observe projections are tail consumers of typed transitions. They must never tail raw event tables to invent runtime state and must never replace task-service transition authority. Scheduler `scheduler.*` signals are `internal/observe` typed events only; this TechSpec does not introduce a scheduler hook taxonomy.

Bridge terminal task notifier flow:

1. `internal/bridges` loads active `bridge_task_subscriptions`.
2. For each subscription, it resolves the cursor using `consumer_id = "bridge_task_subscription:<subscription_id>"`, `stream_name = "task_events"`, and `subject_id = <task_id>`.
3. It lists durable `task_events` where `event_seq > cursor.last_sequence`.
4. It filters candidate terminal notification events: `task.run_completed`, `task.run_failed`, `task.run_canceled`, `task.run_review_approved`, and `task.canceled`.
5. Before delivery, it reloads current task state and review rollup state from `task.Service`/store.
6. It resolves a terminal notification decision:
   - `deliver`: the current task is terminal and the replayed event represents the accepted final terminal result. For run-scoped events, the event's `run_id` must match the accepted terminal run. For review-gated work, `task.run_review_approved` is the accepted-final delivery event.
   - `defer`: the replayed event is a run-level terminal event for a review-required or review-rejected run while review or continuation work is still active, or it has been superseded by a later review continuation. The notifier does not deliver, does not emit mismatch, and does not advance the cursor solely for that deferred event. It may continue scanning later events in the same replay batch and advance to a later accepted-final event only after that final delivery succeeds.
   - `mismatch`: the replayed event claims to be the accepted final terminal result, but the current terminal task state, accepted terminal run id, or terminal outcome disagrees.
7. It sends one message to the configured bridge target through `bridges/deliver` directly only for `deliver`.
8. It advances the cursor only after delivery success is confirmed for the accepted-final event.

Notifier wake-up can come from a hook or `task.EventObserver`, but wake-up is only a nudge. Authority remains the durable replay of `task_events.event_seq`, never channel/thread state. This notifier is one-shot terminal delivery, not progressive session/turn streaming, and it must not use the prompt/session `DeliveryBroker` as its primary consumer path.

If the decision is `mismatch`, the notifier must fail closed:

1. It does not deliver the notification.
2. It does not advance the cursor for that subscription.
3. It records a bounded `last_error` on the cursor and emits `notification.terminal_state_mismatch` through observe/structured logs with `task_id`, `event_seq`, replayed terminal type, current task status, accepted terminal run id, and subscription id.
4. Recovery requires the normal cursor `Reset` path or a task-service repair that makes replay and task state agree. The notifier must not invent a terminal status from channel/thread state or from the bridge delivery target.

Bridge notification envelope:

```json
{
  "delivery_id": "notif:<subscription_id>:<event_seq>",
  "event_type": "final",
  "final": true,
  "seq": 123
}
```

`seq` is the numeric `task_events.event_seq` from the replayed terminal event.

## Extensibility, Agent Manageability, and Config Lifecycle

Extensibility:

- Extend typed task hooks and daemon observers only; do not add a generic event bus.
- Bundled orchestration skills are reusable capability content, not forked runtime formats.
- Task execution profiles are typed task-owned configuration. Extensions may observe profile update hooks, but they must not become profile authority or store profile state in extension metadata.
- `internal/notifications` is a durable cursor primitive only. It does not own task authority, hook dispatch, queue semantics, or event fan-out policy. The first concrete MVP consumer is bridge-delivered terminal task notifications owned by `internal/bridges`.
- No frontend plugin SDK or extension UI API is included in MVP.
- No bridge SDK wire change is required unless bridge delivery consumers need cursor status exposure.

Agent manageability:

- Agents manage task execution through existing session-bound task APIs/tools.
- Agents inspect and update task execution profiles through JSON-capable HTTP/UDS/CLI surfaces with deterministic validation errors.
- Agents inspect runtime state through `/agent/context`, `agh me context`, task read tools, task dashboard/status surfaces, and generated contracts.
- Agents cannot manage ownership through channel messages, skills, raw claim tokens, or scheduler state.

Config lifecycle:

- Add defaults, validation, docs, and generated config/API/CLI surfaces for `[task.orchestration]`.
- Add profile defaults and gates under `[task.orchestration.profile]`.
- `summary_max_bytes`, context limits, spawn-failure limits, scheduler bad-tick thresholds, and default runtime limits must be testable.
- No compatibility aliases or fallback config keys.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `internal/task` | Modified | Central authority gains summary, projection, runtime, execution profile, and circuit-breaker transitions. High risk. | Add service/store invariants, profile validation, and transition tests. |
| `internal/store/globaldb` | Modified | Numbered migrations for task/run fields, execution profile tables, selector side tables, and notification cursors. High risk. | Fresh DB, migrated DB, query-plan, reopen/restart tests. |
| `internal/api/contract` | Modified | New fields and request bodies affect OpenAPI/TypeScript. Medium risk. | Regenerate and run codegen checks. |
| `internal/api/core`, `httpapi`, `udsapi` | Modified | Shared handlers must preserve transport parity and redaction. High risk. | Add parity tests for HTTP/UDS. |
| `internal/daemon/native_tools.go` | Modified | Tool schemas and handlers accept summary. Medium risk. | Add schema and invocation tests. |
| `internal/daemon/task_runtime.go` | Modified | Task worker sessions must receive effective agent/provider/model/sandbox profile. High risk. | Test profile resolution and session start failure paths. |
| `internal/session` | Modified | Session create/start options must accept profile-derived agent/provider/model/sandbox inputs. Medium/high risk. | Test workspace fallback and profile override precedence. |
| `internal/situation` | Modified | Context bundle grows bounded task sections. Medium risk. | Add redaction, truncation, provenance tests. |
| `internal/bridges` | Modified | Owns bridge task notification subscriptions and first `internal/notifications` consumer. Medium/high risk around duplicate delivery. | Add subscription CRUD, durable replay, idempotent delivery, and cursor advancement tests. |
| `internal/scheduler` | Modified | Health telemetry and runtime watchdog recovery. High risk. | Prove scheduler never claims work. |
| `internal/coordinator` | Modified | Deterministic orchestrator skill injection and guided task profile handling. Medium risk. | Test prompt assembly, profile guidance, and avoid duplicate guidance. |
| `internal/skills/bundled` | Modified | Adds two bundled skills. Low/medium risk. | Validate frontmatter, load, search, and view behavior. |
| `internal/notifications` | New | Shared cursor primitive. Medium risk. | Unit and integration tests for monotonic advancement. |
| `web/src/systems/tasks` | Modified | Cursor-seeded SSE, task execution profiles, and new fields. Medium risk. | Vitest coverage for reconnect, profile payload mapping, and mutation errors. |
| `packages/site` | Modified | Config/API/CLI docs. Low risk. | Update generated and authored docs. |

## Test Strategy

### Unit Tests

- `internal/task`: summary validation, execution profile validation, transition projection updates, spawn-failure breaker, max-runtime state, synthetic terminal paths.
- `internal/store/globaldb`: migration registry, column/table presence, run provenance columns/indexes, profile selector indexes, claim query plan, cursor CRUD, monotonic cursor advancement.
- `internal/api/core`: request parsing, summary bounds, redaction, status mapping, HTTP/UDS shared behavior, and operator-only rejection for task-level terminal mutation by session actors.
- `internal/situation`: context bundle bounds, prior attempts, recent events, summaries, provenance, raw-token absence.
- `internal/bridges`: bridge task subscription CRUD, terminal event filtering, task/review-state recheck before delivery, accepted-final decision logic, deferred review-gated run terminal events, `bridges/deliver` invocation, duplicate delivery idempotency, cursor advancement after confirmed delivery, and terminal-state mismatch fail-closed behavior.
- `internal/scheduler`: bad-tick telemetry, cooldown behavior, max-runtime recovery trigger, and a boundary test that fails if scheduler code receives `ClaimNextRun`, terminal mutation methods, raw claim tokens, or writes task-run ownership columns.
- `internal/daemon/task_runtime.go` and `internal/session`: effective worker profile resolution, workspace fallback, provider/model override gates, sandbox `inherit|none|ref`, and startup failure reporting.
- `internal/coordinator`: deterministic `agh-orchestrator` injection, guided profile prompt input, and prompt overlay composition.
- `internal/skills/bundled`: `agh-task-worker` and `agh-orchestrator` load/search/view behavior.
- `web/src/systems/tasks`: generated payload mapping, SSE seed handling, reconnect behavior.

### Integration Tests

- Fresh DB boot with all new schema.
- Migrated DB boot from prior schema with numbered migrations.
- Task execution profile create/read/update through HTTP and UDS parity.
- Task profile update rejects active-run mutations that would alter an already started worker.
- Worker session start uses task profile agent/provider/model/sandbox before workspace defaults.
- `ClaimNextRun` rejects an ineligible agent name and accepts the configured worker agent.
- Participant policy does not grant task ownership or channel authorization, and routing/claim/session-start call sites reject violations with deterministic errors.
- Agent task claim -> heartbeat -> complete with summary through UDS and HTTP parity.
- Agent fail with summary/metadata and no raw claim-token exposure.
- Operator task-level complete/fail on a never-claimed task creates exactly one synthetic terminal run under concurrent requests.
- Operator task-level complete/fail rejects when a token-fenced active run exists.
- HTTP/UDS parity matrix covers `/api/tasks`, `/api/tasks/:id/execution-profile`, `/api/tasks/:id/complete|fail`, `/api/tasks/:id/notifications/bridges`, `/api/tasks/:id/notifications/bridges/:subscription_id`, `/api/task-runs/:id/complete|fail`, `/api/agent/tasks/:run_id/complete|fail`, `/api/agent/context`, and `/api/tasks/dashboard`.
- Coordinator bootstrap includes deterministic `agh-orchestrator` guidance.
- Worker context includes bounded task bundle and no raw token.
- `/api/agent/context` returns no task bundle for a session without an active lease and rejects cross-session `run_id` attempts.
- Task dashboard exposes scheduler/orchestration health.
- Web task detail seeds EventSource from `latest_event_seq` and replays missed events.
- Notification cursor consumer advances only after confirmed delivery and can resume from stored sequence.
- Bridge terminal task notifier replays task events by `event_seq`, defers run-level terminal events while review/continuation is active, delivers only accepted-final terminal notifications through `bridges/deliver`, and advances cursor after confirmed success.
- Bridge terminal task notifier fails closed when a replayed accepted-final terminal event and the current task/review-state recheck disagree.
- Config defaults and validation across CLI/API/docs surfaces where applicable.
- Contract/codegen checks prove `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` contain summary fields, `latest_event_seq`, task orchestration fields, task execution profile DTOs, and context bundle changes.
- Max-runtime integration covers request stop -> grace expiry -> force stop -> terminal write -> projection clear, plus a race where the worker sends `CompleteRunLease` after stop request but before terminal write.

Final verification before completion must include `make verify`.

## Implementation Steps

### Build Order

1. **Schema migrations and store models** - no dependencies.
2. **Task service orchestration fields** - depends on step 1.
3. **Task execution profile validation and stores** - depends on steps 1 and 2.
4. **Summary validation and transition invariants** - depends on step 2.
5. **Worker/session/sandbox profile resolution** - depends on step 3.
6. **Claim eligibility profile filters** - depends on steps 1, 3, and 5.
7. **Notification cursor primitive** - depends on step 1.
8. **Bridge task subscription store and terminal notifier consumer** - depends on steps 1 and 7.
9. **Task context bundle in `internal/situation`** - depends on steps 2, 3, and 4.
10. **API contract DTOs and core handler updates** - depends on steps 2, 3, 4, 6, 8, and 9.
11. **OpenAPI and TypeScript codegen** - depends on step 10.
12. **Native tools and CLI parity** - depends on steps 10 and 11.
13. **Scheduler health and max-runtime recovery** - depends on steps 2 and 4; scheduler observes expired wall-clock budgets and sends `MaxRuntimeExceeded` requests to `task.Service`.
14. **Bundled skills and coordinator deterministic injection** - depends on steps 5, 9, and 12.
15. **Web task SSE/read-model/profile updates** - depends on step 11.
16. **Docs/config lifecycle updates** - depends on steps 3, 7, 8, 11, 12, 13, and 14.
17. **End-to-end QA and `make verify`** - depends on all previous steps.

Max-runtime recovery sequence:

1. `internal/scheduler` reads active runs through a narrow read interface and detects `started_at + effective_max_runtime_seconds < now`.
2. Scheduler calls `RuntimeRecoverySink.RequestMaxRuntimeRecovery(ctx, MaxRuntimeExceeded{...})`.
3. `task.Service` verifies the run is still active and owns the terminal transition.
4. `task.Service` requests managed stop through `SessionExecutor.RequestTaskStop(ctx, sessionID, StopReasonTimedOut)`; if the grace period expires, it escalates through `ForceTaskStop`.
5. Session stop implementation uses existing process-group termination helpers (`SIGTERM` -> grace -> `SIGKILL`) where the backing provider exposes a subprocess.
6. `task.Service` writes `FailRunLease`-equivalent terminal state with reason `timed_out`, records the summary/error, clears `current_run_id`, and emits `task.max_runtime_exceeded`.
7. Scheduler never calls `FailRunLease`, `ForceTaskStop`, `ClaimNextRun`, or store-level claim/terminal writers directly.

### Technical Dependencies

- Existing `task.Service` and `task_runs` remain the authority.
- Contract changes require OpenAPI and generated TypeScript updates in the same implementation sequence.
- Config docs and CLI docs must be updated with runtime behavior.
- Web changes depend on generated contract types.
- Task execution profile implementation depends on existing workspace default agent and sandbox resolution; it must harden those paths instead of introducing a parallel resolver.
- No implementation task may introduce a second queue, scheduler claim path, channel-owned status, or prompt-only safety boundary.
- Boundary tests must enforce scheduler import/interface shape, unexported `current_run_id` projection helper ownership, and coordinator prompt assembly ownership before implementation tasks are considered complete.
- `metadata.agh.always_load.requires_active_task_claim` requires loader-side support in the bundled skill runtime; this TechSpec includes that loader change in the bundled skills implementation task.

## Monitoring and Observability

Key telemetry and exposure paths:

| Event | Exposure |
|-------|----------|
| `task.execution_profile_updated` | `internal/observe` typed event, structured logs, task timeline payload with redacted profile diff. |
| `task.run_summary_recorded` | `internal/observe` typed event, structured logs, task timeline payload when task-scoped. |
| `task.current_run_projection_updated` | `internal/observe` typed event and structured logs; not exposed as task authority. |
| `task.spawn_failure_count_incremented` | `internal/observe` typed event, structured logs, dashboard health counters. |
| `task.spawn_failure_circuit_opened` | `internal/observe` typed event, structured logs, task dashboard failed/blocked health. |
| `task.max_runtime_exceeded` | `internal/observe` typed event, structured logs, task timeline payload. |
| `scheduler.bad_tick_detected` | `internal/observe` typed event, structured logs, task dashboard health. |
| `scheduler.bad_tick_cooldown_started` | `internal/observe` typed event, structured logs, task dashboard health. |
| `notification.cursor_advanced` | `internal/observe` typed event and structured logs; not sent to task SSE unless task-scoped delivery uses a task event. |
| `notification.cursor_advance_failed` | `internal/observe` typed event, structured logs, dashboard/health surface where consumer lag is shown. |
| `coordinator.orchestrator_skill_injected` | `internal/observe` typed event and structured logs during coordinator bootstrap. |

Every new event must be added to the observe coverage matrix in the same implementation change that dispatches it.

Structured fields:

- `task_id`
- `run_id`
- `session_id`
- `workspace_id`
- `coordination_channel_id`
- `execution_profile_id`
- `coordinator_mode`
- `worker_agent_name`
- `worker_provider`
- `sandbox_mode`
- `sandbox_ref`
- `claim_token_hash`
- `current_run_id`
- `summary_bytes`
- `spawn_failure_count`
- `max_runtime_seconds`
- `cursor_consumer_id`
- `cursor_stream_name`
- `last_sequence`
- `last_delivery_id`
- `bridge_subscription_id`
- `bridge_instance_id`

Alerting/health:

- Scheduler bad ticks crossing `scheduler_bad_tick_threshold`.
- Spawn-failure count crossing `spawn_failure_limit`.
- Notification cursor delivery lag/backlog.
- Bridge terminal notification delivery failures and duplicate-delivery suppressions.
- Max-runtime exceeded events.
- Repeated context bundle truncation, if it hides required task context.

## Technical Considerations

### Key Decisions

- **Extend existing autonomy**: avoids a second orchestration subsystem and preserves archived autonomy decisions.
- **Typed state over JSON**: queryable orchestration state belongs in explicit columns/side tables, not opaque payloads.
- **Shared notification cursors**: delivery progress is reusable runtime state, not bridge-only state.
- **Minimal config**: explicit `[task.orchestration]` defaults avoid hardcoding without creating a policy engine.
- **`current_run_id` as projection**: improves read paths while keeping `task_runs` authoritative.
- **Instructional skills only**: bundled skills improve behavior, but runtime services enforce authority.
- **Typed task execution profiles**: per-task runtime selection is task-owned typed state, not metadata, prompt memory, channel state, or coordinator authority.
- **Guided coordinator MVP**: task-specific coordinator policy guides the existing workspace coordinator; dedicated per-task coordinators stay out of scope.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `current_run_id` becomes accidental authority | Medium | Repeat invariant in tasks/tests; restrict mutations to task service/store transitions. |
| Scheduler scope expands into claiming work | Medium | Keep scheduler interfaces without claim methods; add tests proving no claim path. |
| Summary fields store unbounded prompt dumps | Medium | Enforce `summary_max_bytes` at API/tool/service boundaries. |
| Notification cursor drops events | Medium | Advance only after confirmed delivery; test monotonic/idempotent behavior. |
| Bridge terminal notifier duplicates final messages | Medium | Deterministic `delivery_id`, cursor `last_delivery_id`, at-least-once semantics, and idempotent bridge delivery tests. |
| Coordinator skill duplicates/conflicts with prompt overlay | Medium | Move stable guidance into skill; keep only runtime facts in overlay. |
| Task profile creates unclaimable work | Medium | Validate selectors at write time and expose deterministic diagnostics when no eligible worker exists. |
| Sandbox `none` bypasses safety expectations | Medium | Gate through `[task.orchestration.profile].allow_task_sandbox_none`, audit profile updates, and preserve tool/approval/provider authorization. |
| Web SSE race persists | Medium | Seed `after_sequence` from task read payload and replay through existing stream endpoint. |
| Config surface grows too broad | Low/medium | Keep MVP keys limited to accepted `[task.orchestration]` defaults. |

### Delete / Replace Targets

This is not a compatibility-preserving change. Do not add old-field aliases or fallback reads.

Replace or remove:

- Hardcoded coordinator operational guidance that is duplicated by deterministic `agh-orchestrator` skill content.
- Any ad hoc task-live polling logic in web that duplicates the cursor-seeded SSE path.
- Any implementation attempt to encode queryable orchestration state only in `metadata_json` or `result_json`.
- Any task-specific agent/provider/model/channel/peer/sandbox routing hidden only in `metadata_json`.
- Any worker session bootstrap that ignores a validated `TaskExecutionProfile.Worker` and only uses workspace defaults.
- Any coordinator runtime behavior that interprets `CoordinatorProfile.mode = "guided"` as a dedicated per-task coordinator session.
- Any future bridge-specific cursor side-state for terminal task notification delivery; `bridge_task_subscriptions` stores targets, while `notification_cursors` stores delivery progress.
- Any task-level terminal endpoint behavior that tries to force-close active token-fenced runs; active runs use run-level protected transitions only.
- Any exported/public current-run projection maintenance API; projection helpers stay unexported inside `internal/task`.
- Any terminal task notification path that uses the prompt/session `DeliveryBroker` as the consumer; bridge terminal notifications use the `internal/bridges` notifier and `bridges/deliver` directly.

No active `.compozy/tasks/autonomous/` paths are in scope; prior autonomy material is archived under `.compozy/tasks/_archived/`.

## Architecture Decision Records

- [ADR-001: Orchestration Hardening Extends the Existing Autonomy Substrate](adrs/adr-001-orchestration-hardening-extends-existing-autonomy.md) — Keep `orch-improvs` as hardening over `task.Service`, `task_runs`, scheduler, coordinator, and typed hooks.
- [ADR-002: Use Queryable Task-Owned State for Orchestration Hardening](adrs/adr-002-queryable-orchestration-state.md) — Store operational orchestration state in typed columns and side tables, not opaque JSON.
- [ADR-003: Introduce Shared Durable Notification Cursors](adrs/adr-003-shared-durable-notification-cursors.md) — Add `internal/notifications` for confirmed delivery cursor state.
- [ADR-004: Add Minimal Explicit Task Orchestration Config](adrs/adr-004-minimal-task-orchestration-config.md) — Add bounded `[task.orchestration]` defaults and validation.
- [ADR-005: Keep `tasks.current_run_id` as a Denormalized Read Projection](adrs/adr-005-current-run-id-denormalized-projection.md) — Keep `current_run_id` as read-model state only, never authority.
- [ADR-006: Bundled Orchestration Skills Are Instructional, Not Authority](adrs/adr-006-bundled-orchestration-skills-are-instructional.md) — Add `agh-task-worker` and deterministically injected `agh-orchestrator` as guidance only.
- [ADR-010: Task Execution Profiles Are Typed Task-Owned Overlays](adrs/adr-010-task-execution-profiles-are-typed-overlays.md) — Add typed task-owned coordinator, worker, participant, review, and sandbox overlays without adding profile authority.
