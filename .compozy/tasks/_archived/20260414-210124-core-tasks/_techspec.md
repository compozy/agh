# TechSpec: Core Tasks and Subtasks

## Executive Summary

AGH is session-centric today. `internal/session/` owns live ACP-backed execution, `internal/store/sessiondb/` owns per-session append-only history, `internal/observe/` owns projections, and `internal/daemon/` is the sole composition root. That shape is still correct and should remain correct.

This TechSpec adds a first-class task domain without turning `Task` into AGH's universal execution envelope. `Task` is an explicit coordination resource used for backlog, delegation, ownership, tracking, orchestration, and resume-worthy work. `session` remains the general execution primitive. `automation` remains the general trigger and orchestration primitive. Not every agent action, session, or automation run becomes a task.

The core model remains:

- `Task` as the durable coordination record
- `TaskRun` as the durable execution record
- `TaskManager` as the only authority for canonical task lifecycle

Tasks support `global` and `workspace` scope, explicit `parent_task_id` hierarchy, bounded dependency edges, optional network-channel binding, optional ownership, and queue-first execution. Executable subtasks are the default unit of work and, by default, start in dedicated sessions. Existing sessions may be attached only in explicit `resume`, `continue`, or `handoff` flows.

The design keeps broad multi-surface writes in scope for v1 while adding the missing constraints surfaced by review: server-derived identity, clear mutation authority, cooperative cancellation with forced escalation, bounded payload sizes, bounded dependency graphs, and a strict package boundary between `internal/task` and `internal/session`.

## System Architecture

### Domain Positioning

- `internal/task/` owns durable coordination and execution records, lifecycle validation, reconciliation, and task-facing APIs.
- `internal/session/` remains runtime-only and owns live ACP-backed execution.
- `internal/automation/` remains a general trigger and orchestration subsystem. It may create tasks directly or indirectly, but it is not replaced by the task domain.
- `internal/daemon/` remains the only composition root and wires task services to session execution through injected interfaces.

### Component Overview

- `internal/task/`
Purpose: domain types, manager interfaces, validation, reconciliation, authorization checks, lifecycle actions, and task/run services.
Boundary: owns `Task`, `TaskRun`, dependency invariants, mutation rules, and canonical task status transitions.

- `internal/store/globaldb`
Purpose: durable persistence for tasks, dependency edges, runs, task audit records, and idempotency metadata.
Boundary: authoritative task-domain store.

- `internal/session/`
Purpose: create, attach, stop, and observe live sessions.
Boundary: runtime execution only. It does not own task coordination state and is consumed through an injected execution bridge rather than a direct import from `internal/task`.

- `internal/automation/`
Purpose: schedules and triggers operational work.
Boundary: may run jobs without tasks, may create tasks directly, and may also trigger agent flows that explicitly call `task.create`.

- `internal/extension/`
Purpose: extension-host task writes and task-adjacent integrations.
Boundary: may call the task service through guarded capabilities, but does not own task lifecycle.

- `internal/network/`
Purpose: network-originated task/run ingress, peer-context validation, and channel-aware writes.
Boundary: may write through the task service, but canonical task state remains manager-owned.

- `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`
Purpose: expose the task service consistently via REST and UDS.
Boundary: transport only; no duplicated lifecycle logic.

- `internal/cli`
Purpose: `agh task ...` commands backed by daemon APIs.
Boundary: no direct DB or filesystem mutation.

- `internal/observe`
Purpose: read-side projections and health views for tasks and runs.
Boundary: read-only consumers of task state and task audit data.

### High-Level Flow

