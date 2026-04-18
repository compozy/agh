## TC-REG-001: Health Payload Exposes Memory Config and Catalog Stats

**Priority:** P1
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Observe Health
**Requirement:** REQ-MEM-006

---

### Objective

Verify that `/api/observe/health` keeps exposing the new memory configuration and catalog fields after future changes.

---

### Preconditions

- [ ] Daemon is running with memory enabled.
- [ ] A clean corpus has been reindexed at least once.
- [ ] `/api/observe/health` is reachable.

---

### Test Steps

1. Trigger a fresh memory reindex.
   - **Expected:** Reindex succeeds and updates catalog state.

2. Call `GET /api/observe/health`.
   - **Expected:** Response status is `200`.

3. Inspect the `memory` block in the response.
   - **Expected:** It includes `enabled`, `global_dir`, `dream_agent`, `dream_min_hours`, `dream_min_sessions`, `dream_check_interval`, `indexed_files`, `orphaned_files`, and `last_reindex`.

4. Compare dynamic values with the current corpus.
   - **Expected:** `indexed_files` and `orphaned_files` match reality, and `last_reindex` is non-null and recent.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Empty corpus | no memory files | `indexed_files=0`, stable payload |
| Dream disabled | config disables dream consolidation | Dream config fields remain populated, enablement reflects config/runtime |
| Workspace present | one active workspace | `workspace_files` reflects visible workspace corpus |

---

### Related Test Cases

- `SMOKE-002`
- `TC-FUNC-004`
- `TC-REG-002`

---

### Notes

Operator visibility is only useful if it stays aligned with actual catalog state after refactors.
