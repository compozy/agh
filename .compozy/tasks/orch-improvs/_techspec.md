# TechSpec: Orchestration Improvements Program

## Executive Summary

AGH will harden its existing task orchestration substrate, add typed per-task execution profiles, and add a native review-gate continuation loop inspired by the Codex Loop `goal` mechanism. This aggregate TechSpec is the canonical implementation contract for `.compozy/tasks/orch-improvs`; the detailed domain designs live in two child TechSpecs:

- [`_techspec_orchestration.md`](_techspec_orchestration.md): orchestration hardening, task-run projections, `TaskExecutionProfile` worker/coordinator/participant/sandbox overlays, context bundle, cursor-seeded SSE, bundled orchestration skills, durable notification cursors, and bridge terminal notifications.
- [`_techspec_review_gate.md`](_techspec_review_gate.md): post-terminal task-run review gate, typed reviewer verdicts, `TaskExecutionProfile.Review` reviewer selection, reviewer routing through coordination channels/capabilities, and next-round guidance.

The program deliberately keeps runtime authority narrow. `task.Service`, `task_runs`, task events, session-bound lease lookup, native tool policy, coordinator runtime, scheduler boundaries, and spawn lineage remain the authority surfaces. Network channels, bridge threads, skills, notification cursors, prompt overlays, and task execution profiles are collaboration, configuration, selection, and delivery inputs only; they do not own task state, review state, queue semantics, permission boundaries, or terminal state.

The primary trade-off is accepting more typed persistence, migrations, API/UDS/CLI surface area, codegen, and test coverage in exchange for deterministic orchestration, replayable observability, durable terminal notifications, and review-on-stop behavior that does not depend on prompt memory or channel transcripts.

## Normative Sources and Precedence

This aggregate is the canonical coordination spec for implementation order, shared invariants, task generation, cross-feature tests, and peer-review closure. It does not replace the child specs.

Precedence for implementation:

1. ADRs in [`adrs/`](adrs/) are authoritative for accepted decisions.
2. This aggregate [`_techspec.md`](_techspec.md) is authoritative for MVP boundary, sequencing, shared authority model, package boundaries, consolidated surfaces, and cross-feature QA.
3. [`_techspec_orchestration.md`](_techspec_orchestration.md) is authoritative for orchestration hardening internals, including context bundle, SSE, bundled orchestration skills, notification cursor primitive, and bridge terminal notifier details.
4. [`_techspec_review_gate.md`](_techspec_review_gate.md) is authoritative for review-gate internals, including review policy, reviewer resolution, persisted verdicts, continuation run creation, reviewer skill guidance, and review events.

If an implementation agent finds a conflict, update the source ADR or child TechSpec first. Do not silently resolve conflicts inside code, docs, tests, generated contracts, or task files.

## Architectural Boundaries

This aggregate preserves the package boundaries detailed in the child specs:

1. `internal/task` owns task/run transitions, task events, task execution profiles, review requests, review verdicts, continuation run creation, summaries, and projections.
2. `internal/store/globaldb` persists task/run/profile/review/notification state through numbered migrations only.
3. `internal/daemon` remains the composition root for scheduler, coordinator runtime, session start, skills, notifications, bridge consumers, HTTP, and UDS wiring.
4. `internal/scheduler` observes, wakes, and requests task-service-owned recovery; it must not claim, assign, complete, fail, or review runs.
5. `internal/coordinator` guides work and routes reviewers; it must not become queue, claim, terminal-state, or review-verdict authority.
6. `internal/session` applies effective task profile session options but must not query or mutate task authority directly.
7. `internal/notifications` stores durable cursor progress only; it must not become a queue, hook dispatcher, review bus, or fan-out policy engine.
8. `internal/api/contract` owns OpenAPI/generated DTO shapes for task/profile/review/notification surfaces; transport packages must not invent API-only contract fields.
9. HTTP and UDS transport packages mount shared `internal/api/core` handlers instead of duplicating task/profile/review semantics.

## Goals

