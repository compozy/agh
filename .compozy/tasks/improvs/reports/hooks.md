# Improvements Report — internal/hooks

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/hooks/dispatch_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/hooks | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `TestAllEventDescriptorsReturnsFullTaxonomy` | `internal/hooks/introspection_test.go:80` |
| 17 | `sanitizedHookDecl` | `internal/hooks/normalize.go:122` |
| 15 | `TestDispatchPermissionAndContextHooksApplyPatches` | `internal/hooks/hooks_test.go:1414` |
| 13 | `resolveHookExecutorKind` | `internal/hooks/normalize.go:212` |
| 13 | `executeDispatch` | `internal/hooks/dispatch.go:678` |
| 13 | `TestDispatchToolHooksApplyPatches` | `internal/hooks/hooks_test.go:1309` |
| 12 | `TestNewHooksAppliesOptionsAndDefaultResolver` | `internal/hooks/hooks_test.go:1735` |
| 11 | `catalogHookMatchesFilter` | `internal/hooks/introspection.go:378` |
| 11 | `TestHooksCloseDrainsAsyncPool` | `internal/hooks/hooks_test.go:1662` |
| 11 | `TestHookTelemetrySecurityPatchPersistsAllFields` | `internal/hooks/telemetry_test.go:12` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/hooks/dispatch.go` | 1073 | Large typed dispatch surface with repeated wrapper boilerplate around one generic execution path. |
| `internal/hooks/payloads.go` | 728 | Broad schema catalog that mixes multiple event families and clone helpers in one file. |
| `internal/hooks/matcher.go` | 543 | High event-family fan-out with repeated `Matches*` and `match*` adapters. |
| `internal/hooks/hooks.go` | 441 | Runtime construction, reload, fingerprinting, and option wiring share one large unit. |
| `internal/hooks/introspection.go` | 417 | Taxonomy catalog plus filter logic concentrated in one file. |
| `internal/hooks/telemetry.go` | 332 | Metrics state, persistence routing, and patch retention rules are coupled in one file. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 20 internal/hooks` found the following production duplicates at or above the reporting threshold:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/hooks/hooks.go:85-111` | `internal/hooks/hooks.go:128-153` | Repeated `With*DeclarationProvider` / `With*Declarations` option wiring blocks. |
| `internal/hooks/matcher.go:278-300` | `internal/hooks/matcher.go:350-372` | Repeated event-family match adapter wrappers with only payload type differences. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `executeDispatch` | `internal/hooks/dispatch.go:678` | Central message-handling loop for every typed dispatch; currently clones and re-sorts sync hook slices before executing the pipeline. | `BenchmarkDispatchInputPreSubmitSync` |
| `submitAsyncHook` | `internal/hooks/dispatch_async.go:29` | Async goroutine handoff path runs for every async match and captures payload state before background execution. | `BenchmarkSubmitAsyncHookInputPreSubmit` |
| `subprocessProcessEnv` | `internal/hooks/executor_subprocess.go:177` | Rebuilds allowlisted subprocess environments on every subprocess hook execution. | `BenchmarkSubprocessProcessEnv` |

### Optimization — Benchmark Results

Baseline `before` command: `go test ./internal/hooks -run '^$' -bench 'Benchmark(DispatchInputPreSubmitSync|SubmitAsyncHookInputPreSubmit|SubprocessProcessEnv)$' -benchmem -count=5`

Median values below come from `/tmp/hooks-bench-before.txt` and `/tmp/hooks-bench-after.txt` after 5 runs each.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkDispatchInputPreSubmitSync` | 10198 | 9108 | 10143 | 8955 | fixed-with-benchmark |
| `BenchmarkSubmitAsyncHookInputPreSubmit` | 351.5 | 1488 | 649.6 | 2160 | not-hot-confirmed-by-benchmark |
| `BenchmarkSubprocessProcessEnv` | 1468 | 3768 | 1469 | 3768 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/hooks/executor_subprocess.go:155` | `runSubprocessCommand` | Returns when `cmd.Wait` resolves; cancellation path still drains `waitCh` before exit. | Wait helper bridges blocking `Wait` into timeout-aware `select` logic. |
| `internal/hooks/pool.go:102` | `asyncPool.Start` | Worker exits on `ctx.Done()` or closed task channel; tracked with `WaitGroup`. | One goroutine per configured worker. |
| `internal/hooks/pool.go:169` | `asyncPool.Close` | Exits after `wg.Wait` completes and closes `done`. | Close-time waiter that bounds drain behavior. |
| `internal/hooks/pool.go:252` | `asyncPool.Start` via `asyncPoolContext` | Exits on `stopCh` close or parent-context cancellation. | Cancels worker context when pool stops. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/hooks/executor_subprocess.go:154` | 1 | `runSubprocessCommand` | none | same function via `select` | Carries `cmd.Wait` result out of the waiter goroutine. |
| `internal/hooks/pool.go:90` | 0 | `asyncPool.Start` | `stopWorkers` | `asyncPoolContext` | Stop signal for pool-scoped cancellation. |
| `internal/hooks/pool.go:92` | `queueCapacity` | `asyncPool.Start` | `asyncPool.Close` | `worker`, `discardAsyncTasks` | Async hook work queue. |
| `internal/hooks/pool.go:168` | 0 | `asyncPool.Close` | close-wait goroutine | same function via `select` | Signals `wg.Wait` completion during shutdown. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/hooks/hooks.go:22` | read-heavy (`sync.RWMutex`) | `snapshot` and `fingerprint` live registry state | Snapshot map is swapped atomically during rebuilds and read during dispatch/catalog access. |
| `internal/hooks/pool.go:36` | mixed (`sync.RWMutex`) | async pool lifecycle fields (`ctx`, `stopCh`, `tasks`, `started`, `closed`) | Submit fast path uses `RLock`; start/close use `Lock`. |
| `internal/hooks/telemetry.go:32` | write-heavy (`sync.Mutex`) | in-memory metrics counters and latency totals | Small shared metrics map state. |

### Concurrency — Select Audit

All long-lived blocking selects are context-aware. Remaining selects without `ctx.Done()` are bounded by default branches or post-cancellation grace windows:

| File:Line | Reasoning |
| --- | --- |
| `internal/hooks/executor_subprocess.go:167` | Entered only after parent context cancellation; bounded by `waitCh` or the graceful-shutdown timer. |
| `internal/hooks/pool.go:117` | Non-blocking submit fast path uses `default` intentionally for queue-drop behavior. |
| `internal/hooks/pool.go:174` | Shutdown wait races `done` against a drain timeout after the task channel is already closed. |
| `internal/hooks/pool.go:223` | Non-blocking discard loop drains any remaining buffered tasks until empty. |

### Security — Threat Model

- Trust boundaries:
  - Internal session, daemon, automation, and extension layers call `internal/hooks` with structured runtime payloads.
  - Hook declarations come from native registration, config, agent definitions, or skills and are normalized/bound inside this package.
  - Subprocess hooks cross a local process boundary via `exec.Cmd`; native hooks stay in-process.
- Attacker capabilities:
  - A malicious extension or local operator who controls hook declarations can choose matchers, executor kinds, subprocess commands, args, and env overrides.
  - A malicious hook implementation can return arbitrary patch JSON within the declared patch schema.
  - Remote users do not call `internal/hooks` directly; they only influence higher-level runtime payload content that is forwarded to already-configured hooks.
- In-scope assets:
  - Integrity of dispatch ordering, permission decisions, and telemetry records.
  - Safety of subprocess execution boundaries and forwarded payloads.
  - Stability of async dispatch state handed off to background hooks.
- Out-of-scope:
  - A fully compromised local host or privileged operator who already controls the runtime binary/config.
  - Sandboxing of the hook executable itself after `exec.Cmd` launches it.
  - Validation of upstream session/daemon payload semantics outside this package.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/hooks/normalize.go:58` | Hook declarations from config, agent definitions, skills, or native registration | `sanitizedHookDecl`, `resolveHookExecutorKind`, `ValidateMatcherForEvent`, `validateMatcherPatterns` | `BuildBindingState` -> executor binding / matcher runtime | LOW — rejected; declaration authors are trusted extension/config operators in this threat model, and the package validates executor kind and matcher shape before binding. |
