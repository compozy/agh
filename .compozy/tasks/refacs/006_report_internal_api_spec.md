# Refacs Run 006: `internal/api/spec`

## Package

- Import path: `github.com/pedronauck/agh/internal/api/spec`
- Directory: `internal/api/spec`
- Status: completed
- Date: 2026-05-06

## Scope

This run audited the OpenAPI/spec generation package for refactoring and performance opportunities. The package owns the canonical REST operation registry, schema generation, OpenAPI document rendering, and contract metadata consumed by HTTP/UDS route-parity tests and codegen.

Because `internal/api/spec` is a contract/codegen surface, this iteration avoided literal contract edits. The implemented change is behavior-preserving for generated OpenAPI output and fixes registry mutability at the exported `Operations()` boundary.

## Baseline

- `rtk go test ./internal/api/spec -count=1`: passed before edits (`151 passed`)
- `rtk go test -tags integration ./internal/api/spec -count=1`: passed before edits (`151 passed`)
- `rtk golangci-lint run ./internal/api/spec`: passed before edits
- `rtk proxy go test ./internal/api/spec -cover -count=1`: passed before edits (`94.2% of statements`)
- `rtk env CGO_ENABLED=1 go test -race ./internal/api/spec -count=1`: passed before edits
- `rtk make codegen-check`: passed before edits
- No package benchmarks existed: `rtk proxy go test -run '^$' -bench . -benchmem ./internal/api/spec -count=1` passed with no benchmarks.

## Subagent Findings

### Refactoring Explorer

- Found a necessary P1 fix: `Operations()` returned only a shallow top-level copy of the operation registry. Nested slices and maps inside `OperationSpec` remained shared with package-global registry state.
- Confirmed the issue affected `Tags`, `Transports`, `Parameters`, `ParameterSpec.Enum`, `Responses`, and the map request bodies used for webhook operations.
- Recommended keeping the defensive-copy fix/test as the scoped iteration change and deferring broader registry/test reshaping.
- Deferred splitting the monolithic `operationRegistry` into per-surface files. That is valid cleanup, but high-churn and best done as a mechanical move with byte-for-byte OpenAPI proof.
- Deferred response/parameter DSL deduplication because helper abstraction could accidentally change operation ordering, response descriptions, content types, required flags, or parameter semantics.

### Performance Explorer

The read-only performance subagent did not return a final summary after two long waits and was shut down. Local performance evidence was collected directly:

- `rtk hyperfine --warmup 3 --runs 10 'rtk go test ./internal/api/spec -run . -count=1'`
  - Mean: `518.9 ms ± 6.4 ms`
- `rtk proxy go test ./internal/api/spec -run '^TestDocumentTracksRequiredFieldsAndEnums$' -count=30 -cpu=1 -outputdir /tmp -cpuprofile /tmp/api-spec-006.cpu -memprofile /tmp/api-spec-006.mem -memprofilerate=1`
  - Passed after profiling.
- `rtk go tool pprof -top /tmp/api-spec-006.cpu`
  - Production cost was dominated by `openapi3gen` schema generation and OpenAPI validation. No AGH-local request/runtime hotspot was identified.
- `rtk go tool pprof -top /tmp/api-spec-006.mem`
  - Allocation pressure was dominated by `github.com/getkin/kin-openapi/openapi3gen.(*Generator).generateWithoutSaving` and related schema generation.
  - Post-fix `Operations()` cloning showed a small allocation footprint relative to schema generation, and it is required for correctness.

No performance-only change met the implement-now threshold. Caching schema generation or weakening validation would be contract-sensitive and was not justified by this iteration's profile.

## Root Cause

`Operations()` and `authoredContextOperations()` copied the top-level `[]OperationSpec`, but `OperationSpec` contains mutable reference fields:

- `Tags []string`
- `Transports []Transport`
- `Parameters []ParameterSpec`
- `ParameterSpec.Enum []string`
- `Responses []ResponseSpec`
- `RequestBody any`, including `map[string]any{}`
- `ResponseSpec.Body any`

As a result, any caller receiving `Operations()` could mutate nested fields and contaminate later calls to `Operations()`, `Document()`, route-parity tests, or codegen in the same process.

## Changes Implemented

### Defensive Operation Registry Copies

