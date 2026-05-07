# TC-PERF-002: Detached Refresh Lifetime + Daemon Shutdown Join

**Priority:** P0
**Type:** Performance
**Module:** `internal/modelcatalog` refresh wrapper.
**Requirement:** TechSpec SI-11.
**Status:** Not Run

## Objective

Verify request-cancellation does not abort detached refresh work, the configured deadline binds refresh, daemon shutdown joins outstanding refresh workers, and no goroutine leaks remain.

## Preconditions

- [ ] Stub source with controllable latency.
- [ ] Goroutine leak detector or `runtime.NumGoroutine` snapshot harness.

## Test Steps

1. **Cancel mid-flight HTTP refresh.**
   - Configure stub latency = 2s; client timeout = 100ms.
   - **Expected:** Client gets canceled error; daemon completes refresh; SQLite reflects success; goroutine count returns to baseline.
2. **Override request context deadline.**
   - Submit refresh under a context with 50ms deadline.
   - **Expected:** Refresh ignores caller deadline; uses configured deadline.
3. **Daemon shutdown.**
   - Trigger refresh; immediately call daemon shutdown.
   - **Expected:** Shutdown waits for refresh to complete (or hits configured shutdown timeout); SQLite consistent; no orphan goroutine; `Close` on store happens after refresh worker join.
4. **Goroutine leak check.**
   - After 100 cancellation cycles, snapshot `runtime.NumGoroutine`.
   - **Expected:** No monotonic growth.

## Audit Coverage

- C11.
- SI-11, SI-12.

## Pass Criteria

- Refresh completes under cancellation.
- Daemon shuts down cleanly.
- No goroutine leak.

## Failure Criteria

- Refresh aborts when request cancels.
- Goroutine count grows.
- Daemon exits before refresh completes.
