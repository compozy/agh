# Autonomous AGH TechSpec

## Executive Summary

This TechSpec turns the autonomous-system analysis into an implementation plan for AGH as a local-first agent operating system. There is no `_prd.md` for this effort; the primary input is `.compozy/tasks/autonomous/analysis/analysis.md`, the ten slice analyses under `.compozy/tasks/autonomous/analysis/`, and the technical decisions recorded in `adrs/`.

The implementation strategy is to connect existing AGH substrate instead of replacing it: session lifecycle, task runs, network peers/channels, memory, skills/tools, hooks, resources, UDS/HTTP APIs, and daemon composition remain the core. New autonomy behavior is added through four coordinated layers: Situation Surface, Agent Kernel CLI, Autonomy Kernel, and Memory/Self-Correction. The primary trade-off is that the TechSpec is intentionally broad, but the build order keeps the first release practical: start with agent reachability and durable ownership, then add coordinator-led semantic orchestration, then deepen memory/eval loops.

Autonomy extensibility is a first-class requirement. New autonomous behavior must expose hook, resource, provider, or narrow interface extension points when it creates externally meaningful runtime behavior. The design reuses AGH's existing typed hook taxonomy and resource reconciliation system; it does not introduce a generic event bus, a separate autonomy plugin stack, or ad-hoc callbacks.

Autonomy is additive, not a replacement for operator control. Users must still be able to create tasks manually, start sessions manually, prompt sessions directly, and use existing operator surfaces for counter-checks or directed work. Task creation is not blocked by autonomy. Once a user gives a task the go-ahead to run, by publishing, starting, or approving execution so a run is enqueued, the coordinator becomes the default orchestrator for that run. User-created tasks, coordinator-created tasks, and agent-created child tasks use the same task/session/claim/lease contracts; there is no separate manual queue and no coordinator-only queue.

Coordinated execution also has an explicit communication plane. Every workspace-scoped task run enqueued for coordinated execution is bound to a stable coordination channel. The coordinator and workers use that channel for status, requests, blockers, handoffs, review requests, and result exchange. Task ownership and terminal state still live only in `task_runs` through `ClaimNextRun` and token-fenced transitions.

## System Architecture

### Component Overview

**Situation Surface**

Renders the runtime facts an agent needs to act autonomously: self identity, workspace, session, provider/model, current channels, peer roster, task/workflow envelope, capabilities, memory provenance, and relevant system limits. The first implementation lives in `internal/daemon` as prompt providers and augmenters wired through existing `session.PromptProvider` and `session.PromptInputAugmenter` seams.

**Agent Kernel CLI**

Adds agent-shaped verbs over the existing CLI/UDS contract. Operator commands stay explicit; agent commands infer caller identity from `AGH_SESSION_ID`, `AGH_AGENT`, and daemon-issued session context. The first namespaces are `agh me`, `agh ch`, `agh task`, and `agh spawn`, with stable JSON/JSONL output and deterministic exit codes.

**Task Claim and Lease**

Extends the current `task_runs` model with claim fencing and lease state. The task service/globaldb remains the durable source of truth. There is no parallel durable queue. `ClaimNextRun(criteria)` atomically selects and claims one queued run inside SQLite, returns a `claim_token`, and requires that token for heartbeat, completion, failure, and release operations. In this TechSpec, "lease" means task-run ownership lease; it is intentionally separate from future sandbox, workspace runtime, or environment leases.

**Mechanical Scheduler**

Adds a daemon-owned scheduler under `internal/scheduler`. For the MVP it is a sweep/notify component, not the authoritative run claimant. It owns idle-session indexing, capability-aware wakeups, lease sweep, boot recovery, and backpressure signals. Claims are initiated by agents through `agh task next` / `/agent/tasks/claim-next`, by explicit operator assignment through the task service, or by coordinator-created work being pulled by eligible sessions. All paths use `ClaimNextRun` or token-fenced run transitions.

**Coordinator Agent**

Adds a normal managed AGH session that owns semantic orchestration: decomposition, follow-up task creation, delegation intent, validation, and synthesis. It is spawned when a task run is enqueued for coordinated execution, uses a restricted orchestration tool surface, and is configurable by provider/model through global config and workspace override. Manual user-created tasks do not need special metadata at creation time; the publish/start/approval action that enqueues the run is the coordinator trigger. The coordinator communicates with workers through the run's coordination channel but never uses channel messages as task ownership or terminal-status state.

**Safe Spawn and Lineage**

Extends session creation with parent/child metadata, TTL, budget, role overlays, and permission narrowing. Spawned sessions are regular managed sessions with lineage fields and strict caps. Children can never widen parent permissions, tools, or skills.

**Minimal Network Evolution**

Keeps the local autonomy MVP focused: task-run coordination channel binding, channel discovery metadata, peer status, handoff when needed, and enough peer/channel inspection for agents and the coordinator to route work. Cross-daemon swarm, leader election, and broad contract-net behavior remain out of scope for the MVP.

**Memory and Self-Correction**

Makes coordination outcomes durable and useful: agent/session provenance, session summaries, recall provenance, workflow correlation, loop detection, budget/iteration circuit breakers, and a basic eval/replay harness. Memory improvements start with provenance and session summaries before adding broad peer/channel scope extraction.

The MVP includes `agent_name` and `session_id` provenance plumbing for hooks, logs, task-run payloads, and situation rendering. Broader peer/channel memory scopes, automatic per-turn extraction, session-end summaries, and `IterationCurrent` increments remain post-MVP.

**Autonomy Hook and Resource Surface**

Adds typed coordinator, spawn, and task-run events, payloads, patches, dispatch methods, and introspection descriptors through `internal/hooks`. Hook declarations continue to come from config, agents, skills, extensions, and hook binding resources. The MVP adds zero new resource kinds unless a durable user-authored declaration becomes necessary; coordinator config starts in config, transient scheduler state stays in memory or `task_runs`, and workflow/eval resources are post-MVP.

### Data Flow

