---
status: pending
title: Autonomy MVP QA Execution And End-to-End Validation
type: test
complexity: critical
dependencies:
  - task_17
---

# Task 18: Autonomy MVP QA Execution And End-to-End Validation

## Overview
Execute the full QA pass for the autonomy MVP using the artifacts from task_17. This is the final quality gate: it must validate real backend, CLI/UDS, daemon lifecycle, web, and docs flows, fix root-cause regressions, and leave fresh evidence under `.compozy/tasks/autonomous/qa/`.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, tasks 01-17, and QA artifacts from task_17 before live validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/autonomous` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- USE REAL INTEGRATION AND END-TO-END FLOWS AS FINAL PROOF, not one-off mocks or parser-only checks
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source
- FINAL GATES MUST INCLUDE repository, generated contract, web, and site checks affected by autonomy
- NO WORKAROUNDS - every discovered failure must be fixed at root cause with durable regression coverage
</critical>

<requirements>
- MUST use `/qa-execution` with `qa-output-path=.compozy/tasks/autonomous`.
- MUST consume `.compozy/tasks/autonomous/qa/test-plans/` and `.compozy/tasks/autonomous/qa/test-cases/` from task_17.
- MUST execute repository verification plus real backend/store, daemon, CLI/UDS, hook, scheduler, spawn, coordinator, web, and docs scenarios.
- MUST capture fresh evidence in `.compozy/tasks/autonomous/qa/verification-report.md`.
- MUST store issue files, logs, screenshots, and supplementary evidence under `.compozy/tasks/autonomous/qa/` when applicable.
- MUST fix root-cause regressions and add durable regression coverage for every discovered bug.
- MUST rerun final verification gates after the last fix.
</requirements>

## Subtasks
- [ ] 18.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/autonomous` and derive execution matrix from task_17 artifacts.
- [ ] 18.2 Run baseline repository verification gates and record pre-execution health.
- [ ] 18.3 Execute real backend/store/daemon/CLI/UDS/hook/scheduler/spawn/coordinator scenarios from the QA matrix.
- [ ] 18.4 Execute web Tasks UI and `packages/site` docs verification lanes.
- [ ] 18.5 Fix root-cause regressions, add durable tests, and rerun narrow reproductions.
- [ ] 18.6 Rerun final gates and publish `.compozy/tasks/autonomous/qa/verification-report.md`.

## Implementation Details
Use the `/qa-execution` skill exactly as the Hermes hardening execution task does, but target autonomy MVP behavior. Evidence must show that the system handles the core workflow end to end:

1. User creates a task.
2. User starts/publishes/approves it.
3. A coordinator is bootstrapped for executable workspace work.
4. Work is claimed through `ClaimNextRun`.
5. Leases heartbeat and fence mutations correctly.
6. Coordinator delegates through safe spawn when allowed.
7. Scheduler recovers abandoned/expired work without claiming it.
8. Web and docs accurately describe manual-first autonomy.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required execution workflow.
- `.agents/skills/qa-execution/scripts/discover-project-contract.py` - project contract discovery entrypoint.
- `.agents/skills/qa-execution/references/checklist.md` - execution checklist and evidence guidance.
- `Makefile` - repository verification gates including `make verify`.
- `web/AGENTS.md` - frontend verification requirements.
- `packages/site/` - docs verification surface.
- `.compozy/tasks/autonomous/qa/test-plans/` - task_17 execution matrix.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 manual test cases.
- `.compozy/tasks/hermes/task_11.md` - local pattern for QA execution task structure.
- `.resources/paperclip/cli/src/commands/heartbeat-run.ts` - reference for lease heartbeat validation.
- `.resources/hermes/environments/agent_loop.py` - reference for agent loop validation scenarios.
- `.resources/multica/e2e/issues.spec.ts` - reference for task/issue E2E execution style.

