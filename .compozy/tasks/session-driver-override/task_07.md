---
status: pending
title: Session Provider Override QA Plan and Regression Artifacts
type: test
complexity: high
dependencies:
  - task_06
---

# Task 07: Session Provider Override QA Plan and Regression Artifacts

## Overview

Generate the reusable QA planning artifacts for session provider override before live execution begins. This task must leave the feature with a concrete test plan, provider-focused manual test cases, and regression-suite definitions that the follow-up execution task can consume without re-deciding scope, environments, or artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and tasks 01-06 before planning coverage (`_prd.md` is absent; requirements come from the TechSpec and generated tasks)
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/session-driver-override` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/session-driver-override/qa/`
- FOCUS ON WHAT MUST BE PROVEN - config resolution, runtime persistence, migration/repair, API contracts, workspace provider options, web dialog flow, and resume failure behavior all need traceable coverage
- DO NOT EXECUTE THE FLOWS IN THIS TASK - this is planning, traceability, and artifact generation only
- GREENFIELD: nao aceitar smoke generico; cada seam critica precisa de casos rastreaveis para backend, transport, CLI, e web flows
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/session-driver-override`
- MUST generate a feature-level QA plan under `.compozy/tasks/session-driver-override/qa/test-plans/`
- MUST generate manual test cases covering provider override resolution, provider-owned runtime field replacement, create-time validation failures, session persistence, global DB migration, legacy repair, explicit API/CLI/Host API surfaces, workspace provider option discovery, dialog-driven creation, and resume failure UX
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for the follow-up `/qa-execution` task
- MUST include evidence expectations for backend logs/errors, DB migration state, API payloads, CLI output, and browser-visible UI states
- SHOULD map each P0/P1 case back to one of tasks 01-06 or to the corresponding TechSpec rule
</requirements>

## Subtasks
- [ ] 7.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/session-driver-override`
- [ ] 7.2 Write the feature-level provider-override QA plan with scope, risks, environments, and entry/exit criteria
- [ ] 7.3 Generate manual test cases for backend resolution, persistence/migration, API/CLI surfaces, and web dialog/resume UX
- [ ] 7.4 Build regression-suite definitions with explicit P0/P1 flow ordering for `/qa-execution`
- [ ] 7.5 Validate traceability and handoff completeness for task_08

## Implementation Details

See the TechSpec "Testing Approach" plus tasks 01-06. The QA plan should turn the feature into an execution-ready matrix: which workspaces/providers must exist, which create/resume scenarios are P0, which storage states must be simulated, and what evidence counts as proof for backend, transport, and browser behavior.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required workflow, output layout, and naming rules for QA planning
- `.compozy/tasks/session-driver-override/_techspec.md` - authoritative feature scope, testing approach, and risk model
- `.compozy/tasks/session-driver-override/task_01.md` - config-resolution and provider-owned runtime cases that need explicit QA coverage
- `.compozy/tasks/session-driver-override/task_03.md` - migration and legacy repair scenarios that need storage-focused QA
- `.compozy/tasks/session-driver-override/task_04.md` - explicit API/CLI/Host API surface changes that need contract coverage
- `.compozy/tasks/session-driver-override/task_06.md` - dialog flow and resume-failure UX that need browser-facing coverage

### Dependent Files
- `.compozy/tasks/session-driver-override/qa/test-plans/session-provider-override-test-plan.md` - feature-level QA plan created by this task
- `.compozy/tasks/session-driver-override/qa/test-plans/*-regression.md` - regression-suite document(s) consumed by task_08
- `.compozy/tasks/session-driver-override/qa/test-cases/TC-*.md` - manual test cases with priorities, preconditions, and expected results
- `.compozy/tasks/session-driver-override/qa/issues/BUG-*.md` - only if planning uncovers a concrete discrepancy while documenting coverage
- `.compozy/tasks/session-driver-override/task_08.md` - execution task that must consume this artifact set without changing the output path

### Related ADRs
- [ADR-001: Model Session Driver Selection As A Provider Override](adrs/adr-001.md) - QA planning must prove override scope stays on provider/runtime, not agent identity
- [ADR-003: Persist Effective Session Provider And Fail Explicitly On Mismatch](adrs/adr-003.md) - QA planning must prove persistence and explicit-failure semantics
- [ADR-004: Use Explicit Session Creation Surfaces For Provider Selection](adrs/adr-004.md) - QA planning must cover explicit create surfaces and dialog UX
- [ADR-005: Migrate Session Provider State In Place And Repair Legacy Metadata Once](adrs/adr-005.md) - QA planning must cover migration and one-time repair

## Deliverables
- `.compozy/tasks/session-driver-override/qa/test-plans/session-provider-override-test-plan.md`
- One or more `.compozy/tasks/session-driver-override/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Provider-override-focused manual test cases under `.compozy/tasks/session-driver-override/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 coverage for create override, persisted resume, invalid provider failures, legacy repair, transport parity, and web dialog/resume UX **(REQUIRED)**
- Stable artifact layout under `.compozy/tasks/session-driver-override/qa/` that task_08 can consume without path changes **(REQUIRED)**
- Traceability from each P0/P1 case back to tasks 01-06 or the TechSpec rules **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `session-provider-override-test-plan.md` includes objectives, scope, environments, entry/exit criteria, and a risk table focused on resolution, persistence, migration, and UI seams
  - [ ] Manual test cases exist for no-override behavior, explicit provider override, invalid provider create failure, persisted resume, unavailable-provider resume failure, legacy repair, CLI/API/Host API parity, and web dialog/resume flows
  - [ ] Regression-suite documents define smoke, targeted, and full execution lanes with explicit P0/P1 ordering
  - [ ] Each P0/P1 test case names the exact task or TechSpec rule it is proving
- Integration tests:
  - [ ] All generated artifacts land under `.compozy/tasks/session-driver-override/qa/` and can be consumed directly by `/qa-execution`
  - [ ] The regression suite covers backend, storage, transport, CLI, and browser-visible behavior rather than only parser-level checks
  - [ ] Any bug report created during planning references the originating test case or documented discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-report` has been run explicitly and its artifacts are stored under `.compozy/tasks/session-driver-override/qa/`
- Task_08 can begin execution without redefining scope, priorities, or output paths
