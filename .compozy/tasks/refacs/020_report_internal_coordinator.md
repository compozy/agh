# Refacs 020: `internal/coordinator`

## Scope

- Package: `github.com/pedronauck/agh/internal/coordinator`
- Iteration: 020
- Goal: deep refactoring and performance audit for coordinator bootstrap, permission policy, lineage, prompt overlay, and coordinator health helpers.
- Subagents:
  - Read-only refactoring audit for `internal/coordinator`.
  - Read-only performance audit for `internal/coordinator`.

## Baseline

Commands run before changes:

```bash
rtk go test ./internal/coordinator -count=1
rtk golangci-lint run ./internal/coordinator
rtk proxy go test ./internal/coordinator -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/coordinator -count=1
rtk go test -tags integration ./internal/coordinator -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/coordinator/coordinator_test.go
rtk proxy go test ./internal/coordinator -run '^$' -bench . -benchmem -count=3
```

Observed baseline:

- Package tests passed: `10` tests.
- Package lint passed with no issues.
- Coverage: `86.7% of statements`.
- Race package tests passed.
- Integration-tag package tests passed: `10` tests.
- The package had no benchmarks.
- AGH test-shape checker failed because `TestPermissionPolicyRestrictsCoordinatorSurface`, `TestLineageAndHealthySession`, and `TestPromptOverlayUsesPublicAPIsAndRunChannel` asserted directly without `t.Run("Should ...")` subtests.

## Findings

### P1: Coordinator allowlist globals were exported mutable slices

`OperationalMessageKinds` and `ToolAllowlist` were exported package-level slices. They controlled prompt guidance and coordinator session permission policy. Any same-process internal caller could mutate those slices and change coordinator permissions or prompt content for future calls.

Impact: coordinator permission and prompt policy should be stable package constants in practice. Mutable exported slices made those contracts corruptible by accidental caller mutation.

### P1: `writePromptLine` discarded errors through `_`

`writePromptLine` used `fmt.Fprintf` against a `strings.Builder` and discarded both return values with `_`. Even though `strings.Builder` writes cannot fail through this path, the production-code rule forbids underscore-discarded errors. The generic formatting call was also unnecessary for a simple fixed line shape.

### P1: Existing tests failed AGH test-shape conventions

Three tests asserted directly without `Should ...` subtests. This made the package fail the AGH test-conventions helper once the file was touched in this iteration.

### P2: No package-local benchmarks existed

The performance subagent correctly found that pprof would not be useful without a reproducible package-local workload. The package is small and called around heavier daemon operations, but benchmarks are still useful to lock down the pure helper surfaces.

### P2: `Decision` and `PromptInput` duplicate the same field group

`Decision` and `PromptInput` carry the same coordinator context field group, and the daemon maps `Decision` into `PromptInput` manually. This is a data-clump smell, but not changed in this iteration because it crosses into `internal/daemon` and is lower priority than immutable policy surfaces.

## Changes Made

### Correctness and hardening

- Replaced exported mutable slices with unexported arrays:
  - `operationalMessageKinds`
  - `toolAllowlist`
- Added exported accessors returning defensive copies:
  - `OperationalMessageKinds() []string`
  - `ToolAllowlist() []string`
- Updated internal use sites to rely on the private arrays directly where mutation is impossible.
- Updated `PermissionPolicy` to use `ToolAllowlist()` so policy callers receive an independent normalized copy.
- Rewrote `writePromptLine` with explicit `strings.Builder` writes and removed the `fmt` import plus underscore discard.

### Tests

- Normalized all touched tests to AGH `t.Run("Should ...")` shape.
- Added `TestCoordinatorListAccessorsReturnCopies` to prove caller mutation of returned tool/message slices does not affect:
  - `ToolAllowed`
  - `PermissionPolicy`
  - `PromptOverlay`
- Kept existing behavior coverage for bootstrap decisions, permission surface, lineage, healthy-session filtering, and prompt content.

### Benchmarks

Added `coordinator_bench_test.go` with package-local benchmarks for:

- `BenchmarkPromptOverlay`
- `BenchmarkPermissionPolicy`
- `BenchmarkLineage`

The performance subagent did not recommend deeper optimization because there was no evidence that coordinator pure helpers are runtime hotspots. The benchmarks are added as measurement infrastructure for future changes.

## Performance Results

Final benchmark command:

```bash
rtk proxy go test ./internal/coordinator -run '^$' -bench . -benchmem -count=5
```

Observed final results:

- `BenchmarkPromptOverlay`: noisy local timing, about `683 ns/op` to `2190 ns/op`, `2288 B/op`, `6 allocs/op`.
- `BenchmarkPermissionPolicy`: about `1013-1178 ns/op`, `432 B/op`, `4 allocs/op`.
- `BenchmarkLineage`: about `721-873 ns/op`, `456 B/op`, `4 allocs/op`.

No performance victory is claimed beyond removing unnecessary generic formatting in `writePromptLine`. The package is not currently a measured daemon hotspot, and the added benchmarks are primarily guardrails.

## Deferred / Cross-Package Notes

- A small `Decision -> PromptInput` mapper could remove duplicated field mapping in the daemon, but it crosses package boundaries and was not needed for the current fixes.
- `workflowIDFromMetadata` could decode into a tiny typed struct instead of `map[string]any`, but the performance subagent scored it below the implementation threshold without a workload showing `DecideBootstrap` as a hotspot.
- If coordinator grows further, split tests by responsibility (`bootstrap`, `permissions`, `lineage`, `prompt`) before the package test file becomes a mixed contract bundle.

## Validation

Final validation commands:

```bash
rtk go test ./internal/coordinator -count=1
rtk golangci-lint run ./internal/coordinator
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/coordinator/coordinator_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/coordinator/coordinator_bench_test.go
rtk proxy go test ./internal/coordinator -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/coordinator -count=1
rtk go test -tags integration ./internal/coordinator -count=1
rtk proxy go test ./internal/coordinator -run '^$' -bench . -benchmem -count=5
rtk go test ./internal/daemon -run 'TestCoordinatorRuntime' -count=1
rtk go test -tags integration ./internal/daemon -run 'TestDaemonE2E.*Coordinator|TestCoordinatorRuntime' -count=1
rtk go test ./internal/daemon ./internal/task ./internal/session -run 'Coordinator|coordinator|Bootstrap|bootstrap|ExecutableRun|HealthySession' -count=1
rtk rg -n "_\\s*=" internal/coordinator --glob '*.go'
rtk make verify
```

Observed final results:

- Package tests: `15 passed in 1 packages`.
- Package lint: no issues.
- AGH test-shape checks: passed for `coordinator_test.go` and `coordinator_bench_test.go`.
- Package coverage: `88.8% of statements`.
- Race package tests: passed.
- Integration-tag package tests: `15 passed in 1 packages`.
- Package benchmarks: passed.
- Focused daemon coordinator runtime tests: `11 passed in 1 packages`.
- Focused daemon coordinator integration tests: `11 passed in 1 packages`.
- Focused daemon/task/session dependent set: `24 passed in 3 packages`.
- Production/test `_ =` scan in `internal/coordinator`: no matches.
- `make verify`: passed.
