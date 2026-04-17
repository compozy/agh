# Improvements Report — internal/filesnap

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/filesnap/filesnap_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/filesnap | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 6 | `TestFromPath` | `internal/filesnap/filesnap_test.go:11` |
| 6 | `Equal` | `internal/filesnap/filesnap.go:30` |
| 5 | `TestCloneReturnsIndependentCopy` | `internal/filesnap/filesnap_test.go:88` |
| 4 | `TestEqual` | `internal/filesnap/filesnap_test.go:44` |
| 2 | `FromPath` | `internal/filesnap/filesnap.go:17` |
| 2 | `Clone` | `internal/filesnap/filesnap.go:52` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/filesnap/` exceeds 300 LOC. `internal/filesnap/filesnap.go` is 60 LOC.

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 20 internal/filesnap` before fixes:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/filesnap/filesnap_test.go:52-55` | `internal/filesnap/filesnap_test.go:64-67` | Repeated `Equal` fixture setup in adjacent subtests. |
| `internal/filesnap/filesnap_test.go:64-67` | `internal/filesnap/filesnap_test.go:76-79` | Same repeated fixture block. |

Post-fix note: the repeated `Equal` fixtures were collapsed into `equalTestSnapshots`; the only remaining `dupl` hit is a trivial 3-line `os.WriteFile` error check shared between the benchmark and `TestFromPath`.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `FromPath` | `internal/filesnap/filesnap.go:17` | Filesystem stat boundary used by workspace and skills scans whenever snapshot state is refreshed. | `BenchmarkFromPath` |
| `Equal` | `internal/filesnap/filesnap.go:30` | Cache reuse checks in workspace and skills packages compare snapshot maps on reload paths. | `BenchmarkEqual` |
| `Clone` | `internal/filesnap/filesnap.go:52` | Snapshot maps are cloned when cache state is captured for reuse by workspace and skills loaders/watchers. | `BenchmarkClone` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/filesnap/...` before the test cleanup change.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/filesnap/...` after all `internal/filesnap/` changes landed.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkFromPath` | 1201.8 | 304 | 1194.8 | 304 | not-hot-confirmed-by-benchmark |
| `BenchmarkEqual` | 390.6 | 0 | 398.3 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkClone` | 822.5 | 3240 | 813.8 | 3240 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/filesnap/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/filesnap/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/filesnap/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/filesnap/`.

### Security — Threat Model

- Trust boundaries:
  - Internal workspace and skills packages call `filesnap` with filesystem paths and snapshot maps.
  - `filesnap` itself only reads file metadata or compares/copies in-memory snapshot maps.
- Attacker capabilities:
  - A local caller can influence filesystem paths that eventually reach `FromPath`.
  - A local caller can provide snapshot maps derived from previously scanned filesystem state to `Equal` and `Clone`.
- In-scope assets:
  - Correct cache invalidation decisions based on file metadata.
  - Avoiding unintended side effects while reading file metadata.
  - Integrity of cloned/compared snapshot maps used by workspace and skills caches.
- Out-of-scope:
  - The host OS permission model and any race between `os.Stat` and later filesystem use in callers.
  - A malicious privileged local operator with direct control over the repository checkout or runtime filesystem.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/filesnap/filesnap.go:17` | Local path strings forwarded from workspace scanning and skills loading (`internal/workspace/scanner.go:229`, `internal/skills/loader.go:192`, `internal/skills/registry_workspace_cache.go:91`, `internal/skills/watcher.go:203`) | No package-local normalization; callers constrain discovery scope and `os.Stat` enforces filesystem permissions. | `os.Stat(path)` metadata read only. | LOW — rejected; this is a local metadata read surface with no command execution, path rewriting, or content parsing in `filesnap`. |
| `internal/filesnap/filesnap.go:30` | Snapshot maps produced by internal caches before equality checks (`internal/workspace/resolver.go:276`, `internal/skills/registry.go:174`, `internal/skills/registry.go:259`) | Type-constrained `map[string]Snapshot` input only. | In-memory key/value comparison. | LOW — rejected; no I/O, mutation, or trust-boundary crossing occurs inside `Equal`. |
| `internal/filesnap/filesnap.go:52` | Snapshot maps captured from internal loaders and cache state (`internal/workspace/clone.go:12`, `internal/skills/registry.go:263`, `internal/skills/registry.go:282`) | Type-constrained `map[string]Snapshot` input only. | In-memory map allocation/copy. | LOW — rejected; `Clone` only copies already-typed snapshot state and does not amplify caller control. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `FILESNAP-REF-001` | refactoring-analysis | low | `internal/filesnap/filesnap_test.go:44-129` | `TestEqual` repeated the same fixture setup across multiple subtests and still missed the same-size/different-key branch. | fixed |

## Per-Skill Notes

### refactoring-analysis

- Extracted the repeated `TestEqual` fixture setup into `equalTestSnapshots`, which resolved the only non-trivial duplication found in the package.
- Added the missing same-size/different-key subtest so the negative branch at `internal/filesnap/filesnap.go:36-38` is covered.
- Package coverage improved from `90.0%` to `95.0%` via `go test ./internal/filesnap -cover`.

### extreme-software-optimization

- Added `internal/filesnap/filesnap_bench_test.go` before assessing the package so every candidate had baseline and final benchmark data.
- `FromPath` is syscall-bound, while `Equal` and `Clone` are already small, allocation-light helpers. The before/after deltas were noise-level changes caused by rerunning the same code, so no production optimization was justified.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package has no goroutines, channels, mutexes, or `select` statements, so no concrete deadlock or leak path exists inside `internal/filesnap/`.

### security-review

- The package is an internal metadata utility with no network-facing surface. Every inspected input path stayed within local filesystem metadata reads or typed in-memory map operations.
- No HIGH-confidence or MEDIUM-confidence source-to-sink security issue survived the threat-model review.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/filesnap (cached)

DONE 4468 tests in 0.675s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final rerun after the `internal/filesnap` changes and task-tracking updates.
