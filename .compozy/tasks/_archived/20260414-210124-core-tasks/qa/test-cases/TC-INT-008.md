## TC-INT-008: CLI `agh task cancel <id> --reason "No longer needed"` cancels task and children

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that the `agh task cancel` CLI command cancels the target task and propagates cancellation to all its child tasks, including stopping any active runs and their bound sessions.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] UDS server listening on `/tmp/.agh/daemon.sock`
- [ ] `agh` binary available in PATH
- [ ] Seed task hierarchy:
  - Parent task P (scope=global, title="Parent To Cancel", status=pending)
  - Child C1 under P (scope=global, title="Child 1", status=pending)
  - Child C2 under P (scope=global, title="Child 2", status=pending)
  - Grandchild GC1 under C1 (scope=global, title="Grandchild 1", status=pending)
  - Optional: an active run on C1 to verify run cancellation cascade

---

### Test Steps

1. **Cancel the parent task with reason**
   - Input:
     ```bash
     agh task cancel <P.id> --reason "No longer needed"
     ```
   - **Expected:** Exit code 0
   - **Expected:** Output shows Task section with:
     - ID: <P.id>
     - Status: cancelled
     - Updated timestamp later than previous

2. **Verify parent task status via API**
   - Input: `GET http://localhost:2123/api/tasks/<P.id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.status` equals "cancelled"
   - **Expected:** `task.task.closed_at` is a valid timestamp (set on cancellation)

3. **Verify child C1 is cancelled**
   - Input: `GET http://localhost:2123/api/tasks/<C1.id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.status` equals "cancelled"
   - **Expected:** `task.task.closed_at` is set

4. **Verify child C2 is cancelled**
   - Input: `GET http://localhost:2123/api/tasks/<C2.id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.status` equals "cancelled"

5. **Verify grandchild GC1 is cancelled (cascade depth)**
   - Input: `GET http://localhost:2123/api/tasks/<GC1.id>`
   - **Expected:** HTTP 200
   - **Expected:** `task.task.status` equals "cancelled"

6. **Verify cancellation events were recorded**
   - Input: `GET http://localhost:2123/api/tasks/<P.id>` (check events array)
   - **Expected:** Events array contains a "task_cancelled" event
   - **Expected:** Event includes the cancellation reason in its payload

7. **Cancel with metadata**
   - Create a fresh task T2
   - Input:
     ```bash
     agh task cancel <T2.id> --reason "Budget cut" --metadata '{"ticket":"JIRA-123"}'
     ```
   - **Expected:** Exit code 0
   - **Expected:** Task status is cancelled

8. **Cancel an already-cancelled task**
   - Input:
     ```bash
     agh task cancel <P.id> --reason "Double cancel"
     ```
   - **Expected:** Non-zero exit code or 409 Conflict (ErrInvalidStatusTransition)
   - **Expected:** Error message indicates the task cannot transition from cancelled state

9. **Cancel with active run**
   - Create task T3, enqueue and start a run on T3
   - Input: `agh task cancel <T3.id> --reason "abort run"`
   - **Expected:** Exit code 0
   - **Expected:** Task T3 status is "cancelled"
   - **Expected:** The active run on T3 is also cancelled (verify via GET runs)
   - **Expected:** If the run had a bound session, SessionExecutor.RequestTaskStop was called

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Parent P status | After cancel | "cancelled" | [ ] |
| Parent P closed_at | After cancel | Non-null timestamp | [ ] |
| Child C1 status | After parent cancel | "cancelled" | [ ] |
| Child C2 status | After parent cancel | "cancelled" | [ ] |
| Grandchild GC1 status | After parent cancel | "cancelled" | [ ] |
| Cancellation event | events array | Contains "task_cancelled" | [ ] |
| Active run status | After task cancel | "cancelled" | [ ] |
| Exit code (success) | CLI exit | 0 | [ ] |
| Exit code (double cancel) | CLI exit | Non-zero | [ ] |

---

### Error Scenarios
- [ ] Cancel non-existent task ID returns error (ErrTaskNotFound, exit non-zero)
- [ ] Cancel a completed task returns 409 (ErrInvalidStatusTransition)
- [ ] Cancel with metadata exceeding 64 KiB returns 413 (ErrPayloadTooLarge)
- [ ] Cancel with invalid metadata JSON exits with parse error

---

### Related Test Cases
- TC-INT-001: Creates tasks for cancellation
- TC-INT-003: Verifies detail payload after cancellation
- TC-INT-006: CLI create used to set up the hierarchy
- TC-INT-009: Session bridge interaction when runs are active during cancel
