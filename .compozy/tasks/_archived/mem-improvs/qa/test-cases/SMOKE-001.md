## SMOKE-001: Memory Search Returns the Correct Workspace Hit

**Priority:** P0
**Type:** Smoke
**Status:** Not Run
**Estimated Time:** 10 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Requirement:** REQ-MEM-003

---

### Objective

Verify that the new search surface can search a mixed global/workspace corpus and returns the expected workspace memory as the top hit.

---

### Preconditions

- [ ] Daemon/API and CLI are buildable from the current branch.
- [ ] A temp workspace exists using the current `.agh/memory` layout.
- [ ] The corpus contains at least:
  - global memory `prefs.md` with non-auth text
  - workspace memory `auth.md` mentioning `auth sessions`
- [ ] The derived catalog is either fresh or can be rebuilt with `agh memory reindex`.

---

### Test Steps

1. Seed one global memory and one workspace memory with distinct content.
   - **Expected:** Both files exist in their expected directories and can be listed.

2. Run `agh memory search "auth sessions"`.
   - **Expected:** The command succeeds and returns at least one result.

3. Inspect the top result.
   - **Expected:** The first result is the workspace-scoped `auth.md` entry with a relevant snippet.

4. Re-run the same query with `--limit 1`.
   - **Expected:** Exactly one result is returned and it is still the workspace `auth.md` entry.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Extra whitespace | `"  auth sessions  "` | Query is trimmed and still returns the workspace hit |
| Workspace-only filter | `--scope workspace` | Global results are excluded |
| No match | `"nonexistent phrase"` | Empty result set, no crash or malformed output |

---

### Related Test Cases

- `SMOKE-002`
- `TC-FUNC-003`
- `TC-INT-002`

---

### Notes

This is the stop/go check for the rest of the regression run. If the top hit is wrong, continue only after the ranking or scope bug is understood.
