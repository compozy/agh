# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Complete the `internal/cli` improvements pass with report-first inventories, baseline/post-fix benchmarks, package-local fixes, and a final clean `make verify`.

## Important Decisions
- Treat `internal/cli/docpost` as in-scope because it sits under the `internal/cli/` task directory and participates in the hidden `doc` command flow.
- Benchmark the shared CLI hot paths that are actually exercised broadly: `renderHumanTable`, `renderToonArray`, `decodeSSE`, and `(*unixSocketClient).doRequest`.
- Keep UBS as `not-run` unless a real invocation surface appears; no manual substitute.
- Fix the docs-output safety issue in `docpost.Process` itself by rejecting non-empty unmanaged output roots before `cleanOutput`, rather than trying to constrain callers in `doc.go`.
- Keep the TOON rendering optimization limited to builder-based row emission; revert the experimental `decodeSSE` fast path because the benchmark did not improve.

## Learnings
- `internal/cli` coverage baseline is already above the task goal: `81.0%`; `internal/cli/docpost` is `87.1%`.
- Production concurrency surface is small: one goroutine (`daemon.go`) and one buffered channel (`childErrCh`), with both `select` loops already `ctx.Done()`-aware.
- High-confidence bug found: the hidden `doc` command resolves a user-controlled `--output-dir` and passes it to `docpost.Process`, which currently calls `cleanOutput` on any existing directory and can recursively delete unrelated contents.
- Baseline benchmark averages before fixes:
  - `BenchmarkRenderHumanTableLarge`: `134818 ns/op`, `309199.2 B/op`, `1593 allocs/op`
  - `BenchmarkRenderToonArrayLarge`: `85385.8 ns/op`, `130891.8 B/op`, `1551 allocs/op`
  - `BenchmarkDecodeSSELargeStream`: `113787.4 ns/op`, `246216.2 B/op`, `4099 allocs/op`
  - `BenchmarkDoRequestPostJSON`: `1224.2 ns/op`, `2642 B/op`, `29 allocs/op`
- The TOON renderer fix is materially worthwhile: `BenchmarkRenderToonArrayLarge` dropped to `52362.4 ns/op`, `84752 B/op`, and `19 allocs/op`.
- The SSE decoder is not worth changing in this pass: the measured end-state regressed slightly (`113787.4 ns/op` -> `114733.6 ns/op`), so the experiment was reverted and only the multiline-data regression test remains.
- `make verify` passed cleanly on the end-state (`exit 0`; `DONE 4434 tests in 9.854s`).
- Local commit created: `dccb53d5` (`refactor: cli improvements pass`); a cached post-commit `make verify` re-run also passed (`exit 0`; `DONE 4434 tests in 0.878s`).

## Files / Surfaces
- `internal/cli/doc.go`
- `internal/cli/docpost/docpost.go`
- `internal/cli/format.go`
- `internal/cli/client.go`
- `internal/cli/client_test.go`
- `internal/cli/render_test.go`
- `internal/cli/docpost/docpost_test.go`
- `internal/cli/perf_bench_test.go`
- `.compozy/tasks/improvs/reports/cli.md`

## Errors / Corrections
- `.compozy/tasks/improvs/adrs/` does not exist for this PRD; treated as missing optional context rather than a blocker.

## Ready for Next Run
- Task-level implementation and verification are complete; only optional local cleanup remains.
