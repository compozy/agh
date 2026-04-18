# BUG-001: Tasks create modal footer rendered off-screen at shared browser viewport

- Date: 2026-04-17
- Severity: high
- Status: fixed in task_19
- Surface: `web/src/systems/tasks/components/tasks-create-modal.tsx`

## Summary

The shared daemon-served Playwright lane could open the Tasks create modal, but the footer actions (`Save draft` and `Create & enqueue`) rendered below the visible viewport. The operator could fill the form and still be unable to submit it in the standard browser lane.

## Evidence

- Failing command before the fix:
  - `cd web && AGH_E2E_QA_OUTPUT_DIR=../.compozy/tasks/tasks-ui bunx playwright test e2e/tasks.spec.ts`
- Failure symptom:
  - Playwright timed out clicking `tasks-create-modal-save-draft` because the button stayed outside the viewport.
- Regression evidence after the fix:
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-draft-created.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-draft-published.png`

## Root Cause

`TasksCreateModal` used an unconstrained dialog body with no internal scroll region. On the shared Playwright viewport, the form content pushed the footer below the fold, so the primary actions were not reachable.

## Fix

- Constrained the dialog width and height for the shared viewport.
- Converted the form into a flex column with an internal scrolling body so the footer remains reachable.
- Verified the end-to-end draft and publish flow through the shared browser lane.

## Regression Coverage

- `web/e2e/tasks.spec.ts`
- `web/src/systems/tasks/components/tasks-create-modal.test.tsx`
