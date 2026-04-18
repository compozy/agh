## TC-FUNC-003: Search Respects Ranking, Scope, and Limit

**Priority:** P1
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Search
**Requirement:** REQ-MEM-003

---

### Objective

Verify that search ranks the most relevant memory first, preserves scope metadata, and enforces the requested result limit.

---

### Preconditions

- [ ] A seeded corpus exists with at least:
  - one highly relevant workspace memory
  - one partially relevant workspace memory
  - one global memory with overlapping terms
- [ ] Search can be exercised through HTTP or CLI.

---

### Test Steps

1. Seed the corpus with intentionally overlapping keywords such as `auth`, `session`, and `migration`.
   - **Expected:** The corpus is large enough to produce more than one hit.

2. Search for the full phrase that best matches the intended workspace memory.
   - **Expected:** The strongest workspace match is ranked first.

3. Inspect returned metadata.
   - **Expected:** Each result includes `filename`, `scope`, `workspace` when applicable, `score`, `snippet`, and `mod_time`.

4. Re-run with `limit=1` or `--limit 1`.
   - **Expected:** Only the top-ranked result is returned.

5. Re-run with an explicit global filter.
   - **Expected:** Workspace results disappear and only global matches remain.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Empty/blank query | `""` or whitespace only | Validation error or empty-safe handling, never panic |
| Large limit | `limit=50` with fewer docs | Returns all matching docs without error |
| No matches | query not in corpus | Empty list, stable JSON/CLI output |

---

### Related Test Cases

- `SMOKE-001`
- `TC-INT-001`
- `TC-INT-002`

---

### Notes

The most important assertion is relative ordering, not just “some result exists.”
