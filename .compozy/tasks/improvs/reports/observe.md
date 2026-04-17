# Improvements Report — internal/observe

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/observe/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/observe --glob '!**/*_test.go') | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 16 | `New` | `internal/observe/observer.go:200` |
| 12 | `(*Observer).activeCounts` | `internal/observe/health.go:71` |
| 11 | `(*Observer).loadTaskSnapshot` | `internal/observe/tasks.go:313` |
| 10 | `(*Observer).collectTaskHealth` | `internal/observe/tasks.go:223` |
| 10 | `(*Observer).QueryHookRuns` | `internal/observe/query.go:52` |
| 9 | `defaultPermissionModeResolver` | `internal/observe/observer.go:634` |
| 9 | `(*Observer).loadSessionMetadata` | `internal/observe/reconcile.go:31` |
| 9 | `(*Observer).collectBridgeHealth` | `internal/observe/bridges.go:159` |
| 8 | `summarizeRecovery` | `internal/observe/tasks.go:595` |
| 8 | `runStuckAge` | `internal/observe/tasks.go:649` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/observe/observer.go` | 816 | Observer construction, session tracking, hook-store access, event ingestion, and summary helpers are concentrated in one large unit. |
| `internal/observe/tasks.go` | 909 | Task summary, metrics, health, filtering, and sorting logic are concentrated in one file with repeated aggregation and filtering patterns. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/observe` found the following production duplicates at or above the reporting threshold:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/observe/tasks.go:404-412` | `internal/observe/tasks.go:455-463` | Repeated sort closures for task-status and task-run summary rows. |
| `internal/observe/tasks.go:754-765` | `internal/observe/tasks.go:787-798` | Repeated origin-kind filtering helpers for task and task-run slices. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `summarizeEvent` | `internal/observe/observer.go:740` | Runs on every recorded agent event and currently assembles candidate slices before truncation; this is the package’s steady-state event-ingestion hot path. | `BenchmarkSummarizeEventPermission` |
| `taskSummaryFromSnapshot` | `internal/observe/tasks.go:279` | Backs `QueryTaskSummary` and part of `Health`, aggregating task/run/owner buckets for every summary request. | `BenchmarkTaskSummaryFromSnapshotLarge` |
| `taskMetricsFromSnapshot` | `internal/observe/tasks.go:291` | Backs `QueryTaskMetrics` and part of `collectTaskHealth`, aggregating filtered metrics, latencies, and audit counts for each metrics/health request. | `BenchmarkTaskMetricsFromSnapshotLarge` |
| `collectBridgeHealth` | `internal/observe/bridges.go:159` | Traverses every bridge instance plus route and delivery-metric lookup on each bridge-health/daemon-health query. | `BenchmarkCollectBridgeHealthLarge` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/observe/...`

Final `after` command: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/observe/...`

Values below use the median of 5 runs from `/tmp/observe-bench-before.txt` and `/tmp/observe-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkSummarizeEventPermission` | 70.72 | 144 | 9.582 | 0 | fixed-with-benchmark |
| `BenchmarkTaskSummaryFromSnapshotLarge` | 269001 | 94992 | 270069 | 94992 | not-hot-confirmed-by-benchmark |
| `BenchmarkTaskMetricsFromSnapshotLarge` | 515254 | 1576712 | 358198 | 134920 | fixed-with-benchmark |
| `BenchmarkCollectBridgeHealthLarge` | 60194 | 155688 | 60519 | 155688 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No production `go` statements exist in `internal/observe/`.

### Concurrency — Channel Inventory

No production channels are declared in `internal/observe/`.

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/observe/observer.go:92` | read-heavy (`sync.RWMutex`) | `sessions`, `bridgeState`, and runtime-injected observer dependencies such as `hookCatalogSource` and `bridgeSource` | Reads dominate via snapshot/query paths; writes happen during session lifecycle and bridge-runtime updates. |

### Concurrency — Select Audit

No production `select` statements exist in `internal/observe/`.

### Security — Threat Model

