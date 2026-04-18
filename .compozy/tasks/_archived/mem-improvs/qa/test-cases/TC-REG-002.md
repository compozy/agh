## TC-REG-002: Observe Events Include Memory Operations

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Observe Events
**Requirement:** REQ-MEM-007

---

### Objective

Verify that memory operations appear in the global observe event stream/list so operators can audit write, delete, search, and reindex activity.

---

### Preconditions

- [ ] Daemon is running with observe endpoints enabled.
- [ ] `/api/observe/events` is reachable.
- [ ] The tester can trigger memory write, search, delete, and reindex operations.

---

### Test Steps

1. Trigger a memory write, search, and reindex operation against the same daemon.
   - **Expected:** Each operation succeeds.

2. Call `GET /api/observe/events`.
   - **Expected:** Response status is `200` and returns event summaries ordered for consumption.

3. Inspect the returned event types.
   - **Expected:** Recent summaries include `memory.write`, `memory.search`, and `memory.reindex`.

4. Trigger a delete and query observe events again.
   - **Expected:** `memory.delete` also appears without requiring a session-scoped filter.

5. Apply a session filter if supported.
   - **Expected:** Session-scoped filtering excludes daemon-global memory operation rows when appropriate.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| No recent memory ops | clean daemon | No memory rows returned, but endpoint remains stable |
| Limit filter | small event limit | Recent memory events remain visible if within limit |
| Mixed session + daemon events | regular session traffic plus memory ops | Ordering remains stable and types are distinguishable |

---

### Related Test Cases

- `TC-FUNC-001`
- `TC-REG-001`

---

### Notes

This guards against future regressions where memory operations are recorded internally but disappear from operator-facing observe surfaces.
