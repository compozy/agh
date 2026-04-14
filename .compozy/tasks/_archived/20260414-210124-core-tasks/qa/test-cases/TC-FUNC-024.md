## TC-FUNC-024: Cancel propagation to grandchildren (entire subtree)

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-14

---

### Objective
Validate that cancelling a task propagates cancellation recursively through the entire subtree -- children, grandchildren, and all their associated runs. All non-terminal tasks and runs in the subtree must transition to "cancelled".

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager, backing Store, and mock SessionExecutor)
- [ ] Task hierarchy:
  - Root task (status="in_progress")
    - Child A (status="in_progress", has running run with session)
      - Grandchild A1 (status="pending", has queued run)
      - Grandchild A2 (status="ready", no runs)
    - Child B (status="blocked", has no runs)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Cancel the root task**
   - Call CancelTask(ctx, rootID, CancelTask{Reason: "project cancelled"}, actor)
   - **Expected:** No error returned

2. **Verify root task is cancelled**
   - **Expected:** Status == "cancelled", ClosedAt set

3. **Verify Child A is cancelled**
   - **Expected:** Status == "cancelled", ClosedAt set
   - **Expected:** Child A's running run is cancelled, session stop requested

4. **Verify Grandchild A1 is cancelled**
   - **Expected:** Status == "cancelled", ClosedAt set
   - **Expected:** Grandchild A1's queued run is cancelled immediately

5. **Verify Grandchild A2 is cancelled**
   - **Expected:** Status == "cancelled", ClosedAt set (even with no runs)

6. **Verify Child B is cancelled**
   - **Expected:** Status == "cancelled", ClosedAt set (was blocked, now cancelled)

7. **Verify audit trail**
   - **Expected:** task.cancelled events for: root, Child A, Grandchild A1, Grandchild A2, Child B
   - **Expected:** task.run_cancelled events for all active runs in the subtree

8. **Verify no sibling tasks outside the subtree are affected**
   - Create a separate top-level task before the test
   - **Expected:** Unrelated task remains unaffected

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Deep subtree (3+ levels) | Root -> Child -> Grandchild -> Great-grandchild | All levels cancelled |
| Mixed terminal and non-terminal | Some children already completed | Completed children remain "completed"; only non-terminal children cancelled |
| Empty subtree | Root has no children | Only root cancelled |
| Subtree with fan-out | Root has 10 children, each with 5 grandchildren | All 60 tasks in subtree cancelled |

---

### Related Test Cases
- TC-FUNC-022: Cancel task with queued runs
- TC-FUNC-023: Cancel parent with running child runs
- TC-FUNC-025: Cancel already-terminal task
- TC-FUNC-006: Create child task under valid parent
