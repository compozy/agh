# Improvements Report — internal/memory

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/memory/perf_bench_test.go` and `internal/memory/consolidation/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/memory | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `TestStoreLoadIndex` | `internal/memory/store_test.go:410` |
| 19 | `resolveWorkspaces` | `internal/memory/consolidation/runtime.go:285` |
| 17 | `TestAssemblerAssemble` | `internal/memory/assembler_test.go:14` |
| 17 | `(*ConsolidationLock).TryAcquire` | `internal/memory/lock.go:63` |
| 13 | `(*Store).Scan` | `internal/memory/store.go:155` |
| 12 | `TestStoreScanReturnsNewestFirst` | `internal/memory/store_test.go:268` |
| 12 | `(*Service).scanCompletedSessionsSince` | `internal/memory/dream.go:314` |
| 12 | `(*Assembler).PromptSection` | `internal/memory/assembler.go:48` |
| 12 | `TestRuntimeTriggerStates` | `internal/memory/consolidation/runtime_test.go:46` |
| 12 | `TestNewSessionSpawnerCreatesDreamSession` | `internal/memory/consolidation/runtime_test.go:282` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/memory/consolidation/runtime.go` | 463 | Runtime lifecycle, background scheduling, workspace resolution, and dream session spawning are concentrated in one file. |
| `internal/memory/dream.go` | 456 | Consolidation gate evaluation, lock state, workspace prep, and session-history scanning share one large unit. |
| `internal/memory/store.go` | 436 | Store persistence, path validation, index truncation, and scan/index-maintenance helpers live in one file. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/memory` only flagged duplicated test blocks:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/memory/consolidation/runtime_test.go:49-71` | `internal/memory/consolidation/runtime_test.go:73-95` | Table-shape duplicate in trigger-state tests only. |
| `internal/memory/assembler_test.go:17-30` | `internal/memory/assembler_test.go:32-45` | Parallel test setup duplication only. |

No production duplicate block reached the `>= 8 line` threshold.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*Store).Scan` | `internal/memory/store.go:156` | Memory listing is used by API handlers (`internal/api/core/memory.go:162-209`) and extension host API reads (`internal/extension/host_api.go:1037`); the implementation currently performs full file reads/frontmatter parses across the entire directory before capping at 200 headers. | `BenchmarkStoreScanCappedWorkspace` |
| `(*Assembler).PromptSection` | `internal/memory/assembler.go:48` | Prompt assembly runs on the session prompt path (`internal/session/manager_helpers.go:30-38`) and loads/trims both global and workspace `MEMORY.md` indexes on demand. | `BenchmarkAssemblerPromptSectionDualIndex` |
| `(*Service).scanCompletedSessionsSince` | `internal/memory/dream.go:314` | The dream gate scans persisted session metadata on each consolidation eligibility check and is the package’s main session-history I/O loop. | `BenchmarkScanCompletedSessionsSince` |
| `resolveWorkspaces` | `internal/memory/consolidation/runtime.go:285` | Dream session spawning without an explicit workspace walks all sessions, deduplicates by workspace, and sorts candidates before spawning. | `BenchmarkResolveWorkspacesRecentSessions` |

### Optimization — Benchmark Results

Baseline command before production changes: `go test -run '^$' -bench=. -benchmem -count=5 ./internal/memory/...`

