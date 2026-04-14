## SMOKE-003: List Tasks via HTTP API

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that the task list endpoint returns a 200 response with an array of task summaries, supporting basic query filter parameters.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] At least 2 tasks exist (1 global, 1 workspace-scoped) created via prior smoke tests or seeding

---

### Test Steps
1. **List all tasks (no filters)**
   - Input: `GET /api/tasks`
   - **Expected:** 200 OK. Response body is a JSON array. Each element contains at minimum: `id`, `scope`, `title`, `status`, `created_by`, `origin`, `created_at`.

2. **List tasks with scope filter**
   - Input: `GET /api/tasks?scope=global`
   - **Expected:** 200 OK. All returned tasks have `scope: "global"`. No workspace-scoped tasks in results.

3. **List tasks with status filter**
   - Input: `GET /api/tasks?status=pending`
   - **Expected:** 200 OK. All returned tasks have `status: "pending"`.

4. **List tasks with limit parameter**
   - Input: `GET /api/tasks?limit=1`
   - **Expected:** 200 OK. Array contains at most 1 element.

5. **List tasks with invalid filter value**
   - Input: `GET /api/tasks?scope=invalid_scope`
   - **Expected:** 400 Bad Request or empty results (depending on validation strategy). No 500 error.

---

### Related Test Cases
- SMOKE-002: Create a global task
- SMOKE-004: Get task detail by ID
- TC-PERF-005: ListTasks filter performance on large dataset
