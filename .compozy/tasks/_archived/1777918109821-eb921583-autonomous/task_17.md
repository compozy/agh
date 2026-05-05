---
status: completed
title: Autonomy MVP QA Plan And Regression Artifacts
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
  - task_09
  - task_10
  - task_11
  - task_12
  - task_13
  - task_14
  - task_15
  - task_16
---

# Task 17: Autonomy MVP QA Plan And Regression Artifacts

## Overview
Generate the reusable QA planning artifacts for the autonomy MVP before live execution begins. This mirrors the Hermes QA handoff pattern: create a feature-level test plan, regression suites, manual test cases, and traceability matrix under `.compozy/tasks/autonomous/qa/`.

<critical>
- ALWAYS READ `_techspec.md`, ADRs 001-012, tasks 01-16, and changed docs before planning coverage
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/autonomous` before writing or revising QA artifacts
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/autonomous/qa/`
- DO NOT EXECUTE THE FLOWS IN THIS TASK - this is planning, prioritization, traceability, and artifact generation only
- COVER REAL RISK - config, hooks, situation context, identity, coordination channels, lease fencing, scheduler, spawn, coordinator, web, and docs all need explicit P0/P1 coverage
- NO WORKAROUNDS - do not create generic smoke plans that fail to prove the accepted autonomy invariants
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/autonomous`.
- MUST generate a feature-level QA plan under `.compozy/tasks/autonomous/qa/test-plans/`.
- MUST generate manual test cases covering backend, SQLite stores, daemon lifecycle, CLI/UDS, hooks, task leases, coordination channels, scheduler, spawn, coordinator, web Tasks UI, and `packages/site` docs.
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution lanes for task_18.
- MUST define artifact locations under `.compozy/tasks/autonomous/qa/` for issues, screenshots, logs, and verification reporting.
- MUST map every P0/P1 test case to tasks 01-16, the TechSpec, or ADR decisions.
</requirements>

## Subtasks
- [x] 17.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/autonomous`.
- [x] 17.2 Write the feature-level QA plan with scope, risks, environments, entry/exit criteria, and traceability across runtime, web, and docs.
- [x] 17.3 Generate manual test cases for all autonomy MVP tracks and cross-track invariants.
- [x] 17.4 Build regression-suite definitions with smoke, targeted, and full execution lanes for task_18.
- [x] 17.5 Map each P0/P1 case to task files, TechSpec sections, ADRs, or resource-reference lessons.
- [x] 17.6 Confirm the QA artifact layout is stable and ready for `/qa-execution`.

## Implementation Details
Use `.compozy/tasks/hermes/task_10.md` and the Hermes QA artifact layout as the local pattern, but tailor every case to autonomy MVP invariants. The output should help a future executor prove the real system, not just parse docs.

The QA plan must include coverage for references that influenced implementation:
- Paperclip task/issue run orchestration, heartbeat, and agent management plans.
- Hermes runner, scheduler, trajectory, and auxiliary agent references.
- Multica issue/autopilot/inbox/E2E references.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required QA planning workflow.
- `.agents/skills/qa-report/references/test_case_templates.md` - manual test case format guidance.
- `.agents/skills/qa-report/references/regression_testing.md` - regression-suite guidance.
- `.compozy/tasks/autonomous/_techspec.md` - authoritative autonomy MVP behavior.
- `.compozy/tasks/autonomous/adrs/` - accepted architecture decisions.
- `.compozy/tasks/autonomous/task_01.md` through `.compozy/tasks/autonomous/task_16.md` - implementation task requirements.
- `.compozy/tasks/hermes/task_10.md` - local pattern for QA planning task structure.
- `.resources/paperclip/doc/plans/2026-02-20-issue-run-orchestration-plan.md` - reference for run orchestration cases.
- `.resources/paperclip/cli/src/commands/heartbeat-run.ts` - reference for heartbeat/lease case design.
- `.resources/hermes/cron/scheduler.py` - reference for scheduler recovery cases.
- `.resources/hermes/agent/auxiliary_client.py` - reference for delegated agent cases.
- `.resources/multica/e2e/issues.spec.ts` - reference for issue/task E2E case style.

### Dependent Files
- `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-test-plan.md` - feature-level QA plan created by this task.
- `.compozy/tasks/autonomous/qa/test-plans/*-regression.md` - regression suites consumed by task_18.
- `.compozy/tasks/autonomous/qa/test-cases/TC-*.md` - manual test cases with priorities and expected results.
- `.compozy/tasks/autonomous/qa/issues/BUG-*.md` - issue files if planning uncovers concrete discrepancies.
- `.compozy/tasks/autonomous/qa/screenshots/` - reserved for task_18 browser/docs evidence.
- `.compozy/tasks/autonomous/qa/logs/` - reserved for task_18 command/runtime logs.

