# ADR-006: Bundled Orchestration Skills Are Instructional, Not Authority

## Status

Accepted

## Date

2026-05-05

## Context

AGH already has an under-the-hood coordinator/orchestrator:

- The daemon observes task runs.
- The coordinator runtime decides when a coordinator is needed.
- Coordinator sessions are daemon-managed with `session.SessionTypeCoordinator`.
- `coordinator.PromptOverlay` supplies operational instructions.
- Tool allowlists, lineage, safe spawn, session-bound task mutation, and `task_runs` enforce runtime authority.

The selected scope includes bundled orchestration skills. These skills are useful if they make coordinator and worker behavior more auditable, testable, and reusable. They are dangerous if they are treated as permission boundaries or ownership semantics.

## Decision

Create three bundled orchestration skills:

- `agh-task-worker`: guidance for normal spawned or manual worker sessions that execute task runs.
- `agh-orchestrator`: guidance for daemon-managed coordinator sessions.
- `agh-task-reviewer`: guidance for one-shot reviewer sessions that inspect terminal task runs and submit typed persisted review verdicts.

`agh-orchestrator` must be loaded deterministically by the coordinator runtime during coordinator bootstrap. It must not depend on the coordinator remembering to discover and load the skill manually from the catalog.

The coordinator prompt strategy should treat `agh-orchestrator` as a versioned instruction source. The TechSpec should prefer moving stable operational guidance out of hardcoded `coordinator.PromptOverlay` prose and into the bundled skill, while keeping the runtime-specific bootstrap facts in the prompt overlay.

The deterministic injection contract is:

- `internal/daemon/coordinator_runtime` calls a bundled-skill loader during coordinator bootstrap.
- The loader resolves `agh-orchestrator` from the bundled skills filesystem/registry, not from workspace/user catalog discovery.
- The assembled coordinator prompt places the skill body before `coordinator.PromptOverlay`.
- `coordinator.PromptOverlay` carries only runtime facts, public API hints, and run/channel identifiers; it must not duplicate stable orchestration guidance that belongs in the skill.
- The runtime emits `coordinator.orchestrator_skill_injected` after successful assembly.

The bundled skill frontmatter must mark all three skills as `metadata.agh.instructional_only = true` and use `metadata.agh.always_load` for runtime load triggers; `agh-orchestrator` declares coordinator-only deterministic injection, `agh-task-worker` loads only for task-worker sessions with an active task claim or task-tool loop, and `agh-task-reviewer` loads only for reviewer sessions bound to a persisted review request. The implementation must add loader-side support for the `requires_active_task_claim` and `requires_review_request` predicates if they do not already exist.

Mandatory guardrail:

> Bundled orchestration skills are instructional artifacts only. They do not define runtime authority, permission boundaries, task ownership, review verdict authority, queue semantics, or terminal state. Those remain enforced by `task.Service`, `task_runs`, session-bound lease lookup, review-request binding, tool policy, coordinator runtime, scheduler boundaries, and spawn lineage.

The `agh-task-worker` skill should teach:

- how to inspect `agh me context` and `/agent/context`;
- how to claim or continue task runs through session-scoped task APIs/tools;
- how to heartbeat until terminal state;
- how to complete/fail/release with bounded summaries;
- how to use coordination channels for handoff without treating channels as task state;
- how to avoid raw claim-token leakage;
- how to handle blockers and failures.

The `agh-orchestrator` skill should teach:

- how to read context before coordinating;
- how to apply `CoordinatorProfile.mode = "guided"` as task-specific guidance without treating it as ownership authority;
- when to spawn workers;
- how to respect `WorkerProfile`, `ParticipantPolicy`, and `SandboxPolicy` while preserving task-service/session/tool-policy authority;
- how to request and interpret handoff summaries;
- how to use channels for coordination;
- how to avoid assuming ownership from messages;
- how to react to spawn failure, worker failure, stale leases, and task runtime limits;
- how to request and route review sessions without treating channel discussion as verdict state;
- how to end or transfer work through task-service-owned transitions.

The `agh-task-reviewer` skill should teach:

- how to read the review packet, task objective, run summary, terminal result/error, and prior review history;
- how to respect `TaskExecutionProfile.Review` reviewer-selection input without treating it as verdict authority;
- how to use channels for clarification while preserving `submit_run_review` as the only verdict path;
- how to submit `approved`, `rejected`, `blocked`, `error`, `timeout`, or `invalid_output` honestly;
- how to provide bounded `missing_work` and `next_round_guidance` for rejected reviews;
- how to avoid raw claim-token leakage and prompt-only verdicts.

## Consequences

### Positive

- Makes coordinator behavior more auditable than a large hardcoded prompt.
- Gives manual and spawned workers a consistent operational loop.
- Gives reviewer sessions deterministic guidance without making the skill a verdict authority.
- Keeps safety boundaries in runtime code instead of prose.
- Creates reusable instruction content for docs, tests, and future agent setup flows.

### Negative

- Requires prompt assembly changes for deterministic coordinator skill injection.
- Requires tests proving coordinator bootstrap includes the versioned skill content and reviewer bootstrap includes the reviewer skill content.
- Requires care to avoid duplicate or conflicting guidance between `coordinator.PromptOverlay` and the skill body.

### Risks

- If the skill is only available in the catalog, coordinator behavior may be inconsistent. Deterministic injection is required.
- If the skill text claims authority, future contributors may mistake prompt guidance for runtime enforcement.
- If the worker or reviewer skill exposes raw token concepts, it may undermine claim-token secrecy. It should refer only to session-scoped lease/review actions and claim-token hashes where needed.

## Rejected Alternatives

### No new bundled skills

Rejected because orchestration behavior needs reusable guidance for both managed coordinators and normal workers.

### Catalog-only orchestrator skill

Rejected because the under-the-hood coordinator should not depend on manual skill discovery for core behavior.

### Skill as safety boundary

Rejected because prompt instructions cannot enforce ownership, permissions, or terminal state.

## References

- `.compozy/tasks/orch-improvs/analysis/analysis_hermes-orchestrator-skills.md`
- `.compozy/tasks/orch-improvs/analysis/analysis_task-execution-profile.md`
- `.compozy/tasks/orch-improvs/adrs/adr-010-task-execution-profiles-are-typed-overlays.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/_techspec.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/adrs/adr-001.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/adrs/adr-002.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/adrs/adr-003.md`
- `.compozy/tasks/_archived/20260402-013544-supervisor-orchestration/adrs/adr-004.md`
- `internal/coordinator/coordinator.go`
- `internal/daemon/coordinator_runtime.go`
- `internal/skills/bundled/`
