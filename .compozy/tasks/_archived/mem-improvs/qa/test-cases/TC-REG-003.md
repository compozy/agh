## TC-REG-003: Workspace Root Derivation Uses the Current `.agh/memory` Layout

**Priority:** P0
**Type:** Regression
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Workspace Memory Resolution
**Requirement:** REQ-MEM-002, REQ-MEM-009

---

### Objective

Verify that workspace-scoped memory operations resolve against the current `.agh/memory` layout and do not regress to legacy `.compozy/memory` assumptions.

---

### Preconditions

- [ ] A temp workspace exists with the current `.agh/memory` directory populated.
- [ ] An optional legacy `.compozy/memory` directory can be created as a decoy.
- [ ] Search, reindex, and prompt-path checks can run against the workspace.

---

### Test Steps

1. Create a workspace with valid memories under `.agh/memory`.
   - **Expected:** The current-layout corpus is ready for lookup.

2. Optionally create a legacy `.compozy/memory` directory with conflicting or decoy files.
   - **Expected:** The decoy exists but should not affect current-layout behavior.

3. Run workspace-scoped search and reindex.
   - **Expected:** Both operations succeed using the `.agh/memory` corpus, regardless of any decoy legacy directory.

4. Trigger prompt-index loading or prompt recall for the same workspace.
   - **Expected:** The prompt path uses valid memories from `.agh/memory` and does not require `.compozy/memory`.

5. Inspect outputs for legacy leakage.
   - **Expected:** No result, snippet, or prompt content is sourced solely from the decoy legacy directory.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| No legacy directory present | only `.agh/memory` exists | Operations succeed normally |
| Both layouts present | `.agh/memory` + `.compozy/memory` | Current layout wins; no leakage from legacy path |
| Empty `.agh/memory` | legacy dir contains files | Search stays empty rather than using legacy files implicitly |

---

### Related Test Cases

- `TC-FUNC-002`
- `TC-INT-003`

---

### Notes

This explicitly covers the workspace-root bug fixed during implementation.
