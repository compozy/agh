# Refacs Iteration 004: `internal/api/core`

Date: 2026-05-06

Package: `github.com/pedronauck/agh/internal/api/core`

## Scope

This iteration audited the central API core package: HTTP/UDS payload conversion helpers, network projections, prompt streaming, observe SSE helpers, and the package tests that exercise those surfaces.

Read-only subagents were used for:

- Refactoring analysis of large modules, compatibility aliases, duplicated terminology, helper cohesion, and test-shape risks.
- Performance analysis with package tests, coverage, benchmarks, CPU/memory profiling, and targeted hot-path scoring.

Because `internal/api/core` is a large cross-surface package, edits were intentionally scoped to refactors with clear evidence and low contract risk.

## Baseline

- `rtk go test ./internal/api/core -count=1` passed before edits.
- `rtk go test -tags integration ./internal/api/core -count=1` passed before edits.
- `rtk proxy go test ./internal/api/core -cover -count=1` showed `67.0%` statement coverage before edits.
- Existing benchmarks covered SSE writing, observe-event emission, session payload mapping, and agent-event payload mapping.
- CPU/memory profiling across repeated package tests was dominated by test harness and Gin/syscall overhead. No broad production rewrite was justified from the package profile alone.

## Findings

### Fixed: stale `NetworkChannelMessagePayload` compatibility alias

Iteration 003 carried forward a contract smell: `NetworkChannelMessagePayload` was only an alias for `NetworkConversationMessagePayload`.

That alias weakened the current naming model because the same payload now represents public-thread and direct conversation timelines, not only channel-local messages.

Fix:

- Removed the alias from `internal/api/contract/contract.go`.
- Renamed core conversion helpers:
  - `NetworkChannelMessagePayloadFromEntry` -> `NetworkConversationMessagePayloadFromEntry`
  - `NetworkChannelMessagePayloadFromView` -> `NetworkConversationMessagePayloadFromView`
- Updated core network timeline return types to `[]contract.NetworkConversationMessagePayload`.
- Updated direct dependent tests and harness helpers in:
  - `internal/api/core/network_test.go`
  - `internal/api/core/coverage_helpers_test.go`
  - `internal/daemon/daemon_network_collaboration_integration_test.go`
  - `internal/testutil/e2e/runtime_harness.go`
  - `internal/testutil/e2e/runtime_harness_helpers_test.go`

Contract impact:

- JSON shape is unchanged.
- OpenAPI shape is unchanged.
- Generated code had no drift.
- The old Go alias name is deleted as a greenfield hard cut rather than retained as a compatibility bridge.

### Fixed: avoid formatted-allocation churn in `ObserveEventID`

`ObserveEventID` built cursor IDs through formatted strings:

- positive sequence: timestamp + `fmt.Sprintf("%020d", sequence)`
- fallback sequence: timestamp + string concatenation

This function runs for every observe SSE event emitted by `EmitObserveEvents`.

Fix:

- Build the ID into a byte buffer with `time.AppendFormat`.
- Append the sequence with a small zero-padding helper plus `strconv.AppendInt`.
- Preserve the exact external cursor format:
  - `RFC3339Nano|00000000000000000042` for positive sequence values.
  - `RFC3339Nano|<event-id>` when sequence is absent.

### Fixed: prompt-stream SSE envelopes now use typed payload structs

`PromptStreamEncoder` emitted many SSE frames as fresh `map[string]any` literals. That made the stream harder to audit and added avoidable allocations on a stream path that can emit many frames per agent turn.

Fix:

- Added typed internal payload structs for prompt stream frames:
  - `promptStartPayload`
  - `promptBlockPayload`
  - `promptDeltaPayload`
  - `promptToolInputStartPayload`
  - `promptToolInputAvailablePayload`
  - `promptDataEventEnvelope`
  - `promptToolOutputAvailablePayload`
  - `promptErrorPayload`
- Replaced map envelopes in the prompt stream encoder with those structs.
- Preserved existing JSON field names and required field presence, including `toolCallId`, `toolName`, `input`, `output`, `errorText`, and `messageId`.
- Added `BenchmarkPromptStreamEncoderEmit` before the production change to make the optimization measurable.

### Fixed: touched tests now follow AGH test conventions

Several touched files had existing AGH test-shape violations. Because this iteration had to update those files for the payload hard cut, the touched regions were normalized instead of leaving newly edited files in a known-bad shape.

Fix:

- Wrapped inline tests in `t.Run("Should ...")` subtests where required.
- Added `t.Parallel()` for safe unit subtests.
- Kept integration/runtime-harness subtests sequential where shared runtime state makes parallelism unsafe.
- Replaced discarded benchmark results with package-level benchmark sinks.
- Replaced discarded `fmt.Fprint/Fprintf` errors in touched test handlers with explicit error handling.
- Cleaned mechanically spaced subtest names into readable sentences.