1. A human, agent, automation, extension, or network peer explicitly creates a task.
2. Every write is rejected unless the surface resolves to an authenticated principal with task permissions for the target scope.
3. `TaskManager` derives `created_by`, `origin`, and mutation authority from that principal context, not from payload fields.
4. If the task represents executable work, a run is enqueued.
5. When an executable subtask starts, `TaskManager` asks an injected session-execution bridge to create a dedicated session by default.
6. Session state flows back into `TaskRun`, and `TaskManager` reconciles canonical task and run status from runs, hierarchy, dependencies, and explicit lifecycle actions.
7. If a task tree is cancelled, queued work is cancelled immediately and live sessions for the target task and its descendants receive cooperative stop before forced escalation after the daemon-configured timeout.

## Implementation Design

### Core Interfaces

```go
type Manager interface {
    CreateTask(ctx context.Context, spec CreateTask, actor ActorContext) (*Task, error)
    UpdateTask(ctx context.Context, id string, patch TaskPatch, actor ActorContext) (*Task, error)
    CancelTask(ctx context.Context, id string, req CancelTask, actor ActorContext) (*Task, error)

    AddDependency(ctx context.Context, spec AddDependency, actor ActorContext) error
    RemoveDependency(ctx context.Context, taskID, dependsOnID string, actor ActorContext) error

    EnqueueRun(ctx context.Context, spec EnqueueRun, actor ActorContext) (*TaskRun, error)
    ClaimRun(ctx context.Context, runID string, claim ClaimRun, actor ActorContext) (*TaskRun, error)
    StartRun(ctx context.Context, runID string, req StartRun, actor ActorContext) (*TaskRun, error)
    AttachRunSession(ctx context.Context, runID, sessionID string, actor ActorContext) (*TaskRun, error)
    CompleteRun(ctx context.Context, runID string, result RunResult, actor ActorContext) (*TaskRun, error)
    FailRun(ctx context.Context, runID string, failure RunFailure, actor ActorContext) (*TaskRun, error)
    CancelRun(ctx context.Context, runID string, req CancelRun, actor ActorContext) (*TaskRun, error)

    GetTask(ctx context.Context, id string) (*TaskView, error)
    ListTasks(ctx context.Context, query TaskQuery) ([]TaskSummary, error)
}
```

```go
type SessionExecutor interface {
    StartTaskSession(ctx context.Context, spec StartTaskSession) (*SessionRef, error)
    AttachTaskSession(ctx context.Context, runID, sessionID string) (*SessionRef, error)
    RequestTaskStop(ctx context.Context, sessionID string, reason StopReason) error
    ForceTaskStop(ctx context.Context, sessionID string, reason StopReason) error
}
```

`SessionExecutor` is defined in `internal/task`, implemented by a daemon-wired adapter over `internal/session`, and prevents direct package coupling.

### Actor and Identity Model

Identity fields are server-derived and must never be trusted from payloads.

- `created_by`
Purpose: immutable actor identity for task creation.
Examples: human via CLI, human via web, agent via session, automation subsystem, extension runtime, network peer identity.

- `origin`
Purpose: immutable technical ingress context.
Examples: `web`, `cli`, `uds`, `http`, `automation:<rule>`, `extension:<id>`, `network:<peer/channel>`, `agent-session:<id>`.

- `owner`
Purpose: mutable operational responsibility.
Rules:
- optional at creation
- may be auto-assigned
- may remain unowned/pool-backed
- may later be claimed or reassigned by an authorized writer

### Authorization Contract

The task domain accepts writes only from authenticated first-class surfaces. Identity derivation is not sufficient by itself; the service must also resolve explicit authority.

Writer classes in v1:

- local human surfaces: CLI, UDS, HTTP, and web routes backed by the local daemon
- agent-session surfaces: explicit tool or capability calls from authenticated AGH sessions
- automation surfaces: daemon-owned internal automation rules and jobs
- extension surfaces: capability-checked extensions
- network surfaces: authenticated network peers mapped by the network layer

Authorization rules:

- unauthenticated callers may not read or write the task domain
- every write resolves to a server-derived principal before task mutation begins
- in v1, every authenticated first-class writer surface is granted both `global` and `workspace` task creation authority by policy
- `workspace` writes still require the referenced workspace to exist and be resolvable by that principal context
- extensions and network peers must additionally satisfy explicit capability checks in their ingress layers before task writes are allowed
- read access follows the same principal model; local daemon surfaces may read all task scopes in v1, while extension and network reads remain capability-gated

This is intentionally broad for v1, but it is now an explicit contract rather than an implicit assumption.

### Data Models

- `Task`
Fields:
`id`, `identifier`, `scope`, `workspace_id`, `parent_task_id`, `network_channel`, `title`, `description`, `status`, `owner_kind`, `owner_ref`, `created_by_kind`, `created_by_ref`, `origin_kind`, `origin_ref`, `created_at`, `updated_at`, `closed_at`, `metadata_json`

Rules:
- `scope = global` requires `workspace_id = NULL`
- `scope = workspace` requires non-null `workspace_id`
- `parent_task_id` is immutable after creation
- `created_by_*` is immutable
- `origin_*` is immutable
- `owner_*` is nullable and mutable
- `network_channel` is nullable and editable

- `TaskDependency`
Fields:
`task_id`, `depends_on_task_id`, `kind`, `created_at`

Rules:
- separate edge table
- no self-dependency
- cycle detection on edge creation
- dependency edges are bounded by explicit guardrails

- `TaskRun`
Fields:
`id`, `task_id`, `status`, `attempt`, `claimed_by_kind`, `claimed_by_ref`, `session_id`, `origin_kind`, `origin_ref`, `idempotency_key`, `network_channel`, `queued_at`, `claimed_at`, `started_at`, `ended_at`, `error`, `result_json`

Rules:
- `session_id` is nullable until execution starts
- once `session_id` is set, it is immutable
- run origin is snapshotted from the actual ingress/start context
- `network_channel` stores the resolved execution channel for audit and filtering

- `TaskEvent`
Fields:
`id`, `task_id`, `run_id`, `event_type`, `actor_kind`, `actor_ref`, `payload_json`, `timestamp`

Rules:
- immutable audit record
- subject to payload size caps
- records lifecycle actions, actor authority, and forced-stop details
- audit-only; not an event-sourcing backbone

### Lifecycle Model

- `TaskStatus`
Initial v1 enum: `pending`, `blocked`, `ready`, `in_progress`, `completed`, `failed`, `cancelled`

- `TaskRunStatus`
Initial v1 enum: `queued`, `claimed`, `starting`, `running`, `completed`, `failed`, `cancelled`

Canonical task status is owned only by `TaskManager`.
Canonical run status is also owned only by `TaskManager`.

Task reconciliation rules:
- `blocked` when unresolved dependencies exist
- `ready` when dependencies are clear and there is no active run
- `in_progress` when any attached run is `starting` or `running`
- terminal task states come from manager-owned lifecycle actions and run outcomes, not arbitrary caller patches

Run lifecycle:
- queue-first
- may exist before any session exists
- executable subtasks default to dedicated new sessions
- attach to an existing session is explicit, not default
- non-human run lifecycle actions require idempotency keys

### Run Authority and Attachment Rules

External writers may request run lifecycle actions, but they do not patch `TaskRun.status` directly. Every run transition is validated and persisted by `TaskManager`.

Attachment rules:

- `attach-session` is allowed only for explicit `resume`, `continue`, or `handoff` paths
- attachment is valid only while the run is in `claimed` or `starting`
- a run may not attach if `session_id` is already set
- a session may be bound to at most one non-terminal run at a time
- once a run reaches `running` or any terminal state, attachment and rebinding are forbidden
- attach failure is terminal to the request, not silently downgraded to new-session allocation

### Mutability Rules

Immutable after creation:
- `scope`
- `workspace_id`
- `parent_task_id`
- `created_by_kind`
- `created_by_ref`
- `origin_kind`
- `origin_ref`

