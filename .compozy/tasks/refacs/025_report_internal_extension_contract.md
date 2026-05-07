# Refacs 025: `internal/extension/contract`

## Scope

- Package: `github.com/pedronauck/agh/internal/extension/contract`
- Iteration: 025
- Goal: deep refactoring and performance audit for the extension Host API / SDK contract registry.
- Subagents:
  - Read-only refactoring/correctness audit for `internal/extension/contract`.
  - Read-only performance/concurrency audit for `internal/extension/contract` and immediate call paths.

## Baseline

Initial package state:

```bash
rtk go test ./internal/extension/contract -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/extension/contract -count=1
rtk golangci-lint run ./internal/extension/contract
rtk proxy go test ./internal/extension/contract -cover -count=1
rtk proxy go test ./internal/extension/contract -run '^$' -bench . -benchmem -count=3
rtk make codegen-check
```

Observed baseline:

- Package tests passed with `3` tests.
- Race package tests passed.
- Package lint passed with no issues.
- Package coverage was `76.5% of statements`.
- The package had no registered benchmarks.
- `make codegen-check` passed before local source edits.
- Existing package-local tests passed but failed AGH test-shape heuristics because they asserted directly in top-level tests without `t.Run("Should ...")` subtests.

## Findings

### P1: event `since` filters were typed as required in the generated SDK

`SessionEventsParams.Since` and `ObserveEventsParams.Since` had `json:"since"` tags even though the runtime handlers treat omitted `since` as a valid zero-value filter.

Impact: extension authors using the generated TypeScript SDK had to supply `since` for `sessions/events` and `observe/events` even though the Host API accepts omitted values.

Root cause: the Go contract tags did not mark these filter fields with an optional JSON tag, so the SDK generator emitted required `since: ISODateTime` fields. The final fix uses `omitzero` because Go's modernize formatter removes `omitempty` from `time.Time` fields.

### P1: generated SDK drift appeared as soon as the contract tag changed

After making the Go contract truthful, `sdk/typescript/src/generated/contracts.ts` was stale until `make codegen` ran.

Impact: `make codegen-check` correctly rejected the stale generated SDK. Without co-shipping generated output, Go and TypeScript extension contracts would disagree.

Root cause: this package is a generator root for extension contracts, so even small JSON tag changes must run the codegen co-ship path.

### P2: hook contract drift failed as an uncontrolled panic in codegen

`HookContracts()` looked up every hook descriptor payload/patch schema in the manual `namedHookTypes` registry and called `panic(err)` on missing mappings.

Impact: a future hook descriptor drift would crash `cmd/agh-codegen` rather than returning an actionable, wrapped error that identifies the event and missing schema side.

Root cause: the package exposed only a must-style helper even though the primary production call path is the SDK generator, which already propagates errors.

### P3: package-local tests did not cover contract registry invariants

The original tests checked Host API method order, one JSON tag, and representative hook contracts. They did not assert defensive copy behavior, optionality-sensitive event filters, full hook descriptor coverage, or the error path for unknown hook type names.

Impact: contract drift could pass the package unit suite and fail later in codegen or extension consumer typechecking.

Root cause: the tests were smoke tests rather than registry invariants.

## Changes Made

### Truthful optional event filters

- Changed `SessionEventsParams.Since` to `json:"since,omitzero"`.
- Changed `ObserveEventsParams.Since` to `json:"since,omitzero"`.
- Updated the TypeScript SDK generator so `json:",omitzero"` is treated as an optional wire field, matching `omitempty` and pointer fields.
- Added a package-local test proving both contract fields keep optional JSON tags.
- Ran `make codegen`; the final generated diff is the TypeScript extension SDK contract.

### Hook contract error path

- Added `BuildHookContracts() ([]HookContractSpec, error)`.
- Changed hook payload/patch lookup failures to return wrapped errors containing the hook event and whether the missing type came from the payload or patch schema.
- Kept `HookContracts()` as a compatibility wrapper over `BuildHookContracts()`.
- Updated `internal/codegen/sdkts` to use `BuildHookContracts()` and return `build hook contracts: ...` errors instead of depending on a panic.
- Updated direct `sdkts` tests and benchmarks for the `newGenerator() (*generator, error)` signature.

### Contract invariant tests

- Normalized package-local tests to AGH `Should ...` subtest shape.
- Added defensive-copy coverage for `HostAPIMethodSpecs()`.
- Expanded hook contract coverage so every descriptor from `hooks.AllEventDescriptors()` is checked against the generated payload/patch contract names.
- Added defensive-copy coverage for `SDKRootTypes()`.
- Added error-path coverage for unknown hook contract type names.
- Added compatibility-wrapper coverage to prove `HookContracts()` remains equivalent to `BuildHookContracts()` on the current valid registry.

