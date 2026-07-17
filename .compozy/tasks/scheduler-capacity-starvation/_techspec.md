# Technical Specification: Scheduler Capacity-Aware Starvation

## Executive Summary

Marketplace Task 11 real-scenario QA proved that the mechanical scheduler treats a compatible owner-pool session that is busy with earlier serial work as if no compatible worker exists. The later run advances through the starvation ladder and can reach `needs_attention` before its worker becomes free. This specification replaces that conflated eligibility check with an explicit capacity disposition and aligns public starvation pressure with durable convergence episodes.

No standalone PRD or user-story catalog exists for this incident. The approved product input is the Marketplace Task 11 QA evidence plus the user's 2026-07-15 decision to preserve serial execution and reject elastic parallelism. **MVP boundary:** implementation steps 1–7 in this specification deliver capacity-aware serial waiting, durable/public pressure alignment, documentation, focused verification, and one fresh Northstar QA pass. Elastic pools, hybrid expansion, a public capacity-waiting counter, and scheduler auto-scaling are post-MVP and out of scope.

## Goals

- Distinguish compatible-but-busy capacity from true worker absence.
- Keep serial owner-pool backlog queued without spawn, starvation event, or `needs_attention` escalation.
- Preserve convergence for true absence and for compatible idle sessions that repeatedly do not claim.
- Fail closed when capability projection cannot prove either compatibility or absence.
- Preserve `task.Service.ClaimNextRun` as the sole claim authority and one active lease per session.
- Make `starved_run_count` describe active durable convergence episodes rather than raw queued age.
- Shrink the existing oversized scheduler owners while introducing the new behavior.

## Non-Goals

- Starting multiple task-role sessions for the same serial pool.
- Adding pool size, saturation duration, cost, or auto-scaling configuration.
- Adding or changing HTTP routes, CLI verbs, UDS routes, JSON fields, native tools, or Web controls.
- Changing task priority, queue order, claim selection, lease fencing, session supervision, or task-role deduplication.
- Resetting an existing convergence budget merely because compatible capacity becomes temporarily busy.

## System Architecture

### Component Overview

| Component | Responsibility | Change |
| --- | --- | --- |
| `internal/scheduler/capacity.go` | Classify structural match and momentary capacity for one run | New file; owns disposition types and classifier |
| `internal/scheduler/selection.go` | Order queued runs/sessions and select advisory wake targets | New file; moves selection out of oversized `scheduler.go` |
| `internal/scheduler/wake_dispatch.go` | Dispatch wakes, cooldown state, and wake counters | New file; moves dispatch/state helpers out of oversized `scheduler.go` |
| `internal/scheduler/scheduler.go` | Constructor, lifecycle, rebuild, and cycle orchestration | Modified and reduced below 500 lines |
| `internal/scheduler/starvation.go` | Advance or freeze durable convergence episodes | Modified; receives only true convergence candidates |
| `internal/daemon/scheduler_session_source.go` | Project live session identity, state, prompt/lease availability, and capability certainty | New file; capability errors become indeterminate snapshots |
| `internal/daemon/scheduler_waker.go` | Deliver advisory task wake prompts | New file; extracted from oversized `scheduler_runtime.go` |
| `internal/daemon/scheduler_runtime.go` | Composition-root wiring and escalation adapter | Modified and reduced below 500 lines |
| `internal/store/globaldb/global_db_task_scheduler_status.go` | Count queued pressure and active convergence episodes | New file; moves scheduler status reads from pause owner |
| `internal/task/scheduler_status.go` | Compose public pause and queue-pressure status | Modified to call active-episode count |

Data flow:

1. `schedulerSessionSource` projects every live session. Capability projection failure preserves the session with `unknown` capability state instead of deleting it from the snapshot.
2. `selectWakeTargets` asks the capacity classifier to separate structural compatibility from current availability.
3. `available` runs receive advisory wakes. `capacity_waiting` and `indeterminate` runs remain queued and freeze convergence. `unmatched` or repeatedly ignored `available` runs become convergence candidates after the effective age threshold.
4. `runConvergence` remains the only scheduler path that persists `task_run_starvation`, requests a worker, emits `task.run_starved`, or asks the task service for `needs_attention`.
5. Scheduler status counts active starvation side-table rows joined to claimable queued runs; the public payload shape remains unchanged.

