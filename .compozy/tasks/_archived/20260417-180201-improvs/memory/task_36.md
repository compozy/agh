# Task Memory: task_36.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the `internal/workspace` improvements pass with required inventories, package-local benchmarks, any justified fixes inside `internal/workspace/`, a clean `make verify`, and tracking/memory updates.

## Important Decisions
- Treat report/memory/tracking artifacts outside `internal/workspace/` as required workflow files while keeping production/test/benchmark code edits inside the package.
- Use the established UBS wording from prior package tasks because this session still exposes skill instructions but no callable UBS runner.
- Fix the canceled-context rollback cleanup in both registration flows.
- Reject clone/list micro-optimizations unless the exact full benchmark command shows a stable win.

## Learnings
- `internal/workspace` has no production goroutines, channels, or `select` statements; the only production synchronization primitive is `Resolver.mu` guarding the cache map.
- Baseline coverage is already `80.8%` before this task's new regression/benchmark scaffolding.
- Baseline benchmark medians from `go test -bench=. -benchmem -count=5 ./internal/workspace/...`:
  - `BenchmarkResolverResolve/cache_hit-16`: `143534 ns/op`, `35216 B/op`
  - `BenchmarkResolverResolve/cache_miss-16`: `272352 ns/op`, `81818 B/op`
  - `BenchmarkResolverList-16`: `22927 ns/op`, `40960 B/op`
  - `BenchmarkCloneResolvedWorkspace-16`: `364.5 ns/op`, `768 B/op`
- Final package coverage after the regression additions is `81.5%`.
- Final benchmark medians from the required post-fix command:
  - `BenchmarkResolverResolve/cache_hit-16`: `193307 ns/op`, `35221 B/op`
  - `BenchmarkResolverResolve/cache_miss-16`: `254898 ns/op`, `81816 B/op`
  - `BenchmarkResolverList-16`: `24968 ns/op`, `40960 B/op`
  - `BenchmarkCloneResolvedWorkspace-16`: `362.3 ns/op`, `768 B/op`
- The only fixed package-local finding was a correctness bug: rollback cleanup in `Register` and `ResolveOrRegister` must ignore request cancellation so a partial registration does not persist after eager resolution fails.

## Files / Surfaces
- Report: `.compozy/tasks/improvs/reports/workspace.md`
- Bench/test scaffolding: `internal/workspace/perf_bench_test.go`, `internal/workspace/resolver_test.go`
- Production touch points: `internal/workspace/resolver.go`, `internal/workspace/resolver_crud.go`

## Errors / Corrections
- Confirmed the PRD directory has no `adrs/` subdirectory, so ADR context is absent rather than blocked.
- Confirmed the package report did not exist at the start of the run; inventories are now recorded before any production fix.
- A fresh `make verify` initially failed on `gocritic` because the new benchmark used `filepath.Join("/tmp", ...)`; corrected it to join path components without an embedded separator literal and reran the full gate successfully.

## Ready for Next Run
- Task is in finalization: report, memory, and task tracking are ready to be synced with the verified state and committed with an unscoped `refactor:` subject.
