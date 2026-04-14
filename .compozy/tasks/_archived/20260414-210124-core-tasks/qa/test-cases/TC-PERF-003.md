## TC-PERF-003: Maximum Hierarchy Depth (8 Levels) with 64 Children Per Level

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that creating the maximum allowed task hierarchy (8 levels deep, `MaxHierarchyDepth = 8`) with the maximum children per parent (`MaxDirectChildren = 64`) completes within acceptable time bounds and does not exhibit exponential slowdown. Total potential tree: up to 64^8 nodes at full fan-out, but the test focuses on creating one full path and one level of max children.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated principal with full write access
- [ ] Clean task store
- [ ] Sufficient disk space for SQLite (estimated ~5MB for this test)

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Create 8-level deep chain (1 child/level) | <100ms total | <200ms | | [ ] |
| Create 64 children at level 1 | <500ms total | <1000ms | | [ ] |
| Create 64 children at level 8 (deepest) | <500ms total | <1000ms | | [ ] |
| Depth validation per child creation | <5ms | <10ms | | [ ] |
| No exponential slowdown across levels | Level N+1 <= 1.5x Level N | Level N+1 <= 2x Level N | | [ ] |

---

### Test Steps
1. **Create 8-level deep chain**
   - Input: Create root task, then sequentially create one child at each level via `CreateChildTask` (root -> L1 -> L2 -> ... -> L8)
   - Record per-level creation latency
   - **Expected:** All 8 levels created successfully. Total time < 100ms. No `ErrGraphLimitExceeded`.

2. **Attempt to create 9th level (exceeds MaxHierarchyDepth)**
   - Input: Create a child of the level-8 task
   - **Expected:** `ErrGraphLimitExceeded` returned. Error message mentions hierarchy depth. Child not persisted.

3. **Create 64 children at root level**
   - Input: Create 64 children of the root task via `CreateChildTask`
   - Record per-child creation latency
   - **Expected:** All 64 children created. Total time < 500ms. No exponential slowdown.

4. **Attempt to create 65th child (exceeds MaxDirectChildren)**
   - Input: Create a 65th child of the root task
   - **Expected:** `ErrGraphLimitExceeded` returned. Child not persisted.

5. **Create 64 children at the deepest valid level (level 7, so children are at level 8)**
   - Input: Create 64 children at level 7 (children become level 8)
   - Record total creation time
   - **Expected:** Total time < 500ms. Depth validation at level 8 does not add significant overhead compared to level 1.

6. **Compare creation latency across levels**
   - Input: Analyze per-level latency from steps 1, 3, and 5
   - **Expected:** No level shows > 1.5x the latency of the previous level. Linear scaling, not exponential.

7. **Verify parent-child relationships via GetTask detail**
   - Input: `GetTask` for the root task
   - **Expected:** `children` array contains all direct children. Response time < 100ms even with 64 children.

---

### Related Test Cases
- TC-PERF-001: Sequential task creation throughput
- TC-PERF-004: Cancellation propagation on large tree
- SMOKE-004: Task detail retrieval with children
