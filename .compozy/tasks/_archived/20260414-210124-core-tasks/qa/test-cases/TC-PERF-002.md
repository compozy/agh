## TC-PERF-002: Dependency Cycle Detection at MaxDependencyCount (32 Edges)

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that cycle detection remains performant when a task reaches the maximum dependency count (`MaxDependencyCount = 32`). Each dependency edge addition must complete cycle detection within 50ms even at the limit. The system must correctly detect and reject cycles without performance degradation.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated principal with full write access
- [ ] 33+ tasks created (1 target task + 32 dependency targets)
- [ ] Clean dependency graph (no pre-existing edges)

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Cycle detection per edge at 32 deps | <50ms | <100ms | | [ ] |
| Total time to fill 32 dependency slots | <1600ms | <3200ms | | [ ] |
| Cycle detection on rejection (cycle found) | <50ms | <100ms | | [ ] |
| Memory usage for dependency graph traversal | <10MB | <25MB | | [ ] |

---

### Test Steps
1. **Create 33 tasks for dependency graph**
   - Input: Create tasks `dep-target-01` through `dep-target-32` and one `dep-source` task
   - **Expected:** All 33 tasks created successfully.

2. **Add dependencies up to MaxDependencyCount**
   - Input: For each of the 32 target tasks, call `AddDependency` from `dep-source` to `dep-target-NN` with `kind: "blocks"`
   - Record per-edge latency
   - **Expected:** All 32 edges added successfully. Each edge addition completes in < 50ms. No `ErrCycleDetected` or `ErrGraphLimitExceeded`.

3. **Attempt to exceed MaxDependencyCount**
   - Input: Add a 33rd dependency edge
   - **Expected:** `ErrGraphLimitExceeded` returned. Edge not persisted. Error message mentions dependency count limit.

4. **Attempt to create a cycle at the limit**
   - Input: Add a dependency from `dep-target-01` back to `dep-source` (creating a cycle)
   - **Expected:** `ErrCycleDetected` returned within 50ms. Cycle detection does not degrade with 32 existing edges.

5. **Verify status reconciliation**
   - Input: Complete all 32 dependency tasks, then check `dep-source` status
   - **Expected:** `dep-source` transitions from `"blocked"` to `"ready"` as dependencies resolve.

6. **Measure cycle detection in chain topology**
   - Input: Create a chain of 32 tasks (A->B->C->...->AF). Attempt to add AF->A (closing the cycle).
   - **Expected:** `ErrCycleDetected` returned within 50ms despite the graph traversal depth of 32.

---

### Related Test Cases
- TC-PERF-003: Hierarchy depth performance
- TC-SEC-006: SQL injection in dependency operations
