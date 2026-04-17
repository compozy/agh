# Improvements Report — internal/resources

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/resources/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/resources | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 18 | `TestKernelSnapshotSequenceConflictAndResetIntegration` | `internal/resources/kernel_integration_test.go:12` |
| 16 | `NewReconcileDriver` | `internal/resources/reconcile.go:258` |
| 15 | `buildReconcileTopology` | `internal/resources/reconcile.go:350` |
| 15 | `TestTypedStoreReadAuthorityBoundaries` | `internal/resources/typed_test.go:157` |
| 15 | `TestKernelValidationAndAuthorityEdgeCases` | `internal/resources/kernel_test.go:690` |
| 15 | `TestBundleActivationProjectorRegistrationIntegration` | `internal/resources/typed_integration_test.go:61` |
| 14 | `TestBundleActivationProjectorRegistrationDecodesDependenciesExplicitly` | `internal/resources/typed_test.go:413` |
| 13 | `TestTypedStorePutRoundTripPreservesMetadata` | `internal/resources/typed_test.go:299` |
| 13 | `TestKernelAdditionalValidationBranches` | `internal/resources/kernel_test.go:946` |
| 13 | `(*Kernel).ListRaw` | `internal/resources/kernel.go:385` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/resources/kernel.go` | 1370 | Raw CRUD, snapshot sequencing, SQL helpers, row decoding, and source-lock lifecycle all live in one file, making persistence and coordination concerns hard to scan independently. |
| `internal/resources/reconcile.go` | 1045 | Reconcile topology construction, queue/scheduler state, failure handling, event emission, and health reporting are co-located in one large scheduler unit. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 25 internal/resources/*.go` (production-only pairs):

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/resources/kernel.go:247-259` | `internal/resources/kernel.go:260-272` | Adjacent `ExecContext` delete branches for source records and source state repeat the same structure with only query/labels changed. |
| `internal/resources/validate.go:205-223` | `internal/resources/validate.go:225-243` | `normalizeKinds` and `normalizeScopeKinds` are structurally duplicated typed normalization loops. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `buildListRawQuery` | `internal/resources/kernel.go:733` | Every `Kernel.ListRaw` call builds this SQL and argument slice before hitting SQLite. It sits on the typed store read path (`internal/resources/typed.go:112`), API control surface (`internal/api/core/resources.go:72`), extension host API read surface (`internal/extension/host_api_resources.go:27`), and reconcile input loading (`internal/resources/reconcile.go:758`). | `BenchmarkBuildListRawQuery` |
| `ValidateAndCanonicalizeIfRegistered` | `internal/resources/codec.go:222` | This is the raw ingress canonicalization hook for extension snapshot publication and API writes before persistence (`internal/extension/host_api_resources.go:82`, `internal/api/core/resources.go:91`). | `BenchmarkValidateAndCanonicalizeIfRegistered` |
| `(*reconcileDriver).scheduleCascade` | `internal/resources/reconcile.go:539` | Every reconcile trigger expands dependency order through this path before work is queued (`internal/resources/reconcile.go:453`). | `BenchmarkReconcileScheduleCascade` |
| `(*reconcileDriver).buildProjectionInput` | `internal/resources/reconcile.go:753` | Every reconcile pass lists primary and dependency records through this pre-projector load path (`internal/resources/reconcile.go:723`). | `BenchmarkReconcileBuildProjectionInput` |
| `(*Kernel).ListRaw` | `internal/resources/kernel.go:385` | This is the package’s shared raw read loop for typed stores, control APIs, extension-host resource listing, and reconcile record loading. | `BenchmarkKernelListRaw` |
| `(*Kernel).ApplySourceSnapshotRaw` | `internal/resources/kernel.go:432` | This is the extension snapshot write loop, including JSON normalization, source locking, SQLite transaction work, and snapshot-diff application (`internal/extension/host_api_resources.go:103`). | `BenchmarkKernelApplySourceSnapshotRaw` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/resources/...` before production fixes.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/resources/...` in the current workspace after all changes.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkBuildListRawQuery` | 234.4 | 864 | 229.4 | 864 | not-hot-confirmed-by-benchmark |
| `BenchmarkValidateAndCanonicalizeIfRegistered` | 371.9 | 312 | 367.2 | 312 | not-hot-confirmed-by-benchmark |
| `BenchmarkReconcileScheduleCascade` | 481.2 | 568 | 482.6 | 568 | not-hot-confirmed-by-benchmark |
| `BenchmarkReconcileBuildProjectionInput` | 1732.0 | 10688 | 1803.0 | 10688 | not-hot-confirmed-by-benchmark |
| `BenchmarkKernelListRaw` | 92637.0 | 76078 | 91945.0 | 76077 | not-hot-confirmed-by-benchmark |
| `BenchmarkKernelApplySourceSnapshotRaw` | 83867.0 | 20428 | 84239.0 | 20471 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/resources/reconcile.go:327` | `reconcileDriver` | `workerCancel()` cancels `workerCtx`; `run()` exits when `shouldExit()` observes `closed` and no running states, then closes `doneCh`. | Sole production goroutine; owns the reconcile work loop. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/resources/reconcile.go:232` | 1 | `reconcileDriver` | Never closed; non-blocking signal channel written by `notify`/`notifyLocked`. | `waitForWork` | Internal wake-up signal for queued reconcile work. |
| `internal/resources/reconcile.go:233` | 0 | `reconcileDriver` | `run()` closes `doneCh` on exit. | `Close` | Completion signal used by `Close` to await the worker loop. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/resources/kernel.go:126` | write-heavy | `Kernel.sourceLocks` map | Serializes access to the per-source lock registry. |
| `internal/resources/kernel.go:133` | write-heavy | One source’s snapshot/session critical section | `sourceLock` now tracks both the mutex and in-use reference count so idle source entries can be removed safely. |
| `internal/resources/codec.go:133` | read-heavy | `CodecRegistry.codecs` | Guards typed codec registration and resolution. |
| `internal/resources/reconcile.go:220` | write-heavy | `closed`, `queue`, and `kindStates` | Core scheduler state mutex for trigger/queue/close transitions. |

### Concurrency — Select Audit

| File:Line | Notes |
| --- | --- |
| `internal/resources/reconcile.go:522` | `Close` waits on `doneCh` or `ctx.Done()`. |
| `internal/resources/reconcile.go:675` | `waitForWork` is context-aware when there is no timer delay. |
| `internal/resources/reconcile.go:690` | `waitForWork` multiplexes timer expiry, `notifyCh`, and `ctx.Done()`. |
| `internal/resources/reconcile.go:1033` | Non-blocking wake-up send with `default`; intentionally input-bounded, not context-driven. |
| `internal/resources/reconcile.go:1040` | Same non-blocking wake-up send used while the scheduler mutex is held. |

### Security — Threat Model

- Trust boundaries:
  - `internal/resources` sits behind the daemon/API control plane for raw CRUD (`internal/api/core/resources.go:72-128`) and behind the extension host API for extension-scoped reads and snapshot publication (`internal/extension/host_api_resources.go:27-103`).
  - The package also sits beneath typed resource adapters used by daemon subsystems such as bundles, tools, skills, automation, and bridges.
- Attacker capabilities:
  - A malicious extension session can control resource IDs, scope selections, and JSON specs passed through the host API, but only within the actor grants established by the caller context.
  - An operator or API caller can control raw draft/filter fields and expected versions passed through the daemon control surface.
  - Callers do not control SQL text directly; they only supply values consumed by normalization, validation, and parameterized SQLite statements.
- In-scope assets:
  - Integrity of the `resource_records` and `resource_source_state` tables.
  - Scope/source/owner enforcement for reads, direct mutations, and source snapshots.
  - Reconcile scheduler availability and correctness for queued resource projections.
  - Payload-size and snapshot-size enforcement on JSON resource specs.
- Out-of-scope:
  - Authorization policy that determines whether a caller should receive a daemon/operator/extension `MutationActor` in the first place.
  - SQLite engine correctness and external package callers that intentionally execute privileged daemon code paths.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/api/core/resources.go:72-128` | Operator/API request query params and JSON payloads become `ResourceFilter` / `RawDraft` inputs for raw CRUD. | `normalizeActor`, `normalizeFilter`, `normalizeDraft`, `validateActorReadAccess`, `validateActorWriteAccess`, `normalizeJSON`, and optimistic-version checks. | Parameterized SQLite reads/writes in `Kernel.GetRaw`, `Kernel.ListRaw`, `Kernel.PutRaw`, and `Kernel.DeleteRaw`. | LOW — rejected; the package validates fields and uses positional SQL parameters, leaving no confirmed injection sink. |
