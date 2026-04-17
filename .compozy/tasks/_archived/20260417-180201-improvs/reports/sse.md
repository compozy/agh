# Improvements Report — internal/sse

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/sse/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/sse`:

| Complexity | Function | File |
| --- | --- | --- |
| 18 | `Decode` | `internal/sse/decode.go:33` |
| 8 | `decodeLine` | `internal/sse/decode.go:98` |
| 4 | `benchmarkDecode` | `internal/sse/perf_bench_test.go:40` |
| 4 | `TestDecodeStopsOnErrStop` | `internal/sse/decode_test.go:56` |
| 4 | `TestDecodeRejectsNilArguments` | `internal/sse/decode_test.go:11` |
| 4 | `appendDataLine` | `internal/sse/decode.go:126` |
| 3 | `TestDecodeRejectsOversizedPendingEvent` | `internal/sse/decode_test.go:129` |
| 3 | `TestDecodePreservesMultiLineData` | `internal/sse/decode_test.go:105` |
| 3 | `readerIsNil` | `internal/sse/decode.go:141` |
| 2 | `buildBenchmarkStream` | `internal/sse/perf_bench_test.go:60` |

### Refactoring — Files > 300 LOC

No non-test Go file in `internal/sse/` exceeds 300 LOC. Largest non-test file is `internal/sse/decode.go` at 153 LOC.

### Refactoring — Duplication

Output from `dupl -plumbing -t 20 internal/sse` returned no duplicated block at the configured threshold. No duplication reached an 8-line threshold inside `internal/sse/`.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `Decode` single-line event path | `internal/sse/decode.go:33` | The exported decoder is an IO-driven scan loop. Even without a direct in-repo `Decode` caller yet, this is the package's primary steady-state path and must be measured instead of assumed. | `BenchmarkDecodeSingleLineEvents` |
| `Decode` multi-line data aggregation path | `internal/sse/decode.go:48` | The package buffers every `data:` line before invoking the handler. This is the only clearly allocation-heavy path in the decoder and the most plausible package-local optimization target. | `BenchmarkDecodeMultiLineDataEvents` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/sse/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/sse/...`

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkDecodeSingleLineEvents` | 6047 | 67595 | 6404 | 67787 | not-hot-confirmed-by-benchmark |
| `BenchmarkDecodeMultiLineDataEvents` | 8565 | 69899 | 8309 | 69067 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

No `go` statements exist in `internal/sse/`.

### Concurrency — Channel Inventory

No channels are declared in `internal/sse/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` values exist in `internal/sse/`.

### Concurrency — Select Audit

No `select` statements exist in `internal/sse/`.

### Security — Threat Model

- Trust boundaries:
  - `internal/sse` accepts an arbitrary `io.Reader` and converts upstream SSE frames into `Event` values for a caller-provided handler.
  - The package itself does not own transport authentication or origin validation; it trusts the caller to decide which upstream stream is safe to read.
  - `internal/cli/client.go:396-405` currently imports `sse.Event` and `sse.ErrStop`, but retains a duplicated local decoder. That duplication is tracked separately as a deferred refactor, not a package-local security boundary.
- Attacker capabilities:
  - An upstream SSE producer can fully control line content, event framing cadence, and the number of `data:` lines before a blank-line delimiter.
  - A same-process caller can supply any `io.Reader` implementation, including malformed streams and typed-nil readers.
- In-scope assets:
  - Correct stop semantics when handlers return `ErrStop`.
  - Bounded memory usage while buffering one pending event.
  - Accurate preservation of parsed `id`, `event`, and `data` fields passed to handlers.
- Out-of-scope:
  - Authenticating or authorizing the upstream stream source.
  - Validating that `event.Data` contains semantically valid JSON for a particular consumer.
  - Handler-side processing after `Decode` hands off an `Event`.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/sse/decode.go:44` | Arbitrary SSE bytes read from caller-supplied `io.Reader`. | `bufio.Scanner` caps each line at `1 MiB` via `scanner.Buffer(..., maxLineBytes)`. | `scanner.Text()` forwarded to `decodeLine` at `internal/sse/decode.go:70`. | LOW — line-level bounds exist and malformed oversized lines are rejected before parsing. |
