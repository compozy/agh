# ADR-010: Task Execution Profiles Are Typed Task-Owned Overlays

## Status

Accepted

## Date

2026-05-05

## Context

Operators and agents need task-specific control over the runtime shape of orchestration:

- which coordinator guidance applies to a task;
- which worker agents, providers, models, peers, channels, and capabilities may participate;
- which reviewer agents, providers, models, peers, channels, and capabilities should handle review;
- whether a task should inherit the workspace sandbox, run without a sandbox, or use a specific sandbox reference.

AGH already has related primitives, but they are spread across workspace defaults and run/session internals:

- coordinator configuration is global/workspace-level and the daemon runtime enforces a workspace-scoped coordinator model;
- workspaces have `DefaultAgent` and `SandboxRef`;
- worker task sessions currently fall back to workspace defaults because task session start does not pass task-level `AgentName` or `Provider`;
- task runs already have coordination channel and capability fields;
- review-gate routing already supports peer/channel/capability selectors;
- `ClaimCriteria.AgentName` exists, but the global DB claim selector does not yet enforce it.

Putting this policy in `metadata_json` would make it hard to validate, query, index, expose through generated contracts, or apply consistently at session start.

## Decision

Add `TaskExecutionProfile` as typed task-owned state. The profile is an overlay that feeds coordinator guidance, worker session selection, review routing, participant policy, and sandbox selection.

The MVP profile contains:

- `CoordinatorProfile`: task-specific coordinator policy with `mode = "inherit"` or `mode = "guided"`.
- `WorkerProfile`: worker agent/provider/model hints and worker eligibility selectors.
- `ReviewProfile`: reviewer agent/provider/model hints and review selector policy.
- `ParticipantPolicy`: allowed/preferred channels, peers, agents, and capabilities for coordination.
- `SandboxPolicy`: `mode = "inherit"`, `mode = "none"`, or `mode = "ref"` plus optional `sandbox_ref`.

`CoordinatorProfile.mode = "guided"` keeps the existing daemon-managed workspace coordinator. It injects task-specific policy, context, and guidance into the coordinator path, but it does not create a dedicated coordinator session and does not break `max_active_per_workspace`.

`dedicated` coordinator mode is out of MVP. It would require a separate coordinator lifecycle design because simultaneous tasks in one workspace could request different coordinator agents, providers, models, or sandbox needs.

Effective reviewer selection has explicit precedence so `ReviewProfile` convenience fields and nested `ReviewerSelector` fields cannot drift:

1. Config defaults and the task review policy selector provide the baseline.
2. `TaskExecutionProfile.Review.Selector` overlays the baseline.
3. Top-level `ReviewProfile` fields (`agent_name`, `provider`, `model`, `allowed_agent_names`, and `preferred_agent_names`) override same-named nested selector fields.
4. A concrete route request may narrow the effective selector but cannot widen task profile or participant policy.

The profile must be persisted through typed columns and selector side tables, not `metadata_json`. It must be surfaced through create/update/read task DTOs, HTTP/UDS parity, CLI inspection/update commands, generated OpenAPI/TypeScript, docs, and tests.

The profile is not runtime authority:

- task ownership remains in `task_runs` and `task.Service`;
- worker mutation remains session-bound through active lease lookup;
- review verdict authority remains `task.Service.RecordRunReview`;
- participant policy does not grant channel, peer, or agent permission;
- sandbox policy does not bypass tool policy, approval policy, provider authorization, or session authorization;
- coordinator guidance does not create queue, terminal-state, or ownership authority.

`ParticipantPolicy` is enforced as a narrowing policy at concrete runtime call sites:

- task profile validation for explicit channel, peer, agent, and capability selectors;
- coordinator routing for coordination and review channels/peers;
- worker claim filtering for agent/capability eligibility;
- safe-spawn and session-start grant narrowing;
- review routing when participant allowed lists are set.

Violations return deterministic task/profile errors. The policy can reject a route, claim, or spawn; it cannot grant permission that the network, bridge, tool policy, review binding, or task ownership layer would otherwise deny.

Continuation runs created by review rejection use the task's current profile at enqueue time. Reviewed-run native coordination/capability fields are copied only when the current task profile leaves the equivalent worker/participant selector empty.

## Consequences

### Positive

- Makes task-specific runtime selection explicit, queryable, testable, and agent-operable.
- Reuses workspace defaults as defaults while allowing task-level overrides.
- Avoids smuggling orchestration policy through opaque metadata.
- Lets the review-gate spec reuse the same profile shape for reviewer selection.
- Preserves the current coordinator singleton while still giving tasks tailored coordination guidance.

### Negative

- Adds migrations, store methods, contract/codegen work, CLI/UDS/HTTP surfaces, docs, and tests.
- Requires claim-query changes before worker `agent_name` selection is real.
- Requires session start plumbing so task profiles feed agent/provider/model/sandbox selection.
- Requires clear docs to avoid confusing participant policy with permissions.
- Requires implementation tests for every enforcement call site because "documented intent only" is insufficient for task generation.

### Risks

- Per-task coordinator hints could be mistaken for per-task coordinator authority. The MVP must keep `guided` as guidance only.
- Sandbox `none` could be overused if not validated and audited. Config must explicitly allow or reject task-level `none`.
- Agent/provider/model overrides could produce unclaimable work if no eligible worker exists. The task-service and coordinator surfaces must expose deterministic errors and health diagnostics.

## Rejected Alternatives

### Store profile in `metadata_json`

Rejected because worker/reviewer/sandbox selection is operational state. It needs validation, indexes, generated contract types, and deterministic runtime application.

### Dedicated coordinator per task in MVP

Rejected because the current runtime assumes daemon-managed workspace coordinator behavior. Dedicated per-task coordinators require a separate lifecycle, concurrency, budget, and ownership design.

### Workspace defaults only

Rejected because AGH tasks need per-task control over worker/reviewer/sandbox shape without mutating workspace-wide defaults.

### Channel membership as participant authority

Rejected because channels are coordination transport. They may help discover or discuss participants, but they do not grant ownership, review verdict authority, or terminal-state permission.

## References

- [`../_techspec.md`](../_techspec.md)
- [`../_techspec_orchestration.md`](../_techspec_orchestration.md)
- [`../_techspec_review_gate.md`](../_techspec_review_gate.md)
- [`../analysis/analysis_task-execution-profile.md`](../analysis/analysis_task-execution-profile.md)
- `internal/config/autonomy.go`
- `internal/workspace/workspace.go`
- `internal/session/sandbox.go`
- `internal/daemon/task_runtime.go`
- `internal/task/lease.go`
- `internal/store/globaldb/global_db_task_claim.go`
