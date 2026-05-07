# Refactoring and Performance Report: `internal/agentidentity`

> Date: 2026-05-06
> Scope: `github.com/pedronauck/agh/internal/agentidentity`
> Iteration: 002
> Language/Stack: Go, daemon-validated caller identity for agent CLI and UDS operations

## Executive Summary

`internal/agentidentity` is the shared identity-validation boundary for agent-managed CLI and UDS surfaces. It validates untrusted session/agent hints against daemon session state, derives task actor context, and renders deterministic machine-readable identity errors.

The package was already correct and well covered. The necessary refactoring in this iteration was structural: the single `identity.go` file mixed public constants, credentials, session snapshot projection, resolver flow, error payload contracts, JSON rendering, and exit-code mapping. This iteration split those responsibilities into cohesive package files without changing exported API names or runtime behavior. The existing tests were also normalized to AGH test-shape conventions.

Performance analysis found no production hotspot in this package. No performance edit was made.

## Findings

| ID | Priority | Type | Location | Status |
| --- | --- | --- | --- | --- |
| F1 | P2 | Large Module / Mixed Responsibilities | `internal/agentidentity/identity.go` | Fixed |
| F2 | P2 | Test Shape Drift | `internal/agentidentity/identity_test.go` | Fixed |
| F3 | P3 | Repeated identity-error triples | `internal/agentidentity/identity.go` | Left unchanged |
| F4 | P3 | Performance candidates below score threshold | `internal/agentidentity/*` | No-op |

### F1: `identity.go` mixed identity resolution, DTO projection, and error rendering

- Smell: Large Module, Divergent Change
- Before: `identity.go` was 415 lines and owned credentials, session snapshots, lookup validation, actor derivation, error payloads, JSON/JSONL marshaling, and exit-code mapping.
- Change:
  - `credentials.go` now owns `Credentials` and credential normalization.
  - `snapshot.go` now owns `SessionSnapshot`, `SessionLookup`, `SessionSnapshotFromInfo`, and snapshot normalization.
  - `errors.go` now owns identity sentinel errors, `Error`, `ErrorPayload`, JSON/JSONL rendering, and exit-code mapping.
  - `identity.go` now focuses on `Resolve` and its validation flow.
- Result: the public API remains package-local and unchanged, while file-level ownership now matches the package responsibilities described in `internal/CLAUDE.md`.
- Behavior proof: pure file-level move plus import cleanup; package tests, race tests, lint, dependent package tests, and full `make verify` pass.

Current production file sizes:

```text
credentials.go   18 lines
errors.go       146 lines
identity.go     195 lines
snapshot.go      73 lines
```

### F2: `identity_test.go` had pre-existing subtest-convention drift

- Smell: Test structure drift, inline test cases
- Before:
  - table rows in two test functions used non-`Should ...` case names;
  - `TestErrorOutputConventionsRenderStableJSONAndJSONL`, `TestResolveDefaultsAgentSessionOrigin`, and `TestSessionSnapshotFromInfo` asserted directly without `t.Run("Should ...")` subtests.
- Change:
  - renamed table rows to `Should ...`;
  - split inline assertions into independent subtests;
  - kept `t.Parallel()` on independent tests/subtests.
- Result: `identity_test.go` now passes the AGH test-convention checker.
- Behavior proof: assertions and expected values are unchanged; only test structure and names changed.

### F3: repeated identity-error triples remain below the necessary-change threshold

- Smell: Duplicated Code, Long Parameter List, mild Primitive Obsession
- Evidence: `identityError(err, code, message, action)` still receives repeated code/message/action triples for identity-required, lookup-unavailable, stale, mismatch, and unauthorized failures.
- Decision: left unchanged. The current helper is small, tested, and readable. Adding private constructors would be polish, not a necessary refactor, and would add another abstraction layer in a small package.

### F4: performance candidates did not meet the optimization threshold

Performance exploration found no production CPU hotspot in `internal/agentidentity`. Candidate optimizations were explicitly rejected because they either scored below `2.0` or would weaken identity correctness:

| Candidate | Impact | Confidence | Effort | Score | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| Remove duplicate trimming in `Resolve` | 1 | 2 | 2 | 1.0 | Do not change |
| Cache or skip session lookup | 1 | 1 | 4 | 0.25 | Do not change; stale-identity risk |
| Avoid `CloneSessionLineage` in `SessionSnapshotFromInfo` | 1 | 1 | 3 | 0.33 | Do not change; aliasing risk |
| Hand-roll JSON error output | 1 | 2 | 3 | 0.67 | Do not change; error path and contract risk |

## Performance Evidence

No benchmarks exist in this package:

```text
PASS
ok   github.com/pedronauck/agh/internal/agentidentity   0.033s
```

CPU profile after the refactor, using 200 package-test repetitions:

```text
Total samples = 190ms
170ms / 89.47% flat: syscall.rawsyscalln
20ms  / 10.53% flat: runtime.kevent
agentidentity production funcs: 0 flat CPU samples
```

Allocation profile after the refactor shows test harness cost dominating. Package allocations are low and mostly error-path or validation allocations:

```text
Resolve                    144000B cumulative across 200 test repetitions
identityError              140800B flat
lookupSessionSnapshot       64000B cumulative
validateResolveInputs       51200B cumulative
MarshalErrorJSONL           80032B cumulative
MarshalErrorJSON            77440B cumulative
```

Isomorphism notes:

- Ordering preserved: yes; `Resolve` still normalizes credentials, validates inputs, performs exactly one lookup, validates workspace, and derives actor context in the same order.
- Tie-breaking unchanged: yes; error precedence is unchanged.
- Floating-point: N/A.
- RNG seeds: N/A.
- JSON contract preserved: yes; error payload field names and exit-code mapping are unchanged.
- Security semantics preserved: yes; stale/missing/mismatch/unavailable identity classifications are unchanged, and identity is still validated per operation.

## Validation

All commands below passed after the fixes:

```bash
rtk make verify
rtk go test ./internal/agentidentity -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/agentidentity -count=1
rtk golangci-lint run ./internal/agentidentity
rtk proxy go test ./internal/agentidentity -cover -count=1
rtk go test ./internal/agentidentity ./internal/api/core ./internal/api/udsapi ./internal/cli -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/agentidentity/identity_test.go
rtk proxy go test -run '^$' -bench . -benchmem ./internal/agentidentity -count=1
rtk go test ./internal/agentidentity -run . -count=200 -cpu=1 -outputdir /tmp -cpuprofile /tmp/agentidentity-current.cpu -memprofile /tmp/agentidentity-current.mem -memprofilerate=1
rtk proxy go tool pprof -top -nodefraction=0 /tmp/agentidentity-current.cpu
rtk proxy go tool pprof -top -nodefraction=0 /tmp/agentidentity-current.mem
```

Coverage:

```text
coverage: 94.1% of statements
```

Note: `rtk make verify` emitted the existing non-blocking Vite chunk-size warning and macOS linker `-bind_at_load` warning, then exited successfully.

## Next Package

The next deterministic `go list ./internal/...` package after `internal/agentidentity` is:

```text
github.com/pedronauck/agh/internal/api/contract
```
