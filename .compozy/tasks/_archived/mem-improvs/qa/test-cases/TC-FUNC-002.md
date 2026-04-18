## TC-FUNC-002: Missing or Stale `MEMORY.md` Synthesizes a Safe Prompt Index

**Priority:** P0
**Type:** Functional
**Status:** Not Run
**Estimated Time:** 20 minutes
**Created:** 2026-04-17
**Last Updated:** 2026-04-17
**Module:** Memory Prompt Index
**Requirement:** REQ-MEM-002

---

### Objective

Verify that prompt-index loading does not depend blindly on `MEMORY.md` and can synthesize a prompt-safe index from valid memory files when the tracked index is missing or stale.

---

### Preconditions

- [ ] A temp scope contains at least two valid Markdown memory files.
- [ ] The tester can modify or delete the scope-local `MEMORY.md`.
- [ ] A narrow harness or targeted package entry point is available to call the prompt-index loader.

---

### Test Steps

1. Create valid memory files, then remove `MEMORY.md`.
   - **Expected:** The scope now has durable memory files but no prompt index file.

2. Trigger prompt-index loading through the narrow harness or targeted package test.
   - **Expected:** The load succeeds without rewriting files and returns synthesized index content.

3. Recreate `MEMORY.md` with one missing target and one stale entry.
   - **Expected:** The on-disk index now contains invalid references.

4. Trigger prompt-index loading again.
   - **Expected:** The load succeeds, ignores missing targets, and only exposes valid memory entries in the synthesized output.

5. Confirm that no read-only synthesis step rewrote the memory files or `MEMORY.md`.
   - **Expected:** File timestamps and contents remain unchanged until an explicit reindex is requested.

---

### Edge Cases

| Variation | Input | Expected Result |
| --- | --- | --- |
| Completely missing `MEMORY.md` | file absent | Synthesized prompt index is returned |
| Stale index with missing files | index points to deleted docs | Missing targets are ignored |
| Empty scope | no memory files | Empty prompt index, no crash |

---

### Related Test Cases

- `TC-FUNC-004`
- `TC-REG-003`
- `TC-INT-003`

---

### Notes

This is a key resilience requirement because prompt assembly must remain useful even when the human-maintained index drifts.
