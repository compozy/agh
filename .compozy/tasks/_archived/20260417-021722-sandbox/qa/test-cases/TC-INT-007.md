## TC-INT-007: Workspace CRUD exposes sandbox_ref via API

**Priority:** P2 (Medium)
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 2 minutes
**Created:** 2026-04-16
**Task:** 01

---

### Objective

Verify that workspace create/read/update API contracts correctly include `sandbox_ref`, and the field persists through the full CRUD cycle.

---

### Test Steps

1. **Create workspace via API with sandbox_ref**
   - Input: `CreateWorkspaceRequest` with `sandbox_ref: "daytona-dev"`
   - **Expected:** Response `WorkspacePayload` includes `sandbox_ref: "daytona-dev"`

2. **Read workspace**
   - **Expected:** `sandbox_ref` field present in response

3. **Update workspace sandbox_ref**
   - Input: `UpdateWorkspaceRequest` with `sandbox_ref: "local-dev"`
   - **Expected:** Updated workspace shows `sandbox_ref: "local-dev"`

4. **Clear sandbox_ref**
   - Input: `UpdateWorkspaceRequest` with `sandbox_ref: ""`
   - **Expected:** Workspace reverts to empty (will resolve to default)