1. A user, automation, agent, or coordinator creates or updates a task as durable intent through existing task APIs. Creation may produce a draft, blocked, or ready task, but it does not create run ownership and does not start a coordinator.
2. A publish, approval, start, or equivalent execution action explicitly allows the task to run and enqueues a task run through the task service.
3. Run enqueue creates or resolves the run's `coordination_channel_id` for workspace-scoped coordinated work, writes the existing `task.run_enqueued` domain audit event, dispatches the `task.run.enqueued` hook post-commit, and wakes the mechanical scheduler plus coordinator bootstrap path when the run is coordinated.
4. The scheduler refreshes the idle-session registry and notifies eligible sessions that matching claimable work exists.
5. A user-started or autonomous agent calls `ClaimNextRun(criteria)` through `agh task next` or the equivalent UDS endpoint.
6. The task service/globaldb atomically claims one run and returns the claimed task envelope, coordination channel metadata, and `claim_token`.
7. The agent performs work, heartbeats the task-run lease, uses the coordination channel for operational communication when useful, and completes/fails/releases the run with the claim token.
8. Hooks record lifecycle events, memory writes summarize durable outcomes, and observability checks detect loops, timeouts, and budget overruns.
9. The coordinator-agent reads durable task-run outcomes plus relevant coordination channel messages, then creates follow-up tasks or synthesizes final results when semantic orchestration is required.

### Manual Control Contract

Manual operation remains a first-class path:

- Users can create tasks through existing or future operator CLI/HTTP surfaces. Creation may leave a task in draft/blocked/ready state and must not require orchestration metadata.
- Task creation must not enqueue claimable work by itself. Draft tasks require publish; approval-gated tasks require approval; ready tasks still require an explicit run enqueue/start action before any session can claim them.
- Users can start sessions directly. User-started sessions join the idle registry and can claim work when their capabilities match, but the user may also prompt them manually without involving the coordinator.
- Users publish, start, or approve task execution through an explicit action such as `agh task start`, UI start, automation approval, or equivalent API. That action enqueues a run for coordinated execution and triggers the coordinator if no healthy coordinator exists for the workspace.
- Users can explicitly start or stop a coordinator session for inspection, but the daemon still enforces coordinator uniqueness, spawn caps, leases, and permissions.
- Manual assignment and autonomous claim must converge on the same fencing rules. A manually assigned run still carries a claim token and lease if an agent session owns it.
- Operator-facing UI surfaces, including the web Tasks UI, must visually distinguish task creation, publish/approval, run enqueue, and coordinator spawn. The operator should never confuse drafting a task with executing it.

### Scheduler and Claim Authority

`ClaimNextRun` is the only authoritative "next work" primitive. The scheduler does not claim runs directly in the MVP. It wakes eligible sessions, sweeps expired leases, rebuilds idle state, and emits observability events. The actual claim is initiated by the session that will own the work or by an explicit operator assignment endpoint that writes the same token-fenced ownership state.

This avoids two competing placement authorities. The coordinator expresses semantic intent by creating tasks and criteria; the scheduler exposes readiness and recovery; agents or explicit operator actions become run owners only after `ClaimNextRun` or an equivalent token-fenced transition succeeds.

### Coordinator Trigger

The coordinator auto-spawns only when all conditions are true:

- The workspace has no healthy active coordinator session.
- A task run is enqueued by a publish/start/approve-execution action for coordinated execution. In current AGH terms, the durable trigger is the task-run enqueue boundary and its `task.run_enqueued` event, not `task.created`.
- The workspace-scoped run has a stable `coordination_channel_id` created or resolved by the enqueue path.
- Coordinator auto-start is enabled in resolved coordinator config.
- Spawn caps and permission policy allow the coordinator session.

The trigger is idempotent per workspace. A coordinator cannot spawn another coordinator. Task creation alone does not trigger coordinator startup. Manual user-started sessions do not trigger coordinator startup by themselves. Agent-created tasks inside an existing coordinator workflow inherit that workflow context and do not spawn a second coordinator.

Global-scope task runs do not trigger coordinator auto-spawn in the MVP. They require explicit operator assignment or a future daemon-global coordinator decision.

### Task-Channel Coordination Contract

Every workspace-scoped task run enqueued for coordinated execution has one durable coordination channel association. The task service records `coordination_channel_id` on the run at the enqueue/start boundary. Task creation alone does not create claimable work and does not require a coordination channel.

The channel is the operational conversation surface for coordinator and worker sessions. It is used for status updates, requests, replies, blockers, handoffs, review requests, result exchange, and synthesis context. It is not an ownership or status authority. Claim, heartbeat, complete, fail, release, and terminal task-run state remain task service operations guarded by claim tokens.

The coordinator should always bind a coordinated run to a channel, but it does not need to post chat messages for every internal transition. The rule is bind always, speak when useful. Heartbeats, lease extension, and normal terminal transitions should not be mirrored into channel chatter unless an agent needs human-readable coordination context.

Channel messages that relate to coordinated work must carry typed correlation metadata:

- `task_id`
- `run_id`
- `workflow_id` when present
- `coordination_channel_id`
- `message_kind`
- `correlation_id`

The MVP message kinds are:

- `status`
- `request`
- `reply`
- `blocker`
- `handoff`
- `result`
- `review_request`

Raw `claim_token` values must never appear in channel messages, channel read models, logs, SSE payloads, web payloads, or memory summaries. If an agent needs to prove ownership, it uses the task API with its token; it does not send the token through the network.

The implementation may derive the coordination channel from a task, run, or workflow policy, but the chosen `coordination_channel_id` must be stable on the run and visible in `ClaimNextRun` responses and `/agent/context`.

### Architectural Boundaries