- Preserve the existing AGH autonomy substrate while making orchestration state queryable, bounded, replayable, and agent-operable.
- Keep `task_runs` as the only durable execution queue and ownership source.
- Add task context bundles and run summaries so workers, coordinators, reviewers, operators, and web clients can resume work without relying on channel backlog.
- Add cursor-seeded task SSE using `latest_event_seq` so web clients can subscribe without read/stream races.
- Add deterministic bundled orchestration skills: `agh-orchestrator`, `agh-task-worker`, and `agh-task-reviewer`.
- Add `TaskExecutionProfile` as typed task-owned overlays for coordinator guidance, worker selection, review selection, participant policy, and sandbox selection.
- Add `internal/notifications` as a durable cursor primitive and `internal/bridges` terminal task notification consumer.
- Add a post-terminal review gate that can reject incomplete work and enqueue a next run with typed `missing_work` and `next_round_guidance`.
- Route reviewer selection through channels, peers, and capabilities without letting channel messages become verdict authority.
- Expose the full program through CLI, HTTP, UDS, native tools, hooks, generated contracts, and docs.

## Non-Goals

- Do not introduce a second task queue, workflow engine, event bus, dispatcher, or scheduler claim path.
- Do not make channels, bridge threads, prompt overlays, skills, notification cursors, or UI state authoritative for task ownership, run ownership, terminal state, or review verdicts.
- Do not expose raw `claim_token` through HTTP, UDS responses, SSE, channels, logs, web state, skills, memory, docs examples, or review packets.
- Do not implement a `pending_review` run status in MVP. Review gate v1 is a post-terminal continuation loop.
- Do not implement multi-reviewer voting, quorum, consensus, or generic human approval matrices in MVP.
- Do not turn `internal/notifications` into a review workflow bus. It remains a durable delivery cursor primitive.
- Do not make bridge-delivered review workflows the primary review gate path in MVP.
- Do not implement dedicated per-task coordinators in MVP. Coordinator profile mode is limited to `inherit` and `guided`; the daemon-managed workspace coordinator remains the runtime model.
- Do not store task-specific runtime selection only in `metadata_json`.
- Do not let participant policy grant task ownership, review authority, channel permission, or terminal-state permission.
- Do not let per-task sandbox selection bypass tool policy, approval policy, provider authorization, or session authorization.
- Do not fork external formats. If future interop metadata is needed, use namespaced `agh.*` / `metadata.agh.*` extensions.

## MVP Boundary

The MVP has two implementation tracks that share the same task runtime and authority model.

### Track 1: Orchestration Hardening

The orchestration child spec includes:

- Core 1-7 from the orchestration-improvements synthesis.
- Typed task/run orchestration projections, including summaries, max-runtime fields, spawn-failure counters, and `tasks.current_run_id` as a read projection only.
- Task context bundle enrichment through `/agent/context`.
- Cursor-seeded SSE using `latest_event_seq`.
- Bundled `agh-orchestrator` and `agh-task-worker` skills.
- Scheduler health telemetry without scheduler-owned claims.
- Spawn-failure circuit breaker and max-runtime enforcement.
- `TaskExecutionProfile v1` for coordinator `inherit|guided` mode, worker agent/provider/model selection, participant policy, and sandbox `inherit|none|ref` selection.
- `internal/notifications` cursor primitive.
- `internal/bridges` task terminal bridge notifier as the first concrete cursor consumer.

### Track 2: Review Gate

The review-gate child spec includes:

- Task/run review policies with default-off behavior.
- Review profile integration with task execution profiles, including reviewer agent/provider/model hints plus peer/channel/capability selectors.
- Post-terminal review request creation after completed/failed/canceled runs when policy requires review.
- Reviewer resolution by explicit peer, channel membership, and capability filters.
- A bundled `agh-task-reviewer` skill for one-shot reviewer sessions.
- Typed persisted review verdicts: `approved`, `rejected`, `blocked`, `error`, `timeout`, and `invalid_output`.
- Bounded review evidence, `missing_work`, and `next_round_guidance`.
- Next-run creation on rejected reviews until configured max rounds are exhausted, with continuation-run lineage persisted on `task_runs`.
- Review circuit breaker behavior for exhausted rounds, reviewer failures, rapid shallow attempts, or invalid reviewer output.
- Review history APIs, UDS parity, CLI commands, native reviewer tool submission, hooks, observe events, and context-bundle continuation fields.