## Architectural Boundaries

- `internal/scheduler` may import `internal/task` domain types but never calls `ClaimNextRun`, writes claim ownership, or starts sessions directly.
- `internal/task` remains the sole owner of claim, lease, terminal transition, recovery, pause, and `needs_attention` mutations. It must not import `internal/scheduler`.
- `internal/daemon` remains the sole composition root. It adapts session/runtime state into scheduler snapshots and implements escalation side effects through existing task-role and task-service authorities.
- `internal/store/globaldb` persists the existing `task_run_starvation` side table and queue-pressure reads. It does not infer live session capacity.
- HTTP and UDS continue to share `internal/api/core` scheduler handlers and the existing `task.Manager` interface. No transport-specific scheduler semantics are allowed.
- `web/` continues to consume the generated `SchedulerStatusPayload`; no manual DTO or UI-specific capacity calculation is introduced.
- No new internal package or import edge is introduced. New files remain inside existing packages and every production file stays at or below 500 lines after the split.

## Implementation Design

### Core Interfaces

The scheduler receives explicit capability certainty in each session snapshot:

```go
type CapabilityState string

const (
    CapabilityStateKnown   CapabilityState = "known"
    CapabilityStateUnknown CapabilityState = "unknown"
)

type SessionSource interface {
    Sessions(context.Context) ([]SessionSnapshot, error)
}
```

The classifier returns one mutually exclusive disposition:

```go
type CapacityKind string

const (
    CapacityAvailable     CapacityKind = "available"
    CapacityWaiting       CapacityKind = "capacity_waiting"
    CapacityUnmatched     CapacityKind = "unmatched"
    CapacityIndeterminate CapacityKind = "indeterminate"
)
```

```go
type CapacityDisposition struct {
    Kind      CapacityKind
    Available []SessionSnapshot
    Reason    string
}

func classifyRunCapacity(
    work *RunSnapshot,
    sessions []SessionSnapshot,
    occupied map[string]struct{},
) CapacityDisposition
```

Public status depends on active durable convergence episodes, not an age-only query:

```go
type schedulerControlStore interface {
    GetSchedulerPause(context.Context) (SchedulerPauseState, error)
    CountActiveTaskRunClaims(context.Context) (int, error)
    CountQueuedTaskRuns(context.Context, bool) (int, error)
    CountPausedTasks(context.Context) (int, error)
    CountEscalatingQueuedTaskRuns(context.Context) (int, error)
    CountNeedsAttentionTaskRuns(context.Context) (int, error)
}
```

Errors follow existing conventions:

- A session catalog/list failure aborts the cycle because capacity cannot be classified safely.
- A per-session capability projection failure returns an `indeterminate` snapshot and logs the contextual error; it does not remove the session or advance convergence.
- Store and escalation errors are wrapped with `%w`, joined at the cycle boundary, and never discarded.
- Claim races remain task-service concerns; selection re-reads queued status before side effects as today.

### Capacity Classification

Structural matching evaluates these dimensions before availability:

1. The session has a stable non-empty ID and a live reusable state (`starting` or `active`).
2. Task scope and `workspace_id` match exactly; global tasks match only sessions with an empty workspace ID.
3. `coordination_channel_id` matches the session channel when the run declares one.
4. Pool ownership matches `AgentName`; session ownership matches session ID; ownerless tasks accept either.
5. Required capabilities are covered when capability state is `known`.

Availability then evaluates prompting, active task-run lease ownership, and same-cycle reservation. The result rules are:

| Structural result | Availability result | Disposition | Convergence |
| --- | --- | --- | --- |
| At least one known compatible session | At least one idle and unreserved | `available` | Wake; advance only if it remains unclaimed past threshold |
| At least one known compatible session | All starting, prompting, leased, or reserved | `capacity_waiting` | Freeze |
| No known compatible session | No unknown candidate | `unmatched` | Advance after threshold |
| No known compatible session | At least one identity/scope/owner/channel match with unknown capabilities | `indeterminate` | Freeze and diagnose |

When both busy and idle compatible sessions exist, the run is `available` and only idle sessions are wake candidates. A session reserved for an earlier run in the same cycle counts as capacity for later runs, preventing the later runs from being misreported as unmatched.

### Data Models