- `internal/daemon` remains the composition root.
- `internal/session`, `internal/task`, `internal/network`, `internal/memory`, `internal/hooks`, and `internal/resources` do not import `daemon`.
- Scheduler logic lives behind narrow interfaces consumed from `internal/scheduler`.
- Hooks are extension/observation boundaries, not the source of safety invariants.
- Durable ownership state lives in `task_runs`; scheduler state is rebuildable.
- The coordinator-agent is a managed session, not a privileged in-process scheduler.
- Manual operator flows are peers of autonomous flows. They use the same task/session APIs and do not bypass claim tokens, leases, hooks, caps, or permission checks.

## Implementation Design

### Core Interfaces

```go
type ClaimCriteria struct {
	WorkspaceID          string
	// ClaimerSessionID identifies the session that will own the run if the claim succeeds.
	// It is never a filter for already-pinned runs.
	ClaimerSessionID     string
	RequiredCapabilities []string
	PriorityMin          int
	LeaseDuration        time.Duration
	Now                  time.Time
}

type ClaimedRun struct {
	TaskID     string
	RunID      string
	ClaimToken string
	LeaseUntil time.Time
}
```

```go
type CapabilityProvider interface {
	CapabilitiesForSession(ctx context.Context, sessionID string) ([]string, error)
}

type IdleAgentRegistry interface {
	Snapshot(ctx context.Context, workspaceID string) ([]IdleAgent, error)
	Rebuild(ctx context.Context) error
}

type TaskClaimer interface {
	ClaimNextRun(ctx context.Context, criteria ClaimCriteria) (ClaimedRun, error)
	HeartbeatRun(ctx context.Context, runID string, claimToken string, until time.Time) error
	ReleaseRun(ctx context.Context, runID string, claimToken string, reason string) error
}
```

```go
type SpawnOpts struct {
	ParentSessionID string
	AgentName       string
	PromptOverlay   string
	AllowedTools    []string
	AllowedSkills   []string
	TTL             time.Duration
	AutoStopOnParent bool
}
```

```go
type PermissionNarrower interface {
	ValidateChild(parent PermissionSet, child PermissionSet) error
}

type CoordinatorConfig struct {
	Enabled      bool
	AgentName    string
	Provider     string
	Model        string
	MaxChildren  int
	DefaultTTL   time.Duration
	ToolAllowlist []string
}

type CoordinatorConfigResolver interface {
	ResolveCoordinatorConfig(ctx context.Context, workspaceID string) (CoordinatorConfig, error)
}
```

### Data Models

**Task run claim fields**

Extend `task_runs` with:

- `claim_token TEXT`
- `lease_until TIMESTAMP`
- `heartbeat_at TIMESTAMP`
- `coordination_channel_id TEXT`

Indexes:

- `(status, lease_until, queued_at)` for ready/expired run discovery.
- `(coordination_channel_id)` for channel-to-run correlation and task-bound inbox summaries.
- Keep using the existing `(session_id, status)` index for session-owned active runs.
- Unique/partial fencing as needed to prevent multiple active claims for the same run.

Do not add duplicate ownership or actor columns. The existing `task_runs.session_id` is the canonical owning session once a run is bound to a session. Existing `claimed_by_kind`/`claimed_by_ref`, `origin_kind`/`origin_ref`, `queued_at`, and task-level `CreatedBy` remain the canonical actor and enqueue provenance fields.

**Task capability fields**

Use side tables for claim-time matching in the MVP:

- `task_run_required_capabilities(run_id TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE, capability_id TEXT NOT NULL, PRIMARY KEY(run_id, capability_id))`
- `task_run_preferred_capabilities(run_id TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE, capability_id TEXT NOT NULL, PRIMARY KEY(run_id, capability_id))`

Add indexes on `(capability_id, run_id)` for both tables. `ClaimNextRun(criteria)` uses these tables for exact capability filtering. Optional capability JSON may exist only as a denormalized prompt/rendering projection; it is not the matching source of truth.

Execution metadata belongs to the run enqueue/start request boundary, not task creation. In the MVP, `workflow_id`, `execution_mode`, `execution_reason`, and coordination-channel policy details live in existing `metadata_json` unless implementation proves that a promoted column is required for query performance or constraints. Do not add `created_by_actor_*`, `started_by_actor_*`, or `execution_requested_at`; the current task and run actor/origin/timestamp fields already cover those facts.

**Coordination channel message metadata**

Task-bound channel messages use typed envelope metadata, preferably in the existing envelope extension field if one exists:

- `task_id`
- `run_id`
- `workflow_id`
- `coordination_channel_id`
- `message_kind`
- `correlation_id`

The network layer validates the MVP `message_kind` enum and rejects raw `claim_token` fields in coordination message metadata. Channel message metadata is correlation and conversation context only; task ownership and terminal state remain in `task_runs`.

**Session lineage fields**

Add or expose through session metadata:

- `parent_session_id`
- `root_session_id`
- `spawn_depth`
- `spawn_role`
- `ttl_expires_at`
- `auto_stop_on_parent`
- `spawn_budget_json`
- `permission_policy_json`

**Coordinator config**

Add global config under `[autonomy.coordinator]` and workspace-level overrides. Resolution precedence:

1. Workspace override
2. Global `[autonomy.coordinator]`
3. Bundled/default coordinator agent definition

The MVP may ship global config first if workspace override plumbing would delay the kernel. The resolver contract must still preserve the precedence order so workspace overrides can be added without changing call sites.

**Domain events vs hooks bridge**

Existing task-domain events such as `task.run_enqueued` remain immutable audit records written by the task service. They are not the same thing as `internal/hooks` dispatch.

For autonomy hooks, the task package consumes a narrow dispatcher interface injected by the daemon. The daemon implementation maps those task-domain hook requests into the existing `internal/hooks` runtime. Safety-sensitive pre hooks, such as `task.run.pre_claim`, dispatch before the transactional state change so they can deny or narrow the request. Post hooks dispatch after the task service commits the state change and writes the audit event. Do not implement this by tailing the task events table; the call site that owns the state transition co-emits the hook.

The coordinator and scheduler wake path uses the post-commit run-enqueue notification from the task service. Hook handlers may observe or deny only where the payload explicitly supports it; they are not the coordinator trigger source of truth.

