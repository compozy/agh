---
status: completed
title: Harness QA plan and regression artifacts
type: docs
complexity: high
dependencies:
  - task_08
---

# Task 09: Harness QA plan and regression artifacts

## Overview

Generate the reusable QA planning artifacts for the harness architecture before live execution begins. This task must leave the feature with a concrete test plan, runtime-oriented manual test cases, and regression-suite definitions that the follow-up execution task can consume without re-deciding scope, evidence layout, or output paths.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `task_01.md` through `task_08.md` before planning coverage (`_prd.md` is absent; requirements come from the TechSpec and generated tasks)
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/harness` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/harness/qa/`
- FOCUS ON "WHAT" - define coverage, risks, and evidence expectations; do not execute runtime flows or preemptively fix bugs in this task
- PRIORITIZE DAEMON/RUNTIME FLOWS - startup prompting, augmentation, synthetic reentry, detached task-run completion, transcript trust, and observability all need explicit coverage
- GREENFIELD: nao aceitar smoke genrico; cada seam critica do harness precisa ter rastreabilidade explicita na matriz de QA
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/harness`
- MUST generate a feature-level QA plan under `.compozy/tasks/harness/qa/test-plans/`
- MUST generate manual test cases covering startup section selection, ordered augmentation, synthetic prompt submission, transcript/hook/extension support, detached task-runtime mapping, and task-run completion reentry
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for the follow-up `/qa-execution` task
- MUST include operator-observable evidence expectations for event summaries, transport parity, and runtime recovery scenarios
- SHOULD use the competitor references listed in this task as inspiration for scenario breadth when defining regression suites
</requirements>

## Subtasks
- [x] 9.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/harness`
- [x] 9.2 Write the feature-level harness QA plan with scope, risks, environments, and entry/exit criteria
- [x] 9.3 Generate runtime-oriented manual test cases with explicit expected results and edge conditions
- [x] 9.4 Build regression-suite definitions and identify the P0/P1 flows that `/qa-execution` must run first
- [x] 9.5 Validate artifact completeness, traceability, and handoff readiness for `task_10`

## Implementation Details

See TechSpec "Workstream 6: Storage, Observability, and Verification" and the verification expectations embedded in tasks 01-08. This task is the formal handoff from implementation to QA execution: it should define exactly what must be proven about runtime semantics, what evidence counts, and where that evidence will live.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required workflow, output structure, and artifact naming rules for QA planning
- `.compozy/tasks/harness/_techspec.md` - source of truth for harness workstreams, risks, and verification expectations
- `.compozy/tasks/harness/task_04.md` - synthetic prompt submission and session-event persistence scenarios must be covered explicitly
- `.compozy/tasks/harness/task_07.md` - task-run completion and synthetic reentry scenarios are central P0 coverage
- `.compozy/tasks/harness/task_08.md` - observability and integration-hardening expectations must be translated into QA artifacts

### Dependent Files
- `.compozy/tasks/harness/qa/test-plans/harness-test-plan.md` - feature-level QA plan created by this task
- `.compozy/tasks/harness/qa/test-plans/*-regression.md` - regression-suite document(s) consumed by the execution task
- `.compozy/tasks/harness/qa/test-cases/TC-*.md` - manual test cases with priorities and expected results
- `.compozy/tasks/harness/qa/issues/BUG-*.md` - only created if planning uncovers a concrete documented discrepancy
- `.compozy/tasks/harness/task_10.md` - execution task that must consume this artifact set unchanged

### Related ADRs
- [ADR-001: Resolve Harness Behavior from Durable Session Context and Turn Origin](adrs/adr-001.md) - QA planning must cover the context-resolution matrix explicitly
- [ADR-002: Extend Existing Prompt Assembly and Turn Augmentation Seams with Staged Composition](adrs/adr-002.md) - QA planning must cover startup and turn-time seams separately
- [ADR-003: Reuse the Task Runtime for Detached Harness Work and Policy-Based Synthetic Reentry](adrs/adr-003.md) - Detached completion and reentry are core P0 harness behaviors

### External References
- `.resources/openclaw/docs/concepts/qa-e2e-automation.md` - strong reference for structuring QA lanes and runtime-focused artifact sets
- `.resources/hermes/tests/integration/test_checkpoint_resumption.py` - useful regression inspiration for restart/resume integrity scenarios
- `.resources/openfang/docs/api-reference.md` - good source of ideas for externally inspectable async/eventful runtime behaviors
- `.resources/claude-code/utils/task/framework.ts` - useful reference for task lifecycle checkpoints worth covering in QA
- `.resources/claude-code/tasks/LocalMainSessionTask.ts` - good example of background-to-foreground completion behaviors that should become regression scenarios

## Deliverables
- `.compozy/tasks/harness/qa/test-plans/harness-test-plan.md`
- One or more `.compozy/tasks/harness/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Runtime-oriented manual test cases under `.compozy/tasks/harness/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 coverage for startup, augmentation, synthetic reentry, detached completion, and observability **(REQUIRED)**
- A stable artifact layout under `.compozy/tasks/harness/qa/` that the execution task can consume without path changes **(REQUIRED)**
- Explicit mapping from each P0/P1 case to one harness workstream or task file **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `harness-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, and risk assessment
  - [x] Manual test cases exist for startup selection, augmentation, synthetic turns, transcript trust, detached completion, and event-summary visibility
  - [x] Regression-suite documents identify smoke, targeted, and full coverage plus execution order for P0/P1 flows
  - [x] Each P0/P1 case names the exact harness task or TechSpec workstream it proves
- Integration tests:
  - [x] All generated artifacts land under `.compozy/tasks/harness/qa/` and can be consumed directly by `/qa-execution`
  - [x] Test cases trace back to the relevant harness tasks or TechSpec workstreams clearly enough to seed execution
  - [x] Any bug report created during planning references the originating test case or runtime discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- The `/qa-report` workflow has been executed explicitly and its artifacts are stored under `.compozy/tasks/harness/qa/`
- Every critical harness seam has at least one traceable QA artifact
- `task_10` can start execution without redefining scope, output paths, or risk priorities
- The harness feature has a concrete regression plan instead of ad hoc runtime test notes
