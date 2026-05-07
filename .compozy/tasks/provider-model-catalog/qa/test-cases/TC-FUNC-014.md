# TC-FUNC-014: Refresh Deadline Detached From Request Context

**Priority:** P1
**Type:** Functional
**Module:** `internal/modelcatalog.Service.Refresh`
**Requirement:** TechSpec SI-11.
**Status:** Not Run

## Objective

Verify refresh work uses `context.WithoutCancel(ctx)` plus an explicit `context.WithDeadline`, so HTTP/UDS request cancellation does not abort refresh prematurely and refresh deadlines do not leak from the request context.

## Preconditions

- [ ] Refresh stub configured to take longer than the request timeout.
- [ ] Test harness with deterministic clock or sleep-based assertion.

## Test Steps

1. **Cancel the HTTP request mid-refresh.**
   - Trigger `POST /api/providers/codex/models/refresh` with a 100ms client timeout while the source takes 2s.
   - **Expected:** Client receives canceled response; daemon completes refresh through the configured deadline; `model_catalog_sources` records the refresh outcome.
2. **Configured deadline applied.**
   - Configure a refresh deadline (default 60s in TechSpec; configurable per source).
   - **Expected:** Refresh completes within the configured deadline regardless of the request lifetime.
3. **Daemon shutdown joins outstanding refresh workers.**
   - Initiate refresh; gracefully shut daemon.
   - **Expected:** Daemon waits for refresh worker to finish (or hits configured shutdown timeout) before exit; no orphan goroutine; SQLite rows consistent.
4. **Repeated cancellation under storm.**
   - Cancel 100 sequential refresh calls within 50ms.
   - **Expected:** Coalescing prevents storm; one underlying refresh completes; status reflects single outcome.

## Audit Coverage

- C6 task tree (Task 05, Task 11).
- SI-11.

## Pass Criteria

- Refresh outcome recorded after request cancellation.
- Deadlines respected.
- Daemon shutdown clean.

## Failure Criteria

- Refresh aborts when request cancels.
- Deadlines inherited implicitly from request context.
- Goroutine leak after shutdown.
