# Task Memory: task_21.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute the `internal/observe` improvements pass end-to-end: report-first inventories, benchmarks, package-local fixes, clean `make verify`, tracking updates, and one local commit.

## Important Decisions
- Treat `.compozy/tasks/improvs/adrs/` as absent context because the directory does not exist.
- Keep code edits inside `internal/observe/`; treat the report, workflow-memory, ledger, and task-tracking files as required task artifacts outside the package.
- Keep tracking-only files out of the automatic deliverable commit unless the task explicitly requires staging them.
- Keep the large-file refactors for `observer.go` and `tasks.go` deferred rather than mixing a broad structural split into this measured improvements pass.

## Learnings
- Cross-task pattern is stable: inventories and benchmark baselines must be captured before package fixes, and `ubs` should be marked `not-run` if no concrete skill runner is available.
- `truncateSummary` had a steady-state allocation on short event summaries; a byte-length fast path removes that cost without changing behavior.
- `taskMetricsFromSnapshot` was carrying most of the package-local allocation pressure because it built filtered slices and duplicate-ingress helper slices even for permissive queries; count helpers plus zero-copy filters materially reduced heap use.

## Files / Surfaces
- `internal/observe/observer.go`
- `internal/observe/tasks.go`
- `internal/observe/helpers_test.go`
- `internal/observe/hooks_test.go`
- `internal/observe/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/observe.md`

## Errors / Corrections
- A structured-key rewrite for task-summary aggregation was benchmarked, increased memory use, and was reverted instead of being kept as a fake optimization win.

## Ready for Next Run
- Task 21 is complete. Deliverable commit: `87f77b5a` (`refactor: observe improvements pass`).
- Post-commit `make verify` also passed via `/tmp/observe-make-verify-post-commit.txt`; any future follow-up can start from that clean baseline instead of re-running the whole improvements pass.
