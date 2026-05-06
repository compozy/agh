## TC-UI-001: Web Sidebar `AgentCategoryTree` Renders Folders, Leaves, and Active State

**Priority:** P0
**Type:** UI
**Module:** `web/src/components/app-sidebar.tsx` + `web/src/systems/agent`
**Status:** Not Run
**Estimated Time:** 35 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Verify that `AgentCategoryTree` replaces the flat agent list in the sidebar, builds folder and leaf nodes from `AgentPayload.category_path`, sorts folders before leaves, preserves all pre-existing test IDs, exposes new deterministic folder test IDs, expands ancestors of the active agent on first render, and supports keyboard navigation.

---

### Preconditions

- [ ] Web dev server running against an isolated daemon (`AGH_WEB_API_PROXY_TARGET` set from the bootstrap manifest).
- [ ] Workspace seeded with at least:
  - One categorized agent with `category_path: ["Marketing", "Sales"]`.
  - One categorized agent with `category_path: ["Marketing", "Brand"]` (sibling subfolder).
  - One categorized agent with `category_path: ["Engineering"]` (single-segment).
  - One root-level agent with no `category_path`.
- [ ] An active session for one categorized agent (to test status dot + active expansion).

---

### Test Steps

1. **Render the sidebar and confirm structure.**
   - Input: Open `/` (or any in-app route) and inspect the sidebar.
   - **Expected:**
     - Folder nodes appear with `data-testid="agent-category-Marketing"` and `data-testid="agent-category-Marketing/Sales"` (and similar for Brand and Engineering).
     - Each leaf preserves `data-testid="agent-row-${agent.name}"`.
     - Folders render before leaves at every level.
     - Root-level agent appears at the top level alongside the folder nodes (no `Uncategorized` synthetic folder).

2. **Active route.**
   - Input: Navigate to `/agents/<categorized-agent-in-Marketing/Sales>`.
   - **Expected:**
     - The leaf has `data-active="true"` (or the project's equivalent active marker).
     - Both ancestor folders (`Marketing`, then `Marketing/Sales`) are expanded on first render.
     - `data-testid="agent-active-${name}"` is set for the active agent.

3. **Active session dot.**
   - Input: Same active agent has at least one running session.
   - **Expected:** `data-testid="agent-status-dot-${name}"` is rendered with the appropriate active styling.

4. **Default expansion when no agent is active.**
   - Input: Navigate to a route that does not match any agent leaf (e.g., dashboard).
   - **Expected:** Top-level folders are expanded; nested folders may be collapsed; root-level leaves are visible.

5. **Keyboard navigation.**
   - Input: Focus the tree, press `ArrowDown`, `ArrowRight` (expand), `ArrowLeft` (collapse), and `Enter`.
   - **Expected:** Focus moves between siblings, into children, and `Enter` activates the leaf link via TanStack `Link` to `/agents/$name`.

6. **Loading and empty states.**
   - Input: Force the agents query into loading and empty states (via the existing test fixtures or by stopping the daemon).
   - **Expected:** `data-testid="agents-loading"` renders during loading; `data-testid="agents-empty"` renders when the list is empty (and on error, per current behavior).

7. **Casing preservation in labels.**
   - Input: Add an agent with `category_path: ["marketing"]` (lowercase) alongside an existing `Marketing` folder.
   - **Expected:** Two separate sibling folders (`Marketing` and `marketing`) appear in the tree because casing is meaningful. Sorting groups them case-insensitively but does not merge them.

---

### Behavioral Evidence

- Operator journey: a categorized agent is visually grouped and routes correctly.
- Cross-surface: tree DOM agrees with `/api/agents` payload shape.
- Disruption probe: keyboard interaction and active expansion behave correctly.

---

### Audit Coverage

- C4: operator actor.
- C5: Web surface.
- C8: web payload vs DOM.
- C11: keyboard / active disruption.
- C14: `make web-test -- agent-category-tree` plus a manual browser pass with screenshots stored under `qa/screenshots/`.

---

### Pass Criteria

- All seven steps pass with screenshots captured for steps 1, 2, and 4.
- Every existing `data-testid` from the previous flat sidebar still resolves to the same agent.

---

### Failure Criteria

- Any leaf loses its `agent-row-${name}` test ID.
- Folder ordering puts leaves before folders at the same level.
- Default expansion fails to surface ancestors of the active agent.
- A synthetic `Uncategorized` folder appears.
