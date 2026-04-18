# Task 19 Verification Report

- Date: 2026-04-17
- Task: `task_19` - Tasks QA execution and settings-aligned browser E2E
- QA output root: `.compozy/tasks/tasks-ui`
- Execution matrix source:
  - `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-test-plan.md`
  - `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-regression.md`
  - `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-browser-regression.md`
  - `.compozy/tasks/tasks-ui/qa/test-cases/*.md`

## Scope Executed

- Added committed daemon-served Tasks browser coverage in `web/e2e/tasks.spec.ts`.
- Extended the shared browser harness for Tasks seeds, selectors, and QA screenshot mirroring.
- Executed the shipped Tasks operator flow through the shared Playwright lane:
  - open Tasks from the sidebar
  - create a draft task
  - publish it
  - open task detail
  - validate the live Agents fallback state
  - open Dashboard
  - open run detail
  - drill into the linked session
  - open Inbox
  - approve pending work
- Performed the required Settings preflight on the execution branch.

## Issues Found

- `BUG-001`: Tasks create modal footer actions were off-screen at the shared browser viewport. Fixed in production and covered by the shared browser lane.
- Settings browser coverage remains blocked on this branch because the shipped Settings route family is absent.
  - See `.compozy/tasks/tasks-ui/qa/issues/BUG-002-settings-surface-missing-on-branch.md`
- One spec bug was found and corrected during execution:
  - `GET /api/tasks/{id}` returns a task detail envelope shaped like `{ task: { summary, task, ... } }`, not the flatter mutation-response shape. The browser assertion was corrected to read the nested detail payload instead of accidentally passing on `undefined`.

## Evidence Artifacts

- Screenshots:
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-list-seeded.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-draft-created.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-draft-published.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-detail-route.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-live-fallback.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-dashboard.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-run-detail.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-linked-session.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-inbox-approval-pending.png`
  - `.compozy/tasks/tasks-ui/qa/screenshots/tasks-inbox-approval-approved.png`
- Issue reports:
  - `.compozy/tasks/tasks-ui/qa/issues/BUG-001-tasks-create-modal-footer-offscreen.md`
  - `.compozy/tasks/tasks-ui/qa/issues/BUG-002-settings-surface-missing-on-branch.md`

## Verification Commands

- `make web-fmt`
  - passed
- `cd web && bun run test --run e2e/fixtures/selectors.test.ts e2e/fixtures/runtime-seed.test.ts e2e/fixtures/artifacts.test.ts e2e/fixtures/browser-artifact-session.test.ts src/systems/tasks/components/tasks-create-modal.test.tsx`
  - passed
  - `5` files, `22` tests
- `make web-typecheck`
  - passed
- `cd web && AGH_E2E_QA_OUTPUT_DIR=../.compozy/tasks/tasks-ui bunx playwright test e2e/tasks.spec.ts`
  - passed
  - `1` spec, `1` passed
- `make test-e2e-web`
  - passed
  - `7` specs, `7` passed
- `make verify`
  - passed

## Branch Blockers

- The task requirement for committed Settings browser coverage could not be completed truthfully on this branch because `web/src/routes/_app/settings*.tsx` is absent.
- This blocker was reported explicitly instead of fabricating a `web/e2e/settings*.spec.ts` file against a missing shipped surface.

## Verdict

- Tasks browser E2E coverage is now committed in the shared daemon-served `web/e2e` lane.
- The shared repo browser gate and full repository verification gate both pass with the new coverage.
- Settings browser coverage remains an explicit open blocker on this branch and requires the missing shipped surface before a real shared-lane Settings spec can be added.
