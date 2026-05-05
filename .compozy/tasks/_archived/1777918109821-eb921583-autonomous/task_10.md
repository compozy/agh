---
status: completed
title: Operator Start Publish Approval Execution Boundary
type: backend
complexity: high
dependencies:
  - task_06
  - task_08
---

# Task 10: Operator Start Publish Approval Execution Boundary

## Overview
Make the task lifecycle boundary explicit: creating a task records intent, while publish/start/approval enqueues executable work, binds the run to a coordination channel, and triggers coordinator orchestration. This preserves user-created tasks and user-started sessions without forcing users to set orchestration flags at creation time.

<critical>
- ALWAYS READ `_techspec.md`, ADR-005, ADR-010, ADR-011, and ADR-012 before changing task lifecycle behavior
- TASK CREATION MUST NOT ENQUEUE A RUN OR SPAWN A COORDINATOR
- TASK CREATION MUST NOT REQUIRE OR CREATE A COORDINATION CHANNEL
- PUBLISH/START/APPROVAL IS THE EXECUTION BOUNDARY that creates claimable work
- USER-CREATED AND AGENT-CREATED TASKS MUST BOTH BE SUPPORTED
- TESTS REQUIRED - task creation, start, publish, approval, enqueue idempotency, and regression of existing manual flows must be covered
- NO WORKAROUNDS - do not add `orchestration_required` as a user-created-task gate
</critical>

<requirements>
- MUST ensure task creation alone creates no claimable run and no coordinator session.
- MUST make operator publish/start/approval enqueue exactly one task run with idempotency protection.
- MUST create or resolve a stable `coordination_channel_id` for workspace-scoped coordinated runs at enqueue/start time.
- MUST record clear actor/origin metadata for user, agent, coordinator, and automation initiated starts.
- MUST emit task-domain audit events and typed hooks at enqueue/start boundaries through the task_03 bridge.
- MUST preserve existing task drafts, manual session start behavior, and operator run controls.
- MUST update API/CLI/web copy and generated contracts where lifecycle fields are exposed.
</requirements>

## Subtasks
- [x] 10.1 Audit current task creation, publish, run enqueue, and approval code paths.
- [x] 10.2 Refactor creation paths so they never enqueue work implicitly.
- [x] 10.3 Implement or normalize publish/start/approval enqueue behavior with idempotency keys.
- [x] 10.4 Add coordination channel create-or-resolve behavior plus actor/origin and hook/audit metadata at the execution boundary.
- [x] 10.5 Update CLI/API responses and web-facing DTOs if lifecycle state labels change.
- [x] 10.6 Add regression tests for user-created, agent-created, manually started, and approval-started tasks.

## Implementation Details
The user should be able to create a task normally. When the user explicitly starts it, publishes it into executable state, or approves an agent-created task for execution, the daemon enqueues a run and binds that workspace-scoped coordinated run to a coordination channel. Coordinator bootstrap in task_14 observes this boundary; it does not depend on a user-provided `orchestration_required` flag.

Coordination channel binding must be idempotent with run enqueue. Repeated start/approval requests must not create duplicate runs or duplicate coordination channels for the same executable run.

### Relevant Files
- `internal/task/manager.go` - create, enqueue, start, publish, and approval lifecycle.
- `internal/task/types.go` - task/run status, actor, authority, and origin fields.
- `internal/task/events.go` - audit events around enqueue/start.
- `internal/network/*` - channel create-or-resolve and task-run correlation metadata.
- `internal/api/udsapi/task*.go` - CLI-facing task lifecycle handlers.
- `internal/api/httpapi/*task*` - HTTP task lifecycle handlers if present.
- `internal/api/contract/tasks.go` - task and run lifecycle DTOs.
- `internal/cli/task.go` - operator task commands and output text.
- `.resources/multica/packages/core/issues/mutations.ts` - reference for create-vs-start mutation separation.
- `.resources/paperclip/doc/execution-semantics.md` - reference for explicit execution semantics.

### Dependent Files
- `internal/scheduler/*` - task_11 wakes on executable pending runs.
- `internal/daemon/*coordinator*` - task_14 bootstraps coordinator on executable work.
- `web/src/systems/tasks/*` - task_15 updates labels and e2e for the boundary.
- `packages/site/content/runtime/core/tasks/` - task_16 documents lifecycle semantics.

### Related ADRs
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - run enqueue is coordinator trigger.
- [ADR-007: Minimal Network Evolution for Local Autonomy](adrs/adr-007.md) - channel metadata remains local and minimal.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - user-created tasks and sessions remain supported.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - contract/web/docs updates are required when lifecycle fields change.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - coordination channel binding at run enqueue.

## Deliverables
- Explicit task creation versus execution-start boundary.
- Idempotent publish/start/approval enqueue behavior.
- Idempotent coordination channel binding for coordinated runs.
- Audit/hook metadata for execution boundary events.
- Unit tests with 80%+ coverage for lifecycle decision helpers **(REQUIRED)**.
- Integration tests for task creation/start/publish/approval flows **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Creating a task as a user records the task and creates no task run.
  - [x] Creating a task as an agent records actor/origin and creates no task run until approved or started.
  - [x] Publish/start/approval produces one pending run with stable idempotency behavior.
  - [x] Publish/start/approval for workspace-scoped coordinated execution creates or resolves exactly one `coordination_channel_id`.
  - [x] Repeated start requests return the existing run or a documented conflict without duplicate queue entries.
  - [x] Repeated start requests do not create duplicate coordination channels for the same run.
  - [x] Enqueue events include actor/origin fields required by hooks and audit logs.
- Integration tests:
  - [x] A manually created task can be started by the user and then claimed through task_08.
  - [x] A manually created task creates no coordination channel until explicit execution start.
  - [x] An agent-created task can be approved by the user and then claimed through task_08.
  - [x] Manual user-started sessions still work without creating or claiming tasks.
  - [x] Existing task list/detail endpoints represent draft/pending/running/completed state accurately.
  - [x] Generated contracts, web typecheck, and web tests pass if DTOs change.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Task creation remains manual/agent friendly while execution is explicitly started.
- No user-facing `orchestration_required` flag is needed to trigger coordinator orchestration later.
- Coordinated execution has a stable task-run channel without making task creation heavy.
