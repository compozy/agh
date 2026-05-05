# Analysis: Task Execution Profile Runtime Fit

## Purpose

This analysis records the implementation evidence behind `TaskExecutionProfile v1`: typed per-task overlays for coordinator guidance, worker selection, review selection, participant policy, and sandbox choice.

## Existing Runtime Evidence

- `internal/config/autonomy.go` already provides global/workspace coordinator configuration with `agent_name`, `provider`, `model`, and `max_active_per_workspace`. The current coordinator model is workspace-scoped and singleton-oriented, so a per-task dedicated coordinator would change runtime semantics instead of merely adding a task field.
- `internal/daemon/coordinator_runtime.go` enforces coordinator bootstrap through the daemon runtime and keeps one active coordinator per workspace. This supports task-specific `guided` coordinator policy, but not dedicated per-task coordinator sessions in the MVP.
- `internal/workspace/workspace.go` already carries workspace-level `DefaultAgent` and `SandboxRef`. These are natural defaults for per-task worker/sandbox overlays.
- `internal/session/sandbox.go` resolves the sandbox from the workspace during session start. A task-level sandbox overlay can fit by feeding the resolved session create/start options, but it must not bypass tool policy, approval policy, or session authorization.
- `internal/daemon/task_runtime.go` starts task worker sessions as system sessions without passing task-level `AgentName` or `Provider`, which means current workers fall back to workspace defaults. `TaskExecutionProfile.Worker` should feed this path explicitly.
- `internal/task/types.go` already models task ownership, `network_channel`, and run-level `coordination_channel_id`, `required_capabilities`, and `preferred_capabilities`. This provides a natural compatibility point for participant and worker eligibility, but the task-level profile should be typed instead of hidden in `metadata_json`.
- `internal/task/lease.go` includes `ClaimCriteria.AgentName`, `RequiredCapabilities`, `PriorityMin`, and `CoordinationChannelID`. The claim contract has an agent-selection shape, but the store selector must be updated before agent-specific eligibility becomes real.
- `internal/store/globaldb/global_db_task_claim.go` filters claimable runs by pending/leased state, capabilities, coordination channel, and spawn-circuit state, but it does not currently filter by `agent_name`. A profile-aware claim path must add this query behavior and indexes.
- `.compozy/tasks/orch-improvs/_techspec_review_gate.md` already defines `ReviewPolicy` and `ReviewerSelector` for peer/channel/capability routing. `TaskExecutionProfile.Review` should extend this instead of creating a separate reviewer-routing system.

## Design Implications

1. Store per-task runtime selection as typed task-owned state, not `metadata_json`.
2. Preserve the existing coordinator singleton in MVP. `CoordinatorProfile.mode = "guided"` gives the daemon-managed coordinator task-specific instructions and policy hints, but does not create a per-task coordinator session.
3. Apply worker agent/provider/model selection at task-session start and claim eligibility, not through channel messages.
4. Apply review agent/provider/model selection through the review-gate selector and persisted review route metadata, not through channel transcripts.
5. Keep participant policy non-authoritative. It constrains and documents coordination surfaces; it does not grant ownership, verdict authority, or terminal-state permission.
6. Apply sandbox policy only during session start. It selects `inherit`, `none`, or a named `sandbox_ref`; it cannot override tool allowlists, approval policy, or authorization.

## MVP Recommendation

Implement `TaskExecutionProfile v1` with these typed subprofiles:

- `CoordinatorProfile`: `mode = inherit | guided`; optional agent/provider/model hints for the singleton coordinator. `dedicated` is out of scope.
- `WorkerProfile`: agent/provider/model hints plus allowed/preferred agent and capability selectors for worker sessions and claim eligibility.
- `ReviewProfile`: reviewer agent/provider/model hints and peer/channel/capability selectors, integrated with the review-gate `ReviewerSelector`.
- `ParticipantPolicy`: allowed/preferred channels, peers, agents, and capabilities for task coordination.
- `SandboxPolicy`: `mode = inherit | none | ref` plus optional `sandbox_ref`.

The profile is a runtime input owned and validated by `task.Service`. It is not an authority surface; task ownership, review authority, tool policy, session permission, and terminal state remain with their existing runtime owners.
