# Improvements Report — internal/codegen

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 2 benchmarks in `internal/codegen/sdkts/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/codegen | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 14 | `(*generator).tsTypeByKind` | `internal/codegen/sdkts/generate.go:329` |
| 13 | `(*generator).ensureNamed` | `internal/codegen/sdkts/generate.go:144` |
| 11 | `(*generator).structFields` | `internal/codegen/sdkts/generate.go:205` |
| 7 | `jsonFieldName` | `internal/codegen/sdkts/generate.go:244` |
| 7 | `(*generator).resolveNamedTSType` | `internal/codegen/sdkts/generate.go:308` |
| 6 | `shouldAutoEmitNamedType` | `internal/codegen/sdkts/generate.go:616` |
| 6 | `(*generator).primitiveAliasForType` | `internal/codegen/sdkts/generate.go:379` |
| 5 | `tsPrimitiveType` | `internal/codegen/sdkts/generate.go:293` |
| 5 | `namedBaseType` | `internal/codegen/sdkts/generate.go:116` |
| 5 | `isPrimitiveAliasType` | `internal/codegen/sdkts/generate.go:407` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/codegen/sdkts/generate.go` | 638 | Generator graph discovery, recursive reflection/type rendering, enum registries, and TypeScript module assembly all live in one unit. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/codegen` findings:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/codegen/sdkts/generate.go:523-536` | `internal/codegen/sdkts/generate.go:597-610` | Static string-slice builders for enum/value registries repeat the same literal-return shape. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `Generate` | `internal/codegen/sdkts/generate.go:34` | Sole production entry point, walks every SDK root type and assembles the full TypeScript contracts module in one allocation-heavy pass. | `BenchmarkGenerate` |
| `(*generator).structFields` | `internal/codegen/sdkts/generate.go:205` | Recursive reflective field walker used across named and inline struct emission, including embedded-field flattening and tag parsing. | `BenchmarkStructFieldsPromptPayload` |

### Optimization — Benchmark Results

Baseline averages from `go test -bench=. -benchmem -count=5 ./internal/codegen/...` before any production change:

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkGenerate` | 512324.60 | 861509.00 | 398688.40 | 479938.00 | fixed-with-benchmark |
| `BenchmarkStructFieldsPromptPayload` | 37731.00 | 91264.40 | 34283.40 | 75361.00 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `go` statements exist under `internal/codegen/`. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| none | — | — | — | — | No production channels are declared under `internal/codegen/`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `sync.Mutex` or `sync.RWMutex` fields are declared under `internal/codegen/`. |

### Concurrency — Select Audit

No production `select` statements exist under `internal/codegen/`.

### Security — Threat Model

- Trust boundaries:
  - `cmd/agh-codegen` calls `sdkts.Generate()` in-process to render the SDK contracts module.
  - The package reflects over compiled-in internal contract registries and Go type metadata, then returns generated TypeScript source as a string.
- Attacker capabilities:
  - No direct runtime attacker-controlled input reaches this package; its inputs come from trusted, compiled repository types and registries.
  - A malicious repository contributor could alter those internal registries or struct tags before build time, but that is a trusted-source-code problem rather than an external package boundary.
- In-scope assets:
  - Deterministic and correct TypeScript contract generation.
  - Preservation of the expected exported type surfaces consumed by `sdk/typescript`.
- Out-of-scope:
  - Filesystem writes and formatter execution handled by `cmd/agh-codegen`.
  - Malicious source-code changes already compiled into the binary.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/codegen/sdkts/generate.go:71-97` | Compiled-in `HostAPIMethodSpecs()`, `HookContracts()`, and `SDKRootTypes()` registries. | None inside this package; sources are trusted internal code, not runtime input. | `generator.rootTypes`, `generator.hostSpecs`, and `generator.hookSpecs` drive the entire type graph emission. | REJECTED — build-time internal registries are not attacker-controlled at this package boundary. |
