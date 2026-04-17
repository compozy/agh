---
status: completed
title: Composition-root runtime automation and task delegation scenarios
type: test
complexity: critical
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 04: Composition-root runtime automation and task delegation scenarios

## Overview

Add the runtime E2E scenarios that prove automations can create real system sessions and delegate real task runs through the live daemon orchestration graph. This task binds automation, tasks, sessions, and downstream transcript state into one composition-root proof instead of scattering the behavior across package-local tests.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a runtime E2E scenario that triggers a real automation path and proves prompt rendering, system session creation, transcript persistence, run status, and stop semantics.
2. MUST add a runtime E2E scenario that creates a task-backed automation job and proves real task-run delegation, `task_id`, `task_run_id`, lifecycle transitions, and session linkage where applicable.
3. MUST drive these scenarios through real daemon ingress paths such as webhook, observer events, or manual trigger surfaces that already exist in the product.
4. MUST assert automation and task behavior through domain-specific surfaces including automation runs, task records, task-run records, and linked session state.
5. SHOULD keep automation, task, and session orchestration in one task because the composition graph is tightly coupled and splitting it further would create undeclared dependencies.
</requirements>

## Subtasks
- [x] 4.1 Add runtime fixture seeding for automation jobs, triggers, and task-backed automation definitions.
- [x] 4.2 Implement the automation prompt-trigger scenario in `internal/daemon`.
- [x] 4.3 Implement the task-backed automation delegation scenario in `internal/daemon`.
- [x] 4.4 Add artifact capture and assertions for automation runs, task records, task runs, linked sessions, and downstream transcripts.
- [x] 4.5 Add focused regression coverage around delegated run status and session linkage behavior.

## Implementation Details

See TechSpec sections "PR-Required Runtime E2E", "Daemon-Only E2E In Current Product", and "Technical Considerations". These flows are intentionally daemon-lane scenarios because task workflows do not yet exist in the web product surface.

### Relevant Files
- `internal/daemon/daemon_integration_test.go` — composition-root home for automation and task delegation E2E.
- `internal/automation/manager_integration_test.go` — current automation runtime patterns that can inform real ingress setup.
- `internal/automation/trigger_integration_test.go` — trigger-driven automation behaviors that should remain aligned with the daemon-level E2E proof.
- `internal/task/manager_integration_test.go` — task-run lifecycle semantics that the daemon scenario must read through public surfaces.
- `internal/api/core/automation.go` — automation run and projection surface consumed by assertions.
- `internal/api/core/tasks.go` — task and task-run projection surface consumed by assertions.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — must expose automation/task seeding and artifact capture hooks.
- `internal/testutil/acpmock/testdata/` — requires automation-trigger and task-delegation mock-agent scenarios.
- `internal/api/httpapi/httpapi_integration_test.go` — later HTTP parity coverage depends on these daemon-truth scenarios.
- `web/e2e/automation.spec.ts` — later browser automation flow depends on these runtime scenarios for truth.
- `Makefile` — later runtime-lane command wiring must include these scenarios in the E2E target set.

### Related ADRs
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Automation and task delegation proof belongs where the runtime graph is wired together.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Runs, task records, and session linkage are the primary assertion surfaces.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Task lifecycle remains daemon-only E2E until a web task flow exists.

## Deliverables
- Composition-root daemon runtime E2E for automation prompt-trigger session creation
- Composition-root daemon runtime E2E for task-backed automation delegation
- Artifact helpers for automation runs, task records, task runs, and linked session state
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for automation ingress, delegated task runs, and session linkage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Automation and task artifact helpers serialize runs, task records, and session linkage consistently
  - [x] Runtime fixture seeding can register automation jobs and task-backed automation definitions without hidden defaults
  - [x] Assertion helpers distinguish completed session-creating runs from delegated task-backed runs
- Integration tests:
  - [x] Automation prompt trigger creates a completed system session with persisted transcript and run record
  - [x] Task-backed automation run creates a real task run with `task_id`, `task_run_id`, and expected lifecycle progression
  - [x] Linked session or attached-session references remain readable from public task and automation surfaces
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Real daemon E2E covers automation-created sessions and task-backed delegation
- Task lifecycle remains proven end to end even without a web task workflow
- Automation browser work can reuse this runtime lane as the source of truth
