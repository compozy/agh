# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement durable automation scheduler state for Task 04: persist scheduler cursors, advance before dispatch, reconcile missed runs, separate delivery errors, expose diagnostics across backend/API/CLI/web/docs, and prove restart/duplicate-prevention behavior.

## Important Decisions
- Durable scheduler state uses `automation_scheduler_state` plus scheduled-run metadata on `automation_runs` (`fire_id`, `scheduled_at`, `delivery_error`, `delivery_error_at`).
- Scheduled fires are claimed through `ClaimScheduledRun`, which advances the cursor and inserts the reserved run in one transaction before dispatcher delivery begins.
- Boot reconciliation currently implements ADR-002 `skip_missed`: missed cursors are recorded as misfires and advanced to the next future fire without dispatching stale work.
- Delivery failures are recorded on run diagnostics via `delivery_error` fields; scheduler cursor state is not rolled back or polluted with delivery errors.

## Learnings
- `make codegen` updates `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, and `sdk/typescript/src/generated/contracts.ts` after automation contract changes.
- Web automation UI derives its domain types directly from generated OpenAPI responses, so backend contract additions can be surfaced by extending existing component props and fixtures rather than duplicating DTOs.

## Files / Surfaces
- Backend scheduler/store/model: `internal/automation/*`, `internal/automation/model/*`, `internal/store/globaldb/*automation*`, `internal/store/sql_helpers.go`.
- API/CLI: `internal/api/contract/automation.go`, `internal/api/core/automation.go`, `internal/api/core/conversions.go`, `internal/cli/automation.go`.
- Generated clients/contracts: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`.
- Web: `web/src/systems/automation/types.ts`, `index.ts`, fixtures, detail panel, run history, and component tests.
- Docs/QA tracking: automation runtime docs, CLI automation/observe docs, `.compozy/tasks/hermes/task_10.md`.

## Errors / Corrections
- Global DB migration test initially exposed unstable `automation_runs` column ordering from prepended ALTER statements; migration assembly was corrected to add new run columns in stable logical order before creating the unique fire ID index.
- CLI toon output tests needed updating because run lists now expose `scheduled_at` and `delivery_error`.
- `make verify` initially failed on scheduler cancellation ownership lint and an always-nil unregister helper; cancellation ownership is now explicit for lifecycle-owned cancels, and `unregisterLocked` no longer returns an unused error.
- A pre-existing `time.Sleep` polling helper in the touched scheduler integration test was replaced with timer/ticker synchronization to satisfy the Task 04 orchestration-test constraint.

## Ready for Next Run
- Focused backend/store/API/CLI tests passed after scheduler/store/API/CLI changes.
- Focused web component tests, web typecheck, and `make codegen-check` passed after generated contract and UI updates.
- `go test -tags integration ./internal/automation -run TestSchedulerIntegration -count=1` passed after removing `time.Sleep` from the scheduler integration helper.
- Full `make verify` passed after all code and test changes: web format/lint/typecheck/tests/build, Go lint/tests/build, 5797 Go tests, and package-boundary checks.
- Scoped local commit created: `e8a17a4b feat: add durable automation scheduler`.
- Post-commit `make verify` passed: web format/lint/typecheck/tests/build, Go lint/tests/build, 5797 Go tests, and package-boundary checks.
- Remaining before completion: none for Task 04 implementation; `.compozy` tracking files remain untracked per repository staging guidance.