## Delete Targets

This program is additive to the current alpha runtime and has no legacy production state to preserve. It must not add compatibility shims, dual-field aliases, fallback readers, or silent downgrade paths.

The implementation must delete or replace these planned targets when the corresponding functionality lands:

- Prompt-only worker completion summaries for task-run state. Summaries must become bounded typed run/task fields.
- Any ad hoc bridge terminal notification cursor state. Delivery progress belongs in `notification_cursors`; delivery targets belong in `bridge_task_subscriptions`.
- Any future attempt to keep reviewer verdicts only in channel transcripts, prompt text, or session summaries. Verdicts must be persisted in task-owned review tables.
- Any task-specific worker, reviewer, participant, coordinator, or sandbox selection stored only in `metadata_json`.
- Workspace-default-only worker session bootstrap once `TaskExecutionProfile.Worker` lands. Task profiles must feed session start and claim eligibility before workspace defaults are applied.
- Any public helper that lets scheduler, coordinator, API, bridge, notification, extension, or web code mutate `tasks.current_run_id` directly.

## Safety Invariants and Shared Authority Model

The authority rules below apply to both child specs.

1. `task_runs` is the only durable execution queue and ownership source.
2. `ClaimNextRun` is the only authoritative next-work claim primitive.
3. `task.Service` owns task/run transitions, review request creation, review verdict recording, continuation run creation, terminal task mutation, run summaries, task events, and projection updates.
4. Session-bound lease lookup is the only path for agent heartbeat, complete, fail, and release.
5. Scheduler code may observe, wake, recover expired leases, and trigger max-runtime enforcement through task-service-owned transitions. It must not claim, assign, complete, fail, or review runs directly.
6. Coordinator code may guide work, spawn sessions, route reviewers, and coordinate through channels. It must not become ownership authority, queue authority, or terminal-state authority.
7. Network channels carry conversation, handoff, blocker, result, and review-request coordination content. They do not define ownership, terminal state, review verdicts, or replay authority.
8. Bundled skills are instruction bundles only. They do not define permissions, queue semantics, task ownership, review authority, or terminal state.
9. `internal/notifications` stores confirmed delivery progress only. It does not own task authority, hook dispatch, queue semantics, event fan-out policy, or review workflow state.
10. `internal/bridges` owns bridge subscription/target state and confirmed delivery, not task/review authority.
11. `tasks.current_run_id` is a denormalized read projection only.
12. Review verdicts are authoritative only after `task.Service.RecordRunReview` persists them through a native tool/API/UDS/CLI path.
13. `TaskExecutionProfile` is typed task-owned configuration and selection input. It does not grant ownership, review authority, channel permission, sandbox permission, or terminal-state authority.
14. `CoordinatorProfile.mode = "guided"` applies task-specific guidance to the daemon-managed workspace coordinator. It does not create a dedicated per-task coordinator or relax `max_active_per_workspace`.
15. `ParticipantPolicy` constrains eligible coordination surfaces at coordinator routing, worker claim filtering, safe-spawn/session-start grants, and review routing. Runtime permission checks still come from channel membership, peer identity, task ownership, review binding, and tool policy.
16. `SandboxPolicy` selects session sandbox behavior at session start. It cannot bypass approval policy, tool allowlists, provider authorization, or session authorization.

## Core Type Summary

The child specs own the detailed interfaces. The aggregate contract uses this shared shape for task execution profiles:

```go
type TaskExecutionProfile struct {
	Coordinator  CoordinatorProfile `json:"coordinator,omitempty"`
	Worker       WorkerProfile      `json:"worker,omitempty"`
	Review       ReviewProfile      `json:"review,omitempty"`
	Participants ParticipantPolicy  `json:"participants,omitempty"`
	Sandbox      SandboxPolicy      `json:"sandbox,omitempty"`
}
```

