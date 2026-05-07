# Refactoring and Performance Report: `internal/acp`

> Date: 2026-05-06
> Scope: `github.com/pedronauck/agh/internal/acp`
> Iteration: 001
> Language/Stack: Go, ACP JSON-RPC over stdio, subprocess and terminal supervision

## Executive Summary

`internal/acp` is the runtime's concrete ACP client: it launches provider processes, negotiates sessions, sends prompts, translates JSON-RPC updates, handles file/terminal/permission callbacks, and classifies ACP failures. The package was healthy enough to refactor in place, but `handlers.go` had grown into a divergent module mixing inbound dispatch, terminal lifecycle, session update translation, and utility code.

This iteration fixed the package-local issues that were both actionable and safe: terminal lifecycle code was extracted into its own file, inbound handler dispatch no longer allocates a map per request, no-gateway tool interception skips redundant JSON roundtrips, environment assembly avoids `fmt.Sprintf`, and stale integration/cross-compile test issues were repaired.

## Findings

| ID | Priority | Type | Location | Status |
| --- | --- | --- | --- | --- |
| F1 | P1 | Large Module / Divergent Change | `internal/acp/handlers.go` | Fixed |
| F2 | P2 | Avoidable allocation in hot dispatch path | `internal/acp/handlers.go` | Fixed |
| F3 | P2 | Redundant no-gateway JSON roundtrip | `internal/acp/handlers.go` | Fixed |
| F4 | P3 | Avoidable formatting allocation | `internal/acp/terminal.go` | Fixed |
| F5 | P2 | Integration test setup contradicted permission mode semantics | `internal/acp/client_integration_test.go` | Fixed |
| F6 | P2 | Windows test compile broke because a shared test constant was Unix-only | `internal/acp/client_test.go`, `internal/acp/process_tree_test.go` | Fixed |

### F1: `handlers.go` mixed terminal lifecycle with JSON-RPC handling

- Smell: Large Module, Divergent Change
- Before: `handlers.go` was 1,471 lines and owned inbound dispatch, terminal process lifecycle, terminal output buffering, env merging, session update translation, permission event emission, and misc helpers.
- Change: Extracted terminal ownership, process registration, output windowing, env merge, process-tree cleanup, and terminal helper code into `internal/acp/terminal.go`.
- Result: `handlers.go` now focuses on ACP wire handling and translation; terminal-specific behavior has a cohesive home. Current file sizes: `handlers.go` 625 lines, `terminal.go` 870 lines.
- Behavior proof: Pure move plus import cleanup. Existing package tests, integration tests, race tests, and terminal/process-tree tests pass.

### F2: Inbound request dispatch allocated a map and closures per call

- Smell: Performance hotspot, avoidable allocation, lazy dispatch structure
- Location: `AgentProcess.handleInbound`
- Before: every non-session-update inbound JSON-RPC request built a `map[string]func(...)` with closures.
- Change: Replaced the per-call map with a direct `switch method`.
- Isomorphism:
  - Ordering preserved: yes; one request dispatches to exactly one handler as before.
  - Tie-breaking unchanged: yes; method names are unique.
  - Floating-point/RNG: N/A.
  - Unknown method behavior: preserved via `acpsdk.NewMethodNotFound(method)`.

### F3: No-gateway tool interception serialized and deserialized unchanged requests

- Smell: Redundant work on hot callback paths
- Location: `interceptReadTextFileRequest`, `interceptWriteTextFileRequest`, `interceptCreateTerminalRequest`
- Before: even when `ToolExecutionGateway` was nil, read/write/terminal callbacks marshaled typed input to JSON, cloned it, then unmarshaled it back before returning the same logical request.
- Change: Added no-gateway fast paths that preserve the previous normalization:
  - read/write paths still trim `Path`;
  - terminal path still trims `Command`;
  - empty terminal `Args` and `Env` still normalize to nil, matching the prior `omitempty` JSON roundtrip;
  - pointer fields are cloned so aliasing remains equivalent to JSON unmarshal behavior;
  - gateway-enabled paths still use the previous raw JSON patch contract.
- Isomorphism:
  - Ordering preserved: yes; no side effects are reordered.
  - Tie-breaking unchanged: yes.
  - Floating-point/RNG: N/A.
  - Gateway behavior: unchanged when `p.toolGateway != nil`.

### F4: `mergeCommandEnv` used `fmt.Sprintf` for simple `KEY=VALUE` assembly

- Smell: Avoidable formatting allocation
- Location: `mergeCommandEnv`
- Change: Replaced `fmt.Sprintf("%s=%s", name, value)` with direct `name + "=" + value`.
- Isomorphism:
  - Order preserved: yes; `order` slice is unchanged.
  - Last-value-wins semantics preserved: yes; `merged[name]` is unchanged.
  - Empty override values preserved: yes.

### F5: Permission integration tests used `deny_all` while expecting interactive approval