No SQLite schema, migration, frontmatter, config key, API field, or generated type is added.

In-memory field rationale:

| Field | Shape | Purpose |
| --- | --- | --- |
| `SessionSnapshot.CapabilityState` | `CapabilityState` enum | Distinguishes authoritative empty capabilities from projection failure |
| `CapacityDisposition.Kind` | `CapacityKind` enum | Makes available/waiting/unmatched/indeterminate exhaustive |
| `CapacityDisposition.Available` | `[]SessionSnapshot` | Supplies only idle wake candidates without recomputing structural match |
| `CapacityDisposition.Reason` | internal string enum/value | Powers deterministic logs and tests without public contract growth |
| `selectionResult.capacityWaiting` | `[]RunSnapshot` | Reports cycle-local held runs without sending them to convergence |
| `CycleResult.CapacityWaitingRuns` | `int` | Internal operational counter for a completed scheduler cycle |
| `CycleResult.CapacityWaitingRunIDs` | `[]string` | Internal correlated diagnostics; never an API payload |
| `Stats.CapacityWaitingRuns` | `int` | Accumulated in-process scheduler observation |

Existing durable field rationale:

| Field | Shape | Purpose after this change |
| --- | --- | --- |
| `task_run_starvation.run_id` | `TEXT PRIMARY KEY` | Identifies one active convergence episode for a queued run |
| `wake_count` | non-negative integer | Preserves bounded escalation progress across cycles and restart |
| `first_starved_at` | timestamp | Records when true convergence began, not when capacity merely became busy |
| `spawn_requested_at` | nullable timestamp | Coalesces one spawn request per episode |
| `starved_event_at` | nullable timestamp | Enforces exactly-once `task.run_starved` emission per episode |
| `updated_at` | timestamp | Records the last durable episode mutation |

#### Side-table vs JSON Decision

The existing typed `task_run_starvation` side table remains the durable convergence model. Capacity disposition stays rebuildable and in memory because it derives from live sessions each cycle. No capacity or starvation state is stored in `metadata_json`: matchable/countable durable state belongs in typed rows, while volatile availability must not be persisted as stale truth. No migration is required.

### Public Interfaces and Types

No public shape changes.

| Surface | Existing contract | Semantic correction |
| --- | --- | --- |
| HTTP | `GET /api/scheduler` → `SchedulerStatusResponse` | `starved_run_count` counts queued runs with active convergence rows |
| UDS | `GET /api/scheduler` through the shared core handler | Same semantics and response as HTTP |
| CLI | `agh scheduler status -o json` | Same field names; serial capacity wait remains in `queued_run_count`, not `starved_run_count` |
| CLI | `agh scheduler backlog -o json` | Continues to list queued runs in priority order, including capacity-held runs |
| Web | `useSchedulerStatus` and Tasks scheduler panel | Consumes corrected counts; no component or generated type change |
| Events | `task.run_starved`, `task.run_needs_attention` | Fire only for true convergence, never for capacity wait/indeterminate state |

No new deterministic public error is introduced. Existing task recovery remains valid only for genuine `needs_attention` runs.

### Safety Invariants

1. `task.Service.ClaimNextRun` remains the only claim authority; the scheduler never claims or assigns ownership.
2. A session holds at most one active task-run lease; this change does not relax the store fence.
3. Structural compatibility applies scope, exact `workspace_id`, coordination channel, owner, and required capabilities before availability can hold a run.
4. A compatible starting, prompting, leased, or same-cycle-reserved session yields `capacity_waiting`; that run receives no spawn, starvation event, or `needs_attention` mutation.
5. An unknown capability projection yields `indeterminate`; uncertainty never proves absence and never advances convergence.
6. A compatible idle session remains wakeable. If it stays idle and the run stays queued past `min_queued_age`, convergence advances as an uninterested-worker failure.
7. True absence yields `unmatched` and continues through the existing bounded ladder after the effective age threshold.
8. Capacity wait freezes an existing `task_run_starvation` row without incrementing or deleting it. If capacity later disappears or becomes idle/unresponsive, the episode resumes from its durable budget.
9. Global pause and post-resume grace still use `max(run.queued_at, scheduler_pause.updated_at)` before convergence can begin or resume.
10. `starved_run_count` counts active convergence rows joined to claimable queued runs and never counts age-only serial backlog.
11. Terminal, claimed, starting, running, paused-task, and deleted runs cannot remain counted as public starvation pressure.
12. Policy A never starts extra capacity merely because compatible capacity is occupied; session supervision and lease recovery own slow, hung, or dead active workers.

