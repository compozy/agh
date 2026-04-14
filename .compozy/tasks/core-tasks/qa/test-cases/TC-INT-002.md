## TC-INT-002: HTTP GET /api/v1/tasks with query filters returns filtered list

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 8 minutes
**Created:** 2026-04-14

---

### Objective
Validate that HTTP GET `/api/tasks` correctly applies scope, status, owner_kind, owner_ref, network_channel, parent_task_id, and limit query parameters, returning only matching TaskSummaryPayload records in the `tasks` array.

---

### Preconditions
- [ ] AGH daemon running with all subsystems
- [ ] HTTP server listening on TCP :2123
- [ ] At least one workspace registered (for workspace-scoped tasks)
- [ ] Seed data: create the following tasks before running filter tests:
  - Task A: scope=global, status=pending, title="Global Pending A"
  - Task B: scope=global, status=ready, title="Global Ready B"
  - Task C: scope=workspace, status=pending, workspace=<ws>, owner={kind:"human", ref:"alice"}, title="WS Pending C"
  - Task D: scope=workspace, status=in_progress, workspace=<ws>, owner={kind:"agent_session", ref:"sess-1"}, title="WS InProgress D"
  - Task E: scope=global, status=pending, title="Global Pending E", network_channel="ch-test"
  - Task F: child of Task A, scope=global, status=pending, title="Child of A"

---

### Test Steps

1. **List all tasks without filters**
   - Input: `GET http://localhost:2123/api/tasks`
   - **Expected:** HTTP 200
   - **Expected:** Response `{"tasks": [...]}` contains all 6 seeded tasks (A through F)
   - **Expected:** Each entry is a TaskSummaryPayload with id, scope, status, title, created_by, origin, timestamps

2. **Filter by scope=global**
   - Input: `GET http://localhost:2123/api/tasks?scope=global`
   - **Expected:** HTTP 200
   - **Expected:** Only tasks A, B, E, F returned (all global-scope tasks)
   - **Expected:** Tasks C, D excluded (workspace-scoped)

3. **Filter by status=pending**
   - Input: `GET http://localhost:2123/api/tasks?status=pending`
   - **Expected:** HTTP 200
   - **Expected:** Tasks A, C, E, F returned
   - **Expected:** Tasks B, D excluded

4. **Filter by scope=global AND status=pending (combined)**
   - Input: `GET http://localhost:2123/api/tasks?scope=global&status=pending`
   - **Expected:** HTTP 200
   - **Expected:** Tasks A, E, F returned
   - **Expected:** Tasks B, C, D excluded

5. **Filter by owner_kind=human AND owner_ref=alice**
   - Input: `GET http://localhost:2123/api/tasks?owner_kind=human&owner_ref=alice`
   - **Expected:** HTTP 200
   - **Expected:** Only Task C returned

6. **Filter by workspace**
   - Input: `GET http://localhost:2123/api/tasks?scope=workspace&workspace=<ws-ref>`
   - **Expected:** HTTP 200
   - **Expected:** Tasks C, D returned

7. **Filter by network_channel**
   - Input: `GET http://localhost:2123/api/tasks?network_channel=ch-test`
   - **Expected:** HTTP 200
   - **Expected:** Only Task E returned

8. **Filter by parent_task_id**
   - Input: `GET http://localhost:2123/api/tasks?parent_task_id=<task-A-id>`
   - **Expected:** HTTP 200
   - **Expected:** Only Task F returned

9. **Apply limit=2**
   - Input: `GET http://localhost:2123/api/tasks?limit=2`
   - **Expected:** HTTP 200
   - **Expected:** Exactly 2 tasks returned

10. **Empty result set**
    - Input: `GET http://localhost:2123/api/tasks?status=completed`
    - **Expected:** HTTP 200
    - **Expected:** `{"tasks": []}` (empty array, not null)

---

### Data Validation
| Field | Source Value | Expected Value | Status |
|-------|-------------|----------------|--------|
| Response code (all valid queries) | HTTP status | 200 | [ ] |
| tasks array (unfiltered) | length | 6 | [ ] |
| tasks array (scope=global) | length | 4 | [ ] |
| tasks array (status=pending) | length | 4 | [ ] |
| tasks array (scope=global&status=pending) | length | 3 | [ ] |
| tasks array (owner filter) | length | 1 | [ ] |
| tasks array (workspace filter) | length | 2 | [ ] |
| tasks array (channel filter) | length | 1 | [ ] |
| tasks array (parent filter) | length | 1 | [ ] |
| tasks array (limit=2) | length | 2 | [ ] |
| tasks array (no matches) | length | 0 | [ ] |

---

### Error Scenarios
- [ ] Invalid scope value `?scope=bogus` returns 400 (ErrValidation)
- [ ] Invalid status value `?status=bogus` returns 400 (ErrValidation)
- [ ] `owner_kind` without `owner_ref` returns 400 (ErrValidation: both required together)
- [ ] Negative limit `?limit=-1` returns 400 (ErrValidation)
- [ ] `scope=global&workspace=some-ws` returns 400 (ErrInvalidScopeBinding)

---

### Related Test Cases
- TC-INT-001: Creates the tasks used as seed data
- TC-INT-005: UDS endpoint parity for list filtering
- TC-INT-007: CLI `agh task list` exercises the same filters via daemon API
