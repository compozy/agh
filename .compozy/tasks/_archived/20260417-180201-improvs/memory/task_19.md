# Task Memory: task_19.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the report-first improvements pass for `internal/memory/`: mandatory inventories, benchmarks, scoped fixes, report, clean verification evidence, tracking updates, and one local commit.

## Important Decisions
- Treat missing ADR files as absent context unless `.compozy/tasks/improvs/adrs/` exists when checked.
- Keep cross-package findings as deferred report items instead of expanding code scope.
- Update task-tracking files locally but keep tracking-only artifacts out of the automatic code/report commit.

## Learnings
- `_techspec.md` requires every inventory section to exist before the Findings table; missing artifacts auto-fail the task even if `make verify` passes.
- `MEMORY.md` deletion matching must parse the first markdown-link target with balanced parentheses because `cleanFilename` permits filenames like `user(preferences).md`.

## Files / Surfaces
- `internal/memory/` package tree
- `internal/memory/store.go`
- `internal/memory/store_test.go`
- `internal/memory/perf_bench_test.go`
- `internal/memory/consolidation/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/memory.md`

## Errors / Corrections
- `ubs` remains blocked until a callable skill runner is confirmed; if unavailable, task must record `not-run` with the literal tooling limitation.
- Fixed `removeIndexEntry` so deleting one memory only drops the matching first markdown-link target instead of any line that mentions the filename.
- Fixed `Store.Scan` so capped scans sort metadata first and stop after collecting 200 valid headers instead of parsing every file in the directory.

## Ready for Next Run
- Task artifacts are complete; the remaining handoff is the local commit containing only code/report deliverables after the fresh verification evidence.
