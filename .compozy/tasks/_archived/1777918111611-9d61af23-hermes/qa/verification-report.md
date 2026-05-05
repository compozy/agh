# Hermes Hardening QA Verification Report

Task: `task_11.md`  
Date: 2026-04-25  
QA output path: `.compozy/tasks/hermes`  
Execution workflow: `/qa-execution` with task 10 artifacts from `qa/test-plans/` and `qa/test-cases/`

## Verdict

PASS. Hermes hardening QA completed across backend/runtime, CLI, API/SSE, web, and `packages/site` documentation flows. Seven regressions were discovered during live execution or final gates, fixed at root cause, and covered with focused regression evidence. Final repository and web gates are green.

## Final Gates

| Gate | Result | Evidence |
| --- | --- | --- |
| Contract discovery | PASS | `qa/logs/contract-discovery/discover-project-contract.log` |
| Baseline deps | PASS | `qa/logs/baseline/make-deps.log` |
| Baseline repository verify | PASS | `qa/logs/baseline/make-verify-baseline.log` |
| Final repository verify | PASS, 5852 Go tests plus web lint/typecheck/test/build | `qa/logs/final/make-verify-final.log` |
| Final integration | PASS, 6333 tests, 3 Daytona credential skips | `qa/logs/final/make-test-integration-after-fixes.log` |
| Final runtime E2E | PASS | `qa/logs/final/make-test-e2e-runtime.log` |
| Final daemon-served web E2E | PASS, 15 tests | `qa/logs/final/make-test-e2e-web-after-fix.log` |
| Final web lint/typecheck/test | PASS, 1412 Vitest tests | `qa/logs/final/make-web-lint-after-fix.log`, `qa/logs/final/make-web-typecheck-after-fix.log`, `qa/logs/final/make-web-test-after-fix.log` |
| Final site docs build/test/typecheck/browser review | PASS | `qa/logs/TC-REG-002/site-test-after-fix-2.log`, `qa/logs/TC-REG-002/site-typecheck-after-fix.log`, `qa/logs/TC-REG-002/site-build.log`, `qa/logs/TC-REG-002/playwright-site-docs-final.log` |

Notes:

- `make verify` is the repository's blocking verification contract. No separate coverage target is defined in the discovered Makefile/Mage contract; this report records the repository-defined gates plus focused regression and E2E lanes.
- Daytona integration tests were skipped because `DAYTONA_API_KEY` is not present; this is an explicit credentialed-provider skip, not a failing Hermes lane.
- Vite emitted the existing chunk-size warning during build. It is non-blocking and unchanged by Task 11.
- The site static-server log contains a Python `BrokenPipeError` during server shutdown after successful browser validation; no product request failed.
- Browser Use could not be used in this runtime because the installed plugin entrypoint imports Node builtins statically in a form unsupported by the available `js_repl`. Playwright was used for browser evidence.

## Execution Matrix

| Case | Track | Result | Evidence |
| --- | --- | --- | --- |
| TC-INT-001 | Database boot and migration foundations | PASS | `qa/logs/TC-INT-001/`, `go-test-store-retry.log`, `schema-migrations-global.log`, `global-db-tables.log` |
| TC-INT-002 | Observe retention and typed health | PASS | `qa/logs/TC-INT-002/go-test-observe-health.log`, `observe-health.json` |
| TC-INT-003 | ACP failure diagnostics and session lifecycle | PASS | `qa/logs/TC-INT-003/`, including `session-event.json`, `http-session.json`, `observe-health-failures.json`, `crash-bundle-excerpt.json` |
| TC-INT-004 | Durable automation scheduler restart safety | PASS | `qa/logs/TC-INT-004/`, including `job-before-restart.json`, `job-after-real-restart.json`, `observe-health-automation.json` |
| TC-INT-005 | Process registry and interrupt runtime | PASS | `qa/logs/TC-INT-005/go-test-toolruntime-process-registry.log` |
| TC-SEC-001 | MCP OAuth config/status/redaction | PASS after BUG-001 | `qa/logs/TC-SEC-001/`, `BUG-001-remote-mcp-toml-overlay.md` |
| TC-SEC-002 | Skill and managed-extension symlink hardening | PASS | `qa/logs/TC-SEC-002/go-test-symlink-security.log` |
| TC-FUNC-001 | Memory CLI/API visibility and redaction | PASS after BUG-002 | `qa/logs/TC-FUNC-001/`, `BUG-002-memory-operation-log-schema.md` |
| TC-FUNC-002 | Setup/config lifecycle commands | PASS | `qa/logs/TC-FUNC-002/` |
| TC-FUNC-003 | Environment repair and extension diagnostics | PASS | `qa/logs/TC-FUNC-003/` |
| TC-REG-001 | Release config validation | PASS | `qa/logs/TC-REG-001/release-config-test.log`, `release-config-docs-inspection.log` |
| TC-REG-002 | Site documentation consistency | PASS after BUG-003 | `qa/logs/TC-REG-002/`, `qa/screenshots/TC-REG-002/`, `BUG-003-site-landing-test-drift.md` |
| TC-UI-001 | Real web UI against live daemon | PASS | `qa/logs/TC-UI-001/`, `qa/screenshots/TC-UI-001/` |
| TC-UI-002 | Focused settings/automation web regressions | PASS | `qa/logs/TC-UI-002/web-focused-settings-automation-tests.log` |
| TC-UI-003 | Web codegen/lint/typecheck/test gates | PASS | `qa/logs/TC-UI-003/` and final web gate logs |

