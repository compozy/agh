## SMOKE-009: CLI `agh task list` Returns Results

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2-3 minutes
**Created:** 2026-04-14

---

### Objective
Quick sanity check that the CLI command `agh task list` communicates with the daemon over UDS, queries the task store, and returns formatted task results to the terminal.

---

### Preconditions
- [ ] AGH daemon running with task subsystem and UDS API initialized
- [ ] At least one task exists in the store (created via HTTP or CLI)
- [ ] AGH CLI binary available on PATH

---

### Test Steps
1. **Run `agh task list` with no filters**
   - Input: `agh task list`
   - **Expected:** Output displays a list of tasks. Each entry shows at minimum: task ID, title, status, scope. Exit code 0.

2. **Run `agh task list` with scope filter**
   - Input: `agh task list --scope global`
   - **Expected:** Output shows only global-scoped tasks. Exit code 0.

3. **Run `agh task list` with status filter**
   - Input: `agh task list --status pending`
   - **Expected:** Output shows only tasks with pending status. Exit code 0.

4. **Run `agh task list` when no tasks match**
   - Input: `agh task list --status completed` (assuming no completed tasks)
   - **Expected:** Empty output or "no tasks found" message. Exit code 0 (not an error).

5. **Verify UDS communication**
   - Input: Check that the CLI uses the UDS socket (not HTTP) by verifying no HTTP requests in daemon access logs
   - **Expected:** CLI communicates via UDS socket at the configured path.

---

### Related Test Cases
- SMOKE-001: Daemon starts with task subsystem
- SMOKE-003: List tasks via HTTP API
- SMOKE-002: Create a global task
