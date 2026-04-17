# Task Memory: task_24.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Run the required five-skill improvements pass for `internal/resources/`, write `.compozy/tasks/improvs/reports/resources.md`, keep code edits inside `internal/resources/`, pass `make verify`, update tracking, and auto-commit once clean.

## Important Decisions
- Start from inventories and benchmark baselines before any code edits, per techspec.
- Keep the commit scoped to code/report artifacts only; task tracking and workflow-memory files are still updated locally but stay out of the automatic deliverable commit.

## Learnings
- No deeper `AGENTS.md` files cover `internal/resources/`; only the repo-root instructions apply here.
- The worktree already contains unrelated task-tracking edits; these must be left intact and worked around.
- No ADR files exist under `.compozy/tasks/improvs/adrs/`.
- `reconcileDriver.Trigger` was holding the scheduler mutex while calling the event sink, which made re-entrant sinks deadlock the worker path.
- `Kernel.sourceLocks` needed ref-counted entries so idle per-source mutexes can be removed without racing waiters.
- `make verify` now passes cleanly after the package-local fixes; package coverage is `79.4%`.
- Local commit `41012332` (`refactor: resources improvements pass`) was created after selective staging, and the committed tree also passed a fresh post-commit `make verify`.

## Files / Surfaces
- `internal/resources/errors.go`
- `internal/resources/codec.go`
- `internal/resources/reconcile.go`
- `internal/resources/schema.go`
- `internal/resources/validate.go`
- `internal/resources/projector.go`
- `internal/resources/doc.go`
- `internal/resources/types.go`
- `internal/resources/typed.go`
- `internal/resources/kernel.go`
- `internal/resources/perf_bench_test.go`
- `internal/resources/*_test.go`
- `.compozy/tasks/improvs/reports/resources.md`

## Errors / Corrections
- Fixed a re-entrant event-sink deadlock by deferring reconcile event emission until after unlocking `d.mu`.
- Fixed unbounded `sourceLocks` registry growth by ref-counting per-source lock entries and deleting them when idle.

## Ready for Next Run
- Task is complete; only user handoff remains.
