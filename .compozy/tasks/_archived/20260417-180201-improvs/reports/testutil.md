# Improvements Report — internal/testutil

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/testutil/testutil_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo internal/testutil | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 5 | `TestContextIsCanceledOnCleanup` | `internal/testutil/testutil_test.go:12` |
| 5 | `FreeTCPPort` | `internal/testutil/testutil.go:38` |
| 5 | `BenchmarkEqualStringSlices` | `internal/testutil/testutil_bench_test.go:5` |
| 4 | `TestFreeTCPPort` | `internal/testutil/testutil_test.go:83` |
| 3 | `TestEqualStringSlices` | `internal/testutil/testutil_test.go:58` |
| 3 | `TestContextCreatedDuringCleanupRemainsUsable` | `internal/testutil/testutil_test.go:37` |
| 3 | `BenchmarkFreeTCPPort` | `internal/testutil/testutil_bench_test.go:46` |
| 2 | `makeSequenceStrings` | `internal/testutil/testutil_bench_test.go:57` |
| 1 | `EqualStringSlices` | `internal/testutil/testutil.go:29` |
| 1 | `Context` | `internal/testutil/testutil.go:20` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/testutil/` exceeds 300 LOC. The largest non-test file is `internal/testutil/testutil.go` at 72 LOC.

### Refactoring — Duplication

Baseline `dupl -threshold 8 internal/testutil` only reported short overlapping test/benchmark scaffolding and repeated `Fatalf`-style error blocks. No duplicated block reached an 8-line threshold that justified a new abstraction in this package.

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/testutil/testutil.go:48-50` | `internal/testutil/testutil_test.go:89-91` | Three-line positive-port guard and fatal path; too small to extract without obscuring the surrounding test/helper intent. |
| `internal/testutil/testutil.go:64-66` | `internal/testutil/testutil_test.go:94-96` | Three-line `Close` error handling blocks in production helper and test assertion; not a candidate for package-level abstraction. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `EqualStringSlices` | `internal/testutil/testutil.go:29` | Pure comparison loop used by 13 test files outside the package and the only CPU-only helper owned by `internal/testutil`. | `BenchmarkEqualStringSlices/*` |
| `FreeTCPPort` | `internal/testutil/testutil.go:38` | Test-server setup helper used by integration tests (`extensions/bridges/telegram/provider_test.go:1642`) and local package tests; performs repeated address formatting plus localhost bind/close IO. | `BenchmarkFreeTCPPort` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/testutil/...` on a temporary pre-change snapshot materialized from `HEAD` plus `internal/testutil/testutil_bench_test.go`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/testutil/...`

Values below use the median of 5 runs from `/tmp/testutil-bench-before.txt` and `/tmp/testutil-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkEqualStringSlices/equal-small-16` | 4.417 | 0 | 4.426 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkEqualStringSlices/equal-large-16` | 468.1 | 0 | 470.0 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkEqualStringSlices/mismatch-tail-16` | 471.0 | 0 | 471.1 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkEqualStringSlices/different-length-16` | 1.809 | 0 | 1.807 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkFreeTCPPort-16` | 10002 | 736 | 10022 | 736 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/testutil/testutil_test.go:20` | `TestContextIsCanceledOnCleanup` | Goroutine exits after `<-ctx.Done()` and signals `done`; parent test blocks on a bounded `select` with `time.After`. | Test-only lifecycle probe; no production `go` statements exist in `internal/testutil/`. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/testutil/testutil_test.go:16` | 0 | `TestContextIsCanceledOnCleanup` | Goroutine at `internal/testutil/testutil_test.go:20-23` | Parent test `select` at `internal/testutil/testutil_test.go:26-30` | Used to prove the context closes after subtest cleanup. |
| `internal/testutil/testutil_test.go:40` | 1 | `TestContextCreatedDuringCleanupRemainsUsable` | Cleanup callback at `internal/testutil/testutil_test.go:43-50` | Parent test read at `internal/testutil/testutil_test.go:53-55` | Buffered handoff so cleanup can report whether a cleanup-time helper context starts live. |

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/testutil/`.

The package does use one atomic counter at `internal/testutil/testutil.go:17` (`tcpPortCounter`) to spread `FreeTCPPort` scans across the port range; it does not span any blocking section or require lock-order analysis.

### Concurrency — Select Audit

All `select` statements in `internal/testutil/` are input-bounded test probes:

| File:Line | Audit result |
| --- | --- |
| `internal/testutil/testutil_test.go:26` | Bounded by `time.After(time.Second)` while waiting for a test-owned goroutine to close `done`. |

### Security — Threat Model

- Trust boundaries:
  - `internal/testutil` is only imported by same-repository tests and benchmarks; it does not expose HTTP, RPC, subprocess, filesystem, or database interfaces of its own.
  - The package delegates lifecycle semantics to `testing.TB` and localhost port selection to the local kernel via `net.ListenConfig.Listen`.
- Attacker capabilities:
  - Same-process test code can pass arbitrary `[]string` values to `EqualStringSlices`.
  - Same-process test code can call `Context` or `FreeTCPPort` at arbitrary times during a test run.
  - The package has no boundary where an unauthenticated remote actor can supply bytes, paths, URLs, commands, or environment overrides.
- In-scope assets:
  - Correct test-lifecycle cancellation semantics for `Context`.
  - Predictable, non-overlapping local port selection behavior in `FreeTCPPort`.
  - Absence of unsafe source-to-sink flows from test-controlled inputs into command execution, file access, or network destinations beyond localhost probing.
- Out-of-scope:
  - Malicious code already executing inside the same Go test process.
  - OS-level port exhaustion or local firewall policy outside the package.
  - Security properties of downstream code that later uses the returned context or port.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/testutil/testutil.go:20` | `testing.TB` implementation supplied by the Go test harness. | Bounded by `context.WithTimeout(context.Background(), 10s)` and canceled during test cleanup via `t.Cleanup(cancel)`. | `context.Context` returned to same-process test callers. | LOW — rejected; lifecycle handle only, no injection or privilege boundary. |
