# Task Memory: task_19.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute task_19 end to end: add committed Tasks browser E2E coverage in `web/e2e`, run the QA matrix from task_18, write fresh QA artifacts under `.compozy/tasks/tasks-ui/qa/`, and document the Settings blocker explicitly if the route family is still absent on this branch.

## Important Decisions
- Reuse the existing daemon-served Playwright harness and public task/session APIs for deterministic browser seeds instead of introducing ad hoc seed endpoints or a second browser harness.
- Treat missing `web/src/routes/_app/settings*.tsx` as an execution blocker to be recorded in the verification report, not as permission to invent fake Settings coverage.
- Keep the Tasks browser lane assertions aligned to the real API contracts: mutation endpoints return direct task payloads, while `GET /api/tasks/{id}` returns a nested task-detail envelope.

## Learnings
- `web/e2e/` currently contains no `tasks*.spec.ts` or `settings*.spec.ts`, so task_19 must add the Tasks lane from scratch and cannot extend existing task/browser coverage.
- `web/e2e/fixtures/selectors.ts` and `runtime-seed.ts` currently have operator helpers for session/network/automation/bridges only; task-specific helpers are still missing.
- The task route family already exposes stable `data-testid` hooks for list, create modal, dashboard, inbox, detail tabs, runs, and run-detail/session drill-down.
- The repo-local `scripts/discover-project-contract.py` path referenced by the task is absent; QA contract discovery may need to use the skill-bundled script as fallback.
- `web/src/routes/_app/settings*.tsx` is still absent on the current branch at the start of this run.
- The shipped `TasksCreateModal` footer actions were outside the Playwright viewport because the dialog had no constrained-height flex layout or internal scroll region; task_19 now needs the production modal layout fix, not a forced test click.
- The shared Playwright harness can mirror named screenshots directly into `.compozy/tasks/tasks-ui/qa/screenshots/` through `AGH_E2E_QA_OUTPUT_DIR`, which is sufficient for the required browser evidence capture.
- The shared Tasks browser lane now passes end to end after fixing the modal layout bug and correcting the spec to read the nested task-detail response shape.

## Files / Surfaces
- `web/e2e/fixtures/selectors.ts`
- `web/e2e/fixtures/runtime-seed.ts`
- `web/e2e/*.spec.ts`
- `web/src/routes/_app/tasks.tsx`
- `web/src/routes/_app/tasks.$id.tsx`
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx`
- `.compozy/tasks/tasks-ui/qa/`
- `.compozy/tasks/tasks-ui/qa/verification-report.md`
- `.compozy/tasks/tasks-ui/qa/issues/BUG-001-tasks-create-modal-footer-offscreen.md`
- `.compozy/tasks/tasks-ui/qa/issues/BUG-002-settings-surface-missing-on-branch.md`

## Errors / Corrections
- `python3 scripts/discover-project-contract.py --root .` failed because `scripts/discover-project-contract.py` does not exist in the repo root on this branch.
- `bunx playwright test e2e/tasks.spec.ts` exposed a real UI bug: `tasks-create-modal-save-draft` was rendered outside the viewport on the shared browser lane. Fixed in production by making the modal body scroll inside a constrained-height flex dialog.
- The first publish and approval read checks in `web/e2e/tasks.spec.ts` incorrectly treated `GET /api/tasks/{id}` like a flat task mutation response. Corrected the spec to assert against the nested task-detail envelope and the actual mutation responses.

## Ready for Next Run
- Task_19 is verified locally: `make test-e2e-web` and `make verify` passed, QA evidence is written under `.compozy/tasks/tasks-ui/qa/`, and the remaining open item is the explicit Settings blocker on this branch.
