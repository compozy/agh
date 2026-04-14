---
status: pending
title: "Implement `TaskManager` creation, mutation, and identity rules"
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 04: Implement `TaskManager` creation, mutation, and identity rules

## Overview
Implement the daemon-owned service that becomes the single authority for task creation, mutation, lookup, and lifecycle governance. This task is where the domain semantics become executable: actor derivation, authorization checks, mutability rules, ownership behavior, and canonical task-state transitions must all live here.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. `TaskManager` MUST derive `created_by` and `origin` from trusted server-side context rather than payload fields for every writer surface.
2. `TaskManager` MUST enforce the TechSpec mutability rules, including immutable structural fields, optional mutable ownership, and manager-owned canonical task status.
3. `TaskManager` MUST expose create, get, list, update, child-task, and dependency-management operations that remain independently valid once the declared storage dependencies are met.
</requirements>

## Subtasks
- [ ] 4.1 Implement the task service/manager entrypoint and its injected dependencies.
- [ ] 4.2 Implement actor derivation and authorization checks for human, agent, automation, extension, and network writer surfaces.
- [ ] 4.3 Implement create and update flows that enforce ownership, mutability, and scope rules.
- [ ] 4.4 Implement read/list flows with filtering by scope, workspace, status, parent, owner, and channel.
- [ ] 4.5 Implement child-task creation and dependency-edge management through the manager rather than direct store calls.

## Implementation Details
Use the TechSpec sections "Actor and Identity Model", "Authorization Contract", "Mutability Rules", and "Key Decisions" as the contract. This task should keep business rules centralized in `internal/task`, leaving transports and ingress packages as thin callers.

### Relevant Files
- `internal/task/` — New manager/service implementation files created in the task domain.
- `internal/api/core/interfaces.go` — Reference for service-interface style consumed by transport layers.
- `internal/automation/manager.go` — Reference for daemon-owned manager construction and lifecycle patterns.
- `internal/network/validate.go` — Reference for surface-level validation style when deriving trusted caller context.
- `.compozy/tasks/core-tasks/_techspec.md` — Source of the authorization and mutability contract.

### Dependent Files
- `internal/api/core/handlers.go` — Will call into the manager methods created here.
- `internal/daemon/boot.go` — Will compose the new task manager into the daemon.
- `internal/extension/host_api.go` — Will depend on these operations for extension-originated task flows.

### Related ADRs
- [ADR-002: Support Global and Workspace Task Scope with Explicit Hierarchy and Bounded Dependencies](../adrs/adr-002.md) — Governs scope and hierarchy semantics.
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Makes the manager the canonical lifecycle authority.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs actor, origin, and owner semantics.

## Deliverables
- A composed `TaskManager` implementation in `internal/task`.
- Server-side actor derivation and authorization checks for all supported writer surfaces.
- Manager methods for create/get/list/update/children/dependencies with centralized business rules.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for end-to-end manager flows against real storage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Verify payload-supplied identity fields are ignored or rejected in favor of trusted actor context.
  - [ ] Verify updates cannot change immutable structural fields after creation.
  - [ ] Verify owner reassignment, empty-owner creation, and channel updates follow the allowed mutability rules.
  - [ ] Verify `global` task creation and `workspace` task creation require the correct trusted scope context.
- Integration tests:
  - [ ] Verify a task created by an agent session persists `created_by=agent`, immutable `origin`, and no owner when no owner is assigned.
  - [ ] Verify child-task creation and dependency-edge writes go through the manager and persist audit events correctly.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Task creation and mutation rules live in one canonical service instead of being duplicated across transports
- Actor identity, authorization, and task mutability behave exactly as specified in the revised TechSpec
