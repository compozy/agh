## TC-PERF-004: Cancellation Propagation on Tree with 100 Descendants

**Priority:** P1
**Type:** Performance
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that cancelling a parent task propagates cancellation to all descendant tasks and their active runs within 2 seconds. The cancellation must be transitive (parent -> children -> grandchildren) and must handle active run lifecycle transitions (cooperative stop then force stop after grace period).

---

### Preconditions
- [ ] AGH daemon running with task subsystem and session executor initialized
- [ ] Authenticated principal with full write access
- [ ] Tree structure created: root task with 100 descendants across multiple levels
  - Suggested: 10 children, each with 10 grandchildren (2 levels, 110 total including root)
- [ ] Some descendant tasks have active runs in `queued`, `claimed`, `starting`, or `running` states
- [ ] Mock session executor configured to respond to stop requests

---

### Performance Criteria
| Metric | Target | Acceptable | Actual | Status |
|--------|--------|------------|--------|--------|
| Total cancellation propagation time | <2s | <5s | | [ ] |
| All 100 descendants reach `cancelled` status | 100% | 100% | | [ ] |
| All active runs reach `cancelled` status | 100% | 100% | | [ ] |
| Cancellation audit events emitted for all nodes | 100 events | 100 events | | [ ] |
| No orphaned runs left in non-terminal state | 0 orphans | 0 orphans | | [ ] |

---

### Test Steps
1. **Build the task tree**
   - Input: Create root task, 10 children, and 10 grandchildren per child (110 total tasks)
   - **Expected:** All tasks created successfully.

2. **Enqueue and advance runs on select descendants**
   - Input: Enqueue runs on 20 descendant tasks. Claim 10, start 5, leave 10 queued.
   - **Expected:** Runs in expected states: 10 queued, 5 claimed, 5 running.

3. **Cancel the root task**
   - Input: `CancelTask(ctx, rootID, CancelTask{Reason: "perf-test"}, actor)`. Start timer.
   - **Expected:** Root task transitions to `"cancelled"`. Cancellation propagation begins.

4. **Verify all descendants cancelled within 2 seconds**
   - Input: Poll or wait up to 2s, then `ListTasks` with `parent_task_id` filters down the tree
   - **Expected:** All 100 descendant tasks have status `"cancelled"`. All 20 active runs have status `"cancelled"`. Timer shows < 2s total.

5. **Verify audit trail completeness**
   - Input: Query task events for `"task.cancelled"` event type
   - **Expected:** At least 101 cancellation events (root + 100 descendants). Each event includes the cancellation reason.

6. **Verify session stop requests issued**
   - Input: Check mock session executor for stop requests
   - **Expected:** Stop requests issued for all 5 running sessions. Grace period respected before force-stop.

---

### Related Test Cases
- TC-PERF-003: Hierarchy depth and child count limits
- SMOKE-008: Basic task cancellation
- TC-SEC-004: Permission check during cancellation
