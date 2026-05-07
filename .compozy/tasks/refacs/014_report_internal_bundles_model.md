# Iteration 014 Report: `internal/bundles/model`

## Scope

- Package: `github.com/pedronauck/agh/internal/bundles/model`
- Deterministic order index: 14 from `rtk go list ./internal/...`
- Next package: `github.com/pedronauck/agh/internal/cli`
- Analysis modes: `$refactoring-analysis`, `$extreme-software-optimization`, `$systematic-debugging`, `$no-workarounds`
- Subagents: read-only refactoring explorer and read-only performance explorer

## Baseline

Commands run before implementation:

```bash
rtk go test ./internal/bundles/model -count=1
rtk golangci-lint run ./internal/bundles/model
rtk proxy go test ./internal/bundles/model -cover -count=1
rtk go test ./internal/bundles -count=1
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(Build|ListActivations)LargeCatalog$' -benchmem -count=1
```

Observed baseline:

- Package tests: no package-local tests were present at the start of the iteration.
- Package lint: no issues.
- Package coverage: effectively uncovered by direct package tests.
- Caller benchmarks showed the meaningful hot path is `internal/bundles`, not `internal/bundles/model`.
- CPU and allocation profiles did not show `internal/bundles/model` as a production hotspot.

## Findings

### P0: validation accepted normalized scope values without returning canonical models

`Scope.Validate` accepted values after `Scope.Normalize()`, so values such as `Scope(" WORKSPACE ")` were valid. Some caller paths later compared raw `Scope` values when materializing automation jobs, triggers, and bridge instances. That created a domain contract bug: validation could accept a workspace activation that later materialized as global if a non-canonical `Activation` entered through a store or projection boundary.

Implemented:

- Added `Activation.Normalize()` and `Activation.Validated()` as model-owned canonicalization APIs.
- Made `Activation.Validate()` validate the normalized value.
- Used `Activation.Validated()` in `ResourceStore.CreateBundleActivation`, `ResourceStore.UpdateBundleActivation`, `Service.resolveActivation`, and `Service.resolveActivationFromBundleLookup`.
- Made automation and bridge scope conversion helpers compare `scope.Normalize()` instead of raw enum values.
- Added a parent-package regression test proving a validated `Scope(" WORKSPACE ")` activation materializes agent, automation, and bridge resources with workspace scope.

### P1: missing direct model tests

`internal/bundles/model` defines the activation/inventory validation contract, but had no package-local tests. Parent package tests covered higher-level behavior, not the model boundary itself.

Implemented:

- Added `internal/bundles/model/model_test.go`.
- Covered `Scope.Normalize`, `Scope.Validate`, `Activation.Normalize`, `Activation.Validate`, `Activation.Validated`, `InventoryItem.Validate`, and `InventoryItem.Validated`.
- Covered empty scope as a distinct validation error instead of collapsing it into unsupported scope.
- Package-local coverage is now `98.1% of statements`.

### P1: repeated required-field checks obscured the model contract

`Activation.Validate` and `InventoryItem.Validate` repeated the same trim/empty check pattern. The repetition was small, but it made the validation contract harder to extend consistently after adding canonicalization.

Implemented:

- Added a shared `requireNonEmpty` helper.
- Added private normalized-validation helpers so `Validate()` and `Validated()` share one validation body without double-normalizing.
- Added `InventoryItem.Normalize()` and `InventoryItem.Validated()` so inventory data has the same canonicalization shape as activations.

### P2: inventory validation exists but remains mostly a boundary contract

The refactoring explorer confirmed `InventoryItem.Validate()` was not broadly used by production callers. This iteration kept it because the type is part of the bundle activation model contract and now has direct tests. `RecordedAtUTC` remains optional metadata: desired-state materializers build inventory before persistence timestamps exist, while `ResourceStore` inventory listings can populate it from resource update timestamps.

Deferred a larger constructor/materializer rewrite because forcing inventory validation in every caller would cross the package boundary and could change existing behavior for resources whose display names are intentionally empty or derived later.

