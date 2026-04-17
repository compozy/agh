# Improvements Report — internal/frontmatter

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/frontmatter/frontmatter_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/frontmatter | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 8 | `Split` | `internal/frontmatter/frontmatter.go:28` |
| 5 | `TestSplitValidDocument` | `internal/frontmatter/frontmatter_test.go:16` |
| 4 | `findClosingDelimiter` | `internal/frontmatter/frontmatter.go:97` |
| 4 | `TestDecodeValidDocument` | `internal/frontmatter/frontmatter_test.go:77` |
| 4 | `Decode` | `internal/frontmatter/frontmatter.go:61` |
| 3 | `nextLineBoundary` | `internal/frontmatter/frontmatter.go:85` |
| 3 | `benchmarkSplit` | `internal/frontmatter/frontmatter_bench_test.go:62` |
| 3 | `TestSplitErrors` | `internal/frontmatter/frontmatter_test.go:53` |
| 3 | `BenchmarkDecodeLF` | `internal/frontmatter/frontmatter_bench_test.go:47` |
| 2 | `normalizeLineEndings` | `internal/frontmatter/frontmatter.go:77` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/frontmatter/` exceeds 300 LOC. Largest non-test file is `internal/frontmatter/frontmatter.go` at 104 LOC.

### Refactoring — Duplication

Baseline `dupl -plumbing -t 20 internal/frontmatter` output showed only short repeated assertions in tests; no duplicated block reached the 8-line reporting threshold.

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/frontmatter/frontmatter_test.go:43-47` | `internal/frontmatter/frontmatter_test.go:94-98` | 5-line value/assertion repeat; below the 8-line threshold. |
| `internal/frontmatter/frontmatter_test.go:87-89` | `internal/frontmatter/frontmatter_test.go:110-112` | 3-line decode-callback repeat; below the 8-line threshold. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `Split` | `internal/frontmatter/frontmatter.go:28` | Public parsing entry point used by config, skills, memory, extension, and bundled-content loaders; every call normalizes line endings and scans for delimiters. | `BenchmarkSplitLF`, `BenchmarkSplitCRLF` |
| `Decode` | `internal/frontmatter/frontmatter.go:61` | Public helper used by config and memory parsing paths; transitively exercises `Split` on every decode call. | `BenchmarkDecodeLF` |
| `normalizeLineEndings` | `internal/frontmatter/frontmatter.go:77` | Allocation-heavy helper on the hot path of every public call; baseline showed extra allocs even on already-normalized input. | `BenchmarkSplitLF`, `BenchmarkSplitCRLF`, `BenchmarkDecodeLF` |

### Optimization — Benchmark Results

Baseline `before` command: `go test ./internal/frontmatter/... -run '^$' -bench=. -benchmem -count=5`

Final `after` command: `go test ./internal/frontmatter/... -run '^$' -bench=. -benchmem -count=5`

Values below use the median of 5 runs from `/tmp/frontmatter-bench-before.txt` and `/tmp/frontmatter-bench-after.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkSplitLF` | 158.4 | 944 | 130.1 | 592 | fixed-with-benchmark |
| `BenchmarkSplitCRLF` | 587.8 | 1328 | 549.2 | 592 | fixed-with-benchmark |
| `BenchmarkDecodeLF` | 159.7 | 944 | 130.6 | 592 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/frontmatter/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/frontmatter/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/frontmatter/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/frontmatter/`.

### Security — Threat Model

- Trust boundaries:
  - Internal callers pass markdown or metadata-bearing file content into `Split` and `Decode`; this package only normalizes line endings and slices metadata/body boundaries.
  - The package has no network, subprocess, filesystem-write, or reflection boundary of its own.
- Attacker capabilities:
  - A caller can provide attacker-influenced file content from workspace-controlled markdown, agent definitions, skills, memory files, or extension manifests.
  - Callers also choose the metadata decode callback for `Decode`; the callback itself is trusted internal code, not attacker data.
- In-scope assets:
  - Correct parsing boundaries between metadata and body.
  - Non-panicking behavior on malformed input.
  - Avoiding unintended execution, file access, or sink amplification inside this parser package.
- Out-of-scope:
  - Caller-specific schema validation of parsed metadata after `Decode` invokes the provided callback.
  - Authorization decisions in upstream packages.
  - Resource exhaustion caused by callers reading arbitrarily large files into memory before calling this package.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/frontmatter/frontmatter.go:28` | `content []byte` from internal file/content loaders (`internal/skills/loader.go:237`, `internal/skills/bundled/content.go:38`, `internal/extension/host_api.go:2121`) | Line endings normalized from CRLF to LF; malformed/missing delimiters rejected via `ErrMissing` / `ErrUnterminated`. | `normalizeLineEndings`, delimiter scans, returned metadata/body slices. | LOW — rejected; parser-only boundary with no execution, file access, or external call sink inside this package. |
| `internal/frontmatter/frontmatter.go:61` | `content []byte` from higher-level decode callers (`internal/config/agent.go:203`, `internal/memory/store.go:430`) | Same parsing validation as `Split`; `decode == nil` rejected before use. | `Split` followed by caller-supplied `decode(parts.Metadata)`. | LOW — rejected; this package passes bytes to trusted internal decode logic but does not construct commands, queries, or file paths. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `FRONTMATTER-OPT-001` | extreme-software-optimization | low | `internal/frontmatter/frontmatter.go:28` | `Split` always converted input bytes through string-based normalization and delimiter comparisons, adding avoidable allocations on both LF and CRLF inputs. | fixed |

## Per-Skill Notes

### refactoring-analysis

- The package is structurally small, with no file-size issue and no duplication block >= 8 lines.
- The main non-test complexity cluster is concentrated in `Split` and its helpers.
- Package coverage improved slightly from `92.3%` to `92.7%` after extending `TestSplitValidDocument` to cover both LF and CRLF documents directly.

### extreme-software-optimization

- Added `internal/frontmatter/frontmatter_bench_test.go` before changing production code so the package has concrete baseline numbers.
- Replaced string-based normalization and delimiter comparisons with byte-oriented equivalents while preserving the parser's returned metadata/body behavior.
- Benchmarked improvement was material:
  - `BenchmarkSplitLF`: `158.4 ns/op` -> `130.1 ns/op` and `944 B/op` -> `592 B/op`
  - `BenchmarkSplitCRLF`: `587.8 ns/op` -> `549.2 ns/op` and `1328 B/op` -> `592 B/op`
  - `BenchmarkDecodeLF`: `159.7 ns/op` -> `130.6 ns/op` and `944 B/op` -> `592 B/op`

### ubs

- `not-run` due to missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete; the package contains no goroutines, channels, mutexes, or `select` statements.

### security-review

- The package is a pure parser/splitter with no direct external sinks.
- No HIGH-confidence or MEDIUM-confidence source-to-sink vulnerability survived the threat-model review.

## Deferred Items (carry forward)

- None.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
0 issues.
✓  internal/frontmatter (cached)
DONE 4471 tests in 0.980s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0` on the final rerun after task tracking updates.
