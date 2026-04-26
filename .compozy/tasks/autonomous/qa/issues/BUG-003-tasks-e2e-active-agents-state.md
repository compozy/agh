# BUG-003: Tasks browser E2E expected only fallback states for the Agents panel

## Status
- Fixed

## Severity
- P1 QA blocker

## Affected Lane
- `make test-e2e-web`
- `web/e2e/tasks.spec.ts`

## Evidence
- Failing log: `.compozy/tasks/autonomous/qa/logs/final-test-e2e-web.log`
- Error context showed the Agents tab rendering a valid active state: `1 running · 0 idle` with an `Open run` link for the newly published task.
- Narrow rerun after fix: `.compozy/tasks/autonomous/qa/logs/final-test-e2e-web-rerun-failing.log`

## Root Cause
The E2E assertion only accepted empty, no-active, or disconnected multi-agent states. After the autonomy execution boundary and coordinator handoff work, publishing the draft correctly enqueues an active run and the Agents tab renders the task-bound active run instead of an empty fallback.

## Fix
- Added stable Playwright selectors for the Tasks multi-agent panel, summary, and per-task run link.
- Updated `web/e2e/tasks.spec.ts` to assert the active Agents state and run drilldown link for the published task.

## Verification
- `bun run --cwd web typecheck`
- `AGH_E2E_QA_OUTPUT_DIR=../.compozy/tasks/autonomous bunx playwright test e2e/automation.spec.ts e2e/session-onboarding.spec.ts e2e/tasks.spec.ts --reporter=list`
