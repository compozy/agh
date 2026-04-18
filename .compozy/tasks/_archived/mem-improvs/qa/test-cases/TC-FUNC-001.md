## TC-FUNC-001: Write/Delete Keep Markdown, `MEMORY.md`, and Catalog in Sync

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Store
**Requirement:** REQ-MEM-001

---

### Objective

Verify that writing and deleting a memory updates all three persistence layers consistently: the Markdown file, the human-facing `MEMORY.md` index, and the derived search catalog.

---

### Preconditions

- [ ] A writable temp workspace and global memory directory exist.
- [ ] Search and read surfaces are available.
- [ ] The tester can inspect `MEMORY.md` directly in the relevant scope.

---

### Test Steps

1. Run a memory write flow for a new workspace-scoped file such as `release-plan.md`.
   - **Expected:** The write succeeds and the Markdown file is created.

2. Open the corresponding `MEMORY.md` file for that scope.
   - **Expected:** A single entry for `release-plan.md` exists with the expected title/description.

3. Run `agh memory search "release plan"` or call `/api/memory/search`.
   - **Expected:** The new memory is returned with the correct scope and snippet.

4. Delete the same memory through the public delete surface.
   - **Expected:** The delete succeeds and the Markdown file is removed.

5. Re-open `MEMORY.md` and re-run the search.
   - **Expected:** The deleted entry is gone from `MEMORY.md` and no longer appears in search results.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Similar filenames | `auth.md` and `auth-archive.md` | Deleting one does not remove the other from `MEMORY.md` |
| Parentheses in filename | `user(preferences).md` | Index removal still targets only the deleted file |
| Global scope | `--scope global` | Same synchronization rules apply in the global directory |

---

### Related Test Cases

- `SMOKE-002`
- `TC-FUNC-004`
- `TC-REG-002`

---

### Notes

This case directly protects the substring-pruning bug class described in the implementation ledger.