### Related ADRs
- [ADR-001: Phased Autonomy Kernel Scope](adrs/adr-001.md) - QA scope boundary.
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - CLI coverage.
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - lease coverage.
- [ADR-004: Coordinator-Agent Plus Mechanical Scheduler](adrs/adr-004.md) - scheduler/coordinator coverage.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - coordinator config coverage.
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - spawn coverage.
- [ADR-007: Minimal Network Evolution for Local Autonomy](adrs/adr-007.md) - channel coverage.
- [ADR-008: Memory And Self-Correction MVP Boundary](adrs/adr-008.md) - post-MVP boundary checks.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - hook coverage.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual coexistence coverage.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - web/docs/contract coverage.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - coordination channel coverage.

## Deliverables
- `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-test-plan.md`.
- One or more `.compozy/tasks/autonomous/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`.
- Manual test cases under `.compozy/tasks/autonomous/qa/test-cases/` **(REQUIRED)**.
- P0/P1 traceability across backend, CLI/UDS, API contracts, web, docs, and daemon lifecycle **(REQUIRED)**.
- Stable artifact layout under `.compozy/tasks/autonomous/qa/` that task_18 can consume without path changes **(REQUIRED)**.

## Tests
- Artifact validation:
  - [ ] Feature QA plan includes objectives, scope, risk table, environment matrix, entry criteria, and exit criteria.
  - [ ] Manual test cases exist for each autonomy track: config, hooks, situation, identity, coordination channels, claim/lease, scheduler, spawn, coordinator, UI, and docs.
  - [ ] P0 cases prove coordinated run enqueue binds a channel and that channel messages cannot mutate task-run ownership/status.
  - [ ] Regression suite defines smoke, targeted, and full lanes with explicit P0/P1 ordering.
  - [ ] Every P0/P1 case includes expected results and traceability to a task, TechSpec section, ADR, or resource-reference lesson.
  - [ ] Planning artifacts avoid broad post-MVP scope unless explicitly marked out-of-scope.
- Integration/artifact tests:
  - [ ] All generated artifacts live under `.compozy/tasks/autonomous/qa/`.
  - [ ] Task_18 can consume the regression suite without redefining output paths or priorities.
  - [ ] Any planning-discovered discrepancy is written as a structured issue file with source traceability.
  - [ ] The plan includes required `web/` and `packages/site` verification lanes.
- Test coverage target: planning coverage must trace every task_01 through task_16 P0/P1 invariant.
- All artifact checks must pass.

## Success Criteria
- `/qa-report` has been run explicitly with `qa-output-path=.compozy/tasks/autonomous`.
- Task_18 can begin execution without redefining QA scope, priorities, or output paths.
- QA artifacts prove the autonomy MVP through real runtime, CLI, web, and docs behavior rather than superficial smoke checks.

## Completion Notes
- Created `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-test-plan.md` with objectives, scope, artifact layout, environment matrix, risk table, entry/exit criteria, and runtime/web/docs traceability.
- Created `.compozy/tasks/autonomous/qa/test-plans/autonomy-mvp-regression.md` with smoke, targeted, and full regression lanes for task_18 using explicit P0/P1 ordering and evidence destinations.
- Created 18 manual cases under `.compozy/tasks/autonomous/qa/test-cases/` covering config, contracts, hooks, situation context, identity, coordination channels, SQLite leases, task lease CLI/UDS, execution boundary, scheduler, lineage, spawn, coordinator, web Tasks UI, docs, E2E, and post-MVP boundary checks.
- No planning-discovered concrete discrepancies were found, so no `BUG-*` issue files were created.

## Verification Evidence
- `find .compozy/tasks/autonomous/qa -maxdepth 3 -type f -print | sort` confirmed all generated artifacts live under `.compozy/tasks/autonomous/qa/`.
- `find .compozy/tasks/autonomous/qa/test-cases -maxdepth 1 -type f -name 'TC-*.md' | wc -l` returned `18`.
- `rg --files-without-match '### Traceability'`, `rg --files-without-match '\*\*Expected:\*\*'`, and `rg --files-without-match 'TechSpec:'` over `TC-*.md` returned no missing-case output.
- Artifact coverage checks confirmed the required autonomy tracks, task_01 through task_16 references, P0 channel-binding coverage, and P0 channel non-authority coverage.
- `git diff --cached --check` passed after mechanical Markdown whitespace cleanup.
- `make verify` passed after final artifact cleanup.