## Deferred Findings

### Deferred: broad file splits in `internal/api/core`

The package contains several very large files, including network, settings, tasks, conversions, and authored-context surfaces. Splitting them mechanically would improve navigation, but it would add large review noise without a concrete behavior or performance payoff in this iteration.

Decision:

- Do not split large files in this pass.
- Prefer future splits only when paired with a real API boundary cleanup or a behavior-preserving extraction that reduces active complexity.

### Deferred: BaseHandlers interface reshaping

The refactoring subagent found that some handler dependencies still look broad. Reshaping the handler composition boundary would touch many endpoint tests and mocks.

Decision:

- Do not redesign handler dependencies inside this package iteration.
- Treat it as architectural work requiring a dedicated package/surface pass if the same pain repeats.

### Deferred: prompt raw-event parsed/redacted cache

The performance subagent identified repeated raw-event parsing as a possible prompt-stream follow-up. Profiling did not prove it as a package hotspot, and the correctness constraints around redaction, raw payload shape, and tool-name fallback are high.

Decision:

- Do not cache parsed/redacted raw views in this iteration.
- Keep the new prompt benchmark as evidence for future targeted work.

## Performance Evidence

### `ObserveEventID` / observe SSE emission

Command:

```bash
rtk proxy go test -run '^$' -bench 'Benchmark(EmitObserveEvents|PromptStreamEncoderEmit)$' -benchmem ./internal/api/core -count=5
```

Before the `ObserveEventID` change, the clean benchmark sample was approximately:

| Benchmark | ns/op | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkEmitObserveEvents` | ~49,008 | ~41,542 | 384 |

After the change:

| Benchmark | ns/op range | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkEmitObserveEvents` | ~43,973-49,950 | ~38,968 | 256 |

Observed effect:

- Allocations dropped from `384` to `256` per benchmark operation.
- Allocated bytes dropped from about `41.5KB` to about `39.0KB`.
- Latency moved modestly lower in repeated samples, with normal local benchmark variance.

### Prompt stream encoder

The prompt stream benchmark was added before the typed-payload refactor.

Baseline:

| Benchmark | ns/op range | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkPromptStreamEncoderEmit` | ~34,227-35,213 | ~73,013-73,063 | 428 |

After typed payload structs:

| Benchmark | ns/op range | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkPromptStreamEncoderEmit` | ~29,013-30,477 | ~66,810-66,905 | 327 |

Observed effect:

- About `101` fewer allocations per benchmark operation.
- About `6.1KB` fewer allocated bytes per benchmark operation.
- About `13-15%` lower latency in the repeated benchmark sample.

## Validation

Targeted validation:

```bash
rtk go test ./internal/api/core -count=1
rtk go test ./internal/api/contract ./internal/api/core ./internal/testutil/e2e -count=1
rtk go test -tags integration ./internal/api/core -count=1
rtk go test -tags integration ./internal/daemon -run 'TestDaemonE2ENetwork(DirectReplyLifecycleWithMockAgents|WhoisAndCapabilityExchange)' -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/api/core -count=1
rtk golangci-lint run ./internal/api/core
rtk make codegen-check
rtk proxy go test ./internal/api/core -cover -count=1
rtk rg -n "NetworkChannelMessagePayload|NetworkChannelMessagePayloadFrom" internal web packages openapi cmd
rtk proxy go test -run '^$' -bench 'Benchmark(EmitObserveEvents|PromptStreamEncoderEmit)$' -benchmem ./internal/api/core -count=5
```

Test-convention validation:

```bash
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/network_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/coverage_helpers_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/api/core/perf_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/testutil/e2e/runtime_harness_helpers_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/daemon/daemon_network_collaboration_integration_test.go
```

Results:

- `rtk go test ./internal/api/core -count=1` passed: `713 passed in 1 packages`.
- Dependent package smoke tests passed: `945 passed in 3 packages`.
- `internal/api/core` integration tests passed: `734 passed in 1 packages`.
- Affected daemon integration tests passed: `4 passed in 1 packages`.
- Race test for `internal/api/core` passed.
- Package lint passed with no issues.
- `make codegen-check` passed with no generated drift.
- Coverage after the refactor: `67.0% of statements`.
- The code search for `NetworkChannelMessagePayload` returned no matches in code surfaces.
- All touched test files passed the AGH test-conventions checker.
- Full gate: `rtk make verify` passed after the final edits.

## Next Package

Next deterministic package:

```text
github.com/pedronauck/agh/internal/api/httpapi
```