Median values from the 5 baseline runs:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkStoreScanCappedWorkspace` | 13221282 | 5668377 | 5719389 | 2465866 | fixed-with-benchmark |
| `BenchmarkAssemblerPromptSectionDualIndex` | 26460 | 100248 | 26774 | 100248 | not-hot-confirmed-by-benchmark |
| `BenchmarkScanCompletedSessionsSince` | 5831664 | 605789 | 5880838 | 605787 | not-hot-confirmed-by-benchmark |
| `BenchmarkResolveWorkspacesRecentSessions` | 26234 | 60592 | 26492 | 60592 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/memory/consolidation/runtime.go:141` | `(*Runtime).Start` | `dreamCtx.Done()` plus `Runtime.Shutdown()` waiting on `wg`. | Single background dream-check loop created when the runtime starts. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/memory/consolidation/runtime.go:48` | `1` (created at `runtime.go:131`) | `(*Runtime).Start` | none; shutdown uses context cancellation and clears the field under `Runtime.mu` | Background loop in `Start`, writers in `EnqueueCheck` | Bounded best-effort queue for dream-check requests. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/memory/dream.go:60` | write-heavy | `pending` and `priorMtime` lock-state fields | Guards lock-acquisition bookkeeping and pending-run state. |
| `internal/memory/dream.go:61` | write-heavy | end-to-end `Run` serialization | Prevents concurrent consolidation runs from interleaving workspace prep / spawn / release paths. |
| `internal/memory/consolidation/runtime.go:47` | mixed | background runtime lifecycle fields (`checkCh`, `cancel`) | Protects start/shutdown/enqueue coordination. |

### Concurrency — Select Audit

| File:Line | Reasoning |
| --- | --- |
| `internal/memory/consolidation/runtime.go:148` | Main background-loop `select` is context-aware via `dreamCtx.Done()`. |
| `internal/memory/consolidation/runtime.go:175` | `EnqueueCheck` intentionally uses a non-blocking `default` branch on the bounded queue; no missing cancellation bug. |

### Security — Threat Model

- Trust boundaries:
  - `internal/memory` is called from local daemon/API/CLI/extension paths, not directly from the network. The primary callers are `internal/api/core/memory.go`, `internal/cli/memory.go`, `internal/extension/host_api.go`, and `internal/session/manager_helpers.go`.
  - `internal/memory/consolidation` bridges daemon-triggered consolidation requests into session-manager and workspace-resolver surfaces.
- Attacker capabilities:
  - A local API/CLI/extension caller can provide memory filenames, scope values, raw memory-file content, and dream-trigger workspace references.
  - A local user can place malformed files inside the configured memory directories or malformed `meta.json` files inside the sessions directory.
- In-scope assets:
  - File-operation scoping to the configured global/workspace memory directories.
  - Integrity of `MEMORY.md` index maintenance.
  - Safe workspace resolution for dream-trigger requests.
  - Stable dream gate behavior when malformed files are present.
