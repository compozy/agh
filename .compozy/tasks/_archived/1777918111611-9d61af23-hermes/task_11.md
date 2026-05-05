---
status: completed
title: Hermes Hardening QA Execution and End-to-End Validation
type: test
complexity: critical
dependencies:
  - task_10
---

# Task 11: Hermes Hardening QA Execution and End-to-End Validation

## Overview

Execute the full QA pass for Hermes hardening using the artifacts from task_10, then commit durable regression coverage in the repository verification lanes. This task is the final quality gate for the selected issues: it must validate real backend, CLI, API, frontend, and documentation flows, fix root-cause regressions, and leave fresh evidence under the shared QA artifact layout.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and the QA artifacts from task_10 before running any validation
- ACTIVATE `/qa-execution` with `qa-output-path=.compozy/tasks/hermes` before any live verification or evidence capture
- IF QA FINDS A BUG, ACTIVATE `/systematic-debugging` AND `/no-workarounds` BEFORE CHANGING CODE OR TESTS
- FOLLOW THE PROJECT QA CONTRACT - use repository-defined gates and real runtime/API/UI/doc flows as final proof, not one-off scripts
- DO NOT WEAKEN TESTS TO GET GREEN - fix production code or configuration at the source, then rerun the narrow reproduction and full gates
- GREENFIELD: the final QA pass must prove the selected Hermes gaps are closed without legacy compatibility paths
</critical>

<requirements>
- MUST use the `/qa-execution` skill with `qa-output-path=.compozy/tasks/hermes`
- MUST consume `.compozy/tasks/hermes/qa/test-plans/` and `.compozy/tasks/hermes/qa/test-cases/` from task_10 as the execution matrix seed
- MUST execute the repository verification contract plus real backend/runtime, CLI, API/SSE, frontend, and site-doc scenarios
- MUST capture fresh QA evidence in `.compozy/tasks/hermes/qa/verification-report.md` and store issue files, logs, or screenshots under the same artifact root when applicable
- MUST fix root-cause regressions and add or update the narrowest durable regression coverage for every discovered bug
- MUST rerun the repository verification gates after the last fix, including relevant web and site lanes touched by Hermes hardening
- MUST analyze and implement any required follow-up changes in `web/` and `packages/site` discovered during execution
</requirements>

## Subtasks
- [x] 11.1 Activate `/qa-execution` with `qa-output-path=.compozy/tasks/hermes` and derive the execution matrix from task_10 artifacts
- [x] 11.2 Run baseline repository verification gates and establish the pre-execution health state
- [x] 11.3 Execute real backend, CLI, API/SSE, frontend, and docs scenarios for selected Hermes hardening tracks
- [x] 11.4 Fix root-cause regressions, add matching regression coverage, and rerun impacted scenarios
- [x] 11.5 Rerun final verification gates and publish `.compozy/tasks/hermes/qa/verification-report.md`
- [x] 11.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

Use the TechSpec "Testing Approach" plus the QA artifacts from task_10. The execution pass must prove behavior through real seams: database boot and migration, observe retention, ACP failure diagnostics, automation restart safety, MCP OAuth, process registry recovery, memory CLI/API visibility, setup lifecycle commands, environment repair, release config validation, web contract rendering, and site documentation consistency.

### Relevant Files
- `.agents/skills/qa-execution/SKILL.md` - required workflow for execution matrix discovery, evidence capture, and verification reporting
- `.agents/skills/qa-execution/scripts/discover-project-contract.py` - canonical project-contract discovery entrypoint required by `/qa-execution`
- `Makefile` - repository-defined verification gate that must be rerun after the last fix
- `web/AGENTS.md` - frontend verification constraints for web lint, typecheck, and route/system expectations
- `.compozy/tasks/hermes/qa/test-plans/` - task_10 artifacts that seed execution priorities and evidence expectations
- `.compozy/tasks/hermes/qa/test-cases/` - manual cases that define exact backend, CLI, web, and docs flows to run
- `packages/site/` - documentation build and content validation surface

