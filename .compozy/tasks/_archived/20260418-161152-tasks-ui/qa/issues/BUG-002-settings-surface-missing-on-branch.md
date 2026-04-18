# BUG-002: Settings browser coverage blocked because the shipped route surface is absent on this branch

- Date: 2026-04-17
- Severity: blocker
- Status: open
- Surface: `web/src/routes/_app/settings*.tsx`

## Summary

Task 19 requires committed daemon-served browser E2E coverage for Settings in the shared `web/e2e` lane. This execution branch does not contain the shipped Settings route family required to add or extend that coverage safely.

## Evidence

- Branch inspection during task_19 found no `web/src/routes/_app/settings*.tsx` files.
- The app shell still exposes a Settings entry point in the sidebar, but there is no committed route family matching the task requirement for shell navigation, save flow, collection CRUD, and advanced configuration coverage.
- Task 18 planning artifacts already called this out as a preflight blocker, and the branch state remained unchanged during execution.

## Impact

- A committed `web/e2e/settings*.spec.ts` file would be fabricated coverage on this branch.
- Task 19 can ship durable Tasks coverage and full repo verification, but it cannot truthfully close the Settings browser requirement without the missing Settings surface landing first.

## Required Follow-up

- Land the Settings route family and the actual shipped operator flows on this branch.
- Add the matching shared-lane Playwright spec under `web/e2e/` using the same daemon-served harness.