`Operations()` and `authoredContextOperations()` now return defensive copies of operation specs rather than shallow copies.

Added helpers:

- `cloneOperationSpecs`
- `cloneOperationSpec`
- `cloneParameterSpecs`
- `cloneResponseSpecs`
- `cloneSpecValue`

The clone path copies mutable operation metadata and currently clones `map[string]any` request/response bodies used by the spec registry. Value-like contract DTO zero values remain unchanged.

### Regression Test

Added `internal/api/spec/operations_refac_test.go` with `TestOperationsReturnDefensiveCopies`.

The test first reproduced the bug by mutating returned operation metadata, then verified subsequent `Operations()` calls preserve the original registry values:

- operation tags
- transports
- parameter names
- parameter enum values
- response status values
- map request body entries

## Deferred Findings

- Split the large `operationRegistry` in `spec.go` into per-surface operation files. This should be a dedicated mechanical refactor with before/after OpenAPI byte comparison.
- Split the broad `TestDocumentTracksRequiredFieldsAndEnums` table into focused surface tests after the registry file split settles.
- Avoid broad helper/DSL deduplication for common responses and parameters until there is a contract-change reason. Explicit literals are noisy but safer for OpenAPI review.
- Do not optimize `openapi3gen` schema generation or skip OpenAPI validation in this run. The cost is generator-dominated and not a daemon runtime hot path.

## Contract And Codegen Impact

- No OpenAPI contract shape change was intended.
- `rtk make codegen` produced no source-control diff for `openapi`, generated web types, or generated site API references.
- `rtk make codegen-check` passed after the change.
- Web contract consumers were verified with `rtk make web-typecheck` and `rtk make web-test`.

Note: an initial parallel run of `make codegen-check`, `make web-typecheck`, and `make web-test` caused a transient Mage output-file race (`mage_output_file.go`). The failing target was rerun sequentially and passed. The root cause was concurrent invocation of the same codegen-check toolchain, not a code regression.

## Validation

```bash
rtk go test ./internal/api/spec -run '^TestOperationsReturnDefensiveCopies$' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/spec/operations_refac_test.go
rtk golangci-lint run ./internal/api/spec
rtk go test ./internal/api/spec -count=1
rtk go test -tags integration ./internal/api/spec -count=1
rtk make codegen-check
rtk go test ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi ./internal/codegen/openapits ./internal/codegen/sdkts -run 'Test(OperationsReturnDefensiveCopies|OperationsRemainUniqueWithExpandedTaskSurface|Document|Codegen|OpenAPI|Transport|Resource)' -count=1
rtk hyperfine --warmup 3 --runs 10 'rtk go test ./internal/api/spec -run . -count=1'
rtk proxy go test ./internal/api/spec -run '^TestDocumentTracksRequiredFieldsAndEnums$' -count=30 -cpu=1 -outputdir /tmp -cpuprofile /tmp/api-spec-006.cpu -memprofile /tmp/api-spec-006.mem -memprofilerate=1
rtk go tool pprof -top /tmp/api-spec-006.cpu
rtk go tool pprof -top /tmp/api-spec-006.mem
rtk make codegen
rtk go test ./internal/api/spec ./internal/api/httpapi ./internal/api/udsapi ./internal/codegen/openapits ./internal/codegen/sdkts -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/spec -count=1
rtk proxy go test ./internal/api/spec -cover -count=1
rtk make web-test
rtk make web-typecheck
rtk make verify
```

Observed results:

- Focused regression test: `3 passed`
- AGH test-conventions checker: passed for `operations_refac_test.go`
- Unit package after fix: `154 passed`
- Integration-tag package after fix: `154 passed`
- Consumer packages: `613 passed in 5 packages`
- Race package: passed
- Coverage: `93.8% of statements`
- `golangci-lint`: no issues
- `make codegen`: passed
- `make codegen-check`: passed
- `make web-test`: `229` files passed, `1661` tests passed
- `make web-typecheck`: passed when rerun sequentially
- `make verify`: passed

## Files Changed

- `internal/api/spec/authored_context.go`
- `internal/api/spec/spec.go`
- `internal/api/spec/operations_refac_test.go`

## Next Package

`github.com/pedronauck/agh/internal/api/testutil`
