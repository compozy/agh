## TC-FUNC-004: Reindex Rebuilds the Derived Catalog from Markdown Source

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Reindex
**Requirement:** REQ-MEM-001, REQ-MEM-004

---

### Objective

Verify that explicit reindex can reconstruct the catalog from Markdown files after catalog drift, deletion, or initialization from an empty DB.

---

### Preconditions

- [ ] A real SQLite catalog file is used.
- [ ] A mixed corpus of Markdown memory files exists.
- [ ] The tester can simulate drift by clearing or replacing the catalog file.

---

### Test Steps

1. Seed a clean corpus and confirm search returns expected hits.
   - **Expected:** The baseline corpus is searchable.

2. Simulate catalog drift by clearing the catalog file or replacing it with an empty DB while leaving Markdown files intact.
   - **Expected:** The catalog is now stale or empty while source files remain.

3. Run `agh memory reindex`.
   - **Expected:** Reindex succeeds and reports the correct `indexed_files` count.

4. Re-run search against the same corpus.
   - **Expected:** The previously searchable memories are returned again from the rebuilt catalog.

5. Inspect health metadata.
   - **Expected:** `last_reindex` is updated and `indexed_files` matches the restored corpus.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Global-only drift | clear only global catalog state | Reindex restores global hits |
| Workspace-only drift | clear only workspace data | Reindex restores workspace hits |
| Double reindex | run twice consecutively | Second run remains stable and idempotent |

---

### Related Test Cases

- `SMOKE-002`
- `TC-FUNC-001`
- `TC-REG-001`

---

### Notes

This is the recovery test that proves the catalog is truly derived and rebuildable.
