# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Run the required five-skill improvements pass for `internal/bundles/`.
- Produce `.compozy/tasks/improvs/reports/bundles.md` with the mandatory inventories before findings.
- Keep production code edits inside `internal/bundles/`, then reach clean `make verify`.

## Important Decisions
- Follow the report-first workflow used by earlier package tasks.
- Treat missing PRD ADRs as absent context rather than inventing architectural inputs.
- Fix the measured large-catalog activation lookup cost by indexing bundle records once per list/build/reconcile pass instead of re-scanning the slice per activation.
- Remove the duplicated desired-state accumulation loop in `collectDesiredState` by delegating to `collectDesiredStateFromBundleRecords`.

## Learnings
- The workspace is already dirty from other tasks; unrelated files must remain untouched.
- Shared workflow memory already records that package tasks must produce inventories and benchmark baselines before fixes.
- `ListActivations` was paying both repeated `ListBundleResources` calls and repeated bundle-record scans before the fix.
- The shared indexed lookup yields a large list-path win and a smaller build-path win, with a modest allocation tradeoff on the build benchmark.
- Repository commitlint rejects scoped commit subjects, so the task commit used `refactor: bundles improvements pass` instead of the scoped header requested by the task spec.

## Files / Surfaces
- `internal/bundles/`
- `.compozy/tasks/improvs/reports/bundles.md`
- `internal/bundles/service.go`
- `internal/bundles/resource_projection.go`
- `internal/bundles/resource_test.go`
- `internal/bundles/perf_bench_test.go`

## Errors / Corrections
- Initial bundle-record indexing lowercased lookup keys and added avoidable allocation pressure; corrected it to use exact trimmed-key indexing with case-insensitive fallback scanning.

## Ready for Next Run
- Task complete.
- Fresh `make verify` passed before commit and again on `HEAD` after commit (`DONE 4431 tests in 0.823s`).
- Local commit created: `bc417ef5` (`refactor: bundles improvements pass`).
