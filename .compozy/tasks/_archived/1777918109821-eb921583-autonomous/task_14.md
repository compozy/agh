---
status: completed
title: Coordinator Bootstrap And Restricted Orchestration
type: backend
complexity: critical
dependencies:
  - task_04
  - task_06
  - task_09
  - task_10
  - task_11
  - task_13
---

# Task 14: Coordinator Bootstrap And Restricted Orchestration

## Overview
Implement the coordinator-agent runtime that owns semantic orchestration once executable work exists. The coordinator is a normal managed session, configured per workspace, restricted to orchestration tools, bootstrapped from task-run enqueue/start boundaries rather than task creation, and required to coordinate workers through task-bound channels when conversation is needed.

<critical>
- ALWAYS READ `_techspec.md`, ADR-004, ADR-005, ADR-006, ADR-009, ADR-010, ADR-011, and ADR-012 before implementing coordinator behavior
- COORDINATOR SPAWNS ON EXECUTABLE WORK, NOT ON TASK CREATION
- ONE COORDINATOR PER WORKSPACE IN MVP; no global-scope auto-spawn
- COORDINATOR MUST USE RESTRICTED TOOLING and cannot spawn another coordinator
- COORDINATOR MUST USE THE RUN'S COORDINATION CHANNEL for operational worker communication; it must not use channel messages as task ownership state
- TESTS REQUIRED - config precedence, bootstrap idempotency, restricted tools, manual coexistence, and failure recovery must be covered
- NO WORKAROUNDS - do not fake orchestration with hard-coded prompt snippets or private daemon shortcuts
</critical>

<requirements>
- MUST resolve coordinator provider/model/config using workspace override > global autonomy config > bundled/default coordinator definition.
- MUST bootstrap one coordinator session per workspace when a task run is enqueued/started/approved for coordinated execution.
- MUST require coordinated runs to expose a stable `coordination_channel_id` before coordinator orchestration begins.
- MUST inject situation context from task_04 and agent task/spawn/channel tools from tasks 06, 09, and 13.
- MUST use the coordination channel for `status`, `request`, `blocker`, `handoff`, `result`, and `review_request` messages when worker conversation is useful.
- MUST restrict coordinator permissions to orchestration-safe verbs and deny coordinator-to-coordinator spawn.
- MUST integrate with scheduler wake behavior from task_11 without letting scheduler claim work.
- MUST preserve user-created tasks, user-started sessions, and explicit operator task controls.
</requirements>

## Subtasks
- [x] 14.1 Add coordinator session resolver and workspace-scoped singleton bootstrap logic.
- [x] 14.2 Wire bootstrap to task-run enqueue/start/approval events and coordination-channel binding from task_10.
- [x] 14.3 Build coordinator prompt/context assembly using Situation Surface providers from task_04.
- [x] 14.4 Restrict coordinator tool/permission set and enforce no coordinator-to-coordinator spawn.
- [x] 14.5 Add recovery behavior for coordinator crash/stop while executable work remains pending.
- [x] 14.6 Add tests for bootstrap idempotency, config precedence, restricted tools, coordination-channel usage, manual coexistence, and restart.

## Implementation Details
The coordinator is not a new daemon-internal AI loop. It should be represented as a managed session with typed role/lineage and a restricted command/tool surface. Its job is to claim or delegate task runs, create child tasks when needed, spawn bounded children, coordinate through the run channel when useful, synthesize results, and respect task approvals.

The coordinator uses channels for operational conversation only: status requests, blockers, handoffs, result exchange, review requests, and synthesis context. It must use task APIs for claim, heartbeat, complete, fail, release, and terminal status.

Global-scope runs should not auto-spawn a coordinator in MVP. They can remain operator-managed or explicitly assigned until a later global orchestration TechSpec.

### Relevant Files
- `internal/daemon/daemon.go` - composition root and coordinator lifecycle wiring.
- `internal/config/config.go` - coordinator provider/model resolver from task_01.
- `internal/session/manager.go` - coordinator session creation using task_12/task_13 metadata.
- `internal/task/manager.go` - task-run enqueue events and claim APIs.
- `internal/network/*` - coordination channel access and message metadata.
- `internal/daemon/task_runtime.go` - bridge between task runtime and coordinator bootstrap.
- `internal/hooks/*` - coordinator hook payloads and dispatch.
- `internal/api/contract/*` - coordinator/session read model updates if public.
- `.resources/hermes/run_agent.py` - reference for agent runner boot orchestration.
- `.resources/hermes/agent/trajectory.py` - reference for durable reasoning/action trajectories.
- `.resources/paperclip/doc/plans/2026-02-20-issue-run-orchestration-plan.md` - reference for issue-run orchestration boundaries.
- `.resources/multica/packages/core/autopilots/index.ts` - reference for configured autonomous worker concepts.
- `.resources/multica/packages/core/autopilots/mutations.ts` - reference for explicit autopilot start/stop mutations.

### Dependent Files
- `web/src/systems/tasks/*` - task_15 labels coordinator-triggered execution accurately.
- `packages/site/content/runtime/core/autonomy/` - task_16 documents coordinator behavior.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 plans coordinator E2E coverage.

### Related ADRs
- [ADR-004: Coordinator-Agent Plus Mechanical Scheduler](adrs/adr-004.md) - semantic orchestration model.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - bootstrap trigger and config precedence.
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - coordinator spawn restrictions.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - `coordinator.*` hooks.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - autonomy remains additive.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - contracts/docs when public behavior changes.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - coordinator/worker communication contract.

## Deliverables
- Workspace-scoped coordinator bootstrap and recovery.
- Restricted coordinator session role, prompt/context, and tool permissions.
- Integration with task-run enqueue/start boundary and scheduler wake behavior.
- Coordinator use of task-run coordination channels for operational worker communication.
- Unit tests with 80%+ coverage for coordinator resolver/permission helpers **(REQUIRED)**.
- Integration tests proving coordinator bootstrap and manual coexistence **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Coordinator config resolver honors workspace override > global config > bundled default.
  - [x] Bootstrap decision ignores task creation and triggers only on executable run enqueue/start/approval.
  - [x] Bootstrap rejects or delays coordinated work that lacks a required `coordination_channel_id`.
  - [x] Workspace singleton logic prevents duplicate coordinators under concurrent enqueue events.
  - [x] Restricted permission set excludes coordinator-spawn-coordinator and disallowed operator verbs.
  - [x] Global-scope task runs do not auto-spawn coordinators in MVP.
- Integration tests:
  - [x] Starting a user-created task enqueues a run and bootstraps one workspace coordinator.
  - [x] Starting multiple tasks in the same workspace reuses the existing coordinator.
  - [x] Coordinator restart/recovery occurs when executable work remains pending after crash/stop.
  - [x] Coordinator can claim/delegate through public agent APIs, not private manager calls.
  - [x] Coordinator and worker exchange `status`/`blocker`/`result` messages through the run channel, while task completion still uses the token-fenced task API.
  - [x] Manual sessions and operator task commands still work with coordinator enabled and disabled.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Semantic orchestration is handled by a configurable coordinator-agent.
- Coordinator/worker communication is auditable through task-bound channels.
- The daemon remains responsible only for mechanical lifecycle, safety, and API enforcement.
