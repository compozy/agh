## SMOKE-002: Local session lifecycle works end-to-end

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16

---

### Objective

Verify a local session can be created, used (prompt), and stopped without error after the ACP Launcher/ToolHost extraction. This is the primary regression gate.

---

### Preconditions

- [x] Daemon running or integration test harness available
- [x] Local provider registered in registry
- [x] Workspace configured with default (local) environment

---

### Test Steps

1. **Create a session with local environment**
   - Input: Workspace with no `environment_ref` set (defaults to local)
   - **Expected:** Session created, `SessionInfo.Environment.Backend` == `"local"`, environment state transitions through `creating` -> `prepared` -> `running`

2. **Send a prompt to the session**
   - Input: Simple ACP prompt via driver
   - **Expected:** Agent responds, ACP protocol works over Launcher-provided stdin/stdout

3. **Stop the session**
   - **Expected:** Session stops cleanly, `SyncFromRuntime` called (no-op for local), no errors

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| Explicit `environment_ref: ""` | Empty string | Resolves to local |
| Explicit `environment_ref: local` | Profile name pointing to local backend | Same behavior |