### Generated artifacts

- Regenerated `sdk/typescript/src/generated/contracts.ts`.
- `openapi/agh.json` was checked by the repo codegen target and had no final diff.
- The SDK output now marks:
  - `ObserveEventsParams.since?: ISODateTime`
  - `SessionEventsParams.since?: ISODateTime`
- The same codegen run also refreshed stale generated `HookEventFamily` values already implied by current Go sources.

## Files Changed

- `internal/extension/contract/host_api.go`
- `internal/extension/contract/host_api_test.go`
- `internal/extension/contract/sdk.go`
- `internal/extension/contract/sdk_test.go`
- `internal/codegen/sdkts/generate.go`
- `internal/codegen/sdkts/generate_test.go`
- `internal/codegen/sdkts/perf_bench_test.go`
- `sdk/typescript/src/generated/contracts.ts`

## Performance Results

The performance subagent found no runtime optimization justified for this package:

- The package owns static DTO and registry metadata only.
- It has no goroutines, locks, I/O, SQLite, or long-lived mutable exported state.
- `HostAPIMethodSpecs()` and `SDKRootTypes()` already return defensive copies, which is intentional contract isolation.
- `HookContracts()` / `BuildHookContracts()` are codegen-time helpers, not daemon hot paths.

Final focused benchmark command:

```bash
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench 'BenchmarkGenerate|BenchmarkStructFieldsPromptPayload' -benchmem -count=3
```

Observed final results:

- `BenchmarkGenerate`: about `587.1-609.4 us/op`, `684550-684558 B/op`, `1161 allocs/op`.
- `BenchmarkStructFieldsPromptPayload`: about `52.9-53.3 us/op`, `134696 B/op`, `58 allocs/op`.

Interpretation:

- The new error-returning hook contract path did not change generated output and did not introduce a meaningful benchmark-level regression.
- Future optimization should target `internal/codegen/sdkts` reflection/string generation only if codegen time becomes a real bottleneck. This package does not currently need performance edits.

## Deferred / Cross-Package Notes

- Runtime Host API dispatch in `internal/extension` still has an independently maintained handler map. A later parity pass should compare `HostAPIHandler.MethodHandlers()` against every `HostAPIMethodSpecs()` entry and replace remaining raw string keys with `extensioncontract.HostAPIMethod*` constants.
- The contract package remains a broad SDK aggregator over several internal domain packages. A full structural split into host API DTOs, SDK root DTOs, hook registry, and protocol registry is valuable but larger than this iteration and should include generated-contract snapshot strategy.
- The compatibility wrapper `HookContracts()` still panics if the registry is invalid, but the production SDK generator now uses `BuildHookContracts()` and receives ordinary errors. The wrapper remains only for existing internal callers/tests.

## Validation

Final validation commands:

```bash
rtk go test ./internal/extension/contract -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/extension/contract -count=1
rtk golangci-lint run ./internal/extension/contract
rtk proxy go test ./internal/extension/contract -cover -count=1
rtk go test ./internal/extension/contract ./internal/extension ./internal/codegen/sdkts ./internal/api/spec -count=1
rtk golangci-lint run ./internal/extension/contract ./internal/codegen/sdkts ./internal/api/spec
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/contract/host_api_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/extension/contract/sdk_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/generate_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/codegen/sdkts/perf_bench_test.go
rtk make codegen
rtk make codegen-check
rtk make bun-typecheck
rtk make bun-test
rtk proxy go test ./internal/codegen/sdkts -run '^$' -bench 'BenchmarkGenerate|BenchmarkStructFieldsPromptPayload' -benchmem -count=3
rtk make verify
```

Observed final results:

- Package tests: `18 passed in 1 packages`.
- Race package tests: passed.
- Package lint: no issues.
- Package coverage: `85.7% of statements`.
- Direct dependent package set (`extension/contract`, `extension`, `codegen/sdkts`, `api/spec`): `731 passed in 4 packages`.
- Direct dependent lint set (`extension/contract`, `codegen/sdkts`, `api/spec`): no issues.
- AGH test-shape checks: passed for all touched Go test/benchmark files.
- `make codegen`: passed.
- `make codegen-check`: passed.
- `make bun-typecheck`: passed across 5 Turbo tasks.
- `make bun-test`: `357 files`, `2233 tests`, all passed.
- Focused SDK TS benchmarks passed with the results listed above.
- `make verify`: passed.
