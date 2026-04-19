## TC-INT-006: Detached completion wakes the owning session or records an explicit drop without duplicates

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-04-18
**Workstream:** Workstream 4 and Workstream 5
**Traceability:** `task_07.md`, ADR-003

---

### Objective

Validate that detached harness task-run completion is bridged into either a
synthetic wakeup or an explicit observable drop/silent-completion outcome, with
FIFO queueing for busy sessions and duplicate protection across repeated
terminal notifications.

---

### Preconditions

- [ ] A live target session exists and can accept or reject wakeups based on policy
- [ ] Detached harness work can be submitted and completed through the task runtime
- [ ] Session events, transcript output, and harness summaries can all be inspected

---

### Test Steps

1. **Submit detached harness work whose policy should wake a live session**
   - **Expected:** The detached run is accepted and tied to the target session through durable metadata.

2. **Drive the run to a terminal state**
   - **Expected:** The bridge emits completion observability and creates a synthetic wake through the dedicated synthetic prompt path.

3. **Inspect the target session after the wakeup**
   - **Expected:** The session history contains a `synthetic_reentry` event that references the originating `task_run`, and the transcript shows the daemon-originated turn.

4. **Repeat the scenario with a silent/drop policy or an unavailable target session**
   - **Expected:** No synthetic wake is emitted, and the runtime records an explicit drop/silent outcome with the reason visible in harness summaries.

5. **Complete multiple runs targeting the same busy session**
   - **Expected:** Synthetic wakeups queue behind the active turn and preserve FIFO order.

6. **Replay a duplicate terminal notification or cross a restart boundary**
   - **Expected:** The same `task_run` does not emit duplicate synthetic wakeups.

---

### Evidence to Capture

- Task-run ids and target session id
- Persisted `synthetic_reentry` event payload showing originating task-run metadata
- Ordered harness summaries for completion, emitted wake, and dropped/silent outcomes
- Evidence that FIFO ordering held when multiple completions targeted the same session
- Evidence that duplicate terminal handling did not create a second wakeup

---

### Edge Cases & Variations

| Variation | Input / Condition | Expected Result |
| --- | --- | --- |
| Wake-enabled completion | live resumable session | synthetic wake emitted through dedicated path |
| Silent/drop policy | policy says do not wake | no wake, explicit drop/silent summary |
| Missing/stopped target | target session unavailable | drop recorded, no hidden retry |
| Busy target session | active turn already running | wake queued behind active turn |
| Duplicate terminal notification | same `task_run` reported twice | no duplicate wake emitted |

---

### Related Test Cases

- `TC-INT-003`: Synthetic prompt submission persists dedicated events and queues FIFO
- `TC-INT-005`: Detached harness work persists on the task runtime with stable metadata
- `TC-REG-001`: Recovery and duplicate-protection across restart

---

### Notes

Suggested repo-supported runtime anchors:

- `internal/daemon/task_runtime_test.go`
- `internal/daemon/daemon_integration_test.go`
