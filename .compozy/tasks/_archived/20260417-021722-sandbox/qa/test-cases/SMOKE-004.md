## SMOKE-004: Workspace register with --sandbox flag persists

**Priority:** P0 (Critical)
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 1 minute
**Created:** 2026-04-16

---

### Objective

Verify that registering a workspace with `--sandbox` flag persists the `sandbox_ref` in the database and returns it in API responses.

---

### Preconditions

- [x] Daemon running or test harness available
- [x] `sandbox_ref` column exists in workspaces table
- [x] CLI `workspace add` command supports `--sandbox` flag

---

### Test Steps

1. **Register workspace with environment flag**
   - Input: `agh workspace add /tmp/test-ws --sandbox daytona-dev`
   - **Expected:** Workspace created, response includes `sandbox_ref: "daytona-dev"`

2. **Retrieve workspace and verify persistence**
   - Input: `agh workspace list` or GET workspace API
   - **Expected:** Workspace payload includes `sandbox_ref: "daytona-dev"`

3. **Update workspace sandbox**
   - Input: `agh workspace edit <id> --sandbox local-profile`
   - **Expected:** Workspace updated, `sandbox_ref` changed

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No --sandbox flag | `agh workspace add /tmp/test-ws` | `sandbox_ref` is empty string |
| Invalid profile name | `--sandbox nonexistent` | Accepted (validation at session start, not workspace registration) |
