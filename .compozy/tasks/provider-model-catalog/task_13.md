---
status: completed
title: "Real-Scenario QA Execution"
type: test
complexity: critical
dependencies:
    - task_12
---

# Task 13: Real-Scenario QA Execution

## Overview
This task executes the QA plan against an isolated, production-like AGH runtime. It verifies the provider model catalog end-to-end across daemon, config, SQLite, ACP, HTTP, UDS, CLI, extensions, web, generated docs/types, and browser workflows.

<critical>
- ALWAYS READ `qa/test-plans/*`, `qa/test-cases/*`, `_techspec.md`, and all ADRs before executing.
- REFERENCE TECHSPEC for implementation details - do not duplicate here.
- FOCUS ON BEHAVIOR - verify runtime behavior and fix root causes for reproduced defects.
- NO WORKAROUNDS - do not weaken tests to make the suite green.
- TESTS REQUIRED - every fixed bug must include regression coverage.
</critical>

<requirements>
- MUST activate `agh-qa-bootstrap`, `real-scenario-qa`, `qa-execution`, and `agh-worktree-isolation`.
- MUST use unique `AGH_HOME`, daemon ports, and tmux-bridge socket paths for the QA run.
- MUST run CLI/API/agent-manageability E2E checks against a daemon-served runtime.
- MUST run browser E2E for the highest-risk UI workflows via `browser-use:browser`, with `agent-browser` fallback only if unavailable.
- MUST run `make test-e2e-runtime` and `make test-e2e-web` for this UI-bearing feature.
- MUST file `qa/issues/BUG-NNN.md` for each reproduced defect and fix root causes before final verification.
- MUST produce `qa/verification-report.md` with manifest path, lab root, runtime home, base URL, commands, results, and residual risk.
</requirements>

## Subtasks
- [x] 13.1 Bootstrap an isolated QA lab with unique runtime home, ports, and manifest.
- [x] 13.2 Execute planned unit/integration/codegen/web/docs gates from Task 12.
- [x] 13.3 Execute daemon-served CLI, HTTP, UDS, `/api/openai/v1/models`, and Host API parity scenarios.
- [x] 13.4 Execute browser E2E workflows for provider settings, catalog refresh/status, manual model entry, and new session selection.
- [ ] 13.5 File and fix reproduced bugs with regression tests.
- [x] 13.6 Produce the final verification report and run `make verify`.

## QA Execution Result - 2026-05-07

- Fresh isolated lab created:
  - Manifest: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/bootstrap-manifest.json`
  - Lab root: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab`
  - Runtime home: `/var/folders/7x/xg204hnd04b81fczcxvjlhzr0000gn/T/aghqa-6a6b0c93875c/runtime`
  - Base URL / `AGH_WEB_API_PROXY_TARGET`: `http://127.0.0.1:62444`
- Verification report written: `.compozy/tasks/provider-model-catalog/qa/verification-report.md`.
- Final `make verify` passed on rerun: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/logs/final-make-verify-rerun.log`.
- Real-scenario audit is blocked as expected for release-grade proof because no live provider-backed session evidence exists: `/Users/pedronauck/dev/qa-labs/agh-provider-model-catalog-20260507-133813-330549-lab/qa-artifacts/qa/qa-audit-report.md`.
- Cleared/fixed:
  - BUG-001 generated OpenAPI/CLI docs drift; no generated artifact diff remains.
  - BUG-003 runtime E2E ACP helper interface drift.
  - BUG-004 web E2E provider selector contract drift.
- Still open, so Task 13 remains pending:
  - BUG-002 `/api/openai/v1/models` returns catalog data without bearer auth.
  - BUG-005 workspace-scoped provider model metadata is not projected into the daemon-backed session catalog.
  - Live provider-backed ACP session evidence is unavailable in this isolated lab.

## Implementation Details
Use Task 12 QA artifacts as the execution contract. Follow AGH QA bootstrap rules: fresh lab by default, unique ports/home, no default `~/.agh`, and `AGH_WEB_API_PROXY_TARGET` derived from the bootstrap manifest for isolated web QA.

### Relevant Files
- `.compozy/tasks/provider-model-catalog/qa/test-plans/` - QA plan.
- `.compozy/tasks/provider-model-catalog/qa/test-cases/` - concrete test cases.
- `.agents/skills/agh-qa-bootstrap/SKILL.md` - isolated lab setup.
- `.agents/skills/real-scenario-qa/SKILL.md` - release-grade QA execution.
- `.agents/skills/qa-execution/SKILL.md` - behavior-first QA mechanics.
- `.agents/skills/agh-worktree-isolation/SKILL.md` - unique runtime home/ports/socket requirements.
- `web/e2e/` - browser E2E tests and fixtures.
- `internal/testutil/e2e/` - daemon runtime harness tests.

### Dependent Files
- `.compozy/tasks/provider-model-catalog/qa/issues/BUG-NNN.md` - bug reports for reproduced defects.
- `.compozy/tasks/provider-model-catalog/qa/verification-report.md` - final verification evidence.
- `bootstrap-manifest.json` - QA bootstrap output artifact path reported in final evidence.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - end-to-end catalog authority.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - old-key rejection and no compatibility.
- [ADR-003: Extension Model Source Contract](adrs/adr-003-extension-model-source-contract.md) - extension source QA.

## Deliverables
- Isolated QA bootstrap manifest and environment details.
- Completed QA execution with bug reports for reproduced defects.
- Regression tests for each fixed QA bug.
- `qa/verification-report.md` with command evidence and residual risk.
- Final `make verify` evidence **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] all unit test commands from Task 12 pass after any QA fixes.
  - [ ] regression tests exist for every reproduced BUG file.
- Integration tests:
  - [x] `make test-e2e-runtime` passes against an isolated runtime.
  - [x] `make test-e2e-web` passes with `AGH_WEB_API_PROXY_TARGET` from the bootstrap manifest.
  - [x] browser workflow validates Settings > Providers nested model config and new session catalog selection.
  - [x] CLI/HTTP/UDS parity scenario compares list/refresh/status for the same persisted state.
  - [ ] `/api/openai/v1/models` scenario verifies auth, HTTP-only registration, OpenAI-shaped errors, and provider filter.
  - [x] Host API extension model source success/denial scenario passes.
  - [x] `make verify` passes after implemented fixes; QA remains blocked by open BUG-002/BUG-005.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes.
- QA verification report includes bootstrap manifest path, lab root, runtime home, base URL, commands, results, bug links, and residual risk.
