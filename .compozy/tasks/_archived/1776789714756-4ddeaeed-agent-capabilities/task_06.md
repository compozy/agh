---
status: completed
title: Agent Capabilities QA Plan and Regression Artifacts
type: test
complexity: high
dependencies:
  - task_04
---

# Task 06: Agent Capabilities QA Plan and Regression Artifacts

## Overview

Generate the reusable QA planning artifacts for agent capabilities before live execution begins. This task must leave the feature with a concrete test plan, capability-focused manual test cases, and regression-suite definitions that the follow-up execution task can consume without re-deciding scope, priorities, or artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, RFC 003, and tasks 01-05 before planning coverage (`_prd.md` is absent; requirements come from the TechSpec and generated tasks)
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/agent-capabilities` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/agent-capabilities/qa/`
- FOCUS ON WHAT MUST BE PROVEN - loader correctness, join plumbing, brief discovery, rich discovery, empty-catalog behavior, and envelope-size guards all need explicit coverage
- DO NOT EXECUTE THE FLOWS IN THIS TASK - this is planning, traceability, and artifact generation only
- GREENFIELD: nao aceitar smoke generico; cada seam critica precisa de casos rastreaveis para runtime, protocol, e API surfaces
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/agent-capabilities`
- MUST generate a feature-level QA plan under `.compozy/tasks/agent-capabilities/qa/test-plans/`
- MUST generate manual test cases covering local catalog loading, mixed-layout validation failures, runtime join plumbing, brief peer-card projection, explicit rich `whois` discovery, no-catalog behavior, unknown-ID behavior, and oversized-response guards
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for the follow-up `/qa-execution` task
- MUST include evidence expectations for daemon/runtime, router-level envelopes, API payload visibility, and documentation consistency where user-visible
- SHOULD map each P0/P1 case back to one of tasks 01-05 or to the corresponding TechSpec projection rule
</requirements>

## Subtasks
- [x] 6.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/agent-capabilities`
- [x] 6.2 Write the feature-level capability QA plan with scope, risks, environments, and entry/exit criteria
- [x] 6.3 Generate manual test cases for loader, join, brief discovery, rich discovery, empty-catalog, and limit-guard scenarios
- [x] 6.4 Build regression-suite definitions with explicit P0/P1 flow ordering for `/qa-execution`
- [x] 6.5 Validate traceability and handoff completeness for task_07

## Implementation Details

See the TechSpec "Testing Approach" plus tasks 01-05. The QA plan should translate the feature into execution-ready artifacts: which local layouts must be exercised, which protocol shapes must be observed, what counts as evidence, and which failures are P0 versus P1.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required workflow, output layout, and naming rules for QA planning
- `.compozy/tasks/agent-capabilities/_techspec.md` - source of truth for local catalog rules, brief discovery, and rich discovery
- `.compozy/tasks/agent-capabilities/task_01.md` - loader and validation scenarios that need explicit QA coverage
- `.compozy/tasks/agent-capabilities/task_03.md` - brief capability projection scenarios that need operator-visible evidence
- `.compozy/tasks/agent-capabilities/task_04.md` - explicit rich `whois` discovery and envelope-limit scenarios that need P0/P1 prioritization

### Dependent Files
- `.compozy/tasks/agent-capabilities/qa/test-plans/agent-capabilities-test-plan.md` - feature-level QA plan created by this task
- `.compozy/tasks/agent-capabilities/qa/test-plans/*-regression.md` - regression-suite document(s) consumed by task_07
- `.compozy/tasks/agent-capabilities/qa/test-cases/TC-*.md` - manual test cases with priorities, preconditions, and expected results
- `.compozy/tasks/agent-capabilities/qa/issues/BUG-*.md` - only if planning uncovers a concrete discrepancy while documenting coverage
- `.compozy/tasks/agent-capabilities/task_07.md` - execution task that must consume this artifact set without changing the output path

### Related ADRs
- [ADR-001: Explicit Capability Catalogs](adrs/adr-001.md) - QA planning must prove explicit catalogs drive discovery instead of inference
- [ADR-002: Dual Storage Modes Without Merge](adrs/adr-002.md) - QA planning must cover mixed-layout rejection and directory/file mode behavior
- [ADR-003: Soft Outcome-Oriented Capability Model](adrs/adr-003.md) - QA planning must cover required fields, optional metadata, and rich discovery semantics

## Deliverables
- `.compozy/tasks/agent-capabilities/qa/test-plans/agent-capabilities-test-plan.md`
- One or more `.compozy/tasks/agent-capabilities/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Capability-focused manual test cases under `.compozy/tasks/agent-capabilities/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 coverage for loader validity, join plumbing, brief discovery, rich discovery, empty-catalog behavior, unknown-ID handling, and envelope-size guards **(REQUIRED)**
- Stable artifact layout under `.compozy/tasks/agent-capabilities/qa/` that task_07 can consume without path changes **(REQUIRED)**
- Traceability from each P0/P1 case back to tasks 01-05 or the TechSpec projection rules **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `agent-capabilities-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, and a risk table focused on loader and network discovery seams
  - [x] Manual test cases exist for single-file TOML/JSON, directory TOML/JSON, mixed-layout rejection, basename mismatch, duplicate IDs, no-catalog behavior, brief discovery, rich discovery, unknown IDs, and oversized responses
  - [x] Regression-suite documents define smoke, targeted, and full execution lanes with explicit P0/P1 ordering
  - [x] Each P0/P1 test case names the exact task or TechSpec rule it is proving
- Integration tests:
  - [x] All generated artifacts land under `.compozy/tasks/agent-capabilities/qa/` and can be consumed directly by `/qa-execution`
  - [x] The regression suite covers runtime, router, and API-visible behavior rather than only parser-level checks
  - [x] Any bug report created during planning references the originating test case or documented discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-report` has been run explicitly and its artifacts are stored under `.compozy/tasks/agent-capabilities/qa/`
- Task_07 can begin execution without redefining scope, priorities, or output paths