### Dependent Files
- `.compozy/tasks/hermes/qa/verification-report.md` - final QA evidence produced by `/qa-execution`
- `.compozy/tasks/hermes/qa/issues/BUG-*.md` - structured bug reports for failures discovered during execution
- `.compozy/tasks/hermes/qa/screenshots/` - browser or visual evidence captured during execution
- `.compozy/tasks/hermes/qa/logs/` - command, daemon, or integration logs captured as evidence
- `internal/**` - root-cause fix destination for backend/runtime bugs discovered by QA
- `web/src/**` - root-cause fix destination for app regressions discovered by QA
- `packages/site/**` - root-cause fix destination for docs or site regressions discovered by QA

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - QA execution must prove all selected tracks
- [ADR-002: Durable Automation Scheduler State](adrs/adr-002-durable-automation-scheduler-state.md) - QA execution must prove durable scheduler invariants
- [ADR-003: MCP OAuth Auth Subsystem](adrs/adr-003-mcp-oauth-auth-subsystem.md) - QA execution must prove auth lifecycle and redaction
- [ADR-004: Shared Process Registry and Interrupt Runtime](adrs/adr-004-shared-process-registry-and-interrupt-runtime.md) - QA execution must prove process registry and interrupt safety
- [ADR-005: Memory Health and History Before Runtime Context References](adrs/adr-005-memory-health-history-before-runtime-contextrefs.md) - QA execution must prove memory visibility scope

## Deliverables
- Fresh `.compozy/tasks/hermes/qa/verification-report.md` produced by `/qa-execution`
- QA evidence covering every selected Hermes hardening track and shared foundation **(REQUIRED)**
- Root-cause bug fixes plus matching regression tests for any issues discovered during execution **(REQUIRED)**
- Fresh issue files, logs, screenshots, and supplementary evidence under `.compozy/tasks/hermes/qa/` **(REQUIRED)**
- Passing repository verification gates after the final QA fix set **(REQUIRED)**
- Fresh evidence that backend, CLI, API/SSE, web, and `packages/site` flows were validated through real surfaced behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Any bug found in persistence, lifecycle, automation, auth, process registry, memory, config, or environment logic gains a narrow regression proving the exact failed invariant
  - [x] Any bug found in API or contract payloads gains a regression in backend contract or handler tests
  - [x] Any bug found in the web surface gains the narrowest route, hook, adapter, or story regression that proves the actual user-facing failure
  - [x] Any bug found in site docs gains a docs validation, link, build, or content regression where the repo supports it
- Integration tests:
  - [x] Real runtime flows prove durable boot, failure diagnostics, scheduler restart safety, MCP auth, process interrupts, and memory visibility end to end
  - [x] Real CLI flows prove setup/config/environment commands work against isolated temp homes and workspaces
  - [x] Real API/SSE flows prove typed contract changes remain coherent with frontend expectations
  - [x] Real `web/` flows prove impacted settings/session/automation surfaces render and behave correctly against updated contracts
  - [x] Final docs review proves `packages/site` documents the final behavior consistently
  - [x] `make verify` and required web/site verification gates pass from a clean rerun after the final QA fix set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `/qa-execution` has been run explicitly with artifacts stored under `.compozy/tasks/hermes/qa/`
- The selected Hermes issues have fresh end-to-end evidence across runtime, CLI, API/SSE, web, and docs

## Completion Evidence

- Final QA report: `.compozy/tasks/hermes/qa/verification-report.md`
- Bug reports: `.compozy/tasks/hermes/qa/issues/BUG-001-remote-mcp-toml-overlay.md` through `BUG-007-automation-edit-dialog-route-remount.md`
- Final repository gate: `.compozy/tasks/hermes/qa/logs/final/make-verify-final.log`
- Final integration gate: `.compozy/tasks/hermes/qa/logs/final/make-test-integration-after-fixes.log`
- Final runtime E2E gate: `.compozy/tasks/hermes/qa/logs/final/make-test-e2e-runtime.log`
- Final web E2E gate: `.compozy/tasks/hermes/qa/logs/final/make-test-e2e-web-after-fix.log`
- Final web lint/typecheck/test: `.compozy/tasks/hermes/qa/logs/final/make-web-lint-after-fix.log`, `.compozy/tasks/hermes/qa/logs/final/make-web-typecheck-after-fix.log`, `.compozy/tasks/hermes/qa/logs/final/make-web-test-after-fix.log`
- Final site evidence: `.compozy/tasks/hermes/qa/logs/TC-REG-002/site-test-after-fix-2.log`, `.compozy/tasks/hermes/qa/logs/TC-REG-002/site-typecheck-after-fix.log`, `.compozy/tasks/hermes/qa/logs/TC-REG-002/site-build.log`, `.compozy/tasks/hermes/qa/logs/TC-REG-002/playwright-site-docs-final.log`