| `internal/hooks/executor_subprocess.go:84` | Declaration-controlled subprocess command, args, working dir, and env overrides | `execabs.LookPath`, trimmed working dir, fixed allowlist + explicit env merge, no shell invocation | `exec.Cmd{Path, Args, Dir, Env}` in `runSubprocessCommand` | LOW — rejected; execution is direct `exec`, not shell interpolation, and only trusted hook declarations reach this surface. |
| `internal/hooks/pipeline.go:172` | Hook patch bytes returned by native/subprocess executors | JSON decode into typed patch surface; `newPermissionRequestGuard` blocks deny→allow escalation on permission hooks | `apply*Patch` functions and `emitHookRun` persistence path | LOW — rejected; patch influence is constrained to typed fields, and the only privilege-escalation class in-scope is explicitly guarded. |
| `internal/hooks/dispatch.go:678` | Runtime payload content forwarded from session/daemon/automation layers | No sanitization in `internal/hooks`; payloads are treated as data, not commands | Native hook callbacks or subprocess stdin via `Executor.Execute` | LOW — rejected; the package forwards structured data to already-authorized hook executors and does not interpret payload bytes as code or shell fragments. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `HOOKS-CONC-001` | deadlock-finder-and-fixer | high | `internal/hooks/dispatch_async.go:29` | Async hook submission shallow-copies payload structs, so slices/maps/raw JSON alias caller state after dispatch returns. | fixed |
| `HOOKS-PERF-001` | extreme-software-optimization | medium | `internal/hooks/pipeline.go:58` | The sync pipeline clones and re-sorts slices that are already ordered by the registry snapshot on every dispatch. | fixed |
| `HOOKS-REF-001` | refactoring-analysis | medium | `internal/hooks/dispatch.go:1` | `dispatch.go` is a 1k+ LOC large unit dominated by typed wrapper boilerplate around one generic execution path. | deferred |
| `HOOKS-REF-002` | refactoring-analysis | medium | `internal/hooks/types.go:139` | `HookMatcher.AgentType` is normalized and pattern-validated but never allowed for any family or consulted during matching. | deferred |