| `internal/sse/decode.go:113` | `id` field content from an upstream SSE frame. | A single leading space after `:` is stripped. | `event.ID` forwarded to the handler at `internal/sse/decode.go:56`. | LOW — metadata passthrough only; no path, shell, SQL, or HTML sink in this package. |
| `internal/sse/decode.go:115` | `event` field content from an upstream SSE frame. | A single leading space after `:` is stripped. | `event.Event` forwarded to the handler at `internal/sse/decode.go:56`. | LOW — metadata passthrough only; downstream meaning is owned by the caller. |
| `internal/sse/decode.go:117` | Repeated `data:` lines from an upstream SSE frame. | Per-line scanner cap plus aggregate `maxEventBytes` guard in `appendDataLine` at `internal/sse/decode.go:126-138`. | `event.Data` copy at `internal/sse/decode.go:53-54`, then `handler(event)` at `internal/sse/decode.go:56`. | MEDIUM — fixed in-package; previously unbounded per-event buffering could grow memory until a delimiter arrived. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `SSE-COR-001` | deadlock-finder-and-fixer | medium | `internal/sse/decode.go:59` | `ErrStop` now terminates the scan loop instead of being normalized to `nil` while later events continue to drain from the reader. | fixed |
| `SSE-OPT-001` | extreme-software-optimization | low | `internal/sse/decode.go:48` | Replacing the multi-line `strings.Join` path with a bounded reusable buffer reduced the multi-line decode benchmark from `69899 B/op` to `69067 B/op` and from `227 allocs/op` to `195 allocs/op`. | fixed |
| `SSE-SEC-001` | security-review | medium | `internal/sse/decode.go:126` | Pending `data:` lines are now capped per event, preventing an upstream stream from growing decoder memory indefinitely before a frame delimiter arrives. | fixed |
| `SSE-REF-001` | refactoring-analysis | medium | `internal/sse/decode.go:33` | The package decoder behavior is still duplicated out-of-package in `internal/cli/client.go:1489`, which keeps shared fixes from automatically benefiting the CLI path. | deferred |

## Per-Skill Notes

### refactoring-analysis

- No file-size issue exists, and `dupl` found no duplicated block of 8+ lines inside `internal/sse/`.
- Package coverage is now `88.6%`, which is above the repository floor for this pass.
- Cross-package duplication with `internal/cli/client.go:1489-1582` is recorded as deferred because this task cannot edit outside `internal/sse/`.

### extreme-software-optimization

- Added `internal/sse/perf_bench_test.go` before changing decoder behavior so the optimization pass started from measured baselines.
- The single-line path did not expose a useful optimization lever after the correctness/security fixes; its median moved within the same range while retaining `131 allocs/op`, so it is recorded as `not-hot-confirmed-by-benchmark`.
- The multi-line path improved materially once `Decode` buffered `data:` lines directly and copied once on emit, reducing the benchmark median from `8565 ns/op` / `69899 B/op` to `8309 ns/op` / `69067 B/op`.

### ubs

- `not-run` due missing skill-runner support in this session; no manual substitute will be used.

### deadlock-finder-and-fixer

- No goroutines, channels, mutexes, or `select` statements exist in this package.
- The concrete behavior bug here was synchronous control-flow rather than a traditional goroutine deadlock: `ErrStop` previously failed to stop the scan loop, which could keep draining later events from a live stream after the caller requested a stop.
- `TestDecodeStopsOnErrStop` now locks the fixed contract in place.

### security-review

- Threat model and source-to-sink inventory were completed before the package fix, and the only HIGH/MEDIUM-confidence issue that survived was aggregate event buffering.
- The package now enforces the same `1 MiB` limit at the per-event data buffer level that it already enforced at the per-line scanner level.
- No remaining HIGH-confidence or MEDIUM-confidence issue survived the post-fix review.

## Deferred Items (carry forward)

- **`SSE-REF-001`** — Deduplicate the CLI-local decoder at `internal/cli/client.go:1489-1582` so `internal/sse.Decode` becomes the single implementation. This requires an out-of-scope caller edit and should be handled by the future `internal/cli` or follow-up integration task.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
✓  internal/sse (cached)
DONE 4499 tests in 0.779s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:

- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while building the vendored `golangci-lint` binary.

`make verify` exited with code `0`.
