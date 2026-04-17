## SMOKE-004: Workspace register with --environment flag persists

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify that registering a workspace with `--environment` flag persists the `environment_ref` in the database and returns it in API responses.

---

### Preconditions

- [x] Daemon running or test harness available
- [x] `environment_ref` column exists in workspaces table
- [x] CLI `workspace add` command supports `--environment` flag

---

### Test Steps

1. **Register workspace with environment flag**
   - Input: `agh workspace add /tmp/test-ws --environment daytona-dev`
   - **Expected:** Workspace created, response includes `environment_ref: "daytona-dev"`

2. **Retrieve workspace and verify persistence**
   - Input: `agh workspace list` or GET workspace API
   - **Expected:** Workspace payload includes `environment_ref: "daytona-dev"`

3. **Update workspace environment**
   - Input: `agh workspace edit <id> --environment local-profile`
   - **Expected:** Workspace updated, `environment_ref` changed

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No --environment flag | `agh workspace add /tmp/test-ws` | `environment_ref` is empty string |
| Invalid profile name | `--environment nonexistent` | Accepted (validation at session start, not workspace registration) |
