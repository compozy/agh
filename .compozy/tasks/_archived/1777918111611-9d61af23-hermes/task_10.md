---
status: completed
title: Hermes Hardening QA Plan and Regression Artifacts
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
---

# Task 10: Hermes Hardening QA Plan and Regression Artifacts

## Overview

Generate the reusable QA planning artifacts for Hermes hardening before live execution begins. This task creates the feature-level test plan, manual test cases, regression suites, and traceability matrix covering backend, CLI, API, `web/`, and `packages/site` surfaces changed by tasks 01-09.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, tasks 01-09, and changed docs before planning coverage
- ACTIVATE `/qa-report` with `qa-output-path=.compozy/tasks/hermes` before writing or revising any QA artifact
- KEEP THE SAME `qa-output-path` FOR `/qa-execution` - all planning and execution artifacts must live under `.compozy/tasks/hermes/qa/`
- FOCUS ON WHAT MUST BE PROVEN - persistence, lifecycle, automation, MCP auth, process registry, memory visibility, setup, environment, release, web, and docs all need explicit coverage
- DO NOT EXECUTE THE FLOWS IN THIS TASK - this is planning, prioritization, traceability, and artifact generation only
- GREENFIELD: avoid generic smoke plans; every P0/P1 case must prove a selected Hermes hardening invariant
</critical>

<requirements>
- MUST use the `/qa-report` skill with `qa-output-path=.compozy/tasks/hermes`
- MUST generate a feature-level QA plan under `.compozy/tasks/hermes/qa/test-plans/`
- MUST generate manual test cases covering backend persistence, observability, ACP lifecycle, automation, MCP auth, process registry, memory, CLI setup, environment, release, web, and site docs
- MUST produce at least one regression-suite document defining smoke, targeted, and full execution priorities for task_11
- MUST define artifact locations under `.compozy/tasks/hermes/qa/` for issues, screenshots, logs, and verification reporting
- MUST analyze and include required `web/` and `packages/site` verification caused by every implementation task
</requirements>

## Subtasks
- [x] 10.1 Activate `/qa-report` with `qa-output-path=.compozy/tasks/hermes`
- [x] 10.2 Write the feature-level QA plan with scope, risks, environments, and entry/exit criteria across backend, CLI, API, web, and docs
- [x] 10.3 Generate manual test cases for all selected Hermes hardening tracks and cross-track foundations
- [x] 10.4 Build regression-suite definitions with explicit P0/P1 ordering for `/qa-execution`
- [x] 10.5 Map every P0/P1 case to tasks 01-09, the TechSpec, or an ADR decision
- [x] 10.6 Analyze and document any required follow-up verification in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Use the TechSpec "Testing Approach" plus tasks 01-09. The QA plan should translate Hermes hardening into execution-ready evidence: what must be proven through backend stores, daemon runtime, CLI, HTTP/API/SSE, web settings/session surfaces, and public documentation.

Task 04 coverage must explicitly prove durable automation scheduler behavior: cursor advancement
before dispatch, restart after a claimed fire without duplicate dispatch, `skip_missed` boot
reconciliation, delivery-error reporting that does not corrupt scheduler state, CLI/API/web exposure
of scheduler diagnostics, and docs coverage for the operator-visible fields.

Task 06 coverage must explicitly prove shared tool process registry behavior: checkpoint-on-write
records for ACP agents, ACP terminals, environment terminals, hooks, extensions, and shared
subprocess helpers; boot reconciliation that validates PID start-time evidence before signaling;
stale-record cleanup that never kills an unrelated reused PID; scoped prompt cancellation that
interrupts only the active session turn's tool processes; direct process, terminal, hook, and
extension interrupt targeting; restart scenarios covering active local PIDs and remote terminal
records without local PID evidence; and site documentation coverage for operator-visible restart and
interrupt semantics.

Task 07 coverage must explicitly prove memory visibility behavior: `agh memory health` and
`GET /api/memory/health` return consistent typed configured, degraded, and unavailable states;
`agh memory history` and `GET /api/memory/history` return bounded redacted operation history with
workspace, scope, operation, and time filters; history survives daemon restart through the durable
catalog operation log; `web/src/generated/agh-openapi.d.ts` exposes the new typed endpoints; site
memory CLI/API docs describe health/history; and runtime prompt assembly remains unchanged by the
future context-ref/provider-hook interfaces.

Task 09 coverage must explicitly prove environment, extension, and release hardening behavior:
`agh config validate --repair-env` repairs only bounded structured `.env` issues in temp workspaces,
refuses unsupported lines, symlinks, and directories without rewriting user-owned files, and never
prints secret values; extension manifests accept valid `requires_env`, reject invalid or duplicate
environment names, and surface `requires_env` plus `missing_env` consistently through CLI JSON/human
output, HTTP/API payloads, the generated TypeScript contract, and the settings page without leaking
values; release QA validates the GoReleaser Homebrew cask and nFPM `deb`/`rpm` targets while
preserving checksum signing and archive/source/package SBOM coverage; and site docs cover config
repair, extension environment requirements, status diagnostics, and package install trust artifacts.

### Relevant Files
- `.agents/skills/qa-report/SKILL.md` - required workflow, output layout, and artifact naming rules for QA planning
- `.compozy/tasks/hermes/_techspec.md` - authoritative source for selected issue scope and technical decisions
- `.compozy/tasks/hermes/adrs/` - accepted architecture decisions for hardening tracks
- `.compozy/tasks/hermes/task_01.md` through `.compozy/tasks/hermes/task_09.md` - implementation task requirements and verification scope
- `web/AGENTS.md` - frontend verification requirements if web artifacts are planned
- `packages/site/` - docs surfaces that require QA coverage

