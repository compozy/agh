## TC-REG-003: Casing and Segment Order Are Preserved Across Every Surface

**Priority:** P1
**Type:** Regression
**Module:** `internal/config` + `internal/api/*` + `internal/cli` + `web/src/systems/agent`
**Status:** Not Run
**Estimated Time:** 25 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Author intent for `category_path` is the literal segment text and the literal order. Any surface that lowercases, reorders, or deduplicates segments would silently violate the contract. This case asserts cross-surface preservation by checking the same agent through every surface with mixed casing and a near-duplicate sibling.

---

### Test Steps

1. **Author a categorized agent with mixed casing.**
   - Input: AGENT.md `category_path: ["Marketing", "Sales"]` and a sibling AGENT.md with `category_path: ["marketing", "sales"]` (lowercase).
   - **Expected:** Both load successfully and surface as two distinct hierarchies (NOT merged).

2. **CLI JSON parity.**
   - Input: `agh agent list -o json | jq '.agents[] | {name, category_path}'`
   - **Expected:** Two agents — one with `["Marketing", "Sales"]`, one with `["marketing", "sales"]`. Casing preserved exactly.

3. **CLI human parity.**
   - Input: `agh agent list -o human`
   - **Expected:** Two distinct rows with `Marketing / Sales` and `marketing / sales` respectively (case-sensitive labels).

4. **HTTP + UDS parity.**
   - Input: `curl $BASE/api/agents | jq '.agents[].category_path'` and the UDS equivalent.
   - **Expected:** Same case-sensitive arrays.

5. **Web sidebar parity.**
   - Input: Inspect the sidebar.
   - **Expected:** Two distinct top-level folders (`Marketing` and `marketing`) — sorted case-insensitively but rendered with their authored casing. Each contains its own subfolder (`Sales` vs `sales`) and its own leaf.

6. **Order preservation.**
   - Input: A third agent with `category_path: ["Sales", "Marketing"]` (reversed segments).
   - **Expected:** Renders as a separate hierarchy `Sales/Marketing`. The runtime never reorders segments.

---

### Audit Coverage

- C4: parser + presenter actors.
- C5: CLI + HTTP + UDS + Web surfaces.
- C8: cross-surface byte-for-byte parity.
- C11: casing / order disruption probe.
- C14: focused Go + Vitest assertions plus a manual cross-surface diff captured under `qa/`.

---

### Pass Criteria

- All six steps pass.
- No surface lowercases, reorders, or merges segments.

---

### Failure Criteria

- Casing is normalized at any layer.
- `Sales/Marketing` is collapsed onto `Marketing/Sales`.
- Two case-different siblings are merged into a single folder.
