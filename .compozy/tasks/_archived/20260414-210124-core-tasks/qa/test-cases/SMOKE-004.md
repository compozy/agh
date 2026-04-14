## SMOKE-004: Get Task Detail by ID with Children, Dependencies, Runs, and Events

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that the task detail endpoint returns the full `TaskDetailPayload` including the task record, children array, dependencies array, runs array, and events audit trail.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] At least one task exists with:
  - At least 1 child task
  - At least 1 dependency edge
  - At least 1 run (any status)
  - At least 1 audit event (automatically created with the task)

---

### Test Steps
1. **Create a parent task with children and dependencies**
   - Input: Create parent task, create 1 child via `POST /api/tasks/<parent-id>/children`, create a second independent task, add dependency via `POST /api/tasks/<parent-id>/dependencies`
   - **Expected:** All created successfully.

2. **Get task detail by ID**
   - Input: `GET /api/tasks/<parent-id>`
   - **Expected:** 200 OK. Response matches `TaskDetailPayload` structure:
     ```json
     {
       "task": { "id": "...", "scope": "...", "title": "...", ... },
       "children": [{ "id": "...", "title": "...", ... }],
       "dependencies": [{ "task_id": "...", "depends_on_task_id": "...", "kind": "blocks", ... }],
       "runs": [...],
       "events": [{ "event_type": "task.created", ... }, ...]
     }
     ```

3. **Verify children array**
   - **Expected:** `children` contains the child task created in step 1. Each child has `parent_task_id` matching the parent ID.

4. **Verify events audit trail**
   - **Expected:** `events` contains at minimum a `"task.created"` event with `actor` and `origin` fields populated.

5. **Get non-existent task**
   - Input: `GET /api/tasks/nonexistent-id-12345`
   - **Expected:** 404 Not Found. No task data in response.

---

### Related Test Cases
- SMOKE-002: Create a global task
- SMOKE-003: List tasks
- SMOKE-006: Enqueue and claim a run