- Out-of-scope:
  - A fully compromised local operator who already controls the configured memory directories or session metadata directories.
  - Authorization decisions in upstream API/CLI/extension layers before they call into this package.
  - Session-manager or workspace-resolver internals outside the code in `internal/memory/`.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/memory/store.go:113` | Raw memory content plus filename/scope from API, CLI, or extension write requests (`internal/api/core/memory.go:367-394`, `internal/cli/memory.go:477-489`, `internal/extension/host_api.go:2034-2059`) | `parseFrontmatter`, `Header.Validate`, `cleanFilename`, and `Scope.Validate` | `fileutil.AtomicWriteFile` to the scope-local memory directory | LOW — rejected; path traversal is blocked and content is stored verbatim as a local markdown file after strict frontmatter validation. |
| `internal/memory/store.go:81` | Filename/scope for memory reads from API, CLI, or extension callers | `cleanFilename` and `Scope.Validate` via `pathFor` / `dirForScope` | `os.ReadFile` inside the scope-local memory directory | LOW — rejected; local file read constrained to configured directories with no path separator bypass. |
| `internal/memory/store.go:96` | Filename/scope for existence checks from API, CLI, or extension callers | `cleanFilename` and `Scope.Validate` via `pathFor` / `dirForScope` | `os.Stat` inside the scope-local memory directory | LOW — rejected; same validated local-path boundary as `Read`. |
| `internal/memory/store.go:137` | Filename/scope for delete requests from API, CLI, or extension callers | `cleanFilename` and `Scope.Validate` via `pathFor` / `dirForScope` | `os.Remove` plus `MEMORY.md` index rewrite | LOW — rejected; local file deletion remains scoped, and the index rewrite now targets only the deleted entry's first markdown-link destination. |
| `internal/memory/document.go:9` | Raw content passed into header parsing from upstream write-scope resolution | Strict frontmatter decode (`yaml.Strict`) and `Header.Validate` | Returned `Header` metadata used by upstream scope selection | LOW — rejected; this is typed local parsing, not code execution or a deserialization sink. |
| `internal/memory/consolidation/runtime.go:161` | Dream-trigger workspace ref from daemon/API callers (`internal/api/core/memory.go:133`) | `strings.TrimSpace`, `ResolvePath` for path-like refs, then resolver-backed `Resolve` / `ResolveOrRegister` | Workspace selection for dream session spawning | LOW — rejected; workspace resolution is delegated to the trusted resolver surface and the package does not execute shell/file operations directly from the raw ref. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `MEM-REF-001` | refactoring-analysis | medium | `internal/memory/store.go:306` | Index pruning matched any line containing the deleted filename, so deleting one memory could remove unrelated `MEMORY.md` entries that only mentioned that filename in the description. | fixed |
| `MEM-OPT-001` | extreme-software-optimization | medium | `internal/memory/store.go:156` | `Store.Scan` read and parsed every memory file before capping results at 200, inflating latency and allocations on the API/extension list path. | fixed |
| `MEM-REF-002` | refactoring-analysis | medium | `internal/memory/consolidation/runtime.go:39` | `runtime.go` remains a large unit that mixes runtime lifecycle, queueing, workspace discovery, and dream-session spawning. | deferred |

## Per-Skill Notes

### refactoring-analysis

- The test-only duplication from `dupl` was left alone because extracting it would not change production maintenance cost.
- The targeted correctness fix replaced the brittle substring search in `removeIndexEntry` with explicit parsing of the first markdown-link target, including balanced-parenthesis handling for filenames such as `user(preferences).md`, which closes the regression tests without changing the on-disk index format.
- `internal/memory/consolidation/runtime.go` still carries multiple responsibilities. Splitting that file safely would require a broader refactor of the runtime/session-spawning path than this package pass should take on.

### extreme-software-optimization

- Added `internal/memory/perf_bench_test.go` and `internal/memory/consolidation/perf_bench_test.go` before changing production code so the package had measured baselines.
- `Store.Scan` now sorts lightweight metadata first and only reads/parses as many files as needed to return 200 valid headers. On the benchmarked capped workspace path, median latency improved from `13.22 ms` to `5.71 ms`, heap volume dropped from `5.67 MB/op` to `2.47 MB/op`, and alloc count dropped from roughly `108k` to `44k` per scan.
- The other measured candidates stayed effectively flat, so no further performance refactor was justified in this pass.

### ubs

- `not-run` due missing skill-runner support in this session; no CLI/manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete. The package has one bounded background goroutine in the consolidation runtime plus three mutexes and one buffered check queue.
- All long-lived blocking selects are either context-aware (`runtime.go:148`) or intentionally non-blocking (`runtime.go:175`), so no package-local deadlock or goroutine-leak path was confirmed.

### security-review

- The package is a local filesystem/session orchestration layer, not a network-facing surface.
- No HIGH-confidence or MEDIUM-confidence vulnerability survived the source-to-sink review. The main attacker-controlled surfaces are filenames, raw markdown content, and dream-trigger workspace refs, all of which stay confined to validated local paths or resolver-backed workspace selection.
- The corrected index-pruning bug is treated as a data-integrity/correctness issue rather than a reportable security exploit because it does not cross the stated trust boundaries.

## Deferred Items (carry forward)

- `MEM-REF-002` — Splitting `internal/memory/consolidation/runtime.go` into smaller lifecycle/workspace/session-spawn units would likely improve readability, but it would introduce broader structural churn with little immediate payoff after the targeted correctness/perf fixes landed.

## `make verify`

Final command: `make verify`

```text
0 issues.
✓  internal/memory/consolidation
✓  internal/memory
✓  internal/hooks
✓  internal/acp
✓  internal/cli
✓  internal/extension
✓  internal/daemon

DONE 4486 tests
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the successful run:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building `golangci-lint`.
