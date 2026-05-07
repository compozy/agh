# Refacs Iteration 003: `internal/api/contract`

Date: 2026-05-06

Package: `github.com/pedronauck/agh/internal/api/contract`

## Scope

This iteration audited the public Go contract package used by daemon, CLI, HTTP/UDS, extension host APIs, and generated OpenAPI surfaces. The goal was to find necessary refactoring and performance work without changing DTO field names, JSON tags, enum strings, response shapes, or generated contract output.

Read-only subagents were used for:

- Refactoring analysis of Fowler-style smells, dead aliases, large modules, duplicated traversal logic, and test-shape risks.
- Performance analysis with package tests, repeated runs, CPU/memory profiles, and dependent package smoke tests.

## Baseline

- `rtk go test ./internal/api/contract -count=1` passed before edits.
- `rtk go test -tags integration ./internal/api/contract -count=1` passed before edits.
- The package had no existing benchmarks.
- Repeated test profiling showed no production CPU hotspot. Allocation samples were dominated by test harnesses and expected JSON security checks.
- Local coverage before this iteration was around `60.7%` for the package.

## Findings

### Fixed: duplicated recursive JSON safety traversal

`agents.go` and `authored_context.go` both carried nearly identical recursive JSON traversal code:

- `containsRawClaimTokenJSON` scanned object and array trees for unsafe `claim_token` keys.
- `containsUnsafeAuthoredContextJSON` scanned object and array trees, then string leaves, for authored-context credential and prompt leaks.

The policies are different, but the traversal mechanics were duplicated and expensive because each recursive level repeatedly attempted fresh `json.Unmarshal` passes into different shapes.

Fix:

- Added `internal/api/contract/json_safety.go`.
- Extracted a single unexported `containsUnsafeJSON` traversal with strategy predicates for key and string policies.
- Kept `isRawClaimTokenKey`, `isUnsafeAuthoredContextKey`, and `isUnsafeAuthoredContextString` policy-specific and unchanged.
- Updated `containsRawClaimTokenJSON` and `containsUnsafeAuthoredContextJSON` to delegate to the shared traversal.
- Added `internal/api/contract/json_safety_bench_test.go` to keep this safety path measurable.

Contract impact:

- No DTO fields changed.
- No JSON tags changed.
- No enum values changed.
- No generated OpenAPI drift was expected or observed.
- Error wrapping and public exported APIs stayed unchanged.

### Not fixed in this iteration: `NetworkChannelMessagePayload` compatibility alias

The refactoring subagent found `NetworkChannelMessagePayload` in `contract.go` as an internal compatibility alias for `NetworkConversationMessagePayload`.

This should be removed, but the active call sites are mostly in `internal/api/core/network_details.go`, `internal/api/core` tests, and daemon network integration tests. Those touched test files currently have pre-existing AGH test-convention violations, so deleting the alias cleanly would require a broader core/network cleanup that belongs with the upcoming `internal/api/core` package iteration rather than the current contract-only change.

Decision:

- Keep the alias for now.
- Record it as a carry-forward cleanup item for the next iteration.
- Do not introduce another alias or compatibility bridge.

### Not fixed in this iteration: large mechanical file splits

The subagents identified `contract.go` and `authored_context.go` as large modules with multiple contract surfaces. Splitting them into same-package files would improve navigation, but it is a broad mechanical change with high review noise and no behavior or performance benefit for this iteration.

Decision:

- Do not split `contract.go` or `authored_context.go` in this pass.
- Prefer future splits only when paired with a concrete surface cleanup, deleted alias, or behavior-preserving package boundary improvement.

## Performance Evidence

Benchmarks were added before the production refactor and run again after the refactor.

Command:

```bash
rtk proxy go test -run '^$' -bench 'Benchmark(ContainsRawClaimTokenFieldNestedPayload|ValidateAuthoredContextRedactedNestedPayload)' -benchmem ./internal/api/contract -count=5
```

Baseline:

| Benchmark | Mean ns/op | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkContainsRawClaimTokenFieldNestedPayload` | ~533,380 | ~394,328 | 6,133 |
| `BenchmarkValidateAuthoredContextRedactedNestedPayload` | ~606,747 | ~458,762 | ~7,248 |

After refactor:

| Benchmark | Mean ns/op | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkContainsRawClaimTokenFieldNestedPayload` | ~222,195 | ~101,185 | 2,457 |
| `BenchmarkValidateAuthoredContextRedactedNestedPayload` | ~255,259 | ~101,205 | 2,458 |

Observed effect:

- `ContainsRawClaimTokenField` benchmark: about `2.4x` faster, about `74%` fewer allocated bytes, about `60%` fewer allocations.
- `ValidateAuthoredContextRedacted` benchmark: about `2.4x` faster, about `78%` fewer allocated bytes, about `66%` fewer allocations.

This is not a speculative hot-path rewrite. The change removes duplicated traversal code and reduces measurable cost in existing security guardrail functions while preserving the same exported APIs.

## Validation

Targeted validation:

```bash
rtk go test ./internal/api/contract -count=1
rtk go test -tags integration ./internal/api/contract -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/contract -count=1
rtk golangci-lint run ./internal/api/contract
rtk make codegen-check
rtk go test ./internal/api/contract ./internal/api/core ./internal/extension ./internal/cli ./internal/daemon -count=1
rtk proxy go test ./internal/api/contract -cover -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/contract/json_safety_bench_test.go
rtk proxy go test -run '^$' -bench 'Benchmark(ContainsRawClaimTokenFieldNestedPayload|ValidateAuthoredContextRedactedNestedPayload)' -benchmem ./internal/api/contract -count=5
```

Results:

- Package tests passed: `142 passed in 1 packages`.
- Integration package tests passed: `146 passed in 1 packages`.
- Race package tests passed.
- Package lint passed with no issues.
- `make codegen-check` passed with no generated drift.
- Dependent package smoke tests passed: `2667 passed in 5 packages`.
- Coverage after the refactor: `59.9% of statements`.
- New benchmark file passes the AGH test-conventions checker.
- Full gate: `rtk make verify` passed.

## Next Package

Next deterministic package:

```text
github.com/pedronauck/agh/internal/api/core
```

Carry-forward item for that package:

- Remove `NetworkChannelMessagePayload` compatibility alias by renaming `internal/api/core` network helper return types and affected tests to `NetworkConversationMessagePayload`, while normalizing touched tests to AGH test conventions.