- Trust boundaries:
  - Internal API handlers and CLI surfaces call `internal/observe` query methods with parsed filter objects.
  - Daemon/session orchestration calls observer lifecycle methods with session metadata and agent-event payloads.
  - Hook telemetry fan-out calls `WriteHookRecord` for per-session persistence.
- Attacker capabilities:
  - Remote or local clients can influence query filters such as task summary filters and hook-run session selectors through HTTP/UDS APIs.
  - Agent subprocesses can emit arbitrary event text, titles, resources, and usage payloads that flow into observability records.
  - A malicious caller that reaches hook-run lookup can attempt to supply a crafted `session_id` that influences filesystem path resolution.
- In-scope assets:
  - Integrity and scoping of global/session observability records.
  - Filesystem boundary around per-session hook-run databases.
  - Availability and correctness of task/bridge health summaries.
- Out-of-scope:
  - A fully compromised local operator who already controls AGH home paths or the host filesystem.
  - Authorization and transport-layer policy in API packages before queries reach `internal/observe`.
  - Security properties of downstream SQLite/query builders outside this package’s direct control.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/observe/observer.go:401` | `acp.AgentEvent` payloads emitted by agent subprocesses into `OnAgentEvent` | `normalizeObservedAgentEvent`, `validateObservedEvent`, `shouldAggregateUsage`, and summary truncation reject unsupported payloads and blank session/type values before persistence. | `registry.WriteEventSummary`, `registry.UpdateTokenStats`, and `registry.WritePermissionLog`. | LOW — rejected; internal typed event stream with validation and no command/path/HTML sink. |
| `internal/observe/tasks.go:194` | Task summary filters parsed from API query params into `TaskSummaryQuery` | `TaskSummaryQuery.Validate` allowlists scope/owner/origin enums and trims string filters before registry reads. | `registry.ListTasks`, `registry.ListTaskRuns`, `registry.ListTaskEvents`, and in-memory summary aggregation. | LOW — rejected; read-only filtered queries with typed enums and no write side effects. |
| `internal/observe/tasks.go:205` | Task metrics filters parsed from API query params into `TaskMetricsQuery` | `TaskMetricsQuery.Validate` allowlists origin enums; parser constrains `since` to RFC3339 timestamps and `network_channel` to trimmed strings. | `registry.ListTasks`, `registry.ListTaskRuns`, `registry.ListTaskEvents`, `registry.ListNetworkAudit`, and in-memory metrics aggregation. | LOW — rejected; read-only aggregation path with typed filters and no mutation sink. |
| `internal/observe/query.go:31` | Hook catalog filters from API query params (`agent`, `event`, `source`, `mode`) | Upstream parser validates hook-event/source/mode enums before `QueryHookCatalog` reads the runtime source. | `hookCatalogSource.Catalog`. | LOW — rejected; purely in-memory read path with validated enums. |
| `internal/observe/query.go:52` | Hook-run lookup filters from API query params, especially `session` | `HookRunQuery.Validate` checks limit/outcome, the API handler verifies `Sessions.Status(query.SessionID)`, and `sanitizeHookSessionID` now rejects traversal/path-separator input before path resolution. | `openHookRunStore` -> session DB path resolution and `storeHandle.QueryHookRuns`. | LOW — fixed; package-local file-boundary check now rejects unsafe session selectors before any filesystem access. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `OBS-PERF-001` | extreme-software-optimization | medium | `internal/observe/observer.go:740` | Event summarization allocated on every recorded agent event because short summaries still paid the rune-slice truncation cost. | fixed |
| `OBS-PERF-002` | extreme-software-optimization | medium | `internal/observe/tasks.go:291` | Task metrics always materialized filtered event/audit slices and extra enqueue-count slices even when the query accepted nearly every row. | fixed |
| `OBS-SEC-001` | security-review | low | `internal/observe/observer.go:290` | Hook-run session selectors reached session DB path resolution without package-local path-segment validation. | fixed |
| `OBS-REF-001` | refactoring-analysis | medium | `internal/observe/tasks.go:1` | `tasks.go` is a 900+ LOC large unit that mixes read-model construction, filtering, sorting, and health logic. | deferred |
| `OBS-REF-002` | refactoring-analysis | medium | `internal/observe/observer.go:1` | `observer.go` is an 800+ LOC large unit that mixes construction, session lifecycle, hook-store access, event ingestion, and helper logic. | deferred |
| `OBS-REF-003` | refactoring-analysis | low | `internal/observe/tasks.go:404` | The duplication scan still finds repeated summary-sort and origin-filter boilerplate in `tasks.go`. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- The package still has two large non-test files: `internal/observe/tasks.go` at 909 LOC and `internal/observe/observer.go` at 816 LOC.
- I did not split those units inside this task because meaningful decomposition would cross multiple responsibilities at once and risks broad churn in a package that is already at the coverage floor.
- The duplication scan found repeated sort/filter boilerplate in `tasks.go`, but extracting another abstraction for those small helpers would add indirection without materially reducing change cost in this pass.
- Package coverage is still above target at `80.1%` via `go test -cover ./internal/observe/...`.

### extreme-software-optimization

- Added `internal/observe/perf_bench_test.go` before changing production code so the package had measured baselines for every selected candidate.
- `truncateSummary` now takes a byte-length fast path, which removes the steady-state rune-slice allocation for the common short-summary case; `BenchmarkSummarizeEventPermission` improved from `70.72 ns/op, 144 B/op, 1 alloc/op` to `9.582 ns/op, 0 B/op, 0 alloc/op`.
- `taskMetricsFromSnapshot` now counts duplicate-ingress signals without building throwaway slices and returns the original event/audit slices when permissive filters accept every row; `BenchmarkTaskMetricsFromSnapshotLarge` improved from `515254 ns/op, 1576712 B/op, 4113 allocs/op` to `358198 ns/op, 134920 B/op, 4109 allocs/op`.
- `BenchmarkTaskSummaryFromSnapshotLarge` and `BenchmarkCollectBridgeHealthLarge` did not show a winning package-local optimization. A structured-key rewrite was tried for the summary path, worsened memory use, and was reverted rather than kept as a fake win.

### ubs

- `not-run` due missing skill-runner support in this session; no CLI/manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; no production goroutine/channel/select surfaces were found.

### security-review

- Threat model and input-surface inventory were completed before fixes.
- The only package-local boundary issue worth changing was the hook-run session path: `sanitizeHookSessionID` now rejects `.`/`..` and any slash-separated selector before the observer touches the filesystem.
- No remaining HIGH-confidence or MEDIUM-confidence exploit path survived the final review. The other surfaces are read-only, typed, or already validated before reaching downstream stores.

## Deferred Items (carry forward)

- `OBS-REF-001` — Splitting `tasks.go` into summary/metrics/health-specific files would be a larger design cleanup and should be handled as a focused follow-up refactor instead of piggybacking on this pass.
- `OBS-REF-002` — Splitting `observer.go` into constructor/lifecycle/query/helper files would touch multiple responsibilities and deserves a dedicated follow-up.
- `OBS-REF-003` — The remaining sort/filter duplication in `tasks.go` is low-value boilerplate; extracting it now would add abstraction without a measurable win.

## `make verify`

Command: `make verify`

Exit code: `0`

Fresh output excerpt from `/tmp/observe-make-verify.txt` after all closeout edits:

```text
✓  internal/observe (cached)
✓  internal/task (cached)
✓  internal/network (cached)
✓  internal/hooks (cached)

DONE 4488 tests in 0.964s
OK: all package boundaries respected
```

Warnings:

- Non-blocking Node environment warnings were emitted during the web verification steps: `NO_COLOR` ignored because `FORCE_COLOR` is set.
- Non-blocking macOS linker warning was emitted while building the vendored linter toolchain: `ld: warning: -bind_at_load is deprecated on macOS`.

Verdict: PASS