## Data Model Rationale

The program uses typed columns and side tables for every queryable runtime decision:

| Field | Owner | Rationale |
| --- | --- | --- |
| `task_runs.summary` | `internal/task` | Bounded handoff/result summary read by context, web, CLI, and review. |
| `tasks.current_run_id` | `internal/task` | Denormalized read projection only; never queue or ownership authority. |
| `tasks.max_runtime_seconds` | `internal/task` | Queryable per-task watchdog override. |
| `tasks.spawn_failure_count` / `tasks.last_spawn_error` | `internal/task` | Queryable spawn circuit state. |
| `task_runs.review_required`, `task_runs.review_request_round`, `task_runs.review_policy_snapshot`, `task_runs.review_request_id` | `internal/task` | Typed review-request trigger state and recovery marker for terminal runs that require review. These columns are owned by the review-gate migration. |
| `task_runs.parent_run_id`, `task_runs.review_id`, `task_runs.review_round`, `task_runs.continuation_reason`, `task_runs.missing_work_json`, `task_runs.next_round_guidance` | `internal/task` | Typed continuation-run lineage and worker guidance for review-driven follow-up runs. These columns are owned by the review-gate migration. |
| `task_runs.claimed_agent_name`, `task_runs.claimed_peer_id`, `task_runs.terminalized_by_session_id`, `task_runs.terminalized_by_agent_name`, `task_runs.terminalized_by_peer_id`, `task_runs.terminalized_by_actor_kind`, `task_runs.terminalized_by_actor_ref` | `internal/task` | Run provenance used for audit, routing diagnostics, and original-worker review exclusion. |
| `task_execution_profiles` | `internal/task` | Typed task-owned coordinator/worker/review/participant/sandbox profile scalars. |
| `task_profile_agents`, `task_profile_channels`, `task_profile_peers`, `task_profile_capabilities` | `internal/task` | Exact-match selector side tables for worker eligibility, reviewer routing, and participant policy. |
| `notification_cursors` | `internal/notifications` | Durable confirmed delivery progress by consumer/stream/subject. |
| `bridge_task_subscriptions` | `internal/bridges` | Bridge terminal-notification target state. |
| `task_run_reviews` | `internal/task` | Typed persisted review requests, route metadata, verdicts, and continuation guidance. |

Side-table-vs-JSON decision: `metadata_json` and `result_json` remain opaque payloads only. Queryable orchestration, profile, notification, and review state uses columns or side tables because it needs validation, indexes, generated contracts, and deterministic replay.

## End-to-End Lifecycle

```text
create task
  -> task.Service validates and persists TaskExecutionProfile
  -> enqueue run through task.Service
  -> coordinator observes eligible run
  -> coordinator loads agh-orchestrator deterministically with task guided profile when present
  -> worker selection applies WorkerProfile and ParticipantPolicy eligibility
  -> worker session starts with effective agent/provider/model/sandbox profile
  -> worker claims run through ClaimNextRun / session-bound lease
  -> worker uses agh-task-worker, /agent/context, channels, heartbeat, summary fields
  -> run terminalizes through task.Service
  -> task_events receives durable terminal event
  -> if review policy is none: terminal outcome stands
  -> if review policy applies: task.Service creates or returns attempt-1 task_run_reviews row in follow-up review transaction
  -> follow-up review transaction clears task_runs.review_required and writes task_runs.review_request_id
  -> daemon wakes coordinator routing through typed ReviewRouter callback
  -> coordinator routes reviewer through ReviewProfile peer/channel/capability/agent policy
  -> reviewer session is bound to the persisted review request and loads agh-task-reviewer
  -> reviewer submits typed verdict through submit_run_review
  -> approved: terminal outcome is accepted
  -> rejected: task.Service records verdict and enqueues a continuation run with typed guidance in one transaction
  -> blocked/error/timeout/invalid_output: task.Service applies bounded failure policy
  -> bridge terminal notifier replays task_events and delivers only after an accepted final terminal event
  -> notification cursor advances only after confirmed delivery
```

