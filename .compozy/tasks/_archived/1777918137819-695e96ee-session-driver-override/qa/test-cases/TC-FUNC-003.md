## TC-FUNC-003: Invalid Provider Create Fails Before Persistence or Driver Startup

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 12 minutes
**Created:** 2026-04-21
**Last Updated:** 2026-04-21
**Module:** Session create validation ordering
**Traceability:** Task 02 validation-before-persistence requirements; Task 01 invalid-provider validation; TechSpec create flow step 4 and "Testing Approach"

---

### Objective

Verify that an invalid or unavailable provider selection fails explicitly before AGH writes metadata, updates the global index, or starts the driver.

---

### Preconditions

- [ ] Workspace fixture exists with a known valid agent and a provider name that is intentionally invalid or unavailable.
- [ ] Session metadata root and global DB can be inspected for absence checks.
- [ ] Backend logs/errors are being captured.

---

### Test Steps

1. Attempt to create a session with a provider name that does not resolve in the chosen workspace.
   **Expected:** The request fails explicitly with a descriptive validation error naming the provider or resolution problem.

2. Check whether any session metadata directory or `session.json` was created for the failed attempt.
   **Expected:** No session metadata is written.

3. Inspect the global `sessions` index and session list/status surfaces for a new record tied to the failed attempt.
   **Expected:** No new session row or list entry exists.

4. Review startup/backend logs around the failure.
   **Expected:** The logs identify the failure phase as create-time validation and do not show driver startup for the rejected session.

---

### Evidence to Capture

- Error payload or CLI stderr showing the explicit failure.
- File-system evidence that no `session.json` was created.
- SQLite evidence that no `sessions` row was inserted or updated for the failed attempt.
- Backend log lines with `provider` and `phase=create`.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
| --- | --- | --- |
| Unknown provider name | `provider=does-not-exist` | Explicit validation error before persistence. |
| Provider removed from workspace config | Formerly valid provider | Same failure ordering as an unknown provider. |
| Repeated invalid attempts | Same invalid provider multiple times | No orphaned session state accumulates. |

---

### Related Test Cases

- `TC-FUNC-002` for valid override behavior
- `TC-INT-005` for unavailable persisted provider on resume

---

### Notes

Task 08 should treat any partial side effect here as a blocking regression because it undermines every later persistence and resume guarantee.
