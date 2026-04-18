## SMOKE-002: Reindex Completes and Health Reflects Catalog State

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Requirement:** REQ-MEM-001, REQ-MEM-004, REQ-MEM-006

---

### Objective

Verify that explicit reindex succeeds and that the health surface exposes indexed-file counts and last reindex metadata immediately afterward.

---

### Preconditions

- [ ] A temp daemon home and workspace are available.
- [ ] The corpus includes at least one global and one workspace memory file.
- [ ] `/api/observe/health` is reachable or the equivalent CLI/API surface is available.

---

### Test Steps

1. Ensure the mixed corpus exists and note the expected total indexed file count.
   - **Expected:** The expected total can be calculated before reindexing.

2. Run `agh memory reindex`.
   - **Expected:** The command succeeds and reports `indexed_files` equal to the seeded corpus size.

3. Call `GET /api/observe/health`.
   - **Expected:** The response succeeds and includes `memory.enabled`, `indexed_files`, `orphaned_files`, and `last_reindex`.

4. Compare the health payload with the corpus.
   - **Expected:** `indexed_files` matches the corpus size, `orphaned_files` is `0` for a clean corpus, and `last_reindex` is non-null and recent.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Workspace-only reindex | `agh memory reindex --scope workspace` | Only workspace files are counted |
| Global-only reindex | `agh memory reindex --scope global` | Only global files are counted |
| Repeated reindex | run twice | Second run is idempotent and still succeeds |

---

### Related Test Cases

- `TC-FUNC-001`
- `TC-FUNC-004`
- `TC-REG-001`

---

### Notes

This smoke case catches the most expensive class of failure early: the catalog exists but is stale, empty, or invisible to health reporting.