- Root cause: `PermissionModeDenyAll` correctly returns `reject-once, false` from `permissionPolicy.permissionDecision`; it is intentionally non-interactive. The integration tests expected a pending event and then approval/timeout, which is the behavior of `approve_reads` for edit/write requests.
- Change: Updated the two interactive permission integration scenarios to use `PermissionModeApproveReads`, preserving coverage for pending, approve, timeout, and final decision events.
- Additional cleanup: Wrapped integration tests in `t.Run("Should ...")` subtests and documented the `t.Setenv` serial requirement.

### F6: Windows compile failed because a test helper constant was Unix-only

- Root cause: `client_test.go` uses `testWrapperPIDFileEnvKey`, but the constant lived in `process_tree_test.go`, which is guarded by `//go:build !windows`.
- Change: Moved the constant to `process_tree_constants_test.go` without a platform build tag and updated `process_tree_test.go` to follow subtest conventions.

## Performance Evidence

Benchmarks run on Darwin arm64, Apple M4 Max.

### Baseline Before Changes

```text
BenchmarkHandleInboundReadTextFile-16        1021 ns/op   824 B/op   26 allocs/op
BenchmarkMergeCommandEnvWithOverrides-16     1040 ns/op  1433 B/op   31 allocs/op
```

### After Dispatch Switch and Env Assembly Change

```text
BenchmarkHandleInboundReadTextFile-16         840.7 ns/op   696 B/op   18 allocs/op
BenchmarkMergeCommandEnvWithOverrides-16      662.3 ns/op  1224 B/op   18 allocs/op
```

### Final After No-Gateway Fast Path and Terminal Extraction

```text
BenchmarkHandleSessionUpdateAgentMessage-16                 5882 ns/op  3544 B/op  70 allocs/op
BenchmarkHandleInboundReadTextFile-16                        457.5 ns/op 344 B/op   8 allocs/op
BenchmarkManagedTerminalAppendOutputOverflow-16              573.1 ns/op 0 B/op     0 allocs/op
BenchmarkPermissionPolicyResolvePathExistingRelative-16      9955 ns/op  4808 B/op  43 allocs/op
BenchmarkMergeCommandEnvWithOverrides-16                     644.1 ns/op 1224 B/op  18 allocs/op
```

### Opportunity Matrix

| Opportunity | Impact | Confidence | Effort | Score | Implemented |
| --- | ---: | ---: | ---: | ---: | --- |
| No-gateway tool interception fast path | 2 | 5 | 2 | 5.0 | Yes |
| Switch-based inbound dispatch | 2 | 5 | 1 | 10.0 | Yes |
| `mergeCommandEnv` direct string assembly | 1 | 4 | 1 | 4.0 | Yes |
| Optimize session update JSON decoding | 2 | 3 | 4 | 1.5 | No |
| Cache or alter permission path resolution | 2 | 3 | 5 | 1.2 | No |
| Stream line slicing instead of splitting full content | 2 | 2 | 3 | 1.3 | No |

## Not Implemented

- `handleSessionUpdate` JSON decoding remains unchanged. Benchmarks show allocations, but real session traffic profiling is needed before changing ACP SDK union decoding or raw event preservation.
- Permission path resolution remains unchanged. The hotspot is dominated by symlink-aware filesystem checks, and security containment is more important than speculative caching.
- `sliceLines` remains unchanged. A large-file benchmark should be added before replacing `strings.Split` behavior because trailing newline and line-index semantics are easy to change accidentally.
- Larger prompt lifecycle refactors (`runPrompt`) remain for a later `internal/acp` pass or a dedicated package follow-up. This iteration already changed terminal structure, dispatch, tool interception, and integration test setup.

## Validation

All commands below passed after the fixes:

```bash
rtk make verify
rtk go test ./internal/acp -count=1
rtk go test -tags integration ./internal/acp -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/acp -count=1
rtk golangci-lint run ./internal/acp
rtk proxy go test -run '^$' -bench . -benchmem ./internal/acp -count=1
rtk proxy env GOOS=windows GOARCH=amd64 go test -c -o /tmp/acp-windows.test.exe ./internal/acp
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/acp/acp_bench_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/acp/client_integration_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/acp/process_tree_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/acp/process_tree_constants_test.go
```

Note: the first full `rtk make verify` run failed in unrelated `internal/automation`, `internal/cli`, and `internal/heartbeat` race tests with SQLite/context-deadline timeouts while another local web test process was active. The failing packages passed when reproduced in isolation with `rtk env CGO_ENABLED=1 go test -race -parallel=4 ./internal/automation ./internal/cli ./internal/heartbeat -count=1`, and the second full `rtk make verify` run passed.

## Next Package

The next deterministic `go list ./internal/...` package after `internal/acp` is:

```text
github.com/pedronauck/agh/internal/agentidentity
```
