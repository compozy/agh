# Refacs 018: `internal/codegen/sdkts`

## Scope

- Package: `github.com/pedronauck/agh/internal/codegen/sdkts`
- Iteration: 018
- Goal: deep refactoring and performance audit for the TypeScript SDK contract generator.
- Subagents:
  - Read-only refactoring audit.
  - Read-only performance audit.

## Baseline

Commands run before changes:

```bash
rtk go test ./internal/codegen/sdkts -count=1
rtk golangci-lint run ./internal/codegen/sdkts
rtk proxy go test ./internal/codegen/sdkts -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/sdkts -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/generate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/perf_bench_test.go
rtk make codegen-check
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench . -benchmem -count=5
```

Observed baseline:

- Package tests passed: `24` tests.
- Package lint passed with no issues.
- Coverage: `89.9% of statements`.
- Race tests passed.
- `make codegen-check` passed.
- `generate_test.go` failed the AGH test-shape helper because `TestStructFieldsFlattensEmbeddedAndRespectsTags` asserted directly without a `t.Run("Should ...")` subtest.
- Baseline benchmarks:
  - `BenchmarkGenerate`: about `831-864 us/op`, `~1.052 MB/op`, `4263-4264 allocs/op`.
  - `BenchmarkStructFieldsPromptPayload`: about `59-60 us/op`, `138770 B/op`, `201 allocs/op`.

## Findings

### P0: Generated `HookEventFamily` was stale

The SDK generator hard-coded only ten hook families:

- `session`
- `input`
- `prompt`
- `event`
- `agent`
- `turn`
- `message`
- `tool`
- `permission`
- `context`

The runtime hook taxonomy currently exposes sixteen families through `hooks.AllHookEvents()` and `HookEvent.Family()`, including:

- `sandbox`
- `automation`
- `coordinator`
- `task.run`
- `spawn`
- `network`

This was a correctness bug in the generated TypeScript SDK. Extension authors could receive a contract that rejected valid runtime hook families. `make codegen-check` did not catch it because the generator deterministically reproduced the stale list.

### P1: Tests covered generator shape, not semantic enum completeness

The existing generator test asserted determinism and broad block presence, but did not compare generated enum unions against runtime-owned values. This allowed the stale `HookEventFamily` union to pass package tests and codegen checks.

### P1: `structFields` was the largest package-local allocation source

Allocation profiling for `BenchmarkGenerate` showed `structFields` as the largest actionable package-local allocation source:

- Baseline allocation profile: `structFields` was about `45-60%` cumulative allocation space depending on profile run.
- `ensureNamed` accumulated most generator allocations because it repeatedly resolved field metadata for reflected structs.

### P2: Formatting helpers allocated in hot paths

The generator used repeated `fmt.Sprintf`, `fmt.Fprintf`, and `strings.Join` with quoted enum slices in code paths that run for every SDK generation. This was not an end-to-end bottleneck for `cmd/agh-codegen check`, but it was unnecessary package-local allocation churn.

### P2: Field rendering and primitive classification were duplicated

Named interface rendering and inline object rendering both wrote field specs by hand. Primitive kind mapping was duplicated across TypeScript kind rendering, primitive aliases, and primitive alias detection.

## Changes Made

### Correctness

- Replaced the hard-coded `hookEventFamilyValues` list with runtime-derived values from `hooks.AllHookEvents()` plus `HookEvent.Family()`.
- Regenerated `sdk/typescript/src/generated/contracts.ts`.
- The generated `HookEventFamily` union now includes:
  - `sandbox`
  - `automation`
  - `coordinator`
  - `task.run`
  - `spawn`
  - `network`
- Added semantic tests that compare:
  - `hookEventFamilyValues()` to runtime-derived families.
  - the generated `HookEventFamily` TypeScript union to runtime-derived families.

### Refactoring

- Added a per-generator `fieldSpecs` cache keyed by `reflect.Type`, cached only after successful field resolution.
- Extracted common rendering helpers:
  - `renderTypeAlias`
  - `renderInterface`
  - `renderInlineObject`
  - `writeFieldSpec`
- Extracted primitive kind classification into `primitiveKindTSType`.
- Replaced enum union slice construction with direct union rendering.
- Replaced hot `fmt.Fprintf` map rendering with explicit builder writes.
- Added builder `Grow` estimates for generated interface, inline object, enum union, hook maps, and host method maps.
- Added a quoted-string writer with a no-allocation fast path for the generator's simple string literals.

