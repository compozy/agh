## TC-SEC-003: Unauthenticated Request Rejection on All Task Endpoints

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that every task-domain endpoint rejects unauthenticated requests with a 403 (or 401) response and returns no task data. The task system requires an authenticated principal for all operations; anonymous access must be impossible.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] HTTP client configured to send requests WITHOUT authentication credentials
- [ ] At least one task exists in the store (for read endpoint testing)

---

### Test Steps
1. **Unauthenticated POST /api/tasks (create)**
   - Input: `POST /api/tasks` with valid JSON body `{"scope":"global","title":"anon task"}`, no auth headers
   - **Expected:** 403 Forbidden (or 401 Unauthorized). Response body contains no task data.

2. **Unauthenticated GET /api/tasks (list)**
   - Input: `GET /api/tasks`, no auth headers
   - **Expected:** 403 Forbidden. No task summaries returned.

3. **Unauthenticated GET /api/tasks/:id (detail)**
   - Input: `GET /api/tasks/<existing-task-id>`, no auth headers
   - **Expected:** 403 Forbidden. No task detail returned. Must NOT return 404 (which would leak existence).

4. **Unauthenticated PATCH /api/tasks/:id (update)**
   - Input: `PATCH /api/tasks/<existing-task-id>` with `{"title":"hacked"}`, no auth headers
   - **Expected:** 403 Forbidden. Task title unchanged when verified by authenticated read.

5. **Unauthenticated POST /api/tasks/:id/cancel**
   - Input: `POST /api/tasks/<existing-task-id>/cancel`, no auth headers
   - **Expected:** 403 Forbidden. Task status unchanged.

6. **Unauthenticated POST /api/tasks/:id/runs (enqueue)**
   - Input: `POST /api/tasks/<existing-task-id>/runs`, no auth headers
   - **Expected:** 403 Forbidden. No run created.

7. **Unauthenticated POST /api/task-runs/:id/claim**
   - Input: `POST /api/task-runs/<existing-run-id>/claim`, no auth headers
   - **Expected:** 403 Forbidden. Run status unchanged.

8. **Unauthenticated POST /api/task-runs/:id/complete**
   - Input: `POST /api/task-runs/<existing-run-id>/complete`, no auth headers
   - **Expected:** 403 Forbidden. Run status unchanged.

9. **Unauthenticated POST /api/tasks/:id/children (create child)**
   - Input: `POST /api/tasks/<existing-task-id>/children` with valid body, no auth headers
   - **Expected:** 403 Forbidden.

10. **Unauthenticated POST /api/tasks/:id/dependencies (add dependency)**
    - Input: `POST /api/tasks/<existing-task-id>/dependencies` with valid body, no auth headers
    - **Expected:** 403 Forbidden.

11. **Unauthenticated DELETE /api/tasks/:id/dependencies/:depends_on_id**
    - Input: `DELETE /api/tasks/<existing-task-id>/dependencies/<dep-id>`, no auth headers
    - **Expected:** 403 Forbidden.

---

### Attack Vectors
- [ ] Direct HTTP request without any authentication headers
- [ ] Empty or malformed authentication token
- [ ] Expired authentication token
- [ ] Authentication header with invalid scheme
- [ ] Probing task existence via differential 403 vs 404 responses (information leakage)

---

### Related Test Cases
- TC-SEC-004: Extension without task.write capability
- TC-SEC-008: Unauthorized scope read rejection