| `internal/extension/host_api_resources.go:27-55` | Extension host API read requests control resource kind/id/scope filters. | `normalizeActor`, `normalizeFilter`, actor kind/scope/source checks, and `validateActorReadAccess`. | `Kernel.GetRaw` / `Kernel.ListRaw` SQLite reads and typed decode paths. | LOW — rejected; reads are filtered through actor grants and parameterized SQL. |
| `internal/extension/host_api_resources.go:82-103` | Extension snapshot publication controls kind/id/scope/spec JSON for each record. | `ValidateAndCanonicalizeIfRegistered`, `normalizeSnapshot`, `normalizeDraft`, per-record JSON validation/size caps, duplicate-key rejection, source-session nonce checks, and actor write-authority checks. | `Kernel.ApplySourceSnapshotRaw` transactional snapshot write loop. | LOW — rejected; the package enforces JSON validity, size ceilings, source ownership, and monotonically increasing source versions before persistence. |
| `internal/extension/manager.go:1368-1380` | Session activation passes source ID and nonce into `ActivateSourceSession`. | Source normalization/validation plus blank-nonce rejection inside `ActivateSourceSession`. | `resource_source_state` SQLite upsert. | LOW — rejected; inputs are normalized and written through parameterized SQL only. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `RES-REF-001` | refactoring-analysis | medium | `internal/resources/kernel.go:118` | `kernel.go` remains far above the package’s size heuristic and co-locates persistence, locking, and row-decoding concerns. | deferred |
| `RES-REF-002` | refactoring-analysis | medium | `internal/resources/reconcile.go:206` | `reconcile.go` mixes scheduler state, dependency expansion, worker coordination, and sink/reporting responsibilities in one large file. | deferred |
| `RES-CON-001` | deadlock-finder-and-fixer | high | `internal/resources/reconcile.go:425` | `Trigger` previously emitted `ReconcileEventRequested` and coalesced events while `d.mu` was held, so a re-entrant or blocking event sink could deadlock the scheduler. | fixed |
| `RES-CON-002` | deadlock-finder-and-fixer | medium | `internal/resources/kernel.go:1326` | `Kernel.sourceLocks` previously retained idle per-source mutex entries forever, so the registry grew monotonically over the daemon lifetime. | fixed |