## Performance Findings

No model-specific performance patch was warranted.

The performance explorer profiled caller benchmarks and found:

- `internal/bundles/model` does not appear as a CPU or allocation hotspot.
- The measurable costs remain in broader bundle service work: JSON/hash canonicalization, defensive clones, and resource materialization.
- Optimizing `Scope.Normalize`, required-field checks, or validation error allocation would not affect the steady-state benchmark path in a meaningful way.

Final caller benchmark check after this iteration:

```bash
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(Build|ListActivations)LargeCatalog$' -benchmem -count=5
```

Observed after results:

- `BenchmarkServiceListActivationsLargeCatalog`: about `0.97-1.43 ms/op`, `1.539 MB/op`, `10643 allocs/op`
- `BenchmarkServiceBuildLargeCatalog`: about `1.03-1.08 ms/op`, `1.449 MB/op`, `9160-9161 allocs/op`

The small variance is in line with the prior caller benchmark range. The iteration's performance decision was to avoid unproven model micro-optimizations and fix the correctness/refactoring issue instead.

## Deferred

- Collapsing `internal/bundles/model` into `internal/bundles` remains a possible cleanup because the subpackage is small and mostly re-exported by the parent. Deferred because the new canonicalization API gives the boundary real responsibility, and deleting the subpackage would be a broader structural move.
- Structured validation sentinel errors are deferred until callers need stable `errors.Is` classification for validation failures.
- Broad inventory-constructor adoption is deferred until a caller-level pass proves real drift or invalid inventory creation. This iteration documented and tested the model contract without forcing behavior changes across all materializers.
- Additional model microbenchmarks are deferred because caller benchmarks and profiles are more representative.

## Behavior Proof

- Ordering preserved: yes. No activation, inventory, resource, or catalog ordering logic changed.
- Scope behavior: improved. Validated activations now carry canonical scope values before persistence/resolution/materialization.
- ID compatibility: preserved. `ActivationResourceID` already normalizes scope and workspace ID; no stable ID format changed in this iteration.
- Inventory timestamp semantics: preserved. `RecordedAtUTC` remains optional metadata.
- Error semantics: improved for empty scope, which now returns `bundles: scope is required`.
- Public API shape: unchanged. Existing aliases in `internal/bundles` continue to expose the same types.

## Files Changed

- `internal/bundles/model/model.go`
- `internal/bundles/model/model_test.go`
- `internal/bundles/resource_store.go`
- `internal/bundles/resource_projection.go`
- `internal/bundles/service.go`
- `internal/bundles/service_refac_test.go`

## Validation

Final validation commands:

```bash
rtk go test ./internal/bundles/model -count=1
rtk go test ./internal/bundles -run 'Test(ServiceRefacs|Scope|Activation|Inventory)' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/model/model_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/bundles/service_refac_test.go
rtk golangci-lint run ./internal/bundles/model
rtk go test ./internal/bundles/model ./internal/bundles -count=1
rtk golangci-lint run ./internal/bundles/model ./internal/bundles
rtk proxy go test ./internal/bundles/model -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/bundles/model ./internal/bundles -count=1
rtk go test -tags integration ./internal/bundles/model ./internal/bundles -count=1
rtk go test ./internal/bundles ./internal/api/core ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon -count=1
rtk proxy go test ./internal/bundles -run '^$' -bench 'BenchmarkService(Build|ListActivations)LargeCatalog$' -benchmem -count=5
rtk make verify
```

Observed results:

- Model package tests: `34 passed in 1 packages`
- Focused parent regression tests: `6 passed in 1 packages`
- Combined package tests: `101 passed in 2 packages`
- Integration-tag package tests: `104 passed in 2 packages`
- Race package tests: passed
- Package lint: no issues
- Package-local model coverage: `98.1% of statements`
- Direct dependent set: `1828 passed in 5 packages`
- `make verify`: passed