### Tests

- Normalized touched tests in `generate_test.go` to AGH `t.Run("Should ...")` shape.
- Added generated-union semantic assertions for hook families.
- Kept existing behavior tests for named base type resolution, JSON tags, embedded struct flattening, composite TypeScript types, and list-result detection.

## Generated Output

Updated:

- `sdk/typescript/src/generated/contracts.ts`

Relevant diff:

```ts
export type HookEventFamily =
  | "session"
  | "sandbox"
  | "input"
  | "prompt"
  | "event"
  | "automation"
  | "agent"
  | "turn"
  | "message"
  | "tool"
  | "permission"
  | "context"
  | "coordinator"
  | "task.run"
  | "spawn"
  | "network";
```

## Performance Results

Final benchmark command:

```bash
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench . -benchmem -count=5
```

Final observed results:

- `BenchmarkGenerate`: about `590-648 us/op`, `684533-684542 B/op`, `1161 allocs/op`.
- `BenchmarkStructFieldsPromptPayload`: mostly `52-64 us/op` with one noisy outlier, `134696 B/op`, `58 allocs/op`.

Compared with baseline:

- `BenchmarkGenerate` allocations dropped from about `1.052 MB/op` and `4263 allocs/op` to about `685 KB/op` and `1161 allocs/op`.
- `BenchmarkStructFieldsPromptPayload` allocations dropped from `201 allocs/op` to `58 allocs/op`.
- No claim is made that this materially accelerates the full `cmd/agh-codegen check` path; the measured end-to-end caller is dominated by TypeScript formatting and broader codegen work. The optimization is package-local allocation cleanup for a generator whose contract surface is growing.

Final allocation profile:

```bash
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench '^BenchmarkGenerate$' -benchmem -benchtime=100x -memprofile=/tmp/sdkts-018-generate-final.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_space /tmp/sdkts-018-generate-final.mem
rtk proxy go tool pprof -top -nodecount=20 -sample_index=alloc_objects /tmp/sdkts-018-generate-final.mem
```

Post-change profile still shows `structFields` and builder growth as the largest remaining allocation nodes. Further reductions would require more invasive generator architecture changes or a different contract-rendering strategy, which is not justified in this package iteration.

## Deferred / Cross-Package Notes

- A similar hard-coded `hookEventFamilyValues` list exists in `internal/api/spec`. It was not changed in this iteration because the loop rule is one Go package per run. This should be revisited before the overall refacs goal is considered complete, because it may affect the OpenAPI enum surface.
- Full multi-file decomposition of `generate.go` is deferred. The current iteration extracted the highest-value helpers while keeping the package API stable.
- Package-aware auto-emitted TypeScript type names remain a latent risk if two auto-emitted internal structs from different packages share the same Go type name. No active collision was found in this package iteration.

## Validation

Final validation commands:

```bash
rtk go test ./internal/codegen/sdkts -count=1
rtk golangci-lint run ./internal/codegen/sdkts
rtk proxy go test ./internal/codegen/sdkts -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/codegen/sdkts -count=1
rtk go test -tags integration ./internal/codegen/sdkts -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/generate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/perf_bench_test.go
rtk go run ./cmd/agh-codegen sdk-contracts
rtk make codegen-check
rtk go run ./cmd/agh-codegen check
rtk go test ./internal/codegen/sdkts ./cmd/agh-codegen -count=1
rtk golangci-lint run ./internal/codegen/sdkts ./cmd/agh-codegen
rtk make bun-typecheck
rtk make bun-test
rtk proxy go test -tags mage . -count=1
rtk make verify
```

Observed final results:

- Package tests: `29 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `91.8% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `29 passed in 1 packages`.
- AGH test-shape checks: passed for `generate_test.go` and `perf_bench_test.go`.
- `make codegen-check`: passed.
- `go run ./cmd/agh-codegen check`: passed.
- Direct dependent codegen package set: `70 passed in 2 packages`.
- Direct dependent lint set: no issues.
- Bun typecheck: passed.
- Bun tests: `357` files and `2233` tests passed.
- Mage-tag root tests: passed.
- `make verify`: passed.