### Dependent Files
- `.compozy/tasks/autonomous/qa/verification-report.md` - final QA evidence.
- `.compozy/tasks/autonomous/qa/issues/BUG-*.md` - bug reports for failures discovered during execution.
- `.compozy/tasks/autonomous/qa/screenshots/` - browser or docs screenshots when applicable.
- `.compozy/tasks/autonomous/qa/logs/` - command, daemon, test, and integration logs.
- `internal/**` - root-cause fix destination for runtime bugs discovered by QA.
- `web/src/**` - root-cause fix destination for UI regressions discovered by QA.
- `packages/site/**` - root-cause fix destination for docs/site regressions discovered by QA.

### Related ADRs
- [ADR-001: Phased Autonomy Kernel Scope](adrs/adr-001.md) - MVP scope validation.
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - CLI execution validation.
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - claim/lease validation.
- [ADR-004: Coordinator-Agent Plus Mechanical Scheduler](adrs/adr-004.md) - scheduler/coordinator validation.
- [ADR-005: Configurable Spawn-On-Run-Enqueue Coordinator](adrs/adr-005.md) - coordinator bootstrap validation.
- [ADR-006: Safe Spawn With Lineage And Permission Narrowing](adrs/adr-006.md) - spawn safety validation.
- [ADR-007: Minimal Network Evolution for Local Autonomy](adrs/adr-007.md) - channel validation.
- [ADR-008: Memory And Self-Correction MVP Boundary](adrs/adr-008.md) - post-MVP exclusion validation.
- [ADR-009: Autonomy Hooks And Extension Contracts](adrs/adr-009.md) - hook validation.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - manual coexistence validation.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - contract/web/docs validation.

## Deliverables
- Fresh `.compozy/tasks/autonomous/qa/verification-report.md` produced by `/qa-execution`.
- QA evidence covering every autonomy MVP track and cross-track invariant **(REQUIRED)**.
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**.
- Issue files, logs, screenshots, and supplementary evidence under `.compozy/tasks/autonomous/qa/` **(REQUIRED)**.
- Passing repository, generated contract, web, and site verification gates after the final fix set **(REQUIRED)**.

## Tests
- Unit/regression tests:
  - [ ] Any bug found in config, hooks, task lease, scheduler, spawn, coordinator, or contract logic gains a focused regression.
  - [ ] Any bug found in CLI/UDS payload handling gains handler or command tests.
  - [ ] Any bug found in web Tasks UI gains adapter, hook, component, or Playwright regression coverage.
  - [ ] Any bug found in docs/site gains link, build, typecheck, or content validation coverage where supported.
  - [ ] Tests are strengthened to prove correct behavior; failing tests are not weakened to match broken production code.
- Integration/end-to-end tests:
  - [ ] Real SQLite/store flows prove claim token redaction, capability matching, lease heartbeat, stale-token rejection, and expired-lease recovery.
  - [ ] Real daemon/CLI/UDS flows prove `agh me`, `agh ch`, `agh task`, and `agh spawn` commands against isolated homes/workspaces.
  - [ ] Real coordinator flow proves task creation does not start work, start/approval enqueues work, and one workspace coordinator is bootstrapped.
  - [ ] Real scheduler flow proves wake/sweep/recovery without direct claiming.
  - [ ] Real web flow proves task labels/actions match manual-first lifecycle.
  - [ ] Real site/docs flow proves runtime autonomy and CLI docs are current.
  - [ ] `make verify` plus required generated contract, web, and site gates pass after the final fix set.
- Test coverage target: >=80% for changed packages/files and full P0/P1 QA matrix coverage.
- All tests must pass.

## Success Criteria
- `/qa-execution` has been run explicitly with `qa-output-path=.compozy/tasks/autonomous`.
- All required verification gates pass after the final fix set.
- The autonomy MVP has fresh end-to-end evidence across runtime, CLI/UDS, hooks, scheduler, spawn, coordinator, web, and docs.
- Any discovered bugs are fixed at root cause with durable tests and documented evidence.
