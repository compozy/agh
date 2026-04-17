# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the improvements pass for `internal/bridges/` with report-first inventories, broker benchmarks, scoped fixes, and clean verification.

## Important Decisions
- Use `internal/bridges/perf_bench_test.go` for the mandatory optimization evidence instead of piggybacking on unrelated package benchmarks.
- Treat the strongest current correctness lead as a delivery-default validation mismatch between resource-managed bridge specs and the runtime/registry path.
- Unify every bridge-instance delivery-default entry point on `NormalizeDeliveryDefaultsJSON` rather than preserving separate resource and registry validators.
- Replace the broker turn-index string key with a structured `turnIndexKey` only after the benchmark confirmed it was a real allocation source on the lookup path.

## Learnings
- `internal/bridges` is dominated by broker/runtime code: one production goroutine, one production channel, and one production mutex all live in `delivery_broker.go`.
- Baseline benchmark averages before fixes:
  - `BenchmarkBrokerProjectEventTurnLookup`: `26.17 ns/op`, `24 B/op`
  - `BenchmarkBrokerEnqueueEventLockedDelta`: `145.78 ns/op`, `320 B/op`
  - `BenchmarkBrokerPrepareRequestDelta`: `210.52 ns/op`, `512 B/op`
  - `BenchmarkBrokerDeliveryMetricsSnapshot`: `6024.40 ns/op`, `11608 B/op`
- Resource-side delivery-default normalization is stricter than the runtime path: `resource.go` rejects provider-specific keys and bare `thread_id`, while `types.go`, `target.go`, and existing tests treat those shapes as valid/pass-through.
- Post-fix benchmark averages:
  - `BenchmarkBrokerProjectEventTurnLookup`: `9.78 ns/op`, `0 B/op`
  - `BenchmarkBrokerEnqueueEventLockedDelta`: `145.36 ns/op`, `320 B/op`
  - `BenchmarkBrokerPrepareRequestDelta`: `210.76 ns/op`, `512 B/op`
  - `BenchmarkBrokerDeliveryMetricsSnapshot`: `6079.40 ns/op`, `11608 B/op`
- Focused package coverage after the fix set is `80.9%`.

## Files / Surfaces
- `internal/bridges/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/bridges.md`
- `internal/bridges/delivery_broker.go`
- `internal/bridges/registry.go`
- `internal/bridges/registry_test.go`
- `internal/bridges/resource.go`
- `internal/bridges/resource_test.go`
- `internal/bridges/types.go`

## Errors / Corrections
- Replaced an unstable end-to-end `ProjectEvent` benchmark with a deterministic turn-lookup benchmark after the worker-backed version produced noisy long-running results.

## Ready for Next Run
- Completed in local commit `a0c91b92` (`refactor: bridges improvements pass`).
- Post-commit `make verify` re-ran cleanly on `HEAD` (`DONE 4428 tests in 0.646s`).
