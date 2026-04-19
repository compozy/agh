## TC-INT-003: Synthetic prompt submission persists dedicated events and queues FIFO behind active turns

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 4
**Traceability:** `task_04.md`, ADR-001, ADR-003

---

### Objective

Validate that daemon-owned synthetic prompt submission persists
`synthetic_reentry` events with explicit metadata, remains unavailable through
ordinary user-facing prompt paths, and preserves FIFO ordering when the target
session is already busy.

---

### Preconditions

- [ ] A live session exists and can be held busy with an active turn
- [ ] The execution lane can invoke the daemon-owned synthetic prompt path
- [ ] Session events and transcript output can be inspected after the run

---

### Test Steps

1. **Start a normal prompt so the target session has an active turn**
   - **Expected:** The session enters the active-turn state and continues processing without synthetic interference.

2. **Submit a synthetic prompt with valid wake-up metadata through the daemon-owned path**
   - **Expected:** Submission is accepted only through the dedicated synthetic path and not through ordinary prompt ingress.

3. **Inspect persistence while the original turn is still active**
   - **Expected:** Any persisted synthetic input is recorded as `synthetic_reentry`, not `user_message`, and includes the originating task/task-run metadata.

4. **Submit a second synthetic prompt before the active turn completes**
   - **Expected:** Both synthetic turns remain queued behind the active turn and preserve FIFO ordering by completion/submission sequence.

5. **Allow the active turn to finish and inspect the queued synthetic dispatch order**
   - **Expected:** The queued synthetic turns dispatch in FIFO order, and the resulting persisted events/transcript reflect that order.

6. **Attempt synthetic submission with missing or invalid wake-up metadata**
   - **Expected:** The runtime rejects the request cleanly and no invalid synthetic event is persisted.

---

### Evidence to Capture

- Session id and originating task/task-run ids
- Persisted event payload excerpt showing `synthetic_reentry`
- Queue order evidence for multiple synthetic turns
- Validation failure output for invalid metadata

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Single valid synthetic turn | one active session, one valid synthetic request | dedicated event persisted with metadata |
| Multiple queued synthetic turns | second synthetic request arrives before first dispatches | FIFO preserved |
| Busy session | active prompt already running | synthetic prompt waits behind active turn |
| Invalid metadata | missing task/task-run wake fields | request rejected, no persisted synthetic event |
| Ordinary ingress misuse | attempt to reach synthetic behavior through normal prompt path | rejected or unavailable |

---

### Related Test Cases

- `TC-INT-004`: Synthetic turns preserve transcript, hook, and extension trust boundaries
- `TC-INT-006`: Completion-to-reentry bridge drives synthetic wakeups correctly

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/session/manager_integration_test.go`
- `internal/session/manager_test.go`
- `internal/api/httpapi/httpapi_integration_test.go`
- `internal/api/udsapi/udsapi_integration_test.go`