**Hook payloads**

Add typed payload/patch structs for autonomy events:

- `CoordinatorPreSpawnPayload` / `CoordinatorSpawnPatch`
- `TaskRunPreClaimPayload` / optional deny-or-narrow patch
- `TaskRunLeasePayload` / observation patch
- `TaskRunEnqueuedPayload` / observation patch with `coordination_channel_id`
- `SpawnPreCreatePayload` / `SpawnCreatePatch`

`TaskRunPreClaimPayload` may deny a claim. If mutation is enabled in the MVP, it may only add required-capability constraints or raise `PriorityMin`; it must not remove required capabilities, broaden matching criteria, change claimant identity, or mutate committed claim state directly. Scheduler wake/no-match signals are internal observability events, not hooks, until an external policy use case exists.

**Lease invariants**

- Exactly one active claim token may own a non-terminal run.
- Heartbeat, complete, fail, and release must compare both run ownership and claim token.
- A stale heartbeat after recovery fails with an explicit lease/claim error; it never silently extends a recovered lease.
- A late complete after recovery fails; stale results cannot overwrite a newer claimant's progress.
- Sweep and heartbeat are serialized by SQLite transaction boundaries. Sweep uses compare-and-swap predicates against the observed claim token and lease state.
- Boot recovery runs before the scheduler accepts wake/claim traffic.
- Lease extension is bounded by config. The MVP default should be conservative; a session cannot heartbeat indefinitely without progress.
- A session may hold at most one active task-run lease in the MVP. The cap is configurable later, but the claim transaction must enforce the default cap.

**Permission narrowing**

Spawn permission checks compare concrete atoms, not free-form text. The MVP atom space is tools, skills, MCP server IDs, workspace path grants, network channels, and sandbox profile grants. A child permission set must be a subset of the parent set. Unknown child atoms count as widening and reject the spawn. The daemon rejects invalid spawn requests; it must not silently narrow and continue.

**TTL and active leases**

The reaper wins over active leases. When parent-stop or TTL expiry terminates a spawned session, the daemon releases every active run owned by that session with a structured release reason such as `parent_stopped` or `ttl_expired`, emits task-run and spawn hooks, and then stops the session.

### API Endpoints

These are UDS-first agent endpoints with optional HTTP parity where the web UI or external tooling needs the same data.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/agent/me` | Resolve caller session, workspace, agent, capabilities, active channels, and active task leases. |
| `GET` | `/agent/context` | Return the compact situation payload used by `agh me context`. |
| `GET` | `/agent/channels` | List discoverable channels for the caller workspace/session. |
| `GET` | `/agent/channels/{channel}/recv` | Long-poll or stream channel inbox entries. |
| `POST` | `/agent/channels/{channel}/send` | Send message using caller identity and idempotency key. |
| `POST` | `/agent/channels/reply` | Reply to a delivered message by ID. |
| `POST` | `/agent/tasks/claim-next` | Atomically claim the next run matching criteria. |
| `POST` | `/agent/tasks/{run_id}/heartbeat` | Extend a lease using `claim_token`. |
| `POST` | `/agent/tasks/{run_id}/complete` | Complete a claimed run using `claim_token`. |
| `POST` | `/agent/tasks/{run_id}/fail` | Fail a claimed run using `claim_token`. |
| `POST` | `/agent/tasks/{run_id}/release` | Release a claimed run using `claim_token`. |
| `POST` | `/agent/spawn` | Spawn a child session with lineage, TTL, and narrowed permissions. |
| `GET` | `/agent/telemetry/session` | Return recent activity, loop state, budget state, and hook failures. |

Operator-facing task/session endpoints remain explicit and continue to support manual task creation, manual session creation, direct prompting, and explicit assignment. They may reuse the same contract DTOs, but they must not infer agent identity from environment variables.

Read endpoints exposed over HTTP must return `claim_token_hash`, never raw `claim_token`. The raw token is returned only in the synchronous claim response to the claimant on the issuing transport and is never included in list/detail read models, SSE streams, logs, or web UI payloads.

`/agent/channels` returns channel discovery metadata, including purpose, manifest fields when present, and task-run correlation fields for channels the caller may inspect. Channel send/reply endpoints accept the MVP coordination message kinds and correlation metadata for task-bound messages.

`/agent/context` returns the stable `agh me context` payload in this order: `self`, `workspace`, `session`, `task`, `coordination_channel`, `inbox_summary`, `peer_roster`, `capabilities`, `limits`, and `provenance`. Each list section is bounded and includes truncation metadata; full records come from dedicated endpoints.

CLI commands map one-to-one onto these endpoints:

- `agh me`
- `agh me context`
- `agh ch list`
- `agh ch recv --wait`
- `agh ch send`
- `agh ch reply --to-message <id>`
- `agh task next --wait`
- `agh task heartbeat`
- `agh task complete`
- `agh task fail`
- `agh task release`
- `agh task create` for coordinator and permitted agent-side decomposition
- `agh spawn`

Agent-initiated task creation requires a session-level `task.create` capability atom. Coordinator sessions receive it by default; spawned workers do not unless the parent explicitly grants it and permission narrowing allows it.

`agh task next --wait` returns the claimed run's `coordination_channel_id` and channel display metadata when a channel exists. Agents should use `agh ch send`, `agh ch recv --wait`, and `agh ch reply --to-message` for operational coordination, but must use `agh task heartbeat|complete|fail|release` for ownership and terminal state.

Existing or future operator commands remain explicit, including `agh task create --workspace ...`, `agh task publish --workspace ...`, `agh task start --workspace ...`, and `agh session create --agent ...`. `agh task start` is the operator-facing command that enqueues a run for an executable non-draft task. Operator commands are allowed to start sessions for manual verification without triggering task execution by themselves.

### Hook and Extension Surface

Add hook families/events through the existing `internal/hooks` package and expose them through introspection.

**Coordinator lifecycle**

- `coordinator.pre_spawn`
- `coordinator.spawned`
- `coordinator.decision`
- `coordinator.stopped`
- `coordinator.failed`

**Task run ownership**

- `task.run.enqueued`
- `task.run.pre_claim`
- `task.run.post_claim`
- `task.run.lease_extended`
- `task.run.lease_expired`
- `task.run.lease_recovered`
- `task.run.released`

**Spawn lifecycle**

- `spawn.pre_create`
- `spawn.created`
- `spawn.parent_stopped`
- `spawn.ttl_expired`
- `spawn.reaped`

Workflow is not a first-class MVP entity. Use `workflow_id` as correlation metadata on task/session/coordinator payloads, but do not add a `workflow.*` hook family until a workflow package or store exists.

**Extension rules**

- Hooks may deny or narrow pre-commit requests when the patch type supports it.
- Hooks may annotate criteria or metadata before a claim/spawn operation.
- Hooks may observe committed results.
- Hooks cannot bypass `ClaimNextRun`, claim tokens, lease checks, TTL, lineage, spawn caps, or permission narrowing.
- Hook declaration sources remain config, agent definitions, skills, extensions, and hook binding resources.
- New resource kinds are post-MVP unless a durable user-authored declaration is required. Any new resource kind requires typed codecs, validation, projectors, and actor/scope access rules.
- Scheduler wake/no-match/recovery signals stay in metrics, logs, and task-run observability in the MVP. They are not hook events unless a future policy use case requires them.

## Integration Points

### ACP Agent Providers

Coordinator and worker sessions use the existing provider/agent-definition mechanism. Provider/model selection for the coordinator is config-driven. No new external provider SDK is required for the MVP.

### Generated Contract Surface

`internal/api/contract` is the source of truth for transport-agnostic DTOs. MVP steps that add or rename contract fields must run `make codegen` to regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` in the same implementation unit, then update web type consumers and Storybook/MSW fixtures as needed.