The lifecycle intentionally separates execution terminal state from review acceptance. A completed run may be rejected by review and followed by another queued run, but the completed run's terminal status remains historically true.

## Component Ownership

| Component | Track | Responsibility | Boundary |
| --- | --- | --- | --- |
| `internal/task` | Both | Task/run transitions, task events, summaries, projections, task execution profile validation, review requests, review verdicts, continuation run creation | Owns authority; must not import scheduler/coordinator/daemon/web |
| `internal/store/globaldb` | Both | Numbered migrations for task/run projections, task execution profiles, profile selectors, notifications, bridge subscriptions, review policies, review attempts | Persistence only; no orchestration policy |
| `internal/situation` | Both | Bounded task context bundle and review continuation context | No parallel memory system |
| `internal/scheduler` | Orchestration | Health telemetry, wake/recover, max-runtime trigger | No claims or review verdicts |
| `internal/coordinator` | Both | Coordinator decisions, guided profile interpretation, reviewer routing helpers, prompt overlays | No queue or terminal authority |
| `internal/session` | Orchestration | Session start options for effective task agent/provider/model/sandbox profile | No task authority or permission bypass |
| `internal/skills/bundled` | Both | `agh-orchestrator`, `agh-task-worker`, `agh-task-reviewer` content | Instructional only |
| `internal/notifications` | Orchestration | Durable cursor primitive | No event bus, review bus, or hook dispatcher |
| `internal/bridges` | Orchestration | Bridge task subscriptions and terminal notification delivery | Replay from `task_events`, not channel state |
| `internal/api/contract` | Both | OpenAPI/generated DTOs for task/profile/review/notification payloads | Codegen co-ships with contract changes |
| `internal/api/core` | Both | Shared HTTP/UDS semantics | Generated DTO parity; no duplicated transport behavior |
| `cmd/agh` | Both | Agent-operable CLI surfaces | JSON-capable, deterministic errors |
| `web/src/systems/tasks` | Both | Generated DTO consumption, task streams, review read models | No local authority inference |
| `packages/site` | Both | Runtime-truth docs | Not an implementation fallback |

## Data Ownership Matrix

| Data | Owner | Authority Notes |
| --- | --- | --- |
| `tasks.status` | `task.Service` | Derived through task-service transitions and canonical status rules. |
| `task_runs.status` | `task.Service` | Execution history; review gate does not rewrite terminal run status. |
| `task_events.event_seq` | `task.Service` / store | Durable replay authority for task SSE and bridge terminal notifier. |
| `tasks.current_run_id` | `task.Service` | Denormalized read projection only. |
| Run summary/result/error | `task.Service` | Bounded typed fields; channel summaries are not storage. |
| Review trigger and continuation-run lineage/guidance | `task.Service` / store | Review-gate migration owns these `task_runs` columns because they reference `task_run_reviews`; trigger fields are cleared/linked when the review request row is durable, and continuation fields are copied from rejected review verdicts for next worker context. |
| Task context bundle | `internal/situation` assembled from task-owned state | Read-only projection; not authority. |
| Notification cursor | `internal/notifications` | Confirmed delivery progress only. |
| Bridge subscription | `internal/bridges` | Delivery target state only. |
| Review policy | `task.Service` / store | Task-owned policy with config defaults. |
| Task execution profile | `task.Service` / store | Typed task-owned selection input; not authority. |
| Coordinator profile | `task.Service` / coordinator runtime | `inherit` or `guided`; no dedicated coordinator in MVP. |
| Worker profile | `task.Service` / daemon session start | Feeds worker selection, session create/start, and claim eligibility. |
| Participant policy | `task.Service` / coordinator runtime | Eligible channels/peers/agents/capabilities only; not permission authority. |
| Sandbox policy | `task.Service` / session runtime | Session sandbox selection only; no tool-policy bypass. |
| Review request/attempt/verdict | `task.Service` / store | Persisted typed state; channel message is never verdict. |
| Reviewer channel messages | `internal/network` / bridge runtime | Coordination transport only. |
| Bundled skill content | `internal/skills/bundled` | Instructional only. |