### Dependent Files
- `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-test-plan.md` - feature-level QA plan created by this task
- `.compozy/tasks/hermes/qa/test-plans/*-regression.md` - regression-suite document(s) consumed by task_11
- `.compozy/tasks/hermes/qa/test-cases/TC-*.md` - manual test cases with priorities, preconditions, and expected results
- `.compozy/tasks/hermes/qa/issues/BUG-*.md` - only if planning uncovers a concrete discrepancy while documenting coverage
- `.compozy/tasks/hermes/qa/screenshots/` - reserved output path for browser/doc evidence used by `/qa-execution`
- `.compozy/tasks/hermes/task_11.md` - execution task that must consume this artifact set without changing the output path

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - QA planning must cover all selected hardening tracks
- [ADR-002: Durable Automation Scheduler State](adrs/adr-002-durable-automation-scheduler-state.md) - QA planning must prove scheduler durability and duplicate prevention
- [ADR-003: MCP OAuth Auth Subsystem](adrs/adr-003-mcp-oauth-auth-subsystem.md) - QA planning must prove auth lifecycle and redaction
- [ADR-004: Shared Process Registry and Interrupt Runtime](adrs/adr-004-shared-process-registry-and-interrupt-runtime.md) - QA planning must prove process ownership and scoped interrupts
- [ADR-005: Memory Health and History Before Runtime Context References](adrs/adr-005-memory-health-history-before-runtime-contextrefs.md) - QA planning must prove visibility without prompt integration

## Deliverables
- `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-test-plan.md`
- One or more `.compozy/tasks/hermes/qa/test-plans/*-regression.md` documents reusable by `/qa-execution`
- Hermes hardening manual test cases under `.compozy/tasks/hermes/qa/test-cases/` **(REQUIRED)**
- Explicit P0/P1 coverage for backend, CLI, API/SSE, web, and `packages/site` seams changed by tasks 01-09 **(REQUIRED)**
- Stable artifact layout under `.compozy/tasks/hermes/qa/` that task_11 can consume without path changes **(REQUIRED)**
- Traceability from each P0/P1 case back to tasks 01-09, the TechSpec, or ADR rules **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `hermes-hardening-test-plan.md` includes objectives, scope, environment matrix, entry/exit criteria, and a risk table spanning backend, CLI, API, web, and site docs
  - [x] Manual test cases exist for each selected hardening track and shared foundation
  - [x] Regression-suite documents define smoke, targeted, and full execution lanes with explicit P0/P1 ordering
  - [x] Each P0/P1 test case names the exact task, TechSpec rule, issue, or ADR it is proving
- Integration tests:
  - [x] All generated artifacts land under `.compozy/tasks/hermes/qa/` and can be consumed directly by `/qa-execution`
  - [x] The regression suite covers backend, frontend, and documentation-visible behavior rather than only parser-level checks
  - [x] Any bug report created during planning references the originating test case or documented discrepancy clearly
- Test coverage target: >=80%
- All tests must pass

## Completion Notes

- Activated the `qa-report` workflow with `qa-output-path=.compozy/tasks/hermes` and created all planning artifacts under `.compozy/tasks/hermes/qa/`.
- Added the feature QA plan: `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-test-plan.md`.
- Added the task_11 regression suite: `.compozy/tasks/hermes/qa/test-plans/hermes-hardening-regression.md`.
- Added 15 manual test cases under `.compozy/tasks/hermes/qa/test-cases/`, covering persistence, observability, ACP lifecycle, automation, MCP auth, symlink security, process registry, memory, CLI setup, environment/extensions, release packaging, web contracts/UI, and site docs.
- Reserved `.compozy/tasks/hermes/qa/issues/`, `screenshots/`, and `logs/` for task_11 evidence with stable paths.
- Documented required `web/` and `packages/site` verification by implementation task in the QA plan and regression suite.

## Verification Evidence

- Artifact validation:
  - `find .compozy/tasks/hermes/qa -type f -print | sort`
  - `find .compozy/tasks/hermes/qa/test-cases -maxdepth 1 -type f -name 'TC-*.md' | wc -l` returned `15`
  - `rg --files-without-match "### Traceability" .compozy/tasks/hermes/qa/test-cases/TC-*.md` returned no files
  - `rg --files-without-match "\\*\\*Expected:\\*\\*" .compozy/tasks/hermes/qa/test-cases/TC-*.md` returned no files
  - `rg --files-without-match "TechSpec:" .compozy/tasks/hermes/qa/test-cases/TC-*.md` returned no files
- Full verification:
  - `make verify` passed. Key output: web oxlint `Found 0 warnings and 0 errors`, Go lint `0 issues`, Go tests `DONE 5851 tests in 15.822s`, and `OK: all package boundaries respected`.
- Post-commit verification:
  - Commit `92adb526 test: add hermes hardening qa artifacts` created the QA artifact set.
  - `make verify` passed after the commit. Key output: web oxlint `Found 0 warnings and 0 errors`, Go lint `0 issues`, Go tests `DONE 5851 tests in 6.086s`, and `OK: all package boundaries respected`.

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-report` has been run explicitly and its artifacts are stored under `.compozy/tasks/hermes/qa/`
- Task_11 can begin execution without redefining scope, priorities, or output paths
