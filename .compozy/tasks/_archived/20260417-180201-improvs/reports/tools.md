# Improvements Report — internal/tools

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| `refactoring-analysis` | `run` | cyclomatic top-10 + file-size + duplication sections below |
| `extreme-software-optimization` | `run` | benchmarks in `internal/tools/perf_bench_test.go`, results below |
| `ubs` | `not-run` | no callable Skill tool is available in this environment, and the task forbids CLI substitution |
| `deadlock-finder-and-fixer` | `run` | goroutine/channel/mutex/select inventories below |
| `security-review` | `run` | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

| Complexity | Function | Location |
| --- | --- | --- |
| 9 | `validateToolSpec` | `internal/tools/resource.go:23` |
| 8 | `TestToolSourceOrderingAndJSON` | `internal/tools/tool_test.go:131` |
| 7 | `TestToolMarshalJSONCanonical` | `internal/tools/tool_test.go:40` |
| 7 | `(*Tool).UnmarshalJSON` | `internal/tools/tool.go:97` |
| 6 | `assertToolEqual` | `internal/tools/tool_test.go:20` |
| 5 | `TestToolSourceInvalid` | `internal/tools/tool_test.go:158` |
| 5 | `TestToolResourceCodecCanonicalizesInputSchema` | `internal/tools/resource_test.go:20` |
| 5 | `(*ToolSource).UnmarshalText` | `internal/tools/tool.go:61` |
| 4 | `BenchmarkToolUnmarshalJSON` | `internal/tools/perf_bench_test.go:74` |
| 4 | `BenchmarkToolSourceMarshalText` | `internal/tools/perf_bench_test.go:47` |
| 3 | `BenchmarkValidateToolSpec` | `internal/tools/perf_bench_test.go:98` |

### Refactoring — Files > 300 LOC

None. The largest file in `internal/tools/` is `internal/tools/tool_test.go` at 169 LOC; all non-test files are under 120 LOC.

### Refactoring — Duplication

- Remaining meaningful clone group after cleanup:
  `internal/tools/tool_test.go:83-93` ↔ `internal/tools/tool_test.go:94-104`
  This is the pair of explicit expected `Tool` literals in the table-driven JSON decode test.
- Remaining `dupl` output is benchmark/test harness boilerplate (single-line `b.Fatalf`/`t.Fatalf` patterns and shared benchmark scaffolding), not meaningful production duplication.

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark Name |
| --- | --- | --- | --- |
| `ToolSource.String` | `internal/tools/tool.go:45` | Serialization helper used by JSON/text marshaling and API schema generation. | `BenchmarkToolSourceString` |
| `ToolSource.MarshalText` | `internal/tools/tool.go:53` | Text-encoding helper on the same serialization path as `String`. | `BenchmarkToolSourceMarshalText` |
| `(*ToolSource).UnmarshalText` | `internal/tools/tool.go:61` | Enum decode path for every JSON `source` field; current implementation trims and scans dynamically. | `BenchmarkToolSourceUnmarshalText` |
| `(*Tool).UnmarshalJSON` | `internal/tools/tool.go:97` | Allocation-heavy tool decode path used by resource and API payload parsing. | `BenchmarkToolUnmarshalJSON/canonical`, `BenchmarkToolUnmarshalJSON/hook_alias` |
| `validateToolSpec` | `internal/tools/resource.go:23` | Canonical validation path for persisted tool resources; decodes and re-marshals input schema. | `BenchmarkValidateToolSpec` |

### Optimization — Benchmark Results

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | --- | --- | --- | --- | --- |
| `BenchmarkToolSourceString` | 1.813 | 0 | 0.2585 | 0 | fixed-with-benchmark |
| `BenchmarkToolSourceMarshalText` | 16.32 | 16 | 13.13 | 16 | fixed-with-benchmark |
| `BenchmarkToolSourceUnmarshalText` | 40.26 | 16 | 3.899 | 0 | fixed-with-benchmark |
| `BenchmarkToolUnmarshalJSON/canonical` | 2016 | 744 | 1989 | 736 | fixed-with-benchmark |
| `BenchmarkToolUnmarshalJSON/hook_alias` | 2053 | 736 | 2025 | 736 | fixed-with-benchmark |
| `BenchmarkValidateToolSpec` | 1396 | 1761 | 1349 | 1761 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — no callable Skill tool is available in this environment, and the task forbids CLI substitution.

### Concurrency — Goroutine Inventory

No goroutine entry points in `internal/tools/`.

### Concurrency — Channel Inventory

No channels declared in `internal/tools/`.

