# Improvements Report — internal/procutil

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/procutil/procutil_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/procutil`:

| Complexity | Function | File |
| --- | --- | --- |
| 6 | `Signal` | `internal/procutil/procutil_windows.go:37` |
| 5 | `signalZero` | `internal/procutil/procutil_windows.go:63` |
| 4 | `Alive` | `internal/procutil/procutil_windows.go:18` |
| 3 | `TestAliveRejectsNonPositivePIDs` | `internal/procutil/procutil_test.go:19` |
| 3 | `BenchmarkSignalCurrentProcessZero` | `internal/procutil/procutil_bench_test.go:20` |
| 3 | `BenchmarkAliveCurrentProcess` | `internal/procutil/procutil_bench_test.go:9` |
| 3 | `Signal` | `internal/procutil/procutil.go:23` |
| 3 | `Alive` | `internal/procutil/procutil.go:13` |
| 2 | `TestSignalReturnsErrorForMissingProcess` | `internal/procutil/procutil_test.go:49` |
| 2 | `TestSignalRejectsNonPositivePID` | `internal/procutil/procutil_test.go:41` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/procutil/` exceeds 300 LOC. Largest non-test file is `internal/procutil/procutil_windows.go` at 81 LOC.

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 20 internal/procutil` returned no duplicated block at the configured threshold. No duplication reached an 8-line threshold that justified a new abstraction in this package.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `Alive` | `internal/procutil/procutil.go:13` | Exported liveness probe used on daemon-info, daemon-lock, and consolidation-lock PID paths (`internal/cli/daemon.go:308`, `internal/daemon/lock.go:90`, `internal/memory/lock.go:214`). These checks can sit in startup and stale-lock polling paths, so they need measurement instead of assumption. | `BenchmarkAliveCurrentProcess` |
| `Signal` with `syscall.Signal(0)` | `internal/procutil/procutil.go:23` | Exported probe/signal wrapper used by CLI daemon stop and daemon orphan cleanup (`internal/cli/daemon.go:133`, `internal/daemon/orphan.go:40-48`). The zero-signal path is the package's only benchmark-safe signal flow and needed evidence before any optimization claim. | `BenchmarkSignalCurrentProcessZero` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/procutil/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/procutil/...`

Values below use the median of 5 runs from `/tmp/procutil-bench-before.txt` and `/tmp/procutil-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkAliveCurrentProcess` | 206.1 | 0 | 206.9 | 0 | not-hot-confirmed-by-benchmark |
| `BenchmarkSignalCurrentProcessZero` | 206.0 | 0 | 206.7 | 0 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/procutil/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/procutil/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/procutil/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/procutil/`.

### Security — Threat Model

- Trust boundaries:
  - `internal/procutil` is a thin internal syscall wrapper. It is reached only through in-repo callers in CLI, daemon, and memory coordination code.
  - The package boundary is crossed by integer PIDs derived from local daemon metadata, local lock files, or the host process table.
- Attacker capabilities:
  - A same-user local actor may influence daemon-home files such as `daemon.json`, daemon lock files, or consolidation lock files if they already control that filesystem location.
  - A same-user local actor may also create or terminate processes that affect PID reuse and the host process table.
- In-scope assets:
  - Correct rejection of invalid/non-positive PIDs.
  - Accurate liveness checks used to reclaim stale daemon and memory locks.
  - Limiting signal delivery to caller-supplied positive PIDs without introducing broader process-group signaling behavior.
- Out-of-scope:
  - A hostile operator or local user who already controls the AGH home directory or the same account's process table.
  - Authorization decisions in higher-level CLI or daemon flows before they choose which PID to inspect or signal.
  - OS-specific kernel semantics behind `syscall.Kill`, `OpenProcess`, or `TerminateProcess`.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/memory/lock.go:171` | Consolidation lock file contents read from disk into `lockState.rawPID` / `lockState.pid`. | `strings.TrimSpace`, `strconv.Atoi`, and `pid > 0` gate at `internal/memory/lock.go:191-203`. | `l.processAlive(state.pid)` at `internal/memory/lock.go:214`, which resolves to `procutil.Alive` (`internal/procutil/procutil.go:13` on this platform). | LOW — rejected; same-user local file input only influences a boolean liveness probe and cannot expand privilege beyond what the filesystem owner already controls. |
| `internal/daemon/lock.go:75` | Daemon lock file contents read via `readLockPID` into `priorPID`. | `strings.TrimSpace`, `strconv.Atoi`, and `pid > 0` validation in `internal/daemon/lock.go:138-157`. | `deps.processAlive(priorPID)` at `internal/daemon/lock.go:90`, which defaults to `procutil.Alive`. | LOW — rejected; local daemon-home metadata, validated to positive integers, used only to detect stale lock owners. |
| `internal/cli/daemon.go:299` | `daemon.json` PID loaded from disk through `aghdaemon.ReadInfo`. | `Info.Validate` enforces `PID > 0` at `internal/daemon/info.go:27-40` after JSON decode in `internal/daemon/info.go:71-75`. | `deps.processAlive(info.PID)` at `internal/cli/daemon.go:308` and `deps.signalProcess(info.PID, syscall.SIGTERM)` at `internal/cli/daemon.go:133`, which default to `procutil.Alive` / `procutil.Signal`. | LOW — rejected; same-user local metadata can at most target processes the local account could already inspect or signal. No privilege boundary is crossed inside this package. |
| `internal/daemon/orphan.go:30` | Host process table from `ps -axo pid=,ppid=` plus `stalePID` recovered from the daemon lock path. | `strconv.Atoi` with invalid-row rejection in `internal/daemon/orphan.go:94-111`, plus `proc.PID > 0` / `proc.PPID == stalePID` filtering at `internal/daemon/orphan.go:36-38`. | `d.signalProcess(proc.PID, syscall.SIGTERM)` and `d.signalProcess(proc.PID, syscall.SIGKILL)` at `internal/daemon/orphan.go:40-48`, which default to `procutil.Signal`. | LOW — rejected; this is host-local process cleanup scoped to children of an already-validated stale daemon PID, not attacker-controlled remote input. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `PROCUTIL-OPT-001` | extreme-software-optimization | low | `internal/procutil/procutil.go:13` | Baseline benchmarks show the exported syscall wrappers are already zero-allocation shims, so there is no measurable package-local optimization lever worth landing. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- The package remains intentionally small: two production files, one test file, and one benchmark file.
- No file-size issue exists, and the duplication scan returned no block at the configured threshold.
- The only elevated complexity is the Windows-specific signal handling switch, but its size stays well within repository norms and no cross-file smell justified refactoring.

### extreme-software-optimization

- Added `internal/procutil/procutil_bench_test.go` before triage so the performance pass started from measured baselines rather than assumptions.
- Both public wrappers benchmark at roughly `206 ns/op` with `0 B/op`, which points to kernel/syscall cost rather than package-local allocations or data-structure overhead.
- The rerun moved by less than 1 ns/op for both benchmarks, so the variation stayed within measurement noise and confirmed the `wontfix` optimization decision.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package contains no goroutines, channels, mutexes, or `select` statements, so there is no package-local deadlock or goroutine-leak path to fix.

### security-review

- All reachable inputs are local PIDs sourced from daemon metadata, lock files, or the OS process table.
- No HIGH-confidence or MEDIUM-confidence vulnerability survived the threat-model review because this package does not parse network input, spawn shells, build paths, or cross a privilege boundary on its own.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/procutil (1.052s)
DONE 4488 tests in 8.993s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0`.