### Delete Targets

- Delete the conflated `isEligibleSession` implementation; replace it with structural matching plus availability classification and retain no wrapper or alias.
- Delete raw-age-only `CountStarvedQueuedTaskRuns(ctx, now, age)` semantics and signature; replace it with `CountEscalatingQueuedTaskRuns(ctx)` across store, task service, fakes, and tests.
- Delete the daemon behavior that drops a live session when capability projection fails; preserve an indeterminate snapshot instead.
- Delete selection logic that adds every aged run to `convergenceCandidates` before capacity disposition is known.
- Delete no compatibility fields, dual counters, fallback config, or legacy branches because the public payload shape is unchanged.

### Hard-Cut Policy

- **No fallback:** capability projection uncertainty is represented explicitly; it never falls back to an empty capability set or worker absence.
- **No compat shim:** the conflated eligibility helper and age-only starvation query are deleted, not wrapped or aliased.
- **No placeholder:** capacity disposition, status semantics, docs, tests, and QA ship together in this change.

## Integration Points

There is no new external service integration. The existing integration boundaries remain:

- Session manager → daemon scheduler session projection.
- Task/global store → queued and active-run snapshots plus starvation side table.
- Scheduler escalation adapter → task service and task-role runtime.
- Task service → shared HTTP/UDS core handler → CLI/Web consumers.

The real-scenario QA provider remains Claude Sonnet 5/max under the existing provider-home policy; it validates behavior but is not part of the production design.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
| --- | --- | --- | --- |
| Scheduler selection | Modified | High: wrong classification can suppress true recovery or park valid backlog | Exhaustive disposition tests and integration cycles |
| Scheduler lifecycle files | Refactored | Medium: oversized owner split can change state/counter behavior | Move behavior without duplication; run lifecycle and race suites |
| Daemon session projection | Modified | High: capability errors currently look like absence | Preserve indeterminate snapshots and structured errors |
| Task-role runtime | Verified | Medium: policy relies on serial session reuse | Extend canonical reuse/drain integration coverage |
| Global task store | Modified | Medium: status count changes from age to durable episode | Real DB tests for queued/paused/terminal joins |
| Task scheduler status | Modified | Medium: public count semantics change without shape change | Preserve pause/resume grace and HTTP/UDS/CLI parity |
| Web scheduler consumers | No source change | Low: generated shape and component contract are unchanged | Run existing focused component/typecheck gates if backend parity tests reveal drift |
| Site and official skill | Modified | Medium: operators need the corrected meaning of starvation | Update named docs and bundled skill references |
| QA tracker | Modified | High: user-visible recovery behavior changes | Add/reset an untested serial-backlog scenario and record the capacity bug |

## Extensibility Integration Plan

- **Extension manifests and Host API:** no impact — checked `internal/extension`, extension protocol methods, and manifest schemas; capacity classification is daemon-internal and exposes no extension mutation.
- **Hooks:** semantic impact only — existing `task.run_starved` and `task.run_needs_attention` events no longer fire for compatible occupied capacity. No hook ID or payload field changes.
- **Skills/capabilities:** update `skills/agh/references/tasks-and-orchestration.md` and `skills/agh/references/capabilities-and-bundles.md`; capability matching IDs and registries are unchanged.
- **Tools/resources/bundles/registries:** no impact — checked native task tools, resource projection, bundle activation, and registries; scheduler controls are not native tools and no descriptor/digest changes.
- **Bridge SDKs and MCP sidecars:** no impact — no delivery, bridge, MCP, or sidecar lifecycle participates in task capacity classification.
- **Protocol docs:** no AGH Network wire impact — task-run ownership and scheduler events remain local runtime contracts.

## Agent Manageability Plan

