---
status: completed
title: Unified Capabilities QA Plan and Regression Artifacts
type: test
complexity: high
dependencies:
    - task_01
    - task_02
    - task_03
    - task_04
    - task_05
    - task_06
    - task_07
    - task_08
---

# Task 09: Unified Capabilities QA Plan and Regression Artifacts

## Overview

Generate the reusable QA planning artifacts for unified capabilities before live execution begins. This task must leave the feature with a concrete test plan, capability-focused manual test cases, and regression-suite definitions that cover backend, `web/`, and `packages/site` surfaces while staying compatible with the follow-up execution task.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, tasks 01-08, and the rewritten docs before planning coverage (`_prd.md` is absent; requirements come from the TechSpec and generated tasks)
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/unified-capabilities` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/unified-capabilities/qa/`
- FOCUS ON WHAT MUST BE PROVEN - backend unification, transfer semantics, discovery/API coherence, frontend contract alignment, and site documentation consistency all need explicit coverage
- DO NOT EXECUTE THE FLOWS IN THIS TASK - this is planning, prioritization, traceability, and artifact generation only
- GREENFIELD: avoid generic smoke plans; every P0/P1 case must prove a real seam changed by the unification
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/unified-capabilities`
- MUST generate a feature-level QA plan under `.compozy/tasks/unified-capabilities/qa/test-plans/`
- MUST generate manual test cases covering backend schema/digesting, transfer kind replacement, lifecycle preservation, discovery/API contract alignment, frontend network UX, and `packages/site` protocol/runtime docs
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for the follow-up `/qa-execution` task
- MUST define artifact locations under `.compozy/tasks/unified-capabilities/qa/` for issues, screenshots, and verification reporting
- SHOULD map each P0/P1 case back to tasks 01-08 or to the relevant TechSpec/ADR rule it is proving
</requirements>

## Subtasks
- [ ] 9.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/unified-capabilities`
- [ ] 9.2 Write the feature-level QA plan with scope, risks, environments, and entry/exit criteria across backend, web, and docs
- [ ] 9.3 Generate manual test cases for unified capability schema, transfer behavior, discovery/API surfaces, frontend network UX, and site docs
- [ ] 9.4 Build regression-suite definitions with explicit P0/P1 ordering for `/qa-execution`
- [ ] 9.5 Validate traceability and handoff completeness for task_10

## Implementation Details

See the TechSpec "Testing Approach" plus tasks 01-08. The QA plan should translate the unified capability effort into execution-ready evidence: what must be proven on backend/runtime, what the web client must render, what the public docs must say, and where all artifacts will be stored for task_10.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required workflow, output layout, and artifact naming rules for QA planning
- `.compozy/tasks/unified-capabilities/_techspec.md` - authoritative source for schema, transfer, discovery, and validation expectations
- `.compozy/tasks/unified-capabilities/task_04.md` - backend discovery/API seams that need explicit P0/P1 coverage
- `.compozy/tasks/unified-capabilities/task_06.md` - frontend network UX and typed-client coverage requirements
- `.compozy/tasks/unified-capabilities/task_07.md` - `packages/site` protocol reference coverage requirements
- `.compozy/tasks/unified-capabilities/task_08.md` - `packages/site` runtime docs coverage requirements

### Dependent Files
- `.compozy/tasks/unified-capabilities/qa/test-plans/unified-capabilities-test-plan.md` - feature-level QA plan created by this task
- `.compozy/tasks/unified-capabilities/qa/test-plans/*-regression.md` - regression-suite document(s) consumed by task_10
- `.compozy/tasks/unified-capabilities/qa/test-cases/TC-*.md` - manual test cases with priorities, preconditions, and expected results
- `.compozy/tasks/unified-capabilities/qa/issues/BUG-*.md` - only if planning uncovers a concrete discrepancy while documenting coverage
- `.compozy/tasks/unified-capabilities/qa/screenshots/` - reserved output path for browser/doc evidence used by `/qa-execution`
- `.compozy/tasks/unified-capabilities/task_10.md` - execution task that must consume this artifact set without changing the output path

### Related ADRs
- [ADR-001: Capability Is the Single Network Capability Artifact](adrs/adr-001.md) - QA planning must prove the steady-state single-concept model
- [ADR-002: Keep Current Capability Authoring Layouts and Use a Canonical Structured Schema](adrs/adr-002.md) - QA planning must cover local authoring, digesting, and schema behavior
- [ADR-003: Replace `recipe` Wire Semantics with `capability` While Preserving Interaction Behavior](adrs/adr-003.md) - QA planning must prove transfer and lifecycle semantics remain correct

## Deliverables
- `.compozy/tasks/unified-capabilities/qa/test-plans/unified-capabilities-test-plan.md`
- One or more `.compozy/tasks/unified-capabilities/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Unified-capability manual test cases under `.compozy/tasks/unified-capabilities/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 coverage for backend, web, and `packages/site` seams changed by the unification **(REQUIRED)**
- Stable artifact layout under `.compozy/tasks/unified-capabilities/qa/` that task_10 can consume without path changes **(REQUIRED)**
- Traceability from each P0/P1 case back to tasks 01-08 or the TechSpec/ADR rules **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `unified-capabilities-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, and a risk table spanning backend, web, and site docs
  - [ ] Manual test cases exist for unified schema/digest behavior, transfer kind replacement, discovery/API alignment, frontend network peer details, and site protocol/runtime docs
  - [ ] Regression-suite documents define smoke, targeted, and full execution lanes with explicit P0/P1 ordering
  - [ ] Each P0/P1 test case names the exact task, TechSpec rule, or ADR it is proving
- Integration tests:
  - [ ] All generated artifacts land under `.compozy/tasks/unified-capabilities/qa/` and can be consumed directly by `/qa-execution`
  - [ ] The regression suite covers backend, frontend, and documentation-visible behavior rather than only parser-level checks
  - [ ] Any bug report created during planning references the originating test case or documented discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-report` has been run explicitly and its artifacts are stored under `.compozy/tasks/unified-capabilities/qa/`
- Task_10 can begin execution without redefining scope, priorities, or output paths
