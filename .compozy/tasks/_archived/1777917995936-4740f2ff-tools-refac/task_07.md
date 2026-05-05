---
status: completed
title: Automation Tool Family
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 07: Automation Tool Family

## Overview

Promote automation jobs, triggers, and run inspection into the canonical tool surface so agents can manage AGH automation without shelling out or relying on web-only controls. The task should reuse the current automation manager, validators, and persistence model already present on this branch.

<critical>
- ALWAYS READ `_techspec.md` and ADR-006 before widening automation mutation
- REFERENCE TECHSPEC sections "Mutable Surface Policy", "Automation", and "Agent Manageability Plan"
- FOCUS ON WHAT: expose the automation lifecycle through tools; do not fork automation storage or scheduler behavior
- MINIMIZE CODE — reuse current automation validators, manager methods, and persistence
- TESTS REQUIRED — create/update/delete/trigger and run inspection must all prove deterministic parity with existing management paths
</critical>

<requirements>
1. MUST expose automation job and trigger CRUD through the tool surface where the current runtime already has authoritative writers.
2. MUST expose automation run inspection/history and trigger operations through tools.
3. MUST preserve current validation, scheduling, dispatch, and permission semantics for the same job or trigger changes.
4. MUST require approval for mutating automation operations and keep deterministic denial behavior when policy or validation blocks them.
</requirements>

## Subtasks

- [x] 7.1 Add job and trigger inspection tools over current automation read/query paths
- [x] 7.2 Add create/update/delete/enable/disable/trigger tools over current automation writers
- [x] 7.3 Add run inspection/history tools over current automation run records
- [x] 7.4 Wire approval, policy, and validation behavior into the automation tool family
- [x] 7.5 Add unit and integration coverage for lifecycle parity and denial rules

## Implementation Details

See TechSpec sections "Data Models", "Agent Manageability Plan", and "Implementation Steps". The automation tool family should surface the existing lifecycle directly rather than creating a separate tool-only automation API.

### Relevant Files

- `internal/automation/manager.go` — authoritative automation lifecycle entry point
- `internal/automation/validate.go` — current job and trigger validation rules
- `internal/automation/persistence.go` — storage behavior that tool writes must preserve
- `internal/api/core/automation.go` — current public DTOs and handler semantics
- `internal/cli/automation.go` — current operator management path

### Dependent Files

- `internal/daemon/automation_resources.go` — runtime automation exposure already wired through the daemon
- `web/src/systems/automation/*` — current automation UI surface that must stay aligned with shared DTO semantics

### Related ADRs

- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`; checked `web/src/systems/automation/adapters/automation-api.ts`, `hooks/use-automation*.ts`, and `components/automation-*.tsx` because any shared automation DTO change must co-ship with their tests.
- `packages/site`: `packages/site/content/runtime/core/automation/*.mdx` and CLI reference pages under `runtime/cli-reference/automation/`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: affects automation resources and extension-triggered automation flows that rely on shared manager behavior.
- Agent manageability: brings automation CRUD and run inspection into the canonical tool surface instead of leaving it on CLI/UI-only management paths.
- Config lifecycle: no new top-level config keys expected, but tool behavior must respect existing automation defaults and validation rules in config.

## Deliverables

- Automation job and trigger tool family
- Automation run inspection/history tools
- Approval-integrated mutation flow for automation operations
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for lifecycle parity **(REQUIRED)**

## Tests

- Unit tests:
- [x] job and trigger create/update/delete validation matches current manager rules
- [x] enable/disable/trigger operations preserve current scheduler and dispatch semantics
- [x] denied or invalid automation operations return deterministic reason codes
- Integration tests:
- [x] tool-driven automation lifecycle matches existing CLI or API behavior for the same runtime state
- [x] automation run history exposed through tools matches persisted run records and current UI expectations
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agents can manage automation jobs and triggers through dedicated tools
- Automation validation, scheduling, and run-history behavior stays identical to the existing authoritative runtime path