### Concurrency — Mutex Inventory

No `sync.Mutex` or `sync.RWMutex` fields in `internal/tools/`.

### Concurrency — Select Audit

No `select` statements in `internal/tools/`.

### Security — Threat Model

- Trust boundaries:
  `internal/tools/` sits behind internal callers such as daemon resource sync, extension resource publication, API/resource store encode-decode paths, and generated schema metadata. The package itself does not open files, spawn processes, or accept network connections directly.
- Attacker capabilities:
  An attacker may influence JSON payloads that callers decode into `Tool` values or resource specs, including `name`, `tool_name`, `description`, `read_only`, `source`, and `input_schema`.
- In-scope assets:
  Integrity of canonical tool metadata, correct source attribution for tool records, safe schema normalization, and bounded decode/validation behavior inside the package.
- Out-of-scope:
  Caller authentication/authorization, upstream transport validation, extension manifest path resolution, and any downstream execution behavior of the tools described by these structs.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/tools/tool.go:61` | JSON/text `source` field | trims whitespace and checks enum membership during text decode | populates `ToolSource` on decoded structs | rejected: closed enum with metadata-only sink |
| `internal/tools/tool.go:97` | JSON tool payload (`name`, `tool_name`, `description`, `read_only`, `input_schema`, `source`) | JSON decoding plus conflicting-name rejection | populates `Tool` values used by resource/API callers | rejected: metadata-only sink, and in-repo callers already send explicit `source` values |
| `internal/tools/resource.go:23` | Resource spec fields after caller decode | trims strings, validates scope/source, enforces non-empty name | canonical `Tool` resource spec persisted/published by callers | rejected: normalized metadata only, no direct file/network/process sink |
| `internal/tools/resource.go:42` | `tool.input_schema` JSON fragment | JSON decode, object-shape enforcement, canonical re-marshal | normalized `InputSchema` bytes stored on `Tool` specs | rejected: object-only validation with package-level 256 KiB cap and no execution sink |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | `refactoring-analysis` | `low` | `internal/tools/tool_test.go:20` | consolidated repeated per-field `Tool` assertions and resource codec setup into focused test helpers | `fixed` |
| 02 | `extreme-software-optimization` | `low` | `internal/tools/tool.go:61` | `ToolSource.UnmarshalText` allocated on every decode because it trimmed via `string(text)` and scanned lookup data dynamically | `fixed` |
| 03 | `refactoring-analysis` | `low` | `internal/tools/tool_test.go:83` | duplicate expected `Tool` literals remain in the table-driven decode cases | `wontfix` |

## Per-Skill Notes

### refactoring-analysis

- No file in `internal/tools/` exceeds 300 LOC.
- The package's highest production cyclomatic score remains `validateToolSpec` at 9, which is reasonable for the current schema-normalization branches.
- Fixed the duplicated per-field assertion block in `tool_test.go` and repeated codec setup in `resource_test.go`.
- Kept the duplicate expected `Tool` literals at `internal/tools/tool_test.go:83-104` because collapsing them further would trade away table readability for minimal gain.

### extreme-software-optimization

- Benchmarked every plausible hot path in the package via `internal/tools/perf_bench_test.go`.
- Replaced the dynamic `ToolSource` lookup path with dense string/byte tables plus `bytes.TrimSpace`, eliminating the decode-path allocation and reducing `Tool.UnmarshalJSON` by one allocation through the nested enum decode.
- `validateToolSpec` improved slightly as a byproduct of the cheaper source validation path, but the benchmark still shows it is not the hot spot worth further optimization in this package.

### ubs

- `not-run` because this environment does not expose a callable Skill tool for UBS and the task explicitly forbids CLI substitution.

### deadlock-finder-and-fixer

- No goroutines, channels, mutexes, or `select` statements exist in `internal/tools/`, so the concurrency audit is structurally empty and there are no hang/leak paths inside this package.

### security-review

- No high-confidence vulnerabilities identified.
- Every attacker-input surface terminates in metadata normalization or enum validation; the package has no file, process, network, SQL, or template sink.

## Deferred Items (carry forward)

- UBS pass remains blocked until the environment exposes the required Skill tool runner.

## `make verify`

Final clean pass from `make verify`:

```text
(node:42843) Warning: The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.
# github.com/golangci/golangci-lint/v2/cmd/golangci-lint
ld: warning: -bind_at_load is deprecated on macOS
Found 0 warnings and 0 errors.
0 issues.
✓  internal/tools (cached)
✓  internal/cli (cached)
DONE 4512 tests in 1.245s
OK: all package boundaries respected
```