- `agh scheduler status -o json` remains the structured status path; `starved_run_count` now means active convergence episodes.
- `agh scheduler backlog -o json` remains the structured inspection path for all queued work, including serial capacity wait.
- `GET /api/scheduler` and `GET /api/scheduler/backlog` retain HTTP/UDS parity through shared handlers.
- `agh task run recover` remains the repair path only after a genuine `needs_attention` transition; capacity-held runs require no manual recovery.
- No new CLI verb, native `agh__*` tool, response field, or deterministic error is necessary. Agents can distinguish healthy serialization from failure by combining queued count/backlog, active claims, and corrected starvation pressure.

## Config Lifecycle

- Existing `[autonomy.scheduler]` keys remain: `fan_out_after`, `spawn_after`, `event_after`, `needs_attention_after`, and `min_queued_age`.
- Defaults, TOML structs, merge/overlay behavior, validation, live/restart lifecycle, and examples do not change.
- Thresholds apply only after a run is truly unmatched or an available compatible session remains uninterested. They do not govern how long compatible occupied capacity may work.
- No new pool-size, capacity-wait, elastic-spawn, or cost key is added.
- Update the official skill explanation and the runtime autonomy page; generated CLI docs do not change because Cobra verbs/flags/help are unchanged.

## Web/Docs Impact

### `web/`

- No source change — checked `web/src/systems/scheduler/adapters/scheduler-api.ts`, `types.ts`, `hooks/use-scheduler.ts`, scheduler fixtures, and `scheduler-controls-panel.tsx`. Payload fields and UI composition remain correct.
- No generated TypeScript change — `internal/api/contract` and OpenAPI shapes are unchanged.
- Existing Web behavior becomes more truthful because the same `starved_run_count` field stops including serial capacity wait.

### `packages/site`

- Update `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx` with capacity-wait versus starvation semantics and the corrected status meaning.
- Generated scheduler CLI reference pages remain unchanged; no `make cli-docs` output is expected because no command contract changes.
- No protocol illustration or Remotion asset changes.

### QA Impact

- Add a content-addressed `docs/qa/bugs/BUG-20260715-scheduler-busy-pool-starvation.md` record for the distinct incident.
- Add a content-addressed untested scenario proving multiple runs assigned to one pool wait serially without false starvation, or reset the exact existing scenario if the tracker already owns that invariant.
- Keep `TA-047` untested until the fresh Northstar pass proves pause/resume and serial capacity together.

## Testing Approach

Concrete cases are canonical in [`_tests.md`](_tests.md).

- **Unit:** existing Go suites own capacity classification, selection ordering, durable budget freeze/resume, daemon projection certainty, and public status composition. Fakes remain at session/store/escalation I/O boundaries.
- **Integration:** real global DB plus scheduler/task service wiring proves no side effects across 10+ busy cycles, correct claim after capacity release, restart persistence, and workspace/channel isolation.
- **E2E:** the fresh Northstar playbook uses one operator kickoff, real provider sessions, public CLI/API/Web observation, strict evidence audit, and mandatory teardown.
- **Iteration commands:** `CGO_ENABLED=1 go test -race ./internal/scheduler/...`, focused daemon/task/globaldb package tests, direct scoped golangci, and `git diff --check`.
- **Completion command:** the Marketplace program runs one final `make verify` only after Task 11, deep review, and all remediation are complete.

## Test Plan

- **Capacity classification:** extend `TestRunOnceEscalatesStarvedRuns` with known compatible idle, busy, prompting, starting, reserved, unmatched, and unknown-capability cases; run `rtk env CGO_ENABLED=1 go test -race ./internal/scheduler/... -count=1`.
- **Durable convergence:** prove a busy compatible pool survives at least 12 cycles without spawn/event/attention, then prove budget resume after capacity disappearance in the scheduler integration suite.
- **Session projection:** prove known-empty capabilities differ from projection failure in `scheduler_runtime_test.go`; run the focused daemon scheduler/task-role tests under `-race`.
- **Public status:** prove only active `task_run_starvation` episodes count across task/globaldb owners and shared HTTP/UDS/CLI surfaces.
- **Isolation:** prove sessions from another workspace, channel, owner, or capability set cannot suppress convergence.
- **Real scenario:** bootstrap a fresh Northstar lab, use one confirmed kickoff, capture CLI/API/Web/runtime/provider evidence, run the strict audit, and cite `teardown.json` with `clean=true`.
- **Static/final gates:** run direct scoped golangci and `rtk git diff --check` during iteration; reserve the single full `rtk make verify` for final Marketplace completion.