## Per-Skill Notes

### refactoring-analysis

- `kernel.go` and `reconcile.go` remain the only non-test files above the 300-LOC heuristic, and both still merit future decomposition.
- The production duplication scan still shows only same-file/local structural duplication; none of it justified a new abstraction in this pass.
- The lock-registry cleanup fix stayed in `kernel.go` because it addressed a concrete lifecycle bug without forcing a speculative split.

### extreme-software-optimization

- Benchmarks were added in `internal/resources/perf_bench_test.go` before any production change.
- None of the benchmarked candidates justified a performance-targeted refactor in this pass; before/after medians stayed within noise for the pure hot-path candidates or moved slightly due to the correctness fixes.
- The after-run medians were: `229.4 ns/op` (`buildListRawQuery`), `367.2 ns/op` (`ValidateAndCanonicalizeIfRegistered`), `482.6 ns/op` (`scheduleCascade`), `1803 ns/op` (`buildProjectionInput`), `91945 ns/op` (`Kernel.ListRaw`), and `84239 ns/op` (`ApplySourceSnapshotRaw`).

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute will be used.

### deadlock-finder-and-fixer

- Fixed `RES-CON-001` by queueing reconcile events under `d.mu` and emitting them only after releasing the scheduler mutex. Regression coverage: `TestReconcileDriverEventSinkCanReenterTrigger`.
- Fixed `RES-CON-002` by turning per-source locks into ref-counted entries that are removed once no goroutine holds or waits on them. Regression coverage: `TestKernelLockSourceReleasesIdleEntries`.
- All production `select` statements remain either context-aware (`Close`, `waitForWork`) or intentionally non-blocking internal wake-up sends (`notify`, `notifyLocked`).

### security-review

- No HIGH-confidence or MEDIUM-confidence vulnerability survived the threat-model review; all identified surfaces remain low-risk/rejected with explicit validation and sink reasoning.
- The package continues to rely on parameterized SQLite statements, explicit JSON validity/size checks, scope/source/owner validation, and source-session nonce/version enforcement rather than ad-hoc sanitization.
- Coverage moved from `79.2%` to `79.4%` after the new regression tests, which lifts but does not yet clear the 80% package target.

## Deferred Items (carry forward)

- **`RES-REF-001`** — Split `kernel.go` only when there is appetite for a focused persistence-layer decomposition, not opportunistically inside this improvements pass.
- **`RES-REF-002`** — Split `reconcile.go` when the package is ready for a dedicated scheduler/event-sink decomposition rather than a one-off cosmetic extraction.

## `make verify`

Final command: `make verify`

Non-blocking environment/tooling warnings observed during this successful run:

- Node/Bun tooling printed `The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.`
- The vendored `golangci-lint` build emitted `ld: warning: -bind_at_load is deprecated on macOS`.

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/resources (1.407s)
✓  internal/config (1.437s)
✓  internal/hooks (1.63s)
✓  internal/acp (4.531s)
✓  internal/daemon (7.061s)
✓  internal/extension (7.964s)
✓  internal/cli (8.29s)

DONE 4491 tests in 10.216s
OK: all package boundaries respected
```