`web/src/generated/agh-openapi.d.ts` feeds `web/src/lib/api-contract.ts` and the `web/src/systems/*/types.ts` derivations. Every MVP step that changes task-run, session-lineage, spawn, coordinator, or agent-context DTOs must pass `make web-typecheck` and `make web-test` in the same PR. The generated HTTP/read surface exposes `claim_token_hash` only; raw `claim_token` stays on the issuing claim response and never reaches web read models.

### SQLite Global Store

Task claim/lease and session lineage persist in the existing global database. SQLite transactions are the safety boundary. The implementation uses transactional compare-and-claim behavior instead of a separate queue.

### Existing Hook and Resource Systems

Autonomy hooks integrate with current hook declaration providers and hook binding resources. Resource extensions use the existing codec/projector/reconciler contracts.

### Existing Network Transport

The MVP targets local daemon autonomy first. Network protocol changes should be minimal and backward-breaking is acceptable in alpha, but broad cross-daemon orchestration remains out of scope.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|----------------------|-----------------|
| `internal/daemon` | modified | New wiring for situation providers, scheduler, coordinator bootstrap, hooks, and config resolution. Medium risk due to composition scope. | Add explicit constructor options and keep daemon as the only composition root. |
| `internal/session` | modified | Session lineage, spawn opts, situation updates, synthetic prompts, lifecycle hooks. High risk around shutdown and parent/child cleanup. | Add focused lifecycle tests and parent-stop/TTL recovery tests. |
| `internal/task` | modified | Claim/lease API, task capability criteria, heartbeat/complete/release fencing. High risk around races. | Implement with SQLite transactions and race-focused tests. |
| `internal/store/globaldb` | modified | Schema changes for claim/lease/session lineage/capability tables and coordination channel IDs. Medium risk. | Update schema and store tests; greenfield alpha allows clean schema change. |
| `internal/scheduler` | new | Daemon-owned sweep/notify loop with idle registry, lease recovery, and capability-aware wakeups. Medium risk around goroutine lifecycle and backpressure. | Keep state rebuildable and context-owned; scheduler must not be a second run claimant in MVP. |
| `internal/cli` | modified | Agent-facing commands with implicit identity and JSON/JSONL output. Medium risk around contract stability. | Add contract tests for env identity, exit codes, and output schema. |
| `internal/api/contract` | modified | DTOs for agent context, claim, lease, spawn, telemetry, coordinator config. Medium risk. | Keep transport-agnostic DTOs and generated OpenAPI parity where needed. |
| `internal/api/udsapi` | modified | UDS endpoints for agent verbs. Medium risk. | Enforce caller session identity and audit fields. |
| Operator task/session surfaces | modified | Manual task creation and manual session creation must remain first-class. Medium risk if autonomy-only assumptions leak into APIs. | Add explicit tests for user-created tasks and user-started sessions entering the same claim/session contracts. |
| `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `web/src/systems/tasks/types.ts`, `web/src/systems/session/types.ts` | modified in MVP | Contract DTO changes in steps 1, 6, 7, 9, and 10 propagate into web typecheck and Storybook/MSW fixtures. Medium risk because web can break even when broad UI is deferred. | Run `make codegen` with every contract change, update generated types and affected web fixtures, and gate on `make web-typecheck` and `make web-test`. Do not expose raw `claim_token` over HTTP. |
| Operator Tasks UI | modified in MVP | The publish/approve/enqueue distinction becomes load-bearing once run enqueue triggers coordinator bootstrap. Low implementation risk but high product-risk if labels imply task creation starts orchestration. | Minimal copy/label/disabled-state pass in `web/src/systems/tasks/components/` plus an e2e scenario for manual-first flows. No new dashboard or autonomy route. |
| `packages/site` runtime docs | modified in MVP | New CLI verbs and runtime concepts would be undocumented at the step 10 demo milestone. Medium risk for adoption and task handoff. | Add minimum `core/autonomy/` docs and CLI reference pages for `agh me`, `agh ch`, `agh spawn`, and new `agh task` verbs; update hook/config/session docs; run site source generation/typecheck/tests. Keep marketing pages unchanged. |
| `internal/network` | modified in MVP | Minimal channel discovery, task-run coordination channel metadata, status, handoff, reply simplification. Medium risk. | Add only task/run correlation metadata and MVP message kinds; defer multi-home, contract-net, vote/react/escalate, and cross-daemon routing. |
| `internal/memory` | modified later | Agent/session provenance, session summaries, recall provenance. Medium risk around concurrent writes. | MVP only fixes useful provenance needed by task/session ownership; broader summaries are post-MVP. |
| `internal/hooks` | modified | New autonomy events, payloads, patches, dispatch, introspection. Medium risk around hook mutability. | Make safety-sensitive hooks observation-only unless explicitly safe. |
| `internal/resources` | unchanged for MVP | Possible coordinator/eval resource kinds are post-MVP. Low risk if deferred. | Do not add resource kinds in MVP unless a durable user-authored declaration is required. |
| `internal/observe` | modified | Agent-callable telemetry, workflow correlation, loop/budget events. Medium risk. | Use existing append-only event and health surfaces. |
| Broad Web UI visibility | modified later | Coordinator dashboards, lease/heartbeat visualization, spawn lineage trees, idle-agent registry views, and autonomy alerts are useful but not required for the kernel. Low for MVP if deferred. | Defer broad UI until kernel contracts stabilize. Existing Tasks/Sessions surfaces only receive contract and labeling fixes in MVP. |

## Testing Approach

### Unit Tests

- `ClaimNextRun` claims exactly one queued run under concurrent calls.
- Claim, heartbeat, complete, fail, and release all require the correct `claim_token`.
- Expired leases become claimable according to policy.
- Stale heartbeat and late complete after lease recovery fail with explicit errors.
- Sweep and heartbeat transactions serialize without duplicate ownership.
- Scheduler wakeups never claim runs directly in MVP.
- Idle registry rebuilds from durable session/task state at boot.
- Coordinator config resolution follows workspace > global > bundled defaults.
- Task creation alone does not auto-spawn a coordinator and does not create a claimable run.
- Task publish/start/approve-execution enqueues coordinated work and triggers coordinator startup when no healthy coordinator exists.
- Workspace-scoped coordinated run enqueue creates or resolves a stable `coordination_channel_id`.
- Coordination channel messages carry task/run correlation metadata and cannot contain raw `claim_token` values.
- Channel `status` and `result` messages do not mutate task-run ownership or terminal state.
- Spawn options reject permission widening, excessive depth, excessive children, missing TTL, and invalid parent session.
- Permission narrowing rejects unknown child atoms and does not silently narrow.
- TTL/parent-stop releases active leases with structured release reasons.
- Agent CLI identity resolution rejects missing/invalid caller identity.
- Situation providers render stable, bounded context and omit unavailable sections.
- Hook descriptors list every new autonomy event with payload and patch schema.
- Hook patches cannot widen permissions or claim a run outside transactional APIs.

### Integration Tests

- User-created task -> user-started session -> `agh task next --wait` -> agent heartbeat -> run complete.
- Coordinator-created task -> worker session -> `ClaimNextRun` -> completion through the same lease contract.
- Task queued -> scheduler wake -> idle agent pulls with `ClaimNextRun` -> run complete.
- User-created task remains draft/blocked/ready after creation -> no coordinator auto-spawn and no claimable run until explicit execution.
- User publishes/starts/approves that task -> run enqueue emits `task.run_enqueued` -> coordinator spawns -> coordinator decomposes/delegates -> worker claims through `ClaimNextRun`.
- Coordinated task run -> claim response includes `coordination_channel_id` -> worker exchanges `status`/`blocker`/`result` messages through the channel -> completion still happens only through token-fenced task API.
- Parent session stop -> child auto-stop -> active child leases release with `parent_stopped`.
- Lease expiry on daemon restart -> recovery makes the run claimable without duplicate completion.
- `agh ch reply --to-message` replaces delivery shell-snippet guidance using implicit identity.
- Hook binding resource registers an autonomy hook and receives the typed payload.
- Session-end summary writes memory with correct `agent_name`, `session_id`, and provenance.
- Loop detector injects recovery prompt before hard cancel when configured.
- Eval/replay harness can replay a recorded autonomy flow without live provider calls.

### Web and Docs Tests

- Contract-changing MVP tasks regenerate `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts` with `make codegen`, then pass `make web-typecheck` and `make web-test`.
- Tasks UI e2e coverage proves creation does not imply execution, run enqueue is explicit, and coordinator bootstrap happens only after execution is allowed.
- Manual-first e2e coverage proves a user can start a session and prompt it directly without coordinator startup.
- Site docs additions render through `cd packages/site && bun run source:generate`, then pass `cd packages/site && bun run typecheck` and `cd packages/site && bun run test`.
- Storybook/MSW contract fixtures are updated when generated DTOs change.

### Verification Gate

Implementation tasks must pass `make verify`. Integration-heavy tasks should also add targeted `make test-integration` coverage where they touch daemon/session/task scheduling behavior. Tasks that touch `web/` must also pass `make web-typecheck` and `make web-test`; tasks that touch `packages/site` must pass the site source generation, typecheck, and test commands.

## Development Sequencing

### Build Order

Steps 1-10 are the local autonomy MVP. They should feed the first `$cy-create-tasks` decomposition. Steps 11-15 are follow-on TechSpecs unless the user explicitly expands the MVP scope again.

1. **Autonomy contracts and config** - no dependencies. Add config structs, contract DTOs, coordinator config resolution, and no-op feature flags.
2. **Autonomy hook taxonomy** - depends on step 1. Add `coordinator.*`, `spawn.*`, and `task.run.*` events, payloads, patches, dispatch, introspection, and bridge interfaces before behavior depends on them. Scheduler wake/no-match/recovery remain internal observability in the MVP.
3. **Situation Surface** - depends on steps 1 and 2. Add prompt providers, self-capability rendering, task context rendering, and bounded dynamic situation updates.
4. **Agent Kernel CLI identity layer** - depends on step 1. Add caller identity resolution, UDS audit fields, JSON/JSONL conventions, and exit-code taxonomy.
5. **Channel and self-context verbs** - depends on steps 3 and 4. Add `agh me`, `agh me context`, `agh ch recv --wait`, `agh ch reply`, channel discovery/listing, and MVP coordination message metadata/kinds.
6. **Task claim/lease schema and store API** - depends on steps 1 and 2. Add `task_runs` claim fields, `coordination_channel_id`, capability side tables, `ClaimNextRun`, heartbeat, release, fencing tests, lease invariants, and boot recovery ordering.
7. **Agent and operator task verbs** - depends on steps 4 and 6. Add `agh task next`, heartbeat, complete, fail, release, permitted `agh task create`, explicit operator publish/start commands, and claim responses that include coordination-channel metadata; keep existing operator task/session commands functional.
   Demo milestone: at this point a user-started agent can self-claim and complete a queued task end-to-end without the scheduler or coordinator.
8. **Mechanical scheduler sweep/notify** - depends on steps 2, 3, 6, and 7. Add `internal/scheduler`, idle-agent registry, boot rebuild, capability-aware wakeups, and lease sweep. Do not make it a direct run claimant in MVP.
9. **Safe spawn and lineage** - depends on steps 1, 2, and 4. Add `SpawnOpts`, lineage fields, TTL, caps, permission narrowing, parent-aware reaper, and `agh spawn`.
10. **Coordinator-agent bootstrap** - depends on steps 3, 7, 8, and 9. Add task publish/start/approve-execution trigger at the run enqueue boundary, coordination-channel binding/usage, restricted tools, provider/model config, one active coordinator per workspace, and manual override controls.
    Co-ship requirement: step 10 must include the operator Tasks UI copy/labeling pass for the publish/enqueue/coordinator-trigger boundary, one web e2e scenario covering ADR-010 manual-first bookends, and the minimum docs set under `packages/site/content/runtime/core/autonomy/` plus CLI reference pages for `agh me`, `agh ch`, `agh spawn`, and new `agh task` verbs.
11. **Post-MVP network evolution** - depends on MVP validation. Add multi-home sessions, richer peer negotiation, contract-net verbs, and broader handoff/mention semantics only after local autonomy proves the needed wire shape.
12. **Post-MVP memory provenance and session summaries** - depends on MVP validation. Write broader recall provenance and add session-end summaries before broad extraction.
13. **Post-MVP self-correction and telemetry** - depends on MVP validation. Add repetition detector, recovery prompts, agent-callable telemetry, and autonomy alerts beyond the minimal counters.
14. **Post-MVP eval/replay harness** - depends on MVP validation. Add recorded ACP/session trajectories, YAML cases, replay fixtures, and deterministic assertions under a dedicated ADR/TechSpec.
15. **Post-MVP web visibility** - depends on stable backend contracts. Add UI read models for coordinator, leases, child sessions, scheduler state, and alerts after the kernel stabilizes.

### Technical Dependencies

- Existing global SQLite store remains the durable coordination backend.
- Existing session manager remains responsible for process lifecycle and ACP provider execution.
- Existing hooks/resources systems remain the extension substrate.
- Existing CLI/UDS stack remains the agent-callable local control plane.
- Existing operator task/session flows remain supported and explicit.
- Generated OpenAPI and web TypeScript contracts must stay in lockstep with API contract changes.
- Runtime docs must co-ship with the step 10 autonomy demo milestone.
- No external service, NATS cluster, vector database, or remote control plane is required for the MVP.

## Monitoring and Observability

### Metrics and Events

- `scheduler.wake.count`
- `scheduler.wake.no_match`
- `scheduler.lease_sweep.count`
- `scheduler.lease_sweep.error`
- `task.run.claim.success`
- `task.run.claim.error`
- `task.run.lease.extended`
- `task.run.lease.expired`
- `task.run.lease.recovered`
- `coordinator.spawned`
- `coordinator.failed`
- `spawn.created`
- `spawn.rejected`
- `spawn.reaped`
- `session.loop.detected`
- `session.budget.exceeded`
- `manual.task.created`
- `manual.session.created`

### Structured Log Fields

- `workspace_id`
- `session_id`
- `parent_session_id`
- `root_session_id`
- `agent_name`
- `task_id`
- `run_id`
- `claim_token_hash`
- `lease_until`
- `workflow_id`
- `coordinator_session_id`
- `scheduler_reason`
- `hook_event`
- `hook_name`
- `spawn_depth`
- `actor_kind`
- `actor_id`
- `release_reason`

### Alert Conditions

- Lease expiry rate exceeds normal threshold for a workspace.
- Scheduler has ready runs but no eligible idle agents for a sustained period.
- Coordinator spawn repeatedly fails for a workspace.
- Parent session exits while children or active leases remain.
- Loop detector trips repeatedly for the same session or workflow.
- Budget circuit breaker cancels a coordinator session.
- Required autonomy hook fails repeatedly.

## Technical Considerations

### Key Decisions

**Decision: Build a broad phased autonomy kernel, not a narrow one-off MVP.**

Rationale: the approved scope covers items 1-9, and `cy-create-tasks` will split implementation later. A broad TechSpec prevents early local decisions from blocking later autonomy behavior.

Trade-off: the design document is larger. Sequencing keeps implementation incremental, and the first decomposition should cut the MVP at steps 1-10.

**Decision: Keep manual operator control first-class.**

Rationale: autonomy should assist and orchestrate work, not remove user agency. Users must still create tasks, start sessions, prompt sessions directly, and use manual sessions for verification.

Trade-off: APIs must support both explicit operator identity and implicit agent identity. The benefit is one shared task/session lifecycle instead of parallel manual and autonomous systems.

**Decision: Extend `task_runs` for claim/lease instead of adding a scheduler-owned durable queue.**

Rationale: the current task service already owns run lifecycle and recovery. Adding claim fields keeps ownership centralized and avoids duplicate durable state.

Trade-off: the task store becomes more complex and needs strong race tests.

**Decision: Split coordinator-agent semantics from daemon scheduler mechanics.**

Rationale: LLMs are useful for decomposition and validation; deterministic daemon code is required for wakeups, lease sweep, recovery, and permission safety. The MVP scheduler does not directly claim runs; the owning session or explicit operator action claims through `ClaimNextRun`.

Trade-off: two components must cooperate through clear boundaries.

**Decision: Spawn coordinator on coordinated run enqueue and make provider/model configurable.**

Rationale: avoids idle coordinator sessions while making coordinated execution the default once a user publishes, starts, or approves a task for execution. Task creation remains manual and lightweight; execution orchestration starts only when work is allowed to run and a run is enqueued.

Trade-off: first-work latency includes coordinator startup unless prewarmed later.

**Decision: Safe spawn uses hard caps and permission narrowing.**

Rationale: autonomous delegation needs bounded blast radius.

Trade-off: some advanced workflows will need explicit future limit increases. The MVP rejects any child permission atom not present in the parent and releases active leases when TTL or parent-stop reaps a child session.

**Decision: Autonomy extensibility uses hooks/resources, not a new plugin system.**

Rationale: AGH already has typed hooks, introspection, hook binding resources, extension declarations, and resource reconciliation.

Trade-off: each new autonomy behavior must include payload/patch/introspection work. The MVP adds typed hook events but avoids new resource kinds unless a durable user-authored declaration is required.

**Decision: Generated contracts and docs co-ship with autonomy MVP steps.**

Rationale: autonomy changes transport DTOs and operator semantics even when broad UI is deferred. Generated web contracts and minimum docs must move with the kernel so the workspace remains buildable and the step 10 demo is self-explanatory.

Trade-off: MVP tasks touching contracts carry small frontend/docs obligations. The scope is intentionally limited to generated types, task UI labeling, e2e coverage, and runtime docs; dashboards and marketing remain post-MVP.

### Known Risks

**Race conditions in claim/lease**

Likelihood: high. Mitigation: keep claim in one SQLite transaction, require claim tokens for state changes, reject stale heartbeat/late complete after recovery, run boot recovery before accepting wake/claim traffic, and add concurrent tests.

**Coordinator overreach**

Likelihood: medium. Mitigation: coordinator can create tasks and request spawn, but daemon owns claim, task-run lease, caps, and permissions. Creating a task does not start a coordinator or create claimable work; publishing, starting, or approving execution does.

**Hook mutability weakening safety**

Likelihood: medium. Mitigation: safety invariants are enforced after hook patches and inside daemon-owned transactions. Observation-only hooks remain observation-only.

**Manual and autonomous paths diverge**

Likelihood: medium. Mitigation: user-created tasks, coordinator-created tasks, user-started sessions, and spawned sessions all use the same task/session/claim/lease contracts. Operator commands stay explicit; agent commands infer identity.

**Prompt bloat from situation data**

Likelihood: medium. Mitigation: render bounded sections, include summaries by default, expose full details through `agh me context`, and refresh only deltas per turn.

**Spawn lifecycle leaks**

Likelihood: medium. Mitigation: TTL, parent-aware reaper, boot recovery, and child auto-stop are mandatory in the spawn implementation.

**Overbuilding network protocol before local autonomy works**

Likelihood: medium. Mitigation: local autonomy comes first; cross-daemon swarm, election, and broad contract-net semantics stay out of MVP.

**Channels becoming hidden task state**

Likelihood: medium. Mitigation: coordination channels are operational conversation only. Tests and docs must prove claim, heartbeat, complete, fail, release, and terminal task status remain token-fenced task service operations.

**Memory extraction writes noisy facts**

Likelihood: medium. Mitigation: start with session summaries and provenance, then add broader turn/network extraction after telemetry shows useful signal.

## Architecture Decision Records

- [ADR-001: Phased Autonomy Kernel Scope](adrs/adr-001.md) - Implement the cleaned items 1-9 as a phased autonomy kernel, excluding only explicitly out-of-scope swarm/trust/overbuilt items.
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - Make identity-implicit CLI/UDS verbs the first agent control surface; MCP tools can mirror them later.
- [ADR-003: Extend Task Runs for Atomic Claim and Lease](adrs/adr-003.md) - Add claim and lease fields to `task_runs` and implement transactional `ClaimNextRun(criteria)` instead of a parallel queue.
- [ADR-004: Split Semantic Coordination from Mechanical Scheduling](adrs/adr-004.md) - Use a coordinator-agent for semantic orchestration and a daemon scheduler for sweep/notify/recovery safety.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - Spawn one coordinator per workspace when a task run is enqueued by publish/start/approval for coordinated execution.
- [ADR-006: Safe Spawn Requires Lineage, TTL, and Permission Narrowing](adrs/adr-006.md) - Require parent lineage, TTL, hard caps, auto-stop behavior, and child permission narrowing for spawned sessions.
- [ADR-007: Minimal Network Evolution for Local Autonomy](adrs/adr-007.md) - Add only the channel/status/handoff pieces needed for local autonomy before broader cross-daemon swarm work.
- [ADR-008: Memory Provenance Before Rich Memory Scopes](adrs/adr-008.md) - Start memory work with provenance, session summaries, and recall metadata before adding broad peer/channel scopes.
- [ADR-009: Autonomy Hooks and Extension Points Are First-Class Contracts](adrs/adr-009.md) - Add typed autonomy hooks and resource/provider extension contracts through existing AGH extensibility systems.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - Preserve manual task creation and manual session starts as peers of autonomous workflows on the same task/session contracts.
- [ADR-011: Generated Contracts and Documentation Co-Ship with Autonomy MVP Steps](adrs/adr-011.md) - Keep generated web contracts, minimal Tasks UI labeling, and runtime docs in lockstep with autonomy contract changes.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - Bind each coordinated run to a stable network channel for operational agent communication without making channels the task ownership authority.
