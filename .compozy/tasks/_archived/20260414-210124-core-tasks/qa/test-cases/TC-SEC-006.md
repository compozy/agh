## TC-SEC-006: SQL Injection Resistance in Task Fields and Filter Parameters

**Priority:** P0
**Type:** Security
**Risk Level:** Critical
**Status:** Not Run
**Created:** 2026-04-14

---

### Objective
Validate that the task system is immune to SQL injection attacks through all user-controllable input surfaces: task title, description, metadata, filter query parameters, and identifier fields. SQLite parameterized queries must prevent any injection.

---

### Preconditions
- [ ] AGH daemon running with task subsystem initialized
- [ ] Authenticated principal with full write access
- [ ] Access to both create and list task endpoints

---

### Test Steps
1. **SQL injection in task title**
   - Input: `POST /api/tasks` with title: `"'; DROP TABLE tasks; --"`
   - **Expected:** 201 Created. Task persisted with the literal string as the title. Database tables intact. Subsequent `GET /api/tasks` returns the task with the injection string as-is.

2. **SQL injection in task description**
   - Input: `POST /api/tasks` with description: `"\" OR 1=1; UPDATE tasks SET status='cancelled' WHERE 1=1; --"`
   - **Expected:** 201 Created. Description stored literally. No side-effect mutations.

3. **SQL injection in task identifier**
   - Input: `POST /api/tasks` with identifier: `"TASK-1' UNION SELECT * FROM sqlite_master--"`
   - **Expected:** 201 Created (or validation error if identifier format is restricted). No schema leakage.

4. **SQL injection in list filter: scope parameter**
   - Input: `GET /api/tasks?scope=global' OR '1'='1`
   - **Expected:** 400 Bad Request (invalid scope value) or empty results. No data from other scopes returned.

5. **SQL injection in list filter: status parameter**
   - Input: `GET /api/tasks?status=pending' UNION SELECT sql FROM sqlite_master--`
   - **Expected:** 400 Bad Request (invalid status value). No schema information leaked.

6. **SQL injection in list filter: owner_ref parameter**
   - Input: `GET /api/tasks?owner_ref=user-1' OR '1'='1`
   - **Expected:** Empty results or only tasks matching the literal string. No unfiltered data returned.

7. **SQL injection in metadata JSON field**
   - Input: `POST /api/tasks` with metadata: `{"key": "value'); DROP TABLE task_events; --"}`
   - **Expected:** 201 Created. Metadata stored as valid JSON with the injection string as a literal value.

8. **Verify database integrity after all injection attempts**
   - Input: Confirm all core tables exist and row counts are correct
   - **Expected:** Tables `tasks`, `task_runs`, `task_events`, `task_dependencies` all present with expected row counts. No data corruption.

---

### Attack Vectors
- [ ] Classic SQL injection via single quotes in string fields
- [ ] UNION-based injection to extract schema metadata
- [ ] Stacked query injection (`;` followed by destructive SQL)
- [ ] Blind SQL injection via boolean-based filter parameter manipulation
- [ ] Time-based blind injection via SQLite `LIKE` or `GLOB` operators in filters
- [ ] Second-order injection where stored payload is later used in a query

---

### Related Test Cases
- TC-SEC-007: Oversized payload rejection
- SMOKE-002: Basic task creation