## Per-Skill Notes

### refactoring-analysis

- The package has several large non-test files, but most of the size comes from intentionally explicit typed surfaces and schema catalogs.
- `HookMatcher.AgentType` appears to be drifted API surface inside `internal/hooks`: it is declared, normalized, and validated as a pattern, but no event family accepts it and no matcher path reads it.
- Package coverage recovered to `82.6%` after adding focused tests for the new async snapshot helper, which is above the pre-fix baseline and above the package target.

### extreme-software-optimization

- Benchmarks were added before production changes in `internal/hooks/dispatch_bench_test.go`.
- `pipeline.executeWithDisposition` now uses `orderedResolvedHooksIfNeeded`, which keeps already-sorted registry slices on the hot path and only clones/sorts when a caller hands the pipeline an unordered slice.
- `BenchmarkDispatchInputPreSubmitSync` improved from `9108 B/op` to `8955 B/op` and from `108 allocs/op` to `105 allocs/op`, while median latency stayed in the same band (`10198 ns/op` -> `10143 ns/op`).
- `BenchmarkSubmitAsyncHookInputPreSubmit` regressed after the async snapshot fix because the package now deep-clones reference fields before background handoff; the final run measured `649.6 ns/op`, `2160 B/op`, and `7 allocs/op`, which is treated as an acceptable correctness cost rather than a performance win.
- `subprocessProcessEnv` stayed flat, so no subprocess-env optimization was justified.

### ubs

- `not-run` due missing skill-runner support in this session; no CLI/manual substitute was used.

### deadlock-finder-and-fixer

- No deadlock cycle or goroutine leak was found in the current inventories, but the async handoff path exposes mutable shared payload state to background workers.
- The goroutine inventory is small and bounded; all long-lived worker goroutines have explicit cancellation or channel-close shutdown paths.
- Added `TestDispatchInputPreSubmitAsyncHookSeesStablePayloadSnapshot` plus broader clone-helper coverage so async hooks now observe immutable dispatch-time payload snapshots instead of caller-mutated state.

### security-review

- The package is an internal dispatch boundary, not a remote API surface.
- No HIGH-confidence or MEDIUM-confidence exploit path survived the threat model review.
- The async payload-aliasing issue is tracked as a correctness/concurrency finding rather than a standalone reportable security issue because it does not create a new trust-boundary bypass on its own.

## Deferred Items (carry forward)

- `HOOKS-REF-001` — Splitting `dispatch.go` into generated or taxonomy-driven dispatch surfaces would be broader than this task and risks introducing new abstractions; it should be handled as a follow-up design refactor.
- `HOOKS-REF-002` — Fixing or removing `HookMatcher.AgentType` requires a package/API decision across payload producers and generated schema surfaces, not just a local tweak.

## `make verify`

Command executed after the last code change: `make verify`

Excerpt from the passing run:

```text
0 issues.
✓  internal/hooks (1.833s)
✓  internal/memory (7.835s)
✓  internal/skills (10.824s)
✓  internal/observe (15.749s)
✓  internal/cli (18.507s)
✓  internal/session (19.194s)
✓  internal/store/globaldb (21.318s)
✓  internal/extension (23.468s)
✓  internal/daemon (23.931s)

DONE 4483 tests in 26.359s
OK: all package boundaries respected
```

Warnings emitted during the successful run came from existing toolchain noise outside `internal/hooks/`:

- Node printed `The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.`
- The macOS linker printed `ld: warning: -bind_at_load is deprecated on macOS` while compiling `golangci-lint`.