## Issues Discovered and Fixed

| Issue | Root Cause | Fix | Regression Evidence |
| --- | --- | --- | --- |
| BUG-001: Remote MCP TOML overlays reject documented fields | TOML overlay decoder only accepted legacy stdio MCP fields | Added remote MCP `transport`, `url`, and `auth` overlay decode/merge support | `qa/issues/BUG-001-remote-mcp-toml-overlay.md` |
| BUG-002: Fresh daemon memory history schema misses scope columns | Initial global DB schema lacked the Task 07 `memory_operation_log` scope columns | Added migration v6 and schema assertions | `qa/issues/BUG-002-memory-operation-log-schema.md` |
| BUG-003: Site landing tests drifted from current landing copy | Site test expected old accessible heading/card count | Updated site test to current landing semantics | `qa/issues/BUG-003-site-landing-test-drift.md` |
| BUG-004: HTTP prompt disconnect cancels terminal event drain too early | HTTP request cancellation canceled the prompt context before detached drain | Defer prompt cancellation until detached drain completion/timeout | `qa/issues/BUG-004-http-prompt-drain-cancel.md` |
| BUG-005: CLI session-list TOON regression expected the pre-Hermes header | Integration assertion omitted current `failure_kind` field | Updated TOON header assertion to current contract | `qa/issues/BUG-005-cli-session-list-toon-header.md` |
| BUG-006: Reference extension E2E used symlinked SDK dependency | Reference fixture installed from a development source with a symlink escaping the hardened source root | Build a packaged temp install source with materialized SDK files | `qa/issues/BUG-006-reference-extension-sdk-symlink.md` |
| BUG-007: Automation edit dialog loses state when route motion key catches up | App route shell used mutable `router.latestLocation` as a render key and remounted Jobs on local dialog state change | Key route motion from reactive `useLocation` and add route-shell regression | `qa/issues/BUG-007-automation-edit-dialog-route-remount.md` |

## Durable Evidence Highlights

- Real isolated daemon homes were used for runtime/CLI/API flows. Evidence paths include `*.path` files for reproduced homes/workspaces and JSON outputs for daemon/API results.
- Redaction checks were performed for MCP auth and memory body content:
  - `qa/logs/TC-SEC-001/redaction-check.log`
  - `qa/logs/TC-FUNC-001/memory-redaction-check.log`
  - `qa/logs/TC-FUNC-003/env-redaction-check.log`
- Screenshots captured web UI and docs surfaces:
  - `qa/screenshots/TC-UI-001/automation-desktop.png`
  - `qa/screenshots/TC-UI-001/mcp-servers-desktop.png`
  - `qa/screenshots/TC-UI-001/memory-desktop.png`
  - `qa/screenshots/TC-REG-002/session-lifecycle-desktop.png`
  - `qa/screenshots/TC-REG-002/memory-system-mobile.png`
  - `qa/screenshots/TC-REG-002/installation-release-mobile.png`

## Follow-Up Work

No blocking Task 11 follow-up remains. Non-blocking observations:

- The web production build still emits an existing large-chunk warning. It is not a Hermes correctness regression.
- Credentialed Daytona integration coverage remains gated on `DAYTONA_API_KEY`.
