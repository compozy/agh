# TC-PERF-001: Refresh Concurrency - Per-Provider Serialization + Cross-Provider Parallelism

**Priority:** P0
**Type:** Performance
**Module:** `internal/modelcatalog.Service.Refresh`, refresh wrapper.
**Requirement:** TechSpec SI-12, Task 11.
**Status:** Not Run

## Objective

Verify per-provider refresh requests serialize before any subprocess or provider-home work, identical concurrent requests for one provider coalesce, refresh storms across providers proceed in parallel, and SQLite write contention is avoided (no `BUSY` errors).

## Preconditions

- [ ] Stub live sources for `codex`, `anthropic`, `gemini`, `openrouter`, `ollama` with measurable subprocess latency.
- [ ] Test harness counts subprocess invocations and SQLite write attempts.

## Test Steps

1. **N concurrent same-provider refreshes.**
   - Issue 32 simultaneous `POST /api/providers/codex/models/refresh` requests.
   - **Expected:** Exactly one subprocess invocation; all 32 callers receive the same status batch with the same `refresh_request_id`.
2. **N cross-provider refreshes.**
   - Issue refreshes for all 5 providers concurrently.
   - **Expected:** 5 underlying subprocess invocations run in parallel; total wall time approximates the slowest provider, not the sum.
3. **Mixed storm.**
   - Issue 32 same-provider + 32 cross-provider concurrently.
   - **Expected:** Same-provider coalesced; cross-provider parallel; no SQLite `BUSY` error escapes coalescing.
4. **Repeated coalescing returns identical statuses.**
   - **Expected:** Two callers in the same coalesce window see byte-equal status payloads; refresh request id correlated.
5. **SQLite contention.**
   - Drive 100 refreshes/second across 5 providers for 30s.
   - **Expected:** Zero `SQLITE_BUSY` propagated; per-provider serialization holds; no row corruption.

## Audit Coverage

- C5, C6 (Task 11), C11 disruption probe.
- SI-12, SI-13.

## Pass Criteria

- Same-provider coalescing observed.
- Cross-provider parallelism observed.
- No `SQLITE_BUSY` escapes.

## Failure Criteria

- Multiple subprocess invocations per coalesced batch.
- Cross-provider serialized.
- SQLite errors observed.
