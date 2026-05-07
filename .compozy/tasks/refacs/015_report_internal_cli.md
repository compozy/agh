# Iteration 015 Report: `internal/cli`

## Scope

- Package: `github.com/pedronauck/agh/internal/cli`
- Deterministic order index: 15 from `rtk go list ./internal/...`
- Next package: `github.com/pedronauck/agh/internal/cli/docpost`
- Analysis modes: `$refactoring-analysis`, `$extreme-software-optimization`, `$systematic-debugging`, `$no-workarounds`
- Subagents: read-only refactoring explorer and read-only performance explorer

## Baseline

Commands run before implementation:

```bash
rtk go test ./internal/cli -count=1
rtk golangci-lint run ./internal/cli
rtk proxy go test ./internal/cli -cover -count=1
rtk go test -tags integration ./internal/cli -count=1
rtk proxy go test ./internal/cli -run '^$' -bench 'Benchmark(RenderHumanTableLarge|RenderToonArrayLarge|DecodeSSELargeStream|DoRequestPostJSON)$' -benchmem -count=1
```

Observed baseline:

- Package tests: passed (`684 passed in 1 packages`).
- Package lint: no issues.
- Package coverage: `74.9% of statements`.
- Integration-tag package tests: failed with `734 passed, 11 failed`.
- Integration failures split into one CLI test assertion bug in `TestCLITaskRunLifecycleIntegration` plus historical channel/presence failures after daemon restart.
- Baseline benchmark highlights:
  - `BenchmarkRenderHumanTableLarge`: about `136972 ns/op`, `309183 B/op`, `1593 allocs/op`.
  - `BenchmarkRenderToonArrayLarge`: about `58355 ns/op`, `84752 B/op`, `19 allocs/op`.
  - `BenchmarkDecodeSSELargeStream`: about `113780-152744 ns/op`, `246216-246218 B/op`, `4099 allocs/op`.
  - `BenchmarkDoRequestPostJSON`: about `1360 ns/op`, `2642 B/op`, `29 allocs/op`.

## Findings

### P1: CLI duplicated the shared SSE decoder

`internal/cli/client.go` carried a package-local SSE parser even though `internal/sse` already provides the shared decoder and `internal/cli` already aliases `sse.Event`, `sse.Handler`, and `sse.ErrStop`.

Implemented:

- Removed the local CLI SSE parser, line decoder, and typed-nil reader reflection helper.
- Made `decodeSSE` delegate directly to `sse.Decode`.
- Removed now-unused `bufio` and `reflect` imports.

Behavior proof:

- Focused SSE/client tests passed.
- `BenchmarkDecodeSSELargeStream` stayed in the same range because the active implementation is now the shared `internal/sse` decoder. The real streaming allocation work belongs in a later `internal/sse` iteration.

### P1: human table rendering allocated one joined string per row

`renderHumanTable` used `strings.Join(row, "\t")` for every header, separator, and data row before feeding the result into `tabwriter`. Allocation profiling identified row joining as the largest package-local CLI rendering hotspot.

Implemented:

- Replaced the table renderer with a direct `strings.Builder`-based column renderer.
- Preserved title, underline, separator, empty-table, and two-space padded column shape.
- Used rune counts for column widths to stay closer to `tabwriter` behavior for non-ASCII cell content.
- Kept `renderHumanSection` on `tabwriter`, but stopped discarding its write/flush errors internally.

Final benchmark:

```bash
rtk proxy go test ./internal/cli -run '^$' -bench 'Benchmark(RenderHumanTableLarge|RenderToonArrayLarge|DecodeSSELargeStream|DoRequestPostJSON)$' -benchmem -count=5
```

Observed after results:

- `BenchmarkRenderHumanTableLarge`: about `47.8-49.3 us/op`, `127112-127113 B/op`, `22 allocs/op`.
- `BenchmarkRenderToonArrayLarge`: about `50.8-51.8 us/op`, `84752 B/op`, `19 allocs/op`.
- `BenchmarkDecodeSSELargeStream`: about `112.8-114.9 us/op`, `246408 B/op`, `4099 allocs/op`.
- `BenchmarkDoRequestPostJSON`: about `1191-1208 ns/op`, `2641 B/op`, `29 allocs/op`.

The table renderer is now roughly 2.7x faster than the baseline and removes almost all row-rendering allocation churn for the large-table benchmark.

### P1: integration test compared JSON whitespace instead of JSON semantics

`TestCLITaskRunLifecycleIntegration` compared `json.RawMessage` metadata as a minified string. CLI JSON output intentionally pretty-prints with `encoder.SetIndent("", "  ")`, so the test failed even though the metadata was semantically preserved.

Implemented:

- Replaced byte-level metadata checks with `assertDetachedHarnessMetadata`, which unmarshals and validates the expected `schema` field semantically.
- Verified the focused integration test now passes.

### P1: daemon readiness polling owned a blocking wait goroutine

`waitForDaemonStart` started a goroutine solely to call `child.Wait()`. On timeout, the function returned without joining or canceling that goroutine.

Implemented:

- Removed the readiness-local `Wait()` goroutine.
- Kept readiness polling focused on daemon status plus child process liveness through `deps.processAlive` / `procutil.Alive`.
- Added `TestWaitForDaemonStartRefacs` to prove timeout does not call `process.Wait()`.

Behavior note:

- Direct `daemonProcess.Wait()` still exists for explicit process waiting and still carries detached process log attachment where the concrete process provides it.
- Readiness polling now avoids unowned blocking goroutines and reports a generic early-exit error if the child PID is no longer alive before readiness.

### P2: task and network JSON flag parsing were duplicated

Task command JSON flags and network send body parsing both implemented trim, required-value, JSON validation, and raw-message preservation locally.

Implemented:

- Added `parseRequiredJSONRawMessage` as a narrow package-local helper.
- Kept task-specific and network-specific wrappers responsible for their existing user-facing error context and domain validation.
- Added direct tests for valid, empty, and invalid JSON flag parsing.

### P2: session stream tracked an SSE resume cursor but never used it

`streamSessionEvents` updated a local `lastEventID` variable while still passing an empty cursor to `StreamSessionEvents`. That made the state dead code and suggested resume behavior that does not exist.

Implemented:

- Removed the unused cursor state.
- Left explicit resume behavior deferred until the command owns a reconnect loop and tests.

## Integration Failure Classification

After fixing the CLI-owned metadata assertion, the full integration-tag command was rerun:

```bash
rtk go test -tags integration ./internal/cli -count=1
```

Observed result after this iteration:

- `741 passed, 10 failed in 1 packages`.

The remaining failures are still historical channel/presence assertions after daemon restart:

- `TestCLIHistoricalChannelMixedOwnershipAfterDaemonRestartIntegration/Should_keep_the_channel_historical_before_restart`
- `TestCLIHistoricalChannelTaskRunStartAfterDaemonRestartIntegration/Should_keep_the_run-start_channel_historical_before_restart`
- `TestCLIHistoricalChannelTaskNextAfterDaemonRestartIntegration/Should_keep_the_channel_historical_before_restart`
- `TestCLIHistoricalChannelTaskNextAfterDaemonRestartIntegration/Should_persist_the_completed_historical_run_and_leave_no_active_sessions`
- `TestCLIHistoricalChannelTaskRunTerminalAfterDaemonRestartIntegration/Should_cancel_a_historical_task_run_after_daemon_restart`
- `TestCLIHistoricalChannelTaskRunTerminalAfterDaemonRestartIntegration/Should_fail_a_historical_task_run_after_daemon_restart`

Classification:

