# BUG-001: Web E2E Workspace Onboarding Race And Stale Session Start Flow

## Status
Fixed and verified.

## Severity
P1 QA regression.

## Affected Lane
`TC-AUTO-015` Tasks UI manual-first coordinator handoff Playwright lane.

## Failure Evidence
- Initial failure log: `.compozy/tasks/autonomous/qa/logs/targeted/TC-AUTO-015/playwright-tasks-coordinator-handoff.log`
- First rerun log: `.compozy/tasks/autonomous/qa/logs/targeted/TC-AUTO-015/playwright-tasks-coordinator-handoff-rerun.log`
- Passing rerun log: `.compozy/tasks/autonomous/qa/logs/targeted/TC-AUTO-015/playwright-tasks-coordinator-handoff-rerun-2.log`
- Screenshots: `.compozy/tasks/autonomous/qa/screenshots/tasks-approval-handoff-enqueued.png`, `.compozy/tasks/autonomous/qa/screenshots/final.png`

## Root Cause
The daemon-served Playwright specs branched on `workspaceOnboarding.isVisible()` immediately after page navigation. When onboarding rendered just after that instant, tests skipped workspace registration and then waited for shell selectors (`nav-tasks`, `app-sidebar`) that could not exist while the app was still on onboarding.

After fixing that race, the Task 15 handoff spec exposed a stale manual-session flow. The test clicked the agent's new-session button and expected immediate navigation, but the current production UI opens the real session-create dialog and requires submitting `session-create-dialog-submit`.

## Fix
- Added `web/e2e/fixtures/workspace.ts` with `useGlobalWorkspaceIfPrompted`, which waits for either onboarding or the app shell, clicks "Use global workspace" only when prompted, and asserts the shell is visible.
- Replaced every immediate onboarding branch in daemon-served E2E specs with the shared helper.
- Updated `tasks-coordinator-handoff.spec.ts` to submit the real session-create dialog, wait for the `/api/sessions` response, and assert navigation to `/session/:id`.

## Verification
`cd web && AGH_E2E_QA_OUTPUT_DIR=../.compozy/tasks/autonomous bunx playwright test e2e/tasks-coordinator-handoff.spec.ts --reporter=list`

Result: 4 passed.