| `internal/testutil/testutil.go:29` | `left` and `right` slices from same-process test code. | None required; function performs read-only equality. | In-memory comparison via `slices.Equal`. | LOW — rejected; no sink beyond slice comparison and boolean return. |
| `internal/testutil/testutil.go:38` | `testing.TB` handle from same-process test code. | No caller-controlled address/path data is accepted; port candidates are generated internally from PID and an atomic counter. | Localhost bind probe via `net.ListenConfig.Listen` at `internal/testutil/testutil.go:60`. | LOW — rejected; fixed localhost destination with internally generated port numbers, so no SSRF/path/command surface exists. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `TESTUTIL-REF-001` | refactoring-analysis | low | `internal/testutil/testutil.go:29` | `EqualStringSlices` duplicated standard-library slice equality logic with a hand-rolled loop. | fixed |

## Per-Skill Notes

### refactoring-analysis

- The package remains intentionally small: one production file, one test file, and one benchmark file.
- The only worthwhile structural cleanup was removing the hand-rolled `[]string` equality loop in favor of `slices.Equal`, which preserves nil/empty semantics while deleting bespoke logic.
- No file-size or meaningful duplication finding justified a new abstraction.

### extreme-software-optimization

- Added `internal/testutil/testutil_bench_test.go` before changing behavior so the package had measured baselines.
- Benchmarks show `EqualStringSlices` is already allocation-free and sub-500ns even for 256-element inputs, while `FreeTCPPort` is dominated by localhost bind/close work rather than package-local arithmetic.
- Because neither candidate showed a meaningful package-local optimization opportunity, all optimization outcomes were triaged as `not-hot-confirmed-by-benchmark`.

### ubs

- `not-run` due missing skill-runner support in this session; no manual UBS substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the only concurrency-relevant production state is the atomic `tcpPortCounter`.
- No package-local deadlock, goroutine leak, or select starvation path survived the audit.
- Repo-wide verification showed that `Context(t)` is intentionally used inside cleanup callbacks across the workspace, so the package keeps its existing background-rooted timeout semantics and now has a dedicated regression test (`TestContextCreatedDuringCleanupRemainsUsable`) to lock that contract in place.

### security-review

- The package has no external exposure beyond same-process test callers and a localhost port probe.
- No HIGH-confidence or MEDIUM-confidence source-to-sink vulnerability survived the threat-model review.
- All inspected surfaces terminate in lifecycle helpers, in-memory comparison, or localhost listen probes, not command/file/SQL/HTTP sinks.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
      Tests  677 passed (677)
0 issues.
✓  internal/testutil (1.2s)
✓  internal/daemon (24.114s)

DONE 4512 tests in 27.973s
OK: all package boundaries respected
```

Observed non-fatal environment/toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final rerun after the package changes and report updates.
