## TC-FUNC-010: MCP servers global-scope precedence and target-selection behavior

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Route:** `/settings/mcp-servers`
**Traceability:** `task_13`, ADR-002, ADR-003, TechSpec > Scope rules, Collection mutation semantics, Known Risks

---

### Objective

Verify that the MCP Servers route exposes global-scope precedence metadata, supports explicit target selection, and communicates restart-required collection mutations plus fallback behavior clearly.

---

### Preconditions

- [ ] HTTP is bound to loopback so collection mutations are allowed.
- [ ] At least one global MCP server exists, or the executor can create one temporarily.
- [ ] The executor can observe source metadata such as `effective_source`, `shadowed_sources`, and `available_targets`.

---

### Test Steps

1. Open `/settings/mcp-servers` in global scope.
   - **Expected:** The route loads the global collection, shows source/precedence metadata, and offers target selection controls for supported write targets.

2. Select an existing global MCP server or create a temporary one if necessary.
   - **Expected:** The detail/editor state clearly identifies the current effective source and the allowed targets.

3. Edit the server and choose a target path such as `auto`, `config`, or `sidecar`.
   - **Expected:** The target choice is visible before save and its consequences are explained by the route.

4. Save the mutation.
   - **Expected:** The result reports restart-required behavior, and the route shows write-target feedback after refetch.

5. Verify the updated metadata after refetch.
   - **Expected:** `effective_source`, `shadowed_sources`, and the selected target behavior are reflected accurately in the UI.

6. Delete the highest-precedence definition for the same server.
   - **Expected:** If a lower-precedence definition exists, the route explains that it may become effective again; if none exists, the server disappears entirely.

---

### Test Data

| Field | Value | Notes |
|-------|-------|-------|
| Route | `/settings/mcp-servers` | MCP server collection route |
| Temporary server name | `qa-mcp-global` | Use if no safe disposable record exists |
| Target selector values | `auto`, `config`, `sidecar` | Verify only the values exposed by the UI/environment |

---

### Post-conditions

- Delete any temporary MCP server created for the test.
- Restore any edited non-temporary record to baseline.
- Capture a screenshot of the precedence metadata after save or delete.

---

### Edge Cases & Variations

| Variation | Input | Expected Result |
|-----------|-------|-----------------|
| New server with `auto` target | Create disposable record | Default target semantics are explained and match the UI copy |
| Existing server with shadowed source | Edit top-level definition | Refetch shows the correct effective source after save |
| Delete with no fallback source | Remove temp-only server | The row disappears cleanly with no phantom shadowed state |

---

### Related Test Cases

- `TC-INT-011` validates workspace-scoped MCP behavior and cache isolation.
- `TC-FUNC-008` validates simpler collection fallback behavior on providers.

---

### Notes

- This is a P0 route because MCP precedence confusion is one of the highest-risk operator flows in the TechSpec.
