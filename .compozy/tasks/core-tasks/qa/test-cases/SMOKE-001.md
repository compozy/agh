## SMOKE-001: Daemon Starts with Task Subsystem Initialized

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that the AGH daemon boots successfully with the task subsystem fully initialized, including the TaskManager, task store (SQLite tables), observe layer task projections, and HTTP/UDS route registration.

---

### Preconditions
- [ ] AGH binary built (`make build`)
- [ ] No existing daemon process running (clean start)
- [ ] Valid configuration file or defaults available

---

### Test Steps
1. **Start the AGH daemon**
   - Input: `agh daemon start` (or equivalent)
   - **Expected:** Daemon process starts without errors. Exit code 0 for background mode. Log output includes task subsystem initialization messages.

2. **Verify task store tables created**
   - Input: Check daemon startup logs for SQLite migration or table creation
   - **Expected:** Logs confirm `tasks`, `task_runs`, `task_events`, `task_dependencies` tables initialized in `agh.db`.

3. **Verify HTTP task routes registered**
   - Input: `curl -s http://localhost:<port>/api/tasks` (with auth)
   - **Expected:** 200 OK with empty array `[]` (no tasks yet). Confirms route is live.

4. **Verify UDS task routes registered**
   - Input: `agh task list` via CLI (which uses UDS)
   - **Expected:** Empty list returned. No connection errors.

5. **Verify observe layer initialized**
   - Input: Query task metrics endpoint or observe summary
   - **Expected:** Response with zero-valued metrics (no tasks yet). No errors.

---

### Related Test Cases
- SMOKE-002: Create a global task via HTTP API
- SMOKE-009: CLI `agh task list` returns results
