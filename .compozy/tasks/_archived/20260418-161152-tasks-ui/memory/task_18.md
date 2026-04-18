# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Produce the reusable QA planning artifacts for Tasks under `.compozy/tasks/tasks-ui/qa/` so `task_19` can execute against a fixed scope, artifact layout, and priority order.
- Include explicit Settings regression-critical coverage and blocker handling instead of leaving Settings implied by a generic regression note.

## Important Decisions
- Created one feature test plan plus two regression planning documents:
  - `qa/test-plans/tasks-ui-test-plan.md`
  - `qa/test-plans/tasks-ui-regression.md`
  - `qa/test-plans/tasks-ui-browser-regression.md`
- Assigned one manual case per required Tasks surface (`TC-FUNC-001` through `TC-FUNC-009`) and five Settings regression cases (`TC-REG-010` through `TC-REG-014`).
- Treated Settings route presence as a P0 execution preflight because `task_19` requires browser coverage only when the Settings surface exists on the execution branch.

## Learnings
- The planning branch exposes the Tasks route family and the shared `web/e2e` harness, but it does not contain `web/src/routes/_app/settings*.tsx`.
- Existing browser-lane patterns are centered on `web/e2e/fixtures/test.ts`, `runtime.ts`, `runtime-seed.ts`, and `selectors.ts`; task_19 should extend those instead of inventing a second harness.
- `web/playwright.config.ts` currently defines the committed browser lane as `Desktop Chrome`, so the blocking E2E expectation in the plan is desktop-first.

## Files / Surfaces
- `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-test-plan.md`
- `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-regression.md`
- `.compozy/tasks/tasks-ui/qa/test-plans/tasks-ui-browser-regression.md`
- `.compozy/tasks/tasks-ui/qa/test-cases/TC-FUNC-001.md` through `.compozy/tasks/tasks-ui/qa/test-cases/TC-FUNC-009.md`
- `.compozy/tasks/tasks-ui/qa/test-cases/TC-REG-010.md` through `.compozy/tasks/tasks-ui/qa/test-cases/TC-REG-014.md`
- `docs/design/paper/tasks/*`
- `docs/design/paper/settings/*`
- `web/e2e/*`

## Errors / Corrections
- No implementation bug was fixed in this task because the scope stayed on QA planning.
- The absence of Settings routes on the planning branch was handled as a documented execution preflight requirement rather than an implicit skip.

## Ready for Next Run
- Task tracking now marks `task_18` completed in `.compozy/tasks/tasks-ui/task_18.md` and `.compozy/tasks/tasks-ui/_tasks.md`.
- The QA artifact deliverables were committed as `ecd0fdd4` (`docs: add tasks qa plan artifacts`).
- Fresh verification evidence for this docs task is the repo-wide post-commit `make verify` pass after the QA artifact files were written and formatted by the commit hook.
- Run `TC-REG-010` first on the execution branch.
- Seed deterministic Tasks draft, publish, run-detail, aggregate, and multi-agent states before writing the browser specs.
- Keep the verification report, screenshots, and any bug files under `.compozy/tasks/tasks-ui/qa/`.