Editable:
- `title`
- `description`
- `metadata_json`
- `network_channel`
- `owner_kind`
- `owner_ref`

Not directly patchable:
- canonical `Task.status`
- canonical `TaskRun.status`
- run terminal outcomes
- lifecycle fields derived from manager reconciliation

### Cancellation Model

Cancellation is explicit and manager-owned.

Rules:
- any authorized writer may request task cancellation
- cancelling a parent propagates to all non-terminal descendants
- queued descendants cancel immediately
- active sessions for the target task and all non-terminal descendants receive cooperative stop first
- the daemon owns one global graceful-stop timeout for task-driven session cancellation
- after timeout expiry, the manager requests forced stop
- forced stop still results in terminal `cancelled`, with forced termination captured in audit metadata rather than as a different main status

### Cold-Start Recovery

Daemon restart must reconcile orphaned non-terminal runs before accepting new task traffic.

Startup rules:

- on boot, `TaskManager` scans `TaskRun.status IN ('claimed', 'starting', 'running')`
- `claimed` runs with no attached session are re-queued
- `starting` or `running` runs whose attached session is not live are marked `failed` with an orphaned-on-boot failure reason
- task status is reconciled after the sweep completes

This recovery happens during daemon boot and is not deferred to later observer projections.

### Guardrails and Limits

Payload limits:
- `metadata_json <= 16 KB`
- `TaskEvent.payload_json <= 64 KB`
- `result_json <= 64 KB`
- serialized task-domain request bodies that would persist more than 64 KB of JSON payload are rejected before persistence

Larger payloads must be stored as referenced artifacts rather than embedded in task/run rows.

Graph limits:
- maximum hierarchy depth: `8`
- maximum dependency edges per task: `32`
- maximum direct children per task: `64`
- cycles rejected at edge creation time

Dependency edge creation uses a single `BEGIN IMMEDIATE` transaction that re-checks graph validity before insert so cycle detection and edge persistence happen under one write lock.

These limits are part of the v1 contract, not merely implementation details.

## API Surface

Task resource:
- `POST /api/tasks` — create a task
- `GET /api/tasks` — list tasks by scope, workspace, status, owner, parent, and channel
- `GET /api/tasks/:id` — fetch one task with children, dependencies, latest runs, and audit summary
- `PATCH /api/tasks/:id` — edit allowed mutable fields only
- `POST /api/tasks/:id/cancel` — request task cancellation
- `POST /api/tasks/:id/children` — create a child task
- `POST /api/tasks/:id/dependencies` — add a dependency edge
- `DELETE /api/tasks/:id/dependencies/:depends_on_id` — remove a dependency edge

Run resource:
- `POST /api/tasks/:id/runs` — enqueue a run
- `GET /api/tasks/:id/runs` — list runs for a task
- `POST /api/task-runs/:id/claim` — claim a queued run
- `POST /api/task-runs/:id/start` — start a run and create a dedicated session by default
- `POST /api/task-runs/:id/attach-session` — explicitly attach an existing session for resume or handoff
- `POST /api/task-runs/:id/complete` — complete a run
- `POST /api/task-runs/:id/fail` — fail a run
- `POST /api/task-runs/:id/cancel` — cancel a run

Writer ingress:
- HTTP and UDS expose the same task service
- CLI uses daemon APIs only
- automation, extension, and network integrations call the same task manager surface rather than private side channels

## Integration Points

- `internal/session`
Used only behind the injected `SessionExecutor` interface. Default behavior for executable subtasks is dedicated-session start; explicit attach is reserved for resume/handoff flows.

- `internal/automation`
Supports both:
- direct task creation through the task service
- indirect task creation via agent sessions that explicitly call `task.create`

Automation jobs that do not need durable coordination remain outside the task domain.
When automation chooses a task-backed path, `TaskRun` becomes the only execution pipeline for that work item. The automation subsystem must not also create a parallel automation run/session for the same task-backed unit of work.

