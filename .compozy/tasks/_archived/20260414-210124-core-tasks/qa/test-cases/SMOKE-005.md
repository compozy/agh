## SMOKE-005: Update Task Title via PATCH

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that a task's mutable fields (title, description, metadata, owner) can be updated via the PATCH endpoint, while immutable fields (scope, workspace_id, parent_task_id, created_by, origin) remain unchanged.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] At least one task exists

---

### Test Steps
1. **Update task title**
   - Input: `PATCH /api/tasks/<task-id>` with body:
     ```json
     {
       "title": "Updated smoke test title"
     }
     ```
   - **Expected:** 200 OK. Response shows `title: "Updated smoke test title"`. `updated_at` timestamp is newer than `created_at`.

2. **Verify update persisted**
   - Input: `GET /api/tasks/<task-id>`
   - **Expected:** Task detail shows the updated title. All immutable fields (scope, created_by, origin) unchanged from original values.

3. **Update multiple mutable fields**
   - Input: `PATCH /api/tasks/<task-id>` with body:
     ```json
     {
       "description": "Updated description",
       "owner": {"kind": "human", "ref": "user-2"}
     }
     ```
   - **Expected:** 200 OK. Both description and owner updated.

4. **Verify audit event for update**
   - Input: `GET /api/tasks/<task-id>` and inspect events
   - **Expected:** Events array contains a `"task.updated"` event.

5. **PATCH with empty body (no changes)**
   - Input: `PATCH /api/tasks/<task-id>` with body `{}`
   - **Expected:** 200 OK or 400 (no changes). Task unchanged. No spurious audit events.

---

### Related Test Cases
- SMOKE-002: Create a global task
- SMOKE-004: Get task detail
- TC-SEC-006: SQL injection in title field
