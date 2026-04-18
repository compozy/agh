## TC-INT-011: MCP servers workspace scope, target behavior, and scope isolation

**Priority:** P0
**Type:** Integration
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/mcp-servers`
**Traceability:** `task_13`, ADR-002, TechSpec > Scope rules, Known Risks

---

### Objective

Verify that workspace-scoped MCP operations use the active workspace correctly, preserve separate state from global scope, and keep target/precedence behavior legible after refetch.

---

### Preconditions

- [ ] HTTP is bound to loopback so collection mutations are allowed.
- [ ] A workspace fixture exists, recommended id `ws-polybot`.
- [ ] The MCP route exposes workspace scope and at least one workspace selector.
- [ ] The executor can use a disposable server name, for example `qa-mcp-workspace`.

---

### Test Steps

1. Open `/settings/mcp-servers`.
   - **Expected:** The route starts in global scope and shows scope-selection controls.

2. Switch to workspace scope using the workspace fixture.
   - **Expected:** The route clearly indicates the selected workspace, loads workspace-specific data, and keeps the shell/navigation stable.

3. Create or edit `qa-mcp-workspace` while workspace scope is active.
   - **Expected:** The editor shows that the mutation will apply to the selected workspace scope and exposes the valid workspace target options.

4. Save the workspace-scoped mutation.
   - **Expected:** The mutation succeeds with restart-required messaging and a workspace-specific write-target outcome.

5. Switch back to global scope.
   - **Expected:** The global collection view is restored, and the workspace-only record or override does not pollute the global list/detail state.

6. Switch again to the same workspace scope.
   - **Expected:** The workspace-scoped record reappears with the saved values, demonstrating separate cache entries and scope isolation.

7. Delete the workspace-scoped record and confirm cleanup.
   - **Expected:** The workspace record disappears from the workspace view only, and global scope remains unchanged.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Workspace id | `ws-polybot` | Recommended fixture from current route tests |
| Temporary server | `qa-mcp-workspace` | Disposable scoped record |
| Route | `/settings/mcp-servers` | Shared MCP route for both global and workspace scope |

---

### Post-conditions

- Ensure the temporary workspace-scoped record is deleted.
- Reset the route to global scope before leaving the case.
- Capture a screenshot of the workspace scope view if the UI mislabels scope or target state.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| No workspaces available | Empty workspace list | Route renders an explicit empty state and does not fake workspace scope |
| Switch scopes mid-edit | Change scope before saving | Draft state is reset or clearly isolated; stale workspace data does not leak into global scope |
| Workspace target fallback | Delete workspace definition with global definition beneath | Global definition remains visible only when global scope is selected |

---

### Related Test Cases

- `TC-FUNC-010` covers global-scope precedence and target selection.
- `TC-INT-013` covers the transport-level restriction variant for settings mutations.

---

### Notes

- This case is the main P0 proof that workspace-scoped settings editing is limited to MCP servers and behaves explicitly rather than implicitly.
