## TC-INT-007: Workspace CRUD exposes environment_ref via API

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that workspace create/read/update API contracts correctly include `environment_ref`, and the field persists through the full CRUD cycle.

---

### Test Steps

1. **Create workspace via API with environment_ref**
   - Input: `CreateWorkspaceRequest` with `environment_ref: "daytona-dev"`
   - **Expected:** Response `WorkspacePayload` includes `environment_ref: "daytona-dev"`

2. **Read workspace**
   - **Expected:** `environment_ref` field present in response

3. **Update workspace environment_ref**
   - Input: `UpdateWorkspaceRequest` with `environment_ref: "local-dev"`
   - **Expected:** Updated workspace shows `environment_ref: "local-dev"`

4. **Clear environment_ref**
   - Input: `UpdateWorkspaceRequest` with `environment_ref: ""`
   - **Expected:** Workspace reverts to empty (will resolve to default)