| `internal/codegen/sdkts/generate.go:244-267` | Compiled-in Go struct field tags discovered through reflection. | `jsonFieldName` normalizes the tag and skips `json:"-"` fields. | Field names and optionality in generated TypeScript interfaces/object literals. | REJECTED — field tags come from trusted repository types, not external runtime input. |
| `internal/codegen/sdkts/generate.go:616-623` | Reflected package paths and type metadata from compiled internal types. | `shouldAutoEmitNamedType` gates auto-emission to internal named structs/enums/aliases only. | Type queue expansion during recursive emission. | REJECTED — reflected type metadata is derived from compiled code, not attacker input. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/codegen/sdkts/generate.go:34`, `71`, `205`, `244` | Generator output assembly and reflective field discovery paid avoidable allocation churn from final-string trimming, repeated slice growth, and `json` tag splitting. | fixed |
| 02 | refactoring-analysis | medium | `internal/codegen/sdkts/generate.go:1` | `generate.go` remains a 638-LOC multi-responsibility unit spanning graph discovery, reflection helpers, enum registries, and final module rendering. | deferred |
| 03 | refactoring-analysis | low | `internal/codegen/sdkts/generate.go:523` | Static enum/value registry builders still repeat the same literal-return pattern. | wontfix |

## Per-Skill Notes

### refactoring-analysis
- `generate.go` is still the main maintainability pressure point because it combines discovery, rendering, and registry helpers in one file.
- The duplication scan only found static literal-return helpers; I left that as `wontfix` because extracting a generic helper would add abstraction without materially improving correctness or change cost.
- I kept the structural split as deferred work because this pass could land a measurable generator optimization and raise package coverage to `90.0%` without widening package churn.

### extreme-software-optimization
- Added `internal/codegen/sdkts/perf_bench_test.go` so the package now has co-located benchmarks for the only production entry point and its recursive field walker.
- Baseline profiling on `BenchmarkGenerate` showed time in `structFields`, `jsonFieldName`, and final output assembly, alongside heavy allocation/GC pressure.
- Fixed the hot path by pre-sizing root/field/output buffers, avoiding the final `TrimRight` copy in `Generate`, reusing `hooks.AllHookEvents()` once in `hookEventValues`, and replacing `strings.Split` in `jsonFieldName` with a lighter `strings.Cut` loop.
- `BenchmarkGenerate` improved from `512324.60 ns/op, 861509.00 B/op, 4381 allocs/op` to `398688.40 ns/op, 479938.00 B/op, 2290 allocs/op`.
- `BenchmarkStructFieldsPromptPayload` improved from `37731.00 ns/op, 91264.40 B/op, 162 allocs/op` to `34283.40 ns/op, 75361.00 B/op, 133 allocs/op`.
- `go run ./cmd/agh-codegen check` passed after the optimization, which confirmed the generated contracts output remained fresh.

### ubs
`not-run` due missing skill-runner interface in this session; no manual substitute was performed.

### deadlock-finder-and-fixer
- No production deadlock or leak finding was confirmed because the package has no goroutines, channels, mutexes, or blocking `select` statements.
- The concurrency inventories are intentionally empty for this package and still recorded per the task gate.

### security-review
- No high-confidence vulnerabilities identified.
- Every inspected surface is a compiled-in internal registry or reflected type/tag source, so the package does not expose a runtime attacker-controlled boundary of its own.

## Deferred Items (carry forward)

- **02** — Split `internal/codegen/sdkts/generate.go` along discovery/rendering/registry seams only when a future task is ready to absorb a broader structural refactor.

## `make verify`

Command: `make verify`

First pass failed after the new test/optimization code introduced lint/vet issues:

```text
internal/codegen/sdkts/generate.go:258:6: emptyStringTest: replace `len(opts) > 0` with `opts != ""` (gocritic)
internal/codegen/sdkts/perf_bench_test.go:17:6: emptyStringTest: replace `len(out) == 0` with `out == ""` (gocritic)
internal/codegen/sdkts/generate_test.go:29:2: structtag: suspicious space in struct tag value (govet)
internal/codegen/sdkts/generate_test.go:21:2: field skipped is unused (unused)
```

After fixing those root causes, the clean pass succeeded with exit code `0`:

```text
0 issues.
✓  cmd/agh-codegen (cached)
✓  internal/codegen/sdkts (cached)
✓  internal/hooks (cached)
✓  internal/extension (cached)
✓  internal/daemon (cached)
DONE 4458 tests in 0.851s
OK: all package boundaries respected
```
