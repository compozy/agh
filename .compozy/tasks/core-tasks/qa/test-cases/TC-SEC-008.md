## TC-SEC-008: Unauthorized Scope Read Rejected with ErrPermissionDenied

**Priority:** P0
**Type:** Security
**Risk Level:** High
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system enforces read authority checks. Principals with `Authority.Read == false` must receive `ErrPermissionDenied` when attempting to read tasks via `GetTask` or `ListTasks`. Extensions and network peers must have appropriate capabilities to read task data.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Extension actor context with `Authority{Read: false, Write: true}` (read-denied)
- [ ] Network peer actor context with read-denied authority
- [ ] At least one global task and one workspace-scoped task exist in the store

---

### Test Steps
1. **Extension with no read authority attempts GetTask**
   - Input: Extension calls `GetTask(ctx, taskID, actorCtx)` with `Authority{Read: false}`
   - **Expected:** `ErrPermissionDenied` returned. No task data in the response.

2. **Extension with no read authority attempts ListTasks**
   - Input: Extension calls `ListTasks(ctx, TaskQuery{}, actorCtx)` with `Authority{Read: false}`
   - **Expected:** `ErrPermissionDenied` returned. Empty response, no task summaries leaked.

3. **Network peer with no read authority attempts task read**
   - Input: Network peer calls `GetTask` with `Authority{Read: false}`
   - **Expected:** `ErrPermissionDenied` returned.

4. **Read-denied principal attempts to read task runs**
   - Input: Call task run list with `Authority{Read: false}`
   - **Expected:** `ErrPermissionDenied` returned. No run data exposed.

5. **Extension with read authority succeeds (control)**
   - Input: Extension calls `GetTask` with `Authority{Read: true}`
   - **Expected:** Task data returned successfully. Confirms the read gate is the only barrier.

6. **Verify no data leakage in error response**
   - Input: Inspect the error response from steps 1-4
   - **Expected:** Error contains only `"task: permission denied"`. No task IDs, titles, or metadata leaked in the error message.

7. **HTTP error mapping for permission denied**
   - Input: Trigger `ErrPermissionDenied` via HTTP API
   - **Expected:** HTTP status 403 Forbidden. Response body does not include any task data.

---

### Attack Vectors
- [ ] Extension with write-only authority attempts to read task details (write access should not imply read access)
- [ ] Read-denied principal probes task existence via differential error responses (403 vs 404)
- [ ] Read-denied principal attempts to access task events or audit trail
- [ ] Capability escalation by re-deriving actor context with elevated read authority

---

### Related Test Cases
- TC-SEC-003: Unauthenticated request rejection
- TC-SEC-004: Extension without task.write capability
