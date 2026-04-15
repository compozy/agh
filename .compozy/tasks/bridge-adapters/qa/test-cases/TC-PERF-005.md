## TC-PERF-005: Instance Cache Sync Without Delivery Blocking

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Verify the Instance Cache can perform sync/reset operations without blocking in-flight deliveries, without causing deadlocks or goroutine leaks, and that new deliveries immediately use updated cache state after sync completes.

### Preconditions
- [ ] Instance Cache is seeded with initial instance data at init time
- [ ] Delivery pipeline is operational and can process deliveries concurrently
- [ ] A mechanism to trigger cache sync/reset on demand is available (API call or direct method invocation)
- [ ] Mock instance data source can return different data sets for pre-sync and post-sync states
- [ ] Goroutine counting instrumentation is available (`runtime.NumGoroutine()`)
- [ ] Deadlock detection timeout is configured (e.g., test timeout of 30 seconds)

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| In-flight delivery completion during sync | 100% | 100% | | |
| In-flight delivery error rate during sync | 0% | 0% | | |
| Delivery latency during sync vs steady state | < 2x steady state | < 3x steady state | | |
| Time for sync/reset to complete | < 500ms | < 1s | | |
| Post-sync cache consistency (new data visible) | immediate | within 1 delivery | | |
| Goroutine delta after sync (leak check) | 0 | 0 | | |
| Deadlock occurrences | 0 | 0 | | |

### Load Scenarios

1. **Sync during active deliveries**
   - Duration: Start 20 deliveries in-flight (each takes ~50ms via mock handler delay), trigger cache sync at t=10ms
   - Expected: All 20 deliveries complete without error. Deliveries that started before sync use the pre-sync cache data (or post-sync, either is acceptable as long as data is consistent within a single delivery). Sync completes independently of delivery completion.

2. **Delivery consistency after sync**
   - Duration: Seed cache with data set A, start 10 deliveries, trigger sync to data set B, then start 10 more deliveries after sync completes
   - Expected: First 10 deliveries use data set A (or a consistent snapshot). All 10 post-sync deliveries use data set B. No delivery mixes data from set A and set B within a single delivery.

3. **Rapid consecutive syncs**
   - Duration: Trigger 5 sync operations in rapid succession (< 10ms apart) while 10 deliveries are in-flight
   - Expected: All syncs complete without deadlock. Final cache state reflects the last sync's data. In-flight deliveries complete without error. No goroutine leaks from abandoned sync operations.

4. **Sync under zero-load**
   - Duration: With no deliveries in-flight, trigger a cache sync/reset
   - Expected: Sync completes within target time. New deliveries after sync use updated data. Baseline for latency comparison with loaded scenarios.

5. **Deadlock detection**
   - Duration: Start 20 concurrent deliveries that each read from the cache, simultaneously trigger cache sync that writes to the cache, repeat 50 times
   - Expected: No deadlock detected (test completes within 30s timeout). All deliveries complete. All syncs complete. Read-write contention is handled by appropriate locking (RWMutex or equivalent).

6. **Goroutine leak check over repeated syncs**
   - Duration: Record baseline goroutine count. Run 100 cycles of [start 5 deliveries, trigger sync, wait for completion]. Record final goroutine count.
   - Expected: Final goroutine count is within 2 of baseline. No monotonic goroutine growth across cycles. All delivery and sync goroutines are properly cleaned up.

### Test Implementation Notes
- Use `context.WithTimeout` to enforce deadlock detection (30s timeout per test)
- Inject artificial latency (50ms) into mock delivery handlers to ensure deliveries are truly in-flight during sync
- Use `sync.WaitGroup` to track delivery completion
- Record `runtime.NumGoroutine()` at test start and after each scenario for leak detection
- Compare delivery latency distributions (during-sync vs steady-state) using recorded timestamps
- Use distinct data payloads in pre-sync and post-sync cache states to verify which data each delivery used
- Run with `-race` flag to detect read/write races between delivery reads and sync writes

### Related Test Cases
- TC-PERF-002 (delivery broker depends on instance cache for routing)
- TC-PERF-006 (webhook handler may trigger cache lookups during request processing)
