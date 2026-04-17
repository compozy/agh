# Task Memory: task_29.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute the `internal/subprocess` improvements pass end-to-end with report-first ordering, package-local fixes only, clean `make verify`, and final tracking/commit updates.

## Important Decisions
- Treat `_techspec.md` as the execution contract: all five inventory sections and baseline benchmarks must be present in `.compozy/tasks/improvs/reports/subprocess.md` before fix triage.
- Keep `ubs` strictly non-substitutable; if there is no real skill runner path in this environment, record `not-run` with the literal refusal/tooling limitation.
- No ADR files exist under `.compozy/tasks/improvs/adrs`, so there is no extra architecture context beyond the task docs and repository guidance.
- Fix the confirmed shutdown bug by deriving health-probe contexts from `p.lifecycleCtx`, not by adding sleeps or weakening the monitor stop path.
- Leave child-request goroutine fan-out as deferred: bounding it safely requires a transport-level design that preserves response progress on the same stdout stream.
- Optimize `boundedBuffer.Write` in place and prove it with the required full-suite benchmark command instead of speculative micro-edits elsewhere.

## Learnings
- Shared workflow memory already records repo-wide constraints relevant here: report-first package tasks, unscoped commit headers, and benchmark decisions based on full command medians.
- `TestStopHealthMonitorCancelsInFlightProbe` reproduced the bug entirely in-package with an in-memory transport stub by waiting for one pending probe entry, avoiding a flaky subprocess timing dependency.
- The benchmark medians showed only one meaningful package-local performance lever: `BenchmarkBoundedBufferWriteOverflow` improved from `1782 ns/op` / `21760 B/op` to `1043 ns/op` / `10240 B/op`; the other measured candidates were effectively not hot.

## Files / Surfaces
- `internal/subprocess/health.go`
- `internal/subprocess/process.go`
- `internal/subprocess/process_test.go`
- `internal/subprocess/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/subprocess.md`

## Errors / Corrections
- Initial targeted regression test failed because `runHealthProbe` used `context.Background()`, which kept the pending probe alive after lifecycle cancellation and blocked `stopHealthMonitor()`.

## Ready for Next Run
- Task tracking was updated (`task_29.md` completed, `_tasks.md` marked complete) and the implementation commit was created as `58ec60fe` with message `refactor: subprocess improvements pass`.
- A fresh post-commit `make verify` rerun also exited cleanly; final observed tail included `✓  internal/subprocess (cached)`, `DONE 4501 tests in 1.262s`, and `OK: all package boundaries respected`.
