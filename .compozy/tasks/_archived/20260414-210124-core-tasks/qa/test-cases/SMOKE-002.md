## SMOKE-002: Create a Global Task via HTTP API

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that a global-scoped task can be created through the HTTP API, persisted, and returned with server-derived identity fields and correct initial status.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated HTTP client available

---

### Test Steps
1. **Create a global task**
   - Input: `POST /api/tasks` with body:
     ```json
     {
       "scope": "global",
       "title": "Smoke test task",
       "description": "Created during smoke testing"
     }
     ```
   - **Expected:** 201 Created. Response includes:
     - `id`: non-empty string (server-generated, e.g., `"tsk_..."`)
     - `scope`: `"global"`
     - `title`: `"Smoke test task"`
     - `status`: `"pending"` or `"ready"`
     - `created_by.kind`: matches authenticated principal kind
     - `origin.kind`: `"http"`
     - `created_at`: valid ISO 8601 timestamp

2. **Verify task appears in list**
   - Input: `GET /api/tasks`
   - **Expected:** 200 OK. Array contains at least one task with the title `"Smoke test task"`.

3. **Verify task detail retrieval**
   - Input: `GET /api/tasks/<id>` using the ID from step 1
   - **Expected:** 200 OK. Full task detail including empty `children`, `dependencies`, `runs`, and `events` arrays (except the `task.created` event).

---

### Related Test Cases
- SMOKE-001: Daemon starts with task subsystem
- SMOKE-003: List tasks via HTTP API
- TC-SEC-001: Server-derived created_by identity