## Surface Matrix

| Surface | Orchestration Hardening | Review Gate |
| --- | --- | --- |
| HTTP | Task detail/list summary fields, task context, task SSE seed, task execution profile CRUD, bridge notification subscription CRUD | Review request, verdict submission, review history, review policy/profile update |
| UDS | Parity with shared `internal/api/core` task/context/profile/notification surfaces | Parity with review request, verdict, policy/profile, and history surfaces |
| CLI | Task inspect/context/profile/notification commands with JSON output | `agh task review request`, `submit`, `list`, `show`, policy/profile inspect/update |
| Native tools | Session-bound claim/heartbeat/complete/fail/release with summaries | `submit_run_review`; optional review request tool for coordinator sessions |
| Hooks | Existing task lifecycle hooks plus notification cursor hooks | `task.run_review_requested`, `task.run_review_routed`, `task.run_review_recorded`, `task.run_review_approved`, `task.run_review_rejected`, `task.run_review_circuit_opened`, `task.run_review_retry_enqueued` |
| Observe/SSE | `latest_event_seq` seeded stream replay | Review events included in task stream after they are persisted |
| OpenAPI/codegen | `openapi/agh.json`, generated TypeScript, contract tests | Same generated contract artifacts |
| Web | Task summaries, active run projection, scheduler health, notification subscription diagnostics | Review state/read model and next-round guidance visibility |
| Docs | Runtime orchestration, config, skills, notification cursor primitive | Review gate policy, reviewer skill, channels-as-routing boundary |

## Config Lifecycle

Track 1 adds `[task.orchestration]` defaults defined in the orchestration child spec, including summary/context budgets, spawn-failure limits, scheduler health thresholds, max-runtime defaults, and task execution profile defaults:

```toml
[task.orchestration.profile]
default_coordinator_mode = "inherit"
default_worker_mode = "inherit"
default_sandbox_mode = "inherit"
allow_task_provider_override = true
allow_task_sandbox_none = true
```

Track 2 adds nested review defaults under `[task.orchestration.review]`:

```toml
[task.orchestration.review]
default_policy = "none"
max_rounds = 3
max_review_attempts = 2
timeout = "20m"
rapid_terminal_window = "2m"
rapid_terminal_limit = 3
missing_work_max_items = 20
missing_work_item_max_bytes = 512
reason_max_bytes = 2048
review_text_max_bytes = 12000
next_round_guidance_max_bytes = 4096
failure_policy = "block_task"
```

Config implementation must include defaults, validation, docs, CLI/API redacted inspectability, generated examples, and tests. Task-level execution profiles and review policy may override the defaults within the same bounds.

## Implementation Steps

1. Preserve the current reviewed orchestration design as [`_techspec_orchestration.md`](_techspec_orchestration.md).
2. Implement orchestration hardening migrations and task-service transition changes.
3. Implement `TaskExecutionProfile` migrations, store models, task-service validation, profile defaults, and profile CRUD surfaces.
4. Apply worker/session/sandbox profile resolution in daemon/session start and claim eligibility.
5. Implement context bundle and cursor-seeded SSE.
6. Implement bundled `agh-orchestrator` and `agh-task-worker`, including guided profile instructions.
7. Implement `internal/notifications` and the bridge terminal notifier.
8. Run orchestration-specific tests and peer review closure.
9. Implement review-gate migrations and task-service review interfaces, including `task_run_reviews` and all `task_runs` review trigger/continuation columns in the same numbered migration, review-request follow-up transaction, reviewer-session binding, and `RecordRunReview` atomicity.
10. Apply `TaskExecutionProfile.Review` and `ParticipantPolicy` enforcement to reviewer routing.
11. Implement reviewer routing, typed `ReviewRouter` wake callback, `agh-task-reviewer`, bundled-skill `requires_review_request` support, and native `submit_run_review`.
12. Implement review APIs, UDS, CLI, hooks, generated contracts, web/docs impact.
13. Run review-gate-specific peer review.
14. Run aggregate QA covering execution profile selection -> execution -> review rejection -> continuation -> approval -> bridge notification.
15. Generate `_tasks.md` from this aggregate plus the relevant child specs.

