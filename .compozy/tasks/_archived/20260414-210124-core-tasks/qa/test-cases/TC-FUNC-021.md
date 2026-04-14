## TC-FUNC-021: Idempotent run enqueue with same key+origin returns same run

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 5 minutes
**Created:** 2026-04-14

---

### Objective
Validate that enqueueing a run with the same idempotency_key and same origin returns the previously created run (idempotent behavior). Enqueueing with the same idempotency_key but a different origin creates a new, separate run (idempotency is scoped to origin).

---

### Preconditions
- [ ] AGH daemon running (or test harness initialized with TaskManager and backing Store)
- [ ] One existing task in executable state (pending/ready with no blockers)
- [ ] ActorContext with Authority.Write=true

---

### Test Steps

1. **Enqueue a run with idempotency_key="idem-1"**
   - Input:
     ```json
     {
       "task_id": "<task-id>",
       "idempotency_key": "idem-1"
     }
     ```
     ActorContext: Origin={Kind:"cli", Ref:"term-1"}
   - **Expected:** Run created with status="queued"; idempotency record persisted

2. **Record the returned run ID (run-A)**

3. **Enqueue again with same key="idem-1" and same origin**
   - Same ActorContext as step 1
   - **Expected:** Returns the same run (run-A); no new run created; run ID matches step 2

4. **Verify only one run exists for the task**
   - Query runs for the task
   - **Expected:** Exactly one run record

5. **Enqueue with same key="idem-1" but different origin**
   - ActorContext: Origin={Kind:"web", Ref:"browser-1"} (different from step 1)
   - **Expected:** A new, separate run is created (run-B) with a different ID; idempotency is per-origin

6. **Verify two runs now exist for the task**
   - Query runs for the task
   - **Expected:** Two run records (run-A and run-B)

7. **Enqueue with no idempotency_key (empty)**
   - Input: idempotency_key=""
   - **Expected:** Always creates a new run (no deduplication without a key)

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Same key + same origin | Re-enqueue | Returns existing run (idempotent) |
| Same key + different origin | Different origin context | New run created |
| No idempotency_key | key="" | Always new run |
| Same key after run completed | Re-enqueue after original run finished | Depends on implementation; may create new run or return old |
| Whitespace-only key | key="   " | Treated as empty (no deduplication) |

---

### Related Test Cases
- TC-FUNC-014: Enqueue run on ready task
- TC-FUNC-015: Claim queued run
