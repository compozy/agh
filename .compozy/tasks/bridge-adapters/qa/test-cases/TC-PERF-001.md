## TC-PERF-001: Dedup Cache Memory Bounds

**Priority:** P0
**Type:** Performance
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-15

---

### Objective
Verify the DedupCache enforces its maximum size bound (2000 items), correctly evicts oldest entries under pressure, reclaims TTL-expired entries, and does not exhibit unbounded memory growth.

### Preconditions
- [ ] DedupCache implementation is available and importable
- [ ] Test harness can measure heap allocations (e.g., `runtime.ReadMemStats` or `testing.B` with `b.ReportAllocs`)
- [ ] TTL is configurable for test acceleration (use 50ms TTL instead of 5-minute default)
- [ ] No other cache instances running in the test process

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Insert throughput (ops/sec) | > 500,000 | > 200,000 | | |
| Cache size after 5000 inserts (max_size=2000) | exactly 2000 | 2000 | | |
| Eviction latency per insert (p99) | < 5us | < 20us | | |
| Heap delta after insert + full eviction cycle | < 1 MB | < 5 MB | | |
| TTL cleanup reclaims all expired entries | 100% reclaimed | 100% reclaimed | | |
| Concurrent insert throughput (8 goroutines) | > 300,000 ops/sec | > 100,000 ops/sec | | |

### Load Scenarios

1. **Sequential overflow insert**
   - Duration: Insert 5000 unique keys sequentially into a cache with max_size=2000
   - Expected: Cache size is exactly 2000 after all inserts. The 3000 oldest keys are no longer present. The 2000 newest keys are all present.

2. **TTL expiration sweep**
   - Duration: Insert 500 keys with TTL=50ms, wait 100ms, then query all keys
   - Expected: All 500 keys return miss. Internal data structures report size 0. No residual memory from expired entries after cleanup.

3. **Mixed TTL and overflow eviction**
   - Duration: Insert 1500 keys with TTL=50ms, wait 60ms, then insert 1000 more keys with TTL=5s
   - Expected: Expired entries are cleaned before or during new inserts. Cache size is 1000 (only the fresh keys). Memory footprint reflects only live entries.

4. **Concurrent insert stress**
   - Duration: 8 goroutines each insert 1000 unique keys concurrently (8000 total unique keys) into max_size=2000
   - Expected: No data races (pass with `-race`). Cache size is exactly 2000. No panics or deadlocks. Insert throughput measured under contention.

5. **Memory footprint stability**
   - Duration: Run 10 cycles of insert-2000-keys then wait-for-TTL-expiry
   - Expected: Heap allocation delta between cycle 1 and cycle 10 is < 1 MB. No monotonic memory growth indicating a leak.

6. **Duplicate key idempotency**
   - Duration: Insert the same 100 keys 50 times each (5000 total inserts, 100 unique)
   - Expected: Cache size is 100. Insert of existing key refreshes TTL but does not increase size. Throughput is not degraded compared to unique-key inserts.

### Test Implementation Notes
- Use `testing.B` benchmarks for throughput measurements
- Use `runtime.ReadMemStats` before and after each scenario for memory footprint
- Run with `-race` flag to detect concurrent access violations
- Use short TTLs (50ms) to make expiration testable without long waits

### Related Test Cases
- TC-PERF-002 (delivery throughput depends on dedup correctness)
- TC-PERF-003 (batcher may interact with dedup for duplicate detection)
