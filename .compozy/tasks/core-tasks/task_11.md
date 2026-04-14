---
status: completed
title: "Integrate extension host APIs with the task domain"
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
---

# Task 11: Integrate extension host APIs with the task domain

## Overview
Expose the task domain safely to extensions through the existing host API and capability system. This task should let extensions create and manipulate tasks as first-class writers while preserving server-derived identity, explicit capabilities, and manager-owned lifecycle rules.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. Extensions MUST reach task operations only through explicit host API methods protected by capability checks.
2. Extension-originated task writes MUST derive `created_by` and immutable `origin` from trusted extension context, not payload values.
3. Extension integrations MUST preserve manager-owned lifecycle authority and dedicated-session execution rules for executable subtasks.
</requirements>

## Subtasks
- [x] 11.1 Add task-domain methods to the extension host API surface.
- [x] 11.2 Extend capability checks to cover task create/update/run operations.
- [x] 11.3 Carry trusted extension identity and origin metadata into task manager calls.
- [x] 11.4 Support extension-originated task run operations without bypassing lifecycle guards.
- [x] 11.5 Add host API tests for permitted and forbidden task operations.

## Implementation Details
Use the TechSpec "Integration Points" section for extensions and the authorization model accepted during the redesign. Follow the existing host API and capability-checker patterns already used for bridges and other extension-backed operations.

### Relevant Files
- `internal/extension/host_api.go` — Primary host API surface that task methods must extend.
- `internal/extension/host_api_bridges.go` — Reference for daemon-to-extension bridge style and context propagation.
- `internal/extension/capability.go` — Existing capability-checker model that must gain task permissions.
- `internal/extension/manager.go` — Reference for manager-owned extension lifecycle and wiring.
- `internal/task/` — Task manager interface consumed by the extension host API.
- `internal/daemon/extensions.go` — Daemon composition surface for extension-host dependencies.

### Dependent Files
- `internal/extension/host_api_test.go` — Will expand to cover task-domain surfaces.
- `internal/extension/host_api_integration_test.go` — Will need end-to-end task-backed host API cases.

### Related ADRs
- [ADR-003: Use Queue-First TaskRun Lifecycle with Central TaskManager Authority](../adrs/adr-003.md) — Requires lifecycle operations to remain manager-owned.
- [ADR-005: Derive Actor Identity Server-Side and Allow Optional Mutable Ownership](../adrs/adr-005.md) — Governs identity and origin derivation for extension writers.
- [ADR-006: Execute Subtasks Through an Injected Session Bridge with Dedicated Sessions by Default](../adrs/adr-006.md) — Governs how extension-originated executable subtasks run.

## Deliverables
- Extension host API methods for task create/update/query/run flows.
- Capability enforcement for task-domain operations.
- Tests for trusted identity/origin derivation and forbidden operations.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for extension-originated task workflows **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Verify extensions without the required capability cannot create, mutate, or run tasks.
  - [x] Verify payload-supplied identity fields are ignored in favor of trusted extension context.
  - [x] Verify extension-originated run lifecycle requests still pass through manager-owned transition checks.
- Integration tests:
  - [x] Verify a capability-granted extension can create a task and enqueue a run through the host API.
  - [x] Verify an extension can start an executable subtask and receive a dedicated session through the bridge-backed path.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Extensions can participate in the task domain as explicit writers without bypassing daemon authority
- Capability checks and identity derivation remain correct for extension-originated task flows
