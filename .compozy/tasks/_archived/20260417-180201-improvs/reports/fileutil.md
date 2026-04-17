# Improvements Report — internal/fileutil

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/fileutil/atomic_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/fileutil | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 7 | `TestAtomicWriteFileDoesNotCorruptTargetOnFailure` | `internal/fileutil/atomic_test.go:41` |
| 7 | `AtomicWriteFile` | `internal/fileutil/atomic.go:14` |
| 6 | `TestAtomicWriteFileWritesContentAndPermissions` | `internal/fileutil/atomic_test.go:13` |
| 6 | `TestAtomicWriteFilePreservesLiteralWhitespaceInPath` | `internal/fileutil/atomic_test.go:84` |
| 5 | `writeTempFile` | `internal/fileutil/atomic.go:48` |
| 4 | `benchmarkAtomicWriteFile` | `internal/fileutil/atomic_bench_test.go:17` |
| 4 | `TestWriteTempFileReturnsErrorForClosedFile` | `internal/fileutil/atomic_test.go:134` |
| 3 | `syncDir` | `internal/fileutil/atomic_dirsync_unix.go:7` |
| 3 | `TestSyncDirRejectsMissingDirectory` | `internal/fileutil/atomic_test.go:151` |
| 3 | `TestAtomicWriteFileFailsWhenTargetIsDirectory` | `internal/fileutil/atomic_test.go:120` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/fileutil/` exceeds 300 LOC. Largest non-test file is `internal/fileutil/atomic.go` at 65 LOC.

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 20 internal/fileutil` showed only short assertion repeats in tests; no duplicated block reached the 8-line reporting threshold.

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/fileutil/atomic_test.go:28-30` | `internal/fileutil/atomic_test.go:71-73` | 3-line `bytes.Equal` assertion repeat; below the 8-line threshold. |
| `internal/fileutil/atomic_test.go:55-57` | `internal/fileutil/atomic_test.go:125-127` | 3-line setup/assertion repeat; below the 8-line threshold. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `AtomicWriteFile` | `internal/fileutil/atomic.go:14` | Public durability helper used on memory-file writes, memory index rewrites, and session-meta persistence (`internal/memory/store.go:128`, `internal/memory/store.go:326`, `internal/store/meta.go:56`). | `BenchmarkAtomicWriteFile1KiB`, `BenchmarkAtomicWriteFile64KiB` |
| `writeTempFile` | `internal/fileutil/atomic.go:48` | Write/chmod/fsync/close helper on every `AtomicWriteFile` call; exercised transitively by the end-to-end atomic-write benchmarks. | `BenchmarkAtomicWriteFile1KiB`, `BenchmarkAtomicWriteFile64KiB` |
| `syncDir` | `internal/fileutil/atomic_dirsync_unix.go:7` | Unix parent-directory fsync runs on every successful atomic write and is part of the durability cost envelope. | `BenchmarkAtomicWriteFile1KiB`, `BenchmarkAtomicWriteFile64KiB` |

### Optimization — Benchmark Results

Baseline `before` command: `go test ./internal/fileutil/... -run '^$' -bench=. -benchmem -count=5` against a temporary copy of `HEAD`, because the newly added regression test intentionally failed until the path-handling bug was fixed.

Final `after` command: `go test ./internal/fileutil/... -bench=. -benchmem -count=5`

Values below use the median of 5 runs from `/tmp/fileutil-bench-before.txt` and `/tmp/fileutil-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkAtomicWriteFile1KiB` | 10365693 | 1263 | 11807274 | 1264 | not-hot-confirmed-by-benchmark |
| `BenchmarkAtomicWriteFile64KiB` | 10561088 | 1264 | 12114060 | 1263 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/fileutil/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/fileutil/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/fileutil/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/fileutil/`.

### Security — Threat Model

- Trust boundaries:
  - Internal packages call `fileutil.AtomicWriteFile` with a target path, file content, and target permissions.
  - `internal/fileutil` itself only performs local filesystem operations (`CreateTemp`, `Write`, `Chmod`, `Rename`, directory sync) and does not expose a network or RPC surface.
- Attacker capabilities:
  - A local caller that already reaches this helper can influence the `path`, `content`, and `perm` arguments.
  - The package assumes the host filesystem semantics and permissions are enforced by the OS.
- In-scope assets:
  - Integrity of the caller-selected target path.
  - Durability of written content and permissions on supported platforms.
  - Cleanup of the temporary file on write/rename failures.
- Out-of-scope:
  - A malicious OS/filesystem implementation or privileged actor with direct write access to the same directory.
  - Cross-process coordination outside the caller's ownership.
  - Platform-specific filesystem semantics not exercised on this Darwin run.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/fileutil/atomic.go:14` | `path` from internal persistence callers (`internal/memory/store.go:128`, `internal/memory/store.go:326`, `internal/store/meta.go:56`, `sdk/examples/telegram-reference/main.go:872`) | Rejects blank / whitespace-only paths; otherwise preserves the literal caller-selected path. | `filepath.Dir`, `filepath.Base`, `os.CreateTemp`, `os.Rename`, `syncDir`. | LOW — rejected; local file-write surface only, with no path traversal amplification or external execution in this package. |
| `internal/fileutil/atomic.go:14` | `content` from the same callers | None inside `fileutil`; caller is responsible for serialization/validation. | `writeTempFile` -> `(*os.File).Write`. | LOW — rejected; the package writes bytes verbatim and never parses or executes them. |
| `internal/fileutil/atomic.go:14` | `perm` from the same callers | Type-constrained `os.FileMode`. | `writeTempFile` -> `(*os.File).Chmod`. | LOW — rejected; this only affects local file mode bits on the already selected path. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `FILEUTIL-REF-001` | refactoring-analysis | medium | `internal/fileutil/atomic.go:14` | `AtomicWriteFile` trimmed caller paths, which rewrote legitimate whitespace-suffixed filenames before temp-file creation and rename. | fixed |

## Per-Skill Notes

### refactoring-analysis

- The package is structurally small: no file-size issue, no duplication block >= 8 lines, and the only public surface is `AtomicWriteFile`.
- The public-surface review found one correctness-affecting smell: `AtomicWriteFile` used `strings.TrimSpace(path)` as the actual sink path, which silently changed legitimate filenames instead of only rejecting blank input.
- Added `TestAtomicWriteFilePreservesLiteralWhitespaceInPath` to lock the corrected behavior in place. Package coverage remains above 90% (`92.1%` via `go test ./internal/fileutil -cover`).

### extreme-software-optimization

- Added `internal/fileutil/atomic_bench_test.go` before assessing performance so the durability path had concrete numbers.
- Both benchmarks are dominated by tempfile creation, file sync, rename, and directory sync work. The post-fix medians moved around the same syscall-heavy band, which is not a signal for a package-local performance change.
- No production optimization was justified; both candidates are recorded as `not-hot-confirmed-by-benchmark`.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package contains no goroutines, channels, mutexes, or `select` statements, so there is no concrete deadlock or goroutine-leak path inside `internal/fileutil/`.

### security-review

- The package is an internal local-file writer, not a parsing or execution boundary.
- No HIGH-confidence or MEDIUM-confidence source-to-sink security issue survived the threat-model review.
- The fixed path-preservation change improves integrity of the caller-selected sink, but it did not amount to a reportable exploit path under the declared threat model.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/fileutil (cached)
DONE 4469 tests in 1.141s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final rerun after the `internal/fileutil` changes and tracking updates.