## Development Sequencing

### Build Order

1. Add RED cases to the canonical scheduler, daemon, task, and global-store suites named in `_tests.md`.
2. Split scheduler selection/wake responsibilities and daemon session-source/waker responsibilities without behavior changes.
3. Add explicit capability certainty and capacity disposition; route only true candidates into convergence.
4. Hard-cut public starvation counting to active durable episodes and preserve pause/resume grace.
5. Run focused race/lint owners and correct any production defect exposed by the tests.
6. Update official skill, runtime docs, QA bug, and tracker scenario.
7. Build the daemon and run one fresh isolated Northstar QA pass with clean teardown.

### Technical Dependencies

- Existing task-run lease exclusivity and task-role deduplication must remain green.
- Existing `task_run_starvation` schema and generated sqlc queries remain the durable episode authority; no migration/codegen dependency is introduced unless implementation proves a query must move into sqlc.
- The Northstar QA pass depends on a fresh `agh-qa-bootstrap` manifest and provider availability under the declared home policy.

## Monitoring and Observability

- Emit `scheduler.capacity_waiting` once per cycle when held runs exist, with `run_ids`, `count`, and normalized reason values; never include claim tokens or secrets.
- Preserve `scheduler.session_context.error` with `session_id` when capability projection is indeterminate.
- Add internal `CycleResult.CapacityWaitingRuns`, `CapacityWaitingRunIDs`, and cumulative `Stats.CapacityWaitingRuns` for tests and daemon logs; do not add public JSON fields.
- Keep `scheduler.wake.no_match` for true unmatched runs only.
- Keep `scheduler.wake.starved`, `task.run_starved`, spawn requests, and `task.run_needs_attention` exclusive to true convergence.
- Existing active-claim, queued-run, starvation, and needs-attention counts remain the operator health signals. No alert threshold changes.

## Technical Considerations

### Key Decisions

- **Decision:** hold serial backlog while compatible capacity is occupied. **Rationale:** preserves the current one-lease-per-session and one-task-role-session model. **Trade-off:** slow workers can create long queues. **Alternatives rejected:** elastic and hybrid pools add cost and concurrency the user explicitly rejected.
- **Decision:** fail capability uncertainty closed. **Rationale:** missing context cannot prove absence. **Trade-off:** persistent projection errors can hold a run until repaired. **Alternative rejected:** dropping the session repeats the false-absence bug.
- **Decision:** derive public starvation from durable episode rows. **Rationale:** it aligns status with actual convergence without injecting live session dependencies into `internal/task`. **Trade-off:** an existing frozen episode remains visible after convergence began. **Alternative rejected:** raw age conflates healthy serialization with failure.
- **Decision:** split oversized owners before adding logic. **Rationale:** the project hard cap and one-responsibility rule are already violated by the legacy files. **Trade-off:** larger mechanical diff. **Alternative rejected:** appending another helper would deepen the architecture failure.

### Known Risks

- An active session may be prompting for unrelated operator work. Policy A intentionally treats it as occupied capacity; session supervision and operator controls own that delay.
- A stale active-run snapshot may briefly hold a run for one cycle. The next cycle rebuilds state, and lease recovery remains authoritative.
- Same-cycle reservation must be deterministic under priority ordering; ordering tests pin the higher-priority/older run selection, not implementation-specific slice layout.
- Public status query changes must retain paused-task ancestor exclusion. Real DB tests cover the recursive exclusion rather than duplicating SQL in test fakes.

## Architecture Decision Records

- [ADR-001: Hold Serial Backlog While Compatible Capacity Is Busy](adrs/adr-001.md) — preserve serial execution and freeze convergence while compatible capacity is occupied or indeterminate.

## Assumptions and Defaults

- The approved policy is serial waiting; no parallelism is inferred or introduced.
- One active task-run lease per session remains the default and enforced contract.
- Existing scheduler intervals and starvation thresholds remain unchanged.
- Capability certainty is recomputed every scheduler cycle and is not persisted.
- The public payload shape is stable; only `starved_run_count` semantics become accurate.
- No schema migration, OpenAPI regeneration, generated TypeScript change, native-tool update, or Web source change is expected.
- If implementation evidence contradicts any assumption, stop and amend this TechSpec before expanding scope.
