---
status: resolved
file: internal/daemon/harness_reentry_bridge.go
line: 940
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_S2k,comment:PRRC_kwDOR5y4QM65JSEm
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`operationContext()` can make shutdown hang under slow downstream I/O.**

Using `context.WithoutCancel(b.ctx)` for worker-side store/session calls means cancellation won’t stop those calls, while `shutdown()` waits for all workers. If any call blocks, daemon shutdown can stall indefinitely.



As per coding guidelines `**/*.go`: "Every goroutine must have explicit ownership and shutdown via context.Context cancellation" and "No fire-and-forget goroutines — track with sync.WaitGroup or equivalent".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_reentry_bridge.go` around lines 935 - 940,
operationContext() currently strips cancellation (uses context.WithoutCancel)
which allows downstream I/O to ignore shutdown, so change operationContext() in
harnessReentryBridge to return the bridge's real cancellable context (i.e.,
return b.ctx when non-nil, otherwise context.Background()) so operations respect
cancellation from shutdown(); also audit any goroutines started by
harnessReentryBridge methods to accept that context and ensure they are tracked
and awaited by the daemon shutdown (e.g., use the existing shutdown wait group
or add one) rather than being fire-and-forget.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `operationContext()` currently returns `context.WithoutCancel(b.ctx)`, and the bridge uses that helper for store and session lookups inside worker goroutines (`recoverPendingRuns`, `processTerminalRun`, `resolveWakeTargetSnapshot`, `syntheticEventExists`, and summary writes). Because `shutdown()` cancels `b.ctx` and then waits for the worker wait group, any blocked store/session call can ignore shutdown and keep `shutdown()` waiting indefinitely.
- Validation: the bridge already tracks all spawned goroutines through `startWorker()` and the shared wait group, so the lifecycle bookkeeping is in place. The gap is that some tracked workers use a non-cancelable derived context for downstream I/O.
- Fix approach: make `operationContext()` return the bridge's real cancellable context, then keep shutdown-time finalization best-effort by using a bounded fallback context only when the bridge is already canceled. Add regression coverage that blocks a session lookup until its context is canceled and asserts bridge shutdown returns promptly without leaving a fire-and-forget worker behind.
- Implemented: `operationContext()` now returns `b.ctx` directly, `finalizeRunOutcome()` uses a bounded one-second best-effort context only after shutdown cancellation, and event-summary writes reuse the caller-provided context so normal bridge I/O remains cancellation-aware.
- Regression coverage: added `TestDetachedHarnessDaemonScenarios/ShouldCancelBlockedStatusLookupOnBridgeShutdown` in `internal/daemon/daemon_test.go` to block `sessions.Status` until its context is canceled and verify `shutdown()` returns promptly. The pre-existing `ShouldFinalizeHungSyntheticWakeOnBridgeShutdown` scenario still passes, proving the shutdown finalization behavior was preserved.
- Verification:
  - `go test ./internal/daemon -run 'TestDetachedHarnessDaemonScenarios/(ShouldScheduleRescanWhenHarnessReentryQueueIsFull|ShouldRecoverEqualTimestampRunsByTerminalSequence|ShouldCancelBlockedStatusLookupOnBridgeShutdown|ShouldFinalizeHungSyntheticWakeOnBridgeShutdown)' -count=1` → `ok  	github.com/pedronauck/agh/internal/daemon	0.072s`
  - `make verify` → exit code `0`; web checks passed (`167` test files, `1173` tests), Go lint reported `0 issues`, Go test suite completed with `DONE 5342 tests in 12.392s`, and package boundary checks ended with `OK: all package boundaries respected`.