- `internal/extension`
Extensions may create tasks, update mutable fields, enqueue runs, and participate in lifecycle callbacks through capability-checked APIs. Identity remains server-derived from the extension context.

- `internal/network`
Network peers may create tasks and enqueue runs through validated ingress. If `network_channel` is set, network-originated writes must satisfy channel validation rules.
If a stored `network_channel` no longer validates against current network configuration, the task remains readable but new run start/session propagation is rejected until the channel is cleared or corrected; the rejection is logged and audited as a stale-channel condition.

- `internal/observe`
Projects queue depth, task status summaries, stuck runs, ownership views, and forced-cancellation audit details.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/task` | new | New task coordination domain, manager, invariants, and execution bridge interfaces; medium risk | Create package, types, lifecycle logic, validation, and reconciliation |
| `internal/store/globaldb` | modified | New task, dependency, run, audit, and idempotency tables; medium risk | Add schema, limits, CRUD, query indexes, and guardrail enforcement |
| `internal/session` | modified | Session creation/attach/stop exposed through daemon-wired adapter; medium risk | Add bridge implementation without leaking `session` imports into `internal/task` |
| `internal/automation` | modified | Supports both direct and agent-mediated task creation; medium risk | Add explicit task service integration points |
| `internal/extension` | modified | Capability-checked task writes; medium risk | Add guarded task APIs and identity derivation |
| `internal/network` | modified | Channel-aware task/run ingress and origin derivation; medium risk | Add validated writer integration |
| `internal/api/core` | modified | Shared task contracts and action handlers; low risk | Add service interface and action mapping |
| `internal/api/httpapi` | modified | REST routes for tasks and runs; low risk | Add endpoints and tests |
| `internal/api/udsapi` | modified | UDS parity for CLI and local agents; low risk | Add endpoints and tests |
| `internal/cli` | modified | New `agh task` command group; low risk | Add daemon-backed commands |
| `internal/observe` | modified | Task/run projections and health metrics; low risk | Add queries and summaries |
| `internal/daemon` | modified | Wires task service, execution bridge, transports, and observers; low risk | Compose the subsystem cleanly |

## Testing Approach

### Unit Tests

- Validate scope and parent/child invariants
- Validate identity derivation rules and immutable field enforcement
- Validate mutable-field patch rules
- Validate payload limit enforcement
- Validate dependency limits and cycle rejection
- Validate dedicated-session default for executable subtasks
- Validate cancellation propagation, cooperative stop, and forced escalation behavior
- Validate that only `TaskManager` mutates canonical task status

### Integration Tests

- Global DB integration tests with real SQLite for tasks, dependencies, runs, and audit records
- Session bridge tests: queued run -> claimed -> starting -> dedicated session created -> run terminal state reconciled
- Explicit attach-session tests for resume/handoff flows
- Task-tree cancellation tests including cooperative stop and forced escalation after the daemon timeout
- Cold-start recovery tests for orphaned `claimed`, `starting`, and `running` runs
- Automation direct-write and agent-mediated task-creation flows
- Automation non-overlap tests proving task-backed work does not also allocate parallel automation runs
- Extension and network ingress tests with server-derived identity and channel validation
- Attach-session single-assignment tests proving one session cannot bind to multiple live runs
- HTTP and UDS parity tests
- Observe projection tests for ownership, queue depth, and forced-stop audit details

## Development Sequencing

### Build Order

1. Create `internal/task` types, invariants, actor model, payload limits, and global-db schema.
2. Implement task, dependency, run, and audit persistence in `globaldb`.
3. Implement `TaskManager` lifecycle, mutability rules, and status reconciliation.
4. Add the injected session-execution bridge and dedicated-session default for executable subtasks.
5. Implement cold-start orphaned-run recovery and attach-session state gating.
6. Implement task cancellation propagation and daemon-global graceful-stop escalation.
7. Add HTTP, UDS, and CLI surfaces.
8. Add automation, extension, and network writer integrations with explicit non-overlap rules.
9. Add observe projections and end-to-end integration coverage.

### Technical Dependencies

- Global DB schema must land before daemon startup wires the task service.
- Session bridge implementation must exist before run start/cancel flows can be completed.
- Authorization and identity derivation must be resolved in transports before enabling wide writer ingress.
- Daemon-level graceful-stop configuration must be available before cancellation propagation is fully integrated.
- Boot-time orphaned-run sweep must run before the task service starts accepting new run lifecycle actions.

## Monitoring and Observability

- Metrics
`tasks_total{scope,status,channel}`, `task_runs_total{status,origin,channel}`, `task_queue_depth{channel}`, `task_cancel_requests_total{origin}`, `task_forced_stops_total`, `task_claim_latency_ms`, `task_start_latency_ms`, `task_reconcile_duration_ms`

- Logs
Structured fields:
`task_id`, `task_identifier`, `run_id`, `scope`, `workspace_id`, `network_channel`, `created_by_kind`, `created_by_ref`, `origin_kind`, `origin_ref`, `owner_kind`, `owner_ref`, `session_id`, `status_from`, `status_to`, `forced_stop`

- Alerts
Queue age above threshold, repeated reconciliation failures, repeated forced-stop escalations, runs stuck in `claimed` or `starting`, orphaned active runs without live sessions, excessive channel-mismatch rejections

## Technical Considerations

### Key Decisions

- `Task` is an explicit coordination resource, not AGH's universal execution envelope.
- `Task` and `TaskRun` stay split.
- `global` and `workspace` scope remain supported in v1.
- Hierarchy and dependency edges both remain first-class, but with explicit graph guardrails.
- Any authorized writer may create tasks in v1, but identity is always server-derived.
- Executable subtasks use dedicated sessions by default.
- `TaskManager` remains the only authority for canonical task lifecycle.
- `internal/task` talks to execution through an injected bridge, not through direct imports of `internal/session`.
- Optional channel binding remains supported, but as a secondary organizational concern rather than the center of the domain.

### Known Risks

- Broad writer ingress increases replay and idempotency complexity.
Mitigation: require origin metadata and idempotency support for non-human callers.

- Broad writer ingress can still devolve into weak security if capability checks stay implicit.
Mitigation: keep the authorization contract explicit per writer surface and reject writes before persistence when no principal or capability is resolved.

- Large task payloads could still pressure SQLite or transport surfaces if limits are ignored.
Mitigation: enforce hard caps and require external artifact references for larger data.

- Automation can still drift back toward duplicate execution if task-backed and non-task jobs are not kept separate.
Mitigation: require task-backed automation to use the task service as the sole execution authority for that work item.

- Forced cancellation could hide repeated graceful-stop failures.
Mitigation: emit audit records, metrics, and alerts specifically for forced-stop paths.

- Cross-workspace hierarchies and dependencies can still become hard to reason about at scale.
Mitigation: keep graph limits explicit and reject invalid edges up front.

## Architecture Decision Records

- [ADR-001: Separate Task Coordination Records from TaskRun Execution Records](adrs/adr-001.md) — keeps durable coordination separate from execution history.
- [ADR-002: Support Global and Workspace Task Scope with Explicit Hierarchy and Bounded Dependencies](adrs/adr-002.md) — keeps scope, hierarchy, and dependencies while adding explicit guardrails.
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](adrs/adr-003.md) — preserves queue-first execution and central lifecycle authority without turning tasks into the universal executor.
- [ADR-004: Support Optional Task-to-Network-Channel Binding](adrs/adr-004.md) — keeps channel binding as an optional organizational concern.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](adrs/adr-005.md) — formalizes `created_by`, `origin`, and `owner`.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](adrs/adr-006.md) — formalizes the task/session boundary and execution default.