Implementation tasks must state whether they implement:

- aggregate + orchestration child only;
- aggregate + review-gate child only;
- both child specs together.

## Test Strategy

The final implementation must pass `make verify`. Focused test layers:

- Fresh-DB and migrated-DB tests for every migration.
- Task-service transition tests for claim, start, complete, fail, release, cancel, synthetic terminal runs, `current_run_id`, and review continuation.
- Boundary tests proving scheduler/coordinator/channel/skill/notification code cannot mutate ownership or terminal/review state directly.
- Task execution profile tests covering typed persistence, profile precedence, continuation-run precedence, guided coordinator mode, worker agent/provider/model selection, claim filtering, participant policy enforcement/non-authority, sandbox `inherit|none|ref`, and config gates for provider/sandbox overrides.
- Contract/codegen tests for HTTP/UDS/OpenAPI/generated TypeScript parity.
- Task SSE replay tests covering `latest_event_seq`, `Last-Event-ID` precedence, review events, and reconnect behavior.
- Notification cursor tests covering monotonic advance, reset, idempotent replay, duplicate external delivery, deferred review-gated run terminal events, superseded run terminal events, and bridge terminal notification replay after accepted final review.
- Review-gate tests covering approved/rejected/blocked/error/timeout/invalid output, reviewer-session binding, delivery-id idempotency, verdict-plus-continuation atomicity, idempotent review-request uniqueness, `review_required` clearing with `review_request_id`, retry attempt row creation, max rounds, max reviewer attempts, rapid-terminal circuit opening, `parent_review_id` lineage, monotonic `tasks.review_round`, and continuation context injection.
- Real-scenario QA covering a task-specific profile selecting a worker/sandbox shape, a coordinator spawning that worker, a reviewer rejecting incomplete work, a continuation worker applying `missing_work`, reviewer approval, and bridge terminal notification delivery.

## Peer Review Plan

The first two peer-review rounds reviewed the orchestration hardening design and are inherited by [`_techspec_orchestration.md`](_techspec_orchestration.md). The new review-gate child spec and the task execution profile additions introduce state, policy, native tools, hooks, profile selection, session-start behavior, and continuation semantics; they require focused peer review before task generation.

After review-gate findings are incorporated, run a final aggregate peer review focused on:

- cross-spec authority consistency;
- implementation ordering;
- API/config/hook parity;
- absence of channel-owned or notification-owned workflow state;
- absence of profile-owned task authority or sandbox bypass;
- completeness of test and docs impact.

## Architecture Decision Records

- [ADR-001: Orchestration Hardening Extends the Existing Autonomy Substrate](adrs/adr-001-orchestration-hardening-extends-existing-autonomy.md)
- [ADR-002: Use Queryable Task-Owned State for Orchestration Hardening](adrs/adr-002-queryable-orchestration-state.md)
- [ADR-003: Introduce Shared Durable Notification Cursors](adrs/adr-003-shared-durable-notification-cursors.md)
- [ADR-004: Add Minimal Explicit Task Orchestration Config](adrs/adr-004-minimal-task-orchestration-config.md)
- [ADR-005: Keep `tasks.current_run_id` as a Denormalized Read Projection](adrs/adr-005-current-run-id-denormalized-projection.md)
- [ADR-006: Bundled Orchestration Skills Are Instructional, Not Authority](adrs/adr-006-bundled-orchestration-skills-are-instructional.md)
- [ADR-007: Review Gate Is a Post-Terminal Continuation Loop](adrs/adr-007-review-gate-post-terminal-continuation-loop.md)
- [ADR-008: Reviewer Routing Uses Channels Without Channel Authority](adrs/adr-008-review-routing-uses-channels-without-channel-authority.md)
- [ADR-009: Review Verdicts and Continuation Guidance Are Typed Task State](adrs/adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md)
- [ADR-010: Task Execution Profiles Are Typed Task-Owned Overlays](adrs/adr-010-task-execution-profiles-are-typed-overlays.md)
