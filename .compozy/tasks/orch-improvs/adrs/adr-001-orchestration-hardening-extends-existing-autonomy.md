# ADR-001: Orchestration Hardening Extends the Existing Autonomy Substrate

## Status

Accepted

## Date

2026-05-05

## Context

The `orch-improvs` analysis borrows useful orchestration patterns from Hermes, but AGH already has a substantial autonomy substrate:

- `task.Service` and `task_runs` are the canonical task execution authority.
- Task-run claim, heartbeat, release, completion, failure, and recovery are token-fenced and service-owned.
- Agent-facing task mutation is session-bound through active lease lookup; public surfaces expose claim token hashes, not raw claim tokens.
- The mechanical scheduler sweeps expired leases, selects idle sessions, and emits wake prompts, but it never claims work.
- The coordinator runtime is daemon-managed, uses `session.SessionTypeCoordinator`, safe spawn lineage, tool policy, and a prompt overlay.
- Typed task hooks already expose task lifecycle extension points.
- `/agent/context` already provides bounded situation context for agents.
- Task streams already have replay cursor semantics through `after_sequence` and `Last-Event-ID`.
- Workspace defaults already define agent and sandbox defaults, while task runs already carry coordination channel and capability fields.

Archived prior art reinforces these boundaries:

- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/_techspec.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-003.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-004.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-005.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/adrs/adr-012.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/_techspec.md`

The new TechSpec must therefore describe hardening and enrichment of the existing system, not a replacement autonomy subsystem. The later review-gate child spec follows the same posture: it adds task-owned review state and continuation runs without creating a second workflow engine or channel-owned authority.

## Decision

Implement `orch-improvs` as orchestration hardening over existing AGH authorities.

The implementation design must preserve these invariants:

- `task_runs` remains the only durable execution queue.
- `task.Service` remains the sole authority for task-run ownership, lease mutation, terminal state, handoff summary persistence, spawn-failure counters, runtime watchdog transitions, review request creation, review verdict persistence, review circuit state, and review-driven continuation run creation.
- The scheduler remains mechanical: recover expired leases, observe health, detect max-runtime expiry, select idle sessions, and wake. It must not claim task runs, own assignments, stop sessions directly, or write terminal state.
- Max-runtime enforcement uses an actor split: scheduler observes expiry and sends a typed request; `task.Service` requests managed stop and writes the terminal failure transition.
- The coordinator remains daemon-managed orchestration behavior. It can decide when to spawn or guide workers, but it does not become task ownership authority.
- Task execution profiles are typed task-owned overlays for coordinator guidance, worker selection, review selection, participant policy, and sandbox selection. They do not become task authority.
- Coordinator profile mode is limited to `inherit` and `guided` in MVP. `guided` applies task-specific policy to the existing daemon-managed workspace coordinator; it does not create a dedicated per-task coordinator.
- Coordination channels remain conversation, handoff, blocker, result, and review-routing transport. Channel messages may carry handoff, blocker, result, or review-request content, but they do not define task ownership, terminal state, or review verdicts.
- Agent-facing mutation must continue to use shared `BaseHandlers`, UDS/HTTP parity, session-bound lease lookup, and raw claim-token redaction.
- Task-level terminal complete/fail endpoints are operator-only and reject active token-fenced runs. Agent/session actors must use session-bound run-level terminal paths.
- Extensions must use typed task hooks or daemon composition-root observers, not a new generic orchestration event bus.
- `internal/notifications` is a durable cursor primitive only. The first MVP consumer is bridge-delivered terminal task notifications owned by `internal/bridges`, and replay authority remains durable `task_events.event_seq`.
- Review gate is post-terminal in MVP. It does not add `pending_review` to the execution run lifecycle.

## Consequences

### Positive

- Reuses the substrate that was already designed and partially implemented for autonomous AGH.
- Avoids duplicating queue semantics or creating a second source of truth.
- Keeps manual and autonomous task control on the same task/run model.
- Preserves existing security boundaries around session-bound mutation and claim-token secrecy.
- Gives implementation tasks clear package boundaries: `internal/task`, `internal/store/globaldb`, `internal/api/core`, `internal/api/{httpapi,udsapi}`, `internal/scheduler`, `internal/coordinator`, `internal/situation`, and `web/src/systems/tasks`.

### Negative

- Some Hermes ideas must be translated into AGH vocabulary instead of copied directly.
- The TechSpec must explicitly reject attractive shortcuts such as channel-owned status or dispatcher-owned assignments.
- Coordination behavior remains split across runtime authority and instructional guidance, which requires careful tests.

### Risks

- A future implementation could accidentally treat a read projection, channel message, or skill instruction as authority.
- Adding state such as `current_run_id` could be misused by scheduler/coordinator code unless the TechSpec and tasks make the invariant explicit.
- Review routing through channels could be misread as review authority unless the review verdict path is persisted through `task.Service`.
- Task execution profiles could be misread as permission or ownership authority unless implementation tasks preserve the task-service/session/tool-policy boundaries.

## Rejected Alternatives

### New orchestration queue

Rejected because it duplicates `task_runs`, violates the archived autonomy design, and creates competing ownership semantics.

### Scheduler-owned claim and assignment

Rejected because scheduler scope is intentionally limited to recovery and wake notification. Sessions must continue to claim through task APIs.

### Channel-message-owned terminal state

Rejected because channels are operational coordination surfaces, not durable task authority.

### Channel-message-owned review verdicts

Rejected because review verdicts drive continuation runs and must be persisted as task-owned typed state.

### Prompt-only orchestration safety

Rejected because ownership and permissions must be enforced by runtime services, tool policy, session-bound lease lookup, claim-token secrecy, and spawn lineage.

### Dedicated coordinator per task in MVP

Rejected because the current coordinator runtime is daemon-managed and workspace-scoped. Per-task coordinator lifecycles need a separate design for concurrency, budgets, claims, and failure recovery.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-dispatcher.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-cli-tools.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_task-execution-profile.md`
- `.compozy/tasks/orch-improvs/_techspec_review_gate.md`
- `.compozy/tasks/orch-improvs/adrs/adr-010-task-execution-profiles-are-typed-overlays.md`
- `.compozy/tasks/_archived/1777918109821-eb921583-autonomous/_techspec.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/_techspec.md`