- The fixed metadata failure was actionable in `internal/cli` test code.
- The remaining historical channel/presence failures are downstream of CLI request shaping. The tests successfully create sessions/tasks/runs with channel-bound values before checking `agh network channels -o json`; the missing channel/presence data originates in daemon/network/session/task historical read-model reconstruction.
- This iteration did not patch daemon/network state because that would cross the one-package loop boundary.

## Deferred

- Optimize `internal/sse.Decode`: allocation profiling still shows `scanner.Text()`, per-stream scanner buffers, and event-data copies as real streaming costs, but the active implementation is owned by `internal/sse`, not `internal/cli`.
- Split the broad `DaemonClient` interface into command-family consumer interfaces. This is a broad package-wide testability refactor and should be a dedicated structural pass.
- Extract request builders and render field specs from long command functions. This is maintainability work, but it is not required for this iteration's correctness/performance fixes.
- Unify repeated human/TOON task-run field lists behind snapshot-backed output contracts. Deferred to avoid changing agent-facing output without broader CLI surface tests.
- Raise `internal/cli` direct coverage above the 80% package target. Coverage remains below target because the package is very large; this iteration added focused tests for changed behavior but did not attempt a broad coverage campaign.

## Behavior Proof

- CLI JSON output shape: preserved. The renderer still pretty-prints JSON; tests now compare raw metadata semantically where whitespace is incidental.
- Human table shape: preserved for standard CLI cells, with two-space padded aligned columns and the same title/underline/separator/empty output structure.
- SSE behavior: preserved through the shared `internal/sse` decoder.
- Session stream resume behavior: unchanged. The command still passes no `Last-Event-ID`; dead local state was removed.
- Daemon start readiness: preserves status polling and child liveness detection while removing the unowned blocking wait goroutine.
- JSON flag parsing: preserved raw trimmed JSON payloads while centralizing required-value and JSON syntax validation.

## Files Changed

- `internal/cli/client.go`
- `internal/cli/format.go`
- `internal/cli/json_flags.go`
- `internal/cli/json_flags_test.go`
- `internal/cli/session.go`
- `internal/cli/task.go`
- `internal/cli/network.go`
- `internal/cli/daemon.go`
- `internal/cli/daemon_wait_refac_test.go`
- `internal/cli/cli_integration_test.go`

## Validation

Final validation commands:

```bash
rtk go test ./internal/cli -run 'TestParseRequiredJSONRawMessage|TestWaitForDaemonStart|TestRunDaemonDetachedReturnsReadyStatus|Test.*Render|Test.*Output|Test.*Human|Test.*Toon' -count=1
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/json_flags_test.go
rtk python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/cli/daemon_wait_refac_test.go
rtk go test ./internal/cli -count=1
rtk golangci-lint run ./internal/cli
rtk proxy go test ./internal/cli -cover -count=1
rtk env CGO_ENABLED=1 go test -race ./internal/cli -count=1
rtk go test ./cmd/agh ./internal/cli -count=1
rtk go test -tags integration ./internal/cli -run '^TestCLITaskRunLifecycleIntegration$' -count=1
rtk go test -tags integration ./internal/cli -count=1
rtk proxy go test ./internal/cli -run '^$' -bench 'Benchmark(RenderHumanTableLarge|RenderToonArrayLarge|DecodeSSELargeStream|DoRequestPostJSON)$' -benchmem -count=5
rtk make verify
```

Observed results:

- Focused unit/output/refac tests: `136 passed in 1 packages`.
- New test-shape checks: passed for `json_flags_test.go` and `daemon_wait_refac_test.go`.
- Package tests: `690 passed in 1 packages`.
- Package lint: no issues.
- Package coverage: `74.8% of statements`.
- Race package tests: passed.
- `cmd/agh` plus `internal/cli`: `693 passed in 2 packages`.
- Focused lifecycle integration test: passed.
- Full integration-tag `internal/cli`: failed with the 10 downstream historical channel/presence failures classified above.
- Final benchmarks listed in the performance section.
- `make verify`: passed.
