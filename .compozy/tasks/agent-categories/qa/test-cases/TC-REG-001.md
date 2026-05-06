## TC-REG-001: Strict YAML Rejects `categories` Alias and Slash-String Fallback

**Priority:** P1
**Type:** Regression
**Module:** `internal/config` (strict YAML decoder)
**Status:** Not Run
**Estimated Time:** 15 minutes
**Created:** 2026-05-06
**Last Updated:** 2026-05-06

---

### Objective

Lock the greenfield-alpha invariant: there is exactly one canonical name (`category_path`) for the field. Any historical alias (`categories:`) or slash-string fallback (`category_path: "Marketing/Sales"`) MUST fail at parse time so this contract cannot regress through future YAML decoder changes.

---

### Context

Recent changes that may affect this case:

- Implementation added `category_path` as the only canonical field.
- Strict-YAML decode is the only enforcement; there is no compatibility shim.

---

### Test Steps

1. **Reject `categories` alias key.**
   - Input: AGENT.md with `categories: [Marketing, Sales]`.
   - **Expected:** Parse fails with `ErrInvalidAgentFrontmatterKey` (or the equivalent strict-yaml unknown-key error). The agent does NOT load with an empty `category_path`.

2. **Reject scalar string for `category_path`.**
   - Input: AGENT.md with `category_path: "Marketing"` (scalar, not array).
   - **Expected:** Parse fails with a strict-yaml decode error naming `category_path` and the type mismatch.

3. **Reject scalar slash-string for `category_path`.**
   - Input: AGENT.md with `category_path: "Marketing/Sales"`.
   - **Expected:** Parse fails the same way; no auto-split into `["Marketing", "Sales"]`.

4. **Reject `category` (singular) key.**
   - Input: AGENT.md with `category: ["Marketing"]`.
   - **Expected:** Parse fails with the unknown-key error.

5. **Confirm the correct canonical field still parses.**
   - Input: AGENT.md with `category_path: [Marketing, Sales]`.
   - **Expected:** Parse succeeds and `AgentDef.CategoryPath` equals `["Marketing", "Sales"]`.

---

### Critical Path Tests

- [ ] No code path silently ignores the unknown key.
- [ ] No alias is mapped onto `CategoryPath` at any layer (parse, edit, resource codec).

---

### Audit Coverage

- C4: parser actor.
- C5: AGENT.md parse surface.
- C8: contract integrity.
- C11: alias / slash-string disruption.
- C14: `go test ./internal/config -run "Reject(Categories|Category|Slash)"` (or equivalent).

---

### Pass Criteria

- Steps 1–4 fail at parse with a clear error.
- Step 5 succeeds with the expected `CategoryPath`.

---

### Failure Criteria

- Any alias or slash-string is silently mapped onto `CategoryPath`.
- Strict-yaml decode regresses to permissive on unknown keys.
